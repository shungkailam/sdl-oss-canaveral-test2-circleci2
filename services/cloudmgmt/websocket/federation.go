package websocket

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"encoding/json"

	"cloudservices/cloudmgmt/util"
	"cloudservices/common/base"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
)

const (
	// msgTypeRequest represents the type of message published on Redis
	// when a response is synchronously expected. This becomes a part of
	// the channel name on which the request message is published.
	msgTypeRequest string = "request"
	// msgTypeResponse represents the response message published on Redis
	// corresponding to a 'msgTypeRequest' message. This becomes a part of
	// the channel name on which the response message is sent.
	msgTypeResponse string = "response"
)

var (
	podIP                 string
	connKeyExpirationTime = time.Duration(8 * time.Minute)
)

type FederationService struct {
	ID          string
	RedisClient *redis.Client
	PubSub      *redis.PubSub
}

type MessageMetadata struct {
	originHostname string
	tenantID       string
	edgeID         string
	action         string
	messageKey     string
	message        interface{}
	sync           bool
}

type responseData struct {
	Response string `json:"response"`
	Err      error  `json:"err"`
}

func getEdgeKey(tenantID string, edgeID string) string {
	return fmt.Sprintf("%s.%s", tenantID, edgeID)
}

// call this every time we receive reportEdge from an edge
func (federationService FederationService) claimEdge(tenantID string, edgeID string) error {
	key := getEdgeKey(tenantID, edgeID)
	return federationService.RedisClient.Set(key, federationService.ID, connKeyExpirationTime).Err()
}

// call this every time we get disconnect from an edge websocket
// use watch to wrap get and del - only delete if we still own the key
func (federationService FederationService) unclaimEdge(tenantID string, edgeID string) error {
	key := getEdgeKey(tenantID, edgeID)
	// return federationService.RedisClient.Del(key).Err()
	return federationService.RedisClient.Watch(func(tx *redis.Tx) error {
		id, err := tx.Get(key).Result()
		if err != nil {
			return err
		}
		if id == federationService.ID {
			_, err = tx.Pipelined(func(pipe redis.Pipeliner) error {
				return pipe.Del(key).Err()
			})
			return err
		}
		// skip
		return nil
	}, key)
}

// refreshClaimEdges refreshes the edge connections (edge, tenant pair) owned by this current federation service
func (federationService FederationService) refreshClaimEdges(edgeTenants map[string]string) error {
	if len(edgeTenants) == 0 {
		return nil
	}
	glog.Infof("Refreshing edge claims for federation ID %s", federationService.ID)
	commands := map[string]*redis.BoolCmd{}
	_, err := federationService.RedisClient.Pipelined(func(pipe redis.Pipeliner) error {
		for edgeID, tenantID := range edgeTenants {
			key := getEdgeKey(tenantID, edgeID)
			// Only refresh the time. Do not touch the value
			commands[key] = pipe.Expire(key, connKeyExpirationTime)
		}
		return nil
	})
	if err != nil {
		glog.Errorf("Error in refreshing edge connection keys in redis. Error: %s", err.Error())
		return err
	}
	for key, cmd := range commands {
		_, err := cmd.Result()
		if err != nil {
			glog.Errorf("Error in refreshing edge connection for key %s in redis. Error: %s", key, err.Error())
		}
	}
	return nil
}

func (federationService FederationService) containsEdge(tenantID string, edgeID string) bool {
	key := getEdgeKey(tenantID, edgeID)
	return federationService.RedisClient.Get(key).Err() == nil
}

func (federationService FederationService) containsEdges(tenantID string, edgeIDs ...string) map[string]bool {
	connectedFlags := map[string]bool{}
	if len(edgeIDs) == 0 {
		return connectedFlags
	}
	commands := map[string]*redis.IntCmd{}
	_, err := federationService.RedisClient.Pipelined(func(pipe redis.Pipeliner) error {
		for _, edgeID := range edgeIDs {
			key := getEdgeKey(tenantID, edgeID)
			commands[edgeID] = pipe.Exists(key)
		}
		return nil
	})
	if err != nil {
		return connectedFlags
	}
	for edgeID := range commands {
		cmd := commands[edgeID]
		i, err := cmd.Result()
		if err != nil {
			glog.Errorf("Error in getting connection for edge %s. Error: %s", edgeID, err.Error())
			// Ignore
			i = 0
		}
		connectedFlags[edgeID] = (i > 0)
	}
	return connectedFlags
}

func (federationService FederationService) getEdgeOwnerIP(tenantID string, edgeID string) (string, error) {
	key := getEdgeKey(tenantID, edgeID)
	// value at edge key is federation channel id
	id, err := federationService.RedisClient.Get(key).Result()
	if err != nil {
		return "", err
	}
	// value stored at id is pod IP
	return federationService.RedisClient.Get(id).Result()
}
func getRequestChannel(listenerFedID string, senderFedID string, reqID string, tableName string, msgMetadata *MessageMetadata) string {
	msgType := ""
	if msgMetadata.sync {
		msgType = msgTypeRequest
	}
	tokens := []string{listenerFedID, msgMetadata.tenantID, msgMetadata.edgeID, msgMetadata.action, msgMetadata.messageKey, reqID, tableName, msgMetadata.originHostname, msgType, senderFedID}
	// set startIndex to 1 to avoid encoding channel id
	startIndex := 1
	return base.EncodeTokens(tokens, startIndex)
}

func getResponseChannel(listenerFedID string, reqID string, tableName string, msgMetadata *MessageMetadata) string {
	tokens := []string{listenerFedID, msgMetadata.tenantID, msgMetadata.edgeID, msgMetadata.action, msgMetadata.messageKey, reqID, tableName, msgMetadata.originHostname, msgTypeResponse, ""}
	// set startIndex to 1 to avoid encoding channel id
	startIndex := 1
	return base.EncodeTokens(tokens, startIndex)
}

// action = ack or emit
// messageKey = onCreateCategory, ...
// message = normal websocket message payload
// pre-condition: containsEdge && edge is not local
func (federationService FederationService) sendMessageToEdge(ctx context.Context, msgMetadata *MessageMetadata) (string, error) {
	// publish message to redis
	reqID := base.GetRequestID(ctx)
	tableName := base.GetAuditLogTableName(ctx)
	key := getEdgeKey(msgMetadata.tenantID, msgMetadata.edgeID)
	id, err := federationService.RedisClient.Get(key).Result()
	if err != nil {
		return "", err
	}
	if id == federationService.ID {
		return "", fmt.Errorf(base.PrefixRequestID(ctx, "federation: Send message to edge: edge is local, tenantID=%s, edgeID=%s"), msgMetadata.tenantID, msgMetadata.edgeID)
	}

	channel := getRequestChannel(id, federationService.ID, reqID, tableName, msgMetadata)
	glog.Infof(base.PrefixRequestID(ctx, "federation: Send message to edge: channel=%s, msg key=%s, msg=%+v\n"), channel, msgMetadata.messageKey, msgMetadata.message)
	ba, err := json.Marshal(msgMetadata.message)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "federation: Send message to edge, marshal failed with error %s\n"), err.Error())
		return "", err
	}
	// If this is a sync request, subscribe to response channel before publishing.
	var respSub *redis.PubSub
	var respChannel string
	if msgMetadata.sync {
		respChannel = getResponseChannel(federationService.ID, reqID, tableName, msgMetadata)
		respSub = federationService.RedisClient.Subscribe(respChannel)
		defer func() {
			respSub.Unsubscribe()
		}()
	}
	err = federationService.RedisClient.Publish(channel, ba).Err()
	if !msgMetadata.sync {
		return "", err
	}
	ch := respSub.Channel()
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	var resp string
	select {
	case msg := <-ch:
		respMessage := responseData{}
		err = json.Unmarshal([]byte(msg.Payload), &respMessage)
		if err != nil {
			glog.Warningf("federation: failed to unmarshal response %s\n", msg.Payload)
			return resp, err
		}
		return respMessage.Response, respMessage.Err
	case <-t.C:
		glog.Warningf(base.PrefixRequestID(ctx, "Timeout waiting for response: channel=%s"), respChannel)
		resp = ""
		err = http.ErrHandlerTimeout
	}
	return resp, err
}

func (federationService FederationService) eventloop(callback func(ctx context.Context, tableName, origin, edgeID, action, msgName string, msg interface{}, sync bool) (string, error)) {
	glog.Infoln("federation event loop {")
	// Go channel which receives messages.
	ch := federationService.PubSub.Channel()
	// Consume messages.
	for msg := range ch {
		// parse tenantID, edgeID, messageKey off msg.Channel
		glog.Infof("federation event loop got message: channel=%s, payload=%s\n", msg.Channel, msg.Payload)
		// set startIndex to 1 to avoid decoding channel id
		startIndex := 1
		tokens, err := base.DecodeTokens(msg.Channel, startIndex)
		if err != nil || len(tokens) < 5 {
			continue
		}
		id, tenantID, edgeID, action, msgKey := tokens[0], tokens[1], tokens[2], tokens[3], tokens[4]
		if id != federationService.ID {
			continue
		}
		// new reqID at the end for backward compatibility
		var reqID string
		var originHostname string
		var tableName string
		var msgType string
		var originFederationID string
		if len(tokens) > 9 {
			reqID = tokens[5]
			tableName = tokens[6]
			originHostname = tokens[7]
			msgType = tokens[8]
			originFederationID = tokens[9]
		} else {
			reqID = base.GetUUID()
		}
		if msgType == msgTypeResponse {
			// If this is a response, sendMessageToEdge() will directly handle
			// the message returned by the edge.
			continue
		}
		ctx := base.GetAdminContext(reqID, tenantID)
		var doc interface{}
		doc = struct{}{}
		err = json.Unmarshal([]byte(msg.Payload), &doc)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "federation: eventloop: failed to unmarshal %s\n"), msg.Payload)
			continue
		}
		glog.Infof(base.PrefixRequestID(ctx, "federation: eventloop got doc: %+v\n"), doc)
		sync := false
		if msgType == msgTypeRequest {
			// XXX: Ideally, we do not want to call into websocket from a go routine because
			//      it results in loss of ordering of messages. Unfortunately, a sync
			//      request can take long and block the event loop. So we are resorting to
			//      using a go routine. This should be done only in case the ordering of this
			//      message is not a concern.
			go func() {
				sync = true
				resp, err := callback(ctx, tableName, originHostname, edgeID, action, msgKey, doc, sync)
				msgMetadata := &MessageMetadata{
					originHostname: originHostname,
					tenantID:       tenantID,
					edgeID:         edgeID,
					action:         action,
					messageKey:     msgKey,
				}
				channel := getResponseChannel(originFederationID, reqID, tableName, msgMetadata)
				respMessage := responseData{
					Response: resp,
					Err:      err,
				}
				glog.Infof(base.PrefixRequestID(ctx, "federation: Send response to edge: channel=%s, msg key=%s, msg=%+v\n"), channel, msgKey, resp)
				ba, err := json.Marshal(respMessage)
				if err != nil {
					glog.Warningf(base.PrefixRequestID(ctx, "federation: Send response to edge, marshal failed with error %s\n"), err.Error())
					return
				}
				err = federationService.RedisClient.Publish(channel, ba).Err()
				if err != nil {
					glog.Warningf(base.PrefixRequestID(ctx, "federation: Send response to edge, marshal failed with error %s\n"), err.Error())
				}
			}()
			continue
		}
		callback(ctx, tableName, originHostname, edgeID, action, msgKey, doc, sync)
	}
	glog.Infoln("federation event loop }")
}

func (federationService FederationService) IsMyIP(IP string) bool {
	return IP != "" && IP == podIP
}

func (federationService FederationService) init() (err error) {
	id := federationService.ID
	podIP, err = util.GetPodIP()
	if err != nil {
		return
	}
	if podIP == "" {
		return fmt.Errorf("Failed to get pod IP")
	}
	return federationService.RedisClient.Set(id, podIP, 0).Err()
}

func (federationService FederationService) isInitialized() bool {
	key := federationService.ID
	IP, err := federationService.RedisClient.Get(key).Result()
	return err == nil && IP == podIP
}

func NewFederationService(id string, redisClient *redis.Client) *FederationService {
	if redisClient == nil {
		return nil
	}
	channel := fmt.Sprintf("%s.*", id)
	pubsub := redisClient.PSubscribe(channel)
	glog.Infof("federation service created with id=%s\n", id)
	// Wait for confirmation that subscription is created before publishing anything.
	_, err := pubsub.Receive()
	if err != nil {
		panic(err)
	}

	federationService := &FederationService{
		ID:          id,
		RedisClient: redisClient,
		PubSub:      pubsub,
	}

	err = federationService.init()
	if err != nil {
		panic(err)
	}

	return federationService
}

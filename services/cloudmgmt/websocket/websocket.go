package websocket

import (
	"bufio"
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/metrics"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	gapi "cloudservices/cloudmgmt/generated/grpc"
	"cloudservices/common/service"

	"google.golang.org/grpc"

	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"

	"github.com/go-redis/redis"
	gosocketio "github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"
	"github.com/julienschmidt/httprouter"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	WEBSOCKET_SEND_TIMEOUT_SEC = 600
	PROXY_SEND_TIMEOUT_SEC     = 60
)

type wsMessagingServiceImpl struct {
	server *gosocketio.Server
	dbAPI  api.ObjectModelAPI
	// Federation service - will be nil if Cfg.DisableScaleOut
	federationService *FederationService
	mx                sync.Mutex
	// map of channel key to channel, channel key is of form <tenant id>/<edge id>
	edgeChannelMap map[string]*gosocketio.Channel
	// inverse of edgeChannelMap - maintain this for efficiency
	channelEdgeMap map[*gosocketio.Channel]string
}

func makeChannelKey(tenantID string, edgeID string) string {
	return fmt.Sprintf("%s/%s", tenantID, edgeID)
}

func extractTenantAndEdgeIDs(channelKey string) (string, string) {
	tokens := strings.Split(channelKey, "/")
	// TenantID, EdgeID
	return tokens[0], tokens[1]
}

func (py *wsMessagingServiceImpl) GetChannel(tenantID string, edgeID string) *gosocketio.Channel {
	py.mx.Lock()
	defer py.mx.Unlock()
	key := makeChannelKey(tenantID, edgeID)
	ch := py.edgeChannelMap[key]
	return ch
}

// get channel key (tenant id, edge id) for the given channel
// not thread-safe: does not lock mutex (meant to be called after lock is qcquired)
func (py *wsMessagingServiceImpl) getChannelKey(c *gosocketio.Channel) (string, string) {
	tenantID, edgeID := "", ""
	key := py.channelEdgeMap[c]
	if key != "" {
		tenantID, edgeID = extractTenantAndEdgeIDs(key)
	}
	return tenantID, edgeID
}
func (py *wsMessagingServiceImpl) GetConnectedEdgeIDs(tenantID string) []string {
	py.mx.Lock()
	defer py.mx.Unlock()
	edgeIDs := []string{}
	for _, c := range py.server.List(tenantID) {
		tid, eid := py.getChannelKey(c)
		if tid == tenantID && eid != "" {
			edgeIDs = append(edgeIDs, eid)
		}
	}
	return edgeIDs
}

// whether given edge is connected via websocket to this cloudmgmt instance
func (py *wsMessagingServiceImpl) isLocallyConnectedEdge(tenantID string, edgeID string) bool {
	return py.GetChannel(tenantID, edgeID) != nil
}

// whether given edge is connected via websocket to some cloudmgmt instance
func (py *wsMessagingServiceImpl) IsConnectedEdge(tenantID string, edgeID string) bool {
	return py.isLocallyConnectedEdge(tenantID, edgeID) || (py.federationService != nil && py.federationService.containsEdge(tenantID, edgeID))
}

// GetEdgeConnections returns the connection status for the given edges
func (py *wsMessagingServiceImpl) GetEdgeConnections(tenantID string, edgeIDs ...string) map[string]bool {
	connectionFlags := map[string]bool{}
	if edgeIDs == nil || len(edgeIDs) == 0 {
		return connectionFlags
	}
	remoteEdgeIDs := make([]string, 0, len(edgeIDs))
	for _, edgeID := range edgeIDs {
		if py.isLocallyConnectedEdge(tenantID, edgeID) {
			connectionFlags[edgeID] = true
		} else {
			remoteEdgeIDs = append(remoteEdgeIDs, edgeID)
		}
	}
	if py.federationService != nil && len(remoteEdgeIDs) > 0 {
		remoteConnectionFlags := py.federationService.containsEdges(tenantID, remoteEdgeIDs...)
		for key, value := range remoteConnectionFlags {
			connectionFlags[key] = value
		}
	}
	return connectionFlags
}

func (py *wsMessagingServiceImpl) SetChannel(tenantID string, edgeID string, c *gosocketio.Channel) {
	py.mx.Lock()
	defer py.mx.Unlock()
	key := makeChannelKey(tenantID, edgeID)
	py.edgeChannelMap[key] = c
	py.channelEdgeMap[c] = key
	ctx := base.GetAdminContext(base.GetUUID(), tenantID)
	metrics.WebSocketConnections.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "tenant_id": tenantID, "edge_id": edgeID}).Inc()
	if py.federationService != nil {
		err := py.federationService.claimEdge(tenantID, edgeID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to claim edge, tenantID=%s, edgeID=%s. Error: %s"), tenantID, edgeID, err.Error())
		}
	}
	event := model.EdgeConnectionEvent{ID: base.GetUUID(), TenantID: tenantID, EdgeID: edgeID, Status: true}
	err := base.Publisher.Publish(ctx, &event)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to publish event %v. Error: %s"), event, err.Error())
	}
}

func (py *wsMessagingServiceImpl) RemoveChannel(c *gosocketio.Channel) {
	py.mx.Lock()
	defer py.mx.Unlock()
	key := ""
	for k, v := range py.edgeChannelMap {
		if v == c {
			key = k
			break
		}
	}
	if key != "" {
		glog.Infof("Disconnected websocket client key: %s", key)
		tenantID, edgeID := extractTenantAndEdgeIDs(key)
		if py.federationService != nil {
			err := py.federationService.unclaimEdge(tenantID, edgeID)
			if err != nil {
				glog.Errorf("Failed to unclaim edge, tenantID=%s, edgeID=%s. Error: %s", tenantID, edgeID, err.Error())
			}
		}
		delete(py.edgeChannelMap, key)
		delete(py.channelEdgeMap, c)
		metrics.WebSocketConnections.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "tenant_id": tenantID, "edge_id": edgeID}).Dec()
		ctx := context.WithValue(context.Background(), base.RequestIDKey, base.GetUUID)
		event := model.EdgeConnectionEvent{ID: base.GetUUID(), TenantID: tenantID, EdgeID: edgeID, Status: false}
		err := base.Publisher.Publish(ctx, &event)
		if err != nil {
			glog.Errorf("Failed to publish event %v. Error: %s", event, err.Error())
		}
	}
}

func NewWsAuditLog(start time.Time, reqID string, origin string, tenantID string, method string, msgName string) *model.AuditLog {
	auditLog := &model.AuditLog{StartedAt: start}
	reqHeader := fmt.Sprintf("Origin: %s", origin)
	auditLog.RequestHeader = &reqHeader
	auditLog.Hostname = os.Getenv("HOSTNAME")
	auditLog.RequestID = reqID
	auditLog.TenantID = tenantID
	auditLog.RequestMethod = method
	auditLog.RequestURL = msgName
	return auditLog
}
func setAuditLogPayload(msg interface{}, auditLog *model.AuditLog) error {
	// convert msg to string
	ba, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	payload := string(ba)
	auditLog.RequestPayload = &payload
	return nil
}

func fillInAuditLog(ctx context.Context, origin string, tenantID string, method string, msgName string, msg interface{}, auditLog *model.AuditLog) error {
	reqID := base.GetRequestID(ctx)
	authContext, err := base.GetAuthContext(ctx)
	if err == nil {
		m := authContext.Claims
		email, ok := m["email"].(string)
		if ok {
			auditLog.UserEmail = email
		}
	} else {
		glog.Warningf(base.PrefixRequestID(ctx, "fillInAuditLog: failed to get auth context, err: %s\n"), err.Error())
	}
	// convert msg to string
	ba, err := json.Marshal(msg)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "fillInAuditLog: failed to marshall message, err: %s\n"), err.Error())
		return err
	}
	payload := string(ba)
	auditLog.RequestPayload = &payload
	reqHeader := fmt.Sprintf("Origin: %s", origin)
	auditLog.RequestHeader = &reqHeader
	auditLog.Hostname = os.Getenv("HOSTNAME")
	auditLog.RequestID = reqID
	auditLog.TenantID = tenantID
	auditLog.RequestMethod = method
	auditLog.RequestURL = msgName
	return nil
}
func (py *wsMessagingServiceImpl) BroadcastMessage(ctx context.Context, tenantID string, msgName string, msg interface{}) {
	start := time.Now()
	origin := os.Getenv("HOSTNAME")
	glog.Infof(base.PrefixRequestID(ctx, "Broadcast message %s to tenant id %s\n"), msgName, tenantID)
	if py.federationService != nil {
		edgeClusterIDs, err := py.dbAPI.SelectAllEdgeClusterIDs(ctx)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "BroadcastMessage: failed to get all edges for tenant %s. Error: %s"), tenantID, err.Error())
			return
		}
		localEdges := []string{}
		remoteEdges := []string{}
		for _, edgeID := range edgeClusterIDs {
			if py.isLocallyConnectedEdge(tenantID, edgeID) {
				localEdges = append(localEdges, edgeID)
			} else if py.federationService.containsEdge(tenantID, edgeID) {
				remoteEdges = append(remoteEdges, edgeID)
			}
		}
		if len(remoteEdges) == 0 {
			if len(localEdges) != 0 {
				py.server.BroadcastTo(tenantID, msgName, msg)
				if false == *config.Cfg.DisableAuditLog {
					py.writeBroadcastAuditLog(ctx, start, origin, tenantID, msgName, msg)
				}
			}
		} else {
			for _, edgeID := range localEdges {
				go py.sendMessageCommon(ctx, origin, tenantID, edgeID, msgName, msg, false, false)
			}
			for _, edgeID := range remoteEdges {
				go py.sendMessageCommon(ctx, origin, tenantID, edgeID, msgName, msg, false, false)
			}
		}
	} else {
		hostname := os.Getenv("HOSTNAME")
		metrics.WebSocketMessageCount.With(prometheus.Labels{"hostname": hostname, "message_name": msgName, "tenant_id": tenantID, "edge_id": "*"}).Inc()
		py.server.BroadcastTo(tenantID, msgName, msg)
		if false == *config.Cfg.DisableAuditLog {
			py.writeBroadcastAuditLog(ctx, start, hostname, tenantID, msgName, msg)
		}
	}
}

func (py *wsMessagingServiceImpl) writeBroadcastAuditLog(ctx context.Context, start time.Time, origin string, tenantID string, msgName string, msg interface{}) error {
	auditLog := &model.AuditLog{StartedAt: start}
	// fill in auditLog
	err2 := fillInAuditLog(ctx, origin, tenantID, "WS_Broadcast", msgName, msg, auditLog)
	if err2 == nil {
		edgeIDs := py.GetConnectedEdgeIDs(tenantID)
		edgeIDsStr := strings.Join(edgeIDs, ",")
		auditLog.EdgeIDs = &edgeIDsStr
		auditLog.FillInTime()
		err2 = py.dbAPI.WriteAuditLog(ctx, auditLog)
	}
	if err2 != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "BroadcastMessage: Failed to write audit log: %+v, err: %s"), *auditLog, err2.Error())
	}
	return err2
}
func (py *wsMessagingServiceImpl) writeP2PAuditLog(ctx context.Context, start time.Time, origin string, tenantID string, edgeID string, msgName string, msg interface{}, ack bool, err error, resp string) error {
	auditLog := &model.AuditLog{StartedAt: start}
	// fill in auditLog
	method := "WS_Emit"
	if ack {
		method = "WS_Ack"
	}
	err2 := fillInAuditLog(ctx, origin, tenantID, method, msgName, msg, auditLog)
	if err2 == nil {
		auditLog.EdgeIDs = &edgeID

		if err == nil {
			auditLog.ResponseCode = 200
			auditLog.ResponseMessage = &resp
		} else {
			auditLog.ResponseCode = 500
			errMsg := err.Error()
			auditLog.ResponseMessage = &errMsg
		}
		auditLog.FillInTime()
		err2 = py.dbAPI.WriteAuditLog(ctx, auditLog)
	}
	if err2 != nil {
		// log it
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to write audit log: %+v, err: %s"), *auditLog, err2.Error())
	}
	return err2
}
func (py *wsMessagingServiceImpl) sendMessageCommon(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}, ack bool, sync bool) (resp string, err error) {
	glog.Infof(base.PrefixRequestID(ctx, "Send message %s to edge %s\n"), msgName, edgeID)
	start := time.Now()
	var c *gosocketio.Channel
	c = py.GetChannel(tenantID, edgeID)
	// handle error when publishing to redis or websocket timeout/ no edge errors
	defer func() {
		if err != nil {
			err := handleWebsocketErrors(tenantID, edgeID, msgName, msg, err)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error: %s"), err.Error())
			}
		}
	}()

	if c != nil {
		if ack {
			resp, err = c.Ack(msgName, msg, WEBSOCKET_SEND_TIMEOUT_SEC*time.Second)
		} else {
			// TODO: Should this func() error out if sync is true for emit?
			err = c.Emit(msgName, msg)
		}
		if msgName == "onDeleteEdge" || msgName == "onDeleteServiceDomain" {
			glog.Infof(base.PrefixRequestID(ctx, "Send message %s: removing websocket for edge %s\n"), msgName, edgeID)
			c.Leave(tenantID)
			py.RemoveChannel(c)
		}
		metrics.WebSocketMessageCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "message_name": msgName, "tenant_id": tenantID, "edge_id": edgeID}).Inc()
		if false == *config.Cfg.DisableAuditLog {
			py.writeP2PAuditLog(ctx, start, origin, tenantID, edgeID, msgName, msg, ack, err, resp)
		}
		return resp, err
	} else if py.federationService != nil && py.federationService.containsEdge(tenantID, edgeID) {
		action := "emit"
		if ack {
			action = "ack"
		}
		// TODO: Should this func() error out if sync is true for emit?
		if msgName == "executeEdgeUpgrade" {
			upgradeObj := msg.(api.ObjectRequest)
			// remove data before sending over to redis
			glog.Infof(base.PrefixRequestID(ctx, "Message of type executeEdgeUpgrade: removing data"))
			var emptyData strfmt.Base64
			upgrademsg := upgradeObj.Doc.(*model.ExecuteEdgeUpgradeData)
			upgrademsg.UpgradeData = &emptyData
			msg = upgradeObj
		}
		hostname := os.Getenv("HOSTNAME")
		msgMetadata := MessageMetadata{
			originHostname: hostname,
			tenantID:       tenantID,
			edgeID:         edgeID,
			action:         action,
			messageKey:     msgName,
			message:        msg,
			sync:           sync,
		}
		resp, err = py.federationService.sendMessageToEdge(ctx, &msgMetadata)
		metrics.WebSocketFederatedMessageCount.With(prometheus.Labels{"hostname": hostname, "message_name": msgName, "tenant_id": tenantID, "edge_id": edgeID}).Inc()
		return resp, err
	}
	glog.Infof(base.PrefixRequestID(ctx, "Send message %s: no channel for edge %s?\n"), msgName, edgeID)
	return "", fmt.Errorf("SendMessage: no channel for edge %s", edgeID)
}

func (py *wsMessagingServiceImpl) SendMessageSync(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}) (string, error) {
	return py.sendMessageCommon(ctx, origin, tenantID, edgeID, msgName, msg, true, true)
}
func (py *wsMessagingServiceImpl) SendMessage(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}) (string, error) {
	return py.sendMessageCommon(ctx, origin, tenantID, edgeID, msgName, msg, true, false)
}
func (py *wsMessagingServiceImpl) EmitMessage(ctx context.Context, origin string, tenantID string, edgeID string, msgName string, msg interface{}) error {
	_, err := py.sendMessageCommon(ctx, origin, tenantID, edgeID, msgName, msg, false, false)
	return err
}

// makeProxyRequest convert http.Request to model.ProxyRequest
func makeProxyRequest(req *http.Request, url string) (preq *model.ProxyRequest, err error) {
	var dump []byte
	dump, err = httputil.DumpRequestOut(req, true)
	if err != nil {
		return
	}
	preq = &model.ProxyRequest{
		URL:     url,
		Request: dump,
	}
	return
}

// sendProxyRequest send the proxy request over websocket to edge,
// return proxy response
func sendProxyRequest(ctx context.Context, c *gosocketio.Channel, preq *model.ProxyRequest) (presp *model.ProxyResponse, err error) {
	var r string
	r, err = c.Ack(model.HTTP_PROXY_MESSAGE, *preq, time.Second*PROXY_SEND_TIMEOUT_SEC)
	if err != nil {
		return
	}
	glog.Infof(base.PrefixRequestID(ctx, "HTTP proxy: got response: %+v"), r)
	p := &model.ProxyResponse{}
	err = json.Unmarshal([]byte(r), p)
	if err != nil {
		return
	}
	presp = p
	return
}

// fromProxyResponse converts http.Response from model.ProxyResponse
func fromProxyResponse(ctx context.Context, presp *model.ProxyResponse, req *http.Request) (resp *http.Response, err error) {
	var r *http.Response
	r, err = http.ReadResponse(bufio.NewReader(bytes.NewReader(presp.Response)), req)
	if err != nil {
		return
	}
	r.Status = presp.Status
	r.StatusCode = presp.StatusCode
	resp = r
	return
}

// SendHTTPRequest main implementation method for http proxy over websocket
// If edge is directed connected to this cloudmgmt instance,
// will send websocket message to the edge.
// If edge is connected to another cloudmgmt instance,
// will make gRPC call to that instance.
func (py *wsMessagingServiceImpl) SendHTTPRequest(ctx context.Context, tenantID string, edgeID string, req *http.Request, url string) (resp *http.Response, err error) {
	glog.Infof(base.PrefixRequestID(ctx, "Send message %s to edge %s, url %s\n"), model.HTTP_PROXY_MESSAGE, edgeID, url)
	// start := time.Now()
	var c *gosocketio.Channel
	var preq *model.ProxyRequest
	var presp *model.ProxyResponse
	c = py.GetChannel(tenantID, edgeID)
	// TODO - also check edge version
	if c != nil {
		// direct connection to edge
		glog.Infof(base.PrefixRequestID(ctx, "Send DIRECT message to edge %s, url %s\n"), edgeID, url)
		preq, err = makeProxyRequest(req, url)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: dump request error: %s"), err)
			return
		}
		presp, err = sendProxyRequest(ctx, c, preq)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: send request error: %s"), err)
			return
		}
		resp, err = fromProxyResponse(ctx, presp, req)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: call failed: %s"), err)
			return
		}
		// success
		return
	} else {
		if py.federationService != nil {
			var IP string
			IP, err = py.federationService.getEdgeOwnerIP(tenantID, edgeID)
			if err == nil && IP != "" && !py.federationService.IsMyIP(IP) {
				// scale-out connection to edge
				glog.Infof(base.PrefixRequestID(ctx, "Send SCALEOUT message to edge %s, url %s\n"), edgeID, url)
				preq, err = makeProxyRequest(req, url)
				if err != nil {
					glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: dump request error: %s"), err)
					return
				}
				endpoint := fmt.Sprintf("%s:%d", IP, *config.Cfg.GRPCPort)
				handler := func(ctx context.Context, conn *grpc.ClientConn) (err error) {
					c := gapi.NewCloudmgmtServiceClient(conn)
					gpreq := &gapi.ProxyRequest{
						TenantId: tenantID,
						EdgeId:   edgeID,
						Url:      url,
						Request:  preq.Request,
					}
					var gresp *gapi.ProxyResponse
					gresp, err = c.SendHTTPRequest(ctx, gpreq)
					if err != nil {
						return
					}
					status := gresp.GetStatus()
					statusCode := gresp.GetStatusCode()
					presp = &model.ProxyResponse{
						Status:     status,
						Response:   gresp.GetResponse(),
						StatusCode: int(statusCode),
					}
					resp, err = fromProxyResponse(ctx, presp, req)
					if err != nil {
						return
					}
					// ok
					return nil
				}
				// This will hit gRPC server on a different cloudmgmt instance
				err = service.CallClientEndpoint(ctx, endpoint, handler)
				if err != nil {
					glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: call client endpoint failed: %s"), err)
					return
				}
				// success
				return
			}
		}
		err = fmt.Errorf("No channel for edge %s", edgeID)
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: failed: %s"), err)
		return
	}
}

func (py *wsMessagingServiceImpl) MakeWebSocketHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		py.server.ServeHTTP(w, r)
	}
}
func (py *wsMessagingServiceImpl) ResyncState() {
	py.mx.Lock()
	defer py.mx.Unlock()
	glog.Infof("resyncState with federation service {")
	err := py.federationService.init()
	if err != nil {
		glog.Errorf("resyncState: federation service init failed, skip")
		return
	}
	for key := range py.edgeChannelMap {
		tenantID, edgeID := extractTenantAndEdgeIDs(key)
		err := py.federationService.claimEdge(tenantID, edgeID)
		if err != nil {
			glog.Errorf("resyncState: failed to claim edge, tenantID=%s, edgeID=%s. Error: %s\n", tenantID, edgeID, err.Error())
		} else {
			glog.Infof("resyncState: done claim edge, tenantID=%s, edgeID=%s.\n", tenantID, edgeID)
		}
	}
	glog.Infof("resyncState with federation service }")
}

// RefreshEdgeClaims refreshes the edge claims in redis before the TTL
func (py *wsMessagingServiceImpl) RefreshEdgeClaims() {
	if py.federationService != nil {
		py.mx.Lock()
		edgeTenants := map[string]string{}
		for key := range py.edgeChannelMap {
			tenantID, edgeID := extractTenantAndEdgeIDs(key)
			edgeTenants[edgeID] = tenantID
		}
		defer py.mx.Unlock()
		err := py.federationService.refreshClaimEdges(edgeTenants)
		if err != nil {
			glog.Errorf("Error in refreshing edge claims for federation ID %s. Error: %s", py.federationService.ID, err.Error())
		}
	}
}

func handleWebsocketErrors(tenantID string, edgeID string, msgName string, msg interface{}, err error) error {
	switch msgName {
	case "executeEdgeUpgrade":
		return handleUpgradeError(tenantID, edgeID, msgName, msg, err)
	}
	return nil
}

// ConfigureWSMessagingService configures websocket server and returns websocket messaging service
func ConfigureWSMessagingService(dbAPI api.ObjectModelAPI, router *httprouter.Router, redisClient *redis.Client) api.WSMessagingService {
	//create
	var federationService *FederationService
	if redisClient != nil {
		federationService = NewFederationService(base.GetUUID(), redisClient)
	}
	transportOpts := transport.GetDefaultWebsocketTransport()
	transportOpts.SendTimeout = 10 * time.Minute
	server := gosocketio.NewServer(transportOpts)

	msgSvc := &wsMessagingServiceImpl{
		server:            server,
		dbAPI:             dbAPI,
		federationService: federationService,
		edgeChannelMap:    make(map[string]*gosocketio.Channel),
		channelEdgeMap:    make(map[*gosocketio.Channel]string),
	}

	api.SetWebsocketService(msgSvc)

	//handle connected
	server.On(gosocketio.OnConnection, func(c *gosocketio.Channel) {
		token := c.RequestHeader().Get("token")
		if token != "" {
			_, err := crypto.VerifyJWT(token)
			if err != nil {
				glog.Warningf("Couldn't authenticate JWT token. Error: %s", err)
				c.Close()
				return
			}
		} else {
			glog.Warningln("Could not find authorization token for this web socket connection")
			//Note: Allowing the connection, so that old edges can connect.We should enable this
			//once we are sure that all old edges have been upgraded to latest version
			//c.Close()
		}
		glog.Infoln("New websocket client connected")
	})

	//on disconnection handler, if client hangs connection unexpectedly, it will still occurs
	//you can omit function args if you do not need them
	//you can return string value for ack, or return nothing for emit
	server.On(gosocketio.OnDisconnection, func(c *gosocketio.Channel) {
		//caller is not necessary, client will be removed from rooms
		//automatically on disconnect
		//but you can remove client from room whenever you need to
		// c.Leave("room name")
		glog.Infoln("Websocket client disconnected")
		msgSvc.RemoveChannel(c)
	})
	//error catching handler
	server.On(gosocketio.OnError, func(c *gosocketio.Channel) {
		glog.Errorln("Error occurs")
	})

	msgSvc.InitEdge()

	msgSvc.InitSensor()

	msgSvc.InitLog()

	msgSvc.InitApplication()

	msgSvc.InitMLModel()

	router.GET("/socket.io/", msgSvc.MakeWebSocketHandler())

	if federationService != nil {
		go federationService.eventloop(func(ctx context.Context, tableName, origin, edgeID, action, msgName string, msg interface{}, sync bool) (string, error) {
			authContext, err := base.GetAuthContext(ctx)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Missing authcontext. Error: %s"), err.Error())
				return "", err
			}
			tenantID := authContext.TenantID
			glog.Infof(base.PrefixRequestID(ctx, "federation event loop got message tenantID=%s, edgeID=%s, action=%s, message name=%s, message=%+v"), tenantID, edgeID, action, msgName, msg)
			if msgName == "executeEdgeUpgrade" {
				var err error
				msg, err = ModifyExecuteEdgeUpgradeData(dbAPI, edgeID, tenantID, msg)
				if err != nil {
					glog.Warningf(base.PrefixRequestID(ctx, "federation event loop modifyExecuteEdgeUpgradeData %s"), err.Error())
					return "", nil
				}
			}
			ctx = context.WithValue(ctx, base.AuditLogTableNameKey, tableName)
			var resp string
			if sync {
				resp, err = msgSvc.SendMessage(ctx, origin, tenantID, edgeID, msgName, msg)
				if err != nil {
					glog.Warningf(base.PrefixRequestID(ctx, "federation event loop send message error %s"), err.Error())
					err1 := handleWebsocketErrors(tenantID, edgeID, msgName, msg, err)
					if err1 != nil {
						glog.Errorf(base.PrefixRequestID(ctx, "Failed to handle websocket error for tenantID=%s, edgeID=%s, action=%s, message name=%s, message=%+v. Error: %s"), tenantID, edgeID, action, msgName, msg, err1.Error())
					}
				}
				return resp, err
			}

			err = msgSvc.EmitMessage(ctx, origin, tenantID, edgeID, msgName, msg)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "federation event loop send message error %s"), err.Error())
				err1 := handleWebsocketErrors(tenantID, edgeID, msgName, msg, err)
				if err1 != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to handle websocket error for tenantID=%s, edgeID=%s, action=%s, message name=%s, message=%+v. Error: %s"), tenantID, edgeID, action, msgName, msg, err1.Error())
				}
			}
			return "", err
		})

		// interval timer - check redis pod restart once a minute
		// resync state if restart detected
		ticker := time.NewTicker(60 * time.Second)
		connRefreshTicker := time.NewTicker(5 * time.Minute)
		quit := make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					if !federationService.isInitialized() {
						// re-sync states
						msgSvc.ResyncState()
					}
				case <-connRefreshTicker.C:
					msgSvc.RefreshEdgeClaims()
				case <-quit:
					ticker.Stop()
					return
				}
			}
		}()

	}
	return msgSvc
}

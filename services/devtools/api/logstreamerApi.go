package api

import (
	"cloudservices/devtools/config"
	"cloudservices/devtools/devtoolsservice"
	gapi "cloudservices/devtools/generated/grpc"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
)

// push /logs/push/
// pull /logs/fetch

var (
	ErrMaxSimultaneousAppsPipelines = errors.New("limit for maximum number of apps/pipelines to be streamed simultaneously reached")
	ErrMaxSimultaneousEndpoints     = errors.New("limit for maximum number of endpoints to be streamed simultaneously reached")
	ErrEndpointCreation             = errors.New("could not create endpoint")
)

var prefixURLpush, prefixURLfetch string

func init() {
	// ex: prefixURLpush = https://debug-test.ntnxsherlock.com/v1.0/logs/push
	prefixURLpush = config.Cfg.EndpointsPrefix + "/v1.0/logs/push"
	prefixURLfetch = config.Cfg.EndpointsPrefix + "/v1.0/logs/fetch"
}

// CreateEndpoints is the rpc handler for starting a log stream
func (srv *apiServer) GetEndpoints(ctx context.Context, req *gapi.GetEndpointsRequest) (*gapi.GetEndpointsResponse, error) {
	ts := time.Now()
	pubKeyVal := devtoolsservice.RedisKey{
		TenantID:    req.TenantID,
		EdgeID:      req.EdgeID,
		ObjectID:    req.ObjectID,
		ContainerID: req.ContainerID,
		Action:      devtoolsservice.PUBLISHER,
		TS:          ts,
	}

	pubKey, err := convertRedisKeyToString(pubKeyVal)
	if err != nil {
		glog.Errorf("Error marshalling redis vale %v to json: %s", pubKeyVal, err.Error())
		return nil, err
	}

	subKeyVal := devtoolsservice.RedisKey{
		TenantID:    req.TenantID,
		EdgeID:      req.EdgeID,
		ObjectID:    req.ObjectID,
		ContainerID: req.ContainerID,
		TS:          ts,
		Action:      devtoolsservice.SUBSCRIBER,
	}

	subKey, err := convertRedisKeyToString(subKeyVal)
	if err != nil {
		glog.Errorf("Error marshalling redis vale %v to json: %s", subKeyVal, err.Error())
		return nil, err
	}

	pubEndpoint := fmt.Sprintf("%s/%s", prefixURLpush, pubKey)
	subEndpoint := fmt.Sprintf("%s/%s", prefixURLfetch, subKey)
	streamName := fmt.Sprintf("%s%s", "stream", pubKey) // Can use subscriber key as well.
	// It just that publisher and subscriber both have to know the stream which they read/write from/to

	// Check if the endpoint can be created after ensuring the per tenant limits are met.
	if err = srv.isEndpointCreatable(req, pubKey); err != nil {
		glog.Errorf("Error checking if endpoint is creatable: %s", err.Error())
		return nil, err
	}

	glog.V(3).Infof("pEndpoint: %s\nsEndpoint: %s", pubEndpoint, subEndpoint)

	// Write pubKey:{stream, peerkey} to Redis
	if err = srv.redisManager.SetRedisKey(pubKey, devtoolsservice.RedisVal{
		RedisStreamName: streamName,
		PeerRedisKey:    subKey,
	}); err != nil {
		glog.Errorf("Error storing pubKey %q in redis: %s", pubKey, err.Error())
		return nil, ErrEndpointCreation
	}

	// Write subKey:{stream, peerkey} to Redis
	if err = srv.redisManager.SetRedisKey(subKey, devtoolsservice.RedisVal{
		RedisStreamName: streamName,
		PeerRedisKey:    pubKey,
	}); err != nil {
		glog.Errorf("Error storing subKey %q in redis: %s", subKey, err.Error())
		return nil, ErrEndpointCreation
	}

	// Create stream key as well so that heartbeat won't fail.
	// During heartbeat we extend the expiry time for stream. If stream not found, we return err
	if err = srv.redisManager.CreateRedisStream(streamName); err != nil {
		glog.Errorf("Error streamCreate %q in redis: %s", streamName, err.Error())
		return nil, ErrEndpointCreation
	}

	return &gapi.GetEndpointsResponse{
		PublisherEndpoint:  pubEndpoint,
		SubscriberEndpoint: subEndpoint,
	}, nil
}

func convertRedisKeyToString(redisEntryKey devtoolsservice.RedisKey) (string, error) {
	hasher := sha1.New()
	redisKeyString := fmt.Sprintf("%s:%s:%s:%s:%s:%s:%d", redisEntryKey.TenantID, redisEntryKey.EdgeID, redisEntryKey.ObjectID, redisEntryKey.ContainerID,
		config.Cfg.Salt, redisEntryKey.TS.String(), redisEntryKey.Action)
	if _, err := hasher.Write([]byte(redisKeyString)); err != nil {
		glog.Errorf("error writing redis key to hasher: %s", err.Error())
		return "", ErrEndpointCreation
	}

	// Remove '/' in base64 string
	base64Str := strings.ToLower(base64.URLEncoding.EncodeToString(hasher.Sum(nil)))
	return strings.Replace(base64Str, "/", "", -1), nil
}

// isEndpointCreatable() is responsible for either allowing the GET /endpoints API
// to go through or failing it because per tenant limits have been reached.
// The following limits are in place:
// MaxSimultaneousAppsPipelines: Limit on the maximum number of apps/pipelines that
//                               can be streamed simulatneously per tenant.
// MaxSimultaneousEndpoints: Limit on the maximum number of active endpoints per tenant.
// NOTE: We track publisher endpoints for book keeping.
func (srv *apiServer) isEndpointCreatable(req *gapi.GetEndpointsRequest, newEndpoint string) error {
	// endpointsGatekeeper uses Redis sets to store the state used to honor
	// the per tenant limits. The state is stored as follows:
	// Set Name: <tenant id>
	// Set contents: <app or pipeline id>/<endpoint id>
	//
	// Algo:
	// 1. When a GET /endpoints request is received, get all the entries for
	//    set name = request.TenantID
	// 2. Use the response to build an in-memory map [<app or pipeline id>][]<endpoint id>
	// 3. For each entry in the map, call MGET on the []<endpoint id>.
	//    NOTE: <endpoint id> is a key in redis with an expiry
	// 4. Check which endpoints are valid and update the map.
	// 5. Using the updated map, ensure there is space for the new endpoint while
	//    adhering to the per tenant limits
	// 6. Push the updated map back into Redis as a set along with the new endpoint
	//    if it got created.
	objectIDEndpointsMap := map[string][]string{}
	setName := req.TenantID
	// 1. Get the set.
	objectIDEndpointsSet, err := srv.redisManager.GetRedisSet(setName)
	if err != nil {
		return ErrEndpointCreation
	}

	// 2. Build the map.
	for _, e := range objectIDEndpointsSet {
		// Extract the app/pipeline id and the endpoint.
		separatorIndex := strings.Index(e, "/")
		objID := e[0:separatorIndex]
		endpoint := e[separatorIndex+1:]
		objectIDEndpointsMap[objID] = append(objectIDEndpointsMap[objID], endpoint)
	}

	// 3. Get the endpoints for each objID in the map to check their validity.
	// 4. Update the map to have only valid endpoints.
	totalValidEndpoints := 0
	for objID, endpoints := range objectIDEndpointsMap {
		redisVals, err := srv.redisManager.MGetRedisKeys(endpoints)
		if err != nil {
			glog.Errorf("Error fetching endpoints for objID %q: %s", objID, err.Error())
			return ErrEndpointCreation
		}
		// Update the map to have only the valid endpoints
		updatedEndpoints := []string{}
		staleEndpoints := []string{}
		for i, redisVal := range redisVals {
			if redisVal.RedisStreamName != "" {
				updatedEndpoints = append(updatedEndpoints, endpoints[i])
			} else {
				// Stash the stale endpoints for deletion from the redis set.
				staleEndpoints = append(staleEndpoints, fmt.Sprintf("%s/%s", objID, endpoints[i]))
			}
		}
		objectIDEndpointsMap[objID] = updatedEndpoints
		totalValidEndpoints += len(updatedEndpoints)

		// Delete the stale endpoints.
		if err = srv.redisManager.DeleteRedisMembersFromSet(setName, staleEndpoints); err != nil {
			return ErrEndpointCreation
		}
	}

	// 5. Using the updated map, check if the new endpoint can be added.
	if _, ok := objectIDEndpointsMap[req.ObjectID]; !ok {
		// App/pipeline is not present in map. See if there is space of another
		// app/pipeline to be streamed.
		if len(objectIDEndpointsMap) >= *srv.cfg.MaxSimultaneousAppsPipelines {
			glog.Errorf("Limit for max number of apps/pipelines (%d) for simultaneous streaming reached for tenant id: %s",
				*srv.cfg.MaxSimultaneousAppsPipelines, req.TenantID)
			return ErrMaxSimultaneousAppsPipelines
		}
	}

	// The app/pipeline is already being streamed. Check if a new endpoint can be created for that app/pipeline.
	if totalValidEndpoints >= *srv.cfg.MaxSimultaneousEndpoints {
		glog.Errorf("Limit for max number of endpoints (%d) for simultaneous streaming reached for tenant id: %s",
			*srv.cfg.MaxSimultaneousEndpoints, req.TenantID)
		return ErrMaxSimultaneousEndpoints
	}

	objectIDEndpointsMap[req.ObjectID] = append(objectIDEndpointsMap[req.ObjectID], newEndpoint)

	// 6. Build the new set from objectIDEndpointsMap and add it to redis.
	newOjectIDEndpointsSet := []string{}
	for objID, endpoints := range objectIDEndpointsMap {
		for _, endpoint := range endpoints {
			newOjectIDEndpointsSet = append(newOjectIDEndpointsSet,
				fmt.Sprintf("%s/%s", objID, endpoint))
		}
	}
	if err = srv.redisManager.CreateRedisSet(setName, newOjectIDEndpointsSet); err != nil {
		return ErrEndpointCreation
	}
	return nil
}

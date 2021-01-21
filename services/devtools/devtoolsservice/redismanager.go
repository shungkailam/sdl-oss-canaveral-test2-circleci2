package devtoolsservice

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
)

const (
	redisKeyExpiryTime          = 120 * time.Second
	maxNumberOfMessagesInStream = 1000 // RFC TODO increase size?
	// If each pushLogs from edge is 4k size then we can store minimum of 4MB logs for container

	userLogsKey       = "logs"
	KeyNotFoundError  = "redis key not found"
	StreamNoDataError = "redis stream newdata not found"
)

// ACTION can be publihser/subscriber
type ACTION int

const (
	PUBLISHER ACTION = iota
	SUBSCRIBER
)

// RedisKey object encapsulates the fields whose hash constitues
// the redis key.
type RedisKey struct {
	TenantID    string
	EdgeID      string
	ObjectID    string
	ContainerID string
	Action      ACTION
	TS          time.Time // To handle streamLogs request from same container multiple times
}

type RedisVal struct {
	RedisStreamName string // Contains actual logs from container
	PeerRedisKey    string // Subscriber/publisher can look up if there is active publisher/subscriber
}

// RedisManager does io operations with redis
type RedisManager struct {
	redisClient *redis.ClusterClient
}

func NewRedisManager(redisClient *redis.ClusterClient) *RedisManager {
	return &RedisManager{
		redisClient: redisClient,
	}
}

// SetRedisKey creates/updates 'key' with value 'val'.
// 'key' is the 'RedisKey' struct converted into a hashed string
// that represents the endpoint that a client publishes to or
// subscribes from.
// 'val' is 'RedisVal' struct that encapsulates the redis stream
// backing the key and the 'PeerRedisKey'.
// For a publisher, 'PeerRedisKey' is the subscriber key and vice versa.
// 'PeerRedisKey' is used to invalidate the peer connections when necessary.
func (redisManager *RedisManager) SetRedisKey(key string, val RedisVal) error {
	valBytes, err := json.Marshal(val)
	if err != nil {
		glog.Errorf("Unable to marshal val to json: %s", err.Error())
		return err
	}
	err = redisManager.redisClient.Set(key, string(valBytes), redisKeyExpiryTime).Err()
	if err != nil {
		glog.Errorf("Unable to store val %+v in redis: %s", val, err.Error())
	}
	return err
}

func (redisManager *RedisManager) CreateRedisStream(streamName string) error {
	vals := make(map[string]interface{})
	vals["stream"] = "start" // Making sure it is different from userLogsKey so that we don't return this to user
	// We can't create empty stream: https://github.com/antirez/redis/issues/4824
	addArgs := &redis.XAddArgs{
		Stream:       streamName,
		MaxLenApprox: maxNumberOfMessagesInStream,
		Values:       vals,
	}
	if _, err := redisManager.redisClient.XAdd(addArgs).Result(); err != nil {
		glog.Errorf("Stream Create error stream: %s err: %s", streamName, err)
		return err
	}

	if err := redisManager.SetRedisKeyExpiry(streamName); err != nil {
		glog.Errorf("Stream key expiration set failed for streamName: %s, err: %s", streamName, err)
		return err
	}
	return nil
}

// GetRedisKey fetches the value stored in Redis for 'key'.
func (redisManager *RedisManager) GetRedisKey(key string) (RedisVal, error) {
	val := RedisVal{}
	valString, err := redisManager.redisClient.Get(key).Result()
	if err != nil {
		glog.Errorf("Error while fetching redis key %q: %#v", key, err)
		if err == redis.Nil {
			return val, errors.New(KeyNotFoundError)
		}
		return val, err
	}
	if err = json.Unmarshal([]byte(valString), &val); err != nil {
		glog.Errorf("Error unmarshaling redis val %q into RedisVal struct: %s",
			valString, err.Error())
	}
	return val, err
}

func (redisManager *RedisManager) MGetRedisKeys(keys []string) ([]RedisVal, error) {
	redisVals := make([]RedisVal, len(keys))
	if len(keys) == 0 {
		return redisVals, nil
	}
	for i, key := range keys {
		redisVals[i], _ = redisManager.GetRedisKey(key)
	}
	return redisVals, nil
}

func (redisManager *RedisManager) DeleteRedisMembersFromSet(setName string, members []string) error {
	if len(members) == 0 {
		return nil
	}
	setInterface := make([]interface{}, len(members))
	for i, v := range members {
		setInterface[i] = v
	}
	_, err := redisManager.redisClient.SRem(setName, setInterface...).Result()
	if err != nil {
		glog.Errorf("Error deleting stale endpoints from set %q: %s", setName, err.Error())
	}
	return err
}

// CreateRedisSet will create a list of name 'setName'.
func (redisManager *RedisManager) CreateRedisSet(setName string, setElements []string) error {
	if len(setElements) == 0 {
		return nil
	}
	setInterface := make([]interface{}, len(setElements))
	for i, v := range setElements {
		setInterface[i] = v
	}
	_, err := redisManager.redisClient.SAdd(setName, setInterface...).Result()
	if err != nil {
		glog.Errorf("Error creating set %q: %s", setName, err.Error())
		return err
	}
	return nil
}

// SetRedisKeyExpiry sets the expiry of 'key' to 'expiration'.
func (redisManager *RedisManager) SetRedisKeyExpiry(key string) error {
	err := redisManager.redisClient.Expire(key, redisKeyExpiryTime).Err()
	if err != nil {
		glog.Errorf("Cannot set expiry of key %q to %d", key, redisKeyExpiryTime)
	}
	return err
}

// GetStreamLogs gets streamdata after minTimstamp
func (redisManager *RedisManager) GetStreamLogs(streamName, minTimestamp string) (string, string, error) {
	readArgs := &redis.XReadArgs{
		Streams: []string{streamName, minTimestamp},
		Count:   100, // Read 100 chunks at a time so that payload size is limited
		Block:   -1,  // Makes XRead non-blocking
	}
	res := redisManager.redisClient.XRead(readArgs)
	if res.Err() != nil {
		err := res.Err()
		glog.Errorf("XRead Error: %s", err)
		if err == redis.Nil {
			return "", "", errors.New(StreamNoDataError)
		}
		return "", "", err
	}
	streams, err := res.Result()
	if err != nil || len(streams) != 1 {
		if err == nil {
			return "", "", fmt.Errorf("Invalid number of streams returned: %d", len(streams))
		}
		glog.Errorf("StreamsResult Error: %s", err)
		return "", "", err
	}

	var buffer bytes.Buffer
	var valStr string
	var ok bool
	for _, msg := range streams[0].Messages {
		// msg is redis.XMessage which contains Values of type map[string]interface{}
		for key, val := range msg.Values {
			if key == userLogsKey {
				if valStr, ok = val.(string); !ok {
					glog.Errorf("Error in converting interface to string: %v", val)
					continue
				}
				buffer.WriteString(valStr)
			}
			// else not inserted by user. We insert first value with key "stream"
		}
	}

	// len(streams[0].Messages) always >= 1. If there are no messages, then redis throws an error!
	latestTS := streams[0].Messages[len(streams[0].Messages)-1].ID
	return buffer.String(), latestTS, nil
}

func (redisManager *RedisManager) PutStreamData(pubKey, data string) error {
	// Get publisher key
	pubVal, err := redisManager.GetRedisKey(pubKey)
	if err != nil {
		glog.Errorf("GetRedisKey failed for key: %s, err: %s", pubKey, err)
		return err
	}
	// Check if corresponding subscriber is alive
	if _, err = redisManager.GetRedisKey(pubVal.PeerRedisKey); err != nil {
		// TODO Error handling. What if redis is not reachable?
		return errors.New("Subscriber not alive")
	}

	vals := make(map[string]interface{})
	vals[userLogsKey] = data // RedisStreams store data as [](K, V).
	// We are storing all logs with same key in the stream but their IDs are different

	addArgs := &redis.XAddArgs{
		Stream: pubVal.RedisStreamName,
		Values: vals,
	}
	if err = redisManager.redisClient.XAdd(addArgs).Err(); err != nil {
		glog.Errorf("Stream ADD error for pubKey: %s stream: %s err: %s", pubKey, pubVal.RedisStreamName, err)
		return err
	}

	return nil
}

func (redisManager *RedisManager) RecordHeartbeat(key string) error {
	// We need to check if subscriber persent before increasing expiry time for pub key
	// Get publisher key
	pubVal, err := redisManager.GetRedisKey(key)
	if err != nil {
		glog.Errorf("GetRedisKey failed for key: %s, err: %s", key, err)
		return err
	}
	// Check if corresponding subscriber is alive
	if _, err = redisManager.GetRedisKey(pubVal.PeerRedisKey); err != nil {
		// TODO Error handling. What if redis is not reachable?
		return errors.New("Subscriber not alive")
	}

	if done, err := redisManager.redisClient.Expire(key, redisKeyExpiryTime).Result(); err != nil {
		glog.Errorf("Cannot set expiry of key %q to %d", key, redisKeyExpiryTime)
		return err
	} else if !done {
		glog.Errorf("Cannot set expiry of key: %q. Likely key expired!", key)
		return errors.New("Heartbeat failed")
	}

	// Set new expiry time for stream. This is to ensure streams get deleted when publisher gets deleted
	if done, err := redisManager.redisClient.Expire(pubVal.RedisStreamName, redisKeyExpiryTime).Result(); err != nil {
		glog.Errorf("Cannot set expiry of streamkey %q to %d", pubVal.RedisStreamName, redisKeyExpiryTime)
		return err
	} else if !done {
		glog.Errorf("Cannot set expiry of streamkey: %q. Likely that key expired!", key)
		return errors.New("Heartbeat failed")
	}
	return nil
}

// GetRedisSet returns all the elements of set 'setName'.
func (redisManager *RedisManager) GetRedisSet(setName string) ([]string, error) {
	setMembers := []string{}
	// Check if the set exists
	res, err := redisManager.redisClient.Exists(setName).Result()
	if err != nil {
		glog.Errorf("Error fetching redis key %q: %s", setName, err.Error())
		return setMembers, err
	}
	if res == 0 {
		// Set does not exist
		return setMembers, nil
	}

	setMembers, err = redisManager.redisClient.SMembers(setName).Result()
	if err != nil {
		glog.Errorf("Error fetching redis set %q: %s", setName, err.Error())
	}
	return setMembers, err
}

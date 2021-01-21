package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/go-redis/redis"
)

const (
	// redisUnlockScript is redis lua script to release a lock
	redisUnlockScript = `
		if redis.call("get", KEYS[1]) == ARGV[1] then
				return redis.call("del", KEYS[1])
		else
				return 0
		end
		`
)

// simple random string used in redis locking
func randomString() string {
	t := time.Now().UnixNano()
	rnd := rand.New(
		rand.NewSource(t))
	i := rnd.Int()
	return fmt.Sprintf("%d:%d", t, i)
}

// RedisLock - lock the redis resource key for ttlMillis milli-seconds
// return (string, bool)
// On success, bool is set to true and the return string
// can be used to unlock the resource key
func RedisLock(client *redis.Client, resource string, ttlMillis int) (string, bool) {
	val := randomString()
	reply := client.SetNX(resource, val, time.Duration(ttlMillis)*time.Millisecond)
	b := reply.Err() == nil && reply.Val()
	if b {
		return val, b
	}
	return "", false
}

// RedisUnlock - unlock the redis resource key using the val
// return true on success
func RedisUnlock(client *redis.Client, resource string, val string) bool {
	reply := client.Eval(redisUnlockScript, []string{resource}, val)
	v, err := reply.Int()
	return err == nil && v == 1
}

// RedisUnmarshal - unmarshal redis Get result into v
// v stored in redis using Set should implement BinaryMarshaler
// return redis.Nil if key does not exist
func RedisUnmarshal(client *redis.Client, key string, v interface{}) error {
	val, err := client.Get(key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(val), v)
}

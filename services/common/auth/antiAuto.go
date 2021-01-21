package auth

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis"
)

func getNewLoginFailureInfoExpiration(duration time.Duration) float64 {
	return float64(time.Now().UTC().UnixNano() + int64(duration))
}
func newLoginFailureInfo(duration time.Duration) loginFailureInfo {
	return loginFailureInfo{
		Expiration:   getNewLoginFailureInfoExpiration(duration),
		FailureCount: 0,
	}
}
func getRedisKeyForEmail(email string) string {
	return fmt.Sprintf("anti-auto-%s", email)
}

type loginFailureInfo struct {
	Expiration   float64 `json:"expiration"`
	FailureCount int     `json:"failureCount"`
}

func (info loginFailureInfo) IsExpired() bool {
	now := float64(time.Now().UTC().UnixNano())
	return now > info.Expiration
}
func (info loginFailureInfo) IsLocked(threshold int) bool {
	return !info.IsExpired() && info.FailureCount >= threshold
}
func (info loginFailureInfo) MarshalBinary() (data []byte, err error) {
	return json.Marshal(info)
}
func (info loginFailureInfo) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, &info)
}

// LoginTracker struct to help track login failures and email locking
type LoginTracker struct {
	failureMap  map[string]loginFailureInfo
	redisClient *redis.Client
	// After failureCountThreshold of login errors within duration lockDuration,
	// the user account will be locked from login for duration lockDuration.
	// This is to throttle bad login calls to reduce risk of brute force login attacks.
	failureCountThreshold int
	// Lock duration for email
	lockDuration time.Duration
	mx           sync.Mutex
}

// NewLoginTracker create a new LoginTracker
func NewLoginTracker(redisClient *redis.Client, failureCountThreshold int, lockDuration time.Duration) *LoginTracker {
	fmt.Printf(">>> NewLoginTracker, redis nil? %t\n", redisClient == nil)
	failureMap := make(map[string]loginFailureInfo)
	return &LoginTracker{failureMap: failureMap, redisClient: redisClient, failureCountThreshold: failureCountThreshold, lockDuration: lockDuration}
}

// IsLoginLocked check whether the given email is currently locked from login
func (tracker *LoginTracker) IsLoginLocked(email string) bool {
	tracker.mx.Lock()
	defer tracker.mx.Unlock()
	key := getRedisKeyForEmail(email)
	if tracker.redisClient == nil {
		info := tracker.failureMap[key]
		return info.IsLocked(tracker.failureCountThreshold)
	}
	res, err := tracker.redisClient.Get(key).Result()
	if err != nil {
		// typically redis.Nil = not found
		return false
	}
	info := loginFailureInfo{}
	err = json.Unmarshal([]byte(res), &info)
	if err != nil {
		// should not happen, allow login to proceed in this case
		return false
	}
	return info.IsLocked(tracker.failureCountThreshold)
}
func (tracker *LoginTracker) getLoginFailureInfo(email string) loginFailureInfo {
	key := getRedisKeyForEmail(email)
	if tracker.redisClient == nil {
		return tracker.failureMap[key]
	}
	res, err := tracker.redisClient.Get(key).Result()
	if err != nil {
		// typically redis.Nil = not found
		return newLoginFailureInfo(tracker.lockDuration)
	}
	info := loginFailureInfo{}
	err = json.Unmarshal([]byte(res), &info)
	if err != nil {
		// should not happen, treat this as not found
		return newLoginFailureInfo(tracker.lockDuration)
	}
	return info
}

// UpdateLoginFailureInfo increment the login failure count for the email,
// may cause the email to be locked from login
func (tracker *LoginTracker) UpdateLoginFailureInfo(email string) error {
	tracker.mx.Lock()
	defer tracker.mx.Unlock()
	info := tracker.getLoginFailureInfo(email)
	if info.IsExpired() {
		info = newLoginFailureInfo(tracker.lockDuration)
	} else {
		info.Expiration = getNewLoginFailureInfoExpiration(tracker.lockDuration)
	}
	info.FailureCount++
	key := getRedisKeyForEmail(email)
	if tracker.redisClient == nil {
		tracker.failureMap[key] = info
		return nil
	}
	return tracker.redisClient.Set(key, info, tracker.lockDuration).Err()
}

package auth_test

import (
	"cloudservices/common/auth"
	"testing"
	"time"

	"github.com/go-redis/redis"
)

func TestAntiAuto(t *testing.T) {
	const failureCountThreshold = 5
	const lockDuration = 10 * time.Second

	email := "foo@example.com"

	var redisClient *redis.Client

	// // if running redis locally, can uncomment the following to test
	// redisClient = redis.NewClient(&redis.Options{
	// 	Addr:     "localhost:6379",
	// 	Password: "", // no password set
	// 	DB:       0,  // use default DB
	// })

	loginTracker := auth.NewLoginTracker(redisClient, failureCountThreshold, lockDuration)

	for i := 0; i < failureCountThreshold; i++ {
		locked := loginTracker.IsLoginLocked(email)
		if locked {
			t.Fatalf("expect email to not be locked at iteration %d", i)
		}
		loginTracker.UpdateLoginFailureInfo(email)
	}
	locked := loginTracker.IsLoginLocked(email)
	if false == locked {
		t.Fatal("expect email to be locked at loop")
	}
	// wait for auto unlock
	time.Sleep(lockDuration)
	locked = loginTracker.IsLoginLocked(email)
	if locked {
		t.Fatal("expect email to unlock automatically after lock duration")
	}
}

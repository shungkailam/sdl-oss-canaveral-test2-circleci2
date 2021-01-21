package base_test

import (
	"cloudservices/common/base"
	"context"
	"sync"
	"testing"
	"time"
)

var mutex sync.Mutex
var counter int

func incCount() int {
	mutex.Lock()
	defer mutex.Unlock()
	counter++
	return counter
}

func getCount() int {
	mutex.Lock()
	defer mutex.Unlock()
	return counter
}

func schedule(ctx context.Context) {
	base.ScheduleJob(ctx, "test-job", func(ctx context.Context) {
		incCount()
		schedule(ctx)
	}, time.Millisecond*100)
}

func TestScheduler(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	schedule(ctx)
	time.Sleep(time.Second)
	lastCount := getCount()
	idx := 0
	for {
		time.Sleep(time.Second)
		currCount := getCount()
		if currCount == lastCount {
			t.Fatalf("counter expected to increase. found %d", currCount)
		}
		lastCount = currCount
		idx++
		if idx >= 2 {
			cancelFunc()
			break
		}
	}
	// wait for the scheduler to exit
	time.Sleep(time.Second)
	lastCount = getCount()
	time.Sleep(time.Second)
	currCount := getCount()
	if currCount != lastCount {
		t.Fatalf("counter expected to remain the same. expected %d, found %d", lastCount, currCount)
	}
}

package event

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type TestEventsUpserter struct {
	t      *testing.T
	events chan model.Event
}

func (upserter *TestEventsUpserter) UpsertEvents(
	context context.Context,
	docs model.EventUpsertRequest,
	callback func(context.Context, interface{}) error,
) (
	[]model.Event,
	error,
) {
	for _, evt := range docs.Events {
		upserter.events <- evt
	}
	return docs.Events, nil
}

type ExpectedEvent struct {
	Type string
	Path string
	Msg  string
}

type TestEntry struct {
	SvcDomainID    string
	NodeID         string
	NodeVersion    string
	TotalMemoryKB  string
	MemoryFreeKB   string
	TotalStorageKB string
	StorageFreeKB  string
	Events         []ExpectedEvent
}

func TestThresholds(t *testing.T) {
	tenantID := base.GetUUID()
	ctx := base.GetAdminContext(base.GetUUID(), tenantID)
	publisher := base.NewEventPublisher()
	ch := make(chan model.Event)
	err := publisher.Subscribe(
		&NodeInfoEventListener{
			dbAPI: &TestEventsUpserter{
				t:      t,
				events: ch,
			},
		},
	)
	require.NoError(t, err)

	table := []TestEntry{
		{
			SvcDomainID:    "123",
			NodeID:         "456",
			NodeVersion:    "v1.5.0",
			TotalMemoryKB:  "100",
			MemoryFreeKB:   "10",
			TotalStorageKB: "100",
			StorageFreeKB:  "10",
			Events: []ExpectedEvent{
				{"STATUS", "/serviceDomain:123/node:456/status/memoryUsage", "Memory usage low at 90 percent"},
				{"STATUS", "/serviceDomain:123/node:456/status/storageUsage", "Storage usage low at 90 percent"},
			},
		},
		{
			SvcDomainID:    "123",
			NodeID:         "456",
			NodeVersion:    "v1.7.0",
			TotalMemoryKB:  "100",
			MemoryFreeKB:   "1",
			TotalStorageKB: "100",
			StorageFreeKB:  "1",
			Events: []ExpectedEvent{
				{"ALERT", "/serviceDomain:123/node:456/status/memoryUsage", "Memory usage critically high at 99 percent"},
				{"ALERT", "/serviceDomain:123/node:456/status/storageUsage", "Storage usage critically high at 99 percent"},
			},
		},
		{
			SvcDomainID:    "123",
			NodeID:         "456",
			NodeVersion:    "v2.0.0",
			TotalMemoryKB:  "100",
			MemoryFreeKB:   "1",
			TotalStorageKB: "100",
			StorageFreeKB:  "1",
			Events: []ExpectedEvent{
				{"ALERT", "/serviceDomain:123/node:456/status/memoryUsage", "Memory usage critically high at 99 percent"},
				{"ALERT", "/serviceDomain:123/node:456/status/storageUsage", "Storage usage critically high at 99 percent"},
			},
		},
		{
			SvcDomainID:    "123",
			NodeID:         "456",
			NodeVersion:    "v1.6.0",
			TotalMemoryKB:  "100",
			MemoryFreeKB:   "1",
			TotalStorageKB: "100",
			StorageFreeKB:  "1",
			Events: []ExpectedEvent{
				{"STATUS", "/serviceDomain:123/node:456/status/memoryUsage", "Memory usage low at 99 percent"},
				{"ALERT", "/serviceDomain:123/node:456/status/storageUsage", "Storage usage critically high at 99 percent"},
			},
		},
		{
			SvcDomainID:    "123",
			NodeID:         "123",
			NodeVersion:    "v2.0.0",
			TotalMemoryKB:  "100",
			MemoryFreeKB:   "7",
			TotalStorageKB: "100",
			StorageFreeKB:  "7",
			Events:         []ExpectedEvent{},
		},
	}
	for _, e := range table {
		evt := &model.NodeInfoEvent{
			ID: base.GetUUID(),
			Info: &model.NodeInfo{
				NodeEntityModel: model.NodeEntityModel{
					ServiceDomainEntityModel: model.ServiceDomainEntityModel{
						SvcDomainID: e.SvcDomainID,
					},
					NodeID: e.NodeID,
				},
				NodeInfoCore: model.NodeInfoCore{
					TotalMemoryKB:  e.TotalMemoryKB,
					MemoryFreeKB:   e.MemoryFreeKB,
					TotalStorageKB: e.TotalStorageKB,
					StorageFreeKB:  e.StorageFreeKB,
					NodeVersion:    &e.NodeVersion,
				},
			},
		}
		err = publisher.Publish(ctx, evt)
		require.NoError(t, err)
		for _, expectedEvent := range e.Events {
			event := <-ch
			t.Logf("Received event: %#v", event)
			if expectedEvent.Type != event.Type {
				t.Fatalf("Expected event of type %s, got %s",
					expectedEvent.Type, event.Type)
			}
			if expectedEvent.Path != event.Path {
				t.Fatalf("Expected event path %s, got %s",
					expectedEvent.Path, event.Path)
			}
			if expectedEvent.Msg != event.Message {
				t.Fatalf("Expected event message %s, got %s",
					expectedEvent.Msg, event.Message)
			}
		}
	}
	select {
	case <-time.After(time.Second):
		return
	case <-ch:
		t.Fatal("No events expected")
	}
}

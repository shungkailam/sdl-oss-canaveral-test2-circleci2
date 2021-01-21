package event

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
)

// Event names to used in both Event and Listener.
// It implements base.Event
const (
	LowThreshold  = 90 // clear alert at 90%
	HighThreshold = 95 // raise alert at 95%
)

type EventsUpserter interface {
	UpsertEvents(context context.Context, docs model.EventUpsertRequest, callback func(context.Context, interface{}) error) ([]model.Event, error)
}

// NodeInfoEventListener is the listener for node info event
type NodeInfoEventListener struct {
	dbAPI EventsUpserter
}

// getEventPath returns the event path
func getEventPath(svcDomainID, nodeID, suffix string) string {
	suffix = strings.Trim(suffix, "/")
	return fmt.Sprintf("/serviceDomain:%s/node:%s/%s", svcDomainID, nodeID, suffix)
}

func (listener *NodeInfoEventListener) upsertEvent(
	ctx context.Context,
	event model.Event,
	nodeID string,
) error {
	event.Timestamp = time.Now().UTC()
	event.Version = "v1"
	response, err := listener.dbAPI.UpsertEvents(
		ctx,
		model.EventUpsertRequest{
			Events: []model.Event{event},
		},
		nil,
	)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upsert events for node %s. Error: %s"),
			nodeID, err.Error())
	} else if len(response) != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upsert event for node %s"),
			nodeID)
	}
	return err
}

func (listener *NodeInfoEventListener) checkMemoryThresholds(ctx context.Context, event *model.NodeInfoEvent) error {
	var totalMemoryKB int
	var memoryFreeKB int

	// Check whether we're inbetween thresholds
	info := event.Info
	svcDomainID := info.SvcDomainID
	nodeID := info.NodeID
	if _, err := fmt.Sscanf(info.TotalMemoryKB, "%d", &totalMemoryKB); err != nil {
		return fmt.Errorf("Failed to unmarshal NodeInfo.TotalMemoryKB '%s' for node %s: %s",
			info.TotalMemoryKB, nodeID, err)
	}
	if _, err := fmt.Sscanf(info.MemoryFreeKB, "%d", &memoryFreeKB); err != nil {
		return fmt.Errorf("Failed to unmarshal NodeInfo.MemoryFreeKB '%s' for node %s: %s",
			info.MemoryFreeKB, nodeID, err)
	}

	if memoryFreeKB > totalMemoryKB {
		return fmt.Errorf("NodeInfo.MemoryFreeKB (%s) >  NodeInfo.TotalMemoryKB (%s) for node %s",
			info.MemoryFreeKB, info.TotalMemoryKB, nodeID)
	}

	usedMemoryKB := totalMemoryKB - memoryFreeKB
	usedMemoryPct := (100 * usedMemoryKB) / totalMemoryKB

	path := getEventPath(svcDomainID, nodeID, "status/memoryUsage")

	// U2 nodes don't report version (ENG-220498)
	if info.NodeVersion == nil {
		return nil
	}
	// only create ALERT if edge version supports this feature
	feats, _ := api.GetFeaturesForVersion(*info.NodeVersion)

	if usedMemoryPct <= LowThreshold || !feats.HighMemAlert {
		// Cleary any alert
		return listener.upsertEvent(
			ctx,
			model.Event{
				Type:  "STATUS",
				Path:  path,
				State: "",
				Message: fmt.Sprintf("Memory usage low at %d percent",
					usedMemoryPct),
			},
			nodeID,
		)
	}

	if usedMemoryPct <= HighThreshold {
		// We're in nomans land. Unable to either clear or set alert
		return nil
	}

	if feats.HighMemAlert {
		return listener.upsertEvent(
			ctx,
			model.Event{
				Type:     "ALERT",
				Path:     path,
				State:    "OPEN",
				Severity: "CRITICAL",
				Message: fmt.Sprintf("Memory usage critically high at %d percent",
					usedMemoryPct),
			},
			nodeID,
		)
	}
	return nil

}

func (listener *NodeInfoEventListener) checkStorageThresholds(ctx context.Context, event *model.NodeInfoEvent) error {
	var totalStorageKB int
	var storageFreeKB int

	info := event.Info
	svcDomainID := info.SvcDomainID
	nodeID := info.NodeID
	// Check whether we're inbetween thresholds
	if _, err := fmt.Sscanf(info.TotalStorageKB, "%d", &totalStorageKB); err != nil {
		return fmt.Errorf("Failed to unmarshal NodeInfo.TotalStorageKB '%s' for node %s: %s",
			info.TotalStorageKB, nodeID, err)
	}
	if _, err := fmt.Sscanf(info.StorageFreeKB, "%d", &storageFreeKB); err != nil {
		return fmt.Errorf("Failed to unmarshal NodeInfo.StorageFreeKB '%s' for node %s: %s",
			info.StorageFreeKB, nodeID, err)
	}

	if storageFreeKB > totalStorageKB {
		return fmt.Errorf("NodeInfo.StorageFreeKB (%s) >  NodeInfo.TotalStorageKB (%s) for node %s",
			info.StorageFreeKB, info.TotalStorageKB, nodeID)
	}

	usedStorageKB := totalStorageKB - storageFreeKB
	usedStoragePct := (100 * usedStorageKB) / totalStorageKB

	path := getEventPath(svcDomainID, nodeID, "status/storageUsage")

	if usedStoragePct <= LowThreshold {
		// Cleary any alert
		return listener.upsertEvent(
			ctx,
			model.Event{
				Type:  "STATUS",
				Path:  path,
				State: "",
				Message: fmt.Sprintf("Storage usage low at %d percent",
					usedStoragePct),
			},
			nodeID,
		)
	}

	if usedStoragePct <= HighThreshold {
		// We're in nomans land. Unable to either clear or set alert
		return nil
	}

	return listener.upsertEvent(
		ctx,
		model.Event{
			Type:     "ALERT",
			Path:     path,
			State:    "OPEN",
			Severity: "CRITICAL",
			Message: fmt.Sprintf("Storage usage critically high at %d percent",
				usedStoragePct),
		},
		nodeID,
	)
}

func (listener *NodeInfoEventListener) OnEvent(
	ctx context.Context,
	event base.Event,
) error {
	var errMsgs []string

	nodeInfoEvent, ok := event.(*model.NodeInfoEvent)
	if !ok {
		glog.Infof(base.PrefixRequestID(ctx, "Unhandled event %+v"), event)
		return nil
	}
	ctx = base.GetAdminContextWithTenantID(ctx, nodeInfoEvent.Info.TenantID)
	err := listener.checkMemoryThresholds(ctx, nodeInfoEvent)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	err = listener.checkStorageThresholds(ctx, nodeInfoEvent)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	if len(errMsgs) != 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return err
}

func (listener *NodeInfoEventListener) EventName() string {
	return model.NodeInfoEventName
}

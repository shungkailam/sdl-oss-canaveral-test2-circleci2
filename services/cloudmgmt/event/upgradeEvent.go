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

// Event names to used in both Event and Listener
const (
	UpgradeSuccess = "Success"
	UpgradeFailed  = "Failed"
)

// UpgradeEventListener is the listener for edge event.
// It implements base.Event
type UpgradeEventListener struct {
	dbAPI api.ObjectModelAPI
}

func (listener *UpgradeEventListener) upsertEvents(ctx context.Context, event *model.UpgradeEvent) error {
	dbAPI := listener.dbAPI
	timestamp := time.Now().UTC()
	if event.EventState == UpgradeSuccess {
		event.EventType = "STATUS"
		event.Message = "Sending upgrade message to the Edge"
		event.State = "Upgrading"
	} else if event.EventState == UpgradeFailed {
		event.EventType = "ALERT"
		event.State = "Failed"
		if event.Err != nil {
			event.Message = fmt.Sprintf("Websocket send failed %s", event.Err.Error())
		} else {
			event.Message = "Websocket send failed"
		}
	} else {
		return fmt.Errorf("Unknown event state %s, for edge upgrade", event.EventState)
	}
	request := model.EventUpsertRequest{
		Events: []model.Event{
			{
				Type:      event.EventType,
				Path:      fmt.Sprintf("/serviceDomain:%s/upgrade:%s:%s/event", event.EdgeID, event.ReleaseVersion, event.ID),
				State:     event.State,
				Version:   "v1",
				Message:   event.Message,
				Timestamp: timestamp,
			},
			{
				Type:      "STATUS",
				Path:      fmt.Sprintf("/serviceDomain:%s/upgrade:%s:%s/progress", event.EdgeID, event.ReleaseVersion, event.ID),
				State:     event.State,
				Version:   "v1",
				Message:   "0%",
				Timestamp: timestamp,
			},
		},
	}
	response, err := dbAPI.UpsertEvents(ctx, request, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upsert events for edge %s. Error: %s"), event.EdgeID, err.Error())
	} else if len(response) != 2 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upsert event for edge %s"), event.EdgeID)
	}
	return err
}

func (listener *UpgradeEventListener) OnEvent(ctx context.Context, event base.Event) error {
	var errMsgs []string
	upgradeEvent, ok := event.(*model.UpgradeEvent)
	if !ok {
		glog.Infof(base.PrefixRequestID(ctx, "Unhandled event %+v"), event)
		return nil
	}
	glog.Infof(base.PrefixRequestID(ctx, "Received upgrade failed event: %+v"), upgradeEvent)
	ctx = base.GetAdminContextWithTenantID(ctx, upgradeEvent.TenantID)
	err := listener.upsertEvents(ctx, upgradeEvent)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	if len(errMsgs) != 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return err
}

func (listener *UpgradeEventListener) EventName() string {
	return model.UpgradeEventName
}

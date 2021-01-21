package event

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/events"
	"cloudservices/common/model"

	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
)

// EdgeConnectionEventListener is the listener for edge event.
// It implements base.Event
type EdgeConnectionEventListener struct {
	dbAPI api.ObjectModelAPI
}

func (listener *EdgeConnectionEventListener) upsertEvents(ctx context.Context, event *model.EdgeConnectionEvent) error {
	dbAPI := listener.dbAPI
	state := "DISCONNECTED"
	evType := "ALERT"
	message := "Service Domain got disconnected"
	if event.Status {
		state = "CONNECTED"
		evType = "STATUS"
		message = "Service Domain got connected"
	}
	timestamp := time.Now().UTC()
	properties := make(map[string]string)
	properties[events.TTL] = "1440h" // 2 months
	request := model.EventUpsertRequest{
		Events: []model.Event{
			{
				Timestamp:  timestamp,
				Type:       evType,
				Path:       fmt.Sprintf("/serviceDomain:%s/status/cloudConnection", event.EdgeID),
				State:      state,
				Message:    message,
				Version:    "v1",
				Properties: properties,
			},
		},
	}
	response, err := dbAPI.UpsertEvents(ctx, request, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upsert events for edge %s. Error: %s"), event.EdgeID, err.Error())
	} else if len(response) != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to upsert event for edge %s"), event.EdgeID)
	}
	return err
}

func (listener *EdgeConnectionEventListener) updatePendingLogs(ctx context.Context, event *model.EdgeConnectionEvent) error {
	dbAPI := listener.dbAPI
	// The AWS presignd URL for log upload is 45 mins. Hence, increasing it to 5 + 40 mins.
	return dbAPI.ScheduleTimeOutPendingLogsJob(ctx, time.Minute*5, time.Minute*40)
}

func (listener *EdgeConnectionEventListener) updateKubernetesClusterKubeVersion(ctx context.Context, event *model.EdgeConnectionEvent) error {
	dbAPI := listener.dbAPI
	err := dbAPI.UpdateKubernetesClusterKubeVersion(ctx, event.EdgeID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update kubernetes cluster kube version for cluster. Error: %s"), event.EdgeID, err.Error())
	}
	return err
}

func (listener *EdgeConnectionEventListener) OnEvent(ctx context.Context, event base.Event) error {
	var errMsgs []string
	edgeEvent, ok := event.(*model.EdgeConnectionEvent)
	if !ok {
		glog.Infof(base.PrefixRequestID(ctx, "Unhandled event %+v"), event)
		return nil
	}
	if edgeEvent.Status {
		glog.Infof(base.PrefixRequestID(ctx, "Received edge connected event: %+v"), edgeEvent)
	} else {
		glog.Infof(base.PrefixRequestID(ctx, "Received edge disconnected event: %+v"), edgeEvent)
	}
	ctx = base.GetAdminContextWithTenantID(ctx, edgeEvent.TenantID)
	err := listener.updatePendingLogs(ctx, edgeEvent)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	err = listener.upsertEvents(ctx, edgeEvent)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	err = listener.updateKubernetesClusterKubeVersion(ctx, edgeEvent)
	if err != nil {
		errMsgs = append(errMsgs, err.Error())
	}
	if len(errMsgs) != 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return err
}

func (listener *EdgeConnectionEventListener) EventName() string {
	return model.EdgeConnectionEventName
}

package event

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"strings"

	"github.com/golang/glog"
)

// EntityCRUDEventListener is the listener for entity crud event.
// It implements base.Event
type EntityCRUDEventListener struct {
	dbAPI api.ObjectModelAPI
}

func (listener *EntityCRUDEventListener) OnEvent(ctx context.Context, event base.Event) error {
	entityCRUDEvent, ok := event.(*model.EntityCRUDEvent)
	if !ok {
		glog.Infof(base.PrefixRequestID(ctx, "Unhandled event %+v"), event)
		return nil
	}
	if strings.HasPrefix(entityCRUDEvent.Message, "onDelete") {
		// Purge from S3
		return listener.dbAPI.PurgeFiles(ctx, entityCRUDEvent.TenantID, entityCRUDEvent.EntityID)
	}
	return nil
}

func (listener *EntityCRUDEventListener) EventName() string {
	return model.EntityCRUDEventName
}

package event

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
)

// RegisterEventListeners registers all the event listeners
func RegisterEventListeners(dbAPI api.ObjectModelAPI) {
	base.Publisher.Subscribe(&EdgeConnectionEventListener{dbAPI: dbAPI})
	base.Publisher.Subscribe(&UpgradeEventListener{dbAPI: dbAPI})
	base.Publisher.Subscribe(&NodeInfoEventListener{dbAPI: dbAPI})
	base.Publisher.Subscribe(&EntityCRUDEventListener{dbAPI: dbAPI})
}

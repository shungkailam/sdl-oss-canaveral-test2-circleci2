package websocket_test

import (
	"cloudservices/cloudmgmt/apitesthelper"
)

func init() {
	apitesthelper.StartServices(&apitesthelper.StartServicesConfig{StartPort: 9020})
}

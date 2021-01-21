package websocket

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/graarh/golang-socketio"
)

// InitEdge initializes support log related websocket handlers
func (msgSvc *wsMessagingServiceImpl) InitLog() {
	// setup logUploadComplete message handler
	msgSvc.server.On("logUploadComplete", func(c *gosocketio.Channel, msg api.ObjectRequest) interface{} {
		payload := model.LogUploadCompletePayload{}
		err := base.Convert(&msg.Doc, &payload)
		if err != nil {
			return 400
		}
		// https://bucket.s3.us-west-2.amazonaws.com/v1/tenantId/2018/04/18/batchId/edge/edge-batchId.zip?AWSAccessKeyId=...
		tokens := strings.Split(payload.URL, "/")
		if len(tokens) < 11 {
			glog.Errorf("Invalid URL %s", payload.URL)
			return 400
		}
		tenantID := tokens[4]
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		msgSvc.dbAPI.UploadLogComplete(ctx, payload)
		if err != nil {
			return 500
		}
		glog.Infof("Completed logUploadComplete successfully for %s", payload.URL)
		return 200
	})
}

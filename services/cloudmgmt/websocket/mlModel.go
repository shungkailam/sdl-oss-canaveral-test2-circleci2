package websocket

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/base64"
	"encoding/json"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/graarh/golang-socketio"
)

// see: https://phx-it-github-prod-1.eng.nutanix.com/edgecomputing/sherlock-edgemgmt/pull/186

// ReportMLModelStatusRequest describes structure for mlmodel-status payload
// swagger:model ReportMLModelStatusRequest
type ReportMLModelStatusRequest struct {
	// required: true
	TenantID string `json:"tenant_id"`
	// required: true
	EdgeID string `json:"edge_id"`
	// required: true
	EdgeName string `json:"edge_name"`
	// required: true
	ID string `json:"id"`
	// required: true
	ModelName string
	// required: true
	Status []model.MLModelVersionStatus
}

// ReportMLModelStatusResponse describes mlmodel-status
// websocket message response
// swagger:model ReportMLModelStatusResponse
type ReportMLModelStatusResponse struct {
	// required: true
	StatusCode int `json:"statusCode"`
}

func handleMLModelStatusRequest(
	msgSvc *wsMessagingServiceImpl, msg ReportMLModelStatusRequest,
) ReportMLModelStatusResponse {
	tenantID := msg.TenantID
	doc := model.MLModelStatus{
		TenantID: tenantID,
		EdgeID:   msg.EdgeID,
		ModelID:  msg.ID,
		Status:   msg.Status,
	}
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx :=
		context.WithValue(context.Background(), base.AuthContextKey, authContext)
	_, err := msgSvc.dbAPI.CreateMLModelStatus(ctx, &doc, nil)
	statusCode := 200
	if err != nil {
		glog.Errorf("mlmodel-status: handleMLModelStatusRequest: "+
			"CreateMLModelStatus for msg: %+v, failed: %s\n", msg, err.Error())
		statusCode = 500
	}
	return ReportMLModelStatusResponse{
		StatusCode: statusCode,
	}
}

// InitMLModel initializes MLModel related websocket handlers
func (msgSvc *wsMessagingServiceImpl) InitMLModel() {
	// setup application-status message handler
	msgSvc.server.On("mlmodel-status",
		func(c *gosocketio.Channel, arg interface{}) interface{} {
			msg, ok := arg.(ReportMLModelStatusRequest)
			if !ok {
				s, ok := arg.(string)
				if !ok {
					glog.Errorf("mlmodel-status: expected ReportMLModelStatusRequest "+
						"or string, arg=%+v\n", arg)
					return ReportMLModelStatusResponse{
						StatusCode: 400,
					}
				}
				// decode base64 string then unmarshal to ReportMLModelStatusRequest
				ba, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					glog.Errorf("mlmodel-status: decode failed: %s, arg=%+v\n",
						arg, err.Error())
					return ReportMLModelStatusResponse{
						StatusCode: 400,
					}
				}
				msg2 := ReportMLModelStatusRequest{}
				err = json.Unmarshal(ba, &msg2)
				if err != nil {
					glog.Errorf("mlmodel-status: json unmarshal failed: %s, arg=%+v\n",
						arg, err.Error())
					return ReportMLModelStatusResponse{
						StatusCode: 400,
					}
				}
				msg = msg2
			}
			glog.Infof("Received mlmodel-status: tenantID=%s, edgeID=%s, appID=%s",
				msg.TenantID, msg.EdgeID, msg.ID)
			return handleMLModelStatusRequest(msgSvc, msg)
		})
}

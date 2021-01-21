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

// ReportAppStatusRequest describes structure for application-status payload
// swagger:model ReportAppStatusRequest
type ReportAppStatusRequest struct {
	// required: true
	TenantID string `json:"tenant_id"`
	// required: true
	EdgeID string `json:"edge_id"`
	// required: true
	ID string `json:"id"`
	// required: true
	Pods []model.PodStatus `json:"pods"`
	// required: false
	PodMetricses []model.PodMetrics `json:"podMetricses"`
}

// ReportAppStatusResponse describes application-status websocket message response
// swagger:model ReportAppStatusResponse
type ReportAppStatusResponse struct {
	// required: true
	StatusCode int `json:"statusCode"`
}

func handleAppStatusRequest(msgSvc *wsMessagingServiceImpl, msg ReportAppStatusRequest) ReportAppStatusResponse {
	tenantID := msg.TenantID
	doc := model.ApplicationStatus{
		TenantID:      tenantID,
		EdgeID:        msg.EdgeID,
		ApplicationID: msg.ID,
		AppStatus: model.AppStatus{
			PodStatusList:  msg.Pods,
			PodMetricsList: msg.PodMetricses,
		},
	}
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "edge",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	_, err := msgSvc.dbAPI.CreateApplicationStatus(ctx, &doc, nil)
	statusCode := 200
	if err != nil {
		glog.Errorf("application-status: handleAppStatusRequest: CreateApplicationStatus failed: %s\n", err.Error())
		statusCode = 500
	}
	return ReportAppStatusResponse{
		StatusCode: statusCode,
	}
}

// InitApplication initializes application related websocket handlers
func (msgSvc *wsMessagingServiceImpl) InitApplication() {
	// setup application-status message handler
	msgSvc.server.On("application-status", func(c *gosocketio.Channel, arg interface{}) interface{} {
		msg, ok := arg.(ReportAppStatusRequest)
		if !ok {
			s, ok := arg.(string)
			if !ok {
				glog.Errorf("application-status: expected ReportAppStatusRequest or string, arg=%+v\n", arg)
				return ReportAppStatusResponse{
					StatusCode: 400,
				}
			}
			// decode base64 string then unmarshal to ReportAppStatusRequest
			ba, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				glog.Errorf("application-status: decode failed: %s, arg=%+v\n", arg, err.Error())
				return ReportAppStatusResponse{
					StatusCode: 400,
				}
			}
			msg2 := ReportAppStatusRequest{}
			err = json.Unmarshal(ba, &msg2)
			if err != nil {
				glog.Errorf("application-status: json unmarshal failed: %s, arg=%+v\n", arg, err.Error())
				return ReportAppStatusResponse{
					StatusCode: 400,
				}
			}
			msg = msg2
		}
		glog.Infof("Received application-status: tenantID=%s, edgeID=%s, appID=%s", msg.TenantID, msg.EdgeID, msg.ID)
		return handleAppStatusRequest(msgSvc, msg)
	})
}

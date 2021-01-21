package websocket

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/graarh/golang-socketio"
)

// ReportSensorRequest describes structure for reportSensors payload
// swagger:model ReportSensorRequest
type ReportSensorRequest struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Sensors []model.Sensor `json:"sensors"`
}

// ReportSensorsResponse describes reportSensors websocket message response
// swagger:model ReportSensorsResponse
type ReportSensorsResponse struct {
	// required: true
	StatusCode int `json:"statusCode"`
}

func makeReportSensorsResponse(statusCode int) ReportSensorsResponse {
	return ReportSensorsResponse{
		StatusCode: statusCode,
	}
}

// InitSensor initializes sensor related websocket handlers
func (msgSvc *wsMessagingServiceImpl) InitSensor() {
	// setup reportSensors message handler
	msgSvc.server.On("reportSensors", func(c *gosocketio.Channel, msg ReportSensorRequest) interface{} {
		if len(msg.Sensors) == 0 {
			return makeReportSensorsResponse(200)
		}
		authContext := &base.AuthContext{
			TenantID: msg.TenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		for _, sensor := range msg.Sensors {
			_, err := msgSvc.dbAPI.CreateSensor(ctx, &sensor, nil)
			if err != nil {
				return makeReportSensorsResponse(500)
			}
		}
		return makeReportSensorsResponse(201)
	})
}

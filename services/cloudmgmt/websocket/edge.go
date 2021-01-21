package websocket

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	gosocketio "github.com/graarh/golang-socketio"
)

// ReportEdgeResponse describes reportEdge websocket message response
// swagger:model ReportEdgeResponse
type ReportEdgeResponse struct {
	// required: true
	StatusCode int `json:"statusCode"`
	// required: true
	Doc *model.Edge `json:"doc"`
}

// ReportEdgeClusterResponse describes reportEdgeCluster websocket message response
// swagger:model ReportEdgeClusterResponse
type ReportEdgeClusterResponse struct {
	// required: true
	StatusCode int `json:"statusCode"`
	// required: true
	Doc *model.EdgeCluster `json:"doc"`
}

// ReportEdgeInfoResponse describes reportEdgeInfo websocket message response
// swagger:model ReportEdgeInfoResponse
type ReportEdgeInfoResponse struct {
	// required: true
	StatusCode int `json:"statusCode"`
}

// Whether to automatically create the edge if the reported edge is not found in DB
// Set to false, since we no longer support auto-create edge and now require explicit edge onboarding via POST to /v1/edges API
const createEdgeOnReportEdgeNotFound bool = false

func makeReportEdgeErrorResponse(statusCode int, msg string) ReportEdgeResponse {
	glog.Warningf("reportEdge[%d]: %s\n", statusCode, msg)
	return ReportEdgeResponse{
		StatusCode: statusCode,
		Doc:        nil,
	}
}

// InitEdge initializes edge related websocket handlers
func (msgSvc *wsMessagingServiceImpl) InitEdge() {
	// setup reportEdge message handler
	msgSvc.server.On("reportEdge", func(c *gosocketio.Channel, msg api.ObjectRequest) interface{} {
		edge := model.Edge{}
		edgeInfo := model.EdgeUsageInfo{}
		err := base.Convert(&msg.Doc, &edge)
		if err != nil {
			return makeReportEdgeErrorResponse(400, "bad input")
		}
		// avoid empty id error
		if edge.ID == "" {
			edge.ID = base.GetUUID()
		}
		authContext := &base.AuthContext{
			TenantID: edge.TenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		edge2, err := msgSvc.dbAPI.GetEdgeCluster(ctx, edge.ID)
		if err != nil {
			errc, ok := err.(errcode.ErrorCode)
			if ok && errc.GetCode() == errcode.RecordNotFound {
				if createEdgeOnReportEdgeNotFound {
					r, err := msgSvc.dbAPI.CreateEdge(ctx, &edge, nil)
					if err != nil {
						c.Close()
						return makeReportEdgeErrorResponse(400, err.Error())
					}
					edge.ID = r.(model.CreateDocumentResponse).ID
					edge2, err = msgSvc.dbAPI.GetEdgeCluster(ctx, edge.ID)
					if err != nil {
						c.Close()
						return makeReportEdgeErrorResponse(500, err.Error())
					}
					_, err = msgSvc.dbAPI.CreateEdgeInfo(ctx, &edgeInfo, nil)
					if err != nil {
						c.Close()
						return makeReportEdgeErrorResponse(500, err.Error())
					}
				} else {
					c.Close()
					glog.Warningf("reportEdge: edge %s not found\n", edge.ID)
					return makeReportEdgeErrorResponse(400, err.Error())
				}

			} else {
				c.Close()
				return makeReportEdgeErrorResponse(500, err.Error())
			}
		}
		err = c.Join(edge.TenantID)
		if err != nil {
			c.Close()
			return makeReportEdgeErrorResponse(500, err.Error())
		}
		msgSvc.SetChannel(edge.TenantID, edge.ID, c)
		glog.Infof("reportEdge[200]: %+v\n", edge2)
		return ReportEdgeClusterResponse{
			StatusCode: 200,
			Doc:        &edge2,
		}
	})

	// setup reportEdgeInfo message handler
	msgSvc.server.On("reportEdgeInfo", func(c *gosocketio.Channel, msg api.ObjectRequest) interface{} {
		edgeInfo := model.EdgeUsageInfo{}
		err := base.Convert(&msg.Doc, &edgeInfo)
		if err != nil {
			glog.Warningf("Could not convert the recieved info %s", err)
			return makeReportEdgeErrorResponse(400, err.Error())
		}
		authContext := &base.AuthContext{
			TenantID: edgeInfo.TenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		_, err = msgSvc.dbAPI.UpdateEdgeInfo(ctx, &edgeInfo, nil)
		if err != nil {
			glog.Warningf("Could not update edge %s", err)
			return makeReportEdgeErrorResponse(500, err.Error())
		}
		return ReportEdgeInfoResponse{
			StatusCode: 200,
		}
	})
}

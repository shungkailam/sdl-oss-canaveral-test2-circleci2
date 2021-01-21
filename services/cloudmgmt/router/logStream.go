package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
	"reflect"

	"github.com/golang/glog"
)

func getLogStreamRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	LogStreamEndpointsHandle := makeCustomMessageHandle(dbAPI, dbAPI.RequestLogStreamEndpointsW, msgSvc, "logStream", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload, ok := doc.(*model.WSMessagingLogStream)
		if !ok {
			glog.Infof("Got type: %s", reflect.TypeOf(doc))
			return nil
		}
		return &payload.LogStreamInfo.EdgeID
	})
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/logs/stream/endpoints",
			// swagger:route POST /v1.0/logs/stream/endpoints Log LogStreamEndpoints
			//
			// Get the endpoints to stream logs for a given container from an edge.
			//
			// Get the endpoints to stream logs for a given container from an edge.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//	   Responses:
			//		 200: LogStreamEndpointsResponse
			//		 default: APIError
			handle: LogStreamEndpointsHandle,
		},
	}
}

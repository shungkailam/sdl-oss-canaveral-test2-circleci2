package router

import (
	"cloudservices/cloudmgmt/api"
)

func getServiceRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/services",
			// swagger:route GET /v1.0/services Service ServiceList
			//
			// Get services. ntnx:ignore
			//
			// Retrieves service information.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceListResponse
			//       default: APIError
			handle: makeGetAllHandleNoAuth(dbAPI, dbAPI.GetServicesW, "/services"),
		},
		{
			method: "GET",
			path:   "/v1.0/services/",
			handle: makeGetAllHandleNoAuth(dbAPI, dbAPI.GetServicesW, "/services"),
		},
	}
}

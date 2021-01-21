package router

import (
	"cloudservices/cloudmgmt/api"
)

func getServiceBindingRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/servicebindings",
			// swagger:route POST /v1.0/servicebindings Service_Binding ServiceBindingCreate
			//
			// Create a Service Binding.
			//
			// Create a Service Binding
			//
			//     Consumes:
			//	   - application/json
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CreateDocumentResponseV2
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateServiceBindingW, msgSvc, "servicebinding", NOTIFICATION_TENANT),
		},
		{
			method: "GET",
			path:   "/v1.0/servicebindings",
			// swagger:route GET /v1.0/servicebindings Service_Binding ServiceBindingList
			//
			// List Service Bindings.
			//
			// List Service Bindings
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceBindingListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceBindingsW, "/servicebindings"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicebindings/:svcBindingId",
			// swagger:route GET /v1.0/servicebindings/{svcBindingId} Service_Binding ServiceBindingGet
			//
			// Get a Service Binding.
			//
			// Get a Service Binding
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceBindingGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetServiceBindingW, "/servicebindings/:svcBindingId", "svcBindingId"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/servicebindings/:svcBindingId",
			// swagger:route DELETE /v1.0/servicebindings/{svcBindingId} Service_Binding ServiceBindingDelete
			//
			// Delete a Service Binding.
			//
			// Delete a Service Binding
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponseV2
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteServiceBindingW, msgSvc, "servicebinding", NOTIFICATION_TENANT, "svcBindingId"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicebindings/:svcBindingId/status",
			// swagger:route GET /v1.0/servicebindings/{svcBindingId}/status Service_Binding ServiceBindingStatusList
			//
			// Get the status of Service Binding.
			//
			// Get the status of Service Binding on Service Domains
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceBindingStatusListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectServiceBindingStatussW, "/servicebindings/:svcBindingId/status", "svcBindingId"),
		},
	}
}

package router

import (
	"cloudservices/cloudmgmt/api"
)

func getServiceInstanceRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/serviceinstances",
			// swagger:route POST /v1.0/serviceinstances Service_Instance ServiceInstanceCreate
			//
			// Create a Service Instance.
			//
			// Create a Service Instance
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateServiceInstanceW, msgSvc, "serviceinstance", NOTIFICATION_TENANT),
		},
		{
			method: "GET",
			path:   "/v1.0/serviceinstances",
			// swagger:route GET /v1.0/serviceinstances Service_Instance ServiceInstanceList
			//
			// List Service Instances.
			//
			// List Service Instances
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceInstanceListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceInstancesW, "/serviceinstances"),
		},
		{
			method: "PUT",
			path:   "/v1.0/serviceinstances/:svcInstanceId",
			// swagger:route PUT /v1.0/serviceinstances/{svcInstanceId} Service_Instance ServiceInstanceUpdate
			//
			// Update a Service Instance.
			//
			// Update a Service Instance
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateServiceInstanceW, msgSvc, "serviceinstance", NOTIFICATION_TENANT, "svcInstanceId"),
		},
		{
			method: "GET",
			path:   "/v1.0/serviceinstances/:svcInstanceId",
			// swagger:route GET /v1.0/serviceinstances/{svcInstanceId} Service_Instance ServiceInstanceGet
			//
			// Get a Service Instance.
			//
			// Get a Service Instance
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceInstanceGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetServiceInstanceW, "/serviceinstances/:svcInstanceId", "svcInstanceId"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/serviceinstances/:svcInstanceId",
			// swagger:route DELETE /v1.0/serviceinstances/{svcInstanceId} Service_Instance ServiceInstanceDelete
			//
			// Delete a Service Instance.
			//
			// Delete a Service Instance
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteServiceInstanceW, msgSvc, "serviceinstance", NOTIFICATION_TENANT, "svcInstanceId"),
		},
		{
			method: "GET",
			path:   "/v1.0/serviceinstances/:svcInstanceId/status",
			// swagger:route GET /v1.0/serviceinstances/{svcInstanceId}/status Service_Instance ServiceInstanceStatusList
			//
			// Get the status of the Service Instance.
			//
			// Get the status of the Service Instance on Service Domains
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceInstanceStatusListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectServiceInstanceStatussW, "/serviceinstances/:svcInstanceId/status", "svcInstanceId"),
		},
	}
}

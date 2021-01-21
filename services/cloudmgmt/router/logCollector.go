package router

import (
	"cloudservices/cloudmgmt/api"
)

func getLogCollectorRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/logs/collector",
			// swagger:route GET /v1.0/logs/collector LogCollector LogCollectorsList
			//
			// Get configured log collectors in a system.
			//
			// Get log collectors information.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//	   Responses:
			//		 200: LogCollectorListResponse
			//		 default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllLogCollectorsW, "/log/collector"),
		},
		{
			method: "POST",
			path:   "/v1.0/logs/collector",
			// swagger:route POST /v1.0/logs/collector LogCollector LogCollectorCreate
			//
			// Create a log collector.
			//
			// Create a log collector.
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
			//		 200: CreateDocumentResponse
			//		 default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateLogCollectorW, msgSvc, "logcollector", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1.0/logs/collector/:id",
			// swagger:route PUT /v1.0/logs/collector/{id} LogCollector LogCollectorUpdate
			//
			// Update a log collector.
			//
			// Update a log collector by ID {id}.
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
			//		 200: UpdateDocumentResponse
			//		 default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateLogCollectorW, msgSvc, "logcollector", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/logs/collector/:id",
			// swagger:route DELETE /v1.0/logs/collector/{id} LogCollector LogCollectorDelete
			//
			// Delete a log collector.
			//
			// Delete a log collector by ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//	   Responses:
			//		 200: DeleteDocumentResponse
			//		 default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteLogCollectorW, msgSvc, "logcollector", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/logs/collector/:id/start",
			// swagger:route POST /v1.0/logs/collector/{id}/start LogCollector LogCollectorStart
			//
			// Start a log collector.
			//
			// Start a log collector by ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//	   Responses:
			//		 200: UpdateDocumentResponse
			//		 default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.StartLogCollectorW, msgSvc, "logcollector", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/logs/collector/:id/stop",
			// swagger:route POST /v1.0/logs/collector/{id}/stop LogCollector LogCollectorStop
			//
			// Stop a log collector.
			//
			// Stop a log collector by ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//	   Responses:
			//		 200: UpdateDocumentResponse
			//		 default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.StopLogCollectorW, msgSvc, "logcollector", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/logs/collector/:id",
			// swagger:route GET /v1.0/logs/collector/{id} LogCollector LogCollectorGet
			//
			// Get information about log collector
			//
			// Get log collector information by ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//	   Responses:
			//		 200: LogCollectorResponse
			//		 default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetLogCollectorW, "/v1.0/logs/collector/:id", "id"),
		},
	}
}

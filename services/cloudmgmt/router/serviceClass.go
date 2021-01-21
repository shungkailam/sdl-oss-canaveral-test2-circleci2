package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/auth"
)

func getServiceClassRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/serviceclasses",
			// swagger:route POST /v1.0/serviceclasses Service_Class ServiceClassCreate
			//
			// Create a Service Class. ntnx:ignore
			//
			// Create a Service Class
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateServiceClassW, msgSvc, "serviceclass", NOTIFICATION_NONE),

			tenantIDs: []string{auth.OperatorTenantID},

			roles: []string{auth.OperatorRole},
		},
		{
			method: "GET",
			path:   "/v1.0/serviceclasses",
			// swagger:route GET /v1.0/serviceclasses Service_Class ServiceClassList
			//
			// List Service Classes.
			//
			// List Service Classes
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceClassListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceClassesW, "/serviceclasses"),
		},
		{
			method: "PUT",
			path:   "/v1.0/serviceclasses/:svcClassId",
			// swagger:route PUT /v1.0/serviceclasses/{svcClassId} Service_Class ServiceClassUpdate
			//
			// Update a Service Class. ntnx:ignore
			//
			// Update a Service Class
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateServiceClassW, msgSvc, "serviceclass", NOTIFICATION_NONE, "svcClassId"),

			tenantIDs: []string{auth.OperatorTenantID},

			roles: []string{auth.OperatorRole},
		},
		{
			method: "GET",
			path:   "/v1.0/serviceclasses/:svcClassId",
			// swagger:route GET /v1.0/serviceclasses/{svcClassId} Service_Class ServiceClassGet
			//
			// Get a Service Class.
			//
			// Get a Service Class
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceClassGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetServiceClassW, "/serviceclasses/:svcClassId", "svcClassId"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/serviceclasses/:svcClassId",
			// swagger:route DELETE /v1.0/serviceclasses/{svcClassId} Service_Class ServiceClassDelete
			//
			// Delete a Service Class. ntnx:ignore
			//
			// Delete a Service Class
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteServiceClassW, msgSvc, "serviceclass", NOTIFICATION_NONE, "svcClassId"),

			tenantIDs: []string{auth.OperatorTenantID},

			roles: []string{auth.OperatorRole},
		},
	}
}

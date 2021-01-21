package router

import (
	"cloudservices/cloudmgmt/api"
)

func getApplicationStatusRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/applicationstatus",
			// swagger:route GET /v1/applicationstatus ApplicationStatusList
			//
			// Get applications status. ntnx:ignore
			//
			// Retrieves status for all applications.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ApplicationStatusListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsStatusW, "/applicationstatus"),
		},
		{
			method: "GET",
			path:   "/v1.0/applicationstatuses",
			// swagger:route GET /v1.0/applicationstatuses Application_Status ApplicationStatusListV2
			//
			// Get status for all applications.
			//
			// Retrieves status for all applications.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ApplicationStatusListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsStatusWV2, "/applicationstatus"),
		},
		{
			method: "GET",
			path:   "/v1.0/applicationstatuses/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsStatusWV2, "/applicationstatus"),
		},
		{
			method: "GET",
			path:   "/v1/applicationstatus/:id",
			// swagger:route GET /v1/applicationstatus/{id} ApplicationStatusGet
			//
			// Get application status by application ID. ntnx:ignore
			//
			// Retrieve status for an application with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ApplicationStatusListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetApplicationStatusW, "/applicationstatus/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/applicationstatuses/:id",
			// swagger:route GET /v1.0/applicationstatuses/{id} Application_Status ApplicationStatusGetV2
			//
			// Get application status by application ID.
			//
			// Retrieve status for an application with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ApplicationStatusListResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetApplicationStatusWV2, "/applicationstatus/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/applicationstatus/:id",
			// swagger:route DELETE /v1/applicationstatus/{id} ApplicationStatusDelete
			//
			// Delete an application by application ID. ntnx:ignore
			//
			// Deletes the application with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponse
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteApplicationStatusW, msgSvc, "applicationstatus", NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/applicationstatuses/:id",
			// swagger:route DELETE /v1.0/applicationstatuses/{id} Application_Status ApplicationStatusDeleteV2
			//
			// Delete an application by application ID.  ntnx:ignore
			//
			// Deletes the application with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponseV2
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteApplicationStatusWV2, msgSvc, "applicationstatus", NOTIFICATION_NONE, "id"),
		},
		{
			method: "POST",
			path:   "/v1/applicationstatus",
			// swagger:route POST /v1/applicationstatus ApplicationStatusCreate
			//
			// Create application status. ntnx:ignore
			//
			// Creates an application status.
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
			//     Responses:
			//       200: CreateDocumentResponse
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateApplicationStatusW, msgSvc, "applicationstatus", NOTIFICATION_NONE),
		},
		{
			method: "POST",
			path:   "/v1.0/applicationstatuses",
			// swagger:route POST /v1.0/applicationstatuses Application_Status ApplicationStatusCreateV2
			//
			// Create application status.  ntnx:ignore
			//
			// Creates an application status.
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
			//     Responses:
			//       200: CreateDocumentResponseV2
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateApplicationStatusWV2, msgSvc, "applicationstatus", NOTIFICATION_NONE),
		},
	}
}

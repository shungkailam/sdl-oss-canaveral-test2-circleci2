package router

import (
	"cloudservices/cloudmgmt/api"
)

func getProjectServiceRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/projectservices",
			// swagger:route GET /v1.0/projectservices Project_Service ProjectServiceList
			//
			// Get all project services. ntnx:ignore
			//
			// Retrieves a list of all applications.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ProjectServiceListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllProjectServicesW, "/projectservices"),
		},
		{
			method: "GET",
			path:   "/v1.0/projectservices/:id",
			// swagger:route GET /v1.0/projectservices/{id} Project_Service ProjectServiceGet
			//
			// Get project service by ID. ntnx:ignore
			//
			// Retrieves the project service according to its ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ProjectServiceGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetProjectServiceW, "/projectservice/:id", "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/projectservices",
			// swagger:route POST /v1.0/projectservices Project_Service ProjectServiceCreate
			//
			// Create a project service. ntnx:ignore
			//
			// Create a project service.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateProjectServiceW, msgSvc, "projectservice", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1.0/projectservices/:id",
			// swagger:route PUT /v1.0/projectservices/{id} Project_Service ProjectServiceUpdate
			//
			// Update a specific project service with ID {id}. ntnx:ignore
			//
			// Update a specific project service with ID {id}.
			// You cannot change the project associated with the project service or the project service ID.
			// You can change all other attributes.
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
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateProjectServiceW, msgSvc, "projectservice", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/projectservices/:id",
			// swagger:route DELETE /v1.0/projectservices/{id} Project_Service ProjectServiceDelete
			//
			// Delete project service specified by the project service ID. ntnx:ignore
			//
			// Deletes the project service with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteProjectServiceW, msgSvc, "projectservice", NOTIFICATION_TENANT, "id"),
		},
	}
}

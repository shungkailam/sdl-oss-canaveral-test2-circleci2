package router

import (
	"cloudservices/cloudmgmt/api"
)

func getProjectRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/projects",
			// swagger:route GET /v1/projects ProjectList
			//
			// Get projects. ntnx:ignore
			//
			// Retrieves all projects.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ProjectListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllProjectsW, "/projects"),
		},
		{
			method: "GET",
			path:   "/v1/projects/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllProjectsW, "/projects"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects",
			// swagger:route GET /v1.0/projects Project ProjectListV2
			//
			// Get projects.
			//
			// Retrieves all projects.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ProjectListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllProjectsWV2, "/projects"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllProjectsWV2, "/projects"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId",
			// swagger:route GET /v1/projects/{projectId} ProjectGet
			//
			// Get project by its ID. ntnx:ignore
			//
			// Retrieves the project by its given ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ProjectGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetProjectW, "/projects/:projectId", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId",
			// swagger:route GET /v1.0/projects/{projectId} Project ProjectGetV2
			//
			// Get project by its ID.
			//
			// Retrieves the project by its given ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ProjectGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetProjectW, "/projects/:projectId", "projectId"),
		},
		{
			method: "DELETE",
			path:   "/v1/projects/:id",
			// swagger:route DELETE /v1/projects/{id} ProjectDelete
			//
			// Delete a project by ID. ntnx:ignore
			//
			// Deletes a project with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteProjectW, msgSvc, "project", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/projects/:id",
			// swagger:route DELETE /v1.0/projects/{id} Project ProjectDeleteV2
			//
			// Delete a project by ID.
			//
			// Deletes a project with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteProjectWV2, msgSvc, "project", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/projects",
			// swagger:route POST /v1/projects ProjectCreate
			//
			// Create a project. ntnx:ignore
			//
			// Creates a project.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateProjectW, msgSvc, "project", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/projects",
			// swagger:route POST /v1.0/projects Project ProjectCreateV2
			//
			// Create a project.
			//
			// Creates a project.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateProjectWV2, msgSvc, "project", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/projects",
			// swagger:route PUT /v1/projects ProjectUpdate
			//
			// Update projects. ntnx:ignore
			//
			// Updates projects.
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
			//       200: UpdateDocumentResponse
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateProjectW, msgSvc, "project", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/projects/:id",
			// swagger:route PUT /v1/projects/{id} ProjectUpdateV2
			//
			// Update a project by its ID. ntnx:ignore
			//
			// Updates a project by its given ID {id}.
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
			//       200: UpdateDocumentResponse
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateProjectW, msgSvc, "project", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/projects/:id",
			// swagger:route PUT /v1.0/projects/{id} Project ProjectUpdateV3
			//
			// Update a project by its ID.
			//
			// Updates a project by its given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateProjectWV2, msgSvc, "project", NOTIFICATION_TENANT, "id"),
		},
	}
}

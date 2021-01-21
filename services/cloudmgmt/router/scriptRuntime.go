package router

import (
	"cloudservices/cloudmgmt/api"
)

func getScriptRuntimeRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/scriptruntimes",
			// swagger:route GET /v1/scriptruntimes ScriptRuntimeList
			//
			// Gets a function runtime environments. ntnx:ignore
			//
			// Retrieves all function runtime environments.
			//
			// Retrieves all script runtimes.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ScriptRuntimeListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptRuntimesW, "/scriptruntimes"),
		},
		{
			method: "GET",
			path:   "/v1/scriptruntimes/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptRuntimesW, "/scriptruntimes"),
		},
		{
			method: "GET",
			path:   "/v1.0/runtimeenvironments",
			// swagger:route GET /v1.0/runtimeenvironments Runtime_Environment RuntimeEnvironmentList
			//
			// Get runtime environments.
			//
			// Retrieves all runtime environments.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: RuntimeEnvironmentListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptRuntimesWV2, "/scriptruntimes"),
		},
		{
			method: "GET",
			path:   "/v1.0/runtimeenvironments/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptRuntimesWV2, "/scriptruntimes"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/scriptruntimes",
			// swagger:route GET /v1/projects/{projectId}/scriptruntimes ProjectGetScriptRuntimes
			//
			// Gets function runtime environments for a project by ID. ntnx:ignore
			//
			// Retrieves all function runtime environments for a project by a given project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ScriptRuntimeListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllScriptRuntimesForProjectW, "/project-scriptruntimes", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/runtimeenvironments",
			// swagger:route GET /v1.0/projects/{projectId}/runtimeenvironments Runtime_Environment ProjectGetRuntimeEnvironments
			//
			// Gets runtime environments for a project by project ID.
			//
			// Retrieves all runtime environments for a project by a given project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: RuntimeEnvironmentListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllScriptRuntimesForProjectWV2, "/project-scriptruntimes", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/scriptruntimes/:id",
			// swagger:route GET /v1/scriptruntimes/{id} ScriptRuntimeGet
			//
			// Get a function for a runtime by function ID. ntnx:ignore
			//
			// Retrieves the function for a runtime by a given function ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ScriptRuntimeGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetScriptRuntimeW, "/scriptruntimes/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/runtimeenvironments/:id",
			// swagger:route GET /v1.0/runtimeenvironments/{id} Runtime_Environment RuntimeEnvironmentGet
			//
			// Get a runtime environment by its ID.
			//
			// Retrieves a runtime environment with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: RuntimeEnvironmentGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetScriptRuntimeW, "/scriptruntimes/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/scriptruntimes/:id",
			// swagger:route DELETE /v1/scriptruntimes/{id} ScriptRuntimeDelete
			//
			// Delete script runtime. ntnx:ignore
			//
			// Delete runtime environment according to its given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteScriptRuntimeW, msgSvc, "scriptruntime", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/runtimeenvironments/:id",
			// swagger:route DELETE /v1.0/runtimeenvironments/{id} Runtime_Environment RuntimeEnvironmentDelete
			//
			// Delete a runtime environment by its ID.
			//
			// Delete runtime environment according to its given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteScriptRuntimeWV2, msgSvc, "scriptruntime", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/scriptruntimes",
			// swagger:route POST /v1/scriptruntimes ScriptRuntimeCreate
			//
			// Create script runtime. ntnx:ignore
			//
			// Create a script runtime.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateScriptRuntimeW, msgSvc, "scriptruntime", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/runtimeenvironments",
			// swagger:route POST /v1.0/runtimeenvironments Runtime_Environment RuntimeEnvironmentCreate
			//
			// Create a runtime environment for functions.
			//
			// Creates a runtime environment for functions.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateScriptRuntimeWV2, msgSvc, "scriptruntime", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/scriptruntimes",
			// swagger:route PUT /v1/scriptruntimes ScriptRuntimeUpdate
			//
			// Update script runtime. ntnx:ignore
			//
			// Update a script runtime.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateScriptRuntimeW, msgSvc, "scriptruntime", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/scriptruntimes/:id",
			// swagger:route PUT /v1/scriptruntimes/{id} ScriptRuntimeUpdateV2
			//
			// Update script runtime. ntnx:ignore
			//
			// Update a script runtime.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateScriptRuntimeW, msgSvc, "scriptruntime", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/runtimeenvironments/:id",
			// swagger:route PUT /v1.0/runtimeenvironments/{id} Runtime_Environment RuntimeEnvironmentUpdate
			//
			// Update the runtime environment by its ID.
			//
			// Updates a function runtime environment by its given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateScriptRuntimeWV2, msgSvc, "scriptruntime", NOTIFICATION_TENANT, "id"),
		},
	}
}

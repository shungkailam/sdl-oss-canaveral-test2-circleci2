package router

import (
	"cloudservices/cloudmgmt/api"
)

func getScriptRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/scripts",
			// swagger:route GET /v1/scripts ScriptList
			//
			// Get scripts. ntnx:ignore
			//
			// This will retrieve all scripts for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ScriptListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptsW, "/scripts"),
		},
		{
			method: "GET",
			path:   "/v1/scripts/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptsW, "/scripts"),
		},
		{
			method: "GET",
			path:   "/v1.0/functions",
			// swagger:route GET /v1.0/functions Function FunctionList
			//
			// Get functions.
			//
			// Retrieves all functions.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: FunctionListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptsWV2, "/scripts"),
		},
		{
			method: "GET",
			path:   "/v1.0/functions/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllScriptsWV2, "/scripts"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/scripts",
			// swagger:route GET /v1/projects/{projectId}/scripts ProjectGetScripts
			//
			// Get project scripts. ntnx:ignore
			//
			// This will retrieve all scripts for a project.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ScriptListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllScriptsForProjectW, "/project-scripts", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/functions",
			// swagger:route GET /v1.0/projects/{projectId}/functions Function ProjectGetFunctions
			//
			// Get functions by project ID.
			//
			// Retrieves all functions according to a given project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: FunctionListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllScriptsForProjectWV2, "/project-scripts", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/scripts/:id",
			// swagger:route GET /v1/scripts/{id} ScriptGet
			//
			// Get script. ntnx:ignore
			//
			// This will get the script with the given id.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ScriptGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetScriptW, "/scripts/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/functions/:id",
			// swagger:route GET /v1.0/functions/{id} Function FunctionGet
			//
			// Get a function by its ID.
			//
			// Retrieves the function with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: FunctionGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetScriptW, "/scripts/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/scripts/:id",
			// swagger:route DELETE /v1/scripts/{id} ScriptDelete
			//
			// Delete script. ntnx:ignore
			//
			// This will delete the script with the given id.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteScriptW, msgSvc, "script", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/functions/:id",
			// swagger:route DELETE /v1.0/functions/{id} Function FunctionDelete
			//
			// Delete a function by its ID.
			//
			// Deletes the function with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteScriptWV2, msgSvc, "script", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/scripts",
			// swagger:route POST /v1/scripts ScriptCreate
			//
			// Create script. ntnx:ignore
			//
			// This will create a script.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateScriptW, msgSvc, "script", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/functions",
			// swagger:route POST /v1.0/functions Function FunctionCreate
			//
			// Create a function.
			//
			// Creates a function.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateScriptWV2, msgSvc, "script", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/scripts",
			// swagger:route PUT /v1/scripts ScriptUpdate
			//
			// Update script. ntnx:ignore
			//
			// This will update a script.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateScriptW, msgSvc, "script", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/scripts/:id",
			// swagger:route PUT /v1/scripts/{id} ScriptUpdateV2
			//
			// Update script. ntnx:ignore
			//
			// This will update a script.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateScriptW, msgSvc, "script", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/functions/:id",
			// swagger:route PUT /v1.0/functions/{id} Function FunctionUpdate
			//
			// Update function by its ID
			//
			// Updates a function with the given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateScriptWV2, msgSvc, "script", NOTIFICATION_TENANT, "id"),
		},
	}
}

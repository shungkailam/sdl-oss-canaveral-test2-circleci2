package router

import (
	"cloudservices/cloudmgmt/api"
)

func getUserRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/users",
			// swagger:route GET /v1/users UserList
			//
			// Get users. ntnx:ignore
			//
			// Retrieves all users for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: UserListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllUsersW, "/users"),
		},
		{
			method: "GET",
			path:   "/v1/users/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllUsersW, "/users"),
		},
		{
			method: "GET",
			path:   "/v1.0/users",
			// swagger:route GET /v1.0/users User UserListV2
			//
			// Get users. ntnx:ignore
			//
			// Retrieves all users for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: UserListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllUsersWV2, "/users"),
		},
		{
			method: "GET",
			path:   "/v1.0/users/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllUsersWV2, "/users"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/users",
			// swagger:route GET /v1/projects/{projectId}/users ProjectGetUsers
			//
			// Get project users by project ID. ntnx:ignore
			//
			// Retrieves all users for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllUsersForProjectW, "/project-users", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/users",
			// swagger:route GET /v1.0/projects/{projectId}/users User ProjectGetUsersV2
			//
			// Get project users. ntnx:ignore
			//
			// Retrievesall users for a project  by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllUsersForProjectWV2, "/project-users", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/users/:id",
			// swagger:route GET /v1/users/{id} UserGet
			//
			// Get user. ntnx:ignore
			//
			// Retrieves a user with the given id {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetUserW, "/users/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/users/:id",
			// swagger:route GET /v1.0/users/{id} User UserGetV2
			//
			// Get user by ID. ntnx:ignore
			//
			// Retrieves a user with the given id {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetUserW, "/users/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/users/:id",
			// swagger:route DELETE /v1/users/{id} UserDelete
			//
			// Delete user by ID. ntnx:ignore
			//
			// Deletes the user with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteUserW, msgSvc, "user", NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/users/:id",
			// swagger:route DELETE /v1.0/users/{id} User UserDeleteV2
			//
			// Delete user by ID. ntnx:ignore
			//
			// Deletes the user with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteUserWV2, msgSvc, "user", NOTIFICATION_NONE, "id"),
		},
		{
			method: "POST",
			path:   "/v1/users",
			// swagger:route POST /v1/users UserCreate
			//
			// Create user. ntnx:ignore
			//
			// Creates a user.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateUserW, msgSvc, "user", NOTIFICATION_NONE),
		},
		{
			method: "POST",
			path:   "/v1.0/users",
			// swagger:route POST /v1.0/users User UserCreateV2
			//
			// Create user. ntnx:ignore
			//
			// Creates a user.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateUserWV2, msgSvc, "user", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1/users",
			// swagger:route PUT /v1/users UserUpdate
			//
			// Update user. ntnx:ignore
			//
			// Updates a user.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserW, msgSvc, "user", NOTIFICATION_NONE, ""),
		},
		{
			method: "PUT",
			path:   "/v1/users/:id",
			// swagger:route PUT /v1/users/{id} UserUpdateV2
			//
			// Update user with a given ID. ntnx:ignore
			//
			// Updates a user with a given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserW, msgSvc, "user", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/users/:id",
			// swagger:route PUT /v1.0/users/{id} User UserUpdateV3
			//
			// Update user with a given ID. ntnx:ignore
			//
			// Updates a user with a given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserWV2, msgSvc, "user", NOTIFICATION_NONE, "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/isemailavailable",
			// swagger:route GET /v1.0/isemailavailable User IsEmailAvailable
			//
			// Check if the given email is available for create user. ntnx:ignore
			//
			// Checks if the given email is available for create user.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: IsEmailAvailableResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.IsEmailAvailableW, "/isemailavailable"),
		},
	}
}

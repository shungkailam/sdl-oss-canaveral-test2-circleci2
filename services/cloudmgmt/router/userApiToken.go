package router

import (
	"cloudservices/cloudmgmt/api"
)

func getUserApiTokenRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/userapitokensall",
			// swagger:route GET /v1.0/userapitokensall User_API_Token UserApiTokenList
			//
			// Get all user API tokens.
			//
			// Retrieves the API tokens info for all users. Must be infra admin for this to work.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserApiTokenListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllUserApiTokensW, "/userapitokens"),
		},
		{
			method: "GET",
			path:   "/v1.0/userapitokens",
			// swagger:route GET /v1.0/userapitokens User_API_Token UserApiTokenGet
			//
			// Get current user API tokens.
			//
			// Retrieves the API tokens info for the current user.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserApiTokenListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetUserApiTokensW, "/userapitokens", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/userapitokens/:id",
			// swagger:route DELETE /v1.0/userapitokens/{id} User_API_Token UserApiTokenDelete
			//
			// Delete current user API token.
			//
			// Deletes the API token with the given id for the current user.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteUserApiTokenW, msgSvc, "userapitokens", NOTIFICATION_NONE, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/userapitokens",
			// swagger:route POST /v1.0/userapitokens User_API_Token UserApiTokenCreate
			//
			// Create a user API token.
			//
			// Creates a user API token.
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
			//       200: UserApiTokenCreateResponse
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateUserApiTokenW, msgSvc, "userapitokens", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1.0/userapitokens/:id",
			// swagger:route PUT /v1.0/userapitokens/{id} User_API_Token UserApiTokenUpdate
			//
			// Update user API token.
			//
			// Update the API token with the given id. Must be current user or infra admin.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserApiTokenW, msgSvc, "userapitokens", NOTIFICATION_NONE, "id"),
		},
	}
}

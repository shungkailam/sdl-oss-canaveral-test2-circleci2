package router

import (
	"cloudservices/cloudmgmt/api"
)

func getUserPublicKeyRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/userpublickeyall",
			// swagger:route GET /v1.0/userpublickeyall User_Public_Key UserPublicKeyList
			//
			// Get all user public keys.
			//
			// Retrieves the public keys for all users.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserPublicKeyListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllUserPublicKeysW, "/userpublickeyall"),
		},
		{
			method: "GET",
			path:   "/v1.0/userpublickey",
			// swagger:route GET /v1.0/userpublickey User_Public_Key UserPublicKeyGet
			//
			// Get current user public key.
			//
			// Retrieves the public key for the current user.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserPublicKeyGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetUserPublicKeyW, "/userpublickey/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/userpublickey",
			// swagger:route DELETE /v1.0/userpublickey User_Public_Key UserPublicKeyDelete
			//
			// Delete current user public key.
			//
			// Deletes the public key for the current user.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteUserPublicKeyW, msgSvc, "userpublickey", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/userpublickey",
			// swagger:route PUT /v1.0/userpublickey User_Public_Key UserPublicKeyUpdate
			//
			// Upsert current user public key.
			//
			// Upserts the public key of the current user.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserPublicKeyW, msgSvc, "userpublickey", NOTIFICATION_NONE, "id"),
		},
	}
}

package router

import (
	"cloudservices/cloudmgmt/api"
)

func getUserPropsRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/userprops/:id",
			// swagger:route GET /v1/userprops/{id} UserPropsGet
			//
			// Get user properties. ntnx:ignore
			//
			// Retrieves the properties for the user with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserPropsGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetUserPropsW, "/userprops/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/userprops/:id",
			// swagger:route GET /v1.0/userprops/{id} User_Props UserPropsGetV2
			//
			// Get user properties. ntnx:ignore
			//
			// Retrieves the properties for the user with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: UserPropsGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetUserPropsW, "/userprops/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/userprops/:id",
			// swagger:route DELETE /v1/userprops/{id} UserPropsDelete
			//
			// Delete user properties by user ID. ntnx:ignore
			//
			// Deletes the properties for the user with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteUserPropsW, msgSvc, "userprops", NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/userprops/:id",
			// swagger:route DELETE /v1.0/userprops/{id} User_Props UserPropsDeleteV2
			//
			// Delete user properties by ID. ntnx:ignore
			//
			// Deletes the properties for the user with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteUserPropsWV2, msgSvc, "userprops", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1/userprops/:id",
			// swagger:route PUT /v1/userprops/{id} UserPropsUpdate
			//
			// Update user properties by ID. ntnx:ignore
			//
			// Updates the properties of the user with the given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserPropsW, msgSvc, "userprops", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/userprops/:id",
			// swagger:route PUT /v1.0/userprops/{id} User_Props UserPropsUpdateV2
			//
			// Update user properties by ID. ntnx:ignore
			//
			// Updates the properties of the user with the given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateUserPropsWV2, msgSvc, "userprops", NOTIFICATION_NONE, "id"),
		},
	}
}

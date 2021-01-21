package router

import (
	"cloudservices/cloudmgmt/api"
)

func getDataDriverClassRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/datadriverclasses",
			// swagger:route GET /v1.0/datadriverclasses Data_Driver_Class DataDriverClassList
			//
			// Get all data driver class. ntnx:ignore
			//
			// Retrieves a list of all data driver classes.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataDriverClassListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataDriverClassesW, "/datadriverclasses"),
		},
		{
			method: "GET",
			path:   "/v1.0/datadriverclasses/:id",
			// swagger:route GET /v1.0/datadriverclasses/{id} Data_Driver_Class DataDriverClassGet
			//
			// Get a data driver class by ID. ntnx:ignore
			//
			// Get a data driver class according to its given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataDriverClassGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataDriverClassW, "/datadriverclasses/:id", "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/datadriverclasses",
			// swagger:route POST /v1.0/datadriverclasses Data_Driver_Class DataDriverClassCreate
			//
			// Create a data driver class. ntnx:ignore
			//
			// Create a data driver class.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataDriverClassW, msgSvc, EntityTypeDataDriverClass, NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1.0/datadriverclasses/:id",
			// swagger:route PUT /v1.0/datadriverclasses/{id} Data_Driver_Class DataDriverClassUpdate
			//
			// Update a data driver class. ntnx:ignore
			//
			// Update a data driver class.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataDriverClassW, msgSvc, EntityTypeDataDriverClass, NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/datadriverclasses/:id",
			// swagger:route DELETE /v1.0/datadriverclasses/{id} Data_Driver_Class DataDriverClassDelete
			//
			// Delete a specific data driver class. ntnx:ignore
			//
			// Delete a data driver with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataDriverClassW, msgSvc, EntityTypeDataDriverClass, NOTIFICATION_NONE, "id"),
		},
	}
}

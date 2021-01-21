package router

import (
	"cloudservices/cloudmgmt/api"
)

func getDataDriverInstanceRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/datadriverinstances",
			// swagger:route GET /v1.0/datadriverinstances Data_Driver_Instance DataDriverInstancesList
			//
			// Get all data driver instances. ntnx:ignore
			//
			// Retrieves a list of all data driver isntances.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataDriverInstanceListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataDriverInstancesW, "/datadriverinstances"),
		},
		{
			method: "GET",
			path:   "/v1.0/datadriverclasses/:id/instances",
			// swagger:route GET /v1.0/datadriverclasses/{id}/instances Data_Driver_Class DataDriverInstancesByClassIdList
			//
			// Get all data driver instances by class id. ntnx:ignore
			//
			// Retrieves a list of all data driver instances by class ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataDriverClassInstanceListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectAllDataDriverInstancesByClassIdW, "/datadriverclasses/:id/instances", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datadriverinstances/:id",
			// swagger:route GET /v1.0/datadriverinstances/{id} Data_Driver_Instance DataDriverInstanceGet
			//
			// Get a data driver instance by ID. ntnx:ignore
			//
			// Get a data driver instance according to its given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataDriverInstanceGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataDriverInstanceW, "/datadriverinstances/:id", "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/datadriverinstances",
			// swagger:route POST /v1.0/datadriverinstances Data_Driver_Instance DataDriverInstanceCreate
			//
			// Create a data driver instance. ntnx:ignore
			//
			// Create a data driver instance.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataDriverInstanceW, msgSvc, EntityTypeDataDriverInstance, NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1.0/datadriverinstances/:id",
			// swagger:route PUT /v1.0/datadriverinstances/{id} Data_Driver_Instance DataDriverInstanceUpdate
			//
			// Update a data driver instance. ntnx:ignore
			//
			// Update a data driver instance.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataDriverInstanceW, msgSvc, EntityTypeDataDriverInstance, NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/datadriverinstances/:id",
			// swagger:route DELETE /v1.0/datadriverinstances/{id} Data_Driver_Instance DataDriverInstanceDelete
			//
			// Delete a specific data driver instance. ntnx:ignore
			//
			// Delete a data driver instance with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataDriverInstanceW, msgSvc, EntityTypeDataDriverInstance, NOTIFICATION_TENANT, "id"),
		},
	}
}

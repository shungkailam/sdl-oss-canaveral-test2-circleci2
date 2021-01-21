package router

import (
	"cloudservices/cloudmgmt/api"
)

func getDataDriverStreamRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/datadriverinstances/:id/streams",
			// swagger:route GET /v1.0/datadriverinstances/{id}/streams Data_Driver_Stream DataDriverStreamList
			//
			// Get a data driver stream parameters for data driver instance by ID. ntnx:ignore
			//
			// Get a data driver stream parameters according to its instance ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataDriverStreamListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectDataDriverStreamsByInstanceIdW, "/datadriverinstances/:id/streams", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datadriverstreams/:id",
			// swagger:route GET /v1.0/datadriverstreams/{id} Data_Driver_Stream DataDriverStreamGet
			//
			// Get a data driver stream parameters by ID. ntnx:ignore
			//
			// Get a data driver stream parameters according to its given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataDriverStreamGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataDriverStreamW, "/datadriverstreams/:id", "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/datadriverstreams",
			// swagger:route POST /v1.0/datadriverstreams Data_Driver_Stream DataDriverStreamCreate
			//
			// Create a data driver stream parameters. ntnx:ignore
			//
			// Create a data driver stream parameters.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataDriverStreamW, msgSvc, EntityTypeDataDriverStream, NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1.0/datadriverstreams/:id",
			// swagger:route PUT /v1.0/datadriverstreams/{id} Data_Driver_Stream DataDriverStreamUpdate
			//
			// Update a data driver stream parameters. ntnx:ignore
			//
			// Update a data driver stream parameters.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataDriverStreamW, msgSvc, EntityTypeDataDriverStream, NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/datadriverstreams/:id",
			// swagger:route DELETE /v1.0/datadriverstreams/{id} Data_Driver_Stream DataDriverStreamDelete
			//
			// Delete a specific data driver stream parameters. ntnx:ignore
			//
			// Delete a data driver stream parameters with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataDriverStreamW, msgSvc, EntityTypeDataDriverStream, NOTIFICATION_TENANT, "id"),
		},
	}
}

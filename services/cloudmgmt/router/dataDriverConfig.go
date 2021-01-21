package router

import (
	"cloudservices/cloudmgmt/api"
)

func getDataDriverConfigRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/datadriverinstances/:id/configs",
			// swagger:route GET /v1.0/datadriverinstances/{id}/configs Data_Driver_Config DataDriverConfigList
			//
			// Get a data driver config parameters for data driver instance by ID. ntnx:ignore
			//
			// Get a data driver config parameters according to its instance ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataDriverConfigListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectDataDriverConfigsByInstanceIdW, "/datadriverinstances/:id/configs", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datadriverconfigs/:id",
			// swagger:route GET /v1.0/datadriverconfigs/{id} Data_Driver_Config DataDriverConfigGet
			//
			// Get a data driver config parameters by ID. ntnx:ignore
			//
			// Get a data driver config parameters according to its given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataDriverConfigGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataDriverConfigW, "/datadriverconfigs/:id", "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/datadriverconfigs",
			// swagger:route POST /v1.0/datadriverconfigs Data_Driver_Config DataDriverConfigCreate
			//
			// Create a data driver config parameters. ntnx:ignore
			//
			// Create a data driver config parameters.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataDriverConfigW, msgSvc, EntityTypeDataDriverConfig, NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1.0/datadriverconfigs/:id",
			// swagger:route PUT /v1.0/datadriverconfigs/{id} Data_Driver_Config DataDriverConfigUpdate
			//
			// Update a data driver config parameters. ntnx:ignore
			//
			// Update a data driver config parameters.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataDriverConfigW, msgSvc, EntityTypeDataDriverConfig, NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/datadriverconfigs/:id",
			// swagger:route DELETE /v1.0/datadriverconfigs/{id} Data_Driver_Config DataDriverConfigDelete
			//
			// Delete a specific data driver config parameters. ntnx:ignore
			//
			// Delete a data driver config parameters with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataDriverConfigW, msgSvc, EntityTypeDataDriverConfig, NOTIFICATION_TENANT, "id"),
		},
	}
}

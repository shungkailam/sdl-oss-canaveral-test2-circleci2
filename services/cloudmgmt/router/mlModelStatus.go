package router

import (
	"cloudservices/cloudmgmt/api"
)

func getMLModelStatusRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/mlmodelstatuses",
			// swagger:route GET /v1.0/mlmodelstatuses MLModel_Status MLModelStatusList
			//
			// Get status for all ML models.
			//
			// Retrieves status for all ML models.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: MLModelStatusListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllMLModelsStatusW, "/mlmodelstatuses"),
		},
		{
			method: "GET",
			path:   "/v1.0/mlmodelstatuses/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllMLModelsStatusW, "/mlmodelstatuses"),
		},
		{
			method: "GET",
			path:   "/v1.0/mlmodelstatuses/:id",
			// swagger:route GET /v1.0/mlmodelstatuses/{id} MLModel_Status MLModelStatusGet
			//
			// Get ML model status by model ID.
			//
			// Retrieve status for an ML model with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: MLModelStatusListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetMLModelStatusW, "/mlmodelstatuses/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/mlmodelstatuses/:id",
			// (not public) DELETE /v1.0/mlmodelstatuses/{id} MLModel_Status MLModelStatusDelete
			//
			// Delete an ML model status by model ID.
			//
			// Deletes the ML model status with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteMLModelStatusW, msgSvc, "mlmodelstatuses", NOTIFICATION_NONE, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/mlmodelstatuses",
			// (not public) POST /v1.0/mlmodelstatuses MLModel_Status MLModelStatusCreate
			//
			// Create ML model status.
			//
			// Creates an ML model status.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateMLModelStatusW, msgSvc, "mlmodelstatuses", NOTIFICATION_NONE),
		},
	}
}

package router

import (
	"cloudservices/cloudmgmt/api"
)

func getMLModelRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/mlmodels",
			// swagger:route GET /v1.0/mlmodels ML_Model MLModelList
			//
			// Lists machine learning models.
			//
			// Retrieve all machine learning models.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: MLModelListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllMLModelsW, "/mlmodels"),
		},
		{
			method: "GET",
			path:   "/v1.0/mlmodels/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllMLModelsW, "/mlmodels"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/mlmodels",
			// swagger:route GET /v1.0/projects/{projectId}/mlmodels ML_Model ProjectGetMLModels
			//
			// Lists project machine learning models by project ID.
			//
			// Retrieves all machine learning models for a project by its given ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: MLModelListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllMLModelsForProjectW, "/project-mlmodels", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/mlmodels/",
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllMLModelsForProjectW, "/project-mlmodels", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/mlmodels/:id",
			// swagger:route GET /v1.0/mlmodels/{id} ML_Model MLModelGet
			//
			// Get machine learning model by its ID.
			//
			// Retrieves a machine learning model by its given ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: MLModelGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetMLModelW, "/mlmodels/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/mlmodels/:id",
			// swagger:route DELETE /v1.0/mlmodels/{id} ML_Model MLModelDelete
			//
			// Delete a machine learning model  by its ID.
			//
			// Deletes a machine learning model by its given ID.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteMLModelW, msgSvc, "mlmodel", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/mlmodels",
			// swagger:route POST /v1.0/mlmodels ML_Model MLModelCreate
			//
			// Create a machine learning model.
			//
			// Creates a machine learning model.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateMLModelW, msgSvc, "mlmodel", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1.0/mlmodels/:id",
			// swagger:route PUT /v1.0/mlmodels/{id} ML_Model MLModelUpdate
			//
			// Update a machine learning model by its ID.
			//
			// Updates a machine learning model by its given ID.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateMLModelW, msgSvc, "mlmodel", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/mlmodels/:id/versions",
			// swagger:route POST /v1.0/mlmodels/{id}/versions ML_Model MLModelVersionCreate
			//
			// Create a new version of the machine learning model by its ID.
			//
			// Create a new version of the machine learning model by its given ID.
			//
			//     Consumes:
			//     - multipart/form-data
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
			handle: makeCreateHandle2(dbAPI, dbAPI.CreateMLModelVersionW, msgSvc, EntityMessage{"mlmodelversion", "onUpdateMLModel"}, NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/mlmodels/:id/versions/:model_version",
			// swagger:route PUT /v1.0/mlmodels/{id}/versions/{model_version} ML_Model MLModelVersionUpdate
			//
			// Update the version of the machine learning model by its ID.
			//
			// Updates the version of the machine learning model by machine learning model ID.
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
			handle: makeUpdateHandle2(dbAPI, dbAPI.UpdateMLModelVersionW, msgSvc, EntityMessage{"mlmodelversion", "onUpdateMLModel"}, NOTIFICATION_NONE, "id", "model_version"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/mlmodels/:id/versions/:model_version",
			// swagger:route DELETE /v1.0/mlmodels/{id}/versions/{model_version} ML_Model MLModelVersionDelete
			//
			// Delete the version of the machine learning model by its ID.
			//
			// Deletes the version of the machine learning model by machine learning model ID.
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
			handle: makeDeleteHandle2(dbAPI, dbAPI.DeleteMLModelVersionW, msgSvc, EntityMessage{"mlmodelversion", "onUpdateMLModel"}, NOTIFICATION_TENANT, "id", "model_version"),
		},
		// GET /v1.0/mlmodels/<model id>/versions/<model version>/url?expiration_duration=<mins>
		{
			method: "GET",
			path:   "/v1.0/mlmodels/:id/versions/:model_version/url",
			// swagger:route GET /v1.0/mlmodels/{id}/versions/{model_version}/url ML_Model MLModelVersionURLGet
			//
			// Get a pre-signed URL for the machine learning model according to its ID and version.
			//
			// Retrieves a pre-signed URL for the machine learning model according to its ID and version.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: MLModelVersionURLGetResponse
			//       default: APIError
			handle: makeGetHandle2(dbAPI, dbAPI.GetMLModelVersionSignedURLW, "/mlmodels/:id", "id", "model_version"),
		},
	}
}

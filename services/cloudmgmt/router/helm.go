package router

import (
	"cloudservices/cloudmgmt/api"
)

func getHelmRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{

		{
			method: "POST",
			path:   "/v1.0/helm/template",
			// swagger:route POST /v1.0/helm/template Helm HelmTemplate
			//
			// Run Helm Template.
			//
			// Run Helm Template to render Helm Chart.
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
			//       200: HelmTemplateResponse
			//       default: APIError
			handle: makeCreateHandle2(dbAPI, dbAPI.RunHelmTemplateW, msgSvc, EntityMessage{"application", "helmTemplate"}, NOTIFICATION_NONE, "unused"),
		},
		{
			method: "POST",
			path:   "/v1.0/helm/apps",
			// swagger:route POST /v1.0/helm/apps Helm HelmApplicationCreate
			//
			// Create Helm Application.
			//
			// Create a Helm Chart based Application.
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
			handle: makeCreateHandle2(dbAPI, dbAPI.CreateHelmApplicationW, msgSvc, EntityMessage{"application", "onCreateApplication"}, NOTIFICATION_TENANT, "unused"),
		},
		{
			method: "PUT",
			path:   "/v1.0/helm/apps/:id",
			// swagger:route PUT /v1.0/helm/apps/{id} Helm HelmApplicationUpdate
			//
			// Update Helm Application.
			//
			// Update a Helm Chart based Application.
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
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makeCreateHandle2(dbAPI, dbAPI.UpdateHelmApplicationW, msgSvc, EntityMessage{"application", "onUpdateApplication"}, NOTIFICATION_TENANT, "id"),
		},
	}
}

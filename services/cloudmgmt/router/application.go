package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
)

func getApplicationRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/applications",
			// swagger:route GET /v1/applications ApplicationList
			//
			// Get all applications. ntnx:ignore
			//
			// Retrieves a list of all applications.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ApplicationListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsW, "/applications"),
		},
		{
			method: "GET",
			path:   "/v1/applications/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsW, "/applications"),
		},
		{
			method: "GET",
			path:   "/v1.0/applications",
			// swagger:route GET /v1.0/applications Application ApplicationListV2
			//
			// Get all applications.
			//
			// Retrieves a list of all applications.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ApplicationListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsWV2, "/applications"),
		},
		{
			method: "GET",
			path:   "/v1.0/applications/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsWV2, "/applications"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/applications",
			// swagger:route GET /v1/projects/{projectId}/applications ProjectGetApplications
			//
			// Get all applications in a project according to project ID. ntnx:ignore
			//
			// Retrieves a list of all applications in a project with ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ApplicationListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsForProjectW, "/project-applications", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/applications",
			// swagger:route GET /v1.0/projects/{projectId}/applications Application ProjectGetApplicationsV2
			//
			// Get all applications in a project according to project ID.
			//
			// Retrieves a list of all applications in a project with ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ApplicationListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsForProjectWV2, "/project-applications", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/applications/",
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllApplicationsForProjectWV2, "/project-applications", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/application/:id",
			// swagger:route GET /v1/application/{id} ApplicationGet
			//
			// Get application by application ID. ntnx:ignore
			//
			// Retrieves the application according to its ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ApplicationGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetApplicationW, "/application/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/applications/:id",
			// swagger:route GET /v1.0/applications/{id} Application ApplicationGetV2
			//
			// Get application by application ID.
			//
			// Retrieves the application with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ApplicationGetResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetApplicationWV2, "/application/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/applications/:id/containers/:edgeId",
			// swagger:route GET /v1.0/applications/{id}/containers/{edgeId} Application GetApplicationContainers
			//
			// Get containers of an application specified by Application ID running on a specific edge.
			//
			// Gets the containers of an application with the given ID {id} running on edge with id {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: GetApplicationContainersResponse
			//       default: APIError
			handle: makeGetHandle2WithWSCallback(dbAPI, dbAPI.GetApplicationContainersW, "id", "edgeId", msgSvc, "onGetApplicationContainers", NOTIFICATION_EDGE_SYNC, func(doc interface{}) *string {
				i, ok := doc.(model.ApplicationContainersBaseObject)
				if !ok {
					return nil
				}
				return &i.EdgeID
			}),
		},
		{
			method: "POST",
			path:   "/v1/applications/:id/render/:edgeId",
			// swagger:route POST /v1/applications/{id}/render/{edgeId} Application RenderApplication
			//
			// Render Application ID running on a specific edge. ntnx:ignore
			//
			// Render application template with the given ID {id} running on edge with id {edgeId}.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: RenderApplicationResponse
			//       default: APIError
			handle: makePostHandle2(dbAPI, dbAPI.RenderApplicationW, "/application/:id/render/:edgeId", "id", "edgeId"),
		},
		{
			method: "DELETE",
			path:   "/v1/application/:id",
			// swagger:route DELETE /v1/application/{id} ApplicationDelete
			//
			// Delete application specified by the application ID. ntnx:ignore
			//
			// Deletes the application with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteApplicationW, msgSvc, "application", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/applications/:id",
			// swagger:route DELETE /v1.0/applications/{id} Application ApplicationDeleteV2
			//
			// Delete application specified by the application ID.
			//
			// Deletes the application with the given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteApplicationWV2, msgSvc, "application", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/application",
			// swagger:route POST /v1/application ApplicationCreate
			//
			// Create an application. ntnx:ignore
			//
			// Create an application.
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
			//       200: CreateDocumentResponse
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateApplicationW, msgSvc, "application", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/applications",
			// swagger:route POST /v1.0/applications Application ApplicationCreateV2
			//
			// Create an application.
			//
			// Create an application.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateApplicationWV2, msgSvc, "application", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/application",
			// swagger:route PUT /v1/application ApplicationUpdate
			//
			// Update an application. ntnx:ignore
			//
			// Update an existing application.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateApplicationW, msgSvc, "application", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/application/:id",
			// swagger:route PUT /v1/application/{id} ApplicationUpdateV2
			//
			// Update an application specified by its ID. ntnx:ignore
			//
			// Update a specific application with ID {id}.
			// You cannot change the project associated with the application or the application ID.
			// You can change all other attributes.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateApplicationW, msgSvc, "application", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/applications/:id",
			// swagger:route PUT /v1.0/applications/{id} Application ApplicationUpdateV3
			//
			// Update a specific application with ID {id}.
			//
			// Update a specific application with ID {id}.
			// You cannot change the project associated with the application or the application ID.
			// You can change all other attributes.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateApplicationWV2, msgSvc, "application", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/helmapp",
			// swagger:route POST /v1.0/helmapp Application HelmAppCreate
			//
			// Create a new helm chart based app. ntnx:ignore
			//
			// Create a new helm chart based app and return the charts uuid.
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
			handle: makeCreateHandle2(dbAPI, dbAPI.CreateHelmAppW, msgSvc, EntityMessage{"application", "helmapp"}, NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1.0/helmapp/:id/values",
			// swagger:route POST /v1.0/helmapp/{id}/values Application HelmValuesCreate
			//
			// Adds a values file to the helm chart identified by id. ntnx:ignore
			//
			// Adds a values file to the helm chart identified by id.
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
			handle: makeCreateHandle2(dbAPI, dbAPI.CreateHelmValuesW, msgSvc, EntityMessage{"application", "helmvalues"}, NOTIFICATION_TENANT, "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/helmapp/:id",
			// swagger:route GET /v1.0/helmapp/{id} Application HelmAppGetYaml
			//
			// Get application by application ID. ntnx:ignore
			//
			// Retrieves the application with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: HelmAppGetYamlResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetHelmAppYaml, "/helmapp/:id", "id"),
		},
	}
}

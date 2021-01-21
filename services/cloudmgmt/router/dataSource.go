package router

import (
	"cloudservices/cloudmgmt/api"
)

func getDataSourceRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/datasources",
			// swagger:route GET /v1/datasources DataSourceList
			//
			// Get all data sources. ntnx:ignore
			//
			// Retrieves a list of all data sources.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataSourceListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesW, "/datasources"),
		},
		{
			method: "GET",
			path:   "/v1/datasources/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesW, "/datasources"),
		},
		{
			method: "GET",
			path:   "/v1.0/datasources",
			// swagger:route GET /v1.0/datasources Data_Source DataSourceListV2
			//
			// Get all data sources.
			//
			// Retrieves a list of all data sources.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataSourceListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesWV2, "/datasources"),
		},
		{
			method: "GET",
			path:   "/v1.0/datasources/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesWV2, "/datasources"),
		},
		{
			method: "GET",
			path:   "/v1/edges/:edgeId/datasources",
			// swagger:route GET /v1/edges/{edgeId}/datasources EdgeGetDatasources
			//
			// Get all data sources associated with an edge. ntnx:ignore
			//
			// Retrieves a list of all data sources associated with with a edge by its ID {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceListResponse
			//       default: APIError
			handle: makeEdgeGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesForEdgeW, "/edge-datasources", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges/:edgeId/datasources",
			// swagger:route GET /v1.0/edges/{edgeId}/datasources Data_Source EdgeGetDatasourcesV2
			//
			// Get all data sources associated with an edge.
			//
			// Retrieves a list of all data sources associated with a edge by its ID {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceListResponseV2
			//       default: APIError
			handle: makeEdgeGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesForEdgeWV2, "/edge-datasources", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/datasources",
			// swagger:route GET /v1/projects/{projectId}/datasources ProjectGetDatasources
			//
			// Get data sources for a project. ntnx:ignore
			//
			// Retrieves a list of all data sources associated with a project with a given ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesForProjectW, "/project-datasources", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/datasources",
			// swagger:route GET /v1.0/projects/{projectId}/datasources Data_Source ProjectGetDatasourcesV2
			//
			// Get data sources for a project. ntnx:ignore
			//
			// Retrieves a list of all data sources associated with a project with a given ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllDataSourcesForProjectWV2, "/project-datasources", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/datasources/:id",
			// swagger:route GET /v1/datasources/{id} DataSourceGet
			//
			// Get the data source according to its ID. ntnx:ignore
			//
			// Get the data source according to its given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataSourceW, "/datasources/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datasources/:id",
			// swagger:route GET /v1.0/datasources/{id} Data_Source DataSourceGetV2
			//
			// Get a data source according to its ID.
			//
			// Get a data source according to its given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceGetResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataSourceWV2, "/datasources/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datasources/:id/artifacts",
			// swagger:route GET /v1.0/datasources/{id}/artifacts Data_Source DataSourceGetArtifactV2
			//
			// Get data source artifacts according to its ID.
			//
			// Retrieves the artifacts after deploying the data source with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataSourceGetArtifactResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataSourceArtifactWV2, "/datasources/:id/artifacts", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/datasources/:id",
			// swagger:route DELETE /v1/datasources/{id} DataSourceDelete
			//
			// Delete a specific data source. ntnx:ignore
			//
			// Delete a data source with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataSourceW, msgSvc, "datasource", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/datasources/:id",
			// swagger:route DELETE /v1.0/datasources/{id} Data_Source DataSourceDeleteV2
			//
			// Delete a specific data source.
			//
			// Delete a data source with a given ID {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataSourceWV2, msgSvc, "datasource", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "POST",
			path:   "/v1/datasources",
			// swagger:route POST /v1/datasources DataSourceCreate
			//
			// Create a data source. ntnx:ignore
			//
			// Create a data source.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataSourceW, msgSvc, "datasource", NOTIFICATION_EDGE),
		},
		{
			method: "POST",
			path:   "/v1.0/datasources",
			// swagger:route POST /v1.0/datasources Data_Source DataSourceCreateV2
			//
			// Create a data source.
			//
			// Create a data source.
			//
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataSourceWV2, msgSvc, "datasource", NOTIFICATION_EDGE),
		},
		{
			method: "POST",
			path:   "/v1.0/datasources/:id/artifacts",
			// swagger:route POST /v1.0/datasources/{id}/artifacts Data_Source DataSourceCreateArtifactV2
			//
			// Create data source artifact according to its ID. ntnx:ignore
			//
			// Create data source artifact according to its given ID {id}.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataSourceArtifactWV2, msgSvc, "datasourceartifact", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1/datasources",
			// swagger:route PUT /v1/datasources DataSourceUpdate
			//
			// Update a data source. ntnx:ignore
			//
			// Update a data source. You cannot update or change the edge associated with the data source by using this call.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataSourceW, msgSvc, "datasource", NOTIFICATION_EDGE, ""),
		},
		{
			method: "PUT",
			path:   "/v1/datasources/:id",
			// swagger:route PUT /v1/datasources/{id} DataSourceUpdateV2
			//
			// Update a data source. ntnx:ignore
			//
			// Update a data source. You cannot update or change the edge associated with the data source by using this call.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataSourceW, msgSvc, "datasource", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/datasources/:id",
			// swagger:route PUT /v1.0/datasources/{id} Data_Source DataSourceUpdateV3
			//
			// Update a data source.
			//
			// Update a data source. You cannot update or change the edge associated with the data source by using this call.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataSourceWV2, msgSvc, "datasource", NOTIFICATION_EDGE, "id"),
		},
	}
}

package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
)

func getDataStreamRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/datastreams",
			// swagger:route GET /v1/datastreams DataStreamList
			//
			// Gets data pipelines. ntnx:ignore
			//
			// Retrieves all data pipelines for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataStreamListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataStreamsW, "/datastreams"),
		},
		{
			method: "GET",
			path:   "/v1/datastreams/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataStreamsW, "/datastreams"),
		},
		{
			method: "GET",
			path:   "/v1.0/datapipelines",
			// swagger:route GET /v1.0/datapipelines Data_Pipeline DataPipelineList
			//
			// Gets data pipelines.
			//
			// Retrieves all data pipelines for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: DataPipelineListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataStreamsWV2, "/datastreams"),
		},
		{
			method: "GET",
			path:   "/v1.0/datapipelines/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllDataStreamsWV2, "/datastreams"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/datastreams",
			// swagger:route GET /v1/projects/{projectId}/datastreams ProjectGetDataStreams
			//
			// Gets data pipelines for a project. ntnx:ignore
			//
			// Retrieves all data pipelines for a project of a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataStreamListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllDataStreamsForProjectW, "/project-datastreams", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/datapipelines",
			// swagger:route GET /v1.0/projects/{projectId}/datapipelines Data_Pipeline ProjectGetDataPipelines
			//
			// Gets data pipelines for a project.
			//
			// Retrieves all data pipelines for a project with a given ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataPipelineListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllDataStreamsForProjectWV2, "/project-datastreams", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/datastreams/:id",
			// swagger:route GET /v1/datastreams/{id} DataStreamGet
			//
			// Gets data pipeline by its ID. ntnx:ignore
			//
			// Retrieves a data pipelines with a given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataStreamGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataStreamW, "/datastreams/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datapipelines/:id",
			// swagger:route GET /v1.0/datapipelines/{id} Data_Pipeline DataPipelineGet
			//
			// Lists data pipeline by its ID.
			//
			// Retrieves a data pipelines with a given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DataPipelineGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetDataStreamW, "/datastreams/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/datapipelines/:id/containers/:edgeId",
			// swagger:route GET /v1.0/datapipelines/{id}/containers/{edgeId} Data_Pipeline GetDataPipelineContainers
			//
			// Get containers of a data pipeline specified by Datapipeline ID running on a specific edge.
			//
			// Gets the containers of a data pipeline with the given ID {id} running on edge with id {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: GetDataPipelineContainersResponse
			//       default: APIError
			handle: makeGetHandle2WithWSCallback(dbAPI, dbAPI.GetDataPipelineContainersW, "id", "edgeId", msgSvc, "onGetDataPipelineContainers", NOTIFICATION_EDGE_SYNC, func(doc interface{}) *string {
				i, ok := doc.(model.DataPipelineContainersBaseObject)
				if !ok {
					return nil
				}
				return &i.EdgeID
			}),
		},
		{
			method: "DELETE",
			path:   "/v1/datastreams/:id",
			// swagger:route DELETE /v1/datastreams/{id} DataStreamDelete
			//
			// Delete data pipeline. ntnx:ignore
			//
			// This will delete the data pipeline with the given id.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataStreamW, msgSvc, "datastream", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/datapipelines/:id",
			// swagger:route DELETE /v1.0/datapipelines/{id} Data_Pipeline DataPipelineDelete
			//
			// Deletes data pipeline by its ID.
			//
			// Delete the data pipeline with the given id {id}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteDataStreamWV2, msgSvc, "datastream", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "POST",
			path:   "/v1/datastreams",
			// swagger:route POST /v1/datastreams DataStreamCreate
			//
			// Creates a data pipeline. ntnx:ignore
			//
			// Create a data pipeline.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataStreamW, msgSvc, "datastream", NOTIFICATION_TENANT),
		},
		{
			method: "POST",
			path:   "/v1.0/datapipelines",
			// swagger:route POST /v1.0/datapipelines Data_Pipeline DataPipelineCreate
			//
			// Creates a data pipeline.
			//
			// Create a data pipeline.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateDataStreamWV2, msgSvc, "datastream", NOTIFICATION_TENANT),
		},
		{
			method: "PUT",
			path:   "/v1/datastreams",
			// swagger:route PUT /v1/datastreams DataStreamUpdate
			//
			// Updates a data pipeline. ntnx:ignore
			//
			// Update a data pipeline.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataStreamW, msgSvc, "datastream", NOTIFICATION_TENANT, ""),
		},
		{
			method: "PUT",
			path:   "/v1/datastreams/:id",
			// swagger:route PUT /v1/datastreams/{id} DataStreamUpdateV2
			//
			// Updates a data pipeline. ntnx:ignore
			//
			// Update a data pipeline.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataStreamW, msgSvc, "datastream", NOTIFICATION_TENANT, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/datapipelines/:id",
			// swagger:route PUT /v1.0/datapipelines/{id} Data_Pipeline DataPipelineUpdate
			//
			// Updates a data pipeline by its ID
			//
			// Update a data pipeline with a given ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateDataStreamWV2, msgSvc, "datastream", NOTIFICATION_TENANT, "id"),
		},
	}
}

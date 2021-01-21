package router

import (
	"cloudservices/cloudmgmt/api"
)

func getEdgeRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/edges",
			// swagger:route GET /v1/edges EdgeList
			//
			// Get edges. ntnx:ignore
			//
			// Retrieves all edges for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesW, "/edges"),
		},
		{
			method: "GET",
			path:   "/v1/edges/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesW, "/edges"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges",
			// swagger:route GET /v1.0/edges Edge EdgeListV2
			//
			// Get edges. ntnx:ignore
			//
			// Retrieves all edges for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesWV2, "/edges"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgesWV2, "/edges"),
		},
		{
			method: "GET",
			path:   "/v1/projects/:projectId/edges",
			// swagger:route GET /v1/projects/{projectId}/edges ProjectGetEdges
			//
			// Get project edges by ID. ntnx:ignore
			//
			// Retrieves all edges for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllEdgesForProjectW, "/project-edges", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/edges",
			// swagger:route GET /v1.0/projects/{projectId}/edges Edge ProjectGetEdgesV2
			//
			// Get project edges by ID. ntnx:ignore
			//
			// Retrieves all edges for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeListResponseV2
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllEdgesForProjectWV2, "/project-edges", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1/edges/:edgeId",
			// swagger:route GET /v1/edges/{edgeId} EdgeGet
			//
			// Get edge by ID. ntnx:ignore
			//
			// Retrieves the edge with the given ID {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetEdgeW, "/edges/:edgeId", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges/:edgeId",
			// swagger:route GET /v1.0/edges/{edgeId} Edge EdgeGetV2
			//
			// Get edge by ID. ntnx:ignore
			//
			// Retrieves the edge with the given ID {edgeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeGetResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetEdgeWV2, "/edges/:edgeId", "edgeId"),
		},
		{
			method: "DELETE",
			path:   "/v1/edges/:edgeId",
			// swagger:route DELETE /v1/edges/{edgeId} EdgeDelete
			//
			// Delete edge by ID. ntnx:ignore
			//
			// Deletes the edge with the given ID {edgeId}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteEdgeW, msgSvc, "edge", NOTIFICATION_EDGE, "edgeId"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/edges/:edgeId",
			// swagger:route DELETE /v1.0/edges/{edgeId} Edge EdgeDeleteV2
			//
			// Delete edge by ID. ntnx:ignore
			//
			// Deletes the edge with the given ID  {edgeId}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteEdgeWV2, msgSvc, "edge", NOTIFICATION_EDGE, "edgeId"),
		},
		{
			method: "POST",
			path:   "/v1/edges",
			// swagger:route POST /v1/edges EdgeCreate
			//
			// Create edge. ntnx:ignore
			//
			// Creates an edge.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateEdgeW, msgSvc, "edge", NOTIFICATION_NONE),
		},
		{
			method: "POST",
			path:   "/v1.0/edges",
			// swagger:route POST /v1.0/edges Edge EdgeCreateV2
			//
			// Create edge. ntnx:ignore
			//
			// Create an edge.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateEdgeWV2, msgSvc, "edge", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1/edges",
			// swagger:route PUT /v1/edges EdgeUpdate
			//
			// Update an edge. ntnx:ignore
			//
			// Updates an edge.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateEdgeW, msgSvc, "edge", NOTIFICATION_EDGE, ""),
		},
		{
			method: "PUT",
			path:   "/v1/edges/:id",
			// swagger:route PUT /v1/edges/{id} EdgeUpdateV2
			//
			// Update edge. ntnx:ignore
			//
			// Updates an edge.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateEdgeW, msgSvc, "edge", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/edges/:id",
			// swagger:route PUT /v1.0/edges/{id} Edge EdgeUpdateV3
			//
			// Update edge by its ID. ntnx:ignore
			//
			// Updates an edge by its ID {id}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateEdgeWV2, msgSvc, "edge", NOTIFICATION_EDGE, "id"),
		},
		{
			method: "POST",
			// path change b/c httprouter view /v1/edges/:edgeId/handle as conflict route
			path: "/v1/edgehandle/:edgeId",
			// swagger:route POST /v1/edgehandle/{edgeId} EdgeGetHandle
			//
			// Get edge certification. ntnx:ignore
			//
			// Retrieves the certificate and private key for the edge by its given ID {edgeId}.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//         - BearerToken:
			//
			//
			//     Responses:
			//       200: EdgeGetHandleResponse
			//       default: APIError
			handle: makeGetHandleNoAuth2(dbAPI, dbAPI.GetEdgeHandleW, "/v1/edgehandle/:edgeId", "edgeId"),
		},
		{
			method: "POST",
			// path change b/c httprouter view /v1/edges/serialnumber as conflict route
			path: "/v1/edgebyserialnumber",
			// swagger:route POST /v1/edgebyserialnumber EdgeGetBySerialNumber
			//
			// Get edge by serial number. ntnx:ignore
			//
			// Retrieves the edge according to the given serial number.
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
			//
			//     Responses:
			//       200: EdgeGetBySerialNumberResponse
			//       default: APIError
			handle: makeGetHandleNoAuth(dbAPI, dbAPI.GetEdgeBySerialNumberW, "/v1/edgebyserialnumber"),
		},
	}
}

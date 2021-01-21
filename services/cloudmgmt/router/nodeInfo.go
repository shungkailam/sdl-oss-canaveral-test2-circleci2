package router

import (
	"cloudservices/cloudmgmt/api"
)

func getNodeInfoRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/nodesinfo",
			// swagger:route GET /v1.0/nodesinfo Node_Info NodeInfoList
			//
			// Get node resource, build, and version details.
			//
			// Retrieves all node resource, build, and version details.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: NodeInfoListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllNodesInfoWV2, "/nodesinfo"),
		},
		{
			method: "GET",
			path:   "/v1.0/nodesinfo/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllNodesInfoWV2, "/nodesinfo"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/nodesinfo",
			// swagger:route GET /v1.0/projects/{projectId}/nodesinfo Node_Info ProjectGetNodesInfo
			//
			// Get all node information for a project by project ID.
			//
			// Retrieves all node resource, build, and version details by project ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: NodeInfoListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllNodesInfoForProjectWV2, "/project-nodesinfo", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/nodesinfo/:nodeId",
			// swagger:route GET /v1.0/nodesinfo/{nodeId} Node_Info NodeInfoGet
			//
			// Get all node information by node ID.
			//
			// Retrieves all node resource, build, and version details for a given node ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: NodeInfoGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetNodeInfoW, "/nodesinfo", "nodeId"),
		},
		{
			method: "PUT",
			path:   "/v1.0/nodesinfo/:nodeId",
			// swagger:route PUT /v1.0/nodesinfo/{nodeId} Node_Info NodeInfoUpdate
			//
			// Update node information by node ID.
			//
			// Update node resource, build, and version details for a given node ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.CreateNodeInfoW, msgSvc, "/nodesinfo", NOTIFICATION_NONE, "nodeId"),
		},
	}
}

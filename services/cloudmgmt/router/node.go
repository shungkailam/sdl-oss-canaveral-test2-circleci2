package router

import (
	"cloudservices/cloudmgmt/api"
)

func getNodeRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/nodes",
			// swagger:route GET /v1.0/nodes Node NodeList
			//
			// Get all service domain nodes.
			//
			// Retrieves all service domain nodes for your account.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: NodeListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllNodesW, "/nodes"),
		},
		{
			method: "GET",
			path:   "/v1.0/nodes/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllNodesW, "/nodes"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/nodes",
			// swagger:route GET /v1.0/projects/{projectId}/nodes Node ProjectGetNodes
			//
			// Get all service domain nodes associated with a project by project ID.
			//
			// Retrieves all service domain nodes for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: NodeListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllNodesForProjectW, "/project-nodes", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/nodes/:nodeId",
			// swagger:route GET /v1.0/nodes/{nodeId} Node NodeGet
			//
			// Get a node as specified by node ID.
			//
			// Retrieves the node with the given ID {nodeId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: NodeGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetNodeW, "/nodes/:nodeId", "nodeId"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/nodes/:nodeId",
			// swagger:route DELETE /v1.0/nodes/{nodeId} Node NodeDelete
			//
			// Delete a node as specified by node ID.
			//
			// Deletes the node with the given ID  {nodeId}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteNodeW, msgSvc, "node", NOTIFICATION_EDGE, "nodeId"),
		},
		{
			method: "POST",
			path:   "/v1.0/nodes",
			// swagger:route POST /v1.0/nodes Node NodeCreate
			//
			// Create a node.
			//
			// Create a node.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateNodeW, msgSvc, "node", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1.0/nodes/:nodeId",
			// swagger:route PUT /v1.0/nodes/{nodeId} Node NodeUpdate
			//
			// Update a node as specified by node ID.
			//
			// Updates a node by its ID {nodeId}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateNodeW, msgSvc, "node", NOTIFICATION_EDGE, "nodeId"),
		},
		{
			method: "POST",
			path:   "/v1.0/nodebyserialnumber",
			// swagger:route POST /v1.0/nodebyserialnumber NodeGetBySerialNumber
			//
			// Get a node as specified by its serial number. ntnx:ignore
			//
			// Retrieves the node according to the given serial number.
			// You can display the serial number by opening this URL in a browser.
			// Use your service domain IP address: http://service-domain-ip-address:8080/v1/sn
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: NodeGetBySerialNumberResponse
			//       default: APIError
			handle: makePostHandleNoAuth(dbAPI, dbAPI.GetNodeBySerialNumberW, "/v1.0/nodebyserialnumber"),
		},
		{
			method: "POST",
			path:   "/v1.0/nodeonboarded",
			// swagger:route POST /v1.0/nodeonboarded Node NodeOnboarded
			//
			// Update node post onboard info.
			//
			// Updates the onboard info by node ID.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: UpdateDocumentResponseV2
			//       default: APIError
			handle: makePostHandleNoAuth(dbAPI, dbAPI.UpdateNodeOnboardedW, "/v1.0/nodeonboarded"),
		},
	}
}

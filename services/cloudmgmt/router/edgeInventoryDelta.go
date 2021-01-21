package router

import (
	"cloudservices/cloudmgmt/api"
)

func getEdgeInventoryDeltaRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{

		{
			method: "POST",
			path:   "/v1.0/edgeinventorydelta",
			// swagger:route POST /v1.0/edgeinventorydelta Edge_Inventory_Delta GetEdgeInventoryDelta
			//
			// Get the edge inventory delta. ntnx:ignore
			//
			// Retrieves the edge inventory delta: changes for any entity associated with the edge.
			// Entities are projects, applications, data pipelines, functions, and so on.
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
			//       200: GetEdgeInventoryDeltaResponse
			//       default: APIError
			// handle: makeCreateHandle(dbAPI, dbAPI.GetEdgeInventoryDeltaW, msgSvc, "edge", NOTIFICATION_NONE),
			handle: makeGetAllHandle(dbAPI, dbAPI.GetEdgeInventoryDeltaW, "/edgeinventorydelta"),
		},
	}
}

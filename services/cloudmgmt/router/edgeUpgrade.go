package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
)

func getEdgeUpgradeRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	RequestEdgeUpgradeHandle := makeCustomMessageHandle(dbAPI, dbAPI.ExecuteEdgeUpgradeW, msgSvc, "executeEdgeUpgrade", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.ExecuteEdgeUpgradeData)
		edgeID := payload.EdgeID

		return &edgeID
	})
	RequestEdgeUpgradeHandleV2 := makeCustomMessageHandle(dbAPI, dbAPI.ExecuteEdgeUpgradeWV2, msgSvc, "executeEdgeUpgrade", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.ExecuteEdgeUpgradeData)
		edgeID := payload.EdgeID

		return &edgeID
	})

	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/edgesCompatibleUpgrades",
			// swagger:route GET /v1/edgesCompatibleUpgrades EdgeUpgradeList
			//
			// Lists available edge software upgrades. ntnx:ignore
			//
			// Retrieves available edge software upgrades.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeUpgradeListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgeUpgradesW, "/edgesCompatibleUpgrades"),
		},
		{
			method: "GET",
			path:   "/v1/edgesCompatibleUpgrades/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgeUpgradesW, "/edgesCompatibleUpgrades"),
		},
		{
			method: "GET",
			path:   "/v1.0/edgescompatibleupgrades",
			// swagger:route GET /v1.0/edgescompatibleupgrades Edge_Upgrade EdgeUpgradeListV2
			//
			// Lists available edge software upgrades. ntnx:ignore
			//
			// Lists all possible software upgrades that are available for all detected edges.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: EdgeUpgradeListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgeUpgradesWV2, "/edgescompatibleupgrades"),
		},
		{
			method: "GET",
			path:   "/v1.0/edgescompatibleupgrades/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgeUpgradesWV2, "/edgescompatibleupgrades"),
		},
		{
			method: "GET",
			path:   "/v1/edges/:edgeId/upgradecompatible",
			// swagger:route GET /v1/edges/{edgeId}/upgradecompatible EdgeGetUpgrades
			//
			// Lists compatible edge software upgrades by edge ID. ntnx:ignore
			//
			// Retrieves compatible software upgrades for the given edge ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeUpgradeCompatibleListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectEdgeUpgradesByEdgeIDW, "/edge-upgrades/:id", "edgeId"),
		},
		{
			method: "GET",
			path:   "/v1.0/edges/:edgeId/upgradecompatible",
			// swagger:route GET /v1.0/edges/{edgeId}/upgradecompatible Edge_Upgrade EdgeGetUpgradesV2
			//
			// Lists compatible edge software upgrades by edge ID. ntnx:ignore
			//
			// Retrieves compatible software upgrades for the given edge ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: EdgeUpgradeCompatibleListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectEdgeUpgradesByEdgeIDW, "/edge-upgrades/:id", "edgeId"),
		},
		{
			method: "POST",
			path:   "/v1/edges/upgrade",
			// swagger:route POST /v1/edges/upgrade ExecuteEdgeUpgrade
			//
			// Upgrade the edge software. ntnx:ignore
			//
			// Upgrades the edge software.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CreateDocumentResponse
			//       default: APIError
			handle: RequestEdgeUpgradeHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/edges/upgrade",
			// swagger:route POST /v1.0/edges/upgrade Edge_Upgrade ExecuteEdgeUpgradeV2
			//
			// Upgrade the edge software. ntnx:ignore
			//
			// Upgrades the edge software.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: CreateDocumentResponseV2
			//       default: APIError
			handle: RequestEdgeUpgradeHandleV2,
		},
	}
}

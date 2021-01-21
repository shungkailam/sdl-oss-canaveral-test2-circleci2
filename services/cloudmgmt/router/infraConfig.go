package router

import (
	"cloudservices/cloudmgmt/api"
)

func getInfraConfigRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/edgecluster/:id/infraconfig",
			// swagger:route GET /v1.0/edgecluster/{id}/infraconfig Infra_Config InfraConfigGet
			//
			// Get a edgeCluster config by its ID. ntnx:ignore
			//
			// Retrieves a edgeCluster config with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: InfraConfigGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetInfraConfigW, "/edgecluster/:id/infraconfig", "id"),
		},
	}
}

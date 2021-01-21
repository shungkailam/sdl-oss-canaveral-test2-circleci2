package router

import (
	"cloudservices/cloudmgmt/api"
)

func getServiceDomainInfoRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/servicedomainsinfo",
			// swagger:route GET /v1.0/servicedomainsinfo Service_Domain_Info ServiceDomainInfoList
			//
			// Get service domain additional information like artifacts.
			//
			// Retrieves all service domain additional information.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainInfoListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceDomainsInfoW, "/servicedomainsinfo"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomainsinfo/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceDomainsInfoW, "/servicedomainsinfo"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/servicedomainsinfo",
			// swagger:route GET /v1.0/projects/{projectId}/servicedomainsinfo Service_Domain_Info ProjectGetServiceDomainsInfo
			//
			// Get all service domain information for a project as specified by project ID.
			//
			// Retrieves all service domain information for a project as specified by project ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainInfoListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllServiceDomainsInfoForProjectW, "/project-servicedomainsinfo", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomainsinfo/:svcDomainId",
			// swagger:route GET /v1.0/servicedomainsinfo/{svcDomainId} Service_Domain_Info ServiceDomainInfoGet
			//
			// Get all service domain information by service domain ID.
			//
			// Retrieves all service domain additional information for a given service domain ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainInfoGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetServiceDomainInfoW, "/servicedomainsinfo", "svcDomainId"),
		},
		{
			method: "PUT",
			path:   "/v1.0/servicedomainsinfo/:svcDomainId",
			// swagger:route PUT /v1.0/servicedomainsinfo/{svcDomainId} Service_Domain_Info ServiceDomainInfoUpdate
			//
			// Update service domain information by service domain ID.
			//
			// Update service domain additional information for a given service domain ID.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateServiceDomainInfoW, msgSvc, "/servicedomainsinfo", NOTIFICATION_NONE, "svcDomainId"),
		},
	}
}

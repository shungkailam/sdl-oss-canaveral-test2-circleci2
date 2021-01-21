package router

import (
	"cloudservices/cloudmgmt/api"
)

func getServiceDomainRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/servicedomains",
			// swagger:route GET /v1.0/servicedomains Service_Domain ServiceDomainList
			//
			// Get service domains.
			//
			// Retrieves all service domains associated with your account.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceDomainsW, "/servicedomains"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomains/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllServiceDomainsW, "/servicedomains"),
		},
		{
			method: "GET",
			path:   "/v1.0/projects/:projectId/servicedomains",
			// swagger:route GET /v1.0/projects/{projectId}/servicedomains Service_Domain ProjectGetServiceDomains
			//
			// Get all service domains associated with a project by project ID.
			//
			// Retrieves all service domains for a project by project ID {projectId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllServiceDomainsForProjectW, "/projects/:projectId/servicedomains", "projectId"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomains/:svcDomainId/nodes",
			// swagger:route GET /v1.0/servicedomains/{svcDomainId}/nodes Service_Domain ServiceDomainGetNodes
			//
			// Retrieves all nodes for a service domain by service domain ID {svcDomainId}.
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
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllNodesForServiceDomainW, "/servicedomains/:svcDomainId/nodes", "svcDomainId"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomains/:svcDomainId/nodesinfo",
			// swagger:route GET /v1.0/servicedomains/{svcDomainId}/nodesinfo Service_Domain ServiceDomainGetNodesInfo
			//
			// Get nodes info for a service domain by service domain ID.
			//
			// Retrieves all nodes info for a service domain by service domain ID {svcDomainId}.
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
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllNodeInfoForServiceDomainW, "/servicedomains/:svcDomainId/nodesinfo", "svcDomainId"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomains/:svcDomainId",
			// swagger:route GET /v1.0/servicedomains/{svcDomainId} Service_Domain ServiceDomainGet
			//
			// Get a service domain by its ID.
			//
			// Retrieves the service domain with the given ID {svcDomainId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetServiceDomainW, "/servicedomains/:svcDomainId", "svcDomainId"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/servicedomains/:svcDomainId",
			// swagger:route DELETE /v1.0/servicedomains/{svcDomainId} Service_Domain ServiceDomainDelete
			//
			// Delete a service domain as specified by its ID.
			//
			// Deletes the service domain with the given ID  {svcDomainId}.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteServiceDomainW, msgSvc, "servicedomain", NOTIFICATION_EDGE, "svcDomainId"),
		},
		{
			method: "POST",
			path:   "/v1.0/servicedomains",
			// swagger:route POST /v1.0/servicedomains Service_Domain ServiceDomainCreate
			//
			// Create service domain.
			//
			// Create a service domain.
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
			handle: makeCreateHandle(dbAPI, dbAPI.CreateServiceDomainW, msgSvc, "servicedomain", NOTIFICATION_NONE),
		},
		{
			method: "PUT",
			path:   "/v1.0/servicedomains/:svcDomainId",
			// swagger:route PUT /v1.0/servicedomains/{svcDomainId} Service_Domain ServiceDomainUpdate
			//
			// Update a service domain by its ID.
			//
			// Updates a service domain by its ID {svcDomainId}.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateServiceDomainW, msgSvc, "servicedomain", NOTIFICATION_EDGE, "svcDomainId"),
		},
		{
			method: "POST",
			path:   "/v1.0/servicedomainhandle/:svcDomainId",
			// swagger:route POST /v1.0/servicedomainhandle/{svcDomainId} Service_Domain ServiceDomainGetHandle
			//
			// Get service domain certificate. ntnx:ignore
			//
			// Retrieves the certificate and private key for the service domain by its given ID {svcDomainId}.
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
			//       200: ServiceDomainGetHandleResponse
			//       default: APIError
			handle: makeGetHandleNoAuth2(dbAPI, dbAPI.GetServiceDomainHandleW, "/v1.0/servicedomainhandle/:svcDomainId", "svcDomainId"),
		},
		{
			method: "POST",
			path:   "/v1.0/servicedomainsetcertlock",
			// swagger:route POST /v1.0/servicedomainsetcertlock Service_Domain ServiceDomainSetCertLock
			//
			// Set service domain certificate lock. ntnx:ignore
			//
			// Set service domain certificate lock.
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
			//       200: EmptyResponse
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.SetEdgeCertLockW, msgSvc, "servicedomain", NOTIFICATION_NONE),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomains/:svcDomainId/effectiveprofile",
			// swagger:route GET /v1.0/servicedomains/{svcDomainId}/effectiveprofile Service_Domain ServiceDomainGetEffectiveProfile
			//
			// Get a service domain effective profile by ID.
			//
			// Retrieves the service domain effective profile with the given ID {svcDomainId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: ServiceDomainGetEffectiveProfileResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetServiceDomainEffectiveProfileW, "/servicedomains/:svcDomainId/effectiveprofile", "svcDomainId"),
		},
	}
}

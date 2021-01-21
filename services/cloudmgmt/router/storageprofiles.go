package router

import (
	"cloudservices/cloudmgmt/api"
)

func getStorageProfileRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/servicedomains/:svcDomainId/storageprofiles",
			// swagger:route POST /v1.0/servicedomains/{svcDomainId}/storageprofiles Storage_Profile StorageProfileCreate
			//
			// Create a storage profile. ntnx:ignore
			//
			// Create a storage profile on the given service domain ID {svcDomainId}.
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
			handle: makePostHandle3(dbAPI, dbAPI.CreateStorageProfileW, "/v1.0/servicedomains/:svcDomainId/storageprofiles", "svcDomainId"),
		},
		{
			method: "PUT",
			path:   "/v1.0/servicedomains/:svcDomainId/storageprofiles/:id",
			// swagger:route PUT /v1.0/servicedomains/{svcDomainId}/storageprofiles/{id} Storage_Profile StorageProfileUpdate
			//
			// Update storage profile. ntnx:ignore
			//
			// Update the storage profile with {id} on the given service domain ID {svcDomainId}.
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
			handle: makeUpdateHandle3(dbAPI, dbAPI.UpdateStorageProfileW, msgSvc, EntityMessage{"storageprofile", "onUpdateStorageProfile"}, NOTIFICATION_NONE, "svcDomainId", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/servicedomains/:svcDomainId/storageprofiles",
			// swagger:route GET /v1.0/servicedomains/{svcDomainId}/storageprofiles Storage_Profile SvcDomainGetStorageProfiles
			//
			// Get storage profiles according to service domain ID. ntnx:ignore
			//
			// Retrieves all storage profiles for a service domain with a given ID {svcDomainId}
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: StorageProfileListResponse
			//       default: APIError
			handle: makeProjectGetAllHandle(dbAPI, dbAPI.SelectAllStorageProfileForServiceDomainW, "/servicedomains/:svcDomainId/storageprofiles", "svcDomainId"),
		},
	}
}

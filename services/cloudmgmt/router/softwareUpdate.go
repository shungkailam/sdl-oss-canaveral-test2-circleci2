package router

import "cloudservices/cloudmgmt/api"

func getSoftwareUpdateRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/releases",
			// swagger:route GET /v1.0/softwareupdates/releases Software_Update SoftwareReleaseList
			//
			// Get all the available software releases. ntnx:ignore
			//
			// Retrieves all the software releases available.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareReleaseListResponse
			//       default: APIError
			// TODO port from existing API
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSoftwareUpdateReleasesW, "/softwareupdates/releases"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/releases/:release/downloaded-servicedomains",
			// swagger:route GET /v1.0/softwareupdates/releases/{release}/downloaded-servicedomains Software_Update SoftwareDownloadedServiceDomainList
			//
			// Get all service domains that have downloaded available software releases. ntnx:ignore
			//
			// Retrieves all the service domains with the release downloaded.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareDownloadedServiceDomainListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectAllSoftwareDownloadedServiceDomainsW, "/softwareupdates/releases/:release/downloaded-servicedomains", "release"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/downloads",
			// swagger:route GET /v1.0/softwareupdates/downloads Software_Update SoftwareDownloadBatchList
			//
			// Get progress details for all download operations. ntnx:ignore
			//
			// Retrieves progress details about each download operation for all in-progress or completed downloads.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateBatchListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSoftwareDownloadBatchesW, "/softwareupdates/downloads"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/downloads/:batchId",
			// swagger:route GET /v1.0/softwareupdates/downloads/{batchId} Software_Update SoftwareDownloadBatchGet
			//
			// Get the batch in software download phase. ntnx:ignore
			//
			// Retrieves progress details about a download operation as specified by its download batch ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateBatchGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetSoftwareDownloadBatchW, "/softwareupdates/releases/:release/downloaded-servicedomains", "batchId"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/downloads/:batchId/servicedomains",
			// swagger:route GET /v1.0/softwareupdates/downloads/{batchId}/servicedomains Software_Update SoftwareDownloadServiceDomainList
			//
			// Get all the service domains in the software download batch. ntnx:ignore
			//
			// Retrieves all the service domains in the software download batch.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateServiceDomainListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectAllSoftwareDownloadBatchServiceDomainsW, "/softwareupdates/downloads/:batchId/servicedomains", "batchId"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/servicedomains",
			// swagger:route GET /v1.0/softwareupdates/servicedomains Software_Update SoftwareUpdateServiceDomainList
			//
			// Get all the service domains and batches. ntnx:ignore
			//
			// Retrieves all the service domains and batches.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateServiceDomainListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSoftwareUpdateServiceDomainsW, "/softwareupdates/servicedomains"),
		},
		{
			method: "POST",
			path:   "/v1.0/softwareupdates/downloads",
			// swagger:route POST /v1.0/softwareupdates/downloads Software_Update SoftwareDownloadCreate
			//
			// Starts a software download on the selected list of service domains. ntnx:ignore
			//
			// Starts a software download on the selected list of service domains.
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
			handle: makeCreateHandle(dbAPI, dbAPI.StartSoftwareDownloadW, msgSvc, "softwareupdate", NOTIFICATION_EDGE),
		},
		{
			method: "PUT",
			path:   "/v1.0/softwareupdates/downloads/:batchId",
			// swagger:route PUT /v1.0/softwareupdates/downloads/{batchId} Software_Update SoftwareDownloadUpdate
			//
			// Updates an existing software download batch. ntnx:ignore
			//
			// Updates an existing software download batch.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSoftwareDownloadW, msgSvc, "softwareupdate", NOTIFICATION_TENANT, "batchId"),
		},
		{
			method: "PUT",
			path:   "/v1.0/softwareupdates/downloads/:batchId/states",
			// swagger:route PUT /v1.0/softwareupdates/downloads/{batchId}/states Software_Update SoftwareDownloadStateUpdate
			//
			// Updates the state of an existing software download. ntnx:ignore
			//
			// Updates the state of an existing software download operation. The only reportable states are DOWNLOADING, DOWNLOAD_FAILED, DOWNLOAD_CANCELLED and DOWNLOADED.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateStateResponse
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSoftwareDownloadStateW, msgSvc, "softwareupdate", NOTIFICATION_NONE, "batchId"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/upgrades",
			// swagger:route GET /v1.0/softwareupdates/upgrades Software_Update SoftwareUpgradeBatchList
			//
			// Get all the batches in software upgrade phase. ntnx:ignore
			//
			// Retrieves all the batches in software upgrade phase.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateBatchListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllSoftwareUpgradeBatchesW, "/softwareupdates/upgrades"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/upgrades/:batchId",
			// swagger:route GET /v1.0/softwareupdates/upgrades/{batchId} Software_Update SoftwareUpgradeBatchGet
			//
			// Get the batch in software upgrade phase. ntnx:ignore
			//
			// Retrieves the batch in software upgrade phase.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateBatchGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetSoftwareUpgradeBatchW, "/softwareupdates/upgrades/:batchId", "batchId"),
		},
		{
			method: "GET",
			path:   "/v1.0/softwareupdates/upgrades/:batchId/servicedomains",
			// swagger:route GET /v1.0/softwareupdates/upgrades/{batchId}/servicedomains Software_Update SoftwareUpgradeServiceDomainList
			//
			// Get all the service domains in the software upgrade batch. ntnx:ignore
			//
			// Retrieves all the service domains in the software upgrade batch.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateServiceDomainListResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.SelectAllSoftwareUpgradeBatchServiceDomainsW, "/softwareupdates/upgrades/:batchId/servicedomains", "batchId"),
		},
		{
			method: "POST",
			path:   "/v1.0/softwareupdates/upgrades",
			// swagger:route POST /v1.0/softwareupdates/upgrades Software_Update SoftwareUpgradeCreate
			//
			// Starts a software upgrade on the selected list of service domains. ntnx:ignore
			//
			// Starts a software upgrade on the selected list of service domains.
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
			handle: makeCreateHandle(dbAPI, dbAPI.StartSoftwareUpgradeW, msgSvc, "softwareupdate", NOTIFICATION_EDGE),
		},
		{
			method: "PUT",
			path:   "/v1.0/softwareupdates/upgrades/:batchId",
			// swagger:route PUT /v1.0/softwareupdates/upgrades/{batchId} Software_Update SoftwareUpgradeUpdate
			//
			// Updates the state of an existing software upgrade. ntnx:ignore
			//
			// Updates the state of an existing software upgrade.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSoftwareUpgradeW, msgSvc, "softwareupdate", NOTIFICATION_TENANT, "batchId"),
		},
		{
			method: "PUT",
			path:   "/v1.0/softwareupdates/upgrades/:batchId/states",
			// swagger:route PUT /v1.0/softwareupdates/upgrades/{batchId}/states Software_Update SoftwareUpgradeStateUpdate
			//
			// Updates the state of an existing software upgrade. ntnx:ignore
			//
			// Updates the state of an existing software upgrade. The only reportable states are UPDATING, UPDATE_FAILED and UPDATED.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateStateResponse
			//       default: APIError
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateSoftwareUpgradeStateW, msgSvc, "softwareupdate", NOTIFICATION_NONE, "batchId"),
		},
		{
			method: "POST",
			path:   "/v1.0/softwareupdates/credentials",
			// swagger:route POST /v1.0/softwareupdates/credentials Software_Update SoftwareUpdateCredentialsCreate
			//
			// Get credentials to download software update files. ntnx:ignore
			//
			// Retrieves credentials to download software update files.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: SoftwareUpdateCredentialsCreateResponse
			//       default: APIError
			handle: makeCreateHandle(dbAPI, dbAPI.CreateSoftwareDownloadCredentialsW, msgSvc, "softwareupdate", NOTIFICATION_NONE),
		},
	}
}

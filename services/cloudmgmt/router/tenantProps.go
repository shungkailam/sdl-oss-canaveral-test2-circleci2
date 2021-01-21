package router

import (
	"cloudservices/cloudmgmt/api"
)

func getTenantPropsRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/tenantprops/:id",
			// swagger:route GET /v1/tenantprops/{id} TenantPropsGet
			//
			// Get tenant properties by tenant ID. ntnx:ignore
			//
			// Retrieves properties for the tenant with the given ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: TenantPropsGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetTenantPropsW, "/tenantprops/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/tenantprops/:id",
			// swagger:route GET /v1.0/tenantprops/{id} Tenant_Props TenantPropsGetV2
			//
			// Get tenant properties by tenant ID. ntnx:ignore
			//
			// Retrieves properties for the tenant with the given ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: TenantPropsGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetTenantPropsW, "/tenantprops/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/tenantprops/:id",
			// swagger:route DELETE /v1/tenantprops/{id} TenantPropsDelete
			//
			// Delete tenant properties by tenant ID. ntnx:ignore
			//
			// Deletes properties for the tenant with the given ID.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteTenantPropsW, msgSvc, "tenantprops", NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/tenantprops/:id",
			// swagger:route DELETE /v1.0/tenantprops/{id} Tenant_Props TenantPropsDeleteV2
			//
			// Delete tenant properties by tenant ID. ntnx:ignore
			//
			// Deletes properties for the tenant with the given ID.
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
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteTenantPropsWV2, msgSvc, "tenantprops", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1/tenantprops/:id",
			// swagger:route PUT /v1/tenantprops/{id} TenantPropsUpdate
			//
			// Update tenant properties by tenant ID. ntnx:ignore
			//
			// Updates properties for the tenant with the given ID.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateTenantPropsW, msgSvc, "tenantprops", NOTIFICATION_NONE, "id"),
		},
		{
			method: "PUT",
			path:   "/v1.0/tenantprops/:id",
			// swagger:route PUT /v1.0/tenantprops/{id} Tenant_Props TenantPropsUpdateV2
			//
			// Update tenant properties by tenant ID. ntnx:ignore
			//
			// Updates properties for the tenant with the given ID.
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
			handle: makeUpdateHandle(dbAPI, dbAPI.UpdateTenantPropsWV2, msgSvc, "tenantprops", NOTIFICATION_NONE, "id"),
		},
	}
}

package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/auth"
)

// NOTE: do not expose tenant write API via REST
// at least not until we have good operator auth mechanism
func getTenantRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/tenant",
			// swagger:route GET /v1.0/tenant Tenant TenantGet
			//
			// Get tenant info. ntnx:ignore
			//
			// Retrieves metadata for the current tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: TenantGetResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.GetTenantSelfW, "/tenant"),
		},
		{
			method: "GET",
			path:   "/v1.0/tenants/:id",
			// swagger:route GET /v1.0/tenants/{id} Tenant TenantGetByID
			//
			// Get tenant with the ID. ntnx:ignore
			//
			// Gets the tenant with the ID.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: TenantGetResponse
			//       default: APIError

			handle: makeGetHandle(dbAPI, dbAPI.GetTenantW, "/tenants", "id"),

			tenantIDs: []string{auth.OperatorTenantID},

			roles: []string{auth.OperatorTenantRole},
		},
		{
			method: "POST",
			path:   "/v1.0/tenants",
			// swagger:route POST /v1.0/tenants Tenant TenantCreate
			//
			// Create a tenant. ntnx:ignore
			//
			// Creates a tenant by a privileged user
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

			handle: makeCreateHandle(dbAPI, dbAPI.CreateTenantWV2, msgSvc, "/tenants", NOTIFICATION_NONE),

			tenantIDs: []string{auth.OperatorTenantID},

			roles: []string{auth.OperatorTenantRole},
		},
		{
			method: "DELETE",
			path:   "/v1.0/tenants/:id",
			// swagger:route DELETE /v1.0/tenants/{id} Tenant TenantDelete
			//
			// Delete a tenant. ntnx:ignore
			//
			// Deletes a tenant by a privileged user
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

			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteTenantW, msgSvc, "/tenant", NOTIFICATION_NONE, "id"),

			tenantIDs: []string{auth.OperatorTenantID},

			roles: []string{auth.OperatorTenantRole},
		},
	}
}

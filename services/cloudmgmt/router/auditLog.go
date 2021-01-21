package router

import (
	"cloudservices/cloudmgmt/api"
)

func getAuditLogRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/auditlogs",
			// swagger:route GET /v1/auditlogs AuditLogList
			//
			// Lists audit logs. ntnx:ignore
			//
			// Retrieves all audit logs for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: AuditLogListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAuditLogsW, "/auditlogs"),
		},
		{
			method: "GET",
			path:   "/v1.0/auditlogs",
			// swagger:route GET /v1.0/auditlogs Auditlog AuditLogListV2
			//
			// Lists audit logs. ntnx:ignore
			//
			// Retrieves all audit logs for a tenant.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: AuditLogListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAuditLogsW, "/auditlogs"),
		},
		{
			method: "GET",
			path:   "/v1/auditlogs/:id",
			// swagger:route GET /v1/auditlogs/{id} AuditLogGet
			//
			// Get audit log by request ID | date. ntnx:ignore
			//
			// Retrieves the audit log entries for the given request ID or date.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: AuditLogGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetAuditLogW, "/auditlogs/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/auditlogs/:id",
			// swagger:route GET /v1.0/auditlogs/{id} Auditlog AuditLogGetV2
			//
			// Get audit log by request ID or date. ntnx:ignore
			//
			// Retrieves the audit log entries for the given request ID or date.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: AuditLogGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetAuditLogW, "/auditlogs/:id", "id"),
		},
	}
}

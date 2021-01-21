package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"github.com/golang/glog"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func getAuditLogV2Routes(objModelAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	queryAuditLogsV2Handle := getContext(objModelAPI, CheckAuth(objModelAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		glog.V(10).Infoln("queryAuditLogsV2Handle: r.Context() : ", r.Context())
		glog.V(10).Infoln("queryAuditLogsV2Handle: ap : ", ap)
		err := objModelAPI.QueryAuditLogsV2W(r.Context(), w, r)
		handleResponse(w, r, err, "QueryAuditLogsV2, tenantID=%s", ap.TenantID)
	})))
	insertAuditLogV2Handle := getContext(objModelAPI, CheckAuth(objModelAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		glog.V(10).Infoln("insertAuditLogV2Handle: r.Context() : ", r.Context())
		glog.V(10).Infoln("insertAuditLogV2Handle: ap : ", ap)
		err := objModelAPI.InsertAuditLogV2W(r.Context(), w, r)
		handleResponse(w, r, err, "InsertAuditLogV2, tenantID=%s", ap.TenantID)
	})))
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/auditlogsV2",
			// swagger:route POST /v1.0/auditlogsV2 Auditlog QueryAuditLogsV2
			//
			// Lists audit logs matching the provided filter.
			//
			// Retrieves all audit logs matching the filter for a tenant.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: AuditLogV2ListResponse
			//       default: APIError
			handle: queryAuditLogsV2Handle,
		},
		{
			method: "PUT",
			path:   "/v1.0/auditlogsV2",
			// swagger:route PUT /v1.0/auditlogsV2 Auditlog InsertAuditLogV2
			//
			// Inserts Audit logs (used internally). ntnx:ignore
			//
			// This will insert audit log for a tenant.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: AuditLogV2ListResponse
			//       default: APIError
			handle: insertAuditLogV2Handle,
		},
	}
}

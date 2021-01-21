package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func getEventsRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	queryEventsHandle := getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		err := dbAPI.QueryEventsW(r.Context(), w, r)
		handleResponse(w, r, err, "QueryEvents, tenantID=%s", ap.TenantID)
	})))
	upsertEventsHandle := getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		err := dbAPI.UpsertEventsW(r.Context(), w, r.Body, nil)
		handleResponse(w, r, err, "UpsertEvents, tenantID=%s", ap.TenantID)
	})))
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1/events",
			// swagger:route POST /v1/events QueryEvents
			//
			// Lists events. ntnx:ignore
			//
			// Retrieves all events matching the filter for a tenant.
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
			//       200: EventListResponse
			//       default: APIError
			handle: queryEventsHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/events",
			// swagger:route POST /v1.0/events Event QueryEventsV2
			//
			// Lists events matching the provided filter.
			//
			// Retrieves all events matching the filter for a tenant.
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
			//       200: EventListResponse
			//       default: APIError
			handle: queryEventsHandle,
		},
		{
			method: "PUT",
			path:   "/v1/events",
			// swagger:route PUT /v1/events UpsertEvents
			//
			// Upserts events (used internally). ntnx:ignore
			//
			// This will insert/update events for a tenant.
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
			//       200: EventListResponse
			//       default: APIError
			handle: upsertEventsHandle,
		},
		{
			method: "PUT",
			path:   "/v1.0/events",
			// swagger:route PUT /v1.0/events Event UpsertEventsV2
			//
			// Upserts events (used internally). ntnx:ignore
			//
			// This will insert/update events for a tenant.
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
			//       200: EventListResponse
			//       default: APIError
			handle: upsertEventsHandle,
		},
	}
}

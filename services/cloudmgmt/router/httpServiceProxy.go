package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
)

func getHTTPServiceProxyRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	// Note: we are piggy backing on wstun for ws callback
	CreateHTTPServiceProxyHandle := makeCustomMessageHandle(dbAPI, dbAPI.CreateHTTPServiceProxyW, msgSvc, "setupSSHTunneling", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.WstunPayload)
		return &payload.ServiceDomainID
	})
	UpdateHTTPServiceProxyHandle := makeCustomMessageHandle(dbAPI, dbAPI.UpdateHTTPServiceProxyW, msgSvc, "setupSSHTunneling", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.WstunPayload)
		return &payload.ServiceDomainID
	})
	DeleteHTTPServiceProxyHandle := makeCustomMessageHandle(dbAPI, dbAPI.DeleteHTTPServiceProxyW, msgSvc, "teardownSSHTunneling", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.WstunTeardownRequest)
		return &payload.ServiceDomainID
	})

	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/httpserviceproxies",
			// swagger:route GET /v1.0/httpserviceproxies HTTPServiceProxy HTTPServiceProxyList
			//
			// Get all HTTP service proxies.
			//
			// Retrieves a list of all HTTP service proxies.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: HTTPServiceProxyListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllHTTPServiceProxiesW, "/httpserviceproxies"),
		},
		{
			method: "GET",
			path:   "/v1.0/httpserviceproxies/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllHTTPServiceProxiesW, "/httpserviceproxies"),
		},
		{
			method: "GET",
			path:   "/v1.0/httpserviceproxies/:id",
			// swagger:route GET /v1.0/httpserviceproxies/{id} HTTPServiceProxy HTTPServiceProxyGet
			//
			// Get a HTTP service proxy by its ID.
			//
			// Retrieves a HTTP service proxy with the given ID {id}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: HTTPServiceProxyGetResponse
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetHTTPServiceProxyW, "/httpserviceproxies/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/httpserviceproxies/:id",
			// swagger:route DELETE /v1.0/httpserviceproxies/{id} HTTPServiceProxy HTTPServiceProxyDelete
			//
			// Delete HTTP service proxy.
			//
			// Delete the HTTP service proxy with the given ID {id}.
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
			handle: DeleteHTTPServiceProxyHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/httpserviceproxies",
			// swagger:route POST /v1.0/httpserviceproxies HTTPServiceProxy HTTPServiceProxyCreate
			//
			// Create a HTTP service proxy.
			//
			// Create a HTTP service proxy.
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
			//       200: HTTPServiceProxyCreateResponse
			//       default: APIError
			handle: CreateHTTPServiceProxyHandle,
		},
		{
			method: "PUT",
			path:   "/v1.0/httpserviceproxies/:id",
			// swagger:route PUT /v1.0/httpserviceproxies/{id} HTTPServiceProxy HTTPServiceProxyUpdate
			//
			// Update a HTTP service proxy by its ID.
			//
			// Update a HTTP service proxy with the given ID {id}.
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
			//       200: HTTPServiceProxyUpdateResponse
			//       default: APIError
			handle: UpdateHTTPServiceProxyHandle,
		},
	}
}

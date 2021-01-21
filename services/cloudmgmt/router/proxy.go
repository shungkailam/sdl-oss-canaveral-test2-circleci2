package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

const (
	PROXY_HEADER_PREFIX = "X-Ntnx-Xks-"
	phpLen              = len(PROXY_HEADER_PREFIX)
)

func ParseEdgeIDAndURL(path, fullURL string) (edgeID, url string, err error) {
	// path format should be:
	// /http[s]/<edge id>/<path to svc>
	protocol := ""
	err = fmt.Errorf("Bad path: %s, fullURL: %s", path, fullURL)
	if strings.HasPrefix(path, "/https/") {
		protocol = "https"
	} else if strings.HasPrefix(path, "/http/") {
		protocol = "http"
	}
	if len(protocol) == 0 {
		return
	}
	p := path[len(protocol)+2:]
	i := strings.Index(p, "/")
	if i <= 0 {
		return
	}
	edgeID = p[:i]
	pathSuffix := p[i+1:]
	j := strings.Index(fullURL, pathSuffix)
	if j > 0 {
		// use full URL suffix to pick up query parameters if any
		pathSuffix = fullURL[j:]
	}
	url = fmt.Sprintf("%s://%s", protocol, pathSuffix)
	err = nil
	return
}

func rewriteHeaders(ctx context.Context, req *http.Request) {
	h := req.Header
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		if k == "Authorization" {
			// drop cloudmgmt authorization
			continue
		} else if strings.HasPrefix(k, PROXY_HEADER_PREFIX) {
			// strip proxy header prefix
			h2[k[phpLen:]] = vv
		} else {
			h2[k] = vv
		}
	}
	glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy rewrite headers done: %+v to %+v"), h, h2)
	req.Header = h2
}

func makeProxyHandle(dbAPI api.ObjectModelAPI,
	msgSvc api.WSMessagingService,
	msg string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")
		path := ps.ByName("path")
		glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: path: %s, request: %+v"), path, r)

		var err error
		// use for/break as structured goto
		for {
			// RBAC
			// TODO: allow operator to impersonate tenant:
			// if operator, fetch tenant id for edge
			if !auth.IsOperatorRole(ap) && !auth.IsInfraAdminRole(ap) {
				err = fmt.Errorf("Permission denied: tid|uid=%s|%s", ap.TenantID, ap.ID)
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: RBAC error: %s"), err)
				break
			}
			edgeID, u, err2 := ParseEdgeIDAndURL(path, r.URL.String())
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: path error: %s"), err)
				break
			}
			ul, err2 := url.Parse(u)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: url[%s] error: %s"), u, err)
				break
			}
			r.URL = ul
			rewriteHeaders(ctx, r)
			resp, err2 := msgSvc.SendHTTPRequest(ctx, ap.TenantID, edgeID, r, u)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: send req[%+v], url[%s], error: %s"), r, u, err)
				break
			}
			header := w.Header()
			for k, v := range resp.Header {
				header[k] = v
			}
			glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: Response: %+v"), *resp)
			baResp, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: req[%+v], response error: %s"), r, err)
				break
			}
			defer resp.Body.Close()

			// write response status code
			w.WriteHeader(resp.StatusCode)

			_, err2 = w.Write(baResp)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: req[%+v], write error: %s"), r, err)
				break
			}
			break
		}
		handleResponse(w, r, err, "PROXY %s, tenantID=%s", msg, ap.TenantID)
	}))
}

func getProxyRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {

	ProxyHandle := makeProxyHandle(dbAPI, msgSvc, "httpProxy")

	return []routeHandle{
		{
			method: "POST",
			path:   "/v1.0/proxy/*path",
			// swagger:route POST /v1.0/proxy/*path Proxy ProxyPostCall
			//
			// Proxy HTTP API. ntnx:ignore
			//
			// Proxy HTTP API over websocket to Service Domain.
			// The path parameter should be of the form:
			//    http[s]/:svc_domain_id/path_of_http_service
			// The payload will be passed on to Service Domain.
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
			//       200: ProxyCallResponse
			//       default: APIError
			handle: ProxyHandle,
		},
		{
			method: "PUT",
			path:   "/v1.0/proxy/*path",
			// swagger:route PUT /v1.0/proxy/*path Proxy ProxyPutCall
			//
			// Proxy HTTP API. ntnx:ignore
			//
			// Proxy HTTP API over websocket to Service Domain.
			// The path parameter should be of the form:
			//    http[s]/:svc_domain_id/path_of_http_service
			// The payload will be passed on to Service Domain.
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
			//       200: ProxyCallResponse
			//       default: APIError
			handle: ProxyHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/proxy/*path",
			// swagger:route GET /v1.0/proxy/*path Proxy ProxyGetCall
			//
			// Proxy HTTP API. ntnx:ignore
			//
			// Proxy HTTP API over websocket to Service Domain.
			// The path parameter should be of the form:
			//    http[s]/:svc_domain_id/path_of_http_service
			// The payload will be passed on to Service Domain.
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
			//       200: ProxyCallResponse
			//       default: APIError
			handle: ProxyHandle,
		},
		{
			method: "DELETE",
			path:   "/v1.0/proxy/*path",
			// swagger:route DELETE /v1.0/proxy/*path Proxy ProxyDeleteCall
			//
			// Proxy HTTP API. ntnx:ignore
			//
			// Proxy HTTP API over websocket to Service Domain.
			// The path parameter should be of the form:
			//    http[s]/:svc_domain_id/path_of_http_service
			// The payload will be passed on to Service Domain.
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
			//       200: ProxyCallResponse
			//       default: APIError
			handle: ProxyHandle,
		},
	}
}

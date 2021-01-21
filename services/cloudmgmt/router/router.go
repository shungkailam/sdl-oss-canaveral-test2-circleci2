package router

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/cloudmgmt/spa"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	tenantpoolcore "cloudservices/tenantpool/core"
	tenantpoolmodel "cloudservices/tenantpool/model"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"cloudservices/common/metrics"

	"github.com/go-redis/redis"
	"github.com/prometheus/client_golang/prometheus"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	funk "github.com/thoas/go-funk"
	"github.com/xi2/httpgzip"
)

const (
	NOTIFICATION_NONE            = iota
	NOTIFICATION_TENANT          = iota
	NOTIFICATION_EDGE            = iota
	NOTIFICATION_EDGE_SYNC       = iota
	EntityTypeApplication        = "application"
	EntityTypeCategory           = "category"
	EntityTypeCloudCreds         = "cloudcreds"
	EntityTypeDataSource         = "datasource"
	EntityTypeDataStream         = "datastream"
	EntityTypeDockerProfile      = "dockerprofile"
	EntityTypeEdge               = "edge"
	EntityTypeEdgeDevice         = "edgedevice"
	EntityTypeEdgeCluster        = "edgecluster"
	EntityTypeServiceDomain      = "servicedomain"
	EntityTypeNode               = "node"
	EntityTypeProject            = "project"
	EntityTypeScript             = "script"
	EntityTypeScriptRuntime      = "scriptruntime"
	EntityTypeSensor             = "sensor"
	EntityTypeUser               = "user"
	EntityTypeExecuteEdgeUpgrade = "executeEdgeUpgrade"
	EntityTypeMLModel            = "mlmodel"
	EntityTypeProjectService     = "projectservice"
	EntityTypeSoftwareUpdate     = "softwareupdate"
	EntityTypeLogCollector       = "logcollector"
	EntityTypeServiceInstance    = "serviceinstance"
	EntityTypeServiceBinding     = "servicebinding"
	EntityTypeDataDriverClass    = "datadriverclass"
	EntityTypeDataDriverInstance = "datadriverinstance"
	EntityTypeDataDriverConfig   = "datadriverconfig"
	EntityTypeDataDriverStream   = "datadriverstream"

	CanonicalTypeApplication   = "Application"
	CanonicalTypeCategory      = "Category"
	CanonicalTypeCloudCreds    = "CloudCreds"
	CanonicalTypeDataSource    = "DataSource"
	CanonicalTypeDataStream    = "DataStream"
	CanonicalTypeDockerProfile = "DockerProfile"
	CanonicalTypeEdge          = "Edge"
	CanonicalTypeEdgeDevice    = "EdgeDevice"
	// TODO: if we send edgeCluster messages we can maybe change this to edge
	// and not need backed changes on the edge side also for backward compatibility
	// CanonicalTypeEdgeCluster     = "Edge"
	CanonicalTypeEdgeCluster        = "EdgeCluster"
	CanonicalTypeServiceDomain      = "ServiceDomain"
	CanonicalTypeNode               = "Node"
	CanonicalTypeProject            = "Project"
	CanonicalTypeScript             = "Script"
	CanonicalTypeScriptRuntime      = "ScriptRuntime"
	CanonicalTypeSensor             = "Sensor"
	CanonicalTypeUser               = "User"
	CanonicalTypeExecuteEdgeUpgrade = "ExecuteEdgeUpgrade"
	CanonicalTypeMLModel            = "MLModel"
	CanonicalTypeProjectService     = "ProjectService"
	CanonicalTypeSoftwareUpdate     = "SoftwareUpdate"
	CanonicalTypeLogCollector       = "LogCollector"
	CanonicalTypeServiceInstance    = "ServiceInstance"
	CanonicalTypeServiceBinding     = "ServiceBinding"
	CanonicalTypeDataDriverInstance = "DataDriverInstance"
	CanonicalTypeDataDriverConfig   = "DataDriverConfig"

	CreateUpdateOpType = "create/update"
	DeleteOpType       = "delete"

	LogPayloadMaxFields    = 50
	DefaultPayloadMaxBytes = 1 << 20
	MLPayloadMaxBytes      = 1 << 30
)

var (
	// RouteRoleValidator is the role validator for routes
	RouteRoleValidator *service.RouteRoleValidator

	LogPayloadRedactKeys = []string{"password", "pwd", "credentials", "gcpCredential", "awsCredential"}
	RegexMLModelPath     = regexp.MustCompile(".*/mlmodels/[^/]+/versions.*")
)

type EntityMessage struct {
	EntityType string
	Message    string
}

// Custom renderer for types with edge-specific messages on broadcast
type Renderer interface {
	RenderForEdge(dbAPI api.ObjectModelAPI, tenantID, edgeID string) (interface{}, error)
}

type routeHandle struct {
	method    string
	path      string
	tenantIDs []string // Allowed tenant IDs. Empty or nil means any tenant
	roles     []string // Allowed roles. If it is not set, all the default roles work
	handle    httprouter.Handle
}

type responseWriterWrapper struct {
	http.ResponseWriter
	status  int
	capture bool
	buffer  *bytes.Buffer
	length  int
}

func newResponseWriterWrapper(w http.ResponseWriter, capture bool) *responseWriterWrapper {
	return &responseWriterWrapper{w, 200, capture, &bytes.Buffer{}, 0}
}

func (rec *responseWriterWrapper) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
	if code != 200 {
		rec.capture = true
	}
}
func (rec *responseWriterWrapper) Write(b []byte) (int, error) {
	if rec.capture {
		rec.buffer.Write(b)
	}
	rec.length += len(b)
	return rec.ResponseWriter.Write(b)
}

// gunzipBuffer - given a gzip buffer,
// return a new buffer which is gunzipped
func gunzipBuffer(buf *bytes.Buffer) (*bytes.Buffer, error) {
	var buf2 bytes.Buffer
	zr, err := gzip.NewReader(buf)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(&buf2, zr); err != nil {
		return nil, err
	}
	if err := zr.Close(); err != nil {
		return nil, err
	}
	return &buf2, nil
}

// isGzipBuffer - check buffer for gzip magic header
// will not advance buffer
// see http://www.onicos.com/staff/iz/formats/gzip.html
func isGzipBuffer(buf *bytes.Buffer) bool {
	n := buf.Len()
	if n < 3 {
		return false
	}
	ba := buf.Bytes()
	return ba[0] == 31 && ba[1] == 139
}

func getContext(dbAPI api.ObjectModelAPI, handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		writeAuditLogOn := false == *config.Cfg.DisableAuditLog
		writeAuditLogReadReqOn := *config.Cfg.EnableAuditLogOfReadReq
		writeAuditLogPutEventOn := *config.Cfg.EnableAuditLogOfPutEvent
		isGetReq := r.Method == "GET"
		isPostEvent := r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/events")
		isPutEvent := r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/events")
		isHelmTemplate := r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/helm/template")
		// If writeAuditLogPutEventOn is false (=default), treat PutEvent as Read Req and don't write it to audit log
		isReadReq := isGetReq || isPostEvent || (isPutEvent && !writeAuditLogPutEventOn)
		isPostMLModelVersion := RegexMLModelPath.MatchString(r.URL.Path)
		// POST edgehandle response contains edge private key, so don't capture it
		isSkipCapture := strings.Contains(r.URL.Path, "/edgehandle/")
		captureResponse := false == isReadReq && false == isSkipCapture
		shouldWriteAuditLog := writeAuditLogOn && (!isReadReq || writeAuditLogReadReqOn)
		var auditLog *model.AuditLog
		var w2 *responseWriterWrapper
		var auditLogTableName string
		if r.Body != nil {
			// limit request body size to prevent clients from accidentally or maliciously
			// sending a large request and wasting server resources.
			if isPostMLModelVersion {
				r.Body = http.MaxBytesReader(w, r.Body, MLPayloadMaxBytes)
			} else {
				r.Body = http.MaxBytesReader(w, r.Body, DefaultPayloadMaxBytes)
			}
		}
		if shouldWriteAuditLog {
			auditLog = model.NewAuditLogFromRequest(r)
			w2 = newResponseWriterWrapper(w, captureResponse)
			auditLogTableName = api.GetAuditLogTableName()
		}
		ctx := r.Context()
		if glog.V(5) {
			glog.V(5).Infof("Received context: %+v", ctx)
		}
		reqID := r.Header.Get("X-Request-ID")
		if len(reqID) == 0 {
			reqID = base.GetUUID()
		}
		ctx = context.WithValue(ctx, base.RequestIDKey, reqID)
		if shouldWriteAuditLog {
			auditLog.RequestID = reqID
		}

		// Note: the following block of code read the entire request body in memory
		// This is usually okay as we don't expect the request body to be large.
		// However, when the request body could be large, we must avoid this.
		// For example, we skip this for upload new ML model version binary.
		if r.Method != "GET" && r.Body != nil && !isPostMLModelVersion {
			// Disable logging for PUT events at default log level
			shouldLogPayload := (!isPutEvent && !isHelmTemplate && bool(glog.V(3))) || bool(glog.V(4))
			// It is true if at least logging or audit is enabled
			shouldReadPayload := shouldLogPayload || shouldWriteAuditLog
			var payload string
			body, err := ioutil.ReadAll(r.Body)
			// Restore the io.ReadCloser to its original state
			r.Body = ioutil.NopCloser(bytes.NewBuffer(body))
			if err == nil {
				if !shouldLogPayload {
					glog.V(3).Infof(base.PrefixRequestID(ctx, "Received for URI: %s, Method: %s, Params: %+v, Payload size: %d"), r.RequestURI, r.Method, ps, len(body))
				}
				if shouldReadPayload {
					// If RedactJSON fails, it returns the JSON string which is passed to it
					payload = base.RedactJSON(string(body), int(LogPayloadMaxFields), func(property string) bool {
						return funk.Contains(LogPayloadRedactKeys, property)
					})
					if shouldWriteAuditLog {
						auditLog.RequestPayload = &payload
					}
					if shouldLogPayload {
						glog.V(3).Infof(base.PrefixRequestID(ctx, "Received for URI: %s, Method: %s, Params: %+v, Payload size: %d, Body: %s"), r.RequestURI, r.Method, ps, len(body), payload)
					}
				}
			}
		} else {
			glog.V(4).Infof(base.PrefixRequestID(ctx, "Received for URI: %s, Method: %s, Params: %+v"), r.RequestURI, r.Method, ps)
		}
		start := time.Now()
		defer func() {
			stop := time.Since(start)
			var verbose glog.Verbose
			if r.Method == "GET" {
				verbose = glog.V(4)
			} else {
				verbose = glog.V(3)
			}
			if shouldWriteAuditLog {
				// fill in audit log response
				auditLog.ResponseCode = w2.status
				auditLog.ResponseLength = w2.length
				var s string
				if w2.capture {
					if isGzipBuffer(w2.buffer) {
						buf2, err := gunzipBuffer(w2.buffer)
						if err == nil {
							s = buf2.String()
						}
					} else {
						s = w2.buffer.String()
					}
					auditLog.ResponseMessage = &s
				}
				if auditLog.RequestMethod == "POST" {
					if strings.HasSuffix(auditLog.RequestURL, "/login") {
						// login
						// capture identify from successful login
						if auditLog.ResponseCode == 200 {
							// parse s as json
							m := map[string]string{}
							err := json.Unmarshal([]byte(s), &m)
							if err == nil {
								token := m["token"]
								if token != "" {
									model.UpdateAuditLogFromToken(auditLog, token)
								}
							}
						} else {
							// don't log login failure to avoid deny of service attack (DoS)
							// rely on log in ES if want to find login failure info / stats
							shouldWriteAuditLog = false
						}
					} else if strings.HasSuffix(auditLog.RequestURL, "/edgebyserialnumber") ||
						strings.Contains(auditLog.RequestURL, "/edgehandle/") {
						// only log successful edgebyserialnumber / edgehandle calls (again, to avoid DoS)
						shouldWriteAuditLog = auditLog.ResponseCode == 200
					} else {
						// don't write audit log if 401 error to avoid DoS
						shouldWriteAuditLog = auditLog.ResponseCode != 401
					}
				} else {
					// don't write audit log if 401 error to avoid DoS
					shouldWriteAuditLog = auditLog.ResponseCode != 401
				}
			}
			verbose.Infof(base.PrefixRequestID(ctx, "Completed for URI: %s, Method: %s, Params: %+v in %.2f ms"), r.RequestURI, r.Method, ps, float32(stop/time.Millisecond))
			// write audit log out-of-band
			if shouldWriteAuditLog {
				go func() {
					// at this point ctx will not have authContext,
					// so just pass in admin ctx, as write to DB needs it
					ctx2 := base.GetAdminContext(auditLog.RequestID, auditLog.TenantID)
					ctx2 = context.WithValue(ctx2, base.AuditLogTableNameKey, auditLogTableName)
					err := dbAPI.WriteAuditLog(ctx2, auditLog)
					if err != nil {
						glog.Warningf(base.PrefixRequestID(ctx, "Failed to write audit log: %+v, err: %s"), *auditLog, err.Error())
					}
				}()
			}
		}()
		bytes, _ := json.Marshal(ps)
		httpRequestContext := base.HTTPRequestContext{URI: r.RequestURI, Method: r.Method, Params: string(bytes)}
		ctx = context.WithValue(ctx, base.HTTPRequestContextKey, httpRequestContext)
		if shouldWriteAuditLog {
			ctx = context.WithValue(ctx, base.AuditLogTableNameKey, auditLogTableName)
			w2.Header().Set("X-Request-ID", reqID)
			handle(w2, r.WithContext(ctx), ps)
		} else {
			w.Header().Set("X-Request-ID", reqID)
			handle(w, r.WithContext(ctx), ps)
		}
	}
}

func handleResponse(w http.ResponseWriter, r *http.Request, err error, format string, args ...interface{}) {
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	ctx := r.Context()
	if err == nil {
		var verbose glog.Verbose
		if r.Method == "GET" {
			verbose = glog.V(4)
		} else {
			verbose = glog.V(3)
		}
		verbose.Infof(base.PrefixRequestID(ctx, "Response: %s"), msg)
	} else {
		writeErrorResponse(ctx, w, err, msg)
	}
}

func writeErrorResponse(ctx context.Context, w http.ResponseWriter, err error, msg string) {
	errCode, ok := err.(errcode.ErrorCode)
	glog.Errorf(base.PrefixRequestID(ctx, "Response: %s, Error: %s"), msg, err.Error())
	w.Header().Set("Content-Type", "application/json")
	if ok {
		uiMsg, err := errCode.GetUIErrorMessage("en_US")
		if err != nil {
			uiMsg = "Unable to get the UI error message"
		}
		w.WriteHeader(errCode.GetHTTPStatus())
		fmt.Fprintf(w, `{"statusCode": %d, "errorCode": %d, "message": "%s"}`, errCode.GetHTTPStatus(), errCode.GetCode(), uiMsg)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"statusCode": 500, "errorCode": 0, "message": "%s"}`, msg)
	}
}

func make404Handler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, `{"statusCode": 404, "errorCode": 0, "message": "not found"}`)
	}
}

// we use httpgzip package for gzip support,
// see: https://blog.klauspost.com/gzip-performance-for-go-webservers/
// convert httprouter.Handle to http.Handler, use closure to capture extra parameters
func handleToHandler(handle httprouter.Handle, ps httprouter.Params) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, ps)
	})
}

// convert auth.ContextHandler to http.Handler, use closure to capture extra parameters
func authHandleToHandler(handle auth.ContextHandler, ps httprouter.Params, ap *base.AuthContext) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handle(w, r, ps, ap)
	})
}

// use httpgzip to add gzip functionality to auth.ContextHandler
func getAuthGzipHandle(handle auth.ContextHandler) auth.ContextHandler {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		handler := authHandleToHandler(handle, ps, ap)
		gzipHandler := httpgzip.NewHandler(handler, nil)

		gzipHandler.ServeHTTP(w, r)
	}
}

// use httpgzip to add gzip functionality to httprouter.Handle
func getGzipHandle(handle httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		handler := handleToHandler(handle, ps)
		gzipHandler := httpgzip.NewHandler(handler, nil)
		gzipHandler.ServeHTTP(w, r)
	}
}

func panicHandler(w http.ResponseWriter, r *http.Request, p interface{}) {
	glog.Errorf("panic handler called, url path: %s\n", r.URL.Path)
	debug.PrintStack()
	if strings.HasPrefix(r.URL.Path, "/v1/") {
		w.Header().Set("Content-Type", "application/json")
		handleResponse(w, r, errors.New("Internal server error"), "Panicked!")
	} else {
		w.Header().Set("Content-Type", "application/json")
		handleResponse(w, r, errors.New("Internal server error 2"), "Panicked!")
	}
}

func addRouterHandle(router *httprouter.Router, method string, path string, handle httprouter.Handle) {
	router.Handle(method, path, func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		start := time.Now()
		handle(w, r, params)
		elapsed := time.Since(start).Seconds()
		metrics.RESTAPITime.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": method, "path": path}).Observe(elapsed)
	})
}

// ConfigureRouter sets up all REST API endpoints
// it also sets up serving of UI static files and
// redirect 404 to /index.html for SPA support
func ConfigureRouter(dbAPI api.ObjectModelAPI, router *httprouter.Router, redisClient *redis.Client, msgSvc api.WSMessagingService, contentDir string) {
	routesList := [][]routeHandle{
		getEdgeRoutes(dbAPI, msgSvc),
		getCategoryRoutes(dbAPI, msgSvc),
		getCertificatesRoutes(dbAPI, msgSvc),
		getCloudCredsRoutes(dbAPI, msgSvc),
		getDataSourceRoutes(dbAPI, msgSvc),
		getDataStreamRoutes(dbAPI, msgSvc),
		getProjectRoutes(dbAPI, msgSvc),
		getScriptRoutes(dbAPI, msgSvc),
		getScriptRuntimeRoutes(dbAPI, msgSvc),
		getSensorRoutes(dbAPI, msgSvc),
		getApplicationRoutes(dbAPI, msgSvc),
		getKubernetesClusterRoutes(dbAPI, msgSvc),
		getApplicationStatusRoutes(dbAPI, msgSvc),
		getDockerProfileRoutes(dbAPI, msgSvc),
		getUserRoutes(dbAPI, msgSvc),
		getAggregateRoutes(dbAPI),
		getLoginRoutes(dbAPI, redisClient),
		getLogRoutes(dbAPI, msgSvc),
		getLogStreamRoutes(dbAPI, msgSvc),
		getLogCollectorRoutes(dbAPI, msgSvc),
		getEdgeInfoRoutes(dbAPI, msgSvc),
		getWebSocketRoutes(dbAPI, msgSvc),
		getEventsRoutes(dbAPI, msgSvc),
		getEdgeUpgradeRoutes(dbAPI, msgSvc),
		getContainerRegistryProfileRoutes(dbAPI, msgSvc),
		getWstunRoutes(dbAPI, msgSvc),
		getUserPropsRoutes(dbAPI, msgSvc),
		getTenantPropsRoutes(dbAPI, msgSvc),
		getAuditLogRoutes(dbAPI, msgSvc),
		getAuditLogV2Routes(dbAPI, msgSvc),
		getMLModelRoutes(dbAPI, msgSvc),
		getEdgeInventoryDeltaRoutes(dbAPI, msgSvc),
		getMLModelStatusRoutes(dbAPI, msgSvc),
		getInfraConfigRoutes(dbAPI, msgSvc),
		getUserPublicKeyRoutes(dbAPI, msgSvc),
		getUserApiTokenRoutes(dbAPI, msgSvc),
		getServiceRoutes(dbAPI, msgSvc),
		getProjectServiceRoutes(dbAPI, msgSvc),
		getServiceDomainRoutes(dbAPI, msgSvc),
		getNodeRoutes(dbAPI, msgSvc),
		getNodeInfoRoutes(dbAPI, msgSvc),
		getServiceDomainInfoRoutes(dbAPI, msgSvc),
		getSoftwareUpdateRoutes(dbAPI, msgSvc),
		getTenantRoutes(dbAPI, msgSvc),
		getHelmRoutes(dbAPI, msgSvc),
		getStorageProfileRoutes(dbAPI, msgSvc),
		getServiceClassRoutes(dbAPI, msgSvc),
		getServiceInstanceRoutes(dbAPI, msgSvc),
		getServiceBindingRoutes(dbAPI, msgSvc),
		getProxyRoutes(dbAPI, msgSvc),
		getHTTPServiceProxyRoutes(dbAPI, msgSvc),
		getKialiProxyRoutes(dbAPI, msgSvc),
		getK8sDashboardRoutes(dbAPI, msgSvc),
		getDataDriverClassRoutes(dbAPI, msgSvc),
		getDataDriverInstanceRoutes(dbAPI, msgSvc),
		getDataDriverConfigRoutes(dbAPI, msgSvc),
		getDataDriverStreamRoutes(dbAPI, msgSvc),
	}
	RouteRoleValidator = service.NewRouteRoleValidator(router)
	for _, routes := range routesList {
		for _, route := range routes {
			addRouterHandle(router, route.method, route.path, route.handle)
			RouteRoleValidator.SetRouteRoles(route.method, route.path, route.tenantIDs, route.roles)
		}
	}

	// Add ping handler for health check
	service.AddPingHandler(router)
	// Add log level change handler
	service.AddLogLevelHandler(router)

	fileServer := &FileServer{DBAPI: dbAPI}
	// serve static files + SPA support
	router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// First serve with the file server
		if fileServer.ServeFileRequest(w, r) == ErrUnhandled {
			// Fallback to static file handler
			httpgzip.NewHandler(&spa.SPAHandler{ContentDir: contentDir}, nil).ServeHTTP(w, r)
		}
	})
	router.PanicHandler = panicHandler

}

func makeGetAllHandle(dbAPI api.ObjectModelAPI, getAllFn func(context.Context, io.Writer, *http.Request) error, path string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		err := getAllFn(r.Context(), w, r)
		handleResponse(w, r, err, "GET all %s, tenantID=%s", path, ap.TenantID)
	})))
}

func makeEdgeGetAllHandle(dbAPI api.ObjectModelAPI, getAllFn func(context.Context, string, io.Writer, *http.Request) error, path string, paramName string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		edgeID := ps.ByName(paramName)
		err := getAllFn(r.Context(), edgeID, w, r)
		handleResponse(w, r, err, "GET all %s, tenantID=%s, edgeID=%s", path, ap.TenantID, edgeID)
	})))
}

func makeProjectGetAllHandle(dbAPI api.ObjectModelAPI, getAllFn func(context.Context, string, io.Writer, *http.Request) error, path string, paramName string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		projectID := ps.ByName(paramName)
		err := getAllFn(r.Context(), projectID, w, r)
		handleResponse(w, r, err, "GET all %s, tenantID=%s, projectID=%s", path, ap.TenantID, projectID)
	})))
}

func makeGetAllHandleNoAuth(dbAPI api.ObjectModelAPI, getAllFn func(context.Context, io.Writer, *http.Request) error, path string) httprouter.Handle {
	return getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		err := getAllFn(r.Context(), w, r)
		handleResponse(w, r, err, "GET all no auth %s, tenantID=%s", path, "NOT_SET")
	})
}

func makeGetHandle(dbAPI api.ObjectModelAPI, getFn func(context.Context, string, io.Writer, *http.Request) error, path string, paramName string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		id := ps.ByName(paramName)
		err := getFn(r.Context(), id, w, r)
		handleResponse(w, r, err, "GET %s, tenantID=%s, id=%s", path, ap.TenantID, id)
	})))
}

func makeGetHandle2(dbAPI api.ObjectModelAPI, getFn func(context.Context, string, int, io.Writer, *http.Request) error, path string, paramName string, paramName2 string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		id := ps.ByName(paramName)
		p2, _ := strconv.Atoi(ps.ByName(paramName2))
		err := getFn(r.Context(), id, p2, w, r)
		handleResponse(w, r, err, "GET %s, tenantID=%s, id=%s, p2=%d", path, ap.TenantID, id, p2)
	})))
}

func makeGetHandleNoAuth(dbAPI api.ObjectModelAPI, getFn func(context.Context, io.Writer, *http.Request) error, path string) httprouter.Handle {
	return getContext(dbAPI, getGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		err := getFn(r.Context(), w, r)
		handleResponse(w, r, err, "GET %s", path)
	}))
}

func makeGetHandleNoAuth2(dbAPI api.ObjectModelAPI, getFn func(context.Context, string, io.Writer, *http.Request) error, path string, paramName string) httprouter.Handle {
	return getContext(dbAPI, getGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		id := ps.ByName(paramName)
		err := getFn(r.Context(), id, w, r)
		handleResponse(w, r, err, "GET %s", path)
	}))
}

func makePostHandleNoAuth(dbAPI api.ObjectModelAPI, postFn func(context.Context, io.Writer, *http.Request) error, path string) httprouter.Handle {
	return getContext(dbAPI, getGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		err := postFn(r.Context(), w, r)
		handleResponse(w, r, err, "POST %s", path)
	}))
}

func makePostHandleNoAuth2(dbAPI api.ObjectModelAPI, postFn func(context.Context, string, io.Writer, *http.Request) error, path string, paramName string) httprouter.Handle {
	return getContext(dbAPI, getGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		id := ps.ByName(paramName)
		err := postFn(r.Context(), id, w, r)
		handleResponse(w, r, err, "POST %s", path)
	}))
}

func makePostHandle2(dbAPI api.ObjectModelAPI, postFn func(context.Context, string, string, io.Writer, *http.Request) error, path, paramName1, paramName2 string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		p1 := ps.ByName(paramName1)
		p2 := ps.ByName(paramName2)
		err := postFn(r.Context(), p1, p2, w, r)
		handleResponse(w, r, err, "POST %s, tenantID=%s, p1=%s, p2=%s", path, ap.TenantID, p1, p2)
	})))
}
func makePostHandle3(dbAPI api.ObjectModelAPI, postFn func(context.Context, string, io.Writer, *http.Request) error, path, paramName string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		p := ps.ByName(paramName)
		err := postFn(r.Context(), p, w, r)
		handleResponse(w, r, err, "POST %s, tenantID=%s, p=%s", path, ap.TenantID, p)
	})))
}

func makeGetHandle2WithWSCallback(dbAPI api.ObjectModelAPI,
	getFn func(context.Context, string, string, io.Writer, func(context.Context, interface{}) (string, error)) error,
	paramName string,
	paramName2 string,
	msgSvc api.WSMessagingService,
	msg string,
	notificationType int,
	edgeIDResolver func(interface{}) *string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		var fn func(context.Context, interface{}) (string, error)
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) (string, error) {
				reqID := base.GetRequestID(ctx)
				req := api.ObjectRequest{
					RequestID: reqID,
					TenantID:  ap.TenantID,
					Doc:       doc,
				}
				if notificationType == NOTIFICATION_EDGE || notificationType == NOTIFICATION_EDGE_SYNC {
					var resp string
					var err error
					if edgeIDResolver == nil {
						return "", fmt.Errorf(base.PrefixRequestID(ctx, "Invalid edge ID resolver"))
					}
					edgeID := edgeIDResolver(doc)
					if edgeID == nil {
						glog.Errorf(base.PrefixRequestID(ctx, "Edge ID cannot be resolved for doc %+v"), doc)
						return "", fmt.Errorf("Missing edge ID")
					}
					origin := os.Getenv("HOSTNAME")
					if notificationType == NOTIFICATION_EDGE {
						_, err = msgSvc.SendMessage(ctx, origin, ap.TenantID, *edgeID, msg, req)
					} else {
						// For NOTIFICATION_EDGE_SYNC, we expect a response.
						glog.Infof(base.PrefixRequestID(ctx, "Sync api. Expecting websocket response"))
						resp, err = msgSvc.SendMessageSync(ctx, origin, ap.TenantID, *edgeID, msg, req)
					}
					if err == nil {
						glog.Infof(base.PrefixRequestID(ctx, "Websocket send completed for Edge ID: %s, Message: %s, Body: %+v, res: %s"), *edgeID, msg, req, resp)
					} else {
						glog.Infof(base.PrefixRequestID(ctx, "Websocket send failed for Edge ID: %s, Message: %s, Body: %+v. Error: %s"), *edgeID, msg, req, err.Error())
					}
					return resp, err
				}
				return "", nil
			}
		}
		err := getFn(r.Context(), ps.ByName(paramName), ps.ByName(paramName2), w, fn)
		handleResponse(w, r, err, "CUSTOM %s, tenantID=%s", msg, ap.TenantID)
	}))
}

func getCanonicalType(entityType string) string {
	switch entityType {
	case EntityTypeApplication:
		return CanonicalTypeApplication
	case EntityTypeCategory:
		return CanonicalTypeCategory
	case EntityTypeCloudCreds:
		return CanonicalTypeCloudCreds
	case EntityTypeDataSource:
		return CanonicalTypeDataSource
	case EntityTypeDataStream:
		return CanonicalTypeDataStream
	case EntityTypeDockerProfile:
		return CanonicalTypeDockerProfile
	case EntityTypeEdge:
		return CanonicalTypeEdge
	case EntityTypeEdgeDevice:
		return CanonicalTypeEdgeDevice
	case EntityTypeEdgeCluster:
		return CanonicalTypeEdgeCluster
	case EntityTypeServiceDomain:
		return CanonicalTypeServiceDomain
	case EntityTypeNode:
		return CanonicalTypeNode
	case EntityTypeProject:
		return CanonicalTypeProject
	case EntityTypeScript:
		return CanonicalTypeScript
	case EntityTypeScriptRuntime:
		return CanonicalTypeScriptRuntime
	case EntityTypeSensor:
		return CanonicalTypeSensor
	case EntityTypeUser:
		return CanonicalTypeUser
	case EntityTypeExecuteEdgeUpgrade:
		return CanonicalTypeExecuteEdgeUpgrade
	case EntityTypeMLModel:
		return CanonicalTypeMLModel
	case EntityTypeProjectService:
		return CanonicalTypeProjectService
	case EntityTypeSoftwareUpdate:
		return CanonicalTypeSoftwareUpdate
	case EntityTypeLogCollector:
		return CanonicalTypeLogCollector
	case EntityTypeServiceInstance:
		return CanonicalTypeServiceInstance
	case EntityTypeServiceBinding:
		return CanonicalTypeServiceBinding
	case EntityTypeDataDriverInstance:
		return CanonicalTypeDataDriverInstance
	case EntityTypeDataDriverConfig:
		return CanonicalTypeDataDriverConfig
	}
	glog.Warningf("Entity type not set for entityType %s", entityType)
	return ""
}
func getMessageWithPrefix(entityType string, prefix string) string {
	ct := getCanonicalType(entityType)
	if ct != "" {
		return prefix + ct
	}
	return ""
}
func getDeleteMessage(entityType string) string {
	return getMessageWithPrefix(entityType, "onDelete")
}
func getCreateMessage(entityType string) string {
	return getMessageWithPrefix(entityType, "onCreate")
}
func getUpdateMessage(entityType string) string {
	return getMessageWithPrefix(entityType, "onUpdate")
}

func renderForEdge(dbAPI api.ObjectModelAPI,
	tenantID, edgeID, reqID string,
	doc interface{}) (out interface{}, err error) {
	// Document prefers to render itself
	if renderer, ok := doc.(Renderer); ok {
		ctx := base.GetAdminContext(reqID, tenantID)
		glog.Infof(base.PrefixRequestID(ctx, "Invoke custom renderer for %T for edge %s"),
			doc, edgeID)
		doc, err = renderer.RenderForEdge(dbAPI, tenantID, edgeID)
		if err != nil {
			return
		}
	}
	return api.ObjectRequest{
		RequestID: reqID,
		TenantID:  tenantID,
		Doc:       doc,
	}, nil
}

func sendMessageToEdge(ctx context.Context, dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService, tenantID string, doc interface{}, opType string, msg string, req interface{}) error {
	var err error
	reqID := base.GetRequestID(ctx)

	// note: model.GetEdgeID does not work for doc = model.Edge
	edgeID := model.GetEdgeID(doc)

	// special handling for edges/service domains message
	isUpdateEdge := false
	isDeleteEdge := false
	projectsToNotify := []model.Project{}
	switch msg {
	case getUpdateMessage(EntityTypeEdge):
		isUpdateEdge = true
		updateMsg := doc.(model.UpdateEdgeMessage)
		doc = updateMsg.Doc
		projectsToNotify = updateMsg.Projects
		edge, ok := doc.(model.Edge)
		if ok {
			edgeID = &edge.ID
		}
	case getUpdateMessage(EntityTypeServiceDomain):
		isUpdateEdge = true
		updateMsg := doc.(model.UpdateServiceDomainMessage)
		doc = updateMsg.Doc
		projectsToNotify = updateMsg.Projects
		svcDomain, ok := doc.(model.ServiceDomain)
		if ok {
			edgeID = &svcDomain.ID
		}
	case getDeleteMessage(EntityTypeEdge):
		isDeleteEdge = true
	case getDeleteMessage(EntityTypeServiceDomain):
		isDeleteEdge = true
	}

	if edgeID != nil {
		if opType == CreateUpdateOpType {
			req, err = renderForEdge(dbAPI, tenantID, *edgeID, reqID, doc)
			if err != nil {
				return err
			}
		}

		// edge is not currently listening for edge update / delete event,
		// so use Emit to avoid wait
		origin := os.Getenv("HOSTNAME")
		if isUpdateEdge || isDeleteEdge {
			err = msgSvc.EmitMessage(ctx, origin, tenantID, *edgeID, msg, req)
		} else {
			_, err = msgSvc.SendMessage(ctx, origin, tenantID, *edgeID, msg, req)
		}
		if err == nil {
			glog.Infof(base.PrefixRequestID(ctx, "Websocket send completed for Edge ID: %s, Message: %s, Body: %+v"), *edgeID, msg, req)
		} else {
			glog.Infof(base.PrefixRequestID(ctx, "Websocket send failed for Edge ID: %s, Message: %s, Body: %+v. Error: %s"), *edgeID, msg, req, err.Error())
		}
		if isUpdateEdge && len(projectsToNotify) != 0 {
			for _, project := range projectsToNotify {
				msgSvc.EmitMessage(ctx, origin, tenantID, *edgeID, "onUpdateProject", api.ObjectRequest{
					RequestID: reqID,
					TenantID:  tenantID,
					Doc:       project,
				})
			}
		}
		return err
	}
	glog.Warningf(base.PrefixRequestID(ctx, "Skip send message in %s handle: failed to find edge id: %+v\n"), opType, doc)
	return nil
}

func broadcastMessageToTenant(ctx context.Context, dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService, r *http.Request, tenantID string, doc interface{}, opType string, msg string, req interface{}) error {
	// custom handling based on entityType
	var errOut error
	reqID := base.GetRequestID(ctx)
	broadcast := true
	origin := os.Getenv("HOSTNAME")
	pse, ok := doc.(model.ProjectScopedEntity)
	if ok {
		// entity is project scoped
		projectID := pse.GetProjectID()
		docID := pse.GetID()
		// if entity is not a project and not a global one, then only notify relevant edges
		if docID != projectID && projectID != "" {
			project, err := dbAPI.GetProject(r.Context(), projectID)
			if err == nil {
				glog.Infof(base.PrefixRequestID(ctx, "Send message %s for project %s\n"), msg, projectID)
				for _, edgeID := range project.EdgeIDs {
					req := req // don't overwrite parameter
					if opType == CreateUpdateOpType {
						req, errOut = renderForEdge(dbAPI, tenantID, edgeID, reqID, doc)
						if errOut != nil {
							continue
						}
					}
					err = msgSvc.EmitMessage(ctx, origin, tenantID, edgeID, msg, req)
					if err == nil {
						glog.Infof(base.PrefixRequestID(ctx, "Websocket emit completed for Edge ID: %s, Project ID: %s, Message: %s, Body: %+v"), edgeID, projectID, msg, req)
					} else {
						errOut = err
						glog.Infof(base.PrefixRequestID(ctx, "Websocket emit failed for Edge ID: %s, Project ID: %s, Message: %s, Body: %+v. Error: %s"), edgeID, projectID, msg, req, err.Error())
					}
				}
			} else {
				errOut = err
				glog.Warningf(base.PrefixRequestID(ctx, "SendMessage: skipped: failed to find project id: %s\n"), projectID)
			}
			broadcast = false
		}
	} else {
		// if entity is cloud profile or docker profile, then only send to all edges of all projects the entity is associated to
		sdoc, ok := doc.(model.ScopedEntity)
		if ok {
			for _, edgeID := range sdoc.EdgeIDs {
				var x interface{}

				req := req // don't overwrite parameter
				if opType == CreateUpdateOpType {
					req, errOut = renderForEdge(dbAPI, tenantID, edgeID, reqID, doc)
					if errOut != nil {
						continue
					}
				}
				req2, ok := req.(api.ObjectRequest)
				if ok {
					req2.Doc = sdoc.Doc
					x = req2
				} else {
					x = req
				}
				err := msgSvc.EmitMessage(ctx, origin, tenantID, edgeID, msg, x)
				if err == nil {
					glog.Infof(base.PrefixRequestID(ctx, "Websocket emit completed for Edge ID: %s, Message: %s, Body: %+v"), edgeID, msg, x)
				} else {
					errOut = err
					glog.Errorf(base.PrefixRequestID(ctx, "Websocket emit failed for Edge ID: %s, Message: %s, Body: %+v. Error: %s"), edgeID, msg, x, err.Error())
				}
			}
			broadcast = false
		}
	}
	if broadcast {
		if opType == CreateUpdateOpType {
			req = api.ObjectRequest{
				RequestID: reqID,
				TenantID:  tenantID,
				Doc:       doc,
			}
		}
		msgSvc.BroadcastMessage(ctx, tenantID, msg, req)
		glog.Infof(base.PrefixRequestID(ctx, "Broadcast completed for Message: %s, Body: %+v"), reqID, msg, req)
	}
	return errOut
}

func makeDeleteHandle(dbAPI api.ObjectModelAPI, deleteFn func(context.Context, string, io.Writer, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityType string, notificationType int, paramName string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		id := ps.ByName(paramName)
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) error {
				msg := getDeleteMessage(entityType)
				req := api.DeleteRequest{
					TenantID: ap.TenantID,
					ID:       id,
				}
				if notificationType == NOTIFICATION_TENANT {
					broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, doc, DeleteOpType, msg, req)
					glog.Infof(base.PrefixRequestID(ctx, "Broadcast completed for Message: %s, Body: %+v"), msg, req)
				} else if notificationType == NOTIFICATION_EDGE {
					sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, doc, DeleteOpType, msg, req)
				}

				sendDataIfcUpdates := func(endpoints []model.DataIfcEndpoint) {
					for _, e := range endpoints {
						dSource, err := dbAPI.GetDataSource(ctx, e.ID)
						if err != nil {
							glog.Errorf(base.PrefixRequestID(ctx, "Failed to send updates for data source %s. %s"), dSource.ID, err.Error())
						} else if dSource.IfcInfo != nil && dSource.IfcInfo.Kind == model.DataIfcEndpointKindOut {
							glog.Infof(base.PrefixRequestID(ctx, "Sending updates for datasource %s to edge %s"), dSource.Name, dSource.EdgeID)
							sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, dSource, CreateUpdateOpType, getUpdateMessage(EntityTypeDataSource), req)
						}
					}
				}

				// Send an update message to the edge for change in topics
				if entityType == EntityTypeDataStream {
					ds := doc.(model.DataStream)
					sendDataIfcUpdates(ds.DataIfcEndpoints)
				}

				// Send an update message to the edge for change in topics
				if entityType == EntityTypeApplication {
					app := doc.(model.Application)
					sendDataIfcUpdates(app.DataIfcEndpoints)
				}

				return nil
			}
		}
		err := deleteFn(r.Context(), id, w, fn)
		if err == nil {
			// Publish delete event
			event := model.EntityCRUDEvent{ID: base.GetUUID(), TenantID: ap.TenantID, EntityID: id, Message: getDeleteMessage(entityType)}
			base.Publisher.Publish(r.Context(), &event)
		}
		handleResponse(w, r, err, "DELETE %s, tenantID=%s, id=%s", entityType, ap.TenantID, id)
	}))
}

func makeDeleteHandle2(dbAPI api.ObjectModelAPI, deleteFn func(context.Context, string, int, io.Writer, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityMessage EntityMessage, notificationType int, paramName string, paramName2 string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		entityType := entityMessage.EntityType
		msg := entityMessage.Message
		w.Header().Set("Content-Type", "application/json")
		id := ps.ByName(paramName)
		p2, _ := strconv.Atoi(ps.ByName(paramName2))
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) error {
				// Note: here message payload is doc literal, not api.DeleteRequest
				// Delete2 is used by MLModelVersion delete,
				// which will trigger MLModel Update,
				// so opType is CreateUpdateOpType
				if notificationType == NOTIFICATION_TENANT {
					broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, doc, CreateUpdateOpType, msg, doc)
					glog.Infof(base.PrefixRequestID(ctx, "Broadcast completed for Message: %s, Body: %+v"), msg, doc)
				} else if notificationType == NOTIFICATION_EDGE {
					sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, doc, CreateUpdateOpType, msg, doc)
				}
				return nil
			}
		}
		err := deleteFn(r.Context(), id, p2, w, fn)
		if err == nil {
			// Publish delete event
			event := model.EntityCRUDEvent{ID: base.GetUUID(), TenantID: ap.TenantID, EntityID: id, Message: getDeleteMessage(entityType)}
			base.Publisher.Publish(r.Context(), &event)
		}
		handleResponse(w, r, err, "DELETE %s, tenantID=%s, id=%s, p2=%d", entityType, ap.TenantID, id, p2)
	}))
}

func makeCreateHandle(dbAPI api.ObjectModelAPI, createFn func(context.Context, io.Writer, io.Reader, func(context.Context, interface{}) error) error,
	msgSvc api.WSMessagingService, entityType string, notificationType int,
) httprouter.Handle {
	return makeCreateOrUpdateHandle(dbAPI, true, createFn, msgSvc, entityType, notificationType, "")
}

func makeUpdateHandle(dbAPI api.ObjectModelAPI, updateFn func(context.Context, io.Writer, io.Reader, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityType string, notificationType int, paramName string) httprouter.Handle {
	return makeCreateOrUpdateHandle(dbAPI, false, updateFn, msgSvc, entityType, notificationType, paramName)
}

func makeCreateOrUpdateHandle(dbAPI api.ObjectModelAPI, isCreate bool, createOrUpdateFn func(context.Context, io.Writer, io.Reader, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityType string, notificationType int, paramName string) httprouter.Handle {
	getMsgFn := getUpdateMessage
	op := "update"
	if isCreate {
		getMsgFn = getCreateMessage
		op = "create"
	}
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		if !isCreate {
			if paramName != "" {
				authContext, err := base.GetAuthContext(r.Context())
				if err != nil {
					// error
					handleResponse(w, r, err, "%s %s, tenantID=%s", op, entityType, ap.TenantID)
					return
				}
				authContext.ID = ps.ByName(paramName)
			}
		}
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			var opType string
			var msg string
			var req interface{}
			fn = func(ctx context.Context, doc interface{}) error {
				se, ok := doc.(model.StatefulEntity)
				if ok && se.GetEntityState() == model.UndeployEntityState {
					if !isCreate {
						// Undeploy the entity based on the state
						glog.V(3).Infof(base.PrefixRequestID(ctx, "Entity state is %s. Undeploying %s"), se.GetEntityState(), se.GetID())
						msg = getDeleteMessage(entityType)
						req = api.DeleteRequest{
							TenantID: ap.TenantID,
							ID:       se.GetID(),
						}
						opType = DeleteOpType
					}
				} else {
					msg = getMsgFn(entityType)
					opType = CreateUpdateOpType
				}

				sendDataIfcEndpointUpdates := func(endpoints []model.DataIfcEndpoint) {
					for _, e := range endpoints {
						dSource, err := dbAPI.GetDataSource(ctx, e.ID)
						if err != nil {
							glog.Errorf(base.PrefixRequestID(ctx, "Failed to send updates for data source %s. %s"), dSource.ID, err.Error())
						} else if dSource.IfcInfo != nil && dSource.IfcInfo.Kind == model.DataIfcEndpointKindOut {
							glog.Infof(base.PrefixRequestID(ctx, "Sending updates for datasource %s to edge %s as part of %s CRUD"), dSource.Name, dSource.EdgeID, entityType)
							sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, dSource, CreateUpdateOpType, getUpdateMessage(EntityTypeDataSource), req)
						}
					}
				}

				// Send  updates for a all data ifc endpoints associated with the  data stream
				if entityType == EntityTypeDataStream {
					ds := doc.(model.DataStream)
					sendDataIfcEndpointUpdates(ds.DataIfcEndpoints)
				}

				// Send updates for all out data ifcs for apps
				// Intentionally sending this before the app to force certain ordering w.r.t messages on the edge
				if entityType == EntityTypeApplication {
					app := doc.(*api.App)
					sendDataIfcEndpointUpdates(app.DataIfcEndpoints)
				}

				if len(opType) > 0 {
					if notificationType == NOTIFICATION_TENANT {
						broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, doc, opType, msg, req)
						glog.V(3).Infof(base.PrefixRequestID(ctx, "Broadcast completed for Message: %s, Body: %+v"), msg, req)
					} else if notificationType == NOTIFICATION_EDGE {
						sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, doc, opType, msg, req)
						glog.V(3).Infof(base.PrefixRequestID(ctx, "Send completed for Message: %s, Body: %+v"), msg, req)
					}
				}

				// Once we sent the data source update across, we should send any apps that might be impacted by this data source update
				if entityType == EntityTypeDataSource {
					ds := doc.(model.DataSource)
					apps, err := dbAPI.SelectAllApplicationsForDataIfcEndpoint(ctx, ds.ID)
					if err != nil {
						glog.Errorf(base.PrefixRequestID(ctx, "Failed to send the app updates for data ifc %s. %s"), ds.ID, err.Error())
					}

					// Apps are not tied to an edge, hence broadcast
					for _, app := range apps {
						// Skip sending updates for undeployed apps
						if app.GetEntityState() == model.UndeployEntityState {
							continue
						}
						glog.V(3).Infof(base.PrefixRequestID(ctx, "Sending app update message for data source update. %+v"), app)
						broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, app, CreateUpdateOpType, getUpdateMessage(EntityTypeApplication), req)
					}
				}
				return nil
			}
		}
		// Special handle for script update
		if !isCreate && entityType == EntityTypeScript {
			inputReader, err := toScriptWrapper(r)
			if err != nil {
				handleResponse(w, r, err, "%s %s, tenantID=%s", op, entityType, ap.TenantID)
			}
			err = createOrUpdateFn(r.Context(), w, inputReader, fn)
			handleResponse(w, r, err, "%s %s, tenantID=%s", op, entityType, ap.TenantID)
		} else {
			err := createOrUpdateFn(r.Context(), w, r.Body, fn)
			handleResponse(w, r, err, "%s %s, tenantID=%s", op, entityType, ap.TenantID)
		}

	}))
}

func makeCreateHandle2(dbAPI api.ObjectModelAPI, createFn func(context.Context, string, io.Writer, *http.Request, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityMessage EntityMessage, notificationType int, paramName string) httprouter.Handle {
	entityType := entityMessage.EntityType
	msg := entityMessage.Message
	op := "create"

	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) error {
				if notificationType == NOTIFICATION_TENANT {
					broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, doc, CreateUpdateOpType, msg, nil)
				} else if notificationType == NOTIFICATION_EDGE {
					sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, doc, CreateUpdateOpType, msg, nil)
				}
				return nil
			}
		}
		id := ps.ByName(paramName)
		err := createFn(r.Context(), id, w, r, fn)
		handleResponse(w, r, err, "%s %s, tenantID=%s, id=%s", op, entityType, ap.TenantID, id)
	}))
}

func makeUpdateHandle2(dbAPI api.ObjectModelAPI, updateFn func(context.Context, string, int, io.Writer, *http.Request, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityMessage EntityMessage, notificationType int, paramName string, paramName2 string) httprouter.Handle {
	entityType := entityMessage.EntityType
	msg := entityMessage.Message
	op := "update"

	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) error {
				if notificationType == NOTIFICATION_TENANT {
					broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, doc, CreateUpdateOpType, msg, nil)
				} else if notificationType == NOTIFICATION_EDGE {
					sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, doc, CreateUpdateOpType, msg, nil)
				}
				return nil
			}
		}
		id := ps.ByName(paramName)
		p2, _ := strconv.Atoi(ps.ByName(paramName2))
		err := updateFn(r.Context(), id, p2, w, r, fn)
		handleResponse(w, r, err, "%s %s, tenantID=%s, id=%s", op, entityType, ap.TenantID, id)
	}))
}

func makeUpdateHandle3(dbAPI api.ObjectModelAPI, updateFn func(context.Context, string, string, io.Writer, *http.Request, func(context.Context, interface{}) error) error, msgSvc api.WSMessagingService, entityMessage EntityMessage, notificationType int, paramName string, paramName2 string) httprouter.Handle {
	entityType := entityMessage.EntityType
	msg := entityMessage.Message
	op := "update"

	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) error {
				if notificationType == NOTIFICATION_TENANT {
					broadcastMessageToTenant(ctx, dbAPI, msgSvc, r, ap.TenantID, doc, CreateUpdateOpType, msg, nil)
				} else if notificationType == NOTIFICATION_EDGE {
					sendMessageToEdge(ctx, dbAPI, msgSvc, ap.TenantID, doc, CreateUpdateOpType, msg, nil)
				}
				return nil
			}
		}
		id1 := ps.ByName(paramName)
		id2 := ps.ByName(paramName2)
		err := updateFn(r.Context(), id1, id2, w, r, fn)
		handleResponse(w, r, err, "%s %s, tenantID=%s, id=%s", op, entityType, ap.TenantID, id2)
	}))
}

// Handles script update request, takes http request, returns a new io.Reader that is a new ScriptForceUpdate object.
// ScriptForceUpdate.Doc = model.Script{}, ScriptForceUpdate.ForceUpdate = "?forceUpdate=true"
func toScriptWrapper(req *http.Request) (*bytes.Reader, error) {
	doc := &model.Script{}
	// This cast must be successful, since createOrUpdateFn(r.Context(), w, r.Body, fn) is working, which contains a implicit type cast
	src := req.Body.(io.Reader)
	// Read input body as model.Script
	err := base.Decode(&src, doc)
	if err != nil {
		return nil, err
	}
	// Will pass this new wrapper object of script into the update function, in order to pass the forceUpdate option.
	// Compared with modifying function defination of UpdateScript, having a wrapper data structure requires much less modifications.
	// Compared with modifying model.ScriptUpdateParam, we don't have to document this query forceUpdate parameter anywhere,
	// so that this forceUpdate parameter does not confuses customer and can be kept for internal usage.
	docWrapper := model.ScriptForceUpdate{}
	docWrapper.Doc = *doc
	docWrapper.ForceUpdate = false
	// Read force update option from query, this part should be transparent unless there is ?forceUpdate=true in URL query
	query := req.URL.Query()
	forceUpdate := query["forceUpdate"]
	if len(forceUpdate) > 0 {
		fUpdate, err := strconv.ParseBool(forceUpdate[0])
		if err == nil && fUpdate == true {
			docWrapper.ForceUpdate = fUpdate
		}
	}
	// Create new object and new reader for script update option
	docJSON, err := json.Marshal(docWrapper)
	if err != nil {
		return nil, errcode.NewInternalError(fmt.Sprintf("UpdateScript: docWrapper Marshal: %s", err.Error()))
	}
	docReader := bytes.NewReader(docJSON)
	return docReader, nil
}

func makeCustomMessageHandle(dbAPI api.ObjectModelAPI,
	callback func(context.Context, io.Writer, io.Reader, func(context.Context, interface{}) error) error,
	msgSvc api.WSMessagingService,
	msg string,
	notificationType int,
	edgeIDResolver func(interface{}) *string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		var fn func(context.Context, interface{}) error
		if notificationType != NOTIFICATION_NONE {
			fn = func(ctx context.Context, doc interface{}) error {
				reqID := base.GetRequestID(ctx)
				req := api.ObjectRequest{
					RequestID: reqID,
					TenantID:  ap.TenantID,
					Doc:       doc,
				}
				// Log only for create and update
				if notificationType == NOTIFICATION_TENANT {
					// Broadcast does not return error
					msgSvc.BroadcastMessage(ctx, ap.TenantID, msg, req)
					glog.Infof(base.PrefixRequestID(ctx, "Broadcast completed for Message: %s, Body: %+v"), msg, req)
					return nil
				} else if notificationType == NOTIFICATION_EDGE || notificationType == NOTIFICATION_EDGE_SYNC {
					var res string
					var err error
					if edgeIDResolver == nil {
						return fmt.Errorf("Invalid edge ID resolver")
					}
					edgeID := edgeIDResolver(doc)
					if edgeID == nil {
						return fmt.Errorf("No channel found")
					}
					origin := os.Getenv("HOSTNAME")
					if notificationType == NOTIFICATION_EDGE {
						res, err = msgSvc.SendMessage(ctx, origin, ap.TenantID, *edgeID, msg, req)
					} else {
						// For NOTIFICATION_EDGE_SYNC, we expect a response.
						res, err = msgSvc.SendMessageSync(ctx, origin, ap.TenantID, *edgeID, msg, req)
					}
					if err == nil {
						glog.Infof(base.PrefixRequestID(ctx, "Websocket send completed for Edge ID: %s, Message: %s, Body: %+v, res: %s"), *edgeID, msg, req, res)
					} else {
						glog.Infof(base.PrefixRequestID(ctx, "Websocket send failed for Edge ID: %s, Message: %s, Body: %+v. Error: %s"), *edgeID, msg, req, err.Error())
					}
					return err
				}
				return nil
			}
		}
		ap.ID = ps.ByName("id")
		err := callback(r.Context(), w, r.Body, fn)
		handleResponse(w, r, err, "CUSTOM %s, tenantID=%s", msg, ap.TenantID)
	}))
}

// CheckAuth ensures request Authorization header contains valid JWT token
// TODO: add RBAC support
func CheckAuth(dbAPI api.ObjectModelAPI, handle auth.ContextHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		claims, err := auth.VerifyAuthorization(r, dbAPI.GetPublicKeyResolver, dbAPI.GetClaimsVerifier)
		if err == nil {
			tenantID, ok := claims["tenantId"].(string)
			if ok {
				authContext := &base.AuthContext{TenantID: tenantID, Claims: claims}
				role := auth.GetSpecialRole(authContext)
				err = RouteRoleValidator.Validate(r, tenantID, role)
				if err == nil {
					ctx := context.WithValue(r.Context(), base.AuthContextKey, authContext)
					err = updateClaimUser(ctx, dbAPI, claims)
					if err == nil {
						err = updateClaimProjects(ctx, dbAPI, tenantID, claims)
						if err == nil {
							handle(w, r.WithContext(ctx), ps, authContext)
							return
						}
						if _, ok := err.(*errcode.PermissionDeniedError); !ok {
							// updateClaimProjects could fail if incompatible DB upgrade has not yet been applied
							w.Header().Set("Content-Type", "application/json")
							w.WriteHeader(http.StatusInternalServerError)
							fmt.Fprintf(w, `{"statusCode": 500, "message": "server not ready"}`)
							return
						}
					}
				}
			}
		}
		if err != nil {
			glog.Errorf(base.PrefixRequestID(r.Context(), "Failed to authorize the request. Error: %s"), err.Error())
		}
		// Catch all for the error cases not caught above
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprintf(w, `{"statusCode": 401, "message": "Unauthorized"}`)
	}
}

func updateClaimUser(ctx context.Context, dbAPI api.ObjectModelAPI, claims jwt.MapClaims) error {
	// user case
	if email, ok := claims["email"].(string); ok && email != "" {
		authContext, _ := base.GetAuthContext(ctx)
		user, err := dbAPI.GetUserByEmail(ctx, email)
		// get user by email does not check tenant id, so add check here
		if err == nil && authContext != nil && authContext.TenantID == user.TenantID {
			claims["specialRole"] = model.GetUserSpecialRole(&user)
			claims["id"] = user.ID
		} else if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "updateClaimUser for user %s failed. Error: %s"), email, err.Error())
			return err
		}
	} else if userID, ok := claims["id"].(string); ok && userID != "" {
		authContext, _ := base.GetAuthContext(ctx)
		user, err := dbAPI.GetUser(ctx, userID)
		if err == nil && authContext != nil && authContext.TenantID == user.TenantID {
			claims["specialRole"] = model.GetUserSpecialRole(&user)
			claims["id"] = user.ID
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "updateClaimUser for user %s failed. Error: %s"), userID, err.Error())
			return err
		}
	}
	return nil
}

func updateClaimProjects(ctx context.Context, dbAPI api.ObjectModelAPI, tenantID string, claims jwt.MapClaims) error {
	// user case
	userID, ok := claims["id"].(string)
	if ok && userID != "" {
		projectRoles, err := dbAPI.GetUserProjectRoles(ctx, userID)
		if err == nil {
			// Do not block on error
			projectRoles, err = addTrialProjectsIfMissing(ctx, dbAPI, tenantID, userID, projectRoles)
			// Add it irrespective of error.
			// Caller handles the error
			claims["projects"] = projectRoles
			if err != nil {
				authErr, ok := err.(*errcode.PermissionDeniedError)
				if !ok || authErr.Reason != "Trial expired" {
					// Do not block U2 project membership update error because the project is no longer present.
					// Only block if the reason is trial expired because GetProject can return permission error
					err = nil
				}
			}
		}
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "updateClaimProjects for user %s failed: %s"), userID, err.Error())
		}
		return err
	}
	// edge case
	edgeID, ok := claims["edgeId"].(string)
	if ok && edgeID != "" {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims:   claims,
		}
		ctx := context.WithValue(ctx, base.AuthContextKey, authContext)
		// TODO: Change in multinode pt4
		projectRoles, err := dbAPI.GetEdgeProjectRoles(ctx, edgeID)
		if err == nil {
			claims["projects"] = projectRoles
		} else {
			glog.Warningf(base.PrefixRequestID(ctx, "updateClaimProjects for edge %s failed: %s"), edgeID, err.Error())
			return err
		}
	}
	return nil
}

// addTrialProjectsIfMissing adds user to the project(s) created for trial users by bott service.
// projectRoles slice is updated if the addition is successful
func addTrialProjectsIfMissing(ctx context.Context, dbAPI api.ObjectModelAPI, tenantID string, userID string, projectRoles []model.ProjectRole) ([]model.ProjectRole, error) {
	if !*config.Cfg.EnableTrial {
		return projectRoles, nil
	}
	tenantClaim, err := dbAPI.GetTenantClaim(ctx, tenantID)
	if err != nil {
		glog.V(5).Infof(base.PrefixRequestID(ctx, "Failed to get tenant claim %s"), tenantID)
		return projectRoles, err
	}
	if tenantClaim.Trial {
		// The trial tenant is no longer usable
		if tenantClaim.State == tenantpoolcore.Deleting {
			glog.Warningf(base.PrefixRequestID(ctx, "TenantClaim %+v is getting deleted"), tenantClaim)
			return projectRoles, errcode.NewPermissionDeniedError("Trial expired")
		}
	}
	if tenantClaim.Resources == nil {
		glog.V(5).Infof(base.PrefixRequestID(ctx, "No resources found for tenant claim %s"), tenantID)
		return projectRoles, nil
	}
	for resourceID, resource := range tenantClaim.Resources {
		if resource == nil || resource.Type != tenantpoolmodel.ProjectResourceType {
			continue
		}
		missing := true
		for _, projectRole := range projectRoles {
			if projectRole.ProjectID == resourceID {
				missing = false
				break
			}
		}
		if missing {
			glog.V(3).Infof(base.PrefixRequestID(ctx, "Project %s is missing for user %s. Adding it..."), resourceID, userID)
			project, err := dbAPI.GetProject(ctx, resourceID)
			if err != nil {
				glog.V(5).Infof(base.PrefixRequestID(ctx, "Failed to get project %s to add user %s"), resourceID, userID)
				return projectRoles, err
			}
			projectUserInfo := model.ProjectUserInfo{UserID: userID, Role: model.ProjectRoleAdmin}
			project.Users = append(project.Users, projectUserInfo)
			authContext := &base.AuthContext{
				TenantID: tenantID,
				Claims: jwt.MapClaims{
					"specialRole": "admin",
				},
			}
			adminCtx := context.WithValue(ctx, base.AuthContextKey, authContext)
			// No callback as it is just user update
			// Concurrent calls are not expected
			// Otherwise, we need to add another API to perform CAS
			_, err = dbAPI.UpdateProject(adminCtx, &project, nil)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Failed to update project %s to add user %s"), project.ID, userID)
				return projectRoles, err
			}
			projectRoles = append(projectRoles, model.ProjectRole{ProjectID: resourceID, Role: projectUserInfo.Role})
		}
	}
	return projectRoles, nil
}

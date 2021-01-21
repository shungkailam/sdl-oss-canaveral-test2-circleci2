package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
)

const entityTypeHTTPServiceProxy = "httpserviceproxy"

func init() {
	queryMap["SelectHTTPServiceProxiesProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM http_service_proxy_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (project_id IN (:project_ids)) %s`

	queryMap["SelectHTTPServiceProxiesAdminTemplate"] = `SELECT *, count(*) OVER() as total_count FROM http_service_proxy_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (project_id is NULL) %s`

	queryMap["SelectHTTPServiceProxiesAdminProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM http_service_proxy_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (project_id is NULL OR project_id IN (:project_ids)) %s`

	queryMap["CreateHTTPServiceProxy"] = `INSERT INTO http_service_proxy_model (id, tenant_id, edge_cluster_id, name, type, project_id, service_name, service_port, service_namespace, duration, username, password, hostname, hostport, public_key, version, expires_at, created_at, updated_at) VALUES (:id, :tenant_id, :edge_cluster_id, :name, :type, :project_id, :service_name, :service_port, :service_namespace, :duration, :username, :password, :hostname, :hostport, :public_key, :version, :expires_at, :created_at, :updated_at)`

	queryMap["UpdateHTTPServiceProxy"] = `UPDATE http_service_proxy_model SET version = :version, name = :name, duration = :duration, expires_at = :expires_at, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	orderByHelper.Setup(entityTypeHTTPServiceProxy, []string{"id", "version", "created_at", "updated_at", "expired_at", "name", "edge_cluster_id", "type", "service_name", "service_port", "service_namespace", "duration", "project_id", "username", "hostname"})

}

// HTTPServiceProxyDBO is DB object model for http service proxy
type HTTPServiceProxyDBO struct {
	model.ServiceDomainEntityModelDBO
	model.HTTPServiceProxyCore
	ProjectID *string `json:"projectId,omitempty" db:"project_id"`
}

func (app HTTPServiceProxyDBO) GetProjectID() string {
	if app.ProjectID != nil {
		return *app.ProjectID
	}
	return ""
}

func (app HTTPServiceProxyDBO) ToHTTPServiceProxy() (model.HTTPServiceProxy, error) {
	serviceProxy := model.HTTPServiceProxy{}
	err := base.Convert(&app, &serviceProxy)
	return serviceProxy, err
}

type HTTPServiceProxyProjects struct {
	HTTPServiceProxyDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

func dropPublicKeys(ps []model.HTTPServiceProxy) {
	for i := range ps {
		ps[i].PublicKey = nil
	}
}
func getDNS(p model.HTTPServiceProxy) string {
	fillURL(&p)
	i := len("https://")
	return p.DNSURL[i:]
}
func fillURL(p *model.HTTPServiceProxy) {
	base := *config.Cfg.ProxyUrlBase
	ep := p.GetProxyEndpointPath()
	p.URL = fmt.Sprintf("%s/%s", base, ep)
	p.DNSURL = model.MakeProxyURL(base, ep)
}

func fillURLs(ps []model.HTTPServiceProxy) {
	for i := range ps {
		fillURL(&ps[i])
	}
}

// get DB query parameters for http service proxy
func getHTTPServiceProxyDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	sdModel := model.ServiceDomainEntityModelDBO{BaseModelDBO: tenantModel}
	param := HTTPServiceProxyDBO{ServiceDomainEntityModelDBO: sdModel}
	var projectIDs []string
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		projectIDs = []string{projectID}
	} else {
		projectIDs = auth.GetProjectIDs(authContext)
	}
	isInfraAdmin := auth.IsInfraAdminRole(authContext)
	inQuery := true
	key := "SelectHTTPServiceProxiesProjectsTemplate"
	if isInfraAdmin {
		key = "SelectHTTPServiceProxiesAdminProjectsTemplate"
	}
	if len(projectIDs) == 0 {
		// not an error, just return empty response
		if !isInfraAdmin {
			return base.InQueryParam{}, nil // errcode.NewPermissionDeniedError("RBAC")
		}
		inQuery = false
		key = "SelectHTTPServiceProxiesAdminTemplate"
	}
	return base.InQueryParam{
		Param: HTTPServiceProxyProjects{
			HTTPServiceProxyDBO: param,
			ProjectIDs:          projectIDs,
		},
		Key:     key,
		InQuery: inQuery,
	}, nil
}

// internal API
func (dbAPI *dbObjectModelAPI) getHTTPServiceProxiesByProjectsForQuery(context context.Context, dbQueryParam base.InQueryParam, entitiesQueryParam *model.EntitiesQueryParam, filterExpired bool) ([]model.HTTPServiceProxy, int, error) {
	serviceProxies := []model.HTTPServiceProxy{}
	// authContext, err := base.GetAuthContext(context)
	// if err != nil {
	// 	return serviceProxies, 0, err
	// }
	if dbQueryParam.Key == "" {
		return serviceProxies, 0, nil
	}
	// tenantID := authContext.TenantID
	serviceProxyDBOs := []HTTPServiceProxyDBO{}
	// projectIDs := dbQueryParam.Param.(HTTPServiceProxyProjects).ProjectIDs

	var query string
	query, err := buildLimitQuery(entityTypeHTTPServiceProxy, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return serviceProxies, 0, err
	}
	if dbQueryParam.InQuery {
		err = dbAPI.QueryIn(context, &serviceProxyDBOs, query, dbQueryParam.Param)
	} else {
		err = dbAPI.Query(context, &serviceProxyDBOs, query, dbQueryParam.Param)
	}
	if err != nil {
		return serviceProxies, 0, err
	}
	if len(serviceProxyDBOs) == 0 {
		return serviceProxies, 0, nil
	}
	totalCount := 0
	first := true
	for _, serviceProxyDBO := range serviceProxyDBOs {
		serviceProxy := model.HTTPServiceProxy{}
		if first {
			first = false
			if serviceProxyDBO.TotalCount != nil {
				totalCount = *serviceProxyDBO.TotalCount
			}
		}
		err := base.Convert(&serviceProxyDBO, &serviceProxy)
		if err != nil {
			return []model.HTTPServiceProxy{}, 0, err
		}
		serviceProxies = append(serviceProxies, serviceProxy)
	}
	fillURLs(serviceProxies)
	if filterExpired {
		var filteredCount int
		serviceProxies, filteredCount = dbAPI.filterExpiredHTTPServiceProxies(context, serviceProxies)
		totalCount -= filteredCount
	}
	return serviceProxies, totalCount, nil
}

// filter out expired entries, also synchronously
// clean up expired entries. (Previously cleanup was done in a go routine,
// however, UI may try to recreate expired entry and get a name conflict)
// return: filtered list and number filtered
func (dbAPI *dbObjectModelAPI) filterExpiredHTTPServiceProxies(context context.Context, proxies []model.HTTPServiceProxy) ([]model.HTTPServiceProxy, int) {
	glog.V(5).Infof(base.PrefixRequestID(context, "filterExpiredHTTPServiceProxies: %+v"), proxies)
	filtered := []model.HTTPServiceProxy{}
	expired := []model.HTTPServiceProxy{}
	now := time.Now()
	for _, p := range proxies {
		if p.ExpiresAt.Before(now) {
			// expired
			expired = append(expired, p)
		} else {
			filtered = append(filtered, p)
		}
	}
	if len(expired) != 0 {
		for _, p := range expired {
			glog.V(5).Infof(base.PrefixRequestID(context, "filterExpiredHTTPServiceProxies: deleting: %+v"), p)
			r, err := dbAPI.DeleteHTTPServiceProxy(context, p.ID, nil)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(context, "filterExpiredHTTPServiceProxies: failed to delete %+v, err: %s"), p, err)
			} else {
				glog.V(5).Infof(base.PrefixRequestID(context, "filterExpiredHTTPServiceProxies: delete %+v response: %+v"), p, r)
			}
		}
	}
	glog.V(5).Infof(base.PrefixRequestID(context, "filterExpiredHTTPServiceProxies: expiration count: %d"), len(expired))
	return filtered, len(expired)
}

func (dbAPI *dbObjectModelAPI) CreateHTTPServiceProxy(context context.Context, i interface{} /* *model.HTTPServiceProxyCreateParamPayload */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.HTTPServiceProxyCreateResponsePayload{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.HTTPServiceProxyCreateParamPayload)
	if !ok {
		return resp, errcode.NewInternalError("CreateHTTPServiceProxy: type error")
	}

	doc := p.ToHTTPServiceProxy()

	// validate duration
	dur, err := time.ParseDuration(doc.Duration)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "CreateHTTPServiceProxy bad duration: %s, parse error %s\n"), doc.Duration, err)
		return resp, err
	}
	if dur < time.Duration(5)*time.Minute {
		return resp, errcode.NewBadRequestExError("Duration", fmt.Sprintf("The minimum duration is 5 minutes"))
	}

	// TODO: generate username, password,
	// allocate host, port
	doc.Hostname = "hostname"
	doc.Hostport = 23456
	if p.SetupBasicAuth {
		doc.Username = "serviceproxyuser"
		doc.Password = base.GenerateStrongPassword()
	}

	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.V(5).Infof(base.PrefixRequestID(context, "CreateHTTPServiceProxy doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.V(5).Infof(base.PrefixRequestID(context, "CreateHTTPServiceProxy doc.ID was invalid, update it to %s\n"), doc.ID)
	}

	if doc.Type != "PROJECT" {
		if !auth.IsInfraAdminRole(authContext) {
			err = errcode.NewPermissionDeniedError("RBAC/InfraAdmin/Required")
		}
	} else {
		err = auth.CheckRBAC(
			authContext,
			meta.EntityHTTPServiceProxy,
			meta.OperationCreate,
			auth.RbacContext{
				ProjectID:  doc.ProjectID,
				ProjNameFn: GetProjectNameFn(context, dbAPI),
			})
	}

	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	doc.ExpiresAt = now.Add(dur)
	httpServiceProxyDBO := HTTPServiceProxyDBO{}
	err = base.Convert(&doc, &httpServiceProxyDBO)
	if err != nil {
		return resp, err
	}

	doc2 := model.WstunRequest{
		ServiceDomainID:      doc.SvcDomainID,
		Endpoint:             doc.GetEndpoint(),
		TLSEndpoint:          p.TLSEndpoint,
		SkipCertVerification: p.SkipCertVerification,
	}
	dns := ""
	if p.SetupDNS {
		dns = getDNS(doc)
	}
	// piggy back on setup ssh tunneling to proxy the service
	options := setupSSHTunnelingOptions{
		doc:                 doc2,
		skipValidation:      true,
		duration:            dur,
		setupBasicAuth:      p.SetupBasicAuth,
		username:            doc.Username,
		password:            doc.Password,
		disableRewriteRules: p.DisableRewriteRules,
		dns:                 dns,
		headers:             p.Headers,
		publicKey:           "",
	}
	sr, err := dbAPI.setupSSHTunneling(context, options, callback)
	if err != nil {
		return resp, err
	}
	if sr.PublicKey != "" {
		httpServiceProxyDBO.PublicKey = base.StringPtr(sr.PublicKey)
	} else {
		glog.Warningf(base.PrefixRequestID(context, "CreateHTTPServiceProxy: no public key from setup? %+v\n"), sr)
	}

	_, err = dbAPI.NamedExec(context, queryMap["CreateHTTPServiceProxy"], &httpServiceProxyDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating http service proxy for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	// don't need create callback
	resp.ID = doc.ID
	resp.Username = doc.Username
	resp.Password = doc.Password
	resp.ExpiresAt = httpServiceProxyDBO.ExpiresAt
	httpProxy, _ := httpServiceProxyDBO.ToHTTPServiceProxy()
	fillURL(&httpProxy)
	resp.URL = httpProxy.URL
	resp.DNSURL = httpProxy.DNSURL
	return resp, nil
}
func (dbAPI *dbObjectModelAPI) CreateHTTPServiceProxyW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateHTTPServiceProxy, &model.HTTPServiceProxyCreateParamPayload{}, w, r, callback)
}
func (dbAPI *dbObjectModelAPI) UpdateHTTPServiceProxy(context context.Context, i interface{} /* *model.HTTPServiceProxyUpdateParamPayload */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.HTTPServiceProxyUpdateResponsePayload{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	if authContext.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	id := authContext.ID

	doc, err := dbAPI.GetHTTPServiceProxy(context, id)
	if err != nil {
		return resp, errcode.NewBadRequestError("httpServiceProxyID")
	}

	p, ok := i.(*model.HTTPServiceProxyUpdateParamPayload)
	if !ok {
		return resp, errcode.NewInternalError("UpdateHTTPServiceProxy: type error")
	}

	// even if name and duration stays the same, we will still go through
	// update since it will update expiresAt

	if doc.Type != "PROJECT" {
		if !auth.IsInfraAdminRole(authContext) {
			err = errcode.NewPermissionDeniedError("RBAC/InfraAdmin/Required")
		}
	} else {
		err = auth.CheckRBAC(
			authContext,
			meta.EntityHTTPServiceProxy,
			meta.OperationUpdate,
			auth.RbacContext{
				ProjectID:  doc.ProjectID,
				ProjNameFn: GetProjectNameFn(context, dbAPI),
			})
	}

	if err != nil {
		return resp, err
	}

	// apply update
	doc.Name = p.Name
	doc.Duration = p.Duration

	// validate duration
	dur, err := time.ParseDuration(doc.Duration)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "UpdateHTTPServiceProxy bad duration: %s, parse error %s\n"), doc.Duration, err)
		return resp, err
	}
	if dur < time.Duration(5)*time.Minute {
		return resp, errcode.NewBadRequestExError("Duration", fmt.Sprintf("The minimum duration is 5 minutes"))
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	doc.ExpiresAt = now.Add(dur)

	httpServiceProxyDBO := HTTPServiceProxyDBO{}
	err = base.Convert(&doc, &httpServiceProxyDBO)
	if err != nil {
		return resp, err
	}

	doc2 := model.WstunRequest{
		ServiceDomainID:      doc.SvcDomainID,
		Endpoint:             doc.GetEndpoint(),
		TLSEndpoint:          p.TLSEndpoint,
		SkipCertVerification: p.SkipCertVerification,
	}
	dns := ""
	if p.SetupDNS {
		dns = getDNS(doc)
	}
	// piggy back on setup ssh tunneling to update the service proxy
	publicKey := ""
	if doc.PublicKey != nil {
		publicKey = *doc.PublicKey
		glog.Infof(base.PrefixRequestID(context, "UpdateHTTPServiceProxy: reusing public key from existing entry: %s\n"), publicKey)
	} else {
		glog.Warningf(base.PrefixRequestID(context, "UpdateHTTPServiceProxy: no public key from existing entry?\n"))
	}
	options := setupSSHTunnelingOptions{
		doc:                 doc2,
		skipValidation:      true,
		duration:            dur,
		setupBasicAuth:      doc.Password != "",
		username:            doc.Username,
		password:            doc.Password,
		disableRewriteRules: p.DisableRewriteRules,
		dns:                 dns,
		headers:             p.Headers,
		publicKey:           publicKey,
	}
	sr, err := dbAPI.setupSSHTunneling(context, options, callback)
	if err != nil {
		return resp, err
	}
	if sr.PublicKey != "" {
		glog.Infof(base.PrefixRequestID(context, "UpdateHTTPServiceProxy: public key returned from setup: %s\n"), sr.PublicKey)
		httpServiceProxyDBO.PublicKey = base.StringPtr(sr.PublicKey)
	} else {
		glog.Warningf(base.PrefixRequestID(context, "UpdateHTTPServiceProxy: no public key from setup? %+v\n"), sr)
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateHTTPServiceProxy"], &httpServiceProxyDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating http service proxy for ID %s and tenant ID %s. Error: %s"), doc.ID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	// don't need update callback
	resp.ID = doc.ID
	resp.Username = doc.Username
	resp.Password = doc.Password
	resp.ExpiresAt = doc.ExpiresAt
	fillURL(&doc)
	resp.URL = doc.URL
	resp.DNSURL = doc.DNSURL

	return resp, nil
}
func (dbAPI *dbObjectModelAPI) UpdateHTTPServiceProxyW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, dbAPI.UpdateHTTPServiceProxy, &model.HTTPServiceProxyUpdateParamPayload{}, w, r, callback)
}

func (dbAPI *dbObjectModelAPI) getHTTPServiceProxies(context context.Context, projectID string, proxyID string, entitiesQueryParam *model.EntitiesQueryParam, filterExpired bool) (model.HTTPServiceProxyListPayload, error) {
	result := model.HTTPServiceProxyListPayload{HTTPServiceProxyList: []model.HTTPServiceProxy{}}
	dbQueryParam, err := getHTTPServiceProxyDBQueryParam(context, projectID, proxyID)
	if err != nil {
		return result, err
	}
	if dbQueryParam.Key == "" {
		return result, nil
	}
	queryParam := model.GetEntitiesQueryParam(nil)
	httpServiceProxies, _, err := dbAPI.getHTTPServiceProxiesByProjectsForQuery(context, dbQueryParam, queryParam, filterExpired)
	if err != nil {
		return result, err
	}
	result.HTTPServiceProxyList = httpServiceProxies
	return result, nil
}
func (dbAPI *dbObjectModelAPI) SelectAllHTTPServiceProxies(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) (model.HTTPServiceProxyListPayload, error) {
	return dbAPI.getHTTPServiceProxies(context, "", "", entitiesQueryParam, true)
}
func (dbAPI *dbObjectModelAPI) getHTTPServiceProxiesW(context context.Context, projectID string, proxyID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getHTTPServiceProxyDBQueryParam(context, projectID, proxyID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.HTTPServiceProxyListPayload{HTTPServiceProxyList: []model.HTTPServiceProxy{}})
	}
	queryParam := model.GetEntitiesQueryParam(req)
	httpServiceProxies, totalCount, err := dbAPI.getHTTPServiceProxiesByProjectsForQuery(context, dbQueryParam, queryParam, true)
	if err != nil {
		return err
	}
	// no need to return publicKey in user facing API
	dropPublicKeys(httpServiceProxies)
	if len(proxyID) == 0 {
		queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeHTTPServiceProxy}
		entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
		r := model.HTTPServiceProxyListPayload{
			EntityListResponsePayload: entityListResponsePayload,
			HTTPServiceProxyList:      httpServiceProxies,
		}
		return json.NewEncoder(w).Encode(r)
	}
	if len(httpServiceProxies) == 0 {
		return errcode.NewRecordNotFoundError(proxyID)
	}
	return json.NewEncoder(w).Encode(httpServiceProxies[0])
}
func (dbAPI *dbObjectModelAPI) SelectAllHTTPServiceProxiesW(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getHTTPServiceProxiesW(ctx, "", "", w, req)
}
func (dbAPI *dbObjectModelAPI) GetHTTPServiceProxy(ctx context.Context, id string) (model.HTTPServiceProxy, error) {
	return dbAPI.getHTTPServiceProxy(ctx, id, true)
}

// internal API
// typically when we get http service proxy,
// we want to filter out expired entries.
// However, when doing expiration delete,
// we must include expired entries for GET response,
// otherwise the delete will be skipped
// due to entry not found.
func (dbAPI *dbObjectModelAPI) getHTTPServiceProxy(ctx context.Context, id string, filterExpired bool) (model.HTTPServiceProxy, error) {
	if len(id) == 0 {
		return model.HTTPServiceProxy{}, errcode.NewBadRequestError("httpServiceProxyID")
	}
	httpProxies, err := dbAPI.getHTTPServiceProxies(ctx, "", id, nil, filterExpired)
	if err != nil {
		return model.HTTPServiceProxy{}, err
	}
	if len(httpProxies.HTTPServiceProxyList) == 0 {
		return model.HTTPServiceProxy{}, errcode.NewRecordNotFoundError(id)
	}
	return httpProxies.HTTPServiceProxyList[0], nil
}
func (dbAPI *dbObjectModelAPI) GetHTTPServiceProxyW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	return dbAPI.getHTTPServiceProxiesW(ctx, "", id, w, req)
}
func (dbAPI *dbObjectModelAPI) DeleteHTTPServiceProxy(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	// don't filter out expired entries below
	filterExpired := false
	sr, err := dbAPI.getHTTPServiceProxy(context, id, filterExpired)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}

	if sr.Type != "PROJECT" {
		if !auth.IsInfraAdminRole(authContext) {
			err = errcode.NewPermissionDeniedError("RBAC/InfraAdmin/Required")
		}
	} else {
		err = auth.CheckRBAC(
			authContext,
			meta.EntityHTTPServiceProxy,
			meta.OperationDelete,
			auth.RbacContext{
				ProjectID:  sr.ProjectID,
				ProjNameFn: GetProjectNameFn(context, dbAPI),
			})
	}
	if err != nil {
		return resp, err
	}
	// pass existing duration into teardown call
	dur, _ := time.ParseDuration(sr.Duration)
	publicKey := ""
	if sr.PublicKey != nil {
		publicKey = *sr.PublicKey
	} else {
		glog.Warningf(base.PrefixRequestID(context, "DeleteHTTPServiceProxy: no public key?\n"))

	}
	wtd := model.WstunTeardownRequest{
		ServiceDomainID: sr.SvcDomainID,
		Endpoint:        sr.GetEndpoint(),
		PublicKey:       publicKey,
	}
	// piggy back on teardown ssh tunneling to delete the service proxy
	options := teardownSSHTunnelingOptions{
		doc:            wtd,
		skipValidation: true,
		duration:       dur,
		setupBasicAuth: sr.Username != "" && sr.Password != "",
	}
	err = dbAPI.teardownSSHTunneling(context, options, callback)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "DeleteHTTPServiceProxy teardown failed: %s\n"), err)
		return resp, err
	}
	// don't need delete callback
	return DeleteEntity(context, dbAPI, "http_service_proxy_model", "id", id, sr, nil)
}
func (dbAPI *dbObjectModelAPI) DeleteHTTPServiceProxyW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	id := authContext.ID
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteHTTPServiceProxy), id, w, callback)
}

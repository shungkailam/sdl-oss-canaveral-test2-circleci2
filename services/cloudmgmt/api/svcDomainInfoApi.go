package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
)

const (
	entityTypeSvcDomainInfo = "serviceDomainInfo"
)

func init() {
	queryMap["SelectServiceDomainsInfo"] = `SELECT * FROM service_domain_info_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id)`
	queryMap["SelectServiceDomainsInfoInTemplate"] = `SELECT * FROM service_domain_info_model WHERE tenant_id = :tenant_id AND (id IN (:edge_cluster_ids)) %s`
	queryMap["SelectServiceDomainInfoIDsTemplate"] = `SELECT im.id from service_domain_info_model im, edge_cluster_model cm where im.id = cm.id AND im.tenant_id = :tenant_id AND (:type = '' OR cm.type = :type OR (:type = 'EDGE' AND cm.type is null)) %s`
	queryMap["CreateServiceDomainInfo"] = `INSERT INTO service_domain_info_model (id, version, tenant_id, edge_cluster_id, artifacts, created_at, updated_at) VALUES (:id, :version, :tenant_id, :edge_cluster_id, :artifacts, :created_at, :updated_at)
	ON CONFLICT (tenant_id, edge_cluster_id) DO UPDATE SET version = :version, artifacts = :artifacts, updated_at = :updated_at WHERE service_domain_info_model.tenant_id = :tenant_id AND service_domain_info_model.id = :id`

	orderByHelper.Setup(entityTypeSvcDomainInfo, []string{"id", "version", "created_at", "updated_at", "svc_domain_id:edge_cluster_id"})
}

// ServiceDomainInfoDBO is the DB model object
type ServiceDomainInfoDBO struct {
	model.ServiceDomainEntityModelDBO
	Artifacts *json.RawMessage `json:"artifacts,omitempty" db:"artifacts"`
}

func (dbAPI *dbObjectModelAPI) initServiceDomainInfo(ctx context.Context, tx *base.WrappedTx, svcDomainID string, now time.Time) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if now.IsZero() {
		now = base.RoundedNow()
	}
	svcDomainInfoDBO := ServiceDomainInfoDBO{}
	svcDomainInfoDBO.ID = svcDomainID
	svcDomainInfoDBO.SvcDomainID = svcDomainID
	svcDomainInfoDBO.TenantID = authCtx.TenantID
	svcDomainInfoDBO.Version = float64(now.UnixNano())
	svcDomainInfoDBO.CreatedAt = now
	svcDomainInfoDBO.UpdatedAt = now
	_, err = tx.NamedExec(ctx, queryMap["CreateServiceDomainInfo"], &svcDomainInfoDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "initServiceDomainInfo: DB exec failed for %+v. Error: %s "), svcDomainInfoDBO, err.Error())
	}
	return err
}

func (dbAPI *dbObjectModelAPI) getServiceDomainInfoIDsInPage(ctx context.Context, projectID string, queryParam *model.EntitiesQueryParam, targetType model.TargetType) ([]string, []string, error) {
	return dbAPI.GetEntityIDsInPage(ctx, projectID, "", queryParam, func(ctx context.Context, svcDomainEntity *model.ServiceDomainEntityModelDBO, queryParam *model.EntitiesQueryParam) ([]string, error) {
		if svcDomainEntity.SvcDomainID != "" {
			// No need to run query to get itself
			return []string{svcDomainEntity.SvcDomainID}, nil
		}
		query, err := buildQuery(entityTypeSvcDomainInfo, queryMap["SelectServiceDomainInfoIDsTemplate"], queryParam, orderByID)
		if err != nil {
			return []string{}, err
		}
		param := ServiceDomainTypeParam{TenantID: svcDomainEntity.TenantID, SvcDomainID: svcDomainEntity.SvcDomainID, Type: targetType}
		return dbAPI.selectEntityIDsByParam(ctx, query, param)
	})
}

func (dbAPI *dbObjectModelAPI) getAllServiceDomainsInfo(ctx context.Context, projectID, svcDomainID string, req *http.Request) ([]model.ServiceDomainInfo, int, error) {
	svcDomainInfos := []model.ServiceDomainInfo{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcDomainInfos, 0, err
	}
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /servicedomains, the target type is always edge for backward compatibility
	targetType := extractServiceDomainTargetTypeQueryParam(req)
	svcDomainIDs, svcDomainIDsInPage, err := dbAPI.getServiceDomainInfoIDsInPage(ctx, projectID, queryParam, targetType)
	if err != nil {
		return svcDomainInfos, 0, err
	}

	if len(svcDomainIDsInPage) != 0 {
		query, err := buildQuery(entityTypeSvcDomainInfo, queryMap["SelectServiceDomainsInfoInTemplate"], queryParam, orderByID)
		if err != nil {
			return svcDomainInfos, 0, err
		}
		svcDomainInfoDBOs := []ServiceDomainInfoDBO{}
		err = dbAPI.QueryIn(ctx, &svcDomainInfoDBOs, query, ServiceDomainIDsParam{
			TenantID:     authCtx.TenantID,
			SvcDomainIDs: svcDomainIDsInPage,
		})
		if err != nil {
			return svcDomainInfos, 0, err
		}
		// convert svcDomainInfoDBO to svcDomainInfo
		svcDomainIDs := make([]string, 0, len(svcDomainInfoDBOs))
		for _, svcDomainInfoDBO := range svcDomainInfoDBOs {
			svcDomainInfo := model.ServiceDomainInfo{}
			err := base.Convert(&svcDomainInfoDBO, &svcDomainInfo)
			if err != nil {
				return svcDomainInfos, 0, err
			}
			svcDomainIDs = append(svcDomainIDs, svcDomainInfoDBO.ID)
			svcDomainInfos = append(svcDomainInfos, svcDomainInfo)
		}
		featuresMap, err := dbAPI.GetFeaturesForServiceDomains(ctx, svcDomainIDs)
		if err != nil {
			return svcDomainInfos, 0, err
		}
		for i := range svcDomainInfos {
			svcDomainInfoPtr := &svcDomainInfos[i]
			if features, ok := featuresMap[svcDomainInfoPtr.ID]; ok && features != nil {
				svcDomainInfoPtr.Features = *features
			} else {
				svcDomainInfoPtr.Features = model.Features{}
			}
		}
	}
	return svcDomainInfos, len(svcDomainIDs), nil
}

// getAllServiceDomainsInfoW selects all service domain infos for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) getAllServiceDomainsInfoW(ctx context.Context, projectID, svcDomainID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	svcDomainInfos, totalCount, err := dbAPI.getAllServiceDomainsInfo(ctx, projectID, svcDomainID, req)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeSvcDomainInfo})
	r := model.ServiceDomainInfoListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		SvcDomainInfoList:         svcDomainInfos,
	}
	return json.NewEncoder(w).Encode(r)
}

func (dbAPI *dbObjectModelAPI) SelectAllServiceDomainsInfoW(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getAllServiceDomainsInfoW(ctx, "", "", w, req)
}

func (dbAPI *dbObjectModelAPI) SelectAllServiceDomainsInfoForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getAllServiceDomainsInfoW(ctx, projectID, "", w, req)
}

func (dbAPI *dbObjectModelAPI) GetServiceDomainInfo(ctx context.Context, id string) (model.ServiceDomainInfo, error) {
	svcDomainInfo := model.ServiceDomainInfo{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcDomainInfo, err
	}
	tenantID := authCtx.TenantID
	svcDomainInfoDBOs := []ServiceDomainInfoDBO{}
	param := model.BaseModelDBO{TenantID: tenantID, ID: id}
	if id == "" {
		glog.Error(base.PrefixRequestID(ctx, "GetServiceDomainInfo: invalid service domain ID"))
		return svcDomainInfo, errcode.NewBadRequestError("svcDomainId")
	}
	err = dbAPI.Query(ctx, &svcDomainInfoDBOs, queryMap["SelectServiceDomainsInfo"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetServiceDomainInfo: DB select failed for service domain %s. Error: %s\n"), id, err.Error())
		return svcDomainInfo, err
	}
	if !auth.IsInfraAdminRole(authCtx) {
		entities, err := dbAPI.filterServiceDomains(ctx, svcDomainInfoDBOs)
		if err == nil {
			svcDomainInfoDBOs = entities.([]ServiceDomainInfoDBO)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "GetServiceDomainInfo: filter service domains failed for service domain %s: %s\n"), id, err.Error())
		}
	}
	if len(svcDomainInfoDBOs) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "GetServiceDomainInfo: record not found for service domain %s"), id)
		return svcDomainInfo, errcode.NewRecordNotFoundError(id)
	}
	err = base.Convert(&svcDomainInfoDBOs[0], &svcDomainInfo)
	if err != nil {
		return svcDomainInfo, err
	}
	featuresMap, err := dbAPI.GetFeaturesForServiceDomains(ctx, []string{svcDomainInfo.ID})
	if err != nil {
		return svcDomainInfo, err
	}
	if features, ok := featuresMap[svcDomainInfo.ID]; ok && features != nil {
		svcDomainInfo.Features = *features
	} else {
		svcDomainInfo.Features = model.Features{}
	}
	return svcDomainInfo, err
}

func (dbAPI *dbObjectModelAPI) GetServiceDomainInfoW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	svcDomainInfo, err := dbAPI.GetServiceDomainInfo(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, svcDomainInfo)
}

// CreateServiceDomainInfo creates a service domain info object in the DB
func (dbAPI *dbObjectModelAPI) CreateServiceDomainInfo(ctx context.Context, i interface{} /* *model.ServiceDomainInfo */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ServiceDomainInfo)
	if !ok {
		return resp, errcode.NewInternalError("CreateServiceDomainInfo: type error")
	}
	doc := *p
	tenantID := authCtx.TenantID
	doc.TenantID = tenantID
	doc.ID = doc.SvcDomainID

	now := base.RoundedNow()

	epochInNanoSecs := now.Nanosecond()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	svcDomainInfoDBO := ServiceDomainInfoDBO{}
	err = base.Convert(&doc, &svcDomainInfoDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(ctx, queryMap["CreateServiceDomainInfo"], &svcDomainInfoDBO)
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	if callback != nil {
		callback(ctx, resp)
	}
	return resp, err
}

func (dbAPI *dbObjectModelAPI) CreateServiceDomainInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateServiceDomainInfo, &model.ServiceDomainInfo{}, w, r, callback)
}

func (dbAPI *dbObjectModelAPI) UpdateServiceDomainInfo(ctx context.Context, i interface{} /* *model.ServiceDomainInfo */, callback func(context.Context, interface{}) error) (interface{}, error) {
	var createCallback func(context.Context, interface{}) error
	if callback != nil {
		createCallback = func(ctx context.Context, in interface{}) error {
			doc := in.(model.CreateDocumentResponseV2)
			return callback(ctx, model.UpdateDocumentResponseV2{ID: doc.ID})
		}
	}
	resp := model.UpdateDocumentResponseV2{}
	createResp, err := dbAPI.CreateServiceDomainInfo(ctx, i, createCallback)
	if err == nil {
		resp.ID = createResp.(model.CreateDocumentResponseV2).ID
	}
	return resp, err
}

func (dbAPI *dbObjectModelAPI) UpdateServiceDomainInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.UpdateServiceDomainInfo, &model.ServiceDomainInfo{}, w, r, callback)
}

// DeleteServiceDomainInfo deletes a service domain info object with the ID in the DB
func (dbAPI *dbObjectModelAPI) DeleteServiceDomainInfo(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return model.DeleteDocumentResponse{}, err
	}
	doc := model.BaseModelDBO{
		TenantID: authCtx.TenantID,
		ID:       id,
	}
	return DeleteEntity(ctx, dbAPI, "service_domain_info_model", "id", id, doc, callback)
}

func (dbAPI *dbObjectModelAPI) DeleteServiceDomainInfoW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteServiceDomainInfo), id, w, callback)
}

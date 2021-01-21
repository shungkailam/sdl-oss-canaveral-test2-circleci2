package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
)

const entityTypeLogCollectors = "logCollectors"

func init() {
	queryMap["SelectLogCollectorById"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND id in (:ids) %s`
	queryMap["SelectLogCollectorInfra"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND type = :type AND project_id IS NULL AND id = :id %s`
	queryMap["SelectLogCollectorsInfra"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND type = :type AND project_id IS NULL %s`
	queryMap["SelectLogCollectorProject"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND type = :type AND project_id in (:project_ids) AND id = :id %s`
	queryMap["SelectLogCollectorsProject"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND type = :type AND project_id in (:project_ids) %s`
	queryMap["SelectLogCollectorBoth"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND (project_id IS NULL OR project_id in (:project_ids)) AND id = :id %s`
	queryMap["SelectLogCollectorsBoth"] = `SELECT * FROM edge_log_collect_model WHERE tenant_id = :tenant_id AND (project_id IS NULL OR project_id in (:project_ids)) %s`
	queryMap["CreateLogCollector"] = `INSERT INTO edge_log_collect_model 
               ( id,  version,  tenant_id,  name,  project_id,  type,  cloud_creds_id,  sources,  code,  state,  dest,  aws_cloudwatch,  aws_kinesis,  gcp_stackdriver,  created_at,  updated_at) 
		VALUES (:id, :version, :tenant_id, :name, :project_id, :type, :cloud_creds_id, :sources, :code, :state, :dest, :aws_cloudwatch, :aws_kinesis, :gcp_stackdriver, :created_at, :updated_at)`
	queryMap["UpdateLogCollector"] = `UPDATE edge_log_collect_model SET 
				version = :version, name = :name, cloud_creds_id =:cloud_creds_id, sources = :sources, code = :code, state = :state, 
				dest = :dest, aws_cloudwatch = :aws_cloudwatch, aws_kinesis = :aws_kinesis, gcp_stackdriver = :gcp_stackdriver,
				updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["UpdateStateLogCollector"] = `UPDATE edge_log_collect_model SET version = :version, state = :state, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	orderByHelper.Setup(entityTypeLogCollectors, []string{"id", "name", "created_at", "updated_at", "state"})
}

type LogCollectorDBOProjects struct {
	model.LogCollector
	ProjectIds []string `json:"projectIds" db:"project_ids"`
	Ids        []string `json:"ids" db:"ids"`
}

type LogCollectorDBO struct {
	model.BaseModelDBO
	Name               string          `json:"name" db:"name"`
	CloudCredsID       string          `json:"cloudCredsID" db:"cloud_creds_id"`
	Sources            types.JSONText  `json:"sources" db:"sources"`
	Code               *string         `json:"code,omitempty" db:"code"`
	State              string          `json:"state" db:"state"`
	Type               string          `json:"type" db:"type"`
	ProjectID          *string         `json:"projectId,omitempty" db:"project_id"`
	Destination        string          `json:"dest" db:"dest"`
	CloudwatchDetails  *types.JSONText `json:"cloudwatchDetails,omitempty" db:"aws_cloudwatch"`
	KinesisDetails     *types.JSONText `json:"kinesisDetails,omitempty" db:"aws_kinesis"`
	StackdriverDetails *types.JSONText `json:"stackdriverDetails,omitempty" db:"gcp_stackdriver"`
}

// SelectAllLogCollectors return paginated information about log collectors
func (dbAPI *dbObjectModelAPI) SelectAllLogCollectors(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.LogCollector, error) {
	lcs, _, err := dbAPI.getLogCollectors(context, nil, entitiesQueryParam)
	return lcs, err
}

// SelectAllLogCollectorsW return paginated information about log collectors
func (dbAPI *dbObjectModelAPI) SelectAllLogCollectorsW(context context.Context, w io.Writer, r *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(r)

	lcs, totalCount, err := dbAPI.getLogCollectors(context, nil, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeLogCollectors}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	rsp := model.LogCollectorListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		LogCollectorList:          lcs,
	}
	return json.NewEncoder(w).Encode(rsp)
}

// GetLogCollector return information about log collector by id
func (dbAPI *dbObjectModelAPI) GetLogCollector(context context.Context, id string) (model.LogCollector, error) {
	result := model.LogCollector{}

	if len(id) == 0 {
		return result, errcode.NewBadRequestError("id")
	}

	lcs, _, err := dbAPI.getLogCollectors(context, &id, &model.EntitiesQueryParamV1{})
	if err != nil || len(lcs) != 1 {
		return result, errcode.NewRecordNotFoundError(id)
	}

	return lcs[0], nil
}

// GetLogCollectorW return information about log collector by id
func (dbAPI *dbObjectModelAPI) GetLogCollectorW(context context.Context, id string, w io.Writer, r *http.Request) error {
	lc, err := dbAPI.GetLogCollector(context, id)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(lc)
}

// CreateLogCollector creates log collector
func (dbAPI *dbObjectModelAPI) CreateLogCollector(context context.Context, i interface{} /* *model.LogCollector */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.LogCollector)
	if !ok || p == nil {
		return resp, errcode.NewInternalError("CreateLogCollector: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID

	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
	}

	if doc.Type == model.InfraCollector {
		doc.ProjectID = nil
	}

	err = dbAPI.checkPermissions(context, authContext, &doc, meta.OperationCreate)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	cc, err := dbAPI.getCloudProfileByID(context, doc.CloudCredsID)
	if err != nil {
		return resp, err
	}

	err = model.ValidateLogCollector(&doc, cc)
	if err != nil {
		return resp, err
	}

	dbo, err := ToLogCollectorDBO(&doc)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["CreateLogCollector"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating log collector for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	GetAuditlogHandler().addLogCollectorAuditLog(context, dbAPI, &doc, CREATE)

	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// CreateLogCollectorW creates log collector
func (dbAPI *dbObjectModelAPI) CreateLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateLogCollector, &model.LogCollector{}, w, r, callback)
}

func (dbAPI *dbObjectModelAPI) UpdateLogCollector(context context.Context, i interface{} /* *model.LogCollector */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}

	doc, ok := i.(*model.LogCollector)
	if !ok || doc == nil {
		return resp, errcode.NewInternalError("UpdateLogCollector: type error")
	}

	id, err := getLogCollectorId(authContext, doc)
	if err != nil {
		return resp, err
	}

	found, err := dbAPI.GetLogCollector(context, id)
	if err != nil || len(found.ID) == 0 || found.ID != id {
		return resp, errcode.NewRecordNotFoundError(id)
	}

	if doc.Type != found.Type {
		return resp, errcode.NewBadRequestError("Type")
	}
	doc.TenantID = found.TenantID

	err = dbAPI.checkPermissions(context, authContext, doc, meta.OperationUpdate)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now

	if len(doc.State) == 0 {
		doc.State = found.State
	}

	cc, err := dbAPI.getCloudProfileByID(context, doc.CloudCredsID)
	if err != nil {
		return resp, err
	}

	err = model.ValidateLogCollector(doc, cc)
	if err != nil {
		return resp, err
	}

	dbo, err := ToLogCollectorDBO(doc)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateLogCollector"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating log collector for ID %s and tenant ID %s. Error: %s"), doc.ID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	GetAuditlogHandler().addLogCollectorAuditLog(context, dbAPI, doc, UPDATE)
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateLogCollector updates and restarts log collector
func (dbAPI *dbObjectModelAPI) UpdateLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.UpdateLogCollector, &model.LogCollector{}, w, r, callback)
}

func (dbAPI *dbObjectModelAPI) UpdateStateLogCollector(context context.Context, i interface{} /* *model.LogCollector */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}

	doc, ok := i.(*model.LogCollector)
	if !ok || doc == nil {
		return resp, errcode.NewInternalError("UpdateLogCollector: type error")
	}

	id, err := getLogCollectorId(authContext, doc)
	if err != nil {
		return resp, err
	}

	found, err := dbAPI.GetLogCollector(context, id)
	if err != nil || len(found.ID) == 0 || found.ID != id {
		return resp, errcode.NewRecordNotFoundError("logCollectorID")
	}

	err = dbAPI.checkPermissions(context, authContext, &found, meta.OperationUpdate)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	found.Version = float64(epochInNanoSecs)
	found.UpdatedAt = now

	// update state
	found.State = doc.State

	cc, err := dbAPI.getCloudProfileByID(context, doc.CloudCredsID)
	if err != nil {
		return resp, err
	}

	err = model.ValidateLogCollector(&found, cc)
	if err != nil {
		return resp, err
	}

	dbo, err := ToLogCollectorDBO(&found)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateStateLogCollector"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating log collector for ID %s and tenant ID %s. Error: %s"), doc.ID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(found.ID, err)
	}
	if callback != nil {
		go callback(context, found)
	}
	resp.ID = doc.ID
	return resp, nil
}

// StartLogCollectorW starts log collector by id
func (dbAPI *dbObjectModelAPI) StartLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateStateLogCollector, &model.LogCollector{State: model.LogCollectorActive}, w, r, callback)
}

// StopLogCollectorW starts log collector by id
func (dbAPI *dbObjectModelAPI) StopLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateStateLogCollector, &model.LogCollector{State: model.LogCollectorStopped}, w, r, callback)
}

// DeleteLogCollector deletes log collector by id
func (dbAPI *dbObjectModelAPI) DeleteLogCollector(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}

	found, err := dbAPI.GetLogCollector(context, id)
	if err != nil || len(found.ID) == 0 || found.ID != id {
		return resp, errcode.NewRecordNotFoundError("logCollectorID")
	}

	err = dbAPI.checkPermissions(context, authContext, &found, meta.OperationDelete)
	if err != nil {
		return resp, err
	}

	doc := model.LogCollector{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}
	entity, err := DeleteEntity(context, dbAPI, "edge_log_collect_model", "id", id, doc, callback)
	if err == nil {
		GetAuditlogHandler().addLogCollectorAuditLog(context, dbAPI, &doc, DELETE)
	}
	return entity, err
}

// DeleteLogCollectorW deletes log collector by id
func (dbAPI *dbObjectModelAPI) DeleteLogCollectorW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteLogCollector, id, w, callback)
}

func (dbAPI *dbObjectModelAPI) getLogCollectors(context context.Context, id *string, entitiesQueryParam base.QueryParameter) ([]model.LogCollector, int, error) {
	logCollectors := []model.LogCollector{}
	logCollectorDBOs := []LogCollectorDBO{}

	queryParams, err := dbAPI.getLogCollectorsQueryParams(context, id)
	if err != nil {
		return logCollectors, 0, err
	}
	if queryParams.Key == "" {
		return logCollectors, 0, errcode.NewInvalidCredentialsError()
	}

	query, err := buildQuery(entityTypeLogCollectors, queryMap[queryParams.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return logCollectors, 0, err
	}

	if queryParams.InQuery {
		err = dbAPI.QueryIn(context, &logCollectorDBOs, query, queryParams.Param)
	} else {
		err = dbAPI.Query(context, &logCollectorDBOs, query, queryParams.Param)
	}

	// convert
	for _, dbo := range logCollectorDBOs {
		lc, err := FromLogCollectorDBO(&dbo)
		if err != nil {
			return logCollectors, 0, err
		}
		logCollectors = append(logCollectors, lc)
	}
	return logCollectors, 0, err
}

func (dbAPI *dbObjectModelAPI) getLogCollectorsByIds(context context.Context, tenantId string, IDs []string) ([]model.LogCollector, error) {
	logCollectors := []model.LogCollector{}
	logCollectorDBOs := []LogCollectorDBO{}

	queryParams := base.InQueryParam{
		Param: LogCollectorDBOProjects{
			LogCollector: model.LogCollector{
				BaseModel: model.BaseModel{
					TenantID: tenantId,
				},
			},
			Ids: IDs,
		},
		Key:     "SelectLogCollectorById",
		InQuery: true,
	}

	query, err := buildQuery(entityTypeDataStream, queryMap[queryParams.Key], &model.EntitiesQueryParamV1{}, orderByNameID)
	if err != nil {
		return logCollectors, err
	}

	err = dbAPI.QueryIn(context, &logCollectorDBOs, query, queryParams.Param)

	// convert
	for _, dbo := range logCollectorDBOs {
		lc, err := FromLogCollectorDBO(&dbo)
		if err != nil {
			return logCollectors, err
		}
		logCollectors = append(logCollectors, lc)
	}
	return logCollectors, err
}

func (dbAPI *dbObjectModelAPI) getLogCollectorsQueryParams(context context.Context, id *string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}

	var baseModel model.BaseModel
	var isMultiple bool
	if id != nil && len(*id) != 0 {
		baseModel = model.BaseModel{TenantID: authContext.TenantID, ID: *id}
		isMultiple = false
	} else {
		baseModel = model.BaseModel{TenantID: authContext.TenantID}
		isMultiple = true
	}

	var query string
	param := model.LogCollector{BaseModel: baseModel}

	projectIDs := auth.GetProjectIDs(authContext)
	canSeeInfra := auth.IsInfraAdminRole(authContext)
	if canSeeInfra && len(projectIDs) != 0 {
		if isMultiple {
			query = "SelectLogCollectorsBoth"
		} else {
			query = "SelectLogCollectorBoth"
		}

		return base.InQueryParam{
			Param: LogCollectorDBOProjects{
				LogCollector: param,
				ProjectIds:   projectIDs,
			},
			Key:     query,
			InQuery: true,
		}, nil
	} else if canSeeInfra {
		if isMultiple {
			query = "SelectLogCollectorsInfra"
		} else {
			query = "SelectLogCollectorInfra"
		}

		param.Type = model.InfraCollector
		return base.InQueryParam{
			Param: LogCollectorDBOProjects{
				LogCollector: param,
			},
			Key:     query,
			InQuery: false,
		}, nil
	} else {
		if isMultiple {
			query = "SelectLogCollectorsProject"
		} else {
			query = "SelectLogCollectorProject"
		}

		param.Type = model.ProjectCollector
		return base.InQueryParam{
			Param: LogCollectorDBOProjects{
				LogCollector: param,
				ProjectIds:   projectIDs,
			},
			Key:     query,
			InQuery: true,
		}, nil
	}
}

func (dbAPI *dbObjectModelAPI) checkPermissions(context context.Context, authContext *base.AuthContext, lc *model.LogCollector, operation meta.Operation) error {
	if lc.TenantID != authContext.TenantID {
		return errcode.NewPermissionDeniedError("RBAC/TenantId")
	}

	if lc.Type == model.InfraCollector {
		return auth.CheckRBAC(
			authContext,
			meta.EntityInfraLogCollector,
			operation,
			auth.RbacContext{})
	}

	if lc.ProjectID == nil {
		return errcode.NewBadRequestError("ProjectID")
	}

	if !auth.IsProjectMember(*lc.ProjectID, authContext) {
		return errcode.NewPermissionDeniedError("RBAC/ProjectID")
	}

	return auth.CheckRBAC(
		authContext,
		meta.EntityLogCollector,
		operation,
		auth.RbacContext{
			ProjectID:  *lc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
}

func ToLogCollectorDBO(doc *model.LogCollector) (LogCollectorDBO, error) {
	dbo := LogCollectorDBO{}
	err := base.Convert(doc, &dbo)
	return dbo, err
}

func FromLogCollectorDBO(dbo *LogCollectorDBO) (model.LogCollector, error) {
	lc := model.LogCollector{}
	err := base.Convert(dbo, &lc)
	return lc, err
}

func getLogCollectorId(authContext *base.AuthContext, doc *model.LogCollector) (string, error) {
	var id string
	if len(authContext.ID) > 0 {
		id = authContext.ID
	} else {
		id = doc.ID
	}
	if len(id) == 0 {
		return id, errcode.NewBadRequestError("id")
	}
	return id, nil
}

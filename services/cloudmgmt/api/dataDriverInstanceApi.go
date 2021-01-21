package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"

	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
)

const entityTypeDataDriverInstance = "dataDriverInstance"

// DataDriverInstanceDBO is the DB model for Data Driver Instance
type DataDriverInstanceDBO struct {
	model.BaseModelDBO

	Name              string          `json:"name" db:"name" validate:"range=1:200"`
	Description       string          `json:"description,omitempty" db:"description" validate:"range=1:200"`
	DataDriverClassID string          `json:"dataDriverClassID" db:"data_driver_class_id" validate:"range=1:200"`
	ProjectID         string          `json:"projectId" db:"project_id" validate:"range=0:64"`
	StaticParameters  *types.JSONText `json:"staticParameters,omitempty" db:"parameters"`
}

type instanceIdFilter struct {
	TenantID   string   `db:"tenant_id"`
	IDs        []string `db:"ids"`
	ProjectIDs []string `db:"project_ids"`
}

func init() {
	queryMap["CreateDataDriverInstance"] = `INSERT INTO data_driver_instance_model 
               ( id,  version,  tenant_id,  name,  description,  data_driver_class_id,  project_id,  parameters,  created_at,  updated_at)
        VALUES (:id, :version, :tenant_id, :name, :description, :data_driver_class_id, :project_id, :parameters, :created_at, :updated_at)`
	queryMap["UpdateDataDriverInstance"] = `UPDATE data_driver_instance_model SET 
				version = :version, name = :name, description = :description, parameters = :parameters, 
				updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	queryMap["SelectDataDriverInstanceByClassId"] = `SELECT * FROM data_driver_instance_model 
		WHERE tenant_id = :tenant_id AND data_driver_class_id in (:ids)`
	queryMap["SelectDataDriverInstanceByClassIdInProjects"] = `SELECT * FROM data_driver_instance_model 
		WHERE tenant_id = :tenant_id AND data_driver_class_id in (:ids) AND project_id in (:project_ids)`
	queryMap["SelectDataDriverInstanceById"] = `SELECT * FROM data_driver_instance_model 
		WHERE tenant_id = :tenant_id AND id in (:ids)`

	queryMap["SelectDataDriverInstances"] = `SELECT *, count(*) OVER() as total_count FROM data_driver_instance_model 
		WHERE tenant_id = :tenant_id %s`
	queryMap["SelectDataDriverInstancesInProjects"] = `SELECT *, count(*) OVER() as total_count FROM data_driver_instance_model 
		WHERE tenant_id = :tenant_id AND project_id in (:ids) %s`

	orderByHelper.Setup(entityTypeDataDriverInstance, []string{"id", "name", "project_id", "data_driver_class_id", "created_at", "updated_at"})
}

func ToDataDriverInstanceDBO(doc *model.DataDriverInstance) (DataDriverInstanceDBO, error) {
	dbo := DataDriverInstanceDBO{}
	err := base.Convert(doc, &dbo)
	return dbo, err
}

func FromDataDriverInstanceDBO(dbo *DataDriverInstanceDBO) (model.DataDriverInstance, error) {
	ddc := model.DataDriverInstance{}
	err := base.Convert(dbo, &ddc)
	return ddc, err
}

func convertDataDriverInstanceDBOs(dbos []DataDriverInstanceDBO) ([]model.DataDriverInstance, error) {
	dataDriverInstances := []model.DataDriverInstance{}
	for _, dbo := range dbos {
		d, err := FromDataDriverInstanceDBO(&dbo)
		if err != nil {
			return nil, err
		}
		dataDriverInstances = append(dataDriverInstances, d)
	}
	return dataDriverInstances, nil
}

func (dbAPI *dbObjectModelAPI) findDataDriverInstanceDBOs(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]DataDriverInstanceDBO, error) {
	dataDriverInstanceDBOs := []DataDriverInstanceDBO{}

	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID

	var queryParam base.InQueryParam
	if auth.IsInfraAdminRole(authContext) {
		queryParam = base.InQueryParam{
			Param: tenantIdFilter{
				TenantID: tenantID,
			},
			Key:     "SelectDataDriverInstances",
			InQuery: false,
		}
	} else {
		projectIDs := auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return dataDriverInstanceDBOs, nil
		}
		queryParam = base.InQueryParam{
			Param: tenantIdFilter{
				TenantID: tenantID,
				IDs:      projectIDs,
			},
			Key:     "SelectDataDriverInstancesInProjects",
			InQuery: true,
		}
	}

	var sqlQuery string
	if entitiesQueryParam == nil {
		sqlQuery = fmt.Sprintf(queryMap[queryParam.Key], "")
	} else {
		sqlQuery, err = buildLimitQuery(entityTypeDataDriverInstance, queryMap[queryParam.Key], entitiesQueryParam, orderByNameID)
		if err != nil {
			return nil, err
		}
	}

	if queryParam.InQuery {
		err = dbAPI.QueryIn(context, &dataDriverInstanceDBOs, sqlQuery, queryParam.Param)
	} else {
		err = dbAPI.Query(context, &dataDriverInstanceDBOs, sqlQuery, queryParam.Param)
	}
	return dataDriverInstanceDBOs, err
}

func (dbAPI *dbObjectModelAPI) findDataDriverInstanceDBOsByClassId(context context.Context, id string) ([]DataDriverInstanceDBO, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID

	dataDriverInstanceDBOs := []DataDriverInstanceDBO{}
	if auth.IsInfraAdminOrEdgeRole(authContext) {
		err = dbAPI.QueryIn(context, &dataDriverInstanceDBOs, queryMap["SelectDataDriverInstanceByClassId"], tenantIdFilter{
			TenantID: tenantID,
			IDs:      []string{id},
		})
	} else {
		projectIDs := auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return dataDriverInstanceDBOs, nil
		}
		err = dbAPI.QueryIn(context, &dataDriverInstanceDBOs, queryMap["SelectDataDriverInstanceByClassIdInProjects"], instanceIdFilter{
			TenantID:   tenantID,
			IDs:        []string{id},
			ProjectIDs: projectIDs,
		})
	}

	return dataDriverInstanceDBOs, err
}

func (dbAPI *dbObjectModelAPI) getDataDriverInstances(context context.Context, tenantId string, ids []string) ([]model.DataDriverInstance, error) {
	dataDriverInstance := []model.DataDriverInstance{}
	dataDriverInstanceDBOs := []DataDriverInstanceDBO{}

	if len(ids) == 0 {
		return dataDriverInstance, nil
	}

	err := dbAPI.QueryIn(context, &dataDriverInstanceDBOs, queryMap["SelectDataDriverInstanceById"], tenantIdFilter{
		TenantID: tenantId,
		IDs:      ids,
	})
	if err != nil {
		return nil, err
	}

	for _, ddc := range dataDriverInstanceDBOs {
		d, err := FromDataDriverInstanceDBO(&ddc)
		if err != nil {
			return nil, err
		}
		dataDriverInstance = append(dataDriverInstance, d)
	}
	return dataDriverInstance, err
}

func (dbAPI *dbObjectModelAPI) hasDataDriverInstancesForClassId(context context.Context, dataDriverClassId string) (bool, error) {
	instances, err := dbAPI.findDataDriverInstanceDBOsByClassId(context, dataDriverClassId)
	if err != nil {
		return false, err
	}
	return len(instances) > 0, nil
}

func (dbAPI *dbObjectModelAPI) getDataDriverInstanceInventoryByIds(context context.Context, tenantId string, IDs []string) ([]model.DataDriverInstanceInventory, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	instances, err := dbAPI.getDataDriverInstances(context, tenantId, IDs)
	if err != nil {
		return nil, err
	}

	dataDriverClassesIDs := []string{}
	for _, i := range instances {
		dataDriverClassesIDs = append(dataDriverClassesIDs, i.DataDriverClassID)
	}
	dataDriverClassesIDs = base.Unique(dataDriverClassesIDs)

	classes, err := dbAPI.getDataDriverClasses(context, dataDriverClassesIDs)
	if err != nil {
		return nil, err
	}

	dataDriverClasses := make(map[string]model.DataDriverClass)
	for _, c := range classes {
		dataDriverClasses[c.ID] = c
	}

	inventory := make([]model.DataDriverInstanceInventory, 0, len(instances))
	for _, i := range instances {
		configs, _, err := dbAPI.SelectDataDriverConfigsByInstanceId(context, i.ID, &model.EntitiesQueryParam{})
		if err != nil {
			return nil, err
		}
		streams, _, err := dbAPI.SelectDataDriverStreamsByInstanceId(context, i.ID, &model.EntitiesQueryParam{})
		if err != nil {
			return nil, err
		}
		inv := NewDataDriverInstanceInventory(dataDriverClasses[i.DataDriverClassID], i, configs, streams)
		err = inv.RenderForContext(authContext, dbAPI)
		if err != nil {
			return nil, err
		}
		inventory = append(inventory, *inv.DataDriverInstanceInventory)
	}

	return inventory, nil
}

// SelectAllDataDriverInstances return paginated information about data driver instances with specific filter
func (dbAPI *dbObjectModelAPI) SelectAllDataDriverInstances(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverInstance, int, error) {
	dbos, err := dbAPI.findDataDriverInstanceDBOs(context, entitiesQueryParam)
	if err != nil {
		return nil, 0, err
	}

	if len(dbos) == 0 {
		return []model.DataDriverInstance{}, 0, nil
	}

	res, err := convertDataDriverInstanceDBOs(dbos)
	return res, *(dbos[0].TotalCount), err
}

// SelectAllDataDriverInstancesW return paginated information about data driver instances with specific filter
func (dbAPI *dbObjectModelAPI) SelectAllDataDriverInstancesW(context context.Context, w io.Writer, r *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(r)
	instances, totalCount, err := dbAPI.SelectAllDataDriverInstances(context, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeDataDriverInstance})
	resp := model.DataDriverInstanceListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		ListOfDetaDriverInstances: instances,
	}
	return base.DispatchPayload(w, resp)
}

// SelectAllDataDriverInstancesByClassId return paginated information about data driver instances with specific filter
func (dbAPI *dbObjectModelAPI) SelectAllDataDriverInstancesByClassId(context context.Context, id string) ([]model.DataDriverInstance, error) {
	dbos, err := dbAPI.findDataDriverInstanceDBOsByClassId(context, id)
	if err != nil {
		return nil, err
	}
	return convertDataDriverInstanceDBOs(dbos)
}

// SelectAllDataDriverInstancesByClassIdW return information about data driver instances with specific filter
func (dbAPI *dbObjectModelAPI) SelectAllDataDriverInstancesByClassIdW(context context.Context, id string, w io.Writer, r *http.Request) error {
	instances, err := dbAPI.SelectAllDataDriverInstancesByClassId(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, instances)
}

// GetDataDriverInstance get a data driver instance object in the DB
func (dbAPI *dbObjectModelAPI) GetDataDriverInstance(context context.Context, id string) (model.DataDriverInstance, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return model.DataDriverInstance{}, err
	}
	tenantID := authContext.TenantID

	dataDriverInstances, err := dbAPI.getDataDriverInstances(context, tenantID, []string{id})
	if err != nil {
		return model.DataDriverInstance{}, err
	}
	if len(dataDriverInstances) != 1 {
		return model.DataDriverInstance{}, errcode.NewRecordNotFoundError(id)
	}

	instance := dataDriverInstances[0]

	if !auth.IsInfraAdminOrEdgeRole(authContext) {
		if !auth.IsProjectMember(instance.ProjectID, authContext) {
			return model.DataDriverInstance{}, errcode.NewPermissionDeniedError("RBAC/ProjectID")
		}
	}
	return instance, nil
}

// GetDataDriverInstanceW gget a data driver instance object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataDriverInstanceW(context context.Context, dataDriverInstanceID string, w io.Writer, r *http.Request) error {
	project, err := dbAPI.GetDataDriverInstance(context, dataDriverInstanceID)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, project)
}

// CreateDataDriverInstance creates a data driver instance object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataDriverInstance(context context.Context, i interface{} /* *model.DataDriverInstance */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID

	if !auth.IsInfraAdminRole(authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC")
	}

	p, ok := i.(*model.DataDriverInstance)
	if !ok {
		return resp, errcode.NewInternalError("CreateDataDriverInstance: type error")
	}
	doc := *p
	doc.TenantID = tenantID

	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateDataDriverInstance doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateDataDriverInstance doc.ID was invalid, update it to %s\n"), doc.ID)
	}

	_, err = dbAPI.GetProject(context, doc.ProjectID)
	if err != nil {
		return resp, err
	}

	if !auth.IsProjectMember(doc.ProjectID, authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC/ProjectID")
	}

	ddc, err := dbAPI.GetDataDriverClass(context, doc.DataDriverClassID)
	if err != nil || len(ddc.ID) == 0 {
		return resp, errcode.NewRecordNotFoundError(doc.DataDriverClassID)
	}

	err = model.ValidateDataDriverInstance(&doc, ddc.StaticParameterSchema)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	dbo, err := ToDataDriverInstanceDBO(&doc)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["CreateDataDriverInstance"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating data driver instance for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addDataDriverInstanceAuditLog(context, dbAPI, &doc, CREATE)
	return resp, nil
}

// CreateDataDriverInstanceW creates a data driver instance object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataDriverInstanceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDataDriverInstance, &model.DataDriverInstance{}, w, r, callback)
}

// UpdateDataDriverInstance updates a data driver instance object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataDriverInstance(context context.Context, i interface{} /* *model.DataDriverInstance */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC")
	}

	p, ok := i.(*model.DataDriverInstance)
	if !ok {
		return resp, errcode.NewInternalError("UpdateDataDriverInstance: type error")
	}
	doc := *p

	doc.TenantID = authContext.TenantID
	now := base.RoundedNow()
	doc.Version = float64(now.UnixNano())
	doc.UpdatedAt = now

	var id string
	if len(authContext.ID) > 0 {
		id = authContext.ID
	} else {
		id = doc.ID
	}
	if len(id) == 0 {
		return resp, errcode.NewBadRequestError("id")
	}
	doc.ID = id

	found, err := dbAPI.GetDataDriverInstance(context, doc.ID)
	if err != nil {
		return resp, err
	}

	if !auth.IsProjectMember(found.ProjectID, authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC/ProjectID")
	}

	ddc, err := dbAPI.GetDataDriverClass(context, found.DataDriverClassID)
	if err != nil {
		return resp, err
	}
	if err != nil || len(found.ID) == 0 || found.ID != doc.ID {
		return nil, errcode.NewRecordNotFoundError(doc.ID)
	}

	err = model.ValidateDataDriverInstance(&doc, ddc.StaticParameterSchema)
	if err != nil {
		return resp, err
	}

	dbo, err := ToDataDriverInstanceDBO(&doc)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateDataDriverInstance"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating data driver instance for ID %s and tenant ID %s. Error: %s"), doc.ID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(found.ID, err)
	}
	GetAuditlogHandler().addDataDriverInstanceAuditLog(context, dbAPI, &doc, UPDATE)

	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateDataDriverInstanceW updates a data driver instance object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDataDriverInstanceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDataDriverInstance, &model.DataDriverInstance{}, w, r, callback)
}

// DeleteDataDriverInstance delete a data driver instance object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDataDriverInstance(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return nil, errcode.NewPermissionDeniedError("RBAC")
	}

	ddc, err := dbAPI.GetDataDriverInstance(context, id)
	if err != nil || len(ddc.ID) == 0 || ddc.ID != id {
		return nil, errcode.NewRecordNotFoundError(id)
	}

	if !auth.IsProjectMember(ddc.ProjectID, authContext) {
		return nil, errcode.NewPermissionDeniedError("RBAC/ProjectID")
	}

	inUse, err := dbAPI.hasDataDriverConfigsForInstanceId(context, ddc.ID)
	if err != nil {
		return nil, err
	}
	if inUse {
		msg := fmt.Sprintf("data driver instance %s is in use by data driver configs. cannot remove this data driver instance", id)
		return nil, errcode.NewPreConditionFailedError(msg)
	}

	inUse, err = dbAPI.hasDataDriverStreamsForInstanceId(context, ddc.ID)
	if err != nil {
		return nil, err
	}
	if inUse {
		msg := fmt.Sprintf("data driver instance %s is in use by data driver streams. cannot remove this data driver instance", id)
		return nil, errcode.NewPreConditionFailedError(msg)
	}

	entity, err := DeleteEntityV2(context, dbAPI, "data_driver_instance_model", "id", id, ddc, callback)
	if err == nil {
		GetAuditlogHandler().addDataDriverInstanceAuditLog(context, dbAPI, &ddc, DELETE)
	}
	return entity, err
}

// DeleteDataDriverInstanceW delete a data driver instance object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDataDriverInstanceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDataDriverInstance, id, w, callback)
}

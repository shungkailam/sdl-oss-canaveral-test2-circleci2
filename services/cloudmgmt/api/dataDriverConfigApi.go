package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"

	"context"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
)

const entityTypeDataDriverConfig = "dataDriverConfig"

type DataDriverParamsDBO struct {
	model.BaseModelDBO
	Name                 string          `json:"name" db:"name" validate:"range=1:200"`
	Description          string          `json:"description,omitempty" db:"description" validate:"range=1:200"`
	DataDriverInstanceID string          `json:"dataDriverInstanceID" db:"instance_id" validate:"range=1:200"`
	Parameters           *types.JSONText `json:"parameters,omitempty" db:"parameters"`
	Direction            string          `json:"direction,omitempty"  db:"direction"`
	Type                 string          `db:"type"`
}

type paramIdFilter struct {
	Type     string   `db:"type"`
	TenantID string   `db:"tenant_id"`
	IDs      []string `db:"ids"`
}

func init() {
	queryMap["CreateDataDriverParam"] = `INSERT INTO data_driver_params_model 
               ( id,  version,  tenant_id,  name,  description,  instance_id,  direction,  type,  parameters,  created_at,  updated_at)
        VALUES (:id, :version, :tenant_id, :name, :description, :instance_id, :direction, :type, :parameters, :created_at, :updated_at)`

	queryMap["UpdateDataDriverParams"] = `UPDATE data_driver_params_model SET 
				version = :version, name = :name, description = :description, parameters = :parameters, 
				updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	queryMap["SelectDataDriverParamsByIds"] = `SELECT * FROM data_driver_params_model 
		WHERE tenant_id = :tenant_id AND id in (:ids) AND type = :type`
	queryMap["SelectDataDriverParamsByInstanceId"] = `SELECT *, count(*) OVER() as total_count FROM data_driver_params_model 
		WHERE tenant_id = :tenant_id AND instance_id in (:ids) AND type = :type %s`

	orderByHelper.Setup(entityTypeDataDriverConfig, []string{"id", "name", "instance_id", "created_at", "updated_at"})
}

func ToDataDriverParamsDBOFromConfig(doc *model.DataDriverConfig) (DataDriverParamsDBO, error) {
	dbo := DataDriverParamsDBO{}
	err := base.Convert(doc, &dbo)
	dbo.Type = entityTypeDataDriverConfig
	return dbo, err
}

func FromDataDriverParamsDBOToConfig(dbo *DataDriverParamsDBO, binding *model.ServiceDomainBinding) (model.DataDriverConfig, error) {
	ddc := model.DataDriverConfig{}
	if dbo.Type != entityTypeDataDriverConfig {
		return ddc, errcode.NewBadRequestError("Type")
	}
	err := base.Convert(dbo, &ddc)
	ddc.ServiceDomainBinding = *binding
	return ddc, err
}

func (dbAPI *dbObjectModelAPI) loadDatadriverParamById(context context.Context, id string, paramType string) (DataDriverParamsDBO, error) {
	params, err := dbAPI.loadDatadriverParamsByIds(context, []string{id}, paramType)
	if err != nil {
		return DataDriverParamsDBO{}, err
	}
	if len(params) != 1 {
		return DataDriverParamsDBO{}, errcode.NewRecordNotFoundError(id)
	}
	return params[0], nil
}

func (dbAPI *dbObjectModelAPI) loadDatadriverParamsByIds(context context.Context, ids []string, paramType string) ([]DataDriverParamsDBO, error) {
	dbos := []DataDriverParamsDBO{}
	if len(ids) == 0 {
		return dbos, nil
	}

	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dbos, err
	}
	tenantID := authContext.TenantID

	err = dbAPI.QueryIn(context, &dbos, queryMap["SelectDataDriverParamsByIds"], paramIdFilter{
		TenantID: tenantID,
		IDs:      ids,
		Type:     paramType,
	})
	if err != nil {
		return nil, err
	}

	instanceIDMap := map[string]bool{}
	for _, dbo := range dbos {
		instanceIDMap[dbo.DataDriverInstanceID] = true
	}

	instanceIDs := []string{}
	for k, _ := range instanceIDMap {
		instanceIDs = append(instanceIDs, k)
	}

	// Check if we can read instances
	instances, err := dbAPI.getDataDriverInstances(context, tenantID, instanceIDs)
	if err != nil {
		return nil, err
	}

	for _, instance := range instances {
		if !auth.IsProjectMember(instance.ProjectID, authContext) {
			return nil, errcode.NewPermissionDeniedError("RBAC/ProjectID")
		}
	}

	return dbos, nil
}

func (dbAPI *dbObjectModelAPI) loadDatadriverParamsByInstanceId(context context.Context, instanceId string, paramType string, entitiesQueryParam *model.EntitiesQueryParam) ([]DataDriverParamsDBO, error) {
	dbos := []DataDriverParamsDBO{}

	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dbos, err
	}
	tenantID := authContext.TenantID

	// Check if we can read instances
	_, err = dbAPI.GetDataDriverInstance(context, instanceId)
	if err != nil {
		return dbos, err
	}

	query, err := buildLimitQuery(entityTypeDataDriverConfig, queryMap["SelectDataDriverParamsByInstanceId"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return dbos, err
	}
	err = dbAPI.QueryIn(context, &dbos, query, paramIdFilter{
		TenantID: tenantID,
		IDs:      []string{instanceId},
		Type:     paramType,
	})
	if err != nil {
		return nil, err
	}

	return dbos, nil
}

func (dbAPI *dbObjectModelAPI) createDataDriverParam(context context.Context, tx *base.WrappedTx, doc *DataDriverParamsDBO) (*DataDriverParamsDBO, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return doc, err
	}
	tenantID := authContext.TenantID

	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateDataDriverConfig doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateDataDriverConfig doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	doc.TenantID = tenantID

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	_, err = tx.NamedExec(context, queryMap["CreateDataDriverParam"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating data driver instance for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return doc, errcode.TranslateDatabaseError(doc.ID, err)
	}

	return doc, nil
}

func (dbAPI *dbObjectModelAPI) updateDataDriverParam(context context.Context, tx *base.WrappedTx, doc *DataDriverParamsDBO) (*DataDriverParamsDBO, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID

	doc.TenantID = tenantID
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
		return nil, errcode.NewBadRequestError("id")
	}
	doc.ID = id

	_, err = tx.NamedExec(context, queryMap["UpdateDataDriverParams"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating data driver instance for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return doc, errcode.TranslateDatabaseError(doc.ID, err)
	}

	return doc, nil
}

func (dbAPI *dbObjectModelAPI) deleteDataDriverParam(context context.Context, tx *base.WrappedTx, id string) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}

	res, err := base.DeleteTxn(context, tx, "data_driver_params_model", map[string]interface{}{"id": id, "tenant_id": authContext.TenantID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "failed to remove data driver param %s. Error: %s"), id, err.Error())
		return err
	}

	if !base.IsDeleteSuccessful(res) {
		return errcode.NewDatabaseDependencyError(id)
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) hasDataDriverConfigsForInstanceId(context context.Context, instanceId string) (bool, error) {
	params, err := dbAPI.loadDatadriverParamsByInstanceId(context, instanceId, entityTypeDataDriverConfig, nil)
	return len(params) > 0, err
}

// SelectDataDriverConfigsByInstance return all data driver instance config, write output into writer
func (dbAPI *dbObjectModelAPI) SelectDataDriverConfigsByInstanceId(context context.Context, instanceId string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverConfig, int, error) {
	params, err := dbAPI.loadDatadriverParamsByInstanceId(context, instanceId, entityTypeDataDriverConfig, entitiesQueryParam)
	if err != nil {
		return nil, 0, err
	}

	paramIds := make([]string, len(params))
	for _, p := range params {
		paramIds = append(paramIds, p.ID)
	}

	bindings, err := NewDataDriverConfigBinding(context, dbAPI).GetAll(paramIds)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.DataDriverConfig, 0, len(params))
	for _, p := range params {
		binding := bindings[p.ID]
		cfg, err := FromDataDriverParamsDBOToConfig(&p, binding)
		if err != nil {
			return nil, 0, err
		}

		res = append(res, cfg)
	}

	if len(res) == 0 {
		return res, 0, nil
	}

	return res, *params[0].TotalCount, nil
}

// SelectDataDriverConfigsByInstanceW return all data driver instance config, write output into writer
func (dbAPI *dbObjectModelAPI) SelectDataDriverConfigsByInstanceIdW(context context.Context, id string, w io.Writer, r *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(r)
	configs, totalCount, err := dbAPI.SelectDataDriverConfigsByInstanceId(context, id, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeDataDriverConfig})
	resp := model.DataDriverConfigListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		ListOfDataDriverConfigs:   configs,
	}
	return base.DispatchPayload(w, &resp)
}

// GetDataDriverConfig get a data driver config object in the DB
func (dbAPI *dbObjectModelAPI) GetDataDriverConfig(context context.Context, id string) (model.DataDriverConfig, error) {
	params, err := dbAPI.loadDatadriverParamById(context, id, entityTypeDataDriverConfig)
	if err != nil {
		return model.DataDriverConfig{}, err
	}

	binding, err := NewDataDriverConfigBinding(context, dbAPI).Get(id)
	if err != nil {
		return model.DataDriverConfig{}, err
	}
	return FromDataDriverParamsDBOToConfig(&params, binding)
}

// GetDataDriverConfigW gget a data driver config object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataDriverConfigW(context context.Context, dataDriverConfigID string, w io.Writer, r *http.Request) error {
	project, err := dbAPI.GetDataDriverConfig(context, dataDriverConfigID)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, project)
}

// CreateDataDriverConfig creates a data driver config object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataDriverConfig(context context.Context, i interface{} /* *model.DataDriverConfig */, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	p, ok := i.(*model.DataDriverConfig)
	if !ok {
		return nil, errcode.NewInternalError("CreateDataDriverConfig: type error")
	}

	instance, err := dbAPI.GetDataDriverInstance(context, p.DataDriverInstanceID)
	if err != nil {
		return nil, err
	}
	class, err := dbAPI.GetDataDriverClass(context, instance.DataDriverClassID)
	if err != nil {
		return nil, err
	}
	project, err := dbAPI.GetProject(context, instance.ProjectID)
	if err != nil {
		return nil, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataDriverConfig,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  instance.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return nil, err
	}

	err = model.ValidateDataDriverConfig(p, &class.ConfigParameterSchema, &project)
	if err != nil {
		return nil, err
	}

	err = dbAPI.cleanupServiceDomainBinding(context, &project, &p.ServiceDomainBinding)
	if err != nil {
		return nil, err
	}

	var doc *DataDriverParamsDBO
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		param, err := ToDataDriverParamsDBOFromConfig(p)
		if err != nil {
			return err
		}
		doc, err = dbAPI.createDataDriverParam(context, tx, &param)
		if err != nil {
			return err
		}
		err = NewDataDriverConfigBinding(context, dbAPI).Set(tx, doc.ID, &p.ServiceDomainBinding)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if callback != nil {
		go callback(context, doc)
	}

	GetAuditlogHandler().addDataDriverConfigAuditLog(context, dbAPI, doc, instance.ProjectID, CREATE)
	return model.CreateDocumentResponseV2{ID: doc.ID}, nil
}

// CreateDataDriverConfigW creates a data driver config object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataDriverConfigW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDataDriverConfig, &model.DataDriverConfig{}, w, r, callback)
}

// UpdateDataDriverConfig updates a data driver config object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataDriverConfig(context context.Context, i interface{} /* *model.DataDriverConfig */, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	p, ok := i.(*model.DataDriverConfig)
	if !ok {
		return nil, errcode.NewInternalError("UpdateDataDriverConfig: type error")
	}
	var id string
	if len(authContext.ID) > 0 {
		id = authContext.ID
	} else {
		id = p.ID
	}
	if len(id) == 0 {
		return nil, errcode.NewBadRequestError("id")
	}
	p.ID = id

	param, err := dbAPI.loadDatadriverParamById(context, p.ID, entityTypeDataDriverConfig)
	if err != nil {
		return nil, err
	}
	instance, err := dbAPI.GetDataDriverInstance(context, param.DataDriverInstanceID)
	if err != nil {
		return nil, err
	}
	class, err := dbAPI.GetDataDriverClass(context, instance.DataDriverClassID)
	if err != nil {
		return nil, err
	}
	project, err := dbAPI.GetProject(context, instance.ProjectID)
	if err != nil {
		return nil, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataDriverConfig,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:  instance.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return nil, err
	}

	err = model.ValidateDataDriverConfig(p, &class.ConfigParameterSchema, &project)
	if err != nil {
		return nil, err
	}

	err = dbAPI.cleanupServiceDomainBinding(context, &project, &p.ServiceDomainBinding)
	if err != nil {
		return nil, err
	}

	var doc *DataDriverParamsDBO
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		param, err := ToDataDriverParamsDBOFromConfig(p)
		if err != nil {
			return err
		}
		doc, err = dbAPI.updateDataDriverParam(context, tx, &param)
		if err != nil {
			return err
		}
		err = NewDataDriverConfigBinding(context, dbAPI).Set(tx, doc.ID, &p.ServiceDomainBinding)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if callback != nil {
		go callback(context, doc)
	}

	GetAuditlogHandler().addDataDriverConfigAuditLog(context, dbAPI, doc, instance.ProjectID, UPDATE)
	return model.UpdateDocumentResponseV2{ID: doc.ID}, nil
}

// UpdateDataDriverConfigW updates a data driver config object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDataDriverConfigW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDataDriverConfig, &model.DataDriverConfig{}, w, r, callback)
}

// DeleteDataDriverConfig delete a data driver config object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDataDriverConfig(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	param, err := dbAPI.loadDatadriverParamById(context, id, entityTypeDataDriverConfig)
	if err != nil {
		return nil, err
	}

	instance, err := dbAPI.GetDataDriverInstance(context, param.DataDriverInstanceID)
	if err != nil {
		return nil, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataDriverConfig,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  instance.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return nil, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err = NewDataDriverConfigBinding(context, dbAPI).Set(tx, id, nil)
		if err != nil {
			return err
		}
		return dbAPI.deleteDataDriverParam(context, tx, id)
	})
	if err != nil {
		return nil, err
	}

	if callback != nil {
		go callback(context, param)
	}
	GetAuditlogHandler().addDataDriverConfigAuditLog(context, dbAPI, &param, instance.ProjectID, DELETE)

	return model.DeleteDocumentResponseV2{ID: id}, err
}

// DeleteDataDriverConfigW delete a data driver config object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDataDriverConfigW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDataDriverConfig, id, w, callback)
}

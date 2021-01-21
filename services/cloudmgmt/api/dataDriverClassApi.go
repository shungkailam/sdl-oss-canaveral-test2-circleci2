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

const entityTypeDataDriverClass = "dataDriverClass"

// DataDriverClassDBO is the DB model for Data Driver Class
type DataDriverClassDBO struct {
	model.BaseModelDBO

	Name                  string                    `json:"name" db:"name" validate:"range=1:200"`
	Description           string                    `json:"description,omitempty" db:"description" validate:"range=1:200"`
	DataDriverVersion     string                    `json:"driverVersion" db:"driver_version" validate:"range=1:20"`
	MinSvcDomainVersion   string                    `json:"minSvcDomainVersion,omitempty" db:"min_svc_domain_version" validate:"range=0:20"`
	Type                  model.DataDriverClassType `json:"type" validate:"range=1:100"`
	YamlData              string                    `json:"yamlData" db:"yaml_data" validate:"range=1:30720"`
	StaticParameterSchema *types.JSONText           `json:"staticParameterSchema,omitempty" db:"static_schema"`
	ConfigParameterSchema *types.JSONText           `json:"configParameterSchema,omitempty" db:"dynamic_schema"`
	StreamParameterSchema *types.JSONText           `json:"streamParameterSchema,omitempty" db:"stream_schema"`
}

func init() {
	queryMap["CreateDataDriverClass"] = `INSERT INTO data_driver_class_model 
               ( id,  version,  tenant_id,  name,  description,  driver_version,  min_svc_domain_version,  type,  yaml_data,  static_schema,  dynamic_schema,  stream_schema,  created_at,  updated_at)
        VALUES (:id, :version, :tenant_id, :name, :description, :driver_version, :min_svc_domain_version, :type, :yaml_data, :static_schema, :dynamic_schema, :stream_schema, :created_at, :updated_at)`
	queryMap["UpdateDataDriverClass"] = `UPDATE data_driver_class_model SET 
				version = :version, name = :name, description = :description, driver_version = :driver_version, 
				min_svc_domain_version = :min_svc_domain_version, yaml_data = :yaml_data, type = :type,
			    static_schema = :static_schema, dynamic_schema = :dynamic_schema, stream_schema = :stream_schema, 
				updated_at = :updated_at 
			    WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectDataDriverClassById"] = `SELECT * FROM data_driver_class_model WHERE tenant_id = :tenant_id AND id in (:ids)`
	queryMap["SelectDataDriverClasses"] = `SELECT *, count(*) OVER() as total_count FROM data_driver_class_model WHERE tenant_id = :tenant_id %s`

	orderByHelper.Setup(entityTypeDataDriverClass, []string{"id", "name", "created_at", "updated_at", "type"})
}

func ToDataDriverClassDBO(doc *model.DataDriverClass) (DataDriverClassDBO, error) {
	dbo := DataDriverClassDBO{}
	err := base.Convert(doc, &dbo)
	return dbo, err
}

func FromDataDriverClassDBO(dbo *DataDriverClassDBO) (model.DataDriverClass, error) {
	ddc := model.DataDriverClass{}
	err := base.Convert(dbo, &ddc)
	return ddc, err
}

func (dbAPI *dbObjectModelAPI) getDataDriverClasses(context context.Context, ids []string) ([]model.DataDriverClass, error) {
	dataDriverClasses := []model.DataDriverClass{}
	dataDriverClassDBOs := []DataDriverClassDBO{}

	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		return dataDriverClasses, nil
	}

	err = dbAPI.QueryIn(context, &dataDriverClassDBOs, queryMap["SelectDataDriverClassById"], tenantIdFilter{
		TenantID: authContext.TenantID,
		IDs:      ids,
	})
	if err != nil {
		return nil, err
	}

	for _, ddc := range dataDriverClassDBOs {
		d, err := FromDataDriverClassDBO(&ddc)
		if err != nil {
			return nil, err
		}
		dataDriverClasses = append(dataDriverClasses, d)
	}
	return dataDriverClasses, err
}

// SelectAllDataDriverClasses return paginated information about data driver classes
func (dbAPI *dbObjectModelAPI) SelectAllDataDriverClasses(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverClass, int, error) {
	dataDriverClasses := []model.DataDriverClass{}
	dataDriverClassDBOs := []DataDriverClassDBO{}
	err := dbAPI.getEntities(context, entityTypeDataDriverClass, queryMap["SelectDataDriverClasses"], entitiesQueryParam, &dataDriverClassDBOs)
	if err != nil {
		return nil, 0, err
	}

	first := true
	totalCount := 0
	for _, ddc := range dataDriverClassDBOs {
		if first {
			first = false
			if ddc.TotalCount != nil {
				totalCount = *ddc.TotalCount
			}
		}
		doc, err := FromDataDriverClassDBO(&ddc)
		if err != nil {
			return nil, 0, err
		}
		dataDriverClasses = append(dataDriverClasses, doc)
	}

	return dataDriverClasses, totalCount, err
}

// SelectAllDataDriverClassesW return paginated information about data driver classes, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataDriverClassesW(context context.Context, w io.Writer, r *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(r)
	ddcs, totalCount, err := dbAPI.SelectAllDataDriverClasses(context, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeDataDriverClass})
	resp := model.DataDriverClassListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		ListOfDataDrivers:         ddcs,
	}
	return base.DispatchPayload(w, resp)
}

// GetDataDriverClass get a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) GetDataDriverClass(context context.Context, dataDriverClassID string) (model.DataDriverClass, error) {
	dataDriverClasses, err := dbAPI.getDataDriverClasses(context, []string{dataDriverClassID})
	if err != nil {
		return model.DataDriverClass{}, err
	}
	if len(dataDriverClasses) == 0 {
		return model.DataDriverClass{}, errcode.NewRecordNotFoundError(dataDriverClassID)
	}
	return dataDriverClasses[0], nil
}

// GetDataDriverClassW gget a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataDriverClassW(context context.Context, dataDriverClassID string, w io.Writer, r *http.Request) error {
	project, err := dbAPI.GetDataDriverClass(context, dataDriverClassID)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, project)
}

// CreateDataDriverClass creates a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataDriverClass(context context.Context, i interface{} /* *model.DataDriverClass */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC")
	}

	p, ok := i.(*model.DataDriverClass)
	if !ok {
		return resp, errcode.NewInternalError("CreateDataDriverClass: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID

	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateDataDriverClass doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateDataDriverClass doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateDataDriverClass(&doc)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	dbo, err := ToDataDriverClassDBO(&doc)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["CreateDataDriverClass"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating data driver class for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addDataDriverClassAuditLog(context, dbAPI, &doc, CREATE)
	return resp, nil
}

// CreateDataDriverClassW creates a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataDriverClassW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDataDriverClass, &model.DataDriverClass{}, w, r, callback)
}

// UpdateDataDriverClass updates a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataDriverClass(context context.Context, i interface{} /* *model.DataDriverClass */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC")
	}

	p, ok := i.(*model.DataDriverClass)
	if !ok {
		return resp, errcode.NewInternalError("UpdateDataDriverClass: type error")
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

	found, err := dbAPI.GetDataDriverClass(context, doc.ID)
	if err != nil {
		return resp, err
	}

	err = model.ValidateDataDriverClass(&doc)
	if err != nil {
		return resp, err
	}

	inUse, err := dbAPI.hasDataDriverInstancesForClassId(context, doc.ID)
	if err != nil {
		return nil, err
	}
	if inUse {
		msg := fmt.Sprintf("data driver class %s is in use by data driver instances. cannot update this data driver class", id)
		return nil, errcode.NewPreConditionFailedError(msg)
	}

	dbo, err := ToDataDriverClassDBO(&doc)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateDataDriverClass"], &dbo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating data driver class for ID %s and tenant ID %s. Error: %s"), doc.ID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(found.ID, err)
	}
	GetAuditlogHandler().addDataDriverClassAuditLog(context, dbAPI, &doc, UPDATE)

	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateDataDriverClassW updates a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDataDriverClassW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDataDriverClass, &model.DataDriverClass{}, w, r, callback)
}

// DeleteDataDriverClass delete a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDataDriverClass(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return nil, errcode.NewPermissionDeniedError("RBAC")
	}

	ddc, err := dbAPI.GetDataDriverClass(context, id)
	if err != nil || len(ddc.ID) == 0 || ddc.ID != id {
		return nil, errcode.NewRecordNotFoundError(id)
	}

	inUse, err := dbAPI.hasDataDriverInstancesForClassId(context, ddc.ID)
	if err != nil {
		return nil, err
	}
	if inUse {
		msg := fmt.Sprintf("data driver class %s is in use by data driver instances. cannot remove this data driver class", id)
		return nil, errcode.NewPreConditionFailedError(msg)
	}

	entity, err := DeleteEntityV2(context, dbAPI, "data_driver_class_model", "id", id, ddc, callback)
	if err == nil {
		GetAuditlogHandler().addDataDriverClassAuditLog(context, dbAPI, &ddc, DELETE)
	}
	return entity, err
}

// DeleteDataDriverClassW delete a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDataDriverClassW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDataDriverClass, id, w, callback)
}

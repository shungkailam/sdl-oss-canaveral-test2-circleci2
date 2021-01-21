package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"encoding/json"
	"fmt"
	"github.com/jmoiron/sqlx/types"

	"context"
	"io"
	"net/http"
)

const entityTypeDataDriverStream = "dataDriverStream"

type dataDriverStreamLabelDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64  `json:"id" db:"id"`
	ParamsID           string `json:"paramsId" db:"params_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
}

func init() {
	queryMap["CreateDataDriverParamsLabel"] = `INSERT INTO data_driver_label_model (params_id, category_value_id) VALUES (:params_id, :category_value_id)`
	queryMap["SelectDataDriverParamsLabels"] = `SELECT data_driver_label_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
		FROM data_driver_label_model JOIN category_value_model ON data_driver_label_model.category_value_id = category_value_model.id 
		WHERE data_driver_label_model.params_id IN (:ids)`
	queryMap["DeleteDataDriverParamsLabels"] = `DELETE FROM data_driver_label_model WHERE params_id = :params_id`
}

func ToDataDriverParamsDBOFromStream(doc *model.DataDriverStream) (DataDriverParamsDBO, error) {
	dbo := DataDriverParamsDBO{}
	err := base.Convert(doc, &dbo)
	dbo.Type = entityTypeDataDriverStream

	data, err := json.Marshal(doc.Stream)
	if err != nil {
		return dbo, err
	}
	j := types.JSONText(data)
	dbo.Parameters = &j

	return dbo, err
}

func FromDataDriverParamsDBOToStream(dbo *DataDriverParamsDBO, binding *model.ServiceDomainBinding, labels []model.CategoryInfo) (model.DataDriverStream, error) {
	dds := model.DataDriverStream{}
	if dbo.Type != entityTypeDataDriverStream {
		return dds, errcode.NewBadRequestError("Type")
	}
	err := base.Convert(dbo, &dds)
	dds.ServiceDomainBinding = *binding
	dds.Labels = labels

	data := map[string]interface{}{}
	dbo.Parameters.Unmarshal(&data)
	dds.Stream = data

	return dds, err
}

func (dbAPI *dbObjectModelAPI) loadDataDriverStreamLabels(context context.Context, ids []string) (map[string][]model.CategoryInfo, error) {
	res := make(map[string][]model.CategoryInfo)
	if len(ids) == 0 {
		return res, nil
	}

	dbos := []dataDriverStreamLabelDBO{}
	err := dbAPI.QueryIn(context, &dbos, queryMap["SelectDataDriverParamsLabels"], &idFilter{
		IDs: ids,
	})
	if err != nil {
		return nil, err
	}

	for _, id := range ids {
		res[id] = []model.CategoryInfo{}
	}
	for _, dbo := range dbos {
		res[dbo.ParamsID] = append(res[dbo.ParamsID], dbo.CategoryInfo)
	}
	return res, nil
}

func (dbAPI *dbObjectModelAPI) pupulateDataDriverStreamLabels(context context.Context, tx *base.WrappedTx, id string, labels []model.CategoryInfo) error {
	_, err := tx.NamedExec(context, queryMap["DeleteDataDriverParamsLabels"], &dataDriverStreamLabelDBO{
		ParamsID: id,
	})
	if err != nil {
		return err
	}

	if labels == nil {
		return nil
	}

	for _, label := range labels {
		categoryValueDBOs, err := dbAPI.getCategoryValueDBOs(context, CategoryValueDBO{CategoryID: label.ID})
		if err != nil {
			return err
		}
		if len(categoryValueDBOs) == 0 {
			return errcode.NewRecordNotFoundError(label.ID)
		}
		valueFound := false
		for _, categoryValueDBO := range categoryValueDBOs {
			if categoryValueDBO.Value == label.Value {
				_, err := tx.NamedExec(context, queryMap["CreateDataDriverParamsLabel"], &dataDriverStreamLabelDBO{
					ParamsID:        id,
					CategoryValueID: categoryValueDBO.ID,
				})
				if err != nil {
					return errcode.TranslateDatabaseError(id, err)
				}
				valueFound = true
				break
			}
		}
		if !valueFound {
			return errcode.NewRecordNotFoundError(fmt.Sprintf("%s:%s", label.ID, label.Value))
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) hasDataDriverStreamsForInstanceId(context context.Context, instanceId string) (bool, error) {
	params, err := dbAPI.loadDatadriverParamsByInstanceId(context, instanceId, entityTypeDataDriverStream, nil)
	return len(params) > 0, err
}

// SelectDataDriverStreamsByInstance return all data driver instance stream, write output into writer
func (dbAPI *dbObjectModelAPI) SelectDataDriverStreamsByInstanceId(context context.Context, instanceId string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverStream, int, error) {
	params, err := dbAPI.loadDatadriverParamsByInstanceId(context, instanceId, entityTypeDataDriverStream, entitiesQueryParam)
	if err != nil {
		return nil, 0, err
	}

	paramIds := make([]string, len(params))
	for _, p := range params {
		paramIds = append(paramIds, p.ID)
	}

	bindings, err := NewDataDriverStreamBinding(context, dbAPI).GetAll(paramIds)
	if err != nil {
		return nil, 0, err
	}

	labels, err := dbAPI.loadDataDriverStreamLabels(context, paramIds)
	if err != nil {
		return nil, 0, err
	}

	res := make([]model.DataDriverStream, 0, len(params))
	for _, p := range params {
		binding := bindings[p.ID]
		label := labels[p.ID]
		stream, err := FromDataDriverParamsDBOToStream(&p, binding, label)
		if err != nil {
			return nil, 0, err
		}
		res = append(res, stream)
	}

	if len(res) == 0 {
		return res, 0, nil
	}

	return res, *params[0].TotalCount, nil
}

// SelectDataDriverStreamsByInstanceW return all data driver instance streams, write output into writer
func (dbAPI *dbObjectModelAPI) SelectDataDriverStreamsByInstanceIdW(context context.Context, id string, w io.Writer, r *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(r)
	streams, totalCount, err := dbAPI.SelectDataDriverStreamsByInstanceId(context, id, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeDataDriverStream})
	resp := model.DataDriverStreamListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		ListOfDataDriverStreams:   streams,
	}
	return base.DispatchPayload(w, &resp)
}

// GetDataDriverStream get a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) GetDataDriverStream(context context.Context, id string) (model.DataDriverStream, error) {
	param, err := dbAPI.loadDatadriverParamById(context, id, entityTypeDataDriverStream)
	if err != nil {
		return model.DataDriverStream{}, err
	}

	binding, err := NewDataDriverStreamBinding(context, dbAPI).Get(id)
	if err != nil {
		return model.DataDriverStream{}, err
	}

	labels, err := dbAPI.loadDataDriverStreamLabels(context, []string{id})
	if err != nil {
		return model.DataDriverStream{}, err
	}
	label := labels[id]
	return FromDataDriverParamsDBOToStream(&param, binding, label)
}

// GetDataDriverStreamW gget a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataDriverStreamW(context context.Context, id string, w io.Writer, r *http.Request) error {
	project, err := dbAPI.GetDataDriverStream(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, project)
}

// CreateDataDriverStream creates a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataDriverStream(context context.Context, i interface{} /* *model.DataDriverStream */, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	p, ok := i.(*model.DataDriverStream)
	if !ok {
		return nil, errcode.NewInternalError("CreateDataDriverStream: type error")
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
		meta.EntityDataDriverStream,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  instance.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return nil, err
	}

	err = model.ValidateDataDriverStream(p, &class.StreamParameterSchema, &project)
	if err != nil {
		return nil, err
	}

	err = dbAPI.cleanupServiceDomainBinding(context, &project, &p.ServiceDomainBinding)
	if err != nil {
		return nil, err
	}

	var doc *DataDriverParamsDBO
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		param, err := ToDataDriverParamsDBOFromStream(p)
		if err != nil {
			return err
		}
		doc, err = dbAPI.createDataDriverParam(context, tx, &param)
		if err != nil {
			return err
		}
		err = dbAPI.pupulateDataDriverStreamLabels(context, tx, doc.ID, p.Labels)
		if err != nil {
			return err
		}
		err = NewDataDriverStreamBinding(context, dbAPI).Set(tx, doc.ID, &p.ServiceDomainBinding)
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

	GetAuditlogHandler().addDataDriverStreamAuditLog(context, dbAPI, doc, instance.ProjectID, CREATE)
	return model.CreateDocumentResponseV2{ID: doc.ID}, nil
}

// CreateDataDriverStreamW creates a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataDriverStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDataDriverStream, &model.DataDriverStream{}, w, r, callback)
}

// UpdateDataDriverStream updates a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataDriverStream(context context.Context, i interface{} /* *model.DataDriverStream */, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	p, ok := i.(*model.DataDriverStream)
	if !ok {
		return nil, errcode.NewInternalError("UpdateDataDriverStream: type error")
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

	param, err := dbAPI.loadDatadriverParamById(context, p.ID, entityTypeDataDriverStream)
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
		meta.EntityDataDriverStream,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:  instance.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return nil, err
	}

	p.Direction = model.DataDriverStreamDirection(param.Direction) // We do not allow to update direction
	err = model.ValidateDataDriverStream(p, &class.StreamParameterSchema, &project)
	if err != nil {
		return nil, err
	}

	err = dbAPI.cleanupServiceDomainBinding(context, &project, &p.ServiceDomainBinding)
	if err != nil {
		return nil, err
	}

	var doc *DataDriverParamsDBO
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		param, err := ToDataDriverParamsDBOFromStream(p)
		if err != nil {
			return err
		}
		doc, err = dbAPI.updateDataDriverParam(context, tx, &param)
		if err != nil {
			return err
		}
		err = NewDataDriverStreamBinding(context, dbAPI).Set(tx, doc.ID, &p.ServiceDomainBinding)
		if err != nil {
			return err
		}
		err = dbAPI.pupulateDataDriverStreamLabels(context, tx, doc.ID, p.Labels)
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

	GetAuditlogHandler().addDataDriverStreamAuditLog(context, dbAPI, doc, instance.ProjectID, UPDATE)
	return model.UpdateDocumentResponseV2{ID: doc.ID}, nil
}

// UpdateDataDriverStreamW updates a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDataDriverStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDataDriverStream, &model.DataDriverStream{}, w, r, callback)
}

// DeleteDataDriverStream delete a data driver class object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDataDriverStream(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}

	param, err := dbAPI.loadDatadriverParamById(context, id, entityTypeDataDriverStream)
	if err != nil {
		return nil, err
	}

	instance, err := dbAPI.GetDataDriverInstance(context, param.DataDriverInstanceID)
	if err != nil {
		return nil, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataDriverStream,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  instance.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return nil, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err = NewDataDriverStreamBinding(context, dbAPI).Set(tx, id, nil)
		if err != nil {
			return err
		}
		err = dbAPI.pupulateDataDriverStreamLabels(context, tx, id, nil)
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
	GetAuditlogHandler().addDataDriverStreamAuditLog(context, dbAPI, &param, instance.ProjectID, DELETE)

	return model.DeleteDocumentResponseV2{ID: id}, err
}

// DeleteDataDriverStreamW delete a data driver class object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDataDriverStreamW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDataDriverStream, id, w, callback)
}

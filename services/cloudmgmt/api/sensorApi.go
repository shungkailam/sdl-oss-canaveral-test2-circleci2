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
)

func init() {
	queryMap["SelectSensors"] = `SELECT * FROM sensor_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (:edge_id = '' OR edge_id = :edge_id)`
	queryMap["CreateSensor"] = `INSERT INTO sensor_model (id, version, tenant_id, edge_id, topic_name, created_at, updated_at) VALUES (:id, :version, :tenant_id, :edge_id, :topic_name, :created_at, :updated_at)`
	queryMap["UpdateSensor"] = `UPDATE sensor_model SET version = :version, tenant_id = :tenant_id, edge_id = :edge_id, topic_name = :topic_name, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
}

// SensorDBO is DB object model for sensor
type SensorDBO struct {
	model.EdgeBaseModelDBO
	TopicName string `json:"topicName" db:"topic_name"`
}

func (dbAPI *dbObjectModelAPI) getSensors(context context.Context, edgeID string, sensorID string, startPageToken base.PageToken, pageSize int) ([]model.Sensor, error) {
	sensors := []model.Sensor{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return sensors, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: sensorID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel, EdgeID: edgeID}
	param := SensorDBO{EdgeBaseModelDBO: edgeModel}
	_, err = dbAPI.PagedQuery(context, startPageToken, pageSize, func(dbObjPtr interface{}) error {
		sensor := model.Sensor{}
		err := base.Convert(dbObjPtr, &sensor)
		if err != nil {
			return err
		}
		sensors = append(sensors, sensor)
		return nil
	}, queryMap["SelectSensors"], param)
	return sensors, err
}

func (dbAPI *dbObjectModelAPI) getSensorsW(context context.Context, edgeID string, sensorID string, w io.Writer, req *http.Request) error {
	sensorDBOs := []SensorDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: sensorID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel, EdgeID: edgeID}
	param := SensorDBO{EdgeBaseModelDBO: edgeModel}
	err = dbAPI.Query(context, &sensorDBOs, queryMap["SelectSensors"], param)
	if err != nil {
		return err
	}
	if len(sensorID) == 0 {
		return base.DispatchPayload(w, sensorDBOs)
	}
	if len(sensorDBOs) == 0 {
		return errcode.NewRecordNotFoundError(sensorID)
	}
	// return base.DispatchPayload(w, sensorDBOs)
	return json.NewEncoder(w).Encode(sensorDBOs[0])
}

// SelectAllSensors select all sensors for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllSensors(context context.Context) ([]model.Sensor, error) {
	return dbAPI.getSensors(context, "", "", base.StartPageToken, base.MaxRowsLimit)
}

// SelectAllSensorsW select all sensors for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllSensorsW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getSensorsW(context, "", "", w, req)
}

// SelectAllSensorsWV2 select all sensors for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllSensorsWV2(context context.Context, w io.Writer, req *http.Request) error {
	sensorDBOs := []SensorDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	param := SensorDBO{EdgeBaseModelDBO: edgeModel}
	err = dbAPI.Query(context, &sensorDBOs, queryMap["SelectSensors"], param)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, sensorDBOs); handled {
	// 	return err
	// }
	sensors := []model.Sensor{}
	for _, sensorDBO := range sensorDBOs {
		sensor := model.Sensor{}
		err = base.Convert(&sensorDBO, &sensor)
		if err != nil {
			return err
		}
		sensors = append(sensors, sensor)
	}
	queryParam := model.GetEntitiesQueryParam(req)
	queryInfo := ListQueryInfo{
		StartPage:  base.PageToken(""),
		TotalCount: len(sensors),
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.SensorListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		SensorList:                sensors,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllSensorsForEdge select all sensors for the given edge
func (dbAPI *dbObjectModelAPI) SelectAllSensorsForEdge(context context.Context, edgeID string) ([]model.Sensor, error) {
	return dbAPI.getSensors(context, edgeID, "", base.StartPageToken, base.MaxRowsLimit)
}

// SelectAllSensorsForEdgeW select all sensors for the given edge, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllSensorsForEdgeW(context context.Context, edgeID string, w io.Writer, req *http.Request) error {
	return dbAPI.getSensorsW(context, edgeID, "", w, req)
}

// SelectAllSensorsForEdgeWV2 select all sensors for the given edge, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllSensorsForEdgeWV2(context context.Context, edgeID string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	sensorDBOs := []SensorDBO{}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel, EdgeID: edgeID}
	param := SensorDBO{EdgeBaseModelDBO: edgeModel}
	err = dbAPI.Query(context, &sensorDBOs, queryMap["SelectSensors"], param)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, sensorDBOs); handled {
	// 	return err
	// }
	sensors := []model.Sensor{}
	for _, sensorDBO := range sensorDBOs {
		sensor := model.Sensor{}
		err = base.Convert(&sensorDBO, &sensor)
		if err != nil {
			return err
		}
		sensors = append(sensors, sensor)
	}
	queryParam := model.GetEntitiesQueryParam(req)
	queryInfo := ListQueryInfo{
		StartPage:  base.PageToken(""),
		TotalCount: len(sensors),
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.SensorListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		SensorList:                sensors,
	}
	// return base.DispatchPayload(w, sensorDBOs)
	return json.NewEncoder(w).Encode(r)
}

// GetSensor get a sensor object in the DB
func (dbAPI *dbObjectModelAPI) GetSensor(context context.Context, id string) (model.Sensor, error) {
	if len(id) == 0 {
		return model.Sensor{}, errcode.NewBadRequestError("sensorID")
	}
	sensors, err := dbAPI.getSensors(context, "", id, base.StartPageToken, base.MaxRowsLimit)
	if err != nil {
		return model.Sensor{}, err
	}
	if len(sensors) == 0 {
		return model.Sensor{}, errcode.NewRecordNotFoundError(id)
	}
	return sensors[0], nil
}

// GetSensorW get a sensor object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetSensorW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("sensorID")
	}
	// return base.DispatchPayload(w, sensorDBOs[0])
	return dbAPI.getSensorsW(context, "", id, w, req)
}

// CreateSensor creates a sensor object in the DB
func (dbAPI *dbObjectModelAPI) CreateSensor(context context.Context, i interface{} /* *model.Sensor */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Sensor)
	if !ok {
		return resp, errcode.NewInternalError("CreateSensor: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateSensor doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateSensor doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntitySensor,
		meta.OperationCreate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	sensorDBO := SensorDBO{}
	err = base.Convert(&doc, &sensorDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateSensor"], &sensorDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating sensor for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// CreateSensorW creates a sensor object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateSensorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateSensor, &model.Sensor{}, w, r, callback)
}

// CreateSensorWV2 creates a sensor object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateSensorWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateSensor), &model.Sensor{}, w, r, callback)
}

// UpdateSensor update a sensor object in the DB
func (dbAPI *dbObjectModelAPI) UpdateSensor(context context.Context, i interface{} /* *model.Sensor */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Sensor)
	if !ok {
		return resp, errcode.NewInternalError("UpdateSensor: type error")
	}
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	if p.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	err = auth.CheckRBAC(
		authContext,
		meta.EntitySensor,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	sensorDBO := SensorDBO{}
	err = base.Convert(&doc, &sensorDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateSensor"], &sensorDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating sensor for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateSensorW update a sensor object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateSensorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateSensor, &model.Sensor{}, w, r, callback)
}

// UpdateSensorWV2 update a sensor object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateSensorWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateSensor), &model.Sensor{}, w, r, callback)
}

// DeleteSensor delete a sensor object in the DB
func (dbAPI *dbObjectModelAPI) DeleteSensor(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntitySensor,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	doc, err := dbAPI.GetSensor(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	return DeleteEntity(context, dbAPI, "sensor_model", "id", id, doc, callback)
}

// DeleteSensorW delete a sensor object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteSensorW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteSensor, id, w, callback)
}

// DeleteSensorWV2 delete a sensor object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteSensorWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteSensor), id, w, callback)
}

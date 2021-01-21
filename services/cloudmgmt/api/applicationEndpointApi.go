package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
)

func init() {
	queryMap["SelectDataSourceFieldsByTopic"] = `SELECT id, data_source_id, name, mqtt_topic, field_type FROM data_source_field_model WHERE data_source_id = :data_source_id AND mqtt_topic = :mqtt_topic AND name = :name`
	queryMap["CreateApplicationEndpoint"] = `INSERT INTO application_endpoint_model (application_id, tenant_id, created_at, field_name, data_source_id) VALUES (:application_id, :tenant_id, :created_at, :field_name, :data_source_id)`
	queryMap["SelectAppsDataIfcEndpoints"] = `SELECT df.data_source_id as id, df.mqtt_topic as value, ae.application_id as application_id, df.name as field_name  FROM application_endpoint_model as ae JOIN data_source_field_model as df ON df.data_source_id = ae.data_source_id AND df.name=ae.field_name WHERE ae.application_id IN (:application_ids)`
	queryMap["SelectAppEndpointsByDataSources"] = `SELECT df.data_source_id as id, df.mqtt_topic as value, ae.application_id as application_id, df.name as field_name FROM application_endpoint_model as ae JOIN data_source_field_model as df ON df.data_source_id = ae.data_source_id AND ae.field_name=df.name WHERE ae.data_source_id IN (:data_source_ids)`
}

// ApplicationEndpointDBO is DB object model for application endpoints
type ApplicationEndpointDBO struct {
	ID            int       `json:"id" db:"id"`
	ApplicationID string    `json:"applicationId" db:"application_id"`
	FieldName     string    `json:"fieldName" db:"field_name"`
	DataSourceID  string    `json:"dataSourceId" db:"data_source_id"`
	TenantID      string    `json:"tenantId" db:"tenant_id"`
	CreatedAt     time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time `json:"updatedAt" db:"updated_at"`
}

// SelectApplicationEndpointDBO defines the DB object model for select application endpoint
type SelectApplicationEndpointDBO struct {
	// ApplicationID is the ID of the application
	ApplicationID string `json:"applicationId" db:"application_id"`

	// This is the MQTT topic as per current implementation
	Value string `json:"value" db:"value"`

	// Field name of the topic
	FieldName string `json:"fieldName" db:"field_name"`

	// ID is the ID of the data source for the endpoint
	ID string `json:"id" db:"id"`
}

// CreateApplicationEndpoint creates application endpoint within the given txn
func (dbAPI *dbObjectModelAPI) CreateApplicationEndpoint(context context.Context, tx *base.WrappedTx, appID string, tenantID string, endpoint model.DataIfcEndpoint) error {
	var loggedErr error
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()

	// Get the data source field first
	param := DataSourceFieldDBO{DataSourceID: endpoint.ID}
	param.MQTTTopic = endpoint.Value
	param.Name = endpoint.Name
	dsField := []DataSourceFieldDBO{}
	// Query in txn to make sure the query fetches source fields that may have been inserted in the same txn
	err := base.QueryTxn(context, tx, &dsField, queryMap["SelectDataSourceFieldsByTopic"], param)
	if err != nil {
		loggedErr = fmt.Errorf("failed to fetch data source fields for data source %s and topic %s", param.DataSourceID, param.MQTTTopic)
		return errcode.NewInternalError(loggedErr.Error())
	}

	if len(dsField) != 1 {
		loggedErr = fmt.Errorf("expected to find exactly 1 data source field but found %d for endpoint %+v", len(dsField), endpoint)
		return errcode.NewPreConditionFailedError(loggedErr.Error())
	}

	endpointDBO := ApplicationEndpointDBO{
		ApplicationID: appID,
		TenantID:      tenantID,
		FieldName:     dsField[0].Name,
		CreatedAt:     base.RoundedNow(),
		DataSourceID:  endpoint.ID,
	}

	_, err = tx.NamedExec(context, queryMap["CreateApplicationEndpoint"], &endpointDBO)
	if err != nil {
		loggedErr = fmt.Errorf("error creating application endpoint. %s", err.Error())
		dbErr := errcode.TranslateDatabaseError(fmt.Sprintf("%+v", endpointDBO), err)
		// Handle idempotence
		if !errcode.IsDuplicateRecordError(dbErr) {
			return loggedErr
		}
	}

	return nil
}

func (dbAPI *dbObjectModelAPI) FetchApplicationIDsByDataIfcID(context context.Context, dsID string) ([]string, error) {
	var loggedErr error
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()
	selectAppEndpoints := []SelectApplicationEndpointDBO{}

	err := dbAPI.QueryIn(context, &selectAppEndpoints, queryMap["SelectAppEndpointsByDataSources"], DataSourceIdsParam{DataSourceIDs: []string{dsID}})
	if err != nil {
		loggedErr = errcode.NewInternalError(fmt.Sprintf("failed to fetch application endpoints. %s", err.Error()))
		return nil, loggedErr
	}
	appIDs := []string{}
	for _, s := range selectAppEndpoints {
		appIDs = append(appIDs, s.ApplicationID)
	}
	glog.V(5).Infof("app IDs: %+v for data source %s", appIDs, dsID)
	return appIDs, nil
}

// FetchApplicationEndpointsByDataSource fetches app endpoints by data source ID
func (dbAPI *dbObjectModelAPI) FetchApplicationEndpointsByDataSource(context context.Context, dsID string) ([]model.DataIfcEndpoint, error) {
	var loggedErr error
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()

	selectAppEndpoints := []SelectApplicationEndpointDBO{}

	err := dbAPI.QueryIn(context, &selectAppEndpoints, queryMap["SelectAppEndpointsByDataSources"], DataSourceIdsParam{DataSourceIDs: []string{dsID}})
	if err != nil {
		loggedErr = errcode.NewInternalError(fmt.Sprintf("failed to fetch application endpoints. %s", err.Error()))
		return nil, loggedErr
	}
	endpoints := make([]model.DataIfcEndpoint, 0, len(selectAppEndpoints))
	for _, s := range selectAppEndpoints {
		endpoints = append(endpoints, model.DataIfcEndpoint{Name: s.FieldName, ID: s.ID, Value: s.Value})
	}

	glog.V(5).Infof(base.PrefixRequestID(context, "endpoints fetched for %s are %+v"), dsID, endpoints)
	return endpoints, nil
}

// FetchApplicationEndpoints fetches application endpoints from the DB for the given apps and returns them as a map keyed by app ID
func (dbAPI *dbObjectModelAPI) FetchApplicationsEndpoints(context context.Context, appIDs []string) (map[string][]model.DataIfcEndpoint, error) {
	var loggedErr error
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()

	selectAppEndpoints := []SelectApplicationEndpointDBO{}

	err := dbAPI.QueryIn(context, &selectAppEndpoints, queryMap["SelectAppsDataIfcEndpoints"], ApplicationIdsParam{ApplicationIDs: appIDs})
	if err != nil {
		loggedErr = errcode.NewInternalError(fmt.Sprintf("failed to fetch application endpoints. %s", err.Error()))
		return nil, loggedErr
	}

	endpointsByAppID := make(map[string][]model.DataIfcEndpoint)
	for _, s := range selectAppEndpoints {
		endpointsByAppID[s.ApplicationID] = append(endpointsByAppID[s.ApplicationID],
			model.DataIfcEndpoint{Name: s.FieldName, ID: s.ID, Value: s.Value},
		)
	}

	return endpointsByAppID, nil
}

// DeleteAllApplicationEndpoints deletes all application endpoint records for the given appID
func (dbAPI *dbObjectModelAPI) DeleteAllApplicationEndpoints(context context.Context, tx *base.WrappedTx, appID string) error {
	deleteParams := map[string]interface{}{
		"application_id": appID,
	}

	res, err := base.DeleteTxn(context, tx, "application_endpoint_model", deleteParams)
	n, err := res.RowsAffected()
	if err == nil {
		glog.V(5).Infof(base.PrefixRequestID(context, "%d rows affectd by deleting endpoints of app %s"), n, appID)
	}
	return err
}

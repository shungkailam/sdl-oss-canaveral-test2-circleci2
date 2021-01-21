package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"cloudservices/common/utils"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
)

const entityTypeDataSource = "datasource"

// DataSourceDBO is DB object model for data source
type DataSourceDBO struct {
	model.EdgeBaseModelDBO
	Name         string  `json:"name" db:"name"`
	Type         string  `json:"type" db:"type"`
	SensorModel  string  `json:"sensorModel" db:"sensor_model"`
	Connection   string  `json:"connection" db:"connection"`
	Protocol     string  `json:"protocol" db:"protocol"`
	AuthType     string  `json:"authType" db:"auth_type"`
	IfcClass     *string `json:"ifcClass" db:"ifc_class"`
	IfcKind      *string `json:"ifcKind" db:"ifc_kind"`
	IfcProtocol  *string `json:"ifcProtocol" db:"ifc_protocol"`
	IfcImg       *string `json:"ifcImg" db:"ifc_img"`
	IfcProjectID *string `json:"ifcProjectId" db:"ifc_project_id"`
	IfcDriverID  *string `json:"ifcDriverId" db:"ifc_driver_id"`
}
type DataSourceDBO2 struct {
	DataSourceDBO
	EdgeIDs []string `json:"edgeIds" db:"edge_ids"`
}

type DataSourceIfcPortsDBO struct {
	ID           int64  `json:"id" db:"id"`
	DataSourceID string `json:"dataSourceId" db:"data_source_id"`
	Name         string `json:"name" db:"name"`
	Port         int    `json:"port" db:"port"`
}

// DataSourceFieldDBO is the DB model for source field
type DataSourceFieldDBO struct {
	model.DataSourceFieldInfo
	ID           int64  `json:"id" db:"id"`
	DataSourceID string `json:"dataSourceId" db:"data_source_id"`
}

// DataSourceFieldSelectorDBO is the DB model for field selector
type DataSourceFieldSelectorDBO struct {
	ID                 int64  `json:"id" db:"id"`
	DataSourceID       string `json:"dataSourceId" db:"data_source_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
	FieldID            *int64 `json:"FieldId" db:"field_id"`
	model.CategoryInfo `json:"category_info" db:"category_info"`
}

// DataSourceArtifactDBO is the DB model for data source artifact
type DataSourceArtifactDBO struct {
	model.ArtifactBaseModelDBO
	DataSourceID string `json:"dataSourceId" db:"data_source_id"`
}

type DataSourceIdsParam struct {
	DataSourceIDs []string `json:"dataSourceIds" db:"data_source_ids"`
}

func init() {
	queryMap["SelectDataSourcesTemplate1"] = `SELECT * FROM data_source_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) %s`
	queryMap["SelectDataSourcesTemplate"] = `SELECT *, count(*) OVER() as total_count FROM data_source_model WHERE tenant_id = :tenant_id %s`
	queryMap["SelectDataSourcesByIds"] = `SELECT * from data_source_model where id in ('%s')`
	queryMap["SelectDataSourcesByEdgesTemplate1"] = `SELECT * FROM data_source_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND edge_id IN (:edge_ids) %s`
	queryMap["SelectDataSourcesByEdgesTemplate"] = `SELECT *, count(*) OVER() as total_count FROM data_source_model WHERE tenant_id = :tenant_id AND edge_id IN (:edge_ids) %s`
	queryMap["SelectDataSourcesFields"] = `SELECT * FROM data_source_field_model WHERE data_source_id IN (:data_source_ids)`
	queryMap["SelectDataSourceFields"] = `SELECT * FROM data_source_field_model WHERE data_source_id = :id`

	queryMap["SelectDataSourceIfcPorts"] = `SELECT * FROM data_source_ifc_port_model WHERE data_source_id = :data_source_id`
	queryMap["SelectDataSourcesIfcPorts"] = `SELECT * FROM data_source_ifc_port_model WHERE data_source_id IN (:data_source_ids)`

	queryMap["SelectDataSourceFieldSelectors"] = `SELECT data_source_field_selector_model.*,
			category_value_model.category_id "category_info.id", category_value_model.value "category_info.value" 
			FROM data_source_field_selector_model JOIN category_value_model
			ON data_source_field_selector_model.category_value_id = category_value_model.id
			WHERE data_source_field_selector_model.data_source_id = :data_source_id`
	queryMap["SelectDataSourcesFieldSelectors"] = `SELECT data_source_field_selector_model.*,
			category_value_model.category_id "category_info.id", category_value_model.value "category_info.value" 
			FROM data_source_field_selector_model JOIN category_value_model
			ON data_source_field_selector_model.category_value_id = category_value_model.id
			WHERE data_source_field_selector_model.data_source_id IN (:data_source_ids)`

	queryMap["CreateDataSource"] = `INSERT INTO data_source_model (id, version, tenant_id, edge_id, name, 
			type, sensor_model, connection, protocol, auth_type, created_at, 
			updated_at, ifc_class, ifc_kind, ifc_protocol, ifc_img, 
			ifc_project_id, ifc_driver_id) VALUES (:id, :version, :tenant_id, :edge_id, :name, 
			:type, :sensor_model, :connection, :protocol, :auth_type, :created_at, 
			:updated_at, :ifc_class, :ifc_kind, :ifc_protocol, :ifc_img, :ifc_project_id, 
			:ifc_driver_id)`

	queryMap["CreateDataSourceIfcPort"] = `INSERT INTO data_source_ifc_port_model (data_source_id, name, port) VALUES (:data_source_id, :name, :port) RETURNING id`

	queryMap["CreateDataSourceField"] = `INSERT INTO data_source_field_model (name, data_source_id, mqtt_topic, field_type) VALUES (:name, :data_source_id, :mqtt_topic, :field_type) RETURNING id`

	queryMap["CreateDataSourceFieldSelector"] = `INSERT INTO data_source_field_selector_model (data_source_id, field_id, category_value_id) VALUES (:data_source_id, :field_id, :category_value_id)`

	queryMap["UpdateDataSource"] = `UPDATE data_source_model SET version = :version, tenant_id = :tenant_id, edge_id = :edge_id, name = :name, type = :type, sensor_model = :sensor_model, connection = :connection, protocol = :protocol, 
			auth_type = :auth_type, updated_at = :updated_at, ifc_class = :ifc_class,
			ifc_kind = :ifc_kind, ifc_protocol = :ifc_protocol, ifc_img = :ifc_img, 
			ifc_project_id = :ifc_project_id, ifc_driver_id = :ifc_driver_id WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["GetEdgeID"] = `SELECT edge_id FROM data_source_model WHERE tenant_id = :tenant_id AND id = :id`

	queryMap["SelectDataSourceArtifact"] = `SELECT * FROM data_source_artifact_model WHERE tenant_id = :tenant_id AND data_source_id = :data_source_id`
	queryMap["CreateDataSourceArtifact"] = `INSERT INTO data_source_artifact_model (tenant_id, data_source_id, data, version) VALUES (:tenant_id, :data_source_id, :data, :version)`

	orderByHelper.Setup(entityTypeDataSource, []string{"id", "version", "created_at", "updated_at", "name", "type", "connection", "protocol", "auth_type"})
}

func (dbAPI *dbObjectModelAPI) GetDataSourceEdgeID(context context.Context, dataSourceID string) (string, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return "", err
	}
	type EdgeIDQuery struct {
		TenantID string `json:"tenantId" db:"tenant_id"`
		ID       string `json:"id" db:"id"`
	}
	type EdgeIDDBO struct {
		EdgeID string `json:"edgeId" db:"edge_id"`
	}
	param := EdgeIDQuery{
		TenantID: authContext.TenantID,
		ID:       dataSourceID,
	}
	edgeIDDBOs := []EdgeIDDBO{}
	err = dbAPI.Query(context, &edgeIDDBOs, queryMap["GetEdgeID"], param)
	if err != nil {
		return "", err
	}
	if len(edgeIDDBOs) == 0 {
		return "", errcode.NewRecordNotFoundError(dataSourceID)
	}
	return edgeIDDBOs[0].EdgeID, nil
}

/*** Start of common shared private methods for data sources ***/
func (dbAPI *dbObjectModelAPI) getDataSources(context context.Context, edgeID string, dataSourceID string, projectID string, startPage base.PageToken, pageSize int, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error) {
	// only one of edgeID, dataSourceID, projectID can be non empty
	dataSources := []model.DataSource{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dataSources, err
	}
	tenantID := authContext.TenantID
	dataSourceDBOs := []DataSourceDBO{}
	baseModel := model.BaseModelDBO{TenantID: tenantID, ID: dataSourceID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: baseModel, EdgeID: edgeID}
	param := DataSourceDBO{EdgeBaseModelDBO: edgeModel}
	var edgeIDs []string
	var query string
	// the only case we don't need edgeIDs: dataSourceID != "" and IsInfraAdminRole
	if dataSourceID != "" && auth.IsInfraAdminRole(authContext) {
		query, err = buildQuery(entityTypeDataSource, queryMap["SelectDataSourcesTemplate1"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return dataSources, err
		}
		err = dbAPI.Query(context, &dataSourceDBOs, query, tenantIDParam5{TenantID: tenantID, ID: dataSourceID})
		if err != nil {
			return dataSources, err
		}
		dataSources, _, err = dbAPI.dataSourceDBOsToDS(context, dataSourceDBOs)
	} else {
		if edgeID == "" {
			if projectID == "" {
				edges, err := dbAPI.getAllClusterTypes(context, false)
				if err != nil {
					return dataSources, err
				}
				edgeIDs = (funk.Map(edges, func(e interface{}) string { return e.(model.EdgeCluster).ID })).([]string)
			} else {
				// GetProject won't throw RBAC error for infra admin,
				// so add RBAC check here
				if !auth.IsProjectMember(projectID, authContext) {
					return dataSources, errcode.NewPermissionDeniedError("RBAC")
				}
				project, err := dbAPI.GetProject(context, projectID)
				if err != nil {
					return dataSources, err
				}
				edgeIDs = project.EdgeIDs
			}
		} else {
			edgeIDs = []string{edgeID}
		}
		if len(edgeIDs) == 0 {
			return dataSources, nil
		}
		param2 := DataSourceDBO2{DataSourceDBO: param, EdgeIDs: edgeIDs}
		query, err = buildQuery(entityTypeDataSource, queryMap["SelectDataSourcesByEdgesTemplate1"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return dataSources, err
		}
		_, err = dbAPI.NotPagedQueryIn(context, startPage, pageSize, func(dbObjPtr interface{}) error {
			dataSource := model.DataSource{}
			err := base.Convert(dbObjPtr, &dataSource)
			if err != nil {
				return err
			}
			dataSourceDBO := DataSourceDBO{}
			err = base.Convert(dbObjPtr, &dataSourceDBO)
			if err != nil {
				return err
			}
			if dataSourceDBO.IfcClass != nil {
				ifcInfo := model.DataSourceIfcInfo{}
				err = base.Convert(&dataSourceDBO, &ifcInfo)
				if err != nil {
					return err
				}
				dataSource.IfcInfo = &ifcInfo
			}
			err = dbAPI.populateDataSourceIfcInfo(context, &dataSourceDBO, &dataSource)
			if err != nil {
				return err
			}
			dataSources = append(dataSources, dataSource)
			return nil
		}, query, param2)
		if err == nil {
			err = dbAPI.populateDataSourcesFields(context, dataSources)
		}
	}
	return dataSources, err
}

func (dbAPI *dbObjectModelAPI) populateDataSourceIfcPorts(ctx context.Context, dataSource *model.DataSource) error {
	glog.Infof("Populate ifc ports for ID %s", dataSource.ID)
	if (dataSource.IfcInfo) == nil {
		return nil
	}
	dataSourceIdsParam := DataSourceIdsParam{
		DataSourceIDs: []string{dataSource.ID},
	}
	dataSourceIfcPortDBOs := []DataSourceIfcPortsDBO{}
	err := dbAPI.QueryIn(ctx, &dataSourceIfcPortDBOs, queryMap["SelectDataSourcesIfcPorts"], dataSourceIdsParam)
	if err != nil {
		return err
	}

	ports := []model.DataSourceIfcPorts{}
	for _, ifcPort := range dataSourceIfcPortDBOs {
		dataSourceIfcPortInfo := model.DataSourceIfcPorts{}
		err = base.Convert(&ifcPort, &dataSourceIfcPortInfo)
		if err != nil {
			return err
		}
		ports = append(ports, dataSourceIfcPortInfo)
	}

	dataSource.IfcInfo.Ports = ports
	return nil
}

func (dbAPI *dbObjectModelAPI) populateDataSourceIfcInfo(ctx context.Context, dsDBO *DataSourceDBO,
	dataSource *model.DataSource) error {
	// Check if this datasource was derived from an inteface class
	if dsDBO.IfcClass == nil {
		return nil
	}
	ifcInfo := model.DataSourceIfcInfo{}
	err := base.Convert(dsDBO, &ifcInfo)
	if err != nil {
		return err
	}
	dataSource.IfcInfo = &ifcInfo
	return dbAPI.populateDataSourceIfcPorts(ctx, dataSource)
}

func (dbAPI *dbObjectModelAPI) populateDataSourcesFields(ctx context.Context, dataSources []model.DataSource) error {
	if len(dataSources) == 0 {
		return nil
	}
	dataSourceIDs := funk.Map(dataSources, func(dataSource model.DataSource) string { return dataSource.ID }).([]string)
	dataSourceIdsParam := DataSourceIdsParam{
		DataSourceIDs: dataSourceIDs,
	}
	dataSourceFieldDBOs := []DataSourceFieldDBO{}
	err := dbAPI.QueryIn(ctx, &dataSourceFieldDBOs, queryMap["SelectDataSourcesFields"], dataSourceIdsParam)
	if err != nil {
		return err
	}
	// Field ID to field info
	dataSourceFieldInfoMap := map[int64]model.DataSourceFieldInfo{}
	m := map[string][]model.DataSourceFieldInfo{}
	for _, dataSourceFieldDBO := range dataSourceFieldDBOs {
		dataSourceFieldInfo := model.DataSourceFieldInfo{}
		err = base.Convert(&dataSourceFieldDBO, &dataSourceFieldInfo)
		if err != nil {
			return err
		}
		dataSourceFieldInfoMap[dataSourceFieldDBO.ID] = dataSourceFieldInfo
		m[dataSourceFieldDBO.DataSourceID] = append(m[dataSourceFieldDBO.DataSourceID], dataSourceFieldInfo)
	}
	dataSourcesFieldSelectorDBOMap := map[string]map[int64][]DataSourceFieldSelectorDBO{}
	for i := 0; i < len(dataSources); i++ {
		dataSources[i].Fields = m[dataSources[i].ID]
		dataSourcesFieldSelectorDBOMap[dataSources[i].ID] = map[int64][]DataSourceFieldSelectorDBO{}
	}
	dataSourceFieldSelectorDBOs := []DataSourceFieldSelectorDBO{}
	err = dbAPI.QueryIn(ctx, &dataSourceFieldSelectorDBOs, queryMap["SelectDataSourcesFieldSelectors"], dataSourceIdsParam)
	if err != nil {
		return err
	}
	// Category value ID to field selector

	// Group by category value ID
	for _, dataSourceFieldSelectorDBO := range dataSourceFieldSelectorDBOs {
		m := dataSourcesFieldSelectorDBOMap[dataSourceFieldSelectorDBO.DataSourceID]
		m[dataSourceFieldSelectorDBO.CategoryValueID] = append(m[dataSourceFieldSelectorDBO.CategoryValueID], dataSourceFieldSelectorDBO)
	}
	for i := 0; i < len(dataSources); i++ {
		dataSource := &dataSources[i]
		for _, dataSourceFieldSelectorDBOs := range dataSourcesFieldSelectorDBOMap[dataSource.ID] {
			dataSourcefieldSelector := model.DataSourceFieldSelector{}
			for _, sourceFieldSelectorDBO := range dataSourceFieldSelectorDBOs {
				if sourceFieldSelectorDBO.FieldID != nil {
					dataSourceFieldInfo := dataSourceFieldInfoMap[*sourceFieldSelectorDBO.FieldID]
					dataSourcefieldSelector.Scope = append(dataSourcefieldSelector.Scope, dataSourceFieldInfo.Name)
				} else {
					dataSourcefieldSelector.Scope = []string{"__ALL__"}
				}
				dataSourcefieldSelector.CategoryInfo = sourceFieldSelectorDBO.CategoryInfo
			}
			dataSource.Selectors = append(dataSource.Selectors, dataSourcefieldSelector)
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) createDataSourceIfcPorts(ctx context.Context, tx *base.WrappedTx, dataSource model.DataSource) error {
	if dataSource.IfcInfo == nil || len(dataSource.IfcInfo.Ports) == 0 {
		return nil
	}
	for _, port := range dataSource.IfcInfo.Ports {
		ifcPort := DataSourceIfcPortsDBO{DataSourceID: dataSource.ID, Name: port.Name, Port: port.Port}
		// NamedExec does not return the ID in the last inserted ID
		_, err := tx.NamedExec(ctx, queryMap["CreateDataSourceIfcPort"], &ifcPort)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error creating data source ifc port with ID %s. Error: %s"), dataSource.ID, err.Error())
			return errcode.TranslateDatabaseError(dataSource.ID, err)
		}
	}
	return nil
}

// createDataSourceField creates a data source field
// This method should be used when there are no selectors associated with the data source field. An example is data ifc
func (dbAPI *dbObjectModelAPI) createDataSourceField(ctx context.Context, tx *base.WrappedTx, fieldInfo model.DataSourceFieldInfo, dsID string) error {
	glog.V(5).Infof("inserting data source field info %+v for data source %s", fieldInfo, dsID)
	sourceFieldDBO := DataSourceFieldDBO{DataSourceFieldInfo: fieldInfo, DataSourceID: dsID}
	// NamedExec does not return the ID in the last inserted ID
	rows, err := tx.NamedQuery(ctx, queryMap["CreateDataSourceField"], &sourceFieldDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error creating data source field %s. Error: %s"), fieldInfo.Name, err.Error())
		return errcode.TranslateDatabaseError(fieldInfo.Name, err)
	}
	defer rows.Close()
	return nil
}

func (dbAPI *dbObjectModelAPI) createDataSourceFields(ctx context.Context, tx *base.WrappedTx, dataSource model.DataSource) error {
	fieldNameIds := map[string]int64{}
	for _, dataSourceFieldInfo := range dataSource.Fields {
		sourceFieldDBO := DataSourceFieldDBO{DataSourceFieldInfo: dataSourceFieldInfo, DataSourceID: dataSource.ID}
		// NamedExec does not return the ID in the last inserted ID
		rows, err := tx.NamedQuery(ctx, queryMap["CreateDataSourceField"], &sourceFieldDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error creating data source field for ID %s. Error: %s"), dataSource.ID, err.Error())
			return errcode.TranslateDatabaseError(dataSource.ID, err)
		}
		defer rows.Close()
		var id int64
		// Get the inserted primary key
		for rows.Next() {
			err = rows.Scan(&id)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error creating data source field for ID %s. Error: %s"), dataSource.ID, err.Error())
				return errcode.TranslateDatabaseError(dataSource.ID, err)
			}
		}
		fieldNameIds[dataSourceFieldInfo.Name] = id
	}
	for _, dataSourceFieldSelector := range dataSource.Selectors {
		param := CategoryValueDBO{CategoryID: dataSourceFieldSelector.CategoryInfo.ID, Value: dataSourceFieldSelector.CategoryInfo.Value}
		categoryValueDBOs, err := dbAPI.getCategoryValueDBOs(ctx, param)
		if err != nil {
			return err

		}
		if len(categoryValueDBOs) == 0 {
			return errcode.NewRecordNotFoundError(param.CategoryID)
		}
		dataSourceFieldSelectorDBO := DataSourceFieldSelectorDBO{DataSourceID: dataSource.ID, CategoryValueID: categoryValueDBOs[0].ID}
		for _, field := range dataSourceFieldSelector.Scope {
			var fieldID *int64
			if field != "__ALL__" {
				id, ok := fieldNameIds[field]
				if !ok {
					return errcode.NewRecordNotFoundError(field)
				}
				fieldID = &id
			}
			dataSourceFieldSelectorDBO.FieldID = fieldID
			_, err = tx.NamedExec(ctx, queryMap["CreateDataSourceFieldSelector"], &dataSourceFieldSelectorDBO)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error occurred while creating datasource field selector for ID %s. Error: %s"), dataSourceFieldSelectorDBO.Value, err.Error())
				return errcode.TranslateDatabaseError(dataSource.ID, err)
			}
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getDataSourcesW(context context.Context, edgeID string, dataSourceID string, projectID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	dataSources, err := dbAPI.getDataSources(context, edgeID, dataSourceID, projectID, base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
	if err != nil {
		return err
	}
	if len(dataSourceID) == 0 {
		return base.DispatchPayload(w, dataSources)
	}
	if len(dataSources) == 0 {
		return errcode.NewRecordNotFoundError(dataSourceID)
	}
	return json.NewEncoder(w).Encode(dataSources[0])
}

// internal API used by getDataSourcesWV2
func (dbAPI *dbObjectModelAPI) getDataSourcesByEdgesForQuery(context context.Context, edgeIDs []string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataSource, int, error) {
	dataSources := []model.DataSource{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dataSources, 0, err
	}
	tenantID := authContext.TenantID
	dataSourceDBOs := []DataSourceDBO{}

	var query string
	if len(edgeIDs) == 0 {
		if !auth.IsInfraAdminRole(authContext) {
			return dataSources, 0, nil
		}
		query, err = buildLimitQuery(entityTypeDataSource, queryMap["SelectDataSourcesTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return dataSources, 0, err
		}
		err = dbAPI.Query(context, &dataSourceDBOs, query, tenantIDParam3{TenantID: tenantID})
	} else {
		query, err = buildLimitQuery(entityTypeDataSource, queryMap["SelectDataSourcesByEdgesTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return dataSources, 0, err
		}
		err = dbAPI.QueryIn(context, &dataSourceDBOs, query, tenantIDParam3{TenantID: tenantID, EdgeIDs: edgeIDs})
	}
	if err != nil {
		return dataSources, 0, err
	}
	if len(dataSourceDBOs) == 0 {
		return dataSources, 0, nil
	}
	return dbAPI.dataSourceDBOsToDS(context, dataSourceDBOs)
}

func (dbAPI *dbObjectModelAPI) dataSourceDBOsToDS(context context.Context, dataSourceDBOs []DataSourceDBO) ([]model.DataSource, int, error) {
	dataSources := []model.DataSource{}
	totalCount := 0
	first := true
	for _, dataSourceDBO := range dataSourceDBOs {
		dataSource := model.DataSource{}
		if first {
			first = false
			if dataSourceDBO.TotalCount != nil {
				totalCount = *dataSourceDBO.TotalCount
			}
		}
		err := base.Convert(&dataSourceDBO, &dataSource)
		if err != nil {
			return []model.DataSource{}, 0, err
		}

		err = dbAPI.populateDataSourceIfcInfo(context, &dataSourceDBO, &dataSource)
		if err != nil {
			return []model.DataSource{}, 0, err
		}
		dataSources = append(dataSources, dataSource)
	}
	err := dbAPI.populateDataSourcesFields(context, dataSources)
	return dataSources, totalCount, err
}

func (dbAPI *dbObjectModelAPI) getDataSourcesWV2(context context.Context, edgeID string, dataSourceID string, projectID string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	edgeIDs := []string{}
	if edgeID != "" {
		edgeIDs = []string{edgeID}
	} else if projectID != "" {
		project, err := dbAPI.GetProject(context, projectID)
		if err != nil {
			return err
		}
		edgeIDs = project.EdgeIDs
	} else {
		if !auth.IsInfraAdminRole(authContext) {
			projectIDs := auth.GetProjectIDs(authContext)
			projects, err := dbAPI.getProjectsByIDs(context, tenantID, projectIDs)
			if err != nil {
				return err
			}
			edgeIDMap := map[string]bool{}
			// always allow edge to get itself
			if ok, edgeID := base.IsEdgeRequest(authContext); ok && edgeID != "" {
				edgeIDMap[edgeID] = true
				edgeIDs = append(edgeIDs, edgeID)
			}
			for _, project := range projects {
				for _, eid := range project.EdgeIDs {
					if !edgeIDMap[eid] {
						edgeIDMap[eid] = true
						edgeIDs = append(edgeIDs, eid)
					}
				}
			}
		}
	}
	dataSources := []model.DataSource{}
	totalCount := 0
	queryParam := model.GetEntitiesQueryParam(req)
	if len(edgeIDs) != 0 || (auth.IsInfraAdminRole(authContext) && projectID == "") {
		dataSources, totalCount, err = dbAPI.getDataSourcesByEdgesForQuery(context, edgeIDs, queryParam)
		if err != nil {
			return err
		}
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeDataSource}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.DataSourceListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		DataSourceListV2:          model.DataSourcesByID(dataSources).ToV2(),
	}
	return json.NewEncoder(w).Encode(r)
}

/*** End of common shared private methods for data sources ***/

// SelectAllDataSources select all data sources for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllDataSources(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error) {
	// Page size is set to a large valeu for now
	return dbAPI.getDataSources(context, "", "", "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllDataSourcesW select all data sources for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getDataSourcesW(context, "", "", "", w, req)
}

// SelectAllDataSourcesWV2 select all data sources for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getDataSourcesWV2(context, "", "", "", w, req)
}

// SelectAllDataSourcesForEdge select all data sources for the given edge
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesForEdge(context context.Context, edgeID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error) {
	return dbAPI.getDataSources(context, edgeID, "", "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllDataSourcesForEdgeW select all data sources for the given edge, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesForEdgeW(context context.Context, edgeID string, w io.Writer, req *http.Request) error {
	return dbAPI.getDataSourcesW(context, edgeID, "", "", w, req)
}

// SelectAllDataSourcesForEdgeWV2 select all data sources for the given edge, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesForEdgeWV2(context context.Context, edgeID string, w io.Writer, req *http.Request) error {
	return dbAPI.getDataSourcesWV2(context, edgeID, "", "", w, req)
}

// SelectAllDataSourcesForProject select all data sources for the given project
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error) {
	return dbAPI.getDataSources(context, "", "", projectID, base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllDataSourcesForProjectW select all data sources for the given project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getDataSourcesW(context, "", "", projectID, w, req)
}

// SelectAllDataSourcesForProjectWV2 select all data sources for the given project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataSourcesForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getDataSourcesWV2(context, "", "", projectID, w, req)
}

// GetDataSource get a data source object in the DB
func (dbAPI *dbObjectModelAPI) GetDataSource(context context.Context, id string) (model.DataSource, error) {
	if len(id) == 0 {
		return model.DataSource{}, errcode.NewBadRequestError("dataSourceID")
	}
	dataSources, err := dbAPI.getDataSources(context, "", id, "", base.StartPageToken, base.MaxRowsLimit, nil)
	if err != nil {
		return model.DataSource{}, err
	}
	if len(dataSources) == 0 {
		return model.DataSource{}, errcode.NewRecordNotFoundError(id)
	}
	return dataSources[0], nil
}

// dataSourcesByEndpoints fetched data sources for given endpoint objects
func (dbAPI *dbObjectModelAPI) SelectDataSourcesByEndpoints(context context.Context, endpoints []model.DataIfcEndpoint) ([]model.DataSource, error) {
	if len(endpoints) == 0 {
		return []model.DataSource{}, nil
	}
	var loggedErr error
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()

	ifcIDs := make([]string, 0, len(endpoints))
	for _, o := range endpoints {
		ifcIDs = append(ifcIDs, o.ID)
	}
	glog.V(5).Infof("fetching data ifcs %v", ifcIDs)

	datasources, err := dbAPI.getDataSourcesByIDs(context, ifcIDs)

	if err != nil {
		loggedErr = fmt.Errorf("failed to find endpoints (id in %v). %s", ifcIDs, err.Error())
		return nil, errcode.NewInternalError(loggedErr.Error())
	}
	return datasources, nil
}

// GetDataSourceW get a data source object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataSourceW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("dataSourceID")
	}
	return dbAPI.getDataSourcesW(context, "", id, "", w, req)
}

// GetDataSourceWV2 get a data source object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataSourceWV2(context context.Context, dataSourceID string, w io.Writer, req *http.Request) error {
	if len(dataSourceID) == 0 {
		return errcode.NewBadRequestError("dataSourceID")
	}
	dataSources, err := dbAPI.getDataSources(context, "", dataSourceID, "", base.StartPageToken, base.MaxRowsLimit, nil)
	if err != nil {
		return err
	}
	if len(dataSources) == 0 {
		return errcode.NewRecordNotFoundError(dataSourceID)
	}
	return json.NewEncoder(w).Encode(dataSources[0].ToV2())
}

func (dbAPI *dbObjectModelAPI) updateNonU2EdgeDNS(ctx context.Context, dsID string,
	artifactData map[string]interface{}) error {
	ds, err := dbAPI.GetDataSource(ctx, dsID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get datasource %s. Error: %s"),
			dsID, err.Error())
		return err
	}
	edge, err := dbAPI.GetEdge(ctx, ds.EdgeID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get edge info for %s. Error: %s"), ds.EdgeID, err.Error())
		return err
	}

	if artifactData["host"] == edge.IPAddress {
		artifactData["host"] = generateEdgeDNSName(edge)
		glog.Infof(base.PrefixRequestID(ctx, "Setting artifacts to %v"), artifactData["host"])
	} else {
		glog.Infof(base.PrefixRequestID(ctx, "Ignoring aritfcat update for edge, Artifact host %v, edge ip %v"),
			artifactData["host"], edge.IPAddress)
	}
	return nil
}

// CreateDataSourceArtifact creates or overwrites new or existing artifact record for the datasource.
// Only an edge can call this method
func (dbAPI *dbObjectModelAPI) CreateDataSourceArtifact(ctx context.Context, i interface{} /* *model.DataSourceArtifact */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DataSourceArtifact)
	if !ok {
		return resp, errcode.NewInternalError("CreateDataSourceArtifacts: type error")
	}
	isEdgeReq, edgeID := base.IsEdgeRequest(authContext)
	// Only an edge must be able to call this API
	if !isEdgeReq {
		return resp, errcode.NewPermissionDeniedError("role")
	}
	doc := *p
	tenantID := authContext.TenantID
	ds, err := dbAPI.GetDataSource(ctx, doc.DataSourceID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get datasource %s. Error: %s"), doc.DataSourceID, err.Error())
		return resp, err
	}
	if edgeID != ds.EdgeID {
		glog.Errorf(base.PrefixRequestID(ctx, "Data source %s (Edge %s) is not associated with edge %s."), doc.DataSourceID, ds.EdgeID, edgeID)
		return resp, errcode.NewPermissionDeniedError("edgeID")
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := base.DeleteTxn(ctx, tx, "data_source_artifact_model", map[string]interface{}{"tenant_id": tenantID, "data_source_id": doc.DataSourceID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete existing artifact for data source %s"), doc.DataSourceID)
			return err
		}
		dataSourceArtifactDBO := &DataSourceArtifactDBO{}
		err = base.Convert(&doc, dataSourceArtifactDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert artifact for data source %s. Errpr: %s"), doc.DataSourceID, err.Error())
			return err
		}
		dataSourceArtifactDBO.TenantID = tenantID
		_, err = tx.NamedExec(ctx, queryMap["CreateDataSourceArtifact"], dataSourceArtifactDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create artifact for data source %s. Error: %s"), doc.DataSourceID, err.Error())
			return errcode.TranslateDatabaseError(doc.DataSourceID, err)
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.DataSourceID
	if callback != nil {
		go callback(ctx, doc)
	}
	return resp, nil
}

// CreateDataSourceArtifactWV2 creates a data source artifact in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataSourceArtifactWV2(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateDataSourceArtifact, &model.DataSourceArtifact{}, w, r, callback)
}

// GetDataSourceArtifact returns the artifact for the datasource with the ID
func (dbAPI *dbObjectModelAPI) GetDataSourceArtifact(ctx context.Context, dataSourceID string) (model.DataSourceArtifact, error) {
	resp := model.DataSourceArtifact{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	// This verifies the tenant and data source association applying the auth filter
	_, err = dbAPI.GetDataSource(ctx, dataSourceID)
	if err != nil {
		return resp, err
	}
	param := DataSourceArtifactDBO{DataSourceID: dataSourceID}
	param.TenantID = authContext.TenantID
	artifactDBOs := []DataSourceArtifactDBO{}
	err = dbAPI.Query(ctx, &artifactDBOs, queryMap["SelectDataSourceArtifact"], param)
	if err != nil {
		return resp, err
	}
	if len(artifactDBOs) != 1 {
		// Do not report error
		// There can be data sources without any artifacts
		return resp, nil
	}
	artifactDBO := &artifactDBOs[0]
	artifact := &model.DataSourceArtifact{}
	err = base.Convert(artifactDBO, artifact)
	if err != nil {
		return resp, err
	}
	data := artifact.Data

	/*
		- if non-u2 edge {
			// Enhance data with data["host"] = "edge-<id>.<dev/stage//>.xidata.io"
		}
	*/
	err = dbAPI.updateNonU2EdgeDNS(ctx, dataSourceID, data)
	if err != nil {
		return resp, err
	}

	sharedSecret, ok := data["secret"]
	if ok {
		// Enchance data with additional values
		data["expiry"] = base.RoundedNow().Add(time.Hour * 24 * 7).Unix()
		data["token"] = base.GetBase64URLEncodedMD5Hash(fmt.Sprintf("%d %s", data["expiry"], sharedSecret))
	}

	outputData, err := base.SubstituteValues(ctx, data, func(key string) bool {
		// Nested field
		if strings.HasSuffix(key, "__url") {
			return true
		}
		if strings.HasSuffix(key, "__name") {
			return true
		}
		return false
	})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to substitute values for artifacts for data source %s. Error: %s"), dataSourceID, err.Error())
		return resp, err
	}
	resp.DataSourceID = dataSourceID
	resp.Data = outputData
	resp.Version = artifact.Version
	resp.CreatedAt = artifact.CreatedAt
	return resp, nil
}

// GetDataSourceArtifactsWV2 returns the artifacts for the given datasource ID
func (dbAPI *dbObjectModelAPI) GetDataSourceArtifactWV2(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	dataSourceArtifact, err := dbAPI.GetDataSourceArtifact(ctx, id)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get artifact for data source %s. Error: %s"))
		return err
	}
	return json.NewEncoder(w).Encode(dataSourceArtifact)
}

func generateEdgeDNSName(edge model.Edge) string {
	domain := *config.Cfg.EdgeDNSDomain
	return fmt.Sprintf("edge-%v.%v", edge.ID, domain)
}

func needDNSSetup(edge model.Edge, dataSource model.DataSource) bool {
	// Ignore u2 edges
	if edge.Type != nil && *(edge.Type) != string(model.RealTargetType) {
		return false
	}

	// Ignore non-interface data sources or interfaces with no ports
	if dataSource.IfcInfo == nil || len(dataSource.IfcInfo.Ports) == 0 {
		return false
	}
	return true
}

func (dbAPI *dbObjectModelAPI) countDnsIfcOnEdge(ctx context.Context, edgeID string) (int, error) {
	datasources, err := dbAPI.getDataSources(ctx, edgeID, "", "", base.StartPageToken, base.MaxRowsLimit, nil)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, ds := range datasources {
		if ds.IfcInfo == nil || len(ds.IfcInfo.Ports) == 0 {
			continue
		}
		count++
	}
	return count, nil
}

func (dbAPI *dbObjectModelAPI) registerNonU2DNS(ctx context.Context, edge model.Edge, dataSource model.DataSource) error {
	if !needDNSSetup(edge, dataSource) {
		return nil
	}

	glog.Infof(base.PrefixRequestID(ctx, "Creating Route53 entry for edge %v, %v => %v"),
		edge.ID, generateEdgeDNSName(edge), edge.IPAddress)

	domain := *config.Cfg.EdgeDNSDomain
	return utils.UpsertRoute53Entry(ctx, domain, generateEdgeDNSName(edge), edge.IPAddress)
}

func (dbAPI *dbObjectModelAPI) deleteNonU2DNS(ctx context.Context, edge model.Edge, dataSource model.DataSource) error {
	if !needDNSSetup(edge, dataSource) {
		return nil
	}

	ifcCount, err := dbAPI.countDnsIfcOnEdge(ctx, edge.ID)
	if err != nil {
		return err
	}

	// 1 because this is the last ifc that is being deleted, still a minor race between creation and deletion ignore for now
	if ifcCount != 1 {
		glog.Infof(base.PrefixRequestID(ctx, "Skipping delete of Route53 entry as edge %v, still has %v ifc(s) in use"),
			edge.ID, ifcCount)
		return nil
	}

	glog.Infof(base.PrefixRequestID(ctx, "Deleting Route53 entry for edge %v, %v => %v"),
		edge.ID, generateEdgeDNSName(edge), edge.IPAddress)

	domain := *config.Cfg.EdgeDNSDomain
	return utils.DeleteRoute53Entry(ctx, domain, generateEdgeDNSName(edge), edge.IPAddress)
}

// CreateDataSource creates a data source object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataSource(context context.Context, i interface{} /* *model.DataSource */, callback func(context.Context, interface{}) error) (interface{}, error) {
	glog.Infof("Creating data sources")
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DataSource)
	if !ok {
		return resp, errcode.NewInternalError("CreateDataSource: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateDataSource doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateDataSource doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateDataSource(&doc)
	if err != nil {
		return resp, err
	}
	if !ReK8sName.MatchString(doc.Name) {
		return resp, errcode.NewBadRequestError("name")
	}
	// edge must belong to the same tenant
	edge, err := dbAPI.GetEdge(context, doc.EdgeID)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataSource,
		meta.OperationCreate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}

	// Register Route53 DNS for non-u2 edge
	err = dbAPI.registerNonU2DNS(context, edge, doc)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		dataSourceDBO := DataSourceDBO{}
		err := base.Convert(&doc, &dataSourceDBO)
		if err != nil {
			return err
		}
		if doc.IfcInfo != nil {
			err := base.Convert(&doc.IfcInfo, &dataSourceDBO)
			if err != nil {
				return err
			}
		}
		_, err = tx.NamedExec(context, queryMap["CreateDataSource"], dataSourceDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating datasource for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}

		err = dbAPI.createDataSourceIfcPorts(context, tx, doc)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating datasource ifc ports for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}

		return dbAPI.createDataSourceFields(context, tx, doc)
	})
	if err != nil {
		return resp, err
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addDataSourceAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateDataSourceV2 creates an application object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataSourceV2(context context.Context, i interface{} /* *model.DataSourceV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.DataSourceV2)
	if !ok {
		return model.CreateDocumentResponse{}, errcode.NewInternalError("CreateDataSourceV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.CreateDataSource(context, &doc, callback)
}

// CreateDataSourceW creates a data source object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataSourceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDataSource, &model.DataSource{}, w, r, callback)
}

// CreateDataSourceWV2 creates a data source object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateDataSourceWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateDataSourceV2), &model.DataSourceV2{}, w, r, callback)
}

// DataSourceFieldDeleteParams encapsultes the params for deleting a data source field
type DataSourceFieldDeleteParams struct {
	DataSourceID string
	Name         *string
	FieldType    string
	MQTTTopic    *string
}

// deleteDataSourceFieldByParams deletes data source field based on given param
// data source ID and field type are required fields in the params to save from unintended deletion
func (dbAPI *dbObjectModelAPI) deleteDataSourceFieldByParams(context context.Context, tx *base.WrappedTx, params DataSourceFieldDeleteParams) error {
	if params.DataSourceID == "" || params.FieldType == "" {
		return fmt.Errorf("'DataSourceID' and 'FieldType' is required to delete a data source field")
	}

	if params.Name == nil && params.MQTTTopic == nil {
		return fmt.Errorf("'Name' OR 'MQTTTopic' is required to delete a data source field")
	}

	deleteParams := map[string]interface{}{
		"data_source_id": params.DataSourceID,
		"field_type":     params.FieldType,
	}

	if params.Name != nil {
		deleteParams["name"] = *params.Name
	}
	if params.MQTTTopic != nil {
		deleteParams["mqtt_topic"] = *params.MQTTTopic
	}
	_, err := base.DeleteTxn(context, tx, "data_source_field_model", deleteParams)
	return err
}

// checkRemovedFieldsInUse checks for any removed fields that may be in use by other entities like app endpoints
func (dbAPI *dbObjectModelAPI) checkRemovedFieldsInUse(ctx context.Context, dsID string, oldFields, newFields []model.DataSourceFieldInfo) error {
	newFieldByNames := make(map[string]bool)

	for _, newF := range newFields {
		newFieldByNames[newF.Name] = true
	}

	// fields removed
	removedFields := make(map[string]bool)
	for _, old := range oldFields {
		if _, ok := newFieldByNames[old.Name]; !ok {
			removedFields[old.Name] = true
		}
	}

	glog.V(5).Infof("removed fields from data source: %+v", removedFields)

	// nothing to do if no fields are removed
	if len(removedFields) == 0 {
		return nil
	}

	fields, err := dbAPI.FetchApplicationEndpointsByDataSource(ctx, dsID)
	if err != nil {
		return fmt.Errorf("failed to find application endpoints for data source %s", dsID)
	}

	for _, f := range fields {
		if removedFields[f.Name] {
			// TODO: Improve the error to provide the exact app name using this field/topic
			glog.Errorf("field %+v is used by an application. cannot remove this field", f)
			return fmt.Errorf("field %s, topic: %s is used by one or more application", f.Name, f.Value)
		}
	}
	return nil
}

// checkOutDataIfcClaims checks if any of the existing topics for the given data source are associated with
// any topic claims. If yes, then it makes sure that those topics are not removed.
// this makes sure that the pipelines using such data source topics as out data ifc, can continue to work.
func (dbAPI *dbObjectModelAPI) checkOutDataIfcClaims(context context.Context, oldDs model.DataSource, newDS model.DataSource) error {
	if oldDs.IfcInfo == nil || oldDs.IfcInfo.Kind == model.DataIfcEndpointKindIn {
		return nil
	}
	claims, err := dbAPI.fetchDataIfcTopicClaim(context, queryMap["SelectDataIfcClaimByDataSourceID"],
		&DataIfcTopicClaimDBO{TenantID: oldDs.TenantID, DataSourceID: oldDs.ID},
	)
	if err != nil {
		return errcode.NewInternalError(fmt.Sprintf("failed to find topic claims for data source %s as part of validation",
			oldDs.ID),
		)
	}
	topics := make(map[string]bool)

	for _, f := range newDS.Fields {
		topics[f.MQTTTopic] = true
	}

	for _, claim := range claims {
		// Only do this for data streams. applications are symlinked via field names and handle data source
		// updates more gracefully
		if claim.DataStreamID != nil && !topics[claim.Topic] {
			return errcode.NewBadRequestExError("Fields",
				fmt.Sprintf("topic %s is used by data stream %s", claim.Topic, *claim.DataStreamID),
			)
		}
	}
	return nil
}

// UpdateDataSource updates a data source object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataSource(context context.Context, i interface{} /* *model.DataSource */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DataSource)
	if !ok {
		return resp, errcode.NewInternalError("UpdateDataSource: type error")
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

	err = model.ValidateDataSource(&doc)
	if err != nil {
		return resp, err
	}
	if !ReK8sName.MatchString(doc.Name) {
		return resp, errcode.NewBadRequestError("name")
	}
	ds, err := dbAPI.GetDataSource(context, doc.ID)
	if err != nil {
		return resp, err
	}
	if ds.EdgeID != doc.EdgeID {
		// edge id change not allowed
		return resp, errcode.NewBadRequestError("edgeId")
	}
	err = dbAPI.checkRemovedFieldsInUse(context, ds.ID, ds.Fields, doc.Fields)
	if err != nil {
		return resp, errcode.NewBadRequestExError("Fields", err.Error())
	}
	err = dbAPI.checkOutDataIfcClaims(context, ds, doc)
	if err != nil {
		return resp, errcode.NewBadRequestExError("Fields", err.Error())
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataSource,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := base.DeleteTxn(context, tx, "data_source_field_selector_model", map[string]interface{}{"data_source_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting datasource field selectors for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return err
		}
		_, err = base.DeleteTxn(context, tx, "data_source_field_model", map[string]interface{}{"data_source_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting datasource fields for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return err
		}
		_, err = base.DeleteTxn(context, tx, "data_source_ifc_port_model", map[string]interface{}{"data_source_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting datasource ifc ports for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return err
		}
		dataSourceDBO := DataSourceDBO{}
		err = base.Convert(&doc, &dataSourceDBO)
		if err != nil {
			return err
		}

		if doc.IfcInfo != nil {
			err := base.Convert(&doc.IfcInfo, &dataSourceDBO)
			if err != nil {
				return err
			}
		}

		_, err = tx.NamedExec(context, queryMap["UpdateDataSource"], &dataSourceDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in updating datasource for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createDataSourceIfcPorts(context, tx, doc)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in updating datasource ifc ports for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		return dbAPI.createDataSourceFields(context, tx, doc)

	})
	if err != nil {
		return resp, err
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addDataSourceAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateDataSourceV2 creates an application object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataSourceV2(context context.Context, i interface{} /* *model.DataSourceV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.DataSourceV2)
	if !ok {
		return model.CreateDocumentResponse{}, errcode.NewInternalError("UpdateDataSourceV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.UpdateDataSource(context, &doc, callback)
}

// UpdateDataSourceW updates a data source object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDataSourceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDataSource, &model.DataSource{}, w, r, callback)
}

// UpdateDataSourceWV2 updates a data source object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateDataSourceWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateDataSourceV2), &model.DataSourceV2{}, w, r, callback)
}

// DeleteDataSource delete a data source object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDataSource(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataSource,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}

	// Validate that the data source fields are not used by other entities
	endpoints, err := dbAPI.FetchApplicationEndpointsByDataSource(context, id)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("failed to find application endpoints associated with data source %s. %s", id, err.Error()))
	}

	// TODO: Validate that the data source is not associated with a pipeline.

	if len(endpoints) > 0 {
		glog.Errorf(base.PrefixRequestID(context, "data source %s is used by one of more applications endpoints: %+v"), id, endpoints)
		return resp, errcode.NewPreConditionFailedError(
			fmt.Sprintf("cannot delete data source %s as one or more application endpoint(s) depend on it. Endpoints using this data source: %+v", id, endpoints),
		)
	}

	edgeID, err := dbAPI.GetDataSourceEdgeID(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	doc := model.DataSource{
		EdgeBaseModel: model.EdgeBaseModel{
			BaseModel: model.BaseModel{
				TenantID: authContext.TenantID,
				ID:       id,
			},
			EdgeID: edgeID,
		},
	}

	ds, err := dbAPI.GetDataSource(context, id)
	if err != nil {
		return resp, err
	}

	// edge must belong to the same tenant
	// SHLK-595 unblock datasource deletion if there is no node in multinode
	edge, err1 := dbAPI.GetEdge(context, doc.EdgeID)
	if err1 != nil {
		if !errcode.IsRecordNotFound(err1) {
			return resp, err1
		}
	}
	if err1 == nil {
		// Edge is found
		err = dbAPI.deleteNonU2DNS(context, edge, ds)
		if err != nil {
			glog.Infof("Failing datasource delete as dns entry could not be removed, err %v", err)
			return resp, err
		}
	}

	// FIXME: We don't validate if any of the topics of this data source are claimed by an application or datastream.
	// This can adversaly affect input/output for an application or a datastream  after deletion of data source.

	result, err := DeleteEntity(context, dbAPI, "data_source_model", "id", id, doc, callback)
	if err == nil {
		GetAuditlogHandler().addDataSourceAuditLog(dbAPI, context, ds, DELETE)
	}
	return result, err
}

// DeleteDataSourceW delete a data source object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDataSourceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDataSource, id, w, callback)
}

// DeleteDataSourceWV2 delete a data source object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteDataSourceWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteDataSource), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) getDataSourcesByIDs(ctx context.Context, dataSourceIDs []string) ([]model.DataSource, error) {
	dataSources := []model.DataSource{}
	if len(dataSourceIDs) == 0 {
		return dataSources, nil
	}

	dataSourceDBOs := []DataSourceDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &dataSourceDBOs, "data_source_model", dataSourceIDs); err != nil {
		return nil, err
	}

	dataSources, _, err := dbAPI.dataSourceDBOsToDS(ctx, dataSourceDBOs)
	return dataSources, err
}

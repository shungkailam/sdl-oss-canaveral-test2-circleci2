package api

import (
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
	"strings"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	funk "github.com/thoas/go-funk"
)

const entityTypeDataStream = "datastream"

func init() {
	// If state is null, it is DEPLOY by default
	queryMap["SelectDataStreamsTemplate"] = `SELECT * FROM data_stream_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND project_id IN (:project_ids) AND (:state = '' OR state = :state OR (state is null AND :state = 'DEPLOY')) %s`
	// If state is null, it is DEPLOY by default
	queryMap["SelectDataStreamsByProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM data_stream_model WHERE tenant_id = :tenant_id AND project_id IN (:project_ids) AND (:state = '' OR state = :state OR (state is null AND :state = 'DEPLOY')) %s`

	queryMap["SelectDataIfcClaimByDataStream"] = `SELECT data_source_id, data_stream_id, tenant_id, topic FROM data_source_topic_claim WHERE tenant_id = :tenant_id AND data_stream_id = :data_stream_id`
	queryMap["SelectDataIfcClaimByDataStreams"] = `SELECT data_source_id, data_stream_id, tenant_id, topic FROM data_source_topic_claim WHERE data_stream_id IN (:data_stream_ids)`

	queryMap["SelectDataStreamOriginSelectors"] = `SELECT data_stream_origin_model.id, data_stream_origin_model.data_stream_id, data_stream_origin_model.category_value_id, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
		FROM data_stream_origin_model JOIN category_value_model ON data_stream_origin_model.category_value_id = category_value_model.id WHERE data_stream_origin_model.data_stream_id = :data_stream_id`

	queryMap["SelectDataStreamsOriginSelectors"] = `SELECT data_stream_origin_model.id, data_stream_origin_model.data_stream_id, data_stream_origin_model.category_value_id, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
		FROM data_stream_origin_model JOIN category_value_model ON data_stream_origin_model.category_value_id = category_value_model.id WHERE data_stream_origin_model.data_stream_id IN (:data_stream_ids)`

	queryMap["CreateDataStream"] = `INSERT INTO data_stream_model (id, version, tenant_id, name, description, data_type, origin, origin_id, destination, cloud_type, cloud_creds_id, aws_cloud_region, gcp_cloud_region, edge_stream_type, aws_stream_type, az_stream_type, gcp_stream_type, size, enable_sampling, sampling_interval, transformation_args_list, data_retention, project_id, created_at, updated_at, end_point, state) VALUES (:id, :version, :tenant_id, :name, :description, :data_type, :origin, :origin_id, :destination, :cloud_type, :cloud_creds_id, :aws_cloud_region, :gcp_cloud_region, :edge_stream_type, :aws_stream_type, :az_stream_type, :gcp_stream_type, :size, :enable_sampling, :sampling_interval, :transformation_args_list, :data_retention, :project_id, :created_at, :updated_at, :end_point, :state)`

	queryMap["UpdateDataStream"] = `UPDATE data_stream_model SET version = :version, tenant_id = :tenant_id, name = :name, description = :description, data_type = :data_type, origin = :origin, origin_id = :origin_id, destination = :destination, cloud_type = :cloud_type, cloud_creds_id = :cloud_creds_id, aws_cloud_region = :aws_cloud_region, gcp_cloud_region = :gcp_cloud_region, edge_stream_type = :edge_stream_type, aws_stream_type = :aws_stream_type, az_stream_type = :az_stream_type, gcp_stream_type = :gcp_stream_type, size = :size, enable_sampling = :enable_sampling, sampling_interval= :sampling_interval, transformation_args_list = :transformation_args_list, data_retention = :data_retention, project_id = :project_id, updated_at = :updated_at, end_point = :end_point, state = :state WHERE tenant_id = :tenant_id AND id = :id`

	orderByHelper.Setup(entityTypeDataStream, []string{"id", "version", "created_at", "updated_at", "name", "description", "data_type", "origin", "origin_id", "destination", "cloud_type", "cloud_creds_id", "aws_cloud_region", "gcp_cloud_region", "edge_stream_type", "aws_stream_type", "az_stream_type", "gcp_stream_type", "project_id", "end_point"})
}

// DataStreamDBO is DB object model for data stream
type DataStreamDBO struct {
	model.BaseModelDBO
	Name                   string         `json:"name" db:"name"`
	Description            string         `json:"description" db:"description"`
	DataType               string         `json:"dataType" db:"data_type"`
	Origin                 string         `json:"origin" db:"origin"`
	OriginID               *string        `json:"originId" db:"origin_id"`
	Destination            string         `json:"destination" db:"destination"`
	CloudType              *string        `json:"cloudType" db:"cloud_type"`
	CloudCredsID           *string        `json:"cloudCredsId" db:"cloud_creds_id"`
	AWSCloudRegion         *string        `json:"awsCloudRegion" db:"aws_cloud_region"`
	GCPCloudRegion         *string        `json:"gcpCloudRegion" db:"gcp_cloud_region"`
	EdgeStreamType         *string        `json:"edgeStreamType" db:"edge_stream_type"`
	AWSStreamType          *string        `json:"awsStreamType" db:"aws_stream_type"`
	AZStreamType           *string        `json:"azStreamType" db:"az_stream_type"`
	GCPStreamType          *string        `json:"gcpStreamType" db:"gcp_stream_type"`
	Size                   float64        `json:"size" db:"size"`
	EnableSampling         bool           `json:"enableSampling" db:"enable_sampling"`
	SamplingInterval       *int           `json:"samplingInterval" db:"sampling_interval"`
	TransformationArgsList types.JSONText `json:"transformationArgsList" db:"transformation_args_list"`
	DataRetention          types.JSONText `json:"dataRetention" db:"data_retention"`
	ProjectID              *string        `json:"projectId" db:"project_id"`
	EndPoint               *string        `json:"endPoint" db:"end_point"`
	State                  *string        `json:"state" db:"state"`
}

type DataStreamIdsParam struct {
	DataStreamIDs []string `json:"dataStreamIds" db:"data_stream_ids"`
}

func (app DataStreamDBO) GetProjectID() string {
	if app.ProjectID != nil {
		return *app.ProjectID
	}
	return ""
}

type DataStreamProjects struct {
	DataStreamDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for datastream
func getDataStreamDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	isEdgeReq, _ := base.IsEdgeRequest(authContext)
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	// State pointer must be set for query to work
	param := DataStreamDBO{BaseModelDBO: tenantModel, State: base.StringPtr("")}
	if isEdgeReq {
		param.State = model.DeployEntityState.StringPtr()
	}
	var projectIDs []string
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		projectIDs = []string{projectID}
	} else {
		projectIDs = auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return base.InQueryParam{}, nil
		}
	}
	return base.InQueryParam{
		Param: DataStreamProjects{
			DataStreamDBO: param,
			ProjectIDs:    projectIDs,
		},
		Key:     "SelectDataStreamsTemplate",
		InQuery: true,
	}, nil
}

func validateDataStream(dbAPI *dbObjectModelAPI, context context.Context, doc *model.DataStream) error {
	if doc.Destination == model.DestinationCloud {
		cloudCreds, err := dbAPI.GetCloudCreds(context, doc.CloudCredsID)
		if err != nil {
			return errcode.NewBadRequestError("cloudCredsId")
		}
		if cloudCreds.Type != doc.CloudType {
			return errcode.NewBadRequestError("cloudType<>")
		}
		project, err := dbAPI.GetProject(context, doc.ProjectID)
		if err != nil {
			return errcode.NewBadRequestError("projectId")
		}
		if !funk.Contains(project.CloudCredentialIDs, doc.CloudCredsID) {
			return errcode.NewPermissionDeniedError("RBAC/CloudProfile")
		}
	}
	// validate TransformationArgsList
	if len(doc.TransformationArgsList) != 0 {
		scripts, err := dbAPI.SelectAllScriptsForProject(context, doc.ProjectID, nil)
		if err != nil {
			return err
		}
		scriptIDsMap := map[string]bool{}
		for _, script := range scripts {
			scriptIDsMap[script.ID] = true
		}
		for _, ta := range doc.TransformationArgsList {
			script, err := dbAPI.GetScript(context, ta.TransformationID)
			if err != nil {
				return err
			}
			if !scriptIDsMap[ta.TransformationID] {
				if script.ProjectID != "" {
					// RBAC error - script is not global and is not in project
					return errcode.NewBadRequestExError("transformationArgsList", fmt.Sprintf("Transformation %s not accessible in project %s", ta.TransformationID, doc.ProjectID))
				}
			}
			if len(ta.Args) != len(script.Params) {
				return errcode.NewBadRequestExError("transformationArgsList", fmt.Sprintf("Transformation %s args length mismatch", ta.TransformationID))
			}
			for i := range ta.Args {
				tai := ta.Args[i]
				spi := script.Params[i]
				if tai.Name != spi.Name || tai.Type != spi.Type {
					return errcode.NewBadRequestExError("transformationArgsList", fmt.Sprintf("Transformation %s args mismatch", ta.TransformationID))
				}
			}
		}
	}
	// validate origin data stream
	if doc.Origin == "Data Stream" {
		id := doc.OriginID
		dataStreams, err := dbAPI.getDataStreams(context, "", id, nil)
		if err != nil {
			return err
		}
		if len(dataStreams) == 0 {
			return errcode.NewBadRequestExError("originID", fmt.Sprintf("Origin data stream %s not found", doc.OriginID))
		}
		origin := dataStreams[0]
		if origin.ProjectID != doc.ProjectID {
			return errcode.NewBadRequestExError("originID", fmt.Sprintf("Origin data stream %s in different project", doc.OriginID))
		}
		if origin.Destination != model.DestinationEdge {
			return errcode.NewBadRequestExError("originID", fmt.Sprintf("Origin data stream %s not publishing to edge", doc.OriginID))
		}
		if origin.EdgeStreamType != "None" {
			return errcode.NewBadRequestExError("originID", fmt.Sprintf("Origin data stream %s not a real time data stream", doc.OriginID))
		}
	}

	// Make sure that the data source exists for the given data interface endpoint
	if doc.Destination == model.DestinationDataInterface {
		var errFieldName, dataSourceID string
		if len(doc.DataIfcEndpoints) > 0 {
			errFieldName = "DataIfcEndpoints"
			dataSourceID = doc.DataIfcEndpoints[0].ID
		}
		datasources, err := dbAPI.getDataSourcesByIDs(context, []string{dataSourceID})
		if err != nil {
			return errcode.NewInternalError(fmt.Sprintf("failed to fetch data source %s for data stream endpoint %s", dataSourceID, doc.EndPoint))
		}
		if len(datasources) == 0 {
			return errcode.NewBadRequestExError(errFieldName, fmt.Sprintf("Data source %s does not exist", dataSourceID))
		}
		// Only "OUT" data Ifc is allowed
		if datasources[0].IfcInfo == nil || datasources[0].IfcInfo.Kind != model.DataIfcEndpointKindOut {
			return errcode.NewBadRequestExError("OutDataIfc", fmt.Sprintf("incompatible data source %s to be used as data out Ifc", datasources[0].Name))
		}
	}
	if doc.EdgeStreamType == "Kafka" {
		// Make sure Kafka is enabled when selecting Kafka as destination
		if enabledServices, err := dbAPI.enabledServicesInProject(context, doc.ProjectID); err != nil {
			return err
		} else if enabledServices["kafka"] == false {
			return errcode.NewBadRequestExError("EdgeStreamType", "Kafka service not enabled for project")
		}
	}
	return nil
}

func setDefaultFields(dataStreamDBO *DataStreamDBO) {
	if dataStreamDBO.Destination == model.DestinationEdge {
		dataStreamDBO.CloudCredsID = nil
	}
}

func (dbAPI *dbObjectModelAPI) populateOriginSelectors(ctx context.Context, dataStream *model.DataStream) error {
	datastreamOriginSelectors := []DataStreamOriginSelectorDBO{}
	err := dbAPI.Query(ctx, &datastreamOriginSelectors, queryMap["SelectDataStreamOriginSelectors"], DataStreamOriginSelectorDBO{DataStreamID: dataStream.BaseModel.ID})
	if err != nil {
		return err
	}
	for _, datastreamOriginSelector := range datastreamOriginSelectors {
		dataStream.OriginSelectors = append(dataStream.OriginSelectors, datastreamOriginSelector.CategoryInfo)
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) populateOutDataIfc(ctx context.Context, dataStreams []model.DataStream) ([]model.DataStream, error) {
	if len(dataStreams) == 0 {
		return dataStreams, nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return dataStreams, err
	}
	isEdgeReq, edgeID := base.IsEdgeRequest(authContext)

	idsParam := DataStreamIdsParam{DataStreamIDs: make([]string, 0, len(dataStreams))}
	dataStreamsIdxs := make(map[string]int)
	for i, ds := range dataStreams {
		idsParam.DataStreamIDs = append(idsParam.DataStreamIDs, ds.ID)
		dataStreamsIdxs[ds.ID] = i
		dataStreams[i].DataIfcEndpoints = []model.DataIfcEndpoint{}
	}

	rtnClaims := []DataIfcTopicClaimDBO{}
	err = dbAPI.QueryIn(ctx, &rtnClaims, queryMap["SelectDataIfcClaimByDataStreams"], idsParam)
	if err != nil {
		return dataStreams, errcode.NewInternalError(fmt.Sprintf("failed to find data ifc claims for data streams (%s). %s", strings.Join(idsParam.DataStreamIDs, ","), err.Error()))
	}

	toDropIndicesMap := map[int]struct{}{}
	for _, claim := range rtnClaims {
		if claim.DataStreamID == nil {
			glog.Warning(base.PrefixRequestID(ctx, "data stream ID not set claim: %+v"), claim)
		}
		idx := dataStreamsIdxs[*claim.DataStreamID]

		// TODO: Fetch the Value field from data source field model instead of assumming it is the same as topic
		dataStreams[idx].DataIfcEndpoints = append(dataStreams[idx].DataIfcEndpoints,
			model.DataIfcEndpoint{ID: claim.DataSourceID, Name: claim.Topic, Value: claim.Topic},
		)

		// Backwards compatibility for READ-only clients like Xi IoT sensor app
		dataIfc, err := dbAPI.GetDataSource(ctx, claim.DataSourceID)
		if err != nil {
			if isEdgeReq {
				// for edge request, this likely means the dataIfc is for another edge, so drop the data stream
				toDropIndicesMap[idx] = struct{}{}
			} else {
				return dataStreams, errcode.NewInternalError(fmt.Sprintf("failed to find data source %s. %s", claim.DataSourceID, err.Error()))
			}
		} else {
			if !isEdgeReq || dataIfc.EdgeID == edgeID {
				dataStreams[idx].OutDataIfc = &dataIfc
				glog.Infof(base.PrefixRequestID(ctx, "data stream data ifc endpoint: %+v"), dataStreams[idx].DataIfcEndpoints)
			} else if isEdgeReq {
				// for edge request, drop the data stream when the dataIfc is for another edge
				toDropIndicesMap[idx] = struct{}{}
			}
		}
	}
	if len(toDropIndicesMap) != 0 {
		size := len(dataStreams) - len(toDropIndicesMap)
		dss := make([]model.DataStream, size)
		if size != 0 {
			i := 0
			for j := 0; j < len(dataStreams); j++ {
				if _, b := toDropIndicesMap[j]; !b {
					dss[i] = dataStreams[j]
					i++
				}
			}
		}
		dataStreams = dss
	}
	return dataStreams, nil
}

func (dbAPI *dbObjectModelAPI) populateAllOriginSelectors(ctx context.Context, dataStreams []model.DataStream) error {
	if len(dataStreams) == 0 {
		return nil
	}
	datastreamOriginSelectors := []DataStreamOriginSelectorDBO{}
	dataStreamIDs := funk.Map(dataStreams, func(dataStream model.DataStream) string { return dataStream.ID }).([]string)
	err := dbAPI.QueryIn(ctx, &datastreamOriginSelectors, queryMap["SelectDataStreamsOriginSelectors"], DataStreamIdsParam{
		DataStreamIDs: dataStreamIDs,
	})
	if err != nil {
		return err
	}
	dsOriginSelectorsMap := map[string]([]model.CategoryInfo){}
	for _, datastreamOriginSelector := range datastreamOriginSelectors {
		dsOriginSelectorsMap[datastreamOriginSelector.DataStreamID] = append(dsOriginSelectorsMap[datastreamOriginSelector.DataStreamID], datastreamOriginSelector.CategoryInfo)
	}
	for i := 0; i < len(dataStreams); i++ {
		dataStream := &dataStreams[i]
		dataStream.OriginSelectors = dsOriginSelectorsMap[dataStream.ID]
	}
	return nil
}

// internal API used by getDataStreamsWV2
func (dbAPI *dbObjectModelAPI) getDataStreamsByProjectsForQuery(context context.Context, projectIDs []string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataStream, int, error) {
	dataStreams := []model.DataStream{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dataStreams, 0, err
	}
	isEdgeReq, _ := base.IsEdgeRequest(authContext)
	tenantID := authContext.TenantID
	dataStreamDBOs := []DataStreamDBO{}
	query, err := buildLimitQuery(entityTypeDataStream, queryMap["SelectDataStreamsByProjectsTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return dataStreams, 0, err
	}
	queryParam := tenantIDParam2{TenantID: tenantID, ProjectIDs: projectIDs}
	if isEdgeReq {
		queryParam.State = string(model.DeployEntityState)
	}
	err = dbAPI.QueryIn(context, &dataStreamDBOs, query, queryParam)
	if err != nil {
		return dataStreams, 0, err
	}
	if len(dataStreamDBOs) == 0 {
		return dataStreams, 0, nil
	}
	totalCount := 0
	first := true
	for _, dataStreamDBO := range dataStreamDBOs {
		dataStream := model.DataStream{}
		if first {
			first = false
			if dataStreamDBO.TotalCount != nil {
				totalCount = *dataStreamDBO.TotalCount
			}
		}
		err := base.Convert(&dataStreamDBO, &dataStream)
		if err != nil {
			return []model.DataStream{}, 0, err
		}
		dataStream.GenerateEndPointURI()
		dataStreams = append(dataStreams, dataStream)
	}
	err = dbAPI.populateAllOriginSelectors(context, dataStreams)
	if err == nil {
		dataStreams, _ = dbAPI.populateOutDataIfc(context, dataStreams)
	}
	return dataStreams, totalCount, err
}

// internal api used by public non-W apis and by getDataStreamsWV2
func (dbAPI *dbObjectModelAPI) getDataStreams(context context.Context, projectID string, dataStreamID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataStream, error) {
	dataStreams := []model.DataStream{}
	dataStreamDBOs := []DataStreamDBO{}

	dbQueryParam, err := getDataStreamDBQueryParam(context, projectID, dataStreamID)
	if err != nil {
		return dataStreams, err
	}
	if dbQueryParam.Key == "" {
		if len(dataStreamID) == 0 {
			return dataStreams, nil
		}
		return dataStreams, errcode.NewRecordNotFoundError(dataStreamID)
	}

	query, err := buildQuery(entityTypeDataStream, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return dataStreams, err
	}
	err = dbAPI.QueryIn(context, &dataStreamDBOs, query, dbQueryParam.Param)
	if err != nil {
		return dataStreams, err
	}
	for _, dataStreamDBO := range dataStreamDBOs {
		dataStream := model.DataStream{}
		err = base.Convert(&dataStreamDBO, &dataStream)
		if err != nil {
			return dataStreams, err
		}
		dataStream.GenerateEndPointURI()
		dataStreams = append(dataStreams, dataStream)
	}
	err = dbAPI.populateAllOriginSelectors(context, dataStreams)
	if err == nil {
		dataStreams, _ = dbAPI.populateOutDataIfc(context, dataStreams)
	}
	return dataStreams, err
}

// internal api for old public W apis
func (dbAPI *dbObjectModelAPI) getDataStreamsW(context context.Context, projectID string, dataStreamID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	dataStreams, err := dbAPI.getDataStreams(context, projectID, dataStreamID, entitiesQueryParam)
	if err != nil {
		return err
	}
	if len(dataStreamID) == 0 {
		return base.DispatchPayload(w, dataStreams)
	}
	if len(dataStreams) == 0 {
		return errcode.NewRecordNotFoundError(dataStreamID)
	}
	return json.NewEncoder(w).Encode(dataStreams[0])
}

// internal api for new (paged) public W apis
func (dbAPI *dbObjectModelAPI) getDataStreamsWV2(context context.Context, projectID string, dataStreamID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getDataStreamDBQueryParam(context, projectID, dataStreamID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.DataStreamListPayload{DataStreamList: []model.DataStream{}})
	}
	projectIDs := dbQueryParam.Param.(DataStreamProjects).ProjectIDs

	queryParam := model.GetEntitiesQueryParam(req)

	dataStreams, totalCount, err := dbAPI.getDataStreamsByProjectsForQuery(context, projectIDs, queryParam)
	if err != nil {
		return err
	}
	for _, dataStream := range dataStreams {
		dataStream.GenerateEndPointURI()
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeDataStream}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.DataStreamListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		DataStreamList:            dataStreams,
	}
	return json.NewEncoder(w).Encode(r)
}

func (dbAPI *dbObjectModelAPI) getDataPipelineContainers(context context.Context, dataPipelineID string, edgeID string, callback func(context.Context, interface{}) (string, error)) (model.DataPipelineContainers, error) {
	dataPipelineContainers := model.DataPipelineContainers{}
	wsMessagePayload := model.DataPipelineContainersBaseObject{
		DataPipelineID: dataPipelineID,
		EdgeID:         edgeID,
	}
	wsResp, err := callback(context, wsMessagePayload)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error executing websocket callback: %s"), err.Error())
		return dataPipelineContainers, err
	}
	respString := ""
	if err := json.Unmarshal([]byte(wsResp), &respString); err != nil {
		glog.Errorf(base.PrefixRequestID(context, "json unmarshal error: %s"), err.Error())
		return dataPipelineContainers, err
	}
	dataPipelineContainers.DataPipelineContainersBaseObject = model.DataPipelineContainersBaseObject{
		DataPipelineID: dataPipelineID,
		EdgeID:         edgeID,
	}
	dataPipelineContainers.ContainerNames = []string{}
	if err := json.Unmarshal([]byte(respString), &dataPipelineContainers.ContainerNames); err != nil {
		glog.Errorf(base.PrefixRequestID(context, "json unmarshal error: %s"), err.Error())
	}
	return dataPipelineContainers, err
}

// SelectAllDataStreams select all data streams for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllDataStreams(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataStream, error) {
	return dbAPI.getDataStreams(context, "", "", entitiesQueryParam)
}

// SelectAllDataStreamsW select all data streams for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataStreamsW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getDataStreamsW(context, "", "", w, req)
}

// SelectAllDataStreamsWV2 select all data streams for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataStreamsWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getDataStreamsWV2(context, "", "", w, req)
}

// SelectAllDataStreamsForProject select all data streams for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllDataStreamsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataStream, error) {
	return dbAPI.getDataStreams(context, projectID, "", entitiesQueryParam)
}

// SelectAllDataStreamsForProjectW select all data streams for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataStreamsForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getDataStreamsW(context, projectID, "", w, req)
}

// SelectAllDataStreamsForProjectWV2 select all data streams for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDataStreamsForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getDataStreamsWV2(context, projectID, "", w, req)
}

// GetDataStream get a data stream object in the DB
func (dbAPI *dbObjectModelAPI) GetDataStream(context context.Context, id string) (model.DataStream, error) {
	if len(id) == 0 {
		return model.DataStream{}, errcode.NewBadRequestError(id)
	}
	dataStreams, err := dbAPI.getDataStreams(context, "", id, nil)
	if err != nil {
		return model.DataStream{}, err
	}
	dataStreams, _ = dbAPI.populateOutDataIfc(context, dataStreams)
	if len(dataStreams) == 0 {
		return model.DataStream{}, errcode.NewRecordNotFoundError(id)
	}
	dataStreams[0].GenerateEndPointURI()
	return dataStreams[0], nil
}

// GetDataStreamW get a data stream object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDataStreamW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError(id)
	}
	return dbAPI.getDataStreamsW(context, "", id, w, req)
}

// GetDataPipelineContainersW get the containers of a data pipeline object and writes output into writer
func (dbAPI *dbObjectModelAPI) GetDataPipelineContainersW(context context.Context, dataPipelineID string, edgeID string, w io.Writer, callback func(context.Context, interface{}) (string, error)) error {
	if dataPipelineID == "" {
		return errcode.NewBadRequestError("dataPipelineID")
	}
	if edgeID == "" {
		return errcode.NewBadRequestError("edgeID")
	}
	if _, err := dbAPI.GetDataStream(context, dataPipelineID); err != nil {
		return errcode.NewBadRequestError("dataPipelineID")
	}

	// Check for edge version for physical edges before proceeding.
	edge, err := dbAPI.GetEdge(context, edgeID)
	if err != nil {
		return errcode.NewInternalDatabaseError(err.Error())
	}
	if edge.Type == nil || *edge.Type != string(model.CloudTargetType) {
		// This is a physical edge so we need version check.
		edgeInfo, err := dbAPI.GetEdgeInfo(context, edgeID)
		if err != nil {
			return errcode.NewInternalDatabaseError(err.Error())
		}
		if edgeInfo.EdgeVersion == nil {
			// Use old version for upgrade as we need the data
			edgeInfo.EdgeVersion = nilVersion
		}
		feats, _ := GetFeaturesForVersion(*edgeInfo.EdgeVersion)
		if feats.RealTimeLogs != true {
			errMsg := "This feature is not supported on Edge Software Version v1.10 or below."
			return errcode.NewBadRequestExError("Edge version", errMsg)
		}
	}

	resp, err := dbAPI.getDataPipelineContainers(context, dataPipelineID, edgeID, callback)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)
}

func (dbAPI *dbObjectModelAPI) GetDataStreamNames(context context.Context, dsIDs []string) ([]string, error) {
	return dbAPI.getObjectNames(context, "data_stream_model", dsIDs, "dataPipelineID")
}

// CreateDataStream creates a data stream object in the DB
func (dbAPI *dbObjectModelAPI) CreateDataStream(context context.Context, i interface{} /* *model.DataStream */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DataStream)
	if !ok {
		return resp, errcode.NewInternalError("CreateDataStream: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateDataStream doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateDataStream doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateDataStream(&doc)
	if err != nil {
		return resp, err
	}
	if !ReK8sName.MatchString(doc.Name) {
		return resp, errcode.NewBadRequestError("name")
	}

	// set default project for backward compatibility
	if doc.ProjectID == "" {
		doc.ProjectID = GetDefaultProjectID(tenantID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataStream,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	err = validateDataStream(dbAPI, context, &doc)
	if err != nil {
		return resp, err
	}
	if doc.EnableSampling == true && doc.SamplingInterval <= 0 {
		glog.Errorf(base.PrefixRequestID(context, "CreateDataStream is invalid doc.EnableSampling: %v ,doc.SamplingInterval : %f\n"), doc.EnableSampling, doc.SamplingInterval)
		return resp, errcode.NewBadRequestExError("samplingInterval", fmt.Sprintf("Invalid sampling interval: %f", doc.SamplingInterval))
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	doc.SetEndPoint() //If endpoint not specified, set to defaults.

	dataStreamDBO := DataStreamDBO{}
	err = base.Convert(&doc, &dataStreamDBO)
	if err != nil {
		return resp, err
	}
	setDefaultFields(&dataStreamDBO)
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err = tx.NamedExec(context, queryMap["CreateDataStream"], &dataStreamDBO)
		// TODO(): We should handle idempotence here such that retries from client does not end up creating multiple
		// copies of the same data stream
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating datastream for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}

		if doc.Destination == model.DestinationDataInterface {
			glog.V(5).Infof(base.PrefixRequestID(context, "creating data ifc endpoint for %s"), doc.ID)
			var dataIfcEndpoint *model.DataIfcEndpoint
			if len(doc.DataIfcEndpoints) > 0 {
				dataIfcEndpoint = &doc.DataIfcEndpoints[0]
				// Backwards compatible: Previusly datastream.Endpoint was set to Name and Value both.
				dataIfcEndpoint.Value = dataIfcEndpoint.Name
			}

			err = dbAPI.claimDataIfcTopic(context, tx, dataIfcEndpoint, entityTypeDataStream, doc.ID, doc.TenantID)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "failed to claim topic %s for data stream %s. %s"), doc.EndPoint, doc.ID, err.Error())
				return err
			}
		}

		return dbAPI.createOriginSelectors(context, tx, doc.OriginSelectors, entityTypeDataStream, doc.ID)
	})
	if err != nil {
		return resp, err
	}
	doc.GenerateEndPointURI()
	datastreams := []model.DataStream{doc}
	datastreams, err = dbAPI.populateOutDataIfc(context, datastreams)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "failed to populate data ifc. %s"), err.Error())
	}
	if callback != nil {
		if len(datastreams) != 0 {
			doc = datastreams[0]
		}
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addDataPipelineAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateDataStreamW creates a data stream object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDataStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDataStream, &model.DataStream{}, w, r, callback)
}

// CreateDataStreamWV2 creates a data stream object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateDataStreamWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateDataStream), &model.DataStream{}, w, r, callback)
}

// UpdateDataStream updates a data stream object in the DB
func (dbAPI *dbObjectModelAPI) UpdateDataStream(context context.Context, i interface{} /* *model.DataStream */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DataStream)
	if !ok {
		return resp, errcode.NewInternalError("UpdateDataStream: type error")
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
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	doc.SetEndPoint() //If endpoint not specified, set to defaults.

	dataStreamDBO := DataStreamDBO{}

	err = model.ValidateDataStream(&doc)
	if err != nil {
		return resp, err
	}
	ds, err := dbAPI.GetDataStream(context, doc.ID)
	if err != nil {
		return resp, errcode.NewBadRequestError("dataStreamID")
	}
	if !ReK8sName.MatchString(doc.Name) {
		return resp, errcode.NewBadRequestError("name")
	}

	// set default project for backward compatibility
	if doc.ProjectID == "" {
		doc.ProjectID = GetDefaultProjectID(tenantID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataStream,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:    doc.ProjectID,
			OldProjectID: ds.ProjectID,
			ProjNameFn:   GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	err = validateDataStream(dbAPI, context, &doc)
	if err != nil {
		return resp, err
	}
	doc.GenerateEndPointURI()
	err = base.Convert(&doc, &dataStreamDBO)
	if err != nil {
		return resp, err
	}
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := base.DeleteTxn(context, tx, "data_stream_origin_model", map[string]interface{}{"data_stream_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting datastream for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		var claimedDataIfcEndpoint *model.DataIfcEndpoint
		if len(ds.DataIfcEndpoints) > 0 {
			claimedDataIfcEndpoint = &ds.DataIfcEndpoints[0]
		}
		err = dbAPI.unclaimDataIfcTopic(context, tx, claimedDataIfcEndpoint, entityTypeDataStream, doc.ID)

		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "failed to remove data source topic claim for data stream %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		_, err = tx.NamedExec(context, queryMap["UpdateDataStream"], &dataStreamDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in updating datastream for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createOriginSelectors(context, tx, doc.OriginSelectors, entityTypeDataStream, doc.ID)
		if err != nil {
			return err
		}

		if doc.Destination == model.DestinationDataInterface {
			glog.V(5).Infof(base.PrefixRequestID(context, "creating data ifc endpoint for %s"), doc.ID)
			var dataIfcEndpoint *model.DataIfcEndpoint
			if len(doc.DataIfcEndpoints) > 0 {
				dataIfcEndpoint = &doc.DataIfcEndpoints[0]
				// Backwards compatible: Previusly datastream.Endpoint was set to Name and Value both.
				dataIfcEndpoint.Name = dataIfcEndpoint.Value
			}
			err = dbAPI.claimDataIfcTopic(context, tx, dataIfcEndpoint, entityTypeDataStream, doc.ID, doc.TenantID)
		}
		return err
	})
	if err != nil {
		return resp, err
	}
	datastreams := []model.DataStream{doc}
	datastreams, err = dbAPI.populateOutDataIfc(context, datastreams)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "failed to populate out data ifc. %s"), err.Error())
	}
	if callback != nil {
		if len(datastreams) != 0 {
			doc = datastreams[0]
		}
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addDataPipelineAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateDataStreamW updates a data stream object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDataStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDataStream, &model.DataStream{}, w, r, callback)
}

// UpdateDataStreamWV2 updates a data stream object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateDataStreamWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateDataStream), &model.DataStream{}, w, r, callback)
}

// DeleteDataStream delete a data stream object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDataStream(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	// fetch data stream to get project id
	doc, err := dbAPI.GetDataStream(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDataStream,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})

	var ifcEndpoint *model.DataIfcEndpoint

	if len(doc.DataIfcEndpoints) > 0 {
		ifcEndpoint = &doc.DataIfcEndpoints[0]
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err := dbAPI.unclaimDataIfcTopic(context, tx, ifcEndpoint, entityTypeDataStream, doc.ID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in removing the data source topic claim for data stream %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		res, err := base.DeleteTxn(context, tx, "data_stream_model", map[string]interface{}{"id": id, "tenant_id": authContext.TenantID})
		if err != nil {
			return err
		}

		if base.IsDeleteSuccessful(res) {
			resp.ID = id
			if callback != nil {
				go callback(context, doc)
			}
		}
		return nil
	})
	if err == nil {
		GetAuditlogHandler().addDataPipelineAuditLog(dbAPI, context, doc, DELETE)
	}
	return resp, err
}

// DeleteDataStreamW delete a data stream object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDataStreamW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDataStream, id, w, callback)
}

// DeleteDataStreamWV2 delete a data stream object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteDataStreamWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteDataStream), id, w, callback)
}

// GetDataStreamIDs get IDs of all data streams using any of the script with id in scriptIDs
func (dbAPI *dbObjectModelAPI) GetDataStreamIDs(context context.Context, scriptIDs []string) ([]string, error) {
	dsIDs := []string{}
	dataStreams, err := dbAPI.SelectAllDataStreams(context, nil)
	if err != nil {
		return dsIDs, err
	}
	for _, dataStream := range dataStreams {
		for _, ta := range dataStream.TransformationArgsList {
			tid := ta.TransformationID
			if funk.Contains(scriptIDs, tid) {
				dsIDs = append(dsIDs, dataStream.ID)
				break
			}
		}
	}
	return dsIDs, nil
}

func (dbAPI *dbObjectModelAPI) getDataStreamsByIDs(ctx context.Context, dataStreamIDs []string) ([]model.DataStream, error) {
	dataStreams := []model.DataStream{}
	if len(dataStreamIDs) == 0 {
		return dataStreams, nil
	}

	dataStreamDBOs := []DataStreamDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &dataStreamDBOs, "data_stream_model", dataStreamIDs); err != nil {
		return nil, err
	}

	for _, dataStreamDBO := range dataStreamDBOs {
		dataStream := model.DataStream{}
		err := base.Convert(&dataStreamDBO, &dataStream)
		if err != nil {
			return []model.DataStream{}, err
		}
		dataStream.GenerateEndPointURI()
		dataStreams = append(dataStreams, dataStream)
	}
	err := dbAPI.populateAllOriginSelectors(ctx, dataStreams)
	if err == nil {
		dataStreams, _ = dbAPI.populateOutDataIfc(ctx, dataStreams)
	}
	return dataStreams, err
}

package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
)

const (
	entityTypeEdgeDeviceInfo = "edgeDeviceInfo"
	edgeDeviceInfoModelAlias = "im"
)

var (
	edgeDeviceInfoModelAliasMap = map[string]string{
		"edge_cluster_id": "dm",
	}
)

func init() {
	orderByHelper.Setup(entityTypeEdgeDeviceInfo, []string{"id", "version", "created_at", "updated_at", "edge_id", "num_cpu", "total_memory_kb", "total_storage_kb", "gpu_info", "cpu_usage", "memory_free_kb", "storage_free_kb", "gpu_usage", "edge_version", "edge_build_num", "kube_version", "os_version", "edge_cluster_id"})
}

// EdgeDeviceInfoDBO is DB object model for EdgeDeviceInfo
type EdgeDeviceInfoDBO struct {
	model.EdgeDeviceScopedModelDBO
	model.EdgeDeviceInfoCore
	Onboarded  bool              `json:"onboarded,omitempty" db:"onboarded"`
	HealthBits *json.RawMessage  `json:"healthBits,omitempty" db:"health_bits"`
	Artifacts  *json.RawMessage  `json:"artifacts,omitempty" db:"artifacts"`
	Type       *model.TargetType `json:"type" db:"type"`
}

// GetID returns the ID of the EdgeDeviceInfo object
// Impl for model.IdentifiableEntity
func (e EdgeDeviceInfoDBO) GetID() string {
	return e.ID
}

// GetClusterID returns the cluster ID
// Impl for model.ClusterEntity
func (e EdgeDeviceInfoDBO) GetClusterID() string {
	return e.ClusterID
}

func (dbAPI *dbObjectModelAPI) populateEdgeDeviceStatusFields(ctx context.Context, deviceInfo *model.EdgeDeviceInfo) {
	if deviceInfo == nil {
		return
	}
	if deviceInfo.DeviceID == deviceInfo.ClusterID {
		// Old edge
		deviceInfo.Connected = IsEdgeConnected(deviceInfo.TenantID, deviceInfo.ID)
		deviceInfo.Healthy = deviceInfo.Connected
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx, deviceInfo.ID)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Error in getting cert to set onboarding status for edge device %s. Error: %s"), deviceInfo.ID, err.Error())
			return
		}
		deviceInfo.Onboarded = edgeCert.Locked
	} else if deviceInfo.HealthBits != nil {
		deviceInfo.Healthy = isHealthy(ctx, deviceInfo.HealthBits)
		deviceInfo.Connected = (time.Since(deviceInfo.UpdatedAt) <= connectionStatusTimeout)
	}
}

func (dbAPI *dbObjectModelAPI) initEdgeDeviceInfo(ctx context.Context, tx *base.WrappedTx, deviceID string, now time.Time) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if now.IsZero() {
		now = base.RoundedNow()
	}
	deviceInfoDBO := EdgeDeviceInfoDBO{}
	deviceInfoDBO.ID = deviceID
	deviceInfoDBO.DeviceID = deviceID
	deviceInfoDBO.TenantID = authCtx.TenantID
	deviceInfoDBO.Version = float64(now.UnixNano())
	deviceInfoDBO.CreatedAt = now
	deviceInfoDBO.UpdatedAt = now
	_, err = tx.NamedExec(ctx, queryMap["CreateEdgeDeviceInfo"], &deviceInfoDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "createEdgeDeviceInfo: DB exec failed for %+v. Error: %s "), deviceInfoDBO, err.Error())
	}
	return err
}

func (dbAPI *dbObjectModelAPI) selectAllEdgeDevicesInfoDBOs(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]EdgeDeviceInfoDBO, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	deviceInfoDBOs := []EdgeDeviceInfoDBO{}
	tenantID := authCtx.TenantID
	param := model.BaseModelDBO{TenantID: tenantID}
	query, err := buildQueryWithTableAlias(entityTypeEdgeDeviceInfo, queryMap["SelectEdgeDevicesInfoTemplate"], entitiesQueryParam, orderByID, edgeDeviceInfoModelAlias, edgeDeviceInfoModelAliasMap)
	if err != nil {
		return deviceInfoDBOs, err
	}
	err = dbAPI.Query(ctx, &deviceInfoDBOs, query, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "selectAllEdgeDevicesInfo: DB query failed: %s\n"), err.Error())
		return deviceInfoDBOs, err
	}
	if !auth.IsInfraAdminRole(authCtx) {
		entities, err := dbAPI.filterEdgeDevices(ctx, deviceInfoDBOs)
		if err == nil {
			deviceInfoDBOs = entities.([]EdgeDeviceInfoDBO)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "selectAllEdgeDevicesInfo: filter edges failed: %s\n"), err.Error())
		}
	}
	return deviceInfoDBOs, err
}

// SelectAllEdgeDevicesInfo select all EdgeDeviceInfo for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfo(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeDeviceInfo, error) {
	deviceInfoDBOs, err := dbAPI.selectAllEdgeDevicesInfoDBOs(ctx, entitiesQueryParam)
	if err != nil {
		return []model.EdgeDeviceInfo{}, err
	}
	deviceInfos := make([]model.EdgeDeviceInfo, 0, len(deviceInfoDBOs))
	for _, deviceInfoDBO := range deviceInfoDBOs {
		deviceInfo := model.EdgeDeviceInfo{}
		err := base.Convert(&deviceInfoDBO, &deviceInfo)
		if err != nil {
			return deviceInfos, err
		}
		dbAPI.populateEdgeDeviceStatusFields(ctx, &deviceInfo)
		deviceInfos = append(deviceInfos, deviceInfo)
	}
	return deviceInfos, err
}

// SelectAllEdgesInfoW select all EdgeDeviceInfo nfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfoW(ctx context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	deviceInfos, err := dbAPI.SelectAllEdgeDevicesInfo(ctx, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, deviceInfos)
}

// getEdgeDevicesInfoV2 returns the list of EdgeDeviceInfo objects per page with the total count for the given tenant
func (dbAPI *dbObjectModelAPI) getEdgeDevicesInfoV2(ctx context.Context, projectID, clusterID string, req *http.Request) ([]model.EdgeDeviceInfo, int, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /edges, the target type is always edge for backward compatibility
	targetType := extractTargetTypeQueryParam(req)

	deviceIDs, deviceIDsInPage, err := dbAPI.getNodeInfoIDsInPage(ctx, projectID, clusterID, queryParam, targetType)
	if err != nil {
		return nil, 0, err
	}

	deviceInfos := []model.EdgeDeviceInfo{}
	if len(deviceIDsInPage) != 0 {
		// use in query to find edgeInfoDBOs
		query, err := buildQueryWithTableAlias(entityTypeEdgeDeviceInfo, queryMap["SelectEdgeDevicesInfoInEdgesTemplate"], queryParam, orderByID, edgeDeviceInfoModelAlias, edgeDeviceInfoModelAliasMap)
		if err != nil {
			return nil, 0, err
		}
		deviceInfoDBOs := []EdgeDeviceInfoDBO{}
		err = dbAPI.QueryIn(ctx, &deviceInfoDBOs, query, EdgeDeviceIDsParam{
			TenantID:  authCtx.TenantID,
			DeviceIDs: deviceIDsInPage,
		})
		if err != nil {
			return deviceInfos, 0, err
		}
		// convert edgeInfoDBO to edgeInfo
		for _, deviceInfoDBO := range deviceInfoDBOs {
			deviceInfo := model.EdgeDeviceInfo{}
			err := base.Convert(&deviceInfoDBO, &deviceInfo)
			if err != nil {
				return deviceInfos, 0, err
			}
			dbAPI.populateEdgeDeviceStatusFields(ctx, &deviceInfo)
			deviceInfos = append(deviceInfos, deviceInfo)
		}
	}
	return deviceInfos, len(deviceIDs), nil
}

// getEdgeDevicesInfoWV2 select all EdgeDeviceInfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) getEdgeDevicesInfoWV2(ctx context.Context, projectID, clusterID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	deviceInfos, totalCount, err := dbAPI.getEdgeDevicesInfoV2(ctx, projectID, clusterID, req)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeEdgeDeviceInfo})
	r := model.EdgeDeviceInfoListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		EdgeDeviceInfoList:        deviceInfos,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllEdgeDevicesInfoWV2 select all EdgeDeviceInfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfoWV2(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeDevicesInfoWV2(ctx, "", "", w, req)
}

// SelectAllEdgeDevicesInfoForProject select all edge usage info for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfoForProject(ctx context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeDeviceInfo, error) {
	deviceInfos := []model.EdgeDeviceInfo{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return deviceInfos, err
	}
	if !auth.IsProjectMember(projectID, authCtx) {
		return deviceInfos, errcode.NewPermissionDeniedError("RBAC")
	}
	// GetProject will properly fill in project.EdgeIDs
	project, err := dbAPI.GetProject(ctx, projectID)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to get projects with id %s, err=%s\n"), projectID, err.Error())
		return deviceInfos, err
	}
	if len(project.EdgeIDs) == 0 {
		return deviceInfos, nil
	}
	param := EdgeClusterIDsParam{
		TenantID:   authCtx.TenantID,
		ClusterIDs: project.EdgeIDs,
	}
	query, err := buildQueryWithTableAlias(entityTypeEdgeDeviceInfo, queryMap["SelectEdgeDevicesInfoInClustersTemplate"], entitiesQueryParam, orderByID, edgeDeviceInfoModelAlias, edgeDeviceInfoModelAliasMap)
	if err != nil {
		return deviceInfos, err
	}
	err = dbAPI.QueryInWithCallback(ctx, func(dbObjPtr interface{}) error {
		deviceInfo := model.EdgeDeviceInfo{}
		err := base.Convert(dbObjPtr, &deviceInfo)
		if err != nil {
			return err
		}
		dbAPI.populateEdgeDeviceStatusFields(ctx, &deviceInfo)
		deviceInfos = append(deviceInfos, deviceInfo)
		return nil
	}, query, EdgeDeviceInfoDBO{}, param)
	return deviceInfos, err
}

// SelectAllEdgeDevicesInfoForProjectW select all edges info for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfoForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	deviceInfos, err := dbAPI.SelectAllEdgeDevicesInfoForProject(ctx, projectID, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, deviceInfos)
}

// SelectAllEdgeDevicesInfoForProjectWV2 select all edges info for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfoForProjectWV2(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeDevicesInfoWV2(ctx, projectID, "", w, req)
}

// GetEdgeDeviceInfo get an EdgeDeviceInfo object in the DB
func (dbAPI *dbObjectModelAPI) GetEdgeDeviceInfo(ctx context.Context, id string) (model.EdgeDeviceInfo, error) {
	deviceInfo := model.EdgeDeviceInfo{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return deviceInfo, err
	}
	tenantID := authCtx.TenantID
	deviceInfoDBOs := []EdgeDeviceInfoDBO{}
	param := model.BaseModelDBO{TenantID: tenantID, ID: id}
	if id == "" {
		glog.Error(base.PrefixRequestID(ctx, "GetEdgeDeviceInfo: invalid edge ID"))
		return deviceInfo, errcode.NewBadRequestError("edgeId")
	}
	err = dbAPI.Query(ctx, &deviceInfoDBOs, queryMap["SelectEdgeDevicesInfo"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetEdgeDeviceInfo: DB select failed for edge device %s. Error: %s\n"), id, err.Error())
		return deviceInfo, err
	}
	if !auth.IsInfraAdminRole(authCtx) {
		entities, err := dbAPI.filterEdgeDevices(ctx, deviceInfoDBOs)
		if err == nil {
			deviceInfoDBOs = entities.([]EdgeDeviceInfoDBO)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "GetEdgeDeviceInfo: filter edges failed for edge device %s: %s\n"), id, err.Error())
		}
	}
	if len(deviceInfoDBOs) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "GetEdgeDeviceInfo: record not found for edge device id %s"), id)
		return deviceInfo, errcode.NewRecordNotFoundError(id)
	}
	err = base.Convert(&deviceInfoDBOs[0], &deviceInfo)
	if err == nil {
		dbAPI.populateEdgeDeviceStatusFields(ctx, &deviceInfo)
	}
	return deviceInfo, err
}

// GetEdgeDeviceInfoW get an EdgeDeviceInfo object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeDeviceInfoW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	deviceInfo, err := dbAPI.GetEdgeDeviceInfo(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, deviceInfo)
}

// CreateEdgeDeviceInfo creates an edgeInfo object in the DB
// Note: POST /edgesinfo is not there, this method is not used. Only PUT /edges/{id}/info is exposed
// EdgeInfo is initially created when edge is created, see CreateEdge in edgeApi.go
func (dbAPI *dbObjectModelAPI) CreateEdgeDeviceInfo(ctx context.Context, i interface{} /* *model.EdgeDeviceInfo */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.EdgeDeviceInfo)
	if !ok {
		return resp, errcode.NewInternalError("CreateEdgeDeviceInfo: type error")
	}
	doc := *p
	tenantID := authCtx.TenantID
	doc.TenantID = tenantID
	doc.ID = doc.DeviceID

	now := base.RoundedNow()

	epochInNanoSecs := now.Nanosecond()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	edgeDeviceInfoDBO := EdgeDeviceInfoDBO{}
	err = base.Convert(&doc, &edgeDeviceInfoDBO)
	if err != nil {
		return resp, err
	}
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		deviceInfos := []EdgeDeviceInfoDBO{}
		err = base.QueryTxn(ctx, tx, &deviceInfos, queryMap["SelectEdgeDevicesInfo"], model.BaseModelDBO{TenantID: doc.TenantID, ID: doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in querying for existing edge device info for device %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		var existingArtifacts *json.RawMessage
		if len(deviceInfos) > 0 {
			existingArtifacts = deviceInfos[0].Artifacts
			// Cluster ID comes from another table
			edgeDeviceInfoDBO.ClusterID = deviceInfos[0].ClusterID
		}
		if (doc.Artifacts == nil || edgeDeviceInfoDBO.Artifacts == nil) && existingArtifacts != nil {
			edgeDeviceInfoDBO.Artifacts = existingArtifacts
		}
		_, err = tx.NamedExec(ctx, queryMap["CreateEdgeDeviceInfo"], &edgeDeviceInfoDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "CreateEdgeDeviceInfo: DB exec failed for %+v. Error: %s "), doc, err.Error())
			return err
		}
		if len(deviceInfos) == 0 {
			// Fetch to get cluster ID.
			// In normal operation, this block must not be hit because the entry is always inserted when the device is created
			err = base.QueryTxn(ctx, tx, &deviceInfos, queryMap["SelectEdgeDevicesInfo"], model.BaseModelDBO{TenantID: doc.TenantID, ID: doc.ID})
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in querying for just inserted edge device info for device %s. Error: %s"), doc.ID, err.Error())
				return err
			}
			if len(deviceInfos) == 0 {
				errMsg := fmt.Sprintf("Unexpected error: edge device info is expected for device %s", doc.ID)
				glog.Errorf(base.PrefixRequestID(ctx, errMsg))
				return errcode.NewInternalError(errMsg)
			}
			edgeDeviceInfoDBO.ClusterID = deviceInfos[0].ClusterID
		}
		err = base.Convert(&edgeDeviceInfoDBO, &doc)
		if err == nil {
			dbAPI.populateEdgeDeviceStatusFields(ctx, &doc)
			deviceInfo := &doc
			event := &model.NodeInfoEvent{
				ID:   base.GetUUID(),
				Info: deviceInfo.ToNodeInfo(),
			}
			err = base.Publisher.Publish(ctx, event)
		}
		return err
	})
	if err == nil {
		resp.ID = doc.ID
	}
	return resp, err
}

// CreateEdgeDeviceInfoW creates an EdgeDeviceInfo object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateEdgeDeviceInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateEdgeDeviceInfo, &model.EdgeDeviceInfo{}, w, r, callback)
}

// CreateEdgeDeviceInfoWV2 creates an EdgeDeviceInfo object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateEdgeDeviceInfoWV2(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, model.ToCreateV2(dbAPI.CreateEdgeDeviceInfo), &model.EdgeDeviceInfo{}, w, r, callback)
}

// DeleteEdgeDeviceInfo deletes an EdgeDeviceInfo object with the ID in the DB
func (dbAPI *dbObjectModelAPI) DeleteEdgeDeviceInfo(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return model.DeleteDocumentResponse{}, err
	}
	doc := model.BaseModelDBO{
		TenantID: authCtx.TenantID,
		ID:       id,
	}
	return DeleteEntity(ctx, dbAPI, "edge_device_info_model", "id", id, doc, callback)
}

// DeleteEdgeDeviceInfoW deletes an EdgeDeviceInfo object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteEdgeDeviceInfoW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, dbAPI.DeleteEdgeDeviceInfo, id, w, callback)
}

// DeleteEdgeDeviceInfoWV2 deletes an EdgeDeviceInfo info object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteEdgeDeviceInfoWV2(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteEdgeDeviceInfo), id, w, callback)
}

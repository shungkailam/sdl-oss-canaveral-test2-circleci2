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

type multiNodeState int

const (
	multiNodeUnknown multiNodeState = iota
	multiNodeAware
	multiNodeUnAware
)

const (
	entityTypeNodeInfo      = "nodeInfo"
	nodeInfoModelAlias      = "im"
	connectionStatusTimeout = time.Minute * 10
	// failOnMissingExpectedHealthBits indicates whether to fail on missing any expected health bits
	failOnMissingExpectedHealthBits = false
)

var (
	nodeInfoModelAliasMap = map[string]string{
		"edge_cluster_id": "dm",
	}
	// expectedHealthBits are the health bits expected from a healthy node
	expectedHealthBits = map[string]bool{
		"DiskPressure":       false,
		"MemoryPressure":     false,
		"NetworkUnavailable": false,
		"PIDPressure":        false,
		"Ready":              true,
	}
)

func init() {
	queryMap["SelectEdgeDevicesInfo"] = `SELECT im.*, dm.is_onboarded as "onboarded", dm.edge_cluster_id as "edge_cluster_id", cm.type as "type" FROM edge_device_info_model im, edge_device_model dm, edge_cluster_model cm WHERE im.tenant_id = dm.tenant_id AND im.device_id = dm.id AND im.tenant_id = cm.tenant_id AND dm.edge_cluster_id = cm.id AND im.tenant_id = :tenant_id AND (:id = '' OR im.id = :id)`
	queryMap["SelectEdgeDevicesInfoTemplate"] = `SELECT im.*, dm.is_onboarded as "onboarded", dm.edge_cluster_id as "edge_cluster_id", cm.type as "type" FROM edge_device_info_model im, edge_device_model dm, edge_cluster_model cm WHERE im.tenant_id = dm.tenant_id AND im.device_id = dm.id AND im.tenant_id = cm.tenant_id AND dm.edge_cluster_id = cm.id AND im.tenant_id = :tenant_id AND (:id = '' OR im.id = :id) %s`
	queryMap["SelectEdgeDevicesInfoInEdgesTemplate"] = `SELECT im.*, dm.is_onboarded as "onboarded", dm.edge_cluster_id as "edge_cluster_id", cm.type as "type" FROM edge_device_info_model im, edge_device_model dm, edge_cluster_model cm WHERE im.tenant_id = dm.tenant_id AND im.device_id = dm.id AND im.tenant_id = cm.tenant_id AND dm.edge_cluster_id = cm.id AND im.tenant_id = :tenant_id AND im.id IN (:edge_device_ids) %s`
	queryMap["SelectEdgeDevicesInfoInClustersTemplate"] = `SELECT im.*, dm.is_onboarded as "onboarded", dm.edge_cluster_id as "edge_cluster_id", cm.type as "type" FROM edge_device_info_model im, edge_device_model dm, edge_cluster_model cm WHERE im.tenant_id = dm.tenant_id AND im.device_id = dm.id AND im.tenant_id = cm.tenant_id AND dm.edge_cluster_id = cm.id AND im.tenant_id = :tenant_id AND dm.edge_cluster_id IN (:edge_cluster_ids) %s`
	queryMap["SelectEdgeDeviceInfoIDsByTypeTemplate"] = `SELECT dm.id FROM edge_device_info_model im, edge_device_model dm WHERE im.tenant_id = dm.tenant_id AND im.device_id = dm.id AND im.tenant_id = :tenant_id AND (:edge_cluster_id = '' OR dm.edge_cluster_id = :edge_cluster_id) %s`
	queryMap["CreateEdgeDeviceInfo"] = `INSERT INTO edge_device_info_model (id, version, tenant_id, device_id, num_cpu, total_memory_kb, total_storage_kb, gpu_info, cpu_usage, memory_free_kb, storage_free_kb, gpu_usage, edge_version, edge_build_num, kube_version, os_version, created_at, updated_at, artifacts, health_bits) VALUES (:id, :version, :tenant_id, :device_id, :num_cpu, :total_memory_kb, :total_storage_kb, :gpu_info, :cpu_usage, :memory_free_kb, :storage_free_kb, :gpu_usage, :edge_version, :edge_build_num, :kube_version, :os_version, :created_at, :updated_at, :artifacts, :health_bits)
	 ON CONFLICT (tenant_id, device_id) DO UPDATE SET version = :version, num_cpu = :num_cpu, total_memory_kb = :total_memory_kb, total_storage_kb = :total_storage_kb, gpu_info = :gpu_info, cpu_usage = :cpu_usage, memory_free_kb = :memory_free_kb, storage_free_kb = :storage_free_kb, gpu_usage = :gpu_usage, edge_version = :edge_version, edge_build_num = :edge_build_num, kube_version = :kube_version, os_version = :os_version, updated_at = :updated_at, artifacts = :artifacts, health_bits = :health_bits WHERE edge_device_info_model.tenant_id = :tenant_id AND edge_device_info_model.id = :id`

	// Used by onboard API which does not pass tenant ID
	queryMap["SetNodeVersion"] = `UPDATE edge_device_info_model SET version = :version, edge_version = :edge_version, updated_at = :updated_at WHERE (:tenant_id = '' OR tenant_id = :tenant_id) AND id = :id`

	orderByHelper.Setup(entityTypeNodeInfo, []string{"id", "version", "created_at", "updated_at", "node_id:device_id", "num_cpu", "total_memory_kb", "total_storage_kb", "gpu_info", "cpu_usage", "memory_free_kb", "storage_free_kb", "gpu_usage", "node_version:edge_version", "node_build_num:edge_build_num", "kube_version", "os_version", "svc_domain_id:edge_cluster_id"})
}

// NodeInfoDBO is DB object model for NodeInfo
type NodeInfoDBO struct {
	model.NodeEntityModelDBO
	model.NodeInfoCore
	Onboarded  bool              `json:"onboarded,omitempty" db:"onboarded"`
	HealthBits *json.RawMessage  `json:"healthBits,omitempty" db:"health_bits"`
	Artifacts  *json.RawMessage  `json:"artifacts,omitempty" db:"artifacts"`
	Type       *model.TargetType `json:"type" db:"type"`
}

func getMultiNodeState(nodeInfo *model.NodeInfo) multiNodeState {
	if nodeInfo == nil {
		return multiNodeUnknown
	}
	if nodeInfo.NodeVersion == nil {
		return multiNodeUnknown
	}
	nodeVersion := *nodeInfo.NodeVersion
	fts, err := GetFeaturesForVersion(nodeVersion)
	if err != nil {
		glog.Errorf("Error in getting features for version %s. Error: %s", nodeVersion, err.Error())
		return multiNodeUnknown
	}
	if fts.MultiNodeAware {
		return multiNodeAware
	}
	return multiNodeUnAware
}

// isHealthy checks if the input health bits match the expected good health bits
func isHealthy(ctx context.Context, healthBits map[string]bool) bool {
	healthy := len(healthBits) > 0
	for key, val := range expectedHealthBits {
		v, ok := healthBits[key]
		if !ok {
			if failOnMissingExpectedHealthBits {
				healthy = false
				break
			}
		} else if v != val {
			healthy = false
			break
		}
	}
	return healthy
}

// setHealthStatus sets the health related fields
func setHealthStatus(ctx context.Context, nodeInfo *model.NodeInfo) {
	if nodeInfo == nil {
		return
	}
	if len(nodeInfo.HealthBits) == 0 {
		nodeInfo.HealthStatus = model.NodeHealthStatusUnknown
		nodeInfo.Healthy = false
	} else {
		nodeInfo.HealthStatus = model.NodeHealthStatusUnhealthy
		nodeInfo.Healthy = isHealthy(ctx, nodeInfo.HealthBits)
		if nodeInfo.Healthy {
			nodeInfo.HealthStatus = model.NodeHealthStatusHealthy
		}
	}
}

func (dbAPI *dbObjectModelAPI) populateNodeStatusFields(ctx context.Context, nodeInfo *model.NodeInfo, isCloudTargetType bool) {
	if nodeInfo == nil {
		return
	}
	switch getMultiNodeState(nodeInfo) {
	case multiNodeAware:
		setHealthStatus(ctx, nodeInfo)
		nodeInfo.Connected = (time.Since(nodeInfo.UpdatedAt) <= connectionStatusTimeout)
	case multiNodeUnAware:
		dbAPI.populateNodeStatusFieldsOld(ctx, nodeInfo)
	case multiNodeUnknown:
		if isCloudTargetType {
			// U2 edges have nil NodeVersion
			dbAPI.populateNodeStatusFieldsOld(ctx, nodeInfo)
		} else {
			nodeInfo.Connected = false
			nodeInfo.Healthy = false
			nodeInfo.Onboarded = false
		}
	}
}
func (dbAPI *dbObjectModelAPI) populateNodeStatusFieldsOld(ctx context.Context, nodeInfo *model.NodeInfo) {
	// Old edge/node
	nodeInfo.Connected = IsEdgeConnected(nodeInfo.TenantID, nodeInfo.ID)
	nodeInfo.Healthy = nodeInfo.Connected
	edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx, nodeInfo.ID)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Error in getting cert to set onboarding status for node %s. Error: %s"), nodeInfo.ID, err.Error())
		return
	}
	nodeInfo.Onboarded = edgeCert.Locked
}

func (dbAPI *dbObjectModelAPI) initNodeInfo(ctx context.Context, tx *base.WrappedTx, nodeID string, nodeVersion *string, now time.Time) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if now.IsZero() {
		now = base.RoundedNow()
	}
	nodeInfoDBO := NodeInfoDBO{}
	nodeInfoDBO.ID = nodeID
	nodeInfoDBO.NodeID = nodeID
	nodeInfoDBO.TenantID = authCtx.TenantID
	nodeInfoDBO.NodeVersion = nodeVersion
	nodeInfoDBO.Version = float64(now.UnixNano())
	nodeInfoDBO.CreatedAt = now
	nodeInfoDBO.UpdatedAt = now
	_, err = tx.NamedExec(ctx, queryMap["CreateEdgeDeviceInfo"], &nodeInfoDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "initNodeInfo: DB exec failed for %+v. Error: %s "), nodeInfoDBO, err.Error())
	}
	return err
}

func (dbAPI *dbObjectModelAPI) setNodeVersion(ctx context.Context, tx *base.WrappedTx, tenantID, nodeID string, nodeVersion string, now time.Time) error {
	if now.IsZero() {
		now = base.RoundedNow()
	}
	nodeInfoDBO := NodeInfoDBO{}
	nodeInfoDBO.ID = nodeID
	nodeInfoDBO.TenantID = tenantID
	nodeInfoDBO.NodeVersion = &nodeVersion
	nodeInfoDBO.Version = float64(now.UnixNano())
	nodeInfoDBO.UpdatedAt = now
	_, err := tx.NamedExec(ctx, queryMap["SetNodeVersion"], &nodeInfoDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "setNodeVersion: DB exec failed for %+v. Error: %s "), nodeInfoDBO, err.Error())
	}
	return err
}

func (dbAPI *dbObjectModelAPI) getNodeInfoIDsInPage(ctx context.Context, projectID string, svcDomainID string, queryParam *model.EntitiesQueryParam, targetType model.TargetType) ([]string, []string, error) {
	return dbAPI.GetEntityIDsInPage(ctx, projectID, svcDomainID, queryParam, func(ctx context.Context, svcDomainEntity *model.ServiceDomainEntityModelDBO, queryParam *model.EntitiesQueryParam) ([]string, error) {
		query, err := buildQueryWithTableAlias(entityTypeNodeInfo, queryMap["SelectEdgeDeviceInfoIDsByTypeTemplate"], queryParam, orderByNameID, nodeInfoModelAlias, nodeInfoModelAliasMap)
		if err != nil {
			return []string{}, err
		}
		svcDomainTypeParam := ServiceDomainTypeParam{TenantID: svcDomainEntity.TenantID, SvcDomainID: svcDomainEntity.SvcDomainID, Type: targetType}
		return dbAPI.selectEntityIDsByParam(ctx, query, svcDomainTypeParam)
	})
}

func (dbAPI *dbObjectModelAPI) selectAllNodesInfoDBOs(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]NodeInfoDBO, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	nodeInfoDBOs := []NodeInfoDBO{}
	tenantID := authCtx.TenantID
	param := model.BaseModelDBO{TenantID: tenantID}
	query, err := buildQueryWithTableAlias(entityTypeNodeInfo, queryMap["SelectEdgeDevicesInfoTemplate"], entitiesQueryParam, orderByID, nodeInfoModelAlias, nodeInfoModelAliasMap)
	if err != nil {
		return nodeInfoDBOs, err
	}
	err = dbAPI.Query(ctx, &nodeInfoDBOs, query, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "selectAllNodesInfoDBOs: DB query failed: %s\n"), err.Error())
		return nodeInfoDBOs, err
	}
	if !auth.IsInfraAdminRole(authCtx) {
		entities, err := dbAPI.filterNodes(ctx, nodeInfoDBOs)
		if err == nil {
			nodeInfoDBOs = entities.([]NodeInfoDBO)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "selectAllNodesInfoDBOs: filter nodes failed: %s\n"), err.Error())
		}
	}
	return nodeInfoDBOs, err
}

// SelectAllNodesInfo selects all node infos for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllNodesInfo(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.NodeInfo, error) {
	nodeInfoDBOs, err := dbAPI.selectAllNodesInfoDBOs(ctx, entitiesQueryParam)
	if err != nil {
		return []model.NodeInfo{}, err
	}
	nodeInfos := make([]model.NodeInfo, 0, len(nodeInfoDBOs))
	for _, nodeInfoDBO := range nodeInfoDBOs {
		nodeInfo := model.NodeInfo{}
		err := base.Convert(&nodeInfoDBO, &nodeInfo)
		if err != nil {
			return nodeInfos, err
		}
		dbAPI.populateNodeStatusFields(ctx, &nodeInfo, nodeInfoDBO.Type != nil && *nodeInfoDBO.Type == model.CloudTargetType)
		nodeInfos = append(nodeInfos, nodeInfo)
	}
	return nodeInfos, err
}

// SelectAllNodesInfoW select all nodes info for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesInfoW(ctx context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	nodeInfos, err := dbAPI.SelectAllNodesInfo(ctx, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, nodeInfos)
}

// getNodesInfoV2 returns the list of node info objects per page with the total count for the given tenant
func (dbAPI *dbObjectModelAPI) getNodesInfoV2(ctx context.Context, projectID, svcDomainID string, req *http.Request) ([]model.NodeInfo, int, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, 0, err
	}
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /edges or /nodes, the target type is always edge for backward compatibility
	targetType := extractTargetTypeQueryParam(req)

	nodeIDs, nodeIDsInPage, err := dbAPI.getNodeInfoIDsInPage(ctx, projectID, svcDomainID, queryParam, targetType)
	if err != nil {
		return nil, 0, err
	}

	nodeInfos := []model.NodeInfo{}
	if len(nodeIDsInPage) != 0 {
		// use in query to find nodeInfoDBOs
		query, err := buildQueryWithTableAlias(entityTypeNodeInfo, queryMap["SelectEdgeDevicesInfoInEdgesTemplate"], queryParam, orderByID, nodeInfoModelAlias, nodeInfoModelAliasMap)
		if err != nil {
			return nil, 0, err
		}
		nodeInfoDBOs := []NodeInfoDBO{}
		err = dbAPI.QueryIn(ctx, &nodeInfoDBOs, query, NodeIDsParam{
			TenantID: authCtx.TenantID,
			NodeIDs:  nodeIDsInPage,
		})
		if err != nil {
			return nodeInfos, 0, err
		}
		// convert nodeInfoDBO to nodeInfo
		for _, nodeInfoDBO := range nodeInfoDBOs {
			nodeInfo := model.NodeInfo{}
			err := base.Convert(&nodeInfoDBO, &nodeInfo)
			if err != nil {
				return nodeInfos, 0, err
			}
			dbAPI.populateNodeStatusFields(ctx, &nodeInfo, nodeInfoDBO.Type != nil && *nodeInfoDBO.Type == model.CloudTargetType)
			nodeInfos = append(nodeInfos, nodeInfo)
		}
	}
	return nodeInfos, len(nodeIDs), nil
}

// getNodesInfoWV2 selects all NodeInfo for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) getNodesInfoWV2(ctx context.Context, projectID, svcDomainID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	nodeInfos, totalCount, err := dbAPI.getNodesInfoV2(ctx, projectID, svcDomainID, req)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeNodeInfo})
	r := model.NodeInfoListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		NodeInfoList:              nodeInfos,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllNodesInfoWV2 selects all nodes info for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesInfoWV2(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getNodesInfoWV2(ctx, "", "", w, req)
}

// SelectAllNodesInfoForProject selects all nodes info for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllNodesInfoForProject(ctx context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.NodeInfo, error) {
	nodeInfos := []model.NodeInfo{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nodeInfos, err
	}
	if !auth.IsProjectMember(projectID, authCtx) {
		return nodeInfos, errcode.NewPermissionDeniedError("RBAC")
	}
	// GetProject will properly fill in project.EdgeIDs
	project, err := dbAPI.GetProject(ctx, projectID)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to get projects with id %s, err=%s\n"), projectID, err.Error())
		return nodeInfos, err
	}
	if len(project.EdgeIDs) == 0 {
		return nodeInfos, nil
	}
	param := ServiceDomainIDsParam{
		TenantID:     authCtx.TenantID,
		SvcDomainIDs: project.EdgeIDs,
	}
	query, err := buildQueryWithTableAlias(entityTypeNodeInfo, queryMap["SelectEdgeDevicesInfoInClustersTemplate"], entitiesQueryParam, orderByID, nodeInfoModelAlias, nodeInfoModelAliasMap)
	if err != nil {
		return nodeInfos, err
	}
	err = dbAPI.QueryInWithCallback(ctx, func(dbObjPtr interface{}) error {
		nodeInfoDBO := NodeInfoDBO{}
		err := base.Convert(dbObjPtr, &nodeInfoDBO)
		if err != nil {
			return err
		}
		nodeInfo := model.NodeInfo{}
		err = base.Convert(&nodeInfoDBO, &nodeInfo)
		if err != nil {
			return err
		}
		dbAPI.populateNodeStatusFields(ctx, &nodeInfo, nodeInfoDBO.Type != nil && *nodeInfoDBO.Type == model.CloudTargetType)
		nodeInfos = append(nodeInfos, nodeInfo)
		return nil
	}, query, NodeInfoDBO{}, param)
	return nodeInfos, err
}

// SelectAllNodesInfoForProjectW select all nodes info for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesInfoForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	nodeInfos, err := dbAPI.SelectAllNodesInfoForProject(ctx, projectID, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, nodeInfos)
}

// SelectAllNodesInfoForProjectWV2 select all nodes info for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesInfoForProjectWV2(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getNodesInfoWV2(ctx, projectID, "", w, req)
}

// GetNodeInfo get an node info from the DB
func (dbAPI *dbObjectModelAPI) GetNodeInfo(ctx context.Context, id string) (model.NodeInfo, error) {
	nodeInfo := model.NodeInfo{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nodeInfo, err
	}
	tenantID := authCtx.TenantID
	nodeInfoDBOs := []NodeInfoDBO{}
	param := model.BaseModelDBO{TenantID: tenantID, ID: id}
	if id == "" {
		glog.Error(base.PrefixRequestID(ctx, "GetNodeInfo: invalid node ID"))
		return nodeInfo, errcode.NewBadRequestError("nodeId")
	}
	err = dbAPI.Query(ctx, &nodeInfoDBOs, queryMap["SelectEdgeDevicesInfo"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetNodeInfo: DB select failed for the node %s. Error: %s\n"), id, err.Error())
		return nodeInfo, err
	}
	if !auth.IsInfraAdminRole(authCtx) {
		entities, err := dbAPI.filterNodes(ctx, nodeInfoDBOs)
		if err == nil {
			nodeInfoDBOs = entities.([]NodeInfoDBO)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "GetNodeInfo: filter nodes failed for the node %s: %s\n"), id, err.Error())
		}
	}
	if len(nodeInfoDBOs) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "GetNodeInfo: record not found for node %s"), id)
		return nodeInfo, errcode.NewRecordNotFoundError(id)
	}
	err = base.Convert(&nodeInfoDBOs[0], &nodeInfo)
	if err == nil {
		dbAPI.populateNodeStatusFields(ctx, &nodeInfo, nodeInfoDBOs[0].Type != nil && *nodeInfoDBOs[0].Type == model.CloudTargetType)
	}
	return nodeInfo, err
}

// GetNodeInfoW gets node info from the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetNodeInfoW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	nodeInfo, err := dbAPI.GetNodeInfo(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, nodeInfo)
}

// CreateNodeInfo creates a node info object in the DB
// Note: POST /nodesinfo is not there, this method is not used. Only PUT /nodes/{id}/info is exposed
// NodeInfo is initially created when a node is created, see CreateNode in nodeApi.go
func (dbAPI *dbObjectModelAPI) CreateNodeInfo(ctx context.Context, i interface{} /* *model.NodeInfo */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.NodeInfo)
	if !ok {
		return resp, errcode.NewInternalError("CreateNodeInfo: type error")
	}
	doc := *p
	tenantID := authCtx.TenantID
	doc.TenantID = tenantID
	doc.ID = doc.NodeID

	now := base.RoundedNow()

	epochInNanoSecs := now.Nanosecond()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	nodeInfoDBO := NodeInfoDBO{}
	err = base.Convert(&doc, &nodeInfoDBO)
	if err != nil {
		return resp, err
	}
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		nodeInfos := []NodeInfoDBO{}
		err = base.QueryTxn(ctx, tx, &nodeInfos, queryMap["SelectEdgeDevicesInfo"], model.BaseModelDBO{TenantID: doc.TenantID, ID: doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in querying for existing node info for the node %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		var existingArtifacts *json.RawMessage
		if len(nodeInfos) > 0 {
			existingArtifacts = nodeInfos[0].Artifacts
			// Svc domain ID comes from another table
			nodeInfoDBO.SvcDomainID = nodeInfos[0].SvcDomainID
		}
		if (doc.Artifacts == nil || nodeInfoDBO.Artifacts == nil) && existingArtifacts != nil {
			nodeInfoDBO.Artifacts = existingArtifacts
		}
		_, err = tx.NamedExec(ctx, queryMap["CreateEdgeDeviceInfo"], &nodeInfoDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "CreateNodeInfo: DB exec failed for %+v. Error: %s "), doc, err.Error())
			return err
		}
		if len(nodeInfos) == 0 {
			// Fetch to get cluster ID.
			// In normal operation, this block must not be hit because the entry is always inserted when the device is created
			err = base.QueryTxn(ctx, tx, &nodeInfos, queryMap["SelectEdgeDevicesInfo"], model.BaseModelDBO{TenantID: doc.TenantID, ID: doc.ID})
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in querying for just inserted node info for node %s. Error: %s"), doc.ID, err.Error())
				return err
			}
			if len(nodeInfos) == 0 {
				errMsg := fmt.Sprintf("Unexpected error: node info is expected for node %s", doc.ID)
				glog.Errorf(base.PrefixRequestID(ctx, errMsg))
				return errcode.NewInternalError(errMsg)
			}
			nodeInfoDBO.SvcDomainID = nodeInfos[0].SvcDomainID
		}
		err = base.Convert(&nodeInfoDBO, &doc)
		if err == nil {
			dbAPI.populateNodeStatusFields(ctx, &doc, nodeInfoDBO.Type != nil && *nodeInfoDBO.Type == model.CloudTargetType)
			event := &model.NodeInfoEvent{
				ID:   base.GetUUID(),
				Info: &doc,
			}
			err = base.Publisher.Publish(ctx, event)
		}
		return err
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	if callback != nil {
		callback(ctx, resp)
	}
	return resp, err
}

// CreateNodeInfoW creates an node info object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateNodeInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateNodeInfo, &model.NodeInfo{}, w, r, callback)
}

// CreateNodeInfoWV2 creates a node info object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateNodeInfoWV2(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, model.ToCreateV2(dbAPI.CreateNodeInfo), &model.NodeInfo{}, w, r, callback)
}

// DeleteNodeInfo deletes an node info object with the ID in the DB
func (dbAPI *dbObjectModelAPI) DeleteNodeInfo(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
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

// DeleteNodeInfoW deletes a node info object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteNodeInfoW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, dbAPI.DeleteNodeInfo, id, w, callback)
}

// DeleteNodeInfoWV2 deletes an node info object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteNodeInfoWV2(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteNodeInfo), id, w, callback)
}

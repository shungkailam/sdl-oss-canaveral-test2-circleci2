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
)

const (
	// entityTypeNode is the entity type `node`
	entityTypeNode = "node"
	nodeModelAlias = "dm"
)

func init() {
	// get edge by serial number
	queryMap["SelectEdgeDevices"] = `SELECT * FROM edge_device_model WHERE (:serial_number != '' AND LOWER(serial_number) = LOWER(:serial_number)) OR (:serial_number = '' AND (tenant_id = :tenant_id) AND (:id = '' OR id = :id) AND (:edge_cluster_id = '' OR edge_cluster_id = :edge_cluster_id))`

	//TODO: delete next
	// get edge device common function
	queryMap["SelectEdgeDevicesTemplate"] = `SELECT * FROM edge_device_model WHERE (:serial_number != '' AND LOWER(serial_number) = LOWER(:serial_number)) OR (:serial_number = '' AND (tenant_id = :tenant_id) AND (:id = '' OR id = :id) AND (:edge_cluster_id = '' OR edge_cluster_id = :edge_cluster_id)) %s`
	// v2 query, get edge device common function
	queryMap["SelectEdgeDevicesInTemplate"] = `SELECT * FROM edge_device_model WHERE tenant_id = :tenant_id AND (id IN (:edge_device_ids)) %s`

	// v1 query
	queryMap["SelectEdgeDeviceByClusterIdsTemplate"] = `SELECT * FROM edge_device_model WHERE edge_cluster_id IN (:edge_cluster_ids) %s`

	queryMap["CreateEdgeDevice"] = `INSERT INTO edge_device_model (id, role, version, tenant_id, name, description, serial_number, ip_address, gateway, subnet, edge_cluster_id, ssh_public_key, created_at, updated_at) VALUES (:id, :role, :version, :tenant_id, :name, :description, :serial_number, :ip_address, :gateway, :subnet, :edge_cluster_id, :ssh_public_key, :created_at, :updated_at)`

	queryMap["UpdateEdgeDevice"] = `UPDATE edge_device_model SET version = :version, name = :name, description = :description, ip_address = :ip_address, gateway = :gateway, subnet = :subnet, serial_number = :serial_number, edge_cluster_id = :edge_cluster_id, updated_at = :updated_at, role = :role WHERE tenant_id = :tenant_id AND id = :id`

	queryMap["UpdateEdgeDeviceAsBootStrapMaster"] = `UPDATE edge_device_model SET is_bootstrap_master = true WHERE id = :id AND edge_cluster_id = :edge_cluster_id AND NOT EXISTS (SELECT * from edge_device_model where (is_bootstrap_master = true AND edge_cluster_id = :edge_cluster_id))`

	queryMap["SelectTargetDeviceIDsByTypeTemplate"] = `SELECT dm.id FROM edge_device_model dm, edge_cluster_model cm WHERE dm.edge_cluster_id = cm.id AND
		dm.tenant_id = :tenant_id AND (:type = '' OR cm.type = :type OR (:type = 'EDGE' AND cm.type is null)) AND (:edge_cluster_id = '' OR dm.edge_cluster_id = :edge_cluster_id) %s`

	// query around onboard info
	queryMap["UpdateEdgeDeviceOnboarded"] = `UPDATE edge_device_model SET is_onboarded = true, ssh_public_key = :ssh_public_key, updated_at = :updated_at WHERE id = :id AND is_onboarded = false`

	// query to get bootstrap master pub ssh key
	queryMap["SelectBootStrapMasterSshKey"] = `SELECT ssh_public_key FROM edge_device_model WHERE (is_onboarded = true AND is_bootstrap_master = true AND edge_cluster_id = :edge_cluster_id)`

	queryMap["SelectEdgeDeviceIDVirtualIPs"] = `SELECT dm.id, dm.tenant_id, dm.edge_cluster_id, cm.virtual_ip from edge_device_model dm, edge_cluster_model cm WHERE cm.id = dm.edge_cluster_id AND cm.tenant_id = dm.tenant_id AND cm.tenant_id = :tenant_id AND cm.id = :edge_cluster_id`

	orderByHelper.Setup(entityTypeNode, []string{"id", "role", "version", "created_at", "updated_at", "name", "description", "svc_domain_id:edge_cluster_id", "serial_number", "ip_address", "gateway", "subnet", "short_id"})
}

// NodeCoreDBO is DB object model for node core
type NodeCoreDBO struct {
	Name         string          `json:"name" db:"name" validate:"range=1:60"`
	SerialNumber string          `json:"serialNumber" db:"serial_number" validate:"range=0:200"`
	IPAddress    string          `json:"ipAddress" db:"ip_address" validate:"range=0:20"`
	Gateway      string          `json:"gateway" db:"gateway" validate:"range=0:20"`
	Subnet       string          `json:"subnet" db:"subnet" validate:"range=0:20"`
	Role         *types.JSONText `json:"role,omitempty" db:"role"`
}

var defaultNodeRole = toJSONText(`{"master":true,"worker":true}`)

func toJSONText(s string) *types.JSONText {
	jt := types.JSONText([]byte(s))
	return &jt
}

// NodeDBO is DB object model for a node
type NodeDBO struct {
	model.ServiceDomainEntityModelDBO
	NodeCoreDBO
	// Type is null in DB for original nodes
	Description       string  `json:"description" db:"description"`
	IsBootstrapMaster *bool   `json:"isBootstrapMaster" db:"is_bootstrap_master"`
	IsOnboarded       bool    `json:"isOnboarded" db:"is_onboarded"`
	SSHPublicKey      *string `json:"sshPublicKey" db:"ssh_public_key"`
}

func (n *NodeDBO) FillDefaults() {
	if n.Role == nil {
		n.Role = defaultNodeRole
	}
}

// NodeIDsParam used for query
type NodeIDsParam struct {
	TenantID    string   `json:"tenantId" db:"tenant_id"`
	SvcDomainID string   `json:"svcDomainId" db:"edge_cluster_id"`
	NodeIDs     []string `json:"nodeIds" db:"edge_device_ids"`
}

// NodeSSHPubKeyParam used to query SSH pub key
type NodeSSHPubKeyParam struct {
	SSHPubKey   *string `json:"sshPublicKey" db:"ssh_public_key"`
	SvcDomainID string  `json:"svcDomainId" db:"edge_cluster_id"`
}

// NodeVirtualIPParam is used to query for selected node fields and service domain virtual IP
type NodeVirtualIPParam struct {
	model.ServiceDomainEntityModelDBO
	VirtualIP *string `json:"virtualIp" db:"virtual_ip"`
}

// validateVirtualIP returns error if virtual IP is not set when there is already a node in the service domain on adding a node
// or if the service domain is updated to unset virtual IP and there are more than one node in the cluster.
// In case of creation call, it returns the existing node count.
func validateVirtualIP(ctx context.Context, tx *base.WrappedTx, svcDomainID string, updateVirtualIP *string, isCreate bool) (int, error) {
	if updateVirtualIP != nil && *updateVirtualIP != "" && !isCreate {
		// Short circuit
		return 0, nil
	}
	// We are left with - updateVirtualIP is unset (and create/update) or it is an update call or both
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return 0, err
	}
	err = base.LockRows(ctx, tx, "edge_cluster_model", []string{svcDomainID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in virtual IP validation for service domain %s. Error: %s"), svcDomainID, err.Error())
		return 0, err
	}
	virtualIPs := []NodeVirtualIPParam{}
	err = base.QueryTxn(ctx, tx, &virtualIPs, queryMap["SelectEdgeDeviceIDVirtualIPs"], model.ServiceDomainEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: authContext.TenantID}, SvcDomainID: svcDomainID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in virtual IP validation for service domain %s. Error: %s"), svcDomainID, err.Error())
		return 0, err
	}
	nodeCount := len(virtualIPs)
	if nodeCount == 0 {
		// If there is no node, virtual IP can be updated without any constraint
		return nodeCount, nil
	}
	if isCreate {
		// For creation (adding a new node), the existing virtual IP must be set
		if virtualIPs[0].VirtualIP != nil && *virtualIPs[0].VirtualIP != "" {
			return nodeCount, nil
		}
	} else if nodeCount == 1 {
		// For updating the service domain
		return nodeCount, nil
	}
	glog.Errorf(base.PrefixRequestID(ctx, "Error: virtual IP is not set for service domain %s"), svcDomainID)
	return nodeCount, errcode.NewBadRequestError("Virtual IP is required")
}

func (dbAPI *dbObjectModelAPI) filterNodes(ctx context.Context, entities interface{}) (interface{}, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return entities, err
	}
	svcDomainMap, err := dbAPI.getAffiliatedProjectsEdgeClusterIDsMap(ctx)
	if err != nil {
		return entities, err
	}
	// always allow node to get itself
	if ok, svcDomainID := base.IsEdgeRequest(authContext); ok && svcDomainID != "" {
		svcDomainMap[svcDomainID] = true
	}

	return auth.FilterEntitiesByClusterID(entities, svcDomainMap), nil
}

func (dbAPI *dbObjectModelAPI) getNodes(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1, params ...string) ([]model.Node, error) {
	nodes := []model.Node{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nodes, err
	}

	tenantID := authContext.TenantID
	svcDomainID := ""
	if len(params) != 0 {
		svcDomainID = params[0]
	}
	svcDomainModel := model.ServiceDomainEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: tenantID}, SvcDomainID: svcDomainID}
	param := NodeDBO{ServiceDomainEntityModelDBO: svcDomainModel}

	query, err := buildQuery(entityTypeNode, queryMap["SelectEdgeDevicesTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return nodes, err
	}
	_, err = dbAPI.NotPagedQuery(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		nodeDBOPtr := dbObjPtr.(*NodeDBO)
		node := model.Node{}
		err := base.Convert(nodeDBOPtr, &node)
		if err == nil {
			nodes = append(nodes, node)
		}
		return nil
	}, query, param)
	if err != nil {
		return nodes, err
	}
	if len(nodes) == 0 {
		return nodes, nil
	}
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterNodes(ctx, nodes)
		if err == nil {
			nodes = entities.([]model.Node)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "getNodes: filter nodes failed: %s\n"), err.Error())
		}
	}
	return nodes, err
}

func (dbAPI *dbObjectModelAPI) getNodeIDsInPage(ctx context.Context, projectID string, svcDomainID string, queryParam *model.EntitiesQueryParam, targetType model.TargetType) ([]string, []string, error) {
	return dbAPI.GetEntityIDsInPage(ctx, projectID, svcDomainID, queryParam, func(ctx context.Context, svcDomainEntity *model.ServiceDomainEntityModelDBO, queryParam *model.EntitiesQueryParam) ([]string, error) {
		query, err := buildQueryWithTableAlias(entityTypeNode, queryMap["SelectTargetDeviceIDsByTypeTemplate"], queryParam, orderByID, nodeModelAlias, nil)
		if err != nil {
			return []string{}, err
		}
		svcDomainTypeParam := ServiceDomainTypeParam{TenantID: svcDomainEntity.TenantID, SvcDomainID: svcDomainEntity.SvcDomainID, Type: targetType}
		return dbAPI.selectEntityIDsByParam(ctx, query, svcDomainTypeParam)
	})
}

func (dbAPI *dbObjectModelAPI) getNodesW(ctx context.Context, projectID string, clusterID string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	// get query param from request (PageIndex, PageSize, etc)
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /edgedevices, the target type is always edge for backward compatibility
	targetType := extractTargetTypeQueryParam(req)
	nodeIDs, nodeIDsInPage, err := dbAPI.getNodeIDsInPage(ctx, projectID, clusterID, queryParam, targetType)
	if err != nil {
		return err
	}

	nodes := []model.Node{}
	if len(nodeIDsInPage) != 0 {
		nodeDBOs := []NodeDBO{}
		// use in query to find nodeDBOs
		query, err := buildQuery(entityTypeNode, queryMap["SelectEdgeDevicesInTemplate"], queryParam, orderByNameID)
		if err != nil {
			return err
		}
		err = dbAPI.QueryIn(ctx, &nodeDBOs, query, NodeIDsParam{
			TenantID: authContext.TenantID,
			NodeIDs:  nodeIDsInPage,
		})
		if err != nil {
			return err
		}
		for _, nodeDBO := range nodeDBOs {
			node := model.Node{}
			err := base.Convert(&nodeDBO, &node)
			if err != nil {
				return err
			}
			nodes = append(nodes, node)
		}
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: len(nodeIDs), EntityType: entityTypeNode})
	r := model.NodeListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		NodeList:                  nodes,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllNodes selects all nodes for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllNodes(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Node, error) {
	return dbAPI.getNodes(ctx, entitiesQueryParam)
}

// SelectAllNodesW selects all nodes for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesW(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getNodesW(ctx, "", "", w, req)
}

// SelectAllNodesForProject selects all nodes for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllNodesForProject(ctx context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Node, error) {
	nodes := []model.Node{}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nodes, err
	}
	if !auth.IsProjectMember(projectID, authCtx) {
		return nodes, errcode.NewPermissionDeniedError("RBAC")
	}
	// GetProject will properly fill in project.EdgeIDs
	project, err := dbAPI.GetProject(ctx, projectID)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to get projects with id %s, err=%s\n"), projectID, err.Error())
		return nodes, err
	}
	if len(project.EdgeIDs) == 0 {
		return nodes, nil
	}
	param := ServiceDomainIDsParam{
		TenantID:     authCtx.TenantID,
		SvcDomainIDs: project.EdgeIDs,
	}
	query, err := buildQuery(entityTypeNode, queryMap["SelectEdgeDeviceByClusterIdsTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return nodes, err
	}
	err = dbAPI.QueryInWithCallback(ctx, func(dbObjPtr interface{}) error {
		node := model.Node{}
		err := base.Convert(dbObjPtr, &node)
		if err != nil {
			return err
		}
		nodes = append(nodes, node)
		return nil
	}, query, NodeDBO{}, param)
	return nodes, err
}

// SelectAllNodesForProjectW selects all nodes for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getNodesW(ctx, projectID, "", w, req)
}

// GetNode get a node from the DB
func (dbAPI *dbObjectModelAPI) GetNode(ctx context.Context, id string) (model.Node, error) {
	node := model.Node{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return node, err
	}
	tenantID := authContext.TenantID
	nodeDBOs := []NodeDBO{}
	svcDomainModel := model.ServiceDomainEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: tenantID, ID: id}}
	param := NodeDBO{ServiceDomainEntityModelDBO: svcDomainModel}
	if len(id) == 0 {
		return node, errcode.NewBadRequestError("node ID")
	}
	err = dbAPI.Query(ctx, &nodeDBOs, queryMap["SelectEdgeDevices"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetNode: DB select failed for %s. %s\n"), id, err.Error())
		return node, err
	}
	if len(nodeDBOs) == 0 {
		return node, errcode.NewRecordNotFoundError(id)
	}
	nodeDBOPtr := &nodeDBOs[0]
	err = base.Convert(nodeDBOPtr, &node)
	if err != nil {
		return node, err
	}
	nodes := []model.Node{node}
	// filter
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterNodes(ctx, nodes)
		if err == nil {
			nodes = entities.([]model.Node)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "GetNode: filter nodes failed. Error: %s\n"), err.Error())
			return node, err
		}
		if len(nodes) == 0 {
			return node, errcode.NewRecordNotFoundError(id)
		}
	}
	return nodes[0], err
}

// GetNodeW get a node from the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetNodeW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	node, err := dbAPI.GetNode(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, node)
}

// isBootstrapMasterInNodes returns true, if there exists a bootstrap master node
func isBootstrapMasterInNodes(nodes []model.Node) bool {
	for _, node := range nodes {
		if node.IsBootstrapMaster {
			return true
		}
	}
	return false
}

// areAllMasterOnboarded returns true, if all k8s master nodes are onboarded
func areAllMastersOnboarded(nodes []NodeDBO) (bool, error) {
	for _, nodeDBO := range nodes {
		node := model.Node{}
		err := base.Convert(&nodeDBO, &node)
		if err != nil {
			return false, err
		}
		if model.IsK8sMaster(&node) {
			if !nodeDBO.IsOnboarded {
				return false, nil
			}
		}
	}
	return true, nil
}

// GetNodeBySerialNumber get a node by serial number from the DB
func (dbAPI *dbObjectModelAPI) GetNodeBySerialNumber(ctx context.Context, serialNumber string) (model.NodeWithClusterInfo, error) {
	node := model.NodeWithClusterInfo{}
	serialNumber = strings.TrimSpace(serialNumber)
	if len(serialNumber) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Error: Invalid serial number. It is empty"))
		return node, errcode.NewBadRequestError("serialNumber")
	}
	nodeDBOs := []NodeDBO{}
	// Other conditions of the query must not match
	param := NodeDBO{NodeCoreDBO: NodeCoreDBO{SerialNumber: serialNumber}}
	err := dbAPI.Query(ctx, &nodeDBOs, queryMap["SelectEdgeDevices"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetNodeBySerialNumber: DB select failed for serial %s. Error: %s\n"), serialNumber, err.Error())
		return node, err
	}
	if len(nodeDBOs) == 0 {
		return node, errcode.NewBadRequestExError("serialNumber", fmt.Sprintf("Node not found, serialNumber=%s", serialNumber))
	}
	nodeDBOPtr := &nodeDBOs[0]
	err = base.Convert(nodeDBOPtr, &node)
	if err != nil {
		return node, err
	}

	// First K8s master to call this API from the service domain gets promoted to bootstrap master
	if model.IsK8sMaster(&node.Node) {
		result, err := dbAPI.NamedExec(ctx, queryMap["UpdateEdgeDeviceAsBootStrapMaster"], nodeDBOPtr)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in making the node %s bootstrap master. Error: %s"), node.ID, err.Error())
			return node, err
		}
		ok, err := base.DeleteOrUpdateOk(result)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in making the node %s bootstrap master. Error: %s"), node.ID, err.Error())
			return node, err
		}
		if ok {
			node.IsBootstrapMaster = ok
			glog.Infof(base.PrefixRequestID(ctx, "Master selected successfully for service domain %s"), node.SvcDomainID)
		}
	} else {
		// If it's a k8s worker node && there is no bootstrap master, then fail.
		nodes := []model.Node{}
		nodeDBOs := []NodeDBO{}
		param := NodeDBO{}
		param.TenantID = node.TenantID
		param.SvcDomainID = node.SvcDomainID

		reqID := base.GetRequestID(ctx)
		ctx2 := base.GetAdminContext(reqID, node.TenantID)
		err = dbAPI.Query(ctx2, &nodeDBOs, queryMap["SelectEdgeDevices"], param)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx2, "Error in getting all nodes for service domain %s. Error: %s"), node.SvcDomainID, err.Error())
			return node, err
		}
		for _, nodeDBO := range nodeDBOs {
			tempNode := model.Node{}
			err := base.Convert(&nodeDBO, &tempNode)
			if err != nil {
				return node, err
			}
			nodes = append(nodes, tempNode)
		}

		if !isBootstrapMasterInNodes(nodes) {
			errMsg := fmt.Sprintf("Worker node %s in service domain %s waiting for first master to onboard", node.ID, node.SvcDomainID)
			glog.Errorf(base.PrefixRequestID(ctx, errMsg))
			return node, errcode.NewInternalError(errMsg)
		}

		areAllMastersOnboarded, err := areAllMastersOnboarded(nodeDBOs)
		if err != nil {
			return node, err
		}
		if !areAllMastersOnboarded {
			errMsg := fmt.Sprintf("Worker node %s in service domain %s waiting for all k8s masters to onboard", node.ID, node.SvcDomainID)
			glog.Errorf(base.PrefixRequestID(ctx, errMsg))
			return node, errcode.NewInternalError(errMsg)
		}
	}

	sshKeys := []NodeSSHPubKeyParam{}
	// TODO handle for single node?
	// At this point one of the nodes in the service domain is guaranteed to be bootstrap master
	err = dbAPI.Query(ctx, &sshKeys, queryMap["SelectBootStrapMasterSshKey"], NodeSSHPubKeyParam{SvcDomainID: node.SvcDomainID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting bootstrap master public key. Error: %s"), err.Error())
		return node, errcode.TranslateDatabaseError(node.ID, err)
	}
	if len(sshKeys) == 0 {
		if !node.IsBootstrapMaster {
			// Fail for slaves because the slave node can get empty ssh key if the master has
			// first called this API and has not onboarded yet
			errMsg := fmt.Sprintf("No master is found for slave node %s in service domain %s", node.ID, node.SvcDomainID)
			glog.Errorf(base.PrefixRequestID(ctx, errMsg))
			return node, errcode.NewInternalError(errMsg)
		}
		glog.Warningf(base.PrefixRequestID(ctx, "Warning: No master is found in service domain %s"), node.SvcDomainID)
	} else if sshKeys[0].SSHPubKey != nil {
		node.BootstrapMasterSSHPublicKey = *sshKeys[0].SSHPubKey
	}
	return node, err
}

// GetNodeBySerialNumberW get a node by serial number from the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetNodeBySerialNumberW(ctx context.Context, w io.Writer, req *http.Request) error {
	doc := model.SerialNumberPayload{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error decoding into serial number payload. Error: %s"), err.Error())
		return errcode.NewBadRequestError("serialNumber")
	}
	node, err := dbAPI.GetNodeBySerialNumber(ctx, doc.SerialNumber)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, node)
}

// CreateNode creates a node in the DB
func (dbAPI *dbObjectModelAPI) CreateNode(ctx context.Context, i interface{} /* *model.Node */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Node)
	if !ok {
		return resp, errcode.NewInternalError("CreateNode: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(ctx, "CreateNode doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateNode(&doc)
	if err != nil {
		return resp, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityNode,
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
	nodeDBO := NodeDBO{}
	err = base.Convert(&doc, &nodeDBO)
	if err != nil {
		return resp, err
	}
	ftsMap, err := dbAPI.GetFeaturesForServiceDomains(ctx, []string{doc.SvcDomainID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting features for service domain %s. Error: %s"), doc.SvcDomainID, err.Error())
		return resp, err
	}
	// If there is no version, there is no choice - allow creation
	if fts, ok := ftsMap[doc.SvcDomainID]; ok && !fts.MultiNodeAware {
		glog.Errorf(base.PrefixRequestID(ctx, "Failing addition of node to non-multinode service domain %s"), doc.SvcDomainID)
		return resp, errcode.NewInternalError("Unsupported operation for non-multinode aware service domain")
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		nodeCount, err := validateVirtualIP(ctx, tx, nodeDBO.SvcDomainID, nil, true)
		if err != nil {
			return err
		}
		if nodeCount == 0 {
			// First node has the same ID as the service domain ID
			// for old node onboarding to work
			doc.ID = nodeDBO.SvcDomainID
			nodeDBO.ID = nodeDBO.SvcDomainID
		}
		nodeDBO.FillDefaults()
		_, err = tx.NamedExec(ctx, queryMap["CreateEdgeDevice"], &nodeDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating node %s. Error: %s"), doc.ID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.initNodeInfo(ctx, tx, doc.ID, nil, now)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating node info record for node %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addNodeAuditLog(dbAPI, ctx, doc, CREATE)
	return resp, err
}

// CreateNodeW creates a node in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateNodeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, model.ToCreateV2(dbAPI.CreateNode), &model.Node{}, w, r, callback)
}

// UpdateNode updates a node object in the DB
func (dbAPI *dbObjectModelAPI) UpdateNode(ctx context.Context, i interface{} /* *model.Node*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.Node)
	if !ok {
		return model.UpdateDocumentResponse{}, errcode.NewInternalError("UpdateNode: type error")
	}
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
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
	err = model.ValidateNode(&doc)
	if err != nil {
		return resp, err
	}
	if len(doc.ID) == 0 {
		return resp, errcode.NewBadRequestError("nodeId")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityNode,
		meta.OperationUpdate,
		auth.RbacContext{
			ID: doc.ID,
		})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	nodeDBO := NodeDBO{}
	err = base.Convert(&doc, &nodeDBO)
	if err != nil {
		return resp, err
	}
	node, err := dbAPI.GetNode(ctx, doc.ID)
	if err != nil {
		return resp, err
	}

	if node.SvcDomainID != doc.SvcDomainID {
		glog.Errorf(base.PrefixRequestID(ctx, "Service domain ID cannot be modified from %s to %s"), node.SvcDomainID, doc.SvcDomainID)
		return resp, errcode.NewBadRequestError("svcDomainID")
	}
	// get node cert
	nodeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx, doc.SvcDomainID)
	if err != nil {
		return resp, err
	}
	if nodeCert.Locked {
		// disallow changing serial number for locked (= already onboarded) nodes
		// changing letter case is ok
		if strings.ToLower(doc.SerialNumber) != strings.ToLower(node.SerialNumber) {
			return resp, errcode.NewBadRequestError("serialNumber")
		}
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		// note: this method ignores serialNumber update
		nodeDBO.FillDefaults()
		_, err = tx.NamedExec(ctx, queryMap["UpdateEdgeDevice"], &nodeDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "UpdateNode: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		return nil
	})
	if err != nil {
		return resp, err
	}

	if callback != nil {
		go callback(ctx, doc)
	}

	resp.ID = doc.ID
	GetAuditlogHandler().addNodeAuditLog(dbAPI, ctx, doc, UPDATE)
	return resp, nil
}

// UpdateNodeW updates a Node in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateNodeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, model.ToUpdateV2(dbAPI.UpdateNode), &model.Node{}, w, r, callback)
}

// DeleteNode deletes a Node from the DB
func (dbAPI *dbObjectModelAPI) DeleteNode(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	if len(id) == 0 {
		return resp, errcode.NewBadRequestError("id")
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityNode,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	nodeObject, errGetNode := dbAPI.GetNode(ctx, id)
	// doc for callback
	doc := model.Node{
		ServiceDomainEntityModel: model.ServiceDomainEntityModel{
			BaseModel: model.BaseModel{
				TenantID: authContext.TenantID,
				ID:       id,
			},
		},
	}
	nodeOnboarded := false
	fts, err := dbAPI.GetFeaturesForNode(ctx, id)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting features for node %s. Error: %s"), id, err.Error())
		return resp, err
	}
	// If there is no version, there is no choice - allow deletion
	if fts != nil {
		// Not handling transaction as it does not seem critical
		glog.Infof(base.PrefixRequestID(ctx, "Features for node %s: %+v"), id, *fts)
		if fts.MultiNodeAware {
			nodeDBOs := []NodeDBO{}
			param := NodeDBO{}
			param.TenantID = authContext.TenantID
			param.ID = id
			err = dbAPI.Query(ctx, &nodeDBOs, queryMap["SelectEdgeDevices"], param)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the node %s to find onboarding status. Error: %s"), id, err.Error())
				return model.DeleteDocumentResponse{}, err
			}
			if len(nodeDBOs) == 0 {
				// Idempotent deletion
				return model.DeleteDocumentResponse{ID: id}, nil
			}
			nodeOnboarded = nodeDBOs[0].IsOnboarded
		} else {
			// Cert must always exist if the edge is present
			edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx, id)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Error in getting cert to find onboarding status for node %s. Error: %s"), id, err.Error())
				return model.DeleteDocumentResponse{}, err
			}
			nodeOnboarded = edgeCert.Locked
		}
	}

	glog.Infof(base.PrefixRequestID(ctx, "Deleting node %s. Onboarded status %t"), id, nodeOnboarded)

	result, err := DeleteEntity(ctx, dbAPI, "edge_device_model", "id", id, doc, callback)
	if err == nil {
		if errGetNode != nil {
			glog.Error("Error in getting node : ", errGetNode.Error())
		} else {
			GetAuditlogHandler().addNodeAuditLog(dbAPI, ctx, nodeObject, DELETE)
		}
	}
	return result, err
}

// DeleteNodeW deletes a Node from the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteNodeW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteNode), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) UpdateNodeOnboarded(ctx context.Context, doc *model.NodeOnboardInfo) error {
	nodeDBO := NodeDBO{}
	if doc == nil {
		return errcode.NewBadRequestError("doc")
	}
	if doc.NodeID == "" {
		return errcode.NewBadRequestError("nodeId")
	}
	if doc.SSHPublicKey == "" {
		return errcode.NewBadRequestError("sSHPublicKey")
	}
	err := base.ValidateVersion(doc.NodeVersion)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid version %s. Error: %s"), doc.NodeVersion, err.Error())
		return err
	}
	version, err := dbAPI.GetOneNodeVersion(ctx, "", doc.NodeID, true)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting node version. Error: %s"), err.Error())
		return err
	}
	if version != "" && doc.NodeVersion != version {
		errMsg := fmt.Sprintf("Node version must be %s, found %s", version, doc.NodeVersion)
		glog.Errorf(base.PrefixRequestID(ctx, errMsg))
		return errcode.NewBadRequestExError("nodeVersion", errMsg)
	}
	now := base.RoundedNow()
	nodeDBO.BaseModelDBO.ID = doc.NodeID
	nodeDBO.IsOnboarded = true
	nodeDBO.SSHPublicKey = base.StringPtr(doc.SSHPublicKey)
	nodeDBO.UpdatedAt = now
	return dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		result, err := tx.NamedExec(ctx, queryMap["UpdateEdgeDeviceOnboarded"], &nodeDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating node onboarding status for %s. Error: %s"), nodeDBO.ID, err.Error())
			return err
		}
		ok, err := base.DeleteOrUpdateOk(result)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating onboarding status of node %s. Error: %s"), nodeDBO.ID, err.Error())
			return err
		}
		if !ok {
			glog.Errorf(base.PrefixRequestID(ctx, "Error: This device %s could already be onboarded or the version is different"), nodeDBO.ID)
			return errcode.NewInternalError("Node already onboarded")
		}
		// Node info must be updated to have this version
		err = dbAPI.setNodeVersion(ctx, tx, "", doc.NodeID, doc.NodeVersion, now)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating node version for %s. Error: %s"), doc.NodeID, err.Error())
			return err
		}
		return nil
	})
}

// UpdateNodeOnboardedW update post onboard info for a node.
func (dbAPI *dbObjectModelAPI) UpdateNodeOnboardedW(ctx context.Context, w io.Writer, req *http.Request) error {
	doc := model.NodeOnboardInfo{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error decoding into node onboard info. Error: %s"), err.Error())
		return errcode.NewBadRequestError("NodeOnbaordedInfo")
	}
	return dbAPI.UpdateNodeOnboarded(ctx, &doc)
}

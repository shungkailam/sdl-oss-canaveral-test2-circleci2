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

// Defined and reused from edgeApi.go
const (
	// entityTypeEdgeDevice is the entity type `edgeDevice`
	entityTypeEdgeDevice = "edgeDevice"
)

func init() {
	orderByHelper.Setup(entityTypeEdgeDevice, []string{"id", "version", "created_at", "updated_at", "name", "description", "edge_cluster_id", "serial_number", "ip_address", "gateway", "subnet", "short_id"})
}

// EdgeDeviceCoreDBO is DB object model for edge device core
type EdgeDeviceCoreDBO struct {
	Name         string          `json:"name" db:"name" validate:"range=1:60"`
	SerialNumber string          `json:"serialNumber" db:"serial_number" validate:"range=0:200"`
	IPAddress    string          `json:"ipAddress" db:"ip_address" validate:"range=0:20"`
	Gateway      string          `json:"gateway" db:"gateway" validate:"range=0:20"`
	Subnet       string          `json:"subnet" db:"subnet" validate:"range=0:20"`
	Role         *types.JSONText `json:"role,omitempty" db:"role"`
}

// EdgeDeviceDBO is DB object model for edge device
type EdgeDeviceDBO struct {
	model.ClusterEntityModelDBO
	EdgeDeviceCoreDBO
	// Type is null in DB for original edges
	Description       string  `json:"description" db:"description"`
	IsBootstrapMaster *bool   `json:"isBootstrapMaster" db:"is_bootstrap_master"`
	IsOnboarded       bool    `json:"isOnboarded" db:"is_onboarded"`
	SSHPublicKey      *string `json:"sshPublicKey" db:"ssh_public_key"`
}

func (n *EdgeDeviceDBO) FillDefaults() {
	if n.Role == nil {
		n.Role = defaultNodeRole
	}
}

// EdgeDeviceIDsParam used for query
type EdgeDeviceIDsParam struct {
	TenantID  string   `json:"tenantId" db:"tenant_id"`
	ClusterID string   `json:"clusterId" db:"edge_cluster_id"`
	DeviceIDs []string `json:"deviceIds" db:"edge_device_ids"`
}

// EdgeDeviceSSHPubKeyParam used to query SSH pub key
type EdgeDeviceSSHPubKeyParam struct {
	SSHPubKey *string `json:"sshPublicKey" db:"ssh_public_key"`
	ClusterID string  `json:"clusterId" db:"edge_cluster_id"`
}

// EdgeDeviceVirtualIPParam is used to query for selected device fields and cluster virtual IP
type EdgeDeviceVirtualIPParam struct {
	model.ClusterEntityModelDBO
	VirtualIP *string `json:"virtualIp" db:"virtual_ip"`
}

func (dbAPI *dbObjectModelAPI) filterEdgeDevices(context context.Context, entities interface{}) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return entities, err
	}
	edgeClusterMap, err := dbAPI.getAffiliatedProjectsEdgeClusterIDsMap(context)
	if err != nil {
		return entities, err
	}
	// always allow edge device to get itself
	if ok, edgeClusterID := base.IsEdgeRequest(authContext); ok && edgeClusterID != "" {
		edgeClusterMap[edgeClusterID] = true
	}

	return auth.FilterEntitiesByClusterID(entities, edgeClusterMap), nil
}

func (dbAPI *dbObjectModelAPI) getEdgeDevices(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeDevice, error) {
	edgeDevices := []model.EdgeDevice{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeDevices, err
	}

	tenantID := authContext.TenantID
	clusterModel := model.ClusterEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: tenantID}}
	param := EdgeDeviceDBO{ClusterEntityModelDBO: clusterModel}

	query, err := buildQuery(entityTypeEdgeDevice, queryMap["SelectEdgeDevicesTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return edgeDevices, err
	}
	_, err = dbAPI.NotPagedQuery(context, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		edgeDeviceDBOPtr := dbObjPtr.(*EdgeDeviceDBO)
		edgeDevice := model.EdgeDevice{}
		err := base.Convert(edgeDeviceDBOPtr, &edgeDevice)
		if err == nil {
			edgeDevices = append(edgeDevices, edgeDevice)
		}
		return nil
	}, query, param)
	if err != nil {
		return edgeDevices, err
	}
	if len(edgeDevices) == 0 {
		return edgeDevices, nil
	}
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterEdgeDevices(context, edgeDevices)
		if err == nil {
			edgeDevices = entities.([]model.EdgeDevice)
		} else {
			glog.Errorf(base.PrefixRequestID(context, "SelectAllEdgeDevices: filter edge deivces failed: %s\n"), err.Error())
		}
	}
	return edgeDevices, err
}

func (dbAPI *dbObjectModelAPI) getEdgeDevicesW(context context.Context, projectID string, clusterID string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	// get query param from request (PageIndex, PageSize, etc)
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /edgedevices, the target type is always edge for backward compatibility
	targetType := extractTargetTypeQueryParam(req)
	edgeDeviceIDs, edgeDeviceIDsInPage, err := dbAPI.getNodeIDsInPage(context, projectID, clusterID, queryParam, targetType)
	if err != nil {
		return err
	}

	edgeDevices := []model.EdgeDevice{}
	if len(edgeDeviceIDsInPage) != 0 {
		edgeDeviceDBOs := []EdgeDeviceDBO{}
		// use in query to find edgeDeviceDBOs
		query, err := buildQuery(entityTypeEdge, queryMap["SelectEdgeDevicesInTemplate"], queryParam, orderByNameID)
		if err != nil {
			return err
		}
		err = dbAPI.QueryIn(context, &edgeDeviceDBOs, query, EdgeDeviceIDsParam{
			TenantID:  authContext.TenantID,
			DeviceIDs: edgeDeviceIDsInPage,
		})
		if err != nil {
			return err
		}
		// convert edgeDeviceDBO to edgeDevice
		for _, edgeDeviceDBO := range edgeDeviceDBOs {
			edgeDevice := model.EdgeDevice{}
			err := base.Convert(&edgeDeviceDBO, &edgeDevice)
			if err != nil {
				return err
			}
			edgeDevices = append(edgeDevices, edgeDevice)
		}
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: len(edgeDeviceIDs), EntityType: entityTypeEdgeDevice})
	r := model.EdgeDeviceListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		EdgeDeviceList:            edgeDevices,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllEdgeDevicesW select all edge devices for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeDevicesW(context, "", "", w, req)
}

// SelectAllEdgeDevicesForProjectW select all edge devices for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeDevicesW(context, projectID, "", w, req)
}

// GetEdgeDevice get an edge device object in the DB
func (dbAPI *dbObjectModelAPI) GetEdgeDevice(context context.Context, id string) (model.EdgeDevice, error) {
	edgeDevice := model.EdgeDevice{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeDevice, err
	}
	tenantID := authContext.TenantID
	edgeDeviceDBOs := []EdgeDeviceDBO{}
	clusterModel := model.ClusterEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: tenantID, ID: id}}
	param := EdgeDeviceDBO{ClusterEntityModelDBO: clusterModel}
	if len(id) == 0 {
		return edgeDevice, errcode.NewBadRequestError("edgeDeviceID")
	}
	err = dbAPI.Query(context, &edgeDeviceDBOs, queryMap["SelectEdgeDevices"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "GetEdgeDevice: DB select failed: %s\n"), err.Error())
		return edgeDevice, err
	}
	if len(edgeDeviceDBOs) == 0 {
		return edgeDevice, errcode.NewRecordNotFoundError(id)
	}
	edgeDeviceDBOPtr := &edgeDeviceDBOs[0]
	err = base.Convert(edgeDeviceDBOPtr, &edgeDevice)
	if err != nil {
		return edgeDevice, err
	}
	edgeDevices := []model.EdgeDevice{edgeDevice}
	// filter
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterEdgeDevices(context, edgeDevices)
		if err == nil {
			edgeDevices = entities.([]model.EdgeDevice)
		} else {
			glog.Errorf(base.PrefixRequestID(context, "GetEdge: filter edge devices failed: %s\n"), err.Error())
			return edgeDevice, err
		}
		if len(edgeDevices) == 0 {
			return edgeDevice, errcode.NewRecordNotFoundError(id)
		}
	}
	return edgeDevices[0], err
}

// GetEdgeDeviceW get a edge device object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeDeviceW(context context.Context, id string, w io.Writer, req *http.Request) error {
	edgeDevice, err := dbAPI.GetEdgeDevice(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edgeDevice)
}

// GetEdgeDeviceBySerialNumber get edge device object by in the DB by serial number
func (dbAPI *dbObjectModelAPI) GetEdgeDeviceBySerialNumber(context context.Context, serialNumber string) (model.EdgeDeviceWithClusterInfo, error) {
	edgeDevice := model.EdgeDeviceWithClusterInfo{}
	serialNumber = strings.TrimSpace(serialNumber)
	if len(serialNumber) == 0 {
		glog.Errorf(base.PrefixRequestID(context, "Error: Invalid serial number. It is empty"))
		return edgeDevice, errcode.NewBadRequestError("serialNumber")
	}
	edgeDeviceDBOs := []EdgeDeviceDBO{}
	// Other conditions of the query must not match
	param := EdgeDeviceDBO{EdgeDeviceCoreDBO: EdgeDeviceCoreDBO{SerialNumber: serialNumber}}
	err := dbAPI.Query(context, &edgeDeviceDBOs, queryMap["SelectEdgeDevices"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "GetEdgeDeviceBySerialNumber: DB select failed: %s\n"), err.Error())
		return edgeDevice, err
	}
	if len(edgeDeviceDBOs) == 0 {
		return edgeDevice, errcode.NewBadRequestExError("serialNumber", fmt.Sprintf("Edge Device not found, serialNumber=%s", serialNumber))
	}
	edgeDeviceDBOPtr := &edgeDeviceDBOs[0]
	err = base.Convert(edgeDeviceDBOPtr, &edgeDevice)
	if err != nil {
		return edgeDevice, err
	}
	// If this is the first device to call this API in the cluster - then its the first to onboard in the cluster. Make it bootstrap master
	result, err := dbAPI.NamedExec(context, queryMap["UpdateEdgeDeviceAsBootStrapMaster"], edgeDeviceDBOPtr)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in making the edge device %s bootstrap master. Error: %s"), edgeDevice.ID, err.Error())
		// failed to update as master but we can still return the edge device
		return edgeDevice, err
	}
	ok, err := base.DeleteOrUpdateOk(result)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in making the edge device %s bootstrap master. Error: %s"), edgeDevice.ID, err.Error())
		return edgeDevice, err
	}
	if ok {
		edgeDevice.IsBootstrapMaster = ok
		glog.Infof(base.PrefixRequestID(context, "Master selected successfully for cluster %s"), edgeDevice.ClusterID)
	}
	sshKeyParams := []EdgeDeviceSSHPubKeyParam{}
	// TODO handle for single node?
	// At this point one of the devices in the cluster is guaranteed to be bootstrap master
	err = dbAPI.Query(context, &sshKeyParams, queryMap["SelectBootStrapMasterSshKey"], EdgeDeviceSSHPubKeyParam{ClusterID: edgeDevice.ClusterID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in getting bootstrap master public key. Error: %s"), err.Error())
		return edgeDevice, errcode.TranslateDatabaseError(edgeDevice.ID, err)
	}
	if len(sshKeyParams) == 0 {
		glog.Warningf(base.PrefixRequestID(context, "Warning: No master is found in cluster %s"), edgeDevice.ClusterID)
	} else if sshKeyParams[0].SSHPubKey != nil {
		edgeDevice.BootstrapMasterSSHPublicKey = *sshKeyParams[0].SSHPubKey
	}
	// Do not set connection as tenant is not authenticated
	return edgeDevice, err
}

//
// GetEdgeDeviceBySerialNumberW get edge device object by in the DB by serial number
func (dbAPI *dbObjectModelAPI) GetEdgeDeviceBySerialNumberW(context context.Context, w io.Writer, req *http.Request) error {
	doc := model.SerialNumberPayload{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into edge device. Error: %s"), err.Error())
		return errcode.NewBadRequestError("serialNumber")
	}
	edgeDevice, err := dbAPI.GetEdgeDeviceBySerialNumber(context, doc.SerialNumber)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edgeDevice)
}

// CreateEdge creates an edge device object in the DB
func (dbAPI *dbObjectModelAPI) CreateEdgeDevice(context context.Context, i interface{} /* *model.EdgeDevice */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.EdgeDevice)
	if !ok {
		return resp, errcode.NewInternalError("CreateEdgeDevice: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateEdgeDevice doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateEdgeDevice(&doc)
	if err != nil {
		return resp, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeDevice,
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
	edgeDeviceDBO := EdgeDeviceDBO{}
	err = base.Convert(&doc, &edgeDeviceDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		deviceCount, err := validateVirtualIP(context, tx, edgeDeviceDBO.ClusterID, nil, true)
		if err != nil {
			return err
		}
		if deviceCount == 0 {
			// First device has the same ID as the cluster ID
			// for old edge onboarding to work
			doc.ID = edgeDeviceDBO.ClusterID
			edgeDeviceDBO.ID = edgeDeviceDBO.ClusterID
		}
		edgeDeviceDBO.FillDefaults()
		_, err = tx.NamedExec(context, queryMap["CreateEdgeDevice"], &edgeDeviceDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating edge device for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.initEdgeDeviceInfo(context, tx, doc.ID, now)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating edge device info record for device %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	return resp, err
}

// CreateEdgeW creates an edge device object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateEdgeDeviceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateEdgeDevice), &model.EdgeDevice{}, w, r, callback)
}

// UpdateEdgeDevice update an edge device object in the DB
func (dbAPI *dbObjectModelAPI) UpdateEdgeDevice(context context.Context, i interface{} /* *model.EdgeDevice*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.EdgeDevice)
	if !ok {
		return model.UpdateDocumentResponse{}, errcode.NewInternalError("UpdateEdgeDevice: type error")
	}
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
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
	err = model.ValidateEdgeDevice(&doc)
	if err != nil {
		return resp, err
	}
	if len(doc.ID) == 0 {
		return resp, errcode.NewBadRequestError("edgeDeviceID")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeDevice,
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
	edgeDeviceDBO := EdgeDeviceDBO{}
	err = base.Convert(&doc, &edgeDeviceDBO)
	if err != nil {
		return resp, err
	}
	edgeDevice, err := dbAPI.GetEdgeDevice(context, doc.ID)
	if err != nil {
		return resp, err
	}

	if edgeDevice.ClusterID != doc.ClusterID {
		glog.Errorf(base.PrefixRequestID(context, "Cluster ID cannot be modified from %s to %s"), edgeDevice.ClusterID, doc.ClusterID)
		return resp, errcode.NewBadRequestError("clusterID")
	}
	// get edge cert
	edgeCert, err := dbAPI.GetEdgeCertByEdgeID(context, doc.ClusterID)
	if err != nil {
		return resp, err
	}
	if edgeCert.Locked {
		// disallow changing serial number for locked (= already onboarded) edges
		// changing letter case is ok
		if strings.ToLower(doc.SerialNumber) != strings.ToLower(edgeDevice.SerialNumber) {
			return resp, errcode.NewBadRequestError("serialNumber")
		}
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		// note: this method ignores serialNumber update
		// note: this method is also currently allowing us to update the edge cluster

		edgeDeviceDBO.FillDefaults()
		_, err = tx.NamedExec(context, queryMap["UpdateEdgeDevice"], &edgeDeviceDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "UpdateEdgeDevice: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		return nil
	})
	if err != nil {
		return resp, err
	}

	if callback != nil {
		go callback(context, doc)
	}

	resp.ID = doc.ID
	return resp, nil
}

// UpdateEdgeW update an edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateEdgeDeviceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateEdgeDevice), &model.EdgeDevice{}, w, r, callback)
}

// DeleteEdge delete a edge device object in the DB
func (dbAPI *dbObjectModelAPI) DeleteEdgeDevice(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeDevice,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	doc := model.EdgeDevice{
		ClusterEntityModel: model.ClusterEntityModel{
			BaseModel: model.BaseModel{
				TenantID: authContext.TenantID,
				ID:       id,
			},
		},
	}
	return DeleteEntity(context, dbAPI, "edge_device_model", "id", id, doc, callback)
}

// DeleteEdgeDeviceW delete a edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteEdgeDeviceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteEdgeDevice), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) UpdateEdgeDeviceOnboarded(context context.Context, id string, sshPublicKey string) error {
	edgeDeviceDBO := EdgeDeviceDBO{}
	edgeDeviceDBO.BaseModelDBO.ID = id
	edgeDeviceDBO.IsOnboarded = true
	edgeDeviceDBO.SSHPublicKey = base.StringPtr(sshPublicKey)

	result, err := dbAPI.NamedExec(context, queryMap["UpdateEdgeDeviceOnboarded"], &edgeDeviceDBO)

	ok, err := base.DeleteOrUpdateOk(result)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating onboarding status of edge device %s. Error: %s"), edgeDeviceDBO.ID, err.Error())
		return err
	}
	if !ok {
		glog.Errorf(base.PrefixRequestID(context, "Error: This device %s could already be onboarded"), edgeDeviceDBO.ID)
		return errcode.NewInternalError("Device already onboarded")
	}
	return nil
}

// UpdateEdgeDeviceOnboardedW update post onboard info for device.
func (dbAPI *dbObjectModelAPI) UpdateEdgeDeviceOnboardedW(context context.Context, w io.Writer, req *http.Request) error {
	doc := model.EdgeDeviceOnboardInfo{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into device onboard info. Error: %s"), err.Error())
		return errcode.NewBadRequestError("DeviceOnboarded")
	}
	err = dbAPI.UpdateEdgeDeviceOnboarded(context, doc.EdgeDeviceID, doc.SSHPublicKey)
	if err != nil {
		return err
	}

	return err
}

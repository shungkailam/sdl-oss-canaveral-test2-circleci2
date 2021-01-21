package api

import (
	"cloudservices/cloudmgmt/cfssl"
	cfsslModels "cloudservices/cloudmgmt/generated/cfssl/models"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
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
	funk "github.com/thoas/go-funk"
)

const (
	// entityTypeEdge is the entity type `edge`
	entityTypeEdge = "edge"
)

var (
	namedExec = func(tx *base.WrappedTx, ctx context.Context, query string, obj interface{}) error {
		_, err := tx.NamedExec(ctx, query, obj)
		return err
	}
)

func init() {
	// At least one must be returned in case there are multiple ID entries in the same cluster because IDs are unique
	queryMap["EdgeMultinodeAwareQuery"] = `SELECT m2.* FROM edge_device_model m1, edge_device_model m2 WHERE  m1.id = :id AND (m2.edge_cluster_id = m1.edge_cluster_id) AND m2.id != m2.edge_cluster_id`
	orderByHelper.Setup(entityTypeEdge, []string{"id", "version", "created_at", "updated_at", "name", "description", "serial_number", "ip_address", "gateway", "subnet", "edge_devices", "short_id"})
}

// EdgeIDsParam is used in SQL IN query on edge ids
type EdgeIDsParam struct {
	TenantID string   `json:"tenantId" db:"tenant_id"`
	EdgeIDs  []string `json:"edgeIds" db:"edge_ids"`
}

// extract the type query param. It is hidden in the API doc
func extractTargetTypeQueryParam(req *http.Request) model.TargetType {
	var targetType model.TargetType
	if req != nil {
		query := req.URL.Query()
		values := query["type"]
		var value string
		if len(values) == 1 {
			value = values[0]
			if strings.ToUpper(value) == string(model.RealTargetType) {
				targetType = model.RealTargetType
			} else if strings.ToUpper(value) == string(model.CloudTargetType) {
				targetType = model.CloudTargetType
			}
		}
	}
	return targetType
}

func (dbAPI *dbObjectModelAPI) filterEdges(context context.Context, entities interface{}) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return entities, err
	}
	edgeMap, err := dbAPI.getAffiliatedProjectsEdgeIDsMap(context)
	if err != nil {
		return entities, err
	}
	// always allow edge to get itself
	if ok, edgeID := base.IsEdgeRequest(authContext); ok && edgeID != "" {
		edgeMap[edgeID] = true
	}
	return auth.FilterEntitiesByID(entities, edgeMap), nil
}

func (dbAPI *dbObjectModelAPI) isEdgeMultinodeAware(ctx context.Context, id string) (bool, error) {
	// TODO: revisit, for now its a simple check
	// This might miss the case if a user creates a new edgeCluster and edgeDevice with the same id
	edgeDeviceDBOs := []EdgeDeviceDBO{}
	err := dbAPI.Query(ctx, &edgeDeviceDBOs, queryMap["EdgeMultinodeAwareQuery"], model.BaseModelDBO{ID: id})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in multinode aware query for edge %s. Error: %s"), id, err.Error())
		return false, errcode.TranslateDatabaseError(id, err)
	}
	return len(edgeDeviceDBOs) > 0, nil
}

func (dbAPI *dbObjectModelAPI) GetEdgeProjects(ctx context.Context, edgeID string) ([]model.Project, error) {
	return dbAPI.GetEdgeClusterProjects(ctx, edgeID)
}

func (dbAPI *dbObjectModelAPI) GetEdgeProjectRoles(ctx context.Context, edgeID string) ([]model.ProjectRole, error) {
	return dbAPI.GetEdgeClusterProjectRoles(ctx, edgeID)
}

func (dbAPI *dbObjectModelAPI) getEdges(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Edge, error) {
	edges := []model.Edge{}
	edgeDevices, err := dbAPI.getEdgeDevices(context, entitiesQueryParam)
	if err != nil {
		return edges, err
	}
	edges, err = dbAPI.edgesFromDevices(context, edgeDevices)
	return edges, err
}

func (dbAPI *dbObjectModelAPI) getEdgesWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	// TODO: fix targetType, also get only older edges
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	// get query param from request (PageIndex, PageSize, etc)
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /edges, the target type is always edge for backward compatibility
	targetType := extractTargetTypeQueryParam(req)
	edgeDeviceIDs, edgeDeviceIDsInPage, err := dbAPI.getNodeIDsInPage(context, projectID, "", queryParam, targetType)
	if err != nil {
		return err
	}
	edges := []model.Edge{}
	if len(edgeDeviceIDsInPage) != 0 {
		edgeDeviceDBOs := []EdgeDeviceDBO{}
		// use in query to find edgeDeviceDBOs
		query, err := buildQuery(entityTypeEdgeDevice, queryMap["SelectEdgeDevicesInTemplate"], queryParam, orderByNameID)
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
		edgeDevices := []model.EdgeDevice{}
		for _, edgeDeviceDBO := range edgeDeviceDBOs {
			edgeDevice := model.EdgeDevice{}
			err := base.Convert(&edgeDeviceDBO, &edgeDevice)
			if err != nil {
				return err
			}
			edgeDevices = append(edgeDevices, edgeDevice)
		}
		edges, err = dbAPI.edgesFromDevices(context, edgeDevices)
		if err != nil {
			return err
		}
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: len(edgeDeviceIDs), EntityType: entityTypeEdge})
	r := model.EdgeListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		EdgeListV2:                model.EdgesByID(edges).ToV2(),
	}
	return json.NewEncoder(w).Encode(r)

}

// SelectAllEdges select all edges for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllEdges(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Edge, error) {
	return dbAPI.getEdges(context, entitiesQueryParam)
}

// SelectAllEdgesW select all edges for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesW(context context.Context, w io.Writer, req *http.Request) error {
	// can't avoid edgeDBO -> edge conversion here in general since only edge carry labels
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	edges, err := dbAPI.SelectAllEdges(context, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edges)
}

// SelectAllEdgesWV2 select all edges for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesWV2(context context.Context, w io.Writer, req *http.Request) error {
	// can't avoid edgeDBO -> edge conversion here in general since only edge carry labels
	return dbAPI.getEdgesWV2(context, "", w, req)
}

// SelectAllEdgesForProject select all edges for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllEdgesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Edge, error) {
	// This will return both multinode and edges which do not have multinode support
	edges := []model.Edge{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edges, err
	}
	if !auth.IsProjectMember(projectID, authContext) {
		return edges, errcode.NewPermissionDeniedError("RBAC")
	}
	// GetProject will properly fill in project.EdgeIDs
	project, err := dbAPI.GetProject(context, projectID)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(context, "Failed to get projects with id %s, err=%s\n"), projectID, err.Error())
		return edges, err
	}
	if len(project.EdgeIDs) == 0 {
		return edges, nil
	}
	param := EdgeClusterIDsParam{
		ClusterIDs: project.EdgeIDs,
	}
	query, err := buildQuery(entityTypeEdgeDevice, queryMap["SelectEdgeDeviceByClusterIdsTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return edges, err
	}
	edgeDevices := []model.EdgeDevice{}
	err = dbAPI.QueryInWithCallback(context, func(dbObjPtr interface{}) error {
		edgeDeviceDBOPtr := dbObjPtr.(*EdgeDeviceDBO)
		edgeDevice := model.EdgeDevice{}
		err := base.Convert(edgeDeviceDBOPtr, &edgeDevice)
		if err != nil {
			return err
		}
		edgeDevices = append(edgeDevices, edgeDevice)
		return nil
	}, query, EdgeDeviceDBO{}, param)

	if err == nil {
		edges, err = dbAPI.edgesFromDevices(context, edgeDevices)
	}

	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "SelectAllEdgesForProjectW: DB query failed: %s\n"), err.Error())
	}

	return edges, err
}

// SelectAllEdgesForProjectW select all edges for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	edges, err := dbAPI.SelectAllEdgesForProject(context, projectID, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edges)
}

// SelectAllEdgesForProjectWV2 select all edges for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgesForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgesWV2(context, projectID, w, req)
}

// GetEdge get a edge object in the DB.
// A device must be present for this to method to be successful
func (dbAPI *dbObjectModelAPI) GetEdge(context context.Context, id string) (model.Edge, error) {
	edge := model.Edge{}
	edgeDevice, err := dbAPI.GetEdgeDevice(context, id)
	if err != nil {
		if !errcode.IsRecordNotFound(err) {
			return edge, err
		}
		// not found, treat id as edgeClusterID
		edgeDeviceIDs, err2 := dbAPI.getEdgeClusterDeviceIDs(context, id)
		if err2 != nil {
			return edge, err
		}
		if len(edgeDeviceIDs) == 0 {
			return edge, errcode.NewRecordNotFoundError(id)
		}
		// simply use the first edge device id
		edgeDeviceID := edgeDeviceIDs[0]
		edgeDevice, err = dbAPI.GetEdgeDevice(context, edgeDeviceID)
		if err != nil {
			return edge, err
		}
	}
	edge, err = dbAPI.edgeFromDevice(context, edgeDevice)
	if err != nil {
		return edge, err
	}
	return edge, nil
}

// GetEdgeW get a edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeW(context context.Context, id string, w io.Writer, req *http.Request) error {
	edge, err := dbAPI.GetEdge(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edge)
}

// GetEdgeWV2 get a edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeWV2(context context.Context, id string, w io.Writer, req *http.Request) error {
	edge, err := dbAPI.GetEdge(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edge.ToV2())
}

// edgeFromDevice get edge object using edgeDevice and edgeCluster
func (dbAPI *dbObjectModelAPI) edgeFromDevice(context context.Context, edgeDevice model.EdgeDevice) (model.Edge, error) {
	edge := edgeDevice.ToEdge()
	edgeCluster, err := dbAPI.GetEdgeCluster(context, edgeDevice.ClusterID)
	if err != nil {
		return edge, err
	}
	edge.Labels = edgeCluster.Labels
	edge.ShortID = edgeCluster.ShortID
	edge.Type = edgeCluster.Type
	edge.Connected = edgeCluster.Connected
	return edge, nil
}

// edgesFromDevices get edge objects using edgeDevices and edgeClusters
func (dbAPI *dbObjectModelAPI) edgesFromDevices(context context.Context, edgeDevices []model.EdgeDevice) ([]model.Edge, error) {
	edges := []model.Edge{}
	edgeClusterIDs := funk.Map(edgeDevices, func(x interface{}) string {
		return x.(model.EdgeDevice).ClusterID
	}).([]string)
	edgeClusters, err := dbAPI.getEdgeClustersCore(context, edgeClusterIDs, nil)
	if err != nil {
		return edges, err
	}
	// build lookup map in case there are duplicate edge cluster ids
	edgeClusterMap := map[string]*model.EdgeCluster{}
	for i := range edgeClusters {
		pc := &edgeClusters[i]
		edgeClusterMap[pc.ID] = pc
	}

	for i := range edgeDevices {
		edgeDevice := edgeDevices[i]
		edge := edgeDevice.ToEdge()
		pc := edgeClusterMap[edgeDevice.ClusterID]
		if pc == nil {
			return edges, errcode.NewBadRequestExError("edgeClusterID", fmt.Sprintf("Edge Cluster with ID %s could not be found", edgeDevice.ClusterID))
		}
		edgeCluster := *pc
		edge.Labels = edgeCluster.Labels
		edge.ShortID = edgeCluster.ShortID
		edge.Type = edgeCluster.Type
		edge.Connected = edgeCluster.Connected
		edges = append(edges, edge)
	}
	return edges, nil
}

// GetEdgeBySerialNumber get edge object by in the DB by serial number
func (dbAPI *dbObjectModelAPI) GetEdgeBySerialNumber(context context.Context, serialNumber string) (model.EdgeDeviceWithClusterInfo, error) {
	edgeDevice, err := dbAPI.GetEdgeDeviceBySerialNumber(context, serialNumber)
	if err != nil {
		return edgeDevice, err
	}
	return edgeDevice, nil
}

//
// GetEdgeBySerialNumberW get edge object by in the DB by serial number
func (dbAPI *dbObjectModelAPI) GetEdgeBySerialNumberW(context context.Context, w io.Writer, req *http.Request) error {
	doc := model.SerialNumberPayload{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into edge. Error: %s"), err.Error())
		return errcode.NewBadRequestError("SerialNumber")
	}
	edge, err := dbAPI.GetEdgeBySerialNumber(context, doc.SerialNumber)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, edge); handled {
	// 	return err
	// }
	return base.DispatchPayload(w, edge)
}

// CreateEdge creates an edge object in the DB
func (dbAPI *dbObjectModelAPI) CreateEdge(context context.Context, i interface{} /* *model.Edge */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Edge)
	if !ok {
		return resp, errcode.NewInternalError("CreateEdge: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateEdge doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateEdge(&doc)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdge,
		meta.OperationCreate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.Connected = IsEdgeConnected(tenantID, doc.ID)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	edgeClusterDoc := doc.ToEdgeCluster()
	edgeClusterDBO := EdgeClusterDBO{}
	err = base.Convert(&edgeClusterDoc, &edgeClusterDBO)
	if err != nil {
		return resp, err
	}

	edgeDeviceDoc := doc.ToEdgeDevice()
	edgeDeviceDBO := EdgeDeviceDBO{}
	err = base.Convert(&edgeDeviceDoc, &edgeDeviceDBO)
	if err != nil {
		return resp, err
	}

	// first get tenant token
	tenant, err := dbAPI.GetTenant(context, tenantID)
	if err != nil {
		return resp, err
	}
	// Create edge certificates using per-tenant root CA.
	edgeCertResp, err := cfssl.GetCert(tenantID, cfsslModels.CertificatePostParamsTypeServer)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "CreateEdgeCluster: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
		return resp, errcode.NewInternalError(err.Error())
	}
	// store private key encrypted by tenant token (data key)
	edgeEncKey, err := keyService.TenantEncrypt(edgeCertResp.Key, &crypto.Token{EncryptedToken: tenant.Token})
	if err != nil {
		return resp, errcode.NewInternalError(err.Error())
	}

	// Create client certificates for mqtt client on the edge
	clientCertResp, err := cfssl.GetCert(tenantID, cfsslModels.CertificatePostParamsTypeClient)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "CreateEdgeCluster: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
		return resp, errcode.NewInternalError(err.Error())
	}
	// store private key encrypted by tenant token (data key)
	clientEncKey, err := keyService.TenantEncrypt(clientCertResp.Key, &crypto.Token{EncryptedToken: tenant.Token})
	if err != nil {
		return resp, errcode.NewInternalError(err.Error())
	}

	edgeCertDBO := EdgeCertDBO{
		EdgeBaseModelDBO: model.EdgeBaseModelDBO{
			BaseModelDBO: model.BaseModelDBO{
				ID:        base.GetUUID(),
				TenantID:  tenantID,
				Version:   doc.Version,
				CreatedAt: doc.CreatedAt,
				UpdatedAt: doc.UpdatedAt,
			},
			EdgeID: doc.ID,
		},
		EdgeCertCore: model.EdgeCertCore{
			Certificate:       edgeCertResp.Cert,
			PrivateKey:        edgeEncKey,
			ClientCertificate: clientCertResp.Cert,
			ClientPrivateKey:  clientEncKey,
			EdgeCertificate:   edgeCertResp.Cert,
			EdgePrivateKey:    edgeEncKey,
			Locked:            false,
		},
	}
	// Creating the edge device and the edge cluster in one transaction
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := tx.NamedExec(context, queryMap["CreateEdgeCluster"], &edgeClusterDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating edge cluster for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		_, err = tx.NamedExec(context, queryMap["CreateEdgeCert"], &edgeCertDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating edge certificate for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = generateAndSetShortIDForEdgeCluster(context, tx, &edgeClusterDBO, maxShortIDAttempts)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context,
				"Error in creating short ID for edge cluster %s and tenant ID %s. Error: %s"),
				doc.ID, tenantID, err.Error(),
			)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		edgeDeviceDBO.FillDefaults()
		_, err = tx.NamedExec(context, queryMap["CreateEdgeDevice"], &edgeDeviceDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating edge device for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.initEdgeInfo(context, tx, doc.ID, now)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating edge info record for edge %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		return dbAPI.createEdgeClusterLabels(context, tx, &edgeClusterDoc)
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	return resp, err
}

// CreateEdgeV2 creates an edge object in the DB
func (dbAPI *dbObjectModelAPI) CreateEdgeV2(context context.Context, i interface{} /* *model.EdgeV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.EdgeV2)
	if !ok {
		return model.CreateDocumentResponse{}, errcode.NewInternalError("CreateEdgeV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.CreateEdge(context, &doc, callback)
}

// CreateEdgeW creates an edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateEdgeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateEdge, &model.Edge{}, w, r, callback)
}

// CreateEdgeWV2 creates an edge object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateEdgeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateEdgeV2), &model.EdgeV2{}, w, r, callback)
}

// UpdateEdge update an edge object in the DB
func (dbAPI *dbObjectModelAPI) UpdateEdge(context context.Context, i interface{} /* *model.Edge*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	// Only allow old edges to update using this api
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Edge)
	if !ok {
		return resp, errcode.NewInternalError("UpdateEdge: type error")
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
	err = model.ValidateEdge(&doc)
	if err != nil {
		return resp, err
	}
	if len(doc.ID) == 0 {
		return resp, errcode.NewBadRequestError("edgeClusterID")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeCluster,
		meta.OperationUpdate,
		auth.RbacContext{
			ID: doc.ID,
		})
	if err != nil {
		return resp, err
	}
	yes, err := dbAPI.isEdgeMultinodeAware(context, p.ID)
	if err != nil {
		return resp, err
	}

	if yes {
		glog.Errorf(base.PrefixRequestID(context, "Error: Unsupported operation for multinode aware edge %s"), p.ID)
		return resp, errcode.NewInternalError("Unsupported operation for multinode aware edge")
	}

	// get current edge cluster to see if category assignment (labels) changed,
	// if so, figure out if any projects update notification needed
	edgeCluster, err := dbAPI.GetEdgeCluster(context, doc.ID)
	if err != nil {
		return resp, err
	}
	labelsChanged := model.IsLabelsChanged(edgeCluster.Labels, doc.Labels)
	deployedProjects := []model.Project{}
	deployedProjectMap := map[string]bool{}
	if labelsChanged {
		projects, err := dbAPI.SelectAllProjects(context, nil)
		if err != nil {
			return resp, err
		}
		for _, project := range projects {
			if funk.Contains(project.EdgeIDs, edgeCluster.ID) {
				deployedProjects = append(deployedProjects, project)
				deployedProjectMap[project.ID] = true
			}
		}
	}
	edgeDevice, err := dbAPI.GetEdgeDevice(context, doc.ID)
	if err != nil {
		return resp, err
	}
	// get edge cert
	edgeCert, err := dbAPI.GetEdgeCertByEdgeID(context, doc.ID)
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

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	doc.Connected = IsEdgeConnected(tenantID, doc.ID)

	edgeClusterDBO := EdgeClusterDBO{}
	edgeClusterDoc := doc.ToEdgeCluster()
	err = base.Convert(&edgeClusterDoc, &edgeClusterDBO)
	if err != nil {
		return resp, err
	}

	edgeDeviceDBO := EdgeDeviceDBO{}
	edgeDeviceDoc := doc.ToEdgeDevice()
	err = base.Convert(&edgeDeviceDoc, &edgeDeviceDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := base.DeleteTxn(context, tx, "edge_label_model", map[string]interface{}{"edge_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting edge labels for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		// note: this method ignores serialNumber update
		_, err = tx.NamedExec(context, queryMap["UpdateEdgeCluster"], &edgeClusterDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "UpdateEdge: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		edgeDeviceDBO.FillDefaults()
		_, err = tx.NamedExec(context, queryMap["UpdateEdgeDevice"], &edgeDeviceDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "UpdateEdgeDevice: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		if labelsChanged {
			// Only deletions are required, new project additions are not required to be updated
			err = dbAPI.deleteInvalidAppEdgeIDsOnProjectEdgeUpdate(context, tx, tenantID, deployedProjects, []model.EdgeClusterIDLabels{
				model.EdgeClusterIDLabels{ID: doc.ID, Labels: doc.Labels},
			})
			if err != nil {
				return err
			}
		}
		return dbAPI.createEdgeClusterLabels(context, tx, &edgeClusterDoc)
	})
	if callback != nil {
		projectsToNotify := []model.Project{}
		if labelsChanged {
			projects, err := dbAPI.SelectAllProjects(context, nil)
			if err != nil {
				return resp, err
			}
			for _, project := range projects {
				projectShouldDeployToEdgeCluster := funk.Contains(project.EdgeIDs, edgeCluster.ID)
				projectIsDeployedToEdgeCluster := deployedProjectMap[project.ID]
				// Notify edgeCluster if project should be deployed to the edgeCluster, or,
				// if project is already deployed on the edge cluster.
				if projectShouldDeployToEdgeCluster || projectIsDeployedToEdgeCluster {
					projectsToNotify = append(projectsToNotify, project)
				}
			}
		}
		msg := model.UpdateEdgeMessage{
			Doc:      doc,
			Projects: projectsToNotify,
		}
		go callback(context, msg)
	}

	resp.ID = p.ID
	return resp, nil
}

// UpdateEdgeV2 update an edge object in the DB
func (dbAPI *dbObjectModelAPI) UpdateEdgeV2(context context.Context, i interface{} /* *model.EdgeV2*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.EdgeV2)
	if !ok {
		return model.UpdateDocumentResponse{}, errcode.NewInternalError("UpdateEdgeV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.UpdateEdge(context, &doc, callback)
}

// UpdateEdgeW update an edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateEdgeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateEdge, &model.Edge{}, w, r, callback)
}

// UpdateEdgeWV2 update an edge object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateEdgeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateEdgeV2), &model.EdgeV2{}, w, r, callback)
}

// DeleteEdge delete a edge object in the DB
func (dbAPI *dbObjectModelAPI) DeleteEdge(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdge,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}

	yes, err := dbAPI.isEdgeMultinodeAware(context, id)
	if err != nil {
		return resp, err
	}

	if yes {
		glog.Errorf(base.PrefixRequestID(context, "Error: Unsupported operation for multinode aware edge %s"), id)
		return resp, errcode.NewInternalError("Unsupported operation for multinode aware edge")
	}
	// old style edge hence delete cluster and device
	return dbAPI.DeleteEdgeCluster(context, id, callback)
}

// DeleteEdgeW delete a edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteEdgeW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteEdge, id, w, callback)
}

// DeleteEdgeWV2 delete a edge object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteEdgeWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteEdge), id, w, callback)
}

// GetEdgeHandle
func (dbAPI *dbObjectModelAPI) GetEdgeHandle(ctx context.Context, edgeID string, payload model.GetHandlePayload) (model.EdgeCert, error) {
	return dbAPI.GetEdgeClusterHandle(ctx, edgeID, payload)
}

func (dbAPI *dbObjectModelAPI) GetEdgeHandleW(context context.Context, edgeID string, w io.Writer, req *http.Request) error {
	return dbAPI.GetEdgeClusterHandleW(context, edgeID, w, req)
}

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

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	funk "github.com/thoas/go-funk"
)

const (
	// entityTypeEdgeCluster is the entity type `edgeCluster`
	entityTypeEdgeCluster = "edgeCluster"
)

var (
	invalidClusterCertData string
)

func init() {
	invalidClusterCertData = "foobar"

	orderByHelper.Setup(entityTypeEdgeCluster, []string{"id", "version", "created_at", "updated_at", "name", "description"})
}

// EdgeClusterDBO is DB object model for edge cluster
type EdgeClusterDBO struct {
	model.BaseModelDBO
	model.EdgeClusterCore
	Description string `json:"description" db:"description"`
	// Hack to allow null values because sqlx scans all the columns
	Connected *bool `json:"connected,omitempty"`
	// Profile of service domain in JSON.
	Profile *types.JSONText `json:"profile" db:"profile"`
	// Environment variables of service domain in JSON.
	Env *types.JSONText `json:"env" db:"env"`
}

// EdgeClusterLabelDBO is DB object model for edge cluster labels
// For now EdgeClusterID is edge_id as the schema has edge_id in the db
type EdgeClusterLabelDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64  `json:"id" db:"id"`
	EdgeClusterID      string `json:"edgeClusterId" db:"edge_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
}

// EdgeClusterIDsParam is for querying edge clusters
type EdgeClusterIDsParam struct {
	TenantID   string   `json:"tenantId" db:"tenant_id"`
	ClusterIDs []string `json:"clusterIds" db:"edge_cluster_ids"`
}

// Helper method to get cluster dbo pointer
func getClusterDBOPtr(dbObjPtr interface{}) *EdgeClusterDBO {
	edgeClusterDBOPtr := dbObjPtr.(*EdgeClusterDBO)
	return edgeClusterDBOPtr
}

// TODO: rework wrt to multinode if needed
func setClusterConnectionStatus(dbObjPtr interface{}) *EdgeClusterDBO {
	edgeClusterDBOPtr := dbObjPtr.(*EdgeClusterDBO)
	status := IsEdgeConnected(edgeClusterDBOPtr.TenantID, edgeClusterDBOPtr.ID)
	edgeClusterDBOPtr.Connected = &status
	return edgeClusterDBOPtr
}

func setClusterConnectionsStatus(edgeClusters []model.EdgeCluster) []model.EdgeCluster {
	if len(edgeClusters) == 0 {
		return edgeClusters
	}
	tenantID := edgeClusters[0].TenantID
	edgeClusterIDs := funk.Map(edgeClusters, func(edgeCluster model.EdgeCluster) string { return edgeCluster.ID }).([]string)
	connectionFlags := GetEdgeConnections(tenantID, edgeClusterIDs...)
	for idx := range edgeClusters {
		edgeCluster := &edgeClusters[idx]
		edgeCluster.Connected = connectionFlags[edgeCluster.ID]
	}
	return edgeClusters
}

//extract the type query param. It is hidden in the API doc
func extractClusterTargetTypeQueryParam(req *http.Request) model.TargetType {
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

func (dbAPI *dbObjectModelAPI) filterEdgeClusters(context context.Context, entities interface{}) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return entities, err
	}
	edgeClusterMap, err := dbAPI.getAffiliatedProjectsEdgeClusterIDsMap(context)
	if err != nil {
		return entities, err
	}
	// always allow edge to get itself
	if ok, edgeClusterID := base.IsEdgeRequest(authContext); ok && edgeClusterID != "" {
		edgeClusterMap[edgeClusterID] = true
	}
	return auth.FilterEntitiesByID(entities, edgeClusterMap), nil
}

// TODO FIXME - make this method generic
func (dbAPI *dbObjectModelAPI) createEdgeClusterLabels(ctx context.Context, tx *base.WrappedTx, edgeCluster *model.EdgeCluster) error {
	for _, categoryInfo := range edgeCluster.Labels {
		// TODO can be optimized here
		categoryValueDBOs, err := dbAPI.getCategoryValueDBOs(ctx, CategoryValueDBO{CategoryID: categoryInfo.ID})
		if err != nil {
			return err
		}
		if len(categoryValueDBOs) == 0 {
			return errcode.NewRecordNotFoundError(categoryInfo.ID)
		}
		valueFound := false
		for _, categoryValueDBO := range categoryValueDBOs {
			if categoryValueDBO.Value == categoryInfo.Value {
				edgeClusterLabelDBO := EdgeClusterLabelDBO{EdgeClusterID: edgeCluster.ID,
					CategoryValueID: categoryValueDBO.ID}
				_, err = tx.NamedExec(ctx, queryMap["CreateEdgeClusterLabel"], &edgeClusterLabelDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error occurred while creating edge label for ID %s. Error: %s"),
						edgeCluster.ID, err.Error())
					return errcode.TranslateDatabaseError(edgeCluster.ID, err)
				}
				valueFound = true
				break
			}
		}
		if !valueFound {
			return errcode.NewRecordNotFoundError(fmt.Sprintf("%s:%s", categoryInfo.ID, categoryInfo.Value))
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) GetEdgeClusterProjects(ctx context.Context, edgeClusterID string) ([]model.Project, error) {
	projects := []model.Project{}
	edgeCluster, err := dbAPI.GetEdgeCluster(ctx, edgeClusterID)
	if err != nil {
		return projects, err
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return projects, err
	}
	// use infra admin auth context here, since otherwise select all projects
	// will use projects in auth context, which is not yet set at this point
	authContextIA := &base.AuthContext{
		TenantID: authContext.TenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"edgeId":      edgeClusterID,
		},
	}
	newContext := context.WithValue(ctx, base.AuthContextKey, authContextIA)
	allProjects, err := dbAPI.SelectAllProjects(newContext, nil)
	if err != nil {
		return projects, err
	}
	for _, project := range allProjects {
		if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			if model.CategoryMatch(edgeCluster.Labels, project.EdgeSelectors) {
				projects = append(projects, project)
			}
		} else {
			if funk.Contains(project.EdgeIDs, edgeClusterID) {
				projects = append(projects, project)
			}
		}
	}
	return projects, nil
}

// Used to allow access to projects to which access has been given after we handed over the JWT token
// for examle the edge has a JWT token and have been given a calim to certain projects...
// if a project is added to the edge, the JWT token will not have that info, hence we use this to update the token..
// edgeDevices does not need it, edgeCluster should need it
func (dbAPI *dbObjectModelAPI) GetEdgeClusterProjectRoles(ctx context.Context, edgeClusterID string) ([]model.ProjectRole, error) {
	projectRoles := []model.ProjectRole{}
	projects, err := dbAPI.GetEdgeClusterProjects(ctx, edgeClusterID)
	if err != nil {
		return projectRoles, err
	}
	for _, project := range projects {
		projectRoles = append(projectRoles, model.ProjectRole{ProjectID: project.ID, Role: model.ProjectRoleAdmin})
	}
	return projectRoles, nil
}

func (dbAPI *dbObjectModelAPI) populateEdgeClustersLabels(context context.Context, edgeClusters []model.EdgeCluster) error {
	if len(edgeClusters) == 0 {
		return nil
	}
	edgeClusterLabelDBOs := []EdgeClusterLabelDBO{}
	clusterIDs := funk.Map(edgeClusters, func(edgeCluster model.EdgeCluster) string { return edgeCluster.ID }).([]string)
	err := dbAPI.QueryIn(context, &edgeClusterLabelDBOs, queryMap["SelectEdgeClustersLabels"], EdgeClusterIDsParam{
		ClusterIDs: clusterIDs,
	})
	if err != nil {
		return err
	}
	edgeClusterLabelsMap := map[string]([]model.CategoryInfo){}
	for _, edgeClusterLabelDBO := range edgeClusterLabelDBOs {
		edgeClusterLabelsMap[edgeClusterLabelDBO.EdgeClusterID] = append(edgeClusterLabelsMap[edgeClusterLabelDBO.EdgeClusterID],
			edgeClusterLabelDBO.CategoryInfo)
	}
	for i := 0; i < len(edgeClusters); i++ {
		edgeCluster := &edgeClusters[i]
		edgeCluster.Labels = edgeClusterLabelsMap[edgeCluster.ID]
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getEdgeClusters(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeCluster, error) {
	edgeClusters := []model.EdgeCluster{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeClusters, err
	}

	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	param := EdgeClusterDBO{BaseModelDBO: tenantModel, EdgeClusterCore: model.EdgeClusterCore{Type: base.StringPtr("")}}

	query, err := buildQuery(entityTypeEdgeCluster, queryMap["SelectEdgeClustersTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return edgeClusters, err
	}
	_, err = dbAPI.NotPagedQuery(context, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		edgeCluster := model.EdgeCluster{}
		err := base.Convert(dbObjPtr, &edgeCluster)
		if err == nil {
			edgeClusters = append(edgeClusters, edgeCluster)
		}
		return nil
	}, query, param)
	if err != nil {
		return edgeClusters, err
	}
	if len(edgeClusters) == 0 {
		return edgeClusters, nil
	}
	edgeClusters = setClusterConnectionsStatus(edgeClusters)
	err = dbAPI.populateEdgeClustersLabels(context, edgeClusters)
	if err != nil {
		return edgeClusters, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterEdgeClusters(context, edgeClusters)
		if err == nil {
			edgeClusters = entities.([]model.EdgeCluster)
		} else {
			glog.Errorf(base.PrefixRequestID(context, "SelectAllEdgesClusters: filter edge clusters failed: %s\n"), err.Error())
		}
	}
	return edgeClusters, err
}

// getClusterIDsreturns cluster IDs (SD, K8s etc) filtered by the type
func (dbAPI *dbObjectModelAPI) getClusterIDs(ctx context.Context, targetType model.TargetType) ([]string, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return []string{}, err
	}
	param := ServiceDomainTypeParam{TenantID: authContext.TenantID, Type: targetType}
	clusterIDs, err := dbAPI.selectEntityIDsByParam(ctx, queryMap["SelectEdgeClusterIDs"], param)
	if err != nil {
		return []string{}, err
	}
	return clusterIDs, nil
}

// getAllClusterTypes returns all cluster irrespective of type.
// Connections can be optionally retrieved using setConnections param
func (dbAPI *dbObjectModelAPI) getAllClusterTypes(context context.Context, setConnections bool) ([]model.EdgeCluster, error) {
	edgeClusters := []model.EdgeCluster{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeClusters, err
	}

	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	param := EdgeClusterDBO{BaseModelDBO: tenantModel}
	clusterDBOs := []EdgeClusterDBO{}
	err = dbAPI.Query(context, &clusterDBOs, queryMap["SelectAllClusters"], param)
	if err != nil {
		return edgeClusters, err
	}
	for i := range clusterDBOs {
		clusterDBO := &clusterDBOs[i]
		cluster := model.EdgeCluster{}
		err = base.Convert(clusterDBO, &cluster)
		if err == nil {
			edgeClusters = append(edgeClusters, cluster)
		}
	}
	if setConnections {
		edgeClusters = setClusterConnectionsStatus(edgeClusters)
	}
	err = dbAPI.populateEdgeClustersLabels(context, edgeClusters)
	if err != nil {
		return edgeClusters, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterEdgeClusters(context, edgeClusters)
		if err == nil {
			edgeClusters = entities.([]model.EdgeCluster)
		} else {
			glog.Errorf(base.PrefixRequestID(context, "getAllClusters: filter edge clusters failed: %s\n"), err.Error())
		}
	}
	return edgeClusters, err
}

func (dbAPI *dbObjectModelAPI) getEdgeClustersCore(context context.Context, edgeClusterIDsInPage []string, queryParam *model.EntitiesQueryParam) ([]model.EdgeCluster, error) {
	edgeClusters := []model.EdgeCluster{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeClusters, err
	}
	if len(edgeClusterIDsInPage) != 0 {
		edgeClusterDBOs := []EdgeClusterDBO{}
		// use in query to find edgeClusterDBOs
		query, err := buildQuery(entityTypeEdgeCluster, queryMap["SelectEdgeClustersInTemplate"], nil, orderByNameID)
		if err != nil {
			return edgeClusters, err
		}
		err = dbAPI.QueryIn(context, &edgeClusterDBOs, query, EdgeClusterIDsParam{
			TenantID:   authContext.TenantID,
			ClusterIDs: edgeClusterIDsInPage,
		})
		if err != nil {
			return edgeClusters, err
		}
		// convert edgeClusterDBO to edgeCluster
		for _, edgeClusterDBO := range edgeClusterDBOs {
			edgeCluster := model.EdgeCluster{}
			err := base.Convert(&edgeClusterDBO, &edgeCluster)
			if err != nil {
				return edgeClusters, err
			}
			edgeClusters = append(edgeClusters, edgeCluster)
		}
		edgeClusters = setClusterConnectionsStatus(edgeClusters)
		// populate edge Clusters labels
		err = dbAPI.populateEdgeClustersLabels(context, edgeClusters)
		if err != nil {
			return edgeClusters, err
		}
	}
	return edgeClusters, nil
}

func (dbAPI *dbObjectModelAPI) getEdgeClustersW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	// get query param from request (PageIndex, PageSize, etc)
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /edgedevices, the target type is always edge for backward compatibility
	targetType := extractClusterTargetTypeQueryParam(req)
	edgeClusterIDs, edgeClusterIDsInPage, err := dbAPI.getServiceDomainIDsInPage(context, projectID, queryParam, targetType)
	if err != nil {
		return err
	}
	edgeClusters, err := dbAPI.getEdgeClustersCore(context, edgeClusterIDsInPage, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: len(edgeClusterIDs), EntityType: entityTypeEdgeCluster})
	r := model.EdgeClusterListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		EdgeClusterList:           edgeClusters,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllEdgeClusters select all edge clusters for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllEdgeClusters(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeCluster, error) {
	return dbAPI.getEdgeClusters(context, entitiesQueryParam)
}

// SelectAllEdgeClustersW select all edge clusters for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeClustersW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeClustersW(context, "", w, req)
}

// SelectAllEdgeClustersForProjectW select all edge clusters for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeClustersForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeClustersW(context, projectID, w, req)
}

// SelectAllEdgeDevicesForClusterW select all edge devices for the given tenant + cluster, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesForClusterW(context context.Context, clusterID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeDevicesW(context, "", clusterID, w, req)
}

// SelectAllEdgeDevicesInfoForClusterW select all edge devices info for the given tenant + cluster, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeDevicesInfoForClusterW(context context.Context, clusterID string, w io.Writer, req *http.Request) error {
	return dbAPI.getEdgeDevicesInfoWV2(context, "", clusterID, w, req)
}

// GetEdgeCluster get an edge cluster object in the DB
func (dbAPI *dbObjectModelAPI) GetEdgeCluster(context context.Context, id string) (model.EdgeCluster, error) {
	edgeCluster := model.EdgeCluster{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeCluster, err
	}
	tenantID := authContext.TenantID
	edgeClusterDBOs := []EdgeClusterDBO{}
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := EdgeClusterDBO{BaseModelDBO: tenantModel, EdgeClusterCore: model.EdgeClusterCore{Type: base.StringPtr("")}}
	if len(id) == 0 {
		return edgeCluster, errcode.NewBadRequestError("edgeClusterID")
	}
	err = dbAPI.Query(context, &edgeClusterDBOs, queryMap["SelectEdgeClusters"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "GetEdgeCluster: DB select failed: %s\n"), err.Error())
		return edgeCluster, err
	}
	if len(edgeClusterDBOs) == 0 {
		return edgeCluster, errcode.NewRecordNotFoundError(id)
	}
	edgeClusterDBOPtr := setClusterConnectionStatus(&edgeClusterDBOs[0])
	err = base.Convert(edgeClusterDBOPtr, &edgeCluster)
	if err != nil {
		return edgeCluster, err
	}
	edgeClusters := []model.EdgeCluster{edgeCluster}
	err = dbAPI.populateEdgeClustersLabels(context, edgeClusters)
	if err != nil {
		return edgeCluster, err
	}

	// filter
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterEdgeClusters(context, edgeClusters)
		if err == nil {
			edgeClusters = entities.([]model.EdgeCluster)
		} else {
			glog.Errorf(base.PrefixRequestID(context, "GetEdgeCluster: filter edgeClusters failed: %s\n"), err.Error())
			return edgeCluster, err
		}
		if len(edgeClusters) == 0 {
			return edgeCluster, errcode.NewRecordNotFoundError(id)
		}
	}
	return edgeClusters[0], nil
}

// GetEdgeClusterW get an edge cluster object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeClusterW(context context.Context, id string, w io.Writer, req *http.Request) error {
	edgeCluster, err := dbAPI.GetEdgeCluster(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edgeCluster)
}

// generateAndSetShortIDForEdgeDevice generates a short ID for an edge device and sets it in the DB
// It tries for the given number of attempts for duplicate errors. In case of other errors,it exits earlier
func generateAndSetShortIDForEdgeCluster(ctx context.Context, tx *base.WrappedTx, edgeClusterDBO *EdgeClusterDBO, numAttempts int) error {
	var err error
	for i := 0; i < numAttempts; i++ {
		shortID := base.GenerateShortID(shortIDLen, shortIDLetters)
		edgeClusterDBO.ShortID = &shortID
		err = namedExec(tx, ctx, queryMap["UpdateEdgeClusterShortId"], &edgeClusterDBO)
		if err == nil {
			break
		}
		edgeClusterDBO.ShortID = nil
		if errcode.IsDuplicateRecordError(err) {
			continue
		}
		break
	}
	return err
}

// CreateEdgeCluster creates an edge cluster object in the DB
func (dbAPI *dbObjectModelAPI) CreateEdgeCluster(context context.Context, i interface{} /* *model.EdgeCluster */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.EdgeCluster)
	if !ok {
		return resp, errcode.NewInternalError("CreateEdgeCluster: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateEdgeCluster doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateEdgeCluster(&doc)
	if err != nil {
		return resp, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeCluster,
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
	edgeClusterDBO := EdgeClusterDBO{}
	err = base.Convert(&doc, &edgeClusterDBO)
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
		return dbAPI.createEdgeClusterLabels(context, tx, &doc)
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	return resp, err
}

// CreateEdgeClusterW creates an edge cluster object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateEdgeClusterW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateEdgeCluster), &model.EdgeCluster{}, w, r, callback)
}

// UpdateEdgeCluster update an edge cluster object in the DB
func (dbAPI *dbObjectModelAPI) UpdateEdgeCluster(context context.Context, i interface{} /* *model.EdgeCluster*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.EdgeCluster)
	if !ok {
		return resp, errcode.NewInternalError("UpdateEdgeCluster: type error")
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
	err = model.ValidateEdgeCluster(&doc)
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

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	doc.Connected = IsEdgeConnected(tenantID, doc.ID)
	edgeClusterDBO := EdgeClusterDBO{}
	err = base.Convert(&doc, &edgeClusterDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err = validateVirtualIP(context, tx, edgeClusterDBO.ID, edgeClusterDBO.VirtualIP, false)
		if err != nil {
			return err
		}
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
		return dbAPI.createEdgeClusterLabels(context, tx, &doc)
	})
	if err != nil {
		return resp, err
	}

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
		// TODO: Change for backward compat
		//msg := model.UpdateEdgeMessage{
		msg := model.UpdateEdgeClusterMessage{
			Doc:      doc,
			Projects: projectsToNotify,
		}
		go callback(context, msg)
	}

	resp.ID = doc.ID
	return resp, nil
}

// UpdateEdgeClusterW update an edge cluster object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateEdgeClusterW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateEdgeCluster), &model.EdgeCluster{}, w, r, callback)
}

// DeleteEdgeCluster delete an edge cluster object in the DB
func (dbAPI *dbObjectModelAPI) DeleteEdgeCluster(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityEdgeCluster,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	doc := model.EdgeCluster{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}
	return DeleteEntity(context, dbAPI, "edge_cluster_model", "id", id, doc, callback)
}

// DeleteEdgeClusterW delete a edge cluster object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteEdgeClusterW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteEdgeCluster), id, w, callback)
}

// GetEdgeHandle
func (dbAPI *dbObjectModelAPI) GetEdgeClusterHandle(ctx context.Context, edgeClusterID string, payload model.GetHandlePayload) (model.EdgeCert, error) {
	edgeCert := model.EdgeCert{}
	// ctx is passed without auth context
	authContext := &base.AuthContext{
		TenantID: payload.TenantID,
	}
	newCtx := context.WithValue(ctx, base.AuthContextKey, authContext)
	tenant, err := dbAPI.GetTenant(newCtx, payload.TenantID)
	if err != nil {
		return edgeCert, errcode.NewBadRequestExError("tenantID", fmt.Sprintf("Tenant not found, tenantId=%s", payload.TenantID))
	}
	if false == crypto.MatchHashAndPassword(payload.Token, edgeClusterID) {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get token for edge ID %s"), edgeClusterID)
		return edgeCert, errcode.NewBadRequestExError("token", fmt.Sprintf("Bad token, edgeId=%s", edgeClusterID))
	}
	edgeCert2, err := dbAPI.GetEdgeCertByEdgeID(newCtx, edgeClusterID)
	if err != nil {
		return edgeCert, errcode.NewBadRequestExError("edgeClusterID", fmt.Sprintf("Edge cluster cert not found, edgeClusterID=%s", edgeClusterID))
	}
	if edgeCert2.Locked {
		glog.Errorf(base.PrefixRequestID(ctx, "Certificate for edge cluster  ID %s is already locked"), edgeClusterID)
		return edgeCert, errcode.NewBadRequestExError("edgeClusterID", fmt.Sprintf("Edge cluster cert locked, edgeClusterID=%s", edgeClusterID))
	}
	// Decrypt the private key generated using fixed root CA.
	key := ""
	token := &crypto.Token{EncryptedToken: tenant.Token}
	if edgeCert2.PrivateKey != invalidClusterCertData {
		key, err = keyService.TenantDecrypt(edgeCert2.PrivateKey, token)
		if err != nil {
			return edgeCert, errcode.NewInternalError(err.Error())
		}
	}
	// Decrypt the private key generated using per-tenant root CA.
	edgeKey, err := keyService.TenantDecrypt(edgeCert2.EdgePrivateKey, token)
	if err != nil {
		return edgeCert, errcode.NewInternalError(err.Error())
	}
	clientKey, err := keyService.TenantDecrypt(edgeCert2.ClientPrivateKey, token)
	if err != nil {
		return edgeCert, errcode.NewInternalError(err.Error())
	}
	// update DB to mark the cert as locked
	edgeCert2.Locked = true
	_, err = dbAPI.UpdateEdgeCert(newCtx, &edgeCert2, nil)
	if err != nil {
		return edgeCert, err
	}
	// return unencrypted key
	edgeCert2.PrivateKey = key
	edgeCert2.EdgePrivateKey = edgeKey
	edgeCert2.ClientPrivateKey = clientKey
	return edgeCert2, nil
}

func (dbAPI *dbObjectModelAPI) GetEdgeClusterHandleW(context context.Context, edgeClusterID string, w io.Writer, req *http.Request) error {
	payload := model.GetHandlePayload{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &payload)
	if err != nil {
		return errcode.NewBadRequestError("Payload")
	}
	edgeClusterCert, err := dbAPI.GetEdgeClusterHandle(context, edgeClusterID, payload)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edgeClusterCert)
}

//
func (dbAPI *dbObjectModelAPI) SelectEdgeClusterIDLabels(context context.Context) ([]model.EdgeClusterIDLabels, error) {
	resp := []model.EdgeClusterIDLabels{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	edgeLabelList := []model.EdgeClusterIDLabel{}
	query := fmt.Sprintf(queryMap["SelectEdgeClusterIDLabelsTemplate"], authContext.TenantID)
	err = dbAPI.Query(context, &edgeLabelList, query, struct{}{})
	if err != nil {
		return resp, err
	}
	edgeLabelsMap := map[string]*model.EdgeClusterIDLabels{}
	for _, edgeLabel := range edgeLabelList {
		edgeLabels, ok := edgeLabelsMap[edgeLabel.ID]
		if ok {
			edgeLabels.Labels = append(edgeLabels.Labels, edgeLabel.CategoryInfo)
		} else {
			edgeLabelsMap[edgeLabel.ID] = &model.EdgeClusterIDLabels{ID: edgeLabel.ID, Labels: []model.CategoryInfo{edgeLabel.CategoryInfo}}
		}
	}
	for _, edgeLabels := range edgeLabelsMap {
		resp = append(resp, *edgeLabels)
	}
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) SelectAllEdgeClusterIDs(context context.Context) ([]string, error) {
	resp := []string{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	idDBOList := []IDDBO{}
	query := fmt.Sprintf(queryMap["SelectAllEdgeClusterIDsTemplate"], tenantID)
	err = dbAPI.Query(context, &idDBOList, query, struct{}{})
	if err != nil {
		return resp, err
	}
	for _, idDBO := range idDBOList {
		resp = append(resp, idDBO.ID)
	}
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) SelectConnectedEdgeClusterIDs(context context.Context) ([]string, error) {
	ids, err := dbAPI.SelectAllEdgeClusterIDs(context)
	if err != nil {
		return ids, err
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return ids, err
	}
	tenantID := authContext.TenantID
	connMap := GetEdgeConnections(tenantID, ids...)
	return funk.Filter(ids, func(id string) bool {
		return connMap[id]
	}).([]string), nil
}

type getEdgeClusterDeviceIDsParam struct {
	TenantID      string `db:"tenant_id"`
	EdgeClusterID string `db:"edge_cluster_id"`
}
type getEdgeClusterDeviceIDsResult struct {
	ID string `db:"id"`
}

func (dbAPI *dbObjectModelAPI) getEdgeClusterDeviceIDs(context context.Context, edgeClusterID string) ([]string, error) {
	ids := []string{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return ids, err
	}
	tenantID := authContext.TenantID
	param := getEdgeClusterDeviceIDsParam{TenantID: tenantID, EdgeClusterID: edgeClusterID}
	results := []getEdgeClusterDeviceIDsResult{}
	err = dbAPI.Query(context, &results, queryMap["GetEdgeClusterDeviceIDs"], param)
	if err != nil {
		return ids, err
	}
	ids = funk.Map(results, func(x getEdgeClusterDeviceIDsResult) string { return x.ID }).([]string)
	return ids, nil
}

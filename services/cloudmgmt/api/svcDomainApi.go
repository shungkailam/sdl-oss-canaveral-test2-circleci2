package api

import (
	"cloudservices/cloudmgmt/cfssl"
	cfsslModels "cloudservices/cloudmgmt/generated/cfssl/models"
	"cloudservices/common/apptemplate"
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
	// entityTypeServiceDomain is the entity type `edgeCluster`
	entityTypeServiceDomain = "serviceDomain"
	// shortIDLen is the length of the generated short ID in bytes
	shortIDLen = 6

	// shortIDLetters is the alphabet allowed for generating short ID
	shortIDLetters = "abcdefghijklmnopqrstuvwxyz0123456789"

	// maxShortIDAttempts is the maximum number of attempts for generating non-conflicting short ID's
	maxShortIDAttempts = 10
)

var (
	invalidServiceDomainCertData string
)

func init() {
	// queryMap["SelectEdgeClusters"] = `SELECT * FROM edge_cluster_model WHERE ((tenant_id = :tenant_id) AND (:id = '' OR id = :id)) AND (:type = '' OR type = :type OR (:type = 'EDGE' AND type is null))`
	// Kubernetes clusters are not returned by default if type is empty but get by ID returns it irrespective of the type
	queryMap["SelectEdgeClusters"] = `SELECT * FROM edge_cluster_model WHERE tenant_id = :tenant_id AND (id = :id OR (:id = '' AND ((:type = '' AND (type is null OR type != '` + string(model.KubernetesClusterTargetType) + `')) OR type = :type OR (:type = '` + string(model.RealTargetType) + `' AND type is null))))`

	// queryMap["SelectEdgeClustersTemplate"] = `SELECT * FROM edge_cluster_model WHERE ((tenant_id = :tenant_id) AND (:id = '' OR id = :id) AND (:type = '' OR type = :type OR (:type = 'EDGE' AND type is null))) %s`
	// Kubernetes clusters are not returned by default if type is empty but get by ID returns it irrespective of the type
	queryMap["SelectEdgeClustersTemplate"] = `SELECT * FROM edge_cluster_model WHERE ((tenant_id = :tenant_id) AND (id = :id OR (:id = '' AND ((:type = '' AND (type is null OR type != '` + string(model.KubernetesClusterTargetType) + `')) OR type = :type OR (:type = '` + string(model.RealTargetType) + `' AND type is null))))) %s`

	// All clusters irrespective of type - real, cloud, k8s
	queryMap["SelectAllClusters"] = `SELECT * FROM edge_cluster_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id)`

	queryMap["SelectEdgeClustersInTemplate"] = `SELECT * FROM edge_cluster_model WHERE tenant_id = :tenant_id AND (id IN (:edge_cluster_ids)) %s`

	queryMap["SelectEdgeClustersLabels"] = `SELECT edge_label_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
		FROM edge_label_model JOIN category_value_model ON edge_label_model.category_value_id = category_value_model.id WHERE edge_label_model.edge_id IN (:edge_cluster_ids)`
	queryMap["SelectEdgeClusterIDLabelsTemplate"] = `SELECT e.edge_id, c.category_id as id, c.value from edge_label_model e JOIN category_value_model c on e.category_value_id = c.id where e.edge_id in (select id from edge_cluster_model where tenant_id = '%s')`
	queryMap["CreateEdgeClusterLabel"] = `INSERT INTO edge_label_model (edge_id, category_value_id) VALUES (:edge_id, :category_value_id)`

	queryMap["CreateEdgeCluster"] = `INSERT INTO edge_cluster_model (id, version, tenant_id, name, description, short_id, connected, type, virtual_ip, profile, env, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :short_id, :connected, :type, :virtual_ip, :profile, :env, :created_at, :updated_at)`
	queryMap["UpdateEdgeCluster"] = `UPDATE edge_cluster_model SET version = :version, name = :name, description = :description, short_id = :short_id, connected = :connected, virtual_ip = :virtual_ip, profile = :profile, env = :env, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	// Kubernetes clusters are not returned by default if the type is empty
	// Query to fetch cluster IDs with a particular type. If the type is not specified, all IDs (CLOUD and normal edge/SD) except for those with k8s type are returned
	queryMap["SelectEdgeClusterIDs"] = `SELECT id from edge_cluster_model where tenant_id = :tenant_id AND ((:type = '' AND (type is null OR type != '` + string(model.KubernetesClusterTargetType) + `')) OR type = :type OR (:type = '` + string(model.RealTargetType) + `' AND type is NULL))`

	queryMap["UpdateEdgeClusterShortId"] = `UPDATE edge_cluster_model SET short_id = :short_id WHERE tenant_id = :tenant_id AND id = :id`

	queryMap["SelectAllEdgeClusterIDsTemplate"] = `SELECT id from edge_cluster_model where tenant_id = '%s'`

	queryMap["GetEdgeClusterDeviceIDs"] = `SELECT id from edge_device_model WHERE tenant_id = :tenant_id AND edge_cluster_id = :edge_cluster_id`

	orderByHelper.Setup(entityTypeServiceDomain, []string{"id", "version", "created_at", "updated_at", "name", "description"})
}

// ServiceDomainDBO is DB object model for service domain
type ServiceDomainDBO struct {
	model.BaseModelDBO
	model.ServiceDomainCore
	Description string `json:"description" db:"description"`
	// Hack to allow null values because sqlx scans all the columns
	Connected *bool `json:"connected,omitempty"`
	// Profile of service domain in JSON.
	Profile *types.JSONText `json:"profile" db:"profile"`
	// Environment variables of service domain in JSON.
	Env *types.JSONText `json:"env" db:"env"`
}

// ServiceDomainLabelDBO is DB object model for service domain labels
// For now SvcDomainID is edge_id as the schema has edge_id in the db
type ServiceDomainLabelDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64  `json:"id" db:"id"`
	SvcDomainID        string `json:"svcDomainId" db:"edge_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
}

// ServiceDomainIDsParam is for querying service domains
type ServiceDomainIDsParam struct {
	TenantID     string   `json:"tenantId" db:"tenant_id"`
	SvcDomainIDs []string `json:"svcDomainIds" db:"edge_cluster_ids"`
}

type ServiceDomainTypeParam struct {
	TenantID    string           `json:"tenantId" db:"tenant_id"`
	SvcDomainID string           `json:"svcDomainId" db:"edge_cluster_id"`
	Type        model.TargetType `json:"type" db:"type"`
}

func setServiceDomainConnectionStatus(dbObjPtr interface{}) *ServiceDomainDBO {
	svcDomainDBOPtr := dbObjPtr.(*ServiceDomainDBO)
	status := IsEdgeConnected(svcDomainDBOPtr.TenantID, svcDomainDBOPtr.ID)
	svcDomainDBOPtr.Connected = &status
	return svcDomainDBOPtr
}

func setServiceDomainsConnectionStatus(svcDomains []model.ServiceDomain) []model.ServiceDomain {
	if len(svcDomains) == 0 {
		return svcDomains
	}
	tenantID := svcDomains[0].TenantID
	svcDomainIDs := funk.Map(svcDomains, func(svcDomain model.ServiceDomain) string { return svcDomain.ID }).([]string)
	connectionFlags := GetEdgeConnections(tenantID, svcDomainIDs...)
	for idx := range svcDomains {
		svcDomain := &svcDomains[idx]
		svcDomain.Connected = connectionFlags[svcDomain.ID]
	}
	return svcDomains
}

//extract the type query param. It is hidden in the API doc
func extractServiceDomainTargetTypeQueryParam(req *http.Request) model.TargetType {
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

func (dbAPI *dbObjectModelAPI) filterServiceDomains(ctx context.Context, entities interface{}) (interface{}, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return entities, err
	}
	// Edge cluster ID is service domain ID
	svcDomainMap, err := dbAPI.getAffiliatedProjectsEdgeClusterIDsMap(ctx)
	if err != nil {
		return entities, err
	}
	// always allow service domain to get itself
	if ok, svcDomainID := base.IsEdgeRequest(authContext); ok && svcDomainID != "" {
		svcDomainMap[svcDomainID] = true
	}
	return auth.FilterEntitiesByID(entities, svcDomainMap), nil
}

// TODO FIXME - make this method generic
func (dbAPI *dbObjectModelAPI) createServiceDomainLabels(ctx context.Context, tx *base.WrappedTx, svcDomain *model.ServiceDomain) error {
	for _, categoryInfo := range svcDomain.Labels {
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
				svcDomainLabelDBO := ServiceDomainLabelDBO{SvcDomainID: svcDomain.ID,
					CategoryValueID: categoryValueDBO.ID}
				_, err = tx.NamedExec(ctx, queryMap["CreateEdgeClusterLabel"], &svcDomainLabelDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error occurred while creating service domain label for ID %s. Error: %s"),
						svcDomain.ID, err.Error())
					return errcode.TranslateDatabaseError(svcDomain.ID, err)
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

func (dbAPI *dbObjectModelAPI) GetServiceDomainProjects(ctx context.Context, svcDomainID string) ([]model.Project, error) {
	projects := []model.Project{}
	svcDomain, err := dbAPI.GetServiceDomain(ctx, svcDomainID)
	if err != nil {
		return projects, err
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return projects, err
	}
	// use infra admin auth context here, since otherwise select all projects
	// will use projects in auth ctx, which is not yet set at this point
	authContextIA := &base.AuthContext{
		TenantID: authContext.TenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"edgeId":      svcDomainID,
		},
	}
	newContext := context.WithValue(ctx, base.AuthContextKey, authContextIA)
	allProjects, err := dbAPI.SelectAllProjects(newContext, nil)
	if err != nil {
		return projects, err
	}
	for _, project := range allProjects {
		if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			if model.CategoryMatch(svcDomain.Labels, project.EdgeSelectors) {
				projects = append(projects, project)
			}
		} else {
			if funk.Contains(project.EdgeIDs, svcDomainID) {
				projects = append(projects, project)
			}
		}
	}
	return projects, nil
}

// Used to allow access to projects to which access has been given after we handed over the JWT token
// for example the service domain has a JWT token and have been given a calim to certain projects...
// if a project is added to the service domain, the JWT token will not have that info, hence we use this to update the token..
// nodes do not need it, a service domain should need it
func (dbAPI *dbObjectModelAPI) GetServiceDomainProjectRoles(ctx context.Context, svcDomainID string) ([]model.ProjectRole, error) {
	projectRoles := []model.ProjectRole{}
	projects, err := dbAPI.GetServiceDomainProjects(ctx, svcDomainID)
	if err != nil {
		return projectRoles, err
	}
	for _, project := range projects {
		projectRoles = append(projectRoles, model.ProjectRole{ProjectID: project.ID, Role: model.ProjectRoleAdmin})
	}
	return projectRoles, nil
}

func (dbAPI *dbObjectModelAPI) populateServiceDomainLabels(ctx context.Context, svcDomains []model.ServiceDomain) error {
	if len(svcDomains) == 0 {
		return nil
	}
	svcDomainLabelDBOs := []ServiceDomainLabelDBO{}
	svcDomainIDs := funk.Map(svcDomains, func(svcDomain model.ServiceDomain) string { return svcDomain.ID }).([]string)
	err := dbAPI.QueryIn(ctx, &svcDomainLabelDBOs, queryMap["SelectEdgeClustersLabels"], ServiceDomainIDsParam{
		SvcDomainIDs: svcDomainIDs,
	})
	if err != nil {
		return err
	}
	svcDomainLabelsMap := map[string]([]model.CategoryInfo){}
	for _, svcDomainLabelDBO := range svcDomainLabelDBOs {
		svcDomainLabelsMap[svcDomainLabelDBO.SvcDomainID] = append(svcDomainLabelsMap[svcDomainLabelDBO.SvcDomainID],
			svcDomainLabelDBO.CategoryInfo)
	}
	for i := 0; i < len(svcDomains); i++ {
		svcDomain := &svcDomains[i]
		svcDomain.Labels = svcDomainLabelsMap[svcDomain.ID]
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getServiceDomains(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ServiceDomain, error) {
	svcDomains := []model.ServiceDomain{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcDomains, err
	}

	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	param := ServiceDomainDBO{BaseModelDBO: tenantModel, ServiceDomainCore: model.ServiceDomainCore{Type: base.StringPtr("")}}

	query, err := buildQuery(entityTypeServiceDomain, queryMap["SelectEdgeClustersTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return svcDomains, err
	}
	_, err = dbAPI.NotPagedQuery(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		svcDomain := model.ServiceDomain{}
		err := base.Convert(dbObjPtr, &svcDomain)
		if err == nil {
			svcDomains = append(svcDomains, svcDomain)
		}
		return nil
	}, query, param)
	if err != nil {
		return svcDomains, err
	}
	if len(svcDomains) == 0 {
		return svcDomains, nil
	}
	svcDomains = setServiceDomainsConnectionStatus(svcDomains)
	err = dbAPI.populateServiceDomainLabels(ctx, svcDomains)
	if err != nil {
		return svcDomains, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterServiceDomains(ctx, svcDomains)
		if err == nil {
			svcDomains = entities.([]model.ServiceDomain)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "getServiceDomains: filter service domains failed: %s\n"), err.Error())
		}
		// don't return Env if not infra admin or same edge
		currSvcDomainID := ""
		if ok, svcDomainID := base.IsEdgeRequest(authContext); ok && svcDomainID != "" {
			currSvcDomainID = svcDomainID
		}
		for i := range svcDomains {
			if svcDomains[i].ID != currSvcDomainID {
				svcDomains[i].Env = apptemplate.RedactEnvs(svcDomains[i].Env)
			}
		}
	}
	return svcDomains, err
}

func (dbAPI *dbObjectModelAPI) getServiceDomainIDsInPage(ctx context.Context, projectID string, queryParam *model.EntitiesQueryParam, targetType model.TargetType) ([]string, []string, error) {
	// Get all cluster IDs with the type
	clusterIDs, err := dbAPI.getClusterIDs(ctx, targetType)
	if err != nil {
		return []string{}, []string{}, err
	}
	clusterIDsMap := map[string]struct{}{}
	for _, clusterID := range clusterIDs {
		clusterIDsMap[clusterID] = struct{}{}
	}
	return dbAPI.GetEntityIDsInPage(ctx, projectID, "", queryParam, func(ctx context.Context, svcDomainEntity *model.ServiceDomainEntityModelDBO, queryParam *model.EntitiesQueryParam) ([]string, error) {
		if svcDomainEntity.SvcDomainID == "" {
			return clusterIDs, nil
		}
		if _, ok := clusterIDsMap[svcDomainEntity.SvcDomainID]; ok {
			return []string{svcDomainEntity.SvcDomainID}, nil
		}
		return []string{}, nil
	})
}

func (dbAPI *dbObjectModelAPI) getServiceDomainsCore(ctx context.Context, svcDomainIDsInPage []string, queryParam *model.EntitiesQueryParam) ([]model.ServiceDomain, error) {
	svcDomains := []model.ServiceDomain{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcDomains, err
	}
	if len(svcDomainIDsInPage) != 0 {
		svcDomainDBOs := []ServiceDomainDBO{}
		// use in query to find svcDomainDBOs
		query, err := buildQuery(entityTypeServiceDomain, queryMap["SelectEdgeClustersInTemplate"], nil, orderByNameID)
		if err != nil {
			return svcDomains, err
		}
		err = dbAPI.QueryIn(ctx, &svcDomainDBOs, query, ServiceDomainIDsParam{
			TenantID:     authContext.TenantID,
			SvcDomainIDs: svcDomainIDsInPage,
		})
		if err != nil {
			return svcDomains, err
		}
		// convert ServiceDomainDBO to service domain
		for _, ServiceDomainDBO := range svcDomainDBOs {
			svcDomain := model.ServiceDomain{}
			err := base.Convert(&ServiceDomainDBO, &svcDomain)
			if err != nil {
				return svcDomains, err
			}
			svcDomains = append(svcDomains, svcDomain)
		}

		svcDomains = setServiceDomainsConnectionStatus(svcDomains)
		// populate service domain labels
		err = dbAPI.populateServiceDomainLabels(ctx, svcDomains)
		if err != nil {
			return svcDomains, err
		}
		// don't return Env if not infra admin or same edge
		if !auth.IsInfraAdminRole(authContext) {
			currSvcDomainID := ""
			if ok, svcDomainID := base.IsEdgeRequest(authContext); ok && svcDomainID != "" {
				currSvcDomainID = svcDomainID
			}
			for i := range svcDomains {
				if svcDomains[i].ID != currSvcDomainID {
					svcDomains[i].Env = apptemplate.RedactEnvs(svcDomains[i].Env)
				}
			}
		}
	}
	return svcDomains, nil
}

func (dbAPI *dbObjectModelAPI) getServiceDomainsW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	// get query param from request (PageIndex, PageSize, etc)
	queryParam := model.GetEntitiesQueryParam(req)
	// get the target type. For /servicedomains, the target type is always edge for backward compatibility
	targetType := extractServiceDomainTargetTypeQueryParam(req)
	svcDomainIDs, svcDomainIDsInPage, err := dbAPI.getServiceDomainIDsInPage(ctx, projectID, queryParam, targetType)
	if err != nil {
		return err
	}
	svcDomains, err := dbAPI.getServiceDomainsCore(ctx, svcDomainIDsInPage, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: len(svcDomainIDs), EntityType: entityTypeServiceDomain})
	r := model.ServiceDomainListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		SvcDomainList:             svcDomains,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllServiceDomains selects all service domains for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllServiceDomains(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ServiceDomain, error) {
	return dbAPI.getServiceDomains(ctx, entitiesQueryParam)
}

// SelectAllServiceDomainsW selects all service domains for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllServiceDomainsW(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getServiceDomainsW(ctx, "", w, req)
}

// SelectAllServiceDomainsForProjectW select all service domains for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllServiceDomainsForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getServiceDomainsW(ctx, projectID, w, req)
}

// SelectAllNodesForServiceDomainW select all nodes for the given tenant + service domain, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodesForServiceDomainW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	return dbAPI.getNodesW(ctx, "", svcDomainID, w, req)
}

// SelectAllNodeInfoForServiceDomainW selects all nodes info for the given tenant + service domain, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllNodeInfoForServiceDomainW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	return dbAPI.getNodesInfoWV2(ctx, "", svcDomainID, w, req)
}

// GetServiceDomain gets a service domain from the DB
func (dbAPI *dbObjectModelAPI) GetServiceDomain(ctx context.Context, id string) (model.ServiceDomain, error) {
	svcDomain := model.ServiceDomain{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcDomain, err
	}
	tenantID := authContext.TenantID
	svcDomainDBOs := []ServiceDomainDBO{}
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := ServiceDomainDBO{BaseModelDBO: tenantModel, ServiceDomainCore: model.ServiceDomainCore{Type: base.StringPtr("")}}
	if len(id) == 0 {
		return svcDomain, errcode.NewBadRequestError("svcDomainID")
	}
	err = dbAPI.Query(ctx, &svcDomainDBOs, queryMap["SelectEdgeClusters"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetServiceDomain: DB select failed for %s. Error: %s"), id, err.Error())
		return svcDomain, err
	}
	if len(svcDomainDBOs) == 0 {
		return svcDomain, errcode.NewRecordNotFoundError(id)
	}
	svcDomainDBOPtr := setServiceDomainConnectionStatus(&svcDomainDBOs[0])
	err = base.Convert(svcDomainDBOPtr, &svcDomain)
	if err != nil {
		return svcDomain, err
	}
	svcDomains := []model.ServiceDomain{svcDomain}
	err = dbAPI.populateServiceDomainLabels(ctx, svcDomains)
	if err != nil {
		return svcDomain, err
	}
	// filter
	if !auth.IsInfraAdminRole(authContext) {
		entities, err := dbAPI.filterServiceDomains(ctx, svcDomains)
		if err == nil {
			svcDomains = entities.([]model.ServiceDomain)
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, "GetServiceDomain: filter svcDomains failed for %s. Error: %s"), id, err.Error())
			return svcDomain, err
		}
		if len(svcDomains) == 0 {
			return svcDomain, errcode.NewRecordNotFoundError(id)
		}
		// don't return Env if not infra admin or same edge
		if ok, svcDomainID := base.IsEdgeRequest(authContext); !ok || svcDomainID != svcDomains[0].ID {
			svcDomains[0].Env = apptemplate.RedactEnvs(svcDomains[0].Env)
		}
	}
	return svcDomains[0], nil
}

// GetServiceDomainW gets a service domain from the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetServiceDomainW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	svcDomain, err := dbAPI.GetServiceDomain(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, svcDomain)
}

// GetServiceDomainEffectiveProfileW gets a service domain effective profile from the DB, write output into writer
// Service domain effective profile is AND of its profile with Tenant profile
func (dbAPI *dbObjectModelAPI) GetServiceDomainEffectiveProfileW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	svcDomain, err := dbAPI.GetServiceDomain(ctx, id)
	if err != nil {
		return err
	}
	sdProfile := svcDomain.Profile
	if sdProfile == nil {
		return json.NewEncoder(w).Encode(nil)
	}
	tenant, err := dbAPI.GetTenant(ctx, svcDomain.TenantID)
	if err != nil {
		return err
	}
	tntProfile := tenant.Profile
	if tntProfile == nil {
		sdProfile.EnableSSH = false
		sdProfile.Privileged = false
	} else {
		sdProfile.EnableSSH = tntProfile.EnableSSH && sdProfile.EnableSSH
		sdProfile.Privileged = tntProfile.Privileged && sdProfile.Privileged
		if sdProfile.EnableSSH {
			// ensure the service domain version supports the ssh feature
			sdProfile.EnableSSH = false
			nodesInfo, _, err := dbAPI.getNodesInfoV2(ctx, "", svcDomain.ID, nil)
			if err == nil {
				var nodeVersion *string
				for _, nodeInfo := range nodesInfo {
					nodeVersion = nodeInfo.NodeVersion
					if nodeInfo.Onboarded {
						break
					}
				}
				if nodeVersion != nil {
					fts, err := GetFeaturesForVersion(*nodeVersion)
					if err == nil && fts.RemoteSSH {
						sdProfile.EnableSSH = true
					}
				}
			}
		}
	}
	return base.DispatchPayload(w, sdProfile)
}

// generateAndSetShortIDForServiceDomain generates a short ID for a service domain and sets it in the DB
// It tries for the given number of attempts for duplicate errors. In case of other errors,it exits earlier
func generateAndSetShortIDForServiceDomain(ctx context.Context, tx *base.WrappedTx, svcDomainDBO *ServiceDomainDBO, numAttempts int) error {
	var err error
	for i := 0; i < numAttempts; i++ {
		shortID := base.GenerateShortID(shortIDLen, shortIDLetters)
		svcDomainDBO.ShortID = &shortID
		err = namedExec(tx, ctx, queryMap["UpdateEdgeClusterShortId"], &svcDomainDBO)
		if err == nil {
			break
		}
		svcDomainDBO.ShortID = nil
		if errcode.IsDuplicateRecordError(err) {
			continue
		}
		break
	}
	return err
}

func compareServiceDomainTypes(expected, input *string) error {
	if expected == nil {
		expected = base.StringPtr(string(model.RealTargetType))
	}
	if input == nil {
		input = base.StringPtr(string(model.RealTargetType))
	}
	if *expected != *input {
		glog.Errorf("Expected type: %s, found type: %s", *expected, *input)
		return errcode.NewBadRequestError("type")
	}
	return nil
}

// createServiceDomainWithTxnCallback creates a service domain in the DB and invokes the txnCallback before the transaction is committed
func (dbAPI *dbObjectModelAPI) createServiceDomainWithTxnCallback(ctx context.Context, i interface{} /* *model.ServiceDomain */, txnCallback func(*base.WrappedTx, *model.ServiceDomain) error, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ServiceDomain)
	if !ok {
		return resp, errcode.NewInternalError("CreateServiceDomain: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if !base.CheckID(doc.ID) {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(ctx, "CreateServiceDomain doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateServiceDomain(&doc)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityServiceDomain,
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
	svcDomainDBO := ServiceDomainDBO{}
	err = base.Convert(&doc, &svcDomainDBO)
	if err != nil {
		return resp, err
	}
	// first get tenant token
	tenant, err := dbAPI.GetTenant(ctx, tenantID)
	if err != nil {
		return resp, err
	}

	// Create edge/service domain certificates using per-tenant root CA.
	edgeCertResp, err := cfssl.GetCert(tenantID, cfsslModels.CertificatePostParamsTypeServer)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "CreateServiceDomain: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
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
		glog.Errorf(base.PrefixRequestID(ctx, "CreateServiceDomain: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
		return resp, errcode.NewInternalError(err.Error())
	}
	// store private key encrypted by tenant token (data key)
	clientEncKey, err := keyService.TenantEncrypt(clientCertResp.Key, &crypto.Token{EncryptedToken: tenant.Token})
	if err != nil {
		return resp, errcode.NewInternalError(err.Error())
	}

	// service domain cert is still EdgeCert in other APIs
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
		_, err := tx.NamedExec(ctx, queryMap["CreateEdgeCluster"], &svcDomainDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating service domain with ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		_, err = tx.NamedExec(ctx, queryMap["CreateEdgeCert"], &edgeCertDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating certificate for service domain %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = generateAndSetShortIDForServiceDomain(ctx, tx, &svcDomainDBO, maxShortIDAttempts)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx,
				"Error in creating short ID for service domain %s and tenant ID %s. Error: %s"),
				doc.ID, tenantID, err.Error(),
			)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.initServiceDomainInfo(ctx, tx, svcDomainDBO.ID, now)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating service domain info for %s. Error: %s"), svcDomainDBO.ID, err.Error())
			return err
		}
		err = dbAPI.createServiceDomainLabels(ctx, tx, &doc)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating service domain info for %s. Error: %s"), svcDomainDBO.ID, err.Error())
			return err
		}
		if txnCallback != nil {
			err = txnCallback(tx, &doc)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in creating service domain info for %s. Error: %s"), svcDomainDBO.ID, err.Error())
				return err
			}
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addServiceDomainAuditLog(dbAPI, ctx, doc, CREATE)
	return resp, err
}

func (dbAPI *dbObjectModelAPI) CreateServiceDomain(ctx context.Context, i interface{} /* *model.ServiceDomain */, callback func(context.Context, interface{}) error) (interface{}, error) {
	return dbAPI.createServiceDomainWithTxnCallback(ctx, i, nil, callback)
}

// CreateServiceDomainW creates a service domain in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateServiceDomainW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, model.ToCreateV2(dbAPI.CreateServiceDomain), &model.ServiceDomain{}, w, r, callback)
}

// updateServiceDomainWithTxnCallback updates a service domain in the DB and invokes the txnCallback before the transaction is committed
func (dbAPI *dbObjectModelAPI) updateServiceDomainWithTxnCallback(ctx context.Context, p *model.ServiceDomain, txnCallback func(*base.WrappedTx, *model.ServiceDomain) error, callback func(context.Context, interface{}) error) (interface{}, error) {
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
	err = model.ValidateServiceDomain(&doc)
	if err != nil {
		return resp, err
	}
	if len(doc.ID) == 0 {
		return resp, errcode.NewBadRequestError("svcDomainID")
	}
	entityType := meta.EntityServiceDomain
	if doc.Type != nil && *doc.Type == string(model.KubernetesClusterTargetType) {
		entityType = meta.EntityKubernetesCluster
	}
	err = auth.CheckRBAC(
		authContext,
		entityType,
		meta.OperationUpdate,
		auth.RbacContext{
			ID: doc.ID,
		})
	if err != nil {
		return resp, err
	}
	// get current service domain to see if category assignment (labels) changed,
	// if so, figure out if any projects update notification needed
	svcDomain, err := dbAPI.GetServiceDomain(ctx, doc.ID)
	if err != nil {
		return resp, err
	}
	err = compareServiceDomainTypes(svcDomain.Type, doc.Type)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in type comparision for Service Domain %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	labelsChanged := model.IsLabelsChanged(svcDomain.Labels, doc.Labels)
	deployedProjects := []model.Project{}
	deployedProjectMap := map[string]bool{}
	if labelsChanged {
		projects, err := dbAPI.SelectAllProjects(ctx, nil)
		if err != nil {
			return resp, err
		}
		for _, project := range projects {
			if funk.Contains(project.EdgeIDs, svcDomain.ID) {
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
	svcDomainDBO := ServiceDomainDBO{}
	err = base.Convert(&doc, &svcDomainDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err = validateVirtualIP(ctx, tx, svcDomainDBO.ID, svcDomainDBO.VirtualIP, false)
		if err != nil {
			return err
		}
		_, err := base.DeleteTxn(ctx, tx, "edge_label_model", map[string]interface{}{"edge_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting service domain labels for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		// note: this method ignores serialNumber update
		_, err = tx.NamedExec(ctx, queryMap["UpdateEdgeCluster"], &svcDomainDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "UpdateServiceDomain: DB exec failed: %s, tenantID: %s, doc: %+v\n"), err.Error(), tenantID, doc)
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		if labelsChanged {
			// Only deletions are required, new project additions are not required to be updated
			err = dbAPI.deleteInvalidAppEdgeIDsOnProjectEdgeUpdate(ctx, tx, tenantID, deployedProjects, []model.EdgeClusterIDLabels{
				model.EdgeClusterIDLabels{ID: doc.ID, Labels: doc.Labels},
			})
			if err != nil {
				return err
			}
		}
		err = dbAPI.createServiceDomainLabels(ctx, tx, &doc)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating updating domain info for %s. Error: %s"), svcDomainDBO.ID, err.Error())
			return nil
		}
		if txnCallback != nil {
			err = txnCallback(tx, &doc)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in creating updating domain info for %s. Error: %s"), svcDomainDBO.ID, err.Error())
				return err
			}
		}
		return nil
	})
	if err != nil {
		return resp, err
	}

	if callback != nil {
		projectsToNotify := []model.Project{}
		if labelsChanged {
			projects, err := dbAPI.SelectAllProjects(ctx, nil)
			if err != nil {
				return resp, err
			}
			for _, project := range projects {
				// Notify service domain if project is already deployed on the service domain
				// or project should be deployed to it
				if deployedProjectMap[project.ID] || funk.Contains(project.EdgeIDs, svcDomain.ID) {
					projectsToNotify = append(projectsToNotify, project)
				}
			}
		}

		msg := model.UpdateServiceDomainMessage{
			Doc:      doc,
			Projects: projectsToNotify,
		}
		go callback(ctx, msg)
	}

	resp.ID = doc.ID
	GetAuditlogHandler().addServiceDomainAuditLog(dbAPI, ctx, doc, UPDATE)
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) UpdateServiceDomain(ctx context.Context, i interface{} /* *model.ServiceDomain*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	p, ok := i.(*model.ServiceDomain)
	if !ok {
		return resp, errcode.NewInternalError("UpdateServiceDomain: type error")
	}
	if p.Type != nil && *p.Type == string(model.KubernetesClusterTargetType) {
		// Not updatable as kubernetes clusters are updated via dedicated APIs
		return resp, errcode.NewBadRequestExError("type", "Cannot update KubernetesCluster")
	}
	return dbAPI.updateServiceDomainWithTxnCallback(ctx, p, nil, callback)
}

// UpdateServiceDomainW updates a service domain in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateServiceDomainW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, model.ToUpdateV2(dbAPI.UpdateServiceDomain), &model.ServiceDomain{}, w, r, callback)
}

// DeleteServiceDomain deletes a service domain in the DB
func (dbAPI *dbObjectModelAPI) DeleteServiceDomain(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	svcDomain, errGetSvcDomain := dbAPI.GetServiceDomain(ctx, id)
	if errGetSvcDomain != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domain %s. Error: %s"), id, errGetSvcDomain.Error())
		return resp, errGetSvcDomain
	}
	doc := model.ServiceDomain{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}
	entityType := meta.EntityServiceDomain
	if svcDomain.Type != nil && *svcDomain.Type == string(model.KubernetesClusterTargetType) {
		entityType = meta.EntityKubernetesCluster
	}
	err = auth.CheckRBAC(
		authContext,
		entityType,
		meta.OperationDelete,
		auth.RbacContext{ID: svcDomain.ID})
	if err != nil {
		return resp, err
	}
	result, err := DeleteEntity(ctx, dbAPI, "edge_cluster_model", "id", id, doc, callback)
	if err == nil {
		GetAuditlogHandler().addServiceDomainAuditLog(dbAPI, ctx, svcDomain, DELETE)
	}
	return result, err
}

// DeleteServiceDomainW deletes a service domain in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteServiceDomainW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, model.ToDeleteV2(dbAPI.DeleteServiceDomain), id, w, callback)
}

// GetServiceDomainHandle
func (dbAPI *dbObjectModelAPI) GetServiceDomainHandle(ctx context.Context, svcDomainID string, payload model.GetHandlePayload) (model.EdgeCert, error) {
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
	if false == crypto.MatchHashAndPassword(payload.Token, svcDomainID) {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get token for service domain %s"), svcDomainID)
		return edgeCert, errcode.NewBadRequestExError("token", fmt.Sprintf("Bad token, svcDomainId=%s", svcDomainID))
	}
	edgeCert2, err := dbAPI.GetEdgeCertByEdgeID(newCtx, svcDomainID)
	if err != nil {
		return edgeCert, errcode.NewBadRequestExError("svcDomainID", fmt.Sprintf("Service domain cert not found, svcDomainID=%s", svcDomainID))
	}
	if edgeCert2.Locked {
		glog.Errorf(base.PrefixRequestID(ctx, "Certificate for service domain %s is already locked"), svcDomainID)
		return edgeCert, errcode.NewBadRequestExError("svcDomainID", fmt.Sprintf("Service domain cert locked, svcDomainID=%s", svcDomainID))
	}
	// Decrypt the private key generated using fixed root CA.
	key := ""
	token := &crypto.Token{EncryptedToken: tenant.Token}
	if edgeCert2.PrivateKey != invalidServiceDomainCertData {
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

func (dbAPI *dbObjectModelAPI) GetServiceDomainHandleW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	payload := model.GetHandlePayload{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &payload)
	if err != nil {
		return errcode.NewBadRequestError("Payload")
	}
	svcDomainCert, err := dbAPI.GetServiceDomainHandle(ctx, svcDomainID, payload)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, svcDomainCert)
}

//
func (dbAPI *dbObjectModelAPI) SelectServiceDomainIDLabels(ctx context.Context) ([]model.ServiceDomainIDLabels, error) {
	resp := []model.ServiceDomainIDLabels{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	svcDomainLabelList := []model.ServiceDomainIDLabel{}
	query := fmt.Sprintf(queryMap["SelectEdgeClusterIDLabelsTemplate"], authContext.TenantID)
	err = dbAPI.Query(ctx, &svcDomainLabelList, query, struct{}{})
	if err != nil {
		return resp, err
	}
	svcDomainLabelMap := map[string]*model.ServiceDomainIDLabels{}
	for _, svcDomainLabel := range svcDomainLabelList {
		svcDomainLabels, ok := svcDomainLabelMap[svcDomainLabel.ID]
		if ok {
			svcDomainLabels.Labels = append(svcDomainLabels.Labels, svcDomainLabel.CategoryInfo)
		} else {
			svcDomainLabelMap[svcDomainLabel.ID] = &model.ServiceDomainIDLabels{ID: svcDomainLabel.ID, Labels: []model.CategoryInfo{svcDomainLabel.CategoryInfo}}
		}
	}
	for _, svcDomainLabels := range svcDomainLabelMap {
		resp = append(resp, *svcDomainLabels)
	}
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) SelectAllServiceDomainIDs(ctx context.Context) ([]string, error) {
	resp := []string{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	idDBOList := []IDDBO{}
	query := fmt.Sprintf(queryMap["SelectAllEdgeClusterIDsTemplate"], tenantID)
	err = dbAPI.Query(ctx, &idDBOList, query, struct{}{})
	if err != nil {
		return resp, err
	}
	for _, idDBO := range idDBOList {
		resp = append(resp, idDBO.ID)
	}
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) SelectConnectedServiceDomainIDs(ctx context.Context) ([]string, error) {
	ids, err := dbAPI.SelectAllServiceDomainIDs(ctx)
	if err != nil {
		return ids, err
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return ids, err
	}
	tenantID := authContext.TenantID
	connMap := GetEdgeConnections(tenantID, ids...)
	return funk.Filter(ids, func(id string) bool {
		return connMap[id]
	}).([]string), nil
}

func (dbAPI *dbObjectModelAPI) getServiceDomainNodeIDs(ctx context.Context, svcDomainID string) ([]string, error) {
	ids := []string{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return ids, err
	}
	tenantID := authContext.TenantID
	param := NodeIDsParam{TenantID: tenantID, SvcDomainID: svcDomainID}
	results := []IDDBO{}
	err = dbAPI.Query(ctx, &results, queryMap["GetEdgeClusterDeviceIDs"], param)
	if err != nil {
		return ids, err
	}
	ids = funk.Map(results, func(x IDDBO) string { return x.ID }).([]string)
	return ids, nil
}

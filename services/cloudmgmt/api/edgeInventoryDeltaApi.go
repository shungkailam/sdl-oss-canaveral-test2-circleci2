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
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

const (
	// currently edge does not track categories,
	// thus the delta is always fixed and can be big,
	// so as a performance optimization, don't send this
	SEND_CATEGORIES_DELTA = false
)

// The purpose of this module is to provide more efficient mechanism for each
// Edge to sync with cloud. The mechanism is delta based in that each Edge will
// send a summary of its current inventory (object type, id, updatedAt for each
// entity). The cloud will use this info to figure out the delta to send to the
// Edge in the response. The delta will contain IDs for entities to delete as
// well as full metadata for entities to create and update - all in one API
// call. The GetEdgeInventoryDelta method is meant to be called by EVERY Edge
// periodically (say, once every 5 minutes), thus it needs to be as efficient
// and scalable as possible. To this end, the implementation first uses custom
// query to retrieve only (id, updatedAt) EntityVersionMetadata to determine the
// (deleted, created, updated) entities list and only then to do full fetch of
// metadata for entities in the created and updated list.

func init() {
	// Use EID prefix for Edge Inventory Delta to avoid collision
	queryMap["EID_GetCategoriesMetadata"] =
		"select id, updated_at from category_model where tenant_id = '%s'"
	queryMap["EID_GetDataSourcesMetadata"] =
		"select id, updated_at from data_source_model where tenant_id = '%s' and " +
			"edge_id = '%s'"
	queryMap["EID_GetCloudProfilesMetadata"] =
		"select id, updated_at from cloud_creds_model where tenant_id = '%s' " +
			"and ( " +
			"id in (select distinct cloud_creds_id from project_cloud_creds_model where " +
			"project_id in ('%s')) " +
			"OR " +
			"id in (select distinct cloud_creds_id from edge_log_collect_model where " +
			"tenant_id = '%s' and id in ('%s')) " +
			")"
	queryMap["EID_GetContainerRegistriesMetadata"] =
		"select id, updated_at from docker_profile_model where tenant_id = '%s' and " +
			"id in (select distinct docker_profile_id from " +
			"project_docker_profile_model where project_id in ('%s'))"
	queryMap["EID_GetDataStreamsMetadata"] =
		"select id, updated_at from data_stream_model where tenant_id = '%s' and " +
			"project_id in ('%s')"
	queryMap["EID_GetScriptsMetadata"] =
		"select id, updated_at from script_model where tenant_id = '%s' and " +
			"(project_id in ('%s') or project_id is NULL)"
	queryMap["EID_GetProjectServicesMetadata"] =
		"select id, updated_at from project_service_model where tenant_id = '%s' and " +
			"(project_id in ('%s') or project_id is NULL)"
	queryMap["EID_GetServiceInstancesMetadata"] =
		"select id, updated_at from service_instance_model where tenant_id = '%s' and " +
			"(svc_domain_scope_id = '%s' or project_scope_id in ('%s'))"
	queryMap["EID_GetServiceBindingsMetadata"] =
		"select id, updated_at from service_binding_model where tenant_id = '%s' and " +
			"(resource_type is NULL or svc_domain_resource_id = '%s' or project_resource_id in ('%s'))"
	queryMap["EID_GetLogCollectorsMetadata"] =
		"select id, updated_at from edge_log_collect_model where tenant_id = '%s' and " +
			"(project_id in ('%s') or project_id is NULL)"
	queryMap["EID_GetScriptRuntimesMetadata"] =
		"select id, updated_at from script_runtime_model where tenant_id = '%s' and " +
			"(project_id in ('%s') or project_id is NULL)"
	queryMap["EID_GetMLModelsMetadata"] =
		"select id, updated_at from machine_inference_model where tenant_id = '%s' " +
			"and project_id in ('%s')"
	queryMap["EID_GetExplicitApplicationsForEdgeMetadata"] =
		"select id, updated_at from application_model where tenant_id = '%s' and id in " +
			"(select application_id from application_edge_model where edge_id = '%s') " +
			"and project_id in ('%s')"
	queryMap["EID_GetCategoryApplicationsMetadata"] =
		"select id, updated_at from application_model where tenant_id = '%s' and " +
			"project_id in ('%s')"
	queryMap["EID_GetDataDriverInstanceMetadata"] =
		"select id, updated_at from data_driver_instance_model where tenant_id = '%s' and " +
			"project_id in ('%s')"
	queryMap["EID_GetCategoryApplicationsSelectors"] =
		"select t.id, t.updated_at, v.category_id, v.value from application_model as t " +
			"inner join application_edge_selector_model as u on t.id = u.application_id " +
			"inner join category_value_model as v on u.category_value_id = v.id where " +
			"t.tenant_id = '%s' and t.project_id in ('%s')"
	queryMap["EID_GetExplicitProjectsForEdge"] =
		"select id, updated_at from project_model where tenant_id = '%s' and " +
			"edge_selector_type = 'Explicit' and id in " +
			"(select project_id from project_edge_model where edge_id = '%s')"
	queryMap["EID_GetCategoryProjects"] =
		"select id, updated_at from project_model where tenant_id = '%s' and " +
			"edge_selector_type = 'Category'"
	queryMap["EID_GetCategoryProjectsSelectors"] =
		"select t.project_id as id, u.category_id, u.value, v.updated_at from " +
			"project_edge_selector_model as t inner join category_value_model as u on " +
			"t.category_value_id = u.id inner join project_model as v on " +
			"t.project_id = v.id where v.tenant_id = '%s' and " +
			"v.edge_selector_type = 'Category'"
	queryMap["EID_GetEdgeLabels"] =
		"select u.category_id as id, value from edge_label_model as t inner join " +
			"category_value_model as u on t.category_value_id = u.id where edge_id = '%s'"
}

//// Common
// execute sql query to get EntityVersionMetadataList
func (dbAPI *dbObjectModelAPI) getEntityVersionMetadataList(
	ctx context.Context, query string) ([]model.EntityVersionMetadata, error) {
	results := []model.EntityVersionMetadata{}
	err := dbAPI.Queryx(ctx, &results, query, struct{}{})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// get tenant and edge IDs from the context
// return error if context is not edge context.
func (dbAPI *dbObjectModelAPI) getTenantAndEdgeIDs(
	ctx context.Context) (tenantID string, edgeID string, err error) {
	var authContext *base.AuthContext
	var ok bool
	authContext, err = base.GetAuthContext(ctx)
	if err != nil {
		return
	}
	tenantID = authContext.TenantID
	ok, edgeID = base.IsEdgeRequest(authContext)
	if !ok || edgeID == "" {
		err = errcode.NewBadRequestError("edgeID")
		return
	}
	return
}

// getMatchingEntities get entities matching edgeLabels This is used for
// category based edge selection where entities could be projects or
// applications. Entity matches edgeLabels if the entity does not appear in
// entitiesSelectors (no category assignment) or if its entitySelectors match
// edgeLabels.
func getMatchingEntities(
	entities []model.EntityVersionMetadata,
	entitiesSelectors []*model.EntityCategoryInfoMetadata,
	edgeLabels []model.CategoryInfo,
	includeEmpty bool) []model.EntityVersionMetadata {
	results := []model.EntityVersionMetadata{}
	entitySelectorMap := make(map[string]*model.EntityCategoryInfoMetadata,
		len(entitiesSelectors))
	for _, p := range entitiesSelectors {
		entitySelectorMap[p.ID] = p
	}
	for _, entity := range entities {
		p := entitySelectorMap[entity.ID]
		if p == nil {
			if includeEmpty {
				results = append(results, entity)
			}
		} else if model.CategoryMatch(edgeLabels, p.CategoryInfo) {
			results = append(results, model.EntityVersionMetadata{
				ID:        p.ID,
				UpdatedAt: p.UpdatedAt,
			})
		}
	}
	return results
}

// makeEntityCategoryInfoMetadataList takes a list of EntityCategorySelectorInfo
// and groups them based on entity ID to return a list of
// EntityCategorySelectorInfo.
func makeEntityCategoryInfoMetadataList(
	entityCategorySelectorInfoList []model.EntityCategorySelectorInfo,
) []*model.EntityCategoryInfoMetadata {
	pm := make(map[string]*model.EntityCategoryInfoMetadata,
		len(entityCategorySelectorInfoList))
	results := []*model.EntityCategoryInfoMetadata{}
	for _, entityCategorySelectorInfo := range entityCategorySelectorInfoList {
		p := pm[entityCategorySelectorInfo.ID]
		if p == nil {
			p = &model.EntityCategoryInfoMetadata{
				EntityVersionMetadata: model.EntityVersionMetadata{
					ID:        entityCategorySelectorInfo.ID,
					UpdatedAt: entityCategorySelectorInfo.UpdatedAt,
				},
				CategoryInfo: nil,
			}
			pm[entityCategorySelectorInfo.ID] = p
			results = append(results, p)
		}
		p.CategoryInfo = append(p.CategoryInfo, model.CategoryInfo{
			ID:    entityCategorySelectorInfo.CategoryID,
			Value: entityCategorySelectorInfo.Value,
		})
	}
	return results
}

//// Global entities
// Categories
// getCategoriesMetadata get list of EntityVersionMetadata from all categories.
func (dbAPI *dbObjectModelAPI) getCategoriesMetadata(
	ctx context.Context, tenantID string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetCategoriesMetadata"], tenantID)
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

//// Per-Edge entities
// Data Sources
// getDataSourcesMetadata get list of EntityVersionMetadata for all data sources
// belonging to the given edge.
func (dbAPI *dbObjectModelAPI) getDataSourcesMetadata(ctx context.Context,
	tenantID string, edgeID string) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetDataSourcesMetadata"], tenantID, edgeID)
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

//// Project-Associated entities
// Cloud Profiles
// getCloudProfilesMetadata get list of EntityVersionMetadata for all cloud
// profiles associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getCloudProfilesMetadata(
	ctx context.Context, tenantID string, projectIDs []string, logcollectorIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetCloudProfilesMetadata"],
		tenantID, strings.Join(projectIDs, "', '"), tenantID, strings.Join(logcollectorIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Container Registries
// getContainerRegistriesMetadata get list of EntityVersionMetadata for all
// container registry profiles associated with the projects with the given
// project IDs
func (dbAPI *dbObjectModelAPI) getContainerRegistriesMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetContainerRegistriesMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

//// Project-Scoped entities
// DataStreams
// getDataStreamsMetadata get list of EntityVersionMetadata for all data streams
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getDataStreamsMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetDataStreamsMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Scripts
// getScriptsMetadata get list of EntityVersionMetadata for all scripts
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getScriptsMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetScriptsMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// ProjectServices
// getProjectServicesMetadata get list of EntityVersionMetadata for all project services
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getProjectServicesMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetProjectServicesMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// ServiceInstances
// getServiceInstancesMetadata get list of EntityVersionMetadata for all service instances
// for the tenant, edge or project IDs
func (dbAPI *dbObjectModelAPI) getServiceInstancesMetadata(
	ctx context.Context, tenantID, edgeID string, projectIDs []string) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetServiceInstancesMetadata"],
		tenantID, edgeID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// ServiceBindings
// getServiceBindingsMetadata get list of EntityVersionMetadata for all service instances
// for the tenant, edge or project IDs
func (dbAPI *dbObjectModelAPI) getServiceBindingsMetadata(
	ctx context.Context, tenantID, edgeID string, projectIDs []string) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetServiceBindingsMetadata"],
		tenantID, edgeID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// LogCollectors
// getLogCollectorsMetadata get list of EntityVersionMetadata for all log collectors
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getLogCollectorsMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetLogCollectorsMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// DataDriverInstance
// getLogCollectorsMetadata get list of EntityVersionMetadata for all data driver instances
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getDataDriverInstanceMetadata(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetDataDriverInstanceMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// ScriptRuntimes
// getScriptsMetadata get list of EntityVersionMetadata for all scripts runtimes
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getScriptRuntimesMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetScriptRuntimesMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// ML Models
// getScriptsMetadata get list of EntityVersionMetadata for all ML models
// associated with the projects with the given project IDs
func (dbAPI *dbObjectModelAPI) getMLModelsMetadata(
	ctx context.Context, tenantID string, projectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetMLModelsMetadata"],
		tenantID, strings.Join(projectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Applications
// Applications - Explicit
// getExplicitApplicationsForEdgeMetadata get list of EntityVersionMetadata for
// all applications containing the given edge and are contained in the given
// explicit projects (projects with edge selector type = 'Explicit' ).
func (dbAPI *dbObjectModelAPI) getExplicitApplicationsForEdgeMetadata(
	ctx context.Context, tenantID string, edgeID string,
	explicitProjectIDs []string) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetExplicitApplicationsForEdgeMetadata"],
		tenantID, edgeID, strings.Join(explicitProjectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Applications - Category
// getCategoryApplicationsMetadata get list of EntityVersionMetadata for all
// applications which are contained in the given category projects (projects
// with edge selector type = 'Category' ).
func (dbAPI *dbObjectModelAPI) getCategoryApplicationsMetadata(
	ctx context.Context, tenantID string, categoryProjectIDs []string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetCategoryApplicationsMetadata"],
		tenantID, strings.Join(categoryProjectIDs, "', '"))
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Applications - Category
// getCategoryApplicationsSelectors get list of EntityCategoryInfoMetadata for
// all applications which are contained in the given category projects (projects
// with edge selector type = 'Category' ).
func (dbAPI *dbObjectModelAPI) getCategoryApplicationsSelectors(
	ctx context.Context, tenantID string, categoryProjectIDs []string,
) ([]*model.EntityCategoryInfoMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetCategoryApplicationsSelectors"],
		tenantID, strings.Join(categoryProjectIDs, "', '"))
	categoryApplicationSelectorInfoList := []model.EntityCategorySelectorInfo{}
	err :=
		dbAPI.Query(ctx, &categoryApplicationSelectorInfoList, query, struct{}{})
	if err != nil {
		return nil, err
	}
	results :=
		makeEntityCategoryInfoMetadataList(categoryApplicationSelectorInfoList)
	return results, nil
}

//// Projects
// Explicit Projects
// getExplicitProjectsForEdge get list of EntityVersionMetadata for all explicit
// projects containing the given edge
func (dbAPI *dbObjectModelAPI) getExplicitProjectsForEdge(ctx context.Context,
	tenantID string, edgeID string) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetExplicitProjectsForEdge"],
		tenantID, edgeID)
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Category Projects
// getCategoryProjects get list of EntityVersionMetadata for all category
// projects
func (dbAPI *dbObjectModelAPI) getCategoryProjects(
	ctx context.Context, tenantID string,
) ([]model.EntityVersionMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetCategoryProjects"], tenantID)
	return dbAPI.getEntityVersionMetadataList(ctx, query)
}

// Category Projects
// getCategoryProjectsSelectors get list of EntityCategoryInfoMetadata for all
// category projects
func (dbAPI *dbObjectModelAPI) getCategoryProjectsSelectors(
	ctx context.Context, tenantID string,
) ([]*model.EntityCategoryInfoMetadata, error) {
	query := fmt.Sprintf(queryMap["EID_GetCategoryProjectsSelectors"], tenantID)
	categoryProjectSelectorInfoList := []model.EntityCategorySelectorInfo{}
	err :=
		dbAPI.Query(ctx, &categoryProjectSelectorInfoList, query, struct{}{})
	if err != nil {
		return nil, err
	}
	results :=
		makeEntityCategoryInfoMetadataList(categoryProjectSelectorInfoList)
	return results, nil
}

// getEdgeLabels get list of edge labels (CategoryInfo) for the given edge.
func (dbAPI *dbObjectModelAPI) getEdgeLabels(ctx context.Context,
	edgeID string) ([]model.CategoryInfo, error) {
	query := fmt.Sprintf(queryMap["EID_GetEdgeLabels"], edgeID)
	results := []model.CategoryInfo{}
	err := dbAPI.Query(ctx, &results, query, struct{}{})
	if err != nil {
		return nil, err
	}
	return results, nil
}

// makeBooleanMap makes a map[string]bool where all key in keys are mapped to
// true
func makeBooleanMap(keys []string) map[string]bool {
	m := make(map[string]bool, len(keys))
	for _, key := range keys {
		m[key] = true
	}
	return m
}

// getCategoriesByIDsMerge merge two getCategoriesByIDs calls into one
func (dbAPI *dbObjectModelAPI) getCategoriesByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.Category, []model.Category, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getCategoriesByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getCategoriesByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getCategoriesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.Category, 0, n)
	r2 := make([]model.Category, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getDataStreamsByIDsMerge merge two getDataStreamsByIDs calls into one
func (dbAPI *dbObjectModelAPI) getDataStreamsByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.DataStream, []model.DataStream, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getDataStreamsByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getDataStreamsByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getDataStreamsByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.DataStream, 0, n)
	r2 := make([]model.DataStream, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getScriptsByIDsMerge merge two getScriptsByIDs calls into one
func (dbAPI *dbObjectModelAPI) getScriptsByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.Script, []model.Script, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getScriptsByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getScriptsByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getScriptsByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.Script, 0, n)
	r2 := make([]model.Script, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getProjectServicesByIDsMerge merge two getProjectServicesByIDs calls into one
func (dbAPI *dbObjectModelAPI) getProjectServicesByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.ProjectService, []model.ProjectService, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getProjectServicesByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getProjectServicesByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getProjectServicesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.ProjectService, 0, n)
	r2 := make([]model.ProjectService, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getServiceInstancesByIDsMerge merge two getServiceInstancesByIDs calls into one
func (dbAPI *dbObjectModelAPI) getServiceInstancesByIDsMerge(
	ctx context.Context, edgeID string, IDs []string, IDs2 []string,
) ([]model.ServiceInstance, []model.ServiceInstance, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		svcInstances, err := dbAPI.getServiceInstancesByIDs(ctx, edgeID, IDs2)
		return nil, svcInstances, err
	} else if n2 == 0 {
		svcInstances, err := dbAPI.getServiceInstancesByIDs(ctx, edgeID, IDs)
		return svcInstances, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getServiceInstancesByIDs(ctx, edgeID, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.ServiceInstance, 0, n)
	r2 := make([]model.ServiceInstance, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getServiceBindingsByIDsMerge merge two getServiceBindingsByIDs calls into one
func (dbAPI *dbObjectModelAPI) getServiceBindingsByIDsMerge(
	ctx context.Context, edgeID string, IDs []string, IDs2 []string,
) ([]model.ServiceBinding, []model.ServiceBinding, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		svcBindings, err := dbAPI.getServiceBindingsByIDs(ctx, edgeID, IDs2)
		return nil, svcBindings, err
	} else if n2 == 0 {
		svcBindings, err := dbAPI.getServiceBindingsByIDs(ctx, edgeID, IDs)
		return svcBindings, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getServiceBindingsByIDs(ctx, edgeID, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.ServiceBinding, 0, n)
	r2 := make([]model.ServiceBinding, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getScriptRuntimesByIDsMerge merge two getScriptRuntimesByIDs calls into one
func (dbAPI *dbObjectModelAPI) getScriptRuntimesByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.ScriptRuntime, []model.ScriptRuntime, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getScriptRuntimesByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getScriptRuntimesByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getScriptRuntimesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.ScriptRuntime, 0, n)
	r2 := make([]model.ScriptRuntime, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getMLModelsByIDsMerge merge two getMLModelsByIDs calls into one
func (dbAPI *dbObjectModelAPI) getMLModelsByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.MLModel, []model.MLModel, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getMLModelsByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getMLModelsByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getMLModelsByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.MLModel, 0, n)
	r2 := make([]model.MLModel, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getCloudProfilesByIDsMerge merge two getCloudProfilesByIDs calls into one
func (dbAPI *dbObjectModelAPI) getCloudProfilesByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.CloudCreds, []model.CloudCreds, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getCloudProfilesByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getCloudProfilesByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getCloudProfilesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.CloudCreds, 0, n)
	r2 := make([]model.CloudCreds, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getContainerRegistriesByIDsMerge merge two SelectContainerRegistriesByIDs
// calls into one
func (dbAPI *dbObjectModelAPI) getContainerRegistriesByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.ContainerRegistry, []model.ContainerRegistry, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.SelectContainerRegistriesByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.SelectContainerRegistriesByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.SelectContainerRegistriesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.ContainerRegistry, 0, n)
	r2 := make([]model.ContainerRegistry, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getDataSourcesByIDsMerge merge two getDataSourcesByIDs calls into one
func (dbAPI *dbObjectModelAPI) getDataSourcesByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.DataSource, []model.DataSource, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getDataSourcesByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getDataSourcesByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getDataSourcesByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.DataSource, 0, n)
	r2 := make([]model.DataSource, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getApplicationsByIDsMerge merge two getApplicationsByIDs calls into one
func (dbAPI *dbObjectModelAPI) getApplicationsByIDsMerge(
	ctx context.Context, IDs []string, IDs2 []string,
) ([]model.Application, []model.Application, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getApplicationsByIDs(ctx, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getApplicationsByIDs(ctx, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getApplicationsByIDs(ctx, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.Application, 0, n)
	r2 := make([]model.Application, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getProjectsByIDsMerge merge two getProjectsByIDs calls into one
func (dbAPI *dbObjectModelAPI) getProjectsByIDsMerge(
	ctx context.Context, tenantID string, IDs []string, IDs2 []string,
) ([]model.Project, []model.Project, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getProjectsByIDs(ctx, tenantID, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getProjectsByIDs(ctx, tenantID, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	cats, err := dbAPI.getProjectsByIDs(ctx, tenantID, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.Project, 0, n)
	r2 := make([]model.Project, 0, n2)
	for _, c := range cats {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getLogCollectorsByIDsMerge merge two getLogCollectorsByIds calls into one
func (dbAPI *dbObjectModelAPI) getLogCollectorsByIDsMerge(
	ctx context.Context, tenantID string, IDs []string, IDs2 []string,
) ([]model.LogCollector, []model.LogCollector, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		cats, err := dbAPI.getLogCollectorsByIds(ctx, tenantID, IDs2)
		return nil, cats, err
	} else if n2 == 0 {
		cats, err := dbAPI.getLogCollectorsByIds(ctx, tenantID, IDs)
		return cats, nil, err
	}
	idsMap := makeBooleanMap(IDs)
	allIDs := append(IDs, IDs2...)
	lcs, err := dbAPI.getLogCollectorsByIds(ctx, tenantID, allIDs)
	if err != nil {
		return nil, nil, err
	}
	r1 := make([]model.LogCollector, 0, n)
	r2 := make([]model.LogCollector, 0, n2)
	for _, c := range lcs {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getDataDriverInstancesByIDsMerge merge two getDataDriverInstanceInventoryByIds calls into one
func (dbAPI *dbObjectModelAPI) getDataDriverInstancesByIDsMerge(
	ctx context.Context, tenantID string, IDs []string, IDs2 []string,
) ([]model.DataDriverInstanceInventory, []model.DataDriverInstanceInventory, error) {
	n := len(IDs)
	n2 := len(IDs2)
	if n == 0 && n2 == 0 {
		return nil, nil, nil
	} else if n == 0 {
		ddis, err := dbAPI.getDataDriverInstanceInventoryByIds(ctx, tenantID, IDs2)
		return nil, ddis, err
	} else if n2 == 0 {
		ddis, err := dbAPI.getDataDriverInstanceInventoryByIds(ctx, tenantID, IDs)
		return ddis, nil, err
	}
	allIDs := append(IDs, IDs2...)
	idsMap := makeBooleanMap(IDs)
	ddis, err := dbAPI.getDataDriverInstanceInventoryByIds(ctx, tenantID, allIDs)
	if err != nil {
		return nil, nil, err
	}

	r1 := make([]model.DataDriverInstanceInventory, 0, n)
	r2 := make([]model.DataDriverInstanceInventory, 0, n2)
	for _, c := range ddis {
		if idsMap[c.ID] {
			r1 = append(r1, c)
		} else {
			r2 = append(r2, c)
		}
	}
	return r1, r2, nil
}

// getEdgeInventoryDeltaCategories helper method to fill in categories in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaCategories(
	ctx context.Context, tenantID string, edgeID string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	cats, err := dbAPI.getCategoriesMetadata(ctx, tenantID)
	if err != nil {
		return err
	}
	catCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.Categories, cats)
	result.Deleted.Categories = catCI.Deleted.GetIDs()
	ccs, ucs, err :=
		dbAPI.getCategoriesByIDsMerge(ctx,
			catCI.Created.GetIDs(), catCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.Categories, result.Updated.Categories = ccs, ucs
	return nil
}

// getEdgeInventoryDeltaDataStreams helper method to fill in DataStreams in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaDataStreams(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	dstreams, err :=
		dbAPI.getDataStreamsMetadata(ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	dstreamCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.DataPipelines, dstreams)
	result.Deleted.DataPipelines = dstreamCI.Deleted.GetIDs()
	cdstreams, udstreams, err :=
		dbAPI.getDataStreamsByIDsMerge(ctx, dstreamCI.Created.GetIDs(),
			dstreamCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.DataPipelines, result.Updated.DataPipelines =
		cdstreams, udstreams
	return nil
}

// getEdgeInventoryDeltaScripts helper method to fill in Scripts in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaScripts(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	scripts, err := dbAPI.getScriptsMetadata(ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	scriptCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.Functions, scripts)
	result.Deleted.Functions = scriptCI.Deleted.GetIDs()
	cscripts, uscripts, err :=
		dbAPI.getScriptsByIDsMerge(ctx, scriptCI.Created.GetIDs(),
			scriptCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.Functions, result.Updated.Functions = cscripts, uscripts
	return nil
}

// getEdgeInventoryDeltaProjectServices helper method to fill in ProjectServices in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaProjectServices(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	projectServices, err := dbAPI.getProjectServicesMetadata(ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	projectServicesCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.ProjectServices, projectServices)
	result.Deleted.ProjectServices = projectServicesCI.Deleted.GetIDs()
	cProjectServices, uProjectServices, err :=
		dbAPI.getProjectServicesByIDsMerge(ctx, projectServicesCI.Created.GetIDs(),
			projectServicesCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.ProjectServices, result.Updated.ProjectServices = cProjectServices, uProjectServices
	return nil
}

// getEdgeInventoryDeltaServiceInstances helper method to fill in ServiceInstances in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaServiceInstances(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	svcInstances, err := dbAPI.getServiceInstancesMetadata(ctx, tenantID, edgeID, projectIDs)
	if err != nil {
		return err
	}
	svcInstancesCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.SvcInstances, svcInstances)
	result.Deleted.SvcInstances = svcInstancesCI.Deleted.GetIDs()
	cSvcInstances, uSvcInstances, err :=
		dbAPI.getServiceInstancesByIDsMerge(ctx, edgeID, svcInstancesCI.Created.GetIDs(),
			svcInstancesCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.SvcInstances, result.Updated.SvcInstances = cSvcInstances, uSvcInstances
	return nil
}

// getEdgeInventoryDeltaServiceBindings helper method to fill in ServiceBindings in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaServiceBindings(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	svcBindings, err := dbAPI.getServiceBindingsMetadata(ctx, tenantID, edgeID, projectIDs)
	if err != nil {
		return err
	}
	svcBindingsCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.SvcBindings, svcBindings)
	result.Deleted.SvcBindings = svcBindingsCI.Deleted.GetIDs()
	cSvcBindings, uSvcBindings, err :=
		dbAPI.getServiceBindingsByIDsMerge(ctx, edgeID, svcBindingsCI.Created.GetIDs(),
			svcBindingsCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.SvcBindings, result.Updated.SvcBindings = cSvcBindings, uSvcBindings
	return nil
}

// getEdgeInventoryDeltaLogCollectors helper method to fill in LogCollectors in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaLogCollectors(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) (logCollectorIDs []string, error error) {
	logCollectors, error := dbAPI.getLogCollectorsMetadata(ctx, tenantID, projectIDs)
	if error != nil {
		return
	}
	logCollectorsCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.LogCollectors, logCollectors)
	result.Deleted.LogCollectors = logCollectorsCI.Deleted.GetIDs()
	cLogCollectors, uLogCollectors, error :=
		dbAPI.getLogCollectorsByIDsMerge(ctx, tenantID, logCollectorsCI.Created.GetIDs(),
			logCollectorsCI.Updated.GetIDs())
	if error != nil {
		return
	}

	logCollectorIDs =
		model.EntityVersionMetadataList(logCollectors).GetIDs()

	result.Created.LogCollectors, result.Updated.LogCollectors = cLogCollectors, uLogCollectors
	return
}

// getEdgeInventoryDataDriverInstances helper method to fill in DataDriverInstances in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDataDriverInstances(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	instances, err := dbAPI.getDataDriverInstanceMetadata(ctx, tenantID, edgeID, projectIDs)
	if err != nil {
		return err
	}
	instancesCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.DataDriverInstances, instances)
	result.Deleted.DataDriverInstances = instancesCI.Deleted.GetIDs()
	cInstances, uInstances, err :=
		dbAPI.getDataDriverInstancesByIDsMerge(ctx, tenantID, instancesCI.Created.GetIDs(),
			instancesCI.Updated.GetIDs())
	if err != nil {
		return err
	}

	result.Created.DataDriverInstances, result.Updated.DataDriverInstances = cInstances, uInstances
	return nil
}

// getEdgeInventoryDeltaScriptRuntimes helper method to fill in ScriptRuntimes
// in edge inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaScriptRuntimes(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	scriptruntimes, err :=
		dbAPI.getScriptRuntimesMetadata(ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	scriptruntimeCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.RuntimeEnvironments,
			scriptruntimes)
	result.Deleted.RuntimeEnvironments = scriptruntimeCI.Deleted.GetIDs()
	cscriptruntimes, uscriptruntimes, err :=
		dbAPI.getScriptRuntimesByIDsMerge(ctx,
			scriptruntimeCI.Created.GetIDs(), scriptruntimeCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.RuntimeEnvironments, result.Updated.RuntimeEnvironments =
		cscriptruntimes, uscriptruntimes
	return nil
}

// getEdgeInventoryDeltaMLModels helper method to fill in MLModels in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaMLModels(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	mlModels, err := dbAPI.getMLModelsMetadata(ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	mlModelCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.MLModels, mlModels)
	result.Deleted.MLModels = mlModelCI.Deleted.GetIDs()
	cmlModels, umlModels, err := dbAPI.getMLModelsByIDsMerge(ctx,
		mlModelCI.Created.GetIDs(), mlModelCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.MLModels, result.Updated.MLModels = cmlModels, umlModels
	return nil
}

// getEdgeInventoryDeltaCloudProfiles helper method to fill in CloudProfiles in
// edge inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaCloudProfiles(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string, logcollectorIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	cloudprofiles, err :=
		dbAPI.getCloudProfilesMetadata(ctx, tenantID, projectIDs, logcollectorIDs)
	if err != nil {
		return err
	}
	cloudprofileCI := model.GetEntityVersionMetadataChangeInfo(
		payload.CloudProfiles, cloudprofiles)
	result.Deleted.CloudProfiles = cloudprofileCI.Deleted.GetIDs()
	ccloudprofiles, ucloudprofiles, err :=
		dbAPI.getCloudProfilesByIDsMerge(ctx,
			cloudprofileCI.Created.GetIDs(), cloudprofileCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.CloudProfiles, result.Updated.CloudProfiles = ccloudprofiles,
		ucloudprofiles
	return nil
}

// getEdgeInventoryDeltaContainerRegistries helper method to fill in
// ContainerRegistries in edge inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaContainerRegistries(
	ctx context.Context, tenantID string, edgeID string, projectIDs []string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	containerregistries, err := dbAPI.getContainerRegistriesMetadata(
		ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	containerregistryCI := model.GetEntityVersionMetadataChangeInfo(
		payload.ContainerRegistries, containerregistries)
	result.Deleted.ContainerRegistries = containerregistryCI.Deleted.GetIDs()
	ccontainerregistries, ucontainerregistries, err :=
		dbAPI.getContainerRegistriesByIDsMerge(ctx,
			containerregistryCI.Created.GetIDs(), containerregistryCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.ContainerRegistries, result.Updated.ContainerRegistries =
		ccontainerregistries, ucontainerregistries
	return nil
}

// getEdgeInventoryDeltaDataSources helper method to fill in DataSources in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaDataSources(
	ctx context.Context, tenantID string, edgeID string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	datasources, err := dbAPI.getDataSourcesMetadata(ctx, tenantID, edgeID)
	if err != nil {
		return err
	}
	datasourceCI :=
		model.GetEntityVersionMetadataChangeInfo(
			payload.DataSources, datasources)
	result.Deleted.DataSources = datasourceCI.Deleted.GetIDs()
	cdatasources, udatasources, err := dbAPI.getDataSourcesByIDsMerge(ctx,
		datasourceCI.Created.GetIDs(), datasourceCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.DataSources, result.Updated.DataSources =
		cdatasources, udatasources
	return nil
}

// getEdgeInventoryDeltaApplications helper method to fill in Applications in
// edge inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaApplications(
	ctx context.Context, tenantID string, edgeID string,
	explicitProjectIDs []string, categoryProjectIDs []string,
	edgeLabels []model.CategoryInfo,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	explicitApplications, err :=
		dbAPI.getExplicitApplicationsForEdgeMetadata(ctx, tenantID, edgeID,
			explicitProjectIDs)
	if err != nil {
		return err
	}

	allCategoryApplications, err :=
		dbAPI.getCategoryApplicationsMetadata(ctx, tenantID, categoryProjectIDs)
	if err != nil {
		return err
	}
	categoryApplicationsSelectors, err :=
		dbAPI.getCategoryApplicationsSelectors(ctx, tenantID, categoryProjectIDs)
	if err != nil {
		return err
	}
	categoryApplications :=
		getMatchingEntities(allCategoryApplications,
			categoryApplicationsSelectors, edgeLabels, true)
	applications := append(explicitApplications, categoryApplications...)
	applicationCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.Applications, applications)
	result.Deleted.Applications = applicationCI.Deleted.GetIDs()
	capplications, uapplications, err :=
		dbAPI.getApplicationsByIDsMerge(ctx, applicationCI.Created.GetIDs(),
			applicationCI.Updated.GetIDs())
	if err != nil {
		return err
	}
	result.Created.Applications, result.Updated.Applications =
		capplications, uapplications
	return nil
}

// getEdgeInventoryDeltaProjects helper method to fill in Projects in edge
// inventory delta
func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaProjects(
	ctx context.Context, tenantID string, edgeID string,
	payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) (explicitProjectIDs []string, categoryProjectIDs []string, projectIDs []string,
	edgeLabels []model.CategoryInfo, err error) {
	explicitProjects, err :=
		dbAPI.getExplicitProjectsForEdge(ctx, tenantID, edgeID)
	if err != nil {
		return
	}

	allCategoryProjects, err := dbAPI.getCategoryProjects(ctx, tenantID)
	if err != nil {
		return
	}
	categoryProjectsSelectors, err :=
		dbAPI.getCategoryProjectsSelectors(ctx, tenantID)
	if err != nil {
		return
	}
	edgeLabels, err = dbAPI.getEdgeLabels(ctx, edgeID)
	if err != nil {
		return
	}
	categoryProjects :=
		getMatchingEntities(allCategoryProjects,
			categoryProjectsSelectors, edgeLabels, false)

	projects := append(explicitProjects, categoryProjects...)

	explicitProjectIDs =
		model.EntityVersionMetadataList(explicitProjects).GetIDs()
	categoryProjectIDs =
		model.EntityVersionMetadataList(categoryProjects).GetIDs()

	projectIDs = model.EntityVersionMetadataList(projects).GetIDs()

	projectCI :=
		model.GetEntityVersionMetadataChangeInfo(payload.Projects, projects)
	result.Deleted.Projects = projectCI.Deleted.GetIDs()
	cprojects, uprojects, err := dbAPI.getProjectsByIDsMerge(ctx, tenantID,
		projectCI.Created.GetIDs(), projectCI.Updated.GetIDs())
	if err != nil {
		return
	}
	result.Created.Projects, result.Updated.Projects =
		cprojects, uprojects
	return
}

func postProcessEdgeInventoryDeltaResponse(response *model.EdgeInventoryDeltaResponse) {
	// move applications and data pipelines with state = UNDEPLOY
	// from Updated list to Deleted list.
	// drop applications and data pipelines with state = UNDEPLOY
	// from Created list.
	// This should be considered a workaround until edges properly support
	// the UNDEPLOY state.
	postProcessApplications(&response.Created, nil)
	postProcessApplications(&response.Updated, &response.Deleted)
	postProcessDataPipelines(&response.Created, nil)
	postProcessDataPipelines(&response.Updated, &response.Deleted)
}
func postProcessApplications(details *model.EdgeInventoryDetails, deleted *model.EdgeInventoryDeleted) {
	// move applications whose state == "UNDEPLOY" from details to deleted
	undeployedIDs := []string{}
	undeployedIndices := []int{}
	for i := range details.Applications {
		if details.Applications[i].State != nil && *details.Applications[i].State == string(model.UndeployEntityState) {
			if deleted != nil {
				undeployedIDs = append(undeployedIDs, details.Applications[i].ID)
			}
			undeployedIndices = append(undeployedIndices, i)
		}
	}
	if len(undeployedIndices) != 0 {
		if deleted != nil {
			deleted.Applications = append(deleted.Applications, undeployedIDs...)
		}
		n := len(undeployedIndices)
		N := len(details.Applications)
		for i, j := n-1, N-1; i >= 0; i, j = i-1, j-1 {
			details.Applications[undeployedIndices[i]] = details.Applications[j]
		}
		details.Applications = details.Applications[:N-n]
	}
}
func postProcessDataPipelines(details *model.EdgeInventoryDetails, deleted *model.EdgeInventoryDeleted) {
	// move data pipelines whose state == "UNDEPLOY" from details to deleted
	undeployedIDs := []string{}
	undeployedIndices := []int{}
	for i := range details.DataPipelines {
		if details.DataPipelines[i].State != nil && *details.DataPipelines[i].State == string(model.UndeployEntityState) {
			if deleted != nil {
				undeployedIDs = append(undeployedIDs, details.DataPipelines[i].ID)
			}
			undeployedIndices = append(undeployedIndices, i)
		}
	}
	if len(undeployedIndices) != 0 {
		if deleted != nil {
			deleted.DataPipelines = append(deleted.DataPipelines, undeployedIDs...)
		}
		n := len(undeployedIndices)
		N := len(details.DataPipelines)
		for i, j := n-1, N-1; i >= 0; i, j = i-1, j-1 {
			details.DataPipelines[undeployedIndices[i]] = details.DataPipelines[j]
		}
		details.DataPipelines = details.DataPipelines[:N-n]
	}
}

//// Get Edge Inventory Delta - main function
// get edge inventory delta based on the given payload
func (dbAPI *dbObjectModelAPI) GetEdgeInventoryDelta(
	ctx context.Context, payload *model.EdgeInventoryDeltaPayload,
) (*model.EdgeInventoryDeltaResponse, error) {

	start := time.Now()
	logON := glog.V(5)

	result := &model.EdgeInventoryDeltaResponse{
		Deleted: model.EdgeInventoryDeleted{},
		Created: model.EdgeInventoryDetails{},
		Updated: model.EdgeInventoryDetails{},
	}

	tenantID, edgeID, err := dbAPI.getTenantAndEdgeIDs(ctx)
	if err != nil {
		return nil, err
	}

	// Projects
	explicitProjectIDs, categoryProjectIDs, projectIDs, edgeLabels, err :=
		dbAPI.getEdgeInventoryDeltaProjects(ctx, tenantID, edgeID, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:projects")
	}

	// Categories
	if SEND_CATEGORIES_DELTA {
		err = dbAPI.getEdgeInventoryDeltaCategories(
			ctx, tenantID, edgeID, payload, result)
		if err != nil {
			return nil, err
		}

		if logON {
			timeTrack(start, "GetEdgeInventoryDelta:categories")
		}
	}

	// Data Streams
	err = dbAPI.getEdgeInventoryDeltaDataStreams(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:data streams")
	}

	// Scripts
	err = dbAPI.getEdgeInventoryDeltaScripts(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:scripts")
	}

	// ScriptRuntimes
	err = dbAPI.getEdgeInventoryDeltaScriptRuntimes(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:script runtimes")
	}

	// ML Models
	err = dbAPI.getEdgeInventoryDeltaMLModels(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:ml models")
	}

	// Log Collectors
	logcollectorIDs, err := dbAPI.getEdgeInventoryDeltaLogCollectors(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:log collectors")
	}

	// Cloud Profiles
	err = dbAPI.getEdgeInventoryDeltaCloudProfiles(
		ctx, tenantID, edgeID, projectIDs, logcollectorIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:cloud profiles")
	}

	// Container Registries
	err = dbAPI.getEdgeInventoryDeltaContainerRegistries(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:container registries")
	}

	// Data Sources
	err = dbAPI.getEdgeInventoryDeltaDataSources(
		ctx, tenantID, edgeID, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:data sources")
	}

	// Applications
	err = dbAPI.getEdgeInventoryDeltaApplications(
		ctx, tenantID, edgeID, explicitProjectIDs, categoryProjectIDs,
		edgeLabels, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:applications")
	}

	// Project Services
	err = dbAPI.getEdgeInventoryDeltaProjectServices(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:project services")
	}

	// Service Instances
	err = dbAPI.getEdgeInventoryDeltaServiceInstances(
		ctx, tenantID, edgeID, projectIDs, payload, result)

	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:service instances")
	}

	// Service Bindings
	err = dbAPI.getEdgeInventoryDeltaServiceBindings(
		ctx, tenantID, edgeID, projectIDs, payload, result)

	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:service bindings")
	}

	// Software update
	err = dbAPI.getEdgeInventoryDeltaSoftwareUpdates(ctx, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:software updates")
	}

	// Data Driver Instances
	err = dbAPI.getEdgeInventoryDataDriverInstances(
		ctx, tenantID, edgeID, projectIDs, payload, result)
	if err != nil {
		return nil, err
	}

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:data driver instance")
	}

	postProcessEdgeInventoryDeltaResponse(result)

	if logON {
		timeTrack(start, "GetEdgeInventoryDelta:post process")
	}

	return result, nil
}

// GetEdgeInventoryDeltaW is a wrapper on GetEdgeInventoryDelta that takes
// Writer, Request as input it is used to register to router as
// /edgeinventorydelta API handler Calling context could be from edge or infra
// admin. For infra admin, edgeId query parameter is required.
func (dbAPI *dbObjectModelAPI) GetEdgeInventoryDeltaW(ctx context.Context,
	w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	doc := model.EdgeInventoryDeltaPayload{}
	var r io.Reader = req.Body
	defer req.Body.Close()
	err = base.Decode(&r, &doc)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	// allow infra admin to impersonate edge
	if auth.IsInfraAdminRole(authContext) {
		// get edgeID from query parameter
		query := req.URL.Query()
		edgeIDVals := query["edgeId"]
		if len(edgeIDVals) == 1 {
			edgeID := strings.TrimSpace(edgeIDVals[0])
			tenantID := authContext.TenantID
			ctx = context.WithValue(context.Background(), base.AuthContextKey,
				&base.AuthContext{
					TenantID: tenantID,
					Claims: jwt.MapClaims{
						"specialRole": "edge",
						"edgeId":      edgeID,
					},
				})
		} else {
			return errcode.NewBadRequestError("edgeId")
		}
	}

	result, err := dbAPI.GetEdgeInventoryDelta(ctx, &doc)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(*result)
}

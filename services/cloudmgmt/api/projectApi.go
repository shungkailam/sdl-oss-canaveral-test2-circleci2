package api

import (
	"cloudservices/cloudmgmt/config"
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

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	funk "github.com/thoas/go-funk"
)

const entityTypeProject = "project"

func init() {
	// select projects by project ids - needed when populate apps edgeSelectors
	queryMap["SelectProjectsTemplate1"] = `SELECT * FROM project_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) %s`
	queryMap["SelectProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM project_model WHERE tenant_id = :tenant_id %s`
	queryMap["SelectProjectsByIDs"] = `SELECT * FROM project_model WHERE tenant_id = :tenant_id AND id IN (:project_ids)`
	queryMap["SelectProjectsByIDsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM project_model WHERE tenant_id = :tenant_id AND id IN (:project_ids) %s`
	queryMap["SelectProjectUsers"] = `SELECT * FROM project_user_model WHERE project_id = :project_id AND (:user_id = '' OR user_id = :user_id)`
	queryMap["SelectProjectsUsers"] = `SELECT * FROM project_user_model WHERE project_id IN (:project_ids)`
	queryMap["SelectProjectCloudCreds"] = `SELECT * FROM project_cloud_creds_model WHERE project_id = :project_id`
	queryMap["SelectProjectsCloudCreds"] = `SELECT * FROM project_cloud_creds_model WHERE project_id IN (:project_ids)`
	queryMap["SelectProjectDockerProfiles"] = `SELECT * FROM project_docker_profile_model WHERE project_id = :project_id`
	queryMap["SelectProjectsDockerProfiles"] = `SELECT * FROM project_docker_profile_model WHERE project_id IN (:project_ids)`
	queryMap["CreateProject"] = `INSERT INTO project_model (id, version, tenant_id, name, description, edge_selector_type, edge_selectors, privileged, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :edge_selector_type, :edge_selectors, :privileged, :created_at, :updated_at)`
	queryMap["CreateProjectUser"] = `INSERT INTO project_user_model (project_id, user_id, user_role) VALUES (:project_id, :user_id, :user_role)`
	queryMap["CreateProjectDockerProfile"] = `INSERT INTO project_docker_profile_model (project_id, docker_profile_id) VALUES (:project_id, :docker_profile_id)`
	queryMap["CreateProjectCloudCreds"] = `INSERT INTO project_cloud_creds_model (project_id, cloud_creds_id) VALUES (:project_id, :cloud_creds_id)`
	queryMap["UpdateProject"] = `UPDATE project_model SET version = :version, name = :name, description = :description, edge_selector_type = :edge_selector_type, edge_selectors = :edge_selectors, privileged = :privileged, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["CreateProjectEdge"] = `INSERT INTO project_edge_model (project_id, edge_id) VALUES (:project_id, :edge_id)`
	queryMap["SelectProjectEdges"] = `SELECT * FROM project_edge_model WHERE project_id = :project_id`
	queryMap["SelectProjectsEdges"] = `SELECT * FROM project_edge_model WHERE project_id IN (:project_ids)`
	queryMap["CreateProjectEdgeSelector"] = `INSERT INTO project_edge_selector_model (project_id, category_value_id) VALUES (:project_id, :category_value_id)`
	queryMap["SelectProjectEdgeSelectors"] = `SELECT project_edge_selector_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
	  FROM project_edge_selector_model JOIN category_value_model ON project_edge_selector_model.category_value_id = category_value_model.id WHERE project_edge_selector_model.project_id = :project_id`
	queryMap["SelectProjectsEdgeSelectors"] = `SELECT project_edge_selector_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
	  FROM project_edge_selector_model JOIN category_value_model ON project_edge_selector_model.category_value_id = category_value_model.id WHERE project_edge_selector_model.project_id IN (:project_ids)`
	queryMap["CheckTenantTemplate"] = `SELECT tenant_id FROM %s WHERE id IN (:ids)`
	queryMap["SelectProjectDataStreamsUsingCloudCreds"] = `SELECT id from data_stream_model where tenant_id = :tenant_id and project_id = :project_id and cloud_creds_id IN (:cloud_creds_ids)`
	queryMap["SelectProjectsEdgeClusters"] = `SELECT * FROM project_edge_model WHERE project_id IN (:project_ids)`
	orderByHelper.Setup(entityTypeProject, []string{"id", "version", "created_at", "updated_at", "name", "description", "edge_selector_type"})
}

// projectDBO is DB object model for project
type ProjectDBO struct {
	model.BaseModelDBO
	Name             string          `json:"name" db:"name"`
	Description      string          `json:"description" db:"description"`
	EdgeSelectorType string          `json:"edgeSelectorType" db:"edge_selector_type"`
	EdgeSelectors    *types.JSONText `json:"edgeSelectors" db:"edge_selectors"`
	Privileged       *bool           `json:"privileged" db:"privileged"`
}

// ProjectUserDBO is the DB model for ProjectUser
type ProjectUserDBO struct {
	ID        int64  `json:"id" db:"id"`
	ProjectID string `json:"projectId" db:"project_id"`
	UserID    string `json:"userId" db:"user_id"`
	Role      string `json:"role" db:"user_role"`
}

type ProjectCloudCredsDBO struct {
	ID           int64  `json:"id" db:"id"`
	ProjectID    string `json:"projectId" db:"project_id"`
	CloudCredsID string `json:"cloudCredsId" db:"cloud_creds_id"`
}

type ProjectDockerProfileDBO struct {
	ID              int64  `json:"id" db:"id"`
	ProjectID       string `json:"projectId" db:"project_id"`
	DockerProfileID string `json:"dockerProfileId" db:"docker_profile_id"`
}

type ProjectEdgeDBO struct {
	ID        int64  `json:"id" db:"id"`
	ProjectID string `json:"projectId" db:"project_id"`
	EdgeID    string `json:"edgeId" db:"edge_id"`
}
type ProjectEdgeClusterDBO struct {
	ID            int64  `json:"id" db:"id"`
	ProjectID     string `json:"projectId" db:"project_id"`
	EdgeClusterID string `json:"edgeClusterId" db:"edge_id"`
}

type ProjectIdsParam struct {
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}
type TenantProjectIdsParam struct {
	TenantID   string   `json:"tenantId" db:"tenant_id"`
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}
type TenantProjectCloudCredsIdsParam struct {
	TenantID      string   `json:"tenantId" db:"tenant_id"`
	ProjectID     string   `json:"projectId" db:"project_id"`
	CloudCredsIDs []string `json:"cloudCredsIds" db:"cloud_creds_ids"`
}

type ProjectEdgeSelectorDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64  `json:"id" db:"id"`
	ProjectID          string `json:"projectId" db:"project_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
}

type TenantQueryParam struct {
	TenantID string `json:"tenantId" db:"tenant_id"`
}

func (doc ProjectDBO) GetProjectID() string {
	return doc.ID
}

func GetDefaultProjectID(tenantID string) string {
	projectID := base.GetMD5Hash(fmt.Sprintf("%s/default-project", tenantID))
	return *projectID
}

func (dbAPI *dbObjectModelAPI) validateProject(context context.Context, doc *model.Project) error {
	est := doc.EdgeSelectorType
	if est == model.ProjectEdgeSelectorTypeExplicit || est == "explicit" { // lower case for test backward compatibility
		doc.EdgeSelectors = nil
	} else if est == model.ProjectEdgeSelectorTypeCategory {
		doc.EdgeIDs = []string{}
	} else {
		return errcode.NewBadRequestError("project.edgeSelectorType")
	}
	doc.DockerProfileIDs = base.Unique(doc.DockerProfileIDs)

	// ENG-169964: auto add cloud profiles backing each docker profile to project
	if len(doc.DockerProfileIDs) > 0 {
		containerRegistries, err := dbAPI.SelectContainerRegistriesByIDs(context, doc.DockerProfileIDs)
		if err != nil {
			return err
		}
		for _, containerRegistry := range containerRegistries {
			if containerRegistry.CloudCredsID != "" {
				if !funk.Contains(doc.CloudCredentialIDs, containerRegistry.CloudCredsID) {
					doc.CloudCredentialIDs = append(doc.CloudCredentialIDs, containerRegistry.CloudCredsID)
				}
			}
		}
	}

	doc.EdgeIDs = base.Unique(doc.EdgeIDs)
	doc.CloudCredentialIDs = base.Unique(doc.CloudCredentialIDs)
	doc.Users = uniqueProjectUserInfos(doc.Users)

	// validate Users
	for _, user := range doc.Users {
		err := model.ValidateProjectUserInfo(&user)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dbAPI *dbObjectModelAPI) createProjectUsers(context context.Context, tx *base.WrappedTx, doc *model.Project) error {
	userIDs := []string{}
	for _, user := range doc.Users {
		userIDs = append(userIDs, user.UserID)
	}
	err := dbAPI.checkTenant(context, UserTableName, userIDs)
	if err != nil {
		return errcode.NewBadRequestError("Project.Users")
	}
	for _, user := range doc.Users {
		// The DB ID is generated
		projectUserDBO := ProjectUserDBO{ProjectID: doc.ID, UserID: user.UserID, Role: user.Role}
		_, err := tx.NamedExec(context, queryMap["CreateProjectUser"], &projectUserDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating project user %+v. Error: %s"), projectUserDBO, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
	}
	return nil
}
func (dbAPI *dbObjectModelAPI) createProjectDockerProfiles(context context.Context, tx *base.WrappedTx, doc *model.Project) error {
	err := dbAPI.checkTenant(context, ContainerRegistryTableName, doc.DockerProfileIDs)
	if err != nil {
		return errcode.NewBadRequestError("Project.DockerProfileIDs")
	}
	for _, dockerProfileID := range doc.DockerProfileIDs {
		// The DB ID is generated
		projectDockerProfileDBO := ProjectDockerProfileDBO{ProjectID: doc.ID, DockerProfileID: dockerProfileID}
		_, err := tx.NamedExec(context, queryMap["CreateProjectDockerProfile"], &projectDockerProfileDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating project docker profile %+v. Error: %s"), projectDockerProfileDBO, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
	}
	return nil
}
func (dbAPI *dbObjectModelAPI) createProjectCloudCredss(context context.Context, tx *base.WrappedTx, doc *model.Project) error {
	err := dbAPI.checkTenant(context, CloudProfileTableName, doc.CloudCredentialIDs)
	if err != nil {
		return errcode.NewBadRequestError("Project.CloudCredentialIDs")
	}
	for _, cloudCredentialID := range doc.CloudCredentialIDs {
		// The DB ID is generated
		projectCloudCredsDBO := ProjectCloudCredsDBO{ProjectID: doc.ID, CloudCredsID: cloudCredentialID}
		_, err := tx.NamedExec(context, queryMap["CreateProjectCloudCreds"], &projectCloudCredsDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating project cloud creds %+v. Error: %s"), projectCloudCredsDBO, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
	}
	return nil
}
func (dbAPI *dbObjectModelAPI) createProjectEdges(context context.Context, tx *base.WrappedTx, doc *model.Project) error {
	err := dbAPI.checkTenant(context, EdgeClusterTableName, doc.EdgeIDs)
	if err != nil {
		return errcode.NewBadRequestExError("Project.EdgeIDs", err.Error())
	}
	for _, edgeID := range doc.EdgeIDs {
		// The DB ID is generated
		projectEdgeDBO := ProjectEdgeDBO{ProjectID: doc.ID, EdgeID: edgeID}
		_, err := tx.NamedExec(context, queryMap["CreateProjectEdge"], &projectEdgeDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating project edge %+v. Error: %s"), projectEdgeDBO, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
	}
	return nil
}

// TODO FIXME - make this method generic
func (dbAPI *dbObjectModelAPI) createProjectEdgeSelectors(ctx context.Context, tx *base.WrappedTx, project *model.Project) error {
	for _, categoryInfo := range project.EdgeSelectors {
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
				projectEdgeSelectorDBO := ProjectEdgeSelectorDBO{ProjectID: project.ID, CategoryValueID: categoryValueDBO.ID}
				_, err = tx.NamedExec(ctx, queryMap["CreateProjectEdgeSelector"], &projectEdgeSelectorDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error occurred while creating project edge selector for ID %s. Error: %s"), project.ID, err.Error())
					return errcode.TranslateDatabaseError(project.ID, err)
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

func (dbAPI *dbObjectModelAPI) populateProjectsAssociations(ctx context.Context, projects []model.Project) error {
	if len(projects) == 0 {
		return nil
	}
	var err error
	err = dbAPI.populateProjectsEdgeAssociations(ctx, projects)
	if err != nil {
		return err
	}
	err = dbAPI.populateProjectsCloudCredsAssociations(ctx, projects)
	if err != nil {
		return err
	}
	err = dbAPI.populateProjectsDockerProfileAssociations(ctx, projects)
	if err != nil {
		return err
	}
	err = dbAPI.populateProjectsUserAssociations(ctx, projects)
	if err != nil {
		return err
	}
	return nil
}

func getProjectIDsParam(projects []model.Project) ProjectIdsParam {
	projectIDs := []string{}
	for i := 0; i < len(projects); i++ {
		projectIDs = append(projectIDs, projects[i].ID)
	}
	return ProjectIdsParam{
		ProjectIDs: projectIDs,
	}
}

func (dbAPI *dbObjectModelAPI) populateProjectsEdgeAssociations(ctx context.Context, projects []model.Project) error {
	if len(projects) == 0 {
		return nil
	}
	categoryProjects := []*model.Project{}
	explicitProjects := []*model.Project{}
	explicitProjectIDs := []string{}
	categoryProjectIDs := []string{}
	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			categoryProjects = append(categoryProjects, project)
			categoryProjectIDs = append(categoryProjectIDs, project.ID)
		} else {
			explicitProjects = append(explicitProjects, project)
			explicitProjectIDs = append(explicitProjectIDs, project.ID)
		}
	}
	if len(categoryProjects) != 0 {
		projectEdgeSelectorDBOs := []ProjectEdgeSelectorDBO{}
		err := dbAPI.QueryIn(ctx, &projectEdgeSelectorDBOs, queryMap["SelectProjectsEdgeSelectors"], ProjectIdsParam{
			ProjectIDs: categoryProjectIDs,
		})
		if err != nil {
			return err
		}
		projectEdgeSelectorsMap := map[string]([]model.CategoryInfo){}
		for _, projectEdgeSelectorDBO := range projectEdgeSelectorDBOs {
			projectEdgeSelectorsMap[projectEdgeSelectorDBO.ProjectID] = append(projectEdgeSelectorsMap[projectEdgeSelectorDBO.ProjectID], projectEdgeSelectorDBO.CategoryInfo)
		}
		for _, project := range categoryProjects {
			project.EdgeSelectors = projectEdgeSelectorsMap[project.ID]
		}
	}

	projectEdgeIDsMap := map[string]([]string){}

	// first for projects where edgeSelectorType = 'Explicit'
	if len(explicitProjects) != 0 {
		projectIdsParam := ProjectIdsParam{
			ProjectIDs: explicitProjectIDs,
		}
		projectEdgeDBOs := []ProjectEdgeDBO{}
		err := dbAPI.QueryIn(ctx, &projectEdgeDBOs, queryMap["SelectProjectsEdges"], projectIdsParam)
		if err != nil {
			return err
		}
		for _, projectEdgeDBO := range projectEdgeDBOs {
			projectEdgeIDsMap[projectEdgeDBO.ProjectID] = append(projectEdgeIDsMap[projectEdgeDBO.ProjectID], projectEdgeDBO.EdgeID)
		}
	}

	// next for projects where edgeSelectorType = 'Category'
	if len(categoryProjects) != 0 {
		adminCtx, err := makeAdminContext(ctx)
		if err != nil {
			return err
		}
		edges, err := dbAPI.getAllClusterTypes(adminCtx, false) // TODO FIXME - optimize: only select edges with some category assigned
		if err != nil {
			return err
		}
		for _, project := range categoryProjects {
			for _, edge := range edges {
				if model.CategoryMatch(edge.Labels, project.EdgeSelectors) {
					projectEdgeIDsMap[project.ID] = append(projectEdgeIDsMap[project.ID], edge.ID)
				}
			}
		}
	}

	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		project.EdgeIDs = NilToEmptyStrings(projectEdgeIDsMap[project.ID])
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) populateProjectsCloudCredsAssociations(ctx context.Context, projects []model.Project) error {
	if len(projects) == 0 {
		return nil
	}
	projectIdsParam := getProjectIDsParam(projects)
	projectCloudCredsDBOs := []ProjectCloudCredsDBO{}
	err := dbAPI.QueryIn(ctx, &projectCloudCredsDBOs, queryMap["SelectProjectsCloudCreds"], projectIdsParam)
	if err != nil {
		return err
	}
	projectCloudCredentialIDsMap := map[string]([]string){}
	for _, projectCloudCredsDBO := range projectCloudCredsDBOs {
		projectCloudCredentialIDsMap[projectCloudCredsDBO.ProjectID] = append(projectCloudCredentialIDsMap[projectCloudCredsDBO.ProjectID], projectCloudCredsDBO.CloudCredsID)
	}
	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		project.CloudCredentialIDs = NilToEmptyStrings(projectCloudCredentialIDsMap[project.ID])
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) populateProjectsDockerProfileAssociations(ctx context.Context, projects []model.Project) error {
	if len(projects) == 0 {
		return nil
	}
	projectIdsParam := getProjectIDsParam(projects)
	projectDockerProfileDBOs := []ProjectDockerProfileDBO{}
	err := dbAPI.QueryIn(ctx, &projectDockerProfileDBOs, queryMap["SelectProjectsDockerProfiles"], projectIdsParam)
	if err != nil {
		return err
	}
	projectDockerProfileIDsMap := map[string]([]string){}
	for _, projectDockerProfileDBO := range projectDockerProfileDBOs {
		projectDockerProfileIDsMap[projectDockerProfileDBO.ProjectID] = append(projectDockerProfileIDsMap[projectDockerProfileDBO.ProjectID], projectDockerProfileDBO.DockerProfileID)
	}
	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		project.DockerProfileIDs = NilToEmptyStrings(projectDockerProfileIDsMap[project.ID])
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) populateProjectsUserAssociations(ctx context.Context, projects []model.Project) error {
	if len(projects) == 0 {
		return nil
	}
	projectIdsParam := getProjectIDsParam(projects)
	projectUserDBOs := []ProjectUserDBO{}
	err := dbAPI.QueryIn(ctx, &projectUserDBOs, queryMap["SelectProjectsUsers"], projectIdsParam)
	if err != nil {
		return err
	}
	projectUsersMap := map[string]([]model.ProjectUserInfo){}
	for _, projectUserDBO := range projectUserDBOs {
		projectUsersMap[projectUserDBO.ProjectID] = append(projectUsersMap[projectUserDBO.ProjectID],
			model.ProjectUserInfo{
				UserID: projectUserDBO.UserID,
				Role:   projectUserDBO.Role,
			})
	}
	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		project.Users = uniqueProjectUserInfos(projectUsersMap[project.ID])
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getProjectsEtag(ctx context.Context, etag string, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Project, error) {
	projects := []model.Project{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return projects, err
	}
	tenantID := authContext.TenantID

	projectDBOs := []ProjectDBO{}
	baseModel := model.BaseModelDBO{TenantID: tenantID, ID: projectID}
	param := ProjectDBO{BaseModelDBO: baseModel}
	query, err := buildQuery(entityTypeProject, queryMap["SelectProjectsTemplate1"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return projects, err
	}
	err = dbAPI.Query(ctx, &projectDBOs, query, param)
	if err != nil {
		return projects, err
	}
	for _, projectDBO := range projectDBOs {
		project := model.Project{}
		err = base.Convert(&projectDBO, &project)
		if err != nil {
			return projects, err
		}
		projects = append(projects, project)
	}
	err = dbAPI.populateProjectsAssociations(ctx, projects)
	return projects, err
}

func (dbAPI *dbObjectModelAPI) getProjectsByIDs(ctx context.Context, tenantID string, projectIDs []string) ([]model.Project, error) {
	projects := []model.Project{}
	if len(projectIDs) == 0 {
		return projects, nil
	}
	projectDBOs := []ProjectDBO{}
	param := TenantProjectIdsParam{TenantID: tenantID, ProjectIDs: projectIDs}
	err := dbAPI.QueryIn(ctx, &projectDBOs, queryMap["SelectProjectsByIDs"], param)
	if err != nil {
		return projects, err
	}
	for _, projectDBO := range projectDBOs {
		project := model.Project{}
		err = base.Convert(&projectDBO, &project)
		if err != nil {
			return projects, err
		}
		projects = append(projects, project)
	}
	err = dbAPI.populateProjectsAssociations(ctx, projects)
	return projects, err
}

// internal API used by SelectAllProjectsWV2
func (dbAPI *dbObjectModelAPI) getProjectsForQuery(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.Project, int, error) {
	projects := []model.Project{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return projects, 0, err
	}
	tenantID := authContext.TenantID
	projectDBOs := []ProjectDBO{}

	// note: we return all projects for edge to simplify project update processing for edge
	var query string
	if auth.IsInfraAdminOrEdgeRole(authContext) {
		query, err = buildLimitQuery(entityTypeProject, queryMap["SelectProjectsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return projects, 0, err
		}
		err = dbAPI.Query(context, &projectDBOs, query, tenantIDParam2{TenantID: tenantID})
	} else {
		projectIDs := auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return projects, 0, nil
		}
		query, err = buildLimitQuery(entityTypeProject, queryMap["SelectProjectsByIDsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return projects, 0, err
		}
		err = dbAPI.QueryIn(context, &projectDBOs, query, tenantIDParam2{TenantID: tenantID, ProjectIDs: projectIDs})
	}
	if err != nil {
		return projects, 0, err
	}
	if len(projectDBOs) == 0 {
		return projects, 0, nil
	}
	totalCount := 0
	first := true
	for _, projectDBO := range projectDBOs {
		project := model.Project{}
		if first {
			first = false
			if projectDBO.TotalCount != nil {
				totalCount = *projectDBO.TotalCount
			}
		}
		err := base.Convert(&projectDBO, &project)
		if err != nil {
			return []model.Project{}, 0, err
		}
		projects = append(projects, project)
	}
	err = dbAPI.populateProjectsAssociations(context, projects)
	return projects, totalCount, err
}

func (dbAPI *dbObjectModelAPI) getProjectUserDBOs(ctx context.Context, param ProjectUserDBO) ([]ProjectUserDBO, error) {
	projectUserDBOs := []ProjectUserDBO{}
	err := dbAPI.Query(ctx, &projectUserDBOs, queryMap["SelectProjectUsers"], param)
	return projectUserDBOs, err
}

func (dbAPI *dbObjectModelAPI) getProjectDockerProfiles(ctx context.Context, param ProjectDockerProfileDBO) ([]string, error) {
	projectDockerProfiles := []string{}
	projectDockerProfileDBOs := []ProjectDockerProfileDBO{}
	err := dbAPI.Query(ctx, &projectDockerProfileDBOs, queryMap["SelectProjectDockerProfiles"], param)
	if err == nil {
		for _, projectDockerProfileDBO := range projectDockerProfileDBOs {
			projectDockerProfiles = append(projectDockerProfiles, projectDockerProfileDBO.DockerProfileID)
		}
	}
	return projectDockerProfiles, err
}

func (dbAPI *dbObjectModelAPI) getProjectCloudCreds(ctx context.Context, param ProjectCloudCredsDBO) ([]string, error) {
	projectCloudCreds := []string{}
	projectCloudCredsDBOs := []ProjectCloudCredsDBO{}
	err := dbAPI.Query(ctx, &projectCloudCredsDBOs, queryMap["SelectProjectCloudCreds"], param)
	if err == nil {
		for _, projectCloudCredsDBO := range projectCloudCredsDBOs {
			projectCloudCreds = append(projectCloudCreds, projectCloudCredsDBO.CloudCredsID)
		}
	}
	return projectCloudCreds, err
}

func (dbAPI *dbObjectModelAPI) GetProjectEdges(context context.Context, param ProjectEdgeDBO) ([]string, error) {
	edgeIDs := []string{}
	adminCtx, err := makeAdminContext(context)
	if err != nil {
		return edgeIDs, err
	}
	project, err := dbAPI.GetProject(adminCtx, param.ProjectID)
	if err != nil {
		return edgeIDs, err
	}
	if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
		edgeClusterIDLabelsList, err := dbAPI.SelectEdgeClusterIDLabels(adminCtx)
		if err != nil {
			return edgeIDs, err
		}
		for _, edgeClusterIDLabels := range edgeClusterIDLabelsList {
			if model.CategoryMatch(edgeClusterIDLabels.Labels, project.EdgeSelectors) {
				edgeIDs = append(edgeIDs, edgeClusterIDLabels.ID)
			}
		}
	} else {
		projectEdgeDBOs := []ProjectEdgeDBO{}
		err := dbAPI.Query(adminCtx, &projectEdgeDBOs, queryMap["SelectProjectEdges"], param)
		if err == nil {
			for _, projectEdgeDBO := range projectEdgeDBOs {
				edgeIDs = append(edgeIDs, projectEdgeDBO.EdgeID)
			}
		}
	}
	return edgeIDs, err
}

func (dbAPI *dbObjectModelAPI) GetProjectsEdges(context context.Context, projectIDs []string) ([]string, error) {
	edgeIDs := []string{}
	if len(projectIDs) == 0 {
		return edgeIDs, nil
	}
	adminCtx, err := makeAdminContext(context)
	if err != nil {
		return edgeIDs, err
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeIDs, err
	}
	projects, err := dbAPI.getProjectsByIDs(adminCtx, authContext.TenantID, projectIDs)
	if err != nil {
		return edgeIDs, err
	}
	categoryProjects := []*model.Project{}
	explicitProjects := []*model.Project{}
	explicitProjectIDs := []string{}
	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			categoryProjects = append(categoryProjects, project)
		} else {
			explicitProjects = append(explicitProjects, project)
			explicitProjectIDs = append(explicitProjectIDs, project.ID)
		}
	}
	edgeMap := map[string]bool{}
	if len(explicitProjects) != 0 {
		projectEdgeDBOs := []ProjectEdgeDBO{}
		param := ProjectIdsParam{
			ProjectIDs: explicitProjectIDs,
		}
		err = dbAPI.QueryIn(adminCtx, &projectEdgeDBOs, queryMap["SelectProjectsEdges"], param)
		if err != nil {
			return edgeIDs, err
		}
		for _, projectEdgeDBO := range projectEdgeDBOs {
			if !edgeMap[projectEdgeDBO.EdgeID] {
				edgeMap[projectEdgeDBO.EdgeID] = true
				edgeIDs = append(edgeIDs, projectEdgeDBO.EdgeID)
			}
		}
	}
	if len(categoryProjects) != 0 {
		edges, err := dbAPI.SelectEdgeClusterIDLabels(adminCtx)
		if err != nil {
			return edgeIDs, err
		}
		for _, edge := range edges {
			if !edgeMap[edge.ID] {
				for _, project := range categoryProjects {
					if model.CategoryMatch(edge.Labels, project.EdgeSelectors) {
						edgeMap[edge.ID] = true
						edgeIDs = append(edgeIDs, edge.ID)
						break
					}
				}
			}
		}
	}
	return edgeIDs, err
}

func (dbAPI *dbObjectModelAPI) GetProjectsEdgeClusters(context context.Context, projectIDs []string) ([]string, error) {
	edgeClusterIDs := []string{}
	if len(projectIDs) == 0 {
		return edgeClusterIDs, nil
	}
	adminCtx, err := makeAdminContext(context)
	if err != nil {
		return edgeClusterIDs, err
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeClusterIDs, err
	}
	projects, err := dbAPI.getProjectsByIDs(adminCtx, authContext.TenantID, projectIDs)
	if err != nil {
		return edgeClusterIDs, err
	}
	categoryProjects := []*model.Project{}
	explicitProjects := []*model.Project{}
	explicitProjectIDs := []string{}
	for i := 0; i < len(projects); i++ {
		project := &projects[i]
		if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			categoryProjects = append(categoryProjects, project)
		} else {
			explicitProjects = append(explicitProjects, project)
			explicitProjectIDs = append(explicitProjectIDs, project.ID)
		}
	}
	edgeClusterMap := map[string]bool{}
	if len(explicitProjects) != 0 {
		projectEdgeClusterDBOs := []ProjectEdgeClusterDBO{}
		param := ProjectIdsParam{
			ProjectIDs: explicitProjectIDs,
		}
		err = dbAPI.QueryIn(adminCtx, &projectEdgeClusterDBOs, queryMap["SelectProjectsEdgeClusters"], param)
		if err != nil {
			return edgeClusterIDs, err
		}
		for _, projectEdgeClusterDBO := range projectEdgeClusterDBOs {
			if !edgeClusterMap[projectEdgeClusterDBO.EdgeClusterID] {
				edgeClusterMap[projectEdgeClusterDBO.EdgeClusterID] = true
				edgeClusterIDs = append(edgeClusterIDs, projectEdgeClusterDBO.EdgeClusterID)
			}
		}
	}
	if len(categoryProjects) != 0 {
		edgeClusters, err := dbAPI.getAllClusterTypes(adminCtx, false)
		if err != nil {
			return edgeClusterIDs, err
		}
		for _, edgeCluster := range edgeClusters {
			if !edgeClusterMap[edgeCluster.ID] {
				for _, project := range categoryProjects {
					if model.CategoryMatch(edgeCluster.Labels, project.EdgeSelectors) {
						edgeClusterMap[edgeCluster.ID] = true
						edgeClusterIDs = append(edgeClusterIDs, edgeCluster.ID)
						break
					}
				}
			}
		}
	}
	return edgeClusterIDs, err
}

func (dbAPI *dbObjectModelAPI) getAffiliatedProjectsEdgeIDsMap(ctx context.Context) (map[string]bool, error) {
	edgeMap := map[string]bool{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return edgeMap, err
	}
	projectIDs := auth.GetProjectIDs(authContext)
	if len(projectIDs) == 0 {
		return edgeMap, nil
	}
	edgeIDs, err := dbAPI.GetProjectsEdges(ctx, projectIDs)
	if err != nil {
		return edgeMap, err
	}
	for _, edgeID := range edgeIDs {
		edgeMap[edgeID] = true
	}
	return edgeMap, nil
}

func (dbAPI *dbObjectModelAPI) getAffiliatedProjectsEdgeClusterIDsMap(ctx context.Context) (map[string]bool, error) {
	edgeClusterMap := map[string]bool{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return edgeClusterMap, err
	}
	projectIDs := auth.GetProjectIDs(authContext)
	if len(projectIDs) == 0 {
		return edgeClusterMap, nil
	}
	edgeClusterIDs, err := dbAPI.GetProjectsEdgeClusters(ctx, projectIDs)
	if err != nil {
		return edgeClusterMap, err
	}

	for _, edgeClusterID := range edgeClusterIDs {
		edgeClusterMap[edgeClusterID] = true
	}
	return edgeClusterMap, nil
}

// SelectAffiliatedProjects select all projects in a tenant for which the given context is affiliated with
func (dbAPI *dbObjectModelAPI) SelectAffiliatedProjects(context context.Context) ([]model.Project, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return []model.Project{}, err
	}
	// Paging is not used now, hence the row limit is set to a large value
	projects, err := dbAPI.getProjectsEtag(context, "", "", nil)
	return auth.FilterProjectScopedEntities(projects, authContext).([]model.Project), err
}

func (dbAPI *dbObjectModelAPI) getProjects(context context.Context, projectID string, startPageToken base.PageToken, pageSize int, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Project, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return []model.Project{}, err
	}
	// Paging is not used now, hence the row limit is set to a large value
	projects, err := dbAPI.getProjectsEtag(context, "", projectID, entitiesQueryParam)
	if err != nil {
		return []model.Project{}, err
	}
	// note: we return all projects for edge to simplify project update processing for edge
	if auth.IsInfraAdminOrEdgeRole(authContext) {
		return projects, nil
	}
	return auth.FilterProjectScopedEntities(projects, authContext).([]model.Project), err
}

// SelectAllProjects select all projects for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllProjects(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Project, error) {
	return dbAPI.getProjects(context, "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllProjectsW select all projects for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllProjectsW(context context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	projects, err := dbAPI.SelectAllProjects(context, entitiesQueryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, projects)
}

// SelectAllProjectsWV2 select all projects for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllProjectsWV2(context context.Context, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)

	projects, totalCount, err := dbAPI.getProjectsForQuery(context, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeProject}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.ProjectListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		ProjectList:               projects,
	}
	return json.NewEncoder(w).Encode(r)
}

// GetProject get a project object in the DB
func (dbAPI *dbObjectModelAPI) GetProject(context context.Context, projectID string) (model.Project, error) {
	if len(projectID) == 0 {
		return model.Project{}, errcode.NewBadRequestError("projectID")
	}
	projects, err := dbAPI.getProjects(context, projectID, base.StartPageToken, base.MaxRowsLimit, nil)
	if err != nil {
		return model.Project{}, err
	}
	if len(projects) == 0 {
		return model.Project{}, errcode.NewPermissionDeniedError("RBAC")
	}
	return projects[0], nil
}

func (dbAPI *dbObjectModelAPI) GetProjectName(context context.Context, projectID string) (string, error) {
	return dbAPI.getObjectName(context, "project_model", projectID, "projectID")
}

// GetProjectW get a project object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	project, err := dbAPI.GetProject(context, projectID)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, project)
}

// CreateProject creates a project object in the DB
func (dbAPI *dbObjectModelAPI) CreateProject(context context.Context, i interface{} /* *model.Project */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Project)
	if !ok {
		return resp, errcode.NewInternalError("CreateProject: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateProject doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateProject doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityProject,
		meta.OperationCreate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	err = dbAPI.validateProject(context, &doc)
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	doc.Version = float64(now.UnixNano())
	doc.CreatedAt = now
	doc.UpdatedAt = now
	projectDBO := ProjectDBO{}
	err = base.Convert(&doc, &projectDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := tx.NamedExec(context, queryMap["CreateProject"], &projectDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating project %+v. Error: %s"), projectDBO, err.Error())
			return errcode.TranslateDatabaseError(projectDBO.ID, err)
		}
		err = dbAPI.createProjectUsers(context, tx, &doc)
		if err != nil {
			return err
		}
		err = dbAPI.createProjectDockerProfiles(context, tx, &doc)
		if err != nil {
			return err
		}
		err = dbAPI.createProjectCloudCredss(context, tx, &doc)
		if err != nil {
			return err
		}
		err = dbAPI.createProjectEdges(context, tx, &doc)
		if err != nil {
			return err
		}
		return dbAPI.createProjectEdgeSelectors(context, tx, &doc)
	})
	if err != nil {
		return resp, err
	}
	docs := []model.Project{doc}
	err = dbAPI.populateProjectsAssociations(context, docs)
	if err != nil {
		return resp, err
	}
	doc = docs[0]
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addProjectAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateProjectW creates a project object in the DB
func (dbAPI *dbObjectModelAPI) CreateProjectW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateProject, &model.Project{}, w, r, callback)
}

// CreateProjectWV2 creates a project object in the DB
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateProjectWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateProject), &model.Project{}, w, r, callback)
}

// SelectProjectDataStreamsUsingCloudCreds get all data stream ids in the project which are using
// one of the given cloud profile ids
func (dbAPI *dbObjectModelAPI) SelectProjectDataStreamsUsingCloudCreds(context context.Context, tenantID string, projectID string, cloudCredsIDs []string) ([]string, error) {
	dataStreamIDs := []string{}
	if len(cloudCredsIDs) != 0 {
		dataStreamIDObjs := []IDDBO{}
		param := TenantProjectCloudCredsIdsParam{TenantID: tenantID, ProjectID: projectID, CloudCredsIDs: cloudCredsIDs}
		err := dbAPI.QueryIn(context, &dataStreamIDObjs, queryMap["SelectProjectDataStreamsUsingCloudCreds"], param)
		if err != nil {
			return dataStreamIDs, err
		}
		if len(dataStreamIDObjs) != 0 {
			dataStreamIDs = funk.Map(dataStreamIDObjs, func(x IDDBO) string {
				return x.ID
			}).([]string)
		}
	}
	return dataStreamIDs, nil
}

// UpdateProject updates a project object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateProject(context context.Context, i interface{} /* *model.Project */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Project)
	if !ok {
		return resp, errcode.NewInternalError("UpdateProject: type error")
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
	err = dbAPI.validateProject(context, &doc)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityProject,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	project, err := dbAPI.GetProject(context, doc.ID)
	if err != nil {
		return resp, err
	}

	removedCloudCredsIDs := funk.Filter(project.CloudCredentialIDs, func(id string) bool {
		return !funk.Contains(doc.CloudCredentialIDs, id)
	}).([]string)

	dataStreamIDs, err := dbAPI.SelectProjectDataStreamsUsingCloudCreds(context, tenantID, project.ID, removedCloudCredsIDs)
	if err != nil {
		return resp, err
	}
	if len(dataStreamIDs) != 0 {
		return resp, errcode.NewBadRequestExError("Project.CloudCredsIDs", fmt.Sprintf("Error in updating project[%s]: some cloud profiles %v in use by data pipelines %v", project.Name, removedCloudCredsIDs, dataStreamIDs))
	}

	now := base.RoundedNow()
	doc.Version = float64(now.UnixNano())
	doc.UpdatedAt = now
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		projectDBO := ProjectDBO{}
		err := base.Convert(&doc, &projectDBO)
		if err != nil {
			return err
		}

		_, err = tx.NamedExec(context, queryMap["UpdateProject"], &projectDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in updating project for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}

		_, err = base.DeleteTxn(context, tx, "project_user_model", map[string]interface{}{"project_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting project users for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createProjectUsers(context, tx, &doc)
		if err != nil {
			return err
		}
		_, err = base.DeleteTxn(context, tx, "project_docker_profile_model", map[string]interface{}{"project_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting project docker profiles for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createProjectDockerProfiles(context, tx, &doc)
		if err != nil {
			return err
		}
		_, err = base.DeleteTxn(context, tx, "project_cloud_creds_model", map[string]interface{}{"project_id": doc.ID})
		if err != nil {
			glog.Errorf("Error in deleting project cloud creds for ID %s and tenant ID %s. Error: %s", doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createProjectCloudCredss(context, tx, &doc)
		if err != nil {
			return err
		}
		_, err = base.DeleteTxn(context, tx, "project_edge_model", map[string]interface{}{"project_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting project edges for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createProjectEdges(context, tx, &doc)
		if err != nil {
			return err
		}
		_, err = base.DeleteTxn(context, tx, "project_edge_selector_model", map[string]interface{}{"project_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting project edge selectors for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.deleteInvalidAppEdgeIDsOnProjectEdgeUpdate(context, tx, tenantID, []model.Project{doc}, nil)
		if err != nil {
			return err
		}
		return dbAPI.createProjectEdgeSelectors(context, tx, &doc)
	})
	if err != nil {
		return resp, err
	}
	docs := []model.Project{doc}
	err = dbAPI.populateProjectsAssociations(context, docs)
	if err != nil {
		return resp, err
	}
	doc = docs[0]
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addProjectAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateProjectW updates a project object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateProjectW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateProject, &model.Project{}, w, r, callback)
}

// UpdateProjectWV2 updates a project object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateProjectWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateProject), &model.Project{}, w, r, callback)
}

// DeleteProject delete a project object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteProject(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityProject,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	// saving project object to pass to auditlog
	project, errGetProject := dbAPI.GetProject(context, id)
	doc := model.Project{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}

	result, err := DeleteEntity(context, dbAPI, "project_model", "id", id, doc, callback)
	if err == nil {
		if errGetProject != nil {
			glog.Error("Error in getting project info : ", errGetProject.Error())
		} else {
			GetAuditlogHandler().addProjectAuditLog(dbAPI, context, project, DELETE)
		}
	} else {
		glog.Error("Error in deleting project", context, err.Error())
	}
	return result, err
}

// DeleteProjectW delete a project object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteProjectW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteProject, id, w, callback)
}

// DeleteProjectWV2 delete a project object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteProjectWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteProject), id, w, callback)
}

// checkTenant checks all id in entityIDs in tableName table has tenantID
// matching that in the ctx
func (dbAPI *dbObjectModelAPI) checkTenant(ctx context.Context, tableName string, entityIDs []string) error {
	if len(entityIDs) == 0 {
		return nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	tenantID := authContext.TenantID
	tenantQueryParams := []TenantQueryParam{}
	query := fmt.Sprintf(queryMap["CheckTenantTemplate"], tableName)
	err = dbAPI.QueryIn(ctx, &tenantQueryParams, query, idFilter{IDs: entityIDs})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "CheckTenant: query error, tableName=%s. Error: %s"), tableName, err.Error())
		return err
	}
	if len(entityIDs) != len(tenantQueryParams) {
		err = fmt.Errorf("CheckTenant: missing tenant id: %d", len(entityIDs)-len(tenantQueryParams))
		glog.Errorf(base.PrefixRequestID(ctx, "%s"), err.Error())
		return err
	}
	for _, tenantQueryParam := range tenantQueryParams {
		if tenantQueryParam.TenantID != tenantID {
			err = fmt.Errorf("CheckTenant: bad tenant id: %s, expected TenantID: %s", tenantQueryParam.TenantID, tenantID)
			glog.Errorf(base.PrefixRequestID(ctx, "%s"), err.Error())
			return err
		}
	}
	return nil
}

func createBuiltinProjects(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	now := base.RoundedNow()
	for _, project := range config.BuiltinProjects {
		project.ID = GetDefaultProjectID(tenantID)
		project.TenantID = tenantID
		project.Version = float64(now.UnixNano())
		project.CreatedAt = now
		project.UpdatedAt = now
		projectDBO := ProjectDBO{}
		err := base.Convert(&project, &projectDBO)
		if err != nil {
			return err
		}
		_, err = tx.NamedExec(ctx, queryMap["CreateProject"], &projectDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error creating project %+v. Error: %s"), projectDBO, err.Error())
			return errcode.TranslateDatabaseError(projectDBO.ID, err)
		}
	}
	return nil
}

func deleteBuiltinProjects(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	for _, project := range config.BuiltinProjects {
		id := GetDefaultProjectID(tenantID)
		_, err := base.DeleteTxn(ctx, tx, "project_model", map[string]interface{}{"id": id})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Deleting builtIn project %+v with id %s. Error: %s"), project, id, err.Error())
			return err
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) GetProjectNamesByIDs(ctx context.Context, projectIDs []string) (map[string]string, error) {
	return dbAPI.getNamesByIDs(ctx, "project_model", projectIDs)
}

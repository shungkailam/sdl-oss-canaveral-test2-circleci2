package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/apptemplate"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
	"github.com/jmoiron/sqlx/types"
	funk "github.com/thoas/go-funk"
	yaml "gopkg.in/yaml.v2"
)

const (
	entityTypeApplication = "application"
	maxInEndpoints        = 1
	maxOutEndpoints       = 1
	helmChartDir          = "/helmApp"
)

var (
	errHelmWaiting = fmt.Errorf("Waiting for chart files to be uploaded")
)

func init() {
	// If state is null, it is DEPLOY by default
	queryMap["SelectApplicationsTemplate"] = `SELECT id, version, tenant_id, description, name, yaml_data, created_at, updated_at, project_id, state, only_pre_pull_on_update, packaging_type, helm_metadata FROM application_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND project_id IN (:project_ids) AND (:state = '' OR state = :state OR (state is null AND :state = 'DEPLOY')) %s`
	// If state is null, it is DEPLOY by default
	queryMap["SelectApplicationsByProjectsTemplate"] = `SELECT id, version, tenant_id, description, name, yaml_data, created_at, updated_at, project_id, state, only_pre_pull_on_update, packaging_type, helm_metadata, count(*) OVER() as total_count FROM application_model WHERE tenant_id = :tenant_id AND project_id IN (:project_ids)  AND (:id = '' OR id = :id) AND (:state = '' OR state = :state OR (state is null AND :state = 'DEPLOY')) %s`
	queryMap["CreateApplication"] = `INSERT INTO application_model (id, version, tenant_id, name, description, yaml_data, project_id, packaging_type, helm_metadata, created_at, updated_at, state) VALUES (:id, :version, :tenant_id, :name, :description, :yaml_data, :project_id, :packaging_type, :helm_metadata, :created_at, :updated_at, :state)`
	queryMap["UpdateApplication"] = `UPDATE application_model SET version = :version, tenant_id = :tenant_id, name = :name, description = :description, yaml_data = :yaml_data, project_id = :project_id, packaging_type = :packaging_type, helm_metadata = :helm_metadata, updated_at = :updated_at, state = :state, only_pre_pull_on_update = :only_pre_pull_on_update WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["CreateApplicationEdge"] = `INSERT INTO application_edge_model (application_id, edge_id, state) VALUES (:application_id, :edge_id, :state)`
	queryMap["SelectApplicationEdges"] = `SELECT * FROM application_edge_model WHERE application_id = :application_id`
	queryMap["SelectApplicationsEdges"] = `SELECT * FROM application_edge_model WHERE application_id IN (:application_ids)`
	queryMap["CreateApplicationEdgeSelector"] = `INSERT INTO application_edge_selector_model (application_id, category_value_id) VALUES (:application_id, :category_value_id)`
	queryMap["SelectApplicationsEdgeSelectors"] = `SELECT application_edge_selector_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
	  FROM application_edge_selector_model JOIN category_value_model ON application_edge_selector_model.category_value_id = category_value_model.id WHERE application_edge_selector_model.application_id IN (:application_ids)`
	queryMap["SelectApplicationOriginSelectors"] = `SELECT application_origin_model.*, category_value_model.category_id "category_info.id", category_value_model.value "category_info.value"
	  FROM application_origin_model JOIN category_value_model ON application_origin_model.category_value_id = category_value_model.id WHERE application_origin_model.application_id IN (:application_ids)`
	queryMap["DeleteProjectApplicationsEdges"] = "delete from application_edge_model where application_id IN (select id from application_model where project_id = :project_id)"
	queryMap["DeleteProjectApplicationsEdgesList"] = "delete from application_edge_model where application_id IN (select id from application_model where project_id = :project_id) AND edge_id IN (:edge_ids)"
	queryMap["DeleteProjectApplicationsEdgeSelectors"] = "delete from application_edge_selector_model where application_id IN (select id from application_model where project_id = :project_id)"
	orderByHelper.Setup(entityTypeApplication, []string{"id", "version", "created_at", "updated_at", "name", "description", "project_id", "yaml_data"})
	queryMap["SelectDataIfcClaimByApplicationIDs"] = `SELECT data_source_id, application_id, tenant_id, topic FROM data_source_topic_claim WHERE application_id IN (:application_ids)`
	queryMap["SelectApplicationProject"] = `SELECT id, project_id from application_model WHERE tenant_id = :tenant_id AND project_id IN (:project_ids)`
	queryMap["DeleteApplicationEdgesTemplate"] = `DELETE FROM application_edge_model WHERE application_id='%s' AND edge_id IN ('%s')`
	queryMap["DeleteAllApplicationEdgesTemplate"] = `DELETE FROM application_edge_model WHERE application_id IN ('%s')`
	queryMap["DeleteApplicationsEdgeSelectorsTemplate"] = "DELETE FROM application_edge_selector_model WHERE application_id IN (SELECT id FROM application_model WHERE project_id = '%s')"
}

// ApplicationDBO is DB object model for application
type ApplicationDBO struct {
	model.BaseModelDBO
	Name                string          `json:"name" db:"name"`
	Description         string          `json:"description" db:"description"`
	YamlData            string          `json:"yamlData" db:"yaml_data"`
	ProjectID           *string         `json:"projectId" db:"project_id"`
	State               *string         `json:"state" db:"state"`
	OnlyPrePullOnUpdate bool            `json:"onlyPrePullOnUpdate" db:"only_pre_pull_on_update"`
	PackagingType       *string         `json:"packagingType,omitempty" db:"packaging_type"`
	HelmMetadata        *types.JSONText `json:"helmMetadata,omitempty" db:"helm_metadata"`
}

type ApplicationEdgeDBO struct {
	ID            int64  `json:"id" db:"id"`
	ApplicationID string `json:"applicationId" db:"application_id"`
	EdgeID        string `json:"edgeId" db:"edge_id"`
	State         string `json:"state" db:"state"`
}

type ApplicationIdsParam struct {
	ApplicationIDs []string `json:"applicationIds" db:"application_ids"`
}

type ApplicationEdgeSelectorDBO struct {
	model.CategoryInfo `json:"categoryInfo" db:"category_info"`
	ID                 int64  `json:"id" db:"id"`
	ApplicationID      string `json:"applicationId" db:"application_id"`
	CategoryValueID    int64  `json:"categoryValueId" db:"category_value_id"`
}

type Metadata struct {
	Name string `yaml:"name"`
}
type k8sYamlBase struct {
	Metadata `yaml:"metadata"`
	Kind     string `yaml:"kind"`
}
type nameKind struct {
	name string
	kind string
}

func (app ApplicationDBO) GetProjectID() string {
	if app.ProjectID != nil {
		return *app.ProjectID
	}
	return ""
}

type ApplicationProjects struct {
	ApplicationDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for application
func getApplicationDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	isEdgeReq, _ := base.IsEdgeRequest(authContext)
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	// State pointer must be set for query to work
	param := ApplicationDBO{BaseModelDBO: tenantModel, State: base.StringPtr("")}
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
		Param: ApplicationProjects{
			ApplicationDBO: param,
			ProjectIDs:     projectIDs,
		},
		Key:     "SelectApplicationsTemplate",
		InQuery: true,
	}, nil
}

// detectLineBreak returns the relevant platform specific line ending
func detectLineBreak(yaml string) string {
	windowsLineEnding := strings.Contains(yaml, "\r\n")
	if windowsLineEnding && runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

func validateUniqueNameAndKind(isBeingVerified bool, appYaml, appID, curAppID string, k8sObjsMap map[nameKind]bool) error {
	// appID is empty if isBeingVerified is true
	// Do not check for duplicate names for the sane application in case of updates
	if appID == curAppID {
		return nil
	}
	curAppObjs := strings.Split(appYaml, detectLineBreak(appYaml)+"---"+detectLineBreak(appYaml))
	for _, curAppObj := range curAppObjs {
		var curAppObjStruct k8sYamlBase
		err := yaml.Unmarshal([]byte(curAppObj), &curAppObjStruct)
		if err != nil {
			return errcode.NewBadRequestExError("YamlData", fmt.Sprintf("The yaml could not be decoded, Error: %s", err.Error()))
		}
		curAppObjNameKind := nameKind{name: curAppObjStruct.Metadata.Name, kind: curAppObjStruct.Kind}
		if exists := k8sObjsMap[curAppObjNameKind]; exists {
			if isBeingVerified == true {
				return errcode.NewBadRequestExError("YamlData", fmt.Sprintf("The yaml has two objects with name %s "+
					"and kind %s", curAppObjNameKind.name, curAppObjNameKind.kind))
			}
			return errcode.NewBadRequestExError("YamlData", fmt.Sprintf("Another app exists with name %s and kind %s"+
				" in this project", curAppObjNameKind.name, curAppObjNameKind.kind))

		}
		if isBeingVerified == true {
			k8sObjsMap[curAppObjNameKind] = true
		}
	}
	return nil
}

// ValidateDataIfcEndpointsCountLimits validates the given list of data sources contains upto the given number of
// number of data ifcs of each kind
func ValidateDataIfcEndpointsCountLimits(context context.Context, datasources []model.DataSource, limitIns, limitOuts int) error {
	var loggedErr error
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()

	numOuts, numIns := 0, 0
	for _, ds := range datasources {
		if ds.IfcInfo == nil {
			continue
		}
		if ds.IfcInfo.Kind == model.DataIfcEndpointKindOut {
			numOuts++
		} else if ds.IfcInfo.Kind == model.DataIfcEndpointKindIn {
			numIns++
		} else {
			loggedErr = fmt.Errorf("unexpected kind %s for endpoint ID %s", ds.IfcInfo.Kind, ds.ID)
			return loggedErr
		}
	}

	if numOuts > limitOuts {
		loggedErr = fmt.Errorf("expected to find less than %d data sources with Ifc kind OUT, but found %d", limitOuts, numOuts)
		return loggedErr
	}

	if numIns > limitIns {
		loggedErr = fmt.Errorf("expected to find less than %d data sources with Ifc kind IN, but found %d", limitIns, numIns)
		return loggedErr
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) enabledServicesInProject(ctx context.Context, projID string) (map[string]bool, error) {
	res := make(map[string]bool)
	entitiesQueryParam := &model.EntitiesQueryParam{}
	queryParam := &model.ServiceBindingQueryParam{
		BindResourceType: string(model.ServiceBindingProjectResource),
		BindResourceID:   projID,
	}
	svcBindings, err := dbAPI.SelectAllServiceBindings(ctx, entitiesQueryParam, queryParam)
	if err != nil {
		return nil, err
	}
	for _, binding := range svcBindings.SvcBindingList {
		if binding.BindResource != nil && binding.BindResource.ID == projID {
			res[binding.Type] = true
		}
	}
	serviceInstanceQueryParam := &model.ServiceInstanceQueryParam{
		ServiceClassCommonQueryParam: model.ServiceClassCommonQueryParam{
			Scope: model.ServiceClassProjectScope,
		},
		ScopeID: projID,
	}
	svcInstances, err := dbAPI.SelectAllServiceInstances(ctx, entitiesQueryParam, serviceInstanceQueryParam)
	if err != nil {
		return nil, err
	}
	for _, instance := range svcInstances.SvcInstanceList {
		if instance.ScopeID == projID {
			res[instance.Type] = true
		}
	}
	return res, nil
}

func (dbAPI *dbObjectModelAPI) validateAppTemplate(ctx context.Context, app *model.Application) error {
	var (
		// This list must be consistent with what's assumed by edge (edgemgmt/services/services.go)
		KafkaServiceName     = "Kafka"
		ZookeeperServiceName = "KafkaZookeeper"

		// Types for which to check whether binding is use by app
		KafkaServiceType = "kafka"

		svcTypes = map[string]string{
			KafkaServiceName:     KafkaServiceType,
			ZookeeperServiceName: KafkaServiceType,
		}

		enabledServices = map[string]bool{}
		err             error
	)

	if enabledServices, err = dbAPI.enabledServicesInProject(ctx, app.ProjectID); err != nil {
		return err
	}
	edgeServices := map[string]apptemplate.EdgeService{
		KafkaServiceName:     {Endpoint: "kafka"},    // any value will do
		ZookeeperServiceName: {Endpoint: "kafka-zk"}, // any value will do
	}
	// Check whether service is referenced by YAML via template parameter
	_, svcNames, err := apptemplate.RenderWithParams(&apptemplate.AppParameters{
		EdgeParameters: apptemplate.EdgeParameters{
			Services: edgeServices,
		},
	}, app.YamlData)
	if err != nil {
		return err
	}
	for _, svcName := range svcNames {
		// Map service name to correspoinding service type of binding.
		t, ok := svcTypes[svcName]
		// We don't care about this service ;)
		if !ok {
			continue
		}
		// Has this particular service type been enabled in project?
		if !enabledServices[t] {
			return fmt.Errorf("%s service not enabled for project", t)
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) validateApplication(context context.Context, doc *model.Application) error {
	project, err := dbAPI.GetProject(context, doc.ProjectID)
	loggedErr := err
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()

	if err != nil {
		loggedErr = fmt.Errorf("*** validateApplication: GetProject failed for project %s, ctx: %+v\n", doc.ProjectID, context)
		return errcode.NewBadRequestError("projectId")
	}
	if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
		doc.EdgeIDs = nil
		// edgeSelectors validation are done in createApplicationEdgeSelectors
	} else {
		doc.EdgeIDs = base.Unique(doc.EdgeIDs)
		// doc.EdgeIDs must be a subset of project.EdgeIDs
		badEdgeIDs := []string{}
		for _, edgeID := range doc.EdgeIDs {
			if !funk.Contains(project.EdgeIDs, edgeID) {
				badEdgeIDs = append(badEdgeIDs, edgeID)
			}
		}
		if len(badEdgeIDs) != 0 {
			return errcode.NewBadRequestExError("edgeIds", fmt.Sprintf("Edges with IDs %s are not part of the project", strings.Join(badEdgeIDs, ", ")))
		}
		doc.EdgeSelectors = nil
	}
	if err := dbAPI.validateAppTemplate(context, doc); err != nil {
		return errcode.NewBadRequestExError("YamlData", err.Error())
	}

	// TODO: Validate that the data sources are deployed on edges, where this project is deployed
	datasources, err := dbAPI.SelectDataSourcesByEndpoints(context, doc.DataIfcEndpoints)
	if err != nil {
		loggedErr = fmt.Errorf("failed to validate endpoints for application %s(id=%s). %s", doc.Name, doc.ID, err.Error())
		return errcode.NewInternalError(loggedErr.Error())
	}

	loggedErr = ValidateDataIfcEndpointsCountLimits(context, datasources, maxInEndpoints, maxOutEndpoints)
	if loggedErr != nil {
		return errcode.NewBadRequestExError("DataIfcEndpoints", loggedErr.Error())
	}
	// TODO: Uncomment when compass has done the required changes https://jira.nutanix.com/browse/ENG-185779
	// Validate that the name of the application does not collide with other apps in the project
	// k8sObjsMap := make(map[nameKind]bool)
	// err = validateUniqueNameAndKind(true, doc.YamlData, "", doc.ID, k8sObjsMap)
	// if err != nil {
	// 	return err
	// }
	// apps, err := dbAPI.SelectAllApplicationsForProject(context, doc.ProjectID)
	// if err != nil {
	// 	return errcode.NewInternalError(fmt.Sprintf("Could not retrive applications from database %s. Error: %s", *config.Cfg.SQL_DB, err.Error()))
	// }
	// for _, app := range apps {
	// 	err = validateUniqueNameAndKind(false, app.YamlData, app.ID, doc.ID, k8sObjsMap)
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	return nil
}

func (dbAPI *dbObjectModelAPI) createApplicationEdges(ctx context.Context, tx *base.WrappedTx, doc *model.Application) error {
	applicationEdgeDBOMap := map[string]*ApplicationEdgeDBO{}
	for _, edgeID := range doc.EdgeIDs {
		applicationEdgeDBOMap[edgeID] = &ApplicationEdgeDBO{ApplicationID: doc.ID, EdgeID: edgeID, State: string(model.DeployEntityState)}
	}
	for _, edgeID := range doc.ExcludeEdgeIDs {
		applicationEdgeDBO, ok := applicationEdgeDBOMap[edgeID]
		if ok {
			// Override
			applicationEdgeDBO.State = string(model.UndeployEntityState)
		} else {
			applicationEdgeDBOMap[edgeID] = &ApplicationEdgeDBO{ApplicationID: doc.ID, EdgeID: edgeID, State: string(model.UndeployEntityState)}
		}
	}
	for _, applicationEdgeDBO := range applicationEdgeDBOMap {
		// The DB ID is generated
		_, err := tx.NamedExec(ctx, queryMap["CreateApplicationEdge"], applicationEdgeDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error creating application edge %+v. Error: %s"), applicationEdgeDBO, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
	}
	return nil
}

// TODO FIXME - make this method generic
func (dbAPI *dbObjectModelAPI) createApplicationEdgeSelectors(ctx context.Context, tx *base.WrappedTx, application *model.Application) error {
	for _, categoryInfo := range application.EdgeSelectors {
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
				applicationEdgeSelectorDBO := ApplicationEdgeSelectorDBO{ApplicationID: application.ID, CategoryValueID: categoryValueDBO.ID}
				_, err = tx.NamedExec(ctx, queryMap["CreateApplicationEdgeSelector"], &applicationEdgeSelectorDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error occurred while creating application edge selector for ID %s. Error: %s"), application.ID, err.Error())
					return errcode.TranslateDatabaseError(application.ID, err)
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

func (dbAPI *dbObjectModelAPI) populateApplicationOriginSelectors(ctx context.Context, applications []model.Application) error {
	if !*config.Cfg.EnableAppOriginSelectors {
		return nil
	}

	if len(applications) == 0 {
		glog.V(5).Infof(base.PrefixRequestID(ctx, "skipping populating origin selectors"))
		return nil
	}

	applicationIdsParam := ApplicationIdsParam{ApplicationIDs: make([]string, 0, len(applications))}
	for _, app := range applications {
		applicationIdsParam.ApplicationIDs = append(applicationIdsParam.ApplicationIDs, app.ID)
	}
	selectors := []ApplicationEdgeSelectorDBO{}
	glog.V(5).Infof(base.PrefixRequestID(ctx, "Selecting application origin selectors for applications: %s"), strings.Join(applicationIdsParam.ApplicationIDs, ","))
	err := dbAPI.QueryIn(ctx, &selectors, queryMap["SelectApplicationOriginSelectors"], applicationIdsParam)
	if err != nil {
		return err
	}

	// save it in a map such that each selector can be matched to its corresponding application
	selectorsByAppID := make(map[string][]model.CategoryInfo)
	for _, sel := range selectors {
		selectorsByAppID[sel.ApplicationID] = append(selectorsByAppID[sel.ApplicationID], sel.CategoryInfo)
	}
	glog.V(5).Infof("found selectors for %+v", selectorsByAppID)
	for i, app := range applications {
		glog.V(5).Infof("Setting origin selector for app: %s", app.ID)
		if v, ok := selectorsByAppID[app.ID]; ok {
			applications[i].OriginSelectors = &v
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) populateApplicationEndpoints(ctx context.Context, applications []model.Application) error {
	if len(applications) == 0 {
		glog.V(5).Info(base.PrefixRequestID(ctx, "skipping populating application endpoints"))
		return nil
	}

	appIdxs := make(map[string]int)
	appIDs := make([]string, 0, len(applications))
	for i, app := range applications {
		appIdxs[app.ID] = i
		appIDs = append(appIDs, app.ID)
	}

	glog.V(5).Infof(base.PrefixRequestID(ctx, "populating application endpoints for apps %v"), appIDs)
	endpointsByID, err := dbAPI.FetchApplicationsEndpoints(ctx, appIDs)
	if err != nil {
		return err
	}

	for id, endpoints := range endpointsByID {
		applications[appIdxs[id]].DataIfcEndpoints = endpoints
	}
	glog.V(5).Infof(base.PrefixRequestID(ctx, "successfully populated application endpoints for apps %v"), appIDs)

	return nil
}

// resolveApplicationEdges resolves the application edges for applications for category based selection of edges
func (dbAPI *dbObjectModelAPI) resolveApplicationEdges(ctx context.Context, tenantID string, application *model.Application) error {
	projects, err := dbAPI.getProjectsByIDs(ctx, tenantID, []string{application.ProjectID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting project %+v. Error: %s"), application.ProjectID, err.Error())
		return err
	}
	if len(projects) == 0 {
		return errcode.NewRecordNotFoundError("project")
	}
	application.EdgeIDs = base.Unique(application.EdgeIDs)
	excludeEdgeIDsMap := map[string]bool{}
	// ExcludeEdgeIDs may have some IDs which are not in the category selection.
	// These invalid ones are filtered out before saving to DB
	for _, excludeEdgeID := range application.ExcludeEdgeIDs {
		excludeEdgeIDsMap[excludeEdgeID] = false
	}
	project := projects[0]
	if project.EdgeSelectorType != model.ProjectEdgeSelectorTypeCategory {
		application.ExcludeEdgeIDs = base.Unique(application.ExcludeEdgeIDs)
		return nil
	}
	edgeClusterIDLabelsList, err := dbAPI.SelectEdgeClusterIDLabels(ctx)
	if err != nil {
		return err
	}
	application.EdgeIDs = []string{}
	edgeSelectors := model.CategoryAnd(application.EdgeSelectors, project.EdgeSelectors)
	for _, edgeClusterIDLabels := range edgeClusterIDLabelsList {
		if model.CategoryMatch(edgeClusterIDLabels.Labels, edgeSelectors) {
			if _, ok := excludeEdgeIDsMap[edgeClusterIDLabels.ID]; ok {
				// ID is valid
				excludeEdgeIDsMap[edgeClusterIDLabels.ID] = true
			} else {
				application.EdgeIDs = append(application.EdgeIDs, edgeClusterIDLabels.ID)
			}
		}
	}
	application.ExcludeEdgeIDs = []string{}
	for excludeEdgeID, valid := range excludeEdgeIDsMap {
		if valid {
			application.ExcludeEdgeIDs = append(application.ExcludeEdgeIDs, excludeEdgeID)
		}
	}
	return nil
}

// deleteInvalidAppEdgeIDsOnProjectEdgeUpdate removes invalid service domain IDs from the application edge exclusion list if the ID becomes invalid for the application.
// edgeClusterIDLabelsList is optional.
func (dbAPI *dbObjectModelAPI) deleteInvalidAppEdgeIDsOnProjectEdgeUpdate(ctx context.Context, tx *base.WrappedTx, tenantID string, updatedProjects []model.Project, edgeClusterIDLabelsList []model.EdgeClusterIDLabels) error {
	if updatedProjects == nil || len(updatedProjects) == 0 {
		return nil
	}
	tenantProjectIDsParam := TenantProjectIdsParam{
		TenantID:   tenantID,
		ProjectIDs: []string{},
	}

	// Get all the existing projects
	projectIDs := funk.Map(updatedProjects, func(project model.Project) string { return project.ID }).([]string)
	projects, err := dbAPI.getProjectsByIDs(ctx, tenantID, projectIDs)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting project %+v. Error: %s"), projectIDs, err.Error())
		return err
	}
	if len(projects) == 0 {
		return errcode.NewRecordNotFoundError("project")
	}

	projectMap := make(map[string]*model.Project, len(updatedProjects))
	for i := range projects {
		project := &projects[i]
		projectMap[project.ID] = project
		tenantProjectIDsParam.ProjectIDs = append(tenantProjectIDsParam.ProjectIDs, project.ID)
	}

	// Get all the applications in the projects
	applicationDBOs := []ApplicationDBO{}
	err = dbAPI.QueryIn(ctx, &applicationDBOs, queryMap["SelectApplicationProject"], tenantProjectIDsParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting applications for projects %+v. Error: %s"), tenantProjectIDsParam.ProjectIDs, err.Error())
		return err
	}
	if len(applicationDBOs) == 0 {
		glog.V(5).Infof(base.PrefixRequestID(ctx, "No application found for projects %+v"), tenantProjectIDsParam.ProjectIDs)
		// Nothing to update
		return nil
	}
	projectApplicationDBOsMap := map[string][]*ApplicationDBO{}
	for i := range applicationDBOs {
		applicationDBO := &applicationDBOs[i]
		projectApplicationDBOsMap[*applicationDBO.ProjectID] = append(projectApplicationDBOsMap[*applicationDBO.ProjectID], applicationDBO)
	}
	categoryApplicationDBOs := []*ApplicationDBO{}
	for _, project := range updatedProjects {
		existingProject, ok := projectMap[project.ID]
		if !ok {
			// Only project changes are handled
			continue
		}
		applicationDBOs, ok := projectApplicationDBOsMap[project.ID]
		if !ok {
			// No associated applications are found
			continue
		}
		if project.EdgeSelectorType != existingProject.EdgeSelectorType {
			// Any project edge selector change cleans the previous application to edge relationships.
			// Delete all the project edge associations for the applications
			applicationIDs := funk.Map(applicationDBOs, func(applicationDBO *ApplicationDBO) string { return applicationDBO.ID }).([]string)
			deleteQuery := fmt.Sprintf(queryMap["DeleteAllApplicationEdgesTemplate"], strings.Join(applicationIDs, "','"))
			_, err := tx.QueryxContext(ctx, deleteQuery)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting application edges for applications %+v. Error: %s"), applicationIDs, err.Error())
				return errcode.TranslateDatabaseError(project.ID, err)
			}
			if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeExplicit {
				// Application inherits the edge selectors from the project.
				// Delete the selector information
				deleteQuery := fmt.Sprintf(queryMap["DeleteApplicationsEdgeSelectorsTemplate"], project.ID)
				_, err := tx.QueryxContext(ctx, deleteQuery)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting application selectors for project %+v. Error: %s"), project.ID, err.Error())
					return errcode.TranslateDatabaseError(project.ID, err)
				}
			}
		} else if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			// Add the category based application and process later as some single queries
			// can be executed instead of doing multiple times in a loop
			categoryApplicationDBOs = append(categoryApplicationDBOs, applicationDBOs...)
		} else if project.EdgeSelectorType == model.ProjectEdgeSelectorTypeExplicit {
			// Project might have removed some edges.
			// Find and delete those removed edges from application edge relationship
			updatedProjectEdges := map[string]bool{}
			for _, edgeID := range project.EdgeIDs {
				updatedProjectEdges[edgeID] = true
			}
			removedEdgeIDs := []string{}
			for _, edgeID := range existingProject.EdgeIDs {
				if !updatedProjectEdges[edgeID] {
					removedEdgeIDs = append(removedEdgeIDs, edgeID)
				}
			}
			if len(removedEdgeIDs) > 0 {
				for _, applicationDBO := range applicationDBOs {
					deleteQuery := fmt.Sprintf(queryMap["DeleteApplicationEdgesTemplate"], applicationDBO.ID, strings.Join(removedEdgeIDs, "','"))
					_, err := tx.QueryxContext(ctx, deleteQuery)
					if err != nil {
						glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting application edges for application %s. Error: %s"), applicationDBO.ID, err.Error())
						return errcode.TranslateDatabaseError(applicationDBO.ID, err)
					}
				}
			}
		}
	}
	if len(categoryApplicationDBOs) == 0 {
		glog.V(5).Info(base.PrefixRequestID(ctx, "No project with category based selection found"))
		return nil
	}
	// Now update the deploy/undeploy states by recalculating the categories
	// Get the application IDs to get the application edge selectors
	categoryApplicationIDs := funk.Map(categoryApplicationDBOs, func(applicationDBO *ApplicationDBO) string { return applicationDBO.ID }).([]string)
	applicationEdgeSelectorDBOs := []ApplicationEdgeSelectorDBO{}
	err = dbAPI.QueryIn(ctx, &applicationEdgeSelectorDBOs, queryMap["SelectApplicationsEdgeSelectors"], ApplicationIdsParam{
		ApplicationIDs: categoryApplicationIDs,
	})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting application edge selectors for applications %+v. Error: %s"), categoryApplicationIDs, err.Error())
		return err
	}
	// Create application ID to edge selectors mapping
	applicationEdgeSelectorsMap := map[string]([]model.CategoryInfo){}
	for _, applicationEdgeSelectorDBO := range applicationEdgeSelectorDBOs {
		applicationEdgeSelectorsMap[applicationEdgeSelectorDBO.ApplicationID] = append(applicationEdgeSelectorsMap[applicationEdgeSelectorDBO.ApplicationID], applicationEdgeSelectorDBO.CategoryInfo)
	}

	if edgeClusterIDLabelsList == nil || len(edgeClusterIDLabelsList) == 0 {
		// Get all the labels on all the edges
		edgeClusterIDLabelsList, err = dbAPI.SelectEdgeClusterIDLabels(ctx)
		if err != nil {
			return err
		}
	}
	for _, applicationDBO := range categoryApplicationDBOs {
		project := projectMap[*applicationDBO.ProjectID]
		applicationEdgeSelectors := applicationEdgeSelectorsMap[applicationDBO.ID]
		edgeSelectors := model.CategoryAnd(applicationEdgeSelectors, project.EdgeSelectors)
		invalidEdgeIds := []string{}
		for _, edgeClusterIDLabels := range edgeClusterIDLabelsList {
			if !model.CategoryMatch(edgeClusterIDLabels.Labels, edgeSelectors) {
				invalidEdgeIds = append(invalidEdgeIds, edgeClusterIDLabels.ID)
			}
		}
		if len(invalidEdgeIds) > 0 {
			deleteQuery := fmt.Sprintf(queryMap["DeleteApplicationEdgesTemplate"], applicationDBO.ID, strings.Join(invalidEdgeIds, "','"))
			_, err := tx.QueryxContext(ctx, deleteQuery)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting application edges for application %s. Error: %s"), applicationDBO.ID, err.Error())
				return errcode.TranslateDatabaseError(applicationDBO.ID, err)
			}
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) populateApplicationsEdgesAndSelectors(ctx context.Context, tenantID string, applications []model.Application) error {
	if len(applications) == 0 {
		return nil
	}
	// app ID -> EdgeIDs
	applicationEdgeIDsMap := map[string]([]string){}

	// build project map: project id -> project
	projectIDs := funk.Map(applications, func(application model.Application) string { return application.ProjectID }).([]string)
	projects, err := dbAPI.getProjectsByIDs(ctx, tenantID, projectIDs)
	if err != nil {
		return err
	}
	projectMap := map[string]model.Project{}
	for _, project := range projects {
		projectMap[project.ID] = project
	}
	// Get all the application edges irrespective of category or explicit applications
	applicationEdgeDBOs := []ApplicationEdgeDBO{}
	applicationIDs := funk.Map(applications, func(application model.Application) string { return application.ID }).([]string)
	err = dbAPI.QueryIn(ctx, &applicationEdgeDBOs, queryMap["SelectApplicationsEdges"], ApplicationIdsParam{
		ApplicationIDs: applicationIDs,
	})
	if err != nil {
		return err
	}
	applicationEdgesMap := map[string]map[string]string{}
	for i := range applicationEdgeDBOs {
		applicationEdgeDBO := &applicationEdgeDBOs[i]
		applicationEdges, ok := applicationEdgesMap[applicationEdgeDBO.ApplicationID]
		if !ok {
			applicationEdges = map[string]string{}
			applicationEdgesMap[applicationEdgeDBO.ApplicationID] = applicationEdges
		}
		applicationEdges[applicationEdgeDBO.EdgeID] = applicationEdgeDBO.State
	}

	// application by project edgeSelectorType: Explicit vs Category
	categoryApplications := []*model.Application{}
	explicitApplications := []*model.Application{}
	for i := 0; i < len(applications); i++ {
		application := &applications[i]
		if projectMap[application.ProjectID].EdgeSelectorType == model.ProjectEdgeSelectorTypeCategory {
			categoryApplications = append(categoryApplications, application)
		} else {
			explicitApplications = append(explicitApplications, application)
		}
	}
	// ** populate edgeSelectors
	// must do this first as populate EdgeIDs depends on edgeSelectors
	if len(categoryApplications) != 0 {
		applicationEdgeSelectorDBOs := []ApplicationEdgeSelectorDBO{}
		applicationIDs := funk.Map(categoryApplications, func(application *model.Application) string { return application.ID }).([]string)
		err := dbAPI.QueryIn(ctx, &applicationEdgeSelectorDBOs, queryMap["SelectApplicationsEdgeSelectors"], ApplicationIdsParam{
			ApplicationIDs: applicationIDs,
		})
		if err != nil {
			return err
		}
		applicationEdgeSelectorsMap := map[string]([]model.CategoryInfo){}
		for _, applicationEdgeSelectorDBO := range applicationEdgeSelectorDBOs {
			applicationEdgeSelectorsMap[applicationEdgeSelectorDBO.ApplicationID] = append(applicationEdgeSelectorsMap[applicationEdgeSelectorDBO.ApplicationID], applicationEdgeSelectorDBO.CategoryInfo)
		}
		for _, application := range categoryApplications {
			application.EdgeSelectors = applicationEdgeSelectorsMap[application.ID]
		}
	}

	// ** populate edgeIDs
	// 1. Explicit
	if len(explicitApplications) != 0 {
		applicationIDs := funk.Map(explicitApplications, func(application *model.Application) string { return application.ID }).([]string)
		applicationEdgeDBOs := []ApplicationEdgeDBO{}
		err := dbAPI.QueryIn(ctx, &applicationEdgeDBOs, queryMap["SelectApplicationsEdges"], ApplicationIdsParam{
			ApplicationIDs: applicationIDs,
		})
		if err != nil {
			return err
		}
		for _, applicationEdgeDBO := range applicationEdgeDBOs {
			applicationEdgeIDsMap[applicationEdgeDBO.ApplicationID] = append(applicationEdgeIDsMap[applicationEdgeDBO.ApplicationID], applicationEdgeDBO.EdgeID)
		}
	}
	// 2. Category
	if len(categoryApplications) != 0 {
		edgeClusterIDLabelsList, err := dbAPI.SelectEdgeClusterIDLabels(ctx)
		if err != nil {
			return err
		}
		for _, application := range categoryApplications {
			project := projectMap[application.ProjectID]
			edgeSelectors := model.CategoryAnd(application.EdgeSelectors, project.EdgeSelectors)
			for _, edgeClusterIDLabels := range edgeClusterIDLabelsList {
				if model.CategoryMatch(edgeClusterIDLabels.Labels, edgeSelectors) {
					applicationEdgeIDsMap[application.ID] = append(applicationEdgeIDsMap[application.ID], edgeClusterIDLabels.ID)
				}
			}
		}
	}
	// now set application EdgeIDs
	for i := 0; i < len(applications); i++ {
		application := &applications[i]
		application.EdgeIDs = nil
		application.ExcludeEdgeIDs = nil
		assignedEdges := applicationEdgeIDsMap[application.ID]
		for _, edgeID := range assignedEdges {
			if applicationEdges, ok := applicationEdgesMap[application.ID]; ok {
				entityState, ok := applicationEdges[edgeID]
				if !ok || entityState == string(model.DeployEntityState) {
					// Not excluded
					application.EdgeIDs = append(application.EdgeIDs, edgeID)
				} else {
					application.ExcludeEdgeIDs = append(application.ExcludeEdgeIDs, edgeID)
				}
			} else {
				// Category based selection
				application.EdgeIDs = append(application.EdgeIDs, edgeID)
			}
		}
	}
	return nil
}

// internal api used by getApplicationsWV2
func (dbAPI *dbObjectModelAPI) getApplicationsByProjectsForQuery(context context.Context, projectIDs []string, applicationID string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.Application, int, error) {
	applications := []model.Application{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return applications, 0, err
	}
	isEdgeReq, _ := base.IsEdgeRequest(authContext)
	tenantID := authContext.TenantID
	applicationDBOs := []ApplicationDBO{}
	query, err := buildLimitQuery(entityTypeApplication, queryMap["SelectApplicationsByProjectsTemplate"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return applications, 0, err
	}
	param := tenantIDParam5{TenantID: tenantID, ProjectIDs: projectIDs, ID: applicationID}
	if isEdgeReq {
		param.State = string(model.DeployEntityState)
	}
	err = dbAPI.QueryIn(context, &applicationDBOs, query, param)
	if err != nil {
		return applications, 0, err
	}
	if len(applicationDBOs) == 0 {
		return applications, 0, nil
	}
	totalCount := 0
	first := true
	for _, applicationDBO := range applicationDBOs {
		application := model.Application{}
		if first {
			first = false
			if applicationDBO.TotalCount != nil {
				totalCount = *applicationDBO.TotalCount
			}
		}
		err := base.Convert(&applicationDBO, &application)
		if err != nil {
			return []model.Application{}, 0, err
		}
		applications = append(applications, application)
	}
	err = dbAPI.populateApplicationsEdgesAndSelectors(context, tenantID, applications)
	if err == nil {
		err = dbAPI.populateApplicationOriginSelectors(context, applications)
	}

	if err == nil {
		err = dbAPI.populateApplicationEndpoints(context, applications)
	}
	NewApps(applications).RenderForContext(authContext, dbAPI)
	return applications, totalCount, err
}

// internal api for old public W apis
func (dbAPI *dbObjectModelAPI) getApplicationsEtag(
	context context.Context,
	etag, applicationID, projectID string,
	entitiesQueryParam *model.EntitiesQueryParamV1,
	renderApp bool,
) ([]model.Application, error) {
	applications := []model.Application{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return applications, err
	}
	tenantID := authContext.TenantID
	applicationDBOs := []ApplicationDBO{}
	dbQueryParam, err := getApplicationDBQueryParam(context, projectID, applicationID)
	if err != nil {
		return applications, err
	}
	if dbQueryParam.Key == "" {
		return applications, nil
	}
	query, err := buildQuery(entityTypeApplication, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return applications, err
	}
	err = dbAPI.QueryIn(context, &applicationDBOs, query, dbQueryParam.Param)
	if err != nil {
		return applications, err
	}
	for _, applicationDBO := range applicationDBOs {
		application := model.Application{}
		err = base.Convert(&applicationDBO, &application)
		if err != nil {
			return applications, err
		}
		applications = append(applications, application)
	}
	err = dbAPI.populateApplicationsEdgesAndSelectors(context, tenantID, applications)
	if err == nil {
		err = dbAPI.populateApplicationOriginSelectors(context, applications)
	}
	if err == nil {
		err = dbAPI.populateApplicationEndpoints(context, applications)
	}
	if renderApp {
		NewApps(applications).RenderForContext(authContext, dbAPI)
	}
	return applications, err
}

// internal api for old public W apis
func (dbAPI *dbObjectModelAPI) getApplicationsW(context context.Context, projectID string, applicationID string, w io.Writer, req *http.Request) error {
	etag := getEtag(req)
	queryParam := model.GetEntitiesQueryParamV1(req)
	applications, err := dbAPI.getApplicationsEtag(context, etag, applicationID, projectID, queryParam, true)
	if err != nil {
		return err
	}
	if len(applicationID) == 0 {
		return base.DispatchPayload(w, applications)
	}
	if len(applications) == 0 {
		return errcode.NewRecordNotFoundError(applicationID)
	}
	return json.NewEncoder(w).Encode(applications[0])
}

// internal api for new (paged) public W apis
func (dbAPI *dbObjectModelAPI) getApplicationsWV2(context context.Context, projectID string, applicationID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getApplicationDBQueryParam(context, projectID, applicationID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.ApplicationListResponsePayload{ApplicationListV2: []model.ApplicationV2{}})
	}
	projectIDs := dbQueryParam.Param.(ApplicationProjects).ProjectIDs
	queryParam := model.GetEntitiesQueryParam(req)
	applications, totalCount, err := dbAPI.getApplicationsByProjectsForQuery(context, projectIDs, applicationID, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeApplication})

	if len(applicationID) == 0 {
		r := model.ApplicationListResponsePayload{
			EntityListResponsePayload: entityListResponsePayload,
			ApplicationListV2:         model.ApplicationsByID(applications).ToV2(),
		}
		return json.NewEncoder(w).Encode(r)
	}
	if len(applications) == 0 {
		return errcode.NewRecordNotFoundError(applicationID)
	}
	return json.NewEncoder(w).Encode(applications[0].ToV2())
}

func (dbAPI *dbObjectModelAPI) getApplicationContainers(context context.Context, applicationID string, edgeID string, callback func(context.Context, interface{}) (string, error)) (model.ApplicationContainers, error) {
	appContainers := model.ApplicationContainers{}
	wsMessagePayload := model.ApplicationContainersBaseObject{
		ApplicationID: applicationID,
		EdgeID:        edgeID,
	}
	wsResp, err := callback(context, wsMessagePayload)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error executing websocket callback: %s"), err.Error())
		return appContainers, err
	}
	respString := ""
	if err := json.Unmarshal([]byte(wsResp), &respString); err != nil {
		glog.Errorf(base.PrefixRequestID(context, "json unmarshal error: %s"), err.Error())
		return appContainers, err
	}
	appContainers.ApplicationContainersBaseObject = model.ApplicationContainersBaseObject{
		ApplicationID: applicationID,
		EdgeID:        edgeID,
	}
	appContainers.ContainerNames = []string{}
	if err := json.Unmarshal([]byte(respString), &appContainers.ContainerNames); err != nil {
		glog.Errorf(base.PrefixRequestID(context, "json unmarshal error: %s"), err.Error())
	}
	return appContainers, err
}

// SelectAllApplications select all applications for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllApplications(context context.Context) ([]model.Application, error) {
	return dbAPI.getApplicationsEtag(context, "", "", "", nil, true)
}

// SelectAllApplicationsW select all applications for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getApplicationsW(context, "", "", w, req)
}

// SelectAllApplicationsWV2 select all applications for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getApplicationsWV2(context, "", "", w, req)
}

// SelectAllApplicationsForProject select all applications for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsForProject(context context.Context, projectID string) ([]model.Application, error) {
	return dbAPI.getApplicationsEtag(context, "", "", projectID, nil, true)
}

// SelectAllApplicationsForProjectW select all applications for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	// return base.DispatchPayload(w, applications)
	return dbAPI.getApplicationsW(context, projectID, "", w, req)
}

// SelectAllApplicationsForProjectWV2 select all applications for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getApplicationsWV2(context, projectID, "", w, req)
}

// GetApplication get an application object in the DB
func (dbAPI *dbObjectModelAPI) GetApplication(context context.Context, applicationID string) (model.Application, error) {
	if len(applicationID) == 0 {
		return model.Application{}, errcode.NewBadRequestError("applicationID")
	}
	applications, err := dbAPI.getApplicationsEtag(context, "", applicationID, "", nil, true)

	if err != nil {
		return model.Application{}, err
	}
	if len(applications) == 0 {
		return model.Application{}, errcode.NewRecordNotFoundError(applicationID)
	}
	return applications[0], nil
}

// GetApplicationW get a application object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetApplicationW(context context.Context, applicationID string, w io.Writer, req *http.Request) error {
	if len(applicationID) == 0 {
		return errcode.NewBadRequestError("applicationID")
	}

	// return base.DispatchPayload(w, applications[0])
	return dbAPI.getApplicationsW(context, "", applicationID, w, req)
}

// GetApplicationWV2 get a application object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetApplicationWV2(context context.Context, applicationID string, w io.Writer, req *http.Request) error {
	if len(applicationID) == 0 {
		return errcode.NewBadRequestError("applicationID")
	}
	return dbAPI.getApplicationsWV2(context, "", applicationID, w, req)
}

// GetApplicationContainersW get the containers of an application object and writes output into writer
func (dbAPI *dbObjectModelAPI) GetApplicationContainersW(context context.Context, applicationID string, edgeID string, w io.Writer, callback func(context.Context, interface{}) (string, error)) error {
	if applicationID == "" {
		return errcode.NewBadRequestError("applicationID")
	}
	if edgeID == "" {
		return errcode.NewBadRequestError("edgeID")
	}
	if _, err := dbAPI.GetApplication(context, applicationID); err != nil {
		return errcode.NewBadRequestError("applicationID")
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

	resp, err := dbAPI.getApplicationContainers(context, applicationID, edgeID, callback)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)
}

func (dbAPI *dbObjectModelAPI) createOrDeleteAppEndpoints(context context.Context, tx *base.WrappedTx, app *model.Application, create bool) (loggedErr error) {
	defer func() {
		if loggedErr != nil {
			glog.Error(base.PrefixRequestID(context, loggedErr.Error()))
		}
	}()
	op := "create"
	if !create {
		op = "delete"
	}
	glog.V(5).Infof(base.PrefixRequestID(context, "op=%s type for app(id=%s) %s"), op, app.ID, app.Name)

	// Delete all endpoints associated with this app
	if !create {
		glog.V(5).Infof(base.PrefixRequestID(context, "deleting all application endpoints for app(id=%s) %s"), app.ID, app.Name)
		loggedErr = dbAPI.DeleteAllApplicationEndpoints(context, tx, app.ID)
	}

	datasources, err := dbAPI.SelectDataSourcesByEndpoints(context, app.DataIfcEndpoints)
	if err != nil {
		loggedErr = fmt.Errorf("failed to select data sources for given endpoints. %s", err.Error())
		return
	}

	dataSourcesByID := make(map[string]model.DataSource)
	for _, ds := range datasources {
		dataSourcesByID[ds.ID] = ds
	}

	for _, e := range app.DataIfcEndpoints {
		var ds model.DataSource
		var ok bool
		//  Validations
		if ds, ok = dataSourcesByID[e.ID]; !ok {
			loggedErr = errcode.NewPreConditionFailedError(fmt.Sprintf("failed to find data source %s for application %s", e.ID, app.Name))
			return
		}
		if ds.IfcInfo == nil {
			loggedErr = errcode.NewBadRequestExError("DataIfcEndpoints", fmt.Sprintf("cannot %s application endpoint %+v for data sources with no data interface", op, e))
			return
		}
		if ds.IfcInfo.Kind != model.DataIfcEndpointKindIn && ds.IfcInfo.Kind != model.DataIfcEndpointKindOut {
			loggedErr = errcode.NewInternalError(fmt.Sprintf("unexpected data ifc kind %v", ds.IfcInfo.Kind))
			return
		}

		// Claim the topic
		if ds.IfcInfo.Kind == model.DataIfcEndpointKindOut {
			glog.Infof(base.PrefixRequestID(context, "updating data ifc topic claims for application %s with endpoint %+v"),
				app.ID, e,
			)
			if create {
				loggedErr = dbAPI.claimDataIfcTopic(context, tx, &e, entityTypeApplication, app.ID, app.TenantID)
			} else {
				loggedErr = dbAPI.unclaimDataIfcTopic(context, tx, &e, entityTypeApplication, app.ID)
			}
			glog.V(5).Infof(base.PrefixRequestID(context, "successfully claimed/unclaimed topic %s for app(id=%s) %s, endpoint: %+v"), app.ID, app.Name, e)
		}
		if loggedErr != nil {
			return
		}

		if create {
			// Create an entry in the application endpoint table
			glog.V(3).Infof(base.PrefixRequestID(context, "creating application endpoint for app(id=%s) %s: %+v"), app.ID, app.Name, e)
			loggedErr = dbAPI.CreateApplicationEndpoint(context, tx, app.ID, app.TenantID, e)
			if loggedErr != nil {
				return
			}
		}
	}
	return
}

// CreateApplication creates an application object in the DB
func (dbAPI *dbObjectModelAPI) CreateApplication(context context.Context, i interface{} /* *model.Application */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Application)
	if !ok {
		return resp, errcode.NewInternalError("CreateApplication: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateApplication doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateApplication doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	// set default project for backward compatibility
	if doc.ProjectID == "" {
		doc.ProjectID = GetDefaultProjectID(tenantID)
		// check if default project exist
		_, err = dbAPI.GetProject(context, doc.ProjectID)
		if errcode.IsRecordNotFound(err) {
			return resp, errcode.NewBadRequestExError("projectID", fmt.Sprintf("Default project not found, ID: %s", doc.ProjectID))
		} else if err != nil {
			return resp, err
		}
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityApplication,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	project, err := dbAPI.GetProject(context, doc.ProjectID)
	if err != nil {
		return resp, err
	}
	err = model.ValidateApplication(&doc, model.GetK8sSchemaVersion(project.IsPrivileged()))
	if err != nil {
		return resp, err
	}
	if len(doc.Name) == 0 {
		return resp, errcode.NewBadRequestError("name")
	}
	err = dbAPI.validateApplication(context, &doc)
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	applicationDBO := ApplicationDBO{}

	err = base.Convert(&doc, &applicationDBO)
	if err != nil {
		return resp, err
	}
	// Populate the edgeIDs field for category based selection
	err = dbAPI.resolveApplicationEdges(context, tenantID, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error resolving service domain IDs for application %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	doc.ExcludeEdgeIDs = base.Unique(doc.ExcludeEdgeIDs)
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err := tx.NamedExec(context, queryMap["CreateApplication"], &applicationDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating application %+v. Error: %s"), applicationDBO, err.Error())
			return errcode.TranslateDatabaseError(applicationDBO.ID, err)
		}
		err = dbAPI.createApplicationEdges(context, tx, &doc)
		if err != nil {
			return err
		}
		err = dbAPI.createApplicationEdgeSelectors(context, tx, &doc)
		if err != nil {
			return err
		}
		if doc.OriginSelectors != nil {
			err = dbAPI.createOriginSelectors(context, tx, *doc.OriginSelectors, entityTypeApplication, doc.ID)
		}
		err = dbAPI.createOrDeleteAppEndpoints(context, tx, &doc, true)
		if err != nil {
			return err
		}
		return err
	})
	if err != nil {
		return resp, err
	}

	apps := []model.Application{doc}
	err = dbAPI.populateApplicationsEdgesAndSelectors(context, tenantID, apps)
	if err != nil {
		return resp, err
	}

	err = dbAPI.populateApplicationOriginSelectors(context, apps)
	if err != nil {
		return resp, err
	}
	err = dbAPI.populateApplicationEndpoints(context, apps)
	if err != nil {
		return resp, err
	}
	doc = apps[0]
	if callback != nil {
		// Application object contains template in YAML which should be
		// rendered before publishing to edge.
		go callback(context, NewApp(&doc))
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addApplicationAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateApplicationV2 creates an application object in the DB
func (dbAPI *dbObjectModelAPI) CreateApplicationV2(context context.Context, i interface{} /* *model.ApplicationV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.ApplicationV2)
	if !ok {
		return model.CreateDocumentResponse{}, errcode.NewInternalError("CreateApplicationV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.CreateApplication(context, &doc, callback)
}

// CreateApplicationW creates an application object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateApplicationW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateApplication, &model.Application{}, w, r, callback)
}

// CreateApplicationWV2 creates an application object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateApplicationWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateApplicationV2), &model.ApplicationV2{}, w, r, callback)
}

// UpdateApplication update an application object in the DB
func (dbAPI *dbObjectModelAPI) UpdateApplication(context context.Context, i interface{} /* *model.Application */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Application)
	if !ok {
		return resp, errcode.NewInternalError("UpdateApplication: type error")
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

	// fetch application to get old project id
	app, err := dbAPI.GetApplication(context, doc.ID)
	if err != nil {
		return resp, errcode.NewBadRequestError("applicationID")
	}
	// set default project for backward compatibility
	if doc.ProjectID == "" {
		doc.ProjectID = GetDefaultProjectID(tenantID)
		// check if default project exist
		_, err = dbAPI.GetProject(context, doc.ProjectID)
		if errcode.IsRecordNotFound(err) {
			return resp, errcode.NewBadRequestExError("projectID", fmt.Sprintf("Default project not found, ID: %s", doc.ProjectID))
		} else if err != nil {
			return resp, err
		}
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityApplication,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:    doc.ProjectID,
			OldProjectID: app.ProjectID,
			ProjNameFn:   GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}

	if app.IsHelmApp() != doc.IsHelmApp() {
		err = errcode.NewBadRequestExError("PackagingType", fmt.Sprintf("PackagingType must not change"))
		return resp, err
	}

	project, err := dbAPI.GetProject(context, doc.ProjectID)
	if err != nil {
		return resp, err
	}
	err = model.ValidateApplication(&doc, model.GetK8sSchemaVersion(project.IsPrivileged()))
	if err != nil {
		return resp, err
	}
	if len(doc.Name) == 0 {
		return resp, errcode.NewBadRequestError("name")
	}

	err = dbAPI.validateApplication(context, &doc)
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	// Populate the edgeIDs field for category based selection
	err = dbAPI.resolveApplicationEdges(context, tenantID, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error resolving service domain IDs for application %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	doc.ExcludeEdgeIDs = base.Unique(doc.ExcludeEdgeIDs)
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		applicationDBO := ApplicationDBO{}
		err := base.Convert(&doc, &applicationDBO)
		if err != nil {
			return err
		}
		_, err = tx.NamedExec(context, queryMap["UpdateApplication"], &applicationDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in updating application for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		_, err = base.DeleteTxn(context, tx, "application_edge_model", map[string]interface{}{"application_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting application edges for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createApplicationEdges(context, tx, &doc)
		if err != nil {
			return err
		}
		_, err = base.DeleteTxn(context, tx, "application_edge_selector_model", map[string]interface{}{"application_id": doc.ID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in deleting application edge selectors for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		err = dbAPI.createApplicationEdgeSelectors(context, tx, &doc)
		if err != nil {
			return err
		}
		if *config.Cfg.EnableAppOriginSelectors {
			_, err = base.DeleteTxn(context, tx, "application_origin_model", map[string]interface{}{"application_id": doc.ID})
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error in deleting application edge selectors for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
				return errcode.TranslateDatabaseError(doc.ID, err)
			}
			if doc.OriginSelectors != nil {
				err = dbAPI.createOriginSelectors(context, tx, *doc.OriginSelectors, entityTypeApplication, doc.ID)
			}
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "failed to create origin selectors for application. %s"), doc.ID, err.Error())
				return err
			}
		}

		// delete old endpoints
		err = dbAPI.createOrDeleteAppEndpoints(context, tx, &app, false)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "failed to delete applicatiton endpoints e topic claim for application %s. Error: %s"), doc.ID, err.Error())
			return err
		}

		// add new endpoints
		err = dbAPI.createOrDeleteAppEndpoints(context, tx, &doc, true)
		return err
	})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "failed to update applicatin %s. %s"), doc.ID, err.Error())
		return resp, err
	}
	apps := []model.Application{doc}
	err = dbAPI.populateApplicationsEdgesAndSelectors(context, tenantID, apps)
	if err != nil {
		return resp, err
	}
	err = dbAPI.populateApplicationOriginSelectors(context, apps)
	if err != nil {
		return resp, err
	}
	err = dbAPI.populateApplicationEndpoints(context, apps)
	if err != nil {
		return resp, err
	}
	doc = apps[0]
	if callback != nil {
		// Application object contains template in YAML which should be
		// rendered before publishing to edge.
		go callback(context, NewApp(&doc))
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addApplicationAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateApplicationV2 update an application object in the DB
func (dbAPI *dbObjectModelAPI) UpdateApplicationV2(context context.Context, i interface{} /* *model.ApplicationV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.ApplicationV2)
	if !ok {
		return model.UpdateDocumentResponse{}, errcode.NewInternalError("UpdateApplicationV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.UpdateApplication(context, &doc, callback)
}

// UpdateApplicationW update an application object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateApplicationW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateApplication, &model.Application{}, w, r, callback)
}

// UpdateApplicationWV2 update an application object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateApplicationWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateApplicationV2), &model.ApplicationV2{}, w, r, callback)
}

// DeleteApplication delete an application object in the DB
func (dbAPI *dbObjectModelAPI) DeleteApplication(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	// fetch application to get project id
	doc, err := dbAPI.GetApplication(context, id)

	// Handle idempotence
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityApplication,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err := dbAPI.createOrDeleteAppEndpoints(context, tx, &doc, false)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "failed to remove the application endpoints claim application %s. Error: %s"), doc.ID, err.Error())
			return err
		}
		res, err := base.DeleteTxn(context, tx, "application_model", map[string]interface{}{"id": id, "tenant_id": authContext.TenantID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "failed to remove application %s. Error: %s"), doc.ID, err.Error())
			return err
		}

		if base.IsDeleteSuccessful(res) {
			resp.ID = id
			if callback != nil {
				go callback(context, doc)
			}
			GetAuditlogHandler().addApplicationAuditLog(dbAPI, context, doc, DELETE)
		}
		return nil
	})
	return resp, err
}

// DeleteApplicationW delete an application object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteApplicationW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteApplication, id, w, callback)
}

// DeleteApplicationWV2 delete an application object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteApplicationWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteApplication), id, w, callback)
}

type projectEdgeIDsParam struct {
	ProjectID string   `db:"project_id"`
	EdgeIDs   []string `db:"edge_ids"`
}

var reDollarVar = regexp.MustCompile(`\$[0-9]+`)

func (dbAPI *dbObjectModelAPI) DeleteProjectApplicationsEdges(context context.Context, projectID string, edgeIDs []string) error {
	arg := projectEdgeIDsParam{
		ProjectID: projectID,
		EdgeIDs:   edgeIDs,
	}
	db := dbAPI.GetDB()
	var q string
	var needIn bool
	if len(edgeIDs) == 0 {
		q = queryMap["DeleteProjectApplicationsEdges"]
	} else {
		q = queryMap["DeleteProjectApplicationsEdgesList"]
		needIn = true
	}
	q, args, err := db.BindNamed(q, arg)
	if err != nil {
		return err
	}
	if needIn {
		// convert $d back to ? needed by sqlx.In
		q = reDollarVar.ReplaceAllString(q, "?")
		q, args, err = sqlx.In(q, args...)
		if err != nil {
			return err
		}
		q = db.Rebind(q)
	}
	_, err = dbAPI.Exec(context, q, args...)
	return err
}

func (dbAPI *dbObjectModelAPI) DeleteProjectApplicationsEdgeSelectors(context context.Context, projectID string) error {
	arg := projectEdgeIDsParam{
		ProjectID: projectID,
	}
	db := dbAPI.GetDB()
	q := queryMap["DeleteProjectApplicationsEdgeSelectors"]
	q, args, err := db.BindNamed(q, arg)
	if err != nil {
		return err
	}
	_, err = dbAPI.Exec(context, q, args...)
	return err
}

func (dbAPI *dbObjectModelAPI) getApplicationsByIDs(ctx context.Context, applicationIDs []string) ([]model.Application, error) {
	applications := []model.Application{}
	if len(applicationIDs) == 0 {
		return applications, nil
	}

	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID

	applicationDBOs := []ApplicationDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &applicationDBOs, "application_model", applicationIDs); err != nil {
		return nil, err
	}

	for _, applicationDBO := range applicationDBOs {
		application := model.Application{}
		err := base.Convert(&applicationDBO, &application)
		if err != nil {
			return []model.Application{}, err
		}
		applications = append(applications, application)
	}
	err = dbAPI.populateApplicationsEdgesAndSelectors(ctx, tenantID, applications)
	if err == nil {
		err = dbAPI.populateApplicationOriginSelectors(ctx, applications)
	}
	if err == nil {
		dbAPI.populateApplicationEndpoints(ctx, applications)
	}
	NewApps(applications).RenderForContext(authContext, dbAPI)
	return applications, err
}

func (dbAPI *dbObjectModelAPI) SelectAllApplicationsForDataIfcEndpoint(context context.Context, dataIfcID string) ([]model.Application, error) {
	appIds, err := dbAPI.FetchApplicationIDsByDataIfcID(context, dataIfcID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "failed to fetch data ifc endpoints for %s. Error: %s"), dataIfcID, err.Error())
		return nil, err
	}
	glog.V(5).Infof("recieved app Ids %+v associated with data source/ifc %s", appIds, dataIfcID)
	return dbAPI.getApplicationsByIDs(context, appIds)
}

func (dbAPI *dbObjectModelAPI) RenderApplication(context context.Context, appID, edgeID string, param model.RenderApplicationPayload) (model.RenderApplicationResponse, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return model.RenderApplicationResponse{}, err
	}
	tenantID := authContext.TenantID
	if len(appID) == 0 {
		return model.RenderApplicationResponse{}, errcode.NewBadRequestError("appID")
	}
	if len(edgeID) == 0 {
		return model.RenderApplicationResponse{}, errcode.NewBadRequestError("edgeID")
	}
	apps, err := dbAPI.getApplicationsEtag(context, "", appID, "", nil, false)
	if err != nil {
		return model.RenderApplicationResponse{}, err
	}
	if len(apps) == 0 {
		return model.RenderApplicationResponse{}, errcode.NewRecordNotFoundError(appID)
	}
	services := map[string]apptemplate.EdgeService{}
	for name, svc := range param.EdgeServices {
		services[name] = apptemplate.EdgeService{
			Endpoint: svc.Endpoint,
		}
	}
	edgeParams := apptemplate.EdgeParameters{
		Services: services,
	}
	yamlObj, edgeServices, err := NewApp(&apps[0]).RenderForEdgeWithParams(dbAPI,
		tenantID, edgeID, edgeParams)
	if err != nil {
		return model.RenderApplicationResponse{}, err
	}
	return model.RenderApplicationResponse{
		Payload: &model.RenderApplicationResponsePayload{
			AppYaml:      yamlObj,
			EdgeServices: edgeServices,
		},
	}, nil
}

func (dbAPI *dbObjectModelAPI) RenderApplicationW(context context.Context, appID, edgeID string, w io.Writer, r *http.Request) error {
	reader := io.Reader(r.Body)
	doc := model.RenderApplicationPayload{}
	err := base.Decode(&reader, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into render application payload. Error: %s"), err.Error())
		return err
	}
	resp, err := dbAPI.RenderApplication(context, appID, edgeID, doc)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp.Payload)
}

func (dbAPI *dbObjectModelAPI) CreateHelmAppW(context context.Context, unused string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error {
	glog.Infof("Create Helm App")
	resp := model.CreateDocumentResponseV2{}

	// we support both regular and multipart POST
	var reader io.Reader
	var mediaType string
	params := make(map[string]string)
	reader = req.Body
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		mt, ps, err := mime.ParseMediaType(contentType)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error parsing content type in create helm app. Error: %s"), err.Error())
			return errcode.NewBadRequestError("Content-Type")
		}
		mediaType = mt
		params = ps
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(req.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return errcode.NewBadRequestError("Content Not Found")
			}
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error parsing content in create ML model version, Error: %s"), err.Error())
				return errcode.NewBadRequestError("Content")
			}
			filename := p.FileName()
			if filename == "" {
				glog.Infof(base.PrefixRequestID(context, "create ML model version, skip part w/o filename, name: %s, part: %+v\n"), p.FormName(), *p)
				continue
			}
			glog.Infof(base.PrefixRequestID(context, "create ML model version using multipart content with file name: %s\n"), filename)
			reader = p
			break
		}
	}
	resp.ID = base.GetUUID()
	go func() {
		filename := fmt.Sprintf("%v/%v.tgz", helmChartDir, resp.ID)
		file, err := os.Create(filename)
		if err != nil {
			glog.Infof("Failed to create file %v, err %v", filename, err)
			return
		}
		defer file.Close()
		n, err := io.Copy(file, reader)
		if err != nil {
			glog.Infof("Failed to copy chart %v", resp.ID)
			errfile := fmt.Sprintf("%v/%v.tgz.error", helmChartDir, resp.ID)
			ef, err := os.Create(errfile)
			if err != nil {
				glog.Infof("Failed to create err file %v, err %v", errfile, err)
			}
			defer ef.Close()
			return
		}
		glog.Infof("Finished copying helm chart %v, %v bytes", resp.ID, n)
		donefile := fmt.Sprintf("%v/%v.tgz.done", helmChartDir, resp.ID)
		df, err := os.Create(donefile)
		if err != nil {
			glog.Infof("Failed to create done %v file, err %v", donefile, err)
		}
		defer df.Close()
		return
	}()
	glog.Infof("Return Helm App %v", resp.ID)
	return json.NewEncoder(w).Encode(resp)
}

func untarChart(chartID string) (string, error) {
	chartFile := fmt.Sprintf("%v/%v.tgz", helmChartDir, chartID)
	if _, err := os.Stat(chartFile); err != nil {
		glog.Infof("Chart %v is not completely uploaded, err %v", chartFile, err)
		return "", err
	}

	chartDir := fmt.Sprintf("%v/charts/%v", helmChartDir, chartID)
	if err := os.MkdirAll(chartDir, 0700); err != nil {
		glog.Infof("Failed to create dir %v, err %v", chartDir, err)
		return "", err
	}

	untarCmd := exec.Command("tar", "xfz", chartFile, "--directory", chartDir)
	if err := untarCmd.Run(); err != nil {
		glog.Infof("Failed to run command %v", untarCmd)
		return "", err
	}

	findCmd := exec.Command("find", chartDir, "-name", "Chart.yaml")
	yamlFilePath, err := findCmd.Output()
	if err != nil {
		glog.Infof("Failed to find Chart.yaml in chart %v, err %v", chartFile, err)
		return "", err
	}

	yamlPathString := strings.Split(strings.TrimSpace(string(yamlFilePath)), "\n")
	shortest := ""
	for _, s := range yamlPathString {
		if shortest == "" {
			shortest = s
		} else {
			if len(shortest) > len(s) {
				shortest = s
			}
		}
	}

	yamlDir := filepath.Dir(string(shortest))
	glog.Infof("yamlDir for chart %v is %v", chartID, yamlDir)
	return yamlDir, nil
}

func generateYamlFromHelmChart(chartID string) {
	yamlFile := fmt.Sprintf("%v/%v.yaml", helmChartDir, chartID)
	yamlDone := fmt.Sprintf("%v/%v.yaml.done", helmChartDir, chartID)
	yamlErr := fmt.Sprintf("%v/%v.yaml.error", helmChartDir, chartID)

	err := func() error {
		// Check if chart is completely available
		chartFile := fmt.Sprintf("%v/%v.tgz", helmChartDir, chartID)
		chartFileDone := fmt.Sprintf("%v/%v.tgz.done", helmChartDir, chartID)

		if _, err := os.Stat(chartFileDone); err != nil {
			glog.Infof("Chart %v is not completely uploaded, err %v", chartFile, err)
			return errHelmWaiting
		}

		chartYamlDir, err := untarChart(chartID)
		if err != nil {
			return err
		}

		valuesFile := fmt.Sprintf("%v/values-%v.yaml", helmChartDir, chartID)
		valuesFileDone := fmt.Sprintf("%v/values-%v.yaml.done", helmChartDir, chartID)

		var helmYamlCmd *exec.Cmd
		// Check if values file exists for this chart
		_, err = os.Stat(valuesFile)
		if err == nil {
			// Values file exists, lets make sure it is complete
			if _, err := os.Stat(valuesFileDone); err != nil {
				glog.Infof("Waiting for values file %v, err %v", valuesFile, err)
				return errHelmWaiting
			}
			// Create helm yaml with values
			helmYamlCmd = exec.Command("helm", "template", "-f", valuesFile, chartYamlDir)
		} else {
			// Create helm yaml with default values
			helmYamlCmd = exec.Command("helm", "template", chartYamlDir)
		}

		chartYaml, err := helmYamlCmd.Output()
		glog.Infof("Chart yaml is\n%v\n", string(chartYaml))
		f, err := os.Create(yamlFile)
		if err != nil {
			glog.Infof("Failed to create yaml file %v, err %v", yamlFile, err)
			return err
		}
		defer f.Close()
		_, err = f.WriteString(string(chartYaml))
		if err != nil {
			glog.Infof("Failed to write file %v, err %v", yamlFile, err)
			return err
		}

		fd, err := os.Create(yamlDone)
		if err != nil {
			glog.Infof("Failed to create %v, err %v", yamlDone, err)
			return err
		}
		defer fd.Close()
		return nil
	}()

	if err != nil {
		if err == errHelmWaiting {
			return
		}

		ef, err := os.Create(yamlErr)
		if err != nil {
			glog.Infof("Failed to create error file %v, err %v", yamlErr, err)
			return
		}
		defer ef.Close()
		return
	}
}

func (dbAPI *dbObjectModelAPI) GetHelmAppYaml(context context.Context, chartID string, w io.Writer, req *http.Request) error {
	resp := model.HelmAppYamlResponse{ID: chartID, Status: "Done", Yaml: ""}
	// Check if yaml for the chart already exists
	yamlDone := fmt.Sprintf("%v/%v.yaml.done", helmChartDir, chartID)
	if _, err := os.Stat(yamlDone); err == nil {
		// yaml already exists
		yamlFile := fmt.Sprintf("%v/%v.yaml", helmChartDir, chartID)
		yamlData, err := ioutil.ReadFile(yamlFile)
		if err != nil {
			glog.Infof("Failed to read yaml file %v, err %v", yamlFile, err)
			resp.Status = "Error"
			return json.NewEncoder(w).Encode(resp)
		}
		resp.Yaml = string(yamlData)
		return json.NewEncoder(w).Encode(resp)
	}

	// Check if the chart had some errors already
	yamlErr := fmt.Sprintf("%v/%v.yaml.error", helmChartDir, chartID)
	if _, err := os.Stat(yamlErr); err == nil {
		// yaml err, means this chart is troublesome and cannot be converted
		glog.Infof("Chart %v has some errors", chartID)
		resp.Status = "Error"
		return json.NewEncoder(w).Encode(resp)
	}

	// Else working
	go generateYamlFromHelmChart(chartID)

	resp.Status = "Working"
	return json.NewEncoder(w).Encode(resp)
}

func (dbAPI *dbObjectModelAPI) CreateHelmValuesW(context context.Context, chartID string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error {
	glog.Infof("Add Helm Values")
	resp := model.CreateDocumentResponseV2{}

	chartFile := fmt.Sprintf("%v/%v.tgz", helmChartDir, chartID)
	if _, err := os.Stat(chartFile); err != nil {
		glog.Infof("chart %v does not exist, err %v", chartFile, err)
		return errcode.NewBadRequestError("Chart does not exist, cannot add values")
	}

	// we support both regular and multipart POST
	var reader io.Reader
	var mediaType string
	params := make(map[string]string)
	reader = req.Body
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		mt, ps, err := mime.ParseMediaType(contentType)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error parsing content type in create helm app. Error: %s"), err.Error())
			return errcode.NewBadRequestError("Content-Type")
		}
		mediaType = mt
		params = ps
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(req.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err == io.EOF {
				return errcode.NewBadRequestError("Content Not Found")
			}
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error parsing content in create ML model version, Error: %s"), err.Error())
				return errcode.NewBadRequestError("Content")
			}
			filename := p.FileName()
			if filename == "" {
				glog.Infof(base.PrefixRequestID(context, "create ML model version, skip part w/o filename, name: %s, part: %+v\n"), p.FormName(), *p)
				continue
			}
			glog.Infof(base.PrefixRequestID(context, "create ML model version using multipart content with file name: %s\n"), filename)
			reader = p
			break
		}
	}
	resp.ID = chartID
	go func() {
		filename := fmt.Sprintf("%v/values-%v.yaml", helmChartDir, resp.ID)
		file, err := os.Create(filename)
		if err != nil {
			glog.Infof("Failed to create file %v, err %v", filename, err)
			return
		}
		defer file.Close()

		n, err := io.Copy(file, reader)
		if err != nil {
			glog.Infof("Failed to copy values file %v", resp.ID)
			errfile := fmt.Sprintf("%v/values-%v.yaml.error", helmChartDir, resp.ID)
			ef, err := os.Create(errfile)
			if err != nil {
				glog.Infof("Failed to create values err file %v, err %v", errfile, err)
			}
			defer ef.Close()
			return
		}
		glog.Infof("Finished adding values file for chart %v, %v bytes", resp.ID, n)
		donefile := fmt.Sprintf("%v/values-%v.yaml.done", helmChartDir, resp.ID)
		df, err := os.Create(donefile)
		if err != nil {
			glog.Infof("Failed to create values done %v file, err %v", donefile, err)
		}
		defer df.Close()

		generateYamlFromHelmChart(resp.ID)
		return
	}()
	glog.Infof("Return Helm values create %v", resp.ID)
	return json.NewEncoder(w).Encode(resp)
}

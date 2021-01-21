package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"

	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang/glog"
)

const entityTypeProjectService = "projectService"

func init() {
	queryMap["SelectProjectServicesByProjectsTemplate"] = `SELECT * FROM project_service_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (project_id is NULL OR project_id IN (:project_ids)) %s`
	queryMap["CreateProjectService"] = `INSERT INTO project_service_model (id, version, tenant_id, name, service_manifest, project_id, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :service_manifest, :project_id, :created_at, :updated_at)`
	queryMap["UpdateProjectService"] = `UPDATE project_service_model SET version = :version, tenant_id = :tenant_id, name = :name, service_manifest = :service_manifest, project_id = :project_id, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectProjectServiceIDsTemplate"] = `SELECT id from project_service_model where tenant_id = :tenant_id AND project_id is NULL %s`
	queryMap["SelectProjectServiceIDsInProjectsTemplate"] = `SELECT id from project_service_model where tenant_id = :tenant_id AND (project_id is NULL OR project_id IN (:project_ids)) %s`

	orderByHelper.Setup(entityTypeProjectService, []string{"id", "version", "created_at", "updated_at", "name", "service_manifest", "project_id"})
}

// ProjectServiceDBO is DB object model for project service
type ProjectServiceDBO struct {
	model.BaseModelDBO
	Name      string  `json:"name" db:"name"`
	Manifest  string  `json:"serviceManifest" db:"service_manifest"`
	ProjectID *string `json:"projectId" db:"project_id"`
}

func (doc ProjectServiceDBO) GetProjectID() string {
	if doc.ProjectID != nil {
		return *doc.ProjectID
	}
	return ""
}

type ProjectServiceProjects struct {
	ProjectServiceDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for project service
func getProjectServiceDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := ProjectServiceDBO{BaseModelDBO: tenantModel}
	var projectIDs []string
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		projectIDs = []string{projectID}
	} else {
		projectIDs = auth.GetProjectIDs(authContext)
	}
	return base.InQueryParam{
		Param: ProjectServiceProjects{
			ProjectServiceDBO: param,
			ProjectIDs:        projectIDs,
		},
		Key:     "SelectProjectServicesByProjectsTemplate",
		InQuery: true,
	}, nil
}

func (dbAPI *dbObjectModelAPI) getProjectServicesForPage(ctx context.Context, dbQueryParam base.InQueryParam, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ProjectService, base.PageToken, error) {
	projectServices := []model.ProjectService{}
	if dbQueryParam.Key == "" {
		return projectServices, base.NilPageToken, nil
	}
	var queryFn func(context.Context, base.PageToken, int, func(interface{}) error, string, interface{}) (base.PageToken, error)
	if dbQueryParam.InQuery {
		queryFn = dbAPI.NotPagedQueryIn
	} else {
		queryFn = dbAPI.NotPagedQuery
	}
	query, err := buildQuery(entityTypeProjectService, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return projectServices, base.NilPageToken, err
	}
	nextToken, err := queryFn(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		projectService := model.ProjectService{}
		err := base.Convert(dbObjPtr, &projectService)
		if err != nil {
			return err
		}
		projectServices = append(projectServices, projectService)
		return nil
	}, query, dbQueryParam.Param)
	return projectServices, nextToken, err
}

func (dbAPI *dbObjectModelAPI) getProjectServices(context context.Context, projectID string, projectServiceID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ProjectService, error) {
	dbQueryParam, err := getProjectServiceDBQueryParam(context, projectID, projectServiceID)
	if err != nil {
		return []model.ProjectService{}, err
	}
	if dbQueryParam.Key == "" {
		return []model.ProjectService{}, nil
	}
	projectServices, _, err := dbAPI.getProjectServicesForPage(context, dbQueryParam, entitiesQueryParam)
	return projectServices, err
}

func (dbAPI *dbObjectModelAPI) getProjectServicesW(ctx context.Context, projectID string, projectServiceID string, w io.Writer, req *http.Request) error {
	projectServiceDBOs := []ProjectServiceDBO{}
	dbQueryParam, err := getProjectServiceDBQueryParam(ctx, projectID, projectServiceID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		if len(projectServiceID) == 0 {
			return json.NewEncoder(w).Encode([]model.ProjectService{})
		}
		return errcode.NewRecordNotFoundError(projectServiceID)
	}
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	query, err := buildQuery(entityTypeProjectService, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return err
	}
	var queryFn func(context.Context, interface{}, string, interface{}) error
	if dbQueryParam.InQuery {
		queryFn = dbAPI.QueryIn
	} else {
		queryFn = dbAPI.Query
	}
	err = queryFn(ctx, &projectServiceDBOs, query, dbQueryParam.Param)
	if err != nil {
		return err
	}
	if len(projectServiceID) == 0 {
		return base.DispatchPayload(w, projectServiceDBOs)
	}
	if len(projectServiceDBOs) == 0 {
		return errcode.NewRecordNotFoundError(projectServiceID)
	}
	return json.NewEncoder(w).Encode(projectServiceDBOs[0])
}

// SelectAllProjectServices select all project services for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllProjectServices(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ProjectService, error) {
	return dbAPI.getProjectServices(context, "", "", entitiesQueryParam)
}

// SelectAllProjectServicesW select all project services for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllProjectServicesW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getProjectServicesW(context, "", "", w, req)
}

// GetProjectService get a project service object in the DB
func (dbAPI *dbObjectModelAPI) GetProjectService(context context.Context, id string) (model.ProjectService, error) {
	if len(id) == 0 {
		return model.ProjectService{}, errcode.NewBadRequestError("projectServiceID")
	}
	projectServices, err := dbAPI.getProjectServices(context, "", id, nil)
	if err != nil {
		return model.ProjectService{}, err
	}
	if len(projectServices) == 0 {
		return model.ProjectService{}, errcode.NewRecordNotFoundError(id)
	}
	return projectServices[0], nil
}

// GetProjectServiceW get a project service object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetProjectServiceW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("projectServiceID")
	}
	return dbAPI.getProjectServicesW(context, "", id, w, req)
}

// CreateProjectService creates a project service object in the DB
func (dbAPI *dbObjectModelAPI) CreateProjectService(context context.Context, i interface{} /* *model.ProjectService */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ProjectService)
	if !ok {
		return resp, errcode.NewInternalError("CreateProjectService: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateProjectService doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateProjectService doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityProjectService,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	projectServiceDBO := ProjectServiceDBO{}
	err = base.Convert(&doc, &projectServiceDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateProjectService"], &projectServiceDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating project service for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// CreateProjectServiceW creates a project service object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateProjectServiceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateProjectService, &model.ProjectService{}, w, r, callback)
}

// UpdateProjectService update a project service object in the DB
func (dbAPI *dbObjectModelAPI) UpdateProjectService(context context.Context, i interface{} /* *model.ProjectService */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	projectServiceW, ok := i.(*model.ProjectService)
	if !ok {
		return resp, errcode.NewInternalError("UpdateProjectService: type error")
	}
	p := projectServiceW
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	if p.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	sr, err := dbAPI.GetProjectService(context, doc.ID)
	if err != nil {
		return resp, errcode.NewBadRequestError("projectServiceID")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityProjectService,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:    doc.ProjectID,
			OldProjectID: sr.ProjectID,
			ProjNameFn:   GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	projectServiceDBO := ProjectServiceDBO{}
	err = base.Convert(&doc, &projectServiceDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateProjectService"], &projectServiceDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating project service for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateProjectServiceW update a project service object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateProjectServiceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateProjectService, &model.ProjectService{}, w, r, callback)
}

// DeleteProjectService delete a project service object in the DB
func (dbAPI *dbObjectModelAPI) DeleteProjectService(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	sr, err := dbAPI.GetProjectService(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityProjectService,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  sr.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	return DeleteEntity(context, dbAPI, "project_service_model", "id", id, sr, callback)
}

// DeleteProjectServiceW delete a project service object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteProjectServiceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteProjectService, id, w, callback)
}

func (dbAPI *dbObjectModelAPI) getProjectServicesByIDs(ctx context.Context, projectServiceIDs []string) ([]model.ProjectService, error) {
	projectServices := []model.ProjectService{}
	if len(projectServiceIDs) == 0 {
		return projectServices, nil
	}

	projectServiceDBOs := []ProjectServiceDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &projectServiceDBOs, "project_service_model", projectServiceIDs); err != nil {
		return nil, err
	}

	for _, projectServiceDBO := range projectServiceDBOs {
		projectService := model.ProjectService{}
		err := base.Convert(&projectServiceDBO, &projectService)
		if err != nil {
			return []model.ProjectService{}, err
		}
		projectServices = append(projectServices, projectService)
	}
	return projectServices, nil
}

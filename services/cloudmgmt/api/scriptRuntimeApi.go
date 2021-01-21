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
	"strings"

	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
)

const entityTypeScriptRuntime = "scriptruntime"

func init() {
	queryMap["SelectScriptRuntimesTemplate1"] = `SELECT * FROM script_runtime_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND project_id is NULL %s`
	queryMap["SelectScriptRuntimesByProjectsTemplate1"] = `SELECT * FROM script_runtime_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (project_id is NULL OR project_id IN (:project_ids)) %s`
	queryMap["SelectScriptRuntimesTemplate"] = `SELECT *, count(*) OVER() as total_count FROM script_runtime_model WHERE tenant_id = :tenant_id AND project_id is NULL %s`
	queryMap["SelectScriptRuntimesByProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM script_runtime_model WHERE tenant_id = :tenant_id AND (project_id is NULL OR project_id IN (:project_ids)) %s`

	queryMap["CreateScriptRuntime"] = `INSERT INTO script_runtime_model (id, version, tenant_id, name, description, language, builtin, docker_repo_uri, docker_profile_id, dockerfile, project_id, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :language, :builtin, :docker_repo_uri, :docker_profile_id, :dockerfile, :project_id, :created_at, :updated_at)`
	queryMap["UpdateScriptRuntime"] = `UPDATE script_runtime_model SET version = :version, tenant_id = :tenant_id, name = :name, description = :description, language = :language, builtin = :builtin, docker_repo_uri = :docker_repo_uri, docker_profile_id = :docker_profile_id, dockerfile = :dockerfile, project_id = :project_id, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	orderByHelper.Setup(entityTypeScriptRuntime, []string{"id", "version", "created_at", "updated_at", "name", "description", "language", "docker_repo_uri", "docker_profile_id", "project_id"})
}

// ScriptRuntimeDBO is DB object model for script
type ScriptRuntimeDBO struct {
	model.BaseModelDBO
	model.ScriptRuntimeCore
	ProjectID *string `json:"projectId,omitempty" db:"project_id"`
}

func (app ScriptRuntimeDBO) GetProjectID() string {
	if app.ProjectID != nil {
		return *app.ProjectID
	}
	return ""
}

type ScriptRuntimeProjects struct {
	ScriptRuntimeDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for script runtime
func getScriptRuntimeDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := ScriptRuntimeDBO{BaseModelDBO: tenantModel}
	var projectIDs []string
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		projectIDs = []string{projectID}
	} else {
		projectIDs = auth.GetProjectIDs(authContext)
	}
	if len(projectIDs) == 0 {
		return base.InQueryParam{
			Param: ScriptRuntimeProjects{
				ScriptRuntimeDBO: param,
				ProjectIDs:       projectIDs,
			},
			Key:     "SelectScriptRuntimesTemplate1",
			InQuery: false,
		}, nil
	}
	return base.InQueryParam{
		Param: ScriptRuntimeProjects{
			ScriptRuntimeDBO: param,
			ProjectIDs:       projectIDs,
		},
		Key:     "SelectScriptRuntimesByProjectsTemplate1",
		InQuery: true,
	}, nil
}

func canModify(context context.Context, dbAPI *dbObjectModelAPI, runtimeID string) (bool, error) {
	scriptIDs, err := dbAPI.SelectScriptsByRuntimeID(context, runtimeID)
	if err != nil {
		return false, err
	}
	dsIDs, err := dbAPI.GetDataStreamIDs(context, scriptIDs)
	if err != nil {
		return false, err
	}
	cm := len(dsIDs) == 0
	return cm, nil
}

func validateScriptRuntime(context context.Context, dbAPI *dbObjectModelAPI, doc *model.ScriptRuntime) error {
	if doc.Builtin {
		// don't allow creation of builtin script runtime via REST API
		return errcode.NewBadRequestError("builtin")
	}
	// allow empty DockerProfileID for public docker registry
	if doc.DockerProfileID != "" {
		_, err := dbAPI.GetDockerProfile(context, doc.DockerProfileID)
		if err != nil {
			return errcode.NewBadRequestError("dockerProfileId/GET")
		}
	}

	if doc.ProjectID != "" {
		project, err := dbAPI.GetProject(context, doc.ProjectID)
		if err != nil {
			return errcode.NewBadRequestError("projectId/GET")
		}
		if doc.DockerProfileID != "" {
			if !funk.Contains(project.DockerProfileIDs, doc.DockerProfileID) {
				return errcode.NewPermissionDeniedError("RBAC/ScriptRuntime/DockerProfile")
			}
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getScriptRuntimesForPage(ctx context.Context, dbQueryParam base.InQueryParam, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ScriptRuntime, base.PageToken, error) {
	scriptRuntimes := []model.ScriptRuntime{}
	if dbQueryParam.Key == "" {
		return scriptRuntimes, base.NilPageToken, nil
	}
	var queryFn func(context.Context, base.PageToken, int, func(interface{}) error, string, interface{}) (base.PageToken, error)
	if dbQueryParam.InQuery {
		queryFn = dbAPI.NotPagedQueryIn
	} else {
		queryFn = dbAPI.NotPagedQuery
	}
	query, err := buildQuery(entityTypeScriptRuntime, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return scriptRuntimes, base.NilPageToken, err
	}
	nextToken, err := queryFn(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		scriptRuntime := model.ScriptRuntime{}
		err := base.Convert(dbObjPtr, &scriptRuntime)
		if err != nil {
			return err
		}
		scriptRuntimes = append(scriptRuntimes, scriptRuntime)
		return nil
	}, query, dbQueryParam.Param)

	return scriptRuntimes, nextToken, err
}

// internal API used by getScriptRuntimesWV2
func (dbAPI *dbObjectModelAPI) getScriptRuntimesByProjectsForQuery(context context.Context, projectIDs []string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.ScriptRuntime, int, error) {
	scriptRuntimes := []model.ScriptRuntime{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return scriptRuntimes, 0, err
	}
	tenantID := authContext.TenantID
	scriptRuntimeDBOs := []ScriptRuntimeDBO{}

	var query string
	if len(projectIDs) == 0 {
		query, err = buildLimitQuery(entityTypeScriptRuntime, queryMap["SelectScriptRuntimesTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return scriptRuntimes, 0, err
		}
		err = dbAPI.Query(context, &scriptRuntimeDBOs, query, tenantIDParam2{TenantID: tenantID})
	} else {
		query, err = buildLimitQuery(entityTypeScriptRuntime, queryMap["SelectScriptRuntimesByProjectsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return scriptRuntimes, 0, err
		}
		err = dbAPI.QueryIn(context, &scriptRuntimeDBOs, query, tenantIDParam2{TenantID: tenantID, ProjectIDs: projectIDs})
	}
	if err != nil {
		return scriptRuntimes, 0, err
	}
	if len(scriptRuntimeDBOs) == 0 {
		return scriptRuntimes, 0, nil
	}
	totalCount := 0
	first := true
	for _, scriptRuntimeDBO := range scriptRuntimeDBOs {
		scriptRuntime := model.ScriptRuntime{}
		if first {
			first = false
			if scriptRuntimeDBO.TotalCount != nil {
				totalCount = *scriptRuntimeDBO.TotalCount
			}
		}
		err := base.Convert(&scriptRuntimeDBO, &scriptRuntime)
		if err != nil {
			return []model.ScriptRuntime{}, 0, err
		}
		scriptRuntimes = append(scriptRuntimes, scriptRuntime)
	}
	return scriptRuntimes, totalCount, nil
}

func (dbAPI *dbObjectModelAPI) getScriptRuntimes(context context.Context, projectID string, scriptRuntimeID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ScriptRuntime, error) {
	dbQueryParam, err := getScriptRuntimeDBQueryParam(context, projectID, scriptRuntimeID)
	if err != nil {
		return []model.ScriptRuntime{}, err
	}
	if dbQueryParam.Key == "" {
		return []model.ScriptRuntime{}, nil
	}
	scripts, _, err := dbAPI.getScriptRuntimesForPage(context, dbQueryParam, entitiesQueryParam)
	return scripts, err
}

func (dbAPI *dbObjectModelAPI) getScriptRuntimesW(ctx context.Context, projectID string, scriptRuntimeID string, w io.Writer, req *http.Request) error {
	scriptRuntimeDBOs := []ScriptRuntimeDBO{}
	dbQueryParam, err := getScriptRuntimeDBQueryParam(ctx, projectID, scriptRuntimeID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		if len(scriptRuntimeID) == 0 {
			return json.NewEncoder(w).Encode([]model.ScriptRuntime{})
		}
		return errcode.NewRecordNotFoundError(scriptRuntimeID)
	}
	var queryFn func(context.Context, interface{}, string, interface{}) error
	if dbQueryParam.InQuery {
		queryFn = dbAPI.QueryIn
	} else {
		queryFn = dbAPI.Query
	}
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	query, err := buildQuery(entityTypeScriptRuntime, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return err
	}
	err = queryFn(ctx, &scriptRuntimeDBOs, query, dbQueryParam.Param)
	if err != nil {
		return err
	}
	if len(scriptRuntimeID) == 0 {
		return base.DispatchPayload(w, scriptRuntimeDBOs)
	}
	if len(scriptRuntimeDBOs) == 0 {
		return errcode.NewRecordNotFoundError(scriptRuntimeID)
	}
	return json.NewEncoder(w).Encode(scriptRuntimeDBOs[0])
}

func (dbAPI *dbObjectModelAPI) getScriptRuntimesWV2(context context.Context, projectID string, scriptRuntimeID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getScriptRuntimeDBQueryParam(context, projectID, scriptRuntimeID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.ScriptRuntimeListPayload{ScriptRuntimeList: []model.ScriptRuntime{}})
	}
	projectIDs := dbQueryParam.Param.(ScriptRuntimeProjects).ProjectIDs
	queryParam := model.GetEntitiesQueryParam(req)

	scriptruntimes, totalCount, err := dbAPI.getScriptRuntimesByProjectsForQuery(context, projectIDs, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeScriptRuntime}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.ScriptRuntimeListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		ScriptRuntimeList:         scriptruntimes,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllScriptRuntimes select all script runtimes for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllScriptRuntimes(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ScriptRuntime, error) {
	return dbAPI.getScriptRuntimes(context, "", "", entitiesQueryParam)
}

// SelectAllScriptRuntimesW select all script runtimes for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptRuntimesW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptRuntimesW(context, "", "", w, req)
}

// SelectAllScriptRuntimesWV2 select all script runtimes for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptRuntimesWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptRuntimesWV2(context, "", "", w, req)
}

// SelectAllScriptRuntimesForProject select all script runtimes for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllScriptRuntimesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ScriptRuntime, error) {
	return dbAPI.getScriptRuntimes(context, projectID, "", entitiesQueryParam)
}

// SelectAllScriptRuntimesForProjectW select all script runtimes for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptRuntimesForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptRuntimesW(context, projectID, "", w, req)
}

// SelectAllScriptRuntimesForProjectWV2 select all script runtimes for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptRuntimesForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptRuntimesWV2(context, projectID, "", w, req)
}

// GetScriptRuntime get a script runtime object in the DB
func (dbAPI *dbObjectModelAPI) GetScriptRuntime(context context.Context, id string) (model.ScriptRuntime, error) {
	if len(id) == 0 {
		return model.ScriptRuntime{}, errcode.NewBadRequestError("scriptRuntimeID")
	}
	scriptRuntimes, err := dbAPI.getScriptRuntimes(context, "", id, nil)
	if err != nil {
		return model.ScriptRuntime{}, err
	}
	if len(scriptRuntimes) == 0 {
		return model.ScriptRuntime{}, errcode.NewRecordNotFoundError(id)
	}
	return scriptRuntimes[0], nil
}

// GetScriptRuntimeW get a script runtime object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetScriptRuntimeW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("scriptRuntimeID")
	}
	return dbAPI.getScriptRuntimesW(context, "", id, w, req)
}

// CreateScriptRuntime creates a script runtime object in the DB
func (dbAPI *dbObjectModelAPI) CreateScriptRuntime(context context.Context, i interface{} /* *model.ScriptRuntime */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ScriptRuntime)
	if !ok {
		return resp, errcode.NewInternalError("CreateScriptRuntime: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateScriptRuntime doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateScriptRuntime doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateScriptRuntime(&doc)
	if err != nil {
		return resp, err
	}
	if doc.Builtin {
		// don't allow creation of builtin runtimes via REST API
		return resp, errcode.NewBadRequestError("builtin")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityScriptRuntime,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	err = validateScriptRuntime(context, dbAPI, &doc)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	scriptRuntimeDBO := ScriptRuntimeDBO{}
	err = base.Convert(&doc, &scriptRuntimeDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateScriptRuntime"], &scriptRuntimeDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating script runtime for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addRuntimeEnvironmentAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateScriptRuntimeW creates a script runtime object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateScriptRuntimeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateScriptRuntime, &model.ScriptRuntime{}, w, r, callback)
}

// CreateScriptRuntimeWV2 creates a script runtime object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateScriptRuntimeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateScriptRuntime), &model.ScriptRuntime{}, w, r, callback)
}

// UpdateScriptRuntime update a script runtime object in the DB
func (dbAPI *dbObjectModelAPI) UpdateScriptRuntime(context context.Context, i interface{} /* *model.ScriptRuntime */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ScriptRuntime)
	if !ok {
		return resp, errcode.NewInternalError("UpdateScriptRuntime: type error")
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
	err = model.ValidateScriptRuntime(&doc)
	if err != nil {
		return resp, err
	}
	sr, err := dbAPI.GetScriptRuntime(context, doc.ID)
	if err != nil {
		return resp, errcode.NewBadRequestError("scriptRuntimeID")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityScriptRuntime,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:    doc.ProjectID,
			OldProjectID: sr.ProjectID,
			ProjNameFn:   GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	err = validateScriptRuntime(context, dbAPI, &doc)
	if err != nil {
		return resp, err
	}
	if doc.Builtin != sr.Builtin {
		// don't allow change of builtin flag
		return resp, errcode.NewBadRequestError("builtin<>")
	}

	// forbid modification of in-use script runtimes
	cm, err := canModify(context, dbAPI, doc.ID)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("UpdateScriptRuntime: canModify: %s", err.Error()))
	}
	if !cm {
		return resp, errcode.NewRecordInUseError()
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	scriptRuntimeDBO := ScriptRuntimeDBO{}
	err = base.Convert(&doc, &scriptRuntimeDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateScriptRuntime"], &scriptRuntimeDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating script runtime for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addRuntimeEnvironmentAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateScriptRuntimeW update a script runtime object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateScriptRuntimeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateScriptRuntime, &model.ScriptRuntime{}, w, r, callback)
}

// UpdateScriptRuntimeWV2 update a script runtime object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateScriptRuntimeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateScriptRuntime), &model.ScriptRuntime{}, w, r, callback)
}

// DeleteScriptRuntime delete a script runtime object in the DB
func (dbAPI *dbObjectModelAPI) DeleteScriptRuntime(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	sr, err := dbAPI.GetScriptRuntime(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	if sr.Builtin {
		// don't allow deletion of builtin runtimes
		return resp, errcode.NewBadRequestExError("name", fmt.Sprintf("Deletion of builtin runtime [%s] not allowed", sr.Name))
	}
	// forbid modification of in-use script runtimes
	cm, err := canModify(context, dbAPI, id)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("DeleteScriptRuntime: canModify: %s", err.Error()))
	}
	if !cm {
		return resp, errcode.NewRecordInUseError()
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityScriptRuntime,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  sr.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	result, err := DeleteEntity(context, dbAPI, "script_runtime_model", "id", id, sr, callback)
	if err == nil {
		GetAuditlogHandler().addRuntimeEnvironmentAuditLog(dbAPI, context, sr, DELETE)
	}
	return result, err
}

// DeleteScriptRuntimeW delete a script object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteScriptRuntimeW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteScriptRuntime, id, w, callback)
}

// DeleteScriptRuntimeWV2 delete a script object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteScriptRuntimeWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteScriptRuntime), id, w, callback)
}

// GetBuiltinScriptRuntimeID creates the builtin script runtime ID.
// TODO replace with MD5 hash later
func GetBuiltinScriptRuntimeID(tenantID string, suffix string) string {
	return fmt.Sprintf("%s_%s", tenantID, suffix)
}

func createBuiltinScriptRuntimes(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	now := base.RoundedNow()
	for _, scriptRuntime := range config.BuiltinScriptRuntimes {
		scriptRuntime.ID = GetBuiltinScriptRuntimeID(tenantID, scriptRuntime.ID)
		scriptRuntime.TenantID = tenantID
		scriptRuntime.Version = float64(now.UnixNano())
		scriptRuntime.CreatedAt = now
		scriptRuntime.UpdatedAt = now
		scriptRuntimeDBO := ScriptRuntimeDBO{}
		err := base.Convert(&scriptRuntime, &scriptRuntimeDBO)
		if err != nil {
			return nil
		}
		_, err = tx.NamedExec(ctx, queryMap["CreateScriptRuntime"], &scriptRuntimeDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating script runtime for ID %s and tenant ID %s. Error: %s"), scriptRuntime.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(scriptRuntime.ID, err)
		}
	}
	return nil
}

func deleteBuiltinScriptRuntimes(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	for _, scriptRuntime := range config.BuiltinScriptRuntimes {
		id := GetBuiltinScriptRuntimeID(tenantID, scriptRuntime.ID)
		_, err := base.DeleteTxn(ctx, tx, "script_runtime_model", map[string]interface{}{"id": id})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Deleting builtIn script runtime %+v with id %s. Error: %s"), scriptRuntime, id, err.Error())
			return err
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getScriptRuntimesByIDs(ctx context.Context, scriptRuntimeIDs []string) ([]model.ScriptRuntime, error) {
	scriptruntimes := []model.ScriptRuntime{}
	if len(scriptRuntimeIDs) == 0 {
		return scriptruntimes, nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID
	s := strings.Join(scriptRuntimeIDs, "', '")
	query := fmt.Sprintf("select * from script_runtime_model where tenant_id = '%s' and id in ('%s')", tenantID, s)
	scriptRuntimeDBOs := []ScriptRuntimeDBO{}
	err = dbAPI.Query(ctx, &scriptRuntimeDBOs, query, struct{}{})
	if err != nil {
		return nil, err
	}
	for _, scriptRuntimeDBO := range scriptRuntimeDBOs {
		scriptRuntime := model.ScriptRuntime{}
		err := base.Convert(&scriptRuntimeDBO, &scriptRuntime)
		if err != nil {
			return []model.ScriptRuntime{}, err
		}
		scriptruntimes = append(scriptruntimes, scriptRuntime)
	}
	return scriptruntimes, err
}

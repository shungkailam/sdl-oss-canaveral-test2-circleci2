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
	funk "github.com/thoas/go-funk"
)

const entityTypeScript = "script"

var builtinRuntimeEnvironments = []string{"python-env", "python2-env", "tensorflow-python", "node-env", "golang-env"}

func init() {
	queryMap["SelectScriptsTemplate1"] = `SELECT * FROM script_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND project_id is NULL %s`
	queryMap["SelectScriptsByProjectsTemplate1"] = `SELECT * FROM script_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (project_id is NULL OR project_id IN (:project_ids)) %s`
	queryMap["SelectScriptsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM script_model WHERE tenant_id = :tenant_id AND project_id is NULL %s`
	queryMap["SelectScriptsByProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM script_model WHERE tenant_id = :tenant_id AND (project_id is NULL OR project_id IN (:project_ids)) %s`
	queryMap["CreateScript"] = `INSERT INTO script_model (id, version, tenant_id, name, description, type, language, environment, code, params, runtime_id, runtime_tag, builtin, project_id, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :type, :language, :environment, :code, :params, :runtime_id, :runtime_tag, :builtin, :project_id, :created_at, :updated_at)`
	queryMap["UpdateScript"] = `UPDATE script_model SET version = :version, tenant_id = :tenant_id, name = :name, description = :description, type = :type, language = :language, environment = :environment, code = :code, params = :params, runtime_id = :runtime_id, runtime_tag = :runtime_tag, builtin = :builtin, project_id = :project_id, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectScriptsByRuntimeID"] = `SELECT id FROM script_model WHERE tenant_id = :tenant_id AND runtime_id = :runtime_id`
	queryMap["SelectScriptIDsTemplate"] = `SELECT id from script_model where tenant_id = :tenant_id AND project_id is NULL %s`
	queryMap["SelectScriptIDsInProjectsTemplate"] = `SELECT id from script_model where tenant_id = :tenant_id AND (project_id is NULL OR project_id IN (:project_ids)) %s`

	orderByHelper.Setup(entityTypeScript, []string{"id", "version", "created_at", "updated_at", "name", "description", "type", "language", "environment", "code", "runtime_id", "project_id"})
}

// ScriptDBO is DB object model for script
type ScriptDBO struct {
	model.BaseModelDBO
	model.ScriptCoreDBO
	Params types.JSONText `json:"params" db:"params"`
}

func (doc ScriptDBO) GetProjectID() string {
	if doc.ProjectID != nil {
		return *doc.ProjectID
	}
	return ""
}

type ScriptProjects struct {
	ScriptDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for script
func getScriptDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := ScriptDBO{BaseModelDBO: tenantModel}
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
		// global scripts only
		return base.InQueryParam{
			Param: ScriptProjects{
				ScriptDBO:  param,
				ProjectIDs: projectIDs,
			},
			Key:     "SelectScriptsTemplate1",
			InQuery: false,
		}, nil
	}
	return base.InQueryParam{
		Param: ScriptProjects{
			ScriptDBO:  param,
			ProjectIDs: projectIDs,
		},
		Key:     "SelectScriptsByProjectsTemplate1",
		InQuery: true,
	}, nil
}

// This function returns list of data pipeline IDs using the script with the given ID.
// It is used in check to not allow update/delete if script in use by data stream as a transformation.
func dataPipelineIDsUsingScript(context context.Context, dbAPI *dbObjectModelAPI, scriptID string) ([]string, error) {
	dsIDs, err := dbAPI.GetDataStreamIDs(context, []string{scriptID})
	if err != nil {
		return []string{}, err
	}
	return dsIDs, nil
}

func validateScript(context context.Context, dbAPI *dbObjectModelAPI, doc *model.Script) error {
	if doc.Builtin {
		// don't allow creation of builtin script via REST API
		return errcode.NewBadRequestError("builtin")
	}
	if doc.RuntimeID != "" {
		runtime, err := dbAPI.GetScriptRuntime(context, doc.RuntimeID)
		if err != nil {
			return errcode.NewBadRequestError("runtimeId/GET")
		}
		// environment ignored by edge in this case, so no need to validate
		if runtime.ProjectID != "" {
			if doc.ProjectID != "" {
				// project id must match in this case
				if runtime.ProjectID != doc.ProjectID {
					return errcode.NewPermissionDeniedError("RBAC/Script/Runtime")
				}
			} else {
				// although script is global, user still need RBAC for the runtime
				authContext, err := base.GetAuthContext(context)
				if err != nil {
					return err
				}
				if !auth.IsProjectMember(runtime.ProjectID, authContext) {
					return errcode.NewPermissionDeniedError("RBAC/Runtime")
				}
			}
		}
	} else {
		// must use environment of one of the builtin runtimes
		if !funk.Contains(builtinRuntimeEnvironments, doc.Environment) {
			return errcode.NewBadRequestError("environment")
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getScriptsForPage(ctx context.Context, dbQueryParam base.InQueryParam, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Script, base.PageToken, error) {
	scripts := []model.Script{}
	if dbQueryParam.Key == "" {
		return scripts, base.NilPageToken, nil
	}
	var queryFn func(context.Context, base.PageToken, int, func(interface{}) error, string, interface{}) (base.PageToken, error)
	if dbQueryParam.InQuery {
		queryFn = dbAPI.NotPagedQueryIn
	} else {
		queryFn = dbAPI.NotPagedQuery
	}
	query, err := buildQuery(entityTypeScript, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return scripts, base.NilPageToken, err
	}
	nextToken, err := queryFn(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		script := model.Script{}
		err := base.Convert(dbObjPtr, &script)
		if err != nil {
			return err
		}
		scripts = append(scripts, script)
		return nil
	}, query, dbQueryParam.Param)
	return scripts, nextToken, err
}

// internal API used by getScriptsWV2
func (dbAPI *dbObjectModelAPI) getScriptsByProjectsForQuery(context context.Context, projectIDs []string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.Script, int, error) {
	scripts := []model.Script{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return scripts, 0, err
	}
	tenantID := authContext.TenantID
	scriptDBOs := []ScriptDBO{}
	var query string
	if len(projectIDs) == 0 {
		query, err = buildLimitQuery(entityTypeScript, queryMap["SelectScriptsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return scripts, 0, err
		}
		err = dbAPI.Query(context, &scriptDBOs, query, tenantIDParam2{TenantID: tenantID})
	} else {
		query, err = buildLimitQuery(entityTypeScript, queryMap["SelectScriptsByProjectsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return scripts, 0, err
		}
		err = dbAPI.QueryIn(context, &scriptDBOs, query, tenantIDParam2{TenantID: tenantID, ProjectIDs: projectIDs})
	}
	if err != nil {
		return scripts, 0, err
	}
	if len(scriptDBOs) == 0 {
		return scripts, 0, nil
	}
	totalCount := 0
	first := true
	for _, scriptDBO := range scriptDBOs {
		script := model.Script{}
		if first {
			first = false
			if scriptDBO.TotalCount != nil {
				totalCount = *scriptDBO.TotalCount
			}
		}
		err := base.Convert(&scriptDBO, &script)
		if err != nil {
			return []model.Script{}, 0, err
		}
		scripts = append(scripts, script)
	}
	return scripts, totalCount, nil
}

func (dbAPI *dbObjectModelAPI) getScripts(context context.Context, projectID string, scriptID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Script, error) {
	dbQueryParam, err := getScriptDBQueryParam(context, projectID, scriptID)
	if err != nil {
		return []model.Script{}, err
	}
	if dbQueryParam.Key == "" {
		return []model.Script{}, nil
	}
	scripts, _, err := dbAPI.getScriptsForPage(context, dbQueryParam, entitiesQueryParam)
	return scripts, err
}

func (dbAPI *dbObjectModelAPI) getScriptsW(ctx context.Context, projectID string, scriptID string, w io.Writer, req *http.Request) error {
	scriptDBOs := []ScriptDBO{}
	dbQueryParam, err := getScriptDBQueryParam(ctx, projectID, scriptID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		if len(scriptID) == 0 {
			return json.NewEncoder(w).Encode([]model.Script{})
		}
		return errcode.NewRecordNotFoundError(scriptID)
	}
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	query, err := buildQuery(entityTypeScript, queryMap[dbQueryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return err
	}
	var queryFn func(context.Context, interface{}, string, interface{}) error
	if dbQueryParam.InQuery {
		queryFn = dbAPI.QueryIn
	} else {
		queryFn = dbAPI.Query
	}
	err = queryFn(ctx, &scriptDBOs, query, dbQueryParam.Param)
	if err != nil {
		return err
	}
	if len(scriptID) == 0 {
		return base.DispatchPayload(w, scriptDBOs)
	}
	if len(scriptDBOs) == 0 {
		return errcode.NewRecordNotFoundError(scriptID)
	}
	return json.NewEncoder(w).Encode(scriptDBOs[0])
}
func (dbAPI *dbObjectModelAPI) getScriptsWV2(context context.Context, projectID string, scriptID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getScriptDBQueryParam(context, projectID, scriptID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.ScriptListPayload{ScriptList: []model.Script{}})
	}
	projectIDs := dbQueryParam.Param.(ScriptProjects).ProjectIDs
	queryParam := model.GetEntitiesQueryParam(req)

	scripts, totalCount, err := dbAPI.getScriptsByProjectsForQuery(context, projectIDs, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeScript}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.ScriptListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		ScriptList:                scripts,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllScripts select all scripts for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllScripts(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Script, error) {
	return dbAPI.getScripts(context, "", "", entitiesQueryParam)
}

// SelectAllScriptsW select all scripts for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptsW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptsW(context, "", "", w, req)
}

// SelectAllScriptsWV2 select all scripts for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptsWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptsWV2(context, "", "", w, req)
}

// SelectAllScriptsForProject select all scripts for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllScriptsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Script, error) {
	return dbAPI.getScripts(context, projectID, "", entitiesQueryParam)
}

// SelectAllScriptsForProjectW select all scripts for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptsForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptsW(context, projectID, "", w, req)
}

// SelectAllScriptsForProjectWV2 select all scripts for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllScriptsForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getScriptsWV2(context, projectID, "", w, req)
}

// GetScript get a script object in the DB
func (dbAPI *dbObjectModelAPI) GetScript(context context.Context, id string) (model.Script, error) {
	if len(id) == 0 {
		return model.Script{}, errcode.NewBadRequestError("scriptID")
	}
	scripts, err := dbAPI.getScripts(context, "", id, nil)
	if err != nil {
		return model.Script{}, err
	}
	if len(scripts) == 0 {
		return model.Script{}, errcode.NewRecordNotFoundError(id)
	}
	return scripts[0], nil
}

// GetScriptW get a script object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetScriptW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("scriptID")
	}
	return dbAPI.getScriptsW(context, "", id, w, req)
}

// CreateScript creates a script object in the DB
func (dbAPI *dbObjectModelAPI) CreateScript(context context.Context, i interface{} /* *model.Script */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.Script)
	if !ok {
		return resp, errcode.NewInternalError("CreateScript: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateScript doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateScript doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = model.ValidateScript(&doc)
	if err != nil {
		return resp, err
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityScript,
		meta.OperationCreate,
		auth.RbacContext{
			ProjectID:  doc.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	err = validateScript(context, dbAPI, &doc)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	scriptDBO := ScriptDBO{}
	err = base.Convert(&doc, &scriptDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateScript"], &scriptDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating script for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addFunctionAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateScriptW creates a script object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateScriptW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateScript, &model.Script{}, w, r, callback)
}

// CreateScriptWV2 creates a script object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateScriptWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateScript), &model.Script{}, w, r, callback)
}

// UpdateScript update a script object in the DB
func (dbAPI *dbObjectModelAPI) UpdateScript(context context.Context, i interface{} /* *model.ScriptForceUpdate */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	scriptW, ok := i.(*model.ScriptForceUpdate)
	if !ok {
		return resp, errcode.NewInternalError("UpdateScript: type error")
	}
	p := &scriptW.Doc
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	if p.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	err = model.ValidateScript(&doc)
	if err != nil {
		return resp, err
	}
	sr, err := dbAPI.GetScript(context, doc.ID)
	if err != nil {
		return resp, errcode.NewBadRequestError("scriptID")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityScript,
		meta.OperationUpdate,
		auth.RbacContext{
			ProjectID:    doc.ProjectID,
			OldProjectID: sr.ProjectID,
			ProjNameFn:   GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	if doc.Builtin != sr.Builtin {
		// don't allow change of builtin flag
		return resp, errcode.NewBadRequestError("builtin<>")
	}
	err = validateScript(context, dbAPI, &doc)
	if err != nil {
		return resp, err
	}
	if !scriptW.ForceUpdate {
		// forbid modification of in-use script
		dsIDs, err := dataPipelineIDsUsingScript(context, dbAPI, doc.ID)
		if err != nil {
			return resp, errcode.NewInternalError(fmt.Sprintf("UpdateScript: canModify: %s", err.Error()))
		}
		if len(dsIDs) != 0 {
			// only update name and description are allowed in this case
			if !model.ScriptsDifferOnlyByNameAndDesc(&doc, &sr) {
				dsNames, err := dbAPI.GetDataStreamNames(context, dsIDs)
				if err != nil {
					return resp, errcode.NewInternalError(fmt.Sprintf("UpdateScript: getPipelineNames: %s", err.Error()))
				}
				return resp, errcode.NewRecordInUseExError("Function", "Data Pipelines", strings.Join(dsNames, ", "))
			}
			if doc.Name == sr.Name && doc.Description == sr.Description {
				// no change - no op
				return resp, nil
			}
		}
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	scriptDBO := ScriptDBO{}
	err = base.Convert(&doc, &scriptDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateScript"], &scriptDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating script for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addFunctionAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateScriptW update a script object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateScriptW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateScript, &model.ScriptForceUpdate{}, w, r, callback)
}

// UpdateScriptWV2 update a script object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateScriptWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateScript), &model.ScriptForceUpdate{}, w, r, callback)
}

// DeleteScript delete a script object in the DB
func (dbAPI *dbObjectModelAPI) DeleteScript(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	sr, err := dbAPI.GetScript(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	if sr.Builtin {
		// don't allow deletion of builtin script
		return resp, errcode.NewBadRequestExError("name", fmt.Sprintf("Deletion of builtin script [%s] not allowed", sr.Name))
	}
	// forbid modification of in-use script
	dsIDs, err := dataPipelineIDsUsingScript(context, dbAPI, id)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("DeleteScript: canModify: %s", err.Error()))
	}
	if len(dsIDs) != 0 {
		dsNames, err := dbAPI.GetDataStreamNames(context, dsIDs)
		if err != nil {
			return resp, errcode.NewInternalError(fmt.Sprintf("DeleteScript: getPipelineNames: %s", err.Error()))
		}
		return resp, errcode.NewRecordInUseExError("Function", "Data Pipelines", strings.Join(dsNames, ", "))
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityScript,
		meta.OperationDelete,
		auth.RbacContext{
			ProjectID:  sr.ProjectID,
			ProjNameFn: GetProjectNameFn(context, dbAPI),
		})
	if err != nil {
		return resp, err
	}
	result, err := DeleteEntity(context, dbAPI, "script_model", "id", id, sr, callback)
	if err == nil {
		GetAuditlogHandler().addFunctionAuditLog(dbAPI, context, sr, DELETE)
	}
	return result, err
}

// DeleteScriptW delete a script object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteScriptW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteScript, id, w, callback)
}

// DeleteScriptWV2 delete a script object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteScriptWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteScript), id, w, callback)
}

// SelectScriptsByRuntimeID get IDs of all scripts using the given runtimeID
func (dbAPI *dbObjectModelAPI) SelectScriptsByRuntimeID(context context.Context, runtimeID string) ([]string, error) {
	scriptIDs := []string{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return scriptIDs, err
	}
	idObjs := []model.IDObj{}
	tenantID := authContext.TenantID
	param := model.RuntimeIDObj{TenantID: tenantID, RuntimeID: runtimeID}
	err = dbAPI.Query(context, &idObjs, queryMap["SelectScriptsByRuntimeID"], param)
	if err != nil {
		return scriptIDs, err
	}
	for _, idObj := range idObjs {
		scriptIDs = append(scriptIDs, idObj.ID)
	}
	return scriptIDs, nil
}

func (dbAPI *dbObjectModelAPI) getScriptsByIDs(ctx context.Context, scriptIDs []string) ([]model.Script, error) {
	scripts := []model.Script{}
	if len(scriptIDs) == 0 {
		return scripts, nil
	}

	scriptDBOs := []ScriptDBO{}
	if err := dbAPI.queryEntitiesByTenantAndIds(ctx, &scriptDBOs, "script_model", scriptIDs); err != nil {
		return nil, err
	}

	for _, scriptDBO := range scriptDBOs {
		script := model.Script{}
		err := base.Convert(&scriptDBO, &script)
		if err != nil {
			return []model.Script{}, err
		}
		scripts = append(scripts, script)
	}
	return scripts, nil
}

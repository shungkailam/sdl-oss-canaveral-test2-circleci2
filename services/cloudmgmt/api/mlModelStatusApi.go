package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	funk "github.com/thoas/go-funk"
)

func init() {
	queryMap["SelectMLModelsStatus"] =
		"SELECT * FROM machine_inference_status_model " +
			"WHERE tenant_id = :tenant_id " +
			"AND (:model_id = '' OR model_id = :model_id) " +
			"AND (:edge_id = '' OR edge_id = :edge_id)"
	// note: the query works for both create and update
	queryMap["CreateMLModelStatus"] =
		"INSERT INTO machine_inference_status_model " +
			"(tenant_id, edge_id, model_id, version, model_status, " +
			"created_at, updated_at) " +
			"VALUES (:tenant_id, :edge_id, " +
			":model_id, :version, :model_status, :created_at, :updated_at) " +
			"ON CONFLICT (tenant_id, edge_id, model_id) DO UPDATE SET " +
			"version = :version, model_status = :model_status, " +
			"updated_at = :updated_at " +
			"WHERE machine_inference_status_model.tenant_id = :tenant_id " +
			"AND machine_inference_status_model.edge_id = :edge_id " +
			"AND machine_inference_status_model.model_id = :model_id"
}

// MLModelStatusDBO is DB object model for application status
// Note: there is no ID field, as the composite
// (TenantID, EdgeID, MLModelID) serves as the primary key
type MLModelStatusDBO struct {
	Version   float64         `json:"version,omitempty" db:"version"`
	TenantID  string          `json:"tenantId" db:"tenant_id"`
	EdgeID    string          `json:"edgeId" db:"edge_id"`
	ModelID   string          `json:"modelId" db:"model_id"`
	CreatedAt time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time       `json:"updatedAt" db:"updated_at"`
	Status    *types.JSONText `json:"modelStatus" db:"model_status"`
	ProjectID *string         `json:"projectId" db:"project_id"`
}

func (mdl MLModelStatusDBO) GetID() string {
	return mdl.ModelID
}
func (mdl MLModelStatusDBO) GetProjectID() string {
	if mdl.ProjectID != nil {
		return *mdl.ProjectID
	}
	return ""
}

func (dbAPI *dbObjectModelAPI) selectAllMLModelStatusDBOs(
	context context.Context, modelID string,
) ([]MLModelStatusDBO, error) {
	mlModelStatusDBOs := []MLModelStatusDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return mlModelStatusDBOs, err
	}
	tenantID := authContext.TenantID
	param := MLModelStatusDBO{TenantID: tenantID, ModelID: modelID}
	// get ml model status entries
	err =
		dbAPI.Query(context, &mlModelStatusDBOs,
			queryMap["SelectMLModelsStatus"], param)
	if err != nil {
		return mlModelStatusDBOs, err
	}

	// get ml models user has access to for RBAC filtering
	// need this since MLModelStatus does not carry ProjectID
	models, err := dbAPI.SelectAllMLModels(context, nil)
	if err != nil {
		return mlModelStatusDBOs, err
	}
	mdlMap := map[string]*model.MLModel{}

	for i := range models {
		m := &models[i]
		mdlMap[m.ID] = m
	}

	// find all relevant project IDs
	// we need to fetch projects to filter out
	// entries from edges that are no longer in project
	noProjectID := "NO_PROJECT_ID"
	projMap := map[string]bool{}
	projectIDs := []string{}
	for i := range mlModelStatusDBOs {
		mlModelStatusDBO := &mlModelStatusDBOs[i]
		mdl := mdlMap[mlModelStatusDBO.ModelID]
		if mdl != nil {
			mlModelStatusDBO.ProjectID = &mdl.ProjectID
			if !projMap[mdl.ProjectID] {
				projMap[mdl.ProjectID] = true
				projectIDs = append(projectIDs, mdl.ProjectID)
			}
		} else {
			mlModelStatusDBO.ProjectID = &noProjectID
		}
	}

	// filter out stale edge id entries
	projects, err :=
		dbAPI.getProjectsByIDs(context, tenantID, projectIDs)
	if err != nil {
		return mlModelStatusDBOs, err
	}
	projMap2 := map[string]*model.Project{}
	for i := range projects {
		p := &projects[i]
		projMap2[p.ID] = p
	}
	activeMLModelStatusDBOs := []MLModelStatusDBO{}
	for _, mlDBO := range mlModelStatusDBOs {
		p := projMap2[*mlDBO.ProjectID]
		if p != nil && funk.Contains(p.EdgeIDs, mlDBO.EdgeID) {
			activeMLModelStatusDBOs = append(activeMLModelStatusDBOs, mlDBO)
		}
	}
	return activeMLModelStatusDBOs, nil
}

// SelectAllMLModelsStatus select all applications status for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllMLModelsStatus(
	context context.Context) ([]model.MLModelStatus, error) {
	mlModelsStatus := []model.MLModelStatus{}
	mlModelStatusDBOs, err := dbAPI.selectAllMLModelStatusDBOs(context, "")
	if err != nil {
		return mlModelsStatus, err
	}
	for _, mlModelStatusDBO := range mlModelStatusDBOs {
		mlModelStatus := model.MLModelStatus{}
		err := base.Convert(&mlModelStatusDBO, &mlModelStatus)
		if err != nil {
			return mlModelsStatus, err
		}
		mlModelsStatus = append(mlModelsStatus, mlModelStatus)
	}
	return mlModelsStatus, nil
}

func (dbAPI *dbObjectModelAPI) getAllMLModelsStatus(
	context context.Context, modelID string, w io.Writer, req *http.Request,
) error {
	mlModelStatusDBOs, err := dbAPI.selectAllMLModelStatusDBOs(context, modelID)
	if err != nil {
		return err
	}
	queryParam := model.GetEntitiesQueryParam(req)
	totalCount := len(mlModelStatusDBOs)
	startIndex := queryParam.PageIndex * queryParam.PageSize
	if startIndex >= totalCount {
		mlModelStatusDBOs = []MLModelStatusDBO{}
	} else {
		endIndex := startIndex + queryParam.PageSize
		if endIndex > totalCount {
			endIndex = totalCount
		}
		mlModelStatusDBOs = mlModelStatusDBOs[startIndex:endIndex]
	}
	mlModelStatuses := []model.MLModelStatus{}
	for _, mlModelStatusDBO := range mlModelStatusDBOs {
		mlModelStatus := model.MLModelStatus{}
		err = base.Convert(&mlModelStatusDBO, &mlModelStatus)
		if err != nil {
			return err
		}
		mlModelStatuses = append(mlModelStatuses, mlModelStatus)
	}

	queryInfo := ListQueryInfo{
		StartPage:  base.PageToken(""),
		TotalCount: totalCount,
	}
	pagedListResponsePayload :=
		makePagedListResponsePayload(queryParam, &queryInfo)
	r := model.MLModelStatusListPayload{
		PagedListResponsePayload: pagedListResponsePayload,
		MLModelStatusList:        mlModelStatuses,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllMLModelsStatusW select all applications status
// for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllMLModelsStatusW(
	context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getAllMLModelsStatus(context, "", w, req)
}

// GetMLModelStatus select all MLModel statuses
// for the MLModel with the given id
func (dbAPI *dbObjectModelAPI) GetMLModelStatus(
	context context.Context, modelID string,
) ([]model.MLModelStatus, error) {
	mlModelStatuses := []model.MLModelStatus{}
	mlModelStatusDBOs, err := dbAPI.selectAllMLModelStatusDBOs(context, modelID)
	if err != nil {
		return mlModelStatuses, err
	}
	for _, mlModelStatusDBO := range mlModelStatusDBOs {
		mlModelStatus := model.MLModelStatus{}
		err = base.Convert(&mlModelStatusDBO, &mlModelStatus)
		if err != nil {
			return []model.MLModelStatus{}, err
		}
		mlModelStatuses = append(mlModelStatuses, mlModelStatus)
	}
	return mlModelStatuses, nil
}

// GetMLModelStatusW select all MLModel statuses for the MLModel
// with the given id, write output into writer
func (dbAPI *dbObjectModelAPI) GetMLModelStatusW(
	context context.Context, modelID string, w io.Writer, req *http.Request,
) error {
	return dbAPI.getAllMLModelsStatus(context, modelID, w, req)
}

// CreateMLModelStatus creates an MLModel status object in the DB
func (dbAPI *dbObjectModelAPI) CreateMLModelStatus(
	context context.Context, i interface{}, /* *model.MLModelStatus */
	callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.MLModelStatus)
	if !ok {
		return resp, errcode.NewInternalError("CreateMLModelStatus: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	err = model.ValidateMLModelStatus(&doc)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	mlModelStatusDBO := MLModelStatusDBO{}

	err = base.Convert(&doc, &mlModelStatusDBO)
	if err != nil {
		return resp, err
	}

	_, err =
		dbAPI.NamedExec(context,
			queryMap["CreateMLModelStatus"], &mlModelStatusDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context,
			"Error in creating MLModel status for model ID %s "+
				"and tenant ID %s. Error: %s"),
			doc.ModelID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(mlModelStatusDBO.ModelID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ModelID
	return resp, nil
}

// CreateMLModelStatusW creates an application status object in the DB,
// write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateMLModelStatusW(
	context context.Context, w io.Writer, r io.Reader,
	callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateMLModelStatus),
		&model.MLModelStatus{}, w, r, callback)
}

// DeleteMLModelStatus delete MLModel status objects with the given tenantID
// and modelID in the DB
func (dbAPI *dbObjectModelAPI) DeleteMLModelStatus(
	context context.Context, modelID string,
	callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return model.DeleteDocumentResponse{}, err
	}
	doc := MLModelStatusDBO{
		TenantID: authContext.TenantID,
		ModelID:  modelID,
	}
	return DeleteEntity(context, dbAPI, "machine_inference_status_model",
		"model_id", modelID, doc, callback)
}

// DeleteMLModelStatusW delete application status objects with the given
// tenantID and modelID in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteMLModelStatusW(
	context context.Context, modelID string, w io.Writer,
	callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteMLModelStatus),
		modelID, w, callback)
}

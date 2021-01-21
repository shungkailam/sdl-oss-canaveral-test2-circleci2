package api

import (
	"cloudservices/common/auth"
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

const includeDisconnectedEdgesDefault = false

func init() {
	queryMap["SelectApplicationsStatus"] = `SELECT * FROM application_status_model WHERE tenant_id = :tenant_id AND (:application_id = '' OR application_id = :application_id) AND (:edge_id = '' OR edge_id = :edge_id)`
	// note: the query works for both create and update
	queryMap["CreateApplicationStatus"] = `INSERT INTO application_status_model (tenant_id, edge_id, application_id, version, app_status, created_at, updated_at) VALUES (:tenant_id, :edge_id, :application_id, :version, :app_status, :created_at, :updated_at) ON CONFLICT (tenant_id, edge_id, application_id) DO UPDATE SET version = :version, app_status = :app_status, updated_at = :updated_at WHERE application_status_model.tenant_id = :tenant_id AND application_status_model.edge_id = :edge_id AND application_status_model.application_id = :application_id`
}

// ApplicationStatusDBO is DB object model for application status
// Note: there is no ID field, as the composite
// (TenantID, EdgeID, ApplicationID) serves as the primary key
type ApplicationStatusDBO struct {
	Version       float64         `json:"version,omitempty" db:"version"`
	TenantID      string          `json:"tenantId" db:"tenant_id"`
	EdgeID        string          `json:"edgeId" db:"edge_id"`
	ApplicationID string          `json:"applicationId" db:"application_id"`
	CreatedAt     time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time       `json:"updatedAt" db:"updated_at"`
	AppStatus     *types.JSONText `json:"appStatus" db:"app_status"`
	ProjectID     *string         `json:"projectId" db:"project_id"`
}

func (app ApplicationStatusDBO) GetID() string {
	return app.ApplicationID
}
func (app ApplicationStatusDBO) GetProjectID() string {
	if app.ProjectID != nil {
		return *app.ProjectID
	}
	return ""
}

func getIncludeDisconnectedEdges(req *http.Request) bool {
	if req != nil {
		query := req.URL.Query()
		includeDisconnectedEdgesVals := query["includeDisconnectedEdges"]
		if len(includeDisconnectedEdgesVals) == 1 {
			return includeDisconnectedEdgesVals[0] == "true"
		}
	}
	return includeDisconnectedEdgesDefault
}

func (dbAPI *dbObjectModelAPI) selectAllApplicationStatusDBOs(context context.Context, applicationID string, includeDisconnectedEdges bool) ([]ApplicationStatusDBO, error) {
	applicationStatusDBOs := []ApplicationStatusDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return applicationStatusDBOs, err
	}
	tenantID := authContext.TenantID
	param := ApplicationStatusDBO{TenantID: tenantID, ApplicationID: applicationID}
	// Note: can't use PagedQuery as it requires "id" column
	err = dbAPI.Query(context, &applicationStatusDBOs, queryMap["SelectApplicationsStatus"], param)
	if err != nil {
		return applicationStatusDBOs, err
	}

	connectedEdgesMap := map[string]bool{}
	if !includeDisconnectedEdges {
		conntectedEdgeIDs, err := dbAPI.SelectConnectedEdgeClusterIDs(context)
		if err != nil {
			return applicationStatusDBOs, err
		}
		for _, edgeID := range conntectedEdgeIDs {
			connectedEdgesMap[edgeID] = true
		}
	}

	// filter application status
	applications, err := dbAPI.SelectAllApplications(context)
	if err != nil {
		return applicationStatusDBOs, err
	}
	appMap := map[string]model.Application{}
	for _, app := range applications {
		appMap[app.ID] = app
	}
	noProjectID := "NO_PROJECT_ID"
	for i := range applicationStatusDBOs {
		appStatus := &applicationStatusDBOs[i]
		app := appMap[appStatus.ApplicationID]
		appStatus.ProjectID = &app.ProjectID
		if app.ProjectID == "" {
			appStatus.ProjectID = &noProjectID
		}
	}
	applicationStatusDBOs = auth.FilterProjectScopedEntities(applicationStatusDBOs, authContext).([]ApplicationStatusDBO)
	// filter out stale edge id entries
	activeApplicationStatusDBOs := []ApplicationStatusDBO{}
	for _, asDBO := range applicationStatusDBOs {
		for _, app := range applications {
			if app.ID == asDBO.ApplicationID {
				// Unset application state is also DEPLOY for backward compatibility
				if (app.State == nil || *app.State == string(model.DeployEntityState)) &&
					funk.Contains(app.EdgeIDs, asDBO.EdgeID) {

					if includeDisconnectedEdges || connectedEdgesMap[asDBO.EdgeID] {
						activeApplicationStatusDBOs = append(activeApplicationStatusDBOs, asDBO)
					}
				}
				break
			}
		}
	}
	return activeApplicationStatusDBOs, nil
}

// SelectAllApplicationsStatus select all applications status for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsStatus(context context.Context, includeDisconnectedEdges bool) ([]model.ApplicationStatus, error) {
	applicationsStatus := []model.ApplicationStatus{}
	applicationStatusDBOs, err := dbAPI.selectAllApplicationStatusDBOs(context, "", includeDisconnectedEdges)
	if err != nil {
		return applicationsStatus, err
	}
	for _, asDBO := range applicationStatusDBOs {
		applicationStatus := model.ApplicationStatus{}
		err := base.Convert(&asDBO, &applicationStatus)
		if err != nil {
			return applicationsStatus, err
		}
		applicationsStatus = append(applicationsStatus, applicationStatus)
	}
	return applicationsStatus, nil
}

// SelectAllApplicationsStatusW select all applications status for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsStatusW(context context.Context, w io.Writer, req *http.Request) error {
	applicationStatusDBOs, err := dbAPI.selectAllApplicationStatusDBOs(context, "", getIncludeDisconnectedEdges(req))
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, applicationStatusDBOs)
}

func (dbAPI *dbObjectModelAPI) getAllApplicationsStatus(context context.Context, applicationID string, w io.Writer, req *http.Request) error {
	applicationStatusDBOs, err := dbAPI.selectAllApplicationStatusDBOs(context, applicationID, getIncludeDisconnectedEdges(req))
	if err != nil {
		return err
	}
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	applicationStatuses := []model.ApplicationStatus{}
	for _, applicationStatusDBO := range applicationStatusDBOs {
		applicationStatus := model.ApplicationStatus{}
		err = base.Convert(&applicationStatusDBO, &applicationStatus)
		if err != nil {
			return err
		}
		applicationStatuses = append(applicationStatuses, applicationStatus)
	}
	queryParam := model.GetEntitiesQueryParam(req)
	queryInfo := ListQueryInfo{
		StartPage:  base.PageToken(""),
		TotalCount: len(applicationStatuses),
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.ApplicationStatusListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		ApplicationStatusList:     applicationStatuses,
	}
	return encoder.Encode(r)
}

// SelectAllApplicationsStatusWV2 select all applications status for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllApplicationsStatusWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getAllApplicationsStatus(context, "", w, req)
}

// GetApplicationStatus select all application statuses for the app with the given id
func (dbAPI *dbObjectModelAPI) GetApplicationStatus(context context.Context, applicationID string) ([]model.ApplicationStatus, error) {
	applicationStatuses := []model.ApplicationStatus{}
	applicationStatusDBOs, err := dbAPI.selectAllApplicationStatusDBOs(context, applicationID, includeDisconnectedEdgesDefault)
	if err != nil {
		return applicationStatuses, err
	}
	for _, applicationStatusDBO := range applicationStatusDBOs {
		applicationStatus := model.ApplicationStatus{}
		err = base.Convert(&applicationStatusDBO, &applicationStatus)
		if err != nil {
			return []model.ApplicationStatus{}, err
		}
		applicationStatuses = append(applicationStatuses, applicationStatus)
	}
	return applicationStatuses, nil
}

// GetApplicationStatusW select all application statuses for the app with the given id, write output into writer
func (dbAPI *dbObjectModelAPI) GetApplicationStatusW(context context.Context, applicationID string, w io.Writer, req *http.Request) error {
	applicationStatusDBOs, err := dbAPI.selectAllApplicationStatusDBOs(context, applicationID, getIncludeDisconnectedEdges(req))
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, applicationStatusDBOs)
}

// GetApplicationStatusWV2 - similar to GetApplicationStatusW, but paged version with wrapped response
func (dbAPI *dbObjectModelAPI) GetApplicationStatusWV2(context context.Context, applicationID string, w io.Writer, req *http.Request) error {
	return dbAPI.getAllApplicationsStatus(context, applicationID, w, req)
}

// CreateApplicationStatus creates an application status object in the DB
func (dbAPI *dbObjectModelAPI) CreateApplicationStatus(context context.Context, i interface{} /* *model.ApplicationStatus */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	isEdgeReq, _ := base.IsEdgeRequest(authContext)
	// Only an edge must be able to call this API
	if !isEdgeReq {
		return resp, errcode.NewPermissionDeniedError("role")
	}
	p, ok := i.(*model.ApplicationStatus)
	if !ok {
		return resp, errcode.NewInternalError("CreateApplicationStatus: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	err = model.ValidateApplicationStatus(&doc)
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	applicationStatusDBO := ApplicationStatusDBO{}

	err = base.Convert(&doc, &applicationStatusDBO)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["CreateApplicationStatus"], &applicationStatusDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating application status for application ID %s and tenant ID %s. Error: %s"), doc.ApplicationID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(applicationStatusDBO.ApplicationID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ApplicationID
	return resp, nil
}

// CreateApplicationStatusW creates an application status object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateApplicationStatusW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateApplicationStatus, &model.ApplicationStatus{}, w, r, callback)
}

// CreateApplicationStatusWV2 creates an application status object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateApplicationStatusWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateApplicationStatus), &model.ApplicationStatus{}, w, r, callback)
}

// DeleteApplicationStatus delete application status objects with the given tenantID and applicationID in the DB
func (dbAPI *dbObjectModelAPI) DeleteApplicationStatus(context context.Context, applicationID string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return model.DeleteDocumentResponse{}, err
	}
	doc := model.ApplicationStatus{
		TenantID:      authContext.TenantID,
		ApplicationID: applicationID,
	}
	return DeleteEntity(context, dbAPI, "application_status_model", "application_id", applicationID, doc, callback)
}

// DeleteApplicationStatusW delete application status objects with the given tenantID and applicationID in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteApplicationStatusW(context context.Context, applicationID string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteApplicationStatus, applicationID, w, callback)
}

// DeleteApplicationStatusWV2 delete application status objects with the given tenantID and applicationID in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteApplicationStatusWV2(context context.Context, applicationID string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteApplicationStatus), applicationID, w, callback)
}

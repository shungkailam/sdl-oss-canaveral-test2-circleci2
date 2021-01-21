package api

import (
	"bytes"
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
)

const entityTypeContainerRegistry = "containerregistry"

func init() {
	// Reusing the docker_profile_model table
	queryMap["SelectContainerRegistriesTemplate1"] = `SELECT * FROM docker_profile_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) %s`
	queryMap["SelectContainerRegistriesTemplate"] = `SELECT *, count(*) OVER() as total_count FROM docker_profile_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) %s`
	queryMap["SelectContainerRegistriesByProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM docker_profile_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (id IN (SELECT docker_profile_id FROM project_docker_profile_model WHERE project_id IN (:project_ids))) %s`
	queryMap["SelectContainerRegistriesByProjectsTemplate1"] = `SELECT * FROM docker_profile_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (id IN (SELECT docker_profile_id FROM project_docker_profile_model WHERE project_id IN (:project_ids))) %s`
	queryMap["CreateContainerRegistry"] = `INSERT INTO docker_profile_model (id, version, tenant_id, name, description, type, server, user_name, email , pwd, cloud_creds_id, iflag_encrypted, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :type, :server, :user_name, :email , :pwd, :cloud_creds_id, :iflag_encrypted, :created_at, :updated_at)`
	queryMap["UpdateContainerRegistry"] = `UPDATE docker_profile_model SET version = :version, name = :name, description = :description, type = :type, server = :server, user_name = :user_name, email = :email ,pwd = :pwd, cloud_creds_id = :cloud_creds_id, iflag_encrypted = :iflag_encrypted, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectContainerRegistriesByIDs"] = `SELECT * FROM docker_profile_model WHERE tenant_id = :tenant_id AND (id IN (:docker_profile_ids))`

	orderByHelper.Setup(entityTypeContainerRegistry, []string{"id", "version", "created_at", "updated_at", "name", "description", "type", "cloud_creds_id"})
}

// // ContainerRegistryDBO is DB object model for ContainerRegistries
type ContainerRegistryDBO struct {
	model.BaseModelDBO
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	Type           string          `json:"type" db:"type"`
	Server         string          `json:"server" db:"server"`
	UserName       string          `json:"userName" db:"user_name"`
	Email          string          `json:"email" db:"email"`
	Pwd            string          `json:"pwd" db:"pwd"`
	CloudCredsID   *string         `json:"cloudCredsID" db:"cloud_creds_id"`
	IFlagEncrypted *bool           `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
	Credentials    *types.JSONText `json:"credentials,omitempty" db:"credentials"`
}

type ContainerRegistryProjects struct {
	ContainerRegistryDBO
	ProjectIDs base.StringArray `json:"project_ids" db:"project_ids"`
}

func (cr *ContainerRegistryDBO) MaskObject() {
	cr.Pwd = base.MaskString(cr.Pwd, "*", 0, 4)
}
func MaskContainerRegistryDBOs(crs []ContainerRegistryDBO) {
	for i := 0; i < len(crs); i++ {
		(&crs[i]).MaskObject()
	}
}

func getContainerRegistryDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := ContainerRegistryDBO{BaseModelDBO: tenantModel}
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		return base.InQueryParam{
			Param: ContainerRegistryProjects{
				ContainerRegistryDBO: param,
				ProjectIDs:           []string{projectID},
			},
			Key:     "SelectContainerRegistriesByProjectsTemplate1",
			InQuery: true,
		}, nil
	}
	if !auth.IsInfraAdminRole(authContext) {
		projectIDs := auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return base.InQueryParam{}, nil
		}
		return base.InQueryParam{
			Param: ContainerRegistryProjects{
				ContainerRegistryDBO: param,
				ProjectIDs:           projectIDs,
			},
			Key:     "SelectContainerRegistriesByProjectsTemplate1",
			InQuery: true,
		}, nil
	}
	return base.InQueryParam{
		Param:   param,
		Key:     "SelectContainerRegistriesTemplate1",
		InQuery: false,
	}, nil
}

func encryptContainerRegistry(context context.Context, dbAPI *dbObjectModelAPI, doc *model.ContainerRegistry) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	return encryptContainerRegistryT(context, dbAPI, doc, &tenant)
}

func encryptContainerRegistryT(context context.Context, dbAPI *dbObjectModelAPI, doc *model.ContainerRegistry, tenant *model.Tenant) error {
	if doc.Type == "GCP" || doc.Type == "ContainerRegistry" {
		pwd, err := keyService.TenantEncrypt(doc.Pwd, &crypto.Token{EncryptedToken: tenant.Token})
		if err != nil {
			return err
		}
		doc.Pwd = pwd
	} else if doc.Type == "AWS" {
		// creds, err := crypto.TenantEncrypt(doc.Credentials, tenant.Token)
		// if err != nil {
		// 	return err
		// }
		// doc.Credentials = creds
	}
	return nil
}

func encryptContainerRegistryDBO(context context.Context, dbAPI *dbObjectModelAPI, doc *ContainerRegistryDBO) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	return encryptContainerRegistryDBOT(context, dbAPI, doc, &tenant)
}

func encryptContainerRegistryDBOT(context context.Context, dbAPI *dbObjectModelAPI, doc *ContainerRegistryDBO, tenant *model.Tenant) error {
	if doc.Type == "GCP" || doc.Type == "ContainerRegistry" {
		pwd, err := keyService.TenantEncrypt(doc.Pwd, &crypto.Token{EncryptedToken: tenant.Token})
		if err != nil {
			return err
		}
		doc.Pwd = pwd
	} else if doc.Type == "AWS" {
		// creds, err := crypto.TenantEncrypt(*doc.Credentials, tenant.Token)
		// if err != nil {
		// 	return err
		// }
		// doc.Credentials = &creds
	}
	return nil
}

func encryptContainerRegistryDBOs(context context.Context, dbAPI *dbObjectModelAPI, docs []ContainerRegistryDBO) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	for i := range docs {
		err := encryptContainerRegistryDBOT(context, dbAPI, &docs[i], &tenant)
		if err != nil {
			return err
		}
	}
	return nil
}

func decryptContainerRegistry(context context.Context, dbAPI *dbObjectModelAPI, doc *model.ContainerRegistry) error {
	if !doc.IFlagEncrypted {
		return nil
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	return decryptContainerRegistryT(context, dbAPI, doc, token)
}

func decryptContainerRegistryT(context context.Context, dbAPI *dbObjectModelAPI, doc *model.ContainerRegistry, token *crypto.Token) error {
	if !doc.IFlagEncrypted {
		return nil
	}
	if doc.Type == "GCP" || doc.Type == "ContainerRegistry" {
		pwd, err := keyService.TenantDecrypt(doc.Pwd, token)
		if err != nil {
			return err
		}
		doc.Pwd = pwd
	} else if doc.Type == "AWS" {
		// creds, err := crypto.TenantDecrypt(doc.Credentials, tenant.Token)
		// if err != nil {
		// 	return err
		// }
		// doc.Credentials = creds
	}
	return nil
}

func decryptContainerRegistryDBO(context context.Context, dbAPI *dbObjectModelAPI, doc *ContainerRegistryDBO) error {
	if doc.IFlagEncrypted == nil || !*doc.IFlagEncrypted {
		return nil
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	return decryptContainerRegistryDBOT(context, dbAPI, doc, token)
}

func decryptContainerRegistryDBOT(context context.Context, dbAPI *dbObjectModelAPI, doc *ContainerRegistryDBO, token *crypto.Token) error {
	if doc.IFlagEncrypted == nil || !*doc.IFlagEncrypted {
		return nil
	}
	if doc.Type == "GCP" || doc.Type == "ContainerRegistry" {
		pwd, err := keyService.TenantDecrypt(doc.Pwd, token)
		if err != nil {
			return err
		}
		doc.Pwd = pwd
	} else if doc.Type == "AWS" {
		// cred, err := crypto.TenantDecrypt(*doc.Credentials, tenant.Token)
		// if err != nil {
		// 	return err
		// }
		// doc.Credentials = &cred
	}
	return nil
}

func decryptContainerRegistryDBOs(context context.Context, dbAPI *dbObjectModelAPI, docs []ContainerRegistryDBO) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	for i := range docs {
		err := decryptContainerRegistryDBOT(context, dbAPI, &docs[i], token)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) containerRegistryFromCLoudCreds(context context.Context, doc *model.ContainerRegistry, authContext *base.AuthContext) error {

	if len(doc.CloudCredsID) > 0 {
		// Extract email from auth context
		var email string
		var emailContx interface{}
		var ok bool
		if emailContx, ok = authContext.Claims["email"]; !ok {
			// Can not get email from auth context
			err := errcode.NewBadRequestError("Email")
			glog.Errorf(base.PrefixRequestID(context, "Error retriving Email from auth context. Error: %s"), err.Error())
			return err
		}
		if email, ok = emailContx.(string); !ok {
			//can not convert email to string
			err := errcode.NewBadRequestError("Email")
			glog.Errorf(base.PrefixRequestID(context, "Error converting Email to string from auth context. Error: %s"), err.Error())
			return err
		}
		doc.Email = email

		// Validate that cloud creds exist
		_, err := dbAPI.GetCloudCreds(context, doc.CloudCredsID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error retriving Cloud Creds with id. Error: %s"), err.Error())
			return err
		}
		if doc.Type == "AWS" {
			// validate the server, for now I am using a basic check to match 770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev

			server := strings.Split(doc.Server, ".")
			if len(server) != 6 {
				err = errcode.NewBadRequestExError("Server", fmt.Sprintf("Error getting account and region from server %s", doc.Server))
				glog.Errorf(base.PrefixRequestID(context, "Error getting account and region from server. Error: %s"), err.Error())
				return err
			}
		}
	}
	return nil
}

// SelectAllContainerRegistries select all ContainerRegistries for the given tenant
func (dbAPI *dbObjectModelAPI) getContainerRegistries(ctx context.Context, projectID string, containerRegistryID string, startPage base.PageToken, pageSize int, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ContainerRegistry, error) {
	var err error
	containerRegistries := []model.ContainerRegistry{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return containerRegistries, err
	}
	queryParam, err := getContainerRegistryDBQueryParam(ctx, projectID, containerRegistryID)
	if err != nil {
		return containerRegistries, err
	}
	if queryParam.Key == "" {
		return containerRegistries, nil
	}
	var pagedQueryFn func(context.Context, base.PageToken, int, func(interface{}) error, string, interface{}) (base.PageToken, error)

	if queryParam.InQuery {
		pagedQueryFn = dbAPI.NotPagedQueryIn
	} else {
		pagedQueryFn = dbAPI.NotPagedQuery
	}
	query, err := buildQuery(entityTypeContainerRegistry, queryMap[queryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return containerRegistries, err
	}
	_, err = pagedQueryFn(ctx, startPage, pageSize, func(dbObjPtr interface{}) error {
		containerRegistry := model.ContainerRegistry{}
		base.Convert(dbObjPtr, &containerRegistry)
		if err != nil {
			return err
		}
		err = decryptContainerRegistry(ctx, dbAPI, &containerRegistry)
		if err != nil {
			return err
		}
		containerRegistries = append(containerRegistries, containerRegistry)
		return nil
	}, query, queryParam.Param)

	// mask out credentials if not edge
	if !auth.IsEdgeRole(authContext) {
		model.MaskContainerRegistries(containerRegistries)
	}
	return containerRegistries, err
}

// internal API used by getContainerRegistriesWV2
func (dbAPI *dbObjectModelAPI) getContainerRegistriesByProjectsForQuery(context context.Context, projectIDs []string, containerRegistryID string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.ContainerRegistry, int, error) {
	containerRegistries := []model.ContainerRegistry{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return containerRegistries, 0, err
	}
	tenantID := authContext.TenantID
	containerRegistryDBOs := []ContainerRegistryDBO{}

	var query string
	if len(projectIDs) == 0 {
		query, err = buildLimitQuery(entityTypeContainerRegistry, queryMap["SelectContainerRegistriesTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return containerRegistries, 0, err
		}
		err = dbAPI.Query(context, &containerRegistryDBOs, query, tenantIDParam5{TenantID: tenantID, ID: containerRegistryID})
	} else {
		query, err = buildLimitQuery(entityTypeContainerRegistry, queryMap["SelectContainerRegistriesByProjectsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return containerRegistries, 0, err
		}
		err = dbAPI.QueryIn(context, &containerRegistryDBOs, query, tenantIDParam5{TenantID: tenantID, ID: containerRegistryID, ProjectIDs: projectIDs})
	}
	if err != nil {
		return containerRegistries, 0, err
	}
	if len(containerRegistryDBOs) == 0 {
		return containerRegistries, 0, nil
	}
	totalCount := 0
	first := true
	for _, containerRegistryDBO := range containerRegistryDBOs {
		containerRegistry := model.ContainerRegistry{}
		if first {
			first = false
			if containerRegistryDBO.TotalCount != nil {
				totalCount = *containerRegistryDBO.TotalCount
			}
		}
		err := base.Convert(&containerRegistryDBO, &containerRegistry)
		if err != nil {
			return []model.ContainerRegistry{}, 0, err
		}
		err = decryptContainerRegistry(context, dbAPI, &containerRegistry)
		if err != nil {
			return []model.ContainerRegistry{}, 0, err
		}
		containerRegistries = append(containerRegistries, containerRegistry)
	}
	// mask out credentials if not edge
	if !auth.IsEdgeRole(authContext) {
		model.MaskContainerRegistries(containerRegistries)
	}
	return containerRegistries, totalCount, nil
}

func (dbAPI *dbObjectModelAPI) getContainerRegistriesW(context context.Context, projectID string, containerRegistryID string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return errcode.NewPermissionDeniedError("RBAC")
		}
	}
	entitiesQueryParam := model.GetEntitiesQueryParamV1(req)
	containerRegistries, err := dbAPI.getContainerRegistries(context, projectID, containerRegistryID, base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
	if err != nil {
		return err
	}
	if len(containerRegistryID) == 0 {
		return base.DispatchPayload(w, containerRegistries)
	}
	if len(containerRegistries) == 0 {
		return errcode.NewRecordNotFoundError(containerRegistryID)
	}
	return json.NewEncoder(w).Encode(containerRegistries[0])
}

func (dbAPI *dbObjectModelAPI) getContainerRegistriesWV2(context context.Context, projectID string, containerRegistryID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getContainerRegistryDBQueryParam(context, projectID, containerRegistryID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		if containerRegistryID == "" {
			return json.NewEncoder(w).Encode(model.ContainerRegistryListPayload{ContainerRegistryListV2: []model.ContainerRegistryV2{}})
		} else {
			return errcode.NewRecordNotFoundError(containerRegistryID)
		}
	}
	projectIDs := []string{}
	if dbQueryParam.InQuery {
		projectIDs = dbQueryParam.Param.(ContainerRegistryProjects).ProjectIDs
	}

	queryParam := model.GetEntitiesQueryParam(req)

	containerRegistries, totalCount, err := dbAPI.getContainerRegistriesByProjectsForQuery(context, projectIDs, containerRegistryID, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeContainerRegistry}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	if len(containerRegistryID) == 0 {
		r := model.ContainerRegistryListPayload{
			EntityListResponsePayload: entityListResponsePayload,
			ContainerRegistryListV2:   model.ContainerRegistriesByID(containerRegistries).ToV2(),
		}
		return json.NewEncoder(w).Encode(r)
	}
	if len(containerRegistries) == 0 {
		return errcode.NewRecordNotFoundError(containerRegistryID)
	}
	return json.NewEncoder(w).Encode(containerRegistries[0].ToV2())
}

// SelectAllContainerRegistries select all ContainerRegistries for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllContainerRegistries(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ContainerRegistry, error) {
	return dbAPI.getContainerRegistries(context, "", "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllContainerRegistriesW select all ContainerRegistries for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllContainerRegistriesW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getContainerRegistriesW(context, "", "", w, req)
}

// SelectAllContainerRegistriesWV2 select all ContainerRegistries for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllContainerRegistriesWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getContainerRegistriesWV2(context, "", "", w, req)
}

// SelectAllContainerRegistriesForProject select all ContainerRegistries for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllContainerRegistriesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ContainerRegistry, error) {
	return dbAPI.getContainerRegistries(context, projectID, "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllContainerRegistriesForProjectW select all ContainerRegistries for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllContainerRegistriesForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getContainerRegistriesW(context, projectID, "", w, req)
}

// SelectAllContainerRegistriesForProjectWV2 select all ContainerRegistries for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllContainerRegistriesForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getContainerRegistriesWV2(context, projectID, "", w, req)
}

// GetContainerRegistry get a ContainerRegistry object in the DB
func (dbAPI *dbObjectModelAPI) GetContainerRegistry(context context.Context, id string) (model.ContainerRegistry, error) {
	res := model.ContainerRegistry{}
	if len(id) == 0 {
		return res, errcode.NewRecordNotFoundError(id)
	}
	containerRegistries, err := dbAPI.getContainerRegistries(context, "", id, base.StartPageToken, base.MaxRowsLimit, nil)
	if err != nil {
		return res, err
	}
	if len(containerRegistries) == 1 {
		return containerRegistries[0], nil
	}
	return res, errcode.NewRecordNotFoundError(id)
}

// GetContainerRegistryW get a ContainerRegistry object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetContainerRegistryW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("profileID")
	}
	return dbAPI.getContainerRegistriesW(context, "", id, w, req)
}

// GetContainerRegistryWV2 get a ContainerRegistry object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetContainerRegistryWV2(context context.Context, containerRegistryID string, w io.Writer, req *http.Request) error {
	if len(containerRegistryID) == 0 {
		return errcode.NewBadRequestError("id")
	}
	return dbAPI.getContainerRegistriesWV2(context, "", containerRegistryID, w, req)
}

// CreateContainerRegistry creates a ContainerRegistry object in the DB
func (dbAPI *dbObjectModelAPI) CreateContainerRegistry(context context.Context, i interface{} /* *model.ContainerRegistry */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ContainerRegistry)
	if !ok {
		return resp, errcode.NewInternalError("CreateContainerRegistry: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateContainerRegistry doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateContainerRegistry doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityContainerRegistry,
		meta.OperationCreate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	doc.IFlagEncrypted = true
	err = model.ValidateContainerRegistry(doc)
	if err != nil {
		return resp, err
	}
	err = dbAPI.containerRegistryFromCLoudCreds(context, &doc, authContext)
	if err != nil {
		return resp, err
	}
	containerRegistryDBO := ContainerRegistryDBO{}

	err = base.Convert(&doc, &containerRegistryDBO)
	if err != nil {
		return resp, err
	}
	if len(doc.CloudCredsID) == 0 {
		containerRegistryDBO.CloudCredsID = nil
	}
	err = encryptContainerRegistryDBO(context, dbAPI, &containerRegistryDBO)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", tenantID))
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateContainerRegistry"], &containerRegistryDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating container registry for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(containerRegistryDBO.ID, err)
	}

	// no notification in create, since the container registry will not be in any project and thus will not be applicable to any edge
	// if callback != nil {
	// 	go callback(context, doc)
	// }
	resp.ID = doc.ID
	GetAuditlogHandler().addContainerRegistryAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateContainerRegistryV2 creates a container registry object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateContainerRegistryV2(context context.Context, i interface{} /* *model.ContainerRegistryV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.ContainerRegistryV2)
	if !ok {
		return model.CreateDocumentResponse{}, errcode.NewInternalError("CreateContainerRegistryV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.CreateContainerRegistry(context, &doc, callback)
}

// CreateContainerRegistryW creates an container registry profile object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateContainerRegistryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateContainerRegistry, &model.ContainerRegistry{}, w, r, callback)
}

// CreateContainerRegistryWV2 creates an container registry profile object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateContainerRegistryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateContainerRegistryV2), &model.ContainerRegistryV2{}, w, r, callback)
}

// UpdateContainerRegistry update a container registry in the DB
func (dbAPI *dbObjectModelAPI) UpdateContainerRegistry(context context.Context, i interface{} /* *model.ContainerRegistry */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ContainerRegistry)
	if !ok {
		return resp, errcode.NewInternalError("UpdateContainerRegistry: type error")
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
	err = auth.CheckRBAC(
		authContext,
		meta.EntityContainerRegistry,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	doc.IFlagEncrypted = true
	err = model.ValidateContainerRegistry(doc)
	if err != nil {
		return resp, err
	}
	err = dbAPI.containerRegistryFromCLoudCreds(context, &doc, authContext)
	if err != nil {
		return resp, err
	}

	containerRegistryDBO := ContainerRegistryDBO{}
	err = base.Convert(&doc, &containerRegistryDBO)
	if err != nil {
		return resp, err
	}
	if len(doc.CloudCredsID) == 0 {
		containerRegistryDBO.CloudCredsID = nil
	}

	err = encryptContainerRegistryDBO(context, dbAPI, &containerRegistryDBO)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", tenantID))
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateContainerRegistry"], &containerRegistryDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating container registry for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(containerRegistryDBO.ID, err)
	}
	if callback != nil {
		edgeIDs, err := dbAPI.GetAllDockerProfileEdges(context, doc.ID)
		if err == nil && len(edgeIDs) != 0 {
			x := model.ScopedEntity{
				Doc:     doc,
				EdgeIDs: edgeIDs,
			}
			go callback(context, x)
		}
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addContainerRegistryAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateContainerRegistryV2 update an application object in the DB
func (dbAPI *dbObjectModelAPI) UpdateContainerRegistryV2(context context.Context, i interface{} /* *model.ContainerRegistryV2 */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.ContainerRegistryV2)
	if !ok {
		return model.UpdateDocumentResponse{}, errcode.NewInternalError("UpdateContainerRegistryV2: type error")
	}
	doc := p.FromV2()
	return dbAPI.UpdateContainerRegistry(context, &doc, callback)
}

// UpdateContainerRegistryW update a container registry in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateContainerRegistryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateContainerRegistry, &model.ContainerRegistry{}, w, r, callback)
}

// UpdateContainerRegistryWV2 update a container registry in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateContainerRegistryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateContainerRegistryV2), &model.ContainerRegistryV2{}, w, r, callback)
}

// DeleteContainerRegistry delete a container registry object in the DB
func (dbAPI *dbObjectModelAPI) DeleteContainerRegistry(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityContainerRegistry,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	containerReg, errGetContReg := dbAPI.GetContainerRegistry(context, id)
	doc := model.ContainerRegistry{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}
	var x interface{}
	x = doc
	if callback != nil {
		edgeIDs, err := dbAPI.GetAllDockerProfileEdges(context, doc.ID)
		if err == nil && len(edgeIDs) != 0 {
			x = model.ScopedEntity{
				Doc:     doc,
				EdgeIDs: edgeIDs,
			}
		} else {
			callback = nil
		}
	}
	result, err := DeleteEntity(context, dbAPI, "docker_profile_model", "id", id, x, callback)
	if err == nil {
		if errGetContReg != nil {
			glog.Error("Error in getting container registry : ", errGetContReg.Error())
		} else {
			GetAuditlogHandler().addContainerRegistryAuditLog(dbAPI, context, containerReg, DELETE)
		}
	}
	return result, err
}

// DeleteContainerRegistryW delete a container registry object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteContainerRegistryW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteContainerRegistry, id, w, callback)
}

// DeleteContainerRegistryWV2 delete a container registry object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteContainerRegistryWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteContainerRegistry), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) EncryptAllContainerRegistries(ctx context.Context) error {
	tenants, err := dbAPI.SelectAllTenants(ctx)
	if err != nil {
		return err
	}
	for _, tenant := range tenants {
		authContext := &base.AuthContext{
			TenantID: tenant.ID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newContext := context.WithValue(ctx, base.AuthContextKey, authContext)
		dps, err := dbAPI.SelectAllContainerRegistries(newContext, nil)
		if err != nil {
			return err
		}
		for _, dp := range dps {
			if !dp.IFlagEncrypted {
				_, err := dbAPI.UpdateContainerRegistry(newContext, &dp, nil)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (dbAPI *dbObjectModelAPI) EncryptAllContainerRegistriesW(context context.Context, r io.Reader) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	s := buf.String()
	if s != "sherlockkcolrehs" {
		return errcode.NewBadRequestError("payload")
	}
	return dbAPI.EncryptAllContainerRegistries(context)
}

type ContainerRegistryIDsParam struct {
	TenantID             string   `json:"tenantId" db:"tenant_id"`
	ContainerRegistryIDs []string `json:"containerRegistryIds" db:"docker_profile_ids"`
}

func (dbAPI *dbObjectModelAPI) SelectContainerRegistriesByIDs(context context.Context, containerRegistryIDs []string) ([]model.ContainerRegistry, error) {
	containerRegistries := []model.ContainerRegistry{}
	if len(containerRegistryIDs) == 0 {
		return containerRegistries, nil
	}
	containerRegistryDBOs := []ContainerRegistryDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return containerRegistries, err
	}
	param := ContainerRegistryIDsParam{TenantID: authContext.TenantID, ContainerRegistryIDs: containerRegistryIDs}
	err = dbAPI.QueryIn(context, &containerRegistryDBOs, queryMap["SelectContainerRegistriesByIDs"], param)
	if err != nil {
		return containerRegistries, errcode.TranslateDatabaseError("<ids>", err)
	}
	for _, containerRegistryDBO := range containerRegistryDBOs {
		containerRegistry := model.ContainerRegistry{}
		err := base.Convert(&containerRegistryDBO, &containerRegistry)
		if err != nil {
			return []model.ContainerRegistry{}, err
		}
		err = decryptContainerRegistry(context, dbAPI, &containerRegistry)
		if err != nil {
			return []model.ContainerRegistry{}, err
		}
		containerRegistries = append(containerRegistries, containerRegistry)
	}
	return containerRegistries, nil
}

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

func init() {
	queryMap["SelectDockerProfiles"] = `SELECT * FROM docker_profile_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id)`
	queryMap["SelectDockerProfilesByProjects"] = `SELECT * FROM docker_profile_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (id IN (SELECT docker_profile_id FROM project_docker_profile_model WHERE project_id IN (:project_ids)))`
	queryMap["CreateDockerProfile"] = `INSERT INTO docker_profile_model (id, version, tenant_id, name, description, credentials, type, server, user_name, email , pwd, cloud_creds_id, iflag_encrypted, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :description, :credentials, :type, :server, :user_name, :email , :pwd, :cloud_creds_id, :iflag_encrypted, :created_at, :updated_at)`
	queryMap["UpdateDockerProfile"] = `UPDATE docker_profile_model SET version = :version, name = :name, description = :description, credentials = :credentials, type = :type, server = :server, user_name = :user_name, email = :email ,pwd = :pwd, cloud_creds_id = :cloud_creds_id, iflag_encrypted = :iflag_encrypted, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectDockerProfileProjects"] = `SELECT * FROM project_docker_profile_model WHERE docker_profile_id = :docker_profile_id`
	queryMap["SelectDockerProfilesByIDs"] = `SELECT * FROM docker_profile_model WHERE tenant_id = :tenant_id AND (id IN (:docker_profile_ids))`
}

// DockerProfileDBO is DB object model for DockerProfiles
type DockerProfileDBO struct {
	model.BaseModelDBO
	Name           string          `json:"name" db:"name"`
	Description    string          `json:"description" db:"description"`
	Type           string          `json:"type" db:"type"`
	Server         string          `json:"server" db:"server"`
	UserName       string          `json:"userName" db:"user_name"`
	Email          string          `json:"email" db:"email"`
	Pwd            string          `json:"pwd" db:"pwd"`
	CloudCredsID   *string         `json:"cloudCredsID" db:"cloud_creds_id"`
	Credentials    *types.JSONText `json:"credentials" db:"credentials"`
	IFlagEncrypted *bool           `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
}

type DockerProfileProjects struct {
	DockerProfileDBO
	ProjectIDs base.StringArray `json:"project_ids" db:"project_ids"`
}

func (dp *DockerProfileDBO) MaskObject() {
	dp.Pwd = base.MaskString(dp.Pwd, "*", 0, 4)
}
func MaskDockerProfiles(dps []DockerProfileDBO) {
	for i := 0; i < len(dps); i++ {
		dps[i].MaskObject()
	}
}

func getDockerProfileQueryParam(context context.Context, id string) base.InQueryParam {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := DockerProfileDBO{BaseModelDBO: tenantModel}
	if !auth.IsInfraAdminRole(authContext) {
		projectIDs := auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return base.InQueryParam{}
		}
		return base.InQueryParam{
			Param: DockerProfileProjects{
				DockerProfileDBO: param,
				ProjectIDs:       projectIDs,
			},
			Key:     "SelectDockerProfilesByProjects",
			InQuery: true,
		}
	}
	return base.InQueryParam{
		Param:   param,
		Key:     "SelectDockerProfiles",
		InQuery: false,
	}
}

func encryptDockerProfile(context context.Context, dbAPI *dbObjectModelAPI, doc *model.DockerProfile) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	return encryptDockerProfileT(context, dbAPI, doc, &tenant)
}

func encryptDockerProfileT(context context.Context, dbAPI *dbObjectModelAPI, doc *model.DockerProfile, tenant *model.Tenant) error {
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

func encryptDockerProfileDBO(context context.Context, dbAPI *dbObjectModelAPI, doc *DockerProfileDBO) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	return encryptDockerProfileDBOT(context, dbAPI, doc, &tenant)
}

func encryptDockerProfileDBOT(context context.Context, dbAPI *dbObjectModelAPI, doc *DockerProfileDBO, tenant *model.Tenant) error {
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

func encryptDockerProfileDBOs(context context.Context, dbAPI *dbObjectModelAPI, docs []DockerProfileDBO) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	for i := range docs {
		err := encryptDockerProfileDBOT(context, dbAPI, &docs[i], &tenant)
		if err != nil {
			return err
		}
	}
	return nil
}

func decryptDockerProfile(context context.Context, dbAPI *dbObjectModelAPI, doc *model.DockerProfile) error {
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
	return decryptDockerProfileT(context, dbAPI, doc, token)
}

func decryptDockerProfileT(context context.Context, dbAPI *dbObjectModelAPI, doc *model.DockerProfile, token *crypto.Token) error {
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

func decryptDockerProfileDBO(context context.Context, dbAPI *dbObjectModelAPI, doc *DockerProfileDBO) error {
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
	return decryptDockerProfileDBOT(context, dbAPI, doc, token)
}

func decryptDockerProfileDBOT(context context.Context, dbAPI *dbObjectModelAPI, doc *DockerProfileDBO, token *crypto.Token) error {
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

func decryptDockerProfileDBOs(context context.Context, dbAPI *dbObjectModelAPI, docs []DockerProfileDBO) error {
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
		err := decryptDockerProfileDBOT(context, dbAPI, &docs[i], token)
		if err != nil {
			return err
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) dockerProfileFromCloudCreds(context context.Context, doc *model.DockerProfile, authContext *base.AuthContext) error {

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

		// Import docker profile from CloudCreds, we need to do this to support backward compatibility for edges which do not retrive the credentials
		cloudCreds, err := dbAPI.GetCloudCreds(context, doc.CloudCredsID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error retriving Cloud Creds with id. Error: %s"), err.Error())
			return err
		}
		doc.Type = cloudCreds.Type
		if doc.Type == "AWS" {
			//get region and account from server provided

			server := strings.Split(doc.Server, ".")
			if len(server) != 6 {
				err = errcode.NewBadRequestExError("Server", "Invalid server format")
				glog.Errorf(base.PrefixRequestID(context, "Error getting account and region from server. Error: %s"), err.Error())
				return err
			}
			credMap := make(map[string]string)
			credMap["AccessKey"] = cloudCreds.AWSCredential.AccessKey
			credMap["Secret"] = cloudCreds.AWSCredential.Secret
			credMap["Account"] = server[0]
			credMap["Region"] = server[3]

			creds, err := json.Marshal(credMap)
			doc.Credentials = string(creds)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(context, "Error Marshalling Cloud Creds. Error: %s"), err.Error())
				return err
			}
			doc.UserName = "AWS"
		} else if doc.Type == "GCP" {
			doc.UserName = "_json_key"
			creds, err := json.Marshal(cloudCreds.GCPCredential)
			if err != nil {
				fmt.Println(err)
				return err
			}
			doc.Pwd = string(creds)
		} else {
			err := errcode.NewBadRequestError("Type")
			glog.Errorf(base.PrefixRequestID(context, "Error, only AWS and GCP are supported types to import cloud creds from. Error: %s"), err.Error())
			return err
		}
	} else {
		if doc.Type == "AWS" || doc.Type == "GCP" {
			err := errcode.NewBadRequestError("CloudCredsID")
			glog.Errorf(base.PrefixRequestID(context, "Error, cloudCreds not provided but type is %s. Error: %s"), doc.Type, err.Error())
			return err
		}
	}
	return nil
}

// SelectAllDockerProfiles select all DockerProfiles for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllDockerProfiles(context context.Context) ([]model.DockerProfile, error) {
	dockerProfiles := []model.DockerProfile{}
	queryParam := getDockerProfileQueryParam(context, "")
	if queryParam.Key == "" {
		return dockerProfiles, nil
	}
	var err error
	if queryParam.InQuery {
		dockerProfileDBOs := []DockerProfileDBO{}
		err = dbAPI.QueryIn(context, &dockerProfileDBOs, queryMap[queryParam.Key], queryParam.Param)
		if err == nil {
			for _, dockerProfileDBO := range dockerProfileDBOs {
				dockerProfile := model.DockerProfile{}
				err := base.Convert(&dockerProfileDBO, &dockerProfile)
				if err != nil {
					return []model.DockerProfile{}, err
				}
				err = decryptDockerProfile(context, dbAPI, &dockerProfile)
				if err != nil {
					return []model.DockerProfile{}, err
				}
				dockerProfiles = append(dockerProfiles, dockerProfile)
			}
		}
	} else {
		_, err = dbAPI.PagedQuery(context, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
			dockerProfile := model.DockerProfile{}
			err := base.Convert(dbObjPtr, &dockerProfile)
			if err != nil {
				return err
			}
			err = decryptDockerProfile(context, dbAPI, &dockerProfile)
			if err != nil {
				return err
			}
			dockerProfiles = append(dockerProfiles, dockerProfile)
			return nil
		}, queryMap[queryParam.Key], queryParam.Param)
	}
	return dockerProfiles, err
}

// SelectAllDockerProfilesW select all docker Profiles for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDockerProfilesW(context context.Context, w io.Writer, req *http.Request) error {
	dockerProfileDBOs := []DockerProfileDBO{}
	queryParam := getDockerProfileQueryParam(context, "")
	if queryParam.Key != "" {
		err := dbAPI.QueryInMaybe(context, &dockerProfileDBOs, queryMap[queryParam.Key], queryParam)
		if err != nil {
			return err
		}
		err = decryptDockerProfileDBOs(context, dbAPI, dockerProfileDBOs)
		if err != nil {
			return err
		}
		// if handled, err := handleEtag(w, etag, dockerProfileDBOs); handled {
		// 	return err
		// }
	}
	authContext, err := base.GetAuthContext(context)
	if err == nil {
		// mask out credentials if not edge
		if !auth.IsEdgeRole(authContext) {
			MaskDockerProfiles(dockerProfileDBOs)
		}
	}
	return base.DispatchPayload(w, dockerProfileDBOs)
}

// SelectAllDockerProfilesWV2 select all docker Profiles for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDockerProfilesWV2(context context.Context, w io.Writer, req *http.Request) error {
	dockerProfileDBOs := []DockerProfileDBO{}
	queryParam := getDockerProfileQueryParam(context, "")
	if queryParam.Key != "" {
		err := dbAPI.QueryInMaybe(context, &dockerProfileDBOs, queryMap[queryParam.Key], queryParam)
		if err != nil {
			return err
		}
		err = decryptDockerProfileDBOs(context, dbAPI, dockerProfileDBOs)
		if err != nil {
			return err
		}
		// if handled, err := handleEtag(w, etag, dockerProfileDBOs); handled {
		// 	return err
		// }
	}
	authContext, err := base.GetAuthContext(context)
	if err == nil {
		// mask out credentials if not edge
		if !auth.IsEdgeRole(authContext) {
			MaskDockerProfiles(dockerProfileDBOs)
		}
	}
	dockerProfiles := []model.DockerProfile{}
	for _, dockerProfileDBO := range dockerProfileDBOs {
		dockerProfile := model.DockerProfile{}
		err := base.Convert(&dockerProfileDBO, &dockerProfile)
		if err != nil {
			return err
		}
		dockerProfiles = append(dockerProfiles, dockerProfile)
	}
	r := model.DockerProfileListPayload{
		DockerProfileList: dockerProfiles,
	}
	return json.NewEncoder(w).Encode(r)
}

func (dbAPI *dbObjectModelAPI) selectAllDockerProfilesForProject(context context.Context, projectID string) ([]DockerProfileDBO, error) {
	dockerProfileDBOs := []DockerProfileDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dockerProfileDBOs, err
	}
	if !auth.IsProjectMember(projectID, authContext) {
		return dockerProfileDBOs, errcode.NewPermissionDeniedError("RBAC")
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	param := DockerProfileProjects{
		DockerProfileDBO: DockerProfileDBO{BaseModelDBO: tenantModel},
		ProjectIDs:       []string{projectID},
	}
	err = dbAPI.QueryIn(context, &dockerProfileDBOs, queryMap["SelectDockerProfilesByProjects"], param)
	if err == nil {
		err = decryptDockerProfileDBOs(context, dbAPI, dockerProfileDBOs)
	}
	return dockerProfileDBOs, err
}

// SelectAllDockerProfilesForProject select all DockerProfiles for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllDockerProfilesForProject(context context.Context, projectID string) ([]model.DockerProfile, error) {
	dockerProfiles := []model.DockerProfile{}
	dockerProfileDBOs, err := dbAPI.selectAllDockerProfilesForProject(context, projectID)
	if err != nil {
		return dockerProfiles, err
	}
	for _, dockerProfileDBO := range dockerProfileDBOs {
		dockerProfile := model.DockerProfile{}
		err := base.Convert(&dockerProfileDBO, &dockerProfile)
		if err != nil {
			return []model.DockerProfile{}, err
		}
		dockerProfiles = append(dockerProfiles, dockerProfile)
	}
	return dockerProfiles, err
}

// SelectAllDockerProfilesForProjectW select all docker Profiles for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDockerProfilesForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	dockerProfileDBOs, err := dbAPI.selectAllDockerProfilesForProject(context, projectID)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, dockerProfileDBOs); handled {
	// 	return err
	// }
	authContext, err := base.GetAuthContext(context)
	if err == nil {
		// mask out credentials if not edge
		if !auth.IsEdgeRole(authContext) {
			MaskDockerProfiles(dockerProfileDBOs)
		}
	}
	return base.DispatchPayload(w, dockerProfileDBOs)
}

// SelectAllDockerProfilesForProjectWV2 select all docker Profiles for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllDockerProfilesForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	dockerProfileDBOs, err := dbAPI.selectAllDockerProfilesForProject(context, projectID)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, dockerProfileDBOs); handled {
	// 	return err
	// }
	authContext, err := base.GetAuthContext(context)
	if err == nil {
		// mask out credentials if not edge
		if !auth.IsEdgeRole(authContext) {
			MaskDockerProfiles(dockerProfileDBOs)
		}
	}
	dockerProfiles := []model.DockerProfile{}
	for _, dockerProfileDBO := range dockerProfileDBOs {
		dockerProfile := model.DockerProfile{}
		err := base.Convert(&dockerProfileDBO, &dockerProfile)
		if err != nil {
			return err
		}
		dockerProfiles = append(dockerProfiles, dockerProfile)
	}
	r := model.DockerProfileListPayload{
		DockerProfileList: dockerProfiles,
	}
	return json.NewEncoder(w).Encode(r)
}

// GetDockerProfile get a DockerProfile object in the DB
func (dbAPI *dbObjectModelAPI) GetDockerProfile(context context.Context, id string) (model.DockerProfile, error) {
	dockerProfile := model.DockerProfile{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dockerProfile, err
	}
	dockerProfileDBOs := []DockerProfileDBO{}
	queryParam := getDockerProfileQueryParam(context, id)
	if len(id) == 0 {
		return dockerProfile, errcode.NewBadRequestError("profileID")
	}
	if queryParam.Key != "" {
		err := dbAPI.QueryInMaybe(context, &dockerProfileDBOs, queryMap[queryParam.Key], queryParam)
		if err != nil {
			return dockerProfile, err
		}
	}
	if len(dockerProfileDBOs) == 0 {
		return dockerProfile, errcode.NewRecordNotFoundError(id)
	}
	dockerProfileDBO := dockerProfileDBOs[0]
	err = decryptDockerProfileDBO(context, dbAPI, &dockerProfileDBO)
	if err != nil {
		return dockerProfile, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", authContext.TenantID))
	}
	err = base.Convert(&dockerProfileDBO, &dockerProfile)
	return dockerProfile, err
}

// GetDockerProfileW get a docker profile object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetDockerProfileW(context context.Context, id string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	dockerProfileDBOs := []DockerProfileDBO{}
	queryParam := getDockerProfileQueryParam(context, id)
	if len(id) == 0 {
		return errcode.NewBadRequestError("profileID")
	}
	if queryParam.Key != "" {
		err := dbAPI.QueryInMaybe(context, &dockerProfileDBOs, queryMap[queryParam.Key], queryParam)
		if err != nil {
			return err
		}
	}
	if len(dockerProfileDBOs) == 0 {
		return errcode.NewRecordNotFoundError(id)
	}
	dockerProfileDBO := dockerProfileDBOs[0]
	err = decryptDockerProfileDBO(context, dbAPI, &dockerProfileDBO)
	if err != nil {
		return errcode.NewInternalError(fmt.Sprintf("tenantID:%s", authContext.TenantID))
	}
	// if handled, err := handleEtag(w, etag, dockerProfileDBO); handled {
	// 	return err
	// }
	// mask out credentials if not edge
	if !auth.IsEdgeRole(authContext) {
		dockerProfileDBO.MaskObject()
	}
	return base.DispatchPayload(w, dockerProfileDBO)
}

// CreateDockerProfile creates a docker profile object in the DB
func (dbAPI *dbObjectModelAPI) CreateDockerProfile(context context.Context, i interface{} /* *model.DockerProfile */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DockerProfile)
	if !ok {
		return resp, errcode.NewInternalError("CreateDockerProfile: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateDockerProfile doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateDockerProfile doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDockerProfile,
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
	err = model.ValidateDockerProfile(doc)
	if err != nil {
		return resp, err
	}
	err = dbAPI.dockerProfileFromCloudCreds(context, &doc, authContext)
	if err != nil {
		return resp, err
	}
	dockerProfileDBO := DockerProfileDBO{}

	err = base.Convert(&doc, &dockerProfileDBO)
	if err != nil {
		return resp, err
	}
	if len(doc.CloudCredsID) == 0 {
		dockerProfileDBO.CloudCredsID = nil
	}
	err = encryptDockerProfileDBO(context, dbAPI, &dockerProfileDBO)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", tenantID))
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateDockerProfile"], &dockerProfileDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating docker profile for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(dockerProfileDBO.ID, err)
	}
	// no notification in create, since the docker profile will not be in any project and thus will not be applicable to any edge
	// if callback != nil {
	// 	go callback(context, doc)
	// }
	resp.ID = doc.ID
	return resp, nil
}

// CreateDockerProfileW creates an docker profile object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateDockerProfileW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateDockerProfile, &model.DockerProfile{}, w, r, callback)
}

// CreateDockerProfileWV2 creates an docker profile object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateDockerProfileWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateDockerProfile), &model.DockerProfile{}, w, r, callback)
}

// UpdateDockerProfile update a docker profile in the DB
func (dbAPI *dbObjectModelAPI) UpdateDockerProfile(context context.Context, i interface{} /* *model.DockerProfile */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.DockerProfile)
	if !ok {
		return resp, errcode.NewInternalError("UpdateDockerProfile: type error")
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
		meta.EntityDockerProfile,
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
	err = model.ValidateDockerProfile(doc)
	if err != nil {
		return resp, err
	}
	err = dbAPI.dockerProfileFromCloudCreds(context, &doc, authContext)
	if err != nil {
		return resp, err
	}

	dockerProfileDBO := DockerProfileDBO{}
	err = base.Convert(&doc, &dockerProfileDBO)
	if err != nil {
		return resp, err
	}
	if len(doc.CloudCredsID) == 0 {
		dockerProfileDBO.CloudCredsID = nil
	}

	err = encryptDockerProfileDBO(context, dbAPI, &dockerProfileDBO)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", tenantID))
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateDockerProfile"], &dockerProfileDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating docker profile for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(dockerProfileDBO.ID, err)
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
	return resp, nil
}

// UpdateDockerProfileW update a docker profile in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateDockerProfileW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateDockerProfile, &model.DockerProfile{}, w, r, callback)
}

// UpdateDockerProfileWV2 update a docker profile in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateDockerProfileWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateDockerProfile), &model.DockerProfile{}, w, r, callback)
}

// DeleteDockerProfile delete a docker profile object in the DB
func (dbAPI *dbObjectModelAPI) DeleteDockerProfile(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityDockerProfile,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	doc := model.DockerProfile{
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
	return DeleteEntity(context, dbAPI, "docker_profile_model", "id", id, x, callback)
}

// DeleteDockerProfileW delete a docker profile object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteDockerProfileW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteDockerProfile, id, w, callback)
}

// DeleteDockerProfileWV2 delete a docker profile object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteDockerProfileWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteDockerProfile), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) EncryptAllDockerProfiles(ctx context.Context) error {
	tenants, err := dbAPI.SelectAllTenants(nil)
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
		dps, err := dbAPI.SelectAllDockerProfiles(newContext)
		if err != nil {
			return err
		}
		for _, dp := range dps {
			if !dp.IFlagEncrypted {
				_, err := dbAPI.UpdateDockerProfile(newContext, &dp, nil)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (dbAPI *dbObjectModelAPI) EncryptAllDockerProfilesW(context context.Context, r io.Reader) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	s := buf.String()
	if s != "sherlockkcolrehs" {
		return errcode.NewBadRequestError("payload")
	}
	return dbAPI.EncryptAllDockerProfiles(context)
}

func (dbAPI *dbObjectModelAPI) GetAllDockerProfileProjects(context context.Context, dockerProfileID string) ([]string, error) {
	projectIDs := []string{}
	projectDockerProfileDBOs := []ProjectDockerProfileDBO{}
	err := dbAPI.Query(context, &projectDockerProfileDBOs, queryMap["SelectDockerProfileProjects"], ProjectDockerProfileDBO{DockerProfileID: dockerProfileID})
	if err != nil {
		return projectIDs, err
	}
	for _, projectDockerProfileDBO := range projectDockerProfileDBOs {
		projectIDs = append(projectIDs,
			projectDockerProfileDBO.ProjectID)
	}
	return projectIDs, nil
}
func (dbAPI *dbObjectModelAPI) GetAllDockerProfileEdges(context context.Context, dockerProfileID string) ([]string, error) {
	projectIDs, err := dbAPI.GetAllDockerProfileProjects(context, dockerProfileID)
	if err != nil {
		return []string{}, err
	}
	return dbAPI.GetProjectsEdges(context, projectIDs)
}

type DockerProfileIDsParam struct {
	TenantID         string   `json:"tenantId" db:"tenant_id"`
	DockerProfileIDs []string `json:"dockerProfileIds" db:"docker_profile_ids"`
}

func (dbAPI *dbObjectModelAPI) SelectDockerProfilesByIDs(context context.Context, dockerProfileIDs []string) ([]model.DockerProfile, error) {
	dockerProfiles := []model.DockerProfile{}
	dockerProfileDBOs := []DockerProfileDBO{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return dockerProfiles, err
	}
	param := DockerProfileIDsParam{TenantID: authContext.TenantID, DockerProfileIDs: dockerProfileIDs}
	err = dbAPI.QueryIn(context, &dockerProfileDBOs, queryMap["SelectDockerProfilesByIDs"], param)
	if err != nil {
		return dockerProfiles, errcode.TranslateDatabaseError("<ids>", err)
	}
	for _, dockerProfileDBO := range dockerProfileDBOs {
		dockerProfile := model.DockerProfile{}
		err := base.Convert(&dockerProfileDBO, &dockerProfile)
		if err != nil {
			return []model.DockerProfile{}, err
		}
		err = decryptDockerProfile(context, dbAPI, &dockerProfile)
		if err != nil {
			return []model.DockerProfile{}, err
		}
		dockerProfiles = append(dockerProfiles, dockerProfile)
	}
	return dockerProfiles, nil
}

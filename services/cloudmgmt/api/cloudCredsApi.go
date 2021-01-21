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
	"reflect"
	"strings"

	"github.com/go-openapi/errors"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
)

const entityTypeCloudCreds = "cloudcreds"

func init() {
	queryMap["SelectCloudCredsTemplate1"] = `SELECT * FROM cloud_creds_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) %s`
	queryMap["SelectCloudCredsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM cloud_creds_model WHERE tenant_id = :tenant_id %s`
	queryMap["SelectCloudCredsByProjectsTemplate"] = `SELECT *, count(*) OVER() as total_count FROM cloud_creds_model WHERE tenant_id = :tenant_id AND (id IN (SELECT cloud_creds_id FROM project_cloud_creds_model WHERE project_id IN (:project_ids))) %s`
	queryMap["SelectCloudCredsByProjectsTemplate1"] = `SELECT * FROM cloud_creds_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (id IN (SELECT cloud_creds_id FROM project_cloud_creds_model WHERE project_id IN (:project_ids))) %s`
	queryMap["CreateCloudCreds"] = `INSERT INTO cloud_creds_model (id, version, tenant_id, name, type, description, aws_credential, gcp_credential, az_credential, iflag_encrypted, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :type, :description, :aws_credential, :gcp_credential, :az_credential, :iflag_encrypted, :created_at, :updated_at)`
	queryMap["UpdateCloudCreds"] = `UPDATE cloud_creds_model SET version = :version, name = :name, type = :type, description = :description, aws_credential = :aws_credential, gcp_credential = :gcp_credential, az_credential = :az_credential, iflag_encrypted = :iflag_encrypted, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectCloudCredsProjects"] = `SELECT * FROM project_cloud_creds_model WHERE cloud_creds_id = :cloud_creds_id`

	orderByHelper.Setup(entityTypeCloudCreds, []string{"id", "version", "created_at", "updated_at", "name", "description", "type"})
}

// CloudCredsDBO is the DB object for cloud creds
type CloudCredsDBO struct {
	model.BaseModelDBO
	Name           string          `json:"name" db:"name"`
	Type           string          `json:"type" db:"type"`
	Description    string          `json:"description" db:"description"`
	AWSCredential  *types.JSONText `json:"awsCredential,omitempty" db:"aws_credential"`
	GCPCredential  *types.JSONText `json:"gcpCredential,omitempty" db:"gcp_credential"`
	AZCredential   *types.JSONText `json:"azCredential,omitempty" db:"az_credential"`
	IFlagEncrypted *bool           `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
}
type CloudCredsProjects struct {
	CloudCredsDBO
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for cloud profile
func getCloudCredsDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}

	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := CloudCredsDBO{BaseModelDBO: tenantModel}
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		return base.InQueryParam{
			Param: CloudCredsProjects{
				CloudCredsDBO: param,
				ProjectIDs:    []string{projectID},
			},
			Key:     "SelectCloudCredsByProjectsTemplate1",
			InQuery: true,
		}, nil
	}
	if !auth.IsInfraAdminRole(authContext) {
		projectIDs := auth.GetProjectIDs(authContext)
		if len(projectIDs) == 0 {
			return base.InQueryParam{}, nil
		}
		return base.InQueryParam{
			Param: CloudCredsProjects{
				CloudCredsDBO: param,
				ProjectIDs:    projectIDs,
			},
			Key:     "SelectCloudCredsByProjectsTemplate1",
			InQuery: true,
		}, nil
	}
	return base.InQueryParam{
		Param:   param,
		Key:     "SelectCloudCredsTemplate1",
		InQuery: false,
	}, nil
}

func validateCloudCredsDBO(cc *CloudCredsDBO) error {
	if cc.Type == model.AWSType {
		if cc.AWSCredential == nil {
			return errcode.NewBadRequestError("AWSCredential")
		}
		cc.GCPCredential = nil
		cc.AZCredential = nil
		return nil
	}
	if cc.Type == model.GCPType {
		if cc.GCPCredential == nil {
			return errcode.NewBadRequestError("GCPCredential")
		}
		cc.AWSCredential = nil
		cc.AZCredential = nil
		return nil
	}
	if cc.Type == model.AZType {
		if cc.AZCredential == nil {
			return errcode.NewBadRequestError("AZCredential")
		}
		cc.AWSCredential = nil
		cc.GCPCredential = nil
		return nil
	}
	return errcode.NewBadRequestError("Type")
}

func setCloudCredsDefaults(target *model.CloudCreds, src *model.CloudCreds) {
	if target.Name == "" {
		target.Name = src.Name
	}
	if target.Description == "" {
		target.Description = src.Description
	}
	// only copy the rest if Type did not change
	if target.Type == src.Type {
		if target.AWSCredential == nil {
			target.AWSCredential = src.AWSCredential
		}
		if target.GCPCredential == nil {
			target.GCPCredential = src.GCPCredential
		}
		if target.AZCredential == nil {
			target.AZCredential = src.AZCredential
		}
	}
}

// CloudCredsDBOEqual - check if two CloudCredsDBO have equal value
func CloudCredsDBOEqual(c1 *CloudCredsDBO, c2 *CloudCredsDBO) bool {
	a1 := c1.AWSCredential
	g1 := c1.GCPCredential
	z1 := c1.AZCredential
	a2 := c2.AWSCredential
	g2 := c2.GCPCredential
	z2 := c2.AZCredential
	c1.AWSCredential = nil
	c1.GCPCredential = nil
	c1.AZCredential = nil
	c2.AWSCredential = nil
	c2.GCPCredential = nil
	c2.AZCredential = nil
	b := reflect.DeepEqual(c1, c2)
	if b {
		b = model.MarshalEqual(a1, a2) &&
			model.MarshalEqual(g1, g2) &&
			model.MarshalEqual(z1, z2)
	}
	c1.AWSCredential = a1
	c1.GCPCredential = g1
	c1.AZCredential = z1
	c2.AWSCredential = a2
	c2.GCPCredential = g2
	c2.AZCredential = z2
	return b
}

func encryptCloudCreds(dbAPI *dbObjectModelAPI, context context.Context, doc *model.CloudCreds) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	if doc.Type == model.AWSType {
		if doc.AWSCredential != nil {
			secret, err := keyService.TenantEncrypt(doc.AWSCredential.Secret, token)
			if err != nil {
				return err
			}
			doc.AWSCredential.Secret = secret
		}
	} else if doc.Type == model.GCPType {
		if doc.GCPCredential != nil {
			pkey, err := keyService.TenantEncrypt(doc.GCPCredential.PrivateKey, token)
			if err != nil {
				return err
			}
			doc.GCPCredential.PrivateKey = pkey
		}
	} else if doc.Type == model.AZType {
		if doc.AZCredential != nil {
			pkey, err := keyService.TenantEncrypt(doc.AZCredential.StorageKey, token)
			if err != nil {
				return err
			}
			doc.AZCredential.StorageKey = pkey
		}
	}
	return nil
}

func decryptCloudCreds(dbAPI *dbObjectModelAPI, context context.Context, doc *model.CloudCreds) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	if !doc.IFlagEncrypted {
		return nil
	}
	tenant, err := dbAPI.GetTenant(context, authContext.TenantID)
	if err != nil {
		return err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	return decryptCloudCredsT(dbAPI, context, doc, token)
}

func decryptCloudCredsT(dbAPI *dbObjectModelAPI, context context.Context, doc *model.CloudCreds, token *crypto.Token) error {
	if !doc.IFlagEncrypted {
		return nil
	}
	if doc.Type == model.AWSType {
		if doc.AWSCredential != nil {
			secret, err := keyService.TenantDecrypt(doc.AWSCredential.Secret, token)
			if err != nil {
				return err
			}
			doc.AWSCredential.Secret = secret
		}
	} else if doc.Type == model.GCPType {
		if doc.GCPCredential != nil {
			pkey, err := keyService.TenantDecrypt(doc.GCPCredential.PrivateKey, token)
			if err != nil {
				return err
			}
			doc.GCPCredential.PrivateKey = pkey
		}
	} else if doc.Type == model.AZType {
		if doc.AZCredential != nil {
			pkey, err := keyService.TenantDecrypt(doc.AZCredential.StorageKey, token)
			if err != nil {
				return err
			}
			doc.AZCredential.StorageKey = pkey
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getCloudCreds(ctx context.Context, projectID string, cloudCredsID string, startPage base.PageToken, pageSize int, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.CloudCreds, error) {
	cloudCredss := []model.CloudCreds{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return cloudCredss, err
	}
	queryParam, err := getCloudCredsDBQueryParam(ctx, projectID, cloudCredsID)
	if err != nil {
		return cloudCredss, err
	}
	if queryParam.Key == "" {
		return cloudCredss, nil
	}
	tenant, err := dbAPI.GetTenant(ctx, authContext.TenantID)
	if err != nil {
		return cloudCredss, err
	}
	var pagedQueryFn func(context.Context, base.PageToken, int, func(interface{}) error, string, interface{}) (base.PageToken, error)

	if queryParam.InQuery {
		pagedQueryFn = dbAPI.NotPagedQueryIn
	} else {
		pagedQueryFn = dbAPI.NotPagedQuery
	}
	query, err := buildQuery(entityTypeCloudCreds, queryMap[queryParam.Key], entitiesQueryParam, orderByNameID)
	if err != nil {
		return cloudCredss, err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	_, err = pagedQueryFn(ctx, startPage, pageSize, func(dbObjPtr interface{}) error {
		cloudCreds := model.CloudCreds{}
		err := base.Convert(dbObjPtr, &cloudCreds)
		if err != nil {
			return err
		}
		err = decryptCloudCredsT(dbAPI, ctx, &cloudCreds, token)
		if err != nil {
			return err
		}
		cloudCredss = append(cloudCredss, cloudCreds)
		return nil
	}, query, queryParam.Param)
	// mask out credentials if not edge
	if !auth.IsEdgeRole(authContext) {
		model.MaskCloudCreds(cloudCredss)
	}
	return cloudCredss, err
}

// internal API used by getCloudCredsWV2
func (dbAPI *dbObjectModelAPI) getCloudCredsByProjectsForQuery(context context.Context, projectIDs []string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.CloudCreds, int, error) {
	cloudCredss := []model.CloudCreds{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return cloudCredss, 0, err
	}
	tenantID := authContext.TenantID
	tenant, err := dbAPI.GetTenant(context, tenantID)
	if err != nil {
		return cloudCredss, 0, err
	}
	cloudCredsDBOs := []CloudCredsDBO{}

	var query string
	if len(projectIDs) == 0 {
		query, err = buildLimitQuery(entityTypeCloudCreds, queryMap["SelectCloudCredsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return cloudCredss, 0, err
		}
		err = dbAPI.Query(context, &cloudCredsDBOs, query, tenantIDParam2{TenantID: tenantID})
	} else {
		query, err = buildLimitQuery(entityTypeCloudCreds, queryMap["SelectCloudCredsByProjectsTemplate"], entitiesQueryParam, orderByNameID)
		if err != nil {
			return cloudCredss, 0, err
		}
		err = dbAPI.QueryIn(context, &cloudCredsDBOs, query, tenantIDParam2{TenantID: tenantID, ProjectIDs: projectIDs})
	}

	if err != nil {
		return cloudCredss, 0, err
	}
	if len(cloudCredsDBOs) == 0 {
		return cloudCredss, 0, nil
	}
	totalCount := 0
	first := true
	token := &crypto.Token{EncryptedToken: tenant.Token}
	for _, cloudCredsDBO := range cloudCredsDBOs {
		cloudCreds := model.CloudCreds{}
		if first {
			first = false
			if cloudCredsDBO.TotalCount != nil {
				totalCount = *cloudCredsDBO.TotalCount
			}
		}
		err := base.Convert(&cloudCredsDBO, &cloudCreds)
		if err != nil {
			return []model.CloudCreds{}, 0, err
		}
		err = decryptCloudCredsT(dbAPI, context, &cloudCreds, token)
		if err != nil {
			return []model.CloudCreds{}, 0, err
		}
		cloudCredss = append(cloudCredss, cloudCreds)
	}
	// mask out credentials if not edge
	if !auth.IsEdgeRole(authContext) {
		model.MaskCloudCreds(cloudCredss)
	}
	return cloudCredss, totalCount, nil
}

func (dbAPI *dbObjectModelAPI) getCloudCredsW(context context.Context, projectID string, cloudCredsID string, w io.Writer, req *http.Request) error {
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
	cloudCredss, err := dbAPI.getCloudCreds(context, projectID, cloudCredsID, base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
	if err != nil {
		return err
	}
	if len(cloudCredsID) == 0 {
		return base.DispatchPayload(w, cloudCredss)
	}
	if len(cloudCredss) == 0 {
		return errcode.NewRecordNotFoundError(cloudCredsID)
	}
	return json.NewEncoder(w).Encode(cloudCredss[0])
}

func (dbAPI *dbObjectModelAPI) getCloudCredsWV2(context context.Context, projectID string, cloudCredsID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getCloudCredsDBQueryParam(context, projectID, cloudCredsID)
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.CloudCredsListResponsePayload{CloudCredsList: []model.CloudCreds{}})
	}
	projectIDs := []string{}
	if dbQueryParam.InQuery {
		projectIDs = dbQueryParam.Param.(CloudCredsProjects).ProjectIDs
	}

	queryParam := model.GetEntitiesQueryParam(req)

	cloudCredss, totalCount, err := dbAPI.getCloudCredsByProjectsForQuery(context, projectIDs, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeCloudCreds}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.CloudCredsListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		CloudCredsList:            cloudCredss,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllCloudCreds select all CloudCreds for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllCloudCreds(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.CloudCreds, error) {
	return dbAPI.getCloudCreds(context, "", "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllCloudCredsW select all CloudCreds for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCloudCredsW(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getCloudCredsW(context, "", "", w, req)
}

// SelectAllCloudCredsWV2 select all CloudCreds for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCloudCredsWV2(context context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.getCloudCredsWV2(context, "", "", w, req)
}

// SelectAllCloudCredsForProject select all CloudCreds for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllCloudCredsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.CloudCreds, error) {
	return dbAPI.getCloudCreds(context, projectID, "", base.StartPageToken, base.MaxRowsLimit, entitiesQueryParam)
}

// SelectAllCloudCredsForProjectW select all CloudCreds for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCloudCredsForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getCloudCredsW(context, projectID, "", w, req)
}

// SelectAllCloudCredsForProjectWV2 select all CloudCreds for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllCloudCredsForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.getCloudCredsWV2(context, projectID, "", w, req)
}

// GetCloudCreds get a cloud creds object in the DB
func (dbAPI *dbObjectModelAPI) GetCloudCreds(context context.Context, id string) (model.CloudCreds, error) {
	cloudCreds := model.CloudCreds{}
	cloudCredss, err := dbAPI.getCloudCreds(context, "", id, base.StartPageToken, base.MaxRowsLimit, nil)
	if err != nil {
		return cloudCreds, err
	}
	if len(cloudCredss) == 0 {
		return cloudCreds, errcode.NewRecordNotFoundError(id)
	}
	return cloudCredss[0], nil
}

// GetCloudCredsW get a cloud creds object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetCloudCredsW(context context.Context, id string, w io.Writer, req *http.Request) error {
	if len(id) == 0 {
		return errcode.NewBadRequestError("applicationID")
	}
	return dbAPI.getCloudCredsW(context, "", id, w, req)
}

// CreateCloudCreds creates a cloud creds object in the DB
func (dbAPI *dbObjectModelAPI) CreateCloudCreds(context context.Context, i interface{} /* *model.CloudCreds */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.CloudCreds)
	if !ok {
		return resp, errcode.NewInternalError("CreateCloudCreds: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateCloudCreds doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateCloudCreds doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityCloudCreds,
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
	err = encryptCloudCreds(dbAPI, context, &doc)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", tenantID))
	}
	cloudCredsDBO := CloudCredsDBO{}
	err = base.Convert(&doc, &cloudCredsDBO)
	if err != nil {
		return resp, errcode.NewBadRequestError("cloudCreds")
	}
	err = validateCloudCredsDBO(&cloudCredsDBO)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["CreateCloudCreds"], &cloudCredsDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error creating cloud creds for %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	// no notification in create, since the cloud creds will not be in any project and thus will not be applicable to any edge
	// if callback != nil {
	// 	decryptCloudCreds(dbAPI, context, &doc)
	// 	go callback(context, doc)
	// }
	resp.ID = doc.ID
	GetAuditlogHandler().addCloudProfileAuditLog(dbAPI, context, doc, CREATE)
	return resp, nil
}

// CreateCloudCredsW creates a cloud creds object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateCloudCredsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateCloudCreds, &model.CloudCreds{}, w, r, callback)
}

// CreateCloudCredsWV2 creates a cloud creds object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateCloudCredsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateCloudCreds), &model.CloudCreds{}, w, r, callback)
}

// UpdateCloudCreds updates a cloud creds object in the DB
func (dbAPI *dbObjectModelAPI) UpdateCloudCreds(context context.Context, i interface{} /* *model.CloudCreds */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.CloudCreds)
	if !ok {
		return resp, errcode.NewInternalError("UpdateCloudCreds: type error")
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
		meta.EntityCloudCreds,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now

	// fill in empty fields from existing object
	// to support patching semantics
	// Use private version of get so cloud profile is not encrypted
	cloudCreds, err := dbAPI.getCloudProfileByID(context, doc.ID)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("UpdateCloudCreds[%s]: fetch error", doc.ID))
	}
	// use cloudCreds as default
	setCloudCredsDefaults(&doc, cloudCreds)

	doc.IFlagEncrypted = true
	err = encryptCloudCreds(dbAPI, context, &doc)
	if err != nil {
		return resp, errcode.NewInternalError(fmt.Sprintf("tenantID:%s", tenantID))
	}
	cloudCredsDBO := CloudCredsDBO{}
	err = base.Convert(&doc, &cloudCredsDBO)
	if err != nil {
		return resp, err
	}
	err = validateCloudCredsDBO(&cloudCredsDBO)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(context, queryMap["UpdateCloudCreds"], &cloudCredsDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error updating cloud creds for %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}

	if callback != nil {
		decryptCloudCreds(dbAPI, context, &doc)
		edgeIDs, err := dbAPI.GetAllCloudCredsEdges(context, doc.ID)
		if err == nil && len(edgeIDs) != 0 {
			x := model.ScopedEntity{
				Doc:     doc,
				EdgeIDs: edgeIDs,
			}
			go callback(context, x)
		}
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addCloudProfileAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateCloudCredsW updates a cloud creds object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateCloudCredsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateCloudCreds, &model.CloudCreds{}, w, r, callback)
}

// UpdateCloudCredsWV2 updates a cloud creds object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateCloudCredsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateCloudCreds), &model.CloudCreds{}, w, r, callback)
}

// DeleteCloudCreds delete a cloud creds object in the DB
func (dbAPI *dbObjectModelAPI) DeleteCloudCreds(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	cloudCreds, errGetCreds := dbAPI.GetCloudCreds(context, id)
	err = auth.CheckRBAC(
		authContext,
		meta.EntityCloudCreds,
		meta.OperationDelete,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	doc := model.CloudCreds{
		BaseModel: model.BaseModel{
			TenantID: authContext.TenantID,
			ID:       id,
		},
	}
	var x interface{}
	x = doc
	if callback != nil {
		edgeIDs, err := dbAPI.GetAllCloudCredsEdges(context, doc.ID)
		if err == nil && len(edgeIDs) != 0 {
			x = model.ScopedEntity{
				Doc:     doc,
				EdgeIDs: edgeIDs,
			}
		} else {
			callback = nil
		}
	}
	result, err := DeleteEntity(context, dbAPI, "cloud_creds_model", "id", id, x, callback)
	if err == nil {
		if errGetCreds != nil {
			glog.Error("Error in getting cloud creds : ", errGetCreds.Error())
		} else {
			GetAuditlogHandler().addCloudProfileAuditLog(dbAPI, context, cloudCreds, DELETE)
		}
	}
	return result, err
}

// DeleteCloudCredsW delete a cloud creds object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteCloudCredsW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteCloudCreds, id, w, callback)
}

// DeleteCloudCredsWV2 delete a cloud creds object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteCloudCredsWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteCloudCreds), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) EncryptAllCloudCreds(ctx context.Context) error {
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
		ccs, err := dbAPI.SelectAllCloudCreds(newContext, nil)
		if err != nil {
			return err
		}
		for _, cc := range ccs {
			if !cc.IFlagEncrypted {
				_, err := dbAPI.UpdateCloudCreds(newContext, &cc, nil)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
func (dbAPI *dbObjectModelAPI) EncryptAllCloudCredsW(context context.Context, r io.Reader) error {
	buf := new(bytes.Buffer)
	buf.ReadFrom(r)
	s := buf.String()
	if s != "sherlockkcolrehs" {
		return errcode.NewBadRequestError("payload")
	}
	return dbAPI.EncryptAllCloudCreds(context)
}

func (dbAPI *dbObjectModelAPI) GetAllCloudCredsProjects(context context.Context, cloudCredsID string) ([]string, error) {
	projectIDs := []string{}
	projectCloudCredsDBOs := []ProjectCloudCredsDBO{}
	err := dbAPI.Query(context, &projectCloudCredsDBOs, queryMap["SelectCloudCredsProjects"], ProjectCloudCredsDBO{CloudCredsID: cloudCredsID})
	if err != nil {
		return projectIDs, err
	}
	for _, projectCloudCredsDBO := range projectCloudCredsDBOs {
		projectIDs = append(projectIDs,
			projectCloudCredsDBO.ProjectID)
	}
	return projectIDs, nil
}
func (dbAPI *dbObjectModelAPI) GetAllCloudCredsEdges(context context.Context, cloudCredsID string) ([]string, error) {
	projectIDs, err := dbAPI.GetAllCloudCredsProjects(context, cloudCredsID)
	if err != nil {
		return []string{}, err
	}
	return dbAPI.GetProjectsEdges(context, projectIDs)
}

func (dbAPI *dbObjectModelAPI) getCloudProfilesByIDs(ctx context.Context, cloudProfileIDs []string) ([]model.CloudCreds, error) {
	cloudCredss := []model.CloudCreds{}
	if len(cloudProfileIDs) == 0 {
		return cloudCredss, nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID
	s := strings.Join(cloudProfileIDs, "', '")
	query := fmt.Sprintf("select * from cloud_creds_model where tenant_id = '%s' and id in ('%s')", tenantID, s)
	cloudCredsDBOs := []CloudCredsDBO{}
	err = dbAPI.Query(ctx, &cloudCredsDBOs, query, struct{}{})
	if err != nil {
		return nil, err
	}
	tenant, err := dbAPI.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	token := &crypto.Token{EncryptedToken: tenant.Token}
	for _, cloudCredsDBO := range cloudCredsDBOs {
		cloudCreds := model.CloudCreds{}
		err := base.Convert(&cloudCredsDBO, &cloudCreds)
		if err != nil {
			return nil, err
		}
		err = decryptCloudCredsT(dbAPI, ctx, &cloudCreds, token)
		if err != nil {
			return nil, err
		}
		cloudCredss = append(cloudCredss, cloudCreds)
	}
	return cloudCredss, err
}

func (dbAPI *dbObjectModelAPI) getCloudProfileByID(ctx context.Context, cloudProfileID string) (*model.CloudCreds, error) {
	creds, err := dbAPI.getCloudProfilesByIDs(ctx, []string{cloudProfileID})
	if err != nil {
		return nil, err
	}
	if len(creds) != 1 {
		return nil, errors.NotFound("CloudCredsID", cloudProfileID)
	}

	return &creds[0], nil
}

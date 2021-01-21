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
	"github.com/jmoiron/sqlx/types"
)

const entityTypeStorageProfile = "storageprofile"

func init() {
	queryMap["CreateStorageProfile"] = `INSERT INTO storage_profile_model (id,tenant_id, name, type, aos_config, ebs_config, vsphere_config, iflag_encrypted, created_at, updated_at, isdefault) VALUES (:id, :tenant_id, :name, :type, :aos_config, :ebs_config, :vsphere_config, :iflag_encrypted, :created_at, :updated_at, :isdefault)`
	queryMap["UpdateStorageProfile"] = `UPDATE storage_profile_model SET name = :name, type = :type, aos_config = :aos_config, ebs_config = :ebs_config, vsphere_config = :vsphere_config, iflag_encrypted = :iflag_encrypted, updated_at = :updated_at, isdefault = :isdefault WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["CreateSvcDStorageProfile"] = `INSERT INTO svcdomain_storage_profile_model (svc_domain_id,storage_profile_id) VALUES (:svc_domain_id, :storage_profile_id)`
	queryMap["SelectStorageProfileSvcD"] = `SELECT *, count(*) OVER() as total_count FROM storage_profile_model WHERE tenant_id = :tenant_id AND (id IN (SELECT storage_profile_id FROM svcdomain_storage_profile_model WHERE svc_domain_id = :svc_domain_id)) %s`
	orderByHelper.Setup(entityTypeStorageProfile, []string{"id", "created_at", "updated_at", "name", "type"})
}

// StorageProfileDBO is the DB object for storage profile
type StorageProfileDBO struct {
	model.BaseModelDBO
	Name                 string          `json:"name" db:"name"`
	Type                 string          `json:"type" db:"type"`
	NutanixVolumesConfig *types.JSONText `json:"nutanixVolumesConfig,omitempty" db:"aos_config"`
	EBSConfig            *types.JSONText `json:"ebsStorageConfig,omitempty" db:"ebs_config"`
	VSphereConfig        *types.JSONText `json:"vSphereStorageConfig,omitempty" db:"vsphere_config"`
	IFlagEncrypted       *bool           `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
	IsDefault            *bool           `json:"isDefault,omitempty" db:"isdefault"`
}
type ServiceDomainStorageProfileDBO struct {
	ID               int64  `json:"id" db:"id"`
	SvcDomainID      string `json:"svcDomainId" db:"svc_domain_id"`
	StorageProfileID string `json:"storageProfileID" db:"storage_profile_id"`
}

func validateStorageProfileDBO(sp *StorageProfileDBO) error {
	if sp.Name == "" {
		return errcode.NewBadRequestError("Name")
	}
	if sp.Type == "NutanixVolumes" {
		if sp.NutanixVolumesConfig == nil {
			return errcode.NewBadRequestError("NutanixVolumesConfig")
		}
		sp.EBSConfig = nil
		sp.VSphereConfig = nil
		return nil
	}
	if sp.Type == "EBS" {
		if sp.EBSConfig == nil {
			return errcode.NewBadRequestError("EBSConfig")
		}
		sp.NutanixVolumesConfig = nil
		sp.VSphereConfig = nil
		return nil
	}
	if sp.Type == "vSphere" {
		if sp.VSphereConfig == nil {
			return errcode.NewBadRequestError("VSphereConfig")
		}
		sp.NutanixVolumesConfig = nil
		sp.EBSConfig = nil
		return nil
	}
	return errcode.NewBadRequestError("Type")
}

// internal API used by SelectAllStorageProfileForSvcDW
func (dbAPI *dbObjectModelAPI) getStorageProfileBySvcDomainForQuery(context context.Context, svcDomainID string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.StorageProfile, int, error) {
	storageProfileList := []model.StorageProfile{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return storageProfileList, 0, err
	}
	tenantID := authContext.TenantID
	if err != nil {
		return storageProfileList, 0, err
	}
	storageProfileDBOs := []StorageProfileDBO{}

	var query string
	query, err = buildLimitQuery(entityTypeStorageProfile, queryMap["SelectStorageProfileSvcD"], entitiesQueryParam, orderByNameID)
	if err != nil {
		return storageProfileList, 0, err
	}
	type SvcDomainIDParam struct {
		SVCDomainID string `db:"svc_domain_id"`
		TenantID    string `db:"tenant_id"`
	}
	err = dbAPI.QueryIn(context, &storageProfileDBOs, query, SvcDomainIDParam{SVCDomainID: svcDomainID, TenantID: tenantID})

	if err != nil {
		return storageProfileList, 0, err
	}
	if len(storageProfileDBOs) == 0 {
		return storageProfileList, 0, nil
	}
	totalCount := 0
	first := true
	for _, storageProfileDBO := range storageProfileDBOs {
		storageProfile := model.StorageProfile{}
		if first {
			first = false
			if storageProfileDBO.TotalCount != nil {
				totalCount = *storageProfileDBO.TotalCount
			}
		}
		err := base.Convert(&storageProfileDBO, &storageProfile)
		if err != nil {
			return []model.StorageProfile{}, 0, err
		}
		storageProfileList = append(storageProfileList, storageProfile)
	}
	return storageProfileList, totalCount, nil
}

// SelectAllStorageProfileForServiceDomainW select all storage profiles for the given tenant and svc domain, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllStorageProfileForServiceDomainW(context context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	storageProfileList, totalCount, err := dbAPI.getStorageProfileBySvcDomainForQuery(context, svcDomainID, queryParam)
	if err != nil {
		return err
	}
	queryInfo := ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeStorageProfile}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.StorageProfileListResponsePayload{
		EntityListResponsePayload: entityListResponsePayload,
		StorageProfileList:        storageProfileList,
	}
	return json.NewEncoder(w).Encode(r)

}

// CreateStorageProfile creates a storage profile object in the DB
func (dbAPI *dbObjectModelAPI) CreateStorageProfile(context context.Context, svcDomainID string, sp *model.StorageProfile) (interface{}, error) {

	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	doc := *sp
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateStorageProfile doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateStorageProfile doc.ID was invalid, update it to %s\n"), doc.ID)
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityStorageProfile,
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
	doc.IFlagEncrypted = false
	spDBO := StorageProfileDBO{}
	err = base.Convert(&doc, &spDBO)
	if err != nil {
		return resp, errcode.NewBadRequestError("storageProfile")
	}
	err = validateStorageProfileDBO(&spDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err = tx.NamedExec(context, queryMap["CreateStorageProfile"], &spDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error creating storage profile for %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		svcDSPDBO := ServiceDomainStorageProfileDBO{SvcDomainID: svcDomainID, StorageProfileID: spDBO.ID}
		_, err = tx.NamedExec(context, queryMap["CreateSvcDStorageProfile"], &svcDSPDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error in creating storage profile association for service domain %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	return resp, nil
}

// CreateStorageProfileW creates a storage profile object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateStorageProfileW(context context.Context, svcDomainID string, w io.Writer, r *http.Request) error {

	reader := io.Reader(r.Body)
	doc := model.StorageProfile{}
	err := base.Decode(&reader, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into storage profile payload. Error: %s"), err.Error())
		return err
	}
	resp, err := dbAPI.CreateStorageProfile(context, svcDomainID, &doc)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)

}

// UpdateStorageProfile creates a storage profile object in the DB
func (dbAPI *dbObjectModelAPI) UpdateStorageProfile(context context.Context, svcDomainID string, ID string, sp *model.StorageProfile) (interface{}, error) {

	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	doc := *sp
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if doc.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	if ID != doc.ID {
		return resp, errcode.NewBadRequestError("ID")
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityStorageProfile,
		meta.OperationUpdate,
		auth.RbacContext{})
	if err != nil {
		return resp, err
	}
	//Check if node is onboarded
	isSvcDomainOnBoarded := false
	nodesInfo, _, err := dbAPI.getNodesInfoV2(context, "", svcDomainID, nil)
	if err != nil {
		return resp, err
	}
	for _, nodeInfo := range nodesInfo {
		if nodeInfo.Onboarded {
			isSvcDomainOnBoarded = true
			break
		}
	}
	if isSvcDomainOnBoarded {
		return resp, errcode.NewInternalError("Service Domain is on boarded ,so cannot edit the storage profile")
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	doc.IFlagEncrypted = false
	spDBO := StorageProfileDBO{}
	err = base.Convert(&doc, &spDBO)
	if err != nil {
		return resp, errcode.NewBadRequestError("storageProfile")
	}
	err = validateStorageProfileDBO(&spDBO)
	if err != nil {
		return resp, err
	}

	err = dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		_, err = tx.NamedExec(context, queryMap["UpdateStorageProfile"], &spDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context, "Error updating storage profile for %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
			return errcode.TranslateDatabaseError(doc.ID, err)
		}
		return nil
	})
	if err != nil {
		return resp, err
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateStorageProfileW updates a storage profile object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateStorageProfileW(context context.Context, svcDomainID string, ID string, w io.Writer, r *http.Request, callback func(context.Context, interface{}) error) error {

	reader := io.Reader(r.Body)
	doc := model.StorageProfile{}
	err := base.Decode(&reader, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into storage profile payload. Error: %s"), err.Error())
		return err
	}
	resp, err := dbAPI.UpdateStorageProfile(context, svcDomainID, ID, &doc)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)

}

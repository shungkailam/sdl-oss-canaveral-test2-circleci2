package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
)

func init() {
	queryMap["SelectEdgeCerts"] = `SELECT * FROM edge_cert_model WHERE tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (:edge_id = '' OR edge_id = :edge_id)`
	queryMap["CreateEdgeCert"] = `INSERT INTO edge_cert_model (id, version, tenant_id, edge_id, certificate, private_key, client_certificate, client_private_key, edge_certificate, edge_private_key, locked, created_at, updated_at) VALUES (:id, :version, :tenant_id, :edge_id, :certificate, :private_key, :client_certificate, :client_private_key, :edge_certificate, :edge_private_key, :locked, :created_at, :updated_at)`
	queryMap["UpdateEdgeCert"] = `UPDATE edge_cert_model SET version = :version, certificate = :certificate, private_key = :private_key, locked = :locked, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	queryMap["SelectTenantRootCA"] = `SELECT * FROM tenant_rootca_model WHERE tenant_id = :tenant_id`
	queryMap["SetEdgeCertLock"] = `UPDATE edge_cert_model SET version = :version, locked = :locked, updated_at = :updated_at WHERE tenant_id = :tenant_id AND edge_id = :edge_id`
}

// EdgeCertDBO is DB object model for edge cert
type EdgeCertDBO struct {
	model.EdgeBaseModelDBO
	model.EdgeCertCore
}

func (dbAPI *dbObjectModelAPI) GetTenantRootCA(tenantID string) (string, error) {
	// We need to populate CACertificate from the tenant_rootca_model before sending out the response.
	tenantRootCADBOs := []model.TenantRootCA{}
	baseModelDBO := model.BaseModel{TenantID: tenantID}
	tenantRootCAParam := model.TenantRootCA{BaseModel: baseModelDBO}
	err := dbAPI.Query(context.Background(), &tenantRootCADBOs, queryMap["SelectTenantRootCA"],
		tenantRootCAParam)
	if err != nil {
		return "", err
	}
	if len(tenantRootCADBOs) == 0 {
		return "", errcode.NewRecordNotFoundError(tenantID)
	}
	return tenantRootCADBOs[0].Certificate, nil
}

// SelectAllEdgeCerts select all edge certs for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllEdgeCerts(context context.Context) ([]model.EdgeCert, error) {
	edgeCerts := []model.EdgeCert{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeCerts, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	param := EdgeCertDBO{EdgeBaseModelDBO: edgeModel}

	// We need to populate CACertificate from the tenant_rootca_model before sending out the response.
	caCert, err := dbAPI.GetTenantRootCA(tenantID)
	if err != nil {
		return edgeCerts, err
	}
	_, err = dbAPI.PagedQuery(context, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		edgeCert := model.EdgeCert{}
		err := base.Convert(dbObjPtr, &edgeCert)
		if err != nil {
			return err
		}
		edgeCert.EdgeCertCore.CACertificate = caCert
		edgeCerts = append(edgeCerts, edgeCert)
		return nil
	}, queryMap["SelectEdgeCerts"], param)
	return edgeCerts, err
}

// SelectAllEdgeCertsW select all edge certs for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllEdgeCertsW(context context.Context, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	edgeCertDBOs := []EdgeCertDBO{}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	param := EdgeCertDBO{EdgeBaseModelDBO: edgeModel}
	err = dbAPI.Query(context, &edgeCertDBOs, queryMap["SelectEdgeCerts"], param)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, edgeCertDBOs); handled {
	// 	return err
	// }
	// We need to populate CACertificate from the tenant_rootca_model before sending out the response.
	caCert, err := dbAPI.GetTenantRootCA(tenantID)
	if err != nil {
		return err
	}
	for _, edgeCertDBO := range edgeCertDBOs {
		edgeCertDBO.EdgeCertCore.CACertificate = caCert
	}

	return base.DispatchPayload(w, edgeCertDBOs)
}

// GetEdgeCert get a edge cert object in the DB
func (dbAPI *dbObjectModelAPI) GetEdgeCert(context context.Context, id string) (model.EdgeCert, error) {
	edgeCert := model.EdgeCert{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeCert, err
	}
	tenantID := authContext.TenantID
	edgeCertDBOs := []EdgeCertDBO{}
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	param := EdgeCertDBO{EdgeBaseModelDBO: edgeModel}
	if len(id) == 0 {
		return edgeCert, errcode.NewBadRequestError("edgeCertID")
	}
	err = dbAPI.Query(context, &edgeCertDBOs, queryMap["SelectEdgeCerts"], param)
	if err != nil {
		return edgeCert, err
	}
	if len(edgeCertDBOs) == 0 {
		return edgeCert, errcode.NewRecordNotFoundError(id)
	}
	err = base.Convert(&edgeCertDBOs[0], &edgeCert)
	if err != nil {
		return edgeCert, err
	}

	// We need to populate CACertificate from the tenant_rootca_model before sending out the response.
	caCert, err := dbAPI.GetTenantRootCA(tenantID)
	if err != nil {
		return edgeCert, err
	}
	edgeCert.EdgeCertCore.CACertificate = caCert

	return edgeCert, nil
}

// GetEdgeCertW get a edge cert object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetEdgeCertW(context context.Context, id string, w io.Writer, req *http.Request) error {
	edgeCert, err := dbAPI.GetEdgeCert(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, edgeCert)
}

// GetEdgeCertByEdgeID get a edge cert object in the DB
func (dbAPI *dbObjectModelAPI) GetEdgeCertByEdgeID(context context.Context, edgeID string) (model.EdgeCert, error) {
	edgeCert := model.EdgeCert{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return edgeCert, err
	}
	tenantID := authContext.TenantID
	edgeCertDBOs := []EdgeCertDBO{}
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel, EdgeID: edgeID}
	param := EdgeCertDBO{EdgeBaseModelDBO: edgeModel}
	err = dbAPI.Query(context, &edgeCertDBOs, queryMap["SelectEdgeCerts"], param)
	if err != nil {
		return edgeCert, err
	}
	if len(edgeCertDBOs) == 0 {
		return edgeCert, errcode.NewRecordNotFoundError(edgeID)
	}
	err = base.Convert(&edgeCertDBOs[0], &edgeCert)
	if err != nil {
		return edgeCert, err
	}

	// We need to populate CACertificate from the tenant_rootca_model before sending out the response.
	caCert, err := dbAPI.GetTenantRootCA(tenantID)
	if err != nil {
		return edgeCert, err
	}
	edgeCert.EdgeCertCore.CACertificate = caCert

	return edgeCert, nil
}

// CreateEdgeCert creates an edge cert object in the DB
func (dbAPI *dbObjectModelAPI) CreateEdgeCert(context context.Context, i interface{} /* *model.EdgeCert */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.EdgeCert)
	if !ok {
		return resp, errcode.NewInternalError("CreateEdgeCert: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	if base.CheckID(doc.ID) {
		glog.Infof(base.PrefixRequestID(context, "CreateEdgeCert doc.ID was %s\n"), doc.ID)
	} else {
		doc.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(context, "CreateEdgeCert doc.ID was invalid, update it to %s\n"), doc.ID)
	}

	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	edgeCertDBO := EdgeCertDBO{}
	err = base.Convert(&doc, &edgeCertDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["CreateEdgeCert"], &edgeCertDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in creating edge cert for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// CreateEdgeCertW creates an edge cert object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateEdgeCertW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateEdgeCert, &model.EdgeCert{}, w, r, callback)
}

// CreateEdgeCertWV2 creates an edge cert object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateEdgeCertWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateEdgeCert), &model.EdgeCert{}, w, r, callback)
}

// UpdateEdgeCert update an edge object in the DB
func (dbAPI *dbObjectModelAPI) UpdateEdgeCert(context context.Context, i interface{} /* *model.EdgeCert */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.EdgeCert)
	if !ok {
		return resp, errcode.NewInternalError("UpdateEdgeCert: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.UpdatedAt = now
	edgeCertDBO := EdgeCertDBO{}
	err = base.Convert(&doc, &edgeCertDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateEdgeCert"], &edgeCertDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating edge cert for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateEdgeCertW update an edge object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateEdgeCertW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateEdgeCert, &model.EdgeCert{}, w, r, callback)
}

// UpdateEdgeCertWV2 update an edge object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateEdgeCertWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateEdgeCert), &model.EdgeCert{}, w, r, callback)
}

// DeleteEdgeCert delete a edge cert object in the DB
func (dbAPI *dbObjectModelAPI) DeleteEdgeCert(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return model.DeleteDocumentResponse{}, err
	}
	doc := model.EdgeCert{
		EdgeBaseModel: model.EdgeBaseModel{
			BaseModel: model.BaseModel{
				TenantID: authContext.TenantID,
				ID:       id,
			},
		},
	}
	return DeleteEntity(context, dbAPI, "edge_cert_model", "id", id, doc, callback)
}

// DeleteEdgeCertW delete a edge cert object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteEdgeCertW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteEdgeCert, id, w, callback)
}

// DeleteEdgeCertWV2 delete a edge cert object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteEdgeCertWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteEdgeCert), id, w, callback)
}

// DeleteEdgeCertByEdgeID delete edge cert by edge id in DB
func (dbAPI *dbObjectModelAPI) DeleteEdgeCertByEdgeID(context context.Context, edgeID string) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	_, err = dbAPI.Delete(context, "edge_cert_model", map[string]interface{}{"tenant_id": tenantID, "edge_id": edgeID})
	if err != nil {
		return resp, err
	}
	resp = model.DeleteDocumentResponse{
		ID: edgeID,
	}
	return resp, nil
}

// SetEdgeCertLock update lock on an edge cert object in the DB
func (dbAPI *dbObjectModelAPI) SetEdgeCertLock(context context.Context, edgeClusterID string, locked bool) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	// must be infra admin or edge self to do this
	if !auth.IsInfraAdminRole(authContext) &&
		(!auth.IsEdgeRole(authContext) ||
			auth.GetEdgeID(authContext) != edgeClusterID) {
		return errcode.NewPermissionDeniedError("RBAC")
	}
	_, err = dbAPI.GetEdgeCluster(context, edgeClusterID)
	if err != nil {
		return err
	}

	tenantID := authContext.TenantID
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()

	edgeCertDBO := EdgeCertDBO{}
	edgeCertDBO.EdgeID = edgeClusterID
	edgeCertDBO.TenantID = tenantID
	edgeCertDBO.Locked = locked
	edgeCertDBO.Version = float64(epochInNanoSecs)
	edgeCertDBO.UpdatedAt = now

	_, err = dbAPI.NamedExec(context, queryMap["SetEdgeCertLock"], &edgeCertDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating lock on edge cert for edge ID %s and tenant ID %s. Error: %s"), edgeClusterID, tenantID, err.Error())
		return errcode.TranslateDatabaseError(edgeClusterID, err)
	}
	return nil
}

// SetEdgeCertLockW update lock on an edge cert object in the DB, schedule auto locking as appropriate
func (dbAPI *dbObjectModelAPI) SetEdgeCertLockW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	param := model.EdgeCertLockParam{}
	err := base.Decode(&r, &param)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	// EdgeClusterID is required
	if param.EdgeClusterID == "" {
		return errcode.NewMalformedBadRequestError("body")
	}
	err = dbAPI.SetEdgeCertLock(ctx, param.EdgeClusterID, param.Locked)
	if err == nil && param.Locked == false && param.DurationSeconds != 0 {
		// schedule auto locking
		authContext, err2 := base.GetAuthContext(ctx)
		if err2 == nil {
			reqID := base.GetRequestID(ctx)
			timer := time.NewTimer(time.Duration(param.DurationSeconds) * time.Second)
			go func() {
				ctx2 := base.GetAdminContext(reqID, authContext.TenantID)
				<-timer.C
				err2 = dbAPI.SetEdgeCertLock(ctx2, param.EdgeClusterID, true)
				if err2 != nil {
					glog.Errorf(base.PrefixRequestID(ctx2, "Error in auto locking on edge cert for edge ID %s and tenant ID %s. Error: %s"), param.EdgeClusterID, authContext.TenantID, err2.Error())
				}
			}()
		}
	}
	return err
}

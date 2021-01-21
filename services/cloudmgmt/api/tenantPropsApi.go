package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"io"
	"net/http"

	"github.com/golang/glog"
)

func init() {
	queryMap["SelectTenantProps"] = `SELECT * FROM tenant_props_model WHERE tenant_id = :tenant_id`
	queryMap["CreateTenantProps"] = `INSERT INTO tenant_props_model (tenant_id, props, version, created_at, updated_at) VALUES (:tenant_id, :props, :version, :created_at, :updated_at) ON CONFLICT (tenant_id) DO UPDATE SET version = :version, props = :props, updated_at = :updated_at WHERE tenant_props_model.tenant_id = :tenant_id`

}

// make sure logged in user tenant id matches path tenant id
func validateTenantProps(context context.Context, p *model.TenantProps) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	if p.TenantID == "" {
		p.TenantID = authContext.TenantID
		return nil
	}
	return acValidateTenantID(authContext, p.TenantID)
}
func validateTenantID(context context.Context, id string) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	return acValidateTenantID(authContext, id)
}
func acValidateTenantID(authContext *base.AuthContext, id string) error {
	if authContext.TenantID != id {
		return errcode.NewBadRequestError("tenantID")
	}
	// ENG-224548: allow CRUD operations of tenantprops from all users
	// if !auth.IsInfraAdminRole(authContext) {
	// 	return errcode.NewPermissionDeniedError("INFRA_ADMIN")
	// }
	return nil
}

type TenantPropsDBO struct {
	TenantID string `json:"tenantId" db:"tenant_id"`
}

func (dbAPI *dbObjectModelAPI) GetTenantProps(ctx context.Context, id string) (model.TenantProps, error) {
	tenantProps := model.TenantProps{}
	err := validateTenantID(ctx, id)
	if err != nil {
		return tenantProps, err
	}
	param := model.TenantProps{TenantID: id}
	tenantPropsList := []model.TenantProps{}
	err = dbAPI.Query(ctx, &tenantPropsList, queryMap["SelectTenantProps"], param)
	if err != nil {
		return tenantProps, errcode.TranslateDatabaseError(id, err)
	}
	if len(tenantPropsList) == 1 {
		return tenantPropsList[0], nil
	}
	tenantProps.TenantID = id
	return tenantProps, nil
}

func (dbAPI *dbObjectModelAPI) GetTenantPropsW(context context.Context, id string, w io.Writer, req *http.Request) error {
	tenantProps, err := dbAPI.GetTenantProps(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, tenantProps)
}

// UpdateTenantProps updates a tenant props object in the DB
func (dbAPI *dbObjectModelAPI) UpdateTenantProps(ctx context.Context, i interface{} /* *model.TenantProps */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	p, ok := i.(*model.TenantProps)
	if !ok {
		return resp, errcode.NewInternalError("UpdateTenantProps: type error")
	}
	err := validateTenantProps(ctx, p)
	if err != nil {
		return resp, err
	}

	doc := *p
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	_, err = dbAPI.NamedExec(ctx, queryMap["CreateTenantProps"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating tenant props for tenant ID %s. Error: %s"), doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.TenantID, err)
	}
	resp.ID = doc.TenantID
	return resp, nil
}

// UpdateTenantPropsW update a tenant props object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateTenantPropsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateTenantProps, &model.TenantProps{}, w, r, callback)
}

// UpdateTenantPropsWV2 update a tenant props object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateTenantPropsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateTenantProps), &model.TenantProps{}, w, r, callback)
}

// DeleteTenantProps delete a tenant props object in the DB
func (dbAPI *dbObjectModelAPI) DeleteTenantProps(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	// default empty response
	resp := model.DeleteDocumentResponse{}
	err := validateTenantID(context, id)
	if err != nil {
		// treat as not found
		return resp, nil
	}
	tenantProps, err := dbAPI.GetTenantProps(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	return DeleteEntity(context, dbAPI, "tenant_props_model", "tenant_id", id, tenantProps, callback)
}

// DeleteTenantPropsW delete a tenant props object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteTenantPropsW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteTenantProps, id, w, callback)
}

// DeleteTenantPropsWV2 delete a tenant props object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteTenantPropsWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteTenantProps), id, w, callback)
}

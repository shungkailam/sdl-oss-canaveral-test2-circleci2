package api

import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const (
	CreateTenantClientTimeout = time.Second * 120
)

func (dbAPI *dbObjectModelAPI) selectTenants(ctx context.Context, request *gapi.GetTenantsRequest) ([]model.Tenant, error) {
	tenants := []model.Tenant{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetTenants(ctx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenants. Error: %s"), err.Error())
			return err
		}
		for _, gTenant := range response.GetTenants() {
			tenant := model.Tenant{}
			err = base.Convert(gTenant, &tenant)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data. Error: %s"), err.Error())
				return err
			}
			tenants = append(tenants, tenant)
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	return tenants, err
}

// SelectAllTenants select all tenants in the system
func (dbAPI *dbObjectModelAPI) SelectAllTenants(ctx context.Context) ([]model.Tenant, error) {
	request := &gapi.GetTenantsRequest{}
	return dbAPI.selectTenants(ctx, request)
}

// SelectAllTenantsW select all tenants in the system, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllTenantsW(ctx context.Context, w io.Writer, req *http.Request) error {
	tenants, err := dbAPI.SelectAllTenants(ctx)
	if err != nil {
		return err
	}
	// if handled, err := handleEtag(w, etag, tenantDBOs); handled {
	// 	return err
	// }
	return base.DispatchPayload(w, tenants)
}

// GetTenant get a tenant object in the DB. ID is either the tenant ID or the external ID
func (dbAPI *dbObjectModelAPI) GetTenant(ctx context.Context, id string) (model.Tenant, error) {
	tenant := model.Tenant{}
	if len(id) == 0 {
		return tenant, errcode.NewBadRequestError("tenantID")
	}
	request := &gapi.GetTenantsRequest{Id: id, ExternalId: id}
	tenants, err := dbAPI.selectTenants(ctx, request)
	if err != nil {
		return tenant, err
	}
	if len(tenants) == 0 {
		return tenant, errcode.NewRecordNotFoundError(id)
	}
	err = base.Convert(&tenants[0], &tenant)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenant data in GetTenant. Error: %s"), err.Error())
	}
	return tenant, err
}

// GetTenantW get a tenant info object in the DB, write output into writer
// Note: unlike GetTenant, this method writes TenantInfo into writer, not Tenant.
// Reason is we now expose this method via REST API
// (indirectly via GetTenantSelfW to provide access to tenant profile)
// and we don't want to return info like tenant token, etc.
func (dbAPI *dbObjectModelAPI) GetTenantW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	tenant, err := dbAPI.GetTenant(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, tenant.ToTenantInfo())
}

// GetTenantSelfW get current tenant object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetTenantSelfW(ctx context.Context, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return errcode.NewPermissionDeniedError("RBAC")
	}
	tenantID := authContext.TenantID

	return dbAPI.GetTenantW(ctx, tenantID, w, req)
}

// CreateTenant creates a tenant object in the DB
func (dbAPI *dbObjectModelAPI) CreateTenant(ctx context.Context, i interface{} /* *model.Tenant */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, _ := base.GetAuthContext(ctx)
	p, ok := i.(*model.Tenant)
	if !ok {
		return resp, errcode.NewInternalError("CreateTenant: type error")
	}
	doc := *p
	gTenant := &gapi.Tenant{}
	err := base.Convert(p, gTenant)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenant data in GetTenant. Error: %s"), err.Error())
		return resp, err
	}
	request := &gapi.CreateTenantRequest{Tenant: gTenant}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.CreateTenant(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to create tenant. Error: %s"), err.Error())
			return err
		}
		resp.ID = response.GetId()
		return nil
	}
	err = service.CallClientWithTimeout(ctx, service.AccountService, handler, CreateTenantClientTimeout)
	if err == nil {
		if err != nil {
			return nil, err
		}
		if authContext != nil && authContext.TenantID == base.OperatorTenantID {
			// Interfering with tests if the tenant creator is not set to operator tenant
			err1 := dbAPI.CreateBuiltinTenantObjects(ctx, resp.ID)
			if err1 != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Failed to create builtin tenant objects. Error: %s"), err.Error())
			}
		}
		if callback != nil {
			doc.ID = resp.ID
			go callback(ctx, doc)
		}
	} else {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return resp, err
}

// CreateTenantW creates a tenant object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateTenantW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateTenant, &model.Tenant{}, w, r, callback)
}

// CreateTenantWV2 creates a tenant object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateTenantWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateTenant), &model.Tenant{}, w, r, callback)
}

// UpdateTenant update a tenant object in the DB
func (dbAPI *dbObjectModelAPI) UpdateTenant(ctx context.Context, i interface{} /* *model.Tenant */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	p, ok := i.(*model.Tenant)
	if !ok {
		return resp, errcode.NewInternalError("UpdateTenant: type error")
	}
	doc := *p
	gTenant := &gapi.Tenant{}
	err := base.Convert(p, gTenant)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenant data in UpdateTenant. Error: %s"), err.Error())
		return resp, err
	}
	request := &gapi.UpdateTenantRequest{Tenant: gTenant}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.UpdateTenant(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to update tenant. Error: %s"), err.Error())
			return err
		}
		resp.ID = response.GetId()
		return nil
	}
	err = service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	} else if callback != nil {
		doc.ID = resp.ID
		go callback(ctx, doc)
	}
	return resp, err
}

// UpdateTenantW update a tenant object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateTenantW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateTenant, &model.Tenant{}, w, r, callback)
}

// UpdateTenantWV2 update a tenant object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateTenantWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateTenant), &model.Tenant{}, w, r, callback)
}

// DeleteTenant delete a tenant object in the DB
func (dbAPI *dbObjectModelAPI) DeleteTenant(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	// delete all audit logs for this tenant, ignore error
	dbAPI.DeleteTenantAuditLogs(ctx)

	doc := model.Tenant{
		ID: id,
	}
	resp := model.DeleteDocumentResponse{}
	request := &gapi.DeleteTenantRequest{Id: id}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.DeleteTenant(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to delete tenant. Error: %s"), err.Error())
			return err
		}
		resp.ID = response.GetId()
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	} else if callback != nil {
		doc.ID = resp.ID
		go callback(ctx, doc)
	}
	return resp, err
}

// DeleteTenantW delete a tenant object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteTenantW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteTenant, id, w, callback)
}

func (dbAPI *dbObjectModelAPI) DeleteTenantWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteTenant), id, w, callback)
}

// CreateBuiltinTenantObjects creates the builtIn objects like script runtime, category etc
func (dbAPI *dbObjectModelAPI) CreateBuiltinTenantObjects(ctx context.Context, tenantID string) error {
	return dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err := createBuiltinCategories(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		err = createBuiltinScriptRuntimes(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		return createBuiltinProjects(ctx, tx, tenantID)
	})
}

// DeleteBuiltinTenantObjects deletes the builtIn objects like script runtime, category etc
func (dbAPI *dbObjectModelAPI) DeleteBuiltinTenantObjects(ctx context.Context, tenantID string) error {
	return dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err := deleteBuiltinProjects(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		err = deleteBuiltinScriptRuntimes(ctx, tx, tenantID)
		if err != nil {
			return err
		}
		return deleteBuiltinCategories(ctx, tx, tenantID)
	})
}

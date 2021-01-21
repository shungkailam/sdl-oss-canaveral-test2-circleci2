package api

import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/cloudmgmt/cfssl"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/metrics"
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// TODO add external_id in the SQL statements once the DB changes are in to not break the test or existing deployment
	SelectTenants = `SELECT * FROM tenant_model WHERE ((:id = '' OR id = :id) OR (:external_id = '' OR external_id = :external_id)) AND deleted_at IS NULL`
	CreateTenant  = `INSERT INTO tenant_model (id, external_id, version, name, token, description, created_by, created_at, updated_at) VALUES (:id, :external_id, :version, :name, :token, :description, :created_by, :created_at, :updated_at)`
	UpdateTenant  = `UPDATE tenant_model SET external_id = :external_id, version = :version, name = :name, token = :token, description = :description, updated_at = :updated_at WHERE id = :id AND deleted_at IS NULL`
	DeleteTenant  = `UPDATE tenant_model SET external_id = NULL, deleted_at=(now() at time zone 'utc') WHERE id=:id AND (:created_by='' OR created_by = :created_by) AND deleted_at IS NULL`
	// name is special, postgresql way of escape is with double quotes
	CloneTenantUnitTestTemplate       = `INSERT INTO tenant_model (id, version, name, token, description, created_by, created_at, updated_at, external_id, profile) SELECT '%s' id, version, '%s' "name", token, description, :created_by, created_at, updated_at, external_id, profile from tenant_model where id = '%s' AND deleted_at IS NULL`
	CloneTenantRootCAUnitTestTemplate = `INSERT INTO tenant_rootca_model (id, version, tenant_id, certificate, private_key, aws_data_key, created_at, updated_at) SELECT '%s' id, version, '%s' tenant_id, certificate, private_key, aws_data_key, created_at, updated_at from tenant_rootca_model where tenant_id = '%s'`

	// TODO revisit later
	RenameNodeSerialNumberQuery = "update edge_device_model set serial_number=concat(serial_number, '.%d.ntnx-del'), updated_at = (now() at time zone 'utc') where tenant_id='%s' and serial_number not like '%%.ntnx-del'"
)

var (
	mxUnitTest                  sync.Mutex
	unitTestSharedTenantCreated bool
)

// TenantDBO is DB object model for tenant
type TenantDBO struct {
	ID          string          `json:"id" db:"id"`
	ExternalID  *string         `json:"externalId,omitempty" db:"external_id"`
	Version     float64         `json:"version,omitempty" db:"version"`
	Name        string          `json:"name" db:"name"`
	Token       string          `json:"token" db:"token"`
	Description *string         `json:"description" db:"description"`
	Profile     *types.JSONText `json:"profile" db:"profile"`
	CreatedBy   *string         `json:"createdBy" db:"created_by"`
	CreatedAt   time.Time       `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time       `json:"updatedAt" db:"updated_at"`
	DeletedAt   *time.Time      `json:"deletedAt,omitempty" db:"deleted_at"`
}

// Tenant APIs
func (server *apiServer) CreateTenant(ctx context.Context, request *gapi.CreateTenantRequest) (*gapi.CreateTenantResponse, error) {
	if cfssl.IsUnitTestMode() {
		return server.createTenantUnitTest(ctx, request)
	} else {
		return server.createTenantNormal(ctx, request)
	}
}

// createTenantUnitTest ENG-289403
// In unit tests we create lots of test tenants. This causes cfssl server
// to be overwhelmed, leading to request timeout and frequent test failures.
// As a workaround, for unit test we would create a prototype tenant
// with ID cfssl.UnitTestTenantID. All other test tenants would clone
// from this tenant to avoid cfssl create tenant root ca / certs overhead.
func (server *apiServer) createTenantUnitTest(ctx context.Context, request *gapi.CreateTenantRequest) (*gapi.CreateTenantResponse, error) {
	mxUnitTest.Lock()
	defer mxUnitTest.Unlock()
	// create the shared test tenant if it hasn't been created already
	// since account server is scale out, each instance will
	// try to create the shared tenant in a best effort approach
	// to minimize need for distributed synchronization
	if !unitTestSharedTenantCreated {
		// first get the shared tenant
		greq := &gapi.GetTenantsRequest{
			Id: cfssl.UnitTestTenantID,
		}
		r, err := server.GetTenants(ctx, greq)
		// if get failed or shared tenant not there,
		// then do best effort create, ignore error
		if err != nil || len(r.Tenants) != 1 {
			cr := &gapi.CreateTenantRequest{
				Tenant: &gapi.Tenant{
					Id:   cfssl.UnitTestTenantID,
					Name: "CMS unit test tenant",
				},
			}
			server.createTenantNormal(ctx, cr)
		}
		// remember it so we only try create shared tenant once per run
		unitTestSharedTenantCreated = true
	}
	return server.cloneTenantUnitTest(ctx, request)
}
func (server *apiServer) cloneTenantUnitTest(ctx context.Context, request *gapi.CreateTenantRequest) (*gapi.CreateTenantResponse, error) {
	authContext, _ := base.GetAuthContext(ctx)
	tenantDBO := TenantDBO{}
	gTenant := request.GetTenant()
	err := base.Convert(gTenant, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant data conversion error for tenant ID %s. Error: %s\n"), gTenant.GetId(), err.Error())
		return nil, err
	}
	if base.CheckID(tenantDBO.ID) {
		tenantDBO.ID = gTenant.GetId()
		glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant doc.ID was %s\n"), tenantDBO.ID)
	} else {
		tenantDBO.ID = base.GetUUID()
		glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant doc.ID was invalid, update it to %s\n"), tenantDBO.ID)
	}
	if authContext != nil {
		// Set the creator for this copied tenant ID
		userID := authContext.GetUserID()
		if userID != "" {
			tenantDBO.CreatedBy = base.StringPtr(userID)
		}
	}
	stmtCloneTenant := fmt.Sprintf(CloneTenantUnitTestTemplate, tenantDBO.ID, tenantDBO.Name, cfssl.UnitTestTenantID)
	_, err = server.NamedExec(ctx, stmtCloneTenant, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in cloning tenant %s. Error: %s"), tenantDBO.ID, err.Error())
		return nil, errcode.TranslateDatabaseError(tenantDBO.ID, err)
	}
	stmtCloneRootCA := fmt.Sprintf(CloneTenantRootCAUnitTestTemplate, base.GetUUID(), tenantDBO.ID, cfssl.UnitTestTenantID)
	_, err = server.NamedExec(ctx, stmtCloneRootCA, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to clone a root CA for tenant ID %s. Error: %s"), tenantDBO.ID, err.Error())
		// Rollback by deleting the tenant.
		m := map[string]interface{}{}
		m["id"] = tenantDBO.ID
		_, err1 := server.Delete(ctx, "tenant_model", m)
		if err1 != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant ID %s while rolling back"), tenantDBO.ID)
		}
		return nil, errcode.TranslateDatabaseError(tenantDBO.ID, err)
	}
	response := &gapi.CreateTenantResponse{
		Id: tenantDBO.ID,
	}
	return response, nil
}

func (server *apiServer) createTenantNormal(ctx context.Context, request *gapi.CreateTenantRequest) (*gapi.CreateTenantResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "CreateTenant"}).Inc()
	authContext, _ := base.GetAuthContext(ctx)
	gTenant := request.GetTenant()
	tenantDBO := TenantDBO{}
	err := base.Convert(gTenant, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant data conversion error for tenant ID %s. Error: %s\n"), gTenant.GetId(), err.Error())
		return nil, err
	}
	if base.CheckID(tenantDBO.ID) {
		tenantDBO.ID = gTenant.GetId()
		glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant doc.ID was %s\n"), tenantDBO.ID)
	} else {
		tenantDBO.ID = base.GetUUID()
		glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant doc.ID was invalid, update it to %s\n"), tenantDBO.ID)
	}
	if authContext != nil {
		userID := authContext.GetUserID()
		if userID == "" {
			if authContext.TenantID == base.OperatorTenantID {
				// Operator tenant ID can call this API via the REST API.
				// User ID is mandatory to track the tenant
				glog.Errorf(base.PrefixRequestID(ctx, "User ID is required for %s"), base.OperatorTenantID)
				return nil, errcode.NewPermissionDeniedError("userID")
			}
		} else {
			tenantDBO.CreatedBy = base.StringPtr(userID)
		}
	}
	now := base.RoundedNow()
	tenantDBO.Version = float64(now.UnixNano())
	tenantDBO.CreatedAt = now
	tenantDBO.UpdatedAt = now

	if len(tenantDBO.Token) == 0 {
		token, err := server.GetKeyService().GenTenantToken()
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "CreateTenant tenant token creation failed for tenant %s\n"), tenantDBO.ID)
			return nil, errcode.NewInternalError(err.Error())
		}
		tenantDBO.Token = token.EncryptedToken
	}
	_, err = server.NamedExec(ctx, CreateTenant, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating tenant %s. Error: %s"), tenantDBO.ID, err.Error())
		return nil, errcode.TranslateDatabaseError(tenantDBO.ID, err)
	}
	// Create a root CA for the tenant in CFSSL.
	err = cfssl.CreateRootCA(tenantDBO.ID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create a root CA for tenant ID %s. Error: %s"), tenantDBO.ID, err.Error())
		// Rollback by deleting the tenant.
		m := map[string]interface{}{}
		m["id"] = tenantDBO.ID
		_, err1 := server.Delete(ctx, "tenant_model", m)
		if err1 != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant ID %s while rolling back"), tenantDBO.ID)
		}
		return nil, err
	}
	response := &gapi.CreateTenantResponse{
		Id: tenantDBO.ID,
	}
	return response, nil
}

func (server *apiServer) GetTenants(ctx context.Context, request *gapi.GetTenantsRequest) (*gapi.GetTenantsResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "GetTenants"}).Inc()
	tenants := []*gapi.Tenant{}
	startToken, rowSize := getPagingParams(request.GetPaging())
	param := TenantDBO{}
	err := base.Convert(request, &param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetTenants data conversion error. Error: %s\n"), err.Error())
		return nil, err
	}
	nextToken, err := server.PagedQuery(ctx, startToken, rowSize, func(dbObjPtr interface{}) error {
		tenant := &gapi.Tenant{}
		tenantDBO := dbObjPtr.(*TenantDBO)
		tenantDBO.DeletedAt = nil
		err := base.Convert(tenantDBO, tenant)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "GetTenants data conversion error. Error: %s\n"), err.Error())
			return err
		}
		tenants = append(tenants, tenant)
		return nil
	}, SelectTenants, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to query for tenants. Error: %s"), err.Error())
		return nil, err
	}
	response := &gapi.GetTenantsResponse{Tenants: tenants, Paging: &gapi.Paging{StartToken: string(nextToken), Size: uint32(rowSize)}}
	return response, nil
}

func (server *apiServer) UpdateTenant(ctx context.Context, request *gapi.UpdateTenantRequest) (*gapi.UpdateTenantResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "UpdateTenant"}).Inc()
	tenant := request.GetTenant()
	tenantDBO := TenantDBO{}
	err := base.Convert(tenant, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in data conversion for tenant %s. Error: %s"), tenant.GetId(), err.Error())
		return nil, err
	}
	now := base.RoundedNow()
	tenantDBO.Version = float64(now.UnixNano())
	tenantDBO.UpdatedAt = now
	_, err = server.NamedExec(ctx, UpdateTenant, &tenantDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in updating tenant %s. Error: %s"), tenant.GetId(), err.Error())
		return nil, errcode.TranslateDatabaseError(tenantDBO.ID, err)
	}
	response := &gapi.UpdateTenantResponse{Id: tenantDBO.ID}
	return response, nil
}

func (server *apiServer) DeleteTenant(ctx context.Context, request *gapi.DeleteTenantRequest) (*gapi.DeleteTenantResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "DeleteTenant"}).Inc()
	response := &gapi.DeleteTenantResponse{}
	authContext, _ := base.GetAuthContext(ctx)
	param := TenantDBO{ID: request.GetId()}
	if authContext != nil && authContext.TenantID == base.OperatorTenantID {
		// Operator tenant ID can call this API via the REST API.
		// User ID is mandatory to track the tenant
		userID := authContext.GetUserID()
		if userID == "" {
			return nil, errcode.NewPermissionDeniedError("userId")
		}
		tenantDBOs := []TenantDBO{}
		err := server.Query(ctx, &tenantDBOs, SelectTenants, param)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to query for tenants. Error: %s"), err.Error())
			return nil, err
		}
		if len(tenantDBOs) == 0 {
			response.Id = request.GetId()
			return response, nil
		}
		tenantDBO := tenantDBOs[0]
		if tenantDBO.CreatedBy == nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Unknown creator for tenant %s, but request is by %s"), tenantDBO.ID, userID)
			return nil, errcode.NewPermissionDeniedError("userId")
		} else if *tenantDBO.CreatedBy != userID {
			glog.Errorf(base.PrefixRequestID(ctx, "Tenant %s is created by %s, but request is by %s"), tenantDBO.ID, *tenantDBO.CreatedBy, userID)
			return nil, errcode.NewPermissionDeniedError("userId")
		}
		param.CreatedBy = base.StringPtr(userID)
		err = server.DoInTxn(func(tx *base.WrappedTx) error {
			result, err := tx.NamedExec(ctx, DeleteTenant, param)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant %s. Error: %s"), request.GetId(), err.Error())
				return err
			}
			if base.IsDeleteSuccessful(result) {
				response.Id = request.GetId()
			} else {
				glog.Warning(base.PrefixRequestID(ctx, "No rows get affected"))
			}
			return server.DisableGlobalEntities(ctx, tx, request.GetId())
		})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant %s. Error: %s"), request.GetId(), err.Error())
			return nil, err
		}
	} else {
		// Calls made internally by test cases come here as only operator tenant ID is allowed from outside the service.
		// This change is to make sure we do not leave behind test tenant records in the DB
		m := map[string]interface{}{}
		m["id"] = request.GetId()
		result, err := server.Delete(ctx, "tenant_model", m)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant %s. Error: %s"), request.GetId(), err.Error())
			return nil, err
		}
		if base.IsDeleteSuccessful(result) {
			response.Id = request.GetId()
		} else {
			glog.Warning(base.PrefixRequestID(ctx, "No rows get affected"))
		}
	}

	return response, nil
}

func (server *apiServer) DisableGlobalEntities(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	err := server.DeleteUsers(ctx, tx, tenantID)
	if err != nil {
		return err
	}
	// TODO revisit..it is directly accessing edge DB
	updatedAt := base.RoundedNow()
	unixTime := updatedAt.Unix()
	query := fmt.Sprintf(RenameNodeSerialNumberQuery, unixTime, tenantID)
	_, err = tx.NamedExec(ctx, query, struct{}{})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in renaming node serial numbers. Error: %s"), err.Error())
	}
	return err
}

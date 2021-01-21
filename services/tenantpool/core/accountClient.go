package core

import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/service"
	"context"
	"fmt"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const (
	// SoftDeletedEmailSuffix is the suffix for the deleted users
	SoftDeletedEmailSuffix = ".ntnx-del"
)

// PurgeUsers deletes all users under a tenant from account server
func PurgeUsers(ctx context.Context, tenantID string) error {
	adminCtx := ContextWithAdminRole(ctx, tenantID)
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUsers(adminCtx, &gapi.GetUsersRequest{})
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to get users for tenant %s. Error: %s"), tenantID, err.Error())
			return err
		}
		for _, gUser := range response.GetUsers() {
			_, err = client.DeleteUser(adminCtx, &gapi.DeleteUserRequest{Id: gUser.Id})
			if err != nil {
				glog.Error(base.PrefixRequestID(ctx, "Failed to delete user %s for tenant %s. Error: %s"), gUser.Email, tenantID, err.Error())
				return err
			}
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return err
}

// SoftPurgeUsers updates emails of users under a tenant to non-existing invalid ones
func SoftPurgeUsers(ctx context.Context, tenantID string) error {
	adminCtx := ContextWithAdminRole(ctx, tenantID)
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUsers(adminCtx, &gapi.GetUsersRequest{})
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to get users for tenant %s. Error: %s"), tenantID, err.Error())
			return err
		}
		for _, gUser := range response.GetUsers() {
			machineCtx := ContextWithAdminRole(ctx, base.MachineTenantID)
			// Set email to some non-existing one
			if !strings.HasSuffix(gUser.Email, SoftDeletedEmailSuffix) {
				// TODO better choice can be to have a different column.
				// It requires a lot more changes in other user, project APIs.
				// This will be revisited later if it can be an issue because of time constraint.
				// The epoch seconds time avoids unique field issue
				gUser.Email = fmt.Sprintf("%s.%d%s", gUser.Email, time.Now().Unix(), SoftDeletedEmailSuffix)
			}
			_, err = client.UpdateUser(machineCtx, &gapi.UpdateUserRequest{User: gUser})
			if err != nil {
				glog.Error(base.PrefixRequestID(ctx, "Failed to update user %s for tenant %s. Error: %s"), gUser.Email, tenantID, err.Error())
				return err
			}
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return err
}

// CreateTenant creates a tenant in account server
func CreateTenant(ctx context.Context, tenant *gapi.Tenant) (*gapi.Tenant, error) {
	request := &gapi.CreateTenantRequest{Tenant: tenant}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		adminCtx := ContextWithAdminRole(ctx, tenant.Id)
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.CreateTenant(adminCtx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenant. Error: %s"), err.Error())
			return err
		}
		tenant.Id = response.Id
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return tenant, err
}

// GetTenant gets tenant from account server
func GetTenant(ctx context.Context, id string) (*gapi.Tenant, error) {
	var tenant *gapi.Tenant
	request := &gapi.GetTenantsRequest{Id: id}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		adminCtx := ContextWithAdminRole(ctx, id)
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetTenants(adminCtx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenant %s. Error: %s"), id, err.Error())
			return err
		}
		if len(response.Tenants) != 1 {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to find tenant %s"), id)
			return errcode.NewRecordNotFoundError(id)
		}
		tenant = response.Tenants[0]
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return tenant, err
}

// DeleteTenantIfPossible deletes a tenant from account server if it is possible (no dependency).
// Otherwise, it updates the tenant such that it is not usable
func DeleteTenantIfPossible(ctx context.Context, id string) error {
	request := &gapi.DeleteTenantRequest{Id: id}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		_, err := client.DeleteTenant(ctx, request)
		if err != nil {
			// We are dealing with gRPC error. So, it needs translation
			if !errcode.IsDependencyConstraintError(service.ErrorCodeFromError(err)) {
				glog.Error(base.PrefixRequestID(ctx, "Failed to delete tenant %s. Error: %s"), id, err.Error())
				return err
			}
			response, err := client.GetTenants(ctx, &gapi.GetTenantsRequest{Id: id})
			if err != nil {
				glog.Error(base.PrefixRequestID(ctx, "Failed to get tenant %s for deletion. Error: %s"), id, err.Error())
				return err
			}
			if len(response.Tenants) != 1 {
				glog.Error(base.PrefixRequestID(ctx, "Failed to get tenant %s for deletion."), id)
				return errcode.NewBadRequestError(id)
			}
			tenant := response.Tenants[0]
			if len(tenant.ExternalId) > 0 {
				tenant.ExternalId = ""
				_, err = client.UpdateTenant(ctx, &gapi.UpdateTenantRequest{Tenant: tenant})
				if err != nil {
					glog.Error(base.PrefixRequestID(ctx, "Failed to update tenant %s to remove external ID. Error: %s"), id, err.Error())
					return err
				}
			}
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return err
}

// CreateUser creates an user in account server
func CreateUser(ctx context.Context, user *gapi.User) (*gapi.User, error) {
	request := &gapi.CreateUserRequest{User: user}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		adminCtx := ContextWithAdminRole(ctx, user.TenantId)
		client := gapi.NewAccountServiceClient(conn)
		resp, err := client.CreateUser(adminCtx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create user %s. Error: %s"), user.Email, err.Error())
			return err
		}
		user.Id = resp.Id
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return user, err
}

// GetUserByEmail gets the user object by email
func GetUserByEmail(ctx context.Context, email string) (*gapi.User, error) {
	var user *gapi.User
	request := &gapi.GetUserByEmailRequest{Email: email}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUserByEmail(ctx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get user by email %s. Error: %s"), email, err.Error())
			return err
		}
		user = response.User
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return user, err
}

// DeleteUserByEmail deletes an user from account server
func DeleteUserByEmail(ctx context.Context, email string) (*gapi.User, error) {
	var user *gapi.User
	request := &gapi.GetUserByEmailRequest{Email: email}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		adminCtx := ContextWithAdminRole(ctx, "")
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUserByEmail(ctx, request)
		if err != nil {
			errc := service.ErrorCodeFromError(err)
			if _, ok := errc.(*errcode.RecordNotFoundError); ok {
				return nil
			}
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in GetUserByEmail for email %s. Error: %s"), email, err.Error())
			return err
		}
		adminCtx = ContextWithAdminRole(ctx, response.User.TenantId)
		// Idempotent deletion
		_, err = client.DeleteUser(adminCtx, &gapi.DeleteUserRequest{Id: response.User.Id})
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to delete user %s. Error: %s"), email, err.Error())
			return err
		}
		user = response.User
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return user, err
}

// DeleteUser deletes an user from account server
func DeleteUser(ctx context.Context, tenantID, id string) error {
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		adminCtx := ContextWithAdminRole(ctx, tenantID)
		_, err := client.DeleteUser(adminCtx, &gapi.DeleteUserRequest{Id: id})
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to delete user %s for tenant %s. Error: %s"), id, tenantID, err.Error())
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return err
}

// ContextWithAdminRole creates a context with admin role with the tenant ID
func ContextWithAdminRole(ctx context.Context, tenantID string) context.Context {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	return context.WithValue(ctx, base.AuthContextKey, authContext)
}

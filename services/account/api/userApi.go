package api

import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/metrics"
	"cloudservices/common/model"
	"context"
	"os"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	SelectUsers           = `SELECT * FROM user_model WHERE (:email != '' AND email = :email) OR (tenant_id = :tenant_id AND (:id = '' OR id = :id))`
	SelectUsersByProjects = `SELECT * FROM user_model WHERE (email = :email OR (tenant_id = :tenant_id AND (:id = '' OR id = :id))) AND (id IN (SELECT user_id FROM project_user_model WHERE project_id IN (:project_ids)))`
	CreateUser            = `INSERT INTO user_model (id, version, tenant_id, name, email, password, role, created_at, updated_at) VALUES (:id, :version, :tenant_id, :name, :email, :password, :role, :created_at, :updated_at)`
	UpdateUser            = `UPDATE user_model SET version = :version, tenant_id = :tenant_id, name = :name, email = :email, password = :password, role = :role, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`
	SelectUserProjects    = `SELECT project_id, user_role from project_user_model where user_id = :user_id`
	DeleteUsers           = `DELETE FROM user_model WHERE tenant_id=:tenant_id`
)

// UserDBO is DB object model for user
type UserDBO struct {
	model.BaseModelDBO
	Email     string     `json:"email" db:"email"`
	Name      string     `json:"name" db:"name"`
	Password  string     `json:"password" db:"password"`
	Role      *string    `json:"role,omitempty" db:"role"`
	DeletedAt *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

type UserProjects struct {
	UserDBO
	ProjectIDs base.StringArray `json:"project_ids" db:"project_ids"`
}

func getUserQueryParam(ctx context.Context, id string, email string, projectID string) base.InQueryParam {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return base.InQueryParam{}
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	param := UserDBO{BaseModelDBO: tenantModel}
	if len(projectID) > 0 {
		return base.InQueryParam{
			Param: UserProjects{
				UserDBO:    param,
				ProjectIDs: []string{projectID},
			},
			Key:     SelectUsersByProjects,
			InQuery: true,
		}
	}
	if auth.IsInfraAdminRole(authContext) {
		return base.InQueryParam{
			Param:   param,
			Key:     SelectUsers,
			InQuery: false,
		}
	}
	projectIDs := auth.GetProjectIDs(authContext)
	if len(projectIDs) == 0 {
		cid, ok := authContext.Claims["id"].(string)
		if ok {
			if id == "" || id == cid {
				// allow user to update self
				return base.InQueryParam{
					Param: UserDBO{
						BaseModelDBO: model.BaseModelDBO{
							ID:       cid,
							TenantID: tenantID,
						},
					},
					Key: SelectUsers,
				}
			}
		}

		return base.InQueryParam{}
	}
	return base.InQueryParam{
		Param: UserProjects{
			UserDBO:    param,
			ProjectIDs: projectIDs,
		},
		Key:     SelectUsersByProjects,
		InQuery: true,
	}

}

// User APIs
func (server *apiServer) CreateUser(ctx context.Context, request *gapi.CreateUserRequest) (*gapi.CreateUserResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "CreateUser"}).Inc()
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	user := request.GetUser()
	tenantID := authContext.TenantID
	if tenantID != base.MachineTenantID && tenantID != base.OperatorTenantID && user.GetTenantId() != tenantID {
		glog.Errorf(base.PrefixRequestID(ctx, "CreateUser invalid tenant ID %s"), user.GetTenantId())
		return nil, errcode.NewBadRequestError("tenantID")
	}
	modelUser := &model.User{
		BaseModel: model.BaseModel{
			ID:       user.Id,
			Version:  user.Version,
			TenantID: user.TenantId,
		},
		Email:    user.Email,
		Name:     user.Name,
		Password: user.Password,
		Role:     user.Role,
	}
	err = model.ValidateUser(tenantID, modelUser)
	if err != nil {
		return nil, err
	}
	if tenantID == base.OperatorTenantID && tenantID != user.GetTenantId() {
		// Operator tenant is creating for the other tenant
		resp, err := server.GetTenants(ctx, &gapi.GetTenantsRequest{Id: user.GetTenantId()})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the tenant. Error: %s"), err.Error())
			return nil, err
		}
		if len(resp.Tenants) != 1 {
			return nil, errcode.NewRecordNotFoundError("tenantId")
		}
		tenant := resp.Tenants[0]
		if tenant.CreatedBy == "" || tenant.CreatedBy != authContext.GetUserID() {
			return nil, errcode.NewPermissionDeniedError("userId")
		}
	} else if tenantID != base.MachineTenantID {
		err = auth.CheckRBAC(
			authContext,
			meta.EntityUser,
			meta.OperationCreate,
			auth.RbacContext{})
		if err != nil {
			return nil, err
		}
	}
	userDBO := UserDBO{}
	err = base.Convert(user, &userDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in CreateUser. Error: %s"), err.Error())
		return nil, err
	}
	if base.CheckID(userDBO.ID) {
		glog.Infof(base.PrefixRequestID(ctx, "CreateUser user.ID was %s\n"), userDBO.ID)
	} else {
		userDBO.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(ctx, "CreateUser user.ID was invalid, update it to %s"), userDBO.ID)
	}
	password, err := crypto.EncryptPassword(userDBO.Password)
	if err != nil {
		return nil, err
	}
	now := base.RoundedNow()
	userDBO.Email = strings.ToLower(userDBO.Email)
	userDBO.Version = float64(now.UnixNano())
	userDBO.CreatedAt = now
	userDBO.UpdatedAt = now
	userDBO.Password = password
	_, err = server.NamedExec(ctx, CreateUser, &userDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating user with ID %s. Error: %s"), userDBO.ID, err.Error())
		return nil, errcode.TranslateDatabaseError(userDBO.ID, err)
	}
	response := &gapi.CreateUserResponse{Id: userDBO.ID}
	return response, nil
}

func (server *apiServer) GetUsers(ctx context.Context, request *gapi.GetUsersRequest) (*gapi.GetUsersResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "GetUsers"}).Inc()
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	if len(request.GetProjectId()) > 0 {
		if !auth.IsProjectMember(request.GetProjectId(), authContext) {
			glog.Errorf(base.PrefixRequestID(ctx, "RBAC error in GetUsers for request %+v"), request)
			return nil, errcode.NewPermissionDeniedError("RBAC")
		}
	}
	response := &gapi.GetUsersResponse{}
	queryParam := getUserQueryParam(ctx, request.GetId(), request.GetEmail(), request.GetProjectId())
	if queryParam.Key == "" {
		return response, nil
	}
	if queryParam.InQuery {
		userDBOs := []UserDBO{}
		err = server.QueryIn(ctx, &userDBOs, queryParam.Key, queryParam.Param)
		if err != nil {
			return nil, err
		}
		for _, userDBO := range userDBOs {
			user := &gapi.User{}
			err := base.Convert(&userDBO, user)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in GetUsers. Error: %s"), err.Error())
				return nil, err
			}
			response.Users = append(response.Users, user)
		}
	} else {
		nextToken := base.NilPageToken
		startToken, rowSize := getPagingParams(request.GetPaging())
		nextToken, err = server.PagedQueryEx(ctx, startToken, rowSize, func(dbObjPtr interface{}) error {
			user := &gapi.User{}
			err := base.Convert(dbObjPtr, user)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in GetUsers. Error: %s"), err.Error())
				return err
			}
			response.Users = append(response.Users, user)
			return nil
		}, queryParam.Key, queryParam.Param, UserDBO{})
		if nextToken != base.NilPageToken {
			response.Paging = &gapi.Paging{StartToken: string(nextToken), Size: uint32(rowSize)}
		}
	}
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to query user data in GetUsers. Error: %s"), err.Error())
		return nil, err
	}
	return response, err
}

func (server *apiServer) GetUserByEmail(ctx context.Context, request *gapi.GetUserByEmailRequest) (*gapi.GetUserByEmailResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "GetUserByEmail"}).Inc()
	email := request.GetEmail()
	if len(email) == 0 {
		glog.Error(base.PrefixRequestID(ctx, "Invalid email in GetUserByEmail"))
		return nil, errcode.NewBadRequestError("email")
	}
	userDBOs := []UserDBO{}
	param := UserDBO{Email: strings.ToLower(email)}
	err := server.Query(ctx, &userDBOs, SelectUsers, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to query user data in GetUserByEmail. Error: %s"), err.Error())
		return nil, err
	}
	if len(userDBOs) == 0 {
		return nil, errcode.NewRecordNotFoundError(email)
	}
	userDBOs[0].DeletedAt = nil
	gUser := &gapi.User{}
	err = base.Convert(&userDBOs[0], gUser)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in GetUserByEmail. Error: %s"), err.Error())
		return nil, err
	}
	response := &gapi.GetUserByEmailResponse{User: gUser}
	return response, err
}

func (server *apiServer) UpdateUser(ctx context.Context, request *gapi.UpdateUserRequest) (*gapi.UpdateUserResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "UpdateUser"}).Inc()
	response := &gapi.UpdateUserResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return response, err
	}
	user := request.GetUser()
	tenantID := authContext.TenantID
	if tenantID != base.MachineTenantID && user.GetTenantId() != tenantID {
		glog.Errorf(base.PrefixRequestID(ctx, "UpdateUser invalid tenant ID %s"), user.GetTenantId())
		return response, errcode.NewBadRequestError("tenantID")
	}
	modelUser := &model.User{
		BaseModel: model.BaseModel{
			ID:       user.Id,
			Version:  user.Version,
			TenantID: user.TenantId,
		},
		Email:    user.Email,
		Name:     user.Name,
		Password: user.Password,
		Role:     user.Role,
	}
	err = model.ValidateUser(tenantID, modelUser)
	if err != nil {
		return nil, err
	}
	if tenantID != base.MachineTenantID {
		err = auth.CheckRBAC(
			authContext,
			meta.EntityUser,
			meta.OperationUpdate,
			auth.RbacContext{
				ID:           user.GetId(),
				PrivilegedOp: user.Role == "INFRA_ADMIN",
			})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "RBAC error in UpdateUser. Error: %s"), err.Error())
			return response, err
		}
	}
	now := base.RoundedNow()
	userDBO := UserDBO{}
	err = base.Convert(user, &userDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in UpdateUser. Error: %s"), err.Error())
		return response, err
	}
	if !request.DontEncryptPassword && tenantID != base.MachineTenantID {
		password, err := crypto.EncryptPassword(userDBO.Password)
		if err != nil {
			return response, err
		}
		userDBO.Password = password
	}

	userDBO.Email = strings.ToLower(user.Email)
	userDBO.Version = float64(now.UnixNano())
	userDBO.UpdatedAt = now
	_, err = server.NamedExec(ctx, UpdateUser, &userDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in updating user for ID %s. Error: %s"), userDBO.ID, err.Error())
		return response, errcode.TranslateDatabaseError(userDBO.ID, err)
	}
	response.Id = userDBO.ID
	return response, nil
}

func (server *apiServer) DeleteUser(ctx context.Context, request *gapi.DeleteUserRequest) (*gapi.DeleteUserResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "DeleteUser"}).Inc()
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID
	m := map[string]interface{}{}
	if tenantID != base.MachineTenantID {
		err = auth.CheckRBAC(
			authContext,
			meta.EntityUser,
			meta.OperationDelete,
			auth.RbacContext{})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "RBAC error in DeleteUser. Error: %s"), err.Error())
			return nil, err
		}
		m["tenant_id"] = tenantID
	}
	m["id"] = request.GetId()
	result, err := server.Delete(ctx, "user_model", m)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete user with ID %s. Error: %s"), request.GetId(), err.Error())
		return nil, err
	}
	response := &gapi.DeleteUserResponse{}
	if base.IsDeleteSuccessful(result) {
		response.Id = request.GetId()
	} else {
		glog.Warning(base.PrefixRequestID(ctx, "No rows get affected"))
	}
	return response, nil
}

func (server *apiServer) GetUserProjectRoles(ctx context.Context, request *gapi.GetUserProjectRolesRequest) (*gapi.GetUserProjectRolesResponse, error) {
	metrics.GRPCCallCount.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME"), "method": "GetUserProjectRoles"}).Inc()
	type Param struct {
		UserID string `json:"userId" db:"user_id"`
	}
	response := &gapi.GetUserProjectRolesResponse{}
	param := Param{UserID: request.GetId()}
	startToken, rowSize := getPagingParams(request.GetPaging())
	nextToken, err := server.PagedQueryEx(ctx, startToken, rowSize, func(dbObjPtr interface{}) error {
		projectRole := &gapi.ProjectRole{}
		err := base.Convert(dbObjPtr, projectRole)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in GetUserProjectRoles. Error: %s"), err.Error())
			return err
		}
		response.ProjectRoles = append(response.ProjectRoles, projectRole)
		return nil
	}, SelectUserProjects, param, model.ProjectRole{})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to query in GetUserProjectRoles %s. Error: %s"), err.Error())
		return nil, err
	}
	if nextToken != base.NilPageToken {
		response.Paging = &gapi.Paging{StartToken: string(nextToken), Size: uint32(rowSize)}
	}
	return response, nil
}

func (server *apiServer) DeleteUsers(ctx context.Context, tx *base.WrappedTx, tenantID string) error {
	param := UserDBO{}
	param.TenantID = tenantID
	_, err := tx.NamedExec(ctx, DeleteUsers, param)
	return err
}

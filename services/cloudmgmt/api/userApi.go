package api

import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const entityTypeUser = "user"

// TODO FIXME - moving paging logic to account service
// paging logic is here for now since account service does not yet support
// random access paging (only next page)
func init() {
	queryMap["SelectUserIDsTemplate"] = `SELECT id from user_model where tenant_id = :tenant_id %s`
	queryMap["SelectUserIDsInProjectsTemplate"] = `SELECT id from user_model where tenant_id = :tenant_id AND (id IN (SELECT user_id FROM project_user_model WHERE project_id IN (:project_ids))) %s`

	orderByHelper.Setup(entityTypeUser, []string{"id", "version", "created_at", "updated_at", "name", "email", "role"})

}

// ** helper functions for paging
// get user ids - needed for paging support
func (dbAPI *dbObjectModelAPI) getUserIDs(context context.Context, projectIDs []string, queryParam *model.EntitiesQueryParam) ([]string, error) {
	if len(projectIDs) == 0 {
		query, err := buildQuery(entityTypeUser, queryMap["SelectUserIDsTemplate"], queryParam, orderByNameID)
		if err != nil {
			return nil, err
		}
		return dbAPI.selectEntityIDs(context, "", query)
	}
	query, err := buildQuery(entityTypeUser, queryMap["SelectUserIDsInProjectsTemplate"], queryParam, orderByNameID)
	if err != nil {
		return nil, err
	}
	return dbAPI.selectEntityIDs2(context, projectIDs, query)
}

// get paging related query info for user
func (dbAPI *dbObjectModelAPI) getUserListQueryInfo(context context.Context, projectIDs []string, queryParam *model.EntitiesQueryParam) (ListQueryInfo, error) {
	return dbAPI.getEntityListQueryInfo2(context, entityTypeUser, projectIDs, queryParam, dbAPI.getUserIDs)
}

type UserProjects struct {
	model.User
	ProjectIDs []string `json:"projectIds" db:"project_ids"`
}

// get DB query parameters for user
func getUserDBQueryParam(context context.Context, projectID string, id string) (base.InQueryParam, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return base.InQueryParam{}, err
	}

	tenantID := authContext.TenantID
	tenantModel := model.BaseModel{TenantID: tenantID, ID: id}
	param := model.User{BaseModel: tenantModel}
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return base.InQueryParam{}, errcode.NewPermissionDeniedError("RBAC")
		}
		return base.InQueryParam{
			Param: UserProjects{
				User:       param,
				ProjectIDs: []string{projectID},
			},
			Key:     "SelectUsersByProjects",
			InQuery: true,
		}, nil
	}
	if !auth.IsInfraAdminRole(authContext) {
		projectIDs := auth.GetProjectIDs(authContext)
		// let len(projectIDs) == 0 fall through to include self
		return base.InQueryParam{
			Param: UserProjects{
				User:       param,
				ProjectIDs: projectIDs,
			},
			Key:     "SelectUsersByProjects",
			InQuery: true,
		}, nil
	}
	return base.InQueryParam{
		Param:   param,
		Key:     "SelectUsers",
		InQuery: false,
	}, nil
}

func (dbAPI *dbObjectModelAPI) selectUsers(ctx context.Context, request *gapi.GetUsersRequest) ([]model.User, error) {
	users := []model.User{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUsers(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to get users. Error: %s"), err.Error())
			return err
		}
		for _, gUser := range response.GetUsers() {
			user := model.User{}
			err = base.Convert(gUser, &user)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data. Error: %s"), err.Error())
				return err
			}
			users = append(users, user)
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	return users, err
}

func (dbAPI *dbObjectModelAPI) selectUsersWV2(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	dbQueryParam, err := getUserDBQueryParam(ctx, projectID, "")
	if err != nil {
		return err
	}
	if dbQueryParam.Key == "" {
		return json.NewEncoder(w).Encode(model.UserListPayload{UserList: []model.User{}})
	}
	projectIDs := []string{}
	if dbQueryParam.InQuery {
		projectIDs = dbQueryParam.Param.(UserProjects).ProjectIDs
	}

	queryParam := model.GetEntitiesQueryParam(req)
	queryInfo, err := dbAPI.getUserListQueryInfo(ctx, projectIDs, queryParam)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)

	p := &gapi.Paging{StartToken: string(queryInfo.StartPage), Size: uint32(queryParam.PageSize)}
	request := &gapi.GetUsersRequest{Paging: p}

	users, err := dbAPI.selectUsers(ctx, request)
	if err != nil {
		return err
	}
	model.MaskUsers(users)
	r := model.UserListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		UserList:                  users,
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllUsers select all users for the given tenant
func (dbAPI *dbObjectModelAPI) SelectAllUsers(ctx context.Context) ([]model.User, error) {
	request := &gapi.GetUsersRequest{}
	return dbAPI.selectUsers(ctx, request)
}

// SelectAllUsersW select all users for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllUsersW(ctx context.Context, w io.Writer, req *http.Request) error {
	users, err := dbAPI.SelectAllUsers(ctx)
	if err != nil {
		return err
	}
	model.MaskUsers(users)
	return base.DispatchPayload(w, users)
}

// SelectAllUsersWV2 select all users for the given tenant, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllUsersWV2(ctx context.Context, w io.Writer, req *http.Request) error {
	return dbAPI.selectUsersWV2(ctx, "", w, req)
}

// SelectAllUsersForProject select all users for the given tenant + project
func (dbAPI *dbObjectModelAPI) SelectAllUsersForProject(ctx context.Context, projectID string) ([]model.User, error) {
	request := &gapi.GetUsersRequest{ProjectId: projectID}
	return dbAPI.selectUsers(ctx, request)
}

// SelectAllUsersForProjectW select all users for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllUsersForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	users, err := dbAPI.SelectAllUsersForProject(ctx, projectID)
	if err != nil {
		return err
	}
	model.MaskUsers(users)
	return base.DispatchPayload(w, users)
}

// SelectAllUsersForProjectWV2 select all users for the given tenant + project, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllUsersForProjectWV2(ctx context.Context, projectID string, w io.Writer, req *http.Request) error {
	return dbAPI.selectUsersWV2(ctx, projectID, w, req)
}

// GetUser get a user object in the DB
func (dbAPI *dbObjectModelAPI) GetUser(ctx context.Context, id string) (model.User, error) {
	user := model.User{}
	if len(id) == 0 {
		return user, errcode.NewBadRequestError("userID")
	}
	request := &gapi.GetUsersRequest{Id: id}
	users, err := dbAPI.selectUsers(ctx, request)
	if err != nil {
		return user, err
	}
	if len(users) == 0 {
		return user, errcode.NewRecordNotFoundError(id)
	}
	return users[0], err
}

// GetUserW get a user object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) GetUserW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	user, err := dbAPI.GetUser(ctx, id)
	if err != nil {
		return err
	}
	user.MaskObject()
	return base.DispatchPayload(w, user)
}

// GetUserByEmail get user by email
// for internal use only, so no auth / RBAC filter here
func (dbAPI *dbObjectModelAPI) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	user := model.User{}
	request := &gapi.GetUserByEmailRequest{Email: email}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUserByEmail(ctx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in GetUserByEmail for email %s. Error: %s"), email, err.Error())
			return err
		}
		err = base.Convert(response.GetUser(), &user)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in GetUserByEmail for email %s. Error: %s"), email, err.Error())
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
		return user, err
	}
	return user, err
}

// CreateUser creates a user object in the DB
func (dbAPI *dbObjectModelAPI) CreateUser(ctx context.Context, i interface{} /* *model.User */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.User)
	if !ok {
		return resp, errcode.NewInternalError("CreateUser: type error")
	}
	tenantID := authContext.TenantID
	if tenantID != base.MachineTenantID && tenantID != base.OperatorTenantID {
		p.TenantID = tenantID
	}

	gUser := &gapi.User{}
	err = base.Convert(p, gUser)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in CreateUser. Error: %s"), err.Error())
		return resp, err
	}
	request := &gapi.CreateUserRequest{User: gUser}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.CreateUser(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to get users. Error: %s"), err.Error())
			return err
		}
		resp.ID = response.GetId()
		return nil
	}
	err = service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	} else {
		GetAuditlogHandler().addUserAuditLog(dbAPI, ctx, *p, CREATE)
	}

	return resp, err
}

// CreateUserW creates a user object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateUserW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, dbAPI.CreateUser, &model.User{}, w, r, callback)
}

// CreateUserWV2 creates a user object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) CreateUserWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(context, model.ToCreateV2(dbAPI.CreateUser), &model.User{}, w, r, callback)
}

// UpdateUser update a user object in the DB
func (dbAPI *dbObjectModelAPI) UpdateUser(ctx context.Context, i interface{} /*model.User*/, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.User)
	if !ok {
		return resp, errcode.NewInternalError("UpdateUser: type error")
	}
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	if p.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	tenantID := authContext.TenantID
	if tenantID != base.MachineTenantID {
		p.TenantID = tenantID
	}
	// support partial user update
	dontEncryptPassword := false
	if p.HasMissingFields() {
		user, err := dbAPI.GetUser(ctx, p.ID)
		if err != nil {
			return resp, err
		}
		if p.Password == "" {
			dontEncryptPassword = true
		}
		p.FillInMissingFields(&user)
	}

	gUser := &gapi.User{}
	err = base.Convert(p, gUser)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in UpdateUser. Error: %s"), err.Error())
		return resp, err
	}
	request := &gapi.UpdateUserRequest{User: gUser, DontEncryptPassword: dontEncryptPassword}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.UpdateUser(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to get users. Error: %s"), err.Error())
			return err
		}
		resp.ID = response.GetId()
		return nil
	}
	err = service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	} else {
		GetAuditlogHandler().addUserAuditLog(dbAPI, ctx, *p, UPDATE)
	}
	return resp, err
}

// UpdateUserW update a user object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateUserW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateUser, &model.User{}, w, r, callback)
}

// UpdateUserWV2 update a user object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateUserWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateUser), &model.User{}, w, r, callback)
}

// DeleteUser delete a user object in the DB
func (dbAPI *dbObjectModelAPI) DeleteUser(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	user, errGetUser := dbAPI.GetUser(ctx, id)
	request := &gapi.DeleteUserRequest{Id: id}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.DeleteUser(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to delete user %s. Error: %s"), id, err.Error())
			return err
		}
		resp.ID = response.GetId()
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err == nil {
		if errGetUser != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), errGetUser.Error())
		} else {
			GetAuditlogHandler().addUserAuditLog(dbAPI, ctx, user, DELETE)
		}
	}
	return resp, err
}

// DeleteUserW delete a user object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteUserW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteUser, id, w, callback)
}

// DeleteUserWV2 delete a user object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteUserWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteUser), id, w, callback)
}

func (dbAPI *dbObjectModelAPI) GetUserProjectRoles(ctx context.Context, userID string) ([]model.ProjectRole, error) {
	projectRoles := []model.ProjectRole{}
	request := &gapi.GetUserProjectRolesRequest{Id: userID}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUserProjectRoles(ctx, request)
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to get project roles for user %s. Error: %s"), userID, err.Error())
			return err
		}
		for _, gProjectRole := range response.GetProjectRoles() {
			projectRole := model.ProjectRole{}
			err = base.Convert(gProjectRole, &projectRole)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert user data in GetUserProjectRoles for user %s. Error: %s"), userID, err.Error())
				return err
			}
			projectRoles = append(projectRoles, projectRole)
		}
		return nil
	}
	err := service.CallClient(ctx, service.AccountService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in account service call. Error: %s"), err.Error())
	}
	return projectRoles, err
}

func (dbAPI *dbObjectModelAPI) IsEmailAvailableW(context context.Context, w io.Writer, req *http.Request) error {
	var email string
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return errcode.NewPermissionDeniedError("RBAC")
	}
	if req != nil {
		query := req.URL.Query()
		emailVals := query["email"]
		if len(emailVals) == 1 {
			email = emailVals[0]
		}
	}
	if email == "" {
		return errcode.NewBadRequestError("email")
	}
	_, err = dbAPI.GetUserByEmail(context, email)
	if err != nil {
		// email is available if error is not found error
		if _, ok := err.(*errcode.RecordNotFoundError); ok {
			return json.NewEncoder(w).Encode(model.EmailAvailability{
				Email:     email,
				Available: true,
			})
		}
		return err
	}
	return json.NewEncoder(w).Encode(model.EmailAvailability{
		Email:     email,
		Available: false,
	})

}

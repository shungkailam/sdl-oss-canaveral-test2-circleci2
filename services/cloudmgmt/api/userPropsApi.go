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
	queryMap["SelectUserProps"] = `SELECT * FROM user_props_model WHERE tenant_id = :tenant_id AND user_id = :user_id`
	queryMap["CreateUserProps"] = `INSERT INTO user_props_model (tenant_id, user_id, props, version, created_at, updated_at) VALUES (:tenant_id, :user_id, :props, :version, :created_at, :updated_at) ON CONFLICT (user_id) DO UPDATE SET version = :version, props = :props, updated_at = :updated_at WHERE user_props_model.user_id = :user_id`

}

// make sure logged in user id matches payload or path user id
func validateUserID(context context.Context, id string) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	claims := authContext.Claims
	if claims == nil {
		errcode.NewBadRequestError("context")
	}
	userID, ok := claims["id"].(string)
	// user can only delete own user props
	if !ok || userID == "" || id != userID {
		return errcode.NewBadRequestError("userID")
	}
	return nil
}

type UserPropsDBO struct {
	TenantID string `json:"tenantId" db:"tenant_id"`
	UserID   string `json:"userId" db:"user_id"`
}

func (dbAPI *dbObjectModelAPI) GetUserProps(ctx context.Context, id string) (model.UserProps, error) {
	userProps := model.UserProps{}
	err := validateUserID(ctx, id)
	if err != nil {
		return userProps, err
	}
	authContext, _ := base.GetAuthContext(ctx)
	tenantID := authContext.TenantID
	param := model.UserProps{TenantID: tenantID, UserID: id}
	userPropsList := []model.UserProps{}
	err = dbAPI.Query(ctx, &userPropsList, queryMap["SelectUserProps"], param)
	if err != nil {
		return userProps, errcode.TranslateDatabaseError(id, err)
	}
	if len(userPropsList) == 1 {
		return userPropsList[0], nil
	}
	userProps.TenantID = tenantID
	userProps.UserID = id
	return userProps, nil
}

func (dbAPI *dbObjectModelAPI) GetUserPropsW(context context.Context, id string, w io.Writer, req *http.Request) error {
	userProps, err := dbAPI.GetUserProps(context, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, userProps)
}

// UpdateUserProps updates a user props object in the DB
func (dbAPI *dbObjectModelAPI) UpdateUserProps(ctx context.Context, i interface{} /* *model.UserProps */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	p, ok := i.(*model.UserProps)
	if !ok {
		return resp, errcode.NewInternalError("UpdateUserProps: type error")
	}
	var err error
	if p.UserID != "" {
		err = validateUserID(ctx, p.UserID)
		if err != nil {
			return resp, err
		}
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	if authContext.ID == "" {
		return resp, errcode.NewBadRequestError("ID")
	}
	if p.UserID != "" && p.UserID != authContext.ID {
		return resp, errcode.NewBadRequestError("userId")
	}
	p.UserID = authContext.ID
	if p.TenantID != "" && p.TenantID != authContext.TenantID {
		return resp, errcode.NewBadRequestError("tenantId")
	}
	p.TenantID = authContext.TenantID

	doc := *p
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now

	_, err = dbAPI.NamedExec(ctx, queryMap["CreateUserProps"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating user props for user ID %s and tenant ID %s. Error: %s"), doc.UserID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.UserID, err)
	}
	resp.ID = doc.UserID
	return resp, nil

}

// UpdateUserPropsW update a user props object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateUserPropsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateUserProps, &model.UserProps{}, w, r, callback)
}

// UpdateUserPropsWV2 update a user props object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) UpdateUserPropsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateUserProps), &model.UserProps{}, w, r, callback)
}

// DeleteUserProps delete a user props object in the DB
func (dbAPI *dbObjectModelAPI) DeleteUserProps(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	// default empty response
	resp := model.DeleteDocumentResponse{}
	err := validateUserID(context, id)
	if err != nil {
		// treat as not found
		return resp, nil
	}
	userProps, err := dbAPI.GetUserProps(context, id)
	if errcode.IsRecordNotFound(err) {
		return resp, nil
	} else if err != nil {
		return resp, err
	}
	return DeleteEntity(context, dbAPI, "user_props_model", "user_id", id, userProps, callback)
}

// DeleteUserPropsW delete a user props object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteUserPropsW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteUserProps, id, w, callback)
}

// DeleteUserPropsWV2 delete a user props object in the DB, write output into writer
// V2 response is of form {id}, as opposed to {_id}
func (dbAPI *dbObjectModelAPI) DeleteUserPropsWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteUserProps), id, w, callback)
}

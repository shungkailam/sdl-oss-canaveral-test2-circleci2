package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"io"
	"net/http"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

type userApiParam struct {
	UserID   string `db:"user_id"`
	TenantID string `db:"tenant_id"`
}
type userApiParam2 struct {
	ID       string `db:"id"`
	UserID   string `db:"user_id"`
	TenantID string `db:"tenant_id"`
}

func init() {
	queryMap["SelectUserApiTokens"] = `SELECT * from user_api_token_model WHERE (:user_id = '' OR user_id = :user_id) AND tenant_id = :tenant_id`
	queryMap["CreateUserApiToken"] = `INSERT INTO user_api_token_model (tenant_id, user_id, id, active, used_at, created_at, updated_at) VALUES (:tenant_id, :user_id, :id, :active, :used_at, :created_at, :updated_at)`
	queryMap["UpdateUserApiToken"] = `UPDATE user_api_token_model SET active = :active, updated_at = :updated_at WHERE id = :id AND user_id = :user_id AND tenant_id = :tenant_id`
	queryMap["UpdateUserApiTokenUsedTime"] = `UPDATE user_api_token_model SET used_at = :used_at WHERE id = :id AND user_id = :user_id AND tenant_id = :tenant_id`
	queryMap["DeleteUserApiToken"] = `DELETE FROM user_api_token_model WHERE id = :id AND user_id = :user_id AND tenant_id = :tenant_id`
	queryMap["GetUserApiToken"] = `SELECT * from user_api_token_model WHERE id = :id AND user_id = :user_id AND tenant_id = :tenant_id`
}

// SelectAllUserApiTokens get all API token objects in the DB for the current user or for all users if userID == ""
func (dbAPI *dbObjectModelAPI) SelectAllUserApiTokens(ctx context.Context, userID string) ([]model.UserApiToken, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	if userID == "" {
		if !auth.IsInfraAdminRole(authContext) {
			return nil, errcode.NewPermissionDeniedError("RBAC/userID")
		}
	} else {
		userID2, ok := authContext.Claims["id"].(string)
		if !ok || userID != userID2 {
			return nil, errcode.NewBadRequestError("userID")
		}
	}
	param := userApiParam{UserID: userID, TenantID: authContext.TenantID}
	userApiTokenList := []model.UserApiToken{}
	err = dbAPI.Query(ctx, &userApiTokenList, queryMap["SelectUserApiTokens"], param)
	if err != nil {
		return nil, err
	}
	return userApiTokenList, nil
}

// SelectAllUserApiTokensW get all API token objects in the DB for all users, write output into writer
func (dbAPI *dbObjectModelAPI) SelectAllUserApiTokensW(ctx context.Context, w io.Writer, _ *http.Request) error {
	userApiTokenList, err := dbAPI.SelectAllUserApiTokens(ctx, "")
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, userApiTokenList)
}

// GetUserApiTokensW get all API token objects in the DB for the current user, write output into writer
func (dbAPI *dbObjectModelAPI) GetUserApiTokensW(ctx context.Context, _ string, w io.Writer, req *http.Request) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	userID, ok := authContext.Claims["id"].(string)
	if !ok {
		return errcode.NewBadRequestError("userID")
	}
	userApiTokenList, err := dbAPI.SelectAllUserApiTokens(ctx, userID)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, userApiTokenList)
}

// CreateUserApiToken creates a user api token in the DB, response: UserApiTokenCreated
func (dbAPI *dbObjectModelAPI) CreateUserApiToken(ctx context.Context, i interface{} /* *model.UserApiToken */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UserApiTokenCreated{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.UserApiToken)
	if !ok {
		return resp, errcode.NewInternalError("CreateUserApiToken: type error")
	}
	userID, ok := authContext.Claims["id"].(string)
	if !ok {
		return resp, errcode.NewBadRequestError("userID")
	}
	user, err := dbAPI.GetUser(ctx, userID)
	if err != nil {
		return resp, err
	}
	// make sure user only has no more than one LL JWT
	tokens, err := dbAPI.SelectAllUserApiTokens(ctx, userID)
	if err != nil {
		return resp, err
	}
	if len(tokens) >= 2 {
		return resp, errcode.NewBadRequestError("max already reached")
	}

	p.UserID = userID
	p.TenantID = authContext.TenantID

	doc := *p
	now := base.RoundedNow()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	doc.ID = base.GetUUID()
	doc.Active = true

	_, err = dbAPI.NamedExec(ctx, queryMap["CreateUserApiToken"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating user props for user ID %s and tenant ID %s. Error: %s"), doc.UserID, doc.TenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.UserID, err)
	}

	// poor man's transaction:
	// make sure user only has no more than two LL JWT
	tokens, err = dbAPI.SelectAllUserApiTokens(ctx, userID)
	if err != nil {
		return resp, err
	}
	if len(tokens) > 2 {
		dbAPI.DeleteUserApiToken(ctx, doc.ID, nil)
		return resp, errcode.NewBadRequestError("max already reached 2")
	}
	resp.ID = doc.ID
	resp.TenantID = doc.TenantID
	resp.UserID = doc.UserID
	resp.Token = GetUserJWTToken(dbAPI, &user, nil, 0 /* never expires */, ApiKeyTokenType, jwt.MapClaims{
		"tokenId": doc.ID,
	})
	return resp, nil
}

// CreateUserApiTokenW update a user api token in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateUserApiTokenW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	doc := model.UserApiToken{}
	err := base.Decode(&r, &doc)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	resp, err := dbAPI.CreateUserApiToken(ctx, &doc, callback)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)
}

// DeleteUserApiToken delete user api token objects with the given tokenID in the DB
func (dbAPI *dbObjectModelAPI) DeleteUserApiToken(ctx context.Context, tokenID string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponseV2{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	userID, ok := authContext.Claims["id"].(string)
	if !ok {
		return resp, errcode.NewBadRequestError("userID")
	}
	doc := model.UserApiToken{
		TenantID: authContext.TenantID,
		UserID:   userID,
		ID:       tokenID,
	}
	res, err := dbAPI.NamedExec(ctx, queryMap["DeleteUserApiToken"], doc)
	if err != nil {
		return resp, err
	}
	if base.IsDeleteSuccessful(res) {
		resp.ID = tokenID
	}
	return resp, nil
}

// DeleteUserApiTokenW delete user api token objects with the given tokenID in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteUserApiTokenW(ctx context.Context, tokenID string, w io.Writer, callback func(context.Context, interface{}) error) error {
	resp, err := dbAPI.DeleteUserApiToken(ctx, tokenID, nil)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

// UpdateUserApiTokenUsedTime update user api token object used_at time in the DB - to track last usage
func (dbAPI *dbObjectModelAPI) UpdateUserApiTokenUsedTime(context context.Context) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	userID, ok := authContext.Claims["id"].(string)
	if !ok {
		return resp, errcode.NewBadRequestError("userID")
	}
	id, ok := authContext.Claims["tokenId"].(string)
	if !ok {
		return resp, errcode.NewBadRequestError("tokenId")
	}
	doc := model.UserApiToken{
		ID:       id,
		TenantID: tenantID,
		UserID:   userID,
		UsedAt:   base.RoundedNow(),
	}
	_, err = dbAPI.NamedExec(context, queryMap["UpdateUserApiTokenUsedTime"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating user api token for token ID %s user ID %s and tenant ID %s. Error: %s"), doc.ID, doc.UserID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateUserApiToken update a user api token object in the DB
func (dbAPI *dbObjectModelAPI) UpdateUserApiToken(context context.Context, i interface{} /* *model.UserApiToken */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.UserApiToken)
	if !ok {
		return resp, errcode.NewInternalError("UpdateUserApiToken: type error")
	}
	if p.UserID == "" {
		return resp, errcode.NewBadRequestError("userID")
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

	// RBAC: must be infra admin or self
	if !auth.IsInfraAdminRole(authContext) {
		userID, ok := authContext.Claims["id"].(string)
		if !ok {
			return resp, errcode.NewBadRequestError("userID")
		}
		if userID != doc.UserID {
			return resp, errcode.NewBadRequestError("userID")
		}
	}
	doc.UpdatedAt = base.RoundedNow()
	_, err = dbAPI.NamedExec(context, queryMap["UpdateUserApiToken"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating user api token for ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	return resp, nil
}

// UpdateUserApiTokenW update a user api token object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateUserApiTokenW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, model.ToUpdateV2(dbAPI.UpdateUserApiToken), &model.UserApiToken{}, w, r, callback)
}

// GetClaimsVerifier get claims verifier function - which will verify if a given long-lived JWT token is still valid or not
func (dbAPI *dbObjectModelAPI) GetClaimsVerifier() func(jwt.MapClaims) error {
	return func(claims jwt.MapClaims) (err error) {
		typ, ok := claims["type"].(string)
		if ok && typ == "api" {
			err = errcode.NewBadRequestError("apiKey")
			id, ok2 := claims["id"].(string)
			tenantID, ok3 := claims["tenantId"].(string)
			tokenID, ok4 := claims["tokenId"].(string)
			if !ok2 || !ok3 || !ok4 {
				glog.Errorf("ClaimsVerifier: api key required attribute(s) missing: id[%t], tenantId[%t], tokenId[%t]", ok2, ok3, ok4)
				return
			}
			param := userApiParam2{ID: tokenID, UserID: id, TenantID: tenantID}
			userApiTokenList := []model.UserApiToken{}
			authContext := &base.AuthContext{
				TenantID: tenantID,
				Claims: jwt.MapClaims{
					"id":       id,
					"tenantId": tenantID,
					"tokenId":  tokenID,
				},
			}
			ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
			err2 := dbAPI.Query(ctx, &userApiTokenList, queryMap["GetUserApiToken"], param)
			if err2 != nil || len(userApiTokenList) != 1 || !userApiTokenList[0].Active {
				glog.Errorf("ClaimsVerifier: api key not found or inactive id=%s", tokenID)
				return
			}
			// clear error
			err = nil
			go dbAPI.UpdateUserApiTokenUsedTime(ctx)
		}
		return
	}
}

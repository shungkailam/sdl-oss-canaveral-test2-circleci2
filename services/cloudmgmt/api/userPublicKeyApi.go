package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"

	jwt "github.com/dgrijalva/jwt-go"
)

type userIDParam struct {
	ID       string `db:"id"`
	TenantID string `db:"tenant_id"`
}

func init() {
	queryMap["SelectUserPublicKey"] = `SELECT * from user_public_key_model WHERE (:id = '' OR id = :id) AND tenant_id = :tenant_id`
	queryMap["UpdateUserPublicKey"] = `INSERT INTO user_public_key_model (tenant_id, id, public_key, used_at, created_at, updated_at) VALUES (:tenant_id, :id, :public_key, :used_at, :created_at, :updated_at) ON CONFLICT (id) DO UPDATE SET public_key = :public_key, updated_at = :updated_at WHERE user_public_key_model.id = :id`
	queryMap["UpdateUserPublicKeyUsedTime"] = `UPDATE user_public_key_model SET used_at = :used_at WHERE id = :id AND tenant_id = :tenant_id`
}

// SelectAllUserPublicKeys get all public keys from all users of the tenant
// caller must be infra admin for this to work
func (dbAPI *dbObjectModelAPI) SelectAllUserPublicKeys(ctx context.Context) ([]model.UserPublicKey, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return nil, errcode.NewPermissionDeniedError("RBAC/userID")
	}
	param := userIDParam{TenantID: authContext.TenantID}
	userPublicKeyList := []model.UserPublicKey{}
	err = dbAPI.Query(ctx, &userPublicKeyList, queryMap["SelectUserPublicKey"], param)
	if err != nil {
		return nil, err
	}
	return userPublicKeyList, nil
}

// SelectAllUserPublicKeys get all public keys from all users of the tenant, write output into writer
// caller must be infra admin for this to work
func (dbAPI *dbObjectModelAPI) SelectAllUserPublicKeysW(ctx context.Context, w io.Writer, _ *http.Request) error {
	userPublicKeyList, err := dbAPI.SelectAllUserPublicKeys(ctx)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, userPublicKeyList)
}

// GetUserPublicKey get public key for the current user
func (dbAPI *dbObjectModelAPI) GetUserPublicKey(ctx context.Context) (model.UserPublicKey, error) {
	resp := model.UserPublicKey{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	userID, ok := authContext.Claims["id"].(string)
	if !ok || userID == "" {
		return resp, errcode.NewBadRequestError("userID")
	}
	param := userIDParam{ID: userID, TenantID: authContext.TenantID}
	userPublicKeyList := []model.UserPublicKey{}
	err = dbAPI.Query(ctx, &userPublicKeyList, queryMap["SelectUserPublicKey"], param)
	if err != nil {
		return resp, err
	}
	if len(userPublicKeyList) == 0 {
		return resp, errcode.NewRecordNotFoundError(userID)
	}
	return userPublicKeyList[0], nil
}

// GetUserPublicKeyW get public key for the current user, write output into writer
func (dbAPI *dbObjectModelAPI) GetUserPublicKeyW(ctx context.Context, _ string, w io.Writer, _ *http.Request) error {
	key, err := dbAPI.GetUserPublicKey(ctx)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, key)
}

// UpdateUserPublicKey creates a user public key object in the DB
// No CreateUserPublicKey because UpdateUserPublicKey performs upsert (Update/Insert)
func (dbAPI *dbObjectModelAPI) UpdateUserPublicKey(context context.Context, i interface{} /* *model.UserPublicKey */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.UserPublicKey)
	if !ok {
		return resp, errcode.NewInternalError("UpdateUserPublicKey: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	id, ok := authContext.Claims["id"].(string)
	if !ok {
		return resp, errcode.NewBadRequestError("userID")
	}
	doc.ID = id
	now := base.RoundedNow()
	doc.CreatedAt = now
	doc.UpdatedAt = now
	_, err = dbAPI.NamedExec(context, queryMap["UpdateUserPublicKey"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating user public key for user ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	if callback != nil {
		go callback(context, doc)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addAPIKeyAuditLog(dbAPI, context, doc, UPDATE)
	return resp, nil
}

// UpdateUserPublicKeyW updates a user public key object in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateUserPublicKeyW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(context, dbAPI.UpdateUserPublicKey, &model.UserPublicKey{}, w, r, callback)
}

// DeleteUserPublicKey delete user public key objects with the given userID in the DB
func (dbAPI *dbObjectModelAPI) DeleteUserPublicKey(context context.Context, userID string, callback func(context.Context, interface{}) error) (interface{}, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return model.DeleteDocumentResponse{}, err
	}
	userPublicKey, errGetUserPublicKey := dbAPI.GetUserPublicKey(context)
	doc := model.UserPublicKey{
		TenantID: authContext.TenantID,
		ID:       userID,
	}
	result, err := DeleteEntity(context, dbAPI, "user_public_key_model", "id", userID, doc, callback)
	if err == nil {
		if errGetUserPublicKey != nil {
			glog.Error("Error in getting user public key : ", errGetUserPublicKey.Error())
		} else {
			GetAuditlogHandler().addAPIKeyAuditLog(dbAPI, context, userPublicKey, DELETE)
		}
	}
	return result, err
}

// DeleteUserPublicKeyW delete user public key objects with the given userID in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) DeleteUserPublicKeyW(context context.Context, _ string, w io.Writer, callback func(context.Context, interface{}) error) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	userID, ok := authContext.Claims["id"].(string)
	if !ok {
		return errcode.NewBadRequestError("userID")
	}
	return base.DeleteW(context, model.ToDeleteV2(dbAPI.DeleteUserPublicKey), userID, w, callback)
}

// UpdateUserPublicKeyUsedTime update user public key object used_at time in the DB - to track last usage
func (dbAPI *dbObjectModelAPI) UpdateUserPublicKeyUsedTime(context context.Context, i interface{} /* *model.UserPublicKey */) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.UserPublicKey)
	if !ok {
		return resp, errcode.NewInternalError("UpdateUserPublicKeyUsedTime: type error")
	}
	doc := *p
	tenantID := authContext.TenantID
	doc.TenantID = tenantID
	id, ok := authContext.Claims["id"].(string)
	if !ok {
		return resp, errcode.NewBadRequestError("userID")
	}
	doc.ID = id
	now := base.RoundedNow()
	doc.UsedAt = now
	_, err = dbAPI.NamedExec(context, queryMap["UpdateUserPublicKeyUsedTime"], &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating user public key for user ID %s and tenant ID %s. Error: %s"), doc.ID, tenantID, err.Error())
		return resp, errcode.TranslateDatabaseError(doc.ID, err)
	}
	resp.ID = doc.ID
	return resp, nil
}

// GetPublicKeyResolver return a resolver function to resolve current user's public key
func (dbAPI *dbObjectModelAPI) GetPublicKeyResolver() func(*jwt.Token) (interface{}, error) {
	return func(token *jwt.Token) (verifyKey interface{}, err error) {
		var err2 error
		_, isRSA := token.Method.(*jwt.SigningMethodRSA)
		_, isECDSA := token.Method.(*jwt.SigningMethodECDSA)
		claims, ok := token.Claims.(jwt.MapClaims)
		err = fmt.Errorf("Error: public key resolver: Failed to resolve public key")
		if !ok || (!isRSA && !isECDSA) {
			return
		}
		id, ok := claims["id"].(string)
		if !ok {
			glog.Errorln("Error: public key resolver: id missing in JWT token")
			return
		}
		specialRole, ok := claims["specialRole"].(string)
		if !ok {
			glog.Errorln("Error: public key resolver: specialRole missing in JWT token")
			return
		}
		tenantID, ok := claims["tenantId"].(string)
		if !ok {
			glog.Errorln("Error: public key resolver: tenantId missing in JWT token")
			return
		}
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims:   claims,
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		user, err2 := dbAPI.GetUser(ctx, id)
		if err2 != nil {
			glog.Errorf("Error: public key resolver: failed to get user with id: %s, error: %s", id, err2.Error())
			return
		}
		if specialRole != model.GetUserSpecialRole(&user) {
			glog.Errorln("Error: public key resolver: specialRole mismatch in JWT token")
			return
		}
		userPublicKey, err2 := dbAPI.GetUserPublicKey(ctx)
		if err2 != nil {
			glog.Errorf("Error: public key resolver: failed to get user public key with id: %s, error: %s", id, err2.Error())
			return
		}
		verifyBytes := []byte(userPublicKey.PublicKey)
		updatePublicKeyUsedTimestamp := func() (interface{}, error) {
			return dbAPI.UpdateUserPublicKeyUsedTime(ctx, &userPublicKey)
		}
		if isRSA {
			verifyKey, err2 = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
			if err2 != nil {
				glog.Errorf("Error: public key resolver: failed to parse user RSA public key with id: %s, error: %s", id, err2.Error())
				return
			}
		} else {
			verifyKey, err2 = jwt.ParseECPublicKeyFromPEM(verifyBytes)
			if err2 != nil {
				glog.Errorf("Error: public key resolver: failed to parse user ECDSA public key with id: %s, error: %s", id, err2.Error())
				return
			}
		}
		// async update public key used_at timestamp
		go updatePublicKeyUsedTimestamp()
		// clear error
		err = nil
		return

	}
}

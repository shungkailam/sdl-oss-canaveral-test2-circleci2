package api_test

import (
	"bytes"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestUserApiToken(t *testing.T) {
	t.Parallel()

	t.Log("running TestUserApiToken test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	user := createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")
	userId := user.ID
	user2 := createUserWithRole(t, dbAPI, tenantID, "USER")
	user3 := createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteUserApiToken", func(t *testing.T) {
		t.Log("running Create/Get/DeleteUserApiToken test")

		// set context to correspond to user
		// since only user can get/update/delete own public keys
		authContext, err := base.GetAuthContext(ctx1)
		require.NoError(t, err)
		claims := authContext.Claims
		claims["id"] = userId

		// create api token
		users := []model.User{user, user2, user3}
		userCtxs := []context.Context{makeUserContext(user), makeUserContext(user2), makeUserContext(user3)}
		apiTokens := []model.UserApiTokenCreated{}
		for i, u := range users {
			uid := u.ID
			uctx := userCtxs[i]
			doc := model.UserApiToken{UserID: uid, TenantID: tenantID, Active: true}
			t.Logf("about to create user api token, ctx=%+v, doc=%+v", uctx, doc)
			r, err := dbAPI.CreateUserApiToken(uctx, &doc, nil)
			require.NoError(t, err)
			apiTokens = append(apiTokens, r.(model.UserApiTokenCreated))
			t.Logf("create user[%s] first api token got response: %+v", uid, r)

			r, err = dbAPI.CreateUserApiToken(uctx, &doc, nil)
			require.NoError(t, err)
			apiTokens = append(apiTokens, r.(model.UserApiTokenCreated))
			t.Logf("create user[%s] second api token got response: %+v", uid, r)

			r, err = dbAPI.CreateUserApiToken(uctx, &doc, nil)
			require.Errorf(t, err, "create user[%s] 3rd api token must fail", uid)

			tokens, err := dbAPI.SelectAllUserApiTokens(uctx, uid)
			require.NoError(t, err)
			if len(tokens) != 2 {
				t.Fatalf("expect user[%s] api tokens count to be 2", uid)
			}
			t.Logf("Got user api tokens: %+v", tokens)

		}

		apiCtx := makeUserApiContext(users[0], apiTokens[0].ID)
		ur, err := dbAPI.UpdateUserApiTokenUsedTime(apiCtx)
		require.NoError(t, err)
		t.Logf("update api token used at time response: %+v", ur)

		// infra admin can update self api token
		tk := model.UserApiToken{
			ID:       apiTokens[0].ID,
			TenantID: apiTokens[0].TenantID,
			UserID:   apiTokens[0].UserID,
			Active:   false,
		}
		ur2, err := dbAPI.UpdateUserApiToken(userCtxs[0], &tk, nil)
		require.NoError(t, err)
		t.Logf("admin update self api token active flag response: %+v", ur2)

		// infra admin can update user api token
		tk2 := model.UserApiToken{
			ID:       apiTokens[2].ID,
			TenantID: apiTokens[2].TenantID,
			UserID:   apiTokens[2].UserID,
			Active:   false,
		}
		ur2, err = dbAPI.UpdateUserApiToken(userCtxs[0], &tk2, nil)
		require.NoError(t, err)
		t.Logf("admin update user api token active flag response: %+v", ur2)

		// non infra admin user can update self api token
		tk2.Active = true
		ur2, err = dbAPI.UpdateUserApiToken(userCtxs[1], &tk2, nil)
		require.NoError(t, err)
		t.Logf("user update self api token active flag response: %+v", ur2)

		// non infra admin user can not update other api token
		tk.Active = true
		ur2, err = dbAPI.UpdateUserApiToken(userCtxs[1], &tk, nil)
		require.Error(t, err, "user should not be able to update other user api token")

		// infra admin can get all user tokens metadata
		tokens, err := dbAPI.SelectAllUserApiTokens(userCtxs[0], "")
		require.NoError(t, err)
		if len(tokens) != 6 {
			t.Fatal("expect api tokens count to be 6")
		}
		// even infra admin can only get self tokens if id is specified
		tokens, err = dbAPI.SelectAllUserApiTokens(userCtxs[0], users[1].ID)
		require.Error(t, err, "expect infra admin not able to get tokens when user id specified is not self")
		// non infra admin should not be able to get tokens without specify id
		tokens, err = dbAPI.SelectAllUserApiTokens(userCtxs[1], "")
		require.Error(t, err, "expect non infra admin not able to get tokens without specifying user id")
		// non infra admin can get self tokens
		tokens, err = dbAPI.SelectAllUserApiTokens(userCtxs[1], users[1].ID)
		require.NoError(t, err)

		for i, u := range users {
			uid := u.ID
			uctx := userCtxs[i]
			tokens, err := dbAPI.SelectAllUserApiTokens(uctx, uid)
			require.NoError(t, err)
			for _, token := range tokens {
				r2, err := dbAPI.DeleteUserApiToken(uctx, token.ID, nil)
				require.NoError(t, err)
				t.Logf("Delete user api token[%s] response: %+v", token.ID, r2)
			}
		}
	})
}

func TestUserApiTokenW(t *testing.T) {
	t.Parallel()

	t.Log("running TestUserApiTokenW test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	user := createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")
	userId := user.ID
	// user2 := createUserWithRole(t, dbAPI, tenantID, "USER")
	// user3 := createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteUserApiTokenW", func(t *testing.T) {
		t.Log("running Create/Get/DeleteUserApiTokenW test")

		// set context to correspond to user
		// since only user can get/update/delete own public keys
		authContext, err := base.GetAuthContext(ctx1)
		require.NoError(t, err)
		claims := authContext.Claims
		claims["id"] = userId

		// create api token
		userCtx := makeUserContext(user)
		doc := model.UserApiToken{UserID: userId, TenantID: tenantID, Active: true}
		reader, err := objToReader(doc)
		require.NoError(t, err)
		var w bytes.Buffer
		err = dbAPI.CreateUserApiTokenW(userCtx, &w, reader, nil)
		require.NoError(t, err)
		resp := model.UserApiTokenCreated{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)
		t.Log("create user api token got response: ", resp)
		doc.ID = resp.ID

		err = dbAPI.SelectAllUserApiTokensW(userCtx, &w, nil)
		require.NoError(t, err)
		tokens := []model.UserApiToken{}
		err = json.NewDecoder(&w).Decode(&tokens)
		require.NoError(t, err)
		t.Logf("Got user api tokens: %+v", tokens)
		if len(tokens) != 1 {
			t.Fatal("expect user api tokens count to be 1")
		}
		if !tokens[0].Active {
			t.Fatal("expect user api token to be active")
		}

		// now do update:
		doc.Active = false
		reader, err = objToReader(doc)
		require.NoError(t, err)
		err = dbAPI.UpdateUserApiTokenW(ctx1, &w, reader, nil)
		require.NoError(t, err)
		uresp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&uresp)
		require.NoError(t, err)
		t.Log("update user api token got response: ", uresp)

		err = dbAPI.SelectAllUserApiTokensW(userCtx, &w, nil)
		require.NoError(t, err)
		tokens = []model.UserApiToken{}
		err = json.NewDecoder(&w).Decode(&tokens)
		require.NoError(t, err)
		t.Logf("Got user api tokens: %+v", tokens)
		if len(tokens) != 1 {
			t.Fatal("expect user api tokens count to be 1")
		}
		if tokens[0].Active {
			t.Fatal("expect user api token to be inactive")
		}

		err = dbAPI.DeleteUserApiTokenW(ctx1, tokens[0].ID, &w, nil)
		require.NoError(t, err)

		dresp := model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&dresp)
		require.NoError(t, err)

		t.Log("Delete user api token response:", dresp)

		err = dbAPI.SelectAllUserApiTokensW(userCtx, &w, nil)
		require.NoError(t, err)
		tokens = []model.UserApiToken{}
		err = json.NewDecoder(&w).Decode(&tokens)
		require.NoError(t, err)
		t.Logf("Got user api tokens: %+v", tokens)
		if len(tokens) != 0 {
			t.Fatal("expect user api tokens count to be 0 after delete")
		}
	})
}

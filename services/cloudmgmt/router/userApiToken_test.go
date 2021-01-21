package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"

	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	USER_API_TOKEN_PATH     = "/v1.0/userapitokens"
	ALL_USER_API_TOKEN_PATH = "/v1.0/userapitokensall"
)

// create user api token
func createUserApiToken(netClient *http.Client, userApiToken model.UserApiToken, token string) (model.UserApiTokenCreated, string, error) {
	resp := model.UserApiTokenCreated{}
	fmt.Printf("Calling POST on %s", USER_API_TOKEN_PATH)
	fmt.Println()
	reqID, err := doPost(netClient, USER_API_TOKEN_PATH, token, userApiToken, &resp)
	return resp, reqID, err
}

// update user api token
func updateUserApiToken(netClient *http.Client, userApiToken model.UserApiToken, token string) (model.UpdateDocumentResponseV2, string, error) {
	fmt.Printf("updateUserApiToken: path=%s\n", fmt.Sprintf("%s/%s", USER_API_TOKEN_PATH, userApiToken.ID))
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", USER_API_TOKEN_PATH, userApiToken.ID), userApiToken, token)
}

// delete user api token
func deleteUserApiToken(netClient *http.Client, token, tokenID string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, USER_API_TOKEN_PATH, tokenID, token)
}

// get current user api tokens
func getUserApiTokens(netClient *http.Client, token string) ([]model.UserApiToken, error) {
	userApiTokens := []model.UserApiToken{}
	err := doGet(netClient, USER_API_TOKEN_PATH, token, &userApiTokens)
	return userApiTokens, err
}

// get all user api tokens
func getAllUserApiTokens(netClient *http.Client, token string) ([]model.UserApiToken, error) {
	userApiTokens := []model.UserApiToken{}
	err := doGet(netClient, ALL_USER_API_TOKEN_PATH, token, &userApiTokens)
	return userApiTokens, err
}

func TestUserApiToken(t *testing.T) {
	t.Parallel()
	t.Log("running TestUserApiToken test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user2 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test User API token", func(t *testing.T) {
		// login as user to get token

		token := loginUser(t, netClient, user)
		token2 := loginUser(t, netClient, user2)

		users := []model.User{user, user2}
		jwtTokens := []string{token, token2}
		createdApiTokens := []model.UserApiTokenCreated{}
		for i, u := range users {
			token := jwtTokens[i]
			doc := model.UserApiToken{UserID: u.ID, TenantID: tenantID, Active: true}
			resp, _, err := createUserApiToken(netClient, doc, token)
			require.NoError(t, err)
			t.Logf("create user[%s] api token successful: %+v", u.ID, resp)
			createdApiTokens = append(createdApiTokens, resp)
		}

		// api token can now be used for API call
		for _, ct := range createdApiTokens {
			_, err := getUsers(netClient, ct.Token)
			require.NoError(t, err)
		}

		// get user api token
		for i, u := range users {
			token := jwtTokens[i]
			tokens, err := getUserApiTokens(netClient, token)
			require.NoError(t, err)
			t.Logf("Got user[%s] api tokens: %+v", u.ID, tokens)
			if len(tokens) != 1 {
				t.Fatal("expect api tokens count to be 1")
			}
			if !tokens[0].Active {
				t.Fatal("expect api token to be active")
			}
		}

		// get all user api token
		// infra admin
		allTokens, err := getAllUserApiTokens(netClient, token)
		require.NoError(t, err)
		t.Logf("Got all user api tokens: %+v", allTokens)
		if len(allTokens) != 2 {
			t.Fatal("expect all tokens count to be 2")
		}
		for _, tkn := range allTokens {
			if !tkn.Active {
				t.Fatal("expect api token to be active")
			}
		}
		// non infra admin
		allTokens, err = getAllUserApiTokens(netClient, token2)
		require.Error(t, err, "expect get all user api tokens to fail for non infra admin")

		// update to deactivate token
		for i, u := range users {
			token := jwtTokens[i]
			tkn := createdApiTokens[i]
			d := model.UserApiToken{
				ID:       tkn.ID,
				TenantID: tkn.TenantID,
				UserID:   tkn.UserID,
				Active:   false,
			}
			ur, _, err := updateUserApiToken(netClient, d, token)
			require.NoError(t, err)
			t.Logf("update user api token[%d] response: %+v", i, ur)
			tokens, err := getUserApiTokens(netClient, token)
			require.NoError(t, err)
			t.Logf("Got user[%s] api tokens: %+v", u.ID, tokens)
			if len(tokens) != 1 {
				t.Fatal("expect api tokens count to be 1")
			}
			if tokens[0].Active {
				t.Fatal("expect api token to be inactive")
			}

		}

		// now api token inactive and can't make API call any more
		for _, ct := range createdApiTokens {
			_, err := getUsers(netClient, ct.Token)
			require.Errorf(t, err, "expect API call to fail with inactive api token %s", ct.ID)
		}

		// infra admin can update all tokens
		if true {
			token := jwtTokens[0]
			jwt2 := jwtTokens[1]
			tkn := createdApiTokens[1]
			d := model.UserApiToken{
				ID:       tkn.ID,
				TenantID: tkn.TenantID,
				UserID:   tkn.UserID,
				Active:   true,
			}
			_, _, err := updateUserApiToken(netClient, d, token)
			require.NoError(t, err)
			tokens, err := getUserApiTokens(netClient, jwt2)
			require.NoError(t, err)
			if len(tokens) != 1 {
				t.Fatal("expect api tokens count to be 1")
			}
			if !tokens[0].Active {
				t.Fatal("expect api token to be active")
			}
		}

		// non infra admin can not update others token
		if true {
			jwt2 := jwtTokens[1]
			tkn := createdApiTokens[0]
			d := model.UserApiToken{
				ID:       tkn.ID,
				TenantID: tkn.TenantID,
				UserID:   tkn.UserID,
				Active:   true,
			}
			_, _, err := updateUserApiToken(netClient, d, jwt2)
			require.Error(t, err, "non infra admin should not be able to update other token")
		}

		// only self can delete api token
		// first negative test cases:
		for i, token := range jwtTokens {
			dr, _, err := deleteUserApiToken(netClient, token, createdApiTokens[1-i].ID)
			if err == nil && dr.ID != "" {
				t.Fatal("expect delete token by non-self to fail")
			}
		}
		for i := range users {
			token := jwtTokens[i]
			tokens, err := getUserApiTokens(netClient, token)
			require.NoError(t, err)
			if len(tokens) != 1 {
				t.Fatal("expect api tokens count to be 1")
			}
		}
		// next, positive delete test cases:
		for i, token := range jwtTokens {
			_, _, err := deleteUserApiToken(netClient, token, createdApiTokens[i].ID)
			require.NoError(t, err)
		}
		// now get api tokens should return nothing
		for i := range users {
			token := jwtTokens[i]
			tokens, err := getUserApiTokens(netClient, token)
			require.NoError(t, err)
			if len(tokens) != 0 {
				t.Fatal("expect api tokens count to be 0")
			}
		}

		// deleted api token can't make API call any more
		for _, ct := range createdApiTokens {
			_, err := getUsers(netClient, ct.Token)
			require.Errorf(t, err, "expect API call to fail with deleted api token %s", ct.ID)
		}
	})
}

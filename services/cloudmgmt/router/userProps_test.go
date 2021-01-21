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

	"github.com/jmoiron/sqlx/types"
)

const (
	USER_PROPS_PATH = "/v1/userprops"
)

// update user props
func updateUserProps(netClient *http.Client, userProps model.UserProps, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", USER_PROPS_PATH, userProps.UserID), userProps, token)
}

// delete user props
func deleteUserProps(netClient *http.Client, userID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, USER_PROPS_PATH, userID, token)
}

// get user props by id
func getUserPropsByID(netClient *http.Client, userID string, token string) (model.UserProps, error) {
	userProps := model.UserProps{}
	err := doGet(netClient, USER_PROPS_PATH+"/"+userID, token, &userProps)
	return userProps, err
}

func TestUserProps(t *testing.T) {
	t.Parallel()
	t.Log("running TestUserProps test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	t.Logf("created user: %+v", user)

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
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test User Props", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		userProps, err := getUserPropsByID(netClient, user.ID, token)
		require.NoError(t, err)
		t.Logf("got user props: %+v", userProps)

		jt := types.JSONText{}
		ms := map[string]string{}
		ms["foo"] = "bar"
		ms["baz"] = "bam"
		err = base.Convert(&ms, &jt)
		require.NoError(t, err)

		up := model.UserProps{
			TenantID: user.TenantID,
			UserID:   user.ID,
			Props:    jt,
		}
		resp, _, err := updateUserProps(netClient, up, token)
		require.NoError(t, err)
		t.Logf("update user props got response: %+v", resp)

		userProps, err = getUserPropsByID(netClient, user.ID, token)
		require.NoError(t, err)
		t.Logf("got user props 2: %+v", userProps)

		res, _, err := deleteUserProps(netClient, user.ID, token)
		require.NoError(t, err)
		t.Logf("delete user props got response: %+v", res)

		userProps, err = getUserPropsByID(netClient, user.ID, token)
		require.NoError(t, err)
		t.Logf("got user props 3: %+v", userProps)

	})

}

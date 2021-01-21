package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func isPermissionDeniedError(t *testing.T, err error) bool {
	b := strings.Contains(err.Error(), "Permission denied")
	t.Logf(">>> isPermissionDeniedError? err=%s\n", err.Error())
	return b
}

func TestK8sDashboard(t *testing.T) {
	t.Parallel()
	t.Log("running TestK8sDashboard test")

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
	user3 := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user4 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user4.ID, nil)
		dbAPI.DeleteUser(ctx, user3.ID, nil)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Edge", func(t *testing.T) {
		// login as user to get token

		token := loginUser(t, netClient, user)
		token2 := loginUser(t, netClient, user2)
		token3 := loginUser(t, netClient, user3)
		token4 := loginUser(t, netClient, user4)

		t.Logf("%s", token2)
		t.Logf("%s", token3)
		t.Logf("%s", token4)

		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID
		t.Logf("edge created: %+v, id: %s", edge, edgeID)

		getPath := fmt.Sprintf("/v1.0/k8sdashboard/%s/viewonlyUsers", edgeID)
		addPath := fmt.Sprintf("/v1.0/k8sdashboard/%s/viewonlyUsersAdd", edgeID)
		removePath := fmt.Sprintf("/v1.0/k8sdashboard/%s/viewonlyUsersRemove", edgeID)

		getResp := model.K8sDashboardViewonlyUserListPayload{}

		// get viewonly users
		err = doGet(netClient, getPath, token, &getResp)
		require.NoError(t, err)
		require.Equal(t, 0, len(getResp.ViewonlyUserList), "expect viewonly user count to match")

		// expect call get k8s dashboard viewonly token to fail for user2
		// getTokenPath := fmt.Sprintf("/v1.0/k8sdashboard/%s/viewonlyToken", edgeID)
		// getAdminTokenPath := fmt.Sprintf("/v1.0/k8sdashboard/%s/adminToken", edgeID)

		// TODO FIXME - can't really test this fully
		// since it requires calling SD to get token
		// admin user can get both tokens
		// getTokenResp := model.K8sDashboardTokenResponsePayload{}
		// err = doGet(netClient, getTokenPath, token, &getTokenResp)
		// // this will fail with 500 error, but not permission denied
		// require.Equal(t, true, !isPermissionDeniedError(t, err), "admin user should have get viewonly token permission")
		// err = doGet(netClient, getAdminTokenPath, token, &getTokenResp)
		// // this will fail with 500 error, but not permission denied
		// require.Equal(t, true, !isPermissionDeniedError(t, err), "admin user should have get admin token permission")

		// TODO FIXME: uncomment the following once this change has been deployed to test cloud
		// err = doGet(netClient, getTokenPath, token2, &getTokenResp)
		// require.Error(t, err, "expect get viewonly token by user to fail")
		// require.Equal(t, true, isPermissionDeniedError(t, err), "user should not have get viewonly token permission")

		// err = doGet(netClient, getAdminTokenPath, token2, &getTokenResp)
		// require.Equal(t, true, isPermissionDeniedError(t, err), "user should not have get admin token permission")

		// add viewonly users
		body := model.K8sDashboardViewonlyUserParams{
			UserIDs: []string{user2.ID, user3.ID, user4.ID},
		}
		updateResp := model.K8sDashboardViewonlyUserUpdatePayload{}
		_, err = doPost(netClient, addPath, token, body, &updateResp)
		require.NoError(t, err)

		// get viewonly users
		err = doGet(netClient, getPath, token, &getResp)
		require.NoError(t, err)
		require.Equal(t, 3, len(getResp.ViewonlyUserList), "expect viewonly user count to match")

		// // now user2 get viewonly token should succeed
		// err = doGet(netClient, getTokenPath, token2, &getTokenResp)
		// // this will fail with 500 error, but not permission denied
		// require.Equal(t, true, !isPermissionDeniedError(t, err), "viewonly user should have get viewonly token permission")

		// // get admin token should still fail
		// err = doGet(netClient, getAdminTokenPath, token2, &getTokenResp)
		// require.Equal(t, true, isPermissionDeniedError(t, err), "user should not have get admin token permission")

		// remove viewonly users
		_, err = doPost(netClient, removePath, token, body, &updateResp)
		require.NoError(t, err)

		// get viewonly users
		err = doGet(netClient, getPath, token, &getResp)
		require.NoError(t, err)
		require.Equal(t, 0, len(getResp.ViewonlyUserList), "expect viewonly user count to match")

	})
}

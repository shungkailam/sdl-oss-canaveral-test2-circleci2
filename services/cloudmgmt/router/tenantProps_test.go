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
	TENANT_PROPS_PATH = "/v1/tenantprops"
)

// update tenant props
func updateTenantProps(netClient *http.Client, tenantProps model.TenantProps, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", TENANT_PROPS_PATH, tenantProps.TenantID), tenantProps, token)
}

// delete tenant props
func deleteTenantProps(netClient *http.Client, tenantID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, TENANT_PROPS_PATH, tenantID, token)
}

// get tenant props by id
func getTenantPropsByID(netClient *http.Client, tenantID string, token string) (model.TenantProps, error) {
	tenantProps := model.TenantProps{}
	err := doGet(netClient, TENANT_PROPS_PATH+"/"+tenantID, token, &tenantProps)
	return tenantProps, err
}

func TestTenantProps(t *testing.T) {
	t.Parallel()
	t.Log("running TestTenantProps test")

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

	t.Run("Test Tenant Props", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		tenantProps, err := getTenantPropsByID(netClient, tenantID, token)
		require.NoError(t, err)
		t.Logf("got tenant props: %+v", tenantProps)

		jt := types.JSONText{}
		ms := map[string]string{}
		ms["foo"] = "bar"
		ms["baz"] = "bam"
		err = base.Convert(&ms, &jt)
		require.NoError(t, err)

		up := model.TenantProps{
			TenantID: user.TenantID,
			Props:    jt,
		}
		resp, _, err := updateTenantProps(netClient, up, token)
		require.NoError(t, err)
		t.Logf("update tenant props got response: %+v", resp)

		tenantProps, err = getTenantPropsByID(netClient, tenantID, token)
		require.NoError(t, err)
		t.Logf("got tenant props 2: %+v", tenantProps)

		res, _, err := deleteTenantProps(netClient, tenantID, token)
		require.NoError(t, err)
		t.Logf("delete tenant props got response: %+v", res)

		tenantProps, err = getTenantPropsByID(netClient, tenantID, token)
		require.NoError(t, err)
		t.Logf("got tenant props 3: %+v", tenantProps)

	})

}

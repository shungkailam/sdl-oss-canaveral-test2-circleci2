package router_test

/*
import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/model"
	"cloudservices/common/base"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestEdgeUpgrade(t *testing.T) {
	t.Log("running TestEdgeUpgrade test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	t.Logf("created user: %+v\n", user)

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

	t.Run("Test Edge Upgrade", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		var resp interface{}

		entity := model.ExecuteEdgeUpgrade{
			Release: "v3.0.0",
			EdgeIDs: []string{"edge-id-1", "edge-id-2"},
			Force:   false,
		}

		_, err := doPost(netClient, "/v1/edges/upgrade", token, entity, &resp)

		require.NoError(t, err)

	})

}
*/

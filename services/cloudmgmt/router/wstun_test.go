package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	SETUP_SSH_PATH    = "/v1.0/setupsshtunneling"
	TEARDOWN_SSH_PATH = "/v1.0/teardownsshtunneling"
)

func TestSSHTunneling(t *testing.T) {
	t.Parallel()
	t.Log("running TestSSHTunneling test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

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

	t.Run("Test SSH", func(t *testing.T) {
		// login as user to get token
		token := loginUser(t, netClient, user)

		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID

		// setup, assert success
		resp := model.WstunPayload{}
		_, err = doPost(netClient, SETUP_SSH_PATH, token, model.WstunRequest{ServiceDomainID: edgeID}, &resp)
		require.NoError(t, err)
		// teardown, assert success
		_, err = doPost(netClient, TEARDOWN_SSH_PATH, token, model.WstunTeardownRequest{ServiceDomainID: edgeID, PublicKey: ""}, nil)
		require.NoError(t, err)

		// delete edge
		dresp, _, err := deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
		if dresp.ID != edgeID {
			t.Fatal("edge id mismatch")
		}
	})

}

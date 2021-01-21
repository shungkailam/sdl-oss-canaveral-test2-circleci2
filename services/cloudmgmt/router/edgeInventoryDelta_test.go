package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	EDGE_INVENTORY_DELTA_PATH  = "/v1.0/edgeinventorydelta"
	DEBUG_EDGE_INVENTORY_DELTA = false
)

func getEdgeInventoryDelta(netClient *http.Client, path string, payload *model.EdgeInventoryDeltaPayload, token string) (*model.EdgeInventoryDeltaResponse, string, error) {
	resp := &model.EdgeInventoryDeltaResponse{}
	fmt.Printf("Calling POST on %s\n", path)
	reqID, err := doPost(netClient, path, token, *payload, resp)
	return resp, reqID, err
}

func TestEdgeInventoryDelta(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeInventoryDelta test")

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

	t.Run("Test Edge Inventory Delta", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID

		// create cloud profile
		cloudcreds := makeCloudCreds()
		_, _, err = createCloudCreds(netClient, &cloudcreds, token)
		require.NoError(t, err)
		cloudcredsJ, err := getCloudCredsByID(netClient, cloudcreds.ID, token)
		require.NoError(t, err)

		project := makeExplicitProject(tenantID, []string{cloudcreds.ID}, nil, []string{user.ID}, []string{edgeID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectJ, err := getProjectByID(netClient, project.ID, token)
		require.NoError(t, err)

		mdl := createMLModelForProject(t, netClient, tenantID, project.ID, token)

		mdlJ, err := getMLModelByID(netClient, mdl.ID, token)
		require.NoError(t, err)

		// func getEdgeInventoryDelta(netClient *http.Client, path string, payload *model.EdgeInventoryDeltaPayload, token string) (*model.EdgeInventoryDeltaResponse, string, error)
		payload := &model.EdgeInventoryDeltaPayload{}
		rr, _, err := getEdgeInventoryDelta(netClient, EDGE_INVENTORY_DELTA_PATH+"?edgeId="+edgeID, payload, token)
		require.NoError(t, err)
		// TODO FIXME - add some assertions on rr
		baRR, err := json.Marshal(*rr)
		t.Logf("getEdgeInventoryDelta return: %s", string(baRR))

		resp2, _, err := deleteMLModel(netClient, mdl.ID, token)

		if len(rr.Created.Projects) != 1 {
			t.Fatal("expect project count to be 1")
		}
		if !reflect.DeepEqual(projectJ, rr.Created.Projects[0]) {
			t.Fatal("expect project to equal")
		}
		if len(rr.Created.CloudProfiles) != 1 {
			t.Fatal("expect cloud profiles count to be 1")
		}
		if !reflect.DeepEqual(cloudcredsJ, rr.Created.CloudProfiles[0]) {
			t.Fatal("expect cloud profiles to equal")
		}
		if len(rr.Created.MLModels) != 1 {
			t.Fatal("expect ml model count to be 1")
		}
		if !reflect.DeepEqual(mdlJ, rr.Created.MLModels[0]) {
			t.Fatal("expect ml model to equal")
		}

		require.NoError(t, err)
		if resp2.ID != mdl.ID {
			t.Fatal("delete ml model id mismatch")
		}
		resp, _, err := deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("delete project id mismatch")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudcreds.ID, token)
		require.NoError(t, err)
		if resp.ID != cloudcreds.ID {
			t.Fatal("delete cloudcreds id mismatch")
		}

		resp, _, err = deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
		if resp.ID != edgeID {
			t.Fatal("delete edge id mismatch")
		}
	})

}

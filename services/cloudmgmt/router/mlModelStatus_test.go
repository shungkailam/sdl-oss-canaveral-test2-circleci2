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

// create edge
const (
	ML_MODEL_STATUS_PATH = "/v1.0/mlmodelstatuses"
)

// get edges
func getMLModelsStatus(netClient *http.Client, token string) (model.MLModelStatusListPayload, error) {
	response := model.MLModelStatusListPayload{}
	err := doGet(netClient, ML_MODEL_STATUS_PATH, token, &response)
	return response, err
}
func getMLModelsStatusForModel(netClient *http.Client, token string, modelID string) (model.MLModelStatusListPayload, error) {
	response := model.MLModelStatusListPayload{}
	path := fmt.Sprintf("%s/%s", ML_MODEL_STATUS_PATH, modelID)
	err := doGet(netClient, path, token, &response)
	return response, err
}

// create MLModelStatus
func createMLModelStatus(netClient *http.Client, mdl *model.MLModelStatus, token string) (model.CreateDocumentResponseV2, string, error) {
	return createEntityV2(netClient, ML_MODEL_STATUS_PATH, *mdl, token)
}

// delete MLModelStatus
func deleteMLModelStatus(netClient *http.Client, modelID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, ML_MODEL_STATUS_PATH, modelID, token)
}

func TestMLModelStatus(t *testing.T) {
	t.Parallel()
	t.Log("running TestMLModelStatus test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	t.Logf("user created: email: %s, password: %s", user.Email, user.Password)

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

	t.Run("Test MLModel Status", func(t *testing.T) {
		// login as user to get token

		token := loginUser(t, netClient, user)

		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID

		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, []string{edgeID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID
		t.Logf("created project: %+v", project)

		mdl := createMLModelForProject(t, netClient, tenantID, project.ID, token)
		modelID := mdl.ID

		doc := model.MLModelStatus{
			TenantID:  tenantID,
			EdgeID:    edgeID,
			ModelID:   modelID,
			ProjectID: &projectID,
			Status: []model.MLModelVersionStatus{
				{
					Status:  "status",
					Version: 1,
				},
			},
		}

		resp2, _, err := createMLModelStatus(netClient, &doc, token)
		require.NoError(t, err)
		if resp2.ID != modelID {
			t.Fatal("expect create ml model status response ID to match modelID")
		}

		// get MLModel status
		mlModelStatuses, err := getMLModelsStatus(netClient, token)
		require.NoError(t, err)
		if len(mlModelStatuses.MLModelStatusList) != 1 {
			t.Fatal("expect get ml model status to give count 1")
		}

		r, err := getMLModelsStatusForModel(netClient, token, modelID)
		require.NoError(t, err)
		if len(r.MLModelStatusList) != 1 {
			t.Fatal("expect get ml model status by model id to give count 1")
		}

		// delete model status
		dresp2, _, err := deleteMLModelStatus(netClient, modelID, token)
		require.NoError(t, err)
		if dresp2.ID != modelID {
			t.Fatal("model id mismatch")
		}

		// delete model
		dresp2, _, err = deleteMLModel(netClient, modelID, token)
		require.NoError(t, err)
		if dresp2.ID != modelID {
			t.Fatal("model id mismatch")
		}

		// delete project
		resp, _, err := deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("project id mismatch")
		}

		// delete edge
		resp, _, err = deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
		if resp.ID != edgeID {
			t.Fatal("edge id mismatch")
		}

	})

}

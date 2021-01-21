package websocket_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/websocket"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createMLModel(
	t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string,
) model.MLModel {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	mlDoc := model.MLModelMetadata{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		Name:          "test-ml-model" + base.GetUUID(),
		Description:   "test-ml-model-desc",
		FrameworkType: model.FT_TENSORFLOW_DEFAULT,
		ProjectID:     projectID,
	}
	resp, err := dbAPI.CreateMLModel(ctx, &mlDoc, nil)
	require.NoError(t, err)
	t.Logf("create MLModel successful, %s", resp)
	mlModelID := resp.(model.CreateDocumentResponseV2).ID

	// GET ML model by ID
	mlModel, err := dbAPI.GetMLModel(ctx, mlModelID)
	require.NoError(t, err)
	return mlModel
}

// TestMLModel will test MLModel over web socket
func TestMLModel(t *testing.T) {
	t.Parallel()

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenantID := base.GetUUID()
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(
		context.Background(), base.AuthContextKey, authContext)
	// create tenant
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	require.NoError(t, err)

	t.Logf("create tenant successful, %s", resp)

	// create edge
	edgeName := "my-test-edge"
	edgeSerialNumber := base.GetUUID()
	edgeIP := "1.1.1.1"
	edgeSubnet := "255.255.255.0"
	edgeGateway := "1.1.1.1"

	edge := model.Edge{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         edgeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    edgeIP,
				Subnet:       edgeSubnet,
				Gateway:      edgeGateway,
				EdgeDevices:  3,
			},
			StorageCapacity: 100,
			StorageUsage:    80,
		},
		Connected: true,
	}
	resp, err = dbAPI.CreateEdge(ctx, &edge, nil)
	require.NoError(t, err)
	t.Logf("create edge successful, %s", resp)

	edgeID := resp.(model.CreateDocumentResponse).ID

	// create project
	projName := fmt.Sprintf("Where is Waldo-%s", base.GetUUID())
	projDesc := "Find Waldo"
	project := model.Project{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:               projName,
		Description:        projDesc,
		CloudCredentialIDs: []string{},
		DockerProfileIDs:   []string{},
		Users:              []model.ProjectUserInfo{},
		EdgeSelectorType:   model.ProjectEdgeSelectorTypeExplicit,
		EdgeIDs:            []string{edgeID},
		EdgeSelectors:      nil,
	}
	resp, err = dbAPI.CreateProject(ctx, &project, nil)
	require.NoError(t, err)
	t.Logf("create project successful, %s", resp)

	projectID := resp.(model.CreateDocumentResponse).ID

	// add project id for app create permission
	projRoles := []model.ProjectRole{
		{
			ProjectID: projectID,
			Role:      model.ProjectRoleAdmin,
		},
	}
	authContext.Claims["projects"] = projRoles

	// create ML Model
	mlModel := createMLModel(t, dbAPI, tenantID, projectID)
	modelID := mlModel.ID

	// Teardown
	defer func() {
		dbAPI.DeleteMLModel(ctx, modelID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		defer dbAPI.Close()
	}()

	req := websocket.ReportMLModelStatusRequest{
		TenantID:  tenantID,
		EdgeID:    edgeID,
		EdgeName:  edgeName,
		ID:        modelID,
		ModelName: mlModel.Name,
		Status: []model.MLModelVersionStatus{
			{
				Status: model.MLModelStatusActive, Version: 1,
			},
		},
	}

	ba, err := json.Marshal(req)
	require.NoError(t, err)
	bas := base64.StdEncoding.EncodeToString(ba)

	c, err := gosocketio.Dial(
		gosocketio.GetUrl(apitesthelper.TestServer, apitesthelper.TestPort, apitesthelper.TestSecure),
		transport.GetDefaultWebsocketTransport())
	require.NoError(t, err)

	// note: Ack or Emit both works
	result, err := c.Ack("mlmodel-status", bas, time.Second*20)
	require.NoError(t, err)

	t.Log("Ack result to /mlmodel-status: ", result)
	rsp := websocket.ReportMLModelStatusResponse{}
	err = json.Unmarshal([]byte(result), &rsp)
	require.NoError(t, err)
	require.Equal(t, rsp.StatusCode, 200, "response status not ok")
	t.Logf("response: %+v", rsp)

	// delete MLModel status
	delResp, err := dbAPI.DeleteMLModelStatus(ctx, modelID, nil)
	require.NoError(t, err)
	t.Logf("delete application successful, %v", delResp)
}

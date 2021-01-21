package websocket_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/websocket"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestModifyExecuteEdgeUpgradeData(t *testing.T) {
	t.Parallel()
	t.Log("running modifyExecuteEdgeUpgradeData test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	defer dbAPI.Close()
	reqID := base.GetUUID()
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	ctx := base.GetAdminContext(reqID, tenantID)
	defer dbAPI.DeleteTenant(ctx, tenantID, nil)
	edgeDoc := model.Edge{
		BaseModel: model.BaseModel{
			Version:  3,
			TenantID: tenantID,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         "test-edge-1",
				SerialNumber: base.GetUUID(),
				IPAddress:    "1.1.1.1",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
				EdgeDevices:  3,
			},
			StorageCapacity: 100,
			StorageUsage:    80,
		},
		Connected: true,
	}

	resp, err := dbAPI.CreateEdge(ctx, &edgeDoc, nil)
	require.NoError(t, err)
	t.Logf("create edge successful, %s", resp)
	edgeID := (resp.(model.CreateDocumentResponse)).ID
	defer dbAPI.DeleteEdge(ctx, edgeID, nil)
	edgeInfo, err := dbAPI.GetEdgeInfo(ctx, edgeID)
	require.NoError(t, err)
	edgeInfo.EdgeVersion = base.StringPtr("v1.8.0")
	_, err = dbAPI.UpdateEdgeInfo(ctx, &edgeInfo, nil)
	require.NoError(t, err)
	upgradeReqMap := map[string]interface{}{}
	upgradeReqMap["requestId"] = reqID
	doc := map[string]interface{}{
		"data": "blah",
	}
	upgradeReqMap["doc"] = doc
	_, err = websocket.ModifyExecuteEdgeUpgradeData(dbAPI, edgeID, tenantID, upgradeReqMap)
	require.NoError(t, err)
	if len(doc["data"].(string)) != 0 {
		t.Fatalf("Expected the data be cleared, found %v", doc)
	}
	doc["data"] = "blah"
	edgeInfo.EdgeVersion = base.StringPtr("v0.9.0")
	_, err = dbAPI.UpdateEdgeInfo(ctx, &edgeInfo, nil)
	require.NoError(t, err)
	_, err = websocket.ModifyExecuteEdgeUpgradeData(dbAPI, edgeID, tenantID, upgradeReqMap)
	require.NoError(t, err)
	data := doc["data"].(string)
	if len(data) == 0 || data != "blah" {
		t.Fatalf("Expected the data be intact, found %v", doc)
	}
}

package websocket_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/websocket"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type reportEdgePayload struct {
	TenantID string `json:"tenantId"`
	Doc      model.Edge
}

// TestWebSocket will test web socket
func TestWebSocket(t *testing.T) {
	t.Parallel()

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)

	tenantID := base.GetUUID()
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)

	// Create tenant object
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	// create tenant
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	if err != nil {
		dbAPI.Close()
		t.Fatal(err)
	}

	t.Logf("create tenant successful, %s", resp)

	edgeSerialNumber := base.GetUUID()
	var edgeDevices float64 = 3
	var storageCapacity float64 = 100
	var storageUsage float64 = 80
	var version float64 = 5

	edge := model.Edge{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  version,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         "edge-name",
				SerialNumber: edgeSerialNumber,
				IPAddress:    "1.1.1.1",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
				EdgeDevices:  edgeDevices,
			},
			StorageCapacity: storageCapacity,
			StorageUsage:    storageUsage,
		},
		Connected: true,
	}

	// create edge
	resp, err = dbAPI.CreateEdge(ctx, &edge, nil)
	if err != nil {
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
		t.Fatal(err)
	}
	t.Logf("create edge successful, %s", resp)

	edgeId := resp.(model.CreateDocumentResponse).ID
	edge.ID = edgeId

	// Teardown
	defer func() {
		dbAPI.DeleteEdge(ctx, edgeId, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	req := reportEdgePayload{
		TenantID: tenantID,
		Doc:      edge,
	}

	c, err := gosocketio.Dial(
		gosocketio.GetUrl(apitesthelper.TestServer, apitesthelper.TestPort, apitesthelper.TestSecure),
		transport.GetDefaultWebsocketTransport())
	require.NoError(t, err)

	if !c.IsAlive() {
		t.Fatal("expect websocket client to be alive")
	}

	result, err := c.Ack("reportEdge", req, time.Second*20)
	require.NoError(t, err)

	t.Log("Ack result to /reportEdge: ", result)
	rsp := websocket.ReportEdgeResponse{}
	err = json.Unmarshal([]byte(result), &rsp)
	require.NoError(t, err)
	if rsp.StatusCode != 200 {
		t.Fatalf("reportEdge response code(%d) != 200", rsp.StatusCode)
	}
	t.Logf("report edge response: %d, %+v", rsp.StatusCode, *rsp.Doc)

	// test websocket keep alive for 45 seconds
	wg := new(sync.WaitGroup)
	wg.Add(1)
	ticker := time.NewTicker(1 * time.Second)
	maxCount := 45
	defer ticker.Stop()
	go func() {
		count := 1
		for {
			select {
			case <-ticker.C:
				if !c.IsAlive() {
					wg.Done()
					t.Fatalf("expect websocket client to be alive - tick %d", count)
					break
				}
				t.Logf("%d/%d ", count, maxCount)
				count = count + 1
				if count > maxCount {
					wg.Done()
					break
				}
			}
		}
	}()
	wg.Wait()
}

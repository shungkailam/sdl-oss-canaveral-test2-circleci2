package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestLog(t *testing.T) {
	t.Parallel()
	t.Log("running TestLog test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)

	tenantID := base.GetUUID()
	edgeID1 := base.GetUUID()
	edgeID2 := base.GetUUID()

	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)
	// Create the tenant
	tenantDoc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	authContext1 := &base.AuthContext{
		TenantID: tenantID,
	}
	ctx1 := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
	// create tenant
	resp1, err := dbAPI.CreateTenant(ctx, &tenantDoc, nil)
	require.NoError(t, err)
	t.Logf("create tenant successful, %s", resp1)

	// Teardown
	defer func() {
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	edgeDoc1 := model.Edge{
		BaseModel: model.BaseModel{
			ID:       edgeID1,
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

	resp2, err := dbAPI.CreateEdge(ctx, &edgeDoc1, nil)
	require.NoError(t, err)
	t.Logf("create edge successful, %s", resp2)
	defer dbAPI.DeleteEdge(ctx, edgeID1, nil)

	edgeDoc2 := model.Edge{
		BaseModel: model.BaseModel{
			ID:       edgeID2,
			Version:  3,
			TenantID: tenantID,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         "test-edge-2",
				SerialNumber: base.GetUUID(),
				IPAddress:    "1.1.1.2",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
				EdgeDevices:  3,
			},
			StorageCapacity: 100,
			StorageUsage:    80,
		},
		Connected: true,
	}

	resp3, err := dbAPI.CreateEdge(ctx, &edgeDoc2, nil)
	require.NoError(t, err)
	t.Logf("create edge successful, %s", resp3)
	defer dbAPI.DeleteEdge(ctx, edgeID2, nil)
	category := createCategory(t, dbAPI, tenantID)
	catInfos := []model.CategoryInfo{
		{
			ID:    category.ID,
			Value: TestCategoryValue1,
		},
	}
	defer dbAPI.DeleteCategory(ctx, category.ID, nil)
	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, catInfos)
	defer dbAPI.DeleteProject(ctx, project.ID, nil)
	projectCtx, _, _ := makeContext(tenantID, []string{project.ID})
	app := createApplicationWithState(t, dbAPI, tenantID, "test-application-id", project.ID, nil, nil, model.DeployEntityState.StringPtr())
	require.NoError(t, err)
	t.Logf("%+v", app)
	defer dbAPI.DeleteApplication(projectCtx, app.ID, nil)

	t.Run("Create/Get/DeleteLog", func(t *testing.T) {
		t.Log("running reate/Get/DeleteLog test")

		// Test the rest endpoint to skip websocket push
		doc := model.RequestLogUploadPayload{
			EdgeIDs:       []string{edgeID1, edgeID2},
			ApplicationID: app.ID,
		}
		doc1 := model.RequestLogUploadPayload{
			EdgeIDs: []string{edgeID1, edgeID2},
		}
		// Negative test
		doc2 := model.RequestLogUploadPayload{
			EdgeIDs: []string{edgeID1, edgeID1},
		}
		doc3 := model.RequestLogUploadPayload{
			EdgeIDs: []string{"'"},
		}
		resp1, err := dbAPI.RequestLogUpload(ctx, doc1, nil)
		require.NoError(t, err)

		for _, r := range resp1 {
			if len(r.BatchID) == 0 {
				t.Fatalf("Batch ID is expected in the response %+v", r)
			}
			t.Log("Served log upload request URL", r.URL)
		}

		t.Log("request application log upload 1 with infra admin successful ", resp1)

		resp2, err := dbAPI.RequestLogUpload(projectCtx, doc, nil)
		require.NoError(t, err)

		for _, r := range resp2 {
			if len(r.BatchID) == 0 {
				t.Fatalf("Batch ID is expected in the response %+v", r)
			}
			t.Log("Served log upload request URL", r.URL)
		}
		t.Log("request application log upload 2 with user successful ", resp2)

		resp3, err := dbAPI.RequestLogUpload(ctx1, doc1, nil)
		require.Error(t, err, "RequestLogUpload must fail because non infra user requested edge logs")
		if len(resp3) != 0 {
			t.Fatal("RequestLogUpload must fail")
		}
		t.Log("request edge log upload 3 with user failed as expected ", resp3)

		resp4, err := dbAPI.RequestLogUpload(ctx, doc2, nil)
		require.Error(t, err, "RequestLogUpload must fail because of duplicate edge")
		require.Contains(t, err.Error(), "Error creating log entry for edge", "RequestLogUpload must fail with create log entry error")

		if len(resp4) != 1 {
			t.Fatal("RequestLogUpload must fail")
		}
		t.Log("request edge log upload 4 failed as expected ", resp4)

		// bad edgeID must fail
		resp5, err := dbAPI.RequestLogUpload(ctx, doc3, nil)
		require.Error(t, err, "RequestLogUpload must fail because of bad edge ID")
		if len(resp5) != 0 {
			t.Fatal("RequestLogUpload must fail")
		}
		t.Log("request edge log upload 5 failed as expected ", resp5)
		for _, uploadPayload := range resp1 {
			if strings.Contains(uploadPayload.URL, edgeID1) {
				err = dbAPI.UploadLogComplete(ctx, model.LogUploadCompletePayload{URL: uploadPayload.URL, Status: model.LogUploadFailed, ErrorMessage: "Error"})
				require.Errorf(t, err, "Only the edge can call this API")
			}
		}
		edgeCtx1 := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      edgeID1,
			},
		})
		for _, uploadPayload := range resp1 {
			if strings.Contains(uploadPayload.URL, edgeID1) {
				err = dbAPI.UploadLogComplete(edgeCtx1, model.LogUploadCompletePayload{URL: uploadPayload.URL, Status: model.LogUploadFailed, ErrorMessage: "Error"})
				require.NoError(t, err)
			}
		}
		edgeCtx2 := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      edgeID2,
			},
		})
		for _, uploadPayload := range resp1 {
			if strings.Contains(uploadPayload.URL, edgeID2) {
				doc := model.LogUploadCompletePayload{URL: uploadPayload.URL, Status: model.LogUploadFailed, ErrorMessage: "Error"}
				wrapperDoc := model.ObjectResponseLogUploadComplete{TenantID: tenantID, Doc: doc}
				payload, _ := json.Marshal(wrapperDoc)
				err = dbAPI.UploadLogCompleteW(edgeCtx2, strings.NewReader(string(payload)))
				require.NoError(t, err)
			}
		}
		t.Log("update log upload complete successful")
		logEntries, err := dbAPI.SelectAllLogs(ctx, "", nil, nil)
		require.NoError(t, err)

		failedCount := 0
		pendingCount := 0
		successCount := 0
		for _, logEntry := range logEntries {
			t.Log("log entry", logEntry)
			if logEntry.Status == model.LogUploadFailed {
				failedCount++
			} else if logEntry.Status == model.LogUploadPending {
				pendingCount++
			} else if logEntry.Status == model.LogUploadSuccess {
				successCount++
			} else {
				t.Fatalf("Unknown state %s", logEntry.Status)
			}
		}
		if failedCount != 2 || pendingCount != 3 || successCount != 0 {
			t.Fatalf("Mismatched counts. Failed: %d, Pending: %d, Success: %d", failedCount, pendingCount, successCount)
		}
		t.Log("get log entries with infra admin successful")

		logEntries, err = dbAPI.SelectAllLogs(ctx, edgeID2, nil, nil)
		require.NoError(t, err)
		if len(logEntries) != 2 {
			// 2 pending
			t.Fatalf("Expected %d for edge %s, found %d: %+v", 2, edgeID2, len(logEntries), logEntries)
		}

		logEntries, err = dbAPI.SelectAllLogs(ctx, "", []model.LogTag{{Name: model.ApplicationLogTag}}, nil)
		require.NoError(t, err)
		if len(logEntries) != 2 {
			// 2 pending from edge1 and edge2
			t.Fatalf("Expected %d, found %d: %+v", 2, len(logEntries), logEntries)
		}
		for _, logEntry := range logEntries {
			if len(logEntry.Tags) == 0 {
				t.Fatalf("Expected to find a tag, found no tag: %+v", logEntries)
			}
		}

		logEntries, err = dbAPI.SelectAllLogs(ctx, edgeID2, []model.LogTag{{Name: model.ApplicationLogTag}}, nil)
		require.NoError(t, err)
		if len(logEntries) != 1 {
			// 1 pending from edge2
			t.Fatalf("Expected %d for edge %s, found %d: %+v", 2, edgeID2, len(logEntries), logEntries)
		}
		for _, logEntry := range logEntries {
			if len(logEntry.Tags) == 0 {
				t.Fatalf("Expected to find a tag, found no tag: %+v", logEntries)
			}
		}

		logEntries, err = dbAPI.SelectAllLogs(ctx, "", []model.LogTag{{Name: model.ApplicationLogTag, Value: doc.ApplicationID}}, nil)
		require.NoError(t, err)
		if len(logEntries) != 2 {
			// 2 pending from edge1 and edge2
			t.Fatalf("Expected %d, found %d: %+v", 2, len(logEntries), logEntries)
		}
		for _, logEntry := range logEntries {
			if len(logEntry.Tags) == 0 {
				t.Fatalf("Expected to find a tag, found no tag: %+v", logEntries)
			}
			for _, tag := range logEntry.Tags {
				if tag.Name != model.ApplicationLogTag || tag.Value != doc.ApplicationID {
					t.Fatalf("Expected to find a tag name and ID, found no tag name and ID: %+v", logEntries)
				}
			}
		}

		logEntries, err = dbAPI.SelectAllLogs(ctx1, "", nil, nil)
		require.NoError(t, err)

		failedCount = 0
		pendingCount = 0
		successCount = 0
		for _, logEntry := range logEntries {
			t.Log("log entry", logEntry)
			if logEntry.Status == model.LogUploadFailed {
				failedCount++
			} else if logEntry.Status == model.LogUploadPending {
				pendingCount++
			} else if logEntry.Status == model.LogUploadSuccess {
				successCount++
			} else {
				t.Fatalf("Unknown state %s", logEntry.Status)
			}
		}
		if failedCount != 0 || pendingCount != 2 || successCount != 0 {
			t.Fatalf("Mismatched counts. Failed: %d, Pending: %d, Success: %d", failedCount, pendingCount, successCount)
		}
		t.Log("get log entries with user successful")
		// Wait for the pending logs to time out
		time.Sleep(time.Second * 10)
		err = dbAPI.ScheduleTimeOutPendingLogsJob(ctx, time.Second*2, time.Second*5)
		require.NoError(t, err)

		logEntries, err = dbAPI.SelectAllLogs(ctx, "", nil, nil)
		require.NoError(t, err)

		failedCount = 0
		pendingCount = 0
		successCount = 0
		for _, logEntry := range logEntries {
			t.Log("log entry", logEntry)
			if logEntry.Status == model.LogUploadFailed {
				failedCount++
			} else if logEntry.Status == model.LogUploadPending {
				pendingCount++
			} else if logEntry.Status == model.LogUploadSuccess {
				successCount++
			} else {
				t.Fatalf("Unknown state %s", logEntry.Status)
			}
		}
		if failedCount != 2 || pendingCount != 3 || successCount != 0 {
			t.Fatalf("Mismatched counts. Failed: %d, Pending: %d, Success: %d", failedCount, pendingCount, successCount)
		}
		t.Log("get log entries with infra admin before timeout successful")
		// Wait for the scheduler to launch the job
		time.Sleep(time.Second * 5)
		logEntries, err = dbAPI.SelectAllLogs(ctx, "", nil, nil)
		require.NoError(t, err)

		failedCount = 0
		pendingCount = 0
		successCount = 0
		for _, logEntry := range logEntries {
			t.Log("log entry after waiting", logEntry)
			if logEntry.Status == model.LogUploadFailed {
				failedCount++
			} else if logEntry.Status == model.LogUploadPending {
				pendingCount++
			} else if logEntry.Status == model.LogUploadSuccess {
				successCount++
			} else {
				t.Fatalf("Unknown state %s", logEntry.Status)
			}
		}
		if failedCount != 5 || pendingCount != 0 || successCount != 0 {
			t.Fatalf("Mismatched counts after waiting. Failed: %d, Pending: %d, Success: %d", failedCount, pendingCount, successCount)
		}
		t.Log("get log entries with infra admin after timeout successful")

		failedCount = 0
		successCount = 0
		for _, logEntry := range logEntries {
			delResp, err := dbAPI.DeleteLogEntry(ctx1, logEntry.ID, nil)
			if err != nil {
				t.Log(err)
				failedCount++
			} else {
				successCount++
			}
			t.Log("delete log entries response", delResp)
		}
		t.Log("delete log entries successful")
		if failedCount != 3 || successCount != 2 {
			t.Fatalf("Mismatched counts on delete. Failed: %d, Success: %d", failedCount, successCount)
		}
		t.Log("delete log entries with user successful")
		logEntries, err = dbAPI.SelectAllLogs(ctx, "", nil, nil)
		require.NoError(t, err)

		t.Log("get all log entries before delete with infra admin successful")
		failedCount = 0
		successCount = 0
		for _, logEntry := range logEntries {
			delResp, err := dbAPI.DeleteLogEntry(ctx, logEntry.ID, nil)
			if err != nil {
				t.Log(err)
				failedCount++
			} else {
				successCount++
			}
			t.Log("delete log entries response", delResp)
		}
		t.Log("delete log entries with infra user successful")
		if failedCount != 0 || successCount != 3 {
			t.Fatalf("Mismatched counts on delete. Failed: %d,Success: %d", failedCount, successCount)
		}
		count := 0
		logEntries, err = dbAPI.SelectAllLogs(ctx, "", nil, nil)
		require.NoError(t, err)

		if count != 0 {
			t.Fatalf("Mismatched counts on SelectAllLogs. Final count: %d", count)
		}
		t.Log("get final log entries successful")
	})
}

func TestExtractEdgeID(t *testing.T) {
	tcs := []struct {
		in  string
		out string
	}{
		{
			"https://bucket.s3.us-west-2.amazonaws.com/v1/tenantId/2018/04/18/123/456/123-456.tgz?AWSAccessKeyId=.....",
			"456",
		},
		{
			"http://minio-service:9000/sherlock-support-bundle-us-west-2/v1/tid-pmp-test-1/2020/12/10/83a19395-0b2a-4183-bb81-ac0900ba09e2/3ee34458-a63f-4c59-9f36-726d1f35cbe8/3ee34458-a63f-4c59-9f36-726d1f35cbe8-83a19395-0b2a-4183-bb81-ac0900ba09e2.tgz?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=VUHV97HVZ69G2UDSHEGF%2F20201210...",
			"3ee34458-a63f-4c59-9f36-726d1f35cbe8",
		},
		{
			"http://minio-service:9000/sherlock-support-bundle-us-west-2/v1/tid-pmp-test-1/2020/12/10//83a19395-0b2a-4183-bb81-ac0900ba09e2///3ee34458-a63f-4c59-9f36-726d1f35cbe8/3ee34458-a63f-4c59-9f36-726d1f35cbe8-83a19395-0b2a-4183-bb81-ac0900ba09e2.tgz?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=VUHV97HVZ69G2UDSHEGF%2F20201210...",
			"3ee34458-a63f-4c59-9f36-726d1f35cbe8",
		},
	}
	for _, tc := range tcs {
		out, err := api.ExtractEdgeID(tc.in)
		require.NoError(t, err)
		t.Logf("Output: %v", out)
		require.Equal(t, tc.out, out, "Failed test %+v, found %s", tc, out)
	}

}

func TestExtractLogLocation(t *testing.T) {
	tcs := []struct {
		in  string
		out string
	}{
		{
			"https://bucket.s3.us-west-2.amazonaws.com/v1/tenantId/2018/04/18/123/456/123-456.tgz?AWSAccessKeyId=.....",
			"v1/tenantId/2018/04/18/123/456/123-456.tgz",
		},
		{
			"http://minio-service:9000/sherlock-support-bundle-us-west-2/v1/tid-pmp-test-1/2020/12/10/83a19395-0b2a-4183-bb81-ac0900ba09e2/3ee34458-a63f-4c59-9f36-726d1f35cbe8/3ee34458-a63f-4c59-9f36-726d1f35cbe8-83a19395-0b2a-4183-bb81-ac0900ba09e2.tgz?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=VUHV97HVZ69G2UDSHEGF%2F20201210...",
			"v1/tid-pmp-test-1/2020/12/10/83a19395-0b2a-4183-bb81-ac0900ba09e2/3ee34458-a63f-4c59-9f36-726d1f35cbe8/3ee34458-a63f-4c59-9f36-726d1f35cbe8-83a19395-0b2a-4183-bb81-ac0900ba09e2.tgz",
		},
		{
			"http://minio-service:9000/sherlock-support-bundle-us-west-2/v1/tid-pmp-test-1/2020/12/10//83a19395-0b2a-4183-bb81-ac0900ba09e2///3ee34458-a63f-4c59-9f36-726d1f35cbe8/3ee34458-a63f-4c59-9f36-726d1f35cbe8-83a19395-0b2a-4183-bb81-ac0900ba09e2.tgz?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=VUHV97HVZ69G2UDSHEGF%2F20201210...",
			"v1/tid-pmp-test-1/2020/12/10/83a19395-0b2a-4183-bb81-ac0900ba09e2/3ee34458-a63f-4c59-9f36-726d1f35cbe8/3ee34458-a63f-4c59-9f36-726d1f35cbe8-83a19395-0b2a-4183-bb81-ac0900ba09e2.tgz",
		},
	}
	for _, tc := range tcs {
		out, err := api.ExtractLogLocation(tc.in)
		require.NoError(t, err)
		t.Logf("Output: %v", out)
		require.Equal(t, tc.out, out, "Failed test %+v, found %s", tc, out)
	}

}

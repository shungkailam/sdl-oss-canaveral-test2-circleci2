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
	AUDITLOGS_PATH     = "/v1/auditlogs"
	AUDITLOGS_PATH_NEW = "/v1.0/auditlogs"
)

// get category by id
func getAuditLogsByRequestID(netClient *http.Client, reqID string, token string) ([]model.AuditLog, error) {
	auditLogs := []model.AuditLog{}
	err := doGet(netClient, AUDITLOGS_PATH+"/"+reqID, token, &auditLogs)
	return auditLogs, err
}

func TestAuditLog(t *testing.T) {
	t.Parallel()
	t.Log("running TestAuditLog test")

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
		if ctx == nil {
			t.Log("boo")
		}
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Category CUD AuditLog", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		category2 := model.Category{
			Name:    "test-cat-2",
			Purpose: "test category",
			Values:  []string{"v1", "v2", "v3"},
		}
		_, cReqID, err := createCategory(netClient, &category2, token)
		require.NoError(t, err)
		time.Sleep(time.Second)
		// now get audit log for the request id
		logs, err := getAuditLogsByRequestID(netClient, cReqID, token)
		require.NoError(t, err, "failed to get audit log for create category, request id: %s", cReqID)
		if len(logs) == 0 {
			t.Fatal("expect create audit logs length to be nonzero")
		}
		t.Logf("got create category audit logs: count=%d, %+v", len(logs), logs)

		category2Updated := model.Category{
			Name:    "test-cat-2-updated",
			Purpose: "test category updated",
			Values:  []string{"v1", "v2", "v3-updated"},
		}
		ur, uReqID, err := updateCategory(netClient, category2.ID, category2Updated, token)
		require.NoError(t, err)
		if ur.ID != category2.ID {
			t.Fatal("expect update category id to match")
		}
		time.Sleep(time.Second)
		// now get audit log for the request id
		logs, err = getAuditLogsByRequestID(netClient, uReqID, token)
		require.NoError(t, err, "failed to get audit log for update category, request id: %s, category id: %s", uReqID, category2.ID)
		if len(logs) == 0 {
			t.Fatal("expect update audit logs length to be nonzero")
		}
		t.Logf("got update category audit logs: count=%d, %+v", len(logs), logs)

		resp, dReqID, err := deleteCategory(netClient, category2.ID, token)
		require.NoError(t, err)
		if resp.ID != category2.ID {
			t.Fatal("delete category 2 id mismatch")
		}
		time.Sleep(time.Second)
		// now get audit log for the request id
		logs, err = getAuditLogsByRequestID(netClient, dReqID, token)
		require.NoErrorf(t, err, "failed to get audit log for delete category, request id: %s", dReqID)
		if len(logs) == 0 {
			t.Fatal("expect delete audit logs length to be nonzero")
		}
		t.Logf("got delete category audit logs: count=%d, %+v", len(logs), logs)

	})

}

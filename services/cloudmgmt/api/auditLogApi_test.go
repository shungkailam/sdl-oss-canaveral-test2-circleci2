package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func TestAuditLog(t *testing.T) {
	t.Parallel()
	t.Log("running TestAuditLog test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	tenantID := base.GetUUID()
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	authContext2 := &base.AuthContext{
		TenantID: tenantID,
	}
	tableName := api.GetAuditLogTableName()
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	ctx = context.WithValue(ctx, base.AuditLogTableNameKey, tableName)
	ctx2 := context.WithValue(context.Background(), base.AuthContextKey, authContext2)
	ctx2 = context.WithValue(ctx2, base.AuditLogTableNameKey, tableName)

	defer func() {
		dbAPI.Close()
	}()

	t.Run("TestGetAuditLog", func(t *testing.T) {
		t.Log("running GetAuditLog test")
		reqID := base.GetUUID()
		auditLog := &model.AuditLog{StartedAt: time.Now(), RequestID: reqID, TenantID: tenantID}
		err := dbAPI.WriteAuditLog(ctx, auditLog)
		require.NoError(t, err)
		logs, err := dbAPI.GetAuditLog(ctx, reqID)
		require.NoError(t, err)
		if len(logs) != 1 {
			t.Fatalf("expect audit log entries count to be 1, got %d", len(logs))
		}
		log := logs[0]
		if log.TenantID != tenantID || log.RequestID != reqID {
			t.Fatal("expect audit log entry to match")
		}

		url := "http://example.com/foo"
		req := httptest.NewRequest("GET", url, nil)
		queryParams := model.GetAuditLogQueryParam(req)
		resp, err := dbAPI.SelectAuditLogs(ctx, queryParams)
		require.NoError(t, err)
		if resp.PageIndex != 0 || resp.TotalCount != 1 {
			t.Fatal("expect result count to be 1")
		}
		if false == reflect.DeepEqual(resp.AuditLogList, logs) {
			t.Fatal("expec two results to match")
		}

		// non infra admin should not be able to get audit log
		logs, err = dbAPI.GetAuditLog(ctx2, reqID)
		require.Error(t, err, "expect get audit log to fail for non infra admin")

		// select audit logs must fail for non infra admin
		resp, err = dbAPI.SelectAuditLogs(ctx2, queryParams)
		require.Error(t, err, "expect select audit logs to fail for non infra admin")

		err = dbAPI.DeleteTenantAuditLogs(ctx)
		require.NoError(t, err)
		resp, err = dbAPI.SelectAuditLogs(ctx, queryParams)
		require.NoError(t, err)
		if len(resp.AuditLogList) != 0 {
			t.Fatal("expect no logs left")
		}
		logs, err = dbAPI.GetAuditLog(ctx, reqID)
		require.Error(t, err, "expect get by id to fail")

	})

}

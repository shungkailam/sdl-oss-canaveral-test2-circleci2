package core_test

import (
	"cloudservices/common/base"
	"cloudservices/tenantpool/config"
	"cloudservices/tenantpool/core"
	"cloudservices/tenantpool/model"
	"cloudservices/tenantpool/testhelper"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/pkg/errors"
)

func TestAuditLog(t *testing.T) {
	config.Cfg.EnableScanner = base.BoolPtr(false)
	tenantPoolManager, err := core.NewTenantPoolManager(testhelper.NewTestEdgeProvisioner())
	require.NoError(t, err)
	auditLogManager := tenantPoolManager.GetAuditLogManager()
	defer tenantPoolManager.Close()
	t.Run("Running auditLog tests", func(t *testing.T) {
		ctx := context.Background()
		auditLog := &model.AuditLog{
			Email:  fmt.Sprintf("%s@ntnxsherlock.com", base.GetUUID()),
			Actor:  model.AuditLogSystemActor,
			Action: model.AuditLogReserveTenantAction,
		}
		err := auditLogManager.CreateAuditLogHelper(ctx, nil, auditLog, false)
		require.NoError(t, err)
		auditLogs, err := auditLogManager.GetAuditLogs(ctx, &model.AuditLog{Email: auditLog.Email})
		require.NoError(t, err)
		if len(auditLogs) != 1 {
			t.Fatalf("Expected 1, found %d", len(auditLogs))
		}
		if auditLogs[0].Response != model.AuditLogSuccessResponse {
			t.Fatalf("Expected %s, found %s", model.AuditLogSuccessResponse, auditLogs[0].Response)
		}
		auditLog = &model.AuditLog{
			Email:  fmt.Sprintf("%s@ntnxsherlock.com", base.GetUUID()),
			Actor:  model.AuditLogSystemActor,
			Action: model.AuditLogReserveTenantAction,
		}
		err = auditLogManager.CreateAuditLogHelper(ctx, errors.New("NO_RESOURCE"), auditLog, false)
		require.NoError(t, err)
		auditLogs, err = auditLogManager.GetAuditLogs(ctx, &model.AuditLog{Email: auditLog.Email})
		require.NoError(t, err)
		if len(auditLogs) != 1 {
			t.Fatalf("Expected 1, found %d", len(auditLogs))
		}
		if auditLogs[0].Response != model.AuditLogFailedResponse {
			t.Fatalf("Expected %s, found %s", model.AuditLogFailedResponse, auditLogs[0].Response)
		}
		if auditLogs[0].Description != "NO_RESOURCE" {
			t.Fatalf("Expected NO_RESOURCE, found %s", auditLogs[0].Description)
		}
		for _, auditLog := range auditLogs {
			err = auditLogManager.DeleteAuditLog(ctx, auditLog)
			require.NoError(t, err)
			t.Logf("AuditLog: %+v", auditLog)
		}
	})
}

package core_test

import (
	"cloudservices/common/base"
	"cloudservices/tenantpool/core"
	"cloudservices/tenantpool/model"
	"cloudservices/tenantpool/testhelper"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRenameSerialNumberQuery(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	edgeProvisioner := testhelper.NewTestEdgeProvisioner()
	poolManager, err := core.NewTenantPoolManager(edgeProvisioner)
	require.NoError(t, err)
	bookKeeper := poolManager.GetBookKeeper()
	updatedAt, err := time.Parse(time.RFC3339, "2019-11-01T22:08:41+00:00")
	require.NoError(t, err)
	expectedQuery := "update edge_device_model set serial_number=concat(serial_number, '.1572646121.ntnx-del'), updated_at='2019-11-01 22:08:41' where tenant_id='test-tenant' and serial_number not like '%.ntnx-del'"
	query := core.GetRenameSerialNumberQuery("test-tenant", updatedAt)
	if query != expectedQuery {
		t.Fatalf("expected %s, found %s", expectedQuery, query)
	}
	err = bookKeeper.RenameSerialNumbers(ctx, &model.TenantClaim{Trial: true, ID: base.GetUUID()})
	require.NoError(t, err)
}

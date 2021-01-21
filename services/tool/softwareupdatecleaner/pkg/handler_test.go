package pkg_test

import (
	"cloudservices/tool/softwareupdatecleaner/pkg"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSoftwareUpdateCleaner(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	err := pkg.ConnectDB()
	require.NoError(t, err)
	pkg.DBAPI.Close()
	count, err := pkg.DeleteExpiredBatches(ctx)
	require.NoError(t, err)
	t.Logf("Deleted %d", count)
}

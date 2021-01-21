package core_test

import (
	"cloudservices/tenantpool/core"
	"cloudservices/tenantpool/testhelper"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func getDisplayFields(poolStats *core.PoolStats) interface{} {
	return struct {
		poolSize int
		st       float64
	}{
		poolStats.GetPoolSize(),
		poolStats.GetWeightedAvg(),
	}
}

func TestPoolStatsCalculator(t *testing.T) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	defer cancelFunc()
	edgeProvisioner := testhelper.NewTestEdgeProvisioner()
	poolManager, err := core.NewTenantPoolManager(edgeProvisioner)
	require.NoError(t, err)

	minPoolSize := 2
	maxPoolSize := 200
	availableCount := 2
	poolStats := core.NewPoolStats(2.0, availableCount, 1)

	requestCount := 5
	for i := 0; i < 4; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 10
	availableCount = 1
	for i := 0; i < 4; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 20
	availableCount = 1
	for i := 0; i < 4; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 30
	availableCount = 1
	for i := 0; i < 10; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}
	requestCount = 20
	availableCount = 1
	for i := 0; i < 100; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 0
	availableCount = 1
	for i := 0; i < 50; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 10
	availableCount = 1
	for i := 0; i < 20; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 25
	availableCount = 1
	for i := 0; i < 10; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 3
	availableCount = 1
	for i := 0; i < 2; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 5
	availableCount = 1
	for i := 0; i < 2; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}

	requestCount = 0
	availableCount = 1
	for i := 0; i < 2; i++ {
		err = poolManager.CalculatePoolStatsHelper(ctx, poolStats, requestCount, minPoolSize, maxPoolSize)
		require.NoError(t, err)
		t.Logf("requests: %d => %+v", requestCount, getDisplayFields(poolStats))
	}
}

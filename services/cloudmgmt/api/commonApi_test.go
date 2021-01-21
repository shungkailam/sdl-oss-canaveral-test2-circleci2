package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"context"
	"github.com/stretchr/testify/require"
	"os"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestCommonApi(t *testing.T) {
	t.Parallel()
	t.Log("running TestCategory test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	tenantId := base.GetUUID()
	authContext := &base.AuthContext{
		TenantID: tenantId,
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	t.Run("GetAggregate", func(t *testing.T) {
		t.Log("running GetAggregate test")

		err := dbAPI.GetAggregate(ctx, "data_stream_model", "transformation_args_list", os.Stdout)
		require.NoError(t, err)
	})
}

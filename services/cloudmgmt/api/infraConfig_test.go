package api_test

import (
	"cloudservices/common/model"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally

func TestInfraConfig(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeDevice test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	num_devices := 4
	edgeDevices := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	}, "edge", num_devices)
	edgeClusterID := edgeDevices[0].ClusterID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test GetInfraConfig ", func(t *testing.T) {
		// Mark all the devices onboarded
		for i := 0; i < len(edgeDevices); i++ {
			err := dbAPI.UpdateEdgeDeviceOnboarded(ctx1, edgeDevices[i].ID, "fake-ssh-key")
			require.NoError(t, err)
			t.Logf("cluster ID %s", edgeDevices[i].ClusterID)
			t.Logf("device IP  %s", edgeDevices[i].IPAddress)
		}
		infraConfig, err := dbAPI.GetInfraConfig(ctx1, edgeDevices[0].ClusterID)
		require.NoError(t, err)

		expectedK8Config := fmt.Sprintf(`nodes:
- address: 1.1.1.0
  role:
  - controlplane
  - etcd
  - worker
  user: admin
- address: 1.1.1.1
  role:
  - controlplane
  - etcd
  - worker
  user: admin
- address: 1.1.1.2
  role:
  - controlplane
  - etcd
  - worker
  user: admin
- address: 1.1.1.3
  role:
  - worker
  user: admin
ssh_key_path: /home/admin/.ssh/id_rsa
ssh_agent_auth: false
ignore_docker_version: false
cluster_name: %s
`, edgeDevices[0].ClusterID)

		if infraConfig.K8sConfig.ProviderConfig != expectedK8Config {
			t.Fatalf("Unexpected K8 config ")
		}
	})
}

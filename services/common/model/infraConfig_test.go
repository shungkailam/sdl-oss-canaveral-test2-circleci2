package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestInfraConfig will test InfraConfig struct
func TestInfraConfig(t *testing.T) {

	infraConfigs := []model.InfraConfig{
		{
			K8sConfig: model.K8sConfig{
				ProviderType: "RKE",
				ProviderConfig: `nodes:
  - address: 1.2.3.4
    user: ubuntu
    role:
      - controlplane
      - etcd
      - worker`,
			},
			ClusterConfig: model.ClusterConfig{
				FloatingIP: "1.2.3.4",
			},
		},
	}
	infraConfigStrings := []string{
		`{"clusterConfig":{"floatingIP":"1.2.3.4"},"k8sConfig":{"providerType":"RKE","providerConfig":"nodes:\n  - address: 1.2.3.4\n    user: ubuntu\n    role:\n      - controlplane\n      - etcd\n      - worker"}}`,
	}
	infraConfigMaps := []map[string]interface{}{{
		"clusterConfig": map[string]interface{}{
			"floatingIP": "1.2.3.4"}, "k8sConfig": map[string]interface{}{"providerConfig": `nodes:
  - address: 1.2.3.4
    user: ubuntu
    role:
      - controlplane
      - etcd
      - worker`,
			"providerType": "RKE"},
	},
	}

	for i, infraConfig := range infraConfigs {
		infraConfigData, err := json.Marshal(infraConfig)
		require.NoError(t, err, "failed to marshal infraConfig")
		if infraConfigStrings[i] != string(infraConfigData) {
			t.Fatalf("infraConfig json string mismatch: %s", string(infraConfigData))
		}

		m := map[string]interface{}{}
		err = json.Unmarshal(infraConfigData, &m)
		require.NoError(t, err, "failed to unmarshal infraConfig to map")
		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &infraConfigMaps[i]) {
			t.Logf("%+v", infraConfigMaps[i])
			t.Fatalf("infraConfig map marshal mismatch: %+v", m)
		}
	}

}

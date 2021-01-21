package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestNodeInfo will test NodeInfo struct
func TestNodeInfo(t *testing.T) {
	numCPU := "8"
	totalMemory := "3000"
	freeMemory := "200"

	var version float64 = 5
	now := timeNow(t)
	nodeInfos := []model.NodeInfo{
		{
			NodeEntityModel: model.NodeEntityModel{
				ServiceDomainEntityModel: model.ServiceDomainEntityModel{
					BaseModel: model.BaseModel{
						ID:        "node-id",
						TenantID:  "tenant-id",
						Version:   5,
						CreatedAt: now,
						UpdatedAt: now,
					},
					SvcDomainID: "service-domain-id",
				},
				NodeID: "node-id",
			},
			NodeInfoCore: model.NodeInfoCore{
				NumCPU:        numCPU,
				TotalMemoryKB: totalMemory,
				MemoryFreeKB:  freeMemory,
			},
			NodeStatus: model.NodeStatus{
				HealthBits: map[string]bool{
					"bit1": true,
					"bit2": false,
				},
			},
			Artifacts: map[string]interface{}{"nodeIP": "123.321.123.321"},
		},
	}

	nodeInfoStrings := []string{
		`{"id":"node-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","svcDomainId":"service-domain-id","nodeId":"node-id","numCpu":"8","totalMemoryKB":"3000","memoryFreeKB":"200","healthBits":{"bit1":true,"bit2":false},"artifacts":{"nodeIP":"123.321.123.321"}}`,
	}

	nodeInfoMaps := []map[string]interface{}{
		{
			"id":            "node-id",
			"tenantId":      "tenant-id",
			"version":       version,
			"numCpu":        numCPU,
			"totalMemoryKB": totalMemory,
			"memoryFreeKB":  freeMemory,
			"svcDomainId":   "service-domain-id",
			"nodeId":        "node-id",
			"createdAt":     NOW,
			"updatedAt":     NOW,
			"healthBits": map[string]bool{
				"bit1": true,
				"bit2": false,
			},
			"artifacts": map[string]interface{}{"nodeIP": "123.321.123.321"},
		},
	}
	for i, nodeInfo := range nodeInfos {
		nodeData, err := json.Marshal(nodeInfo)
		require.NoError(t, err, "failed to marshal node")
		t.Logf("node json: %s", string(nodeData))
		if nodeInfoStrings[i] != string(nodeData) {
			t.Fatalf("node json string mismatch: %s", string(nodeData))
		}
		// alternative form: m := make(map[string]interface{})
		// JSON of maps are ordered
		m := map[string]interface{}{}
		err = json.Unmarshal(nodeData, &m)
		require.NoError(t, err, "failed to unmarshal node to map")
		if !model.MarshalEqual(&m, &nodeInfoMaps[i]) {
			t.Fatalf("node map mismatch: %+v", m)
		}
	}
}

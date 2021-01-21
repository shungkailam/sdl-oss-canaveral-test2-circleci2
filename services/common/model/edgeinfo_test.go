package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestEdgeInfo will test EdgeInfo struct
func TestEdgeInfo(t *testing.T) {
	numCPU := "8"
	totalMemory := "3000"
	freeMemory := "200"

	var version float64 = 5
	now := timeNow(t)
	edgeinfos := []model.EdgeUsageInfo{
		{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:        "edge-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				EdgeID: "edge-id",
			},
			EdgeInfo: model.EdgeInfo{
				NumCPU:        numCPU,
				TotalMemoryKB: totalMemory,
				MemoryFreeKB:  freeMemory,
			},
			EdgeArtifacts: map[string]interface{}{"edgeIP": "123.321.123.321"},
		},
	}

	edgeInfoStrings := []string{
		`{"id":"edge-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","edgeId":"edge-id","NumCPU":"8","TotalMemoryKB":"3000","MemoryFreeKB":"200","edgeArtifacts":{"edgeIP":"123.321.123.321"}}`,
	}

	edgeInfoMaps := []map[string]interface{}{
		{
			"id":            "edge-id",
			"tenantId":      "tenant-id",
			"version":       version,
			"NumCPU":        numCPU,
			"TotalMemoryKB": totalMemory,
			"MemoryFreeKB":  freeMemory,
			"edgeId":        "edge-id",
			"createdAt":     NOW,
			"updatedAt":     NOW,
			"edgeArtifacts": map[string]interface{}{"edgeIP": "123.321.123.321"},
		},
	}
	for i, edgeinfo := range edgeinfos {
		edgeData, err := json.Marshal(edgeinfo)
		require.NoError(t, err, "failed to marshal edge")

		t.Logf("edge json: %s", string(edgeData))
		if edgeInfoStrings[i] != string(edgeData) {
			t.Fatalf("edge json string mismatch: %s", string(edgeData))
		}
		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(edgeData, &m)
		require.NoError(t, err, "failed to unmarshal edge to map")

		// Copied from dataSource_test.go
		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &edgeInfoMaps[i]) {
			t.Fatalf("edge map mismatch: %+v", m)
		}
	}
}

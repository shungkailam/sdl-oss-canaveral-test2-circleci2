package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

// TestEdgeCluster will test EdgeCluster struct
func TestEdgeCluster(t *testing.T) {
	var version float64 = 5
	now := timeNow(t)
	edgeClusters := []model.EdgeCluster{
		{
			BaseModel: model.BaseModel{
				ID:        "edge-cluster-id",
				TenantID:  "tenant-id",
				Version:   version,
				CreatedAt: now,
				UpdatedAt: now,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name: "edge-cluster-name",
			},
			Labels: nil,
		},
	}

	edgeClusterStrings := []string{
		`{"id":"edge-cluster-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"edge-cluster-name","shortId":null,"virtualIp":null,"description":"","labels":null}`,
	}

	edgeClusterMaps := []map[string]interface{}{
		{
			"id":          "edge-cluster-id",
			"tenantId":    "tenant-id",
			"version":     version,
			"name":        "edge-cluster-name",
			"description": "",
			"shortId":     nil,
			"labels":      nil,
			"virtualIp":   nil,
			"createdAt":   NOW,
			"updatedAt":   NOW,
		},
	}
	for i, edgeCluster := range edgeClusters {
		edgeClusterData, err := json.Marshal(edgeCluster)
		require.NoError(t, err, "failed to marshal edge cluster")

		t.Logf("edge cluster json: %s", string(edgeClusterData))
		if !reflect.DeepEqual(edgeClusterData, []byte(edgeClusterStrings[i])) {
			t.Fatalf("edge cluster json string mismatch: %s\n%sn", string(edgeClusterData), edgeClusterStrings[i])
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(edgeClusterData, &m)
		require.NoError(t, err, "failed to unmarshal edge cluster to map")

		if !reflect.DeepEqual(m, edgeClusterMaps[i]) {
			t.Fatalf("expected %+v, but got %+v", edgeClusterMaps[i], m)
		}
	}
}

type edgeClusterResp struct {
	StatusCode int                `json:"statusCode"`
	Doc        *model.EdgeCluster `json:"doc"`
}

func TestEdgeClusterPtr(t *testing.T) {
	var edgeCluster = model.EdgeCluster{
		BaseModel: model.BaseModel{
			ID:       "edge-id",
			TenantID: "tenant-id",
			Version:  5,
		},
		EdgeClusterCore: model.EdgeClusterCore{
			Name: "edge-cluster-name",
		},
	}
	er1 := edgeClusterResp{
		StatusCode: 200,
		Doc:        &edgeCluster,
	}
	er2 := edgeClusterResp{
		StatusCode: 500,
		Doc:        nil,
	}
	ers1, err := json.Marshal(er1)
	require.NoError(t, err, "failed to marshal er1")

	ers2, err := json.Marshal(er2)
	require.NoError(t, err, "failed to marshal er2")

	t.Logf("er1 marshal to %s", string(ers1))
	t.Logf("er2 marshal to %s", string(ers2))
}

func TestEdgeClusterValidation(t *testing.T) {
	var edgeCluster = model.EdgeCluster{
		BaseModel: model.BaseModel{
			ID:       "edge-id",
			TenantID: "tenant-id",
			Version:  5,
		},
		EdgeClusterCore: model.EdgeClusterCore{
			Name: "edge-cluster-name",
		},
	}
	err := model.ValidateEdgeCluster(&edgeCluster)
	require.NoError(t, err)
	goodNames := []string{
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-82",
		"0123",
		"a.b.c",
		"foo.com",
		"foo-bar.baz",
		"a.b.c.d.e.f",
	}
	for _, name := range goodNames {
		edgeCluster.Name = name
		err = model.ValidateEdgeCluster(&edgeCluster)
		require.NoError(t, err)
		t.Logf("validating edgeCluster name %s length: %d", edgeCluster.Name, len(edgeCluster.Name))
		err = base.ValidateStruct("Name", &edgeCluster, "create")
		require.NoError(t, err, "bad create edgeCluster name: %s", edgeCluster.Name)

		err = base.ValidateStruct("Name", &edgeCluster, "update")
		require.NoError(t, err, "bad update edgeCluster name: %s", edgeCluster.Name)
	}
	badNames := []string{
		"-abcd",
		"abc-",
		"ab c",
		"ab=c",
		"ab,c",
		"abc.",
		"ab.-cd",
		"TestEdge",
		"My Edge 2",
	}
	for _, name := range badNames {
		edgeCluster.Name = name
		err = model.ValidateEdgeCluster(&edgeCluster)
		require.Errorf(t, err, "expect bad name to fail validation: %s", name)
	}
	longNames := []string{
		// too long - max = 60
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-823",
	}
	for _, name := range longNames {
		edgeCluster.Name = name
		err = base.ValidateStruct("Name", &edgeCluster, "create")
		require.Errorf(t, err, "expect long name to fail create validation: %s", name)
		err = base.ValidateStruct("Name", &edgeCluster, "update")
		require.Errorf(t, err, "expect long name to fail update validation: %s", name)
	}
}

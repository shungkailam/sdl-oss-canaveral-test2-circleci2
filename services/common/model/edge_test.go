package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

// TestEdge will test Edge struct
func TestEdge(t *testing.T) {
	var edgeDevices float64 = 3
	var storageCapacity float64 = 100
	var storageUsage float64 = 80
	var version float64 = 5
	now := timeNow(t)
	edges := []model.Edge{
		{
			BaseModel: model.BaseModel{
				ID:        "edge-id",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			EdgeCore: model.EdgeCore{
				EdgeCoreCommon: model.EdgeCoreCommon{
					Name:         "edge-name",
					SerialNumber: "edge-serial-number",
					IPAddress:    "1.1.1.1",
					Subnet:       "255.255.255.0",
					Gateway:      "1.1.1.1",
					EdgeDevices:  edgeDevices,
				},
				StorageCapacity: storageCapacity,
				StorageUsage:    storageUsage,
			},
			Connected: true,
			Labels:    nil,
		},
	}

	edgeStrings := []string{
		`{"id":"edge-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"edge-name","serialNumber":"edge-serial-number","ipAddress":"1.1.1.1","gateway":"1.1.1.1","subnet":"255.255.255.0","edgeDevices":3,"shortId":null,"storageCapacity":100,"storageUsage":80,"connected":true,"description":"","labels":null}`,
	}

	edgeMaps := []map[string]interface{}{
		{
			"id":              "edge-id",
			"tenantId":        "tenant-id",
			"version":         version,
			"name":            "edge-name",
			"description":     "",
			"serialNumber":    "edge-serial-number",
			"subnet":          "255.255.255.0",
			"storageCapacity": storageCapacity,
			"connected":       true,
			"ipAddress":       "1.1.1.1",
			"gateway":         "1.1.1.1",
			"edgeDevices":     edgeDevices,
			"storageUsage":    storageUsage,
			"labels":          nil,
			"createdAt":       NOW,
			"updatedAt":       NOW,
			"shortId":         nil,
		},
	}
	for i, edge := range edges {
		edgeData, err := json.Marshal(edge)
		require.NoError(t, err, "failed to marshal edge")

		t.Logf("edge json: %s", string(edgeData))
		if !reflect.DeepEqual(edgeData, []byte(edgeStrings[i])) {
			t.Fatalf("edge json string mismatch: %s", string(edgeData))
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(edgeData, &m)
		require.NoError(t, err, "failed to unmarshal edge to map")
		if !reflect.DeepEqual(m, edgeMaps[i]) {
			t.Fatalf("expected %+v, but got %+v", edgeMaps[i], m)
		}
	}
}

type edgeResp struct {
	StatusCode int         `json:"statusCode"`
	Doc        *model.Edge `json:"doc"`
}

func TestEdgePtr(t *testing.T) {
	var edgeDevices float64 = 3
	var storageCapacity float64 = 100
	var storageUsage float64 = 80
	// var version float64 = 5
	var edge = model.Edge{
		BaseModel: model.BaseModel{
			ID:       "edge-id",
			TenantID: "tenant-id",
			Version:  5,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         "edge-name",
				SerialNumber: "edge-serial-number",
				IPAddress:    "1.1.1.1",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
				EdgeDevices:  edgeDevices,
			},
			StorageCapacity: storageCapacity,
			StorageUsage:    storageUsage,
		},
		Connected: true,
	}
	er1 := edgeResp{
		StatusCode: 200,
		Doc:        &edge,
	}
	er2 := edgeResp{
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

func TestEdgeValidation(t *testing.T) {
	var edgeDevices float64 = 3
	var storageCapacity float64 = 100
	var storageUsage float64 = 80
	var serialNumber = "Edge-Serial-Number"
	var edge = model.Edge{
		BaseModel: model.BaseModel{
			ID:       "edge-id",
			TenantID: "tenant-id",
			Version:  5,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         "edge-name",
				SerialNumber: serialNumber,
				IPAddress:    "1.1.1.1",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
				EdgeDevices:  edgeDevices,
			},
			StorageCapacity: storageCapacity,
			StorageUsage:    storageUsage,
		},
		Connected: true,
	}
	err := model.ValidateEdge(&edge)
	require.NoError(t, err)
	if edge.SerialNumber != serialNumber {
		t.Fatalf("expect validate edge to not change serial number: %s", edge.SerialNumber)
	}
	goodNames := []string{
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-82",
		"0123",
		"a.b.c",
		"foo.com",
		"foo-bar.baz",
		"a.b.c.d.e.f",
	}
	for _, name := range goodNames {
		edge.Name = name
		err = model.ValidateEdge(&edge)
		require.NoError(t, err)
		t.Logf("validating edge name %s length: %d", edge.Name, len(edge.Name))
		err = base.ValidateStruct("Name", &edge, "create")
		require.NoErrorf(t, err, "bad create edge name: %s", edge.Name)

		err = base.ValidateStruct("Name", &edge, "update")
		require.NoErrorf(t, err, "bad update edge name: %s", edge.Name)
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
		edge.Name = name
		err = model.ValidateEdge(&edge)
		require.Errorf(t, err, "expect bad name to fail validation: %s", name)
	}
	longNames := []string{
		// too long - max = 60
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-823",
	}
	for _, name := range longNames {
		edge.Name = name
		err = base.ValidateStruct("Name", &edge, "create")
		require.Errorf(t, err, "expect long name to fail create validation: %s", name)
		err = base.ValidateStruct("Name", &edge, "update")
		require.Errorf(t, err, "expect long name to fail update validation: %s", name)
	}
}

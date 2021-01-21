package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestEdge will test Edge struct
func TestEdgeDevice(t *testing.T) {
	var version float64 = 5
	now := timeNow(t)
	edgeDevices := []model.EdgeDevice{
		{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					ID:        "edge-device-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				ClusterID: "test-cluster-id",
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         "edge-device-name",
				SerialNumber: "edge-device-serial-number",
				IPAddress:    "1.1.1.1",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
				Role:         &model.NodeRole{Worker: true, Master: true},
			},
		},
	}

	edgeDeviceStrings := []string{
		`{"id":"edge-device-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","clusterId":"test-cluster-id","name":"edge-device-name","serialNumber":"edge-device-serial-number","ipAddress":"1.1.1.1","gateway":"1.1.1.1","subnet":"255.255.255.0","role":{"master":true,"worker":true},"description":""}`,
	}

	edgeDeviceMaps := []map[string]interface{}{
		{
			"id":           "edge-device-id",
			"tenantId":     "tenant-id",
			"version":      version,
			"name":         "edge-device-name",
			"description":  "",
			"serialNumber": "edge-device-serial-number",
			"subnet":       "255.255.255.0",
			"ipAddress":    "1.1.1.1",
			"gateway":      "1.1.1.1",
			"createdAt":    NOW,
			"updatedAt":    NOW,
			"clusterId":    "test-cluster-id",
			"role":         map[string]interface{}{"master": true, "worker": true},
		},
	}
	for i, edgeDevice := range edgeDevices {
		edgeDeviceData, err := json.Marshal(edgeDevice)
		require.NoError(t, err, "failed to marshal edge device")

		t.Logf("edge device json: %s", string(edgeDeviceStrings[i]))
		if !reflect.DeepEqual(edgeDeviceData, []byte(edgeDeviceStrings[i])) {
			t.Fatalf("edge device json string mismatch: \n%s\n %s", string(edgeDeviceData), edgeDeviceStrings[i])
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(edgeDeviceData, &m)
		require.NoError(t, err, "failed to unmarshal edge device to map")

		if !reflect.DeepEqual(m, edgeDeviceMaps[i]) {
			t.Fatalf("expected %+v, but got %+v", edgeDeviceMaps[i], m)
		}
	}
}

type edgeDeviceResp struct {
	StatusCode int               `json:"statusCode"`
	Doc        *model.EdgeDevice `json:"doc"`
}

func TestEdgeDevicePtr(t *testing.T) {
	var edgeDevice = model.EdgeDevice{
		ClusterEntityModel: model.ClusterEntityModel{
			BaseModel: model.BaseModel{
				ID:       "edgedevice-id",
				TenantID: "tenant-id",
				Version:  5,
			},
			ClusterID: "test-cluster-id",
		},
		EdgeDeviceCore: model.EdgeDeviceCore{
			Name:         "edge-device-name",
			SerialNumber: "edge-device-serial-number",
			IPAddress:    "1.1.1.1",
			Subnet:       "255.255.255.0",
			Gateway:      "1.1.1.1",
			Role:         &model.NodeRole{Worker: true, Master: true},
		},
	}
	er1 := edgeDeviceResp{
		StatusCode: 200,
		Doc:        &edgeDevice,
	}
	er2 := edgeDeviceResp{
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

func TestEdgeDeviceValidation(t *testing.T) {
	var serialNumber = "Edge-Serial-Number"
	var edgeDevice = model.EdgeDevice{
		ClusterEntityModel: model.ClusterEntityModel{
			BaseModel: model.BaseModel{
				ID:       "edge-device-id",
				TenantID: "tenant-id",
				Version:  5,
			},
			ClusterID: "test-cluster-id",
		},
		EdgeDeviceCore: model.EdgeDeviceCore{
			Name:         "edge-device-name",
			SerialNumber: serialNumber,
			IPAddress:    "1.1.1.1",
			Subnet:       "255.255.255.0",
			Gateway:      "1.1.1.1",
			Role:         &model.NodeRole{Worker: true, Master: true},
		},
	}
	err := model.ValidateEdgeDevice(&edgeDevice)
	require.NoError(t, err)
	if edgeDevice.SerialNumber != serialNumber {
		t.Fatalf("expect validate edge to not change serial number: %s", edgeDevice.SerialNumber)
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
		edgeDevice.Name = name
		err = model.ValidateEdgeDevice(&edgeDevice)
		require.NoError(t, err)
		t.Logf("validating edge device name %s length: %d", edgeDevice.Name, len(edgeDevice.Name))
		err = base.ValidateStruct("Name", &edgeDevice, "create")
		require.NoError(t, err, "bad create edge device name: %s", edgeDevice.Name)

		err = base.ValidateStruct("Name", &edgeDevice, "update")
		require.NoErrorf(t, err, "bad update edge name: %s", edgeDevice.Name)
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
		edgeDevice.Name = name
		err = model.ValidateEdgeDevice(&edgeDevice)
		require.Errorf(t, err, "expect bad name to fail validation: %s", name)
	}
	longNames := []string{
		// too long - max = 60
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-823",
	}
	for _, name := range longNames {
		edgeDevice.Name = name
		err = base.ValidateStruct("Name", &edgeDevice, "create")
		require.Errorf(t, err, "expect long name to fail create validation: %s", name)
		err = base.ValidateStruct("Name", &edgeDevice, "update")
		require.Errorf(t, err, "expect long name to fail update validation: %s", name)
	}
}

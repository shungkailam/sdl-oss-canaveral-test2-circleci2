package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestNode will test Node struct
func TestNode(t *testing.T) {
	var version float64 = 5
	now := timeNow(t)
	nodes := []model.Node{
		{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:        "node-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				SvcDomainID: "test-service-domain-id",
			},
			NodeCore: model.NodeCore{
				Name:         "node-name",
				SerialNumber: "node-serial-number",
				IPAddress:    "1.1.1.1",
				Subnet:       "255.255.255.0",
				Gateway:      "1.1.1.1",
			},
		},
	}

	nodeStrings := []string{
		`{"id":"node-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","svcDomainId":"test-service-domain-id","name":"node-name","serialNumber":"node-serial-number","ipAddress":"1.1.1.1","gateway":"1.1.1.1","subnet":"255.255.255.0","isBootstrapMaster":false,"description":""}`,
	}

	nodeMaps := []map[string]interface{}{
		{
			"id":                "node-id",
			"tenantId":          "tenant-id",
			"version":           version,
			"name":              "node-name",
			"description":       "",
			"serialNumber":      "node-serial-number",
			"subnet":            "255.255.255.0",
			"isBootstrapMaster": false,
			"ipAddress":         "1.1.1.1",
			"gateway":           "1.1.1.1",
			"createdAt":         NOW,
			"updatedAt":         NOW,
			"svcDomainId":       "test-service-domain-id",
		},
	}
	for i, node := range nodes {
		nodeData, err := json.Marshal(node)
		require.NoError(t, err, "failed to marshal node")
		t.Logf("node json: %s", string(nodeStrings[i]))
		if !reflect.DeepEqual(nodeData, []byte(nodeStrings[i])) {
			t.Fatalf("node json string mismatch: \n%s\n %s", string(nodeData), nodeStrings[i])
		}
		m := make(map[string]interface{})
		err = json.Unmarshal(nodeData, &m)
		require.NoError(t, err, "failed to unmarshal node to map")
		if !reflect.DeepEqual(m, nodeMaps[i]) {
			t.Fatalf("expected %+v, but got %+v", nodeMaps[i], m)
		}
	}
}

type nodeResp struct {
	StatusCode int         `json:"statusCode"`
	Doc        *model.Node `json:"doc"`
}

func TestNodePtr(t *testing.T) {
	var node = model.Node{
		ServiceDomainEntityModel: model.ServiceDomainEntityModel{
			BaseModel: model.BaseModel{
				ID:       "node--id",
				TenantID: "tenant-id",
				Version:  5,
			},
			SvcDomainID: "test-service-domain-id",
		},
		NodeCore: model.NodeCore{
			Name:         "node-name",
			SerialNumber: "node-serial-number",
			IPAddress:    "1.1.1.1",
			Subnet:       "255.255.255.0",
			Gateway:      "1.1.1.1",
		},
	}
	er1 := nodeResp{
		StatusCode: 200,
		Doc:        &node,
	}
	er2 := nodeResp{
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

func TestNodeValidation(t *testing.T) {
	var serialNumber = "Node-Serial-Number"
	var node = model.Node{
		ServiceDomainEntityModel: model.ServiceDomainEntityModel{
			BaseModel: model.BaseModel{
				ID:       "node-id",
				TenantID: "tenant-id",
				Version:  5,
			},
			SvcDomainID: "test-service-domain-id",
		},
		NodeCore: model.NodeCore{
			Name:         "node-name",
			SerialNumber: serialNumber,
			IPAddress:    "1.1.1.1",
			Subnet:       "255.255.255.0",
			Gateway:      "1.1.1.1",
		},
	}
	err := model.ValidateNode(&node)
	require.NoError(t, err)
	if node.SerialNumber != serialNumber {
		t.Fatalf("expect validate node to not change serial number: %s", node.SerialNumber)
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
		node.Name = name
		err = model.ValidateNode(&node)
		require.NoError(t, err)
		t.Logf("validating node name %s length: %d", node.Name, len(node.Name))
		err = base.ValidateStruct("Name", &node, "create")
		require.NoError(t, err, "bad create node name: %s", node.Name)

		err = base.ValidateStruct("Name", &node, "update")
		require.NoError(t, err, "bad update node name: %s", node.Name)
	}
	badNames := []string{
		"-abcd",
		"abc-",
		"ab c",
		"ab=c",
		"ab,c",
		"abc.",
		"ab.-cd",
		"TestNode",
		"My Node 2",
	}
	for _, name := range badNames {
		node.Name = name
		err = model.ValidateNode(&node)
		require.Errorf(t, err, "expect bad name to fail validation: %s", name)
	}
	longNames := []string{
		// too long - max = 60
		"sherlock-test-master-shyan-ming-perng-2018-08-31-10-23-37-823",
	}
	for _, name := range longNames {
		node.Name = name
		err = base.ValidateStruct("Name", &node, "create")
		require.Errorf(t, err, "expect long name to fail create validation: %s", name)
		err = base.ValidateStruct("Name", &node, "update")
		require.Errorf(t, err, "expect long name to fail update validation: %s", name)
	}
}

package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestDataSource will test DataSource struct
func TestDataSource(t *testing.T) {
	now := timeNow(t)
	dataSources := []model.DataSource{
		{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:        "dataSource-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				EdgeID: "edge-id",
			},
			DataSourceCore: model.DataSourceCore{
				Name:       "dataSource-name",
				Type:       "Sensor",
				Connection: "Secure",
				Selectors: []model.DataSourceFieldSelector{
					{
						CategoryInfo: model.CategoryInfo{
							ID:    "cat-id",
							Value: "cat-val",
						},
						Scope: []string{
							"__ALL__",
						},
					},
				},
				Protocol: "MQTT",
				AuthType: "CERTIFICATE",
			},
			Fields: []model.DataSourceFieldInfo{
				{
					DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
						Name:      "field-name-1",
						FieldType: "field-type-1",
					},
					MQTTTopic: "mqtt-topic-1",
				},
				{
					DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
						Name:      "field-name-2",
						FieldType: "field-type-2",
					},
					MQTTTopic: "mqtt-topic-2",
				},
			},
			SensorModel: "Model 3",
		},
	}
	dataSourceStrings := []string{
		`{"id":"dataSource-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","edgeId":"edge-id","name":"dataSource-name","type":"Sensor","connection":"Secure","selectors":[{"id":"cat-id","value":"cat-val","scope":["__ALL__"]}],"protocol":"MQTT","authType":"CERTIFICATE","ifcInfo":null,"fields":[{"name":"field-name-1","fieldType":"field-type-1","mqttTopic":"mqtt-topic-1"},{"name":"field-name-2","fieldType":"field-type-2","mqttTopic":"mqtt-topic-2"}],"sensorModel":"Model 3"}`,
	}

	/*
		map[  selectors:[map[id:cat-id value:cat-val scope:[__ALL__]]] id:dataSource-id tenantId:tenant-id edgeId:edge-id name:dataSource-name type:Sensor]
	*/

	var version float64 = 5
	dataSourceMaps := []map[string]interface{}{
		{
			"id":          "dataSource-id",
			"tenantId":    "tenant-id",
			"version":     version,
			"edgeId":      "edge-id",
			"name":        "dataSource-name",
			"type":        "Sensor",
			"sensorModel": "Model 3",
			"connection":  "Secure",
			"ifcInfo":     nil,
			"fields": []map[string]interface{}{
				{
					"name":      "field-name-1",
					"mqttTopic": "mqtt-topic-1",
					"fieldType": "field-type-1",
				},
				{
					"name":      "field-name-2",
					"mqttTopic": "mqtt-topic-2",
					"fieldType": "field-type-2",
				},
			},
			"selectors": []map[string]interface{}{
				{
					"id":    "cat-id",
					"value": "cat-val",
					"scope": []string{
						"__ALL__",
					},
				},
			},
			"protocol":  "MQTT",
			"authType":  "CERTIFICATE",
			"createdAt": NOW,
			"updatedAt": NOW,
		},
	}

	for i, dataSource := range dataSources {
		err := model.ValidateDataSource(&dataSource)
		require.NoError(t, err)
		dataSourceData, err := json.Marshal(dataSource)
		require.NoError(t, err, "failed to marshal dataSource")

		if dataSourceStrings[i] != string(dataSourceData) {
			t.Fatalf("dataSource json string mismatch Found: [%s], Expected[%s]", string(dataSourceData), dataSourceStrings[i])
		}

		var doc interface{}
		doc = dataSource
		_, ok := doc.(model.ProjectScopedEntity)
		require.False(t, ok, "data source should not be a project scoped entity")

		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(dataSourceData, &m)
		require.NoError(t, err, "failed to unmarshal dataSource to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &dataSourceMaps[i]) {
			// t.Logf("dataSource map marshal mismatch 1: %s\n", dataSourceMaps[i])
			// for k, v := range dataSourceMaps[i] {
			// 	if !reflect.DeepEqual(v, m[k]) {
			// 		if !model.MarshalEqual(v, m[k]) {
			// 			t.Logf("mismatch k=%s, v=%s, m[k]=%s\n", k, v, m[k])
			// 		}
			// 	}
			// }
			t.Fatalf("dataSource map marshal mismatch 2: %s", m)
		}
	}

	ds := dataSources[0]
	eid := model.GetEdgeID(ds)
	if eid == nil {
		t.Fatalf("Failed to find edge id via reflection")
	}

}

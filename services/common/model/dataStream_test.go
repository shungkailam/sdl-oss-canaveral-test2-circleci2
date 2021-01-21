package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

// TestDataStream will test DataStream struct
func TestDataStream(t *testing.T) {
	var size float64 = 1000000
	now := timeNow(t)
	dataStreams := []model.DataStream{
		{
			BaseModel: model.BaseModel{
				ID:        "dataStreams-id",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     "dataStreams-name",
			DataType: "Image",
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    "cat-id",
					Value: "cat-val",
				},
			},
			// OriginID:       "",
			Destination:    "Cloud",
			CloudType:      "AWS",
			CloudCredsID:   "cloud-creds-id",
			AWSCloudRegion: "us-west-2",
			// GCPCloudRegion:   "",
			// EdgeStreamType:   "",
			AWSStreamType: "Kafka",
			// GCPStreamType:    "",
			Size:           size,
			EnableSampling: false,
			// SamplingInterval: 0,
			TransformationArgsList: []model.TransformationArgs{
				{
					TransformationID: "trans-id-1",
					Args:             []model.ScriptParamValue{},
				},
			},
			DataRetention: []model.RetentionInfo{},
			ProjectID:     "proj-id",
		},
	}
	dataStreamStrings := []string{
		`{"id":"dataStreams-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"dataStreams-name","description":"","dataType":"Image","origin":"DataSource","originSelectors":[{"id":"cat-id","value":"cat-val"}],"destination":"Cloud","cloudType":"AWS","cloudCredsId":"cloud-creds-id","awsCloudRegion":"us-west-2","awsStreamType":"Kafka","size":1000000,"enableSampling":false,"transformationArgsList":[{"transformationId":"trans-id-1","args":[]}],"dataRetention":[],"projectId":"proj-id","DataIfcEndpoints":null}`,
	}

	var version float64 = 5
	dataStreamMaps := []map[string]interface{}{
		{
			"id":       "dataStreams-id",
			"tenantId": "tenant-id",
			"version":  version,
			"name":     "dataStreams-name",
			"dataType": "Image",
			"origin":   "DataSource",
			"originSelectors": []map[string]interface{}{
				{
					"id":    "cat-id",
					"value": "cat-val",
				},
			},
			// "originId":       "",
			"destination":    "Cloud",
			"cloudType":      "AWS",
			"cloudCredsId":   "cloud-creds-id",
			"awsCloudRegion": "us-west-2",
			// "gcpCloudRegion":   "",
			// "edgeStreamType":   "",
			"awsStreamType": "Kafka",
			// "gcpStreamType":    "",
			"size":           size,
			"enableSampling": false,
			// "samplingInterval": 0.,
			"transformationArgsList": []map[string]interface{}{
				{
					"transformationId": "trans-id-1",
					"args":             []string{},
				},
			},
			"dataRetention": []model.RetentionInfo{},
		},
	}

	for i, dataStream := range dataStreams {
		var doc interface{}
		doc = dataStream
		_, ok := doc.(model.ProjectScopedEntity)
		if !ok {
			t.Fatal("data stream should be a project scoped entity")
		}

		dataStreamData, err := json.Marshal(dataStream)
		require.NoError(t, err, "failed to marshal dataStream")

		if dataStreamStrings[i] != string(dataStreamData) {
			t.Fatalf("dataStream json string mismatch: %s", string(dataStreamData))
		}
		m := map[string]interface{}{}
		err = json.Unmarshal(dataStreamData, &m)
		require.NoError(t, err, "failed to unmarshal dataStream to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &dataStreamMaps[i]) {
			ok := true
			// t.Logf("dataStream map marshal mismatch 1: %s\n", dataStreamMaps[i])
			for k, v := range dataStreamMaps[i] {
				if !reflect.DeepEqual(v, m[k]) {
					if !model.MarshalEqual(v, m[k]) {
						t.Logf(">>> mismatch k=%s, v=%s, m[k]=%s", k, v, m[k])
						ok = false
					} else {
						t.Logf(">>> marshal equal: %s", k)
					}
				} else {
					t.Logf(">>> deep equal: %s", k)
				}
			}
			if !ok {
				t.Fatalf("dataStream map marshal mismatch 2: %s", m)
			}
		}
	}
}

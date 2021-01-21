package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

// TestContainerRegistry will test ContainerRegistryV2 struct
func TestContainerRegistry(t *testing.T) {
	now := timeNow(t)
	containerRegisties := []model.ContainerRegistryV2{
		{
			BaseModel: model.BaseModel{
				ID:        "containerRegistry-id-1",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:        "containerRegistry-name",
			Type:        "AWS",
			Server:      "a.b.c.d.e.f",
			Description: "dockerProfileDesc",
			CloudProfileInfo: &model.CloudProfileInfo{
				CloudCredsID: "cloudCredsID",
			},
		},
	}
	containerRegistryStrings := []string{
		`{"id":"containerRegistry-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"containerRegistry-name","description":"dockerProfileDesc","type":"AWS","server":"a.b.c.d.e.f","CloudProfileInfo":{"cloudCredsID":"cloudCredsID","email":""}}`,
	}

	var version float64 = 5
	containerRegistryMaps := []map[string]interface{}{
		{

			"id":          "containerRegistry-id-1",
			"tenantId":    "tenant-id",
			"version":     version,
			"createdAt":   NOW,
			"updatedAt":   NOW,
			"name":        "containerRegistry-name",
			"type":        "AWS",
			"server":      "a.b.c.d.e.f",
			"description": "dockerProfileDesc",
			"CloudProfileInfo": map[string]interface{}{
				"cloudCredsID": "cloudCredsID",
				"email":        "",
			},
		},
	}

	for i, containerRegistry := range containerRegisties {
		containerRegistryData, err := json.Marshal(containerRegistry)
		require.NoError(t, err, "failed to marshal containerRegistry")

		if containerRegistryStrings[i] != string(containerRegistryData) {
			t.Fatalf("containerRegistry json string mismatch: %s", string(containerRegistryData))
		}
		m := map[string]interface{}{}
		err = json.Unmarshal(containerRegistryData, &m)
		require.NoError(t, err, "failed to unmarshal containerRegistry to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &containerRegistryMaps[i]) {
			ok := true
			// t.Logf("containerRegistry map marshal mismatch 1: %s\n", containerRegistryMaps[i])
			for k, v := range containerRegistryMaps[i] {
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
				t.Fatalf("containerRegistry map marshal mismatch 2: %s\n%s", m, containerRegistryMaps[i])
			}
		}
	}
}

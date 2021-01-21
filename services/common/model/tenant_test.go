package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

// TestTenant will test Tenant struct
func TestTenant(t *testing.T) {
	now := timeNow(t)
	tenants := []model.Tenant{
		{
			ID:          "tenant-id",
			Version:     0,
			Name:        "tenant-name",
			Token:       "tenant-token",
			Description: "My company",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
		{
			ID:          "tenant-id2",
			Version:     2,
			Name:        "tenant-name2",
			Token:       "tenant-token2",
			Description: "My company",
			CreatedAt:   now,
			UpdatedAt:   now,
		},
	}

	tenantStrings := []string{
		`{"id":"tenant-id","name":"tenant-name","token":"tenant-token","description":"My company","profile":null,"createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z"}`,
		`{"id":"tenant-id2","version":2,"name":"tenant-name2","token":"tenant-token2","description":"My company","profile":null,"createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z"}`,
	}

	var version float64 = 2
	tenantMaps := []map[string]interface{}{
		{
			"id":          "tenant-id",
			"name":        "tenant-name",
			"token":       "tenant-token",
			"createdAt":   NOW,
			"updatedAt":   NOW,
			"description": "My company",
			"profile":     nil,
		},
		{
			"id":          "tenant-id2",
			"version":     version,
			"name":        "tenant-name2",
			"token":       "tenant-token2",
			"createdAt":   NOW,
			"updatedAt":   NOW,
			"description": "My company",
			"profile":     nil,
		},
	}
	for i, tenant := range tenants {
		tenantData, err := json.Marshal(tenant)
		require.NoError(t, err, "failed to marshal tenant")
		if tenantStrings[i] != string(tenantData) {
			t.Fatal("tenant json string mismatch", string(tenantData))
		}
		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(tenantData, &m)
		require.NoError(t, err, "failed to unmarshal tenant to map")
		if !reflect.DeepEqual(m, tenantMaps[i]) {
			t.Fatal("tenant map mismatch")
		}
	}
}

package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func makeDataDriverConfig(t *testing.T, name string, params map[string]interface{}) model.DataDriverConfig {
	now := timeNow(t)
	return model.DataDriverConfig{
		BaseModel: model.BaseModel{
			ID:        "di-id-1",
			TenantID:  "tenant-id",
			Version:   5,
			CreatedAt: now,
			UpdatedAt: now,
		},
		Name:                 name,
		DataDriverInstanceID: "di-id-1",
		Description:          "desc-1",
		Parameters:           params,
		ServiceDomainBinding: model.ServiceDomainBinding{
			ServiceDomainIDs: []string{"edge-1"},
		},
	}
}

// TestDatadriverInstance will test DataDriverConfig struct
func TestDatadriverInstanceConfig(t *testing.T) {
	now := timeNow(t)
	c1 := model.CategoryInfo{
		ID:    "cat-id-1",
		Value: "v1",
	}
	dynamicConfigs := []model.DataDriverConfig{
		{
			BaseModel: model.BaseModel{
				ID:        "config-1",
				TenantID:  "tenant-id",
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			DataDriverInstanceID: "di-id-1",
			Name:                 "name-1",
			Description:          "desc-1",
			Parameters: map[string]interface{}{
				"key1": 1,
			},
			ServiceDomainBinding: model.ServiceDomainBinding{
				ServiceDomainSelectors:  []model.CategoryInfo{c1},
				ExcludeServiceDomainIDs: []string{"no-edge-1"},
			},
		},
		{
			BaseModel: model.BaseModel{
				ID:        "config-2",
				TenantID:  "tenant-id",
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:                 "name-2",
			DataDriverInstanceID: "di-id-1",
			Parameters: map[string]interface{}{
				"key1": 2,
			},
			ServiceDomainBinding: model.ServiceDomainBinding{
				ServiceDomainIDs: []string{"edge-1", "edge-2"},
			},
		},
	}
	dynamicConfigStrings := []string{
		`{"id":"config-1","version":1,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","excludeServiceDomainIds":["no-edge-1"],"serviceDomainSelectors":[{"id":"cat-id-1","value":"v1"}],"name":"name-1","description":"desc-1","dataDriverInstanceID":"di-id-1","parameters":{"key1":1}}`,
		`{"id":"config-2","version":1,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","serviceDomainIds":["edge-1","edge-2"],"name":"name-2","dataDriverInstanceID":"di-id-1","parameters":{"key1":2}}`,
	}

	for i, dataDriver := range dynamicConfigs {
		data, err := json.Marshal(dataDriver)
		require.NoError(t, err, "failed to marshal dynamicConfig")
		require.Equal(t, dynamicConfigStrings[i], string(data), "dataDriverConfig json string mismatch")

		m := map[string]interface{}{}
		err = json.Unmarshal(data, &m)
		require.NoError(t, err, "failed to unmarshal dynamicConfig to map")
	}
}

// TestDatadriverConfigValidate will test ValidateDataDriverConfig
func TestDatadriverConfigValidate(t *testing.T) {
	now := timeNow(t)
	schema1 := []byte(`{
		"type": "object",
		"properties": {
			"field1": {
				"type": "string"
			},
			"field2": {
				"type": "integer"
			}
		}
	}`)
	dd1 := makeDataDriverClass(t, "name", nil, &schema1, nil)
	projectEdges := model.Project{
		Name:             "project-1",
		EdgeSelectorType: model.ProjectEdgeSelectorTypeExplicit,
		EdgeIDs:          []string{"edge-1"},
	}
	projectCategoies := model.Project{
		Name:             "project-2",
		EdgeSelectorType: model.ProjectEdgeSelectorTypeCategory,
		EdgeSelectors: []model.CategoryInfo{{
			ID:    "id-1",
			Value: "val-1",
		}},
	}

	t.Run("Bad", func(t *testing.T) {
		dataDriverConfigs := []model.DataDriverConfig{
			makeDataDriverConfig(t, "name-1", map[string]interface{}{
				"field1": 1,
			}),
			makeDataDriverConfig(t, "name-2", map[string]interface{}{
				"field2": "str",
			}),
			makeDataDriverConfig(t, "name-3", map[string]interface{}{
				"field3": "str",
			}),
			{
				BaseModel: model.BaseModel{
					ID:        "name-4",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                 "name-4",
				DataDriverInstanceID: "",
				Parameters: map[string]interface{}{
					"key1": 2,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edge-1", "edge-2"},
				},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-5",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name: "name-5",
				Parameters: map[string]interface{}{
					"key1": 2,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edge-1", "edge-2"},
				},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-6",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceID: "di-id-1",
				Name:                 "name-6",
				Description:          "desc-1",
				Parameters: map[string]interface{}{
					"key1": 1,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edge-1", "edge-2"},
				},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-7",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceID: "di-id-1",
				Name:                 "name-7",
				Description:          "desc-1",
				Parameters:           map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainSelectors: []model.CategoryInfo{{ID: "id-1", Value: "value-1"}},
				},
			},
			makeDataDriverConfig(t, "", map[string]interface{}{}),
			makeDataDriverConfig(t, " ", map[string]interface{}{}),
		}
		for _, cfg := range dataDriverConfigs {
			t.Run(cfg.Name, func(t *testing.T) {
				err := model.ValidateDataDriverConfig(&cfg, &dd1.ConfigParameterSchema, &projectEdges)
				require.Error(t, err)
			})
		}
	})

	t.Run("Empty/non-empty schema", func(t *testing.T) {
		config := model.DataDriverConfig{
			BaseModel: model.BaseModel{
				ID:       "name-3",
				TenantID: "tenant-id",
			},
			DataDriverInstanceID: "di-id-1",
			Name:                 "name-1",
			Description:          "desc-1",
			Parameters:           nil,
			ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
		}

		err := model.ValidateDataDriverConfig(&config, &dd1.ConfigParameterSchema, &projectEdges)
		require.Error(t, err, "error: config absent, schema - present")

		err = model.ValidateDataDriverConfig(&config, nil, &projectEdges)
		require.NoError(t, err, "no error: both absent")

		config.Parameters = map[string]interface{}{"a": "b"}
		err = model.ValidateDataDriverConfig(&config, nil, &projectEdges)
		require.Error(t, err, "error: config present, schema absent")
	})

	t.Run("Good with edge selector", func(t *testing.T) {
		dataDriverInstances := []model.DataDriverConfig{
			makeDataDriverConfig(t, "name-1", map[string]interface{}{
				"field1": "1",
			}),
			makeDataDriverConfig(t, "name-2", map[string]interface{}{
				"field2": 1,
			}),
			{
				BaseModel: model.BaseModel{
					ID:        "name-3",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceID: "di-id-1",
				Name:                 "name-1",
				Description:          "desc-1",
				Parameters:           map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
			},
		}

		for _, cfg := range dataDriverInstances {
			t.Run(cfg.Name, func(t *testing.T) {
				err := model.ValidateDataDriverConfig(&cfg, &dd1.ConfigParameterSchema, &projectEdges)
				require.NoError(t, err)
			})
		}
	})

	t.Run("Good with category selector", func(t *testing.T) {
		dataDriverConfig := model.DataDriverConfig{
			BaseModel: model.BaseModel{
				ID:        "name-3",
				TenantID:  "tenant-id",
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			DataDriverInstanceID: "di-id-1",
			Name:                 "name-1",
			Description:          "desc-1",
			Parameters:           map[string]interface{}{"field1": "1"},
			ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainSelectors: []model.CategoryInfo{{ID: "c1", Value: "v1"}}},
		}
		err := model.ValidateDataDriverConfig(&dataDriverConfig, &dd1.ConfigParameterSchema, &projectCategoies)
		require.NoError(t, err)

		err = model.ValidateDataDriverConfig(&dataDriverConfig, &dd1.ConfigParameterSchema, &projectEdges)
		require.Error(t, err)
	})
}

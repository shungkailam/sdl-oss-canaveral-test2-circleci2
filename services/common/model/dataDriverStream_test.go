package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func makeDataDriverStream(t *testing.T, name string, params map[string]interface{}) model.DataDriverStream {
	now := timeNow(t)
	return model.DataDriverStream{
		BaseModel: model.BaseModel{
			ID:        "di-id-1",
			TenantID:  "tenant-id",
			Version:   5,
			CreatedAt: now,
			UpdatedAt: now,
		},
		ServiceDomainBinding: model.ServiceDomainBinding{
			ServiceDomainIDs: []string{"edge-1"},
		},
		Name:                 name,
		Description:          "desc-1",
		DataDriverInstanceID: "di-id-1",
		Direction:            model.DataDriverStreamSource,
		Stream:               params,
		Labels: []model.CategoryInfo{
			{
				ID:    "c-id-1",
				Value: "c1-v1",
			},
			{
				ID:    "c-id-2",
				Value: "c2-v1",
			},
		},
	}
}

// TestDatadriverInstance will test DataDriverConfig struct
func TestDatadriverInstanceStream(t *testing.T) {
	now := timeNow(t)
	c1 := model.CategoryInfo{
		ID:    "cat-id-1",
		Value: "v1",
	}
	dynamicConfigs := []model.DataDriverStream{
		{
			BaseModel: model.BaseModel{
				ID:        "config-1",
				TenantID:  "tenant-id",
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ServiceDomainBinding: model.ServiceDomainBinding{
				ServiceDomainSelectors:  []model.CategoryInfo{c1},
				ExcludeServiceDomainIDs: []string{"no-edge-1"},
			},
			Name:                 "name-1",
			Description:          "desc-1",
			DataDriverInstanceID: "di-id-1",
			Direction:            model.DataDriverStreamSource,
			Stream: map[string]interface{}{
				"key1": 1,
			},
			Labels: []model.CategoryInfo{
				{
					ID:    "c1",
					Value: "v1",
				},
				{
					ID:    "c2",
					Value: "v2",
				},
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
			Stream: map[string]interface{}{
				"key1": 2,
			},
			ServiceDomainBinding: model.ServiceDomainBinding{
				ServiceDomainIDs: []string{"edge-1", "edge-2"},
			},
			Labels: []model.CategoryInfo{
				{
					ID:    "c1",
					Value: "v1",
				},
			},
		},
	}
	streamStrings := []string{
		`{"id":"config-1","version":1,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","excludeServiceDomainIds":["no-edge-1"],"serviceDomainSelectors":[{"id":"cat-id-1","value":"v1"}],"name":"name-1","description":"desc-1","dataDriverInstanceID":"di-id-1","direction":"SOURCE","stream":{"key1":1},"labels":[{"id":"c1","value":"v1"},{"id":"c2","value":"v2"}]}`,
		`{"id":"config-2","version":1,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","serviceDomainIds":["edge-1","edge-2"],"name":"name-2","dataDriverInstanceID":"di-id-1","stream":{"key1":2},"labels":[{"id":"c1","value":"v1"}]}`,
	}

	for i, dataDriver := range dynamicConfigs {
		data, err := json.Marshal(dataDriver)
		require.NoError(t, err, "failed to marshal dynamicConfig")
		require.Equal(t, streamStrings[i], string(data), "dataDriverConfig json string mismatch")

		m := map[string]interface{}{}
		err = json.Unmarshal(data, &m)
		require.NoError(t, err, "failed to unmarshal dynamicConfig to map")
	}
}

// TestDatadriverConfigValidate will test ValidateDataDriverConfig
func TestDatadriverStreamValidate(t *testing.T) {
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
	dd1 := makeDataDriverClass(t, "name", nil, nil, &schema1)
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
		dataDriverInstances := []model.DataDriverStream{
			makeDataDriverStream(t, "name-1", map[string]interface{}{
				"field1": 1, // field1 is string
			}),
			makeDataDriverStream(t, "name-2", map[string]interface{}{
				"field2": "str", // field2 is int
			}),
			makeDataDriverStream(t, "name-3", map[string]interface{}{
				"field3": "str", // field3 is not configured
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
				Direction:            model.DataDriverStreamSource,
				DataDriverInstanceID: "", // Empty data driver instance ID
				Stream: map[string]interface{}{
					"key1": 2,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edge-1", "edge-2"},
				},
				Labels: []model.CategoryInfo{{ID: "c1", Value: "v1"}},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-5",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:      "name-5",
				Direction: model.DataDriverStreamSource,
				// No data driver instance ID
				Stream: map[string]interface{}{
					"key1": 2,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edge-1", "edge-2"},
				},
				Labels: []model.CategoryInfo{{ID: "c1", Value: "v1"}},
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
				Direction:            model.DataDriverStreamSource,
				Stream: map[string]interface{}{
					"key1": 1,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{}, // Empty binding
				Labels:               []model.CategoryInfo{{ID: "c1", Value: "v1"}},
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
				// No direction
				Stream:               map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
				Labels:               []model.CategoryInfo{{ID: "c1", Value: "v1"}},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-8",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceID: "di-id-1",
				Name:                 "name-8",
				Description:          "desc-1",
				Direction:            model.DataDriverStreamSource,
				Stream:               map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
				Labels:               []model.CategoryInfo{}, // empty labels
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-9",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                 "name-9",
				Direction:            model.DataDriverStreamSource,
				DataDriverInstanceID: "ddid-1",
				Stream: map[string]interface{}{
					"key1": 2,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edge-10"}, // service domain is out of project
				},
				Labels: []model.CategoryInfo{{ID: "c1", Value: "v1"}},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-10",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                 "name-10",
				Direction:            model.DataDriverStreamSource,
				DataDriverInstanceID: "ddid-1",
				Stream: map[string]interface{}{
					"key1": 2,
				},
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainSelectors: []model.CategoryInfo{{ID: "c1", Value: "v1"}}, // category selector for edge-based project
				},
				Labels: []model.CategoryInfo{{ID: "c1", Value: "v1"}},
			},
			{
				BaseModel: model.BaseModel{
					ID:        "name-11",
					TenantID:  "tenant-id",
					Version:   1,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceID: "di-id-1",
				Name:                 "name-11",
				Description:          "desc-1",
				Direction:            model.DataDriverStreamSink,
				Stream:               map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
				Labels:               []model.CategoryInfo{{ID: "c1", Value: "v1"}},
			},
			makeDataDriverStream(t, "", map[string]interface{}{}),
			makeDataDriverStream(t, " ", map[string]interface{}{}),
		}

		for _, cfg := range dataDriverInstances {
			t.Run(cfg.Name, func(t *testing.T) {
				err := model.ValidateDataDriverStream(&cfg, &dd1.StreamParameterSchema, &projectEdges)
				require.Error(t, err)
			})
		}
	})

	t.Run("Empty/non-empty schema", func(t *testing.T) {
		stream := model.DataDriverStream{
			BaseModel: model.BaseModel{
				ID:       "name-3",
				TenantID: "tenant-id",
			},
			DataDriverInstanceID: "di-id-1",
			Name:                 "name-1",
			Description:          "desc-1",
			Direction:            model.DataDriverStreamSource,
			Stream:               nil,
			ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
			Labels:               []model.CategoryInfo{{ID: "c1", Value: "v1"}},
		}

		err := model.ValidateDataDriverStream(&stream, &dd1.StreamParameterSchema, &projectEdges)
		require.Error(t, err, "error: stream absent, schema - present")

		err = model.ValidateDataDriverStream(&stream, nil, &projectEdges)
		require.NoError(t, err, "no error: both absent")

		stream.Stream = map[string]interface{}{"a": "b"}
		err = model.ValidateDataDriverStream(&stream, nil, &projectEdges)
		require.Error(t, err, "error: stream present, schema absent")
	})

	t.Run("Good with edge selector", func(t *testing.T) {
		dataDriverStreams := []model.DataDriverStream{
			makeDataDriverStream(t, "name-1", map[string]interface{}{
				"field1": "1",
			}),
			makeDataDriverStream(t, "name-2", map[string]interface{}{
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
				Direction:            model.DataDriverStreamSource,
				Stream:               map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
				Labels:               []model.CategoryInfo{{ID: "c1", Value: "v1"}},
			},
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
				Direction:            model.DataDriverStreamSink,
				Stream:               map[string]interface{}{"field1": "1"},
				ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainIDs: []string{"edge-1"}},
			},
		}

		for _, cfg := range dataDriverStreams {
			t.Run(cfg.Name, func(t *testing.T) {
				err := model.ValidateDataDriverStream(&cfg, &dd1.StreamParameterSchema, &projectEdges)
				require.NoError(t, err)
			})
		}
	})

	t.Run("Good with category selector", func(t *testing.T) {
		dataDriverStream := model.DataDriverStream{
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
			Direction:            model.DataDriverStreamSource,
			Stream:               map[string]interface{}{"field1": "1"},
			ServiceDomainBinding: model.ServiceDomainBinding{ServiceDomainSelectors: []model.CategoryInfo{{ID: "c1", Value: "v1"}}},
			Labels:               []model.CategoryInfo{{ID: "c1", Value: "v1"}},
		}
		err := model.ValidateDataDriverStream(&dataDriverStream, &dd1.StreamParameterSchema, &projectCategoies)
		require.NoError(t, err)

		err = model.ValidateDataDriverStream(&dataDriverStream, &dd1.StreamParameterSchema, &projectEdges)
		require.Error(t, err)
	})
}

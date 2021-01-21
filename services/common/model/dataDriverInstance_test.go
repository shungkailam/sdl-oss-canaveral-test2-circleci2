package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func makeDataDriverInstance(t *testing.T, name string, params map[string]interface{}) model.DataDriverInstance {
	now := timeNow(t)
	return model.DataDriverInstance{
		BaseModel: model.BaseModel{
			ID:        "di-id-1",
			TenantID:  "tenant-id",
			Version:   5,
			CreatedAt: now,
			UpdatedAt: now,
		},
		DataDriverInstanceCore: model.DataDriverInstanceCore{
			Name:              name,
			Description:       "desc-1",
			DataDriverClassID: "dd-id-1",
			ProjectID:         "project-123",
			StaticParameters:  params,
		},
	}
}

// TestDatadriverInstance will test DataDriverInstance struct
func TestDatadriverInstance(t *testing.T) {
	now := timeNow(t)
	schama := []byte(`{
		"type": "object",
		"properties": {
			"key1": {
				"type": "string"
			},
			"key2": {
				"type": "integer"
			}
		}
	}`)
	dd := makeDataDriverClass(t, "name", &schama, nil, nil)
	dataDriverInstances := []model.DataDriverInstance{
		makeDataDriverInstance(t, "name-1", map[string]interface{}{
			"key1": "value1",
		}),
		{
			BaseModel: model.BaseModel{
				ID:        "di-id-2",
				TenantID:  "tenant-id",
				Version:   1,
				CreatedAt: now,
				UpdatedAt: now,
			},
			DataDriverInstanceCore: model.DataDriverInstanceCore{
				Name:              "name-2",
				DataDriverClassID: "dd-id-2",
				ProjectID:         "project-123",
				StaticParameters: map[string]interface{}{
					"key2": 2,
				},
			},
		},
	}
	dataDriverInstanceStrings := []string{
		`{"id":"di-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"name-1","description":"desc-1","dataDriverClassID":"dd-id-1","projectId":"project-123","staticParameters":{"key1":"value1"}}`,
		`{"id":"di-id-2","version":1,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"name-2","dataDriverClassID":"dd-id-2","projectId":"project-123","staticParameters":{"key2":2}}`,
	}

	for i, dataDriverInstance := range dataDriverInstances {
		dataDriverInstanceData, err := json.Marshal(dataDriverInstance)
		require.NoError(t, err, "failed to marshal dataDriverInstance: %d", i)

		if dataDriverInstanceStrings[i] != string(dataDriverInstanceData) {
			t.Fatalf("dataDriverInstance json string mismatch: %s", string(dataDriverInstanceData))
		}
		m := map[string]interface{}{}
		err = json.Unmarshal(dataDriverInstanceData, &m)
		require.NoError(t, err, "failed to unmarshal dataDriverInstance to map: %d", i)

		err = model.ValidateDataDriverInstance(&dataDriverInstance, dd.StaticParameterSchema)
		require.NoError(t, err, "failed to validate dataDriverInstance: %d", i)
	}
}

// TestDatadriverInstanceValidate will test ValidateDataDriverInstance
func TestDatadriverInstanceValidate(t *testing.T) {
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
	dd1 := makeDataDriverClass(t, "name", &schema1, nil, nil)
	dd2 := makeDataDriverClass(t, "name", nil, nil, nil)

	t.Run("Non empty schema", func(t *testing.T) {
		dataDriverInstances := []model.DataDriverInstance{
			makeDataDriverInstance(t, "name-1", map[string]interface{}{
				"field1": 1,
			}),
			makeDataDriverInstance(t, "name-2", map[string]interface{}{
				"field2": "str",
			}),
			makeDataDriverInstance(t, "name-3", map[string]interface{}{
				"field3": "str",
			}),
			makeDataDriverInstance(t, "name-4", nil),
		}

		for _, instance := range dataDriverInstances {
			err := model.ValidateDataDriverInstance(&instance, dd1.StaticParameterSchema)
			require.Error(t, err, "Should be error")
		}
	})

	t.Run("Empty schema", func(t *testing.T) {
		dataDriverInstances := []model.DataDriverInstance{
			makeDataDriverInstance(t, "name-1", map[string]interface{}{
				"field1": 1,
			}),
			makeDataDriverInstance(t, "name-2", map[string]interface{}{
				"field2": "str",
			}),
			makeDataDriverInstance(t, "name-3", map[string]interface{}{
				"field3": "str",
			}),
		}

		for _, instance := range dataDriverInstances {
			err := model.ValidateDataDriverInstance(&instance, dd2.StaticParameterSchema)
			require.Error(t, err, "Should be error")
		}
	})

	t.Run("Empty schema and empty values", func(t *testing.T) {
		instance := makeDataDriverInstance(t, "name-1", nil)
		err := model.ValidateDataDriverInstance(&instance, dd2.StaticParameterSchema)
		require.NoError(t, err)
	})

	t.Run("Empty names", func(t *testing.T) {
		instance := makeDataDriverInstance(t, "", nil)
		err := model.ValidateDataDriverInstance(&instance, dd2.StaticParameterSchema)
		require.Error(t, err)

		instance = makeDataDriverInstance(t, " ", nil)
		err = model.ValidateDataDriverInstance(&instance, dd2.StaticParameterSchema)
		require.Error(t, err)
	})
}

package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

func parseDataDriverSchema(dynamic *[]byte) (model.DataDriverParametersSchema, error) {
	d := model.DataDriverParametersSchema{}
	if dynamic == nil {
		return d, nil
	}
	err := base.ConvertFromJSON(*dynamic, &d)
	return d, err
}

func makeDataDriverClass(t *testing.T, name string, static *[]byte, dynamic *[]byte, stream *[]byte) model.DataDriverClass {
	s, err := parseDataDriverSchema(static)
	require.NoError(t, err)

	d, err := parseDataDriverSchema(dynamic)
	require.NoError(t, err)

	st, err := parseDataDriverSchema(stream)
	require.NoError(t, err)

	now := timeNow(t)

	return model.DataDriverClass{
		BaseModel: model.BaseModel{
			ID:        "dd-id-1",
			TenantID:  "tenant-id",
			Version:   5,
			CreatedAt: now,
			UpdatedAt: now,
		},
		DataDriverClassCore: model.DataDriverClassCore{
			Name:                  name,
			Description:           "dd-desc-1",
			DataDriverVersion:     "1.0.2",
			MinSvcDomainVersion:   "1.0.0",
			Type:                  "SOURCE",
			YamlData:              "YAML_DATA",
			StaticParameterSchema: s,
			ConfigParameterSchema: d,
			StreamParameterSchema: st,
		},
	}
}

// TestDatadriverClass will test DataDriverClass struct
func TestDatadriverClass(t *testing.T) {

	schema1 := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "object",
		"properties": {
			"logRetentionBytes": {
				"type": "string",
				"default": "1000000"
			},
			"kafkaVolumeSize": {
				"type": "string",
				"default": "1000000"
			},
			"profile": {
				"description": "Mutually exclusive kafka profiles",
				"type": "string",
				"enum": [
					"Durability",
					"Throughput",
					"Availability"
				],
				"default": "Availability"
			}
		}
	}`)

	schema2 := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "object",
		"properties": {
			"logRetentionBytes": {
				"type": "string",
				"default": "1000000"
			}
		}
	}`)

	schema3 := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "object",
		"properties": {
			"access-level": {
				"type": "string",
				"enum": [
					"ReadOnly",
					"Write"
				]
			}
		},
		"required": ["access-level"]
	}`)

	schema4 := []byte(`
	{
		"title": "test-1",
		"type": "object",
		"properties": {
		}
	}`)

	dataDrivers := []model.DataDriverClass{
		makeDataDriverClass(t, "name-1", &schema1, &schema2, &schema3),
		makeDataDriverClass(t, "name-2", &schema3, &schema1, nil),
		makeDataDriverClass(t, "name-3", &schema4, nil, nil),
		makeDataDriverClass(t, "name-4", nil, nil, nil),
	}

	dataDriverStrings := []string{
		`{"id":"dd-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"name-1","description":"dd-desc-1","driverVersion":"1.0.2","minSvcDomainVersion":"1.0.0","type":"SOURCE","yamlData":"YAML_DATA","staticParameterSchema":{"properties":{"kafkaVolumeSize":{"default":"1000000","type":"string"},"logRetentionBytes":{"default":"1000000","type":"string"},"profile":{"default":"Availability","description":"Mutually exclusive kafka profiles","enum":["Durability","Throughput","Availability"],"type":"string"}},"title":"KafkaCfgOptions","type":"object"},"configParameterSchema":{"properties":{"logRetentionBytes":{"default":"1000000","type":"string"}},"title":"KafkaCfgOptions","type":"object"},"streamParameterSchema":{"properties":{"access-level":{"enum":["ReadOnly","Write"],"type":"string"}},"required":["access-level"],"title":"KafkaCfgOptions","type":"object"}}`,
		`{"id":"dd-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"name-2","description":"dd-desc-1","driverVersion":"1.0.2","minSvcDomainVersion":"1.0.0","type":"SOURCE","yamlData":"YAML_DATA","staticParameterSchema":{"properties":{"access-level":{"enum":["ReadOnly","Write"],"type":"string"}},"required":["access-level"],"title":"KafkaCfgOptions","type":"object"},"configParameterSchema":{"properties":{"kafkaVolumeSize":{"default":"1000000","type":"string"},"logRetentionBytes":{"default":"1000000","type":"string"},"profile":{"default":"Availability","description":"Mutually exclusive kafka profiles","enum":["Durability","Throughput","Availability"],"type":"string"}},"title":"KafkaCfgOptions","type":"object"}}`,
		`{"id":"dd-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"name-3","description":"dd-desc-1","driverVersion":"1.0.2","minSvcDomainVersion":"1.0.0","type":"SOURCE","yamlData":"YAML_DATA","staticParameterSchema":{"properties":{},"title":"test-1","type":"object"}}`,
		`{"id":"dd-id-1","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"name-4","description":"dd-desc-1","driverVersion":"1.0.2","minSvcDomainVersion":"1.0.0","type":"SOURCE","yamlData":"YAML_DATA"}`,
	}

	for i, dataDriver := range dataDrivers {
		dataDriverData, err := json.Marshal(dataDriver)
		require.NoError(t, err, "failed to marshal dataDriverClass")

		if dataDriverStrings[i] != string(dataDriverData) {
			t.Fatalf("dataDriverClass json string mismatch: %s", string(dataDriverData))
		}
		m := model.DataDriverClass{}
		err = json.Unmarshal(dataDriverData, &m)
		require.NoError(t, err, "failed to unmarshal dataDriverClass to map")

		err = model.ValidateDataDriverClass(&m)
		require.NoError(t, err, "failed to validate dataDriverClass")
	}
}

// TestDatadriverClassValidate will test DataDriverClass struct validation
func TestDatadriverClassValidate(t *testing.T) {
	valid := []byte(`
	{
		"type": "object",
		"properties": {
			"logRetentionBytes": {
				"type": "string",
				"default": "1000000"
			}
		}
	}`)

	schema1 := []byte(`
	{
		"properties": {
		}
	}`)
	schema2 := []byte(`
	{
		"properties": []
	}`)
	schema3 := []byte(`
	{
		"type": "test"
	}`)
	schema4 := []byte(`
	{
		"a": "b"
	}`)
	schema5 := []byte(`
	{
		"type": "object",
		"properties": {
			"logRetentionBytes": {
				"type": "test",
				"default": "1000000"
			}
		}
	}`)

	for _, s := range [][]byte{schema1, schema2, schema3, schema4, schema5} {
		m1 := makeDataDriverClass(t, "name-1", &s, nil, nil)
		err := model.ValidateDataDriverClass(&m1)
		require.Error(t, err, "Should be an error on %s", string(s))

		m2 := makeDataDriverClass(t, "name-2", &valid, &s, nil)
		err = model.ValidateDataDriverClass(&m2)
		require.Error(t, err, "Should be an error on %s", string(s))

		m3 := makeDataDriverClass(t, "name-3", &s, &valid, nil)
		err = model.ValidateDataDriverClass(&m3)
		require.Error(t, err, "Should be an error on %s", string(s))

		m4 := makeDataDriverClass(t, "name-4", &s, nil, &valid)
		err = model.ValidateDataDriverClass(&m4)
		require.Error(t, err, "Should be an error on %s", string(s))

		m5 := makeDataDriverClass(t, "name-5", nil, &s, nil)
		err = model.ValidateDataDriverClass(&m5)
		require.Error(t, err, "Should be an error on %s", string(s))

		m6 := makeDataDriverClass(t, "name-6", nil, nil, &s)
		err = model.ValidateDataDriverClass(&m6)
		require.Error(t, err, "Should be an error on %s", string(s))
	}

	m7 := makeDataDriverClass(t, "name-7", nil, nil, nil)
	err := model.ValidateDataDriverClass(&m7)
	require.NoError(t, err, "Should not be an error", err)

	m7.Type = "FAIL"
	err = model.ValidateDataDriverClass(&m7)
	require.Error(t, err, "Should be an error")

	m8 := makeDataDriverClass(t, "", nil, nil, nil)
	err = model.ValidateDataDriverClass(&m8)
	require.Error(t, err, "Should be an error")

	m9 := makeDataDriverClass(t, " ", nil, nil, nil)
	err = model.ValidateDataDriverClass(&m9)
	require.Error(t, err, "Should be an error")
}

package schema_test

import (
	"cloudservices/common/base"
	"cloudservices/common/schema"
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

var invalidRootTypeSchemaSpec = []byte(`{
    "title": "Person",
	"type": "string"
 }`)

var invalidCustomRootFieldsSchemaSpec = []byte(`{
    "title": "Person",
	"type": "object",
	"field1": {
		"type": "string"
	},
	"field2" : {
		"type" : "string",
		"default": "value2"
	},
    "properties": {
        "credentials": {
            "type": "object",
            "additionalProperties": {}
        },
        "endpoint": {
            "type": "string",
            "default": "localhost"
        }
    }
 }`)

var validEmptySchema = []byte(`{
 }`)

var schemaSpec = []byte(`{
	"title": "Person",
	"type": "object",
	"properties": {
		"properties": {
			"type": "object",
			"properties": {
				"credentials" : {
					"type": "object",
					"additionalProperties": {}
				},
				"endpoint": {
					"type": "string",
					"default": "localhost"
				}
			}
		},
		"firstName": {
			"title": "secret",
			"type": "string"
		},
		"middleName": {
			"type": "string",
			"default": "NA"
		},
		"lastName": {
			"type": "string",
			"default": "Unknown"
		},
		"age": {
			"description": "Age in years",
			"type": "integer",
			"minimum": 0
		},
		"friends": {
		  "type" : "array",
		  "items" : { "title" : "REFERENCE", "$ref" : "#" }
		},
		"test": {
			"type": "object",
			"properties": {
				"t1" : {
					"type": "string",
					"default": "testing"
				},
				"t2": {
					"type": "integer",
					"default": 5
				}
			}
		},
		"element": {
            "type": "object",
            "properties": {
                "number" : {"type": "integer", "default": 15}
            },
            "default": {"element": {}}
		},
		"test2": {
			"type": "object",
  			"propertyNames": { "maxLength": 3, "minLength": 3 },
  			"patternProperties": {
				"": { "type": "number" }
			}
		}
	},
	"required": ["firstName", "lastName"]
  }`)

func TestJSONSchema(t *testing.T) {
	ctx := context.TODO()
	invalidSchemaJSON := []byte(`{
		"firstName" : "George",
		"lastName1" : "Michael"
		}`)
	validSchemaJSON := []byte(`{
		"firstName" : "George",
		"lastName" : "Michael"
		}`)

	expectedJsonNotRedacted := []byte(`{
		"element": {
		 "element": {}
		},
		"firstName": "George",
		"middleName": "NA",
		"lastName": "Michael",
		"properties": {
		 "endpoint": "localhost"
		},
		"test": {
		 "t1": "testing",
		 "t2": 5
		}
	   }
	`)
	expectedJsonRedacted := []byte(`{
		"element": {
		 "element": {}
		},
		"firstName": "REDACTED",
		"lastName": "Michael",
		"middleName": "NA",
		"properties": {
		 "endpoint": "localhost"
		},
		"test": {
		 "t1": "testing",
		 "t2": 5
		}
	   }
	`)
	schemaSpecMap := map[string]interface{}{}
	err := base.ConvertFromJSON(schemaSpec, &schemaSpecMap)
	require.NoError(t, err)

	// Validate the spec
	err = schema.ValidateSpecMap(ctx, schemaSpecMap)
	require.NoError(t, err)
	schemaMap := map[string]interface{}{}
	err = base.ConvertFromJSON(invalidSchemaJSON, &schemaMap)
	require.NoError(t, err)

	// Invalid schema
	err = schema.ValidateSchemaMap(ctx, schemaSpecMap, schemaMap)
	require.Error(t, err)
	schemaMap = map[string]interface{}{}
	err = base.ConvertFromJSON(validSchemaJSON, &schemaMap)
	require.NoError(t, err)

	// Validate the schema againt the spec
	err = schema.ValidateSchemaMap(ctx, schemaSpecMap, schemaMap)
	require.NoError(t, err)
	b, _ := json.MarshalIndent(schemaMap, " ", " ")
	t.Logf("schema map: %+v", string(b))
	expectedSchemaMap := map[string]interface{}{}
	err = base.ConvertFromJSON(expectedJsonNotRedacted, &expectedSchemaMap)
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedSchemaMap, schemaMap) {
		t.Fatalf("mistmatched values")
	}
	// Test for redaction
	schemaMap = map[string]interface{}{}
	err = base.ConvertFromJSON(validSchemaJSON, &schemaMap)

	// Redact the secret fields with the metadata built from spec
	err = schema.RedactProperties(ctx, schemaSpecMap, schemaMap)
	require.NoError(t, err)
	b, _ = json.MarshalIndent(schemaMap, " ", " ")
	t.Logf("redacted schema map: %+v", string(b))
	expectedSchemaMap = map[string]interface{}{}
	err = base.ConvertFromJSON(expectedJsonRedacted, &expectedSchemaMap)
	require.NoError(t, err)
	if !reflect.DeepEqual(expectedSchemaMap, schemaMap) {
		t.Fatalf("mistmatched values")
	}

	// Invalid root type
	schemaSpecMap = map[string]interface{}{}
	err = base.ConvertFromJSON(invalidRootTypeSchemaSpec, &schemaSpecMap)
	require.NoError(t, err)

	// Validate the spec
	err = schema.ValidateSpecMap(ctx, schemaSpecMap)
	require.Error(t, err)

	// Invalid unknown fields in the root
	schemaSpecMap = map[string]interface{}{}
	err = base.ConvertFromJSON(invalidCustomRootFieldsSchemaSpec, &schemaSpecMap)
	require.NoError(t, err)

	// Validate the spec
	err = schema.ValidateSpecMap(ctx, schemaSpecMap)
	require.Error(t, err)

	// Valid empty spec
	schemaSpecMap = map[string]interface{}{}
	err = base.ConvertFromJSON(validEmptySchema, &schemaSpecMap)
	require.NoError(t, err)

	// Validate the spec
	err = schema.ValidateSpecMap(ctx, schemaSpecMap)
	require.NoError(t, err)

	// Set a property which does not exist
	inValidSchemaJSON := []byte(`{"test": true}`)
	schemaMap = map[string]interface{}{}
	err = base.ConvertFromJSON(inValidSchemaJSON, &schemaMap)
	require.NoError(t, err)

	// Validate the schema againt the spec
	err = schema.ValidateSchemaMap(ctx, schemaSpecMap, schemaMap)
	require.Error(t, err)

	// Set no property for empty spec
	validSchemaJSON = []byte(`{}`)
	schemaMap = map[string]interface{}{}
	err = base.ConvertFromJSON(validSchemaJSON, &schemaMap)
	require.NoError(t, err)

	// Validate the spec
	err = schema.ValidateSpecMap(ctx, schemaSpecMap)
	require.NoError(t, err)

	// Validate the schema againt the spec
	err = schema.ValidateSchemaMap(ctx, schemaSpecMap, schemaMap)
	require.NoError(t, err)
}

package schema

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/golang/glog"
	"github.com/qri-io/jsonschema"
)

type SecretValue bool

type TerminalValue bool

func loadSchemaSpec(ctx context.Context, schemaSpec []byte) (*jsonschema.RootSchema, error) {
	rs := &jsonschema.RootSchema{}
	if err := json.Unmarshal(schemaSpec, rs); err != nil {
		return nil, errcode.NewBadRequestExError("schemaSpec", err.Error())
	}
	return rs, nil
}

func isTerminal(pather jsonschema.JSONPather) bool {
	anyMoreFields := false
	if jsonContainer, ok := pather.(jsonschema.JSONContainer); ok {
		for _, childPather := range jsonContainer.JSONChildren() {
			// Check for start of a schema
			_, anyMoreFields = childPather.(*jsonschema.Schema)
			if anyMoreFields {
				break
			}
			// Check for object properties
			_, anyMoreFields = childPather.(*jsonschema.Properties)
			if anyMoreFields {
				break
			}
		}
	}
	return !anyMoreFields
}

func extractDefaults(parent string, pather jsonschema.JSONPather) interface{} {
	isProperty := false
	if schema, ok := pather.(*jsonschema.Schema); ok {
		// Sub schema root
		defaultValue := schema.JSONProp("default")
		if defaultValue != nil {
			return defaultValue
		}
		// Check for password and other keywords
		title := schema.Title
		if strings.ToLower(title) == "secret" {
			return SecretValue(true)
		}

		if isTerminal(pather) {
			return TerminalValue(true)
		}
		// Otherwise, recursively analyze
	} else {
		// The key is property
		_, isProperty = pather.(*jsonschema.Properties)
	}

	if jsonContainer, ok := pather.(jsonschema.JSONContainer); ok {
		m := map[string]interface{}{}
		for name, pather := range jsonContainer.JSONChildren() {
			val := extractDefaults(name, pather)
			if val == nil {
				continue
			}
			// Val is not nil only if default is found in the children
			// Property also is for an object i.e a map for a key
			if isProperty {
				m[name] = val
				continue
			}
			return val
		}
		if len(m) == 0 {
			return nil
		}
		return m
	}
	return nil
}

// ExtractDefaults extracts the defauls and builds the directive metadata
func ExtractDefaults(pather jsonschema.JSONPather) map[string]interface{} {
	i := extractDefaults("", pather)
	m, ok := i.(map[string]interface{})
	if !ok {
		m = map[string]interface{}{}
	}
	return m
}

// validateSpec validates a JSON byte array spec returning the root schema and error if any
func validateSpec(ctx context.Context, schemaSpec []byte) (*jsonschema.RootSchema, error) {
	rs, err := loadSchemaSpec(ctx, schemaSpec)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in loading schema spec. Error: %s"), err.Error())
		return nil, err
	}
	err = validateRootSpec(ctx, rs)
	if err != nil {
		return nil, err
	}
	glog.Infof(base.PrefixRequestID(ctx, "Schema '%s' loaded succesfully"), rs.Title)
	return rs, nil
}

// validateSpec validates a JSON map spec returning the root schema and error if any
func validateSpecMap(ctx context.Context, schemaSpec map[string]interface{}) (*jsonschema.RootSchema, error) {
	if schemaSpec == nil {
		schemaSpec = map[string]interface{}{}
	}
	jsonData, err := base.ConvertToJSON(schemaSpec)
	if err != nil {
		return nil, err
	}
	return validateSpec(ctx, jsonData)
}

// validateRootSpec validates the root schema
func validateRootSpec(ctx context.Context, rs *jsonschema.RootSchema) error {
	if rs == nil {
		return nil
	}
	// Slice to keep all the user-defined fields at the root schema
	customFields := []string{}
	// Get all the field names
	pathers := rs.Schema.JSONChildren()
	if len(pathers) > 0 && rs.TopLevelType() != "object" {
		// If the schema properties are present and the type is not root
		err := errcode.NewBadRequestExError("type", "Root schema type must be an object")
		glog.Errorf(base.PrefixRequestID(ctx, "Error: %s"), err.Error())
		return err
	}
	// All the identified keywords
	validators := rs.Schema.Validators
	// Fail if the root schema has additional custom fields at the root
	for key := range pathers {
		if _, ok := validators[key]; ok {
			continue
		}
		// Not a keyword
		customFields = append(customFields, key)
	}
	if len(customFields) > 0 {
		err := errcode.NewBadRequestExError("schema", fmt.Sprintf("Root object has unknown fields %+v", customFields))
		glog.Errorf(base.PrefixRequestID(ctx, "Error: %s"), err.Error())
		return err
	}
	return nil
}

// ValidateSpec loads the schema spec for any error
func ValidateSpec(ctx context.Context, schemaSpec []byte) error {
	_, err := validateSpec(ctx, schemaSpec)
	return err
}

// ValidateSpecMap loads the schema spec for any error
func ValidateSpecMap(ctx context.Context, schemaSpec map[string]interface{}) error {
	_, err := validateSpecMap(ctx, schemaSpec)
	return err
}

// ValidateSchema validates a JSON map against a schema spec
func ValidateSchema(ctx context.Context, schemaSpec, schema []byte) error {
	rs, err := validateSpec(ctx, schemaSpec)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in loading schema spec. Error: %s"), err.Error())
		return err
	}
	if errors, _ := rs.ValidateBytes(schema); len(errors) > 0 {
		errMsgs := []string{}
		for _, e := range errors {
			errMsgs = append(errMsgs, e.Message)
		}
		err = errcode.NewBadRequestExError("schema", strings.Join(errMsgs, "\n"))
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validating schema. Error: %s"), err.Error())
		return err
	}
	return nil
}

// ValidateSchemaMap validates a JSON map against a schema spec map and performs in place cleaning of the keys and default substition.
func ValidateSchemaMap(ctx context.Context, schemaSpecMap map[string]interface{}, schema map[string]interface{}) error {
	if schemaSpecMap == nil || schema == nil {
		return errcode.NewBadRequestExError("schema", "SchemaSpec or Schema is nil")
	}
	rs, err := validateSpecMap(ctx, schemaSpecMap)
	if err != nil {
		return err
	}
	defaults := ExtractDefaults(rs)
	err = MergeProperties(defaults, schema, true, false)
	if err != nil {
		return err
	}
	jsonData, err := base.ConvertToJSON(schema)
	if err != nil {
		return err
	}
	if errors, _ := rs.ValidateBytes(jsonData); len(errors) > 0 {
		errMsgs := []string{}
		for _, e := range errors {
			errMsgs = append(errMsgs, fmt.Sprintf("%+v", e))
		}
		err = errcode.NewBadRequestExError("schema", strings.Join(errMsgs, "\n"))
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validating schema. Error: %s"), err.Error())
		return err
	}
	return nil
}

// BuildDefaultsMap builds the map with the defaults from the schema spec
func BuildDefaultsMap(ctx context.Context, schemaSpecMap map[string]interface{}) (map[string]interface{}, error) {
	if schemaSpecMap == nil || len(schemaSpecMap) == 0 {
		return map[string]interface{}{}, nil
	}
	schemaSpec, err := base.ConvertToJSON(schemaSpecMap)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in spec spec conversion. Error: %s"), err.Error())
		return nil, err
	}
	rs, err := loadSchemaSpec(ctx, schemaSpec)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in loading schema spec. Error: %s"), err.Error())
		return nil, err
	}
	defaults := ExtractDefaults(rs)
	return defaults, nil
}

// RedactProperties merges the defaults in the schema spec into the schema and also redacts the secrets.
// The defaults are subsituted into 'to' map and the garbage extra fields are removed silently or
// error reported depending on the failOnAdditionalFields flag. Secret values are redacted if
// redact secret is set
func RedactProperties(ctx context.Context, schemaSpecMap map[string]interface{}, schema map[string]interface{}) error {
	defaultsMap, err := BuildDefaultsMap(ctx, schemaSpecMap)
	// only redaction is desired
	err = MergeProperties(defaultsMap, schema, false, true)
	if err != nil {
		return err
	}
	return nil
}

// MergeProperties merges properties from 'from' to 'to' map.
// If the 'to' map contains directives, the defaults are subsituted into 'to' map and the garbage extra fields are removed silently or
// error reported depending on the failOnAdditionalFields flag. Secret values are redacted if
// redactSecret is set
func MergeProperties(from map[string]interface{}, to map[string]interface{}, failOnAdditionalFields, redactSecret bool) error {
	if to == nil || from == nil {
		return nil
	}
	keys := map[string]bool{}
	for key := range from {
		keys[key] = true
	}
	for key := range to {
		keys[key] = true
	}
	for key := range keys {
		fromVal, ok := from[key]
		if !ok {
			if _, ok = to[key]; ok {
				if failOnAdditionalFields {
					return errcode.NewBadRequestExError(key, fmt.Sprintf("Unknown field '%s'", key))
				}
			} else {
				delete(to, key)
			}
			continue
		}
		_, isTerminal := fromVal.(TerminalValue)
		if isTerminal {
			continue
		}
		toVal, isKeyFound := to[key]
		_, isSecret := fromVal.(SecretValue)
		if isKeyFound {
			if isSecret {
				if redactSecret {
					to[key] = "REDACTED"
				}
				continue
			}
		} else {
			// Remove the custom markers
			removeMarkers(fromVal)
			to[key] = fromVal
			continue
		}
		// val may not be a map
		inFrom, fromOk := fromVal.(map[string]interface{})
		inTo, toOk := toVal.(map[string]interface{})
		if fromOk && toOk {
			return MergeProperties(inFrom, inTo, failOnAdditionalFields, redactSecret)
		}
		if fromOk || toOk {
			return errcode.NewBadRequestExError(key, fmt.Sprintf("Incompatible types for '%s'", key))
		}
	}
	return nil
}

func removeMarkers(i interface{}) {
	if m, ok := i.(map[string]interface{}); ok {
		for key, val := range m {
			if _, isTerminal := val.(TerminalValue); isTerminal {
				delete(m, key)
				continue
			}
			if _, isSecret := val.(SecretValue); isSecret {
				delete(m, key)
				continue
			}
			removeMarkers(val)
		}
	}
}

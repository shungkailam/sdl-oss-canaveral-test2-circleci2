package model

import (
	"cloudservices/common/errcode"
	"cloudservices/common/schema"
	"context"
	"strings"
)

const (
	// Types of the data driver class
	DataDriverSource DataDriverClassType = "SOURCE"
	DataDriverSink   DataDriverClassType = "SINK"
	DataDriverBoth   DataDriverClassType = "BOTH"
)

// DataDriverClassType is the type of the data driver class
// swagger:model DataDriverClassType
// enum: SOURCE,SINK,BOTH
type DataDriverClassType string

// DataDriverParametersSchema is the type of the data driver class parameters schema (JSON schema object)
// swagger:model DataDriverParametersSchema
type DataDriverParametersSchema map[string]interface{}

// DataDriverParametersValues is the type for the data driver class parameter values
// swagger:model DataDriverParametersValues
type DataDriverParametersValues map[string]interface{}

// DataDriverClass is object model for data driver class
//
// A data driver class represents a logical IoT Data Source/Sink integration.
// swagger:model DataDriverClass
type DataDriverClass struct {
	// required: true
	BaseModel
	// required: true
	DataDriverClassCore
}

type DataDriverClassCore struct {
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`

	// required: false
	Description string `json:"description,omitempty" validate:"range=1:200"`

	//
	// Externa lversion of a data driver.
	// It is possible to have multiple data drivers with the same name, but different versions.
	//
	// required: true
	DataDriverVersion string `json:"driverVersion" validate:"range=1:20"`

	// required: false
	MinSvcDomainVersion string `json:"minSvcDomainVersion,omitempty" validate:"range=0:20"`

	// required: true
	Type DataDriverClassType `json:"type" validate:"range=1:100"`

	//
	// The YAML content for the application.
	//
	// required: true
	YamlData string `json:"yamlData" validate:"range=1:30720"`

	//
	// A definition of static properties (schema).
	// This field is mandatory.
	//
	// required: false
	// example: {"type": "object", "description": "JSON-schema for template (static) parameters", "properties": {}}
	StaticParameterSchema DataDriverParametersSchema `json:"staticParameterSchema,omitempty"`

	//
	// A definition of dynamic config properties (schema).
	// Every data driver instance can have more than one.
	// Skip this field if you want to turn the dynamic property configuration off.
	// Applicable for SOURCE data driver type only.
	//
	// required: false
	// example: {"type": "object", "description": "JSON-schema for dynamic config parameters", "properties": {}}
	ConfigParameterSchema DataDriverParametersSchema `json:"configParameterSchema,omitempty"`

	//
	// A definition of a stream properties (schema).
	// This field is mandatory.
	//
	// required: false
	// example: {"type": "object", "description": "JSON-schema for stream parameters", "properties": {}}
	StreamParameterSchema DataDriverParametersSchema `json:"streamParameterSchema,omitempty"`
}

// DataDriverClassRequestParam is DataDriverClass used as API parameter
// swagger:parameters DataDriverClassCreate DataDriverClassUpdate
// in: body
type DataDriverClassRequestParam struct {
	// Parameters and values used when creating or updating a data driver class
	// in: body
	// required: true
	Body *DataDriverClass `json:"body"`
}

// DataDriverClassGetResponse is a data driver class get response
//
// swagger:response DataDriverClassGetResponse
type DataDriverClassGetResponse struct {
	// in: body
	// required: true
	Payload *DataDriverClass
}

// DataDriverClassListResponsePayload is a data driver class listing payload
//
// payload for DataDriverClassListResponsePayload
type DataDriverClassListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of data driver classes
	// required: true
	ListOfDataDrivers []DataDriverClass `json:"result"`
}

// DataDriverClassListResponse is a a data driver class listing response
//
// swagger:response DataDriverClassListResponse
type DataDriverClassListResponse struct {
	// in: body
	// required: true
	Payload *DataDriverClassListResponsePayload
}

// swagger:parameters DataDriverClassList DataDriverClassGet DataDriverClassCreate DataDriverClassUpdate DataDriverClassDelete
// in: header
type dataDriverClassAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

func ValidateDataDriverClass(model *DataDriverClass) error {
	ctx := context.Background()
	if model == nil {
		return errcode.NewBadRequestError("DataDriverClass")
	}
	model.Name = strings.TrimSpace(model.Name)
	if len(model.Name) == 0 {
		return errcode.NewBadRequestError("Name")
	}

	model.Description = strings.TrimSpace(model.Description)
	model.DataDriverVersion = strings.TrimSpace(model.DataDriverVersion)

	if len(model.MinSvcDomainVersion) > 0 {
		model.DataDriverVersion = strings.TrimSpace(model.DataDriverVersion)
	}
	model.YamlData = strings.TrimSpace(model.YamlData)

	// validate type
	if model.Type != DataDriverSource && model.Type != DataDriverSink && model.Type != DataDriverBoth {
		return errcode.NewBadRequestError("Type")
	}

	// validate static schema
	if model.StaticParameterSchema != nil && len(model.StaticParameterSchema) > 0 {
		err := schema.ValidateSpecMap(ctx, model.StaticParameterSchema)
		if err != nil {
			return errcode.NewBadRequestError("StaticParameterSchema")
		}
	}

	// validate dynamic schema
	if model.ConfigParameterSchema != nil && len(model.ConfigParameterSchema) > 0 {
		err := schema.ValidateSpecMap(ctx, model.ConfigParameterSchema)
		if err != nil {
			return errcode.NewBadRequestError("ConfigParameterSchema")
		}
	}

	// validate stream schema
	if model.StreamParameterSchema != nil && len(model.StreamParameterSchema) > 0 {
		err := schema.ValidateSpecMap(ctx, model.StreamParameterSchema)
		if err != nil {
			return errcode.NewBadRequestError("StreamParameterSchema")
		}
	}

	return nil
}

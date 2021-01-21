package model

import (
	"cloudservices/common/errcode"
	"cloudservices/common/schema"
	"context"
	"strings"
)

// DataDriverConfig is object model for data driver instance's dynamic configuration
//
// A dynamic instance config represents a logical Data Source/Sink integration's dynamic configuration.
// swagger:model DataDriverConfig
type DataDriverConfig struct {
	// required: true
	BaseModel

	// required: true
	ServiceDomainBinding

	// required: true
	Name string `json:"name"`

	// required: false
	Description string `json:"description,omitempty"`

	// required: true
	DataDriverInstanceID string `json:"dataDriverInstanceID" validate:"range=1:200"`

	//
	// A list of sets of key-value pairs for dynamic configuration
	//
	// required: false
	// example: {"parameters": "values", "must_match": "JSON-schema from data driver class"}
	Parameters DataDriverParametersValues `json:"parameters,omitempty"`
}

// DataDriverConfigRequestParam is DataDriverConfig used as API parameter
// swagger:parameters DataDriverConfigCreate DataDriverConfigUpdate
// in: body
type DataDriverConfigRequestParam struct {
	// Parameters and values used when creating or updating a data driver config
	// in: body
	// required: true
	Body *DataDriverConfig `json:"body"`
}

// DataDriverConfigGetResponse is a data driver config get response
//
// swagger:response DataDriverConfigGetResponse
type DataDriverConfigGetResponse struct {
	// in: body
	// required: true
	Payload *DataDriverConfig
}

// DataDriverConfigListResponse is a data driver config listing payload
//
// swagger:response DataDriverConfigListResponse
type DataDriverConfigListResponse struct {
	// in: body
	// required: true
	Payload *DataDriverConfigListResponsePayload
}

// DataDriverConfigListResponsePayload is a data driver config listing payload
//
// payload for DataDriverConfigListResponsePayload
type DataDriverConfigListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of data driver configs
	// required: true
	ListOfDataDriverConfigs []DataDriverConfig `json:"result"`
}

// swagger:parameters DataDriverConfigList DataDriverConfigGet DataDriverConfigDelete DataDriverConfigUpdate DataDriverConfigCreate
// in: header
type dataDriverInstanceConfigAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

func ValidateDataDriverConfig(model *DataDriverConfig, sch *DataDriverParametersSchema, project *Project) error {
	ctx := context.Background()

	if model == nil {
		return errcode.NewBadRequestError("DataDriverConfig")
	}

	model.Name = strings.TrimSpace(model.Name)
	if len(model.Name) == 0 {
		return errcode.NewBadRequestError("Name")
	}

	model.Description = strings.TrimSpace(model.Description)

	model.DataDriverInstanceID = strings.TrimSpace(model.DataDriverInstanceID)
	if len(model.DataDriverInstanceID) == 0 {
		return errcode.NewBadRequestError("DataDriverInstanceID")
	}

	// validate parameters against schema
	if len(model.Parameters) > 0 {
		if sch == nil {
			return errcode.NewBadRequestError("Parameters")
		}
		err := schema.ValidateSchemaMap(ctx, *sch, model.Parameters)
		if err != nil {
			return errcode.NewBadRequestError("Parameters")
		}
	} else if sch != nil && len(*sch) > 0 {
		return errcode.NewBadRequestError("Parameters")
	}

	return ValidateServiceDomainBinding(&model.ServiceDomainBinding, project)
}

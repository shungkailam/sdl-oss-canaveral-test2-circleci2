package model

import (
	"cloudservices/common/errcode"
	"cloudservices/common/schema"
	"context"
	"strings"
)

// DataDriverInstance is object model for data driver instance
//
// A data driver instance represents a logical IoT Data Source/Sink integration.
// swagger:model DataDriverInstance
type DataDriverInstance struct {
	// required: true
	BaseModel

	// required: true
	DataDriverInstanceCore
}

type DataDriverInstanceCore struct {
	// required: true
	Name string `json:"name" validate:"range=1:200"`

	// required: false
	Description string `json:"description,omitempty"`

	// required: true
	DataDriverClassID string `json:"dataDriverClassID" validate:"range=1:200"`

	//
	// ID of parent project.
	//
	// required: true
	ProjectID string `json:"projectId" validate:"range=0:64"`

	//
	// A sets of key-value pairs for static configuration
	//
	// required: false
	// example: {"parameters": "values", "must_match": "JSON-schema from data driver class"}
	StaticParameters DataDriverParametersValues `json:"staticParameters,omitempty"`
}

// DataDriverInstanceRequestParam is DataDriverInstance used as API parameter
// swagger:parameters DataDriverInstanceCreate DataDriverInstanceUpdate
// in: body
type DataDriverInstanceRequestParam struct {
	// Parameters and values used when creating or updating a data driver instance
	// in: body
	// required: true
	Body *DataDriverInstance `json:"body"`
}

// DataDriverInstanceGetResponse is a data driver instance get response
//
// swagger:response DataDriverInstanceGetResponse
type DataDriverInstanceGetResponse struct {
	// in: body
	// required: true
	Payload *DataDriverInstance
}

// DataDriverInstanceListResponsePayload is a data driver instance listing payload
//
// payload for DataDriverInstanceListResponsePayload
type DataDriverInstanceListResponsePayload struct {
	// required: true
	EntityListResponsePayload

	// list of data driver instances
	// required: true
	ListOfDetaDriverInstances []DataDriverInstance `json:"result"`
}

// DataDriverClassInstanceListResponse is a data driver instance listing payload
//
// swagger:response DataDriverClassInstanceListResponse
type DataDriverClassInstanceListResponse struct {
	// in: body
	// required: true
	ListOfDetaDriverInstances []DataDriverInstance
}

// DataDriverInstanceListResponse is a a data driver instance listing response
//
// swagger:response DataDriverInstanceListResponse
type DataDriverInstanceListResponse struct {
	// in: body
	// required: true
	Payload *DataDriverInstanceListResponsePayload
}

// swagger:parameters DataDriverInstancesList DataDriverInstanceGet DataDriverInstanceCreate DataDriverInstanceUpdate DataDriverInstanceDelete DataDriverInstancesByClassIdList
// in: header
type dataDriverInstanceAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

func ValidateDataDriverInstance(model *DataDriverInstance, sch DataDriverParametersSchema) error {
	ctx := context.Background()
	if model == nil {
		return errcode.NewBadRequestError("DataDriverInstance")
	}
	model.Name = strings.TrimSpace(model.Name)
	if len(model.Name) == 0 {
		return errcode.NewBadRequestError("Name")
	}
	model.Description = strings.TrimSpace(model.Description)

	// validate parameters against schema
	if len(model.StaticParameters) > 0 {
		if sch == nil {
			return errcode.NewBadRequestError("StaticParameters")
		}
		err := schema.ValidateSchemaMap(ctx, sch, model.StaticParameters)
		if err != nil {
			return errcode.NewBadRequestError("StaticParameters")
		}
	} else if len(sch) > 0 {
		return errcode.NewBadRequestError("StaticParameters")
	}

	return nil
}

// DataDriverInstanceInventory is used as a websocket payload for Data Driver Instance
// swagger:model DataDriverInstanceInventory
type DataDriverInstanceInventory struct {
	// required: true
	BaseModel
	// required: true
	Doc DataDriverInstance `json:"doc"`
	// required: true
	Class DataDriverClass `json:"class"`
	// required: true
	YamlData string `json:"yamlData"`
	//required: true
	DataDriverConfigs []DataDriverConfig `json:"configs"`
	//required: true
	DataDriverStreams []DataDriverStream `json:"streams"`
}

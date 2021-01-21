package model

import (
	"cloudservices/common/errcode"
	"cloudservices/common/schema"
	"context"
	"strings"
)

const (
	// Types of the data driver stream directions
	DataDriverStreamSource DataDriverStreamDirection = "SOURCE"
	DataDriverStreamSink   DataDriverStreamDirection = "SINK"
)

// DataDriverStreamDirection is the type of the data driver class
// swagger:model DataDriverStreamDirection
// enum: SOURCE,SINK
type DataDriverStreamDirection string

// DataDriverStream is object model for stream originated by a data driver
//
// A dynamic instance config represents a logical Data Source/Sink integration's dynamic configuration.
// swagger:model DataDriverStream
type DataDriverStream struct {
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
	// Data driver stream direction
	//
	// required: true
	Direction DataDriverStreamDirection `json:"direction,omitempty"`

	//
	// A list of sets of key-value pairs for dynamic configuration
	//
	// required: true
	// example: {"stream": [ {"values", "must_match": "JSON-schema from data driver class"} ] }
	Stream DataDriverParametersValues `json:"stream,omitempty"`

	//
	// A list of Category labels for this data driver config.
	// "SOURCE" data driver streams should have at least 1 label.
	// "SINK" should not have any labels.
	//
	// required: false
	Labels []CategoryInfo `json:"labels"`
}

// DataDriverStreamRequestParam is DataDriverStream used as API parameter
// swagger:parameters DataDriverStreamCreate DataDriverStreamUpdate
// in: body
type DataDriverStreamRequestParam struct {
	// Parameters and values used when creating or updating a data driver config
	// in: body
	// required: true
	Body *DataDriverStream `json:"body"`
}

// DataDriverStreamGetResponse is a data driver config get response
//
// swagger:response DataDriverStreamGetResponse
type DataDriverStreamGetResponse struct {
	// in: body
	// required: true
	Payload *DataDriverStream
}

// DataDriverStreamListResponse is a data driver config listing payload
//
// swagger:response DataDriverStreamListResponse
type DataDriverStreamListResponse struct {
	// in: body
	// required: true
	Payload *DataDriverStreamListResponsePayload
}

// DataDriverStreamListResponsePayload is a data driver stream listing payload
//
// payload for DataDriverStreamListResponsePayload
type DataDriverStreamListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of data driver streams
	// required: true
	ListOfDataDriverStreams []DataDriverStream `json:"result"`
}

// swagger:parameters DataDriverStreamList DataDriverStreamGet DataDriverStreamCreate DataDriverStreamDelete DataDriverStreamUpdate
// in: header
type dataDriverInstanceStreamAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

func ValidateDataDriverStream(model *DataDriverStream, sch *DataDriverParametersSchema, project *Project) error {
	ctx := context.Background()

	if model == nil {
		return errcode.NewBadRequestError("DataDriverStream")
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

	if model.Direction == DataDriverStreamSource {
		if len(model.Labels) == 0 {
			return errcode.NewBadRequestError("Labels")
		}
	} else if model.Direction == DataDriverStreamSink {
		if len(model.Labels) != 0 {
			return errcode.NewBadRequestError("Labels")
		}
	} else {
		return errcode.NewBadRequestError("Direction")
	}

	// validate parameters against schema
	if len(model.Stream) > 0 {
		if sch == nil {
			return errcode.NewBadRequestError("Stream")
		}
		err := schema.ValidateSchemaMap(ctx, *sch, model.Stream)
		if err != nil {
			return errcode.NewBadRequestError("Stream")
		}
	} else if sch != nil && len(*sch) > 0 {
		return errcode.NewBadRequestError("Stream")
	}

	return ValidateServiceDomainBinding(&model.ServiceDomainBinding, project)
}

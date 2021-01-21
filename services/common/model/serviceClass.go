package model

import (
	"strings"
	"time"
)

// ServiceClassScopeType is the type of Service Class scope
// swagger:model ServiceClassScopeType
// enum: SERVICEDOMAIN,PROJECT
type ServiceClassScopeType string

// ServiceClassStateType is the state of Service Class
// swagger:model ServiceClassStateType
// enum: FINAL,DRAFT,DEPRECATED
type ServiceClassStateType string

const (
	// ServiceClassServiceDomainScope is the Service Domain scope
	ServiceClassServiceDomainScope = ServiceClassScopeType("SERVICEDOMAIN")
	// ServiceClassProjectScope is the Project scope
	ServiceClassProjectScope = ServiceClassScopeType("PROJECT")

	// ServiceClassFinalState is the final state
	ServiceClassFinalState = ServiceClassStateType("FINAL")
	// ServiceClassDraftState is the draft state
	ServiceClassDraftState = ServiceClassStateType("DRAFT")
	// ServiceClassDeprecatedState is the deprecated state
	ServiceClassDeprecatedState = ServiceClassStateType("DEPRECATED")
)

// Schema holds the definition for the schema
type Schema struct {
	Parameters map[string]interface{} `json:"parameters"`
}

// ServiceInstanceSchema holds the schema for Service Instance of the Service Class
type ServiceInstanceSchema struct {
	// Schema for creating the Service Instance
	Create Schema `json:"create,omitempty"`
	// Schema for updating the Service Instance
	Update Schema `json:"update,omitempty"`
}

// ServiceBindingSchema holds the schema for Service Binding
type ServiceBindingSchema struct {
	// Schema for creating the Servic Binding to the Service Instance
	Create Schema `json:"create,omitempty"`
}

// ServiceClassSchemas holds the schema for the Service Class
type ServiceClassSchemas struct {
	// Schema for Service Instance
	SvcInstance ServiceInstanceSchema `json:"svcInstance,omitempty"`
	// Schema for Service Binding
	SvcBinding ServiceBindingSchema `json:"svcBinding,omitempty"`
}

// ServiceClassCommon carries the basic common information identifying a Service Class
type ServiceClassCommon struct {
	// Type of the Service Class e.g Kafka
	// required: true
	Type string `json:"type" validate:"range=1:200"`
	// Version of the Service Class type
	// required: true
	SvcVersion string `json:"svcVersion" validate:"range=1:200"`
	// Scope of the Service Class e.g servicedomain or project
	// required: true
	Scope ServiceClassScopeType `json:"scope" validate:"range=1:200"`
	// Minimum version of the Service Domain supporting this Service Class
	// required: true
	MinSvcDomainVersion string `json:"minSvcDomainVersion" validate:"range=1:20"`
}

// ServiceClassTag holds the tags for a Service Class
//
// swagger:model ServiceClassTag
type ServiceClassTag struct {
	// Name of the tag
	Name string `json:"name,omitempty"`
	// Value of the tag
	Value string `json:"value,omitempty"`
}

// ServiceClass holds the definition including schemas for the managed service
//
// swagger:model ServiceClass
type ServiceClass struct {
	// required: true
	ServiceClassCommon
	ID string `json:"id" validate:"range=1:64,ignore=create"`
	// required: true
	Name        string `json:"name" validate:"range=1:200"`
	Description string `json:"description" validate:"range=0:1024"`
	// State of the Service Class
	// required: true
	State ServiceClassStateType `json:"state" validate:"range=1:200"`
	// Flag to specify if service binding is supported
	// required: true
	Bindable bool                `json:"bindable"`
	Schemas  ServiceClassSchemas `json:"schemas,omitempty"`
	// Tag name can be repeated to hold multiple values.
	// Tags essential = yes/no and category = some category are required
	Tags      []ServiceClassTag `json:"tags,omitempty"`
	Version   float64           `json:"version"`
	CreatedAt time.Time         `json:"createdAt"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

// swagger:parameters ServiceClassCreate
// in: body
type ServiceClassCreateParam struct {
	// in: body
	// required: true
	Body *ServiceClass `json:"body"`
}

// swagger:parameters ServiceClassUpdate
// in: body
type ServiceClassUpdateParam struct {
	// in: body
	// required: true
	Body *ServiceClass `json:"body"`
}

// Ok
// swagger:response ServiceClassListResponse
type ServiceClassListResponse struct {
	// in: body
	// required: true
	Payload *ServiceClassListPayload
}

// ServiceInstanceListPayload is the payload for ServiceClassListResponse
type ServiceClassListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of Service Classes
	// required: true
	SvcClassList []ServiceClass `json:"result"`
}

// Ok
// swagger:response ServiceClassGetResponse
type ServiceClassGetResponse struct {
	// in: body
	// required: true
	Payload *ServiceClass
}

// swagger:parameters ServiceClassCreate ServiceClassList ServiceClassUpdate ServiceClassGet ServiceClassDelete
// in: header
type serviceClassAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ServiceClassCommonQueryParam carries the common query parameters applicable to related models
type ServiceClassCommonQueryParam struct {
	// Scope of the Service Class
	// in: query
	// required: false
	Scope ServiceClassScopeType `json:"scope"`
	// Type of the Service Class
	// in: query
	// required: false
	Type string `json:"type"`
	// Version of the Service Class
	// in: query
	// required: false
	SvcVersion string `json:"svcVersion"`
}

// ServiceClassQueryParam carries the first class query parameters
// swagger:parameters ServiceClassList
// in: query
type ServiceClassQueryParam struct {
	// in: query
	// required: false
	ServiceClassCommonQueryParam
	// Tags on the Service Class
	// in: query
	// required: false
	Tags []string `json:"tags"`
}

// ParseTags parses the name=value string values to ServiceClassTag values
func (queryParam *ServiceClassQueryParam) ParseTags() ([]ServiceClassTag, error) {
	svcClassTags := []ServiceClassTag{}
	tags := queryParam.Tags
	if tags == nil {
		return svcClassTags, nil
	}
	for _, tag := range tags {
		parts := strings.SplitN(tag, "=", 2)
		if len(parts) == 1 {
			svcClassTag := ServiceClassTag{
				Name: strings.TrimSpace(parts[0]),
			}
			svcClassTags = append(svcClassTags, svcClassTag)
			continue
		}
		if len(parts) == 2 {
			svcClassTag := ServiceClassTag{
				Name:  strings.TrimSpace(parts[0]),
				Value: strings.TrimSpace(parts[1]),
			}
			svcClassTags = append(svcClassTags, svcClassTag)
			continue
		}
	}
	return svcClassTags, nil
}

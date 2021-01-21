package model

import (
	"time"
)

// ServiceInstanceStateType is the state type for Service Instance
// swagger:model ServiceInstanceStateType
// enum: PROVISIONING,PROVISIONED, FAILED
type ServiceInstanceStateType string

const (
	// ServiceInstanceProvisiongState represents Service Instance provisioning state
	ServiceInstanceProvisiongState = ServiceInstanceStateType("PROVISIONING")
	// ServiceInstanceProvisionedState represents Service Instance provisioned state
	ServiceInstanceProvisionedState = ServiceInstanceStateType("PROVISIONED")
	// ServiceInstanceFailedState represents Service Instance failed state
	ServiceInstanceFailedState = ServiceInstanceStateType("FAILED")

	// ServiceInstanceStatusProjectScopedEventPath is the event path template for Service Instance at Project scope
	ServiceInstanceStatusProjectScopedEventPath = "/serviceDomain:${svcDomainId}/project:${projectId}/service:${type}/instance:${svcInstanceId}/status"

	// ServiceInstanceStatusServiceDomainScopedEventPath is the event path template for Service Instance at Service Domain scope
	ServiceInstanceStatusServiceDomainScopedEventPath = "/serviceDomain:${svcDomainId}/service:${type}/instance:${svcInstanceId}/status"
)

// ServiceInstanceState holds the state for each entity e.g Service Domain
type ServiceInstanceState struct {
	// required: true
	SvcDomainID string `json:"svcDomainId"`
	// required: true
	State ServiceInstanceStateType `json:"state"`
	// required: false
	Description string `json:"description,omitempty"`
}

// ServiceInstanceCommon carries the basic common information identifying a Service Instance
type ServiceInstanceCommon struct {
	ServiceClassCommon
	SvcClassID   string `json:"svcClassId"`
	ScopeID      string `json:"scopeId"`
	SvcClassName string `json:"svcClassName"`
}

// ServiceInstance holds the Service Instance information
//
// swagger:model ServiceInstance
type ServiceInstance struct {
	BaseModel
	ServiceInstanceCommon
	Name        string                 `json:"name" validate:"range=1:200"`
	Description string                 `json:"description,omitempty" validate:"range=0:200"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// ServiceInstanceStatus represents the status of the Service Instance
//
// swagger:model ServiceInstanceStatus
type ServiceInstanceStatus struct {
	// required: true
	ServiceInstanceState
	// required: true
	SvcInstanceID string `json:"svcInstanceId"`
	// Properties emitted by the instance
	Properties map[string]interface{} `json:"properties"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
}

// ServiceInstanceParam holds the common parameters for creating or updating Service Instance
//
// swagger:model ServiceInstanceParam
type ServiceInstanceParam struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// required: true
	SvcClassID string `json:"svcClassId"`
	// required: true
	ScopeID    string                 `json:"scopeId"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

// swagger:parameters ServiceInstanceCreate
// in: body
type ServiceInstanceCreateParam struct {
	// in: body
	// required: true
	Body *ServiceInstanceParam `json:"body"`
}

// swagger:parameters ServiceInstanceUpdate
// in: body
type ServiceInstanceUpdateParam struct {
	// in: body
	// required: true
	Body *ServiceInstanceParam `json:"body"`
}

// Ok
// swagger:response ServiceInstanceListResponse
type ServiceInstanceListResponse struct {
	// in: body
	// required: true
	Payload *ServiceInstanceListPayload
}

// ServiceInstanceListPayload is the payload for ServiceInstanceListResponse
type ServiceInstanceListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of Service Instances
	// required: true
	SvcInstanceList []ServiceInstance `json:"result"`
}

// Ok
// swagger:response ServiceInstanceGetResponse
type ServiceInstanceGetResponse struct {
	// in: body
	// required: true
	Payload *ServiceInstance
}

// Ok
// swagger:response ServiceInstanceStatusListResponse
type ServiceInstanceStatusListResponse struct {
	// in: body
	// required: true
	Payload *ServiceInstanceStatusListPayload
}

// ServiceInstanceStatusListPayload is the payload for ServiceInstanceStatusListResponse
type ServiceInstanceStatusListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of Service Instance Status
	// required: true
	SvcInstanceStatusList []ServiceInstanceStatus `json:"result"`
}

// swagger:parameters ServiceInstanceCreate ServiceInstanceList ServiceInstanceUpdate ServiceInstanceGet ServiceInstanceDelete ServiceInstanceStatusList
// in: header
type serviceInstanceAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ServiceInstanceQueryParam carries the first class query parameters
// swagger:parameters ServiceInstanceList
// in: query
type ServiceInstanceQueryParam struct {
	ServiceClassCommonQueryParam
	// Service Class ID
	// in: query
	// required: false
	SvcClassID string `json:"svcClassId"`
	// Service Class scope ID
	// in: query
	// required: false
	ScopeID string `json:"scopeId"`
}

// ServiceInstanceStatusQueryParam carries the first class query parameters
// swagger:parameters ServiceInstanceStatusList
// in: query
type ServiceInstanceStatusQueryParam struct {
	SvcDomainID string `json:"svcDomainId"`
}

// ObjectRequestBaseServiceInstance is used as a websocket payload for Service Instance
// swagger:model ObjectRequestBaseServiceInstance
type ObjectRequestBaseServiceInstance struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ServiceInstance `json:"doc"`
}

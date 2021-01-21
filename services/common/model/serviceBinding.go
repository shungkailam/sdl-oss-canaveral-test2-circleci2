package model

import "time"

// ServiceBindingStateType is the state type for Service Binding
// swagger:model ServiceBindingStateType
// enum: PROVISIONING,PROVISIONED, FAILED
type ServiceBindingStateType string

// ServiceBindingResourceType is the type of the requesting resource
// swagger:model ServiceBindingResourceType
// enum: SERVICEDOMAIN,PROJECT
type ServiceBindingResourceType string

const (
	// ServiceBindingProvisiongState represents Service Binding provisioning state
	ServiceBindingProvisiongState = ServiceBindingStateType("PROVISIONING")
	// ServiceBindingProvisionedState represents Service Binding provisioned state
	ServiceBindingProvisionedState = ServiceBindingStateType("PROVISIONED")
	// ServiceBindingFailedState represents Service Binding failed state
	ServiceBindingFailedState = ServiceBindingStateType("FAILED")

	// ServiceBindingServiceDomainResource is the Service Domain resource
	ServiceBindingServiceDomainResource = ServiceBindingResourceType("SERVICEDOMAIN")
	// ServiceBindingProjectResource is the Project resource
	ServiceBindingProjectResource = ServiceBindingResourceType("PROJECT")

	// ServiceBindingStatusProjectScopedEventPath is the event path template for Service Binding at Project scope
	ServiceBindingStatusProjectScopedEventPath = "/serviceDomain:${svcDomainId}/project:${projectId}/service:${type}/instance:${svcInstanceId}/binding:${svcBindingId}/status"

	// ServiceBindingStatusServiceDomainScopedEventPath is the event path template for Service Binding at Service Domain scope
	ServiceBindingStatusServiceDomainScopedEventPath = "/serviceDomain:${svcDomainId}/service:${type}/instance:${svcInstanceId}/binding:${svcBindingId}/status"
)

// ServiceBindingState holds the state for each entity e.g Service Domain
type ServiceBindingState struct {
	// required: true
	SvcDomainID string `json:"svcDomainId"`
	// required: true
	State ServiceBindingStateType `json:"state"`
	// required: false
	Description string `json:"description"`
}

// ServiceBindingResult is the result information of a Service Binding
type ServiceBindingResult struct {
	Credentials map[string]interface{} `json:"credentials"`
	Endpoints   map[string]interface{} `json:"endpoints"`
}

// ServiceBindingResource is the binding resource to be bound to the Service Instance
type ServiceBindingResource struct {
	Type ServiceBindingResourceType `json:"type" validate:"range=1,200"`
	ID   string                     `json:"id" validate:"range=1:60"`
}

// ServiceBinding holds the Service Binding information
//
// swagger:model ServiceBinding
type ServiceBinding struct {
	// required: true
	BaseModel
	// required: true
	ServiceClassCommon
	// required: true
	Name         string                  `json:"name" validate:"range=1:200"`
	Description  string                  `json:"description,omitempty" validate:"range=0:200"`
	SvcClassID   string                  `json:"svcClassId"`
	SvcClassName string                  `json:"svcClassName"`
	BindResource *ServiceBindingResource `json:"bindResource,omitempty"`
	Parameters   map[string]interface{}  `json:"parameters,omitempty"`
}

// ServiceBindingStatus holds the Service Binding result information
//
// swagger:model ServiceBindingStatus
type ServiceBindingStatus struct {
	// required: true
	ServiceBindingState
	// required: true
	SvcBindingID  string                `json:"svcBindingId"`
	SvcInstanceID string                `json:"svcInstanceId,omitempty"`
	BindResult    *ServiceBindingResult `json:"bindResult,omitempty"`
	CreatedAt     time.Time             `json:"createdAt"`
	UpdatedAt     time.Time             `json:"updatedAt"`
}

// ServiceBindingParam holds the common parameters for creating Service Binding
//
// swagger:model ServiceBindingParam
type ServiceBindingParam struct {
	ID string `json:"id"`
	// required: true
	Name        string `json:"name"`
	Description string `json:"description"`
	// required: true
	SvcClassID   string                  `json:"svcClassId"`
	BindResource *ServiceBindingResource `json:"bindResource,omitempty"`
	Parameters   map[string]interface{}  `json:"parameters,omitempty"`
}

// swagger:parameters ServiceBindingCreate
// in: body
type ServiceBindingCreateParam struct {
	// in: body
	// required: true
	Body *ServiceBindingParam `json:"body"`
}

// Ok
// swagger:response ServiceBindingListResponse
type ServiceBindingListResponse struct {
	// in: body
	// required: true
	Payload *ServiceBindingListPayload
}

// ServiceBindingListPayload is the payload for ServiceBindingListResponse
type ServiceBindingListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of Service Bindings
	// required: true
	SvcBindingList []ServiceBinding `json:"result"`
}

// Ok
// swagger:response ServiceBindingGetResponse
type ServiceBindingGetResponse struct {
	// in: body
	// required: true
	Payload *ServiceBinding
}

// Ok
// swagger:response ServiceBindingStatusListResponse
type ServiceBindingStatusListResponse struct {
	// in: body
	// required: true
	Payload *ServiceBindingStatusListPayload
}

// ServiceBindingStatusListPayload is the payload for ServiceBindingStatusListResponse
type ServiceBindingStatusListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of Service Binding Status
	// required: true
	SvcBindingStatusList []ServiceBindingStatus `json:"result"`
}

// swagger:parameters ServiceBindingCreate ServiceBindingList ServiceBindingGet ServiceBindingDelete ServiceBindingStatusList
// in: header
type serviceBindingAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ServiceBindingQueryParam carries the first class query parameters
// swagger:parameters ServiceBindingList
// in: query
type ServiceBindingQueryParam struct {
	// Service Class ID
	// in: query
	// required: false
	SvcClassID string `json:"svcClassId"`
	// Bind resource type
	// in: query
	// required: false
	BindResourceType string `json:"bindResourceType"`
	// Bind resource ID
	// in: query
	// required: false
	BindResourceID string `json:"bindResourceId"`
}

// ServiceBindingStatusQueryParam carries the first class query parameters
// swagger:parameters ServiceBindingStatusList
// in: query
type ServiceBindingStatusQueryParam struct {
	// Service Domain ID
	// in: query
	// required: false
	SvcDomainID string `json:"svcDomainId"`
	// Service Instance ID
	// in: query
	// required: false
	SvcInstanceID string `json:"svcInstanceId"`
}

// ObjectRequestBaseServiceBinding is used as a websocket payload for Service Binding
// swagger:model ObjectRequestBaseServiceBinding
type ObjectRequestBaseServiceBinding struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ServiceBinding `json:"doc"`
}

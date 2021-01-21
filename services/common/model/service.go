package model

// Service is object model for service
//
// swagger:model Service
type Service struct {
	// required: true
	// enum:IoT,PaaS
	ServiceType string `json:"serviceType"`
}

// Ok
// swagger:response ServiceListResponse
type ServiceListResponse struct {
	// in: body
	// required: true
	Payload *Service
}

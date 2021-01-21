package model

// ServiceDomainInfo has service domain information
//
// swagger:model ServiceDomainInfo
type ServiceDomainInfo struct {
	ServiceDomainEntityModel
	Artifacts map[string]interface{} `json:"artifacts,omitempty"`
	Features  Features               `json:"features"`
}

// ServiceDomainInfoUpdateParam is the swagger wrapper around ServiceDomainInfo
// swagger:parameters ServiceDomainInfoUpdate
// in: body
type ServiceDomainInfoUpdateParam struct {
	// Describes parameters used to create or update a ServiceDomainInfo
	// in: body
	// required: true
	Body *ServiceDomainInfo `json:"body"`
}

// Ok
// swagger:response ServiceDomainInfoGetResponse
type ServiceDomainInfoGetResponse struct {
	// in: body
	// required: true
	Payload *ServiceDomainInfo
}

// swagger:parameters ServiceDomainInfoList ProjectGetServiceDomainsInfo ServiceDomainInfoGet ServiceDomainInfoUpdate
// in: header
type serviceDomainInfoAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response ServiceDomainInfoListResponse
type ServiceDomainInfoListResponse struct {
	// in: body
	// required: true
	Payload *ServiceDomainInfoListPayload
}

// ServiceDomainInfoListPayload is the payload for ServiceDomainInfoListResponse
type ServiceDomainInfoListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of service domain info
	// required: true
	SvcDomainInfoList []ServiceDomainInfo `json:"result"`
}

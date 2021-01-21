package model

// CloudProfile is the object model for cloud credentials.
// swagger:model CloudProfile
type CloudProfile struct {
	// required: true
	CloudCreds
}

// CloudProfileCreateParam is CloudProfile used as API parameter
// swagger:parameters CloudProfileCreate
type CloudProfileCreateParam struct {
	// Description for the cloud profile.
	// in: body
	// required: true
	Body *CloudProfile `json:"body"`
}

// CloudProfileUpdateParam is CloudProfile used as API parameter
// swagger:parameters CloudProfileUpdate
type CloudProfileUpdateParam struct {
	// in: body
	// required: true
	Body *CloudProfile `json:"body"`
}

// Ok
// swagger:response CloudProfileGetResponse
type CloudProfileGetResponse struct {
	// in: body
	// required: true
	Payload *CloudProfile
}

// Ok
// swagger:response CloudProfileListResponse
type CloudProfileListResponse struct {
	// in: body
	// required: true
	Payload *CloudProfileListResponsePayload
}

// payload for CloudProfileListResponse
type CloudProfileListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of cloud profiles
	// required: true
	CloudProfileList []CloudProfile `json:"result"`
}

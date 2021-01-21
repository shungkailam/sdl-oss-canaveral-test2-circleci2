package model

// AggregateSpec is payload for common aggregate request
// in: body
type AggregateSpec struct {
	// required: true
	Type string `json:"type"`
	// required: true
	Field string `json:"field"`
}

// swagger:parameters CommonAggregates
// in: body
type CommonAggregatesParam struct {
	// in: body
	// required: true
	AggregateSpec *AggregateSpec
}

// NestedAggregateSpec is payload for nested aggregate request
// in: body
type NestedAggregateSpec struct {
	// required: true
	AggregateSpec
	// required: true
	NestedField string `json:"nestedField"`
}

// swagger:parameters CommonNestedAggregates
// in: body
type CommonNestedAggregatesParam struct {
	// in: body
	// required: true
	AggregateSpec *NestedAggregateSpec
}

// AggregateInfo is aggregate query response item
// swagger:response AggregateInfo
type AggregateInfo struct {
	// required: true
	DocCount int `json:"doc_count" db:"doc_count"`
	// required: true
	Key string `json:"key" db:"key"`
}

// swagger:parameters CommonAggregates CommonNestedAggregates
// in: header
type commonAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response CommonAggregatesResponse
type CommonAggregatesResponse struct {
	// in: body
	// required: true
	Payload *[]AggregateInfo
}

// Ok
// swagger:response CommonNestedAggregatesResponse
type CommonNestedAggregatesResponse struct {
	// in: body
	// required: true
	Payload *[]AggregateInfo
}

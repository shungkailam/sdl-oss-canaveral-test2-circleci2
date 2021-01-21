package model

// Function is object model for function
//
// Function represent lambdas:
// functions or transformations that can be applied
// to Data Pipelines.
// Functions are tenant-wide objects and the same function
// may be run within an edge, across all edges of a tenant
// or on tenant data in the cloud.
//
// swagger:model Function
type Function struct {
	// required: true
	Script
}

// FunctionCreateParam is Function used as API parameter
// swagger:parameters FunctionCreate
type FunctionCreateParam struct {
	// Describes the function creation request
	// in: body
	// required: true
	Body *Function `json:"body"`
}

// FunctionUpdateParam is Function used as API parameter
// swagger:parameters FunctionUpdate
type FunctionUpdateParam struct {
	// in: body
	// required: true
	Body *Function `json:"body"`
}

// Ok
// swagger:response FunctionGetResponse
type FunctionGetResponse struct {
	// in: body
	// required: true
	Payload *Function
}

// Ok
// swagger:response FunctionListResponse
type FunctionListResponse struct {
	// in: body
	// required: true
	Payload *FunctionListPayload
}

// payload for FunctionListResponse
type FunctionListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of functions
	// required: true
	FunctionList []Function `json:"result"`
}

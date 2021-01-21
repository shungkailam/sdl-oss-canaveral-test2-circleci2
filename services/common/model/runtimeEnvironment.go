package model

// RuntimeEnvironment is the DB object and object model for function runtime.
//
// A RuntimeEnvironment is a Docker container runtime for functions.
// Karbon Platform Services includes several RuntimeEnvironments for built-in (and user defined) functions.
// User can also create custom RuntimeEnvironments which may be
// derived from Karbon Platform Services built-in RuntimeEnvironments.
//
// swagger:model RuntimeEnvironment
type RuntimeEnvironment struct {
	// required: true
	ScriptRuntime
}

// RuntimeEnvironmentCreateParam is RuntimeEnvironment used as API parameter
// swagger:parameters RuntimeEnvironmentCreate
type RuntimeEnvironmentCreateParam struct {
	// Describes the runtime environment creation request.
	// in: body
	// required: true
	Body *RuntimeEnvironment `json:"body"`
}

// RuntimeEnvironmentUpdateParam is RuntimeEnvironment used as API parameter
// swagger:parameters RuntimeEnvironmentUpdate
type RuntimeEnvironmentUpdateParam struct {
	// in: body
	// required: true
	Body *RuntimeEnvironment `json:"body"`
}

// Ok
// swagger:response RuntimeEnvironmentGetResponse
type RuntimeEnvironmentGetResponse struct {
	// in: body
	// required: true
	Payload *RuntimeEnvironment
}

// Ok
// swagger:response RuntimeEnvironmentListResponse
type RuntimeEnvironmentListResponse struct {
	// in: body
	// required: true
	Payload *RuntimeEnvironmentListPayload
}

// payload for RuntimeEnvironmentListResponse
type RuntimeEnvironmentListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of runtime environments
	// required: true
	RuntimeEnvironmentList []RuntimeEnvironment `json:"result"`
}

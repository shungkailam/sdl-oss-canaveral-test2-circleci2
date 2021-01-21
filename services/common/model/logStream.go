package model

// LogStream is object model for requesting log streams.
//
// LogStream is used to specify the edge, app or pipeline, and container
// to stream logs from.
//
// swagger:model LogStream
type LogStream struct {
	//
	// Edge ID from which logs will be streamed.
	//
	// required: true
	EdgeID string `json:"edgeId" validate:"range=1:36"`
	//
	// ID of the application
	//
	// required: false
	ApplicationID string `json:"applicationId" validate:"range=1:36"`
	//
	// ID of the data pipeline
	//
	// required: false
	DataPipelineID string `json:"dataPipelineId" validate:"range=1:36"`
	//
	// Name of the kubernetes container in the pod to
	// stream logs from.
	//
	// required: true
	Container string `json:"container" validate:"range=0:200"`
}

// LogStreamEndpointsParam is LogStream struct used as API parameter
//
// swagger:parameters LogStreamEndpoints
type LogStreamEndpointsParam struct {
	// A description of the log streaming request.
	// in: body
	// required: true
	Request *LogStream `json:"request"`
}

// LogStreamResponsePayload is the url to which logs
// will be streamed.
//
// swagger:model LogStreamResponsePayload
type LogStreamEndpointsResponsePayload struct {
	// URL to which logs are being streamed.
	// in: body
	// required: true
	URL string `json:"url"`
}

// LogStreamEndpointsResponse encapsulates the response sent to clients
// that request for log streams.
//
// swagger:response LogStreamEndpointsResponse
type LogStreamEndpointsResponse struct {
	// in: body
	// required: true
	Payload *LogStreamEndpointsResponsePayload
}

// swagger:parameters LogStreamEndpoints
// in: header
type logStreamAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseLogStream is used as websocket LogStream message.
//
// swagger:model ObjectRequestBaseLogStream
type ObjectRequestBaseLogStream struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc WSMessagingLogStream `json:"doc"`
}

// WSMessagingLogStream is part of the websocket message sent
// to the edge to start streaming logs.
//
// swagger: model WSMessagingLogStream
type WSMessagingLogStream struct {
	LogStreamInfo *LogStream
	ProjectID     string
	URL           string
}

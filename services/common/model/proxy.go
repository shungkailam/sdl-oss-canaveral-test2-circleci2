package model

const (
	HTTP_PROXY_MESSAGE = "httpProxy"
)

// ProxyRequest is used as websocket httpProxy request message
// swagger:model ProxyRequest
type ProxyRequest struct {
	// required: true
	URL string `json:"url"`
	// required: true
	Request []byte `json:"request"`
}

// ProxyResponse is used as websocket httpProxy response message
// swagger:model ProxyResponse
type ProxyResponse struct {
	// required: true
	Status string `json:"status"`
	// required: true
	StatusCode int    `json:"statusCode"`
	Response   []byte `json:"response"`
}

// swagger:model
// Proxy call placeholder payload
type ProxyCallPayload struct {
}

// ProxyCallParam is API parameter placeholder for proxy calls
// These are not really used since proxy call can take
// potentially any kind of parameters.
// swagger:parameters ProxyPostCall ProxyGetCall ProxyPutCall ProxyDeleteCall
type ProxyCallParam struct {
	// in: body
	Body *ProxyCallPayload `json:"body"`
}

// swagger:parameters ProxyPostCall ProxyGetCall ProxyPutCall ProxyDeleteCall
// in: header
type proxyAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// swagger:model
// Proxy call response placeholder payload
type ProxyResponsePayload struct {
}

// Ok
// swagger:response ProxyCallResponse
type ProxyCallResponse struct {
	// in: body
	Payload *ProxyResponsePayload
}

package model

import (
	"fmt"
	"strings"
)

// swagger:model
// SSH Tunneling enable, configure request.
type WstunRequest struct {
	// required: true
	ServiceDomainID string `json:"serviceDomainId" validate:"range=1:36"`
	// ntnx:ignore
	// optional: endpoint = ip:port, if empty, ssh is assumed
	Endpoint string `json:"endpoint,omitempty"`

	// ntnx:ignore
	// optional: whether the endpoint is TLS, only for non ssh
	TLSEndpoint bool `json:"tlsEndpoint,omitempty"`

	// ntnx:ignore
	// optional: whether to skip TLS certification verification for endpoint,
	// only relevant when TLSEndpoint is true
	SkipCertVerification bool `json:"skipCertVerification,omitempty"`
}

func (doc WstunRequest) GetProxyEndpointPath() string {
	return fmt.Sprintf("%s-%s", doc.ServiceDomainID, strings.ReplaceAll(doc.Endpoint, ":", "."))
}

// swagger:model
// SSH Tunneling shut down request.
type WstunTeardownRequest struct {
	// required: true
	ServiceDomainID string `json:"serviceDomainId" validate:"range=1:36"`
	// required: true
	PublicKey string `json:"publicKey"`
	// ntnx:ignore
	// required: false
	// must match endpoint used in setup
	Endpoint string `json:"endpoint,omitempty"`
}

type WstunRequestInternal struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	ServiceDomainID string `json:"serviceDomainId" validate:"range=1:36"`
	// optional: endpoint = ip:port, if empty, ssh is assumed
	Endpoint string `json:"endpoint,omitempty"`
}

type WstunTeardownRequestInternal struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	ServiceDomainID string `json:"serviceDomainId" validate:"range=1:36"`
	// required: true
	PublicKey string `json:"publicKey"`
	// optional: endpoint = ip:port, if empty, ssh is assumed
	Endpoint string `json:"endpoint,omitempty"`
}

// swagger:model
// SSH Tunneling setup response payload
type WstunPayload struct {
	// required: true
	WstunRequest
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Port uint32 `json:"port" validate:"range=20000:32767"`
	// required: true
	Expiration int64 `json:"expiration"`
	// required: true
	PublicKey string `json:"publicKey"`
	// required: true
	PrivateKey string `json:"privateKey"`
	// ntnx:ignore
	// required: false
	URL string `json:"url,omitempty"`
}

// SetupSSHTunnelingParam is WstunRequest used as API parameter
// swagger:parameters SetupSSHTunneling
type SetupSSHTunnelingParam struct {
	// SSH Tunneling setup request param
	// in: body
	// required: true
	Body *WstunRequest `json:"body"`
}

// TeardownSSHTunnelingParam is WstunRequest used as API parameter
// swagger:parameters TeardownSSHTunneling
type TeardownSSHTunnelingParam struct {
	// SSH Tunneling teardown request param
	// in: body
	// required: true
	Body *WstunTeardownRequest `json:"body"`
}

// swagger:parameters SetupSSHTunneling TeardownSSHTunneling
// in: header
type sshAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response SetupSSHTunnelingResponse
type SetupSSHTunnelingResponse struct {
	// SSH Tunneling setup response
	// in: body
	// required: true
	Payload *WstunPayload
}

// Ok
// swagger:response TeardownSSHTunnelingResponse
type TeardownSSHTunnelingResponse struct {
}

// ObjectRequestSetupSSHTunneling is used as websocket setupSSHTunneling message
// swagger:model ObjectRequestSetupSSHTunneling
type ObjectRequestSetupSSHTunneling struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc WstunPayload `json:"doc"`
}

// ObjectRequestTeardownSSHTunneling is used as websocket teardownSSHTunneling message
// swagger:model ObjectRequestTeardownSSHTunneling
type ObjectRequestTeardownSSHTunneling struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc WstunTeardownRequest `json:"doc"`
}

func (r WstunTeardownRequestInternal) ToRequest() WstunRequestInternal {
	return WstunRequestInternal{
		TenantID:        r.TenantID,
		ServiceDomainID: r.ServiceDomainID,
		Endpoint:        r.Endpoint,
	}

}

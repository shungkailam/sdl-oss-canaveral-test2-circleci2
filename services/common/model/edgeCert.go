package model

type EdgeCertCore struct {
	//
	// Certificate for the edge using old/fixed root CA.
	//
	// required: true
	Certificate string `json:"certificate" db:"certificate" validate:"range=0:4096"`
	//
	// Encrypted private key using old/fixed root CA.
	//
	// required: true
	PrivateKey string `json:"privateKey" db:"private_key" validate:"range=0:4096"`
	//
	// Root CA certificate for the tenant.
	//
	// required: true
	CACertificate string `json:"CACertificate" validate:"range=0:4096"`
	//
	// Certificate for mqtt client on the edge
	//
	// required: true
	ClientCertificate string `json:"clientCertificate" db:"client_certificate" validate:"range=0:4096"`
	//
	// Encrypted private key corresponding to the client certificate.
	//
	// required: true
	ClientPrivateKey string `json:"clientPrivateKey" db:"client_private_key" validate:"range=0:4096"`
	//
	// Certificate for the edge using per-tenant root CA.
	//
	// required: true
	EdgeCertificate string `json:"edgeCertificate" db:"edge_certificate" validate:"range=0:4096"`
	//
	// Encrypted private key using per-tenant root CA.
	//
	// required: true
	EdgePrivateKey string `json:"edgePrivateKey" db:"edge_private_key" validate:"range=0:4096"`
	// For security purpose, EdgeCert can only be
	// retrieved once during edge on-boarding.
	// After that locked will be set to true and
	// the REST API endpoint for getting EdgeCert
	// will throw error.
	//
	// required: true

	Locked bool `json:"locked" db:"locked"`
}

// EdgeCert is DB and object model for data source
// swagger:model EdgeCert
type EdgeCert struct {
	// required: true
	EdgeBaseModel
	// required: true
	EdgeCertCore
}

// EdgeCertCreateParam is EdgeCert used as API parameter
// swagger:parameters EdgeCertCreate
type EdgeCertCreateParam struct {
	// This is a edgecerts creation request description
	// in: body
	// required: true
	Body *EdgeCert `json:"body"`
}

// EdgeCertUpdateParam is EdgeCert used as API parameter
// swagger:parameters EdgeCertUpdate
type EdgeCertUpdateParam struct {
	// in: body
	// required: true
	Body *EdgeCert `json:"body"`
}

// Ok
// swagger:response EdgeCertGetResponse
type EdgeCertGetResponse struct {
	// in: body
	// required: true
	Payload *EdgeCert
}

// Ok
// swagger:response EdgeCertListResponse
type EdgeCertListResponse struct {
	// in: body
	// required: true
	Payload *[]EdgeCert
}

// Ok
// swagger:response EdgeCertListResponseV2
type EdgeCertListResponseV2 struct {
	// in: body
	// required: true
	Payload *EdgeCertListPayload
}

// payload for EdgeCertListResponseV2
type EdgeCertListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge certs
	// required: true
	EdgeCertList []EdgeCert `json:"result"`
}

// swagger:parameters EdgeCertList EdgeCertListV2 EdgeCertGet EdgeCertCreate EdgeCertUpdate EdgeCertDelete EdgeClusterSetCertLock ServiceDomainSetCertLock
// in: header
type edgeCertAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseEdgeCert is used as websocket EdgeCert message
// swagger:model ObjectRequestBaseEdgeCert
type ObjectRequestBaseEdgeCert struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc EdgeCert `json:"doc"`
}

// EdgeCertLockPayload is payload for EdgeClusterSetCertLock
// swagger:parameters EdgeClusterSetCertLock ServiceDomainSetCertLock
type EdgeCertLockPayload struct {
	// in: body
	// required: true
	Body *EdgeCertLockParam `json:"body"`
}

// EdgeCertLockParam describes payload for SetEdgeCertLock operation
// swagger:model EdgeCertLockParam
type EdgeCertLockParam struct {
	// required: true
	EdgeClusterID string `json:"edgeClusterId"`
	// required: true
	Locked bool `json:"locked"`
	// If Locked is false and DurationSeconds is greater than 0,
	// then first unlock the edge certification,
	// then auto lock it after DurationSeconds seconds.
	//
	// required: false
	DurationSeconds int `json:"durationSeconds"`
}

// Ok
// swagger:response EmptyResponse
type EmptyResponse struct {
}

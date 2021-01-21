package model

// Certificates is DB and object model for a device's certificates
// swagger:model Certificates
type Certificates struct {
	//
	// Certificate for a device.
	//
	// required: true
	Certificate string `json:"certificate" db:"certificate" validate="range=0:4096"`
	//
	// Encrypted private key corresponding to the certificate.
	//
	// required: true
	PrivateKey string `json:"privateKey" db:"private_key" validate="range=0:4096"`
	//
	// Root CA certificate that signed the device certificate.
	//
	// required: true
	CACertificate string `json:"CACertificate" validate="range=0:4096"`
}

// Ok
// swagger:response CertificatesCreateResponse
type CertificatesCreateResponse struct {
	// in: body
	// required: true
	Payload *Certificates
}

// swagger:parameters CertificatesCreate CertificatesCreateV2 CertificatesDelete
// in: header
type certificatesAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

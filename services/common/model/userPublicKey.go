package model

import "time"

// swagger:model UserPublicKey
type UserPublicKey struct {
	// ID of the user
	// required: true
	ID string `json:"id" db:"id"`
	// Tenant ID of the user
	// required: true
	TenantID string `json:"tenantId" db:"tenant_id"`
	// Public Key of the user
	// required: true
	PublicKey string `json:"publicKey" db:"public_key"`
	// created at timestamp
	// required: true
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// updated at timestamp
	// required: true
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	// last used timestamp
	// required: true
	UsedAt time.Time `json:"usedAt" db:"used_at"`
}

// swagger:model UserPublicKeyUpdatePayload
type UserPublicKeyUpdatePayload struct {
	// Public Key of the user
	// required: true
	PublicKey string `json:"publicKey" db:"public_key"`
}

// swagger:parameters UserPublicKeyList UserPublicKeyGet UserPublicKeyDelete UserPublicKeyUpdate
// in: header
type userPublicKeyAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response UserPublicKeyListResponse
type UserPublicKeyListResponse struct {
	// in: body
	// required: true
	Payload *[]UserPublicKey
}

// Ok
// swagger:response UserPublicKeyGetResponse
type UserPublicKeyGetResponse struct {
	// in: body
	// required: true
	Payload *UserPublicKey
}

// UserPublicKeyUpdateParam is UserPublicKey used as API parameter
// swagger:parameters UserPublicKeyUpdate
type UserPublicKeyUpdateParam struct {
	// in: body
	// required: true
	Body *UserPublicKeyUpdatePayload `json:"body"`
}

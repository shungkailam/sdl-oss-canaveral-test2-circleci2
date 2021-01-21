package model

import "time"

// swagger:model UserApiToken
type UserApiToken struct {
	// ID of the user API token
	ID string `json:"id" db:"id"`
	// ntnx:ignore
	// Tenant ID
	TenantID string `json:"tenantId" db:"tenant_id"`
	// User ID
	// required: true
	UserID string `json:"userId" db:"user_id"`
	// Whether the token is active
	// required: true
	Active bool `json:"active" db:"active"`
	// created at timestamp
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// updated at timestamp
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	// last used timestamp
	UsedAt time.Time `json:"usedAt" db:"used_at"`
}

// swagger:model UserApiTokenCreatePayload
type UserApiTokenCreatePayload struct {
	// Whether the token is active
	// required: true
	Active bool `json:"active" db:"active"`
}

// swagger:model UserApiTokenCreated
type UserApiTokenCreated struct {
	// ID of the user API token
	// required: true
	ID string `json:"id"`
	// Tenant ID
	// required: true
	TenantID string `json:"tenantId"`
	// User ID
	// required: true
	UserID string `json:"userId"`
	// JWT token. User must save away this token.
	// Karbon Platform Services does not store this token and it will not be returned
	// by any subsequent API calls.
	// required: true
	Token string `json:"token"`
}

// swagger:parameters UserApiTokenList UserApiTokenGet UserApiTokenCreate UserApiTokenUpdate UserApiTokenDelete
// in: header
type userApiTokenAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response UserApiTokenListResponse
type UserApiTokenListResponse struct {
	// in: body
	// required: true
	Payload *[]UserApiToken
}

// UserApiTokenUpdateParam is UserApiToken used as API parameter
// swagger:parameters UserApiTokenUpdate
type UserApiTokenUpdateParam struct {
	// in: body
	// required: true
	Body *UserApiToken `json:"body"`
}

// UserApiTokenCreateParam is used in UserApiTokenCreate POST body
// swagger:parameters UserApiTokenCreate
type UserApiTokenCreateParam struct {
	// in: body
	// required: true
	Body *UserApiTokenCreatePayload `json:"body"`
}

// Ok
// swagger:response UserApiTokenCreateResponse
type UserApiTokenCreateResponse struct {
	// in: body
	// required: true
	Payload *UserApiTokenCreated
}

package model

import (
	"time"

	"github.com/jmoiron/sqlx/types"
)

// UserProps provides mechanism to store ad hoc per user properties
// as a JSON object.
// An example use case is to use the props to store whether UI on-boarding
// is done for a given user.
type UserProps struct {
	// ntnx:ignore
	UserID string `json:"user_id" db:"user_id" validate:"range=0:36,ignore=create"`
	// ntnx:ignore
	TenantID string `json:"tenantId" db:"tenant_id" validate:"range=0:36"`
	// ntnx:ignore
	Version float64 `json:"version,omitempty" db:"version"`
	// ntnx:ignore
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// ntnx:ignore
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	// Properties object in JSON format
	Props types.JSONText `json:"props" db:"props" validate:"range=0:4096"`
}

// swagger:parameters UserPropsGet UserPropsGetV2 UserPropsUpdate UserPropsUpdateV2 UserPropsDelete UserPropsDeleteV2
// in: header
type userPropsAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// UserPropsUpdateParam is UserProps used as API parameter
// swagger:parameters UserPropsUpdate UserPropsUpdateV2
type UserPropsUpdateParam struct {
	// in: body
	// required: true
	Body *UserProps `json:"body"`
}

// Ok
// swagger:response UserPropsGetResponse
type UserPropsGetResponse struct {
	// in: body
	// required: true
	Payload *UserProps
}

package model

import (
	"time"

	"github.com/jmoiron/sqlx/types"
)

// TenantProps provides a mechanism to store ad hoc per tenant properties
// as a JSON object.
// An example use case is to use the properties to store whether cloud management console
// onboarding is performed for a given tenant.
type TenantProps struct {
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

// swagger:parameters TenantPropsGet TenantPropsGetV2 TenantPropsUpdate TenantPropsUpdateV2 TenantPropsDelete TenantPropsDeleteV2
// in: header
type tenantPropsAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// TenantPropsUpdateParam is TenantProps used as an API parameter.
// swagger:parameters TenantPropsUpdate TenantPropsUpdateV2
type TenantPropsUpdateParam struct {
	// in: body
	// required: true
	Body *TenantProps `json:"body"`
}

// Ok
// swagger:response TenantPropsGetResponse
type TenantPropsGetResponse struct {
	// in: body
	// required: true
	Payload *TenantProps
}

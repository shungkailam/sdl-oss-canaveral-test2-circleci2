package model

import (
	"time"
)

// Tenant is the DB object and object model for each tenant.
//
// A tenant represents a customer account.
// A tenant may have multiple edges.
// Every object in DB belonging to a tenant
// will have a tenantId field.
// Tenant object, like every other object
// in DB, will have Id and version fields.
// The Id and version fields are marked as optional
// because they are not required in a create operation.
//
// Use Float b/c convert to map will change int to float64
// swagger:model Tenant
type Tenant struct {
	//
	// Unique ID to identify the tenant, which.
	// can be supplied during create or DB generated.
	// For Nice we will have fixed tenant id such as
	//   tenant-id-waldot
	//   tenant-id-rocket-blue
	//
	// required: true
	ID string `json:"id" db:"id" validate:"range=0:36,ignore=create"`
	//
	// Unique tenant ID returned by my.nutanix.com.
	//
	// required: true
	ExternalID string `json:"externalId,omitempty" db:"external_id" validate:"range=0:36"`
	//
	// Version number of object maintained by DB.
	// Not currently used.
	//
	Version float64 `json:"version,omitempty" db:"version"`
	//
	// Tenant name.
	// For example, WalDot, Rocket Blue, and so on. Up to 200 characters.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=0:200"`
	//
	// Unique token for a tenant.
	// Used in authentication.
	//
	// required: true
	Token string `json:"token" db:"token" validate:"range=0:4096"`

	//
	// Tenant description. Up to 200 characters.
	//
	// required: false
	Description string `json:"description" db:"description" validate:"range=0:200"`

	// required: false
	// profile for this tenant.
	Profile *TenantProfile `json:"profile"`

	// ntnx:ignore
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// ntnx:ignore
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// TenantInfo info about tenant returned by GET tenant call
type TenantInfo struct {
	//
	// Unique ID to identify the tenant
	//
	// required: true
	ID string `json:"id"`
	//
	// Tenant name.
	//
	// required: true
	Name string `json:"name"`

	// required: false
	// profile for this tenant.
	Profile *TenantProfile `json:"profile"`
}

// TenantProfile is the object model for tenant profiles.
//
// swagger:model TenantProfile
type TenantProfile struct {
	// required: false
	// Whether to allow privileged applications in this tenant.
	// Please contact Karbon Platform Services support to turn on this feature.
	Privileged bool `json:"privileged"`
	// required: false
	// Whether to allow ssh access for this tenant.
	// Please contact Karbon Platform Services support to turn on this feature.
	EnableSSH bool `json:"enableSSH"`
	// required: false
	// Whether to allow ssh access from cli.
	// Please contact Karbon Platform Services support to turn on this feature.
	AllowCliSSH bool `json:"allowCliSSH"`
}

// TenantParam is Tenant used as API parameter.
// swagger:parameters TenantCreate TenantUpdate
type TenantParam struct {
	// in: body
	// required: true
	Body *Tenant `json:"body"`
}

// ObjectRequestBaseTenant is used as websocket Tenant message.
// swagger:model ObjectRequestBaseTenant
type ObjectRequestBaseTenant struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Tenant `json:"doc"`
}

// swagger:parameters TenantGet TenantGetByID TenantCreate TenantDelete
// in: header
type tenantAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response TenantGetResponse
type TenantGetResponse struct {
	// in: body
	// required: true
	Payload *TenantInfo
}

// swagger:parameters TenantCreate
// in: body
type TenantCreateParam struct {
	// in: body
	// required: true
	Body *Tenant `json:"body"`
}

type TenantsByID []Tenant

func (a TenantsByID) Len() int           { return len(a) }
func (a TenantsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a TenantsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (t Tenant) ToTenantInfo() TenantInfo {
	return TenantInfo{
		ID:      t.ID,
		Name:    t.Name,
		Profile: t.Profile,
	}
}

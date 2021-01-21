package model

import (
	"cloudservices/common/errcode"
	"strings"
	"time"
)

const (
	MLModelStatusActive   = "Active"
	MLModelStatusInActive = "Inactive"
)

type MLModelVersionStatus struct {
	// Status of the ML model version
	// enum: Active,Inactive
	Status string
	// ML model version number
	Version int
}

type MLModelStatus struct {
	Version   float64                `json:"version,omitempty"`
	TenantID  string                 `json:"tenantId"`
	EdgeID    string                 `json:"edgeId"`
	ModelID   string                 `json:"modelId"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
	Status    []MLModelVersionStatus `json:"modelStatus"`
	ProjectID *string                `json:"projectId"`
}

// Ok
// swagger:response MLModelStatusListResponse
type MLModelStatusListResponse struct {
	// in: body
	// required: true
	Payload *MLModelStatusListPayload
}

// payload for MLModelStatusListResponse
type MLModelStatusListPayload struct {
	// required: true
	PagedListResponsePayload
	// list of ML model statuses
	// required: true
	MLModelStatusList []MLModelStatus `json:"result"`
}

// swagger:parameters MLModelStatusList MLModelStatusGet
// in: header
type mlModelStatusAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ValidateMLModelStatus validate MLModelStatus
func ValidateMLModelStatus(model *MLModelStatus) error {
	if model == nil {
		return errcode.NewBadRequestError("MLModelStatus")
	}
	model.TenantID = strings.TrimSpace(model.TenantID)
	if len(model.TenantID) == 0 {
		return errcode.NewBadRequestError("MLModelStatus:TenantID")
	}
	model.EdgeID = strings.TrimSpace(model.EdgeID)
	if len(model.EdgeID) == 0 {
		return errcode.NewBadRequestError("MLModelStatus:EdgeID")
	}
	model.ModelID = strings.TrimSpace(model.ModelID)
	if len(model.ModelID) == 0 {
		return errcode.NewBadRequestError("MLModelStatus:ModelID")
	}
	return nil
}

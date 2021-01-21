package model

import (
	"cloudservices/common/errcode"
	"strings"
	"time"
)

type PodStatus map[string]interface{}
type PodMetrics map[string]interface{}

// AppStatus describes status for an application on one edge
type AppStatus struct {
	// required: true
	PodStatusList []PodStatus `json:"podStatusList"`
	// required: false
	PodMetricsList []PodMetrics `json:"podMetricsList"`
	// required: false
	ImageList []string `json:"imageList"`
}

// ApplicationStatus - the contents of an ApplicationStatus
// swagger:model ApplicationStatus
type ApplicationStatus struct {
	// ntnx:ignore
	Version float64 `json:"version,omitempty"`
	// required: true
	TenantID string `json:"tenantId" validate:"range=1:36"`
	// required: true
	EdgeID string `json:"edgeId" validate:"range=1:36"`
	// required: true
	ApplicationID string `json:"applicationId" validate:"range=1:36"`
	// required: true
	AppStatus AppStatus `json:"appStatus"`
	// ntnx:ignore
	CreatedAt time.Time `json:"createdAt"`
	// ntnx:ignore
	UpdatedAt time.Time `json:"updatedAt"`
}

// Ok
// swagger:response ApplicationStatusListResponse
type ApplicationStatusListResponse struct {
	// in: body
	// required: true
	Payload *[]ApplicationStatus
}

// Ok
// swagger:response ApplicationStatusListResponseV2
type ApplicationStatusListResponseV2 struct {
	// in: body
	// required: true
	Payload *ApplicationStatusListPayload
}

// payload for ApplicationStatusListResponseV2
type ApplicationStatusListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of application statuses
	// required: true
	ApplicationStatusList []ApplicationStatus `json:"result"`
}

// swagger:parameters ApplicationStatusList ApplicationStatusListV2 ApplicationStatusGet ApplicationStatusGetV2 ApplicationStatusDelete ApplicationStatusDeleteV2 ApplicationStatusCreate ApplicationStatusCreateV2
// in: header
type applicationStatusAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ApplicationStatusCreateParam is Application used as API parameter
// swagger:parameters ApplicationStatusCreate ApplicationStatusCreateV2
type ApplicationStatusCreateParam struct {
	// in: body
	// required: true
	Body *ApplicationStatus `json:"body"`
}

// ValidateApplicationStatus validate ApplicationStatus
func ValidateApplicationStatus(model *ApplicationStatus) error {
	if model == nil {
		return errcode.NewBadRequestError("ApplicationStatus")
	}
	model.TenantID = strings.TrimSpace(model.TenantID)
	if len(model.TenantID) == 0 {
		return errcode.NewBadRequestError("ApplicationStatus:TenantID")
	}
	model.EdgeID = strings.TrimSpace(model.EdgeID)
	if len(model.EdgeID) == 0 {
		return errcode.NewBadRequestError("ApplicationStatus:EdgeID")
	}
	model.ApplicationID = strings.TrimSpace(model.ApplicationID)
	if len(model.ApplicationID) == 0 {
		return errcode.NewBadRequestError("ApplicationStatus:ApplicationID")
	}
	return nil
}

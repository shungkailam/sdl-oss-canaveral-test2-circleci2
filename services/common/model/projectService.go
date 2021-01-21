package model

// swagger:parameters ProjectServiceList ProjectServiceGet ProjectServiceCreate ProjectServiceUpdate ProjectServiceDelete
// in: header
type projectServiceAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ProjectService is model for service within a project.
// Those are similar to application but might be instantiated
// by edge stack on demand when required.

// swagger:model ProjectService
type ProjectService struct {
	// required: true
	BaseModel
	// required: true
	ProjectID string `json:"projectId" db:"project_id"`
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	// required: true
	ServiceManifest string `json:"serviceManifest" db:"serviceManifest"`
}

// ObjectRequestBaseProjectService is used as a websocket ProjectService message
// swagger:model ObjectRequestBaseProjectService
type ObjectRequestBaseProjectService struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ProjectService `json:"doc"`
}

// ProjectServiceCreateParam is ProjectService used as API parameter
// swagger:parameters ProjectServiceCreate
type ProjectServiceCreateParam struct {
	// Describes the edge service creation request
	// in: body
	// required: true
	Body *ProjectService `json:"body"`
}

// ProjectServiceUpdateParam is ProjectService used as API parameter
// swagger:parameters ProjectServiceUpdate
type ProjectServiceUpdateParam struct {
	// in: body
	// required: true
	Body *ProjectService `json:"body"`
}

// Ok
// swagger:response ProjectServiceGetResponse
type ProjectServiceGetResponse struct {
	// in: body
	// required: true
	Payload *ProjectService
}

// Ok
// swagger:response ProjectServiceListResponse
type ProjectServiceListResponse struct {
	// in: body
	// required: true
	Payload *ProjectServiceListPayload
}

// payload for ProjectServiceListResponse
type ProjectServiceListPayload []ProjectService

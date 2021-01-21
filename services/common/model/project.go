package model

import "cloudservices/common/errcode"

const (
	ProjectEdgeSelectorTypeCategory = "Category"
	ProjectEdgeSelectorTypeExplicit = "Explicit"
	ProjectRoleAdmin                = "PROJECT_ADMIN"
	ProjectRoleUser                 = "PROJECT_USER"
)

type ProjectUserInfo struct {
	//
	// User Id to be added to the project
	//
	// required: true
	UserID string `json:"userId" db:"user_id"`
	//
	// Valid values for Role are: PROJECT_ADMIN, PROJECT_USER
	//
	// enum: PROJECT_ADMIN,PROJECT_USER
	// required: true
	Role string `json:"role" db:"user_role"`
}

func ValidateProjectUserInfo(info *ProjectUserInfo) error {
	if info.Role != ProjectRoleAdmin && info.Role != ProjectRoleUser {
		return errcode.NewBadRequestError("ProjectUserInfo")
	}
	return nil
}

type ProjectRole struct {
	// required: true
	ProjectID string `json:"projectId" db:"project_id"`
	// enum: PROJECT_ADMIN,PROJECT_USER
	// required: true
	Role string `json:"role" db:"user_role"`
}

// Project is object model for project
//
// A Project is logical grouping of resouces.
// (Edges, CloudCreds, Users, Data Pipelines, and so on.)
//
// swagger:model Project
type Project struct {
	// required: true
	BaseModel
	//
	// Project name.
	//
	// required: true
	Name string `json:"name" validate:"range=1:200"`
	//
	// Describe the project.
	//
	// required: true
	Description string `json:"description" validate:"range=0:200"`
	//
	// List of cloud profile credential IDs that the project can access.
	//
	// required: true
	CloudCredentialIDs []string `json:"cloudCredentialIds"`
	//
	// List of Docker container registry profile IDs that the project can access.
	//
	// required: true
	DockerProfileIDs []string `json:"dockerProfileIds"`
	//
	// List of users who can access the project.
	//
	// required: true
	Users []ProjectUserInfo `json:"users"`
	//
	// Type of edge selector: Category or Explicit.
	// Specify whether edges belonging to this project are
	// given by edgeIDs (Explicit) or edgeSelectors (Category).
	//
	// enum: Category,Explicit
	// required: true
	EdgeSelectorType string `json:"edgeSelectorType" validate:"range=1:20"`
	//
	// List of edge IDs for edges in this project.
	// Only relevant when edgeSelectorType === 'Explicit'
	//
	EdgeIDs []string `json:"edgeIds"`
	//
	// Edge selectors - CategoryInfo list.
	// Only relevant when edgeSelectorType === 'Category'
	//
	EdgeSelectors []CategoryInfo `json:"edgeSelectors"`

	// Privileged projects can use all Kubernetes resources
	Privileged *bool `json:"privileged"`
}

// ProjectCreateParam is Project used as API parameter
// swagger:parameters ProjectCreate ProjectCreateV2
type ProjectCreateParam struct {
	// Describes the project creation request.
	// in: body
	// required: true
	Doc *Project `json:"doc"`
}

// ProjectUpdateParam is Project used as API parameter
// swagger:parameters ProjectUpdate ProjectUpdateV2 ProjectUpdateV3
type ProjectUpdateParam struct {
	// in: body
	// required: true
	Doc *Project `json:"doc"`
}

// Ok
// swagger:response ProjectGetResponse
type ProjectGetResponse struct {
	// in: body
	// required: true
	Payload *Project
}

// Ok
// swagger:response ProjectListResponse
type ProjectListResponse struct {
	// in: body
	// required: true
	Payload *[]Project
}

// Ok
// swagger:response ProjectListResponseV2
type ProjectListResponseV2 struct {
	// in: body
	// required: true
	Payload *ProjectListPayload
}

// payload for ProjectListResponseV2
type ProjectListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of projects
	// required: true
	ProjectList []Project `json:"result"`
}

// swagger:parameters ProjectList ProjectListV2 ProjectGet ProjectGetV2 ProjectCreate ProjectCreateV2 ProjectUpdate ProjectUpdateV2 ProjectUpdateV3 ProjectDelete ProjectDeleteV2
// in: header
type projectAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseProject is used as websocket Project message
// swagger:model ObjectRequestBaseProject
type ObjectRequestBaseProject struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Project `json:"doc"`
}

func (doc Project) GetProjectID() string {
	return doc.ID
}
func (doc Project) IsPrivileged() bool {
	if doc.Privileged != nil {
		return *doc.Privileged
	}
	return false
}

type ProjectsByID []Project

func (a ProjectsByID) Len() int           { return len(a) }
func (a ProjectsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ProjectsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

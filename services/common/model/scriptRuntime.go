package model

import (
	"cloudservices/common/errcode"
	"strings"
)

type ScriptRuntimeCore struct {
	//
	// Name of the runtime environment.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// Runtime description.
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// Runtime enviroment script language.
	//
	// required: true
	Language string `json:"language" db:"language" validate:"range=1:100"`
	//
	// ntnx:ignore
	//
	// Whether this is a built-in script runtime. Always set this to false for user created script runtime.
	//
	// required: true
	Builtin bool `json:"builtin" db:"builtin"`
	//
	// Docker repository URI of the script runtime.
	//
	DockerRepoURI string `json:"dockerRepoURI" db:"docker_repo_uri" validate:"range=0:200"`
	//
	// DockerProfile ID (Container registry profile) used by this script runtime.
	//
	DockerProfileID string `json:"dockerProfileID" db:"docker_profile_id" validate:"range=0:36"`
	//
	// Dockerfile for the script runtime. Serves as documentation for the script runtime.
	//
	Dockerfile string `json:"dockerfile" db:"dockerfile" validate:"range=0:4096"`
}

// ScriptRuntime is the DB object and object model for script runtime.
//
// A ScriptRuntime is a Docker container runtime for scripts.
// Karbon Platform Services includes several ScriptRuntimes for built-in (and user defined) scripts.
// User can also create custom ScriptRuntimes which may be
// derived from Karbon Platform Services built-in ScriptRuntimes.
//
// swagger:model ScriptRuntime
type ScriptRuntime struct {
	// required: true
	BaseModel
	// required: true
	ScriptRuntimeCore
	//
	// ID of parent project, required for custom (non-built-in) script runtimes.
	//
	// required: false
	ProjectID string `json:"projectId,omitempty" db:"project_id"`
}

// ScriptRuntimeCreateParam is ScriptRuntime used as API parameter
// swagger:parameters ScriptRuntimeCreate
type ScriptRuntimeCreateParam struct {
	// Describes the script runtime creation request.
	// in: body
	// required: true
	Body *ScriptRuntime `json:"body"`
}

// ScriptRuntimeUpdateParam is ScriptRuntime used as API parameter
// swagger:parameters ScriptRuntimeUpdate ScriptRuntimeUpdateV2
type ScriptRuntimeUpdateParam struct {
	// in: body
	// required: true
	Body *ScriptRuntime `json:"body"`
}

// Ok
// swagger:response ScriptRuntimeGetResponse
type ScriptRuntimeGetResponse struct {
	// in: body
	// required: true
	Payload *ScriptRuntime
}

// Ok
// swagger:response ScriptRuntimeListResponse
type ScriptRuntimeListResponse struct {
	// in: body
	// required: true
	Payload *[]ScriptRuntime
}

// Ok
// swagger:response ScriptRuntimeListResponseV2
type ScriptRuntimeListResponseV2 struct {
	// in: body
	// required: true
	Payload *ScriptRuntimeListPayload
}

// payload for ScriptRuntimeListResponseV2
type ScriptRuntimeListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of script runtimes
	// required: true
	ScriptRuntimeList []ScriptRuntime `json:"result"`
}

// swagger:parameters ScriptRuntimeList RuntimeEnvironmentList ScriptRuntimeGet RuntimeEnvironmentGet ScriptRuntimeCreate RuntimeEnvironmentCreate ScriptRuntimeUpdate ScriptRuntimeUpdateV2 RuntimeEnvironmentUpdate ScriptRuntimeDelete RuntimeEnvironmentDelete ProjectGetScriptRuntimes ProjectGetRuntimeEnvironments
// in: header
type scriptRuntimeAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseScriptRuntime is used as websocket ScriptRuntime message
// swagger:model ObjectRequestBaseScriptRuntime
type ObjectRequestBaseScriptRuntime struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ScriptRuntime `json:"doc"`
}

func (doc ScriptRuntime) GetProjectID() string {
	return doc.ProjectID
}

type ScriptRuntimesByID []ScriptRuntime

func (a ScriptRuntimesByID) Len() int           { return len(a) }
func (a ScriptRuntimesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ScriptRuntimesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func ValidateScriptRuntime(model *ScriptRuntime) error {
	if model == nil {
		return errcode.NewBadRequestError("ScriptRuntime")
	}
	model.Name = strings.TrimSpace(model.Name)
	model.Language = strings.TrimSpace(model.Language)
	model.DockerRepoURI = strings.TrimSpace(model.DockerRepoURI)
	model.DockerProfileID = strings.TrimSpace(model.DockerProfileID)
	// allow empty DockerProfileID for public docker registry
	// if model.DockerProfileID == "" {
	// 	return errcode.NewBadRequestError("DockerProfileID")
	// }
	return nil
}

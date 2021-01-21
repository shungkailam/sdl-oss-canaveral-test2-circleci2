package model

import (
	"cloudservices/common/errcode"
	"strings"
)

type ScriptCore struct {
	//
	// Function name.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// Provide a description for your function code/script.
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// Type of function code/script: Transformation or Function.
	// Transformation takes a data stream as input
	// and produces a different data stream as output.
	// Function takes a data stream as input
	// but has no constraint on output.
	//
	// enum: Transformation,Function
	// required: true
	Type string `json:"type" db:"type" validate:"options=Transformation:Function"`
	//
	// Programming language for the function code/script.
	// Supported languages are python and javascript
	//
	// required: true
	Language string `json:"language" db:"language" validate:"range=1:20"`
	//
	// Runtime environment for the function code/script.
	// Choose a runtime based on code language. For example:
	// python, javascript, golang
	//
	// required: true
	Environment string `json:"environment" db:"environment" validate:"range=1:4096"`
	//
	// The source code for the function script.
	//
	// required: true
	Code string `json:"code" db:"code"`

	// For backward-compatibility, the following are not marked as required.

	// ID of the ScriptRuntime to use to run this script
	RuntimeID string `json:"runtimeId,omitempty" db:"runtime_id" validate:"range=0:64"`

	// Docker image tag of the ScriptRuntime to use to run this script.
	// If missing or empty, then backend should treat it as "latest"
	RuntimeTag string `json:"runtimeTag,omitempty" db:"runtime_tag" validate:"range=0:128"`

	//
	// ntnx:ignore
	//
	// Whether this is a built-in runtime
	//
	// This should be required, but is not marked as such due to backward compatibility.
	//
	// required: true
	Builtin bool `json:"builtin" db:"builtin"`

	//
	// ID of parent project, required for custom (non-builtin) scripts.
	//
	// required: false
	ProjectID string `json:"projectId,omitempty" db:"project_id" validate:"range=0:64"`

	// note: we don't keep references of DataStreams that use this script here
	// instead, the references are stored in DataStreams.
	// In UI, we show which DataStreams are using this script -
	// to answer that requires an aggregate search query
}
type ScriptCoreDBO struct {
	Name        string  `json:"name" db:"name"`
	Description string  `json:"description" db:"description"`
	Type        string  `json:"type" db:"type"`
	Language    string  `json:"language" db:"language"`
	Environment string  `json:"environment" db:"environment"`
	Code        string  `json:"code" db:"code"`
	RuntimeID   *string `json:"runtimeId,omitempty" db:"runtime_id"`
	RuntimeTag  *string `json:"runtimeTag,omitempty" db:"runtime_tag"`
	Builtin     *bool   `json:"builtin" db:"builtin"`
	ProjectID   *string `json:"projectId,omitempty" db:"project_id"`
}

// Script is object model for script
//
// Script represent lambdas:
// functions or transformations that can be applied
// to DataStreams.
// Scripts are tenant-wide objects and the same script
// may be run within an edge, across all edges of a tenant
// or on tenant data in the cloud.
//
// swagger:model Script
type Script struct {
	// required: true
	BaseModel
	// required: true
	ScriptCore
	//
	// Array of script parameters.
	// required: true
	Params []ScriptParam `json:"params" db:"params"`
}

// ScriptForceUpdate is used to pass the forceUpdate option to script update function
// This object is for internal use only
type ScriptForceUpdate struct {
	Doc         Script `json:"doc"`
	ForceUpdate bool   `json:"forceUpdate"`
}

// ScriptCreateParam is Script used as API parameter
// swagger:parameters ScriptCreate
type ScriptCreateParam struct {
	// Describes the script creation request
	// in: body
	// required: true
	Body *Script `json:"body"`
}

// ScriptUpdateParam is Script used as API parameter
// swagger:parameters ScriptUpdate ScriptUpdateV2
type ScriptUpdateParam struct {
	// in: body
	// required: true
	Body *Script `json:"body"`
}

// Ok
// swagger:response ScriptGetResponse
type ScriptGetResponse struct {
	// in: body
	// required: true
	Payload *Script
}

// Ok
// swagger:response ScriptListResponse
type ScriptListResponse struct {
	// in: body
	// required: true
	Payload *[]Script
}

// Ok
// swagger:response ScriptListResponseV2
type ScriptListResponseV2 struct {
	// in: body
	// required: true
	Payload *ScriptListPayload
}

// payload for ScriptListResponseV2
type ScriptListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of scripts
	// required: true
	ScriptList []Script `json:"result"`
}

// swagger:parameters ScriptList FunctionList ScriptGet FunctionGet ScriptCreate FunctionCreate ScriptUpdate ScriptUpdateV2 FunctionUpdate ScriptDelete FunctionDelete ProjectGetScripts ProjectGetFunctions
// in: header
type scriptAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseScript is used as websocket Script message
// swagger:model ObjectRequestBaseScript
type ObjectRequestBaseScript struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Script `json:"doc"`
}

func (doc Script) GetProjectID() string {
	return doc.ProjectID
}

type ScriptsByID []Script

func (a ScriptsByID) Len() int           { return len(a) }
func (a ScriptsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ScriptsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func ValidateScript(model *Script) error {
	if model == nil {
		return errcode.NewBadRequestError("Script")
	}
	model.Name = strings.TrimSpace(model.Name)
	model.Type = strings.TrimSpace(model.Type)
	model.Language = strings.TrimSpace(model.Language)
	model.Environment = strings.TrimSpace(model.Environment)
	model.RuntimeID = strings.TrimSpace(model.RuntimeID)
	model.RuntimeTag = strings.TrimSpace(model.RuntimeTag)
	model.ProjectID = strings.TrimSpace(model.ProjectID)
	// for backward compatibility, empty RuntimeID may mean script is using builtin runtime
	// if model.RuntimeID == "" {
	// 	return errcode.NewBadRequestError("RuntimeID")
	// }
	return nil
}

func ScriptsDifferOnlyByNameAndDesc(s1 *Script, s2 *Script) bool {
	if s1.Type != s2.Type ||
		s1.Language != s2.Language ||
		s1.Environment != s2.Environment ||
		s1.Code != s2.Code ||
		s1.RuntimeID != s2.RuntimeID ||
		s1.RuntimeTag != s2.RuntimeTag ||
		s1.Builtin != s2.Builtin ||
		s1.ProjectID != s2.ProjectID ||
		s1.ID != s2.ID ||
		s1.TenantID != s2.TenantID {
		return false
	}
	if len(s1.Params) != len(s2.Params) {
		return false
	}
	for i := range s1.Params {
		if s1.Params[i] != s2.Params[i] {
			return false
		}
	}
	return true
}

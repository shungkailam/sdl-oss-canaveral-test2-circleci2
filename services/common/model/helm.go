package model

import (
	"os"
)

// swagger:parameters HelmTemplate HelmApplicationCreate HelmApplicationUpdate
// in: header
type helmAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// swagger:parameters HelmTemplate
// in: formData
// swagger:file
type HelmTemplateParam struct {
	// required: true
	// swagger:file
	// in: formData
	Chart *os.File `json:"chart"`
	// swagger:file
	// in: formData
	Values *os.File `json:"values,omitempty"`
	// required: true
	// in: formData
	Release string `json:"release"`
	// in: formData
	Namespace string `json:"namespace,omitempty"`
}

type HelmTemplateJSONParam struct {
	// required: true
	// base64 encoded chart tgz data
	Chart string `json:"chart"`
	// base64 encoded values.yaml data
	Values string `json:"values,omitempty"`
	// required: true
	Release   string `json:"release"`
	Namespace string `json:"namespace,omitempty"`
}

type HelmTemplateResponse struct {
	// required: true
	// contains helm template and hook yaml string
	AppManifest string `json:"appManifest"`
	// required: true
	// contains helm Chart.yaml string
	Metadata string `json:"metadata"`
	// contains values.yaml string
	Values string `json:"values,omitempty"`
	// contains helm Custom Resource Definitions string
	CRDs string `json:"crds,omitempty"`
}

// Helm template response contains the rendered yaml
// swagger:response HelmTemplateResponse
type HelmTemplateResponseWrapper struct {
	// in: body
	// required: true
	Payload *HelmTemplateResponse
}

// swagger:parameters HelmApplicationCreate
// in: formData
// swagger:file
type HelmAppCreateParam struct {
	// required: true
	// swagger:file
	// in: formData
	Chart *os.File `json:"chart"`
	// swagger:file
	// in: formData
	Values *os.File `json:"values"`
	// required: true
	// in: formData
	Application string/* type: ApplicationV2 */ `json:"application"`
}

// swagger:parameters HelmApplicationUpdate
// in: formData
// swagger:file
type HelmAppUpdateParam struct {
	// swagger:file
	// in: formData
	Chart *os.File `json:"chart"`
	// swagger:file
	// in: formData
	Values *os.File `json:"values"`
	// required: true
	// in: formData
	Application string/* type: ApplicationV2 */ `json:"application"`
}

type HelmAppJSONParam struct {
	// base64 encoded chart tgz data
	Chart string `json:"chart"`
	// base64 encoded values.yaml data
	Values      string        `json:"values"`
	Application ApplicationV2 `json:"application"`
}

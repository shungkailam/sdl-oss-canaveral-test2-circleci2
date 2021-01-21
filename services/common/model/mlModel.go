package model

import (
	"cloudservices/common/errcode"
	"os"
	"strings"
	"time"
)

const (
	FT_TENSORFLOW_DEFAULT = "TensorFlow 1.13.1"
	FT_OPENVINO_DEFAULT   = "OpenVINO 2019_R2"
	FT_TENSORFLOW_2_1_0   = "TensorFlow 2.1.0"
)

// MLModelMetadata is base object model for the machine learning model.
//
// It serves as the payload when creating a machine learning model.
//
// swagger:model MLModelMetadata
type MLModelMetadata struct {
	// required: true
	BaseModel
	//
	// Name for the machine learning model. Maximum length is 200 characters.
	//
	// required: true
	Name string `json:"name" validate:"range=1:200"`
	//
	// Describe the machine learning model.  Maximum length is 200 characters.
	//
	// required: true
	Description string `json:"description" validate:"range=0:200"`
	//
	// Parent project ID associated with this machine learning model.
	//
	// required: true
	ProjectID string `json:"projectId" validate:"range=0:64"`
	//
	// Machine learning model framework type.
	//
	// required: true
	// enum: TensorFlow 1.13.1,OpenVINO 2019_R2,TensorFlow 2.1.0
	FrameworkType string `json:"frameworkType" validate:"range=0:32"`
}

// MLModel is the object model for the machine learning model.
//
// An MLModel represents a machine learning model.
//
// swagger:model MLModel
type MLModel struct {
	// required: true
	MLModelMetadata
	//
	// Machine learning model versions.
	//
	// required: false
	ModelVersions []MLModelVersion `json:"modelVersions"`
}

// MLModelVersion version of a machine learning model.
type MLModelVersion struct {
	// User entered version of the ML model
	ModelVersion int `json:"modelVersion"`
	// ntnx:ignore
	// AWS S3 generated version of the ML model
	S3Version string `json:"s3Version"`
	// A description of the ML model version
	Description string `json:"description"`
	// Size in bytes of the model version binary
	ModelSizeBytes int64 `json:"modelSizeBytes"`
	// ntnx:ignore
	// timestamp feature supported by DB
	CreatedAt *time.Time `json:"createdAt,omitempty"`
	// ntnx:ignore
	// timestamp feature supported by DB
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// MLModelCreateParam is MLModel machine learning model used as an API parameter.
// swagger:parameters MLModelCreate
type MLModelCreateParam struct {
	// in: body
	// required: true
	Body *MLModelMetadata `json:"body"`
}

// MLModelUpdateParam is MLModel machine learning model used as an API parameter.
// swagger:parameters MLModelUpdate
type MLModelUpdateParam struct {
	// in: body
	// required: true
	Body *MLModelMetadata `json:"body"`
}

// Ok
// swagger:response MLModelGetResponse
type MLModelGetResponse struct {
	// in: body
	// required: true
	Payload *MLModel
}

// Ok
// swagger:response MLModelListResponse
type MLModelListResponse struct {
	// in: body
	// required: true
	Payload *MLModelListResponsePayload
}

// payload for MLModelListResponse
type MLModelListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of ML models
	// required: true
	MLModelList []MLModel `json:"result"`
}

// swagger:parameters MLModelList MLModelGet MLModelCreate MLModelUpdate MLModelDelete ProjectGetMLModels MLModelVersionCreate MLModelVersionUpdate MLModelVersionDelete MLModelVersionURLGet
// in: header
type mlModelAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from the login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// swagger:parameters MLModelVersionCreate
// in: formData
// swagger:file
type MLModelVersionCreateBodyParam struct {
	// required: true
	// swagger:file
	// in: formData
	Payload *os.File
}

// Payload *runtime.File
// Payload io.ReadCloser

type MLModelVersionURLGetResponsePayload struct {
	URL string `json:"url"`
}

// Ok
// swagger:response MLModelVersionURLGetResponse
type MLModelVersionURLGetResponse struct {
	// in: body
	// required: true
	Payload *MLModelVersionURLGetResponsePayload
}

// ObjectRequestBaseMLModel is used as a websocket MLModel message.
// swagger:model ObjectRequestBaseMLModel
type ObjectRequestBaseMLModel struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc MLModel `json:"doc"`
}

type MLModelsByID []MLModel

func (a MLModelsByID) Len() int           { return len(a) }
func (a MLModelsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a MLModelsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func ValidateMLModel(model *MLModelMetadata) error {
	if model == nil {
		return errcode.NewBadRequestError("MLModel")
	}
	model.Name = strings.TrimSpace(model.Name)
	model.ProjectID = strings.TrimSpace(model.ProjectID)
	model.Description = strings.TrimSpace(model.Description)
	model.FrameworkType = strings.TrimSpace(model.FrameworkType)
	if model.ProjectID == "" {
		return errcode.NewBadRequestError("ProjectID")
	}
	if model.Name == "" {
		return errcode.NewBadRequestError("name")
	}

	return nil
}

func ValidateMLModelVersion(model *MLModelVersion) error {
	if model == nil {
		return errcode.NewBadRequestError("MLModelVersion")
	}
	model.Description = strings.TrimSpace(model.Description)
	if model.ModelVersion <= 0 {
		return errcode.NewBadRequestError("ModelVersion")
	}
	return nil
}

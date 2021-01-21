 package model

import (
	"cloudservices/common/errcode"
	"strings"
)

// ServiceDomainProfile is the object model for service domain profiles.
//
// swagger:model ServiceDomainProfile
type ServiceDomainProfile struct {
	// required: false
	// Whether to allow privileged applications in this service domain.
	// IoT edges are not multi-tenant and can allow this setting to
	// be true.
	Privileged bool `json:"privileged"`
	// required: false
	// Whether to allow ssh access to this service domain.
	// Contact Karbon Platform Services support to turn on the ssh feature,
	// then infra admin can use this flag to control ssh access
	// to the individual service domain.
	EnableSSH bool `json:"enableSSH"`
	// required: false
	// Type of ingress controller to use on service domain.
	// So far we support Traefik and nginx.
	// enum: Traefik,NGINX
	IngressType string `json:"ingressType,omitempty"`
	// required: false
	// AI Inferencing service configuration settings.
	AIInferencingService *AIInferencingServiceProfile
}

//AIInferencingServiceProfile has configuration setting for different framework types.
type AIInferencingServiceProfile struct {
	// required: false
	// Enable the AI inferencing service
	Enable bool
	// required: false
	// AI Inferencing service runtime settings.
	Runtime []*AIInferencingRuntime
}

// AIInferencingRuntime defines the framework type and accelerator device.
type AIInferencingRuntime struct {
	// required: false
	// enum: TensorFlow1.13.1,TensorFlow2.1.0
	FrameworkType string
	// required: false
	// enum: CPU,GPU
	AcceleratorDevice string
}

// ServiceDomain is the DB object and object model for service domain
//
// swagger:model ServiceDomain
type ServiceDomain struct {
	// required: true
	BaseModel
	// required: true
	ServiceDomainCore
	//
	// EdgeCluster description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// A list of Category labels for this service domain.
	//
	// required: false
	Labels []CategoryInfo `json:"labels"`
	//
	// ntnx:ignore
	// Determines if the service domain is currently connected to Karbon Platform services management plane.
	//
	Connected bool `json:"connected,omitempty" db:"connected"`
	// required: false
	// Embed profile into service domain.
	Profile *ServiceDomainProfile `json:"profile"`
	// required: false
	// Environment variables for the service domain.
	// String representation of environment JSON object.
	// For example: '{"VAR_1":"VALUE_1","VAR_2","VALUE_2"}'
	Env *string `json:"env"`
}

type ServiceDomainCore struct {
	//
	// Service domain name.
	// Maximum length is limited to 60 characters which must satisfy
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:60"`
	//
	// ntnx:ignore
	// ShortID is the unique ID for the given service domain.
	// This ID must be unique for each service domain, for the given tenant.
	// required: false
	ShortID *string `json:"shortId" db:"short_id"`
	//
	// ntnx:ignore
	// Edge type.
	//
	Type *string `json:"type,omitempty" db:"type"`
	//
	// Virtual IP
	//
	VirtualIP *string `json:"virtualIp, omitempty" db:"virtual_ip"`
}

// ServiceDomainCreateParam is ServiceDomain used as API parameter
// swagger:parameters ServiceDomainCreate
// in: body
type ServiceDomainCreateParam struct {
	// Parameters and values used when creating a service domain
	// in: body
	// required: true
	Body *ServiceDomain `json:"body"`
}

// ServiceDomainUpdateParam is ServiceDomain used as API parameter
// swagger:parameters ServiceDomainUpdate
// in: body
type ServiceDomainUpdateParam struct {
	// in: body
	// required: true
	Body *ServiceDomain `json:"body"`
}

// Ok
// swagger:response ServiceDomainGetResponse
type ServiceDomainGetResponse struct {
	// in: body
	// required: true
	Payload *ServiceDomain
}

// Ok
// swagger:response ServiceDomainGetEffectiveProfileResponse
type ServiceDomainGetEffectiveProfileResponse struct {
	// in: body
	// required: true
	Payload *ServiceDomainProfile
}

// Ok
// swagger:response ServiceDomainListResponse
type ServiceDomainListResponse struct {
	// in: body
	// required: true
	Payload *ServiceDomainListPayload
}

// ServiceDomainListPayload is the payload for ServiceDomainListResponse
type ServiceDomainListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of service domains
	// required: true
	SvcDomainList []ServiceDomain `json:"result"`
}

// swagger:parameters ServiceDomainList ProjectGetServiceDomains ServiceDomainGetNodes ServiceDomainGetNodesInfo ServiceDomainGet ServiceDomainDelete ServiceDomainCreate ServiceDomainUpdate ServiceDomainGetFeatures ServiceDomainGetEffectiveProfile
// in: header
type serviceDomainAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ServiceDomainGetHandlePayload is the payload for get service domain handle call
// token: see crypto.GetEdgeHandleToken
type ServiceDomainGetHandlePayload struct {
	// required: true
	Token string `json:"token"`
	// required: true
	TenantID string `json:"tenantId"`
}

// ServiceDomainGetHandleParam is ServiceDomainGetHandlePayload used as API parameter
// token: see crypto.GetEdgeHandleToken
// swagger:parameters ServiceDomainGetHandle
// in: body
type ServiceDomainGetHandleParam struct {
	// in: body
	// required: true
	Body *ServiceDomainGetHandlePayload
}

// Ok
// swagger:response ServiceDomainGetHandleResponse
type ServiceDomainGetHandleResponse struct {
	// in: body
	// required: true
	Payload *EdgeCert
}

type ServiceDomainsByID []ServiceDomain

func (a ServiceDomainsByID) Len() int           { return len(a) }
func (a ServiceDomainsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ServiceDomainsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// ValidateServiceDomain validates a service domain
func ValidateServiceDomain(model *ServiceDomain) error {
	if model == nil {
		return errcode.NewBadRequestError("Service Domain")
	}
	// Validation copied from edge.go + KubernetesClusterTargetType
	if model.Type != nil {
		if len(*model.Type) == 0 || *model.Type == string(RealTargetType) {
			model.Type = nil
		} else if *model.Type != string(CloudTargetType) && *model.Type != string(KubernetesClusterTargetType) {
			return errcode.NewBadRequestError("Type")
		}
	}

	model.Name = strings.TrimSpace(model.Name)
	model.Description = strings.TrimSpace(model.Description)

	// DNS-1123 standard
	// see: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	matched := dns1123Regexp.MatchString(model.Name)
	if matched == false {
		return errcode.NewMalformedBadRequestExError("Name", "Name can include lowercase alphabets and digits only. Name must start and end with an alphabet or digit. Delimiters allowed are '.' and '-'.")
	}
	svcDomainProfile := model.Profile
	if svcDomainProfile == nil {
		return nil
	}
	aiInferencingSvc := svcDomainProfile.AIInferencingService
	if aiInferencingSvc == nil {
		return nil
	}
	//Only one framework type (currently TensorFlow2.1.0) can have accelerator type has gpu
	foundGPU := false
	for _, runtime := range aiInferencingSvc.Runtime {
		if runtime.AcceleratorDevice != "GPU" && runtime.AcceleratorDevice != "CPU" {
			return errcode.NewMalformedBadRequestError("AcceleratorDevice")
		}
		if runtime.FrameworkType != "TensorFlow1.13.1" && runtime.FrameworkType != "TensorFlow2.1.0" {
			return errcode.NewMalformedBadRequestError("FrameworkType")
		}
		if runtime.AcceleratorDevice == "GPU" {
			if foundGPU {
				return errcode.NewBadRequestExError("AcceleratorDevice", "GPU can be assigned to only one FrameworkType")
			}
			foundGPU = true
			if runtime.FrameworkType != "TensorFlow2.1.0" {
				return errcode.NewBadRequestExError("AcceleratorDevice", "GPU can be assigned to only TensorFlow2.1.0 FrameworkType")
			}
		}

	}

	return nil
}

// UpdateServiceDomainMessage is the placeholder for notifying a service domain on project assignment changes
type UpdateServiceDomainMessage struct {
	Doc      ServiceDomain
	Projects []Project
}

type ServiceDomainIDLabel struct {
	CategoryInfo
	ID string `db:"edge_id"`
}

type ServiceDomainIDLabels struct {
	ID     string
	Labels []CategoryInfo
}

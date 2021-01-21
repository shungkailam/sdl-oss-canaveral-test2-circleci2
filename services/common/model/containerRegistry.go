package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"regexp"
)

type CloudProfileInfo struct {
	// required: true
	CloudCredsID string `json:"cloudCredsID,omitempty"  validate:"range=0:36"`
	//
	// ntnx:ignore
	// required: false
	Email string `json:"email" db:"email,omitempty" validate:"range=0:200"`
}

type ContainerRegistryInfo struct {
	//
	// User name for the container registry profile.
	//
	// required: true
	UserName string `json:"userName" db:"user_name" validate:"range=0:200"`
	//
	// Password for the container registry profile.
	//
	// required: true
	Pwd string `json:"pwd" db:"pwd" validate:"range=0:8192"`
	//
	// Email address for the container registry profile user.
	//
	// required: true
	Email string `json:"email" db:"email" validate:"range=0:200"`
}

// ContainerRegistry is the object model for ContainerRegistry
// swagger:model ContainerRegistry
type ContainerRegistry struct {
	// required: true
	BaseModel
	//
	// Name for the container registry profile.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=0:200"`
	//
	// Description for the container registry profile.
	//
	// required: false
	Description string `json:"description,omitempty" db:"description" validate:"range=0:200"`
	//
	// Container registry profile type.
	//
	// enum: AWS,GCP,Azure,ContainerRegistry
	// required: true
	Type string `json:"type" db:"type" validate:"options=AWS:GCP:Azure:ContainerRegistry"`
	//
	// Provide a server URL to the container registry in the format used by your cloud provider.
	// For example, an Amazon AWS Elastic Container Registry (ECR) URL might be:
	// https://aws_account_id.dkr.ecr.region.amazonaws.com
	//
	// required: true
	Server string `json:"server" db:"server" validate:"range=1:512"`
	//
	// ntnx:ignore
	//
	// Internal Flag - encrypted - for internal migration use
	IFlagEncrypted bool `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
	//
	// Existing cloud profile to use with the container registry profile.
	//
	// Required if Type == AWS || Type == GCP
	CloudCredsID string `json:"cloudCredsID,omitempty" validate="options=AWS:GCP:"`
	//
	// Cloud profile user name for use with the container registry profile.
	//
	// Required for container registry profiles.
	UserName string `json:"userName" db:"user_name" validate:"range=0:200"`
	//
	// Email address to associate with the container registry profile.
	//
	// Required for container registry profiles.
	Email string `json:"email" db:"email" validate:"range=0:200"`
	//
	// Password for the container registry profile.
	//
	// Required for container registry profiles.
	Pwd string `json:"pwd" db:"pwd" validate:"range=0:8192"`
}

// ContainerRegistryV2 ContainerRegistry
// swagger:model ContainerRegistryV2
type ContainerRegistryV2 struct {
	// required: true
	BaseModel
	//
	// Name for the container registry.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=0:200"`
	//
	// Description for the container registry.
	//
	// required: false
	Description string `json:"description,omitempty" db:"description" validate:"range=0:200"`
	//
	// Container registry type.
	//
	// enum: AWS,GCP,Azure,ContainerRegistry
	// required: true
	Type string `json:"type" db:"type" validate:"options=AWS:GCP:Azure:ContainerRegistry"`
	//
	// Container registry server URL.
	// For example, an Amazon AWS Elastic Container Registry (ECR) URL might be:
	// https://aws_account_id.dkr.ecr.region.amazonaws.com
	//
	// required: true
	Server string `json:"server" db:"server" validate:"range=1:512"`
	//
	// Info about the cloud profile.
	// Required if Type == AWS || Type == GCP
	//
	CloudProfileInfo *CloudProfileInfo `json:"CloudProfileInfo,omitempty"`
	//
	// Info about the cloud profile.
	// Required if Type == ContainerRegistry
	//
	ContainerRegistryInfo *ContainerRegistryInfo `json:"ContainerRegistryInfo,omitempty"`
	//
	// ntnx:ignore
	//
	// Internal Flag - encrypted - for internal migration use
	IFlagEncrypted bool `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
}

func (cr ContainerRegistry) ToV2() ContainerRegistryV2 {

	v2Cr := ContainerRegistryV2{
		BaseModel:      cr.BaseModel,
		Name:           cr.Name,
		Description:    cr.Description,
		Type:           cr.Type,
		Server:         cr.Server,
		IFlagEncrypted: cr.IFlagEncrypted,
	}
	if v2Cr.Type == "AWS" || v2Cr.Type == "GCP" {
		v2Cr.CloudProfileInfo = &CloudProfileInfo{}
		v2Cr.CloudProfileInfo.CloudCredsID = cr.CloudCredsID
		v2Cr.CloudProfileInfo.Email = cr.Email
	} else {
		v2Cr.ContainerRegistryInfo = &ContainerRegistryInfo{}
		v2Cr.ContainerRegistryInfo.Pwd = cr.Pwd
		v2Cr.ContainerRegistryInfo.UserName = cr.UserName
		v2Cr.ContainerRegistryInfo.Email = cr.Email
	}

	return v2Cr
}
func (v2Cr ContainerRegistryV2) FromV2() ContainerRegistry {
	cr := ContainerRegistry{
		BaseModel:      v2Cr.BaseModel,
		Name:           v2Cr.Name,
		Description:    v2Cr.Description,
		Type:           v2Cr.Type,
		Server:         v2Cr.Server,
		IFlagEncrypted: v2Cr.IFlagEncrypted,
	}
	if cr.Type == "AWS" || cr.Type == "GCP" {
		cr.CloudCredsID = v2Cr.CloudProfileInfo.CloudCredsID
		cr.Email = v2Cr.CloudProfileInfo.Email
	} else {
		cr.UserName = v2Cr.ContainerRegistryInfo.UserName
		cr.Pwd = v2Cr.ContainerRegistryInfo.Pwd
		cr.Email = v2Cr.ContainerRegistryInfo.Email
	}
	return cr
}

// ContainerRegistryCreateParam is ContainerRegistry used as API parameter
// swagger:parameters ContainerRegistryCreate
type ContainerRegistryCreateParam struct {
	// Describes the container registry profile.
	// in: body
	// required: true
	Body *ContainerRegistry `json:"body"`
}

// ContainerRegistryCreateParam is ContainerRegistryV2 used as API parameter
// swagger:parameters ContainerRegistryCreateV2
type ContainerRegistryCreateParamV2 struct {
	// Describes the container registry profile.
	// in: body
	// required: true
	Body *ContainerRegistryV2 `json:"body"`
}

// ContainerRegistryUpdateParam is ContainerRegistry used as API parameter
// swagger:parameters ContainerRegistryUpdate ContainerRegistryUpdateV2
type ContainerRegistryUpdateParam struct {
	// in: body
	// required: true
	Body *ContainerRegistry `json:"body"`
}

// ContainerRegistryUpdateParam is ContainerRegistry used as API parameter
// swagger:parameters ContainerRegistryUpdate ContainerRegistryUpdateV2
type ContainerRegistryUpdateParamV2 struct {
	// in: body
	// required: true
	Body *ContainerRegistryV2 `json:"body"`
}

// Ok
// swagger:response ContainerRegistryGetResponse
type ContainerRegistryGetResponse struct {
	// in: body
	// required: true
	Payload *ContainerRegistry
}

// Ok
// swagger:response ContainerRegistryGetResponseV2
type ContainerRegistryGetResponseV2 struct {
	// in: body
	// required: true
	Payload *ContainerRegistryV2
}

// Ok
// swagger:response ContainerRegistryListResponse
type ContainerRegistryListResponse struct {
	// in: body
	// required: true
	Payload *[]ContainerRegistry
}

// Ok
// swagger:response ContainerRegistryListResponseV2
type ContainerRegistryListResponseV2 struct {
	// in: body
	// required: true
	Payload *ContainerRegistryListPayload
}

// payload for ContainerRegistryListResponseV2
type ContainerRegistryListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of container registries
	// required: true
	ContainerRegistryListV2 []ContainerRegistryV2 `json:"result"`
}

// swagger:parameters ContainerRegistryList ContainerRegistryListV2 ContainerRegistryGet ContainerRegistryGetV2 ContainerRegistryCreate ContainerRegistryCreateV2 ContainerRegistryUpdate ContainerRegistryUpdateV2 ContainerRegistryDelete ContainerRegistryDeleteV2 ProjectGetContainerRegistries ProjectGetContainerRegistriesV2
// in: header
type ContainerRegistryAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// // ObjectRequestBaseContainerRegistry is used as websocket ContainerRegistry message
// // swagger:model ObjectRequestBaseContainerRegistry
type ObjectRequestBaseContainerRegistry struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ContainerRegistry `json:"doc"`
}

type ContainerRegistriesByID []ContainerRegistry

func (a ContainerRegistriesByID) Len() int           { return len(a) }
func (a ContainerRegistriesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ContainerRegistriesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (registries ContainerRegistriesByID) ToV2() []ContainerRegistryV2 {
	v2Cr := []ContainerRegistryV2{}
	for _, cr := range registries {
		v2Cr = append(v2Cr, cr.ToV2())
	}
	return v2Cr
}

type ContainerRegistriesByIDV2 []ContainerRegistryV2

func (a ContainerRegistriesByIDV2) Len() int           { return len(a) }
func (a ContainerRegistriesByIDV2) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ContainerRegistriesByIDV2) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (v2Cr ContainerRegistriesByIDV2) FromV2() []ContainerRegistry {
	crs := []ContainerRegistry{}
	for _, cr := range v2Cr {
		crs = append(crs, cr.FromV2())
	}
	return crs
}

func ValidateContainerRegistry(profile ContainerRegistry) error {

	// Checking if the name contains only letters and alphabets at the start and it can contain . - elsewhere
	matched, _ := regexp.MatchString("^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$", profile.Name)
	if matched == false {
		return errcode.NewBadRequestError("Name")
	}
	if profile.Type == "" {
		return errcode.NewBadRequestError("Type")
	}
	if profile.Type == "ContainerRegistry" {
		if profile.Name == "" {
			return errcode.NewBadRequestError("Name")
		}
		if profile.Server == "" {
			return errcode.NewBadRequestError("Server")
		}
		if profile.UserName == "" {
			return errcode.NewBadRequestError("UserName")
		}
		if profile.Pwd == "" {
			return errcode.NewBadRequestError("Pwd")
		}
		if profile.Email == "" {
			return errcode.NewBadRequestError("Email")
		}
		if len(profile.CloudCredsID) != 0 {
			return errcode.NewMalformedBadRequestError("CloudCredsID")
		}
	} else {
		if profile.Name == "" {
			return errcode.NewBadRequestError("Name")
		}
		if profile.Server == "" {
			return errcode.NewBadRequestError("Server")
		}
		if len(profile.CloudCredsID) == 0 {
			return errcode.NewBadRequestError("CloudCredsID")
		}
	}
	return nil
}

func (cr *ContainerRegistry) MaskObject() {
	cr.Pwd = base.MaskString(cr.Pwd, "*", 0, 4)
}

func MaskContainerRegistries(crs []ContainerRegistry) {
	for i := 0; i < len(crs); i++ {
		(&crs[i]).MaskObject()
	}
}

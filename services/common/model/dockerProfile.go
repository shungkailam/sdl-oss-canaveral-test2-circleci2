package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"regexp"
)

// DockerProfile is the object model for DockerProfile
// swagger:model DockerProfile
type DockerProfile struct {
	// required: true
	BaseModel
	//
	// Name for the DockerProfile.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// Description for the DockerProfile.
	//
	// required: false
	Description string `json:"description,omitempty" db:"description" validate:"range=0:200"`
	//
	// The CloudCreds to import Docker Profile from
	//
	// required: false
	CloudCredsID string `json:"cloudCredsID,omitempty" validate:"range=0:36"`
	//
	// DockerProfile  type.
	//
	// enum: AWS,GCP,Azure,ContainerRegistry
	// required: true
	Type string `json:"type" db:"type" validate:"options=AWS:GCP:Azure:ContainerRegistry"`
	//
	// DockerProfile  server.
	//
	// required: true
	Server string `json:"server" db:"server" validate:"range=0:512"`
	//
	// DockerProfile  user.
	//
	// required: false
	UserName string `json:"userName" db:"user_name" validate:"range=0:200"`
	//
	// DockerProfile  email.
	//
	// required: false
	Email string `json:"email" db:"email" validate:"range=0:200"`
	//
	// DockerProfile  Password.
	//
	// required: false
	Pwd string `json:"pwd" db:"pwd" validate:"range=0:8192"`
	//
	// The Credentials of the DockerProfile.
	//
	// required: false
	Credentials string `json:"credentials,omitempty" db:"credentials" validate:"range=0:4096"`
	//
	// ntnx:ignore
	//
	// Internal Flag - encrypted - for internal migration use
	//
	// required: false
	IFlagEncrypted bool `json:"iflagEncrypted,omitempty" db:"iflag_encrypted"`
}

// DockerProfileCreateParam is DockerProfile used as API parameter
// swagger:parameters DockerProfileCreate
type DockerProfileCreateParam struct {
	// This is a DockerProfile creation request description
	// in: body
	// required: true
	Body *DockerProfile `json:"body"`
}

// DockerProfileUpdateParam is DockerProfile used as API parameter
// swagger:parameters DockerProfileUpdate DockerProfileUpdateV2
type DockerProfileUpdateParam struct {
	// in: body
	// required: true
	Body *DockerProfile `json:"body"`
}

// Ok
// swagger:response DockerProfileGetResponse
type DockerProfileGetResponse struct {
	// in: body
	// required: true
	Payload *DockerProfile
}

// Ok
// swagger:response DockerProfileListResponse
type DockerProfileListResponse struct {
	// in: body
	// required: true
	Payload *[]DockerProfile
}

// Ok
// swagger:response DockerProfileListResponseV2
type DockerProfileListResponseV2 struct {
	// in: body
	// required: true
	Payload *DockerProfileListPayload
}

// payload for DockerProfileListResponseV2
type DockerProfileListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of docker profiles
	// required: true
	DockerProfileList []DockerProfile `json:"result"`
}

// swagger:parameters DockerProfileList DockerProfileListV2 DockerProfileGet DockerProfileCreate DockerProfileUpdate DockerProfileUpdateV2 DockerProfileDelete ProjectGetDockerProfiles ProjectGetDockerProfilesV2
// in: header
type DockerProfileAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseDockerProfile is used as websocket DockerProfile message
// swagger:model ObjectRequestBaseDockerProfile
type ObjectRequestBaseDockerProfile struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc DockerProfile `json:"doc"`
}

type DockerProfilesByID []DockerProfile

func (a DockerProfilesByID) Len() int           { return len(a) }
func (a DockerProfilesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DockerProfilesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func ValidateDockerProfile(profile DockerProfile) error {

	// Checking if the name contains only letters and alphabets at the start and it can contain . - elsewhere
	matched, _ := regexp.MatchString("^[a-z0-9]([a-z0-9.-]*[a-z0-9])?$", profile.Name)
	if matched == false {
		return errcode.NewMalformedBadRequestError("Name")
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

func (cr *DockerProfile) MaskObject() {
	cr.Pwd = base.MaskString(cr.Pwd, "*", 0, 4)
}

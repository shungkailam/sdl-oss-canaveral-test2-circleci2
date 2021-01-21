package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"fmt"
	"strings"

	"regexp"
)

// User is object model for user
//
// User
// User of Sherlock system.
//
// swagger:model User
type User struct {
	// required: true
	BaseModel
	//
	// Email of user
	//
	// required: true
	Email string `json:"email" db:"email" validate:"email,range=1:200"`
	//
	// User name
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=0:200"`
	//
	// SHA-256 hash of user password
	//
	// required: true
	Password string `json:"password" db:"password" validate:"range=0:200"`
	//
	// User role.
	//
	// enum: INFRA_ADMIN,USER
	// required: false
	Role string `json:"role,omitempty" db:"role" validate:"range=0:36"`
}

// UserCreateParam is User used as API parameter
// swagger:parameters UserCreate UserCreateV2
type UserCreateParam struct {
	// This is a user creation request description
	// in: body
	// required: true
	Body *User `json:"body"`
}

// UserUpdateParam is User used as API parameter
// swagger:parameters UserUpdate UserUpdateV2 UserUpdateV3
type UserUpdateParam struct {
	// in: body
	// required: true
	Body *User `json:"body"`
}

// Ok
// swagger:response UserGetResponse
type UserGetResponse struct {
	// in: body
	// required: true
	Payload *User
}

// Ok
// swagger:response UserListResponse
type UserListResponse struct {
	// in: body
	// required: true
	Payload *[]User
}

// Ok
// swagger:response UserListResponseV2
type UserListResponseV2 struct {
	// in: body
	// required: true
	Payload *UserListPayload
}

// payload for UserListResponseV2
type UserListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of users
	// required: true
	UserList []User `json:"result"`
}

// Credential is used for login payload
// swagger:model Credential
type Credential struct {
	// required: true
	Email string `json:"email"`
	// required: true
	Password string `json:"password"`
}

// OAuthCodes is used for OAuth login and token refresh
// swagger:model OAuthCodes
type OAuthCodes struct {
	// required: false
	Code string `json:"code"`
	// required: false
	RefreshToken string `json:"refreshToken"`
}

// OAuthTokenParam is used to get/refresh the session token
// swagger:parameters OAuthTokenCall OAuthTokenCallV2
type OAuthTokenParam struct {
	// in: body
	// required: true
	Request *OAuthCodes
}

// LoginParam is Credential used as API parameter
// swagger:parameters LoginCall LoginCallV2
type LoginParam struct {
	// This is a login credential
	// in: body
	// required: true
	Request *Credential
}

// swagger:parameters UserList UserListV2 UserGet UserGetV2 UserCreate UserCreateV2 UserUpdate UserUpdateV2 UserUpdateV3 UserDelete UserDeleteV2 ProjectGetUsers ProjectGetUsersV2 IsEmailAvailable
// in: header
type userAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseUser is used as websocket User message
// swagger:model ObjectRequestBaseUser
type ObjectRequestBaseUser struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc User `json:"doc"`
}

// EmailAvailability is used as response to isemailavailable query
// swagger:model EmailAvailability
type EmailAvailability struct {
	// required: true
	Email string `json:"email"`
	// required: true
	Available bool `json:"available"`
}

// Ok
// swagger:response IsEmailAvailableResponse
type IsEmailAvailableResponse struct {
	// in: body
	// required: true
	Payload *EmailAvailability
}

//
// IsEmailAvailableQueryParam carries the email to query
// swagger:parameters IsEmailAvailable
// in: query
type IsEmailAvailableQueryParam struct {
	// Email to query for availability.
	// in: query
	// required: true
	Email string `json:"email"`
}

type UsersByID []User

func (a UsersByID) Len() int           { return len(a) }
func (a UsersByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a UsersByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

var reaz = regexp.MustCompile("[a-z]+")
var reAZ = regexp.MustCompile("[A-Z]+")
var reDigit = regexp.MustCompile("[0-9]+")
var reSpecial = regexp.MustCompile("[!@#$%^&*~_/?.,;<>(){}[\\]\\\\=+-]+")
var rePwds = []*regexp.Regexp{reaz, reAZ, reDigit, reSpecial}

const minPasswordLength = 8

func ValidateUser(actorTenantID string, model *User) error {
	if model == nil {
		return errcode.NewBadRequestError("User")
	}
	model.Email = strings.TrimSpace(model.Email)
	model.Name = strings.TrimSpace(model.Name)
	model.Password = strings.TrimSpace(model.Password)
	model.Role = strings.TrimSpace(model.Role)

	if err := ValidateUserRole(actorTenantID, model.Role); err != nil {
		return err
	}

	if actorTenantID != base.MachineTenantID && false == base.IsDemoTenant(model.TenantID) {
		err := ValidatePassword(model.Password)
		if err != nil {
			return err
		}
	}
	return nil
}

func ValidatePassword(password string) error {
	// validate password
	if len(password) < minPasswordLength {
		return errcode.NewBadRequestExError("User", fmt.Sprintf("Bad user password, password length must be at least %d", minPasswordLength))
	}
	notMatchCount := 0
	for _, re := range rePwds {
		if false == re.MatchString(password) {
			notMatchCount++
			if notMatchCount > 0 {
				return errcode.NewBadRequestExError("User", fmt.Sprintf("Bad user password, must have char from all of the following groups: lower case, upper case, digit, special"))
			}
		}
	}
	return nil
}

func ValidateUserRole(actorTenantID, role string) error {
	if role == "INFRA_ADMIN" || role == "USER" {
		return nil
	}
	if actorTenantID == base.MachineTenantID {
		// Allow creation of these hidden roles if the actor is a machine tenant
		if role == "OPERATOR_TENANT" || role == "OPERATOR" {
			return nil
		}
	}
	return errcode.NewBadRequestExError("User", fmt.Sprintf("Bad user role, must be INFRA_ADMIN or USER."))
}

func (user *User) MaskObject() {
	user.Password = base.MaskString(user.Password, "*", 0, 4)
}

func MaskUsers(users []User) {
	for i := 0; i < len(users); i++ {
		(&users[i]).MaskObject()
	}
}

// GetUserSpecialRole returns the special role for the user
func GetUserSpecialRole(user *User) string {
	specialRole := "none"
	if user != nil && user.ID != "" {
		if user.Role == "INFRA_ADMIN" {
			specialRole = "admin"
		} else if user.Role == "OPERATOR" {
			specialRole = "operator"
		} else if user.Role == "OPERATOR_TENANT" {
			specialRole = "operator_tenant"
		}
	}
	return specialRole
}

// HasMissingFields checks whether user object has some fields missing.
// This is used to support partial user update where missing fields
// is replaced by corresponding values from the current user object.
func (user *User) HasMissingFields() bool {
	return user.Email == "" || user.Password == "" || user.Name == "" || user.Role == ""
}

// FillInMissingFields fills in missing fields of user from fromUser.
func (user *User) FillInMissingFields(fromUser *User) {
	if user.Email == "" {
		user.Email = fromUser.Email
	}
	if user.Password == "" {
		user.Password = fromUser.Password
	}
	if user.Name == "" {
		user.Name = fromUser.Name
	}
	if user.Role == "" {
		user.Role = fromUser.Role
	}
}

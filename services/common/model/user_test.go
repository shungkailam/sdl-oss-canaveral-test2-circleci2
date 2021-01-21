package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type userDBO struct {
	model.BaseModel
	Email    string  `json:"email" db:"email"`
	Name     string  `json:"name" db:"name"`
	Password string  `json:"password" db:"password"`
	Role     *string `json:"role,omitempty" db:"role"`
}

// TestUser will test User struct
func TestUser(t *testing.T) {
	now := timeNow(t)
	role := "INFRA_ADMIN"
	users := []model.User{
		{
			BaseModel: model.BaseModel{
				ID:        "user-id",
				Version:   0,
				TenantID:  "tenant-id-waldot",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     "user-name",
			Email:    "user1@example.com",
			Password: "password1",
		},
		{
			BaseModel: model.BaseModel{
				ID:        "user-id2",
				Version:   2,
				TenantID:  "tenant-id-waldot",
				CreatedAt: now,
				UpdatedAt: now,
			},

			Name:     "user-name2",
			Email:    "user2@example.com",
			Password: "password2",
			Role:     role,
		},
	}
	usersDBO := []userDBO{
		{
			BaseModel: model.BaseModel{
				ID:        "user-id",
				Version:   0,
				TenantID:  "tenant-id-waldot",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     "user-name",
			Email:    "user1@example.com",
			Password: "password1",
			Role:     nil,
		},
		{
			BaseModel: model.BaseModel{
				ID:        "user-id2",
				Version:   2,
				TenantID:  "tenant-id-waldot",
				CreatedAt: now,
				UpdatedAt: now,
			},

			Name:     "user-name2",
			Email:    "user2@example.com",
			Password: "password2",
			Role:     &role,
		},
	}

	userStrings := []string{
		`{"id":"user-id","tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","email":"user1@example.com","name":"user-name","password":"password1"}`,
		`{"id":"user-id2","version":2,"tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","email":"user2@example.com","name":"user-name2","password":"password2","role":"INFRA_ADMIN"}`,
	}

	var version float64 = 2
	userMaps := []map[string]interface{}{
		// no version here since omitempty is set
		{
			"id":        "user-id",
			"tenantId":  "tenant-id-waldot",
			"name":      "user-name",
			"email":     "user1@example.com",
			"createdAt": NOW,
			"updatedAt": NOW,
			"password":  "password1",
		},
		{
			"id":        "user-id2",
			"version":   version,
			"tenantId":  "tenant-id-waldot",
			"name":      "user-name2",
			"email":     "user2@example.com",
			"createdAt": NOW,
			"updatedAt": NOW,
			"password":  "password2",
			"role":      "INFRA_ADMIN",
		},
	}
	for i, user := range users {
		userData, err := json.Marshal(user)
		require.NoError(t, err, "failed to marshal user")
		userDBOData, err := json.Marshal(usersDBO[i])
		require.NoError(t, err, "failed to marshal userDBO")
		if userStrings[i] != string(userData) {
			t.Fatal("user json string mismatch", string(userData))
		}
		if userStrings[i] != string(userDBOData) {
			t.Fatal("userDBO json string mismatch", string(userDBOData))
		}
		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(userData, &m)
		require.NoError(t, err, "failed to unmarshal user to map", i)
		if !reflect.DeepEqual(m, userMaps[i]) {
			t.Fatal("user map mismatch", i)
		}
	}
}

func validateUser(t *testing.T, user *model.User, valid bool) {
	err := model.ValidateUser("some-tenant", user)
	if valid {
		require.NoErrorf(t, err, "expect user %+v to be valid, but validation failed with error: %s", *user)
	} else {
		require.Errorf(t, err, "expect user %+v to be invalid", *user)
	}
}

func TestUserValidation(t *testing.T) {
	// no user validation for demo tenants
	demoTenantIDs := []string{"tenant-id-waldot", "tenant-id-numart-stores",
		"tenant-id-smart-retail", "tid-demo-foo"}
	for _, tenantID := range demoTenantIDs {
		validateUser(t, &model.User{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Role: "INFRA_ADMIN",
		}, true)
	}
	tenantID := "foo"
	// validation must fail for bad passwords
	badPasswords := []string{"", "aA0$", "abcdefg", "12345678", "abcd5678", "abcd$%#^", "abCD5678", "0bcd$%#^"}
	for _, pwd := range badPasswords {
		validateUser(t, &model.User{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Password: pwd,
			Role:     "USER",
		}, false)
	}
	// validation must succeed for good passwords
	goodPasswords := []string{"Seaf00d$", "G00dP$ddd", "abCD5678!", "0Bcd$%#^"}
	for _, pwd := range goodPasswords {
		validateUser(t, &model.User{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Password: pwd,
			Role:     "USER",
		}, true)
	}

	goodRoles := []string{"INFRA_ADMIN", "USER", " INFRA_ADMIN  ", " USER"}
	for _, role := range goodRoles {
		validateUser(t, &model.User{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Password: "G00dP$ddd",
			Role:     role,
		}, true)
	}

	badRoles := []string{"infra_admin", "user", "ADMIN", "admin", "user", "admin user"}
	for _, role := range badRoles {
		validateUser(t, &model.User{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Password: "G00dP$ddd",
			Role:     role,
		}, false)
	}

}

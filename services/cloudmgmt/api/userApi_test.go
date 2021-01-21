package api_test

import (
	"bytes"
	accountapi "cloudservices/account/api"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var userName = "John Doe"
var userNameUpdated = "John Doe Updated"
var userPassword = "P@ssw0rd"

func createUser(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.User {
	return createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")
}

func createUserWithRole(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, role string) model.User {

	var userEmail = base.GetUUID() + "@test.com"

	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	// User object, leave ID blank and let create generate it
	user := model.User{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		Name:     userName,
		Email:    userEmail,
		Password: userPassword,
		Role:     role,
	}
	t.Log("Creating user", user)
	// create user
	resp, err := dbAPI.CreateUser(ctx, &user, nil)
	require.NoError(t, err)
	t.Logf("create user successful, %s", resp)

	user.ID = resp.(model.CreateDocumentResponse).ID
	return user
}

func TestUser(t *testing.T) {
	t.Parallel()
	t.Log("running TestUser test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	tenantID := doc.ID
	user := createUser(t, dbAPI, tenantID)
	userId := user.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})
	machineCtx, _, _ := makeContext(base.MachineTenantID, []string{projectID})

	// // Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteUser", func(t *testing.T) {
		t.Log("running Create/Get/DeleteUser test")

		user.Name = userNameUpdated

		user.Role = "ADMIN"
		upResp, err := dbAPI.UpdateUser(ctx1, &user, nil)
		require.Error(t, err, "update user with bad role must fail")

		user.Role = "INFRA_ADMIN"
		upResp, err = dbAPI.UpdateUser(ctx1, &user, nil)
		require.NoError(t, err)

		// fetch user to get password hash
		user, err = dbAPI.GetUser(ctx1, userId)
		require.NoError(t, err)

		// update with empty password will not change password
		passB4Update := user.Password
		user.Password = ""
		upResp, err = dbAPI.UpdateUser(ctx1, &user, nil)
		require.NoError(t, err)
		user, err = dbAPI.GetUser(ctx1, userId)
		require.NoError(t, err)
		t.Logf("get user successful, %+v", user)
		if user.Password != passB4Update {
			t.Fatalf("Mismatched password. Expected: %s, found %s", passB4Update, user.Password)
		}

		// update with current password hash will change password
		upResp, err = dbAPI.UpdateUser(ctx1, &user, nil)
		require.NoError(t, err)
		user, err = dbAPI.GetUser(ctx1, userId)
		require.NoError(t, err)
		t.Logf("get user successful, %+v", user)
		if user.Password == passB4Update {
			t.Fatalf("Expect password to change, but got the same password %s", user.Password)
		}

		user.Password = "simple"
		upResp, err = dbAPI.UpdateUser(ctx1, &user, nil)
		require.Error(t, err, "Simple password must not be accepted!")

		passB4Update = user.Password
		upResp, err = dbAPI.UpdateUser(machineCtx, &user, nil)
		require.NoError(t, err)

		// get user
		user, err = dbAPI.GetUser(ctx1, userId)
		require.NoError(t, err)
		t.Logf("get user successful, %+v", user)

		if user.Password != passB4Update {
			t.Fatalf("Mismatched password. Expected: %s, found %s", passB4Update, user.Password)
		}

		user.Password = base.GenerateStrongPassword()
		upResp, err = dbAPI.UpdateUser(ctx1, &user, nil)
		require.NoError(t, err)
		t.Logf("update user successful, %+v", upResp)

		// get user
		userEmail := user.Email
		user, err = dbAPI.GetUser(ctx1, userId)
		require.NoError(t, err)
		t.Logf("get user successful, %+v", user)

		if user.ID != userId || user.Name != userNameUpdated || user.Email != userEmail {
			t.Fatal("user data mismatch")
		}

		// get all users
		users, err := dbAPI.SelectAllUsers(ctx1)
		require.NoError(t, err)
		if len(users) != 1 {
			t.Fatalf("Expected user count to be 1, got %d", len(users))
		}
		users, err = dbAPI.SelectAllUsers(ctx2)
		require.NoError(t, err)
		if len(users) != 0 {
			t.Fatalf("Expected user 2 count to be 0, got %d", len(users))
		}
		users, err = dbAPI.SelectAllUsers(ctx3)
		require.NoError(t, err)
		if len(users) != 1 {
			t.Fatalf("Expected user 3 count to be 1, got %d", len(users))
		}
		// select all vs select all W
		var w bytes.Buffer
		users1, err := dbAPI.SelectAllUsers(ctx1)
		require.NoError(t, err)
		users2 := &[]model.User{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllUsersW, users2, &w)
		require.NoError(t, err)
		sort.Sort(model.UsersByID(users1))
		sort.Sort(model.UsersByID(*users2))
		// must sync password before comparison, as SelectAllUsersW will mask user password in response
		for i := range *users2 {
			(*users2)[i].Password = users1[i].Password
		}
		if !reflect.DeepEqual(&users1, users2) {
			t.Fatalf("expect select users and select users w results to be equal %+v vs %+v", users1, *users2)
		}

		// get all users for project
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		users, err = dbAPI.SelectAllUsersForProject(newCtx, projectID)
		require.Error(t, err, "expected auth 1 select all users for project to fail")
		users, err = dbAPI.SelectAllUsersForProject(ctx2, projectID)
		require.Error(t, err, "expected auth 2 select all users for project to fail")
		users, err = dbAPI.SelectAllUsersForProject(ctx3, projectID)
		require.NoError(t, err)
		if len(users) != 1 {
			t.Fatalf("Expected auth 3 select all users for project count to be 1, got %d", len(users))
		}

		// get user by email
		user, err = dbAPI.GetUserByEmail(ctx1, userEmail)
		require.NoError(t, err)
		t.Logf("get user by email successful, %+v", user)

		if user.ID != userId || user.Name != userNameUpdated || user.Email != userEmail {
			t.Fatal("user data mismatch")
		}

		var ww bytes.Buffer
		u := fmt.Sprintf("http://example.com/foo?email=%s", url.QueryEscape(userEmail))
		req, err := apitesthelper.NewHTTPRequest("GET", u, nil)
		require.NoError(t, err)
		err = dbAPI.IsEmailAvailableW(ctx1, &ww, req)
		require.NoError(t, err)
		emailAvailability := model.EmailAvailability{}
		err = json.NewDecoder(&ww).Decode(&emailAvailability)
		require.NoError(t, err)
		if emailAvailability.Available || emailAvailability.Email != userEmail {
			t.Fatal("expect email to match and not be available")
		}

		userEmail2 := base.GetUUID() + "foo@bar.com"
		u = fmt.Sprintf("http://example.com/foo?email=%s", url.QueryEscape(userEmail2))
		req, err = apitesthelper.NewHTTPRequest("GET", u, nil)
		require.NoError(t, err)
		err = dbAPI.IsEmailAvailableW(ctx1, &ww, req)
		require.NoError(t, err)
		emailAvailability = model.EmailAvailability{}
		err = json.NewDecoder(&ww).Decode(&emailAvailability)
		require.NoError(t, err)
		if !emailAvailability.Available || emailAvailability.Email != userEmail2 {
			t.Fatal("expect email to match and be available")
		}

		// delete project
		delResp, err := dbAPI.DeleteProject(ctx1, projectID, nil)
		require.NoError(t, err)
		t.Logf("delete project successful, %v", delResp)

		// delete user
		delResp, err = dbAPI.DeleteUser(ctx1, userId, nil)
		require.NoError(t, err)
		t.Logf("delete user successful, %v", delResp)

	})

	// select all users
	t.Run("SelectAllUsers", func(t *testing.T) {
		t.Log("running SelectAllUsers test")
		users, err := dbAPI.SelectAllUsers(ctx1)
		require.NoError(t, err)
		for _, user := range users {
			testForMarshallability(t, user)
		}
	})

	// get all user projects
	t.Run("GetAllUserProjects", func(t *testing.T) {
		t.Log("running GetAllUserProjects test")
		projectRoles, err := dbAPI.GetUserProjectRoles(ctx1, "0635523e-7827-4829-b27a-01dd9f7788d7")
		require.NoError(t, err)
		for _, projectRole := range projectRoles {
			t.Logf("project id: %s, role: %s", projectRole.ProjectID, projectRole.Role)
		}
	})

	t.Run("UserConversion", func(t *testing.T) {
		t.Log("running UserConversion test")
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
		for _, app := range users {
			appDBO := accountapi.UserDBO{}
			app2 := model.User{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := model.User{
			BaseModel: model.BaseModel{
				ID:        id,
				Version:   0,
				TenantID:  "tenant-id-waldot",
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:     "user-name-" + funk.RandomString(10),
			Email:    base.GetUUID() + "@example.com",
			Password: userPassword,
			Role:     "INFRA_ADMIN",
		}
		return dbAPI.CreateUser(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetUser(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteUser(ctx1, id, nil)
	}))
}

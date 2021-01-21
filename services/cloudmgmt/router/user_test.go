package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	USERS_PATH     = "/v1/users"
	USERS_PATH_NEW = "/v1.0/users"
)

// create user
func createUser(netClient *http.Client, user *model.User, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, USERS_PATH, *user, token)
	if err == nil {
		user.ID = resp.ID
	}
	return resp, reqID, err
}

// update user
func updateUser(netClient *http.Client, userID string, user model.User, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", USERS_PATH, userID), user, token)
}

// get users
func getUsers(netClient *http.Client, token string) ([]model.User, error) {
	users := []model.User{}
	err := doGet(netClient, USERS_PATH, token, &users)
	return users, err
}
func getUsersNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.UserListPayload, error) {
	response := model.UserListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", USERS_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}
func getUsersForProject(netClient *http.Client, projectID string, token string) ([]model.User, error) {
	users := []model.User{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/users", token, &users)
	return users, err
}

// delete user
func deleteUser(netClient *http.Client, userID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, USERS_PATH, userID, token)
}

// get user by id
func getUserByID(netClient *http.Client, userID string, token string) (model.User, error) {
	user := model.User{}
	err := doGet(netClient, USERS_PATH+"/"+userID, token, &user)
	return user, err
}

func TestUser(t *testing.T) {
	t.Parallel()
	t.Log("running TestUser test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user2 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test User", func(t *testing.T) {
		token := loginUser(t, netClient, user)
		users, err := getUsers(netClient, token)
		require.NoError(t, err)
		if len(users) != 2 {
			t.Fatalf("expected users count 2, got %d", len(users))
		}

		// create project
		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID, user2.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)

		users2, err := getUsersForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if len(users2) != 2 {
			t.Fatalf("expected users2 count 2, got %d", len(users2))
		}
		sort.Sort(model.UsersByID(users))
		sort.Sort(model.UsersByID(users2))
		if !reflect.DeepEqual(users2, users) {
			t.Fatalf("expect user equal, but %+v != %+v", users2, users)
		}

		token2 := loginUser(t, netClient, user2)
		// try update user to infra admin
		user2.Role = "INFRA_ADMIN"
		_, _, err = updateUser(netClient, user2.ID, user2, token2)
		require.Error(t, err, "expect update normal user to infra admin by self to fail")

		// create user should not require tenantID
		user3Name := base.GetUUID()
		user3 := model.User{
			Email:    fmt.Sprintf("%s@example.com", user3Name),
			Name:     user3Name,
			Password: "P@ssw0rd",
			Role:     "INFRA_ADMIN",
		}
		_, _, err = createUser(netClient, &user3, token)
		require.NoError(t, err)
		// update user should not require tenantID
		user3ID := user3.ID
		user3.Role = "USER"
		user3.ID = ""
		user3.TenantID = ""
		ur, _, err := updateUser(netClient, user3ID, user3, token)
		require.NoError(t, err)
		if ur.ID != user3ID {
			t.Fatal("update user id mismatch")
		}
		// delete user should succeed and return user id
		dr, _, err := deleteUser(netClient, user3ID, token)
		require.NoError(t, err)
		if dr.ID != user3ID {
			t.Fatal("expect delete user response id to match")
		}

		// update infra admin to regular user should succeed
		userID := user.ID
		user.ID = ""
		user.TenantID = ""
		user.Role = "USER"
		ur, _, err = updateUser(netClient, userID, user, token)
		require.NoError(t, err)
		if ur.ID != userID {
			t.Fatal("update user id mismatch")
		}

	})

}

func TestUserPaging(t *testing.T) {
	t.Log("running TestUserPaging test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	userID := user.ID

	rand1 := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test User Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// randomly create some users
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d users...", n)
		nUsers := []model.User{}
		nToAdd := n - 1 // subtract existing user
		for i := 0; i < nToAdd; i++ {
			user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
			nUsers = append(nUsers, user)
		}

		users, err := getUsers(netClient, token)
		require.NoError(t, err)
		if len(users) != n {
			t.Fatalf("expected users count to be %d, but got %d", n, len(users))
		}
		sort.Sort(model.UsersByID(users))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pUsers := []model.User{}
		nRemain := n
		t.Logf("fetch %d users using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nscpts, err := getUsersNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if nscpts.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, nscpts.PageIndex)
			}
			if nscpts.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, nscpts.PageSize)
			}
			if nscpts.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, nscpts.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(nscpts.UserList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nscpts.UserList))
			}
			nRemain -= pageSize
			for _, sr := range nscpts.UserList {
				pUsers = append(pUsers, sr)
			}
		}

		// verify paging api gives same result as old api
		for i := range pUsers {
			if !reflect.DeepEqual(users[i], pUsers[i]) {
				t.Fatalf("expect user equal, but %+v != %+v", users[i], pUsers[i])
			}
		}
		t.Log("get users from paging api gives same result as old api")

		for _, user := range nUsers {
			if user.ID != userID {
				resp, _, err := deleteUser(netClient, user.ID, token)
				require.NoError(t, err)
				if resp.ID != user.ID {
					t.Fatal("delete user id mismatch")
				}
			}
		}

	})

}

func TestUserWithOperator(t *testing.T) {
	t.Parallel()
	t.Log("running TestUserWithOperator test")
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	defer dbAPI.Close()
	machineCtx := machineTenantContext()
	tenantID := auth.OperatorTenantID
	operatorTenantUser := createOperatorUser(t, dbAPI)
	defer dbAPI.DeleteUser(machineCtx, operatorTenantUser.ID, nil)
	var netClient = &http.Client{
		Timeout: time.Minute,
	}
	token := loginUser(t, netClient, operatorTenantUser)
	operatorTenantUserID := getUserIDFromToken(t, token)
	doc := model.UserPublicKey{ID: operatorTenantUser.ID, TenantID: tenantID, PublicKey: dataECPubKey}
	_, _, err = updateUserPublicKey(netClient, doc, token)
	require.NoError(t, err)
	defer deleteUserPublicKey(netClient, token)
	callerToken := createAPICallerToken(t, operatorTenantUserID, dataECPrivKey)
	tenant := model.Tenant{
		Name: "Test drive tenant",
	}
	resp, _, err := createTenant(netClient, tenant, callerToken)
	require.NoError(t, err)
	defer deleteTenant(netClient, resp.ID, callerToken)
	t.Run("Test UserWithOperator", func(t *testing.T) {
		userName := "test-" + base.GetUUID()
		user := model.User{
			BaseModel: model.BaseModel{TenantID: resp.ID},
			Email:     fmt.Sprintf("%s@example.com", userName),
			Name:      userName,
			Password:  "P@ssw0rd",
			Role:      "INFRA_ADMIN",
		}
		usrResp, _, err := createUser(netClient, &user, callerToken)
		require.NoError(t, err)
		token = loginUser(t, netClient, user)
		defer deleteUser(netClient, usrResp.ID, token)
		users, err := getUsers(netClient, token)
		require.NoError(t, err)
		require.Equal(t, 1, len(users), fmt.Sprintf("User count must be 1. found %d", len(users)))
		t.Logf("Users %+v\n", users)
		require.NotEmpty(t, token, "Token must be empty")
		_, _, err = deleteTenant(netClient, resp.ID, callerToken)
		require.NoError(t, err)
		_, err = getUsers(netClient, token)
		require.Error(t, err)
	})
}

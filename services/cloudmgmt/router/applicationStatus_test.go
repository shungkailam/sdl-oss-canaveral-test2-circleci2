package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// create edge
const (
	APP_STATUS_PATH     = "/v1/applicationstatus"
	APP_STATUS_NEW_PATH = "/v1.0/applicationstatuses"
)

// get edges
func getAppsStatus(netClient *http.Client, token string) ([]model.ApplicationStatus, error) {
	appStatuses := []model.ApplicationStatus{}
	err := doGet(netClient, APP_STATUS_PATH, token, &appStatuses)
	return appStatuses, err
}
func getAppsStatusNew(netClient *http.Client, token string) (model.ApplicationStatusListPayload, error) {
	response := model.ApplicationStatusListPayload{}
	err := doGet(netClient, APP_STATUS_NEW_PATH, token, &response)
	return response, err
}
func getAppsStatusForApp(netClient *http.Client, token string, appID string) ([]model.ApplicationStatus, error) {
	appStatuses := []model.ApplicationStatus{}
	path := fmt.Sprintf("%s/%s", APP_STATUS_PATH, appID)
	err := doGet(netClient, path, token, &appStatuses)
	return appStatuses, err
}
func getAppsStatusForAppNew(netClient *http.Client, token string, appID string) (model.ApplicationStatusListPayload, error) {
	response := model.ApplicationStatusListPayload{}
	path := fmt.Sprintf("%s/%s", APP_STATUS_NEW_PATH, appID)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func TestApplicationStatus(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplicationStatus test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

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

	t.Run("Test App Status", func(t *testing.T) {
		// login as user to get token

		token := loginUser(t, netClient, user)

		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		t.Logf("created project: %+v", project)

		application := createApplicationForProject(t, netClient, tenantID, project.ID, token)
		applicationID := application.ID

		// get app status
		appStatuses, err := getAppsStatus(netClient, token)
		require.NoError(t, err)
		r, err := getAppsStatusNew(netClient, token)
		require.NoError(t, err)
		if len(appStatuses) != len(r.ApplicationStatusList) {
			t.Fatal("expect get and get new to give same count")
		}

		appStatuses, err = getAppsStatusForApp(netClient, token, applicationID)
		require.NoError(t, err)

		r, err = getAppsStatusForAppNew(netClient, token, applicationID)
		require.NoError(t, err)
		if len(r.ApplicationStatusList) != len(appStatuses) {
			t.Fatal("expected app statuses for app old and new to give same count")
		}
	})
}

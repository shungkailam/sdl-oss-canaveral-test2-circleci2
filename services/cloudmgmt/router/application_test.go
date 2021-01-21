package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	APPLICATION_PATH      = "/v1/application"
	APPLICATIONS_PATH     = "/v1/applications"
	APPLICATIONS_PATH_NEW = "/v1.0/applications"
)

// create application
func createApplication(netClient *http.Client, application *model.Application, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, APPLICATION_PATH, *application, token)
	if err == nil {
		application.ID = resp.ID
	}
	return resp, reqID, err
}

// update application
func updateApplication(netClient *http.Client, applicationID string, application model.Application, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", APPLICATION_PATH, applicationID), application, token)
}

// get applications
func getApplications(netClient *http.Client, token string) ([]model.Application, error) {
	applications := []model.Application{}
	err := doGet(netClient, APPLICATIONS_PATH, token, &applications)
	return applications, err
}
func getApplicationsNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ApplicationListResponsePayload, error) {
	applications := model.ApplicationListResponsePayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", APPLICATIONS_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &applications)
	return applications, err
}
func getApplicationsForProject(netClient *http.Client, projectID string, token string) ([]model.Application, error) {
	applications := []model.Application{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/applications", token, &applications)
	return applications, err
}

//

// delete application
func deleteApplication(netClient *http.Client, applicationID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, APPLICATION_PATH, applicationID, token)
}

// get application by id
func getApplicationByID(netClient *http.Client, applicationID string, token string) (model.Application, error) {
	application := model.Application{}
	err := doGet(netClient, APPLICATION_PATH+"/"+applicationID, token, &application)
	return application, err
}

func createApplicationForProject(t *testing.T, netClient *http.Client, tenantID string, projectID string, token string) model.Application {
	appName := fmt.Sprintf("app name-%s", base.GetUUID())
	appDesc := "test app"
	randomName := time.Now().UTC().UnixNano()
	appYamlData := fmt.Sprintf(`apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: deployment-demo%d
spec:
  selector:
    matchLabels:
      demo: deployment
  replicas: 5
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
    type: RollingUpdate
  template:
    metadata:
      labels:
        demo: deployment
        version: v1
    spec:
      containers:
      - name: busybox
        image: busybox
        command: [ "sh", "-c", "while true; do echo hostname; sleep 60; done" ]
        volumeMounts:
        - name: content
          mountPath: /data
      - name: nginx
        image: nginx
        volumeMounts:
          - name: content
            mountPath: /usr/share/nginx/html
            readOnly: true
      volumes:
      - name: content`, randomName)
	app := model.Application{
		ApplicationCore: model.ApplicationCore{
			Name:        appName,
			Description: appDesc,
			ProjectID:   projectID,
		},
		YamlData: appYamlData,
	}
	resp, _, err := createApplication(netClient, &app, token)
	require.NoError(t, err)
	t.Logf("create application successful, %s", resp)
	return app
}

func TestApplication(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplication test")

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

	t.Run("Test Application", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		t.Logf("created project: %+v", project)

		application := createApplicationForProject(t, netClient, tenantID, project.ID, token)
		t.Logf("created application: %+v", application)
		applications, err := getApplications(netClient, token)
		require.NoError(t, err)
		t.Logf("got applications: %+v", applications)
		if len(applications) != 1 {
			t.Fatalf("expected apps count to be 1, but got %d", len(applications))
		}
		applicationJ, err := getApplicationByID(netClient, application.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(applications[0], applicationJ) {
			t.Fatalf("expect application J equal, but %+v != %+v", applications[0], applicationJ)
		}
		appsForProject, err := getApplicationsForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(appsForProject[0], applications[0]) {
			t.Fatalf("expect application equal, but %+v != %+v", appsForProject[0], applications[0])
		}

		// update application
		applicationID := application.ID
		application.ID = ""
		application.Name = fmt.Sprintf("%s-Updated", application.Name)
		ur, _, err := updateApplication(netClient, applicationID, application, token)
		require.NoError(t, err)
		if ur.ID != applicationID {
			t.Fatal("expect update application id to match")
		}

		resp, _, err := deleteApplication(netClient, applicationID, token)
		require.NoError(t, err)
		if resp.ID != applicationID {
			t.Fatal("delete app id mismatch")
		}
		resp, _, err = deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("delete project id mismatch")
		}
	})

}

func TestApplicationPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplicationPaging test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

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

	t.Run("Test Application Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)
		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)

		// randomly create some applications
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d apps...", n)
		for i := 0; i < n; i++ {
			createApplicationForProject(t, netClient, tenantID, project.ID, token)
		}

		applications, err := getApplications(netClient, token)
		require.NoError(t, err)
		if len(applications) != n {
			t.Fatalf("expected apps count to be %d, but got %d", n, len(applications))
		}
		sort.Sort(model.ApplicationsByID(applications))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		apps2 := []model.Application{}
		nRemain := n
		t.Logf("fetch %d apps using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			napps, err := getApplicationsNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if napps.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, napps.PageIndex)
			}
			if napps.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, napps.PageSize)
			}
			if napps.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, napps.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(napps.ApplicationListV2) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(napps.ApplicationListV2))
			}
			nRemain -= pageSize
			for _, app := range model.ApplicationsByIDV2(napps.ApplicationListV2).FromV2() {
				apps2 = append(apps2, app)
			}
		}

		// verify paging api gives same result as old api
		for i := range apps2 {
			if !reflect.DeepEqual(applications[i], apps2[i]) {
				t.Fatalf("expect app equal, but %+v != %+v", applications[i], apps2[i])
			}
		}
		t.Log("get apps from paging api gives same result as old api")

		// delete apps
		for _, app := range apps2 {
			resp, _, err := deleteApplication(netClient, app.ID, token)
			require.NoError(t, err)
			if resp.ID != app.ID {
				t.Fatal("delete app id mismatch")
			}
		}
		// delete project
		resp, _, err := deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("delete project id mismatch")
		}
	})
}

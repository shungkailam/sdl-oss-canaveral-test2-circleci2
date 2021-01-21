package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"cloudservices/common/model"

	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"

	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

var appYamlData = `apiVersion: extensions/v1beta1
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
      - name: content`

func testApp(tenantID, projectID, appName string, edgeIDs []string, edgeSelectors []model.CategoryInfo, originSelectors *[]model.CategoryInfo) model.Application {
	return testAppWithState(tenantID, projectID, appName, edgeIDs, edgeSelectors, originSelectors, nil)
}
func testAppWithState(tenantID, projectID, appName string, edgeIDs []string, edgeSelectors []model.CategoryInfo, originSelectors *[]model.CategoryInfo, state *string) model.Application {
	// create application
	appDesc := "test app"
	randomName := time.Now().UTC().UnixNano()
	return model.Application{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		ApplicationCore: model.ApplicationCore{
			Name:            appName,
			Description:     appDesc,
			ProjectID:       projectID,
			EdgeIDs:         edgeIDs,
			EdgeSelectors:   edgeSelectors,
			OriginSelectors: originSelectors,
			State:           state,
		},
		YamlData: fmt.Sprintf(appYamlData, randomName),
	}
}

func createApplication(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, appName string, projectID string, edgeIDs []string, edgeSelectors []model.CategoryInfo) model.Application {
	return createApplicationWithState(t, dbAPI, tenantID, appName, projectID, edgeIDs, edgeSelectors, nil)
}
func createApplicationWithState(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, appName string, projectID string, edgeIDs []string, edgeSelectors []model.CategoryInfo, state *string) model.Application {
	app := testAppWithState(tenantID, projectID, appName, edgeIDs, edgeSelectors, nil, state)
	rtnApp, err := createApplicationWithCallback(t, dbAPI, &app, tenantID, projectID, nil)
	require.NoError(t, err)
	return *rtnApp
}

func createApplicationWithCallback(t *testing.T, dbAPI api.ObjectModelAPI, app *model.Application, tenantID, projectID string,
	callback func(ctx context.Context, doc interface{}) error,
) (*model.Application, error) {
	// create application
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	resp, err := dbAPI.CreateApplication(ctx, app, callback)
	if err != nil {
		return nil, err
	}
	t.Logf("create application successful, %s", resp)
	app.ID = resp.(model.CreateDocumentResponse).ID
	application, err := dbAPI.GetApplication(ctx, app.ID)
	require.NoError(t, err)
	return &application, nil
}

func createApplicationWithCallbackV2(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, appName string,
	projectID string, edgeIDs []string, edgeSelectors []model.CategoryInfo,
	callback func(ctx context.Context, doc interface{}) error,
	originSelectors *[]model.CategoryInfo,
) model.ApplicationV2 {
	// create application
	randomName := time.Now().UTC().UnixNano()
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	appDesc := "test app"

	app := model.ApplicationV2{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		ApplicationCore: model.ApplicationCore{
			Name:          appName,
			Description:   appDesc,
			ProjectID:     projectID,
			EdgeIDs:       edgeIDs,
			EdgeSelectors: edgeSelectors,
		},
		AppManifest: fmt.Sprintf(appYamlData, randomName),
	}

	resp, err := dbAPI.CreateApplicationV2(ctx, &app, callback)
	require.NoError(t, err)
	// log.Printf("create application successful, %s", resp)
	app.ID = resp.(model.CreateDocumentResponse).ID
	return app
}

func TestApplication(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplication test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeID := edge.ID
	// edge 2 is labeled by cat/v2
	edge2 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue2,
		},
	})
	edgeID2 := edge2.ID
	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID

	// project2 is explicit/edge2
	project2 := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, []string{edgeID2})
	projectID2 := project2.ID

	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID, projectID2})

	// get project
	project, err := dbAPI.GetProject(ctx1, projectID)
	require.NoError(t, err)
	// log.Printf("get project successful, %+v", project)

	if project.EdgeSelectorType != model.ProjectEdgeSelectorTypeCategory {
		t.Fatal("expect project edge selector type to be Category")
	}
	if len(project.EdgeSelectors) != 1 {
		t.Fatal("expect project edge selectors count to be 1")
	}
	if len(project.EdgeIDs) != 1 {
		t.Fatal("expect project edge ids count to be 1")
	}

	// get project 2
	project2, err = dbAPI.GetProject(ctx1, projectID2)
	require.NoError(t, err)
	// log.Printf("get project 2 successful, %+v", project2)
	if project2.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
		t.Fatal("expect project 2 edge selector type to be Explicit")
	}
	if len(project2.EdgeSelectors) != 0 {
		t.Fatal("expect project 2 edge selectors count to be 0")
	}
	if len(project2.EdgeIDs) != 1 {
		t.Fatal("expect project 2 edge ids count to be 1")
	}

	var app, app2, app3, app4 model.Application
	var appv2 model.ApplicationV2

	// Teardown
	defer func() {
		dbAPI.DeleteApplication(ctx1, app4.ID, nil)
		dbAPI.DeleteApplication(ctx1, app3.ID, nil)
		dbAPI.DeleteApplication(ctx1, app4.ID, nil)
		dbAPI.DeleteApplication(ctx1, appv2.ID, nil)
		dbAPI.DeleteApplication(ctx1, app.ID, nil)
		dbAPI.DeleteProject(ctx1, projectID2, nil)
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID2, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("app is project/cat/v1 - so should contain edge", func(t *testing.T) {
		app = testApp(tenantID, projectID, "app name", []string{edgeID, edgeID, edgeID}, []model.CategoryInfo{{
			ID:    categoryID,
			Value: TestCategoryValue1,
		}}, nil,
		)
		rtnApp, err := createApplicationWithCallback(t, dbAPI, &app, tenantID, projectID,
			func(ctx context.Context, doc interface{}) error {
				ap := doc.(*api.App)
				// log.Printf("create app callback, %+v", *ap)
				if len(ap.EdgeSelectors) != 1 {
					t.Fatal("expect callback app edge selectors count to be 1")
				}
				if len(ap.EdgeIDs) != 1 {
					t.Fatal("expect callback app edge ids count to be 1")
				}
				return nil
			},
		)
		require.NoError(t, err)
		appID := rtnApp.ID

		// get application
		application, err := dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)
		// log.Printf("get applicaiton successful, %+v", application)

		if len(application.EdgeSelectors) != 1 {
			t.Fatal("expect app edge selectors count to be 1")
		}
		if len(application.EdgeIDs) != 1 {
			t.Fatal("expect app edge ids count to be 1")
		}
	})

	t.Run("app2 is project/cat/v2 - so should not contain any edge", func(t *testing.T) {
		app2 = testApp(tenantID, projectID, "app name 2", []string{edgeID, edgeID}, []model.CategoryInfo{{
			ID:    categoryID,
			Value: TestCategoryValue2,
		}}, nil,
		)
		// app2 is project/cat/v2 - so should not contain any edge
		rtnApp2, err := createApplicationWithCallback(t, dbAPI, &app2, tenantID, projectID,
			func(ctx context.Context, doc interface{}) error {
				ap := doc.(*api.App)
				// log.Printf("create app 2 callback, %+v", *ap)
				if len(ap.EdgeSelectors) != 1 {
					t.Fatal("expect callback app 2 edge selectors count to be 1")
				}
				if len(ap.EdgeIDs) != 0 {
					t.Fatal("expect callback app 2 edge ids count to be 0")
				}
				return nil
			},
		)
		require.NoError(t, err)
		appID2 := rtnApp2.ID
		application2, err := dbAPI.GetApplication(ctx1, appID2)
		require.NoError(t, err)
		// log.Printf("get applicaiton 2 successful, %+v", application2)

		if len(application2.EdgeSelectors) != 1 {
			t.Fatal("expect app 2 edge selectors count to be 1")
		}
		if len(application2.EdgeIDs) != 0 {
			t.Fatal("expect app 2 edge ids count to be 0")
		}
	})

	t.Run("app3 is project2/explicit/edge1 - so should not contain any edge", func(t *testing.T) {
		app3 = testApp(tenantID, projectID2, "app name 3", []string{edgeID2}, []model.CategoryInfo{{
			ID:    categoryID,
			Value: TestCategoryValue1,
		}}, nil,
		)
		// app3 is project2/explicit/edge1 - so should not contain any edge
		rtnApp3, err := createApplicationWithCallback(t, dbAPI, &app3, tenantID, projectID2,
			func(ctx context.Context, doc interface{}) error {
				ap := doc.(*api.App)
				// log.Printf("create app 3 callback, %+v", *ap)
				if len(ap.EdgeSelectors) != 0 {
					t.Fatal("expect callback app 3 edge selectors count to be 0")
				}
				if len(ap.EdgeIDs) != 1 {
					t.Fatal("expect callback app 3 edge ids count to be 1")
				}
				return nil
			},
		)
		require.NoError(t, err)
		appID3 := rtnApp3.ID
		application3, err := dbAPI.GetApplication(ctx1, appID3)
		require.NoError(t, err)
		// log.Printf("get applicaiton 3 successful, %+v", application3)

		if len(application3.EdgeSelectors) != 0 {
			t.Fatal("expect app 3 edge selectors count to be 0")
		}
		if len(application3.EdgeIDs) != 1 {
			t.Fatal("expect app 3 edge ids count to be 1")
		}
	})

	t.Run("app4 is project/explicit/edge2 - so should contain edge2", func(t *testing.T) {
		app4 = testApp(tenantID, projectID2, "app name 4", nil, []model.CategoryInfo{{
			ID:    categoryID,
			Value: TestCategoryValue2,
		}}, nil)
		rtnApp4, err := createApplicationWithCallback(t, dbAPI, &app4, tenantID, projectID2,
			func(ctx context.Context, doc interface{}) error {
				ap := doc.(*api.App)
				// log.Printf("create app 4 callback, %+v", *ap)
				if len(ap.EdgeSelectors) != 0 {
					t.Fatal("expect callback app 4 edge selectors count to be 0")
				}
				if len(ap.EdgeIDs) != 0 {
					t.Fatal("expect callback app 4 edge ids count to be 0")
				}
				return nil
			})
		require.NoError(t, err)
		appID4 := rtnApp4.ID
		application4, err := dbAPI.GetApplication(ctx1, appID4)
		require.NoError(t, err)
		// log.Printf("get applicaiton 4 successful, %+v", application4)

		if len(application4.EdgeSelectors) != 0 {
			t.Fatal("expect app 4 edge selectors count to be 0")
		}
		if len(application4.EdgeIDs) != 0 {
			t.Fatal("expect app 4 edge ids count to be 0")
		}
	})

	t.Run("appv2 is like app, but created using v2 API - is project/cat/v1 - so should contain edge", func(t *testing.T) {
		appv2 = createApplicationWithCallbackV2(t, dbAPI, tenantID, "app name v2", projectID, []string{edgeID, edgeID, edgeID}, []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		}, func(ctx context.Context, doc interface{}) error {
			ap := doc.(*api.App)
			// log.Printf("create app callback, %+v", *ap)
			if len(ap.EdgeSelectors) != 1 {
				t.Fatal("expect callback app edge selectors count to be 1")
			}
			if len(ap.EdgeIDs) != 1 {
				t.Fatal("expect callback app edge ids count to be 1")
			}
			return nil
		}, nil)
		appv2ID := appv2.ID

		// get application v2
		applicationv2, err := dbAPI.GetApplication(ctx1, appv2ID)
		require.NoError(t, err)
		// log.Printf("get applicaiton v2 successful, %+v", applicationv2)

		if len(applicationv2.EdgeSelectors) != 1 {
			t.Fatal("expect app v2 edge selectors count to be 1")
		}
		if len(applicationv2.EdgeIDs) != 1 {
			t.Fatal("expect app v2 edge ids count to be 1")
		}
	})

	t.Run("Create/Get/Delete Application", func(t *testing.T) {
		t.Log("running Create/Get/Delete Application test")

		appName := "app name"
		appDesc := "test app"
		appYamlDataUpdated := `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: deployment-demo
spec:
  selector:
    matchLabels:
      demo: deployment
  replicas: 2
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
      - name: content`

		apps, err := dbAPI.SelectAllApplications(ctx1)
		require.NoError(t, err)
		// log.Printf("select all applications 1 successful, %+v\n", apps)

		apps, err = dbAPI.SelectAllApplications(ctx2)
		require.NoError(t, err)
		// log.Printf("select all applications 2 successful, %+v\n", apps)

		apps, err = dbAPI.SelectAllApplications(ctx3)
		require.NoError(t, err)
		// log.Printf("select all applications 3 successful, %+v\n", apps)

		app.YamlData = appYamlDataUpdated
		_, err = dbAPI.UpdateApplication(ctx1, &app, nil)
		require.NoError(t, err)
		// log.Printf("update application successful, %+v", upResp)

		// select all vs select all W
		var w bytes.Buffer
		apps1, err := dbAPI.SelectAllApplications(ctx1)
		require.NoError(t, err)
		apps2 := &[]model.Application{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllApplicationsW, apps2, &w)
		require.NoError(t, err)
		sort.Sort(model.ApplicationsByID(apps1))
		sort.Sort(model.ApplicationsByID(*apps2))
		if !reflect.DeepEqual(&apps1, apps2) {
			t.Fatalf("expect select applications and select applications w results to be equal %+v vs %+v", apps1, *apps2)
		}

		// get application
		app, err := dbAPI.GetApplication(ctx1, app.ID)
		require.NoError(t, err)
		// log.Printf("get application successful, %+v", app)

		if app.ID != app.ID || app.Name != appName || app.Description != appDesc || app.YamlData != appYamlDataUpdated {
			t.Fatal("application data mismatch")
		}

		err = dbAPI.GetApplicationWV2(ctx1, app.ID, &w, &http.Request{URL: &url.URL{}})
		require.NoError(t, err)
		appv2 := model.ApplicationV2{}
		err = json.NewDecoder(&w).Decode(&appv2)
		require.NoError(t, err)
		appFromV2 := appv2.FromV2()
		if !reflect.DeepEqual(appFromV2, app) {
			t.Fatal("expect app to equal conversion from v2")
		}

		err = dbAPI.SelectAllApplicationsWV2(ctx1, &w, &http.Request{URL: &url.URL{}})
		require.NoError(t, err)
		appsRes := model.ApplicationListResponsePayload{}
		err = json.NewDecoder(&w).Decode(&appsRes)
		require.NoError(t, err)
		appsFromV2 := model.ApplicationsByIDV2(appsRes.ApplicationListV2).FromV2()
		sort.Sort(model.ApplicationsByID(appsFromV2))
		if !reflect.DeepEqual(&apps1, &appsFromV2) {
			t.Fatalf("expect select applications and select applications wv2 results to be equal %+v vs %+v", apps1, appsFromV2)
		}

		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		apps, err = dbAPI.SelectAllApplicationsForProject(newCtx, projectID)
		require.Error(t, err, "expect auth 1 get apps for project to fail")
		apps, err = dbAPI.SelectAllApplicationsForProject(ctx2, projectID)
		require.Error(t, err, "expect auth 2 get apps for project to fail")
		apps, err = dbAPI.SelectAllApplicationsForProject(ctx3, projectID)
		require.NoError(t, err)
		if len(apps) != 3 {
			t.Fatalf("expect auth 3 get apps for project count to be 3, got %d", len(apps))
		}

		app.State = model.UndeployEntityState.StringPtr()
		_, err = dbAPI.UpdateApplication(ctx1, &app, nil)
		require.NoError(t, err)
		apps1, err = dbAPI.SelectAllApplications(ctx1)
		require.NoError(t, err)
		undeployCount := 0
		for _, app := range apps1 {
			if app.State != nil && *app.State == string(model.UndeployEntityState) {
				undeployCount++
			}
		}
		if undeployCount != 1 {
			t.Fatalf("Expected undeploy count of 1, found %d", undeployCount)
		}
		authContext, _ := base.GetAuthContext(ctx1)
		// select all data sources for project from edge
		edgeAuthContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      "1234",
				"projects":    authContext.Claims["projects"],
			},
		}
		edgeCtx := context.WithValue(context.Background(), base.AuthContextKey, edgeAuthContext)
		appsForEdge, err := dbAPI.SelectAllApplications(edgeCtx)
		require.NoError(t, err)
		if len(appsForEdge) != len(apps1)-undeployCount {
			t.Fatalf("Expected %d, found %d", len(apps1)-undeployCount, len(appsForEdge))
		}
		for _, app := range appsForEdge {
			if app.State != nil && *app.State != string(model.DeployEntityState) {
				t.Fatalf("Edge is not expected to see entities not in deploy state")
			}
		}
	})

	t.Run("SelectAllApplications", func(t *testing.T) {
		t.Log("running SelectAllApplications test")
		applications, err := dbAPI.SelectAllApplications(ctx1)
		require.NoError(t, err)
		for _, application := range applications {
			_, err := json.Marshal(application)
			require.NoError(t, err)
		}
	})

	t.Run("ApplicationConversion", func(t *testing.T) {
		t.Log("running ApplicationConversion test")

		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		applications := []model.Application{
			{
				BaseModel: model.BaseModel{
					ID:        "app-id",
					TenantID:  tenantID,
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				ApplicationCore: model.ApplicationCore{
					Name:        "test-app",
					Description: "test application",
					ProjectID:   "proj-id",
				},
				YamlData: "test",
			},
		}
		for _, app := range applications {
			appDBO := api.ApplicationDBO{}
			app2 := model.Application{}
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
		app := testApp(tenantID, projectID, "app name "+funk.RandomString(10), []string{edgeID, edgeID, edgeID}, []model.CategoryInfo{{
			ID:    categoryID,
			Value: TestCategoryValue1,
		}}, nil)
		app.ID = id
		return dbAPI.CreateApplication(ctx1, &app, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetApplication(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteApplication(ctx1, id, nil)
	}))
}

func TestApplicationEdgeSelection(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplicationEdgeSelection test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	edgeID := edge.ID
	// edge 2 is labeled by cat/v2
	edge2 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue2,
	}})
	edgeID2 := edge2.ID
	// edge 3 is labeled by cat/v1
	edge3 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	edgeID3 := edge3.ID
	// edge 4 is labeled by cat/v2
	edge4 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue2,
	}})
	edgeID4 := edge4.ID

	allEdgeIDs := []string{edgeID, edgeID2, edgeID3, edgeID4}
	catInfos := []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}}

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, catInfos)
	projectID := project.ID

	// project2 is explicit, all edges
	project2 := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, allEdgeIDs)
	projectID2 := project2.ID

	// project3 is explicit, all edges
	project3 := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, allEdgeIDs)
	projectID3 := project3.ID

	ctx1, _, _ := makeContext(tenantID, []string{projectID, projectID2, projectID3})

	// get project
	project, err := dbAPI.GetProject(ctx1, projectID)
	require.NoError(t, err)
	// log.Printf("get project successful, %+v", project)

	if project.EdgeSelectorType != model.ProjectEdgeSelectorTypeCategory {
		t.Fatal("expect project edge selector type to be Category")
	}
	if len(project.EdgeSelectors) != 1 {
		t.Fatal("expect project edge selectors count to be 1")
	}
	if len(project.EdgeIDs) != 2 {
		t.Fatal("expect project edge ids count to be 2")
	}

	// get project 2
	project2, err = dbAPI.GetProject(ctx1, projectID2)
	require.NoError(t, err)
	// log.Printf("get project 2 successful, %+v", project2)
	if project2.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
		t.Fatal("expect project 2 edge selector type to be Explicit")
	}
	if len(project2.EdgeSelectors) != 0 {
		t.Fatal("expect project 2 edge selectors count to be 0")
	}
	if len(project2.EdgeIDs) != 4 {
		t.Fatal("expect project 2 edge ids count to be 4")
	}

	// get project 3
	project3, err = dbAPI.GetProject(ctx1, projectID3)
	require.NoError(t, err)
	// log.Printf("get project 3 successful, %+v", project3)
	if project3.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
		t.Fatal("expect project 3 edge selector type to be Explicit")
	}
	if len(project3.EdgeSelectors) != 0 {
		t.Fatal("expect project 3 edge selectors count to be 0")
	}
	if len(project3.EdgeIDs) != 4 {
		t.Fatal("expect project 3 edge ids count to be 4")
	}

	// app is project/cat/v1 - so should contain edge
	app := createApplication(t, dbAPI, tenantID, "app name", projectID, []string{}, catInfos)
	appID := app.ID

	// app2 is project2/explicit/all edges
	app2 := createApplication(t, dbAPI, tenantID, "app name 2", projectID2, allEdgeIDs, catInfos)
	appID2 := app2.ID

	// app3 is project3/explicit/all edges
	app3 := createApplication(t, dbAPI, tenantID, "app name 3", projectID3, allEdgeIDs, catInfos)
	appID3 := app3.ID

	// get application
	application, err := dbAPI.GetApplication(ctx1, appID)
	require.NoError(t, err)
	// log.Printf("get applicaiton successful, %+v", application)

	if len(application.EdgeSelectors) != 1 {
		t.Fatal("expect app edge selectors count to be 1")
	}
	if len(application.EdgeIDs) != 2 {
		t.Fatal("expect app edge ids count to be 2")
	}

	application2, err := dbAPI.GetApplication(ctx1, appID2)
	require.NoError(t, err)
	// log.Printf("get applicaiton 2 successful, %+v", application2)

	if len(application2.EdgeSelectors) != 0 {
		t.Fatal("expect app 2 edge selectors count to be 0")
	}
	if len(application2.EdgeIDs) != 4 {
		t.Fatal("expect app 2 edge ids count to be 4")
	}

	application3, err := dbAPI.GetApplication(ctx1, appID3)
	require.NoError(t, err)
	// log.Printf("get applicaiton 3 successful, %+v", application3)

	if len(application3.EdgeSelectors) != 0 {
		t.Fatal("expect app 3 edge selectors count to be 0")
	}
	if len(application3.EdgeIDs) != 4 {
		t.Fatal("expect app 3 edge ids count to be 4")
	}

	// Teardown
	defer func() {
		dbAPI.DeleteApplication(ctx1, appID, nil)
		dbAPI.DeleteApplication(ctx1, appID2, nil)
		dbAPI.DeleteApplication(ctx1, appID3, nil)
		dbAPI.DeleteProject(ctx1, projectID3, nil)
		dbAPI.DeleteProject(ctx1, projectID2, nil)
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID4, nil)
		dbAPI.DeleteEdge(ctx1, edgeID3, nil)
		dbAPI.DeleteEdge(ctx1, edgeID2, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Application", func(t *testing.T) {
		t.Log("running Create/Get/Delete Application test")

		// project EdgeSelectorType -> Explicit then back to -> Category
		project.EdgeSelectorType = "Explicit"
		project.EdgeIDs = allEdgeIDs
		_, err := dbAPI.UpdateProject(ctx1, &project, nil)
		require.NoError(t, err)
		project.EdgeSelectorType = "Category"
		project.EdgeSelectors = catInfos
		_, err = dbAPI.UpdateProject(ctx1, &project, nil)
		require.NoError(t, err)
		project, err = dbAPI.GetProject(ctx1, projectID)
		require.NoError(t, err)
		// log.Printf("get project successful, %+v", project)

		if project.EdgeSelectorType != model.ProjectEdgeSelectorTypeCategory {
			t.Fatal("expect project edge selector type to be Category")
		}
		if len(project.EdgeSelectors) != 1 {
			t.Fatal("expect project edge selectors count to be 1")
		}
		if len(project.EdgeIDs) != 2 {
			t.Fatal("expect project edge ids count to be 2")
		}

		// project2 EdgeSelectorType -> Category then back to -> Explicit
		project2.EdgeSelectorType = "Category"
		project2.EdgeSelectors = catInfos
		_, err = dbAPI.UpdateProject(ctx1, &project2, nil)
		require.NoError(t, err)
		project2.EdgeSelectorType = "Explicit"
		project2.EdgeIDs = allEdgeIDs
		_, err = dbAPI.UpdateProject(ctx1, &project2, nil)
		require.NoError(t, err)
		project2, err = dbAPI.GetProject(ctx1, projectID2)
		require.NoError(t, err)
		// log.Printf("get project 2 successful, %+v", project2)
		if project2.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
			t.Fatal("expect project 2 edge selector type to be Explicit")
		}
		if len(project2.EdgeSelectors) != 0 {
			t.Fatal("expect project 2 edge selectors count to be 0")
		}
		if len(project2.EdgeIDs) != 4 {
			t.Fatal("expect project 2 edge ids count to be 4")
		}

		// project 3: 4 edges to 2 edges then back to 4 edges
		project3.EdgeIDs = []string{edgeID, edgeID2}
		_, err = dbAPI.UpdateProject(ctx1, &project3, nil)
		require.NoError(t, err)
		project3.EdgeIDs = allEdgeIDs
		_, err = dbAPI.UpdateProject(ctx1, &project3, nil)
		require.NoError(t, err)

		project3, err = dbAPI.GetProject(ctx1, projectID3)
		require.NoError(t, err)
		// log.Printf("get project 3 successful, %+v", project3)
		if project3.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
			t.Fatal("expect project 3 edge selector type to be Explicit")
		}
		if len(project3.EdgeSelectors) != 0 {
			t.Fatal("expect project 3 edge selectors count to be 0")
		}
		if len(project3.EdgeIDs) != 4 {
			t.Fatal("expect project 3 edge ids count to be 4")
		}

		// get application
		application, err := dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)
		// log.Printf("get applicaiton successful, %+v", application)

		// when project EdgeSelectorType change to Explicit,
		// application EdgeSelector should be cleared
		if len(application.EdgeSelectors) != 0 {
			t.Fatal("expect app edge selectors count to be 0")
		}
		if len(application.EdgeIDs) != 2 {
			t.Fatal("expect app edge ids count to be 2")
		}

		application2, err := dbAPI.GetApplication(ctx1, appID2)
		require.NoError(t, err)
		// log.Printf("get applicaiton 2 successful, %+v", application2)

		if len(application2.EdgeSelectors) != 0 {
			t.Fatal("expect app 2 edge selectors count to be 0")
		}
		if len(application2.EdgeIDs) != 0 {
			t.Fatal("expect app 2 edge ids count to be 0")
		}

		application3, err := dbAPI.GetApplication(ctx1, appID3)
		require.NoError(t, err)
		// log.Printf("get applicaiton 3 successful, %+v", application3)

		if len(application3.EdgeSelectors) != 0 {
			t.Fatal("expect app 3 edge selectors count to be 0")
		}
		if len(application3.EdgeIDs) != 2 {
			t.Fatal("expect app 3 edge ids count to be 2")
		}
	})
}

func TestApplicationOriginSelectors(t *testing.T) {
	if !*config.Cfg.EnableAppOriginSelectors {
		t.Skip("skipping origin selectors test because the feature is not enabled")
	}
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	catInfoVal1 := model.CategoryInfo{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{catInfoVal1})
	edgeID := edge.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{catInfoVal1})
	projectID := project.ID

	// Non existent category ID
	nonExistCatIDSel := []model.CategoryInfo{{
		ID:    "non-existent-cat",
		Value: TestCategoryValue1,
	}}
	app := testApp(tenantID, projectID, "test app", []string{edgeID}, []model.CategoryInfo{catInfoVal1}, &nonExistCatIDSel)
	_, err := createApplicationWithCallback(t, dbAPI, &app, tenantID, projectID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Record not found error")

	// one of the category values does not exist
	missingValueInfos := []model.CategoryInfo{catInfoVal1, {ID: categoryID, Value: "does not exist"}}
	app = testApp(tenantID, projectID, "test app", []string{edgeID}, []model.CategoryInfo{catInfoVal1}, &missingValueInfos)
	_, err = createApplicationWithCallback(t, dbAPI, &app, tenantID, projectID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Record not found error")

	validateOriginSels := func(expected, actual model.CategoryInfo) {
		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("expected %+v, but got %v", expected, actual)
		}
	}

	// Successfull case
	correctCatInfos := []model.CategoryInfo{catInfoVal1}
	app = testApp(tenantID, projectID, "test app", []string{edgeID}, []model.CategoryInfo{catInfoVal1}, &correctCatInfos)
	callbackFunc := func(ctx context.Context, doc interface{}) error {
		app := doc.(*api.App)
		if app.OriginSelectors == nil {
			t.Fatalf("expected origin selectors to be non-nil")
		}
		if len(*app.OriginSelectors) != 1 {
			t.Fatalf("expected 1 origin selector to be associated with the app but found %d", len(*app.OriginSelectors))
		}
		validateOriginSels((*app.OriginSelectors)[0], correctCatInfos[0])
		return nil
	}
	resp, err := createApplicationWithCallback(t, dbAPI, &app, tenantID, projectID, callbackFunc)
	require.NoErrorf(t, err, "expected the app to be created successfully")

	appID := resp.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})
	apps, err := dbAPI.SelectAllApplicationsForProject(ctx, projectID)
	require.NoErrorf(t, err, "failed to select apps for project")

	if len(apps) != 1 {
		t.Fatalf("expected 1 app, but got %d", len(apps))
	}
	if apps[0].OriginSelectors == nil {
		t.Fatalf("expected origin selectors to be non-nil")
	}
	validateOriginSels((*apps[0].OriginSelectors)[0], catInfoVal1)

	rtnApp, err := dbAPI.GetApplication(ctx, appID)
	require.NoError(t, err)
	if rtnApp.OriginSelectors == nil {
		t.Fatalf("expected origin selectors to be non-nil")
	}
	validateOriginSels((*rtnApp.OriginSelectors)[0], catInfoVal1)

	app.ID = appID
	*app.OriginSelectors = append(*app.OriginSelectors, model.CategoryInfo{ID: categoryID, Value: TestCategoryValue2})
	_, err = dbAPI.UpdateApplication(ctx, &app, func(ctx context.Context, doc interface{}) error {
		app := doc.(*api.App)
		if app.OriginSelectors == nil {
			t.Fatalf("expected origin selectors to be non-nil, but got  nil")
		}
		if len(*app.OriginSelectors) != 2 {
			t.Fatalf("expected 2 origin selector category values but got %d", len(*app.OriginSelectors))
		}
		return nil
	})
	require.NoError(t, err)

	// Set the origin selectors to nil
	app.OriginSelectors = nil
	_, err = dbAPI.UpdateApplication(ctx, &app, func(ctx context.Context, doc interface{}) error {
		app := doc.(*api.App)
		if app.OriginSelectors != nil {
			t.Fatalf("expected origin selectors to be nil, but got %+v", *app.OriginSelectors)
		}
		return nil
	})
	require.NoError(t, err)

	// Teardown
	defer func() {
		_, err := dbAPI.DeleteApplication(ctx, appID, nil)
		require.NoErrorf(t, err, "failed to delete application: %s", appID)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteCategory(ctx, categoryID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()
}

// assertDataSource checks whether the data source has the expected fields/topics or not
func assertDataSource(ctx context.Context, t *testing.T, dbAPI api.ObjectModelAPI, dsID string, expectedTopics []string) *model.DataSource {
	ds, err := dbAPI.GetDataSource(ctx, dsID)
	require.NoError(t, err, "failed to get data source")

	if len(ds.Fields) != len(expectedTopics) {
		t.Fatalf("expected %d fields, but got %d", len(expectedTopics), len(ds.Fields))
	}
	expected, actual := make(map[string]bool), make(map[string]bool)
	for _, t := range expectedTopics {
		expected[t] = true
	}

	for _, f := range ds.Fields {
		actual[f.MQTTTopic] = true
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("expected topics: %v, but got %v", expected, actual)
	}
	return &ds
}

func assertApplicationEndpoints(ctx context.Context, t *testing.T, dbAPI api.ObjectModelAPI, appID string, expected []model.DataIfcEndpoint) {
	app, err := dbAPI.GetApplication(ctx, appID)
	require.NoError(t, err)
	// only assert equal up to permutation
	sort.Sort(model.DataIfcEndpointsByID(expected))
	sort.Sort(model.DataIfcEndpointsByID(app.DataIfcEndpoints))
	if !reflect.DeepEqual(expected, app.DataIfcEndpoints) {
		t.Fatalf("expected %+v, but  got %+v", expected, app.DataIfcEndpoints)
	}
}

func TestApplicationEndpoints(t *testing.T) {
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	catInfoVal1 := model.CategoryInfo{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}

	// edge 1 is labeled by cat/v1
	edge1 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{catInfoVal1})
	edgeID1 := edge1.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{catInfoVal1})
	projectID := project.ID

	ctx, _, _ := makeContext(tenantID, []string{projectID})

	// Failure Case: Data source does not exist
	app1 := testApp(tenantID, projectID, "test-app1", []string{edgeID1}, []model.CategoryInfo{catInfoVal1}, nil)
	app1.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: "non-existant", Name: "does not matter", Value: "does not matter"}}
	_, err := createApplicationWithCallback(t, dbAPI, &app1, tenantID, projectID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Precondition failed")

	// Failure case: Data source exists but not w/ Ifc Info, that is it is not an interface
	badDataSrc := createDataSource(t, dbAPI, tenantID, edgeID1, categoryID, "v1")
	defer dbAPI.DeleteDataSource(ctx, badDataSrc.ID, nil)
	app1.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: badDataSrc.ID, Name: "does not matter", Value: "does not matter"}}
	_, err = createApplicationWithCallback(t, dbAPI, &app1, tenantID, projectID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Invalid input data")

	// Failure case: Data source with ifcInfo of Kind IN, does not have field/topic required by app
	inIfcInfo := model.DataSourceIfcInfo{Class: "DATAINTERFACE",
		Kind: "IN", Protocol: "DATAINTERFACE", Img: "foo",
		ProjectID: "ingress", DriverID: "bar",
	}
	inputTopic := "test-topic-" + base.GetUUID()
	inputFieldName := "field-name-" + base.GetUUID()
	resp, err := createDataSourceWithSelectorsFields(t, ctx, dbAPI, "data-in-interface-"+base.GetUUID(),
		tenantID, edge1.ID, "Model 3", "DATAINTERFACE", inIfcInfo, nil,
		[]model.DataSourceFieldInfo{
			{
				DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
					Name:      inputFieldName,
					FieldType: "field-type-1",
				},
				MQTTTopic: inputTopic,
			},
		},
	)
	require.NoErrorf(t, err, "failed to create data source")

	inDataSourceID := resp.(model.CreateDocumentResponse).ID
	defer dbAPI.DeleteDataSource(ctx, inDataSourceID, nil)
	app1.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: inDataSourceID, Name: "does_not_exist", Value: "does_not_exist"}}
	_, err = createApplicationWithCallback(t, dbAPI, &app1, tenantID, projectID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Precondition failed")

	// Success case: Create app with correct field/topic of kind IN
	app1.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: inDataSourceID, Name: inputFieldName, Value: inputTopic}}
	rtnApp1, err := createApplicationWithCallback(t, dbAPI, &app1, tenantID, projectID, nil)
	require.NoError(t, err)
	defer func() { dbAPI.DeleteApplication(ctx, rtnApp1.ID, nil) }()
	assertApplicationEndpoints(ctx, t, dbAPI, rtnApp1.ID, app1.DataIfcEndpoints)

	// Success case: Create a new app with only out Ifc
	outIfcInfo := model.DataSourceIfcInfo{Class: "DATAINTERFACE",
		Kind: "OUT", Protocol: "DATAINTERFACE", Img: "foo",
		ProjectID: "ingress", DriverID: "bar",
	}
	randomTopic, randomFieldName := "random-topic-"+base.GetUUID(), "random-field-name-"+base.GetUUID()
	resp, err = createDataSourceWithSelectorsFields(t, ctx, dbAPI, "data-out-interface-"+base.GetUUID(),
		tenantID, edge1.ID, "Model 3", "DATAINTERFACE", outIfcInfo,
		[]model.DataSourceFieldSelector{
			{
				CategoryInfo: model.CategoryInfo{
					ID:    categoryID,
					Value: TestCategoryValue1,
				},
				Scope: []string{randomFieldName},
			},
		},
		[]model.DataSourceFieldInfo{
			{
				DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
					Name:      randomFieldName,
					FieldType: "field-type-1",
				},
				MQTTTopic: randomTopic,
			},
		},
	)
	require.NoError(t, err, "failed to create data source")

	outTopic1, outputTopic2 := "output-topic1-"+base.GetUUID(), "output-topic2-"+base.GetUUID()
	outDataSourceID := resp.(model.CreateDocumentResponse).ID
	defer dbAPI.DeleteDataSource(ctx, outDataSourceID, nil)
	app2 := testApp(tenantID, projectID, "test-app2", []string{edgeID1}, []model.CategoryInfo{catInfoVal1}, nil)
	app2.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: outDataSourceID, Name: "output-topic1", Value: outTopic1}}
	rtnApp2, err := createApplicationWithCallback(t, dbAPI, &app2, tenantID, projectID, nil)
	require.NoError(t, err)
	defer func() { dbAPI.DeleteApplication(ctx, rtnApp2.ID, nil) }()
	assertApplicationEndpoints(ctx, t, dbAPI, rtnApp2.ID, app2.DataIfcEndpoints)

	// Failure case: Update app1 to use the same out Ifc, it will conflict
	rtnApp1.DataIfcEndpoints = append(rtnApp1.DataIfcEndpoints, model.DataIfcEndpoint{Value: outTopic1, Name: "output-topic1", ID: outDataSourceID})
	_, err = dbAPI.UpdateApplication(ctx, rtnApp1, func(ctx context.Context, i interface{}) error {
		t.Fatal("did not expect to be called")
		return nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "please try another topic")

	// Failure case: Update app1 to use the same out Ifc field name but different value of the topic
	rtnApp1.DataIfcEndpoints[1] = model.DataIfcEndpoint{Value: "does not matter", Name: "output-topic1", ID: outDataSourceID}
	_, err = dbAPI.UpdateApplication(ctx, rtnApp1, func(ctx context.Context, i interface{}) error {
		t.Fatal("did not expect to be called")
		return nil
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists and is not claimed by")

	// Success case: Update app2 topic to outputTopic2 and then update app1 to use outTopic1
	rtnApp2.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: outDataSourceID, Name: "output-topic2", Value: outputTopic2}}
	resp, err = dbAPI.UpdateApplication(ctx, rtnApp2, nil)
	require.NoError(t, err)
	assertApplicationEndpoints(ctx, t, dbAPI, rtnApp2.ID, rtnApp2.DataIfcEndpoints)

	rtnApp1.DataIfcEndpoints[1] = model.DataIfcEndpoint{Value: outTopic1, Name: "output-topic1", ID: outDataSourceID}
	resp, err = dbAPI.UpdateApplication(ctx, rtnApp1, nil)
	require.NoError(t, err)
	assertApplicationEndpoints(ctx, t, dbAPI, rtnApp1.ID, rtnApp1.DataIfcEndpoints)

	// Success Case: Delete the app2 and assert that data source fields got updated
	_, err = dbAPI.DeleteApplication(ctx, rtnApp2.ID, nil)
	require.NoError(t, err)
	assertDataSource(ctx, t, dbAPI, outDataSourceID, []string{outTopic1, randomTopic})

	// Success Case: Delete the app1 and assert that data source fields got updated
	_, err = dbAPI.DeleteApplication(ctx, rtnApp1.ID, nil)
	require.NoError(t, err)
	assertDataSource(ctx, t, dbAPI, outDataSourceID, []string{randomTopic})

	// Teardown
	defer func() {
		_, err := dbAPI.DeleteApplication(ctx, rtnApp1.ID, nil)
		require.NoErrorf(t, err, "failed to delete application: %s", rtnApp1.ID)
		_, err = dbAPI.DeleteApplication(ctx, rtnApp2.ID, nil)
		require.NoErrorf(t, err, "failed to delete application: %s", rtnApp2.ID)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteDataSource(ctx, outDataSourceID, nil)
		dbAPI.DeleteEdge(ctx, edgeID1, nil)
		dbAPI.DeleteCategory(ctx, categoryID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()
}

func TestApplicationStartStop(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplicationStartStop test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	edgeID := edge.ID
	// edge 2 is labeled by cat/v2
	edge2 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue2,
	}})
	edgeID2 := edge2.ID
	// edge 3 is labeled by cat/v1
	edge3 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	edgeID3 := edge3.ID

	allEdgeIDs := []string{edgeID, edgeID2, edgeID3}
	catInfos := []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	}

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, catInfos)
	projectID := project.ID

	// project2 is explicit, all edges
	project2 := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, allEdgeIDs)
	projectID2 := project2.ID

	ctx1, _, _ := makeContext(tenantID, []string{projectID, projectID2})

	defer func() {
		dbAPI.DeleteProject(ctx1, projectID2, nil)
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID3, nil)
		dbAPI.DeleteEdge(ctx1, edgeID2, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	// get project
	project, err := dbAPI.GetProject(ctx1, projectID)
	require.NoError(t, err)
	if project.EdgeSelectorType != model.ProjectEdgeSelectorTypeCategory {
		t.Fatal("expect project edge selector type to be Category")
	}
	if len(project.EdgeSelectors) != 1 {
		t.Fatal("expect project edge selectors count to be 1")
	}
	if len(project.EdgeIDs) != 2 {
		t.Fatal("expect project edge ids count to be 2")
	}

	// get project 2
	project2, err = dbAPI.GetProject(ctx1, projectID2)
	require.NoError(t, err)
	if project2.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
		t.Fatal("expect project 2 edge selector type to be Explicit")
	}
	if len(project2.EdgeSelectors) != 0 {
		t.Fatal("expect project 2 edge selectors count to be 0")
	}
	if len(project2.EdgeIDs) != 3 {
		t.Fatal("expect project 2 edge ids count to be 3")
	}

	// app is project/cat/v1 - so should contain edge
	app := createApplication(t, dbAPI, tenantID, "app name", projectID, []string{}, catInfos)
	appID := app.ID

	// app2 is project2/explicit/all edges
	app2 := createApplication(t, dbAPI, tenantID, "app name 2", projectID2, allEdgeIDs, catInfos)
	appID2 := app2.ID

	// Teardown
	defer func() {
		dbAPI.DeleteApplication(ctx1, appID, nil)
		dbAPI.DeleteApplication(ctx1, appID2, nil)
	}()

	t.Run("Start/Stop Application", func(t *testing.T) {
		t.Log("running Start/Stop Application test")

		// get application
		application, err := dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 2 {
			t.Fatalf("expected edge count of 2, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application.ExcludeEdgeIDs))
		}
		application.ExcludeEdgeIDs = []string{edge3.ID}
		_, err = dbAPI.UpdateApplication(ctx1, &application, func(ctx context.Context, doc interface{}) error {
			updatedApp, ok := doc.(*api.App)
			if !ok {
				t.Fatalf("unexpected type in callback %+v", doc)
			}
			if len(updatedApp.EdgeIDs) != 1 {
				t.Fatalf("expected edge count of 1, found %d", len(updatedApp.EdgeIDs))
			}
			if len(updatedApp.ExcludeEdgeIDs) != 1 {
				t.Fatalf("expected excluded edge count of 1, found %d", len(updatedApp.ExcludeEdgeIDs))
			}
			return nil
		})
		require.NoError(t, err)

		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 1 {
			t.Fatalf("expected edge count of 1, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 1 {
			t.Fatalf("expected excluded edge count of 1, found %d", len(application.ExcludeEdgeIDs))
		}

		application2, err := dbAPI.GetApplication(ctx1, appID2)
		require.NoError(t, err)

		if len(application2.EdgeSelectors) != 0 {
			t.Fatalf("expected edge selectors count of 0, found %d", len(application2.EdgeSelectors))
		}
		if len(application2.EdgeIDs) != 3 {
			t.Fatalf("expected edge count of 3, found %d", len(application2.EdgeIDs))
		}
		if len(application2.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application2.ExcludeEdgeIDs))
		}
		// Update edge to a label which is not in the project
		edge.Labels = []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue2,
			},
		}
		_, err = dbAPI.UpdateEdge(ctx1, &edge, nil)
		require.NoError(t, err)

		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 0 {
			t.Fatalf("expected edge count of 0, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 1 {
			t.Fatalf("expected excluded edge count of 1, found %d", len(application.ExcludeEdgeIDs))
		}

		edge.Labels = []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		}
		_, err = dbAPI.UpdateEdge(ctx1, &edge, nil)
		require.NoError(t, err)

		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 1 {
			t.Fatalf("expected edge count of 1, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 1 {
			t.Fatalf("expected excluded edge count of 1, found %d", len(application.ExcludeEdgeIDs))
		}

		edge3.Labels = []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue2,
			},
		}
		_, err = dbAPI.UpdateEdge(ctx1, &edge3, nil)
		require.NoError(t, err)
		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 1 {
			t.Fatalf("expected edge count of 1, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application.ExcludeEdgeIDs))
		}

		edge3.Labels = []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		}
		_, err = dbAPI.UpdateEdge(ctx1, &edge3, nil)
		require.NoError(t, err)
		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 2 {
			t.Fatalf("expected edge count of 2, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application.ExcludeEdgeIDs))
		}

		project.EdgeSelectors = []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue2,
			},
		}
		_, err = dbAPI.UpdateProject(ctx1, &project, nil)
		require.NoError(t, err)
		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 0 {
			t.Fatalf("expected edge count of 0, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application.ExcludeEdgeIDs))
		}
		project.EdgeSelectors = []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		}
		_, err = dbAPI.UpdateProject(ctx1, &project, nil)
		require.NoError(t, err)
		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 1 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 2 {
			t.Fatalf("expected edge count of 2, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application.ExcludeEdgeIDs))
		}

		project2.EdgeIDs = []string{edgeID, edgeID2}
		_, err = dbAPI.UpdateProject(ctx1, &project2, nil)
		require.NoError(t, err)
		application2, err = dbAPI.GetApplication(ctx1, appID2)
		require.NoError(t, err)

		if len(application2.EdgeSelectors) != 0 {
			t.Fatalf("expected edge selectors count of 0, found %d", len(application2.EdgeSelectors))
		}
		if len(application2.EdgeIDs) != 2 {
			t.Fatalf("expected edge count of 3, found %d", len(application2.EdgeIDs))
		}
		if len(application2.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application2.ExcludeEdgeIDs))
		}

		project2.EdgeIDs = []string{edgeID, edgeID2, edgeID3}
		_, err = dbAPI.UpdateProject(ctx1, &project2, nil)
		require.NoError(t, err)

		application2.EdgeIDs = []string{edgeID, edgeID2, edgeID3}
		// This must override
		application2.ExcludeEdgeIDs = []string{edgeID}
		_, err = dbAPI.UpdateApplication(ctx1, &application2, func(ctx context.Context, doc interface{}) error {
			updatedApp, ok := doc.(*api.App)
			if !ok {
				t.Fatalf("unexpected type in callback %+v", doc)
			}
			if len(updatedApp.EdgeIDs) != 2 {
				t.Fatalf("expected edge count of 1, found %d", len(updatedApp.EdgeIDs))
			}
			if len(updatedApp.ExcludeEdgeIDs) != 1 {
				t.Fatalf("expected excluded edge count of 1, found %d", len(updatedApp.ExcludeEdgeIDs))
			}
			return nil
		})
		require.NoError(t, err)
		application2, err = dbAPI.GetApplication(ctx1, appID2)
		require.NoError(t, err)

		if len(application2.EdgeSelectors) != 0 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application2.EdgeSelectors))
		}
		if len(application2.EdgeIDs) != 2 {
			t.Fatalf("expected edge count of 2, found %d", len(application2.EdgeIDs))
		}
		if len(application2.ExcludeEdgeIDs) != 1 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application2.ExcludeEdgeIDs))
		}
		project.EdgeSelectorType = model.ProjectEdgeSelectorTypeExplicit
		project.EdgeSelectors = catInfos
		_, err = dbAPI.UpdateProject(ctx1, &project, nil)
		require.NoError(t, err)
		application, err = dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)

		if len(application.EdgeSelectors) != 0 {
			t.Fatalf("expected edge selectors count of 1, found %d", len(application.EdgeSelectors))
		}
		if len(application.EdgeIDs) != 0 {
			t.Fatalf("expected edge count of 0, found %d", len(application.EdgeIDs))
		}
		if len(application.ExcludeEdgeIDs) != 0 {
			t.Fatalf("expected excluded edge count of 0, found %d", len(application.ExcludeEdgeIDs))
		}
	})
}

func TestApplicationTemplateValidation(t *testing.T) {
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, []string{})
	projectID := project.ID
	context, _, _ := makeContext(tenantID, []string{projectID})
	defer func() {
		dbAPI.DeleteProject(context, projectID, nil)
		dbAPI.DeleteTenant(context, tenantID, nil)
	}()
	// create application
	appDesc := "test app"
	randomName := fmt.Sprintf("app-%v", time.Now().UTC().UnixNano())
	yamlData := `
apiVersion: apps/v1beta1
kind: Deployment
metadata:
	name: kafdrop
spec:
	replicas: 1
	template:
		metadata:
			labels:
				app: kafdrop
		spec:
			containers:
			- name: kafdrop
			  image: thomsch98/kafdrop
			  env:
			  - name: KAFKA_SERVER
			    value: "{{.Services.Kafka.Endpoint}}"
			  ports:
			  - containerPort: 9000`
	app := model.Application{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		ApplicationCore: model.ApplicationCore{
			Name:        randomName,
			Description: appDesc,
			ProjectID:   projectID,
		},
		YamlData: yamlData,
	}

	_, err := dbAPI.CreateApplication(context, &app, nil)
	require.Error(t, err, "Expect validation to fail")
}

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
	DATA_SOURCES_PATH     = "/v1/datasources"
	DATA_SOURCES_PATH_NEW = "/v1.0/datasources"
)

// create datasource
func createDataSource(netClient *http.Client, datasource *model.DataSource, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, DATA_SOURCES_PATH, *datasource, token)
	if err == nil {
		datasource.ID = resp.ID
	}
	return resp, reqID, err
}

// create datasource
func createDataSourceNew(netClient *http.Client, datasourceV2 *model.DataSourceV2, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, DATA_SOURCES_PATH_NEW, *datasourceV2, token)
	if err == nil {
		datasourceV2.ID = resp.ID
	}
	return resp, reqID, err
}

// update datasource
func updateDataSource(netClient *http.Client, datasourceID string, datasource model.DataSource, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", DATA_SOURCES_PATH, datasourceID), datasource, token)
}

// update datasource
func updateDataSourceNew(netClient *http.Client, datasourceID string, datasourceV2 model.DataSourceV2, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", DATA_SOURCES_PATH_NEW, datasourceID), datasourceV2, token)
}

// get datasources
func getDataSources(netClient *http.Client, token string) ([]model.DataSource, error) {
	datasources := []model.DataSource{}
	err := doGet(netClient, DATA_SOURCES_PATH, token, &datasources)
	return datasources, err
}
func getDataSourcesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.DataSourceListPayload, error) {
	response := model.DataSourceListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", DATA_SOURCES_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}
func getDataSourcesForProject(netClient *http.Client, projectID string, token string) ([]model.DataSource, error) {
	datasources := []model.DataSource{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/datasources", token, &datasources)
	return datasources, err
}
func getDataSourcesForProjectNew(netClient *http.Client, projectID string, token string, pageIndex int, pageSize int) (model.DataSourceListPayload, error) {
	response := model.DataSourceListPayload{}
	path := fmt.Sprintf("%s/%s/datasources?pageIndex=%d&pageSize=%d&orderBy=id", PROJECTS_PATH_NEW, projectID, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

// delete datasource
func deleteDataSource(netClient *http.Client, datasourceID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, DATA_SOURCES_PATH, datasourceID, token)
}

// get datasource by id
func getDataSourceByID(netClient *http.Client, datasourceID string, token string) (model.DataSource, error) {
	datasource := model.DataSource{}
	err := doGet(netClient, DATA_SOURCES_PATH+"/"+datasourceID, token, &datasource)
	return datasource, err
}

func setupTenantAndUser(t *testing.T) (netClient *http.Client,
	dbAPI api.ObjectModelAPI, tenantID string, user model.User,
	userID string) {
	netClient = &http.Client{
		Timeout: time.Minute,
	}
	var err error
	dbAPI, err = api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID = tenant.ID
	user = apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	userID = user.ID
	return
}

// setupObjectsForDataSourceCreation will login and create a category
// and an edge needed for creating a data source.
func setupObjectsForDataSourceCreation(t *testing.T, netClient *http.Client, tenantID string, user model.User) (token string,
	categoryID string, edgeID string) {
	// Get token needed for authenticating API calls.
	token = loginUser(t, netClient, user)

	// Create a category
	category := model.Category{
		Name:    "test-cat",
		Purpose: "",
		Values:  []string{"v1", "v2"},
	}
	_, _, err := createCategory(netClient, &category, token)
	require.NoError(t, err)
	categoryID = category.ID

	// Create an edge
	edge, _, err := createEdgeForTenant(netClient, tenantID, token)
	require.NoError(t, err)
	edgeID = edge.ID
	return
}

func teardown(dbAPI api.ObjectModelAPI, tenantID string, userID string) {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	dbAPI.DeleteUser(ctx, userID, nil)
	dbAPI.DeleteTenant(ctx, tenantID, nil)
	dbAPI.Close()
}

func deleteTestObjects(t *testing.T, netClient *http.Client, token string, categoryID string,
	edgeID string, datasourceIDs []string) {
	// Delete DataSource(s)
	for _, datasourceID := range datasourceIDs {
		resp, _, err := deleteDataSource(netClient, datasourceID, token)
		require.NoError(t, err)
		if resp.ID != datasourceID {
			t.Fatal("delete datasource id mismatch")
		}
	}

	// Delete Edge
	resp, _, err := deleteEdge(netClient, edgeID, token)
	require.NoError(t, err)
	if resp.ID != edgeID {
		t.Fatal("delete edge id mismatch")
	}

	// Delete Category
	resp, _, err = deleteCategory(netClient, categoryID, token)
	require.NoError(t, err)
	if resp.ID != categoryID {
		t.Fatal("delete category id mismatch")
	}
}

func createDataSourceObj(t *testing.T, netClient *http.Client, token string, categoryID string, edgeID string) model.DataSource {
	datasource := model.DataSource{
		EdgeBaseModel: model.EdgeBaseModel{
			EdgeID: edgeID,
		},
		DataSourceCore: model.DataSourceCore{
			Name:       fmt.Sprintf("datasource-name-%s", base.GetUUID()),
			Type:       "Sensor",
			Connection: "Secure",
			Selectors: []model.DataSourceFieldSelector{
				{
					CategoryInfo: model.CategoryInfo{
						ID:    categoryID,
						Value: "v1",
					},
					Scope: []string{
						"__ALL__",
					},
				},
			},
			Protocol: "MQTT",
			AuthType: "CERTIFICATE",
		},
		Fields: []model.DataSourceFieldInfo{
			{
				DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
					Name:      "field-name-1",
					FieldType: "field-type-1",
				},
				MQTTTopic: "mqtt-topic-1",
			},
		},
		SensorModel: "Model 3",
	}
	_, _, err := createDataSource(netClient, &datasource, token)
	require.NoError(t, err)
	return datasource
}

func createDataSourceV2Obj(t *testing.T, netClient *http.Client, token string, categoryID string, edgeID string) model.DataSourceV2 {
	datasourceV2 := model.DataSourceV2{
		EdgeBaseModel: model.EdgeBaseModel{
			EdgeID: edgeID,
		},
		DataSourceCoreV2: model.DataSourceCoreV2{
			Name: "datasource-name",
			Type: "Sensor",
			Selectors: []model.DataSourceFieldSelector{
				{
					CategoryInfo: model.CategoryInfo{
						ID:    categoryID,
						Value: "v1",
					},
					Scope: []string{
						"__ALL__",
					},
				},
			},
			Protocol: "MQTT",
			AuthType: "CERTIFICATE",
		},
		FieldsV2: []model.DataSourceFieldInfoV2{
			{
				Name:  "field-name-1",
				Topic: "mqtt-topic-1",
			},
		},
	}
	_, _, err := createDataSourceNew(netClient, &datasourceV2, token)
	require.NoError(t, err)
	return datasourceV2
}

// Cleanup datasource obj and compare with fromDataSourceV2 obj. If they are not equal, fatal!
// NOTE: fromDataSourceV2 object is model.DataSource obj derived from model.DataSourceV2 obj
func cleanupDataSourceObjAndCompare(t *testing.T, datasource *model.DataSource, fromDataSourceV2 *model.DataSource) {
	// Modify following fields in datasources retrieved using /v1/ API:
	// 1. "connection": Removed from /v1.0/ API and hardcoded to 'secure'
	//                  in the backend.
	// 2. "fieldType": Removed from /v1.0/ API and hardcoded to empty string
	//                 in the backend.
	// 3. "sensorModel": Removed from /v1.0/ API and hardcoded to empty string
	//                   in the backend.
	datasource.SensorModel = ""
	datasource.Connection = "Secure"
	for i := range datasource.Fields {
		datasource.Fields[i].FieldType = ""
	}
	if !reflect.DeepEqual(*datasource, *fromDataSourceV2) {
		t.Fatalf("Expect datasources equal, but %+v != %+v", *datasource, *fromDataSourceV2)
	}
}

// Test to create, retrieve and update data sources using /v1/ APIs
func TestDataSource(t *testing.T) {
	t.Log("running TestDataSource test")

	var token, categoryID, edgeID string
	// Setup tenant and user for the test.
	netClient, dbAPI, tenantID, user, userID := setupTenantAndUser(t)

	// Teardown
	defer teardown(dbAPI, tenantID, userID)

	t.Run("Test DataSource", func(t *testing.T) {
		var err error
		token, categoryID, edgeID = setupObjectsForDataSourceCreation(t, netClient, tenantID, user)
		datasource := createDataSourceObj(t, netClient, token, categoryID, edgeID)

		datasources, err := getDataSources(netClient, token)
		require.NoError(t, err)
		t.Logf("got datasources: %+v", datasources)
		if len(datasources) != 1 {
			t.Fatalf("Expected data sources count to be 1, got %d", len(datasources))
		}

		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, []string{edgeID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)

		projects, err := getProjects(netClient, token)
		require.NoError(t, err)
		if len(projects) != 1 {
			t.Fatalf("Expected projects count 1, got %d", len(projects))
		}

		dss, err := getDataSourcesForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if len(dss) != 1 {
			t.Fatalf("Expected data sources count to be 1, got %d", len(datasources))
		}

		if !reflect.DeepEqual(dss[0], datasources[0]) {
			t.Fatalf("Expected data source to equal, but %+v != %+v", dss[0], datasources[0])
		}

		datasourceID := datasource.ID
		datasource.ID = ""
		datasource.Name = fmt.Sprintf("%s-updated", datasource.Name)
		ur, _, err := updateDataSource(netClient, datasourceID, datasource, token)
		require.NoError(t, err)
		if ur.ID != datasourceID {
			t.Fatal("Expected update data source id to match")
		}

		// Cleanup objects after success.
		deleteTestObjects(t, netClient, token, categoryID, edgeID, []string{datasourceID})
	})
}

// Test to create, retrieve and update data sources using /v1.0/ APIs
func TestDataSourceV2(t *testing.T) {
	t.Log("running TestDataSourceV2 test")

	var token, categoryID, edgeID string
	// Setup tenant and user for the test.
	netClient, dbAPI, tenantID, user, userID := setupTenantAndUser(t)

	// Teardown
	defer teardown(dbAPI, tenantID, userID)

	t.Run("Test DataSourceV2", func(t *testing.T) {
		var err error
		token, categoryID, edgeID = setupObjectsForDataSourceCreation(t, netClient, tenantID, user)
		datasourceV2 := createDataSourceV2Obj(t, netClient, token, categoryID, edgeID)

		datasources, err := getDataSourcesNew(netClient, token, 0, 10)
		require.NoError(t, err)
		t.Logf("got datasources: %+v", datasources)
		if len(datasources.DataSourceListV2) != 1 {
			t.Fatalf("Expected data sources count to be 1, got %d", len(datasources.DataSourceListV2))
		}

		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, []string{edgeID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)

		projects, err := getProjects(netClient, token)
		require.NoError(t, err)
		if len(projects) != 1 {
			t.Fatalf("Expected projects count 1, got %d", len(projects))
		}

		dss, err := getDataSourcesForProjectNew(netClient, project.ID, token, 0, 10)
		require.NoError(t, err)
		if len(dss.DataSourceListV2) != 1 {
			t.Fatalf("Expected data sources count to be 1, got %d", len(datasources.DataSourceListV2))
		}

		if !reflect.DeepEqual(dss.DataSourceListV2[0], datasources.DataSourceListV2[0]) {
			t.Fatalf("Expected data source to equal, but %+v != %+v", dss.DataSourceListV2[0], datasources.DataSourceListV2[0])
		}

		datasourceID := datasourceV2.ID
		datasourceV2.ID = ""
		datasourceV2.Name = fmt.Sprintf("%s-updated", datasourceV2.Name)
		ur, _, err := updateDataSourceNew(netClient, datasourceID, datasourceV2, token)
		require.NoError(t, err)
		if ur.ID != datasourceID {
			t.Fatal("Expected update data source id to match")
		}

		// Cleanup objects after success.
		deleteTestObjects(t, netClient, token, categoryID, edgeID, []string{datasourceID})
	})
}

// Test to create data sources using /v1/ API and retrieve using /v1.0/ API
func TestDataSourcePaging(t *testing.T) {
	t.Log("running TestDataSourcePaging test")

	var token, categoryID, edgeID string
	// Setup tenant and user for the test.
	netClient, dbAPI, tenantID, user, userID := setupTenantAndUser(t)
	rand1 := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Teardown
	defer teardown(dbAPI, tenantID, userID)

	t.Run("Test DataSource Paging", func(t *testing.T) {
		var err error
		token, categoryID, edgeID = setupObjectsForDataSourceCreation(t, netClient, tenantID, user)

		// randomly create some data sources
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d data sources...", n)
		for i := 0; i < n; i++ {
			createDataSourceObj(t, netClient, token, categoryID, edgeID)
		}

		datasources, err := getDataSources(netClient, token)
		require.NoError(t, err)
		t.Logf("got datasources: %+v", datasources)
		if len(datasources) != n {
			t.Fatalf("Expected datasources count to be %d, but got %d", n, len(datasources))
		}
		sort.Sort(model.DataSourcesByID(datasources))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pDataSources := []model.DataSource{}
		nRemain := n
		t.Logf("fetch %d data sources using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nccs, err := getDataSourcesNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if nccs.PageIndex != i {
				t.Fatalf("Expected page index to be %d, but got %d", i, nccs.PageIndex)
			}
			if nccs.PageSize != pageSize {
				t.Fatalf("Expected page size to be %d, but got %d", pageSize, nccs.PageSize)
			}
			if nccs.TotalCount != n {
				t.Fatalf("Expected total count to be %d, but got %d", n, nccs.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(nccs.DataSourceListV2) != nexp {
				t.Fatalf("Expected result count to be %d, but got %d", nexp, len(nccs.DataSourceListV2))
			}
			nRemain -= pageSize
			for _, cc := range model.DataSourcesByIDV2(nccs.DataSourceListV2).FromV2() {
				pDataSources = append(pDataSources, cc)
			}
		}

		// verify paging api gives same result as old api
		for i := range pDataSources {
			cleanupDataSourceObjAndCompare(t, &datasources[i], &pDataSources[i])
		}
		t.Log("GET datasources from paging api gives same result as old api")

		// Cleanup objects after success.
		var datasourceIDs []string
		for _, d := range datasources {
			datasourceIDs = append(datasourceIDs, d.ID)
		}
		deleteTestObjects(t, netClient, token, categoryID, edgeID, datasourceIDs)
	})
}

// Test to create data sources using /v1.0/ API and retrieve using /v1/ API
func TestDataSourceRetrieveAPIInterOp(t *testing.T) {
	t.Log("running TestDataSourceAPIRetrieveInterOp test")

	var token, categoryID, edgeID string
	// Setup tenant and user for the test.
	netClient, dbAPI, tenantID, user, userID := setupTenantAndUser(t)

	// Teardown
	defer teardown(dbAPI, tenantID, userID)

	t.Run("Test TestDataSourceAPIRetrieveInterOp", func(t *testing.T) {
		var err error
		token, categoryID, edgeID = setupObjectsForDataSourceCreation(t, netClient, tenantID, user)
		createDataSourceV2Obj(t, netClient, token, categoryID, edgeID)

		// Retrieve datasource using /v1.0/ API
		datasourcesV2, err := getDataSourcesNew(netClient, token, 0, 10)
		require.NoError(t, err)
		t.Logf("Got datasources using /v1.0/ API: %+v", datasourcesV2)
		if len(datasourcesV2.DataSourceListV2) != 1 {
			t.Fatalf("Expected data sources count to be 1, got %d", len(datasourcesV2.DataSourceListV2))
		}

		// Retrieve datasource using /v1/ API
		datasources, err := getDataSources(netClient, token)
		require.NoError(t, err)
		t.Logf("Got datasources using /v1/ API: %+v", datasources)
		if len(datasources) != 1 {
			t.Fatalf("Expected data sources count to be 1, got %d", len(datasources))
		}

		// Compare the two retrieved objects.
		fromDataSourceV2 := datasourcesV2.DataSourceListV2[0].FromV2()
		cleanupDataSourceObjAndCompare(t, &datasources[0], &fromDataSourceV2)

		// Cleanup objects after success.
		deleteTestObjects(t, netClient, token, categoryID, edgeID, []string{datasourcesV2.DataSourceListV2[0].ID})
	})
}

// Test to validate update API interop in the following cases:
// 1. Create a datasource using /v1.0/ API and update using /v1/
// 2. Create a datasource using /v1/ API and update using /v1.0/
func TestDataSourceUpdateAPIInterOp(t *testing.T) {
	t.Log("running TestDataSourceUpdateAPIInterOp test")

	var token, categoryID, edgeID string
	// Setup tenant and user for the test.
	netClient, dbAPI, tenantID, user, userID := setupTenantAndUser(t)

	// Teardown
	defer teardown(dbAPI, tenantID, userID)

	t.Run("Test TestDataSourceUpdateAPIInterOp", func(t *testing.T) {
		var err error
		token, categoryID, edgeID = setupObjectsForDataSourceCreation(t, netClient, tenantID, user)

		// Create datasource with /v1/ API.
		datasource := createDataSourceObj(t, netClient, token, categoryID, edgeID)
		// Update the datasource using /v1.0/ API.
		datasourceID := datasource.ID
		datasource.ID = ""
		datasource.Name = fmt.Sprintf("%s-updated", datasource.Name)
		urV2, _, err := updateDataSourceNew(netClient, datasourceID, datasource.ToV2(), token)
		require.NoError(t, err)
		if urV2.ID != datasourceID {
			t.Fatal("Expected update data source id to match")
		}

		// Create datasource with /v1.0/ API.
		datasourceV2 := createDataSourceV2Obj(t, netClient, token, categoryID, edgeID)
		// Update the datasource using /v1/ API.
		datasourceV2ID := datasourceV2.ID
		datasourceV2.ID = ""
		datasourceV2.Name = fmt.Sprintf("%s-updated", datasourceV2.Name)
		ur, _, err := updateDataSource(netClient, datasourceV2ID, datasourceV2.FromV2(), token)
		require.NoError(t, err)
		if ur.ID != datasourceV2ID {
			t.Fatal("Expected update data source id to match")
		}

		// Cleanup objects after success.
		deleteTestObjects(t, netClient, token, categoryID, edgeID, []string{datasourceID, datasourceV2ID})
	})
}

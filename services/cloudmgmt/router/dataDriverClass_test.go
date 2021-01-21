package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

const (
	DATA_DRIVER_CLASS_PATH = "/v1.0/datadriverclasses"
)

type dataDriverClassByID []model.DataDriverClass

func (a dataDriverClassByID) Len() int           { return len(a) }
func (a dataDriverClassByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a dataDriverClassByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// create data driver class
func createDataDriverClass(netClient *http.Client, ddClass *model.DataDriverClass, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, DATA_DRIVER_CLASS_PATH, *ddClass, token)
	if err == nil {
		ddClass.ID = resp.ID
	}
	return resp, reqID, err
}

// update data driver class
func updateDataDriverClass(netClient *http.Client, ddClass model.DataDriverClass, token string) (model.UpdateDocumentResponseV2, string, error) {
	path := fmt.Sprintf("%s/%s", DATA_DRIVER_CLASS_PATH, ddClass.ID)
	ddClass.ID = ""
	return updateEntityV2(netClient, path, ddClass, token)
}

// get data driver classess
func getDataDriverClasses(netClient *http.Client, token string) ([]model.DataDriverClass, error) {
	response := model.DataDriverClassListResponsePayload{}
	path := fmt.Sprintf("%s?orderBy=id", DATA_DRIVER_CLASS_PATH)
	err := doGet(netClient, path, token, &response)
	return response.ListOfDataDrivers, err
}
func getDataDriverClassesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.DataDriverClassListResponsePayload, error) {
	response := model.DataDriverClassListResponsePayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", DATA_DRIVER_CLASS_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getDataDriverClassByID(netClient *http.Client, ddClassID string, token string) (model.DataDriverClass, error) {
	dd := model.DataDriverClass{}
	err := doGet(netClient, DATA_DRIVER_CLASS_PATH+"/"+ddClassID, token, &dd)
	return dd, err
}

// delete data driver class
func deleteDataDriverClass(netClient *http.Client, ddClassID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, DATA_DRIVER_CLASS_PATH, ddClassID, token)
}

func makeDataDriverClass(name string) model.DataDriverClass {
	return model.DataDriverClass{
		BaseModel: model.BaseModel{
			ID: "ddc-" + funk.RandomString(20),
		},
		DataDriverClassCore: model.DataDriverClassCore{
			Name:              name,
			Description:       "Description",
			DataDriverVersion: "1.0",
			Type:              "SOURCE",
			YamlData:          "TEST YAML",
			StaticParameterSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"key": map[string]interface{}{
						"type":        "string",
						"default":     funk.RandomString(10),
						"description": "description 1",
					},
					"i": map[string]interface{}{
						"type":        "integer",
						"description": "description 2",
					},
				},
			},
			ConfigParameterSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"a": map[string]interface{}{
						"type": "string",
					},
				},
			},
			StreamParameterSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"b": map[string]interface{}{
						"type": "string",
					},
				},
			},
		},
	}
}

func TestDataDriverClass(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverClass test")

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

	t.Run("Test DataDriverClass", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		dd := makeDataDriverClass("name-1")
		_, _, err := createDataDriverClass(netClient, &dd, token)
		require.NoError(t, err)

		dds, err := getDataDriverClasses(netClient, token)
		require.NoError(t, err)
		t.Logf("got datadriverclasses: %+v", dds)
		if len(dds) != 1 {
			t.Fatalf("expect count of dat adriver classes to be 1, got %d", len(dds))
		}

		ddJ, err := getDataDriverClassByID(netClient, dd.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(dds[0], ddJ) {
			t.Fatalf("expect data driver class equal, but %+v != %+v", dds[0], ddJ)
		}

		// update data driver class
		ddID := ddJ.ID
		ddJ.Name = fmt.Sprintf("%s-updated", ddJ.Name)
		ur, _, err := updateDataDriverClass(netClient, ddJ, token)
		require.NoError(t, err)
		if ur.ID != ddID {
			t.Fatal("expect update data driver class id to match")
		}

		// delete the data driver
		resp, _, err := deleteDataDriverClass(netClient, ddID, token)
		require.NoError(t, err)
		if resp.ID != ddID {
			t.Fatal("delete data driver class id mismatch")
		}
	})

	t.Run("Test DataDriverClass in use message", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		project := makeExplicitProject(tenantID, []string{}, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dd := makeDataDriverClass("name-1")
		_, _, err := createDataDriverClass(netClient, &dd, token)
		require.NoError(t, err)
		ddID := dd.ID

		ddi := makeDataDriverInstance("instance-1", ddID, projectID)
		_, _, err = createDataDriverInstance(netClient, &ddi, token)
		require.NoError(t, err)
		ddiID := ddi.ID

		// try to delete data driver class
		_, _, err = deleteDataDriverClass(netClient, ddID, token)
		require.Error(t, err)
		require.Containsf(t, err.Error(), "412 Precondition Failed", "should raise a 412 error")

		// delete the data driver instance
		resp, _, err := deleteDataDriverInstance(netClient, ddiID, token)
		require.NoError(t, err)
		if resp.ID != ddiID {
			t.Fatal("delete data driver instance id mismatch")
		}

		// delete the data driver class
		_, _, err = deleteDataDriverClass(netClient, ddID, token)
		require.NoError(t, err)

		_, _, err = deleteProject(netClient, projectID, token)
		require.NoError(t, err)

	})

	t.Run("Test DataDriverClass Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// randomly create some data driver classes
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d data driver classes...", n)
		for i := 0; i < n; i++ {
			dd := makeDataDriverClass(fmt.Sprintf("dd-class-name-%s", base.GetUUID()))
			_, _, err := createDataDriverClass(netClient, &dd, token)
			require.NoError(t, err)
		}

		ddsAll, err := getDataDriverClasses(netClient, token)
		require.NoError(t, err)
		if len(ddsAll) != n {
			t.Fatalf("expected data driver class count to be %d, but got %d", n, len(ddsAll))
		}
		sort.Sort(dataDriverClassByID(ddsAll))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pDataDriverClasses := map[string]model.DataDriverClass{}
		nRemain := n
		t.Logf("fetch %d data driver classes using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			dds, err := getDataDriverClassesNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if dds.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, dds.PageIndex)
			}
			if dds.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, dds.PageSize)
			}
			if dds.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, dds.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(dds.ListOfDataDrivers) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(dds.ListOfDataDrivers))
			}
			nRemain -= pageSize
			for _, cc := range dds.ListOfDataDrivers {
				pDataDriverClasses[cc.ID] = cc
			}
		}

		// verify paging api gives same result as non-paging api
		for _, cc := range ddsAll {
			if !reflect.DeepEqual(pDataDriverClasses[cc.ID], cc) {
				t.Fatalf("expect data driver class equal, but %+v != %+v", cc, pDataDriverClasses[cc.ID])
			}
		}
		t.Log("get data driver classes from paging api gives same result as old api")

		for _, dd := range ddsAll {
			resp, _, err := deleteDataDriverClass(netClient, dd.ID, token)
			require.NoError(t, err)
			if resp.ID != dd.ID {
				t.Fatal("delete data driver class id mismatch")
			}
		}
	})
}

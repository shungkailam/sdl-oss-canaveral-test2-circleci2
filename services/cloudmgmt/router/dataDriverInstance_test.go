package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"fmt"

	"context"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

const (
	DATA_DRIVER_INSTANCE_PATH = "/v1.0/datadriverinstances"
)

type dataDriverInstanceByID []model.DataDriverInstance

func (a dataDriverInstanceByID) Len() int           { return len(a) }
func (a dataDriverInstanceByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a dataDriverInstanceByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// create data driver instance
func createDataDriverInstance(netClient *http.Client, ddInstance *model.DataDriverInstance, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, DATA_DRIVER_INSTANCE_PATH, *ddInstance, token)
	if err == nil {
		ddInstance.ID = resp.ID
	}
	return resp, reqID, err
}

// update data driver instance
func updateDataDriverInstance(netClient *http.Client, ddInstance *model.DataDriverInstance, token string) (model.UpdateDocumentResponseV2, string, error) {
	path := fmt.Sprintf("%s/%s", DATA_DRIVER_INSTANCE_PATH, ddInstance.ID)
	ddInstance.ID = ""
	return updateEntityV2(netClient, path, ddInstance, token)
}

// get data driver instances
func getDataDriverInstances(netClient *http.Client, token string) ([]model.DataDriverInstance, error) {
	response := model.DataDriverInstanceListResponsePayload{}
	path := fmt.Sprintf("%s?orderBy=id", DATA_DRIVER_INSTANCE_PATH)
	err := doGet(netClient, path, token, &response)
	return response.ListOfDetaDriverInstances, err
}

func getDataDriverInstancesByClassId(netClient *http.Client, token string, classId string) ([]model.DataDriverInstance, error) {
	response := []model.DataDriverInstance{}
	path := fmt.Sprintf("%s/%s/instances", DATA_DRIVER_CLASS_PATH, classId)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getDataDriverInstancesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.DataDriverInstanceListResponsePayload, error) {
	response := model.DataDriverInstanceListResponsePayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", DATA_DRIVER_INSTANCE_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getDataDriverInstanceByID(netClient *http.Client, ddClassID string, token string) (model.DataDriverInstance, error) {
	dd := model.DataDriverInstance{}
	err := doGet(netClient, DATA_DRIVER_INSTANCE_PATH+"/"+ddClassID, token, &dd)
	return dd, err
}

// delete data driver instance
func deleteDataDriverInstance(netClient *http.Client, ddInstanceID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, DATA_DRIVER_INSTANCE_PATH, ddInstanceID, token)
}

func makeDataDriverInstance(name string, driverId string, projectId string) model.DataDriverInstance {
	return model.DataDriverInstance{
		BaseModel: model.BaseModel{
			ID: "ddc-" + funk.RandomString(20),
		},
		DataDriverInstanceCore: model.DataDriverInstanceCore{
			Name:              name,
			Description:       "Description",
			DataDriverClassID: driverId,
			ProjectID:         projectId,
			StaticParameters: map[string]interface{}{
				"key": "value",
				"i":   100,
			},
		},
	}
}

func TestDataDriverInstance(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverInstance test")

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

	t.Run("Test DataDriverInstance", func(t *testing.T) {
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

		ddis, err := getDataDriverInstances(netClient, token)
		require.NoError(t, err)
		t.Logf("got datadriverinstances: %+v", ddis)
		require.Len(t, ddis, 1)

		ddiJ, err := getDataDriverInstanceByID(netClient, ddi.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(ddis[0], ddiJ) {
			t.Fatalf("expect data driver instance equal, but %+v != %+v", ddis[0], ddiJ)
		}

		// update data driver instance
		ddiID := ddiJ.ID
		ddiJ.Name = fmt.Sprintf("%s-updated", ddiJ.Name)
		ur, _, err := updateDataDriverInstance(netClient, &ddiJ, token)
		require.NoError(t, err)
		if ur.ID != ddiID {
			t.Fatal("expect update data driver instance id to match")
		}

		// get by class id
		instances, err := getDataDriverInstancesByClassId(netClient, token, ddID)
		require.NoError(t, err)
		require.Len(t, instances, 1)

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

	t.Run("Test DataDriverInstance Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		project := makeExplicitProject(tenantID, []string{}, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dd := makeDataDriverClass("name-1")
		_, _, err := createDataDriverClass(netClient, &dd, token)
		require.NoError(t, err)
		ddID := dd.ID

		// randomly create some data driver instances
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d data driver instances...", n)
		for i := 0; i < n; i++ {
			dd := makeDataDriverInstance(fmt.Sprintf("dd-instance-name-%s", base.GetUUID()), ddID, projectID)
			_, _, err := createDataDriverInstance(netClient, &dd, token)
			require.NoError(t, err)
		}

		ddsiAll, err := getDataDriverInstances(netClient, token)
		require.NoError(t, err)
		if len(ddsiAll) != n {
			t.Fatalf("expected data driver instance count to be %d, but got %d", n, len(ddsiAll))
		}
		sort.Sort(dataDriverInstanceByID(ddsiAll))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pDataDriverInstances := map[string]model.DataDriverInstance{}
		nRemain := n
		t.Logf("fetch %d data driver instances using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			ddsi, err := getDataDriverInstancesNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if ddsi.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, ddsi.PageIndex)
			}
			if ddsi.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, ddsi.PageSize)
			}
			if ddsi.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, ddsi.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(ddsi.ListOfDetaDriverInstances) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(ddsi.ListOfDetaDriverInstances))
			}
			nRemain -= pageSize
			for _, cc := range ddsi.ListOfDetaDriverInstances {
				pDataDriverInstances[cc.ID] = cc
			}
		}

		// verify paging api gives same result as non-paging api
		for _, cc := range ddsiAll {
			if !reflect.DeepEqual(pDataDriverInstances[cc.ID], cc) {
				t.Fatalf("expect data driver instance equal, but %+v != %+v", cc, pDataDriverInstances[cc.ID])
			}
		}
		t.Log("get data driver instances from paging api gives same result as old api")

		for _, dd := range ddsiAll {
			resp, _, err := deleteDataDriverInstance(netClient, dd.ID, token)
			require.NoError(t, err)
			if resp.ID != dd.ID {
				t.Fatal("delete data driver instances id mismatch")
			}
		}

		// delete the data driver class
		_, _, err = deleteDataDriverClass(netClient, ddID, token)
		require.NoError(t, err)

		_, _, err = deleteProject(netClient, projectID, token)
		require.NoError(t, err)
	})
}

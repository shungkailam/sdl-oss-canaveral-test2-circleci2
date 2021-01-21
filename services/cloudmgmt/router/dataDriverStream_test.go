package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"fmt"

	"context"
	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

const (
	DATA_DRIVER_STREAM_PATH = "/v1.0/datadriverstreams"
)

// create data driver stream
func createDataDriverStream(netClient *http.Client, ddStream *model.DataDriverStream, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, DATA_DRIVER_STREAM_PATH, *ddStream, token)
	if err == nil {
		ddStream.ID = resp.ID
	}
	return resp, reqID, err
}

// update data driver stream
func updateDataDriverStream(netClient *http.Client, ddStream *model.DataDriverStream, token string) (model.UpdateDocumentResponseV2, string, error) {
	path := fmt.Sprintf("%s/%s", DATA_DRIVER_STREAM_PATH, ddStream.ID)
	ddStream.ID = ""
	return updateEntityV2(netClient, path, ddStream, token)
}

// get data driver stream
func getDataDriverStreamsByInstanceId(netClient *http.Client, id string, token string) ([]model.DataDriverStream, error) {
	response := model.DataDriverStreamListResponsePayload{}
	path := fmt.Sprintf("%s/%s/streams", DATA_DRIVER_INSTANCE_PATH, id)
	err := doGet(netClient, path, token, &response)
	return response.ListOfDataDriverStreams, err
}

func getDataDriverStreamByID(netClient *http.Client, id string, token string) (model.DataDriverStream, error) {
	dd := model.DataDriverStream{}
	path := fmt.Sprintf("%s/%s", DATA_DRIVER_STREAM_PATH, id)
	err := doGet(netClient, path, token, &dd)
	return dd, err
}

// delete data driver stream
func deleteDataDriverStream(netClient *http.Client, ddInstanceID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, DATA_DRIVER_STREAM_PATH, ddInstanceID, token)
}

func makeDataDriverStream(name string, ddInstanceID string, category *model.CategoryInfo) model.DataDriverStream {
	return model.DataDriverStream{
		BaseModel: model.BaseModel{
			ID: "ddcfg-" + funk.RandomString(20),
		},
		Name:                 name,
		Description:          "description-" + funk.RandomString(10),
		DataDriverInstanceID: ddInstanceID,
		Direction:            model.DataDriverStreamSource,
		Stream:               map[string]interface{}{"b": funk.RandomString(10)},
		ServiceDomainBinding: model.ServiceDomainBinding{
			ServiceDomainIDs:        nil,
			ExcludeServiceDomainIDs: nil,
			ServiceDomainSelectors:  []model.CategoryInfo{*category},
		},
		Labels: []model.CategoryInfo{*category},
	}
}

func TestDataDriverStream(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverStream test")

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

	t.Run("Test TestDataDriverStream", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		category := model.Category{
			Name:    "test-cat",
			Purpose: "",
			Values:  []string{"v1", "v2"},
		}
		_, _, err := createCategory(netClient, &category, token)
		require.NoError(t, err)
		categoryInfo := model.CategoryInfo{
			ID:    category.ID,
			Value: "v1",
		}

		project := makeCategoryProject(tenantID, []string{}, nil, []string{user.ID}, []model.CategoryInfo{categoryInfo})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dd := makeDataDriverClass("name-1")
		_, _, err = createDataDriverClass(netClient, &dd, token)
		require.NoError(t, err)

		ddi := makeDataDriverInstance("instance-1", dd.ID, projectID)
		_, _, err = createDataDriverInstance(netClient, &ddi, token)
		require.NoError(t, err)

		ddstr := makeDataDriverStream("stream-1", ddi.ID, &categoryInfo)
		_, _, err = createDataDriverStream(netClient, &ddstr, token)
		require.NoError(t, err)
		ddstrID := ddstr.ID

		// find data driver streams
		found, err := getDataDriverStreamByID(netClient, ddstrID, token)
		require.NoError(t, err)
		require.Equal(t, found.ID, ddstrID)

		streams, err := getDataDriverStreamsByInstanceId(netClient, ddi.ID, token)
		require.NoError(t, err)
		require.Len(t, streams, 1)
		require.Equal(t, streams[0].ID, ddstrID)

		// update data driver stream
		ddstr.Name = fmt.Sprintf("%s-updated", ddstr.Name)
		ur, _, err := updateDataDriverStream(netClient, &ddstr, token)
		require.NoError(t, err)
		if ur.ID != ddstrID {
			t.Fatal("expect update data driver stream id to match")
		}

		// delete the data driver stream
		resp, _, err := deleteDataDriverStream(netClient, ddstrID, token)
		require.NoError(t, err)
		if resp.ID != ddstrID {
			t.Fatal("delete data driver stream id mismatch")
		}

		// delete the data driver instance
		_, _, err = deleteDataDriverInstance(netClient, ddi.ID, token)
		require.NoError(t, err)

		// delete the data driver class
		_, _, err = deleteDataDriverClass(netClient, dd.ID, token)
		require.NoError(t, err)

		_, _, err = deleteProject(netClient, projectID, token)
		require.NoError(t, err)

		_, _, err = deleteCategory(netClient, category.ID, token)
		require.NoError(t, err)
	})
}

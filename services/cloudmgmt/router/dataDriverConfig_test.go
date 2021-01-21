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
	DATA_DRIVER_CONFIG_PATH = "/v1.0/datadriverconfigs"
)

// create data driver config
func createDataDriverConfig(netClient *http.Client, ddConfig *model.DataDriverConfig, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, DATA_DRIVER_CONFIG_PATH, *ddConfig, token)
	if err == nil {
		ddConfig.ID = resp.ID
	}
	return resp, reqID, err
}

// update data driver config
func updateDataDriverConfig(netClient *http.Client, ddConfig *model.DataDriverConfig, token string) (model.UpdateDocumentResponseV2, string, error) {
	path := fmt.Sprintf("%s/%s", DATA_DRIVER_CONFIG_PATH, ddConfig.ID)
	ddConfig.ID = ""
	return updateEntityV2(netClient, path, ddConfig, token)
}

// get data driver config
func getDataDriverConfigsByInstanceId(netClient *http.Client, id string, token string) ([]model.DataDriverConfig, error) {
	response := model.DataDriverConfigListResponsePayload{}
	path := fmt.Sprintf("%s/%s/configs", DATA_DRIVER_INSTANCE_PATH, id)
	err := doGet(netClient, path, token, &response)
	return response.ListOfDataDriverConfigs, err
}

func getDataDriverConfigByID(netClient *http.Client, id string, token string) (model.DataDriverConfig, error) {
	dd := model.DataDriverConfig{}
	path := fmt.Sprintf("%s/%s", DATA_DRIVER_CONFIG_PATH, id)
	err := doGet(netClient, path, token, &dd)
	return dd, err
}

// delete data driver config
func deleteDataDriverConfig(netClient *http.Client, ddInstanceID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, DATA_DRIVER_CONFIG_PATH, ddInstanceID, token)
}

func makeDataDriverConfig(name string, ddInstanceID string, category *model.CategoryInfo) model.DataDriverConfig {
	return model.DataDriverConfig{
		BaseModel: model.BaseModel{
			ID: "ddcfg-" + funk.RandomString(20),
		},
		Name:                 name,
		Description:          "description-" + funk.RandomString(10),
		DataDriverInstanceID: ddInstanceID,
		Parameters:           map[string]interface{}{"a": funk.RandomString(10)},
		ServiceDomainBinding: model.ServiceDomainBinding{
			ServiceDomainIDs:        nil,
			ExcludeServiceDomainIDs: nil,
			ServiceDomainSelectors:  []model.CategoryInfo{*category},
		},
	}
}

func TestDataDriverConfig(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverConfig test")

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

	t.Run("Test TestDataDriverConfig", func(t *testing.T) {
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

		ddcfg := makeDataDriverConfig("config-1", ddi.ID, &categoryInfo)
		_, _, err = createDataDriverConfig(netClient, &ddcfg, token)
		require.NoError(t, err)
		ddcfgID := ddcfg.ID

		// find data driver configs
		found, err := getDataDriverConfigByID(netClient, ddcfgID, token)
		require.NoError(t, err)
		require.Equal(t, found.ID, ddcfg.ID)

		configs, err := getDataDriverConfigsByInstanceId(netClient, ddi.ID, token)
		require.NoError(t, err)
		require.Len(t, configs, 1)
		require.Equal(t, configs[0].ID, ddcfg.ID)

		// update data driver config
		ddcfg.Name = fmt.Sprintf("%s-updated", ddcfg.Name)
		ur, _, err := updateDataDriverConfig(netClient, &ddcfg, token)
		require.NoError(t, err)
		if ur.ID != ddcfgID {
			t.Fatal("expect update data driver config id to match")
		}

		// delete the data driver config
		resp, _, err := deleteDataDriverConfig(netClient, ddcfgID, token)
		require.NoError(t, err)
		if resp.ID != ddcfgID {
			t.Fatal("delete data driver config id mismatch")
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

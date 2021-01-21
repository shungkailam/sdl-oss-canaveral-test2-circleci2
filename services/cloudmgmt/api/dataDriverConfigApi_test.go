package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/api/testtool"
	"cloudservices/common/base"
	"cloudservices/common/model"

	"context"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

func makeDataDriverConfig(name string, tenantID string, ddInstanceID string, edgeID string) model.DataDriverConfig {
	return model.DataDriverConfig{
		BaseModel: model.BaseModel{
			ID:       "ddc-id-" + funk.RandomString(10),
			TenantID: tenantID,
		},
		Name:                 name,
		Description:          "description-" + funk.RandomString(10),
		DataDriverInstanceID: ddInstanceID,
		Parameters:           map[string]interface{}{"anotherTest": funk.RandomString(10)},
		ServiceDomainBinding: model.ServiceDomainBinding{
			ServiceDomainIDs:        []string{edgeID},
			ExcludeServiceDomainIDs: nil,
			ServiceDomainSelectors:  nil,
		},
	}
}

func createDataDriverConfig(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, dataDriverInstanceID string, projectID string) model.DataDriverConfig {
	projRoles := []model.ProjectRole{
		{
			ProjectID: projectID,
			Role:      model.ProjectRoleAdmin,
		},
	}
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects":    projRoles,
			"email":       "any@email.com",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	dataDriverInstance := makeDataDriverConfig("ddconf-"+funk.RandomString(10), tenantID, dataDriverInstanceID, projectID)
	dataDriverInstance.TenantID = tenantID

	resp, err := dbAPI.CreateDataDriverInstance(ctx, &dataDriverInstance, nil)
	require.NoError(t, err)
	t.Logf("create data driver config successful, %s", resp)
	createResp := resp.(model.CreateDocumentResponseV2)
	dataDriverInstance, err = dbAPI.GetDataDriverConfig(ctx, createResp.ID)
	require.NoError(t, err)
	return dataDriverInstance
}

func TestDataDriverConfig(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverConfig test")

	// Setup
	dbAPI := newObjectModelAPI(t)

	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID

	impostorTenant := createTenant(t, dbAPI, "test impostor tenant")
	impostorTenantID := impostorTenant.ID

	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	edge2 := createEdge(t, dbAPI, tenantID)
	edge2ID := edge2.ID

	anotherEdge := createEdge(t, dbAPI, tenantID)
	anotherEdgeID := anotherEdge.ID

	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID, edge2ID})
	projectID := project.ID

	anotherProject := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{anotherEdgeID})
	anotherProjectID := anotherProject.ID

	impostorProject := createExplicitProjectCommon(t, dbAPI, impostorTenantID, []string{}, []string{}, []string{}, []string{})
	impostorProjectID := impostorProject.ID

	adminContext, _, _ := makeContext(tenantID, []string{projectID})

	// create data driver class 1
	dataDriverClass := createDataDriverClass(t, dbAPI, tenantID, "test data driver class")
	dataDriverClassID := dataDriverClass.ID

	dataDriverInstance := createDataDriverInstance(t, dbAPI, tenantID, dataDriverClassID, projectID)
	dataDriverInstanceID := dataDriverInstance.ID

	anotherDataDriverClass := createDataDriverClass(t, dbAPI, tenantID, "another test data driver")
	anotherDataDriverClassID := anotherDataDriverClass.ID

	anotherDataDriverInstance := createDataDriverInstance(t, dbAPI, tenantID, anotherDataDriverClassID, anotherProjectID)
	anotherDataDriverInstanceID := anotherDataDriverInstance.ID

	resolveInstanceId := func(pid string) string {
		if pid == projectID {
			return dataDriverInstanceID
		} else {
			return anotherDataDriverInstanceID
		}
	}

	apiTools := testtool.APITestTool("DataDriverConfig").
		ForTenant(tenantID, projectID, anotherProjectID).
		ForImpostor(impostorTenantID, impostorProjectID).
		PermissionsMatrix(testtool.ProjectLevelObject()).
		WithSelector(func(ctx context.Context, tenantId, projectId string) (interface{}, error) {
			configs, _, err := dbAPI.SelectDataDriverConfigsByInstanceId(ctx, resolveInstanceId(projectId), nil)
			return configs, err
		}).
		WithChecker(func(ctx context.Context, id string) (interface{}, error) {
			return dbAPI.GetDataDriverConfig(ctx, id)
		}).
		WithCreator(func(ctx context.Context, id, tenantId, projectId string) (interface{}, error) {
			obj := makeDataDriverConfig("test-"+funk.RandomString(10), tenantId, resolveInstanceId(projectId), edgeID)
			obj.ID = id
			return dbAPI.CreateDataDriverConfig(ctx, &obj, nil)
		}).
		WithUpdater(func(ctx context.Context, id, tenantId, projectId string) (interface{}, error) {
			obj := makeDataDriverConfig("test-"+funk.RandomString(10), tenantId, resolveInstanceId(projectId), edgeID)
			obj.ID = id
			return dbAPI.UpdateDataDriverConfig(ctx, &obj, nil)
		}).
		WithDeleter(func(ctx context.Context, id string) (interface{}, error) {
			return dbAPI.DeleteDataDriverConfig(ctx, id, nil)
		})

	// Teardown
	defer func() {
		dbAPI.DeleteDataDriverInstance(adminContext, anotherDataDriverInstanceID, nil)
		dbAPI.DeleteDataDriverClass(adminContext, anotherDataDriverClassID, nil)
		dbAPI.DeleteDataDriverInstance(adminContext, dataDriverInstanceID, nil)
		dbAPI.DeleteDataDriverClass(adminContext, dataDriverClassID, nil)
		dbAPI.DeleteProject(adminContext, projectID, nil)
		dbAPI.DeleteEdge(adminContext, edgeID, nil)
		dbAPI.DeleteTenant(adminContext, tenantID, nil)

		dbAPI.Close()
	}()

	t.Run("Test data driver config workflow", func(t *testing.T) {
		// initial search should be empty
		dds, count, err := dbAPI.SelectDataDriverConfigsByInstanceId(adminContext, dataDriverInstanceID, nil)
		require.NoError(t, err)
		require.Equal(t, count, 0)
		require.Empty(t, dds, "Error during initial find")

		// create data driver config
		obj1 := makeDataDriverConfig("test-1", tenantID, dataDriverInstanceID, edgeID)
		dd, err := dbAPI.CreateDataDriverConfig(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddsId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddsId, "Data driver config id not found")

		// modify
		obj2 := makeDataDriverConfig("test-2", tenantID, dataDriverInstanceID, edgeID)
		obj2.ID = ddsId
		updated, err := dbAPI.UpdateDataDriverConfig(adminContext, &obj2, nil)
		require.NoError(t, err)
		require.NotNil(t, updated)

		updatedID := updated.(model.UpdateDocumentResponseV2).ID
		require.Equal(t, ddsId, updatedID)

		// find newly created
		dds2, count, err := dbAPI.SelectDataDriverConfigsByInstanceId(adminContext, dataDriverInstanceID, nil)
		require.NoError(t, err)
		require.Equal(t, count, 1)
		require.Len(t, dds2, 1, "Error during find")

		// find by name
		dds2, count, err = dbAPI.SelectDataDriverConfigsByInstanceId(adminContext, dataDriverInstanceID, &model.EntitiesQueryParam{Filter: "name = 'test-2'"})
		require.NoError(t, err)
		require.Equal(t, count, 1)
		require.Len(t, dds2, 1, "Error during find")

		// can not find by incorrect name
		dds2, count, err = dbAPI.SelectDataDriverConfigsByInstanceId(adminContext, dataDriverInstanceID, &model.EntitiesQueryParam{Filter: "name = 'test-0'"})
		require.NoError(t, err)
		require.Equal(t, count, 0)
		require.Empty(t, dds2)

		// get by id
		dd, err = dbAPI.GetDataDriverConfig(adminContext, ddsId)
		require.NoError(t, err)
		require.NotNil(t, dd, "Failed to get by id")

		// delete as admin
		rsp, err := dbAPI.DeleteDataDriverConfig(adminContext, ddsId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		// get all again & it should be empty
		dds, count, err = dbAPI.SelectDataDriverConfigsByInstanceId(adminContext, dataDriverInstanceID, nil)
		require.NoError(t, err)
		require.Equal(t, count, 0)
		require.Empty(t, dds, "Error during last find")
	})

	t.Run("Test instance delete with existing config", func(t *testing.T) {
		// create data driver class
		ddc := makeDataDriverClass("test data driver", tenantID)
		_, err := dbAPI.CreateDataDriverClass(adminContext, &ddc, nil)
		require.NoError(t, err)
		ddcID := ddc.ID

		// create data driver instance
		obj1 := makeDataDriverInstance("test-origin", tenantID, ddcID, projectID)
		ddi, err := dbAPI.CreateDataDriverInstance(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, ddi)
		ddiID := obj1.ID

		// create data driver config
		obj2 := makeDataDriverConfig("test-1", tenantID, ddiID, edgeID)
		dds, err := dbAPI.CreateDataDriverConfig(adminContext, &obj2, nil)
		require.NoError(t, err)
		require.NotNil(t, dds)
		ddsID := obj2.ID

		// try to delete data driver instance
		_, err = dbAPI.DeleteDataDriverInstance(adminContext, ddiID, nil)
		require.Error(t, err)

		// try to delete data driver config
		_, err = dbAPI.DeleteDataDriverConfig(adminContext, ddsID, nil)
		require.NoError(t, err)

		// try to delete data driver instance
		_, err = dbAPI.DeleteDataDriverInstance(adminContext, ddiID, nil)
		require.NoError(t, err)

		// try to delete data driver class
		_, err = dbAPI.DeleteDataDriverClass(adminContext, ddcID, nil)
		require.NoError(t, err)
	})

	t.Run("Test data driver config update fields", func(t *testing.T) {
		// create data driver config
		obj1 := makeDataDriverConfig("test-origin", tenantID, dataDriverInstanceID, edgeID)
		dds, err := dbAPI.CreateDataDriverConfig(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dds)

		ddsId := dds.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddsId, "Data driver config id not found")

		original, err := dbAPI.GetDataDriverConfig(adminContext, ddsId)
		require.NoError(t, err)
		require.NotNil(t, original)

		// modify
		obj2 := makeDataDriverConfig("test-edited", tenantID, anotherDataDriverInstanceID, anotherEdgeID)
		obj2.ID = ddsId
		obj2.ServiceDomainBinding = model.ServiceDomainBinding{
			ServiceDomainIDs: []string{edgeID, edge2ID, anotherEdgeID},
		}
		_, err = dbAPI.UpdateDataDriverConfig(adminContext, &obj2, nil)
		require.NoError(t, err)

		updated, err := dbAPI.GetDataDriverConfig(adminContext, ddsId)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// check values
		require.Equal(t, original.ID, updated.ID)
		require.Equal(t, original.TenantID, updated.TenantID)
		require.NotEqual(t, original.Name, updated.Name)
		require.NotEqual(t, original.Description, updated.Description)
		require.Equal(t, original.DataDriverInstanceID, updated.DataDriverInstanceID)
		require.NotEqual(t, original.Parameters, updated.Parameters)

		require.NotEqual(t, original.ServiceDomainBinding, updated.ServiceDomainBinding)
		require.Len(t, original.ServiceDomainBinding.ServiceDomainIDs, 1)
		require.Len(t, updated.ServiceDomainBinding.ServiceDomainIDs, 2)

		// can delete
		_, err = dbAPI.DeleteDataDriverConfig(adminContext, ddsId, nil)
		require.NoError(t, err)
	})

	t.Run("Test permissions on data driver stream", func(t *testing.T) {
		t.Run("Search", apiTools.SearchRBACTest())
		t.Run("Read", apiTools.ReadRBACTest())
		t.Run("Creation", apiTools.CreateRBACTest())
		t.Run("Update", apiTools.UpdateRBACTest())
		t.Run("Delete", apiTools.DeleteRBACTest())
	})

	t.Run("ID validity", apiTools.IdSanityTest())
}

func TestMappingDataDriverConfigDBO(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	schema := map[string]interface{}{
		"a": 1.0,
		"b": "2",
	}
	schemaText := types.JSONText(`{"a":1,"b":"2"}`)
	tests := []struct {
		name    string
		dds     model.DataDriverConfig
		want    api.DataDriverParamsDBO
		wantErr bool
	}{
		{
			name: "data driver config",
			dds: model.DataDriverConfig{
				BaseModel: model.BaseModel{
					ID:        "dds-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                 "name-1",
				Description:          "description-1",
				DataDriverInstanceID: "ddi-1",
				Parameters:           schema,
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edeg-1"},
				},
			},
			want: api.DataDriverParamsDBO{
				BaseModelDBO: model.BaseModelDBO{
					ID:        "dds-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                 "name-1",
				Description:          "description-1",
				DataDriverInstanceID: "ddi-1",
				Parameters:           &schemaText,
				Type:                 "dataDriverConfig",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.ToDataDriverParamsDBOFromConfig(&tt.dds)
			require.NoError(t, err)
			require.EqualValues(t, got, tt.want)

			back, err := api.FromDataDriverParamsDBOToConfig(&got, &tt.dds.ServiceDomainBinding)
			require.NoError(t, err)
			require.EqualValues(t, back, tt.dds)

			_, err = api.FromDataDriverParamsDBOToStream(&got, &tt.dds.ServiceDomainBinding, []model.CategoryInfo{})
			require.Error(t, err)
		})
	}
}

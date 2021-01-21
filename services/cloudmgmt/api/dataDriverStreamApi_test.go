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

func makeDataDriverStream(name, tenantID, ddInstanceID, edgeID, categoryID string) model.DataDriverStream {
	return model.DataDriverStream{
		BaseModel: model.BaseModel{
			ID:       "dds-id-" + funk.RandomString(10),
			TenantID: tenantID,
		},
		Name:                 name,
		Description:          "description-" + funk.RandomString(10),
		DataDriverInstanceID: ddInstanceID,
		Stream:               map[string]interface{}{"test": funk.RandomString(10)},
		Direction:            model.DataDriverStreamSource,
		ServiceDomainBinding: model.ServiceDomainBinding{
			ServiceDomainIDs:        []string{edgeID},
			ExcludeServiceDomainIDs: nil,
			ServiceDomainSelectors:  nil,
		},
		Labels: []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: "value-1",
			},
		},
	}
}

func createDataDriverStream(t *testing.T, dbAPI api.ObjectModelAPI, tenantID, dataDriverInstanceID, projectID, edgeID, categoryID string) model.DataDriverStream {
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
	dataDriverStream := makeDataDriverStream("ddstream-"+funk.RandomString(10), tenantID, dataDriverInstanceID, edgeID, categoryID)

	resp, err := dbAPI.CreateDataDriverStream(ctx, &dataDriverStream, nil)
	require.NoError(t, err)
	t.Logf("create data driver config successful, %s", resp)
	createResp := resp.(model.CreateDocumentResponseV2)
	dataDriverStream, err = dbAPI.GetDataDriverStream(ctx, createResp.ID)
	require.NoError(t, err)
	return dataDriverStream
}

func TestDataDriverStream(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverStream test")

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

	cat := createCategoryCommon(t, dbAPI, tenantID, "cat-1", []string{"value-1", "value-2"})
	catID := cat.ID

	resolveInstanceId := func(pid string) string {
		if pid == projectID {
			return dataDriverInstanceID
		} else {
			return anotherDataDriverInstanceID
		}
	}

	apiTools := testtool.APITestTool("DataDriverStream").
		ForTenant(tenantID, projectID, anotherProjectID).
		ForImpostor(impostorTenantID, impostorProjectID).
		PermissionsMatrix(testtool.ProjectLevelObject()).
		WithSelector(func(ctx context.Context, tenantId, projectId string) (interface{}, error) {
			streams, _, err := dbAPI.SelectDataDriverStreamsByInstanceId(ctx, resolveInstanceId(projectId), &model.EntitiesQueryParam{})
			return streams, err
		}).
		WithChecker(func(ctx context.Context, id string) (interface{}, error) {
			return dbAPI.GetDataDriverStream(ctx, id)
		}).
		WithCreator(func(ctx context.Context, id, tenantId, projectId string) (interface{}, error) {
			obj := makeDataDriverStream("test-"+funk.RandomString(10), tenantId, resolveInstanceId(projectId), edgeID, catID)
			obj.ID = id
			return dbAPI.CreateDataDriverStream(ctx, &obj, nil)
		}).
		WithUpdater(func(ctx context.Context, id, tenantId, projectId string) (interface{}, error) {
			obj := makeDataDriverStream("test-"+funk.RandomString(10), tenantId, resolveInstanceId(projectId), edgeID, catID)
			obj.ID = id
			return dbAPI.UpdateDataDriverStream(ctx, &obj, nil)
		}).
		WithDeleter(func(ctx context.Context, id string) (interface{}, error) {
			return dbAPI.DeleteDataDriverStream(ctx, id, nil)
		})

	// Teardown
	defer func() {
		dbAPI.DeleteCategory(adminContext, catID, nil)
		dbAPI.DeleteDataDriverInstance(adminContext, anotherDataDriverInstanceID, nil)
		dbAPI.DeleteDataDriverClass(adminContext, anotherDataDriverClassID, nil)
		dbAPI.DeleteDataDriverInstance(adminContext, dataDriverInstanceID, nil)
		dbAPI.DeleteDataDriverClass(adminContext, dataDriverClassID, nil)
		dbAPI.DeleteProject(adminContext, projectID, nil)
		dbAPI.DeleteEdge(adminContext, edgeID, nil)
		dbAPI.DeleteTenant(adminContext, tenantID, nil)

		dbAPI.Close()
	}()

	t.Run("Test data driver stream workflow", func(t *testing.T) {
		// initial search should be empty
		dds, count, err := dbAPI.SelectDataDriverStreamsByInstanceId(adminContext, dataDriverInstanceID, nil)
		require.NoError(t, err)
		require.Equal(t, count, 0)
		require.Empty(t, dds, "Error during initial find")

		// create data driver stream
		obj1 := makeDataDriverStream("test-1", tenantID, dataDriverInstanceID, edgeID, catID)
		dd, err := dbAPI.CreateDataDriverStream(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddsId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddsId, "Data driver stream id not found")

		// modify
		obj2 := makeDataDriverStream("test-2", tenantID, dataDriverInstanceID, edgeID, catID)
		obj2.ID = ddsId
		updated, err := dbAPI.UpdateDataDriverStream(adminContext, &obj2, nil)
		require.NoError(t, err)
		require.NotNil(t, updated)

		updatedID := updated.(model.UpdateDocumentResponseV2).ID
		require.Equal(t, ddsId, updatedID)

		// find newly created
		dds2, count, err := dbAPI.SelectDataDriverStreamsByInstanceId(adminContext, dataDriverInstanceID, nil)
		require.NoError(t, err)
		require.Equal(t, count, 1)
		require.Len(t, dds2, 1, "Error during find")

		// find by name
		dds2, count, err = dbAPI.SelectDataDriverStreamsByInstanceId(adminContext, dataDriverInstanceID, &model.EntitiesQueryParam{Filter: "name = 'test-2'"})
		require.NoError(t, err)
		require.Equal(t, count, 1)
		require.Len(t, dds2, 1, "Error during find")

		// can not find by incorrect name
		dds2, count, err = dbAPI.SelectDataDriverStreamsByInstanceId(adminContext, dataDriverInstanceID, &model.EntitiesQueryParam{Filter: "name = 'test-0'"})
		require.NoError(t, err)
		require.Equal(t, count, 0)
		require.Empty(t, dds2)

		// get by id
		dd, err = dbAPI.GetDataDriverStream(adminContext, ddsId)
		require.NoError(t, err)
		require.NotNil(t, dd, "Failed to get by id")

		// delete as admin
		rsp, err := dbAPI.DeleteDataDriverStream(adminContext, ddsId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		// get all again & it should be empty
		dds, count, err = dbAPI.SelectDataDriverStreamsByInstanceId(adminContext, dataDriverInstanceID, nil)
		require.NoError(t, err)
		require.Equal(t, count, 0)
		require.Empty(t, dds, "Error during last find")
	})

	t.Run("Test data driver stream sink with labels", func(t *testing.T) {
		// create data driver stream
		obj1 := makeDataDriverStream("test-3", tenantID, dataDriverInstanceID, edgeID, catID)
		obj1.Direction = model.DataDriverStreamSink
		_, err := dbAPI.CreateDataDriverStream(adminContext, &obj1, nil)
		require.Error(t, err)

		obj1.Labels = []model.CategoryInfo{}
		dd, err := dbAPI.CreateDataDriverStream(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddsId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddsId, "Data driver stream id not found")

		// delete as admin
		rsp, err := dbAPI.DeleteDataDriverStream(adminContext, ddsId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)
	})

	t.Run("Test data driver stream sink without labels", func(t *testing.T) {
		obj2 := makeDataDriverStream("test-4", tenantID, dataDriverInstanceID, edgeID, catID)
		obj2.Labels = []model.CategoryInfo{}
		_, err := dbAPI.CreateDataDriverStream(adminContext, &obj2, nil)
		require.Error(t, err)
	})

	t.Run("Test instance delete with existing stream", func(t *testing.T) {
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

		// create data driver stream
		obj2 := makeDataDriverStream("test-1", tenantID, ddiID, edgeID, catID)
		dds, err := dbAPI.CreateDataDriverStream(adminContext, &obj2, nil)
		require.NoError(t, err)
		require.NotNil(t, dds)
		ddsID := obj2.ID

		// try to delete data driver instance
		_, err = dbAPI.DeleteDataDriverInstance(adminContext, ddiID, nil)
		require.Error(t, err)

		// delete data driver stream
		_, err = dbAPI.DeleteDataDriverStream(adminContext, ddsID, nil)
		require.NoError(t, err)

		// delete data driver instance
		_, err = dbAPI.DeleteDataDriverInstance(adminContext, ddiID, nil)
		require.NoError(t, err)

		// delete data driver class
		_, err = dbAPI.DeleteDataDriverClass(adminContext, ddcID, nil)
		require.NoError(t, err)
	})

	t.Run("Test data driver stream update fields", func(t *testing.T) {
		// create data driver stream
		obj1 := makeDataDriverStream("test-origin", tenantID, dataDriverInstanceID, edgeID, catID)
		obj1.ServiceDomainBinding = model.ServiceDomainBinding{
			ServiceDomainIDs: []string{edgeID},
		}
		obj1.Labels = []model.CategoryInfo{{ID: catID, Value: "value-1"}}

		dds, err := dbAPI.CreateDataDriverStream(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dds)

		ddsId := dds.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddsId, "Data driver stream id not found")

		original, err := dbAPI.GetDataDriverStream(adminContext, ddsId)
		require.NoError(t, err)
		require.NotNil(t, original)

		// modify
		obj2 := makeDataDriverStream("test-edited", tenantID, anotherDataDriverInstanceID, anotherEdgeID, catID)
		obj2.ID = ddsId
		obj2.Direction = model.DataDriverStreamSink
		obj2.ServiceDomainBinding = model.ServiceDomainBinding{
			ServiceDomainIDs: []string{edgeID, edge2ID},
		}
		obj2.Labels = []model.CategoryInfo{{ID: catID, Value: "value-2"}}

		_, err = dbAPI.UpdateDataDriverStream(adminContext, &obj2, nil)
		require.NoError(t, err)

		updated, err := dbAPI.GetDataDriverStream(adminContext, ddsId)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// check values
		require.Equal(t, original.ID, updated.ID)
		require.Equal(t, original.TenantID, updated.TenantID)
		require.NotEqual(t, original.Name, updated.Name)
		require.NotEqual(t, original.Description, updated.Description)
		require.Equal(t, original.Direction, updated.Direction)
		require.Equal(t, original.DataDriverInstanceID, updated.DataDriverInstanceID)
		require.NotEqual(t, original.Stream, updated.Stream)

		require.NotEqual(t, original.Labels, updated.Labels)
		require.Len(t, original.Labels, 1)
		require.Len(t, updated.Labels, 1)

		require.NotEqual(t, original.ServiceDomainBinding, updated.ServiceDomainBinding)
		require.Len(t, original.ServiceDomainBinding.ServiceDomainIDs, 1)
		require.Len(t, updated.ServiceDomainBinding.ServiceDomainIDs, 2)

		// clenaup
		_, err = dbAPI.DeleteDataDriverStream(adminContext, ddsId, nil)
		require.NoError(t, err)
	})

	t.Run("Test data driver stream and config missuse", func(t *testing.T) {
		// create data driver stream
		obj1 := makeDataDriverStream("test-1", tenantID, dataDriverInstanceID, edgeID, catID)
		dds, err := dbAPI.CreateDataDriverStream(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dds)
		ddsId := dds.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddsId, "Data driver stream id not found")

		// create data driver config
		obj2 := makeDataDriverConfig("test-1", tenantID, dataDriverInstanceID, edgeID)
		ddc, err := dbAPI.CreateDataDriverConfig(adminContext, &obj2, nil)
		require.NoError(t, err)
		require.NotNil(t, ddc)
		ddcId := ddc.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddcId, "Data driver config id not found")

		// modify config as stream
		obj1u := makeDataDriverStream("test-2", tenantID, dataDriverInstanceID, edgeID, catID)
		obj1u.ID = ddcId
		_, err = dbAPI.UpdateDataDriverConfig(adminContext, &obj1u, nil)
		require.Error(t, err)

		// modify stream as config
		obj2u := makeDataDriverConfig("test-2", tenantID, dataDriverInstanceID, edgeID)
		obj2u.ID = ddsId
		_, err = dbAPI.UpdateDataDriverStream(adminContext, &obj2u, nil)
		require.Error(t, err)

		// modify stream as config
		_, err = dbAPI.GetDataDriverConfig(adminContext, ddsId)
		require.Error(t, err)

		// modify config as stream
		_, err = dbAPI.GetDataDriverStream(adminContext, ddcId)
		require.Error(t, err)

		// delete stream as config
		_, err = dbAPI.DeleteDataDriverConfig(adminContext, ddsId, nil)
		require.Error(t, err)

		// delete config as stream
		_, err = dbAPI.DeleteDataDriverStream(adminContext, ddcId, nil)
		require.Error(t, err)

		// Actual delete
		rsp, err := dbAPI.DeleteDataDriverStream(adminContext, ddsId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		rsp, err = dbAPI.DeleteDataDriverConfig(adminContext, ddcId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)
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

func TestMappingDataDriverStreamDBO(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	schema := map[string]interface{}{
		"a": 1.0,
		"b": "2",
	}
	schemaText := types.JSONText(`{"a":1,"b":"2"}`)
	tests := []struct {
		name    string
		dds     model.DataDriverStream
		want    api.DataDriverParamsDBO
		wantErr bool
	}{
		{
			name: "data driver stream",
			dds: model.DataDriverStream{
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
				Direction:            model.DataDriverStreamSource,
				Stream:               schema,
				ServiceDomainBinding: model.ServiceDomainBinding{
					ServiceDomainIDs: []string{"edeg-1"},
				},
				Labels: []model.CategoryInfo{{ID: "c1", Value: "v1"}},
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
				Direction:            "SOURCE",
				DataDriverInstanceID: "ddi-1",
				Parameters:           &schemaText,
				Type:                 "dataDriverStream",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.ToDataDriverParamsDBOFromStream(&tt.dds)
			require.NoError(t, err)
			require.EqualValues(t, got, tt.want)

			back, err := api.FromDataDriverParamsDBOToStream(&got, &tt.dds.ServiceDomainBinding, tt.dds.Labels)
			require.NoError(t, err)
			require.EqualValues(t, back, tt.dds)

			_, err = api.FromDataDriverParamsDBOToConfig(&got, &tt.dds.ServiceDomainBinding)
			require.Error(t, err)
		})
	}
}

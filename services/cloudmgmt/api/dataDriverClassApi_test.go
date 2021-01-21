package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"

	"context"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

func makeDataDriverClass(name string, tenantID string) model.DataDriverClass {
	return model.DataDriverClass{
		BaseModel: model.BaseModel{
			ID:       "dd-id-" + funk.RandomString(10),
			TenantID: tenantID,
		},
		DataDriverClassCore: model.DataDriverClassCore{
			Name:                name,
			Description:         "dd-desc-1" + funk.RandomString(10),
			DataDriverVersion:   "1.0.2 " + funk.RandomString(10),
			MinSvcDomainVersion: "1.0" + funk.RandomString(10),
			Type:                "SOURCE",
			YamlData:            "YAML_DATA " + funk.RandomString(10),
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
					"anotherTest": map[string]interface{}{
						"type":    "string",
						"default": funk.RandomString(10),
					},
				},
			},
			StreamParameterSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"test": map[string]interface{}{
						"type":    "string",
						"default": funk.RandomString(10),
					},
				},
			},
		},
	}
}

func createDataDriverClass(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, name string) model.DataDriverClass {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	dataDriverClass := makeDataDriverClass(name, tenantID)

	resp, err := dbAPI.CreateDataDriverClass(ctx, &dataDriverClass, nil)
	require.NoError(t, err)
	t.Logf("create data driver class successful, %s", resp)
	createResp := resp.(model.CreateDocumentResponseV2)
	dataDriverClass, err = dbAPI.GetDataDriverClass(ctx, createResp.ID)
	require.NoError(t, err)
	return dataDriverClass
}

func TestDataDriverClass(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverClass test")

	// Setup
	dbAPI := newObjectModelAPI(t)

	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID

	impostorTenant := createTenant(t, dbAPI, "test impostor tenant")
	impostorTenantID := impostorTenant.ID

	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	adminContext, _, userContext := makeContext(tenantID, []string{})
	impostorContext, _, _ := makeContext(impostorTenantID, []string{})

	// Teardown
	defer func() {
		dbAPI.DeleteEdge(adminContext, edgeID, nil)
		dbAPI.DeleteTenant(adminContext, tenantID, nil)

		dbAPI.Close()
	}()

	t.Run("Test data driver class workflow", func(t *testing.T) {
		// initial search
		dds, count, err := dbAPI.SelectAllDataDriverClasses(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Zero(t, count)
		require.Empty(t, dds, "Error during initial find")

		// create data driver class
		obj1 := makeDataDriverClass("test-1", tenantID)
		dd, err := dbAPI.CreateDataDriverClass(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddId, "Data driver class id not found")

		// modify
		obj2 := makeDataDriverClass("test-2", tenantID)
		obj2.ID = ddId
		updated, err := dbAPI.UpdateDataDriverClass(adminContext, &obj2, nil)
		updatedID := updated.(model.UpdateDocumentResponseV2).ID
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.Equal(t, ddId, updatedID)

		// modify without id
		obj3 := makeDataDriverClass("test-3", tenantID)
		_, err = dbAPI.UpdateDataDriverClass(adminContext, &obj3, nil)
		require.Error(t, err, "Should fail")

		// project-level modify should fail
		obj3.ID = ddId
		_, err = dbAPI.UpdateDataDriverClass(userContext, &obj3, nil)
		require.Error(t, err, "Should fail")

		// impostor modify should fail
		_, err = dbAPI.UpdateDataDriverClass(impostorContext, &obj3, nil)
		require.Error(t, err, "Should fail")

		// find newly created
		dds2, count, err := dbAPI.SelectAllDataDriverClasses(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, dds2, 1, "Error during find")

		// get by id
		dd, err = dbAPI.GetDataDriverClass(adminContext, ddId)
		require.NoError(t, err)
		require.NotNil(t, dd, "Failed to get by id")

		// search
		dds, count, err = dbAPI.SelectAllDataDriverClasses(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, dds, 1, "Error during find by name")

		// get by incorrect name
		dds, count, err = dbAPI.SelectAllDataDriverClasses(adminContext, &model.EntitiesQueryParam{Filter: "name = 'test-1'"})
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, dds, "Should not find anything")

		// visible by user
		dd, err = dbAPI.GetDataDriverClass(userContext, ddId)
		require.NoError(t, err)
		require.NotNil(t, dd, "Failed to get by id")

		// not visible by impostor
		dd, err = dbAPI.GetDataDriverClass(impostorContext, ddId)
		require.Error(t, err)

		// searcheable by user
		dds, count, err = dbAPI.SelectAllDataDriverClasses(userContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, dds, 1, "Error during find by name")

		// not searcheable by impostor
		dds, count, err = dbAPI.SelectAllDataDriverClasses(impostorContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Len(t, dds, 0, "Error during find by name")

		// delete as user failed
		rsp, err := dbAPI.DeleteDataDriverClass(userContext, ddId, nil)
		require.Error(t, err, "Should fail")

		// delete as impostor failed
		rsp, err = dbAPI.DeleteDataDriverClass(impostorContext, ddId, nil)
		require.Error(t, err, "Should fail")

		// delete as admin
		rsp, err = dbAPI.DeleteDataDriverClass(adminContext, ddId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		// create as user should fail
		obj4 := makeDataDriverClass("test-4", tenantID)
		dd, err = dbAPI.CreateDataDriverClass(userContext, &obj4, nil)
		require.Error(t, err)

		// get all again & it should be empty
		dds, count, err = dbAPI.SelectAllDataDriverClasses(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, dds, "Error during last find")
	})

	t.Run("Test data driver class update fields", func(t *testing.T) {
		// create data driver class
		obj1 := makeDataDriverClass("test-origin", tenantID)
		dd, err := dbAPI.CreateDataDriverClass(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddId, "Data driver class id not found")

		obj1, err = dbAPI.GetDataDriverClass(adminContext, ddId)
		require.NoError(t, err)

		// modify
		obj2 := makeDataDriverClass("test-edited", tenantID)
		obj2.ID = ddId
		obj2.Type = model.DataDriverBoth
		updated, err := dbAPI.UpdateDataDriverClass(adminContext, &obj2, nil)
		require.NoError(t, err)
		require.NotNil(t, updated)

		obj2, err = dbAPI.GetDataDriverClass(adminContext, ddId)
		require.NoError(t, err)

		// check values
		require.Equal(t, obj1.ID, obj2.ID)
		require.Equal(t, obj1.TenantID, obj2.TenantID)
		require.NotEqual(t, obj1.Type, obj2.Type)
		require.NotEqual(t, obj1.Name, obj2.Name)
		require.NotEqual(t, obj1.Description, obj2.Description)
		require.NotEqual(t, obj1.DataDriverVersion, obj2.DataDriverVersion)
		require.NotEqual(t, obj1.MinSvcDomainVersion, obj2.MinSvcDomainVersion)
		require.NotEqualValues(t, obj1.StaticParameterSchema, obj2.StaticParameterSchema)
		require.NotEqualValues(t, obj1.ConfigParameterSchema, obj2.ConfigParameterSchema)
		require.NotEqualValues(t, obj1.StreamParameterSchema, obj2.StreamParameterSchema)
		require.NotEqual(t, obj1.YamlData, obj2.YamlData)

		// can delete
		_, err = dbAPI.DeleteDataDriverClass(adminContext, ddId, nil)
		require.NoError(t, err)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := makeDataDriverClass("name-10", tenantID)
		doc.ID = id
		return dbAPI.CreateDataDriverClass(adminContext, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetDataDriverClass(adminContext, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteDataDriverClass(adminContext, id, nil)
	}))
}

func TestMappingDatadriverClassDBO(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	schema := map[string]interface{}{
		"a": 1.0,
		"b": "2",
	}
	schemaText := types.JSONText(`{"a":1,"b":"2"}`)
	tests := []struct {
		name    string
		ddc     model.DataDriverClass
		want    api.DataDriverClassDBO
		wantErr bool
	}{
		{
			name: "data driver class",
			ddc: model.DataDriverClass{
				BaseModel: model.BaseModel{
					ID:        "ddc-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverClassCore: model.DataDriverClassCore{
					Name:                  "Name 1",
					Description:           "TEST",
					DataDriverVersion:     "1.0",
					MinSvcDomainVersion:   "2.0",
					Type:                  "SOURCE",
					YamlData:              "YAML DATA",
					StaticParameterSchema: schema,
					ConfigParameterSchema: schema,
					StreamParameterSchema: schema,
				},
			},
			want: api.DataDriverClassDBO{
				BaseModelDBO: model.BaseModelDBO{
					ID:        "ddc-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                  "Name 1",
				Description:           "TEST",
				DataDriverVersion:     "1.0",
				MinSvcDomainVersion:   "2.0",
				Type:                  "SOURCE",
				YamlData:              "YAML DATA",
				StaticParameterSchema: &schemaText,
				ConfigParameterSchema: &schemaText,
				StreamParameterSchema: &schemaText,
			},
		},
		{
			name: "empty data driver class",
			ddc: model.DataDriverClass{
				BaseModel: model.BaseModel{
					ID:        "ddc-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverClassCore: model.DataDriverClassCore{
					Name:                "Name 1",
					Description:         "TEST",
					DataDriverVersion:   "1.0",
					MinSvcDomainVersion: "2.0",
					Type:                "SOURCE",
					YamlData:            "YAML DATA",
				},
			},
			want: api.DataDriverClassDBO{
				BaseModelDBO: model.BaseModelDBO{
					ID:        "ddc-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:                "Name 1",
				Description:         "TEST",
				DataDriverVersion:   "1.0",
				MinSvcDomainVersion: "2.0",
				Type:                "SOURCE",
				YamlData:            "YAML DATA",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.ToDataDriverClassDBO(&tt.ddc)
			require.NoError(t, err)
			assert.Equal(t, got, tt.want)

			back, err := api.FromDataDriverClassDBO(&got)
			require.NoError(t, err)
			assert.Equal(t, back, tt.ddc)
		})
	}
}

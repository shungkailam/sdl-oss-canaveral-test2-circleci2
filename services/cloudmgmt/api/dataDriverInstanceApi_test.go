package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/api/testtool"
	"cloudservices/common/base"
	"cloudservices/common/model"

	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
)

var (
	yamlTemplate = `apiVersion: apps/v1
		kind: Deployment
		metadata:
		  name: mqttdatadriver
		spec:
		  replicas: 1
		  selector:
			matchLabels:
			  app: mqttdatadriver
		  template:
			metadata:
			  name: mqttdatadriver
			  labels:
				app: mqttdatadriver
			spec:
			  containers:
				- name: mqttdatadriver
				  image: "770301640873.dkr.ecr.us-west-2.amazonaws.com/edgecomputing/datadriver/mqtt:{{ .Parameters.image_tag }}"
				  imagePullPolicy: Always
				  securityContext:
					runAsUser: 9999
					allowPrivilegeEscalation: false
				  ports:
					- containerPort: 9090
				  env:
					- name: MQTT_CA_CERT
					  value: {{ .Parameters.mqttCACert }}
					- name: MQTT_CLIENT_CERT
					  value: {{ .Parameters.mqttClientCert }}
					- name: MQTT_CLIENT_KEY
					  value: {{ .Parameters.mqttClientKey }}`

	yamlExpected = `apiVersion: apps/v1
		kind: Deployment
		metadata:
		  name: mqttdatadriver
		spec:
		  replicas: 1
		  selector:
			matchLabels:
			  app: mqttdatadriver
		  template:
			metadata:
			  name: mqttdatadriver
			  labels:
				app: mqttdatadriver
			spec:
			  containers:
				- name: mqttdatadriver
				  image: "770301640873.dkr.ecr.us-west-2.amazonaws.com/edgecomputing/datadriver/mqtt:%s"
				  imagePullPolicy: Always
				  securityContext:
					runAsUser: 9999
					allowPrivilegeEscalation: false
				  ports:
					- containerPort: 9090
				  env:
					- name: MQTT_CA_CERT
					  value: %s
					- name: MQTT_CLIENT_CERT
					  value: %s
					- name: MQTT_CLIENT_KEY
					  value: %s`
	templateSchemaJson = `{
		"type": "object",
		"properties": {
		  "image_tag": {
			"type": "string",
			"description": "test docker image tag to render in yaml"
		  },
		  "mqttCACert": {
			"type": "string",
			"description": "test env var to render in yaml"
		  },
		  "mqttClientCert": {
			"type": "string",
			"description": "test env var to render in yaml"
		  },
		  "mqttClientKey": {
			"type": "string",
			"description": "test env var to render in yaml"
		  }
		}
	}`
)

func makeDataDriverInstance(name string, tenantID, ddClassID, projectID string) model.DataDriverInstance {
	return model.DataDriverInstance{
		BaseModel: model.BaseModel{
			ID:       "ddi-id-" + funk.RandomString(10),
			TenantID: tenantID,
		},
		DataDriverInstanceCore: model.DataDriverInstanceCore{
			Name:              name,
			Description:       "ddi-desc-1-" + funk.RandomString(10),
			DataDriverClassID: ddClassID,
			ProjectID:         projectID,
			StaticParameters: map[string]interface{}{
				"key": "value-" + funk.RandomString(10),
				"i":   7,
			},
		},
	}
}

func createDataDriverInstance(t *testing.T, dbAPI api.ObjectModelAPI, tenantID, dataDriverID, projectID string) model.DataDriverInstance {
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

	dataDriverInstance := makeDataDriverInstance("ddi-"+funk.RandomString(10), tenantID, dataDriverID, projectID)

	resp, err := dbAPI.CreateDataDriverInstance(ctx, &dataDriverInstance, nil)
	require.NoError(t, err)
	t.Logf("create data driver class successful, %s", resp)
	createResp := resp.(model.CreateDocumentResponseV2)
	dataDriverInstance, err = dbAPI.GetDataDriverInstance(ctx, createResp.ID)
	require.NoError(t, err)
	return dataDriverInstance
}

func TestDataDriverInstance(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataDriverInstance test")

	// Setup
	dbAPI := newObjectModelAPI(t)

	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID

	impostorTenant := createTenant(t, dbAPI, "test impostor tenant")
	impostorTenantID := impostorTenant.ID

	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
	projectID := project.ID

	anotherProject := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
	anotherProjectID := anotherProject.ID

	impostorProject := createExplicitProjectCommon(t, dbAPI, impostorTenantID, []string{}, []string{}, []string{}, []string{})
	impostorProjectID := impostorProject.ID

	adminContext, _, _ := makeContext(tenantID, []string{projectID})

	// create data driver class 1
	dataDriverClass := createDataDriverClass(t, dbAPI, tenantID, "test data driver class")
	dataDriverClassID := dataDriverClass.ID

	anotherDataDriverClass := createDataDriverClass(t, dbAPI, tenantID, "another test data driver")
	anotherDataDriverClassID := anotherDataDriverClass.ID

	apiTools := testtool.APITestTool("DataDriverInstance").
		ForTenant(tenantID, projectID, anotherProjectID).
		ForImpostor(impostorTenantID, impostorProjectID).
		PermissionsMatrix(testtool.InfraProjectLevelObject()).
		WithSelector(func(ctx context.Context, tenantId, projectId string) (interface{}, error) {
			ddis, _, err := dbAPI.SelectAllDataDriverInstances(ctx, &model.EntitiesQueryParam{})
			return ddis, err
		}).
		WithSelector(func(ctx context.Context, tenantId, projectId string) (interface{}, error) {
			return dbAPI.SelectAllDataDriverInstancesByClassId(ctx, dataDriverClassID)
		}).
		WithChecker(func(ctx context.Context, id string) (interface{}, error) {
			return dbAPI.GetDataDriverInstance(ctx, id)
		}).
		WithCreator(func(ctx context.Context, id, tenantId, projectId string) (interface{}, error) {
			obj := makeDataDriverInstance("test-"+funk.RandomString(10), tenantId, dataDriverClassID, projectId)
			obj.ID = id
			return dbAPI.CreateDataDriverInstance(ctx, &obj, nil)
		}).
		WithUpdater(func(ctx context.Context, id, tenantId, projectId string) (interface{}, error) {
			obj := makeDataDriverInstance("test-"+funk.RandomString(10), tenantId, dataDriverClassID, projectId)
			obj.ID = id
			return dbAPI.UpdateDataDriverInstance(ctx, &obj, nil)
		}).
		WithDeleter(func(ctx context.Context, id string) (interface{}, error) {
			return dbAPI.DeleteDataDriverInstance(ctx, id, nil)
		})

	// Teardown
	defer func() {
		dbAPI.DeleteDataDriverClass(adminContext, anotherDataDriverClassID, nil)
		dbAPI.DeleteDataDriverClass(adminContext, dataDriverClassID, nil)
		dbAPI.DeleteProject(adminContext, projectID, nil)
		dbAPI.DeleteEdge(adminContext, edgeID, nil)
		dbAPI.DeleteTenant(adminContext, tenantID, nil)

		dbAPI.Close()
	}()

	t.Run("Test data driver instance workflow", func(t *testing.T) {
		// initial search should be empty
		ddis, count, err := dbAPI.SelectAllDataDriverInstances(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Zero(t, count)
		require.Empty(t, ddis, "Error during initial find")

		// not found anythong by class id
		ddis, err = dbAPI.SelectAllDataDriverInstancesByClassId(adminContext, dataDriverClassID)
		require.NoError(t, err)
		require.Len(t, ddis, 0)

		// create data driver instance
		obj1 := makeDataDriverInstance("test-1", tenantID, dataDriverClassID, projectID)
		dd, err := dbAPI.CreateDataDriverInstance(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddiId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddiId, "Data driver instance id not found")

		// modify
		obj2 := makeDataDriverInstance("test-2", tenantID, dataDriverClassID, projectID)
		obj2.ID = ddiId
		updated, err := dbAPI.UpdateDataDriverInstance(adminContext, &obj2, nil)
		updatedID := updated.(model.UpdateDocumentResponseV2).ID
		require.NoError(t, err)
		require.NotNil(t, updated)
		require.Equal(t, ddiId, updatedID)

		// find newly created
		ddis2, count, err := dbAPI.SelectAllDataDriverInstances(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, ddis2, 1, "Error during find")

		// get by id
		dd, err = dbAPI.GetDataDriverInstance(adminContext, ddiId)
		require.NoError(t, err)
		require.NotNil(t, dd, "Failed to get by id")

		// search
		ddis, count, err = dbAPI.SelectAllDataDriverInstances(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 1, count)
		require.Len(t, ddis, 1, "Error during find by name")

		// get by incorrect name
		ddis, count, err = dbAPI.SelectAllDataDriverInstances(adminContext, &model.EntitiesQueryParam{Filter: "name = 'test-1'"})
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, ddis, "Should not find anything")

		// get by class id
		ddis, err = dbAPI.SelectAllDataDriverInstancesByClassId(adminContext, dataDriverClassID)
		require.NoError(t, err)
		require.Len(t, ddis, 1)

		// delete as admin
		rsp, err := dbAPI.DeleteDataDriverInstance(adminContext, ddiId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		// get all again & it should be empty
		ddis, count, err = dbAPI.SelectAllDataDriverInstances(adminContext, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Equal(t, 0, count)
		require.Empty(t, ddis, "Error during last find")
	})

	t.Run("Test data driver instance update fields", func(t *testing.T) {
		// create data driver instance
		obj1 := makeDataDriverInstance("test-origin", tenantID, dataDriverClassID, projectID)
		dd, err := dbAPI.CreateDataDriverInstance(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddId, "Data driver instance id not found")

		original, err := dbAPI.GetDataDriverInstance(adminContext, ddId)
		require.NoError(t, err)
		require.NotNil(t, original)

		// modify
		obj2 := makeDataDriverInstance("test-edited", tenantID, anotherDataDriverClassID, anotherProjectID)
		obj2.ID = ddId
		_, err = dbAPI.UpdateDataDriverInstance(adminContext, &obj2, nil)
		require.NoError(t, err)

		updated, err := dbAPI.GetDataDriverInstance(adminContext, ddId)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// check values
		require.Equal(t, original.ID, updated.ID)
		require.Equal(t, original.TenantID, updated.TenantID)
		require.Equal(t, original.ProjectID, updated.ProjectID)
		require.Equal(t, original.DataDriverClassID, updated.DataDriverClassID)
		require.NotEqual(t, original.Name, updated.Name)
		require.NotEqual(t, original.Description, updated.Description)
		require.NotEqual(t, original.StaticParameters, updated.StaticParameters)

		// can delete
		_, err = dbAPI.DeleteDataDriverInstance(adminContext, ddId, nil)
		require.NoError(t, err)
	})

	t.Run("Test instance template enginge", func(t *testing.T) {
		// create data driver class
		ddc := makeDataDriverClass("test data driver", tenantID)
		ddc.YamlData = yamlTemplate
		var staticParameters map[string]interface{}
		err := json.Unmarshal([]byte(templateSchemaJson), &staticParameters)
		require.NoError(t, err)
		ddc.StaticParameterSchema = staticParameters

		resp, err := dbAPI.CreateDataDriverClass(adminContext, &ddc, nil)
		require.NoError(t, err)
		ddcID := resp.(model.CreateDocumentResponseV2).ID

		// create data driver instance
		image_tag := funk.RandomString(100)
		mqttCACert := funk.RandomString(100)
		mqttClientCert := funk.RandomString(100)
		mqttClientKey := funk.RandomString(100)
		obj := makeDataDriverInstance("test-template", tenantID, ddcID, projectID)
		obj.StaticParameters = map[string]interface{}{
			"image_tag":      image_tag,
			"mqttCACert":     mqttCACert,
			"mqttClientCert": mqttClientCert,
			"mqttClientKey":  mqttClientKey,
		}
		dd, err := dbAPI.CreateDataDriverInstance(adminContext, &obj, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		ddiId := dd.(model.CreateDocumentResponseV2).ID
		require.NotEmpty(t, ddiId, "Data driver instance id not found")

		// check YAML templating
		edgeCtx := makeEdgeContext(tenantID, edgeID, nil)
		inventory, err := dbAPI.GetEdgeInventoryDelta(edgeCtx, &model.EdgeInventoryDeltaPayload{})
		require.NoError(t, err)
		require.NotNil(t, inventory)
		require.NotNil(t, inventory.Created.DataDriverInstances)
		require.Len(t, inventory.Created.DataDriverInstances, 1)
		got := inventory.Created.DataDriverInstances[0].YamlData
		expected := fmt.Sprintf(yamlExpected, image_tag, mqttCACert, mqttClientCert, mqttClientKey)
		require.Equal(t, got, expected)

		// cleanup
		rsp, err := dbAPI.DeleteDataDriverInstance(adminContext, ddiId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		rsp, err = dbAPI.DeleteDataDriverClass(adminContext, ddcID, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)
	})

	t.Run("Test class delete or modification with existing instance", func(t *testing.T) {
		// create data driver class
		ddc := makeDataDriverClass("test data driver", tenantID)
		_, err := dbAPI.CreateDataDriverClass(adminContext, &ddc, nil)
		require.NoError(t, err)
		ddcID := ddc.ID

		// create data driver instance
		obj1 := makeDataDriverInstance("test-origin", tenantID, ddcID, projectID)
		dd, err := dbAPI.CreateDataDriverInstance(adminContext, &obj1, nil)
		require.NoError(t, err)
		require.NotNil(t, dd)

		// try to delete data driver class
		_, err = dbAPI.DeleteDataDriverClass(adminContext, ddcID, nil)
		require.Error(t, err)

		// try to modify data driver class
		ddc2 := makeDataDriverClass("test data driver", tenantID)
		ddc2.ID = ddc.ID
		_, err = dbAPI.UpdateDataDriverClass(adminContext, &ddc2, nil)
		require.Error(t, err)

		// try to delete data driver instance
		_, err = dbAPI.DeleteDataDriverInstance(adminContext, obj1.ID, nil)
		require.NoError(t, err)

		// and now we can edit
		_, err = dbAPI.UpdateDataDriverClass(adminContext, &ddc2, nil)
		require.NoError(t, err)

		// try to delete data driver class
		_, err = dbAPI.DeleteDataDriverClass(adminContext, ddcID, nil)
		require.NoError(t, err)

		// data driver not found
		_, err = dbAPI.GetDataDriverClass(adminContext, ddcID)
		require.Error(t, err)
	})

	t.Run("Test permissions on data driver instance", func(t *testing.T) {
		t.Run("Search", apiTools.SearchRBACTest())
		t.Run("Read", apiTools.ReadRBACTest())
		t.Run("Creation", apiTools.CreateRBACTest())
		t.Run("Update", apiTools.UpdateRBACTest())
		t.Run("Delete", apiTools.DeleteRBACTest())
	})

	t.Run("ID validity", apiTools.IdSanityTest())
}

func TestMappingDataDriverInstanceDBO(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	schema := map[string]interface{}{
		"a": 1.0,
		"b": "2",
	}
	schemaText := types.JSONText(`{"a":1,"b":"2"}`)
	tests := []struct {
		name    string
		ddc     model.DataDriverInstance
		want    api.DataDriverInstanceDBO
		wantErr bool
	}{
		{
			name: "data driver instance",
			ddc: model.DataDriverInstance{
				BaseModel: model.BaseModel{
					ID:        "ddi-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceCore: model.DataDriverInstanceCore{
					Name:              "Name 1",
					Description:       "TEST",
					DataDriverClassID: "ddc-id-1",
					ProjectID:         "project-1",
					StaticParameters:  schema,
				},
			},
			want: api.DataDriverInstanceDBO{
				BaseModelDBO: model.BaseModelDBO{
					ID:        "ddi-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:              "Name 1",
				Description:       "TEST",
				DataDriverClassID: "ddc-id-1",
				ProjectID:         "project-1",
				StaticParameters:  &schemaText,
			},
		},
		{
			name: "empty data driver instance",
			ddc: model.DataDriverInstance{
				BaseModel: model.BaseModel{
					ID:        "ddi-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				DataDriverInstanceCore: model.DataDriverInstanceCore{
					Name:              "Name 1",
					Description:       "TEST",
					DataDriverClassID: "ddc-id-1",
					ProjectID:         "project-1",
				},
			},
			want: api.DataDriverInstanceDBO{
				BaseModelDBO: model.BaseModelDBO{
					ID:        "ddi-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:              "Name 1",
				Description:       "TEST",
				DataDriverClassID: "ddc-id-1",
				ProjectID:         "project-1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.ToDataDriverInstanceDBO(&tt.ddc)
			require.NoError(t, err)
			assert.Equal(t, got, tt.want)

			back, err := api.FromDataDriverInstanceDBO(&got)
			require.NoError(t, err)
			assert.Equal(t, back, tt.ddc)
		})
	}
}

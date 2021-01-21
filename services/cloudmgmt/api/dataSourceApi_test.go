package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createDataSourceWithSelectorsFields(t *testing.T, ctx context.Context, dbAPI api.ObjectModelAPI,
	name, tenantID, edgeID, sensorModel, protocol string, ifcInfo model.DataSourceIfcInfo, selectors []model.DataSourceFieldSelector,
	fields []model.DataSourceFieldInfo,
) (interface{}, error) {
	ds := model.DataSource{
		EdgeBaseModel: model.EdgeBaseModel{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			EdgeID: edgeID,
		},
		DataSourceCore: model.DataSourceCore{
			Name:       name,
			Type:       "Sensor",
			Connection: "Secure",
			Selectors:  selectors,
			Protocol:   "MQTT",
			AuthType:   "CERTIFICATE",
		},
		Fields:      fields,
		SensorModel: sensorModel,
	}

	ds.IfcInfo = &ifcInfo
	ds.Protocol = protocol

	return dbAPI.CreateDataSource(ctx, &ds, nil)
}

func createDataSource(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, edgeID string, categoryID string, categoryValue string) model.DataSource {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					Role: model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	doc := generateDataSource(tenantID, edgeID, categoryID, categoryValue)
	// create DataSource
	resp, err := dbAPI.CreateDataSource(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create DataSource successful, %s", resp)

	dataSourceID := resp.(model.CreateDocumentResponse).ID
	dataSource, err := dbAPI.GetDataSource(ctx, dataSourceID)
	require.NoError(t, err)
	return dataSource
}

func generateDataSource(tenantID string, edgeID string, categoryID string, categoryValue string) model.DataSource {
	id := base.GetUUID()
	sensorModel := "Model 3"
	return model.DataSource{
		EdgeBaseModel: model.EdgeBaseModel{
			BaseModel: model.BaseModel{
				ID:       id,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeID: edgeID,
		},
		DataSourceCore: model.DataSourceCore{
			Name:       "datasource-name-" + id,
			Type:       "Sensor",
			Connection: "Secure",
			Selectors: []model.DataSourceFieldSelector{
				{
					CategoryInfo: model.CategoryInfo{
						ID:    categoryID,
						Value: categoryValue,
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
					Name:      "field-name-" + id,
					FieldType: "field-type-1",
				},
				MQTTTopic: "mqtt-topic-" + id,
			},
		},
		SensorModel: sensorModel,
	}
}

func TestDataSource(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataSource test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID

	var edgeID, projectID, categoryID string
	var ctx1, ctx2, ctx3 context.Context

	// test with old edge API
	setup1 := func() {
		edge := createEdge(t, dbAPI, tenantID)
		edgeID = edge.ID
		project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
		projectID = project.ID
		ctx1, ctx2, ctx3 = makeContext(tenantID, []string{projectID})
		category := createCategory(t, dbAPI, tenantID)
		categoryID = category.ID
	}

	// test with new edge cluster / device API
	setup2 := func() {
		edgeDevice := createEdgeDevice(t, dbAPI, tenantID)
		edgeID = edgeDevice.ClusterID
		project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
		projectID = project.ID
		ctx1, ctx2, ctx3 = makeContext(tenantID, []string{projectID})
		category := createCategory(t, dbAPI, tenantID)
		categoryID = category.ID
	}
	cleanup := func() {
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
	}

	// Teardown
	defer func() {
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	core := func() {
		t.Run("Create/Get/Delete TestDataSource", func(t *testing.T) {
			t.Log("running Create/Get/Delete TestDataSource test")

			sensorModel := "Model 3"
			sensorModelUpdated := "Model X"

			// DataSource object, leave ID blank and let create generate it
			doc := model.DataSource{
				EdgeBaseModel: model.EdgeBaseModel{
					BaseModel: model.BaseModel{
						ID:       "",
						TenantID: tenantID,
						Version:  5,
					},
					EdgeID: edgeID,
				},
				DataSourceCore: model.DataSourceCore{
					Name:       "datasource-name",
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
				SensorModel: sensorModel,
			}
			// create DataSource
			resp, err := dbAPI.CreateDataSource(ctx1, &doc, nil)
			require.NoError(t, err)
			t.Logf("create DataSource successful, %s", resp)

			dataSourceId := resp.(model.CreateDocumentResponse).ID

			x, err := dbAPI.GetDataSourceEdgeID(ctx1, dataSourceId)
			require.NoError(t, err)
			if x != edgeID {
				t.Fatalf("Edge ID mismatch %s != %s", x, edgeID)
			}

			// update DataSource
			doc = model.DataSource{
				EdgeBaseModel: model.EdgeBaseModel{
					BaseModel: model.BaseModel{
						ID:       dataSourceId,
						TenantID: tenantID,
						Version:  5,
					},
					EdgeID: edgeID,
				},
				DataSourceCore: model.DataSourceCore{
					Name:       "datasource-name",
					Type:       "Sensor",
					Connection: "Secure",
					Selectors: []model.DataSourceFieldSelector{
						{
							CategoryInfo: model.CategoryInfo{
								ID:    categoryID,
								Value: "v2",
							},
							Scope: []string{
								"field-name-2",
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
					{
						DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
							Name:      "field-name-2",
							FieldType: "field-type-2",
						},
						MQTTTopic: "mqtt-topic-2",
					},
				},
				SensorModel: sensorModelUpdated,
			}
			// get DataSource
			dataSource, err := dbAPI.GetDataSource(ctx1, dataSourceId)
			require.NoError(t, err)
			t.Logf("get DataSource before update successful, %+v", dataSource)

			// select all data sources
			dss, err := dbAPI.SelectAllDataSources(ctx1, nil)
			require.NoError(t, err)
			if len(dss) != 1 {
				t.Fatalf("expect auth 1 DataSources count to be 1, got %d", len(dss))
			}
			dss, err = dbAPI.SelectAllDataSources(ctx2, nil)
			require.NoError(t, err)
			if len(dss) != 0 {
				t.Fatalf("expect auth 2 DataSources count to be 0, got %d", len(dss))
			}
			dss, err = dbAPI.SelectAllDataSources(ctx3, nil)
			require.NoError(t, err)
			if len(dss) != 1 {
				t.Fatalf("expect auth 3 DataSources count to be 1, got %d", len(dss))
			}

			// select all data sources for project
			authContext1 := &base.AuthContext{
				TenantID: tenantID,
				Claims: jwt.MapClaims{
					"specialRole": "admin",
				},
			}
			newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
			dss, err = dbAPI.SelectAllDataSourcesForProject(newCtx, projectID, nil)
			require.Error(t, err, "expect select all data sources 1 for project to fail")
			dss, err = dbAPI.SelectAllDataSourcesForProject(ctx2, projectID, nil)
			require.Error(t, err, "expect select all data sources 2 for project to fail")
			dss, err = dbAPI.SelectAllDataSourcesForProject(ctx3, projectID, nil)
			require.NoError(t, err)
			if len(dss) != 1 {
				t.Fatalf("expect auth 3 DataSources for project count to be 1, got %d", len(dss))
			}

			upResp, err := dbAPI.UpdateDataSource(ctx1, &doc, nil)
			require.NoError(t, err)
			t.Logf("update DataSource successful, %+v", upResp)

			// get DataSource
			dataSource, err = dbAPI.GetDataSource(ctx1, dataSourceId)
			require.NoError(t, err)
			t.Logf("get DataSource successful, %+v", dataSource)

			if dataSource.ID != dataSourceId || dataSource.TenantID != tenantID || dataSource.SensorModel != sensorModelUpdated || len(dataSource.Fields) != 2 {
				t.Fatal("DataSource data mismatch")
			}
			artifact := &model.DataSourceArtifact{
				DataSourceID: dataSourceId,
				ArtifactBaseModel: model.ArtifactBaseModel{
					Data: map[string]interface{}{
						"secret": "12345",
						"channel1": map[string]string{
							"url": "rtmp://34.221.86.34:30070/myapp/channel1?e={{.expiry}}&st={{.token}}&cs={{.clientsecret}}",
						},
						"free2": map[string]string{
							"url": "rtmp://34.221.86.34:30070/myapp/free2?e={{.expiry}}&st={{.token}}&cs={{.clientsecret}}",
						},
					},
				},
			}
			artifactResp, err := dbAPI.CreateDataSourceArtifact(ctx1, artifact, nil)
			require.Errorf(t, err, "Must fail because the caller is not edge")
			// select all data sources for project
			edgeAuthContext := &base.AuthContext{
				TenantID: tenantID,
				Claims: jwt.MapClaims{
					"specialRole": "edge",
					"edgeId":      edgeID,
				},
			}
			edgeCtx := context.WithValue(context.Background(), base.AuthContextKey, edgeAuthContext)
			artifactResp, err = dbAPI.CreateDataSourceArtifact(edgeCtx, artifact, nil)
			require.NoError(t, err)
			id := artifactResp.(model.CreateDocumentResponse).ID
			if id != dataSourceId {
				t.Fatalf("Mismatched data source ID. Expected %s, found %s", dataSourceId, id)
			}
			outArtifact, err := dbAPI.GetDataSourceArtifact(ctx1, dataSourceId)
			require.NoError(t, err)
			if outArtifact.DataSourceID != dataSourceId {
				t.Fatalf("Mismatched data source ID. Expected %s, found %s", dataSourceId, outArtifact.DataSourceID)
			}
			val, ok := outArtifact.Data["channel1"]
			if !ok {
				t.Fatalf("Unexpected artifact data output - missing key")
			}
			innerMap, ok := val.(map[string]interface{})
			if !ok {
				t.Fatalf("Unexpected artifact data output - wrong type")
			}
			url, ok := innerMap["url"]
			if !ok {
				t.Fatalf("Unexpected artifact data output - missing key")
			}
			if !strings.Contains(url.(string), "rtmp://34.221.86.34:30070/myapp/channel1") || strings.Contains(url.(string), "{{.expiry}}") {
				t.Fatalf("Unexpected artifact data output - wrong key value")
			}
			// delete DataSource
			delResp, err := dbAPI.DeleteDataSource(ctx1, dataSourceId, nil)
			require.NoError(t, err)
			t.Logf("delete DataSource successful, %v", delResp)

		})

		// select all DataSource
		t.Run("SelectAllDataSources", func(t *testing.T) {
			t.Log("running SelectAllSensors test")
			dataSources, err := dbAPI.SelectAllDataSources(ctx1, nil)
			require.NoError(t, err)
			for _, dataSource := range dataSources {
				testForMarshallability(t, dataSource)
			}
		})

		// select all DataSource for edge
		t.Run("SelectAllDataSourcesForEdge", func(t *testing.T) {
			edgeId := "ORD"
			t.Log("running SelectAllDataSourcesForEdge test")
			dataSources, err := dbAPI.SelectAllDataSourcesForEdge(ctx1, edgeId, nil)
			require.NoError(t, err)
			for _, dataSource := range dataSources {
				testForMarshallability(t, dataSource)
			}
		})

		t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
			doc := generateDataSource(tenantID, edgeID, categoryID, "v1")
			doc.ID = id
			return dbAPI.CreateDataSource(ctx1, &doc, nil)
		}, func(id string) (interface{}, error) {
			return dbAPI.GetDataSource(ctx1, id)
		}, func(id string) (interface{}, error) {
			return dbAPI.DeleteDataSource(ctx1, id, nil)
		}))
	}

	setup1()
	core()
	cleanup()

	setup2()
	core()
	cleanup()
}

func TestDataSourceWithDataStreamOutIfc(t *testing.T) {
	t.Log("running TestDataSourceWithDataStreamOutIfc test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
	projectID := project.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})
	inIfcInfo := model.DataSourceIfcInfo{Class: "DATAINTERFACE",
		Kind: "OUT", Protocol: "DATAINTERFACE", Img: "foo",
		ProjectID: "ingress", DriverID: "bar",
	}
	topic := "test-topic-" + base.GetUUID()
	fieldName := "field-name-" + base.GetUUID()
	field := model.DataSourceFieldInfo{
		DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
			Name:      fieldName,
			FieldType: "field-type-1",
		},
		MQTTTopic: topic,
	}

	resp, err := createDataSourceWithSelectorsFields(t, ctx, dbAPI, "data-in-interface-"+base.GetUUID(),
		tenantID, edge.ID, "Model 3", "DATAINTERFACE", inIfcInfo, nil,
		nil,
	)
	require.NoErrorf(t, err, "failed to create data source")

	inDataSourceID := resp.(model.CreateDocumentResponse).ID
	defer dbAPI.DeleteDataSource(ctx, inDataSourceID, nil)

	ds, err := dbAPI.GetDataSource(ctx, inDataSourceID)
	require.NoError(t, err)

	streamName := "stream-with-data-out"
	dataStreamDataType := "Custom"
	var size float64 = 1000000
	undeployed := "UNDEPLOY"
	// create datastream with out data interfaces
	streamWithDataOut := model.DataStream{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:                   streamName,
		DataType:               dataStreamDataType,
		Origin:                 "DataSource",
		EndPoint:               field.MQTTTopic,
		OriginID:               "",
		Destination:            model.DestinationDataInterface,
		CloudType:              "",
		CloudCredsID:           cloudCredsID,
		AWSCloudRegion:         "us-west-2",
		GCPCloudRegion:         "",
		EdgeStreamType:         "",
		AWSStreamType:          "",
		GCPStreamType:          "",
		Size:                   size,
		EnableSampling:         false,
		SamplingInterval:       0,
		TransformationArgsList: []model.TransformationArgs{},
		DataRetention:          []model.RetentionInfo{},
		ProjectID:              project.ID,
		DataIfcEndpoints:       []model.DataIfcEndpoint{{ID: ds.ID, Name: field.MQTTTopic, Value: field.MQTTTopic}},
		State:                  &undeployed,
	}

	resp, err = dbAPI.CreateDataStream(ctx, &streamWithDataOut, nil)
	require.NoError(t, err)
	dataStreamID := resp.(model.CreateDocumentResponse).ID
	defer dbAPI.DeleteDataStream(ctx, dataStreamID, nil)

	dataStream, err := dbAPI.GetDataStream(ctx, dataStreamID)
	require.NoError(t, err)

	assertState := func(ID string, expectedState *string) {
		dataStream, err := dbAPI.GetDataStream(ctx, ID)
		require.NoError(t, err)
		if expectedState == nil && dataStream.State != nil {
			t.Fatalf("expected state to be nil, but got %s", *dataStream.State)
		}

		if expectedState != nil && *dataStream.State != *expectedState {
			t.Fatalf("expected state to be %s, but got %s", *expectedState, *dataStream.State)
		}
	}

	deployed := "DEPLOY"
	dataStream.State = &deployed
	_, err = dbAPI.UpdateDataStream(ctx, &dataStream, nil)
	require.NoError(t, err)
	assertState(dataStreamID, nil)

	ds, err = dbAPI.GetDataSource(ctx, inDataSourceID)
	require.NoError(t, err)

	// Failure case: Changing a topic associated with an existing stream is not allowed
	topic2 := "test-topic2-" + base.GetUUID()
	ds.Fields[0].MQTTTopic = topic2
	_, err = dbAPI.UpdateDataSource(ctx, &ds, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "is used by data stream")

	// Success Case: Add another topic
	ds.Fields[0].MQTTTopic = topic
	newField := ds.Fields[0]
	newField.MQTTTopic = topic2
	newField.Name = "new-name" + base.GetUUID()
	ds.Fields = append(ds.Fields, newField)
	_, err = dbAPI.UpdateDataSource(ctx, &ds, nil)
	require.NoError(t, err)
}

func TestDataSourceWithApplicationEndpoints(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataSourceWithApplicationEndpoints test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
	projectID := project.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})
	inIfcInfo := model.DataSourceIfcInfo{Class: "DATAINTERFACE",
		Kind: "IN", Protocol: "DATAINTERFACE", Img: "foo",
		ProjectID: "ingress", DriverID: "bar",
	}
	inputTopic := "test-topic-" + base.GetUUID()
	inputFieldName := "field-name-" + base.GetUUID()
	inputField := model.DataSourceFieldInfo{
		DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
			Name:      inputFieldName,
			FieldType: "field-type-1",
		},
		MQTTTopic: inputTopic,
	}

	resp, err := createDataSourceWithSelectorsFields(t, ctx, dbAPI, "data-in-interface-"+base.GetUUID(),
		tenantID, edge.ID, "Model 3", "DATAINTERFACE", inIfcInfo, nil,
		[]model.DataSourceFieldInfo{inputField},
	)
	require.NoError(t, err, "failed to create data source")
	inDataSourceID := resp.(model.CreateDocumentResponse).ID
	defer dbAPI.DeleteDataSource(ctx, inDataSourceID, nil)

	app1 := testApp(tenantID, projectID, "test-app1", []string{edgeID}, nil, nil)
	app1.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: inDataSourceID, Name: inputFieldName, Value: inputTopic}}
	rtnApp1, err := createApplicationWithCallback(t, dbAPI, &app1, tenantID, projectID, nil)
	require.NoError(t, err)
	defer dbAPI.DeleteApplication(ctx, rtnApp1.ID, nil)

	dataSource, err := dbAPI.GetDataSource(ctx, inDataSourceID)
	require.NoError(t, err)
	t.Logf("get DataSource successful, %+v", dataSource)

	// Failure case: input field is used by an app
	inputField.Name = "random-name" + base.GetUUID()
	dataSource.Fields = []model.DataSourceFieldInfo{inputField}
	_, err = dbAPI.UpdateDataSource(ctx, &dataSource, nil)

	require.Error(t, err)
	require.Contains(t, err.Error(), "is used by one or more application")

	// Success: Add a new field to the data source
	inputField.Name = inputFieldName
	topic2 := "test-topic-" + base.GetUUID()
	field2Name := "field-name-" + base.GetUUID()
	field2 := model.DataSourceFieldInfo{
		DataSourceFieldInfoCore: model.DataSourceFieldInfoCore{
			Name:      field2Name,
			FieldType: "field-type-1",
		},
		MQTTTopic: topic2,
	}
	dataSource.Fields = []model.DataSourceFieldInfo{inputField, field2}
	_, err = dbAPI.UpdateDataSource(ctx, &dataSource, nil)
	require.NoError(t, err)
	dataSource, err = dbAPI.GetDataSource(ctx, inDataSourceID)
	require.NoError(t, err)
	t.Logf("get DataSource successful, %+v", dataSource)
	if len(dataSource.Fields) != 2 {
		t.Fatalf("expected the number fo fields to be %d, but got %d", 1, len(dataSource.Fields))
	}

	// Success: Remove unused field added in previous step
	dataSource.Fields = []model.DataSourceFieldInfo{inputField}
	_, err = dbAPI.UpdateDataSource(ctx, &dataSource, nil)
	require.NoError(t, err)
	dataSource, err = dbAPI.GetDataSource(ctx, inDataSourceID)
	require.NoError(t, err)
	t.Logf("get DataSource successful, %+v", dataSource)
	if len(dataSource.Fields) != 1 {
		t.Fatalf("expected the number fo fields to be %d, but got %d", 1, len(dataSource.Fields))
	}

	// Success case: the first field is not changed but the topic is changed
	inputField.Name = inputFieldName
	inputField.MQTTTopic = inputTopic + "-test"
	dataSource.Fields = []model.DataSourceFieldInfo{inputField}
	_, err = dbAPI.UpdateDataSource(ctx, &dataSource, nil)
	require.NoError(t, err)

	updatedApp, err := dbAPI.GetApplication(ctx, rtnApp1.ID)
	require.NoError(t, err)
	if updatedApp.DataIfcEndpoints[0].Value != inputField.MQTTTopic {
		t.Fatalf("expected input value to be %s, but got %s", inputField.MQTTTopic, updatedApp.DataIfcEndpoints[0].Value)
	}

	// Failure case: Cannot delete a data source with an application depending on it.
	_, err = dbAPI.DeleteDataSource(ctx, inDataSourceID, nil)
	require.Error(t, err)
	require.Contains(t, err.Error(), "as one or more application endpoint(s) depend on it. Endpoints using this data source")

	_, err = dbAPI.DeleteApplication(ctx, rtnApp1.ID, nil)
	require.NoError(t, err)

	// Success case: the data source is not associated with any app any more, hence, this should go through
	_, err = dbAPI.DeleteDataSource(ctx, inDataSourceID, nil)
	require.NoError(t, err)
}

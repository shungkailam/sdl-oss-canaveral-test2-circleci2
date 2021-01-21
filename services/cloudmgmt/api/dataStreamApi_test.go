package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"sync"

	"github.com/stretchr/testify/require"

	"context"
	"fmt"
	"reflect"
	"sort"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createDataStream(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string, categoryID string, categoryValue string, cloudCredsID string, scriptID string) model.DataStream {
	return createDataStreamWithState(t, dbAPI, tenantID, projectID, categoryID, categoryValue, cloudCredsID, scriptID, nil)
}

func createDataStreamWithState(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string, categoryID string, categoryValue string, cloudCredsID string, scriptID string, state *string) model.DataStream {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	doc := generateDataStream(tenantID, categoryID, categoryValue, cloudCredsID, scriptID, projectID, state)

	// create DataStreams
	resp, err := dbAPI.CreateDataStream(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create DataStream successful, %s", resp)
	dataStreamID := resp.(model.CreateDocumentResponse).ID
	dataStream, err := dbAPI.GetDataStream(ctx, dataStreamID)
	require.NoError(t, err)
	return dataStream
}

func generateDataStream(tenantID string, categoryID string, categoryValue string, cloudCredsID string, scriptID string, projectID string, state *string) model.DataStream {
	dataStreamName := "data-streams-name-" + base.GetUUID()
	dataStreamDataType := "Custom"
	return model.DataStream{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:     dataStreamName,
		DataType: dataStreamDataType,
		Origin:   "DataSource",
		OriginSelectors: []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: categoryValue,
			},
		},
		OriginID:         "",
		Destination:      "Cloud",
		CloudType:        "AWS",
		CloudCredsID:     cloudCredsID,
		AWSCloudRegion:   "us-west-2",
		GCPCloudRegion:   "",
		EdgeStreamType:   "",
		AWSStreamType:    "Kafka",
		GCPStreamType:    "",
		Size:             1000000,
		EnableSampling:   false,
		SamplingInterval: 0,
		TransformationArgsList: []model.TransformationArgs{
			{
				TransformationID: scriptID,
				Args:             []model.ScriptParamValue{},
			},
		},
		DataRetention: []model.RetentionInfo{},
		ProjectID:     projectID,
		State:         state,
	}
}

func createDataStreamWithOutIfc(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string, categoryID string, categoryValue string, cloudCredsID string, scriptID string, edgeID string) (model.DataSource, model.DataStream) {
	var size float64 = 1000000
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	streamName := "stream-with-data-out-" + base.GetUUID()
	dataStreamDataType := "Custom"
	topic1 := "test-topic1" + base.GetUUID()
	testDataSource := &model.DataSource{}
	testDataSource.ID = "foo"
	testDataSource.Name = "datasource_name"
	dataSourceIfcInfo := model.DataSourceIfcInfo{Class: "DATAINTERFACE", Kind: "OUT", Protocol: "DATAINTERFACE", Img: "foo", ProjectID: "ingress", DriverID: "bar"}
	testDataSource.IfcInfo = &dataSourceIfcInfo

	// create datastream with out data interfaces
	streamWithdataIfcEndpoint := model.DataStream{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:                   streamName,
		DataType:               dataStreamDataType,
		Origin:                 "DataSource",
		EndPoint:               topic1,
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
		ProjectID:              projectID,
		DataIfcEndpoints:       []model.DataIfcEndpoint{{ID: testDataSource.ID, Name: "does_not_matter", Value: "does_not_matter"}},
	}

	// Create data source
	resp, err := createDataSourceWithSelectorsFields(t, ctx, dbAPI, "data-out-interface-"+base.GetUUID(), tenantID, edgeID, "Model 3", "DATAINTERFACE", dataSourceIfcInfo, nil, nil)
	require.NoError(t, err, "failed to create data source")
	dataSourceID := resp.(model.CreateDocumentResponse).ID
	dataSource, err := dbAPI.GetDataSource(ctx, dataSourceID)
	require.NoError(t, err)
	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{})

	streamWithdataIfcEndpoint.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: dataSourceID, Name: topic1, Value: topic1}}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	resp, err = dbAPI.CreateDataStream(ctx, &streamWithdataIfcEndpoint, func(ctx context.Context, doc interface{}) error {
		ds := doc.(model.DataStream)
		defer wg.Done()
		if len(ds.DataIfcEndpoints) != 1 {
			t.Fatalf("Expected 1 endpoint, but got %d", len(ds.DataIfcEndpoints))
		}
		return nil
	})
	if err != nil {
		wg.Done()
		t.Fatalf("expected nil, but got %s", err.Error())
	}
	wg.Wait()

	createdDataStream, err := dbAPI.GetDataStream(ctx, resp.(model.CreateDocumentResponse).ID)
	if len(createdDataStream.DataIfcEndpoints) != 1 {
		t.Fatalf("expected 1 data Ifc endpoint , but got %d", len(createdDataStream.DataIfcEndpoints))
	}

	if createdDataStream.OutDataIfc == nil {
		t.Fatalf("expected out data Ifc to be set")
	}

	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{topic1})
	return dataSource, createdDataStream
}

func TestDataStream(t *testing.T) {
	t.Parallel()
	t.Log("running TestDataStream test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant1")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileID := dp.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, nil)
	projectID := project.ID
	project2 := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, nil)
	projectID2 := project2.ID
	project3 := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, nil)
	projectID3 := project3.ID
	authContext, _, _ := makeAuthContexts(tenantID, []string{projectID})
	// add proj 2 and 3 to auth context
	authContext.Claims["projects"] = []model.ProjectRole{{
		ProjectID: projectID,
		Role:      model.ProjectRoleAdmin,
	}, {
		ProjectID: projectID2,
		Role:      model.ProjectRoleAdmin,
	}, {
		ProjectID: projectID3,
		Role:      model.ProjectRoleAdmin,
	}}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	scriptRuntime := createScriptRuntime(t, dbAPI, tenantID, projectID, dockerProfileID)
	scriptRuntimeID := scriptRuntime.ID

	script := createScript(t, dbAPI, tenantID, projectID, scriptRuntimeID)
	scriptID := script.ID

	// Teardown
	defer func() {
		dbAPI.DeleteScript(ctx, scriptID, nil)
		dbAPI.DeleteScriptRuntime(ctx, scriptRuntimeID, nil)
		dbAPI.DeleteDockerProfile(ctx, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx, cloudCredsID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteProject(ctx, projectID2, nil)
		dbAPI.DeleteProject(ctx, projectID3, nil)
		dbAPI.DeleteCategory(ctx, categoryID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete DataStream", func(t *testing.T) {
		t.Log("running Create/Get/Delete DataStream test")

		// create script
		sdoc := createScript(t, dbAPI, tenantID, projectID, scriptRuntimeID)
		scriptID := sdoc.ID
		scriptName := sdoc.Name

		// update unused script should be ok
		sdoc.Name = scriptName + "Updated"
		sdoc.ID = scriptID
		sdocW := model.ScriptForceUpdate{Doc: sdoc, ForceUpdate: false}
		resp, err := dbAPI.UpdateScript(ctx, &sdocW, nil)
		require.NoError(t, err)

		// update unused script runtime should be ok
		scriptRuntime.Name = "script runtime updated"
		scriptRuntime.ID = scriptRuntimeID
		resp, err = dbAPI.UpdateScriptRuntime(ctx, &scriptRuntime, nil)
		require.NoError(t, err)

		var size float64 = 1000000
		dataStreamName := "data-streams-name"
		dataStreamName2 := "data-streams-name-2"
		dataStreamName3 := "data-streams-name-3"
		dataStreamDataType := "Custom"
		dataStreamDataTypeUpdated := "Image"

		// DataStream objects, leave ID blank and let create generate it
		doc := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName,
			DataType: dataStreamDataType,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v1",
				},
			},
			OriginID:         "",
			Destination:      "Cloud",
			CloudType:        "AWS",
			CloudCredsID:     cloudCredsID,
			AWSCloudRegion:   "us-west-2",
			GCPCloudRegion:   "",
			EdgeStreamType:   "",
			AWSStreamType:    "Kafka",
			GCPStreamType:    "",
			Size:             size,
			EnableSampling:   false,
			SamplingInterval: 0,
			TransformationArgsList: []model.TransformationArgs{
				{
					TransformationID: scriptID,
					Args:             []model.ScriptParamValue{},
				},
			},
			DataRetention: []model.RetentionInfo{},
			ProjectID:     projectID,
		}
		doc2 := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName2,
			DataType: dataStreamDataType,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v1",
				},
			},
			OriginID:               "",
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID2,
		}
		doc3 := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName3,
			DataType: dataStreamDataType,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v1",
				},
			},
			OriginID:               "",
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID3,
		}

		dataStreams, err := dbAPI.SelectAllDataStreams(ctx, nil)
		require.NoError(t, err)
		if len(dataStreams) != 0 {
			t.Fatal("expect length of data stream to be 0")
		}

		// create DataStreams
		resp, err = dbAPI.CreateDataStream(ctx, &doc, nil)
		require.NoError(t, err)
		dataStreamId := resp.(model.CreateDocumentResponse).ID
		resp, err = dbAPI.CreateDataStream(ctx, &doc2, nil)
		require.NoError(t, err)
		dataStreamId2 := resp.(model.CreateDocumentResponse).ID
		resp, err = dbAPI.CreateDataStream(ctx, &doc3, nil)
		require.NoError(t, err)
		dataStreamId3 := resp.(model.CreateDocumentResponse).ID
		t.Logf("create DataStream successful, id=%s, id2=%s, id3=%s", dataStreamId, dataStreamId2, dataStreamId3)

		// create data stream validation test
		// create must fail if transformation args list contain transformation not in project
		doc2.TransformationArgsList = []model.TransformationArgs{
			{
				TransformationID: scriptID,
				Args:             []model.ScriptParamValue{},
			},
		}
		resp, err = dbAPI.CreateDataStream(ctx, &doc2, nil)
		require.Error(t, err, "expect create stream with inaccessible script id to fail")
		doc.TransformationArgsList = []model.TransformationArgs{
			{
				TransformationID: scriptID,
				Args: []model.ScriptParamValue{{
					ScriptParam: model.ScriptParam{
						Name: "foo",
						Type: "string",
					},
					Value: "bar",
				}},
			},
		}
		resp, err = dbAPI.CreateDataStream(ctx, &doc, nil)
		require.Error(t, err, "expect create stream with mismatch script params to fail")

		dstreamIDs, err := dbAPI.SelectProjectDataStreamsUsingCloudCreds(ctx, tenantID, projectID, []string{cloudCredsID})
		require.NoError(t, err)
		if len(dstreamIDs) != 1 {
			t.Fatalf("expect count of data streams using cloud creds id to be 1, got: %d", len(dstreamIDs))
		}
		if dstreamIDs[0] != dataStreamId {
			t.Fatal("expect data stream using cloud creds id to match")
		}

		// remove in-use cloud profile from project should fail
		savedCCIDs := project.CloudCredentialIDs
		project.CloudCredentialIDs = nil
		_, err = dbAPI.UpdateProject(ctx, &project, nil)
		require.Error(t, err, "expect remove in-use cloud profile from project to fail")
		project.CloudCredentialIDs = savedCCIDs

		scriptIDs := []string{scriptID}
		dsIDs, err := dbAPI.GetDataStreamIDs(ctx, scriptIDs)
		require.NoError(t, err)
		if len(dsIDs) != 1 {
			t.Fatalf("expect to get one data stream ids, got %d", len(dsIDs))
		}

		// update in-use script name should succeed
		sdoc.Name = scriptName
		sdocW = model.ScriptForceUpdate{Doc: sdoc, ForceUpdate: false}
		resp, err = dbAPI.UpdateScript(ctx, &sdocW, nil)
		require.NoError(t, err)

		// update in-use script should fail
		sdoc.Code = fmt.Sprintf("%s\n\n", sdoc.Code)
		sdocW = model.ScriptForceUpdate{Doc: sdoc, ForceUpdate: false}
		resp, err = dbAPI.UpdateScript(ctx, &sdocW, nil)
		require.Error(t, err, "expect update of in-use script to fail")

		// update in-use script runtime should fail
		scriptRuntime.Name = "script runtime"
		resp, err = dbAPI.UpdateScriptRuntime(ctx, &scriptRuntime, nil)
		require.Error(t, err, "expect update of in-use script runtime to fail")

		// update DataStreams
		doc = model.DataStream{
			BaseModel: model.BaseModel{
				ID:       dataStreamId,
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName,
			DataType: dataStreamDataTypeUpdated,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v2",
				},
			},
			OriginID:         "",
			Destination:      "Cloud",
			CloudType:        "AWS",
			CloudCredsID:     cloudCredsID,
			AWSCloudRegion:   "us-west-2",
			GCPCloudRegion:   "",
			EdgeStreamType:   "",
			AWSStreamType:    "Kafka",
			GCPStreamType:    "",
			Size:             size,
			EnableSampling:   false,
			SamplingInterval: 0,
			TransformationArgsList: []model.TransformationArgs{
				{
					TransformationID: scriptID,
					Args:             []model.ScriptParamValue{},
				},
			},
			DataRetention: []model.RetentionInfo{},
			ProjectID:     projectID,
		}
		// get DataStream
		dataStream, err := dbAPI.GetDataStream(ctx, dataStreamId)
		require.NoError(t, err)
		doc2 = model.DataStream{
			BaseModel: model.BaseModel{
				ID:       dataStreamId2,
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName2,
			DataType: dataStreamDataTypeUpdated,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v2",
				},
			},
			OriginID:               "",
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID2,
		}
		// get DataStream
		dataStream2, err := dbAPI.GetDataStream(ctx, dataStreamId2)
		require.NoError(t, err)
		doc3 = model.DataStream{
			BaseModel: model.BaseModel{
				ID:       dataStreamId3,
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName3,
			DataType: dataStreamDataTypeUpdated,
			Origin:   "DataSource",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v2",
				},
			},
			OriginID:               "",
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID3,
		}
		// get DataStream
		dataStream3, err := dbAPI.GetDataStream(ctx, dataStreamId3)
		require.NoError(t, err)
		if dataStream.ID != dataStreamId {
			t.Fatal("wrong data stream 1")
		}
		if dataStream2.ID != dataStreamId2 {
			t.Fatal("wrong data stream 2")
		}
		if dataStream3.ID != dataStreamId3 {
			t.Fatal("wrong data stream 3")
		}
		t.Logf("get DataStream before update successful, %+v", dataStream)

		// select all vs select all W
		var w bytes.Buffer
		dss1, err := dbAPI.SelectAllDataStreams(ctx, nil)
		require.NoError(t, err)
		dss2 := &[]model.DataStream{}
		err = selectAllConverter(ctx, dbAPI.SelectAllDataStreamsW, dss2, &w)
		require.NoError(t, err)
		sort.Sort(model.DataStreamsByID(dss1))
		sort.Sort(model.DataStreamsByID(*dss2))
		if !reflect.DeepEqual(&dss1, dss2) {
			t.Fatalf("expect select data streams and select data streams w results to be equal %+v vs %+v", dss1, *dss2)
		}

		upResp, err := dbAPI.UpdateDataStream(ctx, &doc, nil)
		require.NoError(t, err)
		upResp, err = dbAPI.UpdateDataStream(ctx, &doc2, nil)
		require.NoError(t, err)
		upResp, err = dbAPI.UpdateDataStream(ctx, &doc3, nil)
		require.NoError(t, err)
		t.Logf("update DataStream successful, %+v", upResp)

		// Update to undeploy
		doc3.State = model.UndeployEntityState.StringPtr()
		upResp, err = dbAPI.UpdateDataStream(ctx, &doc3, nil)
		require.NoError(t, err)
		t.Logf("update DataStream successful, %+v", upResp)

		dataStreams, err = dbAPI.SelectAllDataStreams(ctx, nil)
		require.NoError(t, err)
		if len(dataStreams) != 3 {
			t.Fatalf("Expected 3, found %d", len(dataStreams))
		}
		undeployCount := 0
		for _, dataStream = range dataStreams {
			if dataStream.State != nil && *dataStream.State == string(model.UndeployEntityState) {
				undeployCount++
			}
		}
		t.Log("select all DataStream for edge successful")
		if undeployCount != 1 {
			t.Fatalf("Expected undeploy count of 1, found %d", undeployCount)
		}
		// select all data sources for project from edge
		edgeAuthContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      "1234",
				"projects":    authContext.Claims["projects"],
			},
		}
		edgeCtx := context.WithValue(context.Background(), base.AuthContextKey, edgeAuthContext)
		dataStreamsForEdge, err := dbAPI.SelectAllDataStreams(edgeCtx, nil)
		require.NoError(t, err)
		if len(dataStreamsForEdge) != 2 {
			t.Fatalf("Expected 2, found %d \n %+v", len(dataStreams), dataStreamsForEdge)
		}

		for _, dataStream = range dataStreamsForEdge {
			if dataStream.State != nil && *dataStream.State != string(model.DeployEntityState) {
				t.Fatalf("Edge is not expected to see entities not in deploy state")
			}
		}
		t.Log("select all DataStream for edge successful")

		// update data stream validation test
		// update must fail if transformation args list contain transformation not in project
		doc2.TransformationArgsList = []model.TransformationArgs{
			{
				TransformationID: scriptID,
				Args:             []model.ScriptParamValue{},
			},
		}
		upResp, err = dbAPI.UpdateDataStream(ctx, &doc2, nil)
		require.Error(t, err, "expect update stream with inaccessible script id to fail")
		doc.TransformationArgsList = []model.TransformationArgs{
			{
				TransformationID: scriptID,
				Args: []model.ScriptParamValue{
					{
						ScriptParam: model.ScriptParam{
							Name: "foo",
							Type: "string",
						},
						Value: "bar",
					},
				},
			},
		}
		upResp, err = dbAPI.UpdateDataStream(ctx, &doc, nil)
		require.Error(t, err, "expect update stream with mismatch script params to fail")

		// get DataStream
		dataStream, err = dbAPI.GetDataStream(ctx, dataStreamId)
		require.NoError(t, err)
		dataStream2, err = dbAPI.GetDataStream(ctx, dataStreamId2)
		require.NoError(t, err)
		dataStream3, err = dbAPI.GetDataStream(ctx, dataStreamId3)
		require.NoError(t, err)
		t.Logf("get DataStream successful, %+v", dataStream)

		if dataStream.ID != dataStreamId || dataStream.Name != dataStreamName || dataStream.DataType != dataStreamDataTypeUpdated {
			t.Fatal("DataStream data mismatch")
		}
		if dataStream2.ID != dataStreamId2 || dataStream2.Name != dataStreamName2 || dataStream2.DataType != dataStreamDataTypeUpdated {
			t.Fatal("DataStream 2 data mismatch")
		}
		if dataStream3.ID != dataStreamId3 || dataStream3.Name != dataStreamName3 || dataStream3.DataType != dataStreamDataTypeUpdated {
			t.Fatal("DataStream 3 data mismatch")
		}

		dataStreams, err = dbAPI.SelectAllDataStreams(ctx, nil)
		require.NoError(t, err)

		for _, dataStream = range dataStreams {
			testForMarshallability(t, dataStream)
		}
		t.Log("select all DataStream successful")

		// test delete script failure due to in-use
		_, err = dbAPI.DeleteScript(ctx, scriptID, nil)
		require.Error(t, err, "expect deletion of in-use script to fail")

		// delete DataStream
		delResp, err := dbAPI.DeleteDataStream(ctx, dataStreamId, nil)
		require.NoError(t, err)
		delResp, err = dbAPI.DeleteDataStream(ctx, dataStreamId2, nil)
		require.NoError(t, err)
		delResp, err = dbAPI.DeleteDataStream(ctx, dataStreamId3, nil)
		require.NoError(t, err)
		t.Logf("delete DataStream successful, %v", delResp)

		// delete script
		_, err = dbAPI.DeleteScript(ctx, scriptID, nil)
		require.NoError(t, err)
	})

	// select all DataStreams
	t.Run("SelectAllDataStreams", func(t *testing.T) {
		t.Log("running SelectAllDataStreams test")
		dataStreams, err := dbAPI.SelectAllDataStreams(ctx, nil)
		require.NoError(t, err)
		for _, dataStream := range dataStreams {
			testForMarshallability(t, dataStream)
		}
	})

	// Verify origin validation
	t.Run("ValidateOrigin", func(t *testing.T) {
		var size float64 = 1000000
		var dataStreamName1 = "data-streams-name-01"
		var dataStreamName2 = "data-streams-name-02"
		var dataStreamDataType = "Custom"

		t.Log("Test validation of data stream origin")

		doc1 := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName1,
			DataType: dataStreamDataType,
			Origin:   "Data Source",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "v2",
				},
			},
			OriginID:               "",
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID,
		}

		// create DataStreams
		resp, err := dbAPI.CreateDataStream(ctx, &doc1, nil)
		require.NoError(t, err)
		dataStreamId := resp.(model.CreateDocumentResponse).ID

		doc2 := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:                   dataStreamName2,
			DataType:               dataStreamDataType,
			Origin:                 "Data Stream",
			OriginID:               dataStreamId,
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID,
		}
		_, err = dbAPI.CreateDataStream(ctx, &doc2, nil)
		require.Error(t, err, "Expected data stream creation to fail")
		t.Log(err)

		docDataStreamNonExistantCatVal := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:     dataStreamName2,
			DataType: dataStreamDataType,
			Origin:   "Data Source",
			OriginSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: "non-existent",
				},
				{
					ID:    categoryID,
					Value: "v1",
				},
			},
			Destination:            "Cloud",
			CloudType:              "AWS",
			CloudCredsID:           cloudCredsID,
			AWSCloudRegion:         "us-west-2",
			GCPCloudRegion:         "",
			EdgeStreamType:         "",
			AWSStreamType:          "Kafka",
			GCPStreamType:          "",
			Size:                   size,
			EnableSampling:         false,
			SamplingInterval:       0,
			TransformationArgsList: []model.TransformationArgs{},
			DataRetention:          []model.RetentionInfo{},
			ProjectID:              projectID,
		}
		_, err = dbAPI.CreateDataStream(ctx, &docDataStreamNonExistantCatVal, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Record not found error")
		t.Log(err)

		t.Log("Point data stream to Kafka on edge")
		doc1.ID = dataStreamId
		doc1.Destination = "Edge"
		doc1.EdgeStreamType = "Kafka"
		_, err = dbAPI.UpdateDataStream(ctx, &doc1, nil)
		require.Error(t, err, "Expected data stream creation to fail since Kafka not enabled in project")
		t.Log(err)

		_, err = dbAPI.CreateDataStream(ctx, &doc2, nil)
		require.Error(t, err, "Expected data stream creation to fail")
		t.Log(err)

		t.Log("Update data stream to real time stream")
		doc1.EdgeStreamType = "None"
		_, err = dbAPI.UpdateDataStream(ctx, &doc1, nil)
		require.NoError(t, err)

		// data stream creation should succeed now
		_, err = dbAPI.CreateDataStream(ctx, &doc2, nil)
		require.NoError(t, err)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateDataStream(tenantID, categoryID, "v1", cloudCredsID, scriptID, projectID, nil)
		doc.ID = id
		return dbAPI.CreateDataStream(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetDataStream(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteDataStream(ctx, id, nil)
	}))
}

func TestDataStreamDataIfcEndpoint(t *testing.T) {
	streamName := "stream-with-data-out"
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileID := dp.ID

	// Create edge
	edge := createEdge(t, dbAPI, tenantID)
	edge2 := createEdge(t, dbAPI, tenantID)

	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, []string{edge.ID, edge2.ID})
	ctx, _, _ := makeContext(tenantID, []string{project.ID})
	defer func(projectID, cloudCredsID, tenantID string) {
		dbAPI.DeleteCloudCreds(ctx, cloudCredsID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteEdge(ctx, edge.ID, nil)
		dbAPI.DeleteEdge(ctx, edge2.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}(project.ID, cloudCredsID, tenantID)

	// projectID := project.ID
	dataStreamDataType := "Custom"
	var size float64 = 1000000
	topic1, topic2 := "test-topic1"+base.GetUUID(), "test-topic2"+base.GetUUID()
	testDataSource := &model.DataSource{}
	testDataSource.ID = "foo"
	testDataSource.Name = "datasource_name"
	dataSourceIfcInfo := model.DataSourceIfcInfo{Class: "DATAINTERFACE", Kind: "OUT", Protocol: "DATAINTERFACE", Img: "foo", ProjectID: "ingress", DriverID: "bar"}
	testDataSource.IfcInfo = &dataSourceIfcInfo

	// create datastream with out data interfaces
	streamWithdataIfcEndpoint := model.DataStream{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:                   streamName,
		DataType:               dataStreamDataType,
		Origin:                 "DataSource",
		EndPoint:               topic1,
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
		DataIfcEndpoints:       []model.DataIfcEndpoint{{ID: testDataSource.ID, Name: "does_not_matter", Value: "does_not_matter"}},
	}
	resp, err := dbAPI.CreateDataStream(ctx, &streamWithdataIfcEndpoint, nil)
	// expected the create to fail as the data interface has not been created yet
	require.Error(t, err)
	require.Contains(t, err.Error(), "does not exist")

	// Failure case: Name not set on data ifc endpoint
	streamWithdataIfcEndpoint.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: testDataSource.ID, Value: "does_not_matter"}}
	resp, err = dbAPI.CreateDataStream(ctx, &streamWithdataIfcEndpoint, nil)
	// expected the create to fail as the data interface has not been created yet
	require.Error(t, err)
	require.Contains(t, err.Error(), "DataIfcEndpoints[i]/Name")

	// Failure case: More than one data ifc endpoints
	streamWithdataIfcEndpoint.DataIfcEndpoints = []model.DataIfcEndpoint{
		{ID: base.GetUUID(), Name: "does_not_matter", Value: "does_not_matter"},
		{ID: base.GetUUID(), Name: "does_not_matter", Value: "does_not_matter"},
	}
	resp, err = dbAPI.CreateDataStream(ctx, &streamWithdataIfcEndpoint, nil)
	// expected the create to fail as the data interface has not been created yet
	require.Error(t, err)
	require.Contains(t, err.Error(), "maximum, only one data Ifc endpoint is allowed")

	// Create data source
	resp, err = createDataSourceWithSelectorsFields(t, ctx, dbAPI, "data-out-interface-"+base.GetUUID(), tenantID, edge.ID, "Model 3", "DATAINTERFACE", dataSourceIfcInfo, nil, nil)
	require.NoError(t, err, "failed to create data source")
	dataSourceID := resp.(model.CreateDocumentResponse).ID
	resp, err = dbAPI.GetDataSource(ctx, dataSourceID)
	require.NoError(t, err)
	defer func() { dbAPI.DeleteDataSource(ctx, dataSourceID, nil) }()

	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{})

	edgeCtx := makeEdgeContext(tenantID, edge.ID, []string{project.ID})
	edgeCtx2 := makeEdgeContext(tenantID, edge2.ID, []string{project.ID})

	streamWithdataIfcEndpoint.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: dataSourceID, Name: topic1, Value: topic1}}
	wg := &sync.WaitGroup{}
	wg.Add(1)
	resp, err = dbAPI.CreateDataStream(ctx, &streamWithdataIfcEndpoint, func(ctx context.Context, doc interface{}) error {
		ds := doc.(model.DataStream)
		defer wg.Done()
		if len(ds.DataIfcEndpoints) != 1 {
			t.Fatalf("Expected 1 endpoint, but got %d", len(ds.DataIfcEndpoints))
		}
		return nil
	})
	wg.Wait()
	require.NoError(t, err)
	defer func(streamID string) { dbAPI.DeleteDataStream(ctx, streamID, nil) }(resp.(model.CreateDocumentResponse).ID)

	createdDataStream, err := dbAPI.GetDataStream(ctx, resp.(model.CreateDocumentResponse).ID)
	if len(createdDataStream.DataIfcEndpoints) != 1 {
		t.Fatalf("expected 1 data Ifc endpoint , but got %d", len(createdDataStream.DataIfcEndpoints))
	}

	if createdDataStream.OutDataIfc == nil {
		t.Fatalf("expected out data Ifc to be set")
	}

	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{topic1})

	ds1, err := dbAPI.GetDataStream(edgeCtx, resp.(model.CreateDocumentResponse).ID)
	require.NoError(t, err)
	if len(ds1.DataIfcEndpoints) != 1 {
		t.Fatalf("expected 1 data Ifc endpoint , but got %d", len(ds1.DataIfcEndpoints))
	}
	if ds1.OutDataIfc == nil {
		t.Fatalf("expected out data Ifc to be set")
	}

	_, err = dbAPI.GetDataStream(edgeCtx2, resp.(model.CreateDocumentResponse).ID)
	if err == nil {
		// expect not found error
		t.Fatalf("expect get data stream to fail")
	}

	// ctx and edgeCtx will get the data stream, edgeCtx2 will not
	streams, err := dbAPI.SelectAllDataStreams(ctx, nil)
	require.NoError(t, err)
	if len(streams) != 1 {
		t.Fatalf("expect streams count to be 1")
	}
	streams, err = dbAPI.SelectAllDataStreams(edgeCtx, nil)
	require.NoError(t, err)
	if len(streams) != 1 {
		t.Fatalf("expect streams count to be 1")
	}
	streams, err = dbAPI.SelectAllDataStreams(edgeCtx2, nil)
	require.NoError(t, err)
	if len(streams) != 0 {
		t.Fatalf("expect streams count to be 0")
	}

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	defer func() {
		dbAPI.DeleteCategory(ctx, categoryID, nil)
	}()

	// Failure case: Create another pipeline with same data interface and same topic
	stream2 := streamWithdataIfcEndpoint
	stream2.Name = fmt.Sprintf("%s-%s", streamWithdataIfcEndpoint.Name, "2")
	_, err = dbAPI.CreateDataStream(ctx, &stream2, nil)
	// this is expected to fail due to topic already claimed by another pipeline via the name of the endpoint.
	require.Error(t, err)
	require.Contains(t, err.Error(), "Precondition failed")

	// Success case: Update the stream to change the endpoint to "test-topic2".
	createdDataStream.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: dataSourceID, Name: topic2, Value: topic2}}
	_, err = dbAPI.UpdateDataStream(ctx, &createdDataStream, nil)
	require.NoError(t, err)
	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{topic2})

	// Success case: Since the updateDataStream released `test-topic1` in the previous call, this time, stream2 creation should go through
	stream2.DataIfcEndpoints = []model.DataIfcEndpoint{{ID: dataSourceID, Name: topic1, Value: topic1}}
	resp, err = dbAPI.CreateDataStream(ctx, &stream2, nil)
	require.NoError(t, err)
	defer func(streamID string) { dbAPI.DeleteDataStream(ctx, streamID, nil) }(resp.(model.CreateDocumentResponse).ID)
	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{topic1, topic2})

	// Success case: Update the stream to change the destination to be non data Ifc. DataIfc topics should be updated
	createdDataStream.Destination = model.DestinationEdge
	createdDataStream.EdgeStreamType = "MQTT"
	createdDataStream.DataIfcEndpoints = nil
	_, err = dbAPI.UpdateDataStream(ctx, &createdDataStream, nil)
	require.NoError(t, err)
	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{topic1})

	// Make sure that List data stream call returns the correct result
	dataStreams, err := dbAPI.SelectAllDataStreams(ctx, nil)
	require.NoErrorf(t, err, "failed to list data streams")

	expectednumDataOutStreams, actualnumDataOutStreams := 1, 0
	for _, ds := range dataStreams {
		if ds.Destination == model.DestinationDataInterface {
			if ds.OutDataIfc == nil {
				t.Fatalf("expected DataStream/OutDataIfc to be set for streams with destination set to Data Interface")
			}
			if len(ds.DataIfcEndpoints) == 0 {
				t.Fatalf("expected DataIfcEndpoints to be set for streams with destination set to Data Interface")
			}
			actualnumDataOutStreams++
		}
	}

	if expectednumDataOutStreams != actualnumDataOutStreams {
		t.Fatalf("expected %d, but got %d", expectednumDataOutStreams, actualnumDataOutStreams)
	}

	// Delete data stream  such that data source fields get deleted as a result
	callbackCount := 0
	wg.Add(1)
	_, err = dbAPI.DeleteDataStream(ctx, dataStreams[0].ID, func(context.Context, interface{}) error {
		defer wg.Done()
		callbackCount++
		return nil
	})
	require.NoError(t, err)
	wg.Wait()
	if callbackCount != 1 {
		t.Fatalf("expeced delete call back to be called once but, it was called %d", callbackCount)
	}
	assertDataSource(ctx, t, dbAPI, dataSourceID, []string{topic1})

	// Assert that deleting the data source does not cause the 2nd datastream deletion to fail
	callbackCount = 0
	wg.Add(1)
	_, err = dbAPI.DeleteDataSource(ctx, dataSourceID, func(context.Context, interface{}) error {
		defer wg.Done()
		callbackCount++
		return nil
	})
	require.NoError(t, err)

	wg.Wait()
	if callbackCount != 1 {
		t.Fatalf("expeced delete call back to be called once but, it was called %d", callbackCount)
	}

	_, err = dbAPI.DeleteDataStream(ctx, dataStreams[1].ID, nil)
	require.NoError(t, err)

	// Make sure that data streams are deleted
	dataStreams, err = dbAPI.SelectAllDataStreams(ctx, nil)
	require.NoError(t, err)
	if len(dataStreams) != 0 {
		t.Fatalf("expected datastreams to be deleted, but still got %d", len(dataStreams))
	}
}

/* // comment out the test as it is too specialized // TODO FIXME - generalize
func TestDataStreamW(t *testing.T) {
	t.Log("running TestDataStreamW test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	tenantID := "tenant-id-waldot_test"

	ctx1, _, _ := makeContext(tenantID, []string{})

	// select all edges
	t.Run("SelectAllDataStreamsW", func(t *testing.T) {
		t.Log("running SelectAllDataStreamsW test")

		var w bytes.Buffer
		url := url.URL{}
		r := http.Request{URL: &url}
		err := dbAPI.SelectAllDataStreamsW(ctx1, &w, &r)
		require.NoError(t, err)
		p := []model.DataStream{}
		err = json.NewDecoder(&w).Decode(&p)
		require.NoError(t, err)
		t.Logf("got data streams: %+v\n", p)
	})
}
*/

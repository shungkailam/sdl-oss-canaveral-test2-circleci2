package api_test

import (
	"bytes"
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"reflect"
	"sort"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestSensor(t *testing.T) {
	t.Parallel()
	t.Log("running TestSensor test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createEmptyCategoryProject(t, dbAPI, tenantID)
	projectID := project.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	// Teardown
	defer func() {
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Sensor", func(t *testing.T) {
		t.Log("running Create/Get/Delete Sensor test")

		sensorTopicName := "sensor topic"
		sensorTopicNameUpdated := "sensor topic udpated"

		// Sensor object, leave ID blank and let create generate it
		doc := model.Sensor{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:       "",
					TenantID: tenantID,
					Version:  0,
				},
				EdgeID: edgeID,
			},
			TopicName: sensorTopicName,
		}
		// create script
		resp, err := dbAPI.CreateSensor(ctx, &doc, nil)
		require.NoError(t, err)
		t.Logf("create sensor successful, %s", resp)

		sensorId := resp.(model.CreateDocumentResponse).ID

		// update sensor
		doc = model.Sensor{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:       sensorId,
					TenantID: tenantID,
					Version:  0,
				},
				EdgeID: edgeID,
			},
			TopicName: sensorTopicNameUpdated,
		}
		upResp, err := dbAPI.UpdateSensor(ctx, &doc, nil)
		require.NoError(t, err)
		t.Logf("update sensor successful, %+v", upResp)

		// get sensor
		sensor, err := dbAPI.GetSensor(ctx, sensorId)
		require.NoError(t, err)
		t.Logf("get sensor successful, %+v", sensor)

		if sensor.ID != sensorId || sensor.TenantID != tenantID || sensor.TopicName != sensorTopicNameUpdated {
			t.Fatal("sensor data mismatch")
		}

		// select all vs select all W
		var w bytes.Buffer
		sensors1, err := dbAPI.SelectAllSensors(ctx)
		require.NoError(t, err)
		sensors2 := &[]model.Sensor{}
		err = selectAllConverter(ctx, dbAPI.SelectAllSensorsW, sensors2, &w)
		require.NoError(t, err)
		sort.Sort(model.SensorsByID(sensors1))
		sort.Sort(model.SensorsByID(*sensors2))
		if !reflect.DeepEqual(&sensors1, sensors2) {
			t.Fatalf("expect select sensors and select sensors w results to be equal %+v vs %+v", sensors1, *sensors2)
		}

		// delete sensor
		delResp, err := dbAPI.DeleteSensor(ctx, sensorId, nil)
		require.NoError(t, err)
		t.Logf("delete sensor successful, %v", delResp)

	})

	// select all sensors
	t.Run("SelectAllSensors", func(t *testing.T) {
		t.Log("running SelectAllSensors test")
		sensors, err := dbAPI.SelectAllSensors(ctx)
		require.NoError(t, err)
		for _, sensor := range sensors {
			testForMarshallability(t, sensor)
		}
	})

	// select all sensors for edge
	t.Run("SelectAllSensorsForEdge", func(t *testing.T) {
		t.Log("running SelectAllSensorsForEdge test")
		edgeId := "ORD"
		sensors, err := dbAPI.SelectAllSensorsForEdge(ctx, edgeId)
		require.NoError(t, err)
		for _, sensor := range sensors {
			testForMarshallability(t, sensor)
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		return dbAPI.CreateSensor(ctx, &model.Sensor{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:       id,
					TenantID: tenantID,
					Version:  0,
				},
				EdgeID: edgeID,
			},
			TopicName: "",
		}, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetSensor(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteSensor(ctx, id, nil)
	}))
}

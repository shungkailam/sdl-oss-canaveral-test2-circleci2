package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	eventapi "cloudservices/event/api"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvents(t *testing.T) {
	t.Parallel()
	t.Log("running TestEvents test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)

	tenantID := base.GetUUID()
	// Create the tenant
	tenantDoc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test-tenant-" + tenantID,
	}
	projectID1 := "551"
	projectID2 := "552"

	adminCtx, _, projCtx := makeContext(tenantID, []string{projectID1})
	adminAllProjsCtx, _, _ := makeContext(tenantID, []string{projectID1, projectID2})
	_, _, proj2Ctx := makeContext(tenantID, []string{projectID2})
	// create tenant
	resp1, err := dbAPI.CreateTenant(adminCtx, &tenantDoc, nil)
	require.NoError(t, err)
	t.Logf("create tenant successful, %s", resp1)

	// Teardown
	defer func() {
		dbAPI.DeleteTenant(adminCtx, tenantID, nil)
		dbAPI.Close()
	}()
	t.Run("Upsert/Query/Events", func(t *testing.T) {
		t.Log("running Upsert/Query/Eventstest")

		events := []model.Event{
			{
				Timestamp: time.Now().UTC(),
				Type:      "alert",
				Path:      "/serviceDomain:123/datasource:58",
				State:     "yellow",
				Message:   "{test}",
			},
			{
				Timestamp: time.Now().UTC(),
				Type:      "alert",
				Path:      "/serviceDomain:124/datasource:58",
				State:     "yellow",
				Message:   "{test}",
			},
			{
				Timestamp: time.Now().UTC(),
				Type:      "alert",
				Path:      fmt.Sprintf("/serviceDomain:124/project:%s/application:60", projectID1),
				State:     "yellow",
				Message:   "{test}",
			},
			{
				Timestamp: time.Now().UTC(),
				Type:      "alert",
				Path:      fmt.Sprintf("/serviceDomain:124/project:%s/application:90", projectID2),
				State:     "yellow",
				Message:   "{test}",
			},
			{
				Timestamp: time.Now().UTC(),
				Type:      "alert",
				Path:      fmt.Sprintf("/serviceDomain:124/project:%s/service:istio/instance:122/binding:145/status", projectID2),
				State:     "yellow",
				Message:   "{test}",
			},
		}
		event := model.EventUpsertRequest{
			Events: events,
		}
		_, err := dbAPI.UpsertEvents(adminAllProjsCtx, event, nil)
		require.NoError(t, err)

		filter := model.EventFilter{
			Path: "/serviceDomain:123",
		}
		result, err := dbAPI.QueryEvents(adminCtx, filter)
		require.NoError(t, err)

		if len(result) != 1 {
			t.Fatalf("Mismatch in count. Expected 1 but got %d", len(result))
		}
		for i, r := range result {
			t.Log("Event:", r)
			if !strings.HasPrefix(result[i].Path, "/serviceDomain:123") {
				t.Fatalf("Mismatch in count. Expected /serviceDomain:123 prefix but got %s", result[i].Path)
			}
			require.Equal(t, eventapi.InfraEventAudience, result[i].Audience, "Audience mismatched for %s", result[i].Path)
		}
		for _, r := range result {
			t.Log("Event:", r)
		}
		t.Log("query 1 events successful", result)

		filter = model.EventFilter{
			Path: "/serviceDomain:124",
		}
		// As Infra admin
		result, err = dbAPI.QueryEvents(adminCtx, filter)
		require.NoError(t, err)

		if len(result) != 3 {
			t.Fatalf("Mismatch in count. Expected 3 but got %d", len(result))
		}
		infraCount := 0
		projectCount := 0
		infraProjectCount := 0
		for i, r := range result {
			t.Log("Event:", r)
			if !strings.HasPrefix(result[i].Path, "/serviceDomain:124") {
				t.Fatalf("Mismatch in count. Expected /serviceDomain:124 prefix but got %s", result[0].Path)
			}
			if result[i].Audience == eventapi.InfraEventAudience {
				infraCount++
			} else if result[i].Audience == eventapi.ProjectEventAudience {
				projectCount++
			} else if result[i].Audience == eventapi.InfraProjectEventAudience {
				infraProjectCount++
			}
		}
		require.Equal(t, 1, infraCount, "Expected 1 infra event but found %d", infraCount)
		require.Equal(t, 1, projectCount, "Expected 1 project event but found %d", projectCount)
		require.Equal(t, 1, infraProjectCount, "Expected 1 infra project event but found %d", infraProjectCount)
		t.Log("query 2 events successful", result)

		// User v1.0 using REST endpoints
		filter = model.EventFilter{
			Path: "/serviceDomain:124",
		}
		data, _ := json.Marshal(filter)
		request, err := apitesthelper.NewHTTPRequest("POST", "/v1.0/events", ioutil.NopCloser(bytes.NewBuffer(data)))
		require.NoError(t, err)
		writer := &bytes.Buffer{}
		err = dbAPI.QueryEventsW(adminCtx, writer, request)
		require.NoError(t, err)

		response := []model.Event{}
		err = json.Unmarshal(writer.Bytes(), &response)
		require.NoError(t, err)
		if len(response) != 3 {
			t.Fatalf("Mismatch in count. Expected 3 but got %d", len(response))
		}
		// application 90 is treated as infra
		// and is viewable by this infra user even if the project does not belong to
		// this user
		expectedEvents := map[string]string{
			"/serviceDomain:123/datasource:58":                                                                     eventapi.InfraEventAudience,
			"/serviceDomain:124/datasource:58":                                                                     eventapi.InfraEventAudience,
			fmt.Sprintf("/serviceDomain:124/project:%s/application:60", projectID1):                                eventapi.ProjectEventAudience,
			fmt.Sprintf("/serviceDomain:124/project:%s/application:90", projectID2):                                eventapi.ProjectEventAudience,
			fmt.Sprintf("/serviceDomain:124/project:%s/service:istio/instance:122/binding:145/status", projectID2): eventapi.InfraProjectEventAudience,
		}
		for _, event := range response {
			audience, ok := expectedEvents[event.Path]
			if !ok {
				t.Fatalf("Expected one of %+v, found %s", expectedEvents, event.Path)
			}
			require.Equal(t, audience, event.Audience, "Audience mismatched for %s", event.Path)
		}
		t.Log("query 3 events successful", response)

		// Query with pageIndex and pageSize
		request, err = apitesthelper.NewHTTPRequest("POST", "/v1.0/events?pageIndex=1&pageSize=1", ioutil.NopCloser(bytes.NewBuffer(data)))
		require.NoError(t, err)

		writer = &bytes.Buffer{}
		err = dbAPI.QueryEventsW(adminCtx, writer, request)
		require.NoError(t, err)

		response = []model.Event{}
		err = json.Unmarshal(writer.Bytes(), &response)
		require.NoError(t, err)
		if len(response) != 1 {
			t.Fatalf("Mismatch in count. Expected 1 but got %d", len(response))
		}
		for i, r := range response {
			if !strings.HasPrefix(response[i].Path, "/serviceDomain:124") {
				t.Fatalf("Mismatch in count. Expected /serviceDomain:124 prefix but got %s", response[i].Path)
			}
			t.Log("Event:", r)
		}
		t.Log("query 4 events successful", response)

		request, err = apitesthelper.NewHTTPRequest("POST", "/v1.0/events", ioutil.NopCloser(bytes.NewBuffer(data)))
		require.NoError(t, err)
		writer = &bytes.Buffer{}
		// Project context
		err = dbAPI.QueryEventsW(projCtx, writer, request)
		require.NoError(t, err)

		response = []model.Event{}
		err = json.Unmarshal(writer.Bytes(), &response)
		require.NoError(t, err)
		if len(response) != 1 {
			t.Fatalf("Mismatch in count. Expected 1 but got %d", len(response))
		}
		// Only application 60 must be found
		// application 90 is raised as infra
		expectedEvent := fmt.Sprintf("/serviceDomain:124/project:%s/application:60", projectID1)
		require.Equal(t, expectedEvent, response[0].Path, "Path mismatched for %s", response[0].Path)
		require.Equal(t, eventapi.ProjectEventAudience, response[0].Audience, "Audience mismatched for %s", response[0].Path)
		t.Log("query 5 events successful", response)

		request, err = apitesthelper.NewHTTPRequest("POST", "/v1.0/events", ioutil.NopCloser(bytes.NewBuffer(data)))
		require.NoError(t, err)
		writer = &bytes.Buffer{}
		// Project 2 context
		err = dbAPI.QueryEventsW(proj2Ctx, writer, request)
		require.NoError(t, err)

		response = []model.Event{}
		err = json.Unmarshal(writer.Bytes(), &response)
		require.NoError(t, err)
		if len(response) != 2 {
			t.Fatalf("Mismatch in count. Expected 2 but got %d", len(response))
		}
		// application:90 has project audience
		// binding:145 has both project and admin audiences.
		// admin check is done above
		expectedEvents = map[string]string{
			fmt.Sprintf("/serviceDomain:124/project:%s/application:90", projectID2):                                eventapi.ProjectEventAudience,
			fmt.Sprintf("/serviceDomain:124/project:%s/service:istio/instance:122/binding:145/status", projectID2): eventapi.InfraProjectEventAudience,
		}
		for _, event := range response {
			audience, ok := expectedEvents[event.Path]
			if !ok {
				t.Fatalf("Expected one of %+v, found %s", expectedEvents, event.Path)
			}
			require.Equal(t, audience, event.Audience, "Audience mismatched for %s", event.Path)
		}
		t.Log("query 6 events successful", response)
	})
}

type testStruct struct {
	inputEvents           []model.Event
	svcDomainPushTSEvents []model.Event
	latestVersion         time.Time
	outputEvents          []model.Event
}

// Test filter events
func TestFilterEvents(t *testing.T) {
	t.Parallel()
	t.Log("running TestFilterEvents test")

	nowTS := time.Now().UTC()
	// time.String(). This is what the edge sends
	latestTSstr := "2019-04-09 07:27:59.199999 -0000 UTC m=+0.000850568"
	latestTS, err := api.ParseStringTime(context.TODO(), latestTSstr)
	require.NoError(t, err, "Timestamp conversion failed")

	oldTSstr := "2019-04-09 07:27:59.099999 -0000 UTC m=+0.000850568"

	latestTSstr2 := "2019-06-05 06:56:28.94604 -0000 UTC m=+0.000850568"
	// latestTSstr2 used to test backward compatibility of 1 micro sec time
	// difference in updated timestamps
	latestTS2, err := api.ParseStringTime(context.TODO(), latestTSstr2)
	require.NoError(t, err, "Timestamp conversion failed")

	testCases := []testStruct{
		// dataSource
		{
			inputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/dataSource:58/topic:tid/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"sourceVersion": latestTSstr,
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/dataSource:58/topic:tid/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"sourceVersion": oldTSstr,
					},
				},
			},
			latestVersion: latestTS,
			outputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/dataSource:58/topic:tid/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"sourceVersion": latestTSstr,
					},
				},
			},
		},

		// Stream
		{
			inputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/dataPipeline:42/pod:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"streamVersion": latestTSstr,
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/dataPipeline:42/pod:sherlock/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"streamVersion": oldTSstr,
					},
				},
			},
			latestVersion: latestTS,
			outputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/dataPipeline:42/pod:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"streamVersion": latestTSstr,
					},
				},
			},
		},

		// Project
		{
			inputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"projectVersion": latestTSstr,
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"projectVersion": oldTSstr,
					},
				},
			},
			latestVersion: latestTS,
			outputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"projectVersion": latestTSstr,
					},
				},
			},
		},

		// Application
		{
			inputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/container:sherlock/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"applicationVersion": oldTSstr,
					},
				},
			},
			latestVersion: latestTS,
			outputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
					},
				},
			},
		},

		// Application with pushTimeStamp
		{
			inputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/pod:1/container:sherlock/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"applicationVersion": oldTSstr,
						"pushTimeStamp":      "12345677", // cloud and edge both updated the app
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/pod:2/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": oldTSstr, // cloud updated the app but edge is still pulling new images
						"pushTimeStamp":      "12345678",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/pod:4/container:sherlock/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"applicationVersion": oldTSstr,
						"pushTimeStamp":      "12345677",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/pod:5/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "12345678",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/project:42/application:42/pod:4/container:sherlock/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"applicationVersion": oldTSstr,
						"pushTimeStamp":      "123455", // diff pushTS from svcDomain pushTS
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/project:42/application:42/pod:5/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "123456", // same pushts as svcDomain
					},
				},
				{ // Not running on edge
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/project:42/application:41/pod:6/container:sherlock/status",
					State:     "yellow",
					Message:   "stale",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "123455",
					},
				},
				{ // running on edge
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/project:42/application:42/pod:7/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "123456",
					},
				},
			},
			svcDomainPushTSEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/clusterPushTimeStamp",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"pushTimeStamp": "12345678",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/clusterPushTimeStamp",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"pushTimeStamp": "123456",
					},
				},
			},
			latestVersion: latestTS,
			outputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/pod:2/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": oldTSstr,
						"pushTimeStamp":      "12345678",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:123/project:42/application:42/pod:5/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "12345678",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/project:42/application:42/pod:5/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "123456",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "alert",
					Path:      "/serviceDomain:456/project:42/application:42/pod:7/container:sherlock/status",
					State:     "yellow",
					Message:   "latest",
					Properties: map[string]string{
						"applicationVersion": latestTSstr,
						"pushTimeStamp":      "123456",
					},
				},
			},
		},

		// 1 micro sec diff
		{
			inputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "ALERT",
					Path:      "/serviceDomain:123/project:a06f5c60-50f0-4532-b3a9-2ca72bca509f/application:42/pod:nginx-deploy-b4b8684bb-pfk6z/status",
					State:     "",
					Message:   "Successfully assigned project-a06f5c60-50f0-4532-b3a9-2ca72bca509f/nginx-deploy-b4b8684bb-pfk6z to gr-test-1-1",
					Properties: map[string]string{
						"applicationVersion": "2019-06-05T06:56:28.946039Z",
						"context":            "Scheduled",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "ALERT",
					Path:      "/serviceDomain:123/project:a06f5c60-50f0-4532-b3a9-2ca72bca509f/application:42/pod:nginx-deploy-b4b8684bb-pfk6z/status",
					State:     "",
					Message:   "Successfully assigned project-a06f5c60-50f0-4532-b3a9-2ca72bca509f/nginx-deploy-b4b8684bb-pfk6z to gr-test-1-1",
					Properties: map[string]string{
						"applicationVersion": "2019-06-05T06:56:28.946040Z",
						"context":            "Scheduled",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "ALERT",
					Path:      "/serviceDomain:123/project:a06f5c60-50f0-4532-b3a9-2ca72bca509f/application:42/pod:nginx-deploy-b4b8684bb-pfk6z/status",
					State:     "",
					Message:   "Successfully assigned project-a06f5c60-50f0-4532-b3a9-2ca72bca509f/nginx-deploy-b4b8684bb-pfk6z to gr-test-1-1",
					Properties: map[string]string{
						"applicationVersion": "2019-06-05T06:56:28.946035Z",
						"context":            "Scheduled",
					},
				},
			},
			latestVersion: latestTS2,
			outputEvents: []model.Event{
				{
					Timestamp: nowTS,
					Type:      "ALERT",
					Path:      "/serviceDomain:123/project:a06f5c60-50f0-4532-b3a9-2ca72bca509f/application:42/pod:nginx-deploy-b4b8684bb-pfk6z/status",
					State:     "",
					Message:   "Successfully assigned project-a06f5c60-50f0-4532-b3a9-2ca72bca509f/nginx-deploy-b4b8684bb-pfk6z to gr-test-1-1",
					Properties: map[string]string{
						"applicationVersion": "2019-06-05T06:56:28.946039Z",
						"context":            "Scheduled",
					},
				},
				{
					Timestamp: nowTS,
					Type:      "ALERT",
					Path:      "/serviceDomain:123/project:a06f5c60-50f0-4532-b3a9-2ca72bca509f/application:42/pod:nginx-deploy-b4b8684bb-pfk6z/status",
					State:     "",
					Message:   "Successfully assigned project-a06f5c60-50f0-4532-b3a9-2ca72bca509f/nginx-deploy-b4b8684bb-pfk6z to gr-test-1-1",
					Properties: map[string]string{
						"applicationVersion": "2019-06-05T06:56:28.946040Z",
						"context":            "Scheduled",
					},
				},
			},
		},
	}

	selectApps := func(context context.Context) ([]model.Application, error) {
		// svcid - appid
		// 123 - 42
		// 456 - 42
		// 789 - 41
		return []model.Application{
			// App 42
			model.Application{
				BaseModel: model.BaseModel{
					ID: "42",
				},
				ApplicationCore: model.ApplicationCore{
					EdgeIDs: []string{"123", "456"},
				},
			},
			// App 41
			model.Application{
				BaseModel: model.BaseModel{
					ID: "41",
				},
				ApplicationCore: model.ApplicationCore{
					EdgeIDs: []string{"789"},
				},
			},
		}, nil
	}

	for testcasenum, testCase := range testCases {
		filterHelper := func(ctx context.Context, objIDs []string, queryX string, mapUUIDVer map[string]time.Time) {
			for _, id := range objIDs {
				mapUUIDVer[id] = testCase.latestVersion
			}
		}

		evs := api.FilterEventsForLatestVersions(context.Background(), testCase.inputEvents, testCase.svcDomainPushTSEvents, filterHelper, selectApps)

		if len(evs) != len(testCase.outputEvents) {
			t.Fatalf("FilterEventsForLatestVersions Testcase %d Failed. Expected %d but got %d: Events returned: %v", testcasenum, len(testCase.outputEvents), len(evs), evs)
		}

		for i, ev := range evs {
			if !reflect.DeepEqual(ev, testCase.outputEvents[i]) {
				t.Fatalf("FilterEventsForLatestVersions Testcase %d Failed. Expected %v \nBut got %v", testcasenum, testCase.outputEvents[i], ev)
			}
		}

	}
}

func TestSvcDomainPushTimeStamp(t *testing.T) {
	nowTS := time.Now().UTC()
	latestTSstr := "2019-04-09T07:27:59.199999Z"
	input := []model.Event{
		{
			Timestamp: nowTS,
			Type:      "alert",
			Path:      "/serviceDomain:123/pushtimestamp",
			State:     "yellow",
			Message:   "latest",
			Properties: map[string]string{
				"sourceVersion": latestTSstr,
			},
		},
		{
			Timestamp: nowTS,
			Type:      "alert",
			Path:      "/serviceDomain:123/pushtimestamp",
			State:     "yellow",
			Message:   "latest",
			Properties: map[string]string{
				"sourceVersion": latestTSstr,
				"pushTimeStamp": "2019-11-25 22:54:52.409030051 +0000 UTC m=+2117.936259123",
			},
		},
	}

	output := make(map[string]string)
	output["123"] = "2019-11-25 22:54:52.409030051 +0000 UTC m=+2117.936259123"

	svcTSmap := api.GetSvcDomainPushTimestampsMap(input)
	for k, v := range output {
		if svcTSmap[k] == v {
			delete(svcTSmap, k)
			delete(output, k)
		}
	}

	if !(len(svcTSmap) == 0 && len(output) == 0) {
		t.Fatalf("Expected maps to be empty. But found %v and %v", svcTSmap, output)
	}
}

func TestParseStringTime(t *testing.T) {
	strTime := time.Now().String()
	tm, err := api.ParseStringTime(context.TODO(), strTime)
	require.NoError(t, err)
	t.Logf("Time: %v", tm)
	strTime = time.Now().Format(time.RFC3339)
	tm, err = api.ParseStringTime(context.TODO(), strTime)
	require.NoError(t, err)
	t.Logf("Time: %v", tm)
	strTime = "2020-05-29 22:20:43.737604313 +0000 UTC"
	tm, err = api.ParseStringTime(context.TODO(), strTime)
	require.NoError(t, err)
	t.Logf("Time: %v", tm)
}

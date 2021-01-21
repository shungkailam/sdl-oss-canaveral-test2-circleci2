package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestApplicationStatus(t *testing.T) {
	t.Parallel()
	t.Log("running TestApplicationStatus test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeID := edge.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	authContext, authContext2, authContext3 := makeContext(tenantID, []string{projectID})
	app := createApplication(t, dbAPI, tenantID, "app name", projectID, nil, nil)
	appID := app.ID

	// Teardown
	defer func() {
		dbAPI.DeleteApplication(authContext, appID, nil)
		dbAPI.DeleteProject(authContext, projectID, nil)
		dbAPI.DeleteEdge(authContext, edgeID, nil)
		dbAPI.DeleteCategory(authContext, categoryID, nil)
		dbAPI.DeleteTenant(authContext, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete ApplicationStatus", func(t *testing.T) {
		t.Log("running Create/Get/Delete ApplicationStatus test")

		podStatus := make(map[string]interface{})
		podStatusString := `{"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{"kubernetes.io/created-by":"{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicaSet\",\"namespace\":\"shyan\",\"name\":\"cloudmgmt-deployment-86dd869d98\",\"uid\":\"dc80efcf-7338-11e8-9df7-06df7cc3bc4e\",\"apiVersion\":\"extensions\",\"resourceVersion\":\"10914914\"}}\n"},"creationTimestamp":"2018-06-18T20:47:49Z","generateName":"cloudmgmt-deployment-86dd869d98-","labels":{"app":"cloudmgmt","pod-template-hash":"4288425854"},"name":"cloudmgmt-deployment-86dd869d98-j9676","namespace":"shyan","ownerReferences":[{"apiVersion":"extensions/v1beta1","blockOwnerDeletion":true,"controller":true,"kind":"ReplicaSet","name":"cloudmgmt-deployment-86dd869d98","uid":"dc80efcf-7338-11e8-9df7-06df7cc3bc4e"}],"resourceVersion":"10914936","selfLink":"/api/v1/namespaces/shyan/pods/cloudmgmt-deployment-86dd869d98-j9676","uid":"dc87d927-7338-11e8-9df7-06df7cc3bc4e"},"spec":{"containers":[{"command":["sh","-c","exec /usr/src/app/cloudmgmt --sql_db=$SQL_DB --sql_host=$SQL_HOST --sql_password=$SQL_PASSWORD --contentdir=$CONTENTDIR --logtostderr -v 3"],"env":[{"name":"AWS_ACCESS_KEY_ID","valueFrom":{"secretKeyRef":{"key":"accessKeyId","name":"aws-secret"}}},{"name":"AWS_SECRET_ACCESS_KEY","valueFrom":{"secretKeyRef":{"key":"secretAccessKey","name":"aws-secret"}}},{"name":"SQL_DB","valueFrom":{"configMapKeyRef":{"key":"db","name":"db-config"}}},{"name":"SQL_HOST","valueFrom":{"configMapKeyRef":{"key":"dbHost","name":"db-config"}}},{"name":"SQL_PASSWORD","valueFrom":{"secretKeyRef":{"key":"sqlPassword","name":"aws-secret"}}}],"image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imagePullPolicy":"IfNotPresent","name":"cloudmgmt","ports":[{"containerPort":8080,"protocol":"TCP"}],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","volumeMounts":[{"mountPath":"/var/run/secrets/kubernetes.io/serviceaccount","name":"default-token-zmnxs","readOnly":true}]}],"dnsPolicy":"ClusterFirst","nodeName":"ip-172-31-64-135.us-west-2.compute.internal","restartPolicy":"Always","schedulerName":"default-scheduler","securityContext":{},"serviceAccount":"default","serviceAccountName":"default","terminationGracePeriodSeconds":30,"tolerations":[{"effect":"NoExecute","key":"node.alpha.kubernetes.io/notReady","operator":"Exists","tolerationSeconds":300},{"effect":"NoExecute","key":"node.alpha.kubernetes.io/unreachable","operator":"Exists","tolerationSeconds":300}],"volumes":[{"name":"default-token-zmnxs","secret":{"defaultMode":420,"secretName":"default-token-zmnxs"}}]},"status":{"conditions":[{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"Initialized"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:58Z","status":"True","type":"Ready"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"PodScheduled"}],"containerStatuses":[{"containerID":"docker://4f479d7c43b5c6702caf2a53201c6004740a521ad7552a50cf080b3b3f1d647a","image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imageID":"docker-pullable://770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev@sha256:9e3aa147c3a0d12f6cde7f53be077d2dc8b78570ac051df2dd5f1bf0518316cf","lastState":{},"name":"cloudmgmt","ready":true,"restartCount":0,"state":{"running":{"startedAt":"2018-06-18T20:47:57Z"}}}],"hostIP":"172.31.64.135","phase":"Running","podIP":"100.96.2.153","qosClass":"BestEffort","startTime":"2018-06-18T20:47:49Z"}}`

		err := json.Unmarshal([]byte(podStatusString), &podStatus)
		require.NoError(t, err)

		podMetrics := make(map[string]interface{})
		podMetricsString := `{
			"metadata": {
			  "name": "qot-appmqtt-python",
			  "namespace": "c164e9af-3f0d-4b1b-83d4-3defb4301803",
			  "selfLink": "/apis/metrics.k8s.io/v1beta1/namespaces/c164e9af-3f0d-4b1b-83d4-3defb4301803/pods/qot-appmqtt-python",
			  "creationTimestamp": "2018-07-02T23:38:30Z"
			},
			"timestamp": "2018-07-02T23:38:00Z",
			"window": "1m0s",
			"containers": [
			  {
				"name": "qot-sync-service",
				"usage": {
				  "cpu": "0",
				  "memory": "3944Ki"
				}
			  },
			  {
				"name": "qot-timeline-service",
				"usage": {
				  "cpu": "0",
				  "memory": "2080Ki"
				}
			  },
			  {
				"name": "qot-helloworld-mqtt",
				"usage": {
				  "cpu": "0",
				  "memory": "20060Ki"
				}
			  },
			  {
				"name": "dummy-mqtt-actor",
				"usage": {
				  "cpu": "0",
				  "memory": "10888Ki"
				}
			  },
			  {
				"name": "dummy-mqtt-sensor",
				"usage": {
				  "cpu": "0",
				  "memory": "10720Ki"
				}
			  }
			]
		  }`

		err = json.Unmarshal([]byte(podMetricsString), &podMetrics)
		require.NoError(t, err)

		doc := model.ApplicationStatus{
			TenantID:      tenantID,
			EdgeID:        edgeID,
			ApplicationID: appID,
			AppStatus: model.AppStatus{
				PodStatusList: []model.PodStatus{
					podStatus,
				},
				PodMetricsList: []model.PodMetrics{
					podMetrics,
				},
			},
		}

		// create application status
		_, err = dbAPI.CreateApplicationStatus(authContext, &doc, nil)
		require.Error(t, err, "Must fail for users")
		edgeCtx := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
			},
		})
		resp, err := dbAPI.CreateApplicationStatus(edgeCtx, &doc, nil)
		require.NoError(t, err)
		t.Logf("create application status successful, %s", resp)

		data, err := json.Marshal(doc)
		require.NoError(t, err)
		w := httptest.NewRecorder()
		r := strings.NewReader(string(data))
		err = dbAPI.CreateApplicationStatusW(authContext, w, r, nil)
		require.Error(t, err, "Must fail for users")
		w = httptest.NewRecorder()
		r = strings.NewReader(string(data))
		err = dbAPI.CreateApplicationStatusW(edgeCtx, w, r, nil)
		require.NoError(t, err)
		// get application status
		applications, err := dbAPI.SelectAllApplicationsStatus(authContext, true)
		require.NoError(t, err)
		if len(applications) != 1 {
			t.Fatalf("Unexpected app status count %d", len(applications))
		}
		for _, application := range applications {
			testForMarshallability(t, application)
		}

		applications, err = dbAPI.SelectAllApplicationsStatus(authContext2, true)
		require.NoError(t, err)
		if len(applications) != 0 {
			t.Fatalf("Unexpected app status 2 count %d", len(applications))
		}

		applications, err = dbAPI.SelectAllApplicationsStatus(authContext3, true)
		require.NoError(t, err)
		if len(applications) != 1 {
			t.Fatalf("Unexpected app status 3 count %d", len(applications))
		}

		// delete application status
		delResp, err := dbAPI.DeleteApplicationStatus(authContext, appID, nil)
		require.NoError(t, err)
		t.Logf("delete application successful, %v", delResp)

	})

	t.Run("ApplicationStatusConversion", func(t *testing.T) {
		t.Log("running ApplicationStatusConversion test")

		edgeID := "edge-id"
		applicationID := "app-id"
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")

		podStatus := make(map[string]interface{})
		podStatusString := `{"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{"kubernetes.io/created-by":"{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicaSet\",\"namespace\":\"shyan\",\"name\":\"cloudmgmt-deployment-86dd869d98\",\"uid\":\"dc80efcf-7338-11e8-9df7-06df7cc3bc4e\",\"apiVersion\":\"extensions\",\"resourceVersion\":\"10914914\"}}\n"},"creationTimestamp":"2018-06-18T20:47:49Z","generateName":"cloudmgmt-deployment-86dd869d98-","labels":{"app":"cloudmgmt","pod-template-hash":"4288425854"},"name":"cloudmgmt-deployment-86dd869d98-j9676","namespace":"shyan","ownerReferences":[{"apiVersion":"extensions/v1beta1","blockOwnerDeletion":true,"controller":true,"kind":"ReplicaSet","name":"cloudmgmt-deployment-86dd869d98","uid":"dc80efcf-7338-11e8-9df7-06df7cc3bc4e"}],"resourceVersion":"10914936","selfLink":"/api/v1/namespaces/shyan/pods/cloudmgmt-deployment-86dd869d98-j9676","uid":"dc87d927-7338-11e8-9df7-06df7cc3bc4e"},"spec":{"containers":[{"command":["sh","-c","exec /usr/src/app/cloudmgmt --sql_db=$SQL_DB --sql_host=$SQL_HOST --sql_password=$SQL_PASSWORD --contentdir=$CONTENTDIR --logtostderr -v 3"],"env":[{"name":"AWS_ACCESS_KEY_ID","valueFrom":{"secretKeyRef":{"key":"accessKeyId","name":"aws-secret"}}},{"name":"AWS_SECRET_ACCESS_KEY","valueFrom":{"secretKeyRef":{"key":"secretAccessKey","name":"aws-secret"}}},{"name":"SQL_DB","valueFrom":{"configMapKeyRef":{"key":"db","name":"db-config"}}},{"name":"SQL_HOST","valueFrom":{"configMapKeyRef":{"key":"dbHost","name":"db-config"}}},{"name":"SQL_PASSWORD","valueFrom":{"secretKeyRef":{"key":"sqlPassword","name":"aws-secret"}}}],"image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imagePullPolicy":"IfNotPresent","name":"cloudmgmt","ports":[{"containerPort":8080,"protocol":"TCP"}],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","volumeMounts":[{"mountPath":"/var/run/secrets/kubernetes.io/serviceaccount","name":"default-token-zmnxs","readOnly":true}]}],"dnsPolicy":"ClusterFirst","nodeName":"ip-172-31-64-135.us-west-2.compute.internal","restartPolicy":"Always","schedulerName":"default-scheduler","securityContext":{},"serviceAccount":"default","serviceAccountName":"default","terminationGracePeriodSeconds":30,"tolerations":[{"effect":"NoExecute","key":"node.alpha.kubernetes.io/notReady","operator":"Exists","tolerationSeconds":300},{"effect":"NoExecute","key":"node.alpha.kubernetes.io/unreachable","operator":"Exists","tolerationSeconds":300}],"volumes":[{"name":"default-token-zmnxs","secret":{"defaultMode":420,"secretName":"default-token-zmnxs"}}]},"status":{"conditions":[{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"Initialized"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:58Z","status":"True","type":"Ready"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"PodScheduled"}],"containerStatuses":[{"containerID":"docker://4f479d7c43b5c6702caf2a53201c6004740a521ad7552a50cf080b3b3f1d647a","image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imageID":"docker-pullable://770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev@sha256:9e3aa147c3a0d12f6cde7f53be077d2dc8b78570ac051df2dd5f1bf0518316cf","lastState":{},"name":"cloudmgmt","ready":true,"restartCount":0,"state":{"running":{"startedAt":"2018-06-18T20:47:57Z"}}}],"hostIP":"172.31.64.135","phase":"Running","podIP":"100.96.2.153","qosClass":"BestEffort","startTime":"2018-06-18T20:47:49Z"}}`

		err := json.Unmarshal([]byte(podStatusString), &podStatus)
		require.NoError(t, err)

		applications := []model.ApplicationStatus{
			{
				TenantID:      tenantID,
				EdgeID:        edgeID,
				ApplicationID: applicationID,
				Version:       5,
				CreatedAt:     now,
				UpdatedAt:     now,
				AppStatus: model.AppStatus{
					PodStatusList: []model.PodStatus{
						podStatus,
					},
				},
			},
		}

		for _, app := range applications {
			appDBO := api.ApplicationStatusDBO{}
			app2 := model.ApplicationStatus{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})

}

/* // comment out this test as it is too specialized // TODO - generalize this
func TestApplicationStatusByAppID(t *testing.T) {

	t.Log("running TestApplicationStatusByAppID test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	tenantID := "tenant-id-waldot_test"
	projectID := "453a8f34-3ecb-47d6-9516-edc8a5406b77"
	projectID2 := "a7c2ee5acf42a870579fb80812b723b0"
	applicationID := "ac86c719-6840-4e16-8ae3-bd29d9f9370f"
	ctx1, _, _ := makeContext(tenantID, []string{projectID, projectID2})

	// Teardown
	defer func() {
		dbAPI.Close()
	}()

	t.Run("GetApplicationStatusByAppID", func(t *testing.T) {
		t.Log("running GetApplicationStatusByAppID test")

		var w bytes.Buffer
		url := url.URL{
			RawQuery: fmt.Sprintf("pageIndex=%d&pageSize=%d", 0, 60),
		}
		r := http.Request{URL: &url}

		err = dbAPI.SelectAllApplicationsStatusW(ctx1, &w, &r)
		require.NoError(t, err)
		appStatuses := []model.ApplicationStatus{}
		err = json.NewDecoder(&w).Decode(&appStatuses)
		require.NoError(t, err)
		t.Logf(">>> Got app %d statuses\n", len(appStatuses))

		err = dbAPI.GetApplicationStatusWV2(ctx1, applicationID, &w, &r)
		require.NoError(t, err)
		p := model.ApplicationStatusListPayload{
			EntityListResponsePayload: model.EntityListResponsePayload{},
			Result: []model.ApplicationStatus{},
		}
		err = json.NewDecoder(&w).Decode(&p)
		require.NoError(t, err)
		t.Logf(">>> Got app %d statuses for app id %s\n", len(p.Result), applicationID)
		for _, appStatus := range p.Result {
			if appStatus.ApplicationID != applicationID {
				t.Fatal("application id mismatch!")
			}
		}

		err = dbAPI.GetApplicationStatusW(ctx1, applicationID, &w, &r)
		require.NoError(t, err)
		appStatuses2 := []model.ApplicationStatus{}
		err = json.NewDecoder(&w).Decode(&appStatuses2)
		require.NoError(t, err)
		if len(appStatuses2) != len(p.Result) {
			t.Fatal("expect length of appStatuses2 and p.Result to equal")
		}
		for i := 0; i < len(appStatuses2); i++ {
			if !reflect.DeepEqual(appStatuses2[i], p.Result[i]) {
				t.Fatalf("expect app status %d to equal\n", i)
			}
		}

		appStatuses3, err := dbAPI.GetApplicationStatus(ctx1, applicationID)
		require.NoError(t, err)
		if len(appStatuses2) != len(appStatuses3) {
			t.Fatal("expect length of appStatuses2 and appStatuses3 to equal")
		}
		for i := 0; i < len(appStatuses2); i++ {
			if !reflect.DeepEqual(appStatuses2[i], appStatuses3[i]) {
				t.Fatalf("expect app status 2 and 3 at %d to equal\n", i)
			}
		}

	})

}
*/

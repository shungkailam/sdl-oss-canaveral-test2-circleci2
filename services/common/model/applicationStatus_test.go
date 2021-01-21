package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestApplicationStatus will test ApplicationStatus struct
func TestApplicationStatus(t *testing.T) {
	var tenantID = "tenant-id-waldot"
	edgeID := "edge-id"
	applicationID := "app-id"
	now := timeNow(t)

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
	applicationStrings := []string{
		`{"version":5,"tenantId":"tenant-id-waldot","edgeId":"edge-id","applicationId":"app-id","appStatus":{"podStatusList":[{"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{"kubernetes.io/created-by":"{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicaSet\",\"namespace\":\"shyan\",\"name\":\"cloudmgmt-deployment-86dd869d98\",\"uid\":\"dc80efcf-7338-11e8-9df7-06df7cc3bc4e\",\"apiVersion\":\"extensions\",\"resourceVersion\":\"10914914\"}}\n"},"creationTimestamp":"2018-06-18T20:47:49Z","generateName":"cloudmgmt-deployment-86dd869d98-","labels":{"app":"cloudmgmt","pod-template-hash":"4288425854"},"name":"cloudmgmt-deployment-86dd869d98-j9676","namespace":"shyan","ownerReferences":[{"apiVersion":"extensions/v1beta1","blockOwnerDeletion":true,"controller":true,"kind":"ReplicaSet","name":"cloudmgmt-deployment-86dd869d98","uid":"dc80efcf-7338-11e8-9df7-06df7cc3bc4e"}],"resourceVersion":"10914936","selfLink":"/api/v1/namespaces/shyan/pods/cloudmgmt-deployment-86dd869d98-j9676","uid":"dc87d927-7338-11e8-9df7-06df7cc3bc4e"},"spec":{"containers":[{"command":["sh","-c","exec /usr/src/app/cloudmgmt --sql_db=$SQL_DB --sql_host=$SQL_HOST --sql_password=$SQL_PASSWORD --contentdir=$CONTENTDIR --logtostderr -v 3"],"env":[{"name":"AWS_ACCESS_KEY_ID","valueFrom":{"secretKeyRef":{"key":"accessKeyId","name":"aws-secret"}}},{"name":"AWS_SECRET_ACCESS_KEY","valueFrom":{"secretKeyRef":{"key":"secretAccessKey","name":"aws-secret"}}},{"name":"SQL_DB","valueFrom":{"configMapKeyRef":{"key":"db","name":"db-config"}}},{"name":"SQL_HOST","valueFrom":{"configMapKeyRef":{"key":"dbHost","name":"db-config"}}},{"name":"SQL_PASSWORD","valueFrom":{"secretKeyRef":{"key":"sqlPassword","name":"aws-secret"}}}],"image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imagePullPolicy":"IfNotPresent","name":"cloudmgmt","ports":[{"containerPort":8080,"protocol":"TCP"}],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","volumeMounts":[{"mountPath":"/var/run/secrets/kubernetes.io/serviceaccount","name":"default-token-zmnxs","readOnly":true}]}],"dnsPolicy":"ClusterFirst","nodeName":"ip-172-31-64-135.us-west-2.compute.internal","restartPolicy":"Always","schedulerName":"default-scheduler","securityContext":{},"serviceAccount":"default","serviceAccountName":"default","terminationGracePeriodSeconds":30,"tolerations":[{"effect":"NoExecute","key":"node.alpha.kubernetes.io/notReady","operator":"Exists","tolerationSeconds":300},{"effect":"NoExecute","key":"node.alpha.kubernetes.io/unreachable","operator":"Exists","tolerationSeconds":300}],"volumes":[{"name":"default-token-zmnxs","secret":{"defaultMode":420,"secretName":"default-token-zmnxs"}}]},"status":{"conditions":[{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"Initialized"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:58Z","status":"True","type":"Ready"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"PodScheduled"}],"containerStatuses":[{"containerID":"docker://4f479d7c43b5c6702caf2a53201c6004740a521ad7552a50cf080b3b3f1d647a","image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imageID":"docker-pullable://770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev@sha256:9e3aa147c3a0d12f6cde7f53be077d2dc8b78570ac051df2dd5f1bf0518316cf","lastState":{},"name":"cloudmgmt","ready":true,"restartCount":0,"state":{"running":{"startedAt":"2018-06-18T20:47:57Z"}}}],"hostIP":"172.31.64.135","phase":"Running","podIP":"100.96.2.153","qosClass":"BestEffort","startTime":"2018-06-18T20:47:49Z"}}],"podMetricsList":null,"imageList":null},"createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z"}`,
	}

	var version float64 = 5
	appStatusMap := make(map[string][]map[string]interface{})
	appStatusMap["podStatusList"] = []map[string]interface{}{
		podStatus,
	}
	appStatusMap["podMetricsList"] = nil
	appStatusMap["imageList"] = nil
	applicationMaps := []map[string]interface{}{
		{
			"version":       version,
			"tenantId":      tenantID,
			"edgeId":        edgeID,
			"applicationId": applicationID,
			"appStatus":     appStatusMap,
			"createdAt":     NOW,
			"updatedAt":     NOW,
		},
	}

	for i, application := range applications {
		applicationData, err := json.Marshal(application)
		require.NoError(t, err, "failed to marshal application")

		if applicationStrings[i] != string(applicationData) {
			t.Fatalf("application json string mismatch: %s", string(applicationData))
		}

		var doc interface{}
		doc = application
		_, ok := doc.(model.ProjectScopedEntity)
		if ok {
			t.Fatal("application status should not be a project scoped entity")
		}

		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(applicationData, &m)
		require.NoError(t, err, "failed to unmarshal application to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &applicationMaps[i]) {
			t.Logf("%+v", applicationMaps[i])
			t.Fatalf("application map marshal mismatch: %+v", m)
		}
	}

}

package websocket_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/websocket"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/graarh/golang-socketio"
	"github.com/graarh/golang-socketio/transport"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// TestApplication will test application over web socket
func TestApplication(t *testing.T) {
	t.Parallel()

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenantID := base.GetUUID()
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	// create tenant
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	require.NoError(t, err)

	t.Logf("create tenant successful, %s", resp)

	// create edge
	edgeName := "my-test-edge"
	edgeSerialNumber := base.GetUUID()
	edgeIP := "1.1.1.1"
	edgeSubnet := "255.255.255.0"
	edgeGateway := "1.1.1.1"

	edge := model.Edge{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         edgeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    edgeIP,
				Subnet:       edgeSubnet,
				Gateway:      edgeGateway,
				EdgeDevices:  3,
			},
			StorageCapacity: 100,
			StorageUsage:    80,
		},
		Connected: true,
	}
	resp, err = dbAPI.CreateEdge(ctx, &edge, nil)
	require.NoError(t, err)
	t.Logf("create edge successful, %s", resp)

	edgeID := resp.(model.CreateDocumentResponse).ID

	// create project
	projName := fmt.Sprintf("Where is Waldo-%s", base.GetUUID())
	projDesc := "Find Waldo"
	project := model.Project{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:               projName,
		Description:        projDesc,
		CloudCredentialIDs: []string{},
		DockerProfileIDs:   []string{},
		Users:              []model.ProjectUserInfo{},
		EdgeSelectorType:   model.ProjectEdgeSelectorTypeExplicit,
		EdgeIDs:            []string{edgeID},
		EdgeSelectors:      nil,
	}
	resp, err = dbAPI.CreateProject(ctx, &project, nil)
	require.NoError(t, err)
	t.Logf("create project successful, %s", resp)

	projectID := resp.(model.CreateDocumentResponse).ID

	// add project id for app create permission
	projRoles := []model.ProjectRole{
		{
			ProjectID: projectID,
			Role:      model.ProjectRoleAdmin,
		},
	}
	authContext.Claims["projects"] = projRoles

	// create application
	appName := "app name"
	appDesc := "test app"
	appYamlData := `apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: deployment-demo
spec:
  selector:
    matchLabels:
      demo: deployment
  replicas: 5
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
    type: RollingUpdate
  template:
    metadata:
      labels:
        demo: deployment
        version: v1
    spec:
      containers:
      - name: busybox
        image: busybox
        command: [ "sh", "-c", "while true; do echo hostname; sleep 60; done" ]
        volumeMounts:
        - name: content
          mountPath: /data
      - name: nginx
        image: nginx
        volumeMounts:
          - name: content
            mountPath: /usr/share/nginx/html
            readOnly: true
      volumes:
      - name: content`
	app := model.Application{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		ApplicationCore: model.ApplicationCore{
			Name:        appName,
			Description: appDesc,
			ProjectID:   projectID,
		},
		YamlData: appYamlData,
	}

	resp, err = dbAPI.CreateApplication(ctx, &app, nil)
	require.NoError(t, err)
	t.Logf("create application successful, %s", resp)
	appID := resp.(model.CreateDocumentResponse).ID

	// Teardown
	defer func() {
		dbAPI.DeleteApplication(ctx, appID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		defer dbAPI.Close()
	}()

	podStatus := model.PodStatus{}
	podStatusString := `{"apiVersion":"v1","kind":"Pod","metadata":{"annotations":{"kubernetes.io/created-by":"{\"kind\":\"SerializedReference\",\"apiVersion\":\"v1\",\"reference\":{\"kind\":\"ReplicaSet\",\"namespace\":\"shyan\",\"name\":\"cloudmgmt-deployment-86dd869d98\",\"uid\":\"dc80efcf-7338-11e8-9df7-06df7cc3bc4e\",\"apiVersion\":\"extensions\",\"resourceVersion\":\"10914914\"}}\n"},"creationTimestamp":"2018-06-18T20:47:49Z","generateName":"cloudmgmt-deployment-86dd869d98-","labels":{"app":"cloudmgmt","pod-template-hash":"4288425854"},"name":"cloudmgmt-deployment-86dd869d98-j9676","namespace":"shyan","ownerReferences":[{"apiVersion":"extensions/v1beta1","blockOwnerDeletion":true,"controller":true,"kind":"ReplicaSet","name":"cloudmgmt-deployment-86dd869d98","uid":"dc80efcf-7338-11e8-9df7-06df7cc3bc4e"}],"resourceVersion":"10914936","selfLink":"/api/v1/namespaces/shyan/pods/cloudmgmt-deployment-86dd869d98-j9676","uid":"dc87d927-7338-11e8-9df7-06df7cc3bc4e"},"spec":{"containers":[{"command":["sh","-c","exec /usr/src/app/cloudmgmt --sql_db=$SQL_DB --sql_host=$SQL_HOST --sql_password=$SQL_PASSWORD --contentdir=$CONTENTDIR --logtostderr -v 3"],"env":[{"name":"AWS_ACCESS_KEY_ID","valueFrom":{"secretKeyRef":{"key":"accessKeyId","name":"aws-secret"}}},{"name":"AWS_SECRET_ACCESS_KEY","valueFrom":{"secretKeyRef":{"key":"secretAccessKey","name":"aws-secret"}}},{"name":"SQL_DB","valueFrom":{"configMapKeyRef":{"key":"db","name":"db-config"}}},{"name":"SQL_HOST","valueFrom":{"configMapKeyRef":{"key":"dbHost","name":"db-config"}}},{"name":"SQL_PASSWORD","valueFrom":{"secretKeyRef":{"key":"sqlPassword","name":"aws-secret"}}}],"image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imagePullPolicy":"IfNotPresent","name":"cloudmgmt","ports":[{"containerPort":8080,"protocol":"TCP"}],"resources":{},"terminationMessagePath":"/dev/termination-log","terminationMessagePolicy":"File","volumeMounts":[{"mountPath":"/var/run/secrets/kubernetes.io/serviceaccount","name":"default-token-zmnxs","readOnly":true}]}],"dnsPolicy":"ClusterFirst","nodeName":"ip-172-31-64-135.us-west-2.compute.internal","restartPolicy":"Always","schedulerName":"default-scheduler","securityContext":{},"serviceAccount":"default","serviceAccountName":"default","terminationGracePeriodSeconds":30,"tolerations":[{"effect":"NoExecute","key":"node.alpha.kubernetes.io/notReady","operator":"Exists","tolerationSeconds":300},{"effect":"NoExecute","key":"node.alpha.kubernetes.io/unreachable","operator":"Exists","tolerationSeconds":300}],"volumes":[{"name":"default-token-zmnxs","secret":{"defaultMode":420,"secretName":"default-token-zmnxs"}}]},"status":{"conditions":[{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"Initialized"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:58Z","status":"True","type":"Ready"},{"lastProbeTime":null,"lastTransitionTime":"2018-06-18T20:47:49Z","status":"True","type":"PodScheduled"}],"containerStatuses":[{"containerID":"docker://4f479d7c43b5c6702caf2a53201c6004740a521ad7552a50cf080b3b3f1d647a","image":"770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev:708","imageID":"docker-pullable://770301640873.dkr.ecr.us-west-2.amazonaws.com/cloudmgmt-dev@sha256:9e3aa147c3a0d12f6cde7f53be077d2dc8b78570ac051df2dd5f1bf0518316cf","lastState":{},"name":"cloudmgmt","ready":true,"restartCount":0,"state":{"running":{"startedAt":"2018-06-18T20:47:57Z"}}}],"hostIP":"172.31.64.135","phase":"Running","podIP":"100.96.2.153","qosClass":"BestEffort","startTime":"2018-06-18T20:47:49Z"}}`

	err = json.Unmarshal([]byte(podStatusString), &podStatus)
	require.NoError(t, err)

	req := websocket.ReportAppStatusRequest{
		TenantID: tenantID,
		EdgeID:   edgeID,
		ID:       appID,
		Pods: []model.PodStatus{
			podStatus,
		},
		PodMetricses: []model.PodMetrics{},
	}

	ba, err := json.Marshal(req)
	require.NoError(t, err)
	bas := base64.StdEncoding.EncodeToString(ba)

	c, err := gosocketio.Dial(
		gosocketio.GetUrl(apitesthelper.TestServer, apitesthelper.TestPort, apitesthelper.TestSecure),
		transport.GetDefaultWebsocketTransport())
	require.NoError(t, err)

	// note: Ack or Emit both works
	result, err := c.Ack("application-status", bas, time.Second*20)
	require.NoError(t, err)

	t.Log("Ack result to /application-status: ", result)
	rsp := websocket.ReportAppStatusResponse{}
	err = json.Unmarshal([]byte(result), &rsp)
	require.NoError(t, err)
	if rsp.StatusCode != 200 {
		t.Fatal("response status not ok")
	}
	t.Logf("response: %+v", rsp)

	// delete application status
	delResp, err := dbAPI.DeleteApplicationStatus(ctx, appID, nil)
	require.NoError(t, err)
	t.Logf("delete application successful, %v", delResp)
}

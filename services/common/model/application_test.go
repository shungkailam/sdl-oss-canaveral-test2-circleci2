package model_test

import (
	"cloudservices/common/errcode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cloudservices/common/model"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"
)

// TestApplication will test Application struct
func TestApplication(t *testing.T) {
	var tenantID = "tenant-id-waldot"
	now := timeNow(t)
	appYaml := `apiVersion: extensions/v1beta1
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
      initContainers:
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
	applications := []model.Application{
		{
			BaseModel: model.BaseModel{
				ID:        "app-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ApplicationCore: model.ApplicationCore{
				Name:          "test-app",
				Description:   "test application",
				ProjectID:     "proj-id",
				EdgeIDs:       []string{"edge-id-1"},
				EdgeSelectors: nil,
			},
			YamlData: appYaml,
		},
	}
	err := model.ValidateApplication(&applications[0], model.GetK8sSchemaVersion(false))
	require.NoError(t, err)

	applicationStrings := []string{
		fmt.Sprintf(`{"id":"app-id","version":5,"tenantId":"tenant-id-waldot","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"test-app","description":"test application","edgeIds":["edge-id-1"],"projectId":"proj-id","edgeSelectors":null,"originSelectors":null,"dataIfcEndpoints":null,"yamlData":%q,"onlyPrePullOnUpdate":false,"packagingType":null,"helmMetadata":null}`, appYaml),
	}

	var version float64 = 5
	applicationMaps := []map[string]interface{}{
		{
			"id":                  "app-id",
			"version":             version,
			"tenantId":            tenantID,
			"name":                "test-app",
			"description":         "test application",
			"projectId":           "proj-id",
			"edgeIds":             []string{"edge-id-1"},
			"edgeSelectors":       nil,
			"originSelectors":     nil,
			"dataIfcEndpoints":    nil,
			"createdAt":           NOW,
			"updatedAt":           NOW,
			"yamlData":            appYaml,
			"onlyPrePullOnUpdate": false,
			"packagingType":       nil,
			"helmMetadata":        nil,
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
		require.True(t, ok, "application should be a project scoped entity")

		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(applicationData, &m)
		require.NoError(t, err, "failed to unmarshal application to map")

		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &applicationMaps[i]) {
			t.Fatalf("application map marshal mismatch: %+v", m)
		}

		appV2 := application.ToV2()
		app2 := appV2.FromV2()
		if !reflect.DeepEqual(application, app2) {
			t.Fatal("expect app to equal after convert to v2 then back")
		}
	}

}

func TestDataIfcValidation(t *testing.T) {
	var tenantID = "tenant-id-waldot"
	now := timeNow(t)
	appYaml := `apiVersion: extensions/v1beta1
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
	application := model.Application{
		BaseModel: model.BaseModel{
			ID:        "app-id",
			TenantID:  tenantID,
			Version:   5,
			CreatedAt: now,
			UpdatedAt: now,
		},
		ApplicationCore: model.ApplicationCore{
			Name:          "test-app",
			Description:   "test application",
			ProjectID:     "proj-id",
			EdgeIDs:       []string{"edge-id-1"},
			EdgeSelectors: nil,
		},
		YamlData: appYaml,
	}

	testCases := []struct {
		endpoints   []model.DataIfcEndpoint
		errExpected bool
	}{
		{[]model.DataIfcEndpoint{{}}, true},
		{[]model.DataIfcEndpoint{{Name: "foo"}}, true},
		{[]model.DataIfcEndpoint{{Name: "bar", ID: "id"}}, true},
		{[]model.DataIfcEndpoint{{Name: "bar", ID: "id", Value: "value"}}, false},
	}

	for _, testCase := range testCases {
		application.DataIfcEndpoints = testCase.endpoints
		err := model.ValidateApplication(&application, model.GetK8sSchemaVersion(false))
		if (err != nil) != testCase.errExpected {
			t.Fatalf("expected err: %v, but got %v", testCase.errExpected, err != nil)
		}
	}
}

// TestInvalidApplication will test Application validation
func TestInvalidApplication(t *testing.T) {
	var tenantID = "tenant-id-waldot"
	now := timeNow(t)
	type tt struct {
		yamlFileName string
		valid        bool
		privileged   bool
		errorString  string
	}
	var ts []tt
	ts = []tt{
		{"additionalProperties", false, false, "Yaml has following errors: [NetworkPolicy] metadata: Additional property foo is not allowed"},
		{"netpol", true, false, ""},
		{"validHostport", true, false, ""},
		{"privileged", false, false, "Yaml has following errors: [Deployment] spec.template.spec.containers.0.securityContext: Additional property capabilities is not allowed, [Deployment] spec.template.spec.containers.0.securityContext: Additional property privileged is not allowed, [Deployment] spec.template.spec.initContainers.0.securityContext: Additional property privileged is not allowed"},
		{"invalidHostport", false, false, "Yaml has following errors: [Deployment] hostport: Port 10250 is reserved. "},
		{"compassApp", true, false, ""},
		{"compassAppWindowsEOL", true, false, ""},
		{"pvcApp", true, false, ""},
		{"minecraft", true, false, ""},
		{"multipleErrors", false, false, "Yaml has following errors: [Deployment] spec.template.spec.securityContext: Additional property capabilities is not allowed, [Deployment] spec.template.spec.volumes.0: Additional property hostPath is not allowed, [Deployment] spec.template.spec.volumes.1: Additional property hostPath is not allowed, [Deployment] spec.template.spec.volumes.2: Additional property hostPath is not allowed"},
		{"invalidTemplate", true, false, ""},
		{"invalidStorageClass", false, false, "Yaml has following errors: [StatefulSet] storageClassName: storageClassName system is not allowed. Only local storage class is allowed"},
		{"invalidHostpath", false, false, "Yaml has following errors: [Deployment] spec.template.spec.volumes.0: Additional property hostPath is not allowed"},
		{"nullName", false, false, "Yaml has following errors: [ConfigMap] metadata.name: Invalid name ''. Name must be valid DNS-1123 subdomain"},
		{"invalidName", false, false, "Yaml has following errors: [ConfigMap] metadata.name: Invalid name '%%%'. Name must be valid DNS-1123 subdomain"},
		{"labels", false, false, "Yaml has following errors: [Deployment] metadata.labels.: Invalid label name 'blah$'. Name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character, [Deployment] metadata.labels.: Invalid label name 'blah$'. Name part must consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character, [Deployment] metadata.labels.: Invalid label name prefix 'köhler.com'. Prefix part must be valid DNS subdomain, [Deployment] metadata.labels.: Invalid label value 'lamp$'. A valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character, [Deployment] metadata.labels.: Invalid label value '片仮名'. A valid label must be an empty string or consist of alphanumeric characters, '-', '_' or '.', and must start and end with an alphanumeric character"},
		{"ingress", true, false, ""},
		{"oldStatefulset", true, false, ""},
		{"newStatefulset", true, false, ""},
		{"empty", false, false, "Kind is not set"},
		{"empty2", false, false, "Kind is not set"},
		{"empty3", true, false, ""},
		// prometheus operator yaml generated from helm chart
		// see: SHLK-158
		{"prometheusOperator", true, true, ""},
		{"istiovirtualservice", true, false, ""},
		{"extraFeildsIstiovirtualservice", false, false, "Yaml has following errors: [VirtualService](root):  Additional properties [extraField] not allowed"},
		{"invalidNameIstiovirtualservice", false, false, "Yaml has following errors: [VirtualService] http.0.name: Invalid type. Expected: string, given: integer"},
		{"istiodestinationrule", true, false, ""},
	}

	for _, tc := range ts {
		yamlData, err := ioutil.ReadFile(fmt.Sprintf("../testdata/%s.yaml", tc.yamlFileName))
		assert.NoError(t, err)

		app := &model.Application{
			BaseModel: model.BaseModel{
				ID:        "app-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ApplicationCore: model.ApplicationCore{
				Name:          "test-app",
				Description:   "test application",
				ProjectID:     "proj-id",
				EdgeIDs:       []string{"edge-id-1"},
				EdgeSelectors: nil,
			},
			YamlData: string(yamlData),
		}

		t.Logf("Validate %q", tc.yamlFileName)
		err = model.ValidateApplication(app, model.GetK8sSchemaVersion(tc.privileged))
		assert.Falsef(t, err != nil && tc.valid == true, "[%s] Err not expected but got %s", tc.yamlFileName, err)
		assert.Falsef(t, err == nil && tc.valid == false, "[%s] Err expected but got no error %q", tc.yamlFileName, tc.errorString)
		if err != nil {
			e := err.(*errcode.BadRequestExError)
			assert.Falsef(t, tc.valid == false && e.Msg != tc.errorString, "[%s] Result '%s' does not match expected '%s'", tc.yamlFileName, e.Msg, tc.errorString)
		}
	}
}

func TestGetCrdKinds(t *testing.T) {
	type tt struct {
		crds     string
		crdKinds []string
	}
	var ts []tt
	ts = []tt{
		{"", nil},
		{"  kind: CustomResourceDefinition", nil},
		{"  kind: MyResource", []string{"MyResource"}},
		{"kind: CustomResourceDefinition\n kind: MyResource", []string{"MyResource"}},
		{"kind: MyResource\n kind:  MyResource2\n\n", []string{"MyResource", "MyResource2"}},
	}
	for _, tc := range ts {
		kinds := model.GetCrdKinds(tc.crds)
		if !reflect.DeepEqual(tc.crdKinds, kinds) {
			t.Fatalf("expect %q to equal %q", tc.crdKinds, kinds)
		}
	}
}

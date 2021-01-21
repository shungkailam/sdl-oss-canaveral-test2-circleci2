package model_test

import (
	"cloudservices/common/model"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEventPath(t *testing.T) {
	paths := map[string]*model.EventPathComponents{
		"/serviceDomain:12345/project:678/dataPipeline:901/status/liveness": &model.EventPathComponents{
			SvcDomainID:    "12345",
			ProjectID:      "678",
			DataPipelineID: "901",
		},
		"/serviceDomain:12345/project:678/dataPipeline:901/function:10/status": &model.EventPathComponents{
			SvcDomainID:    "12345",
			ProjectID:      "678",
			DataPipelineID: "901",
			Function:       "10",
		},
		"/serviceDomain:12345/project:678/dataPipeline:901/output:connector/status": &model.EventPathComponents{
			SvcDomainID:    "12345",
			ProjectID:      "678",
			DataPipelineID: "901",
			Output:         "connector",
		},
		"/serviceDomain:12345/project:678/dataPipeline:901/$PROTOCOL_TYPE:connector/status": &model.EventPathComponents{
			SvcDomainID:    "12345",
			ProjectID:      "678",
			DataPipelineID: "901",
		},
		"/serviceDomain:12345/project:678/application:11/container:$IMAGE_NAME/status": &model.EventPathComponents{
			SvcDomainID:   "12345",
			ProjectID:     "678",
			ApplicationID: "11",
			Container:     "$IMAGE_NAME",
		},
		"/serviceDomain:12345/dataSource:10/topic:test/status": &model.EventPathComponents{
			SvcDomainID:  "12345",
			DataSourceID: "10",
			Topic:        "test",
		},
		"/serviceDomain:0c4c1731-4e60-4879-8f77-147ae87ec6c7/project:4bbd3e4d-3860-4b16-9dac-1bbe873480a7/service:kafka/kafka/KafkaDeployment": &model.EventPathComponents{
			SvcDomainID: "0c4c1731-4e60-4879-8f77-147ae87ec6c7",
			ProjectID:   "4bbd3e4d-3860-4b16-9dac-1bbe873480a7",
			SvcType:     "kafka",
		},
		"/serviceDomain:0c4c1731-4e60-4879-8f77-147ae87ec6c7/project:e065efd8-9a84-4233-8767-3c4a096e84fa/application:393ba9ad-808a-4bbd-a168-ef583f9f5cda/ingress": &model.EventPathComponents{
			SvcDomainID:   "0c4c1731-4e60-4879-8f77-147ae87ec6c7",
			ProjectID:     "e065efd8-9a84-4233-8767-3c4a096e84fa",
			ApplicationID: "393ba9ad-808a-4bbd-a168-ef583f9f5cda",
		},
	}
	for path, comps := range paths {
		outComps := model.ExtractEventPathComponents(path)
		t.Logf("Path %s --> %+v", path, outComps)
		if !reflect.DeepEqual(outComps, comps) {
			t.Fatalf("expected %+v, found %+v for path %s", comps, outComps, path)
		}
	}
}

func TestGenerateEventQueryPath(t *testing.T) {
	// "/serviceDomain:${svcDomainId}/project:${projectId}/service:${type}/instance:${svcInstanceId}/status"
	in := struct {
		SvcDomainID     string `json:"svcDomainId"`
		ProjectID       string `json:"projectId"`
		Type            string `json:"type"`
		ServiceInstance string `json:"svcInstanceId"`
	}{
		"my-svc-domain-id",
		"my-project-id",
		"my-type",
		"my-svc-instance-id",
	}
	path, subs, err := model.GenerateEventQueryPath(model.ServiceInstanceStatusProjectScopedEventPath, in)
	require.NoError(t, err)
	if subs {
		t.Fatalf("No substitution expected %s", path)
	}
	if path != "/serviceDomain:my-svc-domain-id/project:my-project-id/service:my-type/instance:my-svc-instance-id/status" {
		t.Fatalf("unexpected path %s", path)
	}
	// Additional field sent
	path, subs, err = model.GenerateEventQueryPath(model.ServiceInstanceStatusServiceDomainScopedEventPath, in)
	if subs {
		t.Fatalf("No substitution expected %s", path)
	}
	if path != "/serviceDomain:my-svc-domain-id/service:my-type/instance:my-svc-instance-id/status" {
		t.Fatalf("unexpected path %s", path)
	}
	in.SvcDomainID = ""
	path, subs, err = model.GenerateEventQueryPath(model.ServiceInstanceStatusServiceDomainScopedEventPath, in)
	if !subs {
		t.Fatalf("substitution expected %s", path)
	}
	if path != "/serviceDomain:.*/service:my-type/instance:my-svc-instance-id/status" {
		t.Fatalf("unexpected path %s", path)
	}
}

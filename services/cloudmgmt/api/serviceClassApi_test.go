package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func createServiceClass(t *testing.T, dbAPI api.ObjectModelAPI) *model.ServiceClass {
	ctx := context.TODO()
	svcClass := createServiceClassPayload(t)
	resp, err := dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	createResp := resp.(model.CreateDocumentResponseV2)
	svcClassResp, err := dbAPI.GetServiceClass(ctx, createResp.ID)
	require.NoError(t, err)
	return &svcClassResp
}

func createServiceClassPayload(t *testing.T) *model.ServiceClass {
	svcInstanceCreateSchema := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "object",
		"properties": {
			"logRetentionBytes": {
				"type": "string",
				"default": "1000000"
			},
			"kafkaVolumeSize": {
				"type": "string",
				"default": "1000000"
			},
			"profile": {
				"description": "Mutually exclusive kafka profiles",
				"type": "string",
				"enum": [
					"Durability",
					"Throughput",
					"Availability"
				],
				"default": "Availability"
			}
		}
	}`)
	svcInstanceUpdateSchema := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "object",
		"properties": {
			"logRetentionBytes": {
				"type": "string",
				"default": "1000000"
			}
		}
	}`)
	svcBindingCreateSchema := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "object",
		"properties": {
			"access-level": {
				"type": "string",
				"enum": [
					"ReadOnly",
					"Write"
				]
			}
		},
		"required": ["access-level"]
	}`)
	svcInstanceCreateSchemaMap := map[string]interface{}{}
	err := base.ConvertFromJSON(svcInstanceCreateSchema, &svcInstanceCreateSchemaMap)
	require.NoError(t, err)

	svcInstanceUpdateSchemaMap := map[string]interface{}{}
	err = base.ConvertFromJSON(svcInstanceUpdateSchema, &svcInstanceUpdateSchemaMap)
	require.NoError(t, err)

	svcBindingCreateSchemaMap := map[string]interface{}{}
	err = base.ConvertFromJSON(svcBindingCreateSchema, &svcBindingCreateSchemaMap)
	require.NoError(t, err)

	svcVersion := base.GetUUID()
	svcName := "kafka " + svcVersion
	svcClass := model.ServiceClass{
		ServiceClassCommon: model.ServiceClassCommon{
			Type:                "kafka-" + svcVersion,
			SvcVersion:          svcVersion,
			Scope:               model.ServiceClassProjectScope,
			MinSvcDomainVersion: "v2.0.0",
		},
		Name:        svcName,
		Description: "Runs Kafka service in your project",
		State:       model.ServiceClassFinalState,
		Bindable:    true,
		Schemas: model.ServiceClassSchemas{
			SvcInstance: model.ServiceInstanceSchema{
				Create: model.Schema{
					Parameters: svcInstanceCreateSchemaMap,
				},
				Update: model.Schema{
					Parameters: svcInstanceUpdateSchemaMap,
				},
			},
			SvcBinding: model.ServiceBindingSchema{
				Create: model.Schema{
					Parameters: svcBindingCreateSchemaMap,
				},
			},
		},
		Tags: []model.ServiceClassTag{
			model.ServiceClassTag{
				Name:  "essential",
				Value: "yes",
			},
			model.ServiceClassTag{
				Name:  "category",
				Value: "pub-sub",
			},
		},
	}
	return &svcClass
}

func TestServiceClass(t *testing.T) {
	ctx := context.TODO()
	dbAPI := newObjectModelAPI(t)
	defer dbAPI.Close()
	svcInstanceCreateSchemaInvalid := []byte(`
	{
		"title": "KafkaCfgOptions",
		"type": "invalid",
		"properties": {
			"logRetentinBytes": {
				"type": "string",
				"default": "1000000"
			},
			"kafkaVolumeSize": {
				"type": "string",
				"default": "1000000"
			},
			"profile": {
				"description": "Mutually exclusive kafka profiles",
				"type": "string",
				"enum": [
					"Durability",
					"Throughput",
					"Availability"
				]
			}
		}
	}`)
	svcClass := createServiceClassPayload(t)
	svcInstanceCreateSchemaMap := svcClass.Schemas.SvcInstance.Create.Parameters
	svcInstanceCreateSchemaInvalidMap := map[string]interface{}{}
	err := base.ConvertFromJSON(svcInstanceCreateSchemaInvalid, &svcInstanceCreateSchemaInvalidMap)
	require.NoError(t, err)

	svcClass.Schemas.SvcInstance.Create.Parameters = svcInstanceCreateSchemaInvalidMap
	// Create Service Class with invalid schema spec
	_, err = dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.Error(t, err)
	svcClass.Schemas.SvcInstance.Create.Parameters = svcInstanceCreateSchemaMap
	svcClass.Tags = []model.ServiceClassTag{}
	// Create Service Class with valid schema spec but without required tags
	_, err = dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.Error(t, err)

	svcClass.Tags = []model.ServiceClassTag{
		model.ServiceClassTag{
			Name:  "essential",
			Value: "yes",
		},
		model.ServiceClassTag{
			Name:  "category",
			Value: "pub-sub",
		},
	}
	// Create Service Class with valid schema spec and required tags
	resp, err := dbAPI.CreateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	createResp := resp.(model.CreateDocumentResponseV2)
	defer dbAPI.DeleteServiceClass(ctx, createResp.ID, nil)
	svcClassResp, err := dbAPI.GetServiceClass(ctx, createResp.ID)
	require.NoError(t, err)
	t.Logf("service class: %+v", svcClassResp)
	svcClass.ID = createResp.ID
	svcClass.Version = svcClassResp.Version
	svcClass.CreatedAt = svcClassResp.CreatedAt
	svcClass.UpdatedAt = svcClassResp.UpdatedAt
	if !reflect.DeepEqual(svcClass, &svcClassResp) {
		t.Fatalf("expected:\n%+v \nfound: \n%+v\n", svcClass, svcClassResp)
	}
	entitiesQueryParam := model.EntitiesQueryParam{}
	queryParam := model.ServiceClassQueryParam{}
	queryParam.Scope = model.ServiceClassServiceDomainScope
	queryParam.SvcVersion = svcClass.SvcVersion
	// Search with no assigned scope
	listPayload, err := dbAPI.SelectAllServiceClasses(ctx, &entitiesQueryParam, &queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcClassList) != 0 || listPayload.TotalCount != 0 {
		t.Fatalf("expected 0 service classes. %+v", listPayload)
	}
	queryParam = model.ServiceClassQueryParam{}
	queryParam.Scope = model.ServiceClassProjectScope
	queryParam.SvcVersion = svcClass.SvcVersion
	// Search with assigned scope
	listPayload, err = dbAPI.SelectAllServiceClasses(ctx, &entitiesQueryParam, &queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcClassList) != 1 || listPayload.TotalCount != 1 {
		t.Fatalf("expected 1 service class. %+v", listPayload)
	}
	svcClassResp = listPayload.SvcClassList[0]
	svcClass.Version = svcClassResp.Version
	svcClass.CreatedAt = svcClassResp.CreatedAt
	svcClass.UpdatedAt = svcClassResp.UpdatedAt
	if !reflect.DeepEqual(svcClass, &listPayload.SvcClassList[0]) {
		t.Fatalf("expected:\n%+v \nfound: \n%+v\n", svcClass, svcClassResp)
	}
	updatedSvcVersion := base.GetUUID()
	svcClass.SvcVersion = updatedSvcVersion
	uResp, err := dbAPI.UpdateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	updateResp := uResp.(model.UpdateDocumentResponseV2)

	svcClassResp, err = dbAPI.GetServiceClass(ctx, updateResp.ID)
	require.NoError(t, err)
	t.Logf("service class: %+v", svcClassResp)
	svcClass.ID = createResp.ID
	svcClass.Version = svcClassResp.Version
	svcClass.CreatedAt = svcClassResp.CreatedAt
	svcClass.UpdatedAt = svcClassResp.UpdatedAt
	if !reflect.DeepEqual(svcClass, &svcClassResp) {
		t.Fatalf("expected:\n%+v \nfound: \n%+v\n", svcClass, svcClassResp)
	}
	queryParam.SvcVersion = svcClassResp.SvcVersion
	svcClass.Tags = []model.ServiceClassTag{
		model.ServiceClassTag{Name: "category", Value: "pub-sub"},
		model.ServiceClassTag{Name: "category", Value: "cat1"},
		model.ServiceClassTag{Name: "essential", Value: "yes"},
	}
	_, err = dbAPI.UpdateServiceClass(ctx, svcClass, nil)
	require.NoError(t, err)
	svcClassResp, err = dbAPI.GetServiceClass(ctx, updateResp.ID)
	require.NoError(t, err)
	t.Logf("service class: %+v", svcClassResp)
	svcClass.ID = createResp.ID
	svcClass.Version = svcClassResp.Version
	svcClass.CreatedAt = svcClassResp.CreatedAt
	svcClass.UpdatedAt = svcClassResp.UpdatedAt
	if !reflect.DeepEqual(svcClass, &svcClassResp) {
		t.Fatalf("expected:\n%+v \nfound: \n%+v\n", svcClass, svcClassResp)
	}
	// Select without any tag
	listPayload, err = dbAPI.SelectAllServiceClasses(ctx, &entitiesQueryParam, &queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcClassList) != 1 || listPayload.TotalCount != 1 {
		t.Fatalf("expected 1 service class. %+v", listPayload)
	}
	svcClassResp = listPayload.SvcClassList[0]
	svcClass.Version = svcClassResp.Version
	svcClass.CreatedAt = svcClassResp.CreatedAt
	svcClass.UpdatedAt = svcClassResp.UpdatedAt
	if !reflect.DeepEqual(svcClass, &listPayload.SvcClassList[0]) {
		t.Fatalf("expected:\n%+v \nfound: \n%+v\n", svcClass, svcClassResp)
	}
	// Search with correct name but bogus value
	queryParam.Tags = []string{"category=bogus"}
	listPayload, err = dbAPI.SelectAllServiceClasses(ctx, &entitiesQueryParam, &queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcClassList) != 0 || listPayload.TotalCount != 0 {
		t.Fatalf("expected 0 service class")
	}
	// Search with both the correct name and value
	queryParam.Tags = []string{"category=pub-sub"}
	listPayload, err = dbAPI.SelectAllServiceClasses(ctx, &entitiesQueryParam, &queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcClassList) != 1 || listPayload.TotalCount != 1 {
		t.Fatalf("expected 1 service class. %+v", listPayload)
	}
	svcClassResp = listPayload.SvcClassList[0]
	svcClass.Version = svcClassResp.Version
	svcClass.CreatedAt = svcClassResp.CreatedAt
	svcClass.UpdatedAt = svcClassResp.UpdatedAt
	if !reflect.DeepEqual(svcClass, &listPayload.SvcClassList[0]) {
		t.Fatalf("expected:\n%+v \nfound: \n%+v\n", svcClass, svcClassResp)
	}

	_, err = dbAPI.DeleteServiceClass(ctx, createResp.ID, nil)
	require.NoError(t, err)
	_, err = dbAPI.GetServiceClass(ctx, createResp.ID)
	require.Error(t, err)
}

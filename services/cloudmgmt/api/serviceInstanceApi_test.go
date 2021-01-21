package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func createServiceInstance(ctx context.Context, t *testing.T, dbAPI api.ObjectModelAPI, tenantID, svcClassID, scopeID string) *model.ServiceInstance {
	parameters := map[string]interface{}{
		"logRetentionBytes": "1000000",
		"kafkaVolumeSize":   "1500000",
	}

	svcInstanceParam := createServiceInstancePayload(t, tenantID, svcClassID, scopeID, parameters)
	resp, err := dbAPI.CreateServiceInstance(ctx, svcInstanceParam, nil)
	require.NoError(t, err)
	createResp := resp.(model.CreateDocumentResponseV2)
	svcInstanceResp, err := dbAPI.GetServiceInstance(ctx, createResp.ID)
	require.NoError(t, err)
	return &svcInstanceResp
}

func createServiceInstancePayload(t *testing.T, tenantID, svcClassID, scopeID string, parameters map[string]interface{}) *model.ServiceInstanceParam {
	// admin, no-access, project admin
	uuid := base.GetUUID()
	svcInstanceName := "svc-instance-" + uuid
	svcInstance := &model.ServiceInstanceParam{
		Name:        svcInstanceName,
		Description: "my svc instance",
		SvcClassID:  svcClassID,
		ScopeID:     scopeID,
		Parameters:  parameters,
	}
	return svcInstance
}

func TestServiceInstance(t *testing.T) {
	dbAPI := newObjectModelAPI(t)
	defer dbAPI.Close()
	svcClass := createServiceClassPayload(t)
	ctx := context.TODO()
	opCtx := base.GetOperatorContext(ctx)
	resp, err := dbAPI.CreateServiceClass(opCtx, svcClass, nil)
	require.NoError(t, err)
	createResp := resp.(model.CreateDocumentResponseV2)
	defer dbAPI.DeleteServiceClass(opCtx, createResp.ID, nil)
	svcClass.ID = createResp.ID
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	defer dbAPI.DeleteTenant(ctx, tenantID, nil)
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	edgeDevices := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	}, "edge", 2)
	edgeClusterID := edgeDevices[1].ClusterID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})

	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})
	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
	}()

	for _, edgeDevice := range edgeDevices {
		setEdgeDeviceVersion(t, dbAPI, tenantID, edgeDevice.ID, "v2.0.0")
	}

	svcClass.ID = createResp.ID
	invalidSvcInstanceParam := map[string]interface{}{
		"logRetentionBytes": "1000000",
		"kafkaVolumeSize":   123, // wrong type
	}
	svcInstanceParam := createServiceInstancePayload(t, tenantID, svcClass.ID, projectID, invalidSvcInstanceParam)
	svcInstance := &model.ServiceInstance{}
	err = base.Convert(svcInstanceParam, svcInstance)
	require.NoError(t, err)
	// Create with invalid schema
	_, err = dbAPI.CreateServiceInstance(ctx1, svcInstanceParam, nil)
	require.Error(t, err)
	if !strings.Contains(err.Error(), "kafkaVolumeSize: 123 type should be string") {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
	validSvcInstanceParam := map[string]interface{}{
		"logRetentionBytes": "1000000",
		"kafkaVolumeSize":   "1500000",
	}
	svcInstanceParam = createServiceInstancePayload(t, tenantID, svcClass.ID, projectID, validSvcInstanceParam)
	err = base.Convert(svcInstanceParam, svcInstance)
	require.NoError(t, err)
	svcInstance.TenantID = tenantID
	svcInstance.SvcVersion = svcClass.SvcVersion
	svcInstance.Type = svcClass.Type
	svcInstance.Scope = svcClass.Scope
	svcInstance.SvcClassName = svcClass.Name
	// The default must be picked up
	svcInstance.Parameters["profile"] = "Availability"

	sw := apitesthelper.NewSyncWait(t)
	resp, err = dbAPI.CreateServiceInstance(ctx1, svcInstanceParam, func(ctx context.Context, i interface{}) error {
		scopedEntity := i.(model.ScopedEntity)
		svcInstanceCB := scopedEntity.Doc.(model.ServiceInstance)
		svcInstance.ID = svcInstanceCB.ID
		svcInstance.MinSvcDomainVersion = svcInstanceCB.MinSvcDomainVersion
		svcInstance.Version = svcInstanceCB.Version
		svcInstance.CreatedAt = svcInstanceCB.CreatedAt
		svcInstance.UpdatedAt = svcInstanceCB.UpdatedAt
		t.Logf("callback received with %+v", svcInstanceCB)
		if !reflect.DeepEqual(svcInstance, &svcInstanceCB) {
			t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcInstance, &svcInstanceCB)
		}
		sw.Done()
		return nil
	})
	require.NoError(t, err)
	sw.WaitWithTimeout()
	createResp = resp.(model.CreateDocumentResponseV2)
	svcInstanceParam.ID = createResp.ID
	defer dbAPI.DeleteServiceInstance(ctx1, createResp.ID, nil)
	svcInstanceResp, err := dbAPI.GetServiceInstance(ctx1, createResp.ID)
	require.NoError(t, err)
	t.Logf("service instance: %+v", svcInstanceResp)
	svcInstance.ID = svcInstanceResp.ID
	svcInstance.Version = svcInstanceResp.Version
	svcInstance.CreatedAt = svcInstanceResp.CreatedAt
	svcInstance.UpdatedAt = svcInstanceResp.UpdatedAt
	if !reflect.DeepEqual(svcInstance, &svcInstanceResp) {
		t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcInstance, &svcInstanceResp)
	}
	entitiesQueryParam := &model.EntitiesQueryParam{}
	statusQueryParam := &model.ServiceInstanceStatusQueryParam{SvcDomainID: edgeClusterID}
	statusListPayload, err := dbAPI.SelectServiceInstanceStatuss(ctx1, svcInstance.ID, entitiesQueryParam, statusQueryParam)
	require.NoError(t, err)
	t.Logf("service instance status: %+v", statusListPayload)
	if len(statusListPayload.SvcInstanceStatusList) != 0 || statusListPayload.TotalCount != 0 {
		t.Fatalf("expected 0 service instance status. %+v", statusListPayload)
	}

	statusEventQueryParam := api.ServiceInstanceStatusEventQueryParam{
		ServiceInstanceCommon: svcInstance.ServiceInstanceCommon,
		SvcInstanceID:         svcInstance.ID,
		SvcDomainID:           edgeClusterID,
		ProjectID:             svcInstance.ScopeID,
	}
	ePathTemplate := model.ServiceInstanceStatusProjectScopedEventPath
	ePath, _, err := model.GenerateEventQueryPath(ePathTemplate, statusEventQueryParam)
	require.NoError(t, err)
	event := model.Event{
		Path: ePath,
		Properties: map[string]string{
			"endpoint": "fake-endpoint",
		},
		State:     string(model.ServiceInstanceProvisionedState),
		Type:      "STATUS",
		Timestamp: svcInstance.CreatedAt,
	}
	_, err = dbAPI.UpsertEvents(ctx1, model.EventUpsertRequest{Events: []model.Event{event}}, nil)
	require.NoError(t, err)

	statusListPayload, err = dbAPI.SelectServiceInstanceStatuss(ctx1, svcInstance.ID, entitiesQueryParam, statusQueryParam)
	require.NoError(t, err)
	t.Logf("service instance status: %+v", statusListPayload)
	if len(statusListPayload.SvcInstanceStatusList) != 1 || statusListPayload.TotalCount != 1 {
		t.Fatalf("expected 1 service instance status. %+v", statusListPayload)
	}
	if statusListPayload.SvcInstanceStatusList[0].State != model.ServiceInstanceProvisionedState {
		t.Fatalf("expected state %s, found %s", model.ServiceInstanceProvisionedState, statusListPayload.SvcInstanceStatusList[0].State)
	}
	if statusListPayload.SvcInstanceStatusList[0].Properties["endpoint"] != "fake-endpoint" {
		t.Fatalf("expected properties to endpoint=fake-endpoint, found endpoint=%+v", statusListPayload.SvcInstanceStatusList[0].Properties["endpoint"])
	}
	queryParam := &model.ServiceInstanceQueryParam{ScopeID: "blah"}
	queryParam.Type = "kafka"
	listPayload, err := dbAPI.SelectAllServiceInstances(ctx1, entitiesQueryParam, queryParam)
	require.NoError(t, err) // infra user
	queryParam.Type = svcClass.Type
	queryParam.ScopeID = svcInstance.ScopeID
	listPayload, err = dbAPI.SelectAllServiceInstances(ctx1, entitiesQueryParam, queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcInstanceList) != 1 || listPayload.TotalCount != 1 {
		t.Fatalf("expected 1 service instance. %+v", listPayload)
	}
	svcInstanceResp = listPayload.SvcInstanceList[0]
	svcInstance.Version = svcInstanceResp.Version
	svcInstance.CreatedAt = svcInstanceResp.CreatedAt
	svcInstance.UpdatedAt = svcInstanceResp.UpdatedAt
	if !reflect.DeepEqual(svcInstance, &svcInstanceResp) {
		t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcInstance, &svcInstanceResp)
	}
	// Must fail because some parameter fields are not updatable
	_, err = dbAPI.UpdateServiceInstance(ctx1, svcInstanceParam, nil)
	require.Error(t, err)
	if !strings.Contains(err.Error(), "Unknown field 'kafkaVolumeSize'") {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
	svcInstanceParam.Parameters = map[string]interface{}{
		"logRetentionBytes": "5000000",
	}
	svcInstance.Parameters["logRetentionBytes"] = "5000000"
	sw = apitesthelper.NewSyncWait(t)
	_, err = dbAPI.UpdateServiceInstance(ctx1, svcInstanceParam, func(ctx context.Context, i interface{}) error {
		scopedEntity := i.(model.ScopedEntity)
		svcInstanceCB := scopedEntity.Doc.(model.ServiceInstance)
		svcInstanceCB.ID = svcInstance.ID
		svcInstanceCB.Version = svcInstance.Version
		svcInstanceCB.CreatedAt = svcInstance.CreatedAt
		svcInstanceCB.UpdatedAt = svcInstance.UpdatedAt
		t.Logf("callback received with %+v", svcInstanceCB)
		if !reflect.DeepEqual(svcInstance, &svcInstanceCB) {
			t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcInstance, &svcInstanceCB)
		}
		sw.Done()
		return nil
	})
	require.NoError(t, err)
	sw.WaitWithTimeout()
	svcInstanceResp, err = dbAPI.GetServiceInstance(ctx1, createResp.ID)
	require.NoError(t, err)
	t.Logf("service instance: %+v", svcInstanceResp)
	svcInstance.ID = svcInstanceResp.ID
	svcInstance.Version = svcInstanceResp.Version
	svcInstance.CreatedAt = svcInstanceResp.CreatedAt
	svcInstance.UpdatedAt = svcInstanceResp.UpdatedAt
	if !reflect.DeepEqual(svcInstance, &svcInstanceResp) {
		t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcInstance, &svcInstanceResp)
	}
	sw = apitesthelper.NewSyncWait(t)
	for _, edgeDevice := range edgeDevices {
		// Downgrade just for testing
		setEdgeDeviceVersion(t, dbAPI, tenantID, edgeDevice.ID, "v1.0.0")
	}
	callbackEdgeCount := 0
	_, err = dbAPI.UpdateServiceInstance(ctx1, svcInstanceParam, func(ctx context.Context, i interface{}) error {
		scopedEntity := i.(model.ScopedEntity)
		callbackEdgeCount += len(scopedEntity.EdgeIDs)
		sw.Done()
		return nil
	})
	require.NoError(t, err)
	sw.WaitWithTimeout()
	if callbackEdgeCount != 0 {
		t.Fatalf("expected 0 callback edge count, found %d", callbackEdgeCount)
	}
	eventFilter := model.EventFilter{Path: ePath, Keys: map[string]string{"type": "ALERT"}}
	events, err := dbAPI.QueryEvents(ctx1, eventFilter)
	t.Logf("path %s, events +%v\n", ePath, events)
	require.NoError(t, err)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, found %+v", events)
	}
	if !strings.Contains(events[0].Message, "Minimum Service Domain version") {
		t.Fatalf("expected to fail due to min version violation, found %s", events[0].Message)
	}
	sw = apitesthelper.NewSyncWait(t)
	_, err = dbAPI.DeleteServiceInstance(ctx1, createResp.ID, func(ctx context.Context, i interface{}) error {
		scopedEntity := i.(model.ScopedEntity)
		svcInstanceCB := scopedEntity.Doc.(model.ServiceInstance)
		svcInstanceCB.ID = svcInstance.ID
		svcInstanceCB.Version = svcInstance.Version
		svcInstanceCB.CreatedAt = svcInstance.CreatedAt
		svcInstanceCB.UpdatedAt = svcInstance.UpdatedAt
		t.Logf("callback received with %+v", svcInstanceCB)
		if !reflect.DeepEqual(svcInstance, &svcInstanceCB) {
			t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcInstance, &svcInstanceCB)
		}

		sw.Done()
		return nil
	})
	require.NoError(t, err)
	sw.WaitWithTimeout()
}

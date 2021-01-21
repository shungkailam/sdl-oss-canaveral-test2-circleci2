package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func createServiceBinding(ctx context.Context, t *testing.T, dbAPI api.ObjectModelAPI, tenantID, svcClassID, edgeClusterID, projectID string) *model.ServiceBinding {
	createParam := model.ServiceBindingParam{Name: "binding-" + base.GetUUID(), SvcClassID: svcClassID}
	if strings.TrimSpace(edgeClusterID) != "" {
		createParam.BindResource = &model.ServiceBindingResource{Type: model.ServiceBindingServiceDomainResource, ID: edgeClusterID}
	} else {
		createParam.BindResource = &model.ServiceBindingResource{Type: model.ServiceBindingProjectResource, ID: projectID}
	}
	createParam.Parameters = map[string]interface{}{
		"access-level": "ReadOnly",
	}
	resp, err := dbAPI.CreateServiceBinding(ctx, &createParam, nil)
	require.NoError(t, err)
	createResp := resp.(model.CreateDocumentResponseV2)
	svcBinding, err := dbAPI.GetServiceBinding(ctx, createResp.ID)
	require.NoError(t, err)
	return &svcBinding
}

func TestServiceBinding(t *testing.T) {
	ctx := context.TODO()
	opCtx := base.GetOperatorContext(ctx)
	dbAPI := newObjectModelAPI(t)
	defer dbAPI.Close()
	svcClass := createServiceClassPayload(t)
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

	createParam := model.ServiceBindingParam{Name: "binding-" + base.GetUUID(), SvcClassID: svcClass.ID, BindResource: &model.ServiceBindingResource{Type: model.ServiceBindingServiceDomainResource, ID: edgeClusterID}}
	svcBinding := &model.ServiceBinding{}
	err = base.Convert(createParam, svcBinding)
	require.NoError(t, err)
	svcBinding.TenantID = tenantID
	svcBinding.ServiceClassCommon = svcClass.ServiceClassCommon
	svcBinding.SvcClassName = svcClass.Name
	// Create without the required parameter
	_, err = dbAPI.CreateServiceBinding(ctx1, &createParam, nil)
	require.Error(t, err)
	sw := apitesthelper.NewSyncWait(t)
	createParam.Parameters = map[string]interface{}{
		"access-level": "jsjs",
	}
	// Create with wrong enum value
	_, err = dbAPI.CreateServiceBinding(ctx1, &createParam, nil)
	require.Error(t, err)
	createParam.Parameters = map[string]interface{}{
		"access-level": "ReadOnly",
	}
	svcBinding.Parameters = createParam.Parameters
	// Create the service binding before creating the service instance
	resp, err = dbAPI.CreateServiceBinding(ctx1, &createParam, func(ctx context.Context, i interface{}) error {
		scopedEntity := i.(model.ScopedEntity)
		svcBindingCB := scopedEntity.Doc.(model.ServiceBinding)
		if len(svcBindingCB.ID) == 0 {
			t.Fatal("service binding ID must be set")
		}
		svcBinding.ID = svcBindingCB.ID
		svcBinding.MinSvcDomainVersion = svcBindingCB.MinSvcDomainVersion
		svcBinding.Version = svcBindingCB.Version
		svcBinding.CreatedAt = svcBindingCB.CreatedAt
		svcBinding.UpdatedAt = svcBindingCB.UpdatedAt
		t.Logf("callback received with %+v", svcBindingCB)
		if !reflect.DeepEqual(svcBinding, &svcBindingCB) {
			t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcBinding, &svcBindingCB)
		}
		sw.Done()
		return nil
	})
	require.NoError(t, err)
	sw.WaitWithTimeout()
	createResp = resp.(model.CreateDocumentResponseV2)
	defer dbAPI.DeleteServiceBinding(ctx1, createResp.ID, nil)
	createParam.ID = createResp.ID
	// Create the service instance
	svcInstance := createServiceInstance(ctx1, t, dbAPI, tenantID, svcClass.ID, projectID)
	defer dbAPI.DeleteServiceInstance(ctx1, svcInstance.ID, nil)
	createResp = resp.(model.CreateDocumentResponseV2)
	svcBindingResp, err := dbAPI.GetServiceBinding(ctx1, createResp.ID)
	require.NoError(t, err)
	t.Logf("service binding %+v", svcBindingResp)
	if !reflect.DeepEqual(svcBinding, &svcBindingResp) {
		t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcBinding, &svcBindingResp)
	}
	entitiesQueryParam := &model.EntitiesQueryParam{}
	statusQueryParam := &model.ServiceBindingStatusQueryParam{SvcDomainID: edgeClusterID}
	statusListPayload, err := dbAPI.SelectServiceBindingStatuss(ctx1, createResp.ID, entitiesQueryParam, statusQueryParam)
	require.NoError(t, err)
	t.Logf("service binding status: %+v", statusListPayload)
	if len(statusListPayload.SvcBindingStatusList) != 0 || statusListPayload.TotalCount != 0 {
		t.Fatalf("expected 0 service binding status")
	}
	statusEventQueryParam := api.ServiceBindingStatusEventQueryParam{
		ServiceClassCommon: svcBinding.ServiceClassCommon,
		SvcBindingID:       svcBinding.ID,
		SvcDomainID:        edgeClusterID,
		SvcInstanceID:      svcInstance.ID,
		ProjectID:          svcInstance.ScopeID,
	}
	ePathTemplate := model.ServiceBindingStatusProjectScopedEventPath
	ePath, _, err := model.GenerateEventQueryPath(ePathTemplate, statusEventQueryParam)
	require.NoError(t, err)
	bindResultData, _ := json.Marshal(model.ServiceBindingResult{
		Credentials: map[string]interface{}{
			"user": "test",
			"pass": "test@123",
		},
	})
	event := model.Event{
		Path:      ePath,
		State:     string(model.ServiceBindingProvisionedState),
		Type:      "STATUS",
		Timestamp: svcBinding.CreatedAt,
		Properties: map[string]string{
			"bindResult": string(bindResultData),
		},
	}
	_, err = dbAPI.UpsertEvents(ctx1, model.EventUpsertRequest{Events: []model.Event{event}}, nil)
	require.NoError(t, err)
	statusListPayload, err = dbAPI.SelectServiceBindingStatuss(ctx1, createResp.ID, entitiesQueryParam, statusQueryParam)
	require.NoError(t, err)
	if len(statusListPayload.SvcBindingStatusList) == 0 && statusListPayload.TotalCount == 0 {
		t.Fatalf("expected 1 service binding status. %+v", statusListPayload)
	}
	if statusListPayload.SvcBindingStatusList[0].State != model.ServiceBindingProvisionedState {
		t.Fatalf("expected service binding state %s, found %s", model.ServiceBindingProvisionedState, statusListPayload.SvcBindingStatusList[0].State)
	}
	bindResult := statusListPayload.SvcBindingStatusList[0].BindResult
	if bindResult.Credentials == nil || len(bindResult.Credentials) == 0 {
		t.Fatalf("expected service binding state credentials to be present")
	}
	queryParam := &model.ServiceBindingQueryParam{SvcClassID: "blah"}
	listPayload, err := dbAPI.SelectAllServiceBindings(ctx1, entitiesQueryParam, queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcBindingList) != 0 || listPayload.TotalCount != 0 {
		t.Fatalf("expected 0 service bindings. %+v", listPayload)
	}
	queryParam.SvcClassID = svcInstance.SvcClassID
	listPayload, err = dbAPI.SelectAllServiceBindings(ctx1, entitiesQueryParam, queryParam)
	require.NoError(t, err)
	if len(listPayload.SvcBindingList) != 1 || listPayload.TotalCount != 1 {
		t.Fatalf("expected 1 service bindings. %+v", listPayload)
	}
	svcBindingResp = listPayload.SvcBindingList[0]
	svcBinding.Version = svcBindingResp.Version
	svcBinding.CreatedAt = svcBindingResp.CreatedAt
	svcBinding.UpdatedAt = svcBindingResp.UpdatedAt
	if !reflect.DeepEqual(svcBinding, &svcBindingResp) {
		t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcBinding, &svcBindingResp)
	}
	sw = apitesthelper.NewSyncWait(t)
	_, err = dbAPI.DeleteServiceBinding(ctx1, createResp.ID, func(ctx context.Context, i interface{}) error {
		scopedEntity := i.(model.ScopedEntity)
		svcBindingCB := scopedEntity.Doc.(model.ServiceBinding)
		svcBinding.Version = svcBindingCB.Version
		svcBinding.CreatedAt = svcBindingCB.CreatedAt
		svcBinding.UpdatedAt = svcBindingCB.UpdatedAt
		t.Logf("callback received with %+v", svcBindingCB)
		if !reflect.DeepEqual(svcBinding, &svcBindingCB) {
			t.Fatalf("expected:\n%+v\nfound:\n%+v\n", svcBinding, &svcBindingCB)
		}

		sw.Done()
		return nil
	})
	require.NoError(t, err)
	sw.WaitWithTimeout()
	for _, edgeDevice := range edgeDevices {
		// Downgrade just for testing
		setEdgeDeviceVersion(t, dbAPI, tenantID, edgeDevice.ID, "v1.0.0")
	}
	callbackEdgeCount := 0
	sw = apitesthelper.NewSyncWait(t)
	resp, err = dbAPI.CreateServiceBinding(ctx1, &createParam, func(ctx context.Context, i interface{}) error {
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
	statusEventQueryParam = api.ServiceBindingStatusEventQueryParam{
		ServiceClassCommon: svcBinding.ServiceClassCommon,
		SvcBindingID:       svcBinding.ID,
		SvcDomainID:        edgeClusterID,
		SvcInstanceID:      model.ZeroUUID,
	}
	ePathTemplate = model.ServiceBindingStatusServiceDomainScopedEventPath
	ePath, _, err = model.GenerateEventQueryPath(ePathTemplate, statusEventQueryParam)
	require.NoError(t, err)
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
	_, err = dbAPI.DeleteServiceBinding(ctx1, createResp.ID, nil)
	require.NoError(t, err)
}

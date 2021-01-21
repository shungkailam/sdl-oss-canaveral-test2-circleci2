package api_test

import (
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestEdgeDeviceInfo(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeDeviceInfo test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
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
	// admin, no-access, project admin
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})
	// Teardown
	defer func() {
		for _, edgeDevice := range edgeDevices {
			dbAPI.DeleteEdgeDevice(ctx1, edgeDevice.ID, nil)
		}
		dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete EdgeDeviceInfo", func(t *testing.T) {
		t.Log("running Create/Get/Delete EdgeDeviceInfo test")

		// EdgeDeviceInfo is already there as a part of edge creation
		deviceInfo, err := dbAPI.GetEdgeDeviceInfo(ctx1, edgeDevices[1].ID)
		require.NoError(t, err)
		t.Logf("get edge device info successful, %+v", deviceInfo)
		deviceInfos, err := dbAPI.SelectAllEdgeDevicesInfo(ctx1, nil)
		require.NoError(t, err)
		if len(deviceInfos) != 2 {
			t.Fatalf("expect SelectAllEdgeDevicesInfo result length to be 1, got %d instead", len(deviceInfos))
		}
		deviceInfos, err = dbAPI.SelectAllEdgeDevicesInfo(ctx2, nil)
		require.NoError(t, err)
		if len(deviceInfos) != 0 {
			t.Fatalf("expect SelectAllEdgeDevicesInfo for ctx2 result length to be 0, got %d instead", len(deviceInfos))
		}
		deviceInfos, err = dbAPI.SelectAllEdgeDevicesInfo(ctx3, nil)
		require.NoError(t, err)
		if len(deviceInfos) != 2 {
			t.Fatalf("expect SelectAllEdgeDevicesInfo for ctx3 result length to be 2, got %d instead", len(deviceInfos))
		}

		deviceInfos, err = dbAPI.SelectAllEdgeDevicesInfoForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(deviceInfos) != 2 {
			t.Fatalf("expect SelectAllEdgeDevicesInfoForProject result length to be 2, got %d instead", len(deviceInfos))
		}
		// update edge device info
		NumCPU := "4"
		TotalMemoryKB := "2441"
		TotalStorageKB := "1223"
		GPUInfo := "NVIDIA"
		CPUUsage := "143221"
		MemoryFreeKB := "2121"
		StorageFreeKB := "1234"
		GPUUsage := "12121"
		Artifacts := map[string]interface{}{"edgeIP": "123.321.123.321"}

		healthBits := map[string]bool{
			"DiskPressure":   true,
			"MemoryPressure": false,
			"Ready":          true,
		}
		doc := model.EdgeDeviceInfo{
			EdgeDeviceScopedModel: model.EdgeDeviceScopedModel{
				ClusterEntityModel: model.ClusterEntityModel{
					BaseModel: model.BaseModel{
						ID:       edgeDevices[1].ID,
						TenantID: tenantID,
						Version:  0,
					},
				},
				DeviceID: edgeDevices[1].ID,
			},
			EdgeDeviceInfoCore: model.EdgeDeviceInfoCore{
				NumCPU:         NumCPU,
				TotalMemoryKB:  TotalMemoryKB,
				TotalStorageKB: TotalStorageKB,
				GPUInfo:        GPUInfo,
				CPUUsage:       CPUUsage,
				MemoryFreeKB:   MemoryFreeKB,
				StorageFreeKB:  StorageFreeKB,
				GPUUsage:       GPUUsage,
			},
			EdgeDeviceStatus: model.EdgeDeviceStatus{HealthBits: healthBits},
			Artifacts:        Artifacts,
		}

		listener := NewNodeInfoEventListener(edgeClusterID)
		// CreateEdgeDeviceInfo also updates
		upResp, err := dbAPI.CreateEdgeDeviceInfo(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Logf("update edge device info successful, %+v", upResp)
		event, ok := listener.GetEvent().(*model.NodeInfoEvent)
		if !ok {
			t.Fatal("Failed to get event in time")
		}
		t.Logf("Got event: %+v", *event.Info)
		if event.Info.SvcDomainID != edgeClusterID {
			t.Fatalf("Expected cluster ID %s, found %s", edgeClusterID, event.Info.SvcDomainID)
		}
		err = dbAPI.UpdateEdgeDeviceOnboarded(ctx1, edgeDevices[1].ID, "fake-ssh-key")
		require.NoError(t, err)
		deviceInfo, err = dbAPI.GetEdgeDeviceInfo(ctx1, edgeDevices[1].ID)
		require.NoError(t, err)
		t.Logf("get edge device info successful, %+v", deviceInfo)

		if deviceInfo.Onboarded != true || deviceInfo.Connected != true || deviceInfo.Healthy != false || !reflect.DeepEqual(deviceInfo.HealthBits, healthBits) ||
			deviceInfo.ClusterID != edgeDevices[1].ClusterID || deviceInfo.ID != edgeDevices[1].ID || deviceInfo.TenantID != tenantID || deviceInfo.NumCPU != NumCPU ||
			deviceInfo.TotalMemoryKB != TotalMemoryKB || deviceInfo.TotalStorageKB != TotalStorageKB || deviceInfo.GPUInfo != GPUInfo || deviceInfo.CPUUsage != CPUUsage ||
			deviceInfo.MemoryFreeKB != MemoryFreeKB || deviceInfo.StorageFreeKB != StorageFreeKB || deviceInfo.GPUUsage != GPUUsage || deviceInfo.Artifacts["edgeIP"] != Artifacts["edgeIP"] {

			t.Log("deviceInfo.ClusterID != edgeDevices[1].ClusterID", deviceInfo.ClusterID != edgeDevices[1].ClusterID)
			t.Log("deviceInfo.ID != edgeID", deviceInfo.ID != edgeDevices[1].ID)
			t.Log("deviceInfo.TenantID != tenantID", deviceInfo.TenantID != tenantID)
			t.Log("deviceInfo.NumCPU != NumCPU", deviceInfo.NumCPU != NumCPU)
			t.Log("deviceInfo.TotalMemoryKB != TotalMemoryKB", deviceInfo.TotalMemoryKB != TotalMemoryKB)
			t.Log("deviceInfo.TotalStorageKB != TotalStorageKB", deviceInfo.TotalStorageKB != TotalStorageKB)
			t.Log("deviceInfo.GPUInfo != GPUInfo", deviceInfo.GPUInfo != GPUInfo)
			t.Log("deviceInfo.CPUUsage != CPUUsage", deviceInfo.CPUUsage != CPUUsage)
			t.Log("deviceInfo.MemoryFreeKB != MemoryFreeKB", deviceInfo.MemoryFreeKB != MemoryFreeKB)
			t.Log("deviceInfo.StorageFreeKB != StorageFreeKB", deviceInfo.StorageFreeKB != StorageFreeKB)
			t.Log("deviceInfo.GPUUsage != GPUUsage", deviceInfo.GPUInfo != GPUInfo)
			t.Log("deviceInfo.EdgeArtifacts != EdgeArtifacts", deviceInfo.Artifacts["edgeIP"] != Artifacts["edgeIP"])
			t.Log("deviceInfo.Onboarded != true", deviceInfo.Onboarded != true)
			t.Log("deviceInfo.Connected != true", deviceInfo.Connected != true)
			t.Log("deviceInfo.Healthy != false", deviceInfo.Healthy != false)
			t.Log("!reflect.DeepEqual(deviceInfo.HealthBits, healthBits)", !reflect.DeepEqual(deviceInfo.HealthBits, healthBits))
			t.Fatal("deviceInfo data mismatch")
		}
		healthBits = map[string]bool{
			"DiskPressure":   false,
			"MemoryPressure": false,
			"Ready":          true,
		}
		deviceInfo.HealthBits = healthBits
		// CreateEdgeDeviceInfo also updates
		upResp, err = dbAPI.CreateEdgeDeviceInfo(ctx1, &deviceInfo, nil)
		require.NoError(t, err)
		t.Logf("update edge device info successful, %+v", upResp)
		deviceInfo, err = dbAPI.GetEdgeDeviceInfo(ctx1, edgeDevices[1].ID)
		require.NoError(t, err)
		t.Logf("get edge device info successful, %+v", deviceInfo)

		if deviceInfo.Onboarded != true || deviceInfo.Connected != true || deviceInfo.Healthy != true || !reflect.DeepEqual(deviceInfo.HealthBits, healthBits) ||
			deviceInfo.ClusterID != edgeDevices[1].ClusterID || deviceInfo.ID != edgeDevices[1].ID || deviceInfo.TenantID != tenantID || deviceInfo.NumCPU != NumCPU ||
			deviceInfo.TotalMemoryKB != TotalMemoryKB || deviceInfo.TotalStorageKB != TotalStorageKB || deviceInfo.GPUInfo != GPUInfo || deviceInfo.CPUUsage != CPUUsage ||
			deviceInfo.MemoryFreeKB != MemoryFreeKB || deviceInfo.StorageFreeKB != StorageFreeKB || deviceInfo.GPUUsage != GPUUsage || deviceInfo.Artifacts["edgeIP"] != Artifacts["edgeIP"] {

			t.Log("deviceInfo.ClusterID != edgeDevices[1].ClusterID", deviceInfo.ClusterID != edgeDevices[1].ClusterID)
			t.Log("deviceInfo.ID != edgeID", deviceInfo.ID != edgeDevices[1].ID)
			t.Log("deviceInfo.TenantID != tenantID", deviceInfo.TenantID != tenantID)
			t.Log("deviceInfo.NumCPU != NumCPU", deviceInfo.NumCPU != NumCPU)
			t.Log("deviceInfo.TotalMemoryKB != TotalMemoryKB", deviceInfo.TotalMemoryKB != TotalMemoryKB)
			t.Log("deviceInfo.TotalStorageKB != TotalStorageKB", deviceInfo.TotalStorageKB != TotalStorageKB)
			t.Log("deviceInfo.GPUInfo != GPUInfo", deviceInfo.GPUInfo != GPUInfo)
			t.Log("deviceInfo.CPUUsage != CPUUsage", deviceInfo.CPUUsage != CPUUsage)
			t.Log("deviceInfo.MemoryFreeKB != MemoryFreeKB", deviceInfo.MemoryFreeKB != MemoryFreeKB)
			t.Log("deviceInfo.StorageFreeKB != StorageFreeKB", deviceInfo.StorageFreeKB != StorageFreeKB)
			t.Log("deviceInfo.GPUUsage != GPUUsage", deviceInfo.GPUInfo != GPUUsage)
			t.Log("deviceInfo.EdgeArtifacts != EdgeArtifacts", deviceInfo.Artifacts["edgeIP"] != Artifacts["edgeIP"])
			t.Log("deviceInfo.Onboarded != true", deviceInfo.Onboarded != true)
			t.Log("deviceInfo.Connected != true", deviceInfo.Connected != true)
			t.Log("deviceInfo.Healthy != true", deviceInfo.Healthy != true)
			t.Fatal("deviceInfo data mismatch")
		}
	})
}

package api_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

type NodeInfoEventListener struct {
	ch          chan base.Event
	svcDomainID string
}

func (listener *NodeInfoEventListener) OnEvent(
	ctx context.Context,
	event base.Event,
) error {
	nodeInfoEvent, ok := event.(*model.NodeInfoEvent)
	if !ok || nodeInfoEvent.Info.SvcDomainID != listener.svcDomainID {
		fmt.Printf("Ignoring uninterested event: %+v\n", *nodeInfoEvent.Info)
	} else {
		listener.ch <- event
	}
	return nil
}

func (listener *NodeInfoEventListener) EventName() string {
	return model.NodeInfoEventName
}

func (listener *NodeInfoEventListener) GetEvent() base.Event {
	select {
	case event := <-listener.ch:
		return event
	case <-time.After(5 * time.Second):
		return nil
	}
}

func NewNodeInfoEventListener(svcDomainID string) *NodeInfoEventListener {
	listener := &NodeInfoEventListener{ch: make(chan base.Event, 1), svcDomainID: svcDomainID}
	base.Publisher.Subscribe(listener)
	return listener
}

func TestNodeInfo(t *testing.T) {
	t.Parallel()
	t.Log("running TestNodeInfo test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	nodes := createNodeWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}}, "edge", 2)
	svcDomainID := nodes[0].SvcDomainID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	projectID := project.ID
	// admin, no-access, project admin
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})
	// Teardown
	defer func() {
		for _, node := range nodes {
			dbAPI.DeleteEdgeDevice(ctx1, node.ID, nil)
		}
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete NodeInfo", func(t *testing.T) {
		t.Log("running Create/Get/Delete NodeInfo test")

		// NodeInfo is already there as a part of node creation
		nodeInfo, err := dbAPI.GetNodeInfo(ctx1, nodes[0].ID)
		require.NoError(t, err)
		t.Logf("get node info successful, %+v", nodeInfo)
		nodeInfos, err := dbAPI.SelectAllNodesInfo(ctx1, nil)
		require.NoError(t, err)
		if len(nodeInfos) != 2 {
			t.Fatalf("expect SelectAllNodesInfo result length to be 1, got %d instead", len(nodeInfos))
		}
		nodeInfos, err = dbAPI.SelectAllNodesInfo(ctx2, nil)
		require.NoError(t, err)
		if len(nodeInfos) != 0 {
			t.Fatalf("expect SelectAllNodesInfo for ctx2 result length to be 0, got %d instead", len(nodeInfos))
		}
		nodeInfos, err = dbAPI.SelectAllNodesInfo(ctx3, nil)
		require.NoError(t, err)
		if len(nodeInfos) != 2 {
			t.Fatalf("expect SelectAllNodesInfo for ctx3 result length to be 2, got %d instead", len(nodeInfos))
		}

		nodeInfos, err = dbAPI.SelectAllNodesInfoForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(nodeInfos) != 2 {
			t.Fatalf("expect SelectAllNodesInfoForProject result length to be 2, got %d instead", len(nodeInfos))
		}
		// update node info
		NumCPU := "4"
		TotalMemoryKB := "2441"
		TotalStorageKB := "1223"
		GPUInfo := "NVIDIA"
		CPUUsage := "143221"
		MemoryFreeKB := "2121"
		StorageFreeKB := "1234"
		GPUUsage := "12121"
		NodeVersion := "v1.15.0"
		Artifacts := map[string]interface{}{"nodeIP": "123.321.123.321"}

		healthBits := map[string]bool{
			"DiskPressure":   true,
			"MemoryPressure": false,
			"Ready":          true,
		}
		doc := model.NodeInfo{
			NodeEntityModel: model.NodeEntityModel{
				ServiceDomainEntityModel: model.ServiceDomainEntityModel{
					BaseModel: model.BaseModel{
						ID:       nodes[0].ID,
						TenantID: tenantID,
						Version:  0,
					},
				},
				NodeID: nodes[0].ID,
			},
			NodeInfoCore: model.NodeInfoCore{
				NumCPU:         NumCPU,
				TotalMemoryKB:  TotalMemoryKB,
				TotalStorageKB: TotalStorageKB,
				GPUInfo:        GPUInfo,
				CPUUsage:       CPUUsage,
				MemoryFreeKB:   MemoryFreeKB,
				StorageFreeKB:  StorageFreeKB,
				GPUUsage:       GPUUsage,
			},
			NodeStatus: model.NodeStatus{HealthBits: healthBits},
			Artifacts:  Artifacts,
		}

		listener := NewNodeInfoEventListener(svcDomainID)
		// CreateNodeInfo also updates
		upResp, err := dbAPI.CreateNodeInfo(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Logf("update node info successful, %+v", upResp)
		event, ok := listener.GetEvent().(*model.NodeInfoEvent)
		if !ok {
			t.Fatal("Failed to get event in time")
		}
		t.Logf("Got event: %+v", *event.Info)
		if event.Info.SvcDomainID != svcDomainID {
			t.Fatalf("Expected service domain ID %s, found %s", svcDomainID, event.Info.SvcDomainID)
		}
		// Use a context without any authContext information
		err = dbAPI.UpdateNodeOnboarded(context.Background(), &model.NodeOnboardInfo{NodeID: nodes[0].ID, SSHPublicKey: "fake-ssh-key", NodeVersion: NodeVersion})
		require.NoError(t, err)
		nodeInfo, err = dbAPI.GetNodeInfo(ctx1, nodes[0].ID)
		require.NoError(t, err)
		t.Logf("get node info successful, %+v", nodeInfo)

		if nodeInfo.Onboarded != true || nodeInfo.Connected != true || nodeInfo.Healthy != false || !reflect.DeepEqual(nodeInfo.HealthBits, healthBits) ||
			nodeInfo.SvcDomainID != nodes[0].SvcDomainID || nodeInfo.ID != nodes[0].ID || nodeInfo.TenantID != tenantID || nodeInfo.NumCPU != NumCPU ||
			nodeInfo.TotalMemoryKB != TotalMemoryKB || nodeInfo.TotalStorageKB != TotalStorageKB || nodeInfo.GPUInfo != GPUInfo || nodeInfo.CPUUsage != CPUUsage ||
			nodeInfo.MemoryFreeKB != MemoryFreeKB || nodeInfo.StorageFreeKB != StorageFreeKB || nodeInfo.GPUUsage != GPUUsage ||
			nodeInfo.Artifacts["nodeIP"] != Artifacts["nodeIP"] || nodeInfo.NodeVersion == nil || *nodeInfo.NodeVersion != NodeVersion {

			t.Log("nodeInfo.SvcDomainID != nodes[0].SvcDomainID", nodeInfo.SvcDomainID != nodes[0].SvcDomainID)
			t.Log("nodeInfo.ID != nodeID", nodeInfo.ID != nodes[0].ID)
			t.Log("nodeInfo.TenantID != tenantID", nodeInfo.TenantID != tenantID)
			t.Log("nodeInfo.NumCPU != NumCPU", nodeInfo.NumCPU != NumCPU)
			t.Log("nodeInfo.TotalMemoryKB != TotalMemoryKB", nodeInfo.TotalMemoryKB != TotalMemoryKB)
			t.Log("nodeInfo.TotalStorageKB != TotalStorageKB", nodeInfo.TotalStorageKB != TotalStorageKB)
			t.Log("nodeInfo.GPUInfo != GPUInfo", nodeInfo.GPUInfo != GPUInfo)
			t.Log("nodeInfo.CPUUsage != CPUUsage", nodeInfo.CPUUsage != CPUUsage)
			t.Log("nodeInfo.MemoryFreeKB != MemoryFreeKB", nodeInfo.MemoryFreeKB != MemoryFreeKB)
			t.Log("nodeInfo.StorageFreeKB != StorageFreeKB", nodeInfo.StorageFreeKB != StorageFreeKB)
			t.Log("nodeInfo.GPUUsage != GPUUsage", nodeInfo.GPUUsage != GPUUsage)
			t.Log("nodeInfo.EdgeArtifacts != EdgeArtifacts", nodeInfo.Artifacts["nodeIP"] != Artifacts["nodeIP"])
			t.Log("nodeInfo.Onboarded != true", nodeInfo.Onboarded != true)
			t.Log("nodeInfo.Connected != true", nodeInfo.Connected != true)
			t.Log("nodeInfo.Healthy != false", nodeInfo.Healthy != false)
			t.Log("!reflect.DeepEqual(nodeInfo.HealthBits, healthBits)", !reflect.DeepEqual(nodeInfo.HealthBits, healthBits))
			t.Log("nodeInfo.NodeVersion == nil", nodeInfo.NodeVersion == nil)
			t.Log("*nodeInfo.NodeVersion !=", NodeVersion, *nodeInfo.NodeVersion != NodeVersion)
			t.Log("nodeInfo.HealthStatus == UNKNOWN", nodeInfo.HealthStatus == model.NodeHealthStatusUnknown)
			t.Fatal("nodeInfo data mismatch")
		}
		healthBits = map[string]bool{
			"DiskPressure":   false,
			"MemoryPressure": false,
			"Ready":          true,
		}
		nodeInfo.HealthBits = healthBits
		// CreateNodeInfo also updates
		upResp, err = dbAPI.CreateNodeInfo(ctx1, &nodeInfo, nil)
		require.NoError(t, err)
		t.Logf("update node info successful, %+v", upResp)
		nodeInfo, err = dbAPI.GetNodeInfo(ctx1, nodes[0].ID)
		require.NoError(t, err)
		t.Logf("get node info successful, %+v", nodeInfo)

		if nodeInfo.Onboarded != true || nodeInfo.Connected != true || nodeInfo.Healthy != true || !reflect.DeepEqual(nodeInfo.HealthBits, healthBits) ||
			nodeInfo.SvcDomainID != nodes[0].SvcDomainID || nodeInfo.ID != nodes[0].ID || nodeInfo.TenantID != tenantID || nodeInfo.NumCPU != NumCPU ||
			nodeInfo.TotalMemoryKB != TotalMemoryKB || nodeInfo.TotalStorageKB != TotalStorageKB || nodeInfo.GPUInfo != GPUInfo || nodeInfo.CPUUsage != CPUUsage ||
			nodeInfo.MemoryFreeKB != MemoryFreeKB || nodeInfo.StorageFreeKB != StorageFreeKB || nodeInfo.GPUUsage != GPUUsage || nodeInfo.Artifacts["nodeIP"] != Artifacts["nodeIP"] {

			t.Log("nodeInfo.SvcDomainID != nodes[0].SvcDomainID", nodeInfo.SvcDomainID != nodes[0].SvcDomainID)
			t.Log("nodeInfo.ID != nodeID", nodeInfo.ID != nodes[0].ID)
			t.Log("nodeInfo.TenantID != tenantID", nodeInfo.TenantID != tenantID)
			t.Log("nodeInfo.NumCPU != NumCPU", nodeInfo.NumCPU != NumCPU)
			t.Log("nodeInfo.TotalMemoryKB != TotalMemoryKB", nodeInfo.TotalMemoryKB != TotalMemoryKB)
			t.Log("nodeInfo.TotalStorageKB != TotalStorageKB", nodeInfo.TotalStorageKB != TotalStorageKB)
			t.Log("nodeInfo.GPUInfo != GPUInfo", nodeInfo.GPUInfo != GPUInfo)
			t.Log("nodeInfo.CPUUsage != CPUUsage", nodeInfo.CPUUsage != CPUUsage)
			t.Log("nodeInfo.MemoryFreeKB != MemoryFreeKB", nodeInfo.MemoryFreeKB != MemoryFreeKB)
			t.Log("nodeInfo.StorageFreeKB != StorageFreeKB", nodeInfo.StorageFreeKB != StorageFreeKB)
			t.Log("nodeInfo.GPUUsage != GPUUsage", nodeInfo.GPUInfo != GPUUsage)
			t.Log("nodeInfo.EdgeArtifacts != EdgeArtifacts", nodeInfo.Artifacts["nodeIP"] != Artifacts["nodeIP"])
			t.Log("nodeInfo.Onboarded != true", nodeInfo.Onboarded != true)
			t.Log("nodeInfo.Connected != false", nodeInfo.Connected != true)
			t.Log("nodeInfo.Healthy != true", nodeInfo.Healthy != true)
			t.Log("nodeInfo.HealthStatus == HEALTHY", nodeInfo.HealthStatus == model.NodeHealthStatusHealthy)
			t.Fatal("nodeInfo data mismatch")
		}
	})
}

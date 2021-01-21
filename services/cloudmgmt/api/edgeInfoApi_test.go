package api_test

import (
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestEdgeInfo(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeInfo test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	edge2 := createEdge(t, dbAPI, tenantID)
	edgeID2 := edge2.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, []string{edgeID})
	projectID := project.ID
	ctx, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteEdge(ctx, edgeID2, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete EdgeInfo", func(t *testing.T) {
		t.Log("running Create/Get/Delete EdgeInfo test")

		// EdgeInfo is already there as a part of edge creation
		edgeInfo, err := dbAPI.GetEdgeInfo(ctx, edgeID)
		require.NoError(t, err)
		t.Logf("get edge info successful, %+v", edgeInfo)

		edgeInfos, err := dbAPI.SelectAllEdgesInfo(ctx, nil)
		require.NoError(t, err)
		if len(edgeInfos) != 2 {
			t.Fatalf("expect SelectAllEdgesInfo result length to be 2, got %d instead", len(edgeInfos))
		}
		edgeInfos, err = dbAPI.SelectAllEdgesInfo(ctx2, nil)
		require.NoError(t, err)
		if len(edgeInfos) != 0 {
			t.Fatalf("expect SelectAllEdgesInfo for ctx2 result length to be 0, got %d instead", len(edgeInfos))
		}
		edgeInfos, err = dbAPI.SelectAllEdgesInfo(ctx3, nil)
		require.NoError(t, err)
		if len(edgeInfos) != 1 {
			t.Fatalf("expect SelectAllEdgesInfo for ctx2 result length to be 1, got %d instead", len(edgeInfos))
		}

		edgeInfos, err = dbAPI.SelectAllEdgesInfoForProject(ctx, projectID, nil)
		require.NoError(t, err)
		if len(edgeInfos) != 1 {
			t.Fatalf("expect SelectAllEdgesInfoForProject result length to be 1, got %d instead", len(edgeInfos))
		}
		edgeInfos, err = dbAPI.SelectAllEdgesInfoForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect SelectAllEdgesInfoForProject to fail for ctx2")
		edgeInfos, err = dbAPI.SelectAllEdgesInfoForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(edgeInfos) != 1 {
			t.Fatalf("expect SelectAllEdgesInfoForProject for ctx3 result length to be 1, got %d instead", len(edgeInfos))
		}

		// update edge info
		NumCPU := "4"
		TotalMemoryKB := "2441"
		TotalStorageKB := "1223"
		GPUInfo := "NVIDIA"
		CPUUsage := "143221"
		MemoryFreeKB := "2121"
		StorageFreeKB := "1234"
		GPUUsage := "12121"
		EdgeArtifacts := map[string]interface{}{"edgeIP": "123.321.123.321"}

		doc := model.EdgeUsageInfo{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:       edgeID,
					TenantID: tenantID,
					Version:  0,
				},
				EdgeID: edgeID,
			},
			EdgeInfo: model.EdgeInfo{
				NumCPU:         NumCPU,
				TotalMemoryKB:  TotalMemoryKB,
				TotalStorageKB: TotalStorageKB,
				GPUInfo:        GPUInfo,
				CPUUsage:       CPUUsage,
				MemoryFreeKB:   MemoryFreeKB,
				StorageFreeKB:  StorageFreeKB,
				GPUUsage:       GPUUsage,
			},
			EdgeArtifacts: EdgeArtifacts,
		}
		upResp, err := dbAPI.UpdateEdgeInfo(ctx, &doc, nil)
		require.NoError(t, err)
		t.Logf("update edge info successful, %+v", upResp)

		// get edge info
		edgeInfo, err = dbAPI.GetEdgeInfo(ctx, edgeID)
		require.NoError(t, err)
		t.Logf("get edge info successful, %+v", edgeInfo)

		if edgeInfo.ID != edgeID || edgeInfo.TenantID != tenantID || edgeInfo.NumCPU != NumCPU || edgeInfo.TotalMemoryKB != TotalMemoryKB || edgeInfo.TotalStorageKB != TotalStorageKB || edgeInfo.GPUInfo != GPUInfo || edgeInfo.CPUUsage != CPUUsage || edgeInfo.MemoryFreeKB != MemoryFreeKB || edgeInfo.StorageFreeKB != StorageFreeKB || edgeInfo.GPUUsage != GPUUsage || edgeInfo.EdgeArtifacts["edgeIP"] != EdgeArtifacts["edgeIP"] {
			t.Log("edgeInfo.ID != edgeID", edgeInfo.ID != edgeID)
			t.Log("edgeInfo.TenantID != tenantID", edgeInfo.TenantID != tenantID)
			t.Log("edgeInfo.NumCPU != NumCPU", edgeInfo.NumCPU != NumCPU)
			t.Log("edgeInfo.TotalMemoryKB != TotalMemoryKB", edgeInfo.TotalMemoryKB != TotalMemoryKB)
			t.Log("edgeInfo.TotalStorageKB != TotalStorageKB", edgeInfo.TotalStorageKB != TotalStorageKB)
			t.Log("edgeInfo.GPUInfo != GPUInfo", edgeInfo.GPUInfo != GPUInfo)
			t.Log("edgeInfo.CPUUsage != CPUUsage", edgeInfo.CPUUsage != CPUUsage)
			t.Log("edgeInfo.MemoryFreeKB != MemoryFreeKB", edgeInfo.MemoryFreeKB != MemoryFreeKB)
			t.Log("edgeInfo.StorageFreeKB != StorageFreeKB", edgeInfo.StorageFreeKB != StorageFreeKB)
			t.Log("edgeInfo.GPUUsage != GPUUsage", edgeInfo.GPUInfo != GPUUsage)
			t.Log("edgeInfo.EdgeArtifacts != EdgeArtifacts", edgeInfo.EdgeArtifacts["edgeIP"] != EdgeArtifacts["edgeIP"])
			t.Fatal("edgeInfo data mismatch")
		}

		// update edge info without artifact, edge artifact should be preserved.
		NumCPU = "2"
		TotalMemoryKB = "3241"
		TotalStorageKB = "5432"
		GPUInfo = "AMD"
		CPUUsage = "213"
		MemoryFreeKB = "54325"
		StorageFreeKB = "3421"
		GPUUsage = "345"

		doc = model.EdgeUsageInfo{
			EdgeBaseModel: model.EdgeBaseModel{
				BaseModel: model.BaseModel{
					ID:       edgeID,
					TenantID: tenantID,
					Version:  0,
				},
				EdgeID: edgeID,
			},
			EdgeInfo: model.EdgeInfo{
				NumCPU:         NumCPU,
				TotalMemoryKB:  TotalMemoryKB,
				TotalStorageKB: TotalStorageKB,
				GPUInfo:        GPUInfo,
				CPUUsage:       CPUUsage,
				MemoryFreeKB:   MemoryFreeKB,
				StorageFreeKB:  StorageFreeKB,
				GPUUsage:       GPUUsage,
			},
		}
		upResp, err = dbAPI.UpdateEdgeInfo(ctx, &doc, nil)
		require.NoError(t, err)
		t.Logf("update edge info successful, %+v", upResp)

		// get edge info
		edgeInfo, err = dbAPI.GetEdgeInfo(ctx, edgeID)
		require.NoError(t, err)
		t.Logf("get edge info successful, %+v", edgeInfo)

		if edgeInfo.ID != edgeID || edgeInfo.TenantID != tenantID || edgeInfo.NumCPU != NumCPU || edgeInfo.TotalMemoryKB != TotalMemoryKB || edgeInfo.TotalStorageKB != TotalStorageKB || edgeInfo.GPUInfo != GPUInfo || edgeInfo.CPUUsage != CPUUsage || edgeInfo.MemoryFreeKB != MemoryFreeKB || edgeInfo.StorageFreeKB != StorageFreeKB || edgeInfo.GPUUsage != GPUUsage || edgeInfo.EdgeArtifacts["edgeIP"] != EdgeArtifacts["edgeIP"] {
			t.Log("edgeInfo.ID != edgeID", edgeInfo.ID != edgeID)
			t.Log("edgeInfo.TenantID != tenantID", edgeInfo.TenantID != tenantID)
			t.Log("edgeInfo.NumCPU != NumCPU", edgeInfo.NumCPU != NumCPU)
			t.Log("edgeInfo.TotalMemoryKB != TotalMemoryKB", edgeInfo.TotalMemoryKB != TotalMemoryKB)
			t.Log("edgeInfo.TotalStorageKB != TotalStorageKB", edgeInfo.TotalStorageKB != TotalStorageKB)
			t.Log("edgeInfo.GPUInfo != GPUInfo", edgeInfo.GPUInfo != GPUInfo)
			t.Log("edgeInfo.CPUUsage != CPUUsage", edgeInfo.CPUUsage != CPUUsage)
			t.Log("edgeInfo.MemoryFreeKB != MemoryFreeKB", edgeInfo.MemoryFreeKB != MemoryFreeKB)
			t.Log("edgeInfo.StorageFreeKB != StorageFreeKB", edgeInfo.StorageFreeKB != StorageFreeKB)
			t.Log("edgeInfo.GPUUsage != GPUUsage", edgeInfo.GPUInfo != GPUUsage)
			t.Log("edgeInfo.EdgeArtifacts != EdgeArtifacts", edgeInfo.EdgeArtifacts["edgeIP"] != EdgeArtifacts["edgeIP"])
			t.Fatal("edgeInfo data mismatch")
		}

		// delete edge info
		delResp, err := dbAPI.DeleteEdgeInfo(ctx, edgeID, nil)
		require.NoError(t, err)
		t.Logf("delete edge info successful, %v", delResp)

	})

	// select all edges info
	t.Run("SelectAllEdgesInfo", func(t *testing.T) {
		t.Log("running SelectAllEdgesInfo test")
		edgeInfo, err := dbAPI.SelectAllEdgesInfo(ctx, nil)
		require.NoError(t, err)
		for _, edgeInfo := range edgeInfo {
			testForMarshallability(t, edgeInfo)
		}
	})
}

package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"cloudservices/operator/releases"
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/thoas/go-funk"
)

func createDummyNode(dbAPI api.ObjectModelAPI, tenantID string, svcDomainID string) (model.Node, error) {
	ctx := base.GetAdminContextWithTenantID(context.Background(), tenantID)
	nodeSerialNumber := base.GetUUID()
	nodeName := "my-test-node-" + nodeSerialNumber
	nodeIP := "1.1.1.10"
	nodeSubnet := "255.255.255.0"
	nodeGateway := "1.1.1.1"
	// Node object, leave ID blank and let create generate it
	node := model.Node{
		ServiceDomainEntityModel: model.ServiceDomainEntityModel{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			SvcDomainID: svcDomainID,
		},
		NodeCore: model.NodeCore{
			Name:         nodeName,
			SerialNumber: nodeSerialNumber,
			IPAddress:    nodeIP,
			Subnet:       nodeSubnet,
			Gateway:      nodeGateway,
		},
	}
	resp, err := dbAPI.CreateNode(ctx, &node, nil)
	if err != nil {
		return node, err
	}
	node.ID = resp.(model.CreateDocumentResponse).ID
	nodeInfo, err := dbAPI.GetNodeInfo(ctx, node.ID)
	if err != nil {
		return node, err
	}
	// Set version
	nodeInfo.NodeVersion = base.StringPtr("1.15.0")
	nodeInfo.NumCPU = "4"
	nodeInfo.TotalMemoryKB = "2441"
	nodeInfo.TotalStorageKB = "1223"
	nodeInfo.GPUInfo = "NVIDIA"
	nodeInfo.CPUUsage = "143221"
	nodeInfo.MemoryFreeKB = "2121"
	nodeInfo.StorageFreeKB = "1234"
	nodeInfo.GPUUsage = "12121"
	_, err = dbAPI.CreateNodeInfo(ctx, &nodeInfo, nil)
	if err != nil {
		return node, err
	}
	return node, nil
}

func createServiceDomains(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, count int) []model.ServiceDomain {
	ctx := base.GetAdminContextWithTenantID(context.Background(), tenantID)
	svcDomains := make([]model.ServiceDomain, 0, count)
	for i := 0; i < count; i++ {
		svcDomainName := "my-test-service-domain-" + base.GetUUID()
		// Use demo tenant and demo service domain combination to make the service domain connected
		svcDomainID := apitesthelper.GenerateIDWithPrefix("eid-demo-")
		// Service domain object, leave ID blank and let create generate it
		svcDomain := model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       svcDomainID,
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: svcDomainName,
			},
		}
		resp, err := dbAPI.CreateServiceDomain(ctx, &svcDomain, nil)
		if err != nil {
			for _, svcDomain := range svcDomains {
				// clean up
				dbAPI.DeleteServiceDomain(ctx, svcDomain.ID, nil)
			}
			t.Fatal(err)
		}
		svcDomain.ID = resp.(model.CreateDocumentResponse).ID
		svcDomains = append(svcDomains, svcDomain)
		_, err = createDummyNode(dbAPI, tenantID, svcDomain.ID)
		if err != nil {
			for _, svcDomain := range svcDomains {
				// clean up
				dbAPI.DeleteServiceDomain(ctx, svcDomain.ID, nil)
			}
			t.Fatal(err)
		}
	}
	return svcDomains
}

// TODO more tests to follow
func TestDownload(t *testing.T) {
	t.Log("running TestDownload test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	// Use demo tenant and demo service domain combination to make the service domain connected
	tenantID := apitesthelper.GenerateIDWithPrefix("tid-demo-")
	createTenantWithID(t, dbAPI, "test tenant", tenantID)
	ctx := base.GetAdminContextWithTenantID(context.Background(), tenantID)
	svcDomains := createServiceDomains(t, dbAPI, tenantID, 3)
	svcDomainIDs := funk.Map(svcDomains, func(svcDomain model.ServiceDomain) string {
		return svcDomain.ID
	}).([]string)
	defer func() {
		for _, svcDomain := range svcDomains {
			dbAPI.DeleteServiceDomain(ctx, svcDomain.ID, nil)
		}
		dbAPI.DeleteTenant(ctx, tenantID, nil)
	}()
	t.Logf("service domains %+v", svcDomains)
	releaseList, err := releases.GetLatestRelease()
	require.NoError(t, err)
	if len(releaseList) == 0 {
		t.Fatal("No release found")
	}
	latestRelease := releaseList[0].ID
	t.Logf("using latest release %s", latestRelease)
	w := apitesthelper.NewResponseWriter()
	r, err := apitesthelper.StructToReader(&model.SoftwareDownloadCreate{SvcDomainIDs: svcDomainIDs, Release: latestRelease})
	require.NoError(t, err)
	ch := make(chan int)
	err = dbAPI.StartSoftwareDownloadW(ctx, w, r, func(ctx context.Context, i interface{}) error {
		defer func() {
			ch <- 1
		}()
		t.Logf("callback data %+v", i)
		id := model.GetEdgeID(i)
		if id == nil {
			return errors.New("expected service domain id")
		}
		return nil
	})
	require.NoError(t, err)
	select {
	case <-ch:
	case <-time.After(time.Second * 3):
		t.Fatal("timed out waiting for callback")
	}
	createResp := &model.CreateDocumentResponseV2{}
	err = w.GetBody(createResp)
	require.NoError(t, err)
	batchID := createResp.ID
	t.Logf("start download batch: %s", batchID)
	w.Reset()
	err = dbAPI.SelectAllSoftwareDownloadBatchesW(ctx, w, nil)
	require.NoError(t, err)
	downloadBatchesResp := &model.SoftwareUpdateBatchListPayload{}
	err = w.GetBody(downloadBatchesResp)
	require.NoError(t, err)
	t.Logf("Software download batches: %+v\n", downloadBatchesResp)
	w.Reset()
	req := httptest.NewRequest("GET", "http://localhost", nil)
	err = dbAPI.SelectAllSoftwareUpdateServiceDomainsW(ctx, w, req)
	require.NoError(t, err)
	listSvcDomainsResp := &model.SoftwareUpdateServiceDomainListPayload{}
	err = w.GetBody(listSvcDomainsResp)
	require.NoError(t, err)
	t.Logf("Software update service domains: %+v\n", listSvcDomainsResp)
	require.Equal(t, 3, len(listSvcDomainsResp.SvcDomainList))
	require.NoError(t, err)
	req = httptest.NewRequest("GET", fmt.Sprintf("http://localhost?svcDomainId=%s", svcDomainIDs[0]), nil)
	err = dbAPI.SelectAllSoftwareUpdateServiceDomainsW(ctx, w, req)
	require.NoError(t, err)
	listSvcDomainsResp = &model.SoftwareUpdateServiceDomainListPayload{}
	err = w.GetBody(listSvcDomainsResp)
	require.NoError(t, err)
	t.Logf("Software update service domains: %+v\n", listSvcDomainsResp)
	require.Equal(t, 1, len(listSvcDomainsResp.SvcDomainList))
	require.True(t, listSvcDomainsResp.SvcDomainList[0].IsLatestBatch)
	require.NoError(t, err)
}

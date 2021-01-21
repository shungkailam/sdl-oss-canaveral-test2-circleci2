package api_test

import (
	cmAPI "cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"cloudservices/operator/api"
	gapi "cloudservices/operator/generated/grpc"
	"cloudservices/operator/generated/operator/restapi/operations/edge"
	"cloudservices/operator/releases"
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	"github.com/golang/protobuf/ptypes"
	"github.com/thoas/go-funk"
)

func newObjectModelAPI(t *testing.T) cmAPI.ObjectModelAPI {
	dbAPI, err := cmAPI.NewObjectModelAPI()
	require.NoError(t, err)
	return dbAPI
}

func createTenant(t *testing.T, dbAPI cmAPI.ObjectModelAPI, name string) model.Tenant {
	tenantID := base.GetUUID()
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)
	// Create tenant object
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	// create tenant
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create tenant successful, %s", resp)
	return doc
}

func createServiceDomains(t *testing.T, dbAPI cmAPI.ObjectModelAPI, tenantID string, count int) []model.ServiceDomain {
	ctx := base.GetAdminContextWithTenantID(context.Background(), tenantID)
	svcDomains := make([]model.ServiceDomain, 0, count)
	for i := 0; i < count; i++ {
		svcDomainName := "my-test-service-domain-" + base.GetUUID()
		// Service domain object, leave ID blank and let create generate it
		svcDomain := model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       "",
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
	}
	return svcDomains
}

func uploadTestRelease(t *testing.T) string {
	releaseFileName := "test-release.tgz"
	releaseLocalDir := fmt.Sprintf("/tmp/%s", base.GetUUID())
	releaseFilepath := fmt.Sprintf("%s/%s", releaseLocalDir, releaseFileName)
	err := os.MkdirAll(releaseLocalDir, 0777)
	require.NoError(t, err)
	defer os.RemoveAll(releaseLocalDir)
	t.Logf("Using temp directory %s to create %s", releaseLocalDir, releaseFilepath)
	// Create a file and close it for upgrade
	createReleaseTarFile(t, releaseLocalDir, releaseFileName, "test data")
	file, err := os.OpenFile(releaseFilepath, os.O_RDONLY, 0666)
	require.NoError(t, err)
	defer file.Close()
	release, err := releases.UploadRelease(edge.UploadReleaseParams{
		Changelog:    base.StringPtr("blah blah"),
		UpgradeFiles: file,
		UpgradeType:  "major",
	})
	require.NoError(t, err)
	return release
}

func TestSoftwareUpdate(t *testing.T) {
	t.Log("running TestSoftwareUpdate test")
	// Setup
	apiServer := api.NewAPIServer()
	rpcServer := apiServer.RPCServer
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	ctx := base.GetAdminContextWithTenantID(context.Background(), tenantID)
	svcDomains := createServiceDomains(t, dbAPI, tenantID, 3)
	svcDomainIDs := funk.Map(svcDomains, func(svcDomain model.ServiceDomain) string {
		return svcDomain.ID
	}).([]string)
	svcDomainIDInCtx := svcDomainIDs[0]
	downloadBatchID := ""
	upgradeBatchID := ""
	edgeCtx := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "edge",
			"edgeId":      svcDomainIDInCtx,
		},
	})
	defer func() {
		for _, svcDomain := range svcDomains {
			t.Logf("%+v", svcDomain)
			dbAPI.DeleteServiceDomain(ctx, svcDomain.ID, nil)
		}
		if downloadBatchID != "" {
			rpcServer.DeleteBatch(ctx, &gapi.DeleteBatchRequest{BatchId: downloadBatchID})
		}
		if upgradeBatchID != "" {
			rpcServer.DeleteBatch(ctx, &gapi.DeleteBatchRequest{BatchId: upgradeBatchID})
		}
		dbAPI.DeleteTenant(ctx, tenantID, nil)
	}()
	t.Logf("service domains %+v", svcDomains)
	latestRelease := uploadTestRelease(t)
	defer func() {
		releases.DeleteRelease(edge.DeleteReleaseParams{
			ReleaseID: latestRelease,
		})
	}()
	t.Logf("using latest release %s", latestRelease)
	startDownloadResp, err := rpcServer.StartDownload(ctx, &gapi.StartDownloadRequest{SvcDomainIds: svcDomainIDs, Release: latestRelease})
	require.NoError(t, err)
	t.Logf("start download response %+v", startDownloadResp)
	downloadBatchID = startDownloadResp.BatchId
	if downloadBatchID == "" {
		t.Fatal("batch ID is invalid")
	}
	downloadBatchListResp, err := rpcServer.ListDownloadBatches(ctx, &gapi.ListDownloadBatchesRequest{})
	require.NoError(t, err)
	pageInfo := downloadBatchListResp.PageInfo
	if pageInfo.TotalCount != 1 {
		t.Fatalf("expected a batch count of 1, found %d", pageInfo.TotalCount)
	}
	batch := downloadBatchListResp.Batches[0]
	downloadStat, ok := batch.Stats[string(model.DownloadState)]
	if !ok {
		t.Fatalf("expected download stat, found %+v", batch)
	}
	if downloadStat != 3 {
		t.Fatalf("expected download stats of 3, found %d", downloadStat)
	}
	downloadSvcDomainListResp, err := rpcServer.ListDownloadBatchServiceDomains(ctx, &gapi.ListDownloadBatchServiceDomainsRequest{BatchId: downloadBatchID})
	require.NoError(t, err)
	svcDomainsResp := downloadSvcDomainListResp.SvcDomains
	pageInfo = downloadSvcDomainListResp.PageInfo
	if pageInfo.TotalCount != 3 {
		t.Fatalf("expected a batch count of 3, found %d", pageInfo.TotalCount)
	}
	for _, svcDomainResp := range svcDomainsResp {
		if svcDomainResp.BatchId != downloadBatchID {
			t.Fatalf("expected %s, found %s for batch ID", downloadBatchID, svcDomainResp.BatchId)
		}
	}
	allSvcDomainListResp, err := rpcServer.ListServiceDomains(ctx, &gapi.ListServiceDomainsRequest{})
	require.NoError(t, err)
	svcDomainsResp = allSvcDomainListResp.SvcDomains
	t.Logf("service domains %+v", svcDomainsResp)
	pageInfo = allSvcDomainListResp.PageInfo
	if pageInfo.TotalCount != 3 {
		t.Fatalf("expected a batch count of 3, found %d", pageInfo.TotalCount)
	}
	for _, svcDomainResp := range svcDomainsResp {
		if svcDomainResp.BatchId != downloadBatchID {
			t.Fatalf("expected %s, found %s for batch ID", downloadBatchID, svcDomainResp.BatchId)
		}
	}
	downloadUpdateStateResp, err := rpcServer.UpdateDownloadState(ctx, &gapi.UpdateDownloadStateRequest{BatchId: downloadBatchID, SvcDomainId: svcDomains[0].ID, Release: latestRelease, Progress: 10, Eta: 20, State: string(model.DownloadingState)})
	require.NoError(t, err)
	if downloadUpdateStateResp.BatchId != downloadBatchID {
		t.Fatalf("expected %s, found %s for batch ID", downloadBatchID, downloadUpdateStateResp.BatchId)
	}
	if downloadUpdateStateResp.State != string(model.DownloadingState) {
		t.Fatalf("expected %s, found %s for state", model.DownloadingState, downloadUpdateStateResp.State)
	}
	downloadBatchListResp, err = rpcServer.ListDownloadBatches(ctx, &gapi.ListDownloadBatchesRequest{})
	require.NoError(t, err)
	pageInfo = downloadBatchListResp.PageInfo
	if pageInfo.TotalCount != 1 {
		t.Fatalf("expected a batch count of 1, found %d", pageInfo.TotalCount)
	}
	batch = downloadBatchListResp.Batches[0]
	t.Logf("stats: %+v", batch.Stats)
	downloadStat, ok = batch.Stats[string(model.DownloadState)]
	if !ok {
		t.Fatalf("expected download stat, found %+v", batch)
	}
	if downloadStat != 2 {
		t.Fatalf("expected download stats of 2, found %d", downloadStat)
	}
	downloadingStat, ok := batch.Stats[string(model.DownloadingState)]
	if !ok {
		t.Fatalf("expected downloading stat, found %+v", batch)
	}
	if downloadingStat != 1 {
		t.Fatalf("expected downloading stats of 1, found %d", downloadingStat)
	}
	credsResp, err := rpcServer.CreateDownloadCredentials(ctx, &gapi.CreateDownloadCredentialsRequest{BatchId: downloadBatchID, Release: "1.15.0", AccessType: string(model.AWSCredentialsAccessType)})
	require.NoError(t, err)
	t.Logf("credentials: %+v", credsResp)

	_, err = rpcServer.StartDownload(ctx, &gapi.StartDownloadRequest{SvcDomainIds: svcDomainIDs, Release: latestRelease})
	require.Error(t, err, "Expected to fail as another download is in progress")
	currSvcDomainResp, err := rpcServer.GetCurrentServiceDomain(edgeCtx, &gapi.GetCurrentServiceDomainRequest{})
	require.NoError(t, err)
	currSvcDomain := currSvcDomainResp.SvcDomain
	if currSvcDomain.SvcDomainId != svcDomainIDInCtx {
		t.Fatalf("expected service domain %s, found %s", svcDomainIDInCtx, currSvcDomain.SvcDomainId)
	}
	t.Logf("current service domain %+v", currSvcDomain)
	// Even though the time is same, as long as the service domain in context is in the batch in non-terminal state, it should return it
	currSvcDomainResp, err = rpcServer.GetCurrentServiceDomain(edgeCtx, &gapi.GetCurrentServiceDomainRequest{BatchId: downloadBatchID, StateUpdatedAt: currSvcDomain.StateUpdatedAt})
	require.NoError(t, err)
	currSvcDomain = currSvcDomainResp.SvcDomain
	if currSvcDomain == nil {
		t.Fatal("expected the service domain with ID in the context")
	}
	if currSvcDomain.SvcDomainId != svcDomainIDInCtx {
		t.Fatalf("expected service domain %s, found %s", svcDomainIDInCtx, currSvcDomain.SvcDomainId)
	}
	olderTime, err := ptypes.Timestamp(currSvcDomain.StateUpdatedAt)
	require.NoError(t, err)
	olderTime = olderTime.Add(time.Minute * -5)
	gOlderTime, err := ptypes.TimestampProto(olderTime)
	require.NoError(t, err)
	// If unknown old batch ID, the correct service domain in context must be returned
	currSvcDomainResp, err = rpcServer.GetCurrentServiceDomain(edgeCtx, &gapi.GetCurrentServiceDomainRequest{BatchId: "wrong-batch-in-edge", StateUpdatedAt: gOlderTime})
	require.NoError(t, err)
	currSvcDomain = currSvcDomainResp.SvcDomain
	if currSvcDomain == nil {
		t.Fatal("expected the service domain with ID in the context")
	}
	if currSvcDomain.SvcDomainId != svcDomainIDInCtx {
		t.Fatalf("expected service domain %s, found %s", svcDomainIDInCtx, currSvcDomain.SvcDomainId)
	}
	t.Logf("current service domain %+v", currSvcDomain)
	// Download to downloading
	for _, svcDomain := range svcDomains {
		_, err := rpcServer.UpdateDownloadState(ctx, &gapi.UpdateDownloadStateRequest{BatchId: downloadBatchID, SvcDomainId: svcDomain.ID, Progress: 10, Eta: 0, State: string(model.DownloadingState)})
		require.NoError(t, err)
	}
	// Send empty to check if not overwritten by update
	currSvcDomainResp, err = rpcServer.GetCurrentServiceDomain(edgeCtx, &gapi.GetCurrentServiceDomainRequest{})
	require.NoError(t, err)
	currSvcDomain = currSvcDomainResp.SvcDomain
	if currSvcDomain == nil {
		t.Fatal("expected the service domain with ID in the context")
	}
	t.Logf("current service domain %+v", currSvcDomain)
	// Downloading to downloading
	for _, svcDomain := range svcDomains {
		_, err := rpcServer.UpdateDownloadState(ctx, &gapi.UpdateDownloadStateRequest{BatchId: downloadBatchID, SvcDomainId: svcDomain.ID, Progress: 50, Eta: 0, State: string(model.DownloadingState)})
		require.NoError(t, err)
	}
	// Send empty to check if not overwritten by update
	currSvcDomainResp, err = rpcServer.GetCurrentServiceDomain(edgeCtx, &gapi.GetCurrentServiceDomainRequest{})
	require.NoError(t, err)
	currSvcDomain = currSvcDomainResp.SvcDomain
	if currSvcDomain == nil {
		t.Fatal("expected the service domain with ID in the context")
	}
	// Upgrade when there is no version downloaded
	_, err = rpcServer.StartUpgrade(ctx, &gapi.StartUpgradeRequest{SvcDomainIds: svcDomainIDs, Release: "v1.16.0"})
	require.Error(t, err, "Expected to fail as there is no downloaded release")
	t.Logf("current service domain %+v", currSvcDomain)
	for _, svcDomain := range svcDomains {
		_, err := rpcServer.UpdateDownloadState(ctx, &gapi.UpdateDownloadStateRequest{BatchId: downloadBatchID, SvcDomainId: svcDomain.ID, Progress: 100, Eta: 0, State: string(model.DownloadedState)})
		require.NoError(t, err)
	}
	// Idempotent check
	for _, svcDomain := range svcDomains {
		_, err := rpcServer.UpdateDownloadState(ctx, &gapi.UpdateDownloadStateRequest{BatchId: downloadBatchID, SvcDomainId: svcDomain.ID, Progress: 100, Eta: 0, State: string(model.DownloadedState)})
		require.NoError(t, err)
	}
	downloadBatchListResp, err = rpcServer.ListDownloadBatches(ctx, &gapi.ListDownloadBatchesRequest{})
	require.NoError(t, err)
	batch = downloadBatchListResp.Batches[0]
	t.Logf("stats: %+v", batch.Stats)
	downloadedStat, ok := batch.Stats[string(model.DownloadedState)]
	if !ok {
		t.Fatalf("expected downloaded stat, found %+v", batch)
	}
	if downloadedStat != 3 {
		t.Fatalf("expected downloaded stats of 3, found %d", downloadedStat)
	}
	downloadedResp, err := rpcServer.ListDownloadedServiceDomains(ctx, &gapi.ListDownloadedServiceDomainsRequest{Release: latestRelease})
	require.NoError(t, err)
	for _, svcDomain := range svcDomains {
		if !funk.Contains(downloadedResp.SvcDomainIds, svcDomain.ID) {
			t.Fatalf("expected service domain %s to exist in %+v", svcDomain.ID, downloadedResp.SvcDomainIds)
		}
	}
	currSvcDomainResp, err = rpcServer.GetCurrentServiceDomain(edgeCtx, &gapi.GetCurrentServiceDomainRequest{})
	require.NoError(t, err)
	currSvcDomain = currSvcDomainResp.SvcDomain
	if currSvcDomain != nil {
		t.Fatal("expected the service domain to be nil because there is no active download/upgrade")
	}
	t.Logf("current service domain %+v", currSvcDomain)

	// Start upgrade with a non-existing version
	_, err = rpcServer.StartUpgrade(ctx, &gapi.StartUpgradeRequest{SvcDomainIds: svcDomainIDs, Release: "v1.15.0"})
	require.Error(t, err, "Expected to fail as there is no downloaded release")
	// Start upgrade with the downloaded version
	startUpgradeResp, err := rpcServer.StartUpgrade(ctx, &gapi.StartUpgradeRequest{SvcDomainIds: svcDomainIDs, Release: latestRelease})
	require.NoError(t, err)
	upgradeBatchID = startUpgradeResp.BatchId
	upgradeBatchListResp, err := rpcServer.ListUpgradeBatches(ctx, &gapi.ListUpgradeBatchesRequest{})
	require.NoError(t, err)
	t.Logf("Upgrade batches %+v", upgradeBatchListResp)
	pageInfo = upgradeBatchListResp.PageInfo
	if pageInfo.TotalCount != 1 {
		t.Fatalf("expected a batch count of 1, found %d", pageInfo.TotalCount)
	}
	batch = upgradeBatchListResp.Batches[0]
	t.Logf("stats: %+v", batch.Stats)
	upgradeStat, ok := batch.Stats[string(model.UpgradeState)]
	if !ok {
		t.Fatalf("expected upgrade stat, found %+v", batch)
	}
	if upgradeStat != 3 {
		t.Fatalf("expected upgrade stats of 3, found %d", upgradeStat)
	}
	upgradeSvcDomainListResp, err := rpcServer.ListUpgradeBatchServiceDomains(ctx, &gapi.ListUpgradeBatchServiceDomainsRequest{BatchId: upgradeBatchID})
	require.NoError(t, err)
	svcDomainsResp = upgradeSvcDomainListResp.SvcDomains
	pageInfo = upgradeSvcDomainListResp.PageInfo
	if pageInfo.TotalCount != 3 {
		t.Fatalf("expected a batch count of 3, found %d", pageInfo.TotalCount)
	}
	for _, svcDomainResp := range svcDomainsResp {
		if svcDomainResp.BatchId != upgradeBatchID {
			t.Fatalf("expected %s, found %s for batch ID", upgradeBatchID, svcDomainResp.BatchId)
		}
	}
	upgradeUpdateStateResp, err := rpcServer.UpdateUpgradeState(ctx, &gapi.UpdateUpgradeStateRequest{BatchId: upgradeBatchID, SvcDomainId: svcDomains[0].ID, Progress: 10, Eta: 20, State: string(model.UpgradingState)})
	require.NoError(t, err)
	if upgradeUpdateStateResp.BatchId != upgradeBatchID {
		t.Fatalf("expected %s, found %s for batch ID", upgradeBatchID, upgradeUpdateStateResp.BatchId)
	}
	if upgradeUpdateStateResp.State != string(model.UpgradingState) {
		t.Fatalf("expected %s, found %s for state", model.UpgradingState, upgradeUpdateStateResp.State)
	}
	upgradeBatchListResp, err = rpcServer.ListUpgradeBatches(ctx, &gapi.ListUpgradeBatchesRequest{})
	require.NoError(t, err)
	t.Logf("Upgrade batches %+v", upgradeBatchListResp)
	pageInfo = upgradeBatchListResp.PageInfo
	if pageInfo.TotalCount != 1 {
		t.Fatalf("expected a batch count of 1, found %d", pageInfo.TotalCount)
	}
	batch = upgradeBatchListResp.Batches[0]
	t.Logf("stats: %+v", batch.Stats)
	upgradeStat, ok = batch.Stats[string(model.UpgradeState)]
	if !ok {
		t.Fatalf("expected upgrade stat, found %+v", batch)
	}
	if upgradeStat != 2 {
		t.Fatalf("expected upgrade stats of 2, found %d", upgradeStat)
	}
	upgradingStat, ok := batch.Stats[string(model.UpgradingState)]
	if !ok {
		t.Fatalf("expected upgrading stat, found %+v", batch)
	}
	if upgradingStat != 1 {
		t.Fatalf("expected upgrading stats of 1, found %d", upgradingStat)
	}
	allSvcDomainListResp, err = rpcServer.ListServiceDomains(ctx, &gapi.ListServiceDomainsRequest{})
	require.NoError(t, err)
	svcDomainsResp = allSvcDomainListResp.SvcDomains
	t.Logf("service domains %+v", svcDomainsResp)
	pageInfo = allSvcDomainListResp.PageInfo
	if pageInfo.TotalCount != 6 {
		t.Fatalf("expected a batch count of 6, found %d", pageInfo.TotalCount)
	}
	allSvcDomainListResp, err = rpcServer.ListServiceDomains(ctx, &gapi.ListServiceDomainsRequest{IsLatestBatch: true})
	require.NoError(t, err)
	svcDomainsResp = allSvcDomainListResp.SvcDomains
	t.Logf("service domains with latest batches %+v", svcDomainsResp)
	pageInfo = allSvcDomainListResp.PageInfo
	if pageInfo.TotalCount != 3 {
		t.Fatalf("expected a batch count of 3, found %d", pageInfo.TotalCount)
	}
	for _, svcDomainResp := range svcDomainsResp {
		if svcDomainResp.BatchId != upgradeBatchID {
			t.Fatalf("expected %s, found %s for batch ID", upgradeBatchID, svcDomainResp.BatchId)
		}
	}
	for _, svcDomain := range svcDomains {
		_, err := rpcServer.UpdateUpgradeState(ctx, &gapi.UpdateUpgradeStateRequest{BatchId: upgradeBatchID, SvcDomainId: svcDomain.ID, Progress: 100, Eta: 0, State: string(model.UpgradedState)})
		require.NoError(t, err)
	}
	// Idempotent check
	for _, svcDomain := range svcDomains {
		_, err := rpcServer.UpdateUpgradeState(ctx, &gapi.UpdateUpgradeStateRequest{BatchId: upgradeBatchID, SvcDomainId: svcDomain.ID, Progress: 100, Eta: 0, State: string(model.UpgradedState)})
		require.NoError(t, err)
	}
	upgradeBatchListResp, err = rpcServer.ListUpgradeBatches(ctx, &gapi.ListUpgradeBatchesRequest{})
	require.NoError(t, err)
	batch = upgradeBatchListResp.Batches[0]
	t.Logf("stats: %+v", batch.Stats)
	upgradedStat, ok := batch.Stats[string(model.UpgradedState)]
	if !ok {
		t.Fatalf("expected upgraded stat, found %+v", batch)
	}
	if upgradedStat != 3 {
		t.Fatalf("expected upgraded stats of 3, found %d", upgradedStat)
	}
}

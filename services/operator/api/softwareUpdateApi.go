package api

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	gapi "cloudservices/operator/generated/grpc"
	"context"

	"github.com/golang/protobuf/ptypes"
)

// Start download on a batch of service domains
func (server *rpcServer) StartDownload(ctx context.Context, gRequest *gapi.StartDownloadRequest) (*gapi.StartDownloadResponse, error) {
	request := &model.SoftwareDownloadCreate{SvcDomainIDs: gRequest.SvcDomainIds, Release: gRequest.Release}
	response, err := server.updateHandler.StartDownload(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.StartDownloadResponse{BatchId: response.ID, State: string(response.State)}
	return gResponse, nil
}

// Deprecated: Batches are immutable now.
// Update an existing active download batch e.g cancel, retry
func (server *rpcServer) UpdateDownload(ctx context.Context, gRequest *gapi.UpdateDownloadRequest) (*gapi.UpdateDownloadResponse, error) {
	request := &model.SoftwareDownloadUpdate{BatchID: gRequest.BatchId, Command: model.SoftwareDownloadCommand(gRequest.Command)}
	responseList, err := server.updateHandler.UpdateDownload(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.UpdateDownloadResponse{BatchId: responseList[0].BatchID, SvcDomainStates: make([]*gapi.ServiceDomainState, 0, len(responseList))}
	for i := range responseList {
		response := responseList[i]
		svcDomainState := &gapi.ServiceDomainState{SvcDomainId: response.SvcDomainID, State: string(response.State)}
		gResponse.Release = response.Release
		gResponse.SvcDomainStates = append(gResponse.SvcDomainStates, svcDomainState)
	}
	return gResponse, nil
}

// Update the state including progress and eta of an active download
func (server *rpcServer) UpdateDownloadState(ctx context.Context, gRequest *gapi.UpdateDownloadStateRequest) (*gapi.UpdateDownloadStateResponse, error) {
	request := &model.SoftwareUpdateState{}
	err := base.Convert(gRequest, request)
	if err != nil {
		return nil, err
	}
	response, err := server.updateHandler.UpdateDownloadState(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.UpdateDownloadStateResponse{BatchId: response.BatchID, SvcDomainId: response.SvcDomainID, State: string(response.State)}
	return gResponse, nil
}

// CreateDownloadCredentials creates the credentials needed to download software release files
func (server *rpcServer) CreateDownloadCredentials(ctx context.Context, gRequest *gapi.CreateDownloadCredentialsRequest) (*gapi.CreateDownloadCredentialsResponse, error) {
	request := &model.SoftwareUpdateCredentials{BatchID: gRequest.BatchId, Release: gRequest.Release, AccessType: model.SoftwareUpdateCredentialsAccessType(gRequest.AccessType)}
	response, err := server.updateHandler.CreateDownloadCredentials(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.CreateDownloadCredentialsResponse{BatchId: response.BatchID, Release: response.Release, AccessType: string(response.AccessType), Credentials: response.Credentials}
	return gResponse, nil
}

// List all the download batches
func (server *rpcServer) ListDownloadBatches(ctx context.Context, gRequest *gapi.ListDownloadBatchesRequest) (*gapi.ListDownloadBatchesResponse, error) {
	queryParam := getEntitiesQueryParameter(gRequest.QueryParameter)
	response, err := server.updateHandler.ListDownloadBatches(ctx, gRequest.BatchId, queryParam)
	if err != nil {
		return nil, err
	}
	gBatches := make([]*gapi.SoftwareUpdateBatch, 0, len(response.BatchList))
	for _, batch := range response.BatchList {
		gBatch := &gapi.SoftwareUpdateBatch{}
		err = base.Convert(&batch, gBatch)
		if err != nil {
			return nil, err
		}
		gBatches = append(gBatches, gBatch)
	}
	gResponse := &gapi.ListDownloadBatchesResponse{
		PageInfo: getPageInfo(&response.EntityListResponsePayload),
		Batches:  gBatches,
	}
	return gResponse, nil
}

// List all the service domains with download states in a download batch
func (server *rpcServer) ListDownloadBatchServiceDomains(ctx context.Context, gRequest *gapi.ListDownloadBatchServiceDomainsRequest) (*gapi.ListDownloadBatchServiceDomainsResponse, error) {
	queryParam := getEntitiesQueryParameter(gRequest.QueryParameter)
	response, err := server.updateHandler.ListDownloadBatchServiceDomains(ctx, gRequest.BatchId, gRequest.SvcDomainId, queryParam)
	if err != nil {
		return nil, err
	}
	gSvcDomains := make([]*gapi.SoftwareUpdateServiceDomain, 0, len(response.SvcDomainList))
	for _, svcDomain := range response.SvcDomainList {
		gSvcDomain := &gapi.SoftwareUpdateServiceDomain{}
		err = base.Convert(&svcDomain, gSvcDomain)
		if err != nil {
			return nil, err
		}
		gSvcDomains = append(gSvcDomains, gSvcDomain)
	}
	gResponse := &gapi.ListDownloadBatchServiceDomainsResponse{
		PageInfo:   getPageInfo(&response.EntityListResponsePayload),
		SvcDomains: gSvcDomains,
	}
	return gResponse, nil
}

// List all services domains which have downloaded the release in the request
func (server *rpcServer) ListDownloadedServiceDomains(ctx context.Context, gRequest *gapi.ListDownloadedServiceDomainsRequest) (*gapi.ListDownloadedServiceDomainsResponse, error) {
	queryParam := getEntitiesQueryParameter(gRequest.QueryParameter)
	response, err := server.updateHandler.ListDownloadedServiceDomains(ctx, gRequest.Release, queryParam)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.ListDownloadedServiceDomainsResponse{
		PageInfo:     getPageInfo(&response.EntityListResponsePayload),
		SvcDomainIds: response.SvcDomainList,
	}
	return gResponse, nil
}

// List all services domains with the most recent batches */
func (server *rpcServer) ListServiceDomains(ctx context.Context, gRequest *gapi.ListServiceDomainsRequest) (*gapi.ListServiceDomainsResponse, error) {
	queryParam := getEntitiesQueryParameter(gRequest.QueryParameter)
	response, err := server.updateHandler.ListServiceDomains(ctx, model.SoftwareUpdateBatchType(gRequest.Type), gRequest.SvcDomainId, gRequest.IsLatestBatch, queryParam)
	if err != nil {
		return nil, err
	}
	gSvcDomains := make([]*gapi.SoftwareUpdateServiceDomain, 0, len(response.SvcDomainList))
	for _, svcDomain := range response.SvcDomainList {
		gSvcDomain := &gapi.SoftwareUpdateServiceDomain{}
		err = base.Convert(&svcDomain, gSvcDomain)
		if err != nil {
			return nil, err
		}
		gSvcDomains = append(gSvcDomains, gSvcDomain)
	}
	gResponse := &gapi.ListServiceDomainsResponse{
		PageInfo:   getPageInfo(&response.EntityListResponsePayload),
		SvcDomains: gSvcDomains,
	}
	return gResponse, nil
}

// Start upgrade on a batch of service domains
func (server *rpcServer) StartUpgrade(ctx context.Context, gRequest *gapi.StartUpgradeRequest) (*gapi.StartUpgradeResponse, error) {
	request := &model.SoftwareUpgradeCreate{SvcDomainIDs: gRequest.SvcDomainIds, Release: gRequest.Release}
	response, err := server.updateHandler.StartUpgrade(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.StartUpgradeResponse{BatchId: response.ID, State: string(response.State)}
	return gResponse, nil
}

// Deprecated: Batches are immutable now.
// Update an existing active upgrade batch e.g retry
func (server *rpcServer) UpdateUpgrade(ctx context.Context, gRequest *gapi.UpdateUpgradeRequest) (*gapi.UpdateUpgradeResponse, error) {
	request := &model.SoftwareUpgradeUpdate{BatchID: gRequest.BatchId, Command: model.SoftwareUpgradeCommand(gRequest.Command)}
	responseList, err := server.updateHandler.UpdateUpgrade(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.UpdateUpgradeResponse{BatchId: responseList[0].BatchID, SvcDomainStates: make([]*gapi.ServiceDomainState, 0, len(responseList))}
	for i := range responseList {
		response := responseList[i]
		svcDomainState := &gapi.ServiceDomainState{SvcDomainId: response.SvcDomainID, State: string(response.State)}
		gResponse.Release = response.Release
		gResponse.SvcDomainStates = append(gResponse.SvcDomainStates, svcDomainState)
	}
	return gResponse, nil
}

// Update the state including progress and eta of an active upgrade
func (server *rpcServer) UpdateUpgradeState(ctx context.Context, gRequest *gapi.UpdateUpgradeStateRequest) (*gapi.UpdateUpgradeStateResponse, error) {
	request := &model.SoftwareUpdateState{}
	err := base.Convert(gRequest, request)
	if err != nil {
		return nil, err
	}
	response, err := server.updateHandler.UpdateUpgradeState(ctx, request)
	if err != nil {
		return nil, err
	}
	gResponse := &gapi.UpdateUpgradeStateResponse{BatchId: response.BatchID, SvcDomainId: response.SvcDomainID, State: string(response.State)}
	return gResponse, nil
}

// List all the upgrade batches
func (server *rpcServer) ListUpgradeBatches(ctx context.Context, gRequest *gapi.ListUpgradeBatchesRequest) (*gapi.ListUpgradeBatchesResponse, error) {
	queryParam := getEntitiesQueryParameter(gRequest.QueryParameter)
	response, err := server.updateHandler.ListUpgradeBatches(ctx, gRequest.BatchId, queryParam)
	if err != nil {
		return nil, err
	}
	gBatches := make([]*gapi.SoftwareUpdateBatch, 0, len(response.BatchList))
	for _, batch := range response.BatchList {
		gBatch := &gapi.SoftwareUpdateBatch{}
		err = base.Convert(&batch, gBatch)
		if err != nil {
			return nil, err
		}
		gBatches = append(gBatches, gBatch)
	}
	gResponse := &gapi.ListUpgradeBatchesResponse{
		PageInfo: getPageInfo(&response.EntityListResponsePayload),
		Batches:  gBatches,
	}
	return gResponse, nil
}

// List all the service domains with upgrade states in a upgrade batch
func (server *rpcServer) ListUpgradeBatchServiceDomains(ctx context.Context, gRequest *gapi.ListUpgradeBatchServiceDomainsRequest) (*gapi.ListUpgradeBatchServiceDomainsResponse, error) {
	queryParam := getEntitiesQueryParameter(gRequest.QueryParameter)
	response, err := server.updateHandler.ListUpgradeBatchServiceDomains(ctx, gRequest.BatchId, gRequest.SvcDomainId, queryParam)
	if err != nil {
		return nil, err
	}
	gSvcDomains := make([]*gapi.SoftwareUpdateServiceDomain, 0, len(response.SvcDomainList))
	for _, svcDomain := range response.SvcDomainList {
		gSvcDomain := &gapi.SoftwareUpdateServiceDomain{}
		err = base.Convert(&svcDomain, gSvcDomain)
		if err != nil {
			return nil, err
		}
		gSvcDomains = append(gSvcDomains, gSvcDomain)
	}
	gResponse := &gapi.ListUpgradeBatchServiceDomainsResponse{
		PageInfo:   getPageInfo(&response.EntityListResponsePayload),
		SvcDomains: gSvcDomains,
	}
	return gResponse, nil
}

// Gets the currently active service domain
func (server *rpcServer) GetCurrentServiceDomain(ctx context.Context, gRequest *gapi.GetCurrentServiceDomainRequest) (*gapi.GetCurrentServiceDomainResponse, error) {
	gResponse := &gapi.GetCurrentServiceDomainResponse{}
	request := &model.EntityVersionMetadata{ID: gRequest.BatchId}
	if gRequest.StateUpdatedAt != nil {
		stateUpdatedAt, err := ptypes.Timestamp(gRequest.StateUpdatedAt)
		if err != nil {
			return gResponse, err
		}
		request.UpdatedAt = stateUpdatedAt
	}
	svcDomain, err := server.updateHandler.GetCurrentServiceDomain(ctx, request)
	if err != nil {
		return gResponse, err
	}
	// No current active service domain
	if svcDomain == nil {
		return gResponse, nil
	}
	gSvcDomain := &gapi.SoftwareUpdateServiceDomain{}
	err = base.Convert(svcDomain, gSvcDomain)
	if err != nil {
		return gResponse, err
	}
	gResponse.SvcDomain = gSvcDomain
	return gResponse, nil
}

// Deletes a given batch
func (server *rpcServer) DeleteBatch(ctx context.Context, gRequest *gapi.DeleteBatchRequest) (*gapi.DeleteBatchResponse, error) {
	gResponse := &gapi.DeleteBatchResponse{}
	err := server.updateHandler.DeleteBatch(ctx, gRequest.BatchId)
	if err != nil {
		return nil, err
	}
	return gResponse, nil
}

func getEntitiesQueryParameter(gQueryParam *gapi.QueryParameter) *model.EntitiesQueryParam {
	if gQueryParam == nil {
		return nil
	}
	return &model.EntitiesQueryParam{
		PageQueryParam: model.PageQueryParam{
			PageIndex: int(gQueryParam.PageIndex),
			PageSize:  int(gQueryParam.PageSize),
		},
		OrderBy: gQueryParam.OrderBy,
		Filter:  gQueryParam.Filter,
	}
}

func getPageInfo(payload *model.EntityListResponsePayload) *gapi.PageInfo {
	if payload == nil {
		return nil
	}
	return &gapi.PageInfo{
		PageIndex:   int32(payload.PageIndex),
		PageSize:    int32(payload.PageSize),
		TotalCount:  int32(payload.TotalCount),
		OrderByKeys: payload.OrderByKeys,
	}
}

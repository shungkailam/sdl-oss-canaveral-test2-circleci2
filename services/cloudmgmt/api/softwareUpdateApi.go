package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	gapi "cloudservices/operator/generated/grpc"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/golang/glog"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/grpc"
)

func getQueryParameter(queryParam *model.EntitiesQueryParam) *gapi.QueryParameter {
	if queryParam == nil {
		return nil
	}
	return &gapi.QueryParameter{
		PageIndex: int32(queryParam.PageIndex),
		PageSize:  int32(queryParam.PageSize),
		Filter:    queryParam.Filter,
		OrderBy:   queryParam.OrderBy,
	}
}

func getEntityListResponsePayload(pageInfo *gapi.PageInfo) *model.EntityListResponsePayload {
	if pageInfo == nil {
		return nil
	}
	return &model.EntityListResponsePayload{
		PageIndex:   int(pageInfo.PageIndex),
		PageSize:    int(pageInfo.PageSize),
		TotalCount:  int(pageInfo.TotalCount),
		OrderByKeys: pageInfo.OrderByKeys,
		// TODO set orderBy
	}

}

func (dbAPI *dbObjectModelAPI) checkSoftwareUpdatePreconditions(ctx context.Context, svcDomainIDs []string, targetVer string) error {
	if len(svcDomainIDs) == 0 {
		return errcode.NewBadRequestExError("servicedomains", "No service domain is available for the operation")
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	targetVerObj, err := base.ValidateVersionEx(targetVer)
	if err != nil {
		return err
	}
	versions, err := dbAPI.GetServiceDomainVersions(ctx, svcDomainIDs)
	if err != nil {
		return err
	}
	connMap := GetEdgeConnections(authContext.TenantID, svcDomainIDs...)
	for _, svcDomainID := range svcDomainIDs {
		if connMap[svcDomainID] == false {
			errMsg := fmt.Sprintf("Service domain %s is disconnected", svcDomainID)
			return errcode.NewBadRequestExError("servicedomain", errMsg)
		}
		if ver, ok := versions[svcDomainID]; ok {
			verObj, err := base.ValidateVersionEx(ver)
			if err != nil {
				return err
			}
			if targetVerObj.Equal(verObj) {
				errMsg := fmt.Sprintf("Service domain %s is already in version %s", svcDomainID, ver)
				return errcode.NewBadRequestExError("version", errMsg)
			} else if targetVerObj.LessThan(verObj) {
				errMsg := fmt.Sprintf("Target version %s for service domain %s is lower than the current version %s", targetVer, svcDomainID, ver)
				return errcode.NewBadRequestExError("version", errMsg)
			}
			features, err := GetFeaturesForVersion(ver)
			if err != nil {
				return err
			}
			if features.DownloadAndUpgrade == false {
				errMsg := fmt.Sprintf("Service domain %s does not support download and upgrade feature", svcDomainID)
				return errcode.NewBadRequestExError("servicedomain", errMsg)
			}
		} else {
			// No version found
			errMsg := fmt.Sprintf("Unknown version for service domain %s", svcDomainID)
			return errcode.NewBadRequestExError("servicedomain", errMsg)
		}
	}
	return nil
}

// StartSoftwareDownload initiates software download on a batch of service domains
func (dbAPI *dbObjectModelAPI) StartSoftwareDownload(ctx context.Context, i interface{} /* *model.SoftwareDownloadCreate */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	p, ok := i.(*model.SoftwareDownloadCreate)
	if !ok {
		return resp, errcode.NewInternalError("StartSoftwareDownload: type error")
	}
	doc := *p
	err := dbAPI.checkSoftwareUpdatePreconditions(ctx, doc.SvcDomainIDs, doc.Release)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Preconditions failed. Error: %s"), err.Error())
		return resp, err
	}
	gRequest := &gapi.StartDownloadRequest{SvcDomainIds: doc.SvcDomainIDs, Release: doc.Release}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.StartDownload(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to start download. Error: %s"), err.Error())
			return err
		}
		if callback != nil {
			for _, svcDomainID := range doc.SvcDomainIDs {
				svcDomain := model.SoftwareUpdateServiceDomain{SvcDomainID: svcDomainID, BatchID: response.BatchId}
				svcDomain.Release = doc.Release
				svcDomain.State = model.SoftwareUpdateStateType(response.State)
				go callback(ctx, svcDomain)
			}
		}
		resp.ID = response.BatchId
		return nil
	}
	err = service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// StartSoftwareDownloadW initiates software download on a batch of service domains
func (dbAPI *dbObjectModelAPI) StartSoftwareDownloadW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.StartSoftwareDownload, &model.SoftwareDownloadCreate{}, w, r, callback)
}

// Deprecated: Batches are immutable now.
// UpdateSoftwareDownload updates an active software download batch.
// Only the affected service domains are notified of the update.
func (dbAPI *dbObjectModelAPI) UpdateSoftwareDownload(ctx context.Context, i interface{} /* *model.SoftwareDownloadUpdate */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	p, ok := i.(*model.SoftwareDownloadUpdate)
	if !ok {
		return resp, errcode.NewInternalError("UpdateSoftwareDownload: type error")
	}
	doc := *p
	gRequest := &gapi.UpdateDownloadRequest{BatchId: doc.BatchID, Command: string(doc.Command)}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.UpdateDownload(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update download batch %s with command %s. Error: %s"), doc.BatchID, doc.Command, err.Error())
			return err
		}
		if callback != nil {
			for i := range response.SvcDomainStates {
				svcDomainState := response.SvcDomainStates[i]
				svcDomain := model.SoftwareUpdateServiceDomain{SvcDomainID: svcDomainState.SvcDomainId, BatchID: response.BatchId}
				svcDomain.Release = response.Release
				svcDomain.State = model.SoftwareUpdateStateType(svcDomainState.State)
				go callback(ctx, svcDomain)
			}
		}
		resp.ID = response.BatchId
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// Deprecated: Batches are immutable now.
// UpdateSoftwareDownloadW updates an active software download batch.
// Only the affected service domains are notified of the update
func (dbAPI *dbObjectModelAPI) UpdateSoftwareDownloadW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, dbAPI.UpdateSoftwareDownload, &model.SoftwareDownloadUpdate{}, w, r, callback)
}

// UpdateSoftwareDownloadState updates the download state/progress of a service domain.
// It is called by the service domain
func (dbAPI *dbObjectModelAPI) UpdateSoftwareDownloadState(ctx context.Context, i interface{} /* *model.SoftwareUpdateState */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.SoftwareUpdateState)
	if !ok {
		return model.SoftwareUpdateState{}, errcode.NewInternalError("UpdateSoftwareDownloadState: type error")
	}
	doc := *p
	gRequest := &gapi.UpdateDownloadStateRequest{
		BatchId:     doc.BatchID,
		SvcDomainId: doc.SvcDomainID,
		State:       string(doc.State),
		Progress:    int32(doc.Progress),
		Eta:         int32(doc.ETA),
		Release:     doc.Release,
	}
	if doc.FailureReason != nil && *doc.FailureReason != "" {
		gRequest.FailureReason = *doc.FailureReason
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.UpdateDownloadState(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update download state for service domain %s. Error: %s"), doc.SvcDomainID, err.Error())
			return err
		}
		doc.State = model.SoftwareUpdateStateType(response.State)
		if callback != nil {
			go callback(ctx, &doc)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return doc, err
}

// UpdateSoftwareDownloadStateW updates the download state/progress of a service domain.
// It is called by the service domain
func (dbAPI *dbObjectModelAPI) UpdateSoftwareDownloadStateW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, dbAPI.UpdateSoftwareDownloadState, &model.SoftwareUpdateState{}, w, r, callback)
}

// CreateSoftwareDownloadCredentials creates the download credentials
func (dbAPI *dbObjectModelAPI) CreateSoftwareDownloadCredentials(ctx context.Context, i interface{} /* *model.SoftwareUpdateCredentials */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.SoftwareUpdateCredentialsCreatePayload{}
	p, ok := i.(*model.SoftwareUpdateCredentials)
	if !ok {
		return resp, errcode.NewInternalError("CreateSoftwareDownloadCredentials: type error")
	}
	doc := *p
	resp.SoftwareUpdateCredentials = doc
	gRequest := &gapi.CreateDownloadCredentialsRequest{BatchId: doc.BatchID, Release: doc.Release, AccessType: string(doc.AccessType)}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.CreateDownloadCredentials(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create download credentials for batch %s. Error: %s"), doc.BatchID, err.Error())
			return err
		}
		resp.Credentials = response.Credentials
		if callback != nil {
			go callback(ctx, resp)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// CreateSoftwareDownloadCredentialsW creates the download credentials
func (dbAPI *dbObjectModelAPI) CreateSoftwareDownloadCredentialsW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateSoftwareDownloadCredentials, &model.SoftwareUpdateCredentials{}, w, r, callback)
}

// SelectAllSoftwareDownloadBatches lists all the software download batches irrespective of the states.
// Filters can be applied to filter further
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareDownloadBatches(ctx context.Context, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateBatchListPayload, error) {
	resp := &model.SoftwareUpdateBatchListPayload{}
	gRequest := &gapi.ListDownloadBatchesRequest{QueryParameter: getQueryParameter(queryParam)}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListDownloadBatches(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get software download batches. Error: %s"), err.Error())
			return err
		}
		if response.PageInfo != nil {
			resp.EntityListResponsePayload = *getEntityListResponsePayload(response.PageInfo)
		}
		resp.BatchList = make([]model.SoftwareUpdateBatch, 0, len(response.Batches))
		for _, gBatch := range response.Batches {
			batch := model.SoftwareUpdateBatch{}
			err = base.Convert(gBatch, &batch)
			if err != nil {
				return err
			}
			resp.BatchList = append(resp.BatchList, batch)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// SelectAllSoftwareDownloadBatchesW lists all the software download batches irrespective of the states.
// Filters can be applied to filter further
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareDownloadBatchesW(ctx context.Context, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	batchListPayload, err := dbAPI.SelectAllSoftwareDownloadBatches(ctx, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(batchListPayload)
}

// GetSoftwareDownloadBatch gets a particular download batch by ID
func (dbAPI *dbObjectModelAPI) GetSoftwareDownloadBatch(ctx context.Context, batchID string) (*model.SoftwareUpdateBatch, error) {
	if batchID == "" {
		return nil, errcode.NewBadRequestError("batchId")
	}
	resp := model.SoftwareUpdateBatch{}
	gRequest := &gapi.ListDownloadBatchesRequest{BatchId: batchID}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListDownloadBatches(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get software download batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		if len(response.Batches) == 0 {
			return errcode.NewRecordNotFoundError("batchId")
		}
		err = base.Convert(response.Batches[0], &resp)
		if err != nil {
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return &resp, err
}

// GetSoftwareDownloadBatchW gets a particular download batch by ID
func (dbAPI *dbObjectModelAPI) GetSoftwareDownloadBatchW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error {
	batch, err := dbAPI.GetSoftwareDownloadBatch(ctx, batchID)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(batch)
}

// SelectAllSoftwareDownloadBatchServiceDomains lists all the service domain states in a batch
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareDownloadBatchServiceDomains(ctx context.Context, batchID string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	resp := &model.SoftwareUpdateServiceDomainListPayload{}
	gRequest := &gapi.ListDownloadBatchServiceDomainsRequest{QueryParameter: getQueryParameter(queryParam), BatchId: batchID}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListDownloadBatchServiceDomains(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get service domains in software download batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		if response.PageInfo != nil {
			resp.EntityListResponsePayload = *getEntityListResponsePayload(response.PageInfo)
		}
		resp.SvcDomainList = make([]model.SoftwareUpdateServiceDomain, 0, len(response.SvcDomains))
		for _, gSvcDomain := range response.SvcDomains {
			svcDomain := model.SoftwareUpdateServiceDomain{}
			err = base.Convert(gSvcDomain, &svcDomain)
			if err != nil {
				return err
			}
			resp.SvcDomainList = append(resp.SvcDomainList, svcDomain)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// SelectAllSoftwareDownloadBatchServiceDomainsW lists all the service domain states in a batch
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareDownloadBatchServiceDomainsW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	svcDomainListPayload, err := dbAPI.SelectAllSoftwareDownloadBatchServiceDomains(ctx, batchID, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(svcDomainListPayload)
}

// SelectAllSoftwareDownloadedServiceDomains lists all the service domains which have successfully downloaded the given release
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareDownloadedServiceDomains(ctx context.Context, release string, queryParam *model.EntitiesQueryParam) (*model.SoftwareDownloadedServiceDomainListPayload, error) {
	resp := &model.SoftwareDownloadedServiceDomainListPayload{}
	gRequest := &gapi.ListDownloadedServiceDomainsRequest{QueryParameter: getQueryParameter(queryParam), Release: release}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListDownloadedServiceDomains(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get service domains with downloaded release %s. Error: %s"), release, err.Error())
			return err
		}
		if response.PageInfo != nil {
			resp.EntityListResponsePayload = *getEntityListResponsePayload(response.PageInfo)
		}
		resp.SvcDomainList = response.SvcDomainIds
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// SelectAllSoftwareDownloadedServiceDomainsW lists all the service domains which have successfully downloaded the given release
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareDownloadedServiceDomainsW(ctx context.Context, release string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	svcDomainListPayload, err := dbAPI.SelectAllSoftwareDownloadedServiceDomains(ctx, release, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(svcDomainListPayload)
}

// SelectAllSoftwareUpdateServiceDomains lists all the service domains with the most recent batches
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpdateServiceDomains(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.SoftwareUpdateServiceDomainQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	resp := &model.SoftwareUpdateServiceDomainListPayload{}
	gRequest := &gapi.ListServiceDomainsRequest{QueryParameter: getQueryParameter(entitiesQueryParam)}
	if queryParam != nil {
		gRequest.SvcDomainId = queryParam.SvcDomainID
		gRequest.Type = queryParam.Type
		gRequest.IsLatestBatch = queryParam.IsLatestBatch
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListServiceDomains(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to list service domains with most recent batches. Error: %s"), err.Error())
			return err
		}
		if response.PageInfo != nil {
			resp.EntityListResponsePayload = *getEntityListResponsePayload(response.PageInfo)
		}
		resp.SvcDomainList = make([]model.SoftwareUpdateServiceDomain, 0, len(response.SvcDomains))
		for _, gSvcDomain := range response.SvcDomains {
			svcDomain := model.SoftwareUpdateServiceDomain{}
			err = base.Convert(gSvcDomain, &svcDomain)
			if err != nil {
				return err
			}
			resp.SvcDomainList = append(resp.SvcDomainList, svcDomain)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// SelectAllSoftwareUpdateServiceDomainsW lists all the service domain states in a batch
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpdateServiceDomainsW(ctx context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	// Default for IsLatestBatch is true for backward compatibility
	queryParam := &model.SoftwareUpdateServiceDomainQueryParam{IsLatestBatch: true}
	err := base.GetHTTPQueryParams(req, queryParam)
	if err != nil {
		return err
	}
	svcDomainListPayload, err := dbAPI.SelectAllSoftwareUpdateServiceDomains(ctx, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(svcDomainListPayload)
}

// StartSoftwareUpgrade initiates software upgrade on a batch of service domains
func (dbAPI *dbObjectModelAPI) StartSoftwareUpgrade(ctx context.Context, i interface{} /* *model.SoftwareUpgradeCreate */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	p, ok := i.(*model.SoftwareUpgradeCreate)
	if !ok {
		return resp, errcode.NewInternalError("StartSoftwareUpgrade: type error")
	}
	doc := *p
	err := dbAPI.checkSoftwareUpdatePreconditions(ctx, doc.SvcDomainIDs, doc.Release)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Preconditions failed. Error: %s"), err.Error())
		return resp, err
	}
	gRequest := &gapi.StartUpgradeRequest{SvcDomainIds: doc.SvcDomainIDs, Release: doc.Release}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.StartUpgrade(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to start upgrade. Error: %s"), err.Error())
			return err
		}
		if callback != nil {
			for _, svcDomainID := range doc.SvcDomainIDs {
				svcDomain := model.SoftwareUpdateServiceDomain{SvcDomainID: svcDomainID, BatchID: response.BatchId}
				svcDomain.Release = doc.Release
				svcDomain.State = model.SoftwareUpdateStateType(response.State)
				go callback(ctx, svcDomain)
			}
		}
		resp.ID = response.BatchId
		return nil
	}
	err = service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// StartSoftwareUpgradeW initiates software upgrade on a batch of service domains
func (dbAPI *dbObjectModelAPI) StartSoftwareUpgradeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.StartSoftwareUpgrade, &model.SoftwareUpgradeCreate{}, w, r, callback)
}

// Deprecated: Batches are immutable now.
// UpdateSoftwareUpgrade updates an active software upgrade.
// It supports only retry on a failed upgrade
func (dbAPI *dbObjectModelAPI) UpdateSoftwareUpgrade(ctx context.Context, i interface{} /* *model.SoftwareUpgradeUpdate */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	p, ok := i.(*model.SoftwareUpgradeUpdate)
	if !ok {
		return resp, errcode.NewInternalError("UpdateSoftwareUpgrade: type error")
	}
	doc := *p
	gRequest := &gapi.UpdateUpgradeRequest{BatchId: doc.BatchID, Command: string(doc.Command)}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.UpdateUpgrade(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update upgrade batch %s with command %s. Error: %s"), doc.BatchID, doc.Command, err.Error())
			return err
		}
		if callback != nil {
			for i := range response.SvcDomainStates {
				svcDomainState := response.SvcDomainStates[i]
				svcDomain := model.SoftwareUpdateServiceDomain{SvcDomainID: svcDomainState.SvcDomainId, BatchID: response.BatchId}
				svcDomain.Release = response.Release
				svcDomain.State = model.SoftwareUpdateStateType(svcDomainState.State)
				go callback(ctx, svcDomain)
			}
		}
		resp.ID = response.BatchId
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// Deprecated: Batches are immutable now.
// UpdateSoftwareUpgradeW updates an active software upgrade.
// It supports only retry on a failed upgrade
func (dbAPI *dbObjectModelAPI) UpdateSoftwareUpgradeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, dbAPI.UpdateSoftwareUpgrade, &model.SoftwareUpgradeUpdate{}, w, r, callback)
}

// UpdateSoftwareUpgradeState updates the upgrade state of a service domain.
// It is called by the service domain
func (dbAPI *dbObjectModelAPI) UpdateSoftwareUpgradeState(ctx context.Context, i interface{} /* *model.SoftwareUpdateState */, callback func(context.Context, interface{}) error) (interface{}, error) {
	p, ok := i.(*model.SoftwareUpdateState)
	if !ok {
		return model.SoftwareUpdateState{}, errcode.NewInternalError("UpdateSoftwareUpgradeState: type error")
	}
	doc := *p
	gRequest := &gapi.UpdateUpgradeStateRequest{BatchId: doc.BatchID, SvcDomainId: doc.SvcDomainID, State: string(doc.State), Progress: int32(doc.Progress), Eta: int32(doc.ETA)}
	if doc.FailureReason != nil && *doc.FailureReason != "" {
		gRequest.FailureReason = *doc.FailureReason
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.UpdateUpgradeState(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update upgrade state for service domain %s. Error: %s"), doc.SvcDomainID, err.Error())
			return err
		}
		doc.State = model.SoftwareUpdateStateType(response.State)
		if callback != nil {
			go callback(ctx, &doc)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return doc, err
}

// UpdateSoftwareUpgradeStateW updates the upgrade state of a service domain.
// It is called by the service domain
func (dbAPI *dbObjectModelAPI) UpdateSoftwareUpgradeStateW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.UpdateW(ctx, dbAPI.UpdateSoftwareUpgradeState, &model.SoftwareUpdateState{}, w, r, callback)
}

// SelectAllSoftwareUpgradeBatches lists all the batches irrespective of the batch state
// Filters can be applied further
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpgradeBatches(ctx context.Context, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateBatchListPayload, error) {
	resp := &model.SoftwareUpdateBatchListPayload{}
	gRequest := &gapi.ListUpgradeBatchesRequest{QueryParameter: getQueryParameter(queryParam)}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListUpgradeBatches(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get software upgrade batches. Error: %s"), err.Error())
			return err
		}
		if response.PageInfo != nil {
			resp.EntityListResponsePayload = *getEntityListResponsePayload(response.PageInfo)
		}
		resp.BatchList = make([]model.SoftwareUpdateBatch, 0, len(response.Batches))
		for _, gBatch := range response.Batches {
			batch := model.SoftwareUpdateBatch{}
			err = base.Convert(gBatch, &batch)
			if err != nil {
				return err
			}
			resp.BatchList = append(resp.BatchList, batch)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// SelectAllSoftwareUpgradeBatchesW lists all the batches irrespective of the batch state
// Filters can be applied further
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpgradeBatchesW(ctx context.Context, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	batchListPayload, err := dbAPI.SelectAllSoftwareUpgradeBatches(ctx, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(batchListPayload)
}

// GetSoftwareUpgradeBatch gets an upgrade batch by ID
func (dbAPI *dbObjectModelAPI) GetSoftwareUpgradeBatch(ctx context.Context, batchID string) (*model.SoftwareUpdateBatch, error) {
	if batchID == "" {
		return nil, errcode.NewBadRequestError("batchId")
	}
	resp := model.SoftwareUpdateBatch{}
	gRequest := &gapi.ListUpgradeBatchesRequest{BatchId: batchID}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListUpgradeBatches(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get software upgrade batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		if len(response.Batches) == 0 {
			return errcode.NewRecordNotFoundError("batchId")
		}
		err = base.Convert(response.Batches[0], &resp)
		if err != nil {
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return &resp, err
}

// GetSoftwareUpgradeBatchW gets an upgrade batch by ID
func (dbAPI *dbObjectModelAPI) GetSoftwareUpgradeBatchW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error {
	batch, err := dbAPI.GetSoftwareUpgradeBatch(ctx, batchID)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(batch)
}

// SelectAllSoftwareUpgradeBatchServiceDomains lists all the service domains in an upgrade
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpgradeBatchServiceDomains(ctx context.Context, batchID string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	resp := &model.SoftwareUpdateServiceDomainListPayload{}
	gRequest := &gapi.ListUpgradeBatchServiceDomainsRequest{QueryParameter: getQueryParameter(queryParam), BatchId: batchID}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.ListUpgradeBatchServiceDomains(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get service domains in software upgrade batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		if response.PageInfo != nil {
			resp.EntityListResponsePayload = *getEntityListResponsePayload(response.PageInfo)
		}
		resp.SvcDomainList = make([]model.SoftwareUpdateServiceDomain, 0, len(response.SvcDomains))
		for _, gSvcDomain := range response.SvcDomains {
			svcDomain := model.SoftwareUpdateServiceDomain{}
			err = base.Convert(gSvcDomain, &svcDomain)
			if err != nil {
				return err
			}
			resp.SvcDomainList = append(resp.SvcDomainList, svcDomain)
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return resp, err
}

// SelectAllSoftwareUpgradeBatchServiceDomainsW lists all the service domains in an upgrade
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpgradeBatchServiceDomainsW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error {
	queryParam := model.GetEntitiesQueryParam(req)
	svcDomainListPayload, err := dbAPI.SelectAllSoftwareUpgradeBatchServiceDomains(ctx, batchID, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(svcDomainListPayload)
}

func (dbAPI *dbObjectModelAPI) getEdgeInventoryDeltaSoftwareUpdates(
	ctx context.Context, payload *model.EdgeInventoryDeltaPayload,
	result *model.EdgeInventoryDeltaResponse,
) error {
	gRequest := &gapi.GetCurrentServiceDomainRequest{}
	softwareUpdates := payload.SoftwareUpdates
	if len(softwareUpdates) > 0 {
		// Only one is supported today
		gRequest.BatchId = softwareUpdates[0].ID
		protoTime, err := ptypes.TimestampProto(softwareUpdates[0].UpdatedAt)
		if err != nil {
			return err
		}
		gRequest.StateUpdatedAt = protoTime
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		response, err := client.GetCurrentServiceDomain(ctx, gRequest)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get current active service domain. Error: %s"), err.Error())
			return err
		}
		if response == nil || response.SvcDomain == nil {
			// Nothing found
			return nil
		}
		gSvcDomain := response.SvcDomain
		svcDomain := model.SoftwareUpdateServiceDomain{}
		err = base.Convert(gSvcDomain, &svcDomain)
		if err != nil {
			return err
		}
		if gRequest.BatchId == "" {
			// Latest is unknown to the caller
			result.Created.SoftwareUpdates = []model.SoftwareUpdateServiceDomain{svcDomain}
		} else if gRequest.BatchId != svcDomain.BatchID {
			// A new one is encountered
			result.Deleted.SoftwareUpdates = []string{gRequest.BatchId}
			result.Created.SoftwareUpdates = []model.SoftwareUpdateServiceDomain{svcDomain}
		} else if svcDomain.StateUpdatedAt.After(softwareUpdates[0].UpdatedAt) {
			// No change in batch ID but time has changed
			result.Updated.SoftwareUpdates = []model.SoftwareUpdateServiceDomain{svcDomain}
		}
		// No change
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in operator service call. Error: %s"), err.Error())
	}
	return nil
}

// SelectAllSoftwareUpdateReleasesW select n and n-1 releases and writes output into writer
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpdateReleasesW(context context.Context, w io.Writer, req *http.Request) error {
	// get query param from request (PageIndex, PageSize, etc)
	queryParam := model.GetEntitiesQueryParam(req)
	releases, err := dbAPI.SelectAllSoftwareUpdateReleases(context)
	if err != nil {
		return err
	}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &ListQueryInfo{TotalCount: len(releases)})
	r := model.SoftwareReleaseListResponse{
		Payload: &model.SoftwareReleaseListPayload{
			EntityListResponsePayload: entityListResponsePayload,
			ReleaseList:               releases,
		},
	}
	return json.NewEncoder(w).Encode(r)
}

// SelectAllSoftwareUpdateReleases returns n and n-1 release versions
func (dbAPI *dbObjectModelAPI) SelectAllSoftwareUpdateReleases(ctx context.Context) ([]model.SoftwareRelease, error) {

	softwareReleases := []model.SoftwareRelease{}
	reqID := base.GetRequestID(ctx)
	request := &gapi.ListReleasesRequest{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewReleaseServiceClient(conn)
		releases, err := client.ListReleases(ctx, request)
		if err != nil {
			glog.Errorf("Request %s: Error: %s", reqID, err.Error())
			return err
		}
		//Note: The operator List Releases method returns n and n-1 release from build v1.15.0
		for _, release := range releases.Releases {
			softwareReleases = append(softwareReleases, model.SoftwareRelease{
				Release:   release.Id,
				Changelog: release.Changelog,
			})
		}
		return nil
	}
	err := service.CallClient(ctx, service.OperatorService, handler)

	return softwareReleases, err
}

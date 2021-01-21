package softwareupdate

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/operator/releases"
	"context"
	"fmt"

	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
)

// TODO Add RBAC

func checkSoftwareDownloadPreconditions(ctx context.Context, doc *model.SoftwareDownloadCreate) error {
	if len(doc.SvcDomainIDs) == 0 {
		return errcode.NewBadRequestError("servicedomains")
	}
	err := base.ValidateVersion(doc.Release)
	if err != nil {
		return err
	}
	releaseList, err := releases.GetAllReleases()
	if err != nil {
		errMsg := fmt.Sprintf("Failed to list releases: %s", err.Error())
		return errcode.NewInternalError(errMsg)
	}
	for _, release := range releaseList {
		if release.ID == doc.Release {
			return nil
		}
	}
	return errcode.NewBadRequestExError("release", fmt.Sprintf("Unknown release %s", doc.Release))
}

// StartDownload starts download of the release on a set of service domains
func (handler *Handler) StartDownload(ctx context.Context, doc *model.SoftwareDownloadCreate) (*model.SoftwareUpdateBatch, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	err = checkSoftwareDownloadPreconditions(ctx, doc)
	if err != nil {
		return nil, err
	}
	// Add purge for old batches
	now := base.RoundedNow()
	batchID := base.GetUUID()
	err = handler.dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		// Make sure no service domain is in non-terminal states
		err := handler.CanStartSoftwareUpdate(ctx, tx, doc.SvcDomainIDs)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in service domains validation. Error: %s"), err.Error())
			return err
		}
		batchDBO := &BatchDBO{
			CreatedAt:      now,
			UpdatedAt:      now,
			StateUpdatedAt: now,
			TenantID:       authCtx.TenantID,
			ID:             batchID,
			Release:        doc.Release,
			Type:           string(model.DownloadBatchType),
		}
		// Delete the previously downloaded same release for the sevice domain if any
		err = handler.DeleteDownloadedServiceDomainRelease(ctx, tx, doc.SvcDomainIDs, doc.Release)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting previously downloaded same release. Error: %s"), err.Error())
			return err
		}
		// Insert the new batch
		err = handler.InsertBatch(ctx, tx, batchDBO, doc.SvcDomainIDs)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in inserting batch record %+v. Error: %s"), batchDBO, err.Error())
			return err
		}
		for _, svcDomainID := range doc.SvcDomainIDs {
			svcDomainDBO := &ServiceDomainDBO{
				TenantID:       authCtx.TenantID,
				BatchID:        batchID,
				SvcDomainID:    svcDomainID,
				State:          model.DownloadState,
				CreatedAt:      now,
				UpdatedAt:      now,
				StateUpdatedAt: now,
			}
			_, err = tx.NamedExec(ctx, insertServiceDomainQuery, svcDomainDBO)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in inserting service domain record %+v. Error: %s"), svcDomainDBO, err.Error())
				return err
			}
		}
		return nil
	})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in starting the download on service domains %+v. Error: %s"), doc.SvcDomainIDs, err.Error())
		return nil, err
	}
	response := &model.SoftwareUpdateBatch{
		SoftwareUpdateCommon: model.SoftwareUpdateCommon{
			State:     model.DownloadState,
			Release:   doc.Release,
			CreatedAt: now,
			UpdatedAt: now,
		},
		ID:   batchID,
		Type: model.DownloadBatchType,
		Stats: map[model.SoftwareUpdateStateType]int{
			// All in download state
			model.DownloadState: len(doc.SvcDomainIDs),
		},
	}
	return response, nil
}

// UpdateDownload updates and existing download
func (handler *Handler) UpdateDownload(ctx context.Context, doc *model.SoftwareDownloadUpdate) ([]*model.SoftwareUpdateServiceDomain, error) {
	if doc.Command == model.DownloadCommand {
		return handler.RetryDownload(ctx, doc.BatchID)
	}
	if doc.Command == model.DownloadCancelCommand {
		return handler.CancelDownload(ctx, doc.BatchID)
	}
	return []*model.SoftwareUpdateServiceDomain{}, errcode.NewBadRequestError("command")
}

// RetryDownload retries download on the eligible service domains in the batch
func (handler *Handler) RetryDownload(ctx context.Context, batchID string) ([]*model.SoftwareUpdateServiceDomain, error) {
	response := []*model.SoftwareUpdateServiceDomain{}
	if batchID == "" {
		return response, errcode.NewBadRequestError("batchId")
	}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return response, err
	}
	// TODO make sure the version exists
	err = handler.dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err := handler.CanRetrySoftwareUpdate(ctx, tx, batchID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in batch validation. Error: %s"), err.Error())
			return err
		}
		batchDetails, err := handler.ReadBatchDetails(ctx, tx, batchID, model.DownloadBatchType, "", "", retryableDownloadStates)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting batch details for batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		svcDomainIDs := funk.Map(batchDetails.SvcDomains, func(svcDomain *ServiceDomainDBO) string {
			return svcDomain.SvcDomainID
		}).([]string)
		// Delete the previously downloaded same release for the sevice domain if any
		err = handler.DeleteDownloadedServiceDomainRelease(ctx, tx, svcDomainIDs, batchDetails.Release)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting previously downloaded same release. Error: %s"), err.Error())
			return err
		}
		now := base.RoundedNow()
		// Update the service domains eligible for retry
		for i := range batchDetails.SvcDomains {
			svcDomain := batchDetails.SvcDomains[i]
			nextState, stateChanged, err := handler.NextState(ctx, svcDomain.State, model.DownloadState)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in state transition. Error: %s"), err.Error())
				return err
			}
			if stateChanged {
				// Reset the stats
				svcDomainDBO := &ServiceDomainDBO{
					TenantID:       authCtx.TenantID,
					BatchID:        batchID,
					SvcDomainID:    svcDomain.SvcDomainID,
					State:          nextState,
					CreatedAt:      now,
					UpdatedAt:      now,
					StateUpdatedAt: now,
				}
				_, err = tx.NamedExec(ctx, updateServiceDomainQuery, svcDomainDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error in updating service domain records %+v. Error: %s"), svcDomainDBO, err.Error())
					return err
				}
				svcDomainResp := &model.SoftwareUpdateServiceDomain{}
				err = base.Convert(svcDomainDBO, svcDomainResp)
				if err != nil {
					return err
				}
				svcDomainResp.Release = batchDetails.Release
				response = append(response, svcDomainResp)
			}
		}
		return nil
	})
	return response, err
}

// CancelDownload cancels an existing download
func (handler *Handler) CancelDownload(ctx context.Context, batchID string) ([]*model.SoftwareUpdateServiceDomain, error) {
	response := []*model.SoftwareUpdateServiceDomain{}
	if batchID == "" {
		return response, errcode.NewBadRequestError("batchID")
	}
	err := handler.dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		batchDetails, err := handler.ReadBatchDetails(ctx, tx, batchID, model.DownloadBatchType, "", "", cancellableDownloadStates)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting batch details for batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		now := base.RoundedNow()
		// Update the service domains eligible for retry
		for i := range batchDetails.SvcDomains {
			svcDomain := batchDetails.SvcDomains[i]
			nextState, stateChanged, err := handler.NextState(ctx, svcDomain.State, model.DownloadCancelState)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in state transition. Error: %s"), err.Error())
				return err
			}
			if stateChanged {
				// Do not reset the stats
				svcDomain.State = nextState
				svcDomain.StateUpdatedAt = now
				svcDomain.UpdatedAt = now
				_, err = tx.NamedExec(ctx, updateServiceDomainQuery, svcDomain)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error in updating service domain records %+v. Error: %s"), batchDetails.SvcDomains, err.Error())
					return err
				}
				// Add only the changed ones
				svcDomainResp := &model.SoftwareUpdateServiceDomain{}
				err = base.Convert(svcDomain, svcDomainResp)
				if err != nil {
					return err
				}
				svcDomainResp.Release = batchDetails.Release
				response = append(response, svcDomainResp)
			}
		}
		return nil
	})
	return response, err
}

// UpdateDownloadState is invoked by the edge/service domain to update the download status
func (handler *Handler) UpdateDownloadState(ctx context.Context, doc *model.SoftwareUpdateState) (*model.SoftwareUpdateState, error) {
	if doc.BatchID == "" {
		return nil, errcode.NewBadRequestError("batchID")
	}
	if doc.SvcDomainID == "" {
		return nil, errcode.NewBadRequestError("servicedomain")
	}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	err = handler.dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		batchDetails, err := handler.ReadBatchDetails(ctx, tx, doc.BatchID, model.DownloadBatchType, doc.SvcDomainID, doc.Release, []model.SoftwareUpdateStateType{})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting batch details. Error: %s"), err.Error())
			return err
		}
		svcDomain := batchDetails.SvcDomains[0]
		// Fill or correct
		doc.Release = batchDetails.Release
		nextState, stateChanged, err := handler.NextState(ctx, svcDomain.State, doc.State)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in state transition. Error: %s"), err.Error())
			return err
		}
		now := base.RoundedNow()
		doc.State = nextState
		doc.UpdatedAt = now
		if stateChanged {
			doc.StateUpdatedAt = now
			if nextState == model.DownloadedState {
				downloadedSvcDomainDBO := &DownloadedServiceDomainDBO{TenantID: authCtx.TenantID, BatchID: doc.BatchID, SvcDomainID: doc.SvcDomainID, Release: doc.Release}
				_, err = tx.NamedExec(ctx, insertDownloadedServiceDomainQuery, downloadedSvcDomainDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error in saving downloaded service domain %s with release %s. Error: %s"), doc.SvcDomainID, doc.Release, err.Error())
					return err
				}
			}
		} else {
			doc.StateUpdatedAt = svcDomain.StateUpdatedAt
		}
		svcDomainDBO := &ServiceDomainDBO{}
		err = base.Convert(doc, svcDomainDBO)
		if err != nil {
			return err
		}
		svcDomainDBO.TenantID = authCtx.TenantID
		_, err = tx.NamedExec(ctx, updateServiceDomainQuery, svcDomainDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating service domain records %+v. Error: %s"), svcDomainDBO, err.Error())
			return err
		}
		return nil
	})
	return doc, err
}

// CreateDownloadCredentials returns the credentials required to download files
func (handler *Handler) CreateDownloadCredentials(ctx context.Context, credentials *model.SoftwareUpdateCredentials) (*model.SoftwareUpdateCredentialsCreatePayload, error) {
	var err error
	// TODO validate the batch and edge request
	response := &model.SoftwareUpdateCredentialsCreatePayload{SoftwareUpdateCredentials: *credentials, Credentials: map[string]string{}}
	if credentials.AccessType == model.DockerCredentialsAccessType {
		response.Credentials, err = handler.GetDockerLoginToken(ctx, credentials)
		if err != nil {
			return nil, err
		}
		return response, err
	}
	if credentials.AccessType == model.AWSCredentialsAccessType ||
		credentials.AccessType == model.AWSECRCredentialsAccessType {
		response.Credentials, err = handler.GetAWSFederatedToken(ctx, credentials)
		if err != nil {
			return nil, err
		}
		return response, err
	}
	return nil, errcode.NewBadRequestError("accessType")
}

// ListDownloadBatches lists all the download batches
func (handler *Handler) ListDownloadBatches(ctx context.Context, batchID /* optional */ string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateBatchListPayload, error) {
	return handler.ListBatches(ctx, batchID, model.DownloadBatchType, queryParam)
}

// ListDownloadBatchServiceDomains list all the service domains in a batch
func (handler *Handler) ListDownloadBatchServiceDomains(ctx context.Context, batchID, svcDomainID /* optional */ string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	return handler.ListBatchServiceDomains(ctx, batchID, svcDomainID, queryParam)
}

// ListDownloadedServiceDomains returns all the service domains with the release downloaded
func (handler *Handler) ListDownloadedServiceDomains(ctx context.Context, release string, queryParam *model.EntitiesQueryParam) (*model.SoftwareDownloadedServiceDomainListPayload, error) {
	err := base.ValidateVersion(release)
	if err != nil {
		return nil, err
	}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	svcDomainDBOs := []DownloadedServiceDomainDBO{}
	param := DownloadedServiceDomainDBO{TenantID: authCtx.TenantID, Release: release}
	query, _, err := orderByHelper.BuildPagedQuery(entityTypeSoftwareUpdateDownloadedServiceDomain, selectDownloadedServiceDomainsTemplateQuery, queryParam, defaultOrderBy)
	err = handler.dbAPI.QueryIn(ctx, &svcDomainDBOs, query, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching service domains with downloaded release %s. Error: %s"), release, err.Error())
		return nil, err
	}
	totalCount := 0
	svcDomainIDs := make([]string, 0, len(svcDomainDBOs))
	for _, svcDomainDBO := range svcDomainDBOs {
		svcDomainIDs = append(svcDomainIDs, svcDomainDBO.SvcDomainID)
	}
	svcDomainListPayload := &model.SoftwareDownloadedServiceDomainListPayload{
		EntityListResponsePayload: makeEntityListResponsePayload(entityTypeSoftwareUpdateServiceDomain, queryParam, totalCount),
		SvcDomainList:             svcDomainIDs,
	}
	return svcDomainListPayload, nil
}

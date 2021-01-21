package softwareupdate

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"

	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
)

// StartUpgrade triggers software upgrade on the service domains with the downloaded release
func (handler *Handler) StartUpgrade(ctx context.Context, doc *model.SoftwareUpgradeCreate) (*model.SoftwareUpdateBatch, error) {
	if len(doc.SvcDomainIDs) == 0 {
		return nil, errcode.NewBadRequestError("servicedomains")
	}
	err := base.ValidateVersion(doc.Release)
	if err != nil {
		return nil, err
	}
	authCtx, err := base.GetAuthContext(ctx)
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
		// Validate if the service domains has the release downloaded
		err = handler.VerifyDownloadedServiceDomains(ctx, tx, doc.SvcDomainIDs, doc.Release)
		if err != nil {
			return err
		}
		batchDBO := &BatchDBO{
			CreatedAt:      now,
			UpdatedAt:      now,
			StateUpdatedAt: now,
			TenantID:       authCtx.TenantID,
			ID:             batchID,
			Release:        doc.Release,
			Type:           string(model.UpgradeBatchType),
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
				State:          model.UpgradeState,
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
		glog.Errorf(base.PrefixRequestID(ctx, "Error in starting upgrade on service domains %+v. Error: %s"), doc.SvcDomainIDs, err.Error())
		return nil, err
	}
	response := &model.SoftwareUpdateBatch{
		SoftwareUpdateCommon: model.SoftwareUpdateCommon{
			State:     model.UpgradeState,
			Release:   doc.Release,
			CreatedAt: now,
			UpdatedAt: now,
		},
		ID:   batchID,
		Type: model.UpgradeBatchType,
		Stats: map[model.SoftwareUpdateStateType]int{
			// All in upgrade state
			model.UpgradeState: len(doc.SvcDomainIDs),
		},
	}
	return response, nil
}

// UpdateUpgrade updates the upgrade status
func (handler *Handler) UpdateUpgrade(ctx context.Context, doc *model.SoftwareUpgradeUpdate) ([]*model.SoftwareUpdateServiceDomain, error) {
	if doc.Command == model.UpgradeCommand {
		return handler.RetryUpgrade(ctx, doc.BatchID)
	}
	return []*model.SoftwareUpdateServiceDomain{}, errcode.NewBadRequestError("command")
}

// RetryUpgrade starts upgrade all over again
func (handler *Handler) RetryUpgrade(ctx context.Context, batchID string) ([]*model.SoftwareUpdateServiceDomain, error) {
	response := []*model.SoftwareUpdateServiceDomain{}
	if batchID == "" {
		return response, errcode.NewBadRequestError("batchId")
	}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return response, err
	}
	err = handler.dbAPI.DoInTxn(func(tx *base.WrappedTx) error {
		err := handler.CanRetrySoftwareUpdate(ctx, tx, batchID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in batch validation. Error: %s"), err.Error())
			return err
		}
		batchDetails, err := handler.ReadBatchDetails(ctx, tx, batchID, model.UpgradeBatchType, "", "", retryableUpgradeStates)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting batch details for batch %s. Error: %s"), batchID, err.Error())
			return err
		}
		svcDomainIDs := funk.Map(batchDetails.SvcDomains, func(svcDomain *ServiceDomainDBO) string {
			return svcDomain.SvcDomainID
		}).([]string)
		// Validate if the service domains still have the release
		err = handler.VerifyDownloadedServiceDomains(ctx, tx, svcDomainIDs, batchDetails.Release)
		if err != nil {
			return err
		}
		now := base.RoundedNow()
		// Update the service domains eligible for retry
		for i := range batchDetails.SvcDomains {
			svcDomain := batchDetails.SvcDomains[i]
			nextState, stateChanged, err := handler.NextState(ctx, svcDomain.State, model.UpgradeState)
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

// UpdateUpgradeState updates the state of the upgrade
func (handler *Handler) UpdateUpgradeState(ctx context.Context, doc *model.SoftwareUpdateState) (*model.SoftwareUpdateState, error) {
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
		batchDetails, err := handler.ReadBatchDetails(ctx, tx, doc.BatchID, model.UpgradeBatchType, doc.SvcDomainID, doc.Release, []model.SoftwareUpdateStateType{})
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

// ListUpgradeBatches returns all the upgrade batches
func (handler *Handler) ListUpgradeBatches(ctx context.Context, batchID /* optional */ string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateBatchListPayload, error) {
	return handler.ListBatches(ctx, batchID, model.UpgradeBatchType, queryParam)
}

// ListUpgradeBatchServiceDomains returns all the service domains in a batch
func (handler *Handler) ListUpgradeBatchServiceDomains(ctx context.Context, batchID, svcDomainID /* optional */ string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	return handler.ListBatchServiceDomains(ctx, batchID, svcDomainID, queryParam)
}

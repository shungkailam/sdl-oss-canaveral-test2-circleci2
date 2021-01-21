package softwareupdate

import (
	"bytes"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/operator/common"
	"cloudservices/operator/config"
	"context"
	"encoding/base64"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/golang/glog"
	version "github.com/hashicorp/go-version"
	funk "github.com/thoas/go-funk"
)

const (
	defaultOrderBy                                  = "order by id"
	entityTypeSoftwareUpdateBatch                   = "softwareupdatebatch"
	entityTypeSoftwareUpdateServiceDomain           = "softwareupdateservicedomain"
	entityTypeSoftwareUpdateDownloadedServiceDomain = "softwareupdatedownloadedservicedomain"

	// <s3 bucket>/<s3 prefix>/v<version subpath like 1/15/0>/packages/sherlock_edge_deployer
	s3ReleaseRootDirectoryFormat = "s3://%s/%sv%s/packages/sherlock_edge_deployer"

	softwareUpdateBatchModelAlias         = "bm"
	softwareUpdateServiceDomainModelAlias = "sdm"

	selectBatchesQuery         = `select *, count(*) over() as total_count from software_update_batch_model where tenant_id = :tenant_id and (:id = '' or id = :id) and (:type = '' or type = :type) and (:release = '' or release = :release)`
	selectBatchesTemplateQuery = `select *, count(*) over() as total_count from software_update_batch_model where tenant_id = :tenant_id and (:id = '' or id = :id) and (:type = '' or type = :type) %s`
	selectBatchStatsQuery      = `select batch_id, state, count(*) as count, round(avg(progress)) as progress, max(eta) as eta from software_update_service_domain_model where tenant_id = :tenant_id and batch_id in (:batch_ids) group by batch_id, state`
	// Conditional insert
	insertBatchQuery = `insert into software_update_batch_model(id, tenant_id, type, release, created_at, updated_at) select :id, :tenant_id, :type, :release, :created_at, :updated_at where not exists (select * from software_update_service_domain_model where tenant_id = :tenant_id and svc_domain_id in (:svc_domain_ids) and state in (:states))`
	updateBatchQuery = `update software_update_batch_model set state = :state, updated_at = :updated_at where tenant_id = :tenant_id and batch_id = :batch_id`

	canUpdateBatchQuery = `select sdm1.* from software_update_service_domain_model sdm1, software_update_service_domain_model sdm2 where sdm1.tenant_id = sdm2.tenant_id and sdm2.tenant_id = :tenant_id and sdm2.batch_id = :batch_id and sdm1.svc_domain_id = sdm2.svc_domain_id and sdm1.state in (:states)`
	canCreateBatchQuery = `select svc_domain_id, state from software_update_service_domain_model where tenant_id = :tenant_id and svc_domain_id in (:svc_domain_ids) and state in (:states)`

	// Delete a specific batch
	deleteBatchQuery = `delete from software_update_batch_model where tenant_id = :tenant_id and id = :id`

	// Select for update to lock the rows
	selectServiceDomainsForUpdateQuery    = `select * from software_update_service_domain_model where tenant_id = :tenant_id and batch_id = :batch_id and (:svc_domain_id = '' or svc_domain_id = :svc_domain_id) and state in (:states) for update`
	selectAllServiceDomainsForUpdateQuery = `select * from software_update_service_domain_model where tenant_id = :tenant_id and batch_id = :batch_id and (:svc_domain_id = '' or svc_domain_id = :svc_domain_id) for update`
	selectServiceDomainsTemplateQuery     = `select sdm.*, bm.release as release, count(sdm.*) over() as total_count from software_update_batch_model bm, software_update_service_domain_model sdm where sdm.tenant_id = bm.tenant_id and sdm.batch_id = bm.id and sdm.tenant_id = :tenant_id and sdm.batch_id = :batch_id and (:svc_domain_id = '' or sdm.svc_domain_id = :svc_domain_id) %s`

	// Select service domains with most recent batch with the ability to filter on batch type and service domain type if specified
	selectAllServiceDomainBatchesTemplateQuery       = `select sdm.*, bm.release as release, count(sdm.*) over() as total_count from software_update_service_domain_model sdm, software_update_batch_model bm where bm.id = sdm.batch_id and bm.tenant_id = :tenant_id and (:type = '' or bm.type = :type) and (:svc_domain_id = '' or sdm.svc_domain_id = :svc_domain_id) %s`
	selectAllLatestServiceDomainBatchesTemplateQuery = `select sdm.*, bm.release, count(sdm.*) over() as total_count from (select distinct on(svc_domain_id) * from software_update_service_domain_model where tenant_id = :tenant_id order by svc_domain_id, created_at desc) sdm, software_update_batch_model bm where sdm.batch_id = bm.id and (:type = '' or bm.type = :type) and (:svc_domain_id = '' or sdm.svc_domain_id = :svc_domain_id) %s`

	// Select the latest batch for each service domain
	selectLatestServiceDomainBatchIDsQuery = `select distinct on(svc_domain_id) svc_domain_id, batch_id from software_update_service_domain_model where tenant_id = :tenant_id order by svc_domain_id, created_at desc`

	insertServiceDomainQuery = `insert into software_update_service_domain_model(tenant_id, batch_id, svc_domain_id, state, progress, eta, failure_reason, created_at, updated_at, state_updated_at) values (:tenant_id, :batch_id, :svc_domain_id, :state, :progress, :eta, :failure_reason, :created_at, :updated_at, :state_updated_at)`
	updateServiceDomainQuery = `update software_update_service_domain_model set state = :state, progress = :progress, eta = :eta, failure_reason = :failure_reason, updated_at = :updated_at, state_updated_at = :state_updated_at where tenant_id = :tenant_id and batch_id = :batch_id and svc_domain_id = :svc_domain_id`

	// Select all service domains with state_updated_time > input time or the input service domain itself
	getInventoryDeltaQuery = `select sdm.*, bm.release as release from software_update_batch_model bm, software_update_service_domain_model sdm where sdm.tenant_id = sdm.tenant_id and sdm.batch_id = bm.id and sdm.tenant_id = :tenant_id and sdm.svc_domain_id = :svc_domain_id and (sdm.batch_id = :batch_id or sdm.state_updated_at > :state_updated_at) and sdm.state in (:states) order by sdm.state_updated_at desc`

	selectDownloadedServiceDomainsQuery         = `select *, count(*) over() as total_count from software_update_downloaded_model where tenant_id = :tenant_id and (:release = '' or release = :release) and svc_domain_id in (:svc_domain_ids)`
	selectDownloadedServiceDomainsTemplateQuery = `select *, count(*) over() as total_count from software_update_downloaded_model where tenant_id = :tenant_id and (:release = '' or release = :release) and (:svc_domain_id = '' or svc_domain_id = :svc_domain_id) %s`
	insertDownloadedServiceDomainQuery          = `insert into software_update_downloaded_model(tenant_id, batch_id, svc_domain_id, release, created_at) values (:tenant_id, :batch_id, :svc_domain_id, :release, :created_at) `
	deleteDownloadedServiceDomainsTemplateQuery = `delete from software_update_downloaded_model where tenant_id = :tenant_id and svc_domain_id in ('%s')`
)

var (
	// AWS S3 Bucket policy with limited access
	releaseAWSBucketAccessPolicy = base.MustMarshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": []string{
					"s3:ListBucket",
					"s3:ListObjects",
					"s3:GetObject",
				},
				"Resource": []string{
					"arn:aws:s3:::{{.BUCKET_NAME}}",
					"arn:aws:s3:::{{.RELEASE_PATH}}/*",
				},
			},
		},
	})

	// Minio S3 Bucket policy (ListObjects is not supported)
	releaseMinioBucketAccessPolicy = base.MustMarshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Action": []string{
					"s3:ListBucket",
					"s3:GetObject",
				},
				"Resource": []string{
					"arn:aws:s3:::{{.BUCKET_NAME}}",
					"arn:aws:s3:::{{.RELEASE_PATH}}/*",
				},
			},
		},
	})

	// AWS ECR policy to pull images
	awsECRAccessPolicy = `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Sid": "AllowPull",
				"Effect": "Allow",
				"Action": [
					"ecr:GetDownloadUrlForLayer",
					"ecr:GetAuthorizationToken",
					"ecr:BatchGetImage",
					"ecr:BatchCheckLayerAvailability"
				],
				"Resource": "*"
			}
		]
	}`
	nonTerminalStates = []model.SoftwareUpdateStateType{
		model.DownloadState,
		model.DownloadingState,
		model.DownloadCancelState,
		model.UpgradeState,
		model.UpgradingState,
	}

	// Any update - download/upgrade is forbidden f
	nonTerminalUpdateForbiddenStates = []model.SoftwareUpdateStateType{
		model.DownloadState,
		model.DownloadingState,
		model.UpgradeState,
		model.UpgradingState,
	}

	nonTerminalDownloadStates = []model.SoftwareUpdateStateType{
		model.DownloadState,
		model.DownloadingState,
		model.DownloadCancelState,
	}

	retryableDownloadStates = []model.SoftwareUpdateStateType{
		// cancel is included in terminal
		// as service domain may not respond
		model.DownloadCancelState,
		model.DownloadCancelledState,
		model.DownloadFailedState,
	}

	nonTerminalUpgradeStates = []model.SoftwareUpdateStateType{
		model.UpgradeState,
		model.UpgradingState,
	}

	cancellableDownloadStates = []model.SoftwareUpdateStateType{
		model.DownloadState,
		model.DownloadingState,
	}

	retryableUpgradeStates = []model.SoftwareUpdateStateType{
		model.UpgradeFailedState,
	}

	orderByHelper = base.NewOrderByHelper()
)

func init() {
	// Set up filter fields
	orderByHelper.Setup(entityTypeSoftwareUpdateBatch, []string{"id", "release", "created_at", "updated_at"})
	orderByHelper.Setup(entityTypeSoftwareUpdateServiceDomain, []string{"batch_id", "svc_domain_id", "state", "created_at", "updated_at", "state_updated_at"})
	orderByHelper.Setup(entityTypeSoftwareUpdateDownloadedServiceDomain, []string{"batch_id", "svc_domain_id", "release", "created_at"})
}

// Handler has a set of functions to handle software updates - download and upgrade
type Handler struct {
	awsSession       *session.Session
	dbAPI            *base.DBObjectModelAPI
	stateTransitions map[model.SoftwareUpdateStateType]map[model.SoftwareUpdateStateType]bool
}

// BatchDBO is the DB model for batches
type BatchDBO struct {
	ID             string    `json:"id" db:"id"`
	TenantID       string    `json:"tenantId" db:"tenant_id"`
	Release        string    `json:"release" db:"release"`
	Type           string    `json:"type" db:"type"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
	StateUpdatedAt time.Time `json:"stateUpdatedAt" db:"state_updated_at"`
	TotalCount     int       `json:"totalCount" db:"total_count"`
}

// InsertBatchDBO is the model to insert a batch conditionally such that
// there are no records for a service in non-terminal states.
type InsertBatchDBO struct {
	BatchDBO
	// CSV of service domain IDs
	SvcDomainIDs string `json:"svcDomainIDs" db:"svc_domain_ids"`
	// CSV of states
	States string `json:"states" db:"states"`
}

// ServiceDomainDBO is the DB model for service domains in software update
type ServiceDomainDBO struct {
	ID          int64                         `json:"id" db:"id"`
	TenantID    string                        `json:"tenantId" db:"tenant_id"`
	BatchID     string                        `json:"batchId" db:"batch_id"`
	SvcDomainID string                        `json:"svcDomainId" db:"svc_domain_id"`
	State       model.SoftwareUpdateStateType `json:"state" db:"state"`
	Release     string                        `json:"release" db:"release"`
	// Progress in percentage
	Progress int `json:"progress" db:"progress"`
	// ETA in mins
	ETA            int       `json:"eta" db:"eta"`
	FailureReason  *string   `json:"failureReason,omitempty" db:"failure_reason"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time `json:"updatedAt" db:"updated_at"`
	StateUpdatedAt time.Time `json:"stateUpdatedAt" db:"state_updated_at"`
	TotalCount     int       `json:"totalCount" db:"total_count"`
}

// BatchStatsDBO is the DB model for batch stats
type BatchStatsDBO struct {
	BatchID  string                        `json:"batchId" db:"batch_id"`
	State    model.SoftwareUpdateStateType `json:"state" db:"state"`
	Count    int                           `json:"count" db:"count"`
	Progress int                           `json:"progress" db:"progress"`
	ETA      int                           `json:"eta" db:"eta"`
}

// DownloadedServiceDomainDBO is the DB model for service domains with the release downloaded
type DownloadedServiceDomainDBO struct {
	ID          int64     `json:"id" db:"id"`
	TenantID    string    `json:"tenantId" db:"tenant_id"`
	BatchID     string    `json:"batchId" db:"batch_id"`
	SvcDomainID string    `json:"svcDomainId" db:"svc_domain_id"`
	Release     string    `json:"release" db:"release"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	TotalCount  int       `json:"totalCount" db:"total_count"`
}

// BatchDetails is a placeholder for the batch along with its service domains
type BatchDetails struct {
	BatchDBO
	SvcDomains []*ServiceDomainDBO `json:"svcDomains"`
}

// BatchQueryParam is the query param for batch DBO
type BatchQueryParam struct {
	ID       string                        `json:"id" db:"id"`
	TenantID string                        `json:"tenantId" db:"tenant_id"`
	Release  string                        `json:"release" db:"release"`
	Type     model.SoftwareUpdateBatchType `json:"type" db:"type"`
}

// BatchStatsQueryParam is the query param for stats for a batch
type BatchStatsQueryParam struct {
	TenantID string                          `json:"tenantId" db:"tenant_id"`
	BatchIDs []string                        `json:"batchIds" db:"batch_ids"`
	States   []model.SoftwareUpdateStateType `json:"states" db:"states"`
}

// ServiceDomainQueryParam is the query param for service domain
type ServiceDomainQueryParam struct {
	TenantID       string                          `json:"tenantId" db:"tenant_id"`
	SvcDomainID    string                          `json:"svcDomainId" db:"svc_domain_id"`
	States         []model.SoftwareUpdateStateType `json:"states" db:"states"`
	BatchID        string                          `json:"batchId" db:"batch_id"`
	BatchType      model.SoftwareUpdateBatchType   `json:"type" db:"type"`
	StateUpdatedAt time.Time                       `json:"stateUpdatedAt" db:"state_updated_at"`
}

// ServiceDomainBatchesQueryParam is the query param for service domain batches
type ServiceDomainBatchesQueryParam struct {
	TenantID    string                        `json:"tenantId" db:"tenant_id"`
	SvcDomainID string                        `json:"svcDomainId" db:"svc_domain_id"`
	BatchType   model.SoftwareUpdateBatchType `json:"type" db:"type"`
	BatchIDs    []string                      `json:"batchIds" db:"batch_ids"`
}

// ServiceDomainsQueryParam is the query param for service domains
type ServiceDomainsQueryParam struct {
	TenantID     string                          `json:"tenantId" db:"tenant_id"`
	SvcDomainIDs []string                        `json:"svcDomainIds" db:"svc_domain_ids"`
	States       []model.SoftwareUpdateStateType `json:"states" db:"states"`
	BatchID      string                          `json:"batchId" db:"batch_id"`
	BatchType    model.SoftwareUpdateBatchType   `json:"type" db:"type"`
}

// DownloadedServiceDomainsQueryParam is the query param for downloaded service domains
type DownloadedServiceDomainsQueryParam struct {
	TenantID     string   `json:"tenantId" db:"tenant_id"`
	SvcDomainIDs []string `json:"svcDomainIds" db:"svc_domain_ids"`
	Release      string   `json:"release" db:"release"`
}

// NewHandler returns the handler for software udpdate
func NewHandler() *Handler {
	handler := &Handler{
		stateTransitions: map[model.SoftwareUpdateStateType]map[model.SoftwareUpdateStateType]bool{
			model.DownloadState: map[model.SoftwareUpdateStateType]bool{
				model.DownloadingState:       true,
				model.DownloadCancelState:    true,
				model.DownloadCancelledState: true,
				model.DownloadFailedState:    true,
				model.DownloadedState:        true,
			},
			model.DownloadingState: map[model.SoftwareUpdateStateType]bool{
				model.DownloadCancelState:    true,
				model.DownloadCancelledState: true,
				model.DownloadFailedState:    true,
				model.DownloadedState:        true,
			},
			model.DownloadCancelState: map[model.SoftwareUpdateStateType]bool{
				model.DownloadCancelledState: true,
				model.DownloadState:          true,
			},
			model.DownloadCancelledState: map[model.SoftwareUpdateStateType]bool{
				model.DownloadState: true,
			},
			model.DownloadFailedState: map[model.SoftwareUpdateStateType]bool{
				model.DownloadState: true,
			},
			model.UpgradeState: map[model.SoftwareUpdateStateType]bool{
				model.UpgradingState:     true,
				model.UpgradeFailedState: true,
				model.UpgradedState:      true,
			},
			model.UpgradingState: map[model.SoftwareUpdateStateType]bool{
				model.UpgradeFailedState: true,
				model.UpgradedState:      true,
			},
			model.UpgradeFailedState: map[model.SoftwareUpdateStateType]bool{
				model.UpgradeState: true,
			},
		},
	}

	// Set up db
	dbURL, err := base.GetDBURL(*config.Cfg.SQLDialect, *config.Cfg.SQLDB, *config.Cfg.SQLUser, *config.Cfg.SQLPassword, *config.Cfg.SQLHost, *config.Cfg.SQLPort, *config.Cfg.DisableDBSSL)
	if err != nil {
		glog.Errorf("Failed to construct DB URL. Error: %s", err.Error())
		panic(err)
	}
	roDbURL := dbURL
	if config.Cfg.SQLReadOnlyHost != nil && len(*config.Cfg.SQLReadOnlyHost) > 0 {
		roDbURL, err = base.GetDBURL(*config.Cfg.SQLDialect, *config.Cfg.SQLDB, *config.Cfg.SQLUser, *config.Cfg.SQLPassword, *config.Cfg.SQLReadOnlyHost, *config.Cfg.SQLPort, *config.Cfg.DisableDBSSL)
		if err != nil {
			panic(err)
		}
	}
	dbAPI, err := base.NewDBObjectModelAPI(*config.Cfg.SQLDialect, dbURL, roDbURL, nil)
	if err != nil {
		glog.Errorf("Failed to create db object model API instance. Error: %s", err.Error())
		panic(err)
	}
	handler.dbAPI = dbAPI
	handler.awsSession = common.GetAWSSession()
	return handler
}

func makeEntityListResponsePayload(entityType string, queryParam *model.EntitiesQueryParam, totalCount int) model.EntityListResponsePayload {
	if queryParam == nil {
		// default
		queryParam = &model.EntitiesQueryParam{}
	}
	return model.EntityListResponsePayload{
		PageIndex:   queryParam.PageIndex,
		PageSize:    queryParam.PageSize,
		TotalCount:  totalCount,
		OrderBy:     strings.Join(queryParam.OrderBy, ", "),
		OrderByKeys: orderByHelper.GetOrderByKeys(entityType),
	}
}

// NextState returns the next state if the transition is possible along with a flag indicating if there is any change in state
func (handler *Handler) NextState(ctx context.Context, currentState, input model.SoftwareUpdateStateType) (model.SoftwareUpdateStateType, bool, error) {
	if currentState == input {
		// Allow transition to self
		return input, false, nil
	}
	nextStates, ok := handler.stateTransitions[currentState]
	if !ok {
		errMsg := fmt.Sprintf("Invalid current state %s", currentState)
		glog.Errorf(base.PrefixRequestID(ctx, "Error in NextState. Error: %s"), errMsg)
		return input, false, errcode.NewInternalError(errMsg)
	}
	if !nextStates[input] {
		errMsg := fmt.Sprintf("Invalid next state %s for current state %s", input, currentState)
		glog.Error(base.PrefixRequestID(ctx, errMsg))
		return input, false, errcode.NewBadRequestExError("state", errMsg)
	}
	glog.Infof(base.PrefixRequestID(ctx, "State changed to %s for current state %s with input %s"), input, currentState, input)
	return input, true, nil
}

// CanStartSoftwareUpdate checks if download/upgrade can be initiated on a list of service domains.
// If any of the service domains is in non-terminal state, it cannot be included in the process
func (handler *Handler) CanStartSoftwareUpdate(ctx context.Context, tx *base.WrappedTx, svcDomainIDs []string) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	// Check for all non-terminal states for both download and upgrade
	svcDomainsQueryParam := ServiceDomainsQueryParam{
		TenantID:     authCtx.TenantID,
		SvcDomainIDs: svcDomainIDs,
		States:       nonTerminalUpdateForbiddenStates,
	}
	svcDomainDBOs := []ServiceDomainDBO{}
	err = base.QueryInTxn(ctx, tx, &svcDomainDBOs, canCreateBatchQuery, svcDomainsQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domains %+v in unexpected states %+v. Error: %s"), svcDomainIDs, svcDomainsQueryParam.States, err.Error())
		return err
	}
	if len(svcDomainDBOs) > 0 {
		// Report only one at a time to avoid too long error message
		errMsg := fmt.Sprintf("Invalid state %s for service domain %s", svcDomainDBOs[0].State, svcDomainDBOs[0].SvcDomainID)
		return errcode.NewBadRequestExError("servicedomain", errMsg)
	}
	return nil
}

// CanRetrySoftwareUpdate checks if download/upgrade can be retried on a list of service domains.
// If any of the service domains is in non-terminal state, it cannot be included in the process
func (handler *Handler) CanRetrySoftwareUpdate(ctx context.Context, tx *base.WrappedTx, batchID string) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	svcDomainQueryParam := ServiceDomainQueryParam{
		TenantID: authCtx.TenantID,
		States:   nonTerminalUpdateForbiddenStates,
		BatchID:  batchID,
	}
	svcDomainDBOs := []ServiceDomainDBO{}
	err = base.QueryInTxn(ctx, tx, &svcDomainDBOs, canUpdateBatchQuery, svcDomainQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domains for batch %s. Error: %s"), batchID, err.Error())
		return err
	}
	if len(svcDomainDBOs) > 0 {
		// Report only one at a time to avoid too long error message
		errMsg := fmt.Sprintf("Invalid state  %s for service domain %s", svcDomainDBOs[0].State, svcDomainDBOs[0].SvcDomainID)
		return errcode.NewBadRequestExError("batch", errMsg)
	}
	return nil
}

// InsertBatch inserts a batch record provided the service domains are not in non-terminal states
func (handler *Handler) InsertBatch(ctx context.Context, tx *base.WrappedTx, batchDBO *BatchDBO, svcDomainIDs []string) error {
	states := funk.Map(nonTerminalUpdateForbiddenStates, func(state model.SoftwareUpdateStateType) string {
		return string(state)
	}).([]string)

	insertBatchDBO := &InsertBatchDBO{
		BatchDBO:     *batchDBO,
		SvcDomainIDs: fmt.Sprintf("'%s'", strings.Join(svcDomainIDs, "','")),
		States:       fmt.Sprintf("'%s'", strings.Join(states, "','")),
	}
	// Insert the new batch
	_, err := tx.NamedExec(ctx, insertBatchQuery, insertBatchDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in inserting batch record %+v. Error: %s"), batchDBO, err.Error())
	}
	return err
}

// ReadBatchDetails returns the batch details matching the parameters from the DB
func (handler *Handler) ReadBatchDetails(ctx context.Context, tx *base.WrappedTx, batchID string, batchType model.SoftwareUpdateBatchType, svcDomainID, release string, svcDomainStates []model.SoftwareUpdateStateType) (*BatchDetails, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	batchDBOs := []BatchDBO{}
	batchQueryParam := BatchQueryParam{
		TenantID: authCtx.TenantID,
		ID:       batchID,
		Release:  release,
		Type:     batchType,
	}
	err = handler.dbAPI.QueryIn(ctx, &batchDBOs, selectBatchesQuery, batchQueryParam)
	if err != nil {
		return nil, err
	}
	if len(batchDBOs) == 0 {
		return nil, errcode.NewRecordNotFoundError("batch")
	}
	batchDetails := &BatchDetails{BatchDBO: batchDBOs[0]}
	// Include service domain details only when the states are specified
	svcDomainQueryParam := ServiceDomainQueryParam{
		TenantID:    authCtx.TenantID,
		SvcDomainID: svcDomainID,
		States:      svcDomainStates,
		BatchID:     batchID,
	}
	svcDomainDBOs := []ServiceDomainDBO{}
	if len(svcDomainStates) == 0 {
		// Get service domains irrespective of the states
		err = base.QueryTxn(ctx, tx, &svcDomainDBOs, selectAllServiceDomainsForUpdateQuery, svcDomainQueryParam)
	} else {
		err = base.QueryInTxn(ctx, tx, &svcDomainDBOs, selectServiceDomainsForUpdateQuery, svcDomainQueryParam)
	}
	if err != nil {
		return nil, err
	}
	if len(svcDomainDBOs) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "No service domain %s in batch %s is not in states %+v"), svcDomainID, batchID, svcDomainStates)
		return nil, errcode.NewRecordNotFoundError("servicedomain")
	}
	batchDetails.SvcDomains = make([]*ServiceDomainDBO, 0, len(svcDomainDBOs))
	for i := range svcDomainDBOs {
		svcDomain := &svcDomainDBOs[i]
		batchDetails.SvcDomains = append(batchDetails.SvcDomains, svcDomain)
	}
	return batchDetails, nil
}

// VerifyDownloadedServiceDomains validates if the service domains have already downloaded the release
func (handler *Handler) VerifyDownloadedServiceDomains(ctx context.Context, tx *base.WrappedTx, svcDomainIDs []string, release string) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	svcDomainDBOs := []DownloadedServiceDomainDBO{}
	param := DownloadedServiceDomainsQueryParam{TenantID: authCtx.TenantID, Release: release, SvcDomainIDs: svcDomainIDs}
	err = base.QueryInTxn(ctx, tx, &svcDomainDBOs, selectDownloadedServiceDomainsQuery, param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domains with the release %s downloaded. Error: %s"), release, err.Error())
		return err
	}
	if len(svcDomainDBOs) == len(svcDomainIDs) {
		return nil
	}
	// Find out which service domain IDs are not found
	svcDomainIDsMap := map[string]bool{}
	for _, svcDomainDBO := range svcDomainDBOs {
		svcDomainIDsMap[svcDomainDBO.SvcDomainID] = true
	}
	for _, svcDomainID := range svcDomainIDs {
		if !svcDomainIDsMap[svcDomainID] {
			errMsg := fmt.Sprintf("Service domain %s does not have the release %s", svcDomainID, release)
			glog.Errorf(base.PrefixRequestID(ctx, errMsg))
			return errcode.NewBadRequestExError("servicdomain", errMsg)
		}
	}
	return nil
}

// DeleteDownloadedServiceDomainRelease deletes the downloaded service domain record from the DB
func (handler *Handler) DeleteDownloadedServiceDomainRelease(ctx context.Context, tx *base.WrappedTx, svcDomainIDs []string, release string) error {
	if len(svcDomainIDs) == 0 {
		return errcode.NewBadRequestError("servicedomains")
	}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	queryParam := DownloadedServiceDomainDBO{TenantID: authCtx.TenantID}
	query := fmt.Sprintf(deleteDownloadedServiceDomainsTemplateQuery, strings.Join(svcDomainIDs, "','"))
	_, err = tx.NamedExec(ctx, query, queryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting downloaded service domain releases. Error: %s"), err.Error())
	}
	return err
}

// getFinalBatchDownloadState returns the final stats to be reported for a batch of downloads for service domains
func (handler *Handler) getFinalBatchDownloadState(ctx context.Context, defaultBatchState model.SoftwareUpdateStateType, statsCounts map[model.SoftwareUpdateStateType]int) model.SoftwareUpdateStateType {
	totalStatsCount := 0
	for key, count := range statsCounts {
		totalStatsCount += count
		if key == model.DownloadingState || key == model.DownloadState {
			return model.DownloadingState
		}
	}
	// failed, cancel, cancelled or downloaded
	// Some service domains are in download state
	if _, ok := statsCounts[model.DownloadFailedState]; ok {
		return model.DownloadFailedState
	}
	if _, ok := statsCounts[model.DownloadCancelState]; ok {
		return model.DownloadCancelState
	}
	if _, ok := statsCounts[model.DownloadCancelledState]; ok {
		return model.DownloadCancelledState
	}
	// All are in downloaded states
	if downloadedCount, ok := statsCounts[model.DownloadedState]; ok {
		if downloadedCount == totalStatsCount {
			return model.DownloadedState
		}
	}
	return defaultBatchState
}

// getFinalBatchUpgradeState returns the final stats to be reported for a batch of upgrades for service domains
func (handler *Handler) getFinalBatchUpgradeState(ctx context.Context, defaultBatchState model.SoftwareUpdateStateType, statsCounts map[model.SoftwareUpdateStateType]int) model.SoftwareUpdateStateType {
	totalStatsCount := 0
	for key, count := range statsCounts {
		totalStatsCount += count
		if key == model.UpgradingState || key == model.UpgradeState {
			return model.UpgradingState
		}
	}
	if _, ok := statsCounts[model.UpgradeFailedState]; ok {
		return model.UpgradeFailedState
	}
	// All are in upgraded states
	if upgradedCount, ok := statsCounts[model.UpgradedState]; ok {
		if upgradedCount == totalStatsCount {
			return model.UpgradedState
		}
	}
	return defaultBatchState
}

// ListBatches lists all the batches in the given states
func (handler *Handler) ListBatches(ctx context.Context, batchID string, batchType model.SoftwareUpdateBatchType, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateBatchListPayload, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	query, _, err := orderByHelper.BuildPagedQuery(entityTypeSoftwareUpdateBatch, selectBatchesTemplateQuery, queryParam, defaultOrderBy)
	if err != nil {
		return nil, err
	}
	batchDBOs := []BatchDBO{}
	batchQueryParam := BatchQueryParam{TenantID: authCtx.TenantID, ID: batchID, Type: batchType}
	err = handler.dbAPI.QueryIn(ctx, &batchDBOs, query, batchQueryParam)
	if err != nil {
		return nil, err
	}
	if len(batchDBOs) == 0 {
		return &model.SoftwareUpdateBatchListPayload{BatchList: []model.SoftwareUpdateBatch{}}, nil
	}
	totalCount := 0
	batchIDs := make([]string, 0, len(batchDBOs))
	batchMap := map[string]*model.SoftwareUpdateBatch{}
	for i := range batchDBOs {
		batchDBO := &batchDBOs[i]
		batch := model.SoftwareUpdateBatch{}
		err = base.Convert(batchDBO, &batch)
		if err != nil {
			return nil, err
		}
		if totalCount == 0 {
			totalCount = batchDBO.TotalCount
		}
		batchMap[batch.ID] = &batch
		batchIDs = append(batchIDs, batch.ID)
	}

	statsQueryParam := BatchStatsQueryParam{TenantID: authCtx.TenantID, BatchIDs: batchIDs}
	batchStatsDBOs := []BatchStatsDBO{}
	err = handler.dbAPI.QueryIn(ctx, &batchStatsDBOs, selectBatchStatsQuery, statsQueryParam)
	if err != nil {
		return nil, err
	}
	// Fill up the calcutated stats data
	for i := range batchStatsDBOs {
		batchStateDBO := &batchStatsDBOs[i]
		batch := batchMap[batchStateDBO.BatchID]
		if batch.Stats == nil {
			batch.Stats = map[model.SoftwareUpdateStateType]int{}
		}
		batch.Stats[batchStateDBO.State] = batchStateDBO.Count
		batch.Progress = batchStateDBO.Progress
		batch.ETA = batchStateDBO.ETA
	}
	batchList := make([]model.SoftwareUpdateBatch, 0, len(batchDBOs))
	for key := range batchMap {
		batch := batchMap[key]
		if batch.Type == model.DownloadBatchType {
			batch.State = handler.getFinalBatchDownloadState(ctx, model.SoftwareUpdateStateType(""), batch.Stats)
		} else if batch.Type == model.UpgradeBatchType {
			batch.State = handler.getFinalBatchUpgradeState(ctx, model.SoftwareUpdateStateType(""), batch.Stats)
		}
		batchList = append(batchList, *batch)
	}
	batchListPayload := &model.SoftwareUpdateBatchListPayload{
		EntityListResponsePayload: makeEntityListResponsePayload(entityTypeSoftwareUpdateBatch, queryParam, totalCount),
		BatchList:                 batchList,
	}
	return batchListPayload, err
}

// ListBatchServiceDomains list all the service domains in a batch
func (handler *Handler) ListBatchServiceDomains(ctx context.Context, batchID, svcDomainID string, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	query, _, err := orderByHelper.BuildPagedQueryWithTableAlias(entityTypeSoftwareUpdateServiceDomain, selectServiceDomainsTemplateQuery, queryParam, defaultOrderBy, softwareUpdateServiceDomainModelAlias, nil)
	if err != nil {
		return nil, err
	}
	svcDomainDBOs := []ServiceDomainDBO{}
	svcDomainQueryParam := ServiceDomainQueryParam{
		TenantID:    authCtx.TenantID,
		SvcDomainID: svcDomainID,
		BatchID:     batchID,
	}
	err = handler.dbAPI.QueryIn(ctx, &svcDomainDBOs, query, svcDomainQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domains in an active batch. Error: %s"), err.Error())
		return nil, err
	}
	totalCount := 0
	svcDomainList := make([]model.SoftwareUpdateServiceDomain, 0, len(svcDomainDBOs))
	for i := range svcDomainDBOs {
		svcDomainDBO := &svcDomainDBOs[i]
		svcDomain := model.SoftwareUpdateServiceDomain{}
		err = base.Convert(svcDomainDBO, &svcDomain)
		if err != nil {
			return nil, err
		}
		if totalCount == 0 {
			totalCount = svcDomainDBO.TotalCount
		}
		svcDomainList = append(svcDomainList, svcDomain)
	}
	svcDomainListPayload := &model.SoftwareUpdateServiceDomainListPayload{
		EntityListResponsePayload: makeEntityListResponsePayload(entityTypeSoftwareUpdateServiceDomain, queryParam, totalCount),
		SvcDomainList:             svcDomainList,
	}
	return svcDomainListPayload, nil
}

// ListServiceDomains lists all the service domains with batches
func (handler *Handler) ListServiceDomains(ctx context.Context, batchType model.SoftwareUpdateBatchType, svcDomainID string, isLatestBatch bool, queryParam *model.EntitiesQueryParam) (*model.SoftwareUpdateServiceDomainListPayload, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	currSvcDomainDBOs := []ServiceDomainDBO{}
	svcDomainQueryParam := ServiceDomainQueryParam{
		TenantID:    authCtx.TenantID,
		SvcDomainID: svcDomainID,
		BatchType:   batchType,
	}
	currSvcDomainBatchMap := map[string]string{}
	query := selectAllLatestServiceDomainBatchesTemplateQuery
	if !isLatestBatch {
		query = selectAllServiceDomainBatchesTemplateQuery

		// Fetch the latest batch IDs for the service domains
		err = handler.dbAPI.Query(ctx, &currSvcDomainDBOs, selectLatestServiceDomainBatchIDsQuery, svcDomainQueryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domains in latest batches. Error: %s"), err.Error())
			return nil, err
		}

		for _, svcDomainDBO := range currSvcDomainDBOs {
			currSvcDomainBatchMap[svcDomainDBO.SvcDomainID] = svcDomainDBO.BatchID
		}
	}
	query, _, err = orderByHelper.BuildPagedQueryWithTableAlias(entityTypeSoftwareUpdateServiceDomain, query, queryParam, defaultOrderBy, softwareUpdateServiceDomainModelAlias, nil)
	if err != nil {
		return nil, err
	}
	svcDomainDBOs := []ServiceDomainDBO{}
	err = handler.dbAPI.QueryIn(ctx, &svcDomainDBOs, query, svcDomainQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting service domain batches. Error: %s"), err.Error())
		return nil, err
	}

	totalCount := 0
	svcDomainList := make([]model.SoftwareUpdateServiceDomain, 0, len(svcDomainDBOs))
	for i := range svcDomainDBOs {
		svcDomainDBO := &svcDomainDBOs[i]
		svcDomain := model.SoftwareUpdateServiceDomain{}
		err = base.Convert(svcDomainDBO, &svcDomain)
		if err != nil {
			return nil, err
		}
		if totalCount == 0 {
			totalCount = svcDomainDBO.TotalCount
		}
		if isLatestBatch {
			svcDomain.IsLatestBatch = true
		} else if batchID, ok := currSvcDomainBatchMap[svcDomain.SvcDomainID]; ok && batchID == svcDomain.BatchID {
			svcDomain.IsLatestBatch = true
		}
		svcDomainList = append(svcDomainList, svcDomain)
	}
	svcDomainListPayload := &model.SoftwareUpdateServiceDomainListPayload{
		EntityListResponsePayload: makeEntityListResponsePayload(entityTypeSoftwareUpdateServiceDomain, queryParam, totalCount),
		SvcDomainList:             svcDomainList,
	}
	return svcDomainListPayload, nil

}

// DeleteBatch deletes the specified batch
func (handler *Handler) DeleteBatch(ctx context.Context, batchID string) error {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	batchQueryParam := BatchQueryParam{TenantID: authCtx.TenantID, ID: batchID}
	_, err = handler.dbAPI.NamedExec(ctx, deleteBatchQuery, &batchQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in deleting batches with param %+v"), batchQueryParam)
	}
	return err
}

// GetCurrentServiceDomain returns the latest service domain which can state updated timestamp older than the input timestamp if the ID of the latest matches input ID.
// If the output is nil or it has a different ID, the input service domain ID can be deleted.
func (handler *Handler) GetCurrentServiceDomain(ctx context.Context, version *model.EntityVersionMetadata) (*model.SoftwareUpdateServiceDomain, error) {
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	yes, svcDomainID := base.IsEdgeRequest(authCtx)
	if !yes {
		return nil, errcode.NewBadRequestError("context")
	}
	svcDomainDBOs := []ServiceDomainDBO{}
	queryParam := ServiceDomainQueryParam{TenantID: authCtx.TenantID, BatchID: version.ID, States: nonTerminalStates, SvcDomainID: svcDomainID, StateUpdatedAt: version.UpdatedAt}
	err = handler.dbAPI.QueryIn(ctx, &svcDomainDBOs, getInventoryDeltaQuery, queryParam)
	if err != nil {
		return nil, err
	}
	if len(svcDomainDBOs) == 0 {
		return nil, nil
	}
	svcDomainDBO := &svcDomainDBOs[0]
	svcDomain := &model.SoftwareUpdateServiceDomain{}
	err = base.Convert(svcDomainDBO, svcDomain)
	if err != nil {
		return nil, err
	}
	return svcDomain, nil
}

// GetBucketPolicyDocument gets the AWS bucket policy document
func (handler *Handler) GetBucketPolicyDocument(ctx context.Context, releasePath string) (string, error) {
	if releasePath == "" {
		return "", errcode.NewBadRequestError("releasePath")
	}
	releaseBucketAccessPolicy := releaseAWSBucketAccessPolicy
	storageEngine := strings.ToLower(*config.Cfg.ObjectStorageEngine)
	if storageEngine == "minio" {
		releaseBucketAccessPolicy = releaseMinioBucketAccessPolicy
	}
	policyTemplate, err := template.New("awsPolicyDoc").Parse(releaseBucketAccessPolicy)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in parsing S3 bucket policy %s. Error: %s"), releaseBucketAccessPolicy, err.Error())
		return "", err
	}
	// Just remove the prefix in case it is added to avoid errors
	s3Bucket := strings.TrimPrefix(*config.Cfg.S3Bucket, "s3://")
	s3Bucket = strings.TrimSuffix(s3Bucket, "/")
	releasePath = strings.TrimPrefix(releasePath, "s3://")
	releasePath = strings.TrimSuffix(releasePath, "/")
	substitutions := map[string]string{
		"BUCKET_NAME":  s3Bucket,
		"RELEASE_PATH": releasePath,
	}
	var w bytes.Buffer
	err = policyTemplate.Execute(&w, substitutions)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to substitute in template %s with release path %s. Error: %s"), releaseBucketAccessPolicy, releasePath, err.Error())
		return "", errcode.NewBadRequestError("release")
	}
	//return base64.URLEncoding.EncodeToString(w.Bytes()), nil
	return w.String(), nil
}

// ReleaseToS3Path converts a release to the S3 path like s3://<bucket>/<prefixes>
func (handler *Handler) ReleaseToS3Path(ctx context.Context, release string) (string, error) {
	versionObj, err := version.NewVersion(release)
	if err != nil {
		return "", errcode.NewBadRequestExError("version", err.Error())
	}
	strSegments := funk.Map(versionObj.Segments(), func(segment int) string {
		return fmt.Sprint(segment)
	}).([]string)
	versionPath := strings.Join(strSegments, "/")
	return fmt.Sprintf(s3ReleaseRootDirectoryFormat, *config.Cfg.S3Bucket, *config.Cfg.S3Prefix, versionPath), nil
}

// findNextRune: find index of next rune r in rs starting at start
// return the index or -1 if not found
func findNextRune(rs []rune, r rune, start int) int {
	n := len(rs)
	for i := start; i < n; i++ {
		if rs[i] == r {
			return i
		}
	}
	return -1
}

// extractToRune: if arn has prefix pfx,
// then extract arn after pfx till the next rune r
// return "" if not found
func extractToRune(arn, pfx string, r rune) string {
	if strings.HasPrefix(arn, pfx) {
		rs := []rune(arn)
		n := len(pfx)
		i := findNextRune(rs, r, n)
		if i != -1 {
			a := string(rs[n:i])
			return a
		}
	}
	return ""
}

// getIamArn: given sts ARN and Account string,
// construct the IAM ARN string
// return "" if no match
func getIamArn(arn, acct string) string {
	pfx := fmt.Sprintf("arn:aws:sts::%s:assumed-role/", acct)
	role := extractToRune(arn, pfx, '/')
	if role != "" {
		return fmt.Sprintf("arn:aws:iam::%s:role/%s", acct, role)
	}
	return ""
}

func (handler *Handler) getECRLoginToken(ctx context.Context) (map[string]string, error) {
	responseMap := map[string]string{}
	sess := common.GetOTAAWSSession()
	svc := ecr.New(sess)
	input := &ecr.GetAuthorizationTokenInput{}
	result, err := svc.GetAuthorizationToken(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			errMsg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
			glog.Errorf(base.PrefixRequestID(ctx, errMsg))
			return responseMap, errcode.NewInternalError(errMsg)
		}
		glog.Errorf(base.PrefixRequestID(ctx, err.Error()))
		return responseMap, errcode.NewInternalError(err.Error())
	}

	// Extract base 64 decoded aws pass
	creds, err := base64.StdEncoding.DecodeString(*result.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, err.Error()))
		return responseMap, errcode.NewInternalError(err.Error())
	}
	userPass := strings.Split(string(creds), ":")
	if len(userPass) != 2 {
		errMsg := fmt.Sprintf("username password can not be extracted from creds: %s", creds)
		glog.Errorf(base.PrefixRequestID(ctx, errMsg))
		return responseMap, errcode.NewInternalError(errMsg)
	}
	pass := userPass[1]
	repo := *result.AuthorizationData[0].ProxyEndpoint
	loginToken := fmt.Sprintf("docker login -u AWS -p %s %s", pass, repo)
	responseMap["token"] = loginToken
	responseMap["user"] = "AWS"
	responseMap["password"] = pass
	responseMap["repo"] = repo
	return responseMap, nil
}

func (handler *Handler) getPrivateDockerLoginToken(ctx context.Context) (map[string]string, error) {
	responseMap := map[string]string{}
	user := *config.Cfg.DockerRegistryUser
	pass := *config.Cfg.DockerRegistryPassword
	repo := *config.Cfg.DockerRegistryURL
	loginToken := fmt.Sprintf("docker login -u AWS -p %s %s", pass, repo)
	responseMap["token"] = loginToken
	responseMap["user"] = user
	responseMap["password"] = pass
	responseMap["repo"] = repo
	return responseMap, nil
}

// GetDockerLoginToken returns docker credentials
func (handler *Handler) GetDockerLoginToken(ctx context.Context, credentials *model.SoftwareUpdateCredentials) (map[string]string, error) {
	provider := strings.ToLower(*config.Cfg.DockerRegistryProvider)
	if provider == "aws" {
		return handler.getECRLoginToken(ctx)
	}
	if provider == "private" {
		return handler.getPrivateDockerLoginToken(ctx)
	}
	return map[string]string{}, errcode.NewBadRequestError("dockerRegistryProvider")
}

func (handler *Handler) getAWSCredentials(ctx context.Context, awsPolicy string) (map[string]string, error) {
	responseMap := map[string]string{}
	svc := sts.New(handler.awsSession)
	tokenLifetimeMins := *config.Cfg.AWSFederatedTokenLifetimeMins
	request := &sts.GetFederationTokenInput{
		Name:            aws.String("software-update"),
		DurationSeconds: aws.Int64(int64(tokenLifetimeMins * 60)),
		Policy:          aws.String(awsPolicy),
	}
	var awsCredentials *sts.Credentials
	// This is supposed to work for minio as access key ID and secret are passed in
	response, err := svc.GetFederationTokenWithContext(ctx, request)
	if err != nil {
		// GetFederationToken failed, try AssumeRole
		// Need this b/c if use IAM role for EC2 (e.g., kube2iam),
		// We will be running with a role which can't be granted GetFederationToken
		// but can be granted AssumeRole.
		// For this to work, we need to grant role cloud_operator_user_role
		// the following IAM Policy permission:
		/*
			{
			  "Version": "2012-10-17",
			  "Statement": {
			    "Effect": "Allow",
			    "Action": "sts:AssumeRole",
			    "Resource": "arn:aws:iam::ACCOUNT-ID-WITHOUT-HYPHENS:role/cloud_operator_user_role"
			  }
			}
		*/
		cidInput := &sts.GetCallerIdentityInput{}
		cidOutput, err2 := svc.GetCallerIdentity(cidInput)
		if err2 == nil {
			if cidOutput.Account != nil && cidOutput.Arn != nil {
				// arn is of the form:
				// arn:aws:sts::<account>:assumed-role/<role>/<dont care>
				iamArn := getIamArn(*cidOutput.Arn, *cidOutput.Account)
				if iamArn != "" {
					// iamArn is of form:
					// arn:aws:iam::<account>:role/<role>
					// try assume role
					assumeRoleInput := &sts.AssumeRoleInput{
						RoleSessionName: aws.String("software-update"),
						DurationSeconds: aws.Int64(int64(tokenLifetimeMins * 60)),
						Policy:          aws.String(awsPolicy),
						RoleArn:         aws.String(iamArn),
					}
					assumeRoleOutput, err2 := svc.AssumeRole(assumeRoleInput)
					if err2 == nil {
						awsCredentials = assumeRoleOutput.Credentials
						glog.V(4).Infof(base.PrefixRequestID(ctx, "Using credentials from assume role: %+v"), awsCredentials)
					} else {
						glog.Errorf(base.PrefixRequestID(ctx, err2.Error()))
					}
				} else {
					glog.Errorf(base.PrefixRequestID(ctx, "Assume role: failed to get Arn"))
				}
			} else {
				glog.Errorf(base.PrefixRequestID(ctx, "Caller Identity: Arn or Account null"))
			}
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, err2.Error()))
		}
		if awsCredentials == nil {
			if aerr, ok := err.(awserr.Error); ok {
				errMsg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
				glog.Errorf(base.PrefixRequestID(ctx, errMsg))
			} else {
				glog.Errorf(base.PrefixRequestID(ctx, err.Error()))
			}
			return nil, err
		}
	} else {
		awsCredentials = response.Credentials
		if awsCredentials.AccessKeyId != nil {
			responseMap["aws_access_key_id"] = *awsCredentials.AccessKeyId
		}
		if awsCredentials.SecretAccessKey != nil {
			responseMap["aws_secret_access_key"] = *awsCredentials.SecretAccessKey
		}
		if awsCredentials.SessionToken != nil {
			responseMap["aws_session_token"] = *awsCredentials.SessionToken
		}
	}
	return responseMap, nil
}

// getMinioCredentials returns temporary minio credentials
func (handler *Handler) getMinioCredentials(ctx context.Context, awsPolicy string) (map[string]string, error) {
	responseMap := map[string]string{}
	svc := sts.New(handler.awsSession)
	tokenLifetimeMins := *config.Cfg.AWSFederatedTokenLifetimeMins
	roleArn := "arn:xxx:xxx:xxx:xxxx" // Fake ARN for minio
	response, err := svc.AssumeRole(&sts.AssumeRoleInput{
		RoleSessionName: aws.String("software-update"),
		DurationSeconds: aws.Int64(int64(tokenLifetimeMins * 60)),
		Policy:          aws.String(awsPolicy),
		RoleArn:         &roleArn,
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			errMsg := fmt.Sprintf("%s: %s", aerr.Code(), aerr.Error())
			glog.Errorf(base.PrefixRequestID(ctx, errMsg))
		} else {
			glog.Errorf(base.PrefixRequestID(ctx, err.Error()))
		}
		return nil, err
	}
	awsCredentials := response.Credentials
	if awsCredentials.AccessKeyId != nil {
		responseMap["aws_access_key_id"] = *awsCredentials.AccessKeyId
	}
	if awsCredentials.SecretAccessKey != nil {
		responseMap["aws_secret_access_key"] = *awsCredentials.SecretAccessKey
	}
	if awsCredentials.SessionToken != nil {
		responseMap["aws_session_token"] = *awsCredentials.SessionToken
	}
	return responseMap, nil
}

// GetAWSFederatedToken returns temporary aws credentials
func (handler *Handler) GetAWSFederatedToken(ctx context.Context, credentials *model.SoftwareUpdateCredentials) (map[string]string, error) {
	responseMap := map[string]string{}
	var awsPolicy string
	if credentials.AccessType == model.AWSCredentialsAccessType {
		releasePath, err := handler.ReleaseToS3Path(ctx, credentials.Release)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the release S3 path for release %s. Error: %s"), credentials.Release, err.Error())
			return responseMap, err
		}
		responseMap["path"] = releasePath
		// Get the bucket policy with limited access to this versioned folder only
		awsPolicy, err = handler.GetBucketPolicyDocument(ctx, releasePath)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the S3 bucket policy for release %s. Error: %s"), credentials.Release, err.Error())
			return responseMap, err
		}
		glog.Infof(base.PrefixRequestID(ctx, "Got policy %s"), awsPolicy)
	} else if credentials.AccessType == model.AWSECRCredentialsAccessType {
		awsPolicy = awsECRAccessPolicy
	}
	storageEngine := strings.ToLower(*config.Cfg.ObjectStorageEngine)
	if storageEngine == "aws" {
		return handler.getAWSCredentials(ctx, awsPolicy)
	}
	if storageEngine == "minio" {
		return handler.getMinioCredentials(ctx, awsPolicy)
	}
	return responseMap, errcode.NewBadRequestExError("storageEngine", "Unknown storage engine "+storageEngine)
}

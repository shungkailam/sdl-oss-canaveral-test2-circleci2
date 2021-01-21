package model

import "time"

const (
	// DownloadBatchType indicates the batch request is download type
	DownloadBatchType = SoftwareUpdateBatchType("DOWNLOAD")
	// UpgradeBatchType indicates the batch request is upgrade type
	UpgradeBatchType = SoftwareUpdateBatchType("UPGRADE")

	// DockerCredentialsAccessType indicates docker credentials access type
	DockerCredentialsAccessType = SoftwareUpdateCredentialsAccessType("DOCKER")
	// AWSCredentialsAccessType indicates AWS federated token access type
	AWSCredentialsAccessType = SoftwareUpdateCredentialsAccessType("AWS")

	// AWSECRCredentialsAccessType indicates AWS ECR federated token access type
	AWSECRCredentialsAccessType = SoftwareUpdateCredentialsAccessType("AWS_ECR")

	// DownloadCommand is the command sent by the UI or client to download
	DownloadCommand = SoftwareDownloadCommand("DOWNLOAD")
	// DownloadCancelCommand is the command sent by the UI or client to cancel download
	DownloadCancelCommand = SoftwareDownloadCommand("DOWNLOAD_CANCEL")

	// All the states for download
	DownloadState          = SoftwareUpdateStateType("DOWNLOAD")
	DownloadingState       = SoftwareUpdateStateType("DOWNLOADING")
	DownloadCancelState    = SoftwareUpdateStateType("DOWNLOAD_CANCEL")
	DownloadCancelledState = SoftwareUpdateStateType("DOWNLOAD_CANCELLED")
	DownloadFailedState    = SoftwareUpdateStateType("DOWNLOAD_FAILED")
	DownloadedState        = SoftwareUpdateStateType("DOWNLOADED")

	// UpgradeCommand is the command to start upgrade
	UpgradeCommand = SoftwareUpgradeCommand("UPGRADE")

	// All the states for upgrade
	UpgradeState       = SoftwareUpdateStateType("UPGRADE")
	UpgradingState     = SoftwareUpdateStateType("UPGRADING")
	UpgradeFailedState = SoftwareUpdateStateType("UPGRADE_FAILED")
	UpgradedState      = SoftwareUpdateStateType("UPGRADED")
)

// swagger:parameters SoftwareReleaseList SoftwareDownloadedServiceDomainList SoftwareDownloadBatchList SoftwareDownloadBatchGet SoftwareDownloadServiceDomainList SoftwareUpdateServiceDomainList SoftwareDownloadCreate SoftwareDownloadUpdate SoftwareDownloadStateUpdate SoftwareUpgradeBatchList SoftwareUpgradeBatchGet SoftwareUpgradeServiceDomainList SoftwareUpgradeCreate SoftwareUpgradeUpdate SoftwareUpgradeStateUpdate SoftwareUpdateCredentialsCreate
// in: header
type softwareUpdateAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// SoftwareRelease holds the release info
type SoftwareRelease struct {
	//
	//	This is the release that is avaliable
	//
	// required: true
	Release string `json:"release"`
	//
	// The changes that were made in this release from the previous release
	//
	// required: true
	Changelog string `json:"changelog"`
}

// SoftwareReleaseListPayload is the payload for batch list REST response
type SoftwareReleaseListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of download batches
	// required: true
	ReleaseList []SoftwareRelease `json:"result"`
}

// Ok
// swagger:response SoftwareReleaseListResponse
type SoftwareReleaseListResponse struct {
	// in: body
	// required: true
	Payload *SoftwareReleaseListPayload
}

// SoftwareDownloadedServiceDomainListPayload is the payload for batch list REST response
type SoftwareDownloadedServiceDomainListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of service domains with the release downloaded
	// required: true
	SvcDomainList []string `json:"result"`
}

// Ok
// swagger:response SoftwareDownloadedServiceDomainListResponse
type SoftwareDownloadedServiceDomainListResponse struct {
	// in: body
	// required: true
	Payload *SoftwareDownloadedServiceDomainListPayload
}

// SoftwareUpdateStateType represents the state of the completed/terminated/ongoing update
// swagger:model SoftwareUpdateStateType
// enum: DOWNLOAD,DOWNLOADING,DOWNLOAD_CANCEL,DOWNLOAD_CANCELLED,DOWNLOAD_FAILED,DOWNLOADED,UPGRADE,UPGRADING,UPGRADE_FAILED,UPGRADED
type SoftwareUpdateStateType string

// SoftwareUpdateCommon holds the common stats
type SoftwareUpdateCommon struct {
	State SoftwareUpdateStateType `json:"state"`
	// Progress in percentage
	Progress int `json:"progress"`
	// ETA in mins
	ETA int `json:"eta"`
	// required: true
	Release string `json:"release"`
	// Created timestamp
	CreatedAt time.Time `json:"createdAt"`
	// Record updated timestamp
	UpdatedAt time.Time `json:"updatedAt"`
}

// SoftwareUpdateBatchType defines the type of the batch
// swagger:model SoftwareUpdateBatchType
// enum: DOWNLOAD,UPGRADE
type SoftwareUpdateBatchType string

// SoftwareUpdateBatch is the model representing the batch download/upgrade response
type SoftwareUpdateBatch struct {
	SoftwareUpdateCommon
	// required: true
	ID string `json:"id"`
	// required: true
	Type SoftwareUpdateBatchType `json:"type"`
	// Count for each stat type
	Stats map[SoftwareUpdateStateType]int `json:"stats"`
}

// Ok
// swagger:response SoftwareUpdateBatchGetResponse
type SoftwareUpdateBatchGetResponse struct {
	// in: body
	// required: true
	Payload *SoftwareUpdateBatch
}

// SoftwareUpdateBatchListPayload is the payload for batch list REST response
type SoftwareUpdateBatchListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of download/upgrade batches
	// required: true
	BatchList []SoftwareUpdateBatch `json:"result"`
}

// Ok
// swagger:response SoftwareUpdateBatchListResponse
type SoftwareUpdateBatchListResponse struct {
	// in: body
	// required: true
	Payload *SoftwareUpdateBatchListPayload
}

// SoftwareUpdateServiceDomain is the model representing the batch download/update response for a service domain
type SoftwareUpdateServiceDomain struct {
	SoftwareUpdateCommon
	// required: true
	BatchID string `json:"batchId"`
	// required: true
	SvcDomainID string `json:"svcDomainId"`
	// Failure reason if any
	FailureReason *string `json:"failureReason,omitempty"`
	// State updated timestamp
	StateUpdatedAt time.Time `json:"stateUpdatedAt"`
	// Latest batch
	IsLatestBatch bool `json:"isLatestBatch"`
}

// Required for NOTIFICATION_EDGE
func (svcDomain SoftwareUpdateServiceDomain) GetID() string {
	return svcDomain.SvcDomainID
}

// Required for NOTIFICATION_EDGE
func (svcDomain SoftwareUpdateServiceDomain) GetClusterID() string {
	return svcDomain.SvcDomainID
}

// SoftwareUpdateServiceDomainListPayload is the payload for batch list REST response
type SoftwareUpdateServiceDomainListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of service domain stats
	// required: true
	SvcDomainList []SoftwareUpdateServiceDomain `json:"result"`
}

// Ok
// swagger:response SoftwareUpdateServiceDomainListResponse
type SoftwareUpdateServiceDomainListResponse struct {
	// in: body
	// required: true
	Payload *SoftwareUpdateServiceDomainListPayload
}

// SoftwareDownloadCreate is the model for triggering downloads
type SoftwareDownloadCreate struct {
	SvcDomainIDs []string `json:"svcDomainIds"`
	Release      string   `json:"release"`
}

// swagger:parameters SoftwareDownloadCreate
// in: body
type SoftwareDownloadCreateRequest struct {
	// in: body
	// required: true
	Body *SoftwareDownloadCreate `json:"body"`
}

// SoftwareDownloadCommand indicates the type of commands issued by the client for download phase
// swagger:model SoftwareDownloadCommand
// enum: DOWNLOAD,DOWNLOAD_CANCEL
type SoftwareDownloadCommand string

// SoftwareDownloadUpdate is the model for modifying a download batch like cancel or retry download
type SoftwareDownloadUpdate struct {
	BatchID string                  `json:"batchId"`
	Command SoftwareDownloadCommand `json:"command"`
}

// swagger:parameters SoftwareDownloadUpdate
// in: body
type SoftwareDownloadUpdateRequest struct {
	// in: body
	// required: true
	Body *SoftwareDownloadUpdate `json:"body"`
}

// SoftwareUpgradeCreate is the model for triggering updates
type SoftwareUpgradeCreate struct {
	SvcDomainIDs []string `json:"svcDomainIds"`
	Release      string   `json:"release"`
}

// swagger:parameters SoftwareUpgradeCreate
// in: body
type SoftwareUpgradeCreateRequest struct {
	// in: body
	// required: true
	Body *SoftwareUpgradeCreate `json:"body"`
}

// SoftwareUpgradeCommand indicates the type of commands issued by the client for upgrade phase
// swagger:model SoftwareUpgradeCommand
// enum: UPGRADE
type SoftwareUpgradeCommand string

// SoftwareUpgradeUpdate is the model for modifying an upgrade batch like retry upgrade
type SoftwareUpgradeUpdate struct {
	// required: true
	BatchID string `json:"batchId"`
	// required: true
	Command SoftwareUpgradeCommand `json:"command"`
}

// swagger:parameters SoftwareUpgradeUpdate
// in: body
type SoftwareUpgradeUpdateRequest struct {
	// in: body
	// required: true
	Body *SoftwareUpgradeUpdate `json:"body"`
}

// SoftwareUpdateState is the model for updating state called by service domain (cluster)
type SoftwareUpdateState struct {
	SoftwareUpdateCommon
	// required: true
	SvcDomainID string `json:"svcDomainId"`
	// required: true
	BatchID string `json:"batchId"`
	// Failure reason if any
	FailureReason *string `json:"failureReason,omitempty"`
	// State updated timestamp
	StateUpdatedAt time.Time `json:"stateUpdatedAt"`
}

// swagger:parameters SoftwareDownloadStateUpdate SoftwareUpgradeStateUpdate
// in: body
type SoftwareUpdateStateRequest struct {
	// in: body
	// required: true
	Body *SoftwareUpdateState `json:"body"`
}

// Ok
// swagger:response SoftwareUpdateStateResponse
type SoftwareUpdateStateResponse struct {
	// in: body
	// required: true
	Payload *SoftwareUpdateState
}

// SoftwareUpdateCredentialsAccessType indicates the type of the credentials (docker login or aws credentials)
// swagger:model SoftwareUpdateCredentialsAccessType
// enum: DOCKER,AWS,AWS_ECR
type SoftwareUpdateCredentialsAccessType string

// SoftwareUpdateCredentials is the model for credentials to download software release files
type SoftwareUpdateCredentials struct {
	// required: true
	BatchID string `json:"batchId"`
	// required: true
	Release string `json:"release"`
	// required: true
	AccessType SoftwareUpdateCredentialsAccessType `json:"accessType"`
}

// swagger:parameters SoftwareUpdateCredentialsCreate
// in: body
type SoftwareUpdateCredentialsCreateRequest struct {
	// in: body
	// required: true
	Body *SoftwareUpdateCredentials `json:"body"`
}

// SoftwareUpdateCredentialsCreatePayload is the payload for batch list REST response
type SoftwareUpdateCredentialsCreatePayload struct {
	SoftwareUpdateCredentials
	// Details of the credentials
	Credentials map[string]string `json:"credentials"`
}

// Ok
// swagger:response SoftwareUpdateCredentialsCreateResponse
type SoftwareUpdateCredentialsCreateResponse struct {
	// in: body
	// required: true
	Payload *SoftwareUpdateCredentialsCreatePayload
}

// ObjectRequestBaseSoftwareUpdate is used as a websocket software update message
// swagger:model ObjectRequestBaseSoftwareUpdate
type ObjectRequestBaseSoftwareUpdate struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc SoftwareUpdateServiceDomain `json:"doc"`
}

// SoftwareUpdateServiceDomainQueryParam carries the first class query parameters
// swagger:parameters SoftwareUpdateServiceDomainList
// in: query
type SoftwareUpdateServiceDomainQueryParam struct {
	// Service Domain ID
	// in: query
	// required: false
	SvcDomainID string `json:"svcDomainId"`
	// Software DOWNLOAD/UPGRADE
	// in: query
	// required: false
	Type string `json:"type"`
	// Fetch only latest batches
	// in: query
	// required: false
	IsLatestBatch bool `json:"isLatestBatch"`
}

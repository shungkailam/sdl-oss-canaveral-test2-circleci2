package model

// LogEntry - a log entry describes the metadata for a log bundle
// from an edge collected in part of a batch for a given tenant.
// swagger:model LogEntry
type LogEntry struct {
	EdgeBaseModel
	// ID that identifies logs from different edge as the same batch.
	BatchID string `json:"batchId" validate:"range=1:36"`
	// Location or object key for the log in the bucket.
	Location string `json:"location" validate:"range=1:4096"`
	// Tags carry the properties of the log
	Tags []LogTag `json:"tags"`
	// Status of this log entry.
	Status LogUploadStatus `json:"status"`
	// Error message - optional, should be populated when status == 'FAILED'
	ErrorMessage string `json:"errorMessage,omitempty" validate:"range=0:200"`
}

// LogTag is a name value pair. It can be Application and the specific ID
// swagger:model LogTag
type LogTag struct {
	// Name of the tag
	Name string `json:"name,omitempty"`
	// Value of the tag
	Value string `json:"value,omitempty"`
}

// Status of the log entry - one of PENDING, SUCCESS, FAILED, TIMEDOUT
// swagger:model LogUploadStatus
// enum: PENDING,SUCCESS,FAILED,TIMEDOUT
type LogUploadStatus string

const (
	LogUploadPending  = "PENDING"
	LogUploadSuccess  = "SUCCESS"
	LogUploadFailed   = "FAILED"
	LogUploadTimedOut = "TIMEDOUT"
	ApplicationLogTag = "Application"
)

// swagger:model LogUploadPayload
type LogUploadPayload struct {
	// URL where the log will be uploaded by the edge
	// required: true
	URL string `json:"url"`
	// Optional ID of the application for which the log will be collected
	// required: false
	ApplicationID string `json:"applicationId"`
	// Batch ID of the log upload request
	// required: true
	BatchID string `json:"batchId"`
}

// swagger:model LogDownloadPayload
type LogDownloadPayload struct {
	// URL to download the log
	// required: true
	URL string `json:"url"`
}

// swagger:model LogUploadCompletePayload
type LogUploadCompletePayload struct {
	// required: true
	URL string `json:"url"`
	// required: true
	Status       LogUploadStatus `json:"status"`
	ErrorMessage string          `json:"errorMessage"`
}

// Log upload request from the UI
// swagger:model RequestLogUploadPayload
type RequestLogUploadPayload struct {
	// IDs of the edges from where the logs will be collected
	// required: true
	EdgeIDs []string `json:"edgeIds"`
	// Optional ID of the application for which the log will be collected
	// required: false
	ApplicationID string `json:"applicationId"`
}

// swagger:model RequestLogDownloadPayload
type RequestLogDownloadPayload struct {
	// Unique location of the log that is returned by the log listing API
	// required: true
	Location string `json:"location"`
}

// ObjectRequestLogUpload is used as websocket Log Upload message
// swagger:model ObjectRequestLogUpload
type ObjectRequestLogUpload struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc LogUploadPayload `json:"doc"`
}

// ObjectRequestLogUpload is used as websocket Log Upload complete message
// swagger:model ObjectResponseLogUploadComplete
type ObjectResponseLogUploadComplete struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc LogUploadCompletePayload `json:"doc"`
}

// Enable authorization on the endpoints
// swagger:parameters LogEntriesList LogEntriesListV2 EdgeLogEntriesListV2 EdgeLogEntriesGetV2 ApplicationLogEntriesListV2 ApplicationLogEntriesGetV2 LogEntryDelete LogEntryDeleteV2 LogRequestDownload LogRequestDownloadV2 LogUpload LogUploadV2 LogRequestUpload LogRequestUploadV2 LogUploadComplete LogUploadCompleteV2
// in: header
type logAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response LogEntriesListResponse
type LogEntriesListResponse struct {
	// in: body
	// required: true
	Payload *[]LogEntry
}

// Ok
// swagger:response LogEntriesListResponseV2
type LogEntriesListResponseV2 struct {
	// in: body
	// required: true
	Payload *LogEntriesListPayload
}

type LogEntriesListPayload struct {
	// required: true
	EntityListResponsePayload
	// Response payload containing the log entries
	// required: true
	LogEntryList []LogEntry `json:"result"`
}

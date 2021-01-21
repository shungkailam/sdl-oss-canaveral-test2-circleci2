package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// LogRequestDownloadParam
// parameter for logRequestDownload
// swagger:parameters LogRequestDownloadV2 LogRequestDownload
type LogRequestDownloadParam struct {
	// in: body
	// required: true
	Request *model.RequestLogDownloadPayload
}

// LogUploadParam
// param for logUpload message
// swagger:parameters LogUpload WsMessagingLogUpload
type LogUploadParam struct {
	// in: body
	// required: true
	Payload *model.ObjectRequestLogUpload `json:"payload"`
}

// LogRequestUploadParam
// parameter for logRequestUpload
// swagger:parameters LogRequestUploadV2 LogRequestUpload
type LogRequestUploadParam struct {
	// in: body
	// required: true
	Request *model.RequestLogUploadPayload
}

// Ok
// swagger:response LogRequestUploadResponse
type LogRequestUploadResponse struct {
	// in: body
	// required: true
	Payload []model.LogUploadPayload
}

// Ok
// LogUploadCompleteParam
// param for logUploadComplete message
// swagger:parameters LogUploadComplete WsMessagingLogUploadComplete LogUploadCompleteV2
type LogUploadCompleteParam struct {
	// in: body
	// required: true
	Payload *model.ObjectResponseLogUploadComplete
}

// Ok
// swagger:response LogRequestDownloadResponse
type LogRequestDownloadResponse string

// Ok
// swagger:response LogRequestDownloadResponseV2
type LogRequestDownloadResponseV2 struct {
	// in: body
	// required: true
	Payload model.LogDownloadPayload
}

func getLogRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	RequestLogUploadHandle := makeCustomMessageHandle(dbAPI, dbAPI.RequestLogUploadW, msgSvc, "logUpload", NOTIFICATION_EDGE, func(doc interface{}) *string {
		payload := doc.(*model.LogUploadPayload)
		edgeID, err := api.ExtractEdgeID(payload.URL)
		if err != nil {
			return nil
		}
		return &edgeID
	})

	RequestLogDownloadHandle := getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		err := dbAPI.RequestLogDownloadW(r.Context(), w, r.Body)
		handleResponse(w, r, err, "RequestLogDownload, tenantID=%s", ap.TenantID)
	})))

	UploadLogHandle := getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		err := dbAPI.UploadLogW(r.Context(), r.Body)
		handleResponse(w, r, err, "UploadLog, tenantID=%s", ap.TenantID)
	})))

	UploadLogCompleteHandle := getContext(dbAPI, CheckAuth(dbAPI, getAuthGzipHandle(func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		err := dbAPI.UploadLogCompleteW(r.Context(), r.Body)
		handleResponse(w, r, err, "UploadLogComplete, tenantID=%s", ap.TenantID)
	})))

	return []routeHandle{
		{
			method: "GET",
			path:   "/v1/logs/entries",
			// swagger:route GET /v1/logs/entries LogEntriesList
			//
			// Lists log entries. ntnx:ignore
			//
			// Retrieve all log entries.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LogEntriesListResponse
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllLogsW, "/logs/entries"),
		},
		{
			method: "GET",
			path:   "/v1/logs/entries/",
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllLogsW, "/logs/entries"),
		},
		{
			method: "GET",
			path:   "/v1.0/logs/entries",
			// swagger:route GET /v1.0/logs/entries Log LogEntriesListV2
			//
			// Lists log entries.
			//
			// Retrieve all log entries.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LogEntriesListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllLogsWV2, "/logs/entries"),
		},
		{
			method: "GET",
			path:   "/v1.0/logs/edges",
			// swagger:route GET /v1.0/logs/edges Log EdgeLogEntriesListV2
			//
			// Lists infrastructure log entries for edges.
			//
			// Retrieve all infrastructure log entries.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LogEntriesListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllEdgeLogsWV2, "/logs/edges"),
		},
		{
			method: "GET",
			path:   "/v1.0/logs/edges/:id",
			// swagger:route GET /v1.0/logs/edges/{id} Log EdgeLogEntriesGetV2
			//
			// Lists infrastructure log entries for an edge.
			//
			// Retrieve infrastructure log entries specific to an edge.
			// Use filter on batch ID to get logs entries specific to a batch.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LogEntriesListResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetEdgeLogsWV2, "/logs/edges/:id", "id"),
		},
		{
			method: "GET",
			path:   "/v1.0/logs/applications",
			// swagger:route GET /v1.0/logs/applications Log ApplicationLogEntriesListV2
			//
			// Lists application log entries.
			//
			// Retrieve all the application log entries.
			// Use filter on edge ID and batch ID to get the application log specific to an edge and a batch.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LogEntriesListResponseV2
			//       default: APIError
			handle: makeGetAllHandle(dbAPI, dbAPI.SelectAllApplicationLogsWV2, "/logs/applications"),
		},
		{
			method: "GET",
			path:   "/v1.0/logs/applications/:id",
			// swagger:route GET /v1.0/logs/applications/{id} Log ApplicationLogEntriesGetV2
			//
			// Lists applications log entries specific to an application.
			//
			// Retrieve application log entries specific to an application.
			// Use filter on edge ID and batch ID to get the application log specific to an edge and a batch.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LogEntriesListResponseV2
			//       default: APIError
			handle: makeGetHandle(dbAPI, dbAPI.GetApplicationLogsWV2, "/logs/applications/:id", "id"),
		},
		{
			method: "DELETE",
			path:   "/v1/logs/entries/:id",
			// swagger:route DELETE /v1/logs/entries/{id} LogEntryDelete
			//
			// Delete log entry by ID. ntnx:ignore
			//
			// Deletes the log entry with the given id.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponse
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteLogEntryW, msgSvc, "log", NOTIFICATION_NONE, "id"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/logs/entries/:id",
			// swagger:route DELETE /v1.0/logs/entries/{id} Log LogEntryDeleteV2
			//
			// Delete log entry by ID.
			//
			// Deletes the log entry with the given id.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: DeleteDocumentResponse
			//       default: APIError
			handle: makeDeleteHandle(dbAPI, dbAPI.DeleteLogEntryW, msgSvc, "log", NOTIFICATION_NONE, "id"),
		},
		{
			method: "POST",
			path:   "/v1/logs/requestDownload",
			// swagger:route POST /v1/logs/requestDownload LogRequestDownload
			//
			// Request log download. ntnx:ignore
			//
			// Generates the log download URL.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: LogRequestDownloadResponse
			//       default: APIError
			handle: RequestLogDownloadHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/logs/requestdownload",
			// swagger:route POST /v1.0/logs/requestdownload Log LogRequestDownloadV2
			//
			// Request log download.
			//
			// Generates the log download URL.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: LogRequestDownloadResponseV2
			//       default: APIError
			handle: RequestLogDownloadHandle,
		},
		{
			method: "POST",
			path:   "/v1/logs/upload",
			// swagger:route POST /v1/logs/upload LogUpload
			//
			// Upload Log. ntnx:ignore
			//
			// Upload log - for edge testing.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       default: APIError
			handle: UploadLogHandle,
		},
		{
			method: "POST",
			path:   "/v1/logs/requestUpload",
			// swagger:route POST /v1/logs/requestUpload LogRequestUpload
			//
			// Request log upload. ntnx:ignore
			//
			// Request edges to upload logs to S3.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: LogRequestUploadResponse
			//       default: APIError
			handle: RequestLogUploadHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/logs/requestupload",
			// swagger:route POST /v1.0/logs/requestupload Log LogRequestUploadV2
			//
			// Request log upload.
			//
			// Request edges to upload logs to cloud storage.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: LogRequestUploadResponse
			//       default: APIError
			handle: RequestLogUploadHandle,
		},
		{
			method: "POST",
			path:   "/v1/logs/uploadComplete",
			// swagger:route POST /v1/logs/uploadComplete LogUploadComplete
			//
			// Report log upload complete. ntnx:ignore
			//
			// Edge will use this API to notify log upload complete.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       default: APIError
			handle: UploadLogCompleteHandle,
		},
		{
			method: "POST",
			path:   "/v1.0/logs/uploadcomplete",
			// swagger:route POST /v1.0/logs/uploadcomplete Log LogUploadCompleteV2
			//
			// Report log upload complete.  ntnx:ignore
			//
			// Log upload complete as reported by an edge.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       default: APIError
			handle: UploadLogCompleteHandle,
		},
	}
}

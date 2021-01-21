package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
)

const (
	downloadExpiryMins     = 15
	uploadExpiryMins       = 45
	logFileExt             = "tgz"
	logVersion             = "v1"
	updatePendingLogsJobID = "update-pending-logs"
	entityTypeSupportLog   = "supportlog"
)

func init() {
	// : is the SQL escape character. The actual query passed has tags::jsonb (type cast)
	// There are 3 types of entries based on tags - all, tagged with [] for infra logs, tagged with [{"name": "Application", "value": app-id}] for application logs.
	// Empty array [] is always contained (@>) in a non-empty array. Hence, the array length check is required for array containment check
	queryMap["SelectLogsTemplate"] = `SELECT * FROM log_model WHERE  tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (:location = '' OR location = :location) AND (:edge_id = '' OR edge_id = :edge_id) AND (:ignore_tags = true OR (jsonb_array_length(:tags) = 0 AND jsonb_array_length(tags::::jsonb) = 0) OR (jsonb_array_length(:tags) > 0 AND jsonb_array_length(tags::::jsonb) > 0 AND (tags::::jsonb @> ANY (ARRAY [:tags]::::jsonb[])))) %s`
	queryMap["SelectLogs"] = `SELECT * FROM log_model WHERE  tenant_id = :tenant_id AND (:id = '' OR id = :id) AND (:location = '' OR location = :location) AND (:status = '' OR status = :status)`
	queryMap["CreateLog"] = `INSERT INTO log_model (id, version, tenant_id, batch_id, edge_id, location, tags, status, error_message, created_at, updated_at) VALUES (:id, :version, :tenant_id, :batch_id, :edge_id, :location, :tags, :status, :error_message, :created_at, :updated_at)`
	queryMap["UpdateLog"] = `UPDATE log_model SET version = :version, status = :status, error_message = :error_message, updated_at = :updated_at WHERE tenant_id = :tenant_id AND location = :location`

	queryMap["ScanLogs"] = `SELECT * FROM log_model WHERE status = :status`
	queryMap["UpdatePendingLogs"] = fmt.Sprintf(`UPDATE log_model SET version = :version, status = :status, error_message = :error_message, updated_at = :updated_at WHERE tenant_id = :tenant_id AND status = '%s' AND id = :id`, model.LogUploadPending)

	orderByHelper.Setup(entityTypeSupportLog, []string{"id", "created_at", "updated_at", "batch_id", "edge_id", "location", "status"})
}

// LogDBO is the DB object model for log record
type LogDBO struct {
	model.EdgeBaseModelDBO
	// id that identifies logs from different edge as the same batch.
	BatchID string `json:"batchId" db:"batch_id"`
	// Location or object key for the log in the bucket.
	Location string `json:"location" db:"location"`
	// Tags on this log entry.
	Tags *types.JSONText `json:"tags,omitempty" db:"tags"`
	// Status of this log entry.
	Status string `json:"status" db:"status"`
	// Error message - optional, should be populated when status == 'FAILED'
	ErrorMessage *string `json:"errorMessage,omitempty" db:"error_message"`
}

// LogDBOQueryParam is an extension of LogDBO model to support ignoring of tags column
type LogDBOQueryParam struct {
	LogDBO
	IgnoreTags bool `json:"_" db:"ignore_tags"`
}

// ExtractEdgeID extracts edge ID from the URL
// https://bucket.s3.us-west-2.amazonaws.com/v1/tenantId/2018/04/18/batchId/edge/edge-batchId.tgz?AWSAccessKeyId=...
// or http://minio-service:9000/sherlock-support-bundle-us-west-2/v1/tid-pmp-test-1/2020/12/10/batchId/edge/edge-batchId.tgz?
func ExtractEdgeID(URL string) (string, error) {
	maxIDLen := 36
	ba := make([]byte, maxIDLen)
	// Other following slashes if any will be URL encoded
	idx := strings.LastIndex(URL, "/") - 1
	fillIdx := maxIDLen - 1
	for idx > 0 {
		if URL[idx] == '/' {
			// This check allows // but it is not supposed to happen
			if fillIdx < maxIDLen-1 {
				return string(ba[fillIdx+1 : maxIDLen]), nil
			}
		} else if fillIdx < 0 {
			break
		} else {
			// Fill from the end
			ba[fillIdx] = URL[idx]
			fillIdx--
		}
		idx--
	}
	glog.Errorf("Invalid URL %s", URL)
	return "", errcode.NewBadRequestError("URL")
}

func createS3Key(version string, tenantID string, batchID string, edgeID string) string {
	t := time.Now().UTC()
	return fmt.Sprintf("%s/%s/%d/%02d/%02d/%s/%s/%s-%s.%s",
		version, tenantID, t.Year(), t.Month(), t.Day(), batchID, edgeID, edgeID, batchID, logFileExt)
}

// ExtractLogLocation extracts the location 1/tenantId/2018/04/18/batchId/edge/edge-batchId.tgz from the URLs -
// https://bucket.s3.us-west-2.amazonaws.com/v1/tenantId/2018/04/18/batchId/edge/edge-batchId.tgz?AWSAccessKeyId=...
// or // or http://minio-service:9000/sherlock-support-bundle-us-west-2/v1/tid-pmp-test-1/2020/12/10/batchId/edge/edge-batchId.tgz?
func ExtractLogLocation(URL string) (string, error) {
	// Other following slashes if any will be URL encoded
	idx := strings.LastIndex(URL, "/")
	if idx > 0 {
		// Right side of last /
		fileName := URL[idx+1:]
		fileIdx := strings.Index(fileName, "?")
		if fileIdx > 0 {
			// v1/tenantId/2018/04/18/batchId/edge/edge-batchId.tgz
			maxCompLen := 8
			fillIdx := maxCompLen - 1
			components := make([]string, maxCompLen)
			components[fillIdx] = fileName[0:fileIdx]
			fillIdx--
			subURL := URL
			for fillIdx >= 0 {
				subURL = subURL[0:idx]
				idx = strings.LastIndex(subURL, "/")
				if idx < 0 {
					break
				}
				if idx < len(subURL)-1 {
					components[fillIdx] = subURL[idx+1:]
					fillIdx--
				}
			}
			if fillIdx < 0 {
				return strings.Join(components, "/"), nil
			}
		}
	}
	glog.Errorf("Invalid URL %s", URL)
	return "", errcode.NewBadRequestError("URL")
}

func logDBOAuthFilterPredicate(authContext *base.AuthContext, logDBO *LogDBO) bool {
	logEntry := &model.LogEntry{}
	err := base.Convert(logDBO, logEntry)
	if err != nil {
		return false
	}
	return logEntryAuthFilterPredicate(authContext, logEntry)
}

// There are not many log entries as we clean up every 15 days.
// This needs to be enhanced for better RBAC.
func logEntryAuthFilterPredicate(authContext *base.AuthContext, logEntry *model.LogEntry) bool {
	if !auth.IsInfraAdminRole(authContext) {
		for _, tag := range logEntry.Tags {
			if tag.Name == model.ApplicationLogTag {
				// User is not infra admin but it is application log
				return true
			}
		}
		// Not an application log and user is not infra admin
		return false
	}
	// user is infra admin
	return true
}

func applyPagination(logEntries []model.LogEntry, entitiesQueryParam *model.EntitiesQueryParam) model.LogEntriesListPayload {
	size := len(logEntries)
	if entitiesQueryParam != nil {
		startIndex := entitiesQueryParam.PageIndex * entitiesQueryParam.PageSize
		if startIndex < size {
			endIndex := startIndex + entitiesQueryParam.PageSize
			if endIndex > size {
				endIndex = size
			}
			logEntries = logEntries[startIndex:endIndex]
		} else {
			logEntries = []model.LogEntry{}
		}
	}
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: size, EntityType: entityTypeSupportLog})
	return model.LogEntriesListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		LogEntryList:              logEntries,
	}
}

func getLogUploadURL(context context.Context, edgeID string, batchID string) (string, string, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return "", "", err
	}
	tenantID := authContext.TenantID
	s3Client := s3.New(awsSession)
	key := createS3Key(logVersion, tenantID, batchID, edgeID)
	input := s3.PutObjectInput{
		Bucket:      config.Cfg.LogS3Bucket,
		Key:         aws.String(key),
		ACL:         aws.String("bucket-owner-full-control"),
		ContentType: aws.String("application/x-gzip"),
	}
	req, _ := s3Client.PutObjectRequest(&input)
	url, headers, err := req.PresignRequest(uploadExpiryMins * time.Minute)
	glog.Infof(base.PrefixRequestID(context, "Signed with header %s for tenant ID %s, edge ID %s and batch ID %s"), headers, tenantID, edgeID, batchID)
	if err != nil {
		return "", key, errcode.NewInternalError(err.Error())
	}
	if glog.V(4) {
		glog.V(4).Infof(base.PrefixRequestID(context, "Generated log upload URL %s"), url)
	}
	return url, key, err
}

// SelectAllLogs returns log entries for the optional edgeID (all for empty) and the tags.
// If the tags is nil, it returns all the log entries. It is empty, only the infra/edge logs are returned.
// If it is non-empty, entries with the matching tags are returned.
func (dbAPI *dbObjectModelAPI) SelectAllLogs(context context.Context, edgeID string, tags []model.LogTag, entitiesQueryParam *model.EntitiesQueryParam) ([]model.LogEntry, error) {
	logEntries := []model.LogEntry{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return logEntries, err
	}
	query, err := buildQuery(entityTypeSupportLog, queryMap["SelectLogsTemplate"], entitiesQueryParam, orderByUpdatedAt)
	if err != nil {
		return logEntries, err
	}
	tenantID := authContext.TenantID
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel, EdgeID: strings.TrimSpace(edgeID)}
	param := LogDBOQueryParam{LogDBO: LogDBO{EdgeBaseModelDBO: edgeModel}}
	if tags == nil {
		param.IgnoreTags = true
	} else {
		data, err := json.Marshal(tags)
		if err != nil {
			glog.Error(base.PrefixRequestID(context, "Error in SelectAllLogs. Error: %s"), err.Error())
			return nil, errcode.NewBadRequestError("tags")
		}
		jsonText := types.JSONText(data)
		param.Tags = &jsonText
	}
	_, err = dbAPI.NotPagedQuery(context, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		logEntry := model.LogEntry{}
		err := base.Convert(dbObjPtr, &logEntry)
		if err != nil {
			glog.Error(base.PrefixRequestID(context, "Error in SelectAllLogs. Error: %s"), err.Error())
			return err
		}
		if logEntryAuthFilterPredicate(authContext, &logEntry) {
			logEntries = append(logEntries, logEntry)
		}
		return nil
	}, query, param)
	return logEntries, err
}

func (dbAPI *dbObjectModelAPI) SelectAllLogsW(context context.Context, w io.Writer, req *http.Request) error {
	logEntries, err := dbAPI.SelectAllLogs(context, "", nil, nil)
	if err != nil {
		glog.Error(base.PrefixRequestID(context, "Error in SelectAllLogsW. Error: %s"), err.Error())
		return err
	}
	return base.DispatchPayload(w, logEntries)
}

func (dbAPI *dbObjectModelAPI) SelectAllLogsWV2(context context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	logEntries, err := dbAPI.SelectAllLogs(context, "", nil, entitiesQueryParam)
	if err != nil {
		glog.Error(base.PrefixRequestID(context, "Error in SelectAllLogsWV2. Error: %s"), err.Error())
		return err
	}
	return json.NewEncoder(w).Encode(applyPagination(logEntries, entitiesQueryParam))
}

func (dbAPI *dbObjectModelAPI) SelectAllEdgeLogsWV2(context context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	logEntries, err := dbAPI.SelectAllLogs(context, "", []model.LogTag{}, entitiesQueryParam)
	if err != nil {
		glog.Error(base.PrefixRequestID(context, "Error in SelectAllEdgeLogsWV2. Error: %s"), err.Error())
		return err
	}
	return json.NewEncoder(w).Encode(applyPagination(logEntries, entitiesQueryParam))
}

func (dbAPI *dbObjectModelAPI) SelectAllApplicationLogsWV2(context context.Context, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	logEntries, err := dbAPI.SelectAllLogs(context, "", []model.LogTag{model.LogTag{Name: model.ApplicationLogTag}}, entitiesQueryParam)
	if err != nil {
		glog.Error(base.PrefixRequestID(context, "Error in SelectAllApplicationLogsWV2. Error: %s"), err.Error())
		return err
	}
	return json.NewEncoder(w).Encode(applyPagination(logEntries, entitiesQueryParam))
}

func (dbAPI *dbObjectModelAPI) GetEdgeLogsWV2(context context.Context, edgeID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	logEntries, err := dbAPI.SelectAllLogs(context, edgeID, []model.LogTag{}, entitiesQueryParam)
	if err != nil {
		glog.Error(base.PrefixRequestID(context, "Error in GetEdgeLogsWV2. Error: %s"), err.Error())
		return err
	}
	return json.NewEncoder(w).Encode(applyPagination(logEntries, entitiesQueryParam))
}

func (dbAPI *dbObjectModelAPI) GetApplicationLogsWV2(context context.Context, applicationID string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	logEntries, err := dbAPI.SelectAllLogs(context, "", []model.LogTag{model.LogTag{Name: model.ApplicationLogTag, Value: applicationID}}, entitiesQueryParam)
	if err != nil {
		glog.Error(base.PrefixRequestID(context, "Error in GetApplicationLogsWV2. Error: %s"), err.Error())
		return err
	}
	return json.NewEncoder(w).Encode(applyPagination(logEntries, entitiesQueryParam))
}

func (dbAPI *dbObjectModelAPI) DeleteLogEntry(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	var doc model.LogEntry
	logDBOs := []LogDBO{}
	tenantModel := model.BaseModelDBO{TenantID: tenantID, ID: id}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	param := LogDBOQueryParam{LogDBO: LogDBO{EdgeBaseModelDBO: edgeModel}, IgnoreTags: true}
	err = dbAPI.Query(context, &logDBOs, queryMap["SelectLogs"], param)
	if err != nil {
		return resp, err
	}
	if len(logDBOs) == 0 {
		return resp, nil
	}

	if !logDBOAuthFilterPredicate(authContext, &logDBOs[0]) {
		glog.Error(base.PrefixRequestID(context, "Permission error in DeleteLogEntry"))
		return resp, errcode.NewPermissionDeniedError("RBAC")
	}
	result, err := dbAPI.Delete(context, "log_model", map[string]interface{}{"id": logDBOs[0].ID})
	if err != nil {
		return resp, err
	}
	s3Client := s3.New(awsSession)
	input := s3.DeleteObjectInput{Bucket: config.Cfg.LogS3Bucket, Key: aws.String(logDBOs[0].Location)}
	_, err = s3Client.DeleteObject(&input)
	if err != nil {
		return resp, errcode.NewInternalError(err.Error())
	}
	if base.IsDeleteSuccessful(result) {
		resp.ID = id
		if callback != nil {
			go callback(context, doc)
		}
	}
	return resp, nil
}

func (dbAPI *dbObjectModelAPI) DeleteLogEntryW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(context, dbAPI.DeleteLogEntry, id, w, callback)
}

func (dbAPI *dbObjectModelAPI) RequestLogDownload(context context.Context, payload model.RequestLogDownloadPayload) (string, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return "", err
	}
	tenantID := authContext.TenantID
	logDBOs := []LogDBO{}
	tenantModel := model.BaseModelDBO{TenantID: tenantID}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	param := LogDBOQueryParam{LogDBO: LogDBO{EdgeBaseModelDBO: edgeModel, Status: model.LogUploadSuccess, Location: payload.Location}, IgnoreTags: true}
	err = dbAPI.Query(context, &logDBOs, queryMap["SelectLogs"], param)
	if err != nil {
		return "", err
	}
	if len(logDBOs) == 0 {
		return "", errcode.NewRecordNotFoundError(payload.Location)
	}
	if !logDBOAuthFilterPredicate(authContext, &logDBOs[0]) {
		glog.Error(base.PrefixRequestID(context, "Permission error in RequestLogDownload"))
		return "", errcode.NewPermissionDeniedError("RBAC")
	}
	s3Client := s3.New(awsSession)
	input := s3.GetObjectInput{
		Bucket: config.Cfg.LogS3Bucket,
		Key:    aws.String(payload.Location),
	}
	request, _ := s3Client.GetObjectRequest(&input)
	url, err := request.Presign(downloadExpiryMins * time.Minute)
	glog.Infof(base.PrefixRequestID(context, "Signed key %s for tenant ID %s"), payload.Location, tenantID)
	if err != nil {
		return url, errcode.NewInternalError(err.Error())
	}
	return url, err
}

func (dbAPI *dbObjectModelAPI) RequestLogDownloadW(context context.Context, w io.Writer, r io.Reader) error {
	doc := model.RequestLogDownloadPayload{}
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into log. Error: %s"), err.Error())
		return err
	}
	resp, err := dbAPI.RequestLogDownload(context, doc)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(resp))
	return err
}

func (dbAPI *dbObjectModelAPI) UploadLog(context context.Context, payload model.LogUploadPayload) error {
	return errors.New("Unimplemented")
}

func (dbAPI *dbObjectModelAPI) UploadLogW(context context.Context, r io.Reader) error {
	return errors.New("Unimplemented")
}

// RequestLogUpload is more for testing
func (dbAPI *dbObjectModelAPI) RequestLogUpload(ctx context.Context, payload model.RequestLogUploadPayload, callback func(context.Context, interface{}) error) ([]model.LogUploadPayload, error) {
	resp := []model.LogUploadPayload{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	errMsgs := []string{}
	wg := &sync.WaitGroup{}
	mutex := &sync.Mutex{}
	now := base.RoundedNow()
	batchID := base.GetUUID()
	version := float64(now.UnixNano())
	// Callback wrapper to the input callback
	callbackWrapper := func(uploadPayload *model.LogUploadPayload, logDBO *LogDBO, callback func(context.Context, interface{}) error) {
		err := callback(ctx, uploadPayload)
		if err != nil {
			errMsg := err.Error()
			logDBO.Status = model.LogUploadFailed
			logDBO.ErrorMessage = &errMsg
			// Ignore error
			dbAPI.NamedExec(ctx, queryMap["UpdateLog"], &logDBO)
		}
	}
	for _, edgeID := range payload.EdgeIDs {
		wg.Add(1)
		// Asynchronously update the DB
		go func(edgeID string) {
			defer wg.Done()
			url, key, err := getLogUploadURL(ctx, edgeID, batchID)
			errMsg := ""
			if err == nil {
				tenantModel := model.BaseModelDBO{TenantID: tenantID, Version: version, ID: base.GetUUID(), CreatedAt: now, UpdatedAt: now}
				edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel, EdgeID: edgeID}
				tags := []model.LogTag{}
				// Check if application is running in case of application logs.
				if len(payload.ApplicationID) > 0 {
					// Lookup the application in the db to see if it exists and is in deployed state
					app, dbErr := dbAPI.GetApplication(ctx, payload.ApplicationID)
					if dbErr == nil {
						if app.ApplicationCore.State == nil || (*app.ApplicationCore.State == string(model.DeployEntityState)) {
							tags = append(tags, model.LogTag{Name: model.ApplicationLogTag, Value: payload.ApplicationID})
						} else {
							errMsg = fmt.Sprintf("Application '%s' is in stopped state. Cannot fetch logs.", payload.ApplicationID)
							glog.Errorf(errMsg)
						}
					} else {
						errMsg = fmt.Sprintf("Error fetching application '%s'", payload.ApplicationID)
						glog.Errorf(fmt.Sprintf(base.PrefixRequestID(ctx, "%s from db: %s"), errMsg, dbErr.Error()))
					}
				}

				// Check permissions to get logs.
				if errMsg == "" && !logEntryAuthFilterPredicate(authContext, &model.LogEntry{Tags: tags}) {
					glog.Errorf(base.PrefixRequestID(ctx, "Permission denied in RequestLogUpload for edge %s"), edgeID)
					err = errcode.NewPermissionDeniedError("RBAC")
				}

				if errMsg == "" && err == nil {
					var bytes []byte
					bytes, err = json.Marshal(&tags)
					if err == nil {
						_, err = dbAPI.GetServiceDomain(ctx, edgeID)
						if err == nil {
							tagsJSON := types.JSONText(bytes)
							logDBO := LogDBO{EdgeBaseModelDBO: edgeModel, BatchID: batchID, Location: key, Tags: &tagsJSON, Status: model.LogUploadPending}
							_, err = dbAPI.NamedExec(ctx, queryMap["CreateLog"], &logDBO)
							if err == nil {
								uploadPayload := model.LogUploadPayload{
									URL:           url,
									ApplicationID: payload.ApplicationID,
									BatchID:       batchID,
								}
								mutex.Lock()
								resp = append(resp, uploadPayload)
								mutex.Unlock()
								if callback != nil {
									// Asynchronously send the websocket
									go callbackWrapper(&uploadPayload, &logDBO, callback)
								}
							} else {
								err = errcode.TranslateDatabaseError("log", err)
								glog.Errorf(base.PrefixRequestID(ctx, "Error in creating log for tenant ID %s. Error: %s"), tenantID, err.Error())
								// clear err, replace with user friendly errMsg
								errMsg = fmt.Sprintf("Error creating log entry for edge '%s'", edgeID)
								err = nil
							}
						} else {
							errMsg = fmt.Sprintf("Error fetching edge '%s'", edgeID)
						}
					} else {
						err = errcode.NewBadRequestError("log")
					}
				}
			}
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in creating log for tenant ID %s. Error: %s"), tenantID, err.Error())
				mutex.Lock()
				errMsgs = append(errMsgs, err.Error())
				mutex.Unlock()
			}
			if errMsg != "" {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in creating log for tenant ID %s. Error: %s"), tenantID, errMsg)
				mutex.Lock()
				errMsgs = append(errMsgs, errMsg)
				mutex.Unlock()
			}
		}(edgeID)
	}
	wg.Wait()
	if len(errMsgs) != 0 {
		err = errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return resp, err
}

func (dbAPI *dbObjectModelAPI) RequestLogUploadW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	doc := model.RequestLogUploadPayload{}
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into log. Error: %s"), err.Error())
		return err
	}
	resp, err := dbAPI.RequestLogUpload(context, doc, callback)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

func (dbAPI *dbObjectModelAPI) UploadLogComplete(context context.Context, payload model.LogUploadCompletePayload) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	isEdgeReq, edgeID := base.IsEdgeRequest(authContext)
	// Only an edge must be able to call this API
	if !isEdgeReq {
		return errcode.NewPermissionDeniedError("role")
	}
	if !strings.Contains(payload.URL, edgeID) {
		return errcode.NewPermissionDeniedError("URL")
	}
	tenantID := authContext.TenantID
	location, err := ExtractLogLocation(payload.URL)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error extracting location from URL %s. Error: %s"), payload.URL, err.Error())
		return err
	}
	now := base.RoundedNow()
	version := float64(now.UnixNano())
	tenantModel := model.BaseModelDBO{TenantID: tenantID, Version: version, UpdatedAt: now}
	edgeModel := model.EdgeBaseModelDBO{BaseModelDBO: tenantModel}
	logDBO := LogDBO{EdgeBaseModelDBO: edgeModel, Location: location, Status: string(payload.Status), ErrorMessage: &payload.ErrorMessage}
	glog.V(3).Infof(base.PrefixRequestID(context, "UploadLogComplete: Updating entry status %+v"), logDBO)
	_, err = dbAPI.NamedExec(context, queryMap["UpdateLog"], &logDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error in updating log for URL %s and tenant ID %s. Error: %s"), payload.URL, tenantID, err.Error())
		return errcode.TranslateDatabaseError(location, err)
	}
	return err
}

func (dbAPI *dbObjectModelAPI) UploadLogCompleteW(context context.Context, r io.Reader) error {
	payload := model.ObjectResponseLogUploadComplete{}
	err := base.Decode(&r, &payload)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into log. Error: %s"), err.Error())
		return err
	}
	return dbAPI.UploadLogComplete(context, payload.Doc)
}

// ScheduleTimeOutPendingLogsJob schedules a job to time out pending logs
// It can reschedule if it missed some pending log records.
func (dbAPI *dbObjectModelAPI) ScheduleTimeOutPendingLogsJob(ctx context.Context, delay time.Duration, timeout time.Duration) error {
	// Execute the job after 5 mins and update if state has been pending for over 20 mins.
	base.ScheduleJob(ctx, updatePendingLogsJobID, func(ctx context.Context) {
		pending, err := dbAPI.scanAndTimeOutPendingLogs(ctx, timeout)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to scan and update pending logs. Error: %s"), err.Error())
		}
		if pending {
			dbAPI.ScheduleTimeOutPendingLogsJob(ctx, delay, timeout)
		}
	}, delay)
	// No error
	return nil
}

// scanAndTimeOutPendingLogs scans and updates old pending logs. It returns true if
// some pending ones got missed out in the current iteration.
func (dbAPI *dbObjectModelAPI) scanAndTimeOutPendingLogs(ctx context.Context, timeout time.Duration) (bool, error) {
	pending := false
	startPageToken := base.StartPageToken
	param := LogDBO{Status: model.LogUploadPending}
	errMsgs := []string{}
	glog.V(4).Infof("Scanning pending logs to delete...")
	for {
		// Process in batches of 30 each
		nextToken, err := dbAPI.PagedQuery(ctx, startPageToken, 30, func(dbObjPtr interface{}) error {
			logDBO := dbObjPtr.(*LogDBO)
			elapsedTime := time.Since(logDBO.UpdatedAt)
			if elapsedTime > timeout {
				now := base.RoundedNow()
				logDBO.Version = float64(now.UnixNano())
				logDBO.Status = model.LogUploadFailed
				logDBO.ErrorMessage = base.StringPtr(fmt.Sprintf("Timed out after %f mins", elapsedTime.Minutes()))
				logDBO.UpdatedAt = now
				_, err := dbAPI.NamedExec(ctx, queryMap["UpdatePendingLogs"], logDBO)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to update pending logs. Error: %s"), err.Error())
					errMsgs = append(errMsgs, err.Error())
					// Some states could not be updated, try scheduling again
					pending = true
				}
			} else {
				// Encountered some in pending state which cannot be updated now
				pending = true
			}
			return nil

		}, queryMap["ScanLogs"], param)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating pending logs. Error: %s"), err.Error())
		}
		if nextToken == base.NilPageToken {
			break
		}
		startPageToken = nextToken
	}
	if len(errMsgs) > 0 {
		return pending, errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return pending, nil
}

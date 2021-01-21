package api

import (
	"bytes"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/golang/glog"
)

const entityTypeAuditLog = "auditlog"

var createAuditLogTableTemplate *template.Template

func init() {

	// note: tenant_id column does not contain foreign key on delete cascade constraint
	// this is intentional - we may have reason to keep audit log for a while
	// after tenant deletion.
	// We will provide script to clean such audit log entries on demand.
	// Unit tests that create and delete tenants should clean up
	// audit log entries for such tenants (this is done automatically inside dbAPI.DeleteTenant)
	queryMap["CreateAuditLogTableTemplate"] = `{{with .TableName -}}
	create table if not exists {{.}} (
    tenant_id varchar(36) not null,
    user_email varchar(200) not null,
    edge_ids varchar(8192),
    hostname varchar(64) not null,
    request_id varchar(36) not null,
    request_method varchar(16) not null,
    request_url varchar(200) not null,
    request_payload varchar(1024),
    request_header varchar(8192),
    response_code int not null,
		response_message varchar(1024),
		response_length int not null,
    time_ms real not null,
    started_at timestamp not null,
		created_at timestamp not null);
		create index if not exists {{.}}_tenant on {{.}} (tenant_id);
		create index if not exists {{.}}_request on {{.}} (request_id);
		create index if not exists {{.}}_started on {{.}} (started_at);
		create index if not exists {{.}}_email on {{.}} (user_email);
		create index if not exists {{.}}_hostname on {{.}} (hostname);
		create index if not exists {{.}}_method on {{.}} (request_method);
		create index if not exists {{.}}_url on {{.}} (request_url);
		create index if not exists {{.}}_response_code on {{.}} (response_code);
		create index if not exists {{.}}_response_length on {{.}} (response_length);
		create index if not exists {{.}}_response_message on {{.}} (response_message);
		create index if not exists {{.}}_time on {{.}} (time_ms);
		create index if not exists {{.}}_created on {{.}} (created_at);
		create table if not exists audit_log_from_request_id (
			tenant_id varchar(36) not null,
			request_id varchar(36) not null,
			table_name varchar(20) not null);
		create index if not exists audit_log_from_request_id_tenant on audit_log_from_request_id (tenant_id);
		create index if not exists audit_log_from_request_id_request on audit_log_from_request_id (request_id);
		create index if not exists audit_log_from_request_id_table on audit_log_from_request_id (table_name);
		{{end}}`

	queryMap["CreateAuditLogTemplate"] = `INSERT INTO %s (tenant_id, user_email, edge_ids, hostname, request_id, request_method, request_url, request_payload, request_header, response_code, response_message, response_length, time_ms, started_at, created_at) VALUES (:tenant_id, :user_email, :edge_ids, :hostname, :request_id, :request_method, :request_url, :request_payload, :request_header, :response_code, :response_message, :response_length, :time_ms, :started_at, :created_at)`
	queryMap["GetAuditLogByRequestIDTemplate"] = `SELECT * from %s WHERE request_id = :request_id`
	queryMap["CreateAuditLogReqIDToTableEntry"] = `INSERT INTO audit_log_from_request_id (tenant_id, request_id, table_name) VALUES (:tenant_id, :request_id, :table_name)`
	queryMap["GetAuditLogTableNameForRequestID"] = `SELECT table_name from audit_log_from_request_id WHERE tenant_id = :tenant_id AND request_id = :request_id`
	queryMap["SelectAllAuditLogsTemplate"] = `SELECT *, count(*) OVER() as total_count from %s WHERE tenant_id = :tenant_id and started_at > :start and started_at < :end %s`
	// to clean tenant audit logs: use audit_log_from_request_id
	// to find all audit log tables containing entries for the tenant,
	// then do delete on each table
	queryMap["GetTenantAuditLogTables"] = `SELECT table_name from audit_log_from_request_id WHERE tenant_id = :tenant_id group by table_name`
	queryMap["DeleteTenantAuditLogTemplate"] = `DELETE FROM %s WHERE tenant_id = :tenant_id`
	queryMap["DeleteTenantAuditLogReqToTableMap"] = `DELETE FROM audit_log_from_request_id WHERE tenant_id = :tenant_id`

	queryMap["DeleteOldTableFromAuditLogReqTableTemplate"] = `delete from audit_log_from_request_id where table_name IN (select table_name from information_schema.tables where table_name like 'audit_log_%%' and table_name != 'audit_log_from_request_id' order by table_name desc offset %d)`
	queryMap["SelectOldAuditLogTableNameTemplate"] = `select table_name from information_schema.tables where table_name like 'audit_log_%%' and table_name != 'audit_log_from_request_id' order by table_name desc offset %d`
	createAuditLogTableTemplate = template.Must(template.New("stmt").Parse(queryMap["CreateAuditLogTableTemplate"]))
	orderByHelper.Setup(entityTypeAuditLog, []string{"user_email", "hostname", "request_id", "request_method", "request_url", "request_payload", "request_header", "response_code", "response_message", "response_length", "time_ms", "created_at", "started_at"})
}

// GetAuditLogTableName get audit log table name based on current date
// audit log table name is of form: audit_log_<yyyymmdd>
func GetAuditLogTableName() string {
	return fmt.Sprintf("audit_log_%s", base.GetDateString())
}

type tableNameStruct struct {
	TableName string
}

// createAuditLogTable create audit log table with the given name if it does not exist
func (dbAPI *dbObjectModelAPI) createAuditLogTable(tableName string) error {
	glog.V(4).Infof("createAuditLogTable: tableName=%s\n", tableName)
	db := dbAPI.GetDB()
	var w bytes.Buffer
	r := tableNameStruct{
		TableName: tableName,
	}
	err := createAuditLogTableTemplate.Execute(&w, r)
	if err == nil {
		_, err = db.Exec(w.String())
	}
	return err
}

// struct for data stored in audit_log_from_request_id table
type auditLogReqIDToTableName struct {
	TenantID  string `db:"tenant_id"`
	RequestID string `db:"request_id"`
	TableName string `db:"table_name"`
}

// writeAuditLog write the audit log entry into the given audit log DB table
// assumes table exist (will not create the table)
// also write mapping of request id to table name into audit_log_from_request_id table
func (dbAPI *dbObjectModelAPI) writeAuditLog(ctx context.Context, auditLog *model.AuditLog, tableName string) error {
	stmt := fmt.Sprintf(queryMap["CreateAuditLogTemplate"], tableName)
	_, err := dbAPI.NamedExec(ctx, stmt, auditLog)
	if err == nil {
		// insert entry into audit_log_from_request_id table, ignore error
		if auditLog.TenantID != "" && auditLog.RequestID != "" {
			_, err2 := dbAPI.NamedExec(ctx, queryMap["CreateAuditLogReqIDToTableEntry"], &auditLogReqIDToTableName{TenantID: auditLog.TenantID, RequestID: auditLog.RequestID, TableName: tableName})
			if err2 != nil {
				glog.Warningf("writeAuditLog: fail to write to audit_log_from_request_id table: %s\n", err2.Error())
			}
		}
	}
	return err
}

var reNoTable = regexp.MustCompile(`relation\s+\\"audit_log_\d+\\"\s+does not exist`)

// WriteAuditLog - write audit log entry into table
// internal API, not exposed via REST endpoint
// Will fill in audit log entry with current time and
// use audit log table name based on current time
// Will create the audit log table if it does not exist yet.
// Note: ctx may not have full authContext, only reqID and TenantID
// are guaranteed to be present and match those from auditLog
func (dbAPI *dbObjectModelAPI) WriteAuditLog(ctx context.Context, auditLog *model.AuditLog) error {
	if false == *config.Cfg.DisableAuditLog {
		// first fill in create time and elapsed time
		auditLog.FillInTime()
		auditLog.RequestPayload = base.TruncateStringMaybe(auditLog.RequestPayload, model.LogPayloadMaxLength)
		auditLog.ResponseMessage = base.TruncateStringMaybe(auditLog.ResponseMessage, model.LogResponseMaxLength)
		auditLog.RequestHeader = base.TruncateStringMaybe(auditLog.RequestHeader, model.LogHeaderMaxLength)

		tableName := base.GetAuditLogTableName(ctx)
		if tableName == "" {
			return errcode.NewInternalError("Missing audit log table name")
		}
		// the first log entry for each day should create the audit log table for that day
		err := dbAPI.writeAuditLog(ctx, auditLog, tableName)

		if err != nil {
			glog.V(4).Infof("write audit log failed, error: %s\n", err.Error())
		}

		if err != nil && reNoTable.MatchString(err.Error()) {
			// create table
			// note: err2 is ignored since error likely due to race, i.e.,
			// table created by other concurrent WriteAuditLog
			err2 := dbAPI.createAuditLogTable(tableName)
			if err2 == nil {
				// if WriteAuditLog successfully created audit log table for new day,
				// also do clean up of old audit log tables
				errs := dbAPI.dropOldAuditLogTables(ctx, *config.Cfg.KeepAuditLogTableDays)
				if len(errs) != 0 {
					// log error
					errStrs := []string{}
					for _, err := range errs {
						errStrs = append(errStrs, err.Error())
					}
					glog.Errorf(base.PrefixRequestID(ctx, "Error in dropping old audit log tables. Error: %s"), strings.Join(errStrs, "; "))
				}
			}
			// try again regardless of create table status, in case of race
			err = dbAPI.writeAuditLog(ctx, auditLog, tableName)
		}
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating audit log for tenant ID %s. Error: %s"), auditLog.TenantID, err.Error())
			return errcode.TranslateDatabaseError(auditLog.RequestID, err)
		}
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) dropOldAuditLogTables(ctx context.Context, keepCount int) []error {
	errs := []error{}
	if keepCount > 1 {
		db := dbAPI.GetDB()
		// first delete all req id -> table name mapping entries
		// for old tables from audit_log_from_request_id table
		stmt := fmt.Sprintf(queryMap["DeleteOldTableFromAuditLogReqTableTemplate"], keepCount)
		_, err := db.Exec(stmt)
		if err != nil {
			errs = append(errs, err)
		}
		// next drop all old audit log tables
		tableNameStructs := []auditLogReqIDToTableName{}
		query := fmt.Sprintf(queryMap["SelectOldAuditLogTableNameTemplate"], keepCount)
		param := make(map[string]interface{})
		err = dbAPI.Query(ctx, &tableNameStructs, query, param)
		if err != nil {
			errs = append(errs, err)
		} else if len(tableNameStructs) > 0 {
			tableNames := []string{}
			for _, tableNameStruct := range tableNameStructs {
				tableNames = append(tableNames, tableNameStruct.TableName)
			}
			stmt2 := fmt.Sprintf("drop table %s", strings.Join(tableNames, ", "))
			_, err = db.Exec(stmt2)
			if err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errs
}

type auditLogGetQueryParam struct {
	TenantID  string `db:"tenant_id"`
	RequestID string `db:"request_id"`
}
type auditLogListQueryParam struct {
	TenantID string `db:"tenant_id"`
	Start    string `db:"start"`
	End      string `db:"end"`
}

// GetAuditLog get all audit log entries with the given request id
// A given request id may have multiple audit log entries
// (e.g., CUD REST API will have api entry + websocket entries)
func (dbAPI *dbObjectModelAPI) GetAuditLog(ctx context.Context, reqID string) ([]model.AuditLog, error) {
	if reqID == "" {
		return []model.AuditLog{}, errcode.NewBadRequestError("id")
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return []model.AuditLog{}, err
	}
	if authContext.TenantID == "" {
		return []model.AuditLog{}, errcode.NewBadRequestError("tenantID")
	}
	if !auth.IsInfraAdminRole(authContext) {
		return []model.AuditLog{}, errcode.NewPermissionDeniedError("RBAC/AuditLog")
	}
	tableNameInfoParam := auditLogReqIDToTableName{TenantID: authContext.TenantID, RequestID: reqID}
	tableNameInfoResult := []auditLogReqIDToTableName{}
	err = dbAPI.Query(ctx, &tableNameInfoResult, queryMap["GetAuditLogTableNameForRequestID"], tableNameInfoParam)
	if err != nil {
		return []model.AuditLog{}, err
	}
	if len(tableNameInfoResult) == 0 {
		return []model.AuditLog{}, errcode.NewRecordNotFoundError(reqID)
	}
	query := fmt.Sprintf(queryMap["GetAuditLogByRequestIDTemplate"], tableNameInfoResult[0].TableName)
	auditLogs := []model.AuditLog{}
	err = dbAPI.Query(ctx, &auditLogs, query, auditLogGetQueryParam{TenantID: authContext.TenantID, RequestID: reqID})
	if err != nil {
		return []model.AuditLog{}, err
	}
	return auditLogs, nil
}

// GetAuditLogW wrapper of GetAuditLog used by REST API
func (dbAPI *dbObjectModelAPI) GetAuditLogW(ctx context.Context, reqID string, w io.Writer, r *http.Request) error {
	auditLogs, err := dbAPI.GetAuditLog(ctx, reqID)
	if err == nil {
		err = json.NewEncoder(w).Encode(auditLogs)
	}
	return err
}

// auditLogDBO wrapper of AuditLog to capture total_count (for paging support)
type auditLogDBO struct {
	model.AuditLog
	TotalCount int `db:"total_count"`
}

// getTableName get audit log table name based on the given endDate string
// endDate is of the form yyyy-mm-dd[ hh:mm:ss]
func getTableName(endDate string) string {
	date := base.GetDateStart()
	parts := strings.Split(endDate, " ")
	if len(parts[0]) != 0 {
		date = parts[0]
	}
	return fmt.Sprintf("audit_log_%s", strings.Replace(date, "-", "", -1))
}

// SelectAuditLogs select all audit log entries for the given tenant (via ctx) matching the query parameters
// query parameters include page index, page size, time window start and end
// record sorted in descending time order
func (dbAPI *dbObjectModelAPI) SelectAuditLogs(ctx context.Context, queryParams model.AuditLogQueryParam) (model.AuditLogListResponsePayload, error) {
	resp := model.AuditLogListResponsePayload{
		PagedListResponsePayload: model.PagedListResponsePayload{PageIndex: queryParams.PageIndex, PageSize: queryParams.PageSize},
	}
	// get table name from end
	tableName := getTableName(queryParams.End)
	limit := queryParams.PageSize
	offset := queryParams.PageIndex * limit

	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	if authContext.TenantID == "" {
		return resp, errcode.NewBadRequestError("tenantID")
	}
	if !auth.IsInfraAdminRole(authContext) {
		return resp, errcode.NewPermissionDeniedError("RBAC/AuditLog")
	}

	filterAndOrderBy, err := getFilterAndOrderBy(entityTypeAuditLog, &queryParams.EntitiesQueryParam, "order by started_at DESC")
	if err != nil {
		return resp, errcode.NewBadRequestError("Filter/OrderBy")
	}
	sfx := fmt.Sprintf("%s OFFSET %d LIMIT %d", filterAndOrderBy, offset, limit)
	query := fmt.Sprintf(queryMap["SelectAllAuditLogsTemplate"], tableName, sfx)
	auditLogDBOs := []auditLogDBO{}
	err = dbAPI.Query(ctx, &auditLogDBOs, query, auditLogListQueryParam{TenantID: authContext.TenantID, Start: queryParams.Start, End: queryParams.End})
	if err != nil {
		return resp, err
	}
	auditLogs := []model.AuditLog{}
	totalCount := 0
	for _, auditLogDBO := range auditLogDBOs {
		auditLogs = append(auditLogs, auditLogDBO.AuditLog)
		if totalCount == 0 {
			totalCount = auditLogDBO.TotalCount
		}
	}
	resp.AuditLogList = auditLogs
	resp.PagedListResponsePayload.TotalCount = totalCount
	return resp, nil
}

// SelectAuditLogsW wrapper around SelectAuditLogs
// If query parameter not supplied in request URL,
// will retrieve latest 100 audit log entries today for the tenant
func (dbAPI *dbObjectModelAPI) SelectAuditLogsW(ctx context.Context, w io.Writer, r *http.Request) error {
	queryParams := model.GetAuditLogQueryParam(r)
	result, err := dbAPI.SelectAuditLogs(ctx, queryParams)
	if err == nil {
		err = json.NewEncoder(w).Encode(result)
	}
	return err
}

// DeleteTenantAuditLogs delete all audit log entries for the tenant specified via the ctx
func (dbAPI *dbObjectModelAPI) DeleteTenantAuditLogs(ctx context.Context) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	param := auditLogGetQueryParam{TenantID: authContext.TenantID}
	tableNameRecs := []auditLogReqIDToTableName{}
	err = dbAPI.Query(ctx, &tableNameRecs, queryMap["GetTenantAuditLogTables"], param)
	if err != nil {
		return err
	}
	for _, tableNameRec := range tableNameRecs {
		tableName := tableNameRec.TableName
		stmt := fmt.Sprintf(queryMap["DeleteTenantAuditLogTemplate"], tableName)
		_, err2 := dbAPI.NamedExec(ctx, stmt, param)
		if err2 != nil {
			// log and ignore error
			glog.Warningf("DeleteTenantAuditLogs: failed for tenant %s, table %s, error: %s\n", authContext.TenantID, tableName, err2.Error())
			err = err2
		}
	}
	// now delete all req id -> table name mapping for this tenant
	if err == nil {
		_, err = dbAPI.NamedExec(ctx, queryMap["DeleteTenantAuditLogReqToTableMap"], param)
	}
	return err
}

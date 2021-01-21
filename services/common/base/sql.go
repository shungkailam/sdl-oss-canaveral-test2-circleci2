package base

import (
	"bytes"
	"cloudservices/common/errcode"
	"cloudservices/common/metrics"
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
	"github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/jmoiron/sqlx"
)

// This is the begining of the record
const (
	StartPageToken       = PageToken("0")
	MaxRowsLimit         = 200000
	DeleteSQLQueryFormat = `DELETE FROM "%s" WHERE %s`
	PaginationSuffix     = ` AND id >= '%s' ORDER by id LIMIT %d`
	NilPageToken         = PageToken("")
	ResolveDBCacheTTL    = 5

	// Query to get the columns in a DB constraint by name
	dbContraintQuery = `select kcu.column_name, kcu.constraint_name from information_schema.table_constraints tco join information_schema.key_column_usage kcu  on kcu.constraint_name = tco.constraint_name and kcu.constraint_schema = tco.constraint_schema and kcu.constraint_name = tco.constraint_name where kcu.constraint_name = :constraint_name`
	// Query to get the table to table dependencies
	// Verified to work with minor modification https://www.postgresql.org/message-id/500C2205.2020609@computer.org
	dbDependencyQuery = `select c1.relname as table, c2.relname AS foreign_table from pg_catalog.pg_constraint c join only pg_catalog.pg_class c1 on c1.oid = c.confrelid join only pg_catalog.pg_class c2 on c2.oid = c.conrelid where c1.relkind = 'r' and c.contype = 'f' and c1.relname like '%_model' order by c1.relname`
)

// Only constants must be used in this block
func init() {
	var err error
	ModelNamePattern, err = regexp.Compile("( [a-z_]+_model )|(\"[a-z_]+_model\")")
	if err != nil {
		panic(err)
	}
}

var (
	ModelNamePattern *regexp.Regexp
	// Cache for db constraints to columns
	dbConstraints sync.Map

	// ErrForceRollback forces rollback, can be used for dry run
	ErrForceRollback = errors.New("Rollback")
)

// PageToken is for setting the next starting page
type PageToken string

type DBObjectModelAPI struct {
	db          *sqlx.DB
	roDB        *sqlx.DB
	redisClient *redis.Client
}

type DBConstraint struct {
	Constraint string
	Columns    []string
	Mutex      *sync.Mutex
}

type DBConstraintDBO struct {
	Constraint string `json:"constraint" db:"constraint_name"`
	Column     string `json:"column" db:"column_name"`
}

type TableDependencyPairDBO struct {
	Table        string `json:"table" db:"table"`
	ForeignTable string `json:"foreignTable" db:"foreign_table"`
}

type WrappedTx struct {
	*sqlx.Tx
	DBAPI *DBObjectModelAPI
}

// GetDBConstraintColumns gets the columns in the DB constraint and caches them
func (dbAPI *DBObjectModelAPI) GetDBConstraintColumns(ctx context.Context, constraint string) ([]string, error) {
	// Get from the cache
	actual, _ := dbConstraints.LoadOrStore(constraint, &DBConstraint{
		Columns: []string{},
		Mutex:   &sync.Mutex{},
	})
	dbConstraint := actual.(*DBConstraint)
	dbConstraint.Mutex.Lock()
	defer dbConstraint.Mutex.Unlock()
	columnLen := len(dbConstraint.Columns)
	if columnLen == 0 {
		rows := []DBConstraintDBO{}
		param := DBConstraintDBO{Constraint: constraint}
		err := dbAPI.Query(ctx, &rows, dbContraintQuery, param)
		if err != nil {
			glog.Warningf(PrefixRequestID(ctx, "Failed to fetch columns for DB constraint %s. Error: %s"), constraint, err.Error())
			return nil, err
		}
		for _, row := range rows {
			if row.Column == "tenant_id" && len(rows) > 1 {
				// Tenant ID is implicit. No need to add when other columns are present
				continue
			}
			// Set the reference
			dbConstraint.Columns = append(dbConstraint.Columns, row.Column)
		}
		columnLen = len(dbConstraint.Columns)
		if columnLen == 0 {
			glog.Warningf(PrefixRequestID(ctx, "Failed to fetch columns for DB constraint %s. No columns found"), constraint)
			return nil, errcode.NewRecordNotFoundError(constraint)
		}
		dbConstraint.Constraint = constraint
	}
	// Make a copy before the lock is released
	columnsCopy := make([]string, 0, columnLen)
	columnsCopy = append(columnsCopy, dbConstraint.Columns...)
	return columnsCopy, nil
}

// TranslateDatabaseError translates DB error into friendly error codes
func (dbAPI *DBObjectModelAPI) TranslateDatabaseError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	var pqErr *pq.Error
	var ok bool
	if _, ok = err.(errcode.ErrorCode); ok {
		return err
	}
	// Postgres throws *pq.Error
	if pqErr, ok = err.(*pq.Error); !ok {
		return errcode.NewInternalDatabaseError(err.Error())
	}
	sqlErrorType := errcode.GetSQLErrorType(err)
	if sqlErrorType == errcode.DUPLICATE_RECORD {
		columns, err1 := dbAPI.GetDBConstraintColumns(ctx, pqErr.Constraint)
		if err1 != nil {
			return errcode.NewInternalDatabaseError(err.Error())
		}
		return errcode.NewDatabaseDuplicateError(strings.Join(columns, " "))
	}
	if sqlErrorType == errcode.UNSATISIFIED_DEPENDENCY {
		columns, err1 := dbAPI.GetDBConstraintColumns(ctx, pqErr.Constraint)
		if err1 != nil {
			return errcode.NewInternalDatabaseError(err.Error())
		}
		return errcode.NewDatabaseDependencyError(strings.Join(columns, " "))
	}
	return errcode.NewInternalDatabaseError(err.Error())
}

func (tx *WrappedTx) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	rows, err := tx.Tx.NamedQuery(query, arg)
	if err == nil {
		if IsWriteQuery(query) {
			cacheErr := tx.DBAPI.PutCache(ctx, query, arg)
			if cacheErr != nil {
				glog.Errorf(PrefixRequestID(ctx, "NamedQuery: Failed to put %+v in cache. Error: %s"), arg, cacheErr.Error())
			}
		}
	} else {
		return nil, tx.DBAPI.TranslateDatabaseError(ctx, err)
	}
	return rows, nil
}

func (tx *WrappedTx) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	result, err := tx.Tx.NamedExec(query, arg)
	if err == nil {
		cacheErr := tx.DBAPI.PutCache(ctx, query, arg)
		if cacheErr != nil {
			glog.Errorf(PrefixRequestID(ctx, "NamedExec: Failed to put %+v in cache. Error: %s"), arg, cacheErr.Error())
		}
	} else {
		return nil, tx.DBAPI.TranslateDatabaseError(ctx, err)
	}
	return result, nil
}

func (dbAPI *DBObjectModelAPI) NamedQuery(ctx context.Context, query string, arg interface{}) (*sqlx.Rows, error) {
	rows, err := dbAPI.db.NamedQuery(query, arg)
	if err == nil {
		if IsWriteQuery(query) {
			cacheErr := dbAPI.PutCache(ctx, query, arg)
			if cacheErr != nil {
				glog.Errorf(PrefixRequestID(ctx, "NamedQuery: Failed to put %+v in cache. Error: %s"), arg, cacheErr.Error())
			}
		}
	} else {
		return nil, dbAPI.TranslateDatabaseError(ctx, err)
	}
	return rows, nil
}

func (dbAPI *DBObjectModelAPI) NamedExec(ctx context.Context, query string, arg interface{}) (sql.Result, error) {
	result, err := dbAPI.db.NamedExec(query, arg)
	if err == nil {
		cacheErr := dbAPI.PutCache(ctx, query, arg)
		if cacheErr != nil {
			glog.Errorf(PrefixRequestID(ctx, "NamedExec: Failed to put %+v in cache. Error: %s"), arg, cacheErr.Error())
		}
	} else {
		return nil, dbAPI.TranslateDatabaseError(ctx, err)
	}
	return result, nil
}

func (dbAPI *DBObjectModelAPI) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	result, err := dbAPI.db.Exec(query, args...)
	if err == nil {
		cacheErr := dbAPI.PutCache(ctx, query, args)
		if cacheErr != nil {
			glog.Errorf(PrefixRequestID(ctx, "NamedExec: Failed to put %+v in cache. Error: %s"), args, cacheErr.Error())
		}
	} else {
		return nil, dbAPI.TranslateDatabaseError(ctx, err)
	}
	return result, nil
}

func (dbAPI *DBObjectModelAPI) updateDBConnectionCount() {
	stats := dbAPI.db.Stats()
	metrics.DBConnections.With(prometheus.Labels{"hostname": os.Getenv("HOSTNAME")}).Set(float64(stats.OpenConnections))
}

func NewDBObjectModelAPI(driverName, dataSourceName, readOnlyDataSourceName string, redisClient *redis.Client) (*DBObjectModelAPI, error) {
	db, err := sqlx.Connect(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	roDB := db
	if len(readOnlyDataSourceName) > 0 && dataSourceName != readOnlyDataSourceName {
		roDB, err = sqlx.Connect(driverName, readOnlyDataSourceName)
		if err != nil {
			db.Close()
			return nil, err
		}
	}
	return &DBObjectModelAPI{db: db, roDB: roDB, redisClient: redisClient}, nil
}

func (dbAPI *DBObjectModelAPI) Close() error {
	errMsgs := []string{}
	if dbAPI.db != nil {
		err := dbAPI.db.Close()
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
	}
	if dbAPI.roDB != nil && dbAPI.db != dbAPI.roDB {
		err := dbAPI.roDB.Close()
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
	}
	var err error
	if len(errMsgs) > 0 {
		err = fmt.Errorf("Error closing connections. Error: %s", strings.Join(errMsgs, ","))
	}
	return err
}

// GetDB gets the DB handle to the read and write cluster
func (dbAPI *DBObjectModelAPI) GetDB() *sqlx.DB {
	return dbAPI.db
}

// GetReadOnlyDB returns the db handle to the read only cluster
func (dbAPI *DBObjectModelAPI) GetReadOnlyDB() *sqlx.DB {
	return dbAPI.roDB
}

// GetRedisClient returns the redis client handle
func (dbAPI *DBObjectModelAPI) GetRedisClient() *redis.Client {
	return dbAPI.redisClient
}

// ResolveDB returns write or read DB handle depending on the cache status
func (dbAPI *DBObjectModelAPI) ResolveDB(ctx context.Context, query string, model interface{}) *sqlx.DB {
	if dbAPI.redisClient == nil {
		return dbAPI.GetDB()
	}
	if dbAPI.GetDB() == dbAPI.GetReadOnlyDB() {
		// Short circuit. No need to read cache
		return dbAPI.GetDB()
	}
	// There can be global keys
	keys := GetDBCacheObjectKeys(ctx, query, model, true)
	if len(keys) == 0 {
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "ResolveDB: keys cannot be created for query %s and param %+v"), query, model)
		}
		// Key cannot be resolved from the model.
		// Tenant ID is missing. Read from write DB.
		return dbAPI.GetDB()
	}
	for _, key := range keys {
		_, err := dbAPI.redisClient.Get(key).Result()
		if err == nil {
			if glog.V(4) {
				glog.V(4).Infof(PrefixRequestID(ctx, "ResolveDB: Found keys %+v"), keys)
			}
			// At least one is found to be existing
			return dbAPI.GetDB()
		}
		if err != redis.Nil {
			if glog.V(4) {
				glog.V(4).Infof(PrefixRequestID(ctx, "ResolveDB: Error occurred on querying for key %+v. Error: %s"), key, err.Error())
			}
			// Some read error occurred
			return dbAPI.GetDB()
		}
	}
	return dbAPI.GetReadOnlyDB()
}

// PutCache generates the key from the query and model and puts them into the cache.
// Depending on the object, the generated keys may not have tenant ID.
func (dbAPI *DBObjectModelAPI) PutCache(ctx context.Context, query string, model interface{}) error {
	if dbAPI.redisClient == nil {
		return nil
	}
	if dbAPI.GetDB() == dbAPI.GetReadOnlyDB() {
		return nil
	}
	errMsg := []string{}
	keys := GetDBCacheObjectKeys(ctx, query, model, false)
	for _, key := range keys {
		err := dbAPI.redisClient.Set(key, "true", time.Second*ResolveDBCacheTTL).Err()
		if err != nil {
			errMsg = append(errMsg, err.Error())
		}
	}
	if len(errMsg) == 0 {
		return nil
	}
	return fmt.Errorf(PrefixRequestID(ctx, "PutCache: Error: %s"), strings.Join(errMsg, ","))
}

func getTenantID(ctx context.Context, model interface{}) string {
	var tenantID string
	value := reflect.Indirect(reflect.ValueOf(model))
	if value.Type().Kind() == reflect.Struct {
		fValue := value.FieldByName("TenantID")
		if fValue.IsValid() {
			tenantID = fValue.String()
		}
	}
	if len(tenantID) == 0 {
		// Try to get from the context.
		// The cache key is always either :schema or tenantID:schema.
		// Read always uses both the keys to check. If tenantID is available,
		// cache hit is guaranteed if either key is present. Otherwise,
		// the other key tenantID:schema can be missed.
		authContext, err := GetAuthContext(ctx)
		if err == nil {
			tenantID = authContext.TenantID
		}
	}
	return tenantID
}

// GetDBCacheObjectKeys extracts the model/table names from the query
func GetDBCacheObjectKeys(ctx context.Context, query string, model interface{}, isRead bool) []string {
	keys := []string{}
	tenantID := getTenantID(ctx, model)
	if isRead && len(tenantID) == 0 {
		// Read pattern is without a tenant like get by serial, email etc
		return keys
	}
	// For write/delete, cache with or without the tenant ID
	tokens := ModelNamePattern.FindAllString(query, -1)
	for _, token := range tokens {
		schema := strings.TrimSpace(token)
		schema = strings.TrimLeft(schema, "\"")
		schema = strings.TrimRight(schema, "\"")
		keys = append(keys, fmt.Sprintf("%s:%s", tenantID, schema))
		if isRead {
			// Include everything even without teannt
			keys = append(keys, fmt.Sprintf(":%s", schema))
		}
	}
	return keys
}

func createDeleteQuery(tableName string, params map[string]interface{}) string {
	count := 0
	var buffer bytes.Buffer
	for key := range params {
		count++
		buffer.WriteString(key)
		buffer.WriteString(" = :")
		buffer.WriteString(key)
		if count < len(params) {
			buffer.WriteString(" AND ")
		}
	}
	return fmt.Sprintf(DeleteSQLQueryFormat, tableName, buffer.String())
}

// PagedQuery queries for records with pagination support.
func (dbAPI *DBObjectModelAPI) PagedQuery(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, model interface{}) (PageToken, error) {
	db := dbAPI.ResolveDB(ctx, query, model)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "PagedQuery %s with param %+v took %.2f ms"), query, model, float32(stop/time.Millisecond))
		}
	}()
	// Fetch one more extra row to set the next page
	return StructScanRowsWithCallback(pageSize, callback, func() (*sqlx.Rows, error) {
		pagedQuery := fmt.Sprintf(query+PaginationSuffix, startPage, pageSize+1)
		rows, err := db.NamedQuery(pagedQuery, model)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), model, pagedQuery, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, err
	}, model)
}
func (dbAPI *DBObjectModelAPI) NotPagedQuery(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, model interface{}) (PageToken, error) {
	db := dbAPI.ResolveDB(ctx, query, model)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "NotPagedQuery %s with param %+v took %.2f ms"), query, model, float32(stop/time.Millisecond))
		}
	}()
	// Fetch one more extra row to set the next page
	return StructScanRowsWithCallback(pageSize, callback, func() (*sqlx.Rows, error) {
		rows, err := db.NamedQuery(query, model)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), model, query, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, err
	}, model)
}

// PagedQueryEx queries for records with pagination support and with a different output model
func (dbAPI *DBObjectModelAPI) PagedQueryEx(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, param interface{}, model interface{}) (PageToken, error) {
	db := dbAPI.ResolveDB(ctx, query, param)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "PagedQueryEx %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	// Fetch one more extra row to set the next page
	return StructScanRowsWithCallback(pageSize, callback, func() (*sqlx.Rows, error) {
		pagedQuery := fmt.Sprintf(query+PaginationSuffix, startPage, pageSize+1)
		rows, err := db.NamedQuery(pagedQuery, param)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, pagedQuery, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, err
	}, model)
}

func (dbAPI *DBObjectModelAPI) NotPagedQueryEx(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, param interface{}, model interface{}) (PageToken, error) {
	db := dbAPI.ResolveDB(ctx, query, param)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "NotPagedQueryEx %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	// Fetch one more extra row to set the next page
	return StructScanRowsWithCallback(pageSize, callback, func() (*sqlx.Rows, error) {
		rows, err := db.NamedQuery(query, param)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, query, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, err
	}, model)
}

// StructScanRowsWithCallback calls the callback for every record from the rowSource.StructScanRowsWithCallback.
// If page size is 0, next token is never returned
func StructScanRowsWithCallback(pageSize int, callback func(dbObjPtr interface{}) error, rowSource func() (*sqlx.Rows, error), model interface{}) (PageToken, error) {
	nextToken := NilPageToken
	modelType := reflect.TypeOf(model)
	if modelType.Kind() == reflect.Ptr {
		return nextToken, errors.New("Model param must be a value")
	}
	if rowSource == nil {
		return nextToken, errcode.NewBadRequestError("source")
	}
	rows, err := rowSource()
	if err != nil {
		return nextToken, err
	}
	defer rows.Close()
	index := 0
	for rows.Next() {
		value := reflect.New(modelType)
		dbObjPtr := value.Interface()
		err = rows.StructScan(dbObjPtr)
		if err != nil {
			glog.Errorf("Error scanning for model %s and params %+v. Error: %s", modelType.Name(), model, err.Error())
			return nextToken, errcode.NewInternalDatabaseError(err.Error())
		}

		// Interface returns the current value
		if pageSize > 0 && index == pageSize {
			dbObj := reflect.Indirect(value).Interface()
			idValue := reflect.ValueOf(dbObj).FieldByName("ID")
			if idValue.Kind() == reflect.String {
				nextToken = PageToken(idValue.String())
			}
			break
		}
		// &dbObj makes type cast to *modelDBO fail as it is *interface{}
		err = callback(dbObjPtr)
		if err != nil {
			glog.Errorf("Error in callback for model %s and params %+v. Error: %s", modelType.Name(), model, err.Error())
			return nextToken, err
		}
		index++
	}
	return nextToken, nil
}

// StructScanRows populates the slice with the record from the source
func StructScanRows(slice interface{}, rowSource func() (*sqlx.Rows, error)) error {
	sliceType := reflect.TypeOf(slice)
	if sliceType.Kind() != reflect.Ptr {
		glog.Error("Receiver must be a pointer")
		return errcode.NewBadRequestError("output")
	}
	elemType := sliceType.Elem().Elem()
	if elemType.Kind() == reflect.Ptr {
		glog.Error("Element type must not be a pointer")
		return errcode.NewBadRequestError("output")
	}
	if rowSource == nil {
		return errcode.NewBadRequestError("source")
	}
	rows, err := rowSource()
	if err != nil {
		return err
	}
	defer rows.Close()
	sliceValue := reflect.Indirect(reflect.ValueOf(slice))
	for rows.Next() {
		value := reflect.New(elemType)
		dbObjPtr := value.Interface()
		err := rows.StructScan(dbObjPtr)
		if err != nil {
			return errcode.NewInternalDatabaseError(err.Error())
		}
		dbObj := reflect.Indirect(value)
		sliceValue.Set(reflect.Append(sliceValue, dbObj))
	}
	return nil
}

type InQueryParam struct {
	Param   interface{}
	Key     string
	InQuery bool
}

func (dbAPI *DBObjectModelAPI) QueryInMaybe(ctx context.Context, slice interface{}, query string, param InQueryParam) error {
	if param.InQuery {
		return dbAPI.QueryIn(ctx, slice, query, param.Param)
	} else {
		return dbAPI.Query(ctx, slice, query, param.Param)
	}
}

// GetQueryInResult returns the rows by executing SQL IN query
func (dbAPI *DBObjectModelAPI) GetQueryInResult(ctx context.Context, query string, param interface{}) (*sqlx.Rows, error) {
	start := time.Now()
	defer func() {
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "QueryInResult %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	db := dbAPI.ResolveDB(ctx, query, param)
	query, args, err := sqlx.Named(query, param)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in QueryInResult for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		glog.Errorf("Error in QueryInResult for query %s and params %+v. Error: %s", query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query = db.Rebind(query)
	rows, err := db.Queryx(query, args...)
	if err != nil {
		glog.Errorf("Error in QueryInResult for query %s and params %+v. Error: %s", query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	return rows, nil
}

// GetQueryInResultTxn returns the rows by executing SQL IN query using the transaction handle
func GetQueryInResultTxn(ctx context.Context, tx *WrappedTx, query string, param interface{}) (*sqlx.Rows, error) {
	start := time.Now()
	defer func() {
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "GetQueryInResultTxn %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	query, args, err := sqlx.Named(query, param)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in GetQueryInResultTxn for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		glog.Errorf("Error in GetQueryInResultTxn for query %s and params %+v. Error: %s", query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query = tx.Rebind(query)
	rows, err := tx.Queryx(query, args...)
	if err != nil {
		glog.Errorf("Error in GetQueryInResultTxn for query %s and params %+v. Error: %s", query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	return rows, nil
}

// GetPagedQueryInResult returns the rows by executing SQL paged IN query
func (dbAPI *DBObjectModelAPI) GetPagedQueryInResult(ctx context.Context, query string, param interface{}) (*sqlx.Rows, error) {
	db := dbAPI.ResolveDB(ctx, query, param)
	query, args, err := sqlx.Named(query, param)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in QueryInResult for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in QueryInResult for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query = db.Rebind(query)
	rows, err := db.Queryx(query, args...)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in QueryInResult for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	return rows, nil
}

// QueryInWithCallback is similar to QueryIn except that it makes a call on the callback for every record
func (dbAPI *DBObjectModelAPI) QueryInWithCallback(ctx context.Context, callback func(dbObjPtr interface{}) error, query string, model interface{}, param interface{}) error {
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "QueryInWithCallback %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	_, err := StructScanRowsWithCallback(0, callback, func() (*sqlx.Rows, error) {
		return dbAPI.GetQueryInResult(ctx, query, param)
	}, model)
	return err
}

// Special handling to support 'IN' query
// see https://github.com/jmoiron/sqlx/issues/485
func (dbAPI *DBObjectModelAPI) QueryIn(ctx context.Context, slice interface{}, query string, param interface{}) error {
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "QueryIn %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	return StructScanRows(slice, func() (*sqlx.Rows, error) {
		return dbAPI.GetQueryInResult(ctx, query, param)
	})
}

// QueryInTxn executes the IN query using the transaction handle
func QueryInTxn(ctx context.Context, tx *WrappedTx, slice interface{}, query string, param interface{}) error {
	start := time.Now()
	defer func() {
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "QueryInTxn %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	return StructScanRows(slice, func() (*sqlx.Rows, error) {
		return GetQueryInResultTxn(ctx, tx, query, param)
	})
}

var reDollarVar = regexp.MustCompile(`\$[0-9]+`)

// Special handling to support paged 'IN' query
func (dbAPI *DBObjectModelAPI) PagedQueryIn(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, param interface{}) (PageToken, error) {
	db := dbAPI.ResolveDB(ctx, query, param)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof("PagedQueryIn %s with param %+v took %.2f ms", query, param, float32(stop/time.Millisecond))
		}
	}()
	// Fetch one more extra row to set the next page
	return StructScanRowsWithCallback(pageSize, callback, func() (*sqlx.Rows, error) {
		pagedQuery := fmt.Sprintf(query+PaginationSuffix, startPage, pageSize+1)

		q, args, err := db.BindNamed(pagedQuery, param)
		if err != nil {
			return nil, err
		}
		// convert $d back to ? needed by sqlx.In
		q = reDollarVar.ReplaceAllString(q, "?")
		q, args, err = sqlx.In(q, args...)
		if err != nil {
			return nil, err
		}
		q = db.Rebind(q)
		rows, err := db.Queryx(q, args...)

		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, pagedQuery, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, err
	}, param)
}

// NotPagedQueryIn is a paginated query for IN query
func (dbAPI *DBObjectModelAPI) NotPagedQueryIn(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, param interface{}) (PageToken, error) {
	return dbAPI.NotPagedQueryInEx(ctx, startPage, pageSize, callback, query, param, param)
}

// NotPagedQueryInEx is like NotPagedQueryIn with the ability to pass a different query param model and receiver
func (dbAPI *DBObjectModelAPI) NotPagedQueryInEx(ctx context.Context, startPage PageToken, pageSize int, callback func(dbObjPtr interface{}) error, query string, param interface{}, model interface{}) (PageToken, error) {
	db := dbAPI.ResolveDB(ctx, query, param)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof("NotPagedQueryIn %s with param %+v took %.2f ms", query, param, float32(stop/time.Millisecond))
		}
	}()
	// Fetch one more extra row to set the next page
	return StructScanRowsWithCallback(pageSize, callback, func() (*sqlx.Rows, error) {
		q, args, err := db.BindNamed(query, param)
		if err != nil {
			return nil, err
		}
		// convert $d back to ? needed by sqlx.In
		q = reDollarVar.ReplaceAllString(q, "?")
		q, args, err = sqlx.In(q, args...)
		if err != nil {
			return nil, err
		}
		q = db.Rebind(q)
		rows, err := db.Queryx(q, args...)

		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, query, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, err
	}, model)
}

// Query fetches everything in one shot
func (dbAPI *DBObjectModelAPI) Query(ctx context.Context, slice interface{}, query string, param interface{}) error {
	db := dbAPI.ResolveDB(ctx, query, param)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "Query %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	return StructScanRows(slice, func() (*sqlx.Rows, error) {
		rows, err := db.NamedQuery(query, param)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, query, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, nil
	})
}

// Queryx similar to query, but don't use named query
func (dbAPI *DBObjectModelAPI) Queryx(ctx context.Context, slice interface{}, query string, param interface{}) error {
	db := dbAPI.ResolveDB(ctx, query, param)
	start := time.Now()
	defer func() {
		dbAPI.updateDBConnectionCount()
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "Query %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	return StructScanRows(slice, func() (*sqlx.Rows, error) {
		rows, err := db.Queryx(query)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, query, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, nil
	})
}

// QueryTxn queries in the transaction
func QueryTxn(ctx context.Context, tx *WrappedTx, slice interface{}, query string, param interface{}) error {
	start := time.Now()
	defer func() {
		stop := time.Since(start)
		if glog.V(4) {
			glog.V(4).Infof(PrefixRequestID(ctx, "QueryTxn %s with param %+v took %.2f ms"), query, param, float32(stop/time.Millisecond))
		}
	}()
	return StructScanRows(slice, func() (*sqlx.Rows, error) {
		rows, err := tx.NamedQuery(ctx, query, param)
		if err != nil {
			glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, query, err.Error())
			return nil, errcode.NewInternalDatabaseError(err.Error())
		}
		return rows, nil
	})
}

// Delete deletes the matching records from the table
func (dbAPI *DBObjectModelAPI) Delete(ctx context.Context, tableName string, params map[string]interface{}) (sql.Result, error) {
	db := dbAPI.GetDB()
	query := createDeleteQuery(tableName, params)
	glog.Infoln("Running query", query)
	result, err := db.NamedExec(query, params)
	defer func() {
		dbAPI.updateDBConnectionCount()
	}()
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in deleting records from %s with param %s. Error: %s"), tableName, params, err.Error())
		return nil, dbAPI.TranslateDatabaseError(ctx, err)
	}
	// TODO make delete carry more information
	dbAPI.PutCache(ctx, query, params)
	return result, nil
}

// DeleteIn deletes records matching the values in the IN clause
func (dbAPI *DBObjectModelAPI) DeleteIn(ctx context.Context, query string, param interface{}) (sql.Result, error) {
	db := dbAPI.GetDB()
	query, args, err := sqlx.Named(query, param)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in DeleteIn for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query, args, err = sqlx.In(query, args...)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in DeleteIn for query %s and params %+v. Error: %s"), query, param, err.Error())
		return nil, errcode.NewInternalDatabaseError(err.Error())
	}
	query = db.Rebind(query)
	result, err := db.Exec(query, args...)
	defer func() {
		dbAPI.updateDBConnectionCount()
	}()
	if err != nil {
		return nil, dbAPI.TranslateDatabaseError(ctx, err)
	}
	dbAPI.PutCache(ctx, query, param)
	return result, nil
}

// DeleteTxn deletes the record in the transaction
func DeleteTxn(ctx context.Context, tx *WrappedTx, tableName string, params map[string]interface{}) (sql.Result, error) {
	query := createDeleteQuery(tableName, params)
	glog.Infoln(PrefixRequestID(ctx, "Running query in transaction"), query)
	result, err := tx.NamedExec(ctx, query, params)
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error in deleting records from %s with param %s. Error: %s"), tableName, params, err.Error())
		sqlErrorType := errcode.GetSQLErrorType(err)
		if sqlErrorType == errcode.UNSATISIFIED_DEPENDENCY {
			return result, errcode.NewRecordInUseError()
		}
		return result, errcode.NewInternalDatabaseError(err.Error())
	}
	// TODO make delete carry more information
	tx.DBAPI.PutCache(ctx, query, params)
	return result, nil
}

// DoInTxn starts a transaction and passes the transaction handle to the callback
// It rolls back on error, otherwise the transaction is committted when the callback returns
func (dbAPI *DBObjectModelAPI) DoInTxn(callback func(tx *WrappedTx) error) error {
	db := dbAPI.GetDB()
	tx, err := db.Beginx()
	if err != nil {
		return errcode.NewInternalDatabaseError(err.Error())
	}

	// Need to update err as the defer'ed function above checks its value to
	// either commit or rollback the transaction
	err = callback(&WrappedTx{Tx: tx, DBAPI: dbAPI})
	if err != nil {
		if err == ErrForceRollback {
			err = nil
		} else {
			glog.Errorf("Error occurred in transaction. Rolling back the changes. Error: %s", err.Error())
		}
		sqlErr := tx.Tx.Rollback()
		if sqlErr != nil {
			glog.Errorf("Error occurred in roll back. Error: %s", sqlErr.Error())
			err = errcode.NewInternalDatabaseError(sqlErr.Error())
		}
	} else {
		sqlErr := tx.Tx.Commit()
		if sqlErr != nil {
			glog.Errorf("Error occurred in commit. Error: %s", sqlErr.Error())
			err = errcode.NewInternalDatabaseError(sqlErr.Error())
		}
	}
	return err
}

// EncodeJSONArrayStream uses Rows cursor to write json array
// as stream directly into io Writer to bound memory usage
// for large json arrays
func EncodeJSONArrayStream(rows *sqlx.Rows, w io.Writer, item interface{}) error {
	var err error
	defer rows.Close()
	jw := json.NewEncoder(w)
	_, err = w.Write([]byte{'['})
	if err != nil {
		return err
	}
	firstRow := true
	for rows.Next() {
		if firstRow {
			firstRow = false
		} else {
			_, err = w.Write([]byte{','})
			if err != nil {
				return err
			}
		}
		err = rows.StructScan(item)
		if err != nil {
			return err
		}
		err = jw.Encode(item)
		if err != nil {
			return err
		}
	}
	_, err = w.Write([]byte{']'})
	if err != nil {
		return err
	}
	return nil
}

// IsDeleteSuccessful is a helper method to check if any record is deleted.
func IsDeleteSuccessful(result sql.Result) bool {
	rows, err := result.RowsAffected()
	return err == nil && rows != 0
}

// DeleteOrUpdateOk returns true if the result affected at least a row
func DeleteOrUpdateOk(result sql.Result) (bool, error) {
	if result == nil {
		return false, nil
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return false, errcode.NewInternalDatabaseError(err.Error())
	}
	return rows != 0, nil
}

// DeleteW implement deleteFnW from deleteFn
func DeleteW(context context.Context, deleteFn func(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error), id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	obj, err := deleteFn(context, id, callback)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(obj)
}

func createOrUpdateW(context context.Context, createOrUpdateFn func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error), doc interface{}, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error, operation string) error {
	err := Decode(&r, doc)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	err = ValidateStruct("body", doc, operation)
	if err != nil {
		return err
	}
	resp, err := createOrUpdateFn(context, doc, callback)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)
}

// UpdateW implement updateFnW from updateFn
func UpdateW(context context.Context, updateFn func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error), doc interface{}, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return createOrUpdateW(context, updateFn, doc, w, r, callback, "update")
}

// CreateW implement createFnW from createFn
func CreateW(context context.Context, createFn func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error), doc interface{}, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return createOrUpdateW(context, createFn, doc, w, r, callback, "create")
}

// Value returns the driver compatible value
func (a StringArray) Value() (driver.Value, error) {
	return "{" + strings.Join(a, ",") + "}", nil
}

// ToSQLList returns a list of strings as is used in SQL `WHERE` clause.
// for example: ('1','2','3','4')
func ToSQLList(input []string) string {
	ids := make([]string, 0, len(input))
	for _, id := range input {
		ids = append(ids, fmt.Sprintf("'%s'", id))
	}
	return fmt.Sprintf("%s", strings.Join(ids, ","))
}

// IsWriteQuery parses the sql query to check if it a write query or not
func IsWriteQuery(query string) bool {
	tokens := strings.SplitN(strings.TrimSpace(query), " ", 2)
	if len(tokens) == 0 {
		return false
	}
	sqlCmd := strings.ToLower(strings.TrimSpace(tokens[0]))
	return sqlCmd == "insert" || sqlCmd == "delete" || sqlCmd == "update"
}

// GetTableDependencies returns the table to tables dependencies due to foreign key references.
func (dbAPI *DBObjectModelAPI) GetTableDependencies(ctx context.Context) (map[string]map[string]bool, error) {
	dependencies := map[string]map[string]bool{}
	tablePairDBOs := []TableDependencyPairDBO{}
	err := dbAPI.Query(ctx, &tablePairDBOs, dbDependencyQuery, TableDependencyPairDBO{})
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Failed to get foreign key table dependencies. Error: %s"), err.Error())
		return nil, err
	}
	for _, tablePairDBO := range tablePairDBOs {
		foreignTables, ok := dependencies[tablePairDBO.Table]
		if !ok {
			foreignTables = map[string]bool{}
			dependencies[tablePairDBO.Table] = foreignTables
		}
		foreignTables[tablePairDBO.ForeignTable] = true
	}
	return dependencies, nil
}

// LockRows locks the rows with the entity IDs
func LockRows(ctx context.Context, tx *WrappedTx, table string, entityIDs []string) error {
	if !strings.HasSuffix(table, "_model") {
		return errcode.NewBadRequestError("table")
	}
	if len(entityIDs) == 0 {
		return errcode.NewBadRequestError("ids")
	}
	param := strings.Join(entityIDs, "','")
	query := fmt.Sprintf("SELECT id FROM %s WHERE id in ('%s') FOR UPDATE", table, param)
	rows, err := tx.NamedQuery(ctx, query, struct{}{})
	if err != nil {
		glog.Errorf(PrefixRequestID(ctx, "Error querying with params %+v. Query: %s, Error: %s"), param, query, err.Error())
		return errcode.NewInternalDatabaseError(err.Error())
	}
	defer rows.Close()
	return nil
}

/*
Athena data uploader scans tables and uploads the contents to S3. Athena tables are also created for the tables if they do not exist.
*/

package pkg

import (
	"bytes"
	"cloudservices/common/base"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"text/template"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/athena"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/golang/glog"
)

const (
	// Query to get column names and types of a table
	columnTypeQuery = `SELECT column_name, udt_name as column_type FROM information_schema.columns WHERE table_name=:table_name`
)

var (
	// Type mapping from SQL DB to Athena type for known handled types
	dbTypeToAthenaType = map[string]string{
		"varchar":   "string",
		"bigint":    "bigint",
		"text":      "string",
		"timestamp": "string",
		"int8":      "bigint",
		"int4":      "int",
		"jsonb":     "string",
		"bool":      "boolean",
	}

	createAthenaTableQueryTemplate = template.Must(template.New("CreateAthenaTableQuery").Parse(`CREATE EXTERNAL TABLE IF NOT EXISTS {{.TableName}} ( {{.Columns}} )
		PARTITIONED BY (year string, month string, day string)
		ROW FORMAT SERDE 'org.openx.data.jsonserde.JsonSerDe' WITH SERDEPROPERTIES ( 'ignore.malformed.json'='true')
		LOCATION {{.Location}} TBLPROPERTIES ( 'discover.partitions'='true' );`))

	selectQueryTemplate = template.Must(template.New("SelectQuery").Parse(`SELECT {{.Columns}} FROM {{.TableName}} {{if .Filter}} WHERE {{.Filter}} {{else}} {{end}}`))
)

// AthenaDataUploader is the DB data uploader
type AthenaDataUploader struct {
	config     *ConfigData
	awsSession *session.Session
	dbClient   *base.DBObjectModelAPI
}

// NewAthenaDataUploader returns an instance of DB data uploader
func NewAthenaDataUploader(config *ConfigData) *AthenaDataUploader {
	awsSession := session.Must(session.NewSession(&aws.Config{
		Region: config.AWSRegion},
	))
	uploader := &AthenaDataUploader{
		config:     config,
		awsSession: awsSession,
	}
	return uploader
}

// DBTableColumn is the DAO for fetching table column types
type DBTableColumn struct {
	TableName  string `json:"tableName" db:"table_name"`
	ColumnName string `json:"columnName" db:"column_name"`
	ColumnType string `json:"columnType" db:"column_type"`
}

// QueryFilter is the table query filter
type QueryFilter struct {
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// TableMetadata stores the table metadata
type TableMetadata struct {
	TableName       string
	ColumnMetadatas []*ColumnMetadata
}

// ColumnMetadata represents column metadata
type ColumnMetadata struct {
	ColumnName string
	ColumnType string
}

// Column stores metadata and the value
type Column struct {
	ColumnMetadata
	Value interface{}
}

// Record stores a record - column metadata and values for a row
type Record struct {
	Name    string
	Columns []*Column
}

// generateAthenaCreateTableQuery returns the create query for creating a table in Athena
func (tableMetadata *TableMetadata) generateAthenaCreateTableQuery(athenaTableName, s3Location string) (string, error) {
	var query string
	templateValues := map[string]string{}
	templateValues["TableName"] = athenaTableName
	buffer := &bytes.Buffer{}
	for _, columnMetadata := range tableMetadata.ColumnMetadatas {
		athenaType, ok := dbTypeToAthenaType[columnMetadata.ColumnType]
		if !ok {
			glog.Errorf("Unhandled DB type %s", columnMetadata.ColumnType)
			return query, fmt.Errorf("Unhandled DB type %s", columnMetadata.ColumnType)
		}
		if buffer.Len() > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(columnMetadata.ColumnName)
		buffer.WriteString(" ")
		buffer.WriteString(athenaType)
	}
	templateValues["Columns"] = buffer.String()
	templateValues["Location"] = fmt.Sprintf("'%s'", s3Location)
	outBuffer := &bytes.Buffer{}
	err := createAthenaTableQueryTemplate.Execute(outBuffer, templateValues)
	if err != nil {
		return query, err
	}
	query = outBuffer.String()
	glog.Infof("Generated Athena query: %s", query)
	return query, nil
}

// generateDBSelectQuery generates the select query for SQL DB
func (tableMetadata *TableMetadata) generateDBSelectQuery() (string, error) {
	var query string
	templateValues := map[string]string{}
	templateValues["TableName"] = tableMetadata.TableName
	buffer := &bytes.Buffer{}
	for _, columnMetadata := range tableMetadata.ColumnMetadatas {
		if buffer.Len() > 0 {
			buffer.WriteString(", ")
		}
		buffer.WriteString(columnMetadata.ColumnName)
	}
	templateValues["Columns"] = buffer.String()
	// Add filters
	for _, columnMetadata := range tableMetadata.ColumnMetadatas {
		if columnMetadata.ColumnName == "created_at" {
			templateValues["Filter"] = "created_at < :created_at"
			break
		}
	}
	outBuffer := &bytes.Buffer{}
	err := selectQueryTemplate.Execute(outBuffer, templateValues)
	if err != nil {
		return query, err
	}
	query = outBuffer.String()
	glog.Infof("Generated SQL query: %s", query)
	return query, nil
}

// getTableMetadata returns the metadata of a table - column names and types
func (uploader *AthenaDataUploader) getTableMetadata(ctx context.Context, tableName string) (*TableMetadata, error) {
	tableColumns := []DBTableColumn{}
	err := uploader.dbClient.Query(ctx, &tableColumns, columnTypeQuery, DBTableColumn{TableName: tableName})
	if err != nil {
		return nil, err
	}
	tableMetadata := &TableMetadata{TableName: tableName}
	for _, tableColumn := range tableColumns {
		tableMetadata.ColumnMetadatas = append(tableMetadata.ColumnMetadatas, &ColumnMetadata{
			ColumnName: tableColumn.ColumnName,
			ColumnType: tableColumn.ColumnType,
		})
	}
	return tableMetadata, nil
}

// Start starts the uploading process. This is the entry method for this uploader.
// The time runAt is truncated to the last midnight to fetch records created by last midnight
func (uploader *AthenaDataUploader) Start(ctx context.Context, runAt time.Time, tables ...string) error {
	if len(tables) == 0 {
		return nil
	}
	errMsgs := []string{}
	err := uploader.connectDB(ctx)
	if err != nil {
		glog.Errorf("Failed to connect to DB. Error: %s", err.Error())
		return err
	}
	tableNames := make(chan string) // Unbuffered
	out := make(chan *Record)       // Unbuffered

	dbScanConcurrency := *uploader.config.DBScanConcurrency
	s3UploadConcurreny := *uploader.config.S3UploadConcurrency
	uploadRes := make(chan error, dbScanConcurrency)
	scanRes := make(chan error, s3UploadConcurreny)

	oneDay := time.Hour * 24
	lastMidnight := runAt.Truncate(oneDay)
	// previousMidnight is one day less than the last midnight.
	// This is for putting the objects in previous day's folder
	previousMidnight := lastMidnight.Add(-oneDay)
	glog.Infof("Running upto time: %+v", lastMidnight)
	for i := 0; i < dbScanConcurrency; i++ {
		glog.Infof("Creating s3 uploader...")
		go uploader.uploadS3Objects(ctx, previousMidnight, out, uploadRes)
	}
	for i := 0; i < s3UploadConcurreny; i++ {
		glog.Infof("Creating DB scanner...")
		go uploader.scanTables(ctx, lastMidnight, tableNames, out, scanRes)
	}
	// Activate the scanner
	for i := range tables {
		// This gets blocked as lons as there is an entry as there is no buffer.
		// It makes sure that the last entry is picked up when the call returns
		tableNames <- tables[i]
	}
	glog.Infof("Closing scanner channel")
	close(tableNames)
	counter := s3UploadConcurreny
	for err := range scanRes {
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
		counter--
		if counter == 0 {
			break
		}
	}
	close(scanRes)
	// By this all the rows have been scanned and picked up by the uploader
	glog.Infof("Closing uploader channel")
	close(out)
	counter = dbScanConcurrency
	for err := range uploadRes {
		if err != nil {
			errMsgs = append(errMsgs, err.Error())
		}
		counter--
		if counter == 0 {
			break
		}
	}
	close(uploadRes)
	if len(errMsgs) > 0 {
		err = fmt.Errorf("Errors: %s", strings.Join(errMsgs, "; "))
		glog.Errorf(err.Error())
	} else {
		glog.Infof("DB data uploader completed successfully")
	}
	if err != nil {
		return err
	}
	// Reload partitions
	datePathComponent := getDatePathComponent(previousMidnight)
	for _, tableName := range tables {
		err = uploader.syncTable(ctx, tableName, datePathComponent)
		if err != nil {
			glog.Errorf("Failed to sync table %s", tableName)
			// Ignore error, hive has some issues
		}
	}
	return nil
}

// ConnectDB connects to SQL DB and creates DBObjectModelAPI
func (uploader *AthenaDataUploader) connectDB(ctx context.Context) error {
	dbURL, err := base.GetDBURL(*uploader.config.SQLDialect, *uploader.config.SQLDB,
		*uploader.config.SQLUser, *uploader.config.SQLPassword,
		*uploader.config.SQLHost, *uploader.config.SQLPort, false)
	if err != nil {
		glog.Errorf("Failed to get DB URL. Error: %s", err.Error())
		return err
	}
	dbClient, err := base.NewDBObjectModelAPI(*Cfg.SQLDialect, dbURL, dbURL, nil)
	if err != nil {
		glog.Errorf("Failed to create DB object model API instance. Error: %s", err.Error())
		return err
	}
	uploader.dbClient = dbClient
	return nil
}

// ExecuteAthenaQuery executes an Athena query synchronously
func (uploader *AthenaDataUploader) ExecuteAthenaQuery(ctx context.Context, input *athena.StartQueryExecutionInput) error {
	if input == nil || input.QueryString == nil {
		return errors.New("Invalid query execution input")
	}
	glog.Infof("Running Athena query: %s", *input.QueryString)
	athenaClient := athena.New(uploader.awsSession)
	// Asynchronous method to start a query
	startQueryExecOut, err := athenaClient.StartQueryExecutionWithContext(aws.BackgroundContext(), input)
	if err != nil {
		return err
	}
	tick := time.Tick(time.Second)
	for range tick {
		var getQueryExecOut *athena.GetQueryExecutionOutput
		getQueryExecOut, err = athenaClient.GetQueryExecution(&athena.GetQueryExecutionInput{
			QueryExecutionId: startQueryExecOut.QueryExecutionId,
		})
		if err != nil {
			break
		}
		state := *getQueryExecOut.QueryExecution.Status.State
		if state == athena.QueryExecutionStateSucceeded {
			glog.Infof("Query %s succeeded", *input.QueryString)
			break
		} else if state == athena.QueryExecutionStateFailed {
			reason := *getQueryExecOut.QueryExecution.Status.StateChangeReason
			glog.Errorf("Query %s failed %+v", *input.QueryString, reason)
			err = fmt.Errorf("Query %s failed: %s", *input.QueryString, reason)
			break

		} else if state == athena.QueryExecutionStateCancelled {
			reason := *getQueryExecOut.QueryExecution.Status.StateChangeReason
			glog.Errorf("Query %s cancelled %+v", *input.QueryString, reason)
			err = fmt.Errorf("Query %s cancelled: %s", *input.QueryString, reason)
			break
		}
	}
	return err
}

func (uploader *AthenaDataUploader) s3Location(tableName string) string {
	return fmt.Sprintf("s3://%s/%s/%s", *uploader.config.S3Bucket, *uploader.config.S3Prefix, tableName)
}

func (uploader *AthenaDataUploader) outputS3Location(tableName string) string {
	return fmt.Sprintf("s3://%s/%s/output/%s", *uploader.config.S3Bucket, *uploader.config.S3Prefix, tableName)
}

func (uploader *AthenaDataUploader) athenaTableName(tableName string) string {
	if uploader.config.AthenaTableSuffix == nil || *uploader.config.AthenaTableSuffix == "" {
		return tableName
	}
	return fmt.Sprintf("%s_%s", tableName, *uploader.config.AthenaTableSuffix)
}

// createAthenaTableIfMissing creates Athena table if missing
func (uploader *AthenaDataUploader) createAthenaTableIfMissing(ctx context.Context, tableName string) error {
	athenaClient := athena.New(uploader.awsSession)
	athenaTableName := uploader.athenaTableName(tableName)
	glog.Infof("Getting metadata for Athena table %s...", athenaTableName)
	getTableMetadataInput := &athena.GetTableMetadataInput{
		CatalogName:  uploader.config.AthenaCatalog,
		DatabaseName: uploader.config.AthenaDatabase,
		TableName:    aws.String(athenaTableName),
	}
	getTableMetadata, err := athenaClient.GetTableMetadataWithContext(ctx, getTableMetadataInput)
	if err == nil {
		tableMetadata := getTableMetadata.TableMetadata
		ba, _ := json.MarshalIndent(tableMetadata, " ", " ")
		glog.Infof("Metadata for Athena table %s: \n %+v", athenaTableName, string(ba))
		return nil
	}
	glog.Errorf("Failed to get metadata for Athena table %s. Error: %s", athenaTableName, err.Error())
	aerr, ok := err.(awserr.Error)
	if !ok {
		return err
	}
	if !strings.Contains(aerr.Message(), fmt.Sprintf("Table %s not found", athenaTableName)) {
		return err
	}
	tableMetadata, err := uploader.getTableMetadata(ctx, tableName)
	if err != nil {
		return err
	}
	s3Location := uploader.s3Location(tableName)
	outputS3Location := uploader.outputS3Location(tableName)
	query, err := tableMetadata.generateAthenaCreateTableQuery(athenaTableName, s3Location)
	if err != nil {
		return err
	}
	glog.Infof("Create Athena table query: %s", query)
	err = uploader.ExecuteAthenaQuery(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		QueryExecutionContext: &athena.QueryExecutionContext{
			Catalog:  uploader.config.AthenaCatalog,
			Database: uploader.config.AthenaDatabase,
		},
		ResultConfiguration: &athena.ResultConfiguration{
			OutputLocation: aws.String(outputS3Location),
		},
	})
	if err != nil {
		glog.Errorf("Failed to create Athena table %s. Error: %s", athenaTableName, err.Error())
		return err
	}
	glog.Infof("Created Athena table %s", athenaTableName)
	return nil
}

// fetchRecords queries the table and passes the record to the callback processor
func (uploader *AthenaDataUploader) fetchRecords(ctx context.Context, createdBy time.Time, tableName string, processor func(context.Context, *Record) error) error {
	tableMetadata, err := uploader.getTableMetadata(ctx, tableName)
	if err != nil {
		return err
	}
	query, err := tableMetadata.generateDBSelectQuery()
	if err != nil {
		return err
	}
	filter := QueryFilter{CreatedAt: createdBy}
	glog.Infof("Running query: %s", query)
	rows, err := uploader.dbClient.GetDB().NamedQuery(query, filter)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		data := map[string]interface{}{}
		err = rows.MapScan(data)
		if err != nil {
			return err
		}
		record := &Record{Name: tableMetadata.TableName}
		for i := range tableMetadata.ColumnMetadatas {
			columnMetadata := tableMetadata.ColumnMetadatas[i]
			value := data[columnMetadata.ColumnName]
			record.Columns = append(record.Columns, &Column{ColumnMetadata: *columnMetadata, Value: value})
		}
		err = processor(ctx, record)
		if err != nil {
			glog.Errorf("Error in processing record %+v. Error: %s", record, err.Error())
			return err
		}
	}
	return nil
}

// syncTable repairs the partitions of the given table
func (uploader *AthenaDataUploader) syncTable(ctx context.Context, tableName string, datePathComponent string) error {
	athenaTableName := uploader.athenaTableName(tableName)
	parts := strings.Split(datePathComponent, "/")
	partitions := make([]string, len(parts))
	for i, part := range parts {
		// year=2020
		pair := strings.Split(part, "=")
		partitions[i] = fmt.Sprintf("%s='%s'", pair[0], pair[1])
	}
	query := fmt.Sprintf("ALTER TABLE %s ADD PARTITION(%s)", athenaTableName, strings.Join(partitions, ", "))
	outputS3Location := uploader.outputS3Location(tableName)
	err := uploader.ExecuteAthenaQuery(ctx, &athena.StartQueryExecutionInput{
		QueryString: aws.String(query),
		QueryExecutionContext: &athena.QueryExecutionContext{
			Catalog:  uploader.config.AthenaCatalog,
			Database: uploader.config.AthenaDatabase,
		},
		ResultConfiguration: &athena.ResultConfiguration{
			OutputLocation: aws.String(outputS3Location),
		},
	})
	return err
}

// scanTables reads table name from the channel, queries the table and writes the records to the out channel.
// Error responses are written to the response channel
func (uploader *AthenaDataUploader) scanTables(ctx context.Context, createdBy time.Time, tableNames <-chan string, out chan<- *Record, res chan<- error) {
	defer func() {
		// Drain channel if all go-routines return with error
		for range tableNames {
		}
	}()
	for tableName := range tableNames {
		glog.Infof("Processing table %s...", tableName)
		err := uploader.createAthenaTableIfMissing(ctx, tableName)
		if err != nil {
			res <- err
			return
		}
		glog.Infof("Fetching record from %s...", tableName)
		// execute in go routine
		err = uploader.fetchRecords(ctx, createdBy, tableName, func(ctx context.Context, record *Record) error {
			if record != nil {
				out <- record
			}
			return nil
		})
		if err != nil {
			res <- err
			return
		}
	}

	res <- nil
}

// uploadS3Objects creates an S3 BatchUploadIterator with data from the input channel.
// Error responses are written to the response channel
func (uploader *AthenaDataUploader) uploadS3Objects(ctx context.Context, createdAt time.Time, in <-chan *Record, res chan<- error) {
	defer func() {
		// Drain channel if all go-routines return with error
		for range in {
		}
	}()
	s3Uploader := s3manager.NewUploader(uploader.awsSession)
	s3UploaderIterator := NewS3UploadIterator(ctx, createdAt, uploader.config.S3Bucket, uploader.config.S3Prefix, in)
	err := s3Uploader.UploadWithIterator(aws.BackgroundContext(), s3UploaderIterator)
	if err != nil {
		res <- err
		return
	}
	res <- nil
}

package pkg

import (
	"cloudservices/common/base"
	"flag"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for tenantpool service
type ConfigData struct {
	SQLDialect          *string
	SQLHost             *string
	SQLPort             *int
	SQLUser             *string
	SQLPassword         *string
	SQLDB               *string
	DBURL               *string
	AWSRegion           *string
	S3Bucket            *string
	S3Prefix            *string
	AthenaCatalog       *string
	AthenaDatabase      *string
	AthenaTableSuffix   *string
	DBScanConcurrency   *int
	S3UploadConcurrency *int
	TableNames          *base.MultiValue
}

// Cfg is the global config
var (
	Cfg = &ConfigData{}
)

func init() {
	// Default values
	Cfg.SQLDialect = base.StringPtr(base.GetEnvWithDefault("SQL_DIALECT", "postgres"))
	Cfg.SQLHost = base.StringPtr(base.GetEnvWithDefault("SQL_HOST", "sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com"))
	Cfg.SQLPort = base.IntPtr(base.GetEnvIntWithDefault("SQL_PORT", 5432))
	Cfg.SQLUser = base.StringPtr(base.GetEnvWithDefault("SQL_USER", "root"))
	Cfg.SQLPassword = base.StringPtr(base.GetEnvWithDefault("SQL_PASSWORD", "****"))
	Cfg.SQLDB = base.StringPtr(base.GetEnvWithDefault("SQL_DB", "sherlock_test"))
	Cfg.AWSRegion = base.StringPtr(base.GetEnvWithDefault("AWS_REGION", "us-west-2"))
	Cfg.S3Bucket = base.StringPtr(base.GetEnvWithDefault("ATHENA_S3_BUCKET", "sherlock-dev-db-athena-us-west-2"))
	Cfg.S3Prefix = base.StringPtr(base.GetEnvWithDefault("ATHENA_S3_PREFIX", "test"))
	Cfg.AthenaCatalog = base.StringPtr(base.GetEnvWithDefault("ATHENA_CATALOG", "AwsDataCatalog"))
	Cfg.AthenaDatabase = base.StringPtr(base.GetEnvWithDefault("ATHENA_DB", "sherlock_dev"))
	Cfg.AthenaTableSuffix = base.StringPtr(base.GetEnvWithDefault("ATHENA_TABLE_SUFFIX", ""))

	Cfg.DBScanConcurrency = base.IntPtr(2)
	Cfg.S3UploadConcurrency = base.IntPtr(4)
	Cfg.TableNames = &base.MultiValue{}
}

// LoadFlag parses the command line parameters
func (configData *ConfigData) LoadFlag() {
	configData.SQLDialect = flag.String("sql_dialect", *Cfg.SQLDialect, "<DB dialect>")
	configData.SQLHost = flag.String("sql_host", *Cfg.SQLHost, "<DB host>")
	configData.SQLPort = flag.Int("sql_port", *Cfg.SQLPort, "<DB port>")
	configData.SQLUser = flag.String("sql_user", *Cfg.SQLUser, "<DB user name>")
	configData.SQLPassword = flag.String("sql_password", *Cfg.SQLPassword, "<DB password>")
	configData.SQLDB = flag.String("sql_db", *Cfg.SQLDB, "<DB database>")
	configData.AWSRegion = flag.String("aws_region", *Cfg.AWSRegion, "<AWS region>")
	configData.S3Bucket = flag.String("aws_s3_bucket", *Cfg.S3Bucket, "<AWS S3 bucket>")
	configData.S3Prefix = flag.String("aws_s3_prefix", *Cfg.S3Prefix, "<AWS S3 prefix>")
	configData.AthenaCatalog = flag.String("aws_athena_catalog", *Cfg.AthenaCatalog, "<AWS Athena catalog>")
	configData.AthenaDatabase = flag.String("aws_athena_db", *Cfg.AthenaDatabase, "<AWS Athena database>")
	configData.AthenaTableSuffix = flag.String("aws_athena_table_suffix", *Cfg.AthenaTableSuffix, "<AWS Athena table suffix>")
	configData.DBScanConcurrency = flag.Int("db_scan_concurrency", *Cfg.DBScanConcurrency, "<DB scan concurrency>")
	configData.S3UploadConcurrency = flag.Int("s3_upload_concurrency", *Cfg.S3UploadConcurrency, "<S3 scan concurrency>")
	flag.Var(configData.TableNames, "table", "<DB table to be scanned>")
	flag.Parse()
	if glog.V(5) {
		spew.Dump(*Cfg)
	}
}

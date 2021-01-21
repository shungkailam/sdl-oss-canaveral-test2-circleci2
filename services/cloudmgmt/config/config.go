package config

import (
	"cloudservices/common/base"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for cloudmgmt
type ConfigData struct {
	ContentDir       *string
	Port             *int
	SQL_Dialect      *string
	SQL_Host         *string
	SQL_ReadOnlyHost *string
	SQL_Port         *int
	SQL_User         *string
	SQL_Password     *string
	SQL_DB           *string
	SQL_MaxIdleCnx   *int
	SQL_MaxCnx       *int
	SQL_MaxCnxLife   *time.Duration
	// KMS encrypted version of jwt secret
	JWTSecret *string
	AWSRegion *string
	AWSKMSKey *string
	// Environment
	Environment *string
	// OAuth2
	ClientID         *string
	ClientSecret     *string
	IdentityProvider *string
	RedirectURLs     *base.MultiValue
	EnableXiIoTRole  *bool
	EnableTrial      *bool
	// Enable/disable automatic tenant creation when trial is disabled.
	// e.g  free tier without any trial edge
	EnableTenantCreation *bool

	// OTA key
	OtaAccessKey *string
	OtaSecretKey *string
	// Cfssl Host
	CfsslHost *string
	// Cfssl Port
	CfsslPort *int
	// Cfssl Protocol
	CfsslProtocol *string
	// Support bundle S3 bucket
	LogS3Bucket *string
	// Static file S3 bucket
	FileS3Bucket *string
	// Redis Host
	RedisHost *string
	// Disable redis-based scale out
	DisableScaleOut *bool
	// Prometheus
	PrometheusPort *int
	// Code Coverage
	EnableCodeCoverage *bool
	// Threshold after which email will be locked
	LoginFailureCountThreshold *int
	// Duration in seconds for each email lock
	LoginLockDurationSeconds *int
	// Max batch size for events
	EventsMaxBatchSize *int
	// Disable RDS based audit log
	DisableAuditLog *bool
	// Enable audit log of read request
	EnableAuditLogOfReadReq *bool
	// Enable audit log of put event
	EnableAuditLogOfPutEvent *bool
	// How many days to keep recent audit log tables
	KeepAuditLogTableDays *int
	// ML model S3 bucket
	MLModelS3Bucket *string

	// Enable origin selectors for applications
	EnableAppOriginSelectors *bool

	// Whether to disable DB SSL
	DisableDBSSL *bool

	// Engine for Object Storage
	// e.g., aws, minio, ntnx, ...
	ObjectStorageEngine *string
	MinioAccessKey      *string
	MinioSecretKey      *string
	MinioURL            *string

	// Whether to use KMS for encryption / decryption
	UseKMS *bool

	// Allow hardcode service type.
	// Supported values are IoT or PaaS.
	// By default, cloudmgmt will use endpoint DNS name
	// to determine service type.
	// E.g., the same cloudmgmt instance, if accessed
	// via iot.nutanix.com, will respond with service type IoT,
	// while accessed via paas.nutanix.com will respond with PaaS.
	// This config, if set, will force cloudmgmt to be
	// a fixed service type.
	ServiceType *string

	// Used to set the DNS names for non-u2 edges
	U2Route53AccessKey *string
	U2Route53SecretKey *string
	EdgeDNSDomain      *string

	// wstun-svc Host
	WstunHost *string
	SSHUser   *string

	// gRPC Port
	GRPCPort *int

	// proxy url base
	ProxyUrlBase *string

	// role that other account should assume to access route53 in dev
	// (dev account should not assume this)
	Route53CrossAccountRole *string
}

var Cfg *ConfigData = &ConfigData{}

var contentDir = fmt.Sprintf("%s/src/cloudservices/cloudmgmt/build/ui/dist", os.Getenv("GOPATH"))

func init() {
	fmt.Printf(">>> cloudmgmt.config.init {\n")
	Cfg.ContentDir = base.StringPtr(contentDir)
	Cfg.Port = base.IntPtr(8080)
	Cfg.SQL_Dialect = base.StringPtr("postgres")
	Cfg.SQL_Host = base.StringPtr("sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com")
	// Set to empty because we dont want to use the default which may point to a different environment
	Cfg.SQL_ReadOnlyHost = base.StringPtr("")
	Cfg.SQL_Port = base.IntPtr(5432)
	Cfg.SQL_User = base.StringPtr("root")
	Cfg.SQL_Password = base.StringPtr(os.Getenv("SQL_PASSWORD"))
	Cfg.SQL_DB = base.StringPtr(base.GetEnvWithDefault("SQL_DB", "sherlock_test"))
	Cfg.SQL_MaxIdleCnx = base.IntPtr(0)
	Cfg.SQL_MaxCnx = base.IntPtr(0)
	Cfg.SQL_MaxCnxLife = base.DurationPtr(0)

	Cfg.JWTSecret = base.StringPtr(os.Getenv("JWT_SECRET"))
	Cfg.AWSRegion = base.StringPtr("us-west-2")
	Cfg.AWSKMSKey = base.StringPtr("alias/ntnx/cloudmgmt-dev")

	// OAuth2 security
	Cfg.ClientID = base.StringPtr(os.Getenv("OAUTH_CLIENT_ID"))
	Cfg.ClientSecret = base.StringPtr(os.Getenv("OAUTH_CLIENT_SECRET"))
	Cfg.IdentityProvider = base.StringPtr("https://idp-dev.nutanix.com")
	// Slice of <RedirectURL?clientId=<clientID>&clientSecret=<clientSecret>
	Cfg.RedirectURLs = &base.MultiValue{}
	Cfg.EnableXiIoTRole = base.BoolPtr(true)
	Cfg.EnableTrial = base.BoolPtr(false)
	Cfg.EnableTenantCreation = base.BoolPtr(true)

	//OTA
	Cfg.OtaAccessKey = base.StringPtr("sample_key")
	Cfg.OtaSecretKey = base.StringPtr("ota_secret_key")

	// Cfssl
	Cfg.CfsslHost = base.StringPtr(base.GetEnvWithDefault("CFSSL_HOST", "cfsslserver-svc"))
	Cfg.CfsslPort = base.IntPtr(base.GetEnvIntWithDefault("CFSSL_PORT", 8888))
	Cfg.CfsslProtocol = base.StringPtr(base.GetEnvWithDefault("CFSSL_PROTOCOL", "http"))

	// Support bundle S3 bucket
	Cfg.LogS3Bucket = base.StringPtr("sherlock-support-bundle-us-west-2")

	Cfg.FileS3Bucket = base.StringPtr("sherlock-static-files-dev")

	// Redis
	Cfg.RedisHost = base.StringPtr("redis-svc")
	Cfg.DisableScaleOut = base.BoolPtr(false)

	// Prometheus
	Cfg.PrometheusPort = base.IntPtr(9436)

	// Code Coverage
	Cfg.EnableCodeCoverage = base.BoolPtr(false)

	// login failure count threshold
	Cfg.LoginFailureCountThreshold = base.IntPtr(3)
	// login lock duration seconds
	Cfg.LoginLockDurationSeconds = base.IntPtr(60 * 30)
	// Max batch size for events
	Cfg.EventsMaxBatchSize = base.IntPtr(30)
	// audit log
	Cfg.DisableAuditLog = base.BoolPtr(false)
	Cfg.EnableAuditLogOfReadReq = base.BoolPtr(false)
	Cfg.EnableAuditLogOfPutEvent = base.BoolPtr(false)
	Cfg.KeepAuditLogTableDays = base.IntPtr(7)

	// ML Model
	Cfg.MLModelS3Bucket = base.StringPtr("sherlock-ml-models-dev")

	// App origin selectors
	Cfg.EnableAppOriginSelectors = base.BoolPtr(false)

	// Whether to disable DB SSL
	Cfg.DisableDBSSL = base.BoolPtr(false)

	Cfg.ObjectStorageEngine = base.StringPtr("aws")
	Cfg.MinioAccessKey = base.StringPtr("minio_access_key")
	Cfg.MinioSecretKey = base.StringPtr("minio_secret_key")
	Cfg.MinioURL = base.StringPtr("minio_url")

	// Whether to use KMS for encryption / decryption
	Cfg.UseKMS = base.BoolPtr(true)

	Cfg.ServiceType = base.StringPtr("")

	Cfg.U2Route53AccessKey = base.StringPtr("u2route53_access_key")
	Cfg.U2Route53SecretKey = base.StringPtr("u2route53_secret_key")
	Cfg.EdgeDNSDomain = base.StringPtr("edge_dns_domain")

	// wstun-svc
	Cfg.WstunHost = base.StringPtr("wstun-svc")
	Cfg.SSHUser = base.StringPtr("kubeuser")

	// gRPC
	Cfg.GRPCPort = base.IntPtr(9283)

	// proxy url base
	Cfg.ProxyUrlBase = base.StringPtr("https://wst-test.ntnxsherlock.com")

	Cfg.Route53CrossAccountRole = base.StringPtr("")
}

// LoadFlag parses the command line parameters
func (configData *ConfigData) LoadFlag() {

	Cfg.ContentDir = flag.String("contentdir", *Cfg.ContentDir, "<full path to HTML content dir>")

	// Default is PostgreSQL
	Cfg.SQL_Dialect = flag.String("sql_dialect", *Cfg.SQL_Dialect, "<DB dialect>")
	Cfg.SQL_Host = flag.String("sql_host", *Cfg.SQL_Host, "<DB host>")
	Cfg.SQL_ReadOnlyHost = flag.String("sql_ro_host", *Cfg.SQL_ReadOnlyHost, "<DB read only host>")
	Cfg.SQL_Port = flag.Int("sql_port", *Cfg.SQL_Port, "<DB port>")
	Cfg.SQL_User = flag.String("sql_user", *Cfg.SQL_User, "<DB user name>")
	Cfg.SQL_Password = flag.String("sql_password", *Cfg.SQL_Password, "<DB password>")
	Cfg.SQL_DB = flag.String("sql_db", *Cfg.SQL_DB, "<DB database>")
	Cfg.SQL_MaxIdleCnx = flag.Int("sql_max_idle_connection", *Cfg.SQL_MaxIdleCnx, "<DB max idle connection count>")
	Cfg.SQL_MaxCnx = flag.Int("sql_max_connection", *Cfg.SQL_MaxCnx, "<DB max connection count>")
	Cfg.SQL_MaxCnxLife = flag.Duration("sql_max_connection_life", *Cfg.SQL_MaxCnxLife, "<DB max connection life>")

	Cfg.JWTSecret = flag.String("jwtsecret", *Cfg.JWTSecret, "<JWT secret>")
	Cfg.AWSRegion = flag.String("awsregion", *Cfg.AWSRegion, "<AWS region>")
	Cfg.AWSKMSKey = flag.String("kmskey", *Cfg.AWSKMSKey, "<KMS Key URN or Alias>")
	Cfg.Port = flag.Int("port", *Cfg.Port, "<cloudmgmt server port>")

	// OAuth2 security
	Cfg.ClientID = flag.String("auth_client_id", *Cfg.ClientID, "<Auth Client ID>")
	Cfg.ClientSecret = flag.String("auth_client_secret", *Cfg.ClientSecret, "<Auth Client Secret>")
	Cfg.IdentityProvider = flag.String("auth_idp", *Cfg.IdentityProvider, "<Auth Identity Provider>")
	// RedirectURLs is a slice of URLs
	flag.Var(Cfg.RedirectURLs, "auth_redirect_url", "<Auth Redirect URL And Client Details>")
	// workaround as helm does not support boolean values like yes/no, true/false even with quotes
	xiIoTRoleCmd := flag.String("enable_xi_iot_role", "enable", "<Enable Xi IoT Role>")
	// workaround as helm does not support boolean values like yes/no, true/false even with quotes
	trialCmd := flag.String("enable_trial", "disable", "<Enable Trial>")
	// workaround as helm does not support boolean values like yes/no, true/false even with quotes
	tenantCreationCmd := flag.String("enable_tenant_creation", "enable", "<Enable Tenant Creation for Non-Trials")

	//OTA
	Cfg.OtaAccessKey = flag.String("ota_access_key", *Cfg.OtaAccessKey, "<Operator port>")
	Cfg.OtaSecretKey = flag.String("ota_secret_key", *Cfg.OtaSecretKey, "<ip>")

	// Cfssl
	Cfg.CfsslHost = flag.String("cfsslhost", *Cfg.CfsslHost, "<cfssl host>")
	Cfg.CfsslPort = flag.Int("cfsslport", *Cfg.CfsslPort, "<cfssl port>")
	Cfg.CfsslProtocol = flag.String("cfsslprotocol", *Cfg.CfsslProtocol, "<cfssl protocol>")

	// Support bundle S3 bucket
	Cfg.LogS3Bucket = flag.String("log_s3_bucket", *Cfg.LogS3Bucket, "<Support bundle S3 bucket>")

	// Static file S3 bucket
	Cfg.FileS3Bucket = flag.String("file_s3_bucket", *Cfg.FileS3Bucket, "<Static file S3 bucket>")

	// Redis
	Cfg.RedisHost = flag.String("redishost", *Cfg.RedisHost, "<redis host>")
	Cfg.DisableScaleOut = flag.Bool("disable_scaleout", *Cfg.DisableScaleOut, "disable scaleout")

	// Prometheus
	Cfg.PrometheusPort = flag.Int("prometheus_port", *Cfg.PrometheusPort, "<cloudmgmt prometheus metrics port>")

	// Code Coverage
	Cfg.EnableCodeCoverage = flag.Bool("enableCodeCoverage", false, "Set to true to enable coverage")

	// login failure count threshold
	Cfg.LoginFailureCountThreshold = flag.Int("login_failure_threshold", *Cfg.LoginFailureCountThreshold, "<login failure count threshold>")
	// login lock duration seconds
	Cfg.LoginLockDurationSeconds = flag.Int("login_lock_duration", *Cfg.LoginLockDurationSeconds, "<login lock duration seconds>")
	// Max batch size for events
	Cfg.EventsMaxBatchSize = flag.Int("events_batch_size", *Cfg.EventsMaxBatchSize, "<batch size for events>")

	// audit log
	Cfg.DisableAuditLog = flag.Bool("disable_audit_log", *Cfg.DisableAuditLog, "disable audit log")
	Cfg.EnableAuditLogOfReadReq = flag.Bool("enable_audit_log_of_read_req", *Cfg.EnableAuditLogOfReadReq, "enable audit log of read request")
	Cfg.EnableAuditLogOfPutEvent = flag.Bool("enable_audit_log_of_put_event", *Cfg.EnableAuditLogOfPutEvent, "enable audit log of put event")
	Cfg.KeepAuditLogTableDays = flag.Int("keep_audit_log_table_days", *Cfg.KeepAuditLogTableDays, "days of audit log tables to keep")

	// ML model S3 bucket
	Cfg.MLModelS3Bucket = flag.String("ml_model_s3_bucket", *Cfg.MLModelS3Bucket, "<ML Model S3 bucket>")

	// App origin selectors
	Cfg.EnableAppOriginSelectors = flag.Bool("enable_app_origin_selectors", *Cfg.EnableAppOriginSelectors, "enable application origin selectors")

	// Whether to disable DB SSL
	Cfg.DisableDBSSL = flag.Bool("disable_db_ssl", *Cfg.DisableDBSSL, "disable DB SSL")

	Cfg.ObjectStorageEngine = flag.String("object_storage_engine", *Cfg.ObjectStorageEngine, "<Object Storage Engine>")
	// Minio
	Cfg.MinioAccessKey = flag.String("minio_access_key", *Cfg.MinioAccessKey, "<minio access key>")
	Cfg.MinioSecretKey = flag.String("minio_secret_key", *Cfg.MinioSecretKey, "<minio secret key>")
	Cfg.MinioURL = flag.String("minio_url", *Cfg.MinioURL, "<minio url>")

	Cfg.UseKMS = flag.Bool("use_kms", *Cfg.UseKMS, "use KMS encryption/decryption")

	Cfg.ServiceType = flag.String("service_type", *Cfg.ServiceType, "hardcode ServiceType: IoT or PaaS")

	Cfg.U2Route53AccessKey = flag.String("u2_route53_access_key", *Cfg.U2Route53AccessKey, "u2 route53 access key")
	Cfg.U2Route53SecretKey = flag.String("u2_route53_secret_key", *Cfg.U2Route53SecretKey, "u2 route53 secret key")
	Cfg.EdgeDNSDomain = flag.String("edge_dns_domain", *Cfg.EdgeDNSDomain, "edge dns domain")

	// wstun-svc
	Cfg.WstunHost = flag.String("wstunhost", *Cfg.WstunHost, "<wstun host>")
	Cfg.SSHUser = flag.String("sshuser", *Cfg.SSHUser, "<ssh user>")

	// gRPC
	Cfg.GRPCPort = flag.Int("grpc_port", *Cfg.GRPCPort, "<cloudmgmt grpc port>")

	// proxy url base
	Cfg.ProxyUrlBase = flag.String("proxy_url_base", *Cfg.ProxyUrlBase, "Base URL for service proxy")

	Cfg.Route53CrossAccountRole = flag.String("route53_cross_account_role", *Cfg.Route53CrossAccountRole, "role other account should use to access dev route53, should not be set in dev")

	// parse must be done before sprintf below
	// otherwise DBURL would be incorrect
	flag.Parse()

	if glog.V(5) {
		spew.Dump(*Cfg)
	}

	// bias towards enabling xi-iot role unless explicitly disabled
	Cfg.EnableXiIoTRole = base.BoolPtr((xiIoTRoleCmd == nil) || strings.ToLower(*xiIoTRoleCmd) != "disable")

	// bias towards disabling trial unless explicitly enabled
	Cfg.EnableTrial = base.BoolPtr((trialCmd != nil) && strings.ToLower(*trialCmd) == "enable")

	// bias towards disabling tenant creation unless explicitly enabled
	Cfg.EnableTenantCreation = base.BoolPtr((tenantCreationCmd != nil) && strings.ToLower(*tenantCreationCmd) == "enable")

	// Fix old redirectURL for backward compatibility
	redirectURLs := base.MultiValue{}
	for _, redirectURL := range Cfg.RedirectURLs.Values() {
		if !strings.Contains(redirectURL, "clientId") && !strings.Contains(redirectURL, "clientSecret") {
			redirectURLs.Set(fmt.Sprintf("%s?clientId=%s&clientSecret=%s", redirectURL, *Cfg.ClientID, *Cfg.ClientSecret))
		} else {
			redirectURLs.Set(redirectURL)
		}
	}
	Cfg.RedirectURLs = &redirectURLs

	glog.Infof("config { init using server port: %d\n", *Cfg.Port)
}

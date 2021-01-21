package config

import (
	"cloudservices/common/base"
	"cloudservices/common/service"
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for the cloud operator
type ConfigData struct {
	ContentDir *string
	Port       *int
	GrpcPort   *int

	SQLDialect      *string
	SQLHost         *string
	SQLReadOnlyHost *string
	SQLPort         *int
	SQLUser         *string
	SQLPassword     *string
	SQLDB           *string
	DBURL           *string

	// KMS encrypted version of jwt secret
	JWTSecret *string
	AWSRegion *string
	AWSKMSKey *string

	// S3 config
	S3Bucket             *string
	S3Region             *string
	S3Prefix             *string
	ReleaseHelmChartRepo *string

	// Engine for Object Storage
	// e.g., aws, minio, ntnx, ...
	ObjectStorageEngine *string
	MinioAccessKey      *string
	MinioSecretKey      *string
	MinioURL            *string

	// e.g aws, private, gcp
	DockerRegistryProvider *string
	DockerRegistryUser     *string
	DockerRegistryPassword *string
	DockerRegistryURL      *string

	// Release helm chart version
	ReleaseHelmChartVersion *string

	// Access key for the AWS ECR repository
	OTAAccessKey *string
	OTASecretKey *string

	// AWS federated token lifetime in minutes
	AWSFederatedTokenLifetimeMins *int

	// Code Coverage
	EnableCodeCoverage *bool

	// Whether to use KMS for encryption / decryption
	UseKMS *bool

	// Whether to disable DB SSL
	DisableDBSSL *bool
}

var Cfg *ConfigData = &ConfigData{}

func init() {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		glog.Fatal(err)
	}
	contentDir := dir + "/templates"

	Cfg.ContentDir = base.StringPtr(contentDir)

	serviceConfig := service.GetServiceConfig(service.OperatorService)
	Cfg.GrpcPort = base.IntPtr(serviceConfig.Port)

	// REST service port
	Cfg.Port = base.IntPtr(9000)

	// SQL database
	Cfg.SQLDialect = base.StringPtr("postgres")
	Cfg.SQLHost = base.StringPtr("sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com")
	// Set to empty because we dont want to use the default which may point to a different environment
	Cfg.SQLReadOnlyHost = base.StringPtr("")
	Cfg.SQLPort = base.IntPtr(5432)
	Cfg.SQLUser = base.StringPtr("root")
	Cfg.SQLPassword = base.StringPtr(os.Getenv("SQL_PASSWORD"))
	Cfg.SQLDB = base.StringPtr(base.GetEnvWithDefault("SQL_DB", "sherlock_test"))

	// S3 config
	Cfg.S3Bucket = base.StringPtr("sherlock-dev-releases")
	Cfg.S3Prefix = base.StringPtr("operator-test")
	Cfg.ReleaseHelmChartRepo = base.StringPtr("http://my-release-helm-chart-repo")
	Cfg.S3Region = base.StringPtr("us-west-2")

	// Object store config
	Cfg.ObjectStorageEngine = base.StringPtr("aws")
	Cfg.MinioAccessKey = base.StringPtr("minio_access_key")
	Cfg.MinioSecretKey = base.StringPtr("minio_secret_key")
	Cfg.MinioURL = base.StringPtr("minio_url")

	// Docker registry config
	Cfg.DockerRegistryProvider = base.StringPtr("aws")
	Cfg.DockerRegistryUser = base.StringPtr("docker_registry_user")
	Cfg.DockerRegistryPassword = base.StringPtr("docker_registry_password")
	Cfg.DockerRegistryURL = base.StringPtr("docker_registry_url")

	// Release helm chart version
	Cfg.ReleaseHelmChartVersion = base.StringPtr("v1.0.0")

	// For ECR repository access
	Cfg.OTAAccessKey = base.StringPtr("ota-access-key")
	Cfg.OTASecretKey = base.StringPtr("ota-secret-key")

	Cfg.JWTSecret = base.StringPtr(os.Getenv("JWT_SECRET"))
	Cfg.AWSRegion = base.StringPtr("us-west-2")
	Cfg.AWSKMSKey = base.StringPtr("alias/ntnx/cloudmgmt-dev")

	// AWS federated token lifetime in minutes
	Cfg.AWSFederatedTokenLifetimeMins = base.IntPtr(60)

	// Code Coverage
	Cfg.EnableCodeCoverage = base.BoolPtr(false)

	// Add seperator for folders
	if *Cfg.S3Prefix != "" {
		*Cfg.S3Prefix = *Cfg.S3Prefix + "/"
	}

	if *Cfg.ReleaseHelmChartRepo != "" {
		*Cfg.ReleaseHelmChartRepo = path.Clean(*Cfg.ReleaseHelmChartRepo)
	}

	// Whether to use KMS for encryption / decryption
	Cfg.UseKMS = base.BoolPtr(true)

	// Whether to disable DB SSL
	Cfg.DisableDBSSL = base.BoolPtr(false)
}

// LoadFlag populates the config values from command line
func (configData *ConfigData) LoadFlag() {
	Cfg.ContentDir = flag.String("contentdir", *Cfg.ContentDir, "<full path to HTML content dir>")

	Cfg.Port = flag.Int("port", *Cfg.Port, "<operator server port>")
	Cfg.GrpcPort = flag.Int("rpcport", *Cfg.GrpcPort, "<operator server rpcport>")

	// SQL database
	Cfg.SQLDialect = flag.String("sql_dialect", *Cfg.SQLDialect, "<DB dialect>")
	Cfg.SQLHost = flag.String("sql_host", *Cfg.SQLHost, "<DB host>")
	Cfg.SQLReadOnlyHost = flag.String("sql_ro_host", *Cfg.SQLReadOnlyHost, "<DB read only host>")
	Cfg.SQLPort = flag.Int("sql_port", *Cfg.SQLPort, "<DB port>")
	Cfg.SQLUser = flag.String("sql_user", *Cfg.SQLUser, "<DB user name>")
	Cfg.SQLPassword = flag.String("sql_password", *Cfg.SQLPassword, "<DB password>")
	Cfg.SQLDB = flag.String("sql_db", *Cfg.SQLDB, "<DB database>")

	// S3 config
	Cfg.S3Bucket = flag.String("s3bucket", *Cfg.S3Bucket, "<releases s3 bucket>")
	Cfg.S3Prefix = flag.String("s3prefix", *Cfg.S3Prefix, "<folder within s3 bucket>")
	Cfg.S3Region = flag.String("s3region", *Cfg.S3Region, "<releases s3 bucket region>")

	// Object storage config
	Cfg.ObjectStorageEngine = flag.String("object_storage_engine", *Cfg.ObjectStorageEngine, "<Object storage engine>")
	// Minio config
	Cfg.MinioAccessKey = flag.String("minio_access_key", *Cfg.MinioAccessKey, "<Minio access key>")
	Cfg.MinioSecretKey = flag.String("minio_secret_key", *Cfg.MinioSecretKey, "<Minio secret key>")
	Cfg.MinioURL = flag.String("minio_url", *Cfg.MinioURL, "<minio url>")

	// Docker registry config
	Cfg.DockerRegistryProvider = flag.String("docker_registry_provider", *Cfg.DockerRegistryProvider, "<Docker registry provider>")
	Cfg.DockerRegistryUser = flag.String("docker_registry_user", *Cfg.DockerRegistryUser, "<Docker registry user>")
	Cfg.DockerRegistryPassword = flag.String("docker_registry_password", *Cfg.DockerRegistryPassword, "<Docker registry password>")
	Cfg.DockerRegistryURL = flag.String("docker_registry_url", *Cfg.DockerRegistryURL, "<Docker registry URL>")

	// Release helm chart info
	Cfg.ReleaseHelmChartRepo = flag.String("release_helm_chart_repo", *Cfg.ReleaseHelmChartRepo, "<Release helm chart repo>")
	Cfg.ReleaseHelmChartVersion = flag.String("release_helm_chart_version", *Cfg.ReleaseHelmChartVersion, "<Release helm chart version>")

	Cfg.JWTSecret = flag.String("jwtsecret", *Cfg.JWTSecret, "<JWT secret>")
	Cfg.AWSRegion = flag.String("awsregion", *Cfg.AWSRegion, "<AWS region>")
	Cfg.AWSKMSKey = flag.String("kmskey", *Cfg.AWSKMSKey, "<KMS Key URN or Alias>") // parse must be done before sprintf below

	Cfg.OTAAccessKey = flag.String("ota_access_key", *Cfg.OTAAccessKey, "<OTA AWS key>")
	Cfg.OTASecretKey = flag.String("ota_secret_key", *Cfg.OTASecretKey, "<OTA AWS secret>")

	Cfg.AWSFederatedTokenLifetimeMins = flag.Int("aws_token_lifetime_mins", *Cfg.AWSFederatedTokenLifetimeMins, "<AWS federated token lifetime in minutes>")

	// otherwise DBURL would be incorrect
	// Code Coverage
	Cfg.EnableCodeCoverage = flag.Bool("enableCodeCoverage", false, "Set to true to enable coverage")

	Cfg.UseKMS = flag.Bool("use_kms", *Cfg.UseKMS, "use KMS encryption/decryption")

	// Whether to disable DB SSL
	Cfg.DisableDBSSL = flag.Bool("disable_db_ssl", *Cfg.DisableDBSSL, "disable DB SSL")

	flag.Parse()

	// Add seperator for folders
	if *Cfg.S3Prefix != "" && !strings.HasSuffix(*Cfg.S3Prefix, "/") {
		*Cfg.S3Prefix = *Cfg.S3Prefix + "/"
	}
	if *Cfg.ReleaseHelmChartRepo != "" {
		*Cfg.ReleaseHelmChartRepo = path.Clean(*Cfg.ReleaseHelmChartRepo)
	}
	if glog.V(5) {
		spew.Dump(*Cfg)
	}
}

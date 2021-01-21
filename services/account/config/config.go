package config

import (
	"flag"
	"os"
	"time"

	"cloudservices/common/base"
	"cloudservices/common/service"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for cloudmgmt
type ConfigData struct {
	// KMS encrypted version of jwt secret
	JWTSecret *string
	AWSRegion *string
	AWSKMSKey *string

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
	DBURL            *string
	Port             *int
	Environment      *string
	// Prometheus
	PrometheusPort *int
	// Code Coverage
	EnableCodeCoverage *bool

	// Whether to disable DB SSL
	DisableDBSSL *bool

	// Whether to use KMS for encryption / decryption
	UseKMS *bool

	// Cfssl Host
	CfsslHost *string
	// Cfssl Port
	CfsslPort *int
	// Cfssl Protocol
	CfsslProtocol *string

	// Redis
	RedisHost *string
	// Disable redis-based scale out
	DisableScaleOut *bool
}

var Cfg *ConfigData = &ConfigData{}

func init() {
	// Default values
	serviceConfig := service.GetServiceConfig(service.AccountService)
	Cfg.Port = base.IntPtr(serviceConfig.Port)
	Cfg.JWTSecret = base.StringPtr(os.Getenv("JWT_SECRET"))
	Cfg.AWSRegion = base.StringPtr("us-west-2")
	Cfg.AWSKMSKey = base.StringPtr("alias/ntnx/cloudmgmt-dev")
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
	// Prometheus
	Cfg.PrometheusPort = base.IntPtr(9437)
	// Code Coverage
	Cfg.EnableCodeCoverage = base.BoolPtr(false)

	// Whether to disable DB SSL
	Cfg.DisableDBSSL = base.BoolPtr(false)

	// Whether to use KMS for encryption / decryption
	Cfg.UseKMS = base.BoolPtr(true)

	// Cfssl
	Cfg.CfsslHost = base.StringPtr(base.GetEnvWithDefault("CFSSL_HOST", "cfsslserver-svc"))
	Cfg.CfsslPort = base.IntPtr(base.GetEnvIntWithDefault("CFSSL_PORT", 8888))
	Cfg.CfsslProtocol = base.StringPtr(base.GetEnvWithDefault("CFSSL_PROTOCOL", "http"))

	// Redis
	Cfg.RedisHost = base.StringPtr("redis-svc")
	Cfg.DisableScaleOut = base.BoolPtr(false)
}

// LoadFlag populates the config values from command line
func (configData *ConfigData) LoadFlag() {
	Cfg.Port = flag.Int("port", *Cfg.Port, "<server port>")
	Cfg.JWTSecret = flag.String("jwtsecret", *Cfg.JWTSecret, "<JWT secret>")
	Cfg.AWSRegion = flag.String("awsregion", *Cfg.AWSRegion, "<AWS region>")
	Cfg.AWSKMSKey = flag.String("kmskey", *Cfg.AWSKMSKey, "<KMS Key URN or Alias>")
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
	// Prometheus
	Cfg.PrometheusPort = flag.Int("prometheus_port", *Cfg.PrometheusPort, "<prometheus metrics port>")
	// Code Coverage
	Cfg.EnableCodeCoverage = flag.Bool("enableCodeCoverage", false, "Set to true to enable coverage")

	// Whether to disable DB SSL
	Cfg.DisableDBSSL = flag.Bool("disable_db_ssl", *Cfg.DisableDBSSL, "disable DB SSL")

	Cfg.UseKMS = flag.Bool("use_kms", *Cfg.UseKMS, "use KMS encryption/decryption")

	// Cfssl
	Cfg.CfsslHost = flag.String("cfsslhost", *Cfg.CfsslHost, "<cfssl host>")
	Cfg.CfsslPort = flag.Int("cfsslport", *Cfg.CfsslPort, "<cfssl port>")
	Cfg.CfsslProtocol = flag.String("cfsslprotocol", *Cfg.CfsslProtocol, "<cfssl protocol>")

	// Redis
	Cfg.RedisHost = flag.String("redishost", *Cfg.RedisHost, "<redis host>")
	Cfg.DisableScaleOut = flag.Bool("disable_scaleout", *Cfg.DisableScaleOut, "disable scaleout")

	// parse must be done before sprintf below
	// otherwise DBURL would be incorrect
	flag.Parse()

	if glog.V(5) {
		spew.Dump(*Cfg)
	}

	glog.Infof("config init using server port: %d\n", *Cfg.Port)

}

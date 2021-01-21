package config

import (
	"flag"
	"os"
	"strings"
	"time"

	"cloudservices/common/base"
	"cloudservices/common/service"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for tenantpool service
type ConfigData struct {
	Port              *int
	SQLDialect        *string
	SQLHost           *string
	SQLReadOnlyHost   *string
	SQLPort           *int
	SQLUser           *string
	SQLPassword       *string
	SQLDB             *string
	DBURL             *string
	CloudmgmtEndpoint *string
	// K8s namespace in which the service is running
	// It is used to tag the edge instances
	Namespace            *string
	TenantPoolScanDelay  *time.Duration
	ReserveStateExpiry   *time.Duration
	EdgeProvisionTimeout *time.Duration
	EdgeDeletionTimeout  *time.Duration
	EnableScanner        *bool
	BottURL              *string
	AppChartVersion      *string

	// Tuning parameters for pool stats calculation
	EnablePoolStats           *bool
	PoolStatsWeight           *float64
	PoolStatsTimeFactor       *float64
	PoolStatsSamplingInterval *int
	PoolStatsDownsizeDelay    *int
	PoolStatsDownsizeLimit    *int

	// Redis
	RedisHost *string
	// Disable redis-based scale out
	DisableScaleOut *bool
}

var Cfg *ConfigData = &ConfigData{}

func init() {
	// Default values
	serviceConfig := service.GetServiceConfig(service.TenantPoolService)
	Cfg.Port = base.IntPtr(serviceConfig.Port)
	Cfg.SQLDialect = base.StringPtr("postgres")
	Cfg.SQLHost = base.StringPtr("sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com")

	// Set to empty because we dont want to use the default which may point to a different environment
	Cfg.SQLReadOnlyHost = base.StringPtr("")
	Cfg.SQLPort = base.IntPtr(5432)
	Cfg.SQLUser = base.StringPtr("root")
	Cfg.SQLPassword = base.StringPtr(os.Getenv("SQL_PASSWORD"))
	Cfg.SQLDB = base.StringPtr(base.GetEnvWithDefault("SQL_DB", "sherlock_test"))
	Cfg.CloudmgmtEndpoint = base.StringPtr("https://test.ntnxsherlock.com")
	Cfg.Namespace = base.StringPtr("test")

	// Unit in seconds to ease testing
	Cfg.TenantPoolScanDelay = base.DurationPtr(time.Minute)
	Cfg.EdgeProvisionTimeout = base.DurationPtr(time.Minute * 30)
	Cfg.EdgeDeletionTimeout = base.DurationPtr(time.Minute * 10)
	Cfg.ReserveStateExpiry = base.DurationPtr(time.Minute * 3)
	Cfg.EnableScanner = base.BoolPtr(true)
	Cfg.BottURL = base.StringPtr("https://bottdev.ntnxsherlock.com/v1")
	Cfg.AppChartVersion = base.StringPtr("0.24.0")

	// Tuning parameters for pool stats calculation
	Cfg.EnablePoolStats = base.BoolPtr(true)
	Cfg.PoolStatsWeight = base.Float64Ptr(0.4)
	Cfg.PoolStatsTimeFactor = base.Float64Ptr(5)
	Cfg.PoolStatsSamplingInterval = base.IntPtr(1)
	Cfg.PoolStatsDownsizeDelay = base.IntPtr(3)
	Cfg.PoolStatsDownsizeLimit = base.IntPtr(2)

	// Redis
	Cfg.RedisHost = base.StringPtr("redis-svc")
	Cfg.DisableScaleOut = base.BoolPtr(false)
}

// LoadFlag populates the config values from command line
func (configData *ConfigData) LoadFlag() {
	Cfg.Port = flag.Int("port", *Cfg.Port, "<Server port>")
	Cfg.SQLDialect = flag.String("sql_dialect", *Cfg.SQLDialect, "<DB dialect>")
	Cfg.SQLHost = flag.String("sql_host", *Cfg.SQLHost, "<DB host>")
	Cfg.SQLReadOnlyHost = flag.String("sql_ro_host", *Cfg.SQLReadOnlyHost, "<DB read only host>")
	Cfg.SQLPort = flag.Int("sql_port", *Cfg.SQLPort, "<DB port>")
	Cfg.SQLUser = flag.String("sql_user", *Cfg.SQLUser, "<DB user name>")
	Cfg.SQLPassword = flag.String("sql_password", *Cfg.SQLPassword, "<DB password>")
	Cfg.SQLDB = flag.String("sql_db", *Cfg.SQLDB, "<DB database>")
	Cfg.CloudmgmtEndpoint = flag.String("cloudmgmt_endpoint", *Cfg.CloudmgmtEndpoint, "<Cloudmgmt endpoint>")
	Cfg.Namespace = flag.String("namespace", *Cfg.Namespace, "<Deployment namespace>")
	// e.g 10m, 30m
	Cfg.TenantPoolScanDelay = flag.Duration("tenantpool_scan_delay", *Cfg.TenantPoolScanDelay, "<TenantPool scan delay>")
	Cfg.EdgeProvisionTimeout = flag.Duration("edge_provision_timeout", *Cfg.EdgeProvisionTimeout, "<Edge provisioning time-out>")
	Cfg.EdgeDeletionTimeout = flag.Duration("edge_deletion_timeout", *Cfg.EdgeDeletionTimeout, "<Edge deletion time-out>")
	Cfg.ReserveStateExpiry = flag.Duration("reserve_state_expiry", *Cfg.ReserveStateExpiry, "<Tenant claim reservation expiry>")
	Cfg.BottURL = flag.String("bott_url", *Cfg.BottURL, "<Bott URL>")
	Cfg.AppChartVersion = flag.String("app_chart_version", *Cfg.AppChartVersion, "<App Chart Version>")

	// workaround as helm does not support boolean values like yes/no, true/false even with quotes
	scannerCmd := flag.String("enable_scanner", "enable", "<Enable Scanner>")

	// workaround as helm does not support boolean values like yes/no, true/false even with quotes
	poolStatsCmd := flag.String("enable_poolstats", "enable", "<Enable Scanner>")

	Cfg.PoolStatsWeight = flag.Float64("poolstats_weight", *Cfg.PoolStatsWeight, "<Weight for moving exponential smoothing function>")
	Cfg.PoolStatsTimeFactor = flag.Float64("poolstats_time_factor", *Cfg.PoolStatsTimeFactor, "<Scaling time based multiplier for pool size>")
	Cfg.PoolStatsSamplingInterval = flag.Int("poolstats_sampling_interval", *Cfg.PoolStatsSamplingInterval, "<Sampling interval>")
	Cfg.PoolStatsDownsizeDelay = flag.Int("poolstats_downsize_delay", *Cfg.PoolStatsDownsizeDelay, "<Pool downsize delay")
	Cfg.PoolStatsDownsizeLimit = flag.Int("poolstats_downsize_limit", *Cfg.PoolStatsDownsizeLimit, "<Pool downsize count limit after the delay")

	// Redis
	Cfg.RedisHost = flag.String("redishost", *Cfg.RedisHost, "<redis host>")
	Cfg.DisableScaleOut = flag.Bool("disable_scaleout", *Cfg.DisableScaleOut, "disable scaleout")

	flag.Parse()

	if glog.V(5) {
		spew.Dump(*Cfg)
	}

	// bias towards enabling scanning unless explicitly disabled
	Cfg.EnableScanner = base.BoolPtr(scannerCmd == nil || strings.ToLower(*scannerCmd) != "disable")

	Cfg.EnablePoolStats = base.BoolPtr(poolStatsCmd == nil || strings.ToLower(*poolStatsCmd) != "disable")

	glog.Infof("config init using server port: %d\n", *Cfg.Port)

}

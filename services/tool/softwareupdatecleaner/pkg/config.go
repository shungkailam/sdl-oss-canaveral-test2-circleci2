package pkg

import (
	"cloudservices/common/base"
	"flag"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for tenantpool service
type ConfigData struct {
	SQLDialect     *string
	SQLHost        *string
	SQLPort        *int
	SQLUser        *string
	SQLPassword    *string
	SQLDB          *string
	ElapsedDays    *int
	MaximumBatches *int
	// Whether to disable DB SSL
	DisableDBSSL *bool
}

// Cfg is the global config
var Cfg *ConfigData = &ConfigData{}

func init() {
	// Default values
	Cfg.SQLDialect = base.StringPtr("postgres")
	Cfg.SQLHost = base.StringPtr("sherlock-pg-dev-cluster.cluster-cn6yw4qpwrhi.us-west-2.rds.amazonaws.com")
	Cfg.SQLPort = base.IntPtr(5432)
	Cfg.SQLUser = base.StringPtr("root")
	Cfg.SQLPassword = base.StringPtr(os.Getenv("SQL_PASSWORD"))
	Cfg.SQLDB = base.StringPtr("sherlock_test")
	Cfg.ElapsedDays = base.IntPtr(7)
	Cfg.MaximumBatches = base.IntPtr(3)
	// Whether to disable DB SSL
	Cfg.DisableDBSSL = base.BoolPtr(false)
}

// LoadFlag populates the config values from command line
func (configData *ConfigData) LoadFlag() {
	Cfg.SQLDialect = flag.String("sql_dialect", *Cfg.SQLDialect, "<DB dialect>")
	Cfg.SQLHost = flag.String("sql_host", *Cfg.SQLHost, "<DB host>")
	Cfg.SQLPort = flag.Int("sql_port", *Cfg.SQLPort, "<DB port>")
	Cfg.SQLUser = flag.String("sql_user", *Cfg.SQLUser, "<DB user name>")
	Cfg.SQLPassword = flag.String("sql_password", *Cfg.SQLPassword, "<DB password>")
	Cfg.SQLDB = flag.String("sql_db", *Cfg.SQLDB, "<DB database>")
	Cfg.ElapsedDays = flag.Int("elapsed_days", *Cfg.ElapsedDays, "<Elapsed days for batch>")
	Cfg.MaximumBatches = flag.Int("max_batches", *Cfg.ElapsedDays, "<Maximum number of batches>")
	// Whether to disable DB SSL
	Cfg.DisableDBSSL = flag.Bool("disable_db_ssl", *Cfg.DisableDBSSL, "disable DB SSL")

	flag.Parse()
	if glog.V(5) {
		spew.Dump(*Cfg)
	}
}

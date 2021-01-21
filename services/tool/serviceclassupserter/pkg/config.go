package pkg

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	cloumgmtconfig "cloudservices/cloudmgmt/config"
)

// ServiceClassUpserterConfig stores the configs for the Service Class upserter
type ServiceClassUpserterConfig struct {
	DataDir         string
	DeleteOnMissing bool
	DisableDryRun   bool
}

// Cfg is the singleton for ServiceClassUpserterConfig
var Cfg = &ServiceClassUpserterConfig{}

func init() {
	LoadConfig()
}

// LoadConfig loads the config values from the environment
func LoadConfig() {
	// Parse to avoid the warning of logging before parsing
	flag.Parse()
	Cfg.DataDir, _ = os.Getwd()
	Cfg.DeleteOnMissing = false
	Cfg.DisableDryRun = false

	sqlDialect, ok := os.LookupEnv("SQL_DIALECT")
	if ok {
		cloumgmtconfig.Cfg.SQL_Dialect = &sqlDialect
	} else {
		fmt.Print("Unable to find env SQL_DIALECT\n")
	}
	fmt.Printf("Using default SQL_DIALECT=%s\n", *cloumgmtconfig.Cfg.SQL_Dialect)

	sqlHost, ok := os.LookupEnv("SQL_HOST")
	if ok {
		cloumgmtconfig.Cfg.SQL_Host = &sqlHost
	} else {
		fmt.Print("Unable to find env SQL_HOST\n")
	}
	fmt.Printf("Using default SQL_HOST=%s\n", *cloumgmtconfig.Cfg.SQL_Host)

	sqlPort, ok := os.LookupEnv("SQL_PORT")
	if ok {
		iPort, err := strconv.Atoi(sqlPort)
		if err == nil {
			cloumgmtconfig.Cfg.SQL_Port = &iPort
		} else {
			fmt.Printf("Unable to convert %s into SQL_PORT\n", sqlPort)
		}
	} else {
		fmt.Print("Unable to find env SQL_PORT\n")
	}
	fmt.Printf("Using default SQL_HOST=%d\n", *cloumgmtconfig.Cfg.SQL_Port)

	sqlUser, ok := os.LookupEnv("SQL_USER")
	if ok {
		cloumgmtconfig.Cfg.SQL_User = &sqlUser
	} else {
		fmt.Print("Unable to find env SQL_USER\n")
	}
	fmt.Print("Using default SQL_USER=*****\n")

	sqlPassword, ok := os.LookupEnv("SQL_PASSWORD")
	if ok {
		cloumgmtconfig.Cfg.SQL_Password = &sqlPassword
	} else {
		fmt.Print("Unable to find env SQL_PASSWORD\n")
	}
	fmt.Print("Using default SQL_PASSWORD=*****\n")

	sqlDB, ok := os.LookupEnv("SQL_DB")
	if ok {
		cloumgmtconfig.Cfg.SQL_DB = &sqlDB
	} else {
		fmt.Print("Unable to find env SQL_DB\n")
	}
	fmt.Printf("Using default SQL_DB=%s\n", *cloumgmtconfig.Cfg.SQL_DB)
}

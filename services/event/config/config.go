package config

import (
	"cloudservices/common/base"
	"cloudservices/common/service"
	"flag"

	"github.com/davecgh/go-spew/spew"
	"github.com/golang/glog"
)

// ConfigData describes config data for cloudmgmt
type ConfigData struct {
	Port *int

	AWSRegion *string
	// ElasticSearch
	SearchURL   *string
	Environment *string
	// Code Coverage
	EnableCodeCoverage *bool
	// Whether to disable ES SSL when not using ES from AWS
	DisableESSSL *bool
}

var Cfg *ConfigData = &ConfigData{}

// LoadDefault populates the config values with the defaults
func init() {
	serviceConfig := service.GetServiceConfig(service.EventService)
	Cfg.Port = base.IntPtr(serviceConfig.Port)
	Cfg.AWSRegion = base.StringPtr("us-west-2")
	Cfg.Environment = base.StringPtr("sherlock_test")
	// ElasticSearch
	Cfg.SearchURL = base.StringPtr("https://search-sherlock-dev-d2i33pjtsix556oia2ydlhkln4.us-west-2.es.amazonaws.com")
	// Code Coverage
	Cfg.EnableCodeCoverage = base.BoolPtr(false)
	// Whether to disable ES SSL when not using ES from AWS
	Cfg.DisableESSSL = base.BoolPtr(false)
}

// LoadFlag populates the config values from command line
func (configData *ConfigData) LoadFlag() {

	Cfg.Port = flag.Int("port", *Cfg.Port, "<events server port>")
	Cfg.AWSRegion = flag.String("awsregion", *Cfg.AWSRegion, "<AWS region>")
	Cfg.Environment = flag.String("env", *Cfg.Environment, "<service environment>")
	Cfg.SearchURL = flag.String("search_url", *Cfg.SearchURL, "<Search URL>")
	// Code Coverage
	Cfg.EnableCodeCoverage = flag.Bool("enableCodeCoverage", false, "Set to true to enable coverage")
	// Whether to disable ES SSL when not using ES from AWS
	Cfg.DisableESSSL = flag.Bool("disable_es_ssl", false, "Set to true to disable ES SSL when not using ES from AWS")

	// parse must be done before sprintf below
	// otherwise DBURL would be incorrect
	flag.Parse()

	if glog.V(5) {
		spew.Dump(*Cfg)
	}
	glog.Infof("config init using server port: %d\n", *Cfg.Port)

}

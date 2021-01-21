package config

import (
	"cloudservices/common/service"
	"flag"
	"math/rand"
	"os"
)

const (
	logStreamerPort = 8081
)

type ConfigData struct {
	GRPCPort        *int
	SvcPort         *int
	RedisHost       *string
	RedisPort       *int
	EndpointsPrefix string
	Salt            string
	// Per tenant limits
	MaxSimultaneousAppsPipelines *int
	MaxSimultaneousEndpoints     *int
}

const (
	saltLength = 6 // We will append 6 chars to pub/sub urls
)

var Cfg *ConfigData = &ConfigData{}

func init() {
	serviceConfig := service.GetServiceConfig(service.DevToolsService)
	Cfg.SvcPort = flag.Int("svc_port", logStreamerPort, "<dev tools service port>")
	Cfg.GRPCPort = flag.Int("grpc_port", serviceConfig.Port, "<dev tools grpc service port>")
	//Cfg.RedisHost = flag.String("redis_host", "devtools-redis-svc", "<Redis host for devtools>")
	Cfg.RedisHost = flag.String("redis_host", "devtools-redis-svc.test.svc.cluster.local", "<Redis host for devtools>")
	Cfg.RedisPort = flag.Int("redis_port", 6379, "<redis port>")
	Cfg.MaxSimultaneousAppsPipelines = flag.Int("max_simul_apps_pipelines", 10, "<max simultaneous apps/pipelines per tenant>")
	Cfg.MaxSimultaneousEndpoints = flag.Int("max_simul_endpoints", 100, "<max simultaenous active endpoints per tenant>")
	if val := os.Getenv("ENDPOINTS_PREFIX"); val == "" {
		Cfg.EndpointsPrefix = "https://devtools-test.ntnxsherlock.com"
	} else {
		Cfg.EndpointsPrefix = val
	}

	Cfg.Salt = randomString(saltLength)
}

// randomString - Generate a random string of a-z chars with len = len
func randomString(len int) string {
	bytes := make([]byte, len)
	for i := 0; i < len; i++ {
		bytes[i] = byte(97 + rand.Intn(25)) // a=97 and z = 97+25
	}
	return string(bytes)
}

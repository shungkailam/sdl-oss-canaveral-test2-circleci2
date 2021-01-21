package api

import (
	"cloudservices/tenantpool/config"
	"cloudservices/tenantpool/core"
	gapi "cloudservices/tenantpool/generated/grpc"
	"cloudservices/tenantpool/model"
	"fmt"

	"github.com/go-redis/redis"

	"google.golang.org/grpc"
)

// APIServer is the interface like other microservices - account or event
type APIServer interface {
	Register(gServer *grpc.Server)
	Close() error
	gapi.TenantPoolServiceServer
}

// apiServer is the internal implementation of APIServer
type apiServer struct {
	tenantPoolManager *core.TenantPoolManager
}

// NewAPIServer creates an instance of apiServer
func NewAPIServer() APIServer {
	var redisClient *redis.Client
	if !*config.Cfg.DisableScaleOut {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:6379", *config.Cfg.RedisHost),
			Password: "", // no password set
			DB:       0,  // use default DB
		})
	}
	edgeProvisioner, err := core.NewBottEdgeProvisioner()
	if err != nil {
		panic(err)
	}
	tenantPoolManager, err := core.NewTenantPoolManagerWithRedisClient(edgeProvisioner, redisClient)
	if err != nil {
		panic(err)
	}
	return &apiServer{tenantPoolManager: tenantPoolManager}
}

// NewAPIServerEx creates an instance of apiServer.
// This is used with mock or real implementation of EdgeProvisioner
func NewAPIServerEx(edgeProvisioner model.EdgeProvisioner) APIServer {
	if edgeProvisioner == nil {
		panic("Edge provisioner is not set")
	}
	tenantPoolManager, err := core.NewTenantPoolManager(edgeProvisioner)
	if err != nil {
		panic(err)
	}
	return &apiServer{tenantPoolManager: tenantPoolManager}
}

func (server *apiServer) Register(gServer *grpc.Server) {
	gapi.RegisterTenantPoolServiceServer(gServer, server)
}

func (server *apiServer) Close() error {
	return server.tenantPoolManager.Close()
}

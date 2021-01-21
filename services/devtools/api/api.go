package api

import (
	"cloudservices/devtools/config"
	"cloudservices/devtools/devtoolsservice"
	gapi "cloudservices/devtools/generated/grpc"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"google.golang.org/grpc"
)

// APIServer is the interface like other microservices
type APIServer interface {
	Register(gServer *grpc.Server)
	RegisterAuthHandler(router *httprouter.Router)
	Close() error
	gapi.DevToolsServiceServer
}

// apiServer is the internal implementation of APIServer
type apiServer struct {
	cfg          *config.ConfigData
	redisManager *devtoolsservice.RedisManager
}

// NewAPIServer create a new instance of APIServer
func NewAPIServer(redisManager *devtoolsservice.RedisManager) (APIServer, error) {
	return &apiServer{
		cfg:          config.Cfg,
		redisManager: redisManager,
	}, nil
}

// Register registers 'gServer' with the rpc server.
func (srv *apiServer) Register(gServer *grpc.Server) {
	gapi.RegisterDevToolsServiceServer(gServer, srv)
}

// RegisterAuthHandler registers an in-line auth handler. This handler
// will be invoked to authorize incoming connections.
func (srv *apiServer) RegisterAuthHandler(router *httprouter.Router) {
	router.GET("/auth", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		w.WriteHeader(http.StatusOK)
		return
	})
}

// Close will cleanup the APIServer before returning.
func (srv *apiServer) Close() error {
	//return srv.redisClient.Close()
	return nil
}

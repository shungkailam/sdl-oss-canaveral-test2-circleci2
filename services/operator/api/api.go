package api

import (
	cauth "cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"

	"cloudservices/operator/auth"
	"cloudservices/operator/config"
	"cloudservices/operator/softwareupdate"
	"net/http"

	gapi "cloudservices/operator/generated/grpc"
	"cloudservices/operator/generated/operator/models"
	"cloudservices/operator/generated/operator/restapi"
	"cloudservices/operator/generated/operator/restapi/operations"
	"cloudservices/operator/generated/operator/restapi/operations/edge"
	"cloudservices/operator/generated/operator/restapi/operations/operator"
	"cloudservices/operator/generated/operator/restapi/operations/test"
	"context"
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-openapi/loads"
	"github.com/go-openapi/runtime/middleware"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const (
	HTTPRequestContextKey = ContextKey("HTTPRequestContext")
)

// ContextKey Request context key
type ContextKey string

type HTTPRequestContext struct {
	URI    string
	Method string
	Params string
}

type APIServer struct {
	RPCServer  gapi.ReleaseServiceServer
	Api        *operations.OperatorAPI
	srv        *restapi.Server
	t          time.Time
	cfg        *config.ConfigData
	keyService crypto.KeyService
}

type rpcServer struct {
	updateHandler *softwareupdate.Handler
}

// NewAPIServer starts the API service
func NewAPIServer() *APIServer {
	swaggerSpec, err := loads.Analyzed(restapi.SwaggerJSON, "")
	if err != nil {
		glog.Fatalln(err)
	}
	api := operations.NewOperatorAPI(swaggerSpec)
	keyService := crypto.NewKeyService(*config.Cfg.AWSRegion, *config.Cfg.JWTSecret, *config.Cfg.AWSKMSKey, *config.Cfg.UseKMS)
	// rpcServer is forced to implement the interfaces
	rpcServer := &rpcServer{
		updateHandler: softwareupdate.NewHandler(),
	}
	server := restapi.NewServer(api)
	server.Host = "0.0.0.0"
	server.Port = *config.Cfg.Port
	srv := &APIServer{
		Api:        api,
		srv:        server,
		t:          time.Now(),
		cfg:        config.Cfg,
		keyService: keyService,
		RPCServer:  rpcServer,
	}

	return srv
}

// StartServer starts a server used to serve the swagger spec
func (server *APIServer) StartServer() (err error) {
	// serve API
	glog.Infof("Starting sherlock operator REST api server.")
	if server == nil {
		glog.Errorf("operator is not initialized.")
		return errors.New("operator initialize is not called")
	}

	if err := server.srv.Serve(); err != nil {
		glog.Error(err)
		return err
	}
	return err
}

func (server *APIServer) Register(gServer *grpc.Server) {
	gapi.RegisterReleaseServiceServer(gServer, server.RPCServer)
}

var awsSession *session.Session

func init() {}

func GetHTTPRequestContext(context context.Context) HTTPRequestContext {
	httpContext, ok := context.Value(HTTPRequestContextKey).(HTTPRequestContext)
	if !ok {
		return HTTPRequestContext{}
	}
	return httpContext
}

func checkClaim(i interface{}) middleware.Responder {
	authContext, ok := i.(*base.AuthContext)
	if !ok {
		errStr := "Invalid credentials"
		retErr := &models.Error{Message: &errStr}
		return edge.NewUploadReleaseDefault(http.StatusUnauthorized).WithPayload(retErr)
	}

	if !cauth.IsOperatorRole(authContext) {
		errStr := "Incorrect credentials, does not have valid role"
		retErr := &models.Error{Message: &errStr}
		return edge.NewUploadReleaseDefault(http.StatusUnauthorized).WithPayload(retErr)
	}
	return nil
}

// PingHandler returns pong, used for liveness probe
func (server *APIServer) PingHandler(params test.PingParams) middleware.Responder {
	return test.NewPingOK().WithPayload("pong")
}

// RegisterAPIHandlers registers the api updateHandlers
func (server *APIServer) RegisterAPIHandlers() {

	server.Api.IsRegisteredAuth = auth.LoginHandler
	loginHandler := func(params operator.LoginParams) middleware.Responder {
		return server.LoginHandler(params)
	}
	server.Api.OperatorLoginHandler = operator.LoginHandlerFunc(loginHandler)

	pingHandler := func(params test.PingParams) middleware.Responder {
		return server.PingHandler(params)
	}
	server.Api.TestPingHandler = test.PingHandlerFunc(pingHandler)

	uploadReleaseHandler := func(params edge.UploadReleaseParams, i interface{}) middleware.Responder {
		err := checkClaim(i)
		if err != nil {
			return err
		}
		return server.UploadReleaseHandler(params)
	}
	server.Api.EdgeUploadReleaseHandler = edge.UploadReleaseHandlerFunc(uploadReleaseHandler)

	updateReleaseHandler := func(params edge.UpdateReleaseParams, i interface{}) middleware.Responder {
		err := checkClaim(i)
		if err != nil {
			return err
		}
		return server.UpdateReleaseHandler(params)
	}
	server.Api.EdgeUpdateReleaseHandler = edge.UpdateReleaseHandlerFunc(updateReleaseHandler)

	getReleaseHandler := func(params edge.GetReleaseParams, i interface{}) middleware.Responder {
		err := checkClaim(i)
		if err != nil {
			return err
		}
		return server.GetReleaseHandler(params)
	}
	server.Api.EdgeGetReleaseHandler = edge.GetReleaseHandlerFunc(getReleaseHandler)

	deleteReleaseHandler := func(params edge.DeleteReleaseParams, i interface{}) middleware.Responder {
		err := checkClaim(i)
		if err != nil {
			return err
		}
		return server.DeleteReleaseHandler(params)
	}
	server.Api.EdgeDeleteReleaseHandler = edge.DeleteReleaseHandlerFunc(deleteReleaseHandler)
	listCompatibleReleasesHandler := func(params edge.ListCompatibleReleasesParams, i interface{}) middleware.Responder {
		err := checkClaim(i)
		if err != nil {
			return err
		}
		return server.ListCompatibleReleasesHandler(params)
	}
	server.Api.EdgeListCompatibleReleasesHandler = edge.ListCompatibleReleasesHandlerFunc(listCompatibleReleasesHandler)

	listReleasesHandler := func(params edge.ListReleasesParams, i interface{}) middleware.Responder {
		err := checkClaim(i)
		if err != nil {
			return err
		}
		return server.ListReleasesHandler(params)
	}
	server.Api.EdgeListReleasesHandler = edge.ListReleasesHandlerFunc(listReleasesHandler)

}

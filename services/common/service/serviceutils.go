package service

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cockroachdb/cmux"

	"github.com/golang/glog"

	"google.golang.org/grpc"
)

const (
	ClientTimeout         = time.Second * 60
	defaultServicePort    = 8080
	defaultPrometheusPort = 9436
)

type ServiceName int

const (
	// Add service names here
	AccountService    ServiceName = iota
	EventService      ServiceName = iota
	OperatorService   ServiceName = iota
	TenantPoolService ServiceName = iota
	DevToolsService   ServiceName = iota
	AIService         ServiceName = iota
	AuditLogService   ServiceName = iota
)

var (
	// serviceConfigs keeps the connection info for all the services
	serviceConfigs map[ServiceName]*ServiceConfig
	clientOpts     []grpc.DialOption
	serverOpts     []grpc.ServerOption
	environment    string
)

func init() {
	// Client options
	clientOpts = append(clientOpts, grpc.WithInsecure())
	clientOpts = append(clientOpts, grpc.WithUnaryInterceptor(UnaryClientInterceptor()))
	clientOpts = append(clientOpts, grpc.WithStreamInterceptor(StreamClientInterceptor()))
	// Allow sending 20MB of data using grpc
	clientOpts = append(clientOpts, grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*20)))

	// InitialConnWindowSize returns a ServerOption that sets window size for a connection
	clientOpts = append(clientOpts, grpc.WithInitialConnWindowSize(1042*512))
	// InitialWindowSize returns a ServerOption that sets window size for stream
	clientOpts = append(clientOpts, grpc.WithInitialWindowSize(1042*512))
	// WriteBufferSize determines how much data can be batched before doing a write on the wire
	// 1 MB
	clientOpts = append(clientOpts, grpc.WithWriteBufferSize(1024*1024))
	// WithReadBufferSize lets you set the size of read buffer, this determines how much data can be read at most for each read syscall.
	clientOpts = append(clientOpts, grpc.WithReadBufferSize(1024*1024))

	// Server options
	serverOpts = append(serverOpts, grpc.UnaryInterceptor(UnaryServerInterceptor()))
	serverOpts = append(serverOpts, grpc.StreamInterceptor(StreamServerInterceptor()))
	// InitialConnWindowSize returns a ServerOption that sets window size for a connection
	serverOpts = append(serverOpts, grpc.InitialConnWindowSize(1042*512))
	// InitialWindowSize returns a ServerOption that sets window size for stream
	serverOpts = append(serverOpts, grpc.InitialWindowSize(1042*512))
	// Allow sending 20MB of data using grpc
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(1024*1024*20))
	//grpc.WithDefaultCallOptions()

	// WriteBufferSize determines how much data can be batched before doing a write on the wire
	// 1 MB
	serverOpts = append(serverOpts, grpc.WriteBufferSize(1024*1024))
	// ReadBufferSize lets you set the size of read buffer, this determines how much data can be read at most for one read syscall
	serverOpts = append(serverOpts, grpc.ReadBufferSize(1024*1024))

	// Register services here
	serviceConfigs = map[ServiceName]*ServiceConfig{
		AccountService:    &ServiceConfig{Host: "accountserver-svc", Port: defaultServicePort},
		EventService:      &ServiceConfig{Host: "eventserver-svc", Port: defaultServicePort},
		OperatorService:   &ServiceConfig{Host: "operator-svc", Port: 9001},
		TenantPoolService: &ServiceConfig{Host: "tenantpoolserver-svc", Port: defaultServicePort},
		DevToolsService:   &ServiceConfig{Host: "devtools-svc", Port: defaultServicePort},
		AIService:         &ServiceConfig{Host: "ai-svc", Port: 80},
		AuditLogService:   &ServiceConfig{Host: "auditlog-svc", Port: defaultServicePort},
	}

	// Override for localhost run
	if os.Getenv("SHERLOCK_ENV") == "local" {
		for key, value := range serviceConfigs {
			value.Host = "localhost"
			value.Port = value.Port + int(key) + 1000
		}
	}
}

// ServiceConfig stores the service routing information
type ServiceConfig struct {
	Host string
	Port int
}

// Override from test
func OverrideServiceConfig(service ServiceName, serviceConfig *ServiceConfig) {
	serviceConfigs[service] = serviceConfig
}

func (serviceConfig *ServiceConfig) GetEndpoint() string {
	return fmt.Sprintf("%s:%d", serviceConfig.Host, serviceConfig.Port)
}

// GetConnectionInfo returns the connection info if exists, otherwise default is returned
func GetServiceConfig(service ServiceName) *ServiceConfig {
	serviceConfig, ok := serviceConfigs[service]
	if ok {
		return serviceConfig
	}
	// Default
	return &ServiceConfig{Host: "localhost", Port: defaultServicePort}
}

func StartServer(port int, gHandle func(*grpc.Server, net.Listener, *httprouter.Router) error) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}
	defer listener.Close()
	mux := cmux.New(listener)
	// HTTP1 GET first
	// TODO need to revisit later
	hListener := mux.Match(cmux.HTTP1Fast("GET"))
	gListener := mux.Match(cmux.Any())
	sMux := http.NewServeMux()
	router := httprouter.New()
	// Add ping handler
	AddPingHandler(router)
	// Add log level handler
	AddLogLevelHandler(router)
	sMux.HandleFunc("/", router.ServeHTTP)
	hServer := &http.Server{
		Handler: sMux,
	}
	gServer := grpc.NewServer(serverOpts...)
	errChan := make(chan error)
	go func() {
		glog.Infof("Starting gRPC handler")
		errChan <- gHandle(gServer, gListener, router)
	}()
	go func() {
		glog.Infof("Starting health ping handler")
		errChan <- hServer.Serve(hListener)
	}()
	// setup server for prometheus metrics
	go func() {
		prometheusPort := defaultPrometheusPort - defaultServicePort + port
		mux := http.NewServeMux()
		mux.Handle("/metrics/", promhttp.Handler())
		s := &http.Server{
			Addr:    fmt.Sprintf(":%d", prometheusPort),
			Handler: mux,
		}
		errChan <- s.ListenAndServe()
	}()
	err = mux.Serve()
	if err != nil {
		glog.Errorf("Error occurred in mux listener. Error: %s", err.Error())
		return err
	}
	for {
		select {
		case err := <-errChan:
			glog.Errorf("Error occurred in server. Error: %s", err.Error())
			return err
		}
	}
	return err
}

func CallClient(ctx context.Context, service ServiceName, handler func(context.Context, *grpc.ClientConn) error) error {
	return CallClientWithTimeout(ctx, service, handler, ClientTimeout)
}

func CallClientWithTimeout(ctx context.Context, service ServiceName, handler func(context.Context, *grpc.ClientConn) error, timeout time.Duration) error {
	serviceConfig := GetServiceConfig(service)
	serviceEndpoint := serviceConfig.GetEndpoint()
	return CallClientEndpointWithTimeout(ctx, serviceEndpoint, handler, timeout)
}

func CallClientEndpoint(ctx context.Context, serviceEndpoint string, handler func(context.Context, *grpc.ClientConn) error) error {
	return CallClientEndpointWithTimeout(ctx, serviceEndpoint, handler, ClientTimeout)
}

func CallClientEndpointWithTimeout(ctx context.Context, serviceEndpoint string, handler func(context.Context, *grpc.ClientConn) error, timeout time.Duration) error {
	conn, err := grpc.Dial(serviceEndpoint, clientOpts...)
	if err != nil {
		err = errcode.NewInternalError(fmt.Sprintf("Unable to connect to service endpoint: %s", serviceEndpoint))
		glog.Errorf(base.PrefixRequestID(ctx, "Error: %s"), err.Error())
		return err
	}
	defer conn.Close()
	err = base.Call(ctx, func(ctx context.Context) error {
		return handler(ctx, conn)
	}, timeout)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in callback. Error: %s"), err.Error())
		return ErrorCodeFromError(err)
	}
	return nil
}

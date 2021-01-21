package main

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/cfssl"
	"cloudservices/cloudmgmt/config"
	"cloudservices/cloudmgmt/event"
	gapi "cloudservices/cloudmgmt/generated/grpc"
	cmGrpc "cloudservices/cloudmgmt/grpc"
	cmRouter "cloudservices/cloudmgmt/router"
	"cloudservices/cloudmgmt/ssh"
	"cloudservices/cloudmgmt/websocket"
	"fmt"
	"net"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/go-redis/redis"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"google.golang.org/grpc"
)

func init() {
	config.Cfg.LoadFlag()
	api.InitGlobals()
	cfssl.InitCfssl(*config.Cfg.CfsslProtocol, *config.Cfg.CfsslHost, *config.Cfg.CfsslPort)

}

func main() {

	defer glog.Flush()

	glog.Infof("main { using server port: %d\n", *config.Cfg.Port)

	errChan := make(chan error)

	var redisClient *redis.Client
	if !*config.Cfg.DisableScaleOut {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     fmt.Sprintf("%s:6379", *config.Cfg.RedisHost),
			Password: "", // no password set
			DB:       0,  // use default DB
		})
	}

	dbAPI, err := api.NewObjectModelAPIWithCache(redisClient, true)
	if err != nil {
		// sqlx.Connect failed
		panic(err)
	}
	defer dbAPI.Close()

	err = cmRouter.RegisterOAuthHandlers(dbAPI, config.Cfg.RedirectURLs.Values())
	if err != nil {
		panic(err)
	}

	event.RegisterEventListeners(dbAPI)

	router := httprouter.New()

	msgSvc := websocket.ConfigureWSMessagingService(dbAPI, router, redisClient)

	ssh.ConfigureWSSSHService(dbAPI, router, redisClient)

	cmRouter.ConfigureRouter(dbAPI, router, redisClient, msgSvc, *config.Cfg.ContentDir)

	if !*config.Cfg.DisableScaleOut {
		// for now, only start gRPC server when scale out is on
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *config.Cfg.GRPCPort))
		if err != nil {
			// fail to listen
			panic(err)
		}
		s := grpc.NewServer()
		gapi.RegisterCloudmgmtServiceServer(s, cmGrpc.NewGrpcServer(dbAPI, msgSvc))
		go func() {
			errChan <- s.Serve(lis)
		}()
	}

	// setup server for prometheus metrics
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics/", promhttp.Handler())
		s := &http.Server{
			Addr:    fmt.Sprintf(":%d", *config.Cfg.PrometheusPort),
			Handler: mux,
		}
		errChan <- s.ListenAndServe()
	}()

	go func() {
		errChan <- http.ListenAndServe(fmt.Sprintf(":%d", *config.Cfg.Port), &Server{router})
	}()

	for {
		select {
		case err := <-errChan:
			panic(err)
		}
	}
}

type Server struct {
	router *httprouter.Router
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// add HSTS header, see, e.g., https://hstspreload.org/
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
	s.router.ServeHTTP(w, r)
}

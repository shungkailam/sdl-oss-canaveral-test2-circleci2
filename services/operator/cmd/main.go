package main

import (
	"cloudservices/common/service"
	"cloudservices/operator/api"
	"cloudservices/operator/config"

	"net"

	//cmRouter "cloudoperator/router"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"

	"github.com/golang/glog"
)

func init() {
	config.Cfg.LoadFlag()
}

func main() {

	defer glog.Flush()

	server := api.NewAPIServer()
	server.RegisterAPIHandlers()
	// Start the http server
	go server.StartServer()

	panic(service.StartServer(*config.Cfg.GrpcPort, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
		server.Register(gServer)
		return gServer.Serve(listener)
	}))
}

package main

import (
	"cloudservices/common/service"
	"cloudservices/tenantpool/api"
	"cloudservices/tenantpool/config"
	"net"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

func init() {
	config.Cfg.LoadFlag()
}

// Entry for service
func main() {
	defer glog.Flush()
	glog.Infof("main { using server port: %d }\n", *config.Cfg.Port)
	server := api.NewAPIServer()
	defer server.Close()

	panic(service.StartServer(*config.Cfg.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
		server.Register(gServer)
		return gServer.Serve(listener)
	}))
}

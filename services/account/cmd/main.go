package main

import (
	"cloudservices/account/api"
	"cloudservices/account/config"
	"cloudservices/cloudmgmt/cfssl"
	"cloudservices/common/service"
	"net"

	_ "github.com/go-sql-driver/mysql"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

func init() {
	config.Cfg.LoadFlag()
	cfssl.InitCfssl(*config.Cfg.CfsslProtocol, *config.Cfg.CfsslHost, *config.Cfg.CfsslPort)

}

func main() {
	defer glog.Flush()
	glog.Infof("main { using server port: %d }\n", *config.Cfg.Port)
	server, err := api.NewAPIServer()
	if err != nil {
		panic(err)
	}
	defer server.Close()

	panic(service.StartServer(*config.Cfg.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
		server.Register(gServer)
		return gServer.Serve(listener)
	}))
}

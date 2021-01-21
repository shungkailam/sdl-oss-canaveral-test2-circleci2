package main

import (
	"cloudservices/common/service"
	"cloudservices/event/api"
	"cloudservices/event/config"
	"net"

	"github.com/julienschmidt/httprouter"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

func init() {
	config.Cfg.LoadFlag()
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

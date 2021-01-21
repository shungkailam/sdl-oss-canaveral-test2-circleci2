package api_test

import (
	"cloudservices/common/base"
	"cloudservices/common/service"
	"cloudservices/tenantpool/api"
	"cloudservices/tenantpool/config"
	"context"
	"net"
	"os"
	"os/signal"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/grpc"
)

func TestMain(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Override the defaults
	config.Cfg.TenantPoolScanDelay = base.DurationPtr(time.Second * 30)
	t.Run("Running main tests", func(t *testing.T) {
		defer glog.Flush()
		glog.Infof("main { using server port: %d }\n", *config.Cfg.Port)
		server := api.NewAPIServer()
		defer server.Close()
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			<-c
			os.Exit(0)
		}()
		// Start gRPC server
		panic(service.StartServer(*config.Cfg.Port, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
			server.Register(gServer)
			return gServer.Serve(listener)
		}))
	})
}

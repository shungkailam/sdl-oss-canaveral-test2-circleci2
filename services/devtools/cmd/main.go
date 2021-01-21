package main

import (
	"cloudservices/common/service"
	"cloudservices/devtools/api"
	"cloudservices/devtools/config"
	"flag"
	"fmt"
	"net"
	"strings"
	"time"

	"cloudservices/devtools/devtoolsservice"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"google.golang.org/grpc"

	"github.com/go-redis/redis"
)

func main() {
	flag.Parse()
	glog.Infof("devtools config %v %v %v %v\n", config.Cfg.EndpointsPrefix,
		*config.Cfg.RedisHost, *config.Cfg.RedisPort,
		*config.Cfg.SvcPort)

	defer glog.Flush()

	redisClient := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs:      []string{fmt.Sprintf("%s:6379", *config.Cfg.RedisHost)},
		MaxConnAge: time.Minute,
		Password:   "", // no password set
	})
	// Loop till cluster is ready.
	res, err := redisClient.ClusterInfo().Result()
	if err != nil || !strings.Contains(res, "cluster_state:ok") {
		glog.Errorf("Redis cluster is not ready. res: %s, err: %s", res, err)
		// NOTE: Call to NewClusterClient tries to reach the cluster and seems to cache
		//       the connection to the individual nodes. The only way to get the client
		//       to reconnect is to create a new client.
		glog.Fatal()
	}
	redisManager := devtoolsservice.NewRedisManager(redisClient)

	srv, err := api.NewAPIServer(redisManager)
	if err != nil {
		panic(err)
	}
	defer srv.Close()

	go func() { // Ping redis server
		consecutivePingsFailed := 0
		ticker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-ticker.C:
				pong, err := redisClient.Ping().Result()
				if err != nil {
					glog.V(4).Infof("Error in pinging devtools-redis: %v, consecutive err count: %d", err, consecutivePingsFailed)
					consecutivePingsFailed++
					if consecutivePingsFailed > 3 {
						glog.Errorf("4 consecutive pings failed to redis server")
						panic(err) // Most likely we have stale redis IPs. Restart the pod so that we will get new IPs.
					}
				} else {
					glog.V(4).Infof("Ping response received: %v", pong)
					consecutivePingsFailed = 0 // reset error count!
				}
			}
		}
	}()

	go func() {
		glog.Info("Starting devtools server")
		devtoolsSvc := devtoolsservice.NewDevtoolsServer(redisManager)
		err := devtoolsSvc.Start(*config.Cfg.SvcPort)
		glog.Errorf("devtools svc stopped: %s", err)
	}()

	panic(service.StartServer(*config.Cfg.GRPCPort, func(gServer *grpc.Server, listener net.Listener, router *httprouter.Router) error {
		srv.Register(gServer)
		srv.RegisterAuthHandler(router)
		return gServer.Serve(listener)
	}))
}

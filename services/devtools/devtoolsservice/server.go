package devtoolsservice

import (
	"bytes"
	"cloudservices/devtools/generated/swagger/models"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"
)

// DevtoolsServer serves all devtools services
type DevtoolsServer struct {
	logstreamer *LogStreamer
	handler     *http.Handler
}

func (server *DevtoolsServer) init() {
	router := httprouter.New()

	getStreamHandler := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		resp, apiErr := server.logstreamer.GetStreamLogs(params.ByName("endpoint"), params.ByName("latestts"))
		w.Header().Set("Content-Type", "application/json")
		if apiErr != nil {
			// Handle errors
			//w.WriteHeader(http.StatusPreconditionFailed)
			w.WriteHeader(int(*apiErr.StatusCode))
			if err := json.NewEncoder(w).Encode(*apiErr); err != nil {
				glog.Errorf("Error in sending response to client. getStreamHandler: %s", err)
			}
			return
		}

		if err := json.NewEncoder(w).Encode(resp); err != nil {
			glog.Errorf("Error in sending response to client. getStreamHandler: %s", err)
		}
	}
	router.GET("/v1.0/logs/fetch/:endpoint/:latestts", getStreamHandler)

	putStreamLogsHandler := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		w.Header().Set("Content-Type", "application/json")
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		logs := &models.LogsContent{}
		if err := json.Unmarshal(buf.Bytes(), logs); err != nil {
			response := models.APIError{
				StatusCode: &status500,
				Message:    &status500Message,
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				glog.Errorf("Error in sending response to client. putStreamLogsHandler: %s", err)
			}
			return
		}

		if apiErr := server.logstreamer.PutStreamLogs(params.ByName("endpoint"), *logs.Contents); apiErr != nil {
			// Handle errors
			w.WriteHeader(int(*apiErr.StatusCode))
			if err := json.NewEncoder(w).Encode(apiErr); err != nil {
				glog.Errorf("Error in sending response to client. putStreamLogsHandler: %s", err)
			}
			return
		}
		if err := json.NewEncoder(w).Encode(models.ResponseBase{StatusCode: &status200}); err != nil {
			glog.Errorf("Error in sending response to client. putStreamLogsHandler: %s", err)
		}
	}
	router.PUT("/v1.0/logs/push/:endpoint", putStreamLogsHandler)

	postHeartbeatHandler := func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
		apiErr := server.logstreamer.PublisherHeartbeat(params.ByName("endpoint"))
		w.Header().Set("Content-Type", "application/json")
		if apiErr != nil {
			// Handle errors
			w.WriteHeader(int(*apiErr.StatusCode))
			if err := json.NewEncoder(w).Encode(apiErr); err != nil {
				glog.Errorf("Error in sending response to client. postHeartbeatHandler: %s", err)
			}
			return
		}

		if err := json.NewEncoder(w).Encode(models.ResponseBase{StatusCode: &status200}); err != nil {
			glog.Errorf("Error in sending response to client. postHeartbeatHandler: %s", err)
		}
	}
	router.POST("/v1.0/logs/heartbeat/:endpoint", postHeartbeatHandler)

	optionsMethodHandler := func(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
		if req.Method == "OPTIONS" {
			return
		}
	}
	router.OPTIONS("/v1.0/logs/fetch/:endpoint/:latestts", optionsMethodHandler)
	router.OPTIONS("/v1.0/logs/push/:endpoint", optionsMethodHandler)
	router.OPTIONS("/v1.0/logs/heartbeat/:endpoint", optionsMethodHandler)

	corsObj := cors.New(cors.Options{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost, http.MethodHead, http.MethodOptions, http.MethodPut},
		AllowOriginFunc: func(origin string) bool { return true },
		AllowedHeaders:  []string{"Accept", "Authorization", "Cache-Control", "Content-Type", "Keep-Alive", "Origin", "User-Agent", "X-Requested-With"},
	})
	handler := corsObj.Handler(router)
	server.handler = &handler
}

// NewDevtoolsServer server instantiated and handlers are defined
func NewDevtoolsServer(redisManager *RedisManager) *DevtoolsServer {
	logstreamer := NewLogStreamer(redisManager)
	srv := &DevtoolsServer{
		logstreamer: logstreamer,
	}
	srv.init()
	return srv
}

// Start Starts devtool service
func (server *DevtoolsServer) Start(port int) error {
	return http.ListenAndServe(fmt.Sprintf(":%d", port), *server.handler)
}

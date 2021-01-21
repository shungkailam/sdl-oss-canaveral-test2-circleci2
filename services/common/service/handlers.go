package service

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

// AddPingHandler adds ping handler
func AddPingHandler(router *httprouter.Router) {
	router.GET("/v1/ping", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		fmt.Fprintf(w, "pong")
	})
}

// AddLogLevelHandler adds log level handler
func AddLogLevelHandler(router *httprouter.Router) {
	// Setter for log level
	router.GET("/v1/log/level/:level", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		level := ps.ByName("level")
		err := flag.Set("v", level)
		if err != nil {
			glog.Warningf("Failed to set log level to %s. Error: %s", level, err.Error())
			fmt.Fprintf(w, "Error setting log level to %s", level)
			return
		}
		glog.Infof("Successully set log level to %s", level)
		fmt.Fprintf(w, "Successully set log level to %s", level)
	})

	// Getter for log level
	router.GET("/v1/log/level", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		fl := flag.Lookup("v")
		if fl == nil {
			flag.Set("v", "0")
			fl = flag.Lookup("v")
		}
		level := fl.Value
		fmt.Fprintf(w, "%s", level)
	})
}

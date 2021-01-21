package spa

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/golang/glog"
)

// serve static files + SPA support
// To only serve static files, one could set this to http.FileServer(http.Dir(contentDir))
// However, to support SPA, we want all 404's to be served /index.html so UI can handle the app routing

type SPAHandler struct {
	ContentDir     string
	httpFileServer http.Handler
}

func addNoCacheHeader(w http.ResponseWriter) {
	w.Header().Set("cache-control", "max-age=0")
}

func (spa *SPAHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if spa.httpFileServer == nil {
		spa.httpFileServer = http.FileServer(http.Dir(spa.ContentDir))
	}
	// replace all the multiple forward slashes with one slash
	upath := path.Clean(r.URL.Path)
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	fullpath := spa.ContentDir + upath
	if _, err := os.Stat(fullpath); err == nil {
		// path exists, serve the file
		w.Header().Set("x-frame-options", "SAMEORIGIN")
		w.Header().Set("x-xss-protection", "1; mode=block")
		if upath == "/" || upath == "/index.html" {
			addNoCacheHeader(w)
		}
		spa.httpFileServer.ServeHTTP(w, r)
	} else if strings.HasPrefix(upath, "/v1/") || strings.HasPrefix(upath, "/v1.0/") {
		// non existing REST endpoint (prefix /v1/, /v1.0/) is hit
		glog.Errorf("Unknown REST endpoint: %s\n", upath)
		sendInvalidPathError(w)
	} else {
		// path does not exist, read full index.html file
		glog.Infof("file does not exist: %s\n", fullpath)
		data, err := ioutil.ReadFile(spa.ContentDir + "/index.html")
		if err != nil {
			glog.Errorf("failed to read file: %s, error: %s\n", spa.ContentDir+"/index.html", err.Error())
			sendGetIndexFailedError(w)
			return
		}
		w.Header().Set("x-frame-options", "SAMEORIGIN")
		w.Header().Set("x-xss-protection", "1; mode=block")
		addNoCacheHeader(w)
		_, err = w.Write(data)
		if err != nil {
			glog.Errorf("failed to write file: %s, error: %s\n", spa.ContentDir+"/index.html", err.Error())
			sendGetIndexFailedError(w)
			return
		}
	}
}

func sendGetIndexFailedError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{"statusCode": 500, "message": "get index.html failed"}`)
}

func sendInvalidPathError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, `{"statusCode": 404, "message": "Invalid path"}`)
}

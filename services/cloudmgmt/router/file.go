package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	funk "github.com/thoas/go-funk"
	"github.com/xi2/httpgzip"
)

// FileServer serves file stored in S3 or any other store (future)
type FileServer struct {
	DBAPI api.ObjectModelAPI
}

var (
	// ErrUnhandled is returned when the HTTP path is not handled
	ErrUnhandled = errors.New("Unhandled")
)

// ServeFileRequest is a lightweight handler for the HTTP file requests.
// If the path is not handled by this method, it returns false.
// Path must of the form /v1.0/files/private/<tenantID>/...
func (fs *FileServer) ServeFileRequest(w http.ResponseWriter, r *http.Request) error {
	var err error
	reqID := r.Header.Get("X-Request-ID")
	if len(reqID) == 0 {
		reqID = base.GetUUID()
	}
	ctx := context.WithValue(r.Context(), base.RequestIDKey, reqID)
	defer func() {
		w.Header().Set("X-Request-ID", reqID)
		if err != ErrUnhandled {
			handleResponse(w, r, err, "%s files %s", r.Method, r.URL.Path)
		}
	}()
	// replace all the multiple forward slashes with one slash
	upath := path.Clean(r.URL.Path)
	if !strings.HasPrefix(upath, "/") {
		upath = "/" + upath
	}
	// path prefixes handled by this server
	prefix, found := funk.FindString([]string{"/v1/files/", "/v1.0/files/"}, func(prefix string) bool {
		return strings.HasPrefix(upath, prefix)
	})
	if !found {
		return ErrUnhandled
	}
	fmt.Println("upath", upath)
	// private/<tenantID>/... or public/<tenantID>/...
	upath = strings.TrimLeft(upath, prefix)
	tokens := strings.SplitN(upath, "/", 3)
	if len(tokens) < 2 {
		err = errcode.NewPermissionDeniedError(r.URL.Path)
		return err
	}
	baseFolder, found := funk.FindString([]string{"public", "private"}, func(baseFolder string) bool {
		return baseFolder == tokens[0]
	})
	if !found {
		err = errcode.NewPermissionDeniedError(r.URL.Path)
		return err
	}
	requiresAuth := false
	// Auth is required for non GET or files uploaded to private folder
	if r.Method != "GET" || baseFolder == "private" {
		requiresAuth = true
	}
	var tenantID string
	if requiresAuth {
		ok := false
		claims := jwt.MapClaims{}
		claims, err = auth.VerifyAuthorization(r, fs.DBAPI.GetPublicKeyResolver, fs.DBAPI.GetClaimsVerifier)
		if err == nil {
			tenantID, ok = claims["tenantId"].(string)
			if ok && (len(tenantID) == 0 || tokens[1] != tenantID) {
				ok = false
			}
		}
		if !ok {
			err = errcode.NewPermissionDeniedError(r.URL.Path)
			return err
		}

	}
	http.HandlerFunc((httpgzip.NewHandler(fs.FileHandler(ctx, upath), nil)).ServeHTTP)(w, r)
	return nil
}

// FileHandler returns the handler to serve the file
func (fs *FileServer) FileHandler(ctx context.Context, path string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		ext := filepath.Ext(path)
		if r.Method == "GET" {
			if len(ext) == 0 {
				err = fs.DBAPI.ListFiles(ctx, path, w, r)
			} else {
				err = fs.DBAPI.GetFile(ctx, path, w, r)
			}
		} else if r.Method == "POST" {
			err = fs.DBAPI.CreateFile(ctx, path, w, r, nil)
		} else if r.Method == "DELETE" {
			err = fs.DBAPI.DeleteFile(ctx, path, w, r, nil)
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprintf(w, `{"statusCode": 405, "message": "Method Not Allowed"}`)
			return
		}
		handleResponse(w, r, err, "%s files %s", r.Method, path)
	})
}

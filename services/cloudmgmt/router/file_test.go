package router_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/router"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
	"testing"
)

func TestFileServer(t *testing.T) {
	t.Parallel()
	t.Log("running file server tests")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	fileServer := &router.FileServer{DBAPI: dbAPI}
	_, filename, _, _ := runtime.Caller(0)
	resource := path.Join(path.Dir(filename), "../apitesthelper/youtube.png")
	user := &model.User{Email: "test@nutanix.com", Name: "Test", Role: "INFRA_ADMIN"}
	user.ID = base.GetUUID()
	user.TenantID = base.GetUUID()
	token := api.GetUserJWTToken(dbAPI, user, nil, router.DefaultTokenLifetimeSec, api.DefaultTokenType, nil)
	t.Run("Create/Get/Delete files", func(t *testing.T) {
		t.Log("running Create/Get/List/Delete file test")
		path1 := fmt.Sprintf("/v1/files/private/%s/datasource/123/youtube.png", user.TenantID)
		reader, contentType := getFormData(t, resource, path1)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, path1, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", contentType)
		// Upload to private
		err := fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must be ok but found %d", w.Code)
		}
		path2 := fmt.Sprintf("/v1/files/public/%s/datasource/124/youtube.png", user.TenantID)
		reader, contentType = getFormData(t, resource, path2)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodPost, path2, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", contentType)
		// Upload to public
		err = fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must be ok but found %d", w.Code)
		}
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, path1, reader)
		r.Header.Set("Content-Type", contentType)
		// Get from private without authorization
		err = fileServer.ServeFileRequest(w, r)
		require.Error(t, err, "Must fail")
		if w.Code == http.StatusOK {
			t.Fatalf("Http status must not be ok but found %d", w.Code)
		}
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, path1, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", contentType)
		// Get from private with authorization
		err = fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must be ok but found %d", w.Code)
		}
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, path2, reader)
		r.Header.Set("Content-Type", contentType)
		// Get from public without authorization
		err = fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must be ok but found %d", w.Code)
		}
		eTag := w.Header().Get("ETag")

		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, path1, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", contentType)
		r.Header.Set("If-None-Match", eTag)
		// Get unmodifed file
		err = fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusNotModified {
			t.Fatalf("Http status must be 304 but found %d", w.Code)
		}
		pth := fmt.Sprintf("/v1/files/private/%s/datasource/123/", user.TenantID)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodGet, pth, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", contentType)
		// List files
		err = fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must be ok but found %d", w.Code)
		}
		files := []string{}
		err = json.Unmarshal(w.Body.Bytes(), &files)
		require.NoError(t, err)
		if len(files) != 1 {
			t.Fatalf("Expected 1, found %d", len(files))
		}
		t.Logf("Files: %+v", files)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodDelete, path1, reader)
		r.Header.Set("Content-Type", contentType)
		// Delete file from private without authorization
		err = fileServer.ServeFileRequest(w, r)
		require.Error(t, err, "Must fail")
		if w.Code == http.StatusOK {
			t.Fatalf("Http status must not be ok but found %d", w.Code)
		}
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodDelete, path2, reader)
		r.Header.Set("Content-Type", contentType)
		// Delete file from public without authorization
		err = fileServer.ServeFileRequest(w, r)
		require.Error(t, err, "Must fail")
		if w.Code == http.StatusOK {
			t.Fatalf("Http status must not be ok but found %d", w.Code)
		}
		pth = fmt.Sprintf("/v1/files/private/%s", user.TenantID)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodDelete, pth, reader)
		r.Header.Set("Authorization", "Bearer "+token)
		r.Header.Set("Content-Type", contentType)
		// Delete file from private with authorization
		err = fileServer.ServeFileRequest(w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must be ok but found %d", w.Code)
		}
	})
}

// getFormData is a helper to create form upload data
func getFormData(t *testing.T, filename string, path string) (io.Reader, string) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	defer bodyWriter.Close()
	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	require.NoError(t, err, "error writing to buffer")

	// open file handle
	fh, err := os.Open(filename)
	require.NoError(t, err, "error opening file")
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	require.NoError(t, err)
	reader := ioutil.NopCloser(bodyBuf)
	contentType := bodyWriter.FormDataContentType()
	return reader, contentType
}

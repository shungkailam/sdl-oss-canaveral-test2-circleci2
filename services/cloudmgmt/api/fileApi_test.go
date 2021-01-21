package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"context"
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

func TestFileAPIs(t *testing.T) {
	t.Parallel()
	t.Log("running file APIs tests")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	_, filename, _, _ := runtime.Caller(0)
	resource := path.Join(path.Dir(filename), "../apitesthelper/youtube.png")
	basePath := base.GetUUID()
	ctx := context.Background()
	t.Run("Create/Get/List/Delete files", func(t *testing.T) {
		t.Log("running Create/Get/List/Delete file test")
		path1 := fmt.Sprintf("/public/%s/resource-id1/123/icon.png", basePath)
		reader, contentType := getFormData(t, resource, path1)
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", reader)
		r.Header.Set("Content-Type", contentType)
		// Upload the PNG file
		err := dbAPI.CreateFile(ctx, path1, w, r, nil)
		require.NoError(t, err)
		path2 := fmt.Sprintf("/public/%s/resource-id1/123/icon1.png", basePath)
		reader, contentType = getFormData(t, resource, path2)
		w = httptest.NewRecorder()
		r = httptest.NewRequest(http.MethodPost, "/", reader)
		r.Header.Set("Content-Type", contentType)
		// Upload the PNG file to a different path
		err = dbAPI.CreateFile(ctx, path2, w, r, nil)
		require.NoError(t, err)
		w = httptest.NewRecorder()
		// Download the first file
		err = dbAPI.GetFile(ctx, path1, w, r)
		require.NoError(t, err)
		if w.Code != http.StatusOK {
			t.Fatalf("Http status must 304 but found %d", w.Code)
		}
		eTag := w.Header().Get("ETag")
		// open output file for writing the downloaded file
		fo, err := os.Create("/tmp/test-out.png")
		require.NoError(t, err)
		defer fo.Close()
		if _, err := fo.Write(w.Body.Bytes()); err != nil {
			panic(err)
		}
		w = httptest.NewRecorder()
		r.Header.Set("If-None-Match", eTag)
		// Get unmodified file
		err = dbAPI.GetFile(ctx, path1, w, r)
		require.NoError(t, err)
		if w.Code != http.StatusNotModified {
			t.Fatalf("Http status must 304 but found %d", w.Code)
		}
		w = httptest.NewRecorder()
		pth := fmt.Sprintf("/public/%s/resource-id1/123", basePath)
		// List files
		err = dbAPI.ListFiles(ctx, pth, w, r)
		require.NoError(t, err)
		t.Log(string(w.Body.Bytes()))
		files := []string{}
		err = json.Unmarshal(w.Body.Bytes(), &files)
		require.NoError(t, err)
		if len(files) != 2 {
			t.Fatalf("Expected 2, found %d", len(files))
		}
		t.Logf("Files: %+v", files)
		// Delete the files
		err = dbAPI.PurgeFiles(ctx, basePath, "123")
		require.NoError(t, err)
		w = httptest.NewRecorder()
		pth = fmt.Sprintf("/public/%s/resource-id1/123", basePath)
		// List again and verify
		err = dbAPI.ListFiles(ctx, pth, w, r)
		require.NoError(t, err)
		files = []string{}
		err = json.Unmarshal(w.Body.Bytes(), &files)
		require.NoError(t, err)
		if len(files) != 0 {
			t.Fatalf("Expected 0, found %d", len(files))
		}
		dbAPI.PurgeFiles(ctx, basePath, "123")
	})
}

// getFormData is a helper to create form upload data
func getFormData(t *testing.T, filename string, path string) (io.Reader, string) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	defer bodyWriter.Close()
	// this step is very important
	fileWriter, err := bodyWriter.CreateFormFile("file", filename)
	require.NoError(t, err)
	// open file handle
	fh, err := os.Open(filename)
	require.NoError(t, err)
	defer fh.Close()

	//iocopy
	_, err = io.Copy(fileWriter, fh)
	require.NoError(t, err)
	reader := ioutil.NopCloser(bodyBuf)
	contentType := bodyWriter.FormDataContentType()
	return reader, contentType
}

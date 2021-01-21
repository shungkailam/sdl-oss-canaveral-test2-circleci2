package router_test

import (
	"bytes"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/router"
	"cloudservices/common/model"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func se(path string) string {
	return fmt.Sprintf("%s%s", apitesthelper.RESTServer, path)
}

func objToReader(obj interface{}) (io.Reader, error) {
	objData, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(objData), nil
}

func makePostRequest(url string, contentType string, body io.Reader) (*http.Request, error) {
	req, err := apitesthelper.NewHTTPRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return req, nil
}

func login(netClient *http.Client, email string, password string) (router.LoginResponse, error) {
	loginResponse := router.LoginResponse{}
	credReader, err := objToReader(model.Credential{
		Email:    email,
		Password: password,
	})
	if err != nil {
		return loginResponse, err
	}

	req, err := makePostRequest(se("/v1/login"), "application/json", credReader)
	if err != nil {
		return loginResponse, err
	}
	response, err := apitesthelper.ClientDo(netClient, req)
	if err != nil {
		return loginResponse, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return loginResponse, fmt.Errorf("%d: %s", response.StatusCode, response.Status)
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return loginResponse, err
	}
	err = json.Unmarshal(contents, &loginResponse)
	return loginResponse, err
}

// login using apitesthelper's default user password
func loginUser(t *testing.T, netClient *http.Client, user model.User) string {
	loginResponse, err := login(netClient, user.Email, apitesthelper.UserPassword)
	require.NoError(t, err)
	return loginResponse.Token
}

// login using the password provided in user object
func loginUser2(t *testing.T, netClient *http.Client, user model.User) string {
	loginResponse, err := login(netClient, user.Email, user.Password)
	require.NoError(t, err)
	return loginResponse.Token
}

func doGet2(netClient *http.Client, path string, token string) (*http.Response, error) {
	req, _ := apitesthelper.NewHTTPRequest("GET", se(path), nil)

	fmt.Printf("Calling GET on %s", path)
	fmt.Println()
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	return apitesthelper.ClientDo(netClient, req)
}

func doGet(netClient *http.Client, path string, token string, out interface{}) error {
	response, err := doGet2(netClient, path, token)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%d: %s", response.StatusCode, response.Status)
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(contents, out)
}

func doPost(netClient *http.Client, path string, token string, body interface{}, out interface{}) (string, error) {
	r, err := objToReader(body)
	if err != nil {
		return "", err
	}
	return doPost2(netClient, path, token, r, out, "")
}
func doPost2(netClient *http.Client, path string, token string, body io.Reader, out interface{}, multipartBoundary string) (string, error) {
	req, _ := apitesthelper.NewHTTPRequest("POST", se(path), body)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	if multipartBoundary != "" {
		req.Header.Set("Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", multipartBoundary))
	}

	response, err := apitesthelper.ClientDo(netClient, req)
	if err != nil {
		return "", err
	}
	reqID := response.Header.Get("X-Request-ID")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return reqID, fmt.Errorf("%d: %s", response.StatusCode, response.Status)
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return reqID, err
	}
	if out == nil {
		return reqID, nil
	}
	return reqID, json.Unmarshal(contents, out)
}

func doPut(netClient *http.Client, path string, token string, body interface{}, out interface{}) (string, error) {
	r, err := objToReader(body)
	if err != nil {
		return "", err
	}
	req, _ := apitesthelper.NewHTTPRequest("PUT", se(path), r)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := apitesthelper.ClientDo(netClient, req)
	if err != nil {
		return "", err
	}
	reqID := response.Header.Get("X-Request-ID")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return reqID, fmt.Errorf("%d: %s", response.StatusCode, response.Status)
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return reqID, err
	}
	return reqID, json.Unmarshal(contents, out)
}

func doDelete(netClient *http.Client, path string, token string, out interface{}) (string, error) {
	req, _ := apitesthelper.NewHTTPRequest("DELETE", se(path), nil)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := apitesthelper.ClientDo(netClient, req)
	if err != nil {
		return "", err
	}
	reqID := response.Header.Get("X-Request-ID")
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return reqID, fmt.Errorf("%d: %s", response.StatusCode, response.Status)
	}
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return reqID, err
	}
	return reqID, json.Unmarshal(contents, out)
}

// create entity
func createEntity(netClient *http.Client, path string, entity interface{}, token string) (model.CreateDocumentResponse, string, error) {
	resp := model.CreateDocumentResponse{}
	fmt.Printf("Calling POST on %s", path)
	fmt.Println()
	reqID, err := doPost(netClient, path, token, entity, &resp)
	return resp, reqID, err
}

// create entity V2
func createEntityV2(netClient *http.Client, path string, entity interface{}, token string) (model.CreateDocumentResponseV2, string, error) {
	resp := model.CreateDocumentResponseV2{}
	fmt.Printf("Calling POST on %s", path)
	fmt.Println()
	reqID, err := doPost(netClient, path, token, entity, &resp)
	return resp, reqID, err
}

// create entity V2O
func createEntityV2O(netClient *http.Client, path string, entity interface{}, token string, out interface{}) (string, error) {
	fmt.Printf("Calling POST on %s", path)
	fmt.Println()
	reqID, err := doPost(netClient, path, token, entity, out)
	return reqID, err
}

// update entity
func updateEntity(netClient *http.Client, path string, entity interface{}, token string) (model.UpdateDocumentResponse, string, error) {
	resp := model.UpdateDocumentResponse{}
	fmt.Printf("Calling PUT on %s", path)
	fmt.Println()
	reqID, err := doPut(netClient, path, token, entity, &resp)
	return resp, reqID, err
}

// update entity V2
func updateEntityV2(netClient *http.Client, path string, entity interface{}, token string) (model.UpdateDocumentResponseV2, string, error) {
	resp := model.UpdateDocumentResponseV2{}
	fmt.Printf("Calling PUT on %s", path)
	fmt.Println()
	reqID, err := doPut(netClient, path, token, entity, &resp)
	return resp, reqID, err
}

// delete entity
func deleteEntity(netClient *http.Client, path string, entityID string, token string) (model.DeleteDocumentResponse, string, error) {
	resp := model.DeleteDocumentResponse{}
	fpath := path
	if entityID != "" {
		fpath = fpath + "/" + entityID
	}
	fmt.Printf("Calling DELETE on %s", fpath)
	fmt.Println()
	reqID, err := doDelete(netClient, fpath, token, &resp)
	return resp, reqID, err
}

// delete entity V2
func deleteEntityV2(netClient *http.Client, path string, entityID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	resp := model.DeleteDocumentResponseV2{}
	fpath := path
	if entityID != "" {
		fpath = fpath + "/" + entityID
	}
	fmt.Printf("Calling DELETE on %s", fpath)
	fmt.Println()
	reqID, err := doDelete(netClient, fpath, token, &resp)
	return resp, reqID, err
}

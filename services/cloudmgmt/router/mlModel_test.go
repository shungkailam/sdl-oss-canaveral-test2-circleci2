package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	ML_MODEL_PATH                  = "/v1.0/mlmodels"
	ML_MODEL_VERSION_PATH_TEMPLATE = "/v1.0/mlmodels/%s/versions"
	DEBUG_ML_MODEL                 = false
)

// create ML Model
func createMLModel(netClient *http.Client, mdl *model.MLModelMetadata, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, ML_MODEL_PATH, *mdl, token)
	if err == nil {
		mdl.ID = resp.ID
	}
	return resp, reqID, err
}

// update ML Model
func updateMLModel(netClient *http.Client, modelID string, mdl model.MLModel, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", ML_MODEL_PATH, modelID), mdl, token)
}

// get ML Models
func getMLModels(netClient *http.Client, token string, pageIndex int, pageSize int) (model.MLModelListResponsePayload, error) {
	mdls := model.MLModelListResponsePayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", ML_MODEL_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &mdls)
	return mdls, err
}
func getMLModelsForProject(netClient *http.Client, projectID string, token string) (model.MLModelListResponsePayload, error) {
	mdls := model.MLModelListResponsePayload{}
	err := doGet(netClient, PROJECTS_PATH_NEW+"/"+projectID+"/mlmodels", token, &mdls)
	return mdls, err
}

// delete ML Model
func deleteMLModel(netClient *http.Client, modelID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, ML_MODEL_PATH, modelID, token)
}

// get ML Model by id
func getMLModelByID(netClient *http.Client, modelID string, token string) (model.MLModel, error) {
	mdl := model.MLModel{}
	err := doGet(netClient, ML_MODEL_PATH+"/"+modelID, token, &mdl)
	return mdl, err
}

func createMLModelVersion(netClient *http.Client, modelID string, modelVersion int, description string, token string, body io.Reader, multipartBoundary string) (model.CreateDocumentResponseV2, string, error) {
	path := fmt.Sprintf(ML_MODEL_VERSION_PATH_TEMPLATE, modelID)
	pathQ := fmt.Sprintf("%s?model_version=%d&description=%s", path, modelVersion, url.QueryEscape(description))
	resp := model.CreateDocumentResponseV2{}
	reqID, err := doPost2(netClient, pathQ, token, body, &resp, multipartBoundary)
	return resp, reqID, err
}

func updateMLModelVersion(netClient *http.Client, modelID string, modelVersion int, mdl model.MLModelVersion, token string) (model.UpdateDocumentResponseV2, string, error) {
	pathPrefix := fmt.Sprintf(ML_MODEL_VERSION_PATH_TEMPLATE, modelID)
	path := fmt.Sprintf("%s/%d", pathPrefix, modelVersion)
	return updateEntityV2(netClient, path, mdl, token)
}

func deleteMLModelVersion(netClient *http.Client, modelID string, modelVersion int, token string) (model.DeleteDocumentResponseV2, string, error) {
	path := fmt.Sprintf(ML_MODEL_VERSION_PATH_TEMPLATE, modelID)
	modelVersionString := fmt.Sprintf("%d", modelVersion)
	return deleteEntityV2(netClient, path, modelVersionString, token)
}

func createMLModelForProject(t *testing.T, netClient *http.Client, tenantID string, projectID string, token string) model.MLModel {
	mdlName := fmt.Sprintf("ml model name-%s", base.GetUUID())
	mdlDesc := "test ml model"

	mlDoc := model.MLModelMetadata{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		Name:          mdlName,
		Description:   mdlDesc,
		FrameworkType: model.FT_TENSORFLOW_DEFAULT,
		ProjectID:     projectID,
	}

	_, _, err := createMLModel(netClient, &mlDoc, token)
	require.NoError(t, err)
	// mlDoc.ID = resp.ID
	mdl := model.MLModel{MLModelMetadata: mlDoc}
	if DEBUG_ML_MODEL {
		t.Logf("create ML Model successful, %+v", mdl)
	}
	return mdl
}

func TestMLModel(t *testing.T) {
	t.Parallel()
	t.Log("running TestMLModel test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test ML Model", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		if DEBUG_ML_MODEL {
			t.Logf("created project: %+v", project)
		}

		mdl := createMLModelForProject(t, netClient, tenantID, project.ID, token)
		if DEBUG_ML_MODEL {
			t.Logf("created ML Model: %+v", mdl)
		}
		mdls, err := getMLModels(netClient, token, 0, 100)
		require.NoError(t, err)
		if DEBUG_ML_MODEL {
			t.Logf("got ML Models: %+v", mdls)
		}
		if len(mdls.MLModelList) != 1 {
			t.Fatalf("expected ml models count to be 1, but got %d", len(mdls.MLModelList))
		}
		if DEBUG_ML_MODEL {
			t.Logf("get model for id=%s", mdl.ID)
		}
		mdlJ, err := getMLModelByID(netClient, mdl.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(mdls.MLModelList[0], mdlJ) {
			t.Fatalf("expect ml model J equal, but %+v != %+v", mdls.MLModelList[0], mdlJ)
		}
		mdlsForProject, err := getMLModelsForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(mdlsForProject.MLModelList[0], mdls.MLModelList[0]) {
			t.Fatalf("expect ml model equal, but %+v != %+v", mdlsForProject.MLModelList[0], mdls.MLModelList[0])
		}

		// update ML Model
		modelID := mdl.ID
		mdl.ID = ""
		mdl.Description = fmt.Sprintf("%s-Updated", mdl.Description)
		ur, _, err := updateMLModel(netClient, modelID, mdl, token)
		require.NoError(t, err)
		if ur.ID != modelID {
			t.Fatal("expect update ML Model id to match")
		}

		mdlJ, err = getMLModelByID(netClient, modelID, token)
		require.NoError(t, err)
		if mdlJ.Description != mdl.Description {
			t.Fatalf("expect ml model description equal, but %s != %s", mdlJ.Description, mdl.Description)
		}

		modelVersion1 := 1
		description1 := "description for model v1"
		modelContentLen := 62623947
		s3Response, err := http.Get("https://s3-us-west-2.amazonaws.com/sherlock-object-detection-model/saved_model.zip")
		require.NoError(t, err)
		defer s3Response.Body.Close()
		if s3Response.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Error Status :%s, Code: %d", s3Response.Status, s3Response.StatusCode)
			t.Fatal(errors.New(errMsg))
		}

		cresp, _, err := createMLModelVersion(netClient, modelID, modelVersion1, description1, token, s3Response.Body, "")
		require.NoError(t, err)
		if cresp.ID != modelID {
			t.Fatalf("expect ml model id equal, but %s != %s", cresp.ID, modelID)
		}
		mdlJ, err = getMLModelByID(netClient, modelID, token)
		require.NoError(t, err)
		if DEBUG_ML_MODEL {
			t.Logf("ML model after create model version: %+v", mdlJ)
		}
		if len(mdlJ.ModelVersions) != 1 {
			t.Fatal("expect ml model versions count to be 1")
		}
		mv := mdlJ.ModelVersions[0]
		if mv.Description != description1 || mv.ModelVersion != modelVersion1 || int(mv.ModelSizeBytes) != modelContentLen {
			t.Fatal("expect ml model version object to match")
		}

		modelVersion2 := 2
		description2 := "description for model v2"
		s3Response2, err := http.Get("https://s3-us-west-2.amazonaws.com/sherlock-object-detection-model/saved_model.zip")
		require.NoError(t, err)
		defer s3Response2.Body.Close()
		if s3Response2.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Error Status :%s, Code: %d", s3Response2.Status, s3Response2.StatusCode)
			t.Fatal(errors.New(errMsg))
		}
		cresp, _, err = createMLModelVersion(netClient, modelID, 2, "description for model v2", token, s3Response2.Body, "")
		require.NoError(t, err)
		if cresp.ID != modelID {
			t.Fatalf("expect ml model id equal, but %s != %s", cresp.ID, modelID)
		}
		mdlJ, err = getMLModelByID(netClient, modelID, token)
		require.NoError(t, err)
		if DEBUG_ML_MODEL {
			t.Logf("ML model after create model version: %+v", mdlJ)
		}
		if len(mdlJ.ModelVersions) != 2 {
			t.Fatal("expect ml model versions count to be 2")
		}
		mv = mdlJ.ModelVersions[1]
		if mv.Description != description2 || mv.ModelVersion != modelVersion2 || int(mv.ModelSizeBytes) != modelContentLen {
			t.Fatal("expect ml model version object to match")
		}

		description1Updated := "updated description for model v1"
		mu := model.MLModelVersion{
			Description: description1Updated,
		}
		uresp, _, err := updateMLModelVersion(netClient, modelID, modelVersion1, mu, token)
		require.NoError(t, err)
		if uresp.ID != modelID {
			t.Fatalf("expect ml model id equal, but %s != %s", uresp.ID, modelID)
		}
		mdlJ, err = getMLModelByID(netClient, modelID, token)
		require.NoError(t, err)
		if DEBUG_ML_MODEL {
			t.Logf("ML model after update model version description: %+v", mdlJ)
		}
		if len(mdlJ.ModelVersions) != 2 {
			t.Fatal("expect ml model versions count to be 2")
		}
		mv = mdlJ.ModelVersions[0]
		if mv.Description != description1Updated || mv.ModelVersion != modelVersion1 || int(mv.ModelSizeBytes) != modelContentLen {
			t.Fatal("expect ml model version object to match")
		}

		dresp, _, err := deleteMLModelVersion(netClient, modelID, 1, token)
		require.NoError(t, err)
		if dresp.ID != modelID {
			t.Fatalf("expect ml model id equal, but %s != %s", dresp.ID, modelID)
		}
		mdlJ, err = getMLModelByID(netClient, modelID, token)
		require.NoError(t, err)
		if DEBUG_ML_MODEL {
			t.Logf("ML model after delete model version: %+v", mdlJ)
		}

		resp2, _, err := deleteMLModel(netClient, modelID, token)
		require.NoError(t, err)
		if resp2.ID != modelID {
			t.Fatal("delete ml model id mismatch")
		}
		resp, _, err := deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("delete project id mismatch")
		}
	})

}

func TestMLModelPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestMLModelPaging test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	rand1 := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test ML Model Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)
		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)

		// randomly create some ML Models
		n := 1 + rand1.Intn(11)
		for i := 0; i < n; i++ {
			createMLModelForProject(t, netClient, tenantID, project.ID, token)
		}

		mdls, err := getMLModels(netClient, token, 0, 100)
		require.NoError(t, err)
		if len(mdls.MLModelList) != n {
			t.Fatalf("expected ML models count to be %d, but got %d", n, len(mdls.MLModelList))
		}
		sort.Sort(model.MLModelsByID(mdls.MLModelList))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		mdls2 := []model.MLModel{}
		nRemain := n
		for i := 0; i < nPages; i++ {
			nmdls, err := getMLModels(netClient, token, i, pageSize)
			require.NoError(t, err)
			if nmdls.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, nmdls.PageIndex)
			}
			if nmdls.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, nmdls.PageSize)
			}
			if nmdls.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, nmdls.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(nmdls.MLModelList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nmdls.MLModelList))
			}
			nRemain -= pageSize
			for _, mdl := range model.MLModelsByID(nmdls.MLModelList) {
				mdls2 = append(mdls2, mdl)
			}
		}

		// verify paging api gives same result as old api
		for i := range mdls2 {
			if !reflect.DeepEqual(mdls.MLModelList[i], mdls2[i]) {
				t.Fatalf("expect ML model equal, but %+v != %+v", mdls.MLModelList[i], mdls2[i])
			}
		}

		// delete apps
		for _, mdl := range mdls2 {
			resp, _, err := deleteMLModel(netClient, mdl.ID, token)
			require.NoError(t, err)
			if resp.ID != mdl.ID {
				t.Fatal("delete ML model id mismatch")
			}
		}
		// delete project
		resp, _, err := deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("delete project id mismatch")
		}
	})

}

package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func getMLModelName() string {
	return "test-ml-model-" + base.GetUUID()
}

func createMLModel(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string) model.MLModel {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	mlDoc := model.MLModelMetadata{
		BaseModel: model.BaseModel{
			TenantID: tenantID,
		},
		Name:          getMLModelName(),
		Description:   "test-ml-model-desc",
		FrameworkType: model.FT_TENSORFLOW_DEFAULT,
		ProjectID:     projectID,
	}
	resp, err := dbAPI.CreateMLModel(ctx, &mlDoc, nil)
	require.NoError(t, err)
	t.Logf("create MLModel successful, %s", resp)
	mlModelID := resp.(model.CreateDocumentResponseV2).ID

	// GET ML model by ID
	mlModel, err := dbAPI.GetMLModel(ctx, mlModelID)
	require.NoError(t, err)
	return mlModel
}

func TestMLModel(t *testing.T) {
	t.Parallel()
	t.Log("running TestMLModel test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create MLModel", func(t *testing.T) {
		t.Log("running Create MLModel test")

		// create ML model
		mlDoc := model.MLModelMetadata{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Name:          getMLModelName(),
			Description:   "test-ml-model-desc",
			FrameworkType: model.FT_TENSORFLOW_DEFAULT,
			ProjectID:     projectID,
		}
		resp, err := dbAPI.CreateMLModel(ctx1, &mlDoc, nil)
		require.NoError(t, err)
		t.Logf("Got create ML model response: %+v", resp)

		// update ML model description
		mlDoc.Description = "test-ml-model-desc-updated"
		mlDoc.ID = resp.(model.CreateDocumentResponseV2).ID
		uresp, err := dbAPI.UpdateMLModel(ctx1, &mlDoc, nil)
		require.NoError(t, err)
		t.Logf("Got update ML model response: %+v", uresp)

		// GET ML model by ID
		mdl, err := dbAPI.GetMLModel(ctx1, mlDoc.ID)
		require.NoError(t, err)

		// compare model with input
		mlDoc.Version = mdl.Version
		mlDoc.CreatedAt = mdl.CreatedAt
		mlDoc.UpdatedAt = mdl.UpdatedAt
		mlDoc2 := model.MLModel{
			MLModelMetadata: mlDoc,
		}
		if !reflect.DeepEqual(mlDoc2, mdl) {
			t.Fatal("expect deep equal from GetMLModel")
		}

		// GET ML model by select all
		mdls, err := dbAPI.SelectAllMLModels(ctx1, nil)
		require.NoError(t, err)
		if len(mdls) != 1 {
			t.Fatal("expect SelectAllMLModels len = 1")
		}
		if !reflect.DeepEqual(mdl, mdls[0]) {
			t.Fatal("expect deep equal from SelectAllMLModels")
		}
		// GET ML model by project
		mdls, err = dbAPI.SelectAllMLModelsForProject(ctx1, projectID, nil)
		require.NoError(t, err)
		if len(mdls) != 1 {
			t.Fatal("expect SelectAllMLModelsForProject len = 1")
		}
		if !reflect.DeepEqual(mdl, mdls[0]) {
			t.Fatal("expect deep equal from SelectAllMLModelsForProject")
		}

		// now delete the ML model
		dresp, err := dbAPI.DeleteMLModel(ctx1, resp.(model.CreateDocumentResponseV2).ID, nil)
		require.NoError(t, err)
		t.Logf("Got delete ML model response: %+v", dresp)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := model.MLModelMetadata{
			BaseModel: model.BaseModel{
				ID:       id,
				TenantID: tenantID,
			},
			Name:          getMLModelName(),
			Description:   "test-ml-model-desc",
			FrameworkType: model.FT_TENSORFLOW_DEFAULT,
			ProjectID:     projectID,
		}
		return dbAPI.CreateMLModel(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetMLModel(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteMLModel(ctx1, id, nil)
	}))
}

func TestMLModelW(t *testing.T) {
	t.Parallel()
	t.Log("running TestMLModelW test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	t.Logf("created user, email=%s", user.Email)

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{user.ID}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteUser(ctx1, user.ID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create MLModelW", func(t *testing.T) {
		t.Log("running Create MLModelW test")

		mlDoc := model.MLModelMetadata{
			BaseModel: model.BaseModel{
				TenantID: tenantID,
			},
			Name:          getMLModelName(),
			Description:   "test-ml-model-desc",
			FrameworkType: model.FT_TENSORFLOW_DEFAULT,
			ProjectID:     projectID,
		}

		r, err := objToReader(mlDoc)
		require.NoError(t, err)
		// create ML model
		var w bytes.Buffer
		err = dbAPI.CreateMLModelW(ctx1, &w, r, func(ctx context.Context, i interface{}) error {
			t.Logf("CreateMLModelW callback: i=%+v", i)
			return nil
		})
		require.NoError(t, err)
		resp := model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)
		t.Logf("Got create ML model response: %+v", resp)

		mlDoc.Description = "test-ml-model-desc-updated"
		mlDoc.ID = resp.ID

		r, err = objToReader(mlDoc)
		require.NoError(t, err)
		// update ML model description
		var w2 bytes.Buffer
		err = dbAPI.UpdateMLModelW(ctx1, &w2, r, func(ctx context.Context, i interface{}) error {
			t.Logf("UpdateMLModelW callback: i=%+v", i)
			return nil
		})
		require.NoError(t, err)
		uresp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w2).Decode(&uresp)
		require.NoError(t, err)
		t.Logf("Got update ML model response: %+v", uresp)

		// upload ML model version 1 binary
		var ww bytes.Buffer
		modelVersion := 1
		description := "description for ml model v1"
		url := fmt.Sprintf("http://example.com/foo?model_version=%d&description=%s", modelVersion, url.QueryEscape(description))
		s3Response, err := http.Get("https://s3-us-west-2.amazonaws.com/sherlock-object-detection-model/saved_model.zip")
		require.NoError(t, err)
		defer s3Response.Body.Close()
		if s3Response.StatusCode != http.StatusOK {
			errMsg := fmt.Sprintf("Error Status :%s, Code: %d", s3Response.Status, s3Response.StatusCode)
			t.Fatal(errors.New(errMsg))
		}
		req, err := apitesthelper.NewHTTPRequest("POST", url, s3Response.Body)
		req.Header.Set("Content-Type", "application/octet-stream")
		require.NoError(t, err)
		err = dbAPI.CreateMLModelVersionW(ctx1, mlDoc.ID, &ww, req, func(ctx context.Context, i interface{}) error {
			t.Logf("CreateMLModelVersionW callback: i=%+v", i)
			return nil
		})
		require.NoError(t, err)
		// print response
		cvresp := model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&ww).Decode(&cvresp)
		require.NoError(t, err)
		t.Logf("Got create ML model version binary response: %+v", cvresp)

		// get pre-signed url for model v1 binary
		var ww2 bytes.Buffer
		url2 := fmt.Sprintf("http://example.com/foo?expiration_duration=%d", 5)
		req, err = apitesthelper.NewHTTPRequest("GET", url2, nil)
		require.NoError(t, err)
		err = dbAPI.GetMLModelVersionSignedURLW(ctx1, mlDoc.ID, modelVersion, &ww2, req)
		require.NoError(t, err)
		signedURL := model.MLModelVersionURLGetResponsePayload{}
		err = json.NewDecoder(&ww2).Decode(&signedURL)
		require.NoError(t, err)
		t.Logf("Got pre-signed url for ML model version binary: %s", signedURL.URL)

		// GET ML model by ID
		var w3 bytes.Buffer
		err = dbAPI.GetMLModelW(ctx1, mlDoc.ID, &w3, nil)
		require.NoError(t, err)
		mdl := model.MLModel{}
		err = json.NewDecoder(&w3).Decode(&mdl)
		require.NoError(t, err)
		if len(mdl.ModelVersions) != 1 {
			t.Fatalf("expect model versions count to be 1, but got %d", len(mdl.ModelVersions))
		}
		if mdl.ModelVersions[0].ModelVersion != modelVersion {
			t.Fatalf("expect model version to be %d, but got %d", modelVersion, mdl.ModelVersions[0].ModelVersion)
		}
		if mdl.ModelVersions[0].Description != description {
			t.Fatalf("expect model description to be %s, but got %s", description, mdl.ModelVersions[0].Description)
		}

		// compare model with input
		mlDoc.Version = mdl.Version
		mlDoc.CreatedAt = mdl.CreatedAt
		mlDoc.UpdatedAt = mdl.UpdatedAt
		mlDoc2 := model.MLModel{
			MLModelMetadata: mlDoc,
			ModelVersions:   mdl.ModelVersions,
		}
		if !reflect.DeepEqual(mlDoc2, mdl) {
			t.Fatal("expect deep equal from GetMLModelW")
		}

		// update model version description
		updatedDescription := "updated description for ml model v1"
		mlVerDoc := model.MLModelVersion{
			Description: updatedDescription,
		}
		r, err = objToReader(mlVerDoc)
		require.NoError(t, err)
		// update ML model version
		var wwv bytes.Buffer

		url = "http://example.com"
		req, err = apitesthelper.NewHTTPRequest("PUT", url, r)
		require.NoError(t, err)
		err = dbAPI.UpdateMLModelVersionW(ctx1, mlDoc.ID, modelVersion, &wwv, req, func(ctx context.Context, i interface{}) error {
			t.Fatal("expect callback for UpdateMLModelVersionW to not be called")
			return nil
		})
		require.NoError(t, err)
		err = dbAPI.GetMLModelW(ctx1, mlDoc.ID, &w3, nil)
		require.NoError(t, err)
		mdl = model.MLModel{}
		err = json.NewDecoder(&w3).Decode(&mdl)
		require.NoError(t, err)
		if mdl.ModelVersions[0].Description != updatedDescription {
			t.Fatalf("expect model updated description to be %s, but got %s", updatedDescription, mdl.ModelVersions[0].Description)
		}

		// GET ML model by select all
		var w4 bytes.Buffer
		err = dbAPI.SelectAllMLModelsW(ctx1, &w4, nil)
		rp := model.MLModelListResponsePayload{}
		err = json.NewDecoder(&w4).Decode(&rp)
		require.NoError(t, err)
		t.Logf("SelectAllMLModelsW response: %+v", rp)
		mdls := rp.MLModelList
		if len(mdls) != 1 {
			t.Fatal("expect SelectAllMLModelsW len = 1")
		}
		if !reflect.DeepEqual(mdl, mdls[0]) {
			t.Fatal("expect deep equal from SelectAllMLModelsW")
		}

		// GET ML model by project
		var w5 bytes.Buffer
		err = dbAPI.SelectAllMLModelsForProjectW(ctx1, projectID, &w5, nil)
		rp = model.MLModelListResponsePayload{}
		err = json.NewDecoder(&w5).Decode(&rp)
		require.NoError(t, err)
		t.Logf("SelectAllMLModelsForProjectW response: %+v", rp)
		mdls = rp.MLModelList
		if len(mdls) != 1 {
			t.Fatal("expect SelectAllMLModelsForProjectW len = 1")
		}
		if !reflect.DeepEqual(mdl, mdls[0]) {
			t.Fatal("expect deep equal from SelectAllMLModelsForProjectW")
		}

		// now delete the s3 object (note: DeleteMLModel will also delete all versions)
		dvresp, err := dbAPI.DeleteMLModelVersion(ctx1, mlDoc.ID, modelVersion, func(ctx context.Context, i interface{}) error {
			t.Logf("DeleteMLModelVersion callback: i=%+v", i)
			return nil
		})
		require.NoError(t, err)
		t.Logf("Got delete ML model version binary response: %+v", dvresp)

		// now delete the ML model
		dresp, err := dbAPI.DeleteMLModel(ctx1, resp.ID, func(ctx context.Context, i interface{}) error {
			t.Logf("DeleteMLModel callback: i=%+v", i)
			return nil
		})
		require.NoError(t, err)
		t.Logf("Got delete ML model response: %+v", dresp)

		// sleep 1 second to give callback a chance to run
		time.Sleep(1 * time.Second)
	})
}

package api_test

import (
	"bytes"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelmTemplate(t *testing.T) {
	t.Parallel()
	t.Log("running helm template tests")
	dbAPI := newObjectModelAPI(t)
	ctx := context.Background()
	testRun := func(getDataFn func(*testing.T, string, string) (io.Reader, string),
		chart, values string) {
		reader, contentType := getDataFn(t, chart, values)
		r := httptest.NewRequest(http.MethodPost, "/", reader)
		r.Header.Set("Content-Type", contentType)
		_, err := dbAPI.RunHelmTemplate(ctx, r, "")
		require.NoError(t, err)
	}
	t.Run("helm template", func(t *testing.T) {
		t.Log("running helm template on supported inputs")
		var testInputs = []struct {
			chart  string
			values string
			fn     func(t *testing.T, chart, values string) (io.Reader, string)
		}{
			{"ambassador-5.3.0.tgz", "", getHelmTemplateJsonData},
			{"redis-10.3.1.tgz", "values.yaml", getHelmTemplateJsonData},
			//{"redis-10.3.1.tgz", "", getHelmTemplateJsonData},
			{"redis-10.3.1.tgz", "values.yaml", getHelmTemplateFormData},
			//{"redis-10.3.1.tgz", "", getHelmTemplateFormData},
		}
		for _, ti := range testInputs {
			testRun(ti.fn, ti.chart, ti.values)
		}
	})
}

func TestHelmApplication(t *testing.T) {
	t.Parallel()
	t.Log("running helm application tests")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeID := edge.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{},
		[]string{}, []string{}, []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		})
	projectID := project.ID

	privilegedProject := createCategoryProjectCommon(t, dbAPI, tenantID, []string{},
		[]string{}, []string{}, []model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		}, func(p *model.Project) {
			p.Privileged = base.BoolPtr(true)
		})
	privilegedProjectID := privilegedProject.ID

	ctx1, _, _ := makeContext(tenantID, []string{projectID, privilegedProjectID})

	// get project
	project, err := dbAPI.GetProject(ctx1, projectID)
	require.NoError(t, err)

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteProject(ctx1, privilegedProjectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	getApplication := func(appID string) model.ApplicationV2 {
		app, err := dbAPI.GetApplication(ctx1, appID)
		require.NoError(t, err)
		return app.ToV2()
	}
	noApplication := func(appID string) {
		_, err := dbAPI.GetApplication(ctx1, appID)
		require.Error(t, err, "noApplication: err expected")
		_, ok := err.(*errcode.RecordNotFoundError)
		if !ok {
			t.Fatal("noApplication: RecordNotFoundError expected")
		}
	}

	t.Run("helm application", func(t *testing.T) {
		t.Log("running helm application CRUD operations")

		app := testApp(tenantID, projectID, "app name", []string{edgeID},
			[]model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				}}, nil,
		).ToV2()

		privilegedApp := testApp(tenantID, privilegedProjectID,
			"privileged app name", []string{edgeID}, []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				}}, nil,
		).ToV2()

		createHelmApp := func(reader io.Reader, contentType string,
			expectSuccess bool) (string, error) {
			r := httptest.NewRequest(http.MethodPost, "/", reader)
			r.Header.Set("Content-Type", contentType)
			w := httptest.NewRecorder()
			err := dbAPI.CreateHelmApplicationW(ctx1, "", w, r, nil)
			if err != nil {
				if expectSuccess {
					t.Fatal(err)
				} else {
					return "", err
				}
			}
			if w.Code != http.StatusOK {
				t.Fatalf("Http status not ok: %d", w.Code)
			}
			resp := model.CreateDocumentResponseV2{}
			err = json.NewDecoder(w.Result().Body).Decode(&resp)
			require.NoError(t, err)
			t.Logf("create helm app id: %s", resp.ID)
			return resp.ID, nil
		}

		updateHelmApp := func(reader io.Reader, contentType, appID string,
			expectSuccess bool) (string, error) {
			r := httptest.NewRequest(http.MethodPut, "/", reader)
			r.Header.Set("Content-Type", contentType)
			w := httptest.NewRecorder()
			err := dbAPI.UpdateHelmApplicationW(ctx1, appID, w, r, nil)
			if err != nil {
				if expectSuccess {
					t.Fatal(err)
				} else {
					return "", err
				}
			}
			if w.Code != http.StatusOK {
				t.Fatalf("Http status not ok: %d", w.Code)
			}
			resp := model.UpdateDocumentResponseV2{}
			err = json.NewDecoder(w.Result().Body).Decode(&resp)
			require.NoError(t, err)
			t.Logf("update helm app id: %s", resp.ID)
			return resp.ID, nil
		}

		testRun := func(getDataFn func(*testing.T, model.ApplicationV2,
			string, string) (io.Reader, string), chart, values string, privileged,
			expectSuccess, haveCRDs bool) {
			testApp := app
			if privileged {
				testApp = privilegedApp
			}
			reader, contentType := getDataFn(t, testApp, chart, values)
			appID, err := createHelmApp(reader, contentType, expectSuccess)
			if expectSuccess {
				require.NoError(t, err)
				testApp.ID = appID
				appC := getApplication(appID)
				if appC.AppManifest == "" {
					t.Fatal("empty app manifest")
				}
				if !appC.IsHelmApp() {
					t.Fatal("wrong app packaging type")
				}
				if appC.HelmMetadata.Metadata == "" {
					t.Fatal("empty chart yaml")
				}
				if haveCRDs {
					if appC.HelmMetadata.CRDs == "" {
						t.Fatal("expect to have CRDs")
					}
				} else {
					if appC.HelmMetadata.CRDs != "" {
						t.Fatal("expect to not have CRDs")
					}
				}
				if values != "" {
					if appC.HelmMetadata.Values == "" {
						t.Fatal("expect to have values yaml")
					}
				}
				// first test update with chart
				reader, contentType = getDataFn(t, testApp, chart, values)
				_, err = updateHelmApp(reader, contentType, appID, expectSuccess)
				require.NoError(t, err)
				// next test update without changing chart
				app := getApplication(appID)
				app.Name = app.Name + "-updated"
				reader, contentType = getDataFn(t, app, "", "")
				_, err = updateHelmApp(reader, contentType, appID, expectSuccess)
				require.NoError(t, err)
				testApp.ID = ""
				_, err = dbAPI.DeleteApplication(ctx1, appID, nil)
				require.NoError(t, err)
				noApplication(appID)
			} else {
				require.Error(t, err, "error expected")
			}
		}
		var testInputs = []struct {
			chart         string
			values        string
			privileged    bool
			expectSuccess bool
			haveCRDs      bool
			fn            func(t *testing.T, app model.ApplicationV2,
				chart, values string) (io.Reader, string)
		}{
			{"redis-10.3.1.tgz", "values.yaml", false, true, false, getCreateHelmApplicationFormData},
			{"redis-10.3.1.tgz", "", false, true, false, getCreateHelmApplicationJsonData},
			{"redis-10.3.1.tgz", "values.yaml", false, true, false, getCreateHelmApplicationJsonData},
			{"redis-10.3.1.tgz", "values.yaml", true, true, false, getCreateHelmApplicationJsonData},
			{"minio-5.0.5.tgz", "", false, true, false, getCreateHelmApplicationJsonData},
			{"minio-5.0.5.tgz", "", true, true, false, getCreateHelmApplicationJsonData},
			{"ambassador-5.3.0.tgz", "", true, true, true, getCreateHelmApplicationJsonData},
		}

		for _, ti := range testInputs {
			t.Logf("testing with ti: %+v", ti)
			testRun(ti.fn, ti.chart, ti.values, ti.privileged, ti.expectSuccess, ti.haveCRDs)
		}
	})
}

func getHelmTemplateJsonData(t *testing.T,
	chart, values string) (io.Reader, string) {
	chartData := getTestFileData(t, chart)
	valuesData := ""

	if values != "" {
		valuesData = getTestFileData(t, values)
	}
	content := model.HelmTemplateJSONParam{
		Chart:     chartData,
		Values:    valuesData,
		Release:   "helm-release-name",
		Namespace: "",
	}
	contentBytes := jsonMarshal(t, content)
	reader := bytes.NewReader(contentBytes)
	return reader, "application/json"
}

func getHelmTemplateFormData(t *testing.T,
	chart, values string) (io.Reader, string) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	defer bodyWriter.Close()
	writeFormFile(t, bodyWriter, "chart", chart)
	if values != "" {
		writeFormFile(t, bodyWriter, "values", values)
	}
	bodyWriter.WriteField("release", "helm-release-name")
	bodyWriter.WriteField("namespace", "")
	reader := ioutil.NopCloser(bodyBuf)
	contentType := bodyWriter.FormDataContentType()
	return reader, contentType
}

func getTestFileData(t *testing.T, filename string) string {
	_, basename, _, _ := runtime.Caller(0)
	chartFile := path.Join(path.Dir(basename), "./testdata/"+filename)
	return readFile(t, chartFile)
}

func writeFormFile(t *testing.T, w *multipart.Writer,
	fieldname, filename string) {
	_, basefile, _, _ := runtime.Caller(0)
	filepath := path.Join(path.Dir(basefile), "./testdata/"+filename)

	fileWriter, err := w.CreateFormFile(fieldname, filename)
	require.NoError(t, err)
	fh, err := os.Open(filepath)
	require.NoError(t, err)
	defer fh.Close()
	_, err = io.Copy(fileWriter, fh)
	require.NoError(t, err)
}

func getCreateHelmApplicationJsonData(t *testing.T, app model.ApplicationV2,
	chart, values string) (io.Reader, string) {
	valuesData := ""
	// allow chartData to be empty for helm app update case
	chartData := ""
	if values != "" {
		valuesData = getTestFileData(t, values)
	}
	if chart != "" {
		chartData = getTestFileData(t, chart)
	}
	// make app name unique
	app.Name = app.Name + "-" + base.GetUUID()
	x := model.HelmAppJSONParam{
		Chart:       chartData,
		Values:      valuesData,
		Application: app,
	}
	reader := objToReader2(t, x)
	return reader, "application/json"
}

func getCreateHelmApplicationFormData(t *testing.T, app model.ApplicationV2,
	chart, values string) (io.Reader, string) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	defer bodyWriter.Close()
	// allow chartData to be empty for helm app update case
	if chart != "" {
		writeFormFile(t, bodyWriter, "chart", chart)
	}
	if values != "" {
		writeFormFile(t, bodyWriter, "values", values)
	}
	app.Name = app.Name + "-" + base.GetUUID()
	bodyWriter.WriteField("application", objToString(t, app))
	reader := ioutil.NopCloser(bodyBuf)
	contentType := bodyWriter.FormDataContentType()
	return reader, contentType
}

func objToString(t *testing.T, obj interface{}) string {
	objData, err := json.Marshal(obj)
	require.NoError(t, err)
	return string(objData)
}

func readFile(t *testing.T, filename string) string {
	data, err := ioutil.ReadFile(filename)
	require.NoError(t, err)
	return base64.StdEncoding.EncodeToString(data)
}

func jsonMarshal(t *testing.T, i interface{}) []byte {
	data, err := json.Marshal(i)
	require.NoError(t, err)
	return data
}

package api

import (
	"bytes"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/golang/glog"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/release"
	"sigs.k8s.io/yaml"
)

// runHelmTemplate - expose helm template rendering functionality.
// Run helm template command on the chart with the values.
// @param chart required, path to chart tgz file
// @param values optional, path to values yaml file, ignored if empty
// @param releaseName required, helm release name
// @param namespace required, namespace to use for helm template
func runHelmTemplate(chart, values, releaseName,
	namespace string) (*release.Release, error) {
	settings := cli.New()
	actionConfig := new(action.Configuration)
	err := actionConfig.Init(settings.RESTClientGetter(),
		namespace, os.Getenv("HELM_DRIVER"), glog.Infof)
	if err != nil {
		return nil, err
	}
	ldr, err := loader.Loader(chart)
	if err != nil {
		return nil, err
	}
	chrt, err := ldr.Load()
	if err != nil {
		return nil, err
	}
	var vals chartutil.Values
	if values != "" {
		vals, err = chartutil.ReadValuesFile(values)
	}
	if err != nil {
		return nil, err
	}
	extraAPIs := []string{}
	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.Namespace = namespace
	client.ReleaseName = releaseName
	client.Replace = true    // Skip the name check
	client.ClientOnly = true // so we can run without k8s cluster
	client.DisableHooks = false
	client.APIVersions = chartutil.VersionSet(extraAPIs)
	return client.Run(chrt, vals)
}

// get helm manifest and hooks as a single string
func getManifestAndHooks(rel *release.Release) string {
	sa := []string{rel.Manifest}
	for _, hook := range rel.Hooks {
		sa = append(sa,
			fmt.Sprintf("---\n# Source: %s\n%s\n", hook.Path, hook.Manifest))
	}
	return strings.Join(sa, "")
}

// get helm CRDs as a string
func getCRDsAsString(rel *release.Release) string {
	crds := []string{}
	for _, f := range rel.Chart.CRDs() {
		crds = append(crds, fmt.Sprintf("---\n# Source: %s\n%s\n", f.Name, f.Data))
	}
	return strings.Join(crds, "")
}

func readerToString(reader io.Reader) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	return buf.String()
}

func readerToFile(reader io.Reader, filepath string) (int64, error) {
	file, err := os.Create(filepath)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	return io.Copy(file, reader)
}

// we use project namespace as helm release namespace
func namespaceFromProject(projectID string) string {
	return "project-" + projectID
}

// we use app id for helm release name
// we can't use app name since release name should be
// immutable, while app name could change
func releaseFromString(s string) string {
	// Add prefix r- since sha256 sum might begin with digit but release name can't.
	return fmt.Sprintf("r-%x", sha256.Sum256([]byte(s)))[:12]
}

// AppYaml structure to store helm app + yaml info
type AppYaml struct {
	App *model.ApplicationV2
	// app manifest (template yaml)
	Manifest string
	// metadata (chart.yaml)
	Metadata string
	// values.yaml
	Values string
	// Custom Resource Definitions
	CRDs string
}

// SyncFromRelease sync data from Release into AppYaml
func (apYml *AppYaml) SyncFromRelease(rel *release.Release) (err error) {
	baMetadata, err := yaml.Marshal(rel.Chart.Metadata)
	if err != nil {
		return
	}
	apYml.Manifest = getManifestAndHooks(rel)
	apYml.Metadata = string(baMetadata)
	apYml.CRDs = getCRDsAsString(rel)
	if apYml.App == nil {
		return
	}
	// sync data from apYml into apYml.App
	apYml.App.AppManifest = apYml.Manifest
	apYml.App.PackagingType = base.StringPtr(model.AppPackagingTypeHelm)
	apYml.App.HelmMetadata = &model.HelmAppMetadata{
		Metadata: apYml.Metadata,
		Values:   apYml.Values,
		CRDs:     apYml.CRDs,
	}
	return
}

// helmAppCreateParam is essentially model.HelmTemplateJSONParam,
// but adding an optional *model.ApplicationV2 so it can be used
// for helm create/update application payload as well
// Note: this struct is overloaded.
// When used in POST/PUT body, Chart and Values store
// base64 encoded data.
// When used as response in parseMultipartHelmRequest and
// parseJsonHelmRequest, Chart and Values store
// full path to tmp files containing the data
// so it can readily be consumed by helm SDK.
type helmAppCreateParam struct {
	Chart       string               `json:"chart"`
	Values      string               `json:"values,omitempty"`
	Application *model.ApplicationV2 `json:"application,omitempty"`
	Release     string               `json:"release"`
	Namespace   string               `json:"namespace,omitempty"`
}

func parseMediaType(req *http.Request) (string, map[string]string, error) {
	contentType := req.Header.Get("Content-Type")
	if contentType != "" {
		return mime.ParseMediaType(contentType)
	}
	return "", nil, fmt.Errorf("Bad content type")
}

// parseMultipartHelmRequest parse helm http request multipart/form-data payload
// this function can parse request payload for POST /helm/templates,
// POST /helm/apps and PUT /helm/apps/<app id>
// The application parameter is expected only for helm app create / update
// and not for helm templates. It is the string representation of model.ApplicationV2
// json. When application parameter is present,
// the namespace and release are derived from the application.
// In the response helmAppCreateParam, Chart and Values fields
// carry the full file path to the chart and values content.
func parseMultipartHelmRequest(context context.Context, req *http.Request, appID string,
	params map[string]string) (result *helmAppCreateParam, values, dir string, err error) {
	result = &helmAppCreateParam{Namespace: "default"}
	dir, err = ioutil.TempDir("", "helm")
	if err != nil {
		return
	}
	defer func() {
		// only do clean up on error
		// on success, caller must clean up dir when done
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	mr := multipart.NewReader(req.Body, params["boundary"])
	var p *multipart.Part
	for {
		p, err = mr.NextPart()
		if err == io.EOF {
			err = nil
			break
		}
		if err != nil {
			return
		}
		name := p.FormName()
		if name == "release" {
			result.Release = readerToString(p)
			continue
		} else if name == "namespace" {
			result.Namespace = readerToString(p)
			continue
		} else if name == "application" {
			app := readerToString(p)
			app2 := model.ApplicationV2{}
			err = json.Unmarshal([]byte(app), &app2)
			if err != nil {
				return
			}
			result.Application = &app2
			if appID != "" {
				app2.ID = appID
			} else {
				// when app is passed in, use it for release and namespace
				if app2.ID == "" {
					app2.ID = base.GetUUID()
				}
			}
			result.Namespace = namespaceFromProject(app2.ProjectID)
			result.Release = releaseFromString(app2.ID)
			continue
		}
		filename := p.FileName()
		if filename == "" {
			glog.Infof(base.PrefixRequestID(context,
				"Helm Template: skip part w/o filename, name: %s, part: %+v\n"),
				name, *p)
			continue
		}
		if name == "chart" {
			// helm sdk require chart file or dir, so save chart into file
			result.Chart = filepath.Join(dir, filename)
			_, err = readerToFile(p, result.Chart)
			if err != nil {
				return
			}
		} else if name == "values" {
			// values can be file or byte[], use file for uniformity
			result.Values = filepath.Join(dir, filename)
			_, err = readerToFile(p, result.Values)
			if err != nil {
				return
			}
			var ba []byte
			ba, err = ioutil.ReadFile(result.Values)
			if err != nil {
				return
			}
			values = string(ba)
		}
	}
	return
}

// parseJsonHelmRequest parse helm http request json payload
// this function can parse request payload for POST /helm/templates,
// POST /helm/apps and PUT /helm/apps/<app id>
// The json payload is unmarshaled into helmAppCreateParam where
// the Chart and Values, if non empty, carry base64 encoded data.
// The application parameter is expected only for helm app create / update
// and not for helm templates. When application parameter is present,
// the namespace and release are derived from the application.
// In the response helmAppCreateParam, Chart and Values fields
// carry the full file path to the chart and values content.
func parseJsonHelmRequest(context context.Context, req *http.Request,
	appID string) (result *helmAppCreateParam, values, dir string, err error) {
	result = &helmAppCreateParam{Namespace: "default"}
	dir, err = ioutil.TempDir("", "helm")
	if err != nil {
		return
	}
	defer func() {
		// only do clean up on error
		// on success, caller must clean up dir when done
		if err != nil {
			os.RemoveAll(dir)
		}
	}()
	// application/json
	ba, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	// unmarshal content
	appParam := helmAppCreateParam{}
	err = json.Unmarshal(ba, &appParam)
	if err != nil {
		return
	}
	var chartBytes []byte
	if appID != "" && appParam.Chart == "" {
		// allow update to not supply chart
		result.Chart = ""
	} else {
		chartBytes, err = base64.StdEncoding.DecodeString(appParam.Chart)
		if err != nil {
			return
		}
		// helm sdk require chart file or dir, so save chart into file
		result.Chart = filepath.Join(dir, "chart.tgz")
		err = ioutil.WriteFile(result.Chart, chartBytes, 0640)
		if err != nil {
			return
		}
	}

	// handle values
	if appParam.Values != "" {
		valuesBytes, err2 := base64.StdEncoding.DecodeString(appParam.Values)
		if err2 != nil {
			glog.Errorf(base.PrefixRequestID(context,
				"Helm Template: failed to decode values content: %s"), err2.Error())
			err = errcode.NewBadRequestError("Content")
			return
		}
		// values can be file or byte[], use file for uniformity
		result.Values = filepath.Join(dir, "values.yaml")
		err = ioutil.WriteFile(result.Values, valuesBytes, 0640)
		if err != nil {
			return
		}
		values = string(valuesBytes)
	}
	result.Release = appParam.Release
	if appParam.Namespace != "" {
		result.Namespace = appParam.Namespace
	}
	if appParam.Application != nil {
		result.Application = appParam.Application
		// when app is passed in, use it for release and namespace
		if appParam.Application.ID == "" {
			appParam.Application.ID = base.GetUUID()
		}
		result.Namespace = namespaceFromProject(appParam.Application.ProjectID)
		result.Release = releaseFromString(appParam.Application.ID)
	}
	return
}

//
// RunHelmTemplate - wrapper on runHelmTemplate to expose helm template
// rendering functionality. This function handles parsing of input
// from http request, then feed input into runHelmTemplate,
// finally it converts helm release.Release into AppYaml.
//
// the request content type can be multipart/form-data or application/json
// for application/json, the request payload should be of type
// helmAppCreateParam; for multipart/form-data, the request payload should be
// union of model.HelmTemplateParam and model.HelmAppCreateParam
func (dbAPI *dbObjectModelAPI) RunHelmTemplate(
	context context.Context, req *http.Request, appID string) (result AppYaml, err error) {
	// var chartFilePath, valuesFilePath string
	// var release string
	// namespace := "default"
	ctMultipart := true
	mediaType, params, err2 := parseMediaType(req)
	if err2 != nil {
		glog.Errorf(base.PrefixRequestID(
			context, "Helm Template: error parsing content type. Error: %s"),
			err2.Error())
		err = errcode.NewBadRequestError("Content-Type")
		return
	}
	switch mediaType {
	case "application/json":
		ctMultipart = false
	case "multipart/form-data":
	default:
		glog.Errorf(base.PrefixRequestID(context,
			"Helm Template: unsupported content type: %s"), mediaType)
		err = errcode.NewBadRequestError("Content-Type")
		return
	}

	var htParam *helmAppCreateParam
	var values, dir string
	if ctMultipart {
		htParam, values, dir, err2 = parseMultipartHelmRequest(context, req, appID, params)
	} else {
		htParam, values, dir, err2 = parseJsonHelmRequest(context, req, appID)
	}
	if err2 != nil {
		glog.Errorf(base.PrefixRequestID(context,
			"Helm Template: error parsing content, Error: %s"), err2.Error())
		err = errcode.NewBadRequestError("Content")
		return
	}
	defer os.RemoveAll(dir)

	if (htParam.Chart == "" && appID == "") || htParam.Release == "" {
		glog.Errorf(base.PrefixRequestID(context,
			"Helm Template: required content missing: chart or release"))
		err = errcode.NewBadRequestError("Content")
		return
	}
	var rel *release.Release
	if htParam.Chart != "" {
		rel, err =
			runHelmTemplate(htParam.Chart, htParam.Values, htParam.Release, htParam.Namespace)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context,
				"Helm Template: error rendering content, Error: %s, chart=%q\n"), err.Error(), htParam.Chart)
			err = errcode.NewBadRequestError("ContentRendering")
			return
		}

		result.Values = values
		result.App = htParam.Application
		err = result.SyncFromRelease(rel)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(context,
				"Helm Template: error marshal chart metadata, Error: %s"), err.Error())
			err = errcode.NewBadRequestError("ChartContentMetadata")
			return
		}
	} else {
		// update case with chart unchanged
		// check app packaging type and helm metadata remain unchanged
		result.App = htParam.Application
		app, err2 := dbAPI.GetApplication(context, appID)
		if err2 != nil {
			err = errcode.NewBadRequestError("AppID")
			return
		}
		appV2 := app.ToV2()
		if !appV2.IsHelmApp() || !result.App.IsHelmApp() {
			err = errcode.NewBadRequestError("AppPackagingType")
			return
		}
		if result.App.HelmMetadata == nil || appV2.HelmMetadata == nil {
			err = errcode.NewBadRequestError("AppHelmMetadata")
			return
		}
		if !reflect.DeepEqual(*result.App.HelmMetadata, *appV2.HelmMetadata) {
			err = errcode.NewBadRequestError("AppHelmMetadata")
			return
		}
		// take appManifest from current app
		htParam.Application.AppManifest = appV2.AppManifest
	}

	return
}

// RunHelmTemplateW wrapper on RunHelmTemplate to expose
// helm template rendering functionality
// on success it will write model.HelmTemplateResponse into the Writer w.
func (dbAPI *dbObjectModelAPI) RunHelmTemplateW(
	context context.Context, _ string, w io.Writer, req *http.Request,
	callback func(context.Context, interface{}) error) error {
	resp := model.HelmTemplateResponse{}
	apYml, err := dbAPI.RunHelmTemplate(context, req, "")
	if err != nil {
		return err
	}
	resp.AppManifest = apYml.Manifest
	resp.CRDs = apYml.CRDs
	resp.Metadata = apYml.Metadata
	resp.Values = apYml.Values
	return json.NewEncoder(w).Encode(resp)
}

func (dbAPI *dbObjectModelAPI) CreateHelmApplicationW(
	context context.Context, _ string, w io.Writer, req *http.Request,
	callback func(context.Context, interface{}) error) error {
	apYml, err := dbAPI.RunHelmTemplate(context, req, "")
	if err != nil {
		return err
	}
	if apYml.App == nil {
		return errcode.NewBadRequestError("AppContent")
	}

	// r is of type CreateDocumentResponse
	r, err := dbAPI.CreateApplicationV2(context, apYml.App, callback)
	if err != nil {
		return err
	}
	rd2 := model.CreateDocumentResponseV2{}
	rd := r.(model.CreateDocumentResponse)
	rd2.ID = rd.ID
	return json.NewEncoder(w).Encode(rd2)
}
func (dbAPI *dbObjectModelAPI) UpdateHelmApplicationW(
	context context.Context, id string, w io.Writer, req *http.Request,
	callback func(context.Context, interface{}) error) error {
	apYml, err := dbAPI.RunHelmTemplate(context, req, id)
	if err != nil {
		return err
	}
	if apYml.App == nil {
		return errcode.NewBadRequestError("AppContent")
	}
	apYml.App.ID = id
	// r is of type UpdateDocumentResponse
	r, err := dbAPI.UpdateApplicationV2(context, apYml.App, callback)
	if err != nil {
		return err
	}
	rd2 := model.UpdateDocumentResponseV2{}
	rd := r.(model.UpdateDocumentResponse)
	rd2.ID = rd.ID
	return json.NewEncoder(w).Encode(rd2)
}

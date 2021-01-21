package model

import (
	"cloudservices/common/apptemplate"
	"cloudservices/common/errcode"
	"cloudservices/common/kubeval"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	// Hardcoding for now later we can validate for individual version
	edgeK8sVersionBasic = "v1.15.4-standalone-strict-restricted"
	edgeK8sVersionFull  = "v1.15.4-standalone-strict"

	// DataIfcEndpointKindOut defines the kind OUT
	DataIfcEndpointKindOut = "OUT"

	// DataIfcEndpointKindIn defines the kind IN
	DataIfcEndpointKindIn = "IN"

	AppPackagingTypeHelm = "helm"
)

var reKind = regexp.MustCompile(`\s*kind:\s*([a-zA-Z][a-zA-Z0-9]*)\s*`)

func GetK8sSchemaVersion(privileged bool) string {
	if privileged {
		return edgeK8sVersionFull
	}
	return edgeK8sVersionBasic
}

// DataIfcEndpoint is the endpoint within a given data Ifc
type DataIfcEndpoint struct {
	//
	// ID is the UUID of the data Src/Ifc
	//
	// required: true
	ID string `json:"id"`

	//
	// Name is the name of the field within the data Src/Ifc.
	// This defines invariant between data ifc and application.
	// If an application is associated w/ a given data ifc field name, that field
	// name cannot be removed from the data ifc until application is no longer the consumer of the
	// field.
	// For example: "temperature"
	//
	// required: true
	Name string `json:"name" validate="range=0:200"`

	//
	// Value is the name of the endpoint within the data Src/Ifc. This could be a topic name or bucket name, etc.
	// Artifacts of a data Ifc be affected by this.
	// For example: It could an HLS out Ifc would append this to the last part
	// of the playback URL. An S3 Data Ifc in future, might include this in the s3 url exposed by the data Ifc.
	// This value can be updated using edit data source workflow and edge system is designed to handle such updates.
	//
	// For example: "video"
	//
	// required: true
	Value string `json:"value" validate="range=0:4096"`
}

type ApplicationCore struct {
	//
	// The application name.
	// Maximum length of 200 characters.
	// For example: FaceFeed
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// A description of the application.
	// Maximum length of 200 characters.
	//
	// required: false
	Description string `json:"description" db:"description" validate:"range=0:512"`
	//
	// Edges listed according to ID where the application is deployed.
	// Only relevant if the parent project EdgeSelectorType value is set to Explicit.
	//
	// required: false
	EdgeIDs []string `json:"edgeIds,omitempty"`
	//
	// Edges to be excluded from the application deployment.
	//
	// required: false
	ExcludeEdgeIDs []string `json:"excludeEdgeIds,omitempty"`
	//
	// Parent project ID.
	// Not required (to maintain backward compatibility).
	//
	// required: true
	ProjectID string `json:"projectId,omitempty" db:"project_id" validate:"range=0:64"`
	//
	// Select edges according to CategoryInfo.
	// Only relevant if the parent project EdgeSelectorType value is set to Category.
	//
	EdgeSelectors []CategoryInfo `json:"edgeSelectors"`
	//
	// State of this entity
	//
	// required: false
	State *string `json:"state,omitempty"`

	//
	// OriginSelectors is the list of CategoryInfo used as criteria
	// to feed data into applications.
	//
	// required: false
	OriginSelectors *[]CategoryInfo `json:"originSelectors" db:"origin_selectors"`

	//
	// DataIfcEndpoints is a list of endpoints exposed to an application.
	// For example: DataIfcEndpoint{Name: "test_topic", ID: "data-interface-uuid"}}
	// Expectation from the edge is to be able to provide these endpoints available to the application in some form, like ENVs
	// But cloudmgmt, does not assume anything about the implemenation on the edge.
	// required: false
	//
	DataIfcEndpoints []DataIfcEndpoint `json:"dataIfcEndpoints"`
}

// Application - the contents of an Application
// swagger:model Application
type Application struct {
	BaseModel
	ApplicationCore
	//
	// The YAML content for the application.
	//
	// required: true
	YamlData string `json:"yamlData" db:"yaml_data"`
	//
	// Only pre-pull images on service domains w/o doing an actual update.
	// Service domain which have not yet deployed the app will deploy
	// application like usual.
	// Update will commence once this flag is unset.
	//
	// required: false
	OnlyPrePullOnUpdate bool `json:"onlyPrePullOnUpdate" db:"only_pre_pull_on_update"`

	// PackagingType vanilla or helm, nil = vanilla
	PackagingType *string `json:"packagingType"`

	// HelmMetadata not nil iff PackagingType = helm
	HelmMetadata *HelmAppMetadata `json:"helmMetadata"`
}

// ApplicationV2 - the contents of an application
// swagger:model ApplicationV2
type ApplicationV2 struct {
	BaseModel
	ApplicationCore
	//
	// The kubernetes manifest for the application in YAML format.
	//
	// required: true
	AppManifest string `json:"appManifest" db:"yaml_data"`
	//
	// Only pre-pull images on service domains w/o doing an actual update.
	// Service domain which have not yet deployed the app will deploy
	// application like usual.
	// Update will commence once this flag is unset.
	//
	// required: false
	OnlyPrePullOnUpdate bool `json:"onlyPrePullOnUpdate" db:"only_pre_pull_on_update"`

	// PackagingType vanilla or helm, nil = vanilla
	PackagingType *string `json:"packagingType"`

	// HelmMetadata not nil iff PackagingType = helm
	HelmMetadata *HelmAppMetadata `json:"helmMetadata"`
}

// HelmAppMetadata additional metadata for Helm-chart based application
type HelmAppMetadata struct {
	// required: true
	// Helm Chart.yaml as a string
	Metadata string `json:"metadata"`
	// values.yaml as a string
	Values string `json:"values,omitempty"`
	// Helm chart CRDs as a string
	CRDs string `json:"crds,omitempty"`
}

// ApplicationContainersBaseObject - appID and edgeID for which the containers will
// listed.
// swagger:model ApplicationContainersBaseObject
type ApplicationContainersBaseObject struct {
	ApplicationID string `json:"applicationId"`
	EdgeID        string `json:"edgeId"`
}

// ApplicationContainers encapsulates the container names
// for a specific application on a specific edge.
// swagger:model ApplicationContainers
type ApplicationContainers struct {
	ApplicationContainersBaseObject
	ContainerNames []string `json:"containerNames"`
}

func (app Application) ToV2() ApplicationV2 {
	return ApplicationV2{
		BaseModel:           app.BaseModel,
		ApplicationCore:     app.ApplicationCore,
		AppManifest:         app.YamlData,
		OnlyPrePullOnUpdate: app.OnlyPrePullOnUpdate,
		PackagingType:       app.PackagingType,
		HelmMetadata:        app.HelmMetadata,
	}
}
func (app ApplicationV2) FromV2() Application {
	return Application{
		BaseModel:           app.BaseModel,
		ApplicationCore:     app.ApplicationCore,
		YamlData:            app.AppManifest,
		OnlyPrePullOnUpdate: app.OnlyPrePullOnUpdate,
		PackagingType:       app.PackagingType,
		HelmMetadata:        app.HelmMetadata,
	}
}

func (app ApplicationV2) GetCrdKinds() []string {
	crds := ""
	if app.PackagingType != nil && *app.PackagingType == AppPackagingTypeHelm && app.HelmMetadata != nil {
		crds = app.HelmMetadata.CRDs
	}
	return GetCrdKinds(crds)
}
func (app ApplicationV2) GetCRDs() string {
	crds := ""
	if app.PackagingType != nil && *app.PackagingType == AppPackagingTypeHelm && app.HelmMetadata != nil {
		crds = app.HelmMetadata.CRDs
	}
	return crds
}
func (app Application) GetCrdKinds() []string {
	return app.ToV2().GetCrdKinds()
}
func (app Application) GetCRDs() string {
	return app.ToV2().GetCRDs()
}

// ApplicationCreateParam is Application used as API parameter
// swagger:parameters ApplicationCreate
type ApplicationCreateParam struct {
	// Describes the application creation request.
	// in: body
	// required: true
	Body *Application `json:"body"`
}

// ApplicationCreateParamV2 is ApplicationV2 used as API parameter
// swagger:parameters ApplicationCreateV2
type ApplicationCreateParamV2 struct {
	// Describes the application creation request.
	// in: body
	// required: true
	Body *ApplicationV2 `json:"body"`
}

// GetApplicationContainersResponse is the API response that
// returns a list of container names for a given app on a given edge.
// swagger:response GetApplicationContainersResponse
type GetApplicationContainersResponse struct {
	// in: body
	// required: true
	Payload *ApplicationContainers
}

// ApplicationUpdateParam is Application used as API parameter
// swagger:parameters ApplicationUpdate ApplicationUpdateV2
type ApplicationUpdateParam struct {
	// in: body
	// required: true
	Body *Application `json:"body"`
}

// ApplicationUpdateParamV2 is ApplicationV2 used as API parameter
// swagger:parameters ApplicationUpdateV3
type ApplicationUpdateParamV2 struct {
	// in: body
	// required: true
	Body *ApplicationV2 `json:"body"`
}

// Ok
// swagger:response ApplicationGetResponse
type ApplicationGetResponse struct {
	// in: body
	// required: true
	Payload *Application
}

// Ok
// swagger:response ApplicationGetResponseV2
type ApplicationGetResponseV2 struct {
	// in: body
	// required: true
	Payload *ApplicationV2
}

// Ok
// swagger:response ApplicationListResponse
type ApplicationListResponse struct {
	// in: body
	// required: true
	Payload *[]Application
}

// Ok
// swagger:response ApplicationListResponseV2
type ApplicationListResponseV2 struct {
	// in: body
	// required: true
	Payload *ApplicationListResponsePayload
}

// payload for ApplicationListResponseV2
type ApplicationListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of applications
	// required: true
	ApplicationListV2 []ApplicationV2 `json:"result"`
}

// swagger:parameters ApplicationList ApplicationListV2 ApplicationGet ApplicationGetV2 ApplicationCreate ApplicationCreateV2 ApplicationUpdate ApplicationUpdateV2 ApplicationUpdateV3 ApplicationDelete ApplicationDeleteV2 ProjectGetApplications ProjectGetApplicationsV2 GetApplicationContainers RenderApplication HelmAppCreate HelmValuesCreate HelmAppGetYaml
// in: header
type applicationAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// swagger:parameters HelmAppCreate HelmValuesCreate
// in: formData
// swagger:file
type HelmAppCreateBodyParam struct {
	// required: true
	// swagger:file
	// in: formData
	Payload *os.File
}

// Ok
// swagger:response HelmAppGetYamlResponse
type HelmAppYamlResponse struct {
	// ID of the entity
	// Maximum character length is 64 for project, category, and runtime environment,
	// 36 for other entity types.
	ID     string `json:"id" db:"id" validate:"range=0:64,ignore=create"`
	Status string `json:"status"`
	Yaml   string `json:"yaml"`
}

// ObjectRequestBaseApplication is used as a websocket Application message
// swagger:model ObjectRequestBaseApplication
type ObjectRequestBaseApplication struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Application `json:"doc"`
}

// ObjectRequestBaseApplicationContainers is used as a websocket "getApplicationContainers" message
// swagger:model ObjectRequestBaseApplicationContainers
type ObjectRequestBaseApplicationContainers struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ApplicationContainersBaseObject `json:"doc"`
}

// EdgeService is service definition originating at the edge
// swagger:model EdgeService
type EdgeService struct {
	// required: true
	Endpoint string `json:"endpoint"`
}

// RenderApplicationPayload describes edge services on edge for app template
// engine to render application YAML.
// swagger:model RenderApplicationPayload
type RenderApplicationPayload struct {
	EdgeServices map[string]EdgeService `json:"edgeServices"`
}

// RenderApplicationParam is collection of edge-specific parameters
// swagger:parameters RenderApplication
type RenderApplicationParam struct {
	// in: path
	// required: true
	ID string `json:"id"`
	// in: path
	// required: true
	EdgeID string `json:"edgeId"`
	// in: body
	// required: true
	Payload *RenderApplicationPayload
}

// RenderApplicationResponsePayload containers rendered application
// template along with list of services referenced in YAML.
// swagger:model RenderApplicationResponsePayload
type RenderApplicationResponsePayload struct {
	AppYaml      string   `json:"appYaml"`
	EdgeServices []string `json:"edgeServices"`
}

// Ok
// swagger:response RenderApplicationResponse
type RenderApplicationResponse struct {
	// in: body
	// required: true
	Payload *RenderApplicationResponsePayload
}

func (doc Application) GetProjectID() string {
	return doc.ProjectID
}

func (doc ApplicationV2) GetProjectID() string {
	return doc.ProjectID
}

func (doc Application) GetEntityState() EntityState {
	if doc.State == nil {
		return DeployEntityState
	}
	return EntityState(*doc.State)
}

func (doc ApplicationV2) GetEntityState() EntityState {
	if doc.State == nil {
		return DeployEntityState
	}
	return EntityState(*doc.State)
}

func (doc Application) IsHelmApp() bool {
	return doc.PackagingType != nil && *doc.PackagingType == AppPackagingTypeHelm
}
func (doc ApplicationV2) IsHelmApp() bool {
	return doc.PackagingType != nil && *doc.PackagingType == AppPackagingTypeHelm
}

type ApplicationsByID []Application

func (a ApplicationsByID) Len() int           { return len(a) }
func (a ApplicationsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ApplicationsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (apps ApplicationsByID) ToV2() []ApplicationV2 {
	v2Apps := []ApplicationV2{}
	for _, app := range apps {
		v2Apps = append(v2Apps, app.ToV2())
	}
	return v2Apps
}

type ApplicationsByIDV2 []ApplicationV2

func (a ApplicationsByIDV2) Len() int           { return len(a) }
func (a ApplicationsByIDV2) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ApplicationsByIDV2) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (v2Apps ApplicationsByIDV2) FromV2() []Application {
	apps := []Application{}
	for _, v2App := range v2Apps {
		apps = append(apps, v2App.FromV2())
	}
	return apps
}

type DataIfcEndpointsByID []DataIfcEndpoint

func (a DataIfcEndpointsByID) Len() int           { return len(a) }
func (a DataIfcEndpointsByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a DataIfcEndpointsByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func validateDataIfcEndpoint(model *DataIfcEndpoint) error {
	if model.Name == "" {
		return errcode.NewBadRequestError("DataIfcEndpoint/Name")
	}
	if model.Value == "" {
		return errcode.NewBadRequestError("DataIfcEndpoint/Value")
	}
	if model.ID == "" {
		return errcode.NewBadRequestError("DataIfcEndpoint/ID")
	}
	return nil
}

func GetCrdKinds(crds string) (crdKinds []string) {
	if crds == "" {
		return
	}
	ss := strings.Split(crds, "\n")
	for _, s := range ss {
		r := reKind.FindSubmatch([]byte(s))
		if len(r) > 1 {
			kind := string(r[1])
			if kind != "CustomResourceDefinition" {
				crdKinds = append(crdKinds, kind)
			}
		}
	}
	return
}

func ValidateApplication(model *Application, schemaVersion string) error {
	if model == nil {
		return errcode.NewBadRequestError("Application")
	}
	model.Name = strings.TrimSpace(model.Name)
	model.ProjectID = strings.TrimSpace(model.ProjectID)
	if model.State != nil {
		if len(*model.State) == 0 || *model.State == string(DeployEntityState) {
			// Backward compatibility
			model.State = nil
		} else if *model.State != string(UndeployEntityState) {
			return errcode.NewBadRequestError("State")
		}
	}
	if model.IsHelmApp() {
		if model.HelmMetadata == nil || strings.TrimSpace(model.HelmMetadata.Metadata) == "" {
			return errcode.NewBadRequestExError("HelmMetadata", fmt.Sprintf("Helm Metadata is not set"))
		}
	} else {
		if model.HelmMetadata != nil {
			return errcode.NewBadRequestExError("HelmMetadata", fmt.Sprintf("Helm Metadata must not be set when PackagingType is not helm"))
		}
	}
	// Template variables are supposed to be quoted for YAML validation to work.
	// We take advantage of this fact by using unitialized parameters to validate
	// template itself. For instance ENG-197773 deals with invalid function usage in
	// app template. This can be detected w/o using real parameters.
	params := &apptemplate.AppParameters{}
	renderedYaml, _, err := apptemplate.RenderWithParams(params, model.YamlData)
	if err != nil {
		return errcode.NewBadRequestExError("YamlData",
			fmt.Sprintf("Template could not be rendered: %s", err.Error()))
	}
	results, err := kubeval.Validate([]byte(renderedYaml), "application.yaml", schemaVersion, model.GetCRDs())
	if err != nil {
		return errcode.NewBadRequestExError("YamlData", fmt.Sprintf("%+v", err.Error()))
	}

	for _, result := range results {
		if result.Kind == "" {
			return errcode.NewBadRequestExError("YamlData", fmt.Sprintf("Kind is not set"))
		}
		if len(result.Errors) != 0 {
			var errorList []string
			for _, err := range result.Errors {
				// don't just use error description, keep some context info as well
				errorList = append(errorList, fmt.Sprintf("[%s] %+v", result.Kind, err))
			}
			// When there are multiple errors, ordering of the errors is not guranteed, so we try to fix the order here
			sort.Strings(errorList)
			errors := strings.Join(errorList, ", ")
			return errcode.NewBadRequestExError("YamlData", fmt.Sprintf("Yaml has following errors: %s", errors))
		}
	}

	for _, e := range model.DataIfcEndpoints {
		err = validateDataIfcEndpoint(&e)
		if err != nil {
			return err
		}
	}
	return nil
}

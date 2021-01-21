package router

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	jaegerModels "github.com/jaegertracing/jaeger/model/json"
	"github.com/julienschmidt/httprouter"
	"github.com/kiali/k-charted/model"
	kmodel "github.com/kiali/k-charted/model"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

/////////////////////
// SWAGGER PARAMETERS - GENERAL
/////////////////////

// swagger:parameters appMetrics appDetails graphApp graphAppVersion appDashboard appHealth appOverview
type AppParam struct {
	// The app name (label value).
	//
	// in: path
	// required: true
	Name string `json:"app"`
}

// swagger:parameters graphAppVersion
type AppVersionParam struct {
	// The app version (label value).
	//
	// in: path
	// required: false
	Name string `json:"version"`
}

// swagger:parameters podLogs
type ContainerParam struct {
	// The pod container name. Optional for single-container pod. Otherwise required.
	//
	// in: query
	// required: false
	Name string `json:"container"`
}

// swagger:parameters istioConfigList workloadList workloadDetails serviceDetails serviceHealth spansList tracesList errorTraces tracesDetail workloadValidations appList serviceMetrics appMetrics workloadMetrics istioConfigDetails istioConfigDetailsSubtype istioConfigDelete istioConfigDeleteSubtype istioConfigUpdate istioConfigUpdateSubtype serviceList appDetails graphApp graphAppVersion graphNamespace graphService graphWorkload namespaceMetrics customDashboard appDashboard serviceDashboard workloadDashboard istioConfigCreate istioConfigCreateSubtype namespaceTls podDetails podLogs getThreeScaleService postThreeScaleService patchThreeScaleService deleteThreeScaleService namespaceValidations getIter8Experiments postIter8Experiments patchIter8Experiments deleteIter8Experiments namespaceHealth namespaceMetrics workloadHealth appHealth appOverview namespaceOverview serviceOverview workloadOverview
type NamespaceParam struct {
	// The namespace name.
	//
	// in: path
	// required: true
	Name string `json:"namespace"`
}

// swagger:parameters getIter8Experiments patchIter8Experiments deleteIter8Experiments
type NameParam struct {
	// The name param
	//
	// in: path
	// required: true
	Name string `json:"name"`
}

// swagger:parameters istioConfigDetails istioConfigDetailsSubtype istioConfigDelete istioConfigDeleteSubtype istioConfigUpdate istioConfigUpdateSubtype
type ObjectNameParam struct {
	// The Istio object name.
	//
	// in: path
	// required: true
	Name string `json:"object"`
}

// swagger:parameters istioConfigDetails istioConfigDetailsSubtype istioConfigDelete istioConfigDeleteSubtype istioConfigUpdate istioConfigUpdateSubtype istioConfigCreate istioConfigCreateSubtype
type ObjectTypeParam struct {
	// The Istio object type.
	//
	// in: path
	// required: true
	// pattern: ^(gateways|virtualservices|destinationrules|serviceentries|rules|quotaspecs|quotaspecbindings)$
	Name string `json:"object_type"`
}

// swagger:parameters podDetails podLogs
type PodParam struct {
	// The pod name.
	//
	// in: path
	// required: true
	Name string `json:"pod"`
}

// swagger:parameters serviceDetails spansList tracesList errorTraces tracesDetail serviceMetrics serviceHealth graphService serviceDashboard getThreeScaleService patchThreeScaleService deleteThreeScaleService serviceOverview
type ServiceParam struct {
	// The service name.
	//
	// in: path
	// required: true
	Name string `json:"service"`
}

// swagger:parameters podLogs
type SinceTimeParam struct {
	// The start time for fetching logs. UNIX time in seconds. Default is all logs.
	//
	// in: query
	// required: false
	Name string `json:"sinceTime"`
}

// swagger:parameters customDashboard
type DashboardParam struct {
	// The dashboard resource name.
	//
	// in: path
	// required: true
	Name string `json:"dashboard"`
}

// swagger:parameters workloadDetails workloadValidations workloadMetrics graphWorkload workloadDashboard workloadHealth workloadOverview
type WorkloadParam struct {
	// The workload name.
	//
	// in: path
	// required: true
	Name string `json:"workload"`
}

/////////////////////
// SWAGGER PARAMETERS - GRAPH
// - keep this alphabetized
/////////////////////

// swagger:parameters graphApp graphAppVersion graphNamespaces graphService graphWorkload
type AppendersParam struct {
	// Comma-separated list of Appenders to run. Available appenders: [deadNode, istio, responseTime, securityPolicy, serviceEntry, sidecarsCheck, unusedNode].
	//
	// in: query
	// required: false
	// default: run all appenders
	Name string `json:"appenders"`
}

// swagger:parameters graphApp graphAppVersion graphNamespaces graphService graphWorkload
type DurationGraphParam struct {
	// Query time-range duration (Golang string duration).
	//
	// in: query
	// required: false
	// default: 10m
	Name string `json:"duration"`
}

// swagger:parameters graphApp graphAppVersion graphNamespaces graphService graphWorkload
type GraphTypeParam struct {
	// Graph type. Available graph types: [app, service, versionedApp, workload].
	//
	// in: query
	// required: false
	// default: workload
	Name string `json:"graphType"`
}

// swagger:parameters graphApp graphAppVersion graphNamespaces graphService graphWorkload
type GroupByParam struct {
	// App box grouping characteristic. Available groupings: [app, none, version].
	//
	// in: query
	// required: false
	// default: none
	Name string `json:"groupBy"`
}

// swagger:parameters graphApp graphAppVersion graphNamespaces graphWorkload
type InjectServiceNodes struct {
	// Flag for injecting the requested service node between source and destination nodes.
	//
	// in: query
	// required: false
	// default: false
	Name string `json:"injectServiceNodes"`
}

// swagger:parameters graphNamespaces
type NamespacesParam struct {
	// Comma-separated list of namespaces to include in the graph. The namespaces must be accessible to the client.
	//
	// in: query
	// required: true
	Name string `json:"namespaces"`
}

// swagger:parameters graphApp graphAppVersion graphNamespaces graphService graphWorkload
type QueryTimeParam struct {
	// Unix time (seconds) for query such that time range is [queryTime-duration..queryTime]. Default is now.
	//
	// in: query
	// required: false
	// default: now
	Name string `json:"queryTime"`
}

/////////////////////
// SWAGGER PARAMETERS - METRICS
/////////////////////

// swagger:parameters customDashboard
type AdditionalLabelsParam struct {
	// In custom dashboards, additional labels that are made available for grouping in the UI, regardless which aggregations are defined in the MonitoringDashboard CR
	//
	// in: query
	// required: false
	Name string `json:"additionalLabels"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type AvgParam struct {
	// Flag for fetching histogram average. Default is true.
	//
	// in: query
	// required: false
	// default: true
	Name bool `json:"avg"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type ByLabelsParam struct {
	// List of labels to use for grouping metrics (via Prometheus 'by' clause).
	//
	// in: query
	// required: false
	Name []string `json:"byLabels[]"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics appDashboard serviceDashboard workloadDashboard
type DirectionParam struct {
	// Traffic direction: 'inbound' or 'outbound'.
	//
	// in: query
	// required: false
	// default: outbound
	Name string `json:"direction"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type DurationParam struct {
	// Duration of the query period, in seconds.
	//
	// in: query
	// required: false
	// default: 1800
	Name int `json:"duration"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics
type FiltersParam struct {
	// List of metrics to fetch. Fetch all metrics when empty. List entries are Kiali internal metric names.
	//
	// in: query
	// required: false
	Name []string `json:"filters[]"`
}

// swagger:parameters customDashboard
type LabelsFiltersParam struct {
	// In custom dashboards, labels filters to use when fetching metrics, formatted as key:value pairs. Ex: "app:foo,version:bar".
	//
	// in: query
	// required: false
	//
	Name string `json:"labelsFilters"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type QuantilesParam struct {
	// List of quantiles to fetch. Fetch no quantiles when empty. Ex: [0.5, 0.95, 0.99].
	//
	// in: query
	// required: false
	Name []string `json:"quantiles[]"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type RateFuncParam struct {
	// Prometheus function used to calculate rate: 'rate' or 'irate'.
	//
	// in: query
	// required: false
	// default: rate
	Name string `json:"rateFunc"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type RateIntervalParam struct {
	// Interval used for rate and histogram calculation.
	//
	// in: query
	// required: false
	// default: 1m
	Name string `json:"rateInterval"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics appDashboard serviceDashboard workloadDashboard
type RequestProtocolParam struct {
	// Desired request protocol for the telemetry: For example, 'http' or 'grpc'.
	//
	// in: query
	// required: false
	// default: all protocols
	Name string `json:"requestProtocol"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics appDashboard serviceDashboard workloadDashboard
type ReporterParam struct {
	// Istio telemetry reporter: 'source' or 'destination'.
	//
	// in: query
	// required: false
	// default: source
	Name string `json:"reporter"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics customDashboard appDashboard serviceDashboard workloadDashboard
type StepParam struct {
	// Step between [graph] datapoints, in seconds.
	//
	// in: query
	// required: false
	// default: 15
	Name int `json:"step"`
}

// swagger:parameters serviceMetrics appMetrics workloadMetrics
type VersionParam struct {
	// Filters metrics by the specified version.
	//
	// in: query
	// required: false
	Name string `json:"version"`
}

// swagger:parameters patchThreeScaleHandler deleteThreeScaleHandler
type ThreScaleHandlerNameParam struct {
	// The ThreeScaleHandler name.
	//
	// in: path
	// required: true
	Name string `json:"threescaleHandlerName"`
}

// swagger:parameters root getStatus istioStatus getConfig getPermissions istioConfigList istioConfigDetails istioConfigDelete istioConfigUpdate istioConfigCreate serviceList serviceDetails serviceMetrics serviceHealth spansList tracesList errorTraces tracesDetail serviceDashboard workloadList workloadDetails workloadMetrics workloadDashboard workloadHealth appList appDetails appMetrics appDashboard appHealth namespaceList namespaceHealth namespaceMetrics namespaceValidations namespaceTls podDetails podLogs graphNamespaces graphAppVersion graphApp graphService graphWorkload getThreeScaleInfo getThreeScaleHandlers postThreeScaleHandlers patchThreeScaleHandler deleteThreeScaleHandler getThreeScaleService postThreeScaleService patchThreeScaleService deleteThreeScaleService getIter8 getIter8Experiments iter8Experiments postIter8Experiments patchIter8Experiments deleteIter8Experiments getIter8Metrics grafanaInfo jaegerInfo customDashboard meshTls namespaceOverview serviceOverview workloadOverview appOverview
type ServiceDomainParam struct {
	// ID of ServiceDomain to access.
	//
	// in: query
	// required: true
	Name string `json:"serviceDomain"`
}

// swagger:parameters root getStatus istioStatus getConfig getPermissions istioConfigList istioConfigDetails istioConfigDelete istioConfigUpdate istioConfigCreate serviceList serviceDetails serviceMetrics serviceHealth spansList tracesList errorTraces tracesDetail serviceDashboard workloadList workloadDetails workloadMetrics workloadDashboard workloadHealth appList appDetails appMetrics appDashboard appHealth namespaceList namespaceHealth namespaceMetrics namespaceValidations namespaceTls podDetails podLogs graphNamespaces graphAppVersion graphApp graphService graphWorkload getThreeScaleInfo getThreeScaleHandlers postThreeScaleHandlers patchThreeScaleHandler deleteThreeScaleHandler getThreeScaleService postThreeScaleService patchThreeScaleService deleteThreeScaleService getIter8 getIter8Experiments iter8Experiments postIter8Experiments patchIter8Experiments deleteIter8Experiments getIter8Metrics grafanaInfo jaegerInfo customDashboard meshTls namespaceOverview serviceOverview workloadOverview appOverview
// in: header
type KialiProxyAuthorizationParam struct {
	// Format: Bearer &lt;token>, with &lt;token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

/////////////////////
// SWAGGER RESPONSES
/////////////////////

// HTTP status code 200 and statusInfo model in data
// swagger:response statusInfo
type swaggStatusInfoResp struct {
	// in:body
	Body StatusInfo
}

// HTTP status code 200 and cytoscapejs Config in data
// swagger:response graphResponse
type GraphResponse struct {
	// in:body
	Body GraphConfig
}

// Return a list of Istio components along its status
// swagger:response istioStatusResponse
type IstioStatusResponse struct {
	// in: body
	Body IstioComponentStatus
}

// Return caller permissions per namespace and Istio Config type
// swagger:response istioConfigPermissions
type swaggIstioConfigPermissions struct {
	// in:body
	Body IstioConfigPermissions
}

// HTTP status code 200 and IstioConfigList model in data
// swagger:response istioConfigList
type IstioConfigResponse struct {
	// in:body
	Body IstioConfigList
}

// IstioConfig details of an specific Istio Object
// swagger:response istioConfigDetailsResponse
type IstioConfigDetailsResponse struct {
	// in:body
	Body IstioConfigDetails
}

// Listing all services in the namespace
// swagger:response serviceListResponse
type ServiceListResponse struct {
	// in:body
	Body ServiceList
}

// Metrics response model
// swagger:response kialimetricsResponse
type KialiMetricsResponse struct {
	// in:body
	Body KialiMetrics
}

// serviceHealthResponse contains aggregated health from various sources, for a given service
// swagger:response serviceHealthResponse
type serviceHealthResponse struct {
	// in:body
	Body ServiceHealth
}

// Listing all the information related to a workload
// swagger:response serviceDetailsResponse
type ServiceDetailsResponse struct {
	// in:body
	Body ServiceDetails
}

// Listing all the information related to a Span
// swagger:response spansResponse
type SpansResponse struct {
	// in:body
	Body []Span
}

// Listing all the information related to a Trace
// swagger:response tracesDetailResponse
type TracesDetailResponse struct {
	// in:body
	Body []jaegerModels.Trace
}

// Number of traces in error
// swagger:response errorTracesResponse
type ErrorTracesResponse struct {
	// in:body
	Body int
}

// Dashboard response model
// swagger:response dashboardResponse
type DashboardResponse struct {
	// in:body
	Body model.MonitoringDashboard
}

// Listing all workloads in the namespace
// swagger:response workloadListResponse
type WorkloadListResponse struct {
	// in:body
	Body WorkloadList
}

// Listing all the information related to a workload
// swagger:response workloadDetails
type WorkloadDetailsResponse struct {
	// in:body
	Body Workload
}

// workloadHealthResponse contains aggregated health from various sources, for a given workload
// swagger:response workloadHealthResponse
type workloadHealthResponse struct {
	// in:body
	Body WorkloadHealth
}

// Listing all apps in the namespace
// swagger:response appListResponse
type AppListResponse struct {
	// in:body
	Body AppList
}

// Detailed information of an specific app
// swagger:response appDetails
type AppDetailsResponse struct {
	// in:body
	Body App
}

// List of Namespaces
// swagger:response namespaceList
type NamespaceListResponse struct {
	// in:body
	Body []Namespace
}

// Return the validation status of a specific Namespace
// swagger:response namespaceValidationSummaryResponse
type NamespaceValidationSummaryResponse struct {
	// in:body
	Body IstioValidationSummary
}

// Return the mTLS status of a specific Namespace
// swagger:response namespaceTlsResponse
type NamespaceTlsResponse struct {
	// in:body
	Body MTLSStatus
}

// Return the mTLS status of the whole Mesh
// swagger:response meshTlsResponse
type MeshTlsResponse struct {
	// in:body
	Body MTLSStatus
}

// namespaceAppHealthResponse is a map of app name x health
// swagger:response namespaceAppHealthResponse
type namespaceAppHealthResponse struct {
	// in:body
	Body NamespaceAppHealth
}

// Return if ThreeScale adapter is enabled in Istio and if user has permissions to write adapter's configuration
// swagger:response threeScaleInfoResponse
type ThreeScaleInfoResponse struct {
	// in: body
	Body ThreeScaleInfo
}

// appHealthResponse contains aggregated health from various sources, for a given app
// swagger:response appHealthResponse
type appHealthResponse struct {
	// in:body
	Body AppHealth
}

// Return Threescale rule definition for a given service
// swagger:response threeScaleRuleResponse
type ThreeScaleGetRuleResponse struct {
	// in: body
	Body ThreeScaleServiceRule
}

// Return a Iter8 Experiment detail
// swagger:response iter8ExperimentGetDetailResponse
type Iter8ExperimentsGetDetailResponse struct {
	// in: body
	Body Iter8ExperimentDetail
}

// Return all the descriptor data related to Jaeger
// swagger:response jaegerInfoResponse
type JaegerInfoResponse struct {
	// in: body
	Body JaegerInfo
}

// Return all the descriptor data related to Grafana
// swagger:response grafanaInfoResponse
type GrafanaInfoResponse struct {
	// in: body
	Body GrafanaInfo
}

// Return Iter8 Info
// swagger:response iter8StatusResponse
type Iter8StatusResponse struct {
	// in: body
	Body Iter8Info
}

// Return a list of Iter8 Experiment Items
// swagger:response iter8ExperimentsResponse
type Iter8ExperimentsResponnse struct {
	// in: body
	Body []Iter8ExperimentItem
}

// List of ThreeScale handlers created from Kiali to be used in the adapter's configuration
// swagger:response threeScaleHandlersResponse
type ThreeScaleGetHandlersResponse struct {
	// in: body
	Body ThreeScaleHandlers
}

// Return Namespace combined Info
// swagger:response namespaceOverviewResponse
type NamespaceOverviewResponse struct {
	// in: body
	Body NamespaceOverview
}

// Return Service combined Info
// swagger:response serviceOverviewResponse
type ServiceOverviewResponse struct {
	// in: body
	Body ServicesOverview
}

// Return Workload combined Info
// swagger:response workloadOverviewResponse
type WorkloadOverviewResponse struct {
	// in: body
	Body WorkloadOverview
}

// Return App combined Info
// swagger:response appOverviewResponse
type AppOverviewResponse struct {
	// in: body
	Body AppOverview
}

/////////////////////
// SWAGGER MODELS
/////////////////////

// StatusInfo statusInfo
//
// This is used for returning a response of Kiali Status
//
// swagger:model StatusInfo
type StatusInfo struct {
	// The state of Kiali
	// A hash of key,values with versions of Kiali and state
	//
	// required: true
	Status map[string]string `json:"status"`
	// An array of external services installed
	//
	// required: true
	// swagger:allOf
	ExternalServices []ExternalServiceInfo `json:"externalServices"`
	// An array of warningMessages
	// items.example: Istio version 0.7.1 is not supported, the version should be 0.8.0
	// swagger:allOf
	WarningMessages []string `json:"warningMessages"`
}

// Status response model
//
// This is used for returning a response of Kiali Status
//
// swagger:model externalServiceInfo
type ExternalServiceInfo struct {
	// The name of the service
	//
	// required: true
	// example: Istio
	Name string `json:"name"`

	// The installed version of the service
	//
	// required: false
	// example: 0.8.0
	Version string `json:"version,omitempty"`

	// The service url
	//
	// required: false
	// example: jaeger-query-istio-system.127.0.0.1.nip.io
	Url string `json:"url,omitempty"`
}

// SvcService deals with fetching istio/kubernetes services related content and convert to kiali model
type IstioComponentStatus []ComponentStatus

type ComponentStatus struct {
	// The app label value of the Istio component
	//
	// example: istio-ingressgateway
	// required: true
	Name string `json:"name"`

	// The status of a Istio component
	//
	// example:  Not Found
	// required: true
	Status string `json:"status"`

	// When true, the component is necessary for Istio to function. Otherwise, it is an addon
	//
	// example:  true
	// required: true
	IsCore bool `json:"isCore"`
}

type GraphConfig struct {
	Timestamp int64    `json:"timestamp"`
	Duration  int64    `json:"duration"`
	GraphType string   `json:"graphType"`
	Elements  Elements `json:"elements"`
}

type Elements struct {
	Nodes []*NodeWrapper `json:"nodes"`
	Edges []*EdgeWrapper `json:"edges"`
}

type NodeWrapper struct {
	Data *NodeData `json:"data"`
}

type EdgeWrapper struct {
	Data *EdgeData `json:"data"`
}

type EdgeData struct {
	// Cytoscape Fields
	Id     string `json:"id"`     // unique internal edge ID (e0, e1...)
	Source string `json:"source"` // parent node ID
	Target string `json:"target"` // child node ID

	// App Fields (not required by Cytoscape)
	Traffic      ProtocolTraffic `json:"traffic,omitempty"`      // traffic rates for the edge protocol
	ResponseTime string          `json:"responseTime,omitempty"` // in millis
	IsMTLS       string          `json:"isMTLS,omitempty"`       // set to the percentage of traffic using a mutual TLS connection
}

// ProtocolTraffic supplies all of the traffic information for a single protocol
type ProtocolTraffic struct {
	Protocol  string            `json:"protocol,omitempty"`  // protocol
	Rates     map[string]string `json:"rates,omitempty"`     // map[rate]value
	Responses Responses         `json:"responses,omitempty"` // see comment above
}

type NodeData struct {
	// Cytoscape Fields
	Id     string `json:"id"`               // unique internal node ID (n0, n1...)
	Parent string `json:"parent,omitempty"` // Compound Node parent ID

	// App Fields (not required by Cytoscape)
	NodeType        string            `json:"nodeType"`
	Namespace       string            `json:"namespace"`
	Workload        string            `json:"workload,omitempty"`
	App             string            `json:"app,omitempty"`
	Version         string            `json:"version,omitempty"`
	Service         string            `json:"service,omitempty"`         // requested service for NodeTypeService
	DestServices    []ServiceName     `json:"destServices,omitempty"`    // requested services for [dest] node
	Traffic         []ProtocolTraffic `json:"traffic,omitempty"`         // traffic rates for all detected protocols
	HasCB           bool              `json:"hasCB,omitempty"`           // true (has circuit breaker) | false
	HasMissingSC    bool              `json:"hasMissingSC,omitempty"`    // true (has missing sidecar) | false
	HasVS           bool              `json:"hasVS,omitempty"`           // true (has route rule) | false
	IsDead          bool              `json:"isDead,omitempty"`          // true (has no pods) | false
	IsGroup         string            `json:"isGroup,omitempty"`         // set to the grouping type, current values: [ 'app', 'version' ]
	IsInaccessible  bool              `json:"isInaccessible,omitempty"`  // true if the node exists in an inaccessible namespace
	IsMisconfigured string            `json:"isMisconfigured,omitempty"` // set to misconfiguration list, current values: [ 'labels' ]
	IsOutside       bool              `json:"isOutside,omitempty"`       // true | false
	IsRoot          bool              `json:"isRoot,omitempty"`          // true | false
	IsServiceEntry  string            `json:"isServiceEntry,omitempty"`  // set to the location, current values: [ 'MESH_EXTERNAL', 'MESH_INTERNAL' ]
	IsUnused        bool              `json:"isUnused,omitempty"`        // true | false
}

type ServiceName struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// ResponseFlags is a map of maps. Each response code is broken down by responseFlags:percentageOfTraffic, e.g.:
// "200" : {
//    "-"     : "80.0",
//    "DC"    : "10.0",
//    "FI,FD" : "10.0"
// }, ...
type ResponseFlags map[string]string

// ResponseHosts is a map of maps. Each response host is broken down by responseFlags:percentageOfTraffic, e.g.:
// "200" : {
//    "www.google.com" : "80.0",
//    "www.yahoo.com"  : "20.0"
// }, ...
type ResponseHosts map[string]string

// ResponseDetail holds information broken down by response code.
type ResponseDetail struct {
	Flags ResponseFlags `json:"flags,omitempty"`
	Hosts ResponseHosts `json:"hosts,omitempty"`
}

// Responses maps responseCodes to detailed information for that code
type Responses map[string]*ResponseDetail

// IstioConfigList istioConfigList
//
// This type is used for returning a response of IstioConfigList
//
// swagger:model IstioConfigList
type IstioConfigList struct {
	// The namespace of istioConfiglist
	//
	// required: true
	Namespace              Namespace              `json:"namespace"`
	Gateways               Gateways               `json:"gateways"`
	VirtualServices        VirtualServices        `json:"virtualServices"`
	DestinationRules       DestinationRules       `json:"destinationRules"`
	ServiceEntries         ServiceEntries         `json:"serviceEntries"`
	WorkloadEntries        WorkloadEntries        `json:"workloadEntries"`
	EnvoyFilters           EnvoyFilters           `json:"envoyFilters"`
	Rules                  IstioRules             `json:"rules"`
	Adapters               IstioAdapters          `json:"adapters"`
	Templates              IstioTemplates         `json:"templates"`
	Handlers               IstioHandlers          `json:"handlers"`
	Instances              IstioInstances         `json:"instances"`
	QuotaSpecs             QuotaSpecs             `json:"quotaSpecs"`
	QuotaSpecBindings      QuotaSpecBindings      `json:"quotaSpecBindings"`
	AttributeManifests     AttributeManifests     `json:"attributeManifests"`
	HttpApiSpecs           HttpApiSpecs           `json:"httpApiSpecs"`
	HttpApiSpecBindings    HttpApiSpecBindings    `json:"httpApiSpecBindings"`
	Policies               Policies               `json:"policies"`
	MeshPolicies           MeshPolicies           `json:"meshPolicies"`
	ServiceMeshPolicies    ServiceMeshPolicies    `json:"serviceMeshPolicies"`
	ClusterRbacConfigs     ClusterRbacConfigs     `json:"clusterRbacConfigs"`
	RbacConfigs            RbacConfigs            `json:"rbacConfigs"`
	ServiceMeshRbacConfigs ServiceMeshRbacConfigs `json:"serviceMeshRbacConfigs"`
	ServiceRoles           ServiceRoles           `json:"serviceRoles"`
	ServiceRoleBindings    ServiceRoleBindings    `json:"serviceRoleBindings"`
	Sidecars               Sidecars               `json:"sidecars"`
	AuthorizationPolicies  AuthorizationPolicies  `json:"authorizationPolicies"`
	PeerAuthentications    PeerAuthentications    `json:"peerAuthentications"`
	RequestAuthentications RequestAuthentications `json:"requestAuthentications"`
	IstioValidations       IstioValidations       `json:"istioValidations"`
}

type IstioConfigDetails struct {
	Namespace             Namespace              `json:"namespace"`
	ObjectType            string                 `json:"objectType"`
	Gateway               *Gateway               `json:"gateway"`
	VirtualService        *VirtualService        `json:"virtualService"`
	DestinationRule       *DestinationRule       `json:"destinationRule"`
	ServiceEntry          *ServiceEntry          `json:"serviceEntry"`
	WorkloadEntry         *WorkloadEntry         `json:"workloadEntry"`
	EnvoyFilter           *EnvoyFilter           `json:"envoyFilter"`
	Rule                  *IstioRule             `json:"rule"`
	Adapter               *IstioAdapter          `json:"adapter"`
	Template              *IstioTemplate         `json:"template"`
	Handler               *IstioHandler          `json:"handler"`
	Instance              *IstioInstance         `json:"instance"`
	QuotaSpec             *QuotaSpec             `json:"quotaSpec"`
	QuotaSpecBinding      *QuotaSpecBinding      `json:"quotaSpecBinding"`
	AttributeManifest     *AttributeManifest     `json:"attributeManifest"`
	HttpApiSpec           *HttpApiSpec           `json:"httpApiSpec"`
	HttpApiSpecBinding    *HttpApiSpecBinding    `json:"httpApiSpecBinding"`
	Policy                *Policy                `json:"policy"`
	MeshPolicy            *MeshPolicy            `json:"meshPolicy"`
	ServiceMeshPolicy     *ServiceMeshPolicy     `json:"serviceMeshPolicy"`
	ClusterRbacConfig     *ClusterRbacConfig     `json:"clusterRbacConfig"`
	RbacConfig            *RbacConfig            `json:"rbacConfig"`
	ServiceMeshRbacConfig *ServiceMeshRbacConfig `json:"serviceMeshRbacConfig"`
	ServiceRole           *ServiceRole           `json:"serviceRole"`
	ServiceRoleBinding    *ServiceRoleBinding    `json:"serviceRoleBinding"`
	Sidecar               *Sidecar               `json:"sidecar"`
	AuthorizationPolicy   *AuthorizationPolicy   `json:"authorizationPolicy"`
	PeerAuthentication    *PeerAuthentication    `json:"peerAuthentication"`
	RequestAuthentication *RequestAuthentication `json:"requestAuthentication"`
	Permissions           ResourcePermissions    `json:"permissions"`
	IstioValidation       *IstioValidation       `json:"istioValidation"`
}

// ResourcePermissions holds permission flags for an object type
// True means allowed.
type ResourcePermissions struct {
	Create bool `json:"create"`
	Update bool `json:"update"`
	Delete bool `json:"delete"`
}

// ResourcesPermissions holds a map of permission flags per resource
type ResourcesPermissions map[string]*ResourcePermissions

// IstioConfigPermissions holds a map of ResourcesPermissions per namespace
type IstioConfigPermissions map[string]*ResourcesPermissions

// A Namespace provide a scope for names
// This type is used to describe a set of objects.
//
// swagger:model namespace
type Namespace struct {
	// The id of the namespace.
	//
	// example:  istio-system
	// required: true
	Name string `json:"name"`

	// Creation date of the namespace.
	// There is no need to export this through the API. So, this is
	// set to be ignored by JSON package.
	//
	// required: true
	CreationTimestamp time.Time `json:"-"`

	// Labels for Namespace
	Labels map[string]string `json:"labels"`
}

type Namespaces []Namespace
type NamespaceNames []string

type Gateways []Gateway
type Gateway struct {
	meta_v1.TypeMeta
	Metadata    meta_v1.ObjectMeta `json:"metadata"`
	GatewaySpec struct {
		Servers  interface{} `json:"servers"`
		Selector interface{} `json:"selector"`
	} `json:"spec"`
}

// VirtualServices virtualServices
//
// This type is used for returning an array of VirtualServices with some permission flags
//
// swagger:model virtualServices
// An array of virtualService
// swagger:allOf
type VirtualServices struct {
	Permissions ResourcePermissions `json:"permissions"`
	Items       []VirtualService    `json:"items"`
}

// VirtualService virtualService
//
// This type is used for returning a VirtualService
//
// swagger:model virtualService
type VirtualService struct {
	meta_v1.TypeMeta
	Metadata           meta_v1.ObjectMeta `json:"metadata"`
	VirtualServiceSpec struct {
		Hosts    interface{} `json:"hosts,omitempty"`
		Gateways interface{} `json:"gateways,omitempty"`
		Http     interface{} `json:"http,omitempty"`
		Tcp      interface{} `json:"tcp,omitempty"`
		Tls      interface{} `json:"tls,omitempty"`
		ExportTo interface{} `json:"exportTo,omitempty"`
	} `json:"spec"`
}

// DestinationRules destinationRules
//
// This is used for returning an array of DestinationRules
//
// swagger:model destinationRules
// An array of destinationRule
// swagger:allOf
type DestinationRules struct {
	Permissions ResourcePermissions `json:"permissions"`
	Items       []DestinationRule   `json:"items"`
}

// DestinationRule destinationRule
//
// This is used for returning a DestinationRule
//
// swagger:model destinationRule
type DestinationRule struct {
	meta_v1.TypeMeta
	Metadata            meta_v1.ObjectMeta `json:"metadata"`
	DestinationRuleSpec struct {
		Host          interface{} `json:"host,omitempty"`
		TrafficPolicy interface{} `json:"trafficPolicy,omitempty"`
		Subsets       interface{} `json:"subsets,omitempty"`
		ExportTo      interface{} `json:"exportTo,omitempty"`
	} `json:"spec"`
}

type ServiceEntries []ServiceEntry
type ServiceEntry struct {
	meta_v1.TypeMeta
	Metadata         meta_v1.ObjectMeta `json:"metadata"`
	ServiceEntrySpec struct {
		Hosts            interface{} `json:"hosts"`
		Addresses        interface{} `json:"addresses"`
		Ports            interface{} `json:"ports"`
		Location         interface{} `json:"location"`
		Resolution       interface{} `json:"resolution"`
		Endpoints        interface{} `json:"endpoints"`
		WorkloadSelector interface{} `json:"workloadSelector"`
		ExportTo         interface{} `json:"exportTo"`
		SubjectAltNames  interface{} `json:"subjectAltNames"`
	} `json:"spec"`
}

// WorkloadEntries workloadEntries
//
// This is used for returning an array of WorkloadEntry
//
// swagger:model workloadEntries
// An array of workloadEntry
// swagger:allOf
type WorkloadEntries []WorkloadEntry

// WorkloadEntry workloadEntry
//
// This is used for returning an WorkloadEntry
//
// swagger:model workloadEntry
type WorkloadEntry struct {
	meta_v1.TypeMeta
	Metadata          meta_v1.ObjectMeta `json:"metadata"`
	WorkloadEntrySpec struct {
		Address        interface{} `json:"address"`
		Ports          interface{} `json:"ports"`
		Labels         interface{} `json:"labels"`
		Network        interface{} `json:"network"`
		Locality       interface{} `json:"locality"`
		Weight         interface{} `json:"weight"`
		ServiceAccount interface{} `json:"serviceAccount"`
	} `json:"spec"`
}

// EnvoyFilters envoyFilters
//
// This is used for returning an array of EnvoyFilter
//
// swagger:model envoyFilters
// An array of envoyFilter
// swagger:allOf
type EnvoyFilters []EnvoyFilter

// EnvoyFilter envoyFilter
//
// This is used for returning an EnvoyFilter
//
// swagger:model envoyFilter
type EnvoyFilter struct {
	meta_v1.TypeMeta
	Metadata        meta_v1.ObjectMeta `json:"metadata"`
	EnvoyFilterSpec struct {
		WorkloadSelector interface{} `json:"workloadSelector"`
		ConfigPatches    interface{} `json:"configPatches"`
	} `json:"spec"`
}

type IstioRuleList struct {
	Namespace Namespace   `json:"namespace"`
	Rules     []IstioRule `json:"rules"`
}

// IstioRules istioRules
//
// This type type is used for returning an array of IstioRules
//
// swagger:model istioRules
// An array of istioRule
// swagger:allOf
type IstioRules []IstioRule

// IstioRule istioRule
//
// This type type is used for returning a IstioRule
//
// swagger:model istioRule
type IstioRule struct {
	meta_v1.TypeMeta
	Metadata      meta_v1.ObjectMeta `json:"metadata"`
	IstioRuleSpec struct {
		Match   interface{} `json:"match"`
		Actions interface{} `json:"actions"`
	} `json:"spec"`
}

// IstioAdapters istioAdapters
//
// This type type is used for returning an array of IstioAdapters
//
// swagger:model istioAdapters
// An array of istioAdapter
// swagger:allOf
type IstioAdapters []IstioAdapter

// IstioAdapter istioAdapter
//
// This type type is used for returning a IstioAdapter
//
// swagger:model istioAdapter
type IstioAdapter struct {
	meta_v1.TypeMeta
	Metadata         meta_v1.ObjectMeta `json:"metadata"`
	IstioAdapterSpec interface{}        `json:"spec"`
}

// IstioTemplates istioTemplates
//
// This type type is used for returning an array of IstioTemplates
//
// swagger:model istioTemplates
// An array of istioTemplate
// swagger:allOf
type IstioTemplates []IstioTemplate

// IstioTemplate istioTemplate
//
// This type type is used for returning a IstioTemplate
//
// swagger:model istioTemplate
type IstioTemplate struct {
	meta_v1.TypeMeta
	Metadata          meta_v1.ObjectMeta `json:"metadata"`
	IstioTemplateSpec interface{}        `json:"spec"`
}

// IstioHandlers istioHandlers
//
// This type type is used for returning an array of IstioHandlers
//
// swagger:model istioHandlers
// An array of istioHandler
// swagger:allOf
type IstioHandlers []IstioHandler

// IstioHandler istioHandler
//
// This type type is used for returning a IstioHandler
//
// swagger:model istioHandler
type IstioHandler struct {
	meta_v1.TypeMeta
	Metadata         meta_v1.ObjectMeta `json:"metadata"`
	IstioHandlerSpec interface{}        `json:"spec"`
}

// IstioInstances istioInstances
//
// This type type is used for returning an array of IstioInstances
//
// swagger:model istioInstances
// An array of istioIstance
// swagger:allOf
type IstioInstances []IstioInstance

// IstioInstance istioInstance
//
// This type type is used for returning a IstioInstance
//
// swagger:model istioInstance
type IstioInstance struct {
	meta_v1.TypeMeta
	Metadata          meta_v1.ObjectMeta `json:"metadata"`
	IstioInstanceSpec interface{}        `json:"spec"`
}

type QuotaSpecs []QuotaSpec
type QuotaSpec struct {
	meta_v1.TypeMeta
	Metadata         meta_v1.ObjectMeta `json:"metadata"`
	QuotaSpecSubSpec struct {
		Rules interface{} `json:"rules"`
	} `json:"spec"`
}

type QuotaSpecBindings []QuotaSpecBinding
type QuotaSpecBinding struct {
	meta_v1.TypeMeta
	Metadata             meta_v1.ObjectMeta `json:"metadata"`
	QuotaSpecBindingSpec struct {
		QuotaSpecs interface{} `json:"quotaSpecs"`
		Services   interface{} `json:"services"`
	} `json:"spec"`
}

// AttributeManifests attributeManifests
//
// This is used for returning an array of AttributeManifest
//
// swagger:model attributeManifests
// An array of attributeManifest
// swagger:allOf
type AttributeManifests []AttributeManifest

// AttributeManifest attributeManifest
//
// This is used for returning an AttributeManifest
//
// swagger:model attributeManifest
type AttributeManifest struct {
	meta_v1.TypeMeta
	Metadata              meta_v1.ObjectMeta `json:"metadata"`
	AttributeManifestSpec struct {
		Revision   interface{} `json:"revision"`
		Name       interface{} `json:"name"`
		Attributes interface{} `json:"attributes"`
	} `json:"spec"`
}

// HttpApiSpecs httpApiSpecs
//
// This is used for returning an array of HttpApiSpec
//
// swagger:model httpApiSpecs
// An array of httpApiSpec
// swagger:allOf
type HttpApiSpecs []HttpApiSpec

// HttpApiSpec httpApiSpec
//
// This is used for returning an HttpApiSpec
//
// swagger:model httpApiSpec
type HttpApiSpec struct {
	meta_v1.TypeMeta
	Metadata       meta_v1.ObjectMeta `json:"metadata"`
	HttpApiSubSpec struct {
		Attributes interface{} `json:"attributes"`
		Patterns   interface{} `json:"patterns"`
		ApiKeys    interface{} `json:"apiKeys"`
	} `json:"spec"`
}

// HttpApiSpecBindings httpApiSpecBindings
//
// This is used for returning an array of HttpApiSpecBinding
//
// swagger:model httpApiSpecBindings
// An array of httpApiSpecBinding
// swagger:allOf
type HttpApiSpecBindings []HttpApiSpecBinding

// HttpApiSpecBinding httpApiSpecBinding
//
// This is used for returning an HttpApiSpecBinding
//
// swagger:model httpApiSpecBinding
type HttpApiSpecBinding struct {
	meta_v1.TypeMeta
	Metadata               meta_v1.ObjectMeta `json:"metadata"`
	HttpApiSpecBindingSpec struct {
		Services interface{} `json:"services"`
		ApiSpecs interface{} `json:"apiSpecs"`
	} `json:"spec"`
}

type Policies []Policy
type Policy struct {
	meta_v1.TypeMeta
	Metadata   meta_v1.ObjectMeta `json:"metadata"`
	PolicySpec struct {
		Targets          interface{} `json:"targets"`
		Peers            interface{} `json:"peers"`
		PeerIsOptional   interface{} `json:"peerIsOptional"`
		Origins          interface{} `json:"origins"`
		OriginIsOptional interface{} `json:"originIsOptional"`
		PrincipalBinding interface{} `json:"principalBinding"`
	} `json:"spec"`
}

type MeshPolicySpec struct {
	Targets          interface{} `json:"targets"`
	Peers            interface{} `json:"peers"`
	PeerIsOptional   interface{} `json:"peerIsOptional"`
	Origins          interface{} `json:"origins"`
	OriginIsOptional interface{} `json:"originIsOptional"`
	PrincipalBinding interface{} `json:"principalBinding"`
}

type MeshPolicies []MeshPolicy
type MeshPolicy struct {
	meta_v1.TypeMeta
	Metadata       meta_v1.ObjectMeta `json:"metadata"`
	MeshPolicySpec MeshPolicySpec     `json:"spec"`
}

// ServiceMeshPolicy is a clone of MeshPolicy used by Maistra for multitenancy scenarios
// Used in the same file for easy maintenance

type ServiceMeshPolicies []ServiceMeshPolicy
type ServiceMeshPolicy struct {
	meta_v1.TypeMeta
	Metadata              meta_v1.ObjectMeta `json:"metadata"`
	ServiceMeshPolicySpec MeshPolicySpec     `json:"spec"`
}

type ClusterRbacConfigSpec struct {
	Mode      interface{} `json:"mode"`
	Inclusion interface{} `json:"inclusion"`
	Exclusion interface{} `json:"exclusion"`
}

type ClusterRbacConfigs []ClusterRbacConfig
type ClusterRbacConfig struct {
	meta_v1.TypeMeta
	Metadata              meta_v1.ObjectMeta    `json:"metadata"`
	ClusterRbacConfigSpec ClusterRbacConfigSpec `json:"spec"`
}

type RbacConfigs []RbacConfig
type RbacConfig struct {
	meta_v1.TypeMeta
	Metadata       meta_v1.ObjectMeta `json:"metadata"`
	RbacConfigSpec struct {
		Mode      interface{} `json:"mode"`
		Inclusion interface{} `json:"inclusion"`
		Exclusion interface{} `json:"exclusion"`
	} `json:"spec"`
}

// ServiceMeshRbacConfig is a clone of ClusterRbacPolicy used by Maistra for multitenancy scenarios
// Used in the same file for easy maintenance
type ServiceMeshRbacConfigs []ServiceMeshRbacConfig
type ServiceMeshRbacConfig struct {
	meta_v1.TypeMeta
	Metadata                  meta_v1.ObjectMeta    `json:"metadata"`
	ServiceMeshRbacConfigSpec ClusterRbacConfigSpec `json:"spec"`
}

type ServiceRoles []ServiceRole
type ServiceRole struct {
	meta_v1.TypeMeta
	Metadata        meta_v1.ObjectMeta `json:"metadata"`
	ServiceRoleSpec struct {
		Rules interface{} `json:"rules"`
	} `json:"spec"`
}

type ServiceRoleBindings []ServiceRoleBinding
type ServiceRoleBinding struct {
	meta_v1.TypeMeta
	Metadata               meta_v1.ObjectMeta `json:"metadata"`
	ServiceRoleBindingSpec struct {
		Subjects interface{} `json:"subjects"`
		RoleRef  interface{} `json:"roleRef"`
	} `json:"spec"`
}

type Sidecars []Sidecar
type Sidecar struct {
	meta_v1.TypeMeta
	Metadata    meta_v1.ObjectMeta `json:"metadata"`
	SidecarSpec struct {
		WorkloadSelector      interface{} `json:"workloadSelector"`
		Ingress               interface{} `json:"ingress"`
		Egress                interface{} `json:"egress"`
		OutboundTrafficPolicy interface{} `json:"outboundTrafficPolicy"`
		Localhost             interface{} `json:"localhost"`
	} `json:"spec"`
}

// AuthorizationPolicies authorizationPolicies
//
// This is used for returning an array of AuthorizationPolicies
//
// swagger:model authorizationPolicies
// An array of authorizationPolicy
// swagger:allOf
type AuthorizationPolicies []AuthorizationPolicy

// AuthorizationPolicy authorizationPolicy
//
// This is used for returning an AuthorizationPolicy
//
// swagger:model authorizationPolicy
type AuthorizationPolicy struct {
	meta_v1.TypeMeta
	Metadata                meta_v1.ObjectMeta `json:"metadata"`
	AuthorizationPolicySpec struct {
		Selector interface{} `json:"selector"`
		Rules    interface{} `json:"rules"`
		Action   interface{} `json:"action"`
	} `json:"spec"`
}

// PeerAuthentications peerAuthentications
//
// This is used for returning an array of PeerAuthentication
//
// swagger:model peerAuthentications
// An array of peerAuthentication
// swagger:allOf
type PeerAuthentications []PeerAuthentication

// PeerAuthentication peerAuthentication
//
// This is used for returning an PeerAuthentication
//
// swagger:model peerAuthentication
type PeerAuthentication struct {
	meta_v1.TypeMeta
	Metadata               meta_v1.ObjectMeta `json:"metadata"`
	PeerAuthenticationSpec struct {
		Selector      interface{} `json:"selector"`
		Mtls          interface{} `json:"mtls"`
		PortLevelMtls interface{} `json:"portLevelMtls"`
	} `json:"spec"`
}

// RequestAuthentications requestAuthentications
//
// This is used for returning an array of RequestAuthentication
//
// swagger:model requestAuthentications
// An array of requestAuthentication
// swagger:allOf
type RequestAuthentications []RequestAuthentication

// RequestAuthentication requestAuthentication
//
// This is used for returning an RequestAuthentication
//
// swagger:model requestAuthentication
type RequestAuthentication struct {
	meta_v1.TypeMeta
	Metadata                  meta_v1.ObjectMeta `json:"metadata"`
	RequestAuthenticationSpec struct {
		Selector interface{} `json:"selector"`
		JwtRules interface{} `json:"jwtRules"`
	} `json:"spec"`
}

// NamespaceValidations represents a set of IstioValidations grouped by namespace
type NamespaceValidations map[string]IstioValidations

// IstioValidationKey is the key value composed of an Istio ObjectType and Name.
type IstioValidationKey struct {
	ObjectType string `json:"objectType"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
}

// IstioValidationSummary represents the number of errors/warnings of a set of Istio Validations.
type IstioValidationSummary struct {
	// Number of validations with error severity
	// required: true
	// example: 2
	Errors int `json:"errors"`
	// Number of Istio Objects analyzed
	// required: true
	// example: 6
	ObjectCount int `json:"objectCount"`
	// Number of validations with warning severity
	// required: true
	// example: 4
	Warnings int `json:"warnings"`
}

// IstioValidations represents a set of IstioValidation grouped by IstioValidationKey.
type IstioValidations map[IstioValidationKey]*IstioValidation

// IstioValidation represents a list of checks associated to an Istio object.
// swagger:model
type IstioValidation struct {
	// Name of the object itself
	// required: true
	// example: reviews
	Name string `json:"name"`

	// Type of the object
	// required: true
	// example: virtualservice
	ObjectType string `json:"objectType"`

	// Represents validity of the object: in case of warning, validity remains as true
	// required: true
	// example: false
	Valid bool `json:"valid"`

	// Array of checks. It might be empty.
	Checks []*IstioCheck `json:"checks"`

	// Related objects (only validation errors)
	References []IstioValidationKey `json:"references"`
}

// IstioCheck represents an individual check.
// swagger:model
type IstioCheck struct {
	// Description of the check
	// required: true
	// example: Weight sum should be 100
	Message string `json:"message"`

	// Indicates the level of importance: error or warning
	// required: true
	// example: error
	Severity SeverityLevel `json:"severity"`

	// String that describes where in the yaml file is the check located
	// example: spec/http[0]/route
	Path string `json:"path"`
}

type SeverityLevel string

type ServiceOverview struct {
	// Name of the Service
	// required: true
	// example: reviews-v1
	Name string `json:"name"`
	// Define if Pods related to this Service has an IstioSidecar deployed
	// required: true
	// example: true
	IstioSidecar bool `json:"istioSidecar"`
	// Has label app
	// required: true
	// example: true
	AppLabel bool `json:"appLabel"`
	// Additional detail sample, such as type of api being served (graphql, grpc, rest)
	// example: rest
	// required: false
	AdditionalDetailSample *AdditionalItem `json:"additionalDetailSample"`

	// Labels for Service
	Labels map[string]string `json:"labels"`
}

type ServiceList struct {
	Namespace   Namespace         `json:"namespace"`
	Services    []ServiceOverview `json:"services"`
	Validations IstioValidations  `json:"validations"`
}
type ServiceDefinitionList struct {
	Namespace          Namespace        `json:"namespace"`
	ServiceDefinitions []ServiceDetails `json:"serviceDefinitions"`
}

type ServiceDetails struct {
	Service           Service           `json:"service"`
	IstioSidecar      bool              `json:"istioSidecar"`
	Endpoints         Endpoints         `json:"endpoints"`
	VirtualServices   VirtualServices   `json:"virtualServices"`
	DestinationRules  DestinationRules  `json:"destinationRules"`
	Workloads         WorkloadOverviews `json:"workloads"`
	Health            ServiceHealth     `json:"health"`
	Validations       IstioValidations  `json:"validations"`
	NamespaceMTLS     MTLSStatus        `json:"namespaceMTLS"`
	AdditionalDetails []AdditionalItem  `json:"additionalDetails"`
}

type Services []*Service
type Service struct {
	Name            string            `json:"name"`
	CreatedAt       string            `json:"createdAt"`
	ResourceVersion string            `json:"resourceVersion"`
	Namespace       Namespace         `json:"namespace"`
	Labels          map[string]string `json:"labels"`
	Selectors       map[string]string `json:"selectors"`
	Type            string            `json:"type"`
	Ip              string            `json:"ip"`
	Ports           Ports             `json:"ports"`
	ExternalName    string            `json:"externalName"`
}

type AdditionalItem struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Icon  string `json:"icon"`
}

type Endpoints []Endpoint
type Endpoint struct {
	Addresses Addresses `json:"addresses"`
	Ports     Ports     `json:"ports"`
}

type WorkloadList struct {
	// Namespace where the workloads live in
	// required: true
	// example: bookinfo
	Namespace Namespace `json:"namespace"`

	// Workloads for a given namespace
	// required: true
	Workloads []WorkloadListItem `json:"workloads"`
}

// WorkloadListItem has the necessary information to display the console workload list
type WorkloadListItem struct {
	// Name of the workload
	// required: true
	// example: reviews-v1
	Name string `json:"name"`

	// Type of the workload
	// required: true
	// example: deployment
	Type string `json:"type"`

	// Creation timestamp (in RFC3339 format)
	// required: true
	// example: 2018-07-31T12:24:17Z
	CreatedAt string `json:"createdAt"`

	// Kubernetes ResourceVersion
	// required: true
	// example: 192892127
	ResourceVersion string `json:"resourceVersion"`

	// Define if Pods related to this Workload has an IstioSidecar deployed
	// required: true
	// example: true
	IstioSidecar bool `json:"istioSidecar"`

	// Additional item sample, such as type of api being served (graphql, grpc, rest)
	// example: rest
	// required: false
	AdditionalDetailSample *AdditionalItem `json:"additionalDetailSample"`

	// Workload labels
	Labels map[string]string `json:"labels"`

	// Define if Pods related to this Workload has the label App
	// required: true
	// example: true
	AppLabel bool `json:"appLabel"`

	// Define if Pods related to this Workload has the label Version
	// required: true
	// example: true
	VersionLabel bool `json:"versionLabel"`

	// Number of current workload pods
	// required: true
	// example: 1
	PodCount int `json:"podCount"`
}

type WorkloadOverviews []*WorkloadListItem

// Workload has the details of a workload
type Workload struct {
	WorkloadListItem

	// Number of desired replicas defined by the user in the controller Spec
	// required: true
	// example: 2
	DesiredReplicas int32 `json:"desiredReplicas"`

	// Number of current replicas pods that matches controller selector labels
	// required: true
	// example: 2
	CurrentReplicas int32 `json:"currentReplicas"`

	// Number of available replicas
	// required: true
	// example: 1
	AvailableReplicas int32 `json:"availableReplicas"`

	// Pods bound to the workload
	Pods Pods `json:"pods"`

	// Services that match workload selector
	Services Services `json:"services"`

	// Runtimes and associated dashboards
	Runtimes []kmodel.Runtime `json:"runtimes"`

	// Additional details to display, such as configured annotations
	AdditionalDetails []AdditionalItem `json:"additionalDetails"`
}

type Workloads []*Workload

// NamespaceAppHealth is an alias of map of app name x health
type NamespaceAppHealth map[string]*AppHealth

// NamespaceServiceHealth is an alias of map of service name x health
type NamespaceServiceHealth map[string]*ServiceHealth

// NamespaceWorkloadHealth is an alias of map of workload name x health
type NamespaceWorkloadHealth map[string]*WorkloadHealth

// ServiceHealth contains aggregated health from various sources, for a given service
type ServiceHealth struct {
	Requests RequestHealth `json:"requests"`
}

// AppHealth contains aggregated health from various sources, for a given app
type AppHealth struct {
	WorkloadStatuses []WorkloadStatus `json:"workloadStatuses"`
	Requests         RequestHealth    `json:"requests"`
}

// MTLSStatus describes the current mTLS status of a mesh entity
type MTLSStatus struct {
	// mTLS status: MTLS_ENABLED, MTLS_PARTIALLY_ENABLED, MTLS_NOT_ENABLED
	// required: true
	// example: MTLS_ENABLED
	Status string `json:"status"`
}

type Ports []Port
type Port struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     int32  `json:"port"`
}

type Addresses []Address
type Address struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
	IP   string `json:"ip"`
}

// Pods alias for list of Pod structs
type Pods []*Pod

// Pod holds a subset of v1.Pod data that is meaningful in Kiali
type Pod struct {
	Name                string            `json:"name"`
	Labels              map[string]string `json:"labels"`
	CreatedAt           string            `json:"createdAt"`
	CreatedBy           []Reference       `json:"createdBy"`
	Containers          []*ContainerInfo  `json:"containers"`
	IstioContainers     []*ContainerInfo  `json:"istioContainers"`
	IstioInitContainers []*ContainerInfo  `json:"istioInitContainers"`
	Status              string            `json:"status"`
	AppLabel            bool              `json:"appLabel"`
	VersionLabel        bool              `json:"versionLabel"`
	Annotations         map[string]string `json:"annotations"`
}

// Reference holds some information on the pod creator
type Reference struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// ContainerInfo holds container name and image
type ContainerInfo struct {
	Name  string `json:"name"`
	Image string `json:"image"`
}

// WorkloadHealth contains aggregated health from various sources, for a given workload
type WorkloadHealth struct {
	WorkloadStatus WorkloadStatus `json:"workloadStatus"`
	Requests       RequestHealth  `json:"requests"`
}

// WorkloadStatus gives
// - number of desired replicas defined in the Spec of a controller
// - number of current replicas that matches selector of a controller
// - number of available replicas for a given workload
// In healthy scenarios all variables should point same value.
// When something wrong happens the different values can indicate an unhealthy situation.
// i.e.
// 	desired = 1, current = 10, available = 0 would means that a user scaled down a workload from 10 to 1
//  but in the operaton 10 pods showed problems, so no pod is available/ready but user will see 10 pods under a workload
type WorkloadStatus struct {
	Name              string `json:"name"`
	DesiredReplicas   int32  `json:"desiredReplicas"`
	CurrentReplicas   int32  `json:"currentReplicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
}

// RequestHealth holds several stats about recent request errors
type RequestHealth struct {
	inboundErrorRate    float64
	outboundErrorRate   float64
	inboundRequestRate  float64
	outboundRequestRate float64

	ErrorRatio         float64 `json:"errorRatio"`
	InboundErrorRatio  float64 `json:"inboundErrorRatio"`
	OutboundErrorRatio float64 `json:"outboundErrorRatio"`
}

// KialiMetrics contains all simple metrics and histograms data
type KialiMetrics struct {
	Metrics    map[string]*KialiMetric `json:"metrics"`
	Histograms map[string]Histogram    `json:"histograms"`
}

// KialiMetric holds the Prometheus Matrix model, which contains one or more time series (depending on grouping)
type KialiMetric struct {
	Matrix Matrix `json:"matrix"`
	Err    error  `json:"-"`
}

// Histogram contains KialiMetric objects for several histogram-kind statistics
type Histogram = map[string]*KialiMetric

// Matrix is a list of time series.
type Matrix []*SampleStream

// SampleStream is a stream of Values belonging to an attached COWMetric.
type SampleStream struct {
	Metric KialiMetric  `json:"metric"`
	Values []SamplePair `json:"values"`
}

// SamplePair pairs a SampleValue with a Timestamp.
type SamplePair struct {
	Timestamp Time
	Value     SampleValue
}

// Time is the number of milliseconds since the epoch
// (1970-01-01 00:00 UTC) excluding leap seconds.
type Time int64

// A SampleValue is a representation of a value for a given sample at a given
// time.
type SampleValue float64

type Span struct {
	jaegerModels.Span
	TraceSize int `json:"traceSize"`
}
type AppList struct {
	// Namespace where the apps live in
	// required: true
	// example: bookinfo
	Namespace Namespace `json:"namespace"`

	// Applications for a given namespace
	// required: true
	Apps []AppListItem `json:"applications"`
}

// AppListItem has the necessary information to display the console app list
type AppListItem struct {
	// Name of the application
	// required: true
	// example: reviews
	Name string `json:"name"`

	// Define if all Pods related to the Workloads of this app has an IstioSidecar deployed
	// required: true
	// example: true
	IstioSidecar bool `json:"istioSidecar"`

	// Labels for App
	Labels map[string]string `json:"labels"`
}

type App struct {
	// Namespace where the app lives in
	// required: true
	// example: bookinfo
	Namespace Namespace `json:"namespace"`

	// Name of the application
	// required: true
	// example: reviews
	Name string `json:"name"`

	// Workloads for a given application
	// required: true
	Workloads []WorkloadItem `json:"workloads"`

	// List of service names linked with an application
	// required: true
	ServiceNames []string `json:"serviceNames"`

	// Runtimes and associated dashboards
	Runtimes []kmodel.Runtime `json:"runtimes"`
}

type WorkloadItem struct {
	// Name of a workload member of an application
	// required: true
	// example: reviews-v1
	WorkloadName string `json:"workloadName"`

	// Define if all Pods related to the Workload has an IstioSidecar deployed
	// required: true
	// example: true
	IstioSidecar bool `json:"istioSidecar"`
}

// ThreeScaleInfo shows if 3scale adapter is enabled in cluster and if user has permissions on adapter's configuration
type ThreeScaleInfo struct {
	Enabled     bool                `json:"enabled"`
	Permissions ResourcePermissions `json:"permissions"`
}

// ThreeScaleHAndler represents the minimal info that a user needs to know from the UI to link a service with 3Scale site
type ThreeScaleHandler struct {
	Name        string `json:"name"`
	ServiceId   string `json:"serviceId"`
	SystemUrl   string `json:"systemUrl"`
	AccessToken string `json:"accessToken"`
}

type ThreeScaleHandlers []ThreeScaleHandler

type ThreeScaleServiceRule struct {
	ServiceName           string `json:"serviceName"`
	ServiceNamespace      string `json:"serviceNamespace"`
	ThreeScaleHandlerName string `json:"threeScaleHandlerName"`
}

type Iter8ExperimentDetail struct {
	ExperimentItem  Iter8ExperimentItem   `json:"experimentItem"`
	CriteriaDetails []Iter8CriteriaDetail `json:"criterias"`
	TrafficControl  Iter8TrafficControl   `json:"trafficControl"`
	Permissions     ResourcePermissions   `json:"permissions"`
}

type Iter8Info struct {
	Enabled bool `json:"enabled"`
}

type Iter8ExperimentItem struct {
	Name                   string   `json:"name"`
	Phase                  string   `json:"phase"`
	CreatedAt              int64    `json:"createdAt"`
	Status                 string   `json:"status"`
	Baseline               string   `json:"baseline"`
	BaselinePercentage     int      `json:"baselinePercentage"`
	Candidate              string   `json:"candidate"`
	CandidatePercentage    int      `json:"candidatePercentage"`
	Namespace              string   `json:"namespace"`
	StartedAt              int64    `json:"startedAt"`
	EndedAt                int64    `json:"endedAt"`
	TargetService          string   `json:"targetService"`
	TargetServiceNamespace string   `json:"targetServiceNamespace"`
	AssessmentConclusion   []string `json:"assessmentConclusion"`
}

type Iter8CriteriaDetail struct {
	Name     string                     `json:"name"`
	Criteria Iter8Criteria              `json:"criteria"`
	Metric   Iter8Metric                `json:"metric"`
	Status   Iter8SuccessCrideriaStatus `json:"status"`
}

type Iter8Metric struct {
	AbsentValue        string `json:"absent_value"`
	IsCounter          bool   `json:"is_counter"`
	QueryTemplate      string `json:"query_template"`
	SampleSizeTemplate string `json:"sample_size_template"`
}

type Iter8TrafficControl struct {
	Algorithm            string  `json:"algorithm"`
	Interval             string  `json:"interval"`
	MaxIterations        int     `json:"maxIterations"`
	MaxTrafficPercentage float64 `json:"maxTrafficPercentage"`
	TrafficStepSize      float64 `json:"trafficStepSize"`
}

type Iter8Criteria struct {
	Metric        string  `json:"metric"`
	ToleranceType string  `json:"toleranceType"`
	Tolerance     float64 `json:"tolerance"`
	SampleSize    int     `json:"sampleSize"`
	StopOnFailure bool    `json:"stopOnFailure"`
}

type Iter8SuccessCrideriaStatus struct {
	Conclusions         []string `json:"conclusions"`
	SuccessCriterionMet bool     `json:"success_criterion_met"`
	AbortExperiment     bool     `json:"abort_experiment"`
}

type JaegerInfo struct {
	Enabled              bool     `json:"enabled"`
	Integration          bool     `json:"integration"`
	URL                  string   `json:"url"`
	NamespaceSelector    bool     `json:"namespaceSelector"`
	WhiteListIstioSystem []string `json:"whiteListIstioSystem"`
}

// GrafanaInfo provides information to access Grafana dashboards
type GrafanaInfo struct {
	ExternalLinks []kmodel.ExternalLink `json:"externalLinks"`
}

type NamespaceOverview struct {
	Health      NamespaceAppHealth     `json:"health"`
	Metrics     KialiMetrics           `json:"metrics"`
	Validations IstioValidationSummary `json:"validations"`
}

type ServicesOverview struct {
	Inbound  DashboardResponse `json:"inbound"`
	Outbound DashboardResponse `json:"outbound"`
	Detail   ServiceDetails    `json:"detail"`
	Graph    GraphConfig       `json:"graph"`
	Health   ServiceHealth     `json:"health"`
}
type WorkloadOverview struct {
	Inbound  DashboardResponse `json:"inbound"`
	Outbound DashboardResponse `json:"outbound"`
	Detail   Workload          `json:"detail"`
	Graph    GraphConfig       `json:"graph"`
	Health   WorkloadHealth    `json:"health"`
}
type AppOverview struct {
	Inbound  DashboardResponse `json:"inbound"`
	Outbound DashboardResponse `json:"outbound"`
	Detail   App               `json:"detail"`
	Health   AppHealth         `json:"health"`
}

// KialiTokenInfo provides Kiali Token and its expired time for each service domain
type KialiTokenInfo struct {
	ServiceDomain     string    `json:"serviceDomainID"`
	KialiToken        string    `json:"kialiToken"`
	KialiTokenExpired time.Time `json:"kialiTokenExpiredTime"`
}

// KialiLoginResponse match the response format of kiali authenticate API
type KialiLoginResponse struct {
	Token     string `json: "token"`
	ExpiresOn string `json: "expiresOn"`
}

// CombineResponse provides status code and body response for combined APIs
type CombineResponse struct {
	StatusCode int             `json: statuscode`
	Result     json.RawMessage `json: result`
}

const kialiAuthenticateURL = "http://kiali.istio-system.svc:20001/kiali/api/authenticate"
const kialiAPIURL = "http://kiali.istio-system.svc:20001/kiali/api"

// KialiTokenInfos provides all known kiali token for specific service domains and used for sync
type KialiTokenInfos struct {
	Tokens map[string]KialiTokenInfo `json: KialiTokens`
	mux    sync.Mutex
}

// KialiTokens store all known kiali token
var KialiTokens = KialiTokenInfos{Tokens: make(map[string]KialiTokenInfo)}

// UpdateToken updates kiali tokens to KialiTokenInfos
func UpdateToken(context context.Context, sd string, ap *base.AuthContext, msgSvc api.WSMessagingService) {
	newKialiToken, newExpiredTime, err := getKialiToken(context, sd, ap, msgSvc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Fail to get kiali token, error: %v"), err)
		return
	}
	KialiTokens.Tokens[sd] = KialiTokenInfo{sd, newKialiToken, newExpiredTime}
}

// getKialiToken gets kiali token by send post request to Kiali authenticat APIs in service domains
func getKialiToken(context context.Context, serviceDomain string, ap *base.AuthContext, msgSvc api.WSMessagingService) (string, time.Time, error) {
	url := kialiAuthenticateURL
	var jsonStr = []byte{}
	r, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Fail to generate new http request: %v"), err)
	}
	r.Header.Set("Content-Type", "application/json")
	//default username: sherlock, default passwd: $h3rl0ck!
	username := "sherlock"
	passwd := "$h3rl0ck!"
	token := []byte(username + ":" + passwd)
	encodeToken := base64.StdEncoding.EncodeToString(token)
	r.Header.Set("X-Ntnx-Xks-Authorization", "Basic "+encodeToken)
	ctx := r.Context()
	glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: path: %s, request: %+v"), url, r)

	// var err error
	// use for/break as structured goto
	// RBAC
	// TODO: allow operator to impersonate tenant:
	// if operator, fetch tenant id for edge
	if !auth.IsOperatorRole(ap) && !auth.IsInfraAdminRole(ap) {
		err = fmt.Errorf("Permission denied: tid|uid=%s|%s", ap.TenantID, ap.ID)
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: RBAC error: %s"), err)
		return "", time.Now(), err
	}
	rewriteHeaders(ctx, r)
	resp, err2 := msgSvc.SendHTTPRequest(ctx, ap.TenantID, serviceDomain, r, url)
	if err2 != nil {
		err = err2
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: send req[%+v], url[%s], error: %s"), r, url, err)
		return "", time.Now(), err

	}
	glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: Response: %+v"), *resp)
	baResp, err2 := ioutil.ReadAll(resp.Body)
	if err2 != nil {
		err = err2
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: req[%+v], response error: %s"), r, err)
		return "", time.Now(), err

	}
	defer resp.Body.Close()
	lr := KialiLoginResponse{}
	err = json.Unmarshal(baResp, &lr)
	expiredTime, err2 := time.Parse(time.RFC1123Z, lr.ExpiresOn)
	if err2 != nil {
		err = err2
		glog.Warningf(base.PrefixRequestID(ctx, "Fail to parse kiali token expired time: %v %v"), lr.ExpiresOn, err)
		return "", time.Now(), err
	}
	return lr.Token, expiredTime, err
}

// ParseServiceDomain get service domain from query parameters and return serviceDomainID, KialiToken
func ParseServiceDomain(context context.Context, url *url.URL, ap *base.AuthContext, msgSvc api.WSMessagingService) (string, string, error) {
	var serviceDomain string
	if query := url.Query(); query != nil {
		serviceDomain = query.Get("serviceDomain")
		if serviceDomain != "" {
			delete(query, "serviceDomain")
			KialiTokens.mux.Lock()
			if v, ok := KialiTokens.Tokens[serviceDomain]; ok {
				if v.KialiTokenExpired.Before(time.Now()) {
					UpdateToken(context, serviceDomain, ap, msgSvc)
				}
			} else {
				UpdateToken(context, serviceDomain, ap, msgSvc)
			}
			if len(KialiTokens.Tokens) > 1000 {
				for key, kialiInfos := range KialiTokens.Tokens {
					if kialiInfos.KialiTokenExpired.Before(time.Now()) {
						delete(KialiTokens.Tokens, key)
					}
				}
			}
			if KialiTokens.Tokens[serviceDomain].KialiToken == "" {
				err := errors.New("Fail to parse kiali token")
				KialiTokens.mux.Unlock()
				return serviceDomain, "", err
			}
			KialiTokens.mux.Unlock()
		} else {
			err := errors.New("Service domain must be specified via the serviceDomain query parameter")
			return "", "", err
		}
	} else {
		err := errors.New("Service domain must be specified via the serviceDomain query parameter")
		return "", "", err
	}
	return serviceDomain, KialiTokens.Tokens[serviceDomain].KialiToken, nil
}

// ParsePath parse original request to kiali format request
func ParsePath(url *url.URL) string {
	path := kialiAPIURL
	subpath := strings.Split(url.Path, "v1.0/kiali")[1]
	if subpath == "/graph" {
		path += "/namespaces"
	}
	path += subpath
	if query := url.Query(); query != nil {
		// serviceDomain is used only for getting kiali token
		delete(query, "serviceDomain")
		if len(query) > 0 {
			path += "?"
			for key, value := range query {
				elements := strings.Join(value, ", ")
				path += key + "=" + elements + "&"
			}
		}
	}
	return path
}

// ParseCombinePath parse custom request to kiali format requests, ex: overview
func ParseCombinePath(url *url.URL) (map[string]string, error) {
	paths := make(map[string]string)
	var namespaceOverview = regexp.MustCompile(`namespaces\/.*\/overview`)
	var serviceOverview = regexp.MustCompile(`namespaces\/.*\/services\/.*\/overview`)
	var workloadOverview = regexp.MustCompile(`namespaces\/.*\/workloads\/.*\/overview`)
	var appOverview = regexp.MustCompile(`namespaces\/.*\/apps\/.*\/overview`)
	switch subpath := strings.Split(url.Path, "v1.0/kiali")[1]; {
	case appOverview.MatchString(subpath):
		subpath = strings.ReplaceAll(subpath, "/overview", "")
		paths = map[string]string{
			"outbound": kialiAPIURL + subpath + "/dashboard",
			"inbound":  kialiAPIURL + subpath + "/dashboard?direction=inbound",
			"health":   kialiAPIURL + subpath + "/health",
			"detail":   kialiAPIURL + subpath,
		}
	case serviceOverview.MatchString(subpath):
		subpath = strings.ReplaceAll(subpath, "/overview", "")
		paths = map[string]string{
			"outbound": kialiAPIURL + subpath + "/dashboard",
			"inbound":  kialiAPIURL + subpath + "/dashboard?direction=inbound",
			"health":   kialiAPIURL + subpath + "/health",
			"detail":   kialiAPIURL + subpath,
			"graph":    kialiAPIURL + subpath + "/graph?graphType=service",
		}
	case workloadOverview.MatchString(subpath):
		subpath = strings.ReplaceAll(subpath, "/overview", "")
		paths = map[string]string{
			"outbound": kialiAPIURL + subpath + "/dashboard",
			"inbound":  kialiAPIURL + subpath + "/dashboard?direction=inbound",
			"health":   kialiAPIURL + subpath + "/health",
			"detail":   kialiAPIURL + subpath,
			"graph":    kialiAPIURL + subpath + "/graph?graphType=workload",
		}
	case namespaceOverview.MatchString(subpath):
		subpath = strings.ReplaceAll(subpath, "/overview", "")
		paths = map[string]string{
			"health":      kialiAPIURL + subpath + "/health",
			"metrics":     kialiAPIURL + subpath + "/metrics",
			"validations": kialiAPIURL + subpath + "/validations",
		}
	default:
		err := errors.New("Unable to parse the combine path")
		return nil, err
	}
	return paths, nil
}

func makeHTTPproxyKialiOriginAPI(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService, msg string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		ctx := r.Context()
		edgeID, kialitoken, err := ParseServiceDomain(ctx, r.URL, ap, msgSvc)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Please give a valid service Domain ID: %v"), err)
			w.WriteHeader(http.StatusBadRequest)
			handleResponse(w, r, err, "PROXY %s, tenantID=%s", msg, ap.TenantID)
			return
		}
		path := ParsePath(r.URL)
		requestBody, err := ioutil.ReadAll(r.Body)
		r, err = http.NewRequest(r.Method, path, bytes.NewBuffer(requestBody))
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Fail to generate new http request: %v"), err)
		}
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Ntnx-Xks-Authorization", "Bearer "+kialitoken)
		glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: path: %s, request: %+v"), path, r)

		// var err error
		// use for/break as structured goto
		for {
			// RBAC
			// TODO: allow operator to impersonate tenant:
			// if operator, fetch tenant id for edge
			if !auth.IsOperatorRole(ap) && !auth.IsInfraAdminRole(ap) {
				err = fmt.Errorf("Permission denied: tid|uid=%s|%s", ap.TenantID, ap.ID)
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: RBAC error: %s"), err)
				break
			}
			rewriteHeaders(ctx, r)
			resp, err2 := msgSvc.SendHTTPRequest(ctx, ap.TenantID, edgeID, r, path)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: send req[%+v], url[%s], error: %s"), r, path, err)
				break
			}
			header := w.Header()
			for k, v := range resp.Header {
				header[k] = v
			}
			glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: Response: %+v"), *resp)
			baResp, err2 := ioutil.ReadAll(resp.Body)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: req[%+v], response error: %s"), r, err)
				break
			}
			defer resp.Body.Close()

			// write response status code
			w.WriteHeader(resp.StatusCode)

			_, err2 = w.Write(baResp)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: req[%+v], write error: %s"), r, err)
				break
			}
			break
		}
		handleResponse(w, r, err, "PROXY %s, tenantID=%s", msg, ap.TenantID)
	}))
}

func makeHTTPproxyKialiCombineAPI(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService, msg string) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		w.Header().Set("Content-Type", "application/json")
		edgeID, kialitoken, err := ParseServiceDomain(r.Context(), r.URL, ap, msgSvc)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(r.Context(), "Please give a valid service Domain ID: %v"), err)
			w.WriteHeader(http.StatusBadRequest)
			handleResponse(w, r, err, "PROXY %s, tenantID=%s", msg, ap.TenantID)
			return
		}
		paths, err := ParseCombinePath(r.URL)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(r.Context(), "Fail to parse url: %v"), err)
		}
		requestBody, err := ioutil.ReadAll(r.Body)
		comResp := make(map[string]CombineResponse)
		for key, path := range paths {
			r, err = http.NewRequest(r.Method, path, bytes.NewBuffer(requestBody))
			ctx := r.Context()
			if err != nil {
				glog.Warningf(base.PrefixRequestID(r.Context(), "Fail to generate new http request: %v"), err)
			}
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Ntnx-Xks-Authorization", "Bearer "+kialitoken)

			glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: path: %s, request: %+v"), path, r)

			// var err error
			// use for/break as structured goto
			for {
				// RBAC
				// TODO: allow operator to impersonate tenant:
				// if operator, fetch tenant id for edge
				if !auth.IsOperatorRole(ap) && !auth.IsInfraAdminRole(ap) {
					err = fmt.Errorf("Permission denied: tid|uid=%s|%s", ap.TenantID, ap.ID)
					glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: RBAC error: %s"), err)
					break
				}
				rewriteHeaders(ctx, r)

				resp, err2 := msgSvc.SendHTTPRequest(ctx, ap.TenantID, edgeID, r, path)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: send req[%+v], url[%s], error: %s"), r, path, err)
					break
				}
				glog.V(4).Infof(base.PrefixRequestID(ctx, "HTTP proxy handler: Response: %+v"), *resp)
				baResp, err2 := ioutil.ReadAll(resp.Body)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy handler: req[%+v], response error: %s"), r, err)
					break
				}
				defer resp.Body.Close()

				// combine each api's status code and body response together
				comResp[key] = CombineResponse{resp.StatusCode, baResp}
				break
			}
		}
		err2 := json.NewEncoder(w).Encode(comResp)
		if err2 != nil {
			err = err2
			glog.Warningf(base.PrefixRequestID(r.Context(), "cannot write combine response,  response: %v, error: %v"), comResp, err)
			handleResponse(w, r, err, "PROXY %s, tenantID=%s", msg, ap.TenantID)
		}
	}))
}

func getKialiProxyRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "GET",
			path:   "/v1.0/kiali",
			// swagger:route GET /v1.0/kiali Kiali root
			// ---
			// Endpoint to get the status of Kiali
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: statusInfo
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/status",
			// swagger:route GET /v1.0/kiali/status Kiali getStatus
			// ---
			// Endpoint to get the status of Kiali
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: statusInfo
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/status/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/istio/status",
			// swagger:route GET /v1.0/kiali/istio/status Kiali istioStatus
			// ---
			// Get the status of each components needed in the control plane
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: istioStatusResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/istio/status/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/config",
			// swagger:route GET /v1.0/kiali/config Kiali getConfig
			// ---
			// Endpoint to get the config of Kiali
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: statusInfo
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/config/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/istio/permissions",
			// swagger:route GET /v1.0/kiali/istio/permissions Kiali getPermissions
			//
			// ntnx:ignore
			//
			// Endpoint to get the caller permissions on new Istio Config objects
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: istioConfigPermissions
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/istio/permissions/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/istio",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/istio Kiali istioConfigList
			// ---
			// Endpoint to get the list of Istio Config of a namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: istioConfigList
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/istio/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/istio/:object_type/:object",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/istio/{object_type}/{object} Kiali istioConfigDetails
			//
			// ntnx:ignore
			//
			// Endpoint to get the Istio Config of an Istio object
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: istioConfigDetailsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/kiali/namespaces/:namespace/istio/:object_type/:object",
			// swagger:route DELETE /v1.0/kiali/namespaces/{namespace}/istio/{object_type}/{object} Kiali istioConfigDelete
			//
			// ntnx:ignore
			//
			// Endpoint to delete the Istio Config of an (arbitrary) Istio object
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       400: APIError
			//       200
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "PATCH",
			path:   "/v1.0/kiali/namespaces/:namespace/istio/:object_type/:object",
			// swagger:route PATCH /v1.0/kiali/namespaces/{namespace}/istio/{object_type}/{object} Kiali istioConfigUpdate
			//
			// ntnx:ignore
			//
			// Endpoint to update the Istio Config of an Istio object used for templates and adapters using Json Merge Patch strategy
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: istioConfigDetailsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/namespaces/:namespace/istio/:object_type",
			// swagger:route POST /v1.0/kiali/namespaces/{namespace}/istio/{object_type} Kiali istioConfigCreate
			//
			// ntnx:ignore
			//
			// Endpoint to create an Istio object by using an Istio Config item
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       202
			//       201: istioConfigDetailsResponse
			//       200: istioConfigDetailsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services Kiali serviceList
			// ---
			// Endpoint to get the details of a given service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: serviceListResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service} Kiali serviceDetails
			// ---
			// Endpoint to get the details of a given service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: serviceDetailsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/metrics",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/metrics Kiali serviceMetrics
			// ---
			// Endpoint to fetch metrics to be displayed, related to a single service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: kialimetricsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/metrics/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/health",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/health Kiali serviceHealth
			// ---
			// Get health associated to the given service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: serviceHealthResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/health/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/spans",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/spans Kiali spansList
			// ---
			// Endpoint to get Jaeger spans for a given service
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: spansResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/spans/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/traces",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/traces Kiali tracesList
			// ---
			// Endpoint to get the traces of a given service.
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: tracesDetailResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/traces/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/errortraces",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/errortraces Kiali errorTraces
			// ---
			// Endpoint to get the number of traces in error for a given service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: errorTracesResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/errortraces/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/traces/:traceID",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/traces Kiali tracesDetail
			// ---
			// Endpoint to get a specific trace of a given service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: tracesDetailResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/dashboard",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/dashboard Kiali serviceDashboard
			// ---
			// Endpoint to fetch dashboard to be displayed, related to a single service
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: dashboardResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/dashboard/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads Kiali workloadList
			// ---
			// Endpoint to get the list of workloads for a namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: workloadListResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads/{workload} Kiali workloadDetails
			// ---
			// Endpoint to get the workload details
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: workloadDetails
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/metrics",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads/{workload}/metrics Kiali workloadMetrics
			// ---
			// Endpoint to fetch metrics to be displayed, related to a single workload
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: kialimetricsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/metrics/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/dashboard",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads/{workload}/dashboard Kiali workloadDashboard
			// ---
			// Endpoint to fetch dashboard to be displayed, related to a single workload
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: dashboardResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/dashboard/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/health",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads/{workload}/health Kiali workloadHealth
			// ---
			// Get health associated to the given workload
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: workloadHealthResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/health/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/apps Kiali appList
			// ---
			// Endpoint to get the list of apps for a namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: appListResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/apps/{app} Kiali appDetails
			// ---
			// Endpoint to get the app details
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: appDetails
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/metrics",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/apps/{app}/metrics Kiali appMetrics
			// ---
			// Endpoint to fetch metrics to be displayed, related to a single app
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: kialimetricsResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/metrics/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/dashboard",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/apps/{app}/dashboard Kiali appDashboard
			// ---
			// Endpoint to fetch dashboard to be displayed, related to a single app
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: dashboardResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/dashboard/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/health",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/apps/{app}/health Kiali appHealth
			// ---
			// Get health associated to the given app
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: appHealthResponse
			//       default: APIError
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/health/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces",
			// swagger:route GET /v1.0/kiali/namespaces Kiali namespaceList
			// ---
			// Endpoint to get the list of the available namespaces
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: namespaceList
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/health",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/health Kiali namespaceHealth
			// ---
			// Get health for all objects in the given namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: namespaceAppHealthResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/health/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/metrics",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/metrics Kiali namespaceMetrics
			// ---
			// Endpoint to fetch metrics to be displayed, related to a namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: kialimetricsResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/metrics/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/validations",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/validations Kiali namespaceValidations
			// ---
			// Get validation summary for all objects in the given namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: namespaceValidationSummaryResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/validations/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/tls",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/tls Kiali namespaceTls
			// ---
			// Get TLS status for the given namespace
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: namespaceTlsResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/tls/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/pods/:pod",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/pods/{pod} Kiali podDetails
			// ---
			// Endpoint to get pod details
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: workloadDetails
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/pods/:pod/logs",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/pods/{pod}/logs Kiali podLogs
			// ---
			// Endpoint to get pod logs
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: workloadDetails
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/pods/:pod/logs/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/graph",
			// swagger:route GET /v1.0/kiali/namespaces/graph Kiali graphNamespaces
			// ---
			// The backing JSON for a namespaces graph.
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: graphResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/graph/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/applications/:app/versions/:version/graph",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/applications/{app}/versions/{version}/graph Kiali graphAppVersion
			// ---
			// The backing JSON for a versioned app node detail graph. (supported graphTypes: app | versionedApp)
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: graphResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/applications/:app/versions/:version/graph/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/applications/:app/graph",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/applications/{app}/graph Kiali graphApp
			// ---
			// The backing JSON for an app node detail graph. (supported graphTypes: app | versionedApp)
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: graphResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/applications/:app/graph/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/graph",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/graph Kiali graphService
			// ---
			// The backing JSON for a service node detail graph.
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: graphResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/graph/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/graph",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads/{workload}/graph Kiali graphWorkload
			// ---
			// The backing JSON for a workload node detail graph.
			//
			//     Produces:
			//     - application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: graphResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/graph/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/threescale",
			// swagger:route GET /v1.0/kiali/threescale Kiali getThreeScaleInfo
			//
			// ntnx:ignore
			//
			// Endpoint to check if threescale adapter is present in the cluster and if user can write adapter config
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleInfoResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/threescale/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/threescale/handlers",
			// swagger:route GET /v1.0/kiali/threescale/handlers Kiali getThreeScaleHandlers
			//
			// ntnx:ignore
			//
			// Endpoint to fetch threescale handlers generated from Kiali
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleHandlersResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/threescale/handlers/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/threescale/handlers",
			// swagger:route POST /v1.0/kiali/threescale/handlers Kiali postThreeScaleHandlers
			//
			// ntnx:ignore
			//
			// Endpoint to create a new threescale handler+instance generated by Kiali
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleHandlersResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/threescale/handlers/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "PATCH",
			path:   "/v1.0/kiali/threescale/handlers/:threescaleHandlerName",
			// swagger:route PATCH /v1.0/kiali/threescale/handlers/{threescaleHandlerName} Kiali patchThreeScaleHandler
			//
			// ntnx:ignore
			//
			// Endpoint to update an existing threescale handler generated by Kiali
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleHandlersResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/kiali/threescale/handlers/:threescaleHandlerName",
			// swagger:route DELETE /v1.0/kiali/threescale/handlers/{threescaleHandlerName} Kiali deleteThreeScaleHandler
			//
			// ntnx:ignore
			//
			// Endpoint to delete an existing threescale handler+instance generated by Kiali
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleHandlersResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/threescale/namespaces/:namespace/services/:service",
			// swagger:route GET /v1.0/kiali/threescale/namespaces/{namespace}/services/{service} Kiali getThreeScaleService
			//
			// ntnx:ignore
			//
			// Endpoint to get an existing threescale rule for a given service
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleRuleResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/threescale/namespaces/:namespace/services",
			// swagger:route POST /v1.0/kiali/threescale/namespaces/{namespace}/services Kiali postThreeScaleService
			//
			// ntnx:ignore
			//
			// Endpoint to create a new threescale rule for a given service
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleRuleResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/threescale/namespaces/:namespace/services/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "PATCH",
			path:   "/v1.0/kiali/threescale/namespaces/:namespace/services/:service",
			// swagger:route PATCH /v1.0/kiali/threescale/namespaces/{namespace}/services/{service} Kiali patchThreeScaleService
			//
			// ntnx:ignore
			//
			// Endpoint to update an existing threescale rule for a given service
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: threeScaleRuleResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/kiali/threescale/namespaces/:namespace/services/:service",
			// swagger:route DELETE /v1.0/kiali/threescale/namespaces/{namespace}/services/{service} Kiali deleteThreeScaleService
			//
			// ntnx:ignore
			//
			// Endpoint to delete an existing threescale rule for a given service
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       400: APIError
			//       200
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8",
			// swagger:route GET /v1.0/kiali/iter8 Kiali getIter8
			//
			// ntnx:ignore
			//
			// Endpoint to check if threescale adapter is present in the cluster and if user can write adapter config
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8StatusResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8/namespaces/:namespace/experiments/:name",
			// swagger:route GET /v1.0/kiali/iter8/namespaces/{namespace}/experiments/{name} Kiali getIter8Experiments
			//
			// ntnx:ignore
			//
			// Endpoint to fetch iter8 experiments by namespace and name.
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8ExperimentGetDetailResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8/experiments",
			// swagger:route GET /v1.0/kiali/iter8/experiments Kiali iter8Experiments
			//
			// ntnx:ignore
			//
			// Endpoint to fetch iter8 experiments for all namespaces user have access.
			// User can define a comman separated list of namespaces.
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8ExperimentsResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8/experiments/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/iter8/namespaces/:namespace/experiments",
			// swagger:route POST /v1.0/kiali/iter8/namespaces/{namespace}/experiments Kiali postIter8Experiments
			//
			// ntnx:ignore
			//
			// Endpoint to create new iter8 experiments for a given namespace.
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8ExperimentGetDetailResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "POST",
			path:   "/v1.0/kiali/iter8/namespaces/:namespace/experiments/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "PATCH",
			path:   "/v1.0/kiali/iter8/namespaces/:namespace/experiments/:name",
			// swagger:route PATCH /v1.0/kiali/iter8/experiments/{namespace}/name/{name} Kiali patchIter8Experiments
			//
			// ntnx:ignore
			//
			// Endpoint to update new iter8 experiment (for abort purpose)
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8ExperimentGetDetailResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "DELETE",
			path:   "/v1.0/kiali/iter8/namespaces/:namespace/experiments/:name",
			// swagger:route DELETE /v1.0/kiali/iter8/experiments/namespaces/{namespace}/name/{name} Kiali deleteIter8Experiments
			//
			// ntnx:ignore
			//
			// Endpoint to delete   iter8 experiments
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8StatusResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8/metrics",
			// swagger:route GET /v1.0/kiali/iter8/metrics Kiali getIter8Metrics
			//
			// ntnx:ignore
			//
			// Endpoint to get the analytics metrics
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: iter8StatusResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/iter8/metrics/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/grafana",
			// swagger:route GET /v1.0/kiali/grafana Kiali grafanaInfo
			//
			// ntnx:ignore
			//
			// Get the grafana URL and other descriptors
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: grafanaInfoResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/grafana/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/jaeger",
			// swagger:route GET /v1.0/kiali/jaeger Kiali jaegerInfo
			//
			// ntnx:ignore
			//
			// Get the jaeger URL and other descriptors
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: jaegerInfoResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/jaeger/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/customdashboard/:dashboard",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/customdashboard/{dashboard} Kiali customDashboard
			//
			// ntnx:ignore
			//
			// Endpoint to fetch a custom dashboard
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: dashboardResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/mesh/tls",
			// swagger:route GET /v1.0/kiali/mesh/tls Kiali meshTls
			//
			// ntnx:ignore
			//
			// Get TLS status for the whole mesh
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: meshTlsResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/mesh/tls/",
			handle: makeHTTPproxyKialiOriginAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/overview",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/overview Kiali namespaceOverview
			//
			// ntnx:ignore
			//
			// Endpoint to get the overview info of a given namespace
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: namespaceOverviewResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/overview/",
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/overview",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/services/{service}/overview Kiali serviceOverview
			//
			// ntnx:ignore
			//
			// Endpoint to get the overview info of a given service
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: serviceOverviewResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/services/:service/overview/",
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/overview",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/workloads/{workload}/overview Kiali workloadOverview
			//
			// ntnx:ignore
			//
			// Endpoint to get the overview info of a given workload
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: workloadOverviewResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/workloads/:workload/overview/",
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		}, {
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/overview",
			// swagger:route GET /v1.0/kiali/namespaces/{namespace}/apps/{app}/overview Kiali appOverview
			//
			// ntnx:ignore
			//
			// Endpoint to get the overview info of a given application
			//
			//		Produces:
			//		- application/json
			//
			//		Schemes: http, https
			//
			//     	Security:
			//        - BearerToken:
			//
			// 		Responses:
			//       200: appOverviewResponse
			//       default: APIError
			//
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
		{
			method: "GET",
			path:   "/v1.0/kiali/namespaces/:namespace/apps/:app/overview/",
			handle: makeHTTPproxyKialiCombineAPI(dbAPI, msgSvc, "httpProxy"),
		},
	}
}

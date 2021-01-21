package model

import (
	"cloudservices/common/base"
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// EntityState is the type for state of entities like data streams and applications
type EntityState string

const (
	DeployEntityState   = EntityState("DEPLOY")
	UndeployEntityState = EntityState("UNDEPLOY")
	// EntityCRUDEventName is the name of the generic event related to CRUD operations
	EntityCRUDEventName = "EntityCRUDEvent"
)

var (
	ZeroUUID = base.MustGetUUIDFromBytes(nil)
)

func (entityState EntityState) Ptr() *EntityState {
	return &entityState
}

func (entityState EntityState) StringPtr() *string {
	return base.StringPtr(string(entityState))
}

// BaseModel is the common base for all per tenant objects
// allow empty ID in both create and update since new PUT endpoint will have ID,
// so allow omit ID in payload
type BaseModel struct {
	// ID of the entity
	// Maximum character length is 64 for project, category, and runtime environment,
	// 36 for other entity types.
	ID string `json:"id" db:"id" validate:"range=0:64,ignore=create"`
	// ntnx:ignore
	// Version of entity, implemented using timestamp in nano seconds
	// This is set to float64 since JSON numbers are floating point
	// May lose precision due to truncation but should have milli-second precision
	Version float64 `json:"version,omitempty" db:"version"`
	// ntnx:ignore
	// required: true
	// ID of the tenant this entity belongs to
	TenantID string `json:"tenantId" db:"tenant_id"`
	// ntnx:ignore
	// timestamp feature supported by DB
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	// ntnx:ignore
	// timestamp feature supported by DB
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// equivalent of BaseModel used by DBO structs
type BaseModelDBO struct {
	ID         string    `json:"id" db:"id"`
	Version    float64   `json:"version,omitempty" db:"version"`
	TenantID   string    `json:"tenantId" db:"tenant_id"`
	CreatedAt  time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
	TotalCount *int      `json:"totalCount,omitempty" db:"total_count"`
}

// EdgeBaseModel is the common base for all per edge objects
type EdgeBaseModel struct {
	// required: true
	BaseModel
	// ID of the edge this entity belongs to
	// required: true
	EdgeID string `json:"edgeId" db:"edge_id" validate:"range=1:36"`
}

// EdgeBaseModelDBO is equivalent of EdgeBaseModel used by DBO structs
type EdgeBaseModelDBO struct {
	BaseModelDBO
	EdgeID string `json:"edgeId" db:"edge_id"`
}

// EdgeDeviceScopedModel is the common base for all edge device scoped objects
type EdgeDeviceScopedModel struct {
	ClusterEntityModel
	DeviceID string `json:"deviceId" db:"device_id"`
}

// EdgeDeviceScopedModelDBO is equivalent of EdgeDeviceScopedModel used by DBO structs
type EdgeDeviceScopedModelDBO struct {
	ClusterEntityModelDBO
	DeviceID string `json:"deviceId" db:"device_id"`
}

// ClusterEntityModel is the model for an entity in the cluster
type ClusterEntityModel struct {
	// required: true
	BaseModel
	// ID of the cluster this entity belongs to
	// required: true
	ClusterID string `json:"clusterId" db:"edge_cluster_id"`
}

// ClusterEntityModelDBO is the DB model for an entity in the cluster
type ClusterEntityModelDBO struct {
	// required: true
	BaseModelDBO
	// ID of the cluster this entity belongs to
	// required: true
	ClusterID string `json:"clusterId" db:"edge_cluster_id"`
}

// NodeEntityModel is the common base for all node entity objects
type NodeEntityModel struct {
	ServiceDomainEntityModel
	NodeID string `json:"nodeId" db:"device_id"`
}

// NodeEntityModelDBO is equivalent of NodeEntityModel used by DBO structs
type NodeEntityModelDBO struct {
	ServiceDomainEntityModelDBO
	NodeID string `json:"nodeId" db:"device_id"`
}

// ServiceDomainEntityModel is the model for an entity in the service domain
type ServiceDomainEntityModel struct {
	// required: true
	BaseModel
	// ID of the service domain this entity belongs to
	// required: true
	SvcDomainID string `json:"svcDomainId"`
}

// ServiceDomainEntityModelDBO is the DB model for an entity in the service domain
type ServiceDomainEntityModelDBO struct {
	// required: true
	BaseModelDBO
	// ID of the service domain this entity belongs to
	// required: true
	SvcDomainID string `json:"svcDomainId" db:"edge_cluster_id"`
}

// CreateDocumentResponse - create document response struct
type CreateDocumentResponse struct {
	// ID of the created entity
	// required: true
	ID string `json:"_id"`
}

type ArtifactBaseModel struct {
	Data map[string]interface{} `json:"data"`
	// ntnx:ignore
	// Version of entity, implemented using timestamp in nano seconds
	Version   int64     `json:"version,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type ArtifactBaseModelDBO struct {
	ID       int64            `json:"id" db:"id"`
	TenantID string           `json:"tenantId" db:"tenant_id"`
	Data     *json.RawMessage `json:"data,omitempty" db:"data"`
	// Version of entity, implemented using timestamp in nano seconds
	Version   int64     `json:"version,omitempty" db:"version"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
}

// CreateDocumentResponseV2 - create document response struct
type CreateDocumentResponseV2 struct {
	// ID of the created entity
	// required: true
	ID string `json:"id"`
}

// Ok
// swagger:response CreateDocumentResponse
type CreateDocumentResponseWrapper struct {
	// in: body
	// required: true
	Payload *CreateDocumentResponse
}

// Ok
// swagger:response CreateDocumentResponseV2
type CreateDocumentResponseWrapperV2 struct {
	// in: body
	// required: true
	Payload *CreateDocumentResponseV2
}

// DeleteDocumentResponse - delete document response struct
type DeleteDocumentResponse struct {
	// ID of the deleted entity. Set to empty string if the no entity was found
	// with the given ID.
	// required: true
	ID string `json:"_id"`
}

// DeleteDocumentResponseV2 - delete document response struct
type DeleteDocumentResponseV2 struct {
	// ID of the deleted entity. Set to empty string if the no entity was found
	// with the given ID.
	// required: true
	ID string `json:"id"`
}

// Ok
// swagger:response DeleteDocumentResponse
type DeleteDocumentResponseWrapper struct {
	// in: body
	// required: true
	Payload *DeleteDocumentResponse
}

// Ok
// swagger:response DeleteDocumentResponseV2
type DeleteDocumentResponseWrapperV2 struct {
	// in: body
	// required: true
	Payload *DeleteDocumentResponseV2
}

// UpdateDocumentResponse - update document response struct
type UpdateDocumentResponse struct {
	// ID of the updated entity
	// required: true
	ID string `json:"_id"`
}

// UpdateDocumentResponseV2 - update document response struct
type UpdateDocumentResponseV2 struct {
	// ID of the updated entity
	// required: true
	ID string `json:"id"`
}

// Ok
// swagger:response UpdateDocumentResponse
type UpdateDocumentResponseWrapper struct {
	// in: body
	// required: true
	Payload *UpdateDocumentResponse
}

// Ok
// swagger:response UpdateDocumentResponseV2
type UpdateDocumentResponseWrapperV2 struct {
	// in: body
	// required: true
	Payload *UpdateDocumentResponseV2
}

// A IDParams parameter model.
//
// This is used for operations that require the ID of an entity in the path
// Typically: Delete, Update or per entity Get operations
// swagger:parameters ApplicationDelete ApplicationDeleteV2 GetApplicationContainers DockerProfileDelete DockerProfileDeleteV2 CategoryDelete CategoryDeleteV2 CloudCredsDelete CloudProfileDelete EdgeUpgradeDelete EdgeUpgradeDeleteV2 DataSourceDelete DataSourceDeleteV2 DataStreamDelete DataPipelineDelete GetDataPipelineContainers EdgeCertDelete EdgeCertDeleteV2 ProjectDelete ProjectDeleteV2 ScriptDelete FunctionDelete ScriptRuntimeDelete RuntimeEnvironmentDelete SensorDelete SensorDeleteV2 UserDelete UserDeleteV2 UserPropsDelete UserPropsDeleteV2 TenantPropsDelete TenantPropsDeleteV2 ApplicationGet ApplicationGetV2 ApplicationStatusGet ApplicationStatusGetV2 ApplicationStatusDelete ApplicationStatusDeleteV2 CategoryGet CategoryGetV2 CategoryUsageGet DockerProfileGet DockerProfileGetV2 CloudCredsGet CloudProfileGet DataSourceGet DataSourceGetV2 DataSourceGetArtifactV2 DataSourceCreateArtifactV2 DataStreamGet DataPipelineGet EdgeCertGet EdgeCertGetV2 ScriptGet FunctionGet ScriptRuntimeGet RuntimeEnvironmentGet SensorGet SensorGetV2 UserGet UserGetV2 UserPropsGet UserPropsGetV2 TenantPropsGet TenantPropsGetV2 LogEntryDelete LogEntryDeleteV2 ContainerRegistryDelete ContainerRegistryDeleteV2 ContainerRegistryGet ContainerRegistryGetV2 ContainerRegistryUpdate ApplicationUpdateV2 CategoryUpdateV2 CloudCredsUpdateV2 DataSourceUpdateV2 DataStreamUpdateV2 DockerProfileUpdateV2 EdgeUpdateV2 ProjectUpdateV2 ScriptUpdateV2 ScriptRuntimeUpdateV2 SensorUpdateV2 UserUpdateV2 UserPropsUpdate TenantPropsUpdate ContainerRegistryUpdateV2 ApplicationUpdateV3 CategoryUpdateV3 CloudProfileUpdate DataSourceUpdateV3 DataPipelineUpdate DockerProfileUpdateV3 EdgeUpdateV3 ProjectUpdateV3 FunctionUpdate RuntimeEnvironmentUpdate SensorUpdateV3 UserUpdateV3 UserPropsUpdateV2 TenantPropsUpdateV2 EdgeInfoUpdate EdgeInfoUpdateV2 AuditLogGet AuditLogGetV2 MLModelGet MLModelDelete MLModelUpdate MLModelVersionCreate MLModelVersionUpdate MLModelVersionDelete MLModelVersionURLGet MLModelStatusGet MLModelStatusDelete EdgeLogEntriesGetV2 ApplicationLogEntriesGetV2 InfraConfigGet UserApiTokenUpdate UserApiTokenDelete EdgeDeviceUpdate EdgeClusterUpdate EdgeDeviceInfoUpdate EdgeDeviceOnboardedByID ProjectServiceGet ProjectServiceDelete ProjectServiceUpdate LogCollectorUpdate LogCollectorDelete LogCollectorStart LogCollectorStop LogCollectorGet HelmAppGetYaml HelmValuesCreate HelmApplicationUpdate StorageProfileUpdate HTTPServiceProxyUpdate HTTPServiceProxyDelete HTTPServiceProxyGet KubernetesClustersUpdate KubernetesClustersDelete KubernetesClustersHandle KubernetesClustersGet TenantDelete TenantGetByID DataDriverClassGet DataDriverClassUpdate DataDriverClassDelete DataDriverInstanceGet DataDriverInstanceUpdate DataDriverInstanceDelete DataDriverInstancesByClassIdList DataDriverConfigGet DataDriverConfigUpdate DataDriverConfigDelete DataDriverConfigList DataDriverStreamGet DataDriverStreamUpdate DataDriverStreamDelete DataDriverStreamList
type IDParams struct {
	// ID of the entity
	// in: path
	// required: true
	ID string `json:"id"`
}

// EdgeIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter is an edge id
// Typically used for edge entity or per edge entities
// swagger:parameters GetApplicationContainers EdgeGetDatasources EdgeGetSensors EdgeGetHandle EdgeGet EdgeDelete EdgeGetUpgrades EdgeInfoGet EdgeGetDatasourcesV2 EdgeGetSensorsV2 EdgeGetHandleV2 EdgeGetV2 EdgeDeleteV2 EdgeGetUpgradesV2 EdgeInfoGetV2 GetDataPipelineContainers
type EdgeIDParams struct {
	// ID for the edge
	// in: path
	// required: true
	EdgeID string `json:"edgeId"`
}

// EdgeDeviceIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter is an edge device id
// Typically used for edge device entity or per edge device entities
// swagger:parameters EdgeDeviceGetHandle EdgeDeviceGet EdgeDeviceDelete EdgeDeviceInfoGet EdgeDeviceGet EdgeDeviceDelete EdgeDeviceInfoGet
type EdgeDeviceParams struct {
	// ID for the edge device
	// in: path
	// required: true
	EdgeDeviceID string `json:"edgeDeviceId"`
}

// EdgeClusterParams parameter model
//
// Similar to IDParams, but to call out the id parameter is an edge device id
// Typically used for edge device entity or per edge device entities
// swagger:parameters EdgeClusterGetHandle EdgeClusterGet EdgeClusterDelete EdgeClusterInfoGet EdgeClusterGetHandle EdgeClusterGet EdgeClusterDelete ClusterGetEdgeDevices ClusterGetEdgeDevicesInfo
type EdgeClusterParams struct {
	// ID for the edge cluster
	// in: path
	// required: true
	EdgeClusterID string `json:"edgeClusterId"`
}

// ServiceDomainParams parameter model
//
// Similar to IDParams, but to call out the id parameter is service domain id
// Typically used for service domain entity or per service domain entities
// swagger:parameters ServiceDomainGetNodes ServiceDomainGetNodesInfo ServiceDomainGet ServiceDomainDelete ServiceDomainUpdate ServiceDomainGetHandle ServiceDomainInfoGet ServiceDomainInfoUpdate ServiceDomainGetFeatures ServiceDomainGetEffectiveProfile StorageProfileCreate SvcDomainGetStorageProfiles StorageProfileUpdate K8sDashboardGetAdminToken K8sDashboardGetViewonlyToken K8sDashboardGetUserToken K8sDashboardGetAdminKubeConfig K8sDashboardGetUserKubeConfig K8sDashboardGetViewonlyKubeConfig K8sDashboardGetViewonlyUsers K8sDashboardAddViewonlyUsers K8sDashboardRemoveViewonlyUsers
type ServiceDomainParams struct {
	// ID for the service domain
	// in: path
	// required: true
	SvcDomainID string `json:"svcDomainId"`
}

// BatchIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter for a batch
// Typically used for software update
// swagger:parameters SoftwareDownloadBatchGet SoftwareDownloadServiceDomainList SoftwareDownloadUpdate SoftwareDownloadStateUpdate SoftwareUpgradeBatchGet SoftwareUpgradeServiceDomainList SoftwareUpgradeUpdate SoftwareUpgradeStateUpdate
type BatchIDParams struct {
	// ID for the batch
	// in: path
	// required: true
	BatchID string `json:"batchId"`
}

// ReleaseParams parameter model
//
// Similar to IDParams, but to call out the id parameter for a release version
// Typically used for software update
// swagger:parameters SoftwareDownloadedServiceDomainList
type ReleaseParams struct {
	// release
	// in: path
	// required: true
	Release string `json:"release"`
}

// NodeParams parameter model
//
// Similar to IDParams, but to call out the id parameter is a node id
// Typically used for node entity or per node entities
// swagger:parameters NodeGet NodeDelete NodeUpdate NodeInfoGet NodeInfoUpdate
type NodeParams struct {
	// ID for the node
	// in: path
	// required: true
	NodeID string `json:"nodeId"`
}

// ProjectIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter is a project id
// This is used for operations that want the project ID of an entity in the path
// Typically used for project entity or per project entities
// swagger:parameters ProjectGet ProjectGetDataStreams ProjectGetApplications ProjectGetScriptRuntimes ProjectGetScripts ProjectGetDockerProfiles ProjectGetContainerRegistries ProjectGetCloudCreds ProjectGetUsers ProjectGetEdges ProjectGetEdgesInfo ProjectGetDatasources ProjectGetV2 ProjectGetDataPipelines ProjectGetApplicationsV2 ProjectGetRuntimeEnvironments ProjectGetFunctions ProjectGetDockerProfilesV2 ProjectGetContainerRegistriesV2 ProjectGetCloudProfiles ProjectGetUsersV2 ProjectGetEdgesV2 ProjectGetEdgesInfoV2 ProjectGetDatasourcesV2 ProjectGetMLModels ProjectGetEdgeDevices ProjectGetEdgeClusters ProjectGetEdgeDevicesInfo ProjectGetServiceDomains ProjectGetNodes ProjectGetNodesInfo ProjectGetServiceDomainsInfo
type ProjectIDParams struct {
	// ID for the project
	// in: path
	// required: true
	ID string `json:"projectId"`
}

// ServiceClassIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter for a release version
// Typically used for software update
// swagger:parameters ServiceClassUpdate ServiceClassGet ServiceClassDelete
type ServiceClassIDParams struct {
	// Service Class ID
	// in: path
	// required: true
	SvcClassID string `json:"svcClassId"`
}

// ServiceInstanceIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter for a release version
// Typically used for software update
// swagger:parameters ServiceInstanceUpdate ServiceInstanceGet ServiceInstanceDelete ServiceInstanceStatusList
type ServiceInstanceIDParams struct {
	// Service Instance ID
	// in: path
	// required: true
	SvcInstanceID string `json:"svcInstanceId"`
}

// ServiceBindingIDParams parameter model
//
// Similar to IDParams, but to call out the id parameter for a release version
// Typically used for software update
// swagger:parameters ServiceBindingGet ServiceBindingDelete ServiceBindingStatusList
type ServiceBindingIDParams struct {
	// Service Binding ID
	// in: path
	// required: true
	SvcBindingID string `json:"svcBindingId"`
}

// swagger:parameters MLModelVersionDelete MLModelVersionURLGet
type ModelVersionParams struct {
	// Model version, a positive integer.
	//
	// in: path
	// required: true
	ModelVersion int `json:"model_version"`
}

// swagger:parameters MLModelVersionUpdate
type ModelVersionUpdateParams struct {
	// Model version, a positive integer.
	//
	// in: path
	// required: true
	ModelVersion int `json:"model_version"`
	// Model version description.
	//
	// in: query
	// required: false
	Description string `json:"description"`
}

// ScriptParam - Spec for a script parameter
// script is a legacy term for function
type ScriptParam struct {
	// Name of the parameter
	// required: true
	Name string `json:"name"`
	// Type of the parameter
	// required: true
	Type string `json:"type"`
}

// ScriptParamValue - Instance of a script parameter value
type ScriptParamValue struct {
	// required: true
	ScriptParam
	// Value of the parameter
	// required: true
	Value string `json:"value"`
}

// TransformationArgs - ID and args info for use of  transformation in DataStream.
type TransformationArgs struct {
	// ID for the transformation
	// required: true
	TransformationID string `json:"transformationId"`
	// Array of script param values for the transformation
	// required: true
	Args []ScriptParamValue `json:"args"`
}

// swagger:model DeleteRequest
type DeleteRequest struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	ID string `json:"id"`
}

// swagger:model ResponseBase
type ResponseBase struct {
	// required: true
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message,omitempty"`
}

// Ok
// swagger:response ResponseBaseWrapper
type ResponseBaseWrapper struct {
	// in: body
	// required: true
	Payload *ResponseBase
}

// generic API error response
// swagger:response APIError
type APIError struct {
	// in: body
	// required: true
	Payload *APIErrorPayload
}

// The error message
// swagger:model APIErrorPayload
type APIErrorPayload struct {
	// HTTP status code for the response
	// required: true
	StatusCode int `json:"statusCode"`
	// Karbon Platform Services API error code
	// required: true
	ErrorCode int `json:"errorCode"`
	// Error message
	// required: true
	Message string `json:"message"`
}

// Login failed
// swagger:response
type LoginFailedError struct {
	APIError
}

type NotificationTopics string

type IDObj struct {
	ID string `json:"id" db:"id"`
}
type RuntimeIDObj struct {
	TenantID  string `json:"tenantId" db:"tenant_id"`
	RuntimeID string `json:"runtimeId" db:"runtime_id"`
}

// MarshalEqual - check if two pointers have equal value
func MarshalEqual(p1 interface{}, p2 interface{}) bool {
	if p1 == p2 {
		return true
	}
	if p1 == nil || p2 == nil {
		return false
	}
	d1, _ := json.Marshal(p1)
	d2, _ := json.Marshal(p2)
	return reflect.DeepEqual(d1, d2)
}

// GetEdgeID - returns *EdgeID in embedded EdgeBaseModel
// i should be a struct type which embeds
// EdgeModelBase struct
// e.g., DataSource or Sensor
func GetEdgeID(i interface{}) *string {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Struct {
		for j := 0; j < v.NumField(); j++ {
			field := v.Field(j)
			fieldName := v.Type().Field(j).Name
			if fieldName == "EdgeBaseModel" {
				if field.Kind() == reflect.Struct {
					for k := 0; k < field.NumField(); k++ {
						f := field.Field(k)
						fn := field.Type().Field(k).Name
						if fn == "EdgeID" {
							if f.CanInterface() && f.Kind() == reflect.String {
								s := f.Interface().(string)
								return &s
							}
						}
					}
				}
			}
		}
	}
	// Service domain is the cluster
	if clusterEntity, ok := i.(ClusterEntity); ok {
		clusterID := clusterEntity.GetClusterID()
		return &clusterID
	}

	if svcDomain, ok := i.(ServiceDomain); ok {
		return &svcDomain.ID
	}

	if edge, ok := i.(Edge); ok {
		return &edge.ID
	}
	return nil
}

type ScopedEntity struct {
	EdgeIDs []string
	Doc     interface{}
}

type IdentifiableEntity interface {
	GetID() string
}

type ProjectScopedEntity interface {
	IdentifiableEntity
	GetProjectID() string
}

type StatefulEntity interface {
	IdentifiableEntity
	GetEntityState() EntityState
}

type ClusterEntity interface {
	IdentifiableEntity
	GetClusterID() string
}

func (e BaseModel) GetID() string {
	return e.ID
}
func (e BaseModelDBO) GetID() string {
	return e.ID
}

func (e ClusterEntityModel) GetClusterID() string {
	return e.ClusterID
}

func (e ClusterEntityModelDBO) GetClusterID() string {
	return e.ClusterID
}

// ServiceDomainEntityModel is a cluster entity
func (e ServiceDomainEntityModel) GetClusterID() string {
	return e.SvcDomainID
}

// ServiceDomainEntityModelDBO is a cluster entity
func (e ServiceDomainEntityModelDBO) GetClusterID() string {
	return e.SvcDomainID
}

// PageQueryParam carries the pagination information
// swagger:parameters QueryEventsV2 ApplicationStatusListV2 ApplicationStatusGetV2 MLModelStatusList MLModelStatusGet
// in: query
type PageQueryParam struct {
	// 0-based index of the page to fetch results.
	// in: query
	// required: false
	PageIndex int `json:"pageIndex"`
	// Item count of each page.
	// in: query
	// required: false
	PageSize int `json:"pageSize"`
}

func (pageQueryParam *PageQueryParam) GetPageIndex() int {
	return pageQueryParam.PageIndex
}

func (pageQueryParam *PageQueryParam) GetPageSize() int {
	return pageQueryParam.PageSize
}

// MLModelVersionCreateParam carries the model version and description
// swagger:parameters MLModelVersionCreate
// in: query
type MLModelVersionCreateParam struct {
	// Model version, a positive integer.
	//
	// in: query
	// required: true
	ModelVersion int `json:"model_version"`
	// Model version description.
	//
	// in: query
	// required: false
	Description string `json:"description"`
}

// MLModelVersionGetURLParam carries the expiration duration
// swagger:parameters MLModelVersionURLGet
// in: query
type MLModelVersionGetURLParam struct {
	// Model URL expiration duration in minutes.
	//
	// in: query
	// required: false
	ExpirationDuration int `json:"expiration_duration"`
}

// EntitiesQueryParam carries the common query parameters for the GET endpoints
// swagger:parameters ApplicationListV2 ProjectGetApplicationsV2 CategoryListV2 CloudProfileList ProjectGetCloudProfiles ContainerRegistryListV2 ProjectGetContainerRegistriesV2 ProjectGetContainerRegistriesV2 DataSourceListV2 ProjectGetDatasourcesV2 EdgeGetDatasourcesV2 DataPipelineList ProjectGetDataPipelines EdgeListV2 ProjectGetEdgesV2 ProjectGetEdgesInfoV2 EdgeUpgradeListV2 EdgeInfoListV2 ProjectListV2 FunctionList ProjectGetFunctions RuntimeEnvironmentList ProjectGetRuntimeEnvironments SensorListV2 EdgeGetSensorsV2 UserListV2 ProjectGetUsersV2 MLModelList ProjectGetMLModels LogEntriesListV2 EdgeLogEntriesListV2 EdgeLogEntriesGetV2 ApplicationLogEntriesListV2 ApplicationLogEntriesGetV2 ProjectGetEdgeDevicesInfo EdgeClusterList ProjectGetEdgeClusters ClusterGetEdgeDevices ClusterGetEdgeDevicesInfo EdgeDeviceList ProjectGetEdgeDevices EdgeDeviceInfoList ProjectGetEdgeDevicesInfo  ServiceDomainList ProjectGetServiceDomains ServiceDomainGetNodes ServiceDomainGetNodesInfo NodeList ProjectGetNodes NodeInfoList ProjectGetNodesInfo ServiceDomainInfoList ProjectGetServiceDomainsInfo SoftwareUpdateReleaseList SoftwareDownloadBatchList SoftwareDownloadServiceDomainList SoftwareUpgradeBatchList SoftwareUpgradeServiceDomainList LogCollectorsList ServiceClassList ServiceInstanceList ServiceInstanceStatusList ServiceBindingList ServiceBindingStatusList HTTPServiceProxyList KubernetesClustersList DataDriverClassList DataDriverInstancesList DataDriverStreamList DataDriverConfigList
// in: query
type EntitiesQueryParam struct {
	// in: query
	// required: false
	PageQueryParam
	// Specify result order. Zero or more entries with format: &ltkey> [desc]
	// where orderByKeys lists allowed keys in each response.
	//
	// in: query
	// required: false
	OrderBy []string `json:"orderBy"`
	// Specify result filter. Format is similar to a SQL WHERE clause. For example,
	// to filter object by name with prefix foo, use: name LIKE 'foo%'.
	// Supported filter keys are the same as order by keys.
	// in: query
	// required: false
	Filter string `json:"filter"`
}

// KubernetesClusterBaseModel is the common base for all per kubernetes cluster objects
type KubernetesClusterBaseModel struct {
	// required: true
	BaseModel
	// ID of the kubernetes cluster this entity belongs to
	// required: true
	KubernetesClusterID string `json:"kubernetesClusterID" db:"kubernetes_cluster_id" validate:"range=1:36"`
}

func (param *EntitiesQueryParam) GetFilter() string {
	if param == nil {
		return ""
	}
	return param.Filter
}

func (param *EntitiesQueryParam) GetOrderBy() []string {
	if param == nil {
		return nil
	}
	return param.OrderBy
}

func (param *EntitiesQueryParam) GetPageIndex() int {
	if param == nil {
		return 0
	}
	if param.PageIndex < 0 {
		return 0
	}
	return param.PageIndex
}

func (param *EntitiesQueryParam) GetPageSize() int {
	if param == nil {
		return base.MaxRowsLimit
	}
	if param.PageSize <= 0 {
		return base.MaxRowsLimit
	}
	return param.PageSize
}

// EntitiesQueryParamV1 carries the common query parameters for the GET endpoints
// swagger:parameters ApplicationList ProjectGetApplications CategoryList CloudCredsList ProjectGetCloudCreds ContainerRegistryList ProjectGetContainerRegistries ProjectGetContainerRegistries DataSourceList ProjectGetDatasources EdgeGetDatasources DataStreamList ProjectGetDataStreams EdgeList ProjectGetEdges ProjectGetEdgesInfo EdgeUpgradeList EdgeInfoList ProjectList ScriptList ProjectGetScripts ScriptRuntimeList ProjectGetScriptRuntimes SensorList EdgeGetSensors UserList ProjectGetUsers SvcDomainGetStorageProfiles
// in: query
type EntitiesQueryParamV1 struct {
	// Specify result order. Zero or more entries with format: &ltkey> [desc]
	// where orderByKeys lists allowed keys in each response.
	//
	// in: query
	// required: false
	OrderBy []string `json:"orderBy"`
	// Specify result filter. Format is similar to a SQL WHERE clause. For example,
	// to filter object by name with prefix foo, use: name LIKE 'foo%'.
	// Supported filter keys are the same as order by keys.
	// in: query
	// required: false
	Filter string `json:"filter"`
}

func (param *EntitiesQueryParamV1) GetFilter() string {
	if param == nil {
		return ""
	}
	return param.Filter
}
func (param *EntitiesQueryParamV1) GetOrderBy() []string {
	if param == nil {
		return nil
	}
	return param.OrderBy
}

func (param *EntitiesQueryParamV1) GetPageIndex() int {
	return 0
}

func (param *EntitiesQueryParamV1) GetPageSize() int {
	return base.MaxRowsLimit
}

type EntityListResponsePayload struct {
	// 0-based index of the page to fetch results.
	// required: true
	PageIndex int `json:"pageIndex"`
	// Item count of each page.
	// required: true
	PageSize int `json:"pageSize"`
	// Count of all items matching the query.
	// required: true
	TotalCount int `json:"totalCount"`
	// Specify result order. Zero or more entries with format: &ltkey> [desc]
	// where orderByKeys lists allowed keys in each response.
	// required: false
	OrderBy string `json:"orderBy,omitempty"`
	// Keys that can be used in orderBy.
	// required: false
	OrderByKeys []string `json:"orderByKeys,omitempty"`
}

// AuditLogQueryParam carries the query parameters for the get all audit logs endpoint
// swagger:parameters AuditLogList AuditLogListV2
// in: query
type AuditLogQueryParam struct {
	// in: query
	// required: false
	EntitiesQueryParam
	// Start time for query. Format: yyyy-mm-dd hh:mm:ss, the hh:mm:ss part is optional.
	//
	// in: query
	// required: false
	Start string `json:"start"`
	// End time for query. Format: yyyy-mm-dd hh:mm:ss, the hh:mm:ss part is optional.
	//
	// in: query
	// required: false
	End string `json:"end"`
}

type PagedListResponsePayload struct {
	// 0-based index of the page to fetch results.
	// required: true
	PageIndex int `json:"pageIndex"`
	// Item count of each page.
	// required: true
	PageSize int `json:"pageSize"`
	// Count of all items matching the query.
	// required: true
	TotalCount int `json:"totalCount"`
}

// swagger:parameters LoginTokenV1 ShortLoginTokenV1
// in: header
type authorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// swagger:parameters LoginTokenV1
// in: body
type LoginTokenPayload struct {
	// in: body
	// required: true
	Info map[string]string
}

// GetAuditLogQueryParam extracts the AuditLogQueryParam from the HTTP request
func GetAuditLogQueryParam(req *http.Request) AuditLogQueryParam {
	param := AuditLogQueryParam{
		EntitiesQueryParam: EntitiesQueryParam{
			PageQueryParam: PageQueryParam{
				PageIndex: 0,
				PageSize:  100,
			},
		},
		Start: base.GetDateStart(),
		End:   base.GetDateEnd(),
	}
	if req != nil {
		query := req.URL.Query()
		pageSizeVals := query["pageSize"]
		if len(pageSizeVals) == 1 {
			pSize, err := strconv.Atoi(pageSizeVals[0])
			if err == nil {
				if pSize > 0 {
					param.PageSize = pSize
				}
			}
		}
		pageIndexVals := query["pageIndex"]
		if len(pageIndexVals) == 1 {
			pIndex, err := strconv.Atoi(pageIndexVals[0])
			if err == nil {
				if pIndex >= 0 {
					param.PageIndex = pIndex
				}
			}
		}
		startVals := query["start"]
		if len(startVals) == 1 {
			param.Start = startVals[0]
		}
		endVals := query["end"]
		if len(endVals) == 1 {
			param.End = endVals[0]
		}
		orderByVals := query["orderBy"]
		if len(orderByVals) != 0 {
			param.OrderBy = orderByVals
		}
		filterVals := query["filter"]
		if len(filterVals) == 1 {
			param.Filter = strings.TrimSpace(filterVals[0])
		}
	}
	return param
}

// GetEntitiesQueryParam extracts the EntitiesQueryParam from the HTTP request
func GetEntitiesQueryParam(req *http.Request) *EntitiesQueryParam {
	param := EntitiesQueryParam{
		PageQueryParam: PageQueryParam{
			PageIndex: 0,
			PageSize:  base.MaxRowsLimit,
		},
	}
	if req != nil {
		query := req.URL.Query()
		pageSizeVals := query["pageSize"]
		if len(pageSizeVals) == 1 {
			pSize, err := strconv.Atoi(pageSizeVals[0])
			if err == nil {
				param.PageSize = pSize
			}
		}
		pageIndexVals := query["pageIndex"]
		if len(pageIndexVals) == 1 {
			pIndex, err := strconv.Atoi(pageIndexVals[0])
			if err == nil {
				param.PageIndex = pIndex
			}
		}
		orderByVals := query["orderBy"]
		if len(orderByVals) != 0 {
			param.OrderBy = orderByVals
		}
		filterVals := query["filter"]
		if len(filterVals) == 1 {
			param.Filter = strings.TrimSpace(filterVals[0])
		}
	}
	return &param
}

// GetEntitiesQueryParamV1 extracts the EntitiesQueryParamV1 from the HTTP request
func GetEntitiesQueryParamV1(req *http.Request) *EntitiesQueryParamV1 {
	param := EntitiesQueryParamV1{}
	if req != nil {
		query := req.URL.Query()
		orderByVals := query["orderBy"]
		if len(orderByVals) != 0 {
			param.OrderBy = orderByVals
		}
		filterVals := query["filter"]
		if len(filterVals) == 1 {
			param.Filter = strings.TrimSpace(filterVals[0])
		}
	}
	return &param
}

func GetMLModelVersionCreateParam(req *http.Request) MLModelVersionCreateParam {
	param := MLModelVersionCreateParam{}
	if req != nil {
		query := req.URL.Query()
		modelVersionVals := query["model_version"]
		if len(modelVersionVals) == 1 {
			v, err := strconv.Atoi(modelVersionVals[0])
			if err == nil {
				param.ModelVersion = v
			}
		}
		modelDescriptionVals := query["description"]
		if len(modelDescriptionVals) == 1 {
			param.Description = modelDescriptionVals[0]
		}
	}
	return param
}

func GetMLModelVersionGetURLParam(req *http.Request) MLModelVersionGetURLParam {
	param := MLModelVersionGetURLParam{}
	if req != nil {
		query := req.URL.Query()
		expDurationVals := query["expiration_duration"]
		if len(expDurationVals) == 1 {
			mins, err := strconv.Atoi(expDurationVals[0])
			if err == nil {
				param.ExpirationDuration = mins
			}
		}
	}
	return param
}
func ToCreateV2(createFn func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error)) func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error) {
	return func(context context.Context, i interface{}, callback func(context.Context, interface{}) error) (interface{}, error) {
		r, err := createFn(context, i, callback)
		rd2 := CreateDocumentResponseV2{}
		if err == nil {
			rd := r.(CreateDocumentResponse)
			rd2.ID = rd.ID
		}
		return rd2, err
	}
}
func ToUpdateV2(updateFn func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error)) func(context.Context, interface{}, func(context.Context, interface{}) error) (interface{}, error) {
	return func(context context.Context, i interface{}, callback func(context.Context, interface{}) error) (interface{}, error) {
		r, err := updateFn(context, i, callback)
		rd2 := UpdateDocumentResponseV2{}
		if err == nil {
			rd := r.(UpdateDocumentResponse)
			rd2.ID = rd.ID
		}
		return rd2, err
	}
}
func ToDeleteV2(deleteFn func(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)) func(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	return func(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
		r, err := deleteFn(context, id, callback)
		rd2 := DeleteDocumentResponseV2{}
		if err == nil {
			rd := r.(DeleteDocumentResponse)
			rd2.ID = rd.ID
		}
		return rd2, err
	}
}

// Event definitions
type EntityCRUDEvent struct {
	ID       string
	TenantID string
	EntityID string
	// See getMessageWithPrefix in router.go
	Message    string
	Properties map[string]interface{}
}

func (event *EntityCRUDEvent) IsAsync() bool {
	return true
}

func (event *EntityCRUDEvent) EventName() string {
	return EntityCRUDEventName
}

func (event *EntityCRUDEvent) GetID() string {
	return event.ID
}

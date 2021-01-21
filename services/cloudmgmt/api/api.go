package api

import (
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	tenantpool "cloudservices/tenantpool/model"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
)

// This is the begining of the record
const (
	AdminRole                  = "admin"
	EdgeRole                   = "edge"
	SpecialRoleKey             = "specialRole"
	ProjectsKey                = "projects"
	EdgeTableName              = "edge_model"
	EdgeClusterTableName       = "edge_cluster_model"
	UserTableName              = "user_model"
	CloudProfileTableName      = "cloud_creds_model"
	ContainerRegistryTableName = "docker_profile_model"
	ProjectTableName           = "project_model"
	DefaultPageSize            = 100
)

// ReK8sName - regexp for K8s name - minus '.' since AWS does not allow it
var ReK8sName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

type IDDBO struct {
	ID string `json:"id" db:"id"`
}

// ObjectModelAPI captures all object model APIs
type ObjectModelAPI interface {
	Close() error

	SelectAllCategories(context context.Context, queryParams *model.EntitiesQueryParamV1) ([]model.Category, error)
	SelectAllCategoriesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllCategoriesWV2(context context.Context, w io.Writer, r *http.Request) error
	GetCategory(context context.Context, id string) (model.Category, error)
	GetCategoryW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateCategory(context context.Context, doc interface{} /* *model.Category */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateCategoryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateCategoryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateCategory(context context.Context, doc interface{} /* *model.Category */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateCategoryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateCategoryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteCategory(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteCategoryW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteCategoryWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	GetCategoryNamesByIDs(ctx context.Context, categoryIDs []string) (map[string]string, error)
	SelectAllCategoriesUsageInfo(context context.Context) ([]model.CategoryUsageInfo, error)
	SelectAllCategoriesUsageInfoW(context context.Context, w io.Writer, req *http.Request) error
	GetCategoryDetailUsageInfo(ctx context.Context, categoryID string) (model.CategoryDetailUsageInfo, error)
	GetCategoryDetailUsageInfoW(ctx context.Context, categoryID string, w io.Writer, req *http.Request) error

	SelectAllCloudCreds(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.CloudCreds, error)
	SelectAllCloudCredsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllCloudCredsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllCloudCredsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.CloudCreds, error)
	SelectAllCloudCredsForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllCloudCredsForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetCloudCreds(context context.Context, id string) (model.CloudCreds, error)
	GetCloudCredsW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateCloudCreds(context context.Context, doc interface{} /* *model.CloudCreds */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateCloudCredsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateCloudCredsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateCloudCreds(context context.Context, doc interface{} /* *model.CloudCreds */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateCloudCredsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateCloudCredsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteCloudCreds(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteCloudCredsW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteCloudCredsWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	EncryptAllCloudCreds(context context.Context) error
	EncryptAllCloudCredsW(context context.Context, r io.Reader) error
	GetAllCloudCredsProjects(context context.Context, cloudCredsID string) ([]string, error)
	GetAllCloudCredsEdges(context context.Context, cloudCredsID string) ([]string, error)

	GetAggregate(context context.Context, tableName string, fieldName string, w io.Writer) error

	CreateCertificates(context context.Context, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateCertificatesW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	SelectAllDataSources(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error)
	SelectAllDataSourcesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDataSourcesWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDataSourcesForEdge(context context.Context, edgeID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error)
	SelectAllDataSourcesForEdgeW(context context.Context, edgeID string, w io.Writer, r *http.Request) error
	SelectAllDataSourcesForEdgeWV2(context context.Context, edgeID string, w io.Writer, r *http.Request) error
	SelectAllDataSourcesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataSource, error)
	SelectAllDataSourcesForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllDataSourcesForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetDataSource(context context.Context, id string) (model.DataSource, error)
	GetDataSourceW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetDataSourceWV2(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDataSource(context context.Context, doc interface{} /* *model.DataSource */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataSourceV2(context context.Context, doc interface{} /* *model.DataSourceV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataSourceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateDataSourceWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataSource(context context.Context, doc interface{} /* *model.DataSource */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataSourceV2(context context.Context, doc interface{} /* *model.DataSourceV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataSourceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataSourceWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDataSource(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDataSourceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteDataSourceWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	GetDataSourceEdgeID(context context.Context, id string) (string, error)
	CreateDataSourceArtifact(context context.Context, doc interface{} /* *model.DataSourceArtifact */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataSourceArtifactWV2(authContextIA context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	GetDataSourceArtifact(context context.Context, dataSourceID string) (model.DataSourceArtifact, error)
	GetDataSourceArtifactWV2(context context.Context, id string, w io.Writer, r *http.Request) error

	SelectAllDataStreams(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataStream, error)
	SelectAllDataStreamsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDataStreamsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDataStreamsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.DataStream, error)
	SelectAllDataStreamsForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllDataStreamsForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetDataStream(context context.Context, id string) (model.DataStream, error)
	GetDataStreamW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDataStream(context context.Context, doc interface{} /* *model.DataStream */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateDataStreamWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataStream(context context.Context, doc interface{} /* *model.DataStream */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataStreamWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDataStream(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDataStreamW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteDataStreamWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	GetDataStreamIDs(context context.Context, scriptIDs []string) ([]string, error)
	GetDataStreamNames(context context.Context, dsIDs []string) ([]string, error)
	GetDataPipelineContainersW(context context.Context, dataPipelineID string, edgeID string, w io.Writer, callback func(context.Context, interface{}) (string, error)) error

	SelectAllEdges(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Edge, error)
	SelectAllEdgesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgesWV2(context context.Context, w io.Writer, r *http.Request) error

	SelectAllEdgesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Edge, error)
	SelectAllEdgesForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllEdgesForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error

	GetEdge(context context.Context, id string) (model.Edge, error)
	GetEdgeW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetEdgeWV2(context context.Context, id string, w io.Writer, r *http.Request) error

	GetEdgeBySerialNumber(context context.Context, serialNumber string) (model.EdgeDeviceWithClusterInfo, error)
	GetEdgeBySerialNumberW(context context.Context, w io.Writer, req *http.Request) error

	CreateEdge(context context.Context, doc interface{} /* *model.Edge */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeV2(context context.Context, doc interface{} /* *model.EdgeV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateEdgeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	UpdateEdge(context context.Context, doc interface{} /* *model.Edge */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateEdgeV2(context context.Context, doc interface{} /* *model.EdgeV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateEdgeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateEdgeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	DeleteEdge(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteEdgeW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteEdgeWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	GetEdgeHandle(context context.Context, edgeID string, payload model.GetHandlePayload) (model.EdgeCert, error)
	GetEdgeHandleW(context context.Context, edgeID string, w io.Writer, req *http.Request) error
	GetEdgeProjects(ctx context.Context, edgeID string) ([]model.Project, error)
	GetEdgeProjectRoles(context context.Context, edgeID string) ([]model.ProjectRole, error)

	// Edge Devices
	SelectAllEdgeDevicesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgeDevicesForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetEdgeDevice(context context.Context, id string) (model.EdgeDevice, error)
	GetEdgeDeviceW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetEdgeDeviceBySerialNumber(context context.Context, serialNumber string) (model.EdgeDeviceWithClusterInfo, error)
	GetEdgeDeviceBySerialNumberW(context context.Context, w io.Writer, req *http.Request) error

	CreateEdgeDevice(context context.Context, doc interface{} /* *model.EdgeDevice */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeDeviceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateEdgeDevice(context context.Context, doc interface{} /* *model.EdgeDevice */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateEdgeDeviceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteEdgeDevice(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteEdgeDeviceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	UpdateEdgeDeviceOnboarded(context context.Context, id string, sshPublicKey string) error
	UpdateEdgeDeviceOnboardedW(context context.Context, w io.Writer, req *http.Request) error

	// Nodes
	SelectAllNodes(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Node, error)
	SelectAllNodesW(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllNodesForProject(ctx context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Node, error)
	SelectAllNodesForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error

	GetNode(ctx context.Context, id string) (model.Node, error)
	GetNodeW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	GetNodeBySerialNumber(ctx context.Context, serialNumber string) (model.NodeWithClusterInfo, error)
	GetNodeBySerialNumberW(ctx context.Context, w io.Writer, req *http.Request) error

	CreateNode(ctx context.Context, i interface{} /* *model.Node */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateNodeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	UpdateNode(ctx context.Context, i interface{} /* *model.Node*/, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateNodeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateNodeOnboarded(ctx context.Context, doc *model.NodeOnboardInfo) error
	UpdateNodeOnboardedW(ctx context.Context, w io.Writer, req *http.Request) error

	DeleteNode(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteNodeW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Edge Cluster
	SelectAllEdgeClusters(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeCluster, error)
	SelectAllEdgeClustersW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgeClustersForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllEdgeDevicesForClusterW(context context.Context, clusterID string, w io.Writer, r *http.Request) error
	SelectAllEdgeDevicesInfoForClusterW(context context.Context, clusterID string, w io.Writer, r *http.Request) error

	GetEdgeCluster(context context.Context, id string) (model.EdgeCluster, error)
	GetEdgeClusterW(context context.Context, id string, w io.Writer, r *http.Request) error

	CreateEdgeCluster(context context.Context, doc interface{} /* *model.EdgeCluster */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeClusterW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	UpdateEdgeCluster(context context.Context, doc interface{} /* *model.EdgeCluster */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateEdgeClusterW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	DeleteEdgeCluster(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteEdgeClusterW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	GetEdgeClusterHandle(context context.Context, edgeClusterID string, payload model.GetHandlePayload) (model.EdgeCert, error)
	GetEdgeClusterHandleW(context context.Context, edgeClusterID string, w io.Writer, req *http.Request) error
	GetEdgeClusterProjects(ctx context.Context, edgeClusterID string) ([]model.Project, error)
	GetEdgeClusterProjectRoles(context context.Context, edgeClusterID string) ([]model.ProjectRole, error)
	SelectEdgeClusterIDLabels(context context.Context) ([]model.EdgeClusterIDLabels, error)
	SelectAllEdgeClusterIDs(context context.Context) ([]string, error)
	SelectConnectedEdgeClusterIDs(context context.Context) ([]string, error)

	// Service Domain
	SelectAllServiceDomains(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ServiceDomain, error)
	SelectAllServiceDomainsW(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllServiceDomainsForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error
	SelectAllNodesForServiceDomainW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error
	SelectAllNodeInfoForServiceDomainW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error

	GetServiceDomain(ctx context.Context, id string) (model.ServiceDomain, error)
	GetServiceDomainW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	GetServiceDomainEffectiveProfileW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	CreateServiceDomain(ctx context.Context, i interface{} /* *model.ServiceDomain */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateServiceDomainW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateServiceDomain(ctx context.Context, i interface{} /* *model.ServiceDomain*/, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateServiceDomainW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteServiceDomain(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteServiceDomainW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	GetServiceDomainHandle(ctx context.Context, svcDomainID string, payload model.GetHandlePayload) (model.EdgeCert, error)
	GetServiceDomainHandleW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error

	SelectServiceDomainIDLabels(ctx context.Context) ([]model.ServiceDomainIDLabels, error)
	SelectAllServiceDomainIDs(ctx context.Context) ([]string, error)
	SelectConnectedServiceDomainIDs(ctx context.Context) ([]string, error)

	// Service Domain Info
	SelectAllServiceDomainsInfoW(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllServiceDomainsInfoForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error

	GetServiceDomainInfo(ctx context.Context, id string) (model.ServiceDomainInfo, error)
	GetServiceDomainInfoW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	CreateServiceDomainInfo(ctx context.Context, i interface{} /* *model.ServiceDomainInfo */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateServiceDomainInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateServiceDomainInfo(ctx context.Context, i interface{} /* *model.ServiceDomainInfo */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateServiceDomainInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteServiceDomainInfo(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteServiceDomainInfoW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Service Domain Features
	GetFeaturesForServiceDomains(ctx context.Context, svcDomainIDs []string) (map[string]*model.Features, error)

	// EdgeCert
	SelectAllEdgeCerts(context context.Context) ([]model.EdgeCert, error)
	SelectAllEdgeCertsW(context context.Context, w io.Writer, r *http.Request) error
	GetEdgeCert(context context.Context, id string) (model.EdgeCert, error)
	GetEdgeCertW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetEdgeCertByEdgeID(context context.Context, edgeID string) (model.EdgeCert, error)
	CreateEdgeCert(context context.Context, doc interface{} /* *model.EdgeCert */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeCertW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateEdgeCertWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateEdgeCert(context context.Context, doc interface{} /* *model.EdgeCert */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateEdgeCertW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateEdgeCertWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteEdgeCert(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteEdgeCertW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteEdgeCertWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	SetEdgeCertLock(context context.Context, edgeID string, locked bool) error
	SetEdgeCertLockW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	// Edge Info
	SelectAllEdgesInfo(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeUsageInfo, error)
	SelectAllEdgesInfoW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgesInfoWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgesInfoForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeUsageInfo, error)
	SelectAllEdgesInfoForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error
	SelectAllEdgesInfoForProjectWV2(context context.Context, projectID string, w io.Writer, req *http.Request) error
	GetEdgeInfo(context context.Context, id string) (model.EdgeUsageInfo, error)
	GetEdgeInfoW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateEdgeInfo(context context.Context, doc interface{} /* *model.EdgeUsageInfo */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeInfoW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateEdgeInfoWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateEdgeInfo(context context.Context, doc interface{} /* *model.EdgeUsageInfo */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateEdgeInfoW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateEdgeInfoWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteEdgeInfo(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteEdgeInfoW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteEdgeInfoWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Edge Device Info
	SelectAllEdgeDevicesInfo(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeDeviceInfo, error)
	SelectAllEdgeDevicesInfoW(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllEdgeDevicesInfoWV2(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllEdgeDevicesInfoForProject(ctx context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.EdgeDeviceInfo, error)
	SelectAllEdgeDevicesInfoForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error
	SelectAllEdgeDevicesInfoForProjectWV2(ctx context.Context, projectID string, w io.Writer, req *http.Request) error
	GetEdgeDeviceInfo(ctx context.Context, id string) (model.EdgeDeviceInfo, error)
	GetEdgeDeviceInfoW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	CreateEdgeDeviceInfo(ctx context.Context, i interface{} /* *model.EdgeDeviceInfo */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateEdgeDeviceInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateEdgeDeviceInfoWV2(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteEdgeDeviceInfo(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteEdgeDeviceInfoW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteEdgeDeviceInfoWV2(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Node Info
	SelectAllNodesInfo(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.NodeInfo, error)
	SelectAllNodesInfoW(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllNodesInfoWV2(ctx context.Context, w io.Writer, req *http.Request) error
	SelectAllNodesInfoForProject(ctx context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.NodeInfo, error)
	SelectAllNodesInfoForProjectW(ctx context.Context, projectID string, w io.Writer, req *http.Request) error
	SelectAllNodesInfoForProjectWV2(ctx context.Context, projectID string, w io.Writer, req *http.Request) error
	GetNodeInfo(ctx context.Context, id string) (model.NodeInfo, error)
	GetNodeInfoW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	CreateNodeInfo(ctx context.Context, i interface{} /* *model.NodeInfo */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateNodeInfoW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateNodeInfoWV2(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteNodeInfo(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteNodeInfoWV2(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAffiliatedProjects(context context.Context) ([]model.Project, error)
	SelectAllProjects(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Project, error)
	SelectAllProjectsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllProjectsWV2(context context.Context, w io.Writer, r *http.Request) error
	GetProject(context context.Context, id string) (model.Project, error)
	GetProjectW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateProject(context context.Context, doc interface{} /* *model.Project */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateProjectW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateProjectWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateProject(context context.Context, doc interface{} /* *model.Project */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateProjectW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateProjectWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteProject(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteProjectW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteProjectWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	// Internal API, RBAC is not checked
	GetProjectEdges(context context.Context, param ProjectEdgeDBO) ([]string, error)
	// Internal API, RBAC is not checked
	GetProjectsEdges(context context.Context, projectIDs []string) ([]string, error)
	GetProjectName(context context.Context, projectID string) (string, error)
	SelectProjectDataStreamsUsingCloudCreds(context context.Context, tenantID string, projectID string, cloudCredsIDs []string) ([]string, error)
	GetProjectNamesByIDs(ctx context.Context, projectIDs []string) (map[string]string, error)

	SelectAllScripts(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Script, error)
	SelectAllScriptsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllScriptsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllScriptsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.Script, error)
	SelectAllScriptsForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllScriptsForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetScript(context context.Context, id string) (model.Script, error)
	GetScriptW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateScript(context context.Context, doc interface{} /* *model.Script */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateScriptW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateScriptWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateScript(context context.Context, doc interface{} /* *model.ScriptForceUpdate */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateScriptW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateScriptWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteScript(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteScriptW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteScriptWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	SelectScriptsByRuntimeID(context context.Context, runtimeID string) ([]string, error)

	SelectAllScriptRuntimes(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ScriptRuntime, error)
	SelectAllScriptRuntimesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllScriptRuntimesWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllScriptRuntimesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ScriptRuntime, error)
	SelectAllScriptRuntimesForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllScriptRuntimesForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetScriptRuntime(context context.Context, id string) (model.ScriptRuntime, error)
	GetScriptRuntimeW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateScriptRuntime(context context.Context, doc interface{} /* *model.ScriptRuntime */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateScriptRuntimeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateScriptRuntimeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateScriptRuntime(context context.Context, doc interface{} /* *model.ScriptRuntime */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateScriptRuntimeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateScriptRuntimeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteScriptRuntime(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteScriptRuntimeW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteScriptRuntimeWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAllSensors(context context.Context) ([]model.Sensor, error)
	SelectAllSensorsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllSensorsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllSensorsForEdge(context context.Context, edgeID string) ([]model.Sensor, error)
	SelectAllSensorsForEdgeW(context context.Context, edgeID string, w io.Writer, r *http.Request) error
	SelectAllSensorsForEdgeWV2(context context.Context, edgeID string, w io.Writer, r *http.Request) error
	GetSensor(context context.Context, id string) (model.Sensor, error)
	GetSensorW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateSensor(context context.Context, doc interface{} /* *model.Sensor */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateSensorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateSensorWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateSensor(context context.Context, doc interface{} /* *model.Sensor */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateSensorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateSensorWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteSensor(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteSensorW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteSensorWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAllTenants(context context.Context) ([]model.Tenant, error)
	SelectAllTenantsW(context context.Context, w io.Writer, r *http.Request) error
	GetTenant(context context.Context, id string) (model.Tenant, error)
	GetTenantW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetTenantSelfW(context context.Context, w io.Writer, r *http.Request) error
	// Internal only function needed by both product and test code.
	// Not exposed as an API because we do not allow direct access to tenant root CA.
	GetTenantRootCA(tenantID string) (string, error)
	CreateTenant(context context.Context, doc interface{} /* *model.Tenant */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateTenantW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateTenantWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateTenant(context context.Context, doc interface{} /* *model.Tenant */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateTenantW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateTenantWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteTenant(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteTenantW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteTenantWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Internal only function not exposed as REST API
	CreateBuiltinTenantObjects(ctx context.Context, tenantID string) error
	// Internal only function not exposed as REST API
	DeleteBuiltinTenantObjects(ctx context.Context, tenantID string) error

	GetTenantProps(ctx context.Context, id string) (model.TenantProps, error)
	GetTenantPropsW(context context.Context, id string, w io.Writer, r *http.Request) error
	UpdateTenantProps(ctx context.Context, i interface{} /* *model.TenantProps */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateTenantPropsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateTenantPropsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteTenantProps(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteTenantPropsW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteTenantPropsWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAllUsers(context context.Context) ([]model.User, error)
	SelectAllUsersW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllUsersWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllUsersForProject(context context.Context, projectID string) ([]model.User, error)
	SelectAllUsersForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllUsersForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetUser(context context.Context, id string) (model.User, error)
	GetUserW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetUserByEmail(context context.Context, email string) (model.User, error)
	CreateUser(context context.Context, doc interface{} /* *model.User */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateUserW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateUserWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateUser(context context.Context, doc interface{} /* *model.User */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateUserW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateUserWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteUser(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteUserW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteUserWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	GetUserProjectRoles(context context.Context, userID string) ([]model.ProjectRole, error)
	IsEmailAvailableW(context context.Context, w io.Writer, r *http.Request) error

	GetUserProps(ctx context.Context, id string) (model.UserProps, error)
	GetUserPropsW(context context.Context, id string, w io.Writer, r *http.Request) error
	UpdateUserProps(ctx context.Context, i interface{} /* *model.UserProps */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateUserPropsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateUserPropsWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteUserProps(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteUserPropsW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteUserPropsWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAllApplications(context context.Context) ([]model.Application, error)
	SelectAllApplicationsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllApplicationsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllApplicationsForProject(context context.Context, projectID string) ([]model.Application, error)
	SelectAllApplicationsForDataIfcEndpoint(context context.Context, dataIfcID string) ([]model.Application, error)
	SelectAllApplicationsForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllApplicationsForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetApplication(context context.Context, id string) (model.Application, error)
	GetApplicationW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetApplicationWV2(context context.Context, id string, w io.Writer, r *http.Request) error
	GetApplicationContainersW(context context.Context, applicationID string, edgeID string, w io.Writer, callback func(context.Context, interface{}) (string, error)) error
	CreateApplication(context context.Context, doc interface{} /* *model.Application */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateApplicationV2(context context.Context, doc interface{} /* *model.ApplicationV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateApplicationW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateApplicationWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateApplication(context context.Context, doc interface{} /* *model.Application */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateApplicationW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateApplicationWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteApplication(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteApplicationW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteApplicationWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteProjectApplicationsEdges(context context.Context, projectID string, edgeIDs []string) error
	DeleteProjectApplicationsEdgeSelectors(context context.Context, projectID string) error
	CreateHelmAppW(context context.Context, unused string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error
	CreateHelmValuesW(context context.Context, chartID string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error
	GetHelmAppYaml(context context.Context, chartID string, w io.Writer, req *http.Request) error

	SelectAllApplicationsStatus(context context.Context, includeDisconnectedEdges bool) ([]model.ApplicationStatus, error)
	SelectAllApplicationsStatusW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllApplicationsStatusWV2(context context.Context, w io.Writer, r *http.Request) error
	GetApplicationStatus(context context.Context, applicationID string) ([]model.ApplicationStatus, error)
	GetApplicationStatusW(context context.Context, applicationID string, w io.Writer, req *http.Request) error
	GetApplicationStatusWV2(context context.Context, applicationID string, w io.Writer, req *http.Request) error
	CreateApplicationStatus(context context.Context, doc interface{} /* *model.ApplicationStatus */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateApplicationStatusW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateApplicationStatusWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteApplicationStatus(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteApplicationStatusW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteApplicationStatusWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAllDockerProfiles(context context.Context) ([]model.DockerProfile, error)
	SelectAllDockerProfilesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDockerProfilesWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDockerProfilesForProject(context context.Context, projectID string) ([]model.DockerProfile, error)
	SelectAllDockerProfilesForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllDockerProfilesForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetDockerProfile(context context.Context, id string) (model.DockerProfile, error)
	GetDockerProfileW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDockerProfile(context context.Context, doc interface{} /* *model.DockerProfile */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDockerProfileW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateDockerProfileWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDockerProfile(context context.Context, doc interface{} /* *model.DockerProfile */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDockerProfileW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDockerProfileWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDockerProfile(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDockerProfileW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteDockerProfileWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	EncryptAllDockerProfiles(context context.Context) error
	EncryptAllDockerProfilesW(context context.Context, r io.Reader) error
	GetAllDockerProfileProjects(context context.Context, dockerProfileID string) ([]string, error)
	GetAllDockerProfileEdges(context context.Context, dockerProfileID string) ([]string, error)
	SelectDockerProfilesByIDs(context context.Context, dockerProfileIDs []string) ([]model.DockerProfile, error)

	SelectAllContainerRegistries(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ContainerRegistry, error)
	SelectAllContainerRegistriesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllContainerRegistriesWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllContainerRegistriesForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ContainerRegistry, error)
	SelectAllContainerRegistriesForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	SelectAllContainerRegistriesForProjectWV2(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetContainerRegistry(context context.Context, id string) (model.ContainerRegistry, error)
	GetContainerRegistryW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetContainerRegistryWV2(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateContainerRegistry(context context.Context, doc interface{} /* *model.ContainerRegistry */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateContainerRegistryV2(context context.Context, doc interface{} /* *model.ContainerRegistryV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateContainerRegistryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateContainerRegistryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateContainerRegistry(context context.Context, doc interface{} /* *model.ContainerRegistry */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateContainerRegistryW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateContainerRegistryWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateContainerRegistryV2(context context.Context, i interface{} /* *model.ContainerRegistryV2 */, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteContainerRegistry(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteContainerRegistryW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	DeleteContainerRegistryWV2(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	EncryptAllContainerRegistries(context context.Context) error
	EncryptAllContainerRegistriesW(context context.Context, r io.Reader) error
	SelectContainerRegistriesByIDs(context context.Context, containerRegistryIDs []string) ([]model.ContainerRegistry, error)

	SelectAllLogs(context context.Context, edgeID string, tags []model.LogTag, entitiesQueryParam *model.EntitiesQueryParam) ([]model.LogEntry, error)
	SelectAllLogsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllLogsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgeLogsWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectAllApplicationLogsWV2(context context.Context, w io.Writer, r *http.Request) error
	GetEdgeLogsWV2(context context.Context, id string, w io.Writer, r *http.Request) error
	GetApplicationLogsWV2(context context.Context, id string, w io.Writer, r *http.Request) error
	DeleteLogEntry(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteLogEntryW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	RequestLogDownload(context context.Context, payload model.RequestLogDownloadPayload) (string, error)
	RequestLogDownloadW(context context.Context, w io.Writer, r io.Reader) error
	UploadLog(context context.Context, payload model.LogUploadPayload) error
	UploadLogW(context context.Context, r io.Reader) error
	RequestLogUpload(context context.Context, payload model.RequestLogUploadPayload, callback func(context.Context, interface{}) error) ([]model.LogUploadPayload, error)
	RequestLogUploadW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	RequestLogStreamEndpoints(context context.Context, payload model.LogStream, callback func(context.Context, interface{}) error) (model.LogStreamEndpointsResponsePayload, error)
	RequestLogStreamEndpointsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UploadLogComplete(context context.Context, payload model.LogUploadCompletePayload) error
	UploadLogCompleteW(context context.Context, r io.Reader) error
	ScheduleTimeOutPendingLogsJob(context context.Context, delay time.Duration, timeout time.Duration) error

	QueryEvents(context context.Context, filter model.EventFilter) ([]model.Event, error)
	QueryEventsW(context context.Context, w io.Writer, r *http.Request) error
	UpsertEvents(context context.Context, docs model.EventUpsertRequest, callback func(context.Context, interface{}) error) ([]model.Event, error)
	UpsertEventsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	/* Software update APIs */
	SelectAllEdgeUpgrades(context context.Context) ([]model.EdgeUpgradeCore, error)
	SelectAllEdgeUpgradesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllEdgeUpgradesWV2(context context.Context, w io.Writer, r *http.Request) error
	SelectEdgeUpgradesByEdgeID(context context.Context, id string) ([]model.EdgeUpgradeCore, error)
	SelectEdgeUpgradesByEdgeIDW(context context.Context, edgeid string, w io.Writer, r *http.Request) error

	StartSoftwareDownloadW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateSoftwareDownloadW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateSoftwareDownloadStateW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	CreateSoftwareDownloadCredentialsW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllSoftwareDownloadBatchesW(ctx context.Context, w io.Writer, req *http.Request) error
	GetSoftwareDownloadBatchW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error
	SelectAllSoftwareDownloadBatchServiceDomainsW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error
	SelectAllSoftwareDownloadedServiceDomainsW(ctx context.Context, release string, w io.Writer, req *http.Request) error
	SelectAllSoftwareUpdateReleasesW(context context.Context, w io.Writer, req *http.Request) error
	SelectAllSoftwareUpdateServiceDomainsW(ctx context.Context, w io.Writer, req *http.Request) error

	StartSoftwareUpgradeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateSoftwareUpgradeW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateSoftwareUpgradeStateW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllSoftwareUpgradeBatchesW(ctx context.Context, w io.Writer, req *http.Request) error
	GetSoftwareUpgradeBatchW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error
	SelectAllSoftwareUpgradeBatchServiceDomainsW(ctx context.Context, batchID string, w io.Writer, req *http.Request) error

	ExecuteEdgeUpgrade(context context.Context, doc interface{} /* *model.EdgeUpgrade */, callback func(context.Context, interface{}) error) (interface{}, error)
	ExecuteEdgeUpgradeW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	ExecuteEdgeUpgradeWV2(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	SetupSSHTunneling(context context.Context, doc model.WstunRequest, callback func(context.Context, interface{}) error) (model.WstunPayload, error)
	SetupSSHTunnelingW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	TeardownSSHTunneling(context context.Context, doc model.WstunTeardownRequest, callback func(context.Context, interface{}) error) error
	TeardownSSHTunnelingW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	// Tenant pool internal APIs
	CreateRegistration(ctx context.Context, registration *tenantpool.Registration) error
	GetRegistration(ctx context.Context, registrationID string) (*tenantpool.Registration, error)
	CreateTenantClaim(ctx context.Context, registrationID, tenantID, email string) (*tenantpool.TenantClaim, error)
	GetTenantClaim(ctx context.Context, tenantID string) (*tenantpool.TenantClaim, error)
	ReserveTenantClaim(ctx context.Context, registrationID, email string) (string, error)
	ConfirmTenantClaim(ctx context.Context, registrationID, tenantID, email string) (*tenantpool.TenantClaim, error)

	WriteAuditLog(context context.Context, auditLog *model.AuditLog) error
	GetAuditLog(context context.Context, reqID string) ([]model.AuditLog, error)
	GetAuditLogW(context context.Context, reqID string, w io.Writer, r *http.Request) error
	SelectAuditLogs(ctx context.Context, queryParams model.AuditLogQueryParam) (model.AuditLogListResponsePayload, error)
	SelectAuditLogsW(ctx context.Context, w io.Writer, r *http.Request) error
	DeleteTenantAuditLogs(ctx context.Context) error

	QueryAuditLogsV2(ctx context.Context, filter model.AuditLogV2Filter) ([]model.AuditLogV2, error)
	QueryAuditLogsV2W(context context.Context, w io.Writer, r *http.Request) error
	InsertAuditLogV2(ctx context.Context, req model.AuditLogV2InsertRequest) (string, error)
	InsertAuditLogV2W(context context.Context, w io.Writer, r *http.Request) error

	SelectAllMLModels(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.MLModel, error)
	SelectAllMLModelsW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllMLModelsForProject(context context.Context, projectID string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.MLModel, error)
	SelectAllMLModelsForProjectW(context context.Context, projectID string, w io.Writer, r *http.Request) error
	GetMLModel(context context.Context, id string) (model.MLModel, error)
	GetMLModelW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateMLModel(context context.Context, doc interface{} /* *model.MLModel */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateMLModelW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateMLModel(context context.Context, doc interface{} /* *model.MLModel */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateMLModelW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteMLModel(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteMLModelW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	CreateMLModelVersionW(context context.Context, id string, w io.Writer, r *http.Request, callback func(context.Context, interface{}) error) error
	UpdateMLModelVersionW(context context.Context, id string, modelVersion int, w io.Writer, r *http.Request, callback func(context.Context, interface{}) error) error
	DeleteMLModelVersion(context context.Context, id string, modelVersion int, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteMLModelVersionW(context context.Context, id string, modelVersion int, w io.Writer, callback func(context.Context, interface{}) error) error
	GetMLModelVersionSignedURL(context context.Context, modelID string, modelVersion int, minutes int) (string, error)
	GetMLModelVersionSignedURLW(context context.Context, id string, modelVersion int, w io.Writer, r *http.Request) error

	SelectAllMLModelsStatus(context context.Context) ([]model.MLModelStatus, error)
	SelectAllMLModelsStatusW(context context.Context, w io.Writer, r *http.Request) error
	GetMLModelStatus(context context.Context, modelID string) ([]model.MLModelStatus, error)
	GetMLModelStatusW(context context.Context, modelID string, w io.Writer, req *http.Request) error
	CreateMLModelStatus(context context.Context, doc interface{} /* *model.MLModelStatus */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateMLModelStatusW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteMLModelStatus(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteMLModelStatusW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Static file APIs
	GetFile(ctx context.Context, path string, w http.ResponseWriter, r *http.Request) error
	ListFiles(ctx context.Context, path string, w http.ResponseWriter, r *http.Request) error
	CreateFile(ctx context.Context, path string, w http.ResponseWriter, r *http.Request, callback func(context.Context, interface{}) error) error
	DeleteFile(ctx context.Context, path string, w http.ResponseWriter, r *http.Request, callback func(context.Context, interface{}) error) error
	// Internal
	PurgeFiles(ctx context.Context, tenantID string, id string) error

	GetEdgeInventoryDelta(ctx context.Context, payload *model.EdgeInventoryDeltaPayload) (*model.EdgeInventoryDeltaResponse, error)
	GetEdgeInventoryDeltaW(ctx context.Context, w io.Writer, r *http.Request) error

	GetInfraConfig(context context.Context, id string) (model.InfraConfig, error)
	GetInfraConfigW(context context.Context, id string, w io.Writer, r *http.Request) error

	SelectAllUserPublicKeys(ctx context.Context) ([]model.UserPublicKey, error)
	SelectAllUserPublicKeysW(ctx context.Context, w io.Writer, r *http.Request) error

	GetUserPublicKey(ctx context.Context) (model.UserPublicKey, error)
	GetUserPublicKeyW(ctx context.Context, _ string, w io.Writer, _ *http.Request) error
	// No CreateUserPublicKey because UpdateUserPublicKey performs upsert (Update/Insert)
	UpdateUserPublicKey(context context.Context, i interface{} /* *model.UserPublicKey */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateUserPublicKeyW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteUserPublicKey(context context.Context, userID string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteUserPublicKeyW(context context.Context, userID string, w io.Writer, callback func(context.Context, interface{}) error) error
	// UpdateUserPublicKeyUsedTime only updates UserPublicKey used_at time - to track last usage
	UpdateUserPublicKeyUsedTime(context context.Context, i interface{} /* *model.UserPublicKey */) (interface{}, error)
	// GetPublicKeyResolver return a resolver function to resolve current user's public key
	GetPublicKeyResolver() func(*jwt.Token) (interface{}, error)

	SelectAllUserApiTokens(ctx context.Context, userID string) ([]model.UserApiToken, error)
	SelectAllUserApiTokensW(ctx context.Context, w io.Writer, _ *http.Request) error
	GetUserApiTokensW(context context.Context, userID string, w io.Writer, req *http.Request) error
	CreateUserApiToken(ctx context.Context, i interface{} /* *model.UserApiToken */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateUserApiTokenW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteUserApiToken(context context.Context, tokenID string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteUserApiTokenW(ctx context.Context, tokenID string, w io.Writer, callback func(context.Context, interface{}) error) error
	UpdateUserApiTokenUsedTime(context context.Context) (interface{}, error)
	UpdateUserApiToken(context context.Context, i interface{} /* *model.UserApiToken */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateUserApiTokenW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	GetClaimsVerifier() func(jwt.MapClaims) error

	GetServices(context context.Context, host string) (model.Service, error)
	GetServicesW(context context.Context, w io.Writer, req *http.Request) error
	GetServicesInternal(ctx context.Context, req *http.Request) (model.Service, error)
	GetServiceLandingURL(ctx context.Context, req *http.Request) string

	RenderApplication(context context.Context, appID, edgeID string, param model.RenderApplicationPayload) (model.RenderApplicationResponse, error)
	RenderApplicationW(context context.Context, appID, edgeID string, w io.Writer, r *http.Request) error

	SelectAllProjectServices(context context.Context, entitiesQueryParam *model.EntitiesQueryParamV1) ([]model.ProjectService, error)
	SelectAllProjectServicesW(context context.Context, w io.Writer, req *http.Request) error
	GetProjectService(context context.Context, id string) (model.ProjectService, error)
	GetProjectServiceW(context context.Context, id string, w io.Writer, req *http.Request) error
	CreateProjectService(context context.Context, i interface{} /* *model.ProjectService */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateProjectServiceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateProjectService(context context.Context, i interface{} /* *model.ProjectService */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateProjectServiceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteProjectService(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteProjectServiceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Log collectors
	SelectAllLogCollectors(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.LogCollector, error)
	SelectAllLogCollectorsW(context context.Context, w io.Writer, r *http.Request) error
	GetLogCollector(context context.Context, id string) (model.LogCollector, error)
	GetLogCollectorW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateLogCollector(context context.Context, i interface{} /* *model.LogCollector */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateLogCollector(context context.Context, i interface{} /* *model.LogCollector */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateStateLogCollector(context context.Context, i interface{} /* *model.LogCollector */, callback func(context.Context, interface{}) error) (interface{}, error)
	StartLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	StopLogCollectorW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteLogCollector(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteLogCollectorW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	RunHelmTemplate(context context.Context, req *http.Request, appID string) (result AppYaml, err error)
	RunHelmTemplateW(context context.Context, unused string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error
	CreateHelmApplicationW(context context.Context, unused string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error
	UpdateHelmApplicationW(context context.Context, id string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error

	CreateStorageProfile(ctx context.Context, svcDomainID string, sp *model.StorageProfile) (interface{}, error)
	CreateStorageProfileW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error
	SelectAllStorageProfileForServiceDomainW(context context.Context, svcDomainID string, w io.Writer, req *http.Request) error
	UpdateStorageProfile(ctx context.Context, svcDomainID string, ID string, sp *model.StorageProfile) (interface{}, error)
	UpdateStorageProfileW(ctx context.Context, svcDomainID string, ID string, w io.Writer, req *http.Request, callback func(context.Context, interface{}) error) error

	// Service Class
	CreateServiceClass(ctx context.Context, i interface{} /* *model.ServiceClass */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateServiceClassW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateServiceClass(ctx context.Context, i interface{} /* *model.ServiceClass */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateServiceClassW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllServiceClasses(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceClassQueryParam) (model.ServiceClassListPayload, error)
	SelectAllServiceClassesW(ctx context.Context, w io.Writer, r *http.Request) error
	GetServiceClass(ctx context.Context, id string) (model.ServiceClass, error)
	GetServiceClassW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	DeleteServiceClass(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteServiceClassW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	// Service Instance
	CreateServiceInstance(ctx context.Context, i interface{} /* *model.ServiceInstanceParam */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateServiceInstanceW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateServiceInstance(ctx context.Context, i interface{} /* *model.ServiceInstanceParam */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateServiceInstanceW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllServiceInstances(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceInstanceQueryParam) (model.ServiceInstanceListPayload, error)
	SelectAllServiceInstancesW(ctx context.Context, w io.Writer, r *http.Request) error
	GetServiceInstance(ctx context.Context, id string) (model.ServiceInstance, error)
	GetServiceInstanceW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	DeleteServiceInstance(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteServiceInstanceW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectServiceInstanceStatuss(ctx context.Context, id string, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceInstanceStatusQueryParam) (model.ServiceInstanceStatusListPayload, error)
	SelectServiceInstanceStatussW(ctx context.Context, id string, w io.Writer, req *http.Request) error

	// Service Binding
	CreateServiceBinding(ctx context.Context, i interface{} /* *model.ServiceBindingParam */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateServiceBindingW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllServiceBindings(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceBindingQueryParam) (model.ServiceBindingListPayload, error)
	SelectAllServiceBindingsW(ctx context.Context, w io.Writer, r *http.Request) error
	GetServiceBinding(ctx context.Context, id string) (model.ServiceBinding, error)
	GetServiceBindingW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	DeleteServiceBinding(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteServiceBindingW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectServiceBindingStatuss(ctx context.Context, id string, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceBindingStatusQueryParam) (model.ServiceBindingStatusListPayload, error)
	SelectServiceBindingStatussW(ctx context.Context, id string, w io.Writer, req *http.Request) error

	// HTTP Service Proxy
	CreateHTTPServiceProxy(ctx context.Context, i interface{} /* *model.HTTPServiceProxyCreateParamPayload */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateHTTPServiceProxyW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateHTTPServiceProxy(ctx context.Context, i interface{} /* *model.HTTPServiceProxyUpdateParamPayload */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateHTTPServiceProxyW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllHTTPServiceProxies(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam) (model.HTTPServiceProxyListPayload, error)
	SelectAllHTTPServiceProxiesW(ctx context.Context, w io.Writer, r *http.Request) error
	GetHTTPServiceProxy(ctx context.Context, id string) (model.HTTPServiceProxy, error)
	GetHTTPServiceProxyW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	DeleteHTTPServiceProxy(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteHTTPServiceProxyW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error

	// Kubernetes Cluster
	CreateKubernetesCluster(ctx context.Context, i interface{} /* *model.KubernetesCluster */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateKubernetesClusterW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	SelectAllKubernetesClusters(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam) (model.KubernetesClustersListResponsePayload, error)
	SelectAllKubernetesClustersW(ctx context.Context, w io.Writer, r *http.Request) error
	GetKubernetesCluster(ctx context.Context, id string) (model.KubernetesCluster, error)
	GetKubernetesClusterW(ctx context.Context, id string, w io.Writer, req *http.Request) error
	UpdateKubernetesCluster(ctx context.Context, i interface{} /* *model.KubernetesCluster*/, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateKubernetesClusterW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteKubernetesCluster(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteKubernetesClusterW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
	GetKubernetesClusterHandle(ctx context.Context, kubernetestClusterID string, payload model.GetHandlePayload) (model.KubernetesClusterCert, error)
	GetKubernetesClusterHandleW(ctx context.Context, kubernetestClusterID string, w io.Writer, req *http.Request) error
	GetKubernetesClusterInstaller(ctx context.Context) (model.KubernetesClusterInstaller, error)
	GetKubernetesClusterInstallerW(ctx context.Context, w io.Writer, req *http.Request) error
	UpdateKubernetesClusterKubeVersion(ctx context.Context, kubernetestClusterID string) error // Internal

	GetViewonlyUsersForSD(ctx context.Context, svcDomainID string) ([]model.User, error)
	GetViewonlyUsersForSDW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error
	AddViewonlyUsersToSD(ctx context.Context, svcDomainID string, userIDs []string) error
	AddViewonlyUsersToSDW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error
	RemoveViewonlyUsersFromSD(ctx context.Context, svcDomainID string, userIDs []string) error
	RemoveViewonlyUsersFromSDW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error

	// Data drivers
	SelectAllDataDriverClasses(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverClass, int, error)
	SelectAllDataDriverClassesW(context context.Context, w io.Writer, r *http.Request) error
	GetDataDriverClass(context context.Context, id string) (model.DataDriverClass, error)
	GetDataDriverClassW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDataDriverClass(context context.Context, doc interface{} /* *model.DataDriverClass */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataDriverClassW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataDriverClass(context context.Context, doc interface{} /* *model.DataDriverClass */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataDriverClassW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDataDriverClass(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDataDriverClassW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectAllDataDriverInstances(context context.Context, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverInstance, int, error)
	SelectAllDataDriverInstancesW(context context.Context, w io.Writer, r *http.Request) error
	SelectAllDataDriverInstancesByClassId(context context.Context, id string) ([]model.DataDriverInstance, error)
	SelectAllDataDriverInstancesByClassIdW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetDataDriverInstance(context context.Context, id string) (model.DataDriverInstance, error)
	GetDataDriverInstanceW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDataDriverInstance(context context.Context, doc interface{} /* *model.DataDriverInstance */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataDriverInstanceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataDriverInstance(context context.Context, doc interface{} /* *model.DataDriverInstance */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataDriverInstanceW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDataDriverInstance(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDataDriverInstanceW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectDataDriverConfigsByInstanceId(context context.Context, id string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverConfig, int, error)
	SelectDataDriverConfigsByInstanceIdW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetDataDriverConfig(context context.Context, id string) (model.DataDriverConfig, error)
	GetDataDriverConfigW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDataDriverConfig(context context.Context, doc interface{} /* *model.DataDriverConfig */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataDriverConfigW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataDriverConfig(context context.Context, doc interface{} /* *model.DataDriverConfig */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataDriverConfigW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDataDriverConfig(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDataDriverConfigW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error

	SelectDataDriverStreamsByInstanceId(context context.Context, id string, entitiesQueryParam *model.EntitiesQueryParam) ([]model.DataDriverStream, int, error)
	SelectDataDriverStreamsByInstanceIdW(context context.Context, id string, w io.Writer, r *http.Request) error
	GetDataDriverStream(context context.Context, id string) (model.DataDriverStream, error)
	GetDataDriverStreamW(context context.Context, id string, w io.Writer, r *http.Request) error
	CreateDataDriverStream(context context.Context, doc interface{} /* *model.DataDriverStream */, callback func(context.Context, interface{}) error) (interface{}, error)
	CreateDataDriverStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	UpdateDataDriverStream(context context.Context, doc interface{} /* *model.DataDriverStream */, callback func(context.Context, interface{}) error) (interface{}, error)
	UpdateDataDriverStreamW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error
	DeleteDataDriverStream(context context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error)
	DeleteDataDriverStreamW(context context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error
}

type dbObjectModelAPI struct {
	*base.DBObjectModelAPI
	prod bool
}

// TODO may need to revisit to support table alias
const orderByNameID = "ORDER BY name, id"
const orderByID = "ORDER BY id"
const orderByUpdatedAt = "ORDER BY updated_at DESC"

var sessionID = base.GetUUID()
var queryMap = make(map[string]string)
var orderByHelper = base.NewOrderByHelper()

var awsSession *session.Session

var keyService crypto.KeyService

func isMinio() bool {
	return *config.Cfg.ObjectStorageEngine == "minio"
}

func InitGlobals() {
	var err error
	if isMinio() {
		// Configure to use MinIO Server
		s3Config := &aws.Config{
			Credentials:      credentials.NewStaticCredentials(*config.Cfg.MinioAccessKey, *config.Cfg.MinioSecretKey, ""),
			Endpoint:         aws.String(*config.Cfg.MinioURL),
			Region:           aws.String("us-west-2"),
			DisableSSL:       aws.Bool(true),
			S3ForcePathStyle: aws.Bool(true),
		}
		awsSession, err = session.NewSession(s3Config)
	} else {
		// Create AWS session and AWS S3 client
		awsSession, err = session.NewSession(&aws.Config{
			Region: aws.String(*config.Cfg.AWSRegion)},
		)
	}
	if err != nil {
		panic(err)
	}

	keyService = crypto.NewKeyService(*config.Cfg.AWSRegion, *config.Cfg.JWTSecret, *config.Cfg.AWSKMSKey, *config.Cfg.UseKMS)

}

// NewObjectModelAPI creates a ObjectModelAPI based on sql DB
// Note: api with the same config should be shared
// to minimize resource usage
func NewObjectModelAPI(args ...interface{}) (ObjectModelAPI, error) {
	return NewObjectModelAPIWithCache(nil, args...)
}

// NewObjectModelAPIWithCache creates a ObjectModelAPI based on sql DB with a caching layer
func NewObjectModelAPIWithCache(redisClient *redis.Client, args ...interface{}) (ObjectModelAPI, error) {
	dbURL, err := base.GetDBURL(*config.Cfg.SQL_Dialect, *config.Cfg.SQL_DB, *config.Cfg.SQL_User, *config.Cfg.SQL_Password, *config.Cfg.SQL_Host, *config.Cfg.SQL_Port, *config.Cfg.DisableDBSSL)
	if err != nil {
		return nil, err
	}
	roDbURL := dbURL
	if config.Cfg.SQL_ReadOnlyHost != nil && len(*config.Cfg.SQL_ReadOnlyHost) > 0 {
		roDbURL, err = base.GetDBURL(*config.Cfg.SQL_Dialect, *config.Cfg.SQL_DB, *config.Cfg.SQL_User, *config.Cfg.SQL_Password, *config.Cfg.SQL_ReadOnlyHost, *config.Cfg.SQL_Port, *config.Cfg.DisableDBSSL)
		if err != nil {
			return nil, err
		}
	}
	dbAPI, err := base.NewDBObjectModelAPI(*config.Cfg.SQL_Dialect, dbURL, roDbURL, redisClient)
	if err != nil {
		return nil, err
	}

	// custom DB configurations
	db := dbAPI.GetDB()
	roDB := dbAPI.GetReadOnlyDB()
	haveReadonlyDB := db != roDB

	maxCnx := *config.Cfg.SQL_MaxCnx
	if maxCnx != 0 {
		db.SetMaxOpenConns(maxCnx)
		if haveReadonlyDB {
			roDB.SetMaxOpenConns(maxCnx)
		}
	}

	maxIdleCnx := *config.Cfg.SQL_MaxIdleCnx
	if maxIdleCnx != 0 {
		db.SetMaxIdleConns(maxIdleCnx)
		if haveReadonlyDB {
			roDB.SetMaxIdleConns(maxIdleCnx)
		}
	}

	maxCnxLife := *config.Cfg.SQL_MaxCnxLife
	if maxCnxLife != 0 {
		db.SetConnMaxLifetime(maxCnxLife)
		if haveReadonlyDB {
			roDB.SetConnMaxLifetime(maxCnxLife)
		}
	}

	prod := len(args) != 0

	return &dbObjectModelAPI{
		DBObjectModelAPI: dbAPI,
		prod:             prod,
	}, nil
}

func (dbAPI *dbObjectModelAPI) Close() error {
	return dbAPI.DBObjectModelAPI.Close()
}

// DeleteEntity - generic entity delete function
func DeleteEntity(context context.Context, dbAPI *dbObjectModelAPI, tableName string, idName string, id string, doc interface{}, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponse{}

	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	m := map[string]interface{}{}
	m[idName] = id
	if tableName != "tenant_model" {
		m["tenant_id"] = authContext.TenantID
	}
	result, err := dbAPI.Delete(context, tableName, m)
	if err != nil {
		return resp, err
	}
	if base.IsDeleteSuccessful(result) {
		resp.ID = id
		if callback != nil {
			// delete callback will get its arg from closure
			go callback(context, doc)
		}
	}
	return resp, nil
}

// DeleteEntityV2 is similar to DeleteEntity with V2 response
func DeleteEntityV2(ctx context.Context, dbAPI *dbObjectModelAPI, tableName string, idName string, id string, doc interface{}, callback func(context.Context, interface{}) error) (interface{}, error) {
	respV2 := model.DeleteDocumentResponseV2{}
	resp, err := DeleteEntity(ctx, dbAPI, tableName, idName, id, doc, callback)
	if err != nil {
		return respV2, nil
	}
	respV2.ID = resp.(model.DeleteDocumentResponse).ID
	return respV2, nil
}

type nameStruct struct {
	Name string `db:"name"`
}

func validateObjectID(id string, idName string) error {
	if len(id) == 0 || len(id) > 64 || strings.Contains(id, ";") {
		return errcode.NewBadRequestError(idName)
	}
	return nil
}

// getObjectName generic function to get object name given table name and object id
// This function provides an efficient way to get object name in case fetching entire object can be expensive.
func (dbAPI *dbObjectModelAPI) getObjectName(context context.Context, tableName string, id string, idName string) (string, error) {
	if err := validateObjectID(id, idName); err != nil {
		return "", err
	}
	results := []nameStruct{}
	query := fmt.Sprintf("SELECT name from %s WHERE id = :id", tableName)
	if err := dbAPI.Query(context, &results, query, IDDBO{ID: id}); err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", errcode.NewRecordNotFoundError(id)
	}
	return results[0].Name, nil
}

// GetProjectNameFn uses closure to return a function which can get project name by id
// The returned function can be passed in RbacContext to provide better error message.
func GetProjectNameFn(context context.Context, dbAPI ObjectModelAPI) func(string) string {
	return func(projectID string) string {
		projectName, err := dbAPI.GetProjectName(context, projectID)
		if err != nil {
			return ""
		}
		return projectName
	}
}

// getObjectNames generic function to get object names given table name and object ids
// This function provides an efficient way to get object names in case fetching entire objects can be expensive.
func (dbAPI *dbObjectModelAPI) getObjectNames(context context.Context, tableName string, ids []string, idName string) ([]string, error) {
	if len(ids) == 0 {
		return nil, errcode.NewBadRequestError(idName)
	}
	for _, id := range ids {
		if err := validateObjectID(id, idName); err != nil {
			return nil, err
		}
	}

	results := []nameStruct{}
	query := fmt.Sprintf("SELECT name from %s WHERE id IN (:ids)", tableName)
	if err := dbAPI.QueryIn(context, &results, query, idFilter{IDs: ids}); err != nil {
		return nil, err
	}
	if len(results) == 0 {
		idsStr := strings.Join(ids, ", ")
		return nil, errcode.NewRecordNotFoundError(idsStr)
	}
	names := []string{}
	for _, result := range results {
		names = append(names, result.Name)
	}
	return names, nil
}

func getEtag(r *http.Request) string {
	etag := ""
	if r != nil {
		if match := r.Header.Get("If-None-Match"); match != "" {
			etag = match
		}
	}
	return etag
}

// If content is unchanged (etag match),
// then return 304 to avoid sending the same content
// It checks for the top/root level records.
func handleEtag(w io.Writer, etag string, objs interface{}) (bool, error) {
	rw, ok := w.(http.ResponseWriter)
	if ok {
		data, err := json.Marshal(objs)
		if err != nil {
			return true, errcode.NewDataConversionError(err.Error())
		}
		newEtag := fmt.Sprintf("%s-%x", sessionID, md5.Sum(data))
		if newEtag == etag {
			rw.WriteHeader(http.StatusNotModified)
			return true, nil
		}
		rw.Header().Set("Etag", newEtag)
		return false, nil
	}
	return false, nil
}

func uniqueProjectUserInfos(inList []model.ProjectUserInfo) []model.ProjectUserInfo {
	keys := make(map[string]bool)
	list := []model.ProjectUserInfo{}
	for _, entry := range inList {
		if _, value := keys[entry.UserID]; !value {
			keys[entry.UserID] = true
			list = append(list, entry)
		}
	}
	return list
}

func makeAdminContext(ctx context.Context) (context.Context, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return ctx, err
	}
	authContextIA := &base.AuthContext{
		TenantID: authContext.TenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	newContext := context.WithValue(ctx, base.AuthContextKey, authContextIA)
	return newContext, nil
}

type tenantIDParam struct {
	TenantID  string `db:"tenant_id"`
	ProjectID string `db:"project_id"`
	State     string `db:"state"`
}
type tenantIDParam2 struct {
	TenantID   string   `db:"tenant_id"`
	ProjectIDs []string `db:"project_ids"`
	State      string   `db:"state"`
}
type tenantIDParam3 struct {
	TenantID string   `db:"tenant_id"`
	EdgeIDs  []string `db:"edge_ids"`
	State    string   `db:"state"`
}
type tenantIDParam4 struct {
	TenantID string   `db:"tenant_id"`
	IDs      []string `db:"ids"`
	State    string   `db:"state"`
}
type tenantIDParam5 struct {
	TenantID   string   `db:"tenant_id"`
	ProjectIDs []string `db:"project_ids"`
	ID         string   `db:"id"`
	State      string   `db:"state"`
}

func (dbAPI *dbObjectModelAPI) selectTenantEntityCount(context context.Context, projectID string, query string) (int, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return 0, err
	}
	tenantID := authContext.TenantID
	param := tenantIDParam{
		TenantID:  tenantID,
		ProjectID: projectID,
	}
	count := 0
	rows, err := dbAPI.GetDB().NamedQuery(query, param)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}
func (dbAPI *dbObjectModelAPI) selectTenantEntityCount2(context context.Context, projectIDs []string, query string) (int, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return 0, err
	}
	tenantID := authContext.TenantID
	param := tenantIDParam2{
		TenantID:   tenantID,
		ProjectIDs: projectIDs,
	}
	count := 0

	db := dbAPI.GetDB()
	q, args, err := db.BindNamed(query, param)
	if err != nil {
		return 0, err
	}
	// if needIn {
	// convert $d back to ? needed by sqlx.In
	q = reDollarVar.ReplaceAllString(q, "?")
	q, args, err = sqlx.In(q, args...)
	if err != nil {
		return 0, err
	}
	q = db.Rebind(q)
	// }

	rows, err := db.Query(q, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}
func (dbAPI *dbObjectModelAPI) selectTenantEntityCount3(context context.Context, edgeIDs []string, query string) (int, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return 0, err
	}
	tenantID := authContext.TenantID
	param := tenantIDParam3{
		TenantID: tenantID,
		EdgeIDs:  edgeIDs,
	}
	count := 0

	db := dbAPI.GetDB()
	q, args, err := db.BindNamed(query, param)
	if err != nil {
		return 0, err
	}
	// if needIn {
	// convert $d back to ? needed by sqlx.In
	q = reDollarVar.ReplaceAllString(q, "?")
	q, args, err = sqlx.In(q, args...)
	if err != nil {
		return 0, err
	}
	q = db.Rebind(q)
	// }

	rows, err := db.Query(q, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return 0, err
		}
	}
	return count, nil
}

type idResult struct {
	ID string `db:"id"`
}

func (dbAPI *dbObjectModelAPI) selectEntityIDsByParam(context context.Context, query string, param interface{}) ([]string, error) {
	idsResult := []idResult{}
	err := dbAPI.QueryIn(context, &idsResult, query, param)
	if err != nil {
		glog.Errorf("Failed to get entity IDs by param %+v. Error: %s", param, err.Error())
		return nil, err
	}
	ids := []string{}
	for _, row := range idsResult {
		ids = append(ids, row.ID)
	}
	return ids, nil
}

func (dbAPI *dbObjectModelAPI) selectEntityIDs(ctx context.Context, projectID string, query string) ([]string, error) {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID
	param := tenantIDParam{
		TenantID:  tenantID,
		ProjectID: projectID,
	}

	return dbAPI.selectEntityIDsByParam(ctx, query, param)
}

func (dbAPI *dbObjectModelAPI) selectEntityIDs2(context context.Context, projectIDs []string, query string) ([]string, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID
	param := tenantIDParam2{
		TenantID:   tenantID,
		ProjectIDs: projectIDs,
	}
	db := dbAPI.GetDB()
	q, args, err := db.BindNamed(query, param)
	if err != nil {
		return []string{}, err
	}
	// if needIn {
	// convert $d back to ? needed by sqlx.In
	q = reDollarVar.ReplaceAllString(q, "?")
	q, args, err = sqlx.In(q, args...)
	if err != nil {
		return []string{}, err
	}
	q = db.Rebind(q)
	// }

	ids := []string{}
	rows, err := db.Queryx(q, args...)
	if err != nil {
		return ids, nil
	}
	defer rows.Close()
	for rows.Next() {
		var p idResult
		err = rows.StructScan(&p)
		if err != nil {
			return []string{}, err
		}
		ids = append(ids, p.ID)
	}
	return ids, nil
}
func (dbAPI *dbObjectModelAPI) selectEntityIDs3(context context.Context, edgeIDs []string, query string) ([]string, error) {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return nil, err
	}
	tenantID := authContext.TenantID
	param := tenantIDParam3{
		TenantID: tenantID,
		EdgeIDs:  edgeIDs,
	}
	db := dbAPI.GetDB()
	q, args, err := db.BindNamed(query, param)
	if err != nil {
		return []string{}, err
	}
	// if needIn {
	// convert $d back to ? needed by sqlx.In
	q = reDollarVar.ReplaceAllString(q, "?")
	q, args, err = sqlx.In(q, args...)
	if err != nil {
		return []string{}, err
	}
	q = db.Rebind(q)
	// }

	ids := []string{}
	rows, err := db.Queryx(q, args...)
	if err != nil {
		return ids, nil
	}
	defer rows.Close()
	for rows.Next() {
		var p idResult
		err = rows.StructScan(&p)
		if err != nil {
			return []string{}, err
		}
		ids = append(ids, p.ID)
	}
	return ids, nil
}

type ListQueryInfo struct {
	StartPage  base.PageToken
	TotalCount int
	IDs        []string
	AllIDs     []string
	EntityType string
}

func (dbAPI *dbObjectModelAPI) getEntityListQueryInfoCommon(context context.Context, entityType string, queryParam *model.EntitiesQueryParam, ids []string, idsErr error) (ListQueryInfo, error) {
	queryInfo := ListQueryInfo{EntityType: entityType}
	if idsErr != nil {
		return queryInfo, idsErr
	}
	count := len(ids)
	if count == 0 {
		return queryInfo, nil
	}
	itemIndex := queryParam.PageIndex * queryParam.PageSize
	if itemIndex >= count {
		return queryInfo, errcode.NewRecordNotFoundError("paging")
	}
	if queryParam.PageIndex == 0 {
		queryInfo.StartPage = base.StartPageToken
	} else {
		queryInfo.StartPage = base.PageToken(ids[itemIndex])
	}
	queryInfo.TotalCount = count
	endIndex := itemIndex + queryParam.PageSize
	if endIndex > count {
		queryInfo.IDs = ids[itemIndex:]
	} else {
		queryInfo.IDs = ids[itemIndex:endIndex]
	}
	queryInfo.AllIDs = ids
	return queryInfo, nil
}

func (dbAPI *dbObjectModelAPI) getEntityListQueryInfo(context context.Context, entityType string, projectID string, queryParam *model.EntitiesQueryParam, getIDsFn func(context.Context, string, *model.EntitiesQueryParam) ([]string, error)) (ListQueryInfo, error) {
	ids, idsErr := getIDsFn(context, projectID, queryParam)
	return dbAPI.getEntityListQueryInfoCommon(context, entityType, queryParam, ids, idsErr)
}
func (dbAPI *dbObjectModelAPI) getEntityListQueryInfo2(context context.Context, entityType string, projectIDs []string, queryParam *model.EntitiesQueryParam, getIDsFn func(context.Context, []string, *model.EntitiesQueryParam) ([]string, error)) (ListQueryInfo, error) {
	ids, idsErr := getIDsFn(context, projectIDs, queryParam)
	return dbAPI.getEntityListQueryInfoCommon(context, entityType, queryParam, ids, idsErr)
}

func (dbAPI *dbObjectModelAPI) getEntitiesByIDs(context context.Context, entityType string, queryTemplate string, queryInfo ListQueryInfo, entitiesQueryParam *model.EntitiesQueryParam, outputSlice interface{}) error {
	if len(queryInfo.IDs) == 0 {
		return nil
	}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	param := tenantIDParam4{
		TenantID: authContext.TenantID,
		IDs:      queryInfo.IDs,
	}
	query, err := buildQuery(entityType, queryTemplate, entitiesQueryParam, orderByNameID)
	if err != nil {
		return err
	}
	return dbAPI.QueryIn(context, outputSlice, query, param)
}

func (dbAPI *dbObjectModelAPI) getEntities(context context.Context, entityType string, queryTemplate string, entitiesQueryParam *model.EntitiesQueryParam, outputSlice interface{}) error {
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return err
	}
	param := tenantIDParam{
		TenantID: authContext.TenantID,
	}
	query, err := buildLimitQuery(entityType, queryTemplate, entitiesQueryParam, orderByNameID)
	if err != nil {
		return err
	}
	return dbAPI.Query(context, outputSlice, query, param)
}

func makeEntityListResponsePayload(queryParam *model.EntitiesQueryParam, queryInfo *ListQueryInfo) model.EntityListResponsePayload {
	return model.EntityListResponsePayload{
		PageIndex:   queryParam.PageIndex,
		PageSize:    queryParam.PageSize,
		TotalCount:  queryInfo.TotalCount,
		OrderBy:     strings.Join(queryParam.OrderBy, ", "),
		OrderByKeys: orderByHelper.GetOrderByKeys(queryInfo.EntityType),
	}
}

func makePagedListResponsePayload(queryParam *model.EntitiesQueryParam, queryInfo *ListQueryInfo) model.PagedListResponsePayload {
	return model.PagedListResponsePayload{
		PageIndex:  queryParam.PageIndex,
		PageSize:   queryParam.PageSize,
		TotalCount: queryInfo.TotalCount,
	}
}

func getFilterAndOrderBy(entityType string, queryParam base.QueryParameter, defaultOrderBy string) (filterAndOrderBy string, err error) {
	return orderByHelper.GetFilterAndOrderBy(entityType, queryParam, defaultOrderBy)
}

func getFilterAndOrderByWithTableAlias(entityType string, queryParam base.QueryParameter, defaultOrderBy, defaultAlias string, aliasMapping map[string]string) (filterAndOrderBy string, err error) {
	return orderByHelper.GetFilterAndOrderByWithTableAlias(entityType, queryParam, defaultOrderBy, defaultAlias, aliasMapping)
}

func buildQuery(entityType, queryTemplate string, queryParam base.QueryParameter, defaultOrderBy string) (string, error) {
	return orderByHelper.BuildQuery(entityType, queryTemplate, queryParam, defaultOrderBy)
}

func buildQueryWithTableAlias(entityType, queryTemplate string, queryParam base.QueryParameter, defaultOrderBy, defaultAlias string, aliasMapping map[string]string) (string, error) {
	return orderByHelper.BuildQueryWithTableAlias(entityType, queryTemplate, queryParam, defaultOrderBy, defaultAlias, aliasMapping)
}

func buildLimitQuery(entityType string, queryTemplate string, queryParam *model.EntitiesQueryParam, defaultOrderBy string) (string, error) {
	filterAndOrderBy, err := getFilterAndOrderBy(entityType, queryParam, defaultOrderBy)
	if err != nil {
		return "", err
	}
	pageIndex := 0
	pageSize := DefaultPageSize
	if queryParam != nil {
		pageIndex = queryParam.PageIndex
		if pageIndex < 0 {
			pageIndex = 0
			queryParam.PageIndex = pageIndex
		}
		pageSize = queryParam.PageSize
		if pageSize <= 0 {
			pageSize = DefaultPageSize
			queryParam.PageSize = pageSize
		}
	}
	sfx := fmt.Sprintf("%s OFFSET %d LIMIT %d", filterAndOrderBy, pageIndex*pageSize, pageSize)
	return fmt.Sprintf(queryTemplate, sfx), nil
}

type idNameStruct struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

type idFilter struct {
	IDs []string `db:"ids"`
}

type tenantIdFilter struct {
	TenantID string   `db:"tenant_id"`
	IDs      []string `db:"ids"`
}

func (dbAPI *dbObjectModelAPI) queryEntitiesByTenantAndIds(ctx context.Context, slice interface{}, tableName string, ids []string) error {
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE tenant_id = :tenant_id AND id in (:ids)", tableName)
	params := tenantIdFilter{
		TenantID: authContext.TenantID,
		IDs:      ids,
	}
	return dbAPI.QueryIn(ctx, slice, query, params)
}

func (dbAPI *dbObjectModelAPI) getNamesByIDs(ctx context.Context, tableName string, ids []string) (map[string]string, error) {
	id2NameMap := map[string]string{}
	if len(ids) == 0 {
		return id2NameMap, nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return nil, err
	}
	query := fmt.Sprintf("SELECT id, name FROM %s WHERE tenant_id = :tenant_id AND id in (:ids)", tableName)
	params := tenantIdFilter{
		TenantID: authContext.TenantID,
		IDs:      ids,
	}
	idNames := []idNameStruct{}
	err = dbAPI.QueryIn(ctx, &idNames, query, params)
	if err != nil {
		return nil, err
	}
	for _, idName := range idNames {
		id2NameMap[idName.ID] = idName.Name
	}
	return id2NameMap, nil
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	glog.Infof("TIME TRACK> %s took %s", name, elapsed)
}

func NilToEmptyStrings(strings []string) []string {
	if strings == nil {
		return []string{}
	}
	return strings
}

// GetEntityIDsInPage returns the all the entity IDs, entity IDs per page based on the pagination set in the query param.
// The callback should be able to return entity IDs for all service domains or in a service domain for a tenant.
func (dbAPI *dbObjectModelAPI) GetEntityIDsInPage(ctx context.Context, projectID string, svcDomainID string, queryParam *model.EntitiesQueryParam, callback func(context.Context, *model.ServiceDomainEntityModelDBO, *model.EntitiesQueryParam) ([]string, error)) ([]string, []string, error) {
	entityIDsInPage := []string{}
	entityIDs, err := dbAPI.GetEntityIDs(ctx, projectID, svcDomainID, queryParam, callback)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetEntityIDsInPage:  GetEntityIDs failed. Error: %s\n"), err.Error())
		return entityIDsInPage, entityIDsInPage, err
	}
	// sort entityIDs
	sort.StringSlice(entityIDs).Sort()
	startIndex := queryParam.GetPageIndex() * queryParam.GetPageSize()
	if startIndex < len(entityIDs) {
		endIndex := startIndex + queryParam.GetPageSize()
		if endIndex > len(entityIDs) {
			endIndex = len(entityIDs)
		}
		entityIDsInPage = entityIDs[startIndex:endIndex]
	}
	return entityIDs, entityIDsInPage, nil
}

// GetEntityIDs returns the all the entity IDs.
// The callback should be able to return entity IDs for all service domains or in a service domain for a tenant.
func (dbAPI *dbObjectModelAPI) GetEntityIDs(ctx context.Context, projectID string, svcDomainID string, queryParam *model.EntitiesQueryParam, callback func(context.Context, *model.ServiceDomainEntityModelDBO, *model.EntitiesQueryParam) ([]string, error)) ([]string, error) {
	entityIDs := []string{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return entityIDs, err
	}
	if projectID != "" {
		if !auth.IsProjectMember(projectID, authContext) {
			return entityIDs, errcode.NewPermissionDeniedError("RBAC")
		}
		project, err := dbAPI.GetProject(ctx, projectID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "GetEntityIDs: GetProject failed. Error: %s\n"), err.Error())
			return entityIDs, err
		}
		for _, svcDomainID := range project.EdgeIDs {
			entityIDsInSvcDomain, err := dbAPI.GetEntityIDs(ctx, "", svcDomainID, queryParam, callback)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "GetEntityIDs failed. Error: %s\n"), err.Error())
				return entityIDs, err
			}
			entityIDs = append(entityIDs, entityIDsInSvcDomain...)
		}
	} else if svcDomainID != "" || auth.IsInfraAdminRole(authContext) {
		tenantID := authContext.TenantID
		svcDomainModel := &model.ServiceDomainEntityModelDBO{BaseModelDBO: model.BaseModelDBO{TenantID: tenantID}, SvcDomainID: svcDomainID}
		entityIDs, err = callback(ctx, svcDomainModel, queryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "GetEntityIDs: callback failed. Error: %s\n"), err.Error())
			return entityIDs, err
		}
	} else {
		// all IDs per RBAC
		projectIDs := auth.GetProjectIDs(authContext)
		svcDomainMap := map[string]bool{}
		// always allow service domain get itself
		if ok, svcDomainID := base.IsEdgeRequest(authContext); ok && svcDomainID != "" {
			entityIDsInSvcDomain, err := dbAPI.GetEntityIDs(ctx, "", svcDomainID, queryParam, callback)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "GetEntityIDs failed. Error: %s\n"), err.Error())
				return entityIDs, err
			}
			svcDomainMap[svcDomainID] = true
			entityIDs = append(entityIDs, entityIDsInSvcDomain...)
		}
		if len(projectIDs) != 0 {
			projects, err := dbAPI.getProjectsByIDs(ctx, authContext.TenantID, projectIDs)
			if err != nil {
				return entityIDs, err
			}
			for _, project := range projects {
				for _, svcDomainID := range project.EdgeIDs {
					if !svcDomainMap[svcDomainID] {
						svcDomainMap[svcDomainID] = true
						entityIDsInSvcDomain, err := dbAPI.GetEntityIDs(ctx, "", svcDomainID, queryParam, callback)
						if err != nil {
							glog.Errorf(base.PrefixRequestID(ctx, "GetEntityIDs failed. Error: %s\n"), err.Error())
							return entityIDs, err
						}
						entityIDs = append(entityIDs, entityIDsInSvcDomain...)
					}
				}
			}
		}
	}
	return entityIDs, nil
}

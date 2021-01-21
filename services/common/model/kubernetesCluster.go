package model

import (
	"cloudservices/common/base"
)

// KubernetesCluster - the contents of a kubernetes cluster
// swagger:model KubernetesCluster
type KubernetesCluster struct {
	// required: true
	BaseModel
	//
	// Name of the Kubernetes Cluster.
	// Maximum length of 200 characters.
	//
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:200"`
	//
	// KubernetesCluster description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// Connecting status of the current cluster
	// Either true or false
	//
	// required: false
	Connected bool `json:"connected,omitempty" db:"connected"`
	//
	// Onboarded status of the current cluster
	// Either true or false
	//
	// required: false
	Onboarded bool `json:"onboarded,omitempty" db:"onboarded"`
	//
	// Kubernetes version of the current cluster
	//
	// required: false
	KubeVersion string `json:"kubeVersion,omitempty" db:"kube_version" validate:"range=0:20"`
	//
	// Chart version of the current cluster
	//
	// required: false
	ChartVersion string `json:"chartVersion,omitempty" db:"chart_version" validate:"range=0:20"`
}

// swagger:model KubernetesClustersOnboardInfo
type KubernetesClustersOnboardInfo struct {
	// required: true
	KubernetesClusterID string `json:"id"`
	// required: true
	SSHPublicKey string `json:"sshPublicKey" db:"ssh_pub_key" validate:"range=0:500"`
}

// Ok
// swagger:response KubernetesClustersGetResponse
type KubernetesClustersGetResponse struct {
	// in: body
	// required: true
	Payload *KubernetesCluster
}

// Ok
// swagger:response KubernetesClustersListResponse
type KubernetesClustersListResponse struct {
	// in: body
	// required: true
	Payload *KubernetesClustersListResponsePayload
}

// payload for KubernetesClustersListResponse
type KubernetesClustersListResponsePayload struct {
	// required: true
	EntityListResponsePayload
	// list of applications
	// required: true
	KubernetesClustersList []KubernetesCluster `json:"result"`
}

// KubernetesClusterInstaller info
type KubernetesClusterInstaller struct {
	// This is the unique installer ID which is the version
	// required: true
	ID string `json:"id"`
	//
	//	This is the edge helm download URL
	//
	// required: true
	URL string `json:"url"`
}

// Ok
// swagger:response KubernetesClusterInstallerResponse
type KubernetesClusterInstallerResponse struct {
	// in: body
	// required: true
	Payload *KubernetesClusterInstaller
}

// KubernetesClusterCreateParam is KubernetesCluster used as API parameter
// swagger:parameters KubernetesClustersCreate
type KubernetesClusterCreateParam struct {
	// Describes the kubernetes cluster creation request.
	// in: body
	// required: true
	Body *KubernetesCluster `json:"body"`
}

// KubernetesClusterUpdateParam is KubernetesCluster used as API parameter
// swagger:parameters KubernetesClustersUpdate
type KubernetesClusterUpdateParam struct {
	// in: body
	// required: true
	Body *KubernetesCluster `json:"body"`
}

//  KubernetesClusterHandlePayload payload for Kubernetes Cluster handle call
type KubernetesClusterHandlePayload struct {
	// required: true
	Token string `json:"token"`
	// required: true
	TenantID string `json:"tenantId"`
}

// KubernetesClusterHandleParam payload for Kubernetes Cluster handle call
// swagger:parameters KubernetesClustersHandle
// in: body
type KubernetesClusterHandleParam struct {
	// in: body
	// required: true
	Body *KubernetesClusterHandlePayload
}

type KubernetesClusterCertCore struct {
	//
	// Certificate for the kubernetes cluster using old/fixed root CA.
	//
	// required: true
	Certificate string `json:"certificate" db:"certificate" validate:"range=0:4096"`
	//
	// Encrypted private key using old/fixed root CA.
	//
	// required: true
	PrivateKey string `json:"privateKey" db:"private_key" validate:"range=0:4096"`
	//
	// Root CA certificate for the tenant.
	//
	// required: true
	CACertificate string `json:"CACertificate" validate:"range=0:4096"`
	//
	// ntnx:ignore
	// Certificate for mqtt client on the edge
	//
	// required: true
	ClientCertificate string `json:"clientCertificate" db:"client_certificate" validate:"range=0:4096"`
	//
	// ntnx:ignore
	// Encrypted private key corresponding to the client certificate.
	//
	// required: true
	ClientPrivateKey string `json:"clientPrivateKey" db:"client_private_key" validate:"range=0:4096"`
	//
	// Certificate for the kubernetes cluster using per-tenant root CA.
	//
	// required: true
	KubernetesClusterCertificate string `json:"kubernetesClusterCertificate" db:"kubernetes_cluster_certificate" validate:"range=0:4096"`
	//
	// Encrypted private key using per-tenant root CA.
	//
	// required: true
	KubernetesClusterPrivateKey string `json:"kubernetesClusterPrivateKey" db:"kubernetes_cluster_private_key" validate:"range=0:4096"`
	// For security purpose, KubernetesClusterCert can only be
	// retrieved once during kubernetes cluster on-boarding.
	// After that locked will be set to true and
	// the REST API endpoint for getting KubernetesClusterCert
	// will throw error.
	//
	// required: true

	Locked bool `json:"locked" db:"locked"`
}

// KubernetesClusterCert is DB and object model
// swagger:model KubernetesClusterCert
type KubernetesClusterCert struct {
	// required: true
	KubernetesClusterBaseModel
	// required: true
	KubernetesClusterCertCore
}

// Ok
// swagger:response KubernetesClustersHandleResponse
type KubernetesClustersHandleResponse struct {
	// in: body
	// required: true
	Payload *KubernetesClusterCert
}

// swagger:parameters KubernetesClustersList KubernetesClustersGet KubernetesClustersCreate KubernetesClustersUpdate KubernetesClustersDelete KubernetesClusterInstaller
// in: header
type kubernetesClustersAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

func (cluster *KubernetesCluster) ToServiceDomain() *ServiceDomain {
	svcDomain := &ServiceDomain{
		BaseModel: cluster.BaseModel,
		ServiceDomainCore: ServiceDomainCore{
			Name: cluster.Name,
			Type: base.StringPtr(string(KubernetesClusterTargetType)),
		},
		Description: cluster.Description,
		Connected:   cluster.Connected,
	}
	return svcDomain
}

// FromServiceDomain converts the ServiceDomain object to a KubernetesCluster object
func (cluster *KubernetesCluster) FromServiceDomain(svcDomain *ServiceDomain) {
	cluster.BaseModel = svcDomain.BaseModel
	cluster.Name = svcDomain.Name
	cluster.Description = svcDomain.Description
	cluster.Connected = svcDomain.Connected
}

// FromEdgeCert converts from EdgeCert to KubernetesClusterCert
func (clusterCert *KubernetesClusterCert) FromEdgeCert(edgeCert *EdgeCert) {
	clusterCert.KubernetesClusterBaseModel.BaseModel = edgeCert.BaseModel
	clusterCert.KubernetesClusterBaseModel.KubernetesClusterID = edgeCert.EdgeID
	clusterCert.Certificate = edgeCert.Certificate
	clusterCert.PrivateKey = edgeCert.PrivateKey
	clusterCert.CACertificate = edgeCert.CACertificate
	clusterCert.ClientCertificate = edgeCert.ClientCertificate
	clusterCert.ClientPrivateKey = edgeCert.ClientPrivateKey
	clusterCert.KubernetesClusterCertificate = edgeCert.EdgeCertificate
	clusterCert.KubernetesClusterPrivateKey = edgeCert.EdgePrivateKey
	clusterCert.Locked = edgeCert.Locked
}

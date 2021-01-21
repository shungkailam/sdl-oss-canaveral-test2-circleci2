package model

// K8sConfig is the kubernetes config information
type K8sConfig struct {
	//
	// ProviderType is the provider for kubernetes infrastructure
	// For example: RKE, KOPS etc...
	//
	// required: true
	ProviderType string `json:"providerType"`
	//
	// ProviderConfig is the configuration defining the underlying kubernetes infrastructure. It will return
	// the config depending on the k8sProviderType
	//
	// required: true
	// TODO: create interface for different k8s providers
	ProviderConfig interface{} `json:"providerConfig"`
}

// ClusterConfig is the cluster config information
type ClusterConfig struct {
	//
	// floatingIp is the floating IP for the cluster
	// For example: 10.8.4.12
	//
	// required: true
	FloatingIP string `json:"floatingIP"`
}

// InfraConfig is the infra config information
// swagger:model InfraConfig
type InfraConfig struct {
	// required: true
	ClusterConfig ClusterConfig `json:"clusterConfig"`
	// required: true
	K8sConfig K8sConfig `json:"k8sConfig"`
}

// Ok
// swagger:response InfraConfigGetResponse
type InfraConfigGetResponse struct {
	// in: body
	// required: true
	Payload *InfraConfig
}

// swagger:parameters InfraConfigGet
// in: header
type infraConfigAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

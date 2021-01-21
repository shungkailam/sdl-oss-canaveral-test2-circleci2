package model

const (
	// NodeInfoEventName represents the name of event for node info
	NodeInfoEventName = "NodeInfoEventName"

	// NodeHealthStatusHealthy is the healthy status of a node
	NodeHealthStatusHealthy = NodeHealthStatus("HEALTHY")
	// NodeHealthStatusUnhealthy is the unhealthy status of a node
	NodeHealthStatusUnhealthy = NodeHealthStatus("UNHEALTHY")
	// NodeHealthStatusUnknown is the unknown health status of a node
	NodeHealthStatusUnknown = NodeHealthStatus("UNKNOWN")
)

// NodeInfoCore carries the core configuration of the node
type NodeInfoCore struct {
	//
	// Number of CPUs assigned to the node.
	//
	// required: false
	NumCPU string `json:"numCpu,omitempty" db:"num_cpu" validate:"range=0:20"`
	//
	// Total node memory in KB.
	//
	// required: false
	TotalMemoryKB string `json:"totalMemoryKB,omitempty" db:"total_memory_kb" validate:"range=0:20"`
	//
	// Total node storage capacity in KB.
	//
	// required: false
	TotalStorageKB string `json:"totalStorageKB,omitempty" db:"total_storage_kb" validate:"range=0:20"`
	//
	// Information about GPUs associated with the node.
	//
	// required: false
	GPUInfo string `json:"gpuInfo,omitempty" db:"gpu_info" validate:"range=0:20"`
	//
	// Node CPU usage.
	//
	// required: false
	CPUUsage string `json:"cpuUsage,omitempty" db:"cpu_usage" validate:"range=0:20"`
	//
	// Free (available) node memory in KB.
	//
	// required: false
	MemoryFreeKB string `json:"memoryFreeKB,omitempty" db:"memory_free_kb" validate:"range=0:20"`
	//
	// Free (available) node storage in KB.
	//
	// required: false
	StorageFreeKB string `json:"storageFreeKB,omitempty" db:"storage_free_kb" validate:"range=0:20"`
	//
	// Node GPU Usage.
	//
	// required: false
	GPUUsage string `json:"gpuUsage,omitempty" db:"gpu_usage" validate:"range=0:20"`
	//
	// Node version.
	//
	// required: false
	NodeVersion *string `json:"nodeVersion,omitempty" db:"edge_version" validate:"range=0:20"`
	//
	// Node build number.
	//
	// required: false
	NodeBuildNum *string `json:"nodeBuildNum,omitempty" db:"edge_build_num" validate:"range=0:20"`
	//
	// Node Kubernetes version.
	//
	// required: false
	KubeVersion *string `json:"kubeVersion,omitempty" db:"kube_version" validate:"range=0:20"`
	//
	// Node OS version
	//
	// required: false
	OSVersion *string `json:"osVersion,omitempty" db:"os_version" validate:"range=0:64"`
}

// NodeHealthStatus is the health status of the node
// swagger:model NodeHealthStatus
// enum: HEALTHY,UNHEALTHY,UNKNOWN
type NodeHealthStatus string

//
// NodeStatus is the placeholder for the status of the node
//
// swagger:model NodeStatus
type NodeStatus struct {
	Connected bool `json:"connected,omitempty"`
	// Deprecated. Use healthStatus instead
	Healthy      bool             `json:"healthy,omitempty"`
	HealthStatus NodeHealthStatus `json:"healthStatus,omitempty"`
	HealthBits   map[string]bool  `json:"healthBits,omitempty" db:"health_bits"`
	Onboarded    bool             `json:"onboarded,omitempty"`
}

// NodeInfo has node information like the memory, storage and CPU usage
//
// swagger:model NodeInfo
type NodeInfo struct {
	// required: true
	NodeEntityModel
	// required: true
	NodeInfoCore
	// required: true
	NodeStatus
	//
	// Artifacts is a json object for passing node ip and service ports
	//
	// required: false
	Artifacts map[string]interface{} `json:"artifacts,omitempty"`
}

// NodeInfoUpdateParam is the swagger wrapper around NodeInfo
// swagger:parameters NodeInfoUpdate
// in: body
type NodeInfoUpdateParam struct {
	// Describes parameters used to create or update a node
	// in: body
	// required: true
	Body *NodeInfo `json:"body"`
}

// Ok
// swagger:response NodeInfoGetResponse
type NodeInfoGetResponse struct {
	// in: body
	// required: true
	Payload *NodeInfo
}

// Ok
// swagger:response NodeInfoListResponse
type NodeInfoListResponse struct {
	// in: body
	// required: true
	Payload *NodeInfoListPayload
}

// swagger:parameters NodeInfoList ProjectGetNodesInfo NodeInfoGet NodeInfoUpdate
// in: header
type nodeInfoAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// NodeInfoListPayload is the payload for NodeInfoListResponse
type NodeInfoListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of node info
	// required: true
	NodeInfoList []NodeInfo `json:"result"`
}

// Event definitions
type NodeInfoEvent struct {
	ID   string
	Info *NodeInfo
}

func (event *NodeInfoEvent) IsAsync() bool {
	return true
}

func (event *NodeInfoEvent) EventName() string {
	return NodeInfoEventName
}

func (event *NodeInfoEvent) GetID() string {
	return event.ID
}

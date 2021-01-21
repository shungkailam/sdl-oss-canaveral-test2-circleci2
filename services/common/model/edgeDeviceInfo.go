package model

// EdgeDeviceInfoCore carries the core configuration of the edge device
type EdgeDeviceInfoCore struct {
	//
	// Number of CPUs assigned to the edge.
	//
	// required: false
	NumCPU string `json:"numCpu,omitempty" db:"num_cpu" validate:"range=0:20"`
	//
	// Total edge memory in KB.
	//
	// required: false
	TotalMemoryKB string `json:"totalMemoryKB,omitempty" db:"total_memory_kb" validate:"range=0:20"`
	//
	// Total edge storage capacity in KB.
	//
	// required: false
	TotalStorageKB string `json:"totalStorageKB,omitempty" db:"total_storage_kb" validate:"range=0:20"`
	//
	// Information about GPUs associated with the edge.
	//
	// required: false
	GPUInfo string `json:"gpuInfo,omitempty" db:"gpu_info" validate:"range=0:20"`
	//
	// Edge CPU usage.
	//
	// required: false
	CPUUsage string `json:"cpuUsage,omitempty" db:"cpu_usage" validate:"range=0:20"`
	//
	// Free (available) edge memory in KB.
	//
	// required: false
	MemoryFreeKB string `json:"memoryFreeKB,omitempty" db:"memory_free_kb" validate:"range=0:20"`
	//
	// Free (available) edge storage in KB.
	//
	// required: false
	StorageFreeKB string `json:"storageFreeKB,omitempty" db:"storage_free_kb" validate:"range=0:20"`
	//
	// Edge GPU Usage.
	//
	// required: false
	GPUUsage string `json:"gpuUsage,omitempty" db:"gpu_usage" validate:"range=0:20"`
	//
	// Edge version.
	//
	// required: false
	EdgeVersion *string `json:"edgeVersion,omitempty" db:"edge_version" validate:"range=0:20"`
	//
	// Edge build number.
	//
	// required: false
	EdgeBuildNum *string `json:"edgeBuildNum,omitempty" db:"edge_build_num" validate:"range=0:20"`
	//
	// Edge Kubernetes version.
	//
	// required: false
	KubeVersion *string `json:"kubeVersion,omitempty" db:"kube_version" validate:"range=0:20"`
	//
	// Edge OS version
	//
	// required: false
	OSVersion *string `json:"osVersion,omitempty" db:"os_version" validate:"range=0:64"`
}

//
// EdgeDeviceStatus is the placeholder for the status of the edge device
//
// swagger:model EdgeDeviceStatus
type EdgeDeviceStatus struct {
	Connected  bool            `json:"connected,omitempty"`
	Healthy    bool            `json:"healthy,omitempty"`
	HealthBits map[string]bool `json:"healthBits" db:"health_bits"`
	Onboarded  bool            `json:"onboarded,omitempty"`
}

// EdgeDeviceInfo has edge device information like the memory, storage and CPU usage
//
// swagger:model EdgeDeviceInfo
type EdgeDeviceInfo struct {
	// required: true
	EdgeDeviceScopedModel
	// required: true
	EdgeDeviceInfoCore
	// required: true
	EdgeDeviceStatus
	//
	// Artifacts is a json object for passing edge ip and service ports
	//
	// required: false
	Artifacts map[string]interface{} `json:"artifacts,omitempty"`
}

// EdgeDeviceInfoUpdateParam is the swagger wrapper around EdgeDeviceInfo
// swagger:parameters EdgeDeviceInfoUpdate
// in: body
type EdgeDeviceInfoUpdateParam struct {
	// Describes parameters used to create an edge
	// in: body
	// required: true
	Body *EdgeDeviceInfo `json:"body"`
}

// Ok
// swagger:response EdgeDeviceInfoGetResponse
type EdgeDeviceInfoGetResponse struct {
	// in: body
	// required: true
	Payload *EdgeDeviceInfo
}

// Ok
// swagger:response EdgeDeviceInfoListResponse
type EdgeDeviceInfoListResponse struct {
	// in: body
	// required: true
	Payload []*EdgeDeviceInfoListPayload
}

// swagger:parameters EdgeInfoList EdgeInfoListV2 EdgeInfoGet EdgeInfoGetV2 EdgeInfoUpdate EdgeInfoUpdateV2 ProjectGetEdgesInfo ProjectGetEdgesInfoV2 EdgeDeviceInfoUpdate EdgeDeviceInfoGet ProjectGetEdgeDevicesInfo EdgeDeviceInfoList
// in: header
type edgeDeviceInfoAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response EdgeDeviceInfoListResponseV2
type EdgeDeviceInfoListResponseV2 struct {
	// in: body
	// required: true
	Payload *EdgeDeviceInfoListPayload
}

// EdgeDeviceInfoListPayload is the payload for EdgeDeviceInfoListResponseV2
type EdgeDeviceInfoListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge usage info
	// required: true
	EdgeDeviceInfoList []EdgeDeviceInfo `json:"result"`
}

func (deviceInfo *EdgeDeviceInfo) ToNodeInfo() *NodeInfo {
	return &NodeInfo{
		NodeEntityModel: NodeEntityModel{
			ServiceDomainEntityModel: ServiceDomainEntityModel{
				BaseModel:   deviceInfo.BaseModel,
				SvcDomainID: deviceInfo.ClusterID,
			},
			NodeID: deviceInfo.DeviceID,
		},
		NodeInfoCore: NodeInfoCore{
			NumCPU:         deviceInfo.NumCPU,
			TotalMemoryKB:  deviceInfo.TotalMemoryKB,
			TotalStorageKB: deviceInfo.TotalStorageKB,
			GPUInfo:        deviceInfo.GPUInfo,
			CPUUsage:       deviceInfo.CPUUsage,
			MemoryFreeKB:   deviceInfo.MemoryFreeKB,
			StorageFreeKB:  deviceInfo.StorageFreeKB,
			GPUUsage:       deviceInfo.GPUUsage,
			NodeVersion:    deviceInfo.EdgeVersion,
			NodeBuildNum:   deviceInfo.EdgeBuildNum,
			KubeVersion:    deviceInfo.KubeVersion,
			OSVersion:      deviceInfo.OSVersion,
		},
		NodeStatus: NodeStatus{
			Connected:  deviceInfo.Connected,
			Healthy:    deviceInfo.Healthy,
			HealthBits: deviceInfo.HealthBits,
			Onboarded:  deviceInfo.Onboarded,
		},
		Artifacts: deviceInfo.Artifacts,
	}
}

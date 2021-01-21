package model

type EdgeInfo struct {
	//
	// Number of CPUs assigned to the edge.
	//
	// required: false
	NumCPU string `json:"NumCPU,omitempty" db:"num_cpu" validate:"range=0:20"`
	//
	// Total edge memory in KB.
	//
	// required: false
	TotalMemoryKB string `json:"TotalMemoryKB,omitempty" db:"total_memory_kb" validate:"range=0:20"`
	//
	// Total edge storage capacity in KB.
	//
	// required: false
	TotalStorageKB string `json:"TotalStorageKB,omitempty" db:"total_storage_kb" validate:"range=0:20"`
	//
	// Information about GPUs associated with the edge.
	//
	// required: false
	GPUInfo string `json:"GPUInfo,omitempty" db:"gpu_info" validate:"range=0:20"`
	//
	// Edge CPU usage.
	//
	// required: false
	CPUUsage string `json:"CPUUsage,omitempty" db:"cpu_usage" validate:"range=0:20"`
	//
	// Free (available) edge memory in KB.
	//
	// required: false
	MemoryFreeKB string `json:"MemoryFreeKB,omitempty" db:"memory_free_kb" validate:"range=0:20"`
	//
	// Free (available) edge storage in KB.
	//
	// required: false
	StorageFreeKB string `json:"StorageFreeKB,omitempty" db:"storage_free_kb" validate:"range=0:20"`
	//
	// Edge GPU Usage.
	//
	// required: false
	GPUUsage string `json:"GPUUsage,omitempty" db:"gpu_usage" validate:"range=0:20"`
	//
	// Edge version.
	//
	// required: false
	EdgeVersion *string `json:"EdgeVersion,omitempty" db:"edge_version" validate:"range=0:20"`
	//
	// Edge build number.
	//
	// required: false
	EdgeBuildNum *string `json:"EdgeBuildNum,omitempty" db:"edge_build_num" validate:"range=0:20"`
	//
	// Edge Kubernetes version.
	//
	// required: false
	KubeVersion *string `json:"KubeVersion,omitempty" db:"kube_version" validate:"range=0:20"`
	//
	// Edge OS version
	//
	// required: false
	OSVersion *string `json:"OSVersion,omitempty" db:"os_version" validate:"range=0:64"`
}

// EdgeUsageInfo is the DB object and object model for edgeinfo
//
// EdgeUsageInfo has edge information like the memory, storage and CPU usage
//
// swagger:model EdgeUsageInfo
type EdgeUsageInfo struct {
	// required: true
	EdgeBaseModel
	// required: true
	EdgeInfo
	//
	// Edge artifacts is a json object for passing edge ip and service ports
	//
	// required: false
	EdgeArtifacts map[string]interface{} `json:"edgeArtifacts,omitempty"`
}

// ObjectRequestBaseEdgeInfo is used as websocket Edge message
// swagger:model ObjectRequestBaseEdgeInfo
type ObjectRequestBaseEdgeInfo struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc EdgeUsageInfo `json:"doc"`
}

// ResponseBaseEdgeInfo is used as websocket reportEdgeInfo response
// swagger:model ResponseBaseEdgeInfo
type ResponseBaseEdgeInfo struct {
	// required: true
	ResponseBase
	// required: true
	Doc EdgeUsageInfo `json:"doc"`
}

// EdgeInfoCreateParam is EdgeInfo used as API parameter
// swagger:parameters EdgeInfoCreate EdgeInfoCreateV2
// in: body
type EdgeInfoCreateParam struct {
	// Describes parameters used to create an edge
	// in: body
	// required: true
	Body *EdgeUsageInfo `json:"body"`
}

// EdgeInfoUpdateParam is Edge used as API parameter
// swagger:parameters EdgeInfoUpdate EdgeInfoUpdateV2
// in: body
type EdgeInfoUpdateParam struct {
	// in: body
	// required: true
	Body *EdgeUsageInfo `json:"body"`
}

// Ok
// swagger:response EdgeInfoGetResponse
type EdgeInfoGetResponse struct {
	// in: body
	// required: true
	Payload *EdgeUsageInfo
}

// swagger:parameters EdgeInfoList EdgeInfoListV2 EdgeInfoGet EdgeInfoGetV2 EdgeInfoUpdate EdgeInfoUpdateV2 ProjectGetEdgesInfo ProjectGetEdgesInfoV2
// in: header
type edgeInfoAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// Ok
// swagger:response EdgeInfoListResponse
type EdgeInfoListResponse struct {
	// in: body
	// required: true
	Payload *[]EdgeUsageInfo
}

// Ok
// swagger:response EdgeInfoListResponseV2
type EdgeInfoListResponseV2 struct {
	// in: body
	// required: true
	Payload *EdgeInfoListPayload
}

// payload for EdgeInfoListResponseV2
type EdgeInfoListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge usage info
	// required: true
	EdgeUsageInfoList []EdgeUsageInfo `json:"result"`
}

type EdgesInfoByID []EdgeUsageInfo

func (a EdgesInfoByID) Len() int           { return len(a) }
func (a EdgesInfoByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EdgesInfoByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (usageInfo EdgeUsageInfo) ToEdgeDeviceInfo() EdgeDeviceInfo {
	deviceInfo := EdgeDeviceInfo{}
	deviceInfo.BaseModel = usageInfo.BaseModel
	deviceInfo.ClusterID = usageInfo.ID
	deviceInfo.DeviceID = usageInfo.ID
	deviceInfo.NumCPU = usageInfo.NumCPU
	deviceInfo.TotalMemoryKB = usageInfo.TotalMemoryKB
	deviceInfo.TotalStorageKB = usageInfo.TotalStorageKB
	deviceInfo.GPUInfo = usageInfo.GPUInfo
	deviceInfo.CPUUsage = usageInfo.CPUUsage
	deviceInfo.MemoryFreeKB = usageInfo.MemoryFreeKB
	deviceInfo.StorageFreeKB = usageInfo.StorageFreeKB
	deviceInfo.GPUUsage = usageInfo.GPUUsage
	deviceInfo.EdgeVersion = usageInfo.EdgeVersion
	deviceInfo.EdgeBuildNum = usageInfo.EdgeBuildNum
	deviceInfo.KubeVersion = usageInfo.KubeVersion
	deviceInfo.OSVersion = usageInfo.OSVersion
	deviceInfo.Artifacts = usageInfo.EdgeArtifacts
	return deviceInfo
}

func (deviceInfo EdgeDeviceInfo) ToEdgeUsageInfo() EdgeUsageInfo {
	usageInfo := EdgeUsageInfo{}
	usageInfo.BaseModel = deviceInfo.BaseModel
	usageInfo.EdgeID = deviceInfo.DeviceID
	usageInfo.NumCPU = deviceInfo.NumCPU
	usageInfo.TotalMemoryKB = deviceInfo.TotalMemoryKB
	usageInfo.TotalStorageKB = deviceInfo.TotalStorageKB
	usageInfo.GPUInfo = deviceInfo.GPUInfo
	usageInfo.CPUUsage = deviceInfo.CPUUsage
	usageInfo.MemoryFreeKB = deviceInfo.MemoryFreeKB
	usageInfo.StorageFreeKB = deviceInfo.StorageFreeKB
	usageInfo.GPUUsage = deviceInfo.GPUUsage
	usageInfo.EdgeVersion = deviceInfo.EdgeVersion
	usageInfo.EdgeBuildNum = deviceInfo.EdgeBuildNum
	usageInfo.KubeVersion = deviceInfo.KubeVersion
	usageInfo.OSVersion = deviceInfo.OSVersion
	usageInfo.EdgeArtifacts = deviceInfo.Artifacts
	return usageInfo
}

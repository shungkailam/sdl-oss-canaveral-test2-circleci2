package model

// DataPipeline is object model for data pipeline
//
// DataPipelines are fundamental building blocks for Karbon Platform Services data pipeline.
//
// swagger:model DataPipeline
type DataPipeline struct {
	// required: true
	DataStream
}

// DataPipelineCreateParam is DataPipeline used as API parameter
// swagger:parameters DataPipelineCreate
type DataPipelineCreateParam struct {
	// This is a data pipeline creation request description
	// in: body
	// required: true
	Body *DataPipeline `json:"body"`
}

// DataPipelineUpdateParam is DataPipeline used as API parameter
// swagger:parameters DataPipelineUpdate
type DataPipelineUpdateParam struct {
	// in: body
	// required: true
	Body *DataPipeline `json:"body"`
}

// Ok
// swagger:response DataPipelineGetResponse
type DataPipelineGetResponse struct {
	// in: body
	// required: true
	Payload *DataPipeline
}

// Ok
// swagger:response DataPipelineListResponse
type DataPipelineListResponse struct {
	// in: body
	// required: true
	Payload *DataPipelineListPayload
}

// payload for DataPipelineListResponse
type DataPipelineListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of data pipelines
	// required: true
	DataPipelineList []DataPipeline `json:"result"`
}

// DataPipelineContainersBaseObject - dataPipelineId and edgeID for which the
// containers will listed.
// swagger:model DataPipelineContainersBaseObject
type DataPipelineContainersBaseObject struct {
	DataPipelineID string `json:"dataPipelineId"`
	EdgeID         string `json:"edgeId"`
}

// DataPipelineContainers encapsulates the container names
// for a specific data pipeline on a specific edge.
// swagger:model DataPipelineContainers
type DataPipelineContainers struct {
	DataPipelineContainersBaseObject
	ContainerNames []string `json:"containerNames"`
}

// GetDataPipelineContainersResponse is the API response that
// returns a list of container names for a given pipeline on a given edge.
// swagger:response GetDataPipelineContainersResponse
type GetDataPipelineContainersResponse struct {
	// in: body
	// required: true
	Payload *DataPipelineContainers
}

// ObjectRequestBaseDataPipelineContainers is used as a websocket
// "getDataPipelineContainers" message
// swagger:model ObjectRequestBaseDataPipelineContainers
type ObjectRequestBaseDataPipelineContainers struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc DataPipelineContainersBaseObject `json:"doc"`
}

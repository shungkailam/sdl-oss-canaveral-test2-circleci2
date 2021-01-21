package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
)

// param for OnUpdateEdge message
// swagger:parameters WsMessagingOnUpdateEdge
type OnUpdateEdgeParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseEdge `json:"request"`
}

// param for onCreateCategory, OnUpdateCategory message
// swagger:parameters WsMessagingOnCreateCategory WsMessagingOnUpdateCategory
type OnCreateCategoryParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseCategory `json:"request"`
}

// param for onCreateDataStream, OnUpdateDataStream message
// swagger:parameters WsMessagingOnCreateDataStream WsMessagingOnUpdateDataStream
type OnCreateDataStreamParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseDataStream `json:"request"`
}

// param for onGetDataPipelineContainers message
// swagger:parameters WsMessagingOnGetDataPipelineContainers
type OnGetDataPipelineContainersParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseDataPipelineContainers `json:"request"`
}

// param for onCreateApplication, OnUpdateApplication message
// swagger:parameters WsMessagingOnCreateApplication WsMessagingOnUpdateApplication
type OnCreateApplicationParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseApplication `json:"request"`
}

// param for onCreateProjectService, OnUpdateProjectService message
// swagger:parameters WsMessagingOnCreateProjectService WsMessagingOnUpdateProjectService
type OnCreateProjectServiceParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseProjectService `json:"request"`
}

// param for onGetApplicationContainers message
// swagger:parameters WsMessagingOnGetApplicationContainers
type OnGetApplicationContainersParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseApplicationContainers `json:"request"`
}

// param for onCreateDockerProfile, OnUpdateDockerProfile message
// swagger:parameters WsMessagingOnCreateDockerProfile WsMessagingOnUpdateDockerProfile
type OnCreateDockerProfileParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseDockerProfile `json:"request"`
}

// param for onCreateDataSource, OnUpdateDataSource message
// swagger:parameters WsMessagingOnCreateDataSource WsMessagingOnUpdateDataSource
type OnCreateDataSourceParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseDataSource `json:"request"`
}

// param for onCreateScript, OnUpdateScript message
// swagger:parameters WsMessagingOnCreateScript WsMessagingOnUpdateScript
type OnCreateScriptParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseScript `json:"request"`
}

// param for onCreateScriptRuntime, OnUpdateScriptRuntime message
// swagger:parameters WsMessagingOnCreateScriptRuntime WsMessagingOnUpdateScriptRuntime
type OnCreateScriptRuntimeParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseScriptRuntime `json:"request"`
}

// param for onCreateCloudCreds, OnUpdateCloudCreds message
// swagger:parameters WsMessagingOnCreateCloudCreds WsMessagingOnUpdateCloudCreds
type OnCreateCloudCredsParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseCloudCreds `json:"request"`
}

// param for onCreateLogCollector, OnUpdateLogCollector message
// swagger:parameters WsMessagingOnCreateLogCollector WsMessagingOnUpdateLogCollector
type OnCreateLogCollectorParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseLogCollector `json:"request"`
}

// param for onCreateExecuteEdgeUpgrade message
// swagger:parameters WsMessagingOnCreateExecuteEdgeUpgrade
type OnExecuteEdgeUpgradeParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseExecuteEdgeUpgrade `json:"request"`
}

// param for onCreateProject, OnUpdateProject message
// swagger:parameters WsMessagingOnCreateProject WsMessagingOnUpdateProject
type OnCreateProjectParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseProject `json:"request"`
}

// param for logStream message
// swagger:parameters WsMessagingLogStream
type LogStreamParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseLogStream `json:"request"`
}

// param for onDeleteCategory message
// swagger:parameters WsMessagingOnDeleteEdge WsMessagingOnDeleteCategory WsMessagingOnDeleteDataStream WsMessagingOnDeleteDataSource WsMessagingOnDeleteScript WsMessagingOnDeleteCloudCreds WsMessagingOnDeleteApplication WsMessagingOnDeleteDockerProfile WsMessagingOnDeleteProject WsMessagingOnDeleteMLModel WsMessagingOnDeleteLogCollector
type DeleteRequestParam struct {
	// in: body
	// required: true
	Request *model.DeleteRequest `json:"request"`
}

// ReportEdgeParam
// param for reportEdge message
// swagger:parameters WsMessagingReportEdge
type ReportEdgeParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseEdge `json:"request"`
}

// ReportEdgeInfoParam
// param for reportEdgeInfo message
// swagger:parameters WsMessagingReportEdgeInfo
type ReportEdgeInfoParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseEdgeInfo `json:"request"`
}

// ReportSensorsParam
// param for reportSensors message
// swagger:parameters WsMessagingReportSensors
type ReportSensorsParam struct {
	// in: body
	// required: true
	Request *model.ReportSensorsRequest `json:"request"`
}

// Ok
// swagger:response ResponseBaseEdgeWrapper
type ResponseBaseEdgeWrapper struct {
	// in: body
	// required: true
	Response *model.ResponseBaseEdge
}

// SetupSSHTunnelingParam param for setupSSHTunneling message
// swagger:parameters WsMessagingSetupSSHTunneling
type SetupSSHTunnelingParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestSetupSSHTunneling `json:"request"`
}

// TeardownSSHTunnelingParam param for teardownSSHTunneling message
// swagger:parameters WsMessagingTeardownSSHTunneling
type TeardownSSHTunnelingParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestTeardownSSHTunneling `json:"request"`
}

// param for onCreateMLModel, OnUpdateMLModel message
// swagger:parameters WsMessagingOnCreateMLModel WsMessagingOnUpdateMLModel
type OnCreateMLModelParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseMLModel `json:"request"`
}

// Ok
// swagger:response NotificationTopicsWrapper
type NotificationTopicsWrapper struct {
	// in: body
	// required: true
	Response *model.NotificationTopics
}

// param for onCreateSoftwareUpdate message
// swagger:parameters WsMessagingOnCreateSoftwareUpdate
type OnCreateSoftwareUpdateParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseSoftwareUpdate `json:"request"`
}

// param for OnUpdateSoftwareUpdate message
// swagger:parameters WsMessagingOnUpdateSoftwareUpdate
type OnUpdateSoftwareUpdateParam struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseSoftwareUpdate `json:"request"`
}

// param for OnCreateServiceInstance message
// swagger:parameters WsMessagingOnCreateServiceInstance
type OnCreateServiceInstance struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseServiceInstance `json:"request"`
}

// param for OnUpdateServiceInstance message
// swagger:parameters WsMessagingOnUpdateServiceInstance
type OnUpdateServiceInstance struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseServiceInstance `json:"request"`
}

// param for OnCreateServiceBinding message
// swagger:parameters WsMessagingOnCreateServiceBinding
type OnCreateServiceBinding struct {
	// in: body
	// required: true
	Request *model.ObjectRequestBaseServiceBinding `json:"request"`
}

// param for httpProxy message
// swagger:parameters WsMessagingHTTPProxy
type HTTPProxyParam struct {
	// in: body
	// required: true
	Request *model.ProxyRequest `json:"request"`
}

// Ok
// swagger:response HTTPProxyResponse
type HTTPProxyResponse struct {
	// in: body
	// required: true
	Response *model.ProxyResponse
}

// param for OnCreateDataDriverInstance message
// swagger:parameters WsMessagingOnCreateDataDriverInstance
type OnCreateDataDriverInstance struct {
	// in: body
	// required: true
	Request *model.DataDriverInstance `json:"request"`
}

// param for OnUpdateDataDriverInstance message
// swagger:parameters WsMessagingOnUpdateDataDriverInstance
type OnUpdateDataDriverInstance struct {
	// in: body
	// required: true
	Request *model.DataDriverInstance `json:"request"`
}

func getWebSocketRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateEdge",
			// swagger:route POST /v1/wsdocs/onUpdateEdge WsMessagingOnUpdateEdge
			//
			// Document websocket request / response payload for onUpdateEdge message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteEdge",
			// swagger:route POST /v1/wsdocs/onDeleteEdge WsMessagingOnDeleteEdge
			//
			// Document websocket request / response payload for onDeleteEdge message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateCategory",
			// swagger:route POST /v1/wsdocs/onCreateCategory WsMessagingOnCreateCategory
			//
			// Document websocket request / response payload for onCreateCategory message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateCategory",
			// swagger:route POST /v1/wsdocs/onUpdateCategory WsMessagingOnUpdateCategory
			//
			// Document websocket request / response payload for onUpdateCategory message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteCategory",
			// swagger:route POST /v1/wsdocs/onDeleteCategory WsMessagingOnDeleteCategory
			//
			// Document websocket request / response payload for onDeleteCategory message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateDataSource",
			// swagger:route POST /v1/wsdocs/onCreateDataSource WsMessagingOnCreateDataSource
			//
			// Document websocket request / response payload for onCreateDataSource message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateDataSource",
			// swagger:route POST /v1/wsdocs/onUpdateDataSource WsMessagingOnUpdateDataSource
			//
			// Document websocket request / response payload for onUpdateDataSource message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteDataSource",
			// swagger:route POST /v1/wsdocs/onDeleteDataSource WsMessagingOnDeleteDataSource
			//
			// Document websocket request / response payload for onDeleteDataSource message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateDataStream",
			// swagger:route POST /v1/wsdocs/onCreateDataStream WsMessagingOnCreateDataStream
			//
			// Document websocket request / response payload for onCreateDataStream message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateDataStream",
			// swagger:route POST /v1/wsdocs/onUpdateDataStream WsMessagingOnUpdateDataStream
			//
			// Document websocket request / response payload for onUpdateDataStream message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteDataStream",
			// swagger:route POST /v1/wsdocs/onDeleteDataStream WsMessagingOnDeleteDataStream
			//
			// Document websocket request / response payload for onDeleteDataStream message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "GET",
			path:   "/v1/wsdocs/onGetDataPipelineContainers",
			// swagger:route GET /v1/wsdocs/onGetDataPipelineContainers WsMessagingOnGetDataPipelineContainers
			//
			// Document websocket request / response payload for onGetDataPipelineContainers message.
			//
			//	   Produces:
			//     - application/json
			//
			//	   Responses:
			//		 200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateApplication",
			// swagger:route POST /v1/wsdocs/onCreateApplication WsMessagingOnCreateApplication
			//
			// Document websocket request / response payload for onCreateApplication message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "GET",
			path:   "/v1/wsdocs/onGetApplicationContainers",
			// swagger:route GET /v1/wsdocs/onGetApplicationContainers WsMessagingOnGetApplicationContainers
			//
			// Document websocket request / response payload for onGetApplicationContainers message.
			//
			//	   Produces:
			//     - application/json
			//
			//	   Responses:
			//		 200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateApplication",
			// swagger:route POST /v1/wsdocs/onUpdateApplication WsMessagingOnUpdateApplication
			//
			// Document websocket request / response payload for onUpdateApplication message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteApplication",
			// swagger:route POST /v1/wsdocs/onDeleteApplication WsMessagingOnDeleteApplication
			//
			// Document websocket request / response payload for onDeleteApplication message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateDockerProfile",
			// swagger:route POST /v1/wsdocs/onCreateDockerProfile WsMessagingOnCreateDockerProfile
			//
			// Document websocket request / response payload for onCreateDockerProfile message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateDockerProfile",
			// swagger:route POST /wsdocs/onUpdateDockerProfile WsMessagingOnUpdateDockerProfile
			//
			// Document websocket request / response payload for onUpdateDockerProfile message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteDockerProfile",
			// swagger:route POST /v1/wsdocs/onDeleteDockerProfile WsMessagingOnDeleteDockerProfile
			//
			// Document websocket request / response payload for onDeleteDockerProfile message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateScript",
			// swagger:route POST /v1/wsdocs/onCreateScript WsMessagingOnCreateScript
			//
			// Document websocket request / response payload for onCreateScript message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateScript",
			// swagger:route POST /v1/wsdocs/onUpdateScript WsMessagingOnUpdateScript
			//
			// Document websocket request / response payload for onUpdateScript message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteScript",
			// swagger:route POST /v1/wsdocs/onDeleteScript WsMessagingOnDeleteScript
			//
			// Document websocket request / response payload for onDeleteScript message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateScriptRuntime",
			// swagger:route POST /v1/wsdocs/onCreateScriptRuntime WsMessagingOnCreateScriptRuntime
			//
			// Document websocket request / response payload for onCreateScriptRuntime message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateScriptRuntime",
			// swagger:route POST /v1/wsdocs/onUpdateScriptRuntime WsMessagingOnUpdateScriptRuntime
			//
			// Document websocket request / response payload for onUpdateScriptRuntime message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteScriptRuntime",
			// swagger:route POST /v1/wsdocs/onDeleteScriptRuntime WsMessagingOnDeleteScriptRuntime
			//
			// Document websocket request / response payload for onDeleteScriptRuntime message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateCloudCreds",
			// swagger:route POST /v1/wsdocs/onCreateCloudCreds WsMessagingOnCreateCloudCreds
			//
			// Document websocket request / response payload for onCreateCloudCreds message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateCloudCreds",
			// swagger:route POST /v1/wsdocs/onUpdateCloudCreds WsMessagingOnUpdateCloudCreds
			//
			// Document websocket request / response payload for onUpdateCloudCreds message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteCloudCreds",
			// swagger:route POST /v1/wsdocs/onDeleteCloudCreds WsMessagingOnDeleteCloudCreds
			//
			// Document websocket request / response payload for onDeleteCloudCreds message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateProject",
			// swagger:route POST /v1/wsdocs/onCreateProject WsMessagingOnCreateProject
			//
			// Document websocket request / response payload for onCreateProject message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateProject",
			// swagger:route POST /v1/wsdocs/onUpdateProject WsMessagingOnUpdateProject
			//
			// Document websocket request / response payload for onUpdateProject message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteProject",
			// swagger:route POST /v1/wsdocs/onDeleteProject WsMessagingOnDeleteProject
			//
			// Document websocket request / response payload for onDeleteProject message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/reportEdge",
			// swagger:route POST /v1/wsdocs/reportEdge WsMessagingReportEdge
			//
			// Document websocket request / response payload for reportEdge message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseEdgeWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/reportEdgeInfo",
			// swagger:route POST /v1/wsdocs/reportEdgeInfo WsMessagingReportEdgeInfo
			//
			// Document websocket request / response payload for reportEdgeInfo message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseEdgeWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/reportSensors",
			// swagger:route POST /v1/wsdocs/reportSensors WsMessagingReportSensors
			//
			// Document websocket request / response payload for reportSensors message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "GET",
			path:   "/v1/wsdocs/getNotificationTopics",
			// swagger:route GET /v1/wsdocs/getNotificationTopics WsMessagingGetNotificationTopics
			//
			// Document websocket request / response payload for getNotificationTopics message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: NotificationTopicsWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/logUpload",
			// swagger:route POST /v1/wsdocs/logUpload WsMessagingLogUpload
			//
			// Document websocket request / response payload for logUpload message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/logUploadComplete",
			// swagger:route POST /v1/wsdocs/logUploadComplete WsMessagingLogUploadComplete
			//
			// Document websocket request / response payload for logUploadComplete message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/logStream",
			// swagger:route POST /v1/wsdocs/logStream WsMessagingLogStream
			//
			// Document websocket request / response payload for logStream message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/application-status",
			// swagger:route POST /v1/wsdocs/application-status WsMessagingReportAppStatus
			//
			// Document websocket request / response payload for application-status message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			// handle: reportAppStatusHandle,
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/executeEdgeUpgrade",
			// swagger:route POST /v1/wsdocs/executeEdgeUpgrade WsMessagingOnCreateExecuteEdgeUpgrade
			//
			// Document websocket request / response payload for executeEdgeUpgrade message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/setupSSHTunneling",
			// swagger:route POST /v1/wsdocs/setupSSHTunneling WsMessagingSetupSSHTunneling
			//
			// Document websocket request / response payload for setupSSHTunneling message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/teardownSSHTunneling",
			// swagger:route POST /v1/wsdocs/teardownSSHTunneling WsMessagingTeardownSSHTunneling
			//
			// Document websocket request / response payload for teardownSSHTunneling message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateMLModel",
			// swagger:route POST /v1/wsdocs/onCreateMLModel WsMessagingOnCreateMLModel
			//
			// Document websocket request / response payload for onCreateMLModel message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteMLModel",
			// swagger:route POST /v1/wsdocs/onDeleteMLModel WsMessagingOnDeleteMLModel
			//
			// Document websocket request / response payload for onDeleteMLModel message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateMLModel",
			// swagger:route POST /v1/wsdocs/onUpdateMLModel WsMessagingOnUpdateMLModel
			//
			// Document websocket request / response payload for onUpdateMLModel message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateProjectService",
			// swagger:route POST /v1/wsdocs/onCreateProjectService WsMessagingOnCreateProjectService
			//
			// Document websocket request / response payload for onCreateProjectService message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateProjectService",
			// swagger:route POST /v1/wsdocs/onUpdateProjectService WsMessagingOnUpdateProjectService
			//
			// Document websocket request / response payload for onUpdateProjectService message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteProjectService",
			// swagger:route POST /v1/wsdocs/onDeleteProjectService WsMessagingOnDeleteProjectService
			//
			// Document websocket request / response payload for onDeleteProjectService message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateLogCollector",
			// swagger:route POST /v1/wsdocs/onCreateLogCollector WsMessagingOnCreateLogCollector
			//
			// Document websocket request / response payload for onCreateLogCollector message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateLogCollector",
			// swagger:route POST /v1/wsdocs/onUpdateLogCollector WsMessagingOnUpdateLogCollector
			//
			// Document websocket request / response payload for onUpdateLogCollector message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteLogCollector",
			// swagger:route POST /v1/wsdocs/onDeleteLogCollector WsMessagingOnDeleteLogCollector
			//
			// Document websocket request / response payload for onDeleteLogCollector message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onCreateSoftwareUpdate",
			// swagger:route POST /v1/wsdocs/onCreateSoftwareUpdate WsMessagingOnCreateSoftwareUpdate
			//
			// Document websocket request / response payload for onCreateSoftwareUpdate message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onUpdateSoftwareUpdate",
			// swagger:route POST /v1/wsdocs/onUpdateSoftwareUpdate WsMessagingOnUpdateSoftwareUpdate
			//
			// Document websocket request / response payload for onUpdateSoftwareUpdate message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/OnCreateServiceInstance",
			// swagger:route POST /v1/wsdocs/onCreateSoftwareUpdate WsMessagingOnCreateServiceInstance
			//
			// Document websocket request / response payload for onCreateServiceInstance message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/OnUpdateServiceInstance",
			// swagger:route POST /v1/wsdocs/onUpdateSoftwareUpdate WsMessagingOnUpdateServiceInstance
			//
			// Document websocket request / response payload for onUpdateServiceInstance message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteServiceInstance",
			// swagger:route POST /v1/wsdocs/onDeleteServiceInstance WsMessagingOnDeleteServiceInstance
			//
			// Document websocket request / response payload for onDeleteServiceInstance message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/OnCreateServiceBinding",
			// swagger:route POST /v1/wsdocs/onCreateSoftwareUpdate WsMessagingOnCreateServiceBinding
			//
			// Document websocket request / response payload for onCreateServiceBinding message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteServiceBinding",
			// swagger:route POST /v1/wsdocs/onDeleteServiceBinding WsMessagingOnDeleteServiceBinding
			//
			// Document websocket request / response payload for onDeleteServiceBinding message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/httpProxy",
			// swagger:route POST /v1/wsdocs/httpProxy WsMessagingHTTPProxy
			//
			// Document websocket request / response payload for httpProxy message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: HTTPProxyResponse
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/OnCreateDataDriverInstance",
			// swagger:route POST /v1/wsdocs/OnCreateDataDriverInstance WsMessagingOnCreateDataDriverInstance
			//
			// Document websocket request / response payload for onCreateDataDriverInstance message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/OnUpdateDataDriverInstance",
			// swagger:route POST /v1/wsdocs/OnUpdateDataDriverInstance WsMessagingOnUpdateDataDriverInstance
			//
			// Document websocket request / response payload for onUpdateDataDriverInstance message.
			//
			//     Produces:
			//     - application/json
			//
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
		{
			method: "POST",
			path:   "/v1/wsdocs/onDeleteDataDriverInstance",
			// swagger:route POST /v1/wsdocs/onDeleteDataDriverInstance WsMessagingOnDeleteDataDriverInstance
			//
			// Document websocket request / response payload for onDeleteDataDriverInstance message.
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: ResponseBaseWrapper
			handle: make404Handler(),
		},
	}
}

package meta

// Entity enum type
type Entity int

// Entity constants
const (
	EntityApplication Entity = iota + 1
	EntityApplicationStatus
	EntityCategory
	EntityCloudCreds
	EntityDataSource
	EntityDataStream
	EntityDockerProfile
	EntityContainerRegistry
	EntityEdge
	EntityEdgeDevice
	EntityEdgeCluster
	EntityEdgeCert
	EntityEdgeInfo
	EntityNode
	EntityNodeInfo
	EntityServiceDomain
	EntityLog
	EntityProject
	EntityScript
	EntityScriptRuntime
	EntitySensor
	EntityTenant
	EntityUser
	EntityEdgeUpgrade
	EntityMLModel
	EntityProjectService
	EntityLogCollector
	EntityInfraLogCollector
	EntityStorageProfile
	EntityServiceInstance
	EntityServiceBinding
	EntityHTTPServiceProxy
	EntityKubernetesCluster
	EntityDataDriverClass
	EntityDataDriverInstance
	EntityDataDriverConfig
	EntityDataDriverStream
)

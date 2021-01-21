package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"strings"
)

// NodeRole holds the role of a node in the service domain
type NodeRole struct {
	// required: false
	// Whether to allow node to be master in this service domain.
	// Minimum of three master nodes are required.
	Master bool `json:"master"`
	// required: false
	// Whether to allow node to be worker in this service domain.
	Worker bool `json:"worker"`
}

// NodeCore holds the core properties of a node in the service domain
type NodeCore struct {
	//
	// Node name.
	// Maximum length edge name is determined by kubernetes.
	// Name length limited to 60 and contraints are defined here
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:60"`
	//
	// Node serial number
	//
	// required: true
	SerialNumber string `json:"serialNumber" db:"serial_number" validate:"range=0:200"`
	//
	// Node IP Address
	//
	// required: true
	IPAddress string `json:"ipAddress" db:"ip_address" validate:"range=0:20"`
	//
	// Node Gateway IP address
	//
	// required: true
	Gateway string `json:"gateway" db:"gateway" validate:"range=0:20"`
	//
	// Node subnet mask
	//
	// required: true
	Subnet string `json:"subnet" db:"subnet" validate:"range=0:20"`
	//
	// NodeRole is a json object for passing node roles
	//
	// required: false
	Role *NodeRole `json:"role,omitempty"`
	// required: false
	IsBootstrapMaster bool `json:"isBootstrapMaster"  db:"is_bootstrap_master"`
}

// Node is the DB object and object model for nodes
//
// Node is a node in a service domain.
//
// swagger:model Node
type Node struct {
	// required: true
	ServiceDomainEntityModel
	// required: true
	NodeCore
	//
	// Node description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
}

// ToEdgeCluster converts Node object to EdgeCluster object
func (node *Node) ToEdgeCluster() *EdgeCluster {
	// TODO
	return nil
}

// NodeWithClusterInfo is the DB object and object model for a node and indicates if the node is a bootstrap master
//
// swagger:model NodeWithClusterInfo
type NodeWithClusterInfo struct {
	// required: true
	Node
	// required: true
	IsBootstrapMaster bool `json:"isBootstrapMaster"  db:"is_bootstrap_master"`
	// required: false
	BootstrapMasterSSHPublicKey string `json:"bootstrapMasterSshPublicKey" db:"bootstrap_master_ssh_pub_key" validate:"range=0:500"`
}

// NodeOnboardInfo is object that relays post onboard info.
//
// swagger:model NodeOnboardInfo
type NodeOnboardInfo struct {
	// required: true
	NodeID string `json:"id"`
	// required: true
	SSHPublicKey string `json:"sshPublicKey" db:"ssh_pub_key" validate:"range=0:500"`
	// required: true
	NodeVersion string `json:"nodeVersion"`
}

// ToEdge converts Node into Edge model
func (node Node) ToEdge() Edge {
	return Edge{
		BaseModel: node.BaseModel,
		EdgeCore: EdgeCore{
			EdgeCoreCommon: EdgeCoreCommon{
				Name:         node.NodeCore.Name,
				SerialNumber: node.NodeCore.SerialNumber,
				IPAddress:    node.NodeCore.IPAddress,
				Gateway:      node.NodeCore.Gateway,
				Subnet:       node.NodeCore.Subnet,
				Role:         node.NodeCore.Role,
			},
			StorageCapacity: 0,
			StorageUsage:    0,
		},
		Description: node.Description,
		// See how to fill labels
	}
}

// ToEdgeV2 converts Node into EdgeV2
func (node Node) ToEdgeV2() EdgeV2 {
	return EdgeV2{
		BaseModel: node.BaseModel,
		EdgeCoreCommon: EdgeCoreCommon{
			Name:         node.NodeCore.Name,
			SerialNumber: node.NodeCore.SerialNumber,
			IPAddress:    node.NodeCore.IPAddress,
			Gateway:      node.NodeCore.Gateway,
			Subnet:       node.NodeCore.Subnet,
			Role:         node.NodeCore.Role,
		},
		Description: node.Description,
		// See how to fill labels
	}
}

// NodeCreateParam is Node used as API parameter
// swagger:parameters NodeCreate
// in: body
type NodeCreateParam struct {
	// Parameters and values used when creating a node
	// in: body
	// required: true
	Body *Node `json:"body"`
}

// NodeUpdateParam is Node used as API parameter
// swagger:parameters NodeUpdate
// in: body
type NodeUpdateParam struct {
	// in: body
	// required: true
	Body *Node `json:"body"`
}

// Ok
// swagger:response NodeGetResponse
type NodeGetResponse struct {
	// in: body
	// required: true
	Payload *Node
}

// Ok
// swagger:response NodeListResponse
type NodeListResponse struct {
	// in: body
	// required: true
	Payload *NodeListPayload
}

// NodeListPayload is the payload for NodeListResponse
type NodeListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge devices
	// required: true
	NodeList []Node `json:"result"`
}

// swagger:parameters NodeList ProjectGetNodes NodeGet NodeDelete NodeCreate NodeUpdate
// in: header
type nodeAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// NodeSerialNumberPayload payload to get a node by serial number
// swagger:model NodeSerialNumberPayload
type NodeSerialNumberPayload struct {
	//
	// Node serial number
	//
	// required: true
	NodeSerialNumber string `json:"serialNumber"`
}

// swagger:parameters
// in: body
type NodeGetBySerialNumberParam struct {
	// JSON { serialNumber: string }
	// in: body
	// required: true
	Body *SerialNumberPayload `json:"body"`
}

// SerialNumberPayload payload for get edge by serial number
// swagger:model SerialNumberPayload
type SerialNumberPayload struct {
	//
	// Edge serial number
	//
	// required: true
	SerialNumber string `json:"serialNumber"`
}

// Ok
// swagger:response NodeGetBySerialNumberResponse
type NodeGetBySerialNumberResponse struct {
	// in: body
	// required: true
	Payload *EdgeDeviceWithClusterInfo
}

// swagger:parameters NodeOnboarded
// in: body
type NodeOnboardedByIDParam struct {

	// in: body
	// required: true
	Body *NodeOnboardInfo `json:"body"`
}

// ObjectRequestBaseEdgeDevice is used as websocket Node message
// swagger:model ObjectRequestBaseNode
type ObjectRequestBaseNode struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Node `json:"doc"`
}

// ResponseBaseNode is used as websocket reportNode response
// swagger:model ResponseBaseNode
type ResponseBaseNode struct {
	// required: true
	ResponseBase
	// required: true
	Doc Node `json:"doc"`
}

type UpdateNodeMessage struct {
	Doc      Node
	Projects []Project
}

type NodesByID []Node

func (a NodesByID) Len() int           { return len(a) }
func (a NodesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a NodesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// Node must implement this e.g for auth filter
func (doc Node) GetClusterID() string {
	return doc.SvcDomainID
}

// IsK8sMaster returns true if the node has the role of k8s master
func IsK8sMaster(model *Node) bool {
	if model == nil {
		return false
	}
	return model.Role.Master
}

func ValidateNode(model *Node) error {
	if model == nil {
		return errcode.NewBadRequestError("Node")
	}
	if model.SvcDomainID == "" {
		return errcode.NewBadRequestError("Service domain ID")
	}

	model.SerialNumber = strings.TrimSpace(model.SerialNumber)
	model.Description = strings.TrimSpace(model.Description)

	model.IPAddress = strings.TrimSpace(model.IPAddress)
	if base.IsValidIP4(model.IPAddress) == false {
		return errcode.NewMalformedBadRequestError("IPAddress")
	}

	model.Gateway = strings.TrimSpace(model.Gateway)
	if base.IsValidIP4(model.Gateway) == false {
		return errcode.NewMalformedBadRequestError("Gateway")
	}

	model.Subnet = strings.TrimSpace(model.Subnet)
	if base.IsValidIP4(model.Subnet) == false {
		return errcode.NewMalformedBadRequestError("Subnet")
	}

	// Keeping this check for now, can decide if its not needed later
	// DNS-1123 standard
	// see: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	model.Name = strings.TrimSpace(model.Name)
	if dns1123Regexp.MatchString(model.Name) == false {
		return errcode.NewMalformedBadRequestError("Name")
	}

	return nil
}

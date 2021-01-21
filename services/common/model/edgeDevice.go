package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"strings"
)

// EdgeDeviceCore holds the core properties of an edge device
type EdgeDeviceCore struct {
	//
	// Edge name.
	// Maximum length edge name is determined by kubernetes.
	// Name length limited to 60 as node name is the edge name plus a suffix.
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:60"`
	//
	// Edge device serial number
	//
	// required: true
	SerialNumber string `json:"serialNumber" db:"serial_number" validate:"range=0:200"`
	//
	// Edge device IP Address
	//
	// required: true
	IPAddress string `json:"ipAddress" db:"ip_address" validate:"range=0:20"`
	//
	// Edge Device Gateway IP address
	//
	// required: true
	Gateway string `json:"gateway" db:"gateway" validate:"range=0:20"`
	//
	// Edge subnet mask
	//
	// required: true
	Subnet string `json:"subnet" db:"subnet" validate:"range=0:20"`
	//
	// NodeRole is a json object for passing edge device roles
	//
	// required: false
	Role *NodeRole `json:"role,omitempty"`
}

// EdgeDevice is the DB object and object model for edge devices
//
// EdgeDevice is a node in a Nutanix (Kubernetes) cluster for a tenant.
//
// swagger:model EdgeDevice
type EdgeDevice struct {
	// required: true
	ClusterEntityModel
	// required: true
	EdgeDeviceCore
	//
	// EdgeDevice description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
}

// EdgeDeviceWithClusterInfo is the DB object and object model for edge and indicates if the edge is a bootstrap master
//
// swagger:model EdgeDeviceWithClusterInfo
type EdgeDeviceWithClusterInfo struct {
	// required: true
	EdgeDevice
	// required: true
	IsBootstrapMaster bool `json:"isBootstrapMaster"  db:"is_bootstrap_master"`
	// required: false
	BootstrapMasterSSHPublicKey string `json:"bootstrapMasterSshPublicKey" db:"bootstrap_master_ssh_pub_key" validate:"range=0:500"`
}

// EdgeDeviceOnboardInfo is  object that relays post onboard info.
//
// swagger:model EdgeDeviceOnboardInfo
type EdgeDeviceOnboardInfo struct {
	// required: true
	EdgeDeviceID string `json:"id"`
	// required: true
	SSHPublicKey string `json:"sshPublicKey" db:"ssh_pub_key" validate:"range=0:500"`
}

// ToEdge converts EdgeDevice into Edge model
func (edgeDevice EdgeDevice) ToEdge() Edge {
	return Edge{
		BaseModel: edgeDevice.BaseModel,
		EdgeCore: EdgeCore{
			EdgeCoreCommon: EdgeCoreCommon{
				Name:         edgeDevice.EdgeDeviceCore.Name,
				SerialNumber: edgeDevice.EdgeDeviceCore.SerialNumber,
				IPAddress:    edgeDevice.EdgeDeviceCore.IPAddress,
				Gateway:      edgeDevice.EdgeDeviceCore.Gateway,
				Subnet:       edgeDevice.EdgeDeviceCore.Subnet,
			},
			StorageCapacity: 0,
			StorageUsage:    0,
		},
		Description: edgeDevice.Description,
		// See how to fill labels
	}
}

// ToEdgeV2 converts EdgeDevice into EdgeV2
func (edgeDevice EdgeDevice) ToEdgeV2() EdgeV2 {
	return EdgeV2{
		BaseModel: edgeDevice.BaseModel,
		EdgeCoreCommon: EdgeCoreCommon{
			Name:         edgeDevice.EdgeDeviceCore.Name,
			SerialNumber: edgeDevice.EdgeDeviceCore.SerialNumber,
			IPAddress:    edgeDevice.EdgeDeviceCore.IPAddress,
			Gateway:      edgeDevice.EdgeDeviceCore.Gateway,
			Subnet:       edgeDevice.EdgeDeviceCore.Subnet,
		},
		Description: edgeDevice.Description,
		// See how to fill labels
	}
}

// EdgeDeviceCreateParam is EdgeDevice used as API parameter
// swagger:parameters EdgeDeviceCreate
// in: body
type EdgeDeviceCreateParam struct {
	// Parameters and values used when creating an edge device
	// in: body
	// required: true
	Body *EdgeDevice `json:"body"`
}

// EdgeDeviceUpdateParam is EdgeDevice used as API parameter
// swagger:parameters EdgeDeviceUpdate
// in: body
type EdgeDeviceUpdateParam struct {
	// in: body
	// required: true
	Body *EdgeDevice `json:"body"`
}

// Ok
// swagger:response EdgeDeviceGetResponse
type EdgeDeviceGetResponse struct {
	// in: body
	// required: true
	Payload *EdgeDevice
}

// Ok
// swagger:response EdgeDeviceListResponse
type EdgeDeviceListResponse struct {
	// in: body
	// required: true
	Payload *EdgeDeviceListPayload
}

// payload for EdgeDeviceListResponse
type EdgeDeviceListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge devices
	// required: true
	EdgeDeviceList []EdgeDevice `json:"result"`
}

// swagger:parameters EdgeDeviceList EdgeDeviceGet EdgeDeviceCreate EdgeDeviceUpdate EdgeDeviceDelete ProjectGetEdgeDevices
// in: header
type edgeDeviceAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// EdgeDeviceSerialNumberPayload payload to get an edge device by serial number
// swagger:model EdgeDeviceSerialNumberPayload
type EdgeDeviceSerialNumberPayload struct {
	//
	// Edge serial number
	//
	// required: true
	EdgeDeviceSerialNumber string `json:"serialNumber"`
}

// swagger:parameters EdgeDeviceGetBySerialNumber
// in: body
type EdgeDeviceGetBySerialNumberParam struct {
	// JSON { serialNumber: string }
	// in: body
	// required: true
	Body *SerialNumberPayload `json:"body"`
}

// Ok
// swagger:response EdgeDeviceGetBySerialNumberResponse
type EdgeDeviceGetBySerialNumberResponse struct {
	// in: body
	// required: true
	Payload *EdgeDeviceWithClusterInfo
}

// swagger:parameters EdgeDeviceOnboardedById
// in: body
type EdgeDeviceOnboardedByIdParam struct {

	// in: body
	// required: true
	Body *EdgeDeviceOnboardInfo `json:"body"`
}

// ObjectRequestBaseEdgeDevice is used as websocket Edge Device message
// swagger:model ObjectRequestBaseEdgeDevice
type ObjectRequestBaseEdgeDevice struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc EdgeDevice `json:"doc"`
}

// ResponseBaseEdgeDevice is used as websocket reportEdgeDevice response
// swagger:model ResponseBaseEdge
type ResponseBaseEdgeDevice struct {
	// required: true
	ResponseBase
	// required: true
	Doc EdgeDevice `json:"doc"`
}

type UpdateEdgeDeviceMessage struct {
	Doc      EdgeDevice
	Projects []Project
}

type EdgeDevicesByID []EdgeDevice

func (a EdgeDevicesByID) Len() int           { return len(a) }
func (a EdgeDevicesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EdgeDevicesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (doc EdgeDevice) GetClusterID() string {
	return doc.ClusterID
}

func (edgeDevices EdgeDevicesByID) ToEdgeV2() []EdgeV2 {
	v2Edges := []EdgeV2{}
	for _, edgeDevice := range edgeDevices {
		v2Edges = append(v2Edges, edgeDevice.ToEdgeV2())
	}
	return v2Edges
}

func (edgeDevices EdgeDevicesByID) ToEdge() []Edge {
	Edges := []Edge{}
	for _, edgeDevice := range edgeDevices {
		Edges = append(Edges, edgeDevice.ToEdge())
	}
	return Edges
}

func ValidateEdgeDevice(model *EdgeDevice) error {
	if model == nil {
		return errcode.NewBadRequestError("Edge Device")
	}
	if model.ClusterID == "" {
		return errcode.NewBadRequestError("Edge Cluster ID")
	}
	model.Name = strings.TrimSpace(model.Name)
	model.SerialNumber = strings.TrimSpace(model.SerialNumber)
	model.IPAddress = strings.TrimSpace(model.IPAddress)
	model.Gateway = strings.TrimSpace(model.Gateway)
	model.Subnet = strings.TrimSpace(model.Subnet)
	model.Description = strings.TrimSpace(model.Description)
	if base.IsValidIP4(model.IPAddress) == false {
		return errcode.NewMalformedBadRequestError("IPAddress")
	}
	if base.IsValidIP4(model.Gateway) == false {
		return errcode.NewMalformedBadRequestError("Gateway")
	}
	if base.IsValidIP4(model.Subnet) == false {
		return errcode.NewMalformedBadRequestError("Subnet")
	}

	// Keeping this check for now, can decide if its not needed later
	// DNS-1123 standard
	// see: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	matched := dns1123Regexp.MatchString(model.Name)
	if matched == false {
		return errcode.NewMalformedBadRequestError("Name")
	}

	return nil
}

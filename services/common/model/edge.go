package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"fmt"
	"regexp"
	"strings"
)

// TargetType is the type of the target - edge or virtual in cloud
type TargetType string

const (
	// RealTargetType is the actual physical edge
	RealTargetType = TargetType("EDGE")
	// CloudTargetType is the cloud edge
	CloudTargetType = TargetType("CLOUD")
	// KubernetesClusterTargetType is the kubernetes cluster
	KubernetesClusterTargetType = TargetType("KUBERNETES_CLUSTER")

	// EdgeConnectionEventName is the name for edge connection event
	EdgeConnectionEventName = "EdgeConnectionEvent"
)

type EdgeCoreCommon struct {
	//
	// Edge name.
	// Maximum length edge name is determined by kubernetes.
	// Name length limited to 60 as node name is the edge name plus a suffix.
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:60"`
	//
	// Edge serial number
	//
	// required: true
	SerialNumber string `json:"serialNumber" db:"serial_number" validate:"range=0:200"`
	//
	// Edge IP Address
	//
	// required: true
	IPAddress string `json:"ipAddress" db:"ip_address" validate:"range=0:20"`
	//
	// Edge Gateway IP address
	//
	// required: true
	Gateway string `json:"gateway" db:"gateway" validate:"range=0:20"`
	//
	// Edge subnet mask
	//
	// required: true
	Subnet string `json:"subnet" db:"subnet" validate:"range=0:20"`
	//
	// NodeRole is a json object for passing edge roles
	//
	// required: false
	Role *NodeRole `json:"role,omitempty"`
	//
	// Number of devices (nodes) in this edge
	//
	// required: true
	EdgeDevices float64 `json:"edgeDevices" db:"edge_devices"`

	//
	// ShortID is the unique ID for the given edge.
	// This ID must be unique for each edge, for the given tenant.
	// required: false
	ShortID *string `json:"shortId" db:"short_id"`
}

type EdgeCore struct {
	EdgeCoreCommon
	//
	// Edge storage capacity in GB
	//
	// required: true
	StorageCapacity float64 `json:"storageCapacity" db:"storage_capacity"`
	//
	// Edge storage usage in GB
	//
	// required: true
	StorageUsage float64 `json:"storageUsage" db:"storage_usage"`
}

// Edge is the DB object and object model for edge
//
// An Edge is a Nutanix (Kubernetes) cluster for a tenant.
//
// swagger:model Edge
type Edge struct {
	// required: true
	BaseModel
	// required: true
	EdgeCore
	//
	// Edge type.
	//
	Type *string `json:"type,omitempty" db:"type"`
	//
	// Determines if the edge is currently connected to XI IoT management services.
	//
	Connected bool `json:"connected,omitempty" db:"connected"`
	//
	// Edge description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// A list of Category labels for this edge.
	//
	Labels []CategoryInfo `json:"labels"`
}

// Edge is the DB object and object model for edge
//
// An Edge is a Nutanix (Kubernetes) cluster for a tenant.
//
// swagger:model EdgeV2
type EdgeV2 struct {
	// required: true
	BaseModel
	// required: true
	EdgeCoreCommon
	//
	// Type of edge.
	//
	Type *string `json:"type,omitempty" db:"type"`
	//
	// Determines if the edge is currently connected to XI IoT management services.
	//
	Connected bool `json:"connected,omitempty" db:"connected"`
	//
	// Edge description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	//
	// A list of Category labels for this edge.
	//
	Labels []CategoryInfo `json:"labels"`
}

func (edge Edge) ToV2() EdgeV2 {
	return EdgeV2{
		BaseModel:      edge.BaseModel,
		EdgeCoreCommon: edge.EdgeCore.EdgeCoreCommon,
		Type:           edge.Type,
		Connected:      edge.Connected,
		Description:    edge.Description,
		Labels:         edge.Labels,
	}
}

func (edge EdgeV2) FromV2() Edge {
	return Edge{
		BaseModel: edge.BaseModel,
		EdgeCore: EdgeCore{
			EdgeCoreCommon:  edge.EdgeCoreCommon,
			StorageCapacity: 0,
			StorageUsage:    0,
		},
		Type:        edge.Type,
		Connected:   edge.Connected,
		Description: edge.Description,
		Labels:      edge.Labels,
	}
}

// ToEdgeDevice converts edge to EdgeDevice
// We use the same EdgeClusterId ad the EdgeDeviceId
func (edge Edge) ToEdgeDevice() EdgeDevice {
	return EdgeDevice{
		ClusterEntityModel: ClusterEntityModel{
			BaseModel: edge.BaseModel,
			ClusterID: edge.BaseModel.ID,
		},
		EdgeDeviceCore: EdgeDeviceCore{
			Name:         edge.EdgeCore.EdgeCoreCommon.Name,
			SerialNumber: edge.EdgeCore.EdgeCoreCommon.SerialNumber,
			IPAddress:    edge.EdgeCore.EdgeCoreCommon.IPAddress,
			Gateway:      edge.EdgeCore.EdgeCoreCommon.Gateway,
			Subnet:       edge.EdgeCore.EdgeCoreCommon.Subnet,
		},
		Description: edge.Description,
	}
}

// ToEdgeDevice converts EdgeV2 into EdgeDevice
func (edge EdgeV2) ToEdgeDevice() EdgeDevice {
	return EdgeDevice{
		ClusterEntityModel: ClusterEntityModel{
			BaseModel: edge.BaseModel,
			ClusterID: edge.BaseModel.ID,
		},
		EdgeDeviceCore: EdgeDeviceCore{
			Name:         edge.EdgeCoreCommon.Name,
			SerialNumber: edge.EdgeCoreCommon.SerialNumber,
			IPAddress:    edge.EdgeCoreCommon.IPAddress,
			Gateway:      edge.EdgeCoreCommon.Gateway,
			Subnet:       edge.EdgeCoreCommon.Subnet,
		},
		Description: edge.Description,
	}
}

// ToEdgeCluster converts Edge to EdgeCluster
func (edge Edge) ToEdgeCluster() EdgeCluster {
	return EdgeCluster{
		BaseModel: edge.BaseModel,
		EdgeClusterCore: EdgeClusterCore{
			Name:    edge.EdgeCore.EdgeCoreCommon.Name,
			ShortID: edge.EdgeCoreCommon.ShortID,
			Type:    edge.Type,
		},
		Labels:      edge.Labels,
		Description: edge.Description,
		Connected:   edge.Connected,
	}
}

// ToEdgeCluster converts EdgeV2 to EdgeCluster
func (edge EdgeV2) ToEdgeCluster() EdgeCluster {
	return EdgeCluster{
		BaseModel: edge.BaseModel,
		EdgeClusterCore: EdgeClusterCore{
			Name:    edge.EdgeCoreCommon.Name,
			ShortID: edge.EdgeCoreCommon.ShortID,
			Type:    edge.Type,
		},
		Labels:      edge.Labels,
		Description: edge.Description,
		Connected:   edge.Connected,
	}
}

// EdgeCreateParam is Edge used as API parameter
// swagger:parameters EdgeCreate
// in: body
type EdgeCreateParam struct {
	// Parameters and values used when creating an edge
	// in: body
	// required: true
	Body *Edge `json:"body"`
}

// EdgeCreateParamV2 is Edge used as API parameter
// swagger:parameters EdgeCreateV2
// in: body
type EdgeCreateParamV2 struct {
	// Parameters and values used when creating an edge
	// in: body
	// required: true
	Body *EdgeV2 `json:"body"`
}

// EdgeUpdateParam is Edge used as API parameter
// swagger:parameters EdgeUpdate EdgeUpdateV2
// in: body
type EdgeUpdateParam struct {
	// in: body
	// required: true
	Body *Edge `json:"body"`
}

// EdgeUpdateParamV2 is Edge used as API parameter
// swagger:parameters EdgeUpdateV3
// in: body
type EdgeUpdateParamV2 struct {
	// in: body
	// required: true
	Body *EdgeV2 `json:"body"`
}

// Ok
// swagger:response EdgeGetResponse
type EdgeGetResponse struct {
	// in: body
	// required: true
	Payload *Edge
}

// Ok
// swagger:response EdgeGetResponseV2
type EdgeGetResponseV2 struct {
	// in: body
	// required: true
	Payload *EdgeV2
}

// Ok
// swagger:response EdgeListResponse
type EdgeListResponse struct {
	// in: body
	// required: true
	Payload *[]Edge
}

// Ok
// swagger:response EdgeListResponseV2
type EdgeListResponseV2 struct {
	// in: body
	// required: true
	Payload *EdgeListPayload
}

// payload for EdgeListResponseV2
type EdgeListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edges
	// required: true
	EdgeListV2 []EdgeV2 `json:"result"`
}

// swagger:parameters EdgeList EdgeListV2 EdgeGet EdgeGetV2 EdgeCreate EdgeCreateV2 EdgeUpdate EdgeUpdateV2 EdgeUpdateV3 EdgeDelete EdgeDeleteV2 ProjectGetEdges ProjectGetEdgesV2
// in: header
type edgeAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// GetHandlePayload payload for get edge handle call
// token: see crypto.GetEdgeHandleToken
type GetHandlePayload struct {
	// required: true
	Token string `json:"token"`
	// required: true
	TenantID string `json:"tenantId"`
}

// EdgeGetHandleParam payload for get edge handle call
// token: see crypto.GetEdgeHandleToken
// swagger:parameters EdgeGetHandle
// in: body
type EdgeGetHandleParam struct {
	// in: body
	// required: true
	Body *GetHandlePayload
}

// Ok
// swagger:response EdgeGetHandleResponse
type EdgeGetHandleResponse struct {
	// in: body
	// required: true
	Payload *EdgeCert
}

// swagger:parameters EdgeGetBySerialNumber
// in: body
type EdgeGetBySerialNumberParam struct {
	// JSON { serialNumber: string }
	// in: body
	// required: true
	Body *SerialNumberPayload `json:"body"`
}

// Ok
// swagger:response EdgeGetBySerialNumberResponse
type EdgeGetBySerialNumberResponse struct {
	// in: body
	// required: true
	Payload *Edge
}

// ObjectRequestBaseEdge is used as websocket Edge message
// swagger:model ObjectRequestBaseEdge
type ObjectRequestBaseEdge struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc Edge `json:"doc"`
}

// ResponseBaseEdge is used as websocket reportEdge response
// swagger:model ResponseBaseEdge
type ResponseBaseEdge struct {
	// required: true
	ResponseBase
	// required: true
	Doc Edge `json:"doc"`
}

type UpdateEdgeMessage struct {
	Doc      Edge
	Projects []Project
}

type EdgesByID []Edge

func (a EdgesByID) Len() int           { return len(a) }
func (a EdgesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EdgesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (edges EdgesByID) ToV2() []EdgeV2 {
	v2Edges := []EdgeV2{}
	for _, edge := range edges {
		v2Edges = append(v2Edges, edge.ToV2())
	}
	return v2Edges
}

type EdgesByIDV2 []EdgeV2

func (a EdgesByIDV2) Len() int           { return len(a) }
func (a EdgesByIDV2) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EdgesByIDV2) Less(i, j int) bool { return a[i].ID < a[j].ID }

func (v2Edges EdgesByIDV2) FromV2() []Edge {
	edges := []Edge{}
	for _, v2Edge := range v2Edges {
		edges = append(edges, v2Edge.FromV2())
	}
	return edges
}

var dns1123Regexp = regexp.MustCompile("^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$")

func ValidateEdge(model *Edge) error {
	if model == nil {
		return errcode.NewBadRequestError("Edge")
	}
	if model.Type != nil {
		if len(*model.Type) == 0 || *model.Type == string(RealTargetType) {
			model.Type = nil
		} else if *model.Type != string(CloudTargetType) {
			return errcode.NewBadRequestError("Type")
		}
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

	// DNS-1123 standard
	// see: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	matched := dns1123Regexp.MatchString(model.Name)
	if matched == false {
		return errcode.NewMalformedBadRequestError("Name")
	}

	return nil
}

func IsLabelsChanged(oldLabels []CategoryInfo, newLabels []CategoryInfo) bool {
	if len(oldLabels) != len(newLabels) {
		return true
	}
	if len(oldLabels) == 0 {
		return false
	}
	m := map[string]bool{}
	for _, ci := range oldLabels {
		m[fmt.Sprintf("%s/%s", ci.ID, ci.Value)] = true
	}
	for _, ci := range newLabels {
		if !m[fmt.Sprintf("%s/%s", ci.ID, ci.Value)] {
			return true
		}
	}
	return false
}

// Event definitions
type EdgeConnectionEvent struct {
	TenantID string
	EdgeID   string
	Status   bool
	ID       string
}

func (event *EdgeConnectionEvent) IsAsync() bool {
	return true
}

func (event *EdgeConnectionEvent) EventName() string {
	return EdgeConnectionEventName
}

func (event *EdgeConnectionEvent) GetID() string {
	return event.ID
}

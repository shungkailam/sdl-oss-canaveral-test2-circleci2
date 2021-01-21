package model

import (
	"cloudservices/common/errcode"
	"fmt"
	"strings"
)

// EdgeCluster is the DB object and object model for edge cluster
//
// An Edge  cluster is a Nutanix (Kubernetes) cluster for a tenant comprising of multiple edge devices.
//
// swagger:model EdgeCluster
type EdgeCluster struct {
	// required: true
	BaseModel
	// required: true
	EdgeClusterCore
	//
	// EdgeCluster description
	//
	Description string `json:"description" db:"description" validate:"range=0:200"`
	// //
	// // List of edge Device IDs in cluster
	// //
	// // required: true
	// EdgeDeviceIds []string `json:"edgeDeviceIds,omitempty"`
	//
	// A list of Category labels for this edge cluster.
	//
	// required: false
	Labels []CategoryInfo `json:"labels"`
	//
	// ntnx:ignore
	// Determines if the edge is currently connected to XI IoT management services.
	//
	Connected bool `json:"connected,omitempty" db:"connected"`
}

type EdgeClusterCore struct {
	//
	// EdgeCluster name.
	// Maximum length edge name is determined by kubernetes.
	// Name length limited to 60 as node name is the edge cluster name plus a suffix.
	// https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	// required: true
	Name string `json:"name" db:"name" validate:"range=1:60"`
	//
	// ntnx:ignore
	// ShortID is the unique ID for the given edge.
	// This ID must be unique for each edge, for the given tenant.
	// required: false
	ShortID *string `json:"shortId" db:"short_id"`
	//
	// ntnx:ignore
	// Edge type.
	//
	Type *string `json:"type,omitempty" db:"type"`
	//
	// Virtual IP
	//
	VirtualIP *string `json:"virtualIp, omitempty" db:"virtual_ip"`
}

// EdgeClusterCreateParam is EdgeCluster used as API parameter
// swagger:parameters EdgeClusterCreate
// in: body
type EdgeClusterCreateParam struct {
	// Parameters and values used when creating an edge cluster
	// in: body
	// required: true
	Body *EdgeCluster `json:"body"`
}

// EdgeClusterUpdateParam is EdgeCluster used as API parameter
// swagger:parameters EdgeClusterUpdate
// in: body
type EdgeClusterUpdateParam struct {
	// in: body
	// required: true
	Body *EdgeCluster `json:"body"`
}

// Ok
// swagger:response EdgeClusterGetResponse
type EdgeClusterGetResponse struct {
	// in: body
	// required: true
	Payload *EdgeCluster
}

// Ok
// swagger:response EdgeClusterListResponse
type EdgeClusterListResponse struct {
	// in: body
	// required: true
	Payload *EdgeClusterListPayload
}

// payload for EdgeClusterListResponse
type EdgeClusterListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge cluster
	// required: true
	EdgeClusterList []EdgeCluster `json:"result"`
}

// swagger:parameters EdgeClusterList EdgeClusterGet EdgeClusterCreate EdgeClusterUpdate EdgeClusterDelete ProjectGetEdgeClusters ClusterGetEdgeDevices ClusterGetEdgeDevicesInfo
// in: header
type edgeClusterAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// GetEdgeClusterHandlePayload payload for get edge cluster handle call
// token: see crypto.GetEdgeHandleToken
type GetEdgeClusterHandlePayload struct {
	// required: true
	Token string `json:"token"`
	// required: true
	TenantID string `json:"tenantId"`
}

// EdgeClusterGetHandleParam payload for get edge cluster handle call
// token: see crypto.GetEdgeHandleToken
// swagger:parameters EdgeClusterGetHandle
// in: body
type EdgeClusterGetHandleParam struct {
	// in: body
	// required: true
	Body *GetEdgeClusterHandlePayload
}

// Ok
// swagger:response EdgeClusterGetHandleResponse
type EdgeClusterGetHandleResponse struct {
	// in: body
	// required: true
	Payload *EdgeCert
}

// ObjectRequestBaseEdgeCluster is used as websocket Edge Cluster message
// swagger:model ObjectRequestBaseEdgeCluster
type ObjectRequestBaseEdgeCluster struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc EdgeCluster `json:"doc"`
}

// ResponseBaseEdgeCluster is used as websocket reportEdgeCluster response
// swagger:model ResponseBaseEdgeCluster
type ResponseBaseEdgeCluster struct {
	// required: true
	ResponseBase
	// required: true
	Doc EdgeCluster `json:"doc"`
}

type UpdateEdgeClusterMessage struct {
	Doc      EdgeCluster
	Projects []Project
}

type EdgeClustersByID []EdgeCluster

func (a EdgeClustersByID) Len() int           { return len(a) }
func (a EdgeClustersByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EdgeClustersByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

func ValidateEdgeCluster(model *EdgeCluster) error {
	if model == nil {
		return errcode.NewBadRequestError("Edge Cluster")
	}

	// Validation copied from edge.go
	if model.Type != nil {
		if len(*model.Type) == 0 || *model.Type == string(RealTargetType) {
			model.Type = nil
		} else if *model.Type != string(CloudTargetType) {
			return errcode.NewBadRequestError("Type")
		}
	}
	model.Name = strings.TrimSpace(model.Name)
	model.Description = strings.TrimSpace(model.Description)

	// DNS-1123 standard
	// see: https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go
	matched := dns1123Regexp.MatchString(model.Name)
	if matched == false {
		return errcode.NewMalformedBadRequestError("Name")
	}

	return nil
}

func HaveLabelsChanged(oldLabels []CategoryInfo, newLabels []CategoryInfo) bool {
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

type EdgeClusterIDLabel struct {
	CategoryInfo
	ID string `db:"edge_id"`
}
type EdgeClusterIDLabels struct {
	ID     string
	Labels []CategoryInfo
}

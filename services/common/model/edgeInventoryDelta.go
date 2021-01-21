package model

import (
	"time"
)

// EntityVersionMetadata contains the database ID and UpdatedAt of an entity
// associated with an edge.
// EntityVersionMetadata is used with the get edge inventory delta payload
// to indicate the entity version, by entity ID, for each entity
// associated with an the edge.
// We use UpdatedAt as opposed to Version since
// Version can get truncated during JSON conversion.
type EntityVersionMetadata struct {
	ID        string    `json:"id" db:"id"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// EdgeInventoryDeltaPayload is the payload used in get edge inventory delta
// HTTP POST call. This payload carries a snapshot of current inventory at the
// edge: For each entity type, it contains EntityVersionMetadata of each entity
// instance at the edge.
type EdgeInventoryDeltaPayload struct {
	Projects            []EntityVersionMetadata
	Applications        []EntityVersionMetadata
	ProjectServices     []EntityVersionMetadata
	DataPipelines       []EntityVersionMetadata
	Functions           []EntityVersionMetadata
	RuntimeEnvironments []EntityVersionMetadata
	MLModels            []EntityVersionMetadata
	CloudProfiles       []EntityVersionMetadata
	ContainerRegistries []EntityVersionMetadata
	Categories          []EntityVersionMetadata
	DataSources         []EntityVersionMetadata
	LogCollectors       []EntityVersionMetadata
	SoftwareUpdates     []EntityVersionMetadata
	SvcInstances        []EntityVersionMetadata
	SvcBindings         []EntityVersionMetadata
	DataDriverInstances []EntityVersionMetadata
}

// EdgeInventoryDeleted is used as part of get edge inventory delta response to
// tell the edge which entities no longer exist in the cloud and should be
// deleted at the edge. For each entity type, the value is a list of entity IDs
// to delete.
type EdgeInventoryDeleted struct {
	Projects            []string
	Applications        []string
	ProjectServices     []string
	DataPipelines       []string
	Functions           []string
	RuntimeEnvironments []string
	MLModels            []string
	CloudProfiles       []string
	ContainerRegistries []string
	Categories          []string
	DataSources         []string
	LogCollectors       []string
	SoftwareUpdates     []string
	SvcInstances        []string
	SvcBindings         []string
	DataDriverInstances []string
}

// EdgeInventoryDetails is used as part of get edge inventory delta response to
// tell the edge which entities have been created or updated. For each entity
// type, the value is a list of full details of object of that type the edge
// should create or update.
type EdgeInventoryDetails struct {
	Projects            []Project
	Applications        []Application
	ProjectServices     []ProjectService
	DataPipelines       []DataStream
	Functions           []Script
	RuntimeEnvironments []ScriptRuntime
	MLModels            []MLModel
	CloudProfiles       []CloudCreds
	ContainerRegistries []ContainerRegistry
	Categories          []Category
	DataSources         []DataSource
	LogCollectors       []LogCollector
	SoftwareUpdates     []SoftwareUpdateServiceDomain
	SvcInstances        []ServiceInstance
	SvcBindings         []ServiceBinding
	DataDriverInstances []DataDriverInstanceInventory
}

// EdgeInventoryDeltaResponse is the response for get edge inventory delta. It
// consists of Deleted, Created, and Updated fields of type
// EdgeInventoryDeleted, EdgeInventoryDetails and EdgeInventoryDetails
// described above.
type EdgeInventoryDeltaResponse struct {
	Deleted EdgeInventoryDeleted
	Created EdgeInventoryDetails
	Updated EdgeInventoryDetails
}

// EntityCategorySelectorInfo is used to describe a single category / value
// assignment to an entity. The EntityVersionMetadata is for the entity, while
// the CategoryID and Value are for the assignment. Example entities that can
// have category assignment: project, application, edge.
type EntityCategorySelectorInfo struct {
	// ID         string    `json:"id" db:"id"`
	// UpdatedAt  time.Time `json:"updatedAt" db:"updated_at"`
	EntityVersionMetadata
	CategoryID string `json:"categoryId" db:"category_id"`
	Value      string `json:"value" db:"value"`
}

// EntityCategoryInfoMetadata is used to describe the complete categories /
// values assignment to an entity. The EntityVersionMetadata is for the entity,
// while the []CategoryInfo is the full categories / values.
type EntityCategoryInfoMetadata struct {
	EntityVersionMetadata
	CategoryInfo []CategoryInfo `json:"categoryInfoList" db:"category_info_list"`
}

// EntityVersionMetadataList is alias for []EntityVersionMetadata to add some
// convenient functions (e.g., GetIDs)
type EntityVersionMetadataList []EntityVersionMetadata

func (evmList EntityVersionMetadataList) GetIDs() (IDs []string) {
	for _, evm := range evmList {
		IDs = append(IDs, evm.ID)
	}
	return IDs
}

// EntityVersionMetadataChangeInfo captures the delta between two
// EntityVersionMetadataList
type EntityVersionMetadataChangeInfo struct {
	Deleted EntityVersionMetadataList
	Created EntityVersionMetadataList
	Updated EntityVersionMetadataList
}

// swagger:parameters GetEdgeInventoryDelta
// in: header
type EdgeInventoryDeltaAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// GetEdgeInventoryDeltaParam specifies POST payload for GetEdgeInventoryDelta API
// swagger:parameters GetEdgeInventoryDelta
type GetEdgeInventoryDeltaParam struct {
	// A description of the get inventory delta request.
	// in: body
	// required: true
	Payload *EdgeInventoryDeltaPayload `json:"body"`
}

// Ok
// swagger:response GetEdgeInventoryDeltaResponse
type GetEdgeInventoryDeltaResponse struct {
	// in: body
	// required: true
	Payload *EdgeInventoryDeltaResponse
}

// EdgeQueryParam carries the Edge ID
// swagger:parameters GetEdgeInventoryDelta
// in: query
type EdgeQueryParam struct {
	// ID of Edge to impersonate. Only applicable if called as infra admin.
	// in: query
	// required: false
	EdgeID int `json:"edgeId"`
}

// GetEntityVersionMetadataChangeInfo calculate EntityVersionMetadataChangeInfo
// between from and to []EntityVersionMetadata.
func GetEntityVersionMetadataChangeInfo(
	from []EntityVersionMetadata, to []EntityVersionMetadata,
) EntityVersionMetadataChangeInfo {
	// array size intentionally not initialized since we expect them
	// to remain empty in typical case
	deleted := []EntityVersionMetadata{}
	updated := []EntityVersionMetadata{}
	created := []EntityVersionMetadata{}
	fromMap := make(map[string]*EntityVersionMetadata, len(from))
	toMap := make(map[string]*EntityVersionMetadata, len(to))
	for i := range from {
		p := &from[i]
		fromMap[p.ID] = p
	}
	for i := range to {
		p := &to[i]
		toMap[p.ID] = p
	}
	for _, f := range from {
		p := toMap[f.ID]
		if p == nil {
			deleted = append(deleted, f)
		} else {
			if !f.UpdatedAt.Equal(p.UpdatedAt) {
				updated = append(updated, f)
			}
		}
	}
	for _, t := range to {
		p := fromMap[t.ID]
		if p == nil {
			created = append(created, t)
		}
	}
	return EntityVersionMetadataChangeInfo{
		Deleted: EntityVersionMetadataList(deleted),
		Created: EntityVersionMetadataList(created),
		Updated: EntityVersionMetadataList(updated),
	}
}

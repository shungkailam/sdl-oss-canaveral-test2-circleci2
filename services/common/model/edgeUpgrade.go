package model

const (
	// UpgradeEventName is the name of the event related to edge device upgrade
	UpgradeEventName = "UpgradeEvent"
)

// EdgeUpgradeCore is the object model for EdgeUpgradeCore.
// swagger:model EdgeUpgradeCore
type EdgeUpgradeCore struct {
	//
	//	This is the release that is avaliable
	//
	// required: true
	Release string `json:"release"`
	//
	// The changes that were made in this release from the previous release
	//
	// required: true
	Changelog string `json:"changelog"`
	// We can also store the release data (its large) or get it from operator service(need to get it again)
}

// EdgeUpgrade is object model for EdgeUpgrade
// swagger:model EdgeUpgrade
type EdgeUpgrade struct {
	BaseModel
	EdgeUpgradeCore
	//
	// List of releases that can be upgraded to the new version
	//
	// required: true
	CompatibleReleases []string `json:"compatibleReleases"`
}

// Ok
// swagger:response EdgeUpgradeGetResponse
type EdgeUpgradeGetResponse struct {
	// in: body
	// required: true
	Payload *EdgeUpgrade
}

// Ok
// swagger:response EdgeUpgradeListResponse
type EdgeUpgradeListResponse struct {
	// in: body
	// required: true
	Payload *[]EdgeUpgradeCore
}

// Ok
// swagger:response EdgeUpgradeListResponseV2
type EdgeUpgradeListResponseV2 struct {
	// in: body
	// required: true
	Payload *EdgeUpgradeListPayload
}

// payload for EdgeUpgradeListResponseV2
type EdgeUpgradeListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of edge upgrades
	// required: true
	EdgeUpgradeCoreList []EdgeUpgradeCore `json:"result"`
}

// Ok
// swagger:response EdgeUpgradeCompatibleListResponse
type EdgeUpgradeCompatibleListResponse struct {
	// in: body
	// required: true
	Payload *[]EdgeUpgradeCore
}

// swagger:parameters EdgeUpgradeList EdgeUpgradeListV2 EdgeUpgradeGet EdgeUpgradeGetV2 EdgeUpgradeCreate EdgeUpgradeCreateV2 EdgeUpgradeUpdate EdgeUpgradeUpdateV2 EdgeUpgradeDelete EdgeUpgradeDeleteV2 EdgeGetUpgrades EdgeGetUpgradesV2
// in: header
type edgeUpgradeAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseEdgeUpgrade is used as websocket EdgeUpgrade message
// swagger:model ObjectRequestBaseEdgeUpgrade
type ObjectRequestBaseEdgeUpgrade struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc EdgeUpgrade `json:"doc"`
}

// UpgradeEvent Event definition
type UpgradeEvent struct {
	TenantID       string
	EdgeID         string
	Err            error
	ReleaseVersion string
	ID             string
	// EventState can be failed or success
	EventState string
	EventType  string
	State      string
	Message    string
}

func (event *UpgradeEvent) IsAsync() bool {
	return true
}

func (event *UpgradeEvent) EventName() string {
	return UpgradeEventName
}

func (event *UpgradeEvent) GetID() string {
	return event.ID
}

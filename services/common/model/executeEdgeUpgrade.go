package model

import (
	"github.com/go-openapi/strfmt"
)

// ExecuteEdgeUpgradeData is object model for ExecuteEdgeUpgrades with data
// swagger:model ExecuteEdgeUpgradeData
type ExecuteEdgeUpgradeData struct {
	// required: true
	BaseModel
	//
	// Release for execute edge upgrade.
	//
	// required: true
	Release string `json:"release"`
	//
	// Data for the execute edge upgrade.
	//
	// required: true
	UpgradeData *strfmt.Base64 `json:"data"`
	//
	// Docker login command
	//
	// required: true
	DockerLogin string `json:"dockerLogin"`
	//
	// EdgeID. ID of the specific edge to upgrade.
	//
	// required: true
	EdgeID string `json:"edgeID"`
	//
	// URL for the edge to get the upgrade from
	//
	// required: true
	UpgradeURL string `json:"upgradeURL"`
}

// ExecuteEdgeUpgrade is object model for ExecuteEdgeUpgrade
// swagger:model ExecuteEdgeUpgrade
type ExecuteEdgeUpgrade struct {
	//
	// Version to upgrade to, for the execute edge upgrade.
	//
	// required: true
	Release string `json:"release"`
	//
	// List of edge IDs to upgrade
	//
	// required: true
	EdgeIDs []string `json:"edgeIds"`
	//
	// ntnx:ignore
	//
	// Force for test to skip upgrade version check
	//
	// required: false
	Force bool `json:"force"`
}

// ExecuteEdgeUpgradeID is object model for ExecuteEdgeUpgradeID
// swagger:model ExecuteEdgeUpgradeID
type ExecuteEdgeUpgradeID struct {
	//
	// Version to upgrade to, for the execute edge upgrade.
	//
	// required: true
	Release string `json:"release"`
	//
	// ntnx:ignore
	// List of edge IDs to upgrade
	//
	// required: true
	EdgeID string `json:"edgeId"`
}

// ExecuteEdgeUpgradeParam is ExecuteEdgeUpgrade used as API parameter
// swagger:parameters ExecuteEdgeUpgrade ExecuteEdgeUpgradeV2
type ExecuteEdgeUpgradeParam struct {
	// This is an execute edge upgrade request description
	// in: body
	// required: true
	Body *ExecuteEdgeUpgrade `json:"body"`
}

// ExecuteEdgeUpgradeIDParam is ExecuteEdgeUpgradeID used as API parameter
// swagger:parameters ExecuteEdgeUpgradeID ExecuteEdgeUpgradeIDV2
type ExecuteEdgeUpgradeIDParam struct {
	// This is an execute edge upgrade request description
	// in: body
	// required: true
	Body *ExecuteEdgeUpgradeID `json:"body"`
}

// swagger:parameters ExecuteEdgeUpgrade ExecuteEdgeUpgradeV2
// in: header
type executeEdgeUpgradeAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

// ObjectRequestBaseExecuteEdgeUpgrade is used as websocket ExecuteEdgeUpgrade message
// swagger:model ObjectRequestBaseExecuteEdgeUpgrade
type ObjectRequestBaseExecuteEdgeUpgrade struct {
	// required: true
	TenantID string `json:"tenantId"`
	// required: true
	Doc ExecuteEdgeUpgradeData `json:"doc"`
}

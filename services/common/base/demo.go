package base

import "strings"

var demoTenantIDs = []string{
	"tenant-id-waldot",
	"tenant-id-numart-stores",
	"tenant-id-smart-retail",
}

var demoTenantIDPrefix = "tid-demo-"
var demoEdgeIDPrefix = "eid-demo-"

// IsDemoTenant returns true if the tenantID is for demo (mock)
func IsDemoTenant(tenantID string) bool {
	// for demo tenant, always mark edge as connected
	for _, demoTenantID := range demoTenantIDs {
		if tenantID == demoTenantID {
			return true
		}
	}
	// Override for demo. Any demo ID must begin with the prefixes below.
	// For .NEXT, we have a mix of real and demo edge.
	if strings.HasPrefix(tenantID, demoTenantIDPrefix) {
		return true
	}
	return false
}

// IsDemoTenantEdge returns true if the (tenantID, edgeID) is for demo (mock)
func IsDemoTenantEdge(tenantID string, edgeID string) bool {
	// for demo tenant, always mark edge as connected.
	for _, demoTenantID := range demoTenantIDs {
		if tenantID == demoTenantID {
			return true
		}
	}
	// Override for demo. Any demo ID must begin with the prefixes below.
	// For .NEXT,we have a mix of real and demo edge.
	if strings.HasPrefix(tenantID, demoTenantIDPrefix) && strings.HasPrefix(edgeID, demoEdgeIDPrefix) {
		return true
	}
	return false
}

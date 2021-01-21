package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/feature"
	"cloudservices/common/model"
	"context"

	"github.com/golang/glog"
)

const (
	minVersionURLUpgrade            = "v1.5.0"
	minVersionHighMemAlert          = "v1.7.0"
	minVersionRealTimeLogs          = "v1.11.0"
	minVersionMultiNodeAware        = "v1.15.0"
	minVersionDownloadAndUpgrade    = "v1.15.0"
	minVersionRemoteSSH             = "v1.15.0"
	minVersionProjectUserKubeConfig = "v99.99.99" // TODO Adjust when feature ships
)

var (
	features = &feature.Features{}
)

func init() {
	features.Add("urlUpgrade", minVersionURLUpgrade, "")
	features.Add("highMemAlert", minVersionHighMemAlert, "")
	features.Add("realTimeLogs", minVersionRealTimeLogs, "")
	features.Add("multiNodeAware", minVersionMultiNodeAware, "")
	features.Add("downloadAndUpgrade", minVersionDownloadAndUpgrade, "")
	features.Add("remoteSSH", minVersionRemoteSSH, "")
	features.Add("projectUserKubeConfig ", minVersionProjectUserKubeConfig, "")

	// dm2 is used to match the node ID and get the service domain ID which is used to select multiple dm1 rows.
	// The IDs in dm1 rows are used in im to find the version is not null.
	// This can be called without supplying tenant ID
	queryMap["SelectOneEdgeDeviceInfo"] = `SELECT im.*, dm1.is_onboarded as "onboarded", dm1.edge_cluster_id as "edge_cluster_id" FROM edge_device_info_model im, edge_device_model dm1, edge_device_model dm2 WHERE im.tenant_id = dm1.tenant_id AND im.tenant_id = dm2.tenant_id AND im.id = dm1.id AND dm1.edge_cluster_id = dm2.edge_cluster_id AND (:tenant_id = '' OR dm2.tenant_id = :tenant_id) AND (:exclude_current_node = false OR dm1.id != :device_id) AND dm2.id = :device_id AND im.edge_version is NOT NULL LIMIT 1`
	// Query to get for a list of service domain IDs
	queryMap["SelectServiceDomainVersions"] = `SELECT distinct im.edge_version as "version", dm.edge_cluster_id as "id" FROM edge_device_info_model im, edge_device_model dm WHERE im.tenant_id = dm.tenant_id AND dm.tenant_id = :tenant_id AND im.id = dm.id AND dm.edge_cluster_id in (:edge_cluster_ids) AND im.edge_version is NOT NULL`
}

// ServiceDomainVersionDBO is the DB model to hold query results
type ServiceDomainVersionDBO struct {
	ID      string  `json:"id" db:"id"`
	Version *string `json:"version" db:"version"`
}

// OneNodeVersionQueryParam is DB query param
type OneNodeVersionQueryParam struct {
	TenantID           string `json:"tenantId" db:"tenant_id"`
	NodeID             string `json:"nodeId" db:"device_id"`
	ExcludeCurrentNode bool   `json:"excludeCurrentNode" db:"exclude_current_node"`
}

// GetFeaturesForVersion returns the features that are available for a node version
func GetFeaturesForVersion(ver string) (model.Features, error) {
	fts := model.Features{}
	err := features.Get(ver, &fts)
	if err != nil {
		glog.Errorf("Failed to get features. Error: %s", err.Error())
		return fts, err
	}
	return fts, nil
}

// GetFeaturesForNode returns the features if found.
// If the version is not found to determine the features, it returns nil without any error.
// Error is reported only when errors are encountered in other operations
func (dbAPI *dbObjectModelAPI) GetFeaturesForNode(ctx context.Context, id string) (*model.Features, error) {
	version, err := dbAPI.GetOneNodeVersion(ctx, "", id, false)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting version for node %s. Error: %s"), id, err.Error())
		return nil, err
	}
	if version == "" {
		glog.Infof(base.PrefixRequestID(ctx, "Version is not set for node %s"), id)
		return nil, nil
	}
	fts, err := GetFeaturesForVersion(version)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting features for version %s for node %s. Error: %s"), version, id, err.Error())
		return nil, err
	}
	return &fts, nil
}

func (dbAPI *dbObjectModelAPI) GetServiceDomainVersions(ctx context.Context, svcDomainIDs []string) (map[string]string, error) {
	versions := map[string]string{}
	if len(svcDomainIDs) == 0 {
		glog.Error(base.PrefixRequestID(ctx, "GetServiceDomainVersions: Invalid service domain IDs"))
		return versions, errcode.NewBadRequestError("ids")
	}
	authCtx, err := base.GetAuthContext(ctx)
	if err != nil {
		return versions, err
	}
	param := ServiceDomainIDsParam{TenantID: authCtx.TenantID, SvcDomainIDs: svcDomainIDs}
	svcDomainVersions := []ServiceDomainVersionDBO{}
	err = dbAPI.QueryIn(ctx, &svcDomainVersions, queryMap["SelectServiceDomainVersions"], param)
	if err != nil {
		glog.Error(base.PrefixRequestID(ctx, "GetServiceDomainVersions: Error in getting service domain versions. Error: %s"), err.Error())
		return versions, err
	}
	for _, svcDomainVersion := range svcDomainVersions {
		if svcDomainVersion.Version != nil {
			versions[svcDomainVersion.ID] = *svcDomainVersion.Version
		}
	}
	return versions, nil
}

// FilterServiceDomainIDsByVersion returns the Service Domain IDs with versions greater than or equal to the  minimum version
func (dbAPI *dbObjectModelAPI) FilterServiceDomainIDsByVersion(ctx context.Context, minSvcDomainVersion string, svcDomainIDs []string, predicate func(svcDomainID, svcDomainVersion string, compResult int) bool) ([]string, error) {
	filteredIDs := make([]string, 0, len(svcDomainIDs))
	if len(svcDomainIDs) == 0 || len(minSvcDomainVersion) == 0 {
		return filteredIDs, nil
	}
	versionsMap, err := dbAPI.GetServiceDomainVersions(ctx, svcDomainIDs)
	if err != nil {
		return filteredIDs, nil
	}
	for _, svcDomainID := range svcDomainIDs {
		// Min Service Domain version is always larger for empty Service Domain version
		// because it is never empty (check above)
		compResult := 1
		svcDomainVersion, ok := versionsMap[svcDomainID]
		if ok {
			compResult, err = base.CompareVersions(minSvcDomainVersion, svcDomainVersion)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in version comparison between %s and %s for Service Domain %s. Error: %s"), minSvcDomainVersion, svcDomainVersion, svcDomainID)
				return filteredIDs, err
			}
		}
		shouldAdd := predicate(svcDomainID, svcDomainVersion, compResult)
		if shouldAdd {
			filteredIDs = append(filteredIDs, svcDomainID)
		}
	}
	return filteredIDs, nil
}

// GetOneNodeVersion gets the version by node ID.
// The param excludeCurrentNode tells whether to consider version from this nodeID or not
func (dbAPI *dbObjectModelAPI) GetOneNodeVersion(ctx context.Context, tenantID, nodeID string, excludeCurrentNode bool) (string, error) {
	if nodeID == "" {
		glog.Error(base.PrefixRequestID(ctx, "GetOneNodeVersion: Invalid node ID"))
		return "", errcode.NewBadRequestError("id")
	}
	nodeInfoDBOs := []NodeInfoDBO{}
	param := OneNodeVersionQueryParam{TenantID: tenantID, NodeID: nodeID, ExcludeCurrentNode: excludeCurrentNode}
	err := dbAPI.Query(ctx, &nodeInfoDBOs, queryMap["SelectOneEdgeDeviceInfo"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "GetOneNodeVersion: DB select failed for node %s. Error: %s\n"), nodeID, err.Error())
		return "", err
	}
	if len(nodeInfoDBOs) == 0 || nodeInfoDBOs[0].NodeVersion == nil {
		return "", nil
	}
	return *nodeInfoDBOs[0].NodeVersion, nil
}

// GetFeaturesForServiceDomains gets the features for a list of service domains.
// Features are not available if the version for the service domain is not set
func (dbAPI *dbObjectModelAPI) GetFeaturesForServiceDomains(ctx context.Context, svcDomainIDs []string) (map[string]*model.Features, error) {
	featuresMap := map[string]*model.Features{}
	versions, err := dbAPI.GetServiceDomainVersions(ctx, svcDomainIDs)
	if err != nil {
		return featuresMap, err
	}
	for _, svcDomainID := range svcDomainIDs {
		if version, ok := versions[svcDomainID]; ok {
			features, err := GetFeaturesForVersion(version)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "GetFeaturesForServiceDomains: Error getting features for service domain %s and version %s. Error: %s"), svcDomainID, version, err.Error())
				// Ignore
				continue
			}
			featuresMap[svcDomainID] = &features
		}
	}
	return featuresMap, nil
}

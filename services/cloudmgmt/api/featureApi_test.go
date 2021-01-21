package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestFeature(t *testing.T) {
	t.Parallel()
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	node := createNode(t, dbAPI, tenantID)
	ctx1, _, _ := makeContext(tenantID, []string{})
	// Teardown
	defer func() {
		dbAPI.DeleteEdgeDevice(ctx1, node.ID, nil)
		dbAPI.DeleteServiceDomain(ctx1, node.SvcDomainID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()
	versions := map[string]*model.Features{
		"v1.5.0":  {URLupgrade: true},
		"v1.7.0":  {HighMemAlert: true, URLupgrade: true},
		"v1.12.0": {RealTimeLogs: true, HighMemAlert: true, URLupgrade: true},
		"v1.15.0": {RealTimeLogs: true, HighMemAlert: true, MultiNodeAware: true, DownloadAndUpgrade: true, URLupgrade: true, RemoteSSH: true},
	}

	t.Run("Features test", func(t *testing.T) {
		t.Log("running GetFeaturesForVersion test")
		for ver, fts := range versions {
			outFts, err := api.GetFeaturesForVersion(ver)
			require.NoError(t, err)
			if !reflect.DeepEqual(&outFts, fts) {
				t.Fatalf("expected %+v, found %+v for version %s", fts, outFts, ver)
			}
		}
	})

	t.Run("Create/Get/Delete NodeInfo", func(t *testing.T) {
		t.Log("running Create/Get/Delete NodeInfo test")
		for ver, fts := range versions {
			doc := model.NodeInfo{
				NodeEntityModel: model.NodeEntityModel{
					ServiceDomainEntityModel: model.ServiceDomainEntityModel{
						BaseModel: model.BaseModel{
							TenantID: tenantID,
							Version:  0,
						},
					},
					NodeID: node.ID,
				},
				NodeInfoCore: model.NodeInfoCore{
					NodeVersion: &ver,
				},
			}
			// CreateNodeInfo also updates
			_, err := dbAPI.CreateNodeInfo(ctx1, &doc, nil)
			require.NoError(t, err)
			outFtsMap, err := dbAPI.GetFeaturesForServiceDomains(ctx1, []string{node.SvcDomainID})
			require.NoError(t, err)
			if len(outFtsMap) != 1 {
				t.Fatal("expected one feature")
			}
			outFtsPtr, ok := outFtsMap[node.SvcDomainID]
			if !ok {
				t.Fatalf("expected feature for %s, found %+v", node.SvcDomainID, outFtsMap)
			}
			if !reflect.DeepEqual(outFtsPtr, fts) {
				t.Fatalf("expected %+v, found %+v for version %s", fts, outFtsPtr, ver)
			}
		}
	})
}

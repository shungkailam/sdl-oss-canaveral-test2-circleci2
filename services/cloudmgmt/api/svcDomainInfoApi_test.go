package api_test

import (
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestServiceDomainInfo(t *testing.T) {
	t.Parallel()
	t.Log("running TestServiceDomainInfo test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	node := createNodeWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	}, "edge", 2)[0]
	svcDomainID := node.SvcDomainID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})
	// Teardown
	defer func() {
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete ServiceDomainInfo", func(t *testing.T) {
		t.Log("running Create/Get/Delete ServiceDomainInfo test")

		// ServiceDomainInfo is already there as a part of node creation
		svcDomainInfo, err := dbAPI.GetServiceDomainInfo(ctx1, svcDomainID)
		require.NoError(t, err)
		t.Logf("get service domain info successful, %+v", svcDomainInfo)
		w := apitesthelper.NewResponseWriter()
		err = dbAPI.SelectAllServiceDomainsInfoW(ctx1, w, nil)
		require.NoError(t, err)
		payload := &model.ServiceDomainInfoListPayload{}
		err = w.GetBody(payload)
		require.NoError(t, err)
		t.Logf("got response %+v", payload)
		svcDomainInfos := payload.SvcDomainInfoList
		if len(svcDomainInfos) != 1 {
			t.Fatalf("expected 1 service domain info, found %d", len(svcDomainInfos))
		}
		svcDomainInfo = svcDomainInfos[0]
		if svcDomainInfo.ID != svcDomainID || svcDomainInfo.SvcDomainID != svcDomainID {
			t.Fatalf("svcDomainInfo.ID != svcDomainID -> %v, svcDomainInfo.SvcDomainID != svcDomainID -> %v", svcDomainInfo.ID != svcDomainID, svcDomainInfo.SvcDomainID != svcDomainID)
		}
		w.Reset()
		svcDomainInfo.Artifacts = map[string]interface{}{"cluster": "test"}
		rdr, err := apitesthelper.StructToReader(&svcDomainInfo)
		require.NoError(t, err)
		err = dbAPI.UpdateServiceDomainInfoW(ctx1, w, rdr, nil)
		require.NoError(t, err)
		updateResp := &model.UpdateDocumentResponseV2{}
		err = w.GetBody(updateResp)
		require.NoError(t, err)
		if updateResp.ID != svcDomainID {
			t.Fatalf("expected response %s, found %s", svcDomainID, updateResp.ID)
		}
		w.Reset()
		svcDomainInfo1 := &model.ServiceDomainInfo{}
		err = dbAPI.GetServiceDomainInfoW(ctx1, svcDomainID, w, nil)
		require.NoError(t, err)
		err = w.GetBody(svcDomainInfo1)
		require.NoError(t, err)
		t.Logf("got response %+v", svcDomainInfo1)
		if !reflect.DeepEqual(svcDomainInfo1.Artifacts, svcDomainInfo.Artifacts) {
			t.Fatalf("expected response %+v, found %+v", svcDomainInfo.Artifacts, svcDomainInfo1.Artifacts)
		}
		w.Reset()
		err = dbAPI.SelectAllServiceDomainsInfoForProjectW(ctx1, projectID, w, nil)
		require.NoError(t, err)
		payload = &model.ServiceDomainInfoListPayload{}
		err = w.GetBody(payload)
		require.NoError(t, err)
		t.Logf("got response %+v", payload)
		svcDomainInfos = payload.SvcDomainInfoList
		if len(svcDomainInfos) != 1 {
			t.Fatalf("expected 1 service domain info, found %d", len(svcDomainInfos))
		}
		svcDomainInfo = svcDomainInfos[0]
		if svcDomainInfo.ID != svcDomainID || svcDomainInfo.SvcDomainID != svcDomainID {
			t.Fatalf("svcDomainInfo.ID != svcDomainID -> %v, svcDomainInfo.SvcDomainID != svcDomainID -> %v", svcDomainInfo.ID != svcDomainID, svcDomainInfo.SvcDomainID != svcDomainID)
		}
		w.Reset()
		err = dbAPI.SelectAllServiceDomainsInfoForProjectW(ctx2, projectID, w, nil)
		require.Error(t, err, "Permission must be denied for this project")
		w.Reset()
		err = dbAPI.SelectAllServiceDomainsInfoForProjectW(ctx3, projectID, w, nil)
		require.NoError(t, err)
		payload = &model.ServiceDomainInfoListPayload{}
		err = w.GetBody(payload)
		require.NoError(t, err)
		t.Logf("got response %+v", payload)
		svcDomainInfos = payload.SvcDomainInfoList
		if len(svcDomainInfos) != 1 {
			t.Fatalf("expected 1 service domain info, found %d", len(svcDomainInfos))
		}
		svcDomainInfo = svcDomainInfos[0]
		if svcDomainInfo.ID != svcDomainID || svcDomainInfo.SvcDomainID != svcDomainID {
			t.Fatalf("svcDomainInfo.ID != svcDomainID -> %v, svcDomainInfo.SvcDomainID != svcDomainID -> %v", svcDomainInfo.ID != svcDomainID, svcDomainInfo.SvcDomainID != svcDomainID)
		}
		w.Reset()
		err = dbAPI.GetServiceDomainInfoW(ctx2, svcDomainID, w, nil)
		require.Error(t, err, "No info must be found")
		nodeInfo, err := dbAPI.GetNodeInfo(ctx1, node.ID)
		require.NoError(t, err)
		nodeInfo.NodeVersion = base.StringPtr("v1.15.0")
		// Update the info
		_, err = dbAPI.CreateNodeInfo(ctx1, &nodeInfo, nil)
		require.NoError(t, err)
		// ServiceDomainInfo is already there as a part of node creation
		svcDomainInfo, err = dbAPI.GetServiceDomainInfo(ctx1, svcDomainID)
		require.NoError(t, err)
		t.Logf("features %+v", svcDomainInfo.Features)
		if !svcDomainInfo.Features.RealTimeLogs {
			t.Fatalf("expected true for RealTimeLogs, found %+v", svcDomainInfo.Features)
		}
		w = apitesthelper.NewResponseWriter()
		err = dbAPI.SelectAllServiceDomainsInfoW(ctx1, w, nil)
		require.NoError(t, err)
		payload = &model.ServiceDomainInfoListPayload{}
		err = w.GetBody(payload)
		require.NoError(t, err)
		t.Logf("got response %+v", payload)
		svcDomainInfos = payload.SvcDomainInfoList
		if len(svcDomainInfos) != 1 {
			t.Fatalf("expected 1 service domain info, found %d", len(svcDomainInfos))
		}
		svcDomainInfo = svcDomainInfos[0]
		if !svcDomainInfo.Features.RealTimeLogs {
			t.Fatalf("expected true for RealTimeLogs, found %+v", svcDomainInfo.Features)
		}
	})
}

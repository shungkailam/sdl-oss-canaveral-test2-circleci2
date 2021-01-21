package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createHTTPServiceProxy(t *testing.T, dbAPI api.ObjectModelAPI, tenantID, projectID, edgeID, duration string) model.HTTPServiceProxy {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	doc := generateHTTPServiceProxy(edgeID, projectID, duration)
	// create http service proxy
	resp, err := dbAPI.CreateHTTPServiceProxy(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create HTTPServiceProxy successful, %s", resp)
	doc.ID = resp.(model.HTTPServiceProxyCreateResponsePayload).ID
	pxy, err := dbAPI.GetHTTPServiceProxy(ctx, doc.ID)
	require.NoError(t, err)
	return pxy
}

func generateHTTPServiceProxy(svcDomainID, projectID, duration string) model.HTTPServiceProxyCreateParamPayload {
	uuid := base.GetUUID()
	t := "PROJECT"
	sns := ""
	if projectID == "" {
		t = "SYSTEM"
		sns = "svc-ns-" + uuid
	}
	return model.HTTPServiceProxyCreateParamPayload{
		ID:               "",
		Name:             "http-service-proxy-name-" + uuid,
		Type:             t,
		ProjectID:        projectID,
		ServiceName:      "svc-name-" + uuid,
		ServicePort:      3000,
		ServiceNamespace: sns,
		SvcDomainID:      svcDomainID,
		Duration:         duration,
	}
}

func TestHTTPServiceProxy(t *testing.T) {
	t.Parallel()
	t.Log("running TestHTTPServiceProxy test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete HTTPServiceProxy", func(t *testing.T) {
		t.Log("running Create/Get/Delete HTTPServiceProxy test")

		proxy := createHTTPServiceProxy(t, dbAPI, tenantID, projectID, edgeID, "60m")
		t.Logf("create http service proxy successful, %s", proxy.Name)

		d2 := model.HTTPServiceProxyUpdateParamPayload{
			Name:     proxy.Name + "-updated",
			Duration: "120m",
		}

		// get auth context, set ID
		authContext, err := base.GetAuthContext(ctx1)
		require.NoError(t, err)
		authContext.ID = proxy.ID

		upResp, err := dbAPI.UpdateHTTPServiceProxy(ctx1, &d2, nil)
		require.NoError(t, err)
		t.Logf("update http service proxy successful, %+v", upResp)

		// get http service proxy
		proxy2, err := dbAPI.GetHTTPServiceProxy(ctx1, proxy.ID)
		require.NoError(t, err)
		t.Logf("get http service proxy successful, %+v", proxy2)

		if proxy2.ID != proxy.ID || proxy2.ServiceName != proxy.ServiceName || proxy2.Name != d2.Name || proxy2.Duration != d2.Duration {
			t.Fatal("http service proxy data mismatch")
		}

		// select all http service proxys
		proxies, err := dbAPI.SelectAllHTTPServiceProxies(ctx1, nil)
		require.NoError(t, err)
		if n := len(proxies.HTTPServiceProxyList); n != 1 {
			t.Fatalf("expect auth 1 proxies count to be 1, got %d", n)
		}
		proxies, err = dbAPI.SelectAllHTTPServiceProxies(ctx2, nil)
		require.NoError(t, err)
		if n := len(proxies.HTTPServiceProxyList); n != 0 {
			t.Fatalf("expect auth 2 proxies count to be 0, got %d", n)
		}
		proxies, err = dbAPI.SelectAllHTTPServiceProxies(ctx3, nil)
		require.NoError(t, err)
		if n := len(proxies.HTTPServiceProxyList); n != 1 {
			t.Fatalf("expect auth 3 proxies count to be 1, got %d", n)
		}

		// select all vs select all W
		var w bytes.Buffer
		srts1, err := dbAPI.SelectAllHTTPServiceProxies(ctx1, nil)
		require.NoError(t, err)
		srts2 := &model.HTTPServiceProxyListPayload{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllHTTPServiceProxiesW, srts2, &w)
		require.NoError(t, err)
		sort.Sort(model.HTTPServiceProxiesByID(srts1.HTTPServiceProxyList))
		sort.Sort(model.HTTPServiceProxiesByID(srts2.HTTPServiceProxyList))
		if !reflect.DeepEqual(srts1.HTTPServiceProxyList, srts2.HTTPServiceProxyList) {
			t.Fatalf("expect select http service proxys and select http service proxys w results to be equal %+v vs %+v", srts1, *srts2)
		}

		// delete http service proxy
		delResp, err := dbAPI.DeleteHTTPServiceProxy(ctx1, proxy.ID, nil)
		require.NoError(t, err)
		t.Logf("delete http service proxy successful, %v", delResp)

		// create http service proxy w/o project id (SYSTEM) - require infra admin
		doc2 := generateHTTPServiceProxy(edgeID, "", "60m")
		resp, err := dbAPI.CreateHTTPServiceProxy(ctx2, &doc2, nil)
		if err == nil {
			t.Fatal("expected create global http service proxy by non infra admin to fail")
		}
		resp, err = dbAPI.CreateHTTPServiceProxy(ctx1, &doc2, nil)
		require.NoError(t, err)

		t.Logf("create global http service proxy successful, %s", resp)

		proxyId2 := resp.(model.HTTPServiceProxyCreateResponsePayload).ID

		// delete http service proxy
		delResp, err = dbAPI.DeleteHTTPServiceProxy(ctx2, proxyId2, nil)
		// delete will not work, but no error either,
		// can tell delete fail from response ID
		require.NoError(t, err)
		if delResp.(model.DeleteDocumentResponse).ID == proxyId2 {
			t.Fatalf("expected delete global http service proxy by non infra admin to fail")
		}
		delResp, err = dbAPI.DeleteHTTPServiceProxy(ctx1, proxyId2, nil)
		require.NoError(t, err)
		if delResp.(model.DeleteDocumentResponse).ID != proxyId2 {
			t.Fatalf("expected delete global http service proxy by infra admin to succeed")
		}
		t.Logf("delete global http service proxy successful, %v", delResp)
	})

	// select all http service proxys
	t.Run("SelectAllHTTPServiceProxys", func(t *testing.T) {
		t.Log("running SelectAllHTTPServiceProxys test")
		proxy := createHTTPServiceProxy(t, dbAPI, tenantID, projectID, edgeID, "60m")
		t.Logf("http proxy created, id=%s", proxy.ID)
		httpServiceProxies, err := dbAPI.SelectAllHTTPServiceProxies(ctx1, nil)
		require.NoError(t, err)
		for _, httpServiceProxy := range httpServiceProxies.HTTPServiceProxyList {
			testForMarshallability(t, httpServiceProxy)
		}
		_, err = dbAPI.DeleteHTTPServiceProxy(ctx1, proxy.ID, nil)
		require.NoError(t, err)
	})

	t.Run("HTTPServiceProxyConversion", func(t *testing.T) {
		t.Log("running HTTPServiceProxyConversion test")
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		httpServiceProxies := []model.HTTPServiceProxy{
			{
				ServiceDomainEntityModel: model.ServiceDomainEntityModel{
					BaseModel: model.BaseModel{
						ID:        "http-service-proxy-id",
						TenantID:  "tenant-id",
						Version:   5,
						CreatedAt: now,
						UpdatedAt: now,
					},
					SvcDomainID: edgeID,
				},
				HTTPServiceProxyCore: model.HTTPServiceProxyCore{
					Name: "http-service-proxy-name",
				},
				ProjectID: "proj-id",
			},
		}
		for _, app := range httpServiceProxies {
			appDBO := api.HTTPServiceProxyDBO{}
			app2 := model.HTTPServiceProxy{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})

}

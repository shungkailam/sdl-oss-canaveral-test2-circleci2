package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	HTTP_SERVICE_PROXIES_PATH = "/v1.0/httpserviceproxies"
)

// create httpserviceproxy
func createHTTPServiceProxy(netClient *http.Client, httpserviceproxy *model.HTTPServiceProxyCreateParamPayload, token string) (model.CreateDocumentResponseV2, string, error) {
	out := model.HTTPServiceProxyCreateResponsePayload{}
	resp := model.CreateDocumentResponseV2{}
	reqID, err := createEntityV2O(netClient, HTTP_SERVICE_PROXIES_PATH, *httpserviceproxy, token, &out)
	if err == nil {
		httpserviceproxy.ID = out.ID
		resp.ID = out.ID
	}
	return resp, reqID, err
}

// update httpserviceproxy
func updateHTTPServiceProxy(netClient *http.Client, httpserviceproxyID string, httpserviceproxy model.HTTPServiceProxyUpdateParamPayload, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", HTTP_SERVICE_PROXIES_PATH, httpserviceproxyID), httpserviceproxy, token)
}

// get httpserviceproxies
func getHTTPServiceProxies(netClient *http.Client, token string, pageIndex int, pageSize int) (model.HTTPServiceProxyListPayload, error) {
	response := model.HTTPServiceProxyListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", HTTP_SERVICE_PROXIES_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

// delete httpserviceproxy
func deleteHTTPServiceProxy(netClient *http.Client, httpserviceproxyID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, HTTP_SERVICE_PROXIES_PATH, httpserviceproxyID, token)
}

// get httpserviceproxy by id
func getHTTPServiceProxyByID(netClient *http.Client, httpserviceproxyID string, token string) (model.HTTPServiceProxy, error) {
	httpserviceproxy := model.HTTPServiceProxy{}
	err := doGet(netClient, HTTP_SERVICE_PROXIES_PATH+"/"+httpserviceproxyID, token, &httpserviceproxy)
	return httpserviceproxy, err
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

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test HTTPServiceProxy", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// Create an edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID

		project := makeExplicitProject(tenantID, []string{}, []string{}, []string{user.ID}, []string{edgeID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		httpserviceproxy := generateHTTPServiceProxy(edgeID, projectID, "60m")
		cr, _, err := createHTTPServiceProxy(netClient, &httpserviceproxy, token)
		require.NoError(t, err)
		t.Logf("create http proxy successful, response: %+v\n", cr)

		httpserviceproxies, err := getHTTPServiceProxies(netClient, token, 0, 20)
		require.NoError(t, err)
		t.Logf("got httpserviceproxies: %+v", httpserviceproxies)
		if n := len(httpserviceproxies.HTTPServiceProxyList); n != 1 {
			t.Fatalf("expected http service proxy count to be 1, got %d", n)
		}
		if id := httpserviceproxies.HTTPServiceProxyList[0].ID; id != httpserviceproxy.ID {
			t.Fatalf("create http service proxy ID mismatch '%s' vs '%s'\n", id, httpserviceproxy.ID)
		}

		d2 := model.HTTPServiceProxyUpdateParamPayload{
			Name:     httpserviceproxy.Name + "-updated",
			Duration: "120m",
		}
		httpserviceproxyID := httpserviceproxy.ID
		t.Logf("update http service proxy with id %s\n", httpserviceproxyID)
		ur, _, err := updateHTTPServiceProxy(netClient, httpserviceproxyID, d2, token)
		require.NoError(t, err)
		if ur.ID != httpserviceproxyID {
			t.Fatal("update http service proxy id mismatch")
		}

		httpserviceproxies, err = getHTTPServiceProxies(netClient, token, 0, 20)
		require.NoError(t, err)
		t.Logf("got updated httpserviceproxies: %+v", httpserviceproxies)
		if n := len(httpserviceproxies.HTTPServiceProxyList); n != 1 {
			t.Fatalf("expected http service proxy count to be 1, got %d", n)
		}
		pxy := httpserviceproxies.HTTPServiceProxyList[0]
		if pxy.Name != d2.Name || pxy.Duration != d2.Duration {
			t.Fatal("updated http service proxy data mismatch")
		}

		resp2, _, err := deleteHTTPServiceProxy(netClient, httpserviceproxyID, token)
		require.NoError(t, err)
		if resp2.ID != httpserviceproxyID {
			t.Fatal("delete http service proxy id mismatch")
		}
		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("project id mismatch in delete")
		}
		// Delete Edge
		resp, _, err = deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
		if resp.ID != edgeID {
			t.Fatal("delete edge id mismatch")
		}
	})
}

func TestHTTPServiceProxyPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestHTTPServiceProxyPaging test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	rand1 := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test HTTPServiceProxy Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// Create an edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID

		project := makeExplicitProject(tenantID, []string{}, []string{}, []string{user.ID}, []string{edgeID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		// randomly create some http service proxys
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d httpserviceproxies...", n)
		for i := 0; i < n; i++ {
			uuid := base.GetUUID()
			httpserviceproxy := model.HTTPServiceProxyCreateParamPayload{
				ID:               "",
				Name:             "http-service-proxy-name-" + uuid,
				Type:             "PROJECT",
				ProjectID:        projectID,
				ServiceName:      "svc-name-" + uuid,
				ServicePort:      3000,
				ServiceNamespace: "",
				SvcDomainID:      edgeID,
				Duration:         "60m",
			}
			_, _, err = createHTTPServiceProxy(netClient, &httpserviceproxy, token)
			require.NoError(t, err)
		}

		httpserviceproxies, err := getHTTPServiceProxies(netClient, token, 0, 100)
		require.NoError(t, err)
		if nn := len(httpserviceproxies.HTTPServiceProxyList); nn != n {
			t.Fatalf("expected httpserviceproxies count to be %d, but got %d", n, nn)
		}
		sort.Sort(model.HTTPServiceProxiesByID(httpserviceproxies.HTTPServiceProxyList))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pHTTPServiceProxies := []model.HTTPServiceProxy{}
		nRemain := n
		t.Logf("fetch %d httpserviceproxies using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			npxys, err := getHTTPServiceProxies(netClient, token, i, pageSize)
			require.NoError(t, err)
			if npxys.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, npxys.PageIndex)
			}
			if npxys.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, npxys.PageSize)
			}
			if npxys.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, npxys.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if nn := len(npxys.HTTPServiceProxyList); nn != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, nn)
			}
			nRemain -= pageSize
			for _, sr := range npxys.HTTPServiceProxyList {
				pHTTPServiceProxies = append(pHTTPServiceProxies, sr)
			}
		}

		// verify paging api gives same result as old api
		hps := httpserviceproxies.HTTPServiceProxyList
		for i := range pHTTPServiceProxies {
			if !reflect.DeepEqual(hps[i], pHTTPServiceProxies[i]) {
				t.Fatalf("expect proxy equal, but %+v != %+v", hps[i], pHTTPServiceProxies[i])
			}
		}
		t.Log("get proxies from paging api gives same result as old api")

		for _, httpserviceproxy := range hps {
			resp, _, err := deleteHTTPServiceProxy(netClient, httpserviceproxy.ID, token)
			require.NoError(t, err)
			if resp.ID != httpserviceproxy.ID {
				t.Fatal("delete httpserviceproxy id mismatch")
			}
		}

		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("project id mismatch in delete")
		}

		// Delete Edge
		resp, _, err = deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
		if resp.ID != edgeID {
			t.Fatal("delete edge id mismatch")
		}
	})
}

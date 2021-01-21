package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net/http"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// create edge
const (
	EDGESINFO_PATH     = "/v1/edgesInfo"
	EDGESINFO_PATH_NEW = "/v1.0/edgesinfo"
)

// get edges
func getEdgesInfo(netClient *http.Client, token string) ([]model.EdgeUsageInfo, error) {
	edges := []model.EdgeUsageInfo{}
	err := doGet(netClient, EDGESINFO_PATH, token, &edges)
	return edges, err
}
func getEdgesInfoNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.EdgeInfoListPayload, error) {
	response := model.EdgeInfoListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", EDGESINFO_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func TestEdgeInfoPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeInfoPaging test")

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

	t.Run("Test EdgeInfo Paging", func(t *testing.T) {
		// login as user to get token
		token := loginUser(t, netClient, user)

		// randomly create some edges
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d edges...", n)
		for i := 0; i < n; i++ {
			_, _, err := createEdgeForTenant(netClient, tenantID, token)
			require.NoError(t, err)
		}

		// get edges
		edges, err := getEdgesInfo(netClient, token)
		require.NoError(t, err)
		if len(edges) != n {
			t.Fatalf("expected edges count to be %d, but got %d", n, len(edges))
		}
		sort.Sort(model.EdgesInfoByID(edges))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pEdges := []model.EdgeUsageInfo{}
		nRemain := n
		t.Logf("fetch %d edges using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nscpts, err := getEdgesInfoNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if nscpts.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, nscpts.PageIndex)
			}
			if nscpts.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, nscpts.PageSize)
			}
			if nscpts.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, nscpts.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(nscpts.EdgeUsageInfoList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nscpts.EdgeUsageInfoList))
			}
			nRemain -= pageSize
			for _, sr := range nscpts.EdgeUsageInfoList {
				pEdges = append(pEdges, sr)
			}
		}

		// verify paging api gives same result as old api
		for i := range pEdges {
			if !reflect.DeepEqual(edges[i], pEdges[i]) {
				t.Fatalf("expect edge equal, but %+v != %+v", edges[i], pEdges[i])
			}
		}
		t.Log("get edges from paging api gives same result as old api")

		for _, edge := range edges {
			resp, _, err := deleteEdge(netClient, edge.ID, token)
			require.NoError(t, err)
			if resp.ID != edge.ID {
				t.Fatal("delete edge id mismatch")
			}
		}

	})

}

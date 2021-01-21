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
	edgeDevices     float64 = 0
	storageCapacity float64 = 0
	storageUsage    float64 = 0
	edgeIP                  = "1.1.1.1"
	edgeSubnet              = "255.255.255.0"
	edgeGateway             = "1.1.1.1"

	storageUsageUpdated float64 = 0
	edgeIPUpdated               = "1.1.1.2"
	EDGES_PATH                  = "/v1/edges"
	EDGES_PATH_NEW              = "/v1.0/edges"
)

func createEdgeForTenant(netClient *http.Client, tenantID string, token string) (model.Edge, string, error) {
	return createTargetForTenant(netClient, tenantID, token, model.RealTargetType)
}

func createTargetForTenant(netClient *http.Client, tenantID string, token string, targetType model.TargetType) (model.Edge, string, error) {
	edgeSerialNumber := base.GetUUID()
	edgeName := fmt.Sprintf("my-test-edge-%s", base.GetUUID())
	edge := model.Edge{
		EdgeCore: model.EdgeCore{
			EdgeCoreCommon: model.EdgeCoreCommon{
				Name:         edgeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    edgeIP,
				Subnet:       edgeSubnet,
				Gateway:      edgeGateway,
				EdgeDevices:  edgeDevices,
			},
			StorageCapacity: storageCapacity,
			StorageUsage:    storageUsage,
		},
		Type:      base.StringPtr(string(targetType)),
		Connected: false,
	}
	resp, reqID, err := createEdge(netClient, &edge, token)
	if err != nil {
		return model.Edge{}, "", err
	}
	// Reset for compatibility
	if targetType == model.RealTargetType {
		edge.Type = nil
	}
	edge.ID = resp.ID
	return edge, reqID, nil
}

func createEdge(netClient *http.Client, edge *model.Edge, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, EDGES_PATH, *edge, token)
	if err != nil {
		edge.ID = resp.ID
	}
	return resp, reqID, err
}

// update edge
func updateEdge(netClient *http.Client, edgeID string, edge model.Edge, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", EDGES_PATH, edgeID), edge, token)
}

// get edges
func getEdges(netClient *http.Client, token string) ([]model.Edge, error) {
	edges := []model.Edge{}
	err := doGet(netClient, EDGES_PATH, token, &edges)
	return edges, err
}

func getEdgesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.EdgeListPayload, error) {
	response := model.EdgeListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", EDGES_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getTargets(netClient *http.Client, token string, typeParam string, pageIndex int, pageSize int) (model.EdgeListPayload, error) {
	response := model.EdgeListPayload{}
	path := fmt.Sprintf("%s?type=%s&pageIndex=%d&pageSize=%d&orderBy=id", EDGES_PATH_NEW, typeParam, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getEdgesForProject(netClient *http.Client, projectID string, token string) ([]model.Edge, error) {
	edges := []model.Edge{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/edges", token, &edges)
	return edges, err
}

// delete edge
func deleteEdge(netClient *http.Client, edgeID string, token string) (model.DeleteDocumentResponse, string, error) {
	fmt.Printf(">>> delete edge, id=%s\n", edgeID)
	return deleteEntity(netClient, EDGES_PATH, edgeID, token)
}

// get edge by id
func getEdgeByID(netClient *http.Client, edgeID string, token string) (model.Edge, error) {
	edge := model.Edge{}
	err := doGet(netClient, EDGES_PATH+"/"+edgeID, token, &edge)
	return edge, err
}

func TestEdge(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdge test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user2 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")
	user3 := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user4 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user4.ID, nil)
		dbAPI.DeleteUser(ctx, user3.ID, nil)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Edge", func(t *testing.T) {
		// login as user to get token

		token := loginUser(t, netClient, user)
		token2 := loginUser(t, netClient, user2)
		token3 := loginUser(t, netClient, user3)
		token4 := loginUser(t, netClient, user4)

		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID
		t.Logf("edge created: %+v", edge)

		createdCloudTarget, _, err := createTargetForTenant(netClient, tenantID, token, model.CloudTargetType)
		require.NoError(t, err)
		// get edges

		// // filter by "all"
		response, err := getTargets(netClient, token, "all", 0, 10)
		require.NoError(t, err)
		if len(response.EdgeListV2) != 2 {
			t.Fatalf("expected edges count to be 2, got %d", len(response.EdgeListV2))
		}

		for _, target := range response.EdgeListV2 {
			if target.Type != nil && *target.Type == string(model.CloudTargetType) {
				if createdCloudTarget.ID != target.ID {
					t.Fatalf("Mismatched ID. Expected %s, found %s", createdCloudTarget.ID, target.ID)
				}
			} else {
				if edge.ID != target.ID {
					t.Fatalf("Mismatched ID. Expected %s, found %s", edge.ID, target.ID)
				}
			}
		}

		// filter by "cloud"
		response, err = getTargets(netClient, token, "cloud", 0, 10)
		require.NoError(t, err)
		if len(response.EdgeListV2) != 1 {
			t.Fatalf("expected edges count to be 1, got %d", len(response.EdgeListV2))
		}
		if createdCloudTarget.ID != response.EdgeListV2[0].ID {
			t.Fatalf("Mismatched ID. Expected %s, found %s", createdCloudTarget.ID, response.EdgeListV2[0].ID)
		}

		cloudTarget, err := getEdgeByID(netClient, createdCloudTarget.ID, token)
		require.NoError(t, err)
		if *cloudTarget.Type != string(model.CloudTargetType) {
			t.Fatalf("Wrong target type found")
		}

		cloudTarget, err = getEdgeByID(netClient, cloudTarget.ID, token)
		require.NoError(t, err)
		if *cloudTarget.Type != string(model.CloudTargetType) {
			t.Fatalf("Wrong target type found")
		}

		// filter by "edge"
		response, err = getTargets(netClient, token, "edge", 0, 10)
		require.NoError(t, err)
		if len(response.EdgeListV2) != 1 {
			t.Fatalf("expected edges count to be 1, got %d", len(response.EdgeListV2))
		}
		if edge.ID != response.EdgeListV2[0].ID {
			t.Fatalf("Mismatched ID. Expected %s, found %s", edge.ID, response.EdgeListV2[0].ID)
		}

		// implicit filter by "edge"
		edges, err := getEdges(netClient, token)
		require.NoError(t, err)
		if len(edges) != 2 {
			t.Fatalf("expected edges count to be 2, got %d", len(edges))
		}

		unmatchedTargets := []model.Edge{}
		expectedTargets := map[string]*model.Edge{edge.ID: &edge, createdCloudTarget.ID: &createdCloudTarget}
		for _, target := range edges {
			expectedTarget, ok := expectedTargets[target.ID]
			if !ok {
				t.Fatalf("Unexpected output %+v", expectedTarget)
			}
			// Fill up the DB generated fields
			expectedTarget.TenantID = target.TenantID
			expectedTarget.Version = target.Version
			expectedTarget.CreatedAt = target.CreatedAt
			expectedTarget.UpdatedAt = target.UpdatedAt
			expectedTarget.ShortID = target.ShortID
			if reflect.DeepEqual(*expectedTarget, target) {
				delete(expectedTargets, target.ID)
			} else {
				unmatchedTargets = append(unmatchedTargets, target)
			}
		}
		if len(expectedTargets) != 0 {
			t.Fatalf("Some targets did not match. Expected ones %+v, found ones %+v", expectedTargets, unmatchedTargets)
		}

		t.Logf("Got edges: %+v", edges)

		edges2, err := getEdges(netClient, token2)
		require.NoError(t, err)
		if len(edges2) != 0 {
			t.Fatalf("expected edges 2 count to be 0, got %d", len(edges2))
		}

		// update edge
		edge.TenantID = ""
		edge.ID = ""
		edge.StorageUsage = storageUsageUpdated
		edge.IPAddress = edgeIPUpdated
		updateResp, _, err := updateEdge(netClient, edgeID, edge, token)
		require.NoError(t, err)
		if updateResp.ID != edgeID {
			t.Fatal("update edge id mismatch")
		}
		t.Logf("update edge response: %+v", updateResp)
		edges, err = getEdges(netClient, token)
		require.NoError(t, err)
		edge.ID = edgeID
		if len(edges) != 2 {
			t.Fatalf("expected edges count to be 2, got %d", len(edges))
		}
		unmatchedTargets = []model.Edge{}
		expectedTargets = map[string]*model.Edge{edge.ID: &edge, createdCloudTarget.ID: &createdCloudTarget}
		for _, target := range edges {
			expectedTarget, ok := expectedTargets[target.ID]
			if !ok {
				t.Fatalf("Unexpected output %+v", expectedTarget)
			}
			// Fill up the DB generated fields
			expectedTarget.TenantID = target.TenantID
			expectedTarget.Version = target.Version
			expectedTarget.CreatedAt = target.CreatedAt
			expectedTarget.UpdatedAt = target.UpdatedAt
			if reflect.DeepEqual(*expectedTarget, target) {
				delete(expectedTargets, target.ID)
			} else {
				unmatchedTargets = append(unmatchedTargets, target)
			}
		}
		if len(expectedTargets) != 0 {
			t.Fatalf("Some targets did not match. Expected ones %+v, found ones %+v", expectedTargets, unmatchedTargets)
		}
		t.Logf("Got edges: %+v", edges)

		// create project
		project := makeExplicitProject(tenantID, nil, nil, []string{user.ID, user4.ID}, []string{edge.ID})
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		edges, err = getEdgesForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatalf("expect edges for project count to be 1, got %d", len(edges))
		}
		t.Logf("Got edges for project: %+v", edges)

		edges, err = getEdgesForProject(netClient, project.ID, token2)
		require.Error(t, err, "expect get edges 2 for project to fail due to RBAC")
		edges, err = getEdgesForProject(netClient, project.ID, token3)
		require.Error(t, err, "expect get edges 3 for project to fail due to RBAC")
		edges, err = getEdgesForProject(netClient, project.ID, token4)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatalf("expect edges 4 for project count to be 1, got %d", len(edges))
		}
		if !reflect.DeepEqual(edge, edges[0]) {
			t.Fatalf("edge 4 not equal. Expected %+v, found %+v", edge, edges[0])
		}

		// project with edge selector type = category - update edge category cause project update notification
		category := model.Category{
			Name:    "test-cat",
			Purpose: "",
			Values:  []string{"v1", "v2"},
		}
		_, _, err = createCategory(netClient, &category, token)
		require.NoError(t, err)

		project2 := makeCategoryProject(tenantID, nil, nil, nil, []model.CategoryInfo{
			{
				ID:    category.ID,
				Value: "v1",
			},
		})
		_, _, err = createProject(netClient, &project2, token)
		require.NoError(t, err)
		t.Logf("created project: %+v", project2)

		pj2, err := getProjectByID(netClient, project2.ID, token)
		require.NoError(t, err)
		if len(pj2.EdgeIDs) != 0 {
			t.Fatalf("expect edge ids count to be 0 for project, got %d", len(pj2.EdgeIDs))
		}

		edge.Labels = []model.CategoryInfo{
			{
				ID:    category.ID,
				Value: "v1",
			},
		}
		edge.ID = ""
		edge.TenantID = ""
		updateResp, _, err = updateEdge(netClient, edgeID, edge, token)
		require.NoError(t, err)
		if updateResp.ID != edgeID {
			t.Fatal("update edge id mismatch")
		}

		pj2, err = getProjectByID(netClient, project2.ID, token)
		require.NoError(t, err)
		if len(pj2.EdgeIDs) != 1 {
			t.Fatalf("expect edge ids count to be 1 for project, got %d", len(pj2.EdgeIDs))
		}

		// delete project
		resp, _, err := deleteProject(netClient, project2.ID, token)
		require.NoError(t, err)
		resp, _, err = deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
			t.Fatal("project id mismatch")
		}

		// delete edge
		resp, _, err = deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
		if resp.ID != edgeID {
			t.Fatal("edge id mismatch")
		}

	})

}

func TestEdgePaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgePaging test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user2 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")
	user3 := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user4 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")

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
		dbAPI.DeleteUser(ctx, user4.ID, nil)
		dbAPI.DeleteUser(ctx, user3.ID, nil)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Edge Paging", func(t *testing.T) {
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
		edges, err := getEdges(netClient, token)
		require.NoError(t, err)
		if len(edges) != n {
			t.Fatalf("expected edges count to be %d, but got %d", n, len(edges))
		}
		sort.Sort(model.EdgesByID(edges))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pEdges := []model.Edge{}
		nRemain := n
		t.Logf("fetch %d edges using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nscpts, err := getEdgesNew(netClient, token, i, pageSize)
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
			if len(nscpts.EdgeListV2) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nscpts.EdgeListV2))
			}
			nRemain -= pageSize
			for _, sr := range model.EdgesByIDV2(nscpts.EdgeListV2).FromV2() {
				pEdges = append(pEdges, sr)
			}
		}

		// verify paging api gives same result as old api
		for i := range pEdges {
			// zero out before compare
			edges[i].StorageCapacity = 0
			edges[i].StorageUsage = 0
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

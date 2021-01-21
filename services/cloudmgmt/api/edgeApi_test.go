package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally

const edgeDevices float64 = 3
const storageCapacity float64 = 100
const storageUsage float64 = 80
const storageUsageUpdated float64 = 90

func createEdge(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.Edge {
	return createEdgeWithLabels(t, dbAPI, tenantID, nil)
}

func createEdgeWithLabels(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, labels []model.CategoryInfo) model.Edge {
	// create edge
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	edge := generateEdge(tenantID, labels, 1)
	resp, err := dbAPI.CreateEdge(ctx, &edge, nil)
	require.NoError(t, err)
	edge.ID = resp.(model.CreateDocumentResponse).ID
	return edge
}

func generateEdge(tenantID string, labels []model.CategoryInfo, n int) model.Edge {
	edgeSerialNumber := base.GetUUID()
	edgeName := "my-test-edge-" + edgeSerialNumber
	edgeSerialNumberLen := len(edgeSerialNumber)
	edgeSerialNumber = strings.ToUpper(edgeSerialNumber[:edgeSerialNumberLen/2]) + edgeSerialNumber[edgeSerialNumberLen/2:]
	edgeIP := "1.1.1." + strconv.Itoa(n)
	edgeSubnet := "255.255.255.0"
	edgeGateway := "1.1.1.1"
	return model.Edge{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
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
		Connected: true,
		Labels:    labels,
	}
}

func TestEdge(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdge test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeID := edge.ID
	edgeSerialNumber := edge.SerialNumber
	// project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []string{edgeID})
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
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Edge", func(t *testing.T) {
		t.Log("running Create/Get/Delete Edge test")

		edgeName := "my-test-edge"

		// edgeIP := "edge-ip"
		edgeIPUpdated := "1.1.1.2"
		edgeSubnet := "255.255.255.0"
		edgeGateway := "1.1.1.1"

		// update edge
		doc2 := model.Edge{
			BaseModel: model.BaseModel{
				ID:       edgeID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeCore: model.EdgeCore{
				EdgeCoreCommon: model.EdgeCoreCommon{
					Name:         edgeName,
					SerialNumber: edgeSerialNumber,
					IPAddress:    edgeIPUpdated,
					Subnet:       edgeSubnet,
					Gateway:      edgeGateway,
					EdgeDevices:  edgeDevices,
				},
				StorageCapacity: storageCapacity,
				StorageUsage:    storageUsageUpdated,
			},
			Connected: true,
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue2,
				},
			},
		}
		// get edge
		edge, err := dbAPI.GetEdge(ctx1, edgeID)
		require.NoError(t, err)
		_, err = dbAPI.GetEdge(ctx2, edgeID)
		require.Error(t, err, "expect get edge 2 to fail for non infra admin")
		_, err = dbAPI.GetEdge(ctx3, edgeID)
		require.NoError(t, err)
		// log.Printf("get edge before update successful, %+v", edge)

		// select all edges
		edges, err := dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatalf("expect edges count 1 to be 1, but got: %d", len(edges))
		}
		edges2, err := dbAPI.SelectAllEdges(ctx2, nil)
		require.NoError(t, err)
		if len(edges2) != 0 {
			t.Fatalf("expect edges count 2 to be 0, but got: %d", len(edges2))
		}
		edges3, err := dbAPI.SelectAllEdges(ctx3, nil)
		require.NoError(t, err)
		if len(edges3) != 1 {
			t.Fatalf("expect edges count 3 to be 1, but got: %d", len(edges3))
		}

		edgeSerialNumberUpdateOk := fmt.Sprintf("%s-UPDATE-ok", doc2.SerialNumber)
		doc2.SerialNumber = edgeSerialNumberUpdateOk
		_, err = dbAPI.UpdateEdge(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			// log.Printf("update edge callback: doc=%+v\n", doc)
			return nil
		})
		require.NoError(t, err)
		// now lock the edge cert
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx1, doc2.ID)
		require.NoError(t, err)
		edgeCert.Locked = true
		_, err = dbAPI.UpdateEdgeCert(ctx1, &edgeCert, nil)
		require.NoError(t, err)
		edgeSerialNumberUpdateNotOk := fmt.Sprintf("%s-UPDATE-not-ok", edgeSerialNumber)
		doc2.SerialNumber = edgeSerialNumberUpdateNotOk
		_, err = dbAPI.UpdateEdge(ctx1, &doc2, nil)
		require.Error(t, err, "expect update edge with modified serial number to fail once edge cert is locked")

		// change serial number only by letter case should be ok for locked edge
		edgeSerialNumberLen := len(edgeSerialNumberUpdateOk)
		edgeSerialNumberUpdateOk = strings.ToLower(edgeSerialNumberUpdateOk[:edgeSerialNumberLen/2]) + strings.ToUpper(edgeSerialNumberUpdateOk[edgeSerialNumberLen/2:])
		doc2.SerialNumber = edgeSerialNumberUpdateOk
		_, err = dbAPI.UpdateEdge(ctx1, &doc2, nil)
		require.NoError(t, err)

		// clean up: unlock the edge cert
		edgeCert.Locked = false
		_, err = dbAPI.UpdateEdgeCert(ctx1, &edgeCert, nil)
		require.NoError(t, err)

		_, err = dbAPI.UpdateEdge(ctx2, &doc2, nil)
		require.Error(t, err, "expect update edge 2 to fail for non infra admin")
		_, err = dbAPI.UpdateEdge(ctx3, &doc2, nil)
		require.Error(t, err, "expect update edge 3 to fail for non infra admin")
		// log.Printf("update edge successful, %+v", upResp)

		// get edge
		edge, err = dbAPI.GetEdge(ctx1, edgeID)
		require.NoError(t, err)
		if edge.ID != edgeID || edge.Name != edgeName || edge.SerialNumber != edgeSerialNumberUpdateOk || edge.IPAddress != edgeIPUpdated {
			if edge.ID != edgeID {
				t.Fatalf("edge id mismatch %s != %s", edge.ID, edgeID)
			}
			if edge.Name != edgeName {
				t.Fatal("edge name mismatch")
			}
			if edge.SerialNumber != edgeSerialNumberUpdateOk {
				t.Fatal("edge serial number mismatch")
			}
			if edge.IPAddress != edgeIPUpdated {
				t.Fatal("edge ip address mismatch")
			}
			if !base.IsDNS1123Label(*edge.ShortID) {
				t.Fatalf("edge shortID does not pass DNS1123 requirements")
			}

			if !base.IsValidLabelValue(*edge.ShortID) {
				t.Fatalf("edge shortID is not a valid k8s label")
			}

			t.Fatal("edge data mismatch")
		}
		edge, err = dbAPI.GetEdge(ctx2, edgeID)
		require.Error(t, err, "expected get edge 2 to fail")
		edge, err = dbAPI.GetEdge(ctx3, edgeID)
		require.Error(t, err, "expect get edge to fail since edge is no longer in project")
		// log.Printf("get edge successful, %+v", edge)

		// select all edges
		edges, err = dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatalf("expect edges count 1 to be 1, but got: %d", len(edges))
		}
		edges2, err = dbAPI.SelectAllEdges(ctx2, nil)
		require.NoError(t, err)
		if len(edges2) != 0 {
			t.Fatalf("expect edges count 2 to be 0, but got: %d", len(edges2))
		}
		edges3, err = dbAPI.SelectAllEdges(ctx3, nil)
		require.NoError(t, err)
		if len(edges3) != 0 {
			t.Fatalf("expect edges count 3 to be 0, but got: %d", len(edges3))
		}
		for _, edge := range edges {
			testForMarshallability(t, edge)
		}
		t.Log("get all edges successful")

		// update one more time
		// update edge
		doc2 = model.Edge{
			BaseModel: model.BaseModel{
				ID:       edgeID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeCore: model.EdgeCore{
				EdgeCoreCommon: model.EdgeCoreCommon{
					Name:         edgeName,
					SerialNumber: edgeSerialNumber,
					IPAddress:    edgeIPUpdated,
					Subnet:       edgeSubnet,
					Gateway:      edgeGateway,
					EdgeDevices:  edgeDevices,
				},
				StorageCapacity: storageCapacity,
				StorageUsage:    storageUsageUpdated,
			},
			Connected: true,
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				},
			},
		}
		_, err = dbAPI.UpdateEdge(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			// log.Printf("update edge callback: doc=%+v\n", doc)
			return nil
		})
		require.NoError(t, err)
		// get edge
		edge, err = dbAPI.GetEdge(ctx1, edgeID)
		require.NoError(t, err)
		_, err = dbAPI.GetEdge(ctx2, edgeID)
		require.Error(t, err, "expect get edge 2 to fail for non infra admin")
		_, err = dbAPI.GetEdge(ctx3, edgeID)
		require.NoError(t, err)
		// log.Printf("get edge before update successful, %+v", edge)

		// select all edges
		edges, err = dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatalf("expect edges count 1 to be 1, but got: %d", len(edges))
		}
		edges2, err = dbAPI.SelectAllEdges(ctx2, nil)
		require.NoError(t, err)
		if len(edges2) != 0 {
			t.Fatalf("expect edges count 2 to be 0, but got: %d", len(edges2))
		}
		edges3, err = dbAPI.SelectAllEdges(ctx3, nil)
		require.NoError(t, err)
		if len(edges3) != 1 {
			t.Fatalf("expect edges count 3 to be 1, but got: %d", len(edges3))
		}

		// select all edges for project
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		edges, err = dbAPI.SelectAllEdgesForProject(newCtx, projectID, nil)
		require.Error(t, err, "expect select all edges 1 for project to fail")
		edges2, err = dbAPI.SelectAllEdgesForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect select all edges 2 for project to fail")
		edges3, err = dbAPI.SelectAllEdgesForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(edges3) != 1 {
			t.Fatalf("expect edges for project count 3 to be 1, but got: %d", len(edges3))
		}

		// get edge by serial number
		edgeSN, err := dbAPI.GetEdgeBySerialNumber(ctx1, strings.ToLower(edgeSerialNumber))
		require.NoError(t, err)
		testForMarshallability(t, edgeSN)

		// get edge by serial number
		edgeSN, err = dbAPI.GetEdgeBySerialNumber(ctx1, strings.ToUpper(edgeSerialNumber))
		require.NoError(t, err)
		testForMarshallability(t, edgeSN)

		// get edge handle
		// assert edge cert is not locked
		ec, err := dbAPI.GetEdgeCertByEdgeID(ctx1, edgeID)
		require.NoError(t, err)
		if ec.Locked {
			t.Fatal("unexpected edge cert locked")
		}

		token, err := crypto.EncryptPassword(edgeID)
		require.NoError(t, err)
		payload := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		edgeCert, err = dbAPI.GetEdgeHandle(ctx1, edgeID, payload)
		require.NoError(t, err, "GetEdgeHandle failed")

		if !edgeCert.Locked {
			t.Fatal("unexpected edge cert NOT locked")
		}
		testForMarshallability(t, edgeCert)
	})

	// select all edges
	t.Run("SelectAllEdges", func(t *testing.T) {
		t.Log("running SelectAllEdges test")
		edges, err := dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		for _, edge := range edges {
			testForMarshallability(t, edge)
		}
	})

	// select all edges
	t.Run("Edge get edge", func(t *testing.T) {
		t.Log("running Edge get edge test")

		authContextE := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      edgeID,
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContextE)
		edges, err := dbAPI.SelectAllEdges(newCtx, nil)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatal("expected some edges")
		}
		edge, err := dbAPI.GetEdge(newCtx, edgeID)
		require.NoError(t, err)
		t.Logf("Got edge: %+v", edge)
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(newCtx, edgeID)
		require.NoError(t, err)
		t.Logf("Got edge cert: %+v", edgeCert)
		projectRoles, err := dbAPI.GetEdgeProjectRoles(newCtx, edgeID)
		require.NoError(t, err)
		t.Logf("Got project roles: %+v", projectRoles)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		edge := generateEdge(tenantID, nil, 2)
		edge.ID = id
		return dbAPI.CreateEdge(ctx1, &edge, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetEdge(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteEdge(ctx1, id, nil)
	}))
}

func objToReader(obj interface{}) (io.Reader, error) {
	objData, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(objData), nil
}

func objToReader2(t *testing.T, obj interface{}) io.Reader {
	objData, err := json.Marshal(obj)
	require.NoError(t, err)
	return bytes.NewReader(objData)
}

func TestEdgeW(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeW test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	tenantID := base.GetUUID()
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)
	authContext1 := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx1 := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
	authContext2 := &base.AuthContext{
		TenantID: tenantID,
		Claims:   jwt.MapClaims{},
	}
	ctx2 := context.WithValue(context.Background(), base.AuthContextKey, authContext2)
	authContext3 := &base.AuthContext{
		TenantID: tenantID,
		Claims:   jwt.MapClaims{},
	}
	ctx3 := context.WithValue(context.Background(), base.AuthContextKey, authContext3)
	// Create tenant object
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	// create tenant
	_, err = dbAPI.CreateTenant(ctx1, &doc, nil)
	require.NoError(t, err)

	// log.Printf("create tenant successful, %s", resp)
	defer dbAPI.DeleteTenant(ctx1, tenantID, nil)

	edgeSerialNumber := base.GetUUID()

	t.Run("Create/Get/Delete Edge", func(t *testing.T) {
		t.Log("running Create/Get/Delete Edge test")

		edgeName := "my-test-edge"

		edgeIP := "1.1.1.1"
		edgeIPUpdated := "1.1.1.2"
		edgeSubnet := "255.255.255.0"
		edgeGateway := "1.1.1.1"

		// Edge object, leave ID blank and let create generate it
		doc := model.Edge{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
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
			Connected: true,
		}

		r, err := objToReader(doc)
		require.NoError(t, err)

		// create edge
		var w bytes.Buffer
		err = dbAPI.CreateEdgeW(ctx1, &w, r, nil)
		require.NoError(t, err)
		resp := model.CreateDocumentResponse{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)
		// log.Printf("create edge successful, %s", resp)

		edgeID := resp.ID

		// create project
		projName := fmt.Sprintf("Where is Waldo-%s", base.GetUUID())
		projDesc := "Find Waldo"

		// Project object, leave ID blank and let create generate it
		project := model.Project{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:               projName,
			Description:        projDesc,
			CloudCredentialIDs: nil,
			DockerProfileIDs:   nil,
			Users:              []model.ProjectUserInfo{},
			EdgeSelectorType:   model.ProjectEdgeSelectorTypeExplicit,
			EdgeIDs:            []string{edgeID},
			EdgeSelectors:      nil,
		}
		r, err = objToReader(project)
		require.NoError(t, err)
		err = dbAPI.CreateProjectW(ctx1, &w, r, nil)
		require.NoError(t, err)
		resp = model.CreateDocumentResponse{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)
		// log.Printf("create project successful, %s", resp)
		projectID := resp.ID

		// assign auth3 project membership
		authContext3.Claims["projects"] = []model.ProjectRole{
			{
				ProjectID: projectID,
				Role:      model.ProjectRoleAdmin,
			},
		}

		// update edge
		doc = model.Edge{
			BaseModel: model.BaseModel{
				ID:       edgeID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeCore: model.EdgeCore{
				EdgeCoreCommon: model.EdgeCoreCommon{
					Name:         edgeName,
					SerialNumber: edgeSerialNumber,
					IPAddress:    edgeIPUpdated,
					Subnet:       edgeSubnet,
					Gateway:      edgeGateway,
					EdgeDevices:  edgeDevices,
				},
				StorageCapacity: storageCapacity,
				StorageUsage:    storageUsageUpdated,
			},
			Connected: true,
		}
		r, err = objToReader(doc)
		require.NoError(t, err)

		err = dbAPI.UpdateEdgeW(ctx1, &w, r, nil)
		require.NoError(t, err)
		upResp := model.UpdateDocumentResponse{}
		err = json.NewDecoder(&w).Decode(&upResp)
		require.NoError(t, err)
		// log.Printf("update edge successful, %+v", upResp)

		// get edge
		err = dbAPI.GetEdgeW(ctx1, edgeID, &w, nil)
		require.NoError(t, err)
		edge := model.Edge{}
		err = json.NewDecoder(&w).Decode(&edge)
		require.NoError(t, err)
		// log.Printf("get edge successful, %+v", edge)

		if edge.ID != edgeID || edge.Name != edgeName || edge.SerialNumber != edgeSerialNumber || edge.IPAddress != edgeIPUpdated {
			t.Fatal("edge data mismatch")
		}
		// get all edges
		// auth 1
		err = dbAPI.SelectAllEdgesW(ctx1, &w, nil)
		require.NoError(t, err)
		edges := []model.Edge{}
		err = json.NewDecoder(&w).Decode(&edges)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatal("expect all edges 1 count to be 1")
		}
		// auth 2
		err = dbAPI.SelectAllEdgesW(ctx2, &w, nil)
		require.NoError(t, err)
		edges = []model.Edge{}
		err = json.NewDecoder(&w).Decode(&edges)
		require.NoError(t, err)
		if len(edges) != 0 {
			t.Fatal("expect all edges 2 count to be 0")
		}
		// auth 3
		err = dbAPI.SelectAllEdgesW(ctx3, &w, nil)
		require.NoError(t, err)
		edges = []model.Edge{}
		err = json.NewDecoder(&w).Decode(&edges)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatal("expect all edges 3 count to be 1")
		}

		// select all vs select all W
		edges1, err := dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		// edges2
		edges2 := &[]model.Edge{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllEdgesW, edges2, &w)
		require.NoError(t, err)
		sort.Sort(model.EdgesByID(edges1))
		sort.Sort(model.EdgesByID(*edges2))
		if !reflect.DeepEqual(&edges1, edges2) {
			t.Fatalf("expect select edges and select edges w results to be equal %+v vs %+v", edges1, *edges2)
		}

		// get all edges for project
		// auth 1
		err = dbAPI.SelectAllEdgesForProjectWV2(ctx1, projectID, &w, nil)
		require.Error(t, err, "expect all edges 1 for project to fail")
		// auth 2
		err = dbAPI.SelectAllEdgesForProjectW(ctx2, projectID, &w, nil)
		require.Error(t, err, "expect all edges 2 for project to fail")
		// auth 3
		err = dbAPI.SelectAllEdgesForProjectW(ctx3, projectID, &w, nil)
		require.NoError(t, err)
		edges = []model.Edge{}
		err = json.NewDecoder(&w).Decode(&edges)
		require.NoError(t, err)
		if len(edges) != 1 {
			t.Fatal("expect all edges 3 for project count to be 1")
		}

		// get edge by serial number
		payload := model.SerialNumberPayload{
			SerialNumber: edgeSerialNumber,
		}
		r, err = objToReader(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetEdgeBySerialNumberW(ctx1, &w, req)
		require.NoError(t, err)
		edge = model.Edge{}
		err = json.NewDecoder(&w).Decode(&edge)
		require.NoError(t, err)
		testForMarshallability(t, edge)

		// get edge handle
		token, err := crypto.EncryptPassword(edgeID)
		require.NoError(t, err)
		payload2 := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		r, err = objToReader(payload2)
		require.NoError(t, err)
		req = httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetEdgeHandleW(ctx1, edgeID, &w, req)
		require.NoError(t, err, "GetEdgeHandle failed")

		edgeCert := model.EdgeCert{}
		err = json.NewDecoder(&w).Decode(&edgeCert)
		require.NoError(t, err)
		testForMarshallability(t, edgeCert)

		// delete project
		err = dbAPI.DeleteProjectW(ctx1, projectID, &w, nil)
		require.NoError(t, err)
		delResp := model.DeleteDocumentResponse{}
		err = json.NewDecoder(&w).Decode(&delResp)
		require.NoError(t, err)
		t.Logf("delete project successful, %v", delResp)

		// delete edge
		err = dbAPI.DeleteEdgeW(ctx1, edgeID, &w, nil)
		require.NoError(t, err)
		delResp = model.DeleteDocumentResponse{}
		err = json.NewDecoder(&w).Decode(&delResp)
		require.NoError(t, err)
		t.Logf("delete edge successful, %v", delResp)

	})

	// select all edges
	t.Run("SelectAllEdges", func(t *testing.T) {
		t.Log("running SelectAllEdges test")
		var w bytes.Buffer
		err := dbAPI.SelectAllEdgesW(ctx1, &w, nil)
		require.NoError(t, err)
		edges := []model.Edge{}
		err = json.NewDecoder(&w).Decode(&edges)
		require.NoError(t, err)
		for _, edge := range edges {
			testForMarshallability(t, edge)
		}
	})
}

/* // comment out this test as it is too specialized // TODO - generalize this
func TestEdgeW2(t *testing.T) {
	t.Log("running TestEdgeW test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	tenantID := "tenant-id-waldot_test"
	projectID := "453a8f34-3ecb-47d6-9516-edc8a5406b77"
	projectID2 := "a7c2ee5acf42a870579fb80812b723b0"
	ctx1, _, ctx3 := makeContext(tenantID, []string{projectID, projectID2})

	// select all edges
	t.Run("SelectAllEdges2", func(t *testing.T) {
		t.Log("running SelectAllEdges2 test")
		pszs := []int{2, 28}
		for _, sz := range pszs {
			for i := 0; i < 2; i++ {
				var w bytes.Buffer
				url := url.URL{
					RawQuery: fmt.Sprintf("pageIndex=%d&pageSize=%d", i, sz),
				}
				r := http.Request{URL: &url}
				err := dbAPI.SelectAllEdgesWV2(ctx1, &w, &r)
				require.NoError(t, err)
				p := model.EdgeListPayload{
					EntityListResponsePayload: model.EntityListResponsePayload{},
					Result: []model.Edge{},
				}
				err = json.NewDecoder(&w).Decode(&p)
				require.NoError(t, err)
				edges := p.Result
				t.Logf(">>> Got response: pageIndex=%d, pageSize=%d, result count=%d, totalCount=%d\n", p.PageIndex, p.PageSize, len(p.Result), p.TotalCount)
				for _, edge := range edges {
					testForMarshallability(t, edge)
				}
			}
		}

	})

	// select all edges as infra admin
	t.Run("SelectAllEdgesInfraAdmin", func(t *testing.T) {
		t.Log("running SelectAllEdgesInfraAdmin test")

		var w bytes.Buffer
		url := url.URL{
			RawQuery: fmt.Sprintf("pageIndex=%d&pageSize=%d", 0, 60),
		}
		r := http.Request{URL: &url}
		err := dbAPI.SelectAllEdgesWV2(ctx1, &w, &r)
		require.NoError(t, err)
		p := model.EdgeListPayload{
			EntityListResponsePayload: model.EntityListResponsePayload{},
			Result: []model.Edge{},
		}
		err = json.NewDecoder(&w).Decode(&p)
		require.NoError(t, err)
		edges := p.Result

		var w2 bytes.Buffer
		err = dbAPI.SelectAllEdgesW(ctx1, &w2, &r)
		require.NoError(t, err)
		edges2 := []model.Edge{}
		err = json.NewDecoder(&w2).Decode(&edges2)
		require.NoError(t, err)
		// sort edges2
		if len(edges) != len(edges2) {
			t.Fatalf("expect length of edges (%d) to equal length of edges2 (%d)", len(edges), len(edges2))
		}
		t.Logf("got %d edges\n", len(edges))
		for i := 0; i < len(edges); i++ {
			if !reflect.DeepEqual(edges[i], edges2[i]) {
				t.Fatal("expected [%d]: %+v to equal %+v\n", i, edges[i], edges2[i])
			}
		}

	})

	// select all edges for project
	t.Run("SelectAllEdgesForProjectW", func(t *testing.T) {
		t.Log("running SelectAllEdgesForProjectW test")

		var w bytes.Buffer
		url := url.URL{
			RawQuery: fmt.Sprintf("pageIndex=%d&pageSize=%d", 0, 56),
		}
		r := http.Request{URL: &url}
		err := dbAPI.SelectAllEdgesForProjectWV2(ctx1, projectID, &w, &r)
		require.NoError(t, err)
		p := model.EdgeListPayload{
			EntityListResponsePayload: model.EntityListResponsePayload{},
			Result: []model.Edge{},
		}
		err = json.NewDecoder(&w).Decode(&p)
		require.NoError(t, err)
		edges := p.Result

		var w2 bytes.Buffer
		err = dbAPI.SelectAllEdgesForProjectW(ctx1, projectID, &w2, &r)
		require.NoError(t, err)
		edges2 := []model.Edge{}
		err = json.NewDecoder(&w2).Decode(&edges2)
		require.NoError(t, err)
		// sort edges2
		sort.Sort(model.EdgesByID(edges2))

		if len(edges) != len(edges2) {
			t.Fatalf("expect length of edges (%d) to equal length of edges2 (%d)", len(edges), len(edges2))
		}
		t.Logf("got %d edges\n", len(edges))
		for i := 0; i < len(edges); i++ {
			if !reflect.DeepEqual(edges[i], edges2[i]) {
				t.Fatalf("expected [%d]: %+v to equal %+v\n", i, edges[i], edges2[i])
			}
		}

	})

	// select all edges as non infra admin
	t.Run("SelectAllEdgesNonInfraAdmin", func(t *testing.T) {
		t.Log("running SelectAllEdgesNonInfraAdmin test")

		var w bytes.Buffer
		url := url.URL{
			RawQuery: fmt.Sprintf("pageIndex=%d&pageSize=%d", 0, 56),
		}
		r := http.Request{URL: &url}
		err := dbAPI.SelectAllEdgesWV2(ctx3, &w, &r)
		require.NoError(t, err)
		p := model.EdgeListPayload{
			EntityListResponsePayload: model.EntityListResponsePayload{},
			Result: []model.Edge{},
		}
		err = json.NewDecoder(&w).Decode(&p)
		require.NoError(t, err)
		edges := p.Result

		var w2 bytes.Buffer
		err = dbAPI.SelectAllEdgesW(ctx3, &w2, &r)
		require.NoError(t, err)
		edges2 := []model.Edge{}
		err = json.NewDecoder(&w2).Decode(&edges2)
		require.NoError(t, err)
		// sort edges2
		sort.Sort(model.EdgesByID(edges2))

		if len(edges) != len(edges2) {
			t.Fatalf("expect length of edges (%d) to equal length of edges2 (%d)", len(edges), len(edges2))
		}
		t.Logf("got %d edges\n", len(edges))
		for i := 0; i < len(edges); i++ {
			if !reflect.DeepEqual(edges[i], edges2[i]) {
				t.Fatal("expected [%d]: %+v to equal %+v\n", i, edges[i], edges2[i])
			}
		}

	})
}
*/

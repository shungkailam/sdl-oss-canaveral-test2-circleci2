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
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally
func createEdgeClusterWithLabels(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, labels []model.CategoryInfo) model.EdgeCluster {
	// create edge cluster
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	edgeCluster := generateEdgeCluster(tenantID, labels)
	resp, err := dbAPI.CreateEdgeCluster(ctx, &edgeCluster, nil)
	require.NoError(t, err)
	edgeCluster.ID = resp.(model.CreateDocumentResponse).ID
	return edgeCluster
}

func generateEdgeCluster(tenantID string, labels []model.CategoryInfo) model.EdgeCluster {
	return model.EdgeCluster{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		EdgeClusterCore: model.EdgeClusterCore{
			Name: "my-test-edge-" + base.GetUUID(),
		},
		Labels: labels,
	}
}

func TestEdgeCluster(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeCluster test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	edgeCluster := createEdgeClusterWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeClusterID := edgeCluster.ID
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
		dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Edge Cluster", func(t *testing.T) {
		t.Log("running Create/Get/Delete Edge Cluster test")

		edgeClusterName := "my-test-edge-cluster"
		edgeDescriptionUpdated := "test edge desc"

		// update edge cluster
		doc2 := model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       edgeClusterID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name: edgeClusterName,
			},
			Description: edgeDescriptionUpdated,
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue2,
				},
			},
		}
		// get edge cluster
		edgeCluster, err := dbAPI.GetEdgeCluster(ctx1, edgeClusterID)
		require.NoError(t, err)
		_, err = dbAPI.GetEdgeCluster(ctx2, edgeClusterID)
		require.Error(t, err, "expect get edge cluster 2 to fail for non infra admin")
		_, err = dbAPI.GetEdgeCluster(ctx3, edgeClusterID)
		require.NoError(t, err)

		// select all edge clusters
		edgeClusters, err := dbAPI.SelectAllEdgeClusters(ctx1, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters, 1)

		edgeClusters2, err := dbAPI.SelectAllEdgeClusters(ctx2, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters2, 0)

		edgeClusters3, err := dbAPI.SelectAllEdgeClusters(ctx3, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters3, 1)

		_, err = dbAPI.UpdateEdgeCluster(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)

		_, err = dbAPI.UpdateEdgeCluster(ctx2, &doc2, nil)
		require.Error(t, err, "expect update edge cluster 2 to fail for non infra admin")

		_, err = dbAPI.UpdateEdge(ctx3, &doc2, nil)
		require.Error(t, err, "expect update edge cluster 3 to fail for non infra admin")

		// get edge cluster
		edgeCluster, err = dbAPI.GetEdgeCluster(ctx1, edgeClusterID)
		require.NoError(t, err)
		require.Equal(t, edgeCluster.ID, edgeClusterID)
		require.Equal(t, edgeCluster.Name, edgeClusterName)
		require.Equal(t, edgeCluster.Description, edgeDescriptionUpdated)

		edgeCluster, err = dbAPI.GetEdgeCluster(ctx2, edgeClusterID)
		require.Error(t, err, "expected get edge cluster 2 to fail")

		edgeCluster, err = dbAPI.GetEdgeCluster(ctx3, edgeClusterID)
		require.Error(t, err, "expect get edge cluster to fail since edge cluster is no longer in project")

		// select all edge clusters
		edgeClusters, err = dbAPI.SelectAllEdgeClusters(ctx1, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters, 1)

		edgeClusters2, err = dbAPI.SelectAllEdgeClusters(ctx2, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters2, 0)

		edgeClusters3, err = dbAPI.SelectAllEdgeClusters(ctx3, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters3, 0)

		for _, edgeCluster := range edgeClusters {
			testForMarshallability(t, edgeCluster)
		}
		t.Log("get all edge clusters successful")

		// update one more time
		// update edge cluster
		doc2 = model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       edgeClusterID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name: edgeClusterName,
			},
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				},
			},
		}
		_, err = dbAPI.UpdateEdgeCluster(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)
		// get edge cluster
		edgeCluster, err = dbAPI.GetEdgeCluster(ctx1, edgeClusterID)
		require.NoError(t, err)

		_, err = dbAPI.GetEdgeCluster(ctx2, edgeClusterID)
		require.Error(t, err, "expect get edgeCluster 2 to fail for non infra admin")

		_, err = dbAPI.GetEdgeCluster(ctx3, edgeClusterID)
		require.NoError(t, err)

		// select all edge clusters
		edgeClusters, err = dbAPI.SelectAllEdgeClusters(ctx1, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters, 1)

		edgeClusters2, err = dbAPI.SelectAllEdgeClusters(ctx2, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters2, 0)

		edgeClusters3, err = dbAPI.SelectAllEdgeClusters(ctx3, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters3, 1)

		// get edge handle
		// assert edge cert is not locked
		ec, err := dbAPI.GetEdgeCertByEdgeID(ctx1, edgeClusterID)
		require.NoError(t, err)
		if ec.Locked {
			t.Fatal("unexpected edge cert locked")
		}

		token, err := crypto.EncryptPassword(edgeClusterID)
		require.NoError(t, err)
		payload := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		edgeCert, err := dbAPI.GetEdgeHandle(ctx1, edgeClusterID, payload)
		require.NoError(t, err, "GetEdgeHandle failed")
		require.True(t, edgeCert.Locked, "unexpected edge cert NOT locked")
		testForMarshallability(t, edgeCert)
	})

	// select all edge clusters
	t.Run("SelectAllEdgeClusters", func(t *testing.T) {
		t.Log("running SelectAllEdgeClusters test")
		edgeClusters, err := dbAPI.SelectAllEdgeClusters(ctx1, nil)
		require.NoError(t, err)
		for _, edgeCluster := range edgeClusters {
			testForMarshallability(t, edgeCluster)
		}
	})

	// select all edge clusters
	t.Run("EdgeCluster get edge cluster", func(t *testing.T) {
		t.Log("running Edge cluster get edge cluster test")

		authContextE := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      edgeClusterID,
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContextE)
		edgeClusters, err := dbAPI.SelectAllEdgeClusters(newCtx, nil)
		require.NoError(t, err)
		require.Len(t, edgeClusters, 1)

		edgeCluster, err := dbAPI.GetEdgeCluster(newCtx, edgeClusterID)
		require.NoError(t, err)
		t.Logf("Got edge cluster: %+v", edgeCluster)

		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(newCtx, edgeClusterID)
		require.NoError(t, err)
		t.Logf("Got edge cert: %+v", edgeCert)

		projectRoles, err := dbAPI.GetEdgeProjectRoles(newCtx, edgeClusterID)
		require.NoError(t, err)
		t.Logf("Got project roles: %+v", projectRoles)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateEdgeCluster(tenantID, nil)
		doc.ID = id
		return dbAPI.CreateEdgeCluster(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetEdgeCluster(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteEdgeCluster(ctx1, id, nil)
	}))
}

func TestEdgeClusterW(t *testing.T) {
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

	t.Run("Create/Get/Delete Edge Cluster", func(t *testing.T) {
		t.Log("running Create/Get/Delete Edge Cluster test")

		edgeClusterName := "my-test-edge-cluster"

		edgeDescription := "edge desc 1"
		edgeDescriptionUpdated := "edge desc 2"

		// Edge Cluster object, leave ID blank and let create generate it
		doc := model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name: edgeClusterName,
			},
			Description: edgeDescription,
		}

		r, err := objToReader(doc)
		require.NoError(t, err)

		// create edge Cluster
		var w bytes.Buffer
		err = dbAPI.CreateEdgeClusterW(ctx1, &w, r, nil)
		require.NoError(t, err)
		resp := model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)

		edgeClusterID := resp.ID
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
			EdgeIDs:            []string{edgeClusterID},
			EdgeSelectors:      nil,
		}
		r, err = objToReader(project)
		require.NoError(t, err)
		err = dbAPI.CreateProjectWV2(ctx1, &w, r, nil)
		require.NoError(t, err)
		resp = model.CreateDocumentResponseV2{}
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

		// update edge Cluster
		doc = model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       edgeClusterID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name: edgeClusterName,
			},
			Description: edgeDescriptionUpdated,
		}
		r, err = objToReader(doc)
		require.NoError(t, err)

		err = dbAPI.UpdateEdgeClusterW(ctx1, &w, r, nil)
		require.NoError(t, err)
		upResp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&upResp)
		require.NoError(t, err)

		// get edge cluster
		err = dbAPI.GetEdgeClusterW(ctx1, edgeClusterID, &w, nil)
		require.NoError(t, err)
		edgeCluster := model.EdgeCluster{}
		err = json.NewDecoder(&w).Decode(&edgeCluster)
		require.NoError(t, err)

		if edgeCluster.ID != edgeClusterID || edgeCluster.Name != edgeClusterName || edgeCluster.Description != edgeDescriptionUpdated {
			t.Fatal("edge cluster data mismatch")
		}
		// get all edge clusters
		// auth 1
		err = dbAPI.SelectAllEdgeClustersW(ctx1, &w, nil)
		require.NoError(t, err)
		edgeClusters := model.EdgeClusterListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeClusters)
		require.NoError(t, err)
		if len(edgeClusters.EdgeClusterList) != 1 {
			t.Fatal("expect all edgeClusters 1 count to be 1")
		}
		// auth 2
		err = dbAPI.SelectAllEdgeClustersW(ctx2, &w, nil)
		require.NoError(t, err)
		edgeClusters = model.EdgeClusterListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeClusters)
		require.NoError(t, err)
		if len(edgeClusters.EdgeClusterList) != 0 {
			t.Fatal("expect all edgeClusters 2 count to be 0")
		}
		// auth 3
		err = dbAPI.SelectAllEdgeClustersW(ctx3, &w, nil)
		require.NoError(t, err)
		edgeClusters = model.EdgeClusterListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeClusters)
		require.NoError(t, err)
		if len(edgeClusters.EdgeClusterList) != 1 {
			t.Fatal("expect all edgeClusters 3 count to be 1")
		}

		// select all vs select all W
		edgeClusters1, err := dbAPI.SelectAllEdgeClusters(ctx1, nil)
		require.NoError(t, err)
		// edgeClusters2
		edgeClusters2 := model.EdgeClusterListPayload{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllEdgeClustersW, &edgeClusters2, &w)
		require.NoError(t, err)
		sort.Sort(model.EdgeClustersByID(edgeClusters1))
		sort.Sort(model.EdgeClustersByID(edgeClusters2.EdgeClusterList))
		if !reflect.DeepEqual(&edgeClusters1, &edgeClusters2.EdgeClusterList) {
			t.Fatalf("expect select edge clusters and select edge clusters w results to be equal %#v vs %#v", edgeClusters1, edgeClusters2.EdgeClusterList)
		}

		// get all edge clusters for project
		// auth 1
		err = dbAPI.SelectAllEdgeClustersForProjectW(ctx1, projectID, &w, nil)
		require.Error(t, err, "expect all edge clusters 1 for project to fail")
		// auth 2
		err = dbAPI.SelectAllEdgeClustersForProjectW(ctx2, projectID, &w, nil)
		require.Error(t, err, "expect all edge clusters 2 for project to fail")
		// auth 3
		err = dbAPI.SelectAllEdgeClustersForProjectW(ctx3, projectID, &w, nil)
		require.NoError(t, err)
		edgeClusters = model.EdgeClusterListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeClusters)
		require.NoError(t, err)
		if len(edgeClusters.EdgeClusterList) != 1 {
			t.Fatal("expect all edgeClusters 3 for project count to be 1")
		}

		// get edge handle
		token, err := crypto.EncryptPassword(edgeClusterID)
		require.NoError(t, err)
		payload2 := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		r, err = objToReader(payload2)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetEdgeHandleW(ctx1, edgeClusterID, &w, req)
		require.NoError(t, err, "GetEdgeHandle failed")

		edgeCert := model.EdgeCert{}
		err = json.NewDecoder(&w).Decode(&edgeCert)
		require.NoError(t, err)
		//
		testForMarshallability(t, edgeCert)

		// delete project
		err = dbAPI.DeleteProjectWV2(ctx1, projectID, &w, nil)
		require.NoError(t, err)
		delResp := model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delResp)
		require.NoError(t, err)
		t.Logf("delete project successful, %v", delResp)

		// delete edge
		err = dbAPI.DeleteEdgeClusterW(ctx1, edgeClusterID, &w, nil)
		require.NoError(t, err)
		delResp = model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delResp)
		require.NoError(t, err)
		t.Logf("delete edge successful, %v", delResp)
	})

	// select all edge Clusters
	t.Run("SelectAllEdges", func(t *testing.T) {
		t.Log("running SelectAllEdgeClusters test")
		var w bytes.Buffer
		err := dbAPI.SelectAllEdgeClustersW(ctx1, &w, nil)
		require.NoError(t, err)
		edgeClusters := model.EdgeClusterListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeClusters)
		require.NoError(t, err)
		for _, edgeCluster := range edgeClusters.EdgeClusterList {
			testForMarshallability(t, edgeCluster)
		}
	})
}

func TestClusterVirtualIP(t *testing.T) {
	t.Parallel()
	t.Logf("running TestClusterVirtualIP")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	edgeDevices := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", 1)
	edgeClusterID := edgeDevices[0].ClusterID
	ctx, _, _ := makeContext(tenantID, []string{})
	// Teardown
	defer func() {
		dbAPI.DeleteEdgeCluster(ctx, edgeClusterID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test cluster virtual IP validation", func(t *testing.T) {
		t.Log("running cluster virtual IP validation test")
		edgeDeviceSerialNumber := base.GetUUID()
		edgeDeviceName := "second-device" + edgeDeviceSerialNumber
		edgeDeviceIP := "1.1.1.10"
		edgeDeviceSubnet := "255.255.255.0"
		edgeDeviceGateway := "1.1.1.1"
		edgeDevice := model.EdgeDevice{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					TenantID: tenantID,
					Version:  5,
				},
				ClusterID: edgeClusterID,
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         edgeDeviceName,
				SerialNumber: edgeDeviceSerialNumber,
				IPAddress:    edgeDeviceIP,
				Subnet:       edgeDeviceSubnet,
				Gateway:      edgeDeviceGateway,
			},
		}
		edgeCluster, err := dbAPI.GetEdgeCluster(ctx, edgeClusterID)
		require.NoError(t, err)
		// Make sure virtual IP is unset before the tests
		if edgeCluster.VirtualIP != nil && len(*edgeCluster.VirtualIP) > 0 {
			t.Fatalf("Virtual IP must not be set for edge cluster %+v", edgeCluster)
		}
		// Create second device in the same cluster without the virtual IP set
		_, err = dbAPI.CreateEdgeDevice(ctx, &edgeDevice, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Virtual IP")

		// Now set the virtual IP
		edgeCluster.VirtualIP = base.StringPtr("10.10.10.30")
		_, err = dbAPI.UpdateEdgeCluster(ctx, &edgeCluster, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.NoError(t, err)
		// Create second device now. It must succeed because virtual IP is set
		i, err := dbAPI.CreateEdgeDevice(ctx, &edgeDevice, nil)
		require.NoError(t, err)
		secondDeviceID := i.(model.CreateDocumentResponse).ID
		// Now, try to unset the virtual IP when there are already two devices. It must fail
		edgeCluster.VirtualIP = nil
		_, err = dbAPI.UpdateEdgeCluster(ctx, &edgeCluster, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.Errorf(t, err, "Virtual IP cannot be unset %+v", edgeCluster)
		// Try to change the virtual IP when there are already two devices. It must succeed
		edgeCluster.VirtualIP = base.StringPtr("10.10.10.31")
		_, err = dbAPI.UpdateEdgeCluster(ctx, &edgeCluster, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.NoError(t, err)
		// Delete the second edge device
		_, err = dbAPI.DeleteEdgeDevice(ctx, secondDeviceID, nil)
		require.NoError(t, err)
		edgeCluster.VirtualIP = nil
		// Now, try to unset the virtual IP when there is only one device. It must succeed
		_, err = dbAPI.UpdateEdgeCluster(ctx, &edgeCluster, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.NoError(t, err)
	})
}

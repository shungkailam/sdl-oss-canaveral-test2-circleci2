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
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally

func createEdgeDevice(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.EdgeDevice {
	return createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", 1)[0]
}

func createEdgeDeviceWithLabelsCommon(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, labels []model.CategoryInfo, edgeClusterType string, numberOfEdgeDevices int) []model.EdgeDevice {
	// create edge device
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	edgeClusterType = strings.ToUpper(edgeClusterType)
	edgeClusterName := "my-test-edge-cluster"
	edgeCluster := model.EdgeCluster{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		EdgeClusterCore: model.EdgeClusterCore{
			Name: edgeClusterName,
			Type: &edgeClusterType,
		},
		Labels: labels,
	}
	if numberOfEdgeDevices > 1 {
		edgeCluster.VirtualIP = base.StringPtr("10.20.1.10")
	}
	resp, err := dbAPI.CreateEdgeCluster(ctx, &edgeCluster, nil)
	require.NoError(t, err)
	edgeCluster.ID = resp.(model.CreateDocumentResponse).ID
	edgeDevices := []model.EdgeDevice{}
	for count := 0; count < numberOfEdgeDevices; count++ {
		edgeDevice := generateEdgeDevice(tenantID, edgeCluster.ID, "1.1.1."+strconv.Itoa(count))
		edgeDevice.Name = edgeDevice.Name + "-" + strconv.Itoa(count)
		resp, err = dbAPI.CreateEdgeDevice(ctx, &edgeDevice, nil)
		require.NoError(t, err)
		edgeDevice.ID = resp.(model.CreateDocumentResponse).ID
		edgeDevices = append(edgeDevices, edgeDevice)
	}
	return edgeDevices
}

func generateEdgeDevice(tenantID string, edgeClusterID string, edgeDeviceIP string) model.EdgeDevice {
	edgeDeviceSerialNumber := base.GetUUID()
	edgeDeviceName := "my-test-edge-device-" + edgeDeviceSerialNumber
	edgeDeviceSerialNumberLen := len(edgeDeviceSerialNumber)
	edgeDeviceSerialNumber = strings.ToUpper(edgeDeviceSerialNumber[:edgeDeviceSerialNumberLen/2]) + edgeDeviceSerialNumber[edgeDeviceSerialNumberLen/2:]
	edgeDeviceSubnet := "255.255.255.0"
	edgeDeviceGateway := "1.1.1.1"

	return model.EdgeDevice{
		ClusterEntityModel: model.ClusterEntityModel{
			BaseModel: model.BaseModel{
				ID:       "",
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
}

func setEdgeDeviceVersion(t *testing.T, dbAPI api.ObjectModelAPI, tenantID, deviceID, version string) {
	ctx := base.GetAdminContextWithTenantID(context.TODO(), tenantID)
	deviceInfo, err := dbAPI.GetEdgeDeviceInfo(ctx, deviceID)
	require.NoError(t, err)
	deviceInfo.EdgeVersion = base.StringPtr(version)
	_, err = dbAPI.CreateEdgeDeviceInfo(ctx, &deviceInfo, nil)
	require.NoError(t, err)
}

func TestEdgeDevice(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeDevice test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	edgeDevice := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}}, "EDGE", 2)[0]
	edgeDeviceID := edgeDevice.ID
	edgeDeviceSerialNumber := edgeDevice.SerialNumber
	edgeClusterID := edgeDevice.ClusterID
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
		dbAPI.DeleteEdgeDevice(ctx1, edgeDeviceID, nil)
		dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Edge Device", func(t *testing.T) {
		t.Log("running Create/Get/Delete Edge Device test")

		edgeDeviceName := "my-test-edge-device"
		edgeClusterName := "my-test-edge-cluster"

		edgeDeviceIPUpdated := "1.1.1.2"
		edgeDeviceSubnet := "255.255.255.0"
		edgeDeviceGateway := "1.1.1.1"

		// update edge device
		doc2 := model.EdgeDevice{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					ID:       edgeDeviceID,
					TenantID: tenantID,
					Version:  5,
				},
				ClusterID: edgeClusterID,
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         edgeDeviceName,
				SerialNumber: edgeDeviceSerialNumber,
				IPAddress:    edgeDeviceIPUpdated,
				Subnet:       edgeDeviceSubnet,
				Gateway:      edgeDeviceGateway,
			},
		}

		// get edge device
		edgeDevice, err := dbAPI.GetEdgeDevice(ctx1, edgeDeviceID)
		require.NoError(t, err)
		_, err = dbAPI.GetEdgeDevice(ctx2, edgeDeviceID)
		require.Error(t, err, "expect get edge device 2 to fail for non infra admin")
		_, err = dbAPI.GetEdgeDevice(ctx3, edgeDeviceID)
		require.NoError(t, err)

		edgeDeviceSerialNumberUpdateOk := fmt.Sprintf("%s-UPDATE-ok", doc2.SerialNumber)
		doc2.SerialNumber = edgeDeviceSerialNumberUpdateOk
		_, err = dbAPI.UpdateEdgeDevice(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)
		// now lock the edge cert
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx1, doc2.ClusterID)
		require.NoError(t, err)
		edgeCert.Locked = true
		_, err = dbAPI.UpdateEdgeCert(ctx1, &edgeCert, nil)
		require.NoError(t, err)
		edgeDeviceSerialNumberUpdateNotOk := fmt.Sprintf("%s-UPDATE-not-ok", edgeDeviceSerialNumber)
		doc2.SerialNumber = edgeDeviceSerialNumberUpdateNotOk
		_, err = dbAPI.UpdateEdge(ctx1, &doc2, nil)
		require.Error(t, err, "expect update edge device with modified serial number to fail once edge cert is locked")

		// change serial number only by letter case should be ok for locked edge device
		edgeDeviceSerialNumberLen := len(edgeDeviceSerialNumberUpdateOk)
		edgeDeviceSerialNumberUpdateOk = strings.ToLower(edgeDeviceSerialNumberUpdateOk[:edgeDeviceSerialNumberLen/2]) +
			strings.ToUpper(edgeDeviceSerialNumberUpdateOk[edgeDeviceSerialNumberLen/2:])
		doc2.SerialNumber = edgeDeviceSerialNumberUpdateOk
		_, err = dbAPI.UpdateEdgeDevice(ctx1, &doc2, nil)
		require.NoError(t, err)

		// clean up: unlock the edge cert
		edgeCert.Locked = false
		_, err = dbAPI.UpdateEdgeCert(ctx1, &edgeCert, nil)
		require.NoError(t, err)

		_, err = dbAPI.UpdateEdgeDevice(ctx2, &doc2, nil)
		require.Error(t, err, "expect update edge device 2 to fail for non infra admin")
		_, err = dbAPI.UpdateEdgeDevice(ctx3, &doc2, nil)
		require.Error(t, err, "expect update edge device 3 to fail for non infra admin")

		// update edge cluster and remove from project
		clusterdoc2 := model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       edgeClusterID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name:      edgeClusterName,
				VirtualIP: base.StringPtr("10.20.1.20"),
			},
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue2,
				},
			},
		}
		_, err = dbAPI.UpdateEdgeCluster(ctx1, &clusterdoc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)

		// get edge Device
		edgeDevice, err = dbAPI.GetEdgeDevice(ctx1, edgeDeviceID)
		require.NoError(t, err)
		if edgeDevice.ID != edgeDeviceID || edgeDevice.Name != edgeDeviceName || edgeDevice.SerialNumber != edgeDeviceSerialNumberUpdateOk ||
			edgeDevice.IPAddress != edgeDeviceIPUpdated {
			if edgeDevice.ID != edgeDeviceID {
				t.Fatalf("edge device id mismatch %s != %s", edgeDevice.ID, edgeDeviceID)
			}
			if edgeDevice.Name != edgeDeviceName {
				t.Fatal("edge device name mismatch")
			}
			if edgeDevice.SerialNumber != edgeDeviceSerialNumberUpdateOk {
				t.Fatal("edge device serial number mismatch")
			}
			if edgeDevice.IPAddress != edgeDeviceIPUpdated {
				t.Fatal("edge device ip address mismatch")
			}

			t.Fatal("edge device data mismatch")
		}
		edgeDevice, err = dbAPI.GetEdgeDevice(ctx2, edgeDeviceID)
		require.Error(t, err, "expected get edge device 2 to fail")
		edgeDevice, err = dbAPI.GetEdgeDevice(ctx3, edgeDeviceID)
		require.Error(t, err, "expect get edge device 3 to fail since edge device is no longer in project")

		// select all edges
		edges, err := dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		if len(edges) != 2 {
			t.Fatalf("expect edges count 1 to be 1, but got: %d", len(edges))
		}
		edges2, err := dbAPI.SelectAllEdges(ctx2, nil)
		require.NoError(t, err)
		if len(edges2) != 0 {
			t.Fatalf("expect edges count 2 to be 0, but got: %d", len(edges2))
		}
		edges3, err := dbAPI.SelectAllEdges(ctx3, nil)
		require.NoError(t, err)
		if len(edges3) != 0 {
			t.Fatalf("expect edges count 3 to be 0, but got: %d", len(edges3))
		}
		for _, edge := range edges {
			testForMarshallability(t, edge)
		}
		t.Log("get all edges successful")

		// update one more time
		// update edge device
		doc2 = model.EdgeDevice{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					ID:       edgeDeviceID,
					TenantID: tenantID,
					Version:  5,
				},
				ClusterID: edgeDevice.ClusterID,
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         edgeDeviceName,
				SerialNumber: edgeDeviceSerialNumber,
				IPAddress:    edgeDeviceIPUpdated,
				Subnet:       edgeDeviceSubnet,
				Gateway:      edgeDeviceGateway,
			},
		}
		_, err = dbAPI.UpdateEdgeDevice(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)
		// update edge cluster and remove from project
		clusterdoc2 = model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       edgeClusterID,
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name:      edgeClusterName,
				VirtualIP: base.StringPtr("10.20.1.10"),
			},
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				},
			},
		}
		_, err = dbAPI.UpdateEdgeCluster(ctx1, &clusterdoc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)

		// get edge device
		edgeDevice, err = dbAPI.GetEdgeDevice(ctx1, edgeDeviceID)
		require.NoError(t, err)
		_, err = dbAPI.GetEdgeDevice(ctx2, edgeDeviceID)
		require.Error(t, err, "expect get edge device 2 to fail for non infra admin")
		_, err = dbAPI.GetEdgeDevice(ctx3, edgeDeviceID)
		require.NoError(t, err)
		// select all edges (as we don't have corresponding API for edge device)
		edges, err = dbAPI.SelectAllEdges(ctx1, nil)
		require.NoError(t, err)
		if len(edges) != 2 {
			t.Fatalf("expect edges count 1 to be 2, but got: %d", len(edges))
		}
		edges2, err = dbAPI.SelectAllEdges(ctx2, nil)
		require.NoError(t, err)
		if len(edges2) != 0 {
			t.Fatalf("expect edges count 2 to be 0, but got: %d", len(edges2))
		}
		edges3, err = dbAPI.SelectAllEdges(ctx3, nil)
		require.NoError(t, err)
		if len(edges3) != 2 {
			t.Fatalf("expect edges count 3 to be 2, but got: %d", len(edges3))
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
		if len(edges3) != 2 {
			t.Fatalf("expect edges for project count 3 to be 2, but got: %d", len(edges3))
		}
		// get edge device by serial number
		edgeDeviceSN, err := dbAPI.GetEdgeDeviceBySerialNumber(ctx1, strings.ToLower(edgeDeviceSerialNumber))
		require.NoError(t, err)
		err = dbAPI.UpdateEdgeDeviceOnboarded(ctx1, edgeDeviceID, "fake-public-key")
		require.NoError(t, err)
		// get edge device by serial number
		edgeDeviceSN, err = dbAPI.GetEdgeDeviceBySerialNumber(ctx1, strings.ToLower(edgeDeviceSerialNumber))
		require.NoError(t, err)
		testForMarshallability(t, edgeDeviceSN)

		// get edge by serial number
		edgeDeviceSN, err = dbAPI.GetEdgeDeviceBySerialNumber(ctx1, strings.ToUpper(edgeDeviceSerialNumber))
		require.NoError(t, err)
		testForMarshallability(t, edgeDeviceSN)

		// get edge handle
		// assert edge cert is not locked
		ec, err := dbAPI.GetEdgeCertByEdgeID(ctx1, edgeDevice.ClusterID)
		require.NoError(t, err)
		if ec.Locked {
			t.Fatal("unexpected edge cert locked")
		}

		token, err := crypto.EncryptPassword(edgeDevice.ClusterID)
		require.NoError(t, err)
		payload := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		edgeCert, err = dbAPI.GetEdgeHandle(ctx1, edgeDevice.ClusterID, payload)
		require.NoError(t, err, "GetEdgeHandle failed")
		require.True(t, edgeCert.Locked, "unexpected edge cert NOT locked")
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
				// The edge device can talk to cloud only using the clusterID and not device ID
				"edgeId": edgeClusterID,
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContextE)
		edges, err := dbAPI.SelectAllEdges(newCtx, nil)
		require.NoError(t, err)
		if len(edges) != 2 {
			t.Fatal("expected 2 edges")
		}
		edge, err := dbAPI.GetEdge(newCtx, edgeDeviceID)
		require.NoError(t, err)
		t.Logf("Got edge: %+v", edge)
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(newCtx, edgeClusterID)
		require.NoError(t, err)
		t.Logf("Got edge cert: %+v", edgeCert)
		projectRoles, err := dbAPI.GetEdgeProjectRoles(newCtx, edgeClusterID)
		require.NoError(t, err)
		t.Logf("Got project roles: %+v", projectRoles)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateEdgeDevice(tenantID, edgeClusterID, "1.1.10.0")
		doc.ID = id
		return dbAPI.CreateEdgeDevice(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetEdgeDevice(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteEdgeDevice(ctx1, id, nil)
	}))
}

func TestEdgeDeviceW(t *testing.T) {
	t.Parallel()
	t.Log("running TestEdgeDeviceW test")
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

	t.Run("Create/Get/Delete Edge Device", func(t *testing.T) {
		t.Log("running Create/Get/Delete Edge Device test")
		edgeClusterName := "my-test-edge-cluster"
		// Edge Cluster object, leave ID blank and let create generate it
		clusterdoc := model.EdgeCluster{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			EdgeClusterCore: model.EdgeClusterCore{
				Name: edgeClusterName,
			},
		}

		r, err := objToReader(clusterdoc)
		require.NoError(t, err)

		// create edge cluster
		var w bytes.Buffer
		err = dbAPI.CreateEdgeClusterW(ctx1, &w, r, nil)
		require.NoError(t, err)
		respv2 := model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&respv2)
		require.NoError(t, err)
		edgeClusterID := respv2.ID

		edgeName := "my-test-edge"
		edgeIP := "1.1.1.1"
		edgeIPUpdated := "1.1.1.2"
		edgeSubnet := "255.255.255.0"
		edgeGateway := "1.1.1.1"

		// EdgeDevice object, leave ID blank and let create generate it
		doc := model.EdgeDevice{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					ID:       "",
					TenantID: tenantID,
					Version:  5,
				},
				ClusterID: edgeClusterID,
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         edgeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    edgeIP,
				Subnet:       edgeSubnet,
				Gateway:      edgeGateway,
			},
		}

		r, err = objToReader(doc)
		require.NoError(t, err)

		// create edge device
		err = dbAPI.CreateEdgeDeviceW(ctx1, &w, r, nil)
		require.NoError(t, err)
		respv2 = model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&respv2)
		require.NoError(t, err)
		edgeDeviceID := respv2.ID

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
		err = dbAPI.CreateProjectW(ctx1, &w, r, nil)
		require.NoError(t, err)
		resp := model.CreateDocumentResponse{}
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

		// update edge device
		doc = model.EdgeDevice{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					ID:       edgeDeviceID,
					TenantID: tenantID,
					Version:  5,
				},
				ClusterID: edgeClusterID,
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         edgeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    edgeIPUpdated,
				Subnet:       edgeSubnet,
				Gateway:      edgeGateway,
			},
		}
		r, err = objToReader(doc)
		require.NoError(t, err)

		err = dbAPI.UpdateEdgeDeviceW(ctx1, &w, r, nil)
		require.NoError(t, err)
		upResp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&upResp)
		require.NoError(t, err)

		// get edge device
		err = dbAPI.GetEdgeW(ctx1, edgeDeviceID, &w, nil)
		require.NoError(t, err)
		edgeDevice := model.EdgeDevice{}
		err = json.NewDecoder(&w).Decode(&edgeDevice)
		require.NoError(t, err)

		if edgeDevice.ID != edgeDeviceID || edgeDevice.Name != edgeName || edgeDevice.SerialNumber != edgeSerialNumber ||
			edgeDevice.IPAddress != edgeIPUpdated {
			t.Fatal("edge device data mismatch")
		}
		// get all edge devices
		// auth 1
		err = dbAPI.SelectAllEdgeDevicesW(ctx1, &w, nil)
		require.NoError(t, err)
		edgeDevices := model.EdgeDeviceListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeDevices)
		require.NoError(t, err)
		if len(edgeDevices.EdgeDeviceList) != 1 {
			t.Fatal("expect all edge devices 1 count to be 1")
		}
		// auth 2
		err = dbAPI.SelectAllEdgeDevicesW(ctx2, &w, nil)
		require.NoError(t, err)
		edgeDevices = model.EdgeDeviceListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeDevices)
		require.NoError(t, err)
		if len(edgeDevices.EdgeDeviceList) != 0 {
			t.Fatal("expect all edge devices 2 count to be 0")
		}
		// auth 3
		err = dbAPI.SelectAllEdgeDevicesW(ctx3, &w, nil)
		require.NoError(t, err)
		edgeDevices = model.EdgeDeviceListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeDevices)
		require.NoError(t, err)
		if len(edgeDevices.EdgeDeviceList) != 1 {
			t.Fatal("expect all edge devices 3 count to be 1")
		}

		// select all vs select all W (using edges as edge device does not have select all api)
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
		err = dbAPI.SelectAllEdgeDevicesForProjectW(ctx1, projectID, &w, nil)
		require.Error(t, err, "expect all edge devices 1 for project to fail")
		// auth 2
		err = dbAPI.SelectAllEdgeDevicesForProjectW(ctx2, projectID, &w, nil)
		require.Error(t, err, "expect all edges devices 2 for project to fail")
		// auth 3
		err = dbAPI.SelectAllEdgeDevicesForProjectW(ctx3, projectID, &w, nil)
		require.NoError(t, err)
		edgeDevices = model.EdgeDeviceListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeDevices)
		require.NoError(t, err)
		if len(edgeDevices.EdgeDeviceList) != 1 {
			t.Fatal("expect all edge devices 3 for project count to be 1")
		}

		// get edge by serial number
		payload := model.SerialNumberPayload{
			SerialNumber: edgeSerialNumber,
		}
		r, err = objToReader(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetEdgeDeviceBySerialNumberW(ctx1, &w, req)
		require.NoError(t, err)
		// edgeDeviceWithBootInfo
		edgeDevice = model.EdgeDevice{}
		err = json.NewDecoder(&w).Decode(&edgeDevice)
		require.NoError(t, err)
		testForMarshallability(t, edgeDevice)

		// get edge handle
		token, err := crypto.EncryptPassword(edgeClusterID)
		require.NoError(t, err)
		payload2 := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		r, err = objToReader(payload2)
		require.NoError(t, err)
		req = httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetEdgeHandleW(ctx1, edgeClusterID, &w, req)
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
		// log.Printf("delete project successful, %v", delResp)

		// delete edge device and cluster
		err = dbAPI.DeleteEdgeDeviceW(ctx1, edgeDeviceID, &w, nil)
		require.NoError(t, err)
		delRespV2 := model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delRespV2)
		require.NoError(t, err)
		err = dbAPI.DeleteEdgeClusterW(ctx1, edgeClusterID, &w, nil)
		require.NoError(t, err)
		delRespV2 = model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delRespV2)
		require.NoError(t, err)

	})

	// select all edges
	t.Run("SelectAllEdges", func(t *testing.T) {
		t.Log("running SelectAllEdges test")
		var w bytes.Buffer
		err := dbAPI.SelectAllEdgeDevicesW(ctx1, &w, nil)
		require.NoError(t, err)
		edgeDevices := model.EdgeDeviceListPayload{}
		err = json.NewDecoder(&w).Decode(&edgeDevices)
		require.NoError(t, err)
		for _, edgeDevice := range edgeDevices.EdgeDeviceList {
			testForMarshallability(t, edgeDevice)
		}
	})
}

func TestEdgeDeviceTypeSelect(t *testing.T) {
	t.Parallel()
	// To improve test we can make edges of type cloud and edge live together
	typeTable := []string{"EDGE", "CLOUD"}
	numOfDevicesTable := []int{3, 1, 2, 5}
	for _, numOfDevices := range numOfDevicesTable {
		for _, edgeDeviceType := range typeTable {
			t.Logf("running TestEdgeDeviceTypeSelect for %s test", typeTable)
			// Setup
			dbAPI := newObjectModelAPI(t)
			doc := createTenant(t, dbAPI, "test tenant")
			tenantID := doc.ID
			edgeDevices := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, nil, edgeDeviceType, numOfDevices)

			edgeClusterID := edgeDevices[0].ClusterID

			ctx1, _, _ := makeContext(tenantID, []string{})

			// Teardown
			defer func() {
				// dbAPI.DeleteEdgeDevice(ctx1, edgeDeviceID, nil), deleting cluster should delete the devices
				dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
				dbAPI.DeleteTenant(ctx1, tenantID, nil)
				dbAPI.Close()
			}()

			t.Run("Select all edge devices Edge Device", func(t *testing.T) {
				t.Log("running Get edge device with type")
				// create edge cluster
				edgeDevices := model.EdgeDeviceListPayload{}
				var w bytes.Buffer
				// this should get all edgeDevices = 1
				err := dbAPI.SelectAllEdgeDevicesW(ctx1, &w, nil)
				require.NoError(t, err)
				err = json.NewDecoder(&w).Decode(&edgeDevices)
				require.NoError(t, err)
				if len(edgeDevices.EdgeDeviceList) != numOfDevices {
					t.Fatalf("Expected only %d device but got %d", numOfDevices, len(edgeDevices.EdgeDeviceList))
				}
				nonEdgeDeviceType := "CLOUD"
				if edgeDeviceType == "CLOUD" {
					nonEdgeDeviceType = "EDGE"
				}
				// this should get only edgeDevices who are of type edge = 0
				req, err := http.NewRequest("GET", fmt.Sprintf("http://test.com/?type=%s", nonEdgeDeviceType), nil)
				require.NoError(t, err)
				edgeDevices = model.EdgeDeviceListPayload{}
				err = dbAPI.SelectAllEdgeDevicesW(ctx1, &w, req)
				require.NoError(t, err)
				err = json.NewDecoder(&w).Decode(&edgeDevices)
				require.NoError(t, err)
				if len(edgeDevices.EdgeDeviceList) != 0 {
					t.Log(req.URL)
					t.Fatalf("Expected no devices but got %d", len(edgeDevices.EdgeDeviceList))
				}
				// this should get all edgeDevices are of type cloud = 1
				req, err = http.NewRequest("GET", fmt.Sprintf("http://test.com/?type=%s", edgeDeviceType), nil)
				require.NoError(t, err)
				edgeDevices = model.EdgeDeviceListPayload{}
				err = dbAPI.SelectAllEdgeDevicesW(ctx1, &w, req)
				require.NoError(t, err)
				err = json.NewDecoder(&w).Decode(&edgeDevices)
				require.NoError(t, err)
				if len(edgeDevices.EdgeDeviceList) != numOfDevices {
					t.Fatalf("Expected only %d device but got %d", numOfDevices, len(edgeDevices.EdgeDeviceList))
				}
			})
		}
	}
}

func TestEdgeDeviceBootstrapMaster(t *testing.T) {
	// Could add tests to check only one bootstrap master is possible
	t.Parallel()
	numOfDevicesTable := []int{3, 1, 2, 5}
	// bootstrap master is between 0 and corresponding numOfDevices
	bootStrapMasterDeviceTable := []int{1, 0, 0, 3}

	for indx, numOfDevices := range numOfDevicesTable {
		t.Logf("running TestEdgeDeviceBootstrapMaster test")
		// Setup
		dbAPI := newObjectModelAPI(t)
		doc := createTenant(t, dbAPI, "test tenant")
		tenantID := doc.ID
		edgeDevices := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", numOfDevices)
		edgeClusterID := edgeDevices[0].ClusterID
		ctx1, _, _ := makeContext(tenantID, []string{})
		// Teardown
		defer func() {
			// dbAPI.DeleteEdgeDevice(ctx1, edgeDeviceID, nil), deleting cluster should delete the devices
			dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
			dbAPI.DeleteTenant(ctx1, tenantID, nil)
			dbAPI.Close()
		}()
		t.Run("Get edge by serial number for Edge Device", func(t *testing.T) {
			t.Log("running bootstrap master")
			bootStrapMasterSn := edgeDevices[bootStrapMasterDeviceTable[indx]].SerialNumber
			// bootstrap master calls get edge by sn first
			bootStrapDevice, err := dbAPI.GetEdgeDeviceBySerialNumber(ctx1, bootStrapMasterSn)
			require.NoError(t, err)
			if !bootStrapDevice.IsBootstrapMaster {
				t.Fatalf("expected %d to be bootstrap master but its not", bootStrapMasterDeviceTable[indx])
			}
			for device := 0; device < numOfDevices; device++ {
				deviceSN := edgeDevices[device].SerialNumber
				edgeDevice, err := dbAPI.GetEdgeDeviceBySerialNumber(ctx1, deviceSN)
				require.NoError(t, err)

				if device != bootStrapMasterDeviceTable[indx] {
					// device is not bootstrap master
					if edgeDevice.IsBootstrapMaster {
						t.Fatalf("expected %d to be bootstrap master but got %d", bootStrapMasterDeviceTable[indx], device)
					}
				} else {
					if !edgeDevice.IsBootstrapMaster {
						t.Fatalf("expected %d to be bootstrap master but its not", bootStrapMasterDeviceTable[indx])
					}
				}
			}
		})
	}
}

func TestEdgeAPIDisabled(t *testing.T) {
	t.Parallel()
	t.Logf("running TestEdgeAPIDisabled test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	edgeDevices := createEdgeDeviceWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", 2)
	edgeClusterID := edgeDevices[0].ClusterID
	edgeDeviceID := edgeDevices[0].ID
	ctx1, _, _ := makeContext(tenantID, []string{})
	// Teardown
	defer func() {
		// dbAPI.DeleteEdgeDevice(ctx1, edgeDeviceID, nil), deleting cluster should delete the devices
		dbAPI.DeleteEdgeCluster(ctx1, edgeClusterID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()
	t.Run("Test edge API disabled", func(t *testing.T) {
		t.Log("running TestEdgeAPIDisabled test")
		_, err := dbAPI.DeleteEdge(ctx1, edgeDeviceID, nil)
		require.Errorf(t, err, "expected deprecated err but got %v", err)
		ed := model.EdgeDevice{
			ClusterEntityModel: model.ClusterEntityModel{
				BaseModel: model.BaseModel{
					ID:       "",
					TenantID: tenantID,
					Version:  5,
				},
				ClusterID: edgeClusterID,
			},
			EdgeDeviceCore: model.EdgeDeviceCore{
				Name:         edgeDevices[0].Name,
				SerialNumber: edgeDevices[0].SerialNumber,
				IPAddress:    edgeDevices[0].IPAddress,
				Subnet:       edgeDevices[0].Subnet,
				Gateway:      edgeDevices[0].Gateway,
			},
			Description: "new desc",
		}
		_, err = dbAPI.UpdateEdge(ctx1, ed, nil)
		require.Errorf(t, err, "expected deprecated err but got %v", err)
	})
}

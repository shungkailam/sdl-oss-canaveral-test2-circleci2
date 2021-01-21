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
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	funk "github.com/thoas/go-funk"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally

func createNode(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.Node {
	return createNodeWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", 1)[0]
}

func createNodeWithLabelsCommon(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, labels []model.CategoryInfo, svcDomainType string, numOfNodes int) []model.Node {
	// create node
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	svcDomainType = strings.ToUpper(svcDomainType)
	svcDomainName := "my-test-service-domain-" + base.GetUUID()
	svcDomain := model.ServiceDomain{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		ServiceDomainCore: model.ServiceDomainCore{
			Name: svcDomainName,
			Type: &svcDomainType,
		},
		Labels: labels,
	}
	if numOfNodes > 1 {
		svcDomain.VirtualIP = base.StringPtr("10.20.1.10")
	}
	resp, err := dbAPI.CreateServiceDomain(ctx, &svcDomain, nil)
	require.NoError(t, err)
	svcDomain.ID = resp.(model.CreateDocumentResponse).ID
	nodes := []model.Node{}
	for count := 0; count < numOfNodes; count++ {
		nodeSerialNumber := base.GetUUID()
		nodeName := "my-test-node-" + nodeSerialNumber + "-" + strconv.Itoa(count)
		nodeSerialNumberLen := len(nodeSerialNumber)
		nodeSerialNumber = strings.ToUpper(nodeSerialNumber[:nodeSerialNumberLen/2]) + nodeSerialNumber[nodeSerialNumberLen/2:]
		nodeIP := "1.1.1." + strconv.Itoa(count)
		nodeSubnet := "255.255.255.0"
		nodeGateway := "1.1.1.1"
		// Node object, leave ID blank and let create generate it
		node := model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:       "",
					TenantID: tenantID,
					Version:  5,
				},
				SvcDomainID: svcDomain.ID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: nodeSerialNumber,
				IPAddress:    nodeIP,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}
		resp, err = dbAPI.CreateNode(ctx, &node, nil)
		require.NoError(t, err)
		node.ID = resp.(model.CreateDocumentResponse).ID
		nodes = append(nodes, node)
	}
	return nodes
}

func TestNode(t *testing.T) {
	t.Parallel()
	t.Log("running TestNode test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	nodes := createNodeWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}}, "EDGE", 2)
	node := nodes[1]
	nodeID := node.ID
	nodeSerialNumber := node.SerialNumber
	svcDomainID := node.SvcDomainID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteNode(ctx1, nodeID, nil)
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Node", func(t *testing.T) {
		t.Log("running Create/Get/Delete Node test")

		nodeName := "my-test-node"
		svcDomainName := "my-test-service-domain"

		nodeIPUpdated := "1.1.1.2"
		nodeSubnet := "255.255.255.0"
		nodeGateway := "1.1.1.1"

		if nodes[0].ID != svcDomainID {
			t.Fatalf("expected first node ID to be %s, found %s", svcDomainID, nodes[0].ID)
		}
		// update node
		doc2 := model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:       nodeID,
					TenantID: tenantID,
					Version:  5,
				},
				SvcDomainID: svcDomainID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: nodeSerialNumber,
				IPAddress:    nodeIPUpdated,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}

		// get node
		node, err := dbAPI.GetNode(ctx1, nodeID)
		require.NoError(t, err)
		_, err = dbAPI.GetNode(ctx2, nodeID)
		require.Error(t, err, "expect get node 2 to fail for non infra admin")
		_, err = dbAPI.GetNode(ctx3, nodeID)
		require.NoError(t, err)

		nodeSerialNumberUpdateOk := fmt.Sprintf("%s-UPDATE-ok", doc2.SerialNumber)
		doc2.SerialNumber = nodeSerialNumberUpdateOk
		_, err = dbAPI.UpdateNode(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)
		// now lock the edge cert
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(ctx1, doc2.SvcDomainID)
		require.NoError(t, err)
		edgeCert.Locked = true
		_, err = dbAPI.UpdateEdgeCert(ctx1, &edgeCert, nil)
		require.NoError(t, err)
		nodeSerialNumberUpdateNotOk := fmt.Sprintf("%s-UPDATE-not-ok", nodeSerialNumber)
		doc2.SerialNumber = nodeSerialNumberUpdateNotOk
		_, err = dbAPI.UpdateNode(ctx1, &doc2, nil)
		require.Error(t, err, "expect update node with modified serial number to fail once cert is locked")

		// change serial number only by letter case should be ok for locked node
		nodeSerialNumberLen := len(nodeSerialNumberUpdateOk)
		nodeSerialNumberUpdateOk = strings.ToLower(nodeSerialNumberUpdateOk[:nodeSerialNumberLen/2]) +
			strings.ToUpper(nodeSerialNumberUpdateOk[nodeSerialNumberLen/2:])
		doc2.SerialNumber = nodeSerialNumberUpdateOk
		_, err = dbAPI.UpdateNode(ctx1, &doc2, nil)
		require.NoError(t, err)

		// clean up: unlock the edge cert
		edgeCert.Locked = false
		_, err = dbAPI.UpdateEdgeCert(ctx1, &edgeCert, nil)
		require.NoError(t, err)

		_, err = dbAPI.UpdateNode(ctx2, &doc2, nil)
		require.Error(t, err, "expect update node 2 to fail for non infra admin")
		_, err = dbAPI.UpdateNode(ctx3, &doc2, nil)
		require.Error(t, err, "expect update node 3 to fail for non infra admin")

		// update service domain and remove from project
		svcDomainDoc2 := model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       svcDomainID,
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name:      svcDomainName,
				VirtualIP: base.StringPtr("10.20.1.20"),
			},
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue2,
				},
			},
		}
		_, err = dbAPI.UpdateServiceDomain(ctx1, &svcDomainDoc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)

		// get node
		node, err = dbAPI.GetNode(ctx1, nodeID)
		require.NoError(t, err)
		if node.ID != nodeID || node.Name != nodeName || node.SerialNumber != nodeSerialNumberUpdateOk ||
			node.IPAddress != nodeIPUpdated {
			if node.ID != nodeID {
				t.Fatalf("node id mismatch %s != %s", node.ID, nodeID)
			}
			if node.Name != nodeName {
				t.Fatal("node name mismatch")
			}
			if node.SerialNumber != nodeSerialNumberUpdateOk {
				t.Fatal("node serial number mismatch")
			}
			if node.IPAddress != nodeIPUpdated {
				t.Fatal("node ip address mismatch")
			}

			t.Fatal("node data mismatch")
		}
		node, err = dbAPI.GetNode(ctx2, nodeID)
		require.Error(t, err, "expected get node 2 to fail")
		node, err = dbAPI.GetNode(ctx3, nodeID)
		require.Error(t, err, "expect get node 3 to fail since node is no longer in project")

		// select all nodes
		nodes, err := dbAPI.SelectAllNodes(ctx1, nil)
		require.NoError(t, err)
		if len(nodes) != 2 {
			t.Fatalf("expect nodes count 1 to be 1, but got: %d", len(nodes))
		}
		nodes2, err := dbAPI.SelectAllNodes(ctx2, nil)
		require.NoError(t, err)
		if len(nodes2) != 0 {
			t.Fatalf("expect nodes count 2 to be 0, but got: %d", len(nodes2))
		}
		nodes3, err := dbAPI.SelectAllNodes(ctx3, nil)
		require.NoError(t, err)
		if len(nodes3) != 0 {
			t.Fatalf("expect nodes count 3 to be 0, but got: %d", len(nodes3))
		}
		for _, node := range nodes {
			testForMarshallability(t, node)
		}
		t.Log("get all nodes successful")

		// update one more time
		// update node
		doc2 = model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:       nodeID,
					TenantID: tenantID,
					Version:  5,
				},
				SvcDomainID: node.SvcDomainID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: nodeSerialNumber,
				IPAddress:    nodeIPUpdated,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}
		_, err = dbAPI.UpdateNode(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)
		// update service domain and remove from project
		svcDomainDoc2 = model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       svcDomainID,
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name:      svcDomainName,
				VirtualIP: base.StringPtr("10.20.1.10"),
			},
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				},
			},
		}
		_, err = dbAPI.UpdateServiceDomain(ctx1, &svcDomainDoc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)

		// get node
		node, err = dbAPI.GetNode(ctx1, nodeID)
		require.NoError(t, err)
		_, err = dbAPI.GetNode(ctx2, nodeID)
		require.Error(t, err, "expect get node 2 to fail for non infra admin")
		_, err = dbAPI.GetNode(ctx3, nodeID)
		require.NoError(t, err)
		// select all nodes (as we don't have corresponding API for node)
		nodes, err = dbAPI.SelectAllNodes(ctx1, nil)
		require.NoError(t, err)
		if len(nodes) != 2 {
			t.Fatalf("expect nodes count 1 to be 2, but got: %d", len(nodes))
		}
		nodes2, err = dbAPI.SelectAllNodes(ctx2, nil)
		require.NoError(t, err)
		if len(nodes2) != 0 {
			t.Fatalf("expect nodes count 2 to be 0, but got: %d", len(nodes2))
		}
		nodes3, err = dbAPI.SelectAllNodes(ctx3, nil)
		require.NoError(t, err)
		if len(nodes3) != 2 {
			t.Fatalf("expect nodes count 3 to be 2, but got: %d", len(nodes3))
		}

		// select all nodes for project
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		nodes, err = dbAPI.SelectAllNodesForProject(newCtx, projectID, nil)
		require.Error(t, err, "expect select all nodes 1 for project to fail")
		nodes2, err = dbAPI.SelectAllNodesForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect select all nodes 2 for project to fail")
		nodes3, err = dbAPI.SelectAllNodesForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(nodes3) != 2 {
			t.Fatalf("expect nodes for project count 3 to be 2, but got: %d", len(nodes3))
		}
		// get node by serial number
		nodeSN, err := dbAPI.GetNodeBySerialNumber(ctx1, strings.ToLower(nodeSerialNumber))
		require.NoError(t, err)
		err = dbAPI.UpdateNodeOnboarded(ctx1, &model.NodeOnboardInfo{NodeID: nodeID, SSHPublicKey: "fake-public-key", NodeVersion: "v1.14.0"})
		require.NoError(t, err)
		// get node by serial number
		nodeSN, err = dbAPI.GetNodeBySerialNumber(ctx1, strings.ToLower(nodeSerialNumber))
		require.NoError(t, err)
		testForMarshallability(t, nodeSN)

		// get node by serial number
		nodeSN, err = dbAPI.GetNodeBySerialNumber(ctx1, strings.ToUpper(nodeSerialNumber))
		require.NoError(t, err)
		testForMarshallability(t, nodeSN)

		// get edge handle
		// assert edge cert is not locked
		ec, err := dbAPI.GetEdgeCertByEdgeID(ctx1, node.SvcDomainID)
		require.NoError(t, err)
		if ec.Locked {
			t.Fatal("unexpected edge cert locked")
		}

		token, err := crypto.EncryptPassword(node.SvcDomainID)
		require.NoError(t, err)
		payload := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		edgeCert, err = dbAPI.GetServiceDomainHandle(ctx1, node.SvcDomainID, payload)
		require.NoError(t, err, "GetServiceDomainHandle failed")

		if !edgeCert.Locked {
			t.Fatal("unexpected cert NOT locked")
		}

		testForMarshallability(t, edgeCert)
	})

	// select all nodes
	t.Run("SelectAllNodes", func(t *testing.T) {
		t.Log("running SelectAllNodes test")
		nodes, err := dbAPI.SelectAllNodes(ctx1, nil)
		require.NoError(t, err)
		for _, node := range nodes {
			testForMarshallability(t, node)
		}
	})

	// select all nodes
	t.Run("Node get node", func(t *testing.T) {
		t.Log("running Node get node test")

		authContextE := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				// The node can talk to cloud only using the clusterID and not device ID
				"edgeId": svcDomainID,
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContextE)
		nodes, err := dbAPI.SelectAllNodes(newCtx, nil)
		require.NoError(t, err)
		if len(nodes) != 2 {
			t.Fatal("expected 2 nodes")
		}
		node, err := dbAPI.GetNode(newCtx, nodeID)
		require.NoError(t, err)
		t.Logf("Got node: %+v", node)
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(newCtx, svcDomainID)
		require.NoError(t, err)
		t.Logf("Got edge cert: %+v", edgeCert)
		projectRoles, err := dbAPI.GetEdgeProjectRoles(newCtx, svcDomainID)
		require.NoError(t, err)
		t.Logf("Got project roles: %+v", projectRoles)
	})
}

func TestNodeIdValidation(t *testing.T) {
	t.Parallel()
	t.Log("running TestNodeIdValidation test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	ctx, _, _ := makeContext(tenantID, []string{})

	// svc domain
	svcDomainDoc := generateServiceDomain(tenantID, []model.CategoryInfo{})
	ip := "10.0.0.1"
	svcDomainDoc.VirtualIP = &ip
	svcDomain, _ := dbAPI.CreateServiceDomain(ctx, &svcDomainDoc, nil)
	svcDomainID := svcDomain.(model.CreateDocumentResponse).ID

	// Create the first node
	nodeDoc := generateNode(tenantID, svcDomainID)
	node, _ := dbAPI.CreateNode(ctx, &nodeDoc, nil)
	nodeId := node.(model.CreateDocumentResponse).ID

	// Teardown
	defer func() {
		dbAPI.DeleteNode(ctx, nodeId, nil)
		dbAPI.DeleteServiceDomain(ctx, svcDomainID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateNode(tenantID, svcDomainID)
		doc.ID = id
		return dbAPI.CreateNode(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetNode(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteNode(ctx, id, nil)
	}))
}

func generateNode(tenantID string, svcDomainID string) model.Node {
	return model.Node{
		ServiceDomainEntityModel: model.ServiceDomainEntityModel{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			SvcDomainID: svcDomainID,
		},
		NodeCore: model.NodeCore{
			Name:         strings.ToLower("my-test-node-" + funk.RandomString(10)),
			SerialNumber: base.GetUUID(),
			IPAddress:    "1.1.1.2",
			Subnet:       "255.255.255.0",
			Gateway:      "1.1.1.1",
		},
	}
}

func TestNodeW(t *testing.T) {
	t.Parallel()
	t.Log("running TestNodeW test")
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

	t.Run("Create/Get/Delete Node", func(t *testing.T) {
		t.Log("running Create/Get/Delete Node test")
		svcDomainName := "my-test-service-domain"
		// Service Domain object, leave ID blank and let create generate it
		clusterdoc := model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: svcDomainName,
			},
		}

		r, err := objToReader(clusterdoc)
		require.NoError(t, err)

		// create service domain
		var w bytes.Buffer
		err = dbAPI.CreateServiceDomainW(ctx1, &w, r, nil)
		require.NoError(t, err)
		respv2 := model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&respv2)
		require.NoError(t, err)
		svcDomainID := respv2.ID

		t.Logf("Created service domain %s", svcDomainID)

		nodeName := "my-test-node"
		nodeIP := "1.1.1.1"
		nodeIPUpdated := "1.1.1.2"
		nodeSubnet := "255.255.255.0"
		nodeGateway := "1.1.1.1"

		// EdgeDevice object, leave ID blank and let create generate it
		doc := model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:       "",
					TenantID: tenantID,
					Version:  5,
				},
				SvcDomainID: svcDomainID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    nodeIP,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}

		r, err = objToReader(doc)
		require.NoError(t, err)

		// create node
		err = dbAPI.CreateNodeW(ctx1, &w, r, nil)
		require.NoError(t, err)
		respv2 = model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&respv2)
		require.NoError(t, err)
		nodeID := respv2.ID

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
			EdgeIDs:            []string{svcDomainID},
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

		// update node
		doc = model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:       nodeID,
					TenantID: tenantID,
					Version:  5,
				},
				SvcDomainID: svcDomainID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: edgeSerialNumber,
				IPAddress:    nodeIPUpdated,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}
		r, err = objToReader(doc)
		require.NoError(t, err)

		err = dbAPI.UpdateNodeW(ctx1, &w, r, nil)
		require.NoError(t, err)
		upResp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&upResp)
		require.NoError(t, err)

		// get node
		err = dbAPI.GetNodeW(ctx1, nodeID, &w, nil)
		require.NoError(t, err)
		node := model.Node{}
		err = json.NewDecoder(&w).Decode(&node)
		require.NoError(t, err)

		if node.ID != nodeID || node.Name != nodeName || node.SerialNumber != edgeSerialNumber ||
			node.IPAddress != nodeIPUpdated {
			t.Fatal("node data mismatch")
		}
		// get all nodes
		// auth 1
		err = dbAPI.SelectAllNodesW(ctx1, &w, nil)
		require.NoError(t, err)
		nodes := model.NodeListPayload{}
		err = json.NewDecoder(&w).Decode(&nodes)
		require.NoError(t, err)
		if len(nodes.NodeList) != 1 {
			t.Fatal("expect all nodes 1 count to be 1")
		}
		// auth 2
		err = dbAPI.SelectAllNodesW(ctx2, &w, nil)
		require.NoError(t, err)
		nodes = model.NodeListPayload{}
		err = json.NewDecoder(&w).Decode(&nodes)
		require.NoError(t, err)
		if len(nodes.NodeList) != 0 {
			t.Fatal("expect all nodes 2 count to be 0")
		}
		// auth 3
		err = dbAPI.SelectAllNodesW(ctx3, &w, nil)
		require.NoError(t, err)
		nodes = model.NodeListPayload{}
		err = json.NewDecoder(&w).Decode(&nodes)
		require.NoError(t, err)
		if len(nodes.NodeList) != 1 {
			t.Fatal("expect all nodes 3 count to be 1")
		}

		// select all vs select all W (using nodes as node does not have select all api)
		nodes1, err := dbAPI.SelectAllNodes(ctx1, nil)
		require.NoError(t, err)
		//  nodes2
		nodeListPayload := &model.NodeListPayload{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllNodesW, nodeListPayload, &w)
		require.NoError(t, err)
		nodes2 := nodeListPayload.NodeList
		sort.Sort(model.NodesByID(nodes1))
		sort.Sort(model.NodesByID(nodes2))
		if !reflect.DeepEqual(&nodes1, &nodes2) {
			t.Fatalf("expect select nodes and select nodes w results to be equal %+v vs %+v", nodes1, nodes2)
		}

		// get all nodes for project
		// auth 1
		err = dbAPI.SelectAllNodesForProjectW(ctx1, projectID, &w, nil)
		require.Error(t, err, "expect all nodes 1 for project to fail")
		// auth 2
		err = dbAPI.SelectAllNodesForProjectW(ctx2, projectID, &w, nil)
		require.Error(t, err, "expect all nodes devices 2 for project to fail")
		// auth 3
		err = dbAPI.SelectAllNodesForProjectW(ctx3, projectID, &w, nil)
		require.NoError(t, err)
		nodes = model.NodeListPayload{}
		err = json.NewDecoder(&w).Decode(&nodes)
		require.NoError(t, err)
		if len(nodes.NodeList) != 1 {
			t.Fatal("expect all nodes 3 for project count to be 1")
		}

		payload := model.SerialNumberPayload{
			SerialNumber: edgeSerialNumber,
		}
		r, err = objToReader(payload)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetNodeBySerialNumberW(ctx1, &w, req)
		require.NoError(t, err)
		// nodeWithBootInfo
		node = model.Node{}
		err = json.NewDecoder(&w).Decode(&node)
		require.NoError(t, err)
		testForMarshallability(t, node)

		// get node handle
		token, err := crypto.EncryptPassword(svcDomainID)
		require.NoError(t, err)
		payload2 := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		r, err = objToReader(payload2)
		require.NoError(t, err)
		req = httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetServiceDomainHandleW(ctx1, svcDomainID, &w, req)
		require.NoError(t, err, "GetEdgeHandle failed")

		edgeCert := model.EdgeCert{}
		err = json.NewDecoder(&w).Decode(&edgeCert)
		require.NoError(t, err)
		//
		testForMarshallability(t, edgeCert)

		// delete project
		err = dbAPI.DeleteProjectW(ctx1, projectID, &w, nil)
		require.NoError(t, err)
		delResp := model.DeleteDocumentResponse{}
		err = json.NewDecoder(&w).Decode(&delResp)
		require.NoError(t, err)
		// log.Printf("delete project successful, %v", delResp)

		// delete node and cluster
		err = dbAPI.DeleteNodeW(ctx1, nodeID, &w, nil)
		require.NoError(t, err)
		delRespV2 := model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delRespV2)
		require.NoError(t, err)
		err = dbAPI.DeleteServiceDomainW(ctx1, svcDomainID, &w, nil)
		require.NoError(t, err)
		delRespV2 = model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delRespV2)
		require.NoError(t, err)

	})

	// select all nodes
	t.Run("SelectAllNodes", func(t *testing.T) {
		t.Log("running SelectAllNodes test")
		var w bytes.Buffer
		err := dbAPI.SelectAllNodesW(ctx1, &w, nil)
		require.NoError(t, err)
		nodes := model.NodeListPayload{}
		err = json.NewDecoder(&w).Decode(&nodes)
		require.NoError(t, err)
		for _, node := range nodes.NodeList {
			testForMarshallability(t, node)
		}
	})
}

func TestNodeTypeSelect(t *testing.T) {
	t.Parallel()
	// To improve test we can make nodes of type cloud and edge live together
	typeTable := []string{"EDGE", "CLOUD"}
	numOfDevicesTable := []int{3, 1, 2, 5}
	for _, numOfDevices := range numOfDevicesTable {
		for _, nodeType := range typeTable {
			t.Logf("running TestNodeTypeSelect for %s test", typeTable)
			// Setup
			dbAPI := newObjectModelAPI(t)
			doc := createTenant(t, dbAPI, "test tenant")
			tenantID := doc.ID
			nodes := createNodeWithLabelsCommon(t, dbAPI, tenantID, nil, nodeType, numOfDevices)

			svcDomainID := nodes[0].SvcDomainID

			ctx1, _, _ := makeContext(tenantID, []string{})

			// Teardown
			defer func() {
				// dbAPI.DeleteNode(ctx1, nodeID, nil), deleting cluster should delete the devices
				dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
				dbAPI.DeleteTenant(ctx1, tenantID, nil)
				dbAPI.Close()
			}()

			t.Run("Select all nodes Node", func(t *testing.T) {
				t.Log("running Get node with type")
				// create service domain
				nodes := model.NodeListPayload{}
				var w bytes.Buffer
				// this should get all nodes = 1
				err := dbAPI.SelectAllNodesW(ctx1, &w, nil)
				require.NoError(t, err)
				err = json.NewDecoder(&w).Decode(&nodes)
				require.NoError(t, err)
				if len(nodes.NodeList) != numOfDevices {
					t.Fatalf("Expected only %d device but got %d", numOfDevices, len(nodes.NodeList))
				}
				nonEdgeDeviceType := "CLOUD"
				if nodeType == "CLOUD" {
					nonEdgeDeviceType = "EDGE"
				}
				// this should get only nodes who are of type edge = 0
				req, err := http.NewRequest("GET", fmt.Sprintf("http://test.com/?type=%s", nonEdgeDeviceType), nil)
				require.NoError(t, err)
				nodes = model.NodeListPayload{}
				err = dbAPI.SelectAllNodesW(ctx1, &w, req)
				require.NoError(t, err)
				err = json.NewDecoder(&w).Decode(&nodes)
				require.NoError(t, err)
				if len(nodes.NodeList) != 0 {
					t.Log(req.URL)
					t.Fatalf("Expected no devices but got %d", len(nodes.NodeList))
				}
				// this should get all nodes are of type cloud = 1
				req, err = http.NewRequest("GET", fmt.Sprintf("http://test.com/?type=%s", nodeType), nil)
				require.NoError(t, err)
				nodes = model.NodeListPayload{}
				err = dbAPI.SelectAllNodesW(ctx1, &w, req)
				require.NoError(t, err)
				err = json.NewDecoder(&w).Decode(&nodes)
				require.NoError(t, err)
				if len(nodes.NodeList) != numOfDevices {
					t.Fatalf("Expected only %d device but got %d", numOfDevices, len(nodes.NodeList))
				}
			})
		}
	}
}

func TestNodeBootstrapMaster(t *testing.T) {
	// Could add tests to check only one bootstrap master is possible
	t.Parallel()
	numOfDevicesTable := []int{3, 1, 2, 5}
	// bootstrap master is between 0 and corresponding numOfDevices
	bootStrapMasterDeviceTable := []int{1, 0, 0, 3}

	for indx, numOfDevices := range numOfDevicesTable {
		t.Logf("running TestNodeBootstrapMaster test")
		// Setup
		dbAPI := newObjectModelAPI(t)
		doc := createTenant(t, dbAPI, "test tenant")
		tenantID := doc.ID
		nodes := createNodeWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", numOfDevices)
		svcDomainID := nodes[0].SvcDomainID
		ctx1, _, _ := makeContext(tenantID, []string{})
		// Teardown
		defer func() {
			// dbAPI.DeleteNode(ctx1, nodeID, nil), deleting cluster should delete the devices
			dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
			dbAPI.DeleteTenant(ctx1, tenantID, nil)
			dbAPI.Close()
		}()
		t.Run("Get node by serial number for Node", func(t *testing.T) {
			t.Log("running bootstrap master")
			bootStrapMasterSn := nodes[bootStrapMasterDeviceTable[indx]].SerialNumber
			// bootstrap master calls get node by sn first
			bootStrapDevice, err := dbAPI.GetNodeBySerialNumber(ctx1, bootStrapMasterSn)
			require.NoError(t, err)
			if !bootStrapDevice.IsBootstrapMaster {
				t.Fatalf("expected %d to be bootstrap master but its not", bootStrapMasterDeviceTable[indx])
			}
			if numOfDevices > 1 {
				slaveIdx := (bootStrapMasterDeviceTable[indx] + 1) % numOfDevices
				deviceSN := nodes[slaveIdx].SerialNumber
				_, err = dbAPI.GetNodeBySerialNumber(ctx1, deviceSN)
				require.Error(t, err, "slave device onboard must fail")
			}
			// Onboard master
			err = dbAPI.UpdateNodeOnboarded(ctx1, &model.NodeOnboardInfo{NodeID: bootStrapDevice.ID, SSHPublicKey: "fake-public-key", NodeVersion: "v1.15.0"})
			require.NoError(t, err)
			for device := 0; device < numOfDevices; device++ {
				deviceSN := nodes[device].SerialNumber
				node, err := dbAPI.GetNodeBySerialNumber(ctx1, deviceSN)
				require.NoError(t, err)

				if device != bootStrapMasterDeviceTable[indx] {
					// device is not bootstrap master
					if node.IsBootstrapMaster {
						t.Fatalf("expected %d to be bootstrap master but got %d", bootStrapMasterDeviceTable[indx], device)
					}
				} else {
					if !node.IsBootstrapMaster {
						t.Fatalf("expected %d to be bootstrap master but its not", bootStrapMasterDeviceTable[indx])
					}
				}
			}
		})
	}
}

func TestConcurrentCreateNodes(t *testing.T) {
	type responseMessage struct {
		err error
		id  string
	}

	t.Parallel()
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	ctx, _, _ := makeContext(tenantID, []string{})
	svcDomain := createServiceDomainWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{})
	// Now set the virtual IP
	svcDomain.VirtualIP = base.StringPtr("10.10.10.30")
	_, err := dbAPI.UpdateServiceDomain(ctx, &svcDomain, nil)
	require.NoError(t, err)
	// Teardown
	defer func() {
		dbAPI.DeleteServiceDomain(ctx, svcDomain.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()
	nodeCount := 3
	createNodeFn := func(i int, wg *sync.WaitGroup, errChan chan *responseMessage) error {
		t.Logf("starting node creation for %d", i)
		defer func() {
			wg.Done()
			t.Logf("done node creation for %d", i)
		}()
		nodeSerialNumber := base.GetUUID()
		nodeName := "my-test-node-" + nodeSerialNumber + "-" + strconv.Itoa(i)
		nodeIP := "1.1.1." + strconv.Itoa(i)
		nodeSubnet := "255.255.255.0"
		nodeGateway := "1.1.1.1"
		// Node object, leave ID blank and let create generate it
		node := model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
					ID:       "",
					TenantID: tenantID,
					Version:  5,
				},
				SvcDomainID: svcDomain.ID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: nodeSerialNumber,
				IPAddress:    nodeIP,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}
		resp, err := dbAPI.CreateNode(ctx, &node, nil)
		if err != nil {
			errChan <- &responseMessage{err: err}
			return err
		}
		node.ID = resp.(model.CreateDocumentResponse).ID
		errChan <- &responseMessage{id: node.ID}
		return nil
	}
	// Now go crazy
	for i := 0; i < 10; i++ {
		wg := &sync.WaitGroup{}
		errChan := make(chan *responseMessage, nodeCount)
		for j := 0; j < nodeCount; j++ {
			wg.Add(1)
			go createNodeFn(j, wg, errChan)
		}
		wg.Wait()
		for j := 0; j < nodeCount; j++ {
			msg := <-errChan
			if msg.err != nil {
				t.Fatal(msg.err)
			}
			_, err := dbAPI.DeleteNode(ctx, msg.id, nil)
			require.NoError(t, err)
		}
	}
}

func TestNodeDeletion(t *testing.T) {
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	ctx1, _, _ := makeContext(tenantID, []string{})
	node := createNodeWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{}, "EDGE", 1)[0]
	otherNode := createNodeWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{}, "EDGE", 1)[0]
	nodeID := node.ID
	otherNodeID := otherNode.ID
	nodeCtx := makeEdgeContext(tenantID, nodeID, nil)
	otherNodeCtx := makeEdgeContext(tenantID, otherNodeID, nil)
	svcDomainID := node.SvcDomainID
	otherSvcDomainID := otherNode.SvcDomainID
	defer func() {
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteServiceDomain(ctx1, otherSvcDomainID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	err := dbAPI.SetEdgeCertLock(otherNodeCtx, otherNodeID, true)
	require.NoError(t, err)
	// No version by this time
	// Irrespective of cert lock, it can be deleted
	_, err = dbAPI.DeleteNode(ctx1, otherNodeID, nil)
	require.NoError(t, err)

	err = dbAPI.UpdateNodeOnboarded(ctx1, &model.NodeOnboardInfo{NodeID: nodeID, SSHPublicKey: "fake-public-key", NodeVersion: "v1.15.0"})
	require.NoError(t, err)

	nodeInfo, err := dbAPI.GetNodeInfo(ctx1, nodeID)
	require.NoError(t, err)
	// Hack to downgrade to non-MN aware
	nodeInfo.NodeVersion = base.StringPtr("v1.14.0")
	// CreateNodeInfo also updates
	_, err = dbAPI.CreateNodeInfo(ctx1, &nodeInfo, nil)
	require.NoError(t, err)

	// non-mn lock the cert of the node or svc domain
	err = dbAPI.SetEdgeCertLock(nodeCtx, nodeID, true)
	require.NoError(t, err)

	// Delete service domain must work
	_, err = dbAPI.DeleteServiceDomain(ctx1, otherSvcDomainID, nil)
	require.NoError(t, err)
}

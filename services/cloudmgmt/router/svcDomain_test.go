package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	SERVICE_DOMAINS_PATH = "/v1.0/servicedomains"
	NODES_PATH           = "/v1.0/nodes"
)

func createServiceDomainForTenant(netClient *http.Client, tenantID string, token string, targetType model.TargetType) (model.ServiceDomain, string, error) {
	svcDomainName := fmt.Sprintf("my-test-service-domain-%s", base.GetUUID())
	// Service domain object, leave ID blank and let create generate it
	svcDomain := model.ServiceDomain{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		ServiceDomainCore: model.ServiceDomainCore{
			Name:      svcDomainName,
			VirtualIP: base.StringPtr("10.10.10.5"),
		},
	}
	resp, reqID, err := createServiceDomain(netClient, &svcDomain, token)
	if err != nil {
		return model.ServiceDomain{}, reqID, err
	}
	// Reset for compatibility
	if targetType == model.RealTargetType {
		svcDomain.Type = nil
	}
	svcDomain.ID = resp.ID
	return svcDomain, reqID, nil
}

func createNodesForServiceDomain(netClient *http.Client, tenantID, svcDomainID string, numOfNodes int, token string) ([]model.Node, error) {
	nodes := make([]model.Node, 0, numOfNodes)
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
				SvcDomainID: svcDomainID,
			},
			NodeCore: model.NodeCore{
				Name:         nodeName,
				SerialNumber: nodeSerialNumber,
				IPAddress:    nodeIP,
				Subnet:       nodeSubnet,
				Gateway:      nodeGateway,
			},
		}
		resp, _, err := createEntityV2(netClient, NODES_PATH, node, token)
		if err != nil {
			return nodes, err
		}
		node.ID = resp.ID
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func createServiceDomain(netClient *http.Client, svcDomain *model.ServiceDomain, token string) (model.CreateDocumentResponseV2, string, error) {
	resp, reqID, err := createEntityV2(netClient, SERVICE_DOMAINS_PATH, *svcDomain, token)
	if err == nil {
		svcDomain.ID = resp.ID
	}
	return resp, reqID, err
}

func getServiceDomains(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ServiceDomainListPayload, error) {
	response := model.ServiceDomainListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", SERVICE_DOMAINS_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func getNodes(netClient *http.Client, token string, pageIndex int, pageSize int) (model.NodeListPayload, error) {
	response := model.NodeListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", NODES_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func updateServiceDomain(netClient *http.Client, svcDomain model.ServiceDomain, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", SERVICE_DOMAINS_PATH, svcDomain.ID), svcDomain, token)
}

func updateNode(netClient *http.Client, node model.Node, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", NODES_PATH, node.ID), node, token)
}

func getServiceDomainsForProject(netClient *http.Client, projectID string, token string) ([]model.ServiceDomain, error) {
	svcDomains := []model.ServiceDomain{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/servicedomains", token, &svcDomains)
	return svcDomains, err
}

func getNodesForProject(netClient *http.Client, projectID string, token string) ([]model.Node, error) {
	nodes := []model.Node{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/nodes", token, &nodes)
	return nodes, err
}

func deleteServiceDomain(netClient *http.Client, svcDomainID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, SERVICE_DOMAINS_PATH, svcDomainID, token)
}

func deleteNode(netClient *http.Client, nodeID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, NODES_PATH, nodeID, token)
}

func getServiceDomainByID(netClient *http.Client, svcDomainID string, token string) (model.ServiceDomain, error) {
	svcDomain := model.ServiceDomain{}
	err := doGet(netClient, SERVICE_DOMAINS_PATH+"/"+svcDomainID, token, &svcDomain)
	return svcDomain, err
}

// get node by id
func getNodeByID(netClient *http.Client, nodeID string, token string) (model.Node, error) {
	node := model.Node{}
	err := doGet(netClient, NODES_PATH+"/"+nodeID, token, &node)
	return node, err
}

func TestServiceDomain(t *testing.T) {
	t.Parallel()
	t.Log("running TestServiceDomain test")

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

	t.Run("Test Nodes", func(t *testing.T) {
		// login as user to get token
		token := loginUser(t, netClient, user)

		svcDomain, _, err := createServiceDomainForTenant(netClient, tenantID, token, model.RealTargetType)
		require.NoError(t, err)
		svcDomainID := svcDomain.ID
		t.Logf("service domain created: %+v", svcDomain)

		nodes, err := createNodesForServiceDomain(netClient, tenantID, svcDomainID, 2, token)
		require.NoError(t, err)
		t.Logf("nodes created: %+v", nodes)
		// get nodes
		nodeResp, err := getNodes(netClient, token, 0, 10)
		require.NoError(t, err)
		if len(nodeResp.NodeList) != 2 {
			t.Fatalf("expected nodes count to be 2, got %d", len(nodeResp.NodeList))
		}
		t.Logf("nodes : %+v", nodeResp)
		svcDomainResp, err := getServiceDomains(netClient, token, 0, 10)
		require.NoError(t, err)
		t.Logf("service domains : %+v", svcDomainResp)
		if len(svcDomainResp.SvcDomainList) != 1 {
			t.Fatalf("expected service domain count of 1, found %d", len(svcDomainResp.SvcDomainList))
		}
		for _, node := range nodeResp.NodeList {
			_, _, err = deleteNode(netClient, node.ID, token)
			require.NoError(t, err)
		}
		t.Log("nodes deleted\n")
		_, _, err = deleteServiceDomain(netClient, svcDomainID, token)
		require.NoError(t, err)
		t.Log("service domain deleted\n")
		svcDomainResp, err = getServiceDomains(netClient, token, 0, 10)
		require.NoError(t, err)
		t.Logf("service domains : %+v", svcDomainResp)
		if len(svcDomainResp.SvcDomainList) != 0 {
			t.Fatalf("expected service domain count of 0, found %d", len(svcDomainResp.SvcDomainList))
		}
	})
}

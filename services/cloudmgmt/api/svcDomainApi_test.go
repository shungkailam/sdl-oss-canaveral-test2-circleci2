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
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally
func createServiceDomainWithLabels(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, labels []model.CategoryInfo) model.ServiceDomain {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	svcDomain := generateServiceDomain(tenantID, labels)
	resp, err := dbAPI.CreateServiceDomain(ctx, &svcDomain, nil)
	require.NoError(t, err)
	svcDomain.ID = resp.(model.CreateDocumentResponse).ID
	return svcDomain
}

func generateServiceDomain(tenantID string, labels []model.CategoryInfo) model.ServiceDomain {
	svcDomainName := "my-test-service-domain-" + base.GetUUID()
	domain := model.ServiceDomain{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		ServiceDomainCore: model.ServiceDomainCore{
			Name: svcDomainName,
		},
		Labels: labels,
	}
	return domain
}

func TestServiceDomain(t *testing.T) {
	t.Parallel()
	t.Log("running TestServiceDomain test")

	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	svcDomain := createServiceDomainWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{{
		ID:    categoryID,
		Value: TestCategoryValue1,
	}})
	svcDomainID := svcDomain.ID
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
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Service Domain", func(t *testing.T) {
		t.Log("running Create/Get/Delete Service Domain test")

		svcDomainName := "my-test-service-domain-cluster"
		svcDomainDescUpdated := "test edge desc"

		// update service domain
		doc2 := model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       svcDomainID,
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: svcDomainName,
			},
			Description: svcDomainDescUpdated,
			Labels: []model.CategoryInfo{{
				ID:    categoryID,
				Value: TestCategoryValue2,
			}},
		}
		// get service domain
		svcDomain, err := dbAPI.GetServiceDomain(ctx1, svcDomainID)
		require.NoError(t, err)
		_, err = dbAPI.GetServiceDomain(ctx2, svcDomainID)
		require.Error(t, err, "expect get service domain 2 to fail for non infra admin")
		_, err = dbAPI.GetServiceDomain(ctx3, svcDomainID)
		require.NoError(t, err)

		// select all service domains
		svcDomains, err := dbAPI.SelectAllServiceDomains(ctx1, nil)
		require.NoError(t, err)
		if len(svcDomains) != 1 {
			t.Fatalf("expect service domains count 1 to be 1, but got: %d", len(svcDomains))
		}
		svcDomains2, err := dbAPI.SelectAllServiceDomains(ctx2, nil)
		require.NoError(t, err)
		if len(svcDomains2) != 0 {
			t.Fatalf("expect service domains count 2 to be 0, but got: %d", len(svcDomains2))
		}
		svcDomains3, err := dbAPI.SelectAllServiceDomains(ctx3, nil)
		require.NoError(t, err)
		if len(svcDomains3) != 1 {
			t.Fatalf("expect service domains count 3 to be 1, but got: %d", len(svcDomains3))
		}

		_, err = dbAPI.UpdateServiceDomain(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)

		_, err = dbAPI.UpdateServiceDomain(ctx2, &doc2, nil)
		require.Error(t, err, "expect update service domain 2 to fail for non infra admin")
		_, err = dbAPI.UpdateEdge(ctx3, &doc2, nil)
		require.Error(t, err, "expect update service domain 3 to fail for non infra admin")

		// get service domain
		svcDomain, err = dbAPI.GetServiceDomain(ctx1, svcDomainID)
		require.NoError(t, err)
		if svcDomain.ID != svcDomainID || svcDomain.Name != svcDomainName || svcDomain.Description != svcDomainDescUpdated {
			if svcDomain.ID != svcDomainID {
				t.Fatalf("service domain id mismatch %s != %s", svcDomain.ID, svcDomainID)
			}
			if svcDomain.Name != svcDomainName {
				t.Fatal("service domain name mismatch")
			}
			if svcDomain.Description != svcDomainDescUpdated {
				t.Fatal("service domain description mismatch")
			}
			t.Fatal("service domain data mismatch")
		}
		svcDomain, err = dbAPI.GetServiceDomain(ctx2, svcDomainID)
		require.Error(t, err, "expected get service domain 2 to fail")
		svcDomain, err = dbAPI.GetServiceDomain(ctx3, svcDomainID)
		require.Error(t, err, "expect get service domain to fail since service domain is no longer in project")

		// select all service domains
		svcDomains, err = dbAPI.SelectAllServiceDomains(ctx1, nil)
		require.NoError(t, err)
		if len(svcDomains) != 1 {
			t.Fatalf("expect svcDomains count 1 to be 1, but got: %d", len(svcDomains))
		}
		svcDomains2, err = dbAPI.SelectAllServiceDomains(ctx2, nil)
		require.NoError(t, err)
		if len(svcDomains2) != 0 {
			t.Fatalf("expect svcDomains count 2 to be 0, but got: %d", len(svcDomains2))
		}
		svcDomains3, err = dbAPI.SelectAllServiceDomains(ctx3, nil)
		require.NoError(t, err)
		if len(svcDomains3) != 0 {
			t.Fatalf("expect svcDomains3 count 3 to be 0, but got: %d", len(svcDomains3))
		}
		for _, svcDomain := range svcDomains {
			testForMarshallability(t, svcDomain)
		}
		t.Log("get all service domains successful")

		// update one more time
		// update service domain
		doc2 = model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       svcDomainID,
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: svcDomainName,
			},
			Labels: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue1,
				},
			},
		}
		_, err = dbAPI.UpdateServiceDomain(ctx1, &doc2, func(ctx context.Context, doc interface{}) error {
			return nil
		})
		require.NoError(t, err)
		// get service domain
		svcDomain, err = dbAPI.GetServiceDomain(ctx1, svcDomainID)
		require.NoError(t, err)
		_, err = dbAPI.GetServiceDomain(ctx2, svcDomainID)
		require.Error(t, err, "expect get svcDomain 2 to fail for non infra admin")
		_, err = dbAPI.GetServiceDomain(ctx3, svcDomainID)
		require.NoError(t, err)

		// select all service domains
		svcDomains, err = dbAPI.SelectAllServiceDomains(ctx1, nil)
		require.NoError(t, err)
		if len(svcDomains) != 1 {
			t.Fatalf("expect edges count 1 to be 1, but got: %d", len(svcDomains))
		}
		svcDomains2, err = dbAPI.SelectAllServiceDomains(ctx2, nil)
		require.NoError(t, err)
		if len(svcDomains2) != 0 {
			t.Fatalf("expect svcDomains count 2 to be 0, but got: %d", len(svcDomains2))
		}
		svcDomains3, err = dbAPI.SelectAllServiceDomains(ctx3, nil)
		require.NoError(t, err)
		if len(svcDomains3) != 1 {
			t.Fatalf("expect svcDomains count 3 to be 1, but got: %d", len(svcDomains3))
		}

		// get edge handle
		// assert edge cert is not locked
		ec, err := dbAPI.GetEdgeCertByEdgeID(ctx1, svcDomainID)
		require.NoError(t, err)
		if ec.Locked {
			t.Fatal("unexpected edge cert locked")
		}

		token, err := crypto.EncryptPassword(svcDomainID)
		require.NoError(t, err)
		payload := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		edgeCert, err := dbAPI.GetServiceDomainHandle(ctx1, svcDomainID, payload)
		require.NoError(t, err, "GetServiceDomainHandle failed")

		if !edgeCert.Locked {
			t.Fatal("unexpected edge cert NOT locked")
		}
		testForMarshallability(t, edgeCert)
	})

	// select all service domains
	t.Run("SelectAllServiceDomains", func(t *testing.T) {
		t.Log("running SelectAllServiceDomains test")
		svcDomains, err := dbAPI.SelectAllServiceDomains(ctx1, nil)
		require.NoError(t, err)
		for _, svcDomain := range svcDomains {
			testForMarshallability(t, svcDomain)
		}
	})

	// select all service domains
	t.Run("ServiceDomain get service domain", func(t *testing.T) {
		t.Log("running Edge cluster get service domain test")

		authContextE := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "edge",
				"edgeId":      svcDomainID,
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContextE)
		svcDomains, err := dbAPI.SelectAllServiceDomains(newCtx, nil)
		require.NoError(t, err)
		if len(svcDomains) != 1 {
			t.Fatal("expected some service domains")
		}
		svcDomain, err := dbAPI.GetServiceDomain(newCtx, svcDomainID)
		require.NoError(t, err)
		t.Logf("Got service domain: %+v", svcDomain)
		edgeCert, err := dbAPI.GetEdgeCertByEdgeID(newCtx, svcDomainID)
		require.NoError(t, err)
		t.Logf("Got edge cert: %+v", edgeCert)
		projectRoles, err := dbAPI.GetEdgeProjectRoles(newCtx, svcDomainID)
		require.NoError(t, err)
		t.Logf("Got project roles: %+v", projectRoles)
	})
}

func TestServiceDomainW(t *testing.T) {
	t.Parallel()
	t.Log("running TestServiceDomainW test")
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

	t.Run("Create/Get/Delete Service Domain", func(t *testing.T) {
		t.Log("running Create/Get/Delete Service Domain test")

		svcDomainName := "my-test-service-domain-cluster"

		svcDomainDescription := "service domain desc 1"
		svcDomainDescUpdated := "service domain desc 2"

		// Service Domain object, leave ID blank and let create generate it
		doc := model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: svcDomainName,
			},
			Description: svcDomainDescription,
		}

		r, err := objToReader(doc)
		require.NoError(t, err)

		// create service domain
		var w bytes.Buffer
		err = dbAPI.CreateServiceDomainW(ctx1, &w, r, nil)
		require.NoError(t, err)
		resp := model.CreateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&resp)
		require.NoError(t, err)

		svcDomainID := resp.ID
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

		// update service domain
		doc = model.ServiceDomain{
			BaseModel: model.BaseModel{
				ID:       svcDomainID,
				TenantID: tenantID,
				Version:  5,
			},
			ServiceDomainCore: model.ServiceDomainCore{
				Name: svcDomainName,
			},
			Description: svcDomainDescUpdated,
		}
		r, err = objToReader(doc)
		require.NoError(t, err)

		err = dbAPI.UpdateServiceDomainW(ctx1, &w, r, nil)
		require.NoError(t, err)
		upResp := model.UpdateDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&upResp)
		require.NoError(t, err)

		// get service domain
		err = dbAPI.GetServiceDomainW(ctx1, svcDomainID, &w, nil)
		require.NoError(t, err)
		svcDomain := model.ServiceDomain{}
		err = json.NewDecoder(&w).Decode(&svcDomain)
		require.NoError(t, err)

		if svcDomain.ID != svcDomainID || svcDomain.Name != svcDomainName || svcDomain.Description != svcDomainDescUpdated {
			t.Fatal("service domain data mismatch")
		}
		// get all service domains
		// auth 1
		err = dbAPI.SelectAllServiceDomainsW(ctx1, &w, nil)
		require.NoError(t, err)
		svcDomains := model.ServiceDomainListPayload{}
		err = json.NewDecoder(&w).Decode(&svcDomains)
		require.NoError(t, err)
		if len(svcDomains.SvcDomainList) != 1 {
			t.Fatal("expect all svcDomains 1 count to be 1")
		}
		// auth 2
		err = dbAPI.SelectAllServiceDomainsW(ctx2, &w, nil)
		require.NoError(t, err)
		svcDomains = model.ServiceDomainListPayload{}
		err = json.NewDecoder(&w).Decode(&svcDomains)
		require.NoError(t, err)
		if len(svcDomains.SvcDomainList) != 0 {
			t.Fatal("expect all svcDomains 2 count to be 0")
		}
		// auth 3
		err = dbAPI.SelectAllServiceDomainsW(ctx3, &w, nil)
		require.NoError(t, err)
		svcDomains = model.ServiceDomainListPayload{}
		err = json.NewDecoder(&w).Decode(&svcDomains)
		require.NoError(t, err)
		if len(svcDomains.SvcDomainList) != 1 {
			t.Fatal("expect all svcDomains 3 count to be 1")
		}

		// select all vs select all W
		svcDomains1, err := dbAPI.SelectAllServiceDomains(ctx1, nil)
		require.NoError(t, err)
		// svcDomains2
		svcDomains2 := model.ServiceDomainListPayload{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllServiceDomainsW, &svcDomains2, &w)
		require.NoError(t, err)
		sort.Sort(model.ServiceDomainsByID(svcDomains1))
		sort.Sort(model.ServiceDomainsByID(svcDomains2.SvcDomainList))
		if !reflect.DeepEqual(&svcDomains1, &svcDomains2.SvcDomainList) {
			t.Fatalf("expect select service domains and select service domains w results to be equal %#v vs %#v", svcDomains1, svcDomains2.SvcDomainList)
		}

		// get all service domains for project
		// auth 1
		err = dbAPI.SelectAllServiceDomainsForProjectW(ctx1, projectID, &w, nil)
		require.Error(t, err, "expect all service domains 1 for project to fail")
		// auth 2
		err = dbAPI.SelectAllServiceDomainsForProjectW(ctx2, projectID, &w, nil)
		require.Error(t, err, "expect all service domains 2 for project to fail")
		// auth 3
		err = dbAPI.SelectAllServiceDomainsForProjectW(ctx3, projectID, &w, nil)
		require.NoError(t, err)
		svcDomains = model.ServiceDomainListPayload{}
		err = json.NewDecoder(&w).Decode(&svcDomains)
		require.NoError(t, err)
		if len(svcDomains.SvcDomainList) != 1 {
			t.Fatal("expect all svcDomains 3 for project count to be 1")
		}

		// Project role
		k8sListResp, err := dbAPI.SelectAllKubernetesClusters(ctx3, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		t.Logf("K8s clusters %+v: ", k8sListResp)
		require.Equal(t, len(k8sListResp.KubernetesClustersList), 0, "No k8s cluster must be returned")

		// get edge handle
		token, err := crypto.EncryptPassword(svcDomainID)
		require.NoError(t, err)
		payload2 := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    token,
		}
		r, err = objToReader(payload2)
		require.NoError(t, err)
		req := httptest.NewRequest(http.MethodPost, "/", r)
		// var w bytes.Buffer
		err = dbAPI.GetEdgeHandleW(ctx1, svcDomainID, &w, req)
		require.NoError(t, err, "GetServiceDomainHandle failed")

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
		// log.Printf("delete project successful, %v", delResp)

		// delete service domain
		err = dbAPI.DeleteServiceDomainW(ctx1, svcDomainID, &w, nil)
		require.NoError(t, err)
		delResp = model.DeleteDocumentResponseV2{}
		err = json.NewDecoder(&w).Decode(&delResp)
		require.NoError(t, err)
		t.Logf("delete service domain successful, %v", delResp)

	})

	// select all service domains
	t.Run("SelectAllServiceDomains", func(t *testing.T) {
		t.Log("running SelectAllServiceDomains test")
		var w bytes.Buffer
		err := dbAPI.SelectAllServiceDomainsW(ctx1, &w, nil)
		require.NoError(t, err)
		svcDomains := model.ServiceDomainListPayload{}
		err = json.NewDecoder(&w).Decode(&svcDomains)
		require.NoError(t, err)
		for _, svcDomain := range svcDomains.SvcDomainList {
			testForMarshallability(t, svcDomain)
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateServiceDomain(tenantID, nil)
		doc.ID = id
		return dbAPI.CreateServiceDomain(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetServiceDomain(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteServiceDomain(ctx1, id, nil)
	}))
}

func TestServiceDomainVirtualIP(t *testing.T) {
	t.Parallel()
	t.Logf("running TestServiceDomainVirtualIP")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	nodes := createNodeWithLabelsCommon(t, dbAPI, tenantID, nil, "EDGE", 1)
	svcDomainID := nodes[0].SvcDomainID
	ctx, _, _ := makeContext(tenantID, []string{})
	// Teardown
	defer func() {
		dbAPI.DeleteServiceDomain(ctx, svcDomainID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()
	t.Run("Test cluster virtual IP validation", func(t *testing.T) {
		t.Log("running cluster virtual IP validation test")
		nodeSerialNumber := base.GetUUID()
		nodeName := "second-node" + nodeSerialNumber
		nodeIP := "1.1.1.10"
		nodeSubnet := "255.255.255.0"
		nodeGateway := "1.1.1.1"
		node := model.Node{
			ServiceDomainEntityModel: model.ServiceDomainEntityModel{
				BaseModel: model.BaseModel{
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
		svcDomain, err := dbAPI.GetServiceDomain(ctx, svcDomainID)
		require.NoError(t, err)
		// Make sure virtual IP is unset before the tests
		if svcDomain.VirtualIP != nil && len(*svcDomain.VirtualIP) > 0 {
			t.Fatalf("Virtual IP must not be set for service domain %+v", svcDomain)
		}
		// Create second node in the same cluster without the virtual IP set
		_, err = dbAPI.CreateNode(ctx, &node, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "Virtual IP")

		// Now set the virtual IP
		svcDomain.VirtualIP = base.StringPtr("10.10.10.30")
		_, err = dbAPI.UpdateServiceDomain(ctx, &svcDomain, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.NoError(t, err)
		// Create second node now. It must succeed because virtual IP is set
		i, err := dbAPI.CreateNode(ctx, &node, nil)
		require.NoError(t, err)
		secondDeviceID := i.(model.CreateDocumentResponse).ID
		// Now, try to unset the virtual IP when there are already two nodes. It must fail
		svcDomain.VirtualIP = nil
		_, err = dbAPI.UpdateServiceDomain(ctx, &svcDomain, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.Errorf(t, err, "Virtual IP cannot be unset %+v", svcDomain)
		// Try to change the virtual IP when there are already two nodes. It must succeed
		svcDomain.VirtualIP = base.StringPtr("10.10.10.31")
		_, err = dbAPI.UpdateServiceDomain(ctx, &svcDomain, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.NoError(t, err)
		// Delete the second node
		_, err = dbAPI.DeleteNode(ctx, secondDeviceID, nil)
		require.NoError(t, err)
		svcDomain.VirtualIP = nil
		// Now, try to unset the virtual IP when there is only one node. It must succeed
		_, err = dbAPI.UpdateServiceDomain(ctx, &svcDomain, func(ctx context.Context, doc interface{}) error {
			t.Logf("Callback called with %+v", doc)
			return nil
		})
		require.NoError(t, err)
	})
}

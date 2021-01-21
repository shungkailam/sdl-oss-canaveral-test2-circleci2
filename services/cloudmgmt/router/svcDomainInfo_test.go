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
	"reflect"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	SERVICE_DOMAINS_INFO_PATH = "/v1.0/servicedomainsinfo"
)

func getServiceDomainsInfo(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ServiceDomainInfoListPayload, error) {
	response := model.ServiceDomainInfoListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", SERVICE_DOMAINS_INFO_PATH, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

func updateServiceDomainInfo(netClient *http.Client, svcDomain model.ServiceDomainInfo, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, fmt.Sprintf("%s/%s", SERVICE_DOMAINS_INFO_PATH, svcDomain.ID), svcDomain, token)
}

func getServiceDomainsInfoForProject(netClient *http.Client, projectID string, token string) ([]model.ServiceDomainInfo, error) {
	svcDomainInfos := []model.ServiceDomainInfo{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/servicedomainsinfo", token, &svcDomainInfos)
	return svcDomainInfos, err
}

func deleteServiceDomainInfo(netClient *http.Client, svcDomainID string, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, SERVICE_DOMAINS_INFO_PATH, svcDomainID, token)
}

func getServiceDomainInfoByID(netClient *http.Client, svcDomainID string, token string) (model.ServiceDomainInfo, error) {
	svcDomainInfo := model.ServiceDomainInfo{}
	err := doGet(netClient, SERVICE_DOMAINS_INFO_PATH+"/"+svcDomainID, token, &svcDomainInfo)
	return svcDomainInfo, err
}

func TestServiceDomainInfo(t *testing.T) {
	t.Parallel()
	t.Log("running TestServiceDomainInfo test")

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

		payload, err := getServiceDomainsInfo(netClient, token, 0, 10)
		require.NoError(t, err)
		t.Logf("got response: %+v", payload)
		svcDomainInfos := payload.SvcDomainInfoList
		if len(svcDomainInfos) != 1 {
			t.Fatalf("expected 1 service domain info, found %d", len(svcDomainInfos))
		}
		svcDomainInfo := svcDomainInfos[0]
		if svcDomainInfo.ID != svcDomainID || svcDomainInfo.SvcDomainID != svcDomainID {
			t.Fatalf("svcDomainInfo.ID != svcDomainID -> %v, svcDomainInfo.SvcDomainID != svcDomainID -> %v", svcDomainInfo.ID != svcDomainID, svcDomainInfo.SvcDomainID != svcDomainID)
		}
		svcDomainInfo.Artifacts = map[string]interface{}{"cluster": "test"}
		updateResp, _, err := updateServiceDomainInfo(netClient, svcDomainInfo, token)
		require.NoError(t, err)
		if updateResp.ID != svcDomainID {
			t.Fatalf("expected %s, found %s", svcDomainID, updateResp.ID)
		}
		svcDomainInfo1, err := getServiceDomainInfoByID(netClient, svcDomainID, token)
		require.NoError(t, err)
		t.Logf("got response %+v", svcDomainInfo1)
		if !reflect.DeepEqual(svcDomainInfo1.Artifacts, svcDomainInfo.Artifacts) {
			t.Fatalf("expected response %+v, found %+v", svcDomainInfo.Artifacts, svcDomainInfo1.Artifacts)
		}
	})
}

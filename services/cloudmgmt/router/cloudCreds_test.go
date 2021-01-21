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

const (
	CLOUD_CREDS_PATH     = "/v1/cloudcreds"
	CLOUD_CREDS_PATH_NEW = "/v1.0/cloudprofiles"
)

// create cloudcreds
func createCloudCreds(netClient *http.Client, cloudcreds *model.CloudCreds, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, CLOUD_CREDS_PATH, *cloudcreds, token)
	if err == nil {
		cloudcreds.ID = resp.ID
	}
	return resp, reqID, err
}

// update cloudcreds
func updateCloudCredss(netClient *http.Client, cloudcredsID string, cloudcreds model.CloudCreds, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", CLOUD_CREDS_PATH, cloudcredsID), cloudcreds, token)
}

// get cloudcredss
func getCloudCredss(netClient *http.Client, token string) ([]model.CloudCreds, error) {
	cloudcredss := []model.CloudCreds{}
	err := doGet(netClient, CLOUD_CREDS_PATH, token, &cloudcredss)
	return cloudcredss, err
}
func getCloudCredssNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.CloudCredsListResponsePayload, error) {
	response := model.CloudCredsListResponsePayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", CLOUD_CREDS_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}
func getCloudCredssForProject(netClient *http.Client, projectID string, token string) ([]model.CloudCreds, error) {
	cloudcredss := []model.CloudCreds{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/cloudcreds", token, &cloudcredss)
	return cloudcredss, err
}

// delete cloudcreds
func deleteCloudCreds(netClient *http.Client, cloudcredsID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, CLOUD_CREDS_PATH, cloudcredsID, token)
}

// get cloudcreds by id
func getCloudCredsByID(netClient *http.Client, cloudcredsID string, token string) (model.CloudCreds, error) {
	cloudcreds := model.CloudCreds{}
	err := doGet(netClient, CLOUD_CREDS_PATH+"/"+cloudcredsID, token, &cloudcreds)
	return cloudcreds, err
}

func makeCloudCreds() model.CloudCreds {
	return model.CloudCreds{
		Name:        "aws-cloud-creds-name",
		Type:        "AWS",
		Description: "aws-cloud-creds-desc",
		AWSCredential: &model.AWSCredential{
			AccessKey: "foo",
			Secret:    "bar",
		},
		GCPCredential: nil,
	}
}

func TestCloudCreds(t *testing.T) {
	t.Parallel()
	t.Log("running TestCloudCreds test")

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

	t.Run("Test CloudCreds", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		cloudcreds := makeCloudCreds()
		_, _, err := createCloudCreds(netClient, &cloudcreds, token)
		require.NoError(t, err)

		cloudcredss, err := getCloudCredss(netClient, token)
		require.NoError(t, err)
		t.Logf("got cloudcredss: %+v", cloudcredss)
		if len(cloudcredss) != 1 {
			t.Fatalf("expect count of cloud creds to be 1, got %d", len(cloudcredss))
		}

		cloudcredsJ, err := getCloudCredsByID(netClient, cloudcreds.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(cloudcredss[0], cloudcredsJ) {
			t.Fatalf("expect cloud creds J equal, but %+v != %+v", cloudcredss[0], cloudcredsJ)
		}

		project := makeExplicitProject(tenantID, []string{cloudcredss[0].ID}, nil, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		ccForProject, err := getCloudCredssForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if !reflect.DeepEqual(ccForProject[0], cloudcredss[0]) {
			t.Fatalf("expect cloud creds to equal, but %+v != %+v", ccForProject[0], cloudcredss[0])
		}

		// update cloud creds
		cloudcredsID := cloudcreds.ID
		cloudcreds.ID = ""
		cloudcreds.Name = fmt.Sprintf("%s-updated", cloudcreds.Name)
		ur, _, err := updateCloudCredss(netClient, cloudcredsID, cloudcreds, token)
		require.NoError(t, err)
		if ur.ID != cloudcredsID {
			t.Fatal("expect update cloud creds id to match")
		}

		resp, _, err := deleteCloudCreds(netClient, cloudcredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudcredsID {
			t.Fatal("delete cloud creds id mismatch")
		}
	})

}

func TestCloudCredsPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestCloudCredsPaging test")

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

	t.Run("Test CloudCreds Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// randomly create some cloud creds
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d cloudcreds...", n)
		for i := 0; i < n; i++ {
			cloudcreds := model.CloudCreds{
				Name:        fmt.Sprintf("cloud-creds-name-%s", base.GetUUID()),
				Type:        "AWS",
				Description: "aws-cloud-creds-desc",
				AWSCredential: &model.AWSCredential{
					AccessKey: "foo",
					Secret:    "bar",
				},
				GCPCredential: nil,
			}
			_, _, err := createCloudCreds(netClient, &cloudcreds, token)
			require.NoError(t, err)
		}

		cloudcredss, err := getCloudCredss(netClient, token)
		require.NoError(t, err)
		if len(cloudcredss) != n {
			t.Fatalf("expected cloud creds count to be %d, but got %d", n, len(cloudcredss))
		}
		sort.Sort(model.CloudCredssByID(cloudcredss))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pCloudCredss := []model.CloudCreds{}
		nRemain := n
		t.Logf("fetch %d cloudcreds using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nccs, err := getCloudCredssNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if nccs.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, nccs.PageIndex)
			}
			if nccs.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, nccs.PageSize)
			}
			if nccs.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, nccs.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(nccs.CloudCredsList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nccs.CloudCredsList))
			}
			nRemain -= pageSize
			for _, cc := range nccs.CloudCredsList {
				pCloudCredss = append(pCloudCredss, cc)
			}
		}

		// verify paging api gives same result as old api
		for i := range pCloudCredss {
			if !reflect.DeepEqual(cloudcredss[i], pCloudCredss[i]) {
				t.Fatalf("expect cloudcreds equal, but %+v != %+v", cloudcredss[i], pCloudCredss[i])
			}
		}
		t.Log("get cloudcreds from paging api gives same result as old api")

		for _, cloudcreds := range cloudcredss {
			resp, _, err := deleteCloudCreds(netClient, cloudcreds.ID, token)
			require.NoError(t, err)
			if resp.ID != cloudcreds.ID {
				t.Fatal("delete cloudcreds id mismatch")
			}
		}
	})
}

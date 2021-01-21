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
	CONTAINER_REGISTRIES_PATH     = "/v1/containerregistries"
	CONTAINER_REGISTRIES_PATH_NEW = "/v1.0/containerregistries"
)

// create containerregistry
func createContainerRegistry(netClient *http.Client, containerregistry *model.ContainerRegistry, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, CONTAINER_REGISTRIES_PATH, *containerregistry, token)
	if err == nil {
		containerregistry.ID = resp.ID
	}
	return resp, reqID, err
}

// update containerregistry
func updateContainerRegistry(netClient *http.Client, containerregistryID string, containerregistry model.ContainerRegistry, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", CONTAINER_REGISTRIES_PATH, containerregistryID), containerregistry, token)
}

// get containerregistry
func getContainerRegistries(netClient *http.Client, token string) ([]model.ContainerRegistry, error) {
	containerregistries := []model.ContainerRegistry{}
	err := doGet(netClient, CONTAINER_REGISTRIES_PATH, token, &containerregistries)
	return containerregistries, err
}
func getContainerRegistriesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ContainerRegistryListPayload, error) {
	response := model.ContainerRegistryListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", CONTAINER_REGISTRIES_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}
func getContainerRegistriesForProject(netClient *http.Client, projectID string, token string) ([]model.ContainerRegistry, error) {
	containerregistries := []model.ContainerRegistry{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/containerregistries", token, &containerregistries)
	return containerregistries, err
}

// delete containerregistry
func deleteContainerRegistry(netClient *http.Client, containerregistryID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, CONTAINER_REGISTRIES_PATH, containerregistryID, token)
}

// get containerregistry by id
func getContainerRegistryByID(netClient *http.Client, containerregistryID string, token string) (model.ContainerRegistry, error) {
	containerregistry := model.ContainerRegistry{}
	err := doGet(netClient, CONTAINER_REGISTRIES_PATH+"/"+containerregistryID, token, &containerregistry)
	return containerregistry, err
}

func TestContainerRegistry(t *testing.T) {
	t.Parallel()
	t.Log("running TestContainerRegistry test")

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

	t.Run("Test ContainerRegistry", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		cloudcreds := model.CloudCreds{
			Name:        "aws-cloud-creds-name",
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
		cloudCredsID := cloudcreds.ID

		containerRegistryName := "aws-registry-name"
		containerRegistryDesc := "aws-registry-desc"

		// ContainerRegistry object, leave ID blank and let create generate it
		containerregistry := model.ContainerRegistry{
			Name:         containerRegistryName,
			Type:         "AWS",
			Server:       "a.b.c.d.e.f",
			CloudCredsID: cloudCredsID,
			Description:  containerRegistryDesc,
			UserName:     "username",
			Email:        "test@example.com",
		}
		_, _, err = createContainerRegistry(netClient, &containerregistry, token)
		require.NoError(t, err)
		containerRegistryID := containerregistry.ID

		containerregistries, err := getContainerRegistries(netClient, token)
		require.NoError(t, err)
		t.Logf("got containerregistries: %+v", containerregistries)
		if len(containerregistries) != 1 {
			t.Fatalf("expected container registry profile count to be 1, got %d", len(containerregistries))
		}

		project := makeExplicitProject(tenantID, []string{cloudCredsID}, []string{containerRegistryID}, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		projects, err := getProjects(netClient, token)
		require.NoError(t, err)
		if len(projects) != 1 {
			t.Fatalf("expected projects count 1, got %d", len(projects))
		}

		crForProject, err := getContainerRegistriesForProject(netClient, projectID, token)
		require.NoError(t, err)
		if len(crForProject) != 1 {
			t.Fatalf("expected container registry count 1, got %d", len(crForProject))
		}
		if !reflect.DeepEqual(crForProject[0], containerregistries[0]) {
			t.Fatalf("expect container registry to equal, but %+v != %+v", crForProject[0], containerregistries[0])
		}

		containerregistry.ID = ""
		containerregistry.Name = fmt.Sprintf("%s-updated", containerregistry.Name)
		ur, _, err := updateContainerRegistry(netClient, containerRegistryID, containerregistry, token)
		require.NoError(t, err)
		if ur.ID != containerRegistryID {
			t.Fatal("expect update container registry id to match")
		}

		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("project id mismatch in delete")
		}

		resp, _, err = deleteContainerRegistry(netClient, containerRegistryID, token)
		require.NoError(t, err)
		if resp.ID != containerRegistryID {
			t.Fatal("container profile id mismatch in delete")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("cloud profile id mismatch in delete")
		}

	})

}

func TestContainerRegistryPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestContainerRegistryPaging test")

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

	t.Run("Test ContainerRegistry Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		cloudcreds := model.CloudCreds{
			Name:        "aws-cloud-creds-name",
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
		cloudCredsID := cloudcreds.ID

		// randomly create some container registries
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d container registries...", n)
		for i := 0; i < n; i++ {
			containerRegistryName := fmt.Sprintf("aws-container-registry-name-%s", base.GetUUID())
			containerRegistryDesc := "aws-registry-desc"

			// ContainerRegistry object, leave ID blank and let create generate it
			containerregistry := model.ContainerRegistry{
				Name:         containerRegistryName,
				Type:         "AWS",
				Server:       "a.b.c.d.e.f",
				CloudCredsID: cloudCredsID,
				Description:  containerRegistryDesc,
			}
			_, _, err = createContainerRegistry(netClient, &containerregistry, token)
			require.NoError(t, err)
		}

		containerregistries, err := getContainerRegistries(netClient, token)
		require.NoError(t, err)
		t.Logf("got containerregistries: %+v", containerregistries)
		if len(containerregistries) != n {
			t.Fatalf("expected containerregistries count to be %d, but got %d", n, len(containerregistries))
		}
		sort.Sort(model.ContainerRegistriesByID(containerregistries))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pContainerRegistries := []model.ContainerRegistry{}
		nRemain := n
		t.Logf("fetch %d container registries using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nccs, err := getContainerRegistriesNew(netClient, token, i, pageSize)
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
			if len(nccs.ContainerRegistryListV2) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nccs.ContainerRegistryListV2))
			}
			nRemain -= pageSize
			for _, cc := range model.ContainerRegistriesByIDV2(nccs.ContainerRegistryListV2).FromV2() {
				pContainerRegistries = append(pContainerRegistries, cc)
			}
		}

		// verify paging api gives same result as old api
		for i := range pContainerRegistries {
			if !reflect.DeepEqual(containerregistries[i], pContainerRegistries[i]) {
				t.Fatalf("expect container registry equal, but %+v != %+v", containerregistries[i], pContainerRegistries[i])
			}
		}
		t.Log("get containerregistries from paging api gives same result as old api")

		for _, containerregistry := range containerregistries {
			resp, _, err := deleteContainerRegistry(netClient, containerregistry.ID, token)
			require.NoError(t, err)
			if resp.ID != containerregistry.ID {
				t.Fatal("delete containerregistry id mismatch")
			}
		}

		resp, _, err := deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("cloud profile id mismatch in delete")
		}

	})

}

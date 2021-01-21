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
	SCRIPT_RUNTIMES_PATH     = "/v1/scriptruntimes"
	SCRIPT_RUNTIMES_PATH_NEW = "/v1.0/runtimeenvironments"
)

// create scriptruntime
func createScriptRuntime(netClient *http.Client, scriptruntime *model.ScriptRuntime, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, SCRIPT_RUNTIMES_PATH, *scriptruntime, token)
	if err == nil {
		scriptruntime.ID = resp.ID
	}
	return resp, reqID, err
}

// update scriptruntime
func updateScriptRuntime(netClient *http.Client, scriptruntimeID string, scriptruntime model.ScriptRuntime, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", SCRIPT_RUNTIMES_PATH, scriptruntimeID), scriptruntime, token)
}

// get scriptruntimes
func getScriptRuntimes(netClient *http.Client, token string) ([]model.ScriptRuntime, error) {
	scriptruntimes := []model.ScriptRuntime{}
	err := doGet(netClient, SCRIPT_RUNTIMES_PATH, token, &scriptruntimes)
	return scriptruntimes, err
}
func getScriptRuntimesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ScriptRuntimeListPayload, error) {
	response := model.ScriptRuntimeListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", SCRIPT_RUNTIMES_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}
func getScriptRuntimesForProject(netClient *http.Client, projectID string, token string) ([]model.ScriptRuntime, error) {
	scriptruntimes := []model.ScriptRuntime{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/scriptruntimes", token, &scriptruntimes)
	return scriptruntimes, err
}

// delete scriptruntime
func deleteScriptRuntime(netClient *http.Client, scriptruntimeID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, SCRIPT_RUNTIMES_PATH, scriptruntimeID, token)
}

// get scriptruntime by id
func getScriptRuntimeByID(netClient *http.Client, scriptruntimeID string, token string) (model.ScriptRuntime, error) {
	scriptruntime := model.ScriptRuntime{}
	err := doGet(netClient, SCRIPT_RUNTIMES_PATH+"/"+scriptruntimeID, token, &scriptruntime)
	return scriptruntime, err
}

func TestScriptRuntime(t *testing.T) {
	t.Parallel()
	t.Log("running TestScriptRuntime test")

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

	t.Run("Test ScriptRuntime", func(t *testing.T) {
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

		dockerProfileName := "aws-registry-name"
		dockerProfileDesc := "aws-registry-desc"

		// DockerProfile object, leave ID blank and let create generate it
		dockerprofile := model.DockerProfile{
			Name:         dockerProfileName,
			Type:         "AWS",
			Server:       "a.b.c.d.e.f",
			CloudCredsID: cloudCredsID,
			Description:  dockerProfileDesc,
			UserName:     "username",
			Email:        "test@example.com",
			Credentials:  "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
		}
		_, _, err = createDockerProfile(netClient, &dockerprofile, token)
		require.NoError(t, err)
		dockerProfileID := dockerprofile.ID

		project := makeExplicitProject(tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dockerfile := "docker file"
		// dockerfileUpdated := "docker file updated"

		scriptruntime := model.ScriptRuntime{
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         false,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfile,
			},
			ProjectID: projectID,
		}

		_, _, err = createScriptRuntime(netClient, &scriptruntime, token)
		require.NoError(t, err)

		scriptruntimes, err := getScriptRuntimes(netClient, token)
		require.NoError(t, err)
		t.Logf("got scriptruntimes: %+v", scriptruntimes)
		if len(scriptruntimes) != 1 {
			t.Fatalf("expected script runtime count to be 1, got %d", len(scriptruntimes))
		}

		scriptruntimes2, err := getScriptRuntimesForProject(netClient, projectID, token)
		require.NoError(t, err)
		if len(scriptruntimes2) != 1 {
			t.Fatalf("expected script runtime 2 count to be 1, got %d", len(scriptruntimes2))
		}
		if !reflect.DeepEqual(scriptruntimes2, scriptruntimes) {
			t.Fatalf("expect script runtime equal, but %+v != %+v", scriptruntimes2, scriptruntimes)
		}

		scriptruntimeID := scriptruntime.ID
		scriptruntime.ID = ""
		scriptruntime.TenantID = ""
		ur, _, err := updateScriptRuntime(netClient, scriptruntimeID, scriptruntime, token)
		require.NoError(t, err)
		if ur.ID != scriptruntimeID {
			t.Fatal("update script runtime id mismatch")
		}

		resp, _, err := deleteScriptRuntime(netClient, scriptruntimeID, token)
		require.NoError(t, err)
		if resp.ID != scriptruntimeID {
			t.Fatal("delete script runtime id mismatch")
		}

		resp, _, err = deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("project id mismatch in delete")
		}

		resp, _, err = deleteDockerProfile(netClient, dockerProfileID, token)
		require.NoError(t, err)
		if resp.ID != dockerProfileID {
			t.Fatal("docker profile id mismatch in delete")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("cloud creds id mismatch in delete")
		}

	})

}

func TestScriptRuntimePaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestScriptRuntimePaging test")

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

	t.Run("Test ScriptRuntime Paging", func(t *testing.T) {
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

		dockerProfileName := "aws-registry-name"
		dockerProfileDesc := "aws-registry-desc"

		// DockerProfile object, leave ID blank and let create generate it
		dockerprofile := model.DockerProfile{
			Name:         dockerProfileName,
			Type:         "AWS",
			Server:       "a.b.c.d.e.f",
			CloudCredsID: cloudCredsID,
			Description:  dockerProfileDesc,
			UserName:     "username",
			Email:        "test@example.com",
			Credentials:  "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
		}
		_, _, err = createDockerProfile(netClient, &dockerprofile, token)
		require.NoError(t, err)
		dockerProfileID := dockerprofile.ID

		project := makeExplicitProject(tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		dockerfile := "docker file"
		// dockerfileUpdated := "docker file updated"

		// randomly create some script runtimes
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d scriptruntimes...", n)
		for i := 0; i < n; i++ {
			scriptruntime := model.ScriptRuntime{
				ScriptRuntimeCore: model.ScriptRuntimeCore{
					Name:            fmt.Sprintf("script-runtime-name-%s", base.GetUUID()),
					Description:     "script runtime desc",
					Language:        "python",
					Builtin:         false,
					DockerRepoURI:   "docker-repo-uri",
					DockerProfileID: dockerProfileID,
					Dockerfile:      dockerfile,
				},
				ProjectID: projectID,
			}

			_, _, err = createScriptRuntime(netClient, &scriptruntime, token)
			require.NoError(t, err)
		}

		scriptruntimes, err := getScriptRuntimes(netClient, token)
		require.NoError(t, err)
		if len(scriptruntimes) != n {
			t.Fatalf("expected scriptruntimes count to be %d, but got %d", n, len(scriptruntimes))
		}
		sort.Sort(model.ScriptRuntimesByID(scriptruntimes))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pScriptRuntimes := []model.ScriptRuntime{}
		nRemain := n
		t.Logf("fetch %d scriptruntimes using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nscpts, err := getScriptRuntimesNew(netClient, token, i, pageSize)
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
			if len(nscpts.ScriptRuntimeList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nscpts.ScriptRuntimeList))
			}
			nRemain -= pageSize
			for _, sr := range nscpts.ScriptRuntimeList {
				pScriptRuntimes = append(pScriptRuntimes, sr)
			}
		}

		// verify paging api gives same result as old api
		for i := range pScriptRuntimes {
			if !reflect.DeepEqual(scriptruntimes[i], pScriptRuntimes[i]) {
				t.Fatalf("expect script equal, but %+v != %+v", scriptruntimes[i], pScriptRuntimes[i])
			}
		}
		t.Log("get scripts from paging api gives same result as old api")

		for _, scriptruntime := range scriptruntimes {
			resp, _, err := deleteScriptRuntime(netClient, scriptruntime.ID, token)
			require.NoError(t, err)
			if resp.ID != scriptruntime.ID {
				t.Fatal("delete scriptruntime id mismatch")
			}
		}

		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("project id mismatch in delete")
		}

		resp, _, err = deleteDockerProfile(netClient, dockerProfileID, token)
		require.NoError(t, err)
		if resp.ID != dockerProfileID {
			t.Fatal("docker profile id mismatch in delete")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("cloud creds id mismatch in delete")
		}

	})

}

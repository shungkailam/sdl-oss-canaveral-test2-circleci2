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
	SCRIPTS_PATH     = "/v1/scripts"
	SCRIPTS_PATH_NEW = "/v1.0/functions"
)

// create script
func createScript(netClient *http.Client, script *model.Script, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, SCRIPTS_PATH, *script, token)
	if err == nil {
		script.ID = resp.ID
	}
	return resp, reqID, err
}

// update script
func updateScript(netClient *http.Client, scriptID string, script model.Script, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", SCRIPTS_PATH, scriptID), script, token)
}

// get scripts
func getScripts(netClient *http.Client, token string) ([]model.Script, error) {
	scripts := []model.Script{}
	err := doGet(netClient, SCRIPTS_PATH, token, &scripts)
	return scripts, err
}
func getScriptsNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ScriptListPayload, error) {
	response := model.ScriptListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", SCRIPTS_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}
func getScriptsForProject(netClient *http.Client, projectID string, token string) ([]model.Script, error) {
	scripts := []model.Script{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID+"/scripts", token, &scripts)
	return scripts, err
}

// delete script
func deleteScript(netClient *http.Client, scriptID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, SCRIPTS_PATH, scriptID, token)
}

// get script by id
func getScriptByID(netClient *http.Client, scriptID string, token string) (model.Script, error) {
	script := model.Script{}
	err := doGet(netClient, SCRIPTS_PATH+"/"+scriptID, token, &script)
	return script, err
}

func TestScript(t *testing.T) {
	t.Parallel()
	t.Log("running TestScript test")

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

	t.Run("Test Script", func(t *testing.T) {
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

		scriptName := "script name"
		scriptType := "Transformation"
		scriptLanguage := "Python"
		scriptEnvrionment := "python tensorflow"
		scriptCode := "def main: print"
		// scriptCodeUpdated := "def main: print 'hello'"

		// Script object, leave ID blank and let create generate it
		script := model.Script{
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCode,
				Builtin:     false,
				ProjectID:   projectID,
				RuntimeID:   scriptruntime.ID,
			},
			Params: []model.ScriptParam{},
		}

		_, _, err = createScript(netClient, &script, token)
		require.NoError(t, err)

		scripts, err := getScripts(netClient, token)
		require.NoError(t, err)
		t.Logf("got scripts: %+v", scripts)
		if len(scripts) != 1 {
			t.Fatalf("expected scripts count 1, got %d", len(scripts))
		}

		scripts2, err := getScriptsForProject(netClient, project.ID, token)
		require.NoError(t, err)
		if len(scripts2) != 1 {
			t.Fatalf("expected scripts2 count 1, got %d", len(scripts2))
		}
		if !reflect.DeepEqual(scripts2, scripts) {
			t.Fatalf("expect script equal, but %+v != %+v", scripts2, scripts)
		}

		scriptID := script.ID
		script.ID = ""
		script.TenantID = ""
		ur, _, err := updateScript(netClient, scriptID, script, token)
		require.NoError(t, err)
		if ur.ID != scriptID {
			t.Fatal("update script id mismatch")
		}

		resp, _, err := deleteScript(netClient, scriptID, token)
		require.NoError(t, err)
		if resp.ID != scriptID {
			t.Fatal("delete script id mismatch")
		}

		resp, _, err = deleteScriptRuntime(netClient, scriptruntime.ID, token)
		require.NoError(t, err)
		if resp.ID != scriptruntime.ID {
			t.Fatal("delete scriptruntime id mismatch")
		}

		resp, _, err = deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
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

func TestScriptPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestScriptPaging test")

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

	t.Run("Test Script Paging", func(t *testing.T) {
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

		// randomly create some scripts
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d scripts...", n)
		for i := 0; i < n; i++ {
			scriptName := fmt.Sprintf("script name-%s", base.GetUUID())
			scriptType := "Transformation"
			scriptLanguage := "Python"
			scriptEnvrionment := "python tensorflow"
			scriptCode := "def main: print"
			// scriptCodeUpdated := "def main: print 'hello'"

			// Script object, leave ID blank and let create generate it
			script := model.Script{
				ScriptCore: model.ScriptCore{
					Name:        scriptName,
					Type:        scriptType,
					Language:    scriptLanguage,
					Environment: scriptEnvrionment,
					Code:        scriptCode,
					Builtin:     false,
					ProjectID:   projectID,
					RuntimeID:   scriptruntime.ID,
				},
				Params: []model.ScriptParam{},
			}

			_, _, err = createScript(netClient, &script, token)
			require.NoError(t, err)
		}

		scripts, err := getScripts(netClient, token)
		require.NoError(t, err)
		if len(scripts) != n {
			t.Fatalf("expected scripts count to be %d, but got %d", n, len(scripts))
		}
		sort.Sort(model.ScriptsByID(scripts))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pScripts := []model.Script{}
		nRemain := n
		t.Logf("fetch %d scripts using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nscpts, err := getScriptsNew(netClient, token, i, pageSize)
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
			if len(nscpts.ScriptList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nscpts.ScriptList))
			}
			nRemain -= pageSize
			for _, sr := range nscpts.ScriptList {
				pScripts = append(pScripts, sr)
			}
		}

		// verify paging api gives same result as old api
		for i := range pScripts {
			if !reflect.DeepEqual(scripts[i], pScripts[i]) {
				t.Fatalf("expect script equal, but %+v != %+v", scripts[i], pScripts[i])
			}
		}
		t.Log("get scripts from paging api gives same result as old api")

		for _, script := range scripts {
			resp, _, err := deleteScript(netClient, script.ID, token)
			require.NoError(t, err)
			if resp.ID != script.ID {
				t.Fatal("delete script id mismatch")
			}
		}

		resp, _, err := deleteScriptRuntime(netClient, scriptruntime.ID, token)
		require.NoError(t, err)
		if resp.ID != scriptruntime.ID {
			t.Fatal("delete scriptruntime id mismatch")
		}

		resp, _, err = deleteProject(netClient, project.ID, token)
		require.NoError(t, err)
		if resp.ID != project.ID {
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

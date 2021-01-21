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
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	DOCKER_PROFILES_PATH = "/v1/dockerprofiles"
)

// create dockerprofile
func createDockerProfile(netClient *http.Client, dockerprofile *model.DockerProfile, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, DOCKER_PROFILES_PATH, *dockerprofile, token)
	if err == nil {
		dockerprofile.ID = resp.ID
	}
	return resp, reqID, err
}

// update dockerprofile
func updateDockerProfile(netClient *http.Client, dockerprofileID string, dockerprofile model.DockerProfile, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", DOCKER_PROFILES_PATH, dockerprofileID), dockerprofile, token)
}

// get dockerprofiles
func getDockerProfiles(netClient *http.Client, token string) ([]model.DockerProfile, error) {
	dockerprofiles := []model.DockerProfile{}
	err := doGet(netClient, DOCKER_PROFILES_PATH, token, &dockerprofiles)
	return dockerprofiles, err
}

// delete dockerprofile
func deleteDockerProfile(netClient *http.Client, dockerprofileID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, DOCKER_PROFILES_PATH, dockerprofileID, token)
}

// get dockerprofile by id
func getDockerProfileByID(netClient *http.Client, dockerprofileID string, token string) (model.DockerProfile, error) {
	dockerprofile := model.DockerProfile{}
	err := doGet(netClient, DOCKER_PROFILES_PATH+"/"+dockerprofileID, token, &dockerprofile)
	return dockerprofile, err
}

func TestDockerProfile(t *testing.T) {
	t.Parallel()
	t.Log("running TestDockerProfile test")

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

	t.Run("Test DockerProfile", func(t *testing.T) {
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

		dockerprofiles, err := getDockerProfiles(netClient, token)
		require.NoError(t, err)
		t.Logf("got dockerprofiles: %+v", dockerprofiles)
		if len(dockerprofiles) != 1 {
			t.Fatalf("expected docker profile count to be 1, got %d", len(dockerprofiles))
		}

		project := makeExplicitProject(tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{user.ID}, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		projectID := project.ID

		projects, err := getProjects(netClient, token)
		require.NoError(t, err)
		if len(projects) != 1 {
			t.Fatalf("expected projects count 1, got %d", len(projects))
		}

		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("project id mismatch in delete")
		}

		dockerprofile.Name = fmt.Sprintf("%s-updated", dockerprofile.Name)
		dockerprofile.ID = ""
		ur, _, err := updateDockerProfile(netClient, dockerProfileID, dockerprofile, token)
		require.NoError(t, err)
		if ur.ID != dockerProfileID {
			t.Fatal("docker profile id mismatch in update")
		}

		resp, _, err = deleteDockerProfile(netClient, dockerProfileID, token)
		require.NoError(t, err)
		if resp.ID != dockerProfileID {
			t.Fatal("docker profile id mismatch in delete")
		}

		resp, _, err = deleteCloudCreds(netClient, cloudCredsID, token)
		require.NoError(t, err)
		if resp.ID != cloudCredsID {
			t.Fatal("cloud profile id mismatch in delete")
		}

	})

}

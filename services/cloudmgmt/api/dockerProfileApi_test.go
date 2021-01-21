package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createAWSDockerProfile(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsID string) model.DockerProfile {
	// create docker profile
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any@email.com",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	dp := generateAWSDockerProfile(tenantID)
	resp, err := dbAPI.CreateDockerProfile(ctx, &dp, nil)
	require.NoError(t, err)
	t.Logf("create DockerProfiles successful, %s", resp)

	dp.ID = resp.(model.CreateDocumentResponse).ID
	return dp
}

func generateAWSDockerProfile(tenantID string) model.DockerProfile {
	dockerProfileName := "aws-registry-name-" + strings.ToLower(funk.RandomString(10)) + "a"
	dockerProfileDesc := "aws-registry-desc"
	return model.DockerProfile{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:        dockerProfileName,
		Type:        "ContainerRegistry",
		Server:      "a.b.c.d.e.f",
		Description: dockerProfileDesc,
		UserName:    "demo",
		Email:       "demo@nutanix.com",
		Pwd:         "foo",
		Credentials: "",
	}
}

func TestDockerProfiles(t *testing.T) {
	t.Parallel()
	t.Log("running TestDockerProfiles test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	edge2 := createEdge(t, dbAPI, tenantID)
	edgeID2 := edge2.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileId := dp.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, []string{edgeID})
	projectID := project.ID
	project2 := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, []string{edgeID2})
	projectID2 := project2.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteProject(ctx1, projectID2, nil)
		dbAPI.DeleteDockerProfile(ctx1, dockerProfileId, nil)
		dbAPI.DeleteCloudCreds(ctx1, cloudCredsID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID, nil)
		dbAPI.DeleteEdge(ctx1, edgeID2, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete DockerProfiles", func(t *testing.T) {
		t.Log("running Create/Get/Delete DockerProfiles test")

		dockerProfileDesc := "aws-registry-desc"
		dockerProfileNameUpdated := "aws-registry-name-updated"

		// get DockerProfiles
		dockerProfile, err := dbAPI.GetDockerProfile(ctx1, dockerProfileId)
		require.NoError(t, err)
		t.Logf("get DockerProfiles before update successful, %+v", dockerProfile)

		dockerProfiles, err := dbAPI.SelectDockerProfilesByIDs(ctx1, []string{dockerProfileId})
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatalf("expected dockerProfiles count to be 1, got %d", len(dockerProfiles))
		}
		t.Logf("got docker profiles by ids: %+v", dockerProfiles)

		// Needed as we need email in the auth context
		projRoles := []model.ProjectRole{
			{
				ProjectID: projectID,
				Role:      model.ProjectRoleAdmin,
			},
		}
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
				"projects":    projRoles,
				"email":       "any@email.com",
			},
		}
		//ctx1 = ctx1.Value()
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		dp.Name = dockerProfileNameUpdated
		dp.IFlagEncrypted = false
		dp.Credentials = ""
		upResp, err := dbAPI.UpdateDockerProfile(ctx, &dp, nil)
		require.NoError(t, err)
		t.Logf("update DockerProfiles successful, %+v", upResp)

		// test SelectAllDockerProfiles
		dockerProfiles, err = dbAPI.SelectAllDockerProfiles(ctx1)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("DockerProfiles count mismatch")
		}

		dockerProfiles, err = dbAPI.SelectAllDockerProfiles(ctx2)
		require.NoError(t, err)
		if len(dockerProfiles) != 0 {
			t.Fatal("Unexpected non-zero docker profiles count")
		}

		dockerProfiles, err = dbAPI.SelectAllDockerProfiles(ctx3)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("Unexpected docker profiles count")
		}

		// select all vs select all W
		var w bytes.Buffer
		dps1, err := dbAPI.SelectAllDockerProfiles(ctx1)
		require.NoError(t, err)
		dps2 := &[]model.DockerProfile{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllDockerProfilesW, dps2, &w)
		require.NoError(t, err)
		sort.Sort(model.DockerProfilesByID(dps1))
		sort.Sort(model.DockerProfilesByID(*dps2))
		if !reflect.DeepEqual(&dps1, dps2) {
			t.Fatalf("expect select docker profiles and select docker profiles w results to be equal %+v vs %+v", dps1, *dps2)
		}

		// test SelectAllDockerProfilesForProject
		authContext1 = &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
				"email":       "any@email.com",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		dockerProfiles, err = dbAPI.SelectAllDockerProfilesForProject(newCtx, projectID)
		// expect this to fail, since for project call require project membership, infra admin is not sufficient
		require.Error(t, err, "expect select all docker profiles to fail for auth 1")
		dockerProfiles, err = dbAPI.SelectAllDockerProfilesForProject(ctx2, projectID)
		require.Error(t, err, "expect select all docker profiles to fail for auth 2")

		dockerProfiles, err = dbAPI.SelectAllDockerProfilesForProject(ctx3, projectID)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("Unexpected docker profiles for project count")
		}

		// test GetDockerProfile
		dockerProfile, err = dbAPI.GetDockerProfile(ctx1, dockerProfileId)
		require.NoError(t, err)
		t.Logf("get DockerProfile successful, %+v", dockerProfile)

		if dockerProfile.ID != dockerProfileId || dockerProfile.Name != dockerProfileNameUpdated || dockerProfile.Description != dockerProfileDesc {
			t.Fatal("DockerProfile data mismatch")
		}
		dockerProfile, err = dbAPI.GetDockerProfile(ctx2, dockerProfileId)
		require.Error(t, err, "Expected not found error")
		dockerProfile, err = dbAPI.GetDockerProfile(ctx3, dockerProfileId)
		require.NoError(t, err, "Unexpected GetDockerProfile error")
		if dockerProfile.ID != dockerProfileId || dockerProfile.Name != dockerProfileNameUpdated || dockerProfile.Description != dockerProfileDesc {
			t.Fatal("DockerProfile 3 data mismatch")
		}

		edgeIDs, err := dbAPI.GetAllDockerProfileEdges(ctx1, dockerProfileId)
		require.NoError(t, err, "Unexpected GetAllDockerProfileEdges error")
		if len(edgeIDs) != 2 {
			t.Fatalf("expect docker profile edges count to be 2, got %d", len(edgeIDs))
		}
		sort.Strings(edgeIDs)
		edgeIDs2 := []string{edgeID, edgeID2}
		sort.Strings(edgeIDs2)
		if !reflect.DeepEqual(edgeIDs, edgeIDs2) {
			t.Fatal("edge ids mismatch")
		}

	})

	// select all DockerProfiles
	t.Run("SelectAllDockerProfiles", func(t *testing.T) {
		t.Log("running SelectAllDockerProfilestest")
		dockerProfiles, err := dbAPI.SelectAllDockerProfiles(ctx1)
		require.NoError(t, err)
		for _, dockerProfile := range dockerProfiles {
			testForMarshallability(t, dockerProfile)
		}
	})

	t.Run("DockerProfileConversion", func(t *testing.T) {
		t.Log("running DockerProfileConversion test")
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		dockerProfileList := []model.DockerProfile{
			{
				BaseModel: model.BaseModel{
					ID:        "aws-cloud-creds-id",
					TenantID:  tenantID,
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "foo",
				Type:        "AWS",
				Description: "bar",
				Credentials: "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
			},
		}
		for _, app := range dockerProfileList {
			appDBO := api.DockerProfileDBO{}
			app2 := model.DockerProfile{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateAWSDockerProfile(tenantID)
		doc.ID = id
		return dbAPI.CreateDockerProfile(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetDockerProfile(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteDockerProfile(ctx1, id, nil)
	}))
}

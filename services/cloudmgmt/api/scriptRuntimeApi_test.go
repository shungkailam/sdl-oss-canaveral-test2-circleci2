package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createScriptRuntime(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string, dockerProfileID string) model.ScriptRuntime {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects": []model.ProjectRole{
				{
					ProjectID: projectID,
					Role:      model.ProjectRoleAdmin,
				},
			},
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	doc := generateScriptRuntime(tenantID, dockerProfileID, projectID)
	// create script runtime
	resp, err := dbAPI.CreateScriptRuntime(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create ScriptRuntime successful, %s", resp)
	doc.ID = resp.(model.CreateDocumentResponse).ID
	sr, err := dbAPI.GetScriptRuntime(ctx, doc.ID)
	require.NoError(t, err)
	return sr
}

func generateScriptRuntime(tenantID string, dockerProfileID string, projectID string) model.ScriptRuntime {
	dockerfile := "docker file"
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	return model.ScriptRuntime{
		BaseModel: model.BaseModel{
			ID:        "",
			TenantID:  tenantID,
			Version:   5,
			CreatedAt: now,
			UpdatedAt: now,
		},
		ScriptRuntimeCore: model.ScriptRuntimeCore{
			Name:            "script-runtime-name-" + base.GetUUID(),
			Description:     "script runtime desc",
			Language:        "python",
			Builtin:         false,
			DockerRepoURI:   "docker-repo-uri",
			DockerProfileID: dockerProfileID,
			Dockerfile:      dockerfile,
		},
		ProjectID: projectID,
	}
}

func TestScriptRuntime(t *testing.T) {
	t.Parallel()
	t.Log("running TestScriptRuntime test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileID := dp.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, nil)
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})
	noProjCtx := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteDockerProfile(ctx1, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx1, cloudCredsID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete ScriptRuntime", func(t *testing.T) {
		t.Log("running Create/Get/Delete ScriptRuntime test")

		dockerfile := "docker file"
		dockerfileUpdated := "docker file updated"

		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		doc := model.ScriptRuntime{
			BaseModel: model.BaseModel{
				ID:        "",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         false,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfile,
			},
		}
		// create script runtime
		resp, err := dbAPI.CreateScriptRuntime(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Logf("create script runtime successful, %s", resp)

		scriptId := resp.(model.CreateDocumentResponse).ID

		// get script runtime which has no project association
		script, err := dbAPI.GetScriptRuntime(noProjCtx, scriptId)
		require.NoError(t, err)
		t.Logf("get script runtime successful, %+v", script)

		if script.ID != scriptId || script.Dockerfile != dockerfile {
			t.Fatal("script runtime data mismatch")
		}

		// update script runtime
		doc = model.ScriptRuntime{
			BaseModel: model.BaseModel{
				ID:        scriptId,
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         false,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfileUpdated,
			},
			ProjectID: projectID,
		}
		upResp, err := dbAPI.UpdateScriptRuntime(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Logf("update script runtime successful, %+v", upResp)

		// get script runtime
		script, err = dbAPI.GetScriptRuntime(ctx1, scriptId)
		require.NoError(t, err)
		t.Logf("get script runtime successful, %+v", script)

		if script.ID != scriptId || script.Dockerfile != dockerfileUpdated {
			t.Fatal("script runtime data mismatch")
		}

		// select all script runtimes
		scripts, err := dbAPI.SelectAllScriptRuntimes(ctx1, nil)
		require.NoError(t, err)
		if len(scripts) != 1 {
			t.Fatalf("expect auth 1 scripts count to be 1, got %d", len(scripts))
		}
		scripts, err = dbAPI.SelectAllScriptRuntimes(ctx2, nil)
		require.NoError(t, err)
		if len(scripts) != 0 {
			t.Fatalf("expect auth 2 scripts count to be 0, got %d", len(scripts))
		}
		scripts, err = dbAPI.SelectAllScriptRuntimes(ctx3, nil)
		require.NoError(t, err)
		if len(scripts) != 1 {
			t.Fatalf("expect auth 3 scripts count to be 1, got %d", len(scripts))
		}

		// select all vs select all W
		var w bytes.Buffer
		srts1, err := dbAPI.SelectAllScriptRuntimes(ctx1, nil)
		require.NoError(t, err)
		srts2 := &[]model.ScriptRuntime{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllScriptRuntimesW, srts2, &w)
		require.NoError(t, err)
		sort.Sort(model.ScriptRuntimesByID(srts1))
		sort.Sort(model.ScriptRuntimesByID(*srts2))
		if !reflect.DeepEqual(&srts1, srts2) {
			t.Fatalf("expect select script runtimes and select script runtimes w results to be equal %+v vs %+v", srts1, *srts2)
		}

		// select all script runtimes for project
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		scripts, err = dbAPI.SelectAllScriptRuntimesForProject(newCtx, projectID, nil)
		require.Error(t, err, "expect auth 1 select all script runtimes for project to fail")
		scripts, err = dbAPI.SelectAllScriptRuntimesForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect auth 2 select all script runtimes for project to fail")
		scripts, err = dbAPI.SelectAllScriptRuntimesForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(scripts) != 1 {
			t.Fatalf("expect auth 3 scripts for project count to be 1, got %d", len(scripts))
		}

		// delete script runtime
		delResp, err := dbAPI.DeleteScriptRuntime(ctx1, scriptId, nil)
		require.NoError(t, err)
		t.Logf("delete script runtime successful, %v", delResp)

		//
		doc2 := model.ScriptRuntime{
			BaseModel: model.BaseModel{
				// ID:        "script-runtime-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         true,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfile,
			},
		}
		// create builtin script runtime (should fail)
		resp, err = dbAPI.CreateScriptRuntime(ctx1, &doc2, nil)
		require.Errorf(t, err, "expected creation of builtin script runtime to fail")

		doc2 = model.ScriptRuntime{
			BaseModel: model.BaseModel{
				// ID:        "script-runtime-id",
				TenantID:  tenantID,
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			ScriptRuntimeCore: model.ScriptRuntimeCore{
				Name:            "script-runtime-name",
				Description:     "script runtime desc",
				Language:        "python",
				Builtin:         false,
				DockerRepoURI:   "docker-repo-uri",
				DockerProfileID: dockerProfileID,
				Dockerfile:      dockerfile,
			},
		}
		// create script runtime w/o project id - require infra admin
		resp, err = dbAPI.CreateScriptRuntime(ctx2, &doc2, nil)
		if err == nil {
			t.Fatal("expected create global script runtime by non infra admin to fail")
		}
		resp, err = dbAPI.CreateScriptRuntime(ctx1, &doc2, nil)
		require.NoError(t, err)

		t.Logf("create global script runtime successful, %s", resp)

		scriptId2 := resp.(model.CreateDocumentResponse).ID

		// delete script runtime
		delResp, err = dbAPI.DeleteScriptRuntime(ctx2, scriptId2, nil)
		require.Errorf(t, err, "expected delete global script runtime by non infra admin to fail")

		delResp, err = dbAPI.DeleteScriptRuntime(ctx1, scriptId2, nil)
		require.NoError(t, err)
		t.Logf("delete global script runtime successful, %v", delResp)
	})

	// select all script runtimes
	t.Run("SelectAllScriptRuntimes", func(t *testing.T) {
		t.Log("running SelectAllScriptRuntimes test")
		scriptRuntimes, err := dbAPI.SelectAllScriptRuntimes(ctx1, nil)
		require.NoError(t, err)
		for _, scriptRuntime := range scriptRuntimes {
			testForMarshallability(t, scriptRuntime)
		}
	})

	t.Run("ScriptRuntimeConversion", func(t *testing.T) {
		t.Log("running ScriptRuntimeConversion test")
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		scriptRuntimes := []model.ScriptRuntime{
			{
				BaseModel: model.BaseModel{
					ID:        "script-runtime-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				ScriptRuntimeCore: model.ScriptRuntimeCore{
					Name:            "script-runtime-name",
					Description:     "script runtime desc",
					Language:        "python",
					Builtin:         false,
					DockerRepoURI:   "docker-repo-uri",
					DockerProfileID: dockerProfileID,
					Dockerfile:      "docker file",
				},
				ProjectID: "proj-id",
			},
		}
		for _, app := range scriptRuntimes {
			appDBO := api.ScriptRuntimeDBO{}
			app2 := model.ScriptRuntime{}
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
		doc := generateScriptRuntime(tenantID, dockerProfileID, projectID)
		doc.ID = id
		return dbAPI.CreateScriptRuntime(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetScriptRuntime(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteScriptRuntime(ctx1, id, nil)
	}))
}

package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createScript(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string, scriptRuntimeID string) model.Script {
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

	doc := generateScript(tenantID, projectID, scriptRuntimeID)

	// create script
	resp, err := dbAPI.CreateScript(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create Script successful, %s", resp)
	scriptID := resp.(model.CreateDocumentResponse).ID
	script, err := dbAPI.GetScript(ctx, scriptID)
	require.NoError(t, err)
	return script
}

func generateScript(tenantID string, projectID string, scriptRuntimeID string) model.Script {
	scriptName := "script-name-" + base.GetUUID()
	scriptType := "Transformation"
	scriptLanguage := "Python"
	scriptEnvrionment := "python tensorflow"
	scriptCode := "def main: print"
	return model.Script{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		ScriptCore: model.ScriptCore{
			Name:        scriptName,
			Type:        scriptType,
			Language:    scriptLanguage,
			Environment: scriptEnvrionment,
			Code:        scriptCode,
			Builtin:     false,
			ProjectID:   projectID,
			RuntimeID:   scriptRuntimeID,
		},
		Params: []model.ScriptParam{},
	}
}

func TestScript(t *testing.T) {
	t.Parallel()
	t.Log("running TestScript test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileId := dp.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, nil)
	projectID := project.ID
	project2 := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, nil)
	projectID2 := project2.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID, projectID2})
	scriptRuntime := createScriptRuntime(t, dbAPI, tenantID, projectID, dockerProfileId)
	scriptRuntimeID := scriptRuntime.ID
	scriptRuntime2 := createScriptRuntime(t, dbAPI, tenantID, projectID2, dockerProfileId)
	scriptRuntimeID2 := scriptRuntime2.ID

	// Teardown
	defer func() {
		dbAPI.DeleteScriptRuntime(ctx1, scriptRuntimeID, nil)
		dbAPI.DeleteScriptRuntime(ctx1, scriptRuntimeID2, nil)
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteProject(ctx1, projectID2, nil)
		dbAPI.DeleteDockerProfile(ctx1, dockerProfileId, nil)
		dbAPI.DeleteCloudCreds(ctx1, cloudCredsID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Script", func(t *testing.T) {
		t.Log("running Create/Get/Delete Script test")

		scriptName := "script name"
		scriptType := "Transformation"
		scriptLanguage := "Python"
		scriptEnvrionment := "python tensorflow"
		scriptCode := "def main: print"
		scriptCodeUpdated := "def main: print 'hello'"

		// Script object, leave ID blank and let create generate it
		doc := model.Script{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  0,
			},
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCode,
				Builtin:     false,
				ProjectID:   projectID,
				RuntimeID:   scriptRuntimeID,
			},
			Params: []model.ScriptParam{},
		}
		// create script
		resp, err := dbAPI.CreateScript(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Logf("create script successful, %s", resp)

		scriptId := resp.(model.CreateDocumentResponse).ID

		// create script with ctx2 should fail (no project access)
		doc.Name = scriptName + "-2"
		_, err = dbAPI.CreateScript(ctx2, &doc, nil)
		require.Error(t, err, "create script 2 should fail")
		// create script with ctx3 should succeed
		resp3, err := dbAPI.CreateScript(ctx3, &doc, nil)
		require.NoError(t, err)
		_, err = dbAPI.DeleteScript(ctx3, resp3.(model.CreateDocumentResponse).ID, nil)
		require.NoError(t, err)
		// create script with runtime in different project should fail
		doc.RuntimeID = scriptRuntimeID2
		_, err = dbAPI.CreateScript(ctx1, &doc, nil)
		require.Error(t, err, "create script with runtime in different project should fail")
		doc.RuntimeID = "bad-runtime-id"
		_, err = dbAPI.CreateScript(ctx1, &doc, nil)
		require.Error(t, err, "create script with bad runtime id should fail")

		// restore doc
		doc.Name = scriptName
		doc.RuntimeID = scriptRuntimeID

		// select all scripts
		scripts, err := dbAPI.SelectAllScripts(ctx1, nil)
		require.NoError(t, err)
		if len(scripts) != 1 {
			t.Fatalf("expect auth 1 scripts count to be 1, got %d", len(scripts))
		}
		scripts, err = dbAPI.SelectAllScripts(ctx2, nil)
		require.NoError(t, err)
		if len(scripts) != 0 {
			t.Fatalf("expect auth 2 scripts count to be 0, got %d", len(scripts))
		}
		scripts, err = dbAPI.SelectAllScripts(ctx3, nil)
		require.NoError(t, err)
		if len(scripts) != 1 {
			t.Fatalf("expect auth 3 scripts count to be 1, got %d", len(scripts))
		}

		// select all vs select all W
		var w bytes.Buffer
		scripts1, err := dbAPI.SelectAllScripts(ctx1, nil)
		require.NoError(t, err)
		scripts2 := &[]model.Script{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllScriptsW, scripts2, &w)
		require.NoError(t, err)
		sort.Sort(model.ScriptsByID(scripts1))
		sort.Sort(model.ScriptsByID(*scripts2))
		if !reflect.DeepEqual(&scripts1, scripts2) {
			t.Fatalf("expect select scripts and select scripts w results to be equal %+v vs %+v", scripts1, *scripts2)
		}

		// select all scripts for project
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		scripts, err = dbAPI.SelectAllScriptsForProject(newCtx, projectID, nil)
		require.Error(t, err, "expect auth 1 select all scripts for project to fail")
		scripts, err = dbAPI.SelectAllScriptsForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect auth 2 select all scripts for project to fail")
		scripts, err = dbAPI.SelectAllScriptsForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(scripts) != 1 {
			t.Fatalf("expect auth 3 scripts for project count to be 1, got %d", len(scripts))
		}

		// update script
		doc = model.Script{
			BaseModel: model.BaseModel{
				ID:       scriptId,
				TenantID: tenantID,
				Version:  0,
			},
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCodeUpdated,
				Builtin:     false,
				ProjectID:   projectID,
				RuntimeID:   scriptRuntimeID,
			},
			Params: []model.ScriptParam{},
		}
		docW := model.ScriptForceUpdate{Doc: doc, ForceUpdate: false}
		upResp, err := dbAPI.UpdateScript(ctx1, &docW, nil)
		require.NoError(t, err)
		t.Logf("update script successful, %+v", upResp)

		// get script
		script, err := dbAPI.GetScript(ctx1, scriptId)
		require.NoError(t, err)
		t.Logf("get script successful, %+v", script)

		if script.ID != scriptId || script.Name != scriptName || script.Type != scriptType || scriptEnvrionment != script.Environment || script.Code != scriptCodeUpdated {
			t.Fatal("script data mismatch")
		}

		// update script runtime should fail for script in-use
		// update script runtime
		// first create a data stream to use the script...

		var size float64 = 1000000
		dataStreamName := "data-streams-name"
		dataStreamDataType := "Custom"

		// DataStream objects
		dstr := model.DataStream{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:             dataStreamName,
			DataType:         dataStreamDataType,
			Origin:           "DataSource",
			OriginSelectors:  []model.CategoryInfo{},
			OriginID:         "",
			Destination:      "Cloud",
			CloudType:        "AWS",
			CloudCredsID:     cloudCredsID,
			AWSCloudRegion:   "us-west-2",
			GCPCloudRegion:   "",
			EdgeStreamType:   "",
			AWSStreamType:    "Kafka",
			GCPStreamType:    "",
			Size:             size,
			EnableSampling:   false,
			SamplingInterval: 0,
			TransformationArgsList: []model.TransformationArgs{
				{
					TransformationID: scriptId,
					Args:             []model.ScriptParamValue{},
				},
			},
			DataRetention: []model.RetentionInfo{},
			ProjectID:     projectID,
		}

		// create DataStreams
		resp, err = dbAPI.CreateDataStream(ctx1, &dstr, nil)
		require.NoError(t, err)
		dataStreamId := resp.(model.CreateDocumentResponse).ID

		scriptRuntime.Name = "script-runtime-name-updated"
		_, err = dbAPI.UpdateScriptRuntime(ctx1, &scriptRuntime, nil)
		require.Error(t, err, "update in-use script runtime should fail")
		t.Logf("expected update script runtime error: %s", err.Error())

		_, err = dbAPI.DeleteScriptRuntime(ctx1, scriptRuntime.ID, nil)
		require.Error(t, err, "delete in-use script runtime should fail")
		t.Logf("expected delete script runtime error: %s", err.Error())

		_, err = dbAPI.DeleteDataStream(ctx1, dataStreamId, nil)
		require.NoError(t, err)

		_, err = dbAPI.UpdateScriptRuntime(ctx1, &scriptRuntime, nil)
		require.NoError(t, err)

		// delete script
		delResp, err := dbAPI.DeleteScript(ctx1, scriptId, nil)
		require.NoError(t, err)
		t.Logf("delete script successful, %v", delResp)

		doc = model.Script{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  0,
			},
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCode,
				Builtin:     true,
				ProjectID:   projectID,
				RuntimeID:   scriptRuntimeID,
			},
			Params: []model.ScriptParam{},
		}
		// create builtin script should fail
		resp, err = dbAPI.CreateScript(ctx1, &doc, nil)
		require.Errorf(t, err, "create builtin script should fail")

		doc = model.Script{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  0,
			},
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCode,
				Builtin:     false,
				ProjectID:   "",
				RuntimeID:   scriptRuntimeID,
			},
			Params: []model.ScriptParam{},
		}
		// create global script requires infra admin role
		resp, err = dbAPI.CreateScript(ctx2, &doc, nil)
		if err == nil {
			t.Fatal("create global script requires infra admin role")
		}
		resp, err = dbAPI.CreateScript(ctx1, &doc, nil)
		require.NoError(t, err)
		t.Logf("create global script successful, %s", resp)

		scriptId = resp.(model.CreateDocumentResponse).ID

		// update script
		doc = model.Script{
			BaseModel: model.BaseModel{
				ID:       scriptId,
				TenantID: tenantID,
				Version:  0,
			},
			ScriptCore: model.ScriptCore{
				Name:        scriptName,
				Type:        scriptType,
				Language:    scriptLanguage,
				Environment: scriptEnvrionment,
				Code:        scriptCodeUpdated,
				Builtin:     false,
				ProjectID:   "",
				RuntimeID:   scriptRuntimeID,
			},
			Params: []model.ScriptParam{},
		}
		docW = model.ScriptForceUpdate{Doc: doc, ForceUpdate: false}
		upResp, err = dbAPI.UpdateScript(ctx2, &docW, nil)
		require.Errorf(t, err, "update global script should require infra admin role")

		upResp, err = dbAPI.UpdateScript(ctx1, &docW, nil)
		require.NoError(t, err)

		t.Logf("update global script successful, %+v", upResp)

		// delete script
		delResp, err = dbAPI.DeleteScript(ctx2, scriptId, nil)
		require.Error(t, err, "delete global script should require infra admin role")

		delResp, err = dbAPI.DeleteScript(ctx1, scriptId, nil)
		require.NoError(t, err)
		t.Logf("delete global script successful, %v", delResp)

		// create script without runtime_id (backward compatibility)
		// scriptName := "script name"
		builtinRuntimeEnvironments := []string{"python-env", "python2-env", "tensorflow-python", "node-env", "golang-env"}
		nonBuiltinRuntimeEnvironments := []string{"python3-env", "python-env2", "tensorflow-python2", "node-env2", "golang-env-2"}
		for i, env := range builtinRuntimeEnvironments {
			doc = model.Script{
				BaseModel: model.BaseModel{
					TenantID: tenantID,
				},
				ScriptCore: model.ScriptCore{
					Name:        fmt.Sprintf("%s-%d", scriptName, i),
					Type:        scriptType,
					Language:    scriptLanguage,
					Environment: env,
					Code:        scriptCode,
					Builtin:     false,
					ProjectID:   projectID,
				},
				Params: []model.ScriptParam{},
			}
			resp, err = dbAPI.CreateScript(ctx1, &doc, nil)
			require.NoError(t, err)
			scriptId = resp.(model.CreateDocumentResponse).ID
			delResp, err = dbAPI.DeleteScript(ctx1, scriptId, nil)
			require.NoError(t, err)
		}
		for i, env := range nonBuiltinRuntimeEnvironments {
			doc = model.Script{
				BaseModel: model.BaseModel{
					TenantID: tenantID,
				},
				ScriptCore: model.ScriptCore{
					Name:        fmt.Sprintf("%s-%d", scriptName, i),
					Type:        scriptType,
					Language:    scriptLanguage,
					Environment: env,
					Code:        scriptCode,
					Builtin:     false,
					ProjectID:   projectID,
				},
				Params: []model.ScriptParam{},
			}
			resp, err = dbAPI.CreateScript(ctx1, &doc, nil)
			require.Error(t, err, "expect create script to fail")
		}
	})

	// select all scripts
	t.Run("SelectAllScripts", func(t *testing.T) {
		t.Log("running SelectAllScripts test")
		scripts, err := dbAPI.SelectAllScripts(ctx1, nil)
		require.NoError(t, err)
		for _, script := range scripts {
			testForMarshallability(t, script)
		}
	})

	// SelectScriptsByRuntimeID
	t.Run("SelectScriptsByRuntimeID", func(t *testing.T) {
		t.Log("running SelectScriptsByRuntimeID test")
		authContext := &base.AuthContext{
			TenantID: "tenant-id-kthomas",
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		scripts, err := dbAPI.SelectScriptsByRuntimeID(ctx, "tenant-id-kthomas_sr-python")
		require.NoError(t, err)
		t.Logf("Got script ids: %v", scripts)
	})

	t.Run("ScriptConversion", func(t *testing.T) {
		t.Log("running ScriptConversion test")
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		scripts := []model.Script{
			{
				BaseModel: model.BaseModel{
					ID:        "script-id",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				ScriptCore: model.ScriptCore{
					Name:        "script-name",
					Type:        "Transformation",
					Language:    "python",
					Environment: "python-env",
					Code:        " ",
					Description: "face recognition",
					ProjectID:   "proj-id",
					Builtin:     false,
				},
				Params: []model.ScriptParam{},
			},
		}
		for _, app := range scripts {
			appDBO := api.ScriptDBO{}
			app2 := model.Script{}
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
		doc := generateScript(tenantID, projectID, scriptRuntimeID)
		doc.ID = id
		return dbAPI.CreateScript(ctx1, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetScript(ctx1, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteScript(ctx1, id, nil)
	}))
}

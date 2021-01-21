package model_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// dupe api.ScriptRuntimeDBO here to avoid dependency
type scriptRuntimeDBO struct {
	model.BaseModelDBO
	model.ScriptRuntimeCore
	ProjectID *string `json:"projectId,omitempty" db:"project_id"`
}

// TestScriptRuntime will test ScriptRuntime struct
func TestScriptRuntime(t *testing.T) {
	now := timeNow(t)
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
				DockerProfileID: "docker-profile-id",
				Dockerfile:      "docker file",
			},
			ProjectID: "proj-id",
		},
	}
	scriptRuntimeStrings := []string{
		`{"id":"script-runtime-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"script-runtime-name","description":"script runtime desc","language":"python","builtin":false,"dockerRepoURI":"docker-repo-uri","dockerProfileID":"docker-profile-id","dockerfile":"docker file","projectId":"proj-id"}`,
	}

	var version float64 = 5
	scriptRuntimeMap := []map[string]interface{}{
		{
			"id":              "script-runtime-id",
			"version":         version,
			"tenantId":        "tenant-id",
			"name":            "script-runtime-name",
			"description":     "script runtime desc",
			"language":        "python",
			"builtin":         false,
			"dockerRepoURI":   "docker-repo-uri",
			"dockerProfileID": "docker-profile-id",
			"dockerfile":      "docker file",
			"projectId":       "proj-id",
			"createdAt":       NOW,
			"updatedAt":       NOW,
		},
	}

	scriptRuntimeDBO := scriptRuntimeDBO{}
	err := base.Convert(&scriptRuntimes[0], &scriptRuntimeDBO)
	require.NoError(t, err, "failed to convert script runtime to dbo")
	t.Logf("convert script runtime to dbo returns: %+v", scriptRuntimeDBO)

	for i, script := range scriptRuntimes {
		scriptData, err := json.Marshal(script)
		require.NoError(t, err, "failed to marshal script")
		if scriptRuntimeStrings[i] != string(scriptData) {
			t.Fatal("script json string mismatch", string(scriptData))
		}

		var doc interface{}
		doc = script
		_, ok := doc.(model.ProjectScopedEntity)
		if !ok {
			t.Fatal("script runtime should be a project scoped entity")
		}

		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(scriptData, &m)
		require.NoError(t, err, "failed to unmarshal script to map")
		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &scriptRuntimeMap[i]) {
			t.Fatalf("script map marshal mismatch\n")
		}
	}

}

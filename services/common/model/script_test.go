package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestScript will test Script struct
func TestScript(t *testing.T) {
	now := timeNow(t)
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
	scriptStrings := []string{
		`{"id":"script-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"script-name","description":"face recognition","type":"Transformation","language":"python","environment":"python-env","code":" ","builtin":false,"projectId":"proj-id","params":[]}`,
	}

	var version float64 = 5
	scriptMaps := []map[string]interface{}{
		{
			"id":          "script-id",
			"version":     version,
			"tenantId":    "tenant-id",
			"name":        "script-name",
			"type":        "Transformation",
			"language":    "python",
			"environment": "python-env",
			"code":        " ",
			"projectId":   "proj-id",
			"builtin":     false,
			"params":      []model.ScriptParam{},
			"createdAt":   NOW,
			"updatedAt":   NOW,
			"description": "face recognition",
		},
	}

	for i, script := range scripts {
		scriptData, err := json.Marshal(script)
		require.NoError(t, err, "failed to marshal script")
		if scriptStrings[i] != string(scriptData) {
			t.Fatal("script json string mismatch", string(scriptData))
		}

		var doc interface{}
		doc = script
		_, ok := doc.(model.ProjectScopedEntity)
		if !ok {
			t.Fatal("script should be a project scoped entity")
		}

		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(scriptData, &m)
		require.NoError(t, err, "failed to unmarshal script to map")
		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &scriptMaps[i]) {
			t.Fatalf("script map marshal mismatch\n")
		}
	}

}

func TestScriptsDifferOnlyByNameAndDesc(t *testing.T) {
	var sdTests = []struct {
		s1         *model.Script
		s2         *model.Script
		differOnly bool
	}{
		{&model.Script{}, &model.Script{}, true},
		{&model.Script{ScriptCore: model.ScriptCore{Name: "name-1"}}, &model.Script{ScriptCore: model.ScriptCore{Name: "name-2"}}, true},
		{&model.Script{ScriptCore: model.ScriptCore{Description: "desc-1"}}, &model.Script{ScriptCore: model.ScriptCore{Description: "desc-2"}}, true},
		{&model.Script{ScriptCore: model.ScriptCore{Type: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{Type: "t-2"}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{Language: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{Language: "t-2"}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{Environment: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{Environment: "t-2"}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{Code: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{Code: "t-2"}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{RuntimeID: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{RuntimeID: "t-2"}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{RuntimeTag: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{RuntimeTag: "t-2"}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{Builtin: true}}, &model.Script{ScriptCore: model.ScriptCore{Builtin: false}}, false},
		{&model.Script{ScriptCore: model.ScriptCore{ProjectID: "t-1"}}, &model.Script{ScriptCore: model.ScriptCore{ProjectID: "t-2"}}, false},
		{&model.Script{BaseModel: model.BaseModel{TenantID: "t-1"}}, &model.Script{BaseModel: model.BaseModel{TenantID: "t-2"}}, false},
		{&model.Script{BaseModel: model.BaseModel{ID: "t-1"}}, &model.Script{BaseModel: model.BaseModel{ID: "t-2"}}, false},
		{&model.Script{Params: []model.ScriptParam{}}, &model.Script{}, true},
		{&model.Script{Params: []model.ScriptParam{}}, &model.Script{Params: []model.ScriptParam{}}, true},
		{&model.Script{Params: []model.ScriptParam{
			{
				Name: "n",
				Type: "t",
			},
		}}, &model.Script{Params: []model.ScriptParam{
			{
				Name: "n",
				Type: "t",
			},
		}}, true},
		{&model.Script{Params: []model.ScriptParam{
			{
				Name: "n",
				Type: "t",
			},
			{
				Name: "n2",
				Type: "t2",
			},
		}}, &model.Script{Params: []model.ScriptParam{
			{
				Name: "n",
				Type: "t",
			},
			{
				Name: "n2",
				Type: "t2",
			},
		}}, true},
		{&model.Script{Params: []model.ScriptParam{
			{
				Name: "n",
				Type: "t",
			},
			{
				Name: "n2",
				Type: "t2",
			},
		}}, &model.Script{Params: []model.ScriptParam{
			{
				Name: "n2",
				Type: "t2",
			},
			{
				Name: "n",
				Type: "t",
			},
		}}, false},
	}
	for _, sd := range sdTests {
		if model.ScriptsDifferOnlyByNameAndDesc(sd.s1, sd.s2) != sd.differOnly {
			t.Fatalf("%+v vs %+v expected differOnly = %t", *sd.s1, *sd.s2, sd.differOnly)
		}
	}
}

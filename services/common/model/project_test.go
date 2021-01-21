package model_test

import (
	"cloudservices/common/model"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
)

// TestProject will test Project struct
func TestProject(t *testing.T) {
	now := timeNow(t)
	projects := []model.Project{
		{
			BaseModel: model.BaseModel{
				ID:        "proj-id",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:               "proj-name",
			Description:        "proj-desc",
			CloudCredentialIDs: []string{},
			DockerProfileIDs:   []string{},
			Users: []model.ProjectUserInfo{
				{
					UserID: "user-id",
					Role:   model.ProjectRoleAdmin,
				},
			},
			EdgeSelectorType: model.ProjectEdgeSelectorTypeCategory,
			EdgeIDs:          nil,
			EdgeSelectors: []model.CategoryInfo{
				{
					ID:    "cat-id",
					Value: "cat-val",
				},
			},
		},
		{
			BaseModel: model.BaseModel{
				ID:        "proj-id",
				TenantID:  "tenant-id",
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Name:               "proj-name",
			Description:        "proj-desc",
			CloudCredentialIDs: []string{},
			DockerProfileIDs:   []string{},
			Users:              []model.ProjectUserInfo{},
			EdgeSelectorType:   model.ProjectEdgeSelectorTypeExplicit,
			EdgeIDs: []string{
				"ORD",
				"SFO",
			},
			EdgeSelectors: nil,
		},
	}
	projectStrings := []string{
		`{"id":"proj-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"proj-name","description":"proj-desc","cloudCredentialIds":[],"dockerProfileIds":[],"users":[{"userId":"user-id","role":"PROJECT_ADMIN"}],"edgeSelectorType":"Category","edgeIds":null,"edgeSelectors":[{"id":"cat-id","value":"cat-val"}],"privileged":null}`,
		`{"id":"proj-id","version":5,"tenantId":"tenant-id","createdAt":"2018-01-01T01:01:01Z","updatedAt":"2018-01-01T01:01:01Z","name":"proj-name","description":"proj-desc","cloudCredentialIds":[],"dockerProfileIds":[],"users":[],"edgeSelectorType":"Explicit","edgeIds":["ORD","SFO"],"edgeSelectors":null,"privileged":null}`,
	}

	var version float64 = 5
	projectMaps := []map[string]interface{}{
		{
			"id":                 "proj-id",
			"version":            version,
			"tenantId":           "tenant-id",
			"name":               "proj-name",
			"description":        "proj-desc",
			"cloudCredentialIds": []string{},
			"dockerProfileIds":   []string{},
			"users": []map[string]interface{}{
				{
					"userId": "user-id",
					"role":   model.ProjectRoleAdmin,
				},
			},
			"edgeSelectorType": model.ProjectEdgeSelectorTypeCategory,
			"edgeIds":          nil,
			"edgeSelectors": []map[string]interface{}{
				{
					"id":    "cat-id",
					"value": "cat-val",
				},
			},
			"privileged": nil,
			"createdAt":  NOW,
			"updatedAt":  NOW,
		},
		{
			"id":                 "proj-id",
			"version":            version,
			"tenantId":           "tenant-id",
			"name":               "proj-name",
			"description":        "proj-desc",
			"cloudCredentialIds": []string{},
			"dockerProfileIds":   []string{},
			"users":              []map[string]interface{}{},
			"edgeSelectorType":   model.ProjectEdgeSelectorTypeExplicit,
			"edgeIds": []string{
				"ORD",
				"SFO",
			},
			"edgeSelectors": nil,
			"privileged":    nil,
			"createdAt":     NOW,
			"updatedAt":     NOW,
		},
	}

	for i, project := range projects {
		projectData, err := json.Marshal(project)
		require.NoError(t, err, "failed to marshal project")
		if projectStrings[i] != string(projectData) {
			t.Fatal("project json string mismatch", string(projectData))
		}
		// alternative form: m := make(map[string]interface{})
		m := map[string]interface{}{}
		err = json.Unmarshal(projectData, &m)
		require.NoError(t, err, "failed to unmarshal project to map")
		// reflect.DeepEqual fails on equivalent slices here,
		// so use weaker marshal equal
		if !model.MarshalEqual(&m, &projectMaps[i]) {
			t.Fatalf("project map marshal mismatch\n")
		}
	}

}

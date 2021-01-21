package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createProjectService(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string) model.ProjectService {
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

	sdoc := generateProjectService(tenantID, projectID)

	// create project service
	resp, err := dbAPI.CreateProjectService(ctx, &sdoc, nil)
	require.NoError(t, err)
	t.Logf("create project service successfully, %s", resp)
	id := resp.(model.CreateDocumentResponseV2).ID
	projectService, err := dbAPI.GetProjectService(ctx, id)
	require.NoError(t, err)
	return projectService
}

func generateProjectService(tenantID string, projectID string) model.ProjectService {
	return model.ProjectService{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		ProjectID: projectID,
		Name:      fmt.Sprintf("Kafka-%s", base.GetUUID()),
		ServiceManifest: `
sherlock.nutanix.com/v1
  kind: Kafka
  metadata:
    name: kafka-test
    namespace: single-node-kafka
  spec:
    apiEndpoint: kafka-test:9092
    nodePort: null
`,
	}
}

func TestProjectService(t *testing.T) {
	t.Parallel()
	t.Log("running TestProjectService test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, nil)
	projectID := project.ID

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
	if pss, err := dbAPI.SelectAllProjectServices(ctx, nil); err != nil {
		t.Fatal(err)
	} else if len(pss) != 0 {
		t.Fatal("Expected 0 project service")
	}

	ps := createProjectService(t, dbAPI, tenantID, projectID)

	// Make sure project servcie goes away
	defer func() {
		dbAPI.DeleteProjectService(ctx, ps.ID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	if pss, err := dbAPI.SelectAllProjectServices(ctx, nil); err != nil {
		t.Fatal(err)
	} else if len(pss) != 1 {
		t.Fatal("Expected 1 project service")
	}

	if _, err := dbAPI.DeleteProjectService(ctx, ps.ID, nil); err != nil {
		t.Fatal(err)
	}

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateProjectService(tenantID, projectID)
		doc.ID = id
		return dbAPI.CreateProjectService(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetProjectService(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteProjectService(ctx, id, nil)
	}))
}

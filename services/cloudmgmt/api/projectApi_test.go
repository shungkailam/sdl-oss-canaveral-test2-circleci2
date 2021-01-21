package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"reflect"
	"sort"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	"github.com/thoas/go-funk"
)

func uid2ProjectUserInfo(userID string) model.ProjectUserInfo {
	return model.ProjectUserInfo{
		UserID: userID,
		Role:   model.ProjectRoleAdmin,
	}
}
func uids2ProjectUserInfoList(userIDs []string) []model.ProjectUserInfo {
	return funk.Map(userIDs, uid2ProjectUserInfo).([]model.ProjectUserInfo)
}

func createCategoryProjectCommon(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeSelectors []model.CategoryInfo, fns ...func(*model.Project)) model.Project {
	project, err := createCategoryProjectCommon2(t, dbAPI, tenantID, cloudCredsIDs, dockerProfileIDs, userIDs, edgeSelectors, fns...)
	require.NoError(t, err)
	return project
}

func createCategoryProjectCommon2(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeSelectors []model.CategoryInfo, fns ...func(*model.Project)) (model.Project, error) {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	project := generateProject(tenantID, cloudCredsIDs, dockerProfileIDs, userIDs, edgeSelectors)
	for _, fn := range fns {
		fn(&project)
	}
	// create project
	resp, err := dbAPI.CreateProject(ctx, &project, nil)
	if err != nil {
		return model.Project{}, err
	}
	t.Logf("create project successful, %s", resp)

	project.ID = resp.(model.CreateDocumentResponse).ID
	proj, err := dbAPI.GetProject(ctx, project.ID)
	if err != nil {
		return model.Project{}, err
	}
	return proj, nil
}

func generateProject(tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeSelectors []model.CategoryInfo) model.Project {
	return model.Project{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:               fmt.Sprintf("Where is Waldo-%s", base.GetUUID()),
		Description:        "Find Waldo",
		CloudCredentialIDs: cloudCredsIDs,
		DockerProfileIDs:   dockerProfileIDs,
		Users:              uids2ProjectUserInfoList(userIDs),
		EdgeSelectorType:   model.ProjectEdgeSelectorTypeCategory,
		EdgeIDs:            nil,
		EdgeSelectors:      edgeSelectors,
	}
}

func createExplicitProjectCommon(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeIDs []string, fns ...func(*model.Project)) model.Project {
	project, err := createExplicitProjectCommon2(t, dbAPI, tenantID, cloudCredsIDs, dockerProfileIDs, userIDs, edgeIDs, fns...)
	require.NoError(t, err)
	return project
}
func createExplicitProjectCommon2(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeIDs []string, fns ...func(*model.Project)) (model.Project, error) {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	// Project object, leave ID blank and let create generate it
	project := model.Project{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:               fmt.Sprintf("Where is Waldo-%s", base.GetUUID()),
		Description:        "Find Waldo",
		CloudCredentialIDs: cloudCredsIDs,
		DockerProfileIDs:   dockerProfileIDs,
		Users:              uids2ProjectUserInfoList(userIDs),
		EdgeSelectorType:   model.ProjectEdgeSelectorTypeExplicit,
		EdgeIDs:            edgeIDs,
		EdgeSelectors:      nil,
	}
	for _, fn := range fns {
		fn(&project)
	}
	// create project
	resp, err := dbAPI.CreateProject(ctx, &project, nil)
	if err != nil {
		return model.Project{}, err
	}
	t.Logf("create project successful, %s", resp)

	project.ID = resp.(model.CreateDocumentResponse).ID
	proj, err := dbAPI.GetProject(ctx, project.ID)
	if err != nil {
		return model.Project{}, err
	}
	return proj, nil
}

func createEmptyCategoryProject(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.Project {
	return createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, nil)
}

func TestProject(t *testing.T) {
	t.Parallel()
	t.Log("running TestProject test")

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)

	tenantID := base.GetUUID()
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)

	// Create tenant object
	doc := model.Tenant{
		ID:      tenantID,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any@email.com",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	// Test default project ID generation
	defaultProjectID := api.GetDefaultProjectID(tenantID)
	t.Logf("default project ID for tenant ID %s is %s", tenantID, defaultProjectID)

	// create tenant
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create tenant successful, %s", resp)

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID

	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeID := edge.ID
	edge2 := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue2,
		},
	})
	edgeID2 := edge2.ID

	userName := "John Doe"
	userEmail := fmt.Sprintf("jd-%s@nutanix.com", base.GetUUID())
	userPassword := "P@ssw0rd"

	// User object, leave ID blank and let create generate it
	user := model.User{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  0,
		},
		Name:     userName,
		Email:    userEmail,
		Password: userPassword,
		Role:     "INFRA_ADMIN",
	}
	// create user
	resp, err = dbAPI.CreateUser(ctx, &user, nil)
	require.NoError(t, err)
	t.Logf("create user successful, %s", resp)

	userID := resp.(model.CreateDocumentResponse).ID

	// create cloud creds
	cloudCredsName := "aws-cloud-creds-name"
	cloudCredsDesc := "aws-cloud-creds-desc"

	// CloudCreds object, leave ID blank and let create generate it
	cc := model.CloudCreds{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:        cloudCredsName,
		Type:        "AWS",
		Description: cloudCredsDesc,
		AWSCredential: &model.AWSCredential{
			AccessKey: "foo",
			Secret:    "bar",
		},
		GCPCredential: nil,
	}
	resp, err = dbAPI.CreateCloudCreds(ctx, &cc, nil)
	require.NoError(t, err)
	t.Logf("create CloudCreds successful, %s", resp)
	cloudCredsID := resp.(model.CreateDocumentResponse).ID

	// create docker profile
	dockerProfileName := "aws-registry-name"
	dockerProfileDesc := "aws-registry-desc"

	// DockerProfile object, leave ID blank and let create generate it
	dp := model.DockerProfile{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:         dockerProfileName,
		Type:         "AWS",
		Server:       "a.b.c.d.e.f",
		CloudCredsID: cloudCredsID,
		Description:  dockerProfileDesc,
		Credentials:  "{\"AccessKeyId\":\"AWS-Access\",\"SecretAccessKey\":\"AWS-SecretAccessKey\",\"Account\":\"AWS-test\",\"Region\":\"us-west-2\",\"Server\":\"aws-server\",\"User\":\"aws-user\",\"Pwd\":\"aws-pwd\",\"Email\":\"aws-email\"}",
	}
	// create dockerProfiles
	resp, err = dbAPI.CreateDockerProfile(ctx, &dp, nil)
	require.NoError(t, err)
	t.Logf("create DockerProfiles successful, %s", resp)

	dockerProfileID := resp.(model.CreateDocumentResponse).ID

	// Teardown
	defer func() {
		dbAPI.DeleteDockerProfile(ctx, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx, cloudCredsID, nil)
		dbAPI.DeleteUser(ctx, userID, nil)
		dbAPI.DeleteEdge(ctx, edgeID2, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteCategory(ctx, categoryID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete Project", func(t *testing.T) {
		t.Log("running Create/Get/Delete Project test")

		projName := fmt.Sprintf("Where is Waldo-%s", base.GetUUID())
		projDesc := "Find Waldo"

		// Project object, leave ID blank and let create generate it
		doc := model.Project{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:               projName,
			Description:        projDesc,
			CloudCredentialIDs: []string{
				// cloudCredsID,
			},
			DockerProfileIDs: []string{
				dockerProfileID,
			},
			Users: []model.ProjectUserInfo{
				{
					UserID: userID,
					Role:   model.ProjectRoleAdmin,
				},
			},
			EdgeSelectorType: model.ProjectEdgeSelectorTypeCategory,
			EdgeIDs:          nil,
			EdgeSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue2,
				},
			},
		}
		// create project
		resp, err := dbAPI.CreateProject(ctx, &doc, func(ctx context.Context, doc interface{}) error {
			t.Logf("create project, go doc from callback: %+v", doc)
			pj := doc.(model.Project)
			if len(pj.EdgeIDs) != 1 {
				t.Fatal("expected project in callback to contain one edge")
			}
			return nil
		})
		require.NoError(t, err)
		t.Logf("create project successful, %s", resp)

		projectId := resp.(model.CreateDocumentResponse).ID

		// get project
		project, err := dbAPI.GetProject(ctx, projectId)
		require.NoError(t, err)
		t.Logf("get project successful, %+v", project)

		// project should contain 1 docker profile
		if len(project.DockerProfileIDs) != 1 {
			t.Fatal("expected project to contain one docker profile")
		}

		// project should contain 1 cloud profile (auto added the one that backs the docker profile)
		if len(project.CloudCredentialIDs) != 1 {
			t.Fatal("expected project to contain one cloud profile")
		}

		// project should contain one edge
		if len(project.EdgeIDs) != 1 {
			t.Fatal("expected project to contain one edge")
		}
		if project.EdgeIDs[0] != edgeID2 {
			t.Fatal("expected project to contain one edge with given id")
		}
		if len(project.EdgeSelectors) != 1 {
			t.Fatal("expected project to contain one edge selector")
		}
		if project.EdgeSelectors[0].ID != categoryID || project.EdgeSelectors[0].Value != TestCategoryValue2 {
			t.Fatal("expected project to contain one edge selector with given id and value")
		}

		// try create project with empty docker profiles etc
		doc2 := model.Project{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:               fmt.Sprintf("Where is Waldo-%s", base.GetUUID()),
			Description:        projDesc,
			CloudCredentialIDs: []string{},
			DockerProfileIDs:   []string{},
			Users:              []model.ProjectUserInfo{},
			EdgeSelectorType:   model.ProjectEdgeSelectorTypeCategory,
			EdgeIDs:            nil,
			EdgeSelectors:      []model.CategoryInfo{},
		}
		resp, err = dbAPI.CreateProject(ctx, &doc2, nil)
		require.NoError(t, err)
		t.Logf("create project 2 successful, %s", resp)

		projectId2 := resp.(model.CreateDocumentResponse).ID
		// get project
		project2, err := dbAPI.GetProject(ctx, projectId2)
		require.NoError(t, err)
		t.Logf("get project2 successful, %+v", project2)

		// project 2 should contain 0 edge
		if len(project2.EdgeIDs) != 0 {
			t.Fatal("expected project 2 to contain 0 edge")
		}
		if len(project2.EdgeSelectors) != 0 {
			t.Fatal("expected project 2 to contain 0 edge selector")
		}

		// update project
		doc = model.Project{
			BaseModel: model.BaseModel{
				ID:       projectId,
				TenantID: tenantID,
				Version:  5,
			},
			Name:               projName,
			Description:        projDesc,
			CloudCredentialIDs: []string{},
			DockerProfileIDs: []string{
				dockerProfileID,
			},
			Users: []model.ProjectUserInfo{
				{
					UserID: userID,
					Role:   model.ProjectRoleAdmin,
				},
			},
			EdgeSelectorType: model.ProjectEdgeSelectorTypeExplicit,
			EdgeIDs: []string{
				edgeID,
				edgeID,
				edgeID,
			},
			EdgeSelectors: []model.CategoryInfo{
				{
					ID:    categoryID,
					Value: TestCategoryValue2,
				},
			},
		}
		upResp, err := dbAPI.UpdateProject(ctx, &doc, func(ctx context.Context, doc interface{}) error {
			t.Logf("update project, go doc from callback: %+v", doc)
			pj := doc.(model.Project)
			if len(pj.EdgeIDs) != 1 {
				t.Fatal("expected project in update callback to contain one edge")
			}
			return nil
		})
		require.NoError(t, err)
		t.Logf("update project successful, %+v", upResp)

		// get project
		project, err = dbAPI.GetProject(ctx, projectId)
		require.NoError(t, err)
		t.Logf("get project successful, %+v", project)

		// project should contain 1 docker profile
		if len(project.DockerProfileIDs) != 1 {
			t.Fatal("expected project to contain one docker profile")
		}

		// project should contain 1 cloud profile (auto added the one that backs the docker profile)
		if len(project.CloudCredentialIDs) != 1 {
			t.Fatal("expected project to contain one cloud profile")
		}

		// project should contain one edge
		if len(project.EdgeIDs) != 1 {
			t.Fatal("expected project to contain one edge")
		}
		if project.EdgeIDs[0] != edgeID {
			t.Fatal("expected project to contain one edge with given id")
		}
		if len(project.EdgeSelectors) != 0 {
			t.Fatal("expected project to contain 0 edge selector")
		}

		if project.ID != projectId || project.Name != projName || project.Description != projDesc || project.EdgeSelectorType != model.ProjectEdgeSelectorTypeExplicit {
			t.Fatal("project data mismatch")
		}

		// select all vs select all W
		var w bytes.Buffer
		pjs1, err := dbAPI.SelectAllProjects(ctx, nil)
		require.NoError(t, err)
		pjs2 := &[]model.Project{}
		err = selectAllConverter(ctx, dbAPI.SelectAllProjectsW, pjs2, &w)
		require.NoError(t, err)
		sort.Sort(model.ProjectsByID(pjs1))
		sort.Sort(model.ProjectsByID(*pjs2))
		if !reflect.DeepEqual(&pjs1, pjs2) {
			t.Fatalf("expect select projects and select projects w results to be equal %+v vs %+v", pjs1, *pjs2)
		}

		pes, err := dbAPI.GetProjectEdges(ctx, api.ProjectEdgeDBO{
			ProjectID: projectId,
		})
		require.NoError(t, err)
		if len(pes) != 1 {
			t.Fatalf("expect project %s edges count to be 1, got %d", projectId, len(pes))
		}
		pes2, err := dbAPI.GetProjectsEdges(ctx, []string{projectId})
		require.NoError(t, err)
		if len(pes2) != 1 {
			t.Fatalf("expect projects %s edges count to be 1, got %d", projectId, len(pes))
		}
		if pes[0] != pes2[0] {
			t.Fatalf("expect project and projects %s edges to match, but %s != %s", projectId, pes[0], pes2[0])
		}

		// delete project
		delResp, err := dbAPI.DeleteProject(ctx, projectId, nil)
		require.NoError(t, err)
		t.Logf("delete project successful, %v", delResp)

	})

	// select all projects
	t.Run("SelectAllProjects", func(t *testing.T) {
		t.Log("running SelectAllProjects test")
		projects, err := dbAPI.SelectAllProjects(ctx, nil)
		require.NoError(t, err)
		for _, project := range projects {
			testForMarshallability(t, project)
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := generateProject(tenantID, []string{}, []string{}, []string{}, nil)
		doc.ID = id
		return dbAPI.CreateProject(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetProject(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteProject(ctx, id, nil)
	}))
}

func TestProjectCreateValidation(t *testing.T) {
	t.Parallel()
	t.Log("running TestProjectCreateValidation test")

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)

	// create 2 tenants
	// for each tenant, create 1 edge, 1 user, 1 cloud profile,
	// 1 container registry profile

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSDockerProfile(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileID := dp.ID
	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID
	user := createUser(t, dbAPI, tenantID)
	userID := user.ID

	doc2 := createTenant(t, dbAPI, "test tenant 2")
	tenantID2 := doc2.ID
	cc2 := createCloudCreds(t, dbAPI, tenantID2)
	cloudCredsID2 := cc2.ID
	dp2 := createAWSDockerProfile(t, dbAPI, tenantID2, cloudCredsID2)
	dockerProfileID2 := dp2.ID
	edge2 := createEdge(t, dbAPI, tenantID2)
	edgeID2 := edge2.ID
	user2 := createUser(t, dbAPI, tenantID2)
	userID2 := user2.ID

	ctx := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any@email.com",
		},
	})
	ctx2 := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
		TenantID: tenantID2,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any2@email.com",
		},
	})

	// Teardown
	defer func() {
		dbAPI.DeleteDockerProfile(ctx2, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx2, cloudCredsID2, nil)
		dbAPI.DeleteUser(ctx2, userID2, nil)
		dbAPI.DeleteEdge(ctx2, edgeID2, nil)
		dbAPI.DeleteTenant(ctx2, tenantID2, nil)
		dbAPI.DeleteDockerProfile(ctx, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx, cloudCredsID, nil)
		dbAPI.DeleteUser(ctx, userID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create Project Validation", func(t *testing.T) {
		t.Log("running Create Project Validation tests")

		project := createExplicitProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID, cloudCredsID}, []string{dockerProfileID, dockerProfileID}, []string{userID, userID}, []string{edgeID, edgeID})
		_, err := dbAPI.DeleteProject(ctx, project.ID, nil)
		require.NoError(t, err)

		sssa := [][4][]string{
			{
				{cloudCredsID2},
				nil,
				nil,
				nil,
			},
			{
				{base.GetUUID()},
				nil,
				nil,
				nil,
			},
			{
				nil,
				{dockerProfileID2},
				nil,
				nil,
			},
			{
				nil,
				{base.GetUUID()},
				nil,
				nil,
			},
			{
				nil,
				nil,
				{userID2},
				nil,
			},
			{
				nil,
				nil,
				{base.GetUUID()},
				nil,
			},
			{
				nil,
				nil,
				nil,
				{edgeID2},
			},
			{
				nil,
				nil,
				nil,
				{base.GetUUID()},
			},
		}
		for _, ssa := range sssa {
			_, err := createExplicitProjectCommon2(t, dbAPI, tenantID, ssa[0], ssa[1], ssa[2], ssa[3])
			require.Errorf(t, err, "expect create project to fail with %+v", ssa)
		}

	})

}

func TestGetProjectName(t *testing.T) {
	t.Parallel()
	t.Log("running TestProjectCreateValidation test")

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	// project is cat/v1
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{})
	projectID := project.ID

	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	projectName, err := dbAPI.GetProjectName(ctx1, projectID)
	require.NoError(t, err)

	pnf := api.GetProjectNameFn(ctx1, dbAPI)
	pn2 := pnf(projectID)
	if projectName != pn2 {
		t.Fatalf("%s != %s", projectName, pn2)
	}

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()
}

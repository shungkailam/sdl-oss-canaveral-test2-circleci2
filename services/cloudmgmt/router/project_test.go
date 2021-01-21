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
	"github.com/thoas/go-funk"
)

const (
	PROJECTS_PATH     = "/v1/projects"
	PROJECTS_PATH_NEW = "/v1.0/projects"
)

func uid2ProjectUserInfo(userID string) model.ProjectUserInfo {
	return model.ProjectUserInfo{
		UserID: userID,
		Role:   model.ProjectRoleAdmin,
	}
}
func uids2ProjectUserInfoList(userIDs []string) []model.ProjectUserInfo {
	if userIDs == nil {
		return []model.ProjectUserInfo{}
	}
	return funk.Map(userIDs, uid2ProjectUserInfo).([]model.ProjectUserInfo)
}

func makeExplicitProject(tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeIDs []string) model.Project {
	project := model.Project{
		Name:               fmt.Sprintf("Where is Waldo-%s", base.GetUUID()),
		Description:        "Find Waldo",
		CloudCredentialIDs: api.NilToEmptyStrings(cloudCredsIDs),
		DockerProfileIDs:   api.NilToEmptyStrings(dockerProfileIDs),
		Users:              uids2ProjectUserInfoList(userIDs),
		EdgeSelectorType:   model.ProjectEdgeSelectorTypeExplicit,
		EdgeIDs:            api.NilToEmptyStrings(edgeIDs),
		EdgeSelectors:      nil,
	}
	return project
}
func makeCategoryProject(tenantID string, cloudCredsIDs []string, dockerProfileIDs []string, userIDs []string, edgeSelectors []model.CategoryInfo) model.Project {
	project := model.Project{
		Name:               fmt.Sprintf("Where is Waldo-%s", base.GetUUID()),
		Description:        "Find Waldo",
		CloudCredentialIDs: cloudCredsIDs,
		DockerProfileIDs:   dockerProfileIDs,
		Users:              uids2ProjectUserInfoList(userIDs),
		EdgeSelectorType:   model.ProjectEdgeSelectorTypeCategory,
		EdgeIDs:            nil,
		EdgeSelectors:      edgeSelectors,
	}
	return project
}

// create project
func createProject(netClient *http.Client, project *model.Project, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, PROJECTS_PATH, *project, token)
	if err == nil {
		project.ID = resp.ID
	}
	return resp, reqID, err
}

// update project
func updateProject(netClient *http.Client, projectID string, project model.Project, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", PROJECTS_PATH, projectID), project, token)
}

// get projects
func getProjects(netClient *http.Client, token string) ([]model.Project, error) {
	projects := []model.Project{}
	err := doGet(netClient, PROJECTS_PATH, token, &projects)
	return projects, err
}
func getProjectsNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.ProjectListPayload, error) {
	response := model.ProjectListPayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", PROJECTS_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

// delete project
func deleteProject(netClient *http.Client, projectID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, PROJECTS_PATH, projectID, token)
}

// get project by id
func getProjectByID(netClient *http.Client, projectID string, token string) (model.Project, error) {
	project := model.Project{}
	err := doGet(netClient, PROJECTS_PATH+"/"+projectID, token, &project)
	return project, err
}

func TestProject(t *testing.T) {
	t.Parallel()
	t.Log("running TestProject test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user2 := apitesthelper.CreateUser(t, dbAPI, tenantID, "USER")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Project", func(t *testing.T) {
		// login as user to get token
		token := loginUser(t, netClient, user)
		token2 := loginUser(t, netClient, user2)

		project := makeExplicitProject(tenantID, nil, nil, nil, nil)
		_, _, err = createProject(netClient, &project, token)
		require.NoError(t, err)
		t.Logf("created project: %+v", project)

		projects, err := getProjects(netClient, token)
		require.NoError(t, err)
		if len(projects) != 1 {
			t.Fatalf("expected projects count 1, got %d", len(projects))
		}
		project.TenantID = projects[0].TenantID
		project.Version = projects[0].Version
		project.CreatedAt = projects[0].CreatedAt
		project.UpdatedAt = projects[0].UpdatedAt
		if !reflect.DeepEqual(project, projects[0]) {
			t.Logf("CloudCredentialIDs nil ? %t:%t", projects[0].CloudCredentialIDs == nil, project.CloudCredentialIDs == nil)
			t.Logf("DockerProfileIDs nil ? %t:%t", projects[0].DockerProfileIDs == nil, project.DockerProfileIDs == nil)
			t.Logf("EdgeIDs nil ? %t:%t", projects[0].EdgeIDs == nil, project.EdgeIDs == nil)
			t.Logf("Users nil ? %t:%t", projects[0].Users == nil, project.Users == nil)
			t.Logf("EdgeSelectors nil ? %t:%t", projects[0].EdgeSelectors == nil, project.EdgeSelectors == nil)
			t.Fatalf("expect project equality, but %+v != %+v", project, projects[0])
		}
		t.Logf("got project: %+v", project)

		projects2, err := getProjects(netClient, token2)
		require.NoError(t, err)
		if len(projects2) != 0 {
			t.Fatalf("expected projects 2 count 0, got %d", len(projects))
		}

		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID
		t.Logf("edge created: %+v", edge)

		// update project
		project.EdgeIDs = []string{edgeID}
		projectID := project.ID
		project.ID = ""
		project.TenantID = ""
		ur, _, err := updateProject(netClient, projectID, project, token)
		require.NoError(t, err)
		if ur.ID != projectID {
			t.Fatal("update project id mismatch")
		}

		resp, _, err := deleteProject(netClient, projectID, token)
		require.NoError(t, err)
		if resp.ID != projectID {
			t.Fatal("id mismatch in delete")
		}

	})

}

func TestProjectPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestProjectPaging test")

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

	t.Run("Test Project Paging", func(t *testing.T) {
		// login as user to get token
		token := loginUser(t, netClient, user)

		// randomly create some projects
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d projects...", n)
		for i := 0; i < n; i++ {
			project := makeExplicitProject(tenantID, nil, nil, nil, nil)
			_, _, err = createProject(netClient, &project, token)
			require.NoError(t, err)
		}

		projects, err := getProjects(netClient, token)
		require.NoError(t, err)
		if len(projects) != n {
			t.Fatalf("expected projects count to be %d, but got %d", n, len(projects))
		}
		sort.Sort(model.ProjectsByID(projects))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pProjects := []model.Project{}
		nRemain := n
		t.Logf("fetch %d projects using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nccs, err := getProjectsNew(netClient, token, i, pageSize)
			require.NoError(t, err)
			if nccs.PageIndex != i {
				t.Fatalf("expected page index to be %d, but got %d", i, nccs.PageIndex)
			}
			if nccs.PageSize != pageSize {
				t.Fatalf("expected page size to be %d, but got %d", pageSize, nccs.PageSize)
			}
			if nccs.TotalCount != n {
				t.Fatalf("expected total count to be %d, but got %d", n, nccs.TotalCount)
			}
			nexp := nRemain
			if nexp > pageSize {
				nexp = pageSize
			}
			if len(nccs.ProjectList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nccs.ProjectList))
			}
			nRemain -= pageSize
			for _, cc := range nccs.ProjectList {
				pProjects = append(pProjects, cc)
			}
		}

		// verify paging api gives same result as old api
		for i := range pProjects {
			if !reflect.DeepEqual(projects[i], pProjects[i]) {
				t.Fatalf("expect projects equal, but %+v != %+v", projects[i], pProjects[i])
			}
		}
		t.Log("get projects from paging api gives same result as old api")

		for _, project := range projects {
			resp, _, err := deleteProject(netClient, project.ID, token)
			require.NoError(t, err)
			if resp.ID != project.ID {
				t.Fatal("delete project id mismatch")
			}
		}

	})

}

package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"runtime"
	"testing"

	"github.com/dgrijalva/jwt-go"
	"github.com/google/uuid"
)

func init() {
	apitesthelper.StartServices(&apitesthelper.StartServicesConfig{StartPort: 9090})
}

func newObjectModelAPI(t *testing.T) api.ObjectModelAPI {
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	return dbAPI
}

func makeUserAuthContext(user model.User) *base.AuthContext {
	return &base.AuthContext{
		TenantID: user.TenantID,
		Claims: jwt.MapClaims{
			"specialRole": model.GetUserSpecialRole(&user),
			"id":          user.ID,
		},
	}
}
func makeUserApiAuthContext(user model.User, tokenID string) *base.AuthContext {
	return &base.AuthContext{
		TenantID: user.TenantID,
		Claims: jwt.MapClaims{
			"specialRole": model.GetUserSpecialRole(&user),
			"id":          user.ID,
			"tokenId":     tokenID,
		},
	}
}
func makeUserContext(user model.User) context.Context {
	return context.WithValue(context.Background(), base.AuthContextKey, makeUserAuthContext(user))
}
func makeUserApiContext(user model.User, tokenID string) context.Context {
	return context.WithValue(context.Background(), base.AuthContextKey, makeUserApiAuthContext(user, tokenID))
}

func makeAuthContexts(tenantID string, projectIDs []string) (*base.AuthContext, *base.AuthContext, *base.AuthContext) {
	projRoles := []model.ProjectRole{}
	for _, projectID := range projectIDs {
		projRoles = append(projRoles, model.ProjectRole{
			ProjectID: projectID,
			Role:      model.ProjectRoleAdmin,
		})
	}
	authContext1 := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"projects":    projRoles,
		},
	}
	authContext2 := &base.AuthContext{
		TenantID: tenantID,
		Claims:   jwt.MapClaims{},
	}
	authContext3 := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"projects": projRoles,
		},
	}
	return authContext1, authContext2, authContext3
}

func makeEdgeAuthContext(tenantID string, edgeID string, projectIDs []string) *base.AuthContext {
	projRoles := []model.ProjectRole{}
	for _, projectID := range projectIDs {
		projRoles = append(projRoles, model.ProjectRole{
			ProjectID: projectID,
			Role:      model.ProjectRoleAdmin,
		})
	}

	return &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "edge",
			"edgeId":      edgeID,
			"projects":    projRoles,
		},
	}
}

func makeContext(tenantID string, projectIDs []string) (context.Context, context.Context, context.Context) {
	authContext1, authContext2, authContext3 := makeAuthContexts(tenantID, projectIDs)
	ctx1 := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
	ctx2 := context.WithValue(context.Background(), base.AuthContextKey, authContext2)
	ctx3 := context.WithValue(context.Background(), base.AuthContextKey, authContext3)
	return ctx1, ctx2, ctx3
}

func makeEdgeContext(tenantID string, edgeID string, projectIDs []string) context.Context {

	authContext := makeEdgeAuthContext(tenantID, edgeID, projectIDs)
	return context.WithValue(context.Background(), base.AuthContextKey, authContext)
}

// convert from w (selectAllWFn output) into out
// utility function to help verify selectAll and selectAllW give same result
func selectAllConverter(ctx context.Context, selectAllWFn func(context.Context, io.Writer, *http.Request) error, out interface{}, w io.ReadWriter) error {
	err := selectAllWFn(ctx, w, nil)
	if err != nil {
		return err
	}
	return json.NewDecoder(w).Decode(out)
}

func testSelectAllCommon(t *testing.T, ctx context.Context, selectAllFn func(ctx context.Context, w io.Writer, r *http.Request) error, result interface{}, isZero bool) {
	var w bytes.Buffer
	r := http.Request{URL: &url.URL{}}
	var err error

	err = selectAllFn(ctx, &w, &r)
	require.NoError(t, err)
	err = json.NewDecoder(&w).Decode(result)
	require.NoError(t, err)

	valuePtr := reflect.ValueOf(result)
	value := valuePtr.Elem()
	fnName := runtime.FuncForPC(reflect.ValueOf(selectAllFn).Pointer()).Name()
	if value.Len() != 0 && isZero {
		t.Fatalf("%s: expect select all count to be 0", fnName)
	}
	if value.Len() == 0 && !isZero {
		t.Fatalf("%s: expect select all count to be non zero", fnName)
	}
}

func testSelectAll(t *testing.T, ctx context.Context, selectAllFn func(ctx context.Context, w io.Writer, r *http.Request) error, result interface{}) {
	testSelectAllCommon(t, ctx, selectAllFn, result, true)
}

func testSelectAllV2Common(t *testing.T, ctx context.Context, selectAllFn func(ctx context.Context, w io.Writer, r *http.Request) error, result interface{}, isZero bool, resultFieldName string) {
	var w bytes.Buffer
	r := http.Request{URL: &url.URL{}}
	var err error

	err = selectAllFn(ctx, &w, &r)
	require.NoError(t, err)
	err = json.NewDecoder(&w).Decode(result)
	require.NoError(t, err)

	valuePtr := reflect.ValueOf(result)
	value := valuePtr.Elem()
	fnName := runtime.FuncForPC(reflect.ValueOf(selectAllFn).Pointer()).Name()
	if value.Kind() == reflect.Struct {
		f := value.FieldByName(resultFieldName)
		if f.Len() != 0 && isZero {
			t.Fatalf("%s: expect select all count to be 0", fnName)
		}
		if f.Len() == 0 && !isZero {
			t.Fatalf("%s: expect select all count to be non zero", fnName)
		}
	} else {
		t.Fatalf("%s: expect select all result to be struct", fnName)
	}
}
func testSelectAllV2(t *testing.T, ctx context.Context, selectAllFn func(ctx context.Context, w io.Writer, r *http.Request) error, result interface{}, resultFieldName string) {
	testSelectAllV2Common(t, ctx, selectAllFn, result, true, resultFieldName)
}

// func (dbAPI *dbObjectModelAPI) SelectAllApplicationsForProjectW(context context.Context, projectID string, w io.Writer, req *http.Request) error {
func testSelectAllForProject(t *testing.T, ctx context.Context, projectID string, selectAllFn func(ctx context.Context, projectID string, w io.Writer, r *http.Request) error, result interface{}) {
	var w bytes.Buffer
	r := http.Request{URL: &url.URL{}}
	var err error

	err = selectAllFn(ctx, projectID, &w, &r)
	require.NoError(t, err)
	err = json.NewDecoder(&w).Decode(result)
	require.NoError(t, err)

	valuePtr := reflect.ValueOf(result)
	value := valuePtr.Elem()
	if value.Len() != 0 {
		fnName := runtime.FuncForPC(reflect.ValueOf(selectAllFn).Pointer()).Name()
		t.Fatalf("%s: expect select all for project count to be 0", fnName)
	}
}

func testSelectAllForProjectV2(t *testing.T, ctx context.Context, projectID string, selectAllFn func(ctx context.Context, projectID string, w io.Writer, r *http.Request) error, result interface{}, resultFieldName string) {
	var w bytes.Buffer
	r := http.Request{URL: &url.URL{}}
	var err error

	err = selectAllFn(ctx, projectID, &w, &r)
	require.NoError(t, err)
	err = json.NewDecoder(&w).Decode(result)
	require.NoError(t, err)

	valuePtr := reflect.ValueOf(result)
	value := valuePtr.Elem()
	if value.Kind() == reflect.Struct {
		f := value.FieldByName(resultFieldName)
		if f.Len() != 0 {
			fnName := runtime.FuncForPC(reflect.ValueOf(selectAllFn).Pointer()).Name()
			t.Fatalf("%s: expect select all count to be 0", fnName)
		}
	}
}

func TestGetAllZeroState(t *testing.T) {
	t.Parallel()
	t.Log("running TestGetAllZeroState test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createExplicitProjectCommon(t, dbAPI, tenantID, nil, nil, nil, nil)
	projectID := project.ID

	ctx, _, _ := makeContext(tenantID, []string{projectID})

	defer func() {
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("GetAllZeroState", func(t *testing.T) {
		t.Log("running GetAllZeroState test")

		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllApplicationsForProjectW, &[]model.Application{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllCloudCredsForProjectW, &[]model.CloudCreds{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllContainerRegistriesForProjectW, &[]model.ContainerRegistry{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllDataSourcesForProjectW, &[]model.DataSource{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllDataStreamsForProjectW, &[]model.DataStream{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllDockerProfilesForProjectW, &[]model.DockerProfile{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllEdgesForProjectW, &[]model.Edge{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllScriptsForProjectW, &[]model.Script{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllScriptRuntimesForProjectW, &[]model.ScriptRuntime{})
		testSelectAllForProject(t, ctx, projectID, dbAPI.SelectAllUsersForProjectW, &[]model.User{})

		testSelectAll(t, ctx, dbAPI.SelectAllApplicationsW, &[]model.Application{})
		testSelectAll(t, ctx, dbAPI.SelectAllApplicationsStatusW, &[]model.ApplicationStatus{})
		testSelectAll(t, ctx, dbAPI.SelectAllCategoriesW, &[]model.Category{})
		testSelectAll(t, ctx, dbAPI.SelectAllCloudCredsW, &[]model.CloudCreds{})
		testSelectAll(t, ctx, dbAPI.SelectAllContainerRegistriesW, &[]model.ContainerRegistry{})
		testSelectAll(t, ctx, dbAPI.SelectAllDataSourcesW, &[]model.DataSource{})
		testSelectAll(t, ctx, dbAPI.SelectAllDataStreamsW, &[]model.DataStream{})
		testSelectAll(t, ctx, dbAPI.SelectAllDockerProfilesW, &[]model.DockerProfile{})
		testSelectAll(t, ctx, dbAPI.SelectAllEdgesW, &[]model.Edge{})
		testSelectAll(t, ctx, dbAPI.SelectAllEdgeCertsW, &[]model.EdgeCert{})
		testSelectAll(t, ctx, dbAPI.SelectAllEdgesInfoW, &[]model.EdgeInfo{})
		testSelectAllCommon(t, ctx, dbAPI.SelectAllProjectsW, &[]model.Project{}, false)
		testSelectAll(t, ctx, dbAPI.SelectAllScriptsW, &[]model.Script{})
		testSelectAll(t, ctx, dbAPI.SelectAllScriptRuntimesW, &[]model.ScriptRuntime{})
		testSelectAll(t, ctx, dbAPI.SelectAllSensorsW, &[]model.Sensor{})
		testSelectAll(t, ctx, dbAPI.SelectAllUsersW, &[]model.User{})
		testSelectAllCommon(t, ctx, dbAPI.SelectAllTenantsW, &[]model.Tenant{}, false)

		testSelectAllV2(t, ctx, dbAPI.SelectAllApplicationsWV2, &model.ApplicationListResponsePayload{}, "ApplicationListV2")
		testSelectAllV2(t, ctx, dbAPI.SelectAllApplicationsStatusWV2, &model.ApplicationStatusListPayload{}, "ApplicationStatusList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllCategoriesWV2, &model.CategoryListResponsePayload{}, "CategoryList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllCloudCredsWV2, &model.CloudCredsListResponsePayload{}, "CloudCredsList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllContainerRegistriesWV2, &model.ContainerRegistryListPayload{}, "ContainerRegistryListV2")
		testSelectAllV2(t, ctx, dbAPI.SelectAllDataSourcesWV2, &model.DataSourceListPayload{}, "DataSourceListV2")
		testSelectAllV2(t, ctx, dbAPI.SelectAllDataStreamsWV2, &model.DataStreamListPayload{}, "DataStreamList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllDockerProfilesWV2, &model.DockerProfileListPayload{}, "DockerProfileList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllEdgesWV2, &model.EdgeListPayload{}, "EdgeListV2")
		testSelectAllV2(t, ctx, dbAPI.SelectAllEdgesInfoWV2, &model.EdgeInfoListPayload{}, "EdgeUsageInfoList")
		testSelectAllV2Common(t, ctx, dbAPI.SelectAllProjectsWV2, &model.ProjectListPayload{}, false, "ProjectList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllScriptsWV2, &model.ScriptListPayload{}, "ScriptList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllScriptRuntimesWV2, &model.ScriptRuntimeListPayload{}, "ScriptRuntimeList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllSensorsWV2, &model.SensorListPayload{}, "SensorList")
		testSelectAllV2(t, ctx, dbAPI.SelectAllUsersWV2, &model.UserListPayload{}, "UserList")

		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllApplicationsForProjectWV2, &model.ApplicationListResponsePayload{}, "ApplicationListV2")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllCloudCredsForProjectWV2, &model.CloudCredsListResponsePayload{}, "CloudCredsList")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllContainerRegistriesForProjectWV2, &model.ContainerRegistryListPayload{}, "ContainerRegistryListV2")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllDataSourcesForProjectWV2, &model.DataSourceListPayload{}, "DataSourceListV2")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllDataStreamsForProjectWV2, &model.DataStreamListPayload{}, "DataStreamList")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllDockerProfilesForProjectWV2, &model.DockerProfileListPayload{}, "DockerProfileList")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllEdgesForProjectWV2, &model.EdgeListPayload{}, "EdgeListV2")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllScriptsForProjectWV2, &model.ScriptListPayload{}, "ScriptList")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllScriptRuntimesForProjectWV2, &model.ScriptRuntimeListPayload{}, "ScriptRuntimeList")
		testSelectAllForProjectV2(t, ctx, projectID, dbAPI.SelectAllUsersForProjectWV2, &model.UserListPayload{}, "UserList")
	})
}

func TestDBConstraintError(t *testing.T) {
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	catName := base.GetUUID()
	catValues := []string{"v1", "v2"}
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	categoryID := base.GetUUID()
	categoryDoc := model.Category{
		BaseModel: model.BaseModel{
			ID:       categoryID,
			TenantID: tenantID,
			Version:  0,
		},
		Name:    catName,
		Purpose: "",
		Values:  catValues,
	}
	_, err := dbAPI.CreateCategory(ctx, &categoryDoc, nil)
	require.NoError(t, err)
	defer func() {
		dbAPI.DeleteCategory(ctx, categoryID, nil)
	}()
	// Same ID
	categoryDoc.Name = categoryDoc.Name + "-1"
	_, err = dbAPI.CreateCategory(ctx, &categoryDoc, nil)
	require.Error(t, err, "Error expected")
	var dupErr *errcode.DatabaseDuplicateError
	var ok bool
	if dupErr, ok = err.(*errcode.DatabaseDuplicateError); !ok {
		t.Fatal("DB duplicate error expected")
	}
	if dupErr.ID != "id" {
		t.Fatalf("Mismatched ID. Expected duplicate in 'id', found %s", dupErr.ID)
	}
	// Different ID but same name
	categoryDoc.ID = base.GetUUID()
	categoryDoc.Name = catName
	_, err = dbAPI.CreateCategory(ctx, &categoryDoc, nil)
	require.Error(t, err, "Error expected")
	if dupErr, ok = err.(*errcode.DatabaseDuplicateError); !ok {
		t.Fatal("DB duplicate error expected")
	}
	if dupErr.ID != "name" {
		t.Fatalf("Mismatched ID. Expected duplicate in 'name', found %s", dupErr.ID)
	}

	// Non-existing tenant - FK violation
	tenantID = base.GetUUID()
	authContext = &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx = context.WithValue(context.Background(), base.AuthContextKey, authContext)

	categoryDoc.TenantID = tenantID
	categoryDoc.Name = base.GetUUID()
	_, err = dbAPI.CreateCategory(ctx, &categoryDoc, nil)
	require.Error(t, err, "Error expected")

	depErr, ok := err.(*errcode.DatabaseDependencyError)
	assert.True(t, ok, "DB dependency error expected")
	assert.Equal(t, depErr.ID, "tenant_id")
}

func testForMarshallability(t *testing.T, obj interface{}) {
	data, err := json.Marshal(obj)
	require.NoError(t, err)
	t.Logf("%+v", obj)
	t.Logf("%s", string(data))
}

func testForCreationWithIDs(createFn func(id string) (interface{}, error), getFn func(id string) (interface{}, error), deleteFn func(id string) (interface{}, error)) func(t *testing.T) {
	type testDesc struct {
		name string
		id   string
		keep bool
	}

	return func(t *testing.T) {
		for _, desc := range []testDesc{
			{"Create with empty ID", "", false},
			{"Create with long custom ID", funk.RandomString(37), false},
			{"Create with short custom ID", funk.RandomString(7), false},
			{"Create with UUID ID", base.GetUUID(), true},
		} {
			t.Run(desc.name, func(t *testing.T) {
				invalidId := desc.id
				resp, err := createFn(invalidId)
				require.NoError(t, err)

				var createdID string
				cdr, ok := resp.(model.CreateDocumentResponse)
				if ok {
					createdID = cdr.ID
				} else {
					createdID = resp.(model.CreateDocumentResponseV2).ID
				}

				if desc.keep {
					require.Equal(t, createdID, invalidId)
				} else {
					_, err = uuid.Parse(createdID)
					require.NoErrorf(t, err, "Created object ID is not UUID id=%s", createdID)
					require.NotEqual(t, createdID, invalidId)

					_, err = getFn(invalidId)
					require.Errorf(t, err, "Should not be available via invalid id=%s", invalidId)
				}

				_, err = getFn(createdID)
				require.NoErrorf(t, err, "Should be available via invalid id=%s", createdID)

				_, err = deleteFn(createdID)
				require.NoErrorf(t, err, "Failed to delete id=%s", createdID)
			})
		}
	}
}

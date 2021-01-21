package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	"strings"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	TestCategoryValue1 = "v1"
	TestCategoryValue2 = "v2"
)

func createCategoryCommon(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, name string, values []string) model.Category {
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
		Name:    name,
		Purpose: "",
		Values:  values,
	}
	resp3, err := dbAPI.CreateCategory(ctx, &categoryDoc, nil)
	assert.NoError(t, err)
	t.Logf("create category successful, %s", resp3)

	cat, err := dbAPI.GetCategory(ctx, categoryID)
	assert.NoError(t, err)

	return cat
}

func createCategory(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string) model.Category {
	return createCategoryCommon(t, dbAPI, tenantID, "test-cat", []string{TestCategoryValue1, TestCategoryValue2})
}

func noopCallback(t *testing.T) func(_ context.Context, _ interface{}) error {
	return func(_ context.Context, _ interface{}) error {
		t.Log("---> callback called")
		return nil
	}
}

func TestCategory(t *testing.T) {
	t.Parallel()
	t.Log("running TestCategory test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	project := createEmptyCategoryProject(t, dbAPI, tenantID)
	projectID := project.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})

	defer func() {
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteCategory", func(t *testing.T) {
		t.Log("running Create/Get/DeleteCategory test")

		// Category object, leave ID blank and let create generate it
		cat1In := model.Category{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  0,
			},
			Name:    " test-cat ",
			Purpose: "",
			Values:  []string{" v1 ", "v2"},
		}
		// create first category
		resp, err := dbAPI.CreateCategory(ctx, &cat1In, func(ctx context.Context, arg interface{}) error {
			t.Logf("create category callback invoked: %+v", arg)

			bm, ok := arg.(model.Category)
			assert.True(t, ok, "failed to convert create category callback arg")

			t.Logf("create category callback arg converted to category successfully: %+v", bm)
			return nil
		})
		assert.NoError(t, err)

		t.Logf("create category successful, %s", resp)
		catId1 := resp.(model.CreateDocumentResponse).ID

		// get category 1
		cat1, err := dbAPI.GetCategory(ctx, catId1)
		assert.NoError(t, err)
		t.Logf("get category 1 before update successful, %+v", cat1)

		// check cleaned fields
		assert.Equal(t, cat1.Name, "test-cat", "Name not normalized")
		assert.Equal(t, cat1.TenantID, tenantID, "Tenant ID does not match")

		assert.ElementsMatch(t, cat1.Values, []string{"v1", "v2"})

		cats, err := dbAPI.SelectAllCategories(ctx, nil)
		assert.NoError(t, err)
		assert.Equal(t, len(cats), 1)
		assert.Equal(t, cat1, cats[0])

		// create second category
		cat2In := model.Category{
			BaseModel: model.BaseModel{
				ID:       "",
				TenantID: tenantID,
				Version:  5,
			},
			Name:    "test-cat-2",
			Purpose: "test category",
			Values:  []string{"v1", "v2", "v3"},
		}
		resp2, err := dbAPI.CreateCategory(ctx, &cat2In, nil)
		assert.NoError(t, err)
		t.Logf("create category successful, %v", resp)

		catId2 := resp2.(model.CreateDocumentResponse).ID
		cat2In.ID = catId2

		// update category
		cat2Uin := model.Category{
			BaseModel: model.BaseModel{
				ID:       catId2,
				TenantID: tenantID,
				Version:  5,
			},
			Name:    "test-cat-2-updated",
			Purpose: "test category updated",
			Values:  []string{"v1", "v2", "v3-updated"},
		}
		upResp, err := dbAPI.UpdateCategory(ctx, &cat2Uin, nil)
		assert.NoError(t, err)
		t.Logf("update category 2 successful, %+v", upResp)

		categories, err := dbAPI.SelectAllCategories(ctx, nil)
		assert.NoError(t, err)
		for _, cat := range categories {
			testForMarshallability(t, cat)
		}
		t.Logf("select all categories successful")

		// select all vs select all W
		var w bytes.Buffer
		cats1, err := dbAPI.SelectAllCategories(ctx, nil)
		assert.NoError(t, err)

		cats2 := &[]model.Category{}
		err = selectAllConverter(ctx, dbAPI.SelectAllCategoriesW, cats2, &w)
		assert.NoError(t, err)

		assert.ElementsMatch(t, cats1, *cats2, "expect select categories and select categories w results to be equal")

		// get category 1
		cat1, err = dbAPI.GetCategory(ctx, catId1)
		assert.NoError(t, err)
		t.Logf("get category 1 successful, %+v", cat1)

		// get category 2
		cat2, err := dbAPI.GetCategory(ctx, catId2)
		assert.NoError(t, err)
		t.Logf("get category 2 successful, %+v", cat2)

		// delete category 1
		delResp, err := dbAPI.DeleteCategory(ctx, catId1, noopCallback(t))
		assert.NoError(t, err)
		t.Logf("delete category 1 successful, %v", delResp)

		// delete category 2
		delResp, err = dbAPI.DeleteCategory(ctx, catId2, noopCallback(t))
		assert.NoError(t, err)
		t.Logf("delete category 2 successful, %v", delResp)

		// delete category 3
		delResp, err = dbAPI.DeleteCategory(ctx, "foo", noopCallback(t))
		assert.NoError(t, err)
		t.Logf("delete category 3 successful, %v", delResp)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		return dbAPI.CreateCategory(ctx, &model.Category{
			BaseModel: model.BaseModel{
				ID:       id,
				TenantID: tenantID,
			},
			Name:   "test-cat-" + funk.RandomString(10),
			Values: []string{"v1", "v2"},
		}, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetCategory(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteCategory(ctx, id, nil)
	}))

	// select all categories
	t.Run("SelectAllCategories", func(t *testing.T) {
		t.Log("running SelectAllCategories test")
		categories, err := dbAPI.SelectAllCategories(ctx, nil)
		assert.NoError(t, err)
		for _, cat := range categories {
			testForMarshallability(t, cat)
		}
	})
}

// TestCategoryConversion will test Category conversion
func TestCategoryConversion(t *testing.T) {
	t.Parallel()
	// setup
	tenantID := base.GetUUID()
	//var values = []string{"v1", "v2", "v3"}
	var cat = model.Category{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:    "test-cat",
		Purpose: "test category",
	}
	//valuesData, _ := json.Marshal(values)
	var catDBO = api.CategoryDBO{
		BaseModelDBO: model.BaseModelDBO{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:    "test-cat",
		Purpose: "test category",
	}

	// now verify two-way conversion
	cat2 := model.Category{}
	err := base.Convert(&catDBO, &cat2)
	assert.NoError(t, err, "categoryDBO to category failed")
	assert.Equal(t, cat, cat2)

	catDBO2 := api.CategoryDBO{}
	err = base.Convert(&cat, &catDBO2)
	assert.NoError(t, err, "category to categoryDBO failed")
	assert.Equal(t, catDBO, catDBO2)
}

type catUsageSummaryInfo struct {
	ProjectCount      int
	EdgeCount         int
	DataPipelineCount int
	DataSourceCount   int
	ApplicationCount  int
}

func verifyCatUsageInfo(t *testing.T, catUsage *model.CategoryUsageInfo, catUsageSummary catUsageSummaryInfo) {
	assert.Equal(t, len(catUsage.ProjectIDs), catUsageSummary.ProjectCount, "project count mismatch")
	assert.Equal(t, len(catUsage.EdgeIDs), catUsageSummary.EdgeCount, "edge count mismatch")
	assert.Equal(t, len(catUsage.DataPipelineIDs), catUsageSummary.DataPipelineCount, "data pipeline count mismatch")
	assert.Equal(t, len(catUsage.DataSourceIDs), catUsageSummary.DataSourceCount, "data source count mismatch")
	assert.Equal(t, len(catUsage.ApplicationIDs), catUsageSummary.ApplicationCount, "application count mismatch")
}

func verifyCatUsage(t *testing.T, catUsage *model.CategoryUsage, catUsageSummary catUsageSummaryInfo) {
	if catUsage == nil {
		assert.Zero(t, catUsageSummary.ProjectCount, "project count non zero")
		assert.Zero(t, catUsageSummary.EdgeCount, "edge count non zero")
		assert.Zero(t, catUsageSummary.DataPipelineCount, "data pipeline count non zero")
		assert.Zero(t, catUsageSummary.DataSourceCount, "data source count non zero")
		assert.Zero(t, catUsageSummary.ApplicationCount, "application count non zero")
	} else {
		assert.Equal(t, len(catUsage.ProjectIDs), catUsageSummary.ProjectCount, "project count mismatch")
		assert.Equal(t, len(catUsage.EdgeIDs), catUsageSummary.EdgeCount, "edge count mismatch")
		assert.Equal(t, len(catUsage.DataPipelineIDs), catUsageSummary.DataPipelineCount, "data pipeline count mismatch")
		assert.Equal(t, len(catUsage.DataSourceIDs), catUsageSummary.DataSourceCount, "data source count mismatch")
		assert.Equal(t, len(catUsage.ApplicationIDs), catUsageSummary.ApplicationCount, "application count mismatch")
	}
}

func TestCategoryUsageInfo(t *testing.T) {
	t.Parallel()
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	cat1 := createCategoryCommon(t, dbAPI, tenantID, "test-cat-1", []string{"v1", "v2"})
	cat2 := createCategoryCommon(t, dbAPI, tenantID, "test-cat-2", []string{"v1", "v2"})
	cat3 := createCategoryCommon(t, dbAPI, tenantID, "test-cat-3", []string{"v1", "v2"})

	ci := model.CategoryInfo{
		ID:    cat1.ID,
		Value: "v1",
	}
	ci2 := model.CategoryInfo{
		ID:    cat1.ID,
		Value: "v2",
	}
	c2i := model.CategoryInfo{
		ID:    cat2.ID,
		Value: "v1",
	}

	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{ci2})
	edgeID := edge.ID

	dataSource := createDataSource(t, dbAPI, tenantID, edgeID, cat1.ID, "v1")
	dataSourceID := dataSource.ID

	cloudCreds := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cloudCreds.ID

	dockerProfile := createAWSContainerRegistry(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileID := dockerProfile.ID

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, []model.CategoryInfo{ci, ci2})
	projectID := project.ID

	project2 := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, []model.CategoryInfo{c2i})
	project2ID := project2.ID

	scriptRuntime := createScriptRuntime(t, dbAPI, tenantID, projectID, dockerProfileID)
	scriptRuntimeID := scriptRuntime.ID
	script := createScript(t, dbAPI, tenantID, projectID, scriptRuntimeID)
	scriptID := script.ID

	scriptRuntime2 := createScriptRuntime(t, dbAPI, tenantID, project2ID, dockerProfileID)
	scriptRuntime2ID := scriptRuntime2.ID
	script2 := createScript(t, dbAPI, tenantID, project2ID, scriptRuntime2ID)
	script2ID := script2.ID

	dataStream := createDataStream(t, dbAPI, tenantID, projectID, cat1.ID, "v2", cloudCredsID, scriptID)
	dataStream2 := createDataStream(t, dbAPI, tenantID, project2ID, cat2.ID, "v1", cloudCredsID, script2ID)

	application := createApplication(t, dbAPI, tenantID, "test-app-name-1", projectID, nil, []model.CategoryInfo{ci2})

	application2 := testApp(tenantID, projectID, "test-app-name-2", nil, nil, &[]model.CategoryInfo{ci})
	createApplicationWithCallback(t, dbAPI, &application2, tenantID, projectID, nil)

	ctx, _, _ := makeContext(tenantID, []string{projectID})

	defer func() {
		dbAPI.DeleteApplication(ctx, application.ID, nil)
		dbAPI.DeleteApplication(ctx, application2.ID, nil)
		dbAPI.DeleteDataStream(ctx, dataStream.ID, nil)
		dbAPI.DeleteDataStream(ctx, dataStream2.ID, nil)
		dbAPI.DeleteScript(ctx, scriptID, nil)
		dbAPI.DeleteScript(ctx, script2ID, nil)
		dbAPI.DeleteScriptRuntime(ctx, scriptRuntimeID, nil)
		dbAPI.DeleteScriptRuntime(ctx, scriptRuntime2ID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteProject(ctx, project2ID, nil)
		dbAPI.DeleteDockerProfile(ctx, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx, cloudCredsID, nil)
		dbAPI.DeleteDataSource(ctx, dataSourceID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteCategory(ctx, cat1.ID, nil)
		dbAPI.DeleteCategory(ctx, cat2.ID, nil)
		dbAPI.DeleteCategory(ctx, cat3.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("TestCategoryUsageInfo", func(t *testing.T) {
		t.Log("running TestCategoryUsageInfo test")

		results, err := dbAPI.SelectAllCategoriesUsageInfo(ctx)
		assert.NoError(t, err)
		assert.Equal(t, len(results), 3, "expect len of results to be 3, but got %d", len(results))

		appCount := 1
		if *config.Cfg.EnableAppOriginSelectors {
			appCount = 2
		}

		expectedUsageInfo := map[string]catUsageSummaryInfo{
			cat1.ID: {
				ProjectCount:      1,
				EdgeCount:         1,
				DataPipelineCount: 1,
				DataSourceCount:   1,
				ApplicationCount:  appCount,
			},
			cat2.ID: {
				ProjectCount:      1,
				EdgeCount:         0,
				DataPipelineCount: 1,
				DataSourceCount:   0,
				ApplicationCount:  0,
			},
			cat3.ID: {
				ProjectCount:      0,
				EdgeCount:         0,
				DataPipelineCount: 0,
				DataSourceCount:   0,
				ApplicationCount:  0,
			},
		}
		for _, result := range results {
			verifyCatUsageInfo(t, &result, expectedUsageInfo[result.ID])
		}

		expectedDetailUsageInfo := map[string]map[string]catUsageSummaryInfo{
			cat1.ID: {
				"v1": catUsageSummaryInfo{
					ProjectCount:      1,
					EdgeCount:         0,
					DataPipelineCount: 0,
					DataSourceCount:   1,
					ApplicationCount:  appCount - 1,
				},
				"v2": catUsageSummaryInfo{
					ProjectCount:      1,
					EdgeCount:         1,
					DataPipelineCount: 1,
					DataSourceCount:   0,
					ApplicationCount:  1,
				},
			},
			cat2.ID: {
				"v1": catUsageSummaryInfo{
					ProjectCount:      1,
					EdgeCount:         0,
					DataPipelineCount: 1,
					DataSourceCount:   0,
					ApplicationCount:  0,
				},
				"v2": catUsageSummaryInfo{
					ProjectCount:      0,
					EdgeCount:         0,
					DataPipelineCount: 0,
					DataSourceCount:   0,
					ApplicationCount:  0,
				},
			},
			cat3.ID: {
				"v1": catUsageSummaryInfo{
					ProjectCount:      0,
					EdgeCount:         0,
					DataPipelineCount: 0,
					DataSourceCount:   0,
					ApplicationCount:  0,
				},
				"v2": catUsageSummaryInfo{
					ProjectCount:      0,
					EdgeCount:         0,
					DataPipelineCount: 0,
					DataSourceCount:   0,
					ApplicationCount:  0,
				},
			},
		}
		catIDs := []string{cat1.ID, cat2.ID, cat3.ID}
		for _, catID := range catIDs {
			cdu, err := dbAPI.GetCategoryDetailUsageInfo(ctx, catID)
			assert.NoError(t, err)
			cdum := cdu.UsageMap
			for v := range cdum {
				o := cdum[v]
				e := expectedDetailUsageInfo[catID][v]
				verifyCatUsage(t, o, e)
			}
		}
	})
}

func TestCategoryValuesFiltering(t *testing.T) {
	t.Parallel()
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	ctx, _, _ := makeContext(tenantID, []string{})

	cat := model.Category{
		BaseModel: model.BaseModel{TenantID: tenantID},
		Name:      "test-cat",
		Values:    []string{" v1 ", " V1"},
	}

	_, err := dbAPI.CreateCategory(ctx, &cat, nil)
	assert.Error(t, err, "should fail to create non-normalized values: %s", strings.Join(cat.Values, ","))
}

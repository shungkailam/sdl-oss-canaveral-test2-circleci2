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
)

const (
	CATEGORIES_PATH     = "/v1/categories"
	CATEGORIES_PATH_NEW = "/v1.0/categories"
)

// create category
func createCategory(netClient *http.Client, category *model.Category, token string) (model.CreateDocumentResponse, string, error) {
	resp, reqID, err := createEntity(netClient, CATEGORIES_PATH, *category, token)
	if err == nil {
		category.ID = resp.ID
	}
	return resp, reqID, err
}

// update category
func updateCategory(netClient *http.Client, categoryID string, category model.Category, token string) (model.UpdateDocumentResponse, string, error) {
	return updateEntity(netClient, fmt.Sprintf("%s/%s", CATEGORIES_PATH, categoryID), category, token)
}

// get categories
func getCategories(netClient *http.Client, token string) ([]model.Category, error) {
	categories := []model.Category{}
	err := doGet(netClient, CATEGORIES_PATH, token, &categories)
	return categories, err
}
func getCategoriesNew(netClient *http.Client, token string, pageIndex int, pageSize int) (model.CategoryListResponsePayload, error) {
	response := model.CategoryListResponsePayload{}
	path := fmt.Sprintf("%s?pageIndex=%d&pageSize=%d&orderBy=id", CATEGORIES_PATH_NEW, pageIndex, pageSize)
	err := doGet(netClient, path, token, &response)
	return response, err
}

// delete category
func deleteCategory(netClient *http.Client, categoryID string, token string) (model.DeleteDocumentResponse, string, error) {
	return deleteEntity(netClient, CATEGORIES_PATH, categoryID, token)
}

// get category by id
func getCategoryByID(netClient *http.Client, categoryID string, token string) (model.Category, error) {
	category := model.Category{}
	err := doGet(netClient, CATEGORIES_PATH+"/"+categoryID, token, &category)
	return category, err
}

func TestCategory(t *testing.T) {
	t.Parallel()
	t.Log("running TestCategory test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

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

	t.Run("Test Category", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		category := model.Category{
			Name:    "test-cat",
			Purpose: "",
			Values:  []string{"v1", "v2"},
		}
		_, _, err := createCategory(netClient, &category, token)
		require.NoError(t, err)

		categories, err := getCategories(netClient, token)
		require.NoError(t, err)
		if len(categories) != 1 {
			t.Fatalf("expected category count to be 1, got %d", len(categories))
		}
		t.Logf("got categories: %+v", categories)
		cat := categories[0]
		category.TenantID = cat.TenantID
		category.Version = cat.Version
		category.CreatedAt = cat.CreatedAt
		category.UpdatedAt = cat.UpdatedAt

		sort.Strings(cat.Values)
		sort.Strings(category.Values)
		if !reflect.DeepEqual(category, cat) {
			t.Fatalf("expect category equal, but %+v != %+v", category, cat)
		}
		categoryJ, err := getCategoryByID(netClient, category.ID, token)
		require.NoError(t, err)

		sort.Strings(categoryJ.Values)
		if !reflect.DeepEqual(category, categoryJ) {
			t.Fatalf("expect category J equal, but %+v != %+v", category, categoryJ)
		}

		category2 := model.Category{
			Name:    "test-cat-2",
			Purpose: "test category",
			Values:  []string{"v1", "v2", "v3"},
		}
		_, _, err = createCategory(netClient, &category2, token)
		require.NoError(t, err)
		category2Updated := model.Category{
			Name:    "test-cat-2-updated",
			Purpose: "test category updated",
			Values:  []string{"v1", "v2", "v3-updated"},
		}
		ur, _, err := updateCategory(netClient, category2.ID, category2Updated, token)
		require.NoError(t, err)
		if ur.ID != category2.ID {
			t.Fatal("expect update category id to match")
		}
		categories, err = getCategories(netClient, token)
		require.NoError(t, err)
		if len(categories) != 2 {
			t.Fatalf("expected category count to be 2, got %d", len(categories))
		}
		t.Logf("got categories: %+v", categories)

		resp, _, err := deleteCategory(netClient, category.ID, token)
		require.NoError(t, err)
		if resp.ID != category.ID {
			t.Fatal("delete category id mismatch")
		}
		resp, _, err = deleteCategory(netClient, category2.ID, token)
		require.NoError(t, err)
		if resp.ID != category2.ID {
			t.Fatal("delete category 2 id mismatch")
		}
	})

}

func TestCategoryPaging(t *testing.T) {
	t.Parallel()
	t.Log("running TestCategoryPaging test")

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

	t.Run("Test Category Paging", func(t *testing.T) {
		token := loginUser(t, netClient, user)

		// randomly create some categories
		n := 1 + rand1.Intn(11)
		t.Logf("creating %d categories...", n)
		for i := 0; i < n; i++ {
			category := model.Category{
				Name:    fmt.Sprintf("test-cat-%s", base.GetUUID()),
				Purpose: "",
				Values:  []string{"v1", "v2"},
			}
			_, _, err := createCategory(netClient, &category, token)
			require.NoError(t, err)
		}

		categories, err := getCategories(netClient, token)
		require.NoError(t, err)
		if len(categories) != n {
			t.Fatalf("expected categories count to be %d, but got %d", n, len(categories))
		}
		sort.Sort(model.CategoriesByID(categories))

		pageSize := 1 + rand1.Intn(n)
		nPages := (n + pageSize - 1) / pageSize
		pCategories := []model.Category{}
		nRemain := n
		t.Logf("fetch %d categories using paging api with page size %d, %d pages total...", n, pageSize, nPages)
		for i := 0; i < nPages; i++ {
			nccs, err := getCategoriesNew(netClient, token, i, pageSize)
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
			if len(nccs.CategoryList) != nexp {
				t.Fatalf("expected result count to be %d, but got %d", nexp, len(nccs.CategoryList))
			}
			nRemain -= pageSize
			for _, cc := range nccs.CategoryList {
				pCategories = append(pCategories, cc)
			}
		}

		// verify paging api gives same result as old api
		for i := range pCategories {
			if !reflect.DeepEqual(categories[i], pCategories[i]) {
				t.Fatalf("expect categories equal, but %+v != %+v", categories[i], pCategories[i])
			}
		}
		t.Log("get categories from paging api gives same result as old api")

		for _, category := range categories {
			resp, _, err := deleteCategory(netClient, category.ID, token)
			require.NoError(t, err)
			if resp.ID != category.ID {
				t.Fatal("delete category id mismatch")
			}
		}

	})

}

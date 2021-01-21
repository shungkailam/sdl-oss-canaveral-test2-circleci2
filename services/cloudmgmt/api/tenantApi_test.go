package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"
	"reflect"
	"sort"
	"testing"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createTenantWithID(t *testing.T, dbAPI api.ObjectModelAPI, name string, id string) model.Tenant {
	authContext := &base.AuthContext{
		TenantID: id,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	tenantToken, err := apitesthelper.GenTenantToken()
	require.NoError(t, err)
	// Create tenant object
	doc := model.Tenant{
		ID:      id,
		Version: 0,
		Name:    "test tenant",
		Token:   tenantToken,
	}
	// create tenant
	resp, err := dbAPI.CreateTenant(ctx, &doc, nil)
	require.NoError(t, err)
	t.Logf("create tenant successful, %s", resp)
	return doc
}

func createTenant(t *testing.T, dbAPI api.ObjectModelAPI, name string) model.Tenant {
	tenantID := base.GetUUID()
	return createTenantWithID(t, dbAPI, name, tenantID)
}

func TestTenant(t *testing.T) {
	t.Parallel()
	t.Log("running TestTenant test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	t.Run("Create/Get/Delete Tenant", func(t *testing.T) {
		t.Log("running Create/Get/Delete Tenant test")

		tenantName := "Test tenant 1"
		tenantNameUpdated := "Test tenant 1 updated"
		// TODO FIXME - token will be auto generated
		tenantToken := "tenant-1-token"
		// tenant object, leave ID blank and let create generate it
		doc := model.Tenant{
			ID:      "",
			Version: 0,
			Name:    tenantName,
			Token:   tenantToken,
		}
		// create tenant
		resp, err := dbAPI.CreateTenant(context.Background(), &doc, nil)
		require.NoError(t, err)
		t.Logf("create tenant successful, %s", resp)

		tenantID := resp.(model.CreateDocumentResponse).ID
		ctx := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		})
		/*
			// Waldot test tenant ID is taken as the baseline for backward compatibility check
			// This makes sure the critical fields like Dockerfile match.
			oldTenantID := "tenant-id-waldot_test"
			oldCtx := context.WithValue(context.Background(), base.AuthContextKey, &base.AuthContext{
				TenantID: oldTenantID,
				Claims: jwt.MapClaims{
					"specialRole": "admin",
				},
			})
				err = dbAPI.CreateBuiltinTenantObjects(ctx, tenantID)
				require.NoError(t, err)
				for _, builtinCategory := range config.BuiltinCategories {
					id := api.GetBuiltinCategoryID(tenantID, builtinCategory.ID)
					cat, err := dbAPI.GetCategory(ctx, id)
					if err != nil {
						t.Fatalf("Builtin category %s is not found. Error: %s", id, err.Error())
					}
					// Check backward compatibility
					oldId := api.GetBuiltinCategoryID(oldTenantID, builtinCategory.ID)
					oldCat, err := dbAPI.GetCategory(oldCtx, oldId)
					if err != nil {
						t.Fatalf("Builtin category %s is not found. Error: %s", id, err.Error())
					}
					sort.Strings(cat.Values)
					sort.Strings(oldCat.Values)
					if !reflect.DeepEqual(cat.Values, oldCat.Values) || cat.Name != oldCat.Name || cat.Purpose != oldCat.Purpose {
						t.Fatalf("Mismatched builtin category. Expected %+v, found %+v", oldCat, cat)
					}
				}
				for _, builtinScriptRuntime := range config.BuiltinScriptRuntimes {
					id := api.GetBuiltinScriptRuntimeID(tenantID, builtinScriptRuntime.ID)
					sr, err := dbAPI.GetScriptRuntime(ctx, id)
					if err != nil {
						t.Fatalf("Builtin script runtime %s is not found. Error: %s", id, err.Error())
					}
					// Check backward compatibility
					oldId := api.GetBuiltinScriptRuntimeID(oldTenantID, builtinScriptRuntime.ID)
					oldSr, err := dbAPI.GetScriptRuntime(oldCtx, oldId)
					if sr.Name != oldSr.Name || sr.Description != oldSr.Description ||
						sr.Builtin != oldSr.Builtin || sr.Language != oldSr.Language ||
						sr.DockerRepoURI != oldSr.DockerRepoURI || sr.Dockerfile != oldSr.Dockerfile {

						t.Fatalf("Mismatched builtin script runtime. Expected %+v, found %+v", oldSr, sr)
					}
				}
				{
					id := api.GetDefaultProjectID(tenantID)
					proj, err := dbAPI.GetProject(ctx, id)
					if err != nil {
						t.Fatalf("Builtin project %s is not found. Error: %s", id, err.Error())
					}
					// Check backward compatibility
					oldId := api.GetDefaultProjectID(oldTenantID)
					oldProj, err := dbAPI.GetProject(oldCtx, oldId)
					if proj.Name != oldProj.Name {
						t.Fatalf("Mismatched builtin project. Expected %+v, found %+v", oldProj, proj)
					}
				}
				// Creating again must fail with duplicate error
				err = dbAPI.CreateBuiltinTenantObjects(ctx, tenantID)
				require.Error(t, err, err)
				if _, ok := err.(*errcode.DatabaseDuplicateError); !ok {
					t.Fatal(err)
				}
		*/
		// get tenant
		tenant, err := dbAPI.GetTenant(ctx, tenantID)
		require.NoError(t, err)
		t.Log("get tenant before update successful")

		// select all vs select all W
		testGetAllTenants := false
		if testGetAllTenants {
			var w bytes.Buffer
			tenants1, err := dbAPI.SelectAllTenants(ctx)
			require.NoError(t, err)
			tenants2 := &[]model.Tenant{}
			err = selectAllConverter(ctx, dbAPI.SelectAllTenantsW, tenants2, &w)
			require.NoError(t, err)
			sort.Sort(model.TenantsByID(tenants1))
			sort.Sort(model.TenantsByID(*tenants2))
			if !reflect.DeepEqual(&tenants1, tenants2) {
				n1 := len(tenants1)
				n2 := len(*tenants2)
				if n1 != n2 {
					t.Fatalf(">>> expect select tenants and select tenants w results to be equal, but tenants count mismatch: %d != %d", n1, n2)
				} else {
					for i, tnt := range tenants1 {
						tnt2 := (*tenants2)[i]
						if !reflect.DeepEqual(tnt, tnt2) {
							t.Fatalf(">>> expect select tenants and select tenants w results to be equal, but tenant %+v != %+v", tnt, tnt2)
						}
					}
					t.Fatalf(">>> expect select tenants and select tenants w results to be equal, tenants not deep equal, but all individual tenant deep equal???\n")
				}
			}
		}

		// update tenant
		doc = model.Tenant{
			ID:      tenantID,
			Version: 0,
			Name:    tenantNameUpdated,
			Token:   tenantToken,
		}
		_, err = dbAPI.UpdateTenant(ctx, &doc, nil)
		require.NoError(t, err)
		t.Log("update tenant successful")

		// get tenant
		tenant, err = dbAPI.GetTenant(ctx, tenantID)
		require.NoError(t, err)
		t.Log("get tenant successful")

		if tenant.ID != tenantID || tenant.Name != tenantNameUpdated {
			t.Fatal("tenant data mismatch")
		}

		err = dbAPI.DeleteBuiltinTenantObjects(ctx, tenantID)
		require.NoError(t, err)
		// delete tenant
		_, err = dbAPI.DeleteTenant(ctx, tenantID, nil)
		require.NoError(t, err)
		t.Log("delete tenant successful")
	})
}

func TestTenantIdValidity(t *testing.T) {
	t.Parallel()
	t.Log("running TestTenantIdValidity test")
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	// Teardown
	defer dbAPI.Close()

	authContext := &base.AuthContext{
		TenantID: "",
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		tenantToken, err := apitesthelper.GenTenantToken()
		require.NoError(t, err)

		doc := model.Tenant{
			ID:      id,
			Version: 0,
			Name:    "test tenant",
			Token:   tenantToken,
		}
		return dbAPI.CreateTenant(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetTenant(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteTenant(ctx, id, nil)
	}))
}

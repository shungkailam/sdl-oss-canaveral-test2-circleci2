package api_test

import (
	"bytes"
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createAWSContainerRegistry(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsID string) model.ContainerRegistry {
	// create docker profile
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any@email.com",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	dp := generateContainerRegistry(tenantID, cloudCredsID)
	resp, err := dbAPI.CreateContainerRegistry(ctx, &dp, nil)
	require.NoError(t, err)
	t.Logf("create ContainerRegistries successful, %s", resp)

	dp.ID = resp.(model.CreateDocumentResponse).ID
	cr, err := dbAPI.GetContainerRegistry(ctx, dp.ID)
	require.NoError(t, err)
	return cr
}

func generateContainerRegistry(tenantID string, cloudCredsID string) model.ContainerRegistry {
	return model.ContainerRegistry{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:         "aws-registry-name-" + base.GetUUID(),
		Type:         "AWS",
		Server:       "a.b.c.d.e.f",
		CloudCredsID: cloudCredsID,
		Description:  "aws-registry-desc",
	}
}

func createAWSContainerRegistryV2(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsID string) model.ContainerRegistryV2 {
	// create docker profile
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any@email.com",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	dp := generateContainerRegistryV2(tenantID, cloudCredsID)
	resp, err := dbAPI.CreateContainerRegistryV2(ctx, &dp, nil)
	require.NoError(t, err)
	t.Logf("create ContainerRegistriesWV2 successful, %s", resp)

	dp.ID = resp.(model.CreateDocumentResponse).ID
	cr, err := dbAPI.GetContainerRegistry(ctx, dp.ID)
	require.NoError(t, err)
	return cr.ToV2()
}

func generateContainerRegistryV2(tenantID string, cloudCredsID string) model.ContainerRegistryV2 {
	return model.ContainerRegistryV2{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:        "aws-registry-name-" + base.GetUUID(),
		Type:        "AWS",
		Server:      "a.b.c.d.e.f",
		Description: "aws-registry-desc",
		CloudProfileInfo: &model.CloudProfileInfo{
			CloudCredsID: cloudCredsID,
		},
	}
}

func TestContainerRegistries(t *testing.T) {
	t.Parallel()
	t.Log("running TestContainerRegistries test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSContainerRegistry(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileId := dp.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileId}, []string{}, nil)
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
			"email":       "any@email.com",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteContainerRegistry(ctx1, dockerProfileId, nil)
		dbAPI.DeleteCloudCreds(ctx1, cloudCredsID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete ContainerRegistries", func(t *testing.T) {
		t.Log("running Create/Get/Delete ContainerRegistries test")

		dockerProfileDesc := "aws-registry-desc"
		dockerProfileNameUpdated := "aws-registry-name-updated"

		// get ContainerRegistries
		dockerProfile, err := dbAPI.GetContainerRegistry(ctx1, dockerProfileId)
		require.NoError(t, err)
		t.Logf("get ContainerRegistries before update successful, %+v", dockerProfile)

		dockerProfiles, err := dbAPI.SelectContainerRegistriesByIDs(ctx1, []string{dockerProfileId})
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatalf("expected container registries count to be 1, got %d", len(dockerProfiles))
		}
		t.Logf("got container registries by ids: %+v", dockerProfiles)

		// Needed as we need email in the auth context
		projRoles := []model.ProjectRole{
			{
				ProjectID: projectID,
				Role:      model.ProjectRoleAdmin,
			},
		}
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
				"projects":    projRoles,
				"email":       "any@email.com",
			},
		}
		//ctx1 = ctx1.Value()
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		// update ContainerRegistries
		dp.Name = dockerProfileNameUpdated
		upResp, err := dbAPI.UpdateContainerRegistry(ctx, &dp, nil)
		require.NoError(t, err)
		t.Logf("update ContainerRegistries successful, %+v", upResp)

		// test SelectAllContainerRegistries
		dockerProfiles, err = dbAPI.SelectAllContainerRegistries(ctx1, nil)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("ContainerRegistries count mismatch")
		}

		dockerProfiles, err = dbAPI.SelectAllContainerRegistries(ctx2, nil)
		require.NoError(t, err)
		if len(dockerProfiles) != 0 {
			t.Fatal("Unexpected non-zero docker profiles count")
		}

		dockerProfiles, err = dbAPI.SelectAllContainerRegistries(ctx3, nil)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("Unexpected docker profiles count")
		}

		// select all vs select all W
		var w bytes.Buffer
		dps1, err := dbAPI.SelectAllContainerRegistries(ctx1, nil)
		require.NoError(t, err)
		dps2 := &[]model.ContainerRegistry{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllContainerRegistriesW, dps2, &w)
		require.NoError(t, err)
		sort.Sort(model.ContainerRegistriesByID(dps1))
		sort.Sort(model.ContainerRegistriesByID(*dps2))
		if !reflect.DeepEqual(&dps1, dps2) {
			t.Fatalf("expect select docker profiles and select docker profiles w results to be equal %+v vs %+v", dps1, *dps2)
		}

		// test SelectAllContainerRegistriesForProject
		authContext1 = &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
				"email":       "any@email.com",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		dockerProfiles, err = dbAPI.SelectAllContainerRegistriesForProject(newCtx, projectID, nil)
		// expect this to fail, since for project call require project membership, infra admin is not sufficient
		require.Error(t, err, "expect select all docker profiles to fail for auth 1")
		dockerProfiles, err = dbAPI.SelectAllContainerRegistriesForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect select all docker profiles to fail for auth 2")

		dockerProfiles, err = dbAPI.SelectAllContainerRegistriesForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("Unexpected docker profiles for project count")
		}

		// test GetContainerRegistry
		dockerProfile, err = dbAPI.GetContainerRegistry(ctx1, dockerProfileId)
		require.NoError(t, err)
		t.Logf("get ContainerRegistry successful, %+v", dockerProfile)

		if dockerProfile.ID != dockerProfileId || dockerProfile.Name != dockerProfileNameUpdated || dockerProfile.Description != dockerProfileDesc {
			t.Fatal("ContainerRegistry data mismatch")
		}
		dockerProfile, err = dbAPI.GetContainerRegistry(ctx2, dockerProfileId)
		require.Error(t, err, "Expected not found error")
		dockerProfile, err = dbAPI.GetContainerRegistry(ctx3, dockerProfileId)
		require.NoError(t, err, "Unexpected GetContainerRegistry error")
		if dockerProfile.ID != dockerProfileId || dockerProfile.Name != dockerProfileNameUpdated || dockerProfile.Description != dockerProfileDesc {
			t.Fatal("ContainerRegistry 3 data mismatch")
		}

	})

	// select all ContainerRegistries
	t.Run("SelectAllContainerRegistries", func(t *testing.T) {
		t.Log("running SelectAllContainerRegistriestest")
		dockerProfiles, err := dbAPI.SelectAllContainerRegistries(ctx1, nil)
		require.NoError(t, err)
		for _, dockerProfile := range dockerProfiles {
			testForMarshallability(t, dockerProfile)
		}
	})

	t.Run("ContainerRegistryConversion", func(t *testing.T) {
		t.Log("running ContainerRegistryConversion test")
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		dockerProfileList := []model.ContainerRegistry{
			{
				BaseModel: model.BaseModel{
					ID:        "aws-cloud-creds-id",
					TenantID:  tenantID,
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "foo",
				Type:        "AWS",
				Description: "bar",
			},
		}
		for _, app := range dockerProfileList {
			appDBO := api.ContainerRegistryDBO{}
			app2 := model.ContainerRegistry{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		registry := generateContainerRegistry(tenantID, cloudCredsID)
		registry.ID = id
		return dbAPI.CreateContainerRegistry(ctx, &registry, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetContainerRegistry(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteContainerRegistry(ctx, id, nil)
	}))

	t.Run("ID validity V2", testForCreationWithIDs(func(id string) (interface{}, error) {
		registry := generateContainerRegistryV2(tenantID, cloudCredsID)
		registry.ID = id
		return dbAPI.CreateContainerRegistryV2(ctx, &registry, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetContainerRegistry(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteContainerRegistry(ctx, id, nil)
	}))
}

func TestContainerRegistriesV2(t *testing.T) {
	t.Parallel()
	t.Log("running TestContainerRegistriesV2 test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	cc := createCloudCreds(t, dbAPI, tenantID)
	cloudCredsID := cc.ID
	dp := createAWSContainerRegistryV2(t, dbAPI, tenantID, cloudCredsID)
	dockerProfileID := dp.ID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{cloudCredsID}, []string{dockerProfileID}, []string{}, nil)
	projectID := project.ID
	ctx1, ctx2, ctx3 := makeContext(tenantID, []string{projectID})

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteContainerRegistry(ctx1, dockerProfileID, nil)
		dbAPI.DeleteCloudCreds(ctx1, cloudCredsID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete ContainerRegistries", func(t *testing.T) {
		t.Log("running Create/Get/Delete ContainerRegistries test")

		dockerProfileDesc := "aws-registry-desc"
		dockerProfileNameUpdated := "aws-registry-name-updated"

		// get ContainerRegistries
		var w bytes.Buffer
		err := dbAPI.GetContainerRegistryWV2(ctx1, dockerProfileID, &w, &http.Request{URL: &url.URL{}})
		require.NoError(t, err)
		dockerProfile := model.ContainerRegistryV2{}
		err = json.NewDecoder(&w).Decode(&dockerProfile)
		require.NoError(t, err)
		t.Logf("get ContainerRegistries before update successful, %+v", dockerProfile)

		dockerProfiles, err := dbAPI.SelectContainerRegistriesByIDs(ctx1, []string{dockerProfileID})
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatalf("expected container registries count to be 1, got %d", len(dockerProfiles))
		}
		t.Logf("got container registries by ids: %+v", dockerProfiles)

		// Needed as we need email in the auth context
		projRoles := []model.ProjectRole{
			{
				ProjectID: projectID,
				Role:      model.ProjectRoleAdmin,
			},
		}
		authContext1 := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
				"projects":    projRoles,
				"email":       "any@email.com",
			},
		}
		//ctx1 = ctx1.Value()
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		// update ContainerRegistries
		dp.Name = dockerProfileNameUpdated
		upResp, err := dbAPI.UpdateContainerRegistryV2(ctx, &dp, nil)
		require.NoError(t, err)
		t.Logf("update ContainerRegistries successful, %+v", upResp)

		// select all vs select all W
		cr1, err := dbAPI.SelectAllContainerRegistries(ctx1, nil)
		require.NoError(t, err)

		err = dbAPI.SelectAllContainerRegistriesWV2(ctx, &w, &http.Request{URL: &url.URL{}})
		require.NoError(t, err)

		dpRes := model.ContainerRegistryListPayload{}
		err = json.NewDecoder(&w).Decode(&dpRes)
		require.NoError(t, err)
		dpsFromV2 := model.ContainerRegistriesByIDV2(dpRes.ContainerRegistryListV2).FromV2()
		sort.Sort(model.ContainerRegistriesByID(dpsFromV2))
		if !reflect.DeepEqual(&cr1, &dpsFromV2) {
			t.Fatalf("expect select ContainerRegistries and select ContainerRegistries wv2 results to be equal %+v vs %+v", cr1, dpsFromV2)
		}
		//t.Logf("%+v", cr1)
		if len(dockerProfiles) != 1 {
			t.Fatal("ContainerRegistries count mismatch")
		}

		dps1, err := dbAPI.SelectAllContainerRegistries(ctx1, nil)
		require.NoError(t, err)
		dps2 := &[]model.ContainerRegistry{}
		err = selectAllConverter(ctx1, dbAPI.SelectAllContainerRegistriesW, dps2, &w)
		require.NoError(t, err)
		sort.Sort(model.ContainerRegistriesByID(dps1))
		sort.Sort(model.ContainerRegistriesByID(*dps2))
		if !reflect.DeepEqual(&dps1, dps2) {
			t.Fatalf("expect select docker profiles and select docker profiles w results to be equal %+v vs %+v", dps1, *dps2)
		}

		// test SelectAllContainerRegistriesForProject
		authContext1 = &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
				"email":       "any@email.com",
			},
		}
		newCtx := context.WithValue(context.Background(), base.AuthContextKey, authContext1)
		dockerProfiles, err = dbAPI.SelectAllContainerRegistriesForProject(newCtx, projectID, nil)
		// expect this to fail, since for project call require project membership, infra admin is not sufficient
		require.Error(t, err, "expect select all docker profiles to fail for auth 1")
		dockerProfiles, err = dbAPI.SelectAllContainerRegistriesForProject(ctx2, projectID, nil)
		require.Error(t, err, "expect select all docker profiles to fail for auth 2")

		dockerProfiles, err = dbAPI.SelectAllContainerRegistriesForProject(ctx3, projectID, nil)
		require.NoError(t, err)
		if len(dockerProfiles) != 1 {
			t.Fatal("Unexpected docker profiles for project count")
		}

		// test GetContainerRegistryWV2
		err = dbAPI.GetContainerRegistryWV2(ctx1, dockerProfileID, &w, &http.Request{URL: &url.URL{}})
		require.NoError(t, err)
		dockerProfile = model.ContainerRegistryV2{}
		err = json.NewDecoder(&w).Decode(&dockerProfile)
		require.NoError(t, err)

		t.Logf("get ContainerRegistry successful, %+v", dockerProfile)

		if dockerProfile.ID != dockerProfileID || dockerProfile.Name != dockerProfileNameUpdated || dockerProfile.Description != dockerProfileDesc {
			t.Fatal("ContainerRegistry data mismatch")
		}
		err = dbAPI.GetContainerRegistryWV2(ctx2, dockerProfileID, &w, &http.Request{URL: &url.URL{}})
		require.Error(t, err, "Expected not found error")
		// Project user can not list container registry profiles
		err = dbAPI.GetContainerRegistryWV2(ctx3, dockerProfileID, &w, &http.Request{URL: &url.URL{}})
		require.NoError(t, err)
		dockerProfile = model.ContainerRegistryV2{}
		err = json.NewDecoder(&w).Decode(&dockerProfile)
		require.NoError(t, err)
		t.Logf("get ContainerRegistry successful, %+v", dockerProfile)
		if dockerProfile.ID != dockerProfileID || dockerProfile.Name != dockerProfileNameUpdated || dockerProfile.Description != dockerProfileDesc {
			t.Fatal("ContainerRegistry data mismatch")
		}

	})

	// select all ContainerRegistries
	t.Run("SelectAllContainerRegistries", func(t *testing.T) {
		t.Log("running SelectAllContainerRegistriestest")
		dockerProfiles, err := dbAPI.SelectAllContainerRegistries(ctx1, nil)
		require.NoError(t, err)
		for _, dockerProfile := range dockerProfiles {
			testForMarshallability(t, dockerProfile)
		}
	})

	t.Run("ContainerRegistryConversion", func(t *testing.T) {
		t.Log("running ContainerRegistryConversion test")
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
		dockerProfileList := []model.ContainerRegistry{
			{
				BaseModel: model.BaseModel{
					ID:        "aws-cloud-creds-id",
					TenantID:  tenantID,
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:        "foo",
				Type:        "AWS",
				Description: "bar",
			},
		}
		for _, app := range dockerProfileList {
			appDBO := api.ContainerRegistryDBO{}
			app2 := model.ContainerRegistry{}
			err := base.Convert(&app, &appDBO)
			require.NoError(t, err)
			err = base.Convert(&appDBO, &app2)
			require.NoError(t, err)
			if !reflect.DeepEqual(app, app2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", app, app2)
			}
		}
	})
}

package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func createStorageProfile(t *testing.T, dbAPI api.ObjectModelAPI, svcDomainID string, tenantID string) model.StorageProfile {
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	spName := "sp-" + base.GetUUID()

	volConfig := model.NutanixVolumesStorageProfileConfig{
		PrismElementClusterVIP:  "10.1.1.1",
		PrismElementUserName:    "admin",
		PrismElementPassword:    "admin",
		PrismElementClusterPort: 9123,
		DataServicesIP:          "10.1.1.2",
		DataServicesPort:        9123,
		StorageContainerName:    "default",
		FlashMode:               true,
	}
	// CloudCreds object, leave ID blank and let create generate it
	sp := model.StorageProfile{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:                 spName,
		Type:                 "NutanixVolumes",
		NutanixVolumesConfig: &volConfig,
	}
	// create CloudCreds
	resp, err := dbAPI.CreateStorageProfile(ctx, svcDomainID, &sp)
	require.NoError(t, err)
	t.Logf("create storage profile successful, %s", resp)

	sp.ID = resp.(model.CreateDocumentResponseV2).ID
	return sp
}

func TestStorageProfile(t *testing.T) {
	t.Parallel()
	t.Log("running TestStorageProfile test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	node := createNodeWithLabelsCommon(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	}, "edge", 2)[0]
	svcDomainID := node.SvcDomainID
	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{}, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})
	// Teardown
	defer func() {
		dbAPI.DeleteServiceDomain(ctx1, svcDomainID, nil)
		dbAPI.DeleteCategory(ctx1, categoryID, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()
	t.Run("Create/Get/Update storage profile test", func(t *testing.T) {
		t.Log("running Create/Get/Update storage profile test")
		sp := createStorageProfile(t, dbAPI, svcDomainID, tenantID)
		w := apitesthelper.NewResponseWriter()
		err := dbAPI.SelectAllStorageProfileForServiceDomainW(ctx1, svcDomainID, w, nil)
		require.NoError(t, err)
		payload := &model.StorageProfileListResponsePayload{}
		err = w.GetBody(payload)
		require.NoError(t, err)
		t.Logf("got response %+v", payload)
		spList := payload.StorageProfileList
		if len(spList) != 1 {
			t.Fatalf("expected 1 storage profile info, found %d", len(spList))
		}
		spInfo := spList[0]
		if spInfo.ID != sp.ID || spInfo.Type != "NutanixVolumes" {
			t.Fatalf("Storage profile create and get properties don't match")
		}
		w.Reset()
		sp.Name = "test"
		_, err = dbAPI.UpdateStorageProfile(ctx1, svcDomainID, sp.ID, &sp)
		require.NoError(t, err)

		//Get the storage profile after updating it

		err = dbAPI.SelectAllStorageProfileForServiceDomainW(ctx1, svcDomainID, w, nil)
		require.NoError(t, err)
		payload = &model.StorageProfileListResponsePayload{}
		err = w.GetBody(payload)
		require.NoError(t, err)
		t.Logf("got response %+v", payload)
		spList = payload.StorageProfileList
		if len(spList) != 1 {
			t.Fatalf("expected 1 storage profile info, found %d", len(spList))
		}
		spInfo = spList[0]
		if spInfo.ID != sp.ID || spInfo.Name != "test" {
			t.Fatalf("Storage profile update and get properties don't match")
		}
		w.Reset()
	})

}

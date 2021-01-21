package api_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/jmoiron/sqlx/types"
)

func TestTenantProps(t *testing.T) {
	t.Parallel()
	t.Log("running TestTenantProps test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	user := createUser(t, dbAPI, tenantID)
	userId := user.ID

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx1, _, _ := makeContext(tenantID, []string{projectID})

	// // Teardown x
	defer func() {
		dbAPI.DeleteProject(ctx1, projectID, nil)
		dbAPI.DeleteUser(ctx1, userId, nil)
		dbAPI.DeleteTenant(ctx1, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/DeleteTenantProps", func(t *testing.T) {
		t.Log("running Create/Get/DeleteTenantProps test")

		jt := types.JSONText{}
		ms := map[string]string{}
		ms["foo"] = "bar"
		ms["baz"] = "bam"
		err := base.Convert(&ms, &jt)
		require.NoError(t, err)

		up := model.TenantProps{TenantID: tenantID, Props: jt}
		_, err = dbAPI.UpdateTenantProps(ctx1, &up, nil)
		require.NoError(t, err)

		up2, err := dbAPI.GetTenantProps(ctx1, tenantID)
		require.NoError(t, err)
		t.Logf("Got tenant props: %+v", up2)

		res, err := dbAPI.DeleteTenantProps(ctx1, tenantID, nil)
		require.NoError(t, err)
		t.Logf("Delete tenant props response: %+v", res)

		up2, err = dbAPI.GetTenantProps(ctx1, tenantID)
		require.NoError(t, err)
		t.Logf("Got tenant props 2: %+v", up2)

	})
}

package api_test

import (
	"cloudservices/common/base"
	"cloudservices/common/model"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/jmoiron/sqlx/types"
)

func TestUserProps(t *testing.T) {
	t.Parallel()
	t.Log("running TestUserProps test")
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

	t.Run("Create/Get/DeleteUserProps", func(t *testing.T) {
		t.Log("running Create/Get/DeleteUserProps test")

		// set context to correspond to user
		// since only user can get/update/delete own properties
		authContext, err := base.GetAuthContext(ctx1)
		require.NoError(t, err)
		authContext.ID = userId
		claims := authContext.Claims
		claims["id"] = userId

		jt := types.JSONText{}
		ms := map[string]string{}
		ms["foo"] = "bar"
		ms["baz"] = "bam"
		err = base.Convert(&ms, &jt)
		require.NoError(t, err)

		up := model.UserProps{TenantID: tenantID, UserID: userId, Props: jt}
		_, err = dbAPI.UpdateUserProps(ctx1, &up, nil)
		require.NoError(t, err)

		up2, err := dbAPI.GetUserProps(ctx1, userId)
		require.NoError(t, err)
		t.Logf("Got user props: %+v", up2)

		res, err := dbAPI.DeleteUserProps(ctx1, userId, nil)
		require.NoError(t, err)
		t.Logf("Delete user props response: %+v", res)

		up2, err = dbAPI.GetUserProps(ctx1, userId)
		require.NoError(t, err)
		t.Logf("Got user props 2: %+v", up2)

	})
}

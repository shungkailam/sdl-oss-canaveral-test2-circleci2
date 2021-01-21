package api_test

/*
import (
	"cloudservices/common/model"
	"context"
	"fmt"
	"testing"
)

func TestEdgeUpgrade(t *testing.T) {
	t.Log("running TestEdgeUpgrade test")
	// Setup
	dbAPI := newObjectModelAPI(t)

	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID
	user := createUser(t, dbAPI, tenantID)
	userId := user.ID

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{userId}, nil)
	projectID := project.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})

	// // Teardown x
	defer func() {
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteUser(ctx, userId, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("TestEdgeUpgradeTest", func(t *testing.T) {
		t.Log("running TestEdgeUpgradeTest")

		x := model.ExecuteEdgeUpgrade{
			Release: "v3.0.0",
			EdgeIDs: []string{"edge-id-1", "edge-id-2"},
			Force:   false,
		}

		var doc interface{}
		doc = &x
		_, err := dbAPI.ExecuteEdgeUpgrade(ctx, doc, func(ctx context.Context, y interface{}) error {
			t.Logf(">>> callback: y=%+v\n", y)
			return nil
		})
		require.NoError(t, err)

	})
}
*/

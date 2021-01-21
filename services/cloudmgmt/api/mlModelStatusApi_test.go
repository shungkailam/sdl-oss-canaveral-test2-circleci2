package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func TestMLModelStatus(t *testing.T) {
	t.Parallel()
	t.Log("running TestMLModelStatus test")
	// Setup
	dbAPI := newObjectModelAPI(t)
	doc := createTenant(t, dbAPI, "test tenant")
	tenantID := doc.ID

	category := createCategory(t, dbAPI, tenantID)
	categoryID := category.ID
	// edge 1 is labeled by cat/v1
	edge := createEdgeWithLabels(t, dbAPI, tenantID, []model.CategoryInfo{
		{
			ID:    categoryID,
			Value: TestCategoryValue1,
		},
	})
	edgeID := edge.ID

	// project is cat/v1
	project := createCategoryProjectCommon(
		t, dbAPI, tenantID, []string{}, []string{}, []string{},
		[]model.CategoryInfo{
			{
				ID:    categoryID,
				Value: TestCategoryValue1,
			},
		})
	projectID := project.ID
	authContext, authContext2, authContext3 :=
		makeContext(tenantID, []string{projectID})

	mdl := createMLModel(t, dbAPI, tenantID, projectID)
	modelID := mdl.ID

	// Teardown
	defer func() {
		dbAPI.DeleteMLModel(authContext, modelID, nil)
		dbAPI.DeleteProject(authContext, projectID, nil)
		dbAPI.DeleteEdge(authContext, edgeID, nil)
		dbAPI.DeleteCategory(authContext, categoryID, nil)
		dbAPI.DeleteTenant(authContext, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Create/Get/Delete MLModelStatus", func(t *testing.T) {
		t.Log("running Create/Get/Delete MLModelStatus test")

		doc := model.MLModelStatus{
			TenantID:  tenantID,
			EdgeID:    edgeID,
			ModelID:   modelID,
			ProjectID: &projectID,
			Status: []model.MLModelVersionStatus{
				{
					Status:  "status",
					Version: 1,
				},
			},
		}

		// create MLModel status
		resp, err := dbAPI.CreateMLModelStatus(authContext, &doc, nil)
		require.NoError(t, err)
		t.Logf("create MLModel status successful, %s", resp)

		// get MLModel status
		mlModelStatuses, err := dbAPI.SelectAllMLModelsStatus(authContext)
		require.NoError(t, err)
		if len(mlModelStatuses) != 1 {
			t.Fatalf("Unexpected MLModel status count %d", len(mlModelStatuses))
		}
		for _, mlModelStatus := range mlModelStatuses {
			testForMarshallability(t, mlModelStatus)
		}

		mlModelStatuses, err = dbAPI.SelectAllMLModelsStatus(authContext2)
		require.NoError(t, err)
		if len(mlModelStatuses) != 0 {
			t.Fatalf("Unexpected MLModel status 2 count %d", len(mlModelStatuses))
		}

		mlModelStatuses, err = dbAPI.SelectAllMLModelsStatus(authContext3)
		require.NoError(t, err)
		if len(mlModelStatuses) != 1 {
			t.Fatalf("Unexpected MLModel status 3 count %d", len(mlModelStatuses))
		}

		// delete MLModel status
		delResp, err := dbAPI.DeleteMLModelStatus(authContext, modelID, nil)
		require.NoError(t, err)
		t.Logf("delete MLModel successful, %v", delResp)

	})

	t.Run("MLModelStatusConversion", func(t *testing.T) {
		t.Log("running MLModelStatusConversion test")

		edgeID := "edge-id"
		modelID := "mdl-id"
		now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")

		modelStatuses := []model.MLModelStatus{
			{
				TenantID:  tenantID,
				EdgeID:    edgeID,
				ModelID:   modelID,
				ProjectID: base.StringPtr("proj-id"),
				Version:   5,
				CreatedAt: now,
				UpdatedAt: now,
				Status: []model.MLModelVersionStatus{
					{
						Status:  model.MLModelStatusActive,
						Version: 1,
					},
				},
			},
		}

		for _, modelStatus := range modelStatuses {
			mlDBO := api.MLModelStatusDBO{}
			modelStatus2 := model.MLModelStatus{}
			err := base.Convert(&modelStatus, &mlDBO)
			require.NoError(t, err)
			err = base.Convert(&mlDBO, &modelStatus2)
			require.NoError(t, err)
			if !reflect.DeepEqual(modelStatus, modelStatus2) {
				t.Fatalf("deep equal failed: %+v vs. %+v", modelStatus, modelStatus2)
			}
		}
	})

}

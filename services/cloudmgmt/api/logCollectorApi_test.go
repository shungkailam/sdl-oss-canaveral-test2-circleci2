package api_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/common/model"
	"github.com/jmoiron/sqlx/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thoas/go-funk"
	"testing"
	"time"
)

// Note: to run this test locally you need to have:
// 1. SQL DB running as per settings in config.go
// 2. cfsslserver running locally
func TestLogCollector(t *testing.T) {
	t.Parallel()
	t.Log("running TestLogCollector test")

	// Setup
	dbAPI := newObjectModelAPI(t)

	tenant := createTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID

	edge := createEdge(t, dbAPI, tenantID)
	edgeID := edge.ID

	admin := createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")

	user := createUserWithRole(t, dbAPI, tenantID, "USER")

	impostor := createUserWithRole(t, dbAPI, tenantID, "USER")
	impostorAdmin := createUserWithRole(t, dbAPI, tenantID, "INFRA_ADMIN")

	project := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{user.ID, admin.ID}, nil)
	impostorProject := createCategoryProjectCommon(t, dbAPI, tenantID, []string{}, []string{}, []string{impostorAdmin.ID, impostor.ID}, nil)
	projectID := project.ID
	ctx, _, _ := makeContext(tenantID, []string{projectID})
	adminCtx, _, userCtx := makeContext(tenantID, []string{projectID})
	impostorAdminCtx, _, impostorCtx := makeContext(tenantID, []string{impostorProject.ID})

	cc, err := dbAPI.CreateCloudCreds(ctx, &model.CloudCreds{
		BaseModel: model.BaseModel{
			ID:       "",
			TenantID: tenantID,
			Version:  5,
		},
		Name:        "Test cloud credentials",
		Type:        "AWS",
		Description: "Test",
		AWSCredential: &model.AWSCredential{
			AccessKey: "foo",
			Secret:    "bar",
		},
		GCPCredential: nil,
	}, nil)
	require.NoError(t, err)
	cloudCredsID := cc.(model.CreateDocumentResponse).ID

	dest := model.LogCollectorCloudwatch{
		Destination: "1",
		GroupName:   "2",
		StreamName:  "3",
	}

	sources := model.LogCollectorSources{
		Edges: []string{},
		Tags:  map[string]string{},
	}

	// Teardown
	defer func() {
		dbAPI.DeleteProject(ctx, impostorProject.ID, nil)
		dbAPI.DeleteProject(ctx, projectID, nil)
		dbAPI.DeleteEdge(ctx, edgeID, nil)
		dbAPI.DeleteCloudCreds(ctx, cloudCredsID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)

		dbAPI.Close()
	}()

	t.Run("Test infrastructure log collector for infrastructure user", func(t *testing.T) {
		// initial search
		lcs, err := dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs, "Error during initial find")

		// create log collector
		lc, err := dbAPI.CreateLogCollector(adminCtx, &model.LogCollector{
			Name:              "test name 0",
			Type:              model.InfraCollector,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, lc)

		lcId := lc.(model.CreateDocumentResponse).ID
		require.NotEmpty(t, lcId, "Log collector id not found")

		// modify
		updated, err := dbAPI.UpdateLogCollector(adminCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:              "updated name",
			Type:              model.InfraCollector,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// modify without id
		_, err = dbAPI.UpdateLogCollector(adminCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: "",
			},
			Name:              "updated name",
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.Error(t, err, "Should fail")

		// project modify should fail
		_, err = dbAPI.UpdateLogCollector(userCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:              "updated name",
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.Error(t, err, "Should fail")

		// impostor modify should fail
		_, err = dbAPI.UpdateLogCollector(impostorCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:              "updated name",
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.Error(t, err, "Should fail")

		// update state
		_, err = dbAPI.UpdateStateLogCollector(adminCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:              "updated name",
			State:             model.LogCollectorFailed,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
		}, nil)
		require.NoError(t, err)

		// find newly created
		lcs2, err := dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Len(t, lcs2, 1, "Error during find")

		// get by id
		lc, err = dbAPI.GetLogCollector(adminCtx, lcs2[0].ID)
		require.NoError(t, err)
		require.NotNil(t, lc, "Failed to get by id")

		// get by name
		lcs, err = dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{Filter: "name = 'updated name'"})
		require.NoError(t, err)
		require.Len(t, lcs, 1, "Error during find by name")

		// get by incorrect name
		lcs, err = dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{Filter: "name = 'name'"})
		require.NoError(t, err)
		require.Empty(t, lcs, "Should not find anything")

		// not visible by user
		lcs, err = dbAPI.SelectAllLogCollectors(userCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs, "Should not be visible for user")

		// not visible by impostor
		lcs, err = dbAPI.SelectAllLogCollectors(impostorCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs, "Should not be visible for impostor")

		// delete as user failed
		rsp, err := dbAPI.DeleteLogCollector(userCtx, lcId, nil)
		require.Error(t, err, "Should fail")

		// delete as impostor failed
		rsp, err = dbAPI.DeleteLogCollector(impostorCtx, lcId, nil)
		require.Error(t, err, "Should fail")

		// delete as admin
		rsp, err = dbAPI.DeleteLogCollector(adminCtx, lcId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		// get all again & it should be empty
		lcs3, err := dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs3, "Error during last find")

		// create a project by user
		lc, err = dbAPI.CreateLogCollector(userCtx, &model.LogCollector{
			Name:              "test name 1",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)

		// admin should see this project
		lcs4, err := dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Len(t, lcs4, 1, "Admin should see project level items")

		// cleanup
		rsp, err = dbAPI.DeleteLogCollector(adminCtx, lc.(model.CreateDocumentResponse).ID, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp, "Failed to delete")
	})

	t.Run("Test project log collector allowed for infrastructure user with project access", func(t *testing.T) {
		lc, err := dbAPI.CreateLogCollector(adminCtx, &model.LogCollector{
			Name:              "test name 2",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.NoError(t, err)

		// cleanup
		_, err = dbAPI.DeleteLogCollector(adminCtx, lc.(model.CreateDocumentResponse).ID, nil)
		require.NoError(t, err)
	})

	t.Run("Test project log collector forbidden for infrastructure user", func(t *testing.T) {
		_, err := dbAPI.CreateLogCollector(impostorAdminCtx, &model.LogCollector{
			Name:              "test name 3",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.Error(t, err, "Should fail")
	})

	t.Run("Test infrastructure log collector for project user", func(t *testing.T) {
		lcs, err := dbAPI.SelectAllLogCollectors(userCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs)

		_, err = dbAPI.CreateLogCollector(userCtx, &model.LogCollector{
			Name:              "test name 4",
			Type:              model.InfraCollector,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.Error(t, err, "Should fail")

		lcs, err = dbAPI.SelectAllLogCollectors(userCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs)
	})

	t.Run("Test project log collector for project user", func(t *testing.T) {
		lc, err := dbAPI.CreateLogCollector(userCtx, &model.LogCollector{
			Name:              "test name 5",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)

		require.NoError(t, err)
		require.NotNil(t, lc)

		lcId := lc.(model.CreateDocumentResponse).ID
		require.NotEmpty(t, lcId)

		// visible by admin with access
		lcs, err := dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Len(t, lcs, 1, "Should be visible for admin")

		// NOT visible by admin without access
		lcs, err = dbAPI.SelectAllLogCollectors(impostorAdminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs, "Should not be visible for impostor admin")

		// modify modify by user
		updated, err := dbAPI.UpdateLogCollector(userCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:              "updated name",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// can not modify by admin without access
		updated, err = dbAPI.UpdateLogCollector(impostorAdminCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:         "updated name",
			CloudCredsID: cloudCredsID,
		}, nil)
		require.Error(t, err, "Should fail")

		// can modify by admin with access
		updated, err = dbAPI.UpdateLogCollector(adminCtx, &model.LogCollector{
			BaseModel: model.BaseModel{
				ID: lcId,
			},
			Name:              "updated name",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}, nil)
		require.NoError(t, err)
		require.NotNil(t, updated)

		// delete as impostor failed
		rsp, err := dbAPI.DeleteLogCollector(impostorCtx, lcId, nil)
		require.Error(t, err, "Should fail")

		// delete
		rsp, err = dbAPI.DeleteLogCollector(userCtx, lcId, nil)
		require.NoError(t, err)
		require.NotNil(t, rsp)

		// not visible by admin
		lcs, err = dbAPI.SelectAllLogCollectors(userCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs, "Should be empty")
	})

	t.Run("Test infrastructure log collector for impostor", func(t *testing.T) {
		_, err := dbAPI.CreateLogCollector(impostorCtx, &model.LogCollector{
			Name:              "test name 6",
			Type:              model.InfraCollector,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
		}, nil)
		require.Error(t, err, "Should fail")
	})

	t.Run("Test project log collector for impostor", func(t *testing.T) {
		_, err := dbAPI.CreateLogCollector(impostorCtx, &model.LogCollector{
			Name:              "test name 7",
			Type:              model.ProjectCollector,
			ProjectID:         &projectID,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
		}, nil)
		require.Error(t, err, "Should fail")
	})

	t.Run("Log collectors for impostor", func(t *testing.T) {
		lcs, err := dbAPI.SelectAllLogCollectors(impostorCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		require.Empty(t, lcs)
	})

	t.Run("ID validity", testForCreationWithIDs(func(id string) (interface{}, error) {
		doc := model.LogCollector{
			BaseModel: model.BaseModel{
				ID: id,
			},
			Name:              "test name 0",
			Type:              model.InfraCollector,
			CloudCredsID:      cloudCredsID,
			Destination:       model.AWSCloudWatch,
			CloudwatchDetails: &dest,
			Sources:           sources,
		}
		return dbAPI.CreateLogCollector(ctx, &doc, nil)
	}, func(id string) (interface{}, error) {
		return dbAPI.GetLogCollector(ctx, id)
	}, func(id string) (interface{}, error) {
		return dbAPI.DeleteLogCollector(ctx, id, nil)
	}))

	// select all DataSource for edge
	t.Run("SelectAllLogCollectors", func(t *testing.T) {
		t.Log("running SelectAllLogCollectors test")
		lcs, err := dbAPI.SelectAllLogCollectors(adminCtx, &model.EntitiesQueryParam{})
		require.NoError(t, err)
		for _, lc := range lcs {
			testForMarshallability(t, lc)
		}
	})
}

func createLogCollector(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, cloudCredsID string) model.LogCollector {
	ctx, _, _ := makeContext(tenantID, []string{})

	// create log collector
	lc, err := dbAPI.CreateLogCollector(ctx, &model.LogCollector{
		Name: "test name " + funk.RandomString(10),
		Type: model.InfraCollector,
		Sources: model.LogCollectorSources{
			Edges: nil,
			Tags:  nil,
		},
		CloudCredsID: cloudCredsID,
		Destination:  model.AWSCloudWatch,
		CloudwatchDetails: &model.LogCollectorCloudwatch{
			Destination: "1",
			GroupName:   "2",
			StreamName:  "3",
		},
	}, nil)
	require.NoError(t, err, "Failed to create log collector")

	lcId := lc.(model.CreateDocumentResponse).ID
	require.NotEmpty(t, lcId, "Log collector id not found")

	rsp, err := dbAPI.GetLogCollector(ctx, lcId)
	require.NoError(t, err)

	return rsp
}

func createLogCollectorForProject(t *testing.T, dbAPI api.ObjectModelAPI, tenantID string, projectID string, cloudCredsID string) model.LogCollector {
	ctx, _, _ := makeContext(tenantID, []string{projectID})

	// create log collector
	lc, err := dbAPI.CreateLogCollector(ctx, &model.LogCollector{
		Name:         "test name " + funk.RandomString(10),
		Type:         model.ProjectCollector,
		ProjectID:    &projectID,
		CloudCredsID: cloudCredsID,
		Sources: model.LogCollectorSources{
			Edges: nil,
			Tags:  nil,
		},
		Destination: model.AWSCloudWatch,
		CloudwatchDetails: &model.LogCollectorCloudwatch{
			Destination: "1",
			GroupName:   "2",
			StreamName:  "3",
		},
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, lc)

	lcId := lc.(model.CreateDocumentResponse).ID
	require.NotEmpty(t, lcId, "Log collector id not found")

	rsp, err := dbAPI.GetLogCollector(ctx, lcId)
	require.NoError(t, err)

	return rsp
}

func TestMappingLogCollectorDBO(t *testing.T) {
	now, _ := time.Parse(time.RFC3339, "2018-01-01T01:01:01Z")
	sourceList := model.LogCollectorSources{
		Edges: []string{"e-1", "e-2"},
		Tags: map[string]string{
			"k": "v",
		},
	}
	code := "This is a test code"
	projectID := "project-id"
	cw := types.JSONText(`{"dest":"1","group":"2","stream":"3"}`)

	tests := []struct {
		name    string
		lc      model.LogCollector
		want    api.LogCollectorDBO
		wantErr bool
	}{
		{
			name: "log collector mapping",
			lc: model.LogCollector{
				BaseModel: model.BaseModel{
					ID:        "log-collector-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:         "log-collector-name",
				Type:         model.ProjectCollector,
				State:        model.LogCollectorActive,
				ProjectID:    &projectID,
				Code:         &code,
				Sources:      sourceList,
				CloudCredsID: "cred-id",
				Destination:  model.AWSCloudWatch,
				CloudwatchDetails: &model.LogCollectorCloudwatch{
					Destination: "1",
					GroupName:   "2",
					StreamName:  "3",
				},
			},
			want: api.LogCollectorDBO{
				BaseModelDBO: model.BaseModelDBO{
					ID:        "log-collector-id-1",
					TenantID:  "tenant-id",
					Version:   5,
					CreatedAt: now,
					UpdatedAt: now,
				},
				Name:               "log-collector-name",
				CloudCredsID:       "cred-id",
				Sources:            types.JSONText(`{"edges":["e-1","e-2"],"categories":{"k":"v"}}`),
				Code:               &code,
				State:              "ACTIVE",
				Type:               "Project",
				ProjectID:          &projectID,
				Destination:        "CLOUDWATCH",
				CloudwatchDetails:  &cw,
				KinesisDetails:     nil,
				StackdriverDetails: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := api.ToLogCollectorDBO(&tt.lc)
			require.NoError(t, err)
			assert.Equal(t, got, tt.want)

			back, err := api.FromLogCollectorDBO(&got)
			require.NoError(t, err)
			assert.Equal(t, back, tt.lc)
		})
	}
}

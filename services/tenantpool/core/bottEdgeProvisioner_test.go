package core_test

import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/base"
	"cloudservices/tenantpool/core"
	"cloudservices/tenantpool/model"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBottEdgeProvisioner(t *testing.T) {
	edgeProvisioner, err := core.NewBottEdgeProvisioner()
	require.NoError(t, err)
	ctx := context.Background()
	// Call to create tenant in account service
	tenant, err := core.CreateTenant(ctx, &gapi.Tenant{Name: "Test Tenant"})
	require.NoError(t, err)
	defer func() {
		core.DeleteTenantIfPossible(ctx, tenant.Id)
	}()
	user := &gapi.User{
		TenantId: tenant.Id,
		Name:     "Sherlock Bott Admin",
		Email:    fmt.Sprintf("%s@ntnxsherlock.com", tenant.Id),
		Password: base.GenerateStrongPassword(),
		Role:     "INFRA_ADMIN",
	}
	user, err = core.CreateUser(ctx, user)
	require.NoError(t, err)
	defer func() {
		core.DeleteUser(ctx, user.TenantId, user.Id)
	}()
	t.Logf("Login with email: %s password:%s to see the edge", user.Email, user.Password)
	t.Run("Running BottEdgeProvisioner tests", func(t *testing.T) {
		err := edgeProvisioner.Setup()
		require.NoError(t, err)
		createEdgeConfig := &model.CreateEdgeConfig{
			Name:            "instance",
			TenantID:        tenant.Id,
			SystemUser:      user.Email,
			SystemPassword:  user.Password,
			AppChartVersion: "0.27.0",
		}
		edgeInfo, err := edgeProvisioner.CreateEdge(context.Background(), createEdgeConfig)
		require.NoError(t, err)
		err = doWithDeadline(defaultDeadline, defaultInterval, func() (bool, error) {
			statusInfo, err := edgeProvisioner.GetEdgeStatus(context.Background(), tenant.Id, edgeInfo.ContextID)
			if err != nil {
				return false, err
			}
			if statusInfo.State == core.Created {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err)
		err = doWithDeadline(defaultDeadline, defaultInterval, func() (bool, error) {
			_, err := edgeProvisioner.DeleteEdge(context.Background(), tenant.Id, edgeInfo.ContextID)
			if err != nil {
				return false, err
			}
			return true, nil
		})
		require.NoError(t, err)
		err = doWithDeadline(defaultDeadline, defaultInterval, func() (bool, error) {
			statusInfo, err := edgeProvisioner.GetEdgeStatus(context.Background(), tenant.Id, edgeInfo.ContextID)
			if err != nil {
				return false, err
			}
			if statusInfo.State == core.Deleted {
				return true, nil
			}
			return false, nil
		})
		require.NoError(t, err)
		describeResp, err := edgeProvisioner.DescribeEdge(ctx, tenant.Id, edgeInfo.ContextID)
		require.NoError(t, err)
		data, err := json.Marshal(describeResp)
		require.NoError(t, err)
		t.Log(string(data))
		err = doWithDeadline(defaultDeadline, defaultInterval, func() (bool, error) {
			err := edgeProvisioner.PostDeleteEdges(context.Background(), tenant.Id)
			if err != nil {
				return false, err
			}
			return true, nil
		})
		require.NoError(t, err)
	})
}

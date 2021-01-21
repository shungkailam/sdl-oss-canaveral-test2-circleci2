package api_test

import (
	account "cloudservices/account/generated/grpc"
	"cloudservices/common/base"
	"cloudservices/tenantpool/api"
	"cloudservices/tenantpool/config"
	"cloudservices/tenantpool/core"
	gapi "cloudservices/tenantpool/generated/grpc"
	"cloudservices/tenantpool/model"
	"cloudservices/tenantpool/testhelper"
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/protobuf/ptypes"
	_ "github.com/lib/pq"
)

func TestTenantPool(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	// Override default config
	config.Cfg.TenantPoolScanDelay = base.DurationPtr(time.Second * 3)
	regID1 := base.GetUUID()
	regID2 := base.GetUUID()
	regID3 := base.GetUUID()
	tenant, err := core.CreateTenant(ctx, &account.Tenant{Name: "Test"})
	require.NoError(t, err)
	edgeProvisioner := testhelper.NewTestEdgeProvisioner()
	apiServer := api.NewAPIServerEx(edgeProvisioner)
	tenantPoolManager, err := core.NewTenantPoolManager(edgeProvisioner)
	require.NoError(t, err)
	defer tenantPoolManager.Close()
	regConfig := &model.RegistrationConfigV1{
		VersionInfo:           model.VersionInfo{Version: model.RegConfigV1},
		EdgeCount:             1,
		MinTenantPoolSize:     2,
		MaxTenantPoolSize:     5,
		MaxPendingTenantCount: 2,
		TrialExpiry:           time.Hour,
	}
	config, _ := json.Marshal(regConfig)
	registration1 := &gapi.Registration{
		Id:          regID1,
		Config:      string(config),
		Description: ".NEXT1",
		State:       core.Active,
	}
	registration2 := &gapi.Registration{
		Id:          regID2,
		Config:      string(config),
		Description: ".NEXT2",
		State:       core.Active,
	}
	registration3 := &gapi.Registration{
		Id:          regID3,
		Config:      string(config),
		Description: ".NEXT3",
		State:       core.Active,
	}
	bookKeeper := tenantPoolManager.GetBookKeeper()
	// Cleanup before the test starts
	cleaner := func() {
		bookKeeper.PurgeTenants(ctx, regID1)
		bookKeeper.DeleteRegistration(ctx, regID1)
		bookKeeper.PurgeTenants(ctx, regID2)
		bookKeeper.DeleteRegistration(ctx, regID2)
		bookKeeper.PurgeTenants(ctx, regID3)
		bookKeeper.DeleteRegistration(ctx, regID3)
	}
	cleaner()
	_, err = apiServer.CreateRegistration(ctx, &gapi.CreateRegistrationRequest{Registration: registration1})
	require.NoError(t, err)
	_, err = apiServer.CreateRegistration(ctx, &gapi.CreateRegistrationRequest{Registration: registration2})
	require.NoError(t, err)
	_, err = apiServer.CreateRegistration(ctx, &gapi.CreateRegistrationRequest{Registration: registration3})
	require.NoError(t, err)
	defer cleaner()
	t.Run("Running tenantPoolManager tests", func(t *testing.T) {
		// First wait for 2 tenant claims in creating state from the pool
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Creating}, 2, 2)
		createResp, err := apiServer.CreateTenantClaim(ctx, &gapi.CreateTenantClaimRequest{RegistrationId: regID1, TenantId: tenant.Id})
		require.NoError(t, err)
		getResponse, err := apiServer.GetTenantClaims(ctx, &gapi.GetTenantClaimsRequest{RegistrationId: regID1, TenantId: createResp.TenantClaim.Id})
		require.NoError(t, err)
		if len(getResponse.TenantClaims) != 1 {
			t.Fatal("Expected 1 tenant claim")
		}
		if getResponse.TenantClaims[0].Trial {
			t.Fatal("It must not be trial")
		}
		if getResponse.TenantClaims[0] == nil {
			t.Fatal("Assigned timestamp must be set")
		}
		// All three must be in creating 2 from pool + 1 direct call
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Creating}, 3, 3)
		edgeProvisioner.SetEdgeStatusByCount(ctx, 0, regID1, core.Created)
		// The ones from the pool must be available
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Available}, 2, 2)
		// The direct creation call must assign the tenant
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Assigned}, 1, 1)
		// This sets the state to deleting
		_, err = apiServer.DeleteTenantClaim(ctx, &gapi.DeleteTenantClaimRequest{TenantId: createResp.TenantClaim.Id})
		require.NoError(t, err)
		// The deleting must be gone
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Deleting}, 0, 0)
		// The assigned one must be gone
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Assigned}, 0, 0)
		// Only the tenant claims from min pool size must remain
		testhelper.WaitForState(t, bookKeeper, regID1, []string{core.Available}, 2, 2)
		if edgeProvisioner.GetPostDeleteEdgesCount(createResp.TenantClaim.Id) != 1 {
			t.Fatalf("Expected 1 for PostDeleteEdgesCount, found %d for tenant %s",
				edgeProvisioner.GetPostDeleteEdgesCount(createResp.TenantClaim.Id), createResp.TenantClaim.Id)
		}
	})
	t.Run("Running tenantPoolManager tests", func(t *testing.T) {
		var reservedTenantId string
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Creating}, 2, 2)
		// Fail the edge creation for one edge/tenant
		edgeProvisioner.SetEdgeStatusByCount(ctx, 1, regID2, core.Failed)
		// Tenant state must change to pending delete
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Deleting}, 1, 1)
		// Tenant must be deleted
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Deleting}, 0, 0)
		// All the tenants are pending creation. Assignment must fail
		err = testhelper.DoWithDeadline(testhelper.DefaultDeadline, testhelper.DefaultInterval, func() (bool, error) {
			response, err := apiServer.ReserveTenantClaim(ctx, &gapi.ReserveTenantClaimRequest{RegistrationId: regID2})
			if err != nil {
				return false, err
			}
			reservedTenantId = response.TenantId
			return true, nil
		})
		require.Error(t, err, "Assignment must fail")
		// New edge must be started because of previously failed edge
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Creating}, 2, 2)
		// Update all tenant states to Created
		edgeProvisioner.SetEdgeStatusByCount(ctx, 0, regID2, core.Created)
		// All the tenants are pending creation. Assignment must fail
		err = testhelper.DoWithDeadline(testhelper.DefaultDeadline, testhelper.DefaultInterval, func() (bool, error) {
			response, err := apiServer.ReserveTenantClaim(ctx, &gapi.ReserveTenantClaimRequest{RegistrationId: regID2})
			if err != nil {
				return false, err
			}
			reservedTenantId = response.TenantId
			return true, nil
		})
		require.NoError(t, err)
		getTenantClaimsResponse, err := apiServer.GetTenantClaims(ctx, &gapi.GetTenantClaimsRequest{RegistrationId: regID2, TenantId: reservedTenantId})
		require.NoError(t, err)
		if len(getTenantClaimsResponse.TenantClaims) != 1 {
			t.Fatalf("Exactly one reserved tenant ID is expected for %+v", getTenantClaimsResponse.TenantClaims)
		}
		reservedTenantClaim := getTenantClaimsResponse.TenantClaims[0]
		// Assignment must succeed
		if reservedTenantClaim.State != core.Reserved {
			t.Fatalf("Expected state %s, found %s", core.Reserved, reservedTenantClaim.State)
		}
		if !reservedTenantClaim.Trial {
			t.Fatal("It must be trial")
		}
		if reservedTenantClaim.AssignedAt != nil {
			t.Fatal("Assigned timestamp must not be set")
		}
		if reservedTenantClaim.Resources == nil || len(reservedTenantClaim.Resources) == 0 {
			t.Fatal("Resources must be set")
		}
		if _, ok := reservedTenantClaim.Resources[testhelper.TestProjectID]; !ok {
			t.Fatal("Project ID is missing")
		}
		edgeContextSize := len(reservedTenantClaim.EdgeContexts)
		if edgeContextSize != 1 {
			t.Fatalf("Expected 1 edge context. Found %d", edgeContextSize)
		}
		edgeContext := reservedTenantClaim.EdgeContexts[0]
		if len(edgeContext.EdgeId) == 0 {
			t.Fatalf("Edge ID must be set")
		}
		var confirmedTenantClaim *gapi.TenantClaim
		err = testhelper.DoWithDeadline(testhelper.DefaultDeadline, testhelper.DefaultInterval, func() (bool, error) {
			response, err := apiServer.ConfirmTenantClaim(ctx, &gapi.ConfirmTenantClaimRequest{RegistrationId: regID2, TenantId: reservedTenantClaim.Id})
			if err != nil {
				return false, err
			}
			confirmedTenantClaim = response.TenantClaim
			return true, nil
		})
		require.NoError(t, err)
		if confirmedTenantClaim.ExpiresAt == nil {
			t.Fatal("Expected non-nil expiry time")
		}
		getTenantClaimsResponse, err = apiServer.GetTenantClaims(ctx, &gapi.GetTenantClaimsRequest{TenantId: confirmedTenantClaim.Id})
		require.NoError(t, err)
		if len(getTenantClaimsResponse.TenantClaims) != 1 {
			t.Fatalf("Exactly one reserved tenant ID is expected for %+v", getTenantClaimsResponse.TenantClaims)
		}
		confirmedTenantClaim = getTenantClaimsResponse.TenantClaims[0]
		if confirmedTenantClaim.State != core.Assigned {
			t.Fatalf("Expected state %s, found %s", core.Assigned, confirmedTenantClaim.State)
		}
		if !confirmedTenantClaim.Trial {
			t.Fatal("Expected trial true, found false")
		}
		if confirmedTenantClaim.AssignedAt == nil {
			t.Fatal("Assigned timestamp must be set")
		}
		if confirmedTenantClaim.ExpiresAt == nil {
			t.Fatal("Expected non-nil expiry time")
		}
		confirmedTenantClaim.Trial = false
		trialExpiry, err := ptypes.TimestampProto(base.RoundedNow().Add(time.Hour * 24))
		require.NoError(t, err)
		confirmedTenantClaim.ExpiresAt = trialExpiry
		updateRequest := &gapi.UpdateTenantClaimRequest{TenantClaim: confirmedTenantClaim}
		_, err = apiServer.UpdateTenantClaim(ctx, updateRequest)
		require.NoError(t, err)
		getTenantClaimsResponse, err = apiServer.GetTenantClaims(ctx, &gapi.GetTenantClaimsRequest{TenantId: confirmedTenantClaim.Id})
		require.NoError(t, err)
		if len(getTenantClaimsResponse.TenantClaims) != 1 {
			t.Fatalf("Exactly one reserved tenant ID is expected for %+v", getTenantClaimsResponse.TenantClaims)
		}
		confirmedTenantClaim = getTenantClaimsResponse.TenantClaims[0]
		if confirmedTenantClaim.Trial {
			t.Fatal("Expected trial false, found true")
		}
		if confirmedTenantClaim.ExpiresAt == nil {
			t.Fatal("Expected non-nil expiry time")
		}
		goTime, err := ptypes.Timestamp(confirmedTenantClaim.ExpiresAt)
		require.NoError(t, err)
		if time.Until(goTime) < 23 {
			t.Fatal("Expected expiry to be at least 23 hours later")
		}
		// After the assignment, new tenant must come up with pending create
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Creating}, 1, 1)
		totalEdgeCount := 0
		availableTenantCount := 0
		pendingDeleteTenantCount := 0
		pendingCreateTenantCount := 0
		assignedTenantCount := 0
		failedTenantCount := 0
		pageResponse, err := bookKeeper.ScanTenantClaims(ctx, regID2, "", []string{}, nil, func(registration *model.Registration, tenantClaim *model.TenantClaim) error {
			totalEdgeCount += len(tenantClaim.EdgeContexts)
			switch tenantClaim.State {
			case core.Creating:
				pendingCreateTenantCount++
			case core.Failed:
				failedTenantCount++
			case core.Available:
				availableTenantCount++
			case core.Assigned:
				assignedTenantCount++
			case core.Deleting:
				pendingDeleteTenantCount++

			}
			return nil
		})
		require.NoError(t, err)
		if pageResponse.TotalCount != totalEdgeCount {
			t.Fatalf("Expected %d, found %d", totalEdgeCount, pageResponse.TotalCount)
		}
		if availableTenantCount != 1 && assignedTenantCount != 1 && failedTenantCount != 0 && pendingCreateTenantCount != 0 {
			t.Fatalf("Mismatched counts - %d, %d, %d, %d, %d", availableTenantCount, assignedTenantCount, failedTenantCount, pendingCreateTenantCount, pendingDeleteTenantCount)
		}
		totalExpectedEdgeCount := edgeProvisioner.GetEdgeCount(regID2)
		if totalExpectedEdgeCount != totalEdgeCount {
			t.Fatalf("Mismatched edge count. Expected %d, found %d", totalExpectedEdgeCount, totalEdgeCount)
		}
		// Make the registration inactive
		registration2.State = core.InActive
		_, err = apiServer.UpdateRegistration(ctx, &gapi.UpdateRegistrationRequest{Registration: registration2})
		require.NoError(t, err)
		// Make sure there is one available for assigment
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Available}, 1, 1)
		var reservedTenantId1 string
		// Assignment must fail due to inactive registration
		err = testhelper.DoWithDeadline(testhelper.DefaultDeadline, testhelper.DefaultInterval, func() (bool, error) {
			response, err := apiServer.ReserveTenantClaim(ctx, &gapi.ReserveTenantClaimRequest{RegistrationId: regID2})
			if err != nil {
				return false, err
			}
			reservedTenantId1 = response.TenantId
			return true, nil
		})
		if err == nil || len(reservedTenantId1) > 0 {
			t.Fatal("Assignment must fail")
		}
		_, err = apiServer.DeleteRegistration(ctx, &gapi.DeleteRegistrationRequest{Id: regID2})
		require.NoError(t, err)
		// Update all tenant states to Created
		edgeProvisioner.SetEdgeStatusByCount(ctx, 0, regID2, core.Created)
		// 1 assigned, 1 available, 1 creating
		testhelper.WaitForState(t, bookKeeper, regID2, []string{core.Deleting}, 3, 3)
		// All states should be deleted
		testhelper.WaitForState(t, bookKeeper, regID2, []string{}, 0, 0)
		if edgeProvisioner.GetPostDeleteEdgesCount(reservedTenantId) != 1 {
			t.Fatalf("Expected 1 for PostDeleteEdgesCount, found %d for tenant %s", edgeProvisioner.GetPostDeleteEdgesCount(reservedTenantId), reservedTenantId)
		}
	})
	t.Run("Running recreate tenantClaims test", func(t *testing.T) {
		testhelper.WaitForState(t, bookKeeper, regID3, []string{core.Creating}, 2, 2)
		edgeProvisioner.SetEdgeStatusByCount(ctx, 0, regID3, core.Created)
		testhelper.WaitForState(t, bookKeeper, regID3, []string{core.Available}, 2, 2)
		// All the tenants are pending creation. Assignment must fail
		err = testhelper.DoWithDeadline(testhelper.DefaultDeadline, testhelper.DefaultInterval, func() (bool, error) {
			_, err := apiServer.ReserveTenantClaim(ctx, &gapi.ReserveTenantClaimRequest{RegistrationId: regID3})
			if err != nil {
				return false, err
			}
			return true, nil
		})
		require.NoError(t, err)
		_, err = apiServer.RecreateTenantClaims(ctx, &gapi.RecreateTenantClaimsRequest{RegistrationId: regID3})
		require.NoError(t, err)
		// Only the available one must be in Deleting state
		testhelper.WaitForState(t, bookKeeper, regID3, []string{core.Deleting}, 1, 1)
		testhelper.WaitForState(t, bookKeeper, regID3, []string{core.Creating}, 1, 1)
		// Reserved one must be unaffected
		testhelper.WaitForState(t, bookKeeper, regID3, []string{core.Reserved}, 1, 1)
		// Mark the creating one to Created
		edgeProvisioner.SetEdgeStatusByCount(ctx, 0, regID3, core.Created)
		testhelper.WaitForState(t, bookKeeper, regID3, []string{core.Available}, 1, 1)
	})
}

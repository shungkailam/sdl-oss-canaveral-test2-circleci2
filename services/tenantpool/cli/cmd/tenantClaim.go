package cmd

import (
	"cloudservices/common/base"
	"cloudservices/common/service"
	"context"
	"time"

	gapi "cloudservices/tenantpool/generated/grpc"

	"github.com/golang/protobuf/ptypes"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// tenantClaimCmd represents the tenantClaim command
var tenantClaimCmd = &cobra.Command{
	Use:   "tenantclaim",
	Short: "get, reserve, confirm, update, recreate, delete or scavenge or assign tenantClaim like \"tenantpoolcli tenantclaim get my-tenantclaim\"",
}

type tenantClaimArgs struct {
	id                string
	registrationID    string
	state             string
	email             string
	password          string
	role              string
	filter            string
	orderBy           []string
	trial             bool
	dryRun            bool
	verbose           bool
	updatedBeforeDays int
	trialPeriodHours  int
	pageIndex         int
	pageSize          int
}

var tenantClaimCmdArgs tenantClaimArgs

func init() {
	rootCmd.AddCommand(tenantClaimCmd)
	tenantClaimCreate := &cobra.Command{
		Use:   "create",
		Short: "Create tenantclaim",
		Run:   tenantClaimCreate,
	}

	tenantClaimUpdate := &cobra.Command{
		Use:   "update",
		Short: "Update tenantclaim",
		Run:   tenantClaimUpdate,
	}

	tenantClaimGet := &cobra.Command{
		Use:   "get",
		Short: "Get tenantclaim(s)",
		Run:   tenantClaimGet,
	}

	tenantClaimReserve := &cobra.Command{
		Use:   "reserve",
		Short: "Reserve tenantclaim",
		Run:   tenantClaimReserve,
	}

	tenantClaimConfirm := &cobra.Command{
		Use:   "confirm",
		Short: "Confirm tenantclaim",
		Run:   tenantClaimConfirm,
	}

	tenantClaimDelete := &cobra.Command{
		Use:   "delete",
		Short: "Delete tenantclaim",
		Run:   tenantClaimDelete,
	}

	tenantClaimsRecreate := &cobra.Command{
		Use:   "recreate",
		Short: "Recreate tenantclaims",
		Run:   tenantClaimsRecreate,
	}

	tenantClaimScavenge := &cobra.Command{
		Use:   "scavenge",
		Short: "Scavenge dead tenants created by tenantpool. Dead tenants are the expired trial tenants which do not have any user",
		Run:   tenantClaimScavenge,
	}

	tenantClaimAssign := &cobra.Command{
		Use:   "assign",
		Short: "Assign tenantclaim",
		Run:   tenantClaimAssign,
	}

	tenantClaimCmdArgs.role = "INFRA_ADMIN"
	tenantClaimCmdArgs.dryRun = true
	tenantClaimCmdArgs.updatedBeforeDays = 30
	tenantClaimCmdArgs.pageIndex = 0
	tenantClaimCmdArgs.pageSize = base.MaxRowsLimit

	tenantClaimCreate.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenantclaim ID")
	tenantClaimCreate.Flags().StringVarP(&tenantClaimCmdArgs.registrationID, "registration-id", "r", "", "tenantclaim registration ID")
	tenantClaimCreate.MarkFlagRequired("id")
	tenantClaimCreate.MarkFlagRequired("registration-id")

	tenantClaimUpdate.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenantclaim ID")
	tenantClaimUpdate.Flags().BoolVarP(&tenantClaimCmdArgs.trial, "trial", "t", tenantClaimCmdArgs.trial, "tenantclaim trial")
	tenantClaimUpdate.Flags().IntVarP(&tenantClaimCmdArgs.trialPeriodHours, "expiry", "e", tenantClaimCmdArgs.trialPeriodHours, "tenantclaim trial period")
	tenantClaimUpdate.MarkFlagRequired("id")
	tenantClaimUpdate.MarkFlagRequired("trial")
	tenantClaimUpdate.MarkFlagRequired("expiry")

	tenantClaimGet.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenantclaim ID")
	tenantClaimGet.Flags().StringVarP(&tenantClaimCmdArgs.state, "state", "s", "", "tenantclaim state")
	tenantClaimGet.Flags().StringVarP(&tenantClaimCmdArgs.registrationID, "registration-id", "r", "", "tenantclaim registration ID")
	tenantClaimGet.Flags().StringVarP(&tenantClaimCmdArgs.email, "email", "e", "", "user email address")
	tenantClaimGet.Flags().StringVarP(&tenantClaimCmdArgs.filter, "filter", "f", "", "filter as url encoded conditions")
	tenantClaimGet.Flags().StringArrayVarP(&tenantClaimCmdArgs.orderBy, "order-by", "o", []string{}, "order by")
	tenantClaimGet.Flags().IntVarP(&tenantClaimCmdArgs.pageIndex, "page-index", "j", 0, "start page index")
	tenantClaimGet.Flags().IntVarP(&tenantClaimCmdArgs.pageSize, "page-size", "n", base.MaxRowsLimit, "size of the page")
	tenantClaimGet.Flags().BoolVarP(&tenantClaimCmdArgs.verbose, "verbose", "v", tenantClaimCmdArgs.verbose, "enable/disable verbose output")

	tenantClaimReserve.Flags().StringVarP(&tenantClaimCmdArgs.registrationID, "registration-id", "r", "", "tenantclaim registration ID")
	tenantClaimReserve.MarkFlagRequired("registration-id")

	tenantClaimConfirm.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenantclaim ID")
	tenantClaimConfirm.Flags().StringVarP(&tenantClaimCmdArgs.registrationID, "registration-id", "r", "", "tenantclaim registration ID")
	tenantClaimConfirm.MarkFlagRequired("id")
	tenantClaimConfirm.MarkFlagRequired("registration-id")

	tenantClaimDelete.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenantclaim ID")
	tenantClaimDelete.MarkFlagRequired("id")

	tenantClaimsRecreate.Flags().StringVarP(&tenantClaimCmdArgs.registrationID, "registration-id", "r", "", "tenantclaim registration ID")
	tenantClaimsRecreate.Flags().StringVarP(&tenantClaimCmdArgs.filter, "filter", "f", "", "filter as url encoded conditions")
	tenantClaimsRecreate.MarkFlagRequired("registration-id")

	tenantClaimScavenge.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenant ID")
	tenantClaimScavenge.Flags().IntVarP(&tenantClaimCmdArgs.updatedBeforeDays, "days", "d", 0, "scavenge tenants updated before days")
	tenantClaimScavenge.Flags().BoolVarP(&tenantClaimCmdArgs.dryRun, "dry-run", "r", tenantClaimCmdArgs.dryRun, "dry-run or not")
	tenantClaimScavenge.MarkFlagRequired("days")

	tenantClaimAssign.Flags().StringVarP(&tenantClaimCmdArgs.registrationID, "registration-id", "r", "", "tenantclaim registration ID")
	tenantClaimAssign.Flags().StringVarP(&tenantClaimCmdArgs.id, "id", "i", "", "tenantclaim ID")
	tenantClaimAssign.Flags().StringVarP(&tenantClaimCmdArgs.email, "email", "e", "", "user email address")
	tenantClaimAssign.MarkFlagRequired("registration-id")
	tenantClaimAssign.MarkFlagRequired("id")
	tenantClaimAssign.MarkFlagRequired("email")

	tenantClaimCmd.AddCommand(tenantClaimCreate, tenantClaimUpdate, tenantClaimGet, tenantClaimReserve, tenantClaimConfirm, tenantClaimDelete, tenantClaimsRecreate, tenantClaimScavenge, tenantClaimAssign)

}

func tenantClaimCreate(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.CreateTenantClaim(ctx, &gapi.CreateTenantClaimRequest{RegistrationId: tenantClaimCmdArgs.registrationID, TenantId: tenantClaimCmdArgs.id})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to create tenantclaim. Error: %s"), err.Error())
		}
		bytes, err := base.ConvertToJSONIndent(response.TenantClaim, "  ")
		if err != nil {
			Fatalf(err.Error())
		}
		Infof("Created tenantclaim %s\n", string(bytes))
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimUpdate(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		trialExpiry := base.RoundedNow().Add(time.Hour * time.Duration(tenantClaimCmdArgs.trialPeriodHours))
		protoTime, err := ptypes.TimestampProto(trialExpiry)
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to update tenantclaim. Error: %s"), err.Error())
		}
		response, err := client.UpdateTenantClaim(ctx, &gapi.UpdateTenantClaimRequest{TenantClaim: &gapi.TenantClaim{Id: tenantClaimCmdArgs.id, Trial: tenantClaimCmdArgs.trial, ExpiresAt: protoTime}})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to update tenantclaim. Error: %s"), err.Error())
		}
		Infof("Updated tenantclaim %s\n", response.TenantId)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimGet(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.GetTenantClaims(ctx, &gapi.GetTenantClaimsRequest{
			TenantId:       tenantClaimCmdArgs.id,
			State:          tenantClaimCmdArgs.state,
			RegistrationId: tenantClaimCmdArgs.registrationID,
			Email:          tenantClaimCmdArgs.email,
			Verbose:        tenantClaimCmdArgs.verbose,
			QueryParameter: &gapi.QueryParamater{
				PageIndex: int32(tenantClaimCmdArgs.pageIndex),
				PageSize:  int32(tenantClaimCmdArgs.pageSize),
				Filter:    tenantClaimCmdArgs.filter,
				OrderBy:   tenantClaimCmdArgs.orderBy,
			},
		})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to get tenantclaim(s). Error: %s"), err.Error())
		}
		for _, tenantClaim := range response.TenantClaims {
			bytes, err := base.ConvertToJSONIndent(tenantClaim, "  ")
			if err != nil {
				Fatalf(err.Error())
			}
			Infof("%s\n", string(bytes))
		}
		bytes, err := base.ConvertToJSONIndent(response.PageInfo, "  ")
		if err != nil {
			Fatalf(err.Error())
		}
		Infof("\n%s\n", string(bytes))
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimReserve(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.ReserveTenantClaim(ctx, &gapi.ReserveTenantClaimRequest{RegistrationId: tenantClaimCmdArgs.registrationID})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to reserve tenantclaim(s). Error: %s"), err.Error())
		}
		Infof("Reserved tenantclaim %s\n", response.TenantId)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimConfirm(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.ConfirmTenantClaim(ctx, &gapi.ConfirmTenantClaimRequest{
			RegistrationId: tenantClaimCmdArgs.registrationID,
			TenantId:       tenantClaimCmdArgs.id,
		})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to confirm tenantclaim(s). Error: %s"), err.Error())
		}
		bytes, err := base.ConvertToJSONIndent(response.TenantClaim, "  ")
		if err != nil {
			Fatalf(err.Error())
		}
		Infof("Confirmed tenantclaim %s\n", string(bytes))
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimDelete(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.DeleteTenantClaim(ctx, &gapi.DeleteTenantClaimRequest{TenantId: tenantClaimCmdArgs.id})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to delete tenantclaim. Error: %s"), err.Error())
		}
		Infof("Deleted tenantclaim %s\n", response.TenantId)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimsRecreate(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.RecreateTenantClaims(ctx, &gapi.RecreateTenantClaimsRequest{
			RegistrationId: tenantClaimCmdArgs.registrationID,
			Filter:         tenantClaimCmdArgs.filter,
		})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to recreate tenantclaims. Error: %s"), err.Error())
		}
		Infof("Triggered tenantclaims recreation %s\n", response.RegistrationId)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func tenantClaimScavenge(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	scavenger := NewScavenger()
	defer scavenger.Close()
	updatedBefore := time.Hour * time.Duration(24*tenantClaimCmdArgs.updatedBeforeDays)
	Infof("Dry run set to %t\n", tenantClaimCmdArgs.dryRun)
	tenantsAffected, err := scavenger.Run(ctx, tenantClaimCmdArgs.dryRun, updatedBefore, tenantClaimCmdArgs.id)
	if err != nil {
		Fatalf(err.Error())
	}
	Infof("Scavenged tenant count %d\n", tenantsAffected)
}

func tenantClaimAssign(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.AssignTenantClaim(ctx, &gapi.AssignTenantClaimRequest{
			RegistrationId: tenantClaimCmdArgs.registrationID,
			TenantId:       tenantClaimCmdArgs.id,
			Email:          tenantClaimCmdArgs.email,
		})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to assign tenantclaim. Error: %s"), err.Error())
		}
		Infof("Assigned tenantclaim %s\n", response.TenantId)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

package cmd

import (
	"cloudservices/common/base"
	"cloudservices/common/service"
	"cloudservices/tenantpool/core"
	"cloudservices/tenantpool/model"
	"context"
	"encoding/json"
	"time"

	gapi "cloudservices/tenantpool/generated/grpc"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

// registrationCmd represents the registration command
var registrationCmd = &cobra.Command{
	Use:   "registration",
	Short: "create, get, update or delete registration like \"tenantpool registration get\"",
}

type registrationArgs struct {
	id             string
	description    string
	state          string
	instanceType   string
	filter         string
	orderBy        []string
	edgeCount      int
	minPoolSize    int
	maxPoolSize    int
	maxPendingSize int
	pageIndex      int
	pageSize       int
	trialExpiryHr  int64
	deployApps     bool
}

var regCmdArgs registrationArgs

var regConfigVersion = model.VersionInfo{Version: model.RegConfigV1}

func init() {
	rootCmd.AddCommand(registrationCmd)
	registrationCreate := &cobra.Command{
		Use:   "create",
		Short: "Create registration",
		Run:   registrationCreate,
	}

	registrationGet := &cobra.Command{
		Use:   "get",
		Short: "Get registration(s)",
		Run:   registrationGet,
	}

	registrationUpdate := &cobra.Command{
		Use:   "update",
		Short: "Update registration",
		Run:   registrationUpdate,
	}

	registrationDelete := &cobra.Command{
		Use:   "delete",
		Short: "Delete registration",
		Run:   registrationDelete,
	}

	// Fill defaults
	regCmdArgs.state = core.Active
	regCmdArgs.edgeCount = 1
	regCmdArgs.minPoolSize = 1
	regCmdArgs.maxPoolSize = 2
	regCmdArgs.maxPendingSize = 2
	regCmdArgs.trialExpiryHr = 24 * 15
	regCmdArgs.deployApps = false
	regCmdArgs.pageIndex = 0
	regCmdArgs.pageSize = base.MaxRowsLimit

	registrationCreate.Flags().StringVarP(&regCmdArgs.id, "id", "i", "", "registration ID")
	registrationCreate.Flags().StringVarP(&regCmdArgs.description, "description", "d", "NA", "description")
	registrationCreate.Flags().StringVarP(&regCmdArgs.state, "state", "s", regCmdArgs.state, "registration state")
	registrationCreate.Flags().StringVarP(&regCmdArgs.instanceType, "instance-type", "t", regCmdArgs.instanceType, "instance type")
	registrationCreate.Flags().IntVarP(&regCmdArgs.edgeCount, "edge-count", "n", regCmdArgs.edgeCount, "edge count")
	registrationCreate.Flags().IntVarP(&regCmdArgs.minPoolSize, "min-pool-size", "l", regCmdArgs.minPoolSize, "minimum pool size")
	registrationCreate.Flags().IntVarP(&regCmdArgs.maxPoolSize, "max-pool-size", "u", regCmdArgs.maxPoolSize, "maximum pool size")
	registrationCreate.Flags().IntVarP(&regCmdArgs.maxPendingSize, "max-pending-size", "p", regCmdArgs.maxPendingSize, "maximum pending creation count")
	registrationCreate.Flags().Int64VarP(&regCmdArgs.trialExpiryHr, "trial-expiry", "e", regCmdArgs.trialExpiryHr, "trial expiry hours from now")
	registrationCreate.Flags().BoolVarP(&regCmdArgs.deployApps, "deploy-apps", "a", regCmdArgs.deployApps, "deploy apps and pipelines")
	registrationCreate.MarkFlagRequired("id")

	registrationGet.Flags().StringVarP(&regCmdArgs.id, "id", "i", "", "registration ID")
	registrationGet.Flags().StringVarP(&regCmdArgs.state, "state", "s", regCmdArgs.state, "registration state")
	registrationGet.Flags().StringVarP(&regCmdArgs.filter, "filter", "f", "", "filter as url encoded conditions")
	registrationGet.Flags().StringArrayVarP(&regCmdArgs.orderBy, "order-by", "o", []string{}, "order by")
	registrationGet.Flags().IntVarP(&regCmdArgs.pageIndex, "page-index", "j", 0, "start page index")
	registrationGet.Flags().IntVarP(&regCmdArgs.pageSize, "page-size", "n", base.MaxRowsLimit, "size of the page")

	registrationUpdate.Flags().StringVarP(&regCmdArgs.id, "id", "i", "", "registration ID")
	registrationUpdate.Flags().StringVarP(&regCmdArgs.state, "state", "s", regCmdArgs.state, "registration state")
	registrationUpdate.Flags().IntVarP(&regCmdArgs.minPoolSize, "min-pool-size", "l", regCmdArgs.minPoolSize, "minimum pool size")
	registrationUpdate.Flags().IntVarP(&regCmdArgs.maxPoolSize, "max-pool-size", "u", regCmdArgs.maxPoolSize, "maximum pool size")
	registrationUpdate.Flags().IntVarP(&regCmdArgs.maxPendingSize, "max-pending-size", "p", regCmdArgs.maxPendingSize, "maximum pending creation count")
	registrationUpdate.Flags().Int64VarP(&regCmdArgs.trialExpiryHr, "trial-expiry", "e", regCmdArgs.trialExpiryHr, "trial expiry hours from now")
	registrationUpdate.Flags().BoolVarP(&regCmdArgs.deployApps, "deploy-apps", "a", regCmdArgs.deployApps, "deploy apps and pipelines")
	registrationUpdate.MarkFlagRequired("id")
	registrationUpdate.MarkFlagRequired("state")
	registrationUpdate.MarkFlagRequired("min-pool-size")
	registrationUpdate.MarkFlagRequired("max-pool-size")
	registrationUpdate.MarkFlagRequired("max-pending-size")
	registrationUpdate.MarkFlagRequired("deploy-apps")

	registrationDelete.Flags().StringVarP(&regCmdArgs.id, "id", "i", "", "registration ID")
	registrationDelete.MarkFlagRequired("id")

	registrationCmd.AddCommand(registrationCreate, registrationGet, registrationUpdate, registrationDelete)

}

func registrationCreate(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	regConfig := &model.RegistrationConfigV1{
		VersionInfo:           regConfigVersion,
		EdgeCount:             regCmdArgs.edgeCount,
		InstanceType:          regCmdArgs.instanceType,
		MinTenantPoolSize:     regCmdArgs.minPoolSize,
		MaxTenantPoolSize:     regCmdArgs.maxPoolSize,
		MaxPendingTenantCount: regCmdArgs.maxPendingSize,
		TrialExpiry:           time.Duration(regCmdArgs.trialExpiryHr) * time.Hour,
		DeployApps:            regCmdArgs.deployApps,
	}

	bytes, err := json.Marshal(regConfig)
	if err != nil {
		Fatalf(err.Error())
	}
	registration := &gapi.Registration{Id: regCmdArgs.id, Description: regCmdArgs.description, State: regCmdArgs.state, Config: string(bytes)}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.CreateRegistration(ctx, &gapi.CreateRegistrationRequest{Registration: registration})
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to create registration. Error: %s"), err.Error())
			return err
		}
		Infof("Created registration %s\n", response.Id)
		return nil
	}
	err = service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func registrationGet(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.GetRegistrations(ctx, &gapi.GetRegistrationsRequest{
			Id:    regCmdArgs.id,
			State: regCmdArgs.state,
			QueryParameter: &gapi.QueryParamater{
				PageIndex: int32(regCmdArgs.pageIndex),
				PageSize:  int32(regCmdArgs.pageSize),
				Filter:    regCmdArgs.filter,
				OrderBy:   regCmdArgs.orderBy,
			},
		})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to create registration. Error: %s"), err.Error())
		}
		for _, registration := range response.Registrations {
			bytes, err := base.ConvertToJSONIndent(registration, "  ")
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

func registrationUpdate(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		getResponse, err := client.GetRegistrations(ctx, &gapi.GetRegistrationsRequest{Id: regCmdArgs.id})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to get existing registration for update. Error: %s"), err.Error())
		}
		if len(getResponse.Registrations) == 0 {
			Fatalf(base.PrefixRequestID(ctx, "Failed to get existing registration for update"))
		}
		regConfig := &model.RegistrationConfigV1{}
		registration := getResponse.Registrations[0]
		err = json.Unmarshal([]byte(registration.GetConfig()), regConfig)
		if err != nil {
			Fatalln(base.PrefixRequestID(ctx, "Failed to convert registration config for update. Error %s"), err.Error())
		}
		registration.State = regCmdArgs.state
		regConfig.MinTenantPoolSize = regCmdArgs.minPoolSize
		regConfig.MaxTenantPoolSize = regCmdArgs.maxPoolSize
		regConfig.MaxPendingTenantCount = regCmdArgs.maxPendingSize
		regConfig.TrialExpiry = time.Duration(regCmdArgs.trialExpiryHr) * time.Hour
		regConfig.DeployApps = regCmdArgs.deployApps
		bytes, err := json.Marshal(regConfig)
		if err != nil {
			Fatalln(base.PrefixRequestID(ctx, "Failed to convert registration config for update. Error %s"), err.Error())
		}
		registration.Config = string(bytes)
		response, err := client.UpdateRegistration(ctx, &gapi.UpdateRegistrationRequest{Registration: registration})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to update registration. Error: %s"), err.Error())
		}
		Infof("Updated registration %s\n", response.Id)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

func registrationDelete(cmd *cobra.Command, args []string) {
	if len(regCmdArgs.id) == 0 {
		Fatalf("Registration ID is missing")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.DeleteRegistration(ctx, &gapi.DeleteRegistrationRequest{Id: regCmdArgs.id})
		if err != nil {
			Fatalf(base.PrefixRequestID(ctx, "Failed to delete registration. Error: %s"), err.Error())
		}
		Infof("Deleted registration %s\n", response.Id)
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		Fatalf(err.Error())
	}
}

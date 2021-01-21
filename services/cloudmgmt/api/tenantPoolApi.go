package api

// Cloudmgmt APIs to talk to tenantpool service

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/service"
	gapi "cloudservices/tenantpool/generated/grpc"
	"cloudservices/tenantpool/model"
	"context"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

func (dbAPI *dbObjectModelAPI) CreateRegistration(ctx context.Context, registration *model.Registration) error {
	if registration == nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid registration"))
		return errcode.NewBadRequestError("registration")
	}
	gRegistration := &gapi.Registration{}
	err := base.Convert(registration, gRegistration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert registration %+v"), registration)
		return err
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		_, err := client.CreateRegistration(ctx, &gapi.CreateRegistrationRequest{Registration: gRegistration})
		if err != nil {
			glog.Error(base.PrefixRequestID(ctx, "Failed to create registration. Error: %s"), err.Error())
			return err
		}
		return nil
	}
	err = service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in tenantpool service call. Error: %s"), err.Error())
	}
	return err
}

func (dbAPI *dbObjectModelAPI) GetRegistration(ctx context.Context, registrationID string) (*model.Registration, error) {
	if len(registrationID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid registration ID"))
		return nil, errcode.NewBadRequestError("registrationID")
	}
	registration := &model.Registration{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.GetRegistrations(ctx, &gapi.GetRegistrationsRequest{Id: registrationID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration %s. Error: %s"), registrationID, err.Error())
			return err
		}
		if len(response.Registrations) != 1 {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to find registration %s"), registrationID)
			return errcode.NewRecordNotFoundError(registrationID)
		}
		err = base.Convert(response.Registrations[0], registration)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert registration %s"), registrationID)
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in tenantpool service call. Error: %s"), err.Error())
	}
	return registration, err
}

func (dbAPI *dbObjectModelAPI) CreateTenantClaim(ctx context.Context, registrationID, tenantID, email string) (*model.TenantClaim, error) {
	if len(registrationID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid registration ID"))
		return nil, errcode.NewBadRequestError("registrationID")
	}
	if len(tenantID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid tenant ID"))
		return nil, errcode.NewBadRequestError("tenantID")
	}
	tenantClaim := &model.TenantClaim{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		metadata := &gapi.Metadata{Email: email, RegistrationId: registrationID, TenantId: tenantID}
		response, err := client.CreateTenantClaim(ctx, &gapi.CreateTenantClaimRequest{RegistrationId: registrationID, TenantId: tenantID, Metadata: metadata})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenantClaim %s. Error: %s"), tenantID, err.Error())
			return err
		}
		err = base.Convert(response.TenantClaim, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenantClaim %s"), tenantID)
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in tenantpool service call. Error: %s"), err.Error())
	}
	return tenantClaim, err
}

func (dbAPI *dbObjectModelAPI) GetTenantClaim(ctx context.Context, tenantID string) (*model.TenantClaim, error) {
	if len(tenantID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid tenant ID"))
		return nil, errcode.NewBadRequestError("tenantID")
	}
	tenantClaim := &model.TenantClaim{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		response, err := client.GetTenantClaims(ctx, &gapi.GetTenantClaimsRequest{TenantId: tenantID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenantClaim %s. Error: %s"), tenantID, err.Error())
			return err
		}
		if len(response.TenantClaims) != 1 {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to find tenantClaim %s"), tenantID)
			return errcode.NewRecordNotFoundError(tenantID)
		}
		err = base.Convert(response.TenantClaims[0], tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenantClaim %s"), tenantID)
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in tenantpool service call. Error: %s"), err.Error())
	}
	return tenantClaim, err
}

func (dbAPI *dbObjectModelAPI) ReserveTenantClaim(ctx context.Context, registrationID, email string) (string, error) {
	var reservedTenantID string
	if len(registrationID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid registration ID"))
		return reservedTenantID, errcode.NewBadRequestError("registrationID")
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		metadata := &gapi.Metadata{Email: email, RegistrationId: registrationID}
		response, err := client.ReserveTenantClaim(ctx, &gapi.ReserveTenantClaimRequest{RegistrationId: registrationID, Metadata: metadata})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to reserve tenantclaim(s). Error: %s"), err.Error())
			return err
		}
		reservedTenantID = response.TenantId
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in tenantpool service call. Error: %s"), err.Error())
	}
	return reservedTenantID, err
}

func (dbAPI *dbObjectModelAPI) ConfirmTenantClaim(ctx context.Context, registrationID, tenantID, email string) (*model.TenantClaim, error) {
	tenantClaim := &model.TenantClaim{}
	if len(registrationID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid registration ID"))
		return tenantClaim, errcode.NewBadRequestError("registrationID")
	}
	if len(tenantID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid tenant ID"))
		return tenantClaim, errcode.NewBadRequestError("tenantID")
	}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewTenantPoolServiceClient(conn)
		metadata := &gapi.Metadata{Email: email, RegistrationId: registrationID, TenantId: tenantID}
		response, err := client.ConfirmTenantClaim(ctx, &gapi.ConfirmTenantClaimRequest{RegistrationId: registrationID, TenantId: tenantID, Metadata: metadata})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to confirm tenantclaim(s) for candidate %s. Error: %s"), tenantID, err.Error())
			return err
		}
		err = base.Convert(response.TenantClaim, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenantClaim %s"), tenantID)
			return err
		}
		return nil
	}
	err := service.CallClient(ctx, service.TenantPoolService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in tenantpool service call. Error: %s"), err.Error())
	}
	return tenantClaim, err
}

package api

// Server side handlers for generated gRPC methods

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	cmodel "cloudservices/common/model"
	"cloudservices/tenantpool/core"
	gapi "cloudservices/tenantpool/generated/grpc"
	"cloudservices/tenantpool/model"
	"context"

	"github.com/golang/glog"
)

// CreateRegistration creates a registration with the config
func (server *apiServer) CreateRegistration(ctx context.Context, in *gapi.CreateRegistrationRequest) (*gapi.CreateRegistrationResponse, error) {
	registration := &model.Registration{}
	err := base.Convert(in.Registration, registration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), in.Registration, err.Error())
		return nil, err
	}
	registration, err = server.tenantPoolManager.GetBookKeeper().CreateRegistration(ctx, registration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke CreateRegistration %+v. Error: %s"), registration, err.Error())
		return nil, err
	}
	return &gapi.CreateRegistrationResponse{Id: registration.ID}, nil
}

// GetRegistrations gets the registrations with the ID and state filter
func (server *apiServer) GetRegistrations(ctx context.Context, in *gapi.GetRegistrationsRequest) (*gapi.GetRegistrationsResponse, error) {
	states := []string{}
	if len(in.State) != 0 {
		states = append(states, in.State)
	}
	var queryParam *cmodel.EntitiesQueryParam
	if in.QueryParameter != nil {
		queryParam = &cmodel.EntitiesQueryParam{}
		err := base.Convert(in.QueryParameter, queryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert query parameter. Error: %s"), err.Error())
			return nil, err
		}
	}
	registrations, pageResponse, err := server.tenantPoolManager.GetBookKeeper().GetRegistrations(ctx, in.Id, states, queryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke GetRegistrations %s. Error: %s"), in.Id, err.Error())
	}
	response := &gapi.GetRegistrationsResponse{PageInfo: &gapi.PageInfo{
		PageIndex:   int32(pageResponse.PageIndex),
		PageSize:    int32(pageResponse.PageSize),
		TotalCount:  int32(pageResponse.TotalCount),
		OrderByKeys: core.GetRegistrationOrderByKeys(),
	}}
	for _, registration := range registrations {
		// Golang JSON does not ignore Version field and fails to convert
		registration.Version = 0
		gRegistration := &gapi.Registration{}
		err := base.Convert(registration, gRegistration)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), registration, err.Error())
			return nil, err
		}
		response.Registrations = append(response.Registrations, gRegistration)
	}
	return response, nil
}

// UpdateRegistration updates the registration
func (server *apiServer) UpdateRegistration(ctx context.Context, in *gapi.UpdateRegistrationRequest) (*gapi.UpdateRegistrationResponse, error) {
	registration := &model.Registration{}
	err := base.Convert(in.Registration, registration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), in.Registration, err.Error())
		return nil, err
	}
	registration, err = server.tenantPoolManager.GetBookKeeper().UpdateRegistration(ctx, registration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke UpdateRegistration %+v. Error: %s"), registration, err.Error())
		return nil, err
	}
	return &gapi.UpdateRegistrationResponse{Id: registration.ID}, nil
}

// DeleteRegistration triggers deletion of the registration.
// All the resources like tenant, edges are clean up before the registration is deleted
func (server *apiServer) DeleteRegistration(ctx context.Context, in *gapi.DeleteRegistrationRequest) (*gapi.DeleteRegistrationResponse, error) {
	// Mark the state to Deleting such that the cleaning is graceful
	err := server.tenantPoolManager.GetBookKeeper().UpdateRegistrationState(ctx, in.Id, core.Deleting)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke UpdateRegistrationState %s. Error: %s"), in.Id, err.Error())
		return nil, err
	}
	return &gapi.DeleteRegistrationResponse{Id: in.Id}, nil
}

// CreateTenantClaim creates a tenant claim with existing tenant ID
func (server *apiServer) CreateTenantClaim(ctx context.Context, in *gapi.CreateTenantClaimRequest) (*gapi.CreateTenantClaimResponse, error) {
	tenantClaim, err := server.tenantPoolManager.CreateTenantClaim(ctx, in.RegistrationId, in.TenantId)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke CreateTenantClaim %s. Error: %s"), in.TenantId, err.Error())
		return nil, err
	}
	zeroTenantClaimVersions(tenantClaim)
	gTenantClaim := &gapi.TenantClaim{}
	err = base.Convert(tenantClaim, gTenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), tenantClaim, err.Error())
		return nil, err
	}
	return &gapi.CreateTenantClaimResponse{TenantClaim: gTenantClaim}, nil
}

// UpdateTenantClaim updates a tenant claim with existing tenant ID
func (server *apiServer) UpdateTenantClaim(ctx context.Context, in *gapi.UpdateTenantClaimRequest) (*gapi.UpdateTenantClaimResponse, error) {
	tenantClaim := &model.TenantClaim{}
	err := base.Convert(in.TenantClaim, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), in.TenantClaim, err.Error())
		return nil, err
	}
	err = server.tenantPoolManager.GetBookKeeper().UpdateTenantClaim(ctx, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke UpdateTenantClaim %+v. Error: %s"), in.TenantClaim, err.Error())
		return nil, err
	}
	return &gapi.UpdateTenantClaimResponse{TenantId: tenantClaim.ID}, nil
}

// GetTenantClaims lists all the tenantClaim objects matching the filters in the request
func (server *apiServer) GetTenantClaims(ctx context.Context, in *gapi.GetTenantClaimsRequest) (*gapi.GetTenantClaimsResponse, error) {
	states := []string{}
	if len(in.State) != 0 {
		states = append(states, in.State)
	}
	response := &gapi.GetTenantClaimsResponse{PageInfo: &gapi.PageInfo{OrderByKeys: core.GetTenantClaimOrderByKeys()}}
	var queryParam *cmodel.EntitiesQueryParam
	if in.QueryParameter != nil {
		queryParam = &cmodel.EntitiesQueryParam{}
		err := base.Convert(in.QueryParameter, queryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert query parameter. Error: %s"), err.Error())
			return nil, err
		}
	}
	if len(in.TenantId) == 0 && len(in.Email) > 0 {
		// Passing tenant ID overrides email
		gUser, err := core.GetUserByEmail(ctx, in.Email)
		if err != nil {
			if _, ok := err.(*errcode.RecordNotFoundError); ok {
				return response, nil
			}
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to user by email %s. Error: %s"), in.Email, err.Error())
			return nil, err
		}
		in.TenantId = gUser.TenantId
	}
	pageResponse, err := server.tenantPoolManager.GetBookKeeper().ScanTenantClaims(ctx, in.RegistrationId, in.TenantId, states, queryParam, func(registration *model.Registration, tenantClaim *model.TenantClaim) error {
		gTenantClaim := &gapi.TenantClaim{}
		zeroTenantClaimVersions(tenantClaim)
		err := base.Convert(tenantClaim, gTenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), tenantClaim, err.Error())
			return err
		}
		if in.Verbose {
			server.populateEdgeDetails(ctx, gTenantClaim)
		}
		response.TenantClaims = append(response.TenantClaims, gTenantClaim)
		return nil
	})
	if err != nil {
		return nil, err
	}
	response.PageInfo.PageIndex = int32(pageResponse.PageIndex)
	response.PageInfo.PageSize = int32(pageResponse.PageSize)
	response.PageInfo.TotalCount = int32(pageResponse.TotalCount)
	return response, nil
}

// ReserveTenantClaim reserves a tenant and returns the tentative tenant ID.
// Once the tenant is reserved, it cannot be reserved in subsequent calls till the reservation expires in 30 minutes
func (server *apiServer) ReserveTenantClaim(ctx context.Context, in *gapi.ReserveTenantClaimRequest) (*gapi.ReserveTenantClaimResponse, error) {
	var err error
	var tenantClaim *model.TenantClaim
	defer func() {
		server.createAuditLog(ctx, err, in.GetMetadata(), func(auditLog *model.AuditLog) error {
			auditLog.RegistrationID = in.RegistrationId
			auditLog.Action = model.AuditLogReserveTenantAction
			return nil
		})
	}()
	tenantClaim, err = server.tenantPoolManager.GetBookKeeper().ReserveTenantClaim(ctx, in.RegistrationId)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke ReserveTenantClaim %s. Error: %s"), in.RegistrationId, err.Error())
		return nil, err
	}
	return &gapi.ReserveTenantClaimResponse{TenantId: tenantClaim.ID}, nil
}

// ConfirmTenantClaim confirms the tentative tenant ID reserved previously.
// The users are created and added to this tenant
func (server *apiServer) ConfirmTenantClaim(ctx context.Context, in *gapi.ConfirmTenantClaimRequest) (*gapi.ConfirmTenantClaimResponse, error) {
	var err error
	var tenantClaim *model.TenantClaim
	defer func() {
		server.createAuditLog(ctx, err, in.GetMetadata(), func(auditLog *model.AuditLog) error {
			auditLog.TenantID = in.TenantId
			auditLog.RegistrationID = in.RegistrationId
			auditLog.Action = model.AuditLogConfirmTenantAction
			return nil
		})
	}()
	tenantClaim, err = server.tenantPoolManager.GetBookKeeper().ConfirmTenantClaim(ctx, in.RegistrationId, in.TenantId)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke ConfirmTenantClaim - registration %s, tenant %s. Error: %s"), in.RegistrationId, in.TenantId, err.Error())
		return nil, err
	}
	gTenantClaim := &gapi.TenantClaim{}
	zeroTenantClaimVersions(tenantClaim)
	err = base.Convert(tenantClaim, gTenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), tenantClaim, err.Error())
		return nil, err
	}
	return &gapi.ConfirmTenantClaimResponse{TenantClaim: gTenantClaim}, nil
}

// DeleteTenantClaim triggers the tenant deletion workflow.
// The edges are deleted before the tenant is deleted
func (server *apiServer) DeleteTenantClaim(ctx context.Context, in *gapi.DeleteTenantClaimRequest) (*gapi.DeleteTenantClaimResponse, error) {
	var err error
	defer func() {
		server.createAuditLog(ctx, err, in.GetMetadata(), func(auditLog *model.AuditLog) error {
			auditLog.TenantID = in.TenantId
			auditLog.Action = model.AuditLogDeleteTenantAction
			return nil
		})
	}()
	// Mark the state to Deleting such that the cleaning is graceful
	err = server.tenantPoolManager.GetBookKeeper().TriggerDeleteTenantClaim(ctx, in.TenantId)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke UpdateTenantState for tenant %s. Error: %s"), in.TenantId, err.Error())
		return nil, err
	}
	return &gapi.DeleteTenantClaimResponse{TenantId: in.TenantId}, nil
}

// RecreateTenantClaims recreates the tenantClaims in AVAILABLE state
func (server *apiServer) RecreateTenantClaims(ctx context.Context, in *gapi.RecreateTenantClaimsRequest) (*gapi.RecreateTenantClaimsResponse, error) {
	var err error
	defer func() {
		server.createAuditLog(ctx, err, in.GetMetadata(), func(auditLog *model.AuditLog) error {
			auditLog.Action = model.RecreateTenantClaimsAction
			return nil
		})
	}()
	queryParam := &base.FilterAndOrderByParam{Filter: in.Filter}
	// Mark the states to Deleting for Available records
	err = server.tenantPoolManager.GetBookKeeper().TriggerDeleteTenantClaims(ctx, in.RegistrationId, core.Available, queryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke RecreateTenantClaims for registration %s. Error: %s"), in.RegistrationId, err.Error())
		return nil, err
	}
	return &gapi.RecreateTenantClaimsResponse{RegistrationId: in.RegistrationId}, nil
}

// AssignTenantClaim assigns a tenant claim to an email
func (server *apiServer) AssignTenantClaim(ctx context.Context, in *gapi.AssignTenantClaimRequest) (*gapi.AssignTenantClaimResponse, error) {
	var err error
	defer func() {
		server.createAuditLog(ctx, err, in.GetMetadata(), func(auditLog *model.AuditLog) error {
			auditLog.RegistrationID = in.RegistrationId
			auditLog.TenantID = in.TenantId
			auditLog.Email = in.Email
			auditLog.Action = model.AuditLogAssignTenantAction
			return nil
		})
	}()
	// Assignt the tenant ID to the email
	err = server.tenantPoolManager.GetBookKeeper().AssignTenantClaim(ctx, in.RegistrationId, in.TenantId, in.Email)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke AssignTenantClaim for registration ID %s, tenant ID %s and email %s. Error: %s"), in.RegistrationId, in.TenantId, in.Email, err.Error())
		return nil, err
	}
	return &gapi.AssignTenantClaimResponse{TenantId: in.TenantId}, nil
}

func (server *apiServer) createAuditLog(ctx context.Context, apiError error, metadata *gapi.Metadata, callback func(*model.AuditLog) error) error {
	if metadata == nil {
		return nil
	}
	auditLog := &model.AuditLog{}
	err := base.Convert(metadata, auditLog)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data for %+v. Error: %s"), metadata, err.Error())
		return err
	}
	if callback != nil {
		err = callback(auditLog)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in callback for %+v. Error: %s"), auditLog, err.Error())
			return err
		}
	}
	if len(auditLog.Actor) == 0 {
		auditLog.Actor = model.AuditLogUserActor
	}
	err = server.tenantPoolManager.GetAuditLogManager().CreateAuditLogHelper(ctx, apiError, auditLog, true)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create auditlog for %+v. Error: %s"), auditLog, err.Error())
	}
	return err
}

func (server *apiServer) populateEdgeDetails(ctx context.Context, tenantClaim *gapi.TenantClaim) error {
	for _, edgeContext := range tenantClaim.EdgeContexts {
		response, err := server.tenantPoolManager.GetEdgeProvisioner().DescribeEdge(ctx, tenantClaim.Id, edgeContext.Id)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Failed to get edge info for context %s. Error: %s"), edgeContext.Id, err.Error())
			continue
		}
		data, err := base.ConvertToJSONIndent(response, " ")
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Failed to marshal data %+v. Error: %s"), response, err.Error())
			continue
		}
		edgeContext.Details = string(data)
	}
	return nil
}

// Golang JSON does not ignore Version field and fails to convert
func zeroTenantClaimVersions(tenantClaim *model.TenantClaim) {
	tenantClaim.Version = 0
	for _, edgeContext := range tenantClaim.EdgeContexts {
		edgeContext.Version = 0
	}
}

package core

// BookKeeper persists the registration, tenant and edge records in DB
// It also invokes account service.
import (
	gapi "cloudservices/account/generated/grpc"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	cmodel "cloudservices/common/model"
	"cloudservices/tenantpool/model"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang/glog"
)

const (
	// States of the tenants and edges
	Active    = "ACTIVE"
	InActive  = "INACTIVE"
	Reserved  = "RESERVED"
	Available = "AVAILABLE"
	Assigned  = "ASSIGNED"
	Creating  = "CREATING"
	Deleting  = "DELETING"
	Created   = "CREATED"
	Failed    = "FAILED"
	Deleted   = "DELETED"

	// Name of this entity
	entityTypeTenantClaim  = "tenantclaim"
	entityTypeRegistration = "registration"

	// Default order by
	orderByUpdatedAt = "order by updated_at desc"

	// SQL statements for registration model
	selectRegistrationTemplateQuery = "select *, count(*) OVER() as total_count from tps_registration_model where (:id = '' or id = :id) and (:all_states = true or state in (:states)) %s"
	insertRegistrationQuery         = "insert into tps_registration_model(id, description, config, state, created_at, updated_at) values (:id, :description, :config, :state, :created_at, :updated_at)"
	updateRegistationQuery          = "update tps_registration_model set state = :state, description = :description, config = :config, updated_at = :updated_at where id = :id and (:expected_version = 0 or version = :expected_version) and (:expected_state = '' or state = :expected_state) and (:unexpected_state = '' or state != :unexpected_state)"
	updateRegistrationStateQuery    = "update tps_registration_model set state = :state, updated_at = :updated_at where id = :id and (:expected_version = 0 or version = :expected_version) and (:expected_state = '' or state = :expected_state) and (:unexpected_state = '' or state != :unexpected_state)"
	deleteRegistrationQuery         = "delete from tps_registration_model where id = :id"

	// SQL statements for tenant pool model
	selectTenantPoolQuery               = "select *, count(*) OVER() as total_count from tps_tenant_pool_model where registration_id in (:registration_ids) and (:id = '' or id = :id) and (:all_states = true or state in (:states)) order by updated_at desc"
	selectTenantPoolTemplateQuery       = "select *, count(*) OVER() as total_count from tps_tenant_pool_model where registration_id in (:registration_ids) and (:id = '' or id = :id) and (:all_states = true or state in (:states)) %s"
	createTenantPoolQuery               = "insert into tps_tenant_pool_model(id, registration_id, state, system_user, system_password, trial, resources, created_at, updated_at) values(:id, :registration_id, :state, :system_user, :system_password, :trial, :resources, :created_at, :updated_at)"
	updateTenantPoolQuery               = "update tps_tenant_pool_model set state = :state, resources = :resources, assigned_at = :assigned_at, expires_at = :expires_at, updated_at = :updated_at where id = :id and (:expected_version = 0 or version = :expected_version) and (:expected_state = '' or state = :expected_state) and (:unexpected_state = '' or state != :unexpected_state)"
	updateTenantPoolStateQuery          = "update tps_tenant_pool_model set state = :state, assigned_at = :assigned_at, expires_at = :expires_at, updated_at = :updated_at where id = :id and (:expected_version = 0 or version = :expected_version) and (:expected_state = '' or state = :expected_state) and (:unexpected_state = '' or state != :unexpected_state)"
	updateTenantPoolTrialQuery          = "update tps_tenant_pool_model set trial = :trial, expires_at = :expires_at, updated_at = :updated_at where id = :id"
	updateTenantPoolStatesTemplateQuery = "update tps_tenant_pool_model set state = :state, updated_at = :updated_at where registration_id = :registration_id and (:expected_version = 0 or version = :expected_version) and (:expected_state = '' or state = :expected_state) and (:unexpected_state = '' or state != :unexpected_state) %s"
	purgeTenantPoolQuery                = "delete from tps_tenant_pool_model where registration_id = :registration_id"

	// SQL statements for edge context model
	selectEdgeContextQuery = "select * from tps_edge_context_model where tenant_id = :tenant_id"
	createEdgeContextQuery = "insert into tps_edge_context_model(id, tenant_id, edge_id, state, type, created_at, updated_at) values(:id, :tenant_id, :edge_id, :state, :type, :created_at, :updated_at)"

	// SQL statement for renaming node serial number
	renameNodeSerialNumberQuery = "update edge_device_model set serial_number=concat(serial_number, '.%d.ntnx-del'), updated_at='%s' where tenant_id='%s' and serial_number not like '%%.ntnx-del'"
)

var orderByHelper = base.NewOrderByHelper()

func init() {
	orderByHelper.Setup(entityTypeTenantClaim, []string{"registration_id", "state", "assigned_at", "created_at", "expires_at", "updated_at", "trial"})
	orderByHelper.Setup(entityTypeRegistration, []string{"state", "created_at", "updated_at"})
}

// BookKeeper keeps registration tenant and edge information in the DB
type BookKeeper struct {
	*base.DBObjectModelAPI
}

// RegistrationDBO is the DB model for the registration_model table
type RegistrationDBO struct {
	ID          string    `json:"id" db:"id"`
	Description string    `json:"description" db:"description"`
	Config      string    `json:"config" db:"config"`
	State       string    `json:"state" db:"state"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
	Version     int64     `json:"version,omitempty" db:"version"`
	TotalCount  int       `json:"totalCount" db:"total_count"`
}

// TenantPoolDBO is the DB model for tenant_pool_model table
type TenantPoolDBO struct {
	ID             string           `json:"id" db:"id"`
	State          string           `json:"state" db:"state"`
	RegistrationID string           `json:"registrationId" db:"registration_id"`
	SystemUser     string           `json:"systemUser" db:"system_user"`
	SystemPassword string           `json:"systemPassword" db:"system_password"`
	Trial          bool             `json:"trial" db:"trial"`
	Resources      *json.RawMessage `json:"resources,omitempty" db:"resources"`
	AssignedAt     *time.Time       `json:"assignedAt,omitempty" db:"assigned_at"`
	ExpiresAt      *time.Time       `json:"expiresAt,omitempty" db:"expires_at"`
	CreatedAt      time.Time        `json:"createdAt" db:"created_at"`
	UpdatedAt      time.Time        `json:"updatedAt" db:"updated_at"`
	Version        int64            `json:"version,omitempty" db:"version"`
	TotalCount     int              `json:"totalCount" db:"total_count"`
}

// EdgeContextDBO is the DB model for tenant_pool_model table
type EdgeContextDBO struct {
	ID        string    `json:"id" db:"id"`
	TenantID  string    `json:"tenantId" db:"tenant_id"`
	EdgeID    *string   `json:"edgeId" db:"edge_id"`
	State     string    `json:"state" db:"state"`
	Type      string    `json:"type" db:"type"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	Version   int64     `json:"version,omitempty" db:"version"`
}

type registrationSelectQueryParam struct {
	RegistrationDBO
	AllStates bool     `json:"allStates" db:"all_states"`
	States    []string `json:"states" db:"states"`
}

type registrationUpdateParam struct {
	RegistrationDBO
	UnexpectedState string `json:"unexpectedState" db:"unexpected_state"`
	ExpectedState   string `json:"expectedState" db:"expected_state"`
	ExpectedVersion int64  `json:"expectedVersion" db:"expected_version"`
}

type tenantPoolSelectQueryParam struct {
	ID              string   `json:"id" db:"id"`
	AllStates       bool     `json:"allStates" db:"all_states"`
	States          []string `json:"states" db:"states"`
	RegistrationIDs []string `json:"registrationIds" db:"registration_ids"`
}

type tenantPoolUpdateQueryParam struct {
	TenantPoolDBO
	UnexpectedState string `json:"unexpectedState" db:"unexpected_state"`
	ExpectedState   string `json:"expectedState" db:"expected_state"`
	ExpectedVersion int64  `json:"expectedVersion" db:"expected_version"`
}

func deleteOrUpdateOk(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return errcode.NewInternalDatabaseError(err.Error())
	}
	if rows == 0 {
		return errcode.NewBadRequestExError("version", "Delete/Update failed")
	}
	return nil
}

func getSystemUser(tenantID string) *gapi.User {
	return &gapi.User{
		TenantId: tenantID,
		Name:     "Sherlock Bott Admin",
		Email:    fmt.Sprintf("%s@ntnxsherlock.com", tenantID),
		Password: base.GenerateStrongPassword(),
		Role:     "INFRA_ADMIN",
	}
}

func getUser(tenantID, email string) *gapi.User {
	return &gapi.User{
		TenantId: tenantID,
		Name:     "Trial User",
		Email:    email,
		Password: base.GenerateStrongPassword(),
		Role:     "INFRA_ADMIN",
	}
}

// GetRegistrationOrderByKeys returns the fields for filter and order by for registration table
func GetRegistrationOrderByKeys() []string {
	return orderByHelper.GetOrderByKeys(entityTypeRegistration)
}

// GetTenantClaimOrderByKeys returns the fields for filter and order by for tenantclaim table
func GetTenantClaimOrderByKeys() []string {
	return orderByHelper.GetOrderByKeys(entityTypeTenantClaim)
}

// GetRenameSerialNumberQuery returns the query to rename serial numbers of nodes
func GetRenameSerialNumberQuery(tenantID string, updatedAt time.Time) string {
	unixTime := updatedAt.Unix()
	updatedTimestamp := updatedAt.Format("2006-01-02 15:04:05.999")
	return fmt.Sprintf(renameNodeSerialNumberQuery, unixTime, updatedTimestamp, tenantID)
}

// CreateRegistration creates the registration
func (keeper *BookKeeper) CreateRegistration(ctx context.Context, registration *model.Registration) (*model.Registration, error) {
	err := model.ValidateRegistration(ctx, registration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to validate registration %+v. Error: %s"), registration, err.Error())
		return nil, err
	}
	regDBO := &RegistrationDBO{}
	err = base.Convert(registration, regDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for registration %+v. Error: %s"), registration, err.Error())
		return nil, err
	}
	now := base.RoundedNow()
	regDBO.CreatedAt = now
	regDBO.UpdatedAt = now
	_, err = keeper.NamedExec(ctx, insertRegistrationQuery, regDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create registration %+v. Error: %s"), registration, err.Error())
		return nil, errcode.TranslateDatabaseError("registration", err)
	}
	registration.CreatedAt = now
	registration.UpdatedAt = now
	return registration, nil
}

// GetRegistrations lists all the registrations with the optional ID or the state or both (ANDed)
func (keeper *BookKeeper) GetRegistrations(ctx context.Context, id string, states []string, queryFilterParam base.QueryParameter) ([]*model.Registration, *cmodel.PagedListResponsePayload, error) {
	queryParam := registrationSelectQueryParam{States: states}
	if len(states) == 0 {
		queryParam.AllStates = true
		// Sqlx does not allow empty
		queryParam.States = []string{"dummy"}
	}
	queryParam.ID = id
	registrations := []*model.Registration{}
	query, pageQueryParam, err := orderByHelper.BuildPagedQuery(entityTypeRegistration, selectRegistrationTemplateQuery, queryFilterParam, orderByUpdatedAt)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to build query %s. Error: %s"), selectRegistrationTemplateQuery, err.Error())
		return nil, nil, err
	}
	pageResponse := &cmodel.PagedListResponsePayload{PageIndex: pageQueryParam.GetPageIndex(), PageSize: pageQueryParam.GetPageSize()}
	_, err = keeper.NotPagedQueryIn(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		regDBO := dbObjPtr.(*registrationSelectQueryParam).RegistrationDBO
		registration := &model.Registration{}
		err := base.Convert(regDBO, registration)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for registration %+v. Error: %s"), regDBO, err.Error())
			return err
		}
		pageResponse.TotalCount = regDBO.TotalCount
		registrations = append(registrations, registration)
		return nil
	}, query, queryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration with param %+v. Error: %s"), queryParam, err.Error())
		return nil, nil, err
	}
	return registrations, pageResponse, err
}

// UpdateRegistration updates the registration
func (keeper *BookKeeper) UpdateRegistration(ctx context.Context, registration *model.Registration) (*model.Registration, error) {
	registrations, pageQueryParam, err := keeper.GetRegistrations(ctx, registration.ID, []string{Active, InActive}, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration %s to update. Error: %s"), registration.ID, err.Error())
		return nil, err
	}
	if pageQueryParam.TotalCount != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration %s to update"), registration.ID)
		return nil, errcode.NewRecordNotFoundError(registration.ID)
	}
	err = model.ValidateRegistrationUpdate(ctx, registrations[0], registration)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to validate registration %+v. Error: %s"), registration, err.Error())
		return nil, err
	}
	regDBO := &RegistrationDBO{}
	err = base.Convert(registration, regDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for registration %+v. Error: %s"), registration, err.Error())
		return nil, err
	}
	now := base.RoundedNow()
	regDBO.UpdatedAt = now
	updateParam := &registrationUpdateParam{RegistrationDBO: *regDBO, UnexpectedState: Deleting, ExpectedVersion: registrations[0].Version}
	result, err := keeper.NamedExec(ctx, updateRegistationQuery, updateParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update registration %+v. Error: %s"), regDBO, err.Error())
		return nil, errcode.TranslateDatabaseError("registration", err)
	}
	err = deleteOrUpdateOk(result)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update registration %+v. Error: %s"), regDBO, err.Error())
		return nil, err
	}
	registration.UpdatedAt = now
	return registration, nil
}

// UpdateRegistrationState updates only the state
func (keeper *BookKeeper) UpdateRegistrationState(ctx context.Context, id, state string) error {
	now := base.RoundedNow()
	regDBO := RegistrationDBO{ID: id, State: state, UpdatedAt: now}
	updateParam := &registrationUpdateParam{RegistrationDBO: regDBO, UnexpectedState: Deleting}
	result, err := keeper.NamedExec(ctx, updateRegistrationStateQuery, updateParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update registration %+v. Error: %s"), regDBO, err.Error())
		return errcode.TranslateDatabaseError("registration", err)
	}
	err = deleteOrUpdateOk(result)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update registration %+v. Error: %s"), regDBO, err.Error())
		return err
	}
	return nil
}

// DeleteRegistration marks the registration for deletion
func (keeper *BookKeeper) DeleteRegistration(ctx context.Context, id string) error {
	result, err := keeper.Delete(ctx, "tps_registration_model", map[string]interface{}{"id": id})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete registration %s. Error: %s"), id, err.Error())
		return err
	}
	err = deleteOrUpdateOk(result)
	if err != nil {
		// Record must have been deleted
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete registration %s. Error: %s"), id, err.Error())
	}
	return nil
}

// CreateTenantClaim is called to create a tenant for the first time without any edges.
// The callback is invoked once the tenant is created in the account service
// that can be used to invoke Bott service to create edges.
func (keeper *BookKeeper) CreateTenantClaim(ctx context.Context, registrationID, tenantID string, callback model.TenantClaimProcessor) (*model.TenantClaim, error) {
	registrations, pageQueryParam, err := keeper.GetRegistrations(ctx, registrationID, []string{Active}, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s to add a tenant. Error: %s"), registrationID, err.Error())
		return nil, err
	}
	if pageQueryParam.TotalCount != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s to add a tenant"), registrationID)
		return nil, errcode.NewRecordNotFoundError(registrationID)
	}
	var tenant *gapi.Tenant
	isTrial := false
	if len(tenantID) > 0 {
		tenant, err = GetTenant(ctx, tenantID)
		if err != nil {
			if _, ok := err.(*errcode.RecordNotFoundError); !ok {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenant %s for registration %s. Error: %s"), tenantID, registrationID, err.Error())
				return nil, err
			}
			tenant = nil
			err = nil
		}
	} else {
		tenantID = base.GetUUID()
	}
	if tenant == nil {
		// TODO modify when billing/subscription service is ready
		// Existing tenants are marked non-trial for now
		isTrial = true
		tenant = &gapi.Tenant{Id: tenantID, Name: "Trial Tenant", Description: fmt.Sprintf("Trial tenant for %s", registrationID)}
		// Call to create tenant in account service
		tenant, err = CreateTenant(ctx, tenant)
		if err != nil {
			if _, ok := err.(*errcode.DatabaseDuplicateError); !ok {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenant %s for registration %s. Error: %s"), tenantID, registrationID, err.Error())
				return nil, err
			}
			glog.Errorf(base.PrefixRequestID(ctx, "Tenant %s already exist for registration %s. Error: %s"), tenantID, registrationID, err.Error())
			err = nil
		}
	}
	defer func() {
		if err != nil && isTrial {
			// Best effort to roll back
			delErr := DeleteTenantIfPossible(ctx, tenant.Id)
			if delErr != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Failed to rollback for tenant %s for registration %s. Error: %s"), tenant.Id, registrationID, delErr.Error())
			}
		}
	}()
	user := getSystemUser(tenant.Id)
	user, err = CreateUser(ctx, user)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create system user for tenant %s and registration %s. Error: %s"), tenant.Id, registrationID, err.Error())
		return nil, err
	}
	defer func() {
		if err != nil {
			// Best effort to roll back
			delErr := DeleteUser(ctx, user.TenantId, user.Id)
			if delErr != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Failed to rollback for system user for tenant %s and registration %s. Error: %s"), tenant.Id, registrationID, delErr.Error())
			}
		}
	}()
	now := base.RoundedNow()
	// First put in DB to avoid unlimited creation due to DB failure
	tenantPoolDBO := &TenantPoolDBO{
		ID:             tenant.Id,
		State:          Creating,
		RegistrationID: registrations[0].ID,
		SystemUser:     user.Email,
		SystemPassword: user.Password,
		Trial:          isTrial,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	tenantClaim := &model.TenantClaim{}
	err = base.Convert(tenantPoolDBO, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
		return nil, err
	}
	tenantPoolDBO.SystemPassword = base64.StdEncoding.EncodeToString([]byte(tenantPoolDBO.SystemPassword))
	_, err = keeper.NamedExec(ctx, createTenantPoolQuery, tenantPoolDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
		return nil, err
	}
	err = callback(registrations[0], tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in callback for tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
		// Update to Failed to clean up
		tenantClaim.State = Failed
	}
	err = keeper.UpdateTenantClaimTxn(ctx, registrations[0], tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
	}
	return tenantClaim, err
}

// UpdateTenantClaim updates the tenantClaim trial field.
// Currently, only the trial column is updated
func (keeper *BookKeeper) UpdateTenantClaim(ctx context.Context, tenantClaim *model.TenantClaim) error {
	if tenantClaim == nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid tenantClaim in update tenantClaim"))
		return errcode.NewBadRequestError("tenantClaim")
	}
	if len(tenantClaim.ID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid tenantClaim ID in update tenantClaim"))
		return errcode.NewBadRequestError("tenantClaim.ID")
	}
	now := base.RoundedNow()
	tenantPoolDBO := TenantPoolDBO{ID: tenantClaim.ID, Trial: tenantClaim.Trial, ExpiresAt: tenantClaim.ExpiresAt, UpdatedAt: now}
	// Only the trial column is updated now
	result, err := keeper.NamedExec(ctx, updateTenantPoolTrialQuery, &tenantPoolDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
		return errcode.TranslateDatabaseError("tenantpool", err)
	}
	err = deleteOrUpdateOk(result)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
		return err
	}
	return nil
}

// TriggerDeleteTenantClaim is called to trigger the deletion of the tenant
func (keeper *BookKeeper) TriggerDeleteTenantClaim(ctx context.Context, id string) error {
	now := base.RoundedNow()
	tenantPoolDBO := TenantPoolDBO{ID: id, State: Deleting, UpdatedAt: now}
	updateParam := tenantPoolUpdateQueryParam{TenantPoolDBO: tenantPoolDBO, UnexpectedState: Deleting}
	result, err := keeper.NamedExec(ctx, updateTenantPoolStateQuery, updateParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to trigger deletion of tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
		return errcode.TranslateDatabaseError("tenantpool", err)
	}
	err = deleteOrUpdateOk(result)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to trigger deletion of tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
		return err
	}
	return nil
}

// TriggerDeleteTenantClaims triggers deletion of all the tenantClaims with the given current state
func (keeper *BookKeeper) TriggerDeleteTenantClaims(ctx context.Context, registrationID, currentState string, filterAndOrderByParam base.FilterAndOrderByParameter) error {
	tenantPoolDBO := TenantPoolDBO{RegistrationID: registrationID, State: Deleting, UpdatedAt: base.RoundedNow()}
	updateParam := tenantPoolUpdateQueryParam{TenantPoolDBO: tenantPoolDBO, ExpectedState: Available, UnexpectedState: Deleting}
	query, err := orderByHelper.BuildQuery(entityTypeTenantClaim, updateTenantPoolStatesTemplateQuery, filterAndOrderByParam, "")
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to build update query %s. Error: %s"), updateTenantPoolStatesTemplateQuery, err.Error())
		return err
	}
	_, err = keeper.NamedExec(ctx, query, updateParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to trigger deletion of tenantClaims with state %s for registration %s. Error: %s"), currentState, registrationID, err.Error())
		return errcode.TranslateDatabaseError("tenantpool", err)
	}
	return nil
}

// UpdateTenantClaimTxn is called to update the tenant and the edge states
func (keeper *BookKeeper) UpdateTenantClaimTxn(ctx context.Context, registration *model.Registration, tenantClaim *model.TenantClaim) error {
	err := keeper.DoInTxn(func(tx *base.WrappedTx) error {
		tenantPoolDBO := TenantPoolDBO{}
		err := base.Convert(tenantClaim, &tenantPoolDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
			return err
		}
		now := base.RoundedNow()
		tenantPoolDBO.UpdatedAt = now
		if tenantClaim.State == Assigned && (tenantPoolDBO.AssignedAt == nil || tenantPoolDBO.AssignedAt.IsZero()) {
			tenantPoolDBO.AssignedAt = &now
			if tenantPoolDBO.ExpiresAt == nil || tenantPoolDBO.ExpiresAt.IsZero() {
				trialPeriod, err := keeper.getTrialPeriod(ctx, registration)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to get trial period from %+v. Error: %s"), registration, err.Error())
					return err
				}
				tenantPoolDBO.ExpiresAt = base.TimePtr(now.Add(trialPeriod))
			}
		}
		// We want to make sure we update what we read
		updateQueryParam := &tenantPoolUpdateQueryParam{TenantPoolDBO: tenantPoolDBO, ExpectedVersion: tenantClaim.Version}
		result, err := tx.NamedExec(ctx, updateTenantPoolQuery, updateQueryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
			return errcode.TranslateDatabaseError("tenantPool", err)
		}
		err = deleteOrUpdateOk(result)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
			return err
		}
		err = updateEdgeContexts(ctx, tx, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update edgeContexts %+v. Error: %s"), tenantClaim, err.Error())
		}
		return err
	})
	return err
}

// DeleteTenantClaim deletes the tenant record and the dependent edge contexts
func (keeper *BookKeeper) DeleteTenantClaim(ctx context.Context, tenantClaim *model.TenantClaim) error {
	if tenantClaim.Trial {
		// Remove the association first
		err := DeleteTenantIfPossible(ctx, tenantClaim.ID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to soft delete users for tenant claim %s. Error: %s"), tenantClaim.ID, err.Error())
			return err
		}
		err = SoftPurgeUsers(ctx, tenantClaim.ID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to soft delete users for tenant claim %s. Error: %s"), tenantClaim.ID, err.Error())
			return err
		}
		// TODO later we may try to verify in case someone just logs in around this time.
		// There is a very rare window that the user listing in purging users could not fetch a justly added user
	}
	_, err := DeleteUserByEmail(ctx, tenantClaim.SystemUser)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete system user for tenant claim %s. Error: %s"), tenantClaim.ID, err.Error())
		return err
	}
	result, err := keeper.Delete(ctx, "tps_tenant_pool_model", map[string]interface{}{"id": tenantClaim.ID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant pool %s. Error: %s"), tenantClaim.ID, err.Error())
	}
	err = deleteOrUpdateOk(result)
	if err != nil {
		// Record must have been deleted
		glog.Warningf(base.PrefixRequestID(ctx, "Failed to delete tenant claim %s. Error: %s"), tenantClaim.ID, err.Error())
	}
	return err
}

// RenameSerialNumbers renames serial numbers of nodes added to the trial account such that those nodes can be reused in another tenant
func (keeper *BookKeeper) RenameSerialNumbers(ctx context.Context, tenantClaim *model.TenantClaim) error {
	// For non-trials, serial numbers for the existing edge are not renamed
	if tenantClaim == nil || !tenantClaim.Trial {
		return nil
	}
	query := GetRenameSerialNumberQuery(tenantClaim.ID, base.RoundedNow())
	_, err := keeper.Exec(ctx, query)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in renaming node serial numbers. Error: %s"), err.Error())
	}
	return err
}

// ScanTenantClaims scans the tenant_pool_model for records matching the registration, tenantID and states (all optional)
func (keeper *BookKeeper) ScanTenantClaims(ctx context.Context, registrationID, tenantID string, states []string, queryFilterParam base.QueryParameter, callback model.TenantClaimProcessor) (*cmodel.PagedListResponsePayload, error) {
	// Get the latest registration
	registrations, pageResponse, err := keeper.GetRegistrations(ctx, registrationID, []string{}, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration %s to scan tenantClaims. Error: %s"), registrationID, err.Error())
		return nil, err
	}
	if pageResponse.TotalCount == 0 {
		glog.Infof(base.PrefixRequestID(ctx, "No registration %s found to scan tenantClaims"), registrationID)
		return nil, nil
	}
	scanQueryParam := tenantPoolSelectQueryParam{ID: tenantID, States: states}
	if len(states) == 0 {
		scanQueryParam.AllStates = true
		// Sqlx does not allow empty
		scanQueryParam.States = []string{"dummy"}
	}
	scanQueryParam.RegistrationIDs = []string{}
	trialPeriodMap := map[string]time.Duration{}
	registrationMap := map[string]*model.Registration{}
	for i := range registrations {
		registration := registrations[i]
		trialPeriod, err := keeper.getTrialPeriod(ctx, registration)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get trial period from %+v. Error: %s"), registration, err.Error())
			return nil, err
		}
		// Look only for the tenantclaims with these registrations IDs
		scanQueryParam.RegistrationIDs = append(scanQueryParam.RegistrationIDs, registration.ID)
		trialPeriodMap[registration.ID] = trialPeriod
		registrationMap[registration.ID] = registration
	}
	query, pageQueryParam, err := orderByHelper.BuildPagedQuery(entityTypeTenantClaim, selectTenantPoolTemplateQuery, queryFilterParam, orderByUpdatedAt)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to build query %s. Error: %s"), selectTenantPoolTemplateQuery, err.Error())
		return nil, err
	}
	pageResponse = &cmodel.PagedListResponsePayload{PageIndex: pageQueryParam.GetPageIndex(), PageSize: pageQueryParam.GetPageSize()}
	err = keeper.QueryInWithCallback(ctx, func(dbObjPtr interface{}) error {
		tenantPoolDBO := dbObjPtr.(*TenantPoolDBO)
		tenantClaim := &model.TenantClaim{}
		err := base.Convert(tenantPoolDBO, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
			return err
		}
		pageResponse.TotalCount = tenantPoolDBO.TotalCount
		registration := registrationMap[tenantClaim.RegistrationID]
		// Backward compatibility
		if tenantClaim.State == Assigned && (tenantClaim.ExpiresAt == nil || tenantClaim.ExpiresAt.IsZero()) {
			trialPeriod := trialPeriodMap[tenantClaim.RegistrationID]
			tenantClaim.ExpiresAt = base.TimePtr(tenantClaim.AssignedAt.Add(trialPeriod))
		}
		err = keeper.populateEdgeContexts(ctx, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get edge contexts for tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
			return err
		}
		err = callback(registration, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in callback for tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
		}
		return err
	}, query, TenantPoolDBO{}, scanQueryParam)
	return pageResponse, err
}

// ReserveTenantClaim reserves a tenant under the registration ID and returns the candidate tenant ID.
// The tenant ID must be confirmed in the subsequent call to confirm the reservation
func (keeper *BookKeeper) ReserveTenantClaim(ctx context.Context, registrationID string) (*model.TenantClaim, error) {
	_, pageResponse, err := keeper.GetRegistrations(ctx, registrationID, []string{Active}, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s to reserve a tenantClaim. Error: %s"), registrationID, err.Error())
		return nil, err
	}
	if pageResponse.TotalCount != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s to reserve a tenantClaim"), registrationID)
		return nil, errcode.NewRecordNotFoundError(registrationID)
	}
	scanQueryParam := tenantPoolSelectQueryParam{States: []string{Available}, RegistrationIDs: []string{registrationID}}
	tenantPoolDBOs := []TenantPoolDBO{}
	err = keeper.QueryIn(ctx, &tenantPoolDBOs, selectTenantPoolQuery, scanQueryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to reserve tenant for registration %s. Error: %s"), registrationID, err.Error())
		return nil, err
	}
	var reservedTenant *model.TenantClaim
	for _, tenantPoolDBO := range tenantPoolDBOs {
		tenantClaim := &model.TenantClaim{}
		err = base.Convert(&tenantPoolDBO, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for tenantClaim %+v. Error: %s"), tenantPoolDBO, err.Error())
			return nil, err
		}
		err = keeper.populateEdgeContexts(ctx, tenantClaim)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get edge contexts for tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
			return nil, err
		}
		now := base.RoundedNow()
		tenantPoolDBO := TenantPoolDBO{ID: tenantClaim.ID, RegistrationID: registrationID, State: Reserved, UpdatedAt: now}
		updateQueryParam := &tenantPoolUpdateQueryParam{TenantPoolDBO: tenantPoolDBO, ExpectedState: Available}
		result, err := keeper.NamedExec(ctx, updateTenantPoolStateQuery, updateQueryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim for assignment with param %+v Error: %s"), updateQueryParam, err.Error())
			return nil, errcode.TranslateDatabaseError("tenantPool", err)
		}
		err = deleteOrUpdateOk(result)
		if err == nil {
			tenantClaim.State = Reserved
			reservedTenant = tenantClaim
		} else {
			err = errcode.TranslateDatabaseError("tenantClaim", err)
		}
		if reservedTenant != nil {
			break
		}
	}
	if reservedTenant == nil {
		return nil, errcode.NewRecordNotFoundError("tenantClaim")
	}
	return reservedTenant, nil
}

// ConfirmTenantClaim scans tenant_pool_model to find a record with available state
func (keeper *BookKeeper) ConfirmTenantClaim(ctx context.Context, registrationID, tenantID string) (*model.TenantClaim, error) {
	registrations, pageResponse, err := keeper.GetRegistrations(ctx, registrationID, []string{Active}, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s to confirm the tenantClaim %s. Error: %s"), registrationID, tenantID, err.Error())
		return nil, err
	}
	if pageResponse.TotalCount != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s to confirm the tenantClaim %s"), registrationID, tenantID)
		return nil, errcode.NewRecordNotFoundError(registrationID)
	}
	queryParam := tenantPoolSelectQueryParam{ID: tenantID, States: []string{Reserved}, RegistrationIDs: []string{registrationID}}
	tenantPoolDBOs := []TenantPoolDBO{}
	err = keeper.QueryIn(ctx, &tenantPoolDBOs, selectTenantPoolQuery, queryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenantClaim with param %+v. Error: %s"), queryParam, err.Error())
		return nil, err
	}
	if len(tenantPoolDBOs) != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to find the reservation with param %+v to confirm tenantClaim"), queryParam)
		return nil, errcode.NewRecordNotFoundError(tenantID)
	}
	tenantPoolDBO := tenantPoolDBOs[0]
	now := base.RoundedNow()
	tenantPoolDBO.State = Assigned
	tenantPoolDBO.UpdatedAt = now
	tenantPoolDBO.AssignedAt = &now
	if tenantPoolDBO.ExpiresAt == nil || tenantPoolDBO.ExpiresAt.IsZero() {
		trialPeriod, err := keeper.getTrialPeriod(ctx, registrations[0])
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get trial period from %+v. Error: %s"), registrations[0], err.Error())
			return nil, err
		}
		tenantPoolDBO.ExpiresAt = base.TimePtr(now.Add(trialPeriod))
	}
	tenantClaim := &model.TenantClaim{}
	err = base.Convert(&tenantPoolDBO, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for tenantClaim %+v. Error: %s"), tenantPoolDBOs[0], err.Error())
		return nil, err
	}
	err = keeper.populateEdgeContexts(ctx, tenantClaim)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get edge contexts to confirm tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
		return nil, err
	}
	var assignedTenantClaim *model.TenantClaim
	updateQueryParam := &tenantPoolUpdateQueryParam{TenantPoolDBO: tenantPoolDBO, ExpectedState: Reserved, ExpectedVersion: tenantClaim.Version}
	err = keeper.DoInTxn(func(tx *base.WrappedTx) error {
		// Confirm the tenantClaim
		result, err := tx.NamedExec(ctx, updateTenantPoolStateQuery, updateQueryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim with param %+v Error: %s"), updateQueryParam, err.Error())
			return errcode.TranslateDatabaseError("tenantPool", err)
		}
		err = deleteOrUpdateOk(result)
		if err != nil {
			return errcode.TranslateDatabaseError("tenantClaim", err)
		}
		assignedTenantClaim = tenantClaim
		return nil
	})
	if err != nil {
		return nil, err
	}
	if assignedTenantClaim == nil {
		return nil, errcode.NewRecordNotFoundError("tenantClaim")
	}
	return assignedTenantClaim, nil
}

// AssignTenantClaim assigns an available tenant ID to an email which does not exist in the account server
func (keeper *BookKeeper) AssignTenantClaim(ctx context.Context, registrationID, tenantID, email string) error {
	// updateTenantPoolStateQuery  = "update tps_tenant_pool_model set state = :state, assigned_at = :assigned_at, expires_at = :expires_at, updated_at = :updated_at where id = :id and (:expected_version = 0 or version = :expected_version) and (:expected_state = '' or state = :expected_state) and (:unexpected_state = '' or state != :unexpected_state)"
	if len(tenantID) == 0 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid tenant ID in"))
		return errcode.NewBadRequestError("tenantClaim")
	}
	err := base.ValidateEmail(email)
	if err != nil {
		return err
	}
	registrations, pageResponse, err := keeper.GetRegistrations(ctx, registrationID, []string{Active}, nil)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get active registration %s. Error: %s"), registrationID, err.Error())
		return err
	}
	if pageResponse.TotalCount != 1 {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration %s for tenantclaim assignment"), registrationID)
		return errcode.NewRecordNotFoundError(registrationID)
	}
	trialPeriod, err := keeper.getTrialPeriod(ctx, registrations[0])
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get trial period from %+v. Error: %s"), registrations[0], err.Error())
		return err
	}
	now := base.RoundedNow()
	tenantPoolDBO := TenantPoolDBO{ID: tenantID, State: Reserved, UpdatedAt: now, AssignedAt: &now, ExpiresAt: base.TimePtr(now.Add(trialPeriod))}
	updateQueryParam := &tenantPoolUpdateQueryParam{TenantPoolDBO: tenantPoolDBO, ExpectedState: Available}
	return keeper.DoInTxn(func(tx *base.WrappedTx) error {
		result, err := tx.NamedExec(ctx, updateTenantPoolStateQuery, updateQueryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim for assignment with param %+v Error: %s"), updateQueryParam, err.Error())
			return errcode.TranslateDatabaseError("tenantPool", err)
		}
		err = deleteOrUpdateOk(result)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim for assignment with param %+v Error: %s"), updateQueryParam, err.Error())
			return errcode.TranslateDatabaseError("tenantClaim", err)
		}
		gUser := getUser(tenantID, email)
		_, err = CreateUser(ctx, gUser)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create user %s in cloudmgmt. Error: %s"), email, err.Error())
			return err
		}
		updateQueryParam.ExpectedState = Reserved
		updateQueryParam.State = Assigned
		_, err = tx.NamedExec(ctx, updateTenantPoolStateQuery, updateQueryParam)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenantClaim for assignment with param %+v Error: %s"), updateQueryParam, err.Error())
			return errcode.TranslateDatabaseError("tenantPool", err)
		}
		return nil
	})
}

// PurgeTenants purges the tenant_pool_model table.
// This is used by test only
func (keeper *BookKeeper) PurgeTenants(ctx context.Context, registrationID string) error {
	_, err := keeper.ScanTenantClaims(ctx, registrationID, "", []string{}, nil, func(registration *model.Registration, tenantClaim *model.TenantClaim) error {
		err := PurgeUsers(ctx, tenantClaim.ID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete users for tenant %s from account server. Error: %s"), tenantClaim.ID, err.Error())
		}
		err = DeleteTenantIfPossible(ctx, tenantClaim.ID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete tenant %s from account server. Error: %s"), tenantClaim.ID, err.Error())
		}
		// Ignore error
		return nil
	})
	_, err = keeper.NamedExec(ctx, purgeTenantPoolQuery, &TenantPoolDBO{RegistrationID: registrationID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to purge tenants for registration %s. Error: %s"), registrationID, err.Error())
		return errcode.TranslateDatabaseError("tenantClaims", err)
	}
	return nil
}

// updateEdgeContexts transactionally updates the edge contexts
func updateEdgeContexts(ctx context.Context, tx *base.WrappedTx, tenantClaim *model.TenantClaim) error {
	now := base.RoundedNow()
	_, err := base.DeleteTxn(ctx, tx, "tps_edge_context_model", map[string]interface{}{"tenant_id": tenantClaim.ID})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete edge contexts for tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
		return err
	}
	for _, edgeContext := range tenantClaim.EdgeContexts {
		edgeContextDBO := &EdgeContextDBO{}
		err = base.Convert(edgeContext, edgeContextDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for edgeContext %+v. Error: %s"), edgeContext, err.Error())
			return err
		}
		edgeContextDBO.TenantID = tenantClaim.ID
		edgeContextDBO.CreatedAt = now
		edgeContextDBO.UpdatedAt = now
		_, err = tx.NamedExec(ctx, createEdgeContextQuery, edgeContextDBO)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create edge contexts for %+v. Error: %s"), tenantClaim, err.Error())
			return errcode.TranslateDatabaseError("tenantClaim", err)
		}
	}
	return nil
}

func (keeper *BookKeeper) populateEdgeContexts(ctx context.Context, tenantClaim *model.TenantClaim) error {
	queryParam := EdgeContextDBO{TenantID: tenantClaim.ID}
	edgeContextDBOs := []EdgeContextDBO{}
	err := keeper.Query(ctx, &edgeContextDBOs, selectEdgeContextQuery, queryParam)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to query for edge contexts for tenantClaim %+v. Error: %s"), tenantClaim, err.Error())
		return err
	}
	for i := 0; i < len(edgeContextDBOs); i++ {
		edgeContext := &model.EdgeContext{}
		err = base.Convert(&edgeContextDBOs[i], edgeContext)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in edge context data conversion for edgeContext %+v. Error: %s"), edgeContextDBOs[i], err.Error())
			return err
		}
		tenantClaim.EdgeContexts = append(tenantClaim.EdgeContexts, edgeContext)
	}
	return err
}

func (keeper *BookKeeper) getTrialPeriod(ctx context.Context, registration *model.Registration) (time.Duration, error) {
	var trialPeriod time.Duration
	regConfig, err := registration.GetConfig(ctx)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get registration config %+v. Error: %s"), registration, err.Error())
		return trialPeriod, err
	}
	if regConfig.GetVersionInfo().Version == model.RegConfigV1 {
		configV1 := regConfig.(*model.RegistrationConfigV1)
		trialPeriod = configV1.TrialExpiry
	}
	return trialPeriod, nil
}

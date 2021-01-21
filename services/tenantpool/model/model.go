package model

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/json"
	"time"

	"github.com/golang/glog"
)

// ResourceType is the resource type
type ResourceType string

const (
	// Registration config versions
	RegConfigV1 = "v1"
	// ProjectResourceType is the project resource type
	ProjectResourceType = ResourceType("PROJECT")

	// Currently supported auditlog actors
	AuditLogUserActor   = "USER"
	AuditLogSystemActor = "SYSTEM"

	// Currently supported auditlog actions
	AuditLogReserveTenantAction = "RESERVE_TENANT_CLAIM"
	AuditLogConfirmTenantAction = "CONFIRM_TENANT_CLAIM"
	AuditLogDeleteTenantAction  = "DELETE_TENANT_CLAIM"
	RecreateTenantClaimsAction  = "RECREATE_TENANT_CLAIMS"
	AuditLogAssignTenantAction  = "ASSIGN_TENANT_CLAIM"

	// Currently supported auditlog responses
	AuditLogSuccessResponse = "SUCCESS"
	AuditLogFailedResponse  = "FAILED"
)

// TenantPoolProcessor is used as callback for processing TenantPoolModel
type TenantClaimProcessor func(registration *Registration, tenantClaim *TenantClaim) error

// EdgeProvisioner handles the creation, deletion and status reporting of the edges.
// It makes REST calls to Bott service.
type EdgeProvisioner interface {
	// CreateEdge creates an edge
	CreateEdge(context.Context, *CreateEdgeConfig) (*EdgeInfo, error)
	// GetEdgeStatus returns the status of an edge
	GetEdgeStatus(context.Context, string, string) (*EdgeInfo, error)
	// DeleteEdge deletes an edge
	DeleteEdge(context.Context, string, string) (*EdgeInfo, error)
	// PostDeleteEdges handles clean-up after all the edges for a tenant are deleted
	PostDeleteEdges(context.Context, string) error
	// DescribeEdge returns the details of the edge
	DescribeEdge(context.Context, string, string) (map[string]interface{}, error)
}

// EdgeInfo is the generic response for all the methods in EdgeProvisioner
type EdgeInfo struct {
	// ContextID is the App ID returned by Bott service
	ContextID string `json:"contextId"`
	// Edge is the cloudmgmt edge model
	Edge *model.Edge `json:"edge"`
	// State is one of CREATING, CREATED, FAILED, DELETING, DELETED
	State string `json:"state"`
	// Resources in the edge.
	// Resource ID to resource
	Resources map[string]*Resource `json:"resources"`
}

// Resource identifies an entity generally in the cloud like project
type Resource struct {
	Type ResourceType `json:"type"`
	Name string       `json:"name"`
	ID   string       `json:"id"`
}

// TenantClaim carries information of a tenant along with the edges
type TenantClaim struct {
	// ID is the tenant ID in account service
	ID string `json:"id"`
	// State is one of CREATING, AVAILABLE, RESERVED, CONFIRMED, DELETING
	State string `json:"state"`
	// Registration ID to which this tenant claim is created
	RegistrationID string `json:"registrationId"`
	// System user email to configure the edge
	SystemUser string `json:"systemUser"`
	// System user password to configure the edge
	SystemPassword string `json:"systemPassword"`
	// Trial user or not
	Trial bool `json:"trial"`
	// Resources like project ID are stored as JSON
	// Edge ID to resources from the edge
	Resources map[string]*TenantResource `json:"resources,omitempty"`
	// Edge info for this tenant
	EdgeContexts []*EdgeContext `json:"edgeContexts"`
	AssignedAt   *time.Time     `json:"assignedAt,omitempty"`
	ExpiresAt    *time.Time     `json:"expiresAt,omitempty"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	Version      int64          `json:"version,omitempty"`
}

type TenantResource struct {
	Resource
	EdgeIDs []string `json:"edgeIds"`
}

// EdgeContext carries the provision information of an edge for a tenant
type EdgeContext struct {
	// ContextID is the App ID returned by Bott service
	ID string `json:"id" db:"id"`
	// Actual edge ID
	EdgeID *string `json:"edgeId" db:"edge_id"`
	// State is one of CREATING, CREATED, FAILED, DELETING, DELETED
	State string `json:"state" db:"state"`
	// Type of the edge. Currently, it is VIRTUAL only
	Type      string    `json:"type" db:"type"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	Version   int64     `json:"version,omitempty"`
}

// CreateEdgeConfig is the config used for creating an edge
type CreateEdgeConfig struct {
	TenantID           string   `json:"tenantId"`
	Name               string   `json:"name"`
	SystemUser         string   `json:"systemUser"`
	SystemPassword     string   `json:"systemPassword"`
	InstanceType       string   `json:"instanceType"`
	Tags               []string `json:"tags"`
	DeployApp          bool     `json:"deployApp"` // Deploy U2 apps or not
	DatapipelineDeploy bool     `json:"datapipelineDeploy"`
	DatasourceDeploy   bool     `json:"datasourceDeploy"`
	AppChartVersion    string   `json:"appChartVersion"`
}

// Registration carries information about registration code and configuration
type Registration struct {
	// Registration code or ID
	ID string `json:"id" validate:"range=1:200"`
	// Description about the registration
	Description string `json:"description" validate:"range=0:200"`
	// Config JSON which has a version info
	Config string `json:"config" validate:"range=1"`
	// State can be one of ACTIVE or INACTIVE
	State     string    `json:"state" validate:"options=ACTIVE:INACTIVE"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	Version   int64     `json:"version,omitempty"`
}

// VersionInfo carries the version information
type VersionInfo struct {
	Version string `json:"version" validate:"range=1"`
}

// RegistrationConfig is an interface for different versions of registration configurations
type RegistrationConfig interface {
	GetVersionInfo() *VersionInfo
}

// RegistrationConfigV1 is the V1 version of RegistrationConfig
// Any future registration config must implement RegistrationConfig
type RegistrationConfigV1 struct {
	VersionInfo
	EdgeCount             int    `json:"edgeCount" validate:"range=1:10"`
	InstanceType          string `json:"instanceType" validate:"range=0"`
	MinTenantPoolSize     int    `json:"minTenantPoolSize" validate:"range=0"`
	MaxTenantPoolSize     int    `json:"maxTenantPoolSize" validate:"range=0"`
	MaxPendingTenantCount int    `json:"maxPendingTenantCount" validate:"range=1"`
	// The lifetime of the tenant claim after assignment
	TrialExpiry time.Duration `json:"trialExpiry,omitempty"`
	DeployApps  bool          `json:"deployApps,omitempty"`
}

// AuditLog
type AuditLog struct {
	ID             int64     `json:"id"`
	TenantID       string    `json:"tenantId,omitempty"`       // optional
	RegistrationID string    `json:"registrationId,omitempty"` // optional
	Email          string    `json:"email,omitempty"`          // optional
	Actor          string    `json:"actor" validate:"range=1:200"`
	Action         string    `json:"action" validate:"range=1:200"`
	Response       string    `json:"response" validate:"range=1:200"`
	Description    string    `json:"description,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// GetVersionInfo gives the version info
func (configV1 *RegistrationConfigV1) GetVersionInfo() *VersionInfo {
	return &configV1.VersionInfo
}

// GetConfig unmarshals the config string field into a struct
func (registration *Registration) GetConfig(ctx context.Context) (RegistrationConfig, error) {
	bytes := []byte(registration.Config)
	versionInfo := &VersionInfo{}
	err := json.Unmarshal(bytes, versionInfo)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get version in config %s. Error: %s"), registration.Config, err.Error())
		return nil, errcode.NewBadRequestError("version")
	}
	if versionInfo.Version == RegConfigV1 {
		configV1 := &RegistrationConfigV1{}
		err = json.Unmarshal(bytes, configV1)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get version in config %s. Error: %s"), registration.Config, err.Error())
			return nil, errcode.NewBadRequestExError("config", err.Error())
		}
		return configV1, nil
	}
	return nil, errcode.NewBadRequestError("version")
}

// ValidateRegistration validates the registration
func ValidateRegistration(ctx context.Context, registration *Registration) error {
	if registration == nil {
		return errcode.NewBadRequestError("registration")
	}
	err := base.ValidateStruct("registration", registration, "na")
	if err != nil {
		return err
	}
	config, err := registration.GetConfig(ctx)
	if err != nil {
		return err
	}
	if config.GetVersionInfo().Version == RegConfigV1 {
		configV1 := config.(*RegistrationConfigV1)
		err = base.ValidateStruct("config", configV1, "na")
		if err != nil {
			return err
		}
		if configV1.MaxTenantPoolSize < configV1.MinTenantPoolSize {
			return errcode.NewBadRequestError("poolsize")
		}
	}
	return nil
}

// ValidateRegistrationUpdate validates the input registration for update against the existing registration
func ValidateRegistrationUpdate(ctx context.Context, oldRegistration *Registration, newRegistration *Registration) error {
	err := ValidateRegistration(ctx, newRegistration)
	if err != nil {
		return err
	}
	oldConfig, err := oldRegistration.GetConfig(ctx)
	if err != nil {
		return err
	}
	newConfig, err := newRegistration.GetConfig(ctx)
	if err != nil {
		return err
	}
	if oldConfig.GetVersionInfo().Version != newConfig.GetVersionInfo().Version {
		return errcode.NewBadRequestExError("version", "Non-modifiable value")
	}
	oldRegConfig := oldConfig.(*RegistrationConfigV1)
	newRegConfig := newConfig.(*RegistrationConfigV1)
	if oldRegConfig.Version == RegConfigV1 {
		if newRegConfig.EdgeCount != oldRegConfig.EdgeCount {
			return errcode.NewBadRequestExError("edgeCount", "Non-modifiable value")
		}
		if newRegConfig.InstanceType != oldRegConfig.InstanceType {
			return errcode.NewBadRequestExError("instanceType", "Non-modifiable value")
		}
	}
	return nil
}

// ValidateAuditLog validates the input AuditLog
func ValidateAuditLog(auditLog *AuditLog) error {
	if auditLog == nil {
		return errcode.NewBadRequestError("auditLog")
	}
	if auditLog.Actor != AuditLogUserActor && auditLog.Actor != AuditLogSystemActor {
		return errcode.NewBadRequestError("auditLog.Actor")
	}
	if auditLog.Action != AuditLogReserveTenantAction &&
		auditLog.Action != AuditLogConfirmTenantAction &&
		auditLog.Action != AuditLogDeleteTenantAction {
		return errcode.NewBadRequestError("auditLog.Action")
	}
	if auditLog.Response != AuditLogSuccessResponse && auditLog.Response != AuditLogFailedResponse {
		return errcode.NewBadRequestError("auditLog.Response")
	}
	identityCount := 0
	if len(auditLog.RegistrationID) > 0 {
		identityCount++
	}
	if len(auditLog.TenantID) > 0 {
		identityCount++
	}
	if len(auditLog.Email) > 0 {
		identityCount++
	}
	if identityCount == 0 {
		return errcode.NewBadRequestError("identity")
	}
	return nil
}

package model

import (
	"time"
)

// AuditLogV2 is the object model for audit log
// swagger:model AuditLogV2
type AuditLogV2 struct {
	Timestamp         time.Time `json:"timestamp"`
	TenantID          string    `json:"tenantID"`
	Operation         string    `json:"operation"`
	OperationType     string    `json:"operationType"`
	ResourceName      string    `json:"resourceName,omitempty"`
	ResourceID        string    `json:"resourceID,omitempty"`
	ResourceType      string    `json:"resourceType,omitempty"`
	ModifierName      string    `json:"modifierName"`
	ModifierID        string    `json:"modifierID"`
	ModifierRole      string    `json:"modifierRole"`
	ProjectName       string    `json:"projectName,omitempty"`
	ProjectID         string    `json:"projectID,omitempty"`
	ServiceDomainName string    `json:"serviceDomainName,omitempty"`
	ServiceDomainID   string    `json:"serviceDomainID,omitempty"`
	Payload           string    `json:"payload,omitempty"`
	Scope             string    `json:"scope"` // Allowed values : INFRA, PROJECT
}

// AuditLogV2InsertRequest is the request payload for InsertAuditLogV2
// swagger:model AuditLogV2InsertRequest
type AuditLogV2InsertRequest struct {
	AuditLog AuditLogV2 `json:"auditlog"`
}

// AuditLogsV2InsertRequest is the request payload for InsertAuditLogsV2
// swagger:model AuditLogsV2InsertRequest
type AuditLogsV2InsertRequest struct {
	AuditLogs []AuditLogV2 `json:"auditlogs"`
}

// AuditLogV2InsertParam is used as API parameter for InsertAuditLogV2
// swagger:parameters InsertAuditLogV2
type AuditLogV2InsertParam struct {
	// This is events upsert request description
	// in: body
	// required: true
	Body *AuditLogV2InsertRequest
}

// AuditLogsV2InsertParam is used as API parameter for InsertAuditLogsV2
// swagger:parameters InsertAuditLogsV2
type AuditLogsV2InsertParam struct {
	// This is events upsert request description
	// in: body
	// required: true
	Body *AuditLogsV2InsertRequest
}

// AuditLogV2Filter is the audit log filter in QueryAuditLogsV2.
// StartTime is the later time (inclusive) going back to the earlier EndTime (exclusive)
// swagger:model AuditLogV2Filter
type AuditLogV2Filter struct {
	//
	// TenantID must be provided in order to search audit logs
	//
	// required: true
	//TenantID string `json:"tenantID"`
	//
	// Optional search parameters like "termsKeyValue" : {"modifierName": "user1"}
	//
	TermsKeyValue map[string]AuditLogV2MultipleValues `json:"termsKeyValue"`
	FromDocument  int                                 `json:"fromDocument"`
	PageSize      int                                 `json:"pageSize"`
	GroupBy       string                              `json:"groupBy"`
	//
	// Search for events by this later timestamp (inclusive)
	//
	StartTime *time.Time `json:"startTime"`
	//
	// Search for events by this earlier timestamp (inclusive).
	//
	EndTime *time.Time `json:"endTime"`
	Scopes  []string     `json:"scopes"`
}

type AuditLogV2MultipleValues struct {
	Values []string `json:"values"`
}

// AuditLogV2FilterParam is the audit log filter used as API parameter
// swagger:parameters QueryAuditLog
type AuditLogV2FilterParam struct {
	// in: body
	// required: true
	Payload *AuditLogV2Filter
}

// Ok
// swagger:response AuditLogV2ListResponse
type AuditLogsV2ListResponse struct {
	// in: body
	// required: true
	Payload *[]AuditLogV2
}

// Enable authorization on the endpoints
// swagger:parameters QueryAuditLogsV2
// in: header
type AuditLogAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

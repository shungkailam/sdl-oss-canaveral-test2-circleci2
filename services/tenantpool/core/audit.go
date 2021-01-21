package core

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/tenantpool/model"
	"context"
	"time"

	"github.com/golang/glog"
)

const (
	// Queries
	selectAuditLogQuery = "select * from tps_audit_log_model where (:id = 0 or id = :id) and (:tenant_id = '' or tenant_id = :tenant_id) and (:registration_id = '' or registration_id = :registration_id) and (:email = '' or email = :email) and (:actor = '' or actor = :actor) and (:action = '' or action = :action) and (:response = '' or response = :response)"
	insertAuditLogQuery = "insert into tps_audit_log_model (tenant_id, registration_id, email, actor, action, response, description) values (:tenant_id, :registration_id, :email, :actor, :action, :response, :description)"
	deleteAuditLogQuery = "delete from tps_audit_log_model where (:id = 0 or id = :id) and (:tenant_id = '' or tenant_id = :tenant_id) and (:registration_id = '' or registration_id = :registration_id) and (:email = '' or email = :email) and (:actor = '' or actor = :actor) and (:action = '' or action = :action) and (:response = '' or response = :response)"
	countAuditLogQuery  = "select c.total_count, created_at from tps_audit_log_model t, (select count(distinct email) as total_count from tps_audit_log_model where registration_id = :registration_id and action = :action and created_at > :created_at) c where t.registration_id = :registration_id and t.action = :action and created_at > :created_at order by t.created_at desc limit 1"
)

var (
	createAuditLogAsyncTimeout = time.Second * 5
)

type AuditLogManager struct {
	*base.DBObjectModelAPI
}

type AuditLogDBO struct {
	ID             int64     `json:"id,omitempty" db:"id"`
	TenantID       string    `json:"tenantId,omitempty" db:"tenant_id"`
	RegistrationID string    `json:"registrationId,omitempty" db:"registration_id"`
	Email          string    `json:"email,omitempty" db:"email"`
	Actor          string    `json:"actor" db:"actor"`
	Action         string    `json:"action" db:"action"`
	Response       string    `json:"response" db:"response"`
	Description    string    `json:"description,omitempty" db:"description"`
	CreatedAt      time.Time `json:"createdAt" db:"created_at"`
	TotalCount     int       `json:"totalCount" db:"total_count"`
}

func (manager *AuditLogManager) CreateAuditLog(ctx context.Context, auditLog *model.AuditLog) error {
	err := model.ValidateAuditLog(auditLog)
	if err != nil {
		return errcode.NewBadRequestError("auditLog")
	}
	auditLogDBO := &AuditLogDBO{}
	err = base.Convert(auditLog, auditLogDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for auditlog %+v. Error: %s"), auditLog, err.Error())
		return err
	}
	_, err = manager.NamedExec(ctx, insertAuditLogQuery, auditLogDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create auditlog %+v. Error: %s"), auditLogDBO, err.Error())
		return errcode.TranslateDatabaseError("auditLog", err)
	}
	return nil
}

func (manager *AuditLogManager) CreateAuditLogHelper(ctx context.Context, apiError error, auditLog *model.AuditLog, isAsync bool) error {
	if auditLog == nil {
		return errcode.NewBadRequestError("auditLog")
	}
	if apiError == nil {
		auditLog.Response = model.AuditLogSuccessResponse
	} else {
		auditLog.Response = model.AuditLogFailedResponse
		auditLog.Description = apiError.Error()
	}
	err := model.ValidateAuditLog(auditLog)
	if err != nil {
		return err
	}
	createFn := func(ctx context.Context, cancelFn context.CancelFunc) error {
		if cancelFn != nil {
			defer cancelFn()
		}
		// New error
		err := manager.CreateAuditLog(ctx, auditLog)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create auditlog for %+v. Error: %s"), auditLog, err.Error())
		}
		return err
	}
	if isAsync {
		// incoming ctx could be cancelled after the request is done
		calleeCtx := context.WithValue(context.Background(), base.RequestIDKey, base.GetRequestID(ctx))
		calleeCtx, cancelFn := context.WithTimeout(calleeCtx, createAuditLogAsyncTimeout)
		go createFn(calleeCtx, cancelFn)
	} else {
		err = createFn(ctx, nil)
	}
	return err
}

// TODO pagination and other
// For U2, we just want to dump data. This is used for testing
func (manager *AuditLogManager) GetAuditLogs(ctx context.Context, param *model.AuditLog) ([]*model.AuditLog, error) {
	if param == nil {
		return nil, errcode.NewBadRequestError("param")
	}
	paramDBO := AuditLogDBO{}
	err := base.Convert(param, &paramDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for auditlog %+v. Error: %s"), param, err.Error())
		return nil, err
	}
	auditLogs := []*model.AuditLog{}
	_, err = manager.NotPagedQuery(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		auditLogDBO := dbObjPtr.(*AuditLogDBO)
		auditLog := &model.AuditLog{}
		err := base.Convert(auditLogDBO, auditLog)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for auditlog %+v. Error: %s"), auditLogDBO, err.Error())
			return err
		}
		auditLogs = append(auditLogs, auditLog)
		return nil
	}, selectAuditLogQuery, paramDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get auditlogs for query %+v. Error: %s"), paramDBO, err.Error())
		return nil, errcode.TranslateDatabaseError("auditLog", err)
	}
	return auditLogs, nil
}

// GetAuditLogCount returns the count for the matching rows and the latest entry time
func (manager *AuditLogManager) GetAuditLogCount(ctx context.Context, param *model.AuditLog) (int, time.Time, error) {
	now := base.RoundedNow()
	if param == nil {
		return 0, now, errcode.NewBadRequestError("param")
	}
	paramDBO := AuditLogDBO{}
	err := base.Convert(param, &paramDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for auditlog %+v. Error: %s"), param, err.Error())
		return 0, now, err
	}
	auditLogDBOs := []AuditLogDBO{}
	err = manager.Query(ctx, &auditLogDBOs, countAuditLogQuery, paramDBO)
	if err != nil {
		return 0, now, err
	}
	if len(auditLogDBOs) == 0 {
		return 0, now, nil
	}
	return auditLogDBOs[0].TotalCount, auditLogDBOs[0].CreatedAt, nil
}

// For U2, we just want to dump data. This is used for testing
func (manager *AuditLogManager) DeleteAuditLog(ctx context.Context, param *model.AuditLog) error {
	err := model.ValidateAuditLog(param)
	if err != nil {
		return errcode.NewBadRequestError("param")
	}
	auditLogDBO := &AuditLogDBO{}
	err = base.Convert(param, auditLogDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed in data conversion for auditlog %+v. Error: %s"), param, err.Error())
		return err
	}
	_, err = manager.NamedExec(ctx, deleteAuditLogQuery, auditLogDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete auditlog %+v. Error: %s"), auditLogDBO, err.Error())
		return errcode.TranslateDatabaseError("auditLog", err)
	}
	return nil
}

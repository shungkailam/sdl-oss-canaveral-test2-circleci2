package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"cloudservices/common/schema"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx/types"
	funk "github.com/thoas/go-funk"
)

const (
	entityTypeServiceInstance       = "serviceinstance"
	entityTypeServiceInstanceStatus = "service"
	serviceInstanceModelAlias       = "sim"
	serviceClassModelAlias          = "scm"
	serviceInstanceEventBatchSize   = 20
)

var (
	serviceInstanceModelAliasMap = map[string]string{
		"type":        serviceClassModelAlias,
		"svc_version": serviceClassModelAlias,
	}

	// Lead time given to the push timestamp
	// Delta inventory API which is invoked every 5 minutes.
	// Event will be stale in <= 1 minute since events are pushed every minute
	svcInstancePushLeadTime = time.Duration(time.Minute * 6)
)

func init() {
	queryMap["SelectServiceInstances"] = `SELECT sim.*, scm.name as svc_class_name, scm.type, scm.svc_version, scm.scope, scm.min_svc_domain_version, count(sim.*) OVER() as total_count FROM service_instance_model sim, service_class_model scm WHERE (sim.tenant_id = :tenant_id) AND (:id = '' OR sim.id = :id) AND (sim.svc_class_id = scm.id) AND (:svc_class_id = '' OR scm.id = :svc_class_id) AND (:type = '' OR scm.type = :type) AND (:svc_version = '' OR scm.svc_version = :svc_version) AND (:scope = '' OR scm.scope = :scope) AND ((:filter_svc_domains = false AND sim.svc_domain_scope_id IS NOT NULL) OR (:filter_svc_domains = true AND sim.svc_domain_scope_id IN (:svc_domain_ids)) OR (:filter_projects = false AND sim.project_scope_id IS NOT NULL) OR (:filter_projects = true AND sim.project_scope_id IN (:project_ids)))`
	queryMap["DeltaSelectServiceInstances"] = `SELECT sim.*, scm.name as svc_class_name, scm.type, scm.svc_version, scm.scope, scm.min_svc_domain_version FROM service_instance_model sim, service_class_model scm WHERE (sim.svc_class_id = scm.id) AND (sim.id IN (:svc_instance_ids))`
	queryMap["CreateServiceInstance"] = `INSERT INTO service_instance_model (id, tenant_id, name, description, svc_class_id, svc_domain_scope_id, project_scope_id, parameters, version, created_at, updated_at) VALUES (:id, :tenant_id, :name, :description, :svc_class_id, :svc_domain_scope_id, :project_scope_id, :parameters, :version, :created_at, :updated_at)`
	queryMap["UpdateServiceInstance"] = `UPDATE service_instance_model SET name = :name, description = :description, parameters = :parameters, version = :version, updated_at = :updated_at WHERE tenant_id = :tenant_id AND id = :id`

	orderByHelper.Setup(entityTypeServiceInstance, []string{"id", "name", "type", "svc_version", "version", "created_at", "updated_at"})
	orderByHelper.Setup(entityTypeServiceInstanceStatus, []string{})
}

// ServiceInstanceCommonDBO is the shared common DBO for convenience
type ServiceInstanceCommonDBO struct {
	ServiceClassCommonDBO
	SvcClassID       string  `json:"svcClassId" db:"svc_class_id"`
	SvcClassName     string  `json:"svcClassName" db:"svc_class_name"`
	SvcDomainScopeID *string `json:"svcDomainScopeId" db:"svc_domain_scope_id"` // To use DB constraint
	ProjectScopeID   *string `json:"projectScopeId" db:"project_scope_id"`      // To use DB constraint
}

// ServiceInstanceDBO is the DB model for Service Instance
type ServiceInstanceDBO struct {
	model.BaseModelDBO
	ServiceInstanceCommonDBO
	Name        string          `json:"name" db:"name"`
	Description string          `json:"description" db:"description"`
	Parameters  *types.JSONText `json:"parameters" db:"parameters"`
}

// ServiceInstanceQueryParam is the query param for Service Instance model
type ServiceInstanceQueryParam struct {
	model.BaseModelDBO
	ServiceClassCommonDBO
	FilterIDsParam
	Scope          string   `json:"scope" db:"scope"`
	SvcClassID     string   `json:"svcClassId" db:"svc_class_id"`
	SvcInstanceIDs []string `json:"svcInstanceIds" db:"svc_instance_ids"`
}

// ServiceInstanceStatusEventQueryParam is the query parameter values for
// Service Status event path
type ServiceInstanceStatusEventQueryParam struct {
	model.ServiceInstanceCommon
	SvcInstanceID string `json:"svcInstanceId"`
	SvcDomainID   string `json:"svcDomainId"`
	ProjectID     string `json:"projectId"`
}

// FilterIDsParam holds the filter entity IDs
type FilterIDsParam struct {
	FilterSvcDomains bool     `json:"filterSvcDomains" db:"filter_svc_domains"`
	FilterProjects   bool     `json:"filterProjects" db:"filter_projects"`
	SvcDomainIDs     []string `json:"svcDomainIds" db:"svc_domain_ids"`
	ProjectIDs       []string `json:"projectIds" db:"project_ids"`
}

func (dbAPI *dbObjectModelAPI) ValidateCreateServiceInstance(ctx context.Context, svcClass *model.ServiceClass, svcInstance *model.ServiceInstance) error {
	if svcInstance.Parameters == nil {
		svcInstance.Parameters = map[string]interface{}{}
	}
	if svcClass.Schemas.SvcInstance.Create.Parameters == nil {
		svcClass.Schemas.SvcInstance.Create.Parameters = map[string]interface{}{}
	}
	if len(svcInstance.ScopeID) == 0 {
		return errcode.NewBadRequestError("scopeId")
	}
	err := dbAPI.validateScopeIDTenant(ctx, svcClass, svcInstance)
	if err != nil {
		return err
	}
	if svcClass.Scope == model.ServiceClassServiceDomainScope {
		versionMap, err := dbAPI.GetServiceDomainVersions(ctx, []string{svcInstance.ScopeID})
		if err != nil {
			return err
		}
		svcVersion, ok := versionMap[svcInstance.ScopeID]
		if !ok {
			glog.Warningf(base.PrefixRequestID(ctx, "Service Domain version is unknown for %s"), svcInstance.ScopeID)
			// Do not fail to prevent blocking of Service Domain creation UX workflow
			return nil
		}
		compResult, err := base.CompareVersions(svcClass.MinSvcDomainVersion, svcVersion)
		if err != nil {
			return err
		}
		if compResult > 0 {
			return errcode.NewBadRequestExError("svcVersion", fmt.Sprintf("Minimum version %s is required", svcClass.MinSvcDomainVersion))
		}
	}
	err = schema.ValidateSchemaMap(ctx, svcClass.Schemas.SvcInstance.Create.Parameters, svcInstance.Parameters)
	if err != nil {
		return err
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) ValidateUpdateServiceInstance(ctx context.Context, svcClass *model.ServiceClass, svcInstance *model.ServiceInstance) error {
	if svcInstance.Parameters == nil {
		svcInstance.Parameters = map[string]interface{}{}
	}
	if svcClass.Schemas.SvcInstance.Create.Parameters == nil {
		svcClass.Schemas.SvcInstance.Create.Parameters = map[string]interface{}{}
	}
	// TODO We should not allow update of the scope ID
	err := dbAPI.validateScopeIDTenant(ctx, svcClass, svcInstance)
	if err != nil {
		return err
	}
	err = schema.ValidateSchemaMap(ctx, svcClass.Schemas.SvcInstance.Update.Parameters, svcInstance.Parameters)
	if err != nil {
		return err
	}
	//TODO
	return nil
}

// validateScopeIDTenant validates if the scope ID belongs to the same tenant
func (dbAPI *dbObjectModelAPI) validateScopeIDTenant(ctx context.Context, svcClass *model.ServiceClass, svcInstance *model.ServiceInstance) error {
	if svcClass.Scope == model.ServiceClassServiceDomainScope {
		err := dbAPI.checkTenant(ctx, EdgeClusterTableName, []string{svcInstance.ScopeID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in verifying if the Service Domain scope ID %s belongs to the same tenant. Error: %s"), svcInstance.ScopeID, err.Error())
			return errcode.NewBadRequestExError("scopeId", err.Error())
		}
	} else if svcClass.Scope == model.ServiceClassProjectScope {
		err := dbAPI.checkTenant(ctx, ProjectTableName, []string{svcInstance.ScopeID})
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in verifying if the Project scope ID %s belongs to the same tenant. Error: %s"), svcInstance.ScopeID, err.Error())
			return errcode.NewBadRequestExError("scopeId", err.Error())
		}
	} else {
		return errcode.NewBadRequestError("scope")
	}
	return nil
}

func convertServiceInstanceDBO(svcInstance *model.ServiceInstance, svcInstanceDBO *ServiceInstanceDBO) error {
	err := base.Convert(svcInstance, svcInstanceDBO)
	if err != nil {
		return err
	}
	if svcInstance.Scope == model.ServiceClassServiceDomainScope {
		svcInstanceDBO.SvcDomainScopeID = base.StringPtr(svcInstance.ScopeID)
	} else if svcInstance.Scope == model.ServiceClassProjectScope {
		svcInstanceDBO.ProjectScopeID = base.StringPtr(svcInstance.ScopeID)
	} else {
		return errcode.NewBadRequestError("scope")
	}
	return nil
}

func convertServiceInstance(svcInstanceDBO *ServiceInstanceDBO, svcInstance *model.ServiceInstance) error {
	err := base.Convert(svcInstanceDBO, svcInstance)
	if err != nil {
		return err
	}
	if svcInstanceDBO.Scope == string(model.ServiceClassServiceDomainScope) {
		if svcInstanceDBO.SvcDomainScopeID == nil {
			// Not supposed to happen.
			// This prevents panic
			return errcode.NewInternalError(fmt.Sprintf("Scope ID for Service Domain scope is not set in Service Instance %s", svcInstance.ID))
		}
		svcInstance.ScopeID = *svcInstanceDBO.SvcDomainScopeID
	} else if svcInstanceDBO.Scope == string(model.ServiceClassProjectScope) {
		if svcInstanceDBO.ProjectScopeID == nil {
			// Not supposed to happen.
			// This prevents panic
			return errcode.NewInternalError(fmt.Sprintf("Scope ID for Project scope is not set in Service Instance %s", svcInstance.ID))
		}
		svcInstance.ScopeID = *svcInstanceDBO.ProjectScopeID
	} else {
		return errcode.NewBadRequestError("scope")
	}
	return nil
}

// upsertServiceBindingEvents upserts events corresponding to min Service Domain version requirement
func (dbAPI *dbObjectModelAPI) upsertServiceInstanceEvents(ctx context.Context, svcInstances []*model.ServiceInstance, svcDomainIDs map[string]string) error {
	svcDomainPushTimeStamps, err := dbAPI.GetServiceDomainPushTimeStamps(ctx)
	if err != nil {
		return err
	}
	errMsgs := []string{}
	eventUpserter := func(req model.EventUpsertRequest) {
		if len(req.Events) == 0 {
			return
		}
		_, err := dbAPI.UpsertEvents(ctx, req, nil)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Error in upserting events %+v. Error: %s"), req, err.Error())
			errMsgs = append(errMsgs, err.Error())
		}
	}
	eventUpsertReq := model.EventUpsertRequest{}
	for i := range svcInstances {
		svcInstance := svcInstances[i]
		for svcDomainID, svcDomainVersion := range svcDomainIDs {
			pushTimeStamp, ok := svcDomainPushTimeStamps[svcDomainID]
			if !ok {
				// UTC time string
				pushTimeStamp = base.RoundedNow()
			}
			pushTimeStamp = pushTimeStamp.Add(svcInstancePushLeadTime)
			// Event path params for Service Instance
			eventQueryParam := ServiceInstanceStatusEventQueryParam{
				ServiceInstanceCommon: svcInstance.ServiceInstanceCommon,
				SvcInstanceID:         svcInstance.ID,
				SvcDomainID:           svcDomainID,
			}
			ePathTemplate := model.ServiceInstanceStatusServiceDomainScopedEventPath
			if svcInstance.Scope == model.ServiceClassProjectScope {
				ePathTemplate = model.ServiceInstanceStatusProjectScopedEventPath
				eventQueryParam.ProjectID = svcInstance.ScopeID
			}
			ePath, err := model.GenerateEventUpsertPath(ePathTemplate, eventQueryParam)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Error in generating event path using the template %s with values %+v. Error: %s"), ePathTemplate, eventQueryParam, err.Error())
				errMsgs = append(errMsgs, err.Error())

			} else {
				var message string
				if svcDomainVersion == "" {
					// Unknown version
					message = fmt.Sprintf("Minimum Service Domain version %s is required by the Service Instance", svcInstance.MinSvcDomainVersion)
				} else {
					message = fmt.Sprintf("Minimum Service Domain version %s is required by the Service Instance. Current Service Domain version is %s", svcInstance.MinSvcDomainVersion, svcDomainVersion)
				}
				event := model.Event{
					Path:      ePath,
					Type:      "ALERT",
					Severity:  "CRITICAL",
					State:     string(model.ServiceInstanceFailedState),
					Message:   message,
					Timestamp: base.RoundedNow(),
					Properties: map[string]string{
						pushTimeStampLabel: pushTimeStamp.String(),
					},
					IsInfraEntity: true,
				}
				eventUpsertReq.Events = append(eventUpsertReq.Events, event)
				if len(eventUpsertReq.Events) >= serviceInstanceEventBatchSize {
					eventUpserter(eventUpsertReq)
					eventUpsertReq = model.EventUpsertRequest{}
				}
			}
		}
	}
	// Last few events
	eventUpserter(eventUpsertReq)
	if len(errMsgs) > 0 {
		return errcode.NewInternalError(strings.Join(errMsgs, "\n"))
	}
	return nil
}

func (dbAPI *dbObjectModelAPI) getServiceInstanceServiceDomainIDs(ctx context.Context, svcInstance *model.ServiceInstance) ([]string, error) {
	var svcDomainIDs []string
	var err error
	if svcInstance.Scope == model.ServiceClassServiceDomainScope {
		svcDomainIDs = append(svcDomainIDs, svcInstance.ScopeID)
	} else if svcInstance.Scope == model.ServiceClassProjectScope {
		svcDomainIDs, err = dbAPI.GetProjectEdges(ctx, ProjectEdgeDBO{
			ProjectID: svcInstance.ScopeID,
		})
	}
	if err != nil {
		return nil, err
	}
	if len(svcDomainIDs) == 0 {
		return svcDomainIDs, nil
	}
	invalidSvcDomainIDs := map[string]string{}
	svcDomainIDs, err = dbAPI.FilterServiceDomainIDsByVersion(ctx, svcInstance.MinSvcDomainVersion, svcDomainIDs, func(svcDomainID, svcDomainVersion string, compResult int) bool {
		if compResult <= 0 {
			// minSvcDomainVersion is <= Service Domain version
			// add the ID
			return true
		}
		glog.Warningf(base.PrefixRequestID(ctx, "Service Instance %+v is not supported on Service Domain %s"), svcInstance.ServiceInstanceCommon, svcDomainID)
		invalidSvcDomainIDs[svcDomainID] = svcDomainVersion
		return false
	})
	if len(invalidSvcDomainIDs) > 0 {
		dbAPI.upsertServiceInstanceEvents(ctx, []*model.ServiceInstance{svcInstance}, invalidSvcDomainIDs)
		// ignore error
	}
	return svcDomainIDs, err
}

// getPermittedFilterIDsParam finds all the permitted entity IDs
func (dbAPI *dbObjectModelAPI) getPermittedFilterIDsParam(ctx context.Context, requestedFilterID string) (FilterIDsParam, error) {
	filterIDsParam := FilterIDsParam{FilterSvcDomains: true, FilterProjects: true, SvcDomainIDs: []string{}, ProjectIDs: []string{}}
	authContext, err := base.GetAuthContext(ctx)
	fixFunc := func(filterIDsParam FilterIDsParam) FilterIDsParam {
		if len(filterIDsParam.SvcDomainIDs) == 0 {
			// In-query requires non-empty slice
			filterIDsParam.SvcDomainIDs = append(filterIDsParam.SvcDomainIDs, "\t")
		}
		if len(filterIDsParam.ProjectIDs) == 0 {
			// In-query requires non-empty slice
			filterIDsParam.ProjectIDs = append(filterIDsParam.ProjectIDs, "\t")
		}
		glog.V(5).Infof(base.PrefixRequestID(ctx, "Resolved IDs for requested filter ID %s: %+v"), requestedFilterID, filterIDsParam)
		return filterIDsParam
	}
	if err != nil {
		return filterIDsParam, err
	}
	if auth.IsInfraAdminRole(authContext) {
		filterIDsParam.FilterProjects = false
		filterIDsParam.FilterSvcDomains = false
		return fixFunc(filterIDsParam), nil
	}
	addedProjectIDs := auth.GetProjectIDs(authContext)
	// Non-infra
	// This returns the filtered affiliated Service Domains
	svcDomains, err := dbAPI.SelectAllServiceDomains(ctx, nil)
	if err != nil {
		return filterIDsParam, err
	}
	if len(requestedFilterID) == 0 {
		filterIDsParam.SvcDomainIDs = (funk.Map(svcDomains, func(e interface{}) string { return e.(model.ServiceDomain).ID })).([]string)
		filterIDsParam.ProjectIDs = addedProjectIDs
		return fixFunc(filterIDsParam), nil
	}
	for _, svcDomain := range svcDomains {
		if svcDomain.ID == requestedFilterID {
			filterIDsParam.SvcDomainIDs = append(filterIDsParam.SvcDomainIDs, requestedFilterID)
			return fixFunc(filterIDsParam), nil
		}
	}
	for _, projectID := range addedProjectIDs {
		if projectID == requestedFilterID {
			filterIDsParam.ProjectIDs = append(filterIDsParam.ProjectIDs, requestedFilterID)
			return fixFunc(filterIDsParam), nil
		}
	}
	return filterIDsParam, errcode.NewPermissionDeniedError("RBAC")
}

// CreateServiceInstance creates a Service Instance in the DB
func (dbAPI *dbObjectModelAPI) CreateServiceInstance(ctx context.Context, i interface{} /* *model.ServiceInstanceParam */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ServiceInstanceParam)
	if !ok {
		return resp, errcode.NewInternalError("CreateServiceInstance: type error")
	}
	createParam := *p
	tenantID := authContext.TenantID
	if !base.CheckID(createParam.ID) {
		createParam.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(ctx, "CreateServiceInstance doc.ID was invalid, update it to %s\n"), createParam.ID)
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityServiceInstance,
		meta.OperationCreate,
		auth.RbacContext{ID: createParam.ID})
	if err != nil {
		return resp, err
	}

	if len(createParam.SvcClassID) == 0 {
		return resp, errcode.NewBadRequestError("svcClassId")
	}
	doc := model.ServiceInstance{}
	err = base.Convert(&createParam, &doc)
	if err != nil {
		return resp, err
	}
	doc.TenantID = tenantID

	svcClass, err := dbAPI.GetServiceClass(ctx, doc.SvcClassID)
	if err != nil {
		return resp, err
	}
	doc.ServiceClassCommon = svcClass.ServiceClassCommon
	err = dbAPI.ValidateCreateServiceInstance(ctx, &svcClass, &doc)
	if err != nil {
		return resp, err
	}
	doc.SvcClassName = svcClass.Name
	doc.SvcClassID = svcClass.ID
	svcDomainIDs, err := dbAPI.getServiceInstanceServiceDomainIDs(ctx, &doc)
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	svcInstanceDBO := ServiceInstanceDBO{}

	err = convertServiceInstanceDBO(&doc, &svcInstanceDBO)
	if err != nil {
		return resp, err
	}

	_, err = dbAPI.NamedExec(ctx, queryMap["CreateServiceInstance"], &svcInstanceDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating Service Instance %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	if callback != nil {
		scopeEntity := model.ScopedEntity{
			Doc:     doc,
			EdgeIDs: svcDomainIDs,
		}
		go callback(ctx, scopeEntity)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addServiceInstanceAuditLog(ctx, dbAPI, &doc, CREATE)
	return resp, nil
}

// getServiceInstancesByIDs is used by delta inventory
func (dbAPI *dbObjectModelAPI) getServiceInstancesByIDs(ctx context.Context, svcDomainID string, svcInstanceIDs []string) ([]model.ServiceInstance, error) {
	svcInstances := []model.ServiceInstance{}
	if len(svcInstanceIDs) == 0 {
		return svcInstances, nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcInstances, err
	}
	param := ServiceInstanceQueryParam{SvcInstanceIDs: svcInstanceIDs}
	svcInstanceDBOs := []ServiceInstanceDBO{}
	err = dbAPI.QueryIn(ctx, &svcInstanceDBOs, queryMap["DeltaSelectServiceInstances"], param)
	if err != nil {
		return svcInstances, err
	}
	if len(svcInstanceDBOs) == 0 {
		return svcInstances, nil
	}
	versionMap, err := dbAPI.GetServiceDomainVersions(ctx, []string{svcDomainID})
	if err != nil {
		return svcInstances, err
	}
	invalidSvcInstances := make([]*model.ServiceInstance, 0, len(svcInstanceDBOs))
	svcDomainVersion, ok := versionMap[svcDomainID]
	for _, svcInstanceDBO := range svcInstanceDBOs {
		svcInstance := model.ServiceInstance{}
		err = convertServiceInstance(&svcInstanceDBO, &svcInstance)
		if err != nil {
			return []model.ServiceInstance{}, err
		}
		compResult := 1
		if ok {
			compResult, err = base.CompareVersions(svcInstanceDBO.MinSvcDomainVersion, svcDomainVersion)
			if err != nil {
				return svcInstances, err
			}
		}
		if compResult > 0 {
			// Min Service Domain version is larger than the Service Domain Version
			invalidSvcInstances = append(invalidSvcInstances, &svcInstance)
			continue
		}
		svcInstances = append(svcInstances, svcInstance)
	}
	if len(invalidSvcInstances) > 0 {
		// Async..need a new context
		reqID := base.GetRequestID(ctx)
		adminCtx := base.GetAdminContext(reqID, authContext.TenantID)
		go dbAPI.upsertServiceInstanceEvents(adminCtx, invalidSvcInstances, map[string]string{svcDomainID: svcDomainVersion})
	}
	return svcInstances, nil
}

// CreateServiceInstanceW creates a Service Instance in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateServiceInstanceW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateServiceInstance, &model.ServiceInstanceParam{}, w, r, callback)
}

// UpdateServiceInstance updates the Service Instance
func (dbAPI *dbObjectModelAPI) UpdateServiceInstance(ctx context.Context, i interface{} /* *model.ServiceInstanceParam */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.UpdateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ServiceInstanceParam)
	if !ok {
		return resp, errcode.NewInternalError("UpdateServiceInstance: type error")
	}
	if authContext.ID != "" {
		p.ID = authContext.ID
	}
	updateParam := *p
	tenantID := authContext.TenantID

	if len(updateParam.ID) == 0 {
		return resp, errcode.NewBadRequestError("svcInstanceId")
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityServiceInstance,
		meta.OperationCreate,
		auth.RbacContext{ID: updateParam.ID})
	if err != nil {
		return resp, err
	}
	doc := model.ServiceInstance{}
	err = base.Convert(&updateParam, &doc)
	if err != nil {
		return resp, err
	}
	doc.TenantID = tenantID

	svcClass, err := dbAPI.GetServiceClass(ctx, doc.SvcClassID)
	if err != nil {
		return resp, err
	}
	doc.ServiceClassCommon = svcClass.ServiceClassCommon
	err = dbAPI.ValidateUpdateServiceInstance(ctx, &svcClass, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validating the Service Instance %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	doc.SvcClassName = svcClass.Name
	doc.SvcClassID = svcClass.ID
	svcDomainIDs, err := dbAPI.getServiceInstanceServiceDomainIDs(ctx, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching the Service Domain IDs for scope %s and scope ID %s. Error: %s"), doc.Scope, doc.ScopeID, err.Error())
		return resp, err
	}
	svcInstance, err := dbAPI.GetServiceInstance(ctx, doc.ID)
	if err != nil {
		return resp, err
	}
	err = schema.MergeProperties(svcInstance.Parameters, doc.Parameters, true, false)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in validating the Service Instance %s. Error: %s"), doc.ID, err.Error())
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	doc.Version = float64(epochInNanoSecs)
	doc.CreatedAt = now
	doc.UpdatedAt = now
	svcInstanceDBO := ServiceInstanceDBO{}
	err = convertServiceInstanceDBO(&doc, &svcInstanceDBO)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(ctx, queryMap["UpdateServiceInstance"], &svcInstanceDBO)
	if err != nil {
		return resp, err
	}
	if callback != nil {
		scopeEntity := model.ScopedEntity{
			Doc:     doc,
			EdgeIDs: svcDomainIDs,
		}
		go callback(ctx, scopeEntity)
	}
	resp.ID = doc.ID
	GetAuditlogHandler().addServiceInstanceAuditLog(ctx, dbAPI, &doc, UPDATE)
	return resp, nil
}

// UpdateServiceInstanceW updated the Service Instance in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) UpdateServiceInstanceW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.UpdateServiceInstance, &model.ServiceInstanceParam{}, w, r, callback)
}

// SelectAllServiceInstances returns all the Service Instances matching the query filters
func (dbAPI *dbObjectModelAPI) SelectAllServiceInstances(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceInstanceQueryParam) (model.ServiceInstanceListPayload, error) {
	resp := model.ServiceInstanceListPayload{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}

	query, err := buildQueryWithTableAlias(entityTypeServiceInstance, queryMap["SelectServiceInstances"], entitiesQueryParam, orderByNameID, serviceInstanceModelAlias, serviceInstanceModelAliasMap)
	if err != nil {
		return resp, err
	}
	totalCount := 0
	param := ServiceInstanceQueryParam{}
	err = base.Convert(queryParam, &param)
	if err != nil {
		return resp, err
	}
	filterIDsParam, err := dbAPI.getPermittedFilterIDsParam(ctx, queryParam.ScopeID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the permitted filter IDs for Service Instances. Error: %s"), err.Error())
		return resp, err
	}
	param.FilterIDsParam = filterIDsParam
	param.TenantID = authContext.TenantID
	svcInstances := []model.ServiceInstance{}
	_, err = dbAPI.NotPagedQueryInEx(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		svcInstanceDBO := dbObjPtr.(*ServiceInstanceDBO)
		svcInstance := model.ServiceInstance{}
		err = convertServiceInstance(svcInstanceDBO, &svcInstance)
		if err != nil {
			return err
		}
		if svcInstanceDBO.TotalCount != nil && totalCount == 0 {
			totalCount = *svcInstanceDBO.TotalCount
		}
		svcInstances = append(svcInstances, svcInstance)
		return nil
	}, query, param, ServiceInstanceDBO{})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching Service Instances. Error: %s"), err.Error())
		return resp, err
	}
	// TODO redaction by loading the class schema
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeServiceInstance})
	resp.EntityListResponsePayload = entityListResponsePayload
	resp.SvcInstanceList = svcInstances
	return resp, nil
}

// SelectAllServiceInstancesW returns all the Service Instances matching the query filters, writing the output to the writer
func (dbAPI *dbObjectModelAPI) SelectAllServiceInstancesW(ctx context.Context, w io.Writer, r *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(r)
	queryParam := &model.ServiceInstanceQueryParam{}
	err := base.GetHTTPQueryParams(r, queryParam)
	if err != nil {
		return err
	}
	response, err := dbAPI.SelectAllServiceInstances(ctx, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(response)
}

// GetServiceInstance returns the Service Instance with the given ID
func (dbAPI *dbObjectModelAPI) GetServiceInstance(ctx context.Context, id string) (model.ServiceInstance, error) {
	resp := model.ServiceInstance{}
	if id == "" {
		return resp, errcode.NewBadRequestError("id")
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	filterIDsParam, err := dbAPI.getPermittedFilterIDsParam(ctx, "")
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the permitted filter IDs. Error: %s"), err.Error())
		return resp, err
	}
	param := ServiceInstanceQueryParam{BaseModelDBO: model.BaseModelDBO{TenantID: authContext.TenantID, ID: id}, FilterIDsParam: filterIDsParam}
	svcInstanceDBOs := []ServiceInstanceDBO{}
	err = dbAPI.QueryIn(ctx, &svcInstanceDBOs, queryMap["SelectServiceInstances"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting Service Instance %s. Error: %s"), id, err.Error())
		return resp, err
	}
	if len(svcInstanceDBOs) != 1 {
		return resp, errcode.NewRecordNotFoundError("id")
	}
	svcInstance := model.ServiceInstance{}
	err = convertServiceInstance(&svcInstanceDBOs[0], &svcInstance)
	if err != nil {
		return resp, err
	}
	return svcInstance, nil
}

// GetServiceInstance returns the Service Instance with the given ID, writing the output to the writer
func (dbAPI *dbObjectModelAPI) GetServiceInstanceW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	svcInstance, err := dbAPI.GetServiceInstance(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, svcInstance)
}

// SelectServiceInstanceStatuss returns the Service Instance status for each Service Domain
func (dbAPI *dbObjectModelAPI) SelectServiceInstanceStatuss(ctx context.Context, id string, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceInstanceStatusQueryParam) (model.ServiceInstanceStatusListPayload, error) {
	resp := model.ServiceInstanceStatusListPayload{}
	_, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	svcInstance, err := dbAPI.GetServiceInstance(ctx, id)
	if err != nil {
		return resp, err
	}
	eventQueryParam := ServiceInstanceStatusEventQueryParam{
		ServiceInstanceCommon: svcInstance.ServiceInstanceCommon,
		SvcInstanceID:         svcInstance.ID,
		SvcDomainID:           queryParam.SvcDomainID,
	}
	ePathTemplate := model.ServiceInstanceStatusServiceDomainScopedEventPath
	if eventQueryParam.Scope == model.ServiceClassProjectScope {
		ePathTemplate = model.ServiceInstanceStatusProjectScopedEventPath
		eventQueryParam.ProjectID = svcInstance.ScopeID
	}
	ePath, _, err := model.GenerateEventQueryPath(ePathTemplate, eventQueryParam)
	if err != nil {
		return resp, err
	}
	eventFilter := model.EventFilter{
		Path: ePath,
		Keys: map[string]string{
			"type": "STATUS",
		},
	}
	if entitiesQueryParam != nil {
		eventFilter.Start = entitiesQueryParam.GetPageIndex()
		eventFilter.Size = entitiesQueryParam.GetPageSize()
	}

	events, err := dbAPI.QueryEvents(ctx, eventFilter)
	if err != nil {
		return resp, err
	}
	svcInstanceStatuss := make([]model.ServiceInstanceStatus, 0, len(events))
	for _, event := range events {
		if event.State != string(model.ServiceInstanceProvisiongState) &&
			event.State != string(model.ServiceInstanceProvisionedState) &&
			event.State != string(model.ServiceInstanceFailedState) {
			glog.Warningf(base.PrefixRequestID(ctx, "Invalid event state %s found for %s"), event.State, event.Path)
			continue
		}
		if svcInstance.UpdatedAt.After(event.Timestamp) {
			// Event time is before
			// Adding back a scope can cause this
			continue
		}
		ePathComps := model.ExtractEventPathComponentsN(event.Path, 1)
		if ePathComps == nil || len(ePathComps.SvcDomainID) == 0 {
			glog.Warningf(base.PrefixRequestID(ctx, "Invalid event %s found"), event.Path)
			continue
		}
		svcEventStatus := model.ServiceInstanceStatus{
			ServiceInstanceState: model.ServiceInstanceState{
				SvcDomainID: ePathComps.SvcDomainID,
				State:       model.ServiceInstanceStateType(event.State),
				Description: event.Message,
			},
			SvcInstanceID: id,
			CreatedAt:     event.Timestamp,
			UpdatedAt:     event.Timestamp,
		}
		err = base.Convert(event.Properties, &svcEventStatus.Properties)
		if err != nil {
			glog.Warningf(base.PrefixRequestID(ctx, "Invalid event properties for %s. Error: %s"), event.Path, err.Error())
			// Ignore
		}
		svcInstanceStatuss = append(svcInstanceStatuss, svcEventStatus)
	}
	// Total count is not accurate.
	// TODO later. this requires event API fix
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: len(events), EntityType: entityTypeServiceInstance})
	resp.EntityListResponsePayload = entityListResponsePayload
	resp.SvcInstanceStatusList = svcInstanceStatuss
	return resp, nil
}

// SelectServiceInstanceStatussW returns the Service Instance status for each Service Domain, writing the output to the writer
func (dbAPI *dbObjectModelAPI) SelectServiceInstanceStatussW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	queryParam := &model.ServiceInstanceStatusQueryParam{}
	err := base.GetHTTPQueryParams(req, queryParam)
	if err != nil {
		return err
	}
	response, err := dbAPI.SelectServiceInstanceStatuss(ctx, id, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, response)
}

// DeleteServiceInstance deletes the Service Instance with the given ID
func (dbAPI *dbObjectModelAPI) DeleteServiceInstance(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponseV2{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityServiceInstance,
		meta.OperationDelete,
		auth.RbacContext{ID: id})
	if err != nil {
		return resp, err
	}
	svcInstance, err := dbAPI.GetServiceInstance(ctx, id)
	if err != nil {
		return resp, err
	}
	svcDomainIDs, err := dbAPI.getServiceInstanceServiceDomainIDs(ctx, &svcInstance)
	if err != nil {
		return resp, err
	}
	scopeEntity := model.ScopedEntity{
		Doc:     svcInstance,
		EdgeIDs: svcDomainIDs,
	}
	GetAuditlogHandler().addServiceInstanceAuditLog(ctx, dbAPI, &svcInstance, DELETE)
	return DeleteEntityV2(ctx, dbAPI, "service_instance_model", "id", id, scopeEntity, callback)
}

// DeleteServiceInstanceW deletes the Service Instance with the given ID, write the response to the writer
func (dbAPI *dbObjectModelAPI) DeleteServiceInstanceW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, dbAPI.DeleteServiceInstance, id, w, callback)
}

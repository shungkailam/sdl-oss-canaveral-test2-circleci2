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
)

const (
	entityTypeServiceBinding       = "servicebinding"
	entityTypeServiceBindingStatus = "servicebindingstatus"
	serviceBindingModelAlias       = "sbm"
	serviceBindingStatusModelAlias = "bsm"
	serviceBindingEventBatchSize   = 20
)

var (
	// Lead time given to the push timestamp
	// Delta inventory API which is invoked every 5 minutes.
	// Event will be stale in <= 1 minute since events are pushed every minute
	svcBindingPushLeadTime = time.Duration(time.Minute * 6)
)

func init() {
	queryMap["CreateServiceBinding"] = `INSERT INTO service_binding_model (id, tenant_id, name, description, svc_class_id, resource_type, svc_domain_resource_id, project_resource_id, parameters, version, created_at, updated_at) VALUES (:id, :tenant_id, :name, :description, :svc_class_id, :resource_type, :svc_domain_resource_id, :project_resource_id, :parameters, :version, :created_at, :updated_at)`
	queryMap["SelectServiceBindings"] = `SELECT sbm.*, scm.name as svc_class_name, scm.type, scm.svc_version, scm.scope, scm.min_svc_domain_version, COUNT(sbm.*) OVER() as total_count FROM service_binding_model sbm, service_class_model scm WHERE (sbm.tenant_id = :tenant_id) AND (sbm.svc_class_id = scm.id) AND (:id = '' OR sbm.id = :id) AND (:svc_class_id = '' OR sbm.svc_class_id = :svc_class_id) AND (:resource_type = '' OR sbm.resource_type = :resource_type) AND (sbm.resource_type is NULL OR (:filter_svc_domains = false AND sbm.svc_domain_resource_id IS NOT NULL) OR (:filter_svc_domains = true AND sbm.svc_domain_resource_id IN (:svc_domain_ids)) OR (:filter_projects = false AND sbm.project_resource_id IS NOT NULL) OR  (:filter_projects = true AND sbm.project_resource_id IN (:project_ids)))`
	queryMap["DeltaSelectServiceBindings"] = `SELECT sbm.*, scm.name as svc_class_name, scm.type, scm.svc_version, scm.scope, scm.min_svc_domain_version FROM service_binding_model sbm, service_class_model scm WHERE (sbm.svc_class_id = scm.id) AND (sbm.id IN (:svc_binding_ids))`

	orderByHelper.Setup(entityTypeServiceBinding, []string{"id", "name", "version", "created_at", "updated_at"})
	orderByHelper.Setup(entityTypeServiceBindingStatus, []string{})
}

// ServiceBindingDBO is the DB model for Service Binding
type ServiceBindingDBO struct {
	model.BaseModelDBO
	ServiceInstanceCommonDBO
	Name                string          `json:"name" db:"name"`
	Description         string          `json:"description" db:"description"`
	SvcInstanceID       string          `json:"svcInstanceId" db:"svc_instance_id"`
	ResourceType        *string         `json:"bindResourceType" db:"resource_type"`
	SvcDomainResourceID *string         `json:"svcDomainResourceId" db:"svc_domain_resource_id"`
	ProjectResourceID   *string         `json:"projectResourceId" db:"project_resource_id"`
	Parameters          *types.JSONText `json:"parameters" db:"parameters"`
}

// ServiceBindingQueryParam is the query parameter for Service Binding
type ServiceBindingQueryParam struct {
	model.BaseModelDBO
	FilterIDsParam
	SvcClassID       string   `json:"svcClassId" db:"svc_class_id"`
	BindResourceType string   `json:"bindResourceType" db:"resource_type"`
	BindResourceID   string   `json:"bindResourceId" db:"resource_id"`
	SvcBindingIDs    []string `json:"svcBindingIds" db:"svc_binding_ids"`
}

// ServiceBindingStatusEventQueryParam is the query parameters values used for
// Service Binding status event path
type ServiceBindingStatusEventQueryParam struct {
	model.ServiceClassCommon
	SvcClassID    string `json:"svcClassId"`
	SvcBindingID  string `json:"svcBindingId"`
	SvcInstanceID string `json:"svcInstanceId"`
	SvcDomainID   string `json:"svcDomainId"`
	ProjectID     string `json:"projectId"`
}

func (dbAPI *dbObjectModelAPI) ValidateCreateServiceBinding(ctx context.Context, svcClass *model.ServiceClass, svcBindingParam *model.ServiceBindingParam) error {
	if svcBindingParam.Parameters == nil {
		svcBindingParam.Parameters = map[string]interface{}{}
	}
	if svcClass.Schemas.SvcBinding.Create.Parameters == nil {
		svcClass.Schemas.SvcBinding.Create.Parameters = map[string]interface{}{}
	}
	err := schema.ValidateSchemaMap(ctx, svcClass.Schemas.SvcBinding.Create.Parameters, svcBindingParam.Parameters)
	if err != nil {
		return err
	}
	if len(svcBindingParam.SvcClassID) == 0 {
		return errcode.NewBadRequestError("svcClassId")
	}
	if !svcClass.Bindable {
		return errcode.NewBadRequestExError("svcBinding", "Unbindable Service Class")
	}
	return nil
}

func convertServiceBinding(svcBindingDBO *ServiceBindingDBO, svcBinding *model.ServiceBinding) error {
	err := base.Convert(svcBindingDBO, svcBinding)
	if err != nil {
		return err
	}
	if svcBindingDBO.ResourceType != nil {
		if *svcBindingDBO.ResourceType == string(model.ServiceBindingServiceDomainResource) {
			if svcBindingDBO.SvcDomainResourceID == nil {
				// Not supposed to happen.
				// This prevents panic
				return errcode.NewInternalError(fmt.Sprintf("Resource ID for Service Domain resource is not set in Service Binding %s", svcBinding.ID))
			}
			svcBinding.BindResource = &model.ServiceBindingResource{Type: model.ServiceBindingServiceDomainResource, ID: *svcBindingDBO.SvcDomainResourceID}
		} else if *svcBindingDBO.ResourceType == string(model.ServiceBindingProjectResource) {
			if svcBindingDBO.ProjectResourceID == nil {
				// Not supposed to happen.
				// This prevents panic
				return errcode.NewInternalError(fmt.Sprintf("Resource ID for Project resource is not set in Service Binding %s", svcBinding.ID))
			}
			svcBinding.BindResource = &model.ServiceBindingResource{Type: model.ServiceBindingProjectResource, ID: *svcBindingDBO.ProjectResourceID}
		} else {
			return errcode.NewBadRequestError("resourceType")
		}
	}
	return nil
}

func populateServiceDomainDBO(param *model.ServiceBindingParam, svcBindingDBO *ServiceBindingDBO) error {
	svcBindingDBO.SvcClassID = param.SvcClassID
	bindResourcePtr := param.BindResource
	if bindResourcePtr != nil {
		bindResource := *bindResourcePtr
		svcBindingDBO.ResourceType = base.StringPtr(string(bindResource.Type))
		if bindResource.Type == model.ServiceBindingServiceDomainResource {
			svcBindingDBO.SvcDomainResourceID = &bindResource.ID
		} else if bindResource.Type == model.ServiceBindingProjectResource {
			svcBindingDBO.ProjectResourceID = &bindResource.ID
		} else {
			return errcode.NewBadRequestError("resourceType")
		}
	}
	return nil
}

// getServiceBindingsByIDs is used by delta inventory
func (dbAPI *dbObjectModelAPI) getServiceBindingsByIDs(ctx context.Context, svcDomainID string, svcBindingIDs []string) ([]model.ServiceBinding, error) {
	svcBindings := []model.ServiceBinding{}
	if len(svcBindingIDs) == 0 {
		return svcBindings, nil
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return svcBindings, err
	}
	param := ServiceBindingQueryParam{SvcBindingIDs: svcBindingIDs}
	svcBindingDBOs := []ServiceBindingDBO{}
	err = dbAPI.QueryIn(ctx, &svcBindingDBOs, queryMap["DeltaSelectServiceBindings"], param)
	if err != nil {
		return []model.ServiceBinding{}, err
	}
	if len(svcBindingDBOs) == 0 {
		return svcBindings, nil
	}
	versionMap, err := dbAPI.GetServiceDomainVersions(ctx, []string{svcDomainID})
	if err != nil {
		return svcBindings, err
	}
	invalidSvcBindings := make([]*model.ServiceBinding, 0, len(svcBindingDBOs))
	svcVersion, ok := versionMap[svcDomainID]
	for _, svcBindingDBO := range svcBindingDBOs {
		svcBinding := model.ServiceBinding{}
		err = convertServiceBinding(&svcBindingDBO, &svcBinding)
		if err != nil {
			return []model.ServiceBinding{}, err
		}
		compResult := 1
		if ok {
			compResult, err = base.CompareVersions(svcBindingDBO.MinSvcDomainVersion, svcVersion)
			if err != nil {
				return svcBindings, err
			}
		}
		if compResult > 0 {
			// Min Service Domain version is larger than the Service Domain Version
			invalidSvcBindings = append(invalidSvcBindings, &svcBinding)
			continue
		}
		svcBindings = append(svcBindings, svcBinding)
	}
	if len(invalidSvcBindings) > 0 {
		// Async..need a new context
		reqID := base.GetRequestID(ctx)
		adminCtx := base.GetAdminContext(reqID, authContext.TenantID)
		go dbAPI.upsertServiceBindingEvents(adminCtx, invalidSvcBindings, map[string]string{svcDomainID: svcVersion})
	}
	return svcBindings, nil
}

// upsertServiceBindingEvents upserts events corresponding to min Service Domain version requirement
func (dbAPI *dbObjectModelAPI) upsertServiceBindingEvents(ctx context.Context, svcBindings []*model.ServiceBinding, svcDomainIDs map[string]string) error {
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
	for i := range svcBindings {
		svcBinding := svcBindings[i]
		bindResource := svcBinding.BindResource
		for svcDomainID, svcDomainVersion := range svcDomainIDs {
			glog.Warningf(base.PrefixRequestID(ctx, "Service Binding %+v is not supported on Service Domain %s"), svcBinding, svcDomainID)
			pushTimeStamp, ok := svcDomainPushTimeStamps[svcDomainID]
			if !ok {
				// UTC time string
				pushTimeStamp = base.RoundedNow()
			}
			pushTimeStamp = pushTimeStamp.Add(svcBindingPushLeadTime)

			// Event path params for Service Binding
			eventQueryParam := ServiceBindingStatusEventQueryParam{
				ServiceClassCommon: svcBinding.ServiceClassCommon,
				SvcClassID:         svcBinding.SvcClassID,
				SvcBindingID:       svcBinding.ID,
				SvcDomainID:        svcDomainID,
				SvcInstanceID:      model.ZeroUUID,
			}
			ePathTemplate := model.ServiceBindingStatusServiceDomainScopedEventPath
			if bindResource != nil && bindResource.Type == model.ServiceBindingProjectResource {
				ePathTemplate = model.ServiceBindingStatusProjectScopedEventPath
				eventQueryParam.ProjectID = bindResource.ID
			}
			ePath, err := model.GenerateEventUpsertPath(ePathTemplate, eventQueryParam)
			if err != nil {
				glog.Warningf(base.PrefixRequestID(ctx, "Error in generating event path using the template %s with values %+v. Error: %s"), ePathTemplate, eventQueryParam, err.Error())
				errMsgs = append(errMsgs, err.Error())
			} else {
				var message string
				if svcDomainVersion == "" {
					// Unknown version
					message = fmt.Sprintf("Minimum Service Domain version %s is required by the Service Binding", svcBinding.MinSvcDomainVersion)
				} else {
					message = fmt.Sprintf("Minimum Service Domain version %s is required by the Service Binding. Current Service Domain version is %s", svcBinding.MinSvcDomainVersion, svcDomainVersion)
				}
				event := model.Event{
					Path:      ePath,
					Type:      "ALERT",
					Severity:  "CRITICAL",
					State:     string(model.ServiceBindingFailedState),
					Message:   message,
					Timestamp: base.RoundedNow(),
					Properties: map[string]string{
						pushTimeStampLabel: pushTimeStamp.String(),
					},
					IsInfraEntity: true,
				}
				eventUpsertReq.Events = append(eventUpsertReq.Events, event)
				if len(eventUpsertReq.Events) >= serviceBindingEventBatchSize {
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

func (dbAPI *dbObjectModelAPI) getServiceBindingServiceDomainIDs(ctx context.Context, svcBinding *model.ServiceBinding) ([]string, error) {
	var svcDomainIDs []string
	var err error
	bindResource := svcBinding.BindResource
	if bindResource == nil {
		svcDomainIDs, err = dbAPI.SelectAllServiceDomainIDs(ctx)
	} else if bindResource.Type == model.ServiceBindingServiceDomainResource {
		svcDomainIDs = append(svcDomainIDs, bindResource.ID)
	} else if bindResource.Type == model.ServiceBindingProjectResource {
		// Get the edges where the project is available
		svcDomainIDs, err = dbAPI.GetProjectsEdgeClusters(ctx, []string{bindResource.ID})
	}
	if err != nil {
		return svcDomainIDs, err
	}
	if len(svcDomainIDs) == 0 {
		return svcDomainIDs, nil
	}
	invalidSvcDomainIDs := map[string]string{}
	svcDomainIDs, err = dbAPI.FilterServiceDomainIDsByVersion(ctx, svcBinding.MinSvcDomainVersion, svcDomainIDs, func(svcDomainID, svcDomainVersion string, compResult int) bool {
		if compResult <= 0 {
			// minSvcDomainVersion is <= Service Domain version
			// add the ID
			return true
		}
		glog.Warningf(base.PrefixRequestID(ctx, "Service Binding %+v is not supported on Service Domain %s"), svcBinding, svcDomainID)
		invalidSvcDomainIDs[svcDomainID] = svcDomainVersion
		return false
	})
	if len(invalidSvcDomainIDs) > 0 {
		dbAPI.upsertServiceBindingEvents(ctx, []*model.ServiceBinding{svcBinding}, invalidSvcDomainIDs)
		// ignore error
	}
	return svcDomainIDs, nil
}

// CreateServiceBinding creates a Service Binding in the DB
func (dbAPI *dbObjectModelAPI) CreateServiceBinding(ctx context.Context, i interface{} /* *model.ServiceBindingParam */, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.CreateDocumentResponseV2{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	p, ok := i.(*model.ServiceBindingParam)
	if !ok {
		return resp, errcode.NewInternalError("CreateServiceBinding: type error")
	}
	createParam := *p
	tenantID := authContext.TenantID
	if !base.CheckID(createParam.ID) {
		createParam.ID = base.GetUUID()
		glog.Infof(base.PrefixRequestID(ctx, "CreateServiceBinding doc.ID was invalid, update it to %s\n"), createParam.ID)
	}

	err = auth.CheckRBAC(
		authContext,
		meta.EntityServiceBinding,
		meta.OperationCreate,
		auth.RbacContext{ID: createParam.ID})
	if err != nil {
		return resp, err
	}

	if len(createParam.SvcClassID) == 0 {
		return resp, errcode.NewBadRequestError("svcClassId")
	}
	svcClass, err := dbAPI.GetServiceClass(ctx, createParam.SvcClassID)
	if err != nil {
		return resp, err
	}

	err = dbAPI.ValidateCreateServiceBinding(ctx, &svcClass, &createParam)
	if err != nil {
		return resp, err
	}

	svcBindingDBO := ServiceBindingDBO{}
	err = base.Convert(&createParam, &svcBindingDBO)
	if err != nil {
		return resp, err
	}
	now := base.RoundedNow()
	epochInNanoSecs := now.UnixNano()
	svcBindingDBO.TenantID = tenantID
	svcBindingDBO.Scope = string(svcClass.Scope)
	svcBindingDBO.SvcVersion = svcClass.SvcVersion
	svcBindingDBO.SvcClassName = svcClass.Name
	svcBindingDBO.Type = svcClass.Type
	svcBindingDBO.MinSvcDomainVersion = svcClass.MinSvcDomainVersion
	svcBindingDBO.Version = float64(epochInNanoSecs)
	svcBindingDBO.CreatedAt = now
	svcBindingDBO.UpdatedAt = now

	err = populateServiceDomainDBO(&createParam, &svcBindingDBO)
	if err != nil {
		return resp, err
	}
	svcBinding := model.ServiceBinding{}
	err = convertServiceBinding(&svcBindingDBO, &svcBinding)
	if err != nil {
		return resp, err
	}
	svcDomainIDs, err := dbAPI.getServiceBindingServiceDomainIDs(ctx, &svcBinding)
	if err != nil {
		return resp, err
	}
	_, err = dbAPI.NamedExec(ctx, queryMap["CreateServiceBinding"], &svcBindingDBO)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in creating Service Binding %s. Error: %s"), createParam.ID, err.Error())
		return resp, err
	}
	if callback != nil {
		scopedEntity := model.ScopedEntity{
			Doc:     svcBinding,
			EdgeIDs: svcDomainIDs,
		}
		go callback(ctx, scopedEntity)
	}
	GetAuditlogHandler().addServiceBindingAuditLog(ctx, dbAPI, &svcBinding, CREATE)
	resp.ID = createParam.ID
	return resp, nil
}

// CreateServiceBindingW creates a Service Binding in the DB, write output into writer
func (dbAPI *dbObjectModelAPI) CreateServiceBindingW(ctx context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	return base.CreateW(ctx, dbAPI.CreateServiceBinding, &model.ServiceBindingParam{}, w, r, callback)
}

// SelectAllServiceBindings returns all the Service Bindings matching the query filters
func (dbAPI *dbObjectModelAPI) SelectAllServiceBindings(ctx context.Context, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceBindingQueryParam) (model.ServiceBindingListPayload, error) {
	resp := model.ServiceBindingListPayload{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	query, err := buildQueryWithTableAlias(entityTypeServiceBinding, queryMap["SelectServiceBindings"], entitiesQueryParam, orderByNameID, serviceBindingModelAlias, nil)
	if err != nil {
		return resp, err
	}
	totalCount := 0
	param := ServiceBindingQueryParam{}
	err = base.Convert(queryParam, &param)
	if err != nil {
		return resp, err
	}
	filterIDsParam, err := dbAPI.getPermittedFilterIDsParam(ctx, param.BindResourceID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the permitted filter IDs for Service Bindings. Error: %s"), err.Error())
		return resp, err
	}
	param.FilterIDsParam = filterIDsParam
	param.TenantID = authContext.TenantID
	svcBindings := []model.ServiceBinding{}
	_, err = dbAPI.NotPagedQueryInEx(ctx, base.StartPageToken, base.MaxRowsLimit, func(dbObjPtr interface{}) error {
		svcBindingDBO := dbObjPtr.(*ServiceBindingDBO)
		svcBinding := model.ServiceBinding{}
		err = convertServiceBinding(svcBindingDBO, &svcBinding)
		if err != nil {
			return err
		}
		if svcBindingDBO.TotalCount != nil && totalCount == 0 {
			totalCount = *svcBindingDBO.TotalCount
		}
		svcBindings = append(svcBindings, svcBinding)
		return nil
	}, query, param, ServiceBindingDBO{})
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching Service Bindings. Error: %s"), err.Error())
		return resp, err
	}
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: totalCount, EntityType: entityTypeServiceBinding})
	resp.EntityListResponsePayload = entityListResponsePayload
	resp.SvcBindingList = svcBindings
	return resp, nil
}

// SelectAllServiceBindingsW returns all the Service Bindings matching the query filters, writing the output to the writer
func (dbAPI *dbObjectModelAPI) SelectAllServiceBindingsW(ctx context.Context, w io.Writer, r *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(r)
	queryParam := &model.ServiceBindingQueryParam{}
	err := base.GetHTTPQueryParams(r, queryParam)
	if err != nil {
		return err
	}
	resp, err := dbAPI.SelectAllServiceBindings(ctx, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	return json.NewEncoder(w).Encode(resp)
}

// GetServiceBinding returns the Service Binding with the given ID
func (dbAPI *dbObjectModelAPI) GetServiceBinding(ctx context.Context, id string) (model.ServiceBinding, error) {
	resp := model.ServiceBinding{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	if len(id) == 0 {
		return resp, errcode.NewBadRequestError("id")
	}
	filterIDsParam, err := dbAPI.getPermittedFilterIDsParam(ctx, "")
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the permitted filter IDs for Service Bindings. Error: %s"), err.Error())
		return resp, err
	}
	param := ServiceBindingQueryParam{BaseModelDBO: model.BaseModelDBO{TenantID: authContext.TenantID, ID: id}, FilterIDsParam: filterIDsParam}
	svcBindingDBOs := []ServiceBindingDBO{}
	err = dbAPI.QueryIn(ctx, &svcBindingDBOs, queryMap["SelectServiceBindings"], param)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in fetching Service Binding %s. Error: %s"), id, err.Error())
		return resp, err
	}
	if len(svcBindingDBOs) == 0 {
		return resp, errcode.NewRecordNotFoundError(id)
	}
	svcBinding := model.ServiceBinding{}
	err = convertServiceBinding(&svcBindingDBOs[0], &svcBinding)
	if err != nil {
		return resp, err
	}
	return svcBinding, nil
}

// GetServiceBinding returns the Service Binding with the given ID, writing the output to the writer
func (dbAPI *dbObjectModelAPI) GetServiceBindingW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	svcBinding, err := dbAPI.GetServiceBinding(ctx, id)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, svcBinding)
}

// SelectServiceBindingStatuss returns the Service Bindings with the given Service Binding ID
func (dbAPI *dbObjectModelAPI) SelectServiceBindingStatuss(ctx context.Context, id string, entitiesQueryParam *model.EntitiesQueryParam, queryParam *model.ServiceBindingStatusQueryParam) (model.ServiceBindingStatusListPayload, error) {
	resp := model.ServiceBindingStatusListPayload{}
	_, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	svcInstanceQueryParam := model.ServiceInstanceQueryParam{}
	err = base.Convert(queryParam, &svcInstanceQueryParam)
	if err != nil {
		return resp, err
	}
	svcBinding, err := dbAPI.GetServiceBinding(ctx, id)
	if err != nil {
		return resp, err
	}
	svcInstanceQueryParam.SvcClassID = svcBinding.SvcClassID
	svcInstanceResp, err := dbAPI.SelectAllServiceInstances(ctx, &model.EntitiesQueryParam{}, &svcInstanceQueryParam)
	if err != nil {
		return resp, err
	}
	svcInstanceIDsMap := make(map[string]bool, svcInstanceResp.TotalCount+1)
	for _, svcInstance := range svcInstanceResp.SvcInstanceList {
		svcInstanceIDsMap[svcInstance.ID] = true
	}
	// Query for all Service Instances
	eventQueryParam := ServiceBindingStatusEventQueryParam{
		ServiceClassCommon: svcBinding.ServiceClassCommon,
		SvcClassID:         svcBinding.SvcClassID,
		SvcBindingID:       svcBinding.ID,
		SvcDomainID:        queryParam.SvcDomainID,
	}
	ePathTemplate := model.ServiceBindingStatusServiceDomainScopedEventPath
	if eventQueryParam.Scope == model.ServiceClassProjectScope {
		ePathTemplate = model.ServiceBindingStatusProjectScopedEventPath
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
	svcBindingStatuss := make([]model.ServiceBindingStatus, 0, len(events))
	for _, event := range events {
		if event.State != string(model.ServiceBindingProvisiongState) &&
			event.State != string(model.ServiceBindingProvisionedState) &&
			event.State != string(model.ServiceBindingFailedState) {
			glog.Warningf(base.PrefixRequestID(ctx, "Invalid event state %s found for %s"), event.State, event.Path)
			continue
		}
		if svcBinding.UpdatedAt.After(event.Timestamp) {
			// Event time is before
			// Adding back a scope can cause this
			continue
		}
		ePathComps := model.ExtractEventPathComponentsN(event.Path, 4)
		if ePathComps == nil || len(ePathComps.SvcDomainID) == 0 {
			glog.Warningf(base.PrefixRequestID(ctx, "Invalid event %s found"), event.Path)
			continue
		}

		if !svcInstanceIDsMap[ePathComps.SvcInstanceID] {
			// Binding has no instance to report to.
			// It is pending binding
			glog.Warningf(base.PrefixRequestID(ctx, "Service Instance %s is not found for event %s"), ePathComps.SvcInstanceID, event.Path)
			continue
		}
		svcEventStatus := model.ServiceBindingStatus{
			ServiceBindingState: model.ServiceBindingState{
				SvcDomainID: ePathComps.SvcDomainID,
				State:       model.ServiceBindingStateType(event.State),
				Description: event.Message,
			},
			SvcBindingID: id,
			CreatedAt:    event.Timestamp,
			UpdatedAt:    event.Timestamp,
		}
		if svcEventStatus.State == model.ServiceBindingProvisionedState && event.Properties != nil {
			if result, ok := event.Properties["bindResult"]; ok && len(result) > 0 {
				svcEventStatus.BindResult = &model.ServiceBindingResult{}
				if err = base.ConvertFromJSON([]byte(result), &svcEventStatus.BindResult); err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Error in unmarshalling JSON into bind result for event %+v. Error: %s"), event, err.Error())
					svcEventStatus.State = model.ServiceBindingProvisiongState
					// Ignore
				}
			}
		}
		svcBindingStatuss = append(svcBindingStatuss, svcEventStatus)
	}
	// Total count is not accurate.
	// TODO later. this requires event API fix
	entityListResponsePayload := makeEntityListResponsePayload(entitiesQueryParam, &ListQueryInfo{TotalCount: len(svcBindingStatuss), EntityType: entityTypeServiceBinding})
	resp.EntityListResponsePayload = entityListResponsePayload
	resp.SvcBindingStatusList = svcBindingStatuss
	return resp, nil
}

// SelectServiceBindingStatussW returns the Service Binding Status with the given Service Binding ID, writing the output to the writer
func (dbAPI *dbObjectModelAPI) SelectServiceBindingStatussW(ctx context.Context, id string, w io.Writer, req *http.Request) error {
	entitiesQueryParam := model.GetEntitiesQueryParam(req)
	queryParam := &model.ServiceBindingStatusQueryParam{}
	err := base.GetHTTPQueryParams(req, queryParam)
	response, err := dbAPI.SelectServiceBindingStatuss(ctx, id, entitiesQueryParam, queryParam)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, response)
}

func (dbAPI *dbObjectModelAPI) DeleteServiceBinding(ctx context.Context, id string, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.DeleteDocumentResponseV2{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	err = auth.CheckRBAC(
		authContext,
		meta.EntityServiceBinding,
		meta.OperationDelete,
		auth.RbacContext{ID: id})
	if err != nil {
		return resp, err
	}
	svcBinding, err := dbAPI.GetServiceBinding(ctx, id)
	if err != nil {
		return resp, err
	}
	svcDomainIDs, err := dbAPI.getServiceBindingServiceDomainIDs(ctx, &svcBinding)
	if err != nil {
		return resp, err
	}
	scopeEntity := model.ScopedEntity{
		Doc:     svcBinding,
		EdgeIDs: svcDomainIDs,
	}
	GetAuditlogHandler().addServiceBindingAuditLog(ctx, dbAPI, &svcBinding, DELETE)
	return DeleteEntityV2(ctx, dbAPI, "service_binding_model", "id", id, scopeEntity, callback)
}

// DeleteServiceBindingW deletes the Service Binding with the given ID, write the response to the writer
func (dbAPI *dbObjectModelAPI) DeleteServiceBindingW(ctx context.Context, id string, w io.Writer, callback func(context.Context, interface{}) error) error {
	return base.DeleteW(ctx, dbAPI.DeleteServiceBinding, id, w, callback)
}

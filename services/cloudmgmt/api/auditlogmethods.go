package api

import (
	"cloudservices/common/model"
	"context"
)

func (handler *AuditlogHandler) addApplicationAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.Application, logType string) {
	resourceName := doc.Name
	resourceID := doc.ID
	edgeIDs := doc.EdgeIDs
	var edgeNames []string
	for _, edgeID := range edgeIDs {
		edge, err := dbAPI.GetEdge(context, edgeID)
		if err == nil {
			edgeNames = append(edgeNames, edge.Name)
		}
	}
	projectName, err := dbAPI.GetProjectName(context, doc.ProjectID)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, context, APPLICATION, logType, projectName, doc.ProjectID, resourceName, resourceID, edgeNames, edgeIDs, "")
	}
}

func (handler *AuditlogHandler) addDataSourceAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.DataSource, logType string) {
	edge, err := dbAPI.GetEdge(context, doc.EdgeID)
	if err == nil {
		dataSourceName := doc.Name
		dataSourceNameID := doc.ID
		serviceDomainID := *edge.ShortID
		serviceDomainName := edge.Name
		GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, DATA_SOURCE, logType, "", "", dataSourceName, dataSourceNameID, []string{serviceDomainName}, []string{serviceDomainID}, "")
	}
}

func (handler *AuditlogHandler) addProjectAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.Project, logType string) {
	resourceName := doc.Name
	resourceID := doc.ID
	edgeIDs := doc.EdgeIDs
	var edgeNames []string
	for _, edgeID := range edgeIDs {
		edge, err := dbAPI.GetEdge(context, edgeID)
		if err == nil {
			edgeNames = append(edgeNames, edge.Name)
		}
	}
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, PROJECT, logType,
		doc.Name, doc.ID,
		resourceName, resourceID,
		edgeNames, edgeIDs, "")
}

func (handler *AuditlogHandler) addMLModelAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.MLModelMetadata, logType string) {
	project, err := dbAPI.GetProject(context, doc.ProjectID)
	if err == nil {
		edgeIDs := project.EdgeIDs
		var edgeNames []string
		for _, edgeID := range edgeIDs {
			edge, err := dbAPI.GetEdge(context, edgeID)
			if err == nil {
				edgeNames = append(edgeNames, edge.Name)
			}
		}
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, context, ML_MODEL, logType, project.Name, project.ID, doc.Name, doc.ID, edgeNames, edgeIDs, "")
	}
}

func (handler *AuditlogHandler) addServiceDomainAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.ServiceDomain, logType string) {
	serviceDomainName := doc.ServiceDomainCore.Name
	serviceDomainID := doc.ID
	if doc.Type != nil && *doc.Type == string(model.KubernetesClusterTargetType) {
		GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, KUBERNETES_CLUSTER, logType, "", "", serviceDomainName, serviceDomainID, []string{serviceDomainName}, []string{serviceDomainID}, "")
	} else {
		GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, SERVICE_DOMAIN, logType, "", "", serviceDomainName, serviceDomainID, []string{serviceDomainName}, []string{serviceDomainID}, "")
	}
}

func (handler *AuditlogHandler) addNodeAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.Node, logType string) {
	serviceDomainID := doc.SvcDomainID
	sd, err := dbAPI.GetServiceDomain(context, serviceDomainID)
	if err == nil {
		serviceDomainName := sd.Name
		GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, NODE, logType, "", "", doc.Name, doc.ID, []string{serviceDomainName}, []string{serviceDomainID}, "")
	}
}

func (handler *AuditlogHandler) addUserAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.User, logType string) {
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, USER, logType, "", "", doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addCategoryAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.Category, logType string) {
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, CATEGORY, logType, "", "", doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addContainerRegistryAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.ContainerRegistry, logType string) {
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, CONTAINER_REGISTRY, logType, "", "", doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addDataPipelineAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.DataStream, logType string) {
	resourceName := doc.Name
	resourceID := doc.ID
	projectID := doc.ProjectID
	projectName, err := dbAPI.GetProjectName(context, doc.ProjectID)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, context, DATA_PIPELINE, logType, projectName, projectID, resourceName, resourceID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addFunctionAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.Script, logType string) {
	resourceName := doc.Name
	resourceID := doc.ID
	projectID := doc.ProjectID
	projectName, err := dbAPI.GetProjectName(context, doc.ProjectID)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, context, FUNCTION, logType, projectName, projectID, resourceName, resourceID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addCloudProfileAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.CloudCreds, logType string) {
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, CLOUD_PROFILE, logType, "", "", doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addAPIKeyAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.UserPublicKey, logType string) {
	user, err := dbAPI.GetUser(context, doc.ID)
	if err == nil {
		GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, context, API_KEY, logType, "", "", user.Name, doc.ID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addRuntimeEnvironmentAuditLog(dbAPI *dbObjectModelAPI, context context.Context, doc model.ScriptRuntime, logType string) {
	resourceName := doc.Name
	resourceID := doc.ID
	projectID := doc.ProjectID
	projectName, err := dbAPI.GetProjectName(context, doc.ProjectID)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, context, RUNTIME_ENV, logType, projectName, projectID, resourceName, resourceID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addServiceInstanceAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *model.ServiceInstance, logType string) {
	if doc == nil {
		return
	}
	var err error
	var projectID string
	var projectName string
	if doc.Scope == model.ServiceClassProjectScope {
		projectID = doc.ScopeID
		projectName, err = dbAPI.GetProjectName(ctx, projectID)
		if err != nil {
			return
		}
	}
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, ctx, SERVICE_INSTANCE, logType, projectName, projectID, doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addServiceBindingAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *model.ServiceBinding, logType string) {
	if doc == nil {
		return
	}
	var err error
	var projectID string
	var projectName string
	if doc.BindResource != nil && doc.BindResource.Type == model.ServiceBindingProjectResource {
		projectID = doc.BindResource.ID
		projectName, err = dbAPI.GetProjectName(ctx, projectID)
		if err != nil {
			return
		}
	}
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, ctx, SERVICE_BINDING, logType, projectName, projectID, doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addDataDriverClassAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *model.DataDriverClass, logType string) {
	GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, ctx, DATA_DRIVER_CLASS, logType, "", "", doc.Name, doc.ID, []string{}, []string{}, "")
}

func (handler *AuditlogHandler) addDataDriverInstanceAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *model.DataDriverInstance, logType string) {
	projectName, err := dbAPI.GetProjectName(ctx, doc.ProjectID)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, ctx, DATA_DRIVER_INSTANCE, logType, projectName, doc.ProjectID, doc.Name, doc.ID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addDataDriverConfigAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *DataDriverParamsDBO, projectId string, logType string) {
	projectName, err := dbAPI.GetProjectName(ctx, projectId)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, ctx, DATA_DRIVER_CONFIG, logType, projectName, projectId, doc.Name, doc.ID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addDataDriverStreamAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *DataDriverParamsDBO, projectId string, logType string) {
	projectName, err := dbAPI.GetProjectName(ctx, projectId)
	if err == nil {
		GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, ctx, DATA_DRIVER_STREAM, logType, projectName, projectId, doc.Name, doc.ID, []string{}, []string{}, "")
	}
}

func (handler *AuditlogHandler) addLogCollectorAuditLog(ctx context.Context, dbAPI *dbObjectModelAPI, doc *model.LogCollector, logType string) {
	if doc.Type == model.ProjectCollector {
		projectID := *doc.ProjectID
		projectName, err := dbAPI.GetProjectName(ctx, projectID)
		if err == nil {
			GetAuditlogHandler().InsertProjectScopeAuditLog(dbAPI, ctx, LOG_COLLECTOR, logType, projectName, projectID, doc.Name, doc.ID, []string{}, []string{}, "")
		}

	} else {
		GetAuditlogHandler().InsertInfraScopeAuditLog(dbAPI, ctx, LOG_COLLECTOR, logType, "", "", doc.Name, doc.ID, []string{}, []string{}, "")
	}
}

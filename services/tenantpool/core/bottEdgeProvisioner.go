package core

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	cloudmgmtmodel "cloudservices/common/model"
	"cloudservices/tenantpool/config"
	"cloudservices/tenantpool/generated/swagger/client"
	"cloudservices/tenantpool/generated/swagger/client/iot"
	"cloudservices/tenantpool/generated/swagger/client/setup"
	"cloudservices/tenantpool/generated/swagger/models"
	"cloudservices/tenantpool/model"
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/go-openapi/runtime"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/golang/glog"
)

const (
	DefaultToken    = "aaaa"
	DefaultTenant   = "system"
	DefaultProject  = "starter"
	DefaultUsername = "cloudmgmt"
	DefaultEmail    = "cloudmgmt@xi-iot.com"
	DefaultPassword = "********"
)

type BottEdgeProvisioner struct {
	auth          runtime.ClientAuthInfoWriter
	bottService   *client.BottService
	mutex         *sync.Mutex
	isInitialized bool
}

func NewBottEdgeProvisioner() (*BottEdgeProvisioner, error) {
	result, err := url.Parse(*config.Cfg.BottURL)
	if err != nil {
		return nil, err
	}
	// create the transport
	transport := httptransport.New(result.Host, result.Path, []string{result.Scheme})
	if glog.V(4) {
		transport.SetDebug(true)
	}
	edgeProvisioner := &BottEdgeProvisioner{mutex: &sync.Mutex{}}
	edgeProvisioner.auth = httptransport.APIKeyAuth("bott-token", "header", DefaultToken)
	edgeProvisioner.bottService = client.New(transport, strfmt.Default)
	return edgeProvisioner, nil
}

func (edgeProvisioner *BottEdgeProvisioner) Setup() error {
	edgeProvisioner.mutex.Lock()
	defer edgeProvisioner.mutex.Unlock()
	if edgeProvisioner.isInitialized {
		return nil
	}
	params := setup.NewGetUserTenantUsernameParams()
	params.Tenant = DefaultTenant
	params.Username = DefaultUsername
	_, err := edgeProvisioner.bottService.Setup.GetUserTenantUsername(params, edgeProvisioner.auth)
	if err == nil {
		edgeProvisioner.isInitialized = true
	} else {
		glog.Errorf("Error in getting user. Error: %s", err.Error())
		if httpErr, ok := err.(*setup.GetUserTenantUsernameDefault); ok {
			if httpErr.Code() != 400 || !strings.Contains(*httpErr.Payload.Message, "Error 404") {
				return err
			}
		} else {
			return err
		}
		params := setup.NewPostUserParams()
		params.Spec = &models.User{
			Email:    base.StringPtr(DefaultEmail),
			Name:     base.StringPtr(DefaultUsername),
			Password: base.StringPtr(DefaultPassword),
			Tenant:   DefaultTenant,
			Username: base.StringPtr(DefaultUsername),
		}
		_, err := edgeProvisioner.bottService.Setup.PostUser(params)
		if err != nil {
			glog.Errorf("Error in creating user. Error: %s", err.Error())
			if httpErr, ok := err.(*setup.PostUserDefault); ok {
				if httpErr.Code() != 400 || !strings.Contains(*httpErr.Payload.Message, "duplicate key value") {
					return err
				}
			} else {
				return err
			}
		}
		edgeProvisioner.isInitialized = true
	}
	return nil
}

func (edgeProvisioner *BottEdgeProvisioner) CreateEdge(ctx context.Context, config *model.CreateEdgeConfig) (*model.EdgeInfo, error) {
	contextID, err := edgeProvisioner.getOrCreateContext(ctx, config.TenantID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create edge. Error: %s"), err.Error())
		return nil, err
	}
	_, err = edgeProvisioner.getOrCreateCloudmgmtEndpoint(ctx, contextID, config.TenantID, config.SystemUser, config.SystemPassword)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create edge. Error: %s"), err.Error())
		return nil, err
	}
	createParams := iot.NewCreateAppsParams()
	createParams.Spec = &models.DeployAppsSpec{
		AppType: base.StringPtr("edge"),
		Apps: []*models.DeployAppInfo{
			&models.DeployAppInfo{
				ChartValues: "{}",
				// TODO make it configurable later
				ChartVersion: config.AppChartVersion,
				K8sInfo: &models.K8sDeployment{
					K8sNamespace:  base.StringPtr(""),
					K8sConfigJSON: base.StringPtr(""),
				},
				InstanceType: config.InstanceType,
				Tags:         getTags(config),
				EdgeMeta: &models.EdgeMeta{
					InfraType:          string(cloudmgmtmodel.CloudTargetType),
					SappDeploy:         config.DeployApp,
					DatapipelineDeploy: config.DatapipelineDeploy,
					DatasourceDeploy:   config.DatasourceDeploy,
				},
			},
		},
		ContextID: base.StringPtr(contextID),
		Name:      base.StringPtr(config.Name),
	}
	createOk, err := edgeProvisioner.bottService.Iot.CreateApps(createParams, edgeProvisioner.auth)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to create edge with config %+v. Error: %s"), *createParams, err.Error())
		return nil, err
	}
	glog.Infof(base.PrefixRequestID(ctx, "Created edge with params %+v"), createParams)
	edgeInfo := &model.EdgeInfo{ContextID: *createOk.Payload[0].ID, State: Creating}
	return edgeInfo, nil
}

func (edgeProvisioner *BottEdgeProvisioner) GetEdgeStatus(ctx context.Context, tenantID, appID string) (*model.EdgeInfo, error) {
	params := iot.NewGetAppParams()
	params.Name = getAppNameFromID(appID)
	params.ContextName = tenantID
	params.Project = DefaultProject
	params.Tenant = DefaultTenant
	ok, err := edgeProvisioner.bottService.Iot.GetApp(params, edgeProvisioner.auth)
	var state string
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get edge status %+v. Error: %s"), params, err.Error())
		if httpErr, ok := err.(*iot.GetAppDefault); ok {
			if httpErr.Code() == 404 {
				state = Deleted
			}
		}
		if len(state) == 0 {
			return nil, err
		}
	}
	if ok == nil || ok.Payload == nil || ok.Payload.Status == nil {
		state = Deleted
	} else {
		state = convertAppCreateStatus(*ok.Payload.Status)
	}
	edgeInfo := &model.EdgeInfo{ContextID: appID, State: state}
	if state == Created {
		edgeMeta := ok.Payload.EdgeMeta
		if edgeMeta != nil {
			if len(edgeMeta.SappProjectID) > 0 && len(edgeMeta.SappProjectName) > 0 {
				if edgeInfo.Resources == nil {
					edgeInfo.Resources = map[string]*model.Resource{}
				}
				edgeInfo.Resources[edgeMeta.SappProjectID] = &model.Resource{
					Type: model.ProjectResourceType,
					Name: edgeMeta.SappProjectName,
					ID:   edgeMeta.SappProjectID,
				}
			}
			edgeInfo.Edge = &cloudmgmtmodel.Edge{BaseModel: cloudmgmtmodel.BaseModel{ID: edgeMeta.EdgeID}}
		}
	}
	return edgeInfo, nil
}

func (edgeProvisioner *BottEdgeProvisioner) DeleteEdge(ctx context.Context, tenantID, appID string) (*model.EdgeInfo, error) {
	params := iot.NewDeleteAppParams()
	params.Name = getAppNameFromID(appID)
	params.Project = DefaultProject
	params.Tenant = DefaultTenant
	params.ContextName = tenantID
	_, err := edgeProvisioner.bottService.Iot.DeleteApp(params, edgeProvisioner.auth)
	if err != nil {
		if httpErr, ok := err.(*iot.DeleteAppDefault); !ok || httpErr.Code() != 404 {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete edge %+v. Error: %s"), *params, err.Error())
			return nil, err
		}
		glog.Warningf(base.PrefixRequestID(ctx, "Edge %+v is already deleted"), *params)
		return &model.EdgeInfo{ContextID: appID, State: Deleted}, nil
	}
	return &model.EdgeInfo{ContextID: appID, State: Deleting}, nil
}

func (edgeProvisioner *BottEdgeProvisioner) PostDeleteEdges(ctx context.Context, tenantID string) error {
	cloudmgmtEndpoint := getCloudmgmtEndpointName(tenantID)
	err := edgeProvisioner.deleteCloudmgmtEndpoint(ctx, tenantID, cloudmgmtEndpoint)
	if err != nil {
		if httpErr, ok := err.(*iot.DeleteCloudMgmtEndpointDefault); !ok || httpErr.Code() != 404 {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete cloudmanagement endpoint %s. Error: %s"), cloudmgmtEndpoint, err.Error())
			return err
		}
		glog.Warningf(base.PrefixRequestID(ctx, "Cloudmanagement endpoint %s is already deleted"), cloudmgmtEndpoint)
	}
	err = edgeProvisioner.deleteContext(ctx, tenantID)
	if err != nil {
		if httpErr, ok := err.(*iot.DeleteContextDefault); !ok || httpErr.Code() != 404 {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to delete context %s. Error: %s"), tenantID, err.Error())
			return err
		}
		glog.Warningf(base.PrefixRequestID(ctx, "Context %s is already deleted"), tenantID)
	}
	return nil
}

func (edgeProvisioner *BottEdgeProvisioner) DescribeEdge(ctx context.Context, tenantID, appID string) (map[string]interface{}, error) {
	params := iot.NewGetAppParams()
	params.Name = getAppNameFromID(appID)
	params.Project = DefaultProject
	params.Tenant = DefaultTenant
	params.ContextName = tenantID
	response := map[string]interface{}{}
	getAppOk, err := edgeProvisioner.bottService.Iot.GetApp(params, edgeProvisioner.auth)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get app %+v. Error: %s"), params, err.Error())
		return response, err
	}
	if getAppOk.Payload == nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get app payload %+v. Error: %s"), params, err.Error())
		return response, errcode.NewInternalError("Invalid payload")
	}
	err = base.Convert(getAppOk.Payload, &response)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to convert data %+v. Error: %s"), getAppOk.Payload, err.Error())
		return response, err
	}
	return response, nil
}

func (edgeProvisioner *BottEdgeProvisioner) getOrCreateContext(ctx context.Context, tenantID string) (string, error) {
	createParams := iot.NewPostContextParams()
	createParams.Spec = &models.Context{
		Name:    base.StringPtr(tenantID),
		Project: base.StringPtr(DefaultProject),
	}
	createOk, err := edgeProvisioner.bottService.Iot.PostContext(createParams, edgeProvisioner.auth)
	if err == nil {
		glog.Infof(base.PrefixRequestID(ctx, "Created context %+v"), *createParams)
		return *createOk.Payload.ID, err
	}
	getParams := iot.NewGetContextTenantProjectNameParams()
	getParams.Name = tenantID
	getParams.Project = DefaultProject
	getParams.Tenant = DefaultTenant
	getOk, err := edgeProvisioner.bottService.Iot.GetContextTenantProjectName(getParams, edgeProvisioner.auth)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get context %+v. Error: %s"), *getParams, err.Error())
		return "", err
	}
	return getOk.Payload.ID, err
}

func (edgeProvisioner *BottEdgeProvisioner) getOrCreateCloudmgmtEndpoint(ctx context.Context, contextID, tenantID, email, password string) (string, error) {
	cloudmgmtEndpointName := getCloudmgmtEndpointName(tenantID)
	createParams := iot.NewCreateCloudMgmtEndpointParams()
	createParams.Spec = &models.CloudMgmtEndpoint{
		ContextID: base.StringPtr(contextID),
		Name:      base.StringPtr(cloudmgmtEndpointName),
		Password:  base.StringPtr(password),
		URL:       base.StringPtr(*config.Cfg.CloudmgmtEndpoint),
		UserID:    base.StringPtr(email),
	}
	createOk, err := edgeProvisioner.bottService.Iot.CreateCloudMgmtEndpoint(createParams, edgeProvisioner.auth)
	if err == nil {
		glog.Infof(base.PrefixRequestID(ctx, "Created context %+v"), *createParams)
		return *createOk.Payload.ID, err
	}
	getParams := iot.NewGetCloudMgmtEndpointParams()
	getParams.ContextName = tenantID
	getParams.Name = cloudmgmtEndpointName
	getParams.Project = DefaultProject
	getParams.Tenant = DefaultTenant
	getOk, err := edgeProvisioner.bottService.Iot.GetCloudMgmtEndpoint(getParams, edgeProvisioner.auth)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get cloudmgmt endpoint %+v. Error: %s"), *getParams, err.Error())
		return "", err
	}
	return getOk.Payload.ID, err
}

func (edgeProvisioner *BottEdgeProvisioner) deleteCloudmgmtEndpoint(ctx context.Context, contextName, name string) error {
	params := iot.NewDeleteCloudMgmtEndpointParams()
	params.Name = name
	params.ContextName = contextName
	params.Project = DefaultProject
	params.Tenant = DefaultTenant
	_, err := edgeProvisioner.bottService.Iot.DeleteCloudMgmtEndpoint(params, edgeProvisioner.auth)
	return err
}

func (edgeProvisioner *BottEdgeProvisioner) deleteContext(ctx context.Context, name string) error {
	params := iot.NewDeleteContextParams()
	params.Name = name
	params.Project = DefaultProject
	params.Tenant = DefaultTenant
	_, err := edgeProvisioner.bottService.Iot.DeleteContext(params, edgeProvisioner.auth)
	return err
}

func getCloudmgmtEndpointName(tenantID string) string {
	return fmt.Sprintf("cloudmgmt-%s", tenantID)
}

func getEdgeName(tenantID string) string {
	return fmt.Sprintf("edge-%s", tenantID)
}

func convertAppCreateStatus(status string) string {
	switch status {
	case "Done":
		return Created
	case "InfraFailed":
		return Failed
	case "InfraDone":
		return Creating
	case "Creating":
		return Creating
	}
	return Failed
}

func getAppNameFromID(appID string) string {
	tokens := strings.Split(appID, "/")
	if len(tokens) < 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

func getTags(config *model.CreateEdgeConfig) string {
	tags := config.Tags
	tags = append(tags, fmt.Sprintf("creator=%s", DefaultUsername))
	return strings.Join(tags, ",")
}

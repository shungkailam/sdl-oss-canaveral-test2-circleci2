// Support for passing edge-specific variables to data driver template

package api

import (
	"cloudservices/common/apptemplate"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/model"

	"context"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

// App embeds reference to model.DataDriverClass to augment
// Application with RenderForEdge method.
type DataDriverInstanceInventory struct {
	*model.DataDriverInstanceInventory
}

func NewDataDriverInstanceInventory(
	class model.DataDriverClass,
	instance model.DataDriverInstance,
	config []model.DataDriverConfig,
	streams []model.DataDriverStream,
) *DataDriverInstanceInventory {
	result := DataDriverInstanceInventory{&model.DataDriverInstanceInventory{
		BaseModel:         instance.BaseModel,
		Doc:               instance,
		Class:             class,
		YamlData:          class.YamlData,
		DataDriverConfigs: config,
		DataDriverStreams: streams,
	}}
	return &result
}

func (dd *DataDriverInstanceInventory) render(dbAPI ObjectModelAPI,
	tenantID, edgeID, yaml string,
	values *model.DataDriverParametersValues) (string, []string, error) {
	// Access DB as infra user
	authContext := &base.AuthContext{
		TenantID: tenantID,
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
	edge, err := dbAPI.GetServiceDomain(ctx, edgeID)
	if err != nil {
		glog.Errorf("Failed to get edge %s: %s", edgeID, err)
		return "", nil, err
	}
	projectID := dd.Doc.ProjectID
	proj, err := dbAPI.GetProject(ctx, projectID)
	if err != nil {
		glog.Errorf("Failed to get project %s: %s", projectID, err)
		return "", nil, err
	}
	params := apptemplate.AppParameters{
		EdgeName:    edge.Name,
		EdgeID:      edge.ID,
		ProjectID:   dd.Doc.ProjectID,
		ProjectName: proj.Name,
		Namespace:   fmt.Sprintf("project-%s", projectID),
		AppID:       dd.Doc.ID,
		AppName:     dd.Doc.Name,
		AppVersion:  fmt.Sprintf("%f", dd.Doc.Version),
		Categories:  make(map[string]string),
		Env:         edge.Env,
		Parameters:  nil,
	}
	for _, label := range edge.Labels {
		cat, err := dbAPI.GetCategory(ctx, label.ID)
		if err != nil {
			glog.Errorf("Failed to get category %s: %s", label.ID, err)
			return "", nil, err
		}
		params.Categories[cat.Name] = label.Value
	}
	// add parameters
	params.Parameters, err = dd.flatDataDriverParameters(values)
	if err != nil {
		return "", nil, err
	}

	out, referencedServices, err := apptemplate.RenderWithParams(&params, yaml)
	if err != nil {
		glog.Errorf("Failed to create template %s: %s", projectID, err)
		return "", nil, err
	}
	return out, referencedServices, nil
}

func (dd *DataDriverInstanceInventory) flatDataDriverParameters(values *model.DataDriverParametersValues) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range *values {
		str, ok := v.(string)
		if ok {
			result[k] = string(str)
		} else {
			json, err := base.ConvertToJSON(v)
			if err != nil {
				return nil, err
			}
			result[k] = string(json)
		}
	}
	return result, nil
}

// Render data driver in place for API
func (dd *DataDriverInstanceInventory) RenderForContext(authContext *base.AuthContext, dbAPI ObjectModelAPI) error {
	var err error = nil
	if auth.IsEdgeRole(authContext) {
		edgeID := auth.GetEdgeID(authContext)
		tenantID := authContext.TenantID
		glog.Infof("Render app in edge context %s", edgeID)
		rendered := *dd.DataDriverInstanceInventory
		params := rendered.Doc.StaticParameters
		dd.YamlData, _, err = dd.render(dbAPI, tenantID, edgeID, rendered.YamlData, &params)
	}
	return err
}

// Implement api.Renderer interface.
// Render data driver template in edge context for WS broadcast.
func (dd *DataDriverInstanceInventory) RenderForEdge(dbAPI ObjectModelAPI, tenantID, edgeID string) (out interface{}, err error) {
	glog.Infof("Render app for edge %s", edgeID)
	// copy and modify app
	rendered := *dd.DataDriverInstanceInventory
	params := rendered.Doc.StaticParameters
	rendered.YamlData, _, err = dd.render(dbAPI, tenantID, edgeID, rendered.YamlData, &params)
	if err != nil {
		return
	}
	return rendered, nil
}

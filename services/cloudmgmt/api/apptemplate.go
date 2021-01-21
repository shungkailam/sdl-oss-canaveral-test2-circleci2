// Support for passing edge-specific variables to application template

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
	funk "github.com/thoas/go-funk"
)

// App embeds reference to model.Application to augment
// Application with RenderForEdge method.
type App struct {
	*model.Application
}

func NewApp(app *model.Application) *App {
	return &App{app}
}

func (app *App) render(dbAPI ObjectModelAPI,
	tenantID, edgeID, yaml string,
	edgeParams apptemplate.EdgeParameters) (string, []string, error) {
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
	proj, err := dbAPI.GetProject(ctx, app.GetProjectID())
	if err != nil {
		glog.Errorf("Failed to get project %s: %s", app.GetProjectID(),
			err)
		return "", nil, err
	}
	params := apptemplate.AppParameters{
		EdgeParameters: edgeParams,
		EdgeName:       edge.Name,
		EdgeID:         edge.ID,
		ProjectID:      app.ProjectID,
		ProjectName:    proj.Name,
		Namespace:      fmt.Sprintf("project-%s", app.ProjectID),
		AppID:          app.ID,
		AppName:        app.Name,
		AppVersion:     fmt.Sprintf("%f", app.Version),
		Categories:     make(map[string]string),
		Env:            edge.Env,
	}
	for _, label := range edge.Labels {
		cat, err := dbAPI.GetCategory(ctx, label.ID)
		if err != nil {
			glog.Errorf("Failed to get category %s: %s", label.ID, err)
			return "", nil, err
		}
		params.Categories[cat.Name] = label.Value
	}
	out, referencedServices, err := apptemplate.RenderWithParams(&params, yaml)
	if err != nil {
		glog.Errorf("Failed to create template %s: %s", app.GetProjectID(),
			err)
		return "", nil, err
	}
	return out, referencedServices, nil
}

// Render application in place for API
func (app *App) RenderForContext(authContext *base.AuthContext,
	dbAPI ObjectModelAPI) error {
	if auth.IsEdgeRole(authContext) {
		edgeID := auth.GetEdgeID(authContext)
		glog.Infof("Render app in edge context %s", edgeID)
		out, _, err := app.render(dbAPI, authContext.TenantID,
			edgeID, app.YamlData, apptemplate.EdgeParameters{})
		if err != nil {
			return err
		}
		app.YamlData = out
	}
	return nil
}

// Implement api.Renderer interface.
// Render application template in edge context for WS broadcast.
func (app *App) RenderForEdge(dbAPI ObjectModelAPI,
	tenantID, edgeID string) (out interface{}, err error) {
	glog.Infof("Render app for edge %s", edgeID)
	// copy and modify app
	renderedApp := *app.Application
	renderedApp.YamlData, _, err = app.render(dbAPI,
		tenantID, edgeID, app.YamlData, apptemplate.EdgeParameters{})
	if err != nil {
		return
	}
	return renderedApp, nil
}

// Render application for a particular edge with parameters originating
// at edge.
func (app *App) RenderForEdgeWithParams(dbAPI ObjectModelAPI,
	tenantID, edgeID string, params apptemplate.EdgeParameters) (
	yaml string,
	referencedServices []string,
	err error,
) {
	glog.Infof("Render app for edge %s", edgeID)
	// copy and modify app
	renderedApp := *app.Application
	renderedApp.YamlData, referencedServices, err = app.render(dbAPI,
		tenantID, edgeID, app.YamlData, params)
	if err != nil {
		return
	}
	return renderedApp.YamlData, referencedServices, nil
}

type Apps []*App

func NewApps(apps []model.Application) (res Apps) {
	for i := range apps {
		// ENG-185310 Keep reference to underlying application in order to
		// modify it in place.
		res = append(res, NewApp(&apps[i]))
	}
	return
}

// RenderForContext render all application template in edge context
// Modify all apps in place
func (apps Apps) RenderForContext(inAuthContext *base.AuthContext,
	dbAPI ObjectModelAPI) {
	if auth.IsEdgeRole(inAuthContext) {
		tenantID := inAuthContext.TenantID
		edgeID := auth.GetEdgeID(inAuthContext)
		// Access DB as infra user
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		// re-use edge object for all apps
		edge, err := dbAPI.GetServiceDomain(ctx, edgeID)
		if err != nil {
			glog.Errorf("Failed to get edge %s: %s", edgeID, err)
			return
		}
		// Batch fetching -- performance optimization
		// get all category names in one call
		catIDs := funk.Map(edge.Labels, func(x model.CategoryInfo) string { return x.ID }).([]string)
		catMap, err := dbAPI.GetCategoryNamesByIDs(ctx, catIDs)
		cats := make(map[string]string)
		if err != nil {
			glog.Errorf("Failed to get category names %v: %s", catIDs, err)
			return
		}
		for _, label := range edge.Labels {
			cats[catMap[label.ID]] = label.Value
		}
		// get all project names in one call
		projIDs := funk.Map(apps, func(x *App) string { return x.GetProjectID() }).([]string)
		projMap, err := dbAPI.GetProjectNamesByIDs(ctx, projIDs)
		if err != nil {
			glog.Errorf("Failed to get project names %v: %s", projIDs, err)
			return
		}
		for _, app := range apps {
			params := apptemplate.AppParameters{
				EdgeParameters: apptemplate.EdgeParameters{},
				EdgeName:       edge.Name,
				EdgeID:         edge.ID,
				ProjectID:      app.ProjectID,
				ProjectName:    projMap[app.ProjectID],
				AppID:          app.ID,
				AppName:        app.Name,
				AppVersion:     fmt.Sprintf("%f", app.Version),
				Categories:     cats,
				Env:            edge.Env,
			}
			out, _, err := apptemplate.RenderWithParams(&params, app.YamlData)
			if err != nil {
				glog.Errorf("Failed to create template %s: %s", app.GetProjectID(), err)
				return
			}
			app.YamlData = out
		}
	}

}

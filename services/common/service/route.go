package service

import (
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

var (
	defaultRoles = []string{
		auth.AdminRole,
		auth.EdgeRole,
		auth.OperatorRole,
		auth.OperatorTenantRole,
	}

	defaultMethods = []string{
		"POST",
		"PUT",
		"GET",
		"DELETE",
	}

	defaultTenants = []string{}
)

type tenantRoles struct {
	roles     []string
	tenantIDs []string
}

// RouteRoleValidator is the HTTP route role validator
type RouteRoleValidator struct {
	mutex      *sync.Mutex
	router     *httprouter.Router
	routeRoles map[string]map[string]*tenantRoles
}

// NewRouteRoleValidator returns a new instance of the HTTP route role validator
func NewRouteRoleValidator(router *httprouter.Router) *RouteRoleValidator {
	return &RouteRoleValidator{
		mutex:      &sync.Mutex{},
		router:     router,
		routeRoles: map[string]map[string]*tenantRoles{},
	}
}

// NewRouteRoleValidatorWithDefaults returns a new instance of the HTTP route role validator with defaults loaded
func NewRouteRoleValidatorWithDefaults(router *httprouter.Router) *RouteRoleValidator {
	validator := NewRouteRoleValidator(router)
	for _, method := range defaultMethods {
		validator.SetRouteRoles(method, "default", defaultTenants, defaultRoles)
	}
	return validator
}

// SetRouteRoles sets the route roles
func (validator *RouteRoleValidator) SetRouteRoles(method, path string, tenantIDs, roles []string) error {
	method = strings.ToUpper(method)
	validator.mutex.Lock()
	defer validator.mutex.Unlock()
	pathRoles, ok := validator.routeRoles[method]
	if !ok {
		pathRoles = map[string]*tenantRoles{}
		validator.routeRoles[method] = pathRoles
	}
	tRoles := &tenantRoles{
		roles:     []string{},
		tenantIDs: []string{},
	}
	if tenantIDs != nil {
		tRoles.tenantIDs = append(tRoles.tenantIDs, tenantIDs...)
	}
	if roles != nil {
		tRoles.roles = append(tRoles.roles, roles...)
	}
	pathRoles[path] = tRoles
	return nil
}

// DeleteRouteRoles deletes the route roles
func (validator *RouteRoleValidator) DeleteRouteRoles(method, path string) error {
	method = strings.ToUpper(method)
	validator.mutex.Lock()
	defer validator.mutex.Unlock()
	if pathRoles, ok := validator.routeRoles[method]; ok {
		delete(pathRoles, path)
		if len(pathRoles) == 0 {
			delete(validator.routeRoles, method)
		}
	}
	return nil
}

// Validate validates the route against the given role
func (validator *RouteRoleValidator) Validate(r *http.Request, inTenantID, inRole string) error {
	routePath := path.Clean(r.URL.Path)
	glog.V(4).Infof(base.PrefixRequestID(r.Context(), "Looking up route %s %s"), r.Method, routePath)
	handle, _, slashFound := validator.router.Lookup(r.Method, routePath)
	if handle == nil && !slashFound {
		glog.Errorf(base.PrefixRequestID(r.Context(), "Failed to find path %s %s"), r.Method, routePath)
		return errcode.NewRecordNotFoundError("path")
	}
	pathRole, ok := validator.routeRoles[r.Method]
	if !ok {
		glog.Errorf(base.PrefixRequestID(r.Context(), "Failed to find path %s %s"), r.Method, routePath)
		return errcode.NewRecordNotFoundError("method")
	}
	inRole = strings.ToLower(inRole)
	// Override for non-defaults
	if tRoles, ok := pathRole[routePath]; ok && len(tRoles.roles) > 0 {
		return validator.checkTenantRoles(tRoles, inTenantID, inRole)
	}
	if tRoles, ok := pathRole["default"]; ok && len(tRoles.roles) > 0 {
		return validator.checkTenantRoles(tRoles, inTenantID, inRole)
	}
	return nil
}

func (validator *RouteRoleValidator) checkTenantRoles(tRoles *tenantRoles, inTenantID, inRole string) error {
	for _, role := range tRoles.roles {
		if strings.ToLower(role) == inRole {
			if len(tRoles.tenantIDs) == 0 {
				// No specific tenant specified
				return nil
			}
			for _, tenantID := range tRoles.tenantIDs {
				if tenantID == inTenantID {
					return nil
				}
			}
			break
		}
	}
	return errcode.NewPermissionDeniedError("RBAC error")
}

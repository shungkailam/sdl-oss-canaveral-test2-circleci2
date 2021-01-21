package auth

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/meta"
	"cloudservices/common/model"
	"fmt"
	"net/http"
	"regexp"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	funk "github.com/thoas/go-funk"
)

const (
	OperatorTenantID = "tid-sherlock-operator"

	AdminRole          = "admin"
	EdgeRole           = "edge"
	OperatorRole       = "operator"
	OperatorTenantRole = "operator_tenant"

	SpecialRoleKey = "specialRole"
	ProjectsKey    = "projects"
	EdgeIDKey      = "edgeId"
)

var reBearer = regexp.MustCompile(`^Bearer\s+(.*)`)

type ContextHandler func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext)

type RbacContext struct {
	ProjectID    string
	OldProjectID string
	ID           string
	PrivilegedOp bool
	ProjNameFn   func(string) string
}

func GetSpecialRole(authContext *base.AuthContext) string {
	if i, ok := authContext.Claims[SpecialRoleKey]; ok {
		if specialRole, ok := i.(string); ok {
			return specialRole
		}
	}
	return ""
}

func IsInfraAdminRole(authContext *base.AuthContext) bool {
	result := false
	if authContext != nil && authContext.Claims != nil {
		result = authContext.Claims[SpecialRoleKey] == AdminRole
	}
	return result
}

func IsEdgeRole(authContext *base.AuthContext) bool {
	result := false
	if authContext != nil && authContext.Claims != nil {
		result = authContext.Claims[SpecialRoleKey] == EdgeRole
	}
	return result
}

func IsOperatorRole(authContext *base.AuthContext) bool {
	result := false
	if authContext != nil && authContext.Claims != nil {
		result = authContext.Claims[SpecialRoleKey] == OperatorRole
	}
	return result
}

func GetEdgeID(authContext *base.AuthContext) string {
	val := authContext.Claims[EdgeIDKey]
	if edgeID, ok := val.(string); ok {
		return edgeID
	}
	return ""
}

func IsInfraAdminOrEdgeRole(authContext *base.AuthContext) bool {
	result := false
	if authContext != nil && authContext.Claims != nil {
		sr := authContext.Claims[SpecialRoleKey]
		result = sr == AdminRole || sr == EdgeRole
	}
	return result
}

func IsProjectMember(projectID string, authContext *base.AuthContext) bool {
	if projectID == "" {
		return IsInfraAdminRole(authContext)
	}
	projectRoles := GetProjectRoles(authContext)
	if projectRoles == nil || len(projectRoles) == 0 {
		return false
	}
	for _, pr := range projectRoles {
		if projectID == pr.ProjectID {
			return true
		}
	}
	return false
}

// note: this works for both user and edge authContext
func GetProjectRoles(authContext *base.AuthContext) []model.ProjectRole {
	if authContext != nil && authContext.Claims != nil {
		x := authContext.Claims[ProjectsKey]
		if x != nil {
			projectRoles := []model.ProjectRole{}
			err := base.Convert(&x, &projectRoles)
			if err != nil {
				glog.Errorf("Failed to convert to project roles, %s!\n", err.Error())
				return nil
			}
			return projectRoles
		}
	}
	return nil
}

func GetProjectIDs(authContext *base.AuthContext) []string {
	projectRoles := GetProjectRoles(authContext)
	projectIDs := []string{}
	for _, pr := range projectRoles {
		projectIDs = append(projectIDs, pr.ProjectID)
	}
	return projectIDs
}

func (ctx *RbacContext) projectChanged() bool {
	return ctx.ProjectID != ctx.OldProjectID
}

func isPerProjectEntity(entity meta.Entity) bool {
	return entity == meta.EntityApplication ||
		entity == meta.EntityDataStream ||
		entity == meta.EntityScript ||
		entity == meta.EntityScriptRuntime ||
		entity == meta.EntityMLModel ||
		entity == meta.EntityLogCollector ||
		entity == meta.EntityHTTPServiceProxy ||
		entity == meta.EntityDataDriverStream ||
		entity == meta.EntityDataDriverConfig
}

func isGlobalEntitySupported(entity meta.Entity) bool {
	return entity == meta.EntityScript ||
		entity == meta.EntityScriptRuntime
}

// CheckRBAC - check RBAC permission
func CheckRBAC(authContext *base.AuthContext, entity meta.Entity, operation meta.Operation, ctx RbacContext) error {
	if isPerProjectEntity(entity) {
		switch operation {
		case meta.OperationUpdate:
			if ctx.OldProjectID != "" && ctx.projectChanged() {
				glog.Warningf("attempting to change project id from %s to %s\n", ctx.OldProjectID, ctx.ProjectID)
				return errcode.NewPermissionDeniedError("RBAC/Project/Change")
			}
			fallthrough
		case meta.OperationCreate:
			if ctx.ProjectID == "" && !isGlobalEntitySupported(entity) {
				return errcode.NewPermissionDeniedError("RBAC/Global/NotAllowed")
			}
			fallthrough
		case meta.OperationDelete:
			if !IsProjectMember(ctx.ProjectID, authContext) {
				// permission denied
				s := "RBAC/Project"
				if ctx.ProjNameFn != nil {
					s = fmt.Sprintf("RBAC/Project[%s]", ctx.ProjNameFn(ctx.ProjectID))
				}
				return errcode.NewPermissionDeniedError(s)
			}
		}
	} else if !IsInfraAdminRole(authContext) {
		// allow user to update self except change Role to INFRA_ADMIN
		if operation == meta.OperationUpdate && entity == meta.EntityUser && !IsEdgeRole(authContext) && !ctx.PrivilegedOp {
			id, ok := authContext.Claims["id"].(string)
			if ok && id == ctx.ID {
				glog.Infof("allow user with id %s to update self", id)
				return nil
			}
		}

		if IsEdgeRole(authContext) {
			switch operation {
			// Create operation
			case meta.OperationCreate:
				if entity == meta.EntityServiceInstance {
					glog.Infof("allow edge to create service instance %s", ctx.ID)
					return nil
				}
				if entity == meta.EntityServiceBinding {
					glog.Infof("allow edge to create service binding %s", ctx.ID)
					return nil
				}
			// Update operation
			case meta.OperationUpdate:
				// allow edge to update self
				if entity == meta.EntityEdge {
					id, ok := authContext.Claims["edgeId"].(string)
					if ok && id == ctx.ID {
						glog.Infof("allow edge with id %s to update self", id)
						return nil
					}
				}
				if entity == meta.EntityServiceInstance {
					glog.Infof("allow edge to update service instance %s", ctx.ID)
					return nil
				}
				if entity == meta.EntityServiceBinding {
					glog.Infof("allow edge to update service binding %s", ctx.ID)
					return nil
				}
				if entity == meta.EntityKubernetesCluster {
					id, ok := authContext.Claims["edgeId"].(string)
					if ok && id == ctx.ID {
						glog.Infof("allow edge to update kubernetes cluster %s", ctx.ID)
						return nil
					}
				}
			// Delete operation
			case meta.OperationDelete:
				if entity == meta.EntityServiceInstance {
					glog.Infof("allow edge to delete service instance %s", ctx.ID)
					return nil
				}
				if entity == meta.EntityServiceBinding {
					glog.Infof("allow edge to delete service binding %s", ctx.ID)
					return nil
				}
				if entity == meta.EntityKubernetesCluster {
					id, ok := authContext.Claims["edgeId"].(string)
					if ok && id == ctx.ID {
						glog.Infof("allow edge to delete kubernetes cluster %s", ctx.ID)
						return nil
					}
				}
			}
		}
		return errcode.NewPermissionDeniedError("RBAC/AdminRequired")
	}
	return nil
}

func makeFilterFn(authContext *base.AuthContext) interface{} {
	return func(app model.ProjectScopedEntity) bool {
		projectID := app.GetProjectID()
		if projectID == "" {
			return true
		}
		projectRoles := GetProjectRoles(authContext)
		for _, pr := range projectRoles {
			if projectID == pr.ProjectID {
				return true
			}
		}
		return false
	}
}

func FilterProjectScopedEntities(entities interface{}, authContext *base.AuthContext) interface{} {
	filterFn := makeFilterFn(authContext)
	return funk.Filter(entities, filterFn)
}

func FilterEntitiesByID(entities interface{}, idMap map[string]bool) interface{} {
	return funk.Filter(entities, func(entity model.IdentifiableEntity) bool {
		return idMap[entity.GetID()]
	})
}

func FilterEntitiesByClusterID(entities interface{}, idMap map[string]bool) interface{} {
	return funk.Filter(entities, func(entity model.ClusterEntity) bool {
		return idMap[entity.GetClusterID()]
	})
}

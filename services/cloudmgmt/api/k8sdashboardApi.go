package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
)

const entityTypeViewonlyUser = "viewonlyuser"

func init() {
	queryMap["SelectViewonlyUsersForSDTemplate"] = `SELECT * from user_model WHERE id IN (SELECT user_id FROM service_domain_viewonly_user_model WHERE edge_cluster_id = '%s')`
	queryMap["AddViewonlyUsersToSD"] = `INSERT INTO service_domain_viewonly_user_model (edge_cluster_id, user_id, created_at) VALUES `
	queryMap["RemoveViewonlyUsersFromSDTemplate"] = `DELETE FROM service_domain_viewonly_user_model WHERE edge_cluster_id = '%s' AND user_id IN (%s)`
}

// GetViewonlyUsersForSD get all viewonly users for the Service Domain / Karbon Cluster
func (dbAPI *dbObjectModelAPI) GetViewonlyUsersForSD(ctx context.Context, svcDomainID string) ([]model.User, error) {
	if err := validateObjectID(svcDomainID, "svcDomainId"); err != nil {
		return nil, err
	}
	users := []model.User{}
	query := fmt.Sprintf(queryMap["SelectViewonlyUsersForSDTemplate"], svcDomainID)
	if err := dbAPI.Query(ctx, &users, query, struct{}{}); err != nil {
		return nil, err
	}
	return users, nil
}

// AddViewonlyUsersToSD add viewonly users to Service Domain / Karbon Cluster
// infra admins automatically have full access, so no need to add them
// For now, this method requires infra admin role since Karbon Cluster
// is not part of project scope for now
func (dbAPI *dbObjectModelAPI) AddViewonlyUsersToSD(ctx context.Context, svcDomainID string, userIDs []string) error {
	if err := validateObjectID(svcDomainID, "svcDomainId"); err != nil {
		return err
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return errcode.NewPermissionDeniedError("RBAC")
	}
	us := make([]string, 0, len(userIDs))
	for _, u := range userIDs {
		if err := validateObjectID(u, "userId"); err != nil {
			return err
		}
		us = append(us, fmt.Sprintf("('%s', '%s', NOW())", svcDomainID, u))
	}
	s := strings.Join(us, ", ")
	query := fmt.Sprintf("%s %s", queryMap["AddViewonlyUsersToSD"], s)
	_, err = dbAPI.Exec(ctx, query)
	return err
}

// RemoveViewonlyUsersFromSD remove viewonly users from Service Domain / Karbon Cluster
// For now, this method requires infra admin role since Karbon Cluster
// is not part of project scope for now
func (dbAPI *dbObjectModelAPI) RemoveViewonlyUsersFromSD(ctx context.Context, svcDomainID string, userIDs []string) error {
	if err := validateObjectID(svcDomainID, "svcDomainId"); err != nil {
		return err
	}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return err
	}
	if !auth.IsInfraAdminRole(authContext) {
		return errcode.NewPermissionDeniedError("RBAC")
	}
	for _, u := range userIDs {
		if err := validateObjectID(u, "userId"); err != nil {
			return err
		}
	}
	s := fmt.Sprintf("'%s'", strings.Join(userIDs, "', '"))
	query := fmt.Sprintf(queryMap["RemoveViewonlyUsersFromSDTemplate"], svcDomainID, s)
	_, err = dbAPI.Exec(ctx, query)
	return err
}

// can use makeEdgeGetAllHandle with this
func (dbAPI *dbObjectModelAPI) GetViewonlyUsersForSDW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	users, err := dbAPI.GetViewonlyUsersForSD(ctx, svcDomainID)
	if err != nil {
		// TODO - wrap with user friendly error
		return err
	}
	queryParam := model.GetEntitiesQueryParam(req)
	queryInfo := ListQueryInfo{TotalCount: len(users), EntityType: entityTypeViewonlyUser}
	entityListResponsePayload := makeEntityListResponsePayload(queryParam, &queryInfo)
	r := model.K8sDashboardViewonlyUserListPayload{
		EntityListResponsePayload: entityListResponsePayload,
		ViewonlyUserList:          users,
	}
	return json.NewEncoder(w).Encode(r)
}

// TODO: use makePostHandle3
func (dbAPI *dbObjectModelAPI) AddViewonlyUsersToSDW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	params := model.K8sDashboardViewonlyUserParams{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &params)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	err = dbAPI.AddViewonlyUsersToSD(ctx, svcDomainID, params.UserIDs)
	if err != nil {
		// TODO user friendly error
		return err
	}
	return json.NewEncoder(w).Encode(model.K8sDashboardViewonlyUserUpdatePayload{})
}

func (dbAPI *dbObjectModelAPI) RemoveViewonlyUsersFromSDW(ctx context.Context, svcDomainID string, w io.Writer, req *http.Request) error {
	params := model.K8sDashboardViewonlyUserParams{}
	var r io.Reader = req.Body
	err := base.Decode(&r, &params)
	if err != nil {
		return errcode.NewMalformedBadRequestError("body")
	}
	err = dbAPI.RemoveViewonlyUsersFromSD(ctx, svcDomainID, params.UserIDs)
	if err != nil {
		// TODO user friendly error
		return err
	}
	return json.NewEncoder(w).Encode(model.K8sDashboardViewonlyUserUpdatePayload{})
}

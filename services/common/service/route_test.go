package service_test

import (
	"bytes"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/auth"
	"cloudservices/common/service"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/julienschmidt/httprouter"
)

func TestRouteRole(t *testing.T) {
	dummyRdr := bytes.NewBuffer([]byte{})
	router := httprouter.New()

	routeRoles := []struct {
		method    string
		path      string
		tenantIDs []string
		roles     []string
	}{
		{
			"POST",
			"/v1.0/tenants",
			[]string{
				"tid-sherlock-operator",
			},
			[]string{
				auth.OperatorTenantRole,
			},
		},
		{
			"PUT",
			"/v1.0/tenants/:tenantId",
			nil,
			[]string{
				auth.OperatorTenantRole,
			},
		},
		{
			"DELETE",
			"/v1.0/tenants/:tenantId",
			nil,
			[]string{
				auth.OperatorTenantRole,
			},
		},
	}
	router.Handle("POST", "/v1.0/tenants", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	})
	router.Handle("PUT", "/v1.0/tenants/:tenantId", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	})
	router.Handle("GET", "/v1.0/applications", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	})
	router.Handle("DELETE", "/v1.0/tenants/:tenantId", func(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	})
	validator := service.NewRouteRoleValidatorWithDefaults(router)
	for _, rrs := range routeRoles {
		validator.SetRouteRoles(rrs.method, rrs.path, rrs.tenantIDs, rrs.roles)
	}

	testData := []struct {
		method      string
		path        string
		tenantID    string
		role        string
		expectError bool
	}{
		{
			"POST",
			"/v1.0/tenants/",
			"tid-sherlock-operator",
			auth.OperatorTenantRole,
			false,
		},
		{
			"POST",
			"/v1.0/tenants",
			"tid-sherlock-operator",
			auth.OperatorTenantRole,
			false,
		},
		{
			"POST",
			"/v1.0/tenants",
			"tid-regular-user",
			auth.OperatorTenantRole,
			true,
		},
		{
			"PUT",
			"/v1.0/tenants/123",
			"tid-regular-user",
			auth.OperatorTenantRole,
			false,
		},
		{
			"POST",
			"/v1.0/tenants/",
			"tid-regular-user",
			auth.AdminRole,
			true,
		},
		{
			"GET",
			"/v1.0/applications",
			"tid-regular-user",
			auth.AdminRole,
			false,
		},
		{
			"GET",
			"/v1.0/applications",
			"tid-regular-user",
			"no role",
			true,
		},
	}
	for _, td := range testData {
		t.Logf("Running test data %+v\n", td)
		req, err := apitesthelper.NewHTTPRequest(td.method, td.path, dummyRdr)
		require.NoError(t, err)
		err = validator.Validate(req, td.tenantID, td.role)
		if td.expectError {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}
		t.Logf("Successfully executed test data %+v\n", td)
	}
}

package router_test

import (
	"cloudservices/cloudmgmt/router"
	"context"
	"net/url"
	"testing"

	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/websocket"
	"cloudservices/common/base"

	"github.com/dgrijalva/jwt-go"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseServiceDomainAndURL(t *testing.T) {
	t.Parallel()
	t.Log("running TestParseServiceDomainAndURL test")
	t.Run("Test parse sd id and token", func(t *testing.T) {
		positiveTests := []struct {
			edgeID string
			rawURL string
			path   string
		}{
			{"ab5acdf1-a1a5-46cf-891b-cbdaed3bad56", "https://xinyuan.ntnxsherlock.com/v1.0/kiali/graph?namespaces=default&serviceDomain=ab5acdf1-a1a5-46cf-891b-cbdaed3bad56", "http://kiali.istio-system.svc:20001/kiali/api/namespaces/graph?namespaces=default&"},
		}
		// defer func() {
		// 	dbAPI, err := api.NewObjectModelAPI()
		// 	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
		// 	tenantID := tenant.ID
		// 	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
		// 	authContext := &base.AuthContext{
		// 		TenantID: tenantID,
		// 		Claims: jwt.MapClaims{
		// 			"specialRole": "admin",
		// 		},
		// 	}
		// 	router := httprouter.New()
		// 	msgSvc := websocket.ConfigureWSMessagingService(dbAPI, router, nil)
		// 	ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		// 	dbAPI.DeleteUser(ctx, user.ID, nil)
		// 	dbAPI.DeleteBuiltinTenantObjects(ctx, tenantID)
		// 	dbAPI.DeleteTenant(ctx, tenantID, nil)
		// 	dbAPI.Close()
		// }()
		dbAPI, err := api.NewObjectModelAPI()
		require.NoError(t, err)
		tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
		tenantID := tenant.ID
		user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		r := httprouter.New()
		msgSvc := websocket.ConfigureWSMessagingService(dbAPI, r, nil)
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		for _, pt := range positiveTests {
			u, err := url.Parse(pt.rawURL)
			require.NoError(t, err)
			sd, _, err := router.ParseServiceDomain(ctx, u, authContext, msgSvc)
			// require.NoError(t, err)
			assert.Equal(t, pt.edgeID, sd, "edgeID should be equal")

			path := router.ParsePath(u)
			assert.Equal(t, pt.path, path, "URL should be equal")
		}
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteBuiltinTenantObjects(ctx, tenantID)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	})
}

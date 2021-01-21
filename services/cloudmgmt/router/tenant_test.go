package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/model"
	"context"
	"crypto/ecdsa"
	"fmt"
	"net/http"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/require"
)

const (
	TENANT_PATH = "/v1.0/tenants"
)

var (
	claims = jwt.MapClaims{
		"id":          "10913f57-53a7-449e-be36-8c131833bedb",
		"tenantId":    "tid-sherlock-operator",
		"specialRole": "operator_tenant", // admin, none, operator, etc.
		"exp":         123,
	}
)

func createTenant(netClient *http.Client, tenant model.Tenant, token string) (model.CreateDocumentResponseV2, string, error) {
	return createEntityV2(netClient, TENANT_PATH, tenant, token)
}

func deleteTenant(netClient *http.Client, tenantID, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, TENANT_PATH, tenantID, token)
}

func getECDSAKeys(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	var (
		verifyKey *ecdsa.PublicKey
		signKey   *ecdsa.PrivateKey
	)

	return signKey, verifyKey
}

func createAPICallerToken(t *testing.T, operatorUserID, dataECPrivKey string) string {
	signBytes := []byte(dataECPrivKey)
	signKey, err := jwt.ParseECPrivateKeyFromPEM(signBytes)
	require.NoError(t, err)
	var exp = time.Now().Unix() + 30*60
	claims["id"] = operatorUserID
	claims["exp"] = exp
	token, err := crypto.ECDSASignJWT(signKey, claims)
	require.NoError(t, err)
	return token
}

func getUserIDFromToken(t *testing.T, token string) string {
	tokenClaims := apitesthelper.ExtractJWTClaims(t, token)
	t.Logf("Got claims %+v", tokenClaims)
	operatorUserID, ok := tokenClaims["id"].(string)
	if !ok {
		t.Fatal("Invalid user ID")
	}
	return operatorUserID
}

func machineTenantContext() context.Context {
	authContext := &base.AuthContext{
		TenantID: base.MachineTenantID,
	}
	return context.WithValue(context.Background(), base.AuthContextKey, authContext)
}

func createOperatorUser(t *testing.T, dbAPI api.ObjectModelAPI) model.User {
	id := base.GetUUID()
	operatorTenantUser := model.User{
		BaseModel: model.BaseModel{
			TenantID: base.OperatorTenantID,
		},
		Email:    fmt.Sprintf("test-%s-op@ntnxsherlock.com", id),
		Password: "P@ssw0rd",
		Role:     "OPERATOR_TENANT",
	}

	ctx := machineTenantContext()
	resp, err := dbAPI.CreateUser(ctx, &operatorTenantUser, nil)
	require.NoError(t, err)
	operatorTenantUser.ID = resp.(model.CreateDocumentResponse).ID
	return operatorTenantUser
}

func TestTenant(t *testing.T) {
	t.Parallel()
	t.Log("running TestTenant test")
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	defer dbAPI.Close()
	machineCtx := machineTenantContext()
	tenantID := auth.OperatorTenantID
	operatorTenantUser := createOperatorUser(t, dbAPI)
	defer dbAPI.DeleteUser(machineCtx, operatorTenantUser.ID, nil)
	var netClient = &http.Client{
		Timeout: time.Minute,
	}
	t.Logf("created operator tenant user : %+v\n", operatorTenantUser)
	token := loginUser(t, netClient, operatorTenantUser)
	operatorTenantUserID := getUserIDFromToken(t, token)
	doc := model.UserPublicKey{ID: operatorTenantUser.ID, TenantID: tenantID, PublicKey: dataECPubKey}
	_, _, err = updateUserPublicKey(netClient, doc, token)
	require.NoError(t, err)
	defer deleteUserPublicKey(netClient, token)
	t.Run("Test Tenant", func(t *testing.T) {
		callerToken := createAPICallerToken(t, operatorTenantUserID, dataECPrivKey)
		doc := model.Tenant{
			Name: "Test drive tenant",
		}
		resp, _, err := createTenant(netClient, doc, callerToken)
		require.NoError(t, err)
		defer deleteTenant(netClient, resp.ID, callerToken)
		tenant, err := dbAPI.GetTenant(context.TODO(), resp.ID)
		require.NoError(t, err)
		require.Equal(t, doc.Name, tenant.Name, "Tenant names are different")
		_, _, err = deleteTenant(netClient, resp.ID, callerToken)
		require.NoError(t, err)
		tenant, err = dbAPI.GetTenant(context.TODO(), resp.ID)
		fmt.Printf("%+v\n", tenant)
		require.Error(t, err)
	})
}

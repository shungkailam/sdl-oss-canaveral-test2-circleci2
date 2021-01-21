package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/common/model"
	"context"
	"github.com/stretchr/testify/require"

	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"crypto/rsa"

	"net/http"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

const (
	USER_PUBLIC_KEY_PATH = "/v1.0/userpublickey"

	// ssh-keygen -t rsa -b 4096
	dataRSAPrivKey = `-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----`

	// openssl rsa -in <dataRSAPrivKey> -pubout
	dataRSAPubKey = `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEA4JpapUIdIY5lKiESOD0I
NidkqMCF2NP35h2CteA+L4oQxwofPOdFltTbeEi73YG/VAFaHrdkP/ESuPcZbGjx
IKf84ds9IRousvjJwZJAzMQ335J/JEf6SaA+DSnwhizMXLkOaJrRGlz30NCAoNHd
GB2ePyUUAyNMpmOgKoPZ4ME6rjTaHDuTXspofkVQsxX3vxJ5y73tRAQl8IQhB7gn
osd0oTj0tTIaSPbnB8QwBYnbCbB97Oo9EH6SM9/HtpyGlJYHMi/i2AFwmmJCkhar
HIluAk5rZxirav89t5HwLKDm2ueJKmWmsQXuEpa55dQDMT7tkr0sjduBt6uUFpr9
iNetabZnt4sA1GX5ybEG5emXyU5wOjo9faCldR4K3VLQyMc9t/hb7pYKY4bMgh1w
JZKgCySGGbDhPu3y2I5eFeYJIrWvRm5SUPtzybbgxJUPYZKDmuCvMQvhA/w0VAYn
SEnMJc2q/MziOdIrcP2ABvmXwaok9fcsdWvsyh2SPq6zsWEYqSA5rl/kyso8qJF4
i5u63AkUKzF7SVAtVQLdWV2cJit73jj9RcMW+iKUgoyjmmopG1WHMOu93f4ssHZY
RfwS00BlRDfNjtyF0GGn8XNtRGhkqhhksuPCVDzBtukBd7FSrmZFoSSHrcNshrkH
6VuR/9jyYRLMmtmTOveIBl8CAwEAAQ==
-----END PUBLIC KEY-----`

	// ssh-keygen -t ecdsa -b 521
	dataECPrivKey = `-----BEGIN EC PRIVATE KEY-----
MIHbAgEBBEFXx86hbQGyftpIsthJerc4npdw9PKra9f/JCxSqSjWwcJWqgqrJNkC
2hWUvvkvYCuPALjBep9ydybN4x+rqspNo6AHBgUrgQQAI6GBiQOBhgAEAARTMd8O
TVXgd7R1F2gSifNfD9/W+Ue3S/SZmipaawpnjRFOiNXX8hsQ3+Wd3KqjKBZHsIl/
W5ZQR9ybKoDThpIHAa/cN0RL2tDi/YDRBhMf4lXUfCzGMpz+ZOXqLlRGgeE8sHK7
bfd1kJJc0cE/ehBjxqOk+JQ13DAhKwTB37+FddzL
-----END EC PRIVATE KEY-----`
	// openssl ec -in <dataECPrivKey> -pubout
	dataECPubKey = `-----BEGIN PUBLIC KEY-----
MIGbMBAGByqGSM49AgEGBSuBBAAjA4GGAAQABFMx3w5NVeB3tHUXaBKJ818P39b5
R7dL9JmaKlprCmeNEU6I1dfyGxDf5Z3cqqMoFkewiX9bllBH3JsqgNOGkgcBr9w3
REva0OL9gNEGEx/iVdR8LMYynP5k5eouVEaB4Tywcrtt93WQklzRwT96EGPGo6T4
lDXcMCErBMHfv4V13Ms=
-----END PUBLIC KEY-----`
)

// update user public key
func updateUserPublicKey(netClient *http.Client, userPublicKey model.UserPublicKey, token string) (model.UpdateDocumentResponseV2, string, error) {
	return updateEntityV2(netClient, USER_PUBLIC_KEY_PATH, userPublicKey, token)
}

// delete user public key
func deleteUserPublicKey(netClient *http.Client, token string) (model.DeleteDocumentResponseV2, string, error) {
	return deleteEntityV2(netClient, USER_PUBLIC_KEY_PATH, "", token)
}

// get user public key
func getUserPublicKey(netClient *http.Client, token string) (model.UserPublicKey, error) {
	userPublicKey := model.UserPublicKey{}
	err := doGet(netClient, USER_PUBLIC_KEY_PATH, token, &userPublicKey)
	return userPublicKey, err
}

func getRSAPrivateKey(t *testing.T) *rsa.PrivateKey {
	key, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(dataRSAPrivKey))
	require.NoError(t, err)
	return key
}

func createJWTClaims(id, tenantId, email, role string) jwt.MapClaims {
	var exp = time.Now().Unix() + 30*60
	return jwt.MapClaims{
		"id":          id,
		"tenantId":    tenantId,
		"email":       email,
		"specialRole": role, // admin, none, operator, etc.
		"exp":         exp,
	}

}

func TestUserPublicKey(t *testing.T) {
	t.Parallel()
	t.Log("running TestUserPublicKey test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test User Public Key", func(t *testing.T) {
		// login as user to get token

		token := loginUser(t, netClient, user)
		doc := model.UserPublicKey{ID: user.ID, TenantID: tenantID, PublicKey: dataRSAPubKey}

		resp, _, err := updateUserPublicKey(netClient, doc, token)
		require.NoError(t, err)
		t.Logf("create user public key successful: %+v", resp)

		// get user public key
		pk, err := getUserPublicKey(netClient, token)
		require.NoError(t, err)
		if pk.PublicKey != dataRSAPubKey {
			t.Fatal("expect public key to match")
		}
		t.Logf("Got user public key: %+v", pk)

		// now, use private key to sign a JWT token
		claims := createJWTClaims(user.ID, user.TenantID, user.Email, model.GetUserSpecialRole(&user))
		rsaSignKey := getRSAPrivateKey(t)
		rsaToken, err := crypto.RSASignJWT(rsaSignKey, claims)
		require.NoError(t, err)
		users, err := getUsers(netClient, rsaToken)
		require.NoError(t, err)
		t.Logf("Got users using RSA token: %+v", users)

		// update user public key
		pk.PublicKey = dataECPubKey
		resp, _, err = updateUserPublicKey(netClient, pk, token)
		require.NoError(t, err)
		t.Logf("update user public key successful: %+v", resp)

		// get user public key
		pk, err = getUserPublicKey(netClient, token)
		require.NoError(t, err)
		if pk.PublicKey != dataECPubKey {
			t.Fatal("expect public key to match")
		}
		t.Logf("Got user public key: %+v", pk)

		users, err = getUsers(netClient, rsaToken)
		require.Error(t, err, "Expect get users with rsa token to fail after key update")

		// get user public key after api call
		pk, err = getUserPublicKey(netClient, token)
		require.NoError(t, err)
		t.Logf("Got user public key after api call: %+v", pk)

		// now delete user public key
		dresp, _, err := deleteUserPublicKey(netClient, token)
		require.NoError(t, err)
		t.Logf("Delete user public key response: %+v", dresp)

		// get user public key should fail after delete
		_, err = getUserPublicKey(netClient, token)
		require.Error(t, err, "Expect get public key to fail after delete")

	})

}

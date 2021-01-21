package crypto_test

import (
	"cloudservices/common/crypto"
	"cloudservices/common/model"
	"crypto/ecdsa"
	"crypto/rsa"
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func createJWTClaims(exp int64) jwt.MapClaims {
	return jwt.MapClaims{
		"foo": "bar",
		"exp": exp,
	}
}

func TestJWTSignVerify(t *testing.T) {

	var exp = time.Now().Unix() + 30*60
	// must do this before crypto.SignJWT
	keyService := createKmsKeyService()

	claims := createJWTClaims(exp)
	token, err := crypto.SignJWT(claims)
	require.NoErrorf(t, err, "sign jwt failed %s", err)
	t.Logf("Got JWT token: %s", token)
	claims2, err := crypto.VerifyJWT(token)
	require.NoErrorf(t, err, "verify jwt failed %s", err)
	t.Logf("claims: %+v", claims)
	t.Logf("claims2: %+v", claims2)
	// note: use reflect.DeepEqual will not give equal here due to int64 vs float64 issue
	if !model.MarshalEqual(&claims, &claims2) {
		t.Fatalf("claims not marshal equal\n")
	}

	// attempt to verify with a wrong secret should fail
	jwtSecret2 := "AQIDAHjsaXMu080VfOxOyMG5ljU9Oia7wzXuaEHGY0eBi9M15QFQ9sNOSYY1D9KBaKT3GEL7AAAAfjB8BgkqhkiG9w0BBwagbzBtAgEAMGgGCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQMIQP78xdChW/9ED1UAgEQgDvjdZHtxTN0QQxttCbcKLOty/nrjeYYDP5TgHxh5QVfOI/zPNdNKgePaBTsw1UdZDzGnrNHFF6/oWsokQ=="

	ba, err := keyService.DecryptDataKey(jwtSecret2)
	require.NoError(t, err)
	savedJWTSecret := crypto.JWTSecret
	crypto.JWTSecret = ba
	claims2, err = crypto.VerifyJWT(token)
	require.Errorf(t, err, "unexpected success in VerifyJWT using wrong secret")
	crypto.JWTSecret = savedJWTSecret

	// expired token should give verification error
	exp = time.Now().Unix() - 60
	claims = createJWTClaims(exp)
	token, err = crypto.SignJWT(claims)
	require.NoErrorf(t, err, "sign exp jwt failed %s", err)
	t.Logf("Got exp JWT token: %s", token)
	claims2, err = crypto.VerifyJWT(token)
	require.Errorf(t, err, "expired token should give verification error")

}

func getRSAKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PublicKey) {
	var (
		verifyKey *rsa.PublicKey
		signKey   *rsa.PrivateKey
	)
	signBytes := []byte(dataRSAPrivKey)
	signKey, err := jwt.ParseRSAPrivateKeyFromPEM(signBytes)
	require.NoError(t, err)
	verifyBytes := []byte(dataRSAPubKey)
	verifyKey, err = jwt.ParseRSAPublicKeyFromPEM(verifyBytes)
	require.NoError(t, err)
	return signKey, verifyKey
}

func TestRSAJWTSignVerify(t *testing.T) {

	signKey, verifyKey := getRSAKeys(t)
	claims := createJWTClaims(time.Now().Unix() + 30*60)
	token, err := crypto.RSASignJWT(signKey, claims)
	require.NoErrorf(t, err, "sign jwt failed %s", err)
	t.Logf("Got JWT token: %s", token)
	claims2, err := crypto.RSAVerifyJWT(verifyKey, token)
	require.NoErrorf(t, err, "verify jwt failed %s", err)
	t.Logf("claims: %+v", claims)
	t.Logf("claims2: %+v", claims2)
	// note: use reflect.DeepEqual will not give equal here due to int64 vs float64 issue
	if !model.MarshalEqual(&claims, &claims2) {
		t.Fatalf("claims not marshal equal\n")
	}
}

func getECDSAKeys(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	var (
		verifyKey *ecdsa.PublicKey
		signKey   *ecdsa.PrivateKey
	)
	signBytes := []byte(dataECPrivKey)
	signKey, err := jwt.ParseECPrivateKeyFromPEM(signBytes)
	require.NoError(t, err)
	verifyBytes := []byte(dataECPubKey)
	verifyKey, err = jwt.ParseECPublicKeyFromPEM(verifyBytes)
	require.NoError(t, err)
	return signKey, verifyKey
}

func TestECDSAJWTSignVerify(t *testing.T) {

	signKey, verifyKey := getECDSAKeys(t)
	claims := createJWTClaims(time.Now().Unix() + 30*60)
	token, err := crypto.ECDSASignJWT(signKey, claims)
	require.NoErrorf(t, err, "sign jwt failed %s", err)
	t.Logf("Got JWT token: %s", token)
	claims2, err := crypto.ECDSAVerifyJWT(verifyKey, token)
	require.NoErrorf(t, err, "verify jwt failed %s", err)
	t.Logf("claims: %+v", claims)
	t.Logf("claims2: %+v", claims2)
	// note: use reflect.DeepEqual will not give equal here due to int64 vs float64 issue
	if !model.MarshalEqual(&claims, &claims2) {
		t.Fatalf("claims not marshal equal\n")
	}

}

func TestAllJWTSignVerify(t *testing.T) {
	// must do this before crypto.SignJWT
	createKmsKeyService()

	ecdsaSignKey, ecdsaVerifyKey := getECDSAKeys(t)
	rsaSignKey, rsaVerifyKey := getRSAKeys(t)
	claims := createJWTClaims(time.Now().Unix() + 30*60)
	secToken, err := crypto.SignJWT(claims)
	require.NoErrorf(t, err, "JWTSecret sign jwt failed %s", err)
	ecdsaToken, err := crypto.ECDSASignJWT(ecdsaSignKey, claims)
	require.NoErrorf(t, err, "ECDSA sign jwt failed %s", err)
	rsaToken, err := crypto.RSASignJWT(rsaSignKey, claims)
	require.NoErrorf(t, err, "ECDSA sign jwt failed %s", err)
	publicKeyResolver := func() func(token *jwt.Token) (interface{}, error) {
		return func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
				return rsaVerifyKey, nil
			}
			if _, ok := token.Method.(*jwt.SigningMethodECDSA); ok {
				return ecdsaVerifyKey, nil
			}
			return nil, fmt.Errorf("Failed to resolve public key")
		}
	}
	tokens := []string{secToken, rsaToken, ecdsaToken}
	for _, token := range tokens {
		mc, err := crypto.VerifyJWT2(token, publicKeyResolver)
		require.NoError(t, err)
		if !model.MarshalEqual(&mc, &claims) {
			t.Fatalf("claims not marshal equal\n")
		}
	}
}

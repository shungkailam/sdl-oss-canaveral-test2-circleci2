package crypto_test

import (
	"cloudservices/common/crypto"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"testing"
)

func createKmsKeyService() crypto.KeyService {
	jwtSecret := os.Getenv("JWT_SECRET")
	awsRegion := "us-west-2"
	kmsKey := "alias/ntnx/cloudmgmt-dev"
	keyService := crypto.NewKeyService(awsRegion, jwtSecret, kmsKey, true)
	// crypto.JWTSecret = keyService.GetJWTSecret()
	return keyService
}
func createCryptoKeyService(t *testing.T) crypto.KeyService {
	masterKey, err := crypto.GenerateKeyString()
	require.NoError(t, err)
	jwtSecret, err := crypto.GenerateDataKey(masterKey)
	require.NoError(t, err)
	t.Logf("masterKey: %s", masterKey)
	t.Logf("jwtSecret: %s", jwtSecret)
	return crypto.NewKeyService("", jwtSecret, masterKey, false)
}

func TestKeyService(t *testing.T) {
	keyServices := []crypto.KeyService{createKmsKeyService(), createCryptoKeyService(t)}
	for _, keyService := range keyServices {
		testTenantToken(t, keyService)
		testTenantEncryptDecrypt(t, keyService)
	}
}

func testTenantToken(t *testing.T, keyService crypto.KeyService) {
	token, err := keyService.GenTenantToken()
	require.NoError(t, err, "GenTenantToken failed")

	k, err := keyService.DecryptDataKey(token.EncryptedToken)
	require.NoErrorf(t, err, "DecryptDataKey failed\n")
	if len(k) != 32 {
		t.Fatalf("expect data key length to be 32, got %d", len(k))
	}
	if !reflect.DeepEqual(k, token.DecryptedToken) {
		t.Fatalf("mismatched decrypted keys")
	}
}
func testTenantEncryptDecrypt(t *testing.T, keyService crypto.KeyService) {
	tt, err := keyService.GenTenantToken()
	require.NoError(t, err)
	originalText := "Hello darkness my old friend"
	encryptedStr, err := keyService.TenantEncrypt(originalText, tt)
	require.NoError(t, err)
	dt, err := keyService.TenantDecrypt(encryptedStr, tt)
	require.NoError(t, err)
	if dt != originalText {
		t.Fatalf("Expect decrypted '%s' to match original '%s", dt, originalText)
	}
}

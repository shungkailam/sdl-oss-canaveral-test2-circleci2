package crypto

import (
	"cloudservices/common/base"
	"encoding/base64"
	"errors"

	"crypto/sha256"

	"encoding/hex"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
)

// DecryptDataKeyService separate out this base interface for the helper functions below
type DecryptDataKeyService interface {
	DecryptDataKey(dataKey string) ([]byte, error)
}

func tenantEncrypt(keyService DecryptDataKeyService, str string, token *Token) (string, error) {
	if token == nil {
		return "", errors.New("Invalid token")
	}
	if token.DecryptedToken == nil || len(token.DecryptedToken) == 0 {
		plainToken, err := keyService.DecryptDataKey(token.EncryptedToken)
		if err != nil {
			return "", err
		}
		token.DecryptedToken = plainToken
	}
	return Encrypt(str, token.DecryptedToken)
}

func tenantDecrypt(keyService DecryptDataKeyService, str string, token *Token) (string, error) {
	if token == nil {
		return "", errors.New("Invalid token")
	}
	if token.DecryptedToken == nil || len(token.DecryptedToken) == 0 {
		plainToken, err := keyService.DecryptDataKey(token.EncryptedToken)
		if err != nil {
			return "", err
		}
		token.DecryptedToken = plainToken
	}
	res, err := Decrypt(str, token.DecryptedToken)
	if err != nil {
		return "", err
	}
	return res, nil
}

func mustDecryptDataKey(keyService DecryptDataKeyService, dataKey string) []byte {
	ba, err := keyService.DecryptDataKey(dataKey)
	if err != nil {
		panic(err)
	}
	return ba
}

// Token is the token holder
type Token struct {
	EncryptedToken string
	DecryptedToken []byte
}

// KeyService interface for encryption / decryption service
type KeyService interface {
	DecryptDataKeyService
	GenTenantToken() (*Token, error)
	TenantEncrypt(str string, token *Token) (string, error)
	TenantDecrypt(str string, token *Token) (string, error)
	GetJWTSecret() []byte
}

// NewKeyService create a new KeyService
func NewKeyService(awsRegion string, jwtSecret string, awsKMSKey string, useKMS bool) KeyService {
	if useKMS {
		return newKmsKeyService(awsRegion, jwtSecret, awsKMSKey)
	}
	return newCryptoKeyService(jwtSecret, awsKMSKey)
}

// AWS KMS based KeyService implementation
type kmsKeyService struct {
	jwtSecret  []byte
	awsSession *session.Session
	awsKMS     *kms.KMS
	awsKMSKey  *string
}

func newKmsKeyService(awsRegion string, jwtSecret string, awsKMSKey string) *kmsKeyService {
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(awsRegion)},
	)
	if err != nil {
		panic(err)
	}
	keyService := &kmsKeyService{}
	keyService.awsSession = awsSession
	keyService.awsKMSKey = aws.String(awsKMSKey)
	keyService.awsKMS = kms.New(awsSession)
	keyService.jwtSecret = mustDecryptDataKey(keyService, jwtSecret)
	JWTSecret = keyService.jwtSecret
	return keyService
}

// GenTenantToken generates a new tenent token
// A tenant token is a base64 encoded, encrypted kms data key
func (keyService *kmsKeyService) GenTenantToken() (*Token, error) {
	params := &kms.GenerateDataKeyInput{
		KeyId:   keyService.awsKMSKey,
		KeySpec: aws.String("AES_256"),
	}
	dataKey, err := keyService.awsKMS.GenerateDataKey(params)
	if err != nil {
		return nil, err
	}
	token := base64.StdEncoding.EncodeToString(dataKey.CiphertextBlob)
	plainToken, err := keyService.DecryptDataKey(token)
	if err != nil {
		return nil, err
	}
	return &Token{EncryptedToken: token, DecryptedToken: plainToken}, nil
}

// DecryptDataKey decrypts an encrypted kms data key
// @param dataKey - a base64 encoded, encrypted kms data key
func (keyService *kmsKeyService) DecryptDataKey(dataKey string) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(dataKey)
	if err != nil {
		return nil, err
	}
	input := &kms.DecryptInput{
		CiphertextBlob: data,
	}
	output, err := keyService.awsKMS.Decrypt(input)
	if err != nil {
		return nil, err
	}
	return output.Plaintext, nil
}

// TenantEncrypt encrypt str using the supplied tenant token
func (keyService *kmsKeyService) TenantEncrypt(str string, token *Token) (string, error) {
	return tenantEncrypt(keyService, str, token)
}

// TenantDecrypt decrypt str using the supplied tenant token
func (keyService *kmsKeyService) TenantDecrypt(str string, token *Token) (string, error) {
	return tenantDecrypt(keyService, str, token)
}

func (keyService *kmsKeyService) GetJWTSecret() []byte {
	return keyService.jwtSecret
}

//////////////////////////////////////////////////
// Simple crypto KeyService implementation
//////////////////////////////////////////////////
// GenerateKey generate a 32 byte key, can be used as master key or data key
func GenerateKey() ([]byte, error) {
	p := base.GenerateStrongPasswordWithLength(512)
	h := sha256.New()
	_, err := h.Write([]byte(p))
	if err != nil {
		return nil, err
	}
	return h.Sum(nil), nil
}

// GenerateKeyString generate base64 encoded key
func GenerateKeyString() (string, error) {
	ba, err := GenerateKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ba), nil
}

// GenerateDataKey generate base64 encoded data key encrypted by the master key
func GenerateDataKey(masterKeyString string) (string, error) {
	masterKey, err := base64.StdEncoding.DecodeString(masterKeyString)
	if err != nil {
		return "", err
	}
	_, encryptedKey, err := generateDataKey(masterKey)
	if err != nil {
		return "", err
	}
	return encryptedKey, nil
}

type cryptoKeyService struct {
	jwtSecret []byte
	masterKey []byte
}

func newCryptoKeyService(jwtSecret string, masterKeyString string) *cryptoKeyService {
	keyService := &cryptoKeyService{}
	ba, err := base64.StdEncoding.DecodeString(masterKeyString)
	if err != nil {
		panic(err)
	}
	keyService.masterKey = ba
	if len(ba) != 32 {
		panic("master key length should be 32")
	}
	keyService.jwtSecret = mustDecryptDataKey(keyService, jwtSecret)
	JWTSecret = keyService.jwtSecret
	return keyService
}

func generateDataKey(masterKey []byte) ([]byte, string, error) {
	plainDataKey, err := GenerateKey()
	if err != nil {
		return nil, "", err
	}
	hexPlainDataKey := hex.EncodeToString(plainDataKey)
	hexEncryptedDataKey, err := Encrypt(hexPlainDataKey, masterKey)
	if err != nil {
		return nil, "", err
	}
	baEncryptedDataKey, err := hex.DecodeString(hexEncryptedDataKey)
	if err != nil {
		return nil, "", err
	}
	return plainDataKey, base64.StdEncoding.EncodeToString(baEncryptedDataKey), nil
}

// GenTenantToken generates a new tenent token
// A tenant token is a base64 encoded, encrypted kms data key
func (keyService *cryptoKeyService) GenTenantToken() (*Token, error) {
	plainKey, encryptedKey, err := generateDataKey(keyService.masterKey)
	if err != nil {
		return nil, err
	}
	return &Token{EncryptedToken: encryptedKey, DecryptedToken: plainKey}, nil
}

// DecryptDataKey decrypts an encrypted kms data key
// @param dataKey - a base64 encoded, encrypted kms data key
func (keyService *cryptoKeyService) DecryptDataKey(dataKey string) ([]byte, error) {
	ba, err := base64.StdEncoding.DecodeString(dataKey)
	if err != nil {
		return nil, err
	}
	bas := hex.EncodeToString(ba)
	dbas, err := Decrypt(bas, keyService.masterKey)
	if err != nil {
		return nil, err
	}
	return hex.DecodeString(dbas)
}

// TenantEncrypt encrypt str using the supplied tenant token
func (keyService *cryptoKeyService) TenantEncrypt(str string, token *Token) (string, error) {
	return tenantEncrypt(keyService, str, token)
}

// TenantDecrypt decrypt str using the supplied tenant token
func (keyService *cryptoKeyService) TenantDecrypt(str string, token *Token) (string, error) {
	return tenantDecrypt(keyService, str, token)
}

func (keyService *cryptoKeyService) GetJWTSecret() []byte {
	return keyService.jwtSecret
}

package crypto

import (
	"cloudservices/common/base"
	"crypto"
	cryptoLib "crypto"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/golang/glog"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"

	"golang.org/x/crypto/bcrypt"

	"encoding/hex"
	"encoding/pem"

	"io"
)

var salt = GetRawSha256("sherlock")

// GetRawSha256 get sha256 of the string
func GetRawSha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetSha256 get sha256 of the string with salt
func GetSha256(s string) string {
	return GetRawSha256(s + salt)
}

// EncryptPassword encrypt user password (for store in DB)
// It is also used to generate EdgeHandleToken from edgeID
func EncryptPassword(password string) (string, error) {
	// Hashing the password + salt with the default cost of 10
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password+salt), bcrypt.DefaultCost)
	if err != nil {
		// should not happen, just return original
		glog.Errorf("EncryptPassword failed for %s", base.MaskString(password, "*", 0, 4))
		return "", err
	}
	return string(hashedPassword), nil
}

// MatchHashAndPassword checks whether the hash of password match
func MatchHashAndPassword(hash string, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password+salt))
	if err == nil {
		return true
	}
	return GetSha256(password) == hash
}

// Encrypt encrypt str using the secret and AES 256 GCM algorithm
func Encrypt(str string, key []byte) (string, error) {
	plaintext := []byte(str)
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	ciphertext := aesgcm.Seal(nil, nonce, plaintext, nil)
	hct := fmt.Sprintf("%x%x", nonce, ciphertext)
	return hct, nil
}

// Decrypt decrypt str using the secret and AES 256 CBC algorithm
func Decrypt(str string, key []byte) (string, error) {
	ba, err := hex.DecodeString(str)
	if err != nil {
		return "", err
	}
	if len(ba) < 12 {
		return "", fmt.Errorf("Decrypt: bad input %s", str)
	}
	nonce := ba[:12]
	ciphertext := ba[12:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil
	}

	plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	dstr := fmt.Sprintf("%s", plaintext)
	return dstr, nil
}

// VerifySignature - use public key from certificate to verify
// the given signature matches the msg signed by the private key
func VerifySignature(certificate string, msg string, signature string) error {
	block, _ := pem.Decode([]byte(certificate))
	if block == nil {
		return errors.New("Failed to decode certificate block")
	}
	var cert *x509.Certificate
	var err error
	cert, err = x509.ParseCertificate(block.Bytes)
	if err != nil {
		return err
	}
	rsaPublicKey := cert.PublicKey.(*rsa.PublicKey)
	hashed := sha256.Sum256([]byte(msg))
	dsig, err := hex.DecodeString(signature)
	if err != nil {
		return err
	}
	err = rsa.VerifyPKCS1v15(rsaPublicKey, cryptoLib.SHA256, hashed[:], dsig)
	if err != nil {
		return err
	}
	return nil
}

// GetEdgeEmailPassword get email and password edge can use to login
func GetEdgeEmailPassword(tenantID string, edgeID string, key string) (email string, password string, err error) {
	email = tenantID + "|" + edgeID
	hashed := sha256.Sum256([]byte(email))
	rng := rand.Reader
	block, _ := pem.Decode([]byte(key))
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		err = fmt.Errorf("Failed to decode PEM block containing RSA PRIVATE KEY")
		return
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return
	}
	signature, err := rsa.SignPKCS1v15(rng, privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return
	}
	password = hex.EncodeToString(signature)
	return
}

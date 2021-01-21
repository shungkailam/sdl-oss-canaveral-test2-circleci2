package crypto

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"fmt"

	jwt "github.com/dgrijalva/jwt-go"
)

var JWTSecret []byte

// SignJWT signs the jwt map claim using JWTSecret and returns the signed token
func SignJWT(claims jwt.MapClaims) (string, error) {
	if len(JWTSecret) == 0 {
		return "", errors.New("JWT secret is not set")
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// VerifyJWT verified the token string using JWTSecret and returns the jwt map claim
func VerifyJWT(tokenString string) (jwt.MapClaims, error) {
	if len(JWTSecret) == 0 {
		return nil, errors.New("JWT secret is not set")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		// JWTSecret is a []byte containing your secret
		return JWTSecret, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid jwt")
}

// VerifyJWT2 verified the token string and returns the jwt map claim
// Difference between VerifyJWT and VerifyJWT2: VerifyJWT only verifies HMAC signing algorithm,
// VerifyJWT2 further uses publicKeyResolver to lookup current user public key and use it to
// verify jwt signed by RSA and ECDSA algorithms.
func VerifyJWT2(tokenString string, publicKeyResolver func() func(*jwt.Token) (interface{}, error)) (jwt.MapClaims, error) {
	if len(JWTSecret) == 0 {
		return nil, errors.New("JWT secret is not set")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); ok {
			// JWTSecret is a []byte containing your secret
			return JWTSecret, nil
		}
		if _, ok := token.Method.(*jwt.SigningMethodRSA); ok {
			publicKeyResolverFn := publicKeyResolver()
			pk, err := publicKeyResolverFn(token)
			if err != nil {
				return nil, err
			}
			verifyKey, ok := pk.(*rsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("Failed to resolve RSA public key")
			}
			return verifyKey, nil
		}
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); ok {
			publicKeyResolverFn := publicKeyResolver()
			pk, err := publicKeyResolverFn(token)
			if err != nil {
				return nil, err
			}
			verifyKey, ok := pk.(*ecdsa.PublicKey)
			if !ok {
				return nil, fmt.Errorf("Failed to resolve ECDSA public key")
			}
			return verifyKey, nil
		}
		return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid jwt")
}

// RSASignJWT signs the jwt map claim using rsa.PrivateKey and returns the signed token
func RSASignJWT(signKey *rsa.PrivateKey, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.GetSigningMethod("RS256"), claims)
	return token.SignedString(signKey)
}

// RSAVerifyJWT verified the token string using rsa.PublicKey and returns the jwt map claim
func RSAVerifyJWT(verifyKey *rsa.PublicKey, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid jwt")
}

// ECDSASignJWT signs the jwt map claim using ecdsa.PrivateKey and returns the signed token
func ECDSASignJWT(signKey *ecdsa.PrivateKey, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.GetSigningMethod("ES512"), claims)
	return token.SignedString(signKey)
}

// ECDSAVerifyJWT verified the token string using ecdsa.PublicKey and returns the jwt map claim
func ECDSAVerifyJWT(verifyKey *ecdsa.PublicKey, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodECDSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return verifyKey, nil
	})
	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid jwt")
}

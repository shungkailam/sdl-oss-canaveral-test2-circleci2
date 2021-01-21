package api

import (
	"cloudservices/common/auth"
	"cloudservices/common/crypto"
	"cloudservices/common/model"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// TokenType identifies the token
type TokenType string

const (
	ShortTokenType   = TokenType("short")
	DefaultTokenType = TokenType("default")
	LongTokenType    = TokenType("long")
	ApiKeyTokenType  = TokenType("api")
)

// GetUserJWTToken returns JWT token for users (not edge)
func GetUserJWTToken(dbAPI ObjectModelAPI, user *model.User, authToken *auth.OAuthToken,
	lifetime int64, tokenType TokenType, inClaims jwt.MapClaims) string {
	claims := jwt.MapClaims{
		"tenantId":    user.TenantID,
		"id":          user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"specialRole": model.GetUserSpecialRole(user),
		"roles":       []string{},
		"scopes":      []string{},
		"nbf":         time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	}
	if authToken != nil {
		claims["refreshToken"] = authToken.RefreshToken
	}
	if len(tokenType) > 0 && tokenType != DefaultTokenType {
		claims["type"] = tokenType
	}
	if lifetime != 0 {
		claims["exp"] = time.Now().Unix() + lifetime
	}

	if inClaims != nil {
		for k, v := range inClaims {
			if _, ok := claims[k]; ok {
				// do not override any existing keys
				continue
			}
			claims[k] = v
		}
	}
	token, _ := crypto.SignJWT(claims)
	return token
}

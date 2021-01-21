package auth

import (
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
)

func LoginHandler(token string) (interface{}, error) {
	const prefix = "Bearer "

	token = strings.TrimPrefix(token, prefix)
	// var principal models.Principal
	// principal = "operator"
	claims, err := crypto.VerifyJWT(token)
	if err != nil {
		glog.Errorln(err)
		// On error do not return err as not handled by the library,
		// Empty claim will be rejected later
		return jwt.MapClaims{}, nil
	}

	// var principal models.Principal
	// principal = "operator"
	authContext := &base.AuthContext{TenantID: claims["tenantId"].(string), Claims: claims}

	return authContext, nil
}

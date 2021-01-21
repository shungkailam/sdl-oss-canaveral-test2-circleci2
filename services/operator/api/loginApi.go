package api

import (
	gapi "cloudservices/account/generated/grpc"
	cauth "cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/service"
	"cloudservices/operator/generated/operator/models"
	"cloudservices/operator/generated/operator/restapi/operations/operator"
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-openapi/runtime/middleware"
	"github.com/golang/glog"
	"google.golang.org/grpc"
)

const DefaultTokenLifetimeSec = 60 * 60 // 60 mins

func (server *APIServer) LoginHandler(params operator.LoginParams) middleware.Responder {
	reqID := base.GetUUID()

	glog.Infof("Request %s: Recieved login req", reqID)
	request := &gapi.GetUserByEmailRequest{Email: params.LoginParams.Email}
	user := &gapi.User{}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAccountServiceClient(conn)
		response, err := client.GetUserByEmail(ctx, request)
		if err != nil {
			//glog.Errorf(base.PrefixRequestID(ctx, "Failed in GetUserByEmail for email %s. Error: %s"), params.LoginParams.Email, err.Error())
			glog.Errorf("Failed in GetUserByEmail for email %s. Error: %s", params.LoginParams.Email, err.Error())
			return err
		}
		user = response.GetUser()
		return nil
	}
	err := service.CallClient(params.HTTPRequest.Context(), service.AccountService, handler)
	if err != nil {
		glog.Errorf("Failed in account service call. Error: %s", err.Error())
		errStr := fmt.Sprintf("Login Failed invalid creds")
		retErr := &models.Error{Message: &errStr}
		return operator.NewLoginDefault(http.StatusUnauthorized).WithPayload(retErr)
	}
	resp := &models.LoginResponse{}
	if crypto.MatchHashAndPassword(user.Password, params.LoginParams.Password) && cauth.OperatorRole == strings.ToLower(user.Role) {
		specialRole := "operator"
		claims := jwt.MapClaims{
			"tenantId":    user.TenantId,
			"id":          user.Id,
			"name":        user.Name,
			"email":       user.Email,
			"specialRole": specialRole,
			"roles":       []string{},
			"scopes":      []string{},
			"nbf":         time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			"exp":         time.Now().Unix() + DefaultTokenLifetimeSec,
		}
		// if authToken != nil {
		// 	claims["refreshToken"] = authToken.RefreshToken
		// }
		token, _ := crypto.SignJWT(claims)
		resp.Token = token
	} else {
		glog.Errorf("Request %s: Login Failed invalid creds", reqID)
		errStr := fmt.Sprintf("Login Failed invalid creds")
		retErr := &models.Error{Message: &errStr}
		return operator.NewLoginDefault(http.StatusUnauthorized).WithPayload(retErr)
	}
	return operator.NewLoginOK().WithPayload(resp)
}

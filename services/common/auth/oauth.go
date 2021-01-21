package auth

import (
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/golang/glog"
	funk "github.com/thoas/go-funk"
	"golang.org/x/oauth2"
)

const (
	OAuthAPICallTimeOutSec = 3
	XIAdminRole            = "iot-admin"
	XIUserRole             = "iot-user"
)

var (
	XIRoles           = []string{XIAdminRole, XIUserRole}
	oAuthHandlers     = map[string]*OAuthHandler{}
	oAuthHandlerMutex = &sync.Mutex{}
)

type OAuthToken struct {
	RefreshToken string
	IDToken      jwt.MapClaims
}

type OAuthHandler struct {
	config      *oauth2.Config
	serviceType string
}

type XIIOTRole struct {
	Role         string
	TenantDomain string
	TenantName   string
}

type CreateMyNutanixUserResponse struct {
	TenantID   string `json:"tenantUUID"`
	StatusCode int    `json:"statusCode"`
}

// NewOAuthHandler creates an OAuth handler with the parameters
func NewOAuthHandler(serviceType, clientID, clientSecret, identityProvider, redirectURL string) *OAuthHandler {
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes: []string{
			"openid",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth2/authorize", identityProvider),
			TokenURL: fmt.Sprintf("%s/oauth2/token", identityProvider),
		},
	}
	return &OAuthHandler{config: config, serviceType: serviceType}
}

// RegisterOAuthHandler creates an OAuth handler and registers it with the service type
func RegisterOAuthHandler(serviceType, clientID, clientSecret, identityProvider, redirectURL string) {
	oAuthHandlerMutex.Lock()
	defer oAuthHandlerMutex.Unlock()
	oAuthHandlers[strings.ToLower(serviceType)] = NewOAuthHandler(serviceType, clientID, clientSecret, identityProvider, redirectURL)
}

// LookupOAuthHandler returns the OAuth handler for the service type
func LookupOAuthHandler(serviceType string) *OAuthHandler {
	oAuthHandlerMutex.Lock()
	defer oAuthHandlerMutex.Unlock()
	return oAuthHandlers[strings.ToLower(serviceType)]
}

// VerifyAuthorization verifies the authorization header value and returns the JWT claims
func VerifyAuthorization(r *http.Request, publicKeyResolver func() func(*jwt.Token) (interface{}, error), claimVerifier func() func(jwt.MapClaims) error) (jwt.MapClaims, error) {
	auth := r.Header.Get("authorization")
	m := reBearer.FindStringSubmatch(auth)
	if len(m) < 2 {
		return jwt.MapClaims{}, errcode.NewInvalidCredentialsError()
	}
	token := m[1]
	claims, err := crypto.VerifyJWT2(token, publicKeyResolver)
	if err != nil {
		return jwt.MapClaims{}, errcode.NewInvalidCredentialsError()
	}
	if IsTrialExpired(claims) {
		return jwt.MapClaims{}, errcode.NewInvalidCredentialsError()
	}
	if token, _ := jwt.Parse(token, nil); token != nil {
		// if token is not signed by jwt secret, limit exp duration to 30 minutes
		if token.Header["alg"] != "HS256" {
			reqID := r.Header.Get("X-Request-ID")
			if len(reqID) == 0 {
				reqID = base.GetUUID()
			}
			exp, ok := claims["exp"].(float64)
			if ok {
				secondsToExpire := exp - base.RoundedNow().Sub(time.Unix(0, 0)).Seconds()
				if secondsToExpire > 1800 {
					glog.Errorf("Request %s: Error JWT token expiration too long: %fs (max = 1800s)", reqID, secondsToExpire)
					return jwt.MapClaims{}, errcode.NewInvalidCredentialsError()
				}
			} else {
				glog.Errorf("Request %s: Error JWT token expiration required", reqID)
				return jwt.MapClaims{}, errcode.NewInvalidCredentialsError()
			}
		} else {
			// if it's long lived jwt api token,
			// then use claim verifier to check if the token is still valid
			typ, ok := claims["type"].(string)
			if ok && typ == "api" {
				claimVerifierFn := claimVerifier()
				err = claimVerifierFn(claims)
				if err != nil {
					return jwt.MapClaims{}, errcode.NewInvalidCredentialsError()
				}
			}
		}
	}
	return claims, nil
}

// IsTrialExpired checks for expiry only if trialExpiry is present
func IsTrialExpired(claims jwt.MapClaims) bool {
	now := time.Now().UTC().Unix()
	switch exp := claims["trialExpiry"].(type) {
	case float64:
		return int64(exp) < now
	case json.Number:
		v, _ := exp.Int64()
		return v < now
	default:
		// trialExpiry is absent
		return false
	}
}

// GetAuthRedirectURL gets the oauth authorization endpoint.
func (handler *OAuthHandler) GetAuthRedirectURL(state string) string {
	config := handler.config
	return config.AuthCodeURL(state)
}

// ExchangeForToken gets the id token and refresh token from the auth code.
func (handler *OAuthHandler) ExchangeForToken(ctx context.Context, authCode string) (*OAuthToken, error) {
	config := handler.config
	// Use the custom HTTP client when requesting a token.
	ctx = getContextWithHTTPClient(ctx)
	token, err := config.Exchange(ctx, authCode)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error occurred in getting the token for code: %s. Error: %s"), authCode, err.Error())
		return nil, errcode.NewBadRequestError("authCode")
	}
	return getOAuthToken(ctx, token)
}

// RefreshToken refreshes the existing token
func (handler *OAuthHandler) RefreshToken(ctx context.Context, refreshToken string) (*OAuthToken, error) {
	config := handler.config
	// Use the custom HTTP client when requesting a token.
	ctx = getContextWithHTTPClient(ctx)
	oauthToken, err := config.TokenSource(ctx, &oauth2.Token{RefreshToken: refreshToken}).Token()
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in refreshing token: %s. Error: %s"), refreshToken, err.Error())
		return nil, errcode.NewBadRequestError("refreshToken")
	}
	return getOAuthToken(ctx, oauthToken)
}

// GetOAuthToken returns the OAuthToken which contains all the tokens returned by IDP
func (handler *OAuthHandler) GetOAuthToken(ctx context.Context, authCodes *model.OAuthCodes) (*OAuthToken, error) {
	if authCodes == nil {
		glog.Infof(base.PrefixRequestID(ctx, "Null OAuthCodes"))
		return nil, errcode.NewBadRequestError("OAuthCodes")
	}
	var err error
	var authToken *OAuthToken
	if len(authCodes.Code) > 0 {
		glog.Infof(base.PrefixRequestID(ctx, "Getting the token for code: %s"), authCodes.Code)
		authToken, err = handler.ExchangeForToken(ctx, authCodes.Code)
	} else if len(authCodes.RefreshToken) > 0 {
		glog.Infof(base.PrefixRequestID(ctx, "Refreshing the token for refresh token: %s"), authCodes.RefreshToken)
		// There is a known issue that id_token does not xi_iot field for refresh_token grant
		authToken, err = handler.RefreshToken(ctx, authCodes.RefreshToken)
	} else {
		glog.Infof(base.PrefixRequestID(ctx, "Bad input: %+v"), *authCodes)
		return nil, errcode.NewInvalidCredentialsError()
	}
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in getting the auth token. Error: %s"), err.Error())
		return nil, errcode.NewInvalidCredentialsError()
	}
	return authToken, nil
}

// GetAssignXIIOTRoleURL returns the endpoint for role Xi IoT role assignment
func (handler *OAuthHandler) GetAssignXIIOTRoleURL() string {
	// e.g IoT, GrayMatter
	pathSuffix := strings.ToLower(handler.serviceType)
	return handler.GetMyNutanixURL(fmt.Sprintf("/api/v1/auth/%s", pathSuffix))
}

// GetAssignXIIOTRoleURLWithLandingURL accepts a custom landing page after the role is assigned
func (handler *OAuthHandler) GetAssignXIIOTRoleURLWithLandingURL(landingURL string) string {
	// https://demo-my.nutanix.com/api/v1/auth/iot?RelayState=<landing page>
	return handler.GetMyNutanixURL("/api/v1/auth/iot")
	//return handler.GetMyNutanixURL(fmt.Sprintf("/api/v1/auth/iot?RelayState=%s", landingURL))
}

// GetMyNutanixURL returns the my-nutanix endpoint with the pathSuffix appended if it is not empty
func (handler *OAuthHandler) GetMyNutanixURL(pathSuffix string) string {
	endpoint := handler.config.Endpoint
	myNutanixURL := "https://demo-my.nutanix.com"
	if strings.Contains(endpoint.AuthURL, "://idp.") {
		myNutanixURL = "https://my.nutanix.com"
	} else if strings.Contains(endpoint.AuthURL, "://idp-stage.") {
		myNutanixURL = "https://stage-my.nutanix.com"
	}
	pathSuffix = strings.TrimLeft(pathSuffix, "/")
	if len(pathSuffix) > 0 {
		return fmt.Sprintf("%s/%s", myNutanixURL, pathSuffix)
	}
	return myNutanixURL
}

// GetIdentityUsersCallInfo returns the my-nutanix identity/users endpoint along with the authentication JWT
// The public keys are already with my-nutanix service
func (handler *OAuthHandler) GetIdentityUsersCallInfo(ctx context.Context) (url, jwtToken string, err error) {
	endpoint := handler.config.Endpoint // SignJWT signs the jwt map claim and returns the signed token
	seconds := time.Now().Unix()
	var privateKey string
	var claims jwt.MapClaims
	if strings.Contains(endpoint.AuthURL, "://idp.") {
		claims = jwt.MapClaims{
			"iss": "1eaf1492-cbb7-476f-b13a-9bb765e677dc",
			"aud": "https://my.nutanix.com",
			"iat": seconds,
			"exp": seconds + 60*60,
		}
		privateKey = PrivateKeyProd
	} else if strings.Contains(endpoint.AuthURL, "://idp-stage.") {
		claims = jwt.MapClaims{
			"iss": "d9422b6f-7704-4483-9346-cdb68eff85c2",
			"aud": "https://stage-my.nutanix.com",
			"iat": seconds,
			"exp": seconds + 60*60,
		}
		privateKey = PrivateKeyStage
	} else {
		claims = jwt.MapClaims{
			"iss": "d9422b6f-7704-4483-9346-cdb68eff85c2",
			"aud": "https://demo-my.nutanix.com",
			"iat": seconds,
			"exp": seconds + 60*60,
		}
		privateKey = PrivateKeyDev
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Got my-nutanix claim %+v"), claims)
	jwtToken, err = SignClaimsWithPrivateKey(ctx, privateKey, claims)
	if err != nil {
		return
	}
	url = fmt.Sprintf("%s/api/v2/identity/users", claims["aud"])
	return
}

// CreateMyNutanixIOTUser creates IoT account admin user with iot-admin role
// The response can be 200 or 201 for success
func (handler *OAuthHandler) CreateMyNutanixIOTUser(ctx context.Context, email, firstName, lastName string) (*CreateMyNutanixUserResponse, error) {
	url, jwtToken, err := handler.GetIdentityUsersCallInfo(ctx)
	if err != nil {
		return nil, err
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport}
	// If the user already exists, the names are optional. Otherwise, the user gets an email to reset password and name.
	payload := fmt.Sprintf(`{"email":"%s", "firstName": "%s", "lastName": "%s", "targetId": "iot"}`, email, firstName, lastName)
	request, err := http.NewRequest("POST", url, strings.NewReader(payload))
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", jwtToken))
	response, err := client.Do(request)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to invoke my-nutanix identity/users with payload %+v. Error: %s"), payload, err.Error())
		return nil, errcode.NewBadRequestError("user")
	}
	userResponse := &CreateMyNutanixUserResponse{}
	reader := io.Reader(response.Body)
	err = base.Decode(&reader, userResponse)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to decode response from my-nutanix identity/users with payload %+v. Error: %s"), payload, err.Error())
		return nil, err
	}
	userResponse.StatusCode = response.StatusCode
	return userResponse, nil
}

/*
GetXIIOTRole retrieves the XI IOT role from the id_token claims.

The XI role is received like below from my.nutanix.com

xi_role: [
	{
		"account_approved": true,
		"roles": [
			{
				"name": "internal-tenant-admin"
			},
			{
				"name": "iot-admin"
			},
			{
				"name": "account-admin"
			}
		],
		"tenant-domain": "ca1d7dda-7a82-408e-bc0c-c842eee190ec",
		"tenant-name": "",
		"tenant-properties": {
			"tenant-uuid": "ca1d7dda-7a82-408e-bc0c-c842eee190ec"
		},
		"tenant-status": "PROVISIONED"
	}
]
*/
func GetXIIOTRole(ctx context.Context, oAuthToken *OAuthToken) (*XIIOTRole, error) {
	idToken := oAuthToken.IDToken
	xiRole, ok := idToken["xi_role"].(string)
	if !ok {
		glog.Errorf(base.PrefixRequestID(ctx, "XI role is not present in claims %+v"), idToken)
		return nil, errcode.NewInvalidCredentialsError()
	}
	bytes, err := jwt.DecodeSegment(xiRole)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error decoding XI role claims %+v"), idToken)
		return nil, errcode.NewInvalidCredentialsError()
	}
	claimsArray := []jwt.MapClaims{}
	if err = json.Unmarshal(bytes, &claimsArray); err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error unmarshalling XI role claims %s"), string(bytes))
		return nil, errcode.NewInvalidCredentialsError()
	}
	for _, claims := range claimsArray {
		tenantDomain, ok := claims["tenant-domain"].(string)
		if !ok {
			continue
		}
		tenantName, ok := claims["tenant-name"].(string)
		if !ok {
			// May not be present
			tenantName = "NA"
		}
		roleIfs, ok := claims["roles"].([]interface{})
		if !ok {
			continue
		}
		for _, roleItf := range roleIfs {
			roleMap, ok := roleItf.(map[string]interface{})
			if !ok {
				continue
			}
			role, ok := roleMap["name"].(string)
			if !ok {
				continue
			}
			if funk.Contains(XIRoles, role) {
				glog.Infof(base.PrefixRequestID(ctx, "XI IOT role %s is found for tenant domain %s"), role, tenantDomain)
				return &XIIOTRole{Role: role, TenantDomain: tenantDomain, TenantName: tenantName}, nil
			}
		}
	}
	glog.Errorf(base.PrefixRequestID(ctx, "XI IOT role is not present in the claims %+v"), claimsArray)
	return nil, errcode.NewInvalidCredentialsError()
}

// SignClaimsWithPrivateKey signs the claims with the private key
func SignClaimsWithPrivateKey(ctx context.Context, key string, claims jwt.MapClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(key))
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to sign claims %+v. Error: %s"), claims, err.Error())
		return "", errcode.NewInternalError(err.Error())
	}
	return token.SignedString(privateKey)
}

// getOAuthToken extracts the fields from the JWT token
func getOAuthToken(ctx context.Context, oauthToken *oauth2.Token) (*OAuthToken, error) {
	refreshToken := oauthToken.Extra("refresh_token").(string)
	idTokenClaims, err := extractTokenClaims(ctx, oauthToken, "id_token")
	if err != nil {
		return nil, err
	}
	return &OAuthToken{RefreshToken: refreshToken, IDToken: idTokenClaims}, nil
}

// extractToken extracts token fields from the JWT token
func extractTokenClaims(ctx context.Context, oauthToken *oauth2.Token, tokenName string) (jwt.MapClaims, error) {
	token := oauthToken.Extra(tokenName).(string)
	// It is a JWT token
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid token for %s"), token)
		return nil, errcode.NewInternalError("Invalid token response")
	}
	claims := jwt.MapClaims{}
	payloadBytes, err := jwt.DecodeSegment(parts[1])
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid payload for token %s"), token)
		return nil, errcode.NewInternalError("Invalid payload in token response")
	}
	if err = json.Unmarshal(payloadBytes, &claims); err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Invalid payload for token %s. Error: %s"), tokenName, err.Error())
		return nil, errcode.NewInternalError("Invalid payload in token response")
	}
	return claims, nil
}

// Do not set the http client if it is passed for testing
func getContextWithHTTPClient(ctx context.Context) context.Context {
	httpClient, ok := ctx.Value(oauth2.HTTPClient).(*http.Client)
	if !ok {
		httpClient = &http.Client{Timeout: OAuthAPICallTimeOutSec * time.Second}
		ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)
	}
	return ctx
}

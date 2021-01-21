package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/crypto"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	tenantpoolcore "cloudservices/tenantpool/core"
	tenantpool "cloudservices/tenantpool/model"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/go-redis/redis"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
)

const (
	// shorter lifetime to pick up RBAC change sooner
	DefaultTokenLifetimeSec           = 30 * 60           // 30 mins
	DefaultLongTokenLifeTimeSec       = 20 * 24 * 60 * 60 // 20 days
	EdgeTokenLifetimeSec              = 24 * 60 * 60      // 1 day
	DefaultShortLoginTokenLifetimeSec = 3 * 60            // 3 min
	DefaultInClaimKeyLen              = 16
	DefaultInClaimValLen              = 64
	DefaultInClaimsLen                = 8
)

var (
	// They are variables because test overrides it.
	// Nutanix emails already existing in IoT DB do not have a registration
	// NonNutanixNonTrialRegistrationID is used by non-trial non-nutanix emails
	NonNutanixNonTrialRegistrationID = "DEFAULT_NON_TRIAL_REGISTRATION"
	// NonNutanixRegistrationID is used by trial non-nutanix emails
	NonNutanixRegistrationID = "DEFAULT_REGISTRATION"
	// NutanixRegistrationID is used by trial nutanix emails
	NutanixRegistrationID = "DEFAULT_NUTANIX_REGISTRATION"
)

// RegisterOAuthHandlers registers all OAuth handlers
func RegisterOAuthHandlers(dbAPI api.ObjectModelAPI, redirectURLs []string) error {
	for _, redirectURL := range redirectURLs {
		hostname := redirectURL
		u, err := url.Parse(redirectURL)
		if err != nil {
			glog.Errorf("Failed to register OAuth redirect URL %s. Error: %s", redirectURL, err.Error())
			return err
		}
		if hn := u.Hostname(); len(hn) > 0 {
			hostname = hn
		}
		query := u.Query()
		values := query["clientId"]
		if len(values) == 0 || len(values[0]) == 0 {
			err := fmt.Errorf("Invalid client ID in URL %s", redirectURL)
			glog.Error(err)
			return err
		}
		clientID := values[0]
		values = query["clientSecret"]
		if len(values) == 0 || len(values[0]) == 0 {
			err := fmt.Errorf("Invalid client secret in URL %s", redirectURL)
			glog.Error(err)
			return err
		}
		clientSecret := values[0]
		// Remove all queries
		u.RawQuery = ""
		rURL := u.String()
		service, _ := dbAPI.GetServices(context.Background(), hostname)
		glog.Infof("Registering OAuth handler for %s - redirect URL: %s, client ID: %s", service.ServiceType, rURL, clientID)
		auth.RegisterOAuthHandler(service.ServiceType, clientID, clientSecret, *config.Cfg.IdentityProvider, rURL)
	}
	return nil
}

// LookupOAuthHandler returns the OAuth handler for the service type derived from the request
func LookupOAuthHandler(dbAPI api.ObjectModelAPI, r *http.Request) (*auth.OAuthHandler, error) {
	ctx := r.Context()
	service, err := dbAPI.GetServicesInternal(ctx, r)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get service. Error: %s"), err.Error())
		return nil, err
	}
	oAuthHandler := auth.LookupOAuthHandler(service.ServiceType)
	if oAuthHandler == nil {
		err := errcode.NewRecordNotFoundError("Auth handler")
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get OAuth handler for service %s. Error: %s"), service.ServiceType, err.Error())
		return nil, err
	}
	return oAuthHandler, nil
}

func getLoginRoutes(dbAPI api.ObjectModelAPI, redisClient *redis.Client) []routeHandle {
	var loginTracker = auth.NewLoginTracker(redisClient, *config.Cfg.LoginFailureCountThreshold,
		time.Duration(int64(*config.Cfg.LoginLockDurationSeconds)*int64(time.Second)))
	return []routeHandle{
		{
			method: "POST",
			path:   "/v1/login",
			// swagger:route POST /v1/login LoginCall
			//
			// Login user. ntnx:ignore
			//
			// Lets the user log in.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: LoginResponseWrapper
			//       401: LoginFailedError
			//       default: APIError
			handle: CreateBasicLoginHandler(dbAPI, loginTracker),
		},
		{
			method: "POST",
			path:   "/v1.0/login",
			// swagger:route POST /v1.0/login Auth LoginCallV2
			//
			// Lets the user log in.
			//
			// Lets the user log in.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: LoginResponseWrapper
			//       401: LoginFailedError
			//       default: APIError
			handle: CreateBasicLoginHandler(dbAPI, loginTracker),
		},
		{
			method: "POST",
			path:   "/v1.0/login/shortlogintoken",
			// swagger:route POST /v1.0/login/shortlogintoken Auth ShortLoginTokenV1
			//
			// Generate a short login token.
			//
			// Generates a temporary login token valid for a short duration.
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//        - BearerToken:
			//
			//     Responses:
			//       200: LoginResponseWrapper
			//       default: APIError
			handle: CreateShortLoginToken(dbAPI, loginTracker),
		},
		{
			method: "POST",
			path:   "/v1.0/login/logintoken",
			// swagger:route POST /v1.0/login/logintoken Auth LoginTokenV1
			//
			// Get a login token
			//
			// Generates a login token equivalent to logging in.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: LoginResponseWrapper
			//       default: APIError
			handle: CreateLoginToken(dbAPI, loginTracker),
		},
		{
			method: "GET",
			path:   "/v1/oauth2/authorize",
			// swagger:route GET /v1/oauth2/authorize OAuthAuthorizeCall
			//
			// Login via MyNutanix. ntnx:ignore
			//
			// This will redirect the user to the oauth login page.
			// Note: UI only API
			//
			//     Produces:
			//     - application/html
			//
			//     Responses:
			//       401: LoginFailedError
			//       default: APIError
			handle: CreateOAuthLoginHandler(dbAPI),
		},
		{
			method: "GET",
			path:   "/v1.0/oauth2/authorize",
			// swagger:route GET /v1.0/oauth2/authorize Auth OAuthAuthorizeCallV2
			//
			// Login through MyNutanix. ntnx:ignore
			//
			// This will redirect the user to the oauth login page.
			// Note: UI only API
			//
			//     Produces:
			//     - application/html
			//
			//     Responses:
			//       401: LoginFailedError
			//       default: APIError
			handle: CreateOAuthLoginHandler(dbAPI),
		},
		{
			method: "POST",
			path:   "/v1/oauth2/token",
			// swagger:route POST /v1/oauth2/token OAuthTokenCall
			//
			// Refresh token via MyNutanix. ntnx:ignore
			//
			// This will get the session token from the auth token.
			// Note: UI only API
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: LoginResponseWrapper
			//       401: LoginFailedError
			//       default: APIError
			handle: CreateOAuthTokenHandler(dbAPI),
		},
		{
			method: "POST",
			path:   "/v1.0/oauth2/token",
			// swagger:route POST /v1.0/oauth2/token Auth OAuthTokenCallV2
			//
			// Refresh token via MyNutanix. ntnx:ignore
			//
			// This will get the session token from the auth token.
			// Note: UI only API
			//
			//     Consumes:
			//     - application/json
			//
			//     Produces:
			//     - application/json
			//
			//     Responses:
			//       200: LoginResponseWrapper
			//       401: LoginFailedError
			//       default: APIError
			handle: CreateOAuthTokenHandler(dbAPI),
		},
	}
}

// LoginResponse describes login response
// swagger:model LoginResponse
type LoginResponse struct {
	// required: true
	Token string `json:"token"`
	// required: true
	ID string `json:"_id"`
	// required: true
	Name string `json:"name"`
	// required: true
	Email string `json:"email"`
}

// Ok
// swagger:response LoginResponseWrapper
type LoginResponseWrapper struct {
	// in: body
	// required: true
	Payload *LoginResponse
}

type StateParameter struct {
	ReturnURL string `json:"return_url"`
	Random    string `json:"random"`
}

func verifyEdgeSignatureWithOldAndNewCerts(edgeCert model.EdgeCert, email string, password string) error {
	// First we verify signature with the edge certificates generated using per-tenant root CA.
	// If that fails, we verify signature using the edge certificates generated using fixed root CA.
	var err error
	err = crypto.VerifySignature(edgeCert.EdgeCertificate, email, password)
	if err != nil {
		err = crypto.VerifySignature(edgeCert.Certificate, email, password)
	}
	return err
}

// CreateLoginHandler - create http router handler for login endpoint
func CreateBasicLoginHandler(dbAPI api.ObjectModelAPI, loginTracker *auth.LoginTracker) httprouter.Handle {
	return getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var reader io.Reader = r.Body
		doc := model.Credential{}
		err := base.Decode(&reader, &doc)
		if err != nil {
			handleResponse(w, r, err, "Invalid data. Login failed")
			return
		}
		// check if email locked due to too many login failures
		// apply this to both normal users and edge users
		if loginTracker.IsLoginLocked(doc.Email) {
			// email locked, send 403 error instead of 401
			handleResponse(w, r, errcode.NewPermissionDeniedError("locked"), "Login failed for %s", doc.Email)
			return
		}
		ctx := r.Context()
		te := strings.Split(doc.Email, "|")
		if len(te) == 2 {
			// Edge login
			tenantId := te[0]
			edgeId := te[1]
			// set edge role since get edge require edge or admin role to by-pass project filtering
			authContext := &base.AuthContext{
				TenantID: tenantId,
				Claims: jwt.MapClaims{
					"tenantId":    tenantId,
					"edgeId":      edgeId,
					"specialRole": "edge",
				},
			}
			newContext := context.WithValue(ctx, base.AuthContextKey, authContext)
			_, err := dbAPI.GetEdgeCluster(newContext, edgeId)
			if err == nil {
				edgeCert, err := dbAPI.GetEdgeCertByEdgeID(newContext, edgeId)
				if err == nil {
					err = verifyEdgeSignatureWithOldAndNewCerts(edgeCert, doc.Email, doc.Password)
					if err == nil {
						claims := jwt.MapClaims{
							"tenantId":    tenantId,
							"edgeId":      edgeId,
							"specialRole": "edge",
							"roles":       []string{},
							"scopes":      []string{},
							"nbf":         time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
							"exp":         time.Now().Unix() + EdgeTokenLifetimeSec,
						}
						// fill in projects
						authContext.Claims = claims
						newContext = context.WithValue(ctx, base.AuthContextKey, authContext)
						token, _ := crypto.SignJWT(claims)
						// make an admin JWT token
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, `{"token":"%s"}`, token)
						return
					} else {
						glog.Warningf(base.PrefixRequestID(ctx, "login: verify signature failed for edge id %s - %s\n"), edgeId, err.Error())
					}
				} else {
					glog.Warningf(base.PrefixRequestID(ctx, "login: get edge cert failed for edge id %s - %s\n"), edgeId, err.Error())
				}
			} else {
				glog.Warningf(base.PrefixRequestID(ctx, "login: get edge failed for edge id %s - %s\n"), edgeId, err.Error())
			}
		} else {
			usr, err := dbAPI.GetUserByEmail(ctx, doc.Email)
			if err == nil {
				// compare hash doc.Password with usr.Password
				if crypto.MatchHashAndPassword(usr.Password, doc.Password) {
					// The system user in tenantpoolservice is <tenantID>@ntnxsherlock.com where <tenantID> is UUID.
					// Tenantpoolservice creates only the tenant without the builtin resources
					if strings.ToLower(doc.Email) == fmt.Sprintf("%s@ntnxsherlock.com", usr.TenantID) {
						err = dbAPI.CreateBuiltinTenantObjects(ctx, usr.TenantID)
						if err != nil {
							if _, ok := err.(*errcode.DatabaseDuplicateError); ok {
								// Ignore duplicate error
								err = nil
							} else {
								glog.Errorf(base.PrefixRequestID(ctx, "Error in creating builtin tenant objects for tenant %s. Error: %s"), usr.TenantID, err.Error())
								authErr := errcode.NewInternalError("Builtin resource creation failure")
								handleResponse(w, r, authErr, "Login failed for %s", doc.Email)
								return
							}
						}
					}
					claims := jwt.MapClaims{}
					err = verifyTrialExpiry(ctx, dbAPI, usr.TenantID, nil, claims)
					if err != nil {
						if _, ok := err.(*errcode.RecordNotFoundError); ok {
							// User exists but the tenant claim entries are removed.
							// This is existing paid customer
							err = nil
						}
					}
					if err == nil {
						token := api.GetUserJWTToken(dbAPI, &usr, nil, DefaultTokenLifetimeSec, api.DefaultTokenType, claims)
						glog.V(4).Infof("[200] login successful\n")
						// make an admin JWT token
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprintf(w, `{"token":"%s", "_id": "%s", "name": "%s", "email": "%s"}`, token, usr.ID, usr.Name, usr.Email)
						return
					}
				}
			} else {
				glog.Errorf(base.PrefixRequestID(ctx, "Login failed for %s - %s"), doc.Email, err.Error())
			}
		}
		// client error, so update login failure count
		loginTracker.UpdateLoginFailureInfo(doc.Email)
		// return 401 error
		authErr := errcode.NewInvalidCredentialsError()
		handleResponse(w, r, authErr, "Login failed for %s", doc.Email)
		return
	})
}

// createTokenWithLifetime - create a token of given lifetime
func createTokenWithLifetime(dbAPI api.ObjectModelAPI, loginTracker *auth.LoginTracker,
	lifetime int64, expectInfo bool, tokenType api.TokenType) httprouter.Handle {
	return getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		inputClaims := jwt.MapClaims{}
		if expectInfo {
			var reader io.Reader = r.Body
			doc := model.LoginTokenPayload{}
			err := base.Decode(&reader, &doc.Info)
			if err == nil {
				if len(doc.Info) > DefaultInClaimsLen {
					handleResponse(w, r, err, "Invalid data. Claims limit exceeded. Login failed")
					return
				}
				for k, v := range doc.Info {
					// Skip key and values that exceed the expected length
					if len(k) > DefaultInClaimKeyLen || len(v) > DefaultInClaimValLen {
						continue
					}
					if _, ok := doc.Info[k]; ok {
						// do not override any existing keys
						continue
					}
					inputClaims[k] = v
				}
			}
		}

		ctx := r.Context()

		// Authenticate this call
		claims, err := auth.VerifyAuthorization(r, dbAPI.GetPublicKeyResolver, dbAPI.GetClaimsVerifier)
		if err != nil {
			handleResponse(w, r, err, "Unauthorized user")
			return
		}

		// Lookup the user
		email := claims["email"].(string)
		glog.V(0).Infof(base.PrefixRequestID(ctx, "create token email %+v"), email)

		usr, err := dbAPI.GetUserByEmail(ctx, email)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Login failed for %s - %s"), email, err.Error())
			handleResponse(w, r, err, "Login failed for %s", email)
			return
		}
		err = verifyTrialExpiry(ctx, dbAPI, usr.TenantID, nil, inputClaims)
		if err != nil {
			if _, ok := err.(*errcode.RecordNotFoundError); ok {
				// User exists but the tenant claim entries are removed.
				// This is existing paid customer
				err = nil
			} else {
				glog.Errorf(base.PrefixRequestID(ctx, "Login failed for %s - %s"), email, err.Error())
				handleResponse(w, r, err, "Login failed for %s", email)
				return
			}
		}
		inTokenType, ok := claims["type"].(string)
		// Create a regular token
		var token string
		if tokenType == api.ShortTokenType {
			if ok && inTokenType != string(api.DefaultTokenType) {
				// Only the default token type which may be present or absent can request short token
				glog.Errorf(base.PrefixRequestID(ctx, "Login failed for %s - default token expected"), email)
				handleResponse(w, r, errcode.NewPermissionDeniedError("No permission"), "Login failed for %s", email)
				return
			}
			token = GetShortUserJWTToken(dbAPI, &usr, nil, lifetime, inputClaims)
		} else {
			if tokenType == api.LongTokenType {
				if !ok || inTokenType != string(api.ShortTokenType) {
					// Only the short token can request long token
					glog.Errorf(base.PrefixRequestID(ctx, "Login failed for %s - short token expected"), email)
					handleResponse(w, r, errcode.NewPermissionDeniedError("No permission"), "Login failed for %s", email)
					return
				}
			}
			token = api.GetUserJWTToken(dbAPI, &usr, nil, lifetime, tokenType, inputClaims)
		}
		glog.Infof(base.PrefixRequestID(ctx, "[200] login success. email %+v"), email)
		// make an admin JWT token
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"token":"%s", "_id": "%s", "name": "%s", "email": "%s"}`, token, usr.ID, usr.Name, usr.Email)
		return
	})
}

// CreateShortLoginToken - create token with a short lifetime
func CreateShortLoginToken(dbAPI api.ObjectModelAPI, loginTracker *auth.LoginTracker) httprouter.Handle {
	return createTokenWithLifetime(dbAPI, loginTracker, DefaultShortLoginTokenLifetimeSec, false, api.ShortTokenType)
}

// CreateLoginToken - create a regular login token
func CreateLoginToken(dbAPI api.ObjectModelAPI, loginTracker *auth.LoginTracker) httprouter.Handle {
	return createTokenWithLifetime(dbAPI, loginTracker, DefaultLongTokenLifeTimeSec, true, api.LongTokenType)
}

// CreateOAuthLoginHandler redirects the client to the oauth authorization endpoint.
// UI redirect handler (https://<host>/auth/oath) receives
// state=<JWT state>code=<auth code>session_state=<session_state> as query params
// The JWT state is the claims in the function below. It has the original returnUrl.
// UI needs to compare this state value with the one in the cookie to verify
// that it has not been compromised. The UI handler has to send back the <auth code>
// to the token handler endpoint to get the final Sherlock JWT session token.
func CreateOAuthLoginHandler(dbAPI api.ObjectModelAPI) httprouter.Handle {
	return getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		queryParams := r.URL.Query()
		returnURL := queryParams.Get("returnUrl")
		claims := jwt.MapClaims{
			"returnUrl": returnURL,
			"nbf":       time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
			"exp":       time.Now().Unix() + 60, // 1 min
		}
		state, _ := crypto.SignJWT(claims)
		cookie := http.Cookie{Name: "oauth-state", Value: state, Path: "/", Secure: true}
		http.SetCookie(w, &cookie)
		oAuthHandler, err := LookupOAuthHandler(dbAPI, r)
		if err != nil {
			handleResponse(w, r, err, "Unauthorized")
			return
		}
		authURL := oAuthHandler.GetAuthRedirectURL(state)
		http.Redirect(w, r, authURL, http.StatusFound)
	})
}

// CreateOAuthTokenHandler returns the sherlock JWT token from the auth code (code=<auth code>)
func CreateOAuthTokenHandler(dbAPI api.ObjectModelAPI) httprouter.Handle {
	return getContext(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		var reader io.Reader = r.Body
		ctx := r.Context()
		doc := model.OAuthCodes{}
		err := base.Decode(&reader, &doc)
		if err != nil {
			handleResponse(w, r, err, "Login failed")
			return
		}
		oAuthHandler, err := LookupOAuthHandler(dbAPI, r)
		if err != nil {
			handleResponse(w, r, err, "Unauthorized")
			return
		}
		oAuthToken, err := oAuthHandler.GetOAuthToken(ctx, &doc)
		if err != nil {
			handleResponse(w, r, err, "Unauthorized")
			return
		}
		var user *model.User
		var claims jwt.MapClaims
		if len(doc.Code) == 0 {
			// Refresh is called
			// ID token does not contain xi_iot because of a known issue.
			// Do not create user in this case, we do not have enough information.
			user, claims, err = GetOrUpdateUser(ctx, dbAPI, oAuthToken)
		} else {
			federatedIDP, ok := oAuthToken.IDToken["federated_idp"].(string)
			if !ok {
				// Always supposed to be present
				federatedIDP = "local"
			}
			// Login with auth code is called
			if *config.Cfg.EnableXiIoTRole {
				var xiIOTRole *auth.XIIOTRole
				xiIOTRole, err = auth.GetXIIOTRole(ctx, oAuthToken)
				if err == nil {
					user, claims, err = GetOrCreateUser(ctx, dbAPI, oAuthToken, xiIOTRole)
				} else if federatedIDP == "local" {
					// Role assignment can be done only when the IDP is my-nutanix.
					// It does not apply for external AD server like sts.compass-group.fr
					landingURL := dbAPI.GetServiceLandingURL(ctx, r)
					glog.V(4).Info(base.PrefixRequestID(ctx, "Landing URL: %s"), landingURL)
					assignRoleURL := oAuthHandler.GetAssignXIIOTRoleURLWithLandingURL(landingURL)
					glog.Warningf(base.PrefixRequestID(ctx, "No Xi IoT role found in %+v. Redirecting to %s"), oAuthToken, assignRoleURL)
					w.WriteHeader(http.StatusOK)
					fmt.Fprintf(w, `{"statusCode": %d, "location": "%s"}`, http.StatusFound, assignRoleURL)
					return
				}

			} else {
				user, claims, err = GetOrUpdateUser(ctx, dbAPI, oAuthToken)
			}
		}
		if err != nil {
			handleResponse(w, r, err, "Unauthorized")
			return
		}
		token := api.GetUserJWTToken(dbAPI, user, oAuthToken, DefaultTokenLifetimeSec, api.DefaultTokenType, claims)
		glog.Infof("[200] login successful\n")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"token":"%s", "_id": "%s", "name": "%s", "email": "%s"}`, token, user.ID, user.Name, user.Email)
	})
}

// GetShortUserJWTToken generates a token with the minimal data.
// QR code has to render the short token so it must be very short to be within the pixel limits
func GetShortUserJWTToken(dbAPI api.ObjectModelAPI, user *model.User, authToken *auth.OAuthToken,
	lifetime int64, inClaims jwt.MapClaims) string {
	claims := jwt.MapClaims{
		"email": user.Email,
		"nbf":   time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
		"exp":   time.Now().Unix() + lifetime,
		"type":  api.ShortTokenType,
	}
	if expiry, ok := inClaims["trialExpiry"]; ok {
		claims["trialExpiry"] = expiry
	}
	token, _ := crypto.SignJWT(claims)
	return token
}

// GetOrUpdateUser checks the user from access token in the DB and updates it if neccessary.
// This is unused because xi-iot-role is already enabled in prod
func GetOrUpdateUser(ctx context.Context, dbAPI api.ObjectModelAPI, oAuthToken *auth.OAuthToken) (*model.User, jwt.MapClaims, error) {
	idToken := oAuthToken.IDToken
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Got id_token %+v"), idToken)
	email := idToken["email"].(string)
	givenName := idToken["given_name"].(string)
	lastName := idToken["last_name"].(string)
	fullName := fmt.Sprintf("%s, %s", lastName, givenName)

	authContext := &base.AuthContext{
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}

	ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
	dbUser, err := dbAPI.GetUserByEmail(ctx, email)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in get user by email %s. Error: %s"), email, err.Error())
		return nil, nil, err
	}
	if dbUser.Name != fullName {
		glog.Infof(base.PrefixRequestID(ctx, "Updating user because name has changed"))
		dbUser.Name = fullName
		// Setting the tenant id so we can validate the context during update
		authContext.TenantID = base.MachineTenantID
		ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
		// Update user name
		_, err = dbAPI.UpdateUser(ctx, &dbUser, nil)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in updating user with email %s. Error: %s"), dbUser.Email, err.Error())
			return nil, nil, err
		}
	}
	claims := jwt.MapClaims{}
	err = verifyTrialExpiry(ctx, dbAPI, dbUser.TenantID, nil, claims)
	if err != nil {
		if _, ok := err.(*errcode.RecordNotFoundError); ok {
			// User exists but the tenant claim entries are removed.
			// This is existing paid customer
			err = nil
		}
	}
	return &dbUser, claims, err
}

// GetOrCreateUser creates user and tenant from idToken if missing and the user is entitled to XI IOT
// It handles getting or creating the IoT role from the id_token in oAuthToken, assigning tenantClaim.
func GetOrCreateUser(ctx context.Context, dbAPI api.ObjectModelAPI, oAuthToken *auth.OAuthToken, xiIOTRole *auth.XIIOTRole) (*model.User, jwt.MapClaims, error) {
	if oAuthToken == nil || xiIOTRole == nil {
		glog.Errorf(base.PrefixRequestID(ctx, "No Xi IoT role found in %+v"), oAuthToken)
		return nil, nil, errcode.NewPermissionDeniedError("No permission")
	}
	idToken := oAuthToken.IDToken
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Got id_token %+v"), idToken)
	claims := jwt.MapClaims{}
	// ID token has these claims
	email := idToken["email"].(string)
	givenName := idToken["given_name"].(string)
	lastName := idToken["last_name"].(string)
	fullName := fmt.Sprintf("%s, %s", lastName, givenName)

	tenant := &model.Tenant{
		ExternalID: xiIOTRole.TenantDomain,
		Name:       xiIOTRole.TenantName,
	}

	user := &model.User{
		Email:    email,
		Name:     fullName,
		Password: base.GenerateStrongPassword(),
		Role:     "USER",
	}

	if xiIOTRole.Role == auth.XIAdminRole {
		user.Role = "INFRA_ADMIN"
	}

	authContext := &base.AuthContext{
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}

	ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
	dbUser, err := dbAPI.GetUserByEmail(ctx, email)
	if err == nil {
		err = handleExistingUser(ctx, dbAPI, tenant, user, &dbUser, claims)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to handle existing user %s with external ID %s. Error: %s"), user.Email, tenant.ExternalID, err.Error())
			return nil, nil, err
		}
		return &dbUser, claims, nil
	}
	if _, ok := err.(*errcode.RecordNotFoundError); ok {
		dbUser, err := handleNewUser(ctx, dbAPI, tenant, user, claims)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to handle new user %s with external ID %s. Error: %s"), user.Email, tenant.ExternalID, err.Error())
			return nil, nil, err
		}
		return dbUser, claims, nil
	}
	glog.Errorf(base.PrefixRequestID(ctx, "Error in get user by email %s. Error: %s"), user.Email, err.Error())
	return nil, nil, err
}

// GetTenantClaimRegistrationCode returns the registration code under which the tenant claim will be created/assigned for the user.
// It returns empty string if the email belongs to a Nutanix user and it (its tenant) already exists in IoT DB
func GetTenantClaimRegistrationCode(ctx context.Context, email string, isExisting bool) string {
	var registrationCode string
	if isNutanixEmail(email) {
		if !isExisting {
			// Set only for a non-existing Nutanix email
			registrationCode = NutanixRegistrationID
		}
	} else if isExisting {
		// Non Nutanix existing email
		registrationCode = NonNutanixNonTrialRegistrationID
	} else {
		// Non Nutanix non-existing email
		registrationCode = NonNutanixRegistrationID
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Got registration code %s for user %s"), registrationCode, email)
	return registrationCode
}

// handleExitingUser handles users which are already in account service.
// It also populates the dbUser user with the updated information
func handleExistingUser(ctx context.Context, dbAPI api.ObjectModelAPI, tenant *model.Tenant, currentUser *model.User, dbUser *model.User, outClaims jwt.MapClaims) error {
	dbTenant, err := dbAPI.GetTenant(ctx, tenant.ExternalID)
	if err == nil {
		if dbTenant.ID != dbUser.TenantID {
			// Tenant has changed.
			// User is supposed to belong to the tenant with his external ID.
			// User cannot be updated for tenant change.
			// Better to inform rather than silently moving the tenant association and surprise the user
			glog.Errorf(base.PrefixRequestID(ctx, "Tenant change detected for email %s. Please delete the existing user manually to associate to external ID %s"), currentUser.Email, tenant.ExternalID)
			return errcode.NewPermissionDeniedError("No permission to update tenant")
		}
		err = updateUser(ctx, dbAPI, currentUser, dbUser)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update user %s with external ID %s. Error: %s"), currentUser.Email, tenant.ExternalID, err.Error())
			return err
		}
		if *config.Cfg.EnableTrial {
			err = createTenantClaimIfMissing(ctx, dbAPI, dbTenant.ID, dbUser.Email, outClaims)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get or create tenant claim %s with external ID %s for user %s. Error: %s"), dbTenant.ID, tenant.ExternalID, currentUser.Email, err.Error())
				return err
			}
		}
		return nil
	}
	if _, ok := err.(*errcode.RecordNotFoundError); ok {
		// External ID is not present. Continue to update the tenant for migration
		// This mapping can only be performed by an INFRA ADMIN user
		if currentUser.Role != "INFRA_ADMIN" {
			glog.Errorf(base.PrefixRequestID(ctx, "User %s does not have permission to update tenant"), currentUser.Email)
			return errcode.NewPermissionDeniedError("No permission to create/update tenant")
		}
		// Tenant must always exist because the user depends on the tenant
		dbTenant, err = dbAPI.GetTenant(ctx, dbUser.TenantID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenant with ID %s for email %s. Error: %s"), dbUser.TenantID, dbUser.Email, err.Error())
			return err
		}
		if len(dbTenant.ExternalID) == 0 {
			// There is no external ID association yet that means no other my-nutanix user association
			// This happens when the user logs in for the first time from my-nutanix
			glog.V(3).Infof(base.PrefixRequestID(ctx, "External ID for user %s and tenant %s is not set. Setting it to %s..."), currentUser.Email, dbTenant.ID, tenant.ExternalID)
			dbTenant.ExternalID = tenant.ExternalID
			_, err = dbAPI.UpdateTenant(ctx, &dbTenant, nil)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenant with ID %s for email %s. Error: %s"), dbUser.TenantID, dbUser.Email, err.Error())
				return err
			}
		} else if dbTenant.ExternalID != tenant.ExternalID {
			// Tenant has changed.
			// User is supposed to belong to the tenant with his external ID.
			// User cannot be updated for tenant change.
			// Better to inform rather than silently moving the tenant association and surprise the user
			glog.Errorf(base.PrefixRequestID(ctx, "Tenant change detected for email %s. Please delete the existing user manually to associate to external ID %s"), currentUser.Email, tenant.ExternalID)
			return errcode.NewPermissionDeniedError("No permission to update tenant")
		}
		err = updateUser(ctx, dbAPI, currentUser, dbUser)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update user %s with external ID %s. Error: %s"), currentUser.Email, tenant.ExternalID, err.Error())
			return err
		}
		if *config.Cfg.EnableTrial {
			err = createTenantClaimIfMissing(ctx, dbAPI, dbTenant.ID, dbUser.Email, outClaims)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get or create tenant claim %s with external ID %s for user %s. Error: %s"), dbTenant.ID, tenant.ExternalID, currentUser.Email, err.Error())
				return err
			}
		}
		return nil
	}
	glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenant with external ID %s. Error: %s"), tenant.ExternalID, err.Error())
	return err
}

// handleNewUser handles new user which may or may not have tenant mapping
func handleNewUser(ctx context.Context, dbAPI api.ObjectModelAPI, tenant *model.Tenant, user *model.User, outClaims jwt.MapClaims) (*model.User, error) {
	// Check if there is an existing tenant mapping
	dbTenant, err := dbAPI.GetTenant(ctx, tenant.ExternalID)
	if err == nil {
		if *config.Cfg.EnableTrial {
			err = createTenantClaimIfMissing(ctx, dbAPI, dbTenant.ID, user.Email, outClaims)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Failed to get or create tenant claim %s with external ID %s for user %s. Error: %s"), dbTenant.ID, tenant.ExternalID, user.Email, err.Error())
				return nil, err
			}
		}
	} else {
		if _, ok := err.(*errcode.RecordNotFoundError); !ok {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in getting tenant with external ID %s. Error: %s"), tenant.ExternalID, err.Error())
			return nil, err
		}
		if user.Role != "INFRA_ADMIN" {
			glog.Errorf(base.PrefixRequestID(ctx, "Only user with INFRA_ADMIN role can create/update tenant with external ID %s. Error: %s"), tenant.ExternalID, err.Error())
			return nil, err
		}
		dbTenant, err = acquireTenant(ctx, dbAPI, tenant, user.Email, outClaims)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to acquire tenant for email %s with external ID %s. Error: %s"), user.Email, tenant.ExternalID, err.Error())
			return nil, err
		}
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Creating builtin tenant objects for tenant %s"), dbTenant.ID)
	// Create builtin tenant objects
	err = dbAPI.CreateBuiltinTenantObjects(ctx, dbTenant.ID)
	if err != nil {
		if _, ok := err.(*errcode.DatabaseDuplicateError); !ok {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating builtin tenant objects for tenant %s. Error: %s"), dbTenant.ID, err.Error())
			return nil, err
		}
	}
	// Update the tenant ID
	user.TenantID = dbTenant.ID
	authContext, _ := base.GetAuthContext(ctx)
	authContext.TenantID = dbTenant.ID
	ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Creating user %s"), user.Email)
	_, err = dbAPI.CreateUser(ctx, user, nil)
	if err != nil {
		if _, ok := err.(*errcode.DatabaseDuplicateError); !ok {
			// Some other error
			glog.Errorf(base.PrefixRequestID(ctx, "Error in creating user %+v. Error: %s"), user, err.Error())
			return nil, err
		}
	}
	dbUser, err := dbAPI.GetUserByEmail(ctx, user.Email)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in get user by email %s. Error: %s"), user.Email, err.Error())
		return nil, err
	}
	return &dbUser, nil
}

// acquireTenant creates or acquires (tenant pool)
func acquireTenant(ctx context.Context, dbAPI api.ObjectModelAPI, tenant *model.Tenant, email string, outClaims jwt.MapClaims) (model.Tenant, error) {
	// Tenant with the external ID does not exist
	if *config.Cfg.EnableTrial {
		glog.V(3).Infof(base.PrefixRequestID(ctx, "Assigning tenant %+v"), tenant)
		dbTenant, err := assignTenant(ctx, dbAPI, tenant, email, outClaims)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to assign tenant to external ID %s. Error: %s"), tenant.ExternalID, err.Error())
			return dbTenant, err
		}
		return dbTenant, nil
	} else if !*config.Cfg.EnableTenantCreation {
		err := errcode.NewPermissionDeniedError("Unknown tenant")
		glog.Errorf(base.PrefixRequestID(ctx, "Unknown tenant %+v"), tenant)
		return model.Tenant{}, err
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Creating tenant %+v"), tenant)
	_, err := dbAPI.CreateTenant(ctx, tenant, nil)
	if err != nil {
		// Ignore update
		if _, ok := err.(*errcode.DatabaseDuplicateError); !ok {
			// Some other error
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenant %+v. Error: %s"), tenant, err.Error())
			return model.Tenant{}, err
		}
	}
	// By this line, tenant is already created
	dbTenant, err := dbAPI.GetTenant(ctx, tenant.ExternalID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenant with external ID %s. Error: %s"), tenant.ExternalID, err.Error())
		return dbTenant, err
	}
	return dbTenant, nil
}

// updateUser updates an existing user if some interested fields change
func updateUser(ctx context.Context, dbAPI api.ObjectModelAPI, currentUser *model.User, dbUser *model.User) error {
	// Update the existing user if there is any change requiring DB update
	if dbUser.Name != currentUser.Name || dbUser.Role != currentUser.Role {
		glog.Infof(base.PrefixRequestID(ctx, "Updating user because name or role or both have changed"))
		dbUser.Name = currentUser.Name
		dbUser.Role = currentUser.Role
		authContext, _ := base.GetAuthContext(ctx)
		// Setting the tenant id so we can validate the context during update
		authContext.TenantID = base.MachineTenantID
		ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
		// Update user name
		_, err := dbAPI.UpdateUser(ctx, dbUser, nil)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to update user with email %s. Error: %s"), dbUser.Email, err.Error())
			return err
		}
	}
	return nil
}

// assignTenant assigns a tenant and maps the externalID to it such that any my-nutanix user with
// the externalID belongs to this assigned tenant
func assignTenant(ctx context.Context, dbAPI api.ObjectModelAPI, tenant *model.Tenant, email string, outClaims jwt.MapClaims) (model.Tenant, error) {
	// Create the cloud instance only on the first external ID association (login first time via my-nutanix)
	// For concurrent users with the same external ID, there can be multiple reservations.
	// We solve this by reducing the reservation time
	var dbTenant model.Tenant
	registrationCode := GetTenantClaimRegistrationCode(ctx, email, false)
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Reserving tenantClaim for external ID %s"), tenant.ExternalID)
	tenantID, err := dbAPI.ReserveTenantClaim(ctx, registrationCode, email)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to reserve a tenantClaim. Error: %s"), err.Error())
		return dbTenant, err
	}
	// By this line, tenant is already created by the tenantpool service
	dbTenant, err = dbAPI.GetTenant(ctx, tenantID)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenant with external ID %s. Error: %s"), tenant.ExternalID, err.Error())
		return dbTenant, err
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Assigned tenant %s for external ID %s"), dbTenant.ID, tenant.ExternalID)
	// Only external ID can be updated.
	// Name is still trial as this is in trial flow
	dbTenant.ExternalID = tenant.ExternalID
	_, err = dbAPI.UpdateTenant(ctx, &dbTenant, nil)
	if err != nil {
		// Report duplicate too as the tenant ID is stale
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to update tenant %s for external ID %s. Error: %s"), dbTenant.ID, tenant.ExternalID, err.Error())
		return dbTenant, err
	}
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Confirming tenant %s for external ID %s"), dbTenant.ID, tenant.ExternalID)
	tenantClaim, err := dbAPI.ConfirmTenantClaim(ctx, registrationCode, tenantID, email)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Failed to confirm tenant %s for external ID %s. Error: %s"), dbTenant.ID, tenant.ExternalID, err.Error())
		// Do not try to delete the external ID assignment
		// There can be concurrent calls
		return dbTenant, err
	}
	err = verifyTrialExpiry(ctx, dbAPI, tenantID, tenantClaim, outClaims)
	return dbTenant, err
}

// createTenantClaimIfMissing creates a tenantClaim with the tenantID
func createTenantClaimIfMissing(ctx context.Context, dbAPI api.ObjectModelAPI, tenantID string, email string, outClaims jwt.MapClaims) error {
	glog.V(3).Infof(base.PrefixRequestID(ctx, "Checking for existing tenantClaim for tenant %s"), tenantID)
	tenantClaim, err := dbAPI.GetTenantClaim(ctx, tenantID)
	if err != nil {
		if _, ok := err.(*errcode.RecordNotFoundError); !ok {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenantClaim %s. Error: %s"), tenantID, err.Error())
			return err
		}
		registrationID := GetTenantClaimRegistrationCode(ctx, email, true)
		if len(registrationID) == 0 {
			glog.V(3).Infof(base.PrefixRequestID(ctx, "Tenant claim creation is disabled for Nutanix users. Skippping for user %s"), email)
		} else {
			// For trial users, CreateTenantClaim is not supposed to happen as there is always an edge entry in the DB.
			tenantClaim, err = dbAPI.CreateTenantClaim(ctx, registrationID, tenantID, email)
			if err != nil {
				if _, ok := err.(*errcode.DatabaseDuplicateError); !ok {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to create tenantClaim %s. Error: %s"), tenantID, err.Error())
					// Ignore error depending on the reliability of Bott
					return nil
				}
				tenantClaim, err = dbAPI.GetTenantClaim(ctx, tenantID)
				if err != nil {
					glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenantClaim %s. Error: %s"), tenantID, err.Error())
					return err
				}
			}
		}
	}
	// For non-trial, it simply returns. So, it does not block
	return verifyTrialExpiry(ctx, dbAPI, tenantID, tenantClaim, outClaims)
}

// verifyTrialExpiry verifies if the trial is still valid and adds the expiry if valid
func verifyTrialExpiry(ctx context.Context, dbAPI api.ObjectModelAPI, tenantID string, tenantClaim *tenantpool.TenantClaim, outClaims jwt.MapClaims) error {
	if !*config.Cfg.EnableTrial {
		return nil
	}
	if tenantClaim == nil {
		var err error
		// Read if the tenantClaim is not set
		tenantClaim, err = dbAPI.GetTenantClaim(ctx, tenantID)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Failed to get tenantClaim %s. Error: %s"), tenantID, err.Error())
			return err
		}
	}
	// Only check for trial users
	if !tenantClaim.Trial {
		return nil
	}
	// The tenant is no longer usable
	if tenantClaim.State == tenantpoolcore.Deleting {
		glog.Warningf(base.PrefixRequestID(ctx, "TenantClaim %+v is getting deleted"), tenantClaim)
		return errcode.NewPermissionDeniedError("Trial expired")
	}

	if tenantClaim.ExpiresAt != nil && !tenantClaim.ExpiresAt.IsZero() {
		expiresAt := *tenantClaim.ExpiresAt
		// Consistent behavior is required for login and API calls.
		// If the API calls fail due to trial having expired, the login must fail too
		if time.Since(expiresAt) > 0 {
			glog.Warningf(base.PrefixRequestID(ctx, "Trial expired for %+v"), tenantClaim)
			return errcode.NewPermissionDeniedError("Trial expired")
		}
		// Send the absolute time because UI does not know the starting time
		outClaims["trialExpiry"] = expiresAt
	}
	return nil
}

// isNutanixEmail returns true if the email ends with @nutanix.com
func isNutanixEmail(email string) bool {
	return strings.HasSuffix(strings.ToLower(email), "@nutanix.com")
}

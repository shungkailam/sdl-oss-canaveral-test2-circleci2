package router_test

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/apitesthelper"
	"cloudservices/cloudmgmt/config"
	"cloudservices/cloudmgmt/router"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	scrypto "cloudservices/common/crypto"
	"cloudservices/common/model"
	"cloudservices/tenantpool/core"
	tenantpoolmodel "cloudservices/tenantpool/model"
	"cloudservices/tenantpool/testhelper"
	"context"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/oauth2"
)

const (
	EdgeHandlePath     = "/v1/edgehandle"
	nutanixEmail       = "nsing@nutanix.com"
	nutanixPassword    = "Khogen@123"
	nonNutanixEmail    = "sherlock1@pokemail.net"
	nonNutanixPassword = "Khogen@123"
	// Do not change
	nonNutanixExternalID = "41990ddb-fe9f-4e7d-9db6-af4717b733d3"
	// Disable or enable trial test
	// Trial login test has issues with concurrent builds
	disableTrialLoginTest = true
)

// get edge certificates
func getEdgeCertificates(netClient *http.Client, token string, tenantID string, edgeID string) (model.EdgeCert, error) {
	var edgeCerts model.EdgeCert
	path := fmt.Sprintf("%s/%s", EdgeHandlePath, edgeID)
	fmt.Println("******* PATH *******")
	fmt.Println(path)
	fmt.Println("******* PATH *******")
	password, err := scrypto.EncryptPassword(edgeID)
	if err == nil {
		params := model.GetHandlePayload{
			TenantID: tenantID,
			Token:    password,
		}
		_, err = doPost(netClient, path, token, &params, &edgeCerts)
	}
	return edgeCerts, err
}

func getEdgeEmailPassword(t *testing.T, tenantID string, edgeID string, key string) (email string, password string) {
	email, password, err := scrypto.GetEdgeEmailPassword(tenantID, edgeID, key)
	require.NoError(t, err)
	return
}

func TestLogin(t *testing.T) {
	t.Parallel()
	t.Log("Running TestEdgeLogin test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}
	*config.Cfg.EnableTrial = false
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteBuiltinTenantObjects(ctx, tenantID)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Edge", func(t *testing.T) {
		// login as user to get token
		token := loginUser(t, netClient, user)
		// create edge
		edge, _, err := createEdgeForTenant(netClient, tenantID, token)
		require.NoError(t, err)
		edgeID := edge.ID
		t.Logf("edge created: %+v", edge)

		// get edge certs
		edgeCerts, err := getEdgeCertificates(netClient, token, tenantID, edgeID)
		require.NoError(t, err)
		t.Logf("Got edge cert: %+v", edgeCerts)
		// Edge login based on per-tenant root CA certs
		email, password := getEdgeEmailPassword(t, tenantID, edgeID, edgeCerts.EdgePrivateKey)
		resp, err := login(netClient, email, password)
		require.NoErrorf(t, err, "Login for edge using per-tenant root CA failed: %+v", resp)
		t.Logf("Login using per-tenant root CA successful")
		/*
			// Edge login based on fixed root CA certs
			email, password = getEdgeEmailPassword(t, tenantID, edgeID, edgeCerts.PrivateKey)
			resp, err = login(netClient, email, password)
			if err != nil {
				t.Logf("Response of edge login call: %+v", resp)
				t.Fatalf("Login for edge using fixed root CA failed: %s", err)
			}
			t.Logf("Login using fixed root CA successful")
		*/
		_, _, err = deleteEdge(netClient, edgeID, token)
		require.NoError(t, err)
	})
	/*
		t.Run("Test OAuth2", func(t *testing.T) {
			authContext := &base.AuthContext{
				Claims: jwt.MapClaims{
					"specialRole": "admin",
				},
			}
			ctx := context.WithValue(context.Background(), oauth2.HTTPClient, apitesthelper.GetHTTPClient())
			oAuth2Config := apitesthelper.OAuth2Config
			oAuth2Handler := auth.NewOAuthHandler(oAuth2Config.ClientID, oAuth2Config.ClientSecret, apitesthelper.IDP_BASE_URL, oAuth2Config.RedirectURL)

			url := oAuth2Handler.GetAssignXIIOTRoleURL()
			if url != "https://demo-my.nutanix.com/api/v1/auth/iot" {
				t.Fatalf("Expected https://demo-my.nutanix.com/api/v1/auth/iot, found %s", url)
			}
			code, err := apitesthelper.GetOAuthCode(t, nutanixEmail, nutanixPassword)
			if err != nil {
				t.Fatalf("Error getting OAuth2 code. Error :%s", err.Error())
			}
			doc := model.OAuthCodes{Code: code}
			t.Logf("Getting token")
			tokens, err := oAuth2Handler.GetOAuthToken(ctx, &doc)
			if err != nil {
				t.Fatalf("Error getting token. Error: %s", err.Error())
			}
			federatedIDP := tokens.IDToken["federated_idp"].(string)
			if federatedIDP != "local" {
				t.Fatalf("federated_idp must be local")
			}
			t.Logf("Getting XI IOT role")
			iotRole, err := auth.GetXIIOTRole(ctx, tokens)
			if err != nil {
				t.Fatalf("Get XI IOT role failed. Error: %s", err.Error())
			}
			t.Logf("Found XI IOT role %+v", iotRole)
			// Setup the user
			// Get the test user
			existingUser, err := dbAPI.GetUserByEmail(ctx, nutanixEmail)
			if err == nil {
				authContext.TenantID = existingUser.TenantID
				ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
				_, err = dbAPI.DeleteUser(ctx, existingUser.ID, nil)
				if err != nil {
					t.Fatalf("Error deleting existing user %s. Error: %s", user.ID, err.Error())
				}
			}
			// This must properly create the user
			user, _, err := router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
			if err != nil {
				t.Fatalf("Failed in GetOrCreateUser. Error: %s", err.Error())
			}
			existingUser = *user
			// Get the tenant for the test user
			existingTenant, err := dbAPI.GetTenant(ctx, existingUser.TenantID)
			if err != nil {
				t.Fatalf("Error getting tenant. Error: %s", err.Error())
			}
			authContext.TenantID = existingUser.TenantID
			ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
			if existingTenant.ExternalID != iotRole.TenantDomain {
				t.Fatalf("Nil external ID must be set")
			}

			// Remove the external ID if present
			if len(existingTenant.ExternalID) > 0 {
				existingTenant.ExternalID = ""
				_, err = dbAPI.UpdateTenant(ctx, &existingTenant, nil)
				if err != nil {
					t.Fatalf("Error updating tenant. Error: %s", err.Error())
				}
			}
			t.Logf("Calling with an existing external ID mapping for the user")
			// This mus fill up the external ID for the same tenant
			user, _, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
			if err != nil {
				t.Fatalf("Failed in GetOrCreateUser. Error: %s", err.Error())
			}
			if existingUser.TenantID != user.TenantID {
				t.Fatalf("Unexpected same tenant mapping, existing uid: %s, tenant id: %s, user tenant id: %s", existingUser.ID, existingUser.TenantID, user.TenantID)
			}
			// Get the updated tenant
			existingTenant, err = dbAPI.GetTenant(ctx, user.TenantID)
			if err != nil {
				t.Fatalf("Error getting tenant. Error: %s", err.Error())
			}
			if existingTenant.ExternalID != iotRole.TenantDomain {
				t.Fatalf("Nil external ID must be set")
			}
			// Remove the existing tenant mapping just created above
			existingTenant.ExternalID = ""
			_, err = dbAPI.UpdateTenant(ctx, &existingTenant, nil)
			if err != nil {
				t.Fatalf("Error updating tenant. Error: %s", err.Error())
			}
			_, err = dbAPI.DeleteUser(ctx, user.ID, nil)
			if err != nil {
				t.Fatalf("Error deleting existing user %s. Error: %s", user.ID, err.Error())
			}
			t.Logf("Calling with no mapping at all")
			// The user has no relationship with any tenant now
			user, _, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
			if err != nil {
				t.Fatalf("Failed in GetOrCreateUser. Error: %s", err.Error())
			}
			t.Logf("User %+v", user)
			if existingTenant.ID == user.TenantID {
				t.Fatalf("Existing tenant is picked up. New tenant must be created as there is no mapping")
			}
			// Update the context tenant to latest
			authContext.TenantID = user.TenantID
			ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
			existingTenant, err = dbAPI.GetTenant(ctx, user.TenantID)
			if err != nil {
				t.Fatalf("Error getting tenant. Error: %s", err.Error())
			}
			// Change external ID for the tenant
			existingTenant.ExternalID = "test-" + base.GetUUID()
			_, err = dbAPI.UpdateTenant(ctx, &existingTenant, nil)
			if err != nil {
				t.Fatalf("Error updating tenant. Error: %s", err.Error())
			}
			// The user has no relationship with any tenant now
			_, _, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
			require.Errorf(t, err, "Tenant ID cannot be changed automatically. It must fail")
			// Restore the external ID
			existingTenant.ExternalID = iotRole.TenantDomain
			_, err = dbAPI.UpdateTenant(ctx, &existingTenant, nil)
			if err != nil {
				t.Fatalf("Error updating tenant. Error: %s", err.Error())
			}
			// Get the test user
			finalUser, err := dbAPI.GetUserByEmail(ctx, nutanixEmail)
			require.NoError(t, err)
			t.Logf("Verifying for builtin tenant objects")
			for _, builtinCategory := range config.BuiltinCategories {
				t.Logf("Builtin category %+v", builtinCategory)
				id := api.GetBuiltinCategoryID(finalUser.TenantID, builtinCategory.ID)
				_, err = dbAPI.GetCategory(ctx, id)
				if err != nil {
					t.Fatalf("Category %s is not found. Error: %s", id, err.Error())
				}
			}
			for _, builtinScriptRuntime := range config.BuiltinScriptRuntimes {
				t.Logf("Builtin script runtime %+v", builtinScriptRuntime)
				id := api.GetBuiltinScriptRuntimeID(finalUser.TenantID, builtinScriptRuntime.ID)
				_, err = dbAPI.GetScriptRuntime(ctx, id)
				if err != nil {
					t.Fatalf("ScriptRuntime %s is not found. Error: %s", id, err.Error())
				}
			}
			for _, builtinProject := range config.BuiltinProjects {
				t.Logf("Builtin project %+v", builtinProject)
				id := api.GetDefaultProjectID(finalUser.TenantID)
				_, err = dbAPI.GetProject(ctx, id)
				if err != nil {
					t.Fatalf("Project %s is not found. Error: %s", id, err.Error())
				}
			}
			t.Logf("User %+v", user)
			user, _, err = router.GetOrUpdateUser(ctx, dbAPI, tokens)
			if err != nil {
				t.Fatalf("Failed in GetOrCreateUser. Error: %s", err.Error())
			}
			t.Logf("User %+v", user)
			existingUser, err = dbAPI.GetUserByEmail(ctx, "nsing@nutanix.com")
			if err != nil {
				t.Fatalf("Error getting user by email. Error: %s", err.Error())
			}
			// Update to different user name
			existingUser.Name = "NA"
			authContext.TenantID = existingUser.TenantID
			_, err = dbAPI.UpdateUser(ctx, &existingUser, nil)
			if err != nil {
				t.Fatalf("Error updating user. Error: %s", err.Error())
			}
			t.Logf("Calling GetOrCreateUser with updated user")
			user, _, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
			if err != nil {
				t.Fatalf("Failed in GetOrCreateUser. Error: %s", err.Error())
			}
			t.Logf("User %+v", user)
			existingUser, err = dbAPI.GetUserByEmail(ctx, nutanixEmail)
			if err != nil {
				t.Fatalf("Failed in GetUserByEmail. Error: %s", err.Error())
			}
			if existingUser.Name == "NA" {
				t.Fatalf("Name update failed for GetOrCreateUser")
			}
			// Update to different user name
			existingUser.Name = "NA"
			_, err = dbAPI.UpdateUser(ctx, &existingUser, nil)
			if err != nil {
				t.Fatalf("Error updating user. Error: %s", err.Error())
			}
			// Test the deprecated API GetOrUpdateUser
			t.Logf("Calling GetOrUpdateUser with updated user")
			user, _, err = router.GetOrUpdateUser(ctx, dbAPI, tokens)
			if err != nil {
				t.Fatalf("Failed in GetOrUpdateUser. Error: %s", err.Error())
			}
			existingUser, err = dbAPI.GetUserByEmail(ctx, nutanixEmail)
			if err != nil {
				t.Fatalf("Failed in GetUserByEmail. Error: %s", err.Error())
			}
			t.Logf("User %+v", user)
			if existingUser.Name == "NA" {
				t.Fatalf("Name update failed for GetOrUpdateUser")
			}
			// Test for GetTenantClaimRegistrationCode
			regCode := router.GetTenantClaimRegistrationCode(ctx, "test@nutanix.com", false)
			if regCode != router.NutanixRegistrationID {
				t.Fatalf("Mismatched registration code. Expected %s, found %s", router.NutanixRegistrationID, regCode)
			}
			regCode = router.GetTenantClaimRegistrationCode(ctx, "test@nutanix.com", true)
			if len(regCode) != 0 {
				t.Fatalf("Mismatched registration code. No registration expected, found %s", regCode)
			}
			regCode = router.GetTenantClaimRegistrationCode(ctx, "test@NuTanix.com", false)
			if regCode != router.NutanixRegistrationID {
				t.Fatalf("Mismatched registration code. Expected %s, found %s", router.NutanixRegistrationID, regCode)
			}
			regCode = router.GetTenantClaimRegistrationCode(ctx, "test@gmail.com", false)
			if regCode != router.NonNutanixRegistrationID {
				t.Fatalf("Mismatched registration code. Expected %s, found %s", router.NonNutanixRegistrationID, regCode)
			}
			regCode = router.GetTenantClaimRegistrationCode(ctx, "test@gmail.com", true)
			if regCode != router.NonNutanixNonTrialRegistrationID {
				t.Fatalf("Mismatched registration code. Expected %s, found %s", router.NonNutanixNonTrialRegistrationID, regCode)
			}
		})
	*/
	t.Run("Test Login Token Types", func(t *testing.T) {
		token := loginUser(t, netClient, user)
		loginResponse := &router.LoginResponse{}
		_, err := doPost(netClient, "/v1.0/login/shortlogintoken", token, "{}", loginResponse)
		require.NoError(t, err)
		shortToken := loginResponse.Token
		_, err = doPost(netClient, "/v1.0/login/shortlogintoken", shortToken, "{}", loginResponse)
		require.Error(t, err, "Requesting short token with short token must fail")
		_, err = doPost(netClient, "/v1.0/login/logintoken", token, "{}", loginResponse)
		require.Error(t, err, "Requesting long token with default token must fail")
		_, err = doPost(netClient, "/v1.0/login/logintoken", shortToken, "{}", loginResponse)
		require.NoError(t, err)
		longToken := loginResponse.Token
		_, err = doPost(netClient, "/v1.0/login/shortlogintoken", longToken, "{}", loginResponse)
		require.Error(t, err, "Requesting short token with long token must fail")
		_, err = doPost(netClient, "/v1.0/login/logintoken", longToken, "{}", loginResponse)
		require.Error(t, err, "Requesting long token with long token must fail")
	})

}

func TestTrialLogin(t *testing.T) {
	t.Parallel()
	if disableTrialLoginTest {
		return
	}
	// Enable trial login
	*config.Cfg.EnableTrial = true
	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, apitesthelper.GetHTTPClient())
	oAuth2Config := apitesthelper.OAuth2Config
	oAuth2Handler := auth.NewOAuthHandler("Iot", oAuth2Config.ClientID, oAuth2Config.ClientSecret, apitesthelper.IDP_BASE_URL, oAuth2Config.RedirectURL)
	code, err := apitesthelper.GetOAuthCode(t, nonNutanixEmail, nonNutanixPassword)
	require.NoError(t, err, "Error getting OAuth2 code")
	doc := model.OAuthCodes{Code: code}
	t.Logf("Getting token")
	tokens, err := oAuth2Handler.GetOAuthToken(ctx, &doc)
	require.NoError(t, err, "Error getting token")
	t.Logf("Getting XI IOT role")
	iotRole, err := auth.GetXIIOTRole(ctx, tokens)
	require.NoError(t, err, "Get XI IOT role failed")
	tenantPoolManager, err := core.NewTenantPoolManager(TenantPoolEdgeProvisioner)
	require.NoError(t, err, "Error creating tenantpool manager")
	bookKeeper := tenantPoolManager.GetBookKeeper()
	regConfig := &tenantpoolmodel.RegistrationConfigV1{
		VersionInfo:           tenantpoolmodel.VersionInfo{Version: tenantpoolmodel.RegConfigV1},
		EdgeCount:             1,
		MinTenantPoolSize:     2,
		MaxTenantPoolSize:     5,
		MaxPendingTenantCount: 2,
		TrialExpiry:           time.Hour,
	}
	configJSON, err := json.Marshal(regConfig)
	require.NoError(t, err)
	router.NonNutanixRegistrationID = base.GetUUID()
	router.NutanixRegistrationID = base.GetUUID()
	nonNutanixReg := &tenantpoolmodel.Registration{
		ID:          router.NonNutanixRegistrationID,
		Config:      string(configJSON),
		Description: "non-nutanix",
		State:       core.Active,
	}
	nutanixReg := &tenantpoolmodel.Registration{
		ID:          router.NutanixRegistrationID,
		Config:      string(configJSON),
		Description: "nutanix",
		State:       core.Active,
	}

	cleaner := func() {
		bookKeeper.PurgeTenants(ctx, nonNutanixReg.ID)
		bookKeeper.DeleteRegistration(ctx, nonNutanixReg.ID)
		bookKeeper.PurgeTenants(ctx, nutanixReg.ID)
		bookKeeper.DeleteRegistration(ctx, nutanixReg.ID)
	}

	// Cleanup after the tests
	defer cleaner()

	err = dbAPI.CreateRegistration(ctx, nonNutanixReg)
	require.NoError(t, err, "Failed to create non-nutanix registration")
	err = dbAPI.CreateRegistration(ctx, nutanixReg)
	require.NoError(t, err, "Failed to create nutanix registration")
	testhelper.WaitForState(t, bookKeeper, nonNutanixReg.ID, []string{core.Creating}, 2, 2)
	testhelper.WaitForState(t, bookKeeper, nutanixReg.ID, []string{core.Creating}, 2, 2)
	removeNonNutanixUser(ctx, t, dbAPI)
	// The user has no relationship with any tenant now
	_, _, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
	require.Error(t, err, "New Nutanix user must fail")
	TenantPoolEdgeProvisioner.SetEdgeStatusByCount(ctx, 0, nutanixReg.ID, core.Created)
	testhelper.WaitForState(t, bookKeeper, nutanixReg.ID, []string{core.Available}, 2, 2)
	removeNonNutanixUser(ctx, t, dbAPI)
	_, _, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
	require.Error(t, err, "New Nutanix user must still fail because non-nutanix registration is in creating state")
	TenantPoolEdgeProvisioner.SetEdgeStatusByCount(ctx, 0, nonNutanixReg.ID, core.Created)
	testhelper.WaitForState(t, bookKeeper, nonNutanixReg.ID, []string{core.Available}, 2, 2)
	_, claims, err := router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
	require.NoError(t, err, "New Nutanix user must not fail")

	if _, ok := claims["trialExpiry"]; !ok {
		t.Fatal("Trial expiry must be present")
	}
	// Calling again should still return the trial expiry for existing trial user
	_, claims, err = router.GetOrCreateUser(ctx, dbAPI, tokens, iotRole)
	require.NoError(t, err, "Existing Nutanix user must not fail")

	if _, ok := claims["trialExpiry"]; !ok {
		t.Fatal("Trial expiry must be present")
	}
}

func TestLoginFailure(t *testing.T) {
	t.Parallel()
	t.Log("Running TestLoginFailure test")

	var netClient = &http.Client{
		Timeout: time.Minute,
	}

	// Setup
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	tenant := apitesthelper.CreateTenant(t, dbAPI, "test tenant")
	tenantID := tenant.ID
	user := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")
	user2 := apitesthelper.CreateUser(t, dbAPI, tenantID, "INFRA_ADMIN")

	// Teardown
	defer func() {
		authContext := &base.AuthContext{
			TenantID: tenantID,
			Claims: jwt.MapClaims{
				"specialRole": "admin",
			},
		}
		ctx := context.WithValue(context.Background(), base.AuthContextKey, authContext)
		dbAPI.DeleteUser(ctx, user2.ID, nil)
		dbAPI.DeleteUser(ctx, user.ID, nil)
		dbAPI.DeleteBuiltinTenantObjects(ctx, tenantID)
		dbAPI.DeleteTenant(ctx, tenantID, nil)
		dbAPI.Close()
	}()

	t.Run("Test Login Failure", func(t *testing.T) {
		var err error
		var failureCountThreshold = *config.Cfg.LoginFailureCountThreshold
		var loginLockDuration = time.Duration(int64(*config.Cfg.LoginLockDurationSeconds) * int64(time.Second))

		badPassword := "foobar"
		goodPassword := apitesthelper.UserPassword

		_, err = login(netClient, user.Email, goodPassword)
		require.NoError(t, err, "expect login to succeed with good password")

		_, err = login(netClient, user2.Email, goodPassword)
		require.NoError(t, err, "expect login 2 to succeed with good password")

		// first FailureCountThreshold login failure will be auth errors
		_, authError := login(netClient, user.Email, badPassword)
		for i := 1; i < failureCountThreshold; i++ {
			_, err := login(netClient, user.Email, badPassword)
			if false == reflect.DeepEqual(err, authError) {
				t.Fatal("expect initial errors to be auth errors")
			}
		}

		// now login API endpoint is locked for user, but not for user2
		_, lockedError := login(netClient, user.Email, goodPassword)
		if lockedError == nil {
			t.Fatal("expect login to fail with good password")
		}
		_, err = login(netClient, user2.Email, goodPassword)
		require.NoError(t, err, "expect login 2 to succeed with good password")

		if reflect.DeepEqual(lockedError, authError) {
			t.Fatal("expect locked error != auth error")
		}

		ch := make(chan int, failureCountThreshold)
		// even login with good password will fail for user at this point
		for i := 0; i < failureCountThreshold; i++ {
			// launch login in parallel
			go func() {
				_, err := login(netClient, user.Email, goodPassword)
				if false == reflect.DeepEqual(err, lockedError) {
					t.Fatal("expect user login to give locked error")
				}
				_, err = login(netClient, user2.Email, goodPassword)
				require.NoError(t, err, "expect user2 login to succeed")
				ch <- i
			}()
		}
		// wait for all done
		for i := 0; i < failureCountThreshold; i++ {
			<-ch
		}

		// wait till email is unlocked for login
		t.Logf(">>> Sleeping for %d seconds", loginLockDuration/time.Second)
		time.Sleep(loginLockDuration)

		// now login good password should again be ok
		for i := 0; i < failureCountThreshold; i++ {
			// launch login in parallel
			go func() {
				_, err := login(netClient, user.Email, goodPassword)
				require.NoError(t, err, "expect user login to succeed")

				_, err = login(netClient, user2.Email, goodPassword)
				require.NoError(t, err, "expect user2 login to succeed")

				ch <- i
			}()
		}
		// wait for all done
		for i := 0; i < failureCountThreshold; i++ {
			<-ch
		}

		// login with bad password should again give auth error
		_, err = login(netClient, user.Email, badPassword)
		if false == reflect.DeepEqual(err, authError) {
			t.Fatal("expect new user login error to be auth error")
		}
		_, err = login(netClient, user2.Email, badPassword)
		if false == reflect.DeepEqual(err, authError) {
			t.Fatal("expect new user2 login error to be auth error")
		}

	})
}

// removeNonNutanixUser removes the existing non nutanix user if present
func removeNonNutanixUser(ctx context.Context, t *testing.T, dbAPI api.ObjectModelAPI) {
	authContext := &base.AuthContext{
		Claims: jwt.MapClaims{
			"specialRole": "admin",
		},
	}
	// Setup the user
	// Remove the test user and the tenant mapping so that the user
	existingUser, err := dbAPI.GetUserByEmail(ctx, nonNutanixEmail)
	if err == nil {
		authContext.TenantID = existingUser.TenantID
		ctx = context.WithValue(ctx, base.AuthContextKey, authContext)
		_, err = dbAPI.DeleteUser(ctx, existingUser.ID, nil)
		require.NoErrorf(t, err, "Error deleting existing user %s", existingUser.ID)
	}
	// Get the tenant for the test user
	existingTenant, err := dbAPI.GetTenant(ctx, nonNutanixExternalID)
	if err == nil {
		// Remove the external ID if present
		if len(existingTenant.ExternalID) > 0 {
			existingTenant.ExternalID = ""
			_, err = dbAPI.UpdateTenant(ctx, &existingTenant, nil)
			require.NoError(t, err, "Error updating tenant")
		}
	}
}

func TestLoginToken(t *testing.T) {
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	inClaims := jwt.MapClaims{}
	user := &model.User{Email: "test@ntnxsherlock.com", Name: "test"}
	user.TenantID = "123"
	user.ID = "456"
	token := router.GetShortUserJWTToken(dbAPI, user, nil, 300, inClaims)
	r := httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	jwtToken, err := auth.VerifyAuthorization(r, dbAPI.GetPublicKeyResolver, dbAPI.GetClaimsVerifier)
	require.NoError(t, err)
	t.Logf("%+v", jwtToken)
	if len(jwtToken) != 4 {
		// email, exp and nbf
		t.Fatalf("Expected 4, found %d", len(jwtToken))
	}
	if _, ok := jwtToken["tenantId"]; ok {
		t.Fatalf("TenantID found in short token")
	}
	longToken := api.GetUserJWTToken(dbAPI, user, nil, 600, api.DefaultTokenType, inClaims)
	r = httptest.NewRequest(http.MethodPost, "/login", nil)
	r.Header.Set("Authorization", "Bearer "+longToken)
	jwtToken, err = auth.VerifyAuthorization(r, dbAPI.GetPublicKeyResolver, dbAPI.GetClaimsVerifier)
	require.NoError(t, err)
	t.Logf("%+v", jwtToken)
	var tenantID interface{}
	var ok bool
	if tenantID, ok = jwtToken["tenantId"]; !ok {
		t.Fatalf("TenantID not found in long token")
	}
	if tenantID.(string) != "123" {
		t.Fatalf("Expected tenantID 123 but found %s", tenantID)
	}
}

func TestOAuthHandlerRegistration(t *testing.T) {
	iotRedirectURL := "https://my.ntnxsherlock.com/auth/oauth?clientId=123&clientSecret=456"
	grayMatterRedirectURL := "https://graymatter-my.ntnxsherlock.com/auth/oauth?clientId=abc&clientSecret=def"
	redirectURLs := []string{iotRedirectURL, grayMatterRedirectURL}
	dbAPI, err := api.NewObjectModelAPI()
	require.NoError(t, err)
	err = router.RegisterOAuthHandlers(dbAPI, redirectURLs)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodGet, "https://my.ntnxsherlock.com/v1/test", nil)
	oAuthHandler, err := router.LookupOAuthHandler(dbAPI, r)
	require.NoError(t, err)
	if oAuthHandler == nil {
		t.Fatal("OAuthHandler must not be nil")
	}
	redirectURL := oAuthHandler.GetAuthRedirectURL("")
	if !strings.Contains(redirectURL, url.QueryEscape("https://my.ntnxsherlock.com")) {
		t.Fatalf("Wrong redirect returned %s", redirectURL)
	}
	// append invalid
	redirectURLs = append(redirectURLs, "https://test-my.ntnxsherlock.com/auth/oauth")
	err = router.RegisterOAuthHandlers(dbAPI, redirectURLs)
	require.Error(t, err, "Must fail due to invalid URL")
}

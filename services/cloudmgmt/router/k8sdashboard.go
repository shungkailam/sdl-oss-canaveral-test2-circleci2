package router

import (
	"cloudservices/cloudmgmt/api"
	"cloudservices/cloudmgmt/config"
	"cloudservices/common/auth"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"sigs.k8s.io/yaml"
)

// Use proxy to call kubernetes API at the Edge
// to get k8s dashboard token for admin or viewonly service accounts

// On Service Domain or Karbon Kubernetes cluster,
// k8s dashboard is deployed in the standard location:
// in kubernetes-dashboard namespace.
// In the same namespace, we have created two service accounts:
// admin-user and viewonly-user
// Here we provide API to retrieve k8s tokens for these two service accounts.
// Also, API to retrieve kube config for these service accounts,
// using the token + our proxy endpoint + ca cert from our DNS certificate
// provider (in this case AWS ROOT CA 1)

const (
	dashboardBaseURL          = "https://kubernetes.default.svc/api/v1/namespaces/kubernetes-dashboard"
	dashboardSvcAcctsBaseURL  = dashboardBaseURL + "/serviceaccounts"
	dashboardSecretsBaseURL   = dashboardBaseURL + "/secrets"
	dashboardAdminUserPath    = "/admin-user"
	dashboardViewonlyUserPath = "/viewonly-user"

	// used by dev DNS (*.ntnxsherlock.com)
	AWS_ROOT_CA_1 = `-----BEGIN CERTIFICATE-----
MIIDQTCCAimgAwIBAgITBmyfz5m/jAo54vB4ikPmljZbyjANBgkqhkiG9w0BAQsF
ADA5MQswCQYDVQQGEwJVUzEPMA0GA1UEChMGQW1hem9uMRkwFwYDVQQDExBBbWF6
b24gUm9vdCBDQSAxMB4XDTE1MDUyNjAwMDAwMFoXDTM4MDExNzAwMDAwMFowOTEL
MAkGA1UEBhMCVVMxDzANBgNVBAoTBkFtYXpvbjEZMBcGA1UEAxMQQW1hem9uIFJv
b3QgQ0EgMTCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALJ4gHHKeNXj
ca9HgFB0fW7Y14h29Jlo91ghYPl0hAEvrAIthtOgQ3pOsqTQNroBvo3bSMgHFzZM
9O6II8c+6zf1tRn4SWiw3te5djgdYZ6k/oI2peVKVuRF4fn9tBb6dNqcmzU5L/qw
IFAGbHrQgLKm+a/sRxmPUDgH3KKHOVj4utWp+UhnMJbulHheb4mjUcAwhmahRWa6
VOujw5H5SNz/0egwLX0tdHA114gk957EWW67c4cX8jJGKLhD+rcdqsq08p8kDi1L
93FcXmn/6pUCyziKrlA4b9v7LWIbxcceVOF34GfID5yHI9Y/QCB/IIDEgEw+OyQm
jgSubJrIqg0CAwEAAaNCMEAwDwYDVR0TAQH/BAUwAwEB/zAOBgNVHQ8BAf8EBAMC
AYYwHQYDVR0OBBYEFIQYzIU07LwMlJQuCFmcx7IQTgoIMA0GCSqGSIb3DQEBCwUA
A4IBAQCY8jdaQZChGsV2USggNiMOruYou6r4lK5IpDB/G/wkjUu0yKGX9rbxenDI
U5PMCCjjmCXPI6T53iHTfIUJrU6adTrCC2qJeHZERxhlbI1Bjjt/msv0tadQ1wUs
N+gDS63pYaACbvXy8MWy7Vu33PqUXHeeE6V/Uq2V8viTO96LXFvKWlJbYK8U90vv
o/ufQJVtMVT8QtPHRh8jrdkPSHCa2XV4cdFyQzR1bldZwgJcJmApzyMZFo6IQ6XU
5MsI+yMRQ+hDKXJioaldXgjUkK642M4UwtBV8ob2xJNDd2ZhwLnoQdeXeGADbkpy
rqXRfboQnoZsG4q5WTP468SQvvG5
-----END CERTIFICATE-----`

	// used by prod / stage DNS
	// we may need this in the future if we switch from wst-beta.ntnxsherlock.com
	// to something like wst-prod.iot.nutanix.com
	DIGI_CERT_GLOBAL_ROOT_CA = `-----BEGIN CERTIFICATE-----
MIIDrzCCApegAwIBAgIQCDvgVpBCRrGhdWrJWZHHSjANBgkqhkiG9w0BAQUFADBh
MQswCQYDVQQGEwJVUzEVMBMGA1UEChMMRGlnaUNlcnQgSW5jMRkwFwYDVQQLExB3
d3cuZGlnaWNlcnQuY29tMSAwHgYDVQQDExdEaWdpQ2VydCBHbG9iYWwgUm9vdCBD
QTAeFw0wNjExMTAwMDAwMDBaFw0zMTExMTAwMDAwMDBaMGExCzAJBgNVBAYTAlVT
MRUwEwYDVQQKEwxEaWdpQ2VydCBJbmMxGTAXBgNVBAsTEHd3dy5kaWdpY2VydC5j
b20xIDAeBgNVBAMTF0RpZ2lDZXJ0IEdsb2JhbCBSb290IENBMIIBIjANBgkqhkiG
9w0BAQEFAAOCAQ8AMIIBCgKCAQEA4jvhEXLeqKTTo1eqUKKPC3eQyaKl7hLOllsB
CSDMAZOnTjC3U/dDxGkAV53ijSLdhwZAAIEJzs4bg7/fzTtxRuLWZscFs3YnFo97
nh6Vfe63SKMI2tavegw5BmV/Sl0fvBf4q77uKNd0f3p4mVmFaG5cIzJLv07A6Fpt
43C/dxC//AH2hdmoRBBYMql1GNXRor5H4idq9Joz+EkIYIvUX7Q6hL+hqkpMfT7P
T19sdl6gSzeRntwi5m3OFBqOasv+zbMUZBfHWymeMr/y7vrTC0LUq7dBMtoM1O/4
gdW7jVg/tRvoSSiicNoxBN33shbyTApOB6jtSj1etX+jkMOvJwIDAQABo2MwYTAO
BgNVHQ8BAf8EBAMCAYYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUA95QNVbR
TLtm8KPiGxvDl7I90VUwHwYDVR0jBBgwFoAUA95QNVbRTLtm8KPiGxvDl7I90VUw
DQYJKoZIhvcNAQEFBQADggEBAMucN6pIExIK+t1EnE9SsPTfrgT1eXkIoyQY/Esr
hMAtudXH/vTBH1jLuG2cenTnmCmrEbXjcKChzUyImZOMkXDiqw8cvpOp/2PV5Adg
06O/nVsJ8dWO41P0jmP6P6fbtGbfYmbW0W5BjfIttep3Sp+dWOIrWcBAI+0tKIJF
PnlUkiaY4IBIqDfv8NZ5YBberOgOzW6sRBc4L0na4UU+Krk2U886UAb3LujEV0ls
YSEY1QSteDwsOoBrp+uvFRTp2InBuThs4pFsiv9kuXclVzDAGySj4dzp30d8tbQk
CAUw7C29C79Fv1C5qfPrmAESrciIxpg0X40KPMbp1ZWVbd4=
-----END CERTIFICATE-----`
)

// KubernetesUserType is type of Kubernetes access requested by user
type KubernetesUserType int

const (
	// ViewOnlyUserType user have read-only access to k8s cluster
	ViewOnlyUserType = KubernetesUserType(0)
	// AdminUserType users have full access to k8s cluster
	AdminUserType = iota
	// ProjectUserType user have restricted access to k8s namespaces of assigned projects.
	ProjectUserType = iota
)

type k8sDashboardSecretName struct {
	Name string `json:"name"`
}
type k8sDashboardSvcAccountResponse struct {
	Secrets []k8sDashboardSecretName `json:"secrets"`
}
type k8sDashboardSecretData struct {
	Token string `json:"token"`
}
type k8sDashboardSecretResponse struct {
	Data k8sDashboardSecretData `json:"data"`
}

// from service account, we can get secret name
// e.g., admin-user service account => admin-user-token-rhpjr secret name
func parseSecretNameFromSA(sa []byte) (string, error) {
	sar := k8sDashboardSvcAccountResponse{}
	err := json.Unmarshal(sa, &sar)
	if err != nil {
		return "", err
	}
	if len(sar.Secrets) == 0 {
		return "", fmt.Errorf("Secrets not found for service account")
	}
	return sar.Secrets[0].Name, nil
}

// from secret details we can get token
// this function also perform base64 decode of the token
func parseTokenFromSecret(secret []byte) (string, error) {
	sr := k8sDashboardSecretResponse{}
	err := json.Unmarshal(secret, &sr)
	if err != nil {
		return "", err
	}
	token, err := base64.StdEncoding.DecodeString(sr.Data.Token)
	if err != nil {
		return "", err
	}
	return string(token), nil
}

// kubernetes API server is accessible in default namespace
// under the service name 'kubernetes' at port 443
// Note: in endpoint path, we replace : with .
func makeK8sApiProxyEndpoint(svcDomainId string) string {
	baseURL := *config.Cfg.ProxyUrlBase
	endpointPath := fmt.Sprintf(svcDomainId + "-kubernetes.default.svc.443")
	return model.MakeProxyURL(baseURL, endpointPath)
}

// isViewonlyAccessGranted returns true if the given user
// has been granted viewonly access to the given service domain
func isViewonlyAccessGranted(dbAPI api.ObjectModelAPI, ctx context.Context, svcDomainId, userId string) bool {
	users, err := dbAPI.GetViewonlyUsersForSD(ctx, svcDomainId)
	if err != nil {
		return false
	}
	for _, u := range users {
		if u.ID == userId {
			return true
		}
	}
	return false
}

func dashboardProjectUserPath(userID string) string {
	return fmt.Sprintf("/project-user-%s", userID)
}

// TODO FIXME: when we move to new object model,
// need to adapt this so it works for both service domain and karbon cluster
func makeDashboardHandle(dbAPI api.ObjectModelAPI,
	msgSvc api.WSMessagingService,
	msg string, userType KubernetesUserType, isKubeConfig bool) httprouter.Handle {
	return getContext(dbAPI, CheckAuth(dbAPI, func(w http.ResponseWriter, r *http.Request, ps httprouter.Params, ap *base.AuthContext) {
		ctx := r.Context()
		w.Header().Set("Content-Type", "application/json")
		svcDomainId := ps.ByName("svcDomainId")
		glog.V(4).Infof(base.PrefixRequestID(ctx, "K8s Dashboard handler: svcDomainId: %s, request: %+v"), svcDomainId, r)

		var err error
		var token string
		var u string
		userID, _ := ap.Claims["id"].(string)
		// use for/break as structured goto
	loop:
		for {
			// RBAC
			isInfraAdminRole := auth.IsInfraAdminRole(ap)
			switch userType {
			case AdminUserType:
				// admin access
				if !isInfraAdminRole {
					err = errcode.NewPermissionDeniedError(fmt.Sprintf("tid|uid=%s|%s", ap.TenantID, userID))
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: RBAC error for admin: %s"), err)
					break loop
				}
			case ViewOnlyUserType:
				// viewonly access
				if !isInfraAdminRole && !isViewonlyAccessGranted(dbAPI, ctx, svcDomainId, userID) {
					err = errcode.NewPermissionDeniedError(fmt.Sprintf("tid|uid=%s|%s", ap.TenantID, userID))
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: RBAC error for viewonly: %s"), err)
					break loop
				}
			case ProjectUserType:
				var featuresMap map[string]*model.Features
				// Check whether SD support direct k8s access for project users
				featuresMap, err = dbAPI.GetFeaturesForServiceDomains(ctx, []string{svcDomainId})
				if err != nil {
					errStr := fmt.Sprintf("Error in getting features for service domain %s. Error: %s", svcDomainId, err)
					err = errcode.NewInternalError(errStr)
					glog.Warningf(base.PrefixRequestID(ctx, errStr))
					break loop
				}
				if f, ok := featuresMap[svcDomainId]; ok {
					if f.ProjectUserKubeConfig == false {
						errStr := fmt.Sprintf("Project user kubectl access not available for service domain %s", svcDomainId)
						err = errcode.NewInternalError(errStr)
						glog.Warningf(base.PrefixRequestID(ctx, errStr))
						break loop
					}
				}
			}
			switch userType {
			case AdminUserType:
				u = dashboardSvcAcctsBaseURL + dashboardAdminUserPath
			case ViewOnlyUserType:
				u = dashboardSvcAcctsBaseURL + dashboardViewonlyUserPath
			case ProjectUserType:
				u = dashboardSvcAcctsBaseURL + dashboardProjectUserPath(userID)
			}
			ul, err2 := url.Parse(u)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: url[%s] error: %s"), u, err)
				break
			}
			r.URL = ul
			resp, err2 := msgSvc.SendHTTPRequest(ctx, ap.TenantID, svcDomainId, r, u)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: send sa http request error: %s"), err)
				break
			}
			defer resp.Body.Close()
			body, _ := ioutil.ReadAll(resp.Body)
			saName, err2 := parseSecretNameFromSA(body)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: parse sa error: %s"), err)
				break
			}
			glog.V(4).Infof(base.PrefixRequestID(ctx, "K8s Dashboard handler: svcDomainId: %s, secret name: %s"), svcDomainId, saName)
			secretURL := dashboardSecretsBaseURL + "/" + saName
			ul, err2 = url.Parse(secretURL)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: url[%s] error: %s"), secretURL, err)
				break
			}
			r.URL = ul
			resp, err2 = msgSvc.SendHTTPRequest(ctx, ap.TenantID, svcDomainId, r, secretURL)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: send secret http request error: %s"), err)
				break
			}
			defer resp.Body.Close()
			body, _ = ioutil.ReadAll(resp.Body)
			token, err2 = parseTokenFromSecret(body)
			if err2 != nil {
				err = err2
				glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: parse token error: %s"), err)
				break
			}
			if isKubeConfig {
				// use SD name as k8s cluster name
				// already done RBAC check, but if viewonly user, will not be able to get svc domain
				// so use different ctx for this
				ctx2 := ctx
				if !isInfraAdminRole {
					ctx2 = base.GetAdminContextWithTenantID(ctx, ap.TenantID)
				}
				svcDomain, err2 := dbAPI.GetServiceDomain(ctx2, svcDomainId)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: get service domain error: %s"), err)
					break
				}
				endpoint := makeK8sApiProxyEndpoint(svcDomainId)
				caEncoded := base64.StdEncoding.EncodeToString([]byte(AWS_ROOT_CA_1))
				kcfg := model.MakeKubeConfig(svcDomain.Name, token, endpoint, caEncoded)
				baKcfg, err2 := yaml.Marshal(*kcfg)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: marshal kubeconfig error: %s"), err)
					break
				}
				payload := model.KubeConfigPayload{
					KubeConfig: string(baKcfg),
				}
				baResp, err2 := json.Marshal(payload)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: marshal response error: %s"), err)
					break
				}
				_, err2 = w.Write(baResp)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: req[%+v], write error: %s"), r, err)
					break
				}
			} else {
				payload := model.K8sDashboardTokenResponsePayload{
					Token: token,
				}
				baResp, err2 := json.Marshal(payload)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: marshal response error: %s"), err)
					break
				}
				_, err2 = w.Write(baResp)
				if err2 != nil {
					err = err2
					glog.Warningf(base.PrefixRequestID(ctx, "K8s Dashboard handler: req[%+v], write error: %s"), r, err)
					break
				}
			}
			// done, break out of for loop
			break
		}
		handleResponse(w, r, err, "K8S DASHBOARD %s, tenantID=%s", msg, ap.TenantID)
	}))
}

func getK8sDashboardRoutes(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) []routeHandle {

	AdminTokenHandle := makeDashboardHandle(dbAPI, msgSvc, "k8sDashboard", AdminUserType, false)
	ViewonlyTokenHandle := makeDashboardHandle(dbAPI, msgSvc, "k8sDashboard", ViewOnlyUserType, false)
	ProjectUserTokenHandle := makeDashboardHandle(dbAPI, msgSvc, "k8sDashboard", ProjectUserType, false)
	AdminKubeConfigHandle := makeDashboardHandle(dbAPI, msgSvc, "k8sDashboard", AdminUserType, true)
	ViewonlyKubeConfigHandle := makeDashboardHandle(dbAPI, msgSvc, "k8sDashboard", ViewOnlyUserType, true)
	ProjectUserKubeConfigHandle := makeDashboardHandle(dbAPI, msgSvc, "k8sDashboard", ProjectUserType, true)

	return []routeHandle{

		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/adminToken",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/adminToken K8sDashboard K8sDashboardGetAdminToken
			//
			// Get Admin Token for Kubernetes Dashboard. ntnx:ignore
			//
			// Get Admin Token for Kubernetes Dashboard. Caller must be infra admin.
			// svcDomainId is ID of service domain or kubernetes cluster
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
			//       200: K8sDashboardTokenResponse
			//       default: APIError
			handle: AdminTokenHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/viewonlyToken",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/viewonlyToken K8sDashboard K8sDashboardGetViewonlyToken
			//
			// Get Viewonly Token for Kubernetes Dashboard. ntnx:ignore
			//
			// Get Viewonly Token for Kubernetes Dashboard. Caller must have viewonly access.
			// svcDomainId is ID of service domain or kubernetes cluster
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
			//       200: K8sDashboardTokenResponse
			//       default: APIError
			handle: ViewonlyTokenHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/userToken",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/userToken K8sDashboard K8sDashboardGetUserToken
			//
			// Get User Token for Kubernetes Dashboard. ntnx:ignore
			//
			// Get User Token for Kubernetes Dashboard.
			// svcDomainId is ID of service domain or kubernetes cluster
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
			//       200: K8sDashboardTokenResponse
			//       default: APIError
			handle: ProjectUserTokenHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/adminKubeConfig",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/adminKubeConfig K8sDashboard K8sDashboardGetAdminKubeConfig
			//
			// Get Admin KubeConfig for kubectl. ntnx:ignore
			//
			// Get Admin KubeConfig for kubectl. Caller must be infra admin.
			// svcDomainId is ID of service domain or kubernetes cluster
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
			//       200: K8sDashboardKubeConfigResponse
			//       default: APIError
			handle: AdminKubeConfigHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/userKubeConfig",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/userKubeConfig K8sDashboard K8sDashboardGetUserKubeConfig
			//
			// Get User KubeConfig for kubectl. ntnx:ignore
			//
			// Get User KubeConfig for kubectl.
			// svcDomainId is ID of service domain or kubernetes cluster
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
			//       200: K8sDashboardKubeConfigResponse
			//       default: APIError
			handle: ProjectUserKubeConfigHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/viewonlyKubeConfig",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/viewonlyKubeConfig K8sDashboard K8sDashboardGetViewonlyKubeConfig
			//
			// Get Viewonly KubeConfig for kubectl. ntnx:ignore
			//
			// Get Viewonly KubeConfig for kubectl. Caller must have viewonly access.
			// svcDomainId is ID of service domain or kubernetes cluster
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
			//       200: K8sDashboardKubeConfigResponse
			//       default: APIError
			handle: ViewonlyKubeConfigHandle,
		},
		{
			method: "GET",
			path:   "/v1.0/k8sdashboard/:svcDomainId/viewonlyUsers",
			// swagger:route GET /v1.0/k8sdashboard/{svcDomainId}/viewonlyUsers K8sDashboard K8sDashboardGetViewonlyUsers
			//
			// Get all kubernetes dashboard viewonly users associated with a Service Domain / Karbon Cluster. ntnx:ignore
			//
			// Retrieves a list of all kubernetes dashboard viewonly users associated with a Service Domain by its ID {svcDomainId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: K8sDashboardViewonlyUserListResponse
			//       default: APIError
			handle: makeEdgeGetAllHandle(dbAPI, dbAPI.GetViewonlyUsersForSDW, "k8sDashboard", "svcDomainId"),
		},
		{
			method: "POST",
			path:   "/v1.0/k8sdashboard/:svcDomainId/viewonlyUsersAdd",
			// swagger:route POST /v1.0/k8sdashboard/{svcDomainId}/viewonlyUsersAdd K8sDashboard K8sDashboardAddViewonlyUsers
			//
			// Add kubernetes dashboard viewonly users to a Service Domain / Karbon Cluster. ntnx:ignore
			//
			// Add kubernetes dashboard viewonly users to a Service Domain by its ID {svcDomainId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: K8sDashboardViewonlyUserUpdateResponse
			//       default: APIError
			handle: makePostHandle3(dbAPI, dbAPI.AddViewonlyUsersToSDW, "/v1.0/k8sdashboard/:svcDomainId/viewonlyUsersAdd", "svcDomainId"),
		},
		{
			method: "POST",
			path:   "/v1.0/k8sdashboard/:svcDomainId/viewonlyUsersRemove",
			// swagger:route POST /v1.0/k8sdashboard/{svcDomainId}/viewonlyUsersRemove K8sDashboard K8sDashboardRemoveViewonlyUsers
			//
			// Remove kubernetes dashboard viewonly users from a Service Domain / Karbon Cluster. ntnx:ignore
			//
			// Remove kubernetes dashboard viewonly users from a Service Domain by its ID {svcDomainId}.
			//
			//     Produces:
			//     - application/json
			//
			//     Security:
			//       - BearerToken:
			//
			//     Responses:
			//       200: K8sDashboardViewonlyUserUpdateResponse
			//       default: APIError
			handle: makePostHandle3(dbAPI, dbAPI.RemoveViewonlyUsersFromSDW, "/v1.0/k8sdashboard/:svcDomainId/viewonlyUsersRemove", "svcDomainId"),
		},
	}
}

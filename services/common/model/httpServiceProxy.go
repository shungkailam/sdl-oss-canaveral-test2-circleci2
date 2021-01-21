package model

import (
	"crypto/md5"
	"fmt"
	"strings"
	"time"
)

type HTTPServiceProxyCore struct {
	//
	// HTTP service proxy name.
	// Unique within (tenant, service domain)
	//
	// required: true
	Name string `json:"name" validate:"range=1:200"`

	//
	// Service type for this http proxy.
	//
	// enum: SYSTEM,PROJECT,CUSTOM
	// required: true
	Type string `json:"type" db:"type" validate:"options=SYSTEM:PROJECT:CUSTOM"`

	// Name of the http service.
	// required: true
	ServiceName string `json:"serviceName" db:"service_name" validate:"range=1:64"`

	// Port of the http service.
	// required: true
	ServicePort int `json:"servicePort" db:"service_port"`

	// Namespace of the http service, required when TYPE = SYSTEM
	// required: false
	ServiceNamespace string `json:"serviceNamespace,omitempty" db:"service_namespace" validate:"range=0:200"`

	// Duration of the http service proxy.
	// Example: 600s, 20m, 24h, etc.
	// required: true
	Duration string `json:"duration" db:"duration" validate:"range=2:32"`

	// ntnx:ignore
	// Expires at timestamp - computed from createdAt, updatedAt time and duration
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`

	// Username to login to the service when setupBasicAuth=true.
	// required: false
	Username string `json:"username,omitempty" db:"username" validate:"range=1:32"`

	// Password to login to the service when setupBasicAuth=true.
	// required: false
	Password string `json:"password,omitempty" db:"password" validate:"range=10:64"`

	// ntnx:ignore
	// Name of statefulset host this proxy is served.
	// required: false
	Hostname string `json:"hostname,omitempty" db:"hostname" validate:"range=1:64"`

	// ntnx:ignore
	// TCP port on statefulset host this proxy is served.
	// required: false
	Hostport int `json:"hostport,omitempty" db:"hostport"`

	// ntnx:ignore
	// Public Key. Used for session tracking. Required in Delete payload.
	PublicKey *string `json:"publicKey,omitempty" db:"public_key"`
}

type HTTPServiceProxy struct {
	// required: true
	ServiceDomainEntityModel
	// required: true
	HTTPServiceProxyCore
	//
	// ID of parent project, required when TYPE = PROJECT.
	//
	// required: false
	ProjectID string `json:"projectId,omitempty" validate:"range=0:64"`

	// URL of the service proxy endpoint
	// required: true
	URL string `json:"url"`

	// DNS URL of the service proxy endpoint
	// Valid only if setupDNS is set to true when creating the service proxy
	// required: true
	DNSURL string `json:"dnsURL"`
}

func (a HTTPServiceProxy) GetEndpoint() string {
	if a.Type == "SYSTEM" {
		return fmt.Sprintf("%s.%s.svc:%d", a.ServiceName, a.ServiceNamespace, a.ServicePort)
	} else if a.Type == "CUSTOM" {
		return fmt.Sprintf("%s:%d", a.ServiceName, a.ServicePort)
	} else {
		return fmt.Sprintf("%s.project-%s.svc:%d", a.ServiceName, a.ProjectID, a.ServicePort)
	}
}
func (a HTTPServiceProxy) GetProxyEndpointPath() string {
	return fmt.Sprintf("%s-%s", a.SvcDomainID, strings.ReplaceAll(a.GetEndpoint(), ":", "."))
}

type HTTPServiceProxiesByID []HTTPServiceProxy

func (a HTTPServiceProxiesByID) Len() int           { return len(a) }
func (a HTTPServiceProxiesByID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a HTTPServiceProxiesByID) Less(i, j int) bool { return a[i].ID < a[j].ID }

// HTTPServiceProxyCreateParamPayload holds the parameters for creating HTTP Service Proxy
//
// swagger:model HTTPServiceProxyCreateParamPayload
type HTTPServiceProxyCreateParamPayload struct {
	// ntnx:ignore
	ID string `json:"id"`
	// required: true
	Name string `json:"name"`
	// enum: SYSTEM,PROJECT,CUSTOM
	// required: true
	Type string `json:"type"`

	ProjectID string `json:"projectId,omitempty"`
	// required: true
	ServiceName string `json:"serviceName"`
	// required: true
	ServicePort int `json:"servicePort"`

	// Namespace of the http service, required when TYPE = SYSTEM
	ServiceNamespace string `json:"serviceNamespace,omitempty"`

	// ID of Service Domain to create the http service proxy
	// required: true
	SvcDomainID string `json:"svcDomainId"`

	// Duration of the http service proxy.
	// Example: 600s, 20m, 24h, etc.
	// required: true
	Duration string `json:"duration"`

	// Whether to setup basic auth to protect the endpoint
	// required: true
	SetupBasicAuth bool `json:"setupBasicAuth"`

	// By default, a rewrite rule will be put in place to rewrite service URL path base to /
	// set this flag to true to retain the URL path base.
	// required: true
	DisableRewriteRules bool `json:"disableRewriteRules"`

	// Whether to setup DNS entry for this service.
	// Default is false. Might be useful for services
	// that do not work with URL path.
	// However, bear in mind it may take several minutes
	// for the DNS name to propagate/resolve.
	// required: true
	SetupDNS bool `json:"setupDNS"`

	// Whether the endpoint to proxy to is a TLS endpoint.
	// required: true
	TLSEndpoint bool `json:"tlsEndpoint"`

	// Whether to skip TLS certification verification for endpoint.
	// Only relevant when TLSEndpoint is true.
	// This should be set to true if the endpoint is using a self-signed certificate.
	// required: true
	SkipCertVerification bool `json:"skipCertVerification"`

	// JSON object representation of HTTP headers to overwrite.
	// May be useful for (https) endpoint that require
	// specific Host field for example.
	// required: false
	Headers string `json:"headers,omitempty"`
}

func (pb *HTTPServiceProxyCreateParamPayload) ToHTTPServiceProxy() HTTPServiceProxy {
	r := HTTPServiceProxy{}
	if pb != nil {
		r.ID = pb.ID
		r.Name = pb.Name
		r.Type = pb.Type
		r.ProjectID = pb.ProjectID
		r.ServiceName = pb.ServiceName
		r.ServicePort = pb.ServicePort
		r.ServiceNamespace = pb.ServiceNamespace
		r.SvcDomainID = pb.SvcDomainID
		r.Duration = pb.Duration
	}
	return r
}

// swagger:parameters HTTPServiceProxyCreate
// in: body
type HTTPServiceProxyCreateParam struct {
	// in: body
	// required: true
	Payload *HTTPServiceProxyCreateParamPayload `json:"body"`
}

// swagger:model HTTPServiceProxyCreateResponsePayload
type HTTPServiceProxyCreateResponsePayload struct {
	// ID of the entity
	// required: true
	ID string `json:"id"`

	// Expires at timestamp
	// required: true
	ExpiresAt time.Time `json:"expiresAt"`

	// URL of the service proxy endpoint
	// required: true
	URL string `json:"url"`

	// DNS URL of the service proxy endpoint
	// Valid only if setupDNS is set to true when creating the service proxy
	// required: true
	DNSURL string `json:"dnsURL"`

	// Username to login to the service when setupBasicAuth=true.
	// required: false
	Username string `json:"username,omitempty"`

	// Password to login to the service when setupBasicAuth=true.
	// required: false
	Password string `json:"password,omitempty"`
}

// Ok
// swagger:response HTTPServiceProxyCreateResponse
type HTTPServiceProxyCreateResponse struct {
	// in: body
	// required: true
	Payload *HTTPServiceProxyCreateResponsePayload
}

// swagger:model HTTPServiceProxyUpdateResponsePayload
type HTTPServiceProxyUpdateResponsePayload struct {
	// ID of the entity
	// required: true
	ID string `json:"id"`

	// Expires at timestamp
	ExpiresAt time.Time `json:"expiresAt"`

	// URL of the service proxy endpoint
	// required: true
	URL string `json:"url"`

	// DNS URL of the service proxy endpoint
	// Valid only if setupDNS is set to true when creating the service proxy
	// required: true
	DNSURL string `json:"dnsURL"`

	// Username to login to the service when setupBasicAuth=true.
	// required: false
	Username string `json:"username,omitempty"`

	// Password to login to the service when setupBasicAuth=true.
	// required: false
	Password string `json:"password,omitempty"`
}

// Ok
// swagger:response HTTPServiceProxyUpdateResponse
type HTTPServiceProxyUpdateResponse struct {
	// in: body
	// required: true
	Payload *HTTPServiceProxyUpdateResponsePayload
}

// HTTPServiceProxyUpdateParamPayload holds the parameters for updating HTTP Service Proxy
//
// swagger:model HTTPServiceProxyUpdateParamPayload
type HTTPServiceProxyUpdateParamPayload struct {
	Name string `json:"name"`
	// Duration of the http service proxy.
	// Example: 600s, 20m, 24h, etc.
	Duration string `json:"duration"`
	// By default, a rewrite rule will be put in place to rewrite service URL path base to /
	// set this flag to true to retain the URL path base.
	DisableRewriteRules bool `json:"disableRewriteRules"`
	// Whether to setup DNS entry for this service.
	// Default is false. Might be useful for services
	// that do not work with URL path.
	// However, bear in mind it may take several minutes
	// for the DNS name to propagate/resolve.
	// required: true
	SetupDNS bool `json:"setupDNS"`

	// Whether the endpoint to proxy to is a TLS endpoint.
	// required: true
	TLSEndpoint bool `json:"tlsEndpoint"`

	// Whether to skip TLS certification verification for endpoint.
	// Only relevant when TLSEndpoint is true.
	// This should be set to true if the endpoint is using a self-signed certificate.
	// required: true
	SkipCertVerification bool `json:"skipCertVerification"`

	// JSON object representation of HTTP headers to overwrite.
	// May be useful for (https) endpoint that require
	// specific Host field for example.
	// required: false
	Headers string `json:"headers,omitempty"`
}

// swagger:parameters HTTPServiceProxyUpdate
// in: body
type HTTPServiceProxyUpdateParam struct {
	// in: body
	// required: true
	Payload *HTTPServiceProxyUpdateParamPayload `json:"body"`
}

// Ok
// swagger:response HTTPServiceProxyListResponse
type HTTPServiceProxyListResponse struct {
	// in: body
	// required: true
	Payload *HTTPServiceProxyListPayload
}

// HTTPServiceProxyListPayload is the payload for HTTPServiceProxyListResponse
type HTTPServiceProxyListPayload struct {
	// required: true
	EntityListResponsePayload
	// list of HTTP Service Proxies
	// required: true
	HTTPServiceProxyList []HTTPServiceProxy `json:"result"`
}

// Ok
// swagger:response HTTPServiceProxyGetResponse
type HTTPServiceProxyGetResponse struct {
	// in: body
	// required: true
	Payload *HTTPServiceProxy
}

// HTTPServiceProxyQueryParam carries the first class query parameters
// swagger:parameters HTTPServiceProxyList
// in: query
type HTTPServiceProxyQueryParam struct {
	// Type of the HTTP Service Proxy
	// in: query
	// required: false
	Type string `json:"type"`

	// HTTP Service Proxy Project ID
	// in: query
	// required: false
	ProjectID string `json:"projectId"`

	// HTTP Service Proxy Service Domain ID
	// in: query
	// required: false
	SvcDomainID string `json:"svcDomainId"`

	// Name of the HTTP Service Proxy
	// in: query
	// required: false
	Name string `json:"name"`

	// ServiceName of the HTTP Service Proxy
	// in: query
	// required: false
	ServiceName string `json:"serviceName"`

	// ServiceNamespace of the HTTP Service Proxy
	// in: query
	// required: false
	ServiceNamespace string `json:"serviceNamespace"`
}

// swagger:parameters HTTPServiceProxyCreate HTTPServiceProxyUpdate HTTPServiceProxyList HTTPServiceProxyGet HTTPServiceProxyDelete
// in: header
type httpServiceProxyAuthorizationParam struct {
	// Format: Bearer <token>, with <token> from login API response.
	//
	// in: header
	// required: true
	Authorization string
}

func (doc HTTPServiceProxy) GetProjectID() string {
	return doc.ProjectID
}

func MakeProxyURL(baseURL, endpointPath string) string {
	i := strings.Index(baseURL, ".")
	pfx := baseURL[:i] // e.g., https://wst-<ns>
	sfx := baseURL[i:] // e.g., .ntnxsherlock.com
	// use md5 of endpointPath, since:
	// 1. we need to replace . in endpointPath
	// 2. DNS host part can have at most 63 chars
	m := fmt.Sprintf("%x", md5.Sum([]byte(endpointPath)))
	// https://wst-<ns>-<md5 of endpoint path>.ntnxsherlock.com
	return fmt.Sprintf("%s-%s%s", pfx, m, sfx)
}

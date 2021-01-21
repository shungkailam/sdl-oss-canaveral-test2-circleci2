package api

import (
	"cloudservices/cloudmgmt/cfssl"
	cfsslModels "cloudservices/cloudmgmt/generated/cfssl/models"
	"cloudservices/common/base"
	"cloudservices/common/model"
	"context"
	"io"

	"github.com/golang/glog"
)

// CreateCertificates creates a certificates object
func (dbAPI *dbObjectModelAPI) CreateCertificates(context context.Context, callback func(context.Context, interface{}) error) (interface{}, error) {
	resp := model.Certificates{}
	authContext, err := base.GetAuthContext(context)
	if err != nil {
		return resp, err
	}
	tenantID := authContext.TenantID
	// Create certificates using per-tenant root CA.
	certResp, err := cfssl.GetCert(tenantID, cfsslModels.CertificatePostParamsTypeClient)
	if err != nil {
		return resp, err
	}
	resp.Certificate = certResp.Cert
	resp.PrivateKey = certResp.Key
	resp.CACertificate = certResp.CACert
	// Ignoring callback because we do not talk to edge for certificates creation
	/* 	if callback != nil {
		go callback(context, doc)
	} */
	return resp, nil
}

// CreateCertificatesW creates a certificates object and writes output into writer
func (dbAPI *dbObjectModelAPI) CreateCertificatesW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	reqID := base.GetRequestID(context)
	httpRequestContext := base.GetHTTPRequestContext(context)
	glog.Infof(base.PrefixRequestID(context, "Request %s: URI: %s, Method: %s, Params: %s"), reqID, httpRequestContext.URI, httpRequestContext.Method, httpRequestContext.Params)
	resp, err := dbAPI.CreateCertificates(context, callback)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

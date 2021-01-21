package grpc

import (
	gapi "cloudservices/cloudmgmt/generated/grpc"
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"

	"bufio"
	"bytes"
	"net/url"

	"cloudservices/cloudmgmt/api"
	"cloudservices/common/base"

	"github.com/golang/glog"
)

// cloudmgmt gRPC server implementation
type GrpcServer struct {
	dbAPI  api.ObjectModelAPI
	msgSvc api.WSMessagingService
	gapi.UnimplementedCloudmgmtServiceServer
}

// responseToG convert http.Response to gapi.ProxyResponse
func responseToG(ctx context.Context, resp *http.Response) *gapi.ProxyResponse {
	baResp, err := httputil.DumpResponse(resp, true)
	status := "OK"
	statusCode := resp.StatusCode
	if err != nil {
		status = fmt.Sprintf("Response failed: %s", err.Error())
		statusCode = 500
		baResp = nil
	}
	return &gapi.ProxyResponse{
		Status:     status,
		Response:   baResp,
		StatusCode: int32(statusCode),
	}
}

// requestFromG convert http.Request from gapi.ProxyRequest
func requestFromG(ctx context.Context, gpreq *gapi.ProxyRequest) (req *http.Request, err error) {
	var r *http.Request
	r, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(gpreq.Request)))
	if err != nil {
		return
	}
	u, err := url.Parse(gpreq.Url)
	if err != nil {
		return
	}
	req = r
	req.RequestURI = ""
	req.URL = u
	req.Host = u.Host
	return
}

// SendHTTPRequest handle gRPC call, typically made from another cloudmgmt
// instance to send http request to an edge connected to this instance
func (s *GrpcServer) SendHTTPRequest(ctx context.Context, gpreq *gapi.ProxyRequest) (gresp *gapi.ProxyResponse, err error) {
	var gr *gapi.ProxyResponse
	var req *http.Request
	var resp *http.Response
	glog.Infof(base.PrefixRequestID(ctx, "HTTP proxy: gPRC: send http request: tenantID=%s, edgeID=%s, URL=%s"), gpreq.TenantId, gpreq.EdgeId, gpreq.Url)
	tenantID := gpreq.TenantId
	edgeID := gpreq.EdgeId
	url := gpreq.Url
	req, err = requestFromG(ctx, gpreq)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: gPRC: read request error: %s"), err)
		return
	}
	// typically the following should hit the direct connection path
	resp, err = s.msgSvc.SendHTTPRequest(ctx, tenantID, edgeID, req, url)
	if err != nil {
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: gPRC: send request error: %s"), err)
		return
	}
	gr = responseToG(ctx, resp)
	if gr.Status != "OK" {
		err = fmt.Errorf("Proxy API status: %s", gr.Status)
		glog.Warningf(base.PrefixRequestID(ctx, "HTTP proxy: gPRC: call failed: %s"), err)
		return
	}
	gresp = gr
	glog.Infof(base.PrefixRequestID(ctx, "HTTP proxy: gPRC: send http request successful"))
	return
}

func NewGrpcServer(dbAPI api.ObjectModelAPI, msgSvc api.WSMessagingService) gapi.CloudmgmtServiceServer {
	return &GrpcServer{
		dbAPI:  dbAPI,
		msgSvc: msgSvc,
	}
}

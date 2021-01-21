package api

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
	devtoolsApi "cloudservices/devtools/generated/grpc"
	"context"
	"io"

	"github.com/golang/glog"
	"google.golang.org/grpc"
)

func (dbAPI *dbObjectModelAPI) RequestLogStreamEndpoints(ctx context.Context, payload model.LogStream, callback func(context.Context, interface{}) error) (model.LogStreamEndpointsResponsePayload, error) {
	var err error
	resp := model.LogStreamEndpointsResponsePayload{}
	authContext, err := base.GetAuthContext(ctx)
	if err != nil {
		return resp, err
	}
	if payload.ApplicationID == "" && payload.DataPipelineID == "" {
		return resp, errcode.NewBadRequestError("applicationId/dataPipelineId")
	}

	// Call devtools API to get the pub/sub URLs
	var rpcResp *devtoolsApi.GetEndpointsResponse
	callerFunc := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := devtoolsApi.NewDevToolsServiceClient(conn)
		objectID := ""
		if payload.ApplicationID != "" {
			objectID = payload.ApplicationID
		} else {
			objectID = payload.DataPipelineID
		}
		request := &devtoolsApi.GetEndpointsRequest{
			TenantID:    authContext.TenantID,
			EdgeID:      payload.EdgeID,
			ObjectID:    objectID,
			ContainerID: payload.Container,
		}
		rpcResp, err = client.GetEndpoints(ctx, request)
		if err != nil {
			glog.Errorf("Unable to get urls for pubsub from devtools: %s", err.Error())
			return err
		}
		return nil
	}
	err = service.CallClient(ctx, service.DevToolsService, callerFunc)
	if err != nil {
		return resp, errcode.NewBadRequestExError("Endpoint", err.Error())
	}
	// Get the projectID of the app/pipeline to send to the edge.
	doc := model.WSMessagingLogStream{}
	doc.LogStreamInfo = &model.LogStream{
		EdgeID: payload.EdgeID,
	}
	if payload.ApplicationID != "" {
		if app, err := dbAPI.GetApplication(ctx, payload.ApplicationID); err != nil {
			return resp, errcode.NewBadRequestError("applicationID")
		} else {
			doc.ProjectID = app.ProjectID
			doc.LogStreamInfo.ApplicationID = payload.ApplicationID
		}
	} else {
		if p, err := dbAPI.GetDataStream(ctx, payload.DataPipelineID); err != nil {
			return resp, errcode.NewBadRequestError("dataPipelineID")
		} else {
			doc.ProjectID = p.ProjectID
			doc.LogStreamInfo.DataPipelineID = payload.DataPipelineID
		}
	}
	doc.LogStreamInfo.Container = payload.Container
	doc.URL = rpcResp.PublisherEndpoint
	err = callback(ctx, &doc)
	if err != nil {
		return resp, err
	}
	resp.URL = rpcResp.SubscriberEndpoint
	return resp, err
}

func (dbAPI *dbObjectModelAPI) RequestLogStreamEndpointsW(context context.Context, w io.Writer, r io.Reader, callback func(context.Context, interface{}) error) error {
	doc := model.LogStream{}
	err := base.Decode(&r, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(context, "Error decoding into logStream. Error: %s"), err.Error())
		return err
	}

	// Check for edge version for physical edges before proceeding.
	edge, err := dbAPI.GetEdge(context, doc.EdgeID)
	if err != nil {
		return errcode.NewInternalDatabaseError(err.Error())
	}
	if edge.Type == nil || *edge.Type != string(model.CloudTargetType) {
		// This is a physical edge so we need version check.
		edgeInfo, err := dbAPI.GetEdgeInfo(context, doc.EdgeID)
		if err != nil {
			return errcode.NewInternalDatabaseError(err.Error())
		}
		if edgeInfo.EdgeVersion == nil {
			// Use old version for upgrade as we need the data
			edgeInfo.EdgeVersion = nilVersion
		}
		feats, _ := GetFeaturesForVersion(*edgeInfo.EdgeVersion)
		if feats.RealTimeLogs != true {
			errMsg := "This feature is not supported on Edge Software Version v1.10 or below."
			return errcode.NewBadRequestExError("Edge version", errMsg)
		}
	}

	resp, err := dbAPI.RequestLogStreamEndpoints(context, doc, callback)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, resp)
}

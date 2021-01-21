package api

import (
	"context"
	"io"
	"net/http"

	"github.com/golang/glog"
	"google.golang.org/grpc"

	gapi "cloudservices/cloudmgmt/generated/auditlog"
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"cloudservices/common/model"
	"cloudservices/common/service"
)

func (objectModelAPI *dbObjectModelAPI) QueryAuditLogsV2(ctx context.Context, filter model.AuditLogV2Filter) ([]model.AuditLogV2, error) {
	//Convert input filter to compatible with audit log service api input

	request := &gapi.QueryAuditLogsRequest{}
	err := base.Convert(&filter, request)
	if err != nil {
		return nil, err
	}

	modelAuditLogs := []model.AuditLogV2{}

	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAuditLogServiceClient(conn)
		var gapiAuditLog *gapi.AuditLog

		stream, err := client.QueryAuditLogs(ctx, request)
		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryAuditLogs. Error: %s"), err.Error())
			return err
		}

		for {
			gapiAuditLog, err = stream.Recv()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				err = errcode.NewInternalError(err.Error())
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryAuditLogs. Error: %s"), err.Error())
				return err
			}
			modelAuditlog := model.AuditLogV2{}
			err = base.Convert(gapiAuditLog, &modelAuditlog)
			if err != nil {
				glog.Errorf(base.PrefixRequestID(ctx, "Error in QueryAuditLogs. Error: %s"), err.Error())
				return err
			}
			modelAuditLogs = append(modelAuditLogs, modelAuditlog)
		}
		return nil
	}
	err = service.CallClient(ctx, service.AuditLogService, handler)
	return modelAuditLogs, err
}

func (objectModelAPI *dbObjectModelAPI) QueryAuditLogsV2W(defaultContext context.Context, w io.Writer, r *http.Request) error {
	reader := io.Reader(r.Body)
	_, err := base.GetAuthContext(defaultContext)
	if err != nil {
		return err
	}
	doc := model.AuditLogV2Filter{}
	err = base.Decode(&reader, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(defaultContext, "Error decoding into audit log v2 filter. Error: %s"), err.Error())
		return err
	}
	pageQueryParam := model.GetEntitiesQueryParam(r).PageQueryParam

	if pageQueryParam.PageIndex != 0 {
		doc.FromDocument = pageQueryParam.PageIndex
	}
	if pageQueryParam.PageSize != base.MaxRowsLimit {
		doc.PageSize = pageQueryParam.PageSize
	}
	auditLogs, err := objectModelAPI.QueryAuditLogsV2(defaultContext, doc)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, auditLogs)
}

func (objectModelAPI *dbObjectModelAPI) InsertAuditLogV2(ctx context.Context, req model.AuditLogV2InsertRequest) (string, error) {
	var insertAuditLogResponse *gapi.InsertAuditLogResponse
	gapiAuditLog := gapi.AuditLog{}
	glog.V(3).Infoln("req: ", req)
	err := base.Convert(req.AuditLog, &gapiAuditLog)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in converting go model to grpc request model. Error: %s"), err.Error())
		return "", err
	}
	request := &gapi.InsertAuditLogRequest{Auditlog: &gapiAuditLog}
	handler := func(ctx context.Context, conn *grpc.ClientConn) error {
		client := gapi.NewAuditLogServiceClient(conn)

		insertAuditLogResponse, err = client.InsertAuditLog(ctx, request)

		if err != nil {
			glog.Errorf(base.PrefixRequestID(ctx, "Error in InsertAuditLog. Error: %s"), err.Error())
			return err
		}
		return nil
	}
	err = service.CallClient(ctx, service.AuditLogService, handler)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(ctx, "Error in callClient. Error : %s"), err.Error())
		return "", err
	}
	return insertAuditLogResponse.DocumentID, nil
}

func (objectModelAPI *dbObjectModelAPI) InsertAuditLogV2W(defaultContext context.Context, w io.Writer, r *http.Request) error {
	reader := io.Reader(r.Body)
	_, err := base.GetAuthContext(defaultContext)
	if err != nil {
		return err
	}
	//glog.V(3).Infoln("InsertAuditLogV2W: authContext : ", authContext)
	doc := model.AuditLogV2InsertRequest{}
	err = base.Decode(&reader, &doc)
	if err != nil {
		glog.Errorf(base.PrefixRequestID(defaultContext, "Error decoding into audit log v2 insert request. Error: %s"), err.Error())
		return err
	}

	auditLogID, err := objectModelAPI.InsertAuditLogV2(defaultContext, doc)
	if err != nil {
		return err
	}
	return base.DispatchPayload(w, auditLogID)
}

package service

import (
	"cloudservices/common/base"
	"cloudservices/common/errcode"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorFromHTTPStatus converts errcode.ErrorCode to grpc error.
func ErrorFromHTTPStatus(err error) error {
	if err == nil {
		return err
	}
	errCode, ok := err.(errcode.ErrorCode)
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}
	httpStatus := errCode.GetHTTPStatus()
	data, err1 := base.ConvertToJSON(errCode)
	if err1 != nil {
		return status.Error(codes.Internal, err.Error())
	}
	errorMsg := string(data)
	switch httpStatus {
	case http.StatusOK:
		return nil
	case http.StatusForbidden:
		return status.Error(codes.PermissionDenied, errorMsg)
	case http.StatusUnauthorized:
		return status.Error(codes.Unauthenticated, errorMsg)
	case http.StatusNotImplemented:
		return status.Error(codes.Unimplemented, errorMsg)
	case http.StatusNotFound:
		return status.Error(codes.NotFound, errorMsg)
	case http.StatusBadRequest:
		return status.Error(codes.InvalidArgument, errorMsg)
	}
	return status.Error(codes.Internal, errorMsg)
}

// ErrorCodeFromError converts grpc error to errcode.ErrorCode.
func ErrorCodeFromError(err error) error {
	if err == nil {
		return err
	}
	sts, ok := status.FromError(err)
	if !ok {
		return err
	}
	errorMsg := sts.Message()
	errCode, err1 := errcode.UnmarshalJSON([]byte(errorMsg))
	if err1 != nil {
		return err
	}
	return errCode
}

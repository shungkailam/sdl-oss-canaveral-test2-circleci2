package service

import (
	"cloudservices/common/base"
	"context"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type customServerStream struct {
	ctx context.Context
	grpc.ServerStream
}

func (stream *customServerStream) Context() context.Context {
	return stream.ctx
}

func enrichServerContext(ctx context.Context) (context.Context, error) {
	var reqID string
	var authContext base.AuthContext
	var values []string
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		values, ok = md[string(base.RequestIDKey)]
		reqID = values[0]
		values, ok = md[string(base.AuthContextKey)]
		if ok {
			err := base.ConvertFromJSON([]byte(values[0]), &authContext)
			if err != nil {
				return nil, err
			}
			ctx = context.WithValue(ctx, base.AuthContextKey, &authContext)
		}
	}
	if !ok {
		reqID = base.GetUUID()
	}
	ctx = context.WithValue(ctx, base.RequestIDKey, reqID)
	return ctx, nil
}

func enrichClientContext(ctx context.Context) (context.Context, error) {
	md := metadata.MD{}
	reqID := base.GetRequestID(ctx)
	md[string(base.RequestIDKey)] = append(md[string(base.RequestIDKey)], reqID)
	authContext, err := base.GetAuthContext(ctx)
	if err == nil {
		data, err := base.ConvertToJSON(authContext)
		if err != nil {
			return nil, err
		}
		md[string(base.AuthContextKey)] = append(md[string(base.AuthContextKey)], string(data))
	}
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, nil
}

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, err := enrichServerContext(ctx)
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof(base.PrefixRequestID(ctx, "Invoking method: %s"), info.FullMethod)
		start := time.Now()
		defer func() {
			stop := time.Since(start)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			glog.V(4).Infof(base.PrefixRequestID(ctx, "Completed for method: %s in %.2f ms. Error: %s"), info.FullMethod, float32(stop/time.Millisecond), errMsg)
		}()
		intf, err := handler(ctx, req)
		return intf, ErrorFromHTTPStatus(err)
	}
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx, err := enrichServerContext(stream.Context())
		if err != nil {
			return err
		}
		glog.V(4).Infof(base.PrefixRequestID(ctx, "Invoking method: %s"), info.FullMethod)
		start := time.Now()
		defer func() {
			stop := time.Since(start)
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			glog.V(4).Infof(base.PrefixRequestID(ctx, "Completed for method: %s in %.2f ms. Error: %s"), info.FullMethod, float32(stop/time.Millisecond), errMsg)
		}()
		err = handler(srv, &customServerStream{ServerStream: stream, ctx: ctx})
		return ErrorFromHTTPStatus(err)
	}
}

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx, err := enrichClientContext(ctx)
		if err != nil {
			return err
		}
		glog.V(4).Infof(base.PrefixRequestID(ctx, "Invoking method: %s"), method)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func StreamClientInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx, err := enrichClientContext(ctx)
		if err != nil {
			return nil, err
		}
		glog.V(4).Infof(base.PrefixRequestID(ctx, "Invoking method: %s"), method)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

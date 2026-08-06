package hostname

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	logField = "hostname"
)

type ServerInterceptor struct{}

func NewServerInterceptor() *ServerInterceptor {
	return &ServerInterceptor{}
}

func (i *ServerInterceptor) Unary() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		return handler(wrapHostname(ctx), req)
	}
}

func (i *ServerInterceptor) Stream() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		return handler(srv, &hostnameServerStream{
			ServerStream: ss,
			ctx:          wrapHostname(ss.Context()),
		})
	}
}

type hostnameServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *hostnameServerStream) Context() context.Context {
	return s.ctx
}

func wrapHostname(ctx context.Context) context.Context {
	host := hostnameFromMetadata(ctx)
	if host == "" {
		return ctx
	}
	return xlog.WrapContext(ctx, log.String(logField, host))
}

func hostnameFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(Header)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

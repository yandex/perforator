package nopauth

import (
	"context"
	"net/http"

	"google.golang.org/grpc"

	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
)

func NewProvider() auth.Provider {
	return &provider{}
}

type provider struct{}

func (p provider) GRPC(ignoredMethods []string) auth.GRPCInterceptor {
	return &grpcInterceptor{}
}

func (p provider) HTTP() func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return h
	}
}

type grpcInterceptor struct{}

func (i grpcInterceptor) UnaryServer() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
}

func (i grpcInterceptor) StreamServer() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}

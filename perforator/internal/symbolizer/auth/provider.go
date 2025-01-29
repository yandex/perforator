package auth

import (
	"net/http"

	"google.golang.org/grpc"
)

type Provider interface {
	GRPC(skipMethods []string) GRPCInterceptor
	HTTP() func(http.Handler) http.Handler
}

type GRPCInterceptor interface {
	UnaryServer() grpc.UnaryServerInterceptor
	StreamServer() grpc.StreamServerInterceptor
}

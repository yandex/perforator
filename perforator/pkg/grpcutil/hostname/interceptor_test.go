package hostname

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/yandex/perforator/library/go/core/log/ctxlog"
)

func TestUnaryInterceptorAddsHostname(t *testing.T) {
	interceptor := NewServerInterceptor().Unary()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(Header, "host-1"))

	var got context.Context
	_, err := interceptor(ctx, nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
		got = ctx
		return nil, nil
	})
	require.NoError(t, err)
	require.Equal(t, "host-1", hostnameField(got))
}

func TestUnaryInterceptorMissingHeader(t *testing.T) {
	interceptor := NewServerInterceptor().Unary()

	called := false
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		require.Empty(t, hostnameField(ctx))
		return nil, nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

func TestStreamInterceptorAddsHostname(t *testing.T) {
	interceptor := NewServerInterceptor().Stream()
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(Header, "host-2"))
	ss := &fakeServerStream{ctx: ctx}

	var got context.Context
	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(srv interface{}, stream grpc.ServerStream) error {
		got = stream.Context()
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, "host-2", hostnameField(got))
}

func TestStreamInterceptorMissingHeader(t *testing.T) {
	interceptor := NewServerInterceptor().Stream()
	ss := &fakeServerStream{ctx: context.Background()}

	called := false
	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(srv interface{}, stream grpc.ServerStream) error {
		called = true
		require.Empty(t, hostnameField(stream.Context()))
		return nil
	})
	require.NoError(t, err)
	require.True(t, called)
}

type fakeServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *fakeServerStream) Context() context.Context {
	return s.ctx
}

func hostnameField(ctx context.Context) string {
	for _, field := range ctxlog.ContextFields(ctx) {
		if field.Key() == logField {
			return field.String()
		}
	}
	return ""
}

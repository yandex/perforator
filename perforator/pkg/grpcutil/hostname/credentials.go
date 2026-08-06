package hostname

import (
	"context"

	"google.golang.org/grpc/credentials"
)

const (
	Header = "x-perforator-hostname"
)

type hostCredentials struct {
	hostname string
}

func NewCredentials(hostname string) credentials.PerRPCCredentials {
	return &hostCredentials{hostname: hostname}
}

func (c *hostCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		Header: c.hostname,
	}, nil
}

func (c *hostCredentials) RequireTransportSecurity() bool {
	return false
}

var _ credentials.PerRPCCredentials = (*hostCredentials)(nil)

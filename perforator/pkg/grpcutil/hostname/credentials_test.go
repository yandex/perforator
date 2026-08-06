package hostname

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCredentialsGetRequestMetadata(t *testing.T) {
	creds := NewCredentials("host.example.net")

	md, err := creds.GetRequestMetadata(context.Background())
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		Header: "host.example.net",
	}, md)
}

func TestCredentialsRequireTransportSecurity(t *testing.T) {
	creds := NewCredentials("host")
	require.False(t, creds.RequireTransportSecurity())
}

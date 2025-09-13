package httpmock_test

import (
	"net/http"
	"os"
	"testing"

	"github.com/maxatome/go-testdeep/td"

	"github.com/jarcoal/httpmock"
)

const envVarName = "GONOMOCKS"

func TestEnv(t *testing.T) {
	require := td.Require(t)

	httpmock.DeactivateAndReset()

	defer func(orig string) {
		require.CmpNoError(os.Setenv(envVarName, orig))
	}(os.Getenv(envVarName))

	// put it in an enabled state
	require.CmpNoError(os.Setenv(envVarName, ""))
	require.False(httpmock.Disabled(), "expected not to be disabled")

	client1 := &http.Client{Transport: &http.Transport{}}
	client2 := &http.Client{Transport: &http.Transport{}}

	// make sure an activation works
	httpmock.Activate()
	httpmock.ActivateNonDefault(client1)
	httpmock.ActivateNonDefault(client2)
	require.Cmp(http.DefaultTransport, httpmock.DefaultTransport,
		"expected http.DefaultTransport to be our DefaultTransport")
	require.Cmp(client1.Transport, httpmock.DefaultTransport,
		"expected client1.Transport to be our DefaultTransport")
	require.Cmp(client2.Transport, httpmock.DefaultTransport,
		"expected client2.Transport to be our DefaultTransport")
	httpmock.Deactivate()

	require.CmpNoError(os.Setenv(envVarName, "1"))
	require.True(httpmock.Disabled(), "expected to be disabled")

	// make sure activation doesn't work
	httpmock.Activate()
	httpmock.ActivateNonDefault(client1)
	httpmock.ActivateNonDefault(client2)
	require.Not(http.DefaultTransport, httpmock.DefaultTransport,
		"expected http.DefaultTransport to not be our DefaultTransport")
	require.Not(client1.Transport, httpmock.DefaultTransport,
		"expected client1.Transport to not be our DefaultTransport")
	require.Not(client2.Transport, httpmock.DefaultTransport,
		"expected client2.Transport to not be our DefaultTransport")
	httpmock.Deactivate()
}

package debuginfod_test

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/debuginfod"
)

func TestDebuginfodClient(t *testing.T) {
	ctx := context.Background()

	t.Run("simple", func(t *testing.T) {
		transport := makeMockHTTPTransport()
		client, err := makeTestClient(transport, debuginfod.WithURL("http://debuginfod.localhost"))
		require.NoError(t, err)
		require.NotNil(t, client)

		w := bytes.Buffer{}

		count, err := client.FetchDebugInfo(ctx, "unknownbuildid", &w)
		require.Error(t, err)
		require.Zero(t, count)

		w.Reset()
		count, err = client.FetchDebugInfo(ctx, "knownbuildid", &w)
		require.NoError(t, err)
		require.NotZero(t, count)
		require.Equal(t, "DEBUGINFO", w.String())

		w.Reset()
		count, err = client.FetchExecutable(ctx, "knownbuildid", &w)
		require.NoError(t, err)
		require.NotZero(t, count)
		require.Equal(t, "EXE", w.String())

		w.Reset()
		count, err = client.FetchSection(ctx, "knownbuildid", "testscn", &w)
		require.NoError(t, err)
		require.NotZero(t, count)
		require.Equal(t, "TESTSCN", w.String())

		w.Reset()
		count, err = client.FetchSection(ctx, "knownbuildid", "unknownscn", &w)
		require.Error(t, err)
		require.Zero(t, count)
	})

	t.Run("federation", func(t *testing.T) {
		transport := makeMockHTTPTransport()
		client, err := makeTestClient(transport,
			debuginfod.WithURLs(
				"http://broken.debuginfod.localhost",
				"http://empty.debuginfod.localhost",
				"http://offline.debuginfod.localhost",
				"http://estonian.debuginfod.localhost",
				"http://valid.debuginfod.localhost",
			),
			debuginfod.WithTimeout(time.Second),
		)
		require.NoError(t, err)

		w := bytes.Buffer{}

		count, err := client.FetchDebugInfo(ctx, "unknownbuildid", &w)
		require.Error(t, err)
		require.Zero(t, count)

		count, err = client.FetchDebugInfo(ctx, "knownbuildid", &w)
		require.NoError(t, err)
		require.Equal(t, "DEBUGINFO", w.String())
		require.NotZero(t, count)
	})

	t.Run("env", func(t *testing.T) {
		transport := makeMockHTTPTransport()

		err := os.Setenv("DEBUGINFOD_URLS", "http://valid.debuginfod.localhost,http://broken.debuginfod.localhost")
		require.NoError(t, err)

		client, err := makeTestClient(transport, debuginfod.WithEnvConfig())
		require.NoError(t, err)

		w := bytes.Buffer{}

		count, err := client.FetchDebugInfo(ctx, "unknownbuildid", &w)
		require.Error(t, err)
		require.Zero(t, count)

		count, err = client.FetchDebugInfo(ctx, "knownbuildid", &w)
		require.NoError(t, err)
		require.NotZero(t, count)
		require.Equal(t, "DEBUGINFO", w.String())
	})

	t.Run("nourls", func(t *testing.T) {
		transport := makeMockHTTPTransport()
		client, err := makeTestClient(transport)
		require.ErrorIs(t, err, debuginfod.ErrNoEndpoints)
		require.Nil(t, client)
	})
}

func makeMockHTTPTransport() *httpmock.MockTransport {
	transport := httpmock.NewMockTransport()

	// "http://offline.debuginfod.localhost"
	transport.RegisterNoResponder(httpmock.ConnectionFailure)

	// "http://empty.debuginfod.localhost"
	transport.RegisterRegexpResponder(
		"GET", regexp.MustCompile("http://empty.debuginfod.localhost/.*"),
		httpmock.NewStringResponder(http.StatusNotFound, "Not found"),
	)

	// "http://broken.debuginfod.localhost"
	transport.RegisterRegexpResponder(
		"GET", regexp.MustCompile("http://broken.debuginfod.localhost/.*"),
		httpmock.NewStringResponder(http.StatusInternalServerError, "Internal server error"),
	)

	// "http://estonian.debuginfod.localhost"
	transport.RegisterRegexpResponder(
		"GET", regexp.MustCompile("http://estonian.debuginfod.localhost/.*"),
		makeTimeoutingResponder(time.Second*10, httpmock.NewStringResponder(http.StatusInternalServerError, "Internal server error")),
	)

	for _, url := range []string{"http://debuginfod.localhost", "http://valid.debuginfod.localhost"} {
		transport.RegisterResponder(
			"GET", url+"/buildid/knownbuildid/debuginfo",
			httpmock.NewStringResponder(http.StatusOK, "DEBUGINFO"),
		)
		transport.RegisterResponder(
			"GET", url+"/buildid/knownbuildid/executable",
			httpmock.NewStringResponder(http.StatusOK, "EXE"),
		)
		transport.RegisterResponder(
			"GET", url+"/buildid/knownbuildid/section/testscn",
			httpmock.NewStringResponder(http.StatusOK, "TESTSCN"),
		)
		transport.RegisterRegexpResponder(
			"GET", regexp.MustCompile(url+"/buildid/.*/executable"),
			httpmock.NewStringResponder(http.StatusNotFound, "not found"),
		)
	}

	return transport
}

func makeTestClient(transport *httpmock.MockTransport, opts ...debuginfod.ClientOption) (*debuginfod.Client, error) {
	opts = append(opts, debuginfod.WithHTTPClientOption(func(c *resty.Client) error {
		c.SetTransport(transport)
		return nil
	}))

	return debuginfod.NewClient(opts...)
}

func makeTimeoutingResponder(duration time.Duration, responder httpmock.Responder) httpmock.Responder {
	return func(r *http.Request) (*http.Response, error) {
		time.Sleep(duration)
		return responder(r)
	}
}

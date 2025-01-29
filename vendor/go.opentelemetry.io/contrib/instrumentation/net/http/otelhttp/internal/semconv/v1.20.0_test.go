// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package semconv

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"go.opentelemetry.io/otel/attribute"
)

func TestV120TraceRequest(t *testing.T) {
	// Anything but "http" or "http/dup" works.
	t.Setenv("OTEL_SEMCONV_STABILITY_OPT_IN", "old")
	serv := NewHTTPServer(nil)
	want := func(req testServerReq) []attribute.KeyValue {
		return []attribute.KeyValue{
			attribute.String("http.method", "GET"),
			attribute.String("http.scheme", "http"),
			attribute.String("net.host.name", req.hostname),
			attribute.Int("net.host.port", req.serverPort),
			attribute.String("net.sock.peer.addr", req.peerAddr),
			attribute.Int("net.sock.peer.port", req.peerPort),
			attribute.String("user_agent.original", "Go-http-client/1.1"),
			attribute.String("http.client_ip", req.clientIP),
			attribute.String("net.protocol.version", "1.1"),
			attribute.String("http.target", "/"),
		}
	}
	testTraceRequest(t, serv, want)
}

func TestV120TraceResponse(t *testing.T) {
	testCases := []struct {
		name string
		resp ResponseTelemetry
		want []attribute.KeyValue
	}{
		{
			name: "empty",
			resp: ResponseTelemetry{},
			want: nil,
		},
		{
			name: "no errors",
			resp: ResponseTelemetry{
				StatusCode: 200,
				ReadBytes:  701,
				WriteBytes: 802,
			},
			want: []attribute.KeyValue{
				attribute.Int("http.request_content_length", 701),
				attribute.Int("http.response_content_length", 802),
				attribute.Int("http.status_code", 200),
			},
		},
		{
			name: "with errors",
			resp: ResponseTelemetry{
				StatusCode: 200,
				ReadBytes:  701,
				ReadError:  fmt.Errorf("read error"),
				WriteBytes: 802,
				WriteError: fmt.Errorf("write error"),
			},
			want: []attribute.KeyValue{
				attribute.Int("http.request_content_length", 701),
				attribute.String("http.read_error", "read error"),
				attribute.Int("http.response_content_length", 802),
				attribute.String("http.write_error", "write error"),
				attribute.Int("http.status_code", 200),
			},
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := oldHTTPServer{}.ResponseTraceAttrs(tt.resp)
			assert.ElementsMatch(t, tt.want, got)
		})
	}
}

func TestV120RecordMetrics(t *testing.T) {
	server := NewTestHTTPServer()
	req, err := http.NewRequest("POST", "http://example.com", nil)
	assert.NoError(t, err)

	server.RecordMetrics(context.Background(), ServerMetricData{
		ServerName:   "stuff",
		ResponseSize: 200,
		MetricAttributes: MetricAttributes{
			Req:        req,
			StatusCode: 301,
			AdditionalAttributes: []attribute.KeyValue{
				attribute.String("key", "value"),
			},
		},
		MetricData: MetricData{
			RequestSize: 100,
			ElapsedTime: 300,
		},
	})

	assert.Equal(t, int64(100), server.requestBytesCounter.(*testInst).intValue)
	assert.Equal(t, int64(200), server.responseBytesCounter.(*testInst).intValue)
	assert.Equal(t, float64(300), server.serverLatencyMeasure.(*testInst).floatValue)

	want := []attribute.KeyValue{
		attribute.String("http.scheme", "http"),
		attribute.String("http.method", "POST"),
		attribute.Int64("http.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("net.host.name", "stuff"),
		attribute.String("net.protocol.name", "http"),
		attribute.String("net.protocol.version", "1.1"),
	}

	assert.ElementsMatch(t, want, server.requestBytesCounter.(*testInst).attributes)
	assert.ElementsMatch(t, want, server.responseBytesCounter.(*testInst).attributes)
	assert.ElementsMatch(t, want, server.serverLatencyMeasure.(*testInst).attributes)
}

func TestV120ClientRequest(t *testing.T) {
	body := strings.NewReader("Hello, world!")
	url := "https://example.com:8888/foo/bar?stuff=morestuff"
	req, err := http.NewRequest("POST", url, body)
	assert.NoError(t, err)
	req.Header.Set("User-Agent", "go-test-agent")

	want := []attribute.KeyValue{
		attribute.String("http.method", "POST"),
		attribute.String("http.url", url),
		attribute.String("net.peer.name", "example.com"),
		attribute.Int("net.peer.port", 8888),
		attribute.Int("http.request_content_length", body.Len()),
		attribute.String("user_agent.original", "go-test-agent"),
	}
	got := oldHTTPClient{}.RequestTraceAttrs(req)
	assert.ElementsMatch(t, want, got)
}

func TestV120ClientResponse(t *testing.T) {
	resp := http.Response{
		StatusCode:    200,
		ContentLength: 123,
	}

	want := []attribute.KeyValue{
		attribute.Int("http.response_content_length", 123),
		attribute.Int("http.status_code", 200),
	}

	got := oldHTTPClient{}.ResponseTraceAttrs(&resp)
	assert.ElementsMatch(t, want, got)
}

func TestV120ClientMetrics(t *testing.T) {
	client := NewTestHTTPClient()
	req, err := http.NewRequest("POST", "http://example.com", nil)
	assert.NoError(t, err)

	opts := client.MetricOptions(MetricAttributes{
		Req:        req,
		StatusCode: 301,
		AdditionalAttributes: []attribute.KeyValue{
			attribute.String("key", "value"),
		},
	})

	ctx := context.Background()

	client.RecordResponseSize(ctx, 200, opts.AddOptions())

	client.RecordMetrics(ctx, MetricData{
		RequestSize: 100,
		ElapsedTime: 300,
	}, opts)

	assert.Equal(t, int64(100), client.requestBytesCounter.(*testInst).intValue)
	assert.Equal(t, int64(200), client.responseBytesCounter.(*testInst).intValue)
	assert.Equal(t, float64(300), client.latencyMeasure.(*testInst).floatValue)

	want := []attribute.KeyValue{
		attribute.String("http.method", "POST"),
		attribute.Int64("http.status_code", 301),
		attribute.String("key", "value"),
		attribute.String("net.peer.name", "example.com"),
	}

	assert.ElementsMatch(t, want, client.requestBytesCounter.(*testInst).attributes)
	assert.ElementsMatch(t, want, client.responseBytesCounter.(*testInst).attributes)
	assert.ElementsMatch(t, want, client.latencyMeasure.(*testInst).attributes)
}

func TestStandardizeHTTPMethodMetric(t *testing.T) {
	testCases := []struct {
		method string
		want   attribute.KeyValue
	}{
		{
			method: "GET",
			want:   attribute.String("http.method", "GET"),
		},
		{
			method: "POST",
			want:   attribute.String("http.method", "POST"),
		},
		{
			method: "PUT",
			want:   attribute.String("http.method", "PUT"),
		},
		{
			method: "DELETE",
			want:   attribute.String("http.method", "DELETE"),
		},
		{
			method: "HEAD",
			want:   attribute.String("http.method", "HEAD"),
		},
		{
			method: "OPTIONS",
			want:   attribute.String("http.method", "OPTIONS"),
		},
		{
			method: "CONNECT",
			want:   attribute.String("http.method", "CONNECT"),
		},
		{
			method: "TRACE",
			want:   attribute.String("http.method", "TRACE"),
		},
		{
			method: "PATCH",
			want:   attribute.String("http.method", "PATCH"),
		},
		{
			method: "test",
			want:   attribute.String("http.method", "_OTHER"),
		},
	}
	for _, tt := range testCases {
		t.Run(tt.method, func(t *testing.T) {
			got := standardizeHTTPMethodMetric(tt.method)
			assert.Equal(t, tt.want, got)
		})
	}
}

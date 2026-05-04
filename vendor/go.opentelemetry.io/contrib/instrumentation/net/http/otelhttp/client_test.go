// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestConvenienceWrappers(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))
	orig := otelhttp.DefaultClient
	otelhttp.DefaultClient = &http.Client{
		Transport: otelhttp.NewTransport(
			http.DefaultTransport,
			otelhttp.WithTracerProvider(provider),
		),
	}
	defer func() { otelhttp.DefaultClient = orig }()

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	ctx := context.Background()
	res, err := otelhttp.Get(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = otelhttp.Head(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	res, err = otelhttp.Post(ctx, ts.URL, "text/plain", strings.NewReader("test"))
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	form := make(url.Values)
	form.Set("foo", "bar")
	res, err = otelhttp.PostForm(ctx, ts.URL, form)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	spans := sr.Ended()
	require.Len(t, spans, 4)
	assert.Equal(t, "HTTP GET", spans[0].Name())
	assert.Equal(t, "HTTP HEAD", spans[1].Name())
	assert.Equal(t, "HTTP POST", spans[2].Name())
	assert.Equal(t, "HTTP POST", spans[3].Name())
}

func TestClientWithTraceContext(t *testing.T) {
	sr := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(sr))

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "http requests")

	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	res, err := otelhttp.Get(ctx, ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()

	span.End()

	spans := sr.Ended()
	require.Len(t, spans, 2)
	assert.Equal(t, "HTTP GET", spans[0].Name())
	assert.Equal(t, "http requests", spans[1].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[0].Parent().SpanID())
}

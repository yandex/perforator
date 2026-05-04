// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/instrumentation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/metric/metricdata/metricdatatest"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestTransportFormatter(t *testing.T) {
	httpMethods := []struct {
		name     string
		method   string
		expected string
	}{
		{
			"GET method",
			http.MethodGet,
			"HTTP GET",
		},
		{
			"HEAD method",
			http.MethodHead,
			"HTTP HEAD",
		},
		{
			"POST method",
			http.MethodPost,
			"HTTP POST",
		},
		{
			"PUT method",
			http.MethodPut,
			"HTTP PUT",
		},
		{
			"PATCH method",
			http.MethodPatch,
			"HTTP PATCH",
		},
		{
			"DELETE method",
			http.MethodDelete,
			"HTTP DELETE",
		},
		{
			"CONNECT method",
			http.MethodConnect,
			"HTTP CONNECT",
		},
		{
			"OPTIONS method",
			http.MethodOptions,
			"HTTP OPTIONS",
		},
		{
			"TRACE method",
			http.MethodTrace,
			"HTTP TRACE",
		},
	}

	for _, tc := range httpMethods {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(tc.method, "http://localhost/", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			formattedName := "HTTP " + r.Method

			if formattedName != tc.expected {
				t.Fatalf("unexpected name: got %s, expected %s", formattedName, tc.expected)
			}
		})
	}
}

func TestTransportBasics(t *testing.T) {
	prop := propagation.TraceContext{}
	content := []byte("Hello, world!")

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if span.SpanID() != sc.SpanID() {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), sc.SpanID())
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(http.DefaultTransport, WithPropagators(prop))

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected content: got %s, expected %s", body, content)
	}
}

func TestNilTransport(t *testing.T) {
	prop := propagation.TraceContext{}
	content := []byte("Hello, world!")

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if span.SpanID() != sc.SpanID() {
			t.Fatalf("testing remote SpanID: got %s, expected %s", span.SpanID(), sc.SpanID())
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(nil, WithPropagators(prop))

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Errorf("close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(body, content) {
		t.Fatalf("unexpected content: got %s, expected %s", body, content)
	}
}

const readSize = 42

type readCloser struct {
	readErr, closeErr error
}

func (rc readCloser) Read([]byte) (n int, err error) {
	return readSize, rc.readErr
}

func (rc readCloser) Close() error {
	return rc.closeErr
}

type span struct {
	trace.Span

	ended       bool
	recordedErr error

	statusCode codes.Code
	statusDesc string
}

func (s *span) End(...trace.SpanEndOption) {
	s.ended = true
}

func (s *span) RecordError(err error, _ ...trace.EventOption) {
	s.recordedErr = err
}

func (s *span) SetStatus(c codes.Code, d string) {
	s.statusCode, s.statusDesc = c, d
}

func (s *span) assert(t *testing.T, ended bool, err error, c codes.Code, d string) { //nolint:revive  // ended is not a control flag.
	if ended {
		assert.True(t, s.ended, "not ended")
	} else {
		assert.False(t, s.ended, "ended")
	}

	if err == nil {
		assert.NoError(t, s.recordedErr, "recorded an error")
	} else {
		assert.Equal(t, err, s.recordedErr)
	}

	assert.Equal(t, c, s.statusCode, "status codes not equal")
	assert.Equal(t, d, s.statusDesc, "status description not equal")
}

func TestWrappedBodyRead(t *testing.T) {
	s := new(span)
	called := false
	record := func(int64) { called = true }
	wb := newWrappedBody(s, record, readCloser{})
	n, err := wb.Read([]byte{})
	assert.Equal(t, readSize, n, "wrappedBody returned wrong bytes")
	assert.NoError(t, err)
	s.assert(t, false, nil, codes.Unset, "")
	assert.False(t, called, "record should not have been called")
}

func TestWrappedBodyReadEOFError(t *testing.T) {
	s := new(span)
	called := false
	numRecorded := int64(0)
	record := func(numBytes int64) {
		called = true
		numRecorded = numBytes
	}
	wb := newWrappedBody(s, record, readCloser{readErr: io.EOF})
	n, err := wb.Read([]byte{})
	assert.Equal(t, readSize, n, "wrappedBody returned wrong bytes")
	assert.Equal(t, io.EOF, err)
	s.assert(t, true, nil, codes.Unset, "")
	assert.True(t, called, "record should have been called")
	assert.Equal(t, int64(readSize), numRecorded, "record recorded wrong number of bytes")
}

func TestWrappedBodyReadError(t *testing.T) {
	s := new(span)
	called := false
	record := func(int64) { called = true }
	expectedErr := errors.New("test")
	wb := newWrappedBody(s, record, readCloser{readErr: expectedErr})
	n, err := wb.Read([]byte{})
	assert.Equal(t, readSize, n, "wrappedBody returned wrong bytes")
	assert.Equal(t, expectedErr, err)
	s.assert(t, false, expectedErr, codes.Error, expectedErr.Error())
	assert.False(t, called, "record should not have been called")
}

func TestWrappedBodyClose(t *testing.T) {
	s := new(span)
	called := false
	record := func(int64) { called = true }
	wb := newWrappedBody(s, record, readCloser{})
	assert.NoError(t, wb.Close())
	s.assert(t, true, nil, codes.Unset, "")
	assert.True(t, called, "record should have been called")
}

func TestWrappedBodyClosePanic(t *testing.T) {
	s := new(span)
	var body io.ReadCloser
	wb := newWrappedBody(s, func(int64) {}, body)
	assert.NotPanics(t, func() { wb.Close() }, "nil body should not panic on close")
}

func TestWrappedBodyCloseError(t *testing.T) {
	s := new(span)
	called := false
	record := func(int64) { called = true }
	expectedErr := errors.New("test")
	wb := newWrappedBody(s, record, readCloser{closeErr: expectedErr})
	assert.Equal(t, expectedErr, wb.Close())
	s.assert(t, true, nil, codes.Unset, "")
	assert.True(t, called, "record should have been called")
}

type readWriteCloser struct {
	readCloser

	writeErr error
}

const writeSize = 1

func (rwc readWriteCloser) Write([]byte) (int, error) {
	return writeSize, rwc.writeErr
}

func TestNewWrappedBodyReadWriteCloserImplementation(t *testing.T) {
	wb := newWrappedBody(nil, func(int64) {}, readWriteCloser{})
	assert.Implements(t, (*io.ReadWriteCloser)(nil), wb)
}

func TestNewWrappedBodyReadCloserImplementation(t *testing.T) {
	wb := newWrappedBody(nil, func(int64) {}, readCloser{})
	assert.Implements(t, (*io.ReadCloser)(nil), wb)

	_, ok := wb.(io.ReadWriteCloser)
	assert.False(t, ok, "wrappedBody should not implement io.ReadWriteCloser")
}

func TestWrappedBodyWrite(t *testing.T) {
	s := new(span)
	var rwc io.ReadWriteCloser
	assert.NotPanics(t, func() {
		rwc = newWrappedBody(s, func(int64) {}, readWriteCloser{}).(io.ReadWriteCloser)
	})

	n, err := rwc.Write([]byte{})
	assert.Equal(t, writeSize, n, "wrappedBody returned wrong bytes")
	assert.NoError(t, err)
	s.assert(t, false, nil, codes.Unset, "")
}

func TestWrappedBodyWriteError(t *testing.T) {
	s := new(span)
	expectedErr := errors.New("test")
	var rwc io.ReadWriteCloser
	assert.NotPanics(t, func() {
		rwc = newWrappedBody(s,
			func(int64) {},
			readWriteCloser{
				writeErr: expectedErr,
			}).(io.ReadWriteCloser)
	})
	n, err := rwc.Write([]byte{})
	assert.Equal(t, writeSize, n, "wrappedBody returned wrong bytes")
	assert.ErrorIs(t, err, expectedErr)
	s.assert(t, false, expectedErr, codes.Error, expectedErr.Error())
}

func TestTransportProtocolSwitch(t *testing.T) {
	// This test validates the fix to #1329.

	// Simulate a "101 Switching Protocols" response from the test server.
	response := []byte(strings.Join([]string{
		"HTTP/1.1 101 Switching Protocols",
		"Upgrade: WebSocket",
		"Connection: Upgrade",
		"", "", // Needed for extra CRLF.
	}, "\r\n"))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		conn, buf, err := w.(http.Hijacker).Hijack()
		assert.NoError(t, err)

		_, err = buf.Write(response)
		assert.NoError(t, err)
		assert.NoError(t, buf.Flush())
		assert.NoError(t, conn.Close())
	}))
	defer ts.Close()

	ctx := context.Background()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)

	c := http.Client{Transport: NewTransport(http.DefaultTransport)}
	res, err := c.Do(r)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, res.Body.Close()) })

	assert.Implements(t, (*io.ReadWriteCloser)(nil), res.Body, "invalid body returned for protocol switch")
}

func TestTransportOriginRequestNotModify(t *testing.T) {
	prop := propagation.TraceContext{}

	ctx := context.Background()
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
	})
	ctx = trace.ContextWithRemoteSpanContext(ctx, sc)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)

	expectedRequest := r.Clone(r.Context())

	c := http.Client{Transport: NewTransport(http.DefaultTransport, WithPropagators(prop))}
	res, err := c.Do(r)
	require.NoError(t, err)

	t.Cleanup(func() { require.NoError(t, res.Body.Close()) })

	assert.Equal(t, expectedRequest, r)
}

func TestTransportUsesFormatter(t *testing.T) {
	prop := propagation.TraceContext{}
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := prop.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		span := trace.SpanContextFromContext(ctx)
		if !span.IsValid() {
			t.Fatalf("invalid span wrapping handler: %#v", span)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}

	tr := NewTransport(
		http.DefaultTransport,
		WithTracerProvider(provider),
		WithPropagators(prop),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	spans := spanRecorder.Ended()
	require.NotEmpty(t, spans)
	spanName := spans[0].Name()
	expectedName := "HTTP GET"
	if spanName != expectedName {
		t.Fatalf("unexpected name: got %s, expected %s", spanName, expectedName)
	}
}

func TestTransportErrorStatus(t *testing.T) {
	// Prepare tracing stuff.
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	// Run a server and stop to make sure nothing is listening and force the error.
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()

	// Create our Transport and make request.
	tr := NewTransport(
		http.DefaultTransport,
		WithTracerProvider(provider),
	)
	c := http.Client{Transport: tr}
	r, err := http.NewRequest(http.MethodGet, server.URL, http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := c.Do(r)
	if err == nil {
		if e := resp.Body.Close(); e != nil {
			t.Errorf("close response body: %v", e)
		}
		t.Fatal("transport should have returned an error, it didn't")
	}

	// Check span.
	spans := spanRecorder.Ended()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span; got: %d", len(spans))
	}
	span := spans[0]

	if span.EndTime().IsZero() {
		t.Errorf("span should be ended; it isn't")
	}

	if got := span.Status().Code; got != codes.Error {
		t.Errorf("expected error status code on span; got: %q", got)
	}

	errSubstr := "connect: connection refused"
	if runtime.GOOS == "windows" {
		// tls.Dial returns an error that does not contain the substring "connection refused"
		// on Windows machines
		//
		// ref: "dial tcp 127.0.0.1:50115: connectex: No connection could be made because the target machine actively refused it."
		errSubstr = "No connection could be made because the target machine actively refused it"
	}
	if got := span.Status().Description; !strings.Contains(got, errSubstr) {
		t.Errorf("expected error status message on span; got: %q", got)
	}
}

func TestTransportRequestWithTraceContext(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(content)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_span")

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)

	r = r.WithContext(ctx)

	tr := NewTransport(
		http.DefaultTransport,
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	span.End()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, content, body)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 2)

	assert.Equal(t, "test_span", spans[0].Name())
	assert.Equal(t, "HTTP GET", spans[1].Name())
	assert.NotEmpty(t, spans[1].Parent().SpanID())
	assert.Equal(t, spans[0].SpanContext().SpanID(), spans[1].Parent().SpanID())
}

func TestWithHTTPTrace(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	content := []byte("Hello, world!")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(content)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_span")

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)

	r = r.WithContext(ctx)

	clientTracer := func(ctx context.Context) *httptrace.ClientTrace {
		var span trace.Span
		return &httptrace.ClientTrace{
			GetConn: func(string) {
				_, span = trace.SpanFromContext(ctx).TracerProvider().Tracer("").Start(ctx, "httptrace.GetConn")
			},
			GotConn: func(httptrace.GotConnInfo) {
				if span != nil {
					span.End()
				}
			},
		}
	}

	tr := NewTransport(
		http.DefaultTransport,
		WithClientTrace(clientTracer),
	)

	c := http.Client{Transport: tr}
	res, err := c.Do(r)
	require.NoError(t, err)
	defer func() { assert.NoError(t, res.Body.Close()) }()

	span.End()

	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, content, body)

	spans := spanRecorder.Ended()
	require.Len(t, spans, 3)

	assert.Equal(t, "httptrace.GetConn", spans[0].Name())
	assert.Equal(t, "test_span", spans[1].Name())
	assert.Equal(t, "HTTP GET", spans[2].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.NotEmpty(t, spans[2].Parent().SpanID())
	assert.Equal(t, spans[2].SpanContext().SpanID(), spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[2].Parent().SpanID())
}

func TestTransportMetrics(t *testing.T) {
	requestBody := []byte("john")
	responseBody := []byte("Hello, world!")

	t.Run("make http request and read entire response at once", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(responseBody); err != nil {
				t.Fatal(err)
			}
		}))
		defer ts.Close()

		r, err := http.NewRequest(http.MethodGet, ts.URL, bytes.NewReader(requestBody))
		if err != nil {
			t.Fatal(err)
		}

		tr := NewTransport(
			http.DefaultTransport,
			WithMeterProvider(meterProvider),
		)

		c := http.Client{Transport: tr}
		res, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		// Must read the body or else we won't get response metrics
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		require.Len(t, bodyBytes, 13)
		require.NoError(t, res.Body.Close())

		host, portStr, _ := net.SplitHostPort(r.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			port = 0
		}

		rm := metricdata.ResourceMetrics{}
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)
		require.Len(t, rm.ScopeMetrics, 1)
		attrs := attribute.NewSet(
			attribute.String("http.request.method", "GET"),
			attribute.Int("http.response.status_code", 200),
			attribute.String("server.address", host),
			attribute.Int("server.port", port),
			attribute.String("url.scheme", "http"),
			attribute.String("network.protocol.name", "http"),
			attribute.String("network.protocol.version", "1.1"),
		)
		assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs)
	})

	t.Run("make http request and buffer response", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(responseBody); err != nil {
				t.Fatal(err)
			}
		}))
		defer ts.Close()

		r, err := http.NewRequest(http.MethodGet, ts.URL, bytes.NewReader(requestBody))
		if err != nil {
			t.Fatal(err)
		}

		tr := NewTransport(
			http.DefaultTransport,
			WithMeterProvider(meterProvider),
		)

		c := http.Client{Transport: tr}
		res, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		// Must read the body or else we won't get response metrics
		smallBuf := make([]byte, 10)

		// Read first 10 bytes
		bc, err := res.Body.Read(smallBuf)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, 10, bc)

		// reset byte array
		// Read last 3 bytes
		bc, err = res.Body.Read(smallBuf)
		require.Equal(t, io.EOF, err)
		require.Equal(t, 3, bc)

		require.NoError(t, res.Body.Close())

		host, portStr, _ := net.SplitHostPort(r.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			port = 0
		}

		rm := metricdata.ResourceMetrics{}
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)
		require.Len(t, rm.ScopeMetrics, 1)
		attrs := attribute.NewSet(
			attribute.String("http.request.method", "GET"),
			attribute.Int("http.response.status_code", 200),
			attribute.String("server.address", host),
			attribute.Int("server.port", port),
			attribute.String("url.scheme", "http"),
			attribute.String("network.protocol.name", "http"),
			attribute.String("network.protocol.version", "1.1"),
		)
		assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs)
	})

	t.Run("make http request and close body before reading completely", func(t *testing.T) {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			if _, err := w.Write(responseBody); err != nil {
				t.Fatal(err)
			}
		}))
		defer ts.Close()

		r, err := http.NewRequest(http.MethodGet, ts.URL, bytes.NewReader(requestBody))
		if err != nil {
			t.Fatal(err)
		}

		tr := NewTransport(
			http.DefaultTransport,
			WithMeterProvider(meterProvider),
		)

		c := http.Client{Transport: tr}
		res, err := c.Do(r)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()

		// Must read the body or else we won't get response metrics
		smallBuf := make([]byte, 10)

		// Read first 10 bytes
		bc, err := res.Body.Read(smallBuf)
		if err != nil {
			t.Fatal(err)
		}
		require.Equal(t, 10, bc)

		// close the response body early
		require.NoError(t, res.Body.Close())

		host, portStr, _ := net.SplitHostPort(r.Host)
		if host == "" {
			host = "127.0.0.1"
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			port = 0
		}

		rm := metricdata.ResourceMetrics{}
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)
		require.Len(t, rm.ScopeMetrics, 1)
		attrs := attribute.NewSet(
			attribute.String("http.request.method", "GET"),
			attribute.Int("http.response.status_code", 200),
			attribute.String("server.address", host),
			attribute.Int("server.port", port),
			attribute.String("url.scheme", "http"),
			attribute.String("network.protocol.name", "http"),
			attribute.String("network.protocol.version", "1.1"),
		)
		assertClientScopeMetrics(t, rm.ScopeMetrics[0], attrs)
	})
}

func assertClientScopeMetrics(t *testing.T, sm metricdata.ScopeMetrics, attrs attribute.Set) {
	assert.Equal(t, instrumentation.Scope{
		Name:    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
		Version: Version(),
	}, sm.Scope)

	require.Len(t, sm.Metrics, 2)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    ScopeName,
			Version: Version(),
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "http.client.request.body.size",
				Description: "Size of HTTP client request bodies.",
				Unit:        "By",
				Data: metricdata.Histogram[int64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[int64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
			{
				Name:        "http.client.request.duration",
				Description: "Duration of HTTP client requests.",
				Unit:        "s",
				Data: metricdata.Histogram[float64]{
					Temporality: metricdata.CumulativeTemporality,
					DataPoints: []metricdata.HistogramDataPoint[float64]{
						{
							Attributes: attrs,
						},
					},
				},
			},
		},
	}
	metricdatatest.AssertEqual(t, want, sm, metricdatatest.IgnoreTimestamp(), metricdatatest.IgnoreValue(), metricdatatest.IgnoreExemplars())
}

func TestCustomAttributesHandling(t *testing.T) {
	var rm metricdata.ResourceMetrics
	const (
		clientRequestSize = "http.client.request.size"
		clientDuration    = "http.client.duration"
	)
	ctx := context.TODO()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() {
		err := provider.Shutdown(ctx)
		if err != nil {
			t.Errorf("Error shutting down provider: %v", err)
		}
	}()

	transport := NewTransport(http.DefaultTransport, WithMeterProvider(provider))
	client := http.Client{Transport: transport}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	expectedAttributes := []attribute.KeyValue{
		attribute.String("foo", "fooValue"),
		attribute.String("bar", "barValue"),
	}

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)
	labeler := &Labeler{}
	labeler.Add(expectedAttributes...)
	ctx = ContextWithLabeler(ctx, labeler)
	r = r.WithContext(ctx)

	// test bonus: intententionally ignoring response to confirm that
	// http.client.response.size metric is not recorded
	// by the Transport.RoundTrip logic
	resp, err := client.Do(r)
	require.NoError(t, err)
	defer func() { assert.NoError(t, resp.Body.Close()) }()

	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)

	// http.client.response.size is not recorded so the assert.Len
	// above should be 2 instead of 3(test bonus)
	assert.Len(t, rm.ScopeMetrics[0].Metrics, 2)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case clientRequestSize:
			d, ok := m.Data.(metricdata.Sum[int64])
			assert.True(t, ok)
			assert.Len(t, d.DataPoints, 1)
			containsAttributes(t, d.DataPoints[0].Attributes, expectedAttributes)
		case clientDuration:
			d, ok := m.Data.(metricdata.Histogram[float64])
			assert.True(t, ok)
			assert.Len(t, d.DataPoints, 1)
			containsAttributes(t, d.DataPoints[0].Attributes, expectedAttributes)
		}
	}
}

func TestMetricsExistenceOnRequestError(t *testing.T) {
	var rm metricdata.ResourceMetrics

	ctx := context.Background()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() {
		err := provider.Shutdown(ctx)
		assert.NoError(t, err)
	}()

	transport := NewTransport(http.DefaultTransport, WithMeterProvider(provider))
	client := http.Client{Transport: transport}

	// simulate an error by closing the server
	// before the request is made
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(r)
	if err == nil {
		e := resp.Body.Close()
		assert.NoError(t, e)
	}

	require.Error(t, err)

	err = reader.Collect(ctx, &rm)
	require.NoError(t, err)

	// make sure client request size and duration metrics is recorded
	assert.Len(t, rm.ScopeMetrics[0].Metrics, 2, "should record client request size and duration metrics")
}

func TestDefaultAttributesHandling(t *testing.T) {
	var rm metricdata.ResourceMetrics
	const (
		clientRequestSize = "http.client.request.size"
		clientDuration    = "http.client.duration"
	)
	ctx := context.TODO()
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	defer func() {
		err := provider.Shutdown(ctx)
		if err != nil {
			t.Errorf("Error shutting down provider: %v", err)
		}
	}()

	defaultAttributes := []attribute.KeyValue{
		attribute.String("defaultFoo", "fooValue"),
		attribute.String("defaultBar", "barValue"),
	}

	transport := NewTransport(
		http.DefaultTransport, WithMeterProvider(provider),
		WithMetricAttributesFn(func(_ *http.Request) []attribute.KeyValue {
			return defaultAttributes
		}))
	client := http.Client{Transport: transport}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(r)
	require.NoError(t, err)

	_ = resp.Body.Close()

	err = reader.Collect(ctx, &rm)
	assert.NoError(t, err)

	assert.Len(t, rm.ScopeMetrics[0].Metrics, 2)
	for _, m := range rm.ScopeMetrics[0].Metrics {
		switch m.Name {
		case clientRequestSize:
			d, ok := m.Data.(metricdata.Sum[int64])
			assert.True(t, ok)
			assert.Len(t, d.DataPoints, 1)
			containsAttributes(t, d.DataPoints[0].Attributes, defaultAttributes)
		case clientDuration:
			d, ok := m.Data.(metricdata.Histogram[float64])
			assert.True(t, ok)
			assert.Len(t, d.DataPoints, 1)
			containsAttributes(t, d.DataPoints[0].Attributes, defaultAttributes)
		}
	}
}

func containsAttributes(t *testing.T, attrSet attribute.Set, expected []attribute.KeyValue) {
	for _, att := range expected {
		actualValue, ok := attrSet.Value(att.Key)
		assert.True(t, ok)
		assert.Equal(t, att.Value.AsString(), actualValue.AsString())
	}
}

func BenchmarkTransportRoundTrip(b *testing.B) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "Hello World")
	}))
	defer ts.Close()

	tp := sdktrace.NewTracerProvider()
	mp := sdkmetric.NewMeterProvider()

	r, err := http.NewRequest(http.MethodGet, ts.URL, http.NoBody)
	require.NoError(b, err)

	for _, bb := range []struct {
		name      string
		transport http.RoundTripper
	}{
		{
			name:      "without the otelhttp transport",
			transport: http.DefaultTransport,
		},
		{
			name: "with the otelhttp transport",
			transport: NewTransport(
				http.DefaultTransport,
				WithTracerProvider(tp),
				WithMeterProvider(mp),
			),
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			c := http.Client{Transport: bb.transport}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				resp, _ := c.Do(r)
				resp.Body.Close()
			}
		})
	}
}

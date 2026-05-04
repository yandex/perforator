// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

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
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
)

func TestHandler(t *testing.T) {
	testCases := []struct {
		name               string
		handler            func(*testing.T) http.Handler
		requestBody        io.Reader
		expectedStatusCode int
	}{
		{
			name: "implements flusher",
			handler: func(t *testing.T) http.Handler {
				return NewHandler(
					http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
						assert.Implements(t, (*http.Flusher)(nil), w)

						w.(http.Flusher).Flush()
						_, _ = io.WriteString(w, "Hello, world!\n")
					}), "test_handler",
				)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "succeeds",
			handler: func(t *testing.T) http.Handler {
				return NewHandler(
					http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
						assert.NotNil(t, r.Body)

						b, err := io.ReadAll(r.Body)
						assert.NoError(t, err)
						assert.Equal(t, "hello world", string(b))
					}), "test_handler",
				)
			},
			requestBody:        strings.NewReader("hello world"),
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "succeeds with a nil body",
			handler: func(t *testing.T) http.Handler {
				return NewHandler(
					http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
						assert.Nil(t, r.Body)
					}), "test_handler",
				)
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "succeeds with an http.NoBody",
			handler: func(t *testing.T) http.Handler {
				return NewHandler(
					http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
						assert.Equal(t, http.NoBody, r.Body)
					}), "test_handler",
				)
			},
			requestBody:        http.NoBody,
			expectedStatusCode: http.StatusOK,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := http.NewRequest(http.MethodGet, "http://localhost/", tc.requestBody)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			tc.handler(t).ServeHTTP(rr, r)
			assert.Equal(t, tc.expectedStatusCode, rr.Result().StatusCode)
		})
	}
}

func TestHandlerBasics(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
	rr := httptest.NewRecorder()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(spanRecorder))

	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			l, _ := LabelerFromContext(r.Context())
			l.Add(attribute.String("test", "attribute"))

			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), "test_handler",
		WithTracerProvider(provider),
		WithMeterProvider(meterProvider),
		WithPropagators(propagation.TraceContext{}),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", strings.NewReader("foo"))
	if err != nil {
		t.Fatal(err)
	}
	// set a custom start time 10 minutes in the past.
	startTime := time.Now().Add(-10 * time.Minute)
	r = r.WithContext(ContextWithStartTime(r.Context(), startTime))
	h.ServeHTTP(rr, r)

	rm := metricdata.ResourceMetrics{}
	err = reader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1)
	attrs := attribute.NewSet(
		attribute.String("http.request.method", "GET"),
		attribute.Int64("http.response.status_code", 200),
		attribute.String("network.protocol.name", "http"),
		attribute.String("network.protocol.version", fmt.Sprintf("1.%d", r.ProtoMinor)),
		attribute.String("server.address", r.Host),
		attribute.String("url.scheme", "http"),
		attribute.String("test", "attribute"),
	)
	assertScopeMetrics(t, rm.ScopeMetrics[0], attrs)

	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}

	spans := spanRecorder.Ended()
	if got, expected := len(spans), 1; got != expected {
		t.Fatalf("got %d spans, expected %d", got, expected)
	}
	if !spans[0].SpanContext().IsValid() {
		t.Fatalf("invalid span created: %#v", spans[0].SpanContext())
	}

	d, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := string(d), "hello world"; got != expected {
		t.Fatalf("got %q, expected %q", got, expected)
	}
	assert.Equal(t, startTime, spans[0].StartTime())
}

func assertScopeMetrics(t *testing.T, sm metricdata.ScopeMetrics, attrs attribute.Set) {
	assert.Equal(t, instrumentation.Scope{
		Name:    "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp",
		Version: Version(),
	}, sm.Scope)

	require.Len(t, sm.Metrics, 3)

	want := metricdata.ScopeMetrics{
		Scope: instrumentation.Scope{
			Name:    ScopeName,
			Version: Version(),
		},
		Metrics: []metricdata.Metrics{
			{
				Name:        "http.server.request.body.size",
				Description: "Size of HTTP server request bodies.",
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
				Name:        "http.server.response.body.size",
				Description: "Size of HTTP server response bodies.",
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
				Name:        "http.server.request.duration",
				Description: "Duration of HTTP server requests.",
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

	// verify that the custom start time, which is 10 minutes in the past, is respected.
	assert.GreaterOrEqual(t, sm.Metrics[2].Data.(metricdata.Histogram[float64]).DataPoints[0].Sum, float64(10*time.Minute/time.Second))
}

func TestHandlerEmittedAttributes(t *testing.T) {
	testCases := []struct {
		name       string
		handler    func(http.ResponseWriter, *http.Request)
		attributes []attribute.KeyValue
	}{
		{
			name: "With a success handler",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.response.status_code", http.StatusOK),
			},
		},
		{
			name: "With a failing handler",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.response.status_code", http.StatusBadRequest),
			},
		},
		{
			name: "With an empty handler",
			handler: func(http.ResponseWriter, *http.Request) {
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.response.status_code", http.StatusOK),
			},
		},
		{
			name: "With persisting initial failing status in handler with multiple WriteHeader calls",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.WriteHeader(http.StatusOK)
			},
			attributes: []attribute.KeyValue{
				attribute.Int("http.response.status_code", http.StatusInternalServerError),
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			h := NewHandler(
				http.HandlerFunc(tc.handler), "test_handler",
				WithTracerProvider(provider),
			)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			attrs := sr.Ended()[0].Attributes()

			for _, a := range tc.attributes {
				assert.Contains(t, attrs, a)
			}
		})
	}
}

type respWriteHeaderCounter struct {
	http.ResponseWriter

	headersWritten []int
}

func (rw *respWriteHeaderCounter) WriteHeader(statusCode int) {
	rw.headersWritten = append(rw.headersWritten, statusCode)
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *respWriteHeaderCounter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func TestHandlerPropagateWriteHeaderCalls(t *testing.T) {
	testCases := []struct {
		name                 string
		handler              func(http.ResponseWriter, *http.Request)
		expectHeadersWritten []int
	}{
		{
			name: "With a success handler",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			},
			expectHeadersWritten: []int{http.StatusOK},
		},
		{
			name: "With a failing handler",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
			},
			expectHeadersWritten: []int{http.StatusBadRequest},
		},
		{
			name: "With an empty handler",
			handler: func(http.ResponseWriter, *http.Request) {
			},

			expectHeadersWritten: nil,
		},
		{
			name: "With calling WriteHeader twice",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				w.WriteHeader(http.StatusOK)
			},
			expectHeadersWritten: []int{http.StatusInternalServerError, http.StatusOK},
		},
		{
			name: "When writing the header indirectly through body write",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("hello"))
			},
			expectHeadersWritten: []int{http.StatusOK},
		},
		{
			name: "With a header already written when writing the body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("hello"))
			},
			expectHeadersWritten: []int{http.StatusBadRequest},
		},
		{
			name: "When writing the header indirectly through flush",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				f := w.(http.Flusher)
				f.Flush()
			},
			expectHeadersWritten: []int{http.StatusOK},
		},
		{
			name: "With a header already written when flushing",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				f := w.(http.Flusher)
				f.Flush()
			},
			expectHeadersWritten: []int{http.StatusBadRequest},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			h := NewHandler(
				http.HandlerFunc(tc.handler), "test_handler",
				WithTracerProvider(provider),
			)

			recorder := httptest.NewRecorder()
			rw := &respWriteHeaderCounter{ResponseWriter: recorder}
			h.ServeHTTP(rw, httptest.NewRequest(http.MethodGet, "/", http.NoBody))
			require.Equal(t, tc.expectHeadersWritten, rw.headersWritten, "should propagate all WriteHeader calls to underlying ResponseWriter")
		})
	}
}

func TestHandlerRequestWithTraceContext(t *testing.T) {
	rr := httptest.NewRecorder()

	h := NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write([]byte("hello world"))
			assert.NoError(t, err)
		}), "test_handler")

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
	require.NoError(t, err)

	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	tracer := provider.Tracer("")
	ctx, span := tracer.Start(context.Background(), "test_request")
	r = r.WithContext(ctx)

	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

	span.End()

	spans := spanRecorder.Ended()
	require.Len(t, spans, 2)

	assert.Equal(t, "test_handler", spans[0].Name())
	assert.Equal(t, "test_request", spans[1].Name())
	assert.NotEmpty(t, spans[0].Parent().SpanID())
	assert.Equal(t, spans[1].SpanContext().SpanID(), spans[0].Parent().SpanID())
}

func TestWithSpanNameFormatter(t *testing.T) {
	for _, tt := range []struct {
		name string

		formatter    func(operation string, r *http.Request) string
		wantSpanName string
	}{
		{
			name:         "with the default span name formatter",
			wantSpanName: "test_handler",
		},
		{
			name: "with a custom span name formatter",
			formatter: func(_ string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			},
			wantSpanName: "GET /foo/123",
		},
		{
			name: "with a custom span name formatter using the pattern",
			formatter: func(_ string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.Pattern)
			},
			wantSpanName: "GET /foo/{id}",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spanRecorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(spanRecorder),
			)

			opts := []Option{
				WithTracerProvider(provider),
			}
			if tt.formatter != nil {
				opts = append(opts, WithSpanNameFormatter(tt.formatter))
			}

			mux := http.NewServeMux()
			mux.Handle("/foo/{id}", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				// Nothing to do here
			}))
			h := NewHandler(mux, "test_handler", opts...)

			r, err := http.NewRequest(http.MethodGet, "http://localhost/foo/123", http.NoBody)
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)
			assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

			assert.NoError(t, spanRecorder.ForceFlush(context.Background()))
			spans := spanRecorder.Ended()
			assert.Len(t, spans, 1)
			assert.Equal(t, tt.wantSpanName, spans[0].Name())
		})
	}
}

func TestWithPublicEndpoint(t *testing.T) {
	spanRecorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(spanRecorder),
	)
	remoteSpan := trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	}
	prop := propagation.TraceContext{}
	h := NewHandler(
		http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			s := trace.SpanFromContext(r.Context())
			sc := s.SpanContext()

			// Should be with new root trace.
			assert.True(t, sc.IsValid())
			assert.False(t, sc.IsRemote())
			assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
		}), "test_handler",
		WithPublicEndpoint(),
		WithPropagators(prop),
		WithTracerProvider(provider),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
	require.NoError(t, err)

	sc := trace.NewSpanContext(remoteSpan)
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, r)
	assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

	// Recorded span should be linked with an incoming span context.
	assert.NoError(t, spanRecorder.ForceFlush(ctx))
	done := spanRecorder.Ended()
	require.Len(t, done, 1)
	require.Len(t, done[0].Links(), 1, "should contain link")
	require.True(t, sc.Equal(done[0].Links()[0].SpanContext), "should link incoming span context")
}

func TestWithPublicEndpointFn(t *testing.T) {
	remoteSpan := trace.SpanContextConfig{
		TraceID:    trace.TraceID{0x01},
		SpanID:     trace.SpanID{0x01},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	}
	prop := propagation.TraceContext{}

	for _, tt := range []struct {
		name          string
		fn            func(*http.Request) bool
		handlerAssert func(*testing.T, trace.SpanContext)
		spansAssert   func(*testing.T, trace.SpanContext, []sdktrace.ReadOnlySpan)
	}{
		{
			name: "with the method returning true",
			fn: func(*http.Request) bool {
				return true
			},
			handlerAssert: func(t *testing.T, sc trace.SpanContext) {
				// Should be with new root trace.
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.NotEqual(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, sc trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Len(t, spans[0].Links(), 1, "should contain link")
				require.True(t, sc.Equal(spans[0].Links()[0].SpanContext), "should link incoming span context")
			},
		},
		{
			name: "with the method returning false",
			fn: func(*http.Request) bool {
				return false
			},
			handlerAssert: func(t *testing.T, sc trace.SpanContext) {
				// Should have remote span as parent
				assert.True(t, sc.IsValid())
				assert.False(t, sc.IsRemote())
				assert.Equal(t, remoteSpan.TraceID, sc.TraceID())
			},
			spansAssert: func(t *testing.T, _ trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				require.Len(t, spans, 1)
				require.Empty(t, spans[0].Links(), "should not contain link")
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			spanRecorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(
				sdktrace.WithSpanProcessor(spanRecorder),
			)

			h := NewHandler(
				http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
					s := trace.SpanFromContext(r.Context())
					tt.handlerAssert(t, s.SpanContext())
				}), "test_handler",
				WithPublicEndpointFn(tt.fn),
				WithPropagators(prop),
				WithTracerProvider(provider),
			)

			r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
			require.NoError(t, err)

			sc := trace.NewSpanContext(remoteSpan)
			ctx := trace.ContextWithSpanContext(context.Background(), sc)
			prop.Inject(ctx, propagation.HeaderCarrier(r.Header))

			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, r)
			assert.Equal(t, http.StatusOK, rr.Result().StatusCode)

			// Recorded span should be linked with an incoming span context.
			assert.NoError(t, spanRecorder.ForceFlush(ctx))
			spans := spanRecorder.Ended()
			tt.spansAssert(t, sc, spans)
		})
	}
}

func TestSpanStatus(t *testing.T) {
	testCases := []struct {
		httpStatusCode int
		wantSpanStatus codes.Code
	}{
		{http.StatusOK, codes.Unset},
		{http.StatusBadRequest, codes.Unset},
		{http.StatusInternalServerError, codes.Error},
	}
	for _, tc := range testCases {
		t.Run(strconv.Itoa(tc.httpStatusCode), func(t *testing.T) {
			sr := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider()
			provider.RegisterSpanProcessor(sr)
			h := NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tc.httpStatusCode)
				}), "test_handler",
				WithTracerProvider(provider),
			)

			h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))

			require.Len(t, sr.Ended(), 1, "should emit a span")
			assert.Equal(t, tc.wantSpanStatus, sr.Ended()[0].Status().Code, "should only set Error status for HTTP statuses >= 500")
		})
	}
}

func TestWithRouteTag(t *testing.T) {
	t.Setenv("OTEL_METRICS_EXEMPLAR_FILTER", "always_off")
	route := "/some/route"

	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider()
	tracerProvider.RegisterSpanProcessor(spanRecorder)

	metricReader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(metricReader))

	h := NewHandler(
		WithRouteTag(
			route,
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}),
		),
		"test_handler",
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
	)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", http.NoBody))
	want := semconv.HTTPRouteKey.String(route)

	require.Len(t, spanRecorder.Ended(), 1, "should emit a span")
	gotSpan := spanRecorder.Ended()[0]
	require.Contains(t, gotSpan.Attributes(), want, "should add route to span attributes")

	rm := metricdata.ResourceMetrics{}
	err := metricReader.Collect(context.Background(), &rm)
	require.NoError(t, err)
	require.Len(t, rm.ScopeMetrics, 1, "should emit metrics for one scope")
	gotMetrics := rm.ScopeMetrics[0].Metrics

	for _, m := range gotMetrics {
		switch d := m.Data.(type) {
		case metricdata.Sum[int64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Sum[float64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Histogram[int64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Histogram[float64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Gauge[int64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		case metricdata.Gauge[float64]:
			require.Len(t, d.DataPoints, 1, "metric '%v' should have exactly one data point", m.Name)
			require.Contains(t, d.DataPoints[0].Attributes.ToSlice(), want, "should add route to attributes for metric '%v'", m.Name)

		default:
			require.Fail(t, "metric has unexpected data type", "metric '%v' has unexpected data type %T", m.Name, m.Data)
		}
	}
}

func TestHandlerWithMetricAttributesFn(t *testing.T) {
	const (
		serverRequestSize  = "http.server.request.size"
		serverResponseSize = "http.server.response.size"
		serverDuration     = "http.server.duration"
	)
	testCases := []struct {
		name                        string
		fn                          func(r *http.Request) []attribute.KeyValue
		expectedAdditionalAttribute []attribute.KeyValue
	}{
		{
			name:                        "With a nil function",
			fn:                          nil,
			expectedAdditionalAttribute: []attribute.KeyValue{},
		},
		{
			name: "With a function that returns an additional attribute",
			fn: func(*http.Request) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("fooKey", "fooValue"),
					attribute.String("barKey", "barValue"),
				}
			},
			expectedAdditionalAttribute: []attribute.KeyValue{
				attribute.String("fooKey", "fooValue"),
				attribute.String("barKey", "barValue"),
			},
		},
	}

	for _, tc := range testCases {
		reader := sdkmetric.NewManualReader()
		meterProvider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))

		h := NewHandler(
			http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}), "test_handler",
			WithMeterProvider(meterProvider),
			WithMetricAttributesFn(tc.fn),
		)

		r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
		require.NoError(t, err)
		rr := httptest.NewRecorder()
		h.ServeHTTP(rr, r)

		rm := metricdata.ResourceMetrics{}
		err = reader.Collect(context.Background(), &rm)
		require.NoError(t, err)
		require.Len(t, rm.ScopeMetrics, 1)
		assert.Len(t, rm.ScopeMetrics[0].Metrics, 3)

		// Verify that the additional attribute is present in the metrics.
		for _, m := range rm.ScopeMetrics[0].Metrics {
			switch m.Name {
			case serverRequestSize, serverResponseSize:
				d, ok := m.Data.(metricdata.Sum[int64])
				assert.True(t, ok)
				assert.Len(t, d.DataPoints, 1)
				containsAttributes(t, d.DataPoints[0].Attributes, testCases[0].expectedAdditionalAttribute)
			case serverDuration:
				d, ok := m.Data.(metricdata.Histogram[float64])
				assert.True(t, ok)
				assert.Len(t, d.DataPoints, 1)
				containsAttributes(t, d.DataPoints[0].Attributes, testCases[0].expectedAdditionalAttribute)
			}
		}
	}
}

func BenchmarkHandlerServeHTTP(b *testing.B) {
	tp := sdktrace.NewTracerProvider()
	mp := sdkmetric.NewMeterProvider()

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
	require.NoError(b, err)

	for _, bb := range []struct {
		name    string
		handler http.Handler
	}{
		{
			name: "without the otelhttp handler",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				fmt.Fprint(w, "Hello World")
			}),
		},
		{
			name: "with the otelhttp handler",
			handler: NewHandler(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					fmt.Fprint(w, "Hello World")
				}),
				"test_handler",
				WithTracerProvider(tp),
				WithMeterProvider(mp),
			),
		},
	} {
		b.Run(bb.name, func(b *testing.B) {
			rr := httptest.NewRecorder()

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				bb.handler.ServeHTTP(rr, r)
			}
		})
	}
}

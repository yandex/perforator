// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func TestBasicFilter(t *testing.T) {
	rr := httptest.NewRecorder()

	spanRecorder := tracetest.NewSpanRecorder()
	provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))

	h := otelhttp.NewHandler(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if _, err := io.WriteString(w, "hello world"); err != nil {
				t.Fatal(err)
			}
		}), "test_handler",
		otelhttp.WithTracerProvider(provider),
		otelhttp.WithFilter(func(*http.Request) bool {
			return false
		}),
	)

	r, err := http.NewRequest(http.MethodGet, "http://localhost/", http.NoBody)
	if err != nil {
		t.Fatal(err)
	}
	h.ServeHTTP(rr, r)
	if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
		t.Fatalf("got %d, expected %d", got, expected)
	}
	if got := rr.Header().Get("Traceparent"); got != "" {
		t.Fatal("expected empty trace header")
	}
	if got, expected := len(spanRecorder.Ended()), 0; got != expected {
		t.Fatalf("got %d recorded spans, expected %d", got, expected)
	}
	d, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got, expected := string(d), "hello world"; got != expected {
		t.Fatalf("got %q, expected %q", got, expected)
	}
}

func TestSpanNameFormatter(t *testing.T) {
	testCases := []struct {
		name      string
		formatter func(s string, r *http.Request) string
		operation string
		expected  string
	}{
		{
			name: "default handler formatter",
			formatter: func(operation string, _ *http.Request) string {
				return operation
			},
			operation: "test_operation",
			expected:  "test_operation",
		},
		{
			name: "default transport formatter",
			formatter: func(_ string, r *http.Request) string {
				return "HTTP " + r.Method
			},
			expected: "HTTP GET",
		},
		{
			name: "custom formatter",
			formatter: func(_ string, r *http.Request) string {
				return r.URL.Path
			},
			operation: "",
			expected:  "/hello",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()

			spanRecorder := tracetest.NewSpanRecorder()
			provider := trace.NewTracerProvider(trace.WithSpanProcessor(spanRecorder))
			handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if _, err := io.WriteString(w, "hello world"); err != nil {
					t.Fatal(err)
				}
			})
			h := otelhttp.NewHandler(
				handler,
				tc.operation,
				otelhttp.WithTracerProvider(provider),
				otelhttp.WithSpanNameFormatter(tc.formatter),
			)
			r, err := http.NewRequest(http.MethodGet, "http://localhost/hello", http.NoBody)
			if err != nil {
				t.Fatal(err)
			}
			h.ServeHTTP(rr, r)
			if got, expected := rr.Result().StatusCode, http.StatusOK; got != expected {
				t.Fatalf("got %d, expected %d", got, expected)
			}

			spans := spanRecorder.Ended()
			if assert.Len(t, spans, 1) {
				assert.Equal(t, tc.expected, spans[0].Name())
			}
		})
	}
}

func TestEvent(t *testing.T) {
	t.Run("constant values", func(t *testing.T) {
		assert.Equal(t, otelhttp.ReadEvents, otelhttp.Event(1), "ReadEvents should be 1")
		assert.Equal(t, otelhttp.WriteEvents, otelhttp.Event(2), "WriteEvents should be 2")
	})

	t.Run("unspecifiedEvent", func(t *testing.T) {
		var unspecified otelhttp.Event // zero-value
		assert.Equal(t, otelhttp.Event(0), unspecified, "unspecifiedEvent should be zero-value Event")

		// Validate that unspecifiedEvent is different from defined events
		assert.NotEqual(t, otelhttp.ReadEvents, unspecified, "unspecifiedEvent should not equal ReadEvents")
		assert.NotEqual(t, otelhttp.WriteEvents, unspecified, "unspecifiedEvent should not equal WriteEvents")

		// Validate WithMessageEvents accepts unspecifiedEvent
		opt := otelhttp.WithMessageEvents(unspecified)
		assert.NotNil(t, opt, "WithMessageEvents(unspecifiedEvent) should not return nil")

		// Additional validation: test behavior with unspecified events
		optMultiple := otelhttp.WithMessageEvents(unspecified, otelhttp.ReadEvents)
		assert.NotNil(t, optMultiple, "WithMessageEvents with unspecified and valid events should not return nil")
	})

	t.Run("WithMessageEvents", func(t *testing.T) {
		tests := []struct {
			name   string
			events []otelhttp.Event
		}{
			{name: "ReadEvents", events: []otelhttp.Event{otelhttp.ReadEvents}},
			{name: "WriteEvents", events: []otelhttp.Event{otelhttp.WriteEvents}},
			{name: "multiple events", events: []otelhttp.Event{otelhttp.ReadEvents, otelhttp.WriteEvents}},
			{name: "no events", events: []otelhttp.Event{}},
			{name: "unspecified event only", events: []otelhttp.Event{otelhttp.Event(0)}},
			{name: "mixed with unspecified", events: []otelhttp.Event{otelhttp.Event(0), otelhttp.ReadEvents}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := otelhttp.WithMessageEvents(tt.events...)
				assert.NotNil(t, got, "WithMessageEvents(%v) should not return nil", tt.events)
			})
		}
	})
}

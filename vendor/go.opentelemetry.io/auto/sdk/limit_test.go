// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpanLimit(t *testing.T) {
	tests := []struct {
		name string
		get  func(spanLimits) int
		zero int
		keys []string
	}{
		{
			name: "AttributeValueLengthLimit",
			get:  func(sl spanLimits) int { return sl.AttrValueLen },
			zero: -1,
			keys: []string{
				"OTEL_SPAN_ATTRIBUTE_VALUE_LENGTH_LIMIT",
				"OTEL_ATTRIBUTE_VALUE_LENGTH_LIMIT",
			},
		},
		{
			name: "AttributeCountLimit",
			get:  func(sl spanLimits) int { return sl.Attrs },
			zero: 128,
			keys: []string{
				"OTEL_SPAN_ATTRIBUTE_COUNT_LIMIT",
				"OTEL_ATTRIBUTE_COUNT_LIMIT",
			},
		},
		{
			name: "EventCountLimit",
			get:  func(sl spanLimits) int { return sl.Events },
			zero: 128,
			keys: []string{"OTEL_SPAN_EVENT_COUNT_LIMIT"},
		},
		{
			name: "EventAttributeCountLimit",
			get:  func(sl spanLimits) int { return sl.EventAttrs },
			zero: 128,
			keys: []string{"OTEL_EVENT_ATTRIBUTE_COUNT_LIMIT"},
		},
		{
			name: "LinkCountLimit",
			get:  func(sl spanLimits) int { return sl.Links },
			zero: 128,
			keys: []string{"OTEL_SPAN_LINK_COUNT_LIMIT"},
		},
		{
			name: "LinkAttributeCountLimit",
			get:  func(sl spanLimits) int { return sl.LinkAttrs },
			zero: 128,
			keys: []string{"OTEL_LINK_ATTRIBUTE_COUNT_LIMIT"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("Default", func(t *testing.T) {
				assert.Equal(t, test.zero, test.get(newSpanLimits()))
			})

			t.Run("ValidValue", func(t *testing.T) {
				for _, key := range test.keys {
					t.Run(key, func(t *testing.T) {
						t.Setenv(key, "43")
						assert.Equal(t, 43, test.get(newSpanLimits()))
					})
				}
			})

			t.Run("InvalidValue", func(t *testing.T) {
				for _, key := range test.keys {
					t.Run(key, func(t *testing.T) {
						orig := slog.Default()
						t.Cleanup(func() { slog.SetDefault(orig) })

						var buf bytes.Buffer
						h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{})
						slog.SetDefault(slog.New(h))

						const value = "invalid int value."
						t.Setenv(key, value)
						assert.Equal(t, test.zero, test.get(newSpanLimits()))

						var data map[string]any
						require.NoError(t, json.NewDecoder(&buf).Decode(&data))
						assert.Equal(t, "invalid limit environment variable", data["msg"], "log message")
						assert.Equal(t, "WARN", data["level"], "logged level")
						assert.Equal(t, key, data["key"], "logged key")
						assert.Equal(t, value, data["value"], "logged value")
						assert.NotEmpty(t, data["error"], "logged error")
					})
				}
			})
		})
	}
}

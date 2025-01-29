// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"context"
	"math"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/auto/sdk/internal/telemetry"
)

const tName = "tracer.name"

func TestSpanKindTransform(t *testing.T) {
	tests := map[trace.SpanKind]telemetry.SpanKind{
		trace.SpanKind(-1):          telemetry.SpanKind(0),
		trace.SpanKindUnspecified:   telemetry.SpanKind(0),
		trace.SpanKind(math.MaxInt): telemetry.SpanKind(0),

		trace.SpanKindInternal: telemetry.SpanKindInternal,
		trace.SpanKindServer:   telemetry.SpanKindServer,
		trace.SpanKindClient:   telemetry.SpanKindClient,
		trace.SpanKindProducer: telemetry.SpanKindProducer,
		trace.SpanKindConsumer: telemetry.SpanKindConsumer,
	}

	for in, want := range tests {
		assert.Equal(t, want, spanKind(in), in.String())
	}
}

func TestTracerStartPropagatesOrigCtx(t *testing.T) {
	t.Parallel()

	type ctxKey struct{}
	var key ctxKey
	val := "value"

	ctx := context.WithValue(context.Background(), key, val)
	ctx, _ = TracerProvider().Tracer(tName).Start(ctx, "span.name")

	assert.Equal(t, val, ctx.Value(key))
}

func TestTracerStartReturnsNonNilSpan(t *testing.T) {
	t.Parallel()

	tr := TracerProvider().Tracer(tName)
	_, s := tr.Start(context.Background(), "span.name")
	assert.NotNil(t, s)
}

func TestTracerStartAddsSpanToCtx(t *testing.T) {
	t.Parallel()

	tr := TracerProvider().Tracer(tName)
	ctx, s := tr.Start(context.Background(), "span.name")

	assert.Same(t, s, trace.SpanFromContext(ctx))
}

func TestTracerConcurrentSafe(t *testing.T) {
	t.Parallel()

	const goroutines = 10

	ctx := context.Background()
	run := func(tracer trace.Tracer) <-chan struct{} {
		done := make(chan struct{})

		go func(tr trace.Tracer) {
			defer close(done)

			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(name string) {
					defer wg.Done()
					_, _ = tr.Start(ctx, name)
				}("span" + strconv.Itoa(i))
			}

			wg.Wait()
		}(tracer)

		return done
	}

	assert.NotPanics(t, func() {
		tp := TracerProvider()
		done0, done1 := run(tp.Tracer("t0")), run(tp.Tracer("t1"))

		<-done0
		<-done1
	})
}

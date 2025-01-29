// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sdk

import (
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
)

func TestTracerProviderInstance(t *testing.T) {
	t.Parallel()

	tp0, tp1 := TracerProvider(), TracerProvider()

	assert.Same(t, tracerProviderInstance, tp0)
	assert.Same(t, tracerProviderInstance, tp1)
}

func TestTracerProviderConcurrentSafe(t *testing.T) {
	t.Parallel()

	const goroutines = 10

	run := func(tp trace.TracerProvider) <-chan struct{} {
		done := make(chan struct{})
		go func(tp trace.TracerProvider) {
			defer close(done)

			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func(name, version string) {
					defer wg.Done()
					_ = tp.Tracer(name, trace.WithInstrumentationVersion(version))
				}("tracer"+strconv.Itoa(i%4), strconv.Itoa(i%2))
			}

			wg.Wait()
		}(tp)
		return done
	}

	assert.NotPanics(t, func() {
		done0, done1 := run(TracerProvider()), run(TracerProvider())

		<-done0
		<-done1
	})
}

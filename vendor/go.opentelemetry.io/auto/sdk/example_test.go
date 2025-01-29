// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sdk

import "context"

func ExampleTracerProvider() {
	// Get a Tracer from an auto-instrumented TracerProvider so all spans
	// created will be passed to the auto-instrumentation telemetry pipeline.
	tracer := TracerProvider().Tracer("my.pkg/name")

	// The tracer is used normally to create spans to encapsulate work.
	_, span := tracer.Start(context.Background(), "do.work")
	defer span.End()

	// Do work ...
}

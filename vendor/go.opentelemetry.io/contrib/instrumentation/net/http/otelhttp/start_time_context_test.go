// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package otelhttp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStartTimeFromContext(t *testing.T) {
	ctx := context.Background()
	startTime := StartTimeFromContext(ctx)
	assert.True(t, startTime.IsZero())

	now := time.Now()
	ctx = ContextWithStartTime(ctx, now)
	startTime = StartTimeFromContext(ctx)
	assert.True(t, startTime.Equal(now))
}

package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	testRetryableCode      int32 = 252
	testContextWaitTimeout       = time.Second
)

type testExecFunc func(ctx context.Context, query string, args ...any) error

type testDriverConn struct {
	driver.Conn
	exec testExecFunc
}

func (c *testDriverConn) Exec(ctx context.Context, query string, args ...any) error {
	return c.exec(ctx, query, args...)
}

func newTestConnection(exec testExecFunc, config ExecRetryConfig) *Connection {
	return newConnection(&testDriverConn{exec: exec}, RetryConfig{}, config, mock.NewRegistry(nil))
}

func testExecRetryConfig() ExecRetryConfig {
	return ExecRetryConfig{
		InitialBackoff:      time.Nanosecond,
		MaxBackoff:          time.Nanosecond,
		MaxAttempts:         3,
		RetryableErrorCodes: []int32{testRetryableCode},
	}
}

type testExecRunner struct {
	t         *testing.T
	conn      *Connection
	operation string
}

func newTestExecRunner(t *testing.T, exec testExecFunc, config ExecRetryConfig) *testExecRunner {
	t.Helper()
	return &testExecRunner{
		t:         t,
		conn:      newTestConnection(exec, config),
		operation: "test_insert",
	}
}

func (r *testExecRunner) Exec(ctx context.Context, query string, args ...any) error {
	r.t.Helper()
	return ExecWithRetries(xlog.ForTest(r.t), ctx, r.conn, r.operation, query, args...)
}

func (r *testExecRunner) retryableErrorCode(err error) (int32, bool) {
	return retryableErrorCode(err, r.conn.execRetryableCodes)
}

func (r *testExecRunner) metrics() *execRetryMetrics {
	return r.conn.execRetryMetrics[r.operation]
}

func newRetryableException() error {
	return &clickhousego.Exception{Code: testRetryableCode, Message: "Too many parts"}
}

func waitForContextDone(t *testing.T, ctx context.Context) error {
	t.Helper()

	timer := time.NewTimer(testContextWaitTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		t.Fatal("context was not cancelled")
		return nil
	}
}

func TestExecWithRetriesSucceedsOnFirstAttemptWithinMaxElapsedTime(t *testing.T) {
	attempts := 0
	config := testExecRetryConfig()
	config.MaxElapsedTime = time.Second
	runner := newTestExecRunner(t, func(ctx context.Context, _ string, _ ...any) error {
		attempts++
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("expected deadline")
		}
		return nil
	}, config)

	if err := runner.Exec(t.Context(), "INSERT"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesEventuallySucceeds(t *testing.T) {
	attempts := 0
	config := testExecRetryConfig()
	config.MaxElapsedTime = time.Second
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("wrapped: %w", newRetryableException())
		}
		return nil
	}, config)

	err := runner.Exec(t.Context(), "secret query", "secret argument")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts: got %d, want 3", attempts)
	}
	if got := runner.metrics().retryAttempts[testRetryableCode].(*mock.Counter).Value.Load(); got != 2 {
		t.Fatalf("retry attempts metric: got %d, want 2", got)
	}
	if got := runner.metrics().successAfterRetry.(*mock.Counter).Value.Load(); got != 1 {
		t.Fatalf("success-after-retry metric: got %d, want 1", got)
	}
}

func TestExecWithRetriesReusesOperationMetrics(t *testing.T) {
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		if attempts%2 == 1 {
			return newRetryableException()
		}
		return nil
	}, testExecRetryConfig())

	for i := 0; i < 2; i++ {
		if err := runner.Exec(t.Context(), "INSERT"); err != nil {
			t.Fatalf("Exec: %v", err)
		}
	}

	if got := runner.metrics().retryAttempts[testRetryableCode].(*mock.Counter).Value.Load(); got != 2 {
		t.Fatalf("retry attempts metric: got %d, want 2", got)
	}
	if got := runner.metrics().successAfterRetry.(*mock.Counter).Value.Load(); got != 2 {
		t.Fatalf("success-after-retry metric: got %d, want 2", got)
	}
}

func TestExecWithRetriesStopsAtMaxAttempts(t *testing.T) {
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		return newRetryableException()
	}, testExecRetryConfig())

	err := runner.Exec(t.Context(), "INSERT")
	if _, ok := runner.retryableErrorCode(err); !ok {
		t.Fatalf("error: got %v, want retryable ClickHouse exception", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts: got %d, want 3", attempts)
	}
	if got := runner.metrics().retryBudgetExhausted.(*mock.Counter).Value.Load(); got != 1 {
		t.Fatalf("retry-budget-exhausted metric: got %d, want 1", got)
	}
}

func TestExecWithRetriesStopsAtMaxElapsedTime(t *testing.T) {
	attempts := 0
	config := testExecRetryConfig()
	config.InitialBackoff = time.Second
	config.MaxBackoff = time.Second
	config.MaxAttempts = 2
	config.MaxElapsedTime = 10 * time.Millisecond
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		return newRetryableException()
	}, config)

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error: got %v, want context.DeadlineExceeded", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesMaxElapsedTimeCancelsRunningAttempt(t *testing.T) {
	attempts := 0
	config := testExecRetryConfig()
	config.MaxElapsedTime = 10 * time.Millisecond
	runner := newTestExecRunner(t, func(ctx context.Context, _ string, _ ...any) error {
		attempts++
		return waitForContextDone(t, ctx)
	}, config)

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error: got %v, want context.DeadlineExceeded", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesDeadlineWinsOverRetryableErrorFromRunningAttempt(t *testing.T) {
	attempts := 0
	config := testExecRetryConfig()
	config.MaxElapsedTime = 10 * time.Millisecond
	runner := newTestExecRunner(t, func(ctx context.Context, _ string, _ ...any) error {
		attempts++
		_ = waitForContextDone(t, ctx)
		return newRetryableException()
	}, config)

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error: got %v, want context.DeadlineExceeded", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesHonorsEarlierParentDeadline(t *testing.T) {
	attempts := 0
	config := testExecRetryConfig()
	config.MaxElapsedTime = time.Hour
	runner := newTestExecRunner(t, func(ctx context.Context, _ string, _ ...any) error {
		attempts++
		return waitForContextDone(t, ctx)
	}, config)

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	err := runner.Exec(ctx, "INSERT")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error: got %v, want context.DeadlineExceeded", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesZeroMaxElapsedTimeDoesNotAddDeadline(t *testing.T) {
	config := testExecRetryConfig()
	config.MaxElapsedTime = 0
	runner := newTestExecRunner(t, func(ctx context.Context, _ string, _ ...any) error {
		if deadline, ok := ctx.Deadline(); ok {
			t.Fatalf("unexpected deadline: %s", deadline)
		}
		return nil
	}, config)

	if err := runner.Exec(t.Context(), "INSERT"); err != nil {
		t.Fatalf("Exec: %v", err)
	}
}

func TestExecWithRetriesMaxElapsedTimeWithoutRetries(t *testing.T) {
	attempts := 0
	runner := newTestExecRunner(t, func(ctx context.Context, _ string, _ ...any) error {
		attempts++
		return waitForContextDone(t, ctx)
	}, ExecRetryConfig{MaxElapsedTime: 10 * time.Millisecond})

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error: got %v, want context.DeadlineExceeded", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesStopsOnUnlistedClickHouseCode(t *testing.T) {
	wantErr := &clickhousego.Exception{Code: testRetryableCode + 1, Message: "unlisted error"}
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		return wantErr
	}, testExecRetryConfig())

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error: got %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
	if got := runner.metrics().retryAttempts[testRetryableCode].(*mock.Counter).Value.Load(); got != 0 {
		t.Fatalf("retry attempts metric: got %d, want 0", got)
	}
	if got := runner.metrics().abortedByNonRetryableError.(*mock.Counter).Value.Load(); got != 0 {
		t.Fatalf("aborted-by-non-retryable-error metric: got %d, want 0", got)
	}
}

func TestExecWithRetriesClassifiesNonRetryableErrorAfterRetry(t *testing.T) {
	wantErr := errors.New("connection reset after insert")
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		if attempts == 1 {
			return newRetryableException()
		}
		return wantErr
	}, testExecRetryConfig())

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error: got %v, want %v", err, wantErr)
	}
	if attempts != 2 {
		t.Fatalf("attempts: got %d, want 2", attempts)
	}
	if got := runner.metrics().retryAttempts[testRetryableCode].(*mock.Counter).Value.Load(); got != 1 {
		t.Fatalf("retry attempts metric: got %d, want 1", got)
	}
	if got := runner.metrics().abortedByNonRetryableError.(*mock.Counter).Value.Load(); got != 1 {
		t.Fatalf("aborted-by-non-retryable-error metric: got %d, want 1", got)
	}
	if got := runner.metrics().retryBudgetExhausted.(*mock.Counter).Value.Load(); got != 0 {
		t.Fatalf("retry-budget-exhausted metric: got %d, want 0", got)
	}
}

func TestExecWithRetriesDoesNotRetryUnconfiguredError(t *testing.T) {
	wantErr := errors.New("connection reset after insert")
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		return wantErr
	}, testExecRetryConfig())

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error: got %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesEmptyAllowlistDisablesRetries(t *testing.T) {
	wantErr := newRetryableException()
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		return wantErr
	}, ExecRetryConfig{})

	err := runner.Exec(t.Context(), "INSERT")
	if !errors.Is(err, wantErr) {
		t.Fatalf("error: got %v, want %v", err, wantErr)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
}

func TestExecWithRetriesHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	attempts := 0
	runner := newTestExecRunner(t, func(context.Context, string, ...any) error {
		attempts++
		cancel()
		return newRetryableException()
	}, testExecRetryConfig())

	err := runner.Exec(ctx, "INSERT")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error: got %v, want context.Canceled", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts: got %d, want 1", attempts)
	}
	if got := runner.metrics().contextCancelled.(*mock.Counter).Value.Load(); got != 1 {
		t.Fatalf("context-cancelled metric: got %d, want 1", got)
	}
}

func TestExecRetryConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config ExecRetryConfig
	}{
		{
			name: "missing backoff",
			config: ExecRetryConfig{
				MaxAttempts:         3,
				RetryableErrorCodes: []int32{testRetryableCode},
			},
		},
		{
			name: "unbounded",
			config: ExecRetryConfig{
				InitialBackoff:      time.Second,
				MaxBackoff:          time.Second,
				RetryableErrorCodes: []int32{testRetryableCode},
			},
		},
		{
			name: "duplicate code",
			config: ExecRetryConfig{
				InitialBackoff:      time.Second,
				MaxBackoff:          time.Second,
				MaxAttempts:         3,
				RetryableErrorCodes: []int32{testRetryableCode, testRetryableCode},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.config.validate(); err == nil {
				t.Fatal("ExecRetryConfig.validate unexpectedly succeeded")
			}
		})
	}
}

func TestConnectValidatesExecRetryConfig(t *testing.T) {
	_, err := Connect(t.Context(), &Config{
		ExecRetry: ExecRetryConfig{
			InitialBackoff:      time.Second,
			MaxBackoff:          time.Second,
			RetryableErrorCodes: []int32{testRetryableCode},
		},
	}, mock.NewRegistry(nil))
	if err == nil || !strings.Contains(err.Error(), "invalid Exec retry config") {
		t.Fatalf("Connect error: got %v, want invalid Exec retry config", err)
	}
}

func TestExecWithRetriesRejectsNilConnection(t *testing.T) {
	err := ExecWithRetries(
		xlog.ForTest(t),
		t.Context(),
		nil,
		"test_insert",
		"INSERT",
	)
	if err == nil {
		t.Fatal("ExecWithRetries unexpectedly succeeded")
	}
}

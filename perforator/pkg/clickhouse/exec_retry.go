package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/cenkalti/backoff/v4"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type ExecRetryConfig struct {
	InitialBackoff      time.Duration `yaml:"initial_backoff"`
	MaxBackoff          time.Duration `yaml:"max_backoff"`
	MaxElapsedTime      time.Duration `yaml:"max_elapsed_time"`
	MaxAttempts         uint32        `yaml:"max_attempts"`
	RetryableErrorCodes []int32       `yaml:"retryable_error_codes"`
}

func (c ExecRetryConfig) validate() error {
	if c.InitialBackoff < 0 {
		return errors.New("initial backoff must not be negative")
	}
	if c.MaxBackoff < 0 {
		return errors.New("max backoff must not be negative")
	}
	if c.MaxElapsedTime < 0 {
		return errors.New("max elapsed time must not be negative")
	}
	if len(c.RetryableErrorCodes) == 0 {
		return nil
	}
	if c.InitialBackoff == 0 {
		return errors.New("initial backoff must be positive when retries are enabled")
	}
	if c.MaxBackoff < c.InitialBackoff {
		return errors.New("max backoff must not be less than initial backoff")
	}
	if c.MaxAttempts == 0 && c.MaxElapsedTime == 0 {
		return errors.New("either max attempts or max elapsed time must be set when retries are enabled")
	}

	seenCodes := make(map[int32]struct{}, len(c.RetryableErrorCodes))
	for _, code := range c.RetryableErrorCodes {
		if code <= 0 {
			return fmt.Errorf("retryable error code must be positive: %d", code)
		}
		if _, ok := seenCodes[code]; ok {
			return fmt.Errorf("duplicate retryable error code: %d", code)
		}
		seenCodes[code] = struct{}{}
	}

	return nil
}

type execRetryMetrics struct {
	retryAttempts              map[int32]metrics.Counter
	successAfterRetry          metrics.Counter
	retryBudgetExhausted       metrics.Counter
	contextCancelled           metrics.Counter
	abortedByNonRetryableError metrics.Counter
	elapsed                    metrics.Timer
}

func newExecRetryMetrics(
	reg metrics.Registry,
	operation string,
	retryableErrorCodes []int32,
) *execRetryMetrics {
	metricReg := reg.WithPrefix("clickhouse.exec_retry").WithTags(map[string]string{
		"operation": operation,
	})
	retryMetrics := &execRetryMetrics{
		retryAttempts:              make(map[int32]metrics.Counter, len(retryableErrorCodes)),
		successAfterRetry:          metricReg.WithTags(map[string]string{"outcome": "success_after_retry"}).Counter("outcomes.count"),
		retryBudgetExhausted:       metricReg.WithTags(map[string]string{"outcome": "retry_budget_exhausted"}).Counter("outcomes.count"),
		contextCancelled:           metricReg.WithTags(map[string]string{"outcome": "context_cancelled"}).Counter("outcomes.count"),
		abortedByNonRetryableError: metricReg.WithTags(map[string]string{"outcome": "aborted_by_non_retryable_error"}).Counter("outcomes.count"),
		elapsed:                    metricReg.Timer("elapsed.timer"),
	}
	for _, code := range retryableErrorCodes {
		retryMetrics.retryAttempts[code] = metricReg.WithTags(map[string]string{
			"error_code": strconv.FormatInt(int64(code), 10),
		}).Counter("attempts.count")
	}

	return retryMetrics
}

func retryableErrorCode(err error, retryableCodes map[int32]struct{}) (int32, bool) {
	exception, ok := errors.AsType[*clickhousego.Exception](err)
	if !ok {
		return 0, false
	}
	_, retryable := retryableCodes[exception.Code]
	return exception.Code, retryable
}

func newExecRetryBackOff(config ExecRetryConfig) backoff.BackOff {
	retryBackOff := backoff.NewExponentialBackOff()
	retryBackOff.InitialInterval = config.InitialBackoff
	retryBackOff.MaxInterval = config.MaxBackoff
	// The total elapsed time is enforced through the context in ExecWithRetries,
	// so it also interrupts an attempt that is currently running.
	retryBackOff.MaxElapsedTime = 0
	retryBackOff.Reset()

	if config.MaxAttempts > 0 {
		return backoff.WithMaxRetries(retryBackOff, uint64(config.MaxAttempts-1))
	}
	return retryBackOff
}

func (c *Connection) getExecRetryMetrics(operation string) *execRetryMetrics {
	c.execRetryMetricsMu.Lock()
	defer c.execRetryMetricsMu.Unlock()

	retryMetrics := c.execRetryMetrics[operation]
	if retryMetrics == nil {
		retryMetrics = newExecRetryMetrics(c.execRetryRegistry, operation, c.execRetryConf.RetryableErrorCodes)
		c.execRetryMetrics[operation] = retryMetrics
	}
	return retryMetrics
}

// ExecWithRetries executes a statement using the retry policy configured for the connection.
// A non-zero MaxElapsedTime bounds the total time spent on attempts and backoffs.
func ExecWithRetries(
	l xlog.Logger,
	ctx context.Context,
	conn *Connection,
	operation string,
	query string,
	args ...any,
) error {
	if conn == nil {
		return errors.New("connection is nil")
	}
	if operation == "" {
		return errors.New("operation is empty")
	}
	config := conn.execRetryConf
	if config.MaxElapsedTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.MaxElapsedTime)
		defer cancel()
	}
	if len(config.RetryableErrorCodes) == 0 {
		return conn.Exec(ctx, query, args...)
	}

	retryMetrics := conn.getExecRetryMetrics(operation)

	var (
		attempt           uint32
		retryStarted      bool
		lastRetryableCode int32
	)
	startedAt := time.Now()
	err := backoff.RetryNotify(
		func() error {
			if attempt > 0 {
				retryMetrics.retryAttempts[lastRetryableCode].Inc()
			}
			attempt++

			err := conn.Exec(ctx, query, args...)
			if err == nil {
				return nil
			}

			code, ok := retryableErrorCode(err, conn.execRetryableCodes)
			if !ok {
				return backoff.Permanent(err)
			}
			retryStarted = true
			lastRetryableCode = code
			return err
		},
		backoff.WithContext(newExecRetryBackOff(config), ctx),
		func(err error, nextRetry time.Duration) {
			l.Warn(
				ctx,
				"ClickHouse Exec failed with a retryable error",
				log.String("operation", operation),
				log.Int("attempt", int(attempt)),
				log.Int32("error_code", lastRetryableCode),
				log.Duration("next_retry_in", nextRetry),
				log.Error(err),
			)
		},
	)
	if !retryStarted {
		return err
	}

	retryMetrics.elapsed.RecordDuration(time.Since(startedAt))
	switch {
	case err == nil:
		retryMetrics.successAfterRetry.Inc()
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		retryMetrics.contextCancelled.Inc()
	default:
		if _, retryable := retryableErrorCode(err, conn.execRetryableCodes); retryable {
			retryMetrics.retryBudgetExhausted.Inc()
		} else {
			retryMetrics.abortedByNonRetryableError.Inc()
		}
	}
	return err
}

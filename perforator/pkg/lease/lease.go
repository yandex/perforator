package lease

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const leaseReleaseTimeout = 5 * time.Second

// newToken combines the hostname with a fresh random token.
func newToken() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return fmt.Sprintf("%s-%s", hostname, base64.RawURLEncoding.EncodeToString(b)), nil
}

type leaseOptions struct {
	ttl                     time.Duration
	renewInterval           time.Duration
	maxAcquireRetryAttempts uint32
	acquireRetryInterval    time.Duration
	registry                metrics.Registry
	releaseTimeout          time.Duration
	waitForLease            bool
	operationTimeout        time.Duration
	renewalErrors           metrics.Counter
}

type LeaseOption func(*leaseOptions)

func WithTTL(ttl time.Duration) LeaseOption {
	return func(o *leaseOptions) {
		o.ttl = ttl
	}
}

func WithRenewInterval(interval time.Duration) LeaseOption {
	return func(o *leaseOptions) {
		o.renewInterval = interval
	}
}

// WithReleaseTimeout bounds cleanup independently of caller cancellation.
func WithReleaseTimeout(timeout time.Duration) LeaseOption {
	return func(o *leaseOptions) { o.releaseTimeout = timeout }
}

// WithWaitForLease(false) returns ErrLeaseBusy instead of waiting.
func WithWaitForLease(wait bool) LeaseOption {
	return func(o *leaseOptions) { o.waitForLease = wait }
}

// WithOperationTimeout bounds Acquire/Renew requests; zero adds no timeout.
// Renew is always bounded by lease expiry.
func WithOperationTimeout(timeout time.Duration) LeaseOption {
	return func(o *leaseOptions) { o.operationTimeout = timeout }
}

func WithRenewalErrors(counter metrics.Counter) LeaseOption {
	return func(o *leaseOptions) { o.renewalErrors = counter }
}

// WithMaxAcquireRetryAttempts limits retries after Acquire errors; zero disables them.
func WithMaxAcquireRetryAttempts(retries uint32) LeaseOption {
	return func(o *leaseOptions) {
		o.maxAcquireRetryAttempts = retries
	}
}

func WithMetrics(registry metrics.Registry) LeaseOption {
	return func(o *leaseOptions) {
		o.registry = registry
	}
}

func defaultLeaseOptions() leaseOptions {
	return leaseOptions{
		ttl:                     30 * time.Second,
		maxAcquireRetryAttempts: 4,
		releaseTimeout:          leaseReleaseTimeout,
		waitForLease:            true,
	}
}

var (
	ErrLeaseLost = errors.New("lease was lost")
	ErrLeaseBusy = errors.New("lease is already held")
)

// leaseHolder tracks one acquisition and its heartbeat.
type leaseHolder struct {
	target         Lease
	token          string
	options        leaseOptions
	logger         xlog.Logger
	mu             sync.Mutex
	leaseExpiresAt time.Time
	workers        sync.WaitGroup
	leaseCtx       context.Context
	cancel         context.CancelCauseFunc
	leaseHeld      metrics.Gauge
}

func (h *leaseHolder) start(ctx context.Context) error {
	h.leaseCtx, h.cancel = context.WithCancelCause(ctx)
	if h.leaseHeld != nil {
		h.leaseHeld.Set(1)
	}
	ready := make(chan error)
	h.workers.Add(1)
	go h.watchExpiry(ready)
	// Check acquisition expiry before starting work.
	if err := <-ready; err != nil {
		return err
	}
	if cause := context.Cause(h.leaseCtx); cause != nil {
		return cause
	}
	h.workers.Add(1)
	go h.runRenewal()
	return nil
}

// stop cancels and joins background workers without releasing the lease.
func (h *leaseHolder) stop() {
	h.cancel(nil)
	h.workers.Wait()
}

// renew measures expiry from request start, including request latency.
func (h *leaseHolder) renew(ctx context.Context) (time.Time, error) {
	deadline := time.Now().Add(h.options.ttl)
	if h.options.operationTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, h.options.operationTimeout)
		defer cancel()
	}
	renewed, err := h.target.Renew(ctx, h.token, h.options.ttl)
	if err != nil {
		return time.Time{}, err
	}
	if !renewed {
		return time.Time{}, ErrLeaseLost
	}
	return deadline, nil
}

// acquire returns expiry measured from request start, without starting heartbeat.
func (h *leaseHolder) acquire(ctx context.Context) (time.Time, error) {
	options := h.options
	retryErrors := []error{}
	for {
		acquireTime := time.Now()
		if err := ctx.Err(); err != nil {
			return time.Time{}, context.Cause(ctx)
		}
		requestCtx := ctx
		cancel := func() {}
		if options.operationTimeout > 0 {
			requestCtx, cancel = context.WithTimeout(ctx, options.operationTimeout)
		}
		acquired, err := h.target.Acquire(requestCtx, h.token, options.ttl)
		cancel()
		if err == nil && acquired {
			return acquireTime.Add(options.ttl), nil
		}
		if err != nil {
			h.logger.Warn(ctx, "Failed to acquire lease", log.Error(err))
			retryErrors = append(retryErrors, err)
		} else {
			if !options.waitForLease {
				return time.Time{}, ErrLeaseBusy
			}
			h.logger.Debug(ctx, "Lease is already held")
			retryErrors = retryErrors[:0]
		}
		if err != nil && uint64(len(retryErrors)) > uint64(options.maxAcquireRetryAttempts) {
			return time.Time{}, fmt.Errorf("failed to acquire lease after %d attempts: %w", len(retryErrors), errors.Join(retryErrors...))
		}
		select {
		case <-ctx.Done():
			return time.Time{}, ctx.Err()
		case <-time.After(options.acquireRetryInterval):
		}
	}
}

func (h *leaseHolder) runRenewal() {
	defer h.workers.Done()
	ticker := time.NewTicker(h.options.renewInterval)
	defer ticker.Stop()

	for {
		select {
		case <-h.leaseCtx.Done():
			return
		case <-ticker.C:
		}

		h.mu.Lock()
		expiresAt := h.leaseExpiresAt
		h.mu.Unlock()
		if h.leaseCtx.Err() != nil {
			return
		}
		ctx, cancel := context.WithDeadline(h.leaseCtx, expiresAt)
		renewExpiresAt, err := h.renew(ctx)
		cancel()
		if err != nil && !errors.Is(err, ErrLeaseLost) && h.options.renewalErrors != nil {
			h.options.renewalErrors.Inc()
		}

		h.mu.Lock()
		if h.leaseCtx.Err() != nil {
			h.mu.Unlock()
			return
		}
		if errors.Is(err, ErrLeaseLost) {
			h.cancel(ErrLeaseLost)
			h.mu.Unlock()
			return
		}
		if err != nil {
			h.mu.Unlock()
			h.logger.Warn(h.leaseCtx, "Failed to renew lease", log.Error(err))
			continue
		}
		if !time.Now().Before(renewExpiresAt) {
			h.cancel(fmt.Errorf("%w: renewal response deadline exceeded", ErrLeaseLost))
			h.mu.Unlock()
			return
		}
		// Never shorten confirmed expiry.
		if renewExpiresAt.After(h.leaseExpiresAt) {
			h.leaseExpiresAt = renewExpiresAt
		}
		h.mu.Unlock()
		h.logger.Debug(h.leaseCtx, "Lease renewed")
	}
}

func (h *leaseHolder) watchExpiry(ready chan<- error) {
	defer h.workers.Done()
	defer func() {
		if h.leaseHeld != nil {
			h.leaseHeld.Set(0)
		}
	}()
	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		// Recheck expiry: renewal may have extended it since the timer was set.
		h.mu.Lock()
		expiresAt := h.leaseExpiresAt
		var err error
		if !time.Now().Before(expiresAt) {
			err = fmt.Errorf("%w: renewal deadline exceeded", ErrLeaseLost)
			h.cancel(err)
		}
		h.mu.Unlock()
		if ready != nil {
			ready <- err
			ready = nil
		}
		if err != nil {
			return
		}
		timer.Reset(time.Until(expiresAt))
		select {
		case <-h.leaseCtx.Done():
			return
		case <-timer.C:
		}
	}
}

// LockAndRun acquires the lease with a fresh token and runs action.
// It cancels action's context on lease loss and releases the lease on return.
// Busy leases wait by default; WithWaitForLease(false) returns ErrLeaseBusy.
// Errors report acquisition or startup failure.
// On lease loss, action's context is canceled with ErrLeaseLost.
func LockAndRun(
	ctx context.Context,
	logger xlog.Logger,
	target Lease,
	action func(ctx context.Context, token string),
	opts ...LeaseOption,
) error {
	options := defaultLeaseOptions()
	for _, opt := range opts {
		opt(&options)
	}
	if options.renewInterval == 0 {
		options.renewInterval = options.ttl / 3
	}
	if options.acquireRetryInterval == 0 {
		options.acquireRetryInterval = options.ttl / 3
	}
	if options.ttl <= 0 || options.renewInterval <= 0 || options.acquireRetryInterval <= 0 || options.releaseTimeout <= 0 {
		return fmt.Errorf("lease TTL, renewal/retry intervals and release timeout must be positive")
	}

	if options.operationTimeout < 0 {
		return fmt.Errorf("lease operation timeout must not be negative")
	}
	if target == nil || action == nil {
		return fmt.Errorf("lease and action must be set")
	}
	token, err := newToken()
	if err != nil {
		return err
	}
	logger = logger.WithName("LeaseHolder").With(log.String("holder_id", token))
	run := &leaseHolder{target: target, token: token, options: options, logger: logger}
	deadline, err := run.acquire(ctx)
	if err != nil {
		return err
	}
	// Even a late acquisition response may need cleanup in storage.
	defer func() {
		releaseCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), options.releaseTimeout)
		defer cancel()
		if err := target.Release(releaseCtx, token); err != nil {
			logger.Warn(releaseCtx, "Failed to release lease", log.Error(err))
		}
	}()

	run.leaseExpiresAt = deadline
	if options.registry != nil {
		run.leaseHeld = options.registry.Gauge("lease.held")
	}
	// Join renewal before release, even if startup fails.
	defer run.stop()
	if err := run.start(ctx); err != nil {
		return err
	}
	if cause := context.Cause(run.leaseCtx); cause != nil {
		return cause
	}
	action(run.leaseCtx, token)
	return nil
}

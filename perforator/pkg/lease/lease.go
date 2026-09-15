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

// BuildPerProcessHolderID generates a unique identifier for a lease holder based on the hostname
// and random bytes encoded in base64.
func BuildPerProcessHolderID() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}
	return fmt.Sprintf("%s-%s", hostname, base64.RawURLEncoding.EncodeToString(b)), nil
}

type leaseOptions struct {
	ttl                  time.Duration
	renewInterval        time.Duration
	maxAcquireRetries    uint32
	acquireRetryInterval time.Duration
	registry             metrics.Registry
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

func WithMaxAcquireRetries(retries uint32) LeaseOption {
	return func(o *leaseOptions) {
		o.maxAcquireRetries = retries
	}
}

func WithMetrics(registry metrics.Registry) LeaseOption {
	return func(o *leaseOptions) {
		o.registry = registry
	}
}

func defaultLeaseOptions() leaseOptions {
	return leaseOptions{
		ttl:               30 * time.Second,
		maxAcquireRetries: 5,
	}
}

var (
	ErrLeaseLost = errors.New("lease was lost")
)

// leaseHolder manages a single distributed lease.
type leaseHolder struct {
	storage  Storage
	logger   xlog.Logger
	name     string
	holderID string
	options  leaseOptions

	mu             sync.Mutex // Protects the deadline and its expiry check.
	leaseExpiresAt time.Time
	workers        sync.WaitGroup

	leaseCtx  context.Context
	cancel    context.CancelCauseFunc
	leaseHeld metrics.Gauge
}

// newLeaseHolder creates a new leaseHolder for a specific lease and holder.
func newLeaseHolder(
	logger xlog.Logger,
	storage Storage,
	leaseName string,
	holderID string,
	opts ...LeaseOption,
) *leaseHolder {
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

	h := &leaseHolder{
		logger:   logger.WithName("LeaseHolder").With(log.String("lease_name", leaseName), log.String("holder_id", holderID)),
		storage:  storage,
		name:     leaseName,
		holderID: holderID,
		options:  options,
	}

	if options.registry != nil {
		h.leaseHeld = options.registry.WithTags(map[string]string{
			"name": leaseName,
		}).Gauge("lease.held")
	}

	return h
}

// hold attempts to acquire the lease, retrying if it is already held or if transient errors occur.
// If successful, it starts renewal and expiry monitoring goroutines.
// It blocks until the lease is acquired, the maximum number of retries for storage errors is exceeded,
// or the provided context is canceled.
// The lease lifetime is tied to the context passed to this method.
// If the context is canceled, the lease will be released.
func (h *leaseHolder) hold(ctx context.Context) error {
	if h.options.ttl <= 0 || h.options.renewInterval <= 0 || h.options.acquireRetryInterval <= 0 {
		return fmt.Errorf("lease TTL and renewal/retry intervals must be positive")
	}

	retryErrors := []error{}
	for {
		acquireTime := time.Now()
		acquired, err := h.storage.Acquire(ctx, h.name, h.holderID, h.options.ttl)
		if err == nil && acquired {
			h.leaseExpiresAt = acquireTime.Add(h.options.ttl)
			h.leaseCtx, h.cancel = context.WithCancelCause(ctx)
			if h.leaseHeld != nil {
				h.leaseHeld.Set(1)
			}
			ready := make(chan error)
			h.workers.Add(1)
			go h.watchExpiry(ready)
			// Wait for the initial expiry check before starting any work.
			if err := <-ready; err != nil {
				if closeErr := h.close(); closeErr != nil {
					h.logger.Warn(ctx, "Failed to close lease holder", log.Error(closeErr))
				}
				return err
			}
			h.workers.Add(1)
			go h.runRenewal()
			return nil
		}

		if err != nil {
			h.logger.Warn(ctx, "Failed to acquire lease", log.Error(err))
			retryErrors = append(retryErrors, err)
		} else if !acquired {
			h.logger.Debug(ctx, "Lease is already held")
			retryErrors = retryErrors[:0]
		}

		if len(retryErrors) >= int(h.options.maxAcquireRetries) {
			return fmt.Errorf("failed to acquire lease after %d retries: %w", len(retryErrors), errors.Join(retryErrors...))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(h.options.acquireRetryInterval):
			continue
		}
	}
}

// context returns a context that is canceled if the lease is lost or released.
// Returns nil if the lease has not been acquired.
func (h *leaseHolder) context() context.Context {
	return h.leaseCtx
}

// close stops the renewal process and releases the lease.
// It waits for both background goroutines, including any in-flight renewal.
func (h *leaseHolder) close() error {
	if h.cancel == nil {
		return nil
	}

	h.cancel(nil)
	h.workers.Wait()

	h.release(h.leaseCtx)
	return nil
}

func (h *leaseHolder) release(ctx context.Context) {
	// A late acquisition or renewal may have succeeded in storage even after
	// our confirmed deadline. Cleanup needs its own budget, including when
	// the parent context has already been canceled.
	releaseCtx, cancelReleaseCtx := context.WithTimeout(context.WithoutCancel(ctx), leaseReleaseTimeout)
	defer cancelReleaseCtx()

	if err := h.storage.Release(releaseCtx, h.name, h.holderID); err != nil {
		h.logger.Warn(releaseCtx, "Failed to release lease", log.Error(err))
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
		// Count the TTL from request start, including storage latency.
		renewExpiresAt := time.Now().Add(h.options.ttl)
		renewed, err := h.storage.Renew(ctx, h.name, h.holderID, h.options.ttl)
		cancel()

		h.mu.Lock()
		if h.leaseCtx.Err() != nil {
			h.mu.Unlock()
			return
		}
		if err != nil {
			h.mu.Unlock()
			h.logger.Warn(h.leaseCtx, "Failed to renew lease", log.Error(err))
			continue
		}
		if !renewed {
			h.cancel(ErrLeaseLost)
			h.mu.Unlock()
			return
		}
		h.leaseExpiresAt = renewExpiresAt
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
		// Renewal only extends the deadline; the old timer may fire first.
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

// LockAndRun tries to acquire a lease with the given name.
// If successful, it executes the action function.
// The action function receives a context that is canceled if the lease is lost or released.
// If the lease is already held, it waits until it can acquire it or ctx is canceled.
func LockAndRun(
	ctx context.Context,
	logger xlog.Logger,
	storage Storage,
	leaseName string,
	holderID string,
	action func(ctx context.Context),
	opts ...LeaseOption,
) error {
	holder := newLeaseHolder(logger, storage, leaseName, holderID, opts...)

	if err := holder.hold(ctx); err != nil {
		return err
	}

	defer func() {
		if closeErr := holder.close(); closeErr != nil {
			logger.Warn(ctx, "Failed to close lease holder", log.Error(closeErr))
		}
	}()

	if cause := context.Cause(holder.context()); cause != nil {
		return cause
	}

	action(holder.context())

	return nil
}

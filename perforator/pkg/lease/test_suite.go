package lease

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func RunTests(t *testing.T, factory func() (Lease, error)) {
	logger := xlog.ForTest(t)
	// Real database latency must not consume the lease between test steps.
	integrationOptions := []LeaseOption{
		WithTTL(30 * time.Second), WithRenewInterval(200 * time.Millisecond),
		func(o *leaseOptions) { o.acquireRetryInterval = 200 * time.Millisecond },
	}

	t.Run("Lifecycle", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx := t.Context()
		holder := "holder-1"
		ttl := 5 * time.Second

		logger.Info(ctx, "Acquiring lease")
		acquired, err := s.Acquire(ctx, holder, ttl)
		require.NoError(t, err)
		require.True(t, acquired)

		logger.Info(ctx, "Renewing lease")
		renewed, err := s.Renew(ctx, holder, ttl)
		require.NoError(t, err)
		require.True(t, renewed)

		logger.Info(ctx, "Releasing lease")
		err = s.Release(ctx, holder)
		require.NoError(t, err)

		logger.Info(ctx, "Acquiring lease again after release")
		acquired, err = s.Acquire(ctx, holder, ttl)
		require.NoError(t, err)
		require.True(t, acquired)
	})

	t.Run("ConflictAndTakeover", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx := t.Context()
		holderA := "holder-a"
		holderB := "holder-b"
		ttl := 2 * time.Second

		logger.Info(ctx, "Holder A acquiring lease")
		acquired, err := s.Acquire(ctx, holderA, ttl)
		require.NoError(t, err)
		require.True(t, acquired)

		logger.Info(ctx, "Holder B trying to acquire active lease (should fail)")
		acquired, err = s.Acquire(ctx, holderB, ttl)
		require.NoError(t, err)
		require.False(t, acquired)

		logger.Info(ctx, "Waiting for lease to expire", log.Duration("ttl", ttl))
		time.Sleep(ttl)

		logger.Info(ctx, "Holder B trying to acquire expired lease (takeover)")
		acquired, err = s.Acquire(ctx, holderB, ttl)
		require.NoError(t, err)
		require.True(t, acquired)

		logger.Info(ctx, "Holder A trying to renew lost lease (zombie renew, should fail)")
		renewed, err := s.Renew(ctx, holderA, ttl)
		require.NoError(t, err)
		require.False(t, renewed)

		logger.Info(ctx, "Holder A releasing lost lease (must preserve holder B)")
		require.NoError(t, s.Release(ctx, holderA))

		acquired, err = s.Acquire(ctx, holderA, ttl)
		require.NoError(t, err)
		require.False(t, acquired, "stale release must not make holder B's lease available")

		renewed, err = s.Renew(ctx, holderB, ttl)
		require.NoError(t, err)
		require.True(t, renewed, "holder B must retain its lease after holder A releases")
	})

	t.Run("NamedLeaseLifecycle", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		renewals := make(chan struct{}, 1)
		s = &observedRenewalLease{Lease: s, renewed: renewals}
		aCtx, cancelA := context.WithCancel(ctx)
		started := make(chan context.Context, 1)
		done := make(chan struct{})
		var aErr error
		go func() {
			defer close(done)
			aErr = LockAndRun(aCtx, logger, s, func(ctx context.Context, _ string) {
				started <- ctx
				<-ctx.Done()
			}, integrationOptions...)
		}()
		defer func() { cancelA(); <-done }()
		select {
		case leaseCtx := <-started:
			require.NoError(t, leaseCtx.Err())
		case <-done:
			t.Fatalf("holder A stopped before running its action: %v", aErr)
		}

		// Observe an actual renewal instead of relying on wall-clock TTL expiry.
		select {
		case <-renewals:
		case <-done:
			t.Fatalf("holder A stopped before renewal: %v", aErr)
		case <-ctx.Done():
			t.Fatal("Timed out waiting for renewal")
		}

		bCtx, cancelB := context.WithTimeout(ctx, 2*time.Second)
		defer cancelB()
		err = LockAndRun(bCtx, logger, s, func(context.Context, string) {
			t.Error("holder B acquired while holder A was active")
		}, integrationOptions...)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		cancelA()
		<-done
		require.NoError(t, aErr)
		called := false
		err = LockAndRun(ctx, logger, s, func(ctx context.Context, _ string) {
			require.NoError(t, ctx.Err())
			called = true
		}, integrationOptions...)
		require.NoError(t, err)
		require.True(t, called)
	})

	t.Run("LockAndRun", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
		defer cancel()
		logger := xlog.ForTest(t)

		const iterations = 10
		counter := 0
		var mu sync.Mutex

		runTask := func(id string) error {
			return LockAndRun(ctx, logger, s, func(ctx context.Context, _ string) {
				mu.Lock()
				counter++
				current := counter
				mu.Unlock()

				logger.Info(ctx, "Task started", log.String("id", id), log.Int("counter", current))

				select {
				case <-ctx.Done():
				case <-time.After(200 * time.Millisecond):
				}

				mu.Lock()
				// Concurrent tasks would change the counter.
				assert.Equal(t, current, counter, "Mutual exclusion violated")
				logger.Info(ctx, "Task finished", log.String("id", id))
				mu.Unlock()
			}, integrationOptions...)
		}

		done := make(chan error, iterations)
		for i := 0; i < iterations; i++ {
			go func(id int) {
				done <- runTask(fmt.Sprintf("holder-%d", id))
			}(i)
		}

		for i := 0; i < iterations; i++ {
			select {
			case err := <-done:
				assert.NoError(t, err)
			case <-ctx.Done():
				t.Fatal("Timeout waiting for LockAndRun tasks")
			}
		}

		require.Equal(t, iterations, counter)
	})
}

// observedRenewalLease reports successful database renewals to lifecycle tests.
type observedRenewalLease struct {
	Lease
	renewed chan<- struct{}
}

func (l *observedRenewalLease) Renew(ctx context.Context, token string, ttl time.Duration) (bool, error) {
	renewed, err := l.Lease.Renew(ctx, token, ttl)
	if err == nil && renewed {
		select {
		case l.renewed <- struct{}{}:
		default:
		}
	}
	return renewed, err
}

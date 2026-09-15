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

func RunTests(t *testing.T, factory func() (Storage, error)) {
	logger := xlog.ForTest(t)

	t.Run("Lifecycle", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx := t.Context()
		name := "test-lease-lifecycle"
		holder := "holder-1"
		ttl := 5 * time.Second

		logger.Info(ctx, "Acquiring lease")
		acquired, err := s.Acquire(ctx, name, holder, ttl)
		require.NoError(t, err)
		require.True(t, acquired)

		logger.Info(ctx, "Renewing lease")
		renewed, err := s.Renew(ctx, name, holder, ttl)
		require.NoError(t, err)
		require.True(t, renewed)

		logger.Info(ctx, "Releasing lease")
		err = s.Release(ctx, name, holder)
		require.NoError(t, err)

		logger.Info(ctx, "Acquiring lease again after release")
		acquired, err = s.Acquire(ctx, name, holder, ttl)
		require.NoError(t, err)
		require.True(t, acquired)
	})

	t.Run("ConflictAndTakeover", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx := t.Context()
		name := "test-lease-conflict"
		holderA := "holder-a"
		holderB := "holder-b"
		ttl := 2 * time.Second

		logger.Info(ctx, "Holder A acquiring lease")
		acquired, err := s.Acquire(ctx, name, holderA, ttl)
		require.NoError(t, err)
		require.True(t, acquired)

		logger.Info(ctx, "Holder B trying to acquire active lease (should fail)")
		acquired, err = s.Acquire(ctx, name, holderB, ttl)
		require.NoError(t, err)
		require.False(t, acquired)

		logger.Info(ctx, "Waiting for lease to expire", log.Duration("ttl", ttl))
		time.Sleep(ttl)

		logger.Info(ctx, "Holder B trying to acquire expired lease (takeover)")
		acquired, err = s.Acquire(ctx, name, holderB, ttl)
		require.NoError(t, err)
		require.True(t, acquired)

		logger.Info(ctx, "Holder A trying to renew lost lease (zombie renew, should fail)")
		renewed, err := s.Renew(ctx, name, holderA, ttl)
		require.NoError(t, err)
		require.False(t, renewed)

		logger.Info(ctx, "Holder A releasing lost lease (must preserve holder B)")
		require.NoError(t, s.Release(ctx, name, holderA))

		acquired, err = s.Acquire(ctx, name, holderA, ttl)
		require.NoError(t, err)
		require.False(t, acquired, "stale release must not make holder B's lease available")

		renewed, err = s.Renew(ctx, name, holderB, ttl)
		require.NoError(t, err)
		require.True(t, renewed, "holder B must retain its lease after holder A releases")
	})

	t.Run("LeaseHolder", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx := t.Context()
		name := "test-lease-holder"
		holderA := "holder-a"
		holderB := "holder-b"
		ttl := time.Second

		logger := xlog.ForTest(t)
		hA := newLeaseHolder(logger, s, name, holderA, WithTTL(ttl))
		hB := newLeaseHolder(logger, s, name, holderB,
			WithTTL(ttl),
		)

		logger.Info(ctx, "Holder A holding lease LeaseHolder")
		err = hA.hold(ctx)
		require.NoError(t, err)
		require.NotNil(t, hA.context())
		require.NoError(t, hA.context().Err())

		logger.Info(ctx, "Holder B trying to hold lease LeaseHolder (should fail due to context cancellation)")
		bCtx, cancelBctx := context.WithTimeout(ctx, 2*ttl)
		defer cancelBctx()
		err = hB.hold(bCtx)
		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)

		logger.Info(ctx, "Holder A closing lease")
		err = hA.close()
		require.NoError(t, err)

		logger.Info(ctx, "Holder B holding lease LeaseHolder after A closed")
		err = hB.hold(ctx)
		require.NoError(t, err)
		require.NotNil(t, hB.context())
		require.NoError(t, hB.context().Err())

		logger.Info(ctx, "Holder B closing lease")
		err = hB.close()
		require.NoError(t, err)
	})

	t.Run("LockAndRun", func(t *testing.T) {
		s, err := factory()
		require.NoError(t, err)

		ctx := t.Context()
		name := "test-lock-and-run-sequential"
		ttl := 1 * time.Second
		logger := xlog.ForTest(t)

		const iterations = 10
		counter := 0
		var mu sync.Mutex

		runTask := func(id string) error {
			return LockAndRun(ctx, logger, s, name, id, func(ctx context.Context) {
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
				// If tasks were running in parallel, counter would have been incremented by another task
				assert.Equal(t, current, counter, "Mutual exclusion violated")
				logger.Info(ctx, "Task finished", log.String("id", id))
				mu.Unlock()
			}, WithTTL(ttl))
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
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for LockAndRun tasks")
			}
		}

		require.Equal(t, iterations, counter)
	})
}

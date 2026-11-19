package deploysystem

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

type snapshotSourceFunc func(context.Context) (Snapshot, error)

func (f snapshotSourceFunc) Snapshot(ctx context.Context) (Snapshot, error) {
	return f(ctx)
}

func TestPollingSnapshotAdapterConfiguration(t *testing.T) {
	source := snapshotSourceFunc(func(context.Context) (Snapshot, error) { return Snapshot{}, nil })
	for _, tc := range []struct {
		name     string
		source   Source
		interval time.Duration
		timeout  time.Duration
	}{
		{"nil source", nil, time.Second, time.Second},
		{"zero interval", source, 0, time.Second},
		{"negative interval", source, -time.Second, time.Second},
		{"zero timeout", source, time.Second, 0},
		{"negative timeout", source, time.Second, -time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPollingSnapshotAdapter(tc.source, tc.interval, tc.timeout)
			require.Error(t, err)
		})
	}
}

func TestPollingSnapshotAdapterCadence(t *testing.T) {
	for _, sourceErr := range []error{nil, errors.New("discovery failed")} {
		name := "success"
		if sourceErr != nil {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				want := testSnapshot()
				var calls []time.Time
				var requestCtx context.Context
				source := snapshotSourceFunc(func(ctx context.Context) (Snapshot, error) {
					calls = append(calls, time.Now())
					deadline, ok := ctx.Deadline()
					require.True(t, ok)
					require.Equal(t, time.Now().Add(5*time.Second), deadline)
					requestCtx = ctx
					time.Sleep(time.Second)
					return want, sourceErr
				})
				adapter, err := NewPollingSnapshotAdapter(source, 10*time.Second, 5*time.Second)
				require.NoError(t, err)
				for range 2 {
					got, err := adapter.Snapshot(t.Context())
					require.Equal(t, sourceErr, err)
					require.Equal(t, want, got, "adapter must preserve the source observation")
					require.ErrorIs(t, requestCtx.Err(), context.Canceled)
				}
				require.Equal(t, []time.Time{start, start.Add(11 * time.Second)}, calls)

				// A late caller should not wait another full interval.
				time.Sleep(20 * time.Second)
				now := time.Now()
				_, err = adapter.Snapshot(t.Context())
				require.Equal(t, sourceErr, err)
				require.Equal(t, now, calls[2])
			})
		})
	}
}

func TestPollingSnapshotAdapterRequestTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		var calls []time.Time
		adapter, err := NewPollingSnapshotAdapter(snapshotSourceFunc(func(ctx context.Context) (Snapshot, error) {
			calls = append(calls, time.Now())
			<-ctx.Done()
			return Snapshot{}, ctx.Err()
		}), 10*time.Second, 3*time.Second)
		require.NoError(t, err)
		for range 2 {
			_, err := adapter.Snapshot(t.Context())
			require.ErrorIs(t, err, context.DeadlineExceeded)
		}
		require.Equal(t, []time.Time{start, start.Add(13 * time.Second)}, calls)
		require.Equal(t, 16*time.Second, time.Since(start))
	})
}

func TestPollingSnapshotAdapterCancelWaiting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		calls := 0
		adapter, err := NewPollingSnapshotAdapter(snapshotSourceFunc(func(context.Context) (Snapshot, error) {
			calls++
			return testSnapshot(), nil
		}), 10*time.Second, time.Second)
		require.NoError(t, err)
		_, err = adapter.Snapshot(t.Context())
		require.NoError(t, err)

		cause := errors.New("cancel poll wait")
		ctx, cancel := context.WithTimeoutCause(t.Context(), 2*time.Second, cause)
		defer cancel()
		_, err = adapter.Snapshot(ctx)
		require.ErrorIs(t, err, cause)
		require.Equal(t, 1, calls)
		require.Equal(t, 2*time.Second, time.Since(start))

		// Cancelling a wait does not postpone the scheduled poll.
		_, err = adapter.Snapshot(t.Context())
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, 10*time.Second, time.Since(start))
	})
}

func TestPollingSnapshotAdapterCanceledContext(t *testing.T) {
	cause := errors.New("cancel discovery request")
	adapter, err := NewPollingSnapshotAdapter(snapshotSourceFunc(func(ctx context.Context) (Snapshot, error) {
		return Snapshot{}, context.Cause(ctx)
	}), time.Second, time.Second)
	require.NoError(t, err)
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)
	_, err = adapter.Snapshot(ctx)
	require.ErrorIs(t, err, cause)
}

func TestPollingSnapshotAdapterParentDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		start := time.Now()
		adapter, err := NewPollingSnapshotAdapter(snapshotSourceFunc(func(ctx context.Context) (Snapshot, error) {
			<-ctx.Done()
			return Snapshot{}, ctx.Err()
		}), time.Second, time.Minute)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		defer cancel()
		_, err = adapter.Snapshot(ctx)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, 2*time.Second, time.Since(start))
	})
}

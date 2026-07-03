package boundedcache

import (
	"context"
	"errors"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func bg() context.Context { return context.Background() }

// discardLog records the values the cache discarded on eviction.
type discardLog struct {
	mu     sync.Mutex
	values []string
}

func (d *discardLog) add(value string) {
	d.mu.Lock()
	d.values = append(d.values, value)
	d.mu.Unlock()
}
func (d *discardLog) list() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.values...)
}

// newTestCache makes a string-valued cache whose evicted values land in the log.
func newTestCache(capacity int64) (*Cache[string, string], *discardLog) {
	d := &discardLog{}
	return New[string, string](capacity, d.add), d
}

// fillKey charges size and yields the key as the held value.
func fillKey(size int64) FillFunc[string, string] {
	return func(ctx context.Context, e *Entry[string, string]) (string, error) {
		if err := e.Charge(ctx, size); err != nil {
			return "", err
		}
		return e.key, nil
	}
}

func TestCache_MissThenHit(t *testing.T) {
	a, _ := newTestCache(100)
	var fills atomic.Int32
	fill := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		fills.Add(1)
		return fillKey(10)(ctx, e)
	}
	l, err := a.GetOrFill(bg(), "a", fill)
	require.NoError(t, err)
	require.Equal(t, int64(10), l.Size())
	require.Equal(t, "a", l.Value())
	l.Release()

	l2, err := a.GetOrFill(bg(), "a", fill)
	require.NoError(t, err)
	l2.Release()
	require.Equal(t, int32(1), fills.Load()) // hit
}

func TestCache_SingleFlight(t *testing.T) {
	a, _ := newTestCache(100)
	var fills atomic.Int32
	release := make(chan struct{})
	fill := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		fills.Add(1)
		<-release
		return fillKey(10)(ctx, e)
	}
	const k = 8
	ls := make([]*Lease[string, string], k)
	errs := make([]error, k)
	var wg sync.WaitGroup
	for i := 0; i < k; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); ls[i], errs[i] = a.GetOrFill(bg(), "a", fill) }(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()
	for i := 0; i < k; i++ {
		require.NoError(t, errs[i])
		ls[i].Release()
	}
	require.Equal(t, int32(1), fills.Load())
}

func TestCache_EvictionLRU(t *testing.T) {
	a, d := newTestCache(25)
	for _, k := range []string{"a", "b"} {
		l, err := a.GetOrFill(bg(), k, fillKey(10))
		require.NoError(t, err)
		l.Release()
	}
	l, err := a.GetOrFill(bg(), "c", fillKey(10)) // evicts the coldest (a)
	require.NoError(t, err)
	l.Release()
	require.Equal(t, []string{"a"}, d.list())
}

func TestCache_RefGuardsEviction(t *testing.T) {
	a, d := newTestCache(25)
	refA, err := a.GetOrFill(bg(), "a", fillKey(10)) // kept referenced
	require.NoError(t, err)
	b, _ := a.GetOrFill(bg(), "b", fillKey(10))
	b.Release()                                 // idle
	c, _ := a.GetOrFill(bg(), "c", fillKey(10)) // evicts idle b, not referenced a
	c.Release()
	require.Equal(t, []string{"b"}, d.list())
	refA.Release()
}

func TestCache_ChargeBlocksUntilRelease(t *testing.T) {
	a, _ := newTestCache(10)
	refA, _ := a.GetOrFill(bg(), "a", fillKey(10))
	done := make(chan error, 1)
	go func() {
		l, err := a.GetOrFill(bg(), "b", fillKey(10))
		if err == nil {
			l.Release()
		}
		done <- err
	}()
	select {
	case <-done:
		t.Fatal("b must block while a holds all 10")
	case <-time.After(50 * time.Millisecond):
	}
	refA.Release()
	require.NoError(t, <-done)
}

func TestCache_TryChargeWouldBlock(t *testing.T) {
	a, _ := newTestCache(10)
	refA, _ := a.GetOrFill(bg(), "a", fillKey(10))
	_, err := a.GetOrFill(bg(), "b", func(_ context.Context, e *Entry[string, string]) (string, error) {
		return "", e.TryCharge(10)
	})
	require.ErrorIs(t, err, ErrWouldBlock)
	refA.Release()
}

func TestCache_TooLarge(t *testing.T) {
	a, _ := newTestCache(10)
	_, err := a.GetOrFill(bg(), "a", func(ctx context.Context, e *Entry[string, string]) (string, error) {
		return "", e.Charge(ctx, 11)
	})
	require.ErrorIs(t, err, ErrTooLarge)
}

func TestCache_FillErrorNotCached(t *testing.T) {
	a, _ := newTestCache(100)
	boom := errors.New("boom")
	fail := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		if err := e.Charge(ctx, 10); err != nil {
			return "", err
		}
		return "", boom
	}
	_, err := a.GetOrFill(bg(), "a", fail)
	require.ErrorIs(t, err, boom)
	_, err = a.GetOrFill(bg(), "a", fail) // not cached → the caller can re-run it
	require.ErrorIs(t, err, boom)
	require.Equal(t, int64(0), a.Used()) // charge refunded
}

// TestCache_OnEvictNotUnderLock verifies onEvict runs without the cache lock, so
// it may re-enter the cache (here, read Used) without deadlocking.
func TestCache_OnEvictNotUnderLock(t *testing.T) {
	var a *Cache[string, string]
	a = New[string, string](25, func(string) {
		a.Used() // re-enters c.mu; would deadlock if onEvict ran under it
	})
	for _, k := range []string{"a", "b"} {
		l, err := a.GetOrFill(bg(), k, fillKey(10))
		require.NoError(t, err)
		l.Release()
	}
	done := make(chan struct{})
	go func() {
		l, err := a.GetOrFill(bg(), "c", fillKey(10)) // evicts a → onEvict → a.Used()
		require.NoError(t, err)
		l.Release()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("onEvict deadlocked — it must not run under the cache lock")
	}
}

func TestCache_NewRejectsNonPositiveCapacity(t *testing.T) {
	require.Panics(t, func() { New[string, string](0, nil) })
	require.Panics(t, func() { New[string, string](-1, nil) })
}

// TestCache_NegativeChargeRejected verifies a negative charge watermark is
// rejected rather than silently treated as a no-op.
func TestCache_NegativeChargeRejected(t *testing.T) {
	a, _ := newTestCache(100)
	_, err := a.GetOrFill(bg(), "a", func(ctx context.Context, e *Entry[string, string]) (string, error) {
		return "", e.Charge(ctx, -1)
	})
	require.ErrorIs(t, err, errNegativeSize)

	_, err = a.GetOrFill(bg(), "b", func(_ context.Context, e *Entry[string, string]) (string, error) {
		return "", e.TryCharge(-1)
	})
	require.ErrorIs(t, err, errNegativeSize)
}

// TestCache_FillErrorPropagatesToAllWaiters verifies a failing fill is run once
// and its error is handed to every coalesced waiter — no internal retry, no
// download amplification.
func TestCache_FillErrorPropagatesToAllWaiters(t *testing.T) {
	a, _ := newTestCache(100)
	boom := errors.New("boom")
	release := make(chan struct{})
	var fills atomic.Int32
	fail := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		fills.Add(1)
		<-release
		return "", boom
	}
	const k = 8
	errs := make([]error, k)
	var wg sync.WaitGroup
	for i := 0; i < k; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); _, errs[i] = a.GetOrFill(bg(), "a", fail) }(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	for i := 0; i < k; i++ {
		require.ErrorIs(t, errs[i], boom) // every waiter sees the real error
	}
	require.Equal(t, int32(1), fills.Load()) // fill ran exactly once
	require.Equal(t, int64(0), a.Used())
}

func TestCache_CtxCancelWhileWaiting(t *testing.T) {
	a, _ := newTestCache(10)
	refA, _ := a.GetOrFill(bg(), "a", fillKey(10))
	ctx, cancel := context.WithCancel(bg())
	go func() { time.Sleep(20 * time.Millisecond); cancel() }()
	_, err := a.GetOrFill(ctx, "b", fillKey(10))
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, int64(10), a.Used()) // only a; b's failed fill was cleaned up
	refA.Release()
}

// TestCache_HitIgnoresCanceledCtx verifies a ready hit resolves even when the
// caller's context is already canceled.
func TestCache_HitIgnoresCanceledCtx(t *testing.T) {
	a, _ := newTestCache(100)
	l, err := a.GetOrFill(bg(), "a", fillKey(10))
	require.NoError(t, err)
	l.Release() // ready and idle

	ctx, cancel := context.WithCancel(bg())
	cancel()
	for i := 0; i < 50; i++ { // the race is nondeterministic; hammer it
		l2, err := a.GetOrFill(ctx, "a", fillKey(10))
		require.NoError(t, err)
		require.Equal(t, "a", l2.Value())
		l2.Release()
	}
}

// TestCache_OwnedChargeNeverBlocks verifies that re-charging space the entry
// already owns neither blocks behind another charge holding the turn nor fails.
func TestCache_OwnedChargeNeverBlocks(t *testing.T) {
	a, _ := newTestCache(10)
	charged := make(chan struct{})
	proceed := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := a.GetOrFill(bg(), "a", func(ctx context.Context, e *Entry[string, string]) (string, error) {
			if err := e.Charge(ctx, 10); err != nil {
				return "", err
			}
			close(charged)
			<-proceed
			if err := e.Charge(ctx, 10); err != nil { // already owned
				return "", err
			}
			if err := e.TryCharge(10); err != nil { // already owned
				return "", err
			}
			return e.key, nil
		})
		done <- err
	}()
	<-charged

	bCtx, bCancel := context.WithCancel(bg())
	defer bCancel()
	go func() { _, _ = a.GetOrFill(bCtx, "b", fillKey(10)) }() // blocks holding the turn
	time.Sleep(30 * time.Millisecond)

	close(proceed)
	require.NoError(t, <-done)
}

// TestCache_FirstCallerCancelable verifies the caller that started the fill can
// give up individually: its GetOrFill returns its ctx error while the fill,
// kept alive by another waiter, completes and serves that waiter.
func TestCache_FirstCallerCancelable(t *testing.T) {
	a, _ := newTestCache(100)
	var fills atomic.Int32
	release := make(chan struct{})
	fill := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		fills.Add(1)
		<-release
		return fillKey(10)(ctx, e)
	}

	firstCtx, cancel := context.WithCancel(bg())
	firstDone := make(chan error, 1)
	go func() { _, err := a.GetOrFill(firstCtx, "a", fill); firstDone <- err }()

	waiterDone := make(chan error, 1)
	go func() {
		l, err := a.GetOrFill(bg(), "a", fill)
		if err == nil {
			l.Release()
		}
		waiterDone <- err
	}()
	time.Sleep(50 * time.Millisecond) // both are attached to the same fill

	cancel()
	require.ErrorIs(t, <-firstDone, context.Canceled)

	close(release)
	require.NoError(t, <-waiterDone)
	require.Equal(t, int32(1), fills.Load())
}

// TestCache_AbandonedFillCanceled verifies that when the last waiter gives up,
// the fill is canceled, its charge refunded, and a later caller starts fresh.
func TestCache_AbandonedFillCanceled(t *testing.T) {
	a, _ := newTestCache(100)
	charged := make(chan struct{})
	fill := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		if err := e.Charge(ctx, 10); err != nil {
			return "", err
		}
		close(charged)
		<-ctx.Done() // hold the budget until the cache abandons us
		return "", ctx.Err()
	}

	ctx, cancel := context.WithCancel(bg())
	callerErr := make(chan error, 1)
	go func() { _, err := a.GetOrFill(ctx, "a", fill); callerErr <- err }()
	<-charged
	require.Equal(t, int64(10), a.Used())

	cancel()
	require.ErrorIs(t, <-callerErr, context.Canceled)
	require.Eventually(t, func() bool {
		return a.Used() == 0 && a.Len() == 0 // canceled, refunded, not cached
	}, 2*time.Second, time.Millisecond)

	l, err := a.GetOrFill(bg(), "a", fillKey(10)) // a fresh fill works
	require.NoError(t, err)
	l.Release()
}

// TestCache_PublishAfterAbandonDiscards verifies a fill that ignores its
// cancellation and completes anyway has its value discarded, not cached.
func TestCache_PublishAfterAbandonDiscards(t *testing.T) {
	a, d := newTestCache(100)
	charged := make(chan struct{})
	finish := make(chan struct{})
	fill := func(ctx context.Context, e *Entry[string, string]) (string, error) {
		if err := e.Charge(ctx, 10); err != nil {
			return "", err
		}
		close(charged)
		<-finish // ignore cancellation and complete anyway
		return "late", nil
	}

	ctx, cancel := context.WithCancel(bg())
	callerErr := make(chan error, 1)
	go func() { _, err := a.GetOrFill(ctx, "a", fill); callerErr <- err }()
	<-charged
	cancel()
	require.ErrorIs(t, <-callerErr, context.Canceled)

	close(finish)
	require.Eventually(t, func() bool {
		return slices.Contains(d.list(), "late") // rejected value was disposed
	}, 2*time.Second, time.Millisecond)
	require.Equal(t, int64(0), a.Used())
	require.Equal(t, int64(0), a.Len())
}

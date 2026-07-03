package filecache

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/atomicfs"
)

func bg() context.Context { return context.Background() }

func newTestCache(t *testing.T, capacity int64) *Cache {
	c, err := newCache(t.TempDir(), capacity, nil)
	require.NoError(t, err)
	return c
}

// fillN charges n then writes exactly n bytes.
func fillN(n int64) FillFunc {
	return func(ctx context.Context, w Writer) error {
		if err := w.Charge(ctx, n); err != nil {
			return err
		}
		_, err := w.WriteAt(make([]byte, n), 0)
		return err
	}
}

func cachedSize(c *Cache, key string) (int64, bool) {
	info, err := os.Stat(filepath.Join(c.dir, key))
	if err != nil {
		return 0, false
	}
	return info.Size(), true
}

func dirState(t *testing.T, c *Cache) (final []string, tmp int) {
	t.Helper()
	des, err := os.ReadDir(c.dir)
	require.NoError(t, err)
	for _, de := range des {
		if atomicfs.IsTmp(de.Name()) {
			tmp++
		} else {
			final = append(final, de.Name())
		}
	}
	return final, tmp
}

func TestMissThenHit(t *testing.T) {
	c := newTestCache(t, 100)
	var fills atomic.Int32
	fill := func(ctx context.Context, w Writer) error {
		fills.Add(1)
		return fillN(10)(ctx, w)
	}

	ref, err := c.GetOrFill(bg(), "a", fill)
	require.NoError(t, err)
	require.Equal(t, int64(10), ref.Size())
	require.True(t, strings.HasSuffix(ref.Path(), string(os.PathSeparator)+"a"))
	sz, ok := cachedSize(c, "a")
	require.True(t, ok)
	require.Equal(t, int64(10), sz)
	ref.Release()

	ref2, err := c.GetOrFill(bg(), "a", fill)
	require.NoError(t, err)
	ref2.Release()
	require.Equal(t, int32(1), fills.Load()) // hit, no second fill
}

func TestSingleFlight(t *testing.T) {
	c := newTestCache(t, 100)
	var fills atomic.Int32
	release := make(chan struct{})
	fill := func(ctx context.Context, w Writer) error {
		fills.Add(1)
		<-release
		return fillN(10)(ctx, w)
	}

	const k = 8
	refs := make([]*Ref, k)
	errs := make([]error, k)
	var wg sync.WaitGroup
	for i := 0; i < k; i++ {
		wg.Add(1)
		go func(i int) { defer wg.Done(); refs[i], errs[i] = c.GetOrFill(bg(), "a", fill) }(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	for i := 0; i < k; i++ {
		require.NoError(t, errs[i])
		refs[i].Release()
	}
	require.Equal(t, int32(1), fills.Load())
}

func TestEvictionDeletesFile(t *testing.T) {
	c := newTestCache(t, 25) // two 10-byte files fit, three don't
	for _, k := range []string{"a", "b"} {
		ref, err := c.GetOrFill(bg(), k, fillN(10))
		require.NoError(t, err)
		ref.Release()
	}
	ref, err := c.GetOrFill(bg(), "c", fillN(10)) // evicts the coldest (a)
	require.NoError(t, err)
	ref.Release()

	_, okA := cachedSize(c, "a")
	_, okB := cachedSize(c, "b")
	_, okC := cachedSize(c, "c")
	require.False(t, okA) // file deleted on eviction
	require.True(t, okB)
	require.True(t, okC)
}

func TestRefGuardsEviction(t *testing.T) {
	c := newTestCache(t, 25)
	refA, err := c.GetOrFill(bg(), "a", fillN(10)) // kept referenced
	require.NoError(t, err)
	refB, err := c.GetOrFill(bg(), "b", fillN(10))
	require.NoError(t, err)
	refB.Release()

	refC, err := c.GetOrFill(bg(), "c", fillN(10)) // evicts idle b, not referenced a
	require.NoError(t, err)
	refC.Release()

	_, okA := cachedSize(c, "a")
	_, okB := cachedSize(c, "b")
	require.True(t, okA)
	require.False(t, okB)
	refA.Release()
}

func TestChargeBlocksUntilRelease(t *testing.T) {
	c := newTestCache(t, 10)
	refA, err := c.GetOrFill(bg(), "a", fillN(10))
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		ref, err := c.GetOrFill(bg(), "b", fillN(10))
		if err == nil {
			ref.Release()
		}
		done <- err
	}()

	select {
	case <-done:
		t.Fatal("b must block while a holds all 10")
	case <-time.After(50 * time.Millisecond):
	}

	refA.Release() // frees a → b proceeds
	require.NoError(t, <-done)
}

func TestBareWriteWouldBlock(t *testing.T) {
	c := newTestCache(t, 10)
	refA, err := c.GetOrFill(bg(), "a", fillN(10))
	require.NoError(t, err)
	_, err = c.GetOrFill(bg(), "b", func(_ context.Context, w Writer) error {
		_, werr := w.WriteAt(make([]byte, 10), 0) // bare WriteAt can't wait for space
		return werr
	})
	require.ErrorIs(t, err, ErrWouldBlock)
	refA.Release()
}

func TestChargeTooLarge(t *testing.T) {
	c := newTestCache(t, 10)
	_, err := c.GetOrFill(bg(), "a", func(ctx context.Context, w Writer) error {
		return w.Charge(ctx, 11)
	})
	require.ErrorIs(t, err, ErrTooLarge)
}

func TestBareWriteGrows(t *testing.T) {
	c := newTestCache(t, 100)
	ref, err := c.GetOrFill(bg(), "a", func(_ context.Context, w Writer) error {
		_, err := w.WriteAt(make([]byte, 10), 0) // no Charge; WriteAt charges as it grows
		return err
	})
	require.NoError(t, err)
	require.Equal(t, int64(10), ref.Size())
	ref.Release()
}

func TestBareWriteTooLarge(t *testing.T) {
	c := newTestCache(t, 10)
	_, err := c.GetOrFill(bg(), "a", func(_ context.Context, w Writer) error {
		_, err := w.WriteAt(make([]byte, 15), 0) // past capacity
		return err
	})
	require.ErrorIs(t, err, ErrTooLarge)
}

func TestFillErrorNotCachedNoLeak(t *testing.T) {
	c := newTestCache(t, 100)
	boom := errors.New("boom")
	failing := func(ctx context.Context, w Writer) error {
		if err := w.Charge(ctx, 10); err != nil {
			return err
		}
		return boom
	}
	_, err := c.GetOrFill(bg(), "a", failing)
	require.ErrorIs(t, err, boom)
	_, err = c.GetOrFill(bg(), "a", failing) // not cached → retried
	require.ErrorIs(t, err, boom)

	require.Equal(t, int64(0), c.alloc.Used()) // charge refunded
	final, tmp := dirState(t, c)
	require.Empty(t, final)
	require.Zero(t, tmp) // temp file cleaned up
}

func TestFillENOSPCNotCached(t *testing.T) {
	c := newTestCache(t, 100)
	_, err := c.GetOrFill(bg(), "a", func(ctx context.Context, w Writer) error {
		if err := w.Charge(ctx, 10); err != nil {
			return err
		}
		return syscall.ENOSPC // injected at the fill boundary — no fs mock needed
	})
	require.ErrorIs(t, err, syscall.ENOSPC)
	require.Equal(t, int64(0), c.alloc.Used())
	final, tmp := dirState(t, c)
	require.Empty(t, final)
	require.Zero(t, tmp)
}

func TestCoalesceHoldsAcrossFillerRelease(t *testing.T) {
	c := newTestCache(t, 10) // room for exactly one
	filling := make(chan struct{})
	proceed := make(chan struct{})
	fillerDone := make(chan *Ref, 1)
	go func() {
		ref, err := c.GetOrFill(bg(), "a", func(ctx context.Context, w Writer) error {
			close(filling)
			<-proceed
			return fillN(10)(ctx, w)
		})
		require.NoError(t, err)
		fillerDone <- ref
	}()
	<-filling

	coalDone := make(chan *Ref, 1)
	go func() {
		ref, err := c.GetOrFill(bg(), "a", func(context.Context, Writer) error {
			t.Error("coalescer must not run the fill")
			return nil
		})
		require.NoError(t, err)
		coalDone <- ref
	}()
	time.Sleep(50 * time.Millisecond)

	close(proceed)
	(<-fillerDone).Release() // filler drops its ref immediately
	cref := <-coalDone       // coalescer holds the ref the filler took for it

	// a is still pinned by the coalescer, so a competing fill can't evict it.
	ctx, cancel := context.WithTimeout(bg(), 80*time.Millisecond)
	_, berr := c.GetOrFill(ctx, "b", fillN(10))
	cancel()
	require.ErrorIs(t, berr, context.DeadlineExceeded)

	_, ok := cachedSize(c, "a")
	require.True(t, ok)
	cref.Release()
}

func TestBootRecovery(t *testing.T) {
	dir := t.TempDir()
	c1, err := newCache(dir, 100, nil)
	require.NoError(t, err)
	ref, err := c1.GetOrFill(bg(), "x", fillN(10))
	require.NoError(t, err)
	ref.Release()

	// A leftover partial fill, as a crash would leave it.
	stale := filepath.Join(dir, "y.tmp-3810410072")
	require.NoError(t, os.WriteFile(stale, []byte{1, 2, 3}, 0o600))

	c2, err := newCache(dir, 100, nil) // reopen the same dir
	require.NoError(t, err)
	var filled bool
	ref2, err := c2.GetOrFill(bg(), "x", func(context.Context, Writer) error {
		filled = true
		return nil
	})
	require.NoError(t, err)
	require.False(t, filled) // recovered from disk, not refilled
	require.Equal(t, int64(10), ref2.Size())
	ref2.Release()

	_, err = os.Stat(stale)
	require.ErrorIs(t, err, os.ErrNotExist) // the partial fill was swept
	require.Equal(t, int64(1), c2.alloc.Len())
}

func TestStress(t *testing.T) {
	c := newTestCache(t, 100)
	const (
		workers  = 16
		iters    = 200
		keys     = 12
		fileSize = 10
	)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				key := fmt.Sprintf("k%d", (seed+i)%keys)
				ctx, cancel := context.WithTimeout(bg(), 2*time.Second)
				ref, err := c.GetOrFill(ctx, key, fillN(fileSize))
				cancel()
				if err != nil {
					continue
				}
				time.Sleep(time.Millisecond)
				ref.Release()
			}
		}(w)
	}
	wg.Wait()

	require.LessOrEqual(t, c.alloc.Used(), int64(100)) // budget invariant held throughout
}

package collector

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/lease"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type fakeLeaseStorage struct {
	mu       sync.Mutex
	holder   string
	lost     atomic.Bool
	released chan struct{}
	waiting  chan struct{}
}

func (s *fakeLeaseStorage) Acquire(ctx context.Context, _, holder string, _ time.Duration) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.holder != "" {
		select {
		case s.waiting <- struct{}{}:
		default:
		}
		return false, nil
	}
	s.holder = holder
	return true, nil
}

func (s *fakeLeaseStorage) Renew(_ context.Context, _, holder string, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.holder == holder && !s.lost.Load(), nil
}

func (s *fakeLeaseStorage) Release(_ context.Context, _, holder string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.holder == holder {
		s.holder = ""
	}
	select {
	case s.released <- struct{}{}:
	default:
	}
	return nil
}

func testGC(t *testing.T, ls lease.Storage, f func(context.Context) error) *GC {
	t.Helper()
	st := &stubStorage{collect: func(ctx context.Context) ([]*storage.ObjectMeta, error) {
		return nil, f(ctx)
	}}
	c, _ := testStorageGC(t, st, config.Binary, 1)
	return &GC{collectors: []*storageGC{c}, l: xlog.ForTest(t),
		leaseStorage: ls, leaseName: "test_gc", leaseTTL: 90 * time.Millisecond}
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for GC")
		var zero T
		return zero
	}
}

func TestGCLeaseExclusionAndHandoff(t *testing.T) {
	ls := &fakeLeaseStorage{waiting: make(chan struct{}, 1), released: make(chan struct{}, 2)}
	startA, startB := make(chan struct{}), make(chan struct{})
	run := func(start chan struct{}) func(context.Context) error {
		return func(ctx context.Context) error { close(start); <-ctx.Done(); return ctx.Err() }
	}
	a, b := testGC(t, ls, run(startA)), testGC(t, ls, run(startB))
	ctxA, cancelA := context.WithCancel(t.Context())
	defer cancelA()
	ctxB, cancelB := context.WithCancel(t.Context())
	defer cancelB()
	doneA, doneB := make(chan error, 1), make(chan error, 1)
	go func() { doneA <- a.Run(ctxA, time.Minute) }()
	await(t, startA)
	go func() { doneB <- b.Run(ctxB, time.Minute) }()
	await(t, ls.waiting)
	select {
	case <-startB:
		t.Fatal("standby started collectors without the lease")
	default:
	}
	cancelA()
	require.ErrorIs(t, await(t, doneA), context.Canceled)
	await(t, startB)
	cancelB()
	require.ErrorIs(t, await(t, doneB), context.Canceled)
}

func TestGCLeaseLossDrainsCollectorsBeforeRelease(t *testing.T) {
	ls := &fakeLeaseStorage{released: make(chan struct{}, 1)}
	started, canceled, drain := make(chan struct{}), make(chan struct{}), make(chan struct{})
	defer func() {
		select {
		case <-drain:
		default:
			close(drain)
		}
	}()
	g := testGC(t, ls, func(ctx context.Context) error {
		close(started)
		<-ctx.Done()
		close(canceled)
		<-drain
		return ctx.Err()
	})
	done := make(chan error, 1)
	go func() { done <- g.Run(t.Context(), time.Minute) }()
	await(t, started)
	ls.lost.Store(true)
	await(t, canceled)
	select {
	case <-ls.released:
		t.Fatal("released lease before collector returned")
	default:
	}
	close(drain)
	require.ErrorIs(t, await(t, done), lease.ErrLeaseLost)
	await(t, ls.released)
}

func TestGCWithoutLeaseStorage(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	called := false
	g := testGC(t, nil, func(context.Context) error {
		called = true
		cancel()
		return nil
	})
	require.ErrorIs(t, g.Run(ctx, time.Minute), context.Canceled)
	require.True(t, called)
	require.Error(t, g.Run(t.Context(), 0))
}

package collector

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	yzap "github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type stubStorage struct {
	collect func(context.Context) ([]*storage.ObjectMeta, error)
	delete  func(context.Context, []string) error
}

func (s *stubStorage) CollectExpired(ctx context.Context, _ time.Duration, _ *util.Pagination) ([]*storage.ObjectMeta, error) {
	return s.collect(ctx)
}

func (s *stubStorage) Delete(ctx context.Context, ids []string) error {
	return s.delete(ctx, ids)
}

func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for GC operation")
	}
}

func awaitRun(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for GC shutdown")
		return nil
	}
}

func TestStorageGCIndependentCycles(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	blocked := make(chan struct{})
	slow := &stubStorage{collect: func(ctx context.Context) ([]*storage.ObjectMeta, error) {
		close(blocked)
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	var active atomic.Int32
	var overlap atomic.Bool
	var calls atomic.Int32
	observed := make(chan time.Time, 4)
	fast := &stubStorage{collect: func(ctx context.Context) ([]*storage.ObjectMeta, error) {
		if active.Add(1) != 1 {
			overlap.Store(true)
		}
		defer active.Add(-1)
		select {
		case <-blocked:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		n := calls.Add(1)
		if n == 1 || n > 6 {
			select {
			case observed <- time.Now():
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		// The first iteration fails. Later iterations must still run independently.
		if n <= 6 {
			return nil, errors.New("temporary read failure")
		}
		return nil, nil
	}}
	slowGC, _ := testStorageGC(t, slow, config.Binary, 1)
	fastGC, _ := testStorageGC(t, fast, config.GSYM, 1)
	gc := &GC{collectors: []*storageGC{slowGC, fastGC}}
	const interval = 10 * time.Millisecond
	done := make(chan error, 1)
	go func() { done <- gc.Run(ctx, interval) }()
	var previous time.Time
	for i := 0; i < 3; i++ {
		select {
		case now := <-observed:
			if i > 0 {
				require.GreaterOrEqual(t, now.Sub(previous), interval)
			}
			previous = now
		case <-time.After(5 * time.Second):
			t.Fatal("one storage blocked the other storage's iterations")
		}
	}
	cancel()
	require.ErrorIs(t, awaitRun(t, done), context.Canceled)
	require.False(t, overlap.Load())
}

func TestStorageGCCanceledBeforeRun(t *testing.T) {
	cause := errors.New("GC stopped")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(cause)
	var reads atomic.Int32
	s := &stubStorage{collect: func(context.Context) ([]*storage.ObjectMeta, error) {
		reads.Add(1)
		return nil, nil
	}}
	c, _ := testStorageGC(t, s, config.Binary, 1)
	require.ErrorIs(t, c.run(ctx, time.Hour), cause)
	require.ErrorIs(t, c.collect(ctx), cause)
	require.Zero(t, reads.Load())
}

func TestStorageGCCancellationInterruptsInterval(t *testing.T) {
	cause := errors.New("GC stopped")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	var reads atomic.Int32
	s := &stubStorage{collect: func(context.Context) ([]*storage.ObjectMeta, error) {
		reads.Add(1)
		return nil, nil
	}}
	c, _ := testStorageGC(t, s, config.Binary, 1)
	core, logs := observer.New(zapcore.InfoLevel)
	c.l = xlog.Wrap(&yzap.Logger{L: zap.New(core)})
	done := make(chan error, 1)
	go func() { done <- c.run(ctx, time.Hour) }()
	require.Eventually(t, func() bool {
		return logs.FilterMessage("Waiting before next GC iteration").Len() > 0
	}, 5*time.Second, time.Millisecond)
	cancel(cause)
	require.ErrorIs(t, awaitRun(t, done), cause)
	require.EqualValues(t, 1, reads.Load())
}

func TestStorageGCDoesNotDeleteCanceledPage(t *testing.T) {
	cause := errors.New("GC stopped")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	deleted := false
	s := &stubStorage{
		collect: func(context.Context) ([]*storage.ObjectMeta, error) {
			cancel(cause)
			return []*storage.ObjectMeta{{ID: "expired"}}, nil
		},
		delete: func(context.Context, []string) error { deleted = true; return nil },
	}
	c, _ := testStorageGC(t, s, config.Binary, 1)
	empty, err := c.processPage(ctx, &util.Pagination{Limit: 1})
	require.ErrorIs(t, err, cause)
	require.False(t, empty)
	require.False(t, deleted)
}

func TestStorageGCCancellationDrainsOperation(t *testing.T) {
	for _, operation := range []string{"read", "delete"} {
		t.Run(operation, func(t *testing.T) {
			cause := errors.New("GC stopped")
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)
			started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			unblock := func() { releaseOnce.Do(func() { close(release) }) }
			defer unblock()
			var reads, deletes atomic.Int32
			block := func(ctx context.Context) {
				close(started)
				<-ctx.Done()
				close(canceled)
				<-release
			}
			s := &stubStorage{
				collect: func(ctx context.Context) ([]*storage.ObjectMeta, error) {
					reads.Add(1)
					if operation == "read" {
						block(ctx)
					}
					return []*storage.ObjectMeta{{ID: "expired", LastUsedTimestamp: time.Now().Add(-48 * time.Hour)}}, nil
				},
				delete: func(ctx context.Context, _ []string) error {
					deletes.Add(1)
					block(ctx)
					return ctx.Err()
				},
			}
			c, registry := testStorageGC(t, s, config.Binary, 1)
			done := make(chan error, 1)
			go func() { done <- c.run(ctx, time.Hour) }()
			awaitSignal(t, started)
			cancel(cause)
			awaitSignal(t, canceled)
			select {
			case <-done:
				t.Fatal("run returned before the active operation finished")
			default:
			}
			unblock()
			require.ErrorIs(t, awaitRun(t, done), cause)
			require.EqualValues(t, 1, reads.Load())
			if operation == "read" {
				require.Zero(t, deletes.Load())
			} else {
				require.EqualValues(t, 1, deletes.Load())
			}
			busy, ok := registry.GetIntGauge("busy.gauge")
			require.True(t, ok)
			require.Zero(t, busy.Value.Load())
		})
	}
}

func TestGCCancellationDrainsAllStorages(t *testing.T) {
	cause := errors.New("GC stopped")
	ctx, cancel := context.WithCancelCause(t.Context())
	defer cancel(nil)
	gc := &GC{}
	started := make(chan struct{}, 2)
	finished := make(chan struct{}, 2)
	release := []chan struct{}{make(chan struct{}), make(chan struct{})}
	var once [2]sync.Once
	unblock := func(i int) { once[i].Do(func() { close(release[i]) }) }
	defer unblock(0)
	defer unblock(1)
	var reads, deletes atomic.Int32
	for i, kind := range []config.StorageType{config.Binary, config.GSYM} {
		s := &stubStorage{
			collect: func(ctx context.Context) ([]*storage.ObjectMeta, error) {
				reads.Add(1)
				started <- struct{}{}
				<-ctx.Done()
				<-release[i]
				finished <- struct{}{}
				return []*storage.ObjectMeta{{ID: "expired"}}, nil
			},
			delete: func(context.Context, []string) error { deletes.Add(1); return nil },
		}
		c, _ := testStorageGC(t, s, kind, 1)
		gc.collectors = append(gc.collectors, c)
	}
	done := make(chan error, 1)
	go func() { done <- gc.Run(ctx, time.Hour) }()
	awaitSignal(t, started)
	awaitSignal(t, started)
	cancel(cause)
	unblock(0)
	awaitSignal(t, finished)
	select {
	case <-done:
		t.Fatal("GC returned while a storage operation was still active")
	default:
	}
	unblock(1)
	require.ErrorIs(t, awaitRun(t, done), cause)
	require.EqualValues(t, 2, reads.Load())
	require.Zero(t, deletes.Load())
}

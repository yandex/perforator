//go:build linux || darwin
// +build linux darwin

package logrotate

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const testTimeout = 5 * time.Second

// Gates hold file operations at a known point, before they access the descriptor.
// In particular, closing a stubFile while Write is gated reproduces ErrClosed.
type gate struct {
	entered chan struct{}
	release chan struct{}
	enter   sync.Once
	leave   sync.Once
}

func newGate(t *testing.T) *gate {
	t.Helper()
	g := &gate{entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(g.unblock)
	return g
}

func (g *gate) block() {
	g.enter.Do(func() { close(g.entered) })
	<-g.release
}

func (g *gate) unblock() { g.leave.Do(func() { close(g.release) }) }

type stubFile struct {
	beforeWrite func()
	beforeSync  func()
	beforeClose func()
	writeErr    error
	writeN      int
	syncErr     error
	closeErr    error
	closed      atomic.Bool
	writes      atomic.Int32
	closes      atomic.Int32
}

func (f *stubFile) Write(p []byte) (int, error) {
	f.writes.Add(1)
	if f.beforeWrite != nil {
		f.beforeWrite()
	}
	if f.closed.Load() {
		return 0, os.ErrClosed
	}
	if f.writeErr != nil {
		return f.writeN, f.writeErr
	}
	return len(p), nil
}

func (f *stubFile) Sync() error {
	if f.beforeSync != nil {
		f.beforeSync()
	}
	if f.closed.Load() {
		return os.ErrClosed
	}
	return f.syncErr
}

func (f *stubFile) Close() error {
	f.closes.Add(1)
	if f.beforeClose != nil {
		f.beforeClose()
	}
	if f.closed.Swap(true) {
		return os.ErrClosed
	}
	return f.closeErr
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for operation")
		var zero T
		return zero
	}
}

func runAsync(f func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	return done
}

func requireBlocked(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("operation completed before the file operation was released: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
}

func newTestSink(t *testing.T, openFile func(string) (fileHandle, error)) *sink {
	t.Helper()
	s, err := newLogrotateSink(&url.URL{Path: filepath.Join(t.TempDir(), "test.log")}, openFile, syscall.SIGUSR2)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = s.Close() // A test may intentionally inject a close error.
		await(t, s.listenerDone)
	})
	return s
}

func writeRecord(s *sink) error {
	_, err := s.Write([]byte("record\n"))
	return err
}

func TestConcurrentRotationWaitsForFileOperations(t *testing.T) {
	for _, operation := range []string{"write", "sync"} {
		t.Run(operation, func(t *testing.T) {
			old, next := &stubFile{}, &stubFile{}
			opened := make(chan struct{})
			calls := 0
			s := newTestSink(t, func(string) (fileHandle, error) {
				calls++
				if calls == 1 {
					return old, nil
				}
				close(opened)
				return next, nil
			})
			g := newGate(t)
			var done <-chan error
			if operation == "write" {
				old.beforeWrite = g.block
				done = runAsync(func() error { return writeRecord(s) })
			} else {
				old.beforeSync = g.block
				done = runAsync(s.Sync)
			}
			await(t, g.entered)
			rotated := runAsync(s.reopen)
			await(t, opened)
			requireBlocked(t, rotated)
			require.Zero(t, old.closes.Load())
			g.unblock()
			require.NoError(t, await(t, done))
			require.NoError(t, await(t, rotated))
			require.EqualValues(t, 1, old.closes.Load())
			require.NoError(t, writeRecord(s))
			require.EqualValues(t, 1, next.writes.Load())
		})
	}
}

func TestConcurrentWriteDuringOpen(t *testing.T) {
	old, next := &stubFile{}, &stubFile{}
	var opening *gate
	calls := 0
	s := newTestSink(t, func(string) (fileHandle, error) {
		calls++
		if calls == 1 {
			return old, nil
		}
		opening.block()
		return next, nil
	})
	opening = newGate(t)
	rotated := runAsync(s.reopen)
	await(t, opening.entered)
	require.NoError(t, await(t, runAsync(func() error { return writeRecord(s) })))
	require.NoError(t, await(t, runAsync(s.Sync)))
	require.EqualValues(t, 1, old.writes.Load())
	opening.unblock()
	require.NoError(t, await(t, rotated))
}

func TestConcurrentWriteDuringOldClose(t *testing.T) {
	old, next := &stubFile{}, &stubFile{}
	calls := 0
	s := newTestSink(t, func(string) (fileHandle, error) {
		calls++
		if calls == 1 {
			return old, nil
		}
		return next, nil
	})
	closing := newGate(t)
	old.beforeClose = closing.block
	rotated := runAsync(s.reopen)
	await(t, closing.entered)
	require.NoError(t, await(t, runAsync(func() error { return writeRecord(s) })))
	require.NoError(t, await(t, runAsync(s.Sync)))
	require.EqualValues(t, 1, next.writes.Load())
	closing.unblock()
	require.NoError(t, await(t, rotated))
}

func TestConcurrentCloseWaitsForFileOperations(t *testing.T) {
	for _, operation := range []string{"write", "sync"} {
		t.Run(operation, func(t *testing.T) {
			file := &stubFile{}
			s := newTestSink(t, func(string) (fileHandle, error) { return file, nil })
			g := newGate(t)
			var done <-chan error
			if operation == "write" {
				file.beforeWrite = g.block
				done = runAsync(func() error { return writeRecord(s) })
			} else {
				file.beforeSync = g.block
				done = runAsync(s.Sync)
			}
			await(t, g.entered)
			closed := runAsync(s.Close)
			requireBlocked(t, closed)
			require.Zero(t, file.closes.Load())
			g.unblock()
			require.NoError(t, await(t, done))
			require.NoError(t, await(t, closed))
			require.EqualValues(t, 1, file.closes.Load())
		})
	}
}

func TestConcurrentCloseAndReopen(t *testing.T) {
	old, next := &stubFile{}, &stubFile{}
	var opening *gate
	var calls atomic.Int32
	s := newTestSink(t, func(string) (fileHandle, error) {
		if calls.Add(1) == 1 {
			return old, nil
		}
		opening.block()
		return next, nil
	})
	opening = newGate(t)
	rotated := runAsync(s.reopen)
	await(t, opening.entered)
	closed := runAsync(s.Close)
	requireBlocked(t, closed)
	opening.unblock()
	require.NoError(t, await(t, rotated))
	require.NoError(t, await(t, closed))
	require.EqualValues(t, 1, old.closes.Load())
	require.EqualValues(t, 1, next.closes.Load())
	require.ErrorIs(t, s.reopen(), os.ErrClosed)
	require.EqualValues(t, 2, calls.Load())
}

func TestConcurrentReopensAreSerialized(t *testing.T) {
	files := []*stubFile{{}, {}, {}}
	var calls atomic.Int32
	s := newTestSink(t, func(string) (fileHandle, error) {
		return files[calls.Add(1)-1], nil
	})
	closing := newGate(t)
	files[0].beforeClose = closing.block
	first := runAsync(s.reopen)
	await(t, closing.entered)
	second := runAsync(s.reopen)
	requireBlocked(t, second)
	require.EqualValues(t, 2, calls.Load())
	closing.unblock()
	require.NoError(t, await(t, first))
	require.NoError(t, await(t, second))
	require.NoError(t, s.Close())
	for _, f := range files {
		require.EqualValues(t, 1, f.closes.Load())
	}
}

func TestConcurrentCloseIsIdempotent(t *testing.T) {
	for _, closeErr := range []error{nil, errors.New("close failure")} {
		t.Run(fmt.Sprint(closeErr), func(t *testing.T) {
			file := &stubFile{closeErr: closeErr}
			s := newTestSink(t, func(string) (fileHandle, error) { return file, nil })
			closing := newGate(t)
			file.beforeClose = closing.block
			results := make([]<-chan error, 32)
			for i := range results {
				results[i] = runAsync(s.Close)
			}
			await(t, closing.entered)
			requireBlocked(t, results[0])
			closing.unblock()
			for _, result := range results {
				require.ErrorIs(t, await(t, result), closeErr)
			}
			require.ErrorIs(t, s.Close(), closeErr)
			require.EqualValues(t, 1, file.closes.Load())
			for _, p := range [][]byte{nil, []byte("after close")} {
				n, err := s.Write(p)
				require.Zero(t, n)
				require.Equal(t, &os.PathError{Op: "write", Path: s.path, Err: os.ErrClosed}, err)
			}
			require.Equal(t, &os.PathError{Op: "sync", Path: s.path, Err: os.ErrClosed}, s.Sync())
			require.ErrorIs(t, s.reopen(), os.ErrClosed)
		})
	}
}

func TestReopenKeepsFileOnOpenError(t *testing.T) {
	old, next := &stubFile{}, &stubFile{}
	openErr := errors.New("open failure")
	calls := 0
	s := newTestSink(t, func(string) (fileHandle, error) {
		calls++
		switch calls {
		case 1:
			return old, nil
		case 2:
			return nil, openErr
		default:
			return next, nil
		}
	})
	require.ErrorIs(t, s.reopen(), openErr)
	require.NoError(t, writeRecord(s))
	require.NoError(t, s.Sync())
	require.Zero(t, old.closes.Load())
	require.EqualValues(t, 1, old.writes.Load())
	require.NoError(t, s.reopen())
	require.NoError(t, writeRecord(s))
	require.EqualValues(t, 1, old.closes.Load())
	require.EqualValues(t, 1, next.writes.Load())
}

func TestReopenKeepsNewFileOnCloseError(t *testing.T) {
	closeErr := errors.New("close failure")
	old, next, last := &stubFile{closeErr: closeErr}, &stubFile{}, &stubFile{}
	files := []fileHandle{old, next, last}
	s := newTestSink(t, func(string) (fileHandle, error) {
		f := files[0]
		files = files[1:]
		return f, nil
	})
	require.ErrorIs(t, s.reopen(), closeErr)
	require.NoError(t, writeRecord(s))
	require.NoError(t, s.Sync())
	require.EqualValues(t, 1, next.writes.Load())
	require.NoError(t, s.reopen())
	require.NoError(t, s.Close())
	for _, f := range []*stubFile{old, next, last} {
		require.EqualValues(t, 1, f.closes.Load())
	}
}

func TestFileErrorsAreNotRetried(t *testing.T) {
	writeErr, syncErr := errors.New("partial write"), errors.New("sync failure")
	file := &stubFile{writeErr: writeErr, writeN: 2, syncErr: syncErr}
	s := newTestSink(t, func(string) (fileHandle, error) { return file, nil })
	n, err := s.Write([]byte("record"))
	require.Equal(t, 2, n)
	require.ErrorIs(t, err, writeErr)
	require.EqualValues(t, 1, file.writes.Load())
	require.ErrorIs(t, s.Sync(), syncErr)
}

func TestConstructorValidation(t *testing.T) {
	for name, u := range map[string]*url.URL{
		"user":     {Path: "test.log", User: url.User("user")},
		"fragment": {Path: "test.log", Fragment: "fragment"},
		"port":     {Path: "test.log", Host: "localhost:1234"},
		"host":     {Path: "test.log", Host: "remote"},
	} {
		t.Run(name, func(t *testing.T) {
			s, err := newLogrotateSink(u, func(string) (fileHandle, error) {
				t.Fatal("invalid URL must not open a file")
				return nil, nil
			}, syscall.SIGUSR2)
			require.Error(t, err)
			require.Nil(t, s)
			public, err := NewLogrotateSink(u, syscall.SIGUSR2)
			require.Error(t, err)
			require.Nil(t, public)
		})
	}
	public, err := NewLogrotateSink(&url.URL{Path: t.TempDir()}, syscall.SIGUSR2)
	require.Error(t, err)
	require.Nil(t, public)
}

func TestAppendAndRotation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.log")
	require.NoError(t, os.WriteFile(path, []byte("existing\n"), 0600))
	// Query parameters have historically been ignored by this sink.
	s, err := newLogrotateSink(&url.URL{Path: path, Host: "localhost", RawQuery: "ignored=1"}, openLogFile, syscall.SIGUSR2)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close(); await(t, s.listenerDone) })
	_, err = s.Write([]byte("before\n"))
	require.NoError(t, err)
	require.NoError(t, os.Rename(path, path+".1"))
	require.NoError(t, s.reopen())
	_, err = s.Write([]byte("after\n"))
	require.NoError(t, err)
	require.NoError(t, s.Sync())
	require.NoError(t, s.Close())
	data, err := os.ReadFile(path + ".1")
	require.NoError(t, err)
	require.Equal(t, "existing\nbefore\n", string(data))
	data, err = os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "after\n", string(data))
	info, err := os.Stat(path + ".1")
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0600), info.Mode().Perm())
	info, err = os.Stat(path)
	require.NoError(t, err)
	require.Zero(t, info.Mode().Perm() & ^os.FileMode(0644))
}

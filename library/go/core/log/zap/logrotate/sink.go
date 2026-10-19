//go:build darwin || freebsd || linux
// +build darwin freebsd linux

package logrotate

import (
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"sync"

	"go.uber.org/zap"

	"github.com/yandex/perforator/library/go/core/xerrors"
)

const defaultSchemeName = "logrotate"

// Register logrotate sink in zap sink registry.
// This sink internally is like file sink, but listens to provided logrotate signal
// and reopens file when that signal is delivered
// This can be called only once. Any future calls will result in an error
func RegisterLogrotateSink(sig ...os.Signal) error {
	return RegisterNamedLogrotateSink(defaultSchemeName, sig...)
}

// Same as RegisterLogrotateSink, but use provided schemeName instead of default `logrotate`
// Can be useful in special cases for registering different types of sinks for different signal
func RegisterNamedLogrotateSink(schemeName string, sig ...os.Signal) error {
	factory := func(url *url.URL) (sink zap.Sink, e error) {
		return NewLogrotateSink(url, sig...)
	}
	return zap.RegisterSink(schemeName, factory)
}

type fileHandle interface {
	Write([]byte) (int, error)
	Sync() error
	Close() error
}

// sink itself, use RegisterLogrotateSink to register it in zap machinery.
// Lock order is lifecycleMu -> fileMu. Write and Sync only acquire fileMu.
type sink struct {
	path     string
	openFile func(string) (fileHandle, error)

	// Serialize reopens and Close without blocking writes during OpenFile or
	// the closing of a file that has already been replaced.
	lifecycleMu sync.Mutex
	closeErr    error // guarded by lifecycleMu

	// A reader holds fileMu for the entire file operation, not just the lookup.
	fileMu sync.RWMutex
	file   fileHandle
	closed bool

	notifier     chan os.Signal
	listenerDone chan struct{}
}

// Factory for logrotate sink, which accepts os.Signals to listen to for reloading
// Generally if you don't build your own core it is used by zap machinery.
// See RegisterLogrotateSink.
// The returned sink supports concurrent Write, Sync, rotation and Close.
// Close is idempotent and returns the result of the first close operation.
func NewLogrotateSink(u *url.URL, sig ...os.Signal) (zap.Sink, error) {
	sink, err := newLogrotateSink(u, openLogFile, sig...)
	if err != nil {
		return nil, err
	}
	return sink, nil
}

func newLogrotateSink(u *url.URL, openFile func(string) (fileHandle, error), sig ...os.Signal) (*sink, error) {
	if u.User != nil {
		return nil, fmt.Errorf("user and password not allowed with logrotate file URLs: got %v", u)
	}
	if u.Fragment != "" {
		return nil, fmt.Errorf("fragments not allowed with logrotate file URLs: got %v", u)
	}
	// Error messages are better if we check hostname and port separately.
	if u.Port() != "" {
		return nil, fmt.Errorf("ports not allowed with logrotate file URLs: got %v", u)
	}
	if hn := u.Hostname(); hn != "" && hn != "localhost" {
		return nil, fmt.Errorf("logrotate file URLs must leave host empty or use localhost: got %v", u)
	}

	sink := &sink{
		path:         u.Path,
		openFile:     openFile,
		notifier:     make(chan os.Signal, 1),
		listenerDone: make(chan struct{}),
	}
	if err := sink.reopen(); err != nil {
		return nil, err
	}
	// Do not leave a signal subscription behind if validation or opening fails.
	signal.Notify(sink.notifier, sig...)
	go sink.listenToSignal()
	return sink, nil
}

func openLogFile(path string) (fileHandle, error) {
	return os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
}

func (m *sink) listenToSignal() {
	defer close(m.listenerDone)
	for range m.notifier {
		if err := m.reopen(); err != nil {
			if err == os.ErrClosed {
				return
			}
			// Last chance to signalize about an error
			_, _ = fmt.Fprintf(os.Stderr, "%s", err)
		}
	}
}

func (m *sink) reopen() error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	m.fileMu.RLock()
	closed := m.closed
	m.fileMu.RUnlock()
	if closed {
		return os.ErrClosed
	}

	file, err := m.openFile(m.path)
	if err != nil {
		return xerrors.Errorf("failed to open log file on %s: %w", m.path, err)
	}
	// Wait for all operations using the old file before making it unreachable.
	// lifecycleMu prevents Close from racing with opening or installing the file.
	m.fileMu.Lock()
	old := m.file
	m.file = file
	m.fileMu.Unlock()
	if old != nil {
		if err := old.Close(); err != nil {
			return xerrors.Errorf("failed to close old file: %w", err)
		}
	}
	return nil
}

func (m *sink) Close() error {
	m.lifecycleMu.Lock()
	defer m.lifecycleMu.Unlock()

	m.fileMu.Lock()
	if m.closed {
		m.fileMu.Unlock()
		return m.closeErr
	}
	m.closed = true
	file := m.file
	m.file = nil
	m.fileMu.Unlock()

	signal.Stop(m.notifier)
	close(m.notifier)
	m.closeErr = file.Close()
	// Do not wait for listenerDone: reporting an earlier reopen error to stderr
	// may be blocked. Any queued reopen will observe closed and cannot open a file.
	return m.closeErr
}

func (m *sink) Write(p []byte) (n int, err error) {
	m.fileMu.RLock()
	defer m.fileMu.RUnlock()
	if m.closed {
		return 0, &os.PathError{Op: "write", Path: m.path, Err: os.ErrClosed}
	}
	return m.file.Write(p)
}

func (m *sink) Sync() error {
	m.fileMu.RLock()
	defer m.fileMu.RUnlock()
	if m.closed {
		return &os.PathError{Op: "sync", Path: m.path, Err: os.ErrClosed}
	}
	return m.file.Sync()
}

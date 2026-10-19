//go:build linux || darwin
// +build linux darwin

package logrotate

import (
	"context"
	"errors"
	"io"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const signalHelperEnv = "LOGROTATE_SIGNAL_TEST_HELPER"

func TestSignalProcesses(t *testing.T) {
	for _, mode := range []string{"rotation", "registration", "invalid-url", "open-error", "closed", "blocked-stderr"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*testTimeout)
			defer cancel()
			executable, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestSignalHelper$", "-test.v")
			// The parent also cleans up temporary files of helpers killed by SIGHUP.
			cmd.Env = append(os.Environ(), signalHelperEnv+"="+mode, "TMPDIR="+t.TempDir())
			output, err := cmd.CombinedOutput()
			require.NoError(t, ctx.Err(), "%s", output)
			switch mode {
			case "invalid-url", "open-error", "closed":
				// A leaked subscription would swallow SIGHUP and let the helper exit normally.
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr, "%s", output)
				status, ok := exitErr.Sys().(syscall.WaitStatus)
				require.True(t, ok)
				require.True(t, status.Signaled(), "%s", output)
				require.Equal(t, syscall.SIGHUP, status.Signal(), "%s", output)
			default:
				require.NoError(t, err, "%s", output)
			}
		})
	}
}

func TestSignalHelper(t *testing.T) {
	mode := os.Getenv(signalHelperEnv)
	if mode == "" {
		t.Skip("runs in a subprocess")
	}
	switch mode {
	case "rotation":
		testSignalRotation(t)
	case "registration":
		testSignalRegistration(t)
	case "blocked-stderr":
		testBlockedStderr(t)
	case "invalid-url", "open-error", "closed":
		// Only a subprocess may reset a process-wide signal disposition.
		signal.Reset(syscall.SIGHUP)
		path := filepath.Join(t.TempDir(), "test.log")
		u := &url.URL{Path: path}
		if mode == "invalid-url" {
			u.User = url.User("invalid")
		} else if mode == "open-error" {
			u.Path = filepath.Dir(path) // Opening a directory for writing fails on all supported OSes.
		}
		s, err := NewLogrotateSink(u, syscall.SIGHUP)
		if mode == "closed" {
			require.NoError(t, err)
			require.NoError(t, s.Close())
			await(t, s.(*sink).listenerDone)
		} else {
			require.Error(t, err)
			require.Nil(t, s)
		}
		require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGHUP))
		// Give the runtime time to terminate; a swallowed signal must fail the parent test.
		<-time.After(200 * time.Millisecond)
	default:
		t.Fatalf("unknown subprocess mode %q", mode)
	}
}

func testSignalRotation(t *testing.T) {
	newSink := func(name string, signals ...os.Signal) *sink {
		s, err := newLogrotateSink(&url.URL{Path: filepath.Join(t.TempDir(), name)}, openLogFile, signals...)
		require.NoError(t, err)
		t.Cleanup(func() { _ = s.Close(); await(t, s.listenerDone) })
		return s
	}
	first := newSink("first.log", syscall.SIGHUP, syscall.SIGUSR1)
	second := newSink("second.log", syscall.SIGHUP)
	for _, s := range []*sink{first, second} {
		require.NoError(t, writeRecord(s))
		require.NoError(t, os.Rename(s.path, s.path+".1"))
	}
	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGHUP))
	for _, s := range []*sink{first, second} {
		waitForRotatedRecord(t, s, "after SIGHUP\n")
	}
	// A second configured signal reopens the first sink without touching the second.
	require.NoError(t, os.Rename(first.path, first.path+".2"))
	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGUSR1))
	waitForRotatedRecord(t, first, "after SIGUSR1\n")
	require.NoError(t, first.Close())
	await(t, first.listenerDone)
	// Closing one subscription must not reset the other's signal handling.
	require.NoError(t, os.Rename(second.path, second.path+".2"))
	require.NoError(t, syscall.Kill(syscall.Getpid(), syscall.SIGHUP))
	waitForRotatedRecord(t, second, "after other sink closed\n")
}

func waitForRotatedRecord(t *testing.T, s *sink, marker string) {
	t.Helper()
	require.Eventually(t, func() bool {
		if _, err := s.Write([]byte(marker)); err != nil {
			return false
		}
		data, err := os.ReadFile(s.path)
		return err == nil && strings.Contains(string(data), marker)
	}, testTimeout, 10*time.Millisecond)
}

func testSignalRegistration(t *testing.T) {
	require.NoError(t, RegisterLogrotateSink(syscall.SIGUSR1))
	require.Error(t, RegisterLogrotateSink(syscall.SIGUSR1))
	require.NoError(t, RegisterNamedLogrotateSink("named-logrotate", syscall.SIGHUP))
	require.Error(t, RegisterNamedLogrotateSink("named-logrotate", syscall.SIGHUP))
	require.Error(t, RegisterNamedLogrotateSink("", syscall.SIGHUP))
	for _, scheme := range []string{"logrotate", "named-logrotate"} {
		path := filepath.Join(t.TempDir(), "test.log")
		writer, closeAll, err := zap.Open(scheme + "://" + path)
		require.NoError(t, err)
		t.Cleanup(closeAll)
		logger := zap.New(zapcore.NewCore(zapcore.NewConsoleEncoder(zapcore.EncoderConfig{MessageKey: "msg"}), writer, zapcore.InfoLevel))
		logger.Info("registered sink")
		require.NoError(t, logger.Sync())
		closeAll()
		closeAll() // zap.Open's cleanup may be invoked more than once.
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "registered sink\n", string(data))
	}
}

func testBlockedStderr(t *testing.T) {
	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stderr
	os.Stderr = writer
	defer func() {
		os.Stderr = original
		_ = reader.Close()
		_ = writer.Close()
	}()
	file := &stubFile{}
	var opens atomic.Int32
	s, err := newLogrotateSink(&url.URL{Path: "test.log"}, func(string) (fileHandle, error) {
		if opens.Add(1) == 1 {
			return file, nil
		}
		return nil, errors.New(strings.Repeat("failure ", 1024*1024))
	}, syscall.SIGHUP)
	require.NoError(t, err)
	defer func() { _ = s.Close() }()
	s.notifier <- syscall.SIGHUP
	// Reading the beginning of an error larger than pipe capacity establishes
	// that the listener is reporting it, while the rest of the write is blocked.
	one := make([]byte, 1)
	_, err = io.ReadFull(reader, one)
	require.NoError(t, err)
	// Leave a queued notification behind the blocked diagnostic.
	s.notifier <- syscall.SIGHUP
	require.NoError(t, await(t, runAsync(s.Close)))
	require.EqualValues(t, 1, file.closes.Load())
	drained := runAsync(func() error { _, err := io.Copy(io.Discard, reader); return err })
	await(t, s.listenerDone)
	require.NoError(t, writer.Close())
	require.NoError(t, await(t, drained))
	require.EqualValues(t, 2, opens.Load(), "queued signal reopened a closed sink")
}

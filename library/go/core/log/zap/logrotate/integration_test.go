//go:build linux || darwin
// +build linux darwin

package logrotate

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

func TestConcurrentRotationPreservesRecords(t *testing.T) {
	for _, mode := range []string{"direct", "buffered-small", "buffered-1MiB", "zap"} {
		t.Run(mode, func(t *testing.T) {
			s := newTestSink(t, openLogFile)
			var writer io.Writer = s
			var syncer zapcore.WriteSyncer = s
			stop := s.Sync
			var errorOutput bytes.Buffer
			if mode != "direct" {
				size := 512
				if mode == "buffered-1MiB" {
					size = 1024 * 1024
				}
				buffer := &zapcore.BufferedWriteSyncer{WS: s, Size: size, FlushInterval: time.Second}
				writer, syncer, stop = buffer, buffer, buffer.Stop
				t.Cleanup(func() { _ = buffer.Stop() })
				if mode == "zap" {
					logger := zap.New(zapcore.NewCore(
						zapcore.NewConsoleEncoder(zapcore.EncoderConfig{MessageKey: "msg"}),
						buffer, zapcore.InfoLevel,
					), zap.ErrorOutput(zapcore.Lock(zapcore.AddSync(&errorOutput))))
					writer = writerFunc(func(p []byte) (int, error) {
						logger.Info(strings.TrimSuffix(string(p), "\n"))
						return len(p), nil
					})
				}
			}

			const writers, records, rotations = 4, 2000, 50
			payload := strings.Repeat("x", 240)
			expected := make(map[string]bool, writers*records)
			for w := 0; w < writers; w++ {
				for i := 0; i < records; i++ {
					expected[fmt.Sprintf("%d:%04d:%s", w, i, payload)] = true
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			start := make(chan struct{})
			progress := make(chan struct{}, 1)
			results := make(chan error, writers+2)
			var wg sync.WaitGroup
			launch := func(f func() error) {
				wg.Add(1)
				go func() { defer wg.Done(); <-start; results <- f() }()
			}
			for w := 0; w < writers; w++ {
				launch(func() error {
					for i := 0; i < records; i++ {
						p := []byte(fmt.Sprintf("%d:%04d:%s\n", w, i, payload))
						n, err := writer.Write(p)
						if err != nil {
							return err
						}
						if n != len(p) {
							return io.ErrShortWrite
						}
						select {
						case progress <- struct{}{}:
						default:
						}
					}
					return nil
				})
			}
			launch(func() error {
				for i := 0; i < rotations; i++ {
					// The first rotation starts only after writing has begun.
					if i == 0 {
						select {
						case <-progress:
						case <-ctx.Done():
							return ctx.Err()
						}
					}
					if err := os.Rename(s.path, fmt.Sprintf("%s.%d", s.path, i)); err != nil {
						return err
					}
					if err := s.reopen(); err != nil {
						return err
					}
				}
				return nil
			})
			launch(func() error {
				for i := 0; i < 50; i++ {
					if err := syncer.Sync(); err != nil {
						return err
					}
				}
				return nil
			})
			close(start)
			var operationErrors []error
			for i := 0; i < writers+2; i++ {
				operationErrors = append(operationErrors, await(t, results))
			}
			wg.Wait()
			for _, err := range operationErrors {
				require.NoError(t, err)
			}
			// A poisoned bufio.Writer would continue to fail here after all rotations.
			n, err := writer.Write([]byte("after rotations\n"))
			require.NoError(t, err)
			require.Equal(t, len("after rotations\n"), n)
			expected["after rotations"] = true
			require.NoError(t, stop())
			require.NoError(t, s.Close())
			require.Empty(t, errorOutput.String())

			for i := 0; i <= rotations; i++ {
				path := s.path
				if i < rotations {
					path = fmt.Sprintf("%s.%d", s.path, i)
				}
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				if len(data) > 0 {
					require.Equal(t, byte('\n'), data[len(data)-1], "partial record in %s", path)
				}
				scanner := bufio.NewScanner(bytes.NewReader(data))
				for scanner.Scan() {
					line := scanner.Text()
					require.True(t, expected[line], "duplicate or unexpected record: %q", line)
					delete(expected, line)
				}
				require.NoError(t, scanner.Err())
			}
			require.Empty(t, expected, "records were lost")
		})
	}
}

func TestBufferedPeriodicFlushAfterRotation(t *testing.T) {
	s := newTestSink(t, openLogFile)
	buffer := &zapcore.BufferedWriteSyncer{WS: s, Size: 1024 * 1024, FlushInterval: time.Second}
	t.Cleanup(func() { _ = buffer.Stop() })
	_, err := buffer.Write([]byte("before\n"))
	require.NoError(t, err)
	require.NoError(t, buffer.Sync())
	require.NoError(t, os.Rename(s.path, s.path+".1"))
	require.NoError(t, s.reopen())
	_, err = buffer.Write([]byte("periodic flush\n"))
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		data, err := os.ReadFile(s.path)
		return err == nil && string(data) == "periodic flush\n"
	}, testTimeout, 10*time.Millisecond)
	require.NoError(t, buffer.Stop())
}

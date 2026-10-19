//go:build linux || darwin
// +build linux darwin

package logrotate

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
)

func BenchmarkLogrotateWrite(b *testing.B) {
	benchmarkLogrotateMatrix(b, "")
}

// Isolate synchronization overhead from disk writeback and fsync latency.
func BenchmarkLogrotateWriteToDevNull(b *testing.B) {
	benchmarkLogrotateMatrix(b, os.DevNull)
}

func benchmarkLogrotateMatrix(b *testing.B, destination string) {
	for _, buffered := range []bool{false, true} {
		for _, size := range []int{512, 4096} {
			for _, workers := range []int{1, 8, 32} {
				b.Run(fmt.Sprintf("buffered=%t/bytes=%d/writers=%d", buffered, size, workers), func(b *testing.B) {
					benchmarkLogrotateWrite(b, buffered, size, workers, false, destination)
				})
			}
		}
	}
}

func BenchmarkLogrotateWriteWithRotation(b *testing.B) {
	benchmarkLogrotateWrite(b, true, 4096, 8, true, "")
}

func benchmarkLogrotateWrite(b *testing.B, buffered bool, size, workers int, rotate bool, path string) {
	if path == "" {
		path = filepath.Join(b.TempDir(), "test.log")
	}
	s, err := NewLogrotateSink(&url.URL{Path: path}, syscall.SIGUSR2)
	if err != nil {
		b.Fatal(err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			b.Error(err)
		}
	}()
	var writer zapcore.WriteSyncer = s
	if path == os.DevNull {
		// fsync on /dev/null is unsupported; this benchmark measures Write only.
		writer = zapcore.AddSync(struct{ io.Writer }{s})
	}
	if buffered {
		buffer := &zapcore.BufferedWriteSyncer{WS: writer, Size: 1024 * 1024, FlushInterval: time.Second}
		writer = buffer
		defer func() {
			if err := buffer.Stop(); err != nil {
				b.Error(err)
			}
		}()
	}
	var stopping chan struct{}
	var stopped chan error
	var rotationCount atomic.Int64
	if rotate {
		stopping, stopped = make(chan struct{}), make(chan error, 1)
		go func() {
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := os.Rename(path, path+".rotated"); err != nil {
						stopped <- err
						return
					}
					if err := s.(*sink).reopen(); err != nil {
						stopped <- err
						return
					}
					rotationCount.Add(1)
				case <-stopping:
					stopped <- nil
					return
				}
			}
		}()
		defer func() {
			close(stopping)
			if err := <-stopped; err != nil {
				b.Error(err)
			}
		}()
	}
	previous := runtime.GOMAXPROCS(workers)
	defer runtime.GOMAXPROCS(previous)
	p := bytes.Repeat([]byte("x"), size)
	if _, err := writer.Write(p); err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(size))
	b.ReportAllocs()
	b.ResetTimer()
	if workers == 1 {
		for i := 0; i < b.N; i++ {
			if _, err := writer.Write(p); err != nil {
				b.Fatal(err)
			}
		}
	} else {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				if _, err := writer.Write(p); err != nil {
					b.Error(err)
					return
				}
			}
		})
	}
	b.StopTimer()
	if rotate {
		b.ReportMetric(float64(rotationCount.Load()), "rotations")
	}
}

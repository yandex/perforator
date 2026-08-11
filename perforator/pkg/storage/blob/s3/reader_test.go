package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func blobFetcher(t *testing.T, blob []byte, inflight *atomic.Int64, maxInflight *atomic.Int64) rangeFetcher {
	return func(ctx context.Context, start, end int64) ([]byte, error) {
		if inflight != nil {
			cur := inflight.Add(1)
			defer inflight.Add(-1)
			for {
				prev := maxInflight.Load()
				if cur <= prev || maxInflight.CompareAndSwap(prev, cur) {
					break
				}
			}
		}
		require.Less(t, start, int64(len(blob)))
		require.Less(t, end, int64(len(blob)))
		return blob[start : end+1], nil
	}
}

func TestRangedReader_Roundtrip(t *testing.T) {
	for _, tc := range []struct {
		name        string
		size        int64
		partSize    int64
		concurrency int
	}{
		{"single byte parts", 100, 1, 4},
		{"uneven tail", 1000, 64, 3},
		{"exact multiple", 1024, 128, 2},
		{"one extra byte", 129, 128, 2},
		{"large parts few bytes", 10, 1024, 8},
	} {
		t.Run(tc.name, func(t *testing.T) {
			blob := make([]byte, tc.size)
			for i := range blob {
				blob[i] = byte(i % 251)
			}

			first := blob[:min(tc.partSize, tc.size)]
			r := newRangedReader(context.Background(), blobFetcher(t, blob, nil, nil), first, tc.size, tc.concurrency, tc.partSize)
			defer r.Close()

			data, err := io.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, blob, data)
		})
	}
}

func TestRangedReader_BoundedConcurrency(t *testing.T) {
	blob := make([]byte, 1<<16)
	const concurrency = 4

	var inflight, maxInflight atomic.Int64
	r := newRangedReader(context.Background(), blobFetcher(t, blob, &inflight, &maxInflight), blob[:128], int64(len(blob)), concurrency, 128)
	defer r.Close()

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, blob, data)
	require.LessOrEqual(t, maxInflight.Load(), int64(concurrency))
}

func TestRangedReader_FetchErrorPropagates(t *testing.T) {
	boom := errors.New("fetch failed")
	fetch := func(ctx context.Context, start, end int64) ([]byte, error) {
		if start >= 512 {
			return nil, boom
		}
		return make([]byte, end-start+1), nil
	}

	r := newRangedReader(context.Background(), fetch, make([]byte, 256), 2048, 2, 256)
	defer r.Close()

	_, err := io.ReadAll(r)
	require.ErrorIs(t, err, boom)
}

// TestRangedReader_EarlyClose abandons the stream after the first bytes; all
// fetcher goroutines must exit via context cancellation.
func TestRangedReader_EarlyClose(t *testing.T) {
	blob := make([]byte, 1<<20)
	release := make(chan struct{})
	fetch := func(ctx context.Context, start, end int64) ([]byte, error) {
		select {
		case <-release:
			return blob[start : end+1], nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	r := newRangedReader(context.Background(), fetch, blob[:4096], int64(len(blob)), 4, 4096)

	head := make([]byte, 16)
	_, err := io.ReadFull(r, head)
	require.NoError(t, err)
	require.NoError(t, r.Close())
	close(release)

	_, err = r.Read(head)
	require.Error(t, err)
}

// TestRangedReader_CancelIsNotEOF pins the truncation bug: canceling the
// parent context mid-stream must surface an error, never a clean short EOF.
func TestRangedReader_CancelIsNotEOF(t *testing.T) {
	blob := make([]byte, 1<<20)
	ctx, cancel := context.WithCancel(context.Background())
	fetch := func(ctx context.Context, start, end int64) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	r := newRangedReader(ctx, fetch, blob[:4096], int64(len(blob)), 4, 4096)
	defer r.Close()

	head := make([]byte, 4096)
	_, err := io.ReadFull(r, head)
	require.NoError(t, err)

	cancel()
	data, err := io.ReadAll(r)
	require.ErrorIs(t, err, context.Canceled)
	require.Less(t, len(data)+len(head), len(blob)) // truncated — and reported as an error
}

// TestRangedReader_CloseDuringRead pins that a concurrent Close wakes a
// blocked Read with an error instead of racing on reader state.
func TestRangedReader_CloseDuringRead(t *testing.T) {
	blob := make([]byte, 1<<20)
	fetch := func(ctx context.Context, start, end int64) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	r := newRangedReader(context.Background(), fetch, blob[:4096], int64(len(blob)), 2, 4096)

	head := make([]byte, 4096)
	_, err := io.ReadFull(r, head)
	require.NoError(t, err)

	done := make(chan error, 1)
	go func() {
		_, err := io.ReadAll(r)
		done <- err
	}()
	require.NoError(t, r.Close())
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestParseContentRange(t *testing.T) {
	start, end, total, ok := parseContentRange("bytes 0-123/456")
	require.True(t, ok)
	require.Equal(t, int64(0), start)
	require.Equal(t, int64(123), end)
	require.Equal(t, int64(456), total)

	for _, bad := range []string{
		"bytes 0-123/*",
		"garbage",
		"bytes 0-10/-1",
		"items 0-123/456",
		"bytes 123/456",
		"bytes 10-5/456",
		"bytes 0-500/456",
		"bytes -3-5/456",
	} {
		_, _, _, ok := parseContentRange(bad)
		require.False(t, ok, "expected %q to be rejected", bad)
	}
}

func TestRangedReaderPartitioning(t *testing.T) {
	// The dispatcher must cover [len(first), total) exactly once.
	var mu sync.Mutex
	got := map[int64]int64{}
	fetch := func(ctx context.Context, start, end int64) ([]byte, error) {
		mu.Lock()
		got[start] = end
		mu.Unlock()
		return bytes.Repeat([]byte{1}, int(end-start+1)), nil
	}

	r := newRangedReader(context.Background(), fetch, make([]byte, 100), 250, 1, 100)
	defer r.Close()

	data, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Len(t, data, 250)
	require.Equal(t, map[int64]int64{100: 199, 200: 249}, got)
}

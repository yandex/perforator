package ringstream

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	mrand "math/rand"
	"runtime"
	"sync"
	"testing"
	"time"
)

// waitForBlockedWriters is a test helper that reliably waits until the buffer
// has exactly the specified number of blocked writer goroutines, avoiding
// flaky time.Sleep calls.
func waitForBlockedWriters(b *Adapter, count int) {
	timeout := time.After(time.Second)
	for {
		b.mu.Lock()
		bw := b.blockedWriters
		b.mu.Unlock()
		if bw == count {
			return
		}
		select {
		case <-timeout:
			panic("timeout waiting for blocked writers")
		default:
			runtime.Gosched()
		}
	}
}

func TestAdapter_SequentialWriteRead(t *testing.T) {
	b := NewAdapter(16)
	data := []byte("hello world!")

	n, err := b.WriteAt(data, 0)
	if err != nil || n != len(data) {
		t.Fatalf("WriteAt: n=%d err=%v", n, err)
	}
	b.CloseWithError(nil)

	got, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("got %q, want %q", got, data)
	}
}

func TestAdapter_OutOfOrderWrites(t *testing.T) {
	b := NewAdapter(32)

	if _, err := b.WriteAt([]byte("WORLD"), 6); err != nil {
		t.Fatalf("WriteAt(6): %v", err)
	}

	// Reader must not see anything yet — there's a hole at [0,6).
	done := make(chan struct{})
	var got []byte
	go func() {
		got, _ = io.ReadAll(b)
		close(done)
	}()

	// Since we cannot observe reader state deterministically, we just write
	// the missing piece and verify the reader unblocks and reads everything.
	if _, err := b.WriteAt([]byte("HELLO "), 0); err != nil {
		t.Fatalf("WriteAt(0): %v", err)
	}
	b.CloseWithError(nil)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("reader never finished")
	}

	if string(got) != "HELLO WORLD" {
		t.Fatalf("got %q", got)
	}
}

func TestAdapter_BackPressureBlocksWriter(t *testing.T) {
	b := NewAdapter(8)
	if _, err := b.WriteAt([]byte("12345678"), 0); err != nil {
		t.Fatalf("first write: %v", err)
	}

	writerDone := make(chan error, 1)
	go func() {
		_, err := b.WriteAt([]byte("ABCD"), 8)
		writerDone <- err
	}()

	// Wait deterministically for the writer to block on capacity.
	waitForBlockedWriters(b, 1)

	// Free up 4 bytes.
	out := make([]byte, 4)
	n, err := io.ReadFull(b, out)
	if err != nil || n != 4 {
		t.Fatalf("ReadFull: n=%d err=%v", n, err)
	}
	if string(out) != "1234" {
		t.Fatalf("got %q", out)
	}

	select {
	case err := <-writerDone:
		if err != nil {
			t.Fatalf("writer err: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatalf("writer never unblocked")
	}

	b.CloseWithError(nil)
	rest, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(rest) != "5678ABCD" {
		t.Fatalf("got %q", rest)
	}
}

func TestAdapter_IdempotentRetryAfterPartialRead(t *testing.T) {
	b := NewAdapter(32)

	chunk := []byte("CHUNKABCDE") // 10 bytes
	if _, err := b.WriteAt(chunk, 0); err != nil {
		t.Fatalf("first WriteAt: %v", err)
	}

	// Reader consumes 3 bytes.
	head := make([]byte, 3)
	if _, err := io.ReadFull(b, head); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}
	if string(head) != "CHU" {
		t.Fatalf("head=%q", head)
	}

	// Simulate s3manager retry: same offset, same bytes.
	n, err := b.WriteAt(chunk, 0)
	if err != nil {
		t.Fatalf("retry WriteAt: %v", err)
	}
	if n != len(chunk) {
		t.Fatalf("retry returned n=%d, want %d", n, len(chunk))
	}

	b.CloseWithError(nil)
	tail, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(tail) != "NKABCDE" {
		t.Fatalf("tail=%q", tail)
	}
}

func TestAdapter_CloseWithErrorPropagates(t *testing.T) {
	b := NewAdapter(8)
	myErr := errors.New("download failed")
	b.CloseWithError(myErr)

	if _, err := b.Read(make([]byte, 4)); !errors.Is(err, myErr) {
		t.Fatalf("Read returned %v, want %v", err, myErr)
	}
	if _, err := b.WriteAt([]byte("abcd"), 0); !errors.Is(err, myErr) {
		t.Fatalf("WriteAt returned %v, want %v", err, myErr)
	}
}

func TestAdapter_CloseUnblocksWriter(t *testing.T) {
	b := NewAdapter(8)
	if _, err := b.WriteAt([]byte("12345678"), 0); err != nil {
		t.Fatalf("fill: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := b.WriteAt([]byte("X"), 8)
		done <- err
	}()

	waitForBlockedWriters(b, 1)

	myErr := errors.New("aborted")
	b.CloseWithError(myErr)

	select {
	case err := <-done:
		if !errors.Is(err, myErr) {
			t.Fatalf("writer err=%v, want %v", err, myErr)
		}
	case <-time.After(time.Second):
		t.Fatalf("writer never unblocked")
	}
}

func TestAdapter_EmptyClose(t *testing.T) {
	b := NewAdapter(8)
	b.CloseWithError(nil)
	n, err := b.Read(make([]byte, 4))
	if n != 0 || err != io.EOF {
		t.Fatalf("got n=%d err=%v, want 0/EOF", n, err)
	}
}

// TestAdapter_ConcurrentParallelChunks mimics s3manager.Downloader behaviour:
// 20 goroutines write chunks of 64 KiB in arbitrary order with random jitter,
// reader streams sequentially. Verify byte-for-byte equality with original.
func TestAdapter_ConcurrentParallelChunks(t *testing.T) {
	const (
		chunkSize  = 64 * 1024
		numChunks  = 200
		concurrent = 20
		capacity   = 2 * concurrent * chunkSize
	)

	original := make([]byte, chunkSize*numChunks)
	if _, err := rand.Read(original); err != nil {
		t.Fatalf("rand: %v", err)
	}

	b := NewAdapter(capacity)

	// Producer dispatcher: feed chunk indices in s3manager-style: queue with
	// buffer = concurrent, strictly increasing offset.
	queue := make(chan int, concurrent)
	go func() {
		for i := 0; i < numChunks; i++ {
			queue <- i
		}
		close(queue)
	}()

	var wg sync.WaitGroup
	for w := 0; w < concurrent; w++ {
		wg.Add(1)
		go func(seed int) {
			defer wg.Done()
			rng := mrand.New(mrand.NewSource(int64(seed)))
			for idx := range queue {
				off := int64(idx * chunkSize)
				// Use Gosched to scramble execution order without sleeping.
				if rng.Intn(4) == 0 {
					runtime.Gosched()
				}
				_, err := b.WriteAt(original[off:off+chunkSize], off)
				if err != nil {
					t.Errorf("WriteAt(%d): %v", off, err)
					return
				}
			}
		}(w)
	}

	go func() {
		wg.Wait()
		b.CloseWithError(nil)
	}()

	got, err := io.ReadAll(b)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !bytes.Equal(got, original) {
		t.Fatalf("byte stream mismatch (len got=%d, want=%d)", len(got), len(original))
	}
}

// TestAdapter_DeadlockDetection verifies that when all expected writer
// goroutines are blocked in WriteAt and the reader has no contiguous bytes,
// the buffer immediately returns a deadlock error to all callers.
func TestAdapter_DeadlockDetection(t *testing.T) {
	// capacity=6, concurrency=2. Neither goroutine writes at offset 0, so the
	// reader will never get sequential data while both are stuck.
	b := NewAdapter(6, WithDeadlockDetection(2))

	errs := make(chan error, 2)
	for _, off := range []int64{6, 12} {
		off := off
		go func() {
			_, err := b.WriteAt([]byte("ABCD"), off)
			errs <- err
		}()
	}

	for i := 0; i < 2; i++ {
		select {
		case err := <-errs:
			if err == nil {
				t.Fatal("expected deadlock error, got nil")
			}
			if !bytes.Contains([]byte(err.Error()), []byte("deadlock")) {
				t.Fatalf("expected deadlock error, got: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("deadlock not detected within timeout — writers hung")
		}
	}
}

// TestAdapter_DeadlockNotFiredWhenReaderHasData verifies that the deadlock
// detector does not fire when all writers are blocked but the reader still
// has data to consume (writeEnd > readPos).
func TestAdapter_DeadlockNotFiredWhenReaderHasData(t *testing.T) {
	// capacity=8, concurrency=1. First goroutine fills offset [0,8) so
	// writeEnd=8. Second goroutine then tries to write beyond capacity and
	// blocks. The deadlock check must NOT fire because writeEnd > readPos.
	b := NewAdapter(8, WithDeadlockDetection(1))

	if _, err := b.WriteAt([]byte("12345678"), 0); err != nil {
		t.Fatalf("first fill: %v", err)
	}

	writerDone := make(chan error, 1)
	go func() {
		_, err := b.WriteAt([]byte("ABCD"), 8)
		writerDone <- err
	}()

	waitForBlockedWriters(b, 1)

	// Reader consumes all 8 bytes, freeing space for the second write.
	if _, err := io.ReadFull(b, make([]byte, 8)); err != nil {
		t.Fatalf("ReadFull: %v", err)
	}

	select {
	case err := <-writerDone:
		if err != nil {
			t.Fatalf("writer returned unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("writer never unblocked")
	}

	b.CloseWithError(nil)
	if _, err := io.ReadAll(b); err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
}

// TestAdapter_RetryWithDifferentReadProgress simulates s3manager body-read
// retry: the same chunk gets re-written at varying reader positions.
func TestAdapter_RetryWithDifferentReadProgress(t *testing.T) {
	chunk := make([]byte, 1000)
	for i := range chunk {
		chunk[i] = byte(i)
	}

	for _, readBefore := range []int{0, 1, 100, 500, 999, 1000} {
		t.Run("", func(t *testing.T) {
			b := NewAdapter(4096)
			if _, err := b.WriteAt(chunk, 0); err != nil {
				t.Fatalf("first WriteAt: %v", err)
			}
			if readBefore > 0 {
				head := make([]byte, readBefore)
				if _, err := io.ReadFull(b, head); err != nil {
					t.Fatalf("ReadFull(%d): %v", readBefore, err)
				}
				if !bytes.Equal(head, chunk[:readBefore]) {
					t.Fatalf("head mismatch")
				}
			}
			if _, err := b.WriteAt(chunk, 0); err != nil {
				t.Fatalf("retry WriteAt: %v", err)
			}
			b.CloseWithError(nil)
			tail, err := io.ReadAll(b)
			if err != nil {
				t.Fatalf("ReadAll: %v", err)
			}
			if !bytes.Equal(tail, chunk[readBefore:]) {
				t.Fatalf("tail mismatch at readBefore=%d", readBefore)
			}
		})
	}
}

// TestAdapter_WriteLargerThanCapacity verifies that writes exceeding total
// buffer capacity immediately return an error rather than hanging.
func TestAdapter_WriteLargerThanCapacity(t *testing.T) {
	b := NewAdapter(10)
	n, err := b.WriteAt([]byte("12345678901"), 0)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if n != 0 {
		t.Fatalf("expected n=0, got %d", n)
	}
	if err.Error() != "ringstream: write larger than buffer capacity" {
		t.Fatalf("unexpected error message: %v", err)
	}

	// Write 15 bytes where 5 bytes are before readPos.
	// Since the remaining 10 bytes fit in capacity, this should NOT fail.
	b.readPos = 5
	b.written.frontier = 5
	n, err = b.WriteAt([]byte("123456789012345"), 0)
	if err != nil {
		t.Fatalf("expected success when sliced payload fits capacity, got: %v", err)
	}
	if n != 15 {
		t.Fatalf("expected n=15, got %d", n)
	}
}

// TestAdapter_ZeroLengthOperations verifies empty Read and WriteAt calls do
// nothing and return (0, nil).
func TestAdapter_ZeroLengthOperations(t *testing.T) {
	b := NewAdapter(10)
	n, err := b.WriteAt(nil, 5)
	if n != 0 || err != nil {
		t.Fatalf("WriteAt(nil) returned n=%d err=%v", n, err)
	}
	n, err = b.WriteAt([]byte{}, 5)
	if n != 0 || err != nil {
		t.Fatalf("WriteAt([]byte{}) returned n=%d err=%v", n, err)
	}

	n, err = b.Read(nil)
	if n != 0 || err != nil {
		t.Fatalf("Read(nil) returned n=%d err=%v", n, err)
	}
	n, err = b.Read([]byte{})
	if n != 0 || err != nil {
		t.Fatalf("Read([]byte{}) returned n=%d err=%v", n, err)
	}
}

// TestAdapter_ComplexHoleMerging verifies that writing disjoint segments out of
// order correctly updates the internal holes and contiguous writeEnd without
// losing data.

// Package ringstream adapts an out-of-order io.WriterAt producer (e.g.
// aws-sdk-go's s3manager.Downloader) into a sequential io.Reader consumer
// (e.g. zstd.NewReader) using a fixed-capacity in-memory ring buffer.
//
// WriteAt blocks when the requested write range would overrun the read head;
// Read blocks when no contiguous bytes are available past the read head.
//
// On s3manager body-read retry, the same offset may be re-written after part
// of the previous attempt's bytes were already consumed by the reader. Adapter
// silently skips bytes that fall before readPos in such a re-write — the
// source blob is immutable, so the reader has already seen the correct bytes.
package ringstream

import (
	"errors"
	"fmt"
	"io"
	"sync"
)

var errClosed = errors.New("ringstream: buffer closed")

// Adapter is a fixed-capacity ring buffer implementing both io.WriterAt and
// io.Reader. Concurrent calls to WriteAt are safe. A single goroutine is
// expected to call Read.
type Adapter struct {
	ring        ringBuf
	capacity    int64
	concurrency int

	mu             sync.Mutex
	cond           *sync.Cond
	readPos        int64
	written        intervalSet
	closed         bool
	err            error
	blockedWriters int
}

// Option configures a Adapter at construction time.
type Option func(*Adapter)

// WithDeadlockDetection sets the expected number of goroutines that call WriteAt
// concurrently. When all concurrency goroutines are blocked in WriteAt and
// the reader has no sequential data to consume (writeEnd == readPos), the
// buffer immediately closes itself with a deadlock error and returns it to
// all blocked callers. Zero (the default) disables this detection.
//
// Pass the s3manager Concurrency parameter here so that deadlocks caused by
// an insufficient ring buffer capacity are caught immediately.
func WithDeadlockDetection(concurrency int) Option {
	return func(b *Adapter) { b.concurrency = concurrency }
}

// New returns a Adapter with the given capacity in bytes. capacity must be
// strictly greater than the largest single WriteAt the producer can issue.
func NewAdapter(capacity int, opts ...Option) *Adapter {
	b := &Adapter{
		ring:     newRingBuf(capacity),
		capacity: int64(capacity),
	}
	b.cond = sync.NewCond(&b.mu)
	for _, opt := range opts {
		opt(b)
	}
	return b
}

// CloseWithError signals end-of-stream to the reader. A nil err means clean
// EOF after the reader drains all written bytes; a non-nil err propagates to
// both Read and any blocked WriteAt calls.
func (b *Adapter) CloseWithError(err error) {
	b.mu.Lock()
	if !b.closed {
		b.closed = true
		b.err = err
	}
	b.cond.Broadcast()
	b.mu.Unlock()
}

// WriteAt implements io.WriterAt. Blocks when off+len(p) would overrun
// readPos+capacity; returns the buffer's error if it is closed before space
// becomes available.
func (b *Adapter) WriteAt(p []byte, off int64) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		if b.err != nil {
			return 0, b.err
		}
		return 0, errClosed
	}

	originalLen := len(p)

	// Skip prefix that has already been consumed by the reader (retry path).
	if off < b.readPos {
		skip := b.readPos - off
		if skip >= int64(len(p)) {
			return originalLen, nil
		}
		p = p[skip:]
		off = b.readPos
	}

	if int64(len(p)) > b.capacity {
		return 0, errors.New("ringstream: write larger than buffer capacity")
	}

	end := off + int64(len(p))
	for end > b.readPos+b.capacity {
		if b.closed {
			if b.err != nil {
				return 0, b.err
			}
			return 0, errClosed
		}
		b.blockedWriters++
		if b.concurrency > 0 && b.blockedWriters >= b.concurrency && b.written.frontier == b.readPos {
			// All known writer goroutines are blocked and the reader has no
			// contiguous bytes to consume — this is an unresolvable deadlock.
			// Close the buffer so all other blocked writers also wake up.
			deadlockErr := fmt.Errorf(
				"ringstream: deadlock: all %d writer goroutines are blocked "+
					"and reader has no data (readPos=%d, writeEnd=%d, off=%d, cap=%d)",
				b.concurrency, b.readPos, b.written.frontier, off, b.capacity,
			)
			if !b.closed {
				b.closed = true
				b.err = deadlockErr
			}
			b.cond.Broadcast()
			b.blockedWriters--
			return 0, deadlockErr
		}
		b.cond.Wait()
		b.blockedWriters--
		// Re-resolve after wakeup in case readPos advanced past off.
		if off < b.readPos {
			skip := b.readPos - off
			if skip >= int64(len(p)) {
				return originalLen, nil
			}
			p = p[skip:]
			off = b.readPos
			end = off + int64(len(p))
		}
	}

	b.ring.write(off, p)
	b.written.add(off, end)
	b.cond.Broadcast()
	return originalLen, nil
}

// Read implements io.Reader. Returns io.EOF after CloseWithError(nil) when
// the reader has drained all contiguous bytes.
func (b *Adapter) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for b.readPos >= b.written.frontier {
		if b.closed {
			if b.err != nil {
				return 0, b.err
			}
			return 0, io.EOF
		}
		b.cond.Wait()
	}

	available := b.written.frontier - b.readPos
	n := min(int64(len(p)), available)

	b.ring.read(b.readPos, p[:n])
	b.readPos += n
	b.cond.Broadcast()
	return int(n), nil
}

package s3

import (
	"context"
	"io"
)

// rangeFetcher fetches bytes [start, end] (inclusive) of a blob.
type rangeFetcher func(ctx context.Context, start, end int64) ([]byte, error)

type part struct {
	start, end int64
	ready      chan struct{}
	data       []byte
	err        error
}

// rangedReader streams a blob sequentially while a fixed pool of workers
// fetches parts ahead — the same structure the aws downloader uses internally
// (https://github.com/aws/aws-sdk-go/blob/main/service/s3/s3manager/download.go),
// except parts are handed to the reader through an in-order channel instead of
// being scatter-written into a WriterAt. The channel capacity bounds the
// buffered-but-unconsumed parts, so memory is limited to about
// (concurrency+1) parts; a lagging reader backpressures the fetchers.
// rangedReader is not safe for concurrent Read calls, but Close may be called
// concurrently with Read: Close only cancels the context, and Read observes
// the cancellation.
type rangedReader struct {
	ctx    context.Context
	cancel context.CancelFunc
	parts  chan *part
	cur    *part
	off    int
	err    error
}

// newRangedReader streams a blob of total bytes, reusing the already fetched
// first part. The reader owns cancel and releases everything on Close.
func newRangedReader(
	ctx context.Context,
	fetch rangeFetcher,
	first []byte,
	total int64,
	concurrency int,
	partSize int64,
) io.ReadCloser {
	ctx, cancel := context.WithCancel(ctx)
	r := &rangedReader{
		ctx:    ctx,
		cancel: cancel,
		parts:  make(chan *part, concurrency+1),
		cur:    &part{data: first},
	}

	remaining := total - int64(len(first))
	workers := min(int64(concurrency), (remaining+partSize-1)/partSize)

	jobs := make(chan *part)
	for i := int64(0); i < workers; i++ {
		go func() {
			for p := range jobs {
				p.data, p.err = fetch(ctx, p.start, p.end)
				close(p.ready)
			}
		}()
	}

	go func() {
		defer close(r.parts)
		defer close(jobs)

		for start := int64(len(first)); start < total; start += partSize {
			p := &part{
				start: start,
				end:   min(start+partSize, total) - 1,
				ready: make(chan struct{}),
			}
			select {
			case r.parts <- p:
			case <-ctx.Done():
				return
			}
			select {
			case jobs <- p:
			case <-ctx.Done():
				p.err = ctx.Err()
				close(p.ready)
				return
			}
		}
	}()

	return r
}

func (r *rangedReader) Read(p []byte) (int, error) {
	if r.err != nil {
		return 0, r.err
	}
	if err := r.ctx.Err(); err != nil {
		// Closed or canceled: the stream's integrity is void even if parts
		// are still buffered.
		r.err = err
		return 0, err
	}

	if r.cur == nil {
		var next *part
		var ok bool
		select {
		case next, ok = <-r.parts:
		case <-r.ctx.Done():
			r.err = r.ctx.Err()
			return 0, r.err
		}
		if !ok {
			// The dispatcher also closes the channel when the context dies
			// mid-stream; that must not read as a clean end of stream.
			if err := r.ctx.Err(); err != nil {
				r.err = err
				return 0, err
			}
			r.err = io.EOF
			return 0, io.EOF
		}
		select {
		case <-next.ready:
		case <-r.ctx.Done():
			r.err = r.ctx.Err()
			return 0, r.err
		}
		if next.err != nil {
			r.err = next.err
			r.cancel()
			return 0, r.err
		}
		r.cur, r.off = next, 0
	}

	n := copy(p, r.cur.data[r.off:])
	r.off += n
	if r.off == len(r.cur.data) {
		r.cur = nil
	}
	return n, nil
}

func (r *rangedReader) Close() error {
	r.cancel()
	return nil
}

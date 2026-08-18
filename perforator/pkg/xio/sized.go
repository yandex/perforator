package xio

import (
	"errors"
	"fmt"
	"io"
)

// NewSizedReader reads exactly size bytes from src: it fails if src ends
// before that, and fails if src has more to give after that.
func NewSizedReader(src io.Reader, size int64) io.Reader {
	return &sizedReader{src: src, remaining: size}
}

type sizedReader struct {
	src       io.Reader
	remaining int64
}

func (r *sizedReader) Read(p []byte) (int, error) {
	n, err := r.src.Read(p)

	switch {
	case err != nil && err != io.EOF:
		return n, err
	case err == io.EOF && int64(n) < r.remaining:
		return n, fmt.Errorf("stream ends %d bytes short of its declared size", r.remaining-int64(n))
	case int64(n) > r.remaining:
		// Hand over what was declared and nothing more.
		return int(r.remaining), errors.New("stream continues past its declared size")
	}

	r.remaining -= int64(n)
	return n, err
}

// NewReadMultiCloser returns a ReadCloser reading from r and closing closers
// in the order given, which for a stack of wrappers is outermost first.
func NewReadMultiCloser(r io.Reader, closers ...io.Closer) io.ReadCloser {
	return &readMultiCloser{Reader: r, closers: closers}
}

type readMultiCloser struct {
	io.Reader
	closers []io.Closer
}

func (r *readMultiCloser) Close() error {
	var err error
	for _, closer := range r.closers {
		err = errors.Join(err, closer.Close())
	}
	return err
}

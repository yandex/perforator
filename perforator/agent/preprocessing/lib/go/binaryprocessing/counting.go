package binaryprocessing

import "io"

type countingReader struct {
	reader io.Reader
	count  int
}

func (r *countingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.count += n
	return n, err
}

func (r *countingReader) Count() int {
	return r.count
}

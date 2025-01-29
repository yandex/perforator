package util

import (
	"io"

	"github.com/aws/aws-sdk-go/aws"
)

////////////////////////////////////////////////////////////////////////////////

type WriteAtBuffer = aws.WriteAtBuffer

func NewWriteAtBuffer(buf []byte) *WriteAtBuffer {
	w := aws.NewWriteAtBuffer(buf)
	w.GrowthCoeff = 1.5
	return w
}

////////////////////////////////////////////////////////////////////////////////

func WrapWriterAt(w io.WriterAt) io.Writer {
	return &plainWriterAdapter{w, 0}
}

type plainWriterAdapter struct {
	wrt io.WriterAt
	off int64
}

func (w *plainWriterAdapter) Write(buf []byte) (int, error) {
	n, err := w.wrt.WriteAt(buf, w.off)
	w.off += int64(n)
	return n, err
}

////////////////////////////////////////////////////////////////////////////////

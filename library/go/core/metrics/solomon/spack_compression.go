package solomon

import (
	"encoding/binary"
	"io"
	"sync"

	"github.com/OneOfOne/xxhash"
	"github.com/pierrec/lz4"
)

type CompressionType uint8

const (
	CompressionNone CompressionType = 0x0
	CompressionZlib CompressionType = 0x1
	CompressionZstd CompressionType = 0x2
	CompressionLz4  CompressionType = 0x3
)

const (
	compressionFrameLength = 512 * 1024
	hashTableSize          = 64 * 1024
)

type noCompressionWriteCloser struct {
	underlying io.Writer
	written    int
}

func (w *noCompressionWriteCloser) Write(p []byte) (int, error) {
	n, err := w.underlying.Write(p)
	w.written += n
	return n, err
}

func (w *noCompressionWriteCloser) Close() error {
	return nil
}

type lz4CompressionWriteCloser struct {
	underlying io.Writer
	buffer     []byte
	table      []int
	dst        []byte
	written    int
}

var lz4WriterPool = sync.Pool{
	New: func() any {
		return &lz4CompressionWriteCloser{
			buffer: make([]byte, 0, compressionFrameLength),
			table:  make([]int, hashTableSize),
			dst:    make([]byte, lz4.CompressBlockBound(compressionFrameLength)),
		}
	},
}

func acquireLZ4Writer(w io.Writer) *lz4CompressionWriteCloser {
	cw := lz4WriterPool.Get().(*lz4CompressionWriteCloser)
	cw.underlying = w
	cw.buffer = cw.buffer[:0]
	cw.written = 0
	return cw
}

func releaseLZ4Writer(cw *lz4CompressionWriteCloser) {
	cw.underlying = nil
	lz4WriterPool.Put(cw)
}

func (w *lz4CompressionWriteCloser) flushFrame() (written int, err error) {
	src := w.buffer
	need := lz4.CompressBlockBound(len(src))
	if cap(w.dst) < need {
		w.dst = make([]byte, need)
	} else {
		w.dst = w.dst[:need]
	}

	sz, err := lz4.CompressBlock(src, w.dst, w.table)
	if err != nil {
		return written, err
	}

	var out []byte
	if sz == 0 {
		out = src
	} else {
		out = w.dst[:sz]
	}

	var lenBuf [8]byte
	binary.LittleEndian.PutUint32(lenBuf[0:4], uint32(len(out)))
	binary.LittleEndian.PutUint32(lenBuf[4:8], uint32(len(src)))
	if _, err := w.underlying.Write(lenBuf[:]); err != nil {
		return written, err
	}
	w.written += 8

	n, err := w.underlying.Write(out)
	if err != nil {
		return written, err
	}
	w.written += n

	checksum := xxhash.Checksum32S(out, 0x1337c0de)
	var sumBuf [4]byte
	binary.LittleEndian.PutUint32(sumBuf[:], checksum)
	if _, err := w.underlying.Write(sumBuf[:]); err != nil {
		return written, err
	}
	w.written += 4

	w.buffer = w.buffer[:0]

	return written, nil
}

func (w *lz4CompressionWriteCloser) Write(p []byte) (written int, err error) {
	q := p[:]
	for len(q) > 0 {
		space := compressionFrameLength - len(w.buffer)
		if space == 0 {
			n, err := w.flushFrame()
			if err != nil {
				return written, err
			}
			w.written += n
			space = compressionFrameLength
		}
		length := len(q)
		if length > space {
			length = space
		}
		w.buffer = append(w.buffer, q[:length]...)
		q = q[length:]
	}
	return written, nil
}

func (w *lz4CompressionWriteCloser) Close() error {
	if len(w.buffer) > 0 {
		n, err := w.flushFrame()
		if err != nil {
			return err
		}
		w.written += n
	}

	var trailer [12]byte
	if _, err := w.underlying.Write(trailer[:]); err != nil {
		return err
	}
	w.written += 12

	return nil
}

func newCompressedWriter(w io.Writer, compression CompressionType) io.WriteCloser {
	switch compression {
	case CompressionNone:
		return &noCompressionWriteCloser{w, 0}
	case CompressionZlib:
		panic("zlib compression not supported")
	case CompressionZstd:
		panic("zstd compression not supported")
	case CompressionLz4:
		return acquireLZ4Writer(w)
	default:
		panic("unsupported compression algorithm")
	}
}

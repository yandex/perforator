package asyncfilecache

import (
	"errors"
	"fmt"
	"os"
	"sync/atomic"
)

var (
	ErrSizeMismatch  = errors.New("size mismatch")
	ErrNotAbsentFile = errors.New("cannot open file as its FS state is not `absent`")
)

// thread-safe
type WriterAt struct {
	tmpPath     string
	finalPath   string
	size        uint64
	writtenSize atomic.Uint64
	file        *os.File

	entry *cacheEntry
}

func newSizeMismatchError(path string, written, expected uint64) error {
	return fmt.Errorf(
		"%v: path %s, %d vs %d (written vs expected)",
		ErrSizeMismatch,
		path,
		written,
		expected,
	)
}

func newWriter(entry *cacheEntry) (*WriterAt, error) {
	var err error

	writer := &WriterAt{
		finalPath:   entry.finalPath,
		tmpPath:     entry.finalPath + TmpSuffix,
		size:        entry.size,
		writtenSize: atomic.Uint64{},

		entry: entry,
	}

	opened := entry.tryOpen()
	if !opened {
		return nil, ErrNotAbsentFile
	}
	defer func() {
		if err != nil {
			entry.setFSState(WriteFailed)
		}
	}()

	var file *os.File
	file, err = os.Create(writer.tmpPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file cache writer for %s: %w", writer.tmpPath, err)
	}
	writer.file = file

	return writer, nil
}

func (w *WriterAt) updateWrittenSize(rb uint64) {
	for {
		writtenSize := w.writtenSize.Load()
		if rb > writtenSize {
			if !w.writtenSize.CompareAndSwap(writtenSize, rb) {
				continue
			}
		}

		break
	}
}

// thread-safe with itself
func (w *WriterAt) WriteAt(p []byte, off int64) (int, error) {
	rb := uint64(len(p)) + uint64(off)
	if rb > w.size {
		return 0, newSizeMismatchError(w.tmpPath, rb, w.size)
	}

	n, err := w.file.WriteAt(p, off)
	w.updateWrittenSize(rb)

	return n, err
}

// not thread-safe with WriteAt
func (w *WriterAt) closeImpl() (err error) {
	defer w.file.Close()
	defer func() {
		if err != nil {
			w.entry.setFSState(WriteFailed)
		} else {
			w.entry.setFSState(Stored)
		}
	}()

	writtenSize := w.writtenSize.Load()
	if writtenSize != w.size {
		err = newSizeMismatchError(w.tmpPath, writtenSize, w.size)
		return
	}

	err = os.Rename(w.tmpPath, w.finalPath)
	if err != nil {
		err = fmt.Errorf("failed to rename tmp file in cache: %s -> %s: %w", w.tmpPath, w.finalPath, err)
		return
	}

	return
}

func (w *WriterAt) Close() error {
	w.entry.closeWriter.Do(func() {
		w.entry.closeError = w.closeImpl()
	})

	return w.entry.closeError
}

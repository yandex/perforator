package atomicfs

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

////////////////////////////////////////////////////////////////////////////////

type File struct {
	tmpfile   *os.File
	dstpath   string
	atomicity AtomicityLevel
}

////////////////////////////////////////////////////////////////////////////////

type FileOption func(f *File) error

func WithSync() FileOption {
	return func(f *File) error {
		f.atomicity = AtomicityFull
		return nil
	}
}

func WithMode(mode os.FileMode) FileOption {
	return func(f *File) error {
		return f.tmpfile.Chmod(mode)
	}
}

////////////////////////////////////////////////////////////////////////////////

const tmpsuffix = ".tmp-"

func Create(path string, opts ...FileOption) (f *File, err error) {
	path, err = filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("failed to make tmp file name: %w", err)
	}
	dir, base := filepath.Split(path)

	tmpf, err := os.CreateTemp(dir, base+tmpsuffix)
	if err != nil {
		return nil, err
	}

	f = &File{tmpf, path, AtomicityNoSync}
	defer func() {
		if err != nil {
			_ = f.Discard()
		}
	}()

	// This finalizer is not required, but in some cases this allows us to reduce
	// the number of lost tmp files: File.Discard will remove uncommited tmp file.
	runtime.SetFinalizer(f, (*File).Discard)

	for _, opt := range opts {
		err = opt(f)
		if err != nil {
			return
		}
	}

	return f, nil
}

func (f *File) Write(data []byte) (int, error) {
	return f.tmpfile.Write(data)
}

func (f *File) WriteAt(data []byte, offset int64) (int, error) {
	return f.tmpfile.WriteAt(data, offset)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	return f.tmpfile.Seek(offset, whence)
}

func (f *File) Discard() error {
	if f.tmpfile == nil {
		return nil
	}
	defer func() {
		f.tmpfile = nil
	}()

	err := f.tmpfile.Close()
	if err != nil {
		return err
	}

	return os.Remove(f.tmpfile.Name())
}

func (f *File) Close() (err error) {
	if f.tmpfile == nil {
		return fmt.Errorf("calling atomicfs.File.Close on already finished atomicfs.File")
	}
	defer func() {
		if err != nil {
			_ = f.Discard()
		} else {
			f.tmpfile = nil
		}
	}()

	if f.atomicity != AtomicityNoSync {
		err = f.tmpfile.Sync()
		if err != nil {
			return err
		}
	}

	err = f.tmpfile.Close()
	if err != nil {
		return err
	}

	return os.Rename(f.tmpfile.Name(), f.dstpath)
}

////////////////////////////////////////////////////////////////////////////////

var _ io.Writer = (*File)(nil)
var _ io.WriterAt = (*File)(nil)
var _ io.WriteCloser = (*File)(nil)
var _ io.WriteSeeker = (*File)(nil)

////////////////////////////////////////////////////////////////////////////////

//go:build linux

package remotemem

import (
	"errors"
	"fmt"

	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/linetable"
	"github.com/yandex/perforator/perforator/pkg/linux/procmem"
)

const (
	// maxLinetableSize is the maximum number of bytes we'll read from
	// co_linetable. Anything larger is almost certainly a corrupted/reused
	// PyBytesObject.
	maxLinetableSize = 1 << 20
)

// Reader reads Python objects from a target process via [procmem.Read].
// It is stateless; New returns a zero-value struct so we can add fields
// (e.g. metrics) later without breaking callers.
type Reader struct{}

// New returns a new Reader.
func New() *Reader {
	return &Reader{}
}

// CodeObjectOffsets is the minimal subset of unwinder.PythonInternalsOffsets
// fields needed by ReadCodeLinetable. Pass a freshly-constructed value rather
// than the whole offsets struct so the API is decoupled from the generated type.
type CodeObjectOffsets struct {
	CoFirstlineno uint32
	CoLinetable   uint32
	BytesObSize   uint32 // PyBytesObject.ob_size
	BytesObSval   uint32 // PyBytesObject.ob_sval
}

// ReadCodeLinetable validates the PyCodeObject hasn't been freed/reused, then
// reads co_linetable bytes from the target process and returns a parsed
// [linetable.LocationTable].
//
//  1. Re-read co_firstlineno at codeObjectAddr+CoFirstlineno. If it does not
//     match expectedFirstlineno → return ErrCodeObjectChanged (do not read further).
//  2. Re-read co_linetable PyObject* at codeObjectAddr+CoLinetable. If it does
//     not match expectedLinetableAddr → return ErrCodeObjectChanged.
//  3. Read PyBytesObject.ob_size at expectedLinetableAddr+BytesObSize
//     (Py_ssize_t = int64).
//  4. Read ob_size bytes starting at expectedLinetableAddr+BytesObSval.
//  5. Unmarshal into LocationTable.
//
// Sanity-bound ob_size — refuse anything implausibly large (> 1<<20) or negative.
// BPF/wire addresses arrive as uint64; convert to uintptr at the call site.
func (r *Reader) ReadCodeLinetable(
	pid uint32,
	codeObjectAddr uintptr,
	expectedLinetableAddr uintptr,
	offsets CodeObjectOffsets,
	expectedFirstlineno int32,
) (linetable.LocationTable, error) {
	// Step 1: validate co_firstlineno to detect freed/reused PyCodeObject.
	actualFirstlineno, err := procmem.ReadScalar[int32](int(pid), codeObjectAddr+uintptr(offsets.CoFirstlineno))
	if err != nil {
		return linetable.LocationTable{}, fmt.Errorf("read co_firstlineno: %w", err)
	}
	if actualFirstlineno != expectedFirstlineno {
		return linetable.LocationTable{}, ErrCodeObjectChanged
	}

	// Step 2: read co_linetable PyObject*.
	actualLinetableAddr, err := procmem.ReadScalar[uintptr](int(pid), codeObjectAddr+uintptr(offsets.CoLinetable))
	if err != nil {
		return linetable.LocationTable{}, fmt.Errorf("read co_linetable pointer: %w", err)
	}
	if actualLinetableAddr != expectedLinetableAddr {
		return linetable.LocationTable{}, ErrCodeObjectChanged
	}
	if expectedLinetableAddr == 0 {
		return linetable.LocationTable{}, errors.New("python: co_linetable is NULL")
	}

	// Step 3: read PyBytesObject.ob_size (Py_ssize_t = int64 on amd64).
	obSize, err := procmem.ReadScalar[int64](int(pid), expectedLinetableAddr+uintptr(offsets.BytesObSize))
	if err != nil {
		return linetable.LocationTable{}, fmt.Errorf("read PyBytesObject.ob_size: %w", err)
	}
	if obSize < 0 {
		return linetable.LocationTable{}, fmt.Errorf("python: PyBytesObject.ob_size is negative (%d)", obSize)
	}
	if obSize > maxLinetableSize {
		return linetable.LocationTable{}, fmt.Errorf("python: PyBytesObject.ob_size too large (%d > %d)", obSize, maxLinetableSize)
	}

	// Step 4: read ob_size bytes starting at ob_sval.
	var buf []byte
	if obSize > 0 {
		buf = make([]byte, obSize)
		if err := procmem.Read(int(pid), expectedLinetableAddr+uintptr(offsets.BytesObSval), buf); err != nil {
			return linetable.LocationTable{}, fmt.Errorf("read PyBytesObject.ob_sval: %w", err)
		}
	}

	// Step 5: wrap as LocationTable (fails on empty payload).
	table, err := linetable.Unmarshal(buf, expectedFirstlineno)
	if err != nil {
		return linetable.LocationTable{}, err
	}
	return table, nil
}

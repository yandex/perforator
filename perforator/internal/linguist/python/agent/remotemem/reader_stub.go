//go:build !linux

package remotemem

import (
	"fmt"

	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/linetable"
)

// Reader is a stub on non-Linux platforms.
type Reader struct{}

func New() *Reader {
	return &Reader{}
}

// CodeObjectOffsets mirrors the Linux definition.
type CodeObjectOffsets struct {
	CoFirstlineno uint32
	CoLinetable   uint32
	BytesObSize   uint32
	BytesObSval   uint32
}

func (r *Reader) ReadCodeLinetable(
	pid uint32,
	codeObjectAddr uintptr,
	expectedLinetableAddr uintptr,
	offsets CodeObjectOffsets,
	expectedFirstlineno int32,
) (linetable.LocationTable, error) {
	return linetable.LocationTable{}, fmt.Errorf("python remotemem: unsupported platform")
}

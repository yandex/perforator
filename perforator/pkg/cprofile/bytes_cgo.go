//go:build cgo

package cprofile

import (
	"fmt"
	"unsafe"
)

func checkedGoBytesLen(size uintptr) (int, error) {
	maxInt := int(^uint(0) >> 1)
	if size > uintptr(maxInt) {
		return 0, fmt.Errorf("C buffer of %d bytes does not fit into a Go slice on this architecture", size)
	}
	return int(size), nil
}

// copyCBytes copies a C-owned buffer into Go-owned memory without narrowing its
// size to C.int. Sizes that cannot be represented by a Go slice fail explicitly.
// The caller must keep the buffer alive until this function returns.
func copyCBytes(ptr unsafe.Pointer, size uintptr) ([]byte, error) {
	length, err := checkedGoBytesLen(size)
	if err != nil {
		return nil, err
	}
	if length == 0 {
		return []byte{}, nil
	}
	if ptr == nil {
		return nil, fmt.Errorf("C buffer has nil data pointer with non-zero size %d", size)
	}

	result := make([]byte, length)
	copy(result, unsafe.Slice((*byte)(ptr), length))
	return result, nil
}

//go:build cgo

package cprofile

import (
	"bytes"
	"testing"
	"unsafe"
)

func TestCopyCBytes(t *testing.T) {
	for _, source := range [][]byte{{0}, {0xff, 0, 0x80}, []byte("profile data")} {
		result, err := copyCBytes(unsafe.Pointer(&source[0]), uintptr(len(source)))
		if err != nil {
			t.Fatalf("copyCBytes failed: %v", err)
		}
		if !bytes.Equal(result, source) {
			t.Fatalf("copyCBytes returned %v, want %v", result, source)
		}
		result[0] ^= 0xff
		if source[0] == result[0] {
			t.Fatal("copyCBytes result aliases its source")
		}
	}
}

func TestCopyCBytesEmpty(t *testing.T) {
	var source byte
	for _, ptr := range []unsafe.Pointer{nil, unsafe.Pointer(&source)} {
		result, err := copyCBytes(ptr, 0)
		if err != nil {
			t.Fatalf("copyCBytes failed: %v", err)
		}
		if result == nil || len(result) != 0 {
			t.Fatalf("copyCBytes returned %#v, want a non-nil empty slice", result)
		}
	}
}

func TestCopyCBytesRejectsNilData(t *testing.T) {
	if _, err := copyCBytes(nil, 1); err == nil {
		t.Fatal("copyCBytes accepted nil data with a non-zero size")
	}
}

func TestCopyCBytesRejectsSizeBeyondGoInt(t *testing.T) {
	maxInt := uintptr(^uint(0) >> 1)
	var source byte
	for _, size := range []uintptr{maxInt + 1, ^uintptr(0)} {
		if _, err := checkedGoBytesLen(size); err == nil {
			t.Fatalf("checkedGoBytesLen accepted size %d beyond Go int", size)
		}
		if _, err := copyCBytes(unsafe.Pointer(&source), size); err == nil {
			t.Fatalf("copyCBytes accepted size %d beyond Go int", size)
		}
	}
}

func TestCheckedGoBytesLenBoundaries(t *testing.T) {
	maxInt := uintptr(^uint(0) >> 1)
	// Check lengths without allocating buffers of these sizes.
	sizes := []uintptr{0, 1, 1<<31 - 1, maxInt}
	if unsafe.Sizeof(uintptr(0)) >= 8 {
		sizes = append(sizes, 1<<31, 1<<31+1)
	}
	for _, size := range sizes {
		length, err := checkedGoBytesLen(size)
		if err != nil {
			t.Fatalf("size %d was rejected: %v", size, err)
		}
		if uintptr(length) != size {
			t.Fatalf("size was narrowed to %d, want %d", length, size)
		}
	}
}

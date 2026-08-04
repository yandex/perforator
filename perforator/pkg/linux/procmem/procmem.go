package procmem

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Read copies exactly len(dst) bytes from remoteAddr in process pid into dst
// using process_vm_readv. An empty dst is a no-op.
func Read(pid int, remoteAddr uintptr, dst []byte) error {
	if len(dst) == 0 {
		return nil
	}

	n, err := unix.ProcessVMReadv(
		pid,
		[]unix.Iovec{
			{
				Base: &dst[0],
				Len:  uint64(len(dst)),
			},
		},
		[]unix.RemoteIovec{
			{
				Base: remoteAddr,
				Len:  len(dst),
			},
		},
		0,
	)
	if err != nil {
		return fmt.Errorf("process_vm_readv pid=%d addr=0x%x: %w", pid, remoteAddr, err)
	}
	if n != len(dst) {
		return fmt.Errorf("process_vm_readv pid=%d addr=0x%x: short read (got %d, want %d)", pid, remoteAddr, n, len(dst))
	}
	return nil
}

// ReadScalar reads a fixed-size integer of type T from remoteAddr in process pid
// using the host native endianness (same layout as in the target process).
func ReadScalar[T ~uintptr | uint64 | uint32 | uint16 | uint8 | int64 | int32 | int16](pid int, remoteAddr uintptr) (T, error) {
	var out T
	buf := unsafe.Slice((*byte)(unsafe.Pointer(&out)), unsafe.Sizeof(out))
	if err := Read(pid, remoteAddr, buf); err != nil {
		return 0, err
	}
	return out, nil
}

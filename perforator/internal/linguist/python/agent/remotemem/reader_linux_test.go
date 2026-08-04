//go:build linux

package remotemem

import (
	"encoding/binary"
	"errors"
	"math"
	"os"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/linetable"
	"github.com/yandex/perforator/perforator/pkg/linux/procmem"
)

func selfPID() uint32 {
	return uint32(os.Getpid())
}

// TestProcmem_ReadScalar_OK reads a uint32 from the current process's own memory.
func TestProcmem_ReadScalar_OK(t *testing.T) {
	arr := [1]uint32{0xdeadbeef}
	addr := uintptr(unsafe.Pointer(&arr[0]))

	got, err := procmem.ReadScalar[uint32](int(selfPID()), addr)
	require.NoError(t, err)
	require.Equal(t, arr[0], got)

	// Keep arr alive past the call.
	_ = arr
}

// TestReader_ReadCodeLinetable_OK constructs a synthetic PyCodeObject and
// PyBytesObject in Go heap memory and verifies ReadCodeLinetable retrieves and
// unmarshals the payload.
func TestReader_ReadCodeLinetable_OK(t *testing.T) {
	const coFirstlinenoOff uint32 = 0
	const coLinetableOff uint32 = 8
	const bytesObSizeOff uint32 = 0
	const bytesObSvalOff uint32 = 8

	payload := []byte{0x80, 0x01, 0x02, 0x03}
	const firstlineno int32 = 42

	// Fake PyBytesObject: [ob_size uint64 | ob_sval bytes...]
	bytesObjBuf := make([]byte, int(bytesObSvalOff)+len(payload))
	binary.LittleEndian.PutUint64(bytesObjBuf[bytesObSizeOff:], uint64(len(payload)))
	copy(bytesObjBuf[bytesObSvalOff:], payload)
	bytesObjAddr := uintptr(unsafe.Pointer(&bytesObjBuf[0]))

	// Fake PyCodeObject: [co_firstlineno uint32 at 0 | padding | co_linetable ptr at 8]
	codeObjBuf := make([]byte, int(coLinetableOff)+8)
	binary.LittleEndian.PutUint32(codeObjBuf[coFirstlinenoOff:], uint32(firstlineno))
	binary.LittleEndian.PutUint64(codeObjBuf[coLinetableOff:], uint64(bytesObjAddr))
	codeObjAddr := uintptr(unsafe.Pointer(&codeObjBuf[0]))

	offsets := CodeObjectOffsets{
		CoFirstlineno: coFirstlinenoOff,
		CoLinetable:   coLinetableOff,
		BytesObSize:   bytesObSizeOff,
		BytesObSval:   bytesObSvalOff,
	}

	r := New()
	got, err := r.ReadCodeLinetable(selfPID(), codeObjAddr, offsets, firstlineno)
	require.NoError(t, err)
	require.Equal(t, firstlineno, got.FirstLineno)
	require.Equal(t, payload, got.Raw)

	// Keep buffers alive.
	_ = bytesObjBuf
	_ = codeObjBuf
}

// TestReader_ReadCodeLinetable_FirstLinenoMismatch verifies ErrCodeObjectChanged
// is returned when co_firstlineno does not match the expected value.
func TestReader_ReadCodeLinetable_FirstLinenoMismatch(t *testing.T) {
	const coFirstlinenoOff uint32 = 0
	const coLinetableOff uint32 = 8
	const bytesObSizeOff uint32 = 0
	const bytesObSvalOff uint32 = 8

	payload := []byte{0x01, 0x02}
	const firstlineno int32 = 10

	bytesObjBuf := make([]byte, int(bytesObSvalOff)+len(payload))
	binary.LittleEndian.PutUint64(bytesObjBuf[bytesObSizeOff:], uint64(len(payload)))
	copy(bytesObjBuf[bytesObSvalOff:], payload)
	bytesObjAddr := uintptr(unsafe.Pointer(&bytesObjBuf[0]))

	codeObjBuf := make([]byte, int(coLinetableOff)+8)
	binary.LittleEndian.PutUint32(codeObjBuf[coFirstlinenoOff:], uint32(firstlineno))
	binary.LittleEndian.PutUint64(codeObjBuf[coLinetableOff:], uint64(bytesObjAddr))
	codeObjAddr := uintptr(unsafe.Pointer(&codeObjBuf[0]))

	offsets := CodeObjectOffsets{
		CoFirstlineno: coFirstlinenoOff,
		CoLinetable:   coLinetableOff,
		BytesObSize:   bytesObSizeOff,
		BytesObSval:   bytesObSvalOff,
	}

	r := New()
	got, err := r.ReadCodeLinetable(selfPID(), codeObjAddr, offsets, firstlineno+1)
	require.ErrorIs(t, err, ErrCodeObjectChanged)
	require.Equal(t, linetable.LocationTable{}, got)

	_ = bytesObjBuf
	_ = codeObjBuf
}

// TestReader_ReadCodeLinetable_NullLinetablePtr verifies an error (other than
// ErrCodeObjectChanged) is returned when co_linetable is a NULL pointer.
func TestReader_ReadCodeLinetable_NullLinetablePtr(t *testing.T) {
	const coFirstlinenoOff uint32 = 0
	const coLinetableOff uint32 = 8

	const firstlineno int32 = 5

	codeObjBuf := make([]byte, int(coLinetableOff)+8)
	binary.LittleEndian.PutUint32(codeObjBuf[coFirstlinenoOff:], uint32(firstlineno))
	// co_linetable pointer is zero (NULL) — leave bytes as 0.
	codeObjAddr := uintptr(unsafe.Pointer(&codeObjBuf[0]))

	offsets := CodeObjectOffsets{
		CoFirstlineno: coFirstlinenoOff,
		CoLinetable:   coLinetableOff,
		BytesObSize:   0,
		BytesObSval:   8,
	}

	r := New()
	got, err := r.ReadCodeLinetable(selfPID(), codeObjAddr, offsets, firstlineno)
	require.Error(t, err)
	require.False(t, errors.Is(err, ErrCodeObjectChanged))
	require.Equal(t, linetable.LocationTable{}, got)

	_ = codeObjBuf
}

// TestProcmem_ReadScalar_NonexistentPID verifies an error is returned for
// a PID that cannot possibly be running.
func TestProcmem_ReadScalar_NonexistentPID(t *testing.T) {
	arr := [1]uint32{1}
	addr := uintptr(unsafe.Pointer(&arr[0]))

	_, err := procmem.ReadScalar[uint32](math.MaxInt32, addr)
	require.Error(t, err)

	_ = arr
}

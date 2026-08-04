package jvmbindings

import (
	"fmt"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/copy"
	jvmproto "github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
	"github.com/yandex/perforator/perforator/pkg/linux/procmem"
)

type JVM struct {
	jc      *jvmproto.Cheatsheet
	pid     uint32
	version uint32
}

func NewJVM(jc *jvmproto.Cheatsheet, pid uint32, version uint32) *JVM {
	return &JVM{
		jc:      jc,
		pid:     pid,
		version: version,
	}
}

func (j *JVM) Cheatsheet() *jvmproto.Cheatsheet {
	return j.jc
}

func (j *JVM) Version() uint32 {
	return j.version
}

func ReadScalar[T ~uintptr | uint64 | uint32 | uint16 | uint8 | int32 | int16](j *JVM, addr uintptr) (T, error) {
	out, err := procmem.ReadScalar[T](int(j.pid), addr)
	if err != nil {
		return 0, fmt.Errorf("reading at address %x failed: %w", addr, err)
	}
	return out, nil
}

func ReadObjPtr[T ~struct {
	addr uintptr
	j    *JVM
}](j *JVM, addr uintptr) (T, error) {
	newAddr, err := ReadScalar[uintptr](j, addr)
	if err != nil {
		return T{}, err
	}
	return T{
		addr: newAddr,
		j:    j,
	}, nil
}

func (j *JVM) ReadBytes(addr uintptr, count int) ([]byte, error) {
	out := make([]byte, count)
	if err := procmem.Read(int(j.pid), addr, out); err != nil {
		return nil, fmt.Errorf("reading %d bytes at address %x failed: %w", count, addr, err)
	}
	return out, nil
}

func (j *JVM) ReadString(addr Zstring, limit int) ([]byte, error) {
	// TODO: if limit is big (i.e. much larger than expected actual length), it may be better to read in chunks
	buf, err := j.ReadBytes(uintptr(addr.addr), limit)
	if err != nil {
		return nil, err
	}
	// TODO: this probably does two extra copies
	return []byte(copy.ZeroTerminatedString(buf)), nil
}

func (j *JVM) MakeObjPointer(addr uintptr) ObjectPtr {
	return ObjectPtr{
		addr: addr,
		j:    j,
	}
}

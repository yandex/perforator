package jvmbindings

func (ch CodeHeap) NextSegment() (uint64, error) {
	return ReadScalar[uint64](ch.j, ch.addr+uintptr(*ch.j.jc.CodeHeapNextSegment))
}

func (ch CodeHeap) Memory() VirtualSpace {
	return VirtualSpace{addr: ch.addr + uintptr(*ch.j.jc.CodeHeapMemory)}
}

func (ch CodeHeap) Log2SegmentSize() (int32, error) {
	return ReadScalar[int32](ch.j, ch.addr+uintptr(*ch.j.jc.CodeHeapLog2SegmentSize))
}

func (hb HeapBlock) Used() (bool, error) {
	v, err := ReadScalar[uint8](hb.j, hb.addr+uintptr(*hb.j.jc.HeapBlockHeader+*hb.j.jc.HeapBlockHeaderUsed))
	return (v != 0), err
}

func (hb HeapBlock) Length() (uint32, error) {
	return ReadScalar[uint32](hb.j, hb.addr+uintptr(*hb.j.jc.HeapBlockHeader+*hb.j.jc.HeapBlockHeaderLength))
}

func (hb HeapBlock) AllocatedSpace() CodeBlob {
	return CodeBlob{addr: hb.addr + uintptr(*hb.j.jc.HeapBlockAllocatedSpace), j: hb.j}
}

func (cb CodeBlob) Kind() (uint8, error) {
	return ReadScalar[uint8](cb.j, cb.addr+uintptr(*cb.j.jc.CodeBlobKind))
}

func (cb CodeBlob) Name() (Zstring, error) {
	return ReadObjPtr[Zstring](cb.j, cb.addr+uintptr(*cb.j.jc.CodeBlobName))
}

func (cb CodeBlob) FrameSize() (int32, error) {
	return ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.CodeBlobFrameSize))
}

func (cb CodeBlob) NmethodMethod() (Method, error) {
	return ReadObjPtr[Method](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodMethod))
}

func (m Method) ConstMethod() (ConstMethod, error) {
	return ReadObjPtr[ConstMethod](m.j, m.addr+uintptr(*m.j.jc.MethodConstMethod))
}

func (cm ConstMethod) Consts() (ConstantPool, error) {
	return ReadObjPtr[ConstantPool](cm.j, cm.addr+uintptr(*cm.j.jc.ConstMethodConstants))
}

func (cm ConstMethod) NameIndex() (uint16, error) {
	return ReadScalar[uint16](cm.j, cm.addr+uintptr(*cm.j.jc.ConstMethodNameIndex))
}

func (cp ConstantPool) At(index uint16) (Symbol, error) {
	return ReadObjPtr[Symbol](cp.j, cp.addr+uintptr(*cp.j.jc.ConstantPoolBaseOffset)+uintptr(index)*8)
}

func (cp ConstantPool) PoolHolder() (Klass, error) {
	return ReadObjPtr[Klass](cp.j, cp.addr+uintptr(*cp.j.jc.ConstantPoolPoolHolder))
}

func (k Klass) Name() (Symbol, error) {
	return ReadObjPtr[Symbol](k.j, k.addr+uintptr(*k.j.jc.KlassName))
}

func (s Symbol) Length() (uint16, error) {
	return ReadScalar[uint16](s.j, s.addr+uintptr(*s.j.jc.SymbolLength))
}

func (s Symbol) Body() uintptr {
	return s.addr + uintptr(*s.j.jc.SymbolBody)
}

func (sq StubQueue) CodeBegin() (uint64, error) {
	return ReadScalar[uint64](sq.j, sq.addr+uintptr(*sq.j.jc.StubQueueStubBuffer))
}

func (sq StubQueue) CodeSize() (uint64, error) {
	return ReadScalar[uint64](sq.j, sq.addr+uintptr(*sq.j.jc.StubQueueBufferLimit))
}

func (a CodeHeapArray) Length() (int32, error) {
	return ReadScalar[int32](a.j, a.addr+uintptr(*a.j.jc.GrowableArrayLength))
}

func (a CodeHeapArray) Data() (uint64, error) {
	return ReadScalar[uint64](a.j, a.addr+uintptr(*a.j.jc.GrowableArrayData))
}

func (vs VirtualSpace) Low() (uint64, error) {
	return ReadScalar[uint64](vs.j, vs.addr+uintptr(*vs.j.jc.VirtualSpaceLow))
}

func (j *JVM) AbstractInterpreter_code(mappingAddr uintptr) (StubQueue, error) {
	return ReadObjPtr[StubQueue](j, mappingAddr+uintptr(*j.jc.AbstractInterpreterCode))
}

func (j *JVM) CodeCache_heaps(mappingAddr uintptr) (CodeHeapArray, error) {
	return ReadObjPtr[CodeHeapArray](j, mappingAddr+uintptr(*j.jc.CodeCacheHeaps))
}

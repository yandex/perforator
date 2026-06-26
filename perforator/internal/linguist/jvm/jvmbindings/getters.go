package jvmbindings

import (
	"encoding/binary"
	"fmt"
)

func (ch CodeHeap) NextSegment() (uint64, error) {
	return ReadScalar[uint64](ch.j, ch.addr+uintptr(*ch.j.jc.CodeHeapNextSegment))
}

func (ch CodeHeap) Memory() VirtualSpace {
	return VirtualSpace{
		addr: ch.addr + uintptr(*ch.j.jc.CodeHeapMemory),
		j:    ch.j,
	}
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

func (cb CodeBlob) CodeBegin() (uintptr, error) {
	return ReadScalar[uintptr](cb.j, cb.addr+uintptr(*cb.j.jc.CodeBlobCodeBegin))
}

func (cb CodeBlob) CodeOffset() (int32, error) {
	return ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.CodeBlobCodeOffset))
}

func (cb CodeBlob) NmethodMethod() (Method, error) {
	return ReadObjPtr[Method](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodMethod))
}

type PCDesc struct {
	PCOffset          int32
	ScopeDecodeOffset int32
}

type NmethodInfo struct {
	PCDs       []PCDesc
	ScopesData []byte
	Metadata   []uint64
}

func (cb CodeBlob) nmethodInfo_lte21() (NmethodInfo, error) {
	descsOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodScopesPcs))
	if err != nil {
		return NmethodInfo{}, err
	}
	scopesDataAddr, err := ReadScalar[uintptr](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodScopesDataAddr))
	if err != nil {
		return NmethodInfo{}, err
	}
	dependenciesOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodDependenciesOffset))
	if err != nil {
		return NmethodInfo{}, err
	}
	pcDescsSizeBytes := dependenciesOffset - descsOffset
	if pcDescsSizeBytes < 0 {
		return NmethodInfo{}, fmt.Errorf("unexpected negative pc_descs size")
	}
	if int64(pcDescsSizeBytes)%*cb.j.jc.PcDescSize != 0 {
		return NmethodInfo{}, fmt.Errorf("unexpected pc_desc_size: not divided by %d", *cb.j.jc.PcDescSize)
	}
	scopesData, err := cb.j.ReadBytes(scopesDataAddr, int(uintptr(descsOffset)+cb.addr-scopesDataAddr))
	if err != nil {
		return NmethodInfo{}, fmt.Errorf("failed to read scopes data: %w", err)
	}

	metadataOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodMetadataOffset))

	if err != nil {
		return NmethodInfo{}, err
	}

	metadataBegin := cb.addr + uintptr(metadataOffset)
	metadataEnd := scopesDataAddr

	if metadataEnd < metadataBegin {
		return NmethodInfo{}, fmt.Errorf("unexpected negative metadata size")
	}
	if (metadataEnd-metadataBegin)%8 != 0 {
		return NmethodInfo{}, fmt.Errorf("unexpected metadata size: not divided by 8")
	}
	metadataBytes, err := cb.j.ReadBytes(metadataBegin, int(metadataEnd-metadataBegin))
	if err != nil {
		return NmethodInfo{}, fmt.Errorf("failed to read metadata: %w", err)
	}
	metadata := make([]uint64, len(metadataBytes)/8)
	for i := 0; i < len(metadataBytes)/8; i++ {
		metadata[i] = binary.NativeEndian.Uint64(metadataBytes[i*8 : i*8+8])
	}

	pcDescsBytes, err := cb.j.ReadBytes(cb.addr+uintptr(descsOffset), int(pcDescsSizeBytes))
	if err != nil {
		return NmethodInfo{}, fmt.Errorf("failed to read pc_descs: %w", err)
	}
	pcs := make([]PCDesc, int64(len(pcDescsBytes)) / *cb.j.jc.PcDescSize)
	for i := range pcs {
		baseOffset := i * int(*cb.j.jc.PcDescSize)

		pcOffsetOffset := baseOffset + int(*cb.j.jc.PcDescPcOffset)

		sdoOffset := baseOffset + int(*cb.j.jc.PcDescScopeDecodeOffset)

		pcs[i] = PCDesc{
			PCOffset:          int32(binary.NativeEndian.Uint32(pcDescsBytes[pcOffsetOffset : pcOffsetOffset+4])),
			ScopeDecodeOffset: int32(binary.NativeEndian.Uint32(pcDescsBytes[sdoOffset : sdoOffset+4])),
		}
	}

	return NmethodInfo{
		PCDs:       pcs,
		ScopesData: scopesData,
		Metadata:   metadata,
	}, nil
}

func (cb CodeBlob) nmethodInfo_gte22() (NmethodInfo, error) {
	immData, err := ReadScalar[uintptr](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodImmutableData))
	if err != nil {
		return NmethodInfo{}, err
	}
	scopesOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodScopesPcs))
	if err != nil {
		return NmethodInfo{}, err
	}
	dataOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodScopesDataOffset))
	if err != nil {
		return NmethodInfo{}, err
	}
	speculationsOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodSpeculations))
	if err != nil {
		return NmethodInfo{}, err
	}
	pcDescsSizeBytes := dataOffset - scopesOffset
	if pcDescsSizeBytes < 0 {
		return NmethodInfo{}, fmt.Errorf("unexpected negative pc_descs size")
	}
	if int64(pcDescsSizeBytes)%*cb.j.jc.PcDescSize != 0 {
		return NmethodInfo{}, fmt.Errorf("unexpected pc_desc_size: not divided by %d", *cb.j.jc.PcDescSize)
	}
	scopesData, err := cb.j.ReadBytes(immData+uintptr(dataOffset), int(speculationsOffset-dataOffset))
	if err != nil {
		return NmethodInfo{}, fmt.Errorf("failed to read scopes data: %w", err)
	}

	var metadataBegin, metadataEnd uintptr

	if cb.j.Version() >= 25 {

		mutData, err := ReadScalar[uintptr](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodMutableData))
		if err != nil {
			return NmethodInfo{}, err
		}
		mutDataSize, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodMutableDataSize))
		if err != nil {
			return NmethodInfo{}, err
		}
		relocationSize, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodRelocationSize))
		if err != nil {
			return NmethodInfo{}, err
		}
		metadataBegin = mutData + uintptr(relocationSize)
		metadataEnd = mutData + uintptr(mutDataSize)
	} else {
		dataOffset, err := ReadScalar[int32](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodDataOffset))
		if err != nil {
			return NmethodInfo{}, err
		}
		dataBegin := cb.addr + uintptr(dataOffset)
		metadataOffset, err := ReadScalar[uint16](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodMetadataOffset))
		if err != nil {
			return NmethodInfo{}, err
		}
		jvmciDataOffset, err := ReadScalar[uint16](cb.j, cb.addr+uintptr(*cb.j.jc.NmethodJvmciDataOffset))
		if err != nil {
			return NmethodInfo{}, err
		}
		metadataBegin = dataBegin + uintptr(metadataOffset)
		metadataEnd = dataBegin + uintptr(jvmciDataOffset)
	}
	if metadataEnd < metadataBegin {
		return NmethodInfo{}, fmt.Errorf("unexpected negative metadata size")
	}
	if (metadataEnd-metadataBegin)%8 != 0 {
		return NmethodInfo{}, fmt.Errorf("unexpected metadata size: not divided by 8")
	}
	metadataBytes, err := cb.j.ReadBytes(metadataBegin, int(metadataEnd-metadataBegin))
	if err != nil {
		return NmethodInfo{}, fmt.Errorf("failed to read metadata: %w", err)
	}
	metadata := make([]uint64, len(metadataBytes)/8)
	for i := 0; i < len(metadataBytes)/8; i++ {
		metadata[i] = binary.NativeEndian.Uint64(metadataBytes[i*8 : i*8+8])
	}

	pcDescsBytes, err := cb.j.ReadBytes(immData+uintptr(scopesOffset), int(pcDescsSizeBytes))
	if err != nil {
		return NmethodInfo{}, fmt.Errorf("failed to read pc_descs: %w", err)
	}
	pcs := make([]PCDesc, int64(len(pcDescsBytes)) / *cb.j.jc.PcDescSize)
	for i := range pcs {
		baseOffset := i * int(*cb.j.jc.PcDescSize)

		pcOffsetOffset := baseOffset + int(*cb.j.jc.PcDescPcOffset)

		sdoOffset := baseOffset + int(*cb.j.jc.PcDescScopeDecodeOffset)

		pcs[i] = PCDesc{
			PCOffset:          int32(binary.NativeEndian.Uint32(pcDescsBytes[pcOffsetOffset : pcOffsetOffset+4])),
			ScopeDecodeOffset: int32(binary.NativeEndian.Uint32(pcDescsBytes[sdoOffset : sdoOffset+4])),
		}
	}

	return NmethodInfo{
		PCDs:       pcs,
		ScopesData: scopesData,
		Metadata:   metadata,
	}, nil
}

func (cb CodeBlob) NmethodInfo() (NmethodInfo, error) {
	if cb.j.Version() >= 22 {
		return cb.nmethodInfo_gte22()
	} else {
		return cb.nmethodInfo_lte21()
	}
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

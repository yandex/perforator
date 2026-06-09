package jvmscanner

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	jvmproto "github.com/yandex/perforator/perforator/agent/preprocessing/proto/jvm"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmbindings"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/jvmsupp"
)

const methodNameLimit = 128

type Config struct {
	EnableLineInfoParsing bool
}

type Scanner struct {
	bpf *programstate.State
	l   xlog.Logger

	conf Config
}

func New(l xlog.Logger, bpf *programstate.State, conf Config) *Scanner {
	return &Scanner{
		l:   l,
		bpf: bpf,

		conf: conf,
	}
}

type heapInfo struct {
	segmentSizeLog2 uint8
	begin           uintptr
}

type CodeHeap struct {
	begin           uintptr
	segmentSizeLog2 uint8
	obj             jvmbindings.CodeHeap
}

type ProcessState struct {
	pid     linux.CurrentNamespacePID
	heaps   []CodeHeap
	conf    *jvmproto.Cheatsheet
	version uint32
}

type ProcessInfoForBPF struct {
	InterpreterBegin uint64
	InterpreterEnd   uint64
}

func (s *Scanner) initInterp(j *jvmbindings.JVM, baseAddr uint64) (ProcessInfoForBPF, error) {
	var info ProcessInfoForBPF
	stubQueue, err := j.AbstractInterpreter_code(uintptr(baseAddr))
	if err != nil {
		return info, fmt.Errorf("failed to read AbstractInterpreter::_code: %w", err)
	}
	info.InterpreterBegin, err = stubQueue.CodeBegin()
	if err != nil {
		return info, fmt.Errorf("failed to read interpreter begin: %w", err)
	}
	size, err := stubQueue.CodeSize()
	if err != nil {
		return info, fmt.Errorf("failed to read interpreter size: %w", err)
	}
	info.InterpreterEnd = info.InterpreterBegin + size

	return info, nil
}

const maxCodeCacheHeapsPerProcess = 10
const uintptrSz = unsafe.Sizeof(*new(uintptr))

func (s *Scanner) initJIT(j *jvmbindings.JVM, baseAddr uint64, state *ProcessState) error {
	codeHeaps, err := j.CodeCache_heaps(uintptr(baseAddr))
	if err != nil {
		return fmt.Errorf("failed to read CodeCache::_heaps: %w", err)
	}
	heapCount, err := codeHeaps.Length()
	if err != nil {
		return fmt.Errorf("failed to read heaps array length: %w", err)
	}
	if heapCount < 0 {
		return fmt.Errorf("heaps array length is negative")
	}
	if heapCount > maxCodeCacheHeapsPerProcess {
		return fmt.Errorf("heaps array is too large")
	}
	heapPointers, err := codeHeaps.Data()
	if err != nil {
		return fmt.Errorf("failed to read heaps array data: %w", err)
	}
	for i := range heapCount {
		heap, err := jvmbindings.ReadObjPtr[jvmbindings.CodeHeap](j, uintptr(heapPointers)+uintptr(i)*uintptrSz)
		if err != nil {
			return fmt.Errorf("failed to read heap pointer: %w", err)
		}
		heapMemory := heap.Memory()
		low, err := heapMemory.Low()
		if err != nil {
			return fmt.Errorf("failed to read heap begin: %w", err)
		}
		log2SegmentSize, err := heap.Log2SegmentSize()
		if err != nil {
			return fmt.Errorf("failed to read heap segment size: %w", err)
		}
		if log2SegmentSize < 0 {
			return fmt.Errorf("heap segment size is negative")
		}
		if log2SegmentSize > 29 {
			return fmt.Errorf("heap segment size is too large")
		}
		state.heaps = append(state.heaps, CodeHeap{
			begin:           uintptr(low),
			segmentSizeLog2: uint8(log2SegmentSize),
			obj:             heap,
		})
	}

	return nil
}

func (s *Scanner) InitProcess(pid linux.CurrentNamespacePID, cs *jvmproto.Cheatsheet, version uint32, baseAddress uint64) (*ProcessState, ProcessInfoForBPF, error) {
	err := validateProto(cs, version)
	if err != nil {
		return nil, ProcessInfoForBPF{}, fmt.Errorf("invalid config: %w", err)
	}
	j := jvmbindings.NewJVM(cs, uint32(pid), version)
	infoForBPF, err := s.initInterp(j, baseAddress)
	if err != nil {
		return nil, ProcessInfoForBPF{}, fmt.Errorf("failed to discover interpreter state: %w", err)
	}
	var procState ProcessState
	procState.pid = pid
	procState.conf = cs
	procState.version = version
	err = s.initJIT(j, baseAddress, &procState)
	if err != nil {
		return nil, ProcessInfoForBPF{}, fmt.Errorf("failed to discover JIT state: %w", err)
	}
	return &procState, infoForBPF, err
}

func (s *Scanner) prepare(ps *ProcessState) (*jvmbindings.JVM, error) {
	proc, err := s.bpf.GetProcess(ps.pid)
	if err != nil {
		return nil, &ProcessNotFoundError{fmt.Errorf("process %d not found: %w", ps.pid, err)}
	}
	if proc.LibjvmBinary.Id == ^unwinder.BinaryId(0) {
		// Justification for ProcessNotFoundError tag: it may be possible
		// that process already died, but its pid was reused by a non-java process
		return nil, &ProcessNotFoundError{fmt.Errorf("process has no libjvm mapping")}
	}

	return jvmbindings.NewJVM(ps.conf, uint32(ps.pid), ps.version), nil
}

func (s *Scanner) Symbolize(ctx context.Context, ps *ProcessState, addr uint64) (string, error) {
	jvm, err := s.prepare(ps)
	if err != nil {
		return "", err
	}
	buf, err := readMethodName(jvmbindings.Method(jvm.MakeObjPointer(uintptr(addr))), 128)
	return string(buf), err
}

// scanStep parses one block sequence.
// Return values:
// * Method info (optional)
// * Whether method info is present (i.e. block sequence is used)
// * Block sequence length
// * Error
func (s *Scanner) scanStep(
	ctx context.Context,
	jvm *jvmbindings.JVM,
	curHeap *heapInfo,
	blockIndex uint32,
) (*jvmsupp.MethodInfo, bool, uint32, error) {
	heapBlock := jvmbindings.HeapBlock(jvm.MakeObjPointer(uintptr(curHeap.begin) + (uintptr(blockIndex) << curHeap.segmentSizeLog2)))
	s.l.Trace(ctx, "Processing heap block", log.UInt32("block", blockIndex))
	isUsed, err := heapBlock.Used()
	if err != nil {
		return nil, false, 0, fmt.Errorf("failed to check if block %d is used: %w", blockIndex, err)
	}
	segmentLength, err := heapBlock.Length()
	if err != nil {
		return nil, false, 0, fmt.Errorf("failed to get segment length of segment headed by block %d: %w", blockIndex, err)
	}
	if segmentLength == 0 {
		return nil, false, 0, fmt.Errorf("unexpected length of zero")
	}
	if !isUsed {
		s.l.Trace(ctx, "This block heads free segment, skipping it", log.UInt32("segment_length", segmentLength))
		return nil, false, segmentLength, nil
	}
	s.l.Trace(ctx, "This block heads used segment, scanning it", log.UInt32("segment_length", segmentLength))
	codeBlob := heapBlock.AllocatedSpace()
	var blobName []byte
	var isJIT bool
	if jvm.Version() >= 22 {
		kind, err := codeBlob.Kind()
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to get kind of code blob: %w", err)
		}
		isJIT = (int64(kind) == *jvm.Cheatsheet().CodeBlobKindNmethod)
	} else {
		namePtr, err := codeBlob.Name()
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to get name pointer from code blob %d: %w", blockIndex, err)
		}
		blobName, err = jvm.ReadString(namePtr, methodNameLimit)
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to read code blob name: %w", err)
		}
		name := string(blobName)
		isJIT = (name == "native nmethod" || name == "nmethod")
	}
	var name []byte
	var frameSize int32 = -1
	var symtab *jvmsupp.MethodSymbolizationTable
	if isJIT {
		s.l.Debug(ctx, "Code blob classified as nmethod")
		isJIT = true
		frameSize, err = codeBlob.FrameSize()
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to read nmethod frame size: %w", err)
		}

		if frameSize < 0 {
			return nil, false, 0, fmt.Errorf("unexpected negative frame size for code blob %d: %d", blockIndex, frameSize)
		}

		method, err := codeBlob.NmethodMethod()
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to get method of nmethod %d: %w", blockIndex, err)
		}
		name, err = readMethodName(method, 128)
		if err != nil {
			return nil, false, 0, fmt.Errorf("failed to read method %d name: %w", blockIndex, err)
		}

		if s.conf.EnableLineInfoParsing {
			symtab, err = s.parseInstructionInfo(ctx, jvm, codeBlob)
			if err != nil {
				// TODO: rather than logging, we should send this error as RPC response back to agent.
				s.l.Warn(ctx, "Failed to parse instruction info for nmethod", log.String("name", string(name)), log.Error(err))
				symtab = nil
			}
		}

		s.l.Debug(ctx, "Parsed nmethod", log.String("name", string(name)), log.Int32("frame_size", frameSize))
	} else {
		s.l.Debug(ctx, "Code blob classified as non-nmethod")
		// don't read blob name second time for old JDKs
		if blobName == nil {
			namePtr, err := codeBlob.Name()
			if err != nil {
				return nil, false, 0, fmt.Errorf("failed to get name pointer from code blob %d: %w", blockIndex, err)
			}
			blobName, err = jvm.ReadString(namePtr, 128)
			if err != nil {
				return nil, false, 0, fmt.Errorf("failed to read code blob name: %w", err)
			}
		}
		name = blobName
		s.l.Debug(ctx, "Parsed non-nmethod", log.String("name", string(name)))
	}
	return &jvmsupp.MethodInfo{
		Name:               string(name),
		FrameSizeBytes:     int64(frameSize) * 8,
		IsJit:              isJIT,
		CodeBegin:          uint64(jvmbindings.ObjectPtr(codeBlob).Addr()),
		CodeEnd:            uint64(jvmbindings.ObjectPtr(heapBlock).Addr()) + uint64(segmentLength)<<uint64(curHeap.segmentSizeLog2),
		SymbolizationTable: symtab,
	}, true, segmentLength, nil
}

// Error that may be caused by process being dead
type ProcessNotFoundError struct {
	inner error
}

func (e *ProcessNotFoundError) Error() string {
	return e.inner.Error()
}

func (e *ProcessNotFoundError) Unwrap() error {
	return e.inner
}

func (s *Scanner) ScanProcess(ctx context.Context, state *ProcessState) ([]*jvmsupp.MethodInfo, error) {
	jvm, err := s.prepare(state)
	if err != nil {
		return nil, err
	}
	var detectedMethods []*jvmsupp.MethodInfo
	for heapIdx, heap := range state.heaps {
		blockCount, err := heap.obj.NextSegment()
		if err != nil {
			return nil, fmt.Errorf("failed to get current code heap size: %w", err)
		}
		s.l.Debug(ctx, "Scanning code cache heap", log.Int("heap", heapIdx), log.UInt64("block_count", blockCount))
		var blockIndex uint64
		curHeap := heapInfo{
			begin:           uintptr(heap.begin),
			segmentSizeLog2: uint8(heap.segmentSizeLog2),
		}
		for blockIndex < blockCount {
			minfo, ok, len, err := s.scanStep(ctx, jvm, &curHeap, uint32(blockIndex))
			if err != nil {
				return nil, fmt.Errorf("failed to execute scan step: %w", err)
			}
			if len == 0 {
				return nil, fmt.Errorf("internal error: scan step resulted in zero-length block sequence")
			}
			blockIndex += uint64(len)
			if ok {
				detectedMethods = append(detectedMethods, minfo)
			}
		}
		s.l.Debug(ctx, "Heap scan complete", log.Int("heap", heapIdx))
	}
	s.l.Info(ctx, "Scan complete", log.Int("methods", len(detectedMethods)))
	return detectedMethods, nil
}

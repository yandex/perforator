package runtimewriter

// #include <perforator/agent/collector/nativepipeline/runtime.h>
// #include <stdlib.h>
import "C"

import (
	"fmt"
	"unsafe"
)

// Frame borrows its owner's arena until Finish succeeds or Close is called.
// Offsets keep it valid when the arena grows.
type Frame struct {
	arena      *Writer
	frameIndex int
}

const maxRuntimeStringBytes = 16 * 1024 * 1024

// Writer owns a C arena. Records contain offsets, so realloc cannot invalidate stored references.
// No pointer into the allocation may be retained across allocate or putString.
type Writer struct {
	result      C.TPerforatorRuntimeResult
	capacity    int
	stringBytes int
	nextFrame   int
	inlineIndex int
}

// New allocates a C arena for the complete frame sequence and language ranges.
func New(totalFrameCount, nativePrefixSize, languageCount int) (*Writer, error) {
	if nativePrefixSize < 0 || nativePrefixSize > totalFrameCount {
		return nil, fmt.Errorf("invalid native frame count")
	}
	writer := &Writer{}
	frames, err := writer.reserveRecords(totalFrameCount, C.sizeof_TPerforatorResolvedFrame, int(unsafe.Alignof(C.TPerforatorResolvedFrame{})))
	if err != nil {
		writer.Close()
		return nil, err
	}
	languages, err := writer.reserveRecords(languageCount, C.sizeof_TPerforatorResolvedLanguageSection, int(unsafe.Alignof(C.TPerforatorResolvedLanguageSection{})))
	if err != nil {
		writer.Close()
		return nil, err
	}
	writer.result.FramesOffset = C.size_t(frames)
	writer.result.FrameCount = C.size_t(totalFrameCount)
	writer.result.NativePrefixSize = C.size_t(nativePrefixSize)
	writer.result.LanguagesOffset = C.size_t(languages)
	writer.result.LanguageCount = C.size_t(languageCount)
	return writer, nil
}

// firstFrameIndex is absolute in the output frame array, including the native
// prefix. Native annotations have empty ranges.
func (a *Writer) SetLanguageRange(sectionIndex int, language uint8, firstFrameIndex, frameCount int) error {
	if sectionIndex < 0 || sectionIndex >= int(a.result.LanguageCount) || language >= C.PERFORATOR_LANGUAGE_COUNT ||
		firstFrameIndex < 0 || firstFrameIndex > int(a.result.FrameCount) || frameCount < 0 || frameCount > int(a.result.FrameCount)-firstFrameIndex {
		return fmt.Errorf("invalid runtime language range")
	}
	*a.language(sectionIndex) = C.TPerforatorResolvedLanguageSection{
		Language: C.EPerforatorLanguage(language), FrameBegin: C.size_t(firstFrameIndex), FrameCount: C.size_t(frameCount),
	}
	return nil
}

func (a *Writer) Close() {
	C.free(unsafe.Pointer(a.result.Arena))
	a.result.Arena = nil
}

func (a *Writer) reserveRecords(count, size, alignment int) (int, error) {
	if count < 0 || count > int(^uint(0)>>1)/size {
		return 0, fmt.Errorf("runtime record size overflow")
	}
	return a.allocate(count*size, alignment)
}

func (a *Writer) allocate(size, alignment int) (int, error) {
	maxInt := int(^uint(0) >> 1)
	used := int(a.result.ArenaBytes)
	padding := (-used) & (alignment - 1)
	if size < 0 || padding > maxInt-used || size > maxInt-used-padding {
		return 0, fmt.Errorf("runtime arena size overflow")
	}
	offset, end := used+padding, used+padding+size
	if end > a.capacity {
		capacity := a.capacity
		if capacity < 4096 {
			capacity = 4096
		}
		for capacity < end {
			if capacity > maxInt/2 {
				capacity = end
				break
			}
			capacity *= 2
		}
		data := C.realloc(unsafe.Pointer(a.result.Arena), C.size_t(capacity))
		if data == nil {
			return 0, fmt.Errorf("cannot allocate %d bytes for runtime arena", capacity)
		}
		a.result.Arena = (*C.char)(data)
		a.capacity = capacity
	}
	a.result.ArenaBytes = C.size_t(end)
	return offset, nil
}

func (a *Writer) putString(value string) (C.TPerforatorRuntimeString, error) {
	if len(value) == 0 {
		return C.TPerforatorRuntimeString{}, nil
	}
	if len(value) > maxRuntimeStringBytes-a.stringBytes {
		return C.TPerforatorRuntimeString{}, fmt.Errorf("runtime resolver requested an excessive arena")
	}
	offset, err := a.allocate(len(value), 1)
	if err != nil {
		return C.TPerforatorRuntimeString{}, err
	}
	copy(unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(a.result.Arena), offset)), len(value)), value)
	a.stringBytes += len(value)
	return C.TPerforatorRuntimeString{Offset: C.size_t(offset), Size: C.size_t(len(value))}, nil
}

func (a *Writer) frame(index int) *C.TPerforatorResolvedFrame {
	return (*C.TPerforatorResolvedFrame)(unsafe.Add(unsafe.Pointer(a.result.Arena), int(a.result.FramesOffset)+index*C.sizeof_TPerforatorResolvedFrame))
}

func (a *Writer) language(index int) *C.TPerforatorResolvedLanguageSection {
	return (*C.TPerforatorResolvedLanguageSection)(unsafe.Add(unsafe.Pointer(a.result.Arena), int(a.result.LanguagesOffset)+index*C.sizeof_TPerforatorResolvedLanguageSection))
}

// AddFrame appends frames in native-stack order, then in section/frame order.
// inlineDepth reserves the contiguous inline chain without a temporary slice.
// Complete its functions before adding another frame; empty chains are allowed.
func (a *Writer) AddFrame(address uint64, inlineDepth int) (Frame, error) {
	if a.nextFrame >= int(a.result.FrameCount) || inlineDepth < 0 || uint64(inlineDepth) > uint64(^uint32(0)) {
		return Frame{}, fmt.Errorf("invalid runtime frame count or inline depth")
	}
	if err := a.finishFrame(); err != nil {
		return Frame{}, err
	}
	symbols, err := a.reserveRecords(inlineDepth, C.sizeof_TPerforatorResolvedSymbol, int(unsafe.Alignof(C.TPerforatorResolvedSymbol{})))
	if err != nil {
		return Frame{}, err
	}
	frameIndex := a.nextFrame
	*a.frame(frameIndex) = C.TPerforatorResolvedFrame{
		Address: C.uint64_t(address), SymbolsOffset: C.size_t(symbols), SymbolCount: C.uint32_t(inlineDepth),
	}
	a.nextFrame++
	a.inlineIndex = 0
	return Frame{arena: a, frameIndex: frameIndex}, nil
}

func (w Frame) SetMappingName(name string) error {
	mapping, err := w.arena.putString(name)
	if err != nil {
		return err
	}
	w.arena.frame(w.frameIndex).Mapping = mapping
	return nil
}

func (w Frame) SetAddress(address uint64) {
	w.arena.frame(w.frameIndex).Address = C.uint64_t(address)
}

func normalizedLine(line int32) C.uint32_t {
	if line > 0 {
		return C.uint32_t(line)
	}
	return 0
}

func (w Frame) AddInlineFunction(name, filename string, startLine, line int32) error {
	a := w.arena
	frame := a.frame(w.frameIndex)
	if w.frameIndex != a.nextFrame-1 || a.inlineIndex >= int(frame.SymbolCount) {
		return fmt.Errorf("invalid inline function order or count")
	}
	symbolOffset := int(frame.SymbolsOffset) + a.inlineIndex*C.sizeof_TPerforatorResolvedSymbol
	function, err := a.putString(name)
	if err != nil {
		return err
	}
	file, err := a.putString(filename)
	if err != nil {
		return err
	}
	*(*C.TPerforatorResolvedSymbol)(unsafe.Add(unsafe.Pointer(a.result.Arena), symbolOffset)) = C.TPerforatorResolvedSymbol{
		Name: function, File: file, StartLine: normalizedLine(startLine), Line: normalizedLine(line),
	}
	a.inlineIndex++
	return nil
}

func (a *Writer) finishFrame() error {
	if a.nextFrame != 0 && a.inlineIndex != int(a.frame(a.nextFrame-1).SymbolCount) {
		return fmt.Errorf("runtime returned an incomplete inline chain")
	}
	return nil
}

// Finish validates the result and transfers arena ownership on success. The
// caller frees it with free. On error, the Writer retains ownership for Close.
func (a *Writer) Finish() (C.TPerforatorRuntimeResult, error) {
	if a.nextFrame != int(a.result.FrameCount) {
		return C.TPerforatorRuntimeResult{}, fmt.Errorf("runtime returned an invalid frame count")
	}
	if err := a.finishFrame(); err != nil {
		return C.TPerforatorRuntimeResult{}, err
	}
	result := a.result
	a.result = C.TPerforatorRuntimeResult{}
	return result, nil
}

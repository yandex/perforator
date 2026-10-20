package nativepipeline

// #include <perforator/agent/collector/nativepipeline/runtime.h>
import "C"

import (
	"context"
	"fmt"
	"runtime/cgo"
	"unsafe"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

const (
	JITRuntimeNone JITRuntimeID = JITRuntimeID(C.PerforatorJitRuntimeNone)
	JITRuntimeJVM  JITRuntimeID = JITRuntimeID(C.PerforatorJitRuntimeJvm)
	LanguagePython Language     = Language(C.PERFORATOR_LANGUAGE_PYTHON)
	LanguagePHP    Language     = Language(C.PERFORATOR_LANGUAGE_PHP)
	LanguageLua    Language     = Language(C.PERFORATOR_LANGUAGE_LUA)
	LanguageCount               = int(C.PERFORATOR_LANGUAGE_COUNT)
)

// runtimeCallbacks belongs to one serialized native consumer. Its owner keeps
// the cgo handle live until all callbacks finish. The context applies to the batch.
type runtimeCallbacks struct {
	ctx      context.Context
	resolver RuntimeResolver
}

type interpreterResolution struct {
	frames      []ResolvedInterpreterFrame
	stringBytes int
}

type nativeResolution struct {
	frames      []ResolvedNativeFrame
	stringBytes int
	symbolCount int
}

func processKeyFromC(pid C.uint32_t, startTime C.uint64_t) linux.ProcessKey {
	return linux.ProcessKey{
		Pid:              linux.CurrentNamespacePID(pid),
		ProcessStartTime: uint64(startTime),
	}
}

func retainRuntimeResolution[T any](resolution *C.uint64_t, resolved T) C.int {
	if *resolution == 0 {
		*resolution = C.uint64_t(cgo.NewHandle(resolved))
	}
	return C.PerforatorResolveInsufficientCapacity
}

func finishRuntimeResolution(resolution *C.uint64_t, errorMessage **C.char, err error) C.int {
	if resolution != nil {
		PerforatorGoReleaseRuntimeResolution(*resolution)
		*resolution = 0
	}
	if err == nil {
		return C.PerforatorResolveOk
	}
	if errorMessage != nil {
		*errorMessage = C.CString(err.Error())
	}
	return C.PerforatorResolveFailed
}

//export PerforatorGoReleaseRuntimeResolution
func PerforatorGoReleaseRuntimeResolution(resolution C.uint64_t) {
	if resolution != 0 {
		cgo.Handle(resolution).Delete()
	}
}

//export PerforatorGoResolveInterpreterFrames
func PerforatorGoResolveInterpreterFrames(
	resolverID C.uint64_t,
	crequests *C.TPerforatorInterpreterFrameRequest,
	requestCount C.size_t,
	cresults *C.TPerforatorResolvedInterpreterFrame,
	arena *C.char,
	arenaCapacity C.size_t,
	requiredArenaBytes *C.size_t,
	resolution *C.uint64_t,
	errorMessage **C.char,
) C.int {
	if errorMessage != nil {
		*errorMessage = nil
	}
	if requiredArenaBytes == nil || resolution == nil || (requestCount != 0 && cresults == nil) {
		return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("invalid runtime result buffers"))
	}
	var resolved interpreterResolution
	if *resolution != 0 {
		resolved = cgo.Handle(*resolution).Value().(interpreterResolution)
	} else {
		if requestCount != 0 && crequests == nil {
			return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("missing runtime requests"))
		}
		callbacks := cgo.Handle(resolverID).Value().(*runtimeCallbacks)
		requests := unsafe.Slice(crequests, int(requestCount))
		goRequests := make([]InterpreterFrameRequest, len(requests))
		for i, request := range requests {
			if request.FrameData == nil || request.FrameSize == 0 {
				return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("invalid runtime frame payload at index %d", i))
			}
			goRequests[i] = InterpreterFrameRequest{
				Language: Language(request.Language),
				Key:      processKeyFromC(request.Pid, request.ProcessStartTime),
				Data:     unsafe.Slice((*byte)(request.FrameData), int(request.FrameSize)),
			}
		}
		frames, err := callbacks.resolver.ResolveInterpreterFrames(callbacks.ctx, goRequests)
		if err != nil {
			return finishRuntimeResolution(resolution, errorMessage, err)
		}
		resolved.frames = frames
		for _, frame := range frames {
			resolved.stringBytes += len(frame.Mapping) + len(frame.Name) + len(frame.Filename)
		}
	}
	if len(resolved.frames) != int(requestCount) {
		return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("runtime resolver returned %d results for %d requests", len(resolved.frames), requestCount))
	}
	*requiredArenaBytes = C.size_t(resolved.stringBytes)
	if *requiredArenaBytes > arenaCapacity {
		return retainRuntimeResolution(resolution, resolved)
	}

	results := unsafe.Slice(cresults, len(resolved.frames))
	strings := newStringArenaAt(unsafe.Pointer(arena), resolved.stringBytes)
	for i, frame := range resolved.frames {
		result := C.TPerforatorResolvedInterpreterFrame{
			Mapping:   strings.put(frame.Mapping),
			Name:      strings.put(frame.Name),
			File:      strings.put(frame.Filename),
			Address:   C.uint64_t(frame.Address),
			StartLine: C.int32_t(frame.StartLine),
			Line:      C.int32_t(frame.Line),
			LineInfo:  C.uint8_t(frame.LineInfo),
		}
		if frame.Unsymbolized {
			result.Flags |= C.uint32_t(C.PerforatorResolvedFrameUnsymbolized)
		}
		if frame.HasStartLine {
			result.Flags |= C.uint32_t(C.PerforatorResolvedFrameHasStartLine)
		}
		if frame.HasLine {
			result.Flags |= C.uint32_t(C.PerforatorResolvedFrameHasLine)
		}
		if frame.DoNotCache {
			result.Flags |= C.uint32_t(C.PerforatorResolvedFrameDoNotCache)
		}
		results[i] = result
	}
	strings.finish()
	return finishRuntimeResolution(resolution, errorMessage, nil)
}

//export PerforatorGoResolveNativeFrames
func PerforatorGoResolveNativeFrames(
	resolverID C.uint64_t,
	crequests *C.TPerforatorNativeFrameRequest,
	requestCount C.size_t,
	cresults *C.TPerforatorResolvedNativeFrame,
	csymbols *C.TPerforatorResolvedNativeSymbol,
	symbolCapacity C.size_t,
	symbolCount *C.size_t,
	arena *C.char,
	arenaCapacity C.size_t,
	requiredArenaBytes *C.size_t,
	resolution *C.uint64_t,
	errorMessage **C.char,
) C.int {
	if errorMessage != nil {
		*errorMessage = nil
	}
	if requiredArenaBytes == nil || symbolCount == nil || resolution == nil || (requestCount != 0 && cresults == nil) {
		return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("invalid runtime result buffers"))
	}
	var resolved nativeResolution
	if *resolution != 0 {
		resolved = cgo.Handle(*resolution).Value().(nativeResolution)
	} else {
		if requestCount != 0 && crequests == nil {
			return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("missing runtime requests"))
		}
		callbacks := cgo.Handle(resolverID).Value().(*runtimeCallbacks)
		requests := unsafe.Slice(crequests, int(requestCount))
		goRequests := make([]NativeFrameRequest, len(requests))
		for i, request := range requests {
			goRequests[i] = NativeFrameRequest{
				Key:              processKeyFromC(request.Key.Pid, request.Key.StartTime),
				Address:          uint64(request.Address),
				JITRuntimeID:     JITRuntimeID(request.JitRuntimeId),
				JITMethodAddress: uint64(request.JitMethodAddress),
			}
		}
		resolved.frames = callbacks.resolver.ResolveNativeFrames(callbacks.ctx, goRequests)
		for _, frame := range resolved.frames {
			resolved.stringBytes += len(frame.JITMapping) + len(frame.JVMName)
			resolved.symbolCount += len(frame.JITSymbols)
			for _, symbol := range frame.JITSymbols {
				resolved.stringBytes += len(symbol)
			}
		}
	}
	if len(resolved.frames) != int(requestCount) {
		return finishRuntimeResolution(resolution, errorMessage, fmt.Errorf("runtime resolver returned %d results for %d requests", len(resolved.frames), requestCount))
	}
	*requiredArenaBytes = C.size_t(resolved.stringBytes)
	*symbolCount = C.size_t(resolved.symbolCount)
	if *requiredArenaBytes > arenaCapacity || *symbolCount > symbolCapacity {
		return retainRuntimeResolution(resolution, resolved)
	}

	results := unsafe.Slice(cresults, len(resolved.frames))
	symbols := unsafe.Slice(csymbols, resolved.symbolCount)
	strings := newStringArenaAt(unsafe.Pointer(arena), resolved.stringBytes)
	symbolOffset := 0
	for i, frame := range resolved.frames {
		result := C.TPerforatorResolvedNativeFrame{
			JitMapping:  strings.put(frame.JITMapping),
			SymbolBegin: C.uint32_t(symbolOffset),
			SymbolCount: C.uint32_t(len(frame.JITSymbols)),
		}
		for _, symbol := range frame.JITSymbols {
			symbols[symbolOffset] = C.TPerforatorResolvedNativeSymbol{Name: strings.put(symbol)}
			symbolOffset++
		}
		result.JvmName = strings.put(frame.JVMName)
		if frame.ClearAddress {
			result.ClearAddress = 1
		}
		results[i] = result
	}
	strings.finish()
	return finishRuntimeResolution(resolution, errorMessage, nil)
}

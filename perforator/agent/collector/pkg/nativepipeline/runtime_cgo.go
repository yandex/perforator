package nativepipeline

// #include <perforator/agent/collector/nativepipeline/runtime.h>
// #include <stdlib.h>
import "C"

import (
	"context"
	"fmt"
	"runtime/cgo"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline/runtimewriter"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

type Language = C.EPerforatorLanguage

const (
	LanguagePython Language = C.PERFORATOR_LANGUAGE_PYTHON
	LanguagePHP    Language = C.PERFORATOR_LANGUAGE_PHP
	LanguageJVM    Language = C.PERFORATOR_LANGUAGE_JVM
	LanguageLua    Language = C.PERFORATOR_LANGUAGE_LUA
	LanguageNone   Language = C.PERFORATOR_LANGUAGE_COUNT
	LanguageCount           = int(C.PERFORATOR_LANGUAGE_COUNT)
)

// runtimeCallbacks belongs to one serialized native consumer. Its owner keeps
// the cgo handle live until all callbacks finish. The context belongs to the worker.
type runtimeCallbacks struct {
	ctx      context.Context
	resolver RuntimeResolver
}

//export PerforatorGoSignalName
func PerforatorGoSignalName(signal C.int32_t) *C.char {
	return C.CString(unix.SignalName(syscall.Signal(signal)))
}

//export PerforatorGoResolveRuntimeFrames
func PerforatorGoResolveRuntimeFrames(
	handle C.uint64_t,
	request C.TPerforatorRuntimeRequestPtr,
	output *C.TPerforatorRuntimeResult,
	errorMessage **C.char,
) C.int {
	if errorMessage != nil {
		*errorMessage = nil
	}
	fail := func(err error) C.int {
		if errorMessage != nil {
			*errorMessage = C.CString(err.Error())
		}
		return C.PerforatorResolveFailed
	}
	if output == nil {
		return fail(fmt.Errorf("missing runtime output"))
	}
	*output = C.TPerforatorRuntimeResult{}
	switch {
	case request == nil:
		return fail(fmt.Errorf("missing runtime request"))
	case request.NativeStackSize != 0 && request.NativeStack == nil:
		return fail(fmt.Errorf("missing native stack"))
	case request.LanguagesSize > C.size_t(LanguageCount):
		return fail(fmt.Errorf("too many runtime language sections: %d", request.LanguagesSize))
	case request.LanguagesSize != 0 && request.Languages == nil:
		return fail(fmt.Errorf("missing language sections"))
	}
	input := RuntimeRequest{
		Key: linux.ProcessKey{
			Pid:              linux.CurrentNamespacePID(request.Key.Pid),
			ProcessStartTime: uint64(request.Key.StartTime),
		},
		NativeStack: unsafe.Slice((*uint64)(unsafe.Pointer(request.NativeStack)), int(request.NativeStackSize)),
		Languages:   make([]LanguageSection, int(request.LanguagesSize)),
	}
	for i, section := range unsafe.Slice(request.Languages, int(request.LanguagesSize)) {
		if section.Data == nil || section.FrameSize == 0 || section.Size == 0 || section.Size%C.size_t(section.FrameSize) != 0 {
			return fail(fmt.Errorf("invalid language section %d", i))
		}
		input.Languages[i] = LanguageSection{
			Language: section.Language, FrameSize: uint16(section.FrameSize),
			Data: unsafe.Slice((*byte)(section.Data), int(section.Size)),
		}
	}
	writer, err := newRuntimeWriter(input)
	if err != nil {
		return fail(err)
	}
	defer writer.Close()
	callbacks := cgo.Handle(handle).Value().(*runtimeCallbacks)
	if err := callbacks.resolver.ResolveRuntimeFrames(callbacks.ctx, input, writer); err != nil {
		return fail(err)
	}
	result, err := writer.Finish()
	if err != nil {
		return fail(err)
	}
	*output = *(*C.TPerforatorRuntimeResult)(unsafe.Pointer(&result))
	return C.PerforatorResolveOk
}

func newRuntimeWriter(request RuntimeRequest) (*runtimewriter.Writer, error) {
	totalFrameCount := len(request.NativeStack)
	var seen [LanguageCount]bool
	for _, section := range request.Languages {
		if int(section.Language) >= LanguageCount || seen[section.Language] {
			return nil, fmt.Errorf("unknown or duplicate language %d", section.Language)
		}
		seen[section.Language] = true
		count := len(section.Data) / int(section.FrameSize)
		if count > int(^uint(0)>>1)-totalFrameCount {
			return nil, fmt.Errorf("runtime frame count overflow")
		}
		totalFrameCount += count
	}
	return runtimewriter.New(totalFrameCount, len(request.NativeStack))
}

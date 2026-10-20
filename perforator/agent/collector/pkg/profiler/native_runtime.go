package profiler

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profilerext"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type runtimeFrame interface {
	unwinder.PythonFrame | unwinder.PhpFrame | unwinder.LuaFrame
}

type interpreterFrameResolver[T runtimeFrame] interface {
	ResolveFrame(linux.ProcessKey, *T) models.ResolvedInterpreterFrame
}

type jvmMethodResolver interface {
	SymbolizeInterpretedForProcess(context.Context, linux.ProcessKey, uint64) (string, bool, error)
}

type nativeRuntimeResolver struct {
	python            interpreterFrameResolver[unwinder.PythonFrame]
	php               interpreterFrameResolver[unwinder.PhpFrame]
	lua               interpreterFrameResolver[unwinder.LuaFrame]
	jit               []profilerext.ProcessJITSymbolizer
	jvm               jvmMethodResolver
	log               xlog.Logger
	disableJVMMethods bool
}

var _ nativepipeline.RuntimeResolver = (*nativeRuntimeResolver)(nil)

func (r *nativeRuntimeResolver) ResolveInterpreterFrames(
	_ context.Context,
	requests []nativepipeline.InterpreterFrameRequest,
) ([]nativepipeline.ResolvedInterpreterFrame, error) {
	result := make([]nativepipeline.ResolvedInterpreterFrame, len(requests))
	for i, request := range requests {
		var err error
		switch request.Language {
		case nativepipeline.LanguagePython:
			result[i], err = resolveInterpreterFrame(r.python, request)
		case nativepipeline.LanguagePHP:
			result[i], err = resolveInterpreterFrame(r.php, request)
		case nativepipeline.LanguageLua:
			frame, decodeErr := decodeRuntimeFrame[unwinder.LuaFrame](request.Data)
			switch {
			case decodeErr != nil:
				err = decodeErr
			case frame.Type != unwinder.LuaFrameTypeLua && frame.Type != unwinder.LuaFrameTypeC && frame.Type != unwinder.LuaFrameTypeInvalid:
				err = fmt.Errorf("unsupported Lua frame type %d", frame.Type)
			case r.lua == nil:
				err = fmt.Errorf("Lua resolver is unavailable")
			default:
				result[i] = r.lua.ResolveFrame(request.Key, frame)
			}
		default:
			err = fmt.Errorf("unsupported interpreter language %d", request.Language)
		}
		if err != nil {
			return nil, fmt.Errorf("resolve runtime frame %d: %w", i, err)
		}
	}
	return result, nil
}

func resolveInterpreterFrame[T runtimeFrame](resolver interpreterFrameResolver[T], request nativepipeline.InterpreterFrameRequest) (models.ResolvedInterpreterFrame, error) {
	if resolver == nil {
		return models.ResolvedInterpreterFrame{}, fmt.Errorf("interpreter resolver is unavailable")
	}
	frame, err := decodeRuntimeFrame[T](request.Data)
	if err != nil {
		return models.ResolvedInterpreterFrame{}, err
	}
	return resolver.ResolveFrame(request.Key, frame), nil
}

// The decoded frame borrows data for the resolver call.
func decodeRuntimeFrame[T runtimeFrame](data []byte) (*T, error) {
	var frame T
	size := int(unsafe.Sizeof(frame))
	if len(data) != size {
		return nil, fmt.Errorf("unexpected frame size: got %d, want %d", len(data), size)
	}
	return (*T)(unsafe.Pointer(unsafe.SliceData(data))), nil
}

func (r *nativeRuntimeResolver) ResolveNativeFrames(
	ctx context.Context,
	requests []nativepipeline.NativeFrameRequest,
) []nativepipeline.ResolvedNativeFrame {
	result := make([]nativepipeline.ResolvedNativeFrame, len(requests))
	for i, request := range requests {
		if request.Key.ProcessStartTime == 0 {
			continue
		}
		jitResolved := false
		for _, symbolizer := range r.jit {
			resolved, ok := symbolizer.ResolveForProcess(request.Key, request.Address)
			if !ok {
				continue
			}
			result[i].JITMapping = resolved.MappingName
			result[i].JITSymbols = make([]string, len(resolved.Symbols))
			result[i].ClearAddress = len(resolved.Symbols) > 0
			for j, symbol := range resolved.Symbols {
				result[i].JITSymbols[j] = symbol.Name
				result[i].ClearAddress = result[i].ClearAddress && symbol.Name != "" && symbol.Name != models.UnsymbolizedInterpreterLocation
			}
			jitResolved = true
			break
		}
		if jitResolved || request.JITRuntimeID != nativepipeline.JITRuntimeJVM {
			continue
		}
		if r.disableJVMMethods || r.jvm == nil {
			result[i].JVMName = models.UnsymbolizedInterpreterLocation
			continue
		}
		name, matched, err := r.jvm.SymbolizeInterpretedForProcess(ctx, request.Key, request.JITMethodAddress)
		if !matched {
			continue
		}
		if err != nil {
			name = models.UnsymbolizedInterpreterLocation
			if r.log != nil {
				r.log.Error(ctx, "Failed to symbolize interpreted JVM method", log.Error(err))
			}
		} else {
			result[i].ClearAddress = name != "" && name != models.UnsymbolizedInterpreterLocation
		}
		result[i].JVMName = name
	}
	return result
}

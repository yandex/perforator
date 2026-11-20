package profiler

import (
	"context"
	"fmt"
	"unsafe"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline/runtimewriter"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profilerext"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmregistry"
	lua "github.com/yandex/perforator/perforator/internal/linguist/lua/agent"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	php "github.com/yandex/perforator/perforator/internal/linguist/php/agent"
	python "github.com/yandex/perforator/perforator/internal/linguist/python/agent"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type runtimeFrame interface {
	unwinder.PythonFrame | unwinder.PhpFrame | unwinder.LuaFrame
}

type nativeRuntimeResolver struct {
	python            *python.Registry
	php               *php.StackProcessor
	lua               *lua.StackProcessor
	jit               []profilerext.ProcessJITSymbolizer
	jvm               *jvmregistry.Registry
	log               xlog.Logger
	disableJVMMethods bool
}

type interpreterFrameCounts struct {
	python python.RuntimeFrameCounts
	php    models.RuntimeFrameCounts
	lua    models.RuntimeFrameCounts
}

var _ nativepipeline.RuntimeResolver = (*nativeRuntimeResolver)(nil)

func (r *nativeRuntimeResolver) ResolveRuntimeFrames(
	ctx context.Context,
	request nativepipeline.RuntimeRequest,
	writer *runtimewriter.Writer,
) error {
	var jvmFrames []unwinder.JvmFrame
	for _, section := range request.Languages {
		if section.Language == nativepipeline.LanguageJVM {
			var err error
			jvmFrames, err = viewRuntimeFrames[unwinder.JvmFrame](section)
			if err != nil {
				return fmt.Errorf("decode JVM annotations: %w", err)
			}
		}
	}
	mergeJVM := validJVMIndices(jvmFrames, len(request.NativeStack))
	userJVMFrames := jvmFrames
	if !mergeJVM {
		userJVMFrames = nil
	}
	if err := r.resolveUserFrames(ctx, request, writer, userJVMFrames); err != nil {
		return err
	}
	var counts interpreterFrameCounts
	for _, section := range request.Languages {
		if section.Language == nativepipeline.LanguageJVM {
			if !mergeJVM {
				if err := r.resolveUnboundJVMFrames(ctx, request, writer, jvmFrames); err != nil {
					return err
				}
			}
		} else if err := r.resolveInterpreterSection(request.Key, section, writer, &counts); err != nil {
			return err
		}
	}
	if counts.python.Frames != 0 {
		r.python.RecordRuntimeFrameCounts(&counts.python)
	}
	if counts.php.Frames != 0 {
		r.php.RecordRuntimeFrameCounts(&counts.php)
	}
	if counts.lua.Frames != 0 {
		r.lua.RecordRuntimeFrameCounts(&counts.lua)
	}
	return nil
}

func validJVMIndices(frames []unwinder.JvmFrame, nativeCount int) bool {
	for i, frame := range frames {
		if uint64(frame.Index) >= uint64(nativeCount) || (i != 0 && frame.Index <= frames[i-1].Index) {
			return false
		}
	}
	return true
}

// Invalid annotations keep both stacks. Runtime-resolved native slots still win.
func (r *nativeRuntimeResolver) resolveUnboundJVMFrames(ctx context.Context, request nativepipeline.RuntimeRequest, writer *runtimewriter.Writer, frames []unwinder.JvmFrame) error {
	for _, frame := range frames {
		address := uint64(0)
		if uint64(frame.Index) < uint64(len(request.NativeStack)) {
			index := int(frame.Index)
			address = request.NativeStack[index]
			if address == 0 || writer.IsRuntimeLocation(index) {
				continue
			}
		}
		if err := r.resolveJVMFrame(ctx, request.Key, address, frame.MethodAddr, writer); err != nil {
			return err
		}
	}
	return nil
}

func (r *nativeRuntimeResolver) resolveInterpreterSection(key linux.ProcessKey, section nativepipeline.LanguageSection, writer *runtimewriter.Writer, counts *interpreterFrameCounts) error {
	var err error
	switch section.Language {
	case nativepipeline.LanguagePython:
		if r.python == nil {
			return fmt.Errorf("interpreter resolver is unavailable")
		}
		err = resolveInterpreterSection(section, writer, func(frame *unwinder.PythonFrame, output runtimewriter.Frame) error {
			return r.python.ResolveFrame(key, frame, output, &counts.python)
		})
	case nativepipeline.LanguagePHP:
		if r.php == nil {
			return fmt.Errorf("interpreter resolver is unavailable")
		}
		err = resolveInterpreterSection(section, writer, func(frame *unwinder.PhpFrame, output runtimewriter.Frame) error {
			return r.php.ResolveFrame(key, frame, output, &counts.php)
		})
	case nativepipeline.LanguageLua:
		if r.lua == nil {
			return fmt.Errorf("interpreter resolver is unavailable")
		}
		err = resolveInterpreterSection(section, writer, func(frame *unwinder.LuaFrame, output runtimewriter.Frame) error {
			return r.lua.ResolveFrame(key, frame, output, &counts.lua)
		})
	default:
		return fmt.Errorf("unsupported interpreter language %d", section.Language)
	}
	if err != nil {
		return fmt.Errorf("resolve language %d: %w", section.Language, err)
	}
	return nil
}

func (r *nativeRuntimeResolver) resolveUserFrames(ctx context.Context, request nativepipeline.RuntimeRequest, writer *runtimewriter.Writer, jvmFrames []unwinder.JvmFrame) error {
	nextJVM := 0
	for index, address := range request.NativeStack {
		var jvmFrame *unwinder.JvmFrame
		if nextJVM < len(jvmFrames) && uint64(jvmFrames[nextJVM].Index) == uint64(index) {
			jvmFrame = &jvmFrames[nextJVM]
			nextJVM++
		}
		if address != 0 {
			matched, err := r.tryAppendJITFrame(request.Key, address, index == 0, writer)
			if err != nil {
				return err
			}
			if matched {
				continue
			}
			if jvmFrame != nil {
				if err := r.resolveJVMFrame(ctx, request.Key, address, jvmFrame.MethodAddr, writer); err != nil {
					return err
				}
				continue
			}
		}
		if err := writer.AddNativeAddress(address); err != nil {
			return err
		}
	}
	return nil
}

func (r *nativeRuntimeResolver) tryAppendJITFrame(key linux.ProcessKey, address uint64, isLeaf bool, writer *runtimewriter.Writer) (bool, error) {
	if key.ProcessStartTime == 0 {
		return false, nil
	}
	for _, symbolizer := range r.jit {
		resolved, ok := symbolizer.ResolveForProcess(key, profilerext.Address{
			IP:   address,
			Leaf: isLeaf,
		})
		if !ok {
			continue
		}
		frame, err := writer.AddFrame(address, len(resolved.Symbols))
		if err != nil {
			return true, err
		}
		if err := frame.SetMappingName(resolved.MappingName); err != nil {
			return true, err
		}
		fullySymbolized := len(resolved.Symbols) != 0
		for _, symbol := range resolved.Symbols {
			fullySymbolized = fullySymbolized && symbol.Name != "" && symbol.Name != models.UnsymbolizedInterpreterLocation
			if err := frame.AddInlineFunction(symbol.Name, "", 0, 0); err != nil {
				return true, err
			}
		}
		if fullySymbolized {
			frame.SetAddress(0)
		}
		return true, nil
	}
	return false, nil
}

func (r *nativeRuntimeResolver) resolveJVMFrame(ctx context.Context, key linux.ProcessKey, address, method uint64, writer *runtimewriter.Writer) error {
	name := models.UnsymbolizedInterpreterLocation
	if key.ProcessStartTime != 0 && !r.disableJVMMethods && r.jvm != nil {
		resolved, matched, err := r.jvm.SymbolizeInterpretedForProcess(ctx, key, method)
		if err != nil {
			if r.log != nil {
				r.log.Error(ctx, "Failed to symbolize interpreted JVM method", log.Error(err))
			}
		} else if matched && resolved != "" {
			name = resolved
		}
	}
	frame, err := writer.AddFrame(address, 1)
	if err != nil {
		return err
	}
	frame.SetLanguage(uint8(nativepipeline.LanguageJVM))
	if err := frame.SetMappingName("[jvm]"); err != nil {
		return err
	}
	if name != models.UnsymbolizedInterpreterLocation {
		frame.SetAddress(0)
	}
	return frame.AddInlineFunction(name, "", 0, 0)
}

func resolveInterpreterSection[T runtimeFrame](
	section nativepipeline.LanguageSection,
	writer *runtimewriter.Writer,
	resolve func(*T, runtimewriter.Frame) error,
) error {
	frames, err := viewRuntimeFrames[T](section)
	if err != nil {
		return err
	}
	for i := range frames {
		output, err := writer.AddFrame(0, 1)
		if err != nil {
			return err
		}
		output.SetLanguage(uint8(section.Language))
		if err := resolve(&frames[i], output); err != nil {
			return err
		}
	}
	return nil
}

// Frames borrow the BPF payload for the synchronous resolver call.
func viewRuntimeFrames[T runtimeFrame | unwinder.JvmFrame](section nativepipeline.LanguageSection) ([]T, error) {
	var frame T
	size := int(unsafe.Sizeof(frame))
	if int(section.FrameSize) != size || len(section.Data) == 0 || len(section.Data)%size != 0 {
		return nil, fmt.Errorf("unexpected frame layout: size %d, payload %d, want %d", section.FrameSize, len(section.Data), size)
	}
	return unsafe.Slice((*T)(unsafe.Pointer(unsafe.SliceData(section.Data))), len(section.Data)/size), nil
}

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

var _ nativepipeline.RuntimeResolver = (*nativeRuntimeResolver)(nil)

func (r *nativeRuntimeResolver) ResolveRuntimeFrames(
	ctx context.Context,
	request nativepipeline.RuntimeRequest,
	writer *runtimewriter.Writer,
) error {
	var annotations *nativepipeline.LanguageSection
	for i := range request.Languages {
		if request.Languages[i].Language == nativepipeline.LanguageJVM {
			annotations = &request.Languages[i]
			break
		}
	}
	if err := r.resolveNativeFrames(ctx, request, annotations, writer); err != nil {
		return err
	}
	for _, section := range request.Languages {
		if section.Language != nativepipeline.LanguageJVM {
			if err := r.resolveInterpreterSection(request.Key, section, writer); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *nativeRuntimeResolver) resolveInterpreterSection(key linux.ProcessKey, section nativepipeline.LanguageSection, writer *runtimewriter.Writer) error {
	var err error
	switch section.Language {
	case nativepipeline.LanguagePython:
		if r.python == nil {
			return fmt.Errorf("interpreter resolver is unavailable")
		}
		err = resolveInterpreterSection(r.python.ResolveFrame, key, section, writer)
	case nativepipeline.LanguagePHP:
		if r.php == nil {
			return fmt.Errorf("interpreter resolver is unavailable")
		}
		err = resolveInterpreterSection(r.php.ResolveFrame, key, section, writer)
	case nativepipeline.LanguageLua:
		if r.lua == nil {
			return fmt.Errorf("interpreter resolver is unavailable")
		}
		err = resolveInterpreterSection(r.lua.ResolveFrame, key, section, writer)
	default:
		return fmt.Errorf("unsupported interpreter language %d", section.Language)
	}
	if err != nil {
		return fmt.Errorf("resolve language %d: %w", section.Language, err)
	}
	return nil
}

func (r *nativeRuntimeResolver) resolveNativeFrames(ctx context.Context, request nativepipeline.RuntimeRequest, annotations *nativepipeline.LanguageSection, writer *runtimewriter.Writer) error {
	var frames []unwinder.JvmFrame
	if annotations != nil {
		var err error
		frames, err = viewRuntimeFrames[unwinder.JvmFrame](*annotations)
		if err != nil {
			return fmt.Errorf("decode JVM annotations: %w", err)
		}
		previous := -1
		for _, frame := range frames {
			index := int(frame.Index)
			if index <= previous || index >= len(request.NativeStack) {
				return fmt.Errorf("invalid JVM native stack index %d", index)
			}
			previous = index
		}
	}
	next := 0
	for index, address := range request.NativeStack {
		var frame *unwinder.JvmFrame
		if next < len(frames) && int(frames[next].Index) == index {
			frame = &frames[next]
			next++
		}
		matched := false
		if address != 0 {
			var err error
			isLeaf := index == 0
			matched, err = r.tryAppendJITFrame(request.Key, address, isLeaf, writer)
			if err != nil {
				return err
			}
			if !matched && frame != nil {
				if err := r.resolveJVMFrame(ctx, request.Key, address, frame.MethodAddr, writer); err != nil {
					return err
				}
				matched = true
			}
		}
		if !matched {
			if _, err := writer.AddFrame(address, 0); err != nil {
				return err
			}
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
		for _, symbol := range resolved.Symbols {
			if err := frame.AddInlineFunction(symbol.Name, "", 0, 0); err != nil {
				return true, err
			}
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
	if err := frame.SetMappingName("[jvm]"); err != nil {
		return err
	}
	return frame.AddInlineFunction(name, "", 0, 0)
}

func resolveInterpreterSection[T runtimeFrame](resolve func(linux.ProcessKey, *T, runtimewriter.Frame) error, key linux.ProcessKey, section nativepipeline.LanguageSection, writer *runtimewriter.Writer) error {
	frames, err := viewRuntimeFrames[T](section)
	if err != nil {
		return err
	}
	for i := range frames {
		output, err := writer.AddFrame(0, 1)
		if err != nil {
			return err
		}
		if err := resolve(key, &frames[i], output); err != nil {
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

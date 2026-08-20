package agent

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

// trampolineLinestart marks a frame that is a CPython eval-loop trampoline
// rather than a real code object.
const trampolineLinestart int32 = -1

// StackProcessor renders a Python stack collected by the unwinder into pprof locations.
type StackProcessor struct {
	symbolizer             *Symbolizer
	collectedFrameCount    metrics.Counter
	unsymbolizedFrameCount metrics.Counter
}

func NewStackProcessor(symbolizer *Symbolizer, reg metrics.Registry) *StackProcessor {
	return &StackProcessor{
		symbolizer:             symbolizer,
		collectedFrameCount:    reg.Counter("python.frame.collected.count"),
		unsymbolizedFrameCount: reg.Counter("python.frame.unsymbolized.count"),
	}
}

func (p *StackProcessor) Process(
	builder *profile.SampleBuilder,
	stack *unwinder.PythonStack,
	pid uint32,
) {
	var frames uint32
	for i := 0; i < int(stack.Len); i++ {
		p.processFrame(builder, &stack.Frames[i], pid)
		frames++
	}
	p.collectedFrameCount.Add(int64(frames))
}

func (p *StackProcessor) processFrame(
	builder *profile.SampleBuilder,
	frame *unwinder.PythonFrame,
	pid uint32,
) {
	if frame.SymbolKey.Linestart == trampolineLinestart {
		loc := p.addLocation(builder, frame, 0)
		loc.AddFrame().SetName(python_models.PythonTrampolineFrame).Finish()
		loc.Finish()
		return
	}

	symbol, line, ok := p.symbolizer.SymbolizeFrame(pid, frame)
	if !ok {
		p.unsymbolizedFrameCount.Inc()

		loc := p.addLocation(builder, frame, 0)
		loc.AddFrame().
			SetName(models.UnsymbolizedInterpreterLocation).
			SetStartLine(int64(frame.SymbolKey.Linestart)).
			Finish()
		loc.Finish()
		return
	}

	loc := p.addLocation(builder, frame, line)
	fb := loc.AddFrame().
		SetName(symbol.Name).
		SetFilename(symbol.FileName).
		SetStartLine(int64(frame.SymbolKey.Linestart))
	if line > 0 {
		fb.SetLine(int64(line))
	}
	fb.Finish()
	loc.Finish()
}

func (p *StackProcessor) addLocation(
	builder *profile.SampleBuilder,
	frame *unwinder.PythonFrame,
	line int32,
) *profile.LocationBuilder {
	loc := builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
		ObjectAddress: frame.SymbolKey.ObjectAddr,
		Linestart:     frame.SymbolKey.Linestart,
		Line:          line,
	})
	loc.SetMapping().SetPath(string(profile.PythonSpecialMapping)).Finish()
	return loc
}

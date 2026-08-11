package agent

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

// trampolineLinestart marks a frame that is a CPython eval-loop trampoline
// rather than a real code object.
const trampolineLinestart int32 = -1

// StackProcessor renders a Python stack collected by the unwinder into pprof locations.
type StackProcessor struct {
	symbolizer             *symbolizer.Symbolizer
	collectedFrameCount    metrics.Counter
	unsymbolizedFrameCount metrics.Counter
}

func NewStackProcessor(symbolizer *symbolizer.Symbolizer, reg metrics.Registry) *StackProcessor {
	return &StackProcessor{
		symbolizer:             symbolizer,
		collectedFrameCount:    reg.Counter("python.frame.collected.count"),
		unsymbolizedFrameCount: reg.Counter("python.frame.unsymbolized.count"),
	}
}

func (p *StackProcessor) Process(
	builder *profile.SampleBuilder,
	stack *unwinder.InterpreterStack,
) {
	var frames uint32
	for i := 0; i < int(stack.Len); i++ {
		frame := &stack.Frames[i]

		loc := builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: frame.SymbolKey.ObjectAddr,
			Linestart:     frame.SymbolKey.Linestart,
		})
		loc.SetMapping().SetPath(string(profile.PythonSpecialMapping)).Finish()

		p.processFrame(loc, frame)

		loc.Finish()
		frames++
	}
	p.collectedFrameCount.Add(int64(frames))
}

func (p *StackProcessor) processFrame(
	loc *profile.LocationBuilder,
	frame *unwinder.InterpreterFrame,
) {
	if frame.SymbolKey.Linestart == trampolineLinestart {
		loc.AddFrame().SetName(python_models.PythonTrampolineFrame).Finish()
		return
	}

	symbol, exists := p.symbolizer.Symbolize(&frame.SymbolKey)
	if !exists {
		p.unsymbolizedFrameCount.Inc()
		loc.AddFrame().
			SetName(models.UnsymbolizedInterpreterLocation).
			SetStartLine(int64(frame.SymbolKey.Linestart)).
			Finish()
		return
	}

	loc.AddFrame().
		SetName(symbol.Name).
		SetFilename(symbol.FileName).
		SetStartLine(int64(frame.SymbolKey.Linestart)).
		Finish()
}

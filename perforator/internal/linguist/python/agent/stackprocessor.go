package agent

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// trampolineLinestart marks a frame that is a CPython eval-loop trampoline
// rather than a real code object.
const trampolineLinestart int32 = -1

// StackProcessor renders a Python stack collected by the unwinder into pprof locations.
type StackProcessor struct {
	symbolizer             *Symbolizer
	lineInfo               lineInfoMetrics
	collectedFrameCount    metrics.Counter
	unsymbolizedFrameCount metrics.Counter
}

func NewStackProcessor(symbolizer *Symbolizer, reg metrics.Registry) *StackProcessor {
	return &StackProcessor{
		symbolizer:             symbolizer,
		lineInfo:               newLineInfoMetrics(reg, symbolizer != nil && symbolizer.offsets != nil),
		collectedFrameCount:    reg.Counter("python.frame.collected.count"),
		unsymbolizedFrameCount: reg.Counter("python.frame.unsymbolized.count"),
	}
}

func (p *StackProcessor) Process(
	builder *profile.SampleBuilder,
	stack *unwinder.PythonStack,
	process linux.ProcessKey,
) {
	var frames uint32
	var lineCounts [lineInfoOutcomeCount]int64
	for i := 0; i < int(stack.Len); i++ {
		outcome := p.processFrame(builder, &stack.Frames[i], process)
		lineCounts[outcome]++
		frames++
	}
	p.collectedFrameCount.Add(int64(frames))
	p.lineInfo.record(&lineCounts)
}

func (p *StackProcessor) processFrame(
	builder *profile.SampleBuilder,
	frame *unwinder.PythonFrame,
	process linux.ProcessKey,
) lineInfoOutcome {
	if frame.SymbolKey.Linestart == trampolineLinestart {
		loc := p.addLocation(builder, frame, 0)
		loc.AddFrame().SetName(python_models.PythonTrampolineFrame).Finish()
		loc.Finish()
		return lineSkipped
	}

	symbol, line, ok, resolution := p.symbolizer.symbolizeFrame(process, frame)
	if !ok {
		p.unsymbolizedFrameCount.Inc()

		loc := p.addLocation(builder, frame, 0)
		loc.AddFrame().
			SetName(models.UnsymbolizedInterpreterLocation).
			SetStartLine(int64(frame.SymbolKey.Linestart)).
			Finish()
		loc.Finish()
		return resolution
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
	return resolution
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
		Language:      models.Language(unwinder.LanguagePython),
	})
	loc.SetMapping().SetPath(string(profile.PythonSpecialMapping)).Finish()
	return loc
}

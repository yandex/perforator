package agent

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// StackProcessor renders a PHP stack collected by the unwinder into pprof locations.
type StackProcessor struct {
	symbolizer             *symbolizer.Symbolizer
	collectedFrameCount    metrics.Counter
	unsymbolizedFrameCount metrics.Counter
}

func NewStackProcessor(symbolizer *symbolizer.Symbolizer, reg metrics.Registry) *StackProcessor {
	return &StackProcessor{
		symbolizer:             symbolizer,
		collectedFrameCount:    reg.Counter("php.frame.collected.count"),
		unsymbolizedFrameCount: reg.Counter("php.frame.unsymbolized.count"),
	}
}

func (p *StackProcessor) Process(
	builder *profile.SampleBuilder,
	stack *unwinder.PhpStack,
	process linux.ProcessKey,
) {
	var frames uint32
	for i := 0; i < int(stack.Len); i++ {
		frame := &stack.Frames[i]

		loc := builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: frame.SymbolKey.ObjectAddr,
			Linestart:     frame.SymbolKey.Linestart,
			Language:      models.Language(unwinder.LanguagePhp),
		})
		loc.SetMapping().SetPath(string(profile.PHPSpecialMapping)).Finish()

		p.processFrame(loc, frame, process)

		loc.Finish()
		frames++
	}
	p.collectedFrameCount.Add(int64(frames))
}

func (p *StackProcessor) processFrame(
	loc *profile.LocationBuilder,
	frame *unwinder.PhpFrame,
	process linux.ProcessKey,
) {
	symbol, exists := p.symbolizer.Symbolize(models.Language(unwinder.LanguagePhp), process, &frame.SymbolKey)
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

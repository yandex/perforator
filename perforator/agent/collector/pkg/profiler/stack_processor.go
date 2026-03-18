package profiler

import (
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

type interpreterStackMetrics struct {
	framesCount             uint32
	unsymbolizedFramesCount uint32
}

type sampleStackProcessor struct {
	interpreterSymbolizer *symbolizer.Symbolizer
	langMapping           profile.SpecialMapping
}

func newPythonSampleStackProcessor(symbolizer *symbolizer.Symbolizer) *sampleStackProcessor {
	return &sampleStackProcessor{
		interpreterSymbolizer: symbolizer,
		langMapping:           profile.PythonSpecialMapping,
	}
}

func newPHPSampleStackProcessor(symbolizer *symbolizer.Symbolizer) *sampleStackProcessor {
	return &sampleStackProcessor{
		interpreterSymbolizer: symbolizer,
		langMapping:           profile.PHPSpecialMapping,
	}
}

func (s *sampleStackProcessor) Process(builder *profile.SampleBuilder, stack *unwinder.InterpreterStack) interpreterStackMetrics {
	n := int(stack.Len)
	mtr := interpreterStackMetrics{}

	// Collect all symbol keys up front; SymbolizeBatch resolves them via
	// LRU cache (no syscall) then per-miss eBPF Lookup — before profile construction.
	keys := make([]unwinder.SymbolKey, n)
	for i := range n {
		keys[i] = stack.Frames[i].SymbolKey
	}
	prefetched := s.interpreterSymbolizer.SymbolizeBatch(keys)

	processFrame := s.getFrameProcessor()
	for i := range n {
		loc := builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: stack.Frames[i].SymbolKey.ObjectAddr,
			Linestart:     stack.Frames[i].SymbolKey.Linestart,
		})
		loc.SetMapping().SetPath(string(s.langMapping)).Finish()
		processFrame(s, &mtr, loc, &stack.Frames[i], prefetched[i])
		loc.Finish()
		mtr.framesCount++
	}

	return mtr
}

func processFrameCommon(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame, sym *symbolizer.Symbol) {
	if sym == nil {
		mtr.unsymbolizedFramesCount++
		loc.AddFrame().
			SetName(models.UnsymbolizedInterpreterLocation).
			SetStartLine(int64(frame.SymbolKey.Linestart)).
			Finish()
		return
	}

	loc.AddFrame().
		SetName(sym.Name).
		SetFilename(sym.FileName).
		SetStartLine(int64(frame.SymbolKey.Linestart)).
		Finish()
}

func processPythonFrame(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame, sym *symbolizer.Symbol) {
	if frame.SymbolKey.Linestart == -1 {
		loc.AddFrame().SetName(python_models.PythonTrampolineFrame).Finish()
		return
	}
	processFrameCommon(s, mtr, loc, frame, sym)
}

func (s *sampleStackProcessor) getFrameProcessor() func(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame, sym *symbolizer.Symbol) {
	switch s.langMapping {
	case profile.PythonSpecialMapping:
		return processPythonFrame
	default:
		return processFrameCommon
	}
}

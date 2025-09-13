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
	processFrame := s.getFrameProcessor()
	mtr := interpreterStackMetrics{}

	for i := 0; i < int(stack.Len); i++ {
		loc := builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: stack.Frames[i].SymbolKey.ObjectAddr,
			Linestart:     stack.Frames[i].SymbolKey.Linestart,
		})

		processFrame(s, &mtr, loc, &stack.Frames[i])

		loc.Finish()
		mtr.framesCount++
	}

	return mtr
}

func processFrameCommon(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	symbol, exists := s.interpreterSymbolizer.Symbolize(&frame.SymbolKey)
	if !exists {
		mtr.unsymbolizedFramesCount++
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

func processPythonFrame(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	if frame.SymbolKey.Linestart == -1 {
		loc.AddFrame().SetName(python_models.PythonTrampolineFrame).Finish()
		return
	}

	loc.SetMapping().SetPath(string(s.langMapping)).Finish()
	processFrameCommon(s, mtr, loc, frame)
}

func (s *sampleStackProcessor) getFrameProcessor() func(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	switch s.langMapping {
	case profile.PythonSpecialMapping:
		return processPythonFrame
	default:
		return processFrameCommon
	}
}

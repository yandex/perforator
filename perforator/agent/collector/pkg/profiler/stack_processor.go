package profiler

import (
	"fmt"
	"strconv"

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

func newLuaSampleStackProcessor(symbolizer *symbolizer.Symbolizer) *sampleStackProcessor {
	return &sampleStackProcessor{
		interpreterSymbolizer: symbolizer,
		langMapping:           profile.LuaSpecialMapping,
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

		loc.SetMapping().SetPath(string(s.langMapping)).Finish()
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

	processFrameCommon(s, mtr, loc, frame)
}

// internal frame decoding errors, see lua_stack_walk_error at perforator/agent/collector/progs/unwinder/lua/stack/walk_error.h
var luaStackWalkErrorDescriptions = []string{
	"frame is null",
	"GCfunc is null",
	"bad function in frame",
	"proto is null",
}

// see lj_obj.h
var gcTypes = []string{
	"nil",
	"false",
	"true",
	"light userdata",
	"string",
	"upvalue",
	"thread",
	"proto",
	"function",
	"trace",
	"cdata",
	"table",
	"userdata",
}

type LuaData struct {
	frame *unwinder.InterpreterFrame
}

const (
	LuaObjectAddressPtrBits = 48
	LuaObjectAddressPtrMask = (1 << LuaObjectAddressPtrBits) - 1

	LuaObjectAddressFfidOffset  = LuaObjectAddressPtrBits
	LuaObjectAddressFfidBits    = 8
	LuaObjectAddressFfidMask    = (1 << LuaObjectAddressFfidBits) - 1
	LuaObjectAddressFfidLua     = 0
	LuaObjectAddressFfidC       = 1
	LuaObjectAddressFfidInvalid = -1

	LuaObjectAddressStackWalkErrorOffset = LuaObjectAddressPtrBits
	LuaObjectAddressStackWalkErrorBits   = 2
	LuaObjectAddressStackWalkErrorMask   = (1 << LuaObjectAddressStackWalkErrorBits) - 1

	LuaObjectAddressGctOffset = LuaObjectAddressStackWalkErrorOffset + LuaObjectAddressStackWalkErrorBits
	LuaObjectAddressGctBits   = 8
	LuaObjectAddressGctMask   = (1 << LuaObjectAddressGctBits) - 1

	LuaLinestartNonLuaFrame = -1
)

func (ld *LuaData) getObjectAddress() uint64 {
	return ld.frame.SymbolKey.ObjectAddr
}

func (ld *LuaData) GetLineStart() int32 {
	return ld.frame.SymbolKey.Linestart
}

func (ld *LuaData) GetFrameType() int {
	if ld.GetLineStart() != LuaLinestartNonLuaFrame {
		return LuaObjectAddressFfidLua
	} else if ld.GetPtr() == 0 {
		return LuaObjectAddressFfidInvalid
	}

	return int(ld.GetFfid())
}

func (ld *LuaData) GetPtr() uint64 {
	return ld.getObjectAddress() & LuaObjectAddressPtrMask
}

func (ld *LuaData) GetFfid() uint8 {
	return uint8((ld.getObjectAddress() >> LuaObjectAddressFfidOffset) & LuaObjectAddressFfidMask)
}

func (ld *LuaData) GetErrorKind() unwinder.LuaStackWalkError {
	return unwinder.LuaStackWalkError((ld.getObjectAddress() >> LuaObjectAddressStackWalkErrorOffset) & LuaObjectAddressStackWalkErrorMask)
}

func (ld *LuaData) GetFrameGct() uint8 {
	return uint8((ld.getObjectAddress() >> LuaObjectAddressGctOffset) & LuaObjectAddressGctMask)
}

func processLuaFrame(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	name := "[lua] "
	filename := ""

	luaData := LuaData{frame}
	switch luaData.GetFrameType() {
	case LuaObjectAddressFfidInvalid:
		// Invalid frame
		name += "<invalid lua frame>"

		if int(luaData.GetErrorKind()) < len(luaStackWalkErrorDescriptions) {
			name += ": " + luaStackWalkErrorDescriptions[luaData.GetErrorKind()]

			if luaData.GetErrorKind() == unwinder.LuaStackWalkErrorFrameIsNotFunc && int(luaData.GetFrameGct()) < len(gcTypes) {
				name += ", was " + gcTypes[luaData.GetFrameGct()]
			}
		}
	case LuaObjectAddressFfidLua:
		// Lua frame
		symbol, exists := s.interpreterSymbolizer.Symbolize(&frame.SymbolKey)

		if !exists {
			mtr.unsymbolizedFramesCount++

			name += fmt.Sprintf("unsymbolized lua function: 0x%x", luaData.GetPtr())
		} else {
			name += symbol.Name
			filename = symbol.FileName

			// Usually scripts has @ appended at the beginning.
			// Perforator has same symbol, removing here.
			if len(filename) != 0 && filename[0] == '@' {
				filename = symbol.FileName[1:]
			}

			// TODO: Perforator broke line numbers
			if frame.SymbolKey.Linestart != 0 {
				filename += ":" + strconv.Itoa(int(frame.SymbolKey.Linestart))
			}
		}
	case LuaObjectAddressFfidC:
		// C frame

		// The frame will try to symbolize by postprocess
		name += fmt.Sprintf("function: 0x%x", luaData.GetPtr())
	default:
		// FF frame

		// The frame will try to symbolize by postprocess
		// If postprocess will fail, at least print its builtin number to find the name manually
		name += "function: builtin#" + strconv.Itoa(int(luaData.GetFfid()))
	}

	loc.AddFrame().
		SetName(name).
		SetFilename(filename).
		SetStartLine(int64(frame.SymbolKey.Linestart)).
		Finish()
}

func (s *sampleStackProcessor) getFrameProcessor() func(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	switch s.langMapping {
	case profile.PythonSpecialMapping:
		return processPythonFrame
	case profile.LuaSpecialMapping:
		return processLuaFrame
	default:
		return processFrameCommon
	}
}

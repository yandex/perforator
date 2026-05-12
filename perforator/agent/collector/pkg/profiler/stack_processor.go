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
	println("SPAR: stack_processor::newLuaSampleStackProcessor")

	return &sampleStackProcessor{
		interpreterSymbolizer: symbolizer,
		langMapping:           profile.LuaSpecialMapping,
	}
}

func (s *sampleStackProcessor) Process(builder *profile.SampleBuilder, stack *unwinder.InterpreterStack) interpreterStackMetrics {
	println("SPAR: stack_processor::Process")
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
	println("SPAR: stack_processor::processFrameCommon")
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

// array of Lua built-in names (fast functions)
// hard-coded for now, taken from vmdef.lua (generated file)
// fragile, as array depends on order of files being compiled in LuaJIT!
var ffnames = []string{
	"Lua",
	"C",
	"assert",
	"type",
	"next",
	"pairs",
	"ipairs_aux",
	"ipairs",
	"getmetatable",
	"setmetatable",
	"getfenv",
	"setfenv",
	"rawget",
	"rawset",
	"rawequal",
	"unpack",
	"select",
	"tonumber",
	"tostring",
	"error",
	"pcall",
	"xpcall",
	"loadfile",
	"load",
	"loadstring",
	"dofile",
	"gcinfo",
	"collectgarbage",
	"newproxy",
	"print",
	"coroutine.status",
	"coroutine.running",
	"coroutine.isyieldable",
	"coroutine.create",
	"coroutine.yield",
	"coroutine.resume",
	"coroutine.wrap_aux",
	"coroutine.wrap",
	"math.abs",
	"math.floor",
	"math.ceil",
	"math.sqrt",
	"math.log10",
	"math.exp",
	"math.sin",
	"math.cos",
	"math.tan",
	"math.asin",
	"math.acos",
	"math.atan",
	"math.sinh",
	"math.cosh",
	"math.tanh",
	"math.frexp",
	"math.modf",
	"math.log",
	"math.atan2",
	"math.pow",
	"math.fmod",
	"math.ldexp",
	"math.min",
	"math.max",
	"math.random",
	"math.randomseed",
	"bit.tobit",
	"bit.bnot",
	"bit.bswap",
	"bit.lshift",
	"bit.rshift",
	"bit.arshift",
	"bit.rol",
	"bit.ror",
	"bit.band",
	"bit.bor",
	"bit.bxor",
	"bit.tohex",
	"string.byte",
	"string.char",
	"string.sub",
	"string.rep",
	"string.reverse",
	"string.lower",
	"string.upper",
	"string.dump",
	"string.find",
	"string.match",
	"string.gmatch_aux",
	"string.gmatch",
	"string.gsub",
	"string.format",
	"table.maxn",
	"table.insert",
	"table.concat",
	"table.sort",
	"table.new",
	"table.clear",
	"io.method.close",
	"io.method.read",
	"io.method.write",
	"io.method.flush",
	"io.method.seek",
	"io.method.setvbuf",
	"io.method.lines",
	"io.method.__gc",
	"io.method.__tostring",
	"io.open",
	"io.popen",
	"io.tmpfile",
	"io.close",
	"io.read",
	"io.write",
	"io.flush",
	"io.input",
	"io.output",
	"io.lines",
	"io.type",
	"os.execute",
	"os.remove",
	"os.rename",
	"os.tmpname",
	"os.getenv",
	"os.exit",
	"os.clock",
	"os.date",
	"os.time",
	"os.difftime",
	"os.setlocale",
	"debug.getregistry",
	"debug.getmetatable",
	"debug.setmetatable",
	"debug.getfenv",
	"debug.setfenv",
	"debug.getinfo",
	"debug.getlocal",
	"debug.setlocal",
	"debug.getupvalue",
	"debug.setupvalue",
	"debug.upvalueid",
	"debug.upvaluejoin",
	"debug.sethook",
	"debug.gethook",
	"debug.debug",
	"debug.traceback",
	"jit.on",
	"jit.off",
	"jit.flush",
	"jit.status",
	"jit.security",
	"jit.attach",
	"jit.util.funcinfo",
	"jit.util.funcbc",
	"jit.util.funck",
	"jit.util.funcuvname",
	"jit.util.traceinfo",
	"jit.util.traceir",
	"jit.util.tracek",
	"jit.util.tracesnap",
	"jit.util.tracemc",
	"jit.util.traceexitstub",
	"jit.util.ircalladdr",
	"jit.opt.start",
	"jit.profile.start",
	"jit.profile.stop",
	"jit.profile.dumpstack",
	"ffi.meta.__index",
	"ffi.meta.__newindex",
	"ffi.meta.__eq",
	"ffi.meta.__len",
	"ffi.meta.__lt",
	"ffi.meta.__le",
	"ffi.meta.__concat",
	"ffi.meta.__call",
	"ffi.meta.__add",
	"ffi.meta.__sub",
	"ffi.meta.__mul",
	"ffi.meta.__div",
	"ffi.meta.__mod",
	"ffi.meta.__pow",
	"ffi.meta.__unm",
	"ffi.meta.__tostring",
	"ffi.meta.__pairs",
	"ffi.meta.__ipairs",
	"ffi.clib.__index",
	"ffi.clib.__newindex",
	"ffi.clib.__gc",
	"ffi.callback.free",
	"ffi.callback.set",
	"ffi.cdef",
	"ffi.new",
	"ffi.cast",
	"ffi.typeof",
	"ffi.typeinfo",
	"ffi.istype",
	"ffi.sizeof",
	"ffi.alignof",
	"ffi.offsetof",
	"ffi.errno",
	"ffi.string",
	"ffi.copy",
	"ffi.fill",
	"ffi.abi",
	"ffi.metatype",
	"ffi.gc",
	"ffi.load",
	"buffer.method.free",
	"buffer.method.reset",
	"buffer.method.skip",
	"buffer.method.set",
	"buffer.method.put",
	"buffer.method.putf",
	"buffer.method.get",
	"buffer.method.putcdata",
	"buffer.method.reserve",
	"buffer.method.commit",
	"buffer.method.ref",
	"buffer.method.encode",
	"buffer.method.decode",
	"buffer.method.__gc",
	"buffer.method.__tostring",
	"buffer.method.__len",
	"buffer.new",
	"buffer.encode",
	"buffer.decode",
}

// internal frame decoding errors, see LuaUnwindError at perforator/agent/collector/progs/unwinder/lua/types.h
var frame_decoding_errors = []string{
	"frame is null",
	"gc func is null",
	"bad frame function",
}

// see lj_obj.h
var gct = []string{
	"TNIL",
	"TFALSE",
	"TTRUE",
	"TLIGHTUD",
	"TSTR",
	"TUPVAL",
	"TTHREAD",
	"TPROTO",
	"TFUNC",
	"TTRACE",
	"TCDATA",
	"TTAB",
	"TUDATA",
	"TNUMX",
}

// see LuaEntity
var entities = []string{
	"",
	"metamethod",
	"local",
	"global",
	"method",
	"field",
	"upvalue",
}

type LuaData struct {
	frame *unwinder.InterpreterFrame
}

const (
	LuaPointerMask    = 0xFFFFFFFFFFFF
	LuaPointerBitSize = 48

	LuaContextTypeMask = 0x7

	LuaFfidMask    = 0xFF
	LuaFfidLua     = 0
	LuaFfidC       = 1
	LuaFfidInvalid = -1

	LuaUnwindErrorMask    = 0x3
	LuaUnwindErrorBitSize = 2

	LuaFrameGctMask = 0xF

	LuaLineStartNotUsed = -1
)

func (ld *LuaData) getObjectAddress() uint64 {
	return ld.frame.SymbolKey.ObjectAddr
}

func (ld *LuaData) GetLineStart() int32 {
	return ld.frame.SymbolKey.Linestart
}

func (ld *LuaData) GetFrameType() int {
	if ld.GetLineStart() != LuaLineStartNotUsed {
		return LuaFfidLua
	} else if ld.getObjectAddress() == 0 {
		return LuaFfidInvalid
	}

	return int(ld.GetFfid())
}

func (ld *LuaData) GetPtr() uint64 {
	return ld.getObjectAddress() & LuaPointerMask
}

func (ld *LuaData) GetContextType() unwinder.LuaContextType {
	return unwinder.LuaContextType((ld.getObjectAddress() >> LuaPointerBitSize) & LuaContextTypeMask)
}

func (ld *LuaData) GetFfid() uint8 {
	return uint8((ld.getObjectAddress() >> LuaPointerBitSize) & LuaFfidMask)
}

func (ld *LuaData) GetErrorKind() unwinder.LuaUnwindError {
	return unwinder.LuaUnwindError((ld.getObjectAddress() >> LuaPointerBitSize) & LuaContextTypeMask)
}

func (ld *LuaData) GetFrameGct() uint8 {
	return uint8((ld.getObjectAddress() >> (LuaPointerBitSize + LuaUnwindErrorBitSize)) & LuaFrameGctMask)
}

func processLuaFrame(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	println("SPAR: stack_processor::processLuaFrame")

	loc.SetMapping().SetPath(string(s.langMapping)).Finish()

	symbol, exists := s.interpreterSymbolizer.Symbolize(&frame.SymbolKey)

	if !exists {
		mtr.unsymbolizedFramesCount++

		loc.AddFrame().
			SetName(models.UnsymbolizedInterpreterLocation).
			SetStartLine(int64(frame.SymbolKey.Linestart)).
			Finish()
		return
	}

	luaData := LuaData{frame}
	name := symbol.Name
	filename := symbol.FileName

	switch luaData.GetFrameType() {
	case LuaFfidInvalid:
		// Invalid frame
		name = "<Invalid Lua Frame>"

		if int(luaData.GetFrameGct()) < len(gct) {
			name += "#" + gct[luaData.GetFrameGct()]
		}

		if int(luaData.GetErrorKind()) < len(frame_decoding_errors) {
			name += ": " + frame_decoding_errors[luaData.GetErrorKind()]
		}
	case LuaFfidLua:
		if luaData.GetContextType() != 0 {
			name = entities[luaData.GetContextType()] + " " + name
		}

		// Usually scripts has @ appended at the beginning.
		// Perforator has same symbol, removing here.
		if filename[0] == '@' {
			filename = symbol.FileName[1:]
		}

		filename += ":" + strconv.Itoa(int(frame.SymbolKey.Linestart))
	case LuaFfidC:
		// The frame will be symbolized by postprocess
		name = fmt.Sprintf("function: 0x%x", luaData.GetPtr())
	default:
		// The frame will be symbolized by postprocess
		// If postprocess will fail, at least print its builtin number to find the name manually
		name = "[builtin] " + ffnames[luaData.GetFfid()] + "#" + strconv.Itoa(int(luaData.GetFfid()))
	}

	loc.AddFrame().
		SetName("[lua] " + name).
		SetFilename(filename).
		SetStartLine(int64(frame.SymbolKey.Linestart)).
		Finish()
}

func (s *sampleStackProcessor) getFrameProcessor() func(s *sampleStackProcessor, mtr *interpreterStackMetrics, loc *profile.LocationBuilder, frame *unwinder.InterpreterFrame) {
	println("SPAR: stack_processor::getFrameProcessor")

	switch s.langMapping {
	case profile.PythonSpecialMapping:
		return processPythonFrame
	case profile.LuaSpecialMapping:
		return processLuaFrame
	default:
		return processFrameCommon
	}
}

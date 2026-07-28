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

// array of Lua built-in names (fast functions)
// hard-coded for now, taken from vmdef.lua (generated file)
// fragile, as array depends on order of files being compiled in LuaJIT!
var builtin_function_names = []string{
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

// internal frame decoding errors, see lua_stack_walk_error at perforator/agent/collector/progs/unwinder/lua/stack/walk_error.h
var lua_stack_walk_error_descriptions = []string{
	"frame is null",
	"GCfunc is null",
	"bad function in frame",
	"proto is null",
}

// see lj_obj.h
var gc_types = []string{
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

		if int(luaData.GetErrorKind()) < len(lua_stack_walk_error_descriptions) {
			name += ": " + lua_stack_walk_error_descriptions[luaData.GetErrorKind()]

			if luaData.GetErrorKind() == unwinder.LuaStackWalkErrorFrameIsNotFunc && int(luaData.GetFrameGct()) < len(gc_types) {
				name += ", was " + gc_types[luaData.GetFrameGct()]
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
			if filename[0] == '@' {
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
		name += "function: builtin#" + strconv.Itoa(int(luaData.GetFfid())) + " (" + builtin_function_names[luaData.GetFfid()] + ")"
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

package agent

import (
	"fmt"
	"strconv"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

// Internal frame decoding errors, see lua_stack_walk_error at perforator/agent/collector/progs/unwinder/lua/stack/walk_error.h
var luaStackWalkErrorDescriptions = []string{
	"GCfunc is null",
	"bad function in frame",
}

// See LJ_T object tags in lj_obj.h
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
	LuaObjectAddressStackWalkErrorBits   = 1
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

// StackProcessor renders a Lua stack collected by the unwinder into pprof locations.
type StackProcessor struct {
	symbolizer             *symbolizer.Symbolizer
	collectedFrameCount    metrics.Counter
	unsymbolizedFrameCount metrics.Counter
}

func NewStackProcessor(symbolizer *symbolizer.Symbolizer, reg metrics.Registry) *StackProcessor {
	return &StackProcessor{
		symbolizer:             symbolizer,
		collectedFrameCount:    reg.Counter("lua.frame.collected.count"),
		unsymbolizedFrameCount: reg.Counter("lua.frame.unsymbolized.count"),
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
		loc.SetMapping().SetPath(string(profile.LuaSpecialMapping)).Finish()

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
		symbol, exists := p.symbolizer.Symbolize(&frame.SymbolKey)

		if !exists {
			p.unsymbolizedFrameCount.Inc()
			name += fmt.Sprintf("unsymbolized lua function: 0x%x", luaData.GetPtr())
		} else {
			name += symbol.Name
			filename = symbol.FileName

			// Usually scripts has `@` symbol appended at the beginning.
			// Perforator has the same symbol, removing here.
			if len(filename) != 0 && filename[0] == '@' {
				filename = symbol.FileName[1:]
			}
		}
	case LuaObjectAddressFfidC:
		// C frame

		// TODO: Try to symbolize this frame by postprocess
		name += fmt.Sprintf("function: 0x%x", luaData.GetPtr())
	default:
		// FF frame

		// TODO: Try to symbolize this frame by postprocess
		// If postprocess will fail, at least print its builtin number to find the name manually
		name += "function: builtin#" + strconv.Itoa(int(luaData.GetFfid()))
	}

	loc.AddFrame().
		SetName(name).
		SetFilename(filename).
		SetLine(int64(frame.SymbolKey.Linestart)).
		SetStartLine(int64(frame.SymbolKey.Linestart)).
		Finish()
}

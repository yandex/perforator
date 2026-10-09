package agent

import (
	"fmt"
	"strconv"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// Internal frame decoding errors, see lua_stack_walk_error at perforator/agent/collector/progs/unwinder/lua/stack/walk_error.h
var luaStackWalkErrorDescriptions = []string{
	"GCfunc is null",
	"bad function in frame",
}

const (
	LuaCFunctionId = 1
)

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
	stack *unwinder.LuaStack,
	process linux.ProcessKey,
) {
	var frames uint32
	for i := 0; i < int(stack.Len); i++ {
		frame := &stack.Frames[i]
		p.processFrame(builder, frame, process)

		frames++
	}

	p.collectedFrameCount.Add(int64(frames))
}

func (p *StackProcessor) processFrame(
	builder *profile.SampleBuilder,
	frame *unwinder.LuaFrame,
	process linux.ProcessKey,
) {
	name := "[lua] "
	filename := ""
	firstLine := int64(0)
	currentLine := int64(0)
	clearAddress := false
	var loc *profile.LocationBuilder

	switch frame.Type {
	case unwinder.LuaFrameTypeLua:
		luaFrame := frame.Value.GetLuaFrame()
		symbol, exists := p.symbolizer.Symbolize(models.Language(unwinder.LanguageLua), process, &unwinder.SymbolKey{ObjectAddr: luaFrame.ProtoAddress, Linestart: luaFrame.FirstLine})

		if !exists {
			p.unsymbolizedFrameCount.Inc()
			name += fmt.Sprintf("unsymbolized lua proto: 0x%x", luaFrame.ProtoAddress)
		} else {
			name += "<no name>"

			// Usually scripts has `@` symbol appended at the beginning.
			// Perforator has the same symbol, removing here.
			filename = symbol.FileName
			if len(filename) != 0 && filename[0] == '@' {
				filename = symbol.FileName[1:]
			}
		}

		firstLine = int64(luaFrame.FirstLine)
		currentLine = int64(luaFrame.CurrentLine)
		clearAddress = exists && (symbol.Name != "" || (filename != "" && firstLine > 0))

		loc = builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: luaFrame.ProtoAddress,
			Linestart:     luaFrame.FirstLine,
			Line:          luaFrame.CurrentLine,
			Language:      models.Language(unwinder.LanguageLua),
		})
	case unwinder.LuaFrameTypeC:
		cFrame := frame.Value.GetCFrame()

		// TODO: Try to symbolize this frame by postprocess
		if int(cFrame.Ffid) == LuaCFunctionId {
			name += fmt.Sprintf("function: 0x%x", cFrame.FunctionAddress)
		} else {
			// FF function
			name += "function: builtin#" + strconv.Itoa(int(cFrame.Ffid))
			clearAddress = true
		}

		loc = builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: cFrame.FunctionAddress,
			Linestart:     int32(cFrame.Ffid),
			Language:      models.Language(unwinder.LanguageLua),
		})
	case unwinder.LuaFrameTypeInvalid:
		invalidFrame := frame.Value.GetInvalidFrame()
		name += "<invalid lua frame>"

		if int(invalidFrame.Error) < len(luaStackWalkErrorDescriptions) {
			name += ": " + luaStackWalkErrorDescriptions[invalidFrame.Error]
		}

		loc = builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: uint64(invalidFrame.Error),
			Linestart:     0,
			Language:      models.Language(unwinder.LanguageLua),
		})
	}

	if clearAddress {
		loc.ClearAddress()
	}
	loc.SetMapping().SetPath(string(profile.LuaSpecialMapping)).Finish()
	loc.AddFrame().
		SetName(name).
		SetFilename(filename).
		SetLine(currentLine).
		SetStartLine(firstLine).
		Finish()
	loc.Finish()
}

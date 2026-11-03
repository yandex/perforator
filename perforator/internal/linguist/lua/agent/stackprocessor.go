package agent

import (
	"fmt"
	"strconv"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline/runtimewriter"
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
	line := int64(0)
	clearAddress := false
	var loc *profile.LocationBuilder

	switch frame.Type {
	case unwinder.LuaFrameTypeLua:
		luaFrame := frame.Value.GetLuaFrame()
		symbol, exists := p.symbolizer.Symbolize(models.Language(unwinder.LanguageLua), process, &luaFrame)

		if !exists {
			p.unsymbolizedFrameCount.Inc()
			name += fmt.Sprintf("unsymbolized lua function: 0x%x", luaFrame.ObjectAddr)
		} else {
			if len(symbol.Name) == 0 {
				name += "<no name>"
			} else {
				name += symbol.Name
			}

			filename = symbol.FileName

			// Lua prefixes file-backed chunk names with '@'.
			if len(filename) != 0 && filename[0] == '@' {
				filename = symbol.FileName[1:]
			}
		}

		line = int64(luaFrame.Linestart)
		clearAddress = exists && (symbol.Name != "" || (filename != "" && line > 0))

		loc = builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: luaFrame.ObjectAddr,
			Linestart:     luaFrame.Linestart,
			Language:      models.Language(unwinder.LanguageLua),
		})
	case unwinder.LuaFrameTypeC:
		cFrame := frame.Value.GetCFrame()

		if int(cFrame.Ffid) == LuaCFunctionId {
			name += fmt.Sprintf("function: 0x%x", cFrame.ObjectAddr)
		} else {
			name += "function: builtin#" + strconv.Itoa(int(cFrame.Ffid))
			clearAddress = true
		}

		loc = builder.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: cFrame.ObjectAddr,
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
		SetLine(line).
		SetStartLine(line).
		Finish()
	loc.Finish()
}

func (p *StackProcessor) ResolveFrame(process linux.ProcessKey, frame *unwinder.LuaFrame, output runtimewriter.Frame) error {
	name := "[lua] "
	filename := ""
	line := int64(0)
	clearAddress := false
	var address uint64

	switch frame.Type {
	case unwinder.LuaFrameTypeLua:
		luaFrame := frame.Value.GetLuaFrame()
		symbol, exists := p.symbolizer.Symbolize(models.Language(unwinder.LanguageLua), process, &luaFrame)

		if !exists {
			name += fmt.Sprintf("unsymbolized lua function: 0x%x", luaFrame.ObjectAddr)
		} else {
			if len(symbol.Name) == 0 {
				name += "<no name>"
			} else {
				name += symbol.Name
			}

			filename = symbol.FileName

			// Lua prefixes file-backed chunk names with '@'.
			if len(filename) != 0 && filename[0] == '@' {
				filename = symbol.FileName[1:]
			}
		}

		line = int64(luaFrame.Linestart)
		clearAddress = exists && (symbol.Name != "" || (filename != "" && line > 0))

		address = luaFrame.ObjectAddr
	case unwinder.LuaFrameTypeC:
		cFrame := frame.Value.GetCFrame()

		if int(cFrame.Ffid) == LuaCFunctionId {
			name += fmt.Sprintf("function: 0x%x", cFrame.ObjectAddr)
		} else {
			name += "function: builtin#" + strconv.Itoa(int(cFrame.Ffid))
			clearAddress = true
		}

		address = cFrame.ObjectAddr
	case unwinder.LuaFrameTypeInvalid:
		invalidFrame := frame.Value.GetInvalidFrame()
		name += "<invalid lua frame>"

		if int(invalidFrame.Error) < len(luaStackWalkErrorDescriptions) {
			name += ": " + luaStackWalkErrorDescriptions[invalidFrame.Error]
		}

		address = uint64(invalidFrame.Error)
	default:
		return fmt.Errorf("unsupported Lua frame type %d", frame.Type)
	}

	if err := output.SetMappingName(profile.LuaSpecialMapping); err != nil {
		return err
	}
	if clearAddress {
		address = 0
	}
	output.SetAddress(address)
	return output.AddInlineFunction(name, filename, int32(line), int32(line))
}

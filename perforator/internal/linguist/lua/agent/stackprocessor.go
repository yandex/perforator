package agent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/hashicorp/golang-lru/v2/expirable"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procmem"
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
	collectedNameCount     metrics.Counter
	unsymbolizedNameCount  metrics.Counter
	protos                 *expirable.LRU[luaProtoKey, *luaProto]
	strings                *expirable.LRU[luaStringKey, string]
}

func NewStackProcessor(sym *symbolizer.Symbolizer, reg metrics.Registry) *StackProcessor {
	return &StackProcessor{
		symbolizer:             sym,
		collectedFrameCount:    reg.Counter("lua.frame.collected.count"),
		unsymbolizedFrameCount: reg.Counter("lua.frame.unsymbolized.count"),
		collectedNameCount:     reg.Counter("lua.frame.name.collected.count"),
		unsymbolizedNameCount:  reg.Counter("lua.frame.name.unsymbolized.count"),
		protos:                 expirable.NewLRU[luaProtoKey, *luaProto](luaProtoCacheCapacity, nil, symbolizer.DefaultCacheTTL),
		strings:                expirable.NewLRU[luaStringKey, string](luaStringCacheCapacity, nil, symbolizer.DefaultCacheTTL),
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
		var callerFrame *unwinder.LuaFrame

		// Frames run from callee to caller. The caller's PC names callee's frame.
		if i+1 < int(stack.Len) {
			callerFrame = &stack.Frames[i+1]
		}

		p.processFrame(builder, frame, callerFrame, process)
		frames++
	}

	p.collectedFrameCount.Add(int64(frames))
}

func (p *StackProcessor) processFrame(
	builder *profile.SampleBuilder,
	frame *unwinder.LuaFrame,
	callerFrame *unwinder.LuaFrame,
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
			p.unsymbolizedNameCount.Inc()
			name += fmt.Sprintf("(unsymbolized lua proto: 0x%x)", luaFrame.ProtoAddress)
		} else {
			funcname := p.getFrameName(process, frame, callerFrame, symbol.Name)
			if funcname == "" {
				p.unsymbolizedNameCount.Inc()
				funcname = "<no name>"
			} else {
				p.collectedNameCount.Inc()
			}
			name += funcname

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
			Name:          name,
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

// Agent read/cache budgets, not LuaJIT's maximum allocation sizes.
const (
	luaProtoReadLimit      = 1 << 20
	luaStringReadLimit     = 1 << 10
	luaProtoCacheCapacity  = 256
	luaStringCacheCapacity = 1024
)

// LuaJIT layout for LJ_GC64=1, LJ_FR2=1, x86-64 (lj_obj.h).
const (
	luaProtoSize     = 104
	luaGCStrSize     = 24
	luaGCRefSize     = 8
	luaBCInsSize     = 4
	luaTValueSize    = 8
	luaGCPointerBits = 47
	luaGCMask        = 1<<luaGCPointerBits - 1

	// GCHeader.gct stores the complement of the corresponding LJ_T* tag.
	luaGCTStr               = 4
	luaGCTProto             = 7
	luaGCTFunc              = 8
	luaGCHeaderGCTOffset    = 9
	luaGCStrLengthOffset    = 20
	luaProtoFrameSizeOffset = 11
	luaProtoSizeBCOffset    = 12
	luaProtoKOffset         = 32
	luaProtoSizeKGCOffset   = 48
	luaProtoSizePTOffset    = 56
	luaProtoSizeUVOffset    = 60
	luaProtoFirstLineOffset = 72
	luaProtoNumLineOffset   = 76
	luaProtoUVInfoOffset    = 88
	luaProtoVarInfoOffset   = 96
)

const (
	luaBCRegMax             = 0xff
	luaBCAShift             = 8
	luaBCCShift             = 16
	luaBCDShift             = luaBCCShift
	luaBCBShift             = 24
	luaBCFirstInstruction   = 1 // Instruction zero is the function header.
	luaBCNoPosition         = ^uint32(0)
	luaIteratorControlSlots = 3
)

func luaBCOpcode(instruction uint32) uint8 {
	return uint8(instruction)
}

func luaBCOperandA(instruction uint32) uint32 {
	return (instruction >> luaBCAShift) & luaBCRegMax
}

func luaBCOperandC(instruction uint32) uint32 {
	return (instruction >> luaBCCShift) & luaBCRegMax
}

func luaBCOperandD(instruction uint32) uint32 {
	return instruction >> luaBCDShift
}

const (
	luaBCMov      = 18
	luaBCKnil     = 44
	luaBCUget     = 45
	luaBCGget     = 54
	luaBCTgets    = 57
	luaBCIterc    = 69
	luaBCModeDst  = 1
	luaBCModeBase = 2
	luaVarNameEnd = 0
	luaVarNameMax = 7

	// Name representation of MM_call and MM__MAX; an absent metamethod has no name.
	luaMMCall = "__call"
	luaMMMax  = ""
)

// BCDEF metadata for A operand in opcode order. Only destination/base writes are retained.
var luaBCModeAArray = [...]uint8{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 0, 0, // 0..15
	0, 0, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, // 16..31
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 2, 1, 0, 0, // 32..47
	0, 0, 0, 1, 1, 1, 1, 0, 1, 1, 1, 1, 0, 0, 0, 2, // 48..63
	0, 2, 2, 2, 2, 2, 2, 2, 2, 2, 0, 0, 0, 2, 2, 2, // 64..79
	2, 2, 2, 2, 2, 0, 0, 0, 0, 1, 1, 1, 1, 1, 1, 1, // 80..95
}

// BCDEF metadata for metamethod kind in opcode order.
var luaBCModeMetamethodArray = [...]string{
	"__lt", "__lt", "__le", "__le", "__eq", "__eq", "__eq", "__eq", // 0..7
	"__eq", "__eq", "__eq", "__eq", luaMMMax, luaMMMax, luaMMMax, luaMMMax, // 8..15
	luaMMMax, luaMMMax, luaMMMax, luaMMMax, "__unm", "__len", "__add", "__sub", // 16..23
	"__mul", "__div", "__mod", "__add", "__sub", "__mul", "__div", "__mod", // 24..31
	"__add", "__sub", "__mul", "__div", "__mod", "__pow", "__concat", luaMMMax, // 32..39
	luaMMMax, luaMMMax, luaMMMax, luaMMMax, luaMMMax, luaMMMax, luaMMMax, luaMMMax, // 40..47
	luaMMMax, luaMMMax, luaMMMax, "__gc", "__gc", "__gc", "__index", "__newindex", // 48..55
	"__index", "__index", "__index", "__index", "__newindex", "__newindex", "__newindex", "__newindex", // 56..63
	"__newindex", luaMMCall, luaMMCall, luaMMCall, luaMMCall, luaMMCall, luaMMCall, // 64..70
}

func luaBCModeA(opcode uint8) uint8 {
	return luaBCModeAArray[opcode]
}

func luaBCModeMetamethod(opcode uint8) string {
	return luaBCModeMetamethodArray[opcode]
}

var luaVarNames = [luaVarNameMax]string{
	"", "(for index)", "(for limit)", "(for step)", "(for generator)", "(for state)", "(for control)",
}

type luaProtoKey struct {
	process   linux.ProcessKey
	address   uint64
	firstLine int32
}

type luaStringKey struct {
	process linux.ProcessKey
	proto   *luaProto
	address uint64
}

// GCproto fields used for name lookup, decoded from the eBPF symbol cache.
type luaProtoFields struct {
	gct       uint8
	framesize uint8
	sizebc    uint64
	k         uint64
	sizekgc   uint64
	sizept    uint64
	sizeuv    uint8
	firstline int32
	numline   uint32
	uvinfo    uint64
	varinfo   uint64
}

func decodeLuaProtoFields(data []byte) luaProtoFields {
	return luaProtoFields{
		gct:       data[luaGCHeaderGCTOffset],
		framesize: data[luaProtoFrameSizeOffset],
		sizebc:    uint64(binary.LittleEndian.Uint32(data[luaProtoSizeBCOffset:])),
		k:         binary.LittleEndian.Uint64(data[luaProtoKOffset:]),
		sizekgc:   uint64(binary.LittleEndian.Uint32(data[luaProtoSizeKGCOffset:])),
		sizept:    uint64(binary.LittleEndian.Uint32(data[luaProtoSizePTOffset:])),
		sizeuv:    data[luaProtoSizeUVOffset],
		firstline: int32(binary.LittleEndian.Uint32(data[luaProtoFirstLineOffset:])),
		numline:   binary.LittleEndian.Uint32(data[luaProtoNumLineOffset:]),
		uvinfo:    binary.LittleEndian.Uint64(data[luaProtoUVInfoOffset:]),
		varinfo:   binary.LittleEndian.Uint64(data[luaProtoVarInfoOffset:]),
	}
}

// All GCproto arrays are colocated in this immutable snapshot. String constants
// live outside the proto and are cached separately, after being requested.
type luaProto struct {
	header   luaProtoFields
	bytecode []byte
	kgc      []byte
	uvinfo   []byte
	varinfo  []byte
}

func (p *StackProcessor) readLuaProtoFields(key luaProtoKey, rawStruct string) *luaProto {
	if len(rawStruct) != luaProtoSize {
		return nil
	}

	if proto, ok := p.protos.Get(key); ok {
		return proto
	}

	if key.address == 0 || key.address > luaGCMask-luaProtoSize {
		return nil
	}

	protoStruct := decodeLuaProtoFields([]byte(rawStruct))
	if protoStruct.gct != luaGCTProto || protoStruct.firstline != key.firstLine ||
		protoStruct.sizept < luaProtoSize || protoStruct.sizept > luaProtoReadLimit || protoStruct.sizept > luaGCMask-key.address ||
		protoStruct.sizebc == 0 || protoStruct.sizebc*luaBCInsSize > protoStruct.sizept-luaProtoSize {
		return nil
	}

	data := make([]byte, protoStruct.sizept)
	copy(data, rawStruct)
	if procmem.Read(int(key.process.Pid), uintptr(key.address+luaProtoSize), data[luaProtoSize:]) != nil {
		return nil
	}

	// MRef fields must point inside the proto allocation (or be absent).
	field := func(addr uint64) []byte {
		if addr < key.address+luaProtoSize || addr >= key.address+protoStruct.sizept {
			return nil
		}
		return data[addr-key.address:]
	}

	if protoStruct.k < key.address+luaProtoSize+protoStruct.sizebc*luaBCInsSize+protoStruct.sizekgc*luaGCRefSize || protoStruct.k > key.address+protoStruct.sizept {
		return nil
	}
	proto := &luaProto{
		header:   protoStruct,
		bytecode: data[luaProtoSize : luaProtoSize+protoStruct.sizebc*luaBCInsSize],
		kgc:      data[protoStruct.k-key.address-protoStruct.sizekgc*luaGCRefSize : protoStruct.k-key.address],
		uvinfo:   field(protoStruct.uvinfo),
		varinfo:  field(protoStruct.varinfo),
	}

	// Upvalue names end where the local-variable debug data starts.
	if len(proto.uvinfo) >= len(proto.varinfo) && len(proto.varinfo) != 0 {
		proto.uvinfo = proto.uvinfo[:len(proto.uvinfo)-len(proto.varinfo)]
	}

	p.protos.Add(key, proto)
	return proto
}

// See `lj_debug_funcname` in LuaJIT source code
func (p *StackProcessor) getFrameName(process linux.ProcessKey, currentFrame *unwinder.LuaFrame, previousFrame *unwinder.LuaFrame, protoRaw string) string {
	currentLuaFrame := currentFrame.Value.GetLuaFrame()

	if len(protoRaw) != luaProtoSize {
		return ""
	}
	header := decodeLuaProtoFields([]byte(protoRaw))
	if header.gct != luaGCTProto || header.firstline != currentLuaFrame.FirstLine {
		return ""
	}

	if currentLuaFrame.FirstLine == 0 && header.numline != 0 {
		return "in main chunk"
	}

	if previousFrame == nil || previousFrame.Type != unwinder.LuaFrameTypeLua {
		// TODO: calling function through pcall/xpcall loses function name, but there is a way to get it
		return ""
	}

	previousLuaFrame := previousFrame.Value.GetLuaFrame()
	if previousLuaFrame.BytecodeIndex == luaBCNoPosition {
		return ""
	}

	previousProtoRaw := protoRaw
	if previousLuaFrame.ProtoAddress != currentLuaFrame.ProtoAddress || previousLuaFrame.FirstLine != currentLuaFrame.FirstLine {
		callerSymbol, exists := p.symbolizer.Symbolize(models.Language(unwinder.LanguageLua), process, &unwinder.SymbolKey{ObjectAddr: previousLuaFrame.ProtoAddress, Linestart: previousLuaFrame.FirstLine})
		if !exists {
			return ""
		}
		previousProtoRaw = callerSymbol.Name
	}
	previousFrameProto := p.readLuaProtoFields(luaProtoKey{process, previousLuaFrame.ProtoAddress, previousLuaFrame.FirstLine}, previousProtoRaw)
	if previousFrameProto == nil || uint64(previousLuaFrame.BytecodeIndex)*luaBCInsSize >= uint64(len(previousFrameProto.bytecode)) {
		return ""
	}

	instruction := previousFrameProto.instruction(previousLuaFrame.BytecodeIndex)
	opcode := luaBCOpcode(instruction)
	if int(opcode) >= len(luaBCModeMetamethodArray) {
		return ""
	}

	// Metamethod
	metamethod := luaBCModeMetamethod(opcode)
	if metamethod != luaMMCall {
		return metamethod
	}

	// Regular call
	slot := luaBCOperandA(instruction)
	if opcode == luaBCIterc {
		if slot < luaIteratorControlSlots {
			return ""
		}
		slot -= luaIteratorControlSlots
	}

	return p.getSlotName(process, previousFrameProto, previousLuaFrame.BytecodeIndex, slot)
}

// See `lj_debug_slotname` in LuaJIT source code
func (p *StackProcessor) getSlotName(process linux.ProcessKey, proto *luaProto, bytecodeIndex uint32, slot uint32) string {
restart:
	if name := proto.variableName(bytecodeIndex, slot); name != "" {
		return name
	}

	for bytecodeIndex > luaBCFirstInstruction {
		bytecodeIndex--

		instruction := proto.instruction(bytecodeIndex)
		opcode, a := luaBCOpcode(instruction), luaBCOperandA(instruction)
		if int(opcode) >= len(luaBCModeAArray) {
			return ""
		}

		if luaBCModeA(opcode) == luaBCModeBase {
			if slot >= a && (opcode != luaBCKnil || slot <= luaBCOperandD(instruction)) {
				return ""
			}
		} else if luaBCModeA(opcode) == luaBCModeDst && a == slot {
			switch opcode {
			case luaBCMov:
				slot = luaBCOperandD(instruction)

				// MOV instruction is in AD format, but D can't be larger than proto framesize
				if slot >= uint32(proto.header.framesize) {
					return ""
				}

				goto restart
			case luaBCGget:
				return p.getConstant(process, proto, luaBCOperandD(instruction))
			case luaBCTgets:
				return p.getConstant(process, proto, luaBCOperandC(instruction))
			case luaBCUget:
				return proto.getUpvalueName(luaBCOperandD(instruction))
			default:
				return ""
			}
		}
	}

	return ""
}

// `proto_kgc(pt, ~idx)` with caching
func (p *StackProcessor) getConstant(process linux.ProcessKey, proto *luaProto, index uint32) string {
	// proto_kgc(pt, ~idx): collectable constants precede k, in reverse order.
	if uint64(index) >= uint64(len(proto.kgc))/luaGCRefSize {
		return ""
	}

	address := binary.LittleEndian.Uint64(proto.kgc[uint64(len(proto.kgc))-(uint64(index)+1)*luaGCRefSize:])

	key := luaStringKey{process, proto, address}
	if name, ok := p.strings.Get(key); ok {
		return name
	}

	if address == 0 || address > luaGCMask-luaGCStrSize {
		return ""
	}

	// GCstr, followed by its payload.
	var gcStr [luaGCStrSize]byte
	if procmem.Read(int(process.Pid), uintptr(address), gcStr[:]) != nil || gcStr[luaGCHeaderGCTOffset] != luaGCTStr {
		return ""
	}

	length := binary.LittleEndian.Uint32(gcStr[luaGCStrLengthOffset:])
	if length > luaStringReadLimit || uint64(length) > luaGCMask-address-luaGCStrSize {
		return ""
	}

	data := make([]byte, length)
	if procmem.Read(int(process.Pid), uintptr(address+luaGCStrSize), data) != nil {
		return ""
	}

	// LuaJIT exposes names as C strings, even if a string constant contains NUL.
	if end := bytes.IndexByte(data, 0); end >= 0 {
		data = data[:end]
	}

	name := string(data)
	p.strings.Add(key, name)
	return name
}

func (proto *luaProto) instruction(bytecodeIndex uint32) uint32 {
	return binary.LittleEndian.Uint32(proto.bytecode[uint64(bytecodeIndex)*luaBCInsSize:])
}

// See `lj_debug_uvname` in LuaJIT soucre code
func (proto *luaProto) getUpvalueName(index uint32) string {
	names := proto.uvinfo

	if index >= uint32(proto.header.sizeuv) {
		return ""
	}

	for i := uint32(0); i <= index; i++ {
		end := bytes.IndexByte(names, 0)
		if end < 0 {
			return ""
		}

		if i == index {
			return string(names[:end])
		}

		names = names[end+1:]
	}

	return ""
}

// See `debug_varname` in LuaJIT source code
func (proto *luaProto) variableName(bytecodeIndex uint32, slot uint32) string {
	variableInfo, lastpc := proto.varinfo, uint64(0)

	for len(variableInfo) != 0 && variableInfo[0] != luaVarNameEnd {
		vn := variableInfo[0]
		var name []byte

		if int(vn) < luaVarNameMax {
			variableInfo = variableInfo[1:]
		} else {
			end := bytes.IndexByte(variableInfo, 0)
			if end < 0 {
				return ""
			}

			name, variableInfo = variableInfo[:end], variableInfo[end+1:]
		}

		delta, n := binary.Uvarint(variableInfo)
		if n <= 0 || delta > uint64(^uint32(0))-lastpc {
			return ""
		}

		lastpc += delta
		if lastpc > uint64(bytecodeIndex) {
			break
		}

		variableInfo = variableInfo[n:]
		extent, n := binary.Uvarint(variableInfo)
		if n <= 0 || extent > uint64(^uint32(0))-lastpc {
			return ""
		}
		variableInfo = variableInfo[n:]

		if uint64(bytecodeIndex) < lastpc+extent {
			if slot == 0 {
				if int(vn) < luaVarNameMax {
					return luaVarNames[vn]
				}
				return string(name)
			}
			slot--
		}
	}

	return ""
}

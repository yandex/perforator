package agent

import (
	"encoding/binary"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// Offsets within the unwinder's Lua frame union (unwinder/lua/types.h).
const (
	testLuaFrameFirstLineOffset = 8
	testLuaFramePCOffset        = 16
)

// Frame size and additional opcode values used only for regression fixtures.
const (
	testLuaFrameSize = 4
	testLuaBCCall    = 66
	testLuaBCFnew    = 51
	testLuaBCKshort  = 41
	testLuaBCAddvv   = 32
	testLuaBCIseqv   = 4
)

func luaAD(op, a, d uint32) uint32 { return op | a<<luaBCAShift | d<<luaBCDShift }

type luaTestSymbolSource map[unwinder.InterpreterSymbolKey]unwinder.Symbol

func (s luaTestSymbolSource) SymbolizeInterpreter(language models.Language, process linux.ProcessKey, key *unwinder.SymbolKey) (unwinder.Symbol, bool) {
	symbol, ok := s[unwinder.InterpreterSymbolKey{SymbolKey: *key, Pid: uint32(process.Pid), ProcessStarttime: process.ProcessStartTime, Language: uint8(language)}]
	return symbol, ok
}

func (s luaTestSymbolSource) add(process linux.ProcessKey, data []byte, frame *unwinder.LuaFrame) {
	f := frame.Value.GetLuaFrame()
	symbol := unwinder.Symbol{CodepointSize: 1, NameLength: uint8(len(data))}
	copy(symbol.Data[:], data)
	s[unwinder.InterpreterSymbolKey{
		SymbolKey: unwinder.SymbolKey{ObjectAddr: f.ProtoAddress, Linestart: f.FirstLine},
		Pid:       uint32(process.Pid), ProcessStarttime: process.ProcessStartTime, Language: uint8(unwinder.LanguageLua),
	}] = symbol
}

func newLuaNameTestProcessor(t *testing.T) (*StackProcessor, luaTestSymbolSource) {
	t.Helper()
	source := make(luaTestSymbolSource)
	sym, err := symbolizer.NewLuaSymbolizer(&symbolizer.SymbolizerConfig{}, source, nop.Registry{})
	if err != nil {
		t.Fatal(err)
	}
	return NewStackProcessor(sym, nop.Registry{}), source
}

func luaTestFrameName(p *StackProcessor, process linux.ProcessKey, current, caller *unwinder.LuaFrame) string {
	f := current.Value.GetLuaFrame()
	symbol, exists := p.symbolizer.Symbolize(models.Language(unwinder.LanguageLua), process, &unwinder.SymbolKey{ObjectAddr: f.ProtoAddress, Linestart: f.FirstLine})
	if !exists {
		return ""
	}
	return p.getFrameName(process, current, caller, symbol.Name)
}

// Synthetic GC64 objects are read through procmem, just like target memory.
func luaNameFixture(code []uint32, uvinfo, varinfo []byte, constants ...string) ([]byte, [][]byte, *unwinder.LuaFrame) {
	k := (luaProtoSize + len(code)*luaBCInsSize + len(constants)*luaGCRefSize + luaTValueSize - 1) &^ (luaTValueSize - 1)
	data := make([]byte, k+len(uvinfo)+len(varinfo))
	address := uint64(uintptr(unsafe.Pointer(&data[0])))
	data[luaGCHeaderGCTOffset] = luaGCTProto // GCproto.gct.
	data[luaProtoFrameSizeOffset] = testLuaFrameSize
	binary.LittleEndian.PutUint32(data[luaProtoSizeBCOffset:], uint32(len(code)))
	binary.LittleEndian.PutUint64(data[luaProtoKOffset:], address+uint64(k))
	binary.LittleEndian.PutUint32(data[luaProtoSizeKGCOffset:], uint32(len(constants)))
	binary.LittleEndian.PutUint32(data[luaProtoSizePTOffset:], uint32(len(data)))
	for _, b := range uvinfo {
		if b == 0 {
			data[luaProtoSizeUVOffset]++
		}
	}
	binary.LittleEndian.PutUint32(data[luaProtoFirstLineOffset:], 10)
	if len(uvinfo) != 0 {
		binary.LittleEndian.PutUint64(data[luaProtoUVInfoOffset:], address+uint64(k))
		copy(data[k:], uvinfo)
	}
	if len(varinfo) != 0 {
		binary.LittleEndian.PutUint64(data[luaProtoVarInfoOffset:], address+uint64(k+len(uvinfo)))
		copy(data[k+len(uvinfo):], varinfo)
	}
	for i, ins := range code {
		binary.LittleEndian.PutUint32(data[luaProtoSize+i*luaBCInsSize:], ins)
	}
	strings := make([][]byte, len(constants))
	for i, name := range constants {
		str := make([]byte, luaGCStrSize+len(name)+1)
		str[luaGCHeaderGCTOffset] = luaGCTStr // GCstr.gct.
		binary.LittleEndian.PutUint32(str[luaGCStrLengthOffset:], uint32(len(name)))
		copy(str[luaGCStrSize:], name)
		binary.LittleEndian.PutUint64(data[k-(i+1)*luaGCRefSize:], uint64(uintptr(unsafe.Pointer(&str[0]))))
		strings[i] = str
	}
	frame := &unwinder.LuaFrame{Type: unwinder.LuaFrameTypeLua}
	binary.LittleEndian.PutUint64(frame.Value.UnionBuf[:], address)
	binary.LittleEndian.PutUint32(frame.Value.UnionBuf[testLuaFrameFirstLineOffset:], 10)
	binary.LittleEndian.PutUint32(frame.Value.UnionBuf[testLuaFramePCOffset:], uint32(len(code)-1))
	return data, strings, frame
}

func TestLuaDebugFuncname(t *testing.T) {
	process := linux.ProcessKey{Pid: linux.CurrentNamespacePID(os.Getpid()), ProcessStartTime: 1}
	for _, tc := range []struct {
		name      string
		code      []uint32
		uv, vars  string
		constants []string
		want      string
	}{
		{"local", []uint32{0, 0, luaAD(testLuaBCCall, 0, 0)}, "", "localfn\x00\x01\x03\x00", nil, "localfn"},
		{"local scope", []uint32{0, 0, luaAD(testLuaBCCall, 0, 0)}, "", "expired\x00\x00\x01active\x00\x01\x03\x00", nil, "active"},
		{"local slot", []uint32{0, 0, luaAD(testLuaBCCall, 1, 0)}, "", "first\x00\x01\x03second\x00\x00\x03\x00", nil, "second"},
		{"ULEB128 extent", []uint32{0, 0, luaAD(testLuaBCCall, 0, 0)}, "", "long\x00\x01\x80\x01\x00", nil, "long"},
		{"global", []uint32{0, luaAD(luaBCGget, 0, 1), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"unused", "globalfn"}, "globalfn"},
		{"MOV chain", []uint32{0, luaAD(luaBCGget, 0, 0), luaAD(luaBCMov, 1, 0), luaAD(luaBCMov, 2, 1), luaAD(testLuaBCCall, 2, 0)}, "", "", []string{"alias"}, "alias"},
		{"MOV local", []uint32{0, luaAD(luaBCMov, 1, 0), luaAD(testLuaBCCall, 1, 0)}, "", "original\x00\x00\x02\x00", nil, "original"},
		{"MOV last slot", []uint32{0, luaAD(luaBCGget, testLuaFrameSize-1, 0), luaAD(luaBCMov, 0, testLuaFrameSize-1), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"lastslot"}, "lastslot"},
		{"MOV outside frame", []uint32{0, luaAD(luaBCGget, testLuaFrameSize, 0), luaAD(luaBCMov, 0, testLuaFrameSize), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"invalidslot"}, ""},
		{"field", []uint32{0, luaAD(luaBCTgets, 0, 1) | 2<<luaBCBShift, luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"unused", "fieldfn"}, "fieldfn"},
		{"method FR2", []uint32{0, luaAD(luaBCMov, 2, 3), luaAD(luaBCTgets, 0, 0) | 3<<luaBCBShift, luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"methodfn"}, "methodfn"},
		{"upvalue", []uint32{0, luaAD(luaBCUget, 0, 1), luaAD(testLuaBCCall, 0, 0)}, "unused\x00upvaluefn\x00", "", nil, "upvaluefn"},
		{"ITERC", []uint32{0, luaAD(luaBCGget, 0, 0), luaAD(luaBCIterc, 3, 0)}, "", "", []string{"iterator"}, "iterator"},
		{"fixed varname", []uint32{0, 0, luaAD(luaBCIterc, 3, 0)}, "", "\x04\x01\x03\x00", nil, "(for generator)"},
		{"lambda", []uint32{0, luaAD(testLuaBCFnew, 0, 0), luaAD(testLuaBCCall, 0, 0)}, "", "", nil, ""},
		{"clobbered base", []uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCCall, 0, 0), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"old"}, ""},
		{"KNIL outside slot", []uint32{0, luaAD(luaBCGget, 2, 0), luaAD(luaBCKnil, 0, 1), luaAD(testLuaBCCall, 2, 0)}, "", "", []string{"kept"}, "kept"},
		{"KNIL inside slot", []uint32{0, luaAD(luaBCGget, 2, 0), luaAD(luaBCKnil, 0, 2), luaAD(testLuaBCCall, 2, 0)}, "", "", []string{"old"}, ""},
		{"clobbered destination", []uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCKshort, 0, 0), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"old"}, ""},
		{"stripped upvalue", []uint32{0, luaAD(luaBCUget, 0, 0), luaAD(testLuaBCCall, 0, 0)}, "", "", nil, ""},
		{"invalid constant", []uint32{0, luaAD(luaBCGget, 0, 1), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"only"}, ""},
		{"invalid upvalue", []uint32{0, luaAD(luaBCUget, 0, 1), luaAD(testLuaBCCall, 0, 0)}, "only\x00", "", nil, ""},
		{"invalid ITERC slot", []uint32{0, luaAD(luaBCIterc, 2, 0)}, "", "", nil, ""},
		{"invalid opcode", []uint32{0, luaBCRegMax}, "", "", nil, ""},
		{"truncated varinfo", []uint32{0, luaAD(testLuaBCCall, 0, 0)}, "", "broken\x00\x80", nil, ""},
		{"truncated varname", []uint32{0, luaAD(testLuaBCCall, 0, 0)}, "", "broken", nil, ""},
		{"non-call", []uint32{0, luaAD(testLuaBCKshort, 0, 0)}, "", "", nil, ""},
		{"metamethod add", []uint32{0, luaAD(testLuaBCAddvv, 0, 0)}, "", "", nil, "__add"},
		{"metamethod index", []uint32{0, luaAD(luaBCTgets, 0, 0)}, "", "", nil, "__index"},
		{"metamethod eq", []uint32{0, luaAD(testLuaBCIseqv, 0, 0)}, "", "", nil, "__eq"},
		{"embedded NUL", []uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCCall, 0, 0)}, "", "", []string{"name\x00suffix"}, "name"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, source := newLuaNameTestProcessor(t)
			data, strings, frame := luaNameFixture(tc.code, []byte(tc.uv), []byte(tc.vars), tc.constants...)
			source.add(process, data[:luaProtoSize], frame)
			data[luaGCHeaderGCTOffset] = 0 // Only the captured header is valid.
			if got := luaTestFrameName(p, process, frame, frame); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			runtime.KeepAlive(data)
			runtime.KeepAlive(strings)
		})
	}

	t.Run("cache and process lifetime", func(t *testing.T) {
		p, source := newLuaNameTestProcessor(t)
		data, strings, frame := luaNameFixture([]uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCCall, 0, 0)}, nil, nil, "cached")
		source.add(process, data[:luaProtoSize], frame)
		if got := luaTestFrameName(p, process, frame, frame); got != "cached" {
			t.Fatalf("first read: %q", got)
		}
		data[luaGCHeaderGCTOffset], strings[0][luaGCHeaderGCTOffset] = 0, 0 // Further reads would fail type validation.
		if got := luaTestFrameName(p, process, frame, frame); got != "cached" {
			t.Fatalf("cache hit: %q", got)
		}
		other := process
		other.ProcessStartTime++
		if got := luaTestFrameName(p, other, frame, frame); got != "" {
			t.Fatalf("reused PID shared cached proto: %q", got)
		}
		data[luaGCHeaderGCTOffset] = luaGCTProto
		source.add(other, data[:luaProtoSize], frame)
		if got := luaTestFrameName(p, other, frame, frame); got != "" {
			t.Fatalf("reused PID shared cached string: %q", got)
		}
		binary.LittleEndian.PutUint32(data[luaProtoSizePTOffset:], luaProtoReadLimit+1)
		p, source = newLuaNameTestProcessor(t)
		source.add(process, data[:luaProtoSize], frame)
		if got := luaTestFrameName(p, process, frame, frame); got != "" {
			t.Fatalf("oversized proto: %q", got)
		}
		runtime.KeepAlive(data)
		runtime.KeepAlive(strings)
	})

	t.Run("invalid caller and PC", func(t *testing.T) {
		p, source := newLuaNameTestProcessor(t)
		data, strings, frame := luaNameFixture([]uint32{0, luaAD(testLuaBCAddvv, 0, 0)}, nil, nil)
		source.add(process, data[:luaProtoSize], frame)
		for _, pc := range []uint32{luaBCNoPosition, 2} {
			binary.LittleEndian.PutUint32(frame.Value.UnionBuf[testLuaFramePCOffset:], pc)
			if got := luaTestFrameName(p, process, frame, frame); got != "" {
				t.Fatalf("invalid PC %d: %q", pc, got)
			}
		}
		for _, caller := range []*unwinder.LuaFrame{nil, {Type: unwinder.LuaFrameTypeC}, {Type: unwinder.LuaFrameTypeInvalid}, {Type: unwinder.LuaFrameTypeLua}} {
			if got := luaTestFrameName(p, process, frame, caller); got != "" {
				t.Fatalf("invalid caller: %q", got)
			}
		}
		runtime.KeepAlive(data)
		runtime.KeepAlive(strings)
	})
}

func TestLuaStringCacheProtoLifetime(t *testing.T) {
	process := linux.ProcessKey{Pid: linux.CurrentNamespacePID(os.Getpid()), ProcessStartTime: 1}
	p, source := newLuaNameTestProcessor(t)
	data, constants, frame := luaNameFixture([]uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCCall, 0, 0)}, nil, nil, "oldname")
	source.add(process, data[:luaProtoSize], frame)
	if got := luaTestFrameName(p, process, frame, frame); got != "oldname" {
		t.Fatalf("first lifetime: got %q, want oldname", got)
	}

	// Reuse both addresses; a different first line identifies a new proto.
	copy(constants[0][luaGCStrSize:], "newname")
	binary.LittleEndian.PutUint32(data[luaProtoFirstLineOffset:], 20)
	binary.LittleEndian.PutUint32(frame.Value.UnionBuf[testLuaFrameFirstLineOffset:], 20)
	source.add(process, data[:luaProtoSize], frame)
	if got := luaTestFrameName(p, process, frame, frame); got != "newname" {
		t.Fatalf("new lifetime: got %q, want newname", got)
	}
	runtime.KeepAlive(data)
	runtime.KeepAlive(constants)
}

func TestLuaVariableNameAllocations(t *testing.T) {
	proto := &luaProto{
		varinfo: []byte(strings.Repeat("expired_local\x00\x01\x01", 64) + "active_local\x00\x01\x03\x00"),
	}
	for _, tc := range []struct {
		name      string
		slot      uint32
		want      string
		maxAllocs float64
	}{
		{"match", 0, "active_local", 1},
		{"missing", 1, "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var got string
			allocs := testing.AllocsPerRun(100, func() {
				got = proto.variableName(66, tc.slot)
			})
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if allocs > tc.maxAllocs {
				t.Fatalf("got %g allocations, want at most %g", allocs, tc.maxAllocs)
			}
		})
	}
}

func TestLuaMainChunkName(t *testing.T) {
	process := linux.ProcessKey{Pid: linux.CurrentNamespacePID(os.Getpid()), ProcessStartTime: 1}
	for _, tc := range []struct {
		name      string
		firstline uint32
		numline   uint32
		caller    bool
		want      string
	}{
		{"main without caller", 0, 2, false, "in main chunk"},
		{"main with caller", 0, 2, true, "in main chunk"},
		{"regular function", 10, 2, true, "calledfn"},
		{"regular without lines", 10, 0, true, "calledfn"},
		{"stripped function", 0, 0, true, "calledfn"},
		{"stripped without caller", 0, 0, false, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, source := newLuaNameTestProcessor(t)
			data, _, current := luaNameFixture([]uint32{0, 0}, nil, nil)
			binary.LittleEndian.PutUint32(data[luaProtoFirstLineOffset:], tc.firstline)
			binary.LittleEndian.PutUint32(data[luaProtoNumLineOffset:], tc.numline)
			binary.LittleEndian.PutUint32(current.Value.UnionBuf[testLuaFrameFirstLineOffset:], tc.firstline)
			// Main-chunk naming does not require a valid current PC.
			binary.LittleEndian.PutUint32(current.Value.UnionBuf[testLuaFramePCOffset:], luaBCNoPosition)
			callerData, strings, caller := luaNameFixture([]uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCCall, 0, 0)}, nil, nil, "calledfn")
			source.add(process, data[:luaProtoSize], current)
			source.add(process, callerData[:luaProtoSize], caller)
			if !tc.caller {
				caller = nil
			}
			if got := luaTestFrameName(p, process, current, caller); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			runtime.KeepAlive(data)
			runtime.KeepAlive(callerData)
			runtime.KeepAlive(strings)
		})
	}
}

func TestLuaProtoRequiresCapturedHeader(t *testing.T) {
	process := linux.ProcessKey{Pid: linux.CurrentNamespacePID(os.Getpid())}
	data, _, frame := luaNameFixture([]uint32{0, luaAD(testLuaBCAddvv, 0, 0)}, nil, nil)
	for _, length := range []int{0, luaProtoSize - 1, luaProtoSize + 1} {
		p, source := newLuaNameTestProcessor(t)
		if length != 0 {
			source.add(process, data[:length], frame)
		}
		if got := luaTestFrameName(p, process, frame, frame); got != "" {
			t.Fatalf("header length %d: read proto from memory instead of rejecting symbol: %q", length, got)
		}
	}
	runtime.KeepAlive(data)
}

func TestLuaProcessFrameSkipsUnsymbolizedName(t *testing.T) {
	process := linux.ProcessKey{Pid: linux.CurrentNamespacePID(os.Getpid())}
	p, source := newLuaNameTestProcessor(t)
	data, _, current := luaNameFixture([]uint32{0, 0}, nil, nil)
	callerData, strings, caller := luaNameFixture([]uint32{0, luaAD(luaBCGget, 0, 0), luaAD(testLuaBCCall, 0, 0)}, nil, nil, "calledfn")
	source.add(process, callerData[:luaProtoSize], caller)
	// A valid caller can name the function, but its own symbol is missing.
	b := profile.NewBuilder().AddSampleType("cpu", "cycles")
	sample := b.Add(process).AddValue(1)
	p.processFrame(sample, current, caller, process)
	sample.Finish()
	result := b.FinishRaw()
	want := fmt.Sprintf("[lua] (unsymbolized lua proto: 0x%x)", current.Value.GetLuaFrame().ProtoAddress)
	if got := result.Location[0].Line[0].Function.Name; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if p.protos.Len() != 0 || p.strings.Len() != 0 {
		t.Fatal("unsymbolized frame attempted function-name lookup")
	}
	runtime.KeepAlive(data)
	runtime.KeepAlive(callerData)
	runtime.KeepAlive(strings)
}

func TestLuaLocationsPreserveCallerNames(t *testing.T) {
	b := profile.NewBuilder().AddSampleType("cpu", "cycles")
	for _, name := range []string{"first", "alias", "first"} {
		sample := b.Add(linux.ProcessKey{Pid: 1}).AddValue(1)
		loc := sample.AddInterpreterLocation(&profile.InterpreterLocationKey{
			ObjectAddress: 0xabc,
			Linestart:     10,
			Line:          12,
			Language:      models.Language(unwinder.LanguageLua),
			Name:          name,
		})
		loc.AddFrame().SetName(name).Finish()
		loc.Finish().Finish()
	}
	p := b.FinishRaw()
	if len(p.Location) != 2 || p.Sample[0].Location[0] != p.Sample[2].Location[0] || p.Sample[1].Location[0].Line[0].Function.Name != "alias" {
		t.Fatal("locations merged different caller names or failed to deduplicate the same name")
	}
}

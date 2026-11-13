package lua

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	pprof "github.com/google/pprof/profile"

	collectorprofile "github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/pkg/profile/bundle"
)

// Inputs and expected outputs in these tests run root to leaf.
func mergeFixture(lua, native []string, kernel bool, base uint64) *pprof.Profile {
	nativeMapping := &pprof.Mapping{ID: 1, Start: base, Limit: base + 0x10000, File: "luajit", BuildID: "build"}
	luaMapping := &pprof.Mapping{ID: 2, File: string(collectorprofile.LuaSpecialMapping)}
	kernelMapping := &pprof.Mapping{ID: 3, File: string(collectorprofile.KernelSpecialMapping)}
	p := &pprof.Profile{SampleType: []*pprof.ValueType{{Type: "cpu", Unit: "nanoseconds"}}, PeriodType: &pprof.ValueType{}, Mapping: []*pprof.Mapping{nativeMapping, luaMapping, kernelMapping}}
	add := func(name, system string, mapping *pprof.Mapping, address uint64) *pprof.Location {
		fn := &pprof.Function{ID: uint64(len(p.Function) + 1), Name: name, SystemName: system}
		loc := &pprof.Location{ID: uint64(len(p.Location) + 1), Address: address, Mapping: mapping, Line: []pprof.Line{{Function: fn}}}
		p.Function = append(p.Function, fn)
		p.Location = append(p.Location, loc)
		return loc
	}
	header := add("header", VMInfoSystemName, nativeMapping, base+0x1000)
	header.Line[0].Column = 0x1000
	locations := []*pprof.Location{header}
	for _, name := range slices.Backward(lua) {
		system := ""
		switch name {
		case "C", "D":
			system = NativeCSystemName
		case "FF":
			system = ASMFFSystemName
		case "P":
			system = ProtectedFFSystemName
		}
		locations = append(locations, add(name, system, luaMapping, 0))
	}
	if kernel {
		locations = append(locations, add("kernel", "", kernelMapping, 0))
	}
	for i := len(native) - 1; i >= 0; i-- {
		address := base + 0x3000 + uint64(i)*16
		switch {
		case strings.HasPrefix(native[i], "vm"):
			address = base + 0x1000 + uint64(i)*16
		case native[i] == "ffi trampoline":
			header.Line[0].Function.StartLine = 0x100
			address = base + 0x1f00 // First byte of the FFI tail.
		}
		locations = append(locations, add(native[i], "", nativeMapping, address))
	}
	p.Sample = []*pprof.Sample{{Location: locations, Value: []int64{1}}}
	return p
}

func rootNames(sample *pprof.Sample) []string {
	names := make([]string, 0, len(sample.Location))
	for _, loc := range slices.Backward(sample.Location) {
		names = append(names, loc.Line[0].Function.Name)
	}
	return names
}

func assertNoHeader(t *testing.T, p *pprof.Profile) {
	t.Helper()
	for _, loc := range p.Location {
		if hasSystemName(loc, VMInfoSystemName) {
			t.Fatal("VM metadata left in location table")
		}
	}
	for _, sample := range p.Sample {
		for _, loc := range sample.Location {
			if hasSystemName(loc, VMInfoSystemName) {
				t.Fatal("VM metadata left in sample")
			}
		}
	}
	if err := p.CheckValid(); err != nil {
		t.Fatal(err)
	}
}

func TestPostprocess(t *testing.T) {
	for _, tc := range []struct {
		name        string
		lua, native []string
		want        []string
		light       bool
	}{
		{"only C frames", []string{"C", "D"}, []string{"main", "vm0", "C", "lua_call", "vm1", "D"}, []string{"main", "C", "lua_call", "D", "kernel"}, false},
		{"C-only fallback", []string{"C", "D"}, []string{"main", "helper"}, []string{"main", "helper", "C", "D", "kernel"}, true},
		{"C entry before Lua", []string{"C", "B"}, []string{"main", "vm0", "C", "vm1"}, []string{"main", "C", "B", "kernel"}, false},
		{"VM helper with lost root", []string{"A"}, []string{"vmhelper"}, []string{"vmhelper", "A", "kernel"}, true},
		{"pure Lua", []string{"A", "B"}, []string{"main", "lua_pcall", "vm"}, []string{"main", "lua_pcall", "A", "B", "kernel"}, false},
		{"interleaved", []string{"A", "C", "B"}, []string{"main", "vm0", "C", "lua_pcall", "vm1"}, []string{"main", "A", "C", "lua_pcall", "B", "kernel"}, false},
		{"repeated C", []string{"A", "C", "B", "C", "B"}, []string{"main", "vm0", "C", "vm1", "C", "vm2"}, []string{"main", "A", "C", "B", "C", "B", "kernel"}, false},
		{"protected FF", []string{"A", "P", "B"}, []string{"main", "vm"}, []string{"main", "A", "B", "kernel"}, false},
		{"ASM FF", []string{"A", "FF"}, []string{"main", "vm"}, []string{"main", "A", "vm", "kernel"}, false},
		{"only ASM FF", []string{"FF"}, []string{"main", "vm"}, []string{"main", "vm", "kernel"}, false},
		{"older ASM FF", []string{"A", "FF", "B"}, []string{"main", "vm"}, []string{"main", "A", "FF", "B", "kernel"}, false},
		{"native helper", []string{"A", "FF"}, []string{"main", "vm", "helper"}, []string{"main", "A", "vm", "helper", "kernel"}, false},
		{"invalid Lua", []string{"A", "invalid"}, []string{"main", "vm"}, []string{"main", "A", "invalid", "kernel"}, false},
		{"enter C", []string{"A", "C"}, []string{"main", "vm"}, []string{"main", "A", "kernel"}, false},
		{"inside C", []string{"A", "C"}, []string{"main", "vm", "C"}, []string{"main", "A", "C", "kernel"}, false},
		{"enter Lua", []string{"A", "C"}, []string{"main", "vm0", "C", "vm1"}, []string{"main", "A", "C", "vm1", "kernel"}, false},
		{"extra Lua leaf", []string{"A", "C", "B"}, []string{"main", "vm", "C"}, []string{"main", "A", "C", "B", "kernel"}, false},
		{"unmatched ASM FF leaf", []string{"A", "C", "FF"}, []string{"main", "vm", "C"}, []string{"main", "A", "C", "FF", "kernel"}, false},
		{"native root lost", []string{"A", "C", "B"}, []string{"vm"}, []string{"vm", "A", "C", "B", "kernel"}, true},
		{"Lua root lost", []string{"B"}, []string{"main", "vm0", "C", "vm1"}, []string{"main", "vm0", "C", "vm1", "B", "kernel"}, true},
		{"internal mismatch", []string{"A", "C", "B", "D", "E"}, []string{"main", "vm0", "C", "vm1"}, []string{"main", "vm0", "C", "vm1", "A", "C", "B", "D", "E", "kernel"}, true},
		{"adjacent VM", []string{"A"}, []string{"main", "vm0", "vm1"}, []string{"main", "vm0", "vm1", "A", "kernel"}, true},
		{"no VM", []string{"A", "C", "FF"}, []string{"main", "helper"}, []string{"main", "helper", "A", "C", "FF", "kernel"}, true},
		{"no native", []string{"A", "P", "B"}, nil, []string{"A", "P", "B", "kernel"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := mergeFixture(tc.lua, tc.native, true, 0x100000)
			res := Postprocess(p, true)
			if got := rootNames(p.Sample[0]); !slices.Equal(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			if res.UnmergedStacksCount != boolInt(tc.light) || res.MergedStacksCount != boolInt(!tc.light) {
				t.Fatalf("unexpected results: %+v", res)
			}
			assertNoHeader(t, p)
		})
	}
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func TestHeaderGuardsAndDisabledMerging(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*pprof.Profile)
	}{
		{"Lua truncation", func(p *pprof.Profile) { p.Sample[0].Location[0].Line[0].Line = 2 }},
		{"Lua failure", func(p *pprof.Profile) { p.Sample[0].Location[0].Line[0].Line = 1 }},
		{"unknown flags", func(p *pprof.Profile) { p.Sample[0].Location[0].Line[0].Line = 4 }},
		{"Lua failure and truncation", func(p *pprof.Profile) { p.Sample[0].Location[0].Line[0].Line = 3 }},
		{"invalid FFI tail size", func(p *pprof.Profile) {
			p.Sample[0].Location[0].Line[0].Function.StartLine = 0x1000
		}},
		{"invalid span", func(p *pprof.Profile) { p.Sample[0].Location[0].Line[0].Column = -1 }},
		{"range outside mapping", func(p *pprof.Profile) {
			p.Sample[0].Location[0].Mapping = p.Mapping[0]
			p.Sample[0].Location[0].Line[0].Column = 0x100000
		}},
		{"VM address outside range", func(p *pprof.Profile) {
			p.Sample[0].Location[len(p.Sample[0].Location)-2].Address = 0x200000
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := mergeFixture([]string{"A", "C"}, []string{"main", "vm"}, false, 0x100000)
			tc.mutate(p)
			res := Postprocess(p, true)
			if res.UnmergedStacksCount != 1 || !slices.Equal(rootNames(p.Sample[0]), []string{"main", "vm", "A", "C"}) {
				t.Fatalf("results %+v; stack %v", res, rootNames(p.Sample[0]))
			}
			assertNoHeader(t, p)
		})
	}
	p := mergeFixture([]string{"A", "C", "B"}, []string{"main", "vm"}, false, 0x100000)
	want := rootNames(p.Sample[0])
	want = want[:len(want)-1]
	if res := Postprocess(p, false); res != (PostProcessResults{}) || !slices.Equal(rootNames(p.Sample[0]), want) {
		t.Fatal("disabled merging changed stacks")
	}
	assertNoHeader(t, p)
	// Repeated processing and samples without Lua metadata stay unchanged.
	Postprocess(p, true)
	if !slices.Equal(rootNames(p.Sample[0]), want) {
		t.Fatal("profile without header changed")
	}
}

func TestHeaderSurvivesProfileConversions(t *testing.T) {
	for _, format := range []string{"pprof merge", "yaprof"} {
		t.Run(format, func(t *testing.T) {
			p, err := pprof.Merge([]*pprof.Profile{
				mergeFixture([]string{"A", "C", "B", "FF"}, []string{"main", "vm0", "lj_ccall_func", "ffi trampoline", "qsort", "vm1", "lj_strfmt_num"}, true, 0x100000),
				mergeFixture([]string{"A", "C", "B", "FF"}, []string{"main", "vm0", "lj_ccall_func", "ffi trampoline", "qsort", "vm1", "lj_strfmt_num"}, true, 0x800000),
			})
			if err != nil {
				t.Fatal(err)
			}
			if format == "yaprof" {
				var buf bytes.Buffer
				if err := p.WriteUncompressed(&buf); err != nil {
					t.Fatal(err)
				}
				yaprof, err := bundle.NewPprofBundle(buf.Bytes()).GetOrConvertYaprof()
				if err != nil {
					t.Fatal(err)
				}
				data, err := bundle.NewYaprofBundle(yaprof).GetOrConvertPprof()
				if err != nil {
					t.Fatal(err)
				}
				p, err = pprof.ParseData(data)
				if err != nil {
					t.Fatal(err)
				}
			}
			for _, loc := range p.Location {
				if loc.Mapping != nil && loc.Mapping.File == "luajit" && !hasSystemName(loc, VMInfoSystemName) {
					loc.Line = nil
				}
			}
			res := Postprocess(p, true)
			if res.MergedStacksCount == 0 || res.UnmergedStacksCount != 0 {
				t.Fatalf("%+v", res)
			}
			for _, s := range p.Sample {
				// Native addresses relative to the mapping identify the unsymbolized frames.
				var got []string
				for _, loc := range slices.Backward(s.Location) {
					if len(loc.Line) == 0 {
						got = append(got, fmt.Sprintf("%#x", loc.Address-loc.Mapping.Start))
					} else {
						got = append(got, loc.Line[0].Function.Name)
					}
				}
				want := []string{"0x3000", "A", "0x3020", "0x1f00", "0x3040", "B", "0x1050", "0x3060", "kernel"}
				if !slices.Equal(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
			assertNoHeader(t, p)
		})
	}
}

func TestNativeFailureAfterRecoveringVMBoundaries(t *testing.T) {
	p := mergeFixture([]string{"C", "chunk"}, []string{
		"unmapped root", "_start", "__libc_start_main", "main", "lua_cpcall",
		"vm0", "pmain", "handle_script", "docall", "lua_pcall", "vm1",
	}, false, 0x100000)
	p.Sample[0].Location[len(p.Sample[0].Location)-1].Mapping = nil
	res := Postprocess(p, true)
	if res.MergedStacksCount != 1 || res.UnmergedStacksCount != 0 {
		t.Fatalf("complete VM/C chain rejected: %+v", res)
	}
	want := []string{
		"unmapped root", "_start", "__libc_start_main", "main", "lua_cpcall",
		"pmain", "handle_script", "docall", "lua_pcall", "chunk",
	}
	if got := rootNames(p.Sample[0]); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	assertNoHeader(t, p)
}

func TestASMFFPreservesNativeFrame(t *testing.T) {
	for _, symbolized := range []bool{false, true} {
		p := mergeFixture([]string{"C", "chunk", "FF"}, []string{
			"main", "lua_cpcall", "vm0", "pmain", "lua_pcall", "vm1", "lj_strfmt_num",
		}, true, 0x100000)
		byName := make(map[string]*pprof.Location)
		for _, loc := range p.Location {
			byName[loc.Line[0].Function.Name] = loc
		}
		nativeFF := byName["vm1"]
		if symbolized {
			nativeFF.Line[0].Function.Name = "lj_ff_tostring"
			nativeFF.Line[0].Function.SystemName = "lj_ff_tostring"
		} else {
			nativeFF.Line = nil
		}
		res := Postprocess(p, true)
		if res.MergedStacksCount != 1 || res.UnmergedStacksCount != 0 {
			t.Fatalf("FF stack rejected: %+v", res)
		}
		want := []*pprof.Location{
			byName["kernel"], byName["lj_strfmt_num"], nativeFF, byName["chunk"],
			byName["lua_pcall"], byName["pmain"], byName["lua_cpcall"], byName["main"],
		}
		if !slices.Equal(p.Sample[0].Location, want) {
			t.Fatalf("native FF not retained in place, symbolized=%t", symbolized)
		}
		assertNoHeader(t, p)
	}
}

func TestHeaderMappingGuards(t *testing.T) {
	for _, mapping := range []string{"none", "Lua", "separate native mapping"} {
		t.Run(mapping, func(t *testing.T) {
			p := mergeFixture([]string{"A"}, []string{"main", "vm"}, true, 0x100000)
			header := p.Sample[0].Location[0]
			switch mapping {
			case "none":
				header.Mapping = nil
			case "Lua":
				header.Mapping = p.Mapping[1]
			case "separate native mapping":
				copy := *p.Mapping[0]
				copy.ID = 4
				header.Mapping = &copy
				p.Mapping = append(p.Mapping, &copy)
			}
			light := mapping != "separate native mapping"
			res := Postprocess(p, true)
			if res.MergedStacksCount != boolInt(!light) || res.UnmergedStacksCount != boolInt(light) {
				t.Fatalf("metadata mapping result: %+v", res)
			}
			want := []string{"main", "A", "kernel"}
			if light {
				want = []string{"main", "vm", "A", "kernel"}
			}
			if got := rootNames(p.Sample[0]); !slices.Equal(got, want) {
				t.Fatalf("merged stack: %v", got)
			}
			assertNoHeader(t, p)
		})
	}
}

func TestNativeTruncationFromStackLength(t *testing.T) {
	for _, count := range []int{nativeStackCapacity - 1, nativeStackCapacity} {
		native := make([]string, count)
		for i := range native {
			native[i] = "caller"
		}
		native[count-1] = "vm"
		p := mergeFixture([]string{"A"}, native, true, 0x100000)
		original := slices.Clone(p.Sample[0].Location[1:])
		res := Postprocess(p, true)
		if count == nativeStackCapacity {
			if res.UnmergedStacksCount != 1 {
				t.Fatalf("full stack merged: %+v", res)
			}
			if !slices.Equal(p.Sample[0].Location[2:], original[2:]) {
				t.Fatal("fallback discarded native frames")
			}
		} else if res.MergedStacksCount != 1 || res.UnmergedStacksCount != 0 {
			t.Fatalf("shorter stack rejected: %+v", res)
		}
		assertNoHeader(t, p)
	}
}

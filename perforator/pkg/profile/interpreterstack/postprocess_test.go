package interpreterstack

import (
	"slices"
	"testing"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/pkg/profile/lua"
)

func TestLuaRunsAfterPHPStripping(t *testing.T) {
	native := &pprof.Mapping{Start: 0x1000, Limit: 0x10000}
	header := &pprof.Location{Address: 0x1000, Mapping: native, Line: []pprof.Line{{
		Function: &pprof.Function{SystemName: lua.VMInfoSystemName}, Column: 0x1000,
	}}}
	phpFrame := &pprof.Location{Mapping: &pprof.Mapping{File: profile.PHPSpecialMapping}}
	luaFrame := &pprof.Location{Mapping: &pprof.Mapping{File: profile.LuaSpecialMapping}}
	vm := &pprof.Location{Address: 0x1100, Mapping: native}
	main := &pprof.Location{Address: 0x3000, Mapping: native}
	p := &pprof.Profile{Sample: []*pprof.Sample{{Location: []*pprof.Location{header, phpFrame, luaFrame, vm, main}}}}
	res := Postprocess(p)
	if res.Lua.MergedStacksCount != 1 || res.Lua.UnmergedStacksCount != 0 {
		t.Fatalf("Lua ran before PHP stripping: %+v", res.Lua)
	}
	if !slices.Equal(p.Sample[0].Location, []*pprof.Location{luaFrame, main}) {
		t.Fatalf("unexpected merged stack: %+v", p.Sample[0].Location)
	}
}

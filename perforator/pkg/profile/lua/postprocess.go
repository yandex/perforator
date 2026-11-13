package lua

import (
	"math"
	"slices"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

// Reserved system names carry collector metadata independently of display names.
const (
	VMInfoSystemName      = "perforator.lua.vm_info"
	NativeCSystemName     = "perforator.lua.native_c"
	ASMFFSystemName       = "perforator.lua.asm_ff"
	ProtectedFFSystemName = "perforator.lua.protected_ff"
)

const VMInfoDisplayName = "[lua VM info]"

// VMInfoColumn encodes the VM range length in the header's pprof Column.
func VMInfoColumn(start, end uint64) int64 {
	if end <= start || end-start > math.MaxInt64 {
		return 0
	}
	return int64(end - start)
}

// AddVMInfo stores the FFI tail size in the metadata function's StartLine.
// The caller attaches the native mapping
// and finishes the location so the header rebases with the native stack.
func AddVMInfo(builder *profile.SampleBuilder, start, end uint64, flags, ffiCallSize uint32) *profile.LocationBuilder {
	loc := builder.AddNativeLocationUncached(start)
	loc.AddFrame().
		SetName(VMInfoDisplayName).
		SetMangledName(VMInfoSystemName).
		SetLine(int64(flags)).
		SetStartLine(int64(ffiCallSize)).
		SetColumn(VMInfoColumn(start, end)).
		Finish()
	return loc
}

// Both native unwinders write at most PERF_MAX_STACK_DEPTH entries (dwarf.h).
const nativeStackCapacity = 127

type PostProcessResults struct {
	MergedStacksCount   int
	UnmergedStacksCount int
}

func hasSystemName(loc *pprof.Location, name string) bool {
	if loc == nil {
		return false
	}
	for _, line := range loc.Line {
		if line.Function != nil && line.Function.SystemName == name {
			return true
		}
	}
	return false
}

func isLua(loc *pprof.Location) bool {
	return loc != nil && loc.Mapping != nil && loc.Mapping.File == string(profile.LuaSpecialMapping)
}

func isKernel(loc *pprof.Location) bool {
	return loc != nil && loc.Mapping != nil && loc.Mapping.File == string(profile.KernelSpecialMapping)
}

// StripVMInfo removes transport metadata even when interpreter merging is disabled.
// Remove table entries too: pprof serializes unreferenced locations.
func StripVMInfo(p *pprof.Profile) {
	for _, sample := range p.Sample {
		sample.Location = slices.DeleteFunc(sample.Location, func(loc *pprof.Location) bool {
			return hasSystemName(loc, VMInfoSystemName)
		})
	}
	p.Location = slices.DeleteFunc(p.Location, func(loc *pprof.Location) bool {
		return hasSystemName(loc, VMInfoSystemName)
	})
}

// Postprocess consumes the leading VM header and merges Lua stacks when enabled.
// Location order is leaf to root. Samples without a Lua header are left unchanged.
func Postprocess(p *pprof.Profile, enabled bool) (res PostProcessResults) {
	for _, sample := range p.Sample {
		if len(sample.Location) == 0 || !hasSystemName(sample.Location[0], VMInfoSystemName) {
			continue
		}
		header := sample.Location[0]
		sample.Location = sample.Location[1:]
		if !enabled {
			continue
		}
		luaEnd := 0
		for luaEnd < len(sample.Location) && isLua(sample.Location[luaEnd]) {
			luaEnd++
		}
		if luaEnd == 0 {
			continue
		}
		kernelEnd := luaEnd
		for kernelEnd < len(sample.Location) && isKernel(sample.Location[kernelEnd]) {
			kernelEnd++
		}
		if mergeSample(sample, header, luaEnd, kernelEnd) {
			res.MergedStacksCount++
		} else {
			appendLuaFramesBeforeKernel(sample, luaEnd, kernelEnd)
			res.UnmergedStacksCount++
		}
	}
	StripVMInfo(p)
	return res
}

func appendLuaFramesBeforeKernel(sample *pprof.Sample, luaEnd, kernelEnd int) {
	locs := sample.Location
	out := make([]*pprof.Location, 0, len(locs))
	out = append(out, locs[luaEnd:kernelEnd]...)
	out = append(out, locs[:luaEnd]...)
	out = append(out, locs[kernelEnd:]...)
	sample.Location = out
}

func mergeSample(sample *pprof.Sample, header *pprof.Location, luaEnd, kernelEnd int) bool {
	if header.Mapping == nil || header.Mapping.Limit <= header.Mapping.Start {
		// Without the native mapping, the header cannot be rebased with the native PCs.
		return false
	}
	if len(header.Line) != 1 {
		// The metadata must occupy a single line.
		return false
	}
	meta := header.Line[0]
	if meta.Line != 0 {
		// The Lua unwind failed, was truncated, or reported an unknown flag.
		return false
	}
	if meta.Column <= 0 {
		// The VM range must have a positive size.
		return false
	}
	if header.Address > math.MaxUint64-uint64(meta.Column) {
		// The exclusive end of the VM range would overflow.
		return false
	}
	vmEnd := header.Address + uint64(meta.Column)
	if header.Address < header.Mapping.Start || vmEnd > header.Mapping.Limit {
		// The VM range must fit inside the native mapping.
		return false
	}
	locs := sample.Location
	if len(locs)-kernelEnd >= nativeStackCapacity {
		// A native stack at capacity may be missing its oldest frames.
		return false
	}
	// The FFI tail is VM code, but its outbound trampoline does not anchor Lua frames.
	ffiCallSize := meta.Function.StartLine
	if ffiCallSize < 0 || ffiCallSize >= meta.Column {
		return false
	}
	luaVMEnd := vmEnd - uint64(ffiCallSize)
	isLuaVMFrame := func(loc *pprof.Location) bool {
		return loc != nil && header.Address <= loc.Address && loc.Address < luaVMEnd
	}

	// Each native C placeholder closes one Lua segment, including an empty one.
	var segments [][]*pprof.Location
	var segment []*pprof.Location
	endsAtNativeCall := false
	for i := luaEnd - 1; i >= 0; i-- {
		loc := locs[i]
		if hasSystemName(loc, ProtectedFFSystemName) {
			continue
		}
		if hasSystemName(loc, NativeCSystemName) {
			segments = append(segments, segment)
			segment = nil
			endsAtNativeCall = true
		} else {
			segment = append(segment, loc)
			endsAtNativeCall = false
		}
	}
	if !endsAtNativeCall {
		segments = append(segments, segment)
	}

	vmCount := 0
	previousVM := false
	nativeCallerFound := false
	for i := len(locs) - 1; i >= kernelEnd; i-- {
		vm := isLuaVMFrame(locs[i])
		if vm && previousVM {
			// Adjacent VM entries have no intervening native caller to establish a boundary.
			return false
		}
		if vm {
			// A VM helper can leave a native stack whose callers were lost.
			if vmCount == 0 && !nativeCallerFound {
				// Without a native caller below the first VM frame, the root may be missing.
				return false
			}
			vmCount++
		} else if locs[i] != nil && locs[i].Mapping != nil {
			nativeCallerFound = true
		}
		previousVM = vm
	}
	if vmCount == 0 {
		// There is no native VM frame to anchor a Lua segment.
		return false
	}
	// A Lua leaf entered from C can be appended after the youngest native caller.
	appendLeafLuaSegment := len(segments) == vmCount+1 && !endsAtNativeCall && kernelEnd < len(locs) && !isLuaVMFrame(locs[kernelEnd])
	// A native leaf has entered the VM before its Lua frame was initialized.
	keepLeafVMFrame := vmCount == len(segments)+1 && endsAtNativeCall
	if vmCount != len(segments) && !appendLeafLuaSegment && !keepLeafVMFrame {
		// The VM and Lua segment counts must agree apart from a recoverable leaf transition.
		return false
	}

	// Build root to leaf so a partially initialized leaf cannot shift older segments.
	root := make([]*pprof.Location, 0, len(locs))
	segmentIndex := 0
	for i := len(locs) - 1; i >= kernelEnd; i-- {
		loc := locs[i]
		if isLuaVMFrame(loc) {
			if segmentIndex < len(segments) {
				segment := segments[segmentIndex]
				if last := len(segment) - 1; last >= 0 && hasSystemName(segment[last], ASMFFSystemName) {
					// The VM activation already represents this leaf fast function.
					segment[last] = loc
				}
				// C-only VM entries have no Lua frames to insert.
				root = append(root, segment...)
			} else {
				// The VM entry exists but its Lua frame has not been initialized.
				root = append(root, loc)
			}
			segmentIndex++
		} else {
			root = append(root, loc)
		}
	}
	if appendLeafLuaSegment {
		root = append(root, segments[len(segments)-1]...)
	}
	slices.Reverse(root)
	out := make([]*pprof.Location, 0, kernelEnd-luaEnd+len(root))
	out = append(out, locs[luaEnd:kernelEnd]...)
	sample.Location = append(out, root...)
	return true
}

package lua

import (
	"errors"
	"fmt"
	"slices"

	pprof "github.com/google/pprof/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/lua/hardcode"
	"github.com/yandex/perforator/perforator/internal/linguist/lua/models"
)

const (
	invalid = "<invalid>"
)

// both bounds are included
type StackSubsegment struct {
	Left  int
	Right int
}

func (s *StackSubsegment) Length() int {
	return s.Right - s.Left
}

type NativeAndLuaStackMerger struct {
	sample             *pprof.Sample
	cStackIndex        int
	luaStartStackIndex int
	luaStackIndex      int

	resultStack []*pprof.Location

	luaSegments            []StackSubsegment
	luaInterpreterSegments []StackSubsegment
}

func NewNativeAndLuaStackMerger() *NativeAndLuaStackMerger {
	return &NativeAndLuaStackMerger{
		resultStack:            make([]*pprof.Location, 0, 512),
		luaSegments:            []StackSubsegment{},
		luaInterpreterSegments: []StackSubsegment{},
	}
}

func (m *NativeAndLuaStackMerger) reset(sample *pprof.Sample) {
	m.sample = sample
	m.luaStackIndex = -1
	m.luaStartStackIndex = -1
	m.cStackIndex = len(sample.Location) - 1
	m.luaSegments = m.luaSegments[:0]
	m.luaInterpreterSegments = m.luaInterpreterSegments[:0]
	m.resultStack = m.resultStack[:0]
}

func (m *NativeAndLuaStackMerger) cleanup() {
	m.sample = nil
}

func isInternalLuaEvaluationFunction(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil &&
			(line.Function.Name == invalid || line.Function.SystemName == invalid ||
				hardcode.LuaInternalEvaluationFunctions[line.Function.Name] || hardcode.LuaInternalEvaluationFunctions[line.Function.SystemName]) {
			return true
		}
	}

	return false
}

func isLuaEvaluationEntryPoint(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil &&
			(hardcode.LuaAPIEvaluationFunctions[line.Function.Name] || hardcode.LuaAPIEvaluationFunctions[line.Function.SystemName]) {
			return true
		}
	}

	return false
}

// TLDR: Extract substack from native stack that corresponds to single lua substack
// For example Lua substack may look like: <trampoline lua frame> -> find_and_load -> load_unlocked
//
// Algorithm: This substack starts with some Lua API function for evaluation
// then we consider <invalid> and internal Lua evaluation function frames as the result substack.
// We stop when we see function which is not <invalid> and is not internal Lua evaluation function,
// the stop point might be function like this: `PyCFunction_Call` or `PyImport_ImportModuleLevelObject`
func (m *NativeAndLuaStackMerger) nextCStackLuaInterpreterSegment() (res *StackSubsegment) {
	for ; m.cStackIndex > m.luaStartStackIndex; m.cStackIndex-- {
		i := m.cStackIndex
		isLuaEntryPoint := isLuaEvaluationEntryPoint(m.sample.Location[i])

		if res != nil {
			if isLuaEntryPoint || !isInternalLuaEvaluationFunction(m.sample.Location[i]) {
				break
			}

			// include pcall itself as well
			res.Left = i - 1
		} else if isLuaEntryPoint {
			res = &StackSubsegment{Left: i - 1, Right: i - 1}
		}
	}

	return res
}

func isTrampolineLuaFrame(f *pprof.Function) bool {
	return f.Name == models.LuaTrampolineFrame
}

func (m *NativeAndLuaStackMerger) nextLuaInterpreterSegment() (res *StackSubsegment, err error) {
	if m.luaStackIndex < 0 {
		return nil, nil
	}

	res = &StackSubsegment{Right: m.luaStackIndex}
	m.luaStackIndex--

	for ; m.luaStackIndex >= 0; m.luaStackIndex-- {
		loc := m.sample.Location[m.luaStackIndex]

		if len(loc.Line) != 1 {
			// Lua location must contain exactly one line because it the way we collect them on agent side
			return nil, fmt.Errorf("len(Line) of lua location must be 1, got %d", len(loc.Line))
		}

		if loc.Line[0].Function == nil {
			// *pprof.Function is also set for *pprof.Location on agent, so here we just sanity check this
			return nil, errors.New("*pprof.Function not set for lua *pprof.Location")
		}
	}

	res.Left = m.luaStackIndex + 1
	return res, nil
}

func isLuaLocation(loc *pprof.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(profile.LuaSpecialMapping)
}

func isKernelLocation(loc *pprof.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(profile.KernelSpecialMapping)
}

func (m *NativeAndLuaStackMerger) setStartLuaStackIndex() (foundLuaStack bool) {
	if len(m.sample.Location) == 0 {
		return false
	}

	if !isLuaLocation(m.sample.Location[0]) {
		return false
	}

	for i, loc := range m.sample.Location {
		if !isLuaLocation(loc) {
			break
		}

		m.luaStartStackIndex = i
	}

	return true
}

func (m *NativeAndLuaStackMerger) extractLuaAndCSubstacks() error {
	for seg := m.nextCStackLuaInterpreterSegment(); seg != nil; seg = m.nextCStackLuaInterpreterSegment() {
		m.luaInterpreterSegments = append(m.luaInterpreterSegments, *seg)
	}

	m.luaStackIndex = m.luaStartStackIndex

	for {
		luaSeg, err := m.nextLuaInterpreterSegment()
		if err != nil {
			return err
		}

		if luaSeg == nil {
			break
		}

		m.luaSegments = append(m.luaSegments, *luaSeg)
	}

	return nil
}

type MergeStackStats struct {
	LuaSubStacks   []StackSubsegment
	CSubStacks     []StackSubsegment
	CollectedLua   bool
	PerformedMerge bool
}

/*
TLDR: substitute each lua interpreter substack with higher level lua substack,
then replace the original slice with constructed slice

	`-` - Lua interpreter frame. This frame is replaced with lua frame
	`+` - C non lua interpreter frame. This frame remains.
	`*` - Lua frame
	`|` - frame separator

Example:

	C stack:  | + | - | - | - | - | + | + | - | - | - |
	      merge with
	Lua stack:      |  *  |   *  |   C stack here  ->      |  *  |
	Result:   | + | * | * | + | + | * |
*/
func (m *NativeAndLuaStackMerger) substituteInterpreterStack() {
	prevNative := len(m.sample.Location) - 1

	for i := 0; i < len(m.luaInterpreterSegments); i++ {
		for ; prevNative > m.luaInterpreterSegments[i].Right; prevNative-- {
			m.resultStack = append(m.resultStack, m.sample.Location[prevNative])
		}

		prevNative = m.luaInterpreterSegments[i].Left

		for idx := m.luaSegments[i].Right; idx >= m.luaSegments[i].Left; idx-- {
			m.resultStack = append(m.resultStack, m.sample.Location[idx])
		}
	}

	for ; prevNative > m.luaStartStackIndex; prevNative-- {
		m.resultStack = append(m.resultStack, m.sample.Location[prevNative])
	}

	slices.Reverse(m.resultStack)
	location := m.sample.Location[:0]

	for _, loc := range m.resultStack {
		if len(loc.Line) > 0 && isTrampolineLuaFrame(loc.Line[0].Function) {
			// skip lua trampolines
		} else {
			location = append(location, loc)
		}
	}

	m.sample.Location = location
}

// Remove the last Lua substack if it has not started evaluating
// Lua yet.
func (m *NativeAndLuaStackMerger) trimLastLuaSubstackIfNeeded() {
	if len(m.luaSegments)+1 == len(m.luaInterpreterSegments) {
		m.luaInterpreterSegments = m.luaInterpreterSegments[:len(m.luaInterpreterSegments)-1]
	}
}

func (m *NativeAndLuaStackMerger) putLuaBeforeKernelStack() {
	userspaceStackStartIndex := m.luaStartStackIndex + 1

	for i := m.luaStartStackIndex + 1; i < len(m.sample.Location); i++ {
		if !isKernelLocation(m.sample.Location[i]) {
			userspaceStackStartIndex = i
			break
		}

		m.resultStack = append(m.resultStack, m.sample.Location[i])
	}

	for i := 0; i <= m.luaStartStackIndex; i++ {
		m.resultStack = append(m.resultStack, m.sample.Location[i])
	}

	for i := userspaceStackStartIndex; i < len(m.sample.Location); i++ {
		m.resultStack = append(m.resultStack, m.sample.Location[i])
	}

	m.sample.Location = m.sample.Location[:0]
	m.sample.Location = append(m.sample.Location, m.resultStack...)
}

func (m *NativeAndLuaStackMerger) mergeSubstacksMapping(stats *MergeStackStats) error {
	err := m.extractLuaAndCSubstacks()
	if err != nil {
		return fmt.Errorf("failed to extract lua and c substacks: %w", err)
	}

	//m.trimLastLuaSubstackIfNeeded()

	stats.CSubStacks = append(stats.CSubStacks, m.luaInterpreterSegments...)
	stats.LuaSubStacks = append(stats.LuaSubStacks, m.luaSegments...)

	if len(stats.LuaSubStacks) != len(stats.CSubStacks) {
		// Most probably lua interpreter C stacks are not extracted correctly
		//   so do not continue with merge
		return nil
	}

	if len(stats.LuaSubStacks) == 0 {
		return nil
	}

	m.substituteInterpreterStack()
	stats.PerformedMerge = true

	return nil
}

// Merge stacks inplace for this sample.
// Stack is laid down top to bottom from left to right.
// We expect that first frames are reverse lua frames, then reverse kernel frames, then reversed userspace frames.
// The resulting stack is return using the same layout.
// Here is the code that originates this layout:
// https://github.com/yandex/perforator/blob/f838dd038cc7437bb5674d8ccee2c6086f0bc46c/perforator/agent/collector/pkg/profiler/sample_consumer.go#L480
func (m *NativeAndLuaStackMerger) MergeStacks(s *pprof.Sample) (MergeStackStats, error) {
	m.reset(s)

	if m.sample == nil {
		return MergeStackStats{}, nil
	}

	defer m.cleanup()

	stats := MergeStackStats{}

	stats.CollectedLua = m.setStartLuaStackIndex()
	if !stats.CollectedLua {
		return stats, nil
	}

	var err error
	err = m.mergeSubstacksMapping(&stats)
	if err != nil {
		return stats, err
	}

	if !stats.PerformedMerge {
		m.putLuaBeforeKernelStack()
	}

	return stats, nil
}

type PostProcessResults struct {
	// Number of stacks that do not contain any lua.
	NotLuaStacksCount int

	// Number of stacks that contain lua evaluated stack collected via bpf.
	CollectedLuaStacksCount int
	// Number of stacks that contain native lua evaluation frames but do not contain collect lua stack.
	CollectFailedLuaStacksCount int

	// Number of unmerged stacks out of stacks that have lua collected.
	UnmergedStacksCount int
	// Number of merged stacks out of stacks that have lua collected.
	MergedStacksCount int

	Errors []error
}

func PostprocessSymbolizedProfileWithLua(p *pprof.Profile) (res PostProcessResults) {
	merger := NewNativeAndLuaStackMerger()

	for _, sample := range p.Sample {
		println("START")

		for key, value := range sample.Label {
			fmt.Printf("SPAR: key: %v value: %v\n", key, value)
		}

		stats, err := merger.MergeStacks(sample)
		if err != nil {
			res.Errors = append(res.Errors, err)
		}

		if stats.CollectedLua {
			res.CollectedLuaStacksCount++
		} else if len(stats.CSubStacks) > 0 {
			res.CollectFailedLuaStacksCount++
			continue
		} else {
			res.NotLuaStacksCount++
			continue
		}

		if stats.PerformedMerge {
			res.MergedStacksCount++
		} else {
			res.UnmergedStacksCount++
		}
	}

	return
}

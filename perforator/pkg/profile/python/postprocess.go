package python

import (
	"errors"
	"fmt"
	"slices"

	"github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/internal/linguist/python/hardcode"
	"github.com/yandex/perforator/perforator/internal/linguist/python/models"
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

type NativeAndPythonStackMerger struct {
	sample                *profile.Sample
	cStackIndex           int
	pythonStartStackIndex int
	pythonStackIndex      int

	resultStack []*profile.Location

	pythonSegments             []StackSubsegment
	cPythonInterpreterSegments []StackSubsegment
}

func NewNativeAndPythonStackMerger() *NativeAndPythonStackMerger {
	return &NativeAndPythonStackMerger{
		resultStack:                make([]*profile.Location, 0, 512),
		pythonSegments:             []StackSubsegment{},
		cPythonInterpreterSegments: []StackSubsegment{},
	}
}

func (m *NativeAndPythonStackMerger) reset(sample *profile.Sample) {
	m.sample = sample
	m.pythonStackIndex = -1
	m.pythonStartStackIndex = -1
	m.cStackIndex = len(sample.Location) - 1
	m.pythonSegments = m.pythonSegments[:0]
	m.cPythonInterpreterSegments = m.cPythonInterpreterSegments[:0]
	m.resultStack = m.resultStack[:0]
}

func (m *NativeAndPythonStackMerger) cleanup() {
	m.sample = nil
}

func isInternalCPythonEvaluationFunction(loc *profile.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil &&
			(line.Function.Name == invalid || line.Function.SystemName == invalid ||
				hardcode.CPythonInternalEvaluationFunctions[line.Function.Name] || hardcode.CPythonInternalEvaluationFunctions[line.Function.SystemName]) {
			return true
		}
	}

	return false
}

func isCPythonEvaluationEntryPoint(loc *profile.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil &&
			(hardcode.CPythonAPIEvaluationFunctions[line.Function.Name] || hardcode.CPythonAPIEvaluationFunctions[line.Function.SystemName]) {
			return true
		}
	}

	return false
}

// TLDR: Extract substack from native stack that corresponds to single python substack
// For example Python substack may look like: <trampoline python frame> -> find_and_load -> load_unlocked
//
// Algorithm: This substack starts with some CPython API function for evaluation
// then we consider <invalid> and internal CPython evaluation function frames as the result substack.
// We stop when we see function which is not <invalid> and is not internal CPython evaluation function,
// the stop point might be function like this: `PyCFunction_Call` or `PyImport_ImportModuleLevelObject`
func (m *NativeAndPythonStackMerger) nextCStackPythonInterpreterSegment() (res *StackSubsegment) {
	for ; m.cStackIndex > m.pythonStartStackIndex; m.cStackIndex-- {
		i := m.cStackIndex

		isCPythonEntryPoint := isCPythonEvaluationEntryPoint(m.sample.Location[i])

		if res != nil {
			if isCPythonEntryPoint || !isInternalCPythonEvaluationFunction(m.sample.Location[i]) {
				break
			}

			res.Left = i
		} else {
			if isCPythonEntryPoint {
				res = &StackSubsegment{Left: i, Right: i}
			}
		}
	}

	return res
}

func isTrampolinePythonFrame(f *profile.Function) bool {
	return f.Name == models.PythonTrampolineFrame
}

func (m *NativeAndPythonStackMerger) nextPythonInterpreterSegment() (res *StackSubsegment, err error) {
	if m.pythonStackIndex < 0 {
		return nil, nil
	}

	res = &StackSubsegment{Right: m.pythonStackIndex}
	m.pythonStackIndex--

	for ; m.pythonStackIndex >= 0; m.pythonStackIndex-- {
		loc := m.sample.Location[m.pythonStackIndex]
		if len(loc.Line) != 1 {
			// Python location must contain exactly one line because it the way we collect them on agent side
			return nil, fmt.Errorf("len(Line) of python location must be 1, got %d", len(loc.Line))
		}

		if loc.Line[0].Function == nil {
			// *profile.Function is also set for *profile.Location on agent, so here we just sanity check this
			return nil, errors.New("*profile.Function not set for python *profile.Location")
		}

		if isTrampolinePythonFrame(loc.Line[0].Function) {
			break
		}
	}

	res.Left = m.pythonStackIndex + 1
	return res, nil
}

func isPythonLocation(loc *profile.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(models.PythonSpecialMapping)
}

func (m *NativeAndPythonStackMerger) setStartPythonStackIndex() (foundPythonStack bool) {
	if len(m.sample.Location) == 0 {
		return false
	}

	if !isPythonLocation(m.sample.Location[0]) {
		return false
	}

	for i, loc := range m.sample.Location {
		if !isPythonLocation(loc) {
			break
		}

		m.pythonStartStackIndex = i
	}

	return true
}

func (m *NativeAndPythonStackMerger) extractPythonAndCSubstacks() error {
	for seg := m.nextCStackPythonInterpreterSegment(); seg != nil; seg = m.nextCStackPythonInterpreterSegment() {
		m.cPythonInterpreterSegments = append(m.cPythonInterpreterSegments, *seg)
	}

	m.pythonStackIndex = m.pythonStartStackIndex

	for {
		pythonSeg, err := m.nextPythonInterpreterSegment()
		if err != nil {
			return err
		}

		if pythonSeg == nil {
			break
		}

		m.pythonSegments = append(m.pythonSegments, *pythonSeg)
	}

	return nil
}

type MergeStackStats struct {
	PythonSubStacks []StackSubsegment
	CSubStacks      []StackSubsegment
	CollectedPython bool
	PerformedMerge  bool
}

/*
TLDR: substitute each python interpreter substack with higher level python substack,
then replace the original slice with constructed slice

	`-` - C python interpreter frame. This frame is replaced with python frame
	`+` - C non python interpreter frame. This frame remains.
	`*` - Python frame
	`|` - frame separator

Example:

	C stack:  | + | - | - | - | - | + | + | - | - | - |
	      merge with
	Python stack:      |  *  |   *  |   C stack here  ->      |  *  |
	Result:   | + | * | * | + | + | * |
*/
func (m *NativeAndPythonStackMerger) substituteInterpreterStack() {
	prevNative := len(m.sample.Location) - 1

	for i := 0; i < len(m.cPythonInterpreterSegments); i++ {
		for ; prevNative > m.cPythonInterpreterSegments[i].Right; prevNative-- {
			m.resultStack = append(m.resultStack, m.sample.Location[prevNative])
		}
		prevNative = m.cPythonInterpreterSegments[i].Left - 1

		for idx := m.pythonSegments[i].Right; idx >= m.pythonSegments[i].Left; idx-- {
			m.resultStack = append(m.resultStack, m.sample.Location[idx])
		}
	}

	for ; prevNative > m.pythonStartStackIndex; prevNative-- {
		m.resultStack = append(m.resultStack, m.sample.Location[prevNative])
	}

	slices.Reverse(m.resultStack)
	m.sample.Location = m.sample.Location[:0]
	m.sample.Location = append(m.sample.Location, m.resultStack...)
}

// Remove the last CPython substack if it has not started evaluating
// python yet.
func (m *NativeAndPythonStackMerger) trimLastCPythonSubstackIfNeeded() {
	if len(m.pythonSegments)+1 == len(m.cPythonInterpreterSegments) {
		m.cPythonInterpreterSegments = m.cPythonInterpreterSegments[:len(m.cPythonInterpreterSegments)-1]
	}
}

// Merge stacks inplace for this sample
// Stack is laid down top to bottom from left to right
func (m *NativeAndPythonStackMerger) MergeStacks(s *profile.Sample) (MergeStackStats, error) {
	m.reset(s)
	if m.sample == nil {
		return MergeStackStats{}, nil
	}

	stats := MergeStackStats{}
	stats.CollectedPython = m.setStartPythonStackIndex()
	defer m.cleanup()

	err := m.extractPythonAndCSubstacks()
	if err != nil {
		return stats, fmt.Errorf("failed to extract python and c substacks: %w", err)
	}

	m.trimLastCPythonSubstackIfNeeded()

	stats.CSubStacks = append(stats.CSubStacks, m.cPythonInterpreterSegments...)
	stats.PythonSubStacks = append(stats.PythonSubStacks, m.pythonSegments...)

	if len(stats.PythonSubStacks) != len(stats.CSubStacks) {
		// Most probably python interpreter C stacks are not extracted correctly
		//   so do not continue with merge
		return stats, nil
	}

	if len(stats.PythonSubStacks) == 0 {
		return stats, nil
	}

	m.substituteInterpreterStack()

	stats.PerformedMerge = true
	return stats, nil
}

type PostProcessResults struct {
	// Number of stacks that do not contain any python.
	NotPythonStacksCount int

	// Number of stacks that contain python evaluated stack collected via bpf.
	CollectedPythonStacksCount int
	// Number of stacks that contain native python evaluation frames but do not contain collect python stack.
	CollectFailedPythonStacksCount int

	// Number of unmerged stacks out of stacks that have python collected.
	UnmergedStacksCount int
	// Number of merged stacks out of stacks that have python collected.
	MergedStacksCount int

	Errors []error
}

func PostprocessSymbolizedProfileWithPython(p *profile.Profile) (res PostProcessResults) {
	merger := NewNativeAndPythonStackMerger()
	for _, sample := range p.Sample {
		stats, err := merger.MergeStacks(sample)
		if err != nil {
			res.Errors = append(res.Errors, err)
		}

		if stats.CollectedPython {
			res.CollectedPythonStacksCount++
		} else if len(stats.CSubStacks) > 0 {
			res.CollectFailedPythonStacksCount++
			continue
		} else {
			res.NotPythonStacksCount++
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

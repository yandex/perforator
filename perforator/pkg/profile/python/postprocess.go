package python

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/python/hardcode"
	"github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render"
)

const (
	invalid = "<invalid>"
)

type MergeAlgorithm int

const (
	// This merge algorithm is used for CPython before 3.11
	OneToOnePythonFrameToPyEval MergeAlgorithm = iota
	// This merge algorithm is used after (>=) CPython 3.11
	SubstacksMapping
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
	sample                *pprof.Sample
	cStackIndex           int
	pythonStartStackIndex int
	pythonStackIndex      int

	resultStack []*pprof.Location

	pythonSegments             []StackSubsegment
	cPythonInterpreterSegments []StackSubsegment
}

func NewNativeAndPythonStackMerger() *NativeAndPythonStackMerger {
	return &NativeAndPythonStackMerger{
		resultStack:                make([]*pprof.Location, 0, 512),
		pythonSegments:             []StackSubsegment{},
		cPythonInterpreterSegments: []StackSubsegment{},
	}
}

func (m *NativeAndPythonStackMerger) reset(sample *pprof.Sample) {
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

func isInternalCPythonEvaluationFunction(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil &&
			(line.Function.Name == invalid || line.Function.SystemName == invalid ||
				hardcode.CPythonEntryPointFunctions[line.Function.Name] || hardcode.CPythonEntryPointFunctions[line.Function.SystemName]) {
			return true
		}
	}

	return false
}

func isCPythonEvaluationEntryPoint(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil &&
			(hardcode.CPythonEntryPointFunctions[line.Function.Name] || hardcode.CPythonEntryPointFunctions[line.Function.SystemName]) {
			return true
		}
	}

	return false
}

// This function is called to calculate some data which will be used
// to determine which merge algorithm we should use
func (m *NativeAndPythonStackMerger) determineMergeAlgorithm() MergeAlgorithm {
	pythonFramesCount := 0
	pyEvalFrameDefaultCount := 0

	for _, loc := range m.sample.Location {
		switch {
		case isPythonLocation(loc):
			pythonFramesCount++
		case isCPythonEvaluationEntryPoint(loc):
			pyEvalFrameDefaultCount++
		}
	}

	// We need to check the second condition because of possible inconsistencies between python and native stacks.
	// Suppose `PyEval_EvalFrameDefault` has just started executing, we will collect its frame in native stack,
	// but the corresponding python frame was not pushed on the stack yet.
	if pyEvalFrameDefaultCount == pythonFramesCount || pyEvalFrameDefaultCount == pythonFramesCount+1 {
		return OneToOnePythonFrameToPyEval
	}

	return SubstacksMapping
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

func isTrampolinePythonFrame(f *pprof.Function) bool {
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
			// *pprof.Function is also set for *pprof.Location on agent, so here we just sanity check this
			return nil, errors.New("*pprof.Function not set for python *pprof.Location")
		}

		if isTrampolinePythonFrame(loc.Line[0].Function) {
			break
		}
	}

	res.Left = m.pythonStackIndex + 1
	return res, nil
}

func isPythonLocation(loc *pprof.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(profile.PythonSpecialMapping)
}

func isKernelLocation(loc *pprof.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(profile.KernelSpecialMapping)
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

func (m *NativeAndPythonStackMerger) putPythonBeforeKernelStack() {
	userspaceStackStartIndex := m.pythonStartStackIndex + 1
	for i := m.pythonStartStackIndex + 1; i < len(m.sample.Location); i++ {
		if !isKernelLocation(m.sample.Location[i]) {
			userspaceStackStartIndex = i
			break
		}

		m.resultStack = append(m.resultStack, m.sample.Location[i])
	}

	for i := 0; i <= m.pythonStartStackIndex; i++ {
		m.resultStack = append(m.resultStack, m.sample.Location[i])
	}

	for i := userspaceStackStartIndex; i < len(m.sample.Location); i++ {
		m.resultStack = append(m.resultStack, m.sample.Location[i])
	}

	m.sample.Location = m.sample.Location[:0]
	m.sample.Location = append(m.sample.Location, m.resultStack...)
}

func (m *NativeAndPythonStackMerger) mergeSubstacksMapping(stats *MergeStackStats) error {
	err := m.extractPythonAndCSubstacks()
	if err != nil {
		return fmt.Errorf("failed to extract python and c substacks: %w", err)
	}

	m.trimLastCPythonSubstackIfNeeded()

	stats.CSubStacks = append(stats.CSubStacks, m.cPythonInterpreterSegments...)
	stats.PythonSubStacks = append(stats.PythonSubStacks, m.pythonSegments...)

	if len(stats.PythonSubStacks) != len(stats.CSubStacks) {
		// Most probably python interpreter C stacks are not extracted correctly
		//   so do not continue with merge
		return nil
	}

	if len(stats.PythonSubStacks) == 0 {
		return nil
	}

	m.substituteInterpreterStack()

	stats.PerformedMerge = true
	return nil
}

func (m *NativeAndPythonStackMerger) mergeOneToOnePythonFrameToPyEval(stats *MergeStackStats) error {
	pythonIdx := m.pythonStartStackIndex

	for nativeIdx := len(m.sample.Location) - 1; nativeIdx > m.pythonStartStackIndex; nativeIdx-- {
		idxToCopyLocationFrom := nativeIdx
		if isCPythonEvaluationEntryPoint(m.sample.Location[nativeIdx]) && pythonIdx >= 0 {
			idxToCopyLocationFrom = pythonIdx
			pythonIdx--
		}

		m.resultStack = append(m.resultStack, m.sample.Location[idxToCopyLocationFrom])
	}

	slices.Reverse(m.resultStack)
	m.sample.Location = m.sample.Location[:0]
	m.sample.Location = append(m.sample.Location, m.resultStack...)

	stats.PerformedMerge = true
	return nil
}

// Merge stacks inplace for this sample.
// Stack is laid down top to bottom from left to right.
// We expect that first frames are reverse python frames, then reverse kernel frames, then reversed userspace frames.
// The resulting stack is return using the same layout.
// Here is the code that originates this layout:
// https://github.com/yandex/perforator/blob/f838dd038cc7437bb5674d8ccee2c6086f0bc46c/perforator/agent/collector/pkg/profiler/sample_consumer.go#L480
func (m *NativeAndPythonStackMerger) MergeStacks(s *pprof.Sample) (MergeStackStats, error) {
	m.reset(s)
	if m.sample == nil {
		return MergeStackStats{}, nil
	}
	defer m.cleanup()

	if !m.setStartPythonStackIndex() {
		return MergeStackStats{}, nil
	}
	var err error

	stats := MergeStackStats{}

	switch m.determineMergeAlgorithm() {
	case OneToOnePythonFrameToPyEval:
		err = m.mergeOneToOnePythonFrameToPyEval(&stats)
	case SubstacksMapping:
		err = m.mergeSubstacksMapping(&stats)
	default:
		return MergeStackStats{}, errors.New("unknown merge algorithm")
	}
	if err != nil {
		return stats, err
	}

	if !stats.PerformedMerge {
		m.putPythonBeforeKernelStack()
	}

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

// PrettifyLevel controls how aggressively Python stacks are prettified.
type PrettifyLevel int

const (
	// PrettifyOff disables prettification entirely. This is the default.
	PrettifyOff PrettifyLevel = iota
	// PrettifyMixed removes CPython interpreter frames, importlib and trampoline frames,
	// but keeps user native frames (e.g. PyTorch, NumPy C extensions) for ML workloads.
	PrettifyMixed
	// PrettifyPythonOnly applies Mixed and then removes all remaining non-Python frames,
	// leaving only .py frames.
	PrettifyPythonOnly
)

type options struct {
	prettifyLevel PrettifyLevel
}

func defaultOptions() options {
	return options{
		prettifyLevel: PrettifyOff,
	}
}

type Option func(*options)

// WithPrettifyLevel sets the prettification level.
func WithPrettifyLevel(level PrettifyLevel) Option {
	return func(o *options) {
		o.prettifyLevel = level
	}
}

func containsInterpretedPythonStack(sample *pprof.Sample) bool {
	for _, loc := range sample.Location {
		if isPythonLocation(loc) {
			return true
		}
	}

	return false
}

func containsInterpreterPythonStack(sample *pprof.Sample) bool {
	for _, loc := range sample.Location {
		if isCPythonEvaluationEntryPoint(loc) {
			return true
		}
	}

	return false
}

// This functions assumes that the profile is already symbolized
func Postprocess(p *pprof.Profile, opts ...Option) (res PostProcessResults) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	merger := NewNativeAndPythonStackMerger()
	for _, sample := range p.Sample {
		containsPythonStack := containsInterpretedPythonStack(sample)
		containsInterpreterPythonStack := containsInterpreterPythonStack(sample)

		if containsPythonStack {
			res.CollectedPythonStacksCount++
		} else if containsInterpreterPythonStack {
			res.CollectFailedPythonStacksCount++
		} else {
			res.NotPythonStacksCount++
		}

		if !containsPythonStack {
			continue
		}

		stats, err := merger.MergeStacks(sample)
		if err != nil {
			res.Errors = append(res.Errors, err)
		}

		if stats.PerformedMerge {
			res.MergedStacksCount++
		} else {
			res.UnmergedStacksCount++
		}

		if o.prettifyLevel != PrettifyOff {
			newPrettifier(sample, o.prettifyLevel).prettify()
		}
	}

	return
}

// prettifier performs inplace prettification of the Python sample.
type prettifier struct {
	sample *pprof.Sample
	level  PrettifyLevel
}

func newPrettifier(sample *pprof.Sample, level PrettifyLevel) *prettifier {
	return &prettifier{
		sample: sample,
		level:  level,
	}
}

func isUnsymbolizedLocation(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil && !(render.IsInvalidFunctionName(line.Function.Name) && render.IsInvalidFunctionName(line.Function.SystemName)) {
			return false
		}
	}

	return true
}

func isCPythonOrPythonLocation(loc *pprof.Location) bool {
	return isCPythonLocation(loc) || isPythonLocation(loc)
}

func (p *prettifier) removeUnsymbolizedCPythonFrames() {
	// We remove segments of unsymbolized locations if they are surrounded by CPython or Python locations.
	// First pass: mark segments for removal by setting them to nil
	unsymSegmentStart := -1
	for i := range p.sample.Location {
		isUnsym := isUnsymbolizedLocation(p.sample.Location[i])
		isCPyOrPy := isCPythonOrPythonLocation(p.sample.Location[i])
		prevIsCPyOrPy := i == 0 || isCPythonOrPythonLocation(p.sample.Location[i-1])

		switch {
		case isUnsym && unsymSegmentStart == -1 && prevIsCPyOrPy:
			// Segment started
			unsymSegmentStart = i
		case !isUnsym && unsymSegmentStart != -1:
			// Segment ended
			if isCPyOrPy {
				// Surrounded on both sides — mark for removal
				for j := unsymSegmentStart; j < i; j++ {
					p.sample.Location[j] = nil
				}
			}
			unsymSegmentStart = -1
		}
	}

	// Second pass: filter out nil entries
	p.sample.Location = slices.DeleteFunc(p.sample.Location, func(loc *pprof.Location) bool {
		return loc == nil
	})
}

func isCPythonFunction(name string) bool {
	return strings.HasPrefix(name, "Py") || // CPython
		strings.HasPrefix(name, "_Py") || // CPython
		strings.HasPrefix(name, "__Pyx") || // Cython
		strings.HasPrefix(name, "__pyx") // Cython
}

func isCPythonLocation(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function == nil {
			continue
		}

		if isCPythonFunction(line.Function.Name) || isCPythonFunction(line.Function.SystemName) {
			return true
		}
	}

	return false
}

// Remove all CPython locations except latest ones in the stack.
// The latest CPython locations are those which are called from interpreted Python code
// They may contain useful information for user
// For example it might be PyNumber_InPlaceAdd instead of usual evaluation loop functions
func (p *prettifier) removeCPythonInterpreterLocations() {
	resultIndex := 0
	seenPythonLocation := false

	// keep in mind the reverse layout of the stack
	for _, loc := range p.sample.Location {
		if isPythonLocation(loc) {
			seenPythonLocation = true
		}

		if isCPythonLocation(loc) && seenPythonLocation {
			continue
		}

		p.sample.Location[resultIndex] = loc
		resultIndex++
	}

	p.sample.Location = p.sample.Location[:resultIndex]
}

func isImportlibLocation(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil && strings.Contains(line.Function.Filename, "importlib") {
			return true
		}
	}

	return false
}

func isTrampolinePythonLocation(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if isTrampolinePythonFrame(line.Function) {
			return true
		}
	}

	return false
}

// removeAllNonPythonLocations removes all locations that are not Python locations,
func (p *prettifier) removeAllNonPythonLocations() {
	p.sample.Location = slices.DeleteFunc(p.sample.Location, func(loc *pprof.Location) bool {
		return !isPythonLocation(loc)
	})
}

func (p *prettifier) prettify() {
	if p.level == PrettifyOff {
		return
	}

	// There are a lot of stripped CPython binaries. This can cause <unsymbolized function> frames between CPython interpreter frames.
	// They can be inlined or not. These <unsymbolized function> frames should be treated as CPython intrepreter frames and be removed.
	// Step 1: remove <unsymbolized function> frames between CPython interpreter locations.
	p.removeUnsymbolizedCPythonFrames()

	// CPython interpreter locations usually are not informative, so remove them.
	// Step 2: remove CPython interpreter locations
	p.removeCPythonInterpreterLocations()

	// Step 3: remove importlib, <trampoline python frame> frames
	p.sample.Location = slices.DeleteFunc(p.sample.Location, func(loc *pprof.Location) bool {
		return isImportlibLocation(loc) || isTrampolinePythonLocation(loc)
	})

	// Step 4 (PythonOnly): remove all remaining non-Python frames,
	if p.level == PrettifyPythonOnly {
		p.removeAllNonPythonLocations()
	}
}

// PrettifyProfile runs prettification on all samples that contain Python frames.
func PrettifyProfile(p *pprof.Profile, level PrettifyLevel) {
	if level == PrettifyOff {
		return
	}
	for _, sample := range p.Sample {
		if containsInterpretedPythonStack(sample) {
			newPrettifier(sample, level).prettify()
		}
	}
}

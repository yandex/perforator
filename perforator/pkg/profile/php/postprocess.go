package php

import (
	"strings"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

type NativeAndPHPStackMerger struct {
	sample             *pprof.Sample
	phpStartStackIndex int
	resultStack        []*pprof.Location
}

func NewNativeAndPHPStackMerger() *NativeAndPHPStackMerger {
	return &NativeAndPHPStackMerger{
		resultStack: make([]*pprof.Location, 0, 512),
	}
}

func (m *NativeAndPHPStackMerger) reset(sample *pprof.Sample) {
	m.sample = sample
	m.phpStartStackIndex = -1
	m.resultStack = m.resultStack[:0]
}

func (m *NativeAndPHPStackMerger) cleanup() {
	m.sample = nil
}

const (
	executeExFunctionName = "execute_ex"
)

// isExecuteEx returns true if the location contains the execute_ex function —
// the main PHP VM interpreter loop. Each execute_ex frame in the native stack
// https://github.com/php/php-src/blob/d34c840b4ec5ead2ba44a608d990d0df5b6bbe76/Zend/zend_vm_def.h#L4369
// corresponds to one PHP frame.
func isExecuteEx(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil && (line.Function.Name == executeExFunctionName || line.Function.SystemName == executeExFunctionName) {
			return true
		}
	}
	return false
}

// isPHPInterpreterFrame returns true if the location is part of the PHP VM
// interpreter: either execute_ex itself or a ZEND_* opcode handler.
// These frames are replaced/stripped during merging.
func isPHPInterpreterFrame(loc *pprof.Location) bool {
	for _, line := range loc.Line {
		if line.Function != nil && (line.Function.Name == executeExFunctionName || line.Function.SystemName == executeExFunctionName || strings.HasPrefix(line.Function.Name, "ZEND_") || strings.HasPrefix(line.Function.SystemName, "ZEND_")) {
			return true
		}
	}
	return false
}

func isPHPLocation(loc *pprof.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(profile.PHPSpecialMapping)
}

func isKernelLocation(loc *pprof.Location) bool {
	return loc.Mapping != nil && loc.Mapping.File == string(profile.KernelSpecialMapping)
}

func (m *NativeAndPHPStackMerger) setStartPHPStackIndex() (foundPHPStack bool) {
	if len(m.sample.Location) == 0 {
		return false
	}

	if !isPHPLocation(m.sample.Location[0]) {
		return false
	}

	for i, loc := range m.sample.Location {
		if !isPHPLocation(loc) {
			m.phpStartStackIndex = i - 1
			return true
		}
	}

	m.phpStartStackIndex = len(m.sample.Location) - 1
	return true
}

// MergeStacks merges PHP interpreted stack with native (eBPF-collected) stack inplace.
//
// Stack layout follows pprof convention: Location[0] is the leaf (top of stack).
// The input stack has the following structure:
//
//	[0..P]       PHP frames     (mapping=[php], collected from PHP runtime)
//	[P+1..K]     kernel frames  (mapping=[kernel], optional, present on page faults/syscalls)
//	[K+1..N]     native userspace frames (containing execute_ex and ZEND_* interpreter frames)
//
// The algorithm walks native frames from leaf to root, replacing each execute_ex
// with the corresponding PHP frame. Standalone ZEND_* opcode handlers are stripped.
// If there are more PHP frames than execute_ex frames (common due to eBPF stack depth
// limits), remaining PHP frames are inserted after the last matched execute_ex.
//
// This correctly handles interleaved stacks (php → C → php), e.g. array_map with callback:
//
//	Input:
//	  PHP:    | B | A |
//	  native: | execute_ex | ZEND_DO_FCALL | zif_array_map | execute_ex | zend_execute | main |
//
//	Output:
//	  | B | zif_array_map | A | zend_execute | main |
//
// Deep recursion with single execute_ex (eBPF stack depth limit):
//
//	Input:
//	  PHP:    | heavyComputation x128 |
//	  kernel: | async_page_fault | __do_page_fault |
//	  native: | execute_ex | zend_execute | main | _start |
//
//	Output:
//	  | async_page_fault | __do_page_fault | heavyComputation x128 | zend_execute | main | _start |
//
// If no execute_ex is found in the native stack, PHP frames are inserted after kernel frames.
func (m *NativeAndPHPStackMerger) MergeStacks(s *pprof.Sample) {
	m.reset(s)
	if m.sample == nil {
		return
	}
	defer m.cleanup()

	if !m.setStartPHPStackIndex() {
		return
	}

	locs := m.sample.Location
	nativeStart := m.phpStartStackIndex + 1
	phpCount := m.phpStartStackIndex + 1

	// Kernel frames go first (closest to leaf in merged result).
	kernelEnd := nativeStart
	for kernelEnd < len(locs) && isKernelLocation(locs[kernelEnd]) {
		kernelEnd++
	}
	for i := nativeStart; i < kernelEnd; i++ {
		m.resultStack = append(m.resultStack, locs[i])
	}

	// Count execute_ex frames to know when we've seen the last one.
	executeExCount := 0
	for i := kernelEnd; i < len(locs); i++ {
		if isExecuteEx(locs[i]) {
			executeExCount++
		}
	}

	if executeExCount == 0 {
		// No execute_ex found (likely eBPF stack depth truncation).
		// Native frames are the leaf-side remainder of the truncated stack,
		// so they go before PHP frames.
		for i := kernelEnd; i < len(locs); i++ {
			if !isPHPInterpreterFrame(locs[i]) {
				m.resultStack = append(m.resultStack, locs[i])
			}
		}
		for i := 0; i < phpCount; i++ {
			m.resultStack = append(m.resultStack, locs[i])
		}
	} else {
		phpIdx := 0
		executeExSeen := 0

		for i := kernelEnd; i < len(locs); i++ {
			if isExecuteEx(locs[i]) {
				executeExSeen++
				// Replace execute_ex with corresponding PHP frame.
				if phpIdx < phpCount {
					m.resultStack = append(m.resultStack, locs[phpIdx])
					phpIdx++
				}
				// After last execute_ex, flush remaining unmatched PHP frames.
				if executeExSeen == executeExCount {
					for phpIdx < phpCount {
						m.resultStack = append(m.resultStack, locs[phpIdx])
						phpIdx++
					}
				}
				continue
			}

			// Skip standalone ZEND_* handlers (interpreter dispatch noise).
			if isPHPInterpreterFrame(locs[i]) {
				continue
			}

			m.resultStack = append(m.resultStack, locs[i])
		}
	}

	m.sample.Location = m.sample.Location[:0]
	m.sample.Location = append(m.sample.Location, m.resultStack...)
}

func Postprocess(p *pprof.Profile) {
	merger := NewNativeAndPHPStackMerger()
	for _, sample := range p.Sample {
		merger.MergeStacks(sample)
	}
}

// Strip removes all synthetic [php] interpreter frames from every sample,
// leaving the native stack untouched. Used to hide PHP entirely while
// on-agent PHP unwinding is broken.
func Strip(p *pprof.Profile) {
	for _, sample := range p.Sample {
		out := sample.Location[:0]
		for _, loc := range sample.Location {
			if !isPHPLocation(loc) {
				out = append(out, loc)
			}
		}
		sample.Location = out
	}
}

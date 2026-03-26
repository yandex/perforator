package php

import (
	"slices"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

func phpLoc(funcName string) *pprof.Location {
	return &pprof.Location{
		Mapping: &pprof.Mapping{File: string(profile.PHPSpecialMapping)},
		Line: []pprof.Line{
			{Function: &pprof.Function{Name: funcName}},
		},
	}
}

func kernelLoc(funcName string) *pprof.Location {
	return &pprof.Location{
		Mapping: &pprof.Mapping{File: string(profile.KernelSpecialMapping)},
		Line: []pprof.Line{
			{Function: &pprof.Function{Name: funcName}},
		},
	}
}

func nativeLoc(funcName string) *pprof.Location {
	return &pprof.Location{
		Line: []pprof.Line{
			{Function: &pprof.Function{Name: funcName}},
		},
	}
}

func locationNames(locs []*pprof.Location) []string {
	names := make([]string, len(locs))
	for i, loc := range locs {
		if len(loc.Line) > 0 && loc.Line[0].Function != nil {
			names[i] = loc.Line[0].Function.Name
		} else {
			names[i] = "<no function>"
		}
	}
	return names
}

func TestMergeStacks(t *testing.T) {
	merger := NewNativeAndPHPStackMerger()

	for _, test := range []struct {
		name string
		// Stacks are written root-to-leaf (left=root, right=leaf) for readability.
		// They are reversed before passing to MergeStacks.
		sample       *pprof.Sample
		resultSample *pprof.Sample // nil means sample should not change
	}{
		{
			name: "deep_recursion_with_kernel",
			// 3 PHP frames (representing 128 in real life), kernel interrupt, single execute_ex.
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					nativeLoc("execute_ex"),
					kernelLoc("__do_page_fault"),
					kernelLoc("async_page_fault"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					kernelLoc("__do_page_fault"),
					kernelLoc("async_page_fault"),
				},
			},
		},
		{
			name: "interleaved_php_c_php_array_map",
			// PHP A calls array_map with callback B.
			// In pprof leaf-first order: B(leaf), A(root).
			// Root execute_ex → A, leaf execute_ex → B.
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					nativeLoc("execute_ex"),
					nativeLoc("zif_array_map"),
					nativeLoc("ZEND_DO_FCALL"),
					nativeLoc("execute_ex"),
					phpLoc("B"),
					phpLoc("A"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					phpLoc("B"),
					nativeLoc("zif_array_map"),
					phpLoc("A"),
				},
			},
		},
		{
			name: "native_above_execute_ex",
			// zif_log and libm between kernel and execute_ex.
			// PHP[0]=processData (leaf), PHP[1]=log (root) after reverse.
			// execute_ex maps to processData, native frames above it stay above.
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					nativeLoc("execute_ex"),
					nativeLoc("zif_log"),
					nativeLoc("__ieee754_log_fma"),
					phpLoc("log"),
					phpLoc("processData"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					phpLoc("log"),
					phpLoc("processData"),
					nativeLoc("zif_log"),
					nativeLoc("__ieee754_log_fma"),
				},
			},
		},
		{
			name: "truncated_stack_no_execute_ex",
			// eBPF hit depth limit — no execute_ex captured.
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("__munmap"),
					kernelLoc("apic_timer_interrupt"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					nativeLoc("__munmap"),
					kernelLoc("apic_timer_interrupt"),
				},
			},
		},
		{
			name: "only_native_no_php",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("foo"),
				},
			},
			resultSample: nil, // unchanged
		},
		{
			name: "only_php_frames",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					phpLoc("main"),
					phpLoc("foo"),
					phpLoc("bar"),
				},
			},
			resultSample: nil, // unchanged — no native frames to merge with
		},
		{
			name: "kernel_and_native_above_execute_ex",
			// Kernel interrupt during __munmap called from within execute_ex.
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					nativeLoc("execute_ex"),
					nativeLoc("zend_mm_chunk_free"),
					nativeLoc("__munmap"),
					kernelLoc("release_pages"),
					kernelLoc("entry_SYSCALL_64"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					phpLoc("heavyComputation"),
					nativeLoc("zend_mm_chunk_free"),
					nativeLoc("__munmap"),
					kernelLoc("release_pages"),
					kernelLoc("entry_SYSCALL_64"),
				},
			},
		},
		{
			name: "multiple_execute_ex_more_php_frames",
			// 2 execute_ex but 4 PHP frames (inner recursion truncated by eBPF).
			// After reverse: A(leaf), B, C, D(root) | execute_ex, ZEND_DO_FCALL, zif_array_map, execute_ex, ...
			// First execute_ex (leaf-side) → A, second (root-side) → B + flush C,D.
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					nativeLoc("execute_ex"),
					nativeLoc("zif_array_map"),
					nativeLoc("ZEND_DO_FCALL"),
					nativeLoc("execute_ex"),
					phpLoc("D"),
					phpLoc("C"),
					phpLoc("B"),
					phpLoc("A"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					nativeLoc("_start"),
					nativeLoc("main"),
					nativeLoc("zend_execute"),
					phpLoc("D"),
					phpLoc("C"),
					phpLoc("B"),
					nativeLoc("zif_array_map"),
					phpLoc("A"),
				},
			},
		},
		{
			name: "empty_sample",
			sample: &pprof.Sample{
				Location: []*pprof.Location{},
			},
			resultSample: nil,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			slices.Reverse(test.sample.Location)
			if test.resultSample != nil {
				slices.Reverse(test.resultSample.Location)
			}

			originalLocs := make([]*pprof.Location, len(test.sample.Location))
			copy(originalLocs, test.sample.Location)

			merger.MergeStacks(test.sample)

			expected := test.resultSample
			if expected == nil {
				expected = &pprof.Sample{Location: originalLocs}
			}

			require.Equal(t, locationNames(expected.Location), locationNames(test.sample.Location))
		})
	}
}

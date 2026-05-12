package python

import (
	"slices"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

func createSimpleLocationNative(funcName string, isKernel bool) *pprof.Location {
	loc := &pprof.Location{
		Line: []pprof.Line{
			{
				Function: &pprof.Function{
					Name: funcName,
				},
			},
		},
	}
	if isKernel {
		loc.Mapping = &pprof.Mapping{File: string(profile.KernelSpecialMapping)}
	}

	return loc
}

func createSimpleLocationKernel(funcName string) *pprof.Location {
	return createSimpleLocationNative(funcName, true)
}

func createSimpleLocationUserspace(funcName string) *pprof.Location {
	return createSimpleLocationNative(funcName, false)
}

func createSimpleLocationPython(funcName string) *pprof.Location {
	loc := &pprof.Location{
		Mapping: &pprof.Mapping{File: string(profile.PythonSpecialMapping)},
		Line: []pprof.Line{
			{
				Function: &pprof.Function{
					Name: funcName,
				},
			},
		},
	}

	return loc
}

func createUnsymbolizedLocation() *pprof.Location {
	return &pprof.Location{
		Line: []pprof.Line{
			{
				Function: &pprof.Function{
					Name: "<invalid>",
				},
			},
		},
	}
}

func createImportlibLocation(funcName string) *pprof.Location {
	return &pprof.Location{
		Mapping: &pprof.Mapping{File: string(profile.PythonSpecialMapping)},
		Line: []pprof.Line{
			{
				Function: &pprof.Function{
					Name:     funcName,
					Filename: "<frozen importlib._bootstrap>",
				},
			},
		},
	}
}

func TestMergeStacks_Simple(t *testing.T) {
	merger := NewNativeAndPythonStackMerger()

	for _, test := range []struct {
		name   string
		sample *pprof.Sample
		// if resultSample is nil, then we expect that the sample is not changed
		resultSample   *pprof.Sample
		performedMerge bool
	}{
		{
			name: "busyloop_release",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: true,
		},
		{
			name: "busyloop2_release",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: true,
		},
		{
			name: "busyloop1_debug",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("Py_XDECREF"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("Py_XDECREF"),
				},
			},
			performedMerge: true,
		},
		{
			name: "busyloop2_debug",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("_Py_DECREF_SPECIALIZED"),
					createSimpleLocationUserspace("_PyInterpreterState_GET"),
					createSimpleLocationUserspace("_PyThreadState_GET"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("pymain"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace("callmethod"),
					createSimpleLocationUserspace("_PyObject_CallFunctionVa"),
					createSimpleLocationUserspace("_PyObject_CallNoArgsTstate"),
					createSimpleLocationUserspace("_PyObject_VectorcallTstate"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationUserspace("_PyEval_Vector"),
					createSimpleLocationUserspace("_PyEval_EvalFrame"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("_Py_DECREF_SPECIALIZED"),
					createSimpleLocationUserspace("_PyInterpreterState_GET"),
					createSimpleLocationUserspace("_PyThreadState_GET"),
				},
			},
			performedMerge: true,
		},
		{
			name: "only_native",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
				},
			},
			performedMerge: false,
		},
		{
			name: "incorrect",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: false,
		},
		{
			name: "trim_last_cpython_substack",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("PyObject_CallMethod"),
					createSimpleLocationUserspace(invalid),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace(invalid),
				},
			},
			performedMerge: true,
		},
		{
			name: "python_stack_before_kernel_stack_on_failed_merge",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("foo"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: false,
		},
		{
			name: "one_to_one_python_frame_to_pyeval_cpython_3_10",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: true,
		},
		{
			name: "one_to_one_python_frame_to_pyeval_cpython_3_2",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("foo"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: true,
		},
		{
			name: "one_to_one_python_frame_to_pyeval_cpython_3_2_one_more_py_eval",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("__libc_start_main"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("run_file"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
					createSimpleLocationUserspace("PyRun_SimpleFileExFlags"),
					createSimpleLocationUserspace("PyRun_FileExFlags"),
					createSimpleLocationUserspace("run_mod"),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createSimpleLocationUserspace("PyEval_EvalCodeEx"),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("call_function"),
					createSimpleLocationUserspace("fast_function"),
					createSimpleLocationUserspace("PyEval_EvalFrameEx"),
					createSimpleLocationKernel("apic_timer_interrupt"),
					createSimpleLocationKernel("smp_apic_timer_interrupt"),
				},
			},
			performedMerge: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			slices.Reverse(test.sample.Location)

			originalSample := &pprof.Sample{
				Location: make([]*pprof.Location, len(test.sample.Location)),
			}
			copy(originalSample.Location, test.sample.Location)

			if test.resultSample != nil {
				slices.Reverse(test.resultSample.Location)
			}
			stats, err := merger.MergeStacks(test.sample)
			require.NoError(t, err)

			require.Equal(t, test.performedMerge, stats.PerformedMerge, "Did not perform merge")

			diffSample := originalSample
			if test.resultSample != nil {
				diffSample = test.resultSample
			}

			require.Equal(t, len(diffSample.Location), len(test.sample.Location))
			for i := 0; i < len(diffSample.Location); i++ {
				require.Equal(t, diffSample.Location[i], test.sample.Location[i])
			}
		})
	}
}

func TestPrettifier_RemoveUnsymbolizedCPythonFrames(t *testing.T) {
	for _, test := range []struct {
		name         string
		sample       *pprof.Sample
		resultSample *pprof.Sample
	}{
		{
			name: "surrounded_by_cpython_should_be_removed",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
				},
			},
		},
		{
			name: "not_surrounded_on_right_should_be_kept",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("native_func"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("native_func"),
				},
			},
		},
		{
			name: "not_surrounded_on_left_should_be_kept",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("native_func"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("Py_Main"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("native_func"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("Py_Main"),
				},
			},
		},
		{
			name: "multiple_unsymbolized_frames_should_be_removed",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("PyRun_AnyFileExFlags"),
				},
			},
		},
		{
			name: "surrounded_by_cpython_and_python_should_be_removed",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createUnsymbolizedLocation(),
					createSimpleLocationPython("python_func"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationPython("python_func"),
				},
			},
		},
		{
			name: "at_start_of_stack_should_be_kept",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("Py_Main"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("Py_Main"),
				},
			},
		},
		{
			name: "complex_stack_with_interleaved_unsymbolized_segments",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createUnsymbolizedLocation(),                              // Unsymbolized (keep - not preceded by CPython or Python)
					createUnsymbolizedLocation(),                              // Unsymbolized (keep - not preceded by CPython or Python)
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"), // CPython
					createUnsymbolizedLocation(),                              // Unsymbolized (remove - surrounded by CPython or Python)
					createUnsymbolizedLocation(),                              // Unsymbolized (remove - surrounded by CPython or Python)
					createSimpleLocationUserspace("PyObject_Call"),            // CPython
					createSimpleLocationPython("python_func_a"),               // Python
					createUnsymbolizedLocation(),                              // Unsymbolized (remove)
					createSimpleLocationPython("python_func_b"),               // Python
					createUnsymbolizedLocation(),                              // Unsymbolized (keep - followed by native)
					createSimpleLocationUserspace("_start"),                   // Native
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
					createSimpleLocationUserspace("PyObject_Call"),
					createSimpleLocationPython("python_func_a"),
					createSimpleLocationPython("python_func_b"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("_start"),
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			slices.Reverse(test.sample.Location)
			slices.Reverse(test.resultSample.Location)

			prettifier := newPrettifier(test.sample, PrettifyOff)
			prettifier.removeUnsymbolizedCPythonFrames()

			require.Equal(t, len(test.resultSample.Location), len(test.sample.Location), "Sample length should be updated")

			for i := 0; i < len(test.resultSample.Location); i++ {
				require.Equal(t, test.resultSample.Location[i].Line[0].Function.Name, test.sample.Location[i].Line[0].Function.Name, "Mismatch at index %d", i)
			}
		})
	}
}

func TestPrettifier_Prettify(t *testing.T) {
	for _, test := range []struct {
		name         string
		sample       *pprof.Sample
		resultSample *pprof.Sample
	}{
		{
			name: "full_prettification_flow",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_PyEval_EvalFrameDefault"), // CPython - to remove
					createUnsymbolizedLocation(),                              // Unsymbolized (remove)
					createSimpleLocationUserspace("PyObject_Call"),            // CPython - to remove
					createSimpleLocationPython("python_func_a"),               // Python
					createUnsymbolizedLocation(),                              // Unsymbolized (remove)
					createSimpleLocationPython("python_func_b"),               // Python
					createUnsymbolizedLocation(),                              // Unsymbolized (keep - followed by native)
					createSimpleLocationUserspace("_start"),                   // Native
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("python_func_a"),
					createSimpleLocationPython("python_func_b"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("_start"),
				},
			},
		},
		{
			name: "ml_stack_prettification",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_RunMain"),
					createSimpleLocationUserspace("_PyRun_AnyFileObject"),
					createSimpleLocationUserspace("_PyRun_SimpleFileObject"),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createUnsymbolizedLocation(),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("_PyObject_MakeTpCall"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("_PyObject_Call_Prepend"),
					createSimpleLocationUserspace("_PyObject_FastCallDictTstate"),
					createSimpleLocationPython("_wrapped_call_impl"),
					createUnsymbolizedLocation(),
					createSimpleLocationPython("_call_impl"),
					createUnsymbolizedLocation(),
					createSimpleLocationPython("forward"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("mse_loss"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("broadcast_tensors"),
					createSimpleLocationUserspace("_PyObject_MakeTpCall"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("torch::autograd::THPVariable_broadcast_tensors(_object*, _object*, _object*)"),
					createSimpleLocationUserspace("at::_ops::broadcast_tensors::call(c10::ArrayRef<at::Tensor>)"),
					createSimpleLocationUserspace("c10::impl::OperatorEntry::lookup(c10::DispatchKeySet) const"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("_wrapped_call_impl"),
					createSimpleLocationPython("_call_impl"),
					createSimpleLocationPython("forward"),
					createSimpleLocationPython("mse_loss"),
					createSimpleLocationPython("broadcast_tensors"),
					createSimpleLocationUserspace("_PyObject_MakeTpCall"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("torch::autograd::THPVariable_broadcast_tensors(_object*, _object*, _object*)"),
					createSimpleLocationUserspace("at::_ops::broadcast_tensors::call(c10::ArrayRef<at::Tensor>)"),
					createSimpleLocationUserspace("c10::impl::OperatorEntry::lookup(c10::DispatchKeySet) const"),
				},
			},
		},
		{
			name: "busyloop_prettification_keep_last_cpython_location",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_RunMain"),
					createSimpleLocationUserspace("_PyRun_AnyFileObject"),
					createSimpleLocationUserspace("_PyRun_SimpleFileObject"),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createUnsymbolizedLocation(),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("PyNumber_InPlaceAdd"),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationUserspace("PyNumber_InPlaceAdd"),
				},
			},
		},
		{
			name: "python_cpp_python_prettification",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("Py_Main"),
					createSimpleLocationUserspace("_PyRun_AnyFileObject"),
					createSimpleLocationUserspace("_PyRun_SimpleFileObject"),
					createUnsymbolizedLocation(),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("PyEval_EvalCode"),
					createUnsymbolizedLocation(),
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("main"),
					createSimpleLocationUserspace("_PyFunction_Vectorcall"),
					createSimpleLocationPython("simple"),
					createSimpleLocationUserspace("_PyObject_MakeTpCall"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("cpp_function"),
					createSimpleLocationPython("trampoline python frame"),
					createSimpleLocationPython("python_func_called_from_cpp"),
					createUnsymbolizedLocation(),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createUnsymbolizedLocation(),
					createSimpleLocationUserspace("cpp_function"),
					createSimpleLocationPython("trampoline python frame"),
					createSimpleLocationPython("python_func_called_from_cpp"),
				},
			},
		},
		{
			name: "cython_importlib_trampoline_removal",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("<module>"),
					createSimpleLocationUserspace("__Pyx_PyObject_Call"),    // Cython - should be removed
					createSimpleLocationUserspace("__pyx_pw_6module_1func"), // Cython - should be removed
					createSimpleLocationPython("main"),
					createImportlibLocation("_find_and_load"),               // importlib - should be removed
					createImportlibLocation("_load_unlocked"),               // importlib - should be removed
					createSimpleLocationPython("<trampoline python frame>"), // trampoline - should be removed
					createSimpleLocationPython("user_func"),
					createSimpleLocationUserspace("native_func"),
				},
			},
			resultSample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("<module>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("user_func"),
					createSimpleLocationUserspace("native_func"),
				},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			slices.Reverse(test.sample.Location)
			slices.Reverse(test.resultSample.Location)

			prettifier := newPrettifier(test.sample, PrettifyMixed)
			prettifier.prettify()

			require.Equal(t, len(test.resultSample.Location), len(test.sample.Location), "Sample length should be updated")

			for i := 0; i < len(test.resultSample.Location); i++ {
				require.Equal(t, test.resultSample.Location[i].Line[0].Function.Name, test.sample.Location[i].Line[0].Function.Name, "Mismatch at index %d", i)
			}
		})
	}
}

func TestPrettify_Off(t *testing.T) {
	sample := &pprof.Sample{
		Location: []*pprof.Location{
			createSimpleLocationUserspace("_start"),
			createSimpleLocationUserspace("_PyEval_EvalFrameDefault"),
			createSimpleLocationPython("python_func"),
			createSimpleLocationUserspace("Py_RunMain"),
		},
	}
	original := make([]*pprof.Location, len(sample.Location))
	copy(original, sample.Location)

	newPrettifier(sample, PrettifyOff).prettify()

	require.Equal(t, len(original), len(sample.Location), "Prettify(Off) must not change sample")
	for i := range original {
		require.Equal(t, original[i].Line[0].Function.Name, sample.Location[i].Line[0].Function.Name)
	}
}

func TestRemoveAllNonPythonLocations(t *testing.T) {
	for _, test := range []struct {
		name          string
		sample        *pprof.Sample
		expectedNames []string
	}{
		{
			name: "keeps_only_python_locations",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("handle_request"),
					createSimpleLocationUserspace("at::native::add_kernel"),
					createSimpleLocationPython("forward"),
					createSimpleLocationUserspace("_PyObject_Vectorcall"),
					createSimpleLocationKernel("__sys_epoll_wait"),
				},
			},
			expectedNames: []string{"handle_request", "forward"},
		},
		{
			name: "all_native_returns_empty",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationUserspace("_start"),
					createSimpleLocationUserspace("main"),
					createSimpleLocationKernel("__sys_read"),
				},
			},
			expectedNames: []string{},
		},
		{
			name: "all_python_returns_all",
			sample: &pprof.Sample{
				Location: []*pprof.Location{
					createSimpleLocationPython("func_a"),
					createSimpleLocationPython("func_b"),
				},
			},
			expectedNames: []string{"func_a", "func_b"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := newPrettifier(test.sample, PrettifyOff)
			p.removeAllNonPythonLocations()

			require.Equal(t, len(test.expectedNames), len(test.sample.Location))
			for i, name := range test.expectedNames {
				require.Equal(t, name, test.sample.Location[i].Line[0].Function.Name)
			}
		})
	}
}

func TestPrettify_StrictPython_OnMixedStack(t *testing.T) {
	// Simulate a stack with Python frames and user native frames (e.g. PyTorch).
	// Default should keep native; StrictPython should remove them.
	buildSample := func() *pprof.Sample {
		return &pprof.Sample{
			Location: []*pprof.Location{
				createSimpleLocationPython("model.forward"),
				createSimpleLocationUserspace("_PyObject_Vectorcall"),   // CPython internal — removed in both
				createSimpleLocationUserspace("THPFunction_apply"),      // user native (PyTorch) — kept in Default, removed in Strict
				createSimpleLocationUserspace("at::native::add_kernel"), // user native — kept in Default, removed in Strict
				createSimpleLocationPython("handle_request"),
			},
		}
	}

	// Mixed: CPython internals removed, user native kept.
	sampleDefault := buildSample()
	slices.Reverse(sampleDefault.Location)
	newPrettifier(sampleDefault, PrettifyMixed).prettify()
	defaultNames := make([]string, len(sampleDefault.Location))
	for i, loc := range sampleDefault.Location {
		defaultNames[i] = loc.Line[0].Function.Name
	}
	require.Contains(t, defaultNames, "THPFunction_apply", "Mixed must keep user native frames")
	require.Contains(t, defaultNames, "at::native::add_kernel", "Mixed must keep user native frames")
	require.Contains(t, defaultNames, "model.forward")
	require.Contains(t, defaultNames, "handle_request")
	require.NotContains(t, defaultNames, "_PyObject_Vectorcall", "Mixed must remove CPython internals")

	// PythonOnly: only python frames
	sampleStrict := buildSample()
	slices.Reverse(sampleStrict.Location)
	newPrettifier(sampleStrict, PrettifyPythonOnly).prettify()
	for _, loc := range sampleStrict.Location {
		require.True(t, isPythonLocation(loc), "PythonOnly must leave only Python locations, got: %s", loc.Line[0].Function.Name)
	}
	strictNames := make([]string, len(sampleStrict.Location))
	for i, loc := range sampleStrict.Location {
		strictNames[i] = loc.Line[0].Function.Name
	}
	require.Contains(t, strictNames, "model.forward")
	require.Contains(t, strictNames, "handle_request")
	require.NotContains(t, strictNames, "THPFunction_apply", "PythonOnly must remove user native frames")
	require.NotContains(t, strictNames, "at::native::add_kernel", "PythonOnly must remove user native frames")
}

func TestPrettifyProfile_NonPythonProfile(t *testing.T) {
	// A profile with no Python frames must not be modified by PrettifyProfile at any level.
	buildProfile := func() *pprof.Profile {
		return &pprof.Profile{
			Sample: []*pprof.Sample{
				{
					Location: []*pprof.Location{
						createSimpleLocationUserspace("_start"),
						createSimpleLocationUserspace("main"),
						createSimpleLocationUserspace("do_work"),
					},
				},
			},
		}
	}

	for _, level := range []PrettifyLevel{PrettifyOff, PrettifyMixed, PrettifyPythonOnly} {
		p := buildProfile()
		original := p.Sample[0].Location
		PrettifyProfile(p, level)
		require.Equal(t, len(original), len(p.Sample[0].Location), "non-Python profile must not be modified at level %d", level)
	}
}

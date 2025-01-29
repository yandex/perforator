package python

import (
	"slices"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/linguist/python/models"
)

func createSimpleLocationNative(funcName string) *profile.Location {
	loc := &profile.Location{
		Line: []profile.Line{
			{
				Function: &profile.Function{
					Name: funcName,
				},
			},
		},
	}

	return loc
}

func createSimpleLocationPython(funcName string) *profile.Location {
	loc := &profile.Location{
		Mapping: &profile.Mapping{File: string(models.PythonSpecialMapping)},
		Line: []profile.Line{
			{
				Function: &profile.Function{
					Name: funcName,
				},
			},
		},
	}

	return loc
}

func TestMergeStacks_Simple(t *testing.T) {
	merger := NewNativeAndPythonStackMerger()

	for _, test := range []struct {
		name           string
		sample         *profile.Sample
		resultSample   *profile.Sample
		performedMerge bool
		containsPython bool
	}{
		{
			name: "busyloop_release",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationNative(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "busyloop2_release",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "busyloop1_debug",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative("pymain"),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative("callmethod"),
					createSimpleLocationNative("_PyObject_CallFunctionVa"),
					createSimpleLocationNative("_PyObject_CallNoArgsTstate"),
					createSimpleLocationNative("_PyObject_VectorcallTstate"),
					createSimpleLocationNative("_PyFunction_Vectorcall"),
					createSimpleLocationNative("_PyEval_Vector"),
					createSimpleLocationNative("_PyEval_EvalFrame"),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationNative("Py_XDECREF"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative("pymain"),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative("callmethod"),
					createSimpleLocationNative("_PyObject_CallFunctionVa"),
					createSimpleLocationNative("_PyObject_CallNoArgsTstate"),
					createSimpleLocationNative("_PyObject_VectorcallTstate"),
					createSimpleLocationNative("_PyFunction_Vectorcall"),
					createSimpleLocationNative("_PyEval_Vector"),
					createSimpleLocationNative("_PyEval_EvalFrame"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationNative("Py_XDECREF"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "busyloop2_debug",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative("pymain"),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative("callmethod"),
					createSimpleLocationNative("_PyObject_CallFunctionVa"),
					createSimpleLocationNative("_PyObject_CallNoArgsTstate"),
					createSimpleLocationNative("_PyObject_VectorcallTstate"),
					createSimpleLocationNative("_PyFunction_Vectorcall"),
					createSimpleLocationNative("_PyEval_Vector"),
					createSimpleLocationNative("_PyEval_EvalFrame"),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationNative("_Py_DECREF_SPECIALIZED"),
					createSimpleLocationNative("_PyInterpreterState_GET"),
					createSimpleLocationNative("_PyThreadState_GET"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative("pymain"),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative("callmethod"),
					createSimpleLocationNative("_PyObject_CallFunctionVa"),
					createSimpleLocationNative("_PyObject_CallNoArgsTstate"),
					createSimpleLocationNative("_PyObject_VectorcallTstate"),
					createSimpleLocationNative("_PyFunction_Vectorcall"),
					createSimpleLocationNative("_PyEval_Vector"),
					createSimpleLocationNative("_PyEval_EvalFrame"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationNative("_Py_DECREF_SPECIALIZED"),
					createSimpleLocationNative("_PyInterpreterState_GET"),
					createSimpleLocationNative("_PyThreadState_GET"),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
		{
			name: "only_native",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative("foo"),
				},
			},
			performedMerge: false,
			containsPython: false,
		},
		{
			name: "incorrect",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative("foo"),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			performedMerge: false,
			containsPython: true,
		},
		{
			name: "trim_last_cpython_substack",
			sample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationNative(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
				},
			},
			resultSample: &profile.Sample{
				Location: []*profile.Location{
					createSimpleLocationNative("_start"),
					createSimpleLocationNative("__libc_start_main"),
					createSimpleLocationNative("main"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationPython("<trampoline python frame>"),
					createSimpleLocationPython("main"),
					createSimpleLocationPython("simple"),
					createSimpleLocationPython("foo"),
					createSimpleLocationNative("PyObject_CallMethod"),
					createSimpleLocationNative(invalid),
					createSimpleLocationNative("_PyEval_EvalFrameDefault"),
					createSimpleLocationNative(invalid),
				},
			},
			performedMerge: true,
			containsPython: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			slices.Reverse(test.sample.Location)
			if test.resultSample != nil {
				slices.Reverse(test.resultSample.Location)
			}
			stats, err := merger.MergeStacks(test.sample)
			require.NoError(t, err)

			require.Equal(t, test.performedMerge, stats.PerformedMerge, "Did not perform merge")
			require.Equal(t, test.containsPython, stats.CollectedPython, "Did not collect python")

			if test.performedMerge {
				require.Equal(t, len(test.resultSample.Location), len(test.sample.Location))
				for i := 0; i < len(test.resultSample.Location); i++ {
					require.Equal(t, test.resultSample.Location[i], test.sample.Location[i])
				}
			}
		})
	}
}

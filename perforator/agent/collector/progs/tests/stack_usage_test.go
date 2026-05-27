package bpf_test

import (
	"bytes"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/ebpf/stackusage"
)

const maxStackSize = 512

func testProg(t *testing.T, reqs unwinder.ProgramRequirements) {
	prog, err := unwinder.LoadProg(reqs)
	require.NoError(t, err)
	spec, err := ebpf.LoadCollectionSpecFromReader(bytes.NewReader(prog))
	require.NoError(t, err)

	for progname, prog := range spec.Programs {
		t.Run(progname, func(t *testing.T) {
			usage, comment, err := stackusage.StackUsage(prog)
			require.NoError(t, err)
			t.Log(comment)
			t.Logf("Stack usage: %d bytes", usage)
			assert.LessOrEqual(t, usage, maxStackSize)
		})
	}
}

func TestStackUsage(t *testing.T) {
	t.Run("Debug", func(t *testing.T) {
		testProg(t, unwinder.ProgramRequirements{Debug: true})
	})
	t.Run("Release", func(t *testing.T) {
		testProg(t, unwinder.ProgramRequirements{})
	})
	t.Run("DebugKernel5_4", func(t *testing.T) {
		testProg(t, unwinder.ProgramRequirements{Debug: true, KernelCompatibility: unwinder.KernelCompatibilityLevel5_4})
	})
	t.Run("ReleaseKernel5_4", func(t *testing.T) {
		testProg(t, unwinder.ProgramRequirements{KernelCompatibility: unwinder.KernelCompatibilityLevel5_4})
	})
}

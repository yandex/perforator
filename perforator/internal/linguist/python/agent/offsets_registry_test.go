package agent

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	testBinaryID      = uint64(7)
	testOtherBinaryID = uint64(8)
	testBaseAddress   = uint64(0x100000)
	testPid           = linux.CurrentNamespacePID(4242)
)

func testMappingIdentity(binaryID uint64) pythonMappingIdentity {
	return pythonMappingIdentity{
		binaryID:    binaryID,
		baseAddress: testBaseAddress,
	}
}

func supportedAnalysis() *parse.BinaryAnalysis {
	return supportedAnalysisForVersion(3, 12, 10)
}

func supportedAnalysisForVersion(major, minor, micro uint32) *parse.BinaryAnalysis {
	return &parse.BinaryAnalysis{
		PythonConfig: &python.PythonConfig{
			Version: &python.PythonVersion{Major: major, Minor: minor, Micro: micro},
		},
	}
}

func newTestOffsets() *offsetsRegistry {
	return newOffsetsRegistry(xlog.NewNop())
}

func TestOffsets_OnBinaryDiscovery_SupportedVersion(t *testing.T) {
	r := newTestOffsets()
	r.OnBinaryDiscovery(context.Background(), testBinaryID, "build-id", supportedAnalysis())

	changed := r.updateBinding(context.Background(), testPid, testMappingIdentity(testBinaryID), true, false)
	require.True(t, changed)

	offsets, ok := r.OffsetsForPid(uint32(testPid))
	require.True(t, ok)
	require.NotNil(t, offsets)
}

func TestOffsets_OnBinaryDiscovery_IgnoresNonPython(t *testing.T) {
	r := newTestOffsets()

	r.OnBinaryDiscovery(context.Background(), testBinaryID, "build-id", nil)
	r.OnBinaryDiscovery(context.Background(), testBinaryID, "build-id", &parse.BinaryAnalysis{})

	changed := r.updateBinding(context.Background(), testPid, testMappingIdentity(testBinaryID), true, false)
	require.False(t, changed)
	_, ok := r.OffsetsForPid(uint32(testPid))
	require.False(t, ok)
}

func TestOffsets_OnBinaryDiscovery_UnsupportedVersion(t *testing.T) {
	r := newTestOffsets()
	analysis := &parse.BinaryAnalysis{
		PythonConfig: &python.PythonConfig{
			Version: &python.PythonVersion{Major: 3, Minor: 99, Micro: 0},
		},
	}

	r.OnBinaryDiscovery(context.Background(), testBinaryID, "build-id", analysis)

	changed := r.updateBinding(context.Background(), testPid, testMappingIdentity(testBinaryID), true, false)
	require.False(t, changed)
	_, ok := r.OffsetsForPid(uint32(testPid))
	require.False(t, ok)
}

func TestOffsets_Bind_RetriesAfterBinaryDiscovery(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	changed := r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false)
	require.False(t, changed)
	_, ok := r.OffsetsForPid(uint32(testPid))
	require.False(t, ok, "pid must stay unbound while the binary is unknown")

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	changed = r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false)
	require.True(t, changed)

	_, ok = r.OffsetsForPid(uint32(testPid))
	require.True(t, ok, "rescan must bind the pid once the binary is known")
}

func TestOffsets_OnProcessDeath_RemovesBinding(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false)
	_, ok := r.OffsetsForPid(uint32(testPid))
	require.True(t, ok)

	r.OnProcessDeath(ctx, testPid)

	_, ok = r.OffsetsForPid(uint32(testPid))
	require.False(t, ok)
}

func TestOffsets_DiscoveryRefreshesExistingBinding(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id-3.11", supportedAnalysisForVersion(3, 11, 9))
	r.OnBinaryDiscovery(ctx, testOtherBinaryID, "build-id-3.12", supportedAnalysisForVersion(3, 12, 10))
	identity := testMappingIdentity(testBinaryID)
	require.True(t, r.updateBinding(ctx, testPid, identity, true, true))
	require.True(t, r.updateBinding(ctx, testPid, identity, true, true))
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testOtherBinaryID), true, true))

	r.mu.RLock()
	binding := r.processes[testPid]
	r.mu.RUnlock()
	require.Equal(t, testOtherBinaryID, binding.identity.binaryID)
}

func TestOffsets_RescanKeepsSameMappingIdentity(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false))
	require.False(t, r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false))

	_, ok := r.OffsetsForPid(uint32(testPid))
	require.True(t, ok)
}

func TestOffsets_RescanRefreshesSameBinaryAtDifferentBase(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	first := testMappingIdentity(testBinaryID)
	second := first
	second.baseAddress++
	require.True(t, r.updateBinding(ctx, testPid, first, true, false))
	require.True(t, r.updateBinding(ctx, testPid, second, true, false))

	r.mu.RLock()
	binding := r.processes[testPid]
	r.mu.RUnlock()
	require.Equal(t, second, binding.identity)
}

func TestOffsets_RescanReplacesBindingAfterExec(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id-3.11", supportedAnalysisForVersion(3, 11, 9))
	r.OnBinaryDiscovery(ctx, testOtherBinaryID, "build-id-3.12", supportedAnalysisForVersion(3, 12, 10))
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false))
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testOtherBinaryID), true, false))

	r.mu.RLock()
	binding := r.processes[testPid]
	r.mu.RUnlock()
	require.Equal(t, testOtherBinaryID, binding.identity.binaryID)
}

func TestOffsets_RescanWithoutPythonRemovesBinding(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false))
	require.True(t, r.updateBinding(ctx, testPid, pythonMappingIdentity{}, false, false))

	_, ok := r.OffsetsForPid(uint32(testPid))
	require.False(t, ok)
}

func TestOffsets_UnknownBinaryRemovesBindingAndRetries(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, false))
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testOtherBinaryID), true, false))
	_, ok := r.OffsetsForPid(uint32(testPid))
	require.False(t, ok)

	r.OnBinaryDiscovery(ctx, testOtherBinaryID, "other-build-id", supportedAnalysis())
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testOtherBinaryID), true, false))
	_, ok = r.OffsetsForPid(uint32(testPid))
	require.True(t, ok)
}

func TestOffsets_OffsetsForPid_Unknown(t *testing.T) {
	r := newTestOffsets()
	_, ok := r.OffsetsForPid(1)
	require.False(t, ok)
}

func TestOffsets_ConcurrentAccess(t *testing.T) {
	r := newTestOffsets()
	ctx := context.Background()

	const goroutines = 8
	const iterations = 200

	var wg sync.WaitGroup
	wg.Add(goroutines * 4)

	for g := 0; g < goroutines; g++ {
		base := uint64(g * iterations)

		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.OnBinaryDiscovery(ctx, base+uint64(i), "build-id", supportedAnalysis())
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.updateBinding(
					ctx,
					linux.CurrentNamespacePID(base+uint64(i)),
					pythonMappingIdentity{binaryID: base + uint64(i), baseAddress: base + uint64(i)},
					true,
					false,
				)
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.OffsetsForPid(uint32(base + uint64(i)))
			}
		}()

		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				r.OnProcessDeath(ctx, linux.CurrentNamespacePID(base+uint64(i)))
			}
		}()
	}

	wg.Wait()

	// Keep a deterministic state check after the race-detector stress above.
	r.OnBinaryDiscovery(ctx, testBinaryID, "build-id", supportedAnalysis())
	require.True(t, r.updateBinding(ctx, testPid, testMappingIdentity(testBinaryID), true, true))
	_, ok := r.OffsetsForPid(uint32(testPid))
	require.True(t, ok)
	r.OnProcessDeath(ctx, testPid)
	_, ok = r.OffsetsForPid(uint32(testPid))
	require.False(t, ok)
}

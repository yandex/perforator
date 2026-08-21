package agent

import (
	"context"
	"sync"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	pyoffsets "github.com/yandex/perforator/perforator/internal/linguist/python/offsets"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

// offsetsRegistry maps pid → CPython internals offsets for the sample path.
type offsetsRegistry struct {
	l xlog.Logger

	mu        sync.RWMutex
	binaries  map[uint64]*unwinder.PythonInternalsOffsets
	processes map[linux.CurrentNamespacePID]processBinding
}

type pythonMappingIdentity struct {
	binaryID    uint64
	baseAddress uint64
}

type processBinding struct {
	identity pythonMappingIdentity
	offsets  *unwinder.PythonInternalsOffsets
}

func newOffsetsRegistry(l xlog.Logger) *offsetsRegistry {
	return &offsetsRegistry{
		l:         l.WithName("python-offsets-registry"),
		binaries:  make(map[uint64]*unwinder.PythonInternalsOffsets),
		processes: make(map[linux.CurrentNamespacePID]processBinding),
	}
}

func (r *offsetsRegistry) OnBinaryDiscovery(ctx context.Context, binaryID uint64, buildID string, analysis *parse.BinaryAnalysis) {
	if analysis == nil || analysis.PythonConfig == nil {
		return
	}
	offs, err := pyoffsets.PythonInternalsOffsetsByVersion(analysis.PythonConfig.Version)
	if err != nil {
		r.l.Debug(ctx, "Python offsets unavailable for binary",
			log.String("build_id", buildID),
			log.UInt64("binary_id", binaryID),
			log.Error(err),
		)
		return
	}

	r.mu.Lock()
	// The binary listener has no release callback. Binary IDs are monotonic, so
	// these small shared-offset entries intentionally live until agent shutdown.
	r.binaries[binaryID] = offs
	r.mu.Unlock()

	r.l.Debug(ctx, "Registered Python binary offsets",
		log.String("build_id", buildID),
		log.UInt64("binary_id", binaryID),
	)
}

func (r *offsetsRegistry) OnProcessDiscovery(ctx context.Context, info process.ProcessInfo) bool {
	// Discovery starts a newly known process lifetime. Refresh even an existing
	// PID binding so a stale late-rescan result cannot survive until PID reuse.
	return r.refreshProcess(ctx, info, true)
}

func (r *offsetsRegistry) OnProcessRescan(ctx context.Context, info process.ProcessInfo) bool {
	// exec changes mappings without changing the PID, so positive and negative
	// process classification must be rechecked on every rescan.
	return r.refreshProcess(ctx, info, false)
}

func (r *offsetsRegistry) OnProcessDeath(ctx context.Context, pid linux.CurrentNamespacePID) {
	r.mu.Lock()
	delete(r.processes, pid)
	r.mu.Unlock()
}

func (r *offsetsRegistry) OffsetsForPid(pid uint32) (*unwinder.PythonInternalsOffsets, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	binding, ok := r.processes[linux.CurrentNamespacePID(pid)]
	return binding.offsets, ok
}

func (r *offsetsRegistry) refreshProcess(ctx context.Context, info process.ProcessInfo, replaceExisting bool) bool {
	pid := info.ProcessID()
	identity, found := pythonMappingIdentityForProcess(info)
	return r.updateBinding(ctx, pid, identity, found, replaceExisting)
}

func pythonMappingIdentityForProcess(info process.ProcessInfo) (pythonMappingIdentity, bool) {
	for _, m := range info.Mappings() {
		if m.BinaryClass() == dso.PythonBinaryClass {
			return pythonMappingIdentity{
				binaryID:    m.ID(),
				baseAddress: m.BaseAddress(),
			}, true
		}
	}
	return pythonMappingIdentity{}, false
}

func (r *offsetsRegistry) updateBinding(
	ctx context.Context,
	pid linux.CurrentNamespacePID,
	identity pythonMappingIdentity,
	hasPythonMapping bool,
	replaceExisting bool,
) bool {
	r.mu.Lock()
	current, bound := r.processes[pid]

	if !hasPythonMapping {
		delete(r.processes, pid)
		r.mu.Unlock()
		return bound
	}

	// A new load address usually identifies a new mapping incarnation after
	// same-binary exec or fast PID reuse. This is deliberately best-effort: the
	// process listener does not expose an exec generation, and the address can
	// repeat when ASLR is disabled.
	if !replaceExisting && bound && current.identity == identity {
		r.mu.Unlock()
		return false
	}

	offs, ok := r.binaries[identity.binaryID]
	if !ok {
		delete(r.processes, pid)
		r.mu.Unlock()

		// Do not cache this miss: binary analysis can arrive later, and exec can
		// turn a non-Python process into Python without a process-death event.
		r.l.Debug(ctx, "Python binary offsets not ready; will retry on rescan",
			logfield.CurrentNamespacePID(pid),
			log.UInt64("binary_id", identity.binaryID),
		)
		return bound
	}

	r.processes[pid] = processBinding{
		identity: identity,
		offsets:  offs,
	}
	r.mu.Unlock()
	return true
}

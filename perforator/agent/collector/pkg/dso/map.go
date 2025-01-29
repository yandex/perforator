package dso

import (
	"context"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/karlseguin/ccache/v3"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	bpf "github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/parser"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

// DSO stands for Dynamic Shared Object.
// Just any executable binary file.
type dso struct {
	// Unique ID of the DSO. It is used by eBPF.
	ID uint64

	// Build info of the binary.
	buildInfo *xelf.BuildInfo

	// Link to the allocation.
	bpfAllocationMutex sync.Mutex
	bpfAllocation      *bpf.Allocation
}

////////////////////////////////////////////////////////////////////////////////

type refCountedDSO struct {
	*dso
	refCount int32 // Guarded by RWMutex in Registry
	cached   ccache.TrackedItem[*dso]
}

type registryMetrics struct {
	discoveredTLSVariables metrics.Counter
	usedPages              metrics.Gauge
	cachedPages            metrics.Gauge
}

type Registry struct {
	l      xlog.Logger
	dsosmu sync.RWMutex
	dsos   map[string]*refCountedDSO
	nextid atomic.Uint64
	cache  *ccache.Cache[*dso]

	usagemu     sync.Mutex
	usedPages   int
	cachedPages int

	bpfBinaryManager *bpf.BPFBinaryManager
	binaryParser     *parser.BinaryParser

	metrics registryMetrics
}

func NewRegistry(l xlog.Logger, m metrics.Registry, manager *bpf.BPFBinaryManager) (*Registry, error) {
	binaryParser, err := parser.NewBinaryParser(l.WithName("BinaryParser"), m.WithPrefix("BinaryParser"))
	if err != nil {
		return nil, err
	}

	r := &Registry{
		l:                l.WithName("DSORegistry"),
		bpfBinaryManager: manager,
		dsos:             map[string]*refCountedDSO{},
		binaryParser:     binaryParser,
		metrics: registryMetrics{
			discoveredTLSVariables: m.Counter("tls.variables_discovered.count"),
			usedPages:              m.Gauge("unwind_page_table.pages.used.count"),
			cachedPages:            m.Gauge("unwind_page_table.pages.reclaimable.count"),
		},
	}

	r.cache = ccache.New[*dso](
		ccache.
			Configure[*dso]().
			MaxSize(280 * 1024 * 1024).
			OnDelete(r.onDelete).
			Track(),
	)

	return r, nil
}

// updateResourceUsageMetrics updates metrics after change in page usage.
// Preconditions: d.usagemu is locked.
func (d *Registry) updateResourceUsageMetrics(ctx context.Context) {
	if d.cachedPages < 0 {
		d.l.Error(ctx, "cachedPages underflow")
	}
	if d.usedPages < 0 {
		d.l.Error(ctx, "usedPages underflow")
	}
	d.metrics.cachedPages.Set(float64(d.cachedPages))
	d.metrics.usedPages.Set(float64(d.usedPages))
}

func (d *Registry) onPagesAllocated(ctx context.Context, n int) {
	d.usagemu.Lock()
	defer d.usagemu.Unlock()

	d.usedPages += n

	d.updateResourceUsageMetrics(ctx)
}

func (d *Registry) onPagesMovedToCache(ctx context.Context, n int) {
	d.usagemu.Lock()
	defer d.usagemu.Unlock()

	d.usedPages -= n
	d.cachedPages += n

	d.updateResourceUsageMetrics(ctx)
}

func (d *Registry) onPagesRestoredFromCache(ctx context.Context, n int) {
	d.usagemu.Lock()
	defer d.usagemu.Unlock()

	d.cachedPages -= n
	d.usedPages += n

	d.updateResourceUsageMetrics(ctx)
}

func (d *Registry) get(buildID string) *dso {
	d.dsosmu.RLock()
	defer d.dsosmu.RUnlock()
	dso, present := d.dsos[buildID]
	if !present {
		return nil
	}

	return dso.dso
}

func (d *Registry) getMappingCount() int {
	d.dsosmu.RLock()
	defer d.dsosmu.RUnlock()
	return len(d.dsos)
}

func (d *Registry) trackingFetch(id string, ttl time.Duration, cb func() (*dso, error)) (ccache.TrackedItem[*dso], error) {
	item := d.cache.TrackingGet(id)
	if item != nil {
		// item can be .Expired() now.
		// It is ok to return stale unwind tables.
		return item, nil
	}

	value, err := cb()
	if err != nil {
		return nil, err
	}

	return d.cache.TrackingSet(id, value, ttl), nil
}

func (d *Registry) acquireIfExists(buildID string) *refCountedDSO {
	d.dsosmu.Lock()
	defer d.dsosmu.Unlock()
	dso, present := d.dsos[buildID]
	if present {
		dso.refCount++
	}

	return dso
}

func (d *Registry) ensure(buildID string, newDSO *refCountedDSO) (*refCountedDSO, bool) {
	d.dsosmu.Lock()
	defer d.dsosmu.Unlock()
	dso, present := d.dsos[buildID]
	if present {
		dso.refCount++
		return dso, false
	}

	d.dsos[buildID] = newDSO
	return newDSO, true
}

// Return existing DSO if it exists, otherwise build unwind table and store new DSO
func (d *Registry) register(ctx context.Context, buildInfo *xelf.BuildInfo, file binary.UnsealedFile) (*dso, error) {
	buildID := buildInfo.BuildID

	rcdso := d.acquireIfExists(buildID)
	if rcdso != nil {
		return rcdso.dso, nil
	}

	item, err := d.trackingFetch(buildID, 10*time.Minute, func() (*dso, error) {
		return &dso{
			ID:        d.nextid.Add(1) - 1,
			buildInfo: buildInfo,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	newDSO := &refCountedDSO{
		dso:      item.Value(),
		refCount: 1,
		cached:   item,
	}

	dso, inserted := d.ensure(buildID, newDSO)
	if !inserted {
		item.Release()
		return dso.dso, nil
	}

	if file != nil {
		d.populateDSO(ctx, dso.dso, file.GetFile())
	}

	d.l.Debug(ctx,
		"Processed new DSO",
		log.String("buildid", buildID),
		log.UInt64("id", newDSO.ID),
	)

	return newDSO.dso, nil
}

func (d *Registry) release(ctx context.Context, buildID string) {
	d.cache.Get(buildID) // promote dso in cache

	d.dsosmu.Lock()
	defer d.dsosmu.Unlock()
	dso, present := d.dsos[buildID]
	if !present {
		return
	}

	dso.refCount--
	if dso.refCount == 0 {
		d.l.Debug(ctx, "Release DSO", log.String("buildid", buildID))
		delete(d.dsos, buildID)
		d.freeDSO(ctx, dso.dso)
		dso.cached.Release()
	}
}

func (d *Registry) onDelete(item *ccache.Item[*dso]) {
	dso := item.Value()
	d.maybeReleaseBinary(dso)
	d.l.Debug(context.TODO(), "Delete DSO from cache", log.String("buildid", dso.buildInfo.BuildID))
}

func (d *Registry) maybeReleaseBinary(dso *dso) {
	if d.bpfBinaryManager != nil && dso.bpfAllocation != nil {
		d.bpfBinaryManager.Release(dso.bpfAllocation)
		dso.bpfAllocation = nil
	}
}

func (d *Registry) populateDSO(ctx context.Context, dso *dso, f *os.File) {
	if f == nil || d.bpfBinaryManager == nil {
		return
	}
	buildID := dso.buildInfo.BuildID

	dso.bpfAllocationMutex.Lock()
	defer dso.bpfAllocationMutex.Unlock()

	// Happy path. Our DSO was analyzed previously and unwind tables were cached.
	if alloc := dso.bpfAllocation; alloc != nil {
		if d.bpfBinaryManager.MoveFromCache(alloc) {
			d.onPagesRestoredFromCache(ctx, len(alloc.UnwindTableAllocation.Pages))
			return
		}

		// We had analyzed our DSO previously, but the allocation
		// had been evicted from the BPF DSO unwind table cache.
		// So let's try to release current allocation and create the new one.
		d.bpfBinaryManager.Release(alloc)
		d.l.Debug(ctx, "Removing stale BPF DSO allocation", log.String("buildid", buildID))
	}

	analysis, err := d.binaryParser.Parse(ctx, f)
	if err != nil {
		d.l.Warn(ctx,
			"Failed to build binary analysis",
			log.Error(err),
			log.String("buildid", buildID),
			log.String("filename", f.Name()),
		)
		return
	}

	dso.bpfAllocation, err = d.bpfBinaryManager.Add(buildID, dso.ID, analysis)
	if err != nil {
		d.l.Error(
			ctx,
			"Failed to add binary to bpf binary manager",
			log.String("buildid", buildID),
			log.Error(err),
		)
		return
	}
	d.onPagesAllocated(ctx, len(dso.bpfAllocation.UnwindTableAllocation.Pages))

	if dso.bpfAllocation != nil {
		dso.bpfAllocation.TLSMutex.Lock()
		defer dso.bpfAllocation.TLSMutex.Unlock()
		for _, variable := range analysis.TLSConfig.Variables {
			dso.bpfAllocation.TLSMap[variable.Offset] = variable.Name
			d.l.Info(
				ctx,
				"Extracted tls variables from binary",
				log.String("buildid", buildID),
				log.UInt64("offset", variable.Offset),
				log.String("name", variable.Name),
			)
			d.metrics.discoveredTLSVariables.Inc()
		}
	}
}

func (d *Registry) freeDSO(ctx context.Context, dso *dso) {
	dso.bpfAllocationMutex.Lock()
	defer dso.bpfAllocationMutex.Unlock()

	if dso.bpfAllocation != nil {
		d.onPagesMovedToCache(ctx, len(dso.bpfAllocation.UnwindTableAllocation.Pages))
		d.bpfBinaryManager.MoveToCache(dso.bpfAllocation)
	}
}

////////////////////////////////////////////////////////////////////////////////

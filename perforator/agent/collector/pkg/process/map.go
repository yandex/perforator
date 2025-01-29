package process

import (
	"context"
	eb "encoding/binary"
	"fmt"
	"math"
	"math/bits"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/upload"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/mountinfo"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/linux/vdso"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

type ProcessRegistry struct {
	log xlog.Logger

	procs   map[linux.ProcessID]*processInfo
	procsmu sync.RWMutex
	// incremented each time new scan starts
	procsGeneration atomic.Uint64
	procchan        chan *processInfo

	listeners []Listener

	buildids   *BuildIDCache
	dsoStorage *dso.Storage
	bpf        *machine.BPF
	mounts     *mountinfo.Watcher

	uploader *upload.Scheduler

	metrics        processRegistryMetrics
	processScanner ProcessScanner
}

type processRegistryMetrics struct {
	mappingsDiscovered           metrics.Counter
	mappingsWithoutBuildID       metrics.Counter
	mappingsJitted               metrics.Counter
	mappingsFailedScheduleUpload metrics.Counter
	mappingsFailedNameToHandleAt metrics.Counter
}

type processMap struct {
	*dso.Mapping
	id uint32
}

type processInfo struct {
	id                linux.ProcessID
	state             processState
	lock              sync.RWMutex
	envs              map[string]string
	listenersNotified atomic.Bool

	// Used for deletion purposes. All modifications happen under r.procsmu in shared or exclusive mode
	generation     atomic.Uint64
	mapsgeneration atomic.Uint64
	nextmapid      atomic.Uint32
	registeredmaps map[procfs.Address]processMap
	mapslock       sync.Mutex
}

func (p *processInfo) setState(state processState) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.state == processStateDeleted && state != processStateDeleted {
		return fmt.Errorf("process %d has already been deleted", p.id)
	}

	p.state = state
	return nil
}

var _ ProcessInfo = (*processInfo)(nil)

// ProcessID implements ProcessInfo
func (p *processInfo) ProcessID() linux.ProcessID {
	return p.id
}

func (p *processInfo) setEnvs(envs map[string]string) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.envs = envs
}

// Env implements ProcessInfo
func (p *processInfo) Env() map[string]string {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.envs
}

type processState int

const (
	processStateUnknown processState = iota
	processStateDiscovered
	processStatePopulating
	processStatePopulated
	processStateDeleted

	ProcScanPeriod = 10 * time.Second
)

type UploaderArguments struct {
	Storage client.BinaryStorage
	Conf    upload.SchedulerConfig
}

////////////////////////////////////////////////////////////////////////////////

func NewProcessRegistry(
	l xlog.Logger,
	m metrics.Registry,
	ebpf *machine.BPF,
	mounts *mountinfo.Watcher,
	dsoStorage *dso.Storage,
	uploaderArgs *UploaderArguments,
	processScanner ProcessScanner,
	listeners []Listener,
) (*ProcessRegistry, error) {
	uploader, err := upload.NewUploadScheduler(
		uploaderArgs.Conf,
		uploaderArgs.Storage,
		l.Logger(),
		m,
	)
	if err != nil {
		return nil, err
	}

	p := &ProcessRegistry{
		log:        l,
		procs:      make(map[linux.ProcessID]*processInfo),
		dsoStorage: dsoStorage,
		bpf:        ebpf,
		procchan:   make(chan *processInfo, 8192),
		buildids:   NewBuildIDCache(),
		uploader:   uploader,
		mounts:     mounts,
		metrics: processRegistryMetrics{
			mappingsDiscovered:           m.WithTags(map[string]string{"kind": "discovered"}).Counter("mappings.count"),
			mappingsWithoutBuildID:       m.WithTags(map[string]string{"kind": "nobuildid"}).Counter("mappings.count"),
			mappingsJitted:               m.WithTags(map[string]string{"kind": "jitted"}).Counter("mappings.count"),
			mappingsFailedScheduleUpload: m.WithTags(map[string]string{"kind": "failed_schedule_upload"}).Counter("mappings.count"),
			mappingsFailedNameToHandleAt: m.WithTags(map[string]string{"kind": "failed_name_to_handle_at"}).Counter("mappings.count"),
		},
		processScanner: processScanner,
		listeners:      listeners,
	}

	p.initialize()

	return p, nil
}

func (r *ProcessRegistry) initialize() {
	// Set initial process generation to any non-zero value in order to distinguish
	// zero-initialized atomics inside processInfo from real generations.
	r.procsGeneration.Store(1)
}

func (r *ProcessRegistry) RunWorker(ctx context.Context) error {
	g, newCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.uploader.RunWorker(newCtx)
	})

	g.Go(func() error {
		return r.runHandler(newCtx)
	})

	return g.Wait()
}

func (r *ProcessRegistry) deleteProcess(ctx context.Context, pid linux.ProcessID) {
	r.procsmu.Lock()
	pi := r.procs[pid]
	delete(r.procs, pid)
	r.procsmu.Unlock()

	r.dsoStorage.RemoveProcess(ctx, pid)
	r.removeProcessMappings(ctx, pi)

	err := r.bpf.RemoveProcess(pid)
	if err != nil {
		r.log.Debug(
			ctx,
			"Failed to remove process info from the eBPF mapping",
			log.UInt32("pid", uint32(pid)),
			log.Error(err),
		)
	}

	for _, listener := range r.listeners {
		listener.OnProcessDeath(pid)
	}
}

func (r *ProcessRegistry) collectDeadPids(ctx context.Context, newGen uint64) []linux.ProcessID {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()

	deadPids := []linux.ProcessID{}
	for pid, proc := range r.procs {
		gen := proc.generation.Load()
		if gen == newGen {
			continue
		}

		_ = proc.setState(processStateDeleted)
		deadPids = append(deadPids, pid)

		r.log.Debug(
			ctx,
			"Found dead process",
			log.UInt32("pid", uint32(pid)),
			log.UInt64("newgen", newGen),
			log.UInt64("procgen", gen),
		)
	}

	return deadPids
}

type procScanStats struct {
	BornProcesses  int
	DiedProcesses  int
	AliveProcesses int
}

type processDiscoverer struct {
	r     *ProcessRegistry
	stats *procScanStats
}

func (p *processDiscoverer) discover(ctx context.Context, pid linux.ProcessID) {
	p.r.log.Debug(ctx, "Scanned process", log.UInt32("pid", uint32(pid)))
	discovered := p.r.DiscoverProcess(ctx, linux.ProcessID(pid))
	if discovered {
		p.stats.BornProcesses++
	}
	p.stats.AliveProcesses++
}

func (r *ProcessRegistry) scanProcesses(ctx context.Context) (stats procScanStats, err error) {
	newGen := r.procsGeneration.Add(1)
	processDiscoverer := &processDiscoverer{
		r:     r,
		stats: &stats,
	}
	err = r.processScanner.Scan(ctx, processDiscoverer.discover)

	// TODO: what if process dies between two scans and another process
	//   with same pid occurs. Maybe use process creation timestamp to detect this case?

	// TODO: add unit tests for strange process creations and deletions
	//     for purposes of checking thread-safety and deadlocks

	deadPids := r.collectDeadPids(ctx, newGen)
	stats.DiedProcesses += len(deadPids)
	for _, pid := range deadPids {
		r.deleteProcess(ctx, pid)
	}

	return
}

func (r *ProcessRegistry) RunProcessPoller(ctx context.Context) error {
	_, err := r.scanProcesses(ctx)
	if err != nil {
		return err
	}

	tick := time.NewTicker(ProcScanPeriod)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}

		r.log.Info(ctx, "Run process scanner")
		stats, err := r.scanProcesses(ctx)
		if err != nil {
			r.log.Error(ctx, "Process scanner failed", log.Error(err))
		} else {
			r.log.Info(ctx, "Finished process scanner", log.Any("stats", stats))
		}
	}
}

func (r *ProcessRegistry) DiscoverProcess(ctx context.Context, pid linux.ProcessID) (discovered bool) {
	curgen := r.procsGeneration.Load()

	// Happy-path. Just acquire rlock & lookup the pid in the map.
	r.procsmu.RLock()
	if info, ok := r.procs[pid]; ok {
		r.procsmu.RUnlock()
		info.generation.Store(curgen)
		return false
	}
	r.procsmu.RUnlock()

	// Insert new processInfo into the process map.
	var info *processInfo
	r.procsmu.Lock()
	if _, ok := r.procs[pid]; ok {
		r.procsmu.Unlock()
		return false
	}
	info = &processInfo{
		id:             pid,
		state:          processStateDiscovered,
		registeredmaps: make(map[procfs.Address]processMap),
	}
	info.generation.Store(curgen)
	r.procs[pid] = info
	r.procsmu.Unlock()

	r.tryScheduleProcessUpdate(ctx, info)

	return true
}

func (r *ProcessRegistry) tryScheduleProcessUpdate(ctx context.Context, info *processInfo) {
	desired := r.procsGeneration.Load()
	current := info.mapsgeneration.Load()
	if current >= desired {
		return
	}

	if !info.mapsgeneration.CompareAndSwap(current, desired) {
		return
	}

	// DiscoverProcess should be fast.
	// Add the process to the queue for the async discovery.
	select {
	case r.procchan <- info:
	default:
		r.log.Warn(
			ctx,
			"Failed to enqueue process discovery",
			log.UInt32("pid", uint32(info.id)),
			log.Int("current", int(current)),
			log.Int("desired", int(desired)),
		)
	}
}

func (r *ProcessRegistry) GetEnvs(pid linux.ProcessID) map[string]string {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()
	processInfo, ok := r.procs[pid]
	if ok {
		return processInfo.Env()
	}
	return nil
}

func (r *ProcessRegistry) MaybeRescanProcess(ctx context.Context, pid linux.ProcessID) {
	var p *processInfo

	r.procsmu.RLock()
	p = r.procs[pid]
	r.procsmu.RUnlock()

	if p == nil {
		return
	}

	r.tryScheduleProcessUpdate(ctx, p)
}

func (r *ProcessRegistry) runHandler(ctx context.Context) error {
	var proc *processInfo
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case proc = <-r.procchan:
		}

		err := r.handleProcess(ctx, proc)
		if err != nil {
			r.log.Debug(
				ctx,
				"Failed to handle new process",
				log.UInt32("pid", uint32(proc.id)),
				log.Error(err),
			)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func (r *ProcessRegistry) handleProcess(ctx context.Context, proc *processInfo) error {
	a := processAnalyzer{
		reg:         r,
		proc:        proc,
		log:         r.log.With(log.UInt32("pid", uint32(proc.id))),
		uploader:    r.uploader,
		exemappings: make([]*dso.Mapping, 0, 4),
	}
	return a.run(ctx)
}

type processAnalyzer struct {
	reg         *ProcessRegistry
	proc        *processInfo
	uploader    *upload.Scheduler
	log         xlog.Logger
	exemappings []*dso.Mapping
}

func (a *processAnalyzer) run(ctx context.Context) error {
	err := a.proc.setState(processStatePopulating)
	if err != nil {
		return err
	}

	defer func() {
		_ = a.proc.setState(processStatePopulated)
	}()

	if err := a.loadEnvs(ctx); err != nil {
		// Do not fail entire process discovery, just log an error.
		// A process can have malformed environment file.
		// For example, nginx overwrites original environ:
		// https://github.com/nginx/nginx/blob/master/src/os/unix/ngx_setproctitle.c#L35
		a.log.Debug(ctx, "Failed to load process environment", log.Error(err))
	}

	if !a.proc.listenersNotified.Swap(true) {
		for _, l := range a.reg.listeners {
			l.OnProcessDiscovery(a.proc)
		}
	}

	if err := a.loadMaps(ctx); err != nil {
		return err
	}

	if err := a.storeBPFMaps(ctx); err != nil {
		return err
	}

	return nil
}

func (a *processAnalyzer) loadMaps(ctx context.Context) error {
	return procfs.Process(a.proc.id).ListMappings(func(mapping *procfs.Mapping) error {
		// Skip non-executable mappings.
		if mapping.Permissions&procfs.MappingPermissionExecutable == 0 {
			return nil
		}

		if err := a.processMapping(ctx, mapping); err != nil {
			a.log.Debug(
				ctx,
				"Failed to process mapping",
				log.String("path", mapping.Path),
				log.Error(err),
			)
		}

		return nil
	})
}

func (a *processAnalyzer) processMapping(ctx context.Context, m *procfs.Mapping) error {
	mapping := dso.Mapping{Mapping: *m}
	if mapping.Path == "" {
		// Probably JITed mapping.
		mapping.Path = "[JIT]"
		_, err := a.reg.dsoStorage.AddMapping(ctx, a.proc.id, mapping, nil)
		return err
	}

	if vdso.IsUnsymbolizableVDSOMapping(&mapping.Mapping) {
		_, err := a.reg.dsoStorage.AddMapping(ctx, a.proc.id, mapping, nil)
		return err
	}

	binary := binary.NewProcessMappingBinary(a.proc.id, a.reg.mounts, m)
	a.log.Debug(
		ctx,
		"Found executable mapping",
		log.String("path", mapping.Path),
		log.String("begin", binary.ProcMapFilesPath),
	)

	err := binary.Open()
	if err != nil {
		return fmt.Errorf("failed to analyze executable mapping: %w", err)
	}

	defer func() {
		_ = binary.Close()
	}()

	if mapping.Inode.ID != binary.InodeID {
		return fmt.Errorf(
			"failed to register mapping: inode mismatch, expected %d, got %d",
			mapping.Inode.ID,
			binary.InodeID,
		)
	}

	// This code is racy.
	// Linux does not give us any way to get correct mappings
	// (i.e. ino_generation of the inode) of the process.
	//
	// There is perf_event_open + PERF_RECORD_MMAP2, but there is no guarantee
	// that we won't lose any records (and we WILL lose them).
	//
	// Let's try to get inode & inode generation as soon as possible and hope for the best.
	if mapping.Inode.Gen == 0 {
		mapping.Inode.Gen = binary.InodeGen
	}

	buildinfo, err := a.reg.buildids.Load(BuildIDKey{
		Device: mapping.Device,
		Inode:  mapping.Inode,
	}, binary.GetFile())

	if err != nil {
		return fmt.Errorf("failed to resolve mapping %s buildid: %w", binary.ProcMapFilesPath, err)
	}

	a.reg.metrics.mappingsDiscovered.Inc()

	buildid := buildinfo.BuildID
	if buildid == "" {
		a.reg.metrics.mappingsWithoutBuildID.Inc()
	}

	mapping.BuildInfo = buildinfo
	mapping.BaseAddress = mapping.Begin - buildinfo.LoadBias
	l := a.log.With(log.String("path", mapping.Path), log.String("buildid", buildid))
	l.Debug(ctx, "Found mapping build id", log.Any("buildinfo", mapping.BuildInfo), log.UInt64("baseaddr", mapping.BaseAddress))

	handle, err := binary.Seal()
	if err != nil {
		l.Debug(
			ctx,
			"Failed to seal binary",
			log.String("build_id", buildid),
			log.String("path", mapping.Path),
			log.Error(err),
		)
		a.reg.metrics.mappingsFailedNameToHandleAt.Inc()
		return err
	}

	dso, err := a.reg.dsoStorage.AddMapping(
		ctx,
		a.proc.id,
		mapping,
		binary,
	)
	if err != nil {
		l.Error(ctx, "Failed to register mapping", log.Error(err))
	}

	mapping.DSO = dso
	a.registerMapping(&mapping)

	a.reg.dsoStorage.Compactify(ctx, a.proc.id)

	err = a.uploader.ScheduleBinary(buildid, handle)
	if err != nil {
		a.reg.metrics.mappingsFailedScheduleUpload.Inc()
		l.Debug(ctx, "Failed to schedule binary for upload", log.String("build_id", buildid), log.Error(err))
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (a *processAnalyzer) registerMapping(m *dso.Mapping) {
	a.exemappings = append(a.exemappings, m)
}

func (a *processAnalyzer) storeBPFMaps(ctx context.Context) error {
	sort.Slice(a.exemappings, func(i, j int) bool {
		return a.exemappings[i].Begin < a.exemappings[j].Begin
	})

	a.syncMaps(ctx)

	pi := unwinder.ProcessInfo{
		UnwindType: unwinder.UnwindTypeDwarf,
	}
	if len(a.exemappings) > 0 && a.exemappings[0].DSO != nil {
		pi.MainBinaryId = unwinder.BinaryId(a.exemappings[0].DSO.ID)
	} else {
		pi.MainBinaryId = unwinder.BinaryId(math.MaxUint64)
	}

	a.log.Debug(ctx, "Put process info", log.Any("info", pi))
	err := a.reg.bpf.AddProcess(a.proc.id, &pi)
	if err != nil {
		return err
	}

	return nil
}

func (a *processAnalyzer) syncMaps(ctx context.Context) {
	visited := make(map[uint64]struct{}, len(a.exemappings))

	a.proc.mapslock.Lock()
	defer a.proc.mapslock.Unlock()

	toRemove := make([]processMap, 0)
	toAdd := make([]*dso.Mapping, 0)

	for _, m := range a.exemappings {
		if m.DSO == nil {
			continue
		}
		visited[m.Begin] = struct{}{}

		mapping, ok := a.proc.registeredmaps[m.Begin]
		// Happy path. Mapping exist and points to the valid binary.
		if ok && mapping.DSO.ID == m.DSO.ID && mapping.End == m.End {
			continue
		}

		if ok {
			toRemove = append(toRemove, mapping)
		}
		toAdd = append(toAdd, m)
	}

	for begin, mapping := range a.proc.registeredmaps {
		if _, ok := visited[begin]; ok {
			continue
		}
		toRemove = append(toRemove, mapping)
	}

	for _, m := range toRemove {
		a.reg.removeBPFMap(ctx, a.proc, m)
	}

	for _, m := range toAdd {
		a.reg.addBPFMap(ctx, a.proc, m)
	}
}

func (r *ProcessRegistry) addBPFMap(ctx context.Context, pi *processInfo, m *dso.Mapping) {
	l := r.log.With(log.UInt32("pid", pi.id)).WithName("lpm")
	l.Debug(ctx, "Trying to add eBPF mapping", log.String("buildid", m.BuildInfo.BuildID))

	id := pi.nextmapid.Add(1)

	// Step 1. Populate LPM trie
	err := iterateMappingLPMSegments(m, func(address uint64, prefix uint32) error {
		return r.bpf.AddMappingLPMSegment(&unwinder.ExecutableMappingTrieKey{
			Prefixlen:     32 + prefix,
			Pid:           pi.id,
			AddressPrefix: HostToBigEndian64(address),
		}, &unwinder.ExecutableMappingInfo{
			Id: id,
		})
	})
	if err != nil {
		l.Warn(ctx, "Failed to add eBPF mapping lpm trie segment", log.Error(err))
		return
	}

	// Step 2. Add eBPF mapping to the per-process registry.
	err = r.bpf.AddMapping(&unwinder.ExecutableMappingKey{
		Pid:           pi.id,
		UnusedPadding: 0,
		Id:            id,
	}, &unwinder.ExecutableMapping{
		Begin:    m.Begin,
		End:      m.End,
		BinaryId: m.DSO.ID,
		Offset:   int64(m.BaseAddress),
	})
	if err != nil {
		l.Warn(ctx, "Failed to add eBPF mapping", log.Error(err))
		return
	}

	// Step 3. Now we can finally commit our map to the user-space registery.
	pi.registeredmaps[m.Begin] = processMap{m, id}
}

func HostToBigEndian64(value uint64) uint64 {
	var buf [8]byte
	eb.NativeEndian.PutUint64(buf[:], value)
	return eb.BigEndian.Uint64(buf[:])
}

func (r *ProcessRegistry) removeBPFMap(ctx context.Context, pi *processInfo, m processMap) {
	l := r.log.With(log.UInt32("pid", pi.id)).WithName("lpm")
	l.Debug(ctx, "Trying to remove eBPF mapping", log.String("buildid", m.BuildInfo.BuildID))

	// Step 1. Remove LPM trie
	err := iterateMappingLPMSegments(m.Mapping, func(address uint64, prefix uint32) error {
		return r.bpf.RemoveMappingLPMSegment(&unwinder.ExecutableMappingTrieKey{
			Prefixlen:     32 + prefix,
			Pid:           pi.id,
			AddressPrefix: HostToBigEndian64(address),
		})
	})
	if err != nil {
		l.Warn(ctx, "Failed to remove eBPF mapping lpm trie segment", log.Error(err))
		return
	}

	// Step 2. Remove eBPF mapping from the per-process registry.
	// If this fails, we will retry on the next iteration.
	err = r.bpf.RemoveMapping(&unwinder.ExecutableMappingKey{
		Pid: pi.id,
		Id:  m.id,
	})
	if err != nil {
		l.Warn(ctx, "Failed to remove eBPF mapping", log.Error(err))
		return
	}

	// Step 3. Now we can finally remove our map from user-space registery.
	delete(pi.registeredmaps, m.Begin)
}

func (r *ProcessRegistry) removeProcessMappings(ctx context.Context, pi *processInfo) {
	for _, m := range maps.Values(pi.registeredmaps) {
		r.removeBPFMap(ctx, pi, m)
	}
}

func iterateMappingLPMSegments(m *dso.Mapping, callback func(address uint64, prefix uint32) error) error {
	addr := m.Begin

	for addr < m.End {
		for bits := min(63, bits.TrailingZeros64(addr)); bits >= 0; bits-- {
			width := uint64(1) << bits
			if addr+width <= m.End {
				err := callback(addr, uint32(64-bits))
				if err != nil {
					return err
				}

				addr += width
				break
			}
		}
	}
	if addr != m.End {
		return fmt.Errorf("BUG: invalid LPM segment set, got %x final address for [%x, %x) mapping", addr, m.Begin, m.End)
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (a *processAnalyzer) loadEnvs(ctx context.Context) error {
	envs, err := procfs.Process(a.proc.id).ListEnvs()
	if err != nil {
		return err
	}

	a.log.Debug(ctx, "Put process envs", log.Int("env_count", len(envs)))
	a.proc.setEnvs(envs)
	return nil
}

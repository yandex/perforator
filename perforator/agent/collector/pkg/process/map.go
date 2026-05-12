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

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/exp/maps"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/upload"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/mountinfo"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/linux/vdso"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

type ProcessRegistry struct {
	*pidNamespaceIndex

	log xlog.Logger

	procs   map[linux.CurrentNamespacePID]*processInfo
	procsmu sync.RWMutex
	// incremented each time new scan starts
	procsGeneration atomic.Uint64
	procchan        chan *processInfo

	listeners []Listener

	buildids   *BuildIDCache
	dsoStorage *dso.Storage
	state      *programstate.State
	mounts     *mountinfo.Watcher

	uploader *upload.Scheduler

	metrics        processRegistryMetrics
	processScanner ProcessScanner
}

type processRegistryMetrics struct {
	mappingsDiscovered              metrics.Counter
	mappingsWithoutBuildID          metrics.Counter
	mappingsJitted                  metrics.Counter
	mappingsFailedScheduleUpload    metrics.Counter
	mappingsFailedNameToHandleAt    metrics.Counter
	mappingsFailedELFVaddrRetrieval metrics.Counter

	processesWithEmptyEnvironment metrics.Counter
	processEnvironmentWaitDelay   metrics.Counter
}

type mappingImpl struct {
	m *dso.Mapping
}

func (m mappingImpl) ID() uint64 {
	return m.m.DSO.ID
}

func (m mappingImpl) BaseAddress() uint64 {
	return m.m.BaseAddress
}

func (m mappingImpl) begin() uint64 {
	return m.m.Begin
}

func (m mappingImpl) end() uint64 {
	return m.m.End
}

func (m mappingImpl) Path() string {
	return m.m.Path
}

func (m mappingImpl) BinaryClass() dso.BinaryClass {
	return m.m.DSO.BinaryClass
}

func (m mappingImpl) buildInfo() *xelf.BuildInfo {
	return m.m.BuildInfo
}

type processMap struct {
	Mapping
	id uint32
}

type processInfo struct {
	currentNamespaceID   linux.CurrentNamespacePID
	pidNamespaceIndexKey *pidNamespaceIndexKey

	state             processState
	lock              sync.RWMutex
	envs              map[string]string
	listenersNotified atomic.Bool

	// Used for deletion purposes. All modifications happen under r.procsmu in shared or exclusive mode
	generation     atomic.Uint64
	mapsgeneration atomic.Uint64
	nextmapid      atomic.Uint32
	// map itself can be changed (while holding mapslock), but values must be immutable.
	registeredmaps map[procfs.Address]processMap
	mapslock       sync.Mutex
}

func (p *processInfo) setState(state processState) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.state == processStateDeleted && state != processStateDeleted {
		return fmt.Errorf("process %d has already been deleted", p.currentNamespaceID)
	}

	p.state = state
	return nil
}

var _ ProcessInfo = (*processInfo)(nil)

// ProcessID implements ProcessInfo
func (p *processInfo) ProcessID() linux.CurrentNamespacePID {
	return p.currentNamespaceID
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

// Mappings implements ProcessInfo
func (p *processInfo) Mappings() []Mapping {
	p.mapslock.Lock()
	defer p.mapslock.Unlock()
	mps := make([]Mapping, 0, len(p.registeredmaps))
	for _, mp := range p.registeredmaps {
		mps = append(mps, mp.Mapping)
	}
	return mps
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
	state *programstate.State,
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
		log:               l,
		procs:             make(map[linux.CurrentNamespacePID]*processInfo),
		dsoStorage:        dsoStorage,
		state:             state,
		procchan:          make(chan *processInfo, 8192),
		buildids:          NewBuildIDCache(),
		uploader:          uploader,
		mounts:            mounts,
		pidNamespaceIndex: newPidNamespaceIndex(),
		metrics: processRegistryMetrics{
			mappingsDiscovered:              m.WithTags(map[string]string{"kind": "discovered"}).Counter("mappings.count"),
			mappingsWithoutBuildID:          m.WithTags(map[string]string{"kind": "nobuildid"}).Counter("mappings.count"),
			mappingsJitted:                  m.WithTags(map[string]string{"kind": "jitted"}).Counter("mappings.count"),
			mappingsFailedScheduleUpload:    m.WithTags(map[string]string{"kind": "failed_schedule_upload"}).Counter("mappings.count"),
			mappingsFailedNameToHandleAt:    m.WithTags(map[string]string{"kind": "failed_name_to_handle_at"}).Counter("mappings.count"),
			mappingsFailedELFVaddrRetrieval: m.WithTags(map[string]string{"kind": "failed_elf_vaddr_retrieval"}).Counter("mappings.count"),
			processesWithEmptyEnvironment:   m.Counter("processes.with_empty_environment.count"),
			processEnvironmentWaitDelay:     m.Counter("environment.wait_delay.total.milliseconds"),
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

type WorkerConfig struct {
	// Wait with exponential backoff is reading process environment returns empty result.
	WaitOnEmptyEnv *bool `yaml:"wait_on_empty_environment"`
}

func (r *ProcessRegistry) RunWorker(ctx context.Context, conf WorkerConfig) error {
	g, newCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return r.uploader.RunWorker(newCtx)
	})

	g.Go(func() error {
		return r.runHandler(newCtx, &conf)
	})

	return g.Wait()
}

func (r *ProcessRegistry) deleteProcess(ctx context.Context, pid linux.CurrentNamespacePID) {
	r.procsmu.Lock()
	pi := r.procs[pid]
	delete(r.procs, pid)
	r.procsmu.Unlock()

	r.unregisterPidNamespaceCorrelation(pi)

	r.dsoStorage.RemoveProcess(ctx, pid)
	r.removeProcessMappings(ctx, pi)

	err := r.state.RemoveProcess(pid)
	if err != nil {
		r.log.Debug(
			ctx,
			"Failed to remove process info from the eBPF mapping",
			log.UInt32("current_namespace_pid", uint32(pid)),
			log.Error(err),
		)
	}

	for _, listener := range r.listeners {
		listener.OnProcessDeath(ctx, pid)
	}
}

func (r *ProcessRegistry) collectDeadPids(ctx context.Context, newGen uint64) []linux.CurrentNamespacePID {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()

	deadPids := []linux.CurrentNamespacePID{}
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

func (p *processDiscoverer) discover(ctx context.Context, pid linux.CurrentNamespacePID) {
	p.r.log.Debug(ctx, "Scanned process", log.UInt32("pid", uint32(pid)))
	discovered := p.r.DiscoverProcess(ctx, pid)
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

func (r *ProcessRegistry) RunProcessScanner(ctx context.Context) error {
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

		r.log.Debug(ctx, "Run process scanner")
		stats, err := r.scanProcesses(ctx)
		if err != nil {
			r.log.Error(ctx, "Process scanner failed", log.Error(err))
		} else {
			r.log.Debug(ctx, "Finished process scanner", log.Any("stats", stats))
		}
	}
}

func (r *ProcessRegistry) DiscoverProcess(ctx context.Context, pid linux.CurrentNamespacePID) (discovered bool) {
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
		currentNamespaceID: pid,
		state:              processStateDiscovered,
		registeredmaps:     make(map[procfs.Address]processMap),
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
			log.UInt32("pid", uint32(info.currentNamespaceID)),
			log.Int("current", int(current)),
			log.Int("desired", int(desired)),
		)
	}
}

func (r *ProcessRegistry) GetEnvs(pid linux.CurrentNamespacePID) map[string]string {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()
	processInfo, ok := r.procs[pid]
	if ok {
		return processInfo.Env()
	}
	return nil
}

func (r *ProcessRegistry) MaybeRescanProcess(ctx context.Context, pid linux.CurrentNamespacePID) {
	var p *processInfo

	r.procsmu.RLock()
	p = r.procs[pid]
	r.procsmu.RUnlock()

	if p == nil {
		return
	}

	r.tryScheduleProcessUpdate(ctx, p)
}

func (r *ProcessRegistry) runHandler(ctx context.Context, config *WorkerConfig) error {
	var proc *processInfo
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case proc = <-r.procchan:
		}

		err := r.handleProcess(ctx, proc, config)
		if err != nil {
			r.log.Debug(
				ctx,
				"Failed to handle new process",
				log.UInt32("pid", uint32(proc.currentNamespaceID)),
				log.Error(err),
			)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func (r *ProcessRegistry) handleProcess(ctx context.Context, proc *processInfo, config *WorkerConfig) error {
	a := processAnalyzer{
		reg:         r,
		config:      config,
		proc:        proc,
		log:         r.log.With(log.UInt32("pid", uint32(proc.currentNamespaceID))),
		uploader:    r.uploader,
		exemappings: make([]*dso.Mapping, 0, 4),
	}
	return a.run(ctx)
}

type processAnalyzer struct {
	reg         *ProcessRegistry
	config      *WorkerConfig
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

	a.reg.tryRegisterPidNamespaceCorrelation(ctx, a.proc)

	if err := a.loadMaps(ctx); err != nil {
		return err
	}

	if err := a.storeBPFMaps(ctx); err != nil {
		return err
	}

	// Note that a.registeredmaps must be populated before exposing process object to listeners,
	// so this has to be sequenced after storeBPFMaps.
	if !a.proc.listenersNotified.Swap(true) {
		for _, l := range a.reg.listeners {
			l.OnProcessDiscovery(ctx, a.proc)
		}
	} else {
		for _, l := range a.reg.listeners {
			l.OnProcessRescan(ctx, a.proc)
		}
	}

	return nil
}

func (a *processAnalyzer) loadMaps(ctx context.Context) error {
	return procfs.Open(a.proc.currentNamespaceID).ListMappings(func(mapping *procfs.Mapping) error {
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
		a.reg.dsoStorage.AddMapping(ctx, a.proc.currentNamespaceID, mapping, nil)
		return nil
	}

	if vdso.IsUnsymbolizableVDSOMapping(&mapping.Mapping) {
		a.reg.dsoStorage.AddMapping(ctx, a.proc.currentNamespaceID, mapping, nil)
		return nil
	}

	binary := binary.NewProcessMappingBinary(a.proc.currentNamespaceID, a.reg.mounts, m)
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
		Mtime:  binary.Mtime,
		Size:   binary.Size,
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

	l := a.log.With(log.String("path", mapping.Path), log.String("buildid", buildid))

	mappingELFVaddr, err := xelf.ELFOffsetToVaddr(buildinfo.ExecutableLoadablePhdrs, mapping.Offset)
	if err != nil {
		l.Warn(
			ctx,
			"Failed to obtain mapping ELF vaddr",
			log.Any("mapping", mapping),
			log.Error(err),
		)
		a.reg.metrics.mappingsFailedELFVaddrRetrieval.Inc()
		return err
	}
	mapping.BaseAddress = mapping.Begin - mappingELFVaddr

	l.Debug(
		ctx,
		"Found mapping build id",
		log.Any("buildinfo", mapping.BuildInfo),
		log.UInt64("baseaddr", mapping.BaseAddress),
	)

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

	dso := a.reg.dsoStorage.AddMapping(
		ctx,
		a.proc.currentNamespaceID,
		mapping,
		binary,
	)

	mapping.DSO = dso
	a.registerMapping(&mapping)

	a.reg.dsoStorage.Compactify(ctx, a.proc.currentNamespaceID)

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

func mappedBinaryFromMapping(mapping *dso.Mapping) unwinder.MappedBinary {
	return unwinder.MappedBinary{
		Id:          unwinder.BinaryId(mapping.DSO.ID),
		BaseAddress: mapping.BaseAddress,
	}
}

func (a *processAnalyzer) fillMappedBinaryInfo(pi *unwinder.ProcessInfo, mappings []*dso.Mapping) {
	for _, m := range mappings {
		if m.DSO == nil {
			continue
		}

		switch m.DSO.BinaryClass {
		case dso.PythonBinaryClass:
			pi.PythonBinary = mappedBinaryFromMapping(m)
		case dso.PhpBinaryClass:
			pi.PhpBinary = mappedBinaryFromMapping(m)
		case dso.LuaBinaryClass:
			println("SPAR: map::fillMappedBinaryInfo -> case dso.LuaBinaryClass:")

			pi.LuaBinary = mappedBinaryFromMapping(m)
		case dso.PthreadGlibcBinaryClass:
			pi.PthreadBinary = mappedBinaryFromMapping(m)
		case dso.JvmBinaryClass:
			pi.LibjvmBinary = mappedBinaryFromMapping(m)
		}
	}
}

func newProcessInfo() *unwinder.ProcessInfo {
	return &unwinder.ProcessInfo{
		UnwindType:   unwinder.UnwindTypeMixed,
		LibjvmBinary: unwinder.MappedBinary{BaseAddress: math.MaxUint64},
		MainBinaryId: unwinder.BinaryId(math.MaxUint64),
		PhpBinary:    unwinder.MappedBinary{BaseAddress: math.MaxUint64},
		PythonBinary: unwinder.MappedBinary{BaseAddress: math.MaxUint64},
		LuaBinary:    unwinder.MappedBinary{BaseAddress: math.MaxUint64},
		PthreadBinary: unwinder.MappedBinary{
			BaseAddress: math.MaxUint64,
		},
	}
}

func (a *processAnalyzer) storeBPFMaps(ctx context.Context) error {
	sort.Slice(a.exemappings, func(i, j int) bool {
		return a.exemappings[i].Begin < a.exemappings[j].Begin
	})

	a.syncMaps(ctx)

	pi := newProcessInfo()
	if len(a.exemappings) > 0 && a.exemappings[0].DSO != nil {
		pi.MainBinaryId = unwinder.BinaryId(a.exemappings[0].DSO.ID)
	}
	a.fillMappedBinaryInfo(pi, a.exemappings)
	if pi.LibjvmBinary.BaseAddress != math.MaxUint64 {
		pi.UnwindType = unwinder.UnwindTypeMixed
		a.log.Debug(ctx, "Enabling mixed unwinder", log.UInt32("pid", uint32(a.proc.currentNamespaceID)))
	}

	a.log.Debug(ctx, "Put process info", log.Any("info", pi))
	err := a.reg.state.AddProcess(a.proc.currentNamespaceID, pi)
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
		if ok && mapping.ID() == m.DSO.ID && mapping.end() == m.End {
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
	l := r.log.With(logfield.CurrentNamespacePID(pi.currentNamespaceID)).WithName("lpm")
	l.Debug(ctx, "Trying to add eBPF mapping", log.String("buildid", m.BuildInfo.BuildID))

	id := pi.nextmapid.Add(1)

	// Step 1. Populate LPM trie
	err := iterateMappingLPMSegments(mappingImpl{m}, func(address uint64, prefix uint32) error {
		return r.state.AddMappingLPMSegment(&unwinder.ExecutableMappingTrieKey{
			Prefixlen:     32 + prefix,
			Pid:           uint32(pi.currentNamespaceID),
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
	err = r.state.AddMapping(&unwinder.ExecutableMappingKey{
		Pid:           uint32(pi.currentNamespaceID),
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
	pi.registeredmaps[m.Begin] = processMap{mappingImpl{m}, id}
}

func HostToBigEndian64(value uint64) uint64 {
	var buf [8]byte
	eb.NativeEndian.PutUint64(buf[:], value)
	return eb.BigEndian.Uint64(buf[:])
}

func (r *ProcessRegistry) removeBPFMap(ctx context.Context, pi *processInfo, m processMap) {
	l := r.log.With(logfield.CurrentNamespacePID(pi.currentNamespaceID)).WithName("lpm")
	l.Debug(ctx, "Trying to remove eBPF mapping", log.String("buildid", m.buildInfo().BuildID))

	// Step 1. Remove LPM trie
	err := iterateMappingLPMSegments(m.Mapping, func(address uint64, prefix uint32) error {
		return r.state.RemoveMappingLPMSegment(&unwinder.ExecutableMappingTrieKey{
			Prefixlen:     32 + prefix,
			Pid:           uint32(pi.currentNamespaceID),
			AddressPrefix: HostToBigEndian64(address),
		})
	})
	if err != nil {
		l.Warn(ctx, "Failed to remove eBPF mapping lpm trie segment", log.Error(err))
		return
	}

	// Step 2. Remove eBPF mapping from the per-process registry.
	// If this fails, we will retry on the next iteration.
	err = r.state.RemoveMapping(&unwinder.ExecutableMappingKey{
		Pid: uint32(pi.currentNamespaceID),
		Id:  m.id,
	})
	if err != nil {
		l.Warn(ctx, "Failed to remove eBPF mapping", log.Error(err))
		return
	}

	// Step 3. Now we can finally remove our map from user-space registery.
	delete(pi.registeredmaps, m.begin())
}

func (r *ProcessRegistry) removeProcessMappings(ctx context.Context, pi *processInfo) {
	for _, m := range maps.Values(pi.registeredmaps) {
		r.removeBPFMap(ctx, pi, m)
	}
}

func iterateMappingLPMSegments(m Mapping, callback func(address uint64, prefix uint32) error) error {
	addr := m.begin()

	for addr < m.end() {
		for bits := min(63, bits.TrailingZeros64(addr)); bits >= 0; bits-- {
			width := uint64(1) << bits
			if addr+width <= m.end() {
				err := callback(addr, uint32(64-bits))
				if err != nil {
					return err
				}

				addr += width
				break
			}
		}
	}
	if addr != m.end() {
		return fmt.Errorf("BUG: invalid LPM segment set, got %x final address for [%x, %x) mapping", addr, m.begin(), m.end())
	}

	return nil
}

////////////////////////////////////////////////////////////////////////////////

// tryLoadEnvs returns environment of process `proc` or determines that it is not available yet.
// `firstIter` is a flag that should be set if tryLoadEnvs was already called successfully for this process.
// Second return value is true when environment is available.
func (a *processAnalyzer) tryLoadEnvs(ctx context.Context, proc *procfs.Process, firstIter bool) (map[string]string, bool, error) {
	envs, err := proc.ListEnvs()
	if err != nil {
		return nil, false, err
	}
	if len(envs) > 0 {
		// If we read non-empty data, environment is available.
		return envs, true, nil
	}

	stat, err := proc.Stat()
	if err != nil {
		return nil, false, fmt.Errorf("failed to read process stat: %w", err)
	}

	if stat.State == 'Z' {
		// Process is zombie. We aren't interested in zombies and assume environment is empty.
		return nil, true, nil
	}

	if stat.EnvEnd != 0 {
		// This is a regular process which actually has empty environment.
		return nil, true, nil
	}
	if firstIter {
		isKthread, err := proc.IsKthread()
		if err != nil {
			return nil, false, fmt.Errorf("failed to check whether process is a kthread: %w", err)
		}
		if isKthread {
			// This is a special process directly created by kernelspace. Kthreads have no environment.
			return nil, true, nil
		}
	}
	return nil, false, nil
}

func (a *processAnalyzer) loadEnvs(ctx context.Context) error {
	proc := procfs.Open(a.proc.currentNamespaceID)
	backoff := backoff.NewExponentialBackOff(
		backoff.WithInitialInterval(1*time.Millisecond),
		backoff.WithMultiplier(2),
		backoff.WithMaxElapsedTime(1*time.Second),
	)
	backoff.Reset()
	waitOnEmptyEnv := a.config.WaitOnEmptyEnv != nil && *a.config.WaitOnEmptyEnv

	if waitOnEmptyEnv {
		defer func() {
			a.reg.metrics.processEnvironmentWaitDelay.Add(backoff.GetElapsedTime().Milliseconds())
		}()
	}

	// TODO(PERFORATOR-1102): loop here is hacky attempt to work around some
	// race conditions when we fail to observe correct process environment shortly after process creation.
	for i := 0; ; i++ {
		envs, ok, err := a.tryLoadEnvs(ctx, proc, i == 0)
		if err != nil {
			return err
		}

		if ok {
			a.log.Debug(
				ctx,
				"Put process envs",
				log.Int("env_count", len(envs)),
				log.Int("attempts", i),
			)
			a.proc.setEnvs(envs)
			break
		}

		// Environment is not initialized yet. This is a race
		// with a newly created process.
		sleepFor := backoff.NextBackOff()
		if sleepFor == backoff.Stop || !waitOnEmptyEnv {
			// Level is not DEBUG because it is the only sign of a possible race
			// and processes with actually empty environment are likely to be rare.
			a.log.Warn(ctx, "Timed out waiting for process environment to initialize")
			a.reg.metrics.processesWithEmptyEnvironment.Inc()
			break
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("canceled while obtaining process environment: %w", context.Cause(ctx))
		case <-time.After(sleepFor):
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

// tryRegisterPidNamespaceCorrelation best-effort indexes pid namespace mapping to current namespace pid for a given process.
func (r *ProcessRegistry) tryRegisterPidNamespaceCorrelation(ctx context.Context, pi *processInfo) {
	pidnsInode, err := procfs.Open(pi.currentNamespaceID).GetNamespaces().GetPidInode()
	if err != nil {
		r.log.Debug(ctx, "Failed to get pid namespace inode", log.UInt32("pid", uint32(pi.currentNamespaceID)), log.Error(err))
		return
	}

	namespacedPid, err := procfs.Open(pi.currentNamespaceID).GetNamespacedPID()
	if err != nil {
		r.log.Warn(ctx, "Failed to get namespaced pid", log.UInt32("pid", uint32(pi.currentNamespaceID)), log.Error(err))
		return
	}

	pi.pidNamespaceIndexKey = &pidNamespaceIndexKey{
		namespacedPID:     namespacedPid,
		pidNamespaceInode: pidnsInode,
	}

	r.pidNamespaceIndex.add(namespacedPid, pidnsInode, pi.currentNamespaceID)
}

func (r *ProcessRegistry) unregisterPidNamespaceCorrelation(pi *processInfo) {
	if pi == nil || pi.pidNamespaceIndexKey == nil {
		return
	}

	r.pidNamespaceIndex.remove(pi.pidNamespaceIndexKey.namespacedPID, pi.pidNamespaceIndexKey.pidNamespaceInode)
}

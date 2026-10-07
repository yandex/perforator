package process

import (
	"context"
	eb "encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/bits"
	"os"
	"path"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/cilium/ebpf"
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
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

////////////////////////////////////////////////////////////////////////////////

type processBPFState interface {
	AddProcess(linux.CurrentNamespacePID, *unwinder.ProcessInfo) error
	RemoveProcess(linux.CurrentNamespacePID) error
	AddMappingLPMSegment(*unwinder.ExecutableMappingTrieKey, *unwinder.ExecutableMappingInfo) error
	RemoveMappingLPMSegment(*unwinder.ExecutableMappingTrieKey) error
	AddMapping(*unwinder.ExecutableMappingKey, *unwinder.ExecutableMapping) error
	RemoveMapping(*unwinder.ExecutableMappingKey) error
}

type ProcessRegistry struct {
	*pidNamespaceIndex

	log xlog.Logger

	procs   map[linux.CurrentNamespacePID]*processInfo
	procsmu sync.RWMutex
	// incremented each time new scan starts
	procsGeneration atomic.Uint64
	procchan        chan *processRegistration

	listeners         []Listener
	metadataPublisher MetadataPublisher

	buildids   *BuildIDCache
	dsoStorage *dso.Storage
	state      processBPFState
	mounts     *mountinfo.Watcher

	uploader   *upload.Scheduler
	uploadHost string

	metrics        processRegistryMetrics
	processScanner ProcessScanner
	processFilter  ProcessFilter
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
	// Incomplete mappings remain owned by the lifetime until cleanup succeeds.
	incomplete bool
}

// Pending updates survive lifetime changes within a registration. Removing the
// registration invalidates its pending work. Only discovery and scanning may
// extend liveness; samples must not do so.
type processRegistration struct {
	pid               linux.CurrentNamespacePID
	lifecycleMu       sync.Mutex
	generation        atomic.Uint64
	observedStartTime atomic.Uint64
	updateQueued      atomic.Bool
	refreshNeeded     atomic.Bool
	// Mapping IDs remain monotonic across lifetime replacements.
	nextmapid atomic.Uint32
}

func (p *processRegistration) observeStartTime(startTime uint64) uint64 {
	for observed := p.observedStartTime.Load(); ; observed = p.observedStartTime.Load() {
		if startTime <= observed {
			return observed
		}
		if p.observedStartTime.CompareAndSwap(observed, startTime) {
			return startTime
		}
	}
}

type processInfo struct {
	currentNamespaceID   linux.CurrentNamespacePID
	pidNamespaceIndexKey *pidNamespaceIndexKey
	*processRegistration

	// Retired lifetimes reject analysis and publication, even while cleanup is incomplete.
	retired           atomic.Bool
	envsMu            sync.RWMutex
	envs              map[string]string
	listenersNotified bool // guarded by lifecycleMu
	startTime         atomic.Uint64
	mapsgeneration    atomic.Uint64
	// map itself can be changed (while holding mapslock), but values must be immutable.
	registeredmaps map[procfs.Address]processMap
	mapslock       sync.Mutex
}

var _ ProcessInfo = (*processInfo)(nil)

func (p *processInfo) Key() linux.ProcessKey {
	return linux.ProcessKey{Pid: p.currentNamespaceID, ProcessStartTime: p.startTime.Load()}
}

func (p *processInfo) setEnvs(envs map[string]string) {
	p.envsMu.Lock()
	defer p.envsMu.Unlock()
	p.envs = envs
}

// Env implements ProcessInfo
func (p *processInfo) Env() map[string]string {
	p.envsMu.RLock()
	defer p.envsMu.RUnlock()
	return p.envs
}

// Mappings implements ProcessInfo
func (p *processInfo) Mappings() []Mapping {
	p.mapslock.Lock()
	defer p.mapslock.Unlock()
	mps := make([]Mapping, 0, len(p.registeredmaps))
	for _, mp := range p.registeredmaps {
		if !mp.incomplete {
			mps = append(mps, mp.Mapping)
		}
	}
	return mps
}

const ProcScanPeriod = 10 * time.Second

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
	processFilter ProcessFilter,
	listeners []Listener,
	metadataPublisher MetadataPublisher,
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

	uploadHost, err := os.Hostname()
	if err != nil {
		l.Logger().Warn("Failed to resolve binary upload hostname", log.Error(err))
		uploadHost = ""
	}

	p := &ProcessRegistry{
		log:               l,
		procs:             make(map[linux.CurrentNamespacePID]*processInfo),
		dsoStorage:        dsoStorage,
		state:             state,
		procchan:          make(chan *processRegistration, 8192),
		buildids:          NewBuildIDCache(),
		uploader:          uploader,
		uploadHost:        uploadHost,
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
		processScanner:    processScanner,
		processFilter:     processFilter,
		listeners:         listeners,
		metadataPublisher: metadataPublisher,
	}

	p.procsGeneration.Store(1)

	return p, nil
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
		return r.runHandler(newCtx, func(ctx context.Context, proc *processInfo) error {
			return r.handleProcess(ctx, proc, &conf)
		})
	})

	return g.Wait()
}

// retireProcessLocked requires pi.lifecycleMu. A failed retirement must be retried
// before the registration can be removed or a new lifetime can be installed.
func (r *ProcessRegistry) retireProcessLocked(ctx context.Context, pi *processInfo) bool {
	pi.retired.Store(true)
	if err := ignoreMissingBPFEntry(r.state.RemoveProcess(pi.currentNamespaceID)); err != nil {
		r.log.Debug(ctx, "Failed to remove process from eBPF state",
			log.UInt32("pid", uint32(pi.currentNamespaceID)), log.Error(err))
		return false
	}
	if !r.removeProcessMappings(ctx, pi) {
		return false
	}
	r.unregisterPidNamespaceCorrelation(pi)
	r.dsoStorage.RemoveProcess(ctx, pi.currentNamespaceID)
	r.notifyProcessDeathLocked(ctx, pi)
	return true
}

func (r *ProcessRegistry) isCurrent(info *processInfo) bool {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()
	return r.procs[info.currentNamespaceID] == info
}

func (r *ProcessRegistry) currentProcess(updates *processRegistration) *processInfo {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()
	info := r.procs[updates.pid]
	if info == nil || info.processRegistration != updates {
		return nil
	}
	return info
}

func (r *ProcessRegistry) deleteProcess(ctx context.Context, info *processInfo, scanGeneration uint64) bool {
	info.lifecycleMu.Lock()
	defer info.lifecycleMu.Unlock()
	if !r.isCurrent(info) || info.generation.Load() >= scanGeneration {
		return false
	}
	if !r.retireProcessLocked(ctx, info) {
		return false
	}
	r.procsmu.Lock()
	delete(r.procs, info.currentNamespaceID)
	r.procsmu.Unlock()
	return true
}

func (r *ProcessRegistry) notifyProcessDeathLocked(
	ctx context.Context,
	info *processInfo,
) {
	key := info.Key()
	if r.metadataPublisher != nil {
		r.metadataPublisher.OnProcessDeath(ctx, key)
	}
	for _, listener := range r.listeners {
		listener.OnProcessDeath(ctx, key)
	}
}

// Requires info.lifecycleMu and successful BPF activation of the current lifetime.
func (r *ProcessRegistry) notifyProcessUpdateLocked(ctx context.Context, info *processInfo) {
	if !info.listenersNotified {
		info.listenersNotified = true
		for _, listener := range r.listeners {
			listener.OnProcessDiscovery(ctx, info)
		}
	} else {
		for _, listener := range r.listeners {
			listener.OnProcessRescan(ctx, info)
		}
	}
}

func (r *ProcessRegistry) collectDeadProcesses(newGen uint64) []*processInfo {
	r.procsmu.RLock()
	defer r.procsmu.RUnlock()
	var dead []*processInfo
	for _, proc := range r.procs {
		if proc.generation.Load() < newGen {
			dead = append(dead, proc)
		}
	}
	return dead
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
	p.r.procsmu.RLock()
	info := p.r.procs[pid]
	p.r.procsmu.RUnlock()
	// Filtered PIDs still refresh liveness for their existing registration.
	if p.r.processFilter != nil && !p.r.processFilter(pid) {
		if info != nil {
			info.lifecycleMu.Lock()
			if p.r.currentProcess(info.processRegistration) != nil {
				info.generation.Store(p.r.procsGeneration.Load())
				p.stats.AliveProcesses++
				if info.refreshNeeded.Load() {
					p.r.tryScheduleProcessUpdate(ctx, info, true)
				}
			}
			info.lifecycleMu.Unlock()
		}
		return
	}
	p.stats.AliveProcesses++
	if p.r.DiscoverProcess(ctx, linux.ProcessKey{Pid: pid}) {
		p.stats.BornProcesses++
	}
}

func (r *ProcessRegistry) scanProcesses(ctx context.Context) (stats procScanStats, err error) {
	newGen := r.procsGeneration.Add(1)
	discoverer := &processDiscoverer{r: r, stats: &stats}
	if err = r.processScanner.Scan(ctx, discoverer.discover); err != nil {
		// A partial scan is not evidence of death.
		return
	}
	for _, info := range r.collectDeadProcesses(newGen) {
		if r.deleteProcess(ctx, info, newGen) {
			stats.DiedProcesses++
		}
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

// DiscoverProcess refreshes liveness and records the kernel identity when known.
// A zero start time does not replace a known identity. Samples use ObserveProcessSample.
func (r *ProcessRegistry) DiscoverProcess(ctx context.Context, key linux.ProcessKey) bool {
	return r.discoverProcess(ctx, key, nil)
}

// ObserveProcessSample records identity observations for an existing registration.
// Samples neither extend liveness nor create registrations.
func (r *ProcessRegistry) ObserveProcessSample(ctx context.Context, key linux.ProcessKey) {
	r.procsmu.RLock()
	info := r.procs[key.Pid]
	r.procsmu.RUnlock()
	if info == nil || (key.ProcessStartTime != 0 && key.ProcessStartTime < max(info.startTime.Load(), info.observedStartTime.Load())) {
		return
	}
	updates := info.processRegistration
	updates.observeStartTime(key.ProcessStartTime)
	r.tryScheduleProcessUpdate(ctx, info, key.ProcessStartTime != 0 && key.ProcessStartTime != info.startTime.Load())
}

func (r *ProcessRegistry) applySampleObservation(ctx context.Context, updates *processRegistration) *processInfo {
	if startTime := updates.observedStartTime.Load(); startTime != 0 {
		r.discoverProcess(ctx, linux.ProcessKey{Pid: updates.pid, ProcessStartTime: startTime}, updates)
	}
	return r.currentProcess(updates)
}

func (r *ProcessRegistry) discoverProcess(ctx context.Context, key linux.ProcessKey, expected *processRegistration) bool {
	pid := key.Pid
	startTime := key.ProcessStartTime
	for {
		r.procsmu.Lock()
		info := r.procs[pid]
		if expected != nil && (info == nil || info.processRegistration != expected) {
			r.procsmu.Unlock()
			return false
		}
		if info == nil {
			info = &processInfo{
				currentNamespaceID:  pid,
				processRegistration: &processRegistration{pid: pid},
				registeredmaps:      make(map[procfs.Address]processMap),
			}
			info.startTime.Store(startTime)
			info.observeStartTime(startTime)
			info.generation.Store(r.procsGeneration.Load())
			r.procs[pid] = info
			r.procsmu.Unlock()
			r.tryScheduleProcessUpdate(ctx, info, true)
			return true
		}
		r.procsmu.Unlock()

		info.lifecycleMu.Lock()
		if !r.isCurrent(info) {
			info.lifecycleMu.Unlock()
			continue
		}
		known := info.startTime.Load()
		if startTime != 0 && startTime < known {
			info.lifecycleMu.Unlock()
			return false
		}
		// The newest known identity survives failed retirement and scans without a start time.
		startTime = info.observeStartTime(max(startTime, known))
		if expected == nil {
			info.generation.Store(r.procsGeneration.Load())
		}
		if info.retired.Load() || (startTime != 0 && known != 0 && startTime != known) {
			if !r.retireProcessLocked(ctx, info) {
				info.lifecycleMu.Unlock()
				return false
			}
			next := &processInfo{
				currentNamespaceID:  pid,
				processRegistration: info.processRegistration,
				registeredmaps:      make(map[procfs.Address]processMap),
			}
			next.startTime.Store(startTime)
			r.procsmu.Lock()
			r.procs[pid] = next
			r.procsmu.Unlock()
			info.lifecycleMu.Unlock()
			r.tryScheduleProcessUpdate(ctx, next, true)
			return true
		}
		// Binding a bootstrap identity retains the existing process state.
		bound := startTime != 0 && known == 0
		if bound {
			info.startTime.Store(startTime)
		}
		info.lifecycleMu.Unlock()
		if bound || info.refreshNeeded.Load() {
			r.tryScheduleProcessUpdate(ctx, info, true)
		}
		return false
	}
}

func (r *ProcessRegistry) tryScheduleProcessUpdate(ctx context.Context, info *processInfo, force bool) {
	updates := info.processRegistration
	info = r.currentProcess(updates)
	if info == nil {
		return
	}
	if force {
		updates.refreshNeeded.Store(true)
	}
	desired := r.procsGeneration.Load()
	current := info.mapsgeneration.Load()
	if !force && current >= desired {
		return
	}

	if !updates.updateQueued.CompareAndSwap(false, true) {
		return
	}

	// Discovery does not wait for queue capacity.
	select {
	case r.procchan <- updates:
	default:
		updates.updateQueued.Store(false)
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

	r.tryScheduleProcessUpdate(ctx, p, false)
}

func (r *ProcessRegistry) runHandler(ctx context.Context, handleProcess func(context.Context, *processInfo) error) error {
	var updates *processRegistration
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case updates = <-r.procchan:
		}

		failed := false
		var proc *processInfo
		var observedStartTime uint64
		for {
			observedStartTime = updates.observedStartTime.Load()
			updates.refreshNeeded.Store(false)
			proc = r.applySampleObservation(ctx, updates)
			if proc == nil {
				break
			}
			generation := r.procsGeneration.Load()
			err := handleProcess(ctx, proc)
			if err != nil {
				failed = true
				r.log.Debug(
					ctx,
					"Failed to handle new process",
					log.UInt32("pid", uint32(proc.currentNamespaceID)),
					log.Error(err),
				)
				break
			}
			proc.mapsgeneration.Store(generation)
			if !updates.refreshNeeded.Load() && generation >= r.procsGeneration.Load() {
				break
			}
		}

		if failed {
			// Failed analysis remains pending until a retry can be scheduled.
			updates.refreshNeeded.Store(true)
		}
		updates.updateQueued.Store(false)
		current := r.currentProcess(updates)
		if current == nil {
			continue
		}
		// Errors defer the same lifetime to discovery or scan, but not a newer identity.
		if failed && current == proc && updates.observedStartTime.Load() == observedStartTime {
			continue
		}
		force := updates.refreshNeeded.Load()
		if force || current.mapsgeneration.Load() < r.procsGeneration.Load() {
			r.tryScheduleProcessUpdate(ctx, current, force)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////

func (r *ProcessRegistry) handleProcess(ctx context.Context, proc *processInfo, config *WorkerConfig) error {
	a := processAnalyzer{
		reg:              r,
		config:           config,
		proc:             proc,
		log:              r.log.With(log.UInt32("pid", uint32(proc.currentNamespaceID))),
		uploader:         r.uploader,
		exemappings:      make([]*dso.Mapping, 0, 4),
		preparedMappings: make([]*dso.Mapping, 0, 4),
		startTime:        proc.startTime.Load(),
	}
	return a.run(ctx)
}

type processAnalyzer struct {
	reg              *ProcessRegistry
	config           *WorkerConfig
	proc             *processInfo
	uploader         *upload.Scheduler
	log              xlog.Logger
	exemappings      []*dso.Mapping
	startTime        uint64
	preparedMappings []*dso.Mapping
	envs             map[string]string
}

func (a *processAnalyzer) ensureCurrentIdentity() error {
	if !a.reg.isCurrent(a.proc) {
		return fmt.Errorf("process %d analysis belongs to a retired lifetime", a.proc.currentNamespaceID)
	}
	if current := a.proc.startTime.Load(); current != a.startTime {
		return fmt.Errorf(
			"process identity changed while analyzing pid %d: %d -> %d",
			a.proc.currentNamespaceID,
			a.startTime,
			current,
		)
	}
	return nil
}

func (a *processAnalyzer) run(ctx context.Context) error {
	// Prepared binary references are local to this analysis until commit.
	defer func() { a.reg.dsoStorage.ReleaseMappings(ctx, a.preparedMappings) }()
	if a.proc.retired.Load() {
		return fmt.Errorf("process %d has already been deleted", a.proc.currentNamespaceID)
	}

	if err := a.loadEnvs(ctx); err != nil {
		// Processes may overwrite environ; environment errors are non-fatal.
		a.log.Debug(ctx, "Failed to load process environment", log.Error(err))
	}

	if err := a.loadMaps(ctx); err != nil {
		return err
	}

	return a.storeBPFMaps(ctx)
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

func getBinaryAttributes(mappingPath, uploadHost string) *perforatorstorage.BinaryAttributes {
	mappingPath = strings.TrimSuffix(mappingPath, " (deleted)")
	upload := &perforatorstorage.BinaryUploadMetadata{Host: uploadHost, Path: mappingPath}
	if mappingPath != "" {
		upload.Filename = path.Base(mappingPath)
	}
	return &perforatorstorage.BinaryAttributes{Upload: upload}
}

func (a *processAnalyzer) processMapping(ctx context.Context, m *procfs.Mapping) error {
	mapping := dso.Mapping{Mapping: *m}
	if mapping.Path == "" {
		// Probably JITed mapping.
		mapping.Path = "[JIT]"
		a.preparedMappings = append(a.preparedMappings, &mapping)
		return nil
	}

	if vdso.IsUnsymbolizableVDSOMapping(&mapping.Mapping) {
		a.preparedMappings = append(a.preparedMappings, &mapping)
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

	// Inode generation describes the opened binary, not an atomic procfs snapshot.
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

	a.reg.dsoStorage.PrepareMapping(ctx, &mapping, binary)
	a.preparedMappings = append(a.preparedMappings, &mapping)
	a.registerMapping(&mapping)

	err = a.uploader.ScheduleBinary(buildid, handle, getBinaryAttributes(mapping.Path, a.reg.uploadHost))
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
	// Publication and activation must belong to the same process lifetime.
	a.proc.lifecycleMu.Lock()
	defer a.proc.lifecycleMu.Unlock()

	if a.proc.retired.Load() {
		return fmt.Errorf("process %d was deleted before metadata publication", a.proc.currentNamespaceID)
	}

	if err := a.ensureCurrentIdentity(); err != nil {
		return err
	}
	if a.reg.metadataPublisher != nil {
		// Failed publication or activation must leave the updated mappings
		// unavailable for new sampling, including for an already active PID.
		if err := ignoreMissingBPFEntry(a.reg.state.RemoveProcess(a.proc.currentNamespaceID)); err != nil {
			return fmt.Errorf("failed to disable process before metadata update: %w", err)
		}
	}
	if a.preparedMappings != nil {
		// Committed references remain owned by DSO storage if publication or activation fails.
		a.reg.dsoStorage.ReplaceMappings(ctx, a.proc.currentNamespaceID, a.preparedMappings)
		a.preparedMappings = nil
	}
	if a.envs != nil {
		a.proc.setEnvs(a.envs)
	}
	if a.reg.pidNamespaceIndex != nil {
		a.reg.tryRegisterPidNamespaceCorrelation(ctx, a.proc)
	}
	a.proc.mapslock.Lock()

	sort.Slice(a.exemappings, func(i, j int) bool {
		return a.exemappings[i].Begin < a.exemappings[j].Begin
	})

	err := a.syncMapsLocked(ctx)
	a.proc.mapslock.Unlock()
	if err != nil {
		if a.reg.metadataPublisher == nil {
			if disableErr := ignoreMissingBPFEntry(a.reg.state.RemoveProcess(a.proc.currentNamespaceID)); disableErr != nil {
				err = errors.Join(err, fmt.Errorf("failed to disable process after mapping synchronization: %w", disableErr))
			}
		}
		return fmt.Errorf("failed to synchronize process mappings: %w", err)
	}

	// The publisher may synchronously read process metadata. Publication must
	// succeed before the updated process becomes available for new sampling.
	if a.reg.metadataPublisher != nil {
		if err := a.reg.metadataPublisher.PublishProcess(ctx, a.proc); err != nil {
			return fmt.Errorf("failed to publish process metadata: %w", err)
		}
	}
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
	err = a.reg.state.AddProcess(a.proc.currentNamespaceID, pi)
	if err != nil {
		return err
	}

	// Listener callbacks report a process ready to sample and may synchronously
	// read its metadata.
	a.reg.notifyProcessUpdateLocked(ctx, a.proc)
	return nil
}

func (a *processAnalyzer) syncMapsLocked(ctx context.Context) error {
	visited := make(map[uint64]struct{}, len(a.exemappings))

	toRemove := make([]processMap, 0)
	toAdd := make([]*dso.Mapping, 0)

	for _, m := range a.exemappings {
		if m.DSO == nil {
			continue
		}
		visited[m.Begin] = struct{}{}

		mapping, ok := a.proc.registeredmaps[m.Begin]
		if ok && !mapping.incomplete && mapping.ID() == m.DSO.ID && mapping.end() == m.End {
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
		if err := a.reg.removeBPFMap(ctx, a.proc, m); err != nil {
			return err
		}
	}

	for _, m := range toAdd {
		if err := a.reg.addBPFMap(ctx, a.proc, m); err != nil {
			return err
		}
	}
	return nil
}

func (r *ProcessRegistry) addBPFMap(ctx context.Context, pi *processInfo, m *dso.Mapping) error {
	l := r.log.With(logfield.CurrentNamespacePID(pi.currentNamespaceID)).WithName("lpm")
	l.Debug(ctx, "Trying to add eBPF mapping", log.String("buildid", m.BuildInfo.BuildID))

	id := pi.nextmapid.Add(1)
	pi.registeredmaps[m.Begin] = processMap{Mapping: mappingImpl{m}, id: id, incomplete: true}

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
		return err
	}

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
		return err
	}

	pi.registeredmaps[m.Begin] = processMap{Mapping: mappingImpl{m}, id: id}
	return nil
}

func HostToBigEndian64(value uint64) uint64 {
	var buf [8]byte
	eb.NativeEndian.PutUint64(buf[:], value)
	return eb.BigEndian.Uint64(buf[:])
}

func (r *ProcessRegistry) removeBPFMap(ctx context.Context, pi *processInfo, m processMap) error {
	l := r.log.With(logfield.CurrentNamespacePID(pi.currentNamespaceID)).WithName("lpm")
	l.Debug(ctx, "Trying to remove eBPF mapping", log.String("buildid", m.buildInfo().BuildID))
	m.incomplete = true
	pi.registeredmaps[m.begin()] = m

	err := iterateMappingLPMSegments(m.Mapping, func(address uint64, prefix uint32) error {
		return ignoreMissingBPFEntry(r.state.RemoveMappingLPMSegment(&unwinder.ExecutableMappingTrieKey{
			Prefixlen:     32 + prefix,
			Pid:           uint32(pi.currentNamespaceID),
			AddressPrefix: HostToBigEndian64(address),
		}))
	})
	if err != nil {
		l.Warn(ctx, "Failed to remove eBPF mapping lpm trie segment", log.Error(err))
		return err
	}

	err = ignoreMissingBPFEntry(r.state.RemoveMapping(&unwinder.ExecutableMappingKey{
		Pid: uint32(pi.currentNamespaceID),
		Id:  m.id,
	}))
	if err != nil {
		l.Warn(ctx, "Failed to remove eBPF mapping", log.Error(err))
		return err
	}

	delete(pi.registeredmaps, m.begin())
	return nil
}

func ignoreMissingBPFEntry(err error) error {
	if errors.Is(err, ebpf.ErrKeyNotExist) {
		return nil
	}
	return err
}

func (r *ProcessRegistry) removeProcessMappings(ctx context.Context, pi *processInfo) bool {
	pi.mapslock.Lock()
	defer pi.mapslock.Unlock()
	for _, m := range pi.registeredmaps {
		_ = r.removeBPFMap(ctx, pi, m)
	}
	return len(pi.registeredmaps) == 0
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

// tryLoadEnvs returns the environment and whether it is ready.
// On the first iteration, an empty kernel-thread environment is considered ready.
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

	// A newly created process may expose an empty environment before initialization.
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
			// A successful empty read replaces old values; nil means no successful read.
			if envs == nil {
				envs = make(map[string]string)
			}
			a.envs = envs
			break
		}

		sleepFor := backoff.NextBackOff()
		if sleepFor == backoff.Stop || !waitOnEmptyEnv {
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

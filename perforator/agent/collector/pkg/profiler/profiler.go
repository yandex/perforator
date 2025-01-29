package profiler

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/perfmap"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/internal/linguist/python/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/graceful"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/kallsyms"
	"github.com/yandex/perforator/perforator/pkg/linux/mountinfo"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/linux/uname"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	PerfReaderTimeout = 5 * time.Second
)

type Profiler struct {
	log log.Logger

	conf           *config.Config
	storage        client.Storage
	metrics        profilerMetrics
	processScanner process.ProcessScanner
	sampleCallback machine.RawSampleCallback
	eventListener  EventListener

	bpf          *machine.BPF
	eventmanager *perfevent.EventManager
	mounts       *mountinfo.Watcher
	kallsyms     *kallsyms.KallsymsResolver
	events       map[perfevent.Type]*perfevent.EventBundle
	debugmu      sync.Mutex
	debugmode    bool
	envWhitelist map[string]struct{}
	progready    sync.Once
	perfmap      *perfmap.Registry

	dsoStorage *dso.Storage
	procs      *process.ProcessRegistry

	pythonSymbolizer *symbolizer.Symbolizer

	// Profiling targets
	wholeSystem *multiProfileBuilder
	cgroups     *cgroups.Tracker
	pids        map[int]*trackedProcess
	pidsmu      sync.RWMutex

	profileChan  chan client.LabeledProfile
	commonLabels map[string]string

	wg                      *errgroup.Group
	sampleReaderShutdown    graceful.ShutdownCookie
	profileUploaderShutdown graceful.ShutdownCookie
	ebpfMetricsShutdown     graceful.ShutdownCookie
	shutdownCancel          context.CancelCauseFunc

	podsCgroupTracker *PodsCgroupTracker

	enablePerfMaps    bool
	enablePerfMapsJVM bool
}

type profilerMetrics struct {
	samplesDuration metrics.Counter
	mappingsHit     metrics.Counter
	mappingsMiss    metrics.Counter

	cgroupHits   metrics.Counter
	cgroupMisses metrics.Counter

	resolveTLSErrors           metrics.Counter
	resolveTLSSuccess          metrics.Counter
	recordedTLSVarsFromSamples metrics.Counter
	recordedTLSBytes           metrics.Counter

	unsymbolizedPythonFrameCount metrics.Counter
	collectedPythonFrameCount    metrics.Counter

	droppedProfiles metrics.Counter
}

////////////////////////////////////////////////////////////////////////////////

type option func(p *Profiler) error

func WithStorage(storage client.Storage) option {
	return func(p *Profiler) error {
		if p.storage != nil {
			return fmt.Errorf("refusing to overwrite profiler storage")
		}
		p.storage = storage
		return nil
	}
}

func WithRawSampleCallback(sampleCallback machine.RawSampleCallback) option {
	return func(p *Profiler) error {
		if p.sampleCallback != nil {
			return fmt.Errorf("refusing to overwrite profiler raw sample callback")
		}
		p.sampleCallback = sampleCallback
		return nil
	}
}

func WithEventListener(listener EventListener) option {
	return func(p *Profiler) error {
		p.eventListener = listener
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////

func NewProfiler(c *config.Config, l log.Logger, r metrics.Registry, opts ...option) (*Profiler, error) {
	c.FillDefault()
	l = l.WithName("profiler")

	envWhitelist := make(map[string]struct{})
	for _, env := range c.SampleConsumer.EnvWhitelist {
		envWhitelist[env] = struct{}{}
	}

	profiler := &Profiler{
		conf:         c,
		log:          l,
		mounts:       mountinfo.NewWatcher(l, r),
		events:       make(map[perfevent.Type]*perfevent.EventBundle),
		pids:         make(map[int]*trackedProcess),
		profileChan:  make(chan client.LabeledProfile, 64),
		debugmode:    c.Debug,
		envWhitelist: envWhitelist,

		sampleReaderShutdown:    graceful.NewShutdownCookie(),
		profileUploaderShutdown: graceful.NewShutdownCookie(),
		ebpfMetricsShutdown:     graceful.NewShutdownCookie(),
	}

	scanner := &process.ProcFSScanner{}
	profiler.processScanner = process.NewFilteringProcessScanner(scanner, profiler.shouldDiscoverProcess)

	for _, opt := range opts {
		err := opt(profiler)
		if err != nil {
			return nil, err
		}
	}

	if c.PodsDeploySystemConfig != nil && c.PodsDeploySystemConfig.DeploySystem != "" {
		podsCgroupTracker, err := newPodsCgroupTracker(c.PodsDeploySystemConfig, l)
		if err != nil {
			return nil, err
		}
		profiler.podsCgroupTracker = podsCgroupTracker
	}

	err := profiler.initialize(r)
	if err != nil {
		l.Error("Failed to initialize profiler", log.Error(err))
		return nil, err
	}

	l.Info("Successfully initialized profiler")
	return profiler, nil
}

func (p *Profiler) shouldDiscoverProcess(pid linux.ProcessID) bool {
	if !p.conf.ProcessDiscovery.IgnoreUnrelatedProcesses {
		return true
	}

	if p.wholeSystem != nil {
		return true
	}

	// FIXME(sskvor): Check process cgroup.
	if p.cgroups.NumCgroupNames() > 0 {
		return true
	}

	p.pidsmu.Lock()
	_, found := p.pids[int(pid)]
	p.pidsmu.Unlock()

	return found
}

// Initialize the profiler.
// Prepare and load eBPF programs, tune rlimits, ...
func (p *Profiler) initialize(r metrics.Registry) (err error) {
	// Load eBPF programs
	p.bpf, err = machine.NewBPF(&p.conf.BPF, p.log, r)
	if err != nil {
		return fmt.Errorf("failed to initialize eBPF subsystem: %w", err)
	}

	// Prepare perf event manager
	p.eventmanager, err = perfevent.NewEventManager(p.log, r)
	if err != nil {
		return fmt.Errorf("failed to initialize perf event subsystem: %w", err)
	}

	// Setup system-wide perf events
	err = p.setupPerfEvents()
	if err != nil {
		return fmt.Errorf("failed to setup perf events: %w", err)
	}

	// Link perf events with the eBPF program.
	err = p.installPerfEventBPF()
	if err != nil {
		return fmt.Errorf("failed to link bpf program with perf events: %w", err)
	}

	// Load common profile labels (e.g. nodename or cpu model).
	err = p.setupCommonProfileLabels()
	if err != nil {
		return fmt.Errorf("failed to setup common profile labels: %w", err)
	}

	// Create storage client
	if p.storage == nil {
		if p.conf.StorageClientConfig != nil {
			// Create remote storage client
			p.storage, err = client.NewRemoteStorage(p.conf.StorageClientConfig, xlog.New(p.log), r)
			if err != nil {
				return fmt.Errorf("failed to create storage client: %w", err)
			}
		} else if p.conf.LocalStorageConfig != nil {
			// Create local storage
			p.storage, err = client.NewLocalStorage(p.conf.LocalStorageConfig, p.log)
			if err != nil {
				return fmt.Errorf("failed to create local storage: %w", err)
			}
		} else if p.conf.InMemoryStorage != nil {
			p.storage = client.NewInMemoryStorage(p.conf.InMemoryStorage)
		} else {
			p.log.Warn("Creating dummy storage, not saving profiles")
			p.storage = &client.DummyStorage{}
		}
	}

	// Create python symbolizer
	if enabled := p.conf.BPF.TracePython; enabled == nil || *enabled {
		p.pythonSymbolizer, err = symbolizer.NewSymbolizer(&p.conf.Symbolizer.Python, p.bpf, r)
		if err != nil {
			return err
		}
	}
	p.enablePerfMaps = true
	p.enablePerfMapsJVM = true
	if p.conf.EnablePerfMaps != nil {
		p.enablePerfMaps = *p.conf.EnablePerfMaps
	}
	if p.conf.EnablePerfMapsJVM != nil {
		p.enablePerfMapsJVM = *p.conf.EnablePerfMapsJVM
	}
	if p.enablePerfMaps {
		p.perfmap = perfmap.NewRegistry(p.log, r, p.enablePerfMapsJVM)
	}

	processListeners := []process.Listener{}

	if p.enablePerfMaps {
		processListeners = append(processListeners, p.perfmap)
	}

	bpfManager, err := binary.NewBPFBinaryManager(p.log.WithName("ProcessRegistry"), r.WithPrefix("ProcessRegistry"), p.bpf)
	if err != nil {
		return fmt.Errorf("failed to create bpf binary manager: %w", err)
	}

	p.dsoStorage, err = dso.NewStorage(xlog.New(p.log.WithName("ProcessRegistry")), r.WithPrefix("ProcessRegistry"), bpfManager)
	if err != nil {
		return fmt.Errorf("failed to create dso storage: %w", err)
	}

	// Setup process registry.
	p.procs, err = process.NewProcessRegistry(
		xlog.New(p.log.WithName("ProcessRegistry")),
		r.WithPrefix("ProcessRegistry"),
		p.bpf,
		p.mounts,
		p.dsoStorage,
		&process.UploaderArguments{
			Conf:    p.conf.UploadSchedulerConfig,
			Storage: p.storage,
		},
		p.processScanner,
		processListeners,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize process registry: %w", err)
	}

	// Load kallsyms to map kernel addresses to symbols later.
	p.log.Info("Loading kallsyms")
	p.kallsyms, err = kallsyms.DefaultKallsymsResolver()
	if err != nil {
		return fmt.Errorf("failed to load kallsyms: %w", err)
	}
	p.log.Info("Successfully loaded kallsyms", log.Int("num_symbols", p.kallsyms.Size()))

	// We use cgroup names to identify pods in the system.
	p.log.Info("Loading cgroupsfs state")
	p.cgroups, err = cgroups.NewTracker(p.log, &p.conf.Cgroups)
	if err != nil {
		return fmt.Errorf("failed to load cgroupfs: %w", err)
	}
	p.log.Info("Loaded cgroupfs state", log.String("cgroupfs_version", p.cgroups.CgroupVersion().String()))

	// Prepare eBPF config.
	p.log.Info("Preparing profiler config")
	err = p.setupConfig()
	if err != nil {
		return fmt.Errorf("failed to setup profiler config: %w", err)
	}

	// Register metrics.
	err = p.registerMetrics(r)
	if err != nil {
		return fmt.Errorf("failed to register metrics: %w", err)
	}

	// We are done.
	return nil
}

func (p *Profiler) setupPerfEvents() error {
	for _, event := range p.conf.PerfEvents {
		event := event

		p.log.Debug("Trying to open perf event bundle", log.Any("config", event))

		if p.events[event.Type] != nil {
			return fmt.Errorf("duplicate perf event type %s", event.Type)
		}

		target := &perfevent.Target{
			WholeSystem: true,
		}

		options := &perfevent.Options{
			Type:                   event.Type,
			Frequency:              event.Frequency,
			SampleRate:             event.SampleRate,
			TryToSampleBranchStack: perfevent.ShouldTryToEnableBranchSampling(),
		}

		bundle, err := p.eventmanager.Open(target, options)
		if err != nil {
			return fmt.Errorf("failed to create perf event bundle: %w", err)
		}

		p.events[event.Type] = bundle
		p.log.Debug("Successfully opened perf event bundle")
	}

	return nil
}

func (p *Profiler) installPerfEventBPF() error {
	for _, bundle := range p.events {
		err := bundle.AttachBPF(p.bpf.ProfilerProgramFD())
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Profiler) registerMetrics(r metrics.Registry) error {
	type Labels map[string]string
	p.metrics.samplesDuration = r.Counter("sample_duration.nsec")

	mappings := r.CounterVec("mapping_resolving.count", []string{"status"})
	p.metrics.mappingsHit = mappings.With(Labels{"status": "hit"})
	p.metrics.mappingsMiss = mappings.With(Labels{"status": "miss"})

	tls := r.CounterVec("tls_name_resolving.count", []string{"status"})
	p.metrics.resolveTLSSuccess = tls.With(Labels{"status": "success"})
	p.metrics.resolveTLSErrors = tls.With(Labels{"status": "fail"})

	p.metrics.recordedTLSVarsFromSamples = r.Counter("tls.variables_recorded.count")
	p.metrics.recordedTLSBytes = r.Counter("tls.variables_recorded.bytes")

	p.metrics.droppedProfiles = r.WithTags(Labels{"kind": "dropped"}).Counter("profiles.count")

	p.metrics.unsymbolizedPythonFrameCount = r.Counter("python.frame.unsymbolized.count")
	p.metrics.collectedPythonFrameCount = r.Counter("python.frame.collected.count")

	r.WithTags(Labels{"kind": "tracked"}).FuncIntGauge("cgroup.count", func() int64 {
		if p.cgroups == nil {
			return 0
		}
		return int64(p.cgroups.NumCgroupNames())
	})

	p.metrics.cgroupHits = r.Counter("cgroup.cache.hit.count")
	p.metrics.cgroupMisses = r.Counter("cgroup.cache.miss.count")

	r.FuncGauge("ebpf.memlocked.bytes", func() float64 {
		if p.bpf == nil {
			return 0.0
		}
		count, err := p.bpf.CountMemLockedBytes()
		if err != nil {
			p.log.Error("Failed to count memlocked bytes", log.UInt64("bytes", count))
			return 0.0
		}
		return float64(count)
	})

	return nil
}

func (p *Profiler) setupCommonProfileLabels() error {
	p.commonLabels = make(map[string]string)

	uname, err := uname.Load()
	if err != nil {
		return fmt.Errorf("failed to load kernel release name: %w", err)
	}
	p.commonLabels["kernel"] = uname.Release
	p.commonLabels["host"] = uname.NodeName

	return nil
}

func (p *Profiler) setupConfig() error {
	conf := &unwinder.ProfilerConfig{
		// Do not collect samples from the kernel threads.
		TraceKthreads: false,

		// Sample 1/100 of sched events to reduce overhead.
		SchedSampleModulo: 100,
	}

	cgroupVersion := p.cgroups.CgroupVersion()
	switch cgroupVersion {
	case cgroups.CgroupV1:
		conf.ActiveCgroupEngine = unwinder.CgroupEngineV1
	case cgroups.CgroupV2:
		conf.ActiveCgroupEngine = unwinder.CgroupEngineV2
	default:
		return fmt.Errorf("unsupported cgroup version %v", cgroupVersion)
	}
	p.log.Info("Selected cgroup engine", log.String("engine", conf.ActiveCgroupEngine.String()))

	// Record current pidns.
	pidns, err := procfs.Self().GetNamespaces().GetPidInode()
	if err != nil {
		p.log.Error("Failed to resolve self pid namespace inode number", log.Error(err))
		conf.PidnsInode = 0
	} else {
		p.log.Debug("Resolved self pid namespace inode number", log.UInt64("inode", pidns))
		conf.PidnsInode = uint32(pidns)
	}

	// Setup signal mask
	p.log.Debug("Trying to set signal mask", log.Strings("signals", p.conf.Signals))
	for _, signal := range p.conf.Signals {
		signo := unix.SignalNum(signal)
		if signo == 0 {
			return fmt.Errorf("unknown signal %s", signal)
		}
		if int(signo) >= int(unwinder.SignalMaskBits) {
			return fmt.Errorf("unsupported signal %s: value %d does not fit in mask", signal, int(signo))
		}
		conf.SignalMask |= 1 << int(signo)
	}

	p.log.Info("Configuring the profiler", log.Any("config", conf))
	err = p.bpf.UpdateConfig(conf)
	if err != nil {
		return fmt.Errorf("failed to configure the profiler: %w", err)
	}

	return nil
}

func (p *Profiler) handleWorkerError(ctx context.Context, err error, workerName string) error {
	l := log.With(p.log, log.String("worker", workerName))

	if err == nil {
		l.Debug("Worker finished")
		return nil
	}

	if errors.Is(err, context.Canceled) && context.Cause(ctx) == ErrStopped {
		l.Debug("Worker gracefully stopped")
		return nil
	}

	l.Error("Worker failed", log.Error(err))
	return err
}

var ErrStopped = errors.New("profiler is stopped")

// Start main profiler routine.
// Run will block until ctx is cancelled or an unrecoverrable error is encountered.
func (p *Profiler) Run(ctx context.Context) error {
	err := p.Start(ctx)
	if err != nil {
		return fmt.Errorf("failed to start profile: %w", err)
	}

	err = p.Wait()
	if err != nil {
		return err
	}

	return nil
}

func (p *Profiler) Start(ctx context.Context) error {
	if p.wg != nil {
		return fmt.Errorf("profiler is already running")
	}
	ctx, p.shutdownCancel = context.WithCancelCause(ctx)

	err := p.enablePerfEvents()
	if err != nil {
		return fmt.Errorf("failed to enable perf events: %w", err)
	}

	p.wg, ctx = errgroup.WithContext(ctx)
	if p.enablePerfMaps {
		p.wg.Go(func() error {
			err := p.perfmap.Run(ctx)
			return p.handleWorkerError(ctx, err, "perf map manager")
		})
	}
	p.wg.Go(func() error {
		err := p.runSampleReader(ctx)
		return p.handleWorkerError(ctx, err, "sample reader")
	})
	p.wg.Go(func() error {
		err := p.runProfileSender(ctx)
		return p.handleWorkerError(ctx, err, "profile sender")
	})
	p.wg.Go(func() error {
		err := p.mounts.RunPoller(ctx)
		return p.handleWorkerError(ctx, err, "mount info poller")
	})
	p.wg.Go(func() error {
		err := p.procs.RunProcessPoller(ctx)
		return p.handleWorkerError(ctx, err, "process poller")
	})
	p.wg.Go(func() error {
		err := p.cgroups.RunPoller(ctx)
		return p.handleWorkerError(ctx, err, "cgroup tracker")
	})
	p.wg.Go(func() error {
		err := p.bpf.RunMetricsPoller(ctx, p.ebpfMetricsShutdown.GetSource())
		return p.handleWorkerError(ctx, err, "ebpf metrics pusher")
	})
	if p.podsCgroupTracker != nil {
		p.wg.Go(func() error {
			err := p.runPodsCgroupTracker(ctx)
			return p.handleWorkerError(ctx, err, "pods cgroup tracker")
		})
	}

	concurrency := 4
	if c := p.conf.ProcessDiscovery.Concurrency; c != 0 {
		concurrency = c
	}

	for i := 0; i < concurrency; i++ {
		p.wg.Go(func() error {
			err := p.runProcessDiscovery(ctx)
			return p.handleWorkerError(ctx, err, "process discovery")
		})

		p.wg.Go(func() error {
			err := p.procs.RunWorker(ctx)
			return p.handleWorkerError(ctx, err, "process analyzer")
		})
	}

	return nil
}

func (p *Profiler) enablePerfEvents() error {
	p.log.Debug("Enabling perf events")

	for _, bundle := range p.events {
		err := bundle.Enable()
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Profiler) disablePerfEvents() error {
	p.log.Debug("Disabling perf events")

	for _, bundle := range p.events {
		err := bundle.Disable()
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Profiler) openSampleReader(watermark int, sampleCallback machine.RawSampleCallback) (*machine.PerfReader, error) {
	opts := &machine.PerfReaderOptions{
		PerCPUBufferSize: *p.conf.SampleConsumer.PerfBufferPerCPUSize,
		Watermark:        watermark,
		SampleCallback:   sampleCallback,
	}
	return p.bpf.MakeSampleReader(opts)
}

func (p *Profiler) runSampleReader(ctx context.Context) error {
	stopSource := p.sampleReaderShutdown.GetSource()
	defer stopSource.Finish()

	reader, err := p.openSampleReader(*p.conf.SampleConsumer.PerfBufferWatermark, p.sampleCallback)
	if err != nil {
		return err
	}
	defer reader.Close()

	var sample unwinder.RecordSample

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stopSource.Done():
			goto gracefulstop
		default:
		}

		p.readSample(ctx, reader, &sample)
	}

gracefulstop:
	p.log.Debug("Graceful shutdown has been requested, going to drain sample queue")

	for p.readSample(ctx, reader, &sample) {
		// drain sample queue
	}

	p.log.Debug("Restarting sample reader in order to consume last non-notified samples")
	_ = reader.Close()
	reader, err = p.openSampleReader(0, nil)
	if err != nil {
		return err
	}

	for p.readSample(ctx, reader, &sample) {
		// drain sample queue once again
	}

	return nil
}

func (p *Profiler) readSample(ctx context.Context, reader *machine.PerfReader, sample *unwinder.RecordSample) bool {
	err := reader.Read(ctx, sample)
	if err != nil {
		return false
	}

	p.metrics.samplesDuration.Add(int64(sample.Runtime))

	NewSampleConsumer(p, p.envWhitelist, sample).Consume(ctx)

	return true
}

func (p *Profiler) finishAllProfiles(ctx context.Context) {
drainloop:
	for {
		var profile client.LabeledProfile

		select {
		case profile = <-p.profileChan:
		default:
			break drainloop
		}

		p.trySaveProfile(ctx, profile)
	}

	p.pidsmu.Lock()
	defer p.pidsmu.Unlock()

	if p.wholeSystem != nil {
		p.log.Info("Finishing whole system profile")
		p.trySaveProfiles(ctx, p.wholeSystem.RestartProfiles())
	}

	for pid, process := range p.pids {
		p.log.Info("Finishing process profile", log.Int("pid", pid))
		p.trySaveProfiles(ctx, process.builder.RestartProfiles())
	}

	_ = p.cgroups.ForEachCgroup(func(event cgroups.CgroupEventListener) error {
		cgroup := event.(*trackedCgroup)
		p.log.Info("Finishing cgroup profile", log.String("cgroup", cgroup.conf.Name))
		p.trySaveProfiles(ctx, cgroup.builder.RestartProfiles())
		return nil
	})
}

func (p *Profiler) flushProfile(profile client.LabeledProfile) bool {
	p.log.Debug("Flushing profile",
		log.Any("labels", profile.Labels),
		log.Int("samples", len(profile.Profile.Sample)),
	)
	select {
	case p.profileChan <- profile:
		return true
	default:
		return false
	}
}

func (p *Profiler) runProfileSender(ctx context.Context) error {
	stopSource := p.profileUploaderShutdown.GetSource()
	defer stopSource.Finish()

	var profile client.LabeledProfile
	for i := 0; ; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stopSource.Done():
			goto gracefulstop
		case profile = <-p.profileChan:
		}

		p.trySaveProfile(ctx, profile)
	}

gracefulstop:
	p.log.Debug("Graceful shutdown has been requested, going to drain profile queue")
	p.finishAllProfiles(ctx)
	return nil
}

func (p *Profiler) trySaveProfiles(ctx context.Context, profiles labeledAgentProfiles) {
	for _, profile := range profiles.Profiles {
		p.trySaveProfile(ctx, client.LabeledProfile{
			Profile: profile,
			Labels:  profiles.Labels,
		})
	}
}

func (p *Profiler) trySaveProfile(ctx context.Context, profile client.LabeledProfile) {
	if len(profile.Profile.Sample) == 0 {
		p.log.Debug("Skipping empty profile", log.Any("labels", profile.Labels))
		return
	}

	err := p.storage.StoreProfile(ctx, profile)
	if err != nil {
		p.log.Error("Failed to save profile", log.Error(err))
		return
	}
	p.log.Info("Saved profile",
		log.Any("labels", profile.Labels),
		log.Int("samples", len(profile.Profile.Sample)),
	)
	if p.eventListener != nil {
		for _, s := range profile.Profile.Sample {
			pidList, ok := s.NumLabel["pid"]
			if !ok {
				p.log.Error("Missing pid label in profile", log.Any("actual", s.NumLabel))
				continue
			}
			if len(pidList) != 1 {
				p.log.Error("Unexpected pid label count", log.Int64s("actual", pidList))
				continue
			}
			pid := pidList[0]
			p.eventListener.OnSampleStored(linux.ProcessID(pid))
		}
	}
}

func (p *Profiler) runProcessDiscovery(ctx context.Context) error {
	r, err := p.bpf.MakeProcessReader(&machine.PerfReaderOptions{
		PerCPUBufferSize: 16 * 1024,
		Watermark:        0,
	})
	if err != nil {
		return err
	}
	defer r.Close()

	var sample unwinder.RecordNewProcess
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := r.Read(ctx, &sample)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				p.log.Error("Failed to read sample", log.Error(err))
			}
			continue
		}

		p.log.Debug("Got new process",
			log.UInt32("pid", sample.Pid),
			log.UInt64("starttime", sample.Starttime),
		)
		p.procs.DiscoverProcess(ctx, linux.ProcessID(sample.Pid))
	}
}

// Register cgroup in the profiler.
// If cgroup name is empty, trace whole system.
// Thread safety: it is safe to run AddCgroup concurrently with Run/AddCgroup.
// Use porto/ prefix instead of porto% (like in /sys/fs/cgroup/freezer hierarchy)
func (p *Profiler) AddCgroup(conf *CgroupConfig) error {
	if conf == nil {
		conf = &CgroupConfig{}
	}

	conf.Labels = p.enrichProfileLabels(conf.Labels)

	cg, err := newTrackedCgroup(conf, p.bpf, p.log)
	if err != nil {
		return err
	}

	return p.cgroups.AddCgroup(&cgroups.TrackedCgroup{
		Name:  conf.Name,
		Event: cg,
	}, true /*=reopenEventIfExists*/)
}

func (p *Profiler) TraceWholeSystem(labels map[string]string) error {
	labels = p.enrichProfileLabels(labels)
	p.wholeSystem = newMultiProfileBuilder(labels)
	return p.bpf.PatchConfig(func(conf *unwinder.ProfilerConfig) error {
		conf.TraceWholeSystem = true
		return nil
	})
}

func (p *Profiler) TraceSelf(labels map[string]string) error {
	return p.TracePid(os.Getpid(), labels)
}

func (p *Profiler) TracePid(pid int, labels map[string]string) error {
	labels = p.enrichProfileLabels(labels)

	if pid == 0 {
		pid = os.Getpid()
	}

	trackedProcess, err := newTrackedProcess(pid, labels, p.bpf)
	if err != nil {
		return err
	}

	p.pidsmu.Lock()
	p.pids[pid] = trackedProcess
	p.pidsmu.Unlock()

	p.log.Info("Registered process", log.Int("pid", pid))
	return nil
}

func (p *Profiler) DeleteCgroup(name string) error {
	return p.cgroups.Delete(name)
}

func (p *Profiler) TraceCgroups(configs []*CgroupConfig) error {
	trackedCgroups := make([]*cgroups.TrackedCgroup, 0, len(configs))
	for _, conf := range configs {
		conf.Labels = p.enrichProfileLabels(conf.Labels)

		profiledCgroup, err := newTrackedCgroup(conf, p.bpf, p.log)
		if err != nil {
			return err
		}

		trackedCgroups = append(
			trackedCgroups,
			&cgroups.TrackedCgroup{
				Name:  conf.Name,
				Event: profiledCgroup,
			},
		)
	}

	return p.cgroups.TrackCgroups(trackedCgroups)
}

func (p *Profiler) SetDebugMode(debug bool) (err error) {
	p.debugmu.Lock()
	defer p.debugmu.Unlock()

	if p.debugmode == debug {
		return nil
	}

	defer func() {
		if err == nil {
			p.debugmode = debug
		}
	}()

	p.log.Warn("Toggling debug mode", log.Bool("enabled", debug))

	err = p.bpf.ReloadProgram(debug)
	if err != nil {
		return fmt.Errorf("failed to reload program: %w", err)
	}

	err = p.installPerfEventBPF()
	if err != nil {
		return fmt.Errorf("failed to install new program to the perf events: %w", err)
	}

	err = p.enablePerfEvents()
	if err != nil {
		return fmt.Errorf("failed to enable perf events: %w", err)
	}

	return err
}

func (p *Profiler) enrichProfileLabels(labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	} else {
		labels = maps.Clone(labels)
	}

	for k, v := range p.commonLabels {
		if labels[k] == "" {
			labels[k] = v
		}
	}

	return labels
}

func (p *Profiler) Storage() client.Storage {
	return p.storage
}

func (p *Profiler) Close() error {
	return p.bpf.Close()
}

func (p *Profiler) Stop(ctx context.Context) error {
	// Shutdown sequence:
	// 1. Disable any active perf events.
	// 2. Disable any active eBPF program.
	// 3. Drain sample queue
	// 4. Drain profile queue
	// 5. Abort any running background job (e.g. process, mountinfo and cgroup pollers)

	err := p.disablePerfEvents()
	if err != nil {
		p.log.Error("Failed to disable perf events", log.Error(err))
	}

	err = p.bpf.UnlinkPrograms()
	if err != nil {
		p.log.Error("Failed to disable eBPF programs", log.Error(err))
	}

	p.log.Info("Stopping sample reader")
	err = p.sampleReaderShutdown.Stop(ctx)
	if err != nil {
		return err
	}

	p.log.Info("Stopping profile uploader")
	err = p.profileUploaderShutdown.Stop(ctx)
	if err != nil {
		return err
	}

	p.log.Info("Stopping eBPF metrics calculator")
	err = p.ebpfMetricsShutdown.Stop(ctx)
	if err != nil {
		return err
	}

	p.log.Info("Cancelling background workers context")
	if p.shutdownCancel != nil {
		p.shutdownCancel(ErrStopped)
	}

	p.log.Info("Waiting for background workers to stop")
	return p.Wait()
}

func (p *Profiler) Wait() error {
	return p.wg.Wait()
}

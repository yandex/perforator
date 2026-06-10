package profiler

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path"
	"sync"
	"time"

	"github.com/klauspost/cpuid/v2"
	"golang.org/x/sync/errgroup"
	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/unwindtable"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/perfmap"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profilerext"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	preprocessig_proto "github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	agent_gateway_client "github.com/yandex/perforator/perforator/internal/agent_gateway/client"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmregistry"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/graceful"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/clock"
	"github.com/yandex/perforator/perforator/pkg/linux/kallsyms"
	"github.com/yandex/perforator/perforator/pkg/linux/mountinfo"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/linux/uname"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	PerfReaderTimeout      = 5 * time.Second
	mainSampleConsumerName = "main_sample_consumer"
)

type initialTargets struct {
	selfTargetLabels map[string]string
	cgroupTargets    []*CgroupConfig
	processTargets   []processTarget
	threadTargets    []threadTarget
}

type processTarget struct {
	pid    int
	labels map[string]string
}

type threadTarget struct {
	tid    int
	labels map[string]string
}

type Profiler struct {
	log log.Logger
	reg metrics.Registry

	conf           *config.Config
	storage        client.Storage
	metrics        profilerMetrics
	processScanner process.ProcessScanner
	sampleCallback machine.RawSampleCallback
	eventListener  EventListener
	initialTargets *initialTargets

	sampleProcessor        *sampleProcessor
	sampleConsumerRegistry SampleConsumerRegistry

	bpf              *machine.BPF
	eventmanager     *perfevent.EventManager
	perfEventManager *PerfEventManager
	uprobeRegistry   *uprobeRegistry
	mounts           *mountinfo.Watcher
	kallsyms         *kallsyms.KallsymsResolver
	events           map[perfevent.Type]*PerfEvent
	// uprobes which are created on Profiler startup
	initialUprobes []Uprobe
	debugmu        sync.Mutex
	debugmode      bool
	envWhitelist   map[string]struct{}
	progready      sync.Once
	perfmap        *perfmap.Registry

	processListeners []process.Listener

	dsoStorage *dso.Storage
	procs      *process.ProcessRegistry

	pythonSymbolizer *symbolizer.Symbolizer
	phpSymbolizer    *symbolizer.Symbolizer

	jitSymbolizers []profilerext.JITSymbolizer

	jvmRegistry *jvmregistry.Registry

	// Profiling targets
	wholeSystem SampleConsumer
	cgroups     *cgroups.Tracker
	pids        map[linux.CurrentNamespacePID]*trackedProcess
	pidsmu      sync.RWMutex

	mainSampleConsumer SampleConsumer

	profileChan  chan client.LabeledProfile
	commonLabels map[string]string

	wg                      *errgroup.Group
	profileUploaderShutdown graceful.ShutdownCookie
	ebpfMetricsShutdown     graceful.ShutdownCookie
	shutdownCancel          context.CancelCauseFunc

	podsCgroupTracker *PodsCgroupTracker

	enablePerfMaps    bool
	enablePerfMapsJVM bool

	identity string

	clockConverter *clock.MonotonicClockConverter
}

type languageCollectionMetrics struct {
	unsymbolizedFrameCount metrics.Counter
	collectedFrameCount    metrics.Counter
}

type profilerMetrics struct {
	mappingsHit  metrics.Counter
	mappingsMiss metrics.Counter

	cgroupHits   metrics.Counter
	cgroupMisses metrics.Counter

	resolveTLSErrors           metrics.Counter
	resolveTLSSuccess          metrics.Counter
	recordedTLSVarsFromSamples metrics.Counter
	recordedTLSBytes           metrics.Counter

	unresolvedPerfEventsForSamples metrics.Counter

	pythonMetrics languageCollectionMetrics
	phpMetrics    languageCollectionMetrics

	droppedProfiles metrics.Counter

	sampleProcessingLatencySum metrics.Counter
	sampleProcessingCount      metrics.Counter
}

////////////////////////////////////////////////////////////////////////////////

type Option func(p *Profiler) error

func WithStorage(storage client.Storage) Option {
	return func(p *Profiler) error {
		if p.storage != nil {
			return fmt.Errorf("refusing to overwrite profiler storage")
		}
		p.storage = storage
		return nil
	}
}

func WithRawSampleCallback(sampleCallback machine.RawSampleCallback) Option {
	return func(p *Profiler) error {
		if p.sampleCallback != nil {
			return fmt.Errorf("refusing to overwrite profiler raw sample callback")
		}
		p.sampleCallback = sampleCallback
		return nil
	}
}

func WithEventListener(listener EventListener) Option {
	return func(p *Profiler) error {
		p.eventListener = listener
		return nil
	}
}

func WithProcessListener(listener process.Listener) Option {
	return func(p *Profiler) error {
		p.processListeners = append(p.processListeners, listener)
		return nil
	}
}

func WithSelfTarget(labels map[string]string) Option {
	return func(p *Profiler) error {
		p.initialTargets.selfTargetLabels = labels
		return nil
	}
}

func WithCgroupTarget(config *CgroupConfig) Option {
	return func(p *Profiler) error {
		p.initialTargets.cgroupTargets = append(p.initialTargets.cgroupTargets, config)
		return nil
	}
}

func WithProcessTarget(pid int, labels map[string]string) Option {
	return func(p *Profiler) error {
		p.initialTargets.processTargets = append(p.initialTargets.processTargets, processTarget{
			pid:    pid,
			labels: labels,
		})
		return nil
	}
}

func WithThreadTarget(tid int, labels map[string]string) Option {
	return func(p *Profiler) error {
		p.initialTargets.threadTargets = append(p.initialTargets.threadTargets, threadTarget{
			tid:    tid,
			labels: labels,
		})
		return nil
	}
}

func WithIdentity(id string) Option {
	return func(p *Profiler) error {
		p.identity = id
		return nil
	}
}

////////////////////////////////////////////////////////////////////////////////

func validateConfig(c *config.Config) error {
	if c.BPF.TraceWallTime == nil || *c.BPF.TraceWallTime {
		foundCPUCyclesPerfEvent := false
		for _, perfEventConfig := range c.PerfEvents {
			resolvedPerfEvent := perfevent.GetTypeByNameOrAlias(perfEventConfig.Type)
			if resolvedPerfEvent.Name() == perfevent.CPUCycles.Name() {
				foundCPUCyclesPerfEvent = true
				break
			}
		}

		if !foundCPUCyclesPerfEvent {
			return errors.New("CPUCycles perf event must be configured to trace wall time")
		}
	}

	return nil
}

func NewProfiler(c *config.Config, l log.Logger, r metrics.Registry, opts ...Option) (*Profiler, error) {
	c.FillDefault()
	l = l.WithName("profiler")

	if err := validateConfig(c); err != nil {
		return nil, fmt.Errorf("invalid profiler config: %w", err)
	}

	clockConverter, err := clock.NewMonotonicClockConverter(l, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create clock converter: %w", err)
	}

	envWhitelist := make(map[string]struct{})
	for _, env := range c.SampleConsumer.EnvWhitelist {
		envWhitelist[env] = struct{}{}
	}

	profiler := &Profiler{
		conf: c,
		log:  l,
		reg:  r,

		mounts:         mountinfo.NewWatcher(l, r),
		events:         make(map[perfevent.Type]*PerfEvent),
		pids:           make(map[linux.CurrentNamespacePID]*trackedProcess),
		profileChan:    make(chan client.LabeledProfile, 64),
		debugmode:      c.Debug,
		envWhitelist:   envWhitelist,
		initialTargets: &initialTargets{},

		profileUploaderShutdown: graceful.NewShutdownCookie(),
		ebpfMetricsShutdown:     graceful.NewShutdownCookie(),
		clockConverter:          clockConverter,
	}

	scanner := &process.ProcFSScanner{}
	profiler.processScanner = process.NewFilteringProcessScanner(scanner, profiler.shouldDiscoverProcess)

	for _, opt := range opts {
		err := opt(profiler)
		if err != nil {
			return nil, err
		}
	}

	err = profiler.initialize(r)
	if err != nil {
		l.Error("Failed to initialize profiler", log.Error(err))
		return nil, err
	}

	l.Info("Successfully initialized profiler")
	return profiler, nil
}

func (p *Profiler) shouldDiscoverProcess(pid linux.CurrentNamespacePID) bool {
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

	p.pidsmu.RLock()
	_, found := p.pids[pid]
	p.pidsmu.RUnlock()

	return found
}

func (p *Profiler) initializeStorage(r metrics.Registry) (err error) {
	if p.conf.StorageClientConfigDeprecated != nil {
		conf := p.conf.StorageClientConfigDeprecated

		l := xlog.Wrap(p.log)

		agentGatewayClient, err := agent_gateway_client.NewGatewayClient(conf, l)
		if err != nil {
			return fmt.Errorf("failed to create agent gateway client: %w", err)
		}

		p.storage = client.NewRemoteStorage(
			l,
			r,
			agentGatewayClient.StorageClient,
			p.conf.FeatureFlagsConfig.GetProfileFormat(),
		)
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

	return nil
}

// Initialize the profiler.
// Prepare and load eBPF programs, tune rlimits, ...
func (p *Profiler) initialize(r metrics.Registry) (err error) {
	if p.conf.JVM.Identity != "" {
		p.identity = p.conf.JVM.Identity
	}
	if p.conf.BPF.PinPrefix == "" {
		p.conf.BPF.PinPrefix = p.identity
	}
	// Load eBPF programs
	p.bpf, err = machine.NewBPF(
		&p.conf.BPF,
		p.log,
		r,
		machine.Options{
			EnableJVM: p.conf.FeatureFlagsConfig.JVMEnabled(),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to initialize eBPF subsystem: %w", err)
	}

	// Prepare perf event manager
	p.eventmanager, err = perfevent.NewEventManager(p.log, r)
	if err != nil {
		return fmt.Errorf("failed to initialize perf event subsystem: %w", err)
	}

	p.perfEventManager = NewPerfEventManager(p.bpf, p.eventmanager)

	// Prepare uprobe registry
	p.uprobeRegistry = newUprobeRegistry(p.bpf)

	p.events = make(map[perfevent.Type]*PerfEvent)

	// Setup system-wide perf events
	err = p.setupPerfEvents()
	if err != nil {
		return fmt.Errorf("failed to setup perf events: %w", err)
	}

	// Setup configured uprobes
	p.setupUprobes()

	// Link perf events with the eBPF program.
	err = p.installPerfEventBPF()
	if err != nil {
		return fmt.Errorf("failed to link bpf program with perf events: %w", err)
	}

	// Initialize AMD-Family19h-specific branch-stack sampling, if relevant.
	//
	// AMD Fam19h processors have a very limited BRS (analogue of Intel LBR) support.
	// Due to its limitations, kernel can only sample last branch records when
	// 1. sampling with a specified period (sampling with frequency doesn't work)
	// 2. sampling an AMD-specific event
	// Thus, we have to create an additional perf-event, to which we only attach
	// the LBR-collecting ebpf-program.
	if p.conf.BPF.TraceLBR != nil && *p.conf.BPF.TraceLBR && p.conf.BPF.TraceLBROnAMD != nil && *p.conf.BPF.TraceLBROnAMD {
		p.maybeInitializeAmdFam19hBRSPerfEvent()
	}

	// Load common profile labels (e.g. nodename or cpu model).
	err = p.setupCommonProfileLabels()
	if err != nil {
		return fmt.Errorf("failed to setup common profile labels: %w", err)
	}

	// Initialize storage
	if p.storage == nil {
		err = p.initializeStorage(r)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
	}

	// Create python symbolizer
	if enabled := p.conf.BPF.TracePython; enabled == nil || *enabled {
		p.pythonSymbolizer, err = symbolizer.NewPythonSymbolizer(&p.conf.Symbolizer.Python, p.bpf.State(), r)
		if err != nil {
			return err
		}
	}

	// Create PHP symbolizer
	if p.conf.FeatureFlagsConfig.PhpEnabled() {
		p.phpSymbolizer, err = symbolizer.NewPhpSymbolizer(&p.conf.Symbolizer.Php, p.bpf.State(), r)
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

	if p.enablePerfMaps {
		p.processListeners = append(p.processListeners, p.perfmap)
		p.jitSymbolizers = append(p.jitSymbolizers, p.perfmap)
	}
	// TODO: ProcessRegistry prefix seems wrong
	unwmanager, err := unwindtable.NewBPFManager(p.log.WithName("ProcessRegistry"), r.WithPrefix("ProcessRegistry"), p.bpf.State())
	if err != nil {
		return fmt.Errorf("failed to create unwind table manager: %w", err)
	}

	if p.conf.FeatureFlagsConfig.JVMEnabled() {
		if p.identity == "" {
			return fmt.Errorf("bug: identity is required for jvm support")
		}
		var err error
		p.jvmRegistry, err = jvmregistry.New(
			xlog.Wrap(p.log),
			r.WithPrefix("jvm"),
			p.bpf,
			unwmanager,
			jvmregistry.Options{
				SocketPath:        p.conf.JVM.SocketPathPrefix + p.identity + ".sock",
				ScannerBinaryPath: p.conf.JVM.ScannerBinaryPath,

				MapPrefix:               path.Join(p.conf.BPF.BPFFSRoot, p.conf.BPF.PinPrefix),
				DisambiguateFrameSource: p.conf.JVM.DisambiguateMethodKinds,

				InterpetedSymbolCacheSize: p.conf.JVM.InterpretedMethodSymbolizationCacheSize,
				InterpretedSymbolCacheTTL: p.conf.JVM.InterpretedMethodSymbolizationCacheTTL,

				EnableLineInfoParsing: p.conf.FeatureFlagsConfig.JVMLineInfoEnabled(),
			},
		)
		if err != nil {
			return fmt.Errorf("failed to create jvm registry: %w", err)
		}
		p.processListeners = append(p.processListeners, p.jvmRegistry)
		p.jitSymbolizers = append(p.jitSymbolizers, p.jvmRegistry)
	}

	var binaryListeners []binary.Listener
	if p.conf.FeatureFlagsConfig.JVMEnabled() {
		binaryListeners = append(binaryListeners, p.jvmRegistry)
	}

	bpfManager, err := binary.NewBPFBinaryManager(
		p.log.WithName("ProcessRegistry"),
		r.WithPrefix("ProcessRegistry"),
		p.bpf.State(),
		unwmanager,
		binary.WithAddListeners(binaryListeners...),
	)
	if err != nil {
		return fmt.Errorf("failed to create bpf binary manager: %w", err)
	}

	binaryAnalysisOptions := &preprocessig_proto.BinaryAnalysisOptions{
		PreferredUnwindInfoSource: preprocessig_proto.UnwindInfoSource_Ehframe,
	}
	if p.conf.FeatureFlagsConfig.SframeEnabled() {
		binaryAnalysisOptions.PreferredUnwindInfoSource = preprocessig_proto.UnwindInfoSource_Sframe
	}
	p.dsoStorage, err = dso.NewStorage(xlog.Wrap(p.log.WithName("ProcessRegistry")), r.WithPrefix("ProcessRegistry"), bpfManager, binaryAnalysisOptions)
	if err != nil {
		return fmt.Errorf("failed to create dso storage: %w", err)
	}

	// Setup process registry.
	p.procs, err = process.NewProcessRegistry(
		xlog.Wrap(p.log.WithName("ProcessRegistry")),
		r.WithPrefix("ProcessRegistry"),
		p.bpf.State(),
		p.mounts,
		p.dsoStorage,
		&process.UploaderArguments{
			Conf:    p.conf.UploadSchedulerConfig,
			Storage: p.storage,
		},
		p.processScanner,
		p.processListeners,
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

	if p.conf.PodsDeploySystemConfig != nil && p.conf.PodsDeploySystemConfig.DeploySystem != "" {
		cgroupPrefix := p.cgroups.CgroupPrefix()
		podsCgroupTracker, err := newPodsCgroupTracker(p.conf.PodsDeploySystemConfig, p.log, cgroupPrefix)
		if err != nil {
			return err
		}
		p.podsCgroupTracker = podsCgroupTracker
		p.log.Info("Successfully loaded pods cgroup tracker", log.String("deploy_system", p.conf.PodsDeploySystemConfig.DeploySystem))
	}

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

	p.sampleConsumerRegistry = newSampleConsumerRegistry()

	p.sampleProcessor = newSampleProcessor(
		p.log.WithName("sample_processor"),
		r,
		p.bpf,
		&p.conf.SampleConsumer,
		p.sampleCallback,
		p.sampleConsumerRegistry,
	)

	// Initialize targets
	err = p.initializeTargets()
	if err != nil {
		return fmt.Errorf("failed to initialize targets: %w", err)
	}

	// We are done.
	return nil
}

func (p *Profiler) registerMainSampleConsumer() error {
	// TODO: later add filters only for perf events specified in profiler config.
	allowedUprobes := make(map[uprobe.BinaryInfo]struct{})
	for _, uprobe := range p.initialUprobes {
		allowedUprobes[uprobe.Info().BinaryInfo] = struct{}{}
	}

	var mainPerfEvents []*PerfEvent
	for _, bundle := range p.events {
		mainPerfEvents = append(mainPerfEvents, bundle)
	}

	mainSampleConsumer := NewFilterSampleConsumerAdapter(
		newContinuousProfilingSampleConsumer(p),
		NewORSampleFilter(
			NewUprobeSampleFilter(p, allowedUprobes),
			NewPerfEventIDSampleFilter(mainPerfEvents...),
			NewOtherSampleTypesFilter(),
		),
	)
	err := p.sampleConsumerRegistry.Register(mainSampleConsumerName, mainSampleConsumer)
	if err != nil {
		return fmt.Errorf("failed to register main sample consumer: %w", err)
	}

	p.mainSampleConsumer = mainSampleConsumer
	return nil
}

func (p *Profiler) initializeTargets() error {
	if p.initialTargets.selfTargetLabels != nil {
		_, err := p.TraceSelf(p.initialTargets.selfTargetLabels)
		if err != nil {
			return fmt.Errorf("failed to initialize self tracing: %w", err)
		}
	}

	if len(p.initialTargets.cgroupTargets) > 0 {
		err := p.TraceCgroups(p.initialTargets.cgroupTargets)
		if err != nil {
			return fmt.Errorf("failed to initialize cgroup tracing: %w", err)
		}
	}

	for _, target := range p.initialTargets.processTargets {
		p.log.Info("Registering process", log.Int("pid", target.pid))
		_, err := p.TracePid(linux.CurrentNamespacePID(target.pid), WithProfileLabels(target.labels))
		if err != nil {
			return fmt.Errorf("failed to initialize pid %d tracing: %w", target.pid, err)
		}
	}

	for _, target := range p.initialTargets.threadTargets {
		p.log.Info("Registering thread", log.Int("tid", target.tid))
		_, err := p.TracePid(linux.CurrentNamespacePID(target.tid), WithProfileLabels(target.labels))
		if err != nil {
			return fmt.Errorf("failed to initialize tid %d tracing: %w", target.tid, err)
		}
	}

	return nil
}

func (p *Profiler) setupPerfEvents() error {
	for _, event := range p.conf.PerfEvents {
		event := event

		p.log.Debug("Trying to open perf event bundle", log.Any("config", event))

		typ := perfevent.GetTypeByNameOrAlias(event.Type)
		if typ == nil {
			return fmt.Errorf("failed to find event %q", event.Type)
		}
		eventType, ok := typ.(*perfevent.PerfEventType)
		if !ok {
			return fmt.Errorf("found event %q, but it is not a perf event", event.Type)
		}
		if p.events[eventType] != nil {
			return fmt.Errorf("duplicate perf event type %s", event.Type)
		}

		target := &perfevent.Target{
			WholeSystem: true,
		}

		options := &perfevent.Options{
			Type:                   eventType,
			Frequency:              event.Frequency,
			SampleRate:             event.SampleRate,
			TryToSampleBranchStack: perfevent.ShouldTryToEnableBranchSampling(),
		}

		bundle, err := p.perfEventManager.Open(target, options)
		if err != nil {
			return fmt.Errorf("failed to create perf event bundle: %w", err)
		}

		p.events[eventType] = bundle
		p.log.Debug("Successfully opened perf event bundle")
	}

	return nil
}

func (p *Profiler) installPerfEventBPF() error {
	for _, bundle := range p.events {
		err := bundle.Attach()
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Profiler) setupUprobes() {
	uprobeConfigs := p.conf.Uprobes
	if len(uprobeConfigs) == 0 {
		uprobeConfigs = p.conf.BPF.UprobesDeprecated
	}

	for _, conf := range uprobeConfigs {
		uprobe := p.uprobeRegistry.Create(conf)
		p.initialUprobes = append(p.initialUprobes, uprobe)
	}
}

func (p *Profiler) UprobeManager() UprobeManager {
	return p.uprobeRegistry
}

func (p *Profiler) PerfEventManager() *PerfEventManager {
	return p.perfEventManager
}

func (p *Profiler) SampleConsumerRegistry() SampleConsumerRegistry {
	return p.sampleConsumerRegistry
}

func (p *Profiler) WaitForSampleProcessing(ctx context.Context) error {
	return p.sampleProcessor.waitForSampleProcessing(ctx)
}

func (p *Profiler) PidNamespaceIndex() process.PidNamespaceIndex {
	return p.procs
}

func (p *Profiler) maybeInitializeAmdFam19hBRSPerfEvent() {
	if !(cpuid.CPU.VendorID == cpuid.AMD && cpuid.CPU.Family == 0x19) {
		// Family19h is Zen 3, Zen 3+ and Zen 4 (https://en.wikipedia.org/wiki/List_of_AMD_CPU_microarchitectures)
		//
		// Family1Ah is Zen 5, and Zen 5 supports sampling last branch records
		// with any kind of perf-event via LbrExtV2 out of the box, so we don't have to
		// create an additional one.
		//
		// Zen 4 also supports LbrExtV2, but it's is mutually exclusive with BRS,
		// so either we have already installed the branch sampling in perf_event_open above
		// and the subsequent perf_event_open syscall will fail, or LbrExtV2 wasn't available,
		// and we will try to enable BRS. In any case, we won't sample branches twice.
		return
	}

	p.log.Info("Trying to enable AMD BRS perf-event")

	target := &perfevent.Target{
		WholeSystem: true,
	}
	options := &perfevent.Options{
		Type: perfevent.AMDFam19hBRS,
		// Frequency is not supported here, so we have to provide sampling period.
		// This value seems reasonable to me, but could be easily changed.
		SampleRate:             ptr.T(uint64(1000009)),
		TryToSampleBranchStack: true,
	}

	bundle, err := p.perfEventManager.openImpl(target, options, amdBRSProgram)
	if err != nil {
		p.log.Warn("Failed to enable AMD BRS perf-event")
		// Failure here is okay, BRS support is best-effort.
		return
	}
	defer func() {
		if err != nil {
			_ = bundle.Close()
		}
	}()

	err = bundle.Attach()
	if err != nil {
		p.log.Warn("Failed to attach eBPF-program to the AMD BRS perf-event")
		return
	}

	p.events[options.Type] = bundle
}

func (p *Profiler) registerMetrics(r metrics.Registry) error {
	type Labels map[string]string

	mappings := r.CounterVec("mapping_resolving.count", []string{"status"})
	p.metrics.mappingsHit = mappings.With(Labels{"status": "hit"})
	p.metrics.mappingsMiss = mappings.With(Labels{"status": "miss"})

	tls := r.CounterVec("tls_name_resolving.count", []string{"status"})
	p.metrics.resolveTLSSuccess = tls.With(Labels{"status": "success"})
	p.metrics.resolveTLSErrors = tls.With(Labels{"status": "fail"})

	p.metrics.recordedTLSVarsFromSamples = r.Counter("tls.variables_recorded.count")
	p.metrics.recordedTLSBytes = r.Counter("tls.variables_recorded.bytes")

	p.metrics.unresolvedPerfEventsForSamples = r.Counter("perf_event.unresolved.count")

	p.metrics.droppedProfiles = r.WithTags(Labels{"kind": "dropped"}).Counter("profiles.count")

	p.metrics.pythonMetrics = languageCollectionMetrics{
		unsymbolizedFrameCount: r.Counter("python.frame.unsymbolized.count"),
		collectedFrameCount:    r.Counter("python.frame.collected.count"),
	}
	p.metrics.phpMetrics = languageCollectionMetrics{
		unsymbolizedFrameCount: r.Counter("php.frame.unsymbolized.count"),
		collectedFrameCount:    r.Counter("php.frame.collected.count"),
	}

	r.WithTags(Labels{"kind": "tracked"}).FuncIntGauge("cgroup.count", func() int64 {
		if p.cgroups == nil {
			return 0
		}
		return int64(p.cgroups.NumCgroupNames())
	})

	p.metrics.cgroupHits = r.Counter("cgroup.cache.hit.count")
	p.metrics.cgroupMisses = r.Counter("cgroup.cache.miss.count")

	p.metrics.sampleProcessingLatencySum = r.Counter("samples.processing.total_latency.milliseconds")
	p.metrics.sampleProcessingCount = r.Counter("samples.processing.count")

	r.FuncGauge("ebpf.memlocked.bytes", func() float64 {
		if p.bpf == nil {
			return 0.0
		}
		count, err := p.bpf.State().CountTotalMemLockedBytes()
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
	if p.conf.FeatureFlagsConfig.JVMEnabled() {
		conf.EnableJvm = true
	}

	conf.EnablePhp = p.conf.FeatureFlagsConfig.PhpEnabled()

	// Record current pidns.
	pidns, err := procfs.Self().GetNamespaces().GetPidInode()
	if err != nil {
		p.log.Error("Failed to resolve self pid namespace inode number", log.Error(err))
		conf.PidnsInode = 0
	} else {
		p.log.Debug("Resolved self pid namespace inode number", log.UInt64("inode", uint64(pidns)))
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
	err = p.bpf.State().UpdateConfig(conf)
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
	return fmt.Errorf("worker %q failed: %w", workerName, err)
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

	err = p.uprobeRegistry.attachAll()
	if err != nil {
		return fmt.Errorf("failed to attach uprobes: %w", err)
	}

	err = p.registerMainSampleConsumer()
	if err != nil {
		return fmt.Errorf("failed to register main sample consumer: %w", err)
	}

	p.wg, ctx = errgroup.WithContext(ctx)
	if p.enablePerfMaps {
		p.wg.Go(func() error {
			err := p.perfmap.Run(ctx)
			return p.handleWorkerError(ctx, err, "perf map manager")
		})
	}
	p.wg.Go(func() error {
		err := p.clockConverter.Run(ctx)
		return p.handleWorkerError(ctx, err, "clock converter")
	})
	p.wg.Go(func() error {
		err := p.sampleProcessor.Run(ctx)
		return p.handleWorkerError(ctx, err, "sample processor")
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
		err := p.procs.RunProcessScanner(ctx)
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
	if p.jvmRegistry != nil {
		p.wg.Go(func() error {
			err := p.jvmRegistry.Run(ctx)
			return p.handleWorkerError(ctx, err, "jvm registry")
		})
	}

	if p.conf.ProcessDiscovery.Concurrency <= 0 {
		return fmt.Errorf("invalid process discovery concurrency: %d", p.conf.ProcessDiscovery.Concurrency)
	}

	for i := 0; i < p.conf.ProcessDiscovery.Concurrency; i++ {
		p.wg.Go(func() error {
			err := p.runProcessDiscovery(ctx)
			return p.handleWorkerError(ctx, err, "process discovery")
		})

		p.wg.Go(func() error {
			err := p.procs.RunWorker(ctx, p.conf.ProcessDiscovery.AnalysisConfig)
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

	if p.mainSampleConsumer != nil {
		p.log.Info("Flushing main sample consumer")
		err := p.mainSampleConsumer.Flush(ctx)
		if err != nil {
			p.log.Error("Failed to flush main sample consumer", log.Error(err))
		}
	}
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
			p.eventListener.OnSampleStored(linux.CurrentNamespacePID(pid))
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

		_, err := r.Read(ctx, &sample)
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
		p.procs.DiscoverProcess(ctx, linux.CurrentNamespacePID(sample.Pid))
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

	// TODO: For now we do not provide a way to enable features through AddCgroup
	cg, err := p.newTrackedCgroup(conf, NewSimpleSampleConsumer(p, DefaultSampleConsumerFeatures(), conf.Labels, p.uprobeRegistry.Resolver), p.bpf.State(), p.log)
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
	p.wholeSystem = NewSimpleSampleConsumer(p, DefaultSampleConsumerFeatures(), labels, p.uprobeRegistry.Resolver)
	return p.bpf.State().PatchConfig(func(conf *unwinder.ProfilerConfig) error {
		conf.TraceWholeSystem = true
		return nil
	})
}

func (p *Profiler) TraceSelf(labels map[string]string) (Closer, error) {
	return p.TracePid(linux.CurrentNamespacePID(os.Getpid()), WithProfileLabels(labels))
}

type Closer interface {
	Close(ctx context.Context) error
}

type pidTracingCloser struct {
	profiler *Profiler
	pid      linux.CurrentNamespacePID
}

func (p *pidTracingCloser) Close(ctx context.Context) error {
	trackedProcess := p.profiler.removeTracedPid(p.pid)
	if trackedProcess == nil {
		return nil
	}

	return trackedProcess.close()
}

type traceOptions struct {
	profileLabels map[string]string
	features      SampleConsumerFeatures
}

func defaultTraceOptions() *traceOptions {
	return &traceOptions{
		features:      DefaultSampleConsumerFeatures(),
		profileLabels: make(map[string]string),
	}
}

type TraceOption func(o *traceOptions)

func WithAbsoluteSampleTimeCollection() TraceOption {
	return func(o *traceOptions) {
		o.features.EnableSampleTimeCollection = true
	}
}

func WithProfileLabels(labels map[string]string) TraceOption {
	return func(o *traceOptions) {
		o.profileLabels = labels
	}
}

func (p *Profiler) TracePid(pid linux.CurrentNamespacePID, optAppliers ...TraceOption) (Closer, error) {
	opts := defaultTraceOptions()
	for _, optApplier := range optAppliers {
		optApplier(opts)
	}

	labels := p.enrichProfileLabels(opts.profileLabels)

	trackedProcess, err := p.newTrackedProcess(pid, NewSimpleSampleConsumer(p, opts.features, labels, p.uprobeRegistry.Resolver), p.bpf.State())
	if err != nil {
		return nil, err
	}

	p.pidsmu.Lock()
	p.pids[pid] = trackedProcess
	p.pidsmu.Unlock()

	p.log.Info("Registered process", logfield.CurrentNamespacePID(pid))
	return &pidTracingCloser{
		profiler: p,
		pid:      pid,
	}, nil
}

func (p *Profiler) removeTracedPid(pid linux.CurrentNamespacePID) *trackedProcess {
	p.pidsmu.Lock()
	defer p.pidsmu.Unlock()

	trackedProcess, ok := p.pids[pid]
	if !ok {
		return nil
	}

	delete(p.pids, pid)
	return trackedProcess
}

func (p *Profiler) DeleteCgroup(name string) error {
	return p.cgroups.Delete(name)
}

func (p *Profiler) TraceCgroups(configs []*CgroupConfig) error {
	trackedCgroups := make([]*cgroups.TrackedCgroup, 0, len(configs))
	for _, conf := range configs {
		conf.Labels = p.enrichProfileLabels(conf.Labels)

		// TODO: For now we do not provide a way to enable features through TraceCgroups
		profiledCgroup, err := p.newTrackedCgroup(conf, NewSimpleSampleConsumer(p, DefaultSampleConsumerFeatures(), conf.Labels, p.uprobeRegistry.Resolver), p.bpf.State(), p.log)
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

	if err := p.cgroups.TrackCgroups(trackedCgroups); err != nil {
		return fmt.Errorf("tracking cgroups: %w", err)
	}
	return nil
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

	err = p.uprobeRegistry.detachAll()
	if err != nil {
		return fmt.Errorf("failed to detach uprobes: %w", err)
	}

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

	err = p.uprobeRegistry.attachAll()
	if err != nil {
		return fmt.Errorf("failed to attach uprobes: %w", err)
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
	err := p.uprobeRegistry.detachAll()
	if err != nil {
		return fmt.Errorf("failed to detach uprobes: %w", err)
	}

	for _, uprobe := range p.initialUprobes {
		err := uprobe.Close()
		if err != nil {
			return fmt.Errorf("failed to close uprobe: %w", err)
		}
	}

	p.sampleConsumerRegistry.Unregister(mainSampleConsumerName)

	return p.bpf.Close()
}

func (p *Profiler) Stop(ctx context.Context) error {
	// Shutdown sequence:
	// 1. Disable any active perf events.
	// 2. Detach any active uprobes
	// 3. Disable any active eBPF program.
	// 4. Drain sample queue
	// 5. Drain profile queue
	// 6. Abort any running background job (e.g. process, mountinfo and cgroup pollers)

	err := p.disablePerfEvents()
	if err != nil {
		p.log.Error("Failed to disable perf events", log.Error(err))
	}

	err = p.uprobeRegistry.detachAll()
	if err != nil {
		return fmt.Errorf("failed to detach uprobes: %w", err)
	}

	err = p.bpf.UnlinkPrograms()
	if err != nil {
		p.log.Error("Failed to disable eBPF programs", log.Error(err))
	}

	p.log.Info("Stopping sample reader")
	err = p.sampleProcessor.GracefulStop(ctx)
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

	p.sampleConsumerRegistry.Unregister(mainSampleConsumerName)

	p.log.Info("Waiting for background workers to stop")
	return p.Wait()
}

func (p *Profiler) Wait() error {
	return p.wg.Wait()
}

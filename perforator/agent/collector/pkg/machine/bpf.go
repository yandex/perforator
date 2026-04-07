package machine

import (
	"bytes"
	"context"
	"encoding"
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/containerd/containerd/pkg/cap"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/graceful"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

// We work with large programs.
const (
	verifierLogSizeStart  = 60 * 1024 * 1024
	perfReaderTimeout     = 2 * time.Second
	ebpfMapSizeLimitBytes = 1<<32 - 1<<20
)

type Config struct {
	Debug bool `yaml:"debug"`

	EnablePageTableScaling *bool `yaml:"enable_page_table_scaling"`
	PageTableScaleFactorGB *int  `yaml:"page_table_scale_factor_gb"`

	// Override of page table size, primarily for less memory consumption by tests.
	// Default is ~1GB.
	PageTableSizeKB *uint64 `yaml:"page_table_size_kb"`

	// Collect LBR stacks.
	TraceLBR *bool `yaml:"trace_lbr"`
	// Collect BRS on AMD. Requires `trace_lbr`.
	TraceLBROnAMD *bool `yaml:"trace_lbr_on_amd"`
	// Trace potentially fatal signals.
	TraceSignals *bool `yaml:"trace_signals"`
	// Trace wall time.
	TraceWallTime *bool `yaml:"trace_walltime"`
	// Collect python stacks
	TracePython *bool `yaml:"trace_python"`
	// Configuration for uprobes tracing (deprecated, this field has moved to profiler config)
	UprobesDeprecated []uprobe.Config `yaml:"uprobes,omitempty"`
	// bpffs mount point.
	BPFFSRoot string `yaml:"bpffs_root"`
	// Path prefix to use when pinning BPF objects.
	PinPrefix string `yaml:"pin_prefix"`
	// Explicitly choose desired kernel version support.
	// Possible values: "none" (assume fresh kernel, currently 5.10+), "5.4"
	KernelCompatibility string `yaml:"kernel_compatibility"`
}

type Options struct {
	EnableJVM bool
	EnablePHP bool
}

type BPF struct {
	conf *Config
	opts Options
	log  log.Logger

	maxPageCount metrics.IntGauge
	maxPartCount metrics.IntGauge
	maxSizeBytes metrics.IntGauge
	metrics      metrics.Registry

	collspec        *ebpf.CollectionSpec
	mapreplacements map[string]*ebpf.Map
	state           programstate.State
	// some maps that we need in runtime
	samplesMap   *ebpf.Map
	processesMap *ebpf.Map

	mapsToUnpin []*ebpf.Map

	progsmu          sync.RWMutex
	progdebug        bool
	progkernelcompat unwinder.KernelCompatibilityLevel
	progs            *unwinder.Progs

	links *bpfLinks
}

func NewBPF(conf *Config, log log.Logger, metrics metrics.Registry, opts Options) (*BPF, error) {
	metrics = metrics.WithPrefix("bpf")

	b := &BPF{
		conf: conf,
		opts: opts,
		log:  log.WithName("BPF"),

		maxSizeBytes: metrics.IntGauge("unwind_page_table.size.bytes"),
		maxPartCount: metrics.IntGauge("unwind_page_table.max_parts.count"),
		maxPageCount: metrics.IntGauge("unwind_page_table.max_pages.count"),

		metrics: metrics,

		progdebug: conf.Debug,
	}

	err := b.initialize()
	if err != nil {
		return nil, err
	}

	return b, nil
}

func (b *BPF) currentProgramRequirements() unwinder.ProgramRequirements {
	return unwinder.ProgramRequirements{
		Debug:               b.progdebug,
		PHP:                 b.opts.EnablePHP,
		KernelCompatibility: b.progkernelcompat,
	}
}

func convertSignedBytes(d []int8) string {
	buf := make([]byte, len(d))
	for i, v := range d {
		if v == 0 {
			buf = buf[:i]
			break
		}
		buf[i] = byte(v)
	}
	return string(buf)
}

func (b *BPF) selectKernelCompatibility() error {
	switch b.conf.KernelCompatibility {
	case "5.4":
		b.progkernelcompat = unwinder.KernelCompatibilityLevel5_4
		return nil
	case "none":
		b.progkernelcompat = unwinder.KernelCompatibilityLevelNone
		return nil
	case "":
		b.log.Debug("Checking kernel version")
		var uname syscall.Utsname
		err := syscall.Uname(&uname)
		if err != nil {
			return fmt.Errorf("uname failed: %w", err)
		}
		release := convertSignedBytes(uname.Release[:])
		version := strings.Split(release, "-")[0]
		parts := strings.Split(version, ".")
		if len(parts) < 2 {
			return fmt.Errorf("unexpected kernel release %q", release)
		}
		major, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse kernel major version: %w", err)
		}
		minor, err := strconv.ParseInt(parts[1], 10, 32)
		if err != nil {
			return fmt.Errorf("failed to parse kernel minor version: %w", err)
		}
		b.log.Debug("Found kernel version", log.Int64("major", major), log.Int64("minor", minor), log.String("raw", release))
		if major < 5 || (major == 5 && minor < 10) {
			b.progkernelcompat = unwinder.KernelCompatibilityLevel5_4
		} else {
			b.progkernelcompat = unwinder.KernelCompatibilityLevelNone
		}
		return nil
	default:
		return fmt.Errorf("unsupported kernel compatibility level %q", b.conf.KernelCompatibility)
	}
}

func (b *BPF) initialize() (err error) {
	if _, ok := os.LookupEnv("PERFORATOR_IGNORE_CAPABILITIES"); !ok {
		caps, err := cap.Current()
		if err != nil {
			return fmt.Errorf("failed to list current process capabilities: %w", err)
		}
		if !slices.Contains(caps, "CAP_SYS_ADMIN") {
			return fmt.Errorf("profiler process does not have CAP_SYS_ADMIN capability, please try again as root")
		}
	} else {
		b.log.Warn("Skipped capabilities check because environment variable PERFORATOR_IGNORE_CAPABILITIES is set")
	}

	// eBPF maps must be allocated in locked memory. Remove mlock limit.
	b.log.Debug("Trying to remove mlock limit")
	err = rlimit.RemoveMemlock()
	if err != nil {
		return fmt.Errorf("failed to remove memlock limit: %w", err)
	}
	b.log.Debug("Successfully removed mlock limit")
	err = b.selectKernelCompatibility()
	if err != nil {
		return fmt.Errorf("failed to select kernel compatiblity level: %w", err)
	}
	spec, err := b.loadCollectionSpec(b.currentProgramRequirements())
	if err != nil {
		return fmt.Errorf("failed to load maps and programs: %w", err)
	}
	b.collspec = spec

	err = b.configureAndSetupMaps()
	if err != nil {
		return fmt.Errorf("failed to setup maps: %w", err)
	}

	err = b.setupProgramsUnsafe()
	if err != nil {
		return fmt.Errorf("failed to setup programs: %w", err)
	}

	b.links = newBPFLinks(b.log)
	err = b.links.setup(b.conf, b.progs)
	if err != nil {
		return fmt.Errorf("failed to setup links: %w", err)
	}

	b.log.Info("Successfully initialized eBPF program")

	return nil
}

func (b *BPF) calculatePageTablePageCount() (int, error) {
	npages := int(unwinder.UnwindPageTableNumPagesTotal)

	if pageTableSizeKB := b.conf.PageTableSizeKB; pageTableSizeKB != nil && *pageTableSizeKB > 0 {
		npages = int(*pageTableSizeKB * 1024 / uint64(unwinder.UnwindPageTablePageSize))
		return npages, nil
	}

	if enableScaling := b.conf.EnablePageTableScaling; enableScaling != nil && !*enableScaling {
		return npages, nil
	}

	factor := b.conf.PageTableScaleFactorGB
	if factor == nil {
		return npages, nil
	}

	meminfo, err := procfs.GetMemInfo()
	if err != nil {
		return 0, err
	}

	scale := 1 + meminfo.MemTotal/uint64(*factor<<30)
	npages *= int(scale)
	pageSize := int(unwinder.UnwindPageTablePageSize)

	if npages*pageSize > ebpfMapSizeLimitBytes {
		npages = ebpfMapSizeLimitBytes / pageSize
	}

	return npages, nil
}

func (b *BPF) calculatePageTablePartCount() (int, error) {
	npages, err := b.calculatePageTablePageCount()
	if err != nil {
		return 0, err
	}

	nparts := (npages-1)/int(unwinder.UnwindPageTableNumPagesPerPart) + 1
	return nparts, nil
}

func (b *BPF) prepareUnwindTableSpec(unwindTableMap *ebpf.MapSpec) (programstate.UnwindTableOpts, error) {
	var opts programstate.UnwindTableOpts
	var err error
	opts.PartCount, err = b.calculatePageTablePartCount()
	if err != nil {
		return opts, fmt.Errorf("failed to calculate page table size: %w", err)
	}
	maxPageCount := uint64(unwinder.UnwindPageTableNumPagesPerPart) * uint64(opts.PartCount)

	bytes := uint64(unwinder.UnwindPageTablePageSize) * maxPageCount
	b.log.Debug("Calculated unwind page table size",
		log.Int("parts", opts.PartCount),
		log.UInt64("pages", maxPageCount),
		log.UInt64("bytes", bytes),
	)
	b.maxSizeBytes.Set(int64(bytes))
	b.maxPartCount.Set(int64(opts.PartCount))
	b.maxPageCount.Set(int64(maxPageCount))
	unwindTableMap.MaxEntries = uint32(opts.PartCount)
	if unwindTableMap.InnerMap == nil {
		return opts, fmt.Errorf("unwind_table map does not have inner map spec: %+v", *unwindTableMap)
	}
	opts.PartSpec = unwindTableMap.InnerMap

	opts.Logger = b.log
	opts.Metrics = b.metrics
	return opts, nil
}

func (b *BPF) loadCollectionSpec(reqs unwinder.ProgramRequirements) (*ebpf.CollectionSpec, error) {
	b.log.Debug("Loading eBPF program", log.Bool("debug", reqs.Debug))
	// Load & prepare main program ELF.
	program, err := unwinder.LoadProg(reqs)
	if err != nil {
		return nil, fmt.Errorf("failed to load eBPF program: %w", err)
	}
	b.log.Debug("Parsing eBPF program ELF")
	elf := bytes.NewReader(program)

	spec, err := ebpf.LoadCollectionSpecFromReader(elf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse eBPF program: %w", err)
	}

	b.log.Debug("Successfully parsed eBPF program ELF",
		log.Int64("num_bytes", elf.Size()),
	)

	return spec, nil
}

func (b *BPF) configureAndSetupMaps() (err error) {
	b.log.Debug("Applying runtime customizations")
	unwindTableMapSpec, ok := b.collspec.Maps["unwind_table"]
	if !ok {
		return fmt.Errorf("missing unwind_table map")
	}
	utSpec, err := b.prepareUnwindTableSpec(unwindTableMapSpec)
	if err != nil {
		return fmt.Errorf("failed to prepare unwind table spec: %w", err)
	}

	b.log.Debug("Loading eBPF maps into the kernel")

	maps := &unwinder.Maps{}
	err = b.collspec.LoadAndAssign(maps, nil)
	if err != nil {
		return fmt.Errorf("failed to assign maps: %w", err)
	}

	err = b.pinIfNeeded(maps)
	if err != nil {
		return fmt.Errorf("failed to pin maps: %w", err)
	}

	// Prepare map replacements to be used by programs later.
	b.mapreplacements = make(map[string]*ebpf.Map)
	_ = maps.ForEachNamedMap(func(name string, m *ebpf.Map) error {
		b.mapreplacements[name] = m
		return nil
	})

	b.state = *programstate.New(maps, &utSpec)
	b.samplesMap = maps.Samples
	b.processesMap = maps.Processes
	if b.samplesMap == nil {
		return fmt.Errorf("missing samples map")
	}
	if b.processesMap == nil {
		return fmt.Errorf("missing processes map")
	}

	return nil
}

// setupProgramsUnsafe requires b.progsmu to be locked.
// Close any existing programs and load the new programs, probably with different build flags.
// This routine can be used for online program debugging without restarts.
func (b *BPF) setupProgramsUnsafe() (err error) {
	if b.progs != nil {
		err = b.progs.Close()
		if err != nil {
			return err
		}
		b.progs = nil
	}

	err = b.links.close()
	if err != nil {
		return fmt.Errorf("failed to close links: %w", err)
	}

	// The main interaction with the kernel happens here.
	b.log.Debug("Loading eBPF programs into the kernel")
	b.progs = &unwinder.Progs{}
	if err := b.collspec.LoadAndAssign(b.progs, &ebpf.CollectionOptions{
		Programs: ebpf.ProgramOptions{
			LogSizeStart: verifierLogSizeStart,
		},
		MapReplacements: b.mapreplacements,
	}); err != nil {
		var verr *ebpf.VerifierError
		if errors.As(err, &verr) {
			for idx, line := range verr.Log {
				b.log.Error("Kernel verifier rejected the program",
					log.Int("line", idx),
					log.String("log", line),
				)
			}
		} else {
			b.log.Error("Failed to load eBPF program", log.Error(err))
		}
		return err
	}

	return nil
}

func (b *BPF) UnlinkPrograms() error {
	return b.links.close()
}

func (b *BPF) Close() error {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()
	var errs []error
	errs = append(errs, b.progs.Close(), b.links.close())
	// let's unpin maps before closing them.
	for _, m := range b.mapsToUnpin {
		e := m.Unpin()
		if e != nil {
			// We don't preserve map name here, but it should be available in
			// the underlying FS error anyway.
			errs = append(errs, fmt.Errorf("failed to unpin map: %w", e))
		}
	}
	errs = append(errs, b.state.Close())
	return errors.Join(errs...)
}

////////////////////////////////////////////////////////////////////////////////

func (b *BPF) ProfilerProgramFD() int {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()
	return b.progs.PerforatorPerfEvent.FD()
}

func (b *BPF) AmdBRSProgramFD() int {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()
	return b.progs.PerforatorAmdFam19hBrsEvent.FD()
}

func (b *BPF) ReloadProgram(debug bool) error {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()

	if b.progdebug == debug {
		return nil
	}
	b.progdebug = debug

	err := b.setupProgramsUnsafe()
	if err != nil {
		return fmt.Errorf("failed to reload program: %w", err)
	}

	err = b.links.close()
	if err != nil {
		return fmt.Errorf("failed to close links: %w", err)
	}

	return b.links.setup(b.conf, b.progs)
}

////////////////////////////////////////////////////////////////////////////////

func (b *BPF) GenericUprobeProgram() *ebpf.Program {
	b.progsmu.RLock()
	defer b.progsmu.RUnlock()

	return b.progs.PerforatorUprobe
}

////////////////////////////////////////////////////////////////////////////////

func (b *BPF) State() *programstate.State {
	return &b.state
}

////////////////////////////////////////////////////////////////////////////////

func (b *BPF) RunMetricsPoller(ctx context.Context, stop graceful.ShutdownSource) error {
	defer stop.Finish()

	counters := make([]metrics.Counter, unwinder.MetricCount)
	for m := range unwinder.MetricCount {
		name := metricName(m.CString())
		counters[int(m)] = b.metrics.WithPrefix("prog").Counter(name)
	}

	ncpu := ebpf.MustPossibleCPU()
	prevmetrics := make([][unwinder.MetricCount]uint64, int(ncpu))
	nextmetrics := make([]uint64, int(ncpu))

	ticker := time.NewTicker(time.Second)
	for !stop.IsDone() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-stop.Done():
		case <-ticker.C:
		}

		var deltas [unwinder.MetricCount]int64
		for metric := range int(unwinder.MetricCount) {
			key := unwinder.Metric(metric)
			err := b.state.GetMetric(key, nextmetrics)
			if err != nil {
				b.log.Warn("Failed to load metric value",
					log.Error(err),
					log.Any("metric", metric),
				)
				continue
			}

			for cpu := range ncpu {
				next := nextmetrics[cpu]
				prev := prevmetrics[cpu][metric]
				prevmetrics[cpu][metric] = next
				delta := int64(next) - int64(prev)
				if delta < 0 {
					delta = 0
				}
				deltas[metric] += delta
			}
		}
		for metric := range int(unwinder.MetricCount) {
			counters[metric].Add(deltas[metric])
		}
	}

	return nil
}

func metricName(cname string) string {
	parts := strings.Split(cname, "_")
	if len(parts) < 1 {
		return ""
	}

	if parts[0] == "METRIC" {
		parts = parts[1:]
	}

	for i := range parts {
		parts[i] = strings.ToLower(parts[i])
	}

	return strings.Join(parts, ".")
}

////////////////////////////////////////////////////////////////////////////////

type RawSampleCallback = func(sample []byte)

type PerfReaderOptions struct {
	PerCPUBufferSize int
	Watermark        int
	SampleCallback   RawSampleCallback
}

type PerfReader struct {
	log            log.Logger
	sampleCallback RawSampleCallback
	reader         *perf.Reader
	record         perf.Record

	metrics struct {
		samplesCollected metrics.Counter
		samplesMalformed metrics.Counter
		samplesLost      metrics.Counter
	}
}

func (b *BPF) MakeSampleReader(opts *PerfReaderOptions) (*PerfReader, error) {
	return b.makePerfBufReader(b.samplesMap, "Samples", opts)
}

func (b *BPF) MakeProcessReader(opts *PerfReaderOptions) (*PerfReader, error) {
	return b.makePerfBufReader(b.processesMap, "Processes", opts)
}

func (b *BPF) makePerfBufReader(m *ebpf.Map, name string, opts *PerfReaderOptions) (*PerfReader, error) {
	r, err := perf.NewReaderWithOptions(
		m,
		opts.PerCPUBufferSize,
		perf.ReaderOptions{
			Watermark: opts.Watermark,
		},
	)

	if err != nil {
		return nil, err
	}

	br := &PerfReader{
		reader:         r,
		log:            b.log.WithName(fmt.Sprintf("PerfBufReader.%s", name)),
		sampleCallback: opts.SampleCallback,
	}

	br.instrument(b.metrics.WithPrefix("perfbuf").WithPrefix(name))

	return br, nil
}

func (r *PerfReader) instrument(m metrics.Registry) {
	type Labels map[string]string

	samples := m.CounterVec("samples.count", []string{"status"})
	r.metrics.samplesCollected = samples.With(Labels{"status": "collected"})
	r.metrics.samplesMalformed = samples.With(Labels{"status": "malformed"})
	r.metrics.samplesLost = samples.With(Labels{"status": "lost"})
}

type PerfReadMeta struct {
	CPU int
}

func (r *PerfReader) Read(ctx context.Context, sample encoding.BinaryUnmarshaler) (PerfReadMeta, error) {
	for {
		r.reader.SetDeadline(time.Now().Add(perfReaderTimeout))

		err := r.reader.ReadInto(&r.record)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				r.log.Error("Failed to read sample", log.Error(err))
			}
			return PerfReadMeta{}, err
		}

		if r.record.LostSamples != 0 {
			r.metrics.samplesLost.Add(int64(r.record.LostSamples))
			r.log.Error("Lost samples", log.UInt64("count", r.record.LostSamples))
			continue
		}
		if r.sampleCallback != nil {
			r.sampleCallback(r.record.RawSample)
		}
		err = sample.UnmarshalBinary(r.record.RawSample)
		if err != nil {
			r.metrics.samplesMalformed.Inc()
			r.log.Error("Failed to decode sample", log.Error(err))
			return PerfReadMeta{CPU: r.record.CPU}, err
		}

		r.metrics.samplesCollected.Inc()
		return PerfReadMeta{CPU: r.record.CPU}, nil
	}
}

func (r *PerfReader) Close() error {
	return r.reader.Close()
}

////////////////////////////////////////////////////////////////////////////////

package machine

import (
	"bufio"
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
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/perf"
	"github.com/cilium/ebpf/rlimit"
	"github.com/containerd/containerd/pkg/cap"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/graceful"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/kallsyms"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

// We work with large programs.
const (
	verifierLogSizeStart  = 6 * 1024 * 1024
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
	// Trace potentially fatal signals.
	TraceSignals *bool `yaml:"trace_signals"`
	// Trace wall time.
	TraceWallTime *bool `yaml:"trace_walltime"`
	// Collect python stacks
	TracePython *bool `yaml:"trace_python"`
}

type BPF struct {
	conf *Config
	log  log.Logger

	currentPageCount metrics.IntGauge
	currentPartCount metrics.IntGauge
	maxPageCount     metrics.IntGauge
	maxPartCount     metrics.IntGauge
	maxSizeBytes     metrics.IntGauge
	metrics          metrics.Registry

	maps            *unwinder.Maps
	mapreplacements map[string]*ebpf.Map

	progsmu   sync.Mutex
	progdebug bool
	progs     *unwinder.Progs

	unwindTablePartCount int
	unwindTablePartSpec  *ebpf.MapSpec
	partsmu              sync.RWMutex
	unwindTableParts     map[uint32]*ebpf.Map

	links programLinks
}

type programLinks struct {
	KprobeFinishTaskSwitch  link.Link
	TracepointSignalDeliver link.Link
}

func (p *programLinks) Close() error {
	errs := make([]error, 0)

	if p.KprobeFinishTaskSwitch != nil {
		errs = append(errs, p.KprobeFinishTaskSwitch.Close())
		p.KprobeFinishTaskSwitch = nil
	}
	if p.TracepointSignalDeliver != nil {
		errs = append(errs, p.TracepointSignalDeliver.Close())
		p.TracepointSignalDeliver = nil
	}

	return errors.Join(errs...)
}

func NewBPF(conf *Config, log log.Logger, metrics metrics.Registry) (*BPF, error) {
	metrics = metrics.WithPrefix("bpf")
	b := &BPF{
		conf: conf,
		log:  log.WithName("BPF"),

		currentPageCount: metrics.IntGauge("unwind_page_table.current_pages.count"),
		currentPartCount: metrics.IntGauge("unwind_page_table.current_parts.count"),
		maxSizeBytes:     metrics.IntGauge("unwind_page_table.size.bytes"),
		maxPartCount:     metrics.IntGauge("unwind_page_table.max_parts.count"),
		maxPageCount:     metrics.IntGauge("unwind_page_table.max_pages.count"),

		metrics: metrics,

		unwindTableParts: make(map[uint32]*ebpf.Map),
	}

	err := b.initialize()
	if err != nil {
		return nil, err
	}

	return b, nil
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

	err = b.setupMaps(b.conf.Debug)
	if err != nil {
		return err
	}

	err = b.setupProgramsUnsafe(b.conf.Debug)
	if err != nil {
		return err
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

func (b *BPF) prepareUnwindTableSpec(unwindTableMap *ebpf.MapSpec) error {
	var err error
	b.unwindTablePartCount, err = b.calculatePageTablePartCount()
	if err != nil {
		return fmt.Errorf("failed to calculate page table size: %w", err)
	}
	maxPageCount := uint64(unwinder.UnwindPageTableNumPagesPerPart) * uint64(b.unwindTablePartCount)

	bytes := uint64(unwinder.UnwindPageTablePageSize) * maxPageCount
	b.log.Debug("Calculated unwind page table size",
		log.Int("parts", b.unwindTablePartCount),
		log.UInt64("pages", maxPageCount),
		log.UInt64("bytes", bytes),
	)
	b.maxSizeBytes.Set(int64(bytes))
	b.maxPartCount.Set(int64(b.unwindTablePartCount))
	b.maxPageCount.Set(int64(maxPageCount))
	b.currentPageCount.Set(0)
	b.currentPartCount.Set(0)
	unwindTableMap.MaxEntries = uint32(b.unwindTablePartCount)
	if unwindTableMap.InnerMap == nil {
		return fmt.Errorf("unwind_table map does not have inner map spec: %+v", *unwindTableMap)
	}
	b.unwindTablePartSpec = unwindTableMap.InnerMap
	return nil
}

func (b *BPF) loadCollectionSpec(debug bool) (*ebpf.CollectionSpec, error) {
	// Load & prepare main program ELF.
	b.log.Debug("Parsing eBPF program ELF", log.Bool("debug", debug))
	elf := bytes.NewReader(unwinder.LoadProg(debug))

	spec, err := ebpf.LoadCollectionSpecFromReader(elf)
	if err != nil {
		return nil, fmt.Errorf("failed to load eBPF program: %w", err)
	}

	b.log.Debug("Successfully parsed eBPF program ELF",
		log.Int64("num_bytes", elf.Size()),
	)

	unwindTableMap, ok := spec.Maps["unwind_table"]
	if !ok {
		return nil, fmt.Errorf("unwind_table map not found")
	}

	err = b.prepareUnwindTableSpec(unwindTableMap)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare unwind table spec: %w", err)
	}

	return spec, nil
}

func (b *BPF) setupMaps(debug bool) (err error) {
	spec, err := b.loadCollectionSpec(debug)
	if err != nil {
		return err
	}

	b.log.Debug("Loading eBPF maps into the kernel")

	b.maps = &unwinder.Maps{}
	err = spec.LoadAndAssign(b.maps, nil)
	if err != nil {
		return err
	}

	// Prepare map replacements to be used by programs later.
	b.mapreplacements = make(map[string]*ebpf.Map)
	_ = b.maps.ForEachNamedMap(func(name string, m *ebpf.Map) error {
		b.mapreplacements[name] = m
		return nil
	})

	return nil
}

// setupProgramsUnsafe requires b.progsmu to be locked.
// Close any existing programs and load the new programs, probably in the debug mode.
// This routine can be used for online program debugging without restarts.
func (b *BPF) setupProgramsUnsafe(debug bool) (err error) {
	if b.progs != nil {
		err = b.progs.Close()
		if err != nil {
			return err
		}
		b.progs = nil
	}

	err = b.links.Close()
	if err != nil {
		return fmt.Errorf("failed to close links: %w", err)
	}

	spec, err := b.loadCollectionSpec(debug)
	if err != nil {
		return err
	}

	// The main interaction with the kernel happens here.
	b.log.Debug("Loading eBPF programs into the kernel")
	b.progs = &unwinder.Progs{}
	if err := spec.LoadAndAssign(b.progs, &ebpf.CollectionOptions{
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

	err = b.setupProgramLinks()
	if err != nil {
		return fmt.Errorf("failed to setup links: %w", err)
	}

	return nil
}

func (b *BPF) attachDynamicKprobe(name string, prog *ebpf.Program, opts *link.KprobeOptions) (link.Link, error) {
	resolver, err := kallsyms.DefaultKallsymsResolver()
	if err != nil {
		return nil, err
	}
	symbols, err := resolver.LookupSymbolRegex(name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup kprobe %s: %w", name, err)
	}

	errs := make([]error, 0)
	for _, symbol := range symbols {
		link, err := link.Kprobe(symbol, prog, opts)
		if err == nil {
			b.log.Debug("Found dynamic kprobe target", log.String("regex", name), log.String("symbol", symbol))
			return link, nil
		}
		errs = append(errs, err)
	}

	return nil, fmt.Errorf("failed to attach kprobe %s: %w", name, errors.Join(errs...))
}

func (b *BPF) setupProgramLinks() (err error) {
	if enabled := b.conf.TraceWallTime; enabled != nil && *enabled {
		// See https://github.com/iovisor/bcc/pull/3315
		b.links.KprobeFinishTaskSwitch, err = b.attachDynamicKprobe(`^finish_task_switch(\.isra\.\d+)?$`, b.progs.PerforatorFinishTaskSwitch, nil)
		if err != nil {
			return fmt.Errorf("failed to setup kprobe finish_task_switch link: %w", err)
		}
		defer func() {
			if err != nil {
				_ = b.links.KprobeFinishTaskSwitch.Close()
			}
		}()
	}

	if enabled := b.conf.TraceSignals; enabled != nil && *enabled {
		b.links.TracepointSignalDeliver, err = link.Tracepoint("signal", "signal_deliver", b.progs.PerforatorSignalDeliver, nil)
		if err != nil {
			return fmt.Errorf("failed to setup tracepoint signal_deliver link: %w", err)
		}
		defer func() {
			if err != nil {
				_ = b.links.TracepointSignalDeliver.Close()
			}
		}()
	}

	return nil
}

func (b *BPF) UnlinkPrograms() error {
	return b.links.Close()
}

func (b *BPF) Close() error {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()
	return errors.Join(b.maps.Close(), b.progs.Close(), b.links.Close())
}

func memLockedSize(fd int) (uint64, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d", fd))
	if err != nil {
		return 0, err
	}

	r := bufio.NewScanner(bytes.NewBuffer(b))
	for r.Scan() {
		key, value, _ := strings.Cut(r.Text(), ":\t")
		if key == "memlock" {
			count, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return 0, err
			}
			return count, nil
		}
	}

	return 0, r.Err()
}

func (b *BPF) CountMemLockedBytes() (uint64, error) {
	var locked uint64

	err := b.maps.ForEachMap(func(m *ebpf.Map) error {
		count, err := memLockedSize(m.FD())
		if err != nil {
			return err
		}
		locked += count
		return nil
	})

	if err != nil {
		return 0, err
	}

	return locked, nil
}

////////////////////////////////////////////////////////////////////////////////

func (b *BPF) ProfilerProgramFD() int {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()
	return b.progs.PerforatorPerfEvent.FD()
}

func (b *BPF) ReloadProgram(debug bool) error {
	b.progsmu.Lock()
	defer b.progsmu.Unlock()

	if b.progdebug == debug {
		return nil
	}
	b.progdebug = debug

	return b.setupProgramsUnsafe(debug)
}

func (b *BPF) UpdateConfig(conf *unwinder.ProfilerConfig) error {
	return b.maps.ProfilerConfig.Update(ptr.Uint32(0), conf, ebpf.UpdateAny)
}

func (b *BPF) PatchConfig(patcher func(conf *unwinder.ProfilerConfig) error) error {
	var key = ptr.Uint32(0)
	var conf unwinder.ProfilerConfig

	err := b.maps.ProfilerConfig.Lookup(key, &conf)
	if err != nil {
		return err
	}

	err = patcher(&conf)
	if err != nil {
		return err
	}

	return b.maps.ProfilerConfig.Update(key, &conf, ebpf.UpdateAny)
}

func (b *BPF) AddTracedCgroup(cgroup uint64) error {
	return b.maps.TracedCgroups.Update(cgroup, uint8(0), ebpf.UpdateAny)
}

func (b *BPF) RemoveTracedCgroup(cgroup uint64) error {
	return b.maps.TracedCgroups.Delete(cgroup)
}

func (b *BPF) AddTracedProcess(pid linux.ProcessID) error {
	return b.maps.TracedProcesses.Update(pid, uint8(0), ebpf.UpdateAny)
}

func (b *BPF) RemoveTracedProcess(pid linux.ProcessID) error {
	return b.maps.TracedProcesses.Delete(pid)
}

func (b *BPF) UnwindTablePartCount() int {
	return b.unwindTablePartCount
}

func (b *BPF) AddProcess(pid linux.ProcessID, info *unwinder.ProcessInfo) error {
	return b.maps.ProcessInfo.Put(&pid, info)
}

func (b *BPF) RemoveProcess(pid linux.ProcessID) error {
	return b.maps.ProcessInfo.Delete(&pid)
}

func (b *BPF) AddMappingLPMSegment(key *unwinder.ExecutableMappingTrieKey, value *unwinder.ExecutableMappingInfo) error {
	return b.maps.ExecutableMappingTrie.Update(key, value, ebpf.UpdateAny)
}

func (b *BPF) RemoveMappingLPMSegment(key *unwinder.ExecutableMappingTrieKey) error {
	return b.maps.ExecutableMappingTrie.Delete(key)
}

func (b *BPF) AddMapping(key *unwinder.ExecutableMappingKey, value *unwinder.ExecutableMapping) error {
	return b.maps.ExecutableMappings.Update(key, value, ebpf.UpdateAny)
}

func (b *BPF) RemoveMapping(key *unwinder.ExecutableMappingKey) error {
	return b.maps.ExecutableMappings.Delete(key)
}

func getPartID(pageID unwinder.PageId) uint32 {
	return uint32(pageID) / uint32(unwinder.UnwindPageTableNumPagesPerPart)
}

func getPartPageID(pageID unwinder.PageId) uint32 {
	return uint32(pageID) % uint32(unwinder.UnwindPageTableNumPagesPerPart)
}

func (b *BPF) getUnwindTablePartFast(partID uint32) *ebpf.Map {
	b.partsmu.RLock()
	defer b.partsmu.RUnlock()
	return b.unwindTableParts[partID]
}

func (b *BPF) getUnwindTablePart(partID uint32) (*ebpf.Map, error) {
	part := b.getUnwindTablePartFast(partID)
	if part != nil {
		return part, nil
	}
	b.partsmu.Lock()
	defer b.partsmu.Unlock()
	if b.unwindTableParts[partID] != nil {
		return b.unwindTableParts[partID], nil
	}

	partSpec := b.unwindTablePartSpec.Copy()
	var err error
	part, err = ebpf.NewMap(partSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to create new unwind table part: %w", err)
	}
	partFD := uint32(part.FD())
	b.log.Debug("Allocated new unwind table part", log.UInt32("part_id", partID), log.UInt32("part_fd", partFD))
	err = b.maps.UnwindTable.Put(&partID, &partFD)
	if err != nil {
		cleanupErr := part.Close()
		if cleanupErr != nil {
			b.log.Warn("Failed to cleanup unwind table part after failed insert, it will be leaked", log.Error(cleanupErr))
		}
		return nil, fmt.Errorf("failed to add part into unwind table: %w", err)
	}
	b.unwindTableParts[partID] = part
	partCount := len(b.unwindTableParts)
	b.currentPartCount.Set(int64(partCount))
	// TODO: track page count more accurately
	b.currentPageCount.Set(int64(partCount) * int64(unwinder.UnwindPageTableNumPagesPerPart))
	return part, nil
}

func (b *BPF) PutUnwindTablePage(id unwinder.PageId, page *unwinder.UnwindTablePage) error {
	part, err := b.getUnwindTablePart(getPartID(id))
	if err != nil {
		return fmt.Errorf("failed to get or insert part (id=%d): %w", getPartID(id), err)
	}
	partPageID := getPartPageID(id)
	err = part.Put(&partPageID, page)
	if err != nil {
		return fmt.Errorf("failed to add page into part (id=%d): %w", getPartID(id), err)
	}
	return nil
}

func (b *BPF) PutBinaryUnwindTable(id unwinder.BinaryId, root unwinder.PageId) error {
	return b.maps.UnwindRoots.Update(&id, &root, ebpf.UpdateNoExist)
}

func (b *BPF) DeleteBinaryUnwindTable(id unwinder.BinaryId) error {
	return b.maps.UnwindRoots.Delete(&id)
}

func (b *BPF) AddTLSConfig(id unwinder.BinaryId, tlsInfo *unwinder.TlsBinaryConfig) error {
	return b.maps.TlsStorage.Update(&id, tlsInfo, ebpf.UpdateAny)
}

func (b *BPF) DeleteTLSConfig(id unwinder.BinaryId) error {
	return b.maps.TlsStorage.Delete(&id)
}

func (b *BPF) AddPythonConfig(id unwinder.BinaryId, pythonInfo *unwinder.PythonConfig) error {
	return b.maps.PythonStorage.Update(id, pythonInfo, ebpf.UpdateAny)
}

// TODO: we can use batch lookups into bpf maps
func (b *BPF) SymbolizePython(key *unwinder.PythonSymbolKey) (res unwinder.PythonSymbol, exists bool) {
	err := b.maps.PythonSymbols.Lookup(key, &res)
	if err != nil {
		return
	}

	exists = true
	return
}

func (b *BPF) DeletePythonConfig(id unwinder.BinaryId) error {
	return b.maps.PythonStorage.Delete(&id)
}

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
			err := b.maps.Metrics.Lookup(&key, &nextmetrics)
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
	return b.makePerfBufReader(b.maps.Samples, "Samples", opts)
}

func (b *BPF) MakeProcessReader(opts *PerfReaderOptions) (*PerfReader, error) {
	return b.makePerfBufReader(b.maps.Processes, "Processes", opts)
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

func (r *PerfReader) Read(ctx context.Context, sample encoding.BinaryUnmarshaler) error {
	for {
		r.reader.SetDeadline(time.Now().Add(perfReaderTimeout))

		err := r.reader.ReadInto(&r.record)
		if err != nil {
			if !errors.Is(err, os.ErrDeadlineExceeded) {
				r.log.Error("Failed to read sample", log.Error(err))
			}
			return err
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
			return err
		}

		r.metrics.samplesCollected.Inc()
		return nil
	}
}

func (r *PerfReader) Close() error {
	return r.reader.Close()
}

////////////////////////////////////////////////////////////////////////////////

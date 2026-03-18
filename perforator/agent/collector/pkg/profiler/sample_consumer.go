package profiler

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/copy"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/btime"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/sampletype"
	"github.com/yandex/perforator/perforator/pkg/tls"
)

const (
	InnermostPidnsTidLabel = "innermost_pidns_tid"
	InnermostPidnsPidLabel = "innermost_pidns_pid"
	AbsoluteTimestampLabel = "absolute_timestamp"
)

type SampleConsumerFeatures struct {
	EnableSampleTimeCollection     bool
	EnableInnermostPidnsCollection bool
}

func DefaultSampleConsumerFeatures() SampleConsumerFeatures {
	return SampleConsumerFeatures{
		EnableSampleTimeCollection:     false,
		EnableInnermostPidnsCollection: false,
	}
}

// SampleConsumer is used for sequential consumption of multiple samples during its lifetime
type SampleConsumer interface {
	Consume(ctx context.Context, sample *unwinder.RecordSample)
	Flush(ctx context.Context) error
}

////////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	_ SampleConsumer = (*continuousProfilingSampleConsumer)(nil)
)

type continuousProfilingSampleConsumer struct {
	p *Profiler
}

func newContinuousProfilingSampleConsumer(p *Profiler) *continuousProfilingSampleConsumer {
	return &continuousProfilingSampleConsumer{
		p: p,
	}
}

func (c *continuousProfilingSampleConsumer) tryGetProcessSampleConsumer(sample *unwinder.RecordSample) SampleConsumer {
	c.p.pidsmu.RLock()
	defer c.p.pidsmu.RUnlock()
	if trackedProcess := c.p.pids[linux.CurrentNamespacePID(sample.Pid)]; trackedProcess != nil {
		return trackedProcess.sampleConsumer
	}
	if trackedProcess := c.p.pids[linux.CurrentNamespacePID(sample.Tid)]; trackedProcess != nil {
		return trackedProcess.sampleConsumer
	}
	return nil
}

func (c *continuousProfilingSampleConsumer) getTargetSampleConsumer(sample *unwinder.RecordSample) SampleConsumer {
	if c.p.wholeSystem != nil {
		return c.p.wholeSystem
	}

	if collector := c.tryGetProcessSampleConsumer(sample); collector != nil {
		return collector
	}

	if trackedEvent := c.p.cgroups.GetTrackedEvent(sample.ParentCgroup); trackedEvent != nil {
		return trackedEvent.(*trackedCgroup).sampleConsumer
	}

	return nil
}

func (c *continuousProfilingSampleConsumer) Consume(ctx context.Context, sample *unwinder.RecordSample) {
	targetConsumer := c.getTargetSampleConsumer(sample)
	if targetConsumer == nil {
		c.p.log.Debug(
			"No target sample consumer for profiling sample",
			log.UInt32("pid", sample.Pid),
			log.UInt32("tid", sample.Tid),
			log.UInt64("cgroupid", sample.ParentCgroup),
			log.String("name", c.p.cgroups.CgroupFullName(sample.ParentCgroup)),
		)
		return
	}

	targetConsumer.Consume(ctx, sample)
}

func (c *continuousProfilingSampleConsumer) Flush(ctx context.Context) error {
	errs := make([]error, 0)

	if c.p.wholeSystem != nil {
		c.p.log.Info("Flushing whole system profile")
		err := c.p.wholeSystem.Flush(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to flush whole system profile: %w", err))
		}
	}

	// Snapshot pids under short lock to avoid racing with add/remove.
	c.p.pidsmu.RLock()
	pidsSnapshot := make(map[linux.CurrentNamespacePID]*trackedProcess, len(c.p.pids))
	for k, v := range c.p.pids {
		pidsSnapshot[k] = v
	}
	c.p.pidsmu.RUnlock()

	for pid, process := range pidsSnapshot {
		c.p.log.Info("Flushing process profile", logfield.CurrentNamespacePID(pid))
		err := process.sampleConsumer.Flush(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to flush process %d profile: %w", pid, err))
		}
	}

	err := c.p.cgroups.ForEachCgroup(func(event cgroups.CgroupEventListener) error {
		cgroup := event.(*trackedCgroup)
		c.p.log.Info("Flushing cgroup profile", log.String("cgroup", cgroup.conf.Name))
		err := cgroup.sampleConsumer.Flush(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to flush cgroup %s profile: %w", cgroup.conf.Name, err))
		}
		return nil
	})
	if err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////

var (
	_ SampleConsumer = (*simpleSampleConsumer)(nil)
)

type guardedProfileBuilder struct {
	// This mutex synchronizes SampleBuilder.Finish() calls which add sample to
	// sample array of the underlying profile.Builder with
	// multiProfileBuilder.RestartProfiles() calls which restart the profiles.
	sync.Mutex
	*multiProfileBuilder
}

func (g *guardedProfileBuilder) RestartProfiles() labeledAgentProfiles {
	g.Lock()
	defer g.Unlock()
	return g.multiProfileBuilder.RestartProfiles()
}

type simpleSampleConsumer struct {
	p              *Profiler
	features       SampleConsumerFeatures
	profileBuilder *guardedProfileBuilder
}

func NewSimpleSampleConsumer(
	p *Profiler,
	features SampleConsumerFeatures,
	labels map[string]string,
) *simpleSampleConsumer {
	return &simpleSampleConsumer{
		features:       features,
		profileBuilder: &guardedProfileBuilder{multiProfileBuilder: newMultiProfileBuilder(labels)},
		p:              p,
	}
}

func (c *simpleSampleConsumer) Consume(ctx context.Context, sample *unwinder.RecordSample) {
	oneShotConsumer := newOneShotSampleConsumer(c.p, c.features, c.profileBuilder, sample)
	oneShotConsumer.consume(ctx)
}

func (c *simpleSampleConsumer) Flush(ctx context.Context) error {
	c.p.trySaveProfiles(ctx, c.profileBuilder.RestartProfiles())
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////

// oneShotSampleConsumer is a sample consumer which can only consume one sample during its lifetime
type oneShotSampleConsumer struct {
	p      *Profiler
	sample *unwinder.RecordSample

	profileBuilder *guardedProfileBuilder
	features       SampleConsumerFeatures
	envWhitelist   map[string]struct{}

	// sampleTime is the absolute wall-clock time of the sample, computed once
	// from boot time + CollectionTime and reused across the consumer.
	sampleTime time.Time

	stacklen  int
	env       []formattedEnvVariable
	tls       []formattedTLSVariable
	cgroupRel string

	pythonProcessor *sampleStackProcessor
	phpProcessor    *sampleStackProcessor
}

func computeSampleTime(collectionTime uint64) time.Time {
	bootTime, err := btime.GetBootTime()
	if err != nil {
		panic(fmt.Sprintf("failed to get system boot time: %v", err))
	}
	return bootTime.Add(time.Duration(collectionTime))
}

func newOneShotSampleConsumer(
	p *Profiler,
	features SampleConsumerFeatures,
	profileBuilder *guardedProfileBuilder,
	sample *unwinder.RecordSample,
) *oneShotSampleConsumer {
	return &oneShotSampleConsumer{
		p:               p,
		profileBuilder:  profileBuilder,
		features:        features,
		sample:          sample,
		sampleTime:      computeSampleTime(sample.CollectionTime),
		envWhitelist:    p.envWhitelist,
		pythonProcessor: newPythonSampleStackProcessor(p.pythonSymbolizer),
		phpProcessor:    newPHPSampleStackProcessor(p.phpSymbolizer),
	}
}

func (c *oneShotSampleConsumer) countMetrics(ctx context.Context) {
	// Count mappings cache hit/miss rate.
	var stacklen uint64
	for _, ip := range c.sample.Userstack {
		if ip == 0 {
			continue
		}
		stacklen += 1

		_, err := c.p.dsoStorage.ResolveAddress(ctx, linux.CurrentNamespacePID(c.sample.Pid), ip)
		if err == nil {
			c.p.metrics.mappingsHit.Inc()
		} else {
			c.p.metrics.mappingsMiss.Inc()
		}
	}
}

const (
	// (u64)-1, must match END_OF_CGROUP_LIST in cgroups.h
	endOfCgroupList = ^uint64(0)
)

func (c *oneShotSampleConsumer) collectWorkloadInto(builder *profile.SampleBuilder) {
	var parts []string

	var i int
	for ; i < len(c.sample.CgroupsHierarchy); i++ {
		cg := c.sample.CgroupsHierarchy[i]
		if cg == endOfCgroupList {
			break
		}
	}
	i--
	if c.sample.ParentCgroup == endOfCgroupList && i < len(c.sample.CgroupsHierarchy) {
		// Hierarchy is full (i.e. not truncated) path up to root in this case.
		// Therefore, the outermost cgroup is either "freezer" (for v1 hierarchy) or "cgroup" (for v2 hierarchy),
		// let's skip it.
		i--
	}

	var lastCgroupHit bool
	for ; i >= 0; i-- {
		cg := c.sample.CgroupsHierarchy[i]
		if cg == endOfCgroupList {
			continue
		}
		name := c.p.cgroups.CgroupBaseName(cg)
		if name == "" {
			lastCgroupHit = false
			c.p.log.Warn("Failed to get cgroup name", log.UInt64("cgroupid", cg))
			name = "<unknown cgroup>"
		} else {
			lastCgroupHit = true
		}
		parts = append(parts, name)
	}
	// TODO: currently this metric only measures accesses to innermost cgroups.
	// This is based on the assumption that x is only known if parent(x) is also known.
	// Instead, we should either:
	// - track each attempt individually, or
	// - track whether all accesses were hits.
	if lastCgroupHit {
		c.p.metrics.cgroupHits.Inc()
	} else {
		c.p.metrics.cgroupMisses.Inc()
	}

	c.cgroupRel = strings.Join(parts, "/")
	if c.p.podsCgroupTracker != nil {
		newParts, ok := c.p.podsCgroupTracker.ResolveWorkload(parts)
		if ok {
			parts = newParts
		}
	}

	for _, part := range parts {
		builder.AddStringLabel("workload", part)
	}
}

func parseUInt64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf[:8])
}

func parseString(buf []byte) (uint64, string) {
	len := binary.LittleEndian.Uint64(buf[:8])
	if len == 0 {
		return 0, ""
	}

	return len, string(buf[8 : 8+len])
}

type formattedTLSVariable struct {
	Key   string
	Value string
}

func (c *oneShotSampleConsumer) collectTLS(ctx context.Context) {
	c.tls = make([]formattedTLSVariable, 0)

	for _, variable := range c.sample.TlsValues.Values {
		if variable.Offset == 0 {
			break
		}

		var value string
		switch variable.Type {
		case unwinder.ThreadLocalUint64Type:
			value = fmt.Sprintf("%d", parseUInt64(variable.Value.UnionBuf[:]))

			// rough estimate
			c.p.metrics.recordedTLSBytes.Add(int64(len(value)))
		case unwinder.ThreadLocalStringType:
			var len uint64
			len, value = parseString(variable.Value.UnionBuf[:])

			// rough estimate
			c.p.metrics.recordedTLSBytes.Add(int64(len))
		default:
			continue
		}

		c.p.metrics.recordedTLSVarsFromSamples.Inc()

		varName, err := c.p.dsoStorage.ResolveTLSName(ctx, linux.CurrentNamespacePID(c.sample.Pid), variable.Offset)
		if err != nil {
			c.p.log.Warn(
				"Failed to resolve tls name",
				log.UInt32("pid", c.sample.Pid),
				log.UInt64("offset", variable.Offset),
				log.Error(err),
			)
			c.p.metrics.resolveTLSErrors.Inc()
			continue
		}

		c.p.metrics.resolveTLSSuccess.Inc()
		c.tls = append(c.tls, formattedTLSVariable{
			Key:   tls.BuildTLSLabelKeyFromVariable(varName),
			Value: value,
		})
	}
}

func (c *oneShotSampleConsumer) collectTLSInto(builder *profile.SampleBuilder) {
	for _, tlsVariable := range c.tls {
		builder.AddStringLabel(tlsVariable.Key, tlsVariable.Value)
	}
}

type formattedEnvVariable struct {
	Key   string
	Value string
}

func (c *oneShotSampleConsumer) collectEnvironment() {
	processEnvs := c.p.procs.GetEnvs(linux.CurrentNamespacePID(c.sample.Pid))
	c.doCollectEnvironment(processEnvs)
}

func (c *oneShotSampleConsumer) doCollectEnvironment(processEnvs map[string]string) {
	c.env = make([]formattedEnvVariable, 0, len(processEnvs))
	for key, value := range processEnvs {
		_, ok := c.envWhitelist[key]
		if ok {
			c.env = append(c.env, formattedEnvVariable{
				Key:   env.BuildEnvLabelKey(key),
				Value: value,
			})
		}
	}
}

func (c *oneShotSampleConsumer) collectEnvironmentInto(builder *profile.SampleBuilder) {
	for _, env := range c.env {
		builder.AddStringLabel(env.Key, env.Value)
	}
}

func (c *oneShotSampleConsumer) collectKernelStackInto(builder *profile.SampleBuilder) {
	for _, ip := range c.sample.Kernstack {
		if ip == 0 {
			continue
		}

		loc := builder.AddNativeLocation(ip)

		kfunc := c.p.kallsyms.Resolve(ip)
		if kfunc != "" {
			loc.AddFrame().SetName(kfunc).SetMangledName(kfunc).Finish()
		}

		loc.SetMapping().
			SetOffset(0xffffffffffff0000).
			SetPath(profile.KernelSpecialMapping).
			Finish()

		loc.Finish()
		c.stacklen++
	}
}

func (c *oneShotSampleConsumer) processUserSpaceLocation(ctx context.Context, loc *profile.LocationBuilder, ip uint64) {
	for _, s := range c.p.jitSymbolizers {
		out, ok := s.Resolve(linux.CurrentNamespacePID(c.sample.Pid), ip)
		if !ok {
			continue
		}
		loc.AddFrame().SetName(out.SymbolName).SetMangledName(out.SymbolName).Finish()
		loc.SetMapping().SetPath(out.MappingName).Finish()
		loc.Finish()
		return
	}
	mapping, err := c.p.dsoStorage.ResolveMapping(ctx, linux.CurrentNamespacePID(c.sample.Pid), ip)
	if err == nil {
		offset := mapping.Offset
		if mapping.BuildInfo != nil {
			// This logic is broken for binaries with multiple executable sections (e.g. BOLT-ed binaries),
			// as the offset seems to always become zero for any but first executable mapping.
			// TODO : PERFORATOR-560
			// This only works for binaries with a single executable segment and FirstPhdr.Offset == 0
			// mapping.Begin - mapping.BaseAddress is ELF vaddr of the mapping.
			// Conversion from ELF vaddr to ELF offset is done by subtracting corresponding phdr.Vaddr and adding phdr.Off
			offset = mapping.Begin - mapping.BaseAddress - mapping.BuildInfo.FirstPhdr.Vaddr
		}

		m := loc.SetMapping().
			SetBegin(mapping.Begin).
			SetEnd(mapping.End).
			SetOffset(offset).
			SetPath(mapping.Path)

		if b := mapping.BuildInfo; b != nil {
			m.SetBuildID(b.BuildID)
		}

		m.Finish()
	} else {
		c.p.procs.MaybeRescanProcess(ctx, linux.CurrentNamespacePID(c.sample.Pid))
	}

	loc.Finish()
}

func (c *oneShotSampleConsumer) collectUserStackInto(ctx context.Context, builder *profile.SampleBuilder) {
	for _, ip := range c.sample.Userstack {
		if ip == 0 {
			continue
		}

		loc := builder.AddNativeLocation(ip)
		c.processUserSpaceLocation(ctx, loc, ip)
		c.stacklen++
	}
}

func (c *oneShotSampleConsumer) collectInterpreterStackInto(
	langMtr *languageCollectionMetrics,
	builder *profile.SampleBuilder,
	stackProcessor *sampleStackProcessor,
	stack *unwinder.InterpreterStack,
) {
	mtr := stackProcessor.Process(builder, stack)
	c.stacklen += int(mtr.framesCount)
	langMtr.collectedFrameCount.Add(int64(mtr.framesCount))
	langMtr.unsymbolizedFrameCount.Add(int64(mtr.unsymbolizedFramesCount))
}

func (c *oneShotSampleConsumer) collectStacksInto(ctx context.Context, builder *profile.SampleBuilder) {
	if enablePython := c.p.conf.BPF.TracePython; enablePython != nil && *enablePython {
		c.collectInterpreterStackInto(
			&c.p.metrics.pythonMetrics,
			builder,
			c.pythonProcessor,
			&c.sample.PythonStack,
		)
	}

	if enablePhp := c.p.conf.FeatureFlagsConfig.EnablePHP; enablePhp != nil && *enablePhp {
		c.collectInterpreterStackInto(
			&c.p.metrics.phpMetrics,
			builder,
			c.phpProcessor,
			&c.sample.PhpStack,
		)
	}

	c.collectKernelStackInto(builder)
	c.collectUserStackInto(ctx, builder)
}

func (c *oneShotSampleConsumer) collectSampleTime(builder *profile.SampleBuilder) {
	builder.AddIntLabel(AbsoluteTimestampLabel, c.sampleTime.UnixNano(), "ns")
}

func (c *oneShotSampleConsumer) collectWallTime(builder *profile.SampleBuilder) {
	builder.AddValue(int64(c.sample.Timedelta))
}

func (c *oneShotSampleConsumer) collectEventCount(builder *profile.SampleBuilder) {
	builder.AddValue(int64(c.sample.Value))
}

func (c *oneShotSampleConsumer) collectSignalInto(builder *profile.SampleBuilder) error {
	if c.sample.SampleType != unwinder.SampleTypeTracepointSignalDeliver {
		return fmt.Errorf("cannot collect signal info from sample of type %s", c.sample.SampleType.String())
	}

	signo := c.sample.SampleConfig.GetSig()
	signame := unix.SignalName(syscall.Signal(signo))
	builder.AddStringLabel("signal:name", signame)

	return nil
}

func (c *oneShotSampleConsumer) collectLBRStackInto(ctx context.Context, builder *profile.SampleBuilder) {
	for i := 0; i < int(c.sample.LbrValues.Nr); i++ {
		lbrEntry := c.sample.LbrValues.Entries[i]
		from := lbrEntry.From
		to := lbrEntry.To
		if from == 0 || to == 0 {
			break
		}

		processAddress := func(ip uint64) {
			loc := builder.AddNativeLocation(ip)
			c.processUserSpaceLocation(ctx, loc, ip)
		}
		processAddress(from)
		processAddress(to)
	}
}

// for testing purposes
func (c *oneShotSampleConsumer) initBuilderMinimal(name string, sampleTypes []profile.SampleType) *profile.SampleBuilder {
	return c.profileBuilder.EnsureBuilder(name, sampleTypes).AddTimestampedSample(c.sample.Pid, c.sampleTime)
}

func (c *oneShotSampleConsumer) initBuilderCommon(name string, sampleTypes []profile.SampleType) *profile.SampleBuilder {
	builder := c.initBuilderMinimal(name, sampleTypes).
		AddIntLabel("pid", int64(c.sample.Pid), "pid").
		AddIntLabel("tid", int64(c.sample.Tid), "tid").
		AddStringLabel("comm", copy.ZeroTerminatedString(c.sample.ThreadComm[:])).
		AddStringLabel("process_comm", copy.ZeroTerminatedString(c.sample.ProcessComm[:])).
		AddStringLabel("thread_comm", copy.ZeroTerminatedString(c.sample.ThreadComm[:])).
		AddStringLabel("cgroup", c.p.cgroups.CgroupFullName(c.sample.ParentCgroup))

	if c.features.EnableInnermostPidnsCollection {
		builder.AddIntLabel(InnermostPidnsTidLabel, int64(c.sample.InnermostPidnsTid), "id")
		builder.AddIntLabel(InnermostPidnsPidLabel, int64(c.sample.InnermostPidnsPid), "id")
	}

	c.collectWorkloadInto(builder)
	c.collectEnvironmentInto(builder)
	c.collectTLSInto(builder)

	return builder
}

func (c *oneShotSampleConsumer) recordSample(ctx context.Context) {
	var err error

	c.collectEnvironment()
	c.collectTLS(ctx)

	switch c.sample.SampleType {
	case unwinder.SampleTypePerfEvent:
		perfEvent := c.sample.SampleConfig.GetPerfEvent()
		// TODO: remove this if and implement per-perf-event sample consuming logic using custom perf event filters from sample_filter.go
		if perfEvent.Attr.Type != perfevent.AMDFam19hBRS.Type || perfEvent.Attr.Config != perfevent.AMDFam19hBRS.Config {
			c.recordCPUSample(ctx)
		}
		c.recordLBRSample(ctx)
	case unwinder.SampleTypeKprobeFinishTaskSwitch:
		c.recordCPUSample(ctx)
	case unwinder.SampleTypeTracepointSignalDeliver:
		err = c.recordSignalSample(ctx)
	case unwinder.SampleTypeUprobe:
		c.recordUprobeSample(ctx)
	default:
		c.p.log.Warn("Skipped sample of unknown type", log.Stringer("type", c.sample.SampleType))
	}

	c.logSample(err)
}

// Writing sample to profile builder which can be flushed by other goroutines (via RestartProfiles) requires synchronization.
func (c *oneShotSampleConsumer) finishSample(builder *profile.SampleBuilder) {
	c.profileBuilder.Lock()
	defer c.profileBuilder.Unlock()
	builder.Finish()
}

// On CPU / perf event profiling.
func (c *oneShotSampleConsumer) recordCPUSample(ctx context.Context) {
	hasWallTime := c.p.conf.BPF.TraceWallTime != nil && *c.p.conf.BPF.TraceWallTime

	var resolvedPerfEvent *perfevent.Event
	if c.sample.SampleType == unwinder.SampleTypePerfEvent {
		resolvedPerfEvent = c.p.eventmanager.GetEvent(perfevent.PerfEventID(c.sample.SampleConfig.GetPerfEvent().Id))
		if resolvedPerfEvent == nil {
			c.p.metrics.unresolvedPerfEventsForSamples.Inc()
			c.p.log.Debug("Skipped sample of unresolved perf event", log.Any("perf_event", c.sample.SampleConfig.GetPerfEvent()))
			return
		}
	}

	// In case of a kprobe_finish_task_switch sample we assume cpu cycles profile
	sampleType := perfevent.CPUCycles.Name()
	unit := perfevent.CPUCycles.Unit()
	if resolvedPerfEvent != nil {
		sampleType = resolvedPerfEvent.Type().Name()
		unit = resolvedPerfEvent.Type().Unit()
	}

	sampleTypes := []profile.SampleType{{Kind: sampleType, Unit: unit}}
	if hasWallTime {
		sampleTypes = append(sampleTypes, profile.SampleType{Kind: "wall", Unit: "seconds"})
	}

	builder := c.initBuilderCommon(sampleType, sampleTypes)

	c.collectEventCount(builder)
	c.collectStacksInto(ctx, builder)
	if c.features.EnableSampleTimeCollection {
		c.collectSampleTime(builder)
	}

	if hasWallTime {
		c.collectWallTime(builder)
	}

	c.finishSample(builder)
}

func (c *oneShotSampleConsumer) recordLBRSample(ctx context.Context) {
	if enable := c.p.conf.BPF.TraceLBR; enable == nil || !*enable {
		return
	}

	sampleTypes := []profile.SampleType{{Kind: "lbr", Unit: "stacks"}}
	builder := c.initBuilderCommon("lbr", sampleTypes)
	c.collectEventCount(builder)
	c.collectLBRStackInto(ctx, builder)

	c.finishSample(builder)
}

func (c *oneShotSampleConsumer) recordSignalSample(ctx context.Context) error {
	if enable := c.p.conf.BPF.TraceSignals; enable == nil || !*enable {
		return nil
	}

	sampleTypes := []profile.SampleType{{Kind: "signal", Unit: "count"}}
	builder := c.initBuilderCommon("signal", sampleTypes)

	builder.AddValue(1)
	c.collectStacksInto(ctx, builder)
	if c.features.EnableSampleTimeCollection {
		c.collectSampleTime(builder)
	}

	if err := c.collectSignalInto(builder); err != nil {
		return err
	}

	c.finishSample(builder)

	return nil
}

func (c *oneShotSampleConsumer) resolveUprobe(ctx context.Context) *uprobe.UprobeInfo {
	topStackIP := c.sample.Userstack[0]
	if topStackIP == 0 {
		return nil
	}

	mapping, err := c.p.dsoStorage.ResolveMapping(ctx, linux.CurrentNamespacePID(c.sample.Pid), topStackIP)
	if err != nil {
		c.p.log.Warn("Failed to resolve uprobe mapping", log.UInt64("top_stack_ip", topStackIP), log.Error(err))
		return nil
	}

	// Sanity check, this must never happen
	if mapping.BuildInfo == nil {
		c.p.log.Error("No build info for resolved mapping", log.UInt64("top_stack_ip", topStackIP))
		return nil
	}

	return c.p.uprobeRegistry.Resolve(uprobe.BinaryInfo{
		Offset:  topStackIP - mapping.Begin + mapping.Offset,
		BuildID: mapping.BuildInfo.BuildID,
	})
}

func (c *oneShotSampleConsumer) recordUprobeSample(ctx context.Context) {
	uprobeInfo := c.resolveUprobe(ctx)
	if uprobeInfo == nil {
		c.p.log.Warn("Failed to resolve uprobe info", log.UInt64("top_stack_ip", c.sample.Userstack[0]))
		return
	}

	c.p.log.Debug("Resolved uprobe info", log.Any("uprobe_info", uprobeInfo))

	sampleTypeKind := uprobeInfo.SampleKind
	if sampleTypeKind == "" {
		// fallback to "uprobe"
		sampleTypeKind = sampletype.SampleTypeUprobe
	}
	sampleTypes := []profile.SampleType{{Kind: sampleTypeKind, Unit: "count"}}

	builder := c.initBuilderCommon(uprobeInfo.ProfileName, sampleTypes)

	builder.AddValue(1)
	c.collectStacksInto(ctx, builder)
	if c.features.EnableSampleTimeCollection {
		c.collectSampleTime(builder)
	}

	c.finishSample(builder)
}

func (c *oneShotSampleConsumer) logSample(err error) {
	c.p.log.Debug("Consumed sample",
		log.Error(err),
		log.Stringer("sampletype", c.sample.SampleType),
		log.Binary("sampleconfig", c.sample.SampleConfig.UnionBuf[:]),
		log.UInt64("events", c.sample.Value),
		log.UInt64("timedelta", c.sample.Timedelta),
		log.UInt16("cpu", c.sample.Cpu),
		log.String("threadcomm", copy.ZeroTerminatedString(c.sample.ThreadComm[:])),
		log.String("proccomm", copy.ZeroTerminatedString(c.sample.ProcessComm[:])),
		log.UInt32("pid", c.sample.Pid),
		log.UInt32("tid", c.sample.Tid),
		log.UInt32("innermost_pidns_tid", c.sample.InnermostPidnsTid),
		log.UInt64("starttime", c.sample.Starttime),
		log.String("cgroup", c.p.cgroups.CgroupFullName(c.sample.ParentCgroup)),
		log.String("workload", c.cgroupRel),
		log.UInt64("cgroup_id", c.sample.ParentCgroup),
		log.Int("stacklen", c.stacklen),
		log.UInt32("runtime", c.sample.Runtime),
		log.Int("tlsvars", len(c.tls)),
		log.UInt64("lbrvals", c.sample.LbrValues.Nr),
		log.Int("envvars", len(c.env)),
	)
}

func (c *oneShotSampleConsumer) maybeFlushProfile() {
	if time.Since(c.profileBuilder.ProfileStartTime()) >= c.p.conf.Egress.Interval {
		labeledProfiles := c.profileBuilder.RestartProfiles()
		for _, profile := range labeledProfiles.Profiles {
			flushed := c.p.flushProfile(client.LabeledProfile{
				Profile: profile,
				Labels:  labeledProfiles.Labels,
			})
			if !flushed {
				c.p.metrics.droppedProfiles.Inc()
			}
		}
	}
}

func (c *oneShotSampleConsumer) recordSampleConsumeLatency() {
	latency := time.Since(c.sampleTime)
	c.p.metrics.sampleProcessingLatencySum.Add(int64(latency.Milliseconds()))
}

func (c *oneShotSampleConsumer) consume(ctx context.Context) {
	defer c.recordSampleConsumeLatency()
	c.p.procs.DiscoverProcess(ctx, linux.CurrentNamespacePID(c.sample.Pid))
	c.countMetrics(ctx)
	c.recordSample(ctx)
	c.maybeFlushProfile()
}

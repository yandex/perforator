package profiler

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/copy"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/tls"
)

type SampleConsumer struct {
	p      *Profiler
	sample *unwinder.RecordSample

	profileBuilder *multiProfileBuilder
	envWhitelist   map[string]struct{}

	stacklen  int
	env       []formattedEnvVariable
	tls       []formattedTLSVariable
	cgroupRel string
}

func NewSampleConsumer(p *Profiler, envWhitelist map[string]struct{}, sample *unwinder.RecordSample) *SampleConsumer {
	return &SampleConsumer{
		p:            p,
		sample:       sample,
		envWhitelist: envWhitelist,
	}
}

func (c *SampleConsumer) countMetrics(ctx context.Context) {
	// Count mappings cache hit/miss rate.
	var stacklen uint64
	for _, ip := range c.sample.Userstack {
		if ip == 0 {
			continue
		}
		stacklen += 1

		_, err := c.p.dsoStorage.ResolveAddress(ctx, linux.ProcessID(c.sample.Pid), ip)
		if err == nil {
			c.p.metrics.mappingsHit.Inc()
		} else {
			c.p.metrics.mappingsMiss.Inc()
		}
	}
}

func (c *SampleConsumer) getSampleCollector() bool {
	c.p.pidsmu.RLock()
	defer c.p.pidsmu.RUnlock()

	if c.p.wholeSystem != nil {
		c.profileBuilder = c.p.wholeSystem
		return true
	}

	if trackedProcess := c.p.pids[int(c.sample.Pid)]; trackedProcess != nil {
		c.profileBuilder = trackedProcess.builder
		return true
	}

	if trackedEvent := c.p.cgroups.GetTrackedEvent(c.sample.ParentCgroup); trackedEvent != nil {
		c.profileBuilder = trackedEvent.(*trackedCgroup).builder
		return true
	}

	c.p.log.Debug(
		"Failed to find tracked event",
		log.UInt32("pid", c.sample.Pid),
		log.UInt64("cgroupid", c.sample.ParentCgroup),
		log.String("name", c.p.cgroups.CgroupFullName(c.sample.ParentCgroup)),
	)

	return false
}

const (
	// (u64)-1, must match END_OF_CGROUP_LIST in cgroups.h
	endOfCgroupList = ^uint64(0)
)

func (c *SampleConsumer) collectWorkloadInto(builder *profile.SampleBuilder) {
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

func (c *SampleConsumer) collectTLS(ctx context.Context) {
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

		varName, err := c.p.dsoStorage.ResolveTLSName(ctx, linux.ProcessID(c.sample.Pid), variable.Offset)
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

func (c *SampleConsumer) collectTLSInto(builder *profile.SampleBuilder) {
	for _, tlsVariable := range c.tls {
		builder.AddStringLabel(tlsVariable.Key, tlsVariable.Value)
	}
}

type formattedEnvVariable struct {
	Key   string
	Value string
}

func (c *SampleConsumer) collectEnvironment() {
	processEnvs := c.p.procs.GetEnvs(c.sample.Pid)
	c.doCollectEnvironment(processEnvs)
}

func (c *SampleConsumer) doCollectEnvironment(processEnvs map[string]string) {
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

func (c *SampleConsumer) collectEnvironmentInto(builder *profile.SampleBuilder) {
	for _, env := range c.env {
		builder.AddStringLabel(env.Key, env.Value)
	}
}

func (c *SampleConsumer) collectKernelStackInto(builder *profile.SampleBuilder) {
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

func (c *SampleConsumer) processUserSpaceLocation(ctx context.Context, loc *profile.LocationBuilder, ip uint64) {
	if c.p.enablePerfMaps {
		name, ok := c.p.perfmap.Resolve(linux.ProcessID(c.sample.Pid), ip)
		if ok {
			loc.AddFrame().SetName(name).SetMangledName(name).Finish()
		}
	}
	mapping, err := c.p.dsoStorage.ResolveMapping(ctx, linux.ProcessID(c.sample.Pid), ip)
	if err == nil && mapping != nil {
		offset := uint64(mapping.Offset)
		if mapping.BuildInfo != nil {
			// This logic is broken for binaries with multiple executable sections (e.g. BOLT-ed binaries),
			// as the offset seems to always become zero for any but first executable mapping.
			// TODO : PERFORATOR-560
			offset = mapping.Begin - mapping.BaseAddress - mapping.BuildInfo.FirstPhdrOffset
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
		c.p.procs.MaybeRescanProcess(ctx, linux.ProcessID(c.sample.Pid))
	}

	loc.Finish()
}

func (c *SampleConsumer) collectUserStackInto(ctx context.Context, builder *profile.SampleBuilder) {
	for _, ip := range c.sample.Userstack {
		if ip == 0 {
			continue
		}

		loc := builder.AddNativeLocation(ip)
		c.processUserSpaceLocation(ctx, loc, ip)
		c.stacklen++
	}
}

func (c *SampleConsumer) collectWallTime(builder *profile.SampleBuilder) {
	builder.AddValue(int64(c.sample.Timedelta))
}

func (c *SampleConsumer) collectEventCount(builder *profile.SampleBuilder) {
	builder.AddValue(int64(c.sample.Value))
}

func (c *SampleConsumer) collectSignalInto(builder *profile.SampleBuilder) error {
	if c.sample.SampleType != unwinder.SampleTypeTracepointSignalDeliver {
		return fmt.Errorf("cannot collect signal info from sample of type %s", c.sample.SampleType.String())
	}

	signo := int(c.sample.SampleConfig)
	signame := unix.SignalName(syscall.Signal(signo))
	builder.AddStringLabel("signal:name", signame)

	return nil
}

func (c *SampleConsumer) processPythonFrame(loc *profile.LocationBuilder, frame *unwinder.PythonFrame) {
	if frame.SymbolKey.CoFirstlineno == -1 {
		loc.AddFrame().
			SetName(python_models.PythonTrampolineFrame).
			Finish()
		return
	}

	symbol, exists := c.p.pythonSymbolizer.Symbolize(&frame.SymbolKey)
	if !exists {
		c.p.metrics.unsymbolizedPythonFrameCount.Inc()
		loc.AddFrame().
			SetName(python_models.UnsymbolizedPythonLocation).
			SetStartLine(int64(frame.SymbolKey.CoFirstlineno)).
			Finish()
		return
	}

	loc.AddFrame().
		SetName(symbol.QualName).
		SetFilename(symbol.FileName).
		SetStartLine(int64(frame.SymbolKey.CoFirstlineno)).
		Finish()
}

func (c *SampleConsumer) collectPythonStackInto(builder *profile.SampleBuilder) {
	if enable := c.p.conf.BPF.TracePython; enable == nil || !*enable {
		return
	}

	c.p.metrics.collectedPythonFrameCount.Add(int64(c.sample.PythonStackLen))

	for i := 0; i < int(c.sample.PythonStackLen); i++ {
		frame := &c.sample.PythonStack[i]

		loc := builder.AddPythonLocation(&profile.PythonLocationKey{
			CodeObjectAddress:     frame.SymbolKey.CodeObject,
			CodeObjectFirstLineNo: frame.SymbolKey.CoFirstlineno,
		})

		loc.SetMapping().SetPath(python_models.PythonSpecialMapping).Finish()
		c.processPythonFrame(loc, frame)

		loc.Finish()
		c.stacklen++
	}
}

func (c *SampleConsumer) collectLBRStackInto(ctx context.Context, builder *profile.SampleBuilder) {
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
func (c *SampleConsumer) initBuilderMinimal(name string, sampleTypes []profile.SampleType) *profile.SampleBuilder {
	return c.profileBuilder.EnsureBuilder(name, sampleTypes).Add(c.sample.Pid)
}

func (c *SampleConsumer) initBuilderCommon(name string, sampleTypes []profile.SampleType) *profile.SampleBuilder {
	builder := c.initBuilderMinimal(name, sampleTypes).
		AddIntLabel("pid", int64(c.sample.Pid), "pid").
		AddIntLabel("tid", int64(c.sample.Tid), "tid").
		AddStringLabel("comm", copy.ZeroTerminatedString(c.sample.ThreadComm[:])).
		AddStringLabel("process_comm", copy.ZeroTerminatedString(c.sample.ProcessComm[:])).
		AddStringLabel("thread_comm", copy.ZeroTerminatedString(c.sample.ThreadComm[:])).
		AddStringLabel("cgroup", c.p.cgroups.CgroupFullName(c.sample.ParentCgroup))

	c.collectWorkloadInto(builder)
	c.collectEnvironmentInto(builder)
	c.collectTLSInto(builder)

	return builder
}

func (c *SampleConsumer) recordSample(ctx context.Context) {
	var err error

	c.collectEnvironment()
	c.collectTLS(ctx)

	switch c.sample.SampleType {
	case unwinder.SampleTypePerfEvent:
		c.recordCPUSample(ctx)
		c.recordLBRSample(ctx)
	case unwinder.SampleTypeKprobeFinishTaskSwitch:
		c.recordCPUSample(ctx)
	case unwinder.SampleTypeTracepointSignalDeliver:
		err = c.recordSignalSample(ctx)
	default:
		c.p.log.Warn("Skipped sample of unknown type", log.Stringer("type", c.sample.SampleType))
	}

	c.logSample(err)
}

// On CPU / perf event profiling.
func (c *SampleConsumer) recordCPUSample(ctx context.Context) {
	hasWallTime := c.p.conf.BPF.TraceWallTime != nil && *c.p.conf.BPF.TraceWallTime

	sampleTypes := []profile.SampleType{{Kind: "cpu", Unit: "cycles"}}
	if hasWallTime {
		sampleTypes = append(sampleTypes, profile.SampleType{Kind: "wall", Unit: "seconds"})
	}

	builder := c.initBuilderCommon("cpu", sampleTypes)

	c.collectEventCount(builder)
	c.collectPythonStackInto(builder)
	c.collectKernelStackInto(builder)
	c.collectUserStackInto(ctx, builder)

	if hasWallTime {
		c.collectWallTime(builder)
	}

	builder.Finish()
}

func (c *SampleConsumer) recordLBRSample(ctx context.Context) {
	if enable := c.p.conf.BPF.TraceLBR; enable == nil || !*enable {
		return
	}

	sampleTypes := []profile.SampleType{{Kind: "lbr", Unit: "stacks"}}
	builder := c.initBuilderCommon("lbr", sampleTypes)
	c.collectEventCount(builder)
	c.collectLBRStackInto(ctx, builder)
	builder.Finish()
}

func (c *SampleConsumer) recordSignalSample(ctx context.Context) error {
	if enable := c.p.conf.BPF.TraceSignals; enable == nil || !*enable {
		return nil
	}

	sampleTypes := []profile.SampleType{{Kind: "signal", Unit: "count"}}
	builder := c.initBuilderCommon("signal", sampleTypes)

	builder.AddValue(1)
	c.collectPythonStackInto(builder)
	c.collectKernelStackInto(builder)
	c.collectUserStackInto(ctx, builder)

	if err := c.collectSignalInto(builder); err != nil {
		return err
	}

	builder.Finish()

	return nil
}

func (c *SampleConsumer) logSample(err error) {
	c.p.log.Debug("Consumed sample",
		log.Error(err),
		log.Stringer("sampletype", c.sample.SampleType),
		log.UInt64("sampleconfig", c.sample.SampleConfig),
		log.UInt64("events", c.sample.Value),
		log.UInt64("timedelta", c.sample.Timedelta),
		log.UInt16("cpu", c.sample.Cpu),
		log.String("threadcomm", copy.ZeroTerminatedString(c.sample.ThreadComm[:])),
		log.String("proccomm", copy.ZeroTerminatedString(c.sample.ProcessComm[:])),
		log.UInt32("pid", c.sample.Pid),
		log.UInt32("tid", c.sample.Tid),
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

func (c *SampleConsumer) maybeFlushProfile() {
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

func (c *SampleConsumer) Consume(ctx context.Context) {
	c.p.procs.DiscoverProcess(ctx, linux.ProcessID(c.sample.Pid))
	c.countMetrics(ctx)

	found := c.getSampleCollector()
	if !found {
		return
	}

	c.recordSample(ctx)
	c.maybeFlushProfile()
}

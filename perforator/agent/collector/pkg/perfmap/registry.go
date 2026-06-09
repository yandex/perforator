package perfmap

import (
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profilerext"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmattach"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/symboltable"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/pidfd"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type processState int

const (
	processStateInitializing processState = iota
	processStateInitialized
	processStateTerminalSkip
	processStateTransientSkip
)

type trackedProcess struct {
	state             atomic.Int32
	hasJVMLikeMapping atomic.Bool
	pid               linux.CurrentNamespacePID
	perfmap           *perfMap
	javaConn          *jvmattach.VirtualMachineConn
}

type Registry struct {
	logger        xlog.Logger
	mu            sync.RWMutex
	procs         map[linux.CurrentNamespacePID]*trackedProcess
	syms          *symboltable.Table[linux.CurrentNamespacePID, string]
	jvmDialer     *jvmattach.Dialer
	enableJVM     bool
	started       atomic.Bool
	registerQueue chan *trackedProcess

	totalSyms                   metrics.IntGauge
	processIgnoredEnvParseError metrics.Counter
	processIgnoredNotEnabled    metrics.Counter
	processCount                metrics.IntGauge
	totalRebuildTime            metrics.Gauge
	errorCount                  metrics.Counter
	discoveryDuration           metrics.Timer
}

func NewRegistry(logger log.Logger, mReg metrics.Registry, enableJVM bool) *Registry {
	mReg = mReg.WithPrefix("perfmap")
	discoveryDurationBuckets := metrics.MakeExponentialDurationBuckets(time.Millisecond, 10, 5)
	reg := &Registry{
		logger: xlog.Wrap(logger.WithName("perfmap")),
		procs:  make(map[linux.CurrentNamespacePID]*trackedProcess),
		syms:   symboltable.New[linux.CurrentNamespacePID, string](),
		jvmDialer: &jvmattach.Dialer{
			Logger: xlog.Wrap(logger.WithName("perfmap.jvmattach")),
		},
		enableJVM:     enableJVM,
		registerQueue: make(chan *trackedProcess, 1024),

		totalSyms:                   mReg.IntGauge("symbols_total.count"),
		processIgnoredEnvParseError: mReg.WithTags(map[string]string{"reason": "env_parse_error"}).Counter("ignored_process.count"),
		processIgnoredNotEnabled:    mReg.WithTags(map[string]string{"reason": "not_enabled"}).Counter("ignored_process.count"),
		processCount:                mReg.IntGauge("current_tracked_process.count"),
		totalRebuildTime:            mReg.Gauge("index_total_rebuild_time.seconds"),
		errorCount:                  mReg.Counter("refresh_error.count"),
		discoveryDuration:           mReg.DurationHistogram("discovery.duration.seconds", discoveryDurationBuckets),
	}

	mReg.FuncGauge("discovery.queue.size", func() float64 {
		return float64(len(reg.registerQueue))
	})

	return reg
}

func (r *Registry) addProcessEntry(ctx context.Context, pid linux.CurrentNamespacePID) *trackedProcess {
	tp := &trackedProcess{
		pid: pid,
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.procs[pid]
	if ok {
		r.logger.Error(ctx, "Process already registered", logfield.CurrentNamespacePID(pid))
	}
	r.procs[pid] = tp
	return tp
}

func (r *Registry) registerImpl(ctx context.Context, tp *trackedProcess, nspid linux.NamespacedPID, java bool, pfd *pidfd.FD) bool {
	var conn *jvmattach.VirtualMachineConn
	if java {
		var err error
		conn, err = r.jvmDialer.Dial(ctx, jvmattach.Target{
			ProcessFD:     pfd,
			PID:           tp.pid,
			NamespacedPID: nspid,
		})
		if err != nil {
			r.logger.Info(ctx, "Failed to connect to JVM", logfield.CurrentNamespacePID(tp.pid), logfield.NamespacedPID(nspid), log.Error(err))
			return false
		}
	}

	path := fmt.Sprintf("/proc/%d/root/tmp/perf-%d.map", tp.pid, nspid)
	tp.perfmap = newPerfMap(path)
	tp.javaConn = conn
	return true
}

func (r *Registry) registerSync(ctx context.Context, tp *trackedProcess) processState {
	// It is critical that we open pidfd before reading any process information besides its pid.
	// This way, if discovery races with process termination
	// (and potential pid reuse), pidfd_send_signal inside jvmattach.Dialer.Dial will fail.
	// Otherwise we can be sure that all discovery read consistent data.
	pfd, err := pidfd.Open(tp.pid)
	if err != nil {
		r.logger.Info(ctx, "Failed to open pidfd", logfield.CurrentNamespacePID(tp.pid), log.Error(err))
		return processStateTerminalSkip
	}
	defer func() {
		closeErr := pfd.Close()
		if closeErr != nil {
			r.logger.Warn(ctx, "Failed to close pidfd", logfield.CurrentNamespacePID(tp.pid), log.Error(closeErr))
		}
	}()
	process := procfs.FS().Process(tp.pid)

	env, err := process.ListEnvs()
	if err != nil {
		r.logger.Info(
			ctx,
			"Failed to read process environment, skipping process",
			logfield.CurrentNamespacePID(tp.pid),
			log.Error(err),
		)
		r.processIgnoredEnvParseError.Inc()
		return processStateTerminalSkip
	}
	var perfMapConf *processConfig
	perfMapRawConf, ok := env["__PERFORATOR_ENABLE_PERFMAP"]
	if ok {
		var errs []error
		perfMapConf, errs = parseProcessConfig(perfMapRawConf)
		r.logger.Debug(
			ctx,
			"Process enables perfmap",
			logfield.CurrentNamespacePID(tp.pid),
			log.Any("config", perfMapConf),
			log.Errors("errors", errs),
		)
		if !r.enableJVM && perfMapConf.java {
			r.logger.Info(
				ctx,
				"Process requests JVM integration but feature flag is disabled, this request will be ignored",
				logfield.CurrentNamespacePID(tp.pid),
			)
			perfMapConf.java = false
		}
	}
	if perfMapConf == nil {
		// We can't log environment, it is sensitive
		r.logger.Info(
			ctx,
			"Process does not allow perfmap collection, skipping process (late check)",
			logfield.CurrentNamespacePID(tp.pid),
		)
		r.processIgnoredNotEnabled.Inc()
		return processStateTransientSkip
	}
	if perfMapConf.percentage < 100 {
		if rand.Uint32N(100) >= perfMapConf.percentage {
			r.logger.Debug(ctx, "Process enables perfmap randomly and was not sampled")
			return processStateTerminalSkip
		}
	}
	if perfMapConf.java && perfMapConf.jvmVerifyMethod == jvmVerifyMethodMapping && !tp.hasJVMLikeMapping.Load() {
		r.logger.Info(
			ctx,
			"Process does not have mapping with file named libjvm.so, skipping process",
			logfield.CurrentNamespacePID(tp.pid),
		)
		return processStateTransientSkip
	}

	nspid, err := process.GetNamespacedPID()
	if err != nil {
		r.logger.Warn(
			ctx,
			"Failed to get namespaced pid",
			logfield.CurrentNamespacePID(tp.pid),
			log.Error(err),
		)
		// TODO: conversion from CurrentNamespacePID to NamespacedPID is something we should avoid
		nspid = linux.NamespacedPID(tp.pid)
	} else {
		r.logger.Info(
			ctx,
			"Resolved pid in innermost pid_ns",
			logfield.CurrentNamespacePID(tp.pid),
			log.UInt32("nspid", uint32(nspid)),
		)
	}

	ok = r.registerImpl(ctx, tp, nspid, perfMapConf.java, pfd)
	if !ok {
		return processStateTerminalSkip
	}
	return processStateInitialized
}

const (
	singleProcessRegisterTimeout       = 15 * time.Second
	singleProcessRegisterWarnThreshold = 1 * time.Second
)

func (r *Registry) runRegisterWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case tp := <-r.registerQueue:
			regCtx, cancel := context.WithTimeoutCause(
				ctx,
				singleProcessRegisterTimeout,
				fmt.Errorf("process discovery timeout (%v) exceeded", singleProcessRegisterTimeout),
			)
			defer cancel()
			started := time.Now()
			state := r.registerSync(regCtx, tp)
			if state == processStateTerminalSkip {
				r.mu.Lock()
				// TODO(PERFORATOR-561) here is race condition: we may delete another (newer) process which reused this pid
				delete(r.procs, tp.pid)
				r.mu.Unlock()
			} else {
				tp.state.Store(int32(state))
			}
			elapsed := time.Since(started)
			if elapsed > singleProcessRegisterWarnThreshold {
				r.logger.Warn(
					ctx,
					"Process discovery took too long",
					logfield.CurrentNamespacePID(tp.pid),
					log.Duration("elapsed", elapsed),
					log.Duration("threshold", singleProcessRegisterWarnThreshold),
				)
			} else {
				r.logger.Debug(
					ctx,
					"Process discovery completed",
					logfield.CurrentNamespacePID(tp.pid),
					log.Duration("elapsed", elapsed),
				)
			}
		}
	}
}

func (r *Registry) tryEnqueueForDiscovery(ctx context.Context, tp *trackedProcess, info process.ProcessInfo) {
	// TODO: we will still parse environment the second time, because this check happens outside of pidfd protection region
	_, ok := info.Env()["__PERFORATOR_ENABLE_PERFMAP"]
	if !ok {
		r.logger.Debug(
			ctx,
			"Process does not allow perfmap collection, skipping process (early check)",
			logfield.CurrentNamespacePID(info.ProcessID()),
		)
		tp.state.Store(int32(processStateTransientSkip))
		return
	}
	hasJVMLikeMapping := false
	for _, m := range info.Mappings() {
		if strings.Contains(m.Path(), "libjvm.so") {
			hasJVMLikeMapping = true
			break
		}
	}
	tp.hasJVMLikeMapping.Store(hasJVMLikeMapping)

	select {
	case r.registerQueue <- tp:
		r.logger.Debug(
			ctx,
			"Process enqueued for discovery",
			logfield.CurrentNamespacePID(info.ProcessID()),
		)
	default:
		r.logger.Error(
			ctx,
			"Register queue is full, skipping process",
			logfield.CurrentNamespacePID(info.ProcessID()),
		)
	}
}

// OnProcessDiscovery implements process.Listener
func (r *Registry) OnProcessDiscovery(ctx context.Context, info process.ProcessInfo) {
	tp := r.addProcessEntry(ctx, info.ProcessID())
	r.tryEnqueueForDiscovery(ctx, tp, info)
}

// OnProcessRescan implements process.Listener
func (r *Registry) OnProcessRescan(ctx context.Context, info process.ProcessInfo) {
	tp := r.findProcess(info.ProcessID())
	if tp == nil {
		r.logger.Warn(
			ctx,
			"Got Rescan notification for unknown process",
			log.UInt32("pid", uint32(info.ProcessID())),
		)
		return
	}
	if tp.state.Load() != int32(processStateTransientSkip) {
		return
	}
	r.tryEnqueueForDiscovery(ctx, tp, info)
}

// OnProcessDeath implements process.Listener
func (r *Registry) OnProcessDeath(ctx context.Context, pid linux.CurrentNamespacePID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: cancel discovery if it has not completed yet?
	delete(r.procs, pid)
}

func (r *Registry) findProcess(pid linux.CurrentNamespacePID) *trackedProcess {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.procs[pid]
}

func (r *Registry) Resolve(pid linux.CurrentNamespacePID, ip uint64) (profilerext.JITSymbolizerOutput, bool) {
	name, ok := r.syms.Find(pid, ip)
	if ok {
		return profilerext.JITSymbolizerOutput{
			SymbolName:  name,
			MappingName: profile.JITSpecialMapping,
		}, true
	}
	return profilerext.JITSymbolizerOutput{}, false
}

func trySleepContext(ctx context.Context, dur time.Duration) {
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

func (r *Registry) listRefreshTargets() []*trackedProcess {
	r.mu.RLock()
	defer r.mu.RUnlock()
	targets := make([]*trackedProcess, 0, len(r.procs))
	for _, v := range r.procs {
		if v.state.Load() != int32(processStateInitialized) {
			continue
		}
		targets = append(targets, v)
	}
	return targets
}

func (r *Registry) dumpJVMPerfMap(ctx context.Context, tp *trackedProcess) {
	out, err := tp.javaConn.Execute(ctx, [4]string{"jcmd", "Compiler.perfmap"})
	if err != nil {
		r.logger.Info(
			ctx,
			"Failed to execute Compiler.perfmap",
			logfield.CurrentNamespacePID(tp.pid),
			log.Error(err),
		)
		return
	}
	r.logger.Debug(
		ctx,
		"Executed Compiler.perfmap",
		logfield.CurrentNamespacePID(tp.pid),
		log.String("output", out),
	)
}

func (r *Registry) runRefresher(ctx context.Context) {
	firstiter := true
	for {
		if firstiter {
			firstiter = false
		} else {
			trySleepContext(ctx, 15*time.Second)
		}
		if ctx.Err() != nil {
			break
		}
		totalSyms := 0
		var totalRebuildTime time.Duration
		modified := 0
		errors := 0

		targets := r.listRefreshTargets()
		for _, tp := range targets {
			if r.enableJVM && tp.javaConn != nil {
				r.dumpJVMPerfMap(ctx, tp)
			}
			r.logger.Debug(ctx, "Starting perf map parser", logfield.CurrentNamespacePID(tp.pid))
			newSyms, stats, err := tp.perfmap.refresh()
			if err != nil {
				r.logger.Info(ctx, "Failed to refresh perf map", logfield.CurrentNamespacePID(tp.pid), log.Error(err))
				errors++
				continue
			}
			if newSyms != nil {
				r.syms.Put(tp.pid, newSyms)
			}
			if !stats.skipped {
				modified++
			}
			totalSyms += stats.currentSize
			totalRebuildTime += stats.rebuildTime
		}
		r.logger.Debug(
			ctx,
			"Perf map refresh finished",
			log.Int("tracked_processes", len(targets)),
			log.Int("refreshed_processes", modified),
			log.Int("refresh_errors", errors),
			log.Int("total_current_symbols", totalSyms),
			log.Duration("total_rebuild_time", totalRebuildTime),
		)
		r.processCount.Set(int64(len(targets)))
		r.totalSyms.Set(int64(totalSyms))
		r.totalRebuildTime.Set(totalRebuildTime.Seconds())
		r.errorCount.Add(int64(errors))
	}
}

func (r *Registry) Run(parentCtx context.Context) error {
	if r.started.Swap(true) {
		panic("Registry.Run is one-shot")
	}
	wg, ctx := errgroup.WithContext(parentCtx)
	wg.Go(func() error {
		r.runRefresher(ctx)
		return nil
	})
	wg.Go(func() error {
		r.runRegisterWorker(ctx)
		return nil
	})

	return wg.Wait()
}

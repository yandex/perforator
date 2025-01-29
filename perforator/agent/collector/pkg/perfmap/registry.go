package perfmap

import (
	"context"
	"fmt"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/internal/linguist/jvm/jvmattach"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/pidfd"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type trackedProcess struct {
	initialized atomic.Bool
	pid         linux.ProcessID
	perfmap     *perfMap
	javaConn    *jvmattach.VirtualMachineConn
}

type Registry struct {
	logger        log.Logger
	mu            sync.RWMutex
	procs         map[linux.ProcessID]*trackedProcess
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
		logger: logger.WithName("perfmap"),
		procs:  make(map[uint32]*trackedProcess),
		jvmDialer: &jvmattach.Dialer{
			Logger: xlog.New(logger.WithName("perfmap.jvmattach")),
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

func (r *Registry) addProcessEntry(pid linux.ProcessID) *trackedProcess {
	tp := &trackedProcess{
		pid: pid,
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.procs[pid]
	if ok {
		r.logger.Error("Process already registered", log.UInt32("pid", pid))
	}
	r.procs[pid] = tp
	return tp
}

func (r *Registry) registerImpl(ctx context.Context, tp *trackedProcess, nspid linux.ProcessID, java bool, pfd *pidfd.FD) {
	var conn *jvmattach.VirtualMachineConn
	if java {
		var err error
		conn, err = r.jvmDialer.Dial(ctx, jvmattach.Target{
			ProcessFD:     pfd,
			PID:           tp.pid,
			NamespacedPID: nspid,
		})
		if err != nil {
			r.logger.Info("Failed to connect to JVM", log.UInt32("pid", tp.pid), log.Error(err))
			return
		}
	}

	path := fmt.Sprintf("/proc/%d/root/tmp/perf-%d.map", tp.pid, nspid)
	tp.perfmap = newPerfMap(path)
	tp.javaConn = conn
	tp.initialized.Store(true)
}

func (r *Registry) registerSync(ctx context.Context, tp *trackedProcess) bool {
	// It is critical that we open pidfd before reading any process information besides its pid.
	// This way, if discovery races with process termination
	// (and potential pid reuse), pidfd_send_signal inside jvmattach.Dialer.Dial will fail.
	// Otherwise we can be sure that all discovery read consistent data.
	pfd, err := pidfd.Open(tp.pid)
	if err != nil {
		r.logger.Info("Failed to open pidfd", log.UInt32("pid", tp.pid), log.Error(err))
		return false
	}
	defer func() {
		closeErr := pfd.Close()
		if closeErr != nil {
			r.logger.Warn("Failed to close pidfd", log.UInt32("pid", tp.pid), log.Error(closeErr))
		}
	}()
	process := procfs.FS().Process(tp.pid)

	env, err := process.ListEnvs()
	if err != nil {
		r.logger.Info("Failed to read process environment, skipping process", log.UInt32("pid", tp.pid), log.Error(err))
		r.processIgnoredEnvParseError.Inc()
		return false
	}
	var perfMapConf *processConfig
	perfMapRawConf, ok := env["__PERFORATOR_ENABLE_PERFMAP"]
	if ok {
		var errs []error
		perfMapConf, errs = parseProcessConfig(perfMapRawConf)
		r.logger.Debug(
			"Process enables perfmap",
			log.UInt32("pid", tp.pid),
			log.Any("config", perfMapConf),
			log.Errors("errors", errs),
		)
	}
	if perfMapConf == nil {
		// We can't log environment, it is sensitive
		r.logger.Info("Process does not allow perfmap collection, skipping process (late check)", log.UInt32("pid", tp.pid))
		r.processIgnoredNotEnabled.Inc()
		return false
	}
	if perfMapConf.percentage < 100 {
		if rand.Uint32N(100) >= perfMapConf.percentage {
			r.logger.Debug("Process enables perfmap randomly and was not sampled")
			return false
		}
	}

	nspid, err := process.GetNamespacedPID()
	if err != nil {
		r.logger.Warn("Failed to get namespaced pid", log.UInt32("pid", tp.pid), log.Error(err))
		nspid = tp.pid
	} else {
		r.logger.Info("Resolved pid in innermost pid_ns", log.UInt32("pid", tp.pid), log.UInt32("nspid", nspid))
	}

	r.registerImpl(ctx, tp, nspid, perfMapConf.java, pfd)
	return true
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
			ok := r.registerSync(regCtx, tp)
			if !ok {
				r.mu.Lock()
				// TODO(PERFORATOR-561) here is race condition: we may delete another (newer) process which reused this pid
				delete(r.procs, tp.pid)
				r.mu.Unlock()
			}
			elapsed := time.Since(started)
			if elapsed > singleProcessRegisterWarnThreshold {
				r.logger.Warn(
					"Process discovery took too long",
					log.UInt32("pid", tp.pid),
					log.Duration("elapsed", elapsed),
					log.Duration("threshold", singleProcessRegisterWarnThreshold),
				)
			} else {
				r.logger.Debug("Process discovery completed", log.UInt32("pid", tp.pid), log.Duration("elapsed", elapsed))
			}
		}
	}
}

// OnProcessDiscovery implements process.Listener
func (r *Registry) OnProcessDiscovery(info process.ProcessInfo) {
	// TODO: we will still parse environment the second time, because this check happens outside of pidfd protection region
	_, ok := info.Env()["__PERFORATOR_ENABLE_PERFMAP"]
	if !ok {
		r.logger.Debug("Process does not allow perfmap collection, skipping process (early check)", log.UInt32("pid", info.ProcessID()))
		return
	}

	tp := r.addProcessEntry(info.ProcessID())
	select {
	case r.registerQueue <- tp:
	default:
		r.logger.Error("Register queue is full, skipping process", log.UInt32("pid", info.ProcessID()))
		r.mu.Lock()
		defer r.mu.Unlock()
		delete(r.procs, info.ProcessID())
	}
}

// OnProcessDeath implements process.Listener
func (r *Registry) OnProcessDeath(pid linux.ProcessID) {
	r.mu.Lock()
	defer r.mu.Unlock()
	// TODO: cancel discovery if it has not completed yet?
	delete(r.procs, pid)
}

func (r *Registry) findProcess(pid linux.ProcessID) *trackedProcess {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.procs[pid]
}

func (r *Registry) Resolve(pid linux.ProcessID, ip uint64) (string, bool) {
	tp := r.findProcess(pid)
	if tp == nil || !tp.initialized.Load() {
		return "", false
	}
	return tp.perfmap.find(ip)
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
	targets := make([]*trackedProcess, 0, len(r.procs))
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.procs {
		if !v.initialized.Load() {
			continue
		}
		targets = append(targets, v)
	}
	return targets
}

func (r *Registry) dumpJVMPerfMap(ctx context.Context, tp *trackedProcess) {
	out, err := tp.javaConn.Execute(ctx, [4]string{"jcmd", "Compiler.perfmap"})
	if err != nil {
		r.logger.Info("Failed to execute Compiler.perfmap", log.UInt32("pid", tp.pid), log.Error(err))
		return
	}
	r.logger.Debug("Executed Compiler.perfmap", log.UInt32("pid", tp.pid), log.String("output", out))
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
			r.logger.Debug("Starting perf map parser", log.UInt32("pid", tp.pid))
			stats, err := tp.perfmap.refresh()
			if err != nil {
				r.logger.Info("Failed to refresh perf map", log.UInt32("pid", tp.pid), log.Error(err))
				errors++
				continue
			}
			if !stats.skipped {
				modified++
			}
			totalSyms += stats.currentSize
			totalRebuildTime += stats.rebuildTime
		}
		r.logger.Info(
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

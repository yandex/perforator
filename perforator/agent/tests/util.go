package profiler_test

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	agentprofile "github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	storage "github.com/yandex/perforator/perforator/agent/collector/pkg/storage/client"
	"github.com/yandex/perforator/perforator/internal/logfield"
	"github.com/yandex/perforator/perforator/internal/xmetrics"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/uname"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	PageTableSizeKB uint64 = 100000
)

func doUseCgroupsV2(logger func(string, ...any)) error {
	stat, err := os.Stat("/sys/fs/cgroup/unified")
	if err == nil {
		logger("current path stats: %+v", stat)
		return nil
	}
	return fmt.Errorf("TODO: mount cgroupv2")
}

type cgroupManipulatorT struct {
	prefix string
	isv2   bool
}

func (cm *cgroupManipulatorT) isV2() bool {
	return cm.isv2
}

func (cm *cgroupManipulatorT) makeGroup(name string) error {
	return os.MkdirAll(path.Join(cm.prefix, name), 0755)
}

func (cm *cgroupManipulatorT) moveToCgroup(pid linux.CurrentNamespacePID, cgroup string) error {
	path := path.Join(cm.prefix, cgroup, "cgroup.procs")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open cgroup membership file: %w", err)
	}
	_, err = fmt.Fprintln(f, pid)
	if err != nil {
		return fmt.Errorf("failed to write pid %d to cgroup membership file: %w", pid, err)
	}
	return nil
}

var cgroupManipulator *cgroupManipulatorT

func prepareEnvImpl(logger func(string, ...any)) error {
	useCgroupsV2Env := os.Getenv("PERFORATOR_TEST_USE_CGROUPS_V2")
	logger("%s=%s", "PERFORATOR_TEST_USE_CGROUPS_V2", useCgroupsV2Env)
	if useCgroupsV2Env != "" {
		cgroupManipulator = &cgroupManipulatorT{
			prefix: "/sys/fs/cgroup/unified",
			isv2:   true,
		}
		logger("Using cgroups v2")
		err := doUseCgroupsV2(logger)
		if err != nil {
			return fmt.Errorf("failed to use cgroups v2: %v", err)
		}
	} else {
		cgroupManipulator = &cgroupManipulatorT{
			prefix: "/sys/fs/cgroup/freezer",
			isv2:   false,
		}
	}
	logger("Configuring cgroup hierarchy")

	err := cgroupManipulator.makeGroup("pod/container")
	if err != nil {
		return fmt.Errorf("failed to create cgroup: %w", err)
	}

	return nil
}

var prepareEnvError error
var prepareEnvOnce sync.Once

func prepareEnv(t testing.TB) {
	t.Helper()
	prepareEnvOnce.Do(func() {
		prepareEnvError = prepareEnvImpl(t.Logf)
	})
	if prepareEnvError != nil {
		t.Fatalf("failed to setup environment: %v", prepareEnvError)
	}
}

type sampleWaiter struct {
	ch    chan<- struct{}
	pid   linux.CurrentNamespacePID
	count int
}

type testEventListener struct {
	logger  xlog.Logger
	mu      sync.Mutex
	waiters []*sampleWaiter
	count   map[linux.CurrentNamespacePID]int
}

// OnSampleStored implements profiler.EventListener
func (el *testEventListener) OnSampleStored(pid linux.CurrentNamespacePID) {
	el.logger.Debug(context.TODO(), "Received sample", logfield.CurrentNamespacePID(pid))
	el.mu.Lock()
	defer el.mu.Unlock()
	el.count[pid]++
	newWaiters := make([]*sampleWaiter, 0, len(el.waiters))
	for _, w := range el.waiters {
		if w.pid == pid && w.count <= el.count[pid] {
			close(w.ch)
		} else {
			newWaiters = append(newWaiters, w)
		}
	}
	el.waiters = newWaiters
}

func (el *testEventListener) waitForSampleImpl(pid linux.CurrentNamespacePID, count int) <-chan struct{} {
	el.mu.Lock()
	defer el.mu.Unlock()
	if el.count[pid] >= count {
		return nil
	}
	el.logger.Info(context.TODO(), "Registering in waiter list", logfield.CurrentNamespacePID(pid), log.Int("count", count))
	ch := make(chan struct{})
	el.waiters = append(el.waiters, &sampleWaiter{
		ch:    ch,
		pid:   pid,
		count: count,
	})
	return ch
}

func (el *testEventListener) waitForSamples(ctx context.Context, pid linux.CurrentNamespacePID, count int) error {
	ch := el.waitForSampleImpl(pid, count)
	if ch == nil {
		return nil
	}
	select {
	case <-ch:
		el.logger.Info(ctx, "Wait done", logfield.CurrentNamespacePID(pid), log.Int("count", count))
		return nil
	case <-ctx.Done():
		return fmt.Errorf("interrupted while waiting for %d samples for pid %d: %w", count, pid, context.Cause(ctx))
	}
}

type processDiscoveryWaiter struct {
	ch  chan<- struct{}
	pid linux.CurrentNamespacePID
}

type testProcessListener struct {
	logger     xlog.Logger
	mu         sync.Mutex
	waiters    []*processDiscoveryWaiter
	discovered map[linux.CurrentNamespacePID]struct{}
}

func (l *testProcessListener) OnProcessDiscovery(ctx context.Context, info process.ProcessInfo) {
	l.logger.Debug(ctx, "Discovered process", logfield.CurrentNamespacePID(info.ProcessID()))
	l.mu.Lock()
	defer l.mu.Unlock()
	l.discovered[info.ProcessID()] = struct{}{}
	newWaiters := make([]*processDiscoveryWaiter, 0, len(l.waiters))
	for _, w := range l.waiters {
		if w.pid == info.ProcessID() {
			close(w.ch)
		} else {
			newWaiters = append(newWaiters, w)
		}
	}
	l.waiters = newWaiters
}

func (l *testProcessListener) OnProcessRescan(ctx context.Context, info process.ProcessInfo) {
	l.OnProcessDiscovery(ctx, info)
}

func (l *testProcessListener) OnProcessDeath(ctx context.Context, pid linux.CurrentNamespacePID) {}

func (l *testProcessListener) ensureWaiter(pid linux.CurrentNamespacePID) <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	ch := make(chan struct{})
	if _, ok := l.discovered[pid]; ok {
		close(ch)
		return ch
	}

	l.waiters = append(l.waiters, &processDiscoveryWaiter{
		ch:  ch,
		pid: pid,
	})
	return ch
}

func (l *testProcessListener) waitForProcessRegistration(ctx context.Context, pid linux.CurrentNamespacePID) error {
	ch := l.ensureWaiter(pid)

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("interrupted while waiting for process discovery %d: %w", pid, context.Cause(ctx))
	}
}

func setupProfiler(t testing.TB, c config.Config) (xlog.Logger, xmetrics.Registry, *testEventListener, *testProcessListener, *profiler.Profiler) {
	t.Helper()
	l := xlog.ForTest(t)

	ctx := t.Context()
	release, err := uname.SystemRelease()
	require.NoError(t, err)
	l.Info(ctx, "Loaded kernel version", log.String("release", release))

	r := xmetrics.NewRegistry()

	// Apply common config rewrites.
	c.ProcessDiscovery.IgnoreUnrelatedProcesses = true
	c.BPF.PageTableSizeKB = &PageTableSizeKB
	c.Egress.Interval = time.Second
	if c.BPF.TraceLBR == nil {
		c.BPF.TraceLBR = ptr.Bool(false)
	}

	if cgroupManipulator.isV2() {
		c.Cgroups.CgroupHints = &cgroups.CgroupHints{
			Strong: &cgroups.CgroupHint{
				Version:    cgroups.CgroupV2,
				MountPoint: "/sys/fs/cgroup/unified",
			},
		}
	}
	el := &testEventListener{
		count:  make(map[linux.CurrentNamespacePID]int),
		logger: l.WithName("ProfilerEventListener"),
	}
	pl := &testProcessListener{
		logger:     l.WithName("ProfilerProcessListener"),
		discovered: make(map[linux.CurrentNamespacePID]struct{}),
	}
	p, err := profiler.NewProfiler(&c, l.Logger(), r.WithPrefix("profiler"), profiler.WithEventListener(el), profiler.WithProcessListener(pl))
	require.NoError(t, err)

	return l, r, el, pl, p
}

// decodedProfile keeps test assertions independent of the storage representation.
type decodedProfile struct {
	Profile *agentprofile.Profile
	Labels  map[string]string
}

func decodeStoredProfiles(t *testing.T, stored []storage.LabeledProfile) []decodedProfile {
	t.Helper()
	result := make([]decodedProfile, 0, len(stored))
	for _, p := range stored {
		parsed, err := p.Profile.ParsePprof()
		require.NoError(t, err)
		result = append(result, decodedProfile{Profile: &agentprofile.Profile{Profile: parsed}, Labels: p.Labels})
	}
	return result
}

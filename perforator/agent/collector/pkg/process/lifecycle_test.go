package process

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func (s *publicationBPFState) RemoveProcess(linux.CurrentNamespacePID) error { return nil }

func newLifecycleTestRegistry(t *testing.T) *ProcessRegistry {
	t.Helper()
	l := xlog.ForTest(t)
	storage, err := dso.NewStorage(l, mock.NewRegistry(nil), nil, nil)
	require.NoError(t, err)
	r := &ProcessRegistry{
		log: l, dsoStorage: storage,
		procs:    make(map[linux.CurrentNamespacePID]*processInfo),
		procchan: make(chan *processRegistration, 32),
		state:    &publicationBPFState{enable: func(linux.CurrentNamespacePID, *unwinder.ProcessInfo) error { return nil }},
	}
	r.procsGeneration.Store(1)
	return r
}

type scanFunc func(context.Context, func(context.Context, linux.CurrentNamespacePID)) error

func (f scanFunc) Scan(ctx context.Context, discover func(context.Context, linux.CurrentNamespacePID)) error {
	return f(ctx, discover)
}

func TestReuseDuringScanDoesNotAdvanceScanGeneration(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	r.processScanner = scanFunc(func(ctx context.Context, discover func(context.Context, linux.CurrentNamespacePID)) error {
		discover(ctx, 1)
		r.DiscoverProcess(ctx, linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
		discover(ctx, 42)
		discover(ctx, 2)
		return nil
	})
	stats, err := r.scanProcesses(t.Context())
	require.NoError(t, err)
	require.Zero(t, stats.DiedProcesses)
	require.Len(t, r.procs, 3)
	require.EqualValues(t, 2, r.procsGeneration.Load())
}

func TestFailedScanDoesNotRetireProcesses(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42})
	r.processScanner = scanFunc(func(context.Context, func(context.Context, linux.CurrentNamespacePID)) error {
		return errors.New("incomplete scan")
	})
	stats, err := r.scanProcesses(t.Context())
	require.Error(t, err)
	require.Zero(t, stats.DiedProcesses)
	require.Contains(t, r.procs, linux.CurrentNamespacePID(42))
	r.processScanner = scanFunc(func(context.Context, func(context.Context, linux.CurrentNamespacePID)) error { return nil })
	stats, err = r.scanProcesses(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, stats.DiedProcesses)
	require.Empty(t, r.procs)
}

func TestDiscoveryFilterIsNotADeathSignal(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	interested := true
	r.processScanner = scanFunc(func(ctx context.Context, discover func(context.Context, linux.CurrentNamespacePID)) error {
		discover(ctx, 42)
		return nil
	})
	r.processFilter = func(linux.CurrentNamespacePID) bool { return interested }
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	info := r.procs[42]
	for _, enabled := range []bool{true, false, true} {
		interested = enabled
		stats, err := r.scanProcesses(t.Context())
		require.NoError(t, err)
		require.Zero(t, stats.DiedProcesses)
		require.Same(t, info, r.procs[42])
	}
}

func TestRetiredAnalysisCannotOverwriteOrDeleteNewMappings(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	old := r.procs[42]
	stale := processAnalyzer{
		reg: r, proc: old, startTime: 100, log: r.log,
		preparedMappings: []*dso.Mapping{{Mapping: procfs.Mapping{Begin: 0x1000, End: 0x2000, Path: "old"}}},
		envs:             map[string]string{"lifetime": "old"},
	}
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
	current := r.procs[42]
	fresh := processAnalyzer{
		reg: r, proc: current, startTime: 200, log: r.log,
		preparedMappings: []*dso.Mapping{{Mapping: procfs.Mapping{Begin: 0x1000, End: 0x2000, Path: "new"}}},
		envs:             map[string]string{"lifetime": "new"},
	}
	require.NoError(t, fresh.storeBPFMaps(t.Context()))
	require.Error(t, stale.storeBPFMaps(t.Context()))
	r.dsoStorage.ReleaseMappings(t.Context(), stale.preparedMappings)
	require.False(t, r.deleteProcess(t.Context(), old, r.procsGeneration.Load()+1))
	mapping, err := r.dsoStorage.ResolveMapping(t.Context(), 42, 0x1000)
	require.NoError(t, err)
	require.Equal(t, "new", mapping.Path)
	require.Equal(t, "new", current.Env()["lifetime"])
}

func TestBootstrapIdentityInvalidatesUncommittedAnalysis(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42})
	info := r.procs[42]
	a := processAnalyzer{reg: r, proc: info, log: r.log}
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	require.Same(t, info, r.procs[42])
	require.Error(t, a.storeBPFMaps(t.Context()))
	a.startTime = 100
	require.NoError(t, a.storeBPFMaps(t.Context()))
}

func TestLateSampleCannotRecreateRetiredProcess(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	old := r.procs[42]
	require.True(t, r.deleteProcess(t.Context(), old, r.procsGeneration.Load()+1))
	r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	require.Empty(t, r.procs)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
	current := r.procs[42]
	r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	require.Same(t, current, r.procs[42])
	require.EqualValues(t, 200, current.Key().ProcessStartTime)
}

func TestSampleObservationDoesNotKeepProcessAlive(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	info := r.procs[42]
	r.processScanner = scanFunc(func(ctx context.Context, _ func(context.Context, linux.CurrentNamespacePID)) error {
		r.ObserveProcessSample(ctx, linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
		current := r.applySampleObservation(ctx, <-r.procchan)
		require.EqualValues(t, 200, current.Key().ProcessStartTime)
		require.EqualValues(t, 1, current.generation.Load())
		return nil
	})
	stats, err := r.scanProcesses(t.Context())
	require.NoError(t, err)
	require.Equal(t, 1, stats.DiedProcesses)
	require.Empty(t, r.procs)
	require.EqualValues(t, 1, info.generation.Load())
}

func TestQueuedRefreshFollowsLifetimeReplacement(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	old := r.procs[42]
	r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 300})
	r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
	require.True(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200}))
	require.Len(t, r.procchan, 1, "replacement must reuse the queued registration")
	updates := <-r.procchan
	require.Same(t, old.processRegistration, updates)
	current := r.applySampleObservation(t.Context(), updates)
	require.Same(t, current, r.procs[42])
	require.EqualValues(t, 300, current.Key().ProcessStartTime, "keep the newest queued observation")
	require.True(t, updates.refreshNeeded.Load())
	require.True(t, updates.updateQueued.Load())
}

func TestQueuedRefreshCannotReachNewRegistration(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	old := r.procs[42]
	r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 300})
	updates := <-r.procchan
	require.True(t, r.deleteProcess(t.Context(), old, r.procsGeneration.Load()+1))
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
	current := r.procs[42]
	require.NotSame(t, updates, current.processRegistration)
	require.Nil(t, r.applySampleObservation(t.Context(), updates))
	require.Same(t, current, r.procs[42])
	require.EqualValues(t, 200, current.Key().ProcessStartTime)
}

func TestFullQueuePreservesRefreshForNextScan(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		t.Run(map[bool]string{false: "discovery", true: "filtered"}[filtered], func(t *testing.T) {
			r := newLifecycleTestRegistry(t)
			r.procchan = make(chan *processRegistration, 1)
			r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 1})
			r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
			info := r.procs[42]
			r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
			require.True(t, info.refreshNeeded.Load())
			require.False(t, info.updateQueued.Load())
			<-r.procchan
			r.processFilter = func(linux.CurrentNamespacePID) bool { return !filtered }
			r.processScanner = scanFunc(func(ctx context.Context, discover func(context.Context, linux.CurrentNamespacePID)) error {
				discover(ctx, 42)
				return nil
			})
			_, err := r.scanProcesses(t.Context())
			require.NoError(t, err)
			require.Len(t, r.procchan, 1)
			current := r.applySampleObservation(t.Context(), <-r.procchan)
			require.EqualValues(t, 200, current.Key().ProcessStartTime)
			require.Same(t, info.processRegistration, current.processRegistration)
		})
	}
}

func TestWorkerRetainsFailedRefreshForDiscoveryRetry(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	// Failed procfs analysis must remain retryable on subsequent discovery.
	const pid = linux.CurrentNamespacePID(1<<31 - 1)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: pid, ProcessStartTime: 100})
	info := r.procs[pid]
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- r.runHandler(ctx, func(ctx context.Context, proc *processInfo) error {
			return r.handleProcess(ctx, proc, &WorkerConfig{})
		})
	}()
	require.Eventually(t, func() bool { return !info.updateQueued.Load() }, time.Second, time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	require.True(t, info.refreshNeeded.Load())
	require.Zero(t, info.mapsgeneration.Load())
	require.Empty(t, r.procchan, "failure must not spin in the worker")
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: pid})
	require.Len(t, r.procchan, 1)
	require.Same(t, info.processRegistration, <-r.procchan)
}

func receiveLifecycleResult[T any](t *testing.T, result <-chan T) T {
	t.Helper()
	select {
	case value := <-result:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for process lifecycle operation")
		var zero T
		return zero
	}
}

func TestWorkerProcessesNewIdentityAfterFailedAnalysis(t *testing.T) {
	for _, source := range []string{"discovery", "sample", "bootstrap"} {
		t.Run(source, func(t *testing.T) {
			r := newLifecycleTestRegistry(t)
			startTime := uint64(100)
			if source == "bootstrap" {
				startTime = 0
			}
			r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: startTime})
			var published []uint64
			r.metadataPublisher = &testMetadataPublisher{publish: func(info ProcessInfo) error {
				published = append(published, info.Key().ProcessStartTime)
				return nil
			}}
			activated := make(chan linux.CurrentNamespacePID, 1)
			r.state = &publicationBPFState{enable: func(pid linux.CurrentNamespacePID, _ *unwinder.ProcessInfo) error {
				select {
				case activated <- pid:
				default:
				}
				return nil
			}}
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			stop := sync.OnceFunc(func() {
				cancel()
				require.ErrorIs(t, receiveLifecycleResult(t, done), context.Canceled)
			})
			t.Cleanup(stop)
			go func() {
				firstAnalysis := true
				done <- r.runHandler(ctx, func(ctx context.Context, proc *processInfo) error {
					a := processAnalyzer{reg: r, proc: proc, startTime: proc.Key().ProcessStartTime, log: r.log}
					if firstAnalysis {
						firstAnalysis = false
						if source == "sample" {
							r.ObserveProcessSample(ctx, linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
							return errors.New("old process disappeared during analysis")
						}
						r.DiscoverProcess(ctx, linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
					}
					return a.storeBPFMaps(ctx)
				})
			}()
			require.EqualValues(t, 42, receiveLifecycleResult(t, activated))
			stop()
			require.NotEmpty(t, published)
			for _, identity := range published {
				require.EqualValues(t, 200, identity)
			}
		})
	}
}

func TestBlockedPublicationDoesNotBlockOtherPIDs(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 43, ProcessStartTime: 100})
	entered, release := make(chan struct{}), make(chan struct{})
	releasePublication := sync.OnceFunc(func() { close(release) })
	r.metadataPublisher = &testMetadataPublisher{publish: func(info ProcessInfo) error {
		if info.Key().Pid == 42 {
			close(entered)
			<-release
		}
		return nil
	}}
	old := r.procs[42]
	done := make(chan error, 1)
	t.Cleanup(func() {
		releasePublication()
		_ = receiveLifecycleResult(t, done)
	})
	go func() {
		defer close(done)
		a := processAnalyzer{reg: r, proc: old, startTime: 100, log: r.log}
		done <- a.storeBPFMaps(t.Context())
	}()
	receiveLifecycleResult(t, entered)
	otherDone := make(chan error, 1)
	t.Cleanup(func() {
		releasePublication()
		_ = receiveLifecycleResult(t, otherDone)
	})
	go func() {
		defer close(otherDone)
		other := processAnalyzer{reg: r, proc: r.procs[43], startTime: 100, log: r.log}
		otherDone <- other.storeBPFMaps(t.Context())
	}()
	require.NoError(t, receiveLifecycleResult(t, otherDone))
	releasePublication()
	require.NoError(t, receiveLifecycleResult(t, done))
}

func TestReuseWaitsForPublicationBeforeRetiringOldLifetime(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	old := r.procs[42]
	listener := &lifetimeListener{}
	r.listeners = []Listener{listener}
	entered, release := make(chan struct{}), make(chan struct{})
	releasePublication := sync.OnceFunc(func() { close(release) })
	r.metadataPublisher = &testMetadataPublisher{publish: func(info ProcessInfo) error {
		if info.Key().ProcessStartTime == 100 {
			close(entered)
			<-release
		}
		return nil
	}}
	published := make(chan error, 1)
	t.Cleanup(func() {
		releasePublication()
		_ = receiveLifecycleResult(t, published)
	})
	go func() {
		defer close(published)
		a := processAnalyzer{reg: r, proc: old, startTime: 100, log: r.log}
		published <- a.storeBPFMaps(t.Context())
	}()
	receiveLifecycleResult(t, entered)
	started, reused := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		releasePublication()
		receiveLifecycleResult(t, reused)
	})
	go func() {
		close(started)
		defer close(reused)
		r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200})
	}()
	receiveLifecycleResult(t, started)
	select {
	case <-reused:
		t.Fatal("process lifetime changed before publication completed")
	case <-time.After(50 * time.Millisecond):
	}
	releasePublication()
	require.NoError(t, receiveLifecycleResult(t, published))
	receiveLifecycleResult(t, reused)
	require.Equal(t, []string{"discovery", "death"}, listener.events)
	current := r.procs[42]
	a := processAnalyzer{reg: r, proc: current, startTime: 200, log: r.log}
	require.NoError(t, a.storeBPFMaps(t.Context()))
	require.Equal(t, []string{"discovery", "death", "discovery"}, listener.events)
	require.EqualValues(t, 100, old.Key().ProcessStartTime)
}

type retirementBPFState struct {
	*publicationBPFState
	stage          string
	fail           bool
	segmentRemoved bool
}

func (s *retirementBPFState) RemoveProcess(linux.CurrentNamespacePID) error {
	if s.fail && s.stage == "process" {
		return errors.New("failed to disable process")
	}
	return ebpf.ErrKeyNotExist
}

func (s *retirementBPFState) RemoveMappingLPMSegment(*unwinder.ExecutableMappingTrieKey) error {
	if s.segmentRemoved {
		return ebpf.ErrKeyNotExist
	}
	s.segmentRemoved = true
	return nil
}

func (s *retirementBPFState) RemoveMapping(*unwinder.ExecutableMappingKey) error {
	if s.fail && s.stage == "mapping" {
		return errors.New("failed to remove mapping")
	}
	return nil
}

func TestReuseRetainsCleanupStateUntilBPFRetirementSucceeds(t *testing.T) {
	for _, stage := range []string{"process", "mapping"} {
		t.Run(stage, func(t *testing.T) {
			r := newLifecycleTestRegistry(t)
			r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
			old := r.procs[42]
			listener := &lifetimeListener{}
			r.listeners = []Listener{listener}
			state := &retirementBPFState{publicationBPFState: r.state.(*publicationBPFState), stage: stage, fail: true}
			r.state = state
			if stage == "mapping" {
				mapping := &dso.Mapping{
					Mapping:   procfs.Mapping{Begin: 0x1000, End: 0x2000},
					BuildInfo: &xelf.BuildInfo{BuildID: "old"},
					DSO:       &dso.DSO{ID: 7},
				}
				old.registeredmaps[mapping.Begin] = processMap{Mapping: mappingImpl{m: mapping}, id: 1}
			}
			require.False(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200}))
			require.Same(t, old, r.procs[42], "failed cleanup must remain reachable for retry")
			require.EqualValues(t, 200, old.observedStartTime.Load())
			require.False(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 300}))
			require.False(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200}))
			require.Empty(t, listener.events)
			stale := processAnalyzer{reg: r, proc: old, startTime: 100, log: r.log}
			require.Error(t, stale.storeBPFMaps(t.Context()))
			state.fail = false
			r.processScanner = scanFunc(func(ctx context.Context, discover func(context.Context, linux.CurrentNamespacePID)) error {
				discover(ctx, 42)
				return nil
			})
			_, err := r.scanProcesses(t.Context())
			require.NoError(t, err)
			require.EqualValues(t, 300, r.procs[42].Key().ProcessStartTime)
			r.ObserveProcessSample(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
			current := r.applySampleObservation(t.Context(), <-r.procchan)
			require.EqualValues(t, 300, current.Key().ProcessStartTime)
			require.NotSame(t, old, r.procs[42])
			require.Empty(t, old.registeredmaps)
			require.Equal(t, []string{"death"}, listener.events)
		})
	}
}

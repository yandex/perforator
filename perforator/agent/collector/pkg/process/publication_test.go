package process

import (
	"context"
	"errors"
	"io"
	"maps"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xelf"
)

func TestEnvironmentRefreshClearsEmptyAndPreservesUnreadable(t *testing.T) {
	const helperArg = "process-empty-environment"
	if os.Args[len(os.Args)-1] == helperArg {
		_, err := io.Copy(io.Discard, os.Stdin)
		require.NoError(t, err)
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	child := exec.CommandContext(ctx, executable, "-test.run=^TestEnvironmentRefreshClearsEmptyAndPreservesUnreadable$", "--", helperArg)
	child.Env = []string{}
	input, err := child.StdinPipe()
	require.NoError(t, err)
	require.NoError(t, child.Start())
	t.Cleanup(func() {
		require.NoError(t, input.Close())
		require.NoError(t, child.Wait())
	})

	for _, tc := range []struct {
		name string
		pid  linux.CurrentNamespacePID
		fail bool
	}{
		{"empty", linux.CurrentNamespacePID(child.Process.Pid), false},
		{"unreadable", 1<<31 - 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newLifecycleTestRegistry(t)
			r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: tc.pid, ProcessStartTime: 100})
			info := r.procs[tc.pid]
			var published map[string]string
			r.metadataPublisher = &testMetadataPublisher{publish: func(snapshot ProcessInfo) error {
				published = maps.Clone(snapshot.Env())
				return nil
			}}
			old := map[string]string{"label": "old"}
			initial := processAnalyzer{reg: r, proc: info, startTime: 100, log: r.log, envs: old}
			require.NoError(t, initial.storeBPFMaps(t.Context()))
			require.Equal(t, old, r.GetEnvs(tc.pid))
			require.Equal(t, old, published)

			rescan := processAnalyzer{reg: r, proc: info, startTime: 100, log: r.log, config: &WorkerConfig{}}
			err := rescan.loadEnvs(t.Context())
			if tc.fail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.NoError(t, rescan.storeBPFMaps(t.Context()))
			if tc.fail {
				require.Equal(t, old, r.GetEnvs(tc.pid))
				require.Equal(t, old, published)
			} else {
				require.Empty(t, r.GetEnvs(tc.pid))
				require.Empty(t, published)
			}
		})
	}
}

type rescanBPFState struct {
	active            bool
	binaryID          uint64
	disableError      error
	activateError     error
	disableCalls      int
	activateCalls     int
	checkMappingWrite func()
}

func (s *rescanBPFState) AddProcess(linux.CurrentNamespacePID, *unwinder.ProcessInfo) error {
	s.activateCalls++
	if s.activateError != nil {
		return s.activateError
	}
	s.active = true
	return nil
}

func (s *rescanBPFState) RemoveProcess(linux.CurrentNamespacePID) error {
	s.disableCalls++
	if s.disableError != nil {
		return s.disableError
	}
	if !s.active {
		return ebpf.ErrKeyNotExist
	}
	s.active = false
	return nil
}

func (s *rescanBPFState) AddMappingLPMSegment(*unwinder.ExecutableMappingTrieKey, *unwinder.ExecutableMappingInfo) error {
	s.checkMappingWrite()
	return nil
}

func (s *rescanBPFState) RemoveMappingLPMSegment(*unwinder.ExecutableMappingTrieKey) error {
	s.checkMappingWrite()
	return nil
}

func (s *rescanBPFState) AddMapping(_ *unwinder.ExecutableMappingKey, mapping *unwinder.ExecutableMapping) error {
	s.checkMappingWrite()
	s.binaryID = mapping.BinaryId
	return nil
}

func (s *rescanBPFState) RemoveMapping(*unwinder.ExecutableMappingKey) error {
	s.checkMappingWrite()
	return nil
}

func commitRescanMapping(t *testing.T, r *ProcessRegistry, path string) error {
	t.Helper()
	return commitRescanMappings(t, r, procfs.Mapping{Begin: 0x1000, End: 0x2000, Path: path})
}

func commitRescanMappings(t *testing.T, r *ProcessRegistry, mappings ...procfs.Mapping) error {
	t.Helper()
	info := r.procs[42]
	a := processAnalyzer{
		reg: r, proc: info, startTime: info.Key().ProcessStartTime, log: r.log,
		preparedMappings: make([]*dso.Mapping, 0, len(mappings)),
	}
	for _, m := range mappings {
		mapping := &dso.Mapping{Mapping: m, BuildInfo: &xelf.BuildInfo{BuildID: m.Path}}
		r.dsoStorage.PrepareMapping(t.Context(), mapping, nil)
		a.exemappings = append(a.exemappings, mapping)
		a.preparedMappings = append(a.preparedMappings, mapping)
	}
	if len(mappings) > 0 {
		a.envs = map[string]string{"image": mappings[0].Path}
	}
	defer func() { r.dsoStorage.ReleaseMappings(t.Context(), a.preparedMappings) }()
	return a.storeBPFMaps(t.Context())
}

func TestRescanPublicationFailureAndRetry(t *testing.T) {
	for _, stage := range []string{"metadata", "activation"} {
		t.Run(stage, func(t *testing.T) {
			r := newLifecycleTestRegistry(t)
			state := &rescanBPFState{}
			state.checkMappingWrite = func() {
				require.False(t, state.active, "disable the previous activation before changing BPF mappings")
			}
			r.state = state
			r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
			info := r.procs[42]
			listener := &lifetimeListener{}
			r.listeners = []Listener{listener}
			var published string
			var publicationError error
			r.metadataPublisher = &testMetadataPublisher{publish: func(snapshot ProcessInfo) error {
				require.False(t, state.active, "process must remain disabled throughout publication")
				if publicationError != nil {
					return publicationError
				}
				mappings := snapshot.Mappings()
				require.Len(t, mappings, 1)
				require.Equal(t, state.binaryID, mappings[0].ID())
				require.Equal(t, snapshot.Env()["image"], mappings[0].Path())
				published = mappings[0].Path()
				return nil
			}}

			require.NoError(t, commitRescanMapping(t, r, "old-binary"))
			require.True(t, state.active)
			require.Equal(t, "old-binary", published)
			require.Equal(t, []string{"discovery"}, listener.events)
			oldBinaryID := state.binaryID

			failure := errors.New("rescan failed")
			if stage == "metadata" {
				publicationError = failure
			} else {
				state.activateError = failure
			}
			require.ErrorIs(t, commitRescanMapping(t, r, "new-binary"), failure)
			require.False(t, state.active)
			require.NotEqual(t, oldBinaryID, state.binaryID)
			if stage == "metadata" {
				require.Equal(t, "old-binary", published)
			} else {
				require.Equal(t, "new-binary", published)
			}
			require.Equal(t, []string{"discovery"}, listener.events)
			require.Same(t, info, r.procs[42], "publication failure must not retire the lifetime")
			mapping, err := r.dsoStorage.ResolveMapping(t.Context(), 42, 0x1000)
			require.NoError(t, err)
			require.Equal(t, "new-binary", mapping.Path)
			newBinaryID := state.binaryID
			mappingID := info.registeredmaps[0x1000].id

			publicationError = nil
			state.activateError = nil
			require.NoError(t, commitRescanMapping(t, r, "new-binary"))
			require.True(t, state.active)
			require.Equal(t, "new-binary", published)
			require.Equal(t, newBinaryID, state.binaryID)
			require.Equal(t, mappingID, info.registeredmaps[0x1000].id)
			require.Equal(t, []string{"discovery", "rescan"}, listener.events)
			require.Equal(t, 3, state.disableCalls, "initial registration and retry also tolerate a missing BPF entry")
		})
	}
}

func TestFailedRescanDisablePreservesCommittedState(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	state := &rescanBPFState{checkMappingWrite: func() {}}
	r.state = state
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	listener := &lifetimeListener{}
	r.listeners = []Listener{listener}
	var published string
	r.metadataPublisher = &testMetadataPublisher{publish: func(snapshot ProcessInfo) error {
		published = snapshot.Mappings()[0].Path()
		return nil
	}}
	require.NoError(t, commitRescanMapping(t, r, "old-binary"))
	oldBinaryID := state.binaryID
	state.checkMappingWrite = func() { t.Fatal("failed disable must not change BPF mappings") }
	state.disableError = errors.New("disable failed")
	require.ErrorIs(t, commitRescanMapping(t, r, "new-binary"), state.disableError)
	require.True(t, state.active)
	require.Equal(t, oldBinaryID, state.binaryID)
	require.Equal(t, "old-binary", published)
	require.Equal(t, "old-binary", r.procs[42].Env()["image"])
	require.Equal(t, "old-binary", r.procs[42].Mappings()[0].Path())
	mapping, err := r.dsoStorage.ResolveMapping(t.Context(), 42, 0x1000)
	require.NoError(t, err)
	require.Equal(t, "old-binary", mapping.Path)
	require.Equal(t, []string{"discovery"}, listener.events)
}

func TestGoRescanDoesNotDisableProcess(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	state := &rescanBPFState{checkMappingWrite: func() {}}
	r.state = state
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	listener := &lifetimeListener{}
	r.listeners = []Listener{listener}
	require.NoError(t, commitRescanMapping(t, r, "old-binary"))
	state.checkMappingWrite = func() { require.True(t, state.active) }
	require.NoError(t, commitRescanMapping(t, r, "new-binary"))
	require.True(t, state.active)
	require.Zero(t, state.disableCalls)
	require.Equal(t, []string{"discovery", "rescan"}, listener.events)
}

type mappingFailureBPFState struct {
	*rescanBPFState
	mappings      map[unwinder.ExecutableMappingKey]unwinder.ExecutableMapping
	segments      map[unwinder.ExecutableMappingTrieKey]unwinder.ExecutableMappingInfo
	failOperation string
	failAfter     int
	failure       error
}

func (s *mappingFailureBPFState) checkFailure(operation string) error {
	s.checkMappingWrite()
	if operation == s.failOperation {
		s.failAfter--
		if s.failAfter <= 0 {
			return s.failure
		}
	}
	return nil
}

func (s *mappingFailureBPFState) AddMapping(key *unwinder.ExecutableMappingKey, mapping *unwinder.ExecutableMapping) error {
	if err := s.checkFailure("add mapping"); err != nil {
		return err
	}
	s.mappings[*key] = *mapping
	return nil
}

func (s *mappingFailureBPFState) RemoveMapping(key *unwinder.ExecutableMappingKey) error {
	if err := s.checkFailure("remove mapping"); err != nil {
		return err
	}
	if _, ok := s.mappings[*key]; !ok {
		return ebpf.ErrKeyNotExist
	}
	delete(s.mappings, *key)
	return nil
}

func (s *mappingFailureBPFState) AddMappingLPMSegment(key *unwinder.ExecutableMappingTrieKey, mapping *unwinder.ExecutableMappingInfo) error {
	if err := s.checkFailure("add segment"); err != nil {
		return err
	}
	s.segments[*key] = *mapping
	return nil
}

func (s *mappingFailureBPFState) RemoveMappingLPMSegment(key *unwinder.ExecutableMappingTrieKey) error {
	if err := s.checkFailure("remove segment"); err != nil {
		return err
	}
	if _, ok := s.segments[*key]; !ok {
		return ebpf.ErrKeyNotExist
	}
	delete(s.segments, *key)
	return nil
}

func TestMappingSynchronizationFailureAndRetry(t *testing.T) {
	for _, operation := range []string{"add mapping", "add segment", "remove mapping", "remove segment"} {
		for _, retry := range []string{"same", "changed", "retirement", "go"} {
			t.Run(operation+"/"+retry, func(t *testing.T) {
				r := newLifecycleTestRegistry(t)
				state := &mappingFailureBPFState{
					rescanBPFState: &rescanBPFState{checkMappingWrite: func() {}},
					mappings:       make(map[unwinder.ExecutableMappingKey]unwinder.ExecutableMapping),
					segments:       make(map[unwinder.ExecutableMappingTrieKey]unwinder.ExecutableMappingInfo),
					failure:        errors.New("mapping synchronization failed"),
				}
				r.state = state
				r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
				listener := &lifetimeListener{}
				r.listeners = []Listener{listener}
				publications := 0
				if retry != "go" {
					state.checkMappingWrite = func() { require.False(t, state.active) }
					r.metadataPublisher = &testMetadataPublisher{publish: func(info ProcessInfo) error {
						publications++
						require.False(t, state.active)
						require.Len(t, info.Mappings(), len(state.mappings))
						return nil
					}}
				}
				old := []procfs.Mapping{
					{Begin: 0x1000, End: 0x4000, Path: "old-a"},
					{Begin: 0x5000, End: 0x8000, Path: "old-b"},
				}
				require.NoError(t, commitRescanMappings(t, r, old...))
				require.True(t, state.active)
				publishedBefore := publications
				state.failOperation = operation
				state.failAfter = 2
				next := []procfs.Mapping{
					{Begin: 0x1000, End: 0x4000, Path: "new-a"},
					{Begin: 0x5000, End: 0x8000, Path: "new-b"},
				}
				require.ErrorIs(t, commitRescanMappings(t, r, next...), state.failure)
				require.False(t, state.active)
				require.Equal(t, publishedBefore, publications)
				require.Equal(t, []string{"discovery"}, listener.events)
				for _, mapping := range r.procs[42].Mappings() {
					require.NoError(t, iterateMappingLPMSegments(mapping, func(address uint64, prefix uint32) error {
						key := unwinder.ExecutableMappingTrieKey{Pid: 42, Prefixlen: 32 + prefix, AddressPrefix: HostToBigEndian64(address)}
						segment, ok := state.segments[key]
						require.True(t, ok, "visible mapping must have all LPM segments")
						_, ok = state.mappings[unwinder.ExecutableMappingKey{Pid: 42, Id: segment.Id}]
						require.True(t, ok, "visible mapping must have a BPF record")
						return nil
					}))
				}
				state.failOperation = ""
				if retry == "retirement" {
					require.True(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200}))
					require.Empty(t, state.mappings)
					require.Empty(t, state.segments)
					require.Equal(t, []string{"discovery", "death"}, listener.events)
					return
				}
				if retry == "changed" {
					next = []procfs.Mapping{{Begin: 0x9000, End: 0xc000, Path: "another"}}
				}
				require.NoError(t, commitRescanMappings(t, r, next...))
				require.True(t, state.active)
				require.Len(t, state.mappings, len(next))
				require.Len(t, state.segments, 2*len(next))
				for key, mapping := range state.mappings {
					registered := r.procs[42].registeredmaps[mapping.Begin]
					require.Equal(t, registered.id, key.Id)
					require.Equal(t, registered.ID(), mapping.BinaryId)
				}
				for _, segment := range state.segments {
					_, ok := state.mappings[unwinder.ExecutableMappingKey{Pid: 42, Id: segment.Id}]
					require.True(t, ok, "no orphaned LPM segments after retry")
				}
				require.Equal(t, []string{"discovery", "rescan"}, listener.events)
			})
		}
	}
}

func TestGoMappingFailurePreservesDisableError(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	state := &mappingFailureBPFState{
		rescanBPFState: &rescanBPFState{checkMappingWrite: func() {}},
		mappings:       make(map[unwinder.ExecutableMappingKey]unwinder.ExecutableMapping),
		segments:       make(map[unwinder.ExecutableMappingTrieKey]unwinder.ExecutableMappingInfo),
		failure:        errors.New("mapping synchronization failed"),
	}
	r.state = state
	r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	listener := &lifetimeListener{}
	r.listeners = []Listener{listener}
	require.NoError(t, commitRescanMapping(t, r, "old"))
	state.failOperation = "add mapping"
	state.failAfter = 1
	state.disableError = errors.New("disable failed")
	err := commitRescanMapping(t, r, "new")
	require.ErrorIs(t, err, state.failure)
	require.ErrorIs(t, err, state.disableError)
	require.True(t, state.active, "a failed disable cannot guarantee that sampling has stopped")
	require.Equal(t, 1, state.activateCalls)
	require.Equal(t, []string{"discovery"}, listener.events)

	state.failOperation = ""
	state.disableError = nil
	require.NoError(t, commitRescanMapping(t, r, "new"))
	require.True(t, state.active)
	require.Equal(t, 2, state.activateCalls)
	require.Equal(t, []string{"discovery", "rescan"}, listener.events)
	require.Equal(t, "new", r.procs[42].Mappings()[0].Path())
	require.Len(t, state.mappings, 1)
	require.Len(t, state.segments, 1)
}

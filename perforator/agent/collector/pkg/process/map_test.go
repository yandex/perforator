package process

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type publicationListener struct {
	lifetimeListener
	publish func(ProcessInfo)
}

func (l *publicationListener) OnProcessDiscovery(_ context.Context, info ProcessInfo) {
	l.publish(info)
}

func (l *publicationListener) OnProcessRescan(_ context.Context, info ProcessInfo) {
	l.publish(info)
}

type testMetadataPublisher struct {
	lifetimeListener
	publish func(ProcessInfo) error
}

func (p *testMetadataPublisher) PublishProcess(_ context.Context, info ProcessInfo) error {
	return p.publish(info)
}

type publicationBPFState struct {
	*programstate.State
	enable func(linux.CurrentNamespacePID, *unwinder.ProcessInfo) error
}

func (s *publicationBPFState) AddProcess(pid linux.CurrentNamespacePID, info *unwinder.ProcessInfo) error {
	return s.enable(pid, info)
}

func TestProcessListenersNotifiedAfterBPFActivation(t *testing.T) {
	info := &processInfo{processRegistration: &processRegistration{pid: 42}, currentNamespaceID: 42}
	info.startTime.Store(100)
	var enabled, notified bool
	listener := &publicationListener{publish: func(snapshot ProcessInfo) {
		require.True(t, enabled, "process registration must not unblock signal producers before BPF activation")
		// Listener callbacks may synchronously read process metadata.
		_ = snapshot.Env()
		_ = snapshot.Mappings()
		notified = true
	}}
	state := &publicationBPFState{enable: func(linux.CurrentNamespacePID, *unwinder.ProcessInfo) error {
		enabled = true
		return nil
	}}
	a := processAnalyzer{
		reg:  &ProcessRegistry{state: state, listeners: []Listener{listener}},
		proc: info, startTime: 100, log: xlog.ForTest(t),
	}
	a.reg.procs = map[linux.CurrentNamespacePID]*processInfo{a.proc.Key().Pid: a.proc}
	require.NoError(t, a.storeBPFMaps(t.Context()))
	require.True(t, notified)
}

func TestProcessMetadataPublishedBeforeBPFActivation(t *testing.T) {
	for _, rescan := range []bool{false, true} {
		name := "discovery"
		if rescan {
			name = "rescan"
		}
		t.Run(name, func(t *testing.T) {
			mapping := &dso.Mapping{
				Mapping: procfs.Mapping{Begin: 0x1000, End: 0x2000, Path: "/bin/app"},
				DSO:     &dso.DSO{ID: 7},
			}
			info := &processInfo{
				processRegistration: &processRegistration{pid: 42},
				currentNamespaceID:  42,
				envs:                map[string]string{"my_env": "VALUE"},
				registeredmaps: map[procfs.Address]processMap{
					0x1000: {Mapping: mappingImpl{m: mapping}},
				},
			}
			info.startTime.Store(100)
			info.listenersNotified = rescan
			var events []string
			publisher := &testMetadataPublisher{publish: func(snapshot ProcessInfo) error {
				require.Equal(t, "VALUE", snapshot.Env()["my_env"])
				mappings := snapshot.Mappings()
				require.Len(t, mappings, 1)
				require.Equal(t, "/bin/app", mappings[0].Path())
				require.Equal(t, linux.ProcessKey{Pid: 42, ProcessStartTime: 100}, snapshot.Key())
				events = append(events, "metadata")
				return nil
			}}
			listener := &publicationListener{publish: func(snapshot ProcessInfo) {
				require.Equal(t, []string{"metadata", "bpf"}, events)
				require.Equal(t, "VALUE", snapshot.Env()["my_env"])
				require.Len(t, snapshot.Mappings(), 1)
				events = append(events, "ready")
			}}
			lifetime := &lifetimeListener{alive: rescan}
			state := &publicationBPFState{enable: func(pid linux.CurrentNamespacePID, pi *unwinder.ProcessInfo) error {
				require.Equal(t, []string{"metadata"}, events, "first BPF sample must already have env and mappings")
				require.EqualValues(t, 42, pid)
				require.EqualValues(t, 7, pi.MainBinaryId)
				events = append(events, "bpf")
				return nil
			}}
			a := processAnalyzer{
				reg:  &ProcessRegistry{state: state, listeners: []Listener{listener, lifetime}, metadataPublisher: publisher},
				proc: info, startTime: 100, log: xlog.ForTest(t),
				exemappings: []*dso.Mapping{mapping},
			}
			a.reg.procs = map[linux.CurrentNamespacePID]*processInfo{a.proc.Key().Pid: a.proc}
			require.NoError(t, a.storeBPFMaps(t.Context()))
			require.Equal(t, []string{"metadata", "bpf", "ready"}, events)
			require.Equal(t, []string{name}, lifetime.events)
		})
	}
}

func TestDeletedProcessMetadataIsNotRepublished(t *testing.T) {
	publisher := &testMetadataPublisher{publish: func(ProcessInfo) error {
		t.Fatal("deleted process metadata was republished")
		return nil
	}}
	a := processAnalyzer{
		reg:  &ProcessRegistry{metadataPublisher: publisher},
		proc: &processInfo{processRegistration: &processRegistration{pid: 42}, currentNamespaceID: 42},
	}
	a.proc.retired.Store(true)
	a.reg.procs = map[linux.CurrentNamespacePID]*processInfo{a.proc.Key().Pid: a.proc}
	require.ErrorContains(t, a.storeBPFMaps(t.Context()), "deleted before metadata publication")
}

func TestFailedProcessActivationDoesNotNotifyListeners(t *testing.T) {
	for _, stage := range []string{"metadata", "bpf"} {
		t.Run(stage, func(t *testing.T) {
			info := &processInfo{processRegistration: &processRegistration{pid: 42}, currentNamespaceID: 42}
			info.startTime.Store(100)
			failure := errors.New("activation failed")
			fail := true
			listener := &lifetimeListener{}
			publisher := &testMetadataPublisher{publish: func(ProcessInfo) error {
				if fail && stage == "metadata" {
					return failure
				}
				return nil
			}}
			state := &publicationBPFState{enable: func(linux.CurrentNamespacePID, *unwinder.ProcessInfo) error {
				require.False(t, fail && stage == "metadata", "failed publication must not enable BPF")
				if fail {
					return failure
				}
				return nil
			}}
			a := processAnalyzer{
				reg:  &ProcessRegistry{state: state, listeners: []Listener{listener}, metadataPublisher: publisher},
				proc: info, startTime: 100, log: xlog.ForTest(t),
			}
			a.reg.procs = map[linux.CurrentNamespacePID]*processInfo{a.proc.Key().Pid: a.proc}
			require.ErrorIs(t, a.storeBPFMaps(t.Context()), failure)
			require.Empty(t, listener.events)
			require.False(t, info.listenersNotified)
			fail = false
			a.reg.procs = map[linux.CurrentNamespacePID]*processInfo{a.proc.Key().Pid: a.proc}
			require.NoError(t, a.storeBPFMaps(t.Context()))
			require.Equal(t, []string{"discovery"}, listener.events)
		})
	}
}

func TestProcessMetadataPublisherNotifiedOnDeath(t *testing.T) {
	registry := newLifecycleTestRegistry(t)
	registry.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100})
	publisher := &testMetadataPublisher{}
	listener := &lifetimeListener{}
	registry.listeners = []Listener{listener}
	registry.metadataPublisher = publisher
	require.True(t, registry.deleteProcess(t.Context(), registry.procs[42], registry.procsGeneration.Load()+1))
	require.Equal(t, []string{"death"}, publisher.events)
	require.Equal(t, []string{"death"}, listener.events)
	require.Equal(t, []linux.ProcessKey{{Pid: 42, ProcessStartTime: 100}}, publisher.deaths)
	require.Equal(t, publisher.deaths, listener.deaths)
}

type lifetimeListener struct {
	events         []string
	deaths         []linux.ProcessKey
	alive          bool
	invalidRescans int
}

func (l *lifetimeListener) OnProcessDiscovery(_ context.Context, _ ProcessInfo) {
	l.events = append(l.events, "discovery")
	l.alive = true
}

func (l *lifetimeListener) OnProcessRescan(_ context.Context, _ ProcessInfo) {
	l.events = append(l.events, "rescan")
	if !l.alive {
		l.invalidRescans++
	}
}

func (l *lifetimeListener) OnProcessDeath(_ context.Context, key linux.ProcessKey) {
	l.events = append(l.events, "death")
	l.deaths = append(l.deaths, key)
	l.alive = false
}

func TestPIDReuseStartsNewListenerLifetime(t *testing.T) {
	r := newLifecycleTestRegistry(t)
	listener := &lifetimeListener{}
	r.listeners = []Listener{listener}
	publisher := &testMetadataPublisher{publish: func(ProcessInfo) error { return nil }}
	r.metadataPublisher = publisher
	require.True(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100}))
	old := r.procs[42]
	old.nextmapid.Store(17)
	publish := func(info *processInfo) error {
		a := processAnalyzer{reg: r, proc: info, startTime: info.Key().ProcessStartTime, log: r.log}
		return a.storeBPFMaps(t.Context())
	}
	require.NoError(t, publish(old))
	require.True(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 200}))
	current := r.procs[42]
	require.NotSame(t, old, current)
	require.EqualValues(t, 17, current.nextmapid.Load())
	require.EqualValues(t, 100, old.Key().ProcessStartTime)
	require.NoError(t, publish(current))
	require.Error(t, publish(old))
	require.False(t, r.DiscoverProcess(t.Context(), linux.ProcessKey{Pid: 42, ProcessStartTime: 100}))
	require.Same(t, current, r.procs[42])
	require.Equal(t, []string{"discovery", "death", "discovery"}, listener.events)
	require.Equal(t, []linux.ProcessKey{{Pid: 42, ProcessStartTime: 100}}, listener.deaths)
	require.Equal(t, listener.deaths, publisher.deaths)
	require.Zero(t, listener.invalidRescans)
}

func TestGetBinaryAttributes(t *testing.T) {
	const host = "upload-host.example"

	for _, tc := range []struct {
		name     string
		path     string
		wantPath string
		filename string
	}{
		{name: "shared library", path: "/usr/lib/libc.so.6", wantPath: "/usr/lib/libc.so.6", filename: "libc.so.6"},
		{name: "executable", path: "/app/bin/server", wantPath: "/app/bin/server", filename: "server"},
		{name: "vdso", path: "[vdso]", wantPath: "[vdso]", filename: "[vdso]"},
		{name: "deleted", path: "/usr/lib/libc.so.6 (deleted)", wantPath: "/usr/lib/libc.so.6", filename: "libc.so.6"},
		{name: "internal marker", path: "/app (deleted)/lib.so", wantPath: "/app (deleted)/lib.so", filename: "lib.so"},
		{name: "one suffix", path: "/lib.so (deleted) (deleted)", wantPath: "/lib.so (deleted)", filename: "lib.so (deleted)"},
		{name: "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := &perforatorstorage.BinaryAttributes{Upload: &perforatorstorage.BinaryUploadMetadata{Host: host}}
			if tc.wantPath != "" {
				expected.Upload.Path = tc.wantPath
				expected.Upload.Filename = tc.filename
			}
			if got := getBinaryAttributes(tc.path, host); !proto.Equal(expected, got) {
				t.Fatalf("attributes = %v, want %v", got, expected)
			}
		})
	}
}

func TestGetBinaryAttributesWithoutHost(t *testing.T) {
	expected := &perforatorstorage.BinaryAttributes{Upload: &perforatorstorage.BinaryUploadMetadata{Path: "/usr/lib/libc.so.6", Filename: "libc.so.6"}}
	if got := getBinaryAttributes("/usr/lib/libc.so.6", ""); !proto.Equal(expected, got) {
		t.Fatalf("attributes = %v, want %v", got, expected)
	}
}

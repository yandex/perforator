package deploysystem

import (
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
)

func testSnapshot() Snapshot {
	return Snapshot{Host: "node", ObservedAt: time.Now(), Pods: []Pod{
		{PodID: "a", ServiceName: "same", CgroupPath: "pods/a", Containers: []Container{{CgroupBaseName: "container-a", Name: "app"}}},
		{PodID: "b", ServiceName: "same", CgroupPath: "pods/b"},
	}}
}

func TestSnapshotIdentityAndOwnership(t *testing.T) {
	c := NewResolver(nop.Registry{})
	_, err := c.ResolvePod("a")
	require.ErrorIs(t, err, errNotReady)
	snapshot := testSnapshot()
	snapshot.Pods[0].Metadata = map[string]string{"key": "original"}
	require.NoError(t, c.Publish(snapshot))
	for _, alias := range []string{"same", "", "unknown"} {
		_, err = c.ResolvePod(alias)
		require.ErrorIs(t, err, errNotFound)
	}
	snapshot.Pods[0].Metadata["key"] = "mutated"
	snapshot.Pods[0].ServiceName = "mutated"
	snapshot.Pods[0].Containers[0].Name = "mutated"
	snapshot.Pods[0] = Pod{}
	a, err := c.ResolvePod("a")
	require.NoError(t, err)
	require.Equal(t, RelativeCgroupPath("pods/a"), a.CgroupPath)
	require.Equal(t, "same", a.ServiceName)
	require.Equal(t, "original", a.Metadata["key"])
	a.Metadata["key"] = "reader-private"
	a.ServiceName = "reader-private"
	a.Containers[0].Name = "reader-private"
	a, err = c.ResolvePod("a")
	require.NoError(t, err)
	require.Equal(t, "same", a.ServiceName)
	require.Equal(t, "original", a.Metadata["key"])
	require.Equal(t, "app", a.Containers[0].Name)
	b, err := c.ResolvePod("b")
	require.NoError(t, err)
	require.Equal(t, RelativeCgroupPath("pods/b"), b.CgroupPath)
	workloadLabels, ok := c.ResolveWorkload([]string{"container-a", "child"})
	require.True(t, ok)
	require.Equal(t, []string{"same", "app", "child"}, workloadLabels)
	require.NoError(t, c.Publish(Snapshot{ObservedAt: time.Now()}))
	_, err = c.ResolvePod("a")
	require.ErrorIs(t, err, errNotFound)
	_, ok = c.ResolveWorkload([]string{"container-a"})
	require.False(t, ok)
}

func TestPodsWithoutIdentityResolveContainers(t *testing.T) {
	r := NewResolver(nop.Registry{})
	snapshot := testSnapshot()
	for i := range snapshot.Pods {
		snapshot.Pods[i].PodID = ""
	}
	require.NoError(t, r.Publish(snapshot))
	_, err := r.ResolvePod("")
	require.ErrorIs(t, err, errNotFound)
	workloadLabels, ok := r.ResolveWorkload([]string{"container-a", "child"})
	require.True(t, ok)
	require.Equal(t, []string{"same", "app", "child"}, workloadLabels)

	// Pods without an ID still participate in snapshot validation.
	snapshot.Pods[1].CgroupPath = snapshot.Pods[0].CgroupPath
	require.Error(t, r.Publish(snapshot))
}

func TestInvalidSnapshotDoesNotReplaceResolver(t *testing.T) {
	cases := map[string]func(*Snapshot){
		"duplicate ID":        func(s *Snapshot) { s.Pods[1].PodID = "a" },
		"duplicate cgroup":    func(s *Snapshot) { s.Pods[1].CgroupPath = "pods/a" },
		"duplicate container": func(s *Snapshot) { s.Pods[1].Containers = s.Pods[0].Containers },
		"absolute path":       func(s *Snapshot) { s.Pods[0].CgroupPath = "/pods/a" },
		"empty path":          func(s *Snapshot) { s.Pods[0].CgroupPath = "" },
		"root":                func(s *Snapshot) { s.Pods[0].CgroupPath = "/" },
		"parent":              func(s *Snapshot) { s.Pods[0].CgroupPath = "pods/../a" },
		"dot":                 func(s *Snapshot) { s.Pods[0].CgroupPath = "pods/./a" },
		"current directory":   func(s *Snapshot) { s.Pods[0].CgroupPath = "." },
		"parent directory":    func(s *Snapshot) { s.Pods[0].CgroupPath = ".." },
		"leading parent":      func(s *Snapshot) { s.Pods[0].CgroupPath = "../pods/a" },
		"nul":                 func(s *Snapshot) { s.Pods[0].CgroupPath = "pods/\x00" },
		"zero timestamp":      func(s *Snapshot) { s.ObservedAt = time.Time{} },
		"future timestamp":    func(s *Snapshot) { s.ObservedAt = time.Now().Add(time.Hour) },
		"old timestamp":       func(s *Snapshot) { s.ObservedAt = time.Now().Add(-time.Hour) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := NewResolver(nop.Registry{})
			original := testSnapshot()
			require.NoError(t, c.Publish(original))
			invalid := testSnapshot()
			invalid.Pods[0].ServiceName = "invalid"
			invalid.Pods[0].Containers[0].Name = "invalid"
			invalid.Pods[0].Containers = append(invalid.Pods[0].Containers, Container{
				CgroupBaseName: "container-new", Name: "new",
			})
			invalid.Pods[1].PodID = "new"
			mutate(&invalid)
			require.Error(t, c.Publish(invalid))
			for _, expected := range original.Pods {
				w, err := c.ResolvePod(expected.PodID)
				require.NoError(t, err)
				require.Equal(t, expected, w)
			}
			_, err := c.ResolvePod("new")
			require.ErrorIs(t, err, errNotFound)
			workloadLabels, ok := c.ResolveWorkload([]string{"container-a", "child"})
			require.True(t, ok)
			require.Equal(t, []string{"same", "app", "child"}, workloadLabels)
			_, ok = c.ResolveWorkload([]string{"container-new"})
			require.False(t, ok)
		})
	}
}

func TestSnapshotAgeAndResolution(t *testing.T) {
	m := mock.NewRegistry(mock.NewRegistryOpts())
	r := NewResolver(m)
	snapshotAge, ok := m.GetFuncIntGauge("discovery.snapshot.age")
	require.True(t, ok)
	snapshot := testSnapshot()
	snapshot.ObservedAt = time.Now().Add(-2 * time.Minute)
	require.NoError(t, r.Publish(snapshot))
	// Age uses observation time, not the moment the snapshot was published.
	age := snapshotAge.Function()()
	require.GreaterOrEqual(t, age, int64(120))
	require.Error(t, r.Publish(Snapshot{}))
	require.GreaterOrEqual(t, snapshotAge.Function()(), age)
	pod, err := r.ResolvePod("a")
	require.NoError(t, err)
	require.Equal(t, snapshot.Pods[0], pod)
	_, err = r.ResolvePod("unknown")
	require.ErrorIs(t, err, errNotFound)
	workloadLabels, ok := r.ResolveWorkload([]string{"container-a"})
	require.True(t, ok)
	require.Equal(t, []string{"same", "app"}, workloadLabels)

	require.NoError(t, r.Publish(testSnapshot()))
	require.Less(t, snapshotAge.Function()(), age)
	// An empty successful observation also refreshes age and removes old pods.
	require.NoError(t, r.Publish(Snapshot{ObservedAt: time.Now()}))
	require.Less(t, snapshotAge.Function()(), age)
	_, err = r.ResolvePod("a")
	require.ErrorIs(t, err, errNotFound)
}

func TestSnapshotAgeGrowsWithoutUpdatesOrLookups(t *testing.T) {
	for _, hasSnapshot := range []bool{false, true} {
		name := "waiting for first snapshot"
		if hasSnapshot {
			name = "after successful observation"
		}
		t.Run(name, func(t *testing.T) {
			m := mock.NewRegistry(mock.NewRegistryOpts())
			r := NewResolver(m)
			if hasSnapshot {
				snapshot := testSnapshot()
				snapshot.ObservedAt = time.Now().Add(-2 * time.Minute)
				require.NoError(t, r.Publish(snapshot))
			}
			snapshotAge, ok := m.GetFuncIntGauge("discovery.snapshot.age")
			require.True(t, ok)
			age := snapshotAge.Function()()
			require.GreaterOrEqual(t, age, int64(0))
			require.Eventually(t, func() bool { return snapshotAge.Function()() > age }, 5*time.Second, 10*time.Millisecond)
			// Failed discovery does not reset the age or make the snapshot newer.
			require.Error(t, r.Publish(Snapshot{}))
			require.Greater(t, snapshotAge.Function()(), age)
			pod, err := r.ResolvePod("a")
			if hasSnapshot {
				require.NoError(t, err)
				require.Equal(t, "a", pod.PodID)
			} else {
				require.ErrorIs(t, err, errNotReady)
			}
		})
	}
}

func TestConcurrentSnapshotReaders(t *testing.T) {
	m := mock.NewRegistry(mock.NewRegistryOpts())
	c := NewResolver(m)
	snapshotAge, ok := m.GetFuncIntGauge("discovery.snapshot.age")
	require.True(t, ok)
	snapshots := []Snapshot{testSnapshot(), testSnapshot()}
	for i, generation := range []string{"first", "second"} {
		w := &snapshots[i].Pods[0]
		w.CgroupPath = RelativeCgroupPath("pods/" + generation)
		w.ServiceName = generation
		w.Containers[0].Name = generation + "-app"
	}
	// Pod lookup must not depend on the order in the source snapshot.
	snapshots[1].Pods[0], snapshots[1].Pods[1] = snapshots[1].Pods[1], snapshots[1].Pods[0]
	expected := []Pod{snapshots[0].Pods[0], snapshots[1].Pods[1]}
	require.NoError(t, c.Publish(snapshots[0]))
	start := make(chan struct{})
	var readers sync.WaitGroup
	for range 4 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for range 100 {
				if snapshotAge.Function()() < 0 {
					t.Error("negative snapshot age")
					return
				}
				w, err := c.ResolvePod("a")
				if err != nil || (!reflect.DeepEqual(w, expected[0]) && !reflect.DeepEqual(w, expected[1])) {
					t.Errorf("inconsistent pod: %+v, error: %v", w, err)
					return
				}
				w.ServiceName = "reader-private"
				w.Containers[0].Name = "reader-private"
				// This separate lookup may observe a newer generation than ResolvePod.
				workloadLabels, ok := c.ResolveWorkload([]string{"container-a", "child"})
				if !ok || (!reflect.DeepEqual(workloadLabels, []string{"first", "first-app", "child"}) &&
					!reflect.DeepEqual(workloadLabels, []string{"second", "second-app", "child"})) {
					t.Errorf("inconsistent container metadata: %v", workloadLabels)
					return
				}
			}
		}()
	}
	close(start)
	for i := range 100 {
		snapshot := snapshots[i%len(snapshots)]
		snapshot.ObservedAt = time.Now()
		require.NoError(t, c.Publish(snapshot))
	}
	readers.Wait()
}

func TestContainersUsePodServiceName(t *testing.T) {
	for _, podID := range []string{"pod", ""} {
		name := "with identity"
		if podID == "" {
			name = "without identity"
		}
		t.Run(name, func(t *testing.T) {
			r := NewResolver(nop.Registry{})
			snapshot := Snapshot{ObservedAt: time.Now(), Pods: []Pod{{
				PodID: podID, ServiceName: "service", CgroupPath: "pods/a",
				Containers: []Container{
					{CgroupBaseName: "container-app", Name: "app"},
					{CgroupBaseName: "container-sidecar", Name: "sidecar"},
				},
			}}}
			require.NoError(t, r.Publish(snapshot))
			for _, container := range snapshot.Pods[0].Containers {
				names, ok := r.ResolveWorkload([]string{container.CgroupBaseName, "child"})
				require.True(t, ok)
				require.Equal(t, []string{"service", container.Name, "child"}, names)
			}
			snapshot.Pods[0].ServiceName = "renamed"
			snapshot.ObservedAt = time.Now()
			require.NoError(t, r.Publish(snapshot))
			for _, container := range snapshot.Pods[0].Containers {
				names, ok := r.ResolveWorkload([]string{container.CgroupBaseName})
				require.True(t, ok)
				require.Equal(t, []string{"renamed", container.Name}, names)
			}
		})
	}
}

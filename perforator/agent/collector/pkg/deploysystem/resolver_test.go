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
		{Name: "a", ServiceName: "same", CgroupPath: "pods/a", Containers: []Container{{CgroupBaseName: "container-a", Name: "app"}}},
		{Name: "b", ServiceName: "same", CgroupPath: "pods/b"},
	}}
}

func requireContainer(t *testing.T, resolver *Resolver, parent, child, wantService, wantContainer string) {
	t.Helper()
	service, container, found := resolver.ResolveContainer(parent, child)
	require.True(t, found)
	require.Equal(t, wantService, service)
	require.Equal(t, wantContainer, container)
}

func TestSnapshotIdentityAndOwnership(t *testing.T) {
	c := NewResolver(nop.Registry{})
	_, err := c.FindPodsByName("a")
	require.ErrorIs(t, err, errNotReady)
	snapshot := testSnapshot()
	snapshot.Pods[0].Metadata = map[string]string{"key": "original"}
	require.NoError(t, c.Publish(snapshot))
	for _, alias := range []string{"same", "", "unknown"} {
		missing, err := c.FindPodsByName(alias)
		require.NoError(t, err)
		require.Empty(t, missing)
	}
	snapshot.Pods[0].Metadata["key"] = "mutated"
	snapshot.Pods[0].ServiceName = "mutated"
	snapshot.Pods[0].Containers[0].Name = "mutated"
	snapshot.Pods[0] = Pod{}
	a, err := c.FindPodsByName("a")
	require.NoError(t, err)
	require.Len(t, a, 1)
	require.Equal(t, RelativeCgroupPath("pods/a"), a[0].CgroupPath)
	require.Equal(t, "same", a[0].ServiceName)
	require.Equal(t, "original", a[0].Metadata["key"])
	require.Equal(t, "app", a[0].Containers[0].Name)
	again, err := c.FindPodsByName("a")
	require.NoError(t, err)
	require.Len(t, again, 1)
	require.Same(t, a[0], again[0])
	b, err := c.FindPodsByName("b")
	require.NoError(t, err)
	require.Len(t, b, 1)
	require.Equal(t, RelativeCgroupPath("pods/b"), b[0].CgroupPath)
	requireContainer(t, c, "pods/a", "container-a", "same", "app")
	next := testSnapshot()
	next.Pods[0].Metadata = map[string]string{"key": "next"}
	next.Pods[0].Containers[0].Name = "next"
	require.NoError(t, c.Publish(next))
	current, err := c.FindPodsByName("a")
	require.NoError(t, err)
	require.Len(t, current, 1)
	require.Equal(t, "next", current[0].Metadata["key"])
	require.Equal(t, "original", a[0].Metadata["key"])
	require.Equal(t, "app", a[0].Containers[0].Name)
	require.NoError(t, c.Publish(Snapshot{ObservedAt: time.Now()}))
	missing, err := c.FindPodsByName("a")
	require.NoError(t, err)
	require.Empty(t, missing)
	_, _, found := c.ResolveContainer("pods/a", "container-a")
	require.False(t, found)
}

func TestPodsWithoutNamesResolveContainers(t *testing.T) {
	r := NewResolver(nop.Registry{})
	snapshot := testSnapshot()
	for i := range snapshot.Pods {
		snapshot.Pods[i].Name = ""
	}
	require.NoError(t, r.Publish(snapshot))
	missing, err := r.FindPodsByName("")
	require.NoError(t, err)
	require.Empty(t, missing)
	requireContainer(t, r, "pods/a", "container-a", "same", "app")

	// Pods without a name still participate in snapshot validation.
	snapshot.Pods[1].CgroupPath = snapshot.Pods[0].CgroupPath
	require.Error(t, r.Publish(snapshot))
}

func TestInvalidSnapshotDoesNotReplaceResolver(t *testing.T) {
	cases := map[string]func(*Snapshot){
		"duplicate cgroup":    func(s *Snapshot) { s.Pods[1].CgroupPath = "pods/a" },
		"duplicate container": func(s *Snapshot) { s.Pods[0].Containers = append(s.Pods[0].Containers, s.Pods[0].Containers[0]) },
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
			invalid.Pods[1].Name = "new"
			mutate(&invalid)
			require.Error(t, c.Publish(invalid))
			for _, expected := range original.Pods {
				w, err := c.FindPodsByName(expected.Name)
				require.NoError(t, err)
				require.Equal(t, []*Pod{&expected}, w)
			}
			missing, err := c.FindPodsByName("new")
			require.NoError(t, err)
			require.Empty(t, missing)
			requireContainer(t, c, "pods/a", "container-a", "same", "app")
			_, _, found := c.ResolveContainer("pods/a", "container-new")
			require.False(t, found)
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
	pod, err := r.FindPodsByName("a")
	require.NoError(t, err)
	require.Equal(t, []*Pod{&snapshot.Pods[0]}, pod)
	missing, err := r.FindPodsByName("unknown")
	require.NoError(t, err)
	require.Empty(t, missing)
	requireContainer(t, r, "pods/a", "container-a", "same", "app")

	require.NoError(t, r.Publish(testSnapshot()))
	require.Less(t, snapshotAge.Function()(), age)
	// An empty successful observation also refreshes age and removes old pods.
	require.NoError(t, r.Publish(Snapshot{ObservedAt: time.Now()}))
	require.Less(t, snapshotAge.Function()(), age)
	missing, err = r.FindPodsByName("a")
	require.NoError(t, err)
	require.Empty(t, missing)
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
			pod, err := r.FindPodsByName("a")
			if hasSnapshot {
				require.NoError(t, err)
				require.Len(t, pod, 1)
				require.Equal(t, "a", pod[0].Name)
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
		w.ServiceName = generation
		w.Containers[0].Name = generation + "-app"
	}
	// Pod lookup must not depend on the order in the source snapshot.
	snapshots[1].Pods[0], snapshots[1].Pods[1] = snapshots[1].Pods[1], snapshots[1].Pods[0]
	expected := []*Pod{&snapshots[0].Pods[0], &snapshots[1].Pods[1]}
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
				w, err := c.FindPodsByName("a")
				if err != nil || len(w) != 1 || (!reflect.DeepEqual(w[0], expected[0]) && !reflect.DeepEqual(w[0], expected[1])) {
					t.Errorf("inconsistent pod: %+v, error: %v", w, err)
					return
				}
				// This separate lookup may observe a newer generation than FindPodsByName.
				service, container, found := c.ResolveContainer("pods/a", "container-a")
				if !found || (service != "first" && service != "second") || container != service+"-app" {
					t.Errorf("inconsistent container metadata: service=%q container=%q found=%t", service, container, found)
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
	for _, podName := range []string{"pod", ""} {
		name := "with name"
		if podName == "" {
			name = "without name"
		}
		t.Run(name, func(t *testing.T) {
			r := NewResolver(nop.Registry{})
			snapshot := Snapshot{ObservedAt: time.Now(), Pods: []Pod{{
				Name: podName, ServiceName: "service", CgroupPath: "pods/a",
				Containers: []Container{
					{CgroupBaseName: "container-app", Name: "app"},
					{CgroupBaseName: "container-sidecar", Name: "sidecar"},
				},
			}}}
			require.NoError(t, r.Publish(snapshot))
			for _, container := range snapshot.Pods[0].Containers {
				requireContainer(t, r, "pods/a", container.CgroupBaseName, "service", container.Name)
			}
			snapshot.Pods[0].ServiceName = "renamed"
			snapshot.ObservedAt = time.Now()
			require.NoError(t, r.Publish(snapshot))
			for _, container := range snapshot.Pods[0].Containers {
				requireContainer(t, r, "pods/a", container.CgroupBaseName, "renamed", container.Name)
			}
		})
	}
}

func TestResolveContainerScopesChildToParent(t *testing.T) {
	r := NewResolver(nop.Registry{})
	snapshot := testSnapshot()
	snapshot.Pods[1].ServiceName = "other"
	snapshot.Pods[1].Containers = []Container{{CgroupBaseName: "container-a", Name: "sidecar"}}
	require.NoError(t, r.Publish(snapshot))
	requireContainer(t, r, "pods/a", "container-a", "same", "app")
	requireContainer(t, r, "pods/b", "container-a", "other", "sidecar")
	_, _, found := r.ResolveContainer("pods/missing", "container-a")
	require.False(t, found)

	snapshot.Pods[0].Containers[0].CgroupBaseName = "container-new"
	require.NoError(t, r.Publish(snapshot))
	_, _, found = r.ResolveContainer("pods/a", "container-a")
	require.False(t, found)
	requireContainer(t, r, "pods/a", "container-new", "same", "app")
	requireContainer(t, r, "pods/b", "container-a", "other", "sidecar")
}

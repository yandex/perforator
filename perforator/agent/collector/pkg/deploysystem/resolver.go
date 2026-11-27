package deploysystem

import (
	"errors"
	"fmt"
	"maps"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var errNotReady = errors.New("pod discovery is not ready")

type containerCgroup struct {
	parentPath string
	baseName   string
}

type containerMetadata struct {
	podServiceName string
	containerName  string
}

// Resolver indexes immutable, complete snapshots. A failed refresh leaves the
// last successful snapshot available for lookups, even when it is stale.
type Resolver struct {
	mu          sync.RWMutex
	observedAt  time.Time
	podsByName  map[string][]*Pod
	containers  map[containerCgroup]containerMetadata
	snapshotAge metrics.FuncIntGauge
}

func NewResolver(m metrics.Registry) *Resolver {
	r := &Resolver{}
	createdAt := time.Now()
	// Age in whole seconds, evaluated at collection time. Before the first
	// successful observation, measure how long the resolver has been waiting.
	r.snapshotAge = m.FuncIntGauge("discovery.snapshot.age", func() int64 {
		r.mu.RLock()
		observedAt := r.observedAt
		r.mu.RUnlock()
		if observedAt.IsZero() {
			observedAt = createdAt
		}
		return int64(time.Since(observedAt).Seconds())
	})
	return r
}

func clonePod(pod Pod) *Pod {
	pod.Metadata = maps.Clone(pod.Metadata)
	pod.Containers = slices.Clone(pod.Containers)
	return &pod
}

func validateCgroupPath(relativePath RelativeCgroupPath) error {
	name := string(relativePath)
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsRune(name, 0) {
		return fmt.Errorf("invalid relative pod cgroup path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." {
			return fmt.Errorf("invalid pod cgroup path %q", name)
		}
	}
	return nil
}

func (r *Resolver) Publish(snapshot Snapshot) error {
	if snapshot.ObservedAt.IsZero() || snapshot.ObservedAt.After(time.Now()) {
		return errors.New("invalid pod observation timestamp")
	}
	podsByName := make(map[string][]*Pod)
	containers := make(map[containerCgroup]containerMetadata)
	cgroups := make(map[RelativeCgroupPath]struct{})
	for _, pod := range snapshot.Pods {
		key := pod.CgroupPath
		if err := validateCgroupPath(key); err != nil {
			return err
		}
		if _, exists := cgroups[key]; exists {
			return fmt.Errorf("duplicate pod cgroup %q", pod.CgroupPath)
		}
		cgroups[key] = struct{}{}
		if pod.Name != "" {
			podsByName[pod.Name] = append(podsByName[pod.Name], clonePod(pod))
		}
		for _, container := range pod.Containers {
			if container.CgroupBaseName != "" {
				key := containerCgroup{string(pod.CgroupPath), container.CgroupBaseName}
				if _, exists := containers[key]; exists {
					return fmt.Errorf("duplicate container cgroup %q under %q", key.baseName, key.parentPath)
				}
				containers[key] = containerMetadata{podServiceName: pod.ServiceName, containerName: container.Name}
			}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if snapshot.ObservedAt.Before(r.observedAt) {
		return errors.New("pod observation timestamp went backwards")
	}
	r.observedAt = snapshot.ObservedAt
	r.podsByName = podsByName
	r.containers = containers
	return nil
}

// FindPodsByName returns all local pods with the given name in the last snapshot.
// The returned slice and pods (including Metadata and Containers) are read-only.
// They remain valid after a new snapshot is published.
// It returns an error until the first snapshot is published.
func (r *Resolver) FindPodsByName(name string) ([]*Pod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.observedAt.IsZero() {
		return nil, errNotReady
	}
	return r.podsByName[name], nil
}

// ResolveContainer returns the pod service and container names observed for this cgroup.
func (r *Resolver) ResolveContainer(parentCgroupPath, containerCgroupBaseName string) (podServiceName, containerName string, found bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	info, found := r.containers[containerCgroup{parentCgroupPath, containerCgroupBaseName}]
	return info.podServiceName, info.containerName, found
}

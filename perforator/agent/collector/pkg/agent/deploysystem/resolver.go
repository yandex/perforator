package deploysystem

import (
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var (
	errNotReady = errors.New("pod discovery is not ready")
	errNotFound = errors.New("pod target not found")
)

type containerMetadata struct {
	serviceName   string
	containerName string
}

// Resolver indexes immutable, complete snapshots. A failed refresh leaves the
// last successful snapshot available for lookups, even when it is stale.
type Resolver struct {
	mu          sync.RWMutex
	observedAt  time.Time
	pods        map[string]Pod
	containers  map[string]containerMetadata
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

func clonePod(pod Pod) Pod {
	pod.Containers = slices.Clone(pod.Containers)
	return pod
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
	pods := make(map[string]Pod)
	containers := make(map[string]containerMetadata)
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
		if pod.PodID != "" {
			if _, ok := pods[pod.PodID]; ok {
				return fmt.Errorf("duplicate pod ID %q", pod.PodID)
			}
			pods[pod.PodID] = clonePod(pod)
		}
		for _, container := range pod.Containers {
			if container.CgroupBaseName != "" {
				if _, exists := containers[container.CgroupBaseName]; exists {
					return fmt.Errorf("duplicate container cgroup %q", container.CgroupBaseName)
				}
				containers[container.CgroupBaseName] = containerMetadata{
					serviceName:   pod.ServiceName,
					containerName: container.Name,
				}
			}
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if snapshot.ObservedAt.Before(r.observedAt) {
		return errors.New("pod observation timestamp went backwards")
	}
	r.observedAt = snapshot.ObservedAt
	r.pods = pods
	r.containers = containers
	return nil
}

func (r *Resolver) ResolvePod(id string) (Pod, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.observedAt.IsZero() {
		return Pod{}, errNotReady
	}
	pod, ok := r.pods[id]
	if !ok {
		return Pod{}, fmt.Errorf("%w: %q", errNotFound, id)
	}
	return clonePod(pod), nil
}

// ResolveWorkload replaces a recognized container cgroup name with service
// and container names, preserving the remaining cgroup path components.
// It returns false when the first component is not a known container.
func (r *Resolver) ResolveWorkload(cgroupPathComponents []string) ([]string, bool) {
	if len(cgroupPathComponents) == 0 {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if container, ok := r.containers[cgroupPathComponents[0]]; ok {
		return append([]string{container.serviceName, container.containerName}, cgroupPathComponents[1:]...), true
	}
	return nil, false
}

// Package deploysystem supplies local deployment facts to the agent.
package deploysystem

import (
	"context"
	"time"
)

// Source observes local pods. The agent calls Snapshot serially, without imposing
// a polling interval or a per-call deadline. Sources control their update cadence:
// Snapshot may wait for the next observation and must honor context cancellation.
// Sources handle initialization and individual I/O timeouts internally. Polling
// sources can use PollingSnapshotAdapter to schedule and bound their requests.
type Source interface {
	Snapshot(context.Context) (Snapshot, error)
}

type Container struct {
	CgroupBaseName string
	Name           string
}

// RelativeCgroupPath is a canonical path relative to the selected cgroup filesystem
// mount point (the freezer hierarchy on cgroup v1, or the unified hierarchy on v2).
// Resolver.Publish validates paths before accepting a snapshot.
type RelativeCgroupPath string

// Pod describes an observed pod, not an opened or pinned kernel target.
type Pod struct {
	// Name can be shared by multiple local pods.
	Name string
	// Metadata holds key-value pairs supplied by the deployment source.
	Metadata    map[string]string
	ServiceName string
	CgroupPath  RelativeCgroupPath
	Containers  []Container
}

// Snapshot is a complete observation: an empty Pods list means no pods.
// ObservedAt records when discovery began, not when a cached result was read.
type Snapshot struct {
	Host       string
	ObservedAt time.Time
	Pods       []Pod
}

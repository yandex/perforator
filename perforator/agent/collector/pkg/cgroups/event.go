package cgroups

// CgroupEventListener is an interface for the event on cgroup which is tracked,

// 	it is reopened either when IsStale() returns true (maybe not immediately)
// 	or when cgroupID of freezer cgroup has changed (for example cgroup was recreated with the same name)

type CgroupEventListener interface {
	// Called on start of cgroup tracking.
	// May not be thread-safe, will be called under mutex.
	Open(name string, cgroupID uint64) error

	// Called on finish of cgroup tracking.
	// May not be thread-safe, will be called under mutex.
	Close()

	// Determines if event needs to be reopened or not.
	// Must be thread-safe relative to itself only.
	IsStale() bool
}

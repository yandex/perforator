package linux

type processID uint32

// ProcessID in the pid namespace of the current process
type CurrentNamespacePID processID

// ProcessID in the pid namespace of the target process
type NamespacedPID processID

type PIDNamespaceInode uint64

// ProcessKey identifies one process lifetime in the agent PID namespace.
// ProcessStartTime uses the monotonic starttime recorded in BPF samples.
type ProcessKey struct {
	Pid              uint32
	ProcessStartTime uint64
}

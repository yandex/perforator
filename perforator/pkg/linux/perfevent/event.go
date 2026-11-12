package perfevent

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux/uname"
)

////////////////////////////////////////////////////////////////////////////////

func ShouldTryToEnableBranchSampling() bool {
	// The `bpf_read_branch_records` function we use (if available) in eBPF side of the agent
	// is introduced in 5.7 kernel. Without this functionality in eBPF, sampling branches is a
	// pure overhead and should be avoided.
	releaseVersion, err := uname.SystemRelease()
	if err != nil {
		return false
	}

	majorMinor := []int{0, 0}
	idx := 0

	for i := 0; i < len(releaseVersion); i++ {
		c := releaseVersion[i]
		if c >= '0' && c <= '9' {
			majorMinor[idx] = majorMinor[idx]*10 + int(c-'0')
		} else {
			idx += 1
			if idx >= len(majorMinor) {
				break
			}
		}
	}
	major, minor := majorMinor[0], majorMinor[1]

	return major > 5 || (major == 5 && minor >= 7)
}

////////////////////////////////////////////////////////////////////////////////

type Target struct {
	// ID of the process to trace.
	ProcessID *int

	// FD of the cgroup to trace.
	CgroupFD *int

	// If set, trace whole system.
	WholeSystem bool
}

type eventTarget struct {
	t   Target
	cpu int
}

type Options struct {
	// Type of the perf event.
	Type Type

	// Event sampling rate.
	SampleRate *uint64

	// Number of events per second, HZ.
	// The kernel will try to select sampling rate to match the requested frequency.
	Frequency *uint64

	// Pin events on the CPU.
	Pinned bool

	// Create perf event enabled by default.
	Enable bool

	// Enable branch sampling (PERF_SAMPLE_BRANCH_STACK)
	SampleBranchStack bool
}

type eventOptions struct {
	o             Options
	groupLeaderFD int
}

type handle struct {
	fd *os.File
	id ID
}

// Close this perf event. Stop generating samples, detach attached eBPF programs.
func (h *handle) close() error {
	return h.fd.Close()
}

// Start generating events by this perf event.
// New events by default are enabled.
func (h *handle) enable() error {
	return unix.IoctlSetInt(int(h.fd.Fd()), unix.PERF_EVENT_IOC_ENABLE, 1)
}

// Stop generating events by this perf event.
func (h *handle) disable() error {
	return unix.IoctlSetInt(int(h.fd.Fd()), unix.PERF_EVENT_IOC_DISABLE, 1)
}

// Attach existing eBPF program to the perf event.
func (h *handle) attachBPF(bpfFD int) error {
	return unix.IoctlSetInt(int(h.fd.Fd()), unix.PERF_EVENT_IOC_SET_BPF, bpfFD)
}

func getPerfEventID(fd *os.File) (id ID, err error) {
	var rawID uint64
	_, _, errno := unix.Syscall(syscall.SYS_IOCTL, fd.Fd(), unix.PERF_EVENT_IOC_ID, uintptr(unsafe.Pointer(&rawID)))
	if errno != 0 {
		err = errno
		return
	}
	return ID(rawID), nil
}

func newHandle(target *eventTarget, options *eventOptions) (*handle, error) {
	attr := makePerfEventAttr(options)

	h, err := newHandleFromAttr(attr, target, options.groupLeaderFD)
	if err != nil {
		return nil, fmt.Errorf("failed to create perf event: %w", err)
	}
	return h, nil
}

func newHandleFromAttr(attr *unix.PerfEventAttr, target *eventTarget, groupFD int) (hndl *handle, err error) {
	flags := 0
	flags |= unix.PERF_FLAG_FD_CLOEXEC

	pid := -1
	if target.t.ProcessID != nil {
		pid = *target.t.ProcessID
	} else if target.t.CgroupFD != nil {
		pid = *target.t.CgroupFD
		flags |= unix.PERF_FLAG_PID_CGROUP
	}

	cpu := target.cpu

	fd, err := unix.PerfEventOpen(attr, pid, cpu, groupFD, flags)
	if err != nil {
		return nil, fmt.Errorf("syscall perf_event_open(attr=%+v, pid=%d, cpu=%d, group_fd=%d, flags=%d) failed: %w", attr, pid, cpu, groupFD, flags, err)
	}

	file := os.NewFile(uintptr(fd), fmt.Sprintf("perf_event_open/%d/%d", pid, cpu))
	defer func() {
		if err != nil {
			_ = file.Close()
		}
	}()

	id, err := getPerfEventID(file)
	if err != nil {
		return nil, fmt.Errorf("failed to get perf event id: %w", err)
	}

	return &handle{file, id}, nil
}

func makePerfEventAttr(options *eventOptions) *unix.PerfEventAttr {
	attr := &unix.PerfEventAttr{}
	attr.Size = uint32(unsafe.Sizeof(*attr))

	attr.Type = options.o.Type.Type
	attr.Config = options.o.Type.Config

	attr.Sample_type = 0 |
		unix.PERF_SAMPLE_CALLCHAIN |
		unix.PERF_SAMPLE_IP |
		unix.PERF_SAMPLE_TID

	if options.o.SampleBranchStack {
		attr.Sample_type |= unix.PERF_SAMPLE_BRANCH_STACK
		attr.Branch_sample_type = 0 |
			unix.PERF_SAMPLE_BRANCH_USER |
			unix.PERF_SAMPLE_BRANCH_ANY
	}

	if options.o.Frequency != nil {
		attr.Sample = *options.o.Frequency
		attr.Bits |= unix.PerfBitFreq
	} else if options.o.SampleRate != nil {
		attr.Sample = *options.o.SampleRate
	}

	if options.o.Pinned {
		attr.Bits |= unix.PerfBitPinned
	}

	if !options.o.Enable {
		attr.Bits |= unix.PerfBitDisabled
	}

	return attr
}

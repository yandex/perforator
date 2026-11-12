package custom_profiling_operation

import (
	"os"
	"strconv"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
	"github.com/yandex/perforator/perforator/internal/profilingevent"
	"github.com/yandex/perforator/perforator/pkg/linux/perfevent"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type lifecycleUprobe struct {
	detached bool
	closed   bool
}

func (u *lifecycleUprobe) Attach() error { return nil }
func (u *lifecycleUprobe) Detach() error {
	u.detached = true
	return nil
}
func (u *lifecycleUprobe) Close() error {
	u.closed = true
	return nil
}
func (u *lifecycleUprobe) Info() *uprobe.UprobeInfo { return nil }

func TestDisableEventSourcesKeepsUprobeUntilDrain(t *testing.T) {
	u := &lifecycleUprobe{}
	c := &operationController{uprobes: []profiler.Uprobe{u}}
	require.NoError(t, c.disableEventSources())
	require.True(t, u.detached)
	require.False(t, u.closed)
}

func TestDisableEventSourcesKeepsPerfEventIDs(t *testing.T) {
	if os.Getenv("PERFORATOR_RUN_PRIVILEGED_TESTS") != "1" {
		t.Skip("requires Linux perf_event access; set PERFORATOR_RUN_PRIVILEGED_TESTS=1")
	}
	manager, err := perfevent.NewManager(xlog.ForTest(t).Logger(), &nop.Registry{})
	require.NoError(t, err)
	pid := os.Getpid()
	frequency := uint64(99)
	event, err := profiler.NewPerfEventManager(nil, manager).Open(
		&perfevent.Target{ProcessID: &pid},
		&perfevent.Options{Type: *profilingevent.CPUClock, Frequency: &frequency, Enable: true},
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, event.Close()) })
	// Bundle intentionally exposes no event-ID enumeration. Read the kernel IDs
	// from this process's perf fds and select only the bundle under test.
	fds, err := os.ReadDir("/proc/self/fd")
	require.NoError(t, err)
	var ids []perfevent.ID
	for _, entry := range fds {
		fd, err := strconv.Atoi(entry.Name())
		require.NoError(t, err)
		var id uint64
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), unix.PERF_EVENT_IOC_ID, uintptr(unsafe.Pointer(&id)))
		if errno == 0 && event.ContainsEvent(perfevent.ID(id)) {
			ids = append(ids, perfevent.ID(id))
		}
	}
	require.NotEmpty(t, ids)
	c := &operationController{perfEvents: []*profiler.PerfEvent{event}}
	require.NoError(t, c.disableEventSources())
	for _, id := range ids {
		require.True(t, event.ContainsEvent(id), "buffered sample must still match its operation")
		require.NotNil(t, manager.GetEvent(id), "buffered sample must still resolve its event")
	}
}

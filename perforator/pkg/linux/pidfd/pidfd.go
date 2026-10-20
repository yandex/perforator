package pidfd

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

type FD struct {
	mu sync.RWMutex
	fd int
}

func Open(pid linux.CurrentNamespacePID) (*FD, error) {
	flags := 0
	fd, err := unix.PidfdOpen(int(pid), flags)
	if err != nil {
		return nil, err
	}
	return &FD{fd: fd}, nil
}

// OpenForProcess returns an owned pidfd after checking the expected process identity.
func OpenForProcess(key linux.ProcessKey) (*FD, error) {
	fd, err := Open(key.Pid)
	if err != nil {
		return nil, err
	}
	stat, err := procfs.Open(key.Pid).Stat()
	// Linux reports /proc starttime in USER_HZ (100 Hz on supported architectures).
	if err == nil && key.ProcessStartTime != 0 && stat.StartTimeTicks != key.ProcessStartTime/10_000_000 {
		err = fmt.Errorf("process lifetime does not match sample")
	}
	if err == nil {
		err = fd.CheckAlive()
	}
	if err != nil {
		_ = fd.Close()
		return nil, err
	}
	return fd, nil
}

func (fd *FD) Close() error {
	fd.mu.Lock()
	defer fd.mu.Unlock()
	if fd.fd < 0 {
		return nil
	}
	raw := fd.fd
	fd.fd = -1
	return unix.Close(raw)
}

func (fd *FD) SendSignal(sig unix.Signal) error {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	if fd.fd < 0 {
		return os.ErrClosed
	}
	flags := 0
	var siginfo *unix.Siginfo
	return unix.PidfdSendSignal(fd.fd, sig, siginfo, flags)
}

// CheckAlive returns nil while the process has not completed, os.ErrProcessDone
// after all threads exit, or os.ErrClosed after Close. Other poll errors are returned.
func (fd *FD) CheckAlive() error {
	fd.mu.RLock()
	defer fd.mu.RUnlock()
	if fd.fd < 0 {
		return os.ErrClosed
	}
	fds := []unix.PollFd{{Fd: int32(fd.fd), Events: unix.POLLIN}}
	for {
		_, err := unix.Poll(fds, 0)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return err
		}
		if fds[0].Revents != 0 {
			return os.ErrProcessDone
		}
		return nil
	}
}

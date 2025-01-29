package pidfd

import (
	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

type FD struct {
	fd int
}

func Open(pid linux.ProcessID) (*FD, error) {
	flags := 0
	fd, err := unix.PidfdOpen(int(pid), flags)
	if err != nil {
		return nil, err
	}
	return &FD{fd: fd}, nil
}

func (fd *FD) Close() error {
	return unix.Close(fd.fd)
}

func (fd *FD) SendSignal(sig unix.Signal) error {
	flags := 0
	var siginfo *unix.Siginfo
	return unix.PidfdSendSignal(fd.fd, sig, siginfo, flags)
}

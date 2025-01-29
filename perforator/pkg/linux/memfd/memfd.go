package memfd

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

func NewFile(debugname string) (*os.File, error) {
	fd, err := unix.MemfdCreate(debugname, 0)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/proc/self/fd/%d", fd)
	return os.NewFile(uintptr(fd), path), nil
}

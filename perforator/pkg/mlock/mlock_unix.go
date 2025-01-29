//go:build unix

package mlock

import (
	"syscall"

	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

func LockExecutableMappings() error {
	return procfs.Self().ListMappings(func(m *procfs.Mapping) error {
		if m.Permissions&procfs.MappingPermissionRXP != procfs.MappingPermissionRXP {
			return nil
		}

		return mlock(uintptr(m.Begin), uintptr(m.End))
	})
}

func mlock(from, to uintptr) error {
	_, _, errno := syscall.Syscall(syscall.SYS_MLOCK, from, to-from, 0)
	if errno != 0 {
		return errno
	}
	return nil
}

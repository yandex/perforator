package vdso

import (
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux/memfd"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

const (
	vdsoMappingName = "[vdso]"
)

// `Virtual DSO mappings' are the mappings prepared by the kernel
// that are baked with kernel's built-in virtual ELFs.
// See man 7 vdso, for example.
//
// Format of the special mappings:
// 7ffe14b51000-7ffe14b52000 r-xp 00000000 00:00 0                          [vdso]
// ffffffffff600000-ffffffffff601000 --xp 00000000 00:00 0                  [vsyscall]
func IsVDSOMapping(m *procfs.Mapping) bool {
	if m.Permissions&procfs.MappingPermissionExecutable == 0 {
		return false
	}

	if m.Inode.ID != 0 ||
		m.Device.Maj != 0 ||
		m.Device.Min != 0 ||
		len(m.Path) == 0 {
		return false
	}

	return m.Path[0] == '[' && m.Path[len(m.Path)-1] == ']'
}

// Only [vdso] mapping can be analyzed.
// [vsyscall] is not available for the userspace, [vvar] is not executable.
func IsUnsymbolizableVDSOMapping(m *procfs.Mapping) bool {
	return IsVDSOMapping(m) && m.Path != vdsoMappingName
}

// Load acts like `func Load() ([]byte, error)`.
var LoadVDSO = sync.OnceValues(func() ([]byte, error) {
	var mapping *procfs.Mapping

	err := procfs.Self().ListMappings(func(m *procfs.Mapping) error {
		if !IsVDSOMapping(m) {
			return nil
		}

		if m.Path == vdsoMappingName {
			mapping = m
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if mapping == nil {
		return nil, fmt.Errorf("failed to locate vdso in /proc/self/maps")
	}

	// Seems that it is not possible to access VDSO via unsafe.Pointer directly.
	// (*byte)(unsafe.Pointer(uintptr(mapping.Begin))) fails with
	// `unsafeptr: possible misuse of unsafe.Pointer`
	mem, err := os.Open("/proc/self/mem")
	if err != nil {
		return nil, fmt.Errorf("failed to open /proc/self/mem: %w", err)
	}
	defer func() { _ = mem.Close() }()

	r := io.NewSectionReader(mem, int64(mapping.Begin), int64(mapping.End-mapping.Begin))
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return buf, nil
})

// GetVDSOFile acts like `func GetVDSOFile() (*os.File, error)`.
var getVDSOFile = sync.OnceValues(func() (f *os.File, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("failed to prepare VDSO temporary file: %w", err)
		}
	}()

	buf, err := LoadVDSO()
	if err != nil {
		return nil, fmt.Errorf("failed to load VDSO: %w", err)
	}

	f, err = memfd.NewFile("vdso")
	if err != nil {
		return nil, fmt.Errorf("failed to create memfd: %w", err)
	}

	_, err = f.Write(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to write VDSO contents to the memfd file: %w", err)
	}

	_, err = f.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to seek to the beginning of the VDSO memfd file: %w", err)
	}

	return f, nil
})

func OpenVDSO() (*os.File, error) {
	f, err := getVDSOFile()
	if err != nil {
		return nil, err
	}

	fd, err := unix.Open(fmt.Sprintf("/proc/self/fd/%d", f.Fd()), 0, 0)
	if err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), fmt.Sprintf("/proc/self/fd/%d", fd)), nil
}

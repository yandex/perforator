package cgroups

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"golang.org/x/sys/unix"
)

type CgroupVersion int

const (
	CgroupV1 CgroupVersion = 1
	CgroupV2 CgroupVersion = 2
)

func (v CgroupVersion) String() string {
	switch v {
	case CgroupV1:
		return "V1"
	case CgroupV2:
		return "V2"
	default:
		return fmt.Sprintf("unknown (%d)", v)
	}
}

type cgroupFS struct {
	prefix  string
	version CgroupVersion
}

func tryHint(hint *CgroupHint) (*cgroupFS, error) {
	var stats unix.Statfs_t
	err := unix.Statfs(hint.MountPoint, &stats)
	if err != nil {
		return nil, fmt.Errorf("statfs failed: %w", err)
	}
	var expectedTypeName string
	var expectedType int64

	switch hint.Version {
	case CgroupV1:
		expectedTypeName = "CGROUP_SUPER_MAGIC"
		expectedType = unix.CGROUP_SUPER_MAGIC
	case CgroupV2:
		expectedTypeName = "CGROUP2_SUPER_MAGIC"
		expectedType = unix.CGROUP2_SUPER_MAGIC
	default:
		return nil, fmt.Errorf("unknown cgroup version: %d", hint.Version)
	}
	if stats.Type != expectedType {
		return nil, fmt.Errorf("type mismatch: expected %s (%d), got %d", expectedTypeName, expectedType, stats.Type)
	}
	return &cgroupFS{
		prefix:  hint.MountPoint,
		version: hint.Version,
	}, nil
}

func findCgroups(hints *CgroupHints) (*cgroupFS, error) {
	var candidates []CgroupHint
	if hints != nil {
		if hints.Strong != nil {
			fs, err := tryHint(hints.Strong)
			if err != nil {
				return nil, fmt.Errorf("failed to use strong hint %+v: %w", hints.Strong, err)
			}
			return fs, nil
		}
		candidates = hints.Weak
	}
	candidates = append(
		candidates,
		CgroupHint{
			Version:    CgroupV1,
			MountPoint: "/sys/fs/cgroup/freezer",
		},
		CgroupHint{
			Version:    CgroupV2,
			MountPoint: "/sys/fs/cgroup/unified",
		},
		CgroupHint{
			Version:    CgroupV2,
			MountPoint: "/sys/fs/cgroup",
		},
	)
	var errs []error
	for _, candidate := range candidates {
		fs, err := tryHint(&candidate)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to use weak hint %+v: %w", candidate, err))
			continue
		}
		return fs, nil
	}
	return nil, fmt.Errorf("failed to find cgroups fs: all candidates failed: %w", errors.Join(errs...))
}

type nameCache interface {
	addCgroup(name string) (uint64, error)
	cgroupFullName(id uint64) string
	cgroupBaseName(id uint64) string
	populate() error
	cgroupVersion() CgroupVersion
}

type cgroupNameCache struct {
	fs          *cgroupFS
	updmu       sync.Mutex
	mu          sync.RWMutex
	id2baseName map[uint64]string
	id2fullName map[uint64]string
}

func newCgroupNameCache(fs *cgroupFS) (*cgroupNameCache, error) {
	m := &cgroupNameCache{
		fs:          fs,
		id2baseName: make(map[uint64]string),
		id2fullName: make(map[uint64]string),
	}
	go func() { _ = m.populate() }()
	return m, nil
}

// Fill cgroup cache map.
//
// Note that this method is inherently racy.
// cgroup map does not support actual state of the cgroups in the system because
// there can be thousands of short-lived cgroups in the production environment,
// and there is no efficient way to track them without full scan before Linux 5.7 with PERF_RECORD_CGROUP.
// So we provide slightly stale view of the system, and the user should use this cache to
// resolve names of long-lived cgroups (for example, top-level porto containers).
func (m *cgroupNameCache) populate() error {
	m.updmu.Lock()
	defer m.updmu.Unlock()

	return filepath.WalkDir(m.fs.prefix, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		name, err := filepath.Rel(m.fs.prefix, path)
		if err != nil {
			return err
		}

		_, err = m.addCgroup(name)
		return err
	})
}

func GetCgroupID(cgroupPath string) (uint64, error) {
	parent := filepath.Dir(cgroupPath)

	dir, err := os.Open(parent)
	if err != nil {
		return ^uint64(0), err
	}
	defer dir.Close()

	handle, _, err := unix.NameToHandleAt(int(dir.Fd()), cgroupPath, 0)
	if err != nil {
		return ^uint64(0), err
	}

	if handle.Size() != 8 {
		return ^uint64(0), fmt.Errorf("failed to get cgroup %s file handle, expected size of 8 bytes, got %d", cgroupPath, handle.Size())
	}

	var id uint64
	err = binary.Read(bytes.NewBuffer(handle.Bytes()), binary.LittleEndian, &id)
	if err != nil {
		return ^uint64(0), fmt.Errorf("failed to parse cgroup file handle: %w", err)
	}

	// That piece of shit is really undocumented.
	// ID of the cgroup in the kernel is ((inode generation) << 32 | inode & 0xffffffff).
	// See struct kernfs_node & struct cgroup.
	//
	// Inode numbers are frequently reused, so we should use inode generation.
	return id, nil
}

func (m *cgroupNameCache) addCgroup(name string) (uint64, error) {
	cgpath := filepath.Join(m.fs.prefix, name)
	id, err := GetCgroupID(cgpath)
	if err != nil {
		return ^uint64(0), err
	}

	m.mu.Lock()
	m.id2baseName[id] = filepath.Base(cgpath)
	m.id2fullName[id] = name
	m.mu.Unlock()

	return id, nil
}

func (m *cgroupNameCache) cgroupFullName(id uint64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.id2fullName[id]
}

// Try to find cgroup name by id.
func (m *cgroupNameCache) cgroupBaseName(id uint64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.id2baseName[id]
}

func (m *cgroupNameCache) cgroupVersion() CgroupVersion {
	return m.fs.version
}

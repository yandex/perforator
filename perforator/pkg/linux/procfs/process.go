package procfs

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

////////////////////////////////////////////////////////////////////////////////

func Process(pid linux.ProcessID) *process {
	return FS().Process(pid)
}

func Self() *process {
	return FS().Self()
}

////////////////////////////////////////////////////////////////////////////////

func (f *procfs) Self() *process {
	return &process{fs: f.fs, self: true}
}

func (f *procfs) Process(pid linux.ProcessID) *process {
	return &process{fs: f.fs, pid: pid}
}

////////////////////////////////////////////////////////////////////////////////

type Address = uint64

type Device struct {
	Maj uint32
	Min uint32
}

func (d Device) Mkdev() uint64 {
	return unix.Mkdev(d.Maj, d.Min)
}

type Inode struct {
	// Index of the inode.
	ID uint64
	// Always zero for mappings from /proc/pid/maps.
	Gen uint32
}

type Mapping struct {
	// First address covered by the mapping
	// in the virtual address space of the process.
	Begin Address
	// One-past-the-end address covered by the mapping
	// in the virtual address space of the process.
	End Address
	// A file the mapping is backed by.
	// For virtual file-like mappings the path can be artifactory like [vdso].
	Permissions MappingPermissions
	// Device of the file.
	Device Device
	// Inode of the file.
	Inode Inode
	// Offset from the beginning of the file to the beginning of the mapping.
	Offset int64
	// Path
	Path string
}

type MappingPermissions int

const (
	MappingPermissionNone       MappingPermissions = 0b00000000
	MappingPermissionPrivate    MappingPermissions = 0b00000001
	MappingPermissionShared     MappingPermissions = 0b00000010
	MappingPermissionExecutable MappingPermissions = 0b00000100
	MappingPermissionWriteable  MappingPermissions = 0b00001000
	MappingPermissionReadable   MappingPermissions = 0b00010000

	MappingPermissionRXP MappingPermissions = MappingPermissionReadable | MappingPermissionExecutable | MappingPermissionPrivate
	MappingPermissionRXS MappingPermissions = MappingPermissionReadable | MappingPermissionExecutable | MappingPermissionShared
	MappingPermissionRWP MappingPermissions = MappingPermissionReadable | MappingPermissionWriteable | MappingPermissionPrivate
)

////////////////////////////////////////////////////////////////////////////////

type process struct {
	fs   fs.FS
	pid  linux.ProcessID
	self bool
}

func (p *process) child(name string) string {
	var pid string
	if p.self {
		pid = "self"
	} else {
		pid = fmt.Sprint(p.pid)
	}
	return fmt.Sprintf("%s/%s", pid, name)
}

func (p *process) ListMappings(callback func(m *Mapping) error) error {
	path := p.child("maps")

	f, err := p.fs.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	s := bufio.NewScanner(bufio.NewReader(f))
	for s.Scan() {
		var mapping Mapping
		err = ParseProcessMapping(&mapping, s.Bytes(), &path)
		if err != nil {
			return err
		}

		err = callback(&mapping)
		if err != nil {
			return err
		}
	}
	if err := s.Err(); err != nil {
		return err
	}

	return nil
}

func parseEnvs(r io.Reader) (map[string]string, error) {
	s := bufio.NewScanner(r)
	s.Split(splitByNull)
	res := make(map[string]string)
	for s.Scan() {
		line := s.Text()

		if line == "" {
			continue
		}

		// Env value can contain '='.
		values := strings.SplitN(line, "=", 2)
		if len(values) != 2 {
			return nil, fmt.Errorf("failed to parse line %q", s.Text())
		}
		res[values[0]] = values[1]
	}

	return res, s.Err()
}

func (p *process) ListEnvs() (map[string]string, error) {
	path := p.child("environ")
	f, err := p.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()
	return parseEnvs(bufio.NewReader(f))
}

func (p *process) GetNamespaces() *namespaces {
	return &namespaces{p}
}

// GetNamespacedPID resolved process PID in the innermost pid namespace it is member of.
// For example, if a process is the initial process in a container, this function typically
// returns 1. If process is not nested in any pid namespaces, its pid will be returned.
//
// Warning: GetNamespacedPID works by parsing text in status file. Current implementation likely
// can be deceived into returning wrong result, e.g. if process is named "\nNSpid: 42",
// this function will return 42. It should not be used for security sensitive checks
// until this concern is verified.
func (p *process) GetNamespacedPID() (linux.ProcessID, error) {
	path := p.child("status")
	statusF, err := p.fs.Open(path)
	if err != nil {
		return 0, fmt.Errorf("failed to open process status: %w", err)
	}
	status, err := io.ReadAll(statusF)
	if err != nil {
		return 0, fmt.Errorf("failed to read process status: %w", err)
	}
	lines := strings.Split(string(status), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "NSpid:") {
			parts := strings.Split(line, "\t")
			innermost := parts[len(parts)-1]
			num, err := strconv.ParseUint(innermost, 10, 32)
			if err != nil {
				return 0, fmt.Errorf("failed to parse pid %q: %w", innermost, err)
			}
			return linux.ProcessID(num), nil
		}
	}
	return 0, fmt.Errorf("failed to find NSpid in process status")
}

////////////////////////////////////////////////////////////////////////////////

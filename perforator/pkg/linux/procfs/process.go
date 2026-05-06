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

func Open(pid linux.CurrentNamespacePID) *Process {
	return FS().Process(pid)
}

func Self() *Process {
	return FS().Self()
}

////////////////////////////////////////////////////////////////////////////////

func (f *procfs) Self() *Process {
	return &Process{fs: f.fs, self: true}
}

func (f *procfs) Process(pid linux.CurrentNamespacePID) *Process {
	return &Process{fs: f.fs, pid: pid}
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
	Offset uint64
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

type Process struct {
	fs   fs.FS
	pid  linux.CurrentNamespacePID
	self bool
}

func (p *Process) child(name string) string {
	var pid string
	if p.self {
		pid = "self"
	} else {
		pid = fmt.Sprint(p.pid)
	}
	return fmt.Sprintf("%s/%s", pid, name)
}

func (p *Process) ListMappings(callback func(m *Mapping) error) error {
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

func (p *Process) ListEnvs() (map[string]string, error) {
	path := p.child("environ")
	f, err := p.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()
	return parseEnvs(bufio.NewReader(f))
}

func (p *Process) GetNamespaces() *namespaces {
	return &namespaces{p}
}

func (p *Process) readStatusField(field string) (string, bool, error) {
	path := p.child("status")
	statusF, err := p.fs.Open(path)
	if err != nil {
		return "", false, fmt.Errorf("failed to open process status: %w", err)
	}
	status, err := io.ReadAll(statusF)
	if err != nil {
		return "", false, fmt.Errorf("failed to read process status: %w", err)
	}
	prefix := field + ":"
	for line := range strings.SplitSeq(string(status), "\n") {
		val, ok := strings.CutPrefix(line, prefix)
		if ok {
			return val, true, nil
		}
	}
	return "", false, nil
}

// GetNamespacedPID resolved process PID in the innermost pid namespace it is member of.
// For example, if a process is the initial process in a container, this function typically
// returns 1. If process is not nested in any pid namespaces, its pid will be returned.
//
// Warning: GetNamespacedPID works by parsing text in status file. Current implementation likely
// can be deceived into returning wrong result, e.g. if process is named "\nNSpid: 42",
// this function will return 42. It should not be used for security sensitive checks
// until this concern is verified.
func (p *Process) GetNamespacedPID() (linux.NamespacedPID, error) {
	val, ok, err := p.readStatusField("NSpid")
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("failed to find NSpid in process status")

	}
	parts := strings.Split(val, "\t")
	innermost := parts[len(parts)-1]
	num, err := strconv.ParseUint(innermost, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse pid %q: %w", innermost, err)
	}
	return linux.NamespacedPID(num), nil
}

func (p *Process) IsKthread() (bool, error) {
	val, ok, err := p.readStatusField("Kthread")
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("failed to find Kthread in process status")
	}
	val = strings.TrimSpace(val)
	switch val {
	case "0":
		return false, nil
	case "1":
		return true, nil
	default:
		return false, fmt.Errorf("unknown Kthread value %q", val)
	}
}

// GetComm returns the command name of the process
func (p *Process) GetComm() (string, error) {
	path := p.child("comm")
	f, err := p.fs.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return "", fmt.Errorf("failed to read %s: %w", path, err)
	}

	// Remove trailing newline
	return strings.TrimSuffix(string(data), "\n"), nil
}

type ProcStat struct {
	State  byte
	EnvEnd uint64
}

// See man 5 proc_pid_stat.
// Note that we subtract 1 because numbering in man is 1-based.
const stateFieldNumber = 3 - 1
const envEndStatFieldNumber = 51 - 1

func (p *Process) Stat() (*ProcStat, error) {
	path := p.child("stat")
	statF, err := p.fs.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", path, err)
	}
	defer statF.Close()

	data, err := io.ReadAll(statF)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", path, err)
	}
	fields := strings.Fields(string(data))

	if len(fields) <= max(stateFieldNumber, envEndStatFieldNumber) {
		return nil, fmt.Errorf("unexpected stat: not enough fields")
	}

	envEnv, err := strconv.ParseUint(fields[envEndStatFieldNumber], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse env_env: %w", err)
	}

	if len(fields[stateFieldNumber]) != 1 {
		return nil, fmt.Errorf("unexpected stat: state field is not a single character")
	}

	return &ProcStat{
		State:  fields[stateFieldNumber][0],
		EnvEnd: envEnv,
	}, nil
}

////////////////////////////////////////////////////////////////////////////////

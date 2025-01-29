package binary

import (
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/mountinfo"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/linux/vdso"
)

////////////////////////////////////////////////////////////////////////////////

type SealedFile interface {
	Unseal() (UnsealedFile, error)
	ID() string
}

type UnsealedFile interface {
	GetFile() *os.File
	Close() error
	Seal() (SealedFile, error)
}

////////////////////////////////////////////////////////////////////////////////

var _ UnsealedFile = (*OpenedFile)(nil)
var _ UnsealedFile = (*ProcessMappingBinary)(nil)
var _ SealedFile = (*SealedVDSOFile)(nil)
var _ SealedFile = (*SealedPath)(nil)

////////////////////////////////////////////////////////////////////////////////

type SealedMultiHandle struct {
	lock    sync.RWMutex
	handles map[string]SealedFile
	ids     []string
	idslen  int
	id      string
}

func (h *SealedMultiHandle) AddHandles(handle ...SealedFile) {
	h.lock.Lock()
	defer h.lock.Unlock()

	if h.handles == nil {
		h.handles = make(map[string]SealedFile)
	}

	for _, handle := range handle {
		id := handle.ID()
		if h.handles[id] != nil {
			continue
		}

		h.handles[id] = handle
		h.ids = append(h.ids, id)
		h.idslen += len(id)
	}

	h.updateCachedID()
}

func (h *SealedMultiHandle) updateCachedID() {
	sort.Strings(h.ids)

	var b strings.Builder
	b.Grow(h.idslen + len(h.ids) + 20)

	b.WriteString("multihandle {")
	for i, id := range h.ids {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(id)
	}
	b.WriteByte('}')

	h.id = b.String()
}

func (h *SealedMultiHandle) Unseal() (UnsealedFile, error) {
	h.lock.RLock()
	defer h.lock.RUnlock()

	var err error
	for _, hndl := range h.handles {
		f, e := hndl.Unseal()
		if e == nil {
			return f, nil
		}
		err = errors.Join(err, e)
	}

	if err == nil {
		return nil, fmt.Errorf("no file handles found")
	}
	return nil, err
}

func (h *SealedMultiHandle) ID() string {
	h.lock.RLock()
	defer h.lock.RUnlock()
	return h.id
}

////////////////////////////////////////////////////////////////////////////////

type SealedVDSOFile struct{}

func (h *SealedVDSOFile) Unseal() (UnsealedFile, error) {
	f, err := vdso.OpenVDSO()
	if err != nil {
		return nil, fmt.Errorf("failed to open vdso file: %w", err)
	}

	return &OpenedFile{f, h}, nil
}

func (h *SealedVDSOFile) ID() string {
	return "vdso"
}

////////////////////////////////////////////////////////////////////////////////

type SealedPath struct {
	path string
}

func (h *SealedPath) Unseal() (UnsealedFile, error) {
	f, err := os.Open(h.path)
	if err != nil {
		return nil, err
	}
	return &OpenedFile{f, h}, nil
}

func (h *SealedPath) ID() string {
	return "path " + h.path
}

////////////////////////////////////////////////////////////////////////////////

type FileHandle struct {
	handle unix.FileHandle
	mount  *mountinfo.MountPoint
}

func (h *FileHandle) Unseal() (UnsealedFile, error) {
	mountfd, err := h.mount.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open mountpoint: %w", err)
	}
	defer mountfd.Close()

	fd, err := unix.OpenByHandleAt(int(mountfd.Fd()), h.handle, unix.O_RDONLY)
	if err != nil {
		return nil, err
	}

	f := os.NewFile(uintptr(fd), "")
	if f == nil {
		_ = unix.Close(fd)
		return nil, fmt.Errorf("fd %d is not a valid file descriptor for os.Newfile", fd)
	}

	return &OpenedFile{f, h}, nil
}

func (h *FileHandle) ID() string {
	return "filehandle " + hex.EncodeToString(h.handle.Bytes()) + ", " + h.mount.String()
}

////////////////////////////////////////////////////////////////////////////////

type OpenedFile struct {
	file   *os.File
	sealed SealedFile
}

func (f *OpenedFile) GetFile() *os.File {
	return f.file
}

func (f *OpenedFile) Seal() (SealedFile, error) {
	return f.sealed, nil
}

func (f *OpenedFile) Close() error {
	return f.file.Close()
}

////////////////////////////////////////////////////////////////////////////////

type ProcessMappingBinary struct {
	pid     linux.ProcessID
	mapping *procfs.Mapping
	mounts  *mountinfo.Watcher

	ProcMapFilesPath string
	ProcRootFilePath string
	InodeID          uint64
	InodeGen         uint32
	mtime            time.Time

	file   *os.File
	isVDSO bool

	handles []SealedFile
	errors  []error
}

func NewProcessMappingBinary(
	pid linux.ProcessID,
	mounts *mountinfo.Watcher,
	mapping *procfs.Mapping,
) *ProcessMappingBinary {
	return &ProcessMappingBinary{
		mapping:          mapping,
		pid:              pid,
		mounts:           mounts,
		ProcMapFilesPath: fmt.Sprintf("/proc/%d/map_files/%x-%x", pid, mapping.Begin, mapping.End),
		ProcRootFilePath: fmt.Sprintf("/proc/%d/root/%s", pid, mapping.Path),
	}
}

func (p *ProcessMappingBinary) tryCalcSecondaryHandle() (SealedFile, error) {
	mappingFile, err := os.Open(p.ProcRootFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", p.ProcRootFilePath, err)
	}
	defer mappingFile.Close()

	stat := &unix.Stat_t{}
	err = unix.Fstat(int(mappingFile.Fd()), stat)
	if err != nil {
		return nil, fmt.Errorf("failed fstat on %s: %w", p.ProcRootFilePath, err)
	}

	mtime := time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)
	if stat.Ino != p.InodeID || p.mtime != mtime {
		return nil, fmt.Errorf(
			"mismatched file %s, expected inode %d, mtime %d, actual inode %d, mtime %d",
			p.ProcRootFilePath,
			p.InodeID,
			p.mtime.UnixNano(),
			stat.Ino,
			mtime.UnixNano(),
		)
	}

	return p.openFileHandle(mappingFile)
}

func (p *ProcessMappingBinary) openFileHandle(file *os.File) (SealedFile, error) {
	fileHandle, mountID, err := unix.NameToHandleAt(
		int(file.Fd()),
		"",
		unix.AT_SYMLINK_FOLLOW|unix.AT_EMPTY_PATH,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call name_to_handle_at: %w", err)
	}

	mountPoint := p.mounts.GetMountPoint(mountID)
	if mountPoint == nil {
		return nil, fmt.Errorf("unknown mount point %d", mountID)
	}

	return &FileHandle{fileHandle, mountPoint}, nil
}

func (p *ProcessMappingBinary) tryCalcHandle() {
	if handle, err := p.openFileHandle(p.file); err == nil {
		p.handles = append(p.handles, handle)
	} else {
		p.errors = append(p.errors, err)
	}

	if handle, err := p.tryCalcSecondaryHandle(); err == nil {
		p.handles = append(p.handles, handle)
	} else {
		p.errors = append(p.errors, err)
	}

	p.handles = append(p.handles, &SealedPath{p.ProcMapFilesPath})
	p.handles = append(p.handles, &SealedPath{p.ProcRootFilePath})
}

func (p *ProcessMappingBinary) Open() error {
	if vdso.IsVDSOMapping(p.mapping) {
		f, err := vdso.OpenVDSO()
		if err != nil {
			return fmt.Errorf("failed to open VDSO pseudofile: %w", err)
		}
		p.file = f
		p.isVDSO = true
		return nil
	}

	var errClose error
	file, err := os.Open(p.ProcMapFilesPath)
	if err != nil {
		return err
	}

	defer func() {
		if errClose != nil {
			_ = file.Close()
		}
	}()
	p.file = file

	stat := unix.Stat_t{}
	errClose = unix.Fstat(int(p.file.Fd()), &stat)
	if errClose != nil {
		return fmt.Errorf("failed to get mapping %s inode: %w", p.ProcMapFilesPath, err)
	}
	p.InodeID = stat.Ino
	p.mtime = time.Unix(stat.Mtim.Sec, stat.Mtim.Nsec)

	gen, err := linux.GetInodeGeneration(p.file)
	if err == nil {
		p.InodeGen = uint32(gen)
	}

	p.tryCalcHandle()
	return nil
}

func (p *ProcessMappingBinary) GetFile() *os.File {
	return p.file
}

func (p *ProcessMappingBinary) Seal() (SealedFile, error) {
	if p.isVDSO {
		return &SealedVDSOFile{}, nil
	}

	if len(p.handles) == 0 {
		return nil, errors.Join(p.errors...)
	}

	handle := &SealedMultiHandle{}
	handle.AddHandles(p.handles...)
	return handle, nil
}

func (p *ProcessMappingBinary) Close() error {
	if p.file != nil {
		return p.file.Close()
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

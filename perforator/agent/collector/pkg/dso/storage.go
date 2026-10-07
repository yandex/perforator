package dso

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	bpf "github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
	preprocessig_proto "github.com/yandex/perforator/perforator/agent/preprocessing/proto/parse"
	"github.com/yandex/perforator/perforator/pkg/disjointsegmentsets"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var (
	ErrNoSuchProcess    = errors.New("no such process")
	ErrUnknownMapping   = errors.New("address points to unknown mapping")
	ErrNoMainMapping    = errors.New("no main mapping")
	ErrNoDSOMainMapping = errors.New("no dso structure for main mapping")
	ErrNoBpfAllocation  = errors.New("no bpf allocation")
	ErrNoTLSAtOffset    = errors.New("no tls at offset")
)

type Address = procfs.Address

// Simple representation of a executable mapping.
type Mapping struct {
	procfs.Mapping

	// Unique ID of the binary. Should not be empty.
	BuildInfo *xelf.BuildInfo

	// Processed binary. May be empty.
	DSO *DSO

	// Base address. Difference between in-memory virtual address inside the mapping
	// and virtual address inside the corresponding ELF file.
	// Zero for non-PIC executables, non-zero for dynamic libraries and PIC executables.
	// See https://refspecs.linuxbase.org/elf/gabi4+/ch5.pheader.html for details.
	BaseAddress Address
}

// Resolved location inside DSO.
type Location struct {
	// Path to the DSO, relative to the process mount namespace.
	Path string

	// Inode of the DSO. Should be used to verify DSO validity.
	Inode procfs.Inode

	// Offset from the beginning of the DSO file.
	Offset procfs.Address
}

type storageMetrics struct {
	processesCount metrics.FuncIntGauge
	mappingsCount  metrics.FuncIntGauge
}

// NB: Storage operates with userspace processes, not threads.
type Storage struct {
	mutex         sync.RWMutex
	pidToMappings map[linux.CurrentNamespacePID]*processExecutableMaps
	registry      *Registry
	metrics       *storageMetrics
}

func NewStorage(l xlog.Logger, m metrics.Registry, u *bpf.BPFBinaryManager, options *preprocessig_proto.BinaryAnalysisOptions) (*Storage, error) {
	registry, err := NewRegistry(l, m, u, options)
	if err != nil {
		return nil, err
	}

	storage := &Storage{
		pidToMappings: map[linux.CurrentNamespacePID]*processExecutableMaps{},
		registry:      registry,
	}

	metrics := &storageMetrics{
		processesCount: m.WithTags(map[string]string{"kind": "current"}).FuncIntGauge(
			"alive_processes.count",
			func() int64 {
				return int64(storage.GetProcessCount())
			},
		),
		mappingsCount: m.WithTags(map[string]string{"kind": "current"}).FuncIntGauge(
			"alive_mappings.count",
			func() int64 {
				return int64(registry.getMappingCount())
			},
		),
	}
	storage.metrics = metrics
	return storage, nil
}

// Thread safe
func (d *Storage) GetProcessCount() int {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return len(d.pidToMappings)
}

// Add process to the cache.
// Thread safe
func (d *Storage) AddProcess(pid linux.CurrentNamespacePID) {
	d.ensureProcess(pid, false /*=lock*/)
}

// Remove process and it's mappings from the cache.
// Thread safe.
// Atomic relative to AddMapping
func (d *Storage) RemoveProcess(ctx context.Context, pid linux.CurrentNamespacePID) {
	maps := d.deleteAndLoadProcess(pid)
	if maps == nil {
		return
	}

	maps.lock.Lock()
	defer maps.lock.Unlock()
	for _, vmap := range maps.maps {
		d.removeMapping(ctx, &vmap)
	}
}

func (d *Storage) removeMapping(ctx context.Context, vmap *versionedMapping) {
	if vmap.BuildInfo != nil {
		d.registry.release(ctx, vmap.BuildInfo.BuildID)
	}
}

// Add mapping for the process.
// Thread safe.
// Atomic relative to RemoveProcess
func (d *Storage) AddMapping(ctx context.Context, pid linux.CurrentNamespacePID, mapping Mapping, binary binary.UnsealedFile) *DSO {
	d.PrepareMapping(ctx, &mapping, binary)

	maps := d.ensureProcess(pid, true /*=lock*/)
	defer maps.lock.Unlock()
	maps.addMappingLocked(mapping)

	return mapping.DSO
}

// PrepareMapping acquires a binary reference without changing any PID-indexed
// state. The caller must transfer it to ReplaceMappings or release it through
// ReleaseMappings exactly once.
func (d *Storage) PrepareMapping(ctx context.Context, mapping *Mapping, file binary.UnsealedFile) {
	if mapping.BuildInfo != nil {
		mapping.DSO = d.registry.register(ctx, mapping.BuildInfo, file)
	}
}

// ReleaseMappings releases prepared references that have not been transferred
// to ReplaceMappings. It does not change the process's committed mappings.
func (d *Storage) ReleaseMappings(ctx context.Context, mappings []*Mapping) {
	for _, mapping := range mappings {
		if mapping.BuildInfo != nil {
			d.registry.release(ctx, mapping.BuildInfo.BuildID)
		}
	}
}

// ReplaceMappings transfers the prepared references to the process atomically.
// Process lifecycle ordering is the caller's responsibility.
func (d *Storage) ReplaceMappings(ctx context.Context, pid linux.CurrentNamespacePID, mappings []*Mapping) {
	maps := d.ensureProcess(pid, true /*=lock*/)
	defer maps.lock.Unlock()
	for _, old := range maps.maps {
		d.removeMapping(ctx, &old)
	}
	maps.maps = nil
	for _, mapping := range mappings {
		maps.addMappingLocked(*mapping)
	}
}

// Remove unused process mappings.
func (d *Storage) Compactify(ctx context.Context, pid linux.CurrentNamespacePID) int {
	maps := d.ensureProcess(pid, false /*=lock*/)
	return maps.sortMaps(ctx)
}

// Find DSO using address.
// Thread safe.
// Returns a copy of the Mapping to avoid data races.
func (d *Storage) ResolveMapping(ctx context.Context, pid linux.CurrentNamespacePID, address procfs.Address) (Mapping, error) {
	maps := d.findProcess(pid)
	if maps == nil {
		return Mapping{}, ErrNoSuchProcess
	}

	maps.lock.RLock()
	defer maps.lock.RUnlock()

	m, err := maps.resolveAddressLocked(ctx, address)
	if err != nil {
		return Mapping{}, err
	}
	return *m, nil
}

// Find DSO+offset using address.
// Thread safe
func (d *Storage) ResolveAddress(ctx context.Context, pid linux.CurrentNamespacePID, address procfs.Address) (*Location, error) {
	mapping, err := d.ResolveMapping(ctx, pid, address)
	if err != nil {
		return nil, err
	}

	return &Location{
		Path:   mapping.Path,
		Inode:  mapping.Inode,
		Offset: uint64(mapping.Offset) + (address - mapping.Begin),
	}, nil
}

// Resolve tls variable name by variable offset.
func (d *Storage) ResolveTLSName(ctx context.Context, pid linux.CurrentNamespacePID, offset uint64) (string, error) {
	m := d.findProcess(pid)
	if m == nil {
		return "", ErrNoSuchProcess
	}

	m.lock.RLock()
	defer m.lock.RUnlock()

	mainMap := m.mainMappingLocked(ctx)
	if mainMap == nil {
		return "", ErrNoMainMapping
	}

	if mainMap.DSO == nil {
		return "", ErrNoDSOMainMapping
	}

	if mainMap.DSO.bpfAllocation == nil {
		return "", ErrNoBpfAllocation
	}

	mainMap.DSO.bpfAllocation.TLSMutex.RLock()
	defer mainMap.DSO.bpfAllocation.TLSMutex.RUnlock()

	name, ok := mainMap.DSO.bpfAllocation.TLSMap[offset]
	if !ok {
		return "", ErrNoTLSAtOffset
	}
	return name, nil
}

// Get or create process maps.
// Thread safe
func (d *Storage) ensureProcess(pid linux.CurrentNamespacePID, lock bool) *processExecutableMaps {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	maps, ok := d.pidToMappings[pid]
	if !ok {
		maps = &processExecutableMaps{s: d}
		d.pidToMappings[pid] = maps
	}
	if lock {
		maps.lock.Lock()
	}

	return maps
}

func (d *Storage) deleteAndLoadProcess(pid linux.CurrentNamespacePID) *processExecutableMaps {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	process, ok := d.pidToMappings[pid]
	if !ok {
		return nil
	}

	delete(d.pidToMappings, pid)
	return process
}

// Try to find process maps.
// Thread safe
func (d *Storage) findProcess(pid linux.CurrentNamespacePID) *processExecutableMaps {
	d.mutex.RLock()
	defer d.mutex.RUnlock()
	return d.pidToMappings[pid]
}

type versionedMapping struct {
	Mapping
	generation int
}

// SegmentBegin implements disjointsegmentsets.Item
func (m versionedMapping) SegmentBegin() uint64 {
	return m.Begin
}

// SegmentEnd implements disjointsegmentsets.Item
func (m versionedMapping) SegmentEnd() uint64 {
	return m.End
}

// GenerationNumber implements disjointsegmentsets.Item
func (m versionedMapping) GenerationNumber() int {
	return m.generation
}

type processExecutableMaps struct {
	s          *Storage
	lock       sync.RWMutex
	maps       []versionedMapping
	generation int
	sorted     bool
}

func (m *processExecutableMaps) addMappingLocked(mapping Mapping) {
	m.sorted = false
	m.maps = append(m.maps, versionedMapping{mapping, m.generation})
	m.generation++
}

func (m *processExecutableMaps) sortForReadLocked(ctx context.Context) {
	for !m.sorted {
		m.lock.RUnlock()
		m.sortMaps(ctx)
		m.lock.RLock()
	}
}

func (m *processExecutableMaps) mainMappingLocked(ctx context.Context) *Mapping {
	m.sortForReadLocked(ctx)

	if len(m.maps) == 0 {
		return nil
	}

	return &m.maps[0].Mapping
}

func (m *processExecutableMaps) resolveAddressLocked(ctx context.Context, address procfs.Address) (*Mapping, error) {
	m.sortForReadLocked(ctx)

	i := sort.Search(len(m.maps), func(i int) bool {
		return m.maps[i].Begin > address
	})

	if i == 0 || m.maps[i-1].End <= address {
		return nil, ErrUnknownMapping
	}
	return &m.maps[i-1].Mapping, nil
}

func (m *processExecutableMaps) sortMaps(ctx context.Context) int {
	m.lock.Lock()
	defer m.lock.Unlock()

	sort.Slice(m.maps, func(i, j int) bool {
		if m.maps[i].Begin == m.maps[j].Begin {
			return m.maps[i].End < m.maps[j].End
		}
		return m.maps[i].Begin < m.maps[j].Begin
	})
	m.sorted = true
	m.pruneInvalidatedMaps(ctx)
	return len(m.maps)
}

// Requires m.lock
// Requires m.sorted (m.maps should be sorted)
//
// We are required to support interface without explicit mapping removal:
// For example, using perf_event_open(2) the kernel can generate events for mmap(2) calls, but not for munmap(2).
//
// We keep generation number for each mapping and remove each mapping that overlaps with another mapping with a higher generation number.
func (m *processExecutableMaps) pruneInvalidatedMaps(ctx context.Context) {
	if !m.sorted {
		panic("broken precondition: maps should be sorted")
	}

	retained, pruned := disjointsegmentsets.Prune(m.maps)

	for _, p := range pruned {
		m.s.removeMapping(ctx, &p)
	}

	m.maps = retained
}

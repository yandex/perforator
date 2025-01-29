package dso

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	bpf "github.com/yandex/perforator/perforator/agent/collector/pkg/dso/bpf/binary"
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
)

// Simple representation of a executable mapping.
type Mapping struct {
	procfs.Mapping

	// Unique ID of the binary. Should not be empty.
	BuildInfo *xelf.BuildInfo

	// Processed binary. May be empty.
	DSO *dso

	// Base address. Difference between in-memory virtual address inside the mapping
	// and virtual address inside the corresponding ELF file.
	// Zero for non-PIC executables, non-zero for dynamic libraries and PIC executables.
	// See https://refspecs.linuxbase.org/elf/gabi4+/ch5.pheader.html for details.
	BaseAddress uint64
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

type Storage struct {
	mutex         sync.RWMutex
	pidToMappings map[linux.ProcessID]*processExecutableMaps
	registry      *Registry
	metrics       *storageMetrics
}

func NewStorage(l xlog.Logger, m metrics.Registry, u *bpf.BPFBinaryManager) (*Storage, error) {
	registry, err := NewRegistry(l, m, u)
	if err != nil {
		return nil, err
	}

	storage := &Storage{
		pidToMappings: map[linux.ProcessID]*processExecutableMaps{},
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
func (d *Storage) AddProcess(pid linux.ProcessID) {
	d.ensureProcess(pid, false /*=lock*/)
}

// Remove process and it's mappings from the cache.
// Thread safe.
// Atomic relative to AddMapping
func (d *Storage) RemoveProcess(ctx context.Context, pid linux.ProcessID) {
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
func (d *Storage) AddMapping(ctx context.Context, pid linux.ProcessID, mapping Mapping, binary binary.UnsealedFile) (*dso, error) {
	var dso *dso
	var err error

	if mapping.BuildInfo != nil {
		dso, err = d.registry.register(ctx, mapping.BuildInfo, binary)
		if err != nil {
			return nil, err
		}
		mapping.DSO = dso
	}

	maps := d.ensureProcess(pid, true /*=lock*/)
	defer maps.lock.Unlock()
	maps.addMappingLocked(mapping)

	return dso, nil
}

// Remove unused process mappings.
func (d *Storage) Compactify(ctx context.Context, pid linux.ProcessID) int {
	maps := d.ensureProcess(pid, false /*=lock*/)
	return maps.sortMaps(ctx)
}

// Find DSO using address.
// Thread safe
func (d *Storage) ResolveMapping(ctx context.Context, pid linux.ProcessID, address procfs.Address) (*Mapping, error) {
	maps := d.findProcess(pid)
	if maps == nil {
		return nil, ErrNoSuchProcess
	}

	maps.lock.RLock()
	defer maps.lock.RUnlock()

	return maps.resolveAddressLocked(ctx, address)
}

// Find DSO+offset using address.
// Thread safe
func (d *Storage) ResolveAddress(ctx context.Context, pid linux.ProcessID, address procfs.Address) (*Location, error) {
	m, err := d.ResolveMapping(ctx, pid, address)
	if err != nil {
		return nil, err
	}

	return &Location{
		Path:   m.Path,
		Inode:  m.Inode,
		Offset: uint64(m.Offset) + (address - m.Begin),
	}, nil
}

// resolve tls variable name by variable offset
func (d *Storage) ResolveTLSName(ctx context.Context, pid linux.ProcessID, offset uint64) (string, error) {
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

	return mainMap.DSO.bpfAllocation.TLSMap[offset], nil
}

// Get or create process maps.
// Thread safe
func (d *Storage) ensureProcess(pid linux.ProcessID, lock bool) *processExecutableMaps {
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

func (d *Storage) deleteAndLoadProcess(pid linux.ProcessID) *processExecutableMaps {
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
func (d *Storage) findProcess(pid linux.ProcessID) *processExecutableMaps {
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

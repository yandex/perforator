package programstate

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/cilium/ebpf"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

// BPF program state accessor
type State struct {
	// TODO: this struct is currently too smart.
	// It should have neither logger nor metrics
	logger log.Logger
	maps   *unwinder.Maps

	// protects changes of profiler config map
	configUpdateMu sync.Mutex

	unwindTablePartCount int
	unwindTablePartSpec  *ebpf.MapSpec
	partsmu              sync.RWMutex
	unwindTableParts     map[uint32]*ebpf.Map

	currentPartCount metrics.IntGauge
	currentPageCount metrics.IntGauge
}

type UnwindTableOpts struct {
	PartSpec  *ebpf.MapSpec
	PartCount int

	Logger  log.Logger
	Metrics metrics.Registry
}

func New(maps *unwinder.Maps, unwindTable *UnwindTableOpts) *State {
	s := &State{maps: maps}
	if unwindTable != nil {
		s.logger = unwindTable.Logger

		s.unwindTablePartSpec = unwindTable.PartSpec
		s.unwindTablePartCount = unwindTable.PartCount
		s.unwindTableParts = make(map[uint32]*ebpf.Map)

		s.currentPageCount = unwindTable.Metrics.IntGauge("unwind_page_table.current_pages.count")
		s.currentPartCount = unwindTable.Metrics.IntGauge("unwind_page_table.current_parts.count")
	}
	return s
}

func (s *State) Close() error {
	return s.maps.Close()
}

func memLockedBytes(fd int) (uint64, error) {
	b, err := os.ReadFile(fmt.Sprintf("/proc/self/fdinfo/%d", fd))
	if err != nil {
		return 0, err
	}

	s := bufio.NewScanner(bytes.NewBuffer(b))
	for s.Scan() {
		key, value, _ := strings.Cut(s.Text(), ":\t")
		if key == "memlock" {
			count, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return 0, err
			}
			return count, nil
		}
	}

	return 0, s.Err()
}

func (s *State) CountTotalMemLockedBytes() (uint64, error) {
	var locked uint64

	err := s.maps.ForEachMap(func(m *ebpf.Map) error {
		count, err := memLockedBytes(m.FD())
		if err != nil {
			return err
		}
		locked += count
		return nil
	})

	if err != nil {
		return 0, err
	}

	return locked, nil
}

func (s *State) UpdateConfig(conf *unwinder.ProfilerConfig) error {
	return s.maps.ProfilerConfig.Update(ptr.Uint32(0), conf, ebpf.UpdateAny)
}

// PatchConfig modifies profiler config according to the provided patcher.
// Warning: When several State objects refer to the same profiler_config map,
// PatchConfig is safe from race conditions only as long as all concurrent calls
// happen on the same State object. Otherwise, racing modification can be silently lost.
func (s *State) PatchConfig(patcher func(conf *unwinder.ProfilerConfig) error) error {
	s.configUpdateMu.Lock()
	defer s.configUpdateMu.Unlock()
	var key = ptr.Uint32(0)
	var conf unwinder.ProfilerConfig

	err := s.maps.ProfilerConfig.Lookup(key, &conf)
	if err != nil {
		return err
	}

	err = patcher(&conf)
	if err != nil {
		return err
	}

	return s.maps.ProfilerConfig.Update(key, &conf, ebpf.UpdateAny)
}

////////////////////////////////////////////////////////////////////////////////

func (s *State) AddTracedCgroup(cgroup uint64) error {
	return s.maps.TracedCgroups.Update(cgroup, uint8(0), ebpf.UpdateAny)
}

func (s *State) RemoveTracedCgroup(cgroup uint64) error {
	return s.maps.TracedCgroups.Delete(cgroup)
}

////////////////////////////////////////////////////////////////////////////////

func (s *State) AddTracedProcess(pid linux.CurrentNamespacePID) error {
	return s.maps.TracedProcesses.Update(pid, uint8(0), ebpf.UpdateAny)
}

func (s *State) RemoveTracedProcess(pid linux.CurrentNamespacePID) error {
	return s.maps.TracedProcesses.Delete(pid)
}

////////////////////////////////////////////////////////////////////////////////

func (s *State) AddProcess(pid linux.CurrentNamespacePID, info *unwinder.ProcessInfo) error {
	return s.maps.ProcessInfo.Put(&pid, info)
}

func (s *State) RemoveProcess(pid linux.CurrentNamespacePID) error {
	return s.maps.ProcessInfo.Delete(&pid)
}

func (s *State) GetProcess(pid linux.CurrentNamespacePID) (*unwinder.ProcessInfo, error) {
	var info unwinder.ProcessInfo
	err := s.maps.ProcessInfo.Lookup(&pid, &info)
	return &info, err
}

func (s *State) AddMappingLPMSegment(key *unwinder.ExecutableMappingTrieKey, value *unwinder.ExecutableMappingInfo) error {
	return s.maps.ExecutableMappingTrie.Update(key, value, ebpf.UpdateAny)
}

func (s *State) RemoveMappingLPMSegment(key *unwinder.ExecutableMappingTrieKey) error {
	return s.maps.ExecutableMappingTrie.Delete(key)
}

func (s *State) AddMapping(key *unwinder.ExecutableMappingKey, value *unwinder.ExecutableMapping) error {
	return s.maps.ExecutableMappings.Update(key, value, ebpf.UpdateAny)
}

func (s *State) RemoveMapping(key *unwinder.ExecutableMappingKey) error {
	return s.maps.ExecutableMappings.Delete(key)
}

func (s *State) AddBinaryUnwindTable(id unwinder.BinaryId, root unwinder.PageId) error {
	return s.maps.BinaryUnwindRoots.Update(&id, &root, ebpf.UpdateNoExist)
}

func (s *State) DeleteBinaryUnwindTable(id unwinder.BinaryId) error {
	return s.maps.BinaryUnwindRoots.Delete(&id)
}

func (s *State) PutProcessUnwindTable(pid linux.CurrentNamespacePID, root unwinder.PageId) error {
	// unlike binary, process unwind table may evolve over time
	// and hence needs to be updated
	return s.maps.ProcessUnwindRoots.Update(&pid, &root, ebpf.UpdateAny)
}

func (s *State) DeleteProcessUnwindTable(pid linux.CurrentNamespacePID) error {
	return s.maps.ProcessUnwindRoots.Delete(&pid)
}

func (s *State) GetMetric(metric unwinder.Metric, val []uint64) error {
	// we expect that val has suitable size
	return s.maps.Metrics.Lookup(&metric, val)
}

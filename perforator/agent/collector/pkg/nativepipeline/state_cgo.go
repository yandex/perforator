package nativepipeline

// #include <perforator/agent/collector/nativepipeline/state.h>
// #include <perforator/lib/profile/c/error.h>
// #include <stdlib.h>
import "C"

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/yandex/perforator/perforator/pkg/linux"
)

// Mapping describes an executable process address range [Begin, End).
type Mapping struct {
	Begin uint64
	End   uint64
	// Offset translates an IP to a profile binary address: IP - Begin + Offset.
	Offset uint64
	// FileOffset is the byte offset in the mapped file corresponding to Begin.
	FileOffset uint64
	Path       string
	BuildID    string
}

type StringLabel struct {
	Key   string
	Value string
}

type TLSName struct {
	Offset uint64
	Name   string
}

type ProcessSnapshot struct {
	// Key uses the agent PID namespace and the BPF process start time in
	// nanoseconds since boot. A zero start time means the identity is unknown.
	Key linux.ProcessKey
	// Version must be nonzero and increase per PID across publications,
	// removals and PID reuse for the lifetime of State.
	Version     uint64
	Mappings    []Mapping
	Environment []StringLabel
	TLSNames    []TLSName
}

type KernelSymbol struct {
	Address uint64
	Name    string
}

type CgroupName struct {
	ID       uint64
	BaseName string
	FullName string
}

type Workload struct {
	CgroupBaseName string
	ServiceName    string
	ContainerName  string
}

// CgroupSnapshot replaces all names and workloads. Version must be nonzero
// and increase across cgroup snapshots published to the same State.
type CgroupSnapshot struct {
	Version   uint64
	Names     []CgroupName
	Workloads []Workload
}

type TargetKind uint8

const (
	TargetWholeSystem TargetKind = TargetKind(C.PERFORATOR_TARGET_WHOLE_SYSTEM)
	TargetProcess     TargetKind = TargetKind(C.PERFORATOR_TARGET_PROCESS)
	TargetCgroup      TargetKind = TargetKind(C.PERFORATOR_TARGET_CGROUP)
	TargetCustom      TargetKind = TargetKind(C.PERFORATOR_TARGET_CUSTOM)
)

type TargetUprobe struct {
	BuildID     string
	Offset      uint64
	ProfileName string
	SampleKind  string
}

type Target struct {
	// ID uniquely identifies the target within a snapshot and must be nonzero.
	ID   uint64
	Kind TargetKind
	// Key is an agent-namespace PID for TargetProcess, a cgroup ID for
	// TargetCgroup, or zero for TargetWholeSystem. TargetCustom accepts
	// an agent-namespace PID, or zero to apply globally.
	Key                   uint64
	EnableSampleTime      bool
	EnableInnermostPIDNS  bool
	Labels                []StringLabel
	CreatedTimestampNanos int64
	StartTimestampNanos   int64
	EndTimestampNanos     int64
	CollectAllPerfEvents  bool
	PerfEvents            []PerfEvent
	Uprobes               []TargetUprobe
	// Pending targets accumulate samples without publishing profiles.
	Pending bool
}

// TargetSnapshot replaces all targets and base uprobes. Version must be nonzero
// and increase across target snapshots published to the same State.
type TargetSnapshot struct {
	Version     uint64
	BaseUprobes []TargetUprobe
	Targets     []Target
}

type PerfEvent struct {
	ID   uint64
	Kind string
	Unit string
}

// PerfEventSnapshot replaces all perf events. Version must be nonzero
// and increase across perf event snapshots published to the same State.
type PerfEventSnapshot struct {
	Version uint64
	Events  []PerfEvent
}

// State owns a native metadata store and must not be copied.
// Methods may be called concurrently. Replacements copy their inputs;
// callers must not modify input snapshots until the call returns.
// Invalid snapshots return an error even when their version is stale.
type State struct {
	mu     sync.RWMutex
	handle C.TPerforatorNativeState
	users  int
}

func NewState() (*State, error) {
	var handle C.TPerforatorNativeState
	if err := unwrapError(C.PerforatorNativeStateCreate(&handle)); err != nil {
		return nil, err
	}
	state := &State{handle: handle}
	runtime.SetFinalizer(state, func(s *State) { _ = s.Close() })
	return state, nil
}

// Close releases the store. It is idempotent and fails while a native consumer
// retains the handle; the caller may retry after that consumer releases it.
func (s *State) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.users != 0 {
		return errors.New("native pipeline state is still in use")
	}
	if s.handle != nil {
		C.PerforatorNativeStateDispose(s.handle)
		s.handle = nil
	}
	runtime.SetFinalizer(s, nil)
	return nil
}

// The caller must retain State until releaseHandle and stop using the native
// handle before releasing it.
func (s *State) acquireHandle() (C.TPerforatorNativeState, error) {
	if s == nil {
		return nil, errors.New("native pipeline state is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.handle == nil {
		return nil, errors.New("native pipeline state is closed")
	}
	s.users++
	return s.handle, nil
}

func (s *State) releaseHandle() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.users == 0 {
		panic("native pipeline state handle released without acquisition")
	}
	s.users--
}

func (s *State) withHandle(f func(C.TPerforatorNativeState) C.TPerforatorError) error {
	if s == nil {
		return errors.New("native pipeline state is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.handle == nil {
		return errors.New("native pipeline state is closed")
	}
	return unwrapError(f(s.handle))
}

func unwrapError(err C.TPerforatorError) error {
	if err == nil {
		return nil
	}
	message := C.GoString(C.PerforatorErrorString(err))
	C.PerforatorErrorDispose(err)
	return fmt.Errorf("native pipeline call failed: %s", message)
}

type cArena struct {
	allocations []unsafe.Pointer
}

func (a *cArena) close() {
	for _, allocation := range a.allocations {
		C.free(allocation)
	}
	*a = cArena{}
}

func (a *cArena) bytes(size int) unsafe.Pointer {
	if size == 0 {
		return nil
	}
	p := C.malloc(C.size_t(size))
	a.allocations = append(a.allocations, p)
	return p
}

func cArray[T any](a *cArena, count int) []T {
	if count == 0 {
		return nil
	}
	var zero T
	p := a.bytes(count * int(unsafe.Sizeof(zero)))
	return unsafe.Slice((*T)(p), count)
}

type stringArena struct {
	bytes  []byte
	offset int
}

func newStringArena(a *cArena, size int) *stringArena {
	return newStringArenaAt(a.bytes(size), size)
}

// The arena borrows C-allocated memory, which must remain valid while in use.
func newStringArenaAt(p unsafe.Pointer, size int) *stringArena {
	var bytes []byte
	if size != 0 {
		if p == nil {
			panic("native pipeline: nil storage for a non-empty string arena")
		}
		bytes = unsafe.Slice((*byte)(p), size)
	}
	return &stringArena{bytes: bytes}
}

func (a *stringArena) put(value string) C.TPerforatorStringView {
	if value == "" {
		return C.TPerforatorStringView{}
	}
	if len(value) > len(a.bytes)-a.offset {
		panic("native pipeline: string arena size calculation is out of sync with ABI marshalling")
	}
	begin := a.offset
	copy(a.bytes[begin:], value)
	a.offset += len(value)
	return C.TPerforatorStringView{
		Data: (*C.char)(unsafe.Pointer(&a.bytes[begin])),
		Size: C.size_t(len(value)),
	}
}

func (a *stringArena) finish() {
	if a.offset != len(a.bytes) {
		panic("native pipeline: string arena size calculation is out of sync with ABI marshalling")
	}
}

func processStringBytes(snapshot *ProcessSnapshot) int {
	size := 0
	for _, mapping := range snapshot.Mappings {
		size += len(mapping.Path) + len(mapping.BuildID)
	}
	for _, label := range snapshot.Environment {
		size += len(label.Key) + len(label.Value)
	}
	for _, name := range snapshot.TLSNames {
		size += len(name.Name)
	}
	return size
}

// ReplaceProcess replaces one process snapshot. Older lifetimes, stale versions
// and lifetimes with retained tombstones are ignored with a nil error.
// An unknown-identity tombstone requires a known start time and a newer version.
func (s *State) ReplaceProcess(snapshot *ProcessSnapshot) error {
	if snapshot == nil {
		return errors.New("process snapshot is nil")
	}
	var arena cArena
	defer arena.close()
	strings := newStringArena(&arena, processStringBytes(snapshot))

	mappings := cArray[C.TPerforatorProcessMapping](&arena, len(snapshot.Mappings))
	for i, mapping := range snapshot.Mappings {
		mappings[i] = C.TPerforatorProcessMapping{
			Begin:      C.uint64_t(mapping.Begin),
			End:        C.uint64_t(mapping.End),
			Offset:     C.uint64_t(mapping.Offset),
			FileOffset: C.uint64_t(mapping.FileOffset),
			Path:       strings.put(mapping.Path),
			BuildId:    strings.put(mapping.BuildID),
		}
	}

	environment := cArray[C.TPerforatorStringLabel](&arena, len(snapshot.Environment))
	for i, label := range snapshot.Environment {
		environment[i] = C.TPerforatorStringLabel{Key: strings.put(label.Key), Value: strings.put(label.Value)}
	}

	tlsNames := cArray[C.TPerforatorTLSName](&arena, len(snapshot.TLSNames))
	for i, name := range snapshot.TLSNames {
		tlsNames[i] = C.TPerforatorTLSName{Offset: C.uint64_t(name.Offset), Name: strings.put(name.Name)}
	}

	csnapshot := C.TPerforatorProcessSnapshot{
		Key: C.TPerforatorProcessKey{
			Pid:       C.uint32_t(snapshot.Key.Pid),
			StartTime: C.uint64_t(snapshot.Key.ProcessStartTime),
		},
		Version:          C.uint64_t(snapshot.Version),
		MappingCount:     C.size_t(len(mappings)),
		EnvironmentCount: C.size_t(len(environment)),
		TLSNameCount:     C.size_t(len(tlsNames)),
	}
	if len(mappings) != 0 {
		csnapshot.Mappings = &mappings[0]
	}
	if len(environment) != 0 {
		csnapshot.Environment = &environment[0]
	}
	if len(tlsNames) != 0 {
		csnapshot.TLSNames = &tlsNames[0]
	}
	strings.finish()

	return s.withHandle(func(handle C.TPerforatorNativeState) C.TPerforatorError {
		return C.PerforatorNativeStateReplaceProcess(handle, &csnapshot)
	})
}

// RemoveProcess retires this lifetime or an older one using the same version
// sequence as ReplaceProcess. Stale removals are ignored with a nil error.
// An unknown start time preserves the last known identity. Tombstones reject
// late publications while retained (the latest 65536 accepted removals).
func (s *State) RemoveProcess(key linux.ProcessKey, version uint64) error {
	return s.withHandle(func(handle C.TPerforatorNativeState) C.TPerforatorError {
		return C.PerforatorNativeStateRemoveProcess(
			handle,
			C.TPerforatorProcessKey{Pid: C.uint32_t(key.Pid), StartTime: C.uint64_t(key.ProcessStartTime)},
			C.uint64_t(version),
		)
	})
}

// ReplaceKernelSymbols replaces the complete symbol table without version checks.
// Symbols must be sorted by address in nondecreasing order.
func (s *State) ReplaceKernelSymbols(symbols []KernelSymbol) error {
	stringBytes := 0
	for _, symbol := range symbols {
		stringBytes += len(symbol.Name)
	}
	var arena cArena
	defer arena.close()
	strings := newStringArena(&arena, stringBytes)
	csymbols := cArray[C.TPerforatorKernelSymbol](&arena, len(symbols))
	for i, symbol := range symbols {
		csymbols[i] = C.TPerforatorKernelSymbol{
			Address: C.uint64_t(symbol.Address),
			Name:    strings.put(symbol.Name),
		}
	}
	snapshot := C.TPerforatorKernelSymbolSnapshot{SymbolCount: C.size_t(len(csymbols))}
	if len(csymbols) != 0 {
		snapshot.Symbols = &csymbols[0]
	}
	strings.finish()
	return s.withHandle(func(handle C.TPerforatorNativeState) C.TPerforatorError {
		return C.PerforatorNativeStateReplaceKernelSymbols(handle, &snapshot)
	})
}

func cgroupStringBytes(snapshot *CgroupSnapshot) int {
	size := 0
	for _, name := range snapshot.Names {
		size += len(name.BaseName) + len(name.FullName)
	}
	for _, workload := range snapshot.Workloads {
		size += len(workload.CgroupBaseName) + len(workload.ServiceName) + len(workload.ContainerName)
	}
	return size
}

// ReplaceCgroups replaces all cgroup metadata. Stale versions return nil without changes.
func (s *State) ReplaceCgroups(snapshot *CgroupSnapshot) error {
	if snapshot == nil {
		return errors.New("cgroup snapshot is nil")
	}
	var arena cArena
	defer arena.close()
	strings := newStringArena(&arena, cgroupStringBytes(snapshot))

	names := cArray[C.TPerforatorCgroupName](&arena, len(snapshot.Names))
	for i, name := range snapshot.Names {
		names[i] = C.TPerforatorCgroupName{
			Id:       C.uint64_t(name.ID),
			BaseName: strings.put(name.BaseName),
			FullName: strings.put(name.FullName),
		}
	}
	workloads := cArray[C.TPerforatorWorkload](&arena, len(snapshot.Workloads))
	for i, workload := range snapshot.Workloads {
		workloads[i] = C.TPerforatorWorkload{
			CgroupBaseName: strings.put(workload.CgroupBaseName),
			ServiceName:    strings.put(workload.ServiceName),
			ContainerName:  strings.put(workload.ContainerName),
		}
	}

	csnapshot := C.TPerforatorCgroupSnapshot{
		Version:       C.uint64_t(snapshot.Version),
		NameCount:     C.size_t(len(names)),
		WorkloadCount: C.size_t(len(workloads)),
	}
	if len(names) != 0 {
		csnapshot.Names = &names[0]
	}
	if len(workloads) != 0 {
		csnapshot.Workloads = &workloads[0]
	}
	strings.finish()
	return s.withHandle(func(handle C.TPerforatorNativeState) C.TPerforatorError {
		return C.PerforatorNativeStateReplaceCgroups(handle, &csnapshot)
	})
}

func targetStringBytes(snapshot *TargetSnapshot) int {
	size := 0
	for _, uprobe := range snapshot.BaseUprobes {
		size += len(uprobe.BuildID) + len(uprobe.ProfileName) + len(uprobe.SampleKind)
	}
	for _, target := range snapshot.Targets {
		for _, label := range target.Labels {
			size += len(label.Key) + len(label.Value)
		}
		for _, event := range target.PerfEvents {
			size += len(event.Kind) + len(event.Unit)
		}
		for _, uprobe := range target.Uprobes {
			size += len(uprobe.BuildID) + len(uprobe.ProfileName) + len(uprobe.SampleKind)
		}
	}
	return size
}

func boolByte(value bool) C.uint8_t {
	if value {
		return 1
	}
	return 0
}

// ReplaceTargets replaces all targets and base uprobes. Stale versions return nil without changes.
func (s *State) ReplaceTargets(snapshot *TargetSnapshot) error {
	if snapshot == nil {
		return errors.New("target snapshot is nil")
	}
	var arena cArena
	defer arena.close()
	strings := newStringArena(&arena, targetStringBytes(snapshot))

	targets := cArray[C.TPerforatorTarget](&arena, len(snapshot.Targets))
	labelCount, perfEventCount, uprobeCount := 0, 0, len(snapshot.BaseUprobes)
	for _, target := range snapshot.Targets {
		labelCount += len(target.Labels)
		perfEventCount += len(target.PerfEvents)
		uprobeCount += len(target.Uprobes)
	}
	labels := cArray[C.TPerforatorStringLabel](&arena, labelCount)
	perfEvents := cArray[C.TPerforatorPerfEvent](&arena, perfEventCount)
	uprobes := cArray[C.TPerforatorTargetUprobe](&arena, uprobeCount)
	for i, uprobe := range snapshot.BaseUprobes {
		uprobes[i] = C.TPerforatorTargetUprobe{
			BuildId:     strings.put(uprobe.BuildID),
			Offset:      C.uint64_t(uprobe.Offset),
			ProfileName: strings.put(uprobe.ProfileName),
			SampleKind:  strings.put(uprobe.SampleKind),
		}
	}
	labelOffset, perfEventOffset, uprobeOffset := 0, 0, len(snapshot.BaseUprobes)
	for i, target := range snapshot.Targets {
		for j, label := range target.Labels {
			labels[labelOffset+j] = C.TPerforatorStringLabel{
				Key:   strings.put(label.Key),
				Value: strings.put(label.Value),
			}
		}
		for j, event := range target.PerfEvents {
			perfEvents[perfEventOffset+j] = C.TPerforatorPerfEvent{
				Id:   C.uint64_t(event.ID),
				Kind: strings.put(event.Kind),
				Unit: strings.put(event.Unit),
			}
		}
		for j, uprobe := range target.Uprobes {
			uprobes[uprobeOffset+j] = C.TPerforatorTargetUprobe{
				BuildId:     strings.put(uprobe.BuildID),
				Offset:      C.uint64_t(uprobe.Offset),
				ProfileName: strings.put(uprobe.ProfileName),
				SampleKind:  strings.put(uprobe.SampleKind),
			}
		}
		targets[i] = C.TPerforatorTarget{
			Id:                    C.uint64_t(target.ID),
			Kind:                  C.TPerforatorTargetKind(target.Kind),
			Key:                   C.uint64_t(target.Key),
			EnableSampleTime:      boolByte(target.EnableSampleTime),
			EnableInnermostPidns:  boolByte(target.EnableInnermostPIDNS),
			LabelCount:            C.size_t(len(target.Labels)),
			CreatedTimestampNanos: C.int64_t(target.CreatedTimestampNanos),
			StartTimestampNanos:   C.int64_t(target.StartTimestampNanos),
			EndTimestampNanos:     C.int64_t(target.EndTimestampNanos),
			CollectAllPerfEvents:  boolByte(target.CollectAllPerfEvents),
			Pending:               boolByte(target.Pending),
			PerfEventCount:        C.size_t(len(target.PerfEvents)),
			UprobeCount:           C.size_t(len(target.Uprobes)),
		}
		if len(target.Labels) != 0 {
			targets[i].Labels = &labels[labelOffset]
		}
		if len(target.PerfEvents) != 0 {
			targets[i].PerfEvents = &perfEvents[perfEventOffset]
		}
		if len(target.Uprobes) != 0 {
			targets[i].Uprobes = &uprobes[uprobeOffset]
		}
		labelOffset += len(target.Labels)
		perfEventOffset += len(target.PerfEvents)
		uprobeOffset += len(target.Uprobes)
	}

	csnapshot := C.TPerforatorTargetSnapshot{
		Version:         C.uint64_t(snapshot.Version),
		BaseUprobeCount: C.size_t(len(snapshot.BaseUprobes)),
		TargetCount:     C.size_t(len(targets)),
	}
	if len(snapshot.BaseUprobes) != 0 {
		csnapshot.BaseUprobes = &uprobes[0]
	}
	if len(targets) != 0 {
		csnapshot.Targets = &targets[0]
	}
	strings.finish()
	return s.withHandle(func(handle C.TPerforatorNativeState) C.TPerforatorError {
		return C.PerforatorNativeStateReplaceTargets(handle, &csnapshot)
	})
}

// ReplacePerfEvents replaces all perf events. Stale versions return nil without changes.
func (s *State) ReplacePerfEvents(snapshot *PerfEventSnapshot) error {
	if snapshot == nil {
		return errors.New("perf event snapshot is nil")
	}
	stringBytes := 0
	for _, event := range snapshot.Events {
		stringBytes += len(event.Kind) + len(event.Unit)
	}
	var arena cArena
	defer arena.close()
	strings := newStringArena(&arena, stringBytes)
	events := cArray[C.TPerforatorPerfEvent](&arena, len(snapshot.Events))
	for i, event := range snapshot.Events {
		events[i] = C.TPerforatorPerfEvent{
			Id:   C.uint64_t(event.ID),
			Kind: strings.put(event.Kind),
			Unit: strings.put(event.Unit),
		}
	}
	csnapshot := C.TPerforatorPerfEventSnapshot{
		Version:    C.uint64_t(snapshot.Version),
		EventCount: C.size_t(len(events)),
	}
	if len(events) != 0 {
		csnapshot.Events = &events[0]
	}
	strings.finish()
	return s.withHandle(func(handle C.TPerforatorNativeState) C.TPerforatorError {
		return C.PerforatorNativeStateReplacePerfEvents(handle, &csnapshot)
	})
}

package jit_test

import (
	"context"
	"encoding/binary"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/internal/linguist/lua/jit"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

type hints []programstate.LuaJITVM

func (h *hints) LuaJITVMs() ([]programstate.LuaJITVM, error)   { return *h, nil }
func (h *hints) DeleteLuaJITVM(vm programstate.LuaJITVM) error { return nil }

type publication struct {
	table *unwind.UnwindTable
	err   error
}

func (p *publication) UpdateLuaJITTable(key linux.ProcessKey, table *unwind.UnwindTable) error {
	if p.err != nil {
		return p.err
	}
	p.table = table
	return nil
}

type processMemory struct {
	memory
	maps []procfs.Mapping
}

func (m processMemory) Read(_ linux.ProcessKey, address uint64, dst []byte) error {
	return m.read(address, dst)
}
func (m processMemory) Mappings(_ linux.ProcessKey) ([]procfs.Mapping, error) { return m.maps, nil }

type processInfo struct{ key linux.ProcessKey }

func (p processInfo) Key() linux.ProcessKey       { return p.key }
func (p processInfo) Env() map[string]string      { return nil }
func (p processInfo) Mappings() []process.Mapping { return nil }

func TestRegistryRefreshPublishesAndRemovesNativeRules(t *testing.T) {
	m, vm, maps := traceMemory()
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm), BinaryBase: 0}}
	pub := &publication{}
	registry := jit.NewRegistry(nil, &h, pub, processMemory{m, maps})
	registry.OnProcessDiscovery(context.Background(), processInfo{key})
	require.NoError(t, registry.Refresh(context.Background()))
	require.NotNil(t, pub.table)
	require.Equal(t, []uint64{0x10000, 0x10004, 0x10009}, pub.table.StartPc)
	registry.OnProcessDeath(context.Background(), key)
	require.Nil(t, pub.table)
}

func TestRegistryRetriesFailedPublication(t *testing.T) {
	m, vm, maps := traceMemory()
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}
	pub := &publication{err: errors.New("map full")}
	registry := jit.NewRegistry(nil, &h, pub, processMemory{m, maps})
	registry.OnProcessDiscovery(context.Background(), processInfo{key})
	require.Error(t, registry.Refresh(context.Background()))
	require.Nil(t, pub.table)
	pub.err = nil
	require.NoError(t, registry.Refresh(context.Background()))
	require.NotNil(t, pub.table)
}

func luaConfig(vm jit.VM) unwinder.LuaConfig {
	return unwinder.LuaConfig{OffsetGToVmState: vm.OffsetGToVMState, OffsetGToDispatch: vm.OffsetGToDispatch, VmStartPc: vm.VMStart, VmEndPc: vm.VMEnd}
}

func TestRegistryCombinesVMs(t *testing.T) {
	m, vm, maps := traceMemory()
	m[0x110000] = append([]byte{}, m[vm.GlobalState]...)
	binary.LittleEndian.PutUint64(m[0x110000][0x170:], 0x81000)
	binary.LittleEndian.PutUint64(m[0x110000][192:], 0x81000)
	binary.LittleEndian.PutUint64(m[0x110000][1128:], 0x21000)
	m[0x81000] = make([]byte, 24)
	m[0x81000][9] = 6
	binary.LittleEndian.PutUint64(m[0x81000][16:], 0x110000)
	m[0x21000] = make([]byte, 16)
	binary.LittleEndian.PutUint64(m[0x21000][8:], 0x31000)
	m[0x31000] = append([]byte{}, m[0x30000]...)
	binary.LittleEndian.PutUint64(m[0x31000][88:], 0x12000)
	m[0x12000] = append([]byte{}, m[0x10000]...)
	maps = append(maps, procfs.Mapping{Begin: 0x12000, End: 0x13000, Permissions: procfs.MappingPermissionRXP})
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{
		{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)},
		{Process: key, GlobalState: 0x110000, Config: luaConfig(vm)},
	}
	pub := &publication{}
	r := jit.NewRegistry(nil, &h, pub, processMemory{m, maps})
	r.OnProcessDiscovery(context.Background(), processInfo{key})
	require.NoError(t, r.Refresh(context.Background()))
	require.Equal(t, []uint64{0x10000, 0x10004, 0x10009, 0x12000, 0x12004, 0x12009}, pub.table.StartPc)
}

func TestRegistryRemovesFlushedTraceRules(t *testing.T) {
	m, vm, maps := traceMemory()
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}
	pub := &publication{}
	r := jit.NewRegistry(nil, &h, pub, processMemory{m, maps})
	r.OnProcessDiscovery(context.Background(), processInfo{key})
	require.NoError(t, r.Refresh(context.Background()))
	clear(m[0x20000])
	require.NoError(t, r.Refresh(context.Background()))
	require.Nil(t, pub.table)
}

func TestRegistryRetriesRetirementWithoutReadingDeadProcess(t *testing.T) {
	m, vm, maps := traceMemory()
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}
	pub := &publication{}
	r := jit.NewRegistry(nil, &h, pub, processMemory{m, maps})
	r.OnProcessDiscovery(context.Background(), processInfo{key})
	require.NoError(t, r.Refresh(context.Background()))
	pub.err = errors.New("transient failure")
	r.OnProcessDeath(context.Background(), key)
	pub.err = nil
	clear(m)
	require.NoError(t, r.Refresh(context.Background()))
	require.Nil(t, pub.table)
	require.NoError(t, r.Refresh(context.Background()))
}

func TestRegistryRejectsInvalidRefreshInterval(t *testing.T) {
	for _, value := range []string{"bad", "0s", "-1s", ""} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("PERFORATOR_LUA_JIT_REFRESH_INTERVAL", value)
			h := hints{}
			r := jit.NewRegistry(nil, &h, &publication{}, nil)
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			require.ErrorContains(t, r.Run(ctx), "PERFORATOR_LUA_JIT_REFRESH_INTERVAL")
		})
	}
}

type scanNotifications struct{ calls chan struct{} }

func (s scanNotifications) LuaJITVMs() ([]programstate.LuaJITVM, error) {
	select {
	case s.calls <- struct{}{}:
	default:
	}
	return nil, nil
}
func (s scanNotifications) DeleteLuaJITVM(programstate.LuaJITVM) error { return nil }

func TestRegistryUsesConfiguredRefreshInterval(t *testing.T) {
	t.Setenv("PERFORATOR_LUA_JIT_REFRESH_INTERVAL", "5ms")
	source := scanNotifications{calls: make(chan struct{}, 2)}
	r := jit.NewRegistry(nil, source, &publication{}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()
	for range 2 {
		select {
		case <-source.calls:
		case <-ctx.Done():
			t.Fatal("registry did not refresh at the configured interval")
		}
	}
	cancel()
	require.NoError(t, <-done)
}

type layoutReadOnce struct {
	processMemory
	vmStart uint64
	reads   int
}

func (m *layoutReadOnce) Read(key linux.ProcessKey, address uint64, dst []byte) error {
	if address == m.vmStart {
		m.reads++
		if m.reads > 1 {
			return errors.New("VM text must not be reread during refresh")
		}
	}
	return m.processMemory.Read(key, address, dst)
}

func TestRegistryCachesLayoutAndRereadsMutableTraceMetadata(t *testing.T) {
	m, vm, maps := traceMemory()
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}
	pub := &publication{}
	memory := &layoutReadOnce{processMemory: processMemory{m, maps}, vmStart: vm.VMStart}
	r := jit.NewRegistry(nil, &h, pub, memory)
	r.OnProcessDiscovery(context.Background(), processInfo{key})
	require.NoError(t, r.Refresh(context.Background()))
	require.NotNil(t, pub.table)
	require.NoError(t, r.Refresh(context.Background()))
	require.NotNil(t, pub.table)
	clear(m[0x20000])
	require.NoError(t, r.Refresh(context.Background()))
	require.Nil(t, pub.table)
	require.Equal(t, 1, memory.reads)
}

type mappingFailureMemory struct {
	processMemory
	mappings int
	vmReads  int
	vmStart  uint64
}

func (m *mappingFailureMemory) Mappings(key linux.ProcessKey) ([]procfs.Mapping, error) {
	m.mappings++
	if m.mappings == 4 {
		return nil, errors.New("process identity changed")
	}
	return m.processMemory.Mappings(key)
}
func (m *mappingFailureMemory) Read(key linux.ProcessKey, address uint64, dst []byte) error {
	if address == m.vmStart {
		m.vmReads++
	}
	return m.processMemory.Read(key, address, dst)
}
func TestRegistryInvalidatesLayoutAfterFinalIdentityCheckFails(t *testing.T) {
	m, vm, maps := traceMemory()
	key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
	h := hints{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}
	memory := &mappingFailureMemory{processMemory: processMemory{m, maps}, vmStart: vm.VMStart}
	r := jit.NewRegistry(nil, &h, &publication{}, memory)
	r.OnProcessDiscovery(context.Background(), processInfo{key})
	require.NoError(t, r.Refresh(context.Background()))
	require.ErrorContains(t, r.Refresh(context.Background()), "process identity changed")
	require.NoError(t, r.Refresh(context.Background()))
	require.Equal(t, 2, memory.vmReads, "observed identity failure must discard the cached layout")
}

type changingVMSource struct {
	mu    sync.Mutex
	vms   []programstate.LuaJITVM
	calls chan struct{}
}

func (s *changingVMSource) LuaJITVMs() ([]programstate.LuaJITVM, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := append([]programstate.LuaJITVM{}, s.vms...)
	select {
	case s.calls <- struct{}{}:
	default:
	}
	return result, nil
}
func (s *changingVMSource) DeleteLuaJITVM(programstate.LuaJITVM) error { return nil }
func (s *changingVMSource) discover(vm programstate.LuaJITVM) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vms = append(s.vms, vm)
}

type tableNotifications chan *unwind.UnwindTable

func (p tableNotifications) UpdateLuaJITTable(_ linux.ProcessKey, table *unwind.UnwindTable) error {
	p <- table
	return nil
}
func TestFirstLuaSampleTriggersScanBeforePeriodicRefresh(t *testing.T) {
	for _, registration := range []string{"registered", "discovered_after_sample", "identity_after_sample"} {
		t.Run(registration, func(t *testing.T) {
			t.Setenv("PERFORATOR_LUA_JIT_REFRESH_INTERVAL", "10s")
			m, vm, maps := traceMemory()
			key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
			bootstrap := linux.ProcessKey{Pid: key.Pid}
			source := &changingVMSource{calls: make(chan struct{}, 8)}
			tables := make(tableNotifications, 8)
			r := jit.NewRegistry(nil, source, tables, processMemory{m, maps})
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			done := make(chan error, 1)
			defer func() { cancel(); require.NoError(t, <-done) }()
			if registration == "registered" {
				r.OnProcessDiscovery(ctx, processInfo{key})
			}
			if registration == "identity_after_sample" {
				r.OnProcessDiscovery(ctx, processInfo{bootstrap})
			}
			go func() { done <- r.Run(ctx) }()
			awaitScan := func() {
				select {
				case <-source.calls:
				case <-ctx.Done():
					t.Fatal("first Lua sample did not trigger an early scan")
				}
			}
			awaitScan()
			source.discover(programstate.LuaJITVM{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)})
			r.ObserveLuaSample(key)
			if registration != "registered" {
				awaitScan()
				if registration == "discovered_after_sample" {
					r.OnProcessDiscovery(ctx, processInfo{key})
				} else {
					r.OnProcessIdentityChanged(ctx, bootstrap, key)
				}
			}
			for {
				select {
				case table := <-tables:
					if table != nil {
						require.Equal(t, []uint64{0x10000, 0x10004, 0x10009}, table.StartPc)
						// Drain already-observed scans; ordinary rescans must
						// not become new first-sample wake-ups.
						for len(source.calls) > 0 {
							<-source.calls
						}
						r.OnProcessRescan(ctx, processInfo{key})
						select {
						case <-source.calls:
							t.Fatal("same-process rescan triggered an extra early scan")
						case <-time.After(100 * time.Millisecond):
						}
						return
					}
				case <-ctx.Done():
					t.Fatal("rules were not published before the periodic refresh")
				}
			}
		})
	}
}

type trackedVMHints struct {
	values    []programstate.LuaJITVM
	deletions int
}

func (s *trackedVMHints) LuaJITVMs() ([]programstate.LuaJITVM, error) { return s.values, nil }
func (s *trackedVMHints) DeleteLuaJITVM(programstate.LuaJITVM) error {
	s.deletions++
	s.values = nil
	return nil
}

type changingCodeMemory struct {
	processMemory
	enabled, persistent bool
	codeReads           int
}

func (m *changingCodeMemory) Read(key linux.ProcessKey, address uint64, dst []byte) error {
	if m.enabled && address == 0x10000 {
		m.codeReads++
		if m.codeReads%2 == 0 && (m.persistent || m.codeReads == 2) {
			m.memory[0x10000][4] ^= 1
		}
	}
	return m.processMemory.Read(key, address, dst)
}

func TestSnapshotRacesRetryWithoutDiscardingVMDiscovery(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		t.Run(map[bool]string{false: "one_race", true: "retry_budget_exhausted"}[persistent], func(t *testing.T) {
			m, vm, maps := traceMemory()
			key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
			source := &trackedVMHints{values: []programstate.LuaJITVM{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}}
			pub := &publication{}
			remote := &changingCodeMemory{processMemory: processMemory{m, maps}, persistent: persistent}
			r := jit.NewRegistry(nil, source, pub, remote)
			r.OnProcessDiscovery(context.Background(), processInfo{key})
			require.NoError(t, r.Refresh(context.Background()))
			require.NotNil(t, pub.table)
			remote.enabled = true
			err := r.Refresh(context.Background())
			if persistent {
				require.ErrorContains(t, err, "snapshot changed")
				require.Nil(t, pub.table, "unvalidated old rules must not survive an actual code change")
			} else {
				require.NoError(t, err)
			}
			require.Zero(t, source.deletions, "a snapshot race is not VM deallocation")
			remote.enabled = false
			require.NoError(t, r.Refresh(context.Background()))
			require.NotNil(t, pub.table, "retained discovery must recover without a new sample hint")
		})
	}
}

func TestStableVMIdentityChangesInvalidateDiscovery(t *testing.T) {
	for _, field := range []string{"mainthref", "thread_type", "global_reference"} {
		t.Run(field, func(t *testing.T) {
			m, vm, maps := traceMemory()
			key := linux.ProcessKey{Pid: 42, ProcessStartTime: 100000000}
			source := &trackedVMHints{values: []programstate.LuaJITVM{{Process: key, GlobalState: vm.GlobalState, Config: luaConfig(vm)}}}
			pub := &publication{}
			r := jit.NewRegistry(nil, source, pub, processMemory{m, maps})
			r.OnProcessDiscovery(context.Background(), processInfo{key})
			require.NoError(t, r.Refresh(context.Background()))
			require.NotNil(t, pub.table)
			switch field {
			case "mainthref":
				binary.LittleEndian.PutUint64(m[vm.GlobalState][192:], 0x81000)
			case "thread_type":
				m[0x80000][9] = 7
			case "global_reference":
				binary.LittleEndian.PutUint64(m[0x80000][16:], 0)
			}
			require.ErrorContains(t, r.Refresh(context.Background()), "VM identity")
			require.Equal(t, 1, source.deletions)
			require.Nil(t, pub.table)
		})
	}
}

package jit

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine/programstate"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/process"
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/linux/procmem"
)

type VMSource interface {
	LuaJITVMs() ([]programstate.LuaJITVM, error)
	DeleteLuaJITVM(programstate.LuaJITVM) error
}

type TablePublisher interface {
	UpdateLuaJITTable(linux.ProcessKey, *unwind.UnwindTable) error
}

// ProcessMemory is the remote process boundary used during a refresh.
type ProcessMemory interface {
	Read(linux.ProcessKey, uint64, []byte) error
	Mappings(linux.ProcessKey) ([]procfs.Mapping, error)
}

type vmKey struct {
	process linux.ProcessKey
	vm      VM
}

type Registry struct {
	observed             sync.Map // ProcessKey -> first Lua sample time.
	lastObservationPrune time.Time
	wake                 chan struct{}
	layouts              map[vmKey]*vmLayout
	mu                   sync.Mutex
	source               VMSource
	publisher            TablePublisher
	memory               ProcessMemory
	log                  log.Logger
	processes            map[linux.CurrentNamespacePID]linux.ProcessKey
	retiring             map[linux.ProcessKey]bool
}

func NewRegistry(l log.Logger, source VMSource, publisher TablePublisher, memory ProcessMemory) *Registry {
	if memory == nil {
		memory = remoteMemory{}
	}
	return &Registry{
		wake:      make(chan struct{}, 1),
		layouts:   make(map[vmKey]*vmLayout),
		source:    source,
		publisher: publisher,
		memory:    memory,
		log:       l,
		processes: make(map[linux.CurrentNamespacePID]linux.ProcessKey),
		retiring:  make(map[linux.ProcessKey]bool),
	}
}

func (r *Registry) Refresh(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.pruneObservations(time.Now())
	hints, err := r.source.LuaJITVMs()
	if err != nil {
		return err
	}
	grouped := make(map[linux.ProcessKey][]programstate.LuaJITVM)
	for _, hint := range hints {
		current := r.processes[hint.Process.Pid]
		if current != hint.Process || current.ProcessStartTime == 0 {
			if current.ProcessStartTime > hint.Process.ProcessStartTime {
				_ = r.source.DeleteLuaJITVM(hint)
			}
			continue
		}
		grouped[hint.Process] = append(grouped[hint.Process], hint)
	}
	var errs []error
	activeLayouts := make(map[vmKey]bool)
	for _, key := range r.processes {
		if err := ctx.Err(); err != nil {
			return err
		}
		if key.ProcessStartTime == 0 {
			continue
		}
		var rows []nativeRow
		targets := grouped[key]
		if r.retiring[key] {
			targets = nil
		}
		for _, hint := range targets {
			vm := VM{GlobalState: hint.GlobalState, OffsetGToVMState: hint.Config.OffsetGToVmState, OffsetGToDispatch: hint.Config.OffsetGToDispatch, VMStart: hint.BinaryBase + hint.Config.VmStartPc, VMEnd: hint.BinaryBase + hint.Config.VmEndPc}
			cacheKey := vmKey{key, vm}
			layout := r.layouts[cacheKey]
			if layout == nil {
				layout = &vmLayout{}
			}
			table, err := r.scanVM(ctx, key, vm, layout)
			if err != nil {
				var invalid *vmIdentityError
				if errors.As(err, &invalid) {
					_ = r.source.DeleteLuaJITVM(hint)
				}
				errs = append(errs, fmt.Errorf("VM %#x in process %d failed: %w", hint.GlobalState, key.Pid, err))
				continue
			}
			r.layouts[cacheKey] = layout
			activeLayouts[cacheKey] = true
			for i, pc := range table.StartPc {
				offset := int32(-1)
				if cfa := table.Dict[table.Cfa[i]].GetRegisterOffset(); cfa != nil {
					offset = cfa.Offset
				}
				rows = append(rows, nativeRow{pc, pc + table.PcRange[i], offset})
			}
		}
		combined, err := rowsToTable(rows)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if len(combined.StartPc) == 0 {
			combined = nil
		}
		err = r.publisher.UpdateLuaJITTable(key, combined)
		if err != nil {
			errs = append(errs, err)
		} else if r.retiring[key] {
			delete(r.processes, key.Pid)
			delete(r.retiring, key)
		}
	}
	for key := range r.layouts {
		if !activeLayouts[key] {
			delete(r.layouts, key)
		}
	}
	return errors.Join(errs...)
}

// Retry a moving VM snapshot in the same refresh cycle. Only a stable identity
// mismatch invalidates discovery. Exhaustion still withdraws unvalidated rules;
// retaining old rows blindly would be unsafe after actual code reuse or teardown.
func (r *Registry) scanVM(ctx context.Context, key linux.ProcessKey, vm VM, layout *vmLayout) (*unwind.UnwindTable, error) {
	const maxAttempts = 3
	var table *unwind.UnwindTable
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err = ctx.Err(); err != nil {
			return nil, err
		}
		mappings, mapErr := r.memory.Mappings(key)
		if mapErr != nil {
			return nil, mapErr
		}
		table, err = readTable(func(address uint64, dst []byte) error { return r.memory.Read(key, address, dst) }, vm, mappings, layout)
		if err == nil {
			_, err = r.memory.Mappings(key)
			return table, err
		}
		var invalid *vmIdentityError
		if errors.As(err, &invalid) {
			return nil, err
		}
	}
	return nil, err
}

func (r *Registry) OnProcessDiscovery(ctx context.Context, info process.ProcessInfo) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := info.Key()
	previous, known := r.processes[key.Pid]
	if known && previous != key && previous.ProcessStartTime != 0 {
		if err := r.publisher.UpdateLuaJITTable(previous, nil); err != nil && r.log != nil {
			r.log.Warn("Failed to remove previous LuaJIT process table", log.Error(err))
		}
		r.forgetLayouts(previous)
		delete(r.retiring, previous)
		r.observed.Delete(previous)
	}
	r.processes[key.Pid] = key
	if !known || previous != key {
		if _, ok := r.observed.Load(key); ok {
			r.requestScan()
		}
	}
}

func (r *Registry) OnProcessRescan(ctx context.Context, info process.ProcessInfo) {
	// Exec or library replacement can preserve PID and process start time.
	r.mu.Lock()
	r.forgetLayouts(info.Key())
	r.mu.Unlock()
	r.OnProcessDiscovery(ctx, info)
}

func (r *Registry) OnProcessIdentityChanged(ctx context.Context, previous, current linux.ProcessKey) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.processes[previous.Pid] == previous && previous != current {
		r.forgetLayouts(previous)
		r.processes[current.Pid] = current
		r.observed.Delete(previous)
		if _, ok := r.observed.Load(current); ok {
			r.requestScan()
		}
	}
}

func (r *Registry) OnProcessDeath(ctx context.Context, key linux.ProcessKey) {
	r.observed.Delete(key)
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.processes[key.Pid] != key {
		return
	}
	r.forgetLayouts(key)
	if err := r.publisher.UpdateLuaJITTable(key, nil); err != nil {
		r.retiring[key] = true
		if r.log != nil {
			r.log.Warn("Failed to remove LuaJIT unwind table", log.Error(err))
		}
		return
	}
	delete(r.processes, key.Pid)
	delete(r.retiring, key)
}

func (r *Registry) forgetLayouts(process linux.ProcessKey) {
	for key := range r.layouts {
		if key.process == process {
			delete(r.layouts, key)
		}
	}
}

// Samples may arrive after death or before a registration that never succeeds.
// Periodic maintenance expires orphaned observations without losing the brief
// sample-before-registration window used to retry discovery.
func (r *Registry) pruneObservations(now time.Time) {
	if now.Sub(r.lastObservationPrune) < 10*time.Second {
		return
	}
	r.lastObservationPrune = now
	r.observed.Range(func(k, v any) bool {
		key := k.(linux.ProcessKey)
		if now.Sub(v.(time.Time)) >= time.Minute && (r.processes[key.Pid] != key || r.retiring[key]) {
			r.observed.Delete(key)
		}
		return true
	})
}

// ObserveLuaSample requests discovery once per process using the existing sample
// stream. It does no remote reads and never waits for the refresh worker.
// The resulting rules can only help later samples.
func (r *Registry) ObserveLuaSample(key linux.ProcessKey) {
	if key.Pid == 0 || key.ProcessStartTime == 0 {
		return
	}
	if _, loaded := r.observed.Load(key); loaded {
		return
	}
	if _, loaded := r.observed.LoadOrStore(key, time.Now()); !loaded {
		r.requestScan()
	}
}

func (r *Registry) requestScan() {
	select {
	case r.wake <- struct{}{}:
	default:
	}
}

func (r *Registry) Run(ctx context.Context) error {
	interval := 10 * time.Second
	if value, ok := os.LookupEnv("PERFORATOR_LUA_JIT_REFRESH_INTERVAL"); ok {
		var err error
		interval, err = time.ParseDuration(value)
		if err != nil || interval <= 0 {
			return fmt.Errorf("invalid PERFORATOR_LUA_JIT_REFRESH_INTERVAL %q: expected a positive duration", value)
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := r.Refresh(ctx); err != nil && r.log != nil {
			r.log.Warn("Failed to refresh LuaJIT unwind tables", log.Error(err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		case <-r.wake:
		}
	}
}

type remoteMemory struct{}

func (remoteMemory) Read(key linux.ProcessKey, address uint64, dst []byte) error {
	return procmem.Read(int(key.Pid), uintptr(address), dst)
}

func (remoteMemory) Mappings(key linux.ProcessKey) ([]procfs.Mapping, error) {
	p := procfs.Open(key.Pid)
	stat, err := p.Stat()
	// Linux exposes /proc starttime in USER_HZ (100) ticks, BPF uses ns.
	if err != nil {
		return nil, err
	}
	if key.ProcessStartTime == 0 || key.ProcessStartTime/10_000_000 != stat.StartTimeTicks {
		return nil, fmt.Errorf("process identity changed")
	}
	var mappings []procfs.Mapping
	err = p.ListMappings(func(m *procfs.Mapping) error { mappings = append(mappings, *m); return nil })
	return mappings, err
}

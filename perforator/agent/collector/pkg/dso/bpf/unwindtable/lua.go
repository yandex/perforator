package unwindtable

import (
	"errors"
	"fmt"
	"sync"

	"github.com/cilium/ebpf"
	"golang.org/x/sys/unix"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

type luaJITTable struct {
	key     linux.ProcessKey
	table   *unwind.UnwindTable
	current *Allocation
	retired []*Allocation
}

// LuaJITPublisher manages LuaJIT rows in the existing per-PID root map. Pages
// are released only after readers of a removed root have finished.
type LuaJITPublisher struct {
	manager *BPFManager
	mu      sync.Mutex
	tables  map[linux.CurrentNamespacePID]*luaJITTable
}

func NewLuaJITPublisher(manager *BPFManager) *LuaJITPublisher {
	return &LuaJITPublisher{manager: manager, tables: make(map[linux.CurrentNamespacePID]*luaJITTable)}
}

func (p *LuaJITPublisher) UpdateLuaJITTable(key linux.ProcessKey, table *unwind.UnwindTable) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	old := p.tables[key.Pid]
	if old == nil {
		if table == nil {
			return nil
		}
		old = &luaJITTable{key: key}
	} else if old.key.ProcessStartTime > key.ProcessStartTime {
		return nil
	}

	if table == nil || old.key != key {
		if err := p.detach(old); err != nil {
			return err
		}
	}
	// If a grace period failed previously, do not accumulate more pages.
	if err := p.reclaim(old); err != nil {
		if detachErr := p.detach(old); detachErr != nil {
			return detachErr
		}
		return err
	}
	if table == nil {
		delete(p.tables, key.Pid)
		return nil
	}
	if old.current != nil && proto.Equal(old.table, table) {
		return nil
	}

	next, err := p.manager.Add(AllocationID{PID: key.Pid}, table)
	if err != nil {
		// The changed snapshot may have removed or reused a trace address.
		if detachErr := p.detach(old); detachErr != nil {
			return errors.Join(err, detachErr)
		}
		return errors.Join(err, p.reclaim(old))
	}
	if old.current != nil {
		old.retired = append(old.retired, old.current)
	}
	old.key, old.table, old.current = key, table, next
	p.tables[key.Pid] = old
	if err := p.reclaim(old); err != nil {
		if detachErr := p.detach(old); detachErr != nil {
			return detachErr
		}
		return err
	}
	return nil
}

func (p *LuaJITPublisher) detach(table *luaJITTable) error {
	if table.current == nil {
		return nil
	}
	if err := p.manager.state.DeleteProcessUnwindTable(table.key.Pid); err != nil && !errors.Is(err, ebpf.ErrKeyNotExist) {
		return err
	}
	table.retired = append(table.retired, table.current)
	table.current, table.table = nil, nil
	return nil
}

func (p *LuaJITPublisher) reclaim(table *luaJITTable) error {
	if len(table.retired) == 0 {
		return nil
	}
	// MEMBARRIER_CMD_GLOBAL waits for an RCU grace period, after which
	// non-sleepable BPF programs can no longer be using the retired roots.
	if _, _, errno := unix.Syscall(unix.SYS_MEMBARRIER, 1 /* MEMBARRIER_CMD_GLOBAL */, 0, 0); errno != 0 {
		return fmt.Errorf("waiting for LuaJIT unwind readers: %w", errno)
	}
	for _, old := range table.retired {
		old.statemu.Lock()
		old.id = AllocationID{} // The root was removed or replaced before the grace period.
		p.manager.release(old)
		old.statemu.Unlock()
	}
	table.retired = nil
	return nil
}

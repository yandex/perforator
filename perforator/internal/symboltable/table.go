package symboltable

import (
	"slices"
	"sync"
	"sync/atomic"
)

type Entry[V any] struct {
	Begin uint64
	Size  uint64
	Data  V
}

type keySpace[V any] struct {
	current atomic.Pointer[[]Entry[V]]
}

// Table is a simple container which stores a set of entries (e.g. address ranges)
// belonging to different key spaces (e.g. processes). It can then find a entry, containing specified point.
// Key spaces are indexed with uint32 keys.
type Table[K ~uint32, V any] struct {
	mu     sync.RWMutex
	spaces map[K]*keySpace[V]
}

func New[K ~uint32, V any]() *Table[K, V] {
	return &Table[K, V]{
		spaces: make(map[K]*keySpace[V]),
	}
}

func (ks *keySpace[V]) reset(newEntries []Entry[V]) {
	ks.current.Store(&newEntries)
}

// Put replaces current set of entries for process pid with newEntries.
// It expects that newEntries is pairwise disjoint and sorted on Begin.
func (t *Table[K, V]) Put(ksID K, newEntries []Entry[V]) {
	var ks *keySpace[V]
	{
		t.mu.RLock()
		ks = t.spaces[ksID]
		if ks != nil {
			ks.reset(newEntries)
		}
		t.mu.RUnlock()
	}
	if ks != nil {
		return
	}
	t.mu.Lock()
	ks = t.spaces[ksID]
	if ks == nil {
		ks = new(keySpace[V])
		t.spaces[ksID] = ks
	}
	ks.reset(newEntries)
	t.mu.Unlock()
}

func (t *Table[K, V]) Remove(ksID K) {
	t.mu.Lock()
	delete(t.spaces, ksID)
	t.mu.Unlock()
}

func (t *Table[K, V]) Find(ksID K, addr uint64) (V, bool) {
	var entries []Entry[V]
	{
		t.mu.RLock()
		ks := t.spaces[ksID]
		if ks != nil {
			entries = *ks.current.Load()
		}
		t.mu.RUnlock()
	}
	pos, ok := slices.BinarySearchFunc(entries, addr, func(s Entry[V], target uint64) int {
		if target < s.Begin {
			return 1
		}
		if target >= s.Begin+s.Size {
			return -1
		}
		return 0
	})
	if !ok {
		var zero V
		return zero, false
	}
	return entries[pos].Data, true
}

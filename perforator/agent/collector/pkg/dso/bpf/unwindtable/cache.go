package unwindtable

import (
	"container/heap"
	"sync"
)

////////////////////////////////////////////////////////////////////////////////

type cache struct {
	mu sync.Mutex
	h  cacheheap
}

func (c *cache) put(a *Allocation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	heap.Push(&c.h, a)
}

func (c *cache) pop() *Allocation {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.h.Len() == 0 {
		return nil
	}
	return heap.Pop(&c.h).(*Allocation)
}

func (c *cache) remove(a *Allocation) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if a.cacheid == invalidCacheID {
		panic("Trying to remove allocation which is not cached")
	}
	_ = heap.Remove(&c.h, a.cacheid)
}

////////////////////////////////////////////////////////////////////////////////

var _ heap.Interface = (*cacheheap)(nil)

type cacheheap struct {
	items []*Allocation
}

const invalidCacheID = -1

////////////////////////////////////////////////////////////////////////////////

// Len implements heap.Interface.
func (c *cacheheap) Len() int {
	return len(c.items)
}

// Less implements heap.Interface.
func (c *cacheheap) Less(i int, j int) bool {
	return len(c.items[i].Pages) > len(c.items[j].Pages)
}

// Pop implements heap.Interface.
func (c *cacheheap) Pop() any {
	i := len(c.items) - 1
	alloc := c.items[i]
	alloc.cacheid = invalidCacheID
	c.items[i] = nil
	c.items = c.items[0:i]
	return alloc
}

// Push implements heap.Interface.
func (c *cacheheap) Push(x any) {
	alloc := x.(*Allocation)
	alloc.cacheid = len(c.items)
	c.items = append(c.items, alloc)
}

// Swap implements heap.Interface.
func (c *cacheheap) Swap(i int, j int) {
	c.items[i], c.items[j] = c.items[j], c.items[i]
	c.items[i].cacheid = i
	c.items[j].cacheid = j
}

////////////////////////////////////////////////////////////////////////////////

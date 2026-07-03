// Package boundedcache is a keyed, refcounted, size-bounded cache with
// single-flight fills. It is storage-agnostic: callers supply the value and how
// to dispose it, and the cache owns the index, the size budget, coalescing, and
// LRU eviction. Fills run on cache-owned goroutines, so they are not bound to
// the callers that started them; see GetOrFill.
package boundedcache

import (
	"container/list"
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"golang.org/x/sync/semaphore"
)

var (
	// ErrTooLarge means the request can never fit, even in an empty cache.
	ErrTooLarge = errors.New("boundedcache: item exceeds cache capacity")
	// ErrWouldBlock means a non-blocking charge can't be satisfied right now.
	ErrWouldBlock   = errors.New("boundedcache: charge would block")
	errNegativeSize = errors.New("boundedcache: negative size")
	// errAbandoned is what a fill that outlived all its waiters resolves to.
	errAbandoned = errors.New("boundedcache: fill abandoned")
)

// Cache keeps all state under one lock. Blocking budget admission is serialized
// through a capacity-1 turnstile (FIFO, so a large charge is not starved by
// smaller ones); non-blocking charges don't queue — a held turn is a queued
// charge's claim, and they yield to it. A fill charges the budget through its
// Entry for what it stores (FillFunc documents the contract); afterwards
// callers hold only a Lease.
type Cache[K comparable, V any] struct {
	turn *semaphore.Weighted // serializes blocking charges (FIFO)

	// mu guards all mutable state of the cache and its entries, except
	// Entry.size (atomic, see below); it is not held during fills, waits,
	// or onEvict.
	mu       sync.Mutex
	cap      int64
	used     int64
	byKey    map[K]*Entry[K, V]
	idle     list.List     // unreferenced ready entries, LRU; front = coldest
	idleSize int64         // total size of entries in idle (the reclaimable budget)
	changed  chan struct{} // when non-nil, closed as space frees (the active charge waits on it)

	onEvict func(V) // disposes an evicted (or rejected) value; may be nil
}

// Entry is the in-flight slot a fill charges the budget through via
// Charge/TryCharge; the rest is the Cache's bookkeeping. Once done is closed,
// err==nil means the value is ready and err!=nil means the fill failed.
type Entry[K comparable, V any] struct {
	c     *Cache[K, V]
	key   K
	value V
	size  atomic.Int64 // charged budget; grows during the fill, read lock-free on the fast path, written under c.mu (paired with Cache.used)
	refs  int
	done  chan struct{} // closed when the fill finishes
	err   error
	elem  *list.Element // in c.idle iff refs==0 and the entry is ready

	fillCancel context.CancelFunc // aborts the fill; set for the fill's lifetime
}

// Lease is a hold on a ready value: it is not evicted while a Lease to it is open.
type Lease[K comparable, V any] struct {
	e    *Entry[K, V]
	once sync.Once
}

// FillFunc produces the value for a missing key, charging the budget through
// the Entry for what it stores — up front via Charge, for guaranteed space, or
// as it goes via TryCharge. The charge is the entry's size.
//
// A fill runs on a cache-owned goroutine and may outlive the GetOrFill call that
// started it (its result is shared with coalesced callers). It must not capture
// state scoped to the caller; anything request-scoped it needs, it acquires and
// releases itself. The Entry is valid only until the fill returns: the charge
// grows monotonically while it lives, which is what makes the lock-free
// "already charged" checks below sound.
type FillFunc[K comparable, V any] func(ctx context.Context, e *Entry[K, V]) (value V, err error)

// New returns a cache bounded to capacity, counted in whatever size units the
// caller measures values in (must be positive). Evicted values are disposed
// through onEvict (may be nil), which is called without the cache lock held, so
// it may block.
func New[K comparable, V any](capacity int64, onEvict func(V)) *Cache[K, V] {
	if capacity <= 0 {
		panic("boundedcache: capacity must be positive")
	}
	return &Cache[K, V]{
		turn:    semaphore.NewWeighted(1),
		cap:     capacity,
		byKey:   make(map[K]*Entry[K, V]),
		onEvict: onEvict,
	}
}

func (c *Cache[K, V]) Cap() int64 { return c.cap }

func (c *Cache[K, V]) Used() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.used
}

func (c *Cache[K, V]) Len() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return int64(len(c.byKey))
}

// GetOrFill returns a Lease on the value for key. On a miss it starts fill on a
// cache-owned goroutine and context (values flow from ctx, cancellation does
// not), then waits like any other caller: cancellably, with its own ctx. The
// fill is kept alive while at least one caller waits for it and is canceled when
// the last one gives up. A failed fill's error is returned to every waiter
// as-is; the cache does not retry — budget contention is resolved by Charge
// blocking, so any error a fill returns is external, and retry policy belongs to
// the caller.
func (c *Cache[K, V]) GetOrFill(ctx context.Context, key K, fill FillFunc[K, V]) (*Lease[K, V], error) {
	c.mu.Lock()
	e := c.byKey[key]
	if e == nil {
		fctx, cancel := context.WithCancel(context.WithoutCancel(ctx))
		e = &Entry[K, V]{c: c, key: key, refs: 1, done: make(chan struct{}), fillCancel: cancel}
		c.byKey[key] = e
		go func() {
			defer cancel()
			c.runFill(fctx, e, fill)
		}()
	} else {
		c.ref(e)
	}
	c.mu.Unlock()

	if err := e.wait(ctx); err != nil {
		c.mu.Lock()
		c.unref(e)
		abandoned := e.refs == 0 && !e.ready() // this call was the fill's last waiter
		c.mu.Unlock()
		if abandoned {
			e.fillCancel()
		}
		return nil, err
	}
	return &Lease[K, V]{e: e}, nil
}

// Charge raises the entry's charge to at least hi, waiting its turn (FIFO) and
// then blocking until enough idle entries free up. ctx-cancellable. A charge
// the entry already owns is a no-op that never blocks.
func (e *Entry[K, V]) Charge(ctx context.Context, hi int64) error {
	c := e.c
	if ok, err := e.needsGrow(hi); !ok {
		return err
	}
	if err := c.turn.Acquire(ctx, 1); err != nil {
		return err
	}
	defer c.turn.Release(1)

	c.mu.Lock()
	for {
		evicted, ok := c.grow(e, hi)
		if ok {
			c.mu.Unlock()
			c.discard(evicted...)
			return nil
		}
		if c.changed == nil {
			c.changed = make(chan struct{})
		}
		ch := c.changed
		c.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return ctx.Err()
		}
		c.mu.Lock()
	}
}

// TryCharge is the non-blocking Charge. TryCharges serialize on the cache lock,
// so a failed turn acquisition means a blocking charge is queued and the space
// is its; otherwise ErrWouldBlock means the space genuinely isn't reclaimable
// right now.
func (e *Entry[K, V]) TryCharge(hi int64) error {
	c := e.c
	if ok, err := e.needsGrow(hi); !ok {
		return err
	}
	c.mu.Lock()
	// mu→turn inverts Charge's turn→mu order; that is deadlock-free only
	// because this acquisition never waits.
	if !c.turn.TryAcquire(1) {
		c.mu.Unlock()
		return ErrWouldBlock
	}
	evicted, ok := c.grow(e, hi)
	c.turn.Release(1)
	c.mu.Unlock()
	c.discard(evicted...)
	if !ok {
		return ErrWouldBlock
	}
	return nil
}

// Value returns the held value; Size its size; Release drops the hold. value
// and size are frozen once the Lease exists, so neither needs the lock.
func (l *Lease[K, V]) Value() V    { return l.e.value }
func (l *Lease[K, V]) Size() int64 { return l.e.size.Load() }

func (l *Lease[K, V]) Release() {
	l.once.Do(func() {
		c := l.e.c
		c.mu.Lock()
		c.unref(l.e) // never abandons: a Lease holds a ready entry
		c.mu.Unlock()
	})
}

func (c *Cache[K, V]) runFill(ctx context.Context, e *Entry[K, V], fill FillFunc[K, V]) {
	value, err := fill(ctx, e)
	if err != nil {
		c.mu.Lock()
		c.fail(e, err)
		c.mu.Unlock()
		return
	}
	if !c.publish(e, value) {
		c.discard(value)
	}
}

// publish hands the value to waiters, under one lock. It reports false for an
// entry no longer in the index (abandoned while the fill raced to completion):
// such an entry could never be reached again, so it is refunded and its value
// is left for the caller to discard.
func (c *Cache[K, V]) publish(e *Entry[K, V], value V) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byKey[e.key] != e { // abandoned: only unref removes an in-flight entry
		c.fail(e, errAbandoned)
		return false
	}
	e.value = value
	close(e.done)
	return true
}

// fail aborts an in-flight entry: refunds its charge, drops it from the index
// (unless a successor already claimed the key), and publishes err to waiters.
// Must hold c.mu.
func (c *Cache[K, V]) fail(e *Entry[K, V], err error) {
	c.used -= e.size.Load()
	if c.byKey[e.key] == e {
		delete(c.byKey, e.key)
	}
	e.err = err
	close(e.done)
	c.wake()
}

// needsGrow validates the watermark and reports whether the charge must grow to
// own it; false with a nil error is the lock-free fast path (hi already
// charged), which also skips the turn. The answer is advisory — grow re-checks
// under c.mu — and safely so in both directions: the charge only grows while
// the Entry lives, so "charged" can't be revoked, and a stale "must grow" just
// takes the lock to find nothing to do.
func (e *Entry[K, V]) needsGrow(hi int64) (bool, error) {
	switch {
	case hi < 0:
		return false, errNegativeSize
	case hi > e.c.cap:
		return false, ErrTooLarge
	case hi <= e.size.Load():
		return false, nil
	}
	return true, nil
}

// grow raises the entry's charge to hi, evicting idle entries as needed and
// returning them for disposal after unlocking. It is the authoritative half of
// the needsGrow double-check: concurrent charges serialize here, and a
// watermark the entry already owns succeeds without evicting. The caller must
// hold the turn and c.mu; hi must not exceed the capacity.
func (c *Cache[K, V]) grow(e *Entry[K, V], hi int64) (evicted []V, ok bool) {
	need := hi - e.size.Load()
	if need <= 0 {
		return nil, true
	}
	evicted, ok = c.makeRoom(need)
	if ok {
		c.used += need
		e.size.Store(hi)
	}
	return evicted, ok
}

// makeRoom evicts idle entries (coldest first) until need more size fits,
// returning their values for the caller to dispose after unlocking. It first
// checks that the reclaimable budget suffices, so a charge that can't be
// satisfied evicts nothing. Must hold c.mu.
func (c *Cache[K, V]) makeRoom(need int64) (evicted []V, ok bool) {
	if c.used-c.idleSize+need > c.cap {
		return nil, false
	}
	for c.used+need > c.cap {
		e := c.idle.Front().Value.(*Entry[K, V])
		c.unpark(e)
		delete(c.byKey, e.key)
		c.used -= e.size.Load()
		evicted = append(evicted, e.value)
	}
	return evicted, true
}

// ref adds a hold, unparking an idle entry. Must hold c.mu.
func (c *Cache[K, V]) ref(e *Entry[K, V]) {
	if e.elem != nil { // parked implies refs == 0
		c.unpark(e)
	}
	e.refs++
}

// unref drops a hold: the last hold parks a ready entry for reclaim, while an
// in-flight fill whose last waiter left is abandoned — dropped from the index,
// so callers arriving from now on start a fresh fill and never observe the
// doomed one. Canceling the abandoned fill is on the caller, outside the lock;
// only GetOrFill can observe that state, since a Lease always holds a ready
// entry. Must hold c.mu.
func (c *Cache[K, V]) unref(e *Entry[K, V]) {
	if e.err != nil {
		return // the fill failed; nothing is cached
	}
	e.refs--
	if e.refs > 0 {
		return
	}
	if e.ready() {
		c.park(e)
		c.wake()
		return
	}
	delete(c.byKey, e.key)
}

// park puts an unreferenced ready entry on the reclaim list; unpark takes it
// back off. They keep idle and idleSize in step. Must hold c.mu.
func (c *Cache[K, V]) park(e *Entry[K, V]) {
	e.elem = c.idle.PushBack(e)
	c.idleSize += e.size.Load()
}

func (c *Cache[K, V]) unpark(e *Entry[K, V]) {
	c.idle.Remove(e.elem)
	e.elem = nil
	c.idleSize -= e.size.Load()
}

// wait blocks until the fill finishes: nil if it produced the value, else the
// fill's error. err is set before done is closed, so no lock is needed. A ready
// entry (done already closed) resolves even under a canceled ctx — a hit never
// fails for cancellation.
func (e *Entry[K, V]) wait(ctx context.Context) error {
	select {
	case <-e.done:
		return e.err
	default:
	}
	select {
	case <-e.done:
		return e.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *Entry[K, V]) ready() bool {
	select {
	case <-e.done:
		return e.err == nil
	default:
		return false
	}
}

// discard disposes values via onEvict, off the cache lock so it can take its time
// (onEvict owns its own error handling).
func (c *Cache[K, V]) discard(values ...V) {
	if c.onEvict == nil {
		return
	}
	for _, v := range values {
		c.onEvict(v)
	}
}

// wake re-checks the charge currently holding the turn, if one is waiting.
// Must hold c.mu.
func (c *Cache[K, V]) wake() {
	if c.changed != nil {
		close(c.changed)
		c.changed = nil
	}
}

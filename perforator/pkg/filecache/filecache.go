// Package filecache is a refcounted, size-bounded on-disk cache for large
// artifacts (gsym/DWARF binaries) downloaded once and then mmap'd.
//
//   - GetOrFill returns a guarding Ref; a file is never evicted while any Ref to
//     it is open. A missing entry is produced once (single-flight) by a Fill.
//   - A fill writes through a Writer. Charge blocks until budget is owned (FIFO,
//     so a large download isn't starved); a bare WriteAt charges as it grows,
//     best-effort. The charge is the entry's size.
//
// The keys, byte budget, single-flight, and eviction live in a generic
// boundedcache.Cache; this layer is its file binding — the value is the cached
// file's path, and onEvict deletes it.
package filecache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"

	"github.com/dustin/go-humanize"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/pkg/atomicfs"
	"github.com/yandex/perforator/perforator/pkg/boundedcache"
)

var (
	// ErrTooLarge means the request can never fit, even in an empty cache.
	ErrTooLarge = boundedcache.ErrTooLarge
	// ErrWouldBlock means a non-blocking charge can't be satisfied right now.
	ErrWouldBlock = boundedcache.ErrWouldBlock
)

type Config struct {
	RootPath string `yaml:"root_path"`
	MaxSize  string `yaml:"max_size"`
}

// FillFunc produces a missing entry by writing it to w, which is valid only
// until the fill returns. It may outlive the GetOrFill call that started it and
// must not capture caller-scoped state (see boundedcache.FillFunc).
type FillFunc func(ctx context.Context, w Writer) error

// Writer is what a FillFunc writes through.
type Writer interface {
	io.WriterAt
	// Charge blocks until this fill owns budget up to hi (ctx cancels).
	Charge(ctx context.Context, hi int64) error
}

type cacheMetrics struct {
	hits, misses, evictions, fillErrors, noSpace, discardErrors metrics.Counter
}

type Cache struct {
	dir     string
	alloc   *boundedcache.Cache[string, string] // value = the cached file's path
	metrics cacheMetrics
}

func NewFileCache(cfg *Config, reg metrics.Registry) (*Cache, error) {
	capacity, err := humanize.ParseBytes(cfg.MaxSize)
	if err != nil {
		return nil, fmt.Errorf("filecache: invalid max_size %q: %w", cfg.MaxSize, err)
	}
	return newCache(cfg.RootPath, int64(capacity), reg)
}

func newCache(root string, capacity int64, reg metrics.Registry) (*Cache, error) {
	if capacity <= 0 {
		return nil, errors.New("filecache: capacity must be greater than zero")
	}
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("filecache: failed to create root %q: %w", root, err)
	}
	c := &Cache{dir: root}
	c.alloc = boundedcache.New[string, string](capacity, c.evict)
	c.registerMetrics(reg)
	if err := c.loadExisting(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Cache) Dir() string { return c.dir }

// evict is the cache's onEvict hook: it deletes the file and records the outcome.
// A missing file is not a failure.
func (c *Cache) evict(path string) {
	c.metrics.evictions.Inc()
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		c.metrics.discardErrors.Inc()
	}
}

func (c *Cache) registerMetrics(reg metrics.Registry) {
	if reg == nil {
		reg = nop.Registry{}
	}
	reg = reg.WithPrefix("filecache")
	c.metrics.hits = reg.Counter("hits")
	c.metrics.misses = reg.Counter("misses")
	c.metrics.evictions = reg.Counter("evictions")
	c.metrics.fillErrors = reg.Counter("fill_errors")
	c.metrics.noSpace = reg.Counter("no_space_errors")
	c.metrics.discardErrors = reg.Counter("discard_errors")

	reg.FuncIntGauge("capacity_bytes", func() int64 { return c.alloc.Cap() })
	reg.FuncIntGauge("used_bytes", func() int64 { return c.alloc.Used() })
	reg.FuncIntGauge("entries", func() int64 { return c.alloc.Len() })
}

// GetOrFill returns a guarding Ref for key, producing it once via fill on a miss.
// The fill runs on a cache-owned goroutine: it is kept alive by whoever waits for
// it, not by this caller's ctx, and is canceled when the last waiter gives up.
func (c *Cache) GetOrFill(ctx context.Context, key string, fill FillFunc) (*Ref, error) {
	// filled is written on the fill goroutine before the entry resolves and read
	// only after a successful wait, which synchronizes the two.
	filled := false
	l, err := c.alloc.GetOrFill(ctx, key, func(ctx context.Context, e *boundedcache.Entry[string, string]) (string, error) {
		filled = true
		path, err := c.materialize(ctx, key, e, fill)
		if err != nil {
			c.metrics.fillErrors.Inc()
			if errors.Is(err, syscall.ENOSPC) {
				c.metrics.noSpace.Inc()
			}
		}
		return path, err
	})
	if err != nil {
		return nil, err
	}
	if filled {
		c.metrics.misses.Inc()
	} else {
		c.metrics.hits.Inc()
	}
	return &Ref{lease: l}, nil
}

// materialize runs the fill through atomicfs (temp file → fsync → atomic rename
// into place), yielding the cached file's path. Nothing reaches the final path
// on failure.
func (c *Cache) materialize(ctx context.Context, key string, e *boundedcache.Entry[string, string], fill FillFunc) (string, error) {
	dst := filepath.Join(c.dir, key)
	f, err := atomicfs.Create(dst, atomicfs.WithSync())
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Discard() }() // no-op after a successful Close

	if err := fill(ctx, &writer{e: e, f: f}); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return dst, nil
}

// loadExisting rebuilds the index from disk: leftover temp files are partial
// fills (deleted); final files are trusted (renamed only after fsync) and enter
// the cache as ready, reclaimable items via the normal fill path.
func (c *Cache) loadExisting() error {
	dirents, err := os.ReadDir(c.dir)
	if err != nil {
		return fmt.Errorf("filecache: failed to scan root %q: %w", c.dir, err)
	}
	for _, de := range dirents {
		if de.IsDir() {
			continue
		}
		path := filepath.Join(c.dir, de.Name())
		if atomicfs.IsTmp(de.Name()) {
			_ = os.Remove(path) // partial fill
			continue
		}
		info, err := de.Info()
		if err != nil {
			continue
		}
		size := info.Size()
		l, err := c.alloc.GetOrFill(context.Background(), de.Name(), func(ctx context.Context, e *boundedcache.Entry[string, string]) (string, error) {
			if err := e.Charge(ctx, size); err != nil {
				return "", err
			}
			return path, nil
		})
		if err != nil {
			_ = os.Remove(path) // no longer fits
			continue
		}
		l.Release() // ready and now reclaimable
	}
	return nil
}

// writer is the Writer handed to a FillFunc: WriteAt goes to the temp file,
// charging the budget as it grows. The entry's charge is lock-free on the fast
// path, so a WriteAt within it takes no cache lock.
type writer struct {
	e *boundedcache.Entry[string, string]
	f *atomicfs.File
}

func (w *writer) Charge(ctx context.Context, n int64) error { return w.e.Charge(ctx, n) }

func (w *writer) WriteAt(p []byte, off int64) (int, error) {
	if err := w.e.TryCharge(off + int64(len(p))); err != nil {
		return 0, err
	}
	return w.f.WriteAt(p, off)
}

// Ref is a guard on a cached file: the file is not evicted while it is open.
type Ref struct {
	lease *boundedcache.Lease[string, string]
}

func (r *Ref) Path() string { return r.lease.Value() }
func (r *Ref) Size() int64  { return r.lease.Size() }
func (r *Ref) Release()     { r.lease.Release() }

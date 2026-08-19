package linetable

import (
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/karlseguin/ccache/v3"
)

// CacheKey deduplicates parsed location tables per sampled code object state.
type CacheKey struct {
	Pid            uint32
	CodeObjectPtr  uint64
	CoLinetablePtr uint64
	CoFirstlineno  int32
}

const (
	// DefaultMaxSize is the default byte budget for cached co_linetable payloads.
	DefaultMaxSize = 16 << 20 // 16 MiB
	// DefaultCacheTTL is used when Config.TTL is zero.
	DefaultCacheTTL = 5 * time.Minute
	// cacheEntryCost is charged for every entry on top of len(Raw) so empty
	// tables and tombstones still consume budget.
	cacheEntryCost = 64
)

// Config configures the linetable LRU. Empty MaxSize / zero TTL use defaults.
type Config struct {
	// MaxSize is a humanize-parseable byte budget, e.g. "16MB".
	MaxSize string        `yaml:"max_size"`
	TTL     time.Duration `yaml:"ttl"`
}

// cacheValue wraps LocationTable so ccache can charge Size() against MaxSize.
type cacheValue struct {
	LocationTable
}

func (v *cacheValue) Size() int64 {
	return int64(len(v.Raw)) + cacheEntryCost
}

// Cache is a byte-budget LRU of LocationTable keyed by CacheKey, backed by
// ccache, with per-entry TTL.
type Cache struct {
	inner    *ccache.Cache[*cacheValue]
	budget   int64
	ttl      time.Duration
	stopOnce sync.Once
}

// NewCache builds a cache from cfg. Invalid MaxSize returns an error.
func NewCache(cfg Config) (*Cache, error) {
	budget := int64(DefaultMaxSize)
	if cfg.MaxSize != "" && cfg.MaxSize != "0" {
		n, err := humanize.ParseBytes(cfg.MaxSize)
		if err != nil {
			return nil, fmt.Errorf("linetable cache: invalid max_size %q: %w", cfg.MaxSize, err)
		}
		if n == 0 {
			budget = DefaultMaxSize
		} else {
			budget = int64(n)
		}
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	return newCache(budget, ttl), nil
}

func newCache(budget int64, ttl time.Duration) *Cache {
	if budget <= 0 {
		budget = DefaultMaxSize
	}
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	inner := ccache.New(ccache.Configure[*cacheValue]().
		MaxSize(budget).
		GetsPerPromote(1))
	return &Cache{
		inner:  inner,
		budget: budget,
		ttl:    ttl,
	}
}

func cacheKeyString(key CacheKey) string {
	// Fixed-width binary key avoids fmt.Sprintf allocations on the hot path.
	var b [24]byte
	binary.LittleEndian.PutUint32(b[0:4], key.Pid)
	binary.LittleEndian.PutUint64(b[4:12], key.CodeObjectPtr)
	binary.LittleEndian.PutUint64(b[12:20], key.CoLinetablePtr)
	binary.LittleEndian.PutUint32(b[20:24], uint32(key.CoFirstlineno))
	return string(b[:])
}

func (c *Cache) Get(key CacheKey) (LocationTable, bool) {
	s := cacheKeyString(key)
	item := c.inner.Get(s)
	if item == nil {
		return LocationTable{}, false
	}
	if item.Expired() {
		c.inner.Delete(s)
		return LocationTable{}, false
	}
	return item.Value().LocationTable, true
}

// Add stores table under key and takes ownership of table.Raw; do not mutate
// the backing array after Add returns.
func (c *Cache) Add(key CacheKey, table LocationTable) {
	size := int64(len(table.Raw)) + cacheEntryCost
	if size > c.budget {
		return
	}
	c.inner.Set(cacheKeyString(key), &cacheValue{LocationTable: table}, c.ttl)
}

// AddTombstone records a negative-cache entry so Get hits without another
// remote read of co_linetable.
func (c *Cache) AddTombstone(key CacheKey) {
	c.Add(key, LocationTable{Unresolvable: true})
}

func (c *Cache) InvalidatePid(pid uint32) {
	if c == nil {
		return
	}
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], pid)
	c.inner.DeletePrefix(string(b[:]))
}

// Stop shuts down the ccache worker. The Cache must not be used afterward.
func (c *Cache) Stop() {
	if c == nil {
		return
	}
	c.stopOnce.Do(c.inner.Stop)
}

func (c *Cache) Len() int {
	return c.inner.ItemCount()
}

// Cap returns the configured byte budget.
func (c *Cache) Cap() int64 {
	return c.budget
}

// Used returns the sum of charged entry sizes under the budget.
func (c *Cache) Used() int64 {
	return c.inner.GetSize()
}

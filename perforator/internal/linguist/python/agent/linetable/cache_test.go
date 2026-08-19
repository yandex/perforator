package linetable

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func newTestCache(t *testing.T, budget int64, ttl time.Duration) *Cache {
	t.Helper()
	c := newCache(budget, ttl)
	t.Cleanup(c.Stop)
	return c
}

func TestCache_GetAdd(t *testing.T) {
	c := newTestCache(t, 256, time.Minute)
	key := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 10}
	table := LocationTable{FirstLineno: 10, Raw: []byte{0x80, 0x01}}

	_, ok := c.Get(key)
	require.False(t, ok)

	c.Add(key, table)
	got, ok := c.Get(key)
	require.True(t, ok)
	require.Equal(t, table, got)
}

func TestCache_KeyFieldsDistinct(t *testing.T) {
	c := newTestCache(t, 512, time.Minute)

	k1 := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 10}
	k2 := CacheKey{Pid: 2, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 10} // different pid
	k3 := CacheKey{Pid: 1, CodeObjectPtr: 0x3000, CoLinetablePtr: 0x2000, CoFirstlineno: 10} // different code object
	k4 := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x4000, CoFirstlineno: 10} // different linetable
	k5 := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 20} // different firstlineno

	c.Add(k1, LocationTable{FirstLineno: 10, Raw: []byte{0x80}})
	c.Add(k2, LocationTable{FirstLineno: 10, Raw: []byte{0x90}})
	c.Add(k3, LocationTable{FirstLineno: 10, Raw: []byte{0xa0}})
	c.Add(k4, LocationTable{FirstLineno: 10, Raw: []byte{0xb0}})
	c.Add(k5, LocationTable{FirstLineno: 20, Raw: []byte{0xc0}})

	got, ok := c.Get(k1)
	require.True(t, ok)
	require.Equal(t, []byte{0x80}, got.Raw)
	got, ok = c.Get(k2)
	require.True(t, ok)
	require.Equal(t, []byte{0x90}, got.Raw)
	got, ok = c.Get(k3)
	require.True(t, ok)
	require.Equal(t, []byte{0xa0}, got.Raw)
	got, ok = c.Get(k4)
	require.True(t, ok)
	require.Equal(t, []byte{0xb0}, got.Raw)
	got, ok = c.Get(k5)
	require.True(t, ok)
	require.Equal(t, []byte{0xc0}, got.Raw)
}

func TestCache_KeySerializationIsFixedWidth(t *testing.T) {
	key := CacheKey{Pid: 1, CodeObjectPtr: 2, CoLinetablePtr: 3, CoFirstlineno: 4}
	require.Len(t, cacheKeyString(key), 24)
}

func TestCache_OversizedRejected(t *testing.T) {
	c := newTestCache(t, cacheEntryCost, time.Minute)
	key := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 1}
	c.Add(key, LocationTable{FirstLineno: 1, Raw: []byte{1, 2, 3, 4, 5}})
	_, ok := c.Get(key)
	require.False(t, ok)
}

func TestCache_Defaults(t *testing.T) {
	c, err := NewCache(Config{})
	require.NoError(t, err)
	t.Cleanup(c.Stop)
	require.Equal(t, int64(DefaultMaxSize), c.Cap())
	require.Equal(t, DefaultCacheTTL, c.ttl)

	c, err = NewCache(Config{MaxSize: "1KB"})
	require.NoError(t, err)
	t.Cleanup(c.Stop)
	require.Equal(t, int64(1000), c.Cap())
}

func TestCache_InvalidMaxSize(t *testing.T) {
	_, err := NewCache(Config{MaxSize: "not-a-size"})
	require.Error(t, err)
}

func TestCache_StopIdempotent(t *testing.T) {
	c := newCache(256, time.Minute)
	c.Stop()
	c.Stop()
}

func TestCache_TTLExpiry(t *testing.T) {
	c := newTestCache(t, 256, 20*time.Millisecond)
	key := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 1}
	c.Add(key, LocationTable{FirstLineno: 1, Raw: []byte{0x80}})

	got, ok := c.Get(key)
	require.True(t, ok)
	require.Equal(t, []byte{0x80}, got.Raw)

	time.Sleep(40 * time.Millisecond)
	_, ok = c.Get(key)
	require.False(t, ok, "expired entry must miss")
}

func TestCache_Tombstone(t *testing.T) {
	c := newTestCache(t, 256, time.Minute)
	key := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0xbeef, CoFirstlineno: 10}

	c.AddTombstone(key)
	got, ok := c.Get(key)
	require.True(t, ok)
	require.True(t, got.Unresolvable)
	require.Nil(t, got.Raw)

	c.Add(key, LocationTable{FirstLineno: 10, Raw: []byte{0x80}})
	got, ok = c.Get(key)
	require.True(t, ok)
	require.False(t, got.Unresolvable)
	require.Equal(t, []byte{0x80}, got.Raw)
}

func TestCache_InvalidatePid(t *testing.T) {
	c := newTestCache(t, 512, time.Minute)

	keep := CacheKey{Pid: 1, CodeObjectPtr: 0x1000, CoLinetablePtr: 0x2000, CoFirstlineno: 10}
	drop1 := CacheKey{Pid: 2, CodeObjectPtr: 0x3000, CoLinetablePtr: 0x4000, CoFirstlineno: 20}
	drop2 := CacheKey{Pid: 2, CodeObjectPtr: 0x5000, CoLinetablePtr: 0x6000, CoFirstlineno: 30}
	tomb := CacheKey{Pid: 2, CodeObjectPtr: 0x7000, CoLinetablePtr: 0x8000, CoFirstlineno: 40}

	c.Add(keep, LocationTable{FirstLineno: 10, Raw: []byte{0x80}})
	c.Add(drop1, LocationTable{FirstLineno: 20, Raw: []byte{0x90}})
	c.Add(drop2, LocationTable{FirstLineno: 30, Raw: []byte{0xa0}})
	c.AddTombstone(tomb)

	c.InvalidatePid(2)

	got, ok := c.Get(keep)
	require.True(t, ok)
	require.Equal(t, []byte{0x80}, got.Raw)

	_, ok = c.Get(drop1)
	require.False(t, ok)
	_, ok = c.Get(drop2)
	require.False(t, ok)
	_, ok = c.Get(tomb)
	require.False(t, ok)
}

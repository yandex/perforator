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
	key := CacheKey{Pid: 1, CoLinetablePtr: 0x1000, CoFirstlineno: 10}
	table := LocationTable{FirstLineno: 10, Raw: []byte{0x80, 0x01}}

	_, ok := c.Get(key)
	require.False(t, ok)

	c.Add(key, table)
	got, ok := c.Get(key)
	require.True(t, ok)
	require.Equal(t, table, got)
}

func TestCache_KeyFieldsDistinct(t *testing.T) {
	c := newTestCache(t, 256, time.Minute)

	k1 := CacheKey{Pid: 1, CoLinetablePtr: 0x1000, CoFirstlineno: 10}
	k2 := CacheKey{Pid: 1, CoLinetablePtr: 0x2000, CoFirstlineno: 10} // different ptr
	k3 := CacheKey{Pid: 1, CoLinetablePtr: 0x1000, CoFirstlineno: 20} // different firstlineno

	c.Add(k1, LocationTable{FirstLineno: 10, Raw: []byte{0x80}})
	c.Add(k2, LocationTable{FirstLineno: 10, Raw: []byte{0x90}})
	c.Add(k3, LocationTable{FirstLineno: 20, Raw: []byte{0xa0}})

	got, ok := c.Get(k1)
	require.True(t, ok)
	require.Equal(t, []byte{0x80}, got.Raw)
	got, ok = c.Get(k2)
	require.True(t, ok)
	require.Equal(t, []byte{0x90}, got.Raw)
	got, ok = c.Get(k3)
	require.True(t, ok)
	require.Equal(t, []byte{0xa0}, got.Raw)
}

func TestCache_OversizedRejected(t *testing.T) {
	c := newTestCache(t, cacheEntryCost, time.Minute)
	key := CacheKey{Pid: 1, CoLinetablePtr: 0x1000, CoFirstlineno: 1}
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

func TestCache_TTLExpiry(t *testing.T) {
	c := newTestCache(t, 256, 20*time.Millisecond)
	key := CacheKey{Pid: 1, CoLinetablePtr: 0x1000, CoFirstlineno: 1}
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
	key := CacheKey{Pid: 1, CoLinetablePtr: 0xbeef, CoFirstlineno: 10}

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

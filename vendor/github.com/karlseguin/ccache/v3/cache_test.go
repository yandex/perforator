package ccache

import (
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/karlseguin/ccache/v3/assert"
)

func Test_Setnx(t *testing.T) {
	cache := New(Configure[string]())
	defer cache.Stop()
	assert.Equal(t, cache.ItemCount(), 0)

	cache.Set("spice", "flow", time.Minute)
	assert.Equal(t, cache.ItemCount(), 1)

	// set if exists
	cache.Setnx("spice", "worm", time.Minute)
	assert.Equal(t, cache.ItemCount(), 1)
	assert.Equal(t, cache.Get("spice").Value(), "flow")

	// set if not exists
	cache.Delete("spice")
	cache.Setnx("spice", "worm", time.Minute)
	assert.Equal(t, cache.Get("spice").Value(), "worm")

	assert.Equal(t, cache.ItemCount(), 1)
}

func Test_Extend(t *testing.T) {
	cache := New(Configure[string]())
	defer cache.Stop()
	assert.Equal(t, cache.ItemCount(), 0)

	// non exist
	ok := cache.Extend("spice", time.Minute*10)
	assert.Equal(t, false, ok)

	// exist
	cache.Set("spice", "flow", time.Minute)
	assert.Equal(t, cache.ItemCount(), 1)

	ok = cache.Extend("spice", time.Minute*10) // 10 + 10
	assert.Equal(t, true, ok)

	item := cache.Get("spice")
	less := time.Minute*22 < time.Duration(item.expires)
	assert.Equal(t, true, less)
	more := time.Minute*18 < time.Duration(item.expires)
	assert.Equal(t, true, more)

	assert.Equal(t, cache.ItemCount(), 1)
}

func Test_CacheDeletesAValue(t *testing.T) {
	cache := New(Configure[string]())
	defer cache.Stop()
	assert.Equal(t, cache.ItemCount(), 0)

	cache.Set("spice", "flow", time.Minute)
	cache.Set("worm", "sand", time.Minute)
	assert.Equal(t, cache.ItemCount(), 2)

	cache.Delete("spice")
	assert.Equal(t, cache.Get("spice"), nil)
	assert.Equal(t, cache.Get("worm").Value(), "sand")
	assert.Equal(t, cache.ItemCount(), 1)
}

func Test_CacheDeletesAPrefix(t *testing.T) {
	cache := New(Configure[string]())
	defer cache.Stop()
	assert.Equal(t, cache.ItemCount(), 0)

	cache.Set("aaa", "1", time.Minute)
	cache.Set("aab", "2", time.Minute)
	cache.Set("aac", "3", time.Minute)
	cache.Set("ac", "4", time.Minute)
	cache.Set("z5", "7", time.Minute)
	assert.Equal(t, cache.ItemCount(), 5)

	assert.Equal(t, cache.DeletePrefix("9a"), 0)
	assert.Equal(t, cache.ItemCount(), 5)

	assert.Equal(t, cache.DeletePrefix("aa"), 3)
	assert.Equal(t, cache.Get("aaa"), nil)
	assert.Equal(t, cache.Get("aab"), nil)
	assert.Equal(t, cache.Get("aac"), nil)
	assert.Equal(t, cache.Get("ac").Value(), "4")
	assert.Equal(t, cache.Get("z5").Value(), "7")
	assert.Equal(t, cache.ItemCount(), 2)
}

func Test_CacheDeletesAFunc(t *testing.T) {
	cache := New(Configure[int]())
	defer cache.Stop()
	assert.Equal(t, cache.ItemCount(), 0)

	cache.Set("a", 1, time.Minute)
	cache.Set("b", 2, time.Minute)
	cache.Set("c", 3, time.Minute)
	cache.Set("d", 4, time.Minute)
	cache.Set("e", 5, time.Minute)
	cache.Set("f", 6, time.Minute)
	assert.Equal(t, cache.ItemCount(), 6)

	assert.Equal(t, cache.DeleteFunc(func(key string, item *Item[int]) bool {
		return false
	}), 0)
	assert.Equal(t, cache.ItemCount(), 6)

	assert.Equal(t, cache.DeleteFunc(func(key string, item *Item[int]) bool {
		return item.Value() < 4
	}), 3)
	assert.Equal(t, cache.ItemCount(), 3)

	assert.Equal(t, cache.DeleteFunc(func(key string, item *Item[int]) bool {
		return key == "d"
	}), 1)
	assert.Equal(t, cache.ItemCount(), 2)
}

func Test_CacheOnDeleteCallbackCalled(t *testing.T) {
	onDeleteFnCalled := int32(0)
	onDeleteFn := func(item *Item[string]) {
		if item.key == "spice" {
			atomic.AddInt32(&onDeleteFnCalled, 1)
		}
	}

	cache := New(Configure[string]().OnDelete(onDeleteFn))
	defer cache.Stop()

	cache.Set("spice", "flow", time.Minute)
	cache.Set("worm", "sand", time.Minute)

	cache.SyncUpdates() // wait for worker to pick up preceding updates

	cache.Delete("spice")
	cache.SyncUpdates()

	assert.Equal(t, cache.Get("spice"), nil)
	assert.Equal(t, cache.Get("worm").Value(), "sand")
	assert.Equal(t, atomic.LoadInt32(&onDeleteFnCalled), 1)
}

func Test_CacheSingleItemSizeAccounting(t *testing.T) {
	cache := New(Configure[*SizedItem]().GetsPerPromote(1))
	defer cache.Stop()

	// Add a single item to the cache
	cache.Set("only", &SizedItem{1, 10}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 10)

	// Get the item to trigger promotion
	// With a single item, next==nil && prev==nil, so it may be
	// incorrectly treated as a "new" item and size double-counted
	cache.Get("only")
	cache.Get("only")
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 10) // Should still be 10, not 20 or 30
}

func Test_CacheSingleItemDelete(t *testing.T) {
	onDeleteCalled := int32(0)
	onDeleteFn := func(item *Item[string]) {
		atomic.AddInt32(&onDeleteCalled, 1)
	}

	cache := New(Configure[string]().OnDelete(onDeleteFn))
	defer cache.Stop()

	// Add a single item
	cache.Set("only", "value", time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 1)

	// Delete the only item in the cache
	// With a single item, next==nil && prev==nil, so the delete
	// path may incorrectly skip size decrement and onDelete callback
	cache.Delete("only")
	cache.SyncUpdates()

	assert.Equal(t, cache.Get("only"), nil)
	assert.Equal(t, atomic.LoadInt32(&onDeleteCalled), 1) // onDelete should be called
	assert.Equal(t, cache.GetSize(), 0)                   // Size should be 0
}

func Test_CacheFetchesExpiredItems(t *testing.T) {
	cache := New(Configure[string]())
	defer cache.Stop()

	fn := func() (string, error) { return "moo-moo", nil }

	cache.Set("beef", "moo", time.Second*-1)
	assert.Equal(t, cache.Get("beef").Value(), "moo")

	out, _ := cache.Fetch("beef", time.Second, fn)
	assert.Equal(t, out.Value(), "moo-moo")
}

func Test_CacheGCsTheOldestItems(t *testing.T) {
	cache := New(Configure[int]().MaxSize(100).PercentToPrune(10))
	defer cache.Stop()

	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("9"), nil)
	assert.Equal(t, cache.Get("10").Value(), 10)
	assert.Equal(t, cache.ItemCount(), 90)
}

func Test_CachePromotedItemsDontGetPruned(t *testing.T) {
	cache := New(Configure[int]().MaxSize(100).PercentToPrune(10).GetsPerPromote(1))
	defer cache.Stop()

	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SyncUpdates()
	cache.Get("9")
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("9").Value(), 9)
	assert.Equal(t, cache.Get("10"), nil)
	assert.Equal(t, cache.Get("11").Value(), 11)
}

func Test_GetWithoutPromoteDoesNotPromote(t *testing.T) {
	cache := New(Configure[int]().MaxSize(100).PercentToPrune(10).GetsPerPromote(1))
	defer cache.Stop()

	for i := 0; i < 100; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SyncUpdates()
	cache.GetWithoutPromote("9")
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("9"), nil)
	assert.Equal(t, cache.Get("10").Value(), 10)
	assert.Equal(t, cache.Get("11").Value(), 11)
}

func Test_CacheTrackerDoesNotCleanupHeldInstance(t *testing.T) {
	cache := New(Configure[int]().MaxSize(10).PercentToPrune(10).Track())
	defer cache.Stop()

	item0 := cache.TrackingSet("0", 0, time.Minute)

	cache.Set("1", 1, time.Minute)
	item1 := cache.TrackingGet("1")

	for i := 2; i < 11; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}

	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("0").Value(), 0)
	assert.Equal(t, cache.Get("1").Value(), 1)
	item0.Release()
	item1.Release()

	for i := 1; i < 5; i++ {
		cache.Set(strconv.Itoa(i+20), i, time.Minute)
	}
	cache.GC()
	assert.Equal(t, cache.Get("0"), nil)
	assert.Equal(t, cache.Get("1"), nil)
}

func Test_CacheRemovesOldestItemWhenFull(t *testing.T) {
	onDeleteFnCalled := false
	onDeleteFn := func(item *Item[int]) {
		if item.key == "0" {
			onDeleteFnCalled = true
		}
	}

	cache := New(Configure[int]().MaxSize(5).PercentToPrune(1).OnDelete(onDeleteFn))
	defer cache.Stop()

	for i := 0; i < 7; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SyncUpdates()
	assert.Equal(t, cache.Get("0"), nil)
	assert.Equal(t, cache.Get("1"), nil)
	assert.Equal(t, cache.Get("2").Value(), 2)
	assert.Equal(t, onDeleteFnCalled, true)
	assert.Equal(t, cache.ItemCount(), 5)
}

func Test_CacheRemovesOldestItemWhenFullBySizer(t *testing.T) {
	cache := New(Configure[*SizedItem]().MaxSize(50).PercentToPrune(15))
	defer cache.Stop()

	for i := 0; i < 25; i++ {
		cache.Set(strconv.Itoa(i), &SizedItem{i, 2}, time.Minute)
	}
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("0"), nil)
	assert.Equal(t, cache.Get("1"), nil)
	assert.Equal(t, cache.Get("2"), nil)
	assert.Equal(t, cache.Get("3"), nil)
	assert.Equal(t, cache.Get("4").Value().id, 4)
	assert.Equal(t, cache.GetDropped(), 4)
	assert.Equal(t, cache.GetDropped(), 0)
}

func Test_CacheSetUpdatesSizeOnDelta(t *testing.T) {
	cache := New(Configure[*SizedItem]())
	defer cache.Stop()

	cache.Set("a", &SizedItem{0, 2}, time.Minute)
	cache.Set("b", &SizedItem{0, 3}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
	cache.Set("b", &SizedItem{0, 3}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
	cache.Set("b", &SizedItem{0, 4}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 6)
	cache.Set("b", &SizedItem{0, 2}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 4)
	cache.Delete("b")
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 2)
}

func Test_CacheReplaceDoesNotchangeSizeIfNotSet(t *testing.T) {
	cache := New(Configure[*SizedItem]())
	defer cache.Stop()

	cache.Set("1", &SizedItem{1, 2}, time.Minute)
	cache.Set("2", &SizedItem{1, 2}, time.Minute)
	cache.Set("3", &SizedItem{1, 2}, time.Minute)
	cache.Replace("4", &SizedItem{1, 2})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 6)
}

func Test_CacheReplaceChangesSize(t *testing.T) {
	cache := New(Configure[*SizedItem]())
	defer cache.Stop()

	cache.Set("1", &SizedItem{1, 2}, time.Minute)
	cache.Set("2", &SizedItem{1, 2}, time.Minute)

	cache.Replace("2", &SizedItem{1, 2})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 4)

	cache.Replace("2", &SizedItem{1, 1})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 3)

	cache.Replace("2", &SizedItem{1, 3})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
}

func Test_CacheResizeOnTheFly(t *testing.T) {
	cache := New(Configure[int]().MaxSize(50).PercentToPrune(10))
	defer cache.Stop()

	for i := 0; i < 50; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SetMaxSize(3)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 47)
	assert.Equal(t, cache.Get("46"), nil)
	assert.Equal(t, cache.Get("47").Value(), 47)
	assert.Equal(t, cache.Get("48").Value(), 48)
	assert.Equal(t, cache.Get("49").Value(), 49)

	cache.Set("50", 50, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 1)
	assert.Equal(t, cache.Get("47"), nil)
	assert.Equal(t, cache.Get("48").Value(), 48)
	assert.Equal(t, cache.Get("49").Value(), 49)
	assert.Equal(t, cache.Get("50").Value(), 50)

	cache.SetMaxSize(10)
	cache.Set("51", 51, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 0)
	assert.Equal(t, cache.Get("48").Value(), 48)
	assert.Equal(t, cache.Get("49").Value(), 49)
	assert.Equal(t, cache.Get("50").Value(), 50)
	assert.Equal(t, cache.Get("51").Value(), 51)
}

func Test_CacheForEachFunc(t *testing.T) {
	cache := New(Configure[int]().MaxSize(3).PercentToPrune(1))
	defer cache.Stop()

	assert.List(t, forEachKeys[int](cache), []string{})

	cache.Set("1", 1, time.Minute)
	assert.List(t, forEachKeys(cache), []string{"1"})

	cache.Set("2", 2, time.Minute)
	cache.SyncUpdates()
	assert.List(t, forEachKeys(cache), []string{"1", "2"})

	cache.Set("3", 3, time.Minute)
	cache.SyncUpdates()
	assert.List(t, forEachKeys(cache), []string{"1", "2", "3"})

	cache.Set("4", 4, time.Minute)
	cache.SyncUpdates()
	assert.List(t, forEachKeys(cache), []string{"2", "3", "4"})

	cache.Set("stop", 5, time.Minute)
	cache.SyncUpdates()
	assert.DoesNotContain(t, forEachKeys(cache), "stop")

	cache.Set("6", 6, time.Minute)
	cache.SyncUpdates()
	assert.DoesNotContain(t, forEachKeys(cache), "stop")
}

func Test_CachePrune(t *testing.T) {
	maxSize := int64(500)
	cache := New(Configure[string]().MaxSize(maxSize).PercentToPrune(50))
	defer cache.Stop()

	epoch := 0
	for i := 0; i < 10000; i++ {
		epoch += 1
		expired := make([]string, 0)
		for i := 0; i < 50; i += 1 {
			key := strconv.FormatInt(rand.Int63n(maxSize*20), 10)
			item := cache.Get(key)
			if item == nil || item.TTL() > 1*time.Minute {
				expired = append(expired, key)
			}
		}
		for _, key := range expired {
			cache.Set(key, key, 5*time.Minute)
		}
		if epoch%500 == 0 {
			assert.True(t, cache.GetSize() <= 500)
		}
	}
}

func Test_ConcurrentStop(t *testing.T) {
	for i := 0; i < 100; i++ {
		cache := New(Configure[string]())
		r := func() {
			for {
				key := strconv.Itoa(int(rand.Int31n(100)))
				switch rand.Int31n(3) {
				case 0:
					cache.Get(key)
				case 1:
					cache.Set(key, key, time.Minute)
				case 2:
					cache.Delete(key)
				}
			}
		}
		go r()
		go r()
		go r()
		time.Sleep(time.Millisecond * 10)
		cache.Stop()
	}
}

func Test_ConcurrentClearAndSet(t *testing.T) {
	for i := 0; i < 1000000; i++ {
		var stop atomic.Bool
		var wg sync.WaitGroup

		cache := New(Configure[string]())
		r := func() {
			for !stop.Load() {
				cache.Set("a", "a", time.Minute)
			}
			wg.Done()
		}
		go r()
		wg.Add(1)
		cache.Clear()
		stop.Store(true)
		wg.Wait()
		cache.SyncUpdates()

		// The point of this test is to make sure that the cache's lookup and its
		// recency list are in sync. But the two aren't written to atomically:
		// the lookup is written to directly from the call to Set, whereas the
		// list is maintained by the background worker. This can create a period
		// where the two are out of sync. Even SyncUpdate is helpless here, since
		// it can only sync what's been written to the buffers.
		for i := 0; i < 10; i++ {
			expectedCount := 0
			if cache.list.Head != nil {
				expectedCount = 1
			}
			actualCount := cache.ItemCount()
			if expectedCount == actualCount {
				return
			}
			time.Sleep(time.Millisecond)
		}
		t.Errorf("cache list and lookup are not consistent")
		t.FailNow()
		cache.Stop()
	}
}

func BenchmarkFrequentSets(b *testing.B) {
	cache := New(Configure[int]())
	defer cache.Stop()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := strconv.Itoa(n)
		cache.Set(key, n, time.Minute)
	}
}

func BenchmarkFrequentGets(b *testing.B) {
	cache := New(Configure[int]())
	defer cache.Stop()
	numKeys := 500
	for i := 0; i < numKeys; i++ {
		key := strconv.Itoa(i)
		cache.Set(key, i, time.Minute)
	}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := strconv.FormatInt(rand.Int63n(int64(numKeys)), 10)
		cache.Get(key)
	}
}

func BenchmarkGetWithPromoteSmall(b *testing.B) {
	getsPerPromotes := 5
	cache := New(Configure[int]().GetsPerPromote(int32(getsPerPromotes)))
	defer cache.Stop()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := strconv.Itoa(n)
		cache.Set(key, n, time.Minute)
		for i := 0; i < getsPerPromotes; i++ {
			cache.Get(key)
		}
	}
}

func BenchmarkGetWithPromoteLarge(b *testing.B) {
	getsPerPromotes := 100
	cache := New(Configure[int]().GetsPerPromote(int32(getsPerPromotes)))
	defer cache.Stop()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		key := strconv.Itoa(n)
		cache.Set(key, n, time.Minute)
		for i := 0; i < getsPerPromotes; i++ {
			cache.Get(key)
		}
	}
}

type SizedItem struct {
	id int
	s  int64
}

func (s *SizedItem) Size() int64 {
	return s.s
}

func forEachKeys[T any](cache *Cache[T]) []string {
	keys := make([]string, 0, 10)
	cache.ForEachFunc(func(key string, i *Item[T]) bool {
		if key == "stop" {
			return false
		}
		keys = append(keys, key)
		return true
	})
	sort.Strings(keys)
	return keys
}

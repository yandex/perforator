package ccache

import (
	"math/rand"
	"sort"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/karlseguin/ccache/v3/assert"
)

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
	cache.Set("spice", "flow", time.Minute)
	cache.Set("worm", "sand", time.Minute)

	cache.SyncUpdates() // wait for worker to pick up preceding updates

	cache.Delete("spice")
	cache.SyncUpdates()

	assert.Equal(t, cache.Get("spice"), nil)
	assert.Equal(t, cache.Get("worm").Value(), "sand")
	assert.Equal(t, atomic.LoadInt32(&onDeleteFnCalled), 1)
}

func Test_CacheFetchesExpiredItems(t *testing.T) {
	cache := New(Configure[string]())
	fn := func() (string, error) { return "moo-moo", nil }

	cache.Set("beef", "moo", time.Second*-1)
	assert.Equal(t, cache.Get("beef").Value(), "moo")

	out, _ := cache.Fetch("beef", time.Second, fn)
	assert.Equal(t, out.Value(), "moo-moo")
}

func Test_CacheGCsTheOldestItems(t *testing.T) {
	cache := New(Configure[int]().ItemsToPrune(10))
	for i := 0; i < 500; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("9"), nil)
	assert.Equal(t, cache.Get("10").Value(), 10)
	assert.Equal(t, cache.ItemCount(), 490)
}

func Test_CachePromotedItemsDontGetPruned(t *testing.T) {
	cache := New(Configure[int]().ItemsToPrune(10).GetsPerPromote(1))
	for i := 0; i < 500; i++ {
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
	cache := New(Configure[int]().ItemsToPrune(10).GetsPerPromote(1))
	for i := 0; i < 500; i++ {
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
	cache := New(Configure[int]().ItemsToPrune(11).Track())
	item0 := cache.TrackingSet("0", 0, time.Minute)
	for i := 1; i < 11; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	item1 := cache.TrackingGet("1")
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("0").Value(), 0)
	assert.Equal(t, cache.Get("1").Value(), 1)
	item0.Release()
	item1.Release()
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

	cache := New(Configure[int]().MaxSize(5).ItemsToPrune(1).OnDelete(onDeleteFn))
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
	cache := New(Configure[*SizedItem]().MaxSize(9).ItemsToPrune(2))
	for i := 0; i < 7; i++ {
		cache.Set(strconv.Itoa(i), &SizedItem{i, 2}, time.Minute)
	}
	cache.SyncUpdates()
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
	cache.Set("1", &SizedItem{1, 2}, time.Minute)
	cache.Set("2", &SizedItem{1, 2}, time.Minute)
	cache.Set("3", &SizedItem{1, 2}, time.Minute)
	cache.Replace("4", &SizedItem{1, 2})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 6)
}

func Test_CacheReplaceChangesSize(t *testing.T) {
	cache := New(Configure[*SizedItem]())
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
	cache := New(Configure[int]().MaxSize(9).ItemsToPrune(1))
	for i := 0; i < 5; i++ {
		cache.Set(strconv.Itoa(i), i, time.Minute)
	}
	cache.SetMaxSize(3)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 2)
	assert.Equal(t, cache.Get("0"), nil)
	assert.Equal(t, cache.Get("1"), nil)
	assert.Equal(t, cache.Get("2").Value(), 2)
	assert.Equal(t, cache.Get("3").Value(), 3)
	assert.Equal(t, cache.Get("4").Value(), 4)

	cache.Set("5", 5, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 1)
	assert.Equal(t, cache.Get("2"), nil)
	assert.Equal(t, cache.Get("3").Value(), 3)
	assert.Equal(t, cache.Get("4").Value(), 4)
	assert.Equal(t, cache.Get("5").Value(), 5)

	cache.SetMaxSize(10)
	cache.Set("6", 6, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 0)
	assert.Equal(t, cache.Get("3").Value(), 3)
	assert.Equal(t, cache.Get("4").Value(), 4)
	assert.Equal(t, cache.Get("5").Value(), 5)
	assert.Equal(t, cache.Get("6").Value(), 6)
}

func Test_CacheForEachFunc(t *testing.T) {
	cache := New(Configure[int]().MaxSize(3).ItemsToPrune(1))
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
	cache := New(Configure[string]().MaxSize(maxSize).ItemsToPrune(50))
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

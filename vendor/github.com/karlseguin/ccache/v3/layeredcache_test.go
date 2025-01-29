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

func Test_LayedCache_GetsANonExistantValue(t *testing.T) {
	cache := newLayered[string]()
	assert.Equal(t, cache.Get("spice", "flow"), nil)
	assert.Equal(t, cache.ItemCount(), 0)
}

func Test_LayedCache_SetANewValue(t *testing.T) {
	cache := newLayered[string]()
	cache.Set("spice", "flow", "a value", time.Minute)
	assert.Equal(t, cache.Get("spice", "flow").Value(), "a value")
	assert.Equal(t, cache.Get("spice", "stop"), nil)
	assert.Equal(t, cache.ItemCount(), 1)
}

func Test_LayedCache_SetsMultipleValueWithinTheSameLayer(t *testing.T) {
	cache := newLayered[string]()
	cache.Set("spice", "flow", "value-a", time.Minute)
	cache.Set("spice", "must", "value-b", time.Minute)
	cache.Set("leto", "sister", "ghanima", time.Minute)
	assert.Equal(t, cache.Get("spice", "flow").Value(), "value-a")
	assert.Equal(t, cache.Get("spice", "must").Value(), "value-b")
	assert.Equal(t, cache.Get("spice", "worm"), nil)

	assert.Equal(t, cache.Get("leto", "sister").Value(), "ghanima")
	assert.Equal(t, cache.Get("leto", "brother"), nil)
	assert.Equal(t, cache.Get("baron", "friend"), nil)
	assert.Equal(t, cache.ItemCount(), 3)
}

func Test_LayedCache_ReplaceDoesNothingIfKeyDoesNotExist(t *testing.T) {
	cache := newLayered[string]()
	assert.Equal(t, cache.Replace("spice", "flow", "value-a"), false)
	assert.Equal(t, cache.Get("spice", "flow"), nil)
}

func Test_LayedCache_ReplaceUpdatesTheValue(t *testing.T) {
	cache := newLayered[string]()
	cache.Set("spice", "flow", "value-a", time.Minute)
	assert.Equal(t, cache.Replace("spice", "flow", "value-b"), true)
	assert.Equal(t, cache.Get("spice", "flow").Value(), "value-b")
	assert.Equal(t, cache.ItemCount(), 1)
	//not sure how to test that the TTL hasn't changed sort of a sleep..
}

func Test_LayedCache_DeletesAValue(t *testing.T) {
	cache := newLayered[string]()
	cache.Set("spice", "flow", "value-a", time.Minute)
	cache.Set("spice", "must", "value-b", time.Minute)
	cache.Set("leto", "sister", "ghanima", time.Minute)
	cache.Delete("spice", "flow")
	assert.Equal(t, cache.Get("spice", "flow"), nil)
	assert.Equal(t, cache.Get("spice", "must").Value(), "value-b")
	assert.Equal(t, cache.Get("spice", "worm"), nil)
	assert.Equal(t, cache.Get("leto", "sister").Value(), "ghanima")
	assert.Equal(t, cache.ItemCount(), 2)
}

func Test_LayedCache_DeletesAPrefix(t *testing.T) {
	cache := newLayered[string]()
	assert.Equal(t, cache.ItemCount(), 0)

	cache.Set("spice", "aaa", "1", time.Minute)
	cache.Set("spice", "aab", "2", time.Minute)
	cache.Set("spice", "aac", "3", time.Minute)
	cache.Set("leto", "aac", "3", time.Minute)
	cache.Set("spice", "ac", "4", time.Minute)
	cache.Set("spice", "z5", "7", time.Minute)
	assert.Equal(t, cache.ItemCount(), 6)

	assert.Equal(t, cache.DeletePrefix("spice", "9a"), 0)
	assert.Equal(t, cache.ItemCount(), 6)

	assert.Equal(t, cache.DeletePrefix("spice", "aa"), 3)
	assert.Equal(t, cache.Get("spice", "aaa"), nil)
	assert.Equal(t, cache.Get("spice", "aab"), nil)
	assert.Equal(t, cache.Get("spice", "aac"), nil)
	assert.Equal(t, cache.Get("spice", "ac").Value(), "4")
	assert.Equal(t, cache.Get("spice", "z5").Value(), "7")
	assert.Equal(t, cache.ItemCount(), 3)
}

func Test_LayedCache_DeletesAFunc(t *testing.T) {
	cache := newLayered[int]()
	assert.Equal(t, cache.ItemCount(), 0)

	cache.Set("spice", "a", 1, time.Minute)
	cache.Set("leto", "b", 2, time.Minute)
	cache.Set("spice", "c", 3, time.Minute)
	cache.Set("spice", "d", 4, time.Minute)
	cache.Set("spice", "e", 5, time.Minute)
	cache.Set("spice", "f", 6, time.Minute)
	assert.Equal(t, cache.ItemCount(), 6)

	assert.Equal(t, cache.DeleteFunc("spice", func(key string, item *Item[int]) bool {
		return false
	}), 0)
	assert.Equal(t, cache.ItemCount(), 6)

	assert.Equal(t, cache.DeleteFunc("spice", func(key string, item *Item[int]) bool {
		return item.Value() < 4
	}), 2)
	assert.Equal(t, cache.ItemCount(), 4)

	assert.Equal(t, cache.DeleteFunc("spice", func(key string, item *Item[int]) bool {
		return key == "d"
	}), 1)
	assert.Equal(t, cache.ItemCount(), 3)

}

func Test_LayedCache_OnDeleteCallbackCalled(t *testing.T) {
	onDeleteFnCalled := int32(0)
	onDeleteFn := func(item *Item[string]) {
		if item.group == "spice" && item.key == "flow" {
			atomic.AddInt32(&onDeleteFnCalled, 1)
		}
	}

	cache := Layered[string](Configure[string]().OnDelete(onDeleteFn))
	cache.Set("spice", "flow", "value-a", time.Minute)
	cache.Set("spice", "must", "value-b", time.Minute)
	cache.Set("leto", "sister", "ghanima", time.Minute)

	cache.SyncUpdates()
	cache.Delete("spice", "flow")
	cache.SyncUpdates()

	assert.Equal(t, cache.Get("spice", "flow"), nil)
	assert.Equal(t, cache.Get("spice", "must").Value(), "value-b")
	assert.Equal(t, cache.Get("spice", "worm"), nil)
	assert.Equal(t, cache.Get("leto", "sister").Value(), "ghanima")

	assert.Equal(t, atomic.LoadInt32(&onDeleteFnCalled), 1)
}

func Test_LayedCache_DeletesALayer(t *testing.T) {
	cache := newLayered[string]()
	cache.Set("spice", "flow", "value-a", time.Minute)
	cache.Set("spice", "must", "value-b", time.Minute)
	cache.Set("leto", "sister", "ghanima", time.Minute)
	cache.DeleteAll("spice")
	assert.Equal(t, cache.Get("spice", "flow"), nil)
	assert.Equal(t, cache.Get("spice", "must"), nil)
	assert.Equal(t, cache.Get("spice", "worm"), nil)
	assert.Equal(t, cache.Get("leto", "sister").Value(), "ghanima")
}

func Test_LayeredCache_GCsTheOldestItems(t *testing.T) {
	cache := Layered(Configure[int]().ItemsToPrune(10))
	cache.Set("xx", "a", 23, time.Minute)
	for i := 0; i < 500; i++ {
		cache.Set(strconv.Itoa(i), "a", i, time.Minute)
	}
	cache.Set("xx", "b", 9001, time.Minute)
	//let the items get promoted (and added to our list)
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("xx", "a"), nil)
	assert.Equal(t, cache.Get("xx", "b").Value(), 9001)
	assert.Equal(t, cache.Get("8", "a"), nil)
	assert.Equal(t, cache.Get("9", "a").Value(), 9)
	assert.Equal(t, cache.Get("10", "a").Value(), 10)
}

func Test_LayeredCache_PromotedItemsDontGetPruned(t *testing.T) {
	cache := Layered(Configure[int]().ItemsToPrune(10).GetsPerPromote(1))
	for i := 0; i < 500; i++ {
		cache.Set(strconv.Itoa(i), "a", i, time.Minute)
	}
	cache.SyncUpdates()
	cache.Get("9", "a")
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("9", "a").Value(), 9)
	assert.Equal(t, cache.Get("10", "a"), nil)
	assert.Equal(t, cache.Get("11", "a").Value(), 11)
}

func Test_LayeredCache_GetWithoutPromoteDoesNotPromote(t *testing.T) {
	cache := Layered(Configure[int]().ItemsToPrune(10).GetsPerPromote(1))
	for i := 0; i < 500; i++ {
		cache.Set(strconv.Itoa(i), "a", i, time.Minute)
	}
	cache.SyncUpdates()
	cache.GetWithoutPromote("9", "a")
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("9", "a"), nil)
	assert.Equal(t, cache.Get("10", "a").Value(), 10)
	assert.Equal(t, cache.Get("11", "a").Value(), 11)
}

func Test_LayeredCache_TrackerDoesNotCleanupHeldInstance(t *testing.T) {
	cache := Layered(Configure[int]().ItemsToPrune(10).Track())
	item0 := cache.TrackingSet("0", "a", 0, time.Minute)
	for i := 1; i < 11; i++ {
		cache.Set(strconv.Itoa(i), "a", i, time.Minute)
	}
	item1 := cache.TrackingGet("1", "a")
	cache.SyncUpdates()
	cache.GC()
	assert.Equal(t, cache.Get("0", "a").Value(), 0)
	assert.Equal(t, cache.Get("1", "a").Value(), 1)
	item0.Release()
	item1.Release()
	cache.GC()
	assert.Equal(t, cache.Get("0", "a"), nil)
	assert.Equal(t, cache.Get("1", "a"), nil)
}

func Test_LayeredCache_RemovesOldestItemWhenFull(t *testing.T) {
	onDeleteFnCalled := false
	onDeleteFn := func(item *Item[int]) {
		if item.key == "a" {
			onDeleteFnCalled = true
		}
	}
	cache := Layered(Configure[int]().MaxSize(5).ItemsToPrune(1).OnDelete(onDeleteFn))

	cache.Set("xx", "a", 23, time.Minute)
	for i := 0; i < 7; i++ {
		cache.Set(strconv.Itoa(i), "a", i, time.Minute)
	}
	cache.Set("xx", "b", 9001, time.Minute)
	cache.SyncUpdates()

	assert.Equal(t, cache.Get("xx", "a"), nil)
	assert.Equal(t, cache.Get("0", "a"), nil)
	assert.Equal(t, cache.Get("1", "a"), nil)
	assert.Equal(t, cache.Get("2", "a"), nil)
	assert.Equal(t, cache.Get("3", "a").Value(), 3)
	assert.Equal(t, cache.Get("xx", "b").Value(), 9001)
	assert.Equal(t, cache.GetDropped(), 4)
	assert.Equal(t, cache.GetDropped(), 0)
	assert.Equal(t, onDeleteFnCalled, true)
}

func Test_LayeredCache_ResizeOnTheFly(t *testing.T) {
	cache := Layered(Configure[int]().MaxSize(9).ItemsToPrune(1))
	for i := 0; i < 5; i++ {
		cache.Set(strconv.Itoa(i), "a", i, time.Minute)
	}
	cache.SyncUpdates()

	cache.SetMaxSize(3)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 2)
	assert.Equal(t, cache.Get("0", "a"), nil)
	assert.Equal(t, cache.Get("1", "a"), nil)
	assert.Equal(t, cache.Get("2", "a").Value(), 2)
	assert.Equal(t, cache.Get("3", "a").Value(), 3)
	assert.Equal(t, cache.Get("4", "a").Value(), 4)

	cache.Set("5", "a", 5, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 1)
	assert.Equal(t, cache.Get("2", "a"), nil)
	assert.Equal(t, cache.Get("3", "a").Value(), 3)
	assert.Equal(t, cache.Get("4", "a").Value(), 4)
	assert.Equal(t, cache.Get("5", "a").Value(), 5)

	cache.SetMaxSize(10)
	cache.Set("6", "a", 6, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetDropped(), 0)
	assert.Equal(t, cache.Get("3", "a").Value(), 3)
	assert.Equal(t, cache.Get("4", "a").Value(), 4)
	assert.Equal(t, cache.Get("5", "a").Value(), 5)
	assert.Equal(t, cache.Get("6", "a").Value(), 6)
}

func Test_LayeredCache_RemovesOldestItemWhenFullBySizer(t *testing.T) {
	cache := Layered(Configure[*SizedItem]().MaxSize(9).ItemsToPrune(2))
	for i := 0; i < 7; i++ {
		cache.Set("pri", strconv.Itoa(i), &SizedItem{i, 2}, time.Minute)
	}
	cache.SyncUpdates()
	assert.Equal(t, cache.Get("pri", "0"), nil)
	assert.Equal(t, cache.Get("pri", "1"), nil)
	assert.Equal(t, cache.Get("pri", "2"), nil)
	assert.Equal(t, cache.Get("pri", "3"), nil)
	assert.Equal(t, cache.Get("pri", "4").Value().id, 4)
}

func Test_LayeredCache_SetUpdatesSizeOnDelta(t *testing.T) {
	cache := Layered(Configure[*SizedItem]())
	cache.Set("pri", "a", &SizedItem{0, 2}, time.Minute)
	cache.Set("pri", "b", &SizedItem{0, 3}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
	cache.Set("pri", "b", &SizedItem{0, 3}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
	cache.Set("pri", "b", &SizedItem{0, 4}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 6)
	cache.Set("pri", "b", &SizedItem{0, 2}, time.Minute)
	cache.Set("sec", "b", &SizedItem{0, 3}, time.Minute)
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 7)
	cache.Delete("pri", "b")
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
}

func Test_LayeredCache_ReplaceDoesNotchangeSizeIfNotSet(t *testing.T) {
	cache := Layered(Configure[*SizedItem]())
	cache.Set("pri", "1", &SizedItem{1, 2}, time.Minute)
	cache.Set("pri", "2", &SizedItem{1, 2}, time.Minute)
	cache.Set("pri", "3", &SizedItem{1, 2}, time.Minute)
	cache.Replace("sec", "3", &SizedItem{1, 2})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 6)
}

func Test_LayeredCache_ReplaceChangesSize(t *testing.T) {
	cache := Layered(Configure[*SizedItem]())
	cache.Set("pri", "1", &SizedItem{1, 2}, time.Minute)
	cache.Set("pri", "2", &SizedItem{1, 2}, time.Minute)

	cache.Replace("pri", "2", &SizedItem{1, 2})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 4)

	cache.Replace("pri", "2", &SizedItem{1, 1})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 3)

	cache.Replace("pri", "2", &SizedItem{1, 3})
	cache.SyncUpdates()
	assert.Equal(t, cache.GetSize(), 5)
}

func Test_LayeredCache_EachFunc(t *testing.T) {
	cache := Layered(Configure[int]().MaxSize(3).ItemsToPrune(1))
	assert.List(t, forEachKeysLayered[int](cache, "1"), []string{})

	cache.Set("1", "a", 1, time.Minute)
	assert.List(t, forEachKeysLayered[int](cache, "1"), []string{"a"})

	cache.Set("1", "b", 2, time.Minute)
	cache.SyncUpdates()
	assert.List(t, forEachKeysLayered[int](cache, "1"), []string{"a", "b"})

	cache.Set("1", "c", 3, time.Minute)
	cache.SyncUpdates()
	assert.List(t, forEachKeysLayered[int](cache, "1"), []string{"a", "b", "c"})

	cache.Set("1", "d", 4, time.Minute)
	cache.SyncUpdates()
	assert.List(t, forEachKeysLayered[int](cache, "1"), []string{"b", "c", "d"})

	// iteration is non-deterministic, all we know for sure is "stop" should not be in there
	cache.Set("1", "stop", 5, time.Minute)
	cache.SyncUpdates()
	assert.DoesNotContain(t, forEachKeysLayered[int](cache, "1"), "stop")

	cache.Set("1", "e", 6, time.Minute)
	cache.SyncUpdates()
	assert.DoesNotContain(t, forEachKeysLayered[int](cache, "1"), "stop")
}

func Test_LayeredCachePrune(t *testing.T) {
	maxSize := int64(500)
	cache := Layered(Configure[string]().MaxSize(maxSize).ItemsToPrune(50))
	epoch := 0
	for i := 0; i < 10000; i++ {
		epoch += 1
		expired := make([]string, 0)
		for i := 0; i < 50; i += 1 {
			key := strconv.FormatInt(rand.Int63n(maxSize*20), 10)
			item := cache.Get(key, key)
			if item == nil || item.TTL() > 1*time.Minute {
				expired = append(expired, key)
			}
		}
		for _, key := range expired {
			cache.Set(key, key, key, 5*time.Minute)
		}
		if epoch%500 == 0 {
			assert.True(t, cache.GetSize() <= 500)
		}
	}
}

func Test_LayeredConcurrentStop(t *testing.T) {
	for i := 0; i < 100; i++ {
		cache := Layered(Configure[string]())
		r := func() {
			for {
				key := strconv.Itoa(int(rand.Int31n(100)))
				switch rand.Int31n(3) {
				case 0:
					cache.Get(key, key)
				case 1:
					cache.Set(key, key, key, time.Minute)
				case 2:
					cache.Delete(key, key)
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
func newLayered[T any]() *LayeredCache[T] {
	c := Layered[T](Configure[T]())
	c.Clear()
	return c
}

func forEachKeysLayered[T any](cache *LayeredCache[T], primary string) []string {
	keys := make([]string, 0, 10)
	cache.ForEachFunc(primary, func(key string, i *Item[T]) bool {
		if key == "stop" {
			return false
		}
		keys = append(keys, key)
		return true
	})
	sort.Strings(keys)
	return keys
}

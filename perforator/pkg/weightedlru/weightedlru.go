package weightedlru

import (
	"errors"
	"fmt"
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

type CacheItem struct {
	weight   uint64
	refCount uint64

	Value any
}

type WeightedLRUCache struct {
	sumWeights uint64
	maxSize    uint64

	mutex              sync.RWMutex
	acquired           map[string]*CacheItem
	sumAcquiredWeights uint64
	released           *lru.Cache

	onEvict func(key, value any)
}

func NewWeightedLRUCache(
	maxSize uint64,
	maxElements int,
	evictionCallback func(key, value any),
) (*WeightedLRUCache, error) {
	result := &WeightedLRUCache{
		acquired: make(map[string]*CacheItem),
		maxSize:  maxSize,
		onEvict:  evictionCallback,
	}

	var err error
	result.released, err = lru.NewWithEvict(maxElements, func(key, val any) {
		// must be called under WeightedLRUCache.mutex.Lock()
		if val.(*CacheItem).refCount > 0 {
			return
		}
		result.sumWeights -= val.(*CacheItem).weight
		result.onEvict(key, val)
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// must be called under WeightedLRUCache.mutex.Lock()
func (c *WeightedLRUCache) get(key string, acquire bool) (item *CacheItem, acquiredBefore bool) {
	value, ok := c.acquired[key]
	if ok {
		if acquire {
			value.refCount++
		}
		return value, true
	}

	cacheItem, ok := c.released.Get(key)
	if ok {
		if acquire {
			cacheItem.(*CacheItem).refCount++
			c.acquired[key] = cacheItem.(*CacheItem)
			c.released.Remove(key)
			c.sumAcquiredWeights += cacheItem.(*CacheItem).weight
		}
		return cacheItem.(*CacheItem), false
	}

	return nil, false
}

// must be called under WeightedLRUCache.Lock()
func (c *WeightedLRUCache) freeCapacity(weight uint64) error {
	if c.maxSize-c.sumAcquiredWeights < weight {
		return fmt.Errorf(
			"cannot free enough capacity for weight %d, sumWeights: %d, sumAcquiredWeights %d, max size: %d",
			weight,
			c.sumWeights,
			c.sumAcquiredWeights,
			c.maxSize,
		)
	}

	for c.sumWeights+weight > c.maxSize {
		_, _, ok := c.released.RemoveOldest()
		if !ok {
			panic("cannot free capacity because released cache is empty")
		}
	}

	return nil
}

type LockedItem struct {
	Inserted       bool
	AcquiredBefore bool
	Value          any
}

// Add item to cache and acquire its ownership.
// If item is acquired than it cannot be dropped from cache,
//
//	if cache capacity is not enough other calls to Acquire will fail
//
// Returns present or fetched value and releaser object.
// Release() must be called, otherwise the item will never be removed from cache
func (c *WeightedLRUCache) Acquire(
	key string,
	weight uint64,
	fetcher func() any,
) (*LockedItem, error) {
	if weight > c.maxSize {
		return nil, errors.New("item weight is more than cache capacity")
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, acquiredBefore := c.get(key /*acquire=*/, true)
	if item != nil {
		return &LockedItem{AcquiredBefore: acquiredBefore, Value: item.Value}, nil
	}

	err := c.freeCapacity(weight)
	if err != nil {
		return nil, err
	}

	val := fetcher()
	c.acquired[key] = &CacheItem{weight: weight, refCount: 1, Value: val}
	c.sumWeights += weight
	c.sumAcquiredWeights += weight

	return &LockedItem{
		Inserted: true,
		Value:    val,
	}, nil
}

// must be called under c.mutex
func (c *WeightedLRUCache) releaseAcquiredIfPossible(key string) *CacheItem {
	item, ok := c.acquired[key]
	if !ok {
		panic(fmt.Sprintf("trying to release item which is not acquired, key: %s", key))
	}

	item.refCount--
	if item.refCount > 0 {
		return nil
	}

	delete(c.acquired, key)
	c.sumAcquiredWeights -= item.weight
	return item
}

func (c *WeightedLRUCache) Release(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item := c.releaseAcquiredIfPossible(key)
	if item != nil {
		c.released.Add(key, item)
	}
}

// purges item if it is not acquired by somebody else
func (c *WeightedLRUCache) ReleaseTryPurge(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item := c.releaseAcquiredIfPossible(key)
	if item == nil {
		return false
	}

	c.sumWeights -= item.weight
	c.onEvict(key, item)
	return true
}

func (c *WeightedLRUCache) PurgeReleased() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.released.Purge()
}

// Add item to cache without acquiring its ownership.
// This means that the item might be dropped from cache
//
//	at any moment.
//
// Returns present value or fetcher value
func (c *WeightedLRUCache) Add(
	key string,
	weight uint64,
	fetcher func() any,
) (any, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	item, _ := c.get(key /*acquire=*/, false)
	if item != nil {
		return item.Value, nil
	}

	err := c.freeCapacity(weight)
	if err != nil {
		return nil, err
	}

	value := fetcher()
	c.sumWeights += weight
	c.released.Add(key, &CacheItem{weight: weight, Value: value})
	return value, nil
}

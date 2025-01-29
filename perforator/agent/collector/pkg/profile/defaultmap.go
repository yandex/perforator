package profile

import (
	"sync"

	"golang.org/x/exp/constraints"
)

type DefaultMap[K constraints.Ordered, V any] struct {
	init func(K) *V
	fini func(K, *V)

	mu sync.RWMutex
	m  map[K]*V
}

func NewDefaultMap[K constraints.Ordered, V any](
	init func(K) *V,
	fini func(K, *V),
) *DefaultMap[K, V] {
	return &DefaultMap[K, V]{
		init: init,
		fini: fini,
		m:    make(map[K]*V),
	}
}

func (m *DefaultMap[K, V]) Get(key K) *V {
	m.mu.RLock()
	value, ok := m.m[key]
	m.mu.RUnlock()

	if ok {
		return value
	}

	newval := m.init(key)

	m.mu.Lock()
	value, ok = m.m[key]
	if ok {
		m.mu.Unlock()
		if m.fini != nil {
			m.fini(key, newval)
		}
		return value
	}
	m.m[key] = newval
	m.mu.Unlock()

	return newval
}

func (m *DefaultMap[K, V]) Erase(key K) {
	var value *V
	m.mu.Lock()
	value, ok := m.m[key]
	if ok {
		delete(m.m, key)
	}
	m.mu.Unlock()

	if m.fini != nil {
		m.fini(key, value)
	}
}

func (m *DefaultMap[K, V]) Clear() {
	var tmp map[K]*V

	m.mu.Lock()
	tmp = m.m
	m.m = make(map[K]*V)
	m.mu.Unlock()

	for k, v := range tmp {
		if m.fini != nil {
			m.fini(k, v)
		}
	}
}

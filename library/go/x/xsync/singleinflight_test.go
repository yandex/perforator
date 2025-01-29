package xsync

import (
	"sync"
	"testing"

	"golang.org/x/sync/singleflight"
)

func BenchmarkDo(b *testing.B) {
	flight := NewSingleInflight()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			flight.Do(func() {})
		}
	})
}

func BenchmarkXSyncSingleFlight(b *testing.B) {
	var flight singleflight.Group
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = flight.Do("key", func() (interface{}, error) {
				return nil, nil
			})
		}
	})
}

func BenchmarkSlowImplDo(b *testing.B) {
	flight := newSlowSingleInflight()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			flight.Do(func() {})
		}
	})
}

type slowerSingleInflight struct {
	onceLock     sync.RWMutex
	updatingOnce *sync.Once
}

func newSlowSingleInflight() slowerSingleInflight {
	return slowerSingleInflight{
		updatingOnce: new(sync.Once),
	}
}

func (i *slowerSingleInflight) Do(f func()) {
	i.getOnce().Do(func() {
		f()
		i.setOnce()
	})
}

func (i *slowerSingleInflight) getOnce() *sync.Once {
	i.onceLock.RLock()
	defer i.onceLock.RUnlock()
	return i.updatingOnce
}

func (i *slowerSingleInflight) setOnce() {
	i.onceLock.Lock()
	defer i.onceLock.Unlock()
	i.updatingOnce = new(sync.Once)
}

package cgroups

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
)

func createLogger() (log.Logger, error) {
	lconf := zap.KVConfig(log.DebugLevel)
	lconf.OutputPaths = []string{"stderr"}
	return zap.New(lconf)
}

type testEvent struct {
	CloseCalls  int
	OpenCalls   int
	Opened      bool
	alwaysStale bool
}

func newTestEvent(alwaysStale bool) *testEvent {
	return &testEvent{
		alwaysStale: alwaysStale,
	}
}

func (e *testEvent) Open(name string, cgroupID uint64) error {
	if e.Opened {
		return errors.New("test event is already opened")
	}
	e.OpenCalls++
	e.Opened = true
	return nil
}

func (e *testEvent) Close() {
	if !e.Opened {
		panic("close called on closed event")
	}

	e.Opened = false
	e.CloseCalls++
}

func (e *testEvent) IsStale() bool {
	return e.alwaysStale
}

type testNameCache struct {
	counter      uint64
	cache        map[string]uint64
	reverseCache map[uint64]string
	mutex        sync.Mutex
}

func newTestNameCache() *testNameCache {
	return &testNameCache{
		counter:      0,
		cache:        make(map[string]uint64),
		reverseCache: make(map[uint64]string),
	}
}

func (n *testNameCache) addCgroup(name string) (uint64, error) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	if id, ok := n.cache[name]; ok {
		return id, nil
	}

	n.cache[name] = n.counter
	n.reverseCache[n.counter] = filepath.Base(name)
	n.counter++
	return n.counter - 1, nil
}

func (n *testNameCache) cgroupFullName(id uint64) string {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	return n.reverseCache[id]
}

func (n *testNameCache) cgroupBaseName(id uint64) string {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	return filepath.Base(n.reverseCache[id])
}

func (n *testNameCache) updateCgroup(name string) uint64 {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	id, ok := n.cache[name]
	if !ok {
		return ^uint64(0)
	}

	delete(n.reverseCache, id)
	n.cache[name] = n.counter
	n.reverseCache[n.counter] = name
	n.counter++
	return n.counter - 1
}

func (n *testNameCache) populate() error {
	return nil
}

func (n *testNameCache) cgroupVersion() CgroupVersion {
	panic("Unsupported")
}

func TestTracker_Simple(t *testing.T) {
	logger, err := createLogger()
	require.NoError(t, err)

	nameCache := newTestNameCache()

	tracker, err := newTrackerImpl(logger, nameCache)
	require.NoError(t, err)

	event1 := newTestEvent(false)
	event2 := newTestEvent(false)
	event3 := newTestEvent(true)
	cgrpName1 := "/sys/fs/cgroup/freezer/porto/tier0-attr-base"
	cgrpName2 := "/sys/fs/cgroup/freezer/porto/yt-arnold"
	cgrpName3 := "/sys/fs/cgroup/freezer/porto/tier0-base"

	err = tracker.AddCgroup(
		&TrackedCgroup{
			Name:  cgrpName1,
			Event: event1,
		},
		true,
	)
	require.NoError(t, err)

	require.Equal(t, int(1), event1.OpenCalls)
	require.Equal(t, int(0), event1.CloseCalls)
	require.True(t, event1.Opened)
	require.Equal(t, int(1), tracker.NumCgroupNames())

	err = tracker.Delete(cgrpName1)
	require.NoError(t, err)

	require.Equal(t, int(1), event1.OpenCalls)
	require.Equal(t, int(1), event1.CloseCalls)
	require.False(t, event1.Opened)
	require.Equal(t, int(0), tracker.NumCgroupNames())
	require.Equal(t, int(0), tracker.NumCgroupIDs())

	err = tracker.AddCgroup(
		&TrackedCgroup{
			Name:  cgrpName2,
			Event: event2,
		},
		true,
	)
	require.NoError(t, err)

	err = tracker.TrackCgroups(
		[]*TrackedCgroup{
			{
				Name:  cgrpName1,
				Event: event1,
			},
			{
				Name:  cgrpName3,
				Event: event3,
			},
		},
	)
	require.NoError(t, err)

	require.Equal(t, int(1), event2.OpenCalls)
	require.Equal(t, int(1), event2.CloseCalls)
	require.False(t, event2.Opened)

	require.True(t, event1.Opened)
	require.True(t, event3.Opened)

	err = tracker.checkUpdatedCgroups()
	require.NoError(t, err)

	require.True(t, event3.Opened)
	require.Equal(t, int(1), event3.CloseCalls)
	require.True(t, event1.Opened)
	require.Equal(t, int(2), event1.OpenCalls)

	nameCache.updateCgroup(cgrpName1)
	err = tracker.checkUpdatedCgroups()
	require.NoError(t, err)

	require.True(t, event3.Opened)
	require.Equal(t, int(2), event3.CloseCalls)
	require.True(t, event1.Opened)
	require.Equal(t, int(3), event1.OpenCalls)

	require.Equal(t, int(2), tracker.NumCgroupIDs())
	require.Equal(t, int(2), tracker.NumCgroupNames())
}

func TestTracker_Concurrent(t *testing.T) {
	logger, err := createLogger()
	require.NoError(t, err)

	nameCache := newTestNameCache()

	tracker, err := newTrackerImpl(logger, nameCache)
	require.NoError(t, err)

	// check `checkUpdatedCgroups`, `AddCgroup`, `Delete`, `TrackCgroups`
	cgroupsCount := 1000

	g, _ := errgroup.WithContext(context.Background())

	events := []CgroupEventListener{}

	for j := 0; j < 2; j++ {
		g.Go(func() error {
			for i := 0; i < 3*cgroupsCount; i++ {
				idx := i % cgroupsCount
				newEvent := newTestEvent(idx%2 == 0)
				events = append(events, newEvent)
				err = tracker.AddCgroup(
					&TrackedCgroup{
						Name:  fmt.Sprintf("%d", idx),
						Event: newEvent,
					},
					false,
				)
				require.NoError(t, err)
			}
			return nil
		})
	}

	for j := 0; j < 2; j++ {
		g.Go(func() error {
			for i := 0; i < 3*cgroupsCount; i++ {
				err = tracker.Delete(fmt.Sprintf("%d", i%cgroupsCount))
				require.NoError(t, err)
			}
			return nil
		})
	}

	g.Go(func() error {
		for j := cgroupsCount - 1; j >= 0; j-- {
			nameCache.updateCgroup(fmt.Sprintf("%d", j))
		}
		return nil
	})

	g.Go(func() error {
		for i := 0; i < 100; i++ {
			err = tracker.checkUpdatedCgroups()
			require.NoError(t, err)
		}
		return nil
	})

	err = g.Wait()
	require.NoError(t, err)
	require.Equal(t, tracker.NumCgroupNames(), tracker.NumCgroupIDs())

	for i := 0; i < cgroupsCount; i++ {
		err = tracker.Delete(fmt.Sprintf("%d", i))
		require.NoError(t, err)
	}

	for _, event := range events {
		testEv := event.(*testEvent)
		require.False(t, testEv.Opened)
		require.Equal(t, testEv.OpenCalls, testEv.CloseCalls)
	}
}

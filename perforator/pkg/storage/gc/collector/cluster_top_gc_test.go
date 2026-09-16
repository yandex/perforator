package collector

import (
	"context"
	"errors"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/mock"
	clustertopgc "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/gc"
	generation "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/generations"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func TestClusterTopSelection(t *testing.T) {
	cutoff := time.Now()
	old := cutoff.Add(-time.Hour)
	rows := []clustertopgc.Generation{
		{ID: 1, To: old, Status: generation.StatusFinished}, // legacy
		{ID: 2, To: old, Status: generation.StatusScheduled, BucketCount: 4},
		{ID: 3, To: old, Status: generation.StatusFinished, BucketCount: 4, HasPendingJobs: true},
		{ID: 4, To: old, Status: generation.StatusFinished, BucketCount: 4},
		{ID: 5, To: cutoff, Status: generation.StatusFinished, BucketCount: 4},                // TTL boundary
		{ID: 6, To: cutoff.Add(time.Hour), Status: generation.StatusDeleting, BucketCount: 4}, // resume despite TTL
		{ID: 7, To: old, Status: generation.StatusFinished, BucketCount: 4},                   // latest finished
		{ID: 8, To: old, Status: generation.StatusScheduled, BucketCount: 4},                  // ID watermark
	}
	candidates := selectClusterTopGenerations(rows, cutoff)
	require.Equal(t, []clustertopgc.Generation{rows[3], rows[5]}, candidates)
	require.Empty(t, selectClusterTopGenerations(rows[6:7], cutoff), "never delete the only generation")
}

type fakeClusterTopGCStorage struct {
	generations      []clustertopgc.Generation
	data             map[uint32]bool
	jobs             int64
	calls            []string
	jobTimes         []time.Time
	dataResponseLost bool
	fail             string
	cancelAt         string
	cancel           context.CancelFunc
	markRejected     bool
}

func (s *fakeClusterTopGCStorage) step(name string) error {
	s.calls = append(s.calls, name)
	if name == s.cancelAt {
		s.cancel()
	}
	if name == s.fail {
		return errors.New("injected failure")
	}
	return nil
}

func (s *fakeClusterTopGCStorage) ListGenerations(context.Context) ([]clustertopgc.Generation, error) {
	return slices.Clone(s.generations), s.step("list")
}

func (s *fakeClusterTopGCStorage) MarkDeleting(_ context.Context, id uint32, _ time.Time) (bool, error) {
	if err := s.step("mark"); err != nil {
		return false, err
	}
	if s.markRejected {
		return false, nil
	}
	for i := range s.generations {
		if s.generations[i].ID == id {
			s.generations[i].Status = generation.StatusDeleting
		}
	}
	return true, nil
}

func (s *fakeClusterTopGCStorage) DeleteData(_ context.Context, id uint32) error {
	err := s.step("data")
	if err == nil || s.dataResponseLost {
		// A mutation may complete before a failed response reaches the GC.
		delete(s.data, id)
	}
	return err
}

func (s *fakeClusterTopGCStorage) DeleteJobs(_ context.Context, _ uint32, size uint32) (int64, error) {
	s.jobTimes = append(s.jobTimes, time.Now())
	if err := s.step("jobs"); err != nil {
		return 0, err
	}
	count := min(s.jobs, int64(size))
	s.jobs -= count
	return count, nil
}

func (s *fakeClusterTopGCStorage) DeleteGeneration(_ context.Context, id uint32) error {
	if err := s.step("generation"); err != nil {
		return err
	}
	s.generations = slices.DeleteFunc(s.generations, func(g clustertopgc.Generation) bool { return g.ID == id })
	return nil
}

func testClusterTopGC() (*clusterTopGC, *fakeClusterTopGCStorage) {
	conf := config.DefaultClusterTopConfig()
	conf.Enabled = true
	s := &fakeClusterTopGCStorage{
		generations: []clustertopgc.Generation{{ID: 1, To: time.Now().Add(-2 * conf.TTL), Status: generation.StatusFinished, BucketCount: 2}, {ID: 2, To: time.Now(), Status: generation.StatusFinished, BucketCount: 2}},
		data:        map[uint32]bool{1: true, 2: true},
		jobs:        5001,
	}
	return newClusterTopGC(xlog.NewNop(), mock.NewRegistry(nil), conf, s), s
}

func TestClusterTopCleanupOrderAndRate(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, s := testClusterTopGC()
		require.NoError(t, c.collect(t.Context()))
		require.Equal(t, []string{"list", "mark", "data", "jobs", "jobs", "jobs", "generation"}, s.calls)
		require.EqualValues(t, 0, s.jobs)
		require.Equal(t, map[uint32]bool{2: true}, s.data)
		require.Len(t, s.generations, 1)
		require.EqualValues(t, 2, s.generations[0].ID)
		for i := 1; i < len(s.jobTimes); i++ {
			require.GreaterOrEqual(t, s.jobTimes[i].Sub(s.jobTimes[i-1]), c.conf.JobsInterval)
		}
	})
}

func TestClusterTopResumeAfterEachFailure(t *testing.T) {
	for _, step := range []string{"list", "mark", "data", "data-committed", "jobs", "generation"} {
		t.Run(step, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c, s := testClusterTopGC()
				s.fail = step
				if step == "data-committed" {
					s.fail = "data"
					s.dataResponseLost = true
				}
				require.Error(t, c.collect(t.Context()))
				require.Equal(t, s.fail, s.calls[len(s.calls)-1], "stop before subsequent stages")
				require.Len(t, s.generations, 2, "retain metadata for retry")
				s.fail = ""
				require.NoError(t, c.collect(t.Context()))
				require.Len(t, s.generations, 1)
				require.Equal(t, map[uint32]bool{2: true}, s.data)
				require.Zero(t, s.jobs)
			})
		})
	}
}

func TestClusterTopRejectedTransition(t *testing.T) {
	c, s := testClusterTopGC()
	s.markRejected = true
	require.NoError(t, c.collect(t.Context()))
	require.Equal(t, []string{"list", "mark"}, s.calls)
	require.Equal(t, generation.StatusFinished, s.generations[0].Status)
	require.Equal(t, map[uint32]bool{1: true, 2: true}, s.data)
	require.EqualValues(t, 5001, s.jobs)
}

func TestClusterTopStopsAfterCanceledOperation(t *testing.T) {
	for _, step := range []string{"list", "mark", "data", "jobs"} {
		t.Run(step, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				c, s := testClusterTopGC()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				s.cancelAt = step
				s.cancel = cancel
				require.ErrorIs(t, c.collect(ctx), context.Canceled)
				require.Equal(t, step, s.calls[len(s.calls)-1])
				require.Len(t, s.generations, 2)
			})
		})
	}
}

func TestClusterTopCancellationDuringWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, s := testClusterTopGC()
		s.generations = nil
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- c.run(ctx) }()
		synctest.Wait()
		require.Equal(t, []string{"list"}, s.calls)
		cancel()
		synctest.Wait()
		require.ErrorIs(t, <-done, context.Canceled)
		require.Equal(t, []string{"list"}, s.calls)
	})
}

type blockingClusterTopGCStorage struct {
	*fakeClusterTopGCStorage
	started, finish chan struct{}
}

func (s *blockingClusterTopGCStorage) DeleteData(ctx context.Context, id uint32) error {
	close(s.started)
	<-s.finish
	return s.fakeClusterTopGCStorage.DeleteData(ctx, id)
}

func TestClusterTopUsesSharedLeaseAndDrains(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		c, s := testClusterTopGC()
		blocked := &blockingClusterTopGCStorage{s, make(chan struct{}), make(chan struct{})}
		c.storage = blocked
		ls := &fakeLeaseStorage{released: make(chan struct{}, 1), waiting: make(chan struct{}, 1)}
		g := &GC{clusterTop: c, l: xlog.NewNop(), leaseStorage: ls, leaseName: "gc", leaseTTL: 30 * time.Second}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() { done <- g.Run(ctx, time.Minute) }()
		<-blocked.started
		ls.mu.Lock()
		require.NotEmpty(t, ls.holder)
		ls.mu.Unlock()
		cancel()
		synctest.Wait()
		require.Empty(t, done, "GC must wait for the active data deletion")
		require.Empty(t, ls.released, "lease must remain held until data deletion returns")
		close(blocked.finish)
		synctest.Wait()
		require.ErrorIs(t, <-done, context.Canceled)
		require.Len(t, s.generations, 2)
		require.EqualValues(t, 5001, s.jobs)
		require.Equal(t, "data", s.calls[len(s.calls)-1])
	})
}

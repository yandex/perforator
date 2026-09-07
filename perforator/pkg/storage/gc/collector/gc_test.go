package collector

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	metricsmock "github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type pageResult struct {
	ids       []string
	readErr   error
	deleteErr error
}

type scriptedStorage struct {
	t          *testing.T
	pages      []pageResult
	next       int
	ttls       []time.Duration
	pagination []util.Pagination
	shards     []storage.ShardParams
	deletes    [][]string
}

func (s *scriptedStorage) CollectExpired(_ context.Context, ttl time.Duration, pagination *util.Pagination, shard *storage.ShardParams) ([]*storage.ObjectMeta, error) {
	s.t.Helper()
	require.Less(s.t, s.next, len(s.pages), "unexpected extra page read")
	page := s.pages[s.next]
	s.next++
	s.ttls = append(s.ttls, ttl)
	s.pagination = append(s.pagination, *pagination)
	s.shards = append(s.shards, *shard)
	metas := make([]*storage.ObjectMeta, 0, len(page.ids))
	for _, id := range page.ids {
		metas = append(metas, &storage.ObjectMeta{ID: id, LastUsedTimestamp: time.Now().Add(-ttl - time.Hour)})
	}
	return metas, page.readErr
}

func (s *scriptedStorage) Delete(_ context.Context, ids []string) error {
	s.t.Helper()
	page := s.pages[s.next-1]
	require.Equal(s.t, page.ids, ids)
	s.deletes = append(s.deletes, append([]string(nil), ids...))
	return page.deleteErr
}

func testShard(t *testing.T, s storage.Storage, kind config.StorageType, pageSize uint32) (*shardCollector, *metricsmock.Registry) {
	t.Helper()
	conf := config.Config{Storages: []config.StorageConfig{{Type: kind, TTL: config.TTLConfig{TTL: 24 * time.Hour}, DeletePageSize: pageSize}}}
	conf.FillDefault()
	registry := metricsmock.NewRegistry(nil)
	c, err := newGC(xlog.ForTest(t), &conf.Storages[0], string(kind), s, registry)
	require.NoError(t, err)
	tagged, ok := registry.GetWithTags(map[string]string{"storage_type": string(kind)})
	require.True(t, ok)
	return c.(*collector).shards[0].(*shardCollector), tagged
}

func testTimer(t *testing.T, r *metricsmock.Registry, name string) *metricsmock.Timer {
	t.Helper()
	timer, ok := r.GetTimer(name)
	require.True(t, ok)
	return timer
}

func TestCollectPages(t *testing.T) {
	readErr := errors.New("read failed")
	deleteErr := errors.New("delete failed")
	lastErr := errors.New("last failure")
	tests := []struct {
		name              string
		pages             []pageResult
		deletes           [][]string
		deleted, failures int64
		wantErr           error
		limit             bool
	}{
		{name: "empty", pages: []pageResult{{}}},
		{name: "multiple pages", pages: []pageResult{{ids: []string{"a", "b"}}, {ids: []string{"c"}}, {}}, deletes: [][]string{{"a", "b"}, {"c"}}, deleted: 3},
		{name: "read failure then empty", pages: []pageResult{{readErr: readErr}, {}}, failures: 1, wantErr: readErr},
		{name: "retry failed delete", pages: []pageResult{{ids: []string{"a"}, deleteErr: deleteErr}, {ids: []string{"a"}}, {}}, deletes: [][]string{{"a"}, {"a"}}, deleted: 1, failures: 1, wantErr: deleteErr},
		{name: "last error survives successful pages", pages: []pageResult{{readErr: readErr}, {ids: []string{"a"}}, {readErr: lastErr}, {}}, deletes: [][]string{{"a"}}, deleted: 1, failures: 2, wantErr: lastErr},
		{name: "six read errors", pages: []pageResult{{readErr: readErr}, {readErr: readErr}, {readErr: readErr}, {readErr: readErr}, {readErr: readErr}, {readErr: readErr}}, failures: 6, limit: true},
		{name: "success does not reset error count", pages: []pageResult{{readErr: readErr}, {ids: []string{"a"}}, {readErr: readErr}, {ids: []string{"b"}}, {readErr: readErr}, {readErr: readErr}, {readErr: readErr}, {readErr: readErr}}, deletes: [][]string{{"a"}, {"b"}}, deleted: 2, failures: 6, limit: true},
		{name: "six delete errors", pages: []pageResult{{ids: []string{"a"}, deleteErr: deleteErr}, {ids: []string{"a"}, deleteErr: deleteErr}, {ids: []string{"a"}, deleteErr: deleteErr}, {ids: []string{"a"}, deleteErr: deleteErr}, {ids: []string{"a"}, deleteErr: deleteErr}, {ids: []string{"a"}, deleteErr: deleteErr}}, deletes: [][]string{{"a"}, {"a"}, {"a"}, {"a"}, {"a"}, {"a"}}, failures: 6, limit: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &scriptedStorage{t: t, pages: tt.pages}
			c, registry := testShard(t, s, config.Binary, 2)
			readTimer := testTimer(t, registry, "collect_expired.timer")
			deleteTimer := testTimer(t, registry, "delete.timer")
			successRegistry, ok := registry.GetWithTags(map[string]string{"status": "success"})
			require.True(t, ok)
			failureRegistry, ok := registry.GetWithTags(map[string]string{"status": "failed"})
			require.True(t, ok)
			successTimer := testTimer(t, successRegistry, "shard_iterations.timer")
			failureTimer := testTimer(t, failureRegistry, "shard_iterations.timer")
			for _, timer := range []*metricsmock.Timer{readTimer, deleteTimer, successTimer, failureTimer} {
				timer.Value.Store(-1)
			}
			err := c.Collect(t.Context())
			switch {
			case tt.limit:
				require.EqualError(t, err, "exceeded page errors iteration limit 5")
			case tt.wantErr != nil:
				require.ErrorIs(t, err, tt.wantErr)
			default:
				require.NoError(t, err)
			}
			require.Equal(t, len(tt.pages), s.next)
			require.Equal(t, tt.deletes, s.deletes)
			for i := range s.pagination {
				require.Equal(t, 24*time.Hour, s.ttls[i])
				require.Equal(t, util.Pagination{Offset: 0, Limit: 2}, s.pagination[i])
				require.Equal(t, storage.ShardParams{NumShards: 1}, s.shards[i])
			}
			deletedRegistry, ok := registry.GetWithTags(map[string]string{"kind": "deleted"})
			require.True(t, ok)
			deleted, ok := deletedRegistry.GetCounter("objects.count")
			require.True(t, ok)
			require.Equal(t, tt.deleted, deleted.Value.Load())
			failures, ok := registry.GetCounter("delete_error.count")
			require.True(t, ok)
			require.Equal(t, tt.failures, failures.Value.Load())
			busy, ok := registry.GetIntGauge("busy_shards.gauge")
			require.True(t, ok)
			require.Zero(t, busy.Value.Load())
			hasReadSuccess, hasDeleteSuccess := false, false
			for _, page := range tt.pages {
				hasReadSuccess = hasReadSuccess || page.readErr == nil
				hasDeleteSuccess = hasDeleteSuccess || (len(page.ids) > 0 && page.readErr == nil && page.deleteErr == nil)
			}
			require.Equal(t, hasReadSuccess, readTimer.Value.Load() >= 0)
			require.Equal(t, hasDeleteSuccess, deleteTimer.Value.Load() >= 0)
			// Preserve the existing metric classification, including failed iterations.
			require.GreaterOrEqual(t, successTimer.Value.Load(), time.Duration(0))
			require.Equal(t, time.Duration(-1), failureTimer.Value.Load())
		})
	}
}

func TestCollectPageSize(t *testing.T) {
	for _, kind := range []config.StorageType{config.Binary, config.GSYM, config.Profile} {
		for _, size := range []uint32{0, 7, 500} {
			t.Run(string(kind)+"/"+time.Duration(size).String(), func(t *testing.T) {
				s := &scriptedStorage{t: t, pages: []pageResult{{}}}
				c, _ := testShard(t, s, kind, size)
				require.NoError(t, c.Collect(t.Context()))
				expected := uint64(size)
				if size == 0 {
					expected = 100
				}
				require.Equal(t, []util.Pagination{{Offset: 0, Limit: expected}}, s.pagination)
			})
		}
	}
}

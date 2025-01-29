package collector

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	PageErrorsIterationLimit = 5
)

type ShardCollector interface {
	Collect(ctx context.Context) error
	LastCollection() time.Time
	InProgress() bool
}

type shardCollector struct {
	l    xlog.Logger
	conf *config.StorageConfig

	shardIndex uint32
	numShards  uint32
	lastGC     time.Time

	storage storage.Storage

	metrics *collectorMetrics

	mutex sync.RWMutex
}

func (c *shardCollector) processPage(
	ctx context.Context,
	pagination *util.Pagination,
) (emptyPage bool, err error) {
	tm := time.Now()

	metas, err := c.storage.CollectExpired(
		ctx,
		c.conf.TTL.TTL,
		pagination,
		&storage.ShardParams{
			ShardIndex: c.shardIndex,
			NumShards:  c.numShards,
		},
	)
	if err != nil {
		return false, err
	}
	c.metrics.collectExpiredTimer.RecordDuration(time.Since(tm))

	if len(metas) == 0 {
		return true, nil
	}

	IDs := make([]string, 0, len(metas))
	for _, meta := range metas {
		IDs = append(IDs, meta.ID)
	}

	tm = time.Now()
	err = c.storage.Delete(ctx, IDs)
	if err != nil {
		return false, err
	}
	c.metrics.deleteTimer.RecordDuration(time.Since(tm))
	c.metrics.deletedObjects.Add(int64(len(metas)))

	for _, meta := range metas {
		if meta.LastUsedTimestamp.Add(c.conf.TTL.TTL).After(time.Now()) {
			c.l.Error(ctx, "Deleted object which is not expired")
		}

		c.l.Debug(
			ctx,
			"Removed object",
			log.String("id", meta.ID),
			log.Time("last_used_timestamp", meta.LastUsedTimestamp),
		)
	}

	return false, nil
}

func (c *shardCollector) Collect(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.metrics.busyShards.Add(1)
	defer c.metrics.busyShards.Add(-1)
	defer func() {
		c.lastGC = time.Now()
		c.l.Info(ctx, "Finished collecting shard")
	}()

	var err error
	tm := time.Now()
	defer func() {
		if err != nil {
			c.metrics.failedShardIterationsTimer.RecordDuration(time.Since(tm))
		} else {
			c.metrics.successShardIterationsTimer.RecordDuration(time.Since(tm))
		}
	}()

	c.l.Info(ctx, "Collecting shard")

	pageSize := uint64(c.conf.DeletePageSize)
	if pageSize == 0 {
		pageSize = 100
	}
	pagination := &util.Pagination{Offset: 0, Limit: pageSize}
	pageErrors := 0

	var deleteError error
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		emptyPage, err := c.processPage(ctx, pagination)
		if err != nil {
			c.l.Error(ctx, "Failed to process page",
				log.UInt64("page_size", pageSize),
				log.Error(err),
			)
			deleteError = err
			c.metrics.deleteErrors.Inc()
			pageErrors++
		}

		if pageErrors > PageErrorsIterationLimit {
			err = fmt.Errorf(
				"exceeded page errors iteration limit %d",
				PageErrorsIterationLimit,
			)
			return err
		}

		if emptyPage {
			break
		}
	}

	return deleteError
}

func (c *shardCollector) LastCollection() time.Time {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.lastGC
}

func (c *shardCollector) InProgress() bool {
	locked := c.mutex.TryLock()
	if locked {
		c.mutex.Unlock()
		return false
	}

	return true
}

func newGCShard(
	l xlog.Logger,
	shardIndex, numShards uint32,
	conf *config.StorageConfig,
	storage storage.Storage,
	metrics *collectorMetrics,
) (*shardCollector, error) {
	return &shardCollector{
		l: l.With(
			log.UInt32("shard_index", shardIndex),
			log.UInt32("num_shards", numShards),
			log.Duration("ttl", conf.TTL.TTL),
		),
		conf:       conf,
		shardIndex: shardIndex,
		numShards:  numShards,
		storage:    storage,
		metrics:    metrics,
	}, nil
}

package collector

import (
	"context"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/binary"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	profilestorage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	MaxShards = 4096
)

type collectorMetrics struct {
	deletedObjects metrics.Counter
	deleteErrors   metrics.Counter

	collectExpiredTimer metrics.Timer
	deleteTimer         metrics.Timer

	successShardIterationsTimer metrics.Timer
	failedShardIterationsTimer  metrics.Timer

	busyShards metrics.IntGauge
}

type Collector interface {
	Type() config.StorageType
	Run(ctx context.Context, interval time.Duration) error
}

type collector struct {
	l xlog.Logger

	storage storage.Storage
	gcConf  *config.StorageConfig

	semaphore semaphore.Weighted
	shards    []ShardCollector

	metrics *collectorMetrics
}

func (c *collector) Type() config.StorageType {
	return c.gcConf.Type
}

func (c *collector) Run(ctx context.Context, interval time.Duration) error {
	numClearedShards := 0

	for i := 0; ; i = (i + 1) % len(c.shards) {
		shardIndex := i
		shard := c.shards[shardIndex]

		if numClearedShards >= len(c.shards) {
			c.l.Info(ctx,
				"Sleep after processing all shards",
				log.Duration("interval", interval),
				log.Int("num_shards", len(c.shards)),
			)
			time.Sleep(interval)
			numClearedShards = 0
		}

		if (!shard.LastCollection().IsZero() && time.Since(shard.LastCollection()) < interval) ||
			shard.InProgress() {
			numClearedShards++
			continue
		}
		numClearedShards = 0

		go func() {
			_ = c.semaphore.Acquire(ctx, 1)
			defer c.semaphore.Release(1)

			err := shard.Collect(ctx)
			if err != nil {
				c.l.Error(ctx,
					"Failed to collect shard",
					log.Int("shard_index", shardIndex),
					log.Int("num_shards", len(c.shards)),
					log.Error(err),
				)
			}
		}()
	}
}

func isPowerOfTwo(num uint32) bool {
	if num == 0 {
		return true
	}

	for num%2 == 0 {
		num /= 2
	}
	return num == 1
}

func (c *collector) initConcurrency() error {
	concurrency := int64(1)
	if c.gcConf.Concurrency != nil && c.gcConf.Concurrency.Concurrency > 1 {
		concurrency = int64(c.gcConf.Concurrency.Concurrency)
	}
	c.semaphore = *semaphore.NewWeighted(concurrency)

	numShards := uint32(1)
	if c.gcConf.Concurrency != nil && c.gcConf.Concurrency.Shards > 1 {
		numShards = c.gcConf.Concurrency.Shards
	}

	if !isPowerOfTwo(numShards) {
		return fmt.Errorf("number of shards %d is not power of 2", numShards)
	}

	if numShards > MaxShards {
		return fmt.Errorf("number of shards %d is more than max shards %d", numShards, MaxShards)
	}

	c.shards = make([]ShardCollector, 0, numShards)
	for i := uint32(0); i < numShards; i++ {
		shard, err := newGCShard(
			c.l,
			i,
			numShards,
			c.gcConf,
			c.storage,
			c.metrics,
		)
		if err != nil {
			return err
		}

		c.shards = append(c.shards, shard)
	}

	return nil
}

func newGcStorageMetrics(r metrics.Registry) *collectorMetrics {
	return &collectorMetrics{
		deletedObjects:              r.WithTags(map[string]string{"kind": "deleted"}).Counter("objects.count"),
		deleteErrors:                r.Counter("delete_error.count"),
		collectExpiredTimer:         r.Timer("collect_expired.timer"),
		deleteTimer:                 r.Timer("delete.timer"),
		busyShards:                  r.IntGauge("busy_shards.gauge"),
		successShardIterationsTimer: r.WithTags(map[string]string{"status": "success"}).Timer("shard_iterations.timer"),
		failedShardIterationsTimer:  r.WithTags(map[string]string{"status": "failed"}).Timer("shard_iterations.timer"),
	}
}

func NewProfileGC(l xlog.Logger, gcConf *config.StorageConfig, profileStorage profilestorage.Storage, r metrics.Registry) (Collector, error) {
	r = r.WithTags(map[string]string{"storage_type": "profile"})

	gc := &collector{
		l:       l.WithName("profile_gc"),
		storage: profileStorage,
		gcConf:  gcConf,
		metrics: newGcStorageMetrics(r),
	}

	err := gc.initConcurrency()
	if err != nil {
		return nil, err
	}

	return gc, nil
}

func NewBinaryGC(l xlog.Logger, gcConf *config.StorageConfig, binaryStorage binary.Storage, r metrics.Registry) (Collector, error) {
	r = r.WithTags(map[string]string{"storage_type": "binary"})

	gc := &collector{
		l:       l.WithName("binary_gc"),
		storage: binaryStorage,
		gcConf:  gcConf,
		metrics: newGcStorageMetrics(r),
	}

	err := gc.initConcurrency()
	if err != nil {
		return nil, err
	}

	return gc, nil
}

func NewCollector(l xlog.Logger, r metrics.Registry, gcConf *config.StorageConfig, storageBundle *bundle.StorageBundle) (Collector, error) {
	switch gcConf.Type {
	case config.Profile:
		return NewProfileGC(l, gcConf, storageBundle.ProfileStorage, r)
	case config.Binary:
		return NewBinaryGC(l, gcConf, storageBundle.BinaryStorage.Binary(), r)
	default:
		return nil, fmt.Errorf("unsupported storage type %s", string(gcConf.Type))
	}
}

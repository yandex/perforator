package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/storage/gc/config"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	PageErrorsIterationLimit = 5
)

type storageGC struct {
	l        xlog.Logger
	storage  storage.Storage
	ttl      time.Duration
	pageSize uint64
	metrics  *collectorMetrics
}

func newStorageGC(l xlog.Logger, r metrics.Registry, conf config.StorageConfig, st storage.Storage) *storageGC {
	pageSize := uint64(conf.DeletePageSize)
	if pageSize == 0 {
		pageSize = 100
	}
	kind := string(conf.Type)
	return &storageGC{
		l: l.WithName(kind + "_gc").With(
			log.Duration("ttl", conf.TTL),
		),
		storage:  st,
		ttl:      conf.TTL,
		pageSize: pageSize,
		metrics:  newGcStorageMetrics(r.WithTags(map[string]string{"storage_type": kind})),
	}
}

func (c *storageGC) run(ctx context.Context, interval time.Duration) error {
	for {
		if err := context.Cause(ctx); err != nil {
			return err
		}
		err := c.collect(ctx)
		if err := context.Cause(ctx); err != nil {
			return err
		}
		if err != nil {
			c.l.Error(ctx, "Failed to collect expired objects", log.Error(err))
		}
		c.l.Info(ctx, "Waiting before next GC iteration", log.Duration("interval", interval))
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return context.Cause(ctx)
		case <-timer.C:
		}
	}
}

func (c *storageGC) processPage(
	ctx context.Context,
	pagination *util.Pagination,
) (emptyPage bool, err error) {
	tm := time.Now()

	metas, err := c.storage.CollectExpired(
		ctx,
		c.ttl,
		pagination,
	)
	if err != nil {
		return false, err
	}
	c.metrics.collectExpiredTimer.RecordDuration(time.Since(tm))

	if len(metas) == 0 {
		return true, nil
	}

	if err := context.Cause(ctx); err != nil {
		return false, err
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
		if meta.LastUsedTimestamp.Add(c.ttl).After(time.Now()) {
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

func (c *storageGC) collect(ctx context.Context) error {
	c.metrics.busy.Add(1)
	defer c.metrics.busy.Add(-1)
	defer func() {
		c.l.Info(ctx, "Finished collecting expired objects")
	}()

	var err error
	tm := time.Now()
	defer func() {
		if err != nil {
			c.metrics.failedIterationsTimer.RecordDuration(time.Since(tm))
		} else {
			c.metrics.successIterationsTimer.RecordDuration(time.Since(tm))
		}
	}()

	c.l.Info(ctx, "Collecting expired objects")

	pageSize := c.pageSize
	pagination := &util.Pagination{Offset: 0, Limit: pageSize}
	pageErrors := 0

	var deleteError error
	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
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

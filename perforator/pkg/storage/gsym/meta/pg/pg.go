package pg

import (
	"context"
	"fmt"
	"time"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	gsymmeta "github.com/yandex/perforator/perforator/pkg/storage/gsym/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	defaultCollectExpiredLimit = 10000
)

type storageMetrics struct {
	failedUpdateLastUsedTimestamp  metrics.Counter
	successUpdateLastUsedTimestamp metrics.Counter
}

type Storage struct {
	l       xlog.Logger
	cluster *hasql.Cluster
	opts    *Options
	metrics *storageMetrics
}

type Options struct {
	LastUsedTimestampUpdateInterval time.Duration
}

func (o *Options) fillDefault() {
	if o.LastUsedTimestampUpdateInterval == time.Duration(0) {
		o.LastUsedTimestampUpdateInterval = 5 * time.Minute
	}
}

func NewPostgresGSYMStorage(l xlog.Logger, reg metrics.Registry, cluster *hasql.Cluster, opts Options) *Storage {
	opts.fillDefault()

	return &Storage{
		l:       l.WithName("PostgresGSYMStorage"),
		cluster: cluster,
		opts:    &opts,
		metrics: &storageMetrics{
			failedUpdateLastUsedTimestamp:  reg.Counter("gsym.postgres.failed_update_last_used_timestamp.count"),
			successUpdateLastUsedTimestamp: reg.Counter("gsym.postgres.success_update_last_used_timestamp.count"),
		},
	}
}

type gsymRow struct {
	BuildID           string    `db:"build_id"`
	CompressedSize    uint64    `db:"compressed_size"`
	UncompressedSize  uint64    `db:"uncompressed_size"`
	LastUsedTimestamp time.Time `db:"last_used_timestamp"`
}

func rowToGSYMMeta(row *gsymRow) *gsymmeta.GSYMMeta {
	return &gsymmeta.GSYMMeta{
		BuildID:           row.BuildID,
		CompressedSize:    row.CompressedSize,
		UncompressedSize:  row.UncompressedSize,
		LastUsedTimestamp: row.LastUsedTimestamp,
	}
}

func (s *Storage) updateLastUsedTimestamp(ctx context.Context, buildIDs []string) error {
	if len(buildIDs) == 0 {
		return nil
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`UPDATE gsym
		 SET last_used_timestamp = NOW()
		 WHERE build_id = ANY($1)`,
		buildIDs,
	)

	return err
}

func (s *Storage) updateLastUsedTimestampsIfNeeded(ctx context.Context, rows []gsymRow) {
	var needsUpdate []string
	updateThreshold := time.Now().Add(-s.opts.LastUsedTimestampUpdateInterval)

	for _, row := range rows {
		if row.LastUsedTimestamp.IsZero() || row.LastUsedTimestamp.Before(updateThreshold) {
			needsUpdate = append(needsUpdate, row.BuildID)
		}
	}

	if len(needsUpdate) > 0 {
		s.l.Debug(ctx, "Updating timestamps for GSYMs", log.Array("build_ids", needsUpdate))
		if err := s.updateLastUsedTimestamp(ctx, needsUpdate); err != nil {
			s.metrics.failedUpdateLastUsedTimestamp.Inc()
			s.l.Warn(
				ctx, "Failed to update last used timestamp for GSYMs",
				log.Array("build_ids", needsUpdate),
				log.Error(err),
			)
		} else {
			s.metrics.successUpdateLastUsedTimestamp.Inc()
		}
	}
}

func (s *Storage) GetGSYMs(
	ctx context.Context,
	buildIDs []string,
) ([]*gsymmeta.GSYMMeta, error) {
	if len(buildIDs) == 0 {
		return []*gsymmeta.GSYMMeta{}, nil
	}

	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	rows := []gsymRow{}
	err = alive.DBx().SelectContext(
		ctx,
		&rows,
		`SELECT build_id, compressed_size, uncompressed_size, last_used_timestamp
			FROM gsym
			WHERE build_id = ANY($1)
			ORDER BY build_id ASC`,
		buildIDs,
	)
	if err != nil {
		return nil, err
	}

	s.updateLastUsedTimestampsIfNeeded(ctx, rows)

	res := make([]*gsymmeta.GSYMMeta, 0, len(rows))
	for _, row := range rows {
		res = append(res, rowToGSYMMeta(&row))
	}

	return res, nil
}

func (s *Storage) CollectExpiredGSYMs(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
) ([]*gsymmeta.GSYMMeta, error) {
	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	var offset uint64
	if pagination != nil {
		offset = pagination.Offset
	}
	var limit uint64 = defaultCollectExpiredLimit
	if pagination != nil && pagination.Limit != 0 {
		limit = pagination.Limit
	}
	limitStr := fmt.Sprintf("%d", limit)

	rows := []gsymRow{}
	err = alive.DBx().SelectContext(
		ctx,
		&rows,
		`SELECT build_id, compressed_size, uncompressed_size, last_used_timestamp
			FROM gsym
			WHERE last_used_timestamp <= $1
			ORDER BY build_id ASC LIMIT $2 OFFSET $3`,
		time.Now().Add(-ttl),
		limitStr,
		offset,
	)
	if err != nil {
		return nil, err
	}

	res := make([]*gsymmeta.GSYMMeta, 0, len(rows))
	for _, row := range rows {
		res = append(res, rowToGSYMMeta(&row))
	}

	return res, nil
}

func (s *Storage) RemoveGSYMs(
	ctx context.Context,
	buildIDs []string,
) error {
	l := s.l.With(log.Array("build_ids", buildIDs))

	l.Info(ctx, "Removing GSYMs")
	if len(buildIDs) == 0 {
		return nil
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`DELETE FROM gsym
			WHERE build_id = ANY($1)`,
		buildIDs,
	)
	if err != nil {
		s.l.Error(ctx, "Failed to remove GSYMs", log.Error(err))
		return err
	}

	s.l.Info(ctx, "Removed GSYMs")
	return nil
}

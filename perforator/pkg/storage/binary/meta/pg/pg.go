package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type storageMetrics struct {
	failedUpdateLastUsedTimestamp metrics.Counter
}

type Storage struct {
	l       xlog.Logger
	reg     metrics.Registry
	cluster *hasql.Cluster
	opts    *Options
	metrics *storageMetrics
}

type Options struct {
	DropStuckUploadPeriod time.Duration
}

func (o *Options) fillDefault() {
	if o.DropStuckUploadPeriod == time.Duration(0) {
		o.DropStuckUploadPeriod = 5 * time.Minute
	}
}

func NewPostgresBinaryStorage(l xlog.Logger, reg metrics.Registry, cluster *hasql.Cluster, opts Options) *Storage {
	opts.fillDefault()

	return &Storage{
		l:       l.WithName("PostgresBinaryStorage"),
		reg:     reg,
		cluster: cluster,
		opts:    &opts,
		metrics: &storageMetrics{
			failedUpdateLastUsedTimestamp: reg.Counter("binaries.postgres.failed_update_last_used_timestamp.count"),
		},
	}
}

func (s *Storage) updateInactiveUpload(ctx context.Context, tx *sqlx.Tx, binaryMeta *binarymeta.BinaryMeta) error {
	newRow := BinaryMetaToRow(binaryMeta)
	newRow.UploadStatus = string(binarymeta.InProgress)

	_, err := tx.ExecContext(
		ctx,
		`UPDATE binaries
			SET blob_size = $1,
				ts = $2,
				attributes = $3,
				upload_status = $4,
				last_used_timestamp = NOW()
			WHERE build_id = $5`,
		newRow.BlobSize,
		newRow.Timestamp,
		newRow.Attributes,
		newRow.UploadStatus,
		newRow.BuildID,
	)

	return err
}

func (s *Storage) storeBinary(ctx context.Context, tx *sqlx.Tx, binaryMeta *binarymeta.BinaryMeta) error {
	newRow := BinaryMetaToRow(binaryMeta)
	newRow.UploadStatus = string(binarymeta.InProgress)

	_, err := tx.ExecContext(
		ctx,
		`INSERT INTO binaries(build_id, blob_size, ts, attributes, upload_status, last_used_timestamp)
			VALUES ($1, $2, $3, $4, $5, NOW())`,
		newRow.BuildID,
		newRow.BlobSize,
		newRow.Timestamp,
		newRow.Attributes,
		newRow.UploadStatus,
	)

	return err
}

func (s *Storage) StoreBinary(
	ctx context.Context,
	binaryMeta *binarymeta.BinaryMeta,
) (binarymeta.Commiter, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := primary.DBx().BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var row BinaryRow
	err = tx.GetContext(
		ctx,
		&row,
		`SELECT build_id, blob_size, ts, attributes, upload_status, last_used_timestamp 
			FROM binaries 
			WHERE build_id = $1
			FOR UPDATE`,
		binaryMeta.BuildID,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if err == nil {
		if row.UploadStatus == string(binarymeta.Uploaded) {
			return nil, binarymeta.ErrAlreadyUploaded
		}

		if row.UploadStatus == string(binarymeta.InProgress) &&
			time.Since(row.LastUsedTimestamp) < s.opts.DropStuckUploadPeriod {
			return nil, binarymeta.ErrUploadInProgress
		}

		err = s.updateInactiveUpload(ctx, tx, binaryMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to update inactive previous upload: %w", err)
		}
	} else {
		err = s.storeBinary(ctx, tx, binaryMeta)
		if err != nil {
			return nil, fmt.Errorf("failed to store binary: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		s.l.Error(ctx, "Failed to commit tx to store binary", log.String("build_id", binaryMeta.BuildID), log.Error(err))
		return nil, err
	}

	s.l.Info(ctx, "Saved binary meta", log.String("build_id", binaryMeta.BuildID))

	return &committer{
		l:       s.l,
		buildID: binaryMeta.BuildID,
		cluster: s.cluster,
	}, nil
}

func (s *Storage) updateLastUsedTimestamp(
	ctx context.Context,
	buildIDs []string,
) error {
	if len(buildIDs) == 0 {
		return nil
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`UPDATE binaries 
			SET last_used_timestamp = NOW() 
			WHERE build_id = ANY($1) AND upload_status != $2`,
		buildIDs,
		string(binarymeta.InProgress),
	)

	return err
}

func (s *Storage) GetBinaries(
	ctx context.Context,
	buildIDs []string,
) ([]*binarymeta.BinaryMeta, error) {
	if len(buildIDs) == 0 {
		return []*binarymeta.BinaryMeta{}, nil
	}

	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	rows := []BinaryRow{}
	err = alive.DBx().SelectContext(
		ctx,
		&rows,
		`SELECT 
			b.build_id,
			b.blob_size,
			COALESCE(g.uncompressed_size, 0) gsym_blob_size,
			b.ts,
			b.attributes,
			b.upload_status,
			b.last_used_timestamp
		FROM binaries b LEFT OUTER JOIN gsym g on b.build_id = g.build_id
			WHERE b.build_id = ANY($1)
			ORDER BY b.build_id ASC`,
		buildIDs,
	)
	if err != nil {
		return nil, err
	}

	if err := s.updateLastUsedTimestamp(ctx, buildIDs); err != nil {
		s.metrics.failedUpdateLastUsedTimestamp.Inc()
		s.l.Warn(
			ctx, "Failed to update last used timestamp for binaries",
			log.Array("build_ids", buildIDs),
			log.Error(err),
		)
	}

	res := make([]*binarymeta.BinaryMeta, 0, len(rows))
	for _, row := range rows {
		res = append(res, RowToBinaryMeta(&row))
	}

	return res, nil
}

func (s *Storage) CollectExpiredBinaries(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
) ([]*binarymeta.BinaryMeta, error) {
	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	var offset uint64
	if pagination != nil {
		offset = pagination.Offset
	}
	limitStr := "ALL"
	if pagination != nil && pagination.Limit != 0 {
		limitStr = fmt.Sprintf("%d", pagination.Limit)
	}

	rows := []BinaryRow{}
	err = alive.DBx().SelectContext(
		ctx,
		&rows,
		`SELECT build_id, blob_size, ts, attributes, upload_status, last_used_timestamp
			FROM binaries
			WHERE last_used_timestamp <= $1
			ORDER BY build_id ASC LIMIT $2 OFFSET $3`,
		time.Now().Add(-ttl),
		limitStr,
		offset,
	)
	if err != nil {
		return nil, err
	}

	res := make([]*binarymeta.BinaryMeta, 0, len(rows))
	for _, row := range rows {
		res = append(res, RowToBinaryMeta(&row))
	}

	return res, nil
}

func (s *Storage) RemoveBinaries(
	ctx context.Context,
	buildIDs []string,
) error {
	l := s.l.With(log.Array("build_ids", buildIDs))

	l.Info(ctx, "Removing binaries")
	if len(buildIDs) == 0 {
		return nil
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`DELETE FROM binaries
			WHERE build_id = ANY($1)`,
		buildIDs,
	)
	if err != nil {
		s.l.Error(ctx, "Failed to remove binaries", log.Error(err))
		return err
	}

	s.l.Info(ctx, "Removed binaries")
	return nil
}

package pg

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jmoiron/sqlx"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

type storageMetrics struct {
	failedUpdateLastUsedTimestamp  metrics.Counter
	successUpdateLastUsedTimestamp metrics.Counter
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
	// Minimum interval between last_used_timestamp updates for the same binary
	LastUsedTimestampUpdateInterval time.Duration
}

func (o *Options) fillDefault() {
	if o.DropStuckUploadPeriod == time.Duration(0) {
		o.DropStuckUploadPeriod = binarymeta.DefaultUploadClaimStaleAfter
	}
	if o.LastUsedTimestampUpdateInterval == time.Duration(0) {
		o.LastUsedTimestampUpdateInterval = time.Hour
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
			failedUpdateLastUsedTimestamp:  reg.Counter("binaries.postgres.failed_update_last_used_timestamp.count"),
			successUpdateLastUsedTimestamp: reg.Counter("binaries.postgres.success_update_last_used_timestamp.count"),
		},
	}
}

func (s *Storage) updateInactiveUpload(
	ctx context.Context,
	tx *sqlx.Tx,
	buildID string,
	timestamp time.Time,
	uncompressedSize uint64,
	compression compressionpb.CompressionMethod,
	attributes map[string]string,
) error {
	compressionStr, err := compressionMethodToString(ctx, compression)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`UPDATE binaries
			SET uncompressed_size = $1,
				blob_size = 0,
				ts = $2,
				upload_status = $3,
				last_used_timestamp = NOW(),
				compression = $4,
				attributes = $5
			WHERE build_id = $6`,
		uncompressedSize,
		timestamp,
		binarymeta.InProgress,
		compressionStr,
		attributes,
		buildID,
	)

	return err
}

func (s *Storage) storeBinary(
	ctx context.Context,
	tx *sqlx.Tx,
	buildID string,
	timestamp time.Time,
	uncompressedSize uint64,
	compression compressionpb.CompressionMethod,
	attributes map[string]string,
) error {
	compressionStr, err := compressionMethodToString(ctx, compression)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO binaries(build_id, blob_size, uncompressed_size, ts, attributes, upload_status, last_used_timestamp, compression)
			VALUES ($1, 0, $2, $3, $4, $5, NOW(), $6)`,
		buildID,
		uncompressedSize,
		timestamp,
		attributes,
		binarymeta.InProgress,
		compressionStr,
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "binaries_pkey" {
		return binarymeta.ErrUploadInProgress
	}

	return err
}

func (s *Storage) BeginUpload(
	ctx context.Context,
	buildID string,
	timestamp time.Time,
	opts ...binarymeta.Option,
) (binarymeta.UploadClaim, error) {
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
		`SELECT build_id, blob_size, ts, attributes, upload_status, last_used_timestamp, compression, uncompressed_size
			FROM binaries
			WHERE build_id = $1
			FOR UPDATE`,
		buildID,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	options := binarymeta.DefaultBinaryMetaOptions()
	for _, opt := range opts {
		opt.Apply(options)
	}

	uncompressedSize := options.Compression.UnverifiedUncompressedSize
	compression := options.Compression.Method

	if compression != compressionpb.CompressionMethod_None && uncompressedSize == 0 {
		return nil, fmt.Errorf("uncompressed size is required when compression is set: %s", compression.String())
	}

	if err == nil {
		if row.UploadStatus == string(binarymeta.Uploaded) {
			return nil, binarymeta.ErrAlreadyUploaded
		}

		if row.UploadStatus == string(binarymeta.InProgress) &&
			time.Since(row.LastUsedTimestamp) < s.opts.DropStuckUploadPeriod {
			return nil, binarymeta.ErrUploadInProgress
		}

		err = s.updateInactiveUpload(ctx, tx, buildID, timestamp, uncompressedSize, compression, options.Attributes.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to update inactive previous upload: %w", err)
		}
	} else {
		err = s.storeBinary(ctx, tx, buildID, timestamp, uncompressedSize, compression, options.Attributes.Attributes)
		if err != nil {
			return nil, fmt.Errorf("failed to store binary: %w", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		s.l.Error(ctx, "Failed to commit tx to store binary", log.String("build_id", buildID), log.Error(err))
		return nil, err
	}

	s.l.Info(ctx, "Saved binary meta", log.String("build_id", buildID))

	return &uploadClaim{
		l:           s.l,
		buildID:     buildID,
		cluster:     s.cluster,
		compression: compression,
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
		 WHERE build_id = ANY($1) 
		 AND upload_status != $2`,
		buildIDs,
		string(binarymeta.InProgress),
	)

	return err
}

func (s *Storage) updateLastUsedTimestampsIfNeeded(ctx context.Context, rows []BinaryRow) {
	var needsUpdate []string
	now := time.Now()
	updateThreshold := now.Add(-s.opts.LastUsedTimestampUpdateInterval)

	for _, row := range rows {
		if row.UploadStatus != string(binarymeta.InProgress) &&
			(row.LastUsedTimestamp.IsZero() || row.LastUsedTimestamp.Before(updateThreshold)) {
			needsUpdate = append(needsUpdate, row.BuildID)
		}
	}

	if len(needsUpdate) > 0 {
		s.l.Debug(ctx, "Updating timestamps for binaries", log.Array("build_ids", needsUpdate))
		if err := s.updateLastUsedTimestamp(ctx, needsUpdate); err != nil {
			s.metrics.failedUpdateLastUsedTimestamp.Inc()
			s.l.Warn(
				ctx, "Failed to update last used timestamp for binaries",
				log.Array("build_ids", needsUpdate),
				log.Error(err),
			)
		} else {
			s.metrics.successUpdateLastUsedTimestamp.Inc()
		}
	}
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
			b.last_used_timestamp,
			b.compression,
			b.uncompressed_size
		FROM binaries b LEFT OUTER JOIN gsym g on b.build_id = g.build_id
			WHERE b.build_id = ANY($1)
			ORDER BY b.build_id ASC`,
		buildIDs,
	)
	if err != nil {
		return nil, err
	}

	s.updateLastUsedTimestampsIfNeeded(ctx, rows)

	res := make([]*binarymeta.BinaryMeta, 0, len(rows))
	for _, row := range rows {
		meta, err := RowToBinaryMeta(&row)
		if err != nil {
			return nil, err
		}
		res = append(res, meta)
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
		`SELECT build_id, blob_size, ts, attributes, upload_status, last_used_timestamp, compression, uncompressed_size
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
		meta, err := RowToBinaryMeta(&row)
		if err != nil {
			return nil, err
		}
		res = append(res, meta)
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

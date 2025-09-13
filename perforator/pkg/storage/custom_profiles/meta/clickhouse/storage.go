package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/Masterminds/squirrel"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	clickhouse_helper "github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiles/meta"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

var _ meta.Storage = (*Storage)(nil)

type Storage struct {
	l    xlog.Logger
	conn *clickhouse_helper.Connection

	profilesInserted       metrics.Counter
	profilesInsertedFailed metrics.Counter
	profilesInsertionTimer metrics.Timer
}

func NewStorage(
	l xlog.Logger,
	metrics metrics.Registry,
	conn *clickhouse_helper.Connection,
) *Storage {
	l = l.WithName("Clickhouse.CustomProfiles")

	metrics = metrics.WithPrefix("custom_profiles_clickhouse")
	return &Storage{
		l:    l,
		conn: conn,

		profilesInserted:       metrics.WithTags(map[string]string{"status": "success"}).Counter("profiles.insertion.count"),
		profilesInsertedFailed: metrics.WithTags(map[string]string{"status": "fail"}).Counter("profiles.insertion.count"),
		profilesInsertionTimer: metrics.Timer("profiles.insertion.timer"),
	}
}

func (s *Storage) StoreCustomProfile(
	ctx context.Context,
	meta *meta.CustomProfileMeta,
) error {
	profile := customProfileModelFromMeta(meta)

	err := s.asyncInsertProfile(ctx, profile)
	if err != nil {
		s.l.Error(
			ctx,
			"Failed to store custom profile",
			log.Error(err),
			log.String("profile_id", profile.ID),
		)
		s.profilesInsertedFailed.Inc()
		return err
	}

	s.profilesInserted.Inc()
	return nil
}

func (s *Storage) asyncInsertProfile(ctx context.Context, profile *CustomProfileRow) error {
	start := time.Now()
	defer func() {
		s.profilesInsertionTimer.RecordDuration(time.Since(start))
	}()

	builder := squirrel.Insert("custom_profiles").
		Columns("id", "operation_id", "from_timestamp", "to_timestamp", "build_ids", "attributes").
		Values(profile.ID, profile.OperationID, profile.FromTimestamp, profile.ToTimestamp, profile.BuildIDs, profile.Attributes)

	sql, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	s.l.Debug(ctx, "Executing custom profile async insert",
		log.String("sql", sql),
		log.String("profile_id", profile.ID))

	err = s.conn.AsyncInsert(ctx, sql, true /*=wait*/, args...)
	if err != nil {
		return fmt.Errorf("failed to async insert profile: %w", err)
	}

	return nil
}

func scanCustomProfileRow(rows driver.Rows) (*meta.CustomProfileMeta, error) {
	row := CustomProfileRow{}
	if err := rows.ScanStruct(&row); err != nil {
		return nil, fmt.Errorf("failed to scan struct from row: %w", err)
	}
	return customProfileMetaFromModel(&row), nil
}

func (s *Storage) GetOperationProfiles(
	ctx context.Context,
	operationID string,
) ([]*meta.CustomProfileMeta, error) {
	builder := squirrel.Select("id", "operation_id", "from_timestamp", "to_timestamp", "build_ids", "attributes").
		From("custom_profiles").
		Where(squirrel.Eq{"operation_id": operationID}).
		OrderBy("from_timestamp ASC")

	sql, args, err := builder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select query: %w", err)
	}

	return clickhouse_helper.QueryWithRetries(s.l, ctx, s.conn, sql, scanCustomProfileRow, args...)
}

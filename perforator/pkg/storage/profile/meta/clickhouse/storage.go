package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/env"
	"github.com/yandex/perforator/perforator/pkg/sqlbuilder"
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	MaxRowsToRead = 300000000
)

var _ meta.Storage = (*Storage)(nil)

type Storage struct {
	l    xlog.Logger
	conf *Config
	conn *clickhouse.Connection

	batchsize     int
	batchinterval time.Duration

	profilechan chan *ProfileRow
	senderonce  sync.Once

	rowsSent    metrics.Counter
	rowsLost    metrics.Counter
	batchesSent metrics.Counter
	batchesLost metrics.Counter
}

func NewStorage(
	l xlog.Logger,
	metrics metrics.Registry,
	conn *clickhouse.Connection,
	conf *Config,
) (*Storage, error) {
	l = l.WithName("clickhouse")

	metrics = metrics.WithPrefix("clickhouse")
	return &Storage{
		l:             l,
		conf:          conf,
		conn:          conn,
		batchsize:     int(conf.Batching.Size),
		batchinterval: conf.Batching.Interval,

		rowsSent:    metrics.Counter("rows.sent.count"),
		rowsLost:    metrics.Counter("rows.lost.count"),
		batchesSent: metrics.Counter("batches.sent.count"),
		batchesLost: metrics.Counter("batches.lost.count"),
	}, nil
}

func scanServiceRow(rows driver.Rows) (*meta.ServiceMetadata, error) {
	row := ServiceRow{}
	if err := rows.ScanStruct(&row); err != nil {
		return nil, fmt.Errorf("failed to scan struct from row: %w", err)
	}
	return serviceMetaFromModel(&row), nil
}

// ListServices implements meta.Storage.
func (s *Storage) ListServices(
	ctx context.Context,
	query *meta.ServiceQuery,
) ([]*meta.ServiceMetadata, error) {
	builder := sqlbuilder.Select().
		Values("service,max(timestamp) AS max_timestamp, sum(1) AS profile_count").
		From("profiles").
		GroupBy("service").
		OrderBy(makeOrderBy(&query.SortOrder))

	if query.Limit != 0 {
		builder.Limit(query.Limit)
	}
	if query.Offset != 0 {
		builder.Offset(query.Offset)
	}
	if query.Regex != nil {
		builder.Where(fmt.Sprintf("match(service, '%s')", sqlbuilder.Escape(*query.Regex)))
	}
	if query.MaxStaleAge != nil {
		builder.Having(fmt.Sprintf("max_timestamp >= %.3f", getTimestampFraction(time.Now().Add(-*query.MaxStaleAge))))
	}

	sql, err := builder.Query()
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Selecting services from clickhouse", log.String("sql", sql))
	return clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, scanServiceRow)
}

func suggestSupported(column string) bool {
	return !nonStringColumns[column]
}

func scanSuggestionRow(rows driver.Rows) (*meta.Suggestion, error) {
	var value string
	if err := rows.Scan(&value); err != nil {
		return nil, fmt.Errorf("failed to scan string from row: %w", err)
	}
	return &meta.Suggestion{Value: value}, nil
}

// ListSuggestions implements meta.Storage.
func (s *Storage) ListSuggestions(
	ctx context.Context,
	query *meta.SuggestionsQuery,
) ([]*meta.Suggestion, error) {
	var columns []string
	if env.IsEnvMatcherField(query.Field) {
		columns = []string{envsColumn}
	} else {
		columns = labelsToColumns[query.Field]
	}
	if len(columns) == 0 {
		s.l.Debug(
			ctx,
			"Cannot find suggestions for unknown field",
			log.String("field", query.Field),
		)
		return nil, nil
	}
	if len(columns) > 1 {
		s.l.Debug(
			ctx,
			fmt.Sprintf(
				"More than one column matching field `%s`. Using only the first one",
				query.Field,
			),
		)
	}
	column := columns[0]
	if !suggestSupported(column) {
		return nil, nil
	}

	profileQuery := &meta.ProfileQuery{
		Selector: query.Selector,
	}
	builder, err := makeSelectProfilesQueryBuilder(profileQuery)
	if err != nil {
		return nil, err
	}

	if column == envsColumn {
		column = "envValue"
		envKey, ok := env.BuildEnvKeyFromMatcherField(query.Field)
		if !ok {
			return nil, fmt.Errorf("failed to build env key from query field: %v", query.Field)
		}
		prefix := env.BuildConcatenatedEnv(envKey, "")
		builder.Values(fmt.Sprintf(
			`if(isNotNull(arrayFirstOrNull(s -> startsWith(s, '%s'), %s) as envElement), substring(envElement, length('%s') + 1), NULL) as %s`,
			prefix,
			envsColumn,
			prefix,
			column,
		))
		builder.Where("isNotNull(envElement)")
	} else {
		builder.Values(column)
	}

	builder.
		GroupBy(column).
		OrderBy(&sqlbuilder.OrderBy{
			Columns:    []string{column},
			Descending: false,
		})

	if query.Regex != nil {
		builder.Where(fmt.Sprintf("match(%s, '%s')", column, sqlbuilder.Escape(*query.Regex)))
	}
	if query.Limit != 0 {
		builder.Limit(query.Limit)
	}
	if query.Offset != 0 {
		builder.Offset(query.Offset)
	}

	// to prevent full scans
	builder.
		Settings(fmt.Sprintf("max_rows_to_read=%d", MaxRowsToRead)).
		Settings("read_overflow_mode='break'")

	sql, err := builder.Query()
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Searching for suggestions in clickhouse", log.String("sql", sql))
	return clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, scanSuggestionRow)
}

// StoreProfile implements meta.Storage.
func (s *Storage) StoreProfile(
	ctx context.Context,
	meta *meta.ProfileMetadata,
) error {
	s.senderonce.Do(func() {
		s.setupBatcher(context.Background())
	})

	profile := profileModelFromMeta(meta)

	select {
	case s.profilechan <- profile:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func scanProfileRow(rows driver.Rows) (*meta.ProfileMetadata, error) {
	row := ProfileRow{}
	if err := rows.ScanStruct(&row); err != nil {
		return nil, fmt.Errorf("failed to scan struct from row: %w", err)
	}
	return profileMetaFromModel(&row), nil
}

// SelectProfiles implements meta.Storage.
func (s *Storage) SelectProfiles(
	ctx context.Context,
	query *meta.ProfileQuery,
) ([]*meta.ProfileMetadata, error) {
	sql, err := buildSelectProfilesQuery(query)
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Select profiles", log.String("sql", sql))

	return clickhouse.QueryWithRetries(s.l, ctx, s.conn, sql, scanProfileRow)
}

// GetProfiles implements meta.Storage.
func (s *Storage) GetProfiles(
	ctx context.Context,
	profileIDs []string,
) ([]*meta.ProfileMetadata, error) {
	builder := sqlbuilder.Select().
		Values(AllColumns).
		From("profiles").
		Where(fmt.Sprintf("id IN [%s]", sqlbuilder.BuildQuotedList(profileIDs))).
		Where("expired = false")

	query, err := builder.Query()
	if err != nil {
		return nil, err
	}

	s.l.Debug(ctx, "Get profiles from clickhouse", log.String("sql", query))
	return clickhouse.QueryWithRetries(s.l, ctx, s.conn, query, scanProfileRow)
}

// RemoveProfiles implements meta.Storage.
func (s *Storage) RemoveProfiles(
	ctx context.Context,
	profileIDs []string,
) error {
	return fmt.Errorf("clickhouse storage does not support profile removing")
}

// CollectExpiredProfiles implements meta.Storage.
func (s *Storage) CollectExpiredProfiles(
	ctx context.Context,
	ttl time.Duration,
	pagination *util.Pagination,
	shardParams storage.ShardParams,
) ([]*meta.ProfileMetadata, error) {
	return nil, fmt.Errorf("clickhouse storage does not support profile removing")
}

func (s *Storage) setupBatcher(ctx context.Context) {
	s.profilechan = make(chan *ProfileRow, 1000)
	go func() { _ = s.runBatcher(ctx) }()
}

func (s *Storage) runBatcher(ctx context.Context) error {
	batch := make([]*ProfileRow, 0, s.batchsize)
	ticker := time.NewTicker(s.batchinterval)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case profile := <-s.profilechan:
			batch = append(batch, profile)
			if len(batch) >= s.batchsize {
				batch = s.sendBatch(ctx, batch)
			}
		case <-ticker.C:
			batch = s.sendBatch(ctx, batch)
		}
	}
}

func (s *Storage) sendBatch(ctx context.Context, rows []*ProfileRow) (next []*ProfileRow) {
	err := s.sendBatchImpl(ctx, rows)
	if err != nil {
		s.l.Error(ctx, "Failed to send batch",
			log.Error(err),
			log.Int("lost_profiles", len(rows)),
		)
		s.batchesLost.Inc()
		s.rowsLost.Add(int64(len(rows)))
	} else {
		s.batchesSent.Inc()
		s.rowsSent.Add(int64(len(rows)))
	}

	return rows[:0]
}

func (s *Storage) sendBatchImpl(ctx context.Context, rows []*ProfileRow) error {
	if len(rows) == 0 {
		return nil
	}

	query, err := buildInsertQuery(rows)
	if err != nil {
		return fmt.Errorf("failed to build insert query: %w", err)
	}

	s.l.Debug(ctx, "Executing batch insert", log.String("query", query))

	return s.conn.Exec(ctx, query)
}

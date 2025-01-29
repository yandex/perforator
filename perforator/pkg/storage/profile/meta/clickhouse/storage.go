package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
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
	conn driver.Conn

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
	conn driver.Conn,
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

func scanServices(rows driver.Rows) ([]*meta.ServiceMetadata, error) {
	result := []*meta.ServiceMetadata{}
	row := ServiceRow{}

	for rows.Next() {
		if err := rows.ScanStruct(&row); err != nil {
			return nil, fmt.Errorf("failed to scan struct from row: %w", err)
		}
		result = append(result, serviceMetaFromModel(&row))
	}

	return result, rows.Err()
}

func Retry[T any](f func() (T, error), retries uint32) (val T, err error) {
	if retries == 0 {
		retries = 1
	}

	for i := uint32(0); i < retries; i++ {
		val, err = f()
		if err == nil {
			return
		}
	}

	return
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

	return Retry(func() ([]*meta.ServiceMetadata, error) {
		s.l.Debug(ctx, "Selecting services from clickhouse", log.String("sql", sql))
		rows, err := s.conn.Query(ctx, sql)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return scanServices(rows)
	}, s.conf.ReadRequestRetries)
}

func suggestSupported(column string) bool {
	return !nonStringColumns[column]
}

func scanSuggestions(rows driver.Rows) ([]*meta.Suggestion, error) {
	result := []*meta.Suggestion{}
	var value string

	for rows.Next() {
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan string from row: %w", err)
		}
		result = append(result, &meta.Suggestion{Value: value})
	}

	return result, rows.Err()
}

// ListSuggestions implements meta.Storage.
func (s *Storage) ListSuggestions(
	ctx context.Context,
	query *meta.SuggestionsQuery,
) ([]*meta.Suggestion, error) {
	columns := labelsToColumns[query.Field]
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
	builder, err := makeSelectProfilesQueryBuilder(profileQuery, false)
	if err != nil {
		return nil, err
	}

	builder.
		Values(column).
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

	return Retry(func() ([]*meta.Suggestion, error) {
		s.l.Debug(ctx, "Searching for suggestions in clickhouse", log.String("sql", sql))
		rows, err := s.conn.Query(ctx, sql)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		return scanSuggestions(rows)
	}, s.conf.ReadRequestRetries)
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

func scanProfiles(rows driver.Rows) ([]*meta.ProfileMetadata, error) {
	result := []*meta.ProfileMetadata{}
	row := ProfileRow{}

	for rows.Next() {
		if err := rows.ScanStruct(&row); err != nil {
			return nil, fmt.Errorf("failed to scan struct from row: %w", err)
		}
		result = append(result, profileMetaFromModel(&row))
	}

	return result, rows.Err()
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

	return Retry(func() ([]*meta.ProfileMetadata, error) {
		s.l.Debug(ctx, "Selecting profiles from clickhouse", log.String("sql", sql))
		rows, err := s.conn.Query(ctx, sql)
		if err != nil {
			return nil, fmt.Errorf("failed query: %w", err)
		}
		defer rows.Close()

		return scanProfiles(rows)
	}, s.conf.ReadRequestRetries)
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

	return Retry(func() ([]*meta.ProfileMetadata, error) {
		s.l.Debug(ctx, "Get profiles from clickhouse", log.String("sql", query))
		rows, err := s.conn.Query(ctx, query)
		if err != nil {
			return nil, fmt.Errorf("failed query: %w", err)
		}
		defer rows.Close()

		return scanProfiles(rows)
	}, s.conf.ReadRequestRetries)
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
	batch, err := s.conn.PrepareBatch(
		ctx,
		fmt.Sprintf(
			`INSERT INTO profiles (%s) SETTINGS async_insert=1, wait_for_async_insert=1`,
			AllColumns,
		),
	)
	if err != nil {
		return fmt.Errorf("failed to prepare batch: %w", err)
	}

	defer func() { _ = batch.Abort() }()

	for i, row := range rows {
		row := row
		err = batch.AppendStruct(row)
		if err != nil {
			return fmt.Errorf("failed to serialize row %d: %w", i, err)
		}
	}

	return batch.Send()
}

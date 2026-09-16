package generations

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	hasql "golang.yandex/hasql/sqlx"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/foreach"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

const (
	coalescedStatusColumn = "COALESCE(status, 'finished') AS status"
)

var (
	generationStatusMap = map[string]perforator.ClusterTopGenerationStatus{
		StatusFinished:  perforator.ClusterTopGenerationStatus_COMPLETED,
		StatusScheduled: perforator.ClusterTopGenerationStatus_IN_PROGRESS,
	}
)

type PostgresGenerationsStorage struct {
	logger  xlog.Logger
	cluster *hasql.Cluster
}

type clusterTopGenerationRow struct {
	ID     uint32    `db:"id"`
	From   time.Time `db:"from_ts"`
	To     time.Time `db:"to_ts"`
	Status string    `db:"status"`
}

func NewStorage(
	logger xlog.Logger,
	cluster *hasql.Cluster,
) *PostgresGenerationsStorage {
	return &PostgresGenerationsStorage{
		logger:  logger.WithName("ClusterTopGenerationsStorage"),
		cluster: cluster,
	}
}

func mapToProto(rows []*clusterTopGenerationRow) []*perforator.ClusterTopGeneration {
	return foreach.Map(rows, func(row *clusterTopGenerationRow) *perforator.ClusterTopGeneration {
		return &perforator.ClusterTopGeneration{
			ID:               row.ID,
			From:             timestamppb.New(row.From),
			To:               timestamppb.New(row.To),
			GenerationStatus: generationStatusMap[row.Status],
		}
	})
}

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

func (s *PostgresGenerationsStorage) ListGenerations(ctx context.Context) ([]*perforator.ClusterTopGeneration, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for primary: %w", err)
	}
	query := psql.Select("id", "from_ts", "to_ts", coalescedStatusColumn).
		From("cluster_top_generations").
		Where("status IS DISTINCT FROM 'deleting'").
		OrderBy("id DESC")

	sql, args, err := query.ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	s.logger.Debug(ctx, "Listing generations in postgres", log.String("sql", sql))

	var rows []*clusterTopGenerationRow
	err = primary.DBx().SelectContext(ctx, &rows, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("can't list generations: %w", err)
	}

	return mapToProto(rows), nil
}

func (s *PostgresGenerationsStorage) Exists(ctx context.Context, id uint32) (bool, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return false, err
	}
	var exists bool
	err = primary.DBx().GetContext(ctx, &exists, `
SELECT EXISTS (
    SELECT 1 FROM cluster_top_generations WHERE id = $1 AND status IS DISTINCT FROM 'deleting'
)`, id)
	return exists, err
}

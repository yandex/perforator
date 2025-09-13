package cluster_top

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	hasql "golang.yandex/hasql/sqlx"
)

type queueItem struct {
	Service    string `db:"service"`
	Generation uint32 `db:"generation"`
}

type clusterTopGeneration struct {
	From time.Time `db:"from_ts"`
	To   time.Time `db:"to_ts"`
}

func (g *clusterTopGeneration) toTimeRange() TimeRange {
	return TimeRange{
		From: g.From,
		To:   g.To,
	}
}

type PgServiceProcessingHandler struct {
	serviceName string
	generation  int
	timeRange   TimeRange

	tx *sqlx.Tx
}

func newPgServiceProcessingHandler(
	service string,
	generation int,
	timeRange TimeRange,
	tx *sqlx.Tx,
) *PgServiceProcessingHandler {
	return &PgServiceProcessingHandler{
		serviceName: service,
		generation:  generation,
		timeRange:   timeRange,
		tx:          tx,
	}
}

func (h *PgServiceProcessingHandler) GetServiceName() string {
	return h.serviceName
}

func (h *PgServiceProcessingHandler) GetGeneration() int {
	return h.generation
}

func (h *PgServiceProcessingHandler) GetTimeRange() TimeRange {
	return h.timeRange
}

func (h *PgServiceProcessingHandler) Finalize(ctx context.Context, processingErr error) {
	newStatus := "done"
	if processingErr != nil {
		newStatus = "failed"
	}
	_, finalizationErr := h.tx.ExecContext(
		ctx,
		`UPDATE cluster_top_services
		SET
			status=$2
		WHERE
			service=$1 AND generation=$3
		`,
		h.GetServiceName(),
		newStatus,
		h.GetGeneration(),
	)
	if finalizationErr == nil {
		_ = h.tx.Commit()
	} else {
		_ = h.tx.Rollback()
	}
}

type PgServiceSelector struct {
	cluster *hasql.Cluster
}

func NewPgServiceSelector(cluster *hasql.Cluster) *PgServiceSelector {
	return &PgServiceSelector{
		cluster: cluster,
	}
}

func (s *PgServiceSelector) SelectService(ctx context.Context, heavy bool) (ServiceProcessingHandler, error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, err
	}

	tx, err := primary.DBx().BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var queueItem queueItem
	err = tx.GetContext(
		ctx,
		&queueItem,
		`SELECT
			service,
			generation
		FROM cluster_top_services
		WHERE
			status='ready' AND heavy=$1
		ORDER BY profiles_count DESC LIMIT 1
		FOR UPDATE SKIP LOCKED
		`,
		heavy,
	)
	if err != nil {
		return nil, err
	}

	var clusterTopGeneration clusterTopGeneration
	err = tx.GetContext(
		ctx,
		&clusterTopGeneration,
		`SELECT
			from_ts,
			to_ts
		FROM cluster_top_generations
		WHERE
			id=$1
		`,
		queueItem.Generation,
	)

	return newPgServiceProcessingHandler(
		queueItem.Service,
		int(queueItem.Generation),
		clusterTopGeneration.toTimeRange(),
		tx,
	), nil
}

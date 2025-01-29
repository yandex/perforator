package app

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const maxProcessingAttempts = 3

type QueueItem struct {
	BuildID            string    `db:"build_id"`
	CreatedAt          time.Time `db:"created_at"`
	Status             string    `db:"status"`
	ProcessingAttempts int       `db:"processing_attempts"`
}

type PgTransactionHandler struct {
	l xlog.Logger

	queueItem QueueItem
	tx        *sqlx.Tx
}

func NewPgTransactionHandler(l xlog.Logger, queueItem QueueItem, tx *sqlx.Tx) *PgTransactionHandler {
	return &PgTransactionHandler{
		l:         l,
		queueItem: queueItem,
		tx:        tx,
	}
}

func (h *PgTransactionHandler) GetBinaryID() string {
	return h.queueItem.BuildID
}

func (h *PgTransactionHandler) Finalize(ctx context.Context, processingErr error) {
	processingAttemps := h.queueItem.ProcessingAttempts + 1

	var newStatus string
	var lastError string

	if processingErr == nil {
		newStatus = "done"
	} else {
		if processingAttemps == maxProcessingAttempts {
			newStatus = "failed"
		} else {
			newStatus = "ready"
		}

		lastError = processingErr.Error()
	}

	_, finalizationErr := h.tx.ExecContext(
		ctx,
		`UPDATE binary_processing_queue
		SET
			status=$2,
			processing_attempts=$3,
			last_error=$4
		WHERE
			build_id=$1
		`,
		h.queueItem.BuildID,
		newStatus,
		processingAttemps,
		lastError,
	)

	h.l.Info(ctx, "Updating binary queue",
		log.String("build_id", h.queueItem.BuildID),
		log.String("status", newStatus),
	)

	if finalizationErr == nil {
		err := h.tx.Commit()
		if err != nil {
			h.l.Warn(ctx, "Failed to commit binary queue update", log.Error(err))
		}
	} else {
		h.l.Warn(ctx, "Failed to update the queue", log.Error(finalizationErr))
		_ = h.tx.Rollback()
	}
}

func (h *PgTransactionHandler) SetGSYMSizes(ctx context.Context, uncompressedSize uint64, compressedSize uint64) error {
	_, err := h.tx.ExecContext(
		ctx,
		`INSERT INTO gsym(build_id, uncompressed_size, compressed_size)
		VALUES ($1, $2, $3)
		ON CONFLICT(build_id) DO UPDATE
		SET
			uncompressed_size = $2,
			compressed_size = $3
		`,
		h.queueItem.BuildID,
		uncompressedSize,
		compressedSize,
	)

	return err
}

func (h *PgTransactionHandler) rollback() {
	_ = h.tx.Rollback()
}

func (h *PgTransactionHandler) commit() {
	_ = h.tx.Commit()
}

type PgBinarySelector struct {
	l xlog.Logger

	cluster *hasql.Cluster
}

func NewPgBinarySelector(l xlog.Logger, cluster *hasql.Cluster) (*PgBinarySelector, error) {
	if cluster == nil {
		return nil, fmt.Errorf("failed to initialize binary selector: cluster is nil")
	}

	return &PgBinarySelector{
		l:       l,
		cluster: cluster,
	}, nil
}

func (s *PgBinarySelector) SelectBinary(ctx context.Context) (BinaryTranscationHandler, error) {
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

	var queueItem QueueItem
	err = tx.GetContext(
		ctx,
		&queueItem,
		`SELECT
			build_id, created_at, status, processing_attempts
		FROM binary_processing_queue
		WHERE
			status='ready'
		ORDER BY created_at DESC LIMIT 1
		FOR UPDATE SKIP LOCKED
		`,
	)
	if err != nil {
		return nil, err
	}

	return NewPgTransactionHandler(s.l, queueItem, tx), nil
}

func (s *PgBinarySelector) GetQueuedBinariesCount(ctx context.Context) (uint64, error) {
	secondary, err := s.cluster.WaitForStandbyPreferred(ctx)
	if err != nil {
		return 0, err
	}

	var readyBinariesCount uint64
	err = secondary.DBx().GetContext(
		ctx,
		&readyBinariesCount,
		`SELECT
			COUNT(*)
		FROM binary_processing_queue
		WHERE
			status='ready'
		`,
	)
	if err != nil {
		return 0, err
	}

	return readyBinariesCount, nil
}

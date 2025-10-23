package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	hasql "golang.yandex/hasql/sqlx"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

const (
	operationsTable            = "custom_profiling_operations"
	defaultListOperationsLimit = 10000
)

type Storage struct {
	logger  xlog.Logger
	cluster *hasql.Cluster
}

func NewStorage(
	logger xlog.Logger,
	cluster *hasql.Cluster,
) *Storage {
	return &Storage{
		logger:  logger.WithName("CustomProfilingOperationStorage"),
		cluster: cluster,
	}
}

func (s *Storage) InsertOperation(ctx context.Context, id custom_profiling_operation.OperationID, spec *cpo_proto.OperationSpec) (*cpo_proto.Operation, error) {
	if spec == nil {
		return nil, errors.New("nil spec")
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for primary replica: %w", err)
	}

	createdAt := timestamppb.Now()
	operation := &cpo_proto.Operation{
		ID: string(id),
		Meta: &cpo_proto.OperationMeta{
			CreatedAt: createdAt,
		},
		Spec: spec,
	}

	row, err := operationToRow(operation)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal row: %w", err)
	}

	// Insert or return existing operation in one query
	var resultRow operationRow
	err = primary.DBx().GetContext(
		ctx,
		&resultRow,
		`INSERT INTO `+operationsTable+` (id, meta, spec, status, target_state)
            VALUES ($1, $2, $3, $4, $5)
            ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
            RETURNING id, meta, spec, status, target_state`,
		row.ID, row.Meta, row.Spec, row.Status, row.TargetState,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert or get: %w", err)
	}

	result, err := rowToOperation(&resultRow)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result: %w", err)
	}

	return result, nil
}

func (s *Storage) GetOperation(ctx context.Context, id custom_profiling_operation.OperationID) (*cpo_proto.Operation, error) {
	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for alive replica: %w", err)
	}

	var row operationRow
	err = alive.DBx().GetContext(
		ctx,
		&row,
		`SELECT id, meta, spec, status, target_state FROM `+operationsTable+` WHERE id = $1`,
		string(id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	operation, err := rowToOperation(&row)
	if err != nil {
		return nil, err
	}

	return operation, nil
}

func (s *Storage) StopOperation(ctx context.Context, id custom_profiling_operation.OperationID) error {
	return s.updateOperation(ctx, id, func(operation *cpo_proto.Operation) error {
		if operation.Status != nil && custom_profiling_operation.IsTerminalState(operation.Status.State) {
			return errors.New("operation already finished")
		}

		now := timestamppb.Now()
		operation.TargetState = &cpo_proto.OperationTargetState{
			State:     cpo_proto.OperationState_Stopped,
			UpdatedAt: now,
		}

		return nil
	})
}

func (s *Storage) UpdateOperationStatus(
	ctx context.Context,
	id custom_profiling_operation.OperationID,
	newStatus *cpo_proto.OperationStatus,
) error {
	if newStatus == nil {
		return errors.New("new status is nil")
	}

	if newStatus.Timestamp == nil || newStatus.Timestamp.AsTime().IsZero() {
		return errors.New("new status timestamp is not set")
	}

	return s.updateOperation(ctx, id, func(operation *cpo_proto.Operation) error {
		if operation.Status != nil && operation.Status.Timestamp != nil &&
			operation.Status.Timestamp.AsTime().After(newStatus.Timestamp.AsTime()) {
			// Stale status update is not applied
			return nil
		}

		if operation.Status != nil && !custom_profiling_operation.IsAllowedStateChange(operation.Status.State, newStatus.State) {
			return fmt.Errorf("invalid state change from %s to %s", operation.Status.State.String(), newStatus.State.String())
		}

		operation.Status = newStatus
		return nil
	})
}

func (s *Storage) ListOperations(ctx context.Context, filter *custom_profiling_operation.OperationFilter, pagination *util.Pagination) ([]*cpo_proto.Operation, error) {
	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for alive replica: %w", err)
	}

	psql := squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	query := psql.Select("id", "meta", "spec", "status", "target_state").
		From(operationsTable).
		OrderBy("(meta->>'CreatedAt') DESC")

	if filter != nil {
		if filter.EndsAfter != nil {
			query = query.Where("(spec->'TimeInterval'->>'To')::timestamptz >= ?", *filter.EndsAfter)
		}

		if filter.StartsBefore != nil {
			query = query.Where("(spec->'TimeInterval'->>'From')::timestamptz < ?", *filter.StartsBefore)
		}

		if len(filter.States) > 0 {
			stateStrings := make([]string, 0, len(filter.States))
			for _, state := range filter.States {
				stateStrings = append(stateStrings, stateToString(state))
			}
			// Use COALESCE to convert NULL to empty string for Unknown state matching
			query = query.Where(squirrel.Eq{"COALESCE(status->>'State', '')": stateStrings})
		}
	}

	if pagination != nil && pagination.Limit != 0 {
		query = query.Limit(pagination.Limit)
	} else {
		query = query.Limit(defaultListOperationsLimit)
	}

	if pagination != nil && pagination.Offset != 0 {
		query = query.Offset(pagination.Offset)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	s.logger.Debug(ctx, "Listing operations in postgres", log.String("sql", sql))

	var rows []*operationRow
	err = alive.DBx().SelectContext(ctx, &rows, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("can't list operations: %w", err)
	}

	operations, err := rowsToOperations(rows)
	if err != nil {
		return nil, fmt.Errorf("can't list operations: %w", err)
	}

	return operations, nil
}

func (s *Storage) updateOperation(ctx context.Context, id custom_profiling_operation.OperationID, cb func(operation *cpo_proto.Operation) error) error {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for primary replica: %w", err)
	}

	tx, err := primary.DBx().BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to start tx: %w", err)
	}
	defer func() {
		if rollbackErr := tx.Rollback(); rollbackErr != nil && rollbackErr != sql.ErrTxDone {
			s.logger.Warn(ctx, "Failed to rollback transaction", log.Error(rollbackErr))
		}
	}()

	operation, err := getOperationForUpdate(ctx, tx, id)
	if err != nil {
		return err
	}

	err = cb(operation)
	if err != nil {
		return err
	}

	err = putOperation(ctx, tx, operation)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func getOperationForUpdate(ctx context.Context, tx *sqlx.Tx, id custom_profiling_operation.OperationID) (*cpo_proto.Operation, error) {
	var row operationRow
	err := tx.GetContext(
		ctx,
		&row,
		`SELECT id, meta, spec, status, target_state FROM `+operationsTable+` WHERE id = $1 FOR UPDATE`,
		string(id),
	)
	if err != nil {
		return nil, err
	}

	operation, err := rowToOperation(&row)
	if err != nil {
		return nil, err
	}

	return operation, nil
}

func putOperation(ctx context.Context, tx *sqlx.Tx, operation *cpo_proto.Operation) error {
	row, err := operationToRow(operation)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE `+operationsTable+` SET status = $1, target_state = $2 WHERE id = $3`,
		row.Status, row.TargetState, row.ID)
	if err != nil {
		return err
	}

	return nil
}

package postgrestaskservice

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	"github.com/yandex/perforator/perforator/pkg/sqlbuilder"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

const (
	tasksTable            = "tasks"
	idempotencyIndexTable = "tasks_idempotency_index"
)

type TaskService struct {
	logger   xlog.Logger
	cluster  *hasql.Cluster
	config   *Config
	hostname string

	pingPeriod  time.Duration
	pingTimeout time.Duration
}

func NewTaskService(
	config *Config,
	cluster *hasql.Cluster,
	logger xlog.Logger,
	metrics metrics.Registry,
) (*TaskService, error) {
	config.fillDefault()

	logger = logger.WithName("TaskService")

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get self hostname: %w", err)
	}

	service := &TaskService{
		logger:      logger,
		cluster:     cluster,
		config:      config,
		hostname:    hostname,
		pingPeriod:  config.PingPeriod,
		pingTimeout: config.PingTimeout,
	}

	return service, nil
}

func (s *TaskService) GetTask(ctx context.Context, id asynctask.TaskID) (*asynctask.Task, error) {
	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	var row TaskRow
	err = alive.DBx().GetContext(
		ctx,
		&row,
		`SELECT id, idempotency_key, meta, spec, status, result
		FROM tasks
		WHERE id = $1`,
		string(id),
	)
	if err != nil {
		return nil, err
	}

	task, err := RowToTask(&row)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) ListTasks(ctx context.Context, filter *asynctask.TaskFilter, limit uint64, offset uint64) ([]asynctask.Task, error) {
	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, fmt.Errorf("can't list tasks: %w", err)
	}

	builder := sqlbuilder.Select().
		From(tasksTable).
		Values("id, idempotency_key, meta, spec, status, result").
		OrderBy(&sqlbuilder.OrderBy{
			Columns:    []string{`(meta->>'CreationTime')::bigint`},
			Descending: true,
		}).
		Offset(offset).
		Limit(limit)

	if author := filter.Author; author != "" {
		builder.Where(fmt.Sprintf(`(meta->>'Author') = '%s'`, author))
	}

	if ts := filter.From.UnixMicro(); ts != 0 {
		builder.Where(fmt.Sprintf(`(meta->>'CreationTime')::bigint > %d`, ts))
	}

	if ts := filter.To.UnixMicro(); ts != 0 {
		builder.Where(fmt.Sprintf(`(meta->>'CreationTime')::bigint < %d`, ts))
	}

	query, err := builder.Query()
	if err != nil {
		return nil, fmt.Errorf("can't list tasks: %w", err)
	}

	var rows []*TaskRow
	err = alive.DBx().SelectContext(
		ctx,
		&rows,
		query,
	)
	if err != nil {
		return nil, fmt.Errorf("can't list tasks: %w", err)
	}

	tasks, err := RowsToTasks(rows)
	if err != nil {
		return nil, fmt.Errorf("can't list tasks: %w", err)
	}

	return tasks, nil
}

func (s *TaskService) AddTask(
	ctx context.Context,
	meta *perforator.TaskMeta,
	spec *perforator.TaskSpec,
) (asynctask.TaskID, error) {
	if meta == nil || spec == nil {
		return "", fmt.Errorf("can't add task: nil meta or spec")
	}
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return "", err
	}

	var id asynctask.TaskID
	id, err = s.genID()
	if err != nil {
		return "", err
	}

	meta.ID = string(id)
	meta.CreationTime = time.Now().UnixMicro()

	row, err := TaskToRow(&asynctask.Task{
		ID:   id,
		Meta: meta,
		Spec: spec,
		Status: &perforator.TaskStatus{
			State: perforator.TaskState_Created,
		},
		Result: &perforator.TaskResult{},
	})
	if err != nil {
		return "", fmt.Errorf("failed to marshal row: %w", err)
	}

	// get or insert logic
	err = primary.DBx().GetContext(
		ctx,
		&id,
		// we need DO UPDATE SET for RETURNING to work. It only works for the last inserted row.
		// If row was not inserted (e.g. DO NOTHING), it will not return any id.
		`INSERT INTO tasks (id, idempotency_key, meta, spec, status, result)
            VALUES ($1, $2, $3, $4, $5, $6)
            ON CONFLICT (idempotency_key)
			DO UPDATE SET idempotency_key = EXCLUDED.idempotency_key
            RETURNING id`,
		row.ID, row.IdempotencyKey, row.Meta, row.Spec, row.Status, row.Result,
	)
	if err != nil {
		return "", fmt.Errorf("failed get or insert: %w", err)
	}

	return id, nil
}

func (s *TaskService) FinishTask(ctx context.Context, id asynctask.TaskID, result *perforator.TaskResult) error {
	return s.updateTask(ctx, id, func(task *asynctask.Task) error {
		if task.GetStatus().State == perforator.TaskState_Finished {
			return asynctask.ErrTaskAlreadyFinished
		}
		task.GetStatus().State = perforator.TaskState_Finished
		task.Result = result
		return nil
	})
}

func (s *TaskService) FailTask(ctx context.Context, id asynctask.TaskID, err string) error {
	return s.updateTask(ctx, id, func(task *asynctask.Task) error {
		if task.GetStatus().State == perforator.TaskState_Finished {
			return asynctask.ErrTaskAlreadyFinished
		}
		task.GetStatus().State = perforator.TaskState_Failed
		task.GetStatus().Error = err
		return nil
	})
}

func (s *TaskService) pickTask(ctx context.Context) (task *asynctask.Task, stop func(), err error) {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return
	}

	tx, err := primary.DBx().BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		err = fmt.Errorf("failed to start tx: %w", err)
		return
	}
	defer func() {
		_ = tx.Rollback()
	}()

	t, err := s.selectNextTask(ctx, tx)
	if err != nil {
		return
	}
	if t == nil {
		return
	}

	if err = s.lockTask(ctx, tx, t); err != nil {
		return
	}

	if err = tx.Commit(); err != nil {
		return
	}

	if t.GetStatus().GetState() != perforator.TaskState_Running {
		err = asynctask.ErrAttemptsLimitReached
		return
	}

	stop = s.startPingTask(ctx, t.ID)
	task = t
	err = nil

	return
}

func (s *TaskService) PickTask(ctx context.Context) (task *asynctask.Task, stop func(), err error) {
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}

		task, stop, err = s.pickTask(ctx)
		if errors.Is(err, asynctask.ErrAttemptsLimitReached) {
			continue
		}
		return
	}
}

func (s *TaskService) selectNextTask(ctx context.Context, tx *sqlx.Tx) (*asynctask.Task, error) {
	deadline := time.Now().Add(-s.pingTimeout)

	var row TaskRow
	err := tx.QueryRowContext(
		ctx,
		`SELECT id, idempotency_key, meta, spec, status, result
		FROM tasks
		WHERE (status->>'State' = 'Running' AND (status->>'LastPing')::bigint < $1)
		OR status->>'State' = 'Created'
		ORDER BY status->>'State', (status->>'LastPing')::bigint
		LIMIT 1
		FOR UPDATE`,
		deadline.UnixMicro(),
	).Scan(&row.ID, &row.IdempotencyKey, &row.Meta, &row.Spec, &row.Status, &row.Result)
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to select next task: %w", err)
	}

	task, err := RowToTask(&row)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	return task, nil
}

func (s *TaskService) lockTask(ctx context.Context, tx *sqlx.Tx, task *asynctask.Task) error {
	attemptCount := len(task.GetStatus().GetAttempts())
	if attemptCount >= s.config.MaxAttempts {
		s.logger.Warn(ctx, "Failing task after too many attempts",
			logTask(task.ID),
			log.Time("lastping", time.UnixMicro(task.GetStatus().GetLastPing())),
		)
		task.GetStatus().State = perforator.TaskState_Failed
		task.GetStatus().Error = "Too many execution attempts"
		return s.putTask(ctx, tx, task)
	}

	if state := task.GetStatus().State; state == perforator.TaskState_Running {
		prev := ""
		if attemptCount > 0 {
			prev = task.GetStatus().GetAttempts()[attemptCount-1].GetExecutor()
		}

		s.logger.Warn(ctx, "Locking stale task",
			logTask(task.ID),
			log.String("state", state.String()),
			log.String("prevhost", prev),
			log.Time("lastping", time.UnixMicro(task.GetStatus().GetLastPing())),
		)
	} else {
		s.logger.Info(ctx, "Locking new task",
			logTask(task.ID),
			log.String("state", state.String()),
		)
	}

	now := time.Now().UnixMicro()
	task.GetStatus().State = perforator.TaskState_Running
	task.GetStatus().LastPing = now
	task.GetStatus().Attempts = append(task.GetStatus().Attempts, &perforator.TaskExecution{
		Executor:  s.hostname,
		StartTime: now,
	})

	return s.putTask(ctx, tx, task)
}

func (s *TaskService) startPingTask(ctx context.Context, id asynctask.TaskID) (stop func()) {
	done := make(chan bool, 1)

	go func() {
		ticker := time.NewTicker(s.pingPeriod)
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			case <-ticker.C:
			}

			if err := s.pingTask(ctx, id); err != nil {
				s.logger.Warn(ctx, "Failed to ping task",
					logTask(id),
					log.Error(err),
				)
			}
		}
	}()

	return func() {
		close(done)
	}
}

func (s *TaskService) pingTask(ctx context.Context, id asynctask.TaskID) error {
	return s.updateTask(ctx, id, func(task *asynctask.Task) error {
		now := time.Now().UnixMicro()
		task.GetStatus().LastPing = now
		if l := len(task.GetStatus().GetAttempts()); l > 0 {
			task.GetStatus().GetAttempts()[l-1].LastSeenTime = now
		}
		return nil
	})
}

func (s *TaskService) updateTask(ctx context.Context, id asynctask.TaskID, cb func(task *asynctask.Task) error) error {
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	tx, err := primary.DBx().BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to start tx: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	task, err := s.getTaskForUpdate(ctx, tx, id)
	if err != nil {
		return err
	}

	err = cb(task)
	if err != nil {
		return err
	}

	err = s.putTask(ctx, tx, task)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *TaskService) getTaskForUpdate(ctx context.Context, tx *sqlx.Tx, id asynctask.TaskID) (*asynctask.Task, error) {
	var row TaskRow
	err := tx.GetContext(
		ctx,
		&row,
		`SELECT id, idempotency_key, meta, spec, status, result
		FROM tasks
		WHERE id = $1
		FOR UPDATE`,
		string(id),
	)
	if err != nil {
		return nil, err
	}

	task, err := RowToTask(&row)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskService) putTask(ctx context.Context, tx *sqlx.Tx, task *asynctask.Task) error {
	row, err := TaskToRow(task)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO tasks (id, idempotency_key, meta, spec, status, result)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET idempotency_key = EXCLUDED.idempotency_key,
			meta = EXCLUDED.meta,
			spec = EXCLUDED.spec,
			status = EXCLUDED.status,
			result = EXCLUDED.result`,
		row.ID, row.IdempotencyKey, row.Meta, row.Spec, row.Status, row.Result)
	if err != nil {
		return err
	}

	return nil
}

func (s *TaskService) genID() (asynctask.TaskID, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return asynctask.TaskID(id.String()), nil
}

func logTask(id asynctask.TaskID) log.Field {
	return log.String("task.id", string(id))
}

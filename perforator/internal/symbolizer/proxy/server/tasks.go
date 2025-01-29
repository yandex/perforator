package server

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

// GetTask implements perforator.TaskServiceServer.
func (s *PerforatorServer) GetTask(
	ctx context.Context,
	req *perforator.GetTaskRequest,
) (*perforator.GetTaskResponse, error) {
	task, err := s.tasks.GetTask(ctx, asynctask.TaskID(req.GetTaskID()))
	if err != nil {
		return nil, err
	}

	return &perforator.GetTaskResponse{
		Spec:   task.Spec,
		Status: task.Status,
		Result: task.Result,
	}, nil
}

// StartTask implements perforator.TaskServiceServer.
func (s *PerforatorServer) StartTask(
	ctx context.Context,
	req *perforator.StartTaskRequest,
) (*perforator.StartTaskResponse, error) {
	spec := req.GetSpec()
	spec.TraceBaggage = &perforator.TraceBaggage{
		Baggage: make(map[string]string),
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.MapCarrier(spec.TraceBaggage.Baggage))

	meta := &perforator.TaskMeta{}
	if user := auth.UserFromContext(ctx); user != nil {
		meta.Author = user.Login
	}
	if key := req.GetIdempotencyKey(); key != "" {
		meta.IdempotencyKey = key
	}

	id, err := s.tasks.AddTask(ctx, meta, req.GetSpec())
	if err != nil {
		return nil, err
	}

	return &perforator.StartTaskResponse{TaskID: string(id)}, nil
}

// ListTasks implements perforator.TaskServiceServer.
func (s *PerforatorServer) ListTasks(ctx context.Context, req *perforator.ListTasksRequest) (*perforator.ListTasksResponse, error) {
	query := req.GetQuery()
	pagination := req.GetPagination()

	var offset uint64

	if pagination != nil {
		offset = pagination.Offset
	}

	var limit uint64

	if pagination != nil && pagination.Limit != 0 {
		limit = pagination.Limit
	} else {
		limit = 100
	}

	filter := &asynctask.TaskFilter{
		Author: query.GetAuthor(),
		From:   query.GetFrom().AsTime(),
		To:     query.GetTo().AsTime(),
	}

	tasks, err := s.tasks.ListTasks(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	var res = make([]*perforator.Task, 0, limit)
	for _, task := range tasks {
		res = append(res, &perforator.Task{
			Meta:   task.Meta,
			Spec:   task.Spec,
			Status: task.Status,
			Result: task.Result,
		})
	}

	return &perforator.ListTasksResponse{
		Tasks: res,
	}, nil
}

func (s *PerforatorServer) runAsyncTasks(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}

		for {
			spawned, err := s.pollTasks(ctx)
			if err != nil {
				return err
			}
			if !spawned {
				break
			}
		}
	}
}

func (s *PerforatorServer) pollTasks(ctx context.Context) (spawned bool, err error) {
	if err := s.tasksemaphore.Acquire(ctx, 1); err != nil {
		return false, err
	}
	defer func() {
		if !spawned {
			s.tasksemaphore.Release(1)
		}
	}()

	task, stop, err := s.tasks.PickTask(ctx)
	if err != nil {
		s.l.Warn(ctx, "Failed to pick async task", log.Error(err))
		return false, nil
	}

	if task == nil {
		return false, nil
	}

	go s.runTask(ctx, task, stop)
	return true, nil
}

func (s *PerforatorServer) runTask(ctx context.Context, task *asynctask.Task, stop func()) {
	defer s.tasksemaphore.Release(1)
	defer stop()

	kind := s.taskKindString(task.GetSpec())
	metricTags := map[string]string{"kind": kind}
	s.metrics.tasksRunningCount.With(metricTags).Add(1)
	defer s.metrics.tasksRunningCount.With(metricTags).Add(-1)

	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(task.Spec.GetTraceBaggage().GetBaggage()))
	ctx, span := otel.Tracer("TaskService").Start(ctx, "PerforatorServer.runTask")
	defer span.End()

	ctx = auth.ContextWithUser(ctx, &auth.User{Login: task.GetMeta().GetAuthor()})
	ctx = xlog.WrapContext(ctx, log.String("task.id", string(task.ID)))

	s.l.Info(ctx, "Starting async task")
	s.metrics.tasksStartedCount.With(metricTags).Inc()

	res, err := s.runTaskImpl(ctx, task.GetSpec())
	if err != nil {
		s.metrics.tasksFailedCount.With(metricTags).Inc()

		s.l.Error(ctx, "Failed async task", log.Error(err))
		if err := s.tasks.FailTask(ctx, task.ID, err.Error()); err != nil {
			s.l.Error(ctx, "Failed to store task failure", log.Error(err))
		}

		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return
	}

	if err := s.tasks.FinishTask(ctx, task.ID, res); err != nil {
		s.metrics.tasksFailedCount.With(metricTags).Inc()
		s.l.Error(ctx, "Failed to store task result", log.Error(err))
		return
	}

	s.metrics.tasksFinishedCount.With(metricTags).Inc()
	s.l.Info(ctx, "Finished async task")
}

func (s *PerforatorServer) isBannedUser(user string) bool {
	return s.bannedUsers.IsBanned(user)
}

func (s *PerforatorServer) runTaskImpl(ctx context.Context, spec *perforator.TaskSpec) (*perforator.TaskResult, error) {
	if user := auth.UserFromContext(ctx); user != nil && s.isBannedUser(user.Login) {
		s.l.Error(ctx, "User is banned, skipping task", log.String("user", user.Login))
		return nil, fmt.Errorf("user %s is banned", user.Login)
	}

	result := &perforator.TaskResult{}

	switch v := spec.GetKind().(type) {
	case *perforator.TaskSpec_MergeProfiles:
		res, err := s.MergeProfiles(ctx, v.MergeProfiles)
		if err != nil {
			return nil, err
		}
		result.Kind = &perforator.TaskResult_MergeProfiles{MergeProfiles: res}
		return result, nil

	case *perforator.TaskSpec_DiffProfiles:
		res, err := s.DiffProfiles(ctx, v.DiffProfiles)
		if err != nil {
			return nil, err
		}
		result.Kind = &perforator.TaskResult_DiffProfiles{DiffProfiles: res}
		return result, nil

	case *perforator.TaskSpec_GeneratePGOProfile:
		res, err := s.GeneratePGOProfile(ctx, v.GeneratePGOProfile)
		if err != nil {
			return nil, err
		}
		result.Kind = &perforator.TaskResult_GeneratePGOProfile{GeneratePGOProfile: res}
		return result, nil

	default:
		return nil, fmt.Errorf("unsupported task kind %+v", v)
	}
}

func (s *PerforatorServer) taskKindString(spec *perforator.TaskSpec) string {
	switch spec.GetKind().(type) {
	case *perforator.TaskSpec_MergeProfiles:
		return "MergeProfiles"
	case *perforator.TaskSpec_DiffProfiles:
		return "DiffProfiles"
	case *perforator.TaskSpec_GeneratePGOProfile:
		return "GeneratePGOProfile"
	default:
		return "UnknownTaskKind"
	}
}

func (s *PerforatorServer) waitTasks(ctx context.Context, taskIDs ...string) ([]*perforator.TaskResult, error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	results := make([]*perforator.TaskResult, len(taskIDs))
	runningTasks := len(taskIDs)

	for runningTasks > 0 {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}

		for i, taskID := range taskIDs {
			if results[i] != nil {
				continue
			}

			t, err := s.tasks.GetTask(ctx, asynctask.TaskID(taskID))
			if err != nil {
				s.l.Error(ctx, "Failed to poll task", log.String("id", taskID), log.Error(err))
				continue
			}

			state := t.GetStatus().GetState()
			if !asynctask.IsFinalState(state) {
				continue
			}

			switch state {
			case perforator.TaskState_Failed:
				return nil, fmt.Errorf("subtask failed after %d attempts: %s",
					len(t.GetStatus().GetAttempts()),
					t.GetStatus().GetError(),
				)
			case perforator.TaskState_Finished:
				results[i] = t.GetResult()
				runningTasks--
			}
		}
	}

	return results, nil
}

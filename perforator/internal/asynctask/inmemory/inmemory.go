package inmemorytaskservice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/asynctask"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type InMemoryTaskService struct {
	logger   xlog.Logger
	config   *Config
	hostname string

	taskMap map[asynctask.TaskID]*asynctask.Task
	m       sync.Mutex

	pingPeriod  time.Duration
	pingTimeout time.Duration
}

func NewInMemoryTaskService(
	config *Config,
	logger xlog.Logger,
	metrics metrics.Registry,
) (*InMemoryTaskService, error) {
	config.fillDefault()

	logger = logger.WithName("TaskService")

	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("failed to get self hostname: %w", err)
	}

	service := &InMemoryTaskService{
		logger,
		config,
		hostname,
		make(map[asynctask.TaskID]*asynctask.Task),
		sync.Mutex{},
		config.PingPeriod,
		config.PingTimeout,
	}

	return service, nil
}

func (s *InMemoryTaskService) GetTask(ctx context.Context, id asynctask.TaskID) (*asynctask.Task, error) {
	return s.getTask(id)
}

func (s *InMemoryTaskService) ListTasks(ctx context.Context, filter *asynctask.TaskFilter, limit uint64, offset uint64) ([]asynctask.Task, error) {
	return s.getTasks(ctx, filter, int(limit), int(offset))
}

func (s *InMemoryTaskService) AddTask(
	ctx context.Context,
	meta *perforator.TaskMeta,
	spec *perforator.TaskSpec,
) (asynctask.TaskID, error) {
	id, err := s.genID()
	if err != nil {
		return "", err
	}

	meta.ID = string(id)
	meta.CreationTime = time.Now().UnixMicro()

	s.m.Lock()
	defer s.m.Unlock()
	s.taskMap[id] = &asynctask.Task{
		ID:   id,
		Meta: meta,
		Spec: spec,
		Status: &perforator.TaskStatus{
			State: perforator.TaskState_Created,
		},
		Result: &perforator.TaskResult{},
	}

	return id, nil
}

func (s *InMemoryTaskService) FinishTask(ctx context.Context, id asynctask.TaskID, result *perforator.TaskResult) error {
	return s.updateTask(ctx, id, func(task *asynctask.Task) error {
		if task.GetStatus().State == perforator.TaskState_Finished {
			return asynctask.ErrTaskAlreadyFinished
		}
		task.GetStatus().State = perforator.TaskState_Finished
		task.Result = result
		return nil
	})
}

func (s *InMemoryTaskService) FailTask(ctx context.Context, id asynctask.TaskID, err string) error {
	return s.updateTask(ctx, id, func(task *asynctask.Task) error {
		if task.GetStatus().State == perforator.TaskState_Finished {
			return asynctask.ErrTaskAlreadyFinished
		}
		task.GetStatus().State = perforator.TaskState_Failed
		task.GetStatus().Error = err
		return nil
	})
}

func (s *InMemoryTaskService) pickTask(ctx context.Context) (task *asynctask.Task, stop func(), err error) {
	s.m.Lock()
	defer s.m.Unlock()
	deadline := time.Now().Add(-s.pingTimeout)

	for _, v := range s.taskMap {
		if (v.Status.State == perforator.TaskState_Running && v.Status.LastPing < deadline.UnixMicro()) || v.Status.State == perforator.TaskState_Created {
			task = v
			break
		}
	}

	if task == nil {
		return
	}

	if err = s.lockTask(ctx, task); err != nil {
		return
	}

	if task.GetStatus().GetState() != perforator.TaskState_Running {
		err = asynctask.ErrAttemptsLimitReached
		return
	}

	stop = s.startPingTask(ctx, task.ID)
	err = nil

	return
}

func (s *InMemoryTaskService) PickTask(ctx context.Context) (task *asynctask.Task, stop func(), err error) {
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

func (s *InMemoryTaskService) lockTask(ctx context.Context, task *asynctask.Task) error {
	attemptCount := len(task.GetStatus().GetAttempts())
	if attemptCount >= s.config.MaxAttempts {
		s.logger.Warn(ctx, "Failing task after too many attempts",
			logTask(task.ID),
			log.Time("lastping", time.UnixMicro(task.GetStatus().GetLastPing())),
		)
		task.GetStatus().State = perforator.TaskState_Failed
		task.GetStatus().Error = "Too many execution attempts"
		s.taskMap[task.ID] = task
		return nil
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

	s.taskMap[task.ID] = task
	return nil
}

func (s *InMemoryTaskService) startPingTask(ctx context.Context, id asynctask.TaskID) (stop func()) {
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

func (s *InMemoryTaskService) pingTask(ctx context.Context, id asynctask.TaskID) error {
	return s.updateTask(ctx, id, func(task *asynctask.Task) error {
		now := time.Now().UnixMicro()
		task.GetStatus().LastPing = now
		if l := len(task.GetStatus().GetAttempts()); l > 0 {
			task.GetStatus().GetAttempts()[l-1].LastSeenTime = now
		}
		return nil
	})
}

func (s *InMemoryTaskService) updateTask(ctx context.Context, id asynctask.TaskID, cb func(task *asynctask.Task) error) error {
	s.m.Lock()
	defer s.m.Unlock()

	task, ok := s.taskMap[id]
	if !ok {
		return fmt.Errorf("there is no task with id: %v", id)
	}

	err := cb(task)
	if err != nil {
		return err
	}

	s.taskMap[task.ID] = task

	return nil
}

func (s *InMemoryTaskService) getTask(id asynctask.TaskID) (*asynctask.Task, error) {
	s.m.Lock()
	defer s.m.Unlock()
	t, ok := s.taskMap[id]
	if !ok {
		return nil, fmt.Errorf("there is no task with id: %v", id)
	}

	return t, nil
}

type InMemoryTaskFilter struct {
	Author string
	From   time.Time
	To     time.Time
}

func (s *InMemoryTaskService) getTasks(ctx context.Context, filter *asynctask.TaskFilter, limit int, offset int) ([]asynctask.Task, error) {
	s.m.Lock()
	defer s.m.Unlock()
	fmt.Println(filter)

	var res []asynctask.Task
	for _, task := range s.taskMap {
		if s.isFiltered(task, filter) {
			fmt.Printf("%v\n\n", task)
			continue
		}

		res = append(res, *task)
	}

	if len(res) == 0 || offset >= len(res) {
		return nil, nil
	}
	limit = min(len(res), limit)

	sort.Slice(res, func(i, j int) bool {
		return res[i].Meta.CreationTime < res[j].Meta.CreationTime
	})

	return res[offset:limit], nil
}

func (s *InMemoryTaskService) isFiltered(task *asynctask.Task, filter *asynctask.TaskFilter) bool {
	if filter.Author != "" && task.Meta.Author != filter.Author {
		return true
	}

	if from := filter.From.UnixMicro(); from != 0 && task.Meta.CreationTime < from {
		return true
	}

	if to := filter.To.UnixMicro(); to != 0 && task.Meta.CreationTime > to {
		return true
	}
	return false
}

func (s *InMemoryTaskService) genID() (asynctask.TaskID, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return asynctask.TaskID(id.String()), nil
}

func logTask(id asynctask.TaskID) log.Field {
	return log.String("task.id", string(id))
}
